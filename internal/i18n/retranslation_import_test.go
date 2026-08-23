package i18n

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func writeTranslationResult(t *testing.T, root, batchID, locale string, units []retranslationResultFileUnit) string {
	t.Helper()
	data, err := json.Marshal(retranslationResultFile{BatchID: batchID, Locale: locale, Units: units})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "translation-result.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func makeRetranslationImportBatch(t *testing.T, count int) (string, string, RetranslationBatchManifest) {
	t.Helper()
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	result, err := ExportRetranslationBatch(root, retranslationTestCatalog(count), RetranslationExportOptions{Locale: "zh-CN", Limit: count})
	if err != nil {
		t.Fatal(err)
	}
	return root, result.BatchID, readRetranslationManifest(t, root, result.BatchID)
}

func writeRawResponseArchive(t *testing.T, root string, manifest RetranslationBatchManifest, remove string, extra bool) string {
	t.Helper()
	path := filepath.Join(root, "raw-responses.zip")
	output, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(output)
	names := make([]string, 0, len(manifest.Units))
	for _, unit := range manifest.Units {
		name := filepath.Base(filepath.FromSlash(unit.InputPath))
		if name != remove {
			names = append(names, name)
		}
	}
	if extra {
		names = append(names, "extra.article")
	}
	sort.Strings(names)
	for _, name := range names {
		entry, err := writer.Create("raw-responses/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("* Imported\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRetranslationImportRawResponses(t *testing.T) {
	root, batchID, manifest := makeRetranslationImportBatch(t, 2)
	archive := writeRawResponseArchive(t, root, manifest, "", false)
	result, err := ImportRetranslationRawResponses(root, RetranslationImportOptions{Locale: "zh-CN", BatchID: batchID, Archive: archive})
	if err != nil {
		t.Fatal(err)
	}
	if result.Locale != "zh-CN" || result.BatchID != batchID || result.UnitCount != 2 || result.RawDirectory != filepath.ToSlash(filepath.Join("data", "retranslation-runs", "zh-CN", batchID, "raw-responses")) {
		t.Fatalf("import result = %+v", result)
	}
	for _, unit := range manifest.Units {
		name := filepath.Base(filepath.FromSlash(unit.InputPath))
		data, err := os.ReadFile(filepath.Join(root, result.RawDirectory, name))
		if err != nil || string(data) != "* Imported\n" {
			t.Fatalf("imported raw response %s = %q, err=%v", name, data, err)
		}
	}
	for _, name := range []string{"result.json", "candidates", "validation"} {
		if _, err := os.Stat(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, name)); !os.IsNotExist(err) {
			t.Fatalf("import unexpectedly created %s: %v", name, err)
		}
	}
}

func TestRetranslationImportRejectsMissingAndExtraRawResponses(t *testing.T) {
	for _, test := range []struct {
		name   string
		remove string
		extra  bool
		want   string
	}{
		{name: "missing", remove: "lesson-1.article", want: "raw response count"},
		{name: "extra", extra: true, want: "unexpected archive raw response"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, batchID, manifest := makeRetranslationImportBatch(t, 2)
			archive := writeRawResponseArchive(t, root, manifest, test.remove, test.extra)
			_, err := ImportRetranslationRawResponses(root, RetranslationImportOptions{Locale: "zh-CN", BatchID: batchID, Archive: archive})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("import error = %v, want %q", err, test.want)
			}
			rawDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "raw-responses")
			if _, statErr := os.Stat(rawDir); !os.IsNotExist(statErr) {
				t.Fatalf("failed import created raw-responses: %v", statErr)
			}
		})
	}
}

func TestRetranslationImportRejectsMissingBatch(t *testing.T) {
	root := t.TempDir()
	_, err := ImportRetranslationRawResponses(root, RetranslationImportOptions{Locale: "zh-CN", BatchID: "chatgpt-zh-CN-001", Archive: filepath.Join(root, "missing.zip")})
	if err == nil || !strings.Contains(err.Error(), "inspect retranslation batch") {
		t.Fatalf("import error = %v", err)
	}
}

func TestRetranslationImportRejectsExistingRawResponses(t *testing.T) {
	root, batchID, manifest := makeRetranslationImportBatch(t, 1)
	archive := writeRawResponseArchive(t, root, manifest, "", false)
	rawDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "raw-responses")
	if err := os.Mkdir(rawDir, 0755); err != nil {
		t.Fatal(err)
	}
	_, err := ImportRetranslationRawResponses(root, RetranslationImportOptions{Locale: "zh-CN", BatchID: batchID, Archive: archive})
	if err == nil || !strings.Contains(err.Error(), "already has raw responses") {
		t.Fatalf("import error = %v", err)
	}
}

func TestImportRetranslationResultImportsIncrementalRawResponses(t *testing.T) {
	root, batchID, _ := makeRetranslationImportBatch(t, 2)
	first := writeTranslationResult(t, root, batchID, "zh-CN", []retranslationResultFileUnit{{UnitID: "lesson-1", Content: "* 第一篇\n"}})
	result, err := ImportRetranslationResult(root, first)
	if err != nil {
		t.Fatal(err)
	}
	if result.UnitCount != 1 {
		t.Fatalf("unit count = %d, want 1", result.UnitCount)
	}
	data, err := os.ReadFile(filepath.Join(root, result.RawDirectory, "lesson-1.article"))
	if err != nil || string(data) != "* 第一篇\n" {
		t.Fatalf("first raw response = %q, err=%v", data, err)
	}
	second := writeTranslationResult(t, root, batchID, "zh-CN", []retranslationResultFileUnit{{UnitID: "lesson-2", Content: "* 第二篇\n"}})
	if _, err := ImportRetranslationResult(root, second); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportRetranslationResult(root, first); err == nil || !strings.Contains(err.Error(), "already has raw response") {
		t.Fatalf("duplicate import error = %v", err)
	}
}
