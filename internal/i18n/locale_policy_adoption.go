package i18n

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
)

const LocalePolicyAdoptionResultSchema = "go-learning/locale-language-audit-adoption-result/v1"

type LocalePolicyAdoptionOptions struct {
	Locale, Input                          string
	Apply                                  bool
	PriorSHA256, InputSHA256, ResultSHA256 string
}

type LocalePolicyAdoptionResult struct {
	Schema            string `json:"schema"`
	Locale            string `json:"locale"`
	PolicySchema      string `json:"policy_schema"`
	PolicyVersion     int    `json:"policy_version"`
	InputPolicySHA256 string `json:"input_policy_sha256"`
	PriorSHA256       string `json:"prior_sha256"`
	ResultSHA256      string `json:"result_sha256"`
	AlreadyCurrent    bool   `json:"already_current"`
	Applied           bool   `json:"applied"`
	Mutation          string `json:"mutation"`
}

type localePolicyAdoptionPlan struct {
	result                   LocalePolicyAdoptionResult
	policy, prior, output    []byte
	policyInfo, metadataInfo os.FileInfo
}

// This extends existing locale authority only. It cannot initialize a locale,
// replace a different policy, or change any pre-existing metadata bytes.
func AdoptLocaleLanguagePolicy(root string, o LocalePolicyAdoptionOptions) (*LocalePolicyAdoptionResult, error) {
	return adoptLocaleLanguagePolicy(root, o, nil)
}

// The private boundary allows deterministic mutation-detection regressions,
// without a production/global hook or a public option for skipping guards.
func adoptLocaleLanguagePolicy(root string, o LocalePolicyAdoptionOptions, beforeRecheck func() error) (result *LocalePolicyAdoptionResult, err error) {
	if err := ValidateLocaleName(o.Locale); err != nil {
		return nil, fmt.Errorf("mutation=none: %w", err)
	}
	if o.Apply {
		if !validSHA256(o.PriorSHA256) || !validSHA256(o.InputSHA256) || !validSHA256(o.ResultSHA256) {
			return nil, fmt.Errorf("mutation=none: apply requires exact prior/input/result SHA-256")
		}
	} else if o.PriorSHA256 != "" || o.InputSHA256 != "" || o.ResultSHA256 != "" {
		return nil, fmt.Errorf("mutation=none: SHA pins require --apply")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("mutation=none: %w", err)
	}
	defer r.Close()
	plan, err := planLocalePolicyAdoption(r, o)
	if err != nil {
		return nil, fmt.Errorf("mutation=none: %w", err)
	}
	if !o.Apply {
		return &plan.result, nil
	}
	parent := "locales/" + o.Locale
	dir, err := r.Open(parent)
	if err != nil {
		return nil, fmt.Errorf("mutation=none: %w", err)
	}
	defer dir.Close()
	unlock, err := lockLocalePolicyAdoption(dir)
	if err != nil {
		return nil, fmt.Errorf("mutation=none: locale adoption publisher busy/unsupported: %w", err)
	}
	defer unlock()
	parentRoot, err := r.OpenRoot(parent)
	if err != nil {
		return nil, fmt.Errorf("mutation=none: %w", err)
	}
	defer parentRoot.Close()
	pinned, err := parentRoot.Stat(".")
	locked, e := dir.Stat()
	if err != nil || e != nil || !os.SameFile(pinned, locked) {
		return nil, fmt.Errorf("mutation=none: locale directory identity changed")
	}
	// Another publisher may have completed before the directory lock was taken.
	fresh, err := planLocalePolicyAdoption(r, o)
	if err != nil {
		return nil, fmt.Errorf("mutation=none: %w", err)
	}
	if !sameLocalePolicyAdoptionPlan(plan, fresh) {
		return nil, fmt.Errorf("mutation=none: adoption inputs changed before publication")
	}
	if fresh.result.AlreadyCurrent {
		return &fresh.result, nil
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return nil, fmt.Errorf("mutation=none: %w", err)
	}
	temporary := ".language-policy-adopt-" + hex.EncodeToString(random[:])
	f, err := parentRoot.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("mutation=none: %w", err)
	}
	defer func() {
		if cleanupErr := parentRoot.Remove(temporary); cleanupErr != nil && !os.IsNotExist(cleanupErr) {
			result = nil
			err = fmt.Errorf("mutation=unknown: staging cleanup failed: %v; operation: %v", cleanupErr, err)
		}
	}()
	if err = f.Chmod(plan.metadataInfo.Mode().Perm()); err == nil {
		_, err = f.Write(plan.output)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, fmt.Errorf("mutation=none: stage policy metadata: %w", err)
	}
	if beforeRecheck != nil {
		if err := beforeRecheck(); err != nil {
			return nil, fmt.Errorf("mutation=none: %w", err)
		}
	}
	fresh, err = planLocalePolicyAdoption(r, o)
	if err != nil {
		return nil, fmt.Errorf("mutation=none: %w", err)
	}
	if !sameLocalePolicyAdoptionPlan(plan, fresh) {
		return nil, fmt.Errorf("mutation=none: concurrent metadata/policy mutation")
	}
	// Pin the parent directory as well as both regular-file identities. No
	// persistent lock/receipt/pointer is created by this operation.
	now, err := r.Stat(parent)
	opened, e := dir.Stat()
	if err != nil || e != nil || !os.SameFile(now, opened) {
		return nil, fmt.Errorf("mutation=none: locale directory changed")
	}
	if err := parentRoot.Rename(temporary, "locale.json"); err != nil {
		return nil, fmt.Errorf("mutation=unknown: atomic metadata replacement failed: %w", err)
	}
	if err := dir.Sync(); err != nil {
		return nil, fmt.Errorf("mutation=unknown: metadata replacement durability unconfirmed: %w", err)
	}
	saved, _, err := readLocalePolicyAdoptionFile(r, parent+"/locale.json")
	if err != nil || !bytes.Equal(saved, plan.output) {
		return nil, fmt.Errorf("mutation=unknown: metadata changed after atomic publication: %v", err)
	}
	plan.result.Applied = true
	plan.result.Mutation = "known"
	return &plan.result, nil
}

