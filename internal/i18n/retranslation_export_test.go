package i18n

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func retranslationTestCatalog(count int) *Catalog {
	catalog := &Catalog{}
	for i := 1; i <= count; i++ {
		source := []byte("* Page\n\nUse `Go` on this page.\n")
		catalog.Pages = append(catalog.Pages, Page{
			ID: "lesson/" + fmtInt(i), Article: "lesson.article", SectionNumber: i,
			Route: "/lesson/" + fmtInt(i), SourceSHA256: sum(source), Source: source,
		})
	}
	return catalog
}

func fmtInt(value int) string {
	return strconv.Itoa(value)
}

func writeRetranslationTestGlossary(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "locales", "zh-CN")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "glossary.yaml"), []byte("mandatory:\n  Go: Go\nkeep:\n  - Go\n  - gofmt\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func retranslationTestExample(id, path, source string) Example {
	data := []byte(source)
	return Example{ID: id, SourcePath: path, Source: data, SourceSHA256: sum(data)}
}

func readRetranslationManifest(t *testing.T, root, batchID string) RetranslationBatchManifest {
	t.Helper()
	path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest RetranslationBatchManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func TestRetranslationExportAutomaticBatchProgression(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	catalog := retranslationTestCatalog(23)
	var gotIDs []string
	for batch, wantCount := range []int{10, 10, 3} {
		result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Limit: 10})
		if err != nil {
			t.Fatalf("batch %d: %v", batch+1, err)
		}
		wantBatchID := "chatgpt-zh-CN-00" + fmtInt(batch+1)
		if result.BatchID != wantBatchID || result.PageCount != wantCount || result.AllExported {
			t.Fatalf("batch %d result = %+v, want id=%s count=%d", batch+1, result, wantBatchID, wantCount)
		}
		manifest := readRetranslationManifest(t, root, result.BatchID)
		if manifest.PageCount != wantCount || manifest.ProtectionMode != "default" || manifest.TranslationUnit != "present.Section" {
			t.Fatalf("batch %d manifest = %+v", batch+1, manifest)
		}
		entries, err := os.ReadDir(filepath.Join(root, result.BatchPath, "inputs"))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != wantCount {
			t.Fatalf("batch %d inputs = %d, want %d", batch+1, len(entries), wantCount)
		}
		for _, page := range manifest.Pages {
			input, err := os.ReadFile(filepath.Join(root, result.BatchPath, filepath.FromSlash(page.InputPath)))
			if err != nil {
				t.Fatal(err)
			}
			if len(input) == 0 || sum(input) != page.InputSHA256 {
				t.Fatalf("%s input hash/content mismatch", page.PageID)
			}
			gotIDs = append(gotIDs, page.PageID)
		}
	}
	wantIDs := make([]string, 0, len(catalog.Pages))
	for _, page := range catalog.Pages {
		wantIDs = append(wantIDs, page.ID)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("exported IDs = %v, want catalog order %v", gotIDs, wantIDs)
	}
	before, err := os.ReadDir(filepath.Join(root, "data", "retranslation-runs", "zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadDir(filepath.Join(root, "data", "retranslation-runs", "zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.AllExported || result.PageCount != 0 || len(after) != len(before) {
		t.Fatalf("completed export result=%+v directories=%d->%d", result, len(before), len(after))
	}
}

func TestRetranslationExportLimitTwentyKeepsIndependentPages(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	catalog := retranslationTestCatalog(23)
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindPage, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	manifest := readRetranslationManifest(t, root, result.BatchID)
	if result.PageCount != 20 || manifest.PageCount != 20 || len(manifest.Pages) != 20 {
		t.Fatalf("limit 20 result=%+v manifest count=%d", result, len(manifest.Pages))
	}
}

func TestRetranslationExportExplicitPageKindDefaultsToTen(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	result, err := ExportRetranslationBatch(root, retranslationTestCatalog(23), RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindPage})
	if err != nil {
		t.Fatal(err)
	}
	if result.PageCount != 10 {
		t.Fatalf("explicit page kind count=%d, want 10", result.PageCount)
	}
}

