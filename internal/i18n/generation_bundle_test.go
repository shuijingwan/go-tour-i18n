package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func copyBundleAuthority(t *testing.T, root string, paths []string) {
	t.Helper()
	sourceRoot := repoRoot(t)
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func generationBundleFixture(t *testing.T, locale string, generator RetranslationGenerator) (string, *Catalog, string) {
	t.Helper()
	root := t.TempDir()
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	writeRetranslationTestGlossaryForLocale(t, root, locale)
	catalog := retranslationTestCatalog(2)
	exported, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: locale, Generator: generator, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	return root, catalog, exported.BatchID
}

func TestGenerationBundleResultImportIsDeterministicAtomicAndNoOverwrite(t *testing.T) {
	root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
	bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
	if err != nil {
		t.Fatal(err)
	}
	repeated, repeatedManifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
	if err != nil || !bytes.Equal(bundle, repeated) || manifest.InputIdentitySHA256 != repeatedManifest.InputIdentitySHA256 {
		t.Fatalf("generation bundle is not deterministic: err=%v", err)
	}
	if manifest.TaskKind != "translation-unit-initial" || len(manifest.ExpectedOutputs) != 2 {
		t.Fatalf("manifest=%+v", manifest)
	}
	outputDir := t.TempDir()
	for _, expected := range manifest.ExpectedOutputs {
		input := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "inputs", filepath.Base(expected.BundlePath))
		data, err := os.ReadFile(input)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(outputDir, filepath.Base(expected.BundlePath)), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	resultBundle, resultManifest, err := PackGenerationResultBundle(root, catalog, bundle, "codex", FormalGenerationModel, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if resultManifest.Provider != "codex" || len(resultManifest.Outputs) != 2 {
		t.Fatalf("result manifest=%+v", resultManifest)
	}
	imported, err := ImportGenerationResultBundle(root, catalog, bundle, resultBundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.InstalledPaths) != 2 {
		t.Fatalf("import=%+v", imported)
	}
	if _, err := ImportGenerationResultBundle(root, catalog, bundle, resultBundle); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate import was not rejected: %v", err)
	}
}

func TestGenerationBundleRejectsStaleAuthorityBeforeInstall(t *testing.T) {
	root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
	bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
	if err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	for _, expected := range manifest.ExpectedOutputs {
		data, _ := os.ReadFile(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "inputs", filepath.Base(expected.BundlePath)))
		_ = os.WriteFile(filepath.Join(outputDir, filepath.Base(expected.BundlePath)), data, 0644)
	}
	resultBundle, _, err := PackGenerationResultBundle(root, catalog, bundle, "codex", FormalGenerationModel, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	glossaryPath := filepath.Join(root, "locales", "zh-CN", "glossary.yaml")
	if err := os.WriteFile(glossaryPath, []byte("mandatory:\n  Go: Go language\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = ImportGenerationResultBundle(root, catalog, bundle, resultBundle)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale bundle was accepted: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "raw-responses")); !os.IsNotExist(statErr) {
		t.Fatalf("stale import left formal output: %v", statErr)
	}
}

func TestGenerationBundleProviderNeutralAndLocaleIsolated(t *testing.T) {
	rootCodex, catalogCodex, batchCodex := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
	rootChatGPT, catalogChatGPT, batchChatGPT := generationBundleFixture(t, "ja-JP", RetranslationGeneratorChatGPT)
	codex, codexManifest, err := ExportTranslationUnitGenerationBundle(rootCodex, catalogCodex, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchCodex})
	if err != nil {
		t.Fatal(err)
	}
	chatgpt, chatgptManifest, err := ExportTranslationUnitGenerationBundle(rootChatGPT, catalogChatGPT, GenerationBundleOptions{Locale: "ja-JP", BatchID: batchChatGPT})
	if err != nil {
		t.Fatal(err)
	}
	if codexManifest.Kind != chatgptManifest.Kind || codexManifest.SchemaVersion != chatgptManifest.SchemaVersion {
		t.Fatalf("providers received incompatible contracts: codex=%+v chatgpt=%+v", codexManifest, chatgptManifest)
	}
	if bytes.Equal(codex, chatgpt) || codexManifest.Locale == chatgptManifest.Locale || codexManifest.BatchID == chatgptManifest.BatchID {
		t.Fatal("two locale artifacts were not identity-isolated")
	}
}

func TestRetryGenerationBundleRequiresMatchingRetryableValidationAndResult(t *testing.T) {
	root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	manifest := readRetranslationManifestForLocale(t, root, "zh-CN", batchID)
	if err := os.Mkdir(filepath.Join(batchDir, "raw-responses"), 0755); err != nil {
		t.Fatal(err)
	}
	for index, record := range manifest.Units {
		data, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(record.InputPath)))
		if err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			data = []byte(strings.Replace(string(data), "⟪GTI18N_", "⟪BROKEN_", 1))
		}
		if err := os.WriteFile(filepath.Join(batchDir, "raw-responses", filepath.Base(record.InputPath)), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	processed, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID})
	if err != nil {
		t.Fatal(err)
	}
	failedID := processed.Units[0].UnitID
	if processed.Units[0].Status != "restore_failed" {
		t.Fatalf("fixture did not produce retryable failure: %+v", processed.Units[0])
	}
	_, retryManifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID, UnitID: failedID})
	if err != nil {
		t.Fatal(err)
	}
	if retryManifest.TaskKind != "translation-unit-retry" || retryManifest.Attempt != 2 || len(retryManifest.ExpectedOutputs) != 1 {
		t.Fatalf("retry manifest=%+v", retryManifest)
	}
	resultPath := filepath.Join(batchDir, "result.json")
	resultData, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	var result RetranslationProcessResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		t.Fatal(err)
	}
	result.Units[0].Status = "passed"
	mutated, _ := json.MarshalIndent(result, "", "  ")
	if err := os.WriteFile(resultPath, append(mutated, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID, UnitID: failedID}); err == nil || !strings.Contains(err.Error(), "does not match result.json") {
		t.Fatalf("mismatched retry result was accepted: %v", err)
	}
}
