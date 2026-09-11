package main

import (
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

var projectFinalizationREADME = projectRootREADME

const finalizationPlaceholder = "<!-- first-production-finalization:start -->\n" +
	"- production receipt identity: `PENDING`\n" +
	"- production machine acceptance: `PENDING`\n" +
	"- production browser acceptance: `PENDING`\n" +
	"- unresolved production blocker: `PENDING`\n" +
	"- overall final decision: `PENDING`\n" +
	"- decision: `pending`\n" +
	"<!-- first-production-finalization:end -->"

const legacyFinalizationPlaceholder = "<!-- first-production-finalization:start -->\n" +
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

func validateFinalizationPlaceholder(evidence []byte) error {
	if bytes.Count(evidence, []byte("<!-- first-production-finalization:start -->")) != 1 ||
		bytes.Count(evidence, []byte("<!-- first-production-finalization:end -->")) != 1 {
		return fmt.Errorf("Surface Review evidence must contain exactly one untouched first-production finalization placeholder")
	}
	if bytes.Count(evidence, []byte(finalizationPlaceholder))+bytes.Count(evidence, []byte(legacyFinalizationPlaceholder)) != 1 {
		return fmt.Errorf("Surface Review evidence must contain exactly one untouched first-production finalization placeholder")
	}
	return nil
}

func firstProductionEvidencePreflightCommand(root string, catalog *i18n.Catalog, args []string) error {
	fs := flag.NewFlagSet("first-production evidence-preflight", flag.ContinueOnError)
	releaseDir := fs.String("release-dir", "", "formal local release directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *releaseDir == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: first-production evidence-preflight --release-dir <release-dir>")
	}
	releasePath, err := filepath.Abs(*releaseDir)
	if err != nil {
		return err
	}
	info, err := os.Stat(releasePath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("formal release directory is missing: %s", *releaseDir)
	}
	if !strings.HasPrefix(filepath.Base(releasePath), "go-tour-release-") {
		return fmt.Errorf("release directory must be go-tour-release-<name>")
	}
	locale, err := readReleaseLocale(filepath.Join(releasePath, "release.json"))
	if err != nil {
		return err
	}
	identityPath := filepath.Join(root, "production", "identity.json")
	if err := validateProductionIdentity(root, identityPath); err != nil {
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
		if profile.Locale == locale {
			profiles = append(profiles, profile)
		}
	}
	if len(profiles) != 1 || profiles[0].State != "first-production" {
		return fmt.Errorf("first-production evidence preflight requires exactly one locale %s with production_state=first-production", locale)
	}
	gate, err := i18n.RequireUniqueCurrentLocaleSurfaceReviewAGate(root, locale, catalog)
	if err != nil {
		return err
	}
	gatePath, err := i18n.LocaleSurfaceReviewAGatePath(root, locale, gate.ReviewID)
	if err != nil {
		return err
	}
	evidence, err := os.ReadFile(strings.TrimSuffix(gatePath, ".a-gate.json") + ".md")
	if err != nil {
		return fmt.Errorf("read Surface Review evidence: %w", err)
	}
	if err := validateFinalizationPlaceholder(evidence); err != nil {
		return err
	}
	fmt.Printf("FIRST PRODUCTION EVIDENCE PREFLIGHT: PASS (locale=%s review_id=%s)\n", locale, gate.ReviewID)
	return nil
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
	return finalizeFirstProduction(root, catalog, *releaseDir, *reviewID, os.Stdout, validateProductionIdentity)
}

func finalizeFirstProduction(root string, catalog *i18n.Catalog, releaseDir, reviewID string, output io.Writer, validate func(string, string) error) error {
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
	if _, err := i18n.RequireCurrentLocaleSurfaceReviewAByReviewID(root, release, reviewID, catalog); err != nil {
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
	if err := validateFinalizationPlaceholder(evidence); err != nil {
		return err
	}
	placeholder := finalizationPlaceholder
	if bytes.Contains(evidence, []byte(legacyFinalizationPlaceholder)) {
		placeholder = legacyFinalizationPlaceholder
	}
	finalized := strings.Replace(string(evidence), placeholder, renderFinalization(receipt), 1)
	newIdentity, err := replaceTargetState(identityBytes, release)
	if err != nil {
		return err
	}
	if err := validateCandidateIdentity(root, newIdentity, validate); err != nil {
		return err
	}
	newREADME, err := projectFinalizationREADME(root, newIdentity)
	if err != nil {
		return err
	}
	readmePath := filepath.Join(root, "README.md")
	oldREADME, err := os.ReadFile(readmePath)
	if err != nil {
		return fmt.Errorf("read README: %w", err)
	}
	if err := commitFinalization(evidencePath, evidence, []byte(finalized), identityPath, identityBytes, newIdentity, readmePath, oldREADME, newREADME, root, validate); err != nil {
		return err
	}
	fmt.Fprintf(output, "FIRST PRODUCTION FINALIZATION: PASS (locale=%s review_id=%s production_state=live)\n", release, reviewID)
	return nil
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
	return fmt.Sprintf("<!-- first-production-finalization:start -->\n- production receipt identity: `locale=%s hostname=%s release=%s`\n- production machine acceptance: `passed`\n- production browser acceptance: `passed`\n- unresolved production blocker: `none`\n- overall final decision: `passed`\n- decision: `passed`\n<!-- first-production-finalization:end -->", r.Locale, r.Hostname, r.Release)
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
func commitFinalization(evidencePath string, oldEvidence, newEvidence []byte, identityPath string, oldIdentity, newIdentity []byte, readmePath string, oldREADME, newREADME []byte, root string, validate func(string, string) error) error {
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
	readmeTemp, err := atomicWrite(readmePath, newREADME)
	if err != nil {
		return err
	}
	defer os.Remove(readmeTemp)
	if err = os.Rename(evidenceTemp, evidencePath); err != nil {
		return err
	}
	if err = os.Rename(readmeTemp, readmePath); err != nil {
		_ = os.WriteFile(evidencePath, oldEvidence, 0644)
		return err
	}
	if err = os.Rename(identityTemp, identityPath); err != nil {
		_ = os.WriteFile(evidencePath, oldEvidence, 0644)
		_ = os.WriteFile(readmePath, oldREADME, 0644)
		return err
	}
	if err = validate(root, identityPath); err != nil {
		_ = os.WriteFile(identityPath, oldIdentity, 0644)
		_ = os.WriteFile(evidencePath, oldEvidence, 0644)
		_ = os.WriteFile(readmePath, oldREADME, 0644)
		return err
	}
	return nil
}
