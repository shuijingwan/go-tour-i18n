package i18n

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func generationRecoveryFixture(t *testing.T, count int) (string, *Catalog, string, []byte, GenerationBundleManifest) {
	t.Helper()
	root := t.TempDir()
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	writeRetranslationTestGlossaryForLocale(t, root, "zh-CN")
	catalog := retranslationTestCatalog(count)
	exported, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
		Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, Limit: count,
	})
	if err != nil {
		t.Fatal(err)
	}
	bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{
		Locale: "zh-CN", BatchID: exported.BatchID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return root, catalog, exported.BatchID, bundle, manifest
}
func generationRecoveryInput(t *testing.T, root, batchID string, expected GenerationBundleExpectedOutput) []byte {
	t.Helper()
	path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "inputs", filepath.Base(expected.BundlePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeInterruptedGenerationTarGzip(t *testing.T, root, batchID, path string, expected []GenerationBundleExpectedOutput) {
	t.Helper()
	var tarData bytes.Buffer
	writer := tar.NewWriter(&tarData)
	for _, output := range expected {
		data := generationRecoveryInput(t, root, batchID, output)
		header := &tar.Header{Name: filepath.Base(output.BundlePath), Mode: 0600, Size: int64(len(data))}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressed)
	if _, err := gzipWriter.Write(tarData.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	data := compressed.Bytes()
	if len(data) <= 8 {
		t.Fatal("gzip fixture unexpectedly short")
	}
	if err := os.WriteFile(path, data[:len(data)-8], 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGenerationRecoveryRecovers53Of60FromInterruptedTarGzip(t *testing.T) {
	root, catalog, batchID, bundle, manifest := generationRecoveryFixture(t, 60)
	archive := filepath.Join(t.TempDir(), "interrupted-staging.tar.gz")
	writeInterruptedGenerationTarGzip(t, root, batchID, archive, manifest.ExpectedOutputs[:53])
	target := filepath.Join(t.TempDir(), "recovered")
	result, err := RecoverGenerationOutputs(root, catalog, bundle, []string{archive}, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Recovered) != 53 || len(result.Missing) != 7 {
		t.Fatalf("recovery counts: recovered=%d missing=%d", len(result.Recovered), len(result.Missing))
	}
	if len(result.Issues) == 0 {
		t.Fatal("truncated gzip artifact did not preserve integrity failure evidence")
	}
	for index, missing := range result.Missing {
		want := manifest.ExpectedOutputs[index+53].BundlePath
		if missing != want {
			t.Fatalf("missing[%d]=%q, want %q", index, missing, want)
		}
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 53 {
		t.Fatalf("recovered staging entries=%d err=%v", len(entries), err)
	}
	for _, expected := range manifest.ExpectedOutputs[53:] {
		data := generationRecoveryInput(t, root, batchID, expected)
		if err := os.WriteFile(filepath.Join(target, filepath.Base(expected.BundlePath)), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	imported, err := ImportGenerationOutputDirectory(root, catalog, bundle, "chatgpt", FormalChatGPTGenerationModel, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.InstalledPaths) != 60 {
		t.Fatalf("imported outputs=%d, want 60", len(imported.InstalledPaths))
	}
}

func TestGenerationRecoveryCombinesSourcesAndRejectsInvalidCandidate(t *testing.T) {
	root, catalog, batchID, bundle, manifest := generationRecoveryFixture(t, 2)
	first := manifest.ExpectedOutputs[0]
	second := manifest.ExpectedOutputs[1]
	sourceDir := t.TempDir()
	firstData := generationRecoveryInput(t, root, batchID, first)
	broken := bytes.Replace(firstData, []byte("⟪GTI18N_"), []byte("⟪BROKEN_"), 1)
	if bytes.Equal(firstData, broken) {
		t.Fatal("fixture has no protected token")
	}
	if err := os.WriteFile(filepath.Join(sourceDir, filepath.Base(first.BundlePath)), broken, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, filepath.Base(second.BundlePath)), generationRecoveryInput(t, root, batchID, second), 0644); err != nil {
		t.Fatal(err)
	}
	fallbackDir := t.TempDir()
	fallback := filepath.Join(fallbackDir, filepath.Base(first.BundlePath))
	if err := os.WriteFile(fallback, firstData, 0644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "combined")
	result, err := RecoverGenerationOutputs(root, catalog, bundle, []string{sourceDir, fallback}, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Recovered) != 2 || len(result.Missing) != 0 {
		t.Fatalf("combined recovery=%+v", result)
	}
	if len(result.Issues) != 1 || !strings.Contains(result.Issues[0].Reason, "protected output validation failed") {
		t.Fatalf("invalid candidate evidence=%+v", result.Issues)
	}
}

func TestGenerationRecoveryRejectsConflictingValidCandidates(t *testing.T) {
	root, catalog, batchID, bundle, manifest := generationRecoveryFixture(t, 1)
	expected := manifest.ExpectedOutputs[0]
	original := generationRecoveryInput(t, root, batchID, expected)
	alternate := bytes.Replace(original, []byte("Use "), []byte("Please use "), 1)
	if bytes.Equal(original, alternate) {
		t.Fatal("fixture ordinary text was not changed")
	}
	firstDir := t.TempDir()
	secondDir := t.TempDir()
	name := filepath.Base(expected.BundlePath)
	if err := os.WriteFile(filepath.Join(firstDir, name), original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondDir, name), alternate, 0644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "conflict")
	if _, err := RecoverGenerationOutputs(root, catalog, bundle, []string{firstDir, secondDir}, target); err == nil || !strings.Contains(err.Error(), "conflicting valid recovery candidates") {
		t.Fatalf("conflicting valid candidates were not rejected: %v", err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("conflict recovery created output: %v", err)
	}
}

func TestGenerationRecoveryRejectsStaleBundleBeforeWriting(t *testing.T) {
	root, catalog, _, bundle, _ := generationRecoveryFixture(t, 2)
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "stale-recovery")
	glossary := filepath.Join(root, "locales", "zh-CN", "glossary.yaml")
	if err := os.WriteFile(glossary, []byte("mandatory:\n  Go: Golang\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverGenerationOutputs(root, catalog, bundle, []string{source}, target); err == nil {
		t.Fatal("stale generation bundle was accepted for recovery")
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("stale recovery created output: %v", err)
	}
}
