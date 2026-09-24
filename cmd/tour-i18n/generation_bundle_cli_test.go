package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestGenerationBundleImportRequiresExactlyOneResultSource(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "generation.zip")
	if err := os.WriteFile(bundle, []byte("generation bundle fixture"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"--bundle", bundle},
		{"--bundle", bundle, "--result", filepath.Join(root, "result.zip"), "--input-dir", filepath.Join(root, "outputs"), "--provider", "codex", "--model", i18n.FormalGenerationModel},
	} {
		if err := importGenerationResultBundleCommand(root, nil, args); err == nil || !strings.Contains(err.Error(), "usage: generation-bundle import") {
			t.Fatalf("args %v did not enforce mutually exclusive result sources: %v", args, err)
		}
	}
}

func TestGenerationBundleDirectoryImportRequiresProviderAndModel(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "generation.zip")
	if err := os.WriteFile(bundle, []byte("generation bundle fixture"), 0644); err != nil {
		t.Fatal(err)
	}
	args := []string{"--bundle", bundle, "--input-dir", filepath.Join(root, "outputs")}
	if err := importGenerationResultBundleCommand(root, nil, args); err == nil || !strings.Contains(err.Error(), "--provider and --model are required") {
		t.Fatalf("missing direct-import provenance flags were not rejected: %v", err)
	}
}
