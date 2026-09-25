package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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
	resultBundle, resultManifest, err := PackGenerationResultBundle(root, catalog, bundle, "codex", PreviousCodexGenerationModel, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if resultManifest.Provider != "codex" || resultManifest.Model != PreviousCodexGenerationModel || len(resultManifest.Outputs) != 2 {
		t.Fatalf("result manifest=%+v", resultManifest)
	}
	imported, err := ImportGenerationResultBundle(root, catalog, bundle, resultBundle)
	if err != nil {
		t.Fatal(err)
	}
	if imported.Model != PreviousCodexGenerationModel || len(imported.InstalledPaths) != 2 {
		t.Fatalf("import=%+v", imported)
	}
	if _, err := ImportGenerationResultBundle(root, catalog, bundle, resultBundle); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate import was not rejected: %v", err)
	}
}

func TestGenerationBundleResultPackCanonicalizesExampleCandidateEOF(t *testing.T) {
	root := t.TempDir()
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	writeRetranslationTestGlossaryForLocale(t, root, "zh-CN")
	example := retranslationTestExample(
		"example:demo/main.go",
		"_content/tour/demo/main.go",
		"package main\n\n// Translate this ordinary comment.\nfunc main() {}\n",
	)
	catalog := &Catalog{Examples: []Example{example}}
	exported, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
		Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{example.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: exported.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	batchManifest := readRetranslationManifest(t, root, exported.BatchID)
	input, err := os.ReadFile(filepath.Join(root, exported.BatchPath, filepath.FromSlash(batchManifest.Units[0].InputPath)))
	if err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outputDir, filepath.Base(manifest.ExpectedOutputs[0].BundlePath)), input, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := PackGenerationResultBundle(root, catalog, bundle, "chatgpt", FormalChatGPTGenerationModel, outputDir); err != nil {
		t.Fatal(err)
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

func writeGenerationFixtureOutputs(t *testing.T, root, batchID, outputDir string, manifest GenerationBundleManifest) {
	t.Helper()
	batchDir := filepath.Join(root, "data", "retranslation-runs", manifest.Locale, batchID)
	for _, expected := range manifest.ExpectedOutputs {
		input, err := os.ReadFile(filepath.Join(batchDir, "inputs", filepath.Base(expected.BundlePath)))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(outputDir, filepath.Base(expected.BundlePath)), input, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGenerationBundleDirectoryImportMatchesResultBundleAndProcesses(t *testing.T) {
	rootDirect, catalogDirect, batchDirect := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
	rootZip, catalogZip, batchZip := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
	bundleDirect, manifest, err := ExportTranslationUnitGenerationBundle(rootDirect, catalogDirect, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchDirect})
	if err != nil {
		t.Fatal(err)
	}
	bundleZip, _, err := ExportTranslationUnitGenerationBundle(rootZip, catalogZip, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchZip})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bundleDirect, bundleZip) {
		t.Fatal("equivalent fixtures produced different canonical Generation Bundles")
	}
	outputs := t.TempDir()
	writeGenerationFixtureOutputs(t, rootDirect, batchDirect, outputs, manifest)
	direct, err := ImportGenerationOutputDirectory(rootDirect, catalogDirect, bundleDirect, "codex", FormalGenerationModel, outputs)
	if err != nil {
		t.Fatal(err)
	}
	zipOutputs := t.TempDir()
	writeGenerationFixtureOutputs(t, rootZip, batchZip, zipOutputs, manifest)
	resultZIP, _, err := PackGenerationResultBundle(rootZip, catalogZip, bundleZip, "codex", FormalGenerationModel, zipOutputs)
	if err != nil {
		t.Fatal(err)
	}
	viaZIP, err := ImportGenerationResultBundle(rootZip, catalogZip, bundleZip, resultZIP)
	if err != nil {
		t.Fatal(err)
	}
	if direct.Provider != viaZIP.Provider || direct.Model != viaZIP.Model || direct.Attempt != viaZIP.Attempt ||
		direct.InputIdentitySHA256 != viaZIP.InputIdentitySHA256 || direct.GenerationBundleSHA256 != viaZIP.GenerationBundleSHA256 ||
		!equalStrings(direct.InstalledPaths, viaZIP.InstalledPaths) {
		t.Fatalf("import identities differ: direct=%+v ZIP=%+v", direct, viaZIP)
	}
	for _, path := range direct.InstalledPaths {
		a, err := os.ReadFile(filepath.Join(rootDirect, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(rootZip, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("formal output differs for %s", path)
		}
	}
	for _, importedRoot := range []struct {
		root, batchID string
		catalog       *Catalog
	}{
		{root: rootDirect, batchID: batchDirect, catalog: catalogDirect},
		{root: rootZip, batchID: batchZip, catalog: catalogZip},
	} {
		receiptData, err := os.ReadFile(filepath.Join(importedRoot.root, "data", "retranslation-runs", "zh-CN", importedRoot.batchID, "generation-imports", "initial-manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var receipt GenerationResultBundleManifest
		if err := json.Unmarshal(receiptData, &receipt); err != nil {
			t.Fatal(err)
		}
		if receipt.GenerationBundleSHA256 != direct.GenerationBundleSHA256 || receipt.Kind != "go-tour-i18n/generation-result-bundle" || receipt.Provider != "codex" || receipt.Model != FormalGenerationModel || len(receipt.Outputs) != len(manifest.ExpectedOutputs) {
			t.Fatalf("incomplete provenance receipt: %+v", receipt)
		}
		processed, err := ProcessRetranslationBatch(importedRoot.root, importedRoot.catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: importedRoot.batchID})
		if err != nil || processed.ValidationPassed != len(manifest.ExpectedOutputs) {
			t.Fatalf("import did not proceed through automatic validation: result=%+v err=%v", processed, err)
		}
	}
}

func TestGenerationBundleDirectoryImportSupportsExampleRevisionAndRetry(t *testing.T) {
	t.Run("example", func(t *testing.T) {
		root := t.TempDir()
		copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
		writeRetranslationTestGlossaryForLocale(t, root, "zh-CN")
		example := retranslationTestExample("example:demo/main.go", "_content/tour/demo/main.go", "package main\n\n// Translate this ordinary comment.\nfunc main() {}\n")
		catalog := &Catalog{Examples: []Example{example}}
		exported, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{example.ID}})
		if err != nil {
			t.Fatal(err)
		}
		bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: exported.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		outputs := t.TempDir()
		writeGenerationFixtureOutputs(t, root, exported.BatchID, outputs, manifest)
		if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "chatgpt", FormalChatGPTGenerationModel, outputs); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("revision", func(t *testing.T) {
		root, catalog, _ := makeRetranslationReviewBatchFixture(t, 2, "qc-001")
		copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
		recordQualityCheckRatings(t, root, catalog, "qc-001", "", "A", []string{"lesson/2"})
		recordQualityCheckRatings(t, root, catalog, "qc-001", "", "B", []string{"lesson/1"})
		exported, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}, AllowReexport: true, PreviousSnapshotID: "qc-001"})
		if err != nil {
			t.Fatal(err)
		}
		bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: exported.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		if manifest.TaskKind != "translation-unit-revision" {
			t.Fatalf("task kind=%s", manifest.TaskKind)
		}
		outputs := t.TempDir()
		writeGenerationFixtureOutputs(t, root, exported.BatchID, outputs, manifest)
		if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "codex", PreviousCodexGenerationModel, outputs); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("retry", func(t *testing.T) {
		root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
		batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
		manifest := readRetranslationManifest(t, root, batchID)
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
		if err != nil || processed.RestoreFailed != 1 {
			t.Fatalf("fixture process=%+v err=%v", processed, err)
		}
		failedID := processed.Units[0].UnitID
		bundle, retryManifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID, UnitID: failedID})
		if err != nil {
			t.Fatal(err)
		}
		if retryManifest.Attempt != 2 || retryManifest.TaskKind != "translation-unit-retry" {
			t.Fatalf("retry manifest=%+v", retryManifest)
		}
		outputs := t.TempDir()
		writeGenerationFixtureOutputs(t, root, batchID, outputs, retryManifest)
		imported, err := ImportGenerationOutputDirectory(root, catalog, bundle, "codex", FormalGenerationModel, outputs)
		if err != nil || imported.Attempt != 2 {
			t.Fatalf("retry import=%+v err=%v", imported, err)
		}
	})
}

