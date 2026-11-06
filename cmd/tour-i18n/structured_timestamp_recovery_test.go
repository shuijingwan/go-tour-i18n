package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestStructuredTimestampRecoveryCommandContract(t *testing.T) {
	doc, err := os.ReadFile("../../docs/STRUCTURED_TIMESTAMP_RECOVERY.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"COMMAND", "OPERATION", "INPUT_KIND", "INPUT_PATH_CONTRACT", "REQUIRED_FLAGS", "FORBIDDEN_FLAGS", "STDOUT_KIND", "MUTATION_EXPECTATION", "IDEMPOTENCE", "CURRENTNESS_CHECK", "PRIOR_OR_CAS", "SUCCESS_STOP", "FAILURE_STOP", "RECOVERY_REPLAY", "original_generated_at_status", "unavailable", "generated_at", "不能机器证明", "first-real-use", "exact ancestor", "currentness_bridge", "current_bundle_identity_sha256", "current_to_origin", "普通 current-check 和 generic", "不改成 current identity"} {
		if !strings.Contains(string(doc), field) {
			t.Fatal("canonical contract missing", field)
		}
	}
	root := t.TempDir()
	raw := []byte(`{"schema":"go-learning/structured-generation-timestamp-recovery/v1/request","bundle":{"path":"missing.zip","sha256":"` + strings.Repeat("a", 64) + `"},"attested_by":"maintainer-local"}` + "\n")
	if err := os.WriteFile(filepath.Join(root, "request.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	sha := contentidentity.SHA256Hex(raw)
	valid := []string{"structured-recover-timestamp", "--request", "request.json", "--request-sha256", sha}
	if err := siteContentCommand(root, valid); err == nil || !strings.Contains(err.Error(), "missing.zip") {
		t.Fatal("readonly request dispatch", err)
	}
	for _, args := range [][]string{
		{}, {"--request", "request.json"}, {"--request-sha256", sha}, {"--request", filepath.Join(root, "request.json"), "--request-sha256", sha},
		{"--request", "../request.json", "--request-sha256", sha}, {"--request", "request.json", "--request-sha256", strings.Repeat("b", 64)},
		{"--request", "request.json", "--request-sha256", sha, "extra"}, {"--request", "request.json", "--request-sha256", sha, "--apply=invalid"},
	} {
		if err := siteContentCommand(root, append([]string{"structured-recover-timestamp"}, args...)); err == nil {
			t.Fatal("invalid command accepted", args)
		}
	}
	for _, flag := range []string{"bundle", "input", "output", "locale", "package", "provider", "model", "session", "generated-at", "prior-sha256", "skip-currentness"} {
		if err := siteContentCommand(root, append(append([]string{}, valid...), "--"+flag, "override")); err == nil {
			t.Fatal("override accepted", flag)
		}
	}
	for _, old := range []string{"structured-import", "structured-recover"} {
		if err := siteContentCommand(root, []string{old, "--timestamp-recovery", "request.json"}); err == nil {
			t.Fatal("ordinary command acquired unavailable mode", old)
		}
	}
	if err := os.Symlink("request.json", filepath.Join(root, "link.json")); err != nil {
		t.Fatal(err)
	}
	if err := siteContentCommand(root, []string{"structured-recover-timestamp", "--request", "link.json", "--request-sha256", sha}); err == nil {
		t.Fatal("symlink accepted")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 2 {
		t.Fatal("invalid/readonly command mutated authority", entries, err)
	}
}