func sameLocalePolicyAdoptionPlan(a, b *localePolicyAdoptionPlan) bool {
	return a.result == b.result && bytes.Equal(a.policy, b.policy) && bytes.Equal(a.prior, b.prior) && bytes.Equal(a.output, b.output) && os.SameFile(a.policyInfo, b.policyInfo) && os.SameFile(a.metadataInfo, b.metadataInfo)
}

func planLocalePolicyAdoption(r *os.Root, o LocalePolicyAdoptionOptions) (*localePolicyAdoptionPlan, error) {
	policy, pi, err := readLocalePolicyAdoptionFile(r, o.Input)
	if err != nil {
		return nil, err
	}
	authority, err := languageaudit.DecodeAuthority(policy, o.Locale)
	if err != nil {
		return nil, err
	}
	if authority.Version != 1 {
		return nil, fmt.Errorf("initial policy adoption requires policy_version=1")
	}
	metadata, mi, err := readLocalePolicyAdoptionFile(r, "locales/"+o.Locale+"/locale.json")
	if err != nil {
		return nil, err
	}
	var locale Locale
	if err := contentidentity.StrictJSON(metadata, &locale); err != nil {
		return nil, fmt.Errorf("invalid existing locale metadata: %w", err)
	}
	if err := ValidateLocaleIdentityBytes(o.Locale, metadata); err != nil {
		return nil, err
	}
	if locale.HTMLLang != o.Locale || locale.Phase != "scaffold" {
		return nil, fmt.Errorf("existing locale metadata identity/phase mismatch")
	}
	var fields map[string]json.RawMessage
	if err := contentidentity.StrictJSON(metadata, &fields); err != nil {
		return nil, err
	}
	output := metadata
	current := false
	canonical, err := json.Marshal(authority)
	if err != nil {
		return nil, err
	}
	if existing, ok := fields["language_audit"]; ok {
		saved, err := languageaudit.DecodeAuthority(existing, o.Locale)
		if err != nil {
			return nil, fmt.Errorf("invalid existing policy: %w", err)
		}
		b, err := json.Marshal(saved)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(b, canonical) {
			return nil, fmt.Errorf("existing different language_audit policy cannot be overwritten")
		}
		current = true
	} else {
		// Strict JSON and required identity fields guarantee a nonempty root object.
		// Insert only this member: preserve all prior bytes, order and whitespace.
		end := bytes.LastIndexByte(metadata, '}')
		output = append(append(append([]byte(nil), metadata[:end]...), []byte(",\"language_audit\":"+string(canonical))...), metadata[end:]...)
		var check Locale
		if err := contentidentity.StrictJSON(output, &check); err != nil {
			return nil, err
		}
	}
	result := LocalePolicyAdoptionResult{Schema: LocalePolicyAdoptionResultSchema, Locale: o.Locale, PolicySchema: authority.Schema, PolicyVersion: authority.Version, InputPolicySHA256: sum(policy), PriorSHA256: sum(metadata), ResultSHA256: sum(output), AlreadyCurrent: current, Mutation: "none"}
	if o.Apply && (result.PriorSHA256 != o.PriorSHA256 || result.InputPolicySHA256 != o.InputSHA256 || result.ResultSHA256 != o.ResultSHA256) {
		return nil, fmt.Errorf("prior/input/result SHA-256 CAS mismatch; require fresh preflight")
	}
	return &localePolicyAdoptionPlan{result, policy, metadata, output, pi, mi}, nil
}

func readLocalePolicyAdoptionFile(r *os.Root, path string) ([]byte, os.FileInfo, error) {
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") || filepath.ToSlash(filepath.Clean(path)) != path {
		return nil, nil, fmt.Errorf("path must be canonical repository-relative: %q", path)
	}
	prefix := ""
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return nil, nil, fmt.Errorf("path must be repository-relative without traversal")
		}
		if prefix != "" {
			prefix += "/"
		}
		prefix += part
		info, err := r.Lstat(prefix)
		if err != nil {
			return nil, nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, nil, fmt.Errorf("symlink forbidden: %s", prefix)
		}
	}
	info, err := r.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("regular file required: %s", path)
	}
	f, err := r.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, nil, fmt.Errorf("file identity changed: %s", path)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, err
	}
	after, err := r.Lstat(path)
	if err != nil || !os.SameFile(opened, after) || !after.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("file changed while reading: %s", path)
	}
	return b, opened, nil
}