func TestGenerationBundleDirectoryImportRejectsUnsafeAndInvalidOutputs(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(t *testing.T, dir, filename string)
	}{
		{name: "missing", mutate: func(t *testing.T, dir, filename string) {
			if err := os.Remove(filepath.Join(dir, filename)); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "extra", mutate: func(t *testing.T, dir, filename string) {
			if err := os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("extra\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "directory", mutate: func(t *testing.T, dir, filename string) {
			if err := os.Remove(filepath.Join(dir, filename)); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(dir, filename), 0755); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "symlink", mutate: func(t *testing.T, dir, filename string) {
			if err := os.Remove(filepath.Join(dir, filename)); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("/etc/passwd", filepath.Join(dir, filename)); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
		}},
		{name: "fifo", mutate: func(t *testing.T, dir, filename string) {
			if err := os.Remove(filepath.Join(dir, filename)); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(filepath.Join(dir, filename), 0600); err != nil {
				t.Skipf("FIFO unavailable: %v", err)
			}
		}},
		{name: "invalid UTF-8", mutate: func(t *testing.T, dir, filename string) {
			if err := os.WriteFile(filepath.Join(dir, filename), []byte{0xff, '\n'}, 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "BOM", mutate: func(t *testing.T, dir, filename string) {
			if err := os.WriteFile(filepath.Join(dir, filename), []byte{0xef, 0xbb, 0xbf, 'x', '\n'}, 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "EOF", mutate: func(t *testing.T, dir, filename string) {
			if err := os.WriteFile(filepath.Join(dir, filename), []byte("no final newline"), 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "protected token", mutate: func(t *testing.T, dir, filename string) {
			p := filepath.Join(dir, filename)
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			changed := bytes.Replace(b, []byte("⟪GTI18N_"), []byte("⟪BROKEN_"), 1)
			if bytes.Equal(changed, b) {
				t.Fatal("fixture has no protected token")
			}
			if err := os.WriteFile(p, changed, 0644); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
			bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			writeGenerationFixtureOutputs(t, root, batchID, dir, manifest)
			filename := filepath.Base(manifest.ExpectedOutputs[0].BundlePath)
			tc.mutate(t, dir, filename)
			if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "codex", FormalGenerationModel, dir); err == nil {
				t.Fatal("unsafe or invalid generation output was accepted")
			}
			if _, err := os.Lstat(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "raw-responses")); !os.IsNotExist(err) {
				t.Fatalf("failed import left formal raw responses: %v", err)
			}
		})
	}
}

func TestGenerationBundleDirectoryImportRejectsIdentityAndOverwrite(t *testing.T) {
	for _, tc := range []struct{ name, provider, model string }{
		{name: "wrong provider", provider: "chatgpt", model: FormalGenerationModel},
		{name: "wrong model", provider: "codex", model: "other-model"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
			bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			writeGenerationFixtureOutputs(t, root, batchID, dir, manifest)
			if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, tc.provider, tc.model, dir); err == nil {
				t.Fatal("invalid generation identity was accepted")
			}
			if _, err := os.Lstat(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "raw-responses")); !os.IsNotExist(err) {
				t.Fatalf("identity failure left raw responses: %v", err)
			}
		})
	}
	t.Run("empty formal directory exists", func(t *testing.T) {
		root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
		bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		writeGenerationFixtureOutputs(t, root, batchID, dir, manifest)
		rawDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "raw-responses")
		if err := os.Mkdir(rawDir, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "codex", FormalGenerationModel, dir); err == nil || !strings.Contains(err.Error(), "raw-responses") {
			t.Fatalf("existing empty formal directory was replaced: %v", err)
		}
		entries, err := os.ReadDir(rawDir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("formal directory changed: entries=%v err=%v", entries, err)
		}
	})
	t.Run("already imported", func(t *testing.T) {
		root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
		bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		writeGenerationFixtureOutputs(t, root, batchID, dir, manifest)
		if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "codex", FormalGenerationModel, dir); err != nil {
			t.Fatal(err)
		}
		if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "codex", FormalGenerationModel, dir); err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("duplicate direct import was not rejected: %v", err)
		}
	})
}

