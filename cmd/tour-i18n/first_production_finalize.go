package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const firstProductionReceiptSchema = "go-tour-i18n/first-production-receipt/v1"

const finalizationPlaceholder = "<!-- first-production-finalization:start -->\n" +
	"- production receipt identity: `PENDING`\n" +
	"- production machine acceptance: `PENDING`\n" +
	"- production browser acceptance: `PENDING`\n" +
	"- production visual HUMAN gate: `PENDING`\n" +
	"- unresolved production blocker: `PENDING`\n" +
	"- overall final decision: `PENDING`\n" +
	"- decision: `pending`\n" +
	"<!-- first-production-finalization:end -->"

type firstProductionReceipt struct {
	Schema   string                          `json:"schema"`
	Locale   string                          `json:"locale"`
	Hostname string                          `json:"hostname"`
	Release  string                          `json:"release"`
	Result   string                          `json:"result"`
	Stages   map[string]firstProductionStage `json:"stages"`
}
type firstProductionStage struct {
	Result string `json:"result"`
}
type finalizeIdentity struct {
	Locales []finalizeProfile `json:"locales"`
}
type finalizeProfile struct {
	Locale   string `json:"locale"`
	Hostname string `json:"production_hostname"`
	State    string `json:"production_state"`
}

func finalizeFirstProductionCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("first-production finalize", flag.ContinueOnError)
	releaseDir := fs.String("release-dir", "", "formal local release directory")
	reviewID := fs.String("review-id", "", "existing Surface Review evidence id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *releaseDir == "" || *reviewID == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: first-production finalize --release-dir <release-dir> --review-id <review-id>")
	}
	if !stdinIsTTY() {
		return fmt.Errorf("first-production finalize requires an interactive TTY for VISUAL-PASS")
	}
	return finalizeFirstProduction(root, catalog, *releaseDir, *reviewID, os.Stdin, os.Stdout, true, validateProductionIdentity)
}

func stdinIsTTY() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func finalizeFirstProduction(root string, catalog *i18n.Catalog, releaseDir, reviewID string, input io.Reader, output io.Writer, tty bool, validate func(string, string) error) error {
	if !tty {
		return fmt.Errorf("first-production finalize requires an interactive TTY for VISUAL-PASS")
	}
	releasePath, err := filepath.Abs(releaseDir)
	if err != nil {
		return err
	}
	info, err := os.Stat(releasePath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("formal release directory is missing: %s", releaseDir)
	}
	releaseName := filepath.Base(releasePath)
	if !strings.HasPrefix(releaseName, "go-tour-release-") {
		return fmt.Errorf("release directory must be go-tour-release-<name>")
	}
	release, err := readReleaseLocale(filepath.Join(releasePath, "release.json"))
	if err != nil {
		return err
	}
	identityPath := filepath.Join(root, "production", "identity.json")
	if err := validate(root, identityPath); err != nil {
		return err
	}
	identityBytes, err := os.ReadFile(identityPath)
	if err != nil {
		return err
	}
	var identity finalizeIdentity
	if json.Unmarshal(identityBytes, &identity) != nil {
		return fmt.Errorf("malformed production identity")
	}
	profiles := []finalizeProfile{}
	for _, profile := range identity.Locales {
		if profile.Locale == release {
			profiles = append(profiles, profile)
		}
	}
	if len(profiles) != 1 {
		return fmt.Errorf("production identity must contain exactly one locale %s", release)
	}
	profile := profiles[0]
	if profile.State != "first-production" {
		return fmt.Errorf("first-production finalize requires production_state=first-production, got %s", profile.State)
	}
	receiptPath := filepath.Join(filepath.Dir(releasePath), releaseName+".first-production-receipt.json")
	receipt, err := readFinalReceipt(receiptPath)
	if err != nil {
		return err
	}
	formalRelease := strings.TrimPrefix(releaseName, "go-tour-release-")
	if receipt.Locale != release || receipt.Hostname != profile.Hostname || receipt.Release != formalRelease {
		return fmt.Errorf("first-production receipt identity does not match release and production identity")
	}
	if receipt.Result != "passed" || receipt.Stages["public-machine"].Result != "PASS" || receipt.Stages["browser"].Result != "PASS" {
		return fmt.Errorf("first-production receipt requires passed public-machine and browser stages")
	}
	if err := i18n.RequireCurrentLocaleSurfaceReviewA(root, release, catalog); err != nil {
		return err
	}
	evidencePath, err := i18n.LocaleSurfaceReviewAGatePath(root, release, reviewID)
	if err != nil {
		return err
	}
	evidencePath = strings.TrimSuffix(evidencePath, ".a-gate.json") + ".md"
	evidence, err := os.ReadFile(evidencePath)
	if err != nil {
		return fmt.Errorf("read Surface Review evidence: %w", err)
	}
	if bytes.Count(evidence, []byte("<!-- first-production-finalization:start -->")) != 1 || bytes.Count(evidence, []byte("<!-- first-production-finalization:end -->")) != 1 || bytes.Count(evidence, []byte(finalizationPlaceholder)) != 1 {
		return fmt.Errorf("Surface Review evidence must contain exactly one untouched first-production finalization placeholder")
	}
	fmt.Fprint(output, "Complete the formal desktop/mobile visual HUMAN gate, then type VISUAL-PASS exactly: ")
	line, readErr := bufio.NewReader(input).ReadString('\n')
	if readErr != nil && len(line) == 0 {
		return fmt.Errorf("VISUAL-PASS confirmation was not received")
	}
	if strings.TrimSpace(line) != "VISUAL-PASS" {
		return fmt.Errorf("VISUAL-PASS confirmation did not match")
	}
	finalized := strings.Replace(string(evidence), finalizationPlaceholder, renderFinalization(receipt), 1)
	newIdentity, err := replaceTargetState(identityBytes, release)
	if err != nil {
		return err
	}
	if err := validateCandidateIdentity(root, newIdentity, validate); err != nil {
		return err
	}
	return commitFinalization(evidencePath, evidence, []byte(finalized), identityPath, identityBytes, newIdentity, root, validate)
}

