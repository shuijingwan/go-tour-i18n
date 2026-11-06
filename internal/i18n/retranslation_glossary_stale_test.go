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
		copyBundleAuthority(t, root, []string{"locales/zh-CN/locale.json"})
		recordSyntheticUnifiedReview(t, root, reviewID+"-unified")
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

type glossaryStaleLegacyManifestFixture struct {
	SchemaVersion   int                              `json:"schema_version"`
	BatchID         string                           `json:"batch_id"`
	Locale          string                           `json:"locale"`
	ProtectionMode  string                           `json:"protection_mode"`
	TranslationUnit string                           `json:"translation_unit"`
	PageCount       int                              `json:"page_count"`
	Pages           []glossaryStaleLegacyPageFixture `json:"pages"`
}

type glossaryStaleLegacyPageFixture struct {
	PageID              string `json:"page_id"`
	Article             string `json:"article"`
	SourceSHA256        string `json:"source_sha256"`
	InputPath           string `json:"input_path"`
	InputSHA256         string `json:"input_sha256"`
	ProtectedTokenCount int    `json:"protected_token_count"`
}

func convertGlossaryStaleBatchToLegacyEOF(t *testing.T, root, batchID string) (*glossaryStaleLegacyManifestFixture, map[string][]byte) {
	t.Helper()
	manifest := readRetranslationManifest(t, root, batchID)
	legacy := &glossaryStaleLegacyManifestFixture{
		SchemaVersion: 1, BatchID: batchID, Locale: "zh-CN", ProtectionMode: "default",
		TranslationUnit: "present.Section", PageCount: len(manifest.Units),
		Pages: make([]glossaryStaleLegacyPageFixture, 0, len(manifest.Units)),
	}
	original := map[string][]byte{}
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	for _, unit := range manifest.Units {
		path := filepath.Join(batchDir, filepath.FromSlash(unit.InputPath))
		input, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		input = append(append([]byte(nil), input...), '\n')
		if err := os.WriteFile(path, input, 0644); err != nil {
			t.Fatal(err)
		}
		original[unit.UnitID] = append([]byte(nil), input...)
		legacy.Pages = append(legacy.Pages, glossaryStaleLegacyPageFixture{
			PageID: unit.UnitID, Article: filepath.Base(unit.SourcePath), SourceSHA256: unit.SourceSHA256,
			InputPath: unit.InputPath, InputSHA256: sum(input), ProtectedTokenCount: unit.ProtectedTokenCount,
		})
	}
	if err := writeTranslationJSON(filepath.Join(batchDir, "manifest.json"), legacy); err != nil {
		t.Fatal(err)
	}
	return legacy, original
}