func TestGenerationBundleDirectoryImportRejectsStaleAndMachineInvalidCandidates(t *testing.T) {
	t.Run("stale generation bundle", func(t *testing.T) {
		root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
		bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		writeGenerationFixtureOutputs(t, root, batchID, dir, manifest)
		glossaryPath := filepath.Join(root, "locales", "zh-CN", "glossary.yaml")
		if err := os.WriteFile(glossaryPath, []byte("mandatory:\n  Go: Golang\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "codex", FormalGenerationModel, dir); err == nil || !strings.Contains(err.Error(), "stale") {
			t.Fatalf("stale bundle accepted: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "raw-responses")); !os.IsNotExist(err) {
			t.Fatalf("stale import left formal output: %v", err)
		}
	})

	t.Run("glossary and candidate validator", func(t *testing.T) {
		root := t.TempDir()
		copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
		writeGoExampleValidationGlossary(t, root)
		if _, _, err := RecordGlossaryReview(root, "zh-CN", "test-review", "independent-reviewer", "passed", nil); err != nil {
			t.Fatal(err)
		}
		source := []byte("package main\n\n// A goroutine sends the value through the channel.\nfunc main() { println(1) }\n")
		example := Example{ID: "example:basics/channel.go", SourcePath: "_content/tour/basics/channel.go", Source: source, SourceSHA256: sum(source)}
		catalog := &Catalog{Examples: []Example{example}}
		exported, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindExample, UnitIDs: []string{example.ID}})
		if err != nil {
			t.Fatal(err)
		}
		bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: exported.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		writeGenerationFixtureOutputs(t, root, exported.BatchID, dir, manifest)
		name := filepath.Base(manifest.ExpectedOutputs[0].BundlePath)
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		candidate := strings.Replace(string(data), " sends the value through the channel.", " 使用了幻灯片。", 1)
		if candidate == string(data) {
			t.Fatal("candidate fixture did not contain the target text")
		}
		if err := os.WriteFile(path, []byte(candidate), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "codex", FormalGenerationModel, dir); err == nil || !strings.Contains(err.Error(), "staged output machine validation failed") {
			t.Fatalf("glossary-invalid candidate accepted: %v", err)
		}
	})
}

