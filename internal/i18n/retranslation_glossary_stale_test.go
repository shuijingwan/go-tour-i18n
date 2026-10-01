package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func glossaryStaleTestCatalog() *Catalog {
	pages := []struct {
		id, title, body string
	}{
		{"lesson/1", "One", "If an initializer is present, use it."},
		{"lesson/2", "Two", "This sentence has no affected keep term."},
		{"lesson/3", "Three", "Test that a key is present before use."},
	}
	catalog := &Catalog{}
	for index, page := range pages {
		source := []byte("* " + page.title + "\n\n" + page.body + "\n")
		catalog.Pages = append(catalog.Pages, Page{
			ID: page.id, Article: "lesson.article", SectionNumber: index + 1,
			Route: "/" + page.id, Source: source, SourceSHA256: sum(source),
		})
	}
	return catalog
}

func writeGlossaryStaleTestGlossary(t *testing.T, root, reviewID string, keepPresent bool) {
	t.Helper()
	dir := filepath.Join(root, "locales", "zh-CN")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	text := "mandatory:\n  Go: Go\nkeep:\n  - Go\n"
	if keepPresent {
		text += "  - present\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "glossary.yaml"), []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	if reviewID != "" {
		if _, _, err := RecordGlossaryReview(root, "zh-CN", reviewID, "independent-reviewer", "passed", nil); err != nil {
			t.Fatal(err)
		}
	}
}

func processGlossaryStaleTestBatch(t *testing.T, root string, catalog *Catalog, batchID string) {
	t.Helper()
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	manifest := readRetranslationManifest(t, root, batchID)
	if err := os.Mkdir(filepath.Join(batchDir, "raw-responses"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, record := range manifest.Units {
		input, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(record.InputPath)))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(batchDir, "raw-responses", filepath.Base(record.InputPath)), input, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
}

func snapshotDirectoryBytes(t *testing.T, root string) map[string][]byte {
	t.Helper()
	result := map[string][]byte{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(relative)] = data
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestGlossaryStaleRecoveryExportsOnlyDriftedUnitsAndBuildsFullSnapshot(t *testing.T) {
	root := t.TempDir()
	catalog := glossaryStaleTestCatalog()
	writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
	initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
		Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitKind: UnitKindPage,
	})
	if err != nil {
		t.Fatal(err)
	}
	processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
	materializeSnapshotSources(t, root, catalog)
	oldSnapshot, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-001"})
	if err != nil {
		t.Fatal(err)
	}
	for _, unit := range oldSnapshot.Units {
		if _, err := RecordQualityCheckResults(root, catalog, QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "qc-001", UnitIDs: []string{unit.UnitID}, Rating: "A"}); err != nil {
			t.Fatal(err)
		}
	}
	oldBatchDir := filepath.Join(root, initial.BatchPath)
	oldBytes := snapshotDirectoryBytes(t, oldBatchDir)

	writeGlossaryStaleTestGlossary(t, root, "current-glossary", false)
	recovery, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
		Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, AllowReexport: true, GlossaryStale: true,
		UnitIDs: []string{"lesson/1", "lesson/3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if recovery.BatchID != "chatgpt-zh-CN-002" || !reflect.DeepEqual(recovery.UnitIDs, []string{"lesson/1", "lesson/3"}) {
		t.Fatalf("recovery result=%+v", recovery)
	}
	manifest := readRetranslationManifest(t, root, recovery.BatchID)
	for _, unit := range manifest.Units {
		if unit.ReexportReason != RetranslationReexportReasonGlossaryInputStale || unit.PreviousBatchID != initial.BatchID ||
			unit.PreviousInputPath == "" || !validSHA256(unit.PreviousInputSHA256) || unit.RevisionFeedbackSource != "" || unit.PreviousSnapshotID != "" {
			t.Fatalf("glossary-stale provenance=%+v", unit)
		}
	}
	if got := snapshotDirectoryBytes(t, oldBatchDir); !reflect.DeepEqual(got, oldBytes) {
		t.Fatal("glossary-stale export mutated the previous batch")
	}

	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	bundle, generation, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: recovery.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	if generation.TaskKind != "translation-unit-glossary-stale-recovery" {
		t.Fatalf("generation task_kind=%q", generation.TaskKind)
	}
	staging := t.TempDir()
	for _, expected := range generation.ExpectedOutputs {
		var record RetranslationBatchUnit
		for _, candidate := range manifest.Units {
			if candidate.UnitID == expected.UnitID {
				record = candidate
				break
			}
		}
		input, err := os.ReadFile(filepath.Join(root, recovery.BatchPath, filepath.FromSlash(record.InputPath)))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(staging, filepath.Base(expected.BundlePath)), input, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "chatgpt", FormalChatGPTGenerationModel, staging); err != nil {
		t.Fatal(err)
	}
	processed, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: recovery.BatchID})
	if err != nil || processed.ValidationPassed != 2 {
		t.Fatalf("process recovery result=%+v err=%v", processed, err)
	}
	newSnapshot, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshotUnit(t, newSnapshot, "lesson/1").SelectedBatchID != recovery.BatchID || snapshotUnit(t, newSnapshot, "lesson/3").SelectedBatchID != recovery.BatchID ||
		snapshotUnit(t, newSnapshot, "lesson/2").SelectedBatchID != initial.BatchID {
		t.Fatalf("new Snapshot did not combine recovered and trusted candidates: %+v", newSnapshot.Units)
	}
	scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
	if err != nil {
		t.Fatal(err)
	}
	if scope.CarryForwardCount != 0 || scope.PendingCount != 3 {
		t.Fatalf("glossary change carried old QC forward: %+v", scope)
	}
	for _, pending := range scope.Pending {
		if pending.Reason != QualityCheckScopeReasonGlossaryChanged || pending.RequiredAction != QualityCheckActionRequired {
			t.Fatalf("pending QC after glossary change=%+v", pending)
		}
	}
	if _, err := readQualityCheckSnapshotForReview(root, "zh-CN", "qc-001"); err == nil || !strings.Contains(err.Error(), "glossary hash mismatch") {
		t.Fatalf("old Snapshot unexpectedly became current: %v", err)
	}
}

