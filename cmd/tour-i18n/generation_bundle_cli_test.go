package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestGenerationBundleExportContract(t *testing.T) {
	doc, err := os.ReadFile("../../docs/TRANSLATION_TASK_SPEC.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{"同一 complete package-selection contract", "--content-package tour-v1", "default complete scope", "无 foreign adapter", "UseTranslationPackage()", "batch 每个 Unit 都属于该 scope", "formal/comment-revision-transport"} {
		if !strings.Contains(string(doc), rule) {
			t.Fatal("canonical Generation package/transport contract missing", rule)
		}
	}
	root := t.TempDir()
	output := filepath.Join(root, "generation.zip")
	for _, args := range [][]string{
		{"--locale", "zh-CN", "--output", output},
		{"--batch-id", "chatgpt-zh-CN-001", "--output", output},
		{"--locale", "zh-CN", "--batch-id", "chatgpt-zh-CN-001"},
		{"--locale", "zh-CN", "--batch-id", "chatgpt-zh-CN-001", "--output", output, "manifest.json"},
	} {
		if err := exportGenerationBundleCommand(root, nil, args, ""); err == nil || !strings.Contains(err.Error(), "usage: generation-bundle export") {
			t.Fatalf("export flags/path contract: %v: %v", args, err)
		}
	}
	args := []string{"--locale", "zh-CN", "--batch-id", "chatgpt-zh-CN-001", "--output", output}
	// Reach the intended missing-Catalog guard under the current explicit
	// locale-policy contract, without installing any repository locale policy.
	metadata := i18n.Locale{Locale: "zh-CN", HTMLLang: "zh-CN", LanguageName: "fixture", EnglishName: "fixture", Phase: "scaffold", TranslationUnit: "present.Section", LanguageAudit: testLocaleInitLanguagePolicy("zh-CN")}
	courseContextWrite(t, root, "locales/zh-CN/locale.json", courseContextJSON(t, metadata))
	if err := exportGenerationBundleCommand(root, nil, args, ""); err == nil || !strings.Contains(err.Error(), "catalog is required") {
		t.Fatal("missing canonical authority accepted", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("failed preparation published transport", err)
	}
	// Publication is transport-only and never overwrites an existing path,
	// even when the bytes of an exact deterministic reconstruction match.
	data := []byte("isolated deterministic transport fixture")
	if _, err := writeNewTransportOutput(output, data); err != nil {
		t.Fatal(err)
	}
	if _, err := writeNewTransportOutput(output, data); err == nil {
		t.Fatal("existing output overwritten")
	}
	got, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatal("original transport changed", err)
	}
}

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
		if err := importGenerationResultBundleCommand(root, nil, args, ""); err == nil || !strings.Contains(err.Error(), "usage: generation-bundle import") {
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
	if err := importGenerationResultBundleCommand(root, nil, args, ""); err == nil || !strings.Contains(err.Error(), "--provider and --model are required") {
		t.Fatalf("missing direct-import provenance flags were not rejected: %v", err)
	}
}

func TestGenerationBundleRecoverRequiresSourceAndOutput(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "generation.zip")
	if err := os.WriteFile(bundle, []byte("generation bundle fixture"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"--bundle", bundle, "--output-dir", filepath.Join(root, "recovered")},
		{"--bundle", bundle, "--source", filepath.Join(root, "partial")},
	} {
		if err := recoverGenerationBundleCommand(root, nil, args); err == nil || !strings.Contains(err.Error(), "usage: generation-bundle recover") {
			t.Fatalf("args %v did not enforce recovery source/output contract: %v", args, err)
		}
	}
}