func TestGenerationInstallPathRejectsTraversal(t *testing.T) {
	for _, path := range []string{"../outside", "raw-responses/../outside", "/tmp/output", "raw-responses\\\\outside", "raw-responses//lesson.article"} {
		if err := validateGenerationInstallPath(path); err == nil {
			t.Errorf("unsafe path %q accepted", path)
		}
	}
}

func TestPendingGenerationImportBlocksProcessing(t *testing.T) {
	root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	pending := filepath.Join(batchDir, "generation-imports", "import.pending")
	if err := os.MkdirAll(filepath.Dir(pending), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pending, []byte("unknown mutation"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err == nil || !strings.Contains(err.Error(), "pending or unknown mutation state") {
		t.Fatalf("process did not fail closed on pending import: %v", err)
	}
	for _, name := range []string{"candidates", "validation", "result.json"} {
		if _, err := os.Lstat(filepath.Join(batchDir, name)); !os.IsNotExist(err) {
			t.Fatalf("pending import created %s: %v", name, err)
		}
	}
}

func TestGenerationBundleZIPImportRejectsAttemptMismatch(t *testing.T) {
	root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
	bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
	if err != nil {
		t.Fatal(err)
	}
	outputs := t.TempDir()
	writeGenerationFixtureOutputs(t, root, batchID, outputs, manifest)
	resultBundle, _, err := PackGenerationResultBundle(root, catalog, bundle, "codex", FormalGenerationModel, outputs)
	if err != nil {
		t.Fatal(err)
	}
	files, err := ReadTransportBundle(resultBundle, 256, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var resultManifest GenerationResultBundleManifest
	if err := decodeStrictBundleJSON(files["manifest.json"], &resultManifest); err != nil {
		t.Fatal(err)
	}
	resultManifest.Attempt++
	manifestData, err := json.MarshalIndent(resultManifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestData = append(manifestData, '\n')
	entries := []TransportBundleEntry{}
	for path, data := range files {
		if path != "manifest.json" {
			entries = append(entries, TransportBundleEntry{Path: path, Data: data})
		}
	}
	mutatedResult, err := WriteDeterministicTransportBundle(manifestData, entries)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportGenerationResultBundle(root, catalog, bundle, mutatedResult); err == nil || !strings.Contains(err.Error(), "identity does not match") {
		t.Fatalf("attempt-mismatched result bundle was accepted: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "raw-responses")); !os.IsNotExist(err) {
		t.Fatalf("attempt mismatch left formal output: %v", err)
	}
}

func TestFormalGenerationIdentitySupportsProviderSpecificAndHistoricalModels(t *testing.T) {
	for provider, want := range map[string]string{
		"codex":   FormalGenerationModel,
		"chatgpt": FormalChatGPTGenerationModel,
	} {
		got, err := DefaultGenerationModelForProvider(provider)
		if err != nil || got != want {
			t.Fatalf("DefaultGenerationModelForProvider(%q) = %q, %v; want %q", provider, got, err, want)
		}
	}
	if _, err := DefaultGenerationModelForProvider("unknown"); err == nil {
		t.Fatal("unknown provider returned a default model")
	}
	tests := []struct {
		name, provider, model, batchID string
		wantErr                        bool
	}{
		{name: "codex default sol high", provider: "codex", model: FormalGenerationModel, batchID: "codex-locale-001"},
		{name: "codex previously approved luna high", provider: "codex", model: PreviousCodexGenerationModel, batchID: "codex-locale-002"},
		{name: "chatgpt sol high", provider: "chatgpt", model: FormalChatGPTGenerationModel, batchID: "chatgpt-locale-001"},
		{name: "chatgpt rejects codex-only luna model", provider: "chatgpt", model: PreviousCodexGenerationModel, batchID: "chatgpt-locale-002", wantErr: true},
		{name: "codex rejects unknown model", provider: "codex", model: "other-model", batchID: "codex-locale-003", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := requireFormalGenerationIdentity(tc.provider, tc.model, tc.batchID)
			if (err != nil) != tc.wantErr {
				t.Fatalf("requireFormalGenerationIdentity() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