func TestRetranslationExportAutomaticExampleProgression(t *testing.T) {
	catalog, err := BuildCatalog(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	seen := map[string]bool{}
	for batch, wantCount := range []int{10, 9} {
		result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindExample})
		if err != nil {
			t.Fatalf("example batch %d: %v", batch+1, err)
		}
		if result.PageCount != wantCount {
			t.Fatalf("example batch %d count=%d, want %d", batch+1, result.PageCount, wantCount)
		}
		manifest := readRetranslationManifest(t, root, result.BatchID)
		if manifest.TranslationUnit != "go.example" || len(manifest.Pages) != wantCount {
			t.Fatalf("example batch %d manifest=%+v", batch+1, manifest)
		}
		for _, record := range manifest.Pages {
			if seen[record.UnitID] {
				t.Fatalf("example %s was exported twice", record.UnitID)
			}
			seen[record.UnitID] = true
			unit, err := catalog.Unit(record.UnitID)
			if err != nil {
				t.Fatal(err)
			}
			hasContent, err := hasTranslatableGoExampleComment(unit.Source)
			if err != nil || !hasContent {
				t.Fatalf("ineligible example %s was exported: hasContent=%t err=%v", record.UnitID, hasContent, err)
			}
		}
	}
	if len(seen) != 19 {
		t.Fatalf("automatically exported examples=%d, want 19", len(seen))
	}
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindExample})
	if err != nil {
		t.Fatal(err)
	}
	if !result.AllExported || result.PageCount != 0 {
		t.Fatalf("completed example export result=%+v", result)
	}
}

func TestRetranslationExportAutomaticExampleLimitTwentySelectsCorpusNineteen(t *testing.T) {
	catalog, err := BuildCatalog(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindExample, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.PageCount != 19 || len(result.PageIDs) != 19 {
		t.Fatalf("example limit 20 result=%+v, want 19 eligible units", result)
	}
}

func TestRetranslationExportExplicitExamples(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	examples := []Example{
		retranslationTestExample("example:basics/one.go", "_content/tour/basics/one.go", "package main\n\n// Go starts this comment.\nfunc main() {}\n"),
		retranslationTestExample("example:basics/two.go", "_content/tour/basics/two.go", "package main\n\n// Run gofmt before continuing.\nfunc main() {}\n"),
	}
	catalog := &Catalog{Examples: examples}
	ids := []string{examples[0].ID, examples[1].ID}
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", PageIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	manifest := readRetranslationManifest(t, root, result.BatchID)
	if manifest.TranslationUnit != "go.example" || manifest.PageCount != 2 || !reflect.DeepEqual(result.PageIDs, ids) {
		t.Fatalf("example export result=%+v manifest=%+v", result, manifest)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for i, record := range manifest.Pages {
		unit, err := catalog.Unit(ids[i])
		if err != nil {
			t.Fatal(err)
		}
		if record.UnitID != unit.ID || record.UnitKind != UnitKindExample || record.SourcePath != unit.SourcePath || record.PageID != "" {
			t.Fatalf("example manifest record=%+v", record)
		}
		if filepath.Ext(record.InputPath) != ".go" || record.SourceSHA256 != sum(unit.Source) {
			t.Fatalf("example identity/path record=%+v", record)
		}
		input, err := os.ReadFile(filepath.Join(root, result.BatchPath, filepath.FromSlash(record.InputPath)))
		if err != nil {
			t.Fatal(err)
		}
		protected, err := prepareTranslationUnitInput(unit, glossary)
		if err != nil {
			t.Fatal(err)
		}
		if string(input) != protected.Text || sum(input) != record.InputSHA256 || !strings.Contains(string(input), "GTI18N") {
			t.Fatalf("%s protected input/hash mismatch: %q", unit.ID, input)
		}
	}
}

func TestRetranslationExportExplicitUnitKindValidation(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	example := retranslationTestExample("example:demo/comment.go", "_content/tour/demo/comment.go", "package main\n\n// Translate this comment.\nfunc main() {}\n")
	catalog := retranslationTestCatalog(1)
	catalog.Examples = []Example{example}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindExample, PageIDs: []string{example.ID}}); err != nil {
		t.Fatalf("matching explicit example kind: %v", err)
	}

	root = t.TempDir()
	writeRetranslationTestGlossary(t, root)
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindPage, PageIDs: []string{example.ID}}); err == nil || !strings.Contains(err.Error(), "不一致") {
		t.Fatalf("mismatched explicit kind error=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "retranslation-runs")); !os.IsNotExist(err) {
		t.Fatalf("kind mismatch created batch data: %v", err)
	}

	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKind("invalid")}); err == nil || !strings.Contains(err.Error(), "只支持 page 或 example") {
		t.Fatalf("invalid kind error=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "retranslation-runs")); !os.IsNotExist(err) {
		t.Fatalf("invalid kind created batch data: %v", err)
	}
}

