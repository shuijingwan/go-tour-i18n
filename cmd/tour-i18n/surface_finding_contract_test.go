package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The semantic owner/record/selection chain is exercised in sitecontent's
// isolated TestSurfaceSharedCompositionFindingAuthority. This direct dispatch
// test binds its formal path, input-kind and flag contract to the CLI.
func TestSurfaceFindingOwnerCommandContract(t *testing.T) {
	doc, err := os.ReadFile("../../docs/INTEGRATED_SURFACE_REVIEW.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"Mandatory shared composition finding owner", "Shared representation successor 的 exact revision authority", "exact predecessor proof", "dependency re-attestation", "整个 predecessor closure 和 scope", "不进入 current-eligibility cache", "COMMAND / OPERATION", "INPUT_KIND", "REQUIRED_FLAGS / FORBIDDEN_FLAGS", "INPUT_PATH_CONTRACT", "STDOUT_KIND", "MUTATION_EXPECTATION", "IDEMPOTENCE", "CURRENTNESS_CHECK", "PRIOR_OR_CAS", "SUCCESS_STOP", "FAILURE_STOP / RECOVERY", "ShellExtractionContract", "candidate_artifacts", "TargetPath", "repository-relative", "不创建 normalized substitute input"} {
		if !strings.Contains(string(doc), field) {
			t.Fatal("canonical finding contract missing", field)
		}
	}
	root := t.TempDir()
	input := filepath.Join(root, "result.json")
	if err := os.WriteFile(input, []byte(`{"schema":"go-learning/integrated-locale-surface/v1","identity_sha256":"already-recorded"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := siteContentCommand(root, []string{"surface-record", "--input", "result.json"}); err == nil || !strings.Contains(err.Error(), "invalid integrated review identity") {
		t.Fatal("record accepted sealed receipt instead of original Reviewer result", err)
	}
	for _, args := range [][]string{
		{"surface-record"},
		{"surface-record", "--input", input},
		{"surface-record", "--input", "../result.json"},
		{"surface-record", "--input", "result.json", "--locale", "zh-CN"},
		{"surface-record", "--input", "result.json", "--finding", "replacement.json"},
		{"surface-record", "--input", "result.json", "--output", "substitute.json"},
		{"surface-record", "--input", "result.json", "--apply"},
		{"surface-record", "--input", "result.json", "extra"},
		{"structured-export", "--locale", "zh-CN", "--task", "revision", "--output", "out.zip"},
		{"structured-export", "--locale", "zh-CN", "--task", "initial", "--finding", "result.json", "--output", "out.zip"},
		{"structured-export", "--locale", "zh-CN", "--task", "revision", "--finding", "receipt.json", "--output", "out.zip", "--apply"},
		{"structured-export", "--locale", "zh-CN", "--task", "revision", "--finding", "receipt.json", "--output", "out.zip", "--allow-stale"},
		{"structured-export", "--locale", "zh-CN", "--task", "revision", "--finding", "receipt.json", "--output", "out.zip", "--successor", "receipt.json"},
	} {
		if err := siteContentCommand(root, args); err == nil {
			t.Fatal("invalid finding command accepted", args)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "result.json" {
		t.Fatal("invalid command published receipt/transport/substitute", entries, err)
	}
}