func readReleaseLocale(path string) (string, error) {
	var v struct {
		Locale string `json:"locale"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if json.Unmarshal(data, &v) != nil || v.Locale == "" {
		return "", fmt.Errorf("invalid release.json locale")
	}
	return v.Locale, nil
}
func readFinalReceipt(path string) (firstProductionReceipt, error) {
	var receipt firstProductionReceipt
	data, err := os.ReadFile(path)
	if err != nil {
		return receipt, fmt.Errorf("read first-production receipt: %w", err)
	}
	if json.Unmarshal(data, &receipt) != nil || receipt.Schema != firstProductionReceiptSchema || receipt.Locale == "" || receipt.Hostname == "" || receipt.Release == "" || receipt.Stages == nil {
		return receipt, fmt.Errorf("malformed first-production receipt")
	}
	return receipt, nil
}
func renderFinalization(r firstProductionReceipt) string {
	return fmt.Sprintf("<!-- first-production-finalization:start -->\n- production receipt identity: `locale=%s hostname=%s release=%s`\n- production machine acceptance: `passed`\n- production browser acceptance: `passed`\n- production visual HUMAN gate: `passed` (maintainer confirmation)\n- unresolved production blocker: `none`\n- overall final decision: `passed`\n- decision: `passed`\n<!-- first-production-finalization:end -->", r.Locale, r.Hostname, r.Release)
}

func replaceTargetState(data []byte, locale string) ([]byte, error) {
	needle := []byte(`"locale": "` + locale + `"`)
	at := bytes.Index(data, needle)
	if at < 0 || bytes.Count(data, needle) != 1 {
		return nil, fmt.Errorf("target locale is not uniquely represented in production identity")
	}
	start := bytes.LastIndex(data[:at], []byte("{"))
	end := bytes.Index(data[at:], []byte("\n    }"))
	if start < 0 || end < 0 {
		return nil, fmt.Errorf("cannot identify target production identity object")
	}
	end += at + len("\n    }")
	object := data[start:end]
	old := []byte(`"production_state": "first-production"`)
	if bytes.Count(object, old) != 1 {
		return nil, fmt.Errorf("target production_state is not exactly first-production")
	}
	offset := start + bytes.Index(object, old)
	result := append([]byte{}, data[:offset]...)
	result = append(result, []byte(`"production_state": "live"`)...)
	result = append(result, data[offset+len(old):]...)
	return result, nil
}
func validateProductionIdentity(root, identityPath string) error {
	command := exec.Command("python3", filepath.Join(root, "scripts", "production-identity.py"), "--identity", identityPath, "validate")
	command.Dir = root
	if out, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("production identity validation failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
func validateCandidateIdentity(root string, data []byte, validate func(string, string) error) error {
	temp, err := os.CreateTemp(filepath.Join(root, "production"), ".identity-finalize-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(data); err == nil {
		err = temp.Close()
	}
	if err != nil {
		return err
	}
	return validate(root, name)
}
func atomicWrite(path string, data []byte) (string, error) {
	temp, err := os.CreateTemp(filepath.Dir(path), ".finalize-*")
	if err != nil {
		return "", err
	}
	if _, err = temp.Write(data); err == nil {
		err = temp.Chmod(0644)
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(temp.Name())
		return "", err
	}
	return temp.Name(), nil
}
func commitFinalization(evidencePath string, oldEvidence, newEvidence []byte, identityPath string, oldIdentity, newIdentity []byte, root string, validate func(string, string) error) error {
	evidenceTemp, err := atomicWrite(evidencePath, newEvidence)
	if err != nil {
		return err
	}
	defer os.Remove(evidenceTemp)
	identityTemp, err := atomicWrite(identityPath, newIdentity)
	if err != nil {
		return err
	}
	defer os.Remove(identityTemp)
	if err = os.Rename(evidenceTemp, evidencePath); err != nil {
		return err
	}
	if err = os.Rename(identityTemp, identityPath); err != nil {
		_ = os.WriteFile(evidencePath, oldEvidence, 0644)
		return err
	}
	if err = validate(root, identityPath); err != nil {
		_ = os.WriteFile(identityPath, oldIdentity, 0644)
		_ = os.WriteFile(evidencePath, oldEvidence, 0644)
		return err
	}
	return nil
}