func TestRetranslationExportRejectsNonTranslatableAndMixedExamples(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	catalog := retranslationTestCatalog(1)
	catalog.Examples = []Example{
		retranslationTestExample("example:demo/empty.go", "_content/tour/demo/empty.go", "package main\n\nfunc main() {}\n"),
		retranslationTestExample("example:demo/comment.go", "_content/tour/demo/comment.go", "package main\n\n// Translate this comment.\nfunc main() {}\n"),
	}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", PageIDs: []string{catalog.Examples[0].ID}}); err == nil || !strings.Contains(err.Error(), "没有需要翻译的普通自然语言注释") {
		t.Fatalf("non-translatable example error=%v", err)
	}
	base := filepath.Join(root, "data", "retranslation-runs", "zh-CN")
	if entries, err := os.ReadDir(base); err == nil && len(entries) != 0 {
		t.Fatalf("failed export created batch entries: %v", entries)
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", PageIDs: []string{"lesson/1", catalog.Examples[1].ID}}); err == nil || !strings.Contains(err.Error(), "不能混合") {
		t.Fatalf("mixed-kind error=%v", err)
	}
}

func TestRetranslationExportDefaultSelectionRemainsPageOnly(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	catalog := retranslationTestCatalog(1)
	catalog.Examples = []Example{retranslationTestExample("example:demo/comment.go", "_content/tour/demo/comment.go", "package main\n\n// Translate this comment.\nfunc main() {}\n")}
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	manifest := readRetranslationManifest(t, root, result.BatchID)
	if !reflect.DeepEqual(result.PageIDs, []string{"lesson/1"}) || manifest.TranslationUnit != "present.Section" {
		t.Fatalf("default export selected non-page unit: result=%+v manifest=%+v", result, manifest)
	}
}

func TestRetranslationExportAutomaticProgressionAfterFutureManualPage(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	catalog := retranslationTestCatalog(23)

	manualPageID := catalog.Pages[14].ID
	manual, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
		Locale: "zh-CN", BatchID: "manual-future-page", PageIDs: []string{manualPageID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manual.PageIDs, []string{manualPageID}) {
		t.Fatalf("manual page IDs = %v, want [%s]", manual.PageIDs, manualPageID)
	}

	exported := map[string]bool{manualPageID: true}
	for {
		var want []string
		for _, page := range catalog.Pages {
			if exported[page.ID] {
				continue
			}
			want = append(want, page.ID)
			if len(want) == 10 {
				break
			}
		}

		result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(want) == 0 {
			if !result.AllExported || result.PageCount != 0 {
				t.Fatalf("completed result = %+v", result)
			}
			break
		}
		if result.AllExported || !reflect.DeepEqual(result.PageIDs, want) {
			t.Fatalf("automatic page IDs = %v, want earliest unexported %v", result.PageIDs, want)
		}
		for _, pageID := range result.PageIDs {
			if exported[pageID] {
				t.Fatalf("page_id %q exported more than once", pageID)
			}
			exported[pageID] = true
		}
	}

	base := filepath.Join(root, "data", "retranslation-runs", "zh-CN")
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		manifest := readRetranslationManifest(t, root, entry.Name())
		for _, page := range manifest.Pages {
			counts[page.PageID]++
		}
	}
	if len(counts) != len(catalog.Pages) {
		t.Fatalf("exported page count = %d, want %d", len(counts), len(catalog.Pages))
	}
	for _, page := range catalog.Pages {
		if counts[page.ID] != 1 {
			t.Errorf("page_id %q occurrence count = %d, want 1", page.ID, counts[page.ID])
		}
	}
}

func TestRetranslationExportManualSelectionErrors(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	catalog := retranslationTestCatalog(3)
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", PageIDs: []string{"lesson/1", "lesson/1"}}); err == nil || !strings.Contains(err.Error(), "duplicate requested") {
		t.Fatalf("duplicate request error = %v", err)
	}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", PageIDs: []string{"lesson/1", "lesson/1"}, AllowReexport: true}); err == nil || !strings.Contains(err.Error(), "duplicate requested") {
		t.Fatalf("duplicate reexport request error = %v", err)
	}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", PageIDs: []string{"missing/1"}}); err == nil || !strings.Contains(err.Error(), "unknown page_id") {
		t.Fatalf("unknown request error = %v", err)
	}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", PageIDs: []string{"missing/1"}, AllowReexport: true}); err == nil || !strings.Contains(err.Error(), "unknown page_id") {
		t.Fatalf("unknown reexport request error = %v", err)
	}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", AllowReexport: true}); err == nil || !strings.Contains(err.Error(), "requires at least one --id") {
		t.Fatalf("reexport without id error = %v", err)
	}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Limit: -1}); err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("invalid limit error = %v", err)
	}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Limit: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", PageIDs: []string{"lesson/1"}}); err == nil || !strings.Contains(err.Error(), "already exported") {
		t.Fatalf("already exported error = %v", err)
	}
	revision, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", PageIDs: []string{"lesson/1"}, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	if revision.BatchID != "chatgpt-zh-CN-002" || len(revision.PageIDs) != 1 || revision.PageIDs[0] != "lesson/1" {
		t.Fatalf("revision result = %+v", revision)
	}
	automatic, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if automatic.BatchID != "chatgpt-zh-CN-003" || !reflect.DeepEqual(automatic.PageIDs, []string{"lesson/2", "lesson/3"}) {
		t.Fatalf("automatic result after revision = %+v", automatic)
	}
}