func TestGlossaryStaleRecoveryRejectsUnaffectedUnitAndStaleReview(t *testing.T) {
	root := t.TempDir()
	catalog := glossaryStaleTestCatalog()
	writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
	initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindPage})
	if err != nil {
		t.Fatal(err)
	}
	processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
	writeGlossaryStaleTestGlossary(t, root, "", false)
	options := RetranslationExportOptions{Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{"lesson/1"}}
	if _, err := ExportRetranslationBatch(root, catalog, options); err == nil || !strings.Contains(err.Error(), "Glossary Review gate missing or stale") {
		t.Fatalf("stale Glossary Review error=%v", err)
	}
	writeGlossaryStaleTestGlossary(t, root, "current-glossary", false)
	options.UnitIDs = []string{"lesson/2"}
	if _, err := ExportRetranslationBatch(root, catalog, options); err == nil || !strings.Contains(err.Error(), "has no glossary-induced drift") {
		t.Fatalf("unaffected unit error=%v", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "data", "retranslation-runs", "zh-CN"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("failed recovery created batch: entries=%v err=%v", entries, err)
	}
}

func TestGlossaryStaleRecoveryRejectsUntrustedLatestEvidence(t *testing.T) {
	tests := []struct {
		name, want string
		mutate     func(t *testing.T, root, batchID string)
		processed  bool
	}{
		{name: "missing result", want: "requires a latest processed result"},
		{name: "failed result", processed: true, want: "is not passed", mutate: func(t *testing.T, root, batchID string) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "result.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			result, err := decodeRetranslationProcessResult(data)
			if err != nil {
				t.Fatal(err)
			}
			result.Units[0].Status = "validation_failed"
			result.ValidationPassed--
			result.ValidationFailed++
			if err := writeTranslationJSON(path, result); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "saved input hash mismatch", processed: true, want: "saved input hash does not match manifest", mutate: func(t *testing.T, root, batchID string) {
			manifest := readRetranslationManifest(t, root, batchID)
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, filepath.FromSlash(manifest.Units[0].InputPath))
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(appendBeforeRetranslationArtifactEOF(string(data), " changed")), 0644); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			catalog := glossaryStaleTestCatalog()
			writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
			initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}})
			if err != nil {
				t.Fatal(err)
			}
			if test.processed {
				processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
			}
			if test.mutate != nil {
				test.mutate(t, root, initial.BatchID)
			}
			writeGlossaryStaleTestGlossary(t, root, "current-glossary", false)
			_, err = ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{"lesson/1"}})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
		})
	}

	t.Run("stale source identity", func(t *testing.T) {
		root := t.TempDir()
		catalog := glossaryStaleTestCatalog()
		writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
		initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}})
		if err != nil {
			t.Fatal(err)
		}
		processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
		catalog.Pages[0].Source = []byte("* One\n\nIf the updated initializer is present, use it.\n")
		catalog.Pages[0].SourceSHA256 = sum(catalog.Pages[0].Source)
		writeGlossaryStaleTestGlossary(t, root, "current-glossary", false)
		_, err = ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{"lesson/1"}})
		if err == nil || !strings.Contains(err.Error(), "source identity does not match current Catalog") {
			t.Fatalf("stale source error=%v", err)
		}
	})
}