func writeGlossaryStaleEquivalentTestGlossary(t *testing.T, root, reviewID string) {
	t.Helper()
	path := filepath.Join(root, "locales", "zh-CN", "glossary.yaml")
	text := "# byte-only refresh\nmandatory:\n  Go: Go\nkeep:\n  - Go\n  - present\n"
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecordGlossaryReview(root, "zh-CN", reviewID, "independent-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	copyBundleAuthority(t, root, []string{"locales/zh-CN/locale.json"})
	recordSyntheticUnifiedReview(t, root, reviewID+"-unified")
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
	if _, err := ExportRetranslationBatch(root, catalog, options); err == nil || !strings.Contains(err.Error(), "unified Glossary Review missing/stale/ambiguous or non-passed") {
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

func TestGlossaryStaleRecoveryLegacyArtifactEOF(t *testing.T) {
	t.Run("EOF-only difference is not glossary drift", func(t *testing.T) {
		root := t.TempDir()
		catalog := glossaryStaleTestCatalog()
		writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
		initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}})
		if err != nil {
			t.Fatal(err)
		}
		processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
		convertGlossaryStaleBatchToLegacyEOF(t, root, initial.BatchID)
		writeGlossaryStaleEquivalentTestGlossary(t, root, "current-glossary")
		_, err = ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
			Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{"lesson/1"},
		})
		if err == nil || !strings.Contains(err.Error(), "has no glossary-induced drift") {
			t.Fatalf("EOF-only legacy recovery error=%v", err)
		}
	})

	t.Run("real drift preserves historical provenance and writes single LF", func(t *testing.T) {
		root := t.TempDir()
		catalog := glossaryStaleTestCatalog()
		writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
		initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}})
		if err != nil {
			t.Fatal(err)
		}
		processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
		_, historical := convertGlossaryStaleBatchToLegacyEOF(t, root, initial.BatchID)
		writeGlossaryStaleTestGlossary(t, root, "current-glossary", false)
		recovery, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
			Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{"lesson/1"},
		})
		if err != nil {
			t.Fatal(err)
		}
		manifest := readRetranslationManifest(t, root, recovery.BatchID)
		if manifest.SchemaVersion != 2 || manifest.ArtifactEOF != retranslationArtifactEOFSingleLF || len(manifest.Units) != 1 {
			t.Fatalf("recovery manifest=%+v", manifest)
		}
		record := manifest.Units[0]
		oldInput := historical[record.UnitID]
		if record.PreviousBatchID != initial.BatchID || record.PreviousInputSHA256 != sum(oldInput) || record.PreviousInputSHA256 == sum(canonicalizeRetranslationArtifactEOF(oldInput)) {
			t.Fatalf("legacy previous provenance=%+v", record)
		}
		newInput, err := os.ReadFile(filepath.Join(root, recovery.BatchPath, filepath.FromSlash(record.InputPath)))
		if err != nil {
			t.Fatal(err)
		}
		if err := validateRetranslationArtifactEOF(newInput); err != nil {
			t.Fatalf("new recovery input EOF: %v", err)
		}
		oldPath := filepath.Join(root, initial.BatchPath, filepath.FromSlash(record.PreviousInputPath))
		unchanged, err := os.ReadFile(oldPath)
		if err != nil || !bytes.Equal(unchanged, oldInput) {
			t.Fatalf("historical input mutated: err=%v", err)
		}
	})

	t.Run("unsupported EOF policy remains rejected", func(t *testing.T) {
		root := t.TempDir()
		catalog := glossaryStaleTestCatalog()
		writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
		initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}})
		if err != nil {
			t.Fatal(err)
		}
		processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
		manifest := readRetranslationManifest(t, root, initial.BatchID)
		manifest.ArtifactEOF = "unknown_policy"
		if err := writeTranslationJSON(filepath.Join(root, initial.BatchPath, "manifest.json"), manifest); err != nil {
			t.Fatal(err)
		}
		writeGlossaryStaleTestGlossary(t, root, "current-glossary", false)
		_, err = ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
			Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{"lesson/1"},
		})
		if err == nil || !strings.Contains(err.Error(), "incompatible manifest metadata") {
			t.Fatalf("unsupported EOF error=%v", err)
		}
	})

	integrityTests := []struct {
		name   string
		want   string
		mutate func(t *testing.T, root, batchID string, legacy *glossaryStaleLegacyManifestFixture)
	}{
		{name: "SHA mismatch", want: "saved input hash does not match manifest", mutate: func(t *testing.T, root, batchID string, legacy *glossaryStaleLegacyManifestFixture) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, filepath.FromSlash(legacy.Pages[0].InputPath))
			input, _ := os.ReadFile(path)
			if err := os.WriteFile(path, append([]byte("changed "), input...), 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "token count mismatch", want: "protected token count", mutate: func(t *testing.T, root, batchID string, legacy *glossaryStaleLegacyManifestFixture) {
			legacy.Pages[0].ProtectedTokenCount++
		}},
		{name: "duplicate token", want: "contains duplicate protected token", mutate: func(t *testing.T, root, batchID string, legacy *glossaryStaleLegacyManifestFixture) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, filepath.FromSlash(legacy.Pages[0].InputPath))
			input, _ := os.ReadFile(path)
			token := translationTokenRE.Find(input)
			if len(token) == 0 {
				t.Fatal("test input has no protected token")
			}
			input = append(append(append([]byte(nil), token...), ' '), input...)
			if err := os.WriteFile(path, input, 0644); err != nil {
				t.Fatal(err)
			}
			legacy.Pages[0].InputSHA256 = sum(input)
			legacy.Pages[0].ProtectedTokenCount = len(translationTokenRE.FindAll(input, -1))
		}},
	}
	for _, test := range integrityTests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			catalog := glossaryStaleTestCatalog()
			writeGlossaryStaleTestGlossary(t, root, "old-glossary", true)
			initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1"}})
			if err != nil {
				t.Fatal(err)
			}
			processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
			legacy, _ := convertGlossaryStaleBatchToLegacyEOF(t, root, initial.BatchID)
			test.mutate(t, root, initial.BatchID, legacy)
			if err := writeTranslationJSON(filepath.Join(root, initial.BatchPath, "manifest.json"), legacy); err != nil {
				t.Fatal(err)
			}
			writeGlossaryStaleTestGlossary(t, root, "current-glossary", false)
			_, err = ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
				Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{"lesson/1"},
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("legacy integrity error=%v, want %q", err, test.want)
			}
		})
	}
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
	if manifest.ArtifactEOF != retranslationArtifactEOFSingleLF {
		t.Fatalf("new recovery artifact_eof=%q", manifest.ArtifactEOF)
	}
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