func TestRetranslationExportRejectsBrokenAndDuplicateHistory(t *testing.T) {
	catalog := retranslationTestCatalog(3)
	t.Run("broken manifest", func(t *testing.T) {
		root := t.TempDir()
		writeRetranslationTestGlossary(t, root)
		dir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", "chatgpt-zh-CN-001")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN"}); err == nil || !strings.Contains(err.Error(), "parse retranslation manifest") {
			t.Fatalf("broken manifest error = %v", err)
		}
	})
	t.Run("duplicate historical page remains exported", func(t *testing.T) {
		root := t.TempDir()
		writeRetranslationTestGlossary(t, root)
		base := filepath.Join(root, "data", "retranslation-runs", "zh-CN")
		for i := 1; i <= 2; i++ {
			batchID := "chatgpt-zh-CN-00" + fmtInt(i)
			dir := filepath.Join(base, batchID)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			manifest := RetranslationBatchManifest{SchemaVersion: 1, BatchID: batchID, Locale: "zh-CN", ProtectionMode: "default", TranslationUnit: "present.Section", PageCount: 1, Pages: []RetranslationBatchPage{{PageID: "lesson/1"}}}
			if err := writeTranslationJSON(filepath.Join(dir, "manifest.json"), manifest); err != nil {
				t.Fatal(err)
			}
		}
		result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if result.BatchID != "chatgpt-zh-CN-003" || !reflect.DeepEqual(result.PageIDs, []string{"lesson/2", "lesson/3"}) {
			t.Fatalf("automatic result with duplicate history = %+v", result)
		}
	})
}

func TestRetranslationExportRejectsExistingExplicitBatch(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	catalog := retranslationTestCatalog(2)
	batchID := "manual-recovery"
	dir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := RetranslationBatchManifest{SchemaVersion: 1, BatchID: batchID, Locale: "zh-CN", ProtectionMode: "default", TranslationUnit: "present.Section", PageCount: 1, Pages: []RetranslationBatchPage{{PageID: "lesson/1"}}}
	if err := writeTranslationJSON(filepath.Join(dir, "manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", BatchID: batchID, PageIDs: []string{"lesson/2"}}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing batch error = %v", err)
	}
}

func TestRetranslationExportMatchesTranslationRunnerDefaultInput(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	catalog := retranslationTestCatalog(1)
	page := catalog.Pages[0]
	if err := os.MkdirAll(filepath.Join(root, "locales", "zh-CN"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeStatuses(filepath.Join(root, "locales", "zh-CN", "status.tsv"), []Status{{PageID: page.ID, State: "pending", SourceSHA256: page.SourceSHA256}}); err != nil {
		t.Fatal(err)
	}
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	exported, err := os.ReadFile(filepath.Join(root, result.BatchPath, "inputs", flattenedPageArticleName(page.ID)))
	if err != nil {
		t.Fatal(err)
	}
	var sent TranslationAPIRequest
	client := &TranslationClient{Endpoint: "https://example.invalid", HTTP: mockHTTP(func(request *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatal(err)
		}
		response := `{"id":"test","choices":[{"message":{"role":"assistant","content":"invalid"},"finish_reason":"stop"}],"usage":{}}`
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})}
	runner := TranslationRunner{Root: root, Catalog: catalog, Client: client, Dev: true}
	if _, err := runner.Run(context.Background(), page.ID, "zh-CN", "test-key"); err != nil {
		t.Fatal(err)
	}
	if len(sent.Messages) != 2 || !strings.HasSuffix(sent.Messages[1].Content, string(exported)) {
		t.Fatalf("runner user message does not contain the exact exported protected input\nexported: %q\nuser: %q", exported, sent.Messages[1].Content)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	protected := prepareDefaultTranslationInput(page.Source, page.SourceSHA256, glossary)
	if string(exported) != protected.Text {
		t.Fatalf("exported bytes differ from shared Default input\nexported: %q\ndefault: %q", exported, protected.Text)
	}
}
