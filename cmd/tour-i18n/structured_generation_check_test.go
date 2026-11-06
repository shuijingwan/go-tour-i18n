package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestStructuredGenerationCheckCommandContract(t *testing.T) {
	doc, err := os.ReadFile("../../docs/SITE_V2_WORKFLOW.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"site-content structured-generation-check", "current downstream validation of one published StructuredGeneration", "INPUT_KIND", "INPUT_PATH_CONTRACT", "REQUIRED_FLAGS", "FORBIDDEN_FLAGS", "STDOUT_KIND", "MUTATION_EXPECTATION", "IDEMPOTENCE", "CURRENTNESS_CHECK", "PRIOR_OR_CAS", "SUCCESS_STOP", "FAILURE_STOP", "RECOVERY_REPLAY", "go-learning/structured-generation-check/v1", "original_generated_at_status", "structured-generation-check-first-real-use-gate"} {
		if !strings.Contains(string(doc), field) {
			t.Fatal("missing canonical contract", field)
		}
	}
	root := t.TempDir()
	raw := []byte("{}\n")
	sha := contentidentity.SHA256Hex(raw)
	if err := os.WriteFile(filepath.Join(root, "generation.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	valid := []string{"structured-generation-check", "--input", "generation.json", "--input-sha256", sha}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = writer
	checkErr := siteContentCommand(root, valid)
	os.Stdout = stdout
	_ = writer.Close()
	output, _ := io.ReadAll(reader)
	_ = reader.Close()
	if checkErr == nil || strings.Contains(checkErr.Error(), "requires input/input-sha256") || bytes.Contains(output, []byte("PASS")) {
		t.Fatal("valid flags did not dispatch to real checker", checkErr, string(output))
	}
	for _, args := range [][]string{
		{}, {"--input", "generation.json"}, {"--input-sha256", sha},
		{"--input", filepath.Join(root, "generation.json"), "--input-sha256", sha},
		{"--input", "../generation.json", "--input-sha256", sha},
		{"--input", "generation.json", "--input-sha256", strings.Repeat("a", 64)},
		{"--input", "generation.json", "--input-sha256", sha, "extra"},
	} {
		if err := siteContentCommand(root, append([]string{"structured-generation-check"}, args...)); err == nil {
			t.Fatal("invalid arguments accepted", args)
		}
	}
	for _, arg := range [][]string{{"--apply"}, {"--apply=false"}, {"--record"}, {"--provider", "chatgpt"}, {"--model", "gpt-5.6-sol-high"}, {"--session", "other"}, {"--generated-at", "UNKNOWN"}, {"--bundle", "bundle.zip"}, {"--output", "out.json"}, {"--prior-sha256", sha}, {"--skip-currentness"}, {"--locale", "zh-CN"}} {
		if err := siteContentCommand(root, append(append([]string{}, valid...), arg...)); err == nil {
			t.Fatal("override accepted", arg)
		}
	}
	if err := os.Symlink("generation.json", filepath.Join(root, "link.json")); err != nil {
		t.Fatal(err)
	}
	if err := siteContentCommand(root, []string{"structured-generation-check", "--input", "link.json", "--input-sha256", sha}); err == nil {
		t.Fatal("symlink accepted")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 2 {
		t.Fatal("readonly failure mutated root", entries, err)
	}
	got, err := os.ReadFile(filepath.Join(root, "generation.json"))
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatal("input changed", err)
	}
}