func TestGlossaryStaleRecoveryPreservesKindAndLimitGuards(t *testing.T) {
	t.Run("mixed Page and Example", func(t *testing.T) {
		root := t.TempDir()
		catalog := glossaryStaleTestCatalog()
		example := retranslationTestExample("example:lesson/present.go", "_content/tour/lesson/present.go", "package main\n\n// Check that present value first.\nfunc main() {}\n")
		catalog.Examples = []Example{example}
		writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
		page, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}})
		if err != nil {
			t.Fatal(err)
		}
		processGlossaryStaleTestBatch(t, root, catalog, page.BatchID)
		exportedExample, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitKind: UnitKindExample, UnitIDs: []string{example.ID}})
		if err != nil {
			t.Fatal(err)
		}
		processGlossaryStaleTestBatch(t, root, catalog, exportedExample.BatchID)
		writeGlossaryStaleTestGlossary(t, root, "current-glossary", false)
		_, err = ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
			Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{"lesson/1", example.ID},
		})
		if err == nil || !strings.Contains(err.Error(), "不能混合") {
			t.Fatalf("mixed recovery error=%v", err)
		}
	})

	t.Run("more than sixty", func(t *testing.T) {
		root := t.TempDir()
		writeRetranslationTestGlossary(t, root)
		catalog := retranslationTestCatalog(61)
		ids := make([]string, 0, 61)
		for _, page := range catalog.Pages {
			ids = append(ids, page.ID)
		}
		_, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: ids})
		if err == nil || !strings.Contains(err.Error(), "must not contain more than 60") {
			t.Fatalf("limit error=%v", err)
		}
	})

	t.Run("authorization modes are exclusive", func(t *testing.T) {
		root := t.TempDir()
		writeRetranslationTestGlossary(t, root)
		_, err := ExportRetranslationBatch(root, retranslationTestCatalog(1), RetranslationExportOptions{
			Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, PreviousSnapshotID: "qc-001", UnitIDs: []string{"lesson/1"},
		})
		if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
			t.Fatalf("mixed authorization error=%v", err)
		}
	})
}

func TestGlossaryStaleRecoveryCurrentInputMatchesNewManifest(t *testing.T) {
	root := t.TempDir()
	catalog := glossaryStaleTestCatalog()
	writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
	initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}})
	if err != nil {
		t.Fatal(err)
	}
	processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
	writeGlossaryStaleTestGlossary(t, root, "current-glossary", false)
	recovery, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{"lesson/1"}})
	if err != nil {
		t.Fatal(err)
	}
	manifest := readRetranslationManifest(t, root, recovery.BatchID)
	input, err := os.ReadFile(filepath.Join(root, recovery.BatchPath, filepath.FromSlash(manifest.Units[0].InputPath)))
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	unit, _ := catalog.Unit("lesson/1")
	protected, err := prepareTranslationUnitInput(unit, glossary)
	if err != nil {
		t.Fatal(err)
	}
	want := canonicalizeRetranslationArtifactEOF([]byte(protected.Text))
	if !bytes.Equal(input, want) || manifest.Units[0].InputSHA256 != sum(want) || manifest.Units[0].ProtectedTokenCount != len(protected.Tokens) {
		t.Fatalf("new recovery input does not match current glossary/protector")
	}
}

func TestGlossaryStaleRecoveryManifestRejectsRevisionAuthorizationMix(t *testing.T) {
	root := t.TempDir()
	catalog := glossaryStaleTestCatalog()
	writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
	initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}})
	if err != nil {
		t.Fatal(err)
	}
	processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
	writeGlossaryStaleTestGlossary(t, root, "current-glossary", false)
	recovery, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{"lesson/1"}})
	if err != nil {
		t.Fatal(err)
	}
	manifest := readRetranslationManifest(t, root, recovery.BatchID)
	manifest.Units[0].PreviousSnapshotID = "qc-001"
	manifest.Units[0].RevisionFeedbackSource = "quality_check"
	path := filepath.Join(root, recovery.BatchPath, "manifest.json")
	if err := writeTranslationJSON(path, manifest); err != nil {
		t.Fatal(err)
	}
	_, err = readRetranslationProcessManifest(filepath.Dir(path), "zh-CN", recovery.BatchID)
	if err == nil || !strings.Contains(err.Error(), "mixes glossary-stale provenance") {
		t.Fatalf("mixed manifest provenance error=%v", err)
	}
}
