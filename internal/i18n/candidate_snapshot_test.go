package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func materializeSnapshotSources(t *testing.T, root string, catalog *Catalog) {
	t.Helper()
	articles := map[string][]byte{}
	for _, page := range catalog.Pages {
		path := filepath.ToSlash(filepath.Join("_content", "tour", page.Article))
		articles[path] = append(articles[path], page.Source...)
	}
	for path, data := range articles {
		writeSnapshotSource(t, root, path, data)
	}
	for _, example := range catalog.Examples {
		if example.EligibleTranslation || !example.EligibilityKnown {
			writeSnapshotSource(t, root, example.SourcePath, example.Source)
		}
	}
}

func writeSnapshotSource(t *testing.T, root, path string, data []byte) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func snapshotUnit(t *testing.T, manifest *QualityCheckSnapshotManifest, id string) QualityCheckSnapshotUnit {
	t.Helper()
	for _, unit := range manifest.Units {
		if unit.UnitID == id {
			return unit
		}
	}
	t.Fatalf("snapshot unit %q not found", id)
	return QualityCheckSnapshotUnit{}
}

func addExportOnlySnapshotBatch(t *testing.T, root string, catalog *Catalog, batchID string, ids []string) {
	t.Helper()
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{
		Locale: "zh-CN", BatchID: batchID, UnitIDs: ids, Limit: len(ids), AllowReexport: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	resultPath := filepath.Join(root, result.BatchPath, "result.json")
	if _, err := os.Stat(resultPath); !os.IsNotExist(err) {
		t.Fatalf("export-only batch result.json stat error = %v, want not exist", err)
	}
}

func TestQualityCheckSnapshotFreezesCompleteWorkflowWithoutCopyingEvidence(t *testing.T) {
	root, catalog := complete122PromotionFixture(t)
	// Deliberately make Catalog order differ from lexical/batch order. Snapshot
	// indices must remain Pages-first and follow the two Catalog inventories.
	catalog.Pages[0], catalog.Pages[len(catalog.Pages)-1] = catalog.Pages[len(catalog.Pages)-1], catalog.Pages[0]
	catalog.Examples[0], catalog.Examples[len(catalog.Examples)-1] = catalog.Examples[len(catalog.Examples)-1], catalog.Examples[0]
	materializeSnapshotSources(t, root, catalog)
	statusPath := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	statusBefore, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}

	manifest, path, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-2026-08-24"})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.UnitCount != 122 || manifest.PageCount != 103 || manifest.ExampleCount != 19 || len(manifest.Units) != 122 {
		t.Fatalf("snapshot counts = %+v", manifest)
	}
	if path != "data/quality-check-snapshots/zh-CN/qc-2026-08-24/manifest.json" {
		t.Fatalf("manifest path = %q", path)
	}
	for i, unit := range manifest.Units {
		if unit.Index != i+1 {
			t.Fatalf("unit %s index=%d, want %d", unit.UnitID, unit.Index, i+1)
		}
		var wantID string
		if i < len(catalog.Pages) {
			wantID = catalog.Pages[i].ID
			if unit.UnitKind != UnitKindPage || unit.PageSection == nil || unit.PageSection.Article != catalog.Pages[i].Article || unit.PageSection.SectionNumber != catalog.Pages[i].SectionNumber {
				t.Fatalf("Page snapshot identity[%d] = %+v", i, unit)
			}
		} else {
			wantID = catalog.Examples[i-len(catalog.Pages)].ID
			if unit.UnitKind != UnitKindExample || unit.PageSection != nil {
				t.Fatalf("Example snapshot identity[%d] = %+v", i, unit)
			}
		}
		if unit.UnitID != wantID {
			t.Fatalf("stable order[%d]=%q, want %q", i, unit.UnitID, wantID)
		}
		for _, referenced := range []string{unit.SourcePath, unit.CandidatePath, unit.ValidationPath} {
			if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(referenced))); err != nil || !info.Mode().IsRegular() {
				t.Fatalf("snapshot reference %q is not an existing regular file: %v", referenced, err)
			}
		}
	}
	entries, err := os.ReadDir(filepath.Dir(filepath.Join(root, filepath.FromSlash(path))))
	if err != nil || len(entries) != 1 || entries[0].Name() != "manifest.json" {
		t.Fatalf("snapshot persisted more than manifest.json: entries=%v err=%v", entries, err)
	}
	statusAfter, err := os.ReadFile(statusPath)
	if err != nil || !bytes.Equal(statusBefore, statusAfter) {
		t.Fatalf("snapshot changed status.tsv: err=%v", err)
	}
}

func TestQualityCheckSnapshotLatestRevisionBatchWinsPerUnit(t *testing.T) {
	root, catalog, firstBatch := processedPromotionFixture(t, 2)
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-007", []string{"lesson/1"})
	addProcessedPromotionBatch(t, root, catalog, "codex-zh-CN-008", []string{"lesson/1"})
	materializeSnapshotSources(t, root, catalog)
	manifest, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "revision"})
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshotUnit(t, manifest, "lesson/1").SelectedBatchID; got != "codex-zh-CN-008" {
		t.Fatalf("latest revision batch = %q", got)
	}
	if got := snapshotUnit(t, manifest, "lesson/2").SelectedBatchID; got != firstBatch {
		t.Fatalf("unrevised unit batch = %q, want %q", got, firstBatch)
	}
}

func TestQualityCheckSnapshotSkipsIntermediateExportOnlyBatch(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	addExportOnlySnapshotBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1"})
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-003", []string{"lesson/1"})
	materializeSnapshotSources(t, root, catalog)

	manifest, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "export-only-history"})
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshotUnit(t, manifest, "lesson/1").SelectedBatchID; got != "chatgpt-zh-CN-003" {
		t.Fatalf("selected batch = %q, want later processed batch", got)
	}
}

func TestSelectLatestRetranslationUnitsDoesNotSelectExportOnlyBatch(t *testing.T) {
	root, catalog, processedBatch := processedPromotionFixture(t, 1)
	addExportOnlySnapshotBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1"})

	latest, err := selectLatestRetranslationUnits(root, catalog, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	choice, ok := latest.selectedByID["lesson/1"]
	if !ok {
		t.Fatal("processed result was not selected")
	}
	if choice.batchID != processedBatch {
		t.Fatalf("selected batch = %q, want processed batch %q", choice.batchID, processedBatch)
	}
}

func TestQualityCheckSnapshotExportOnlyEvidenceIsStillMissingCandidate(t *testing.T) {
	root, catalog, batchID := makeRetranslationProcessBatch(t, 1)
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	if err := os.RemoveAll(filepath.Join(batchDir, "raw-responses")); err != nil {
		t.Fatal(err)
	}
	materializeSnapshotSources(t, root, catalog)

	_, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "export-only-missing"})
	if err == nil || !strings.Contains(err.Error(), "lesson/1: no current-source retranslation candidate") {
		t.Fatalf("export-only snapshot error = %v", err)
	}
}

func TestQualityCheckSnapshotRejectsCorruptExistingResult(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	addExportOnlySnapshotBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1"})
	resultPath := filepath.Join(root, "data", "retranslation-runs", "zh-CN", "chatgpt-zh-CN-002", "result.json")
	if err := os.WriteFile(resultPath, []byte("{not-json\n"), 0644); err != nil {
		t.Fatal(err)
	}
	materializeSnapshotSources(t, root, catalog)

	_, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "corrupt-result"})
	if err == nil || !strings.Contains(err.Error(), `parse retranslation result for "chatgpt-zh-CN-002"`) {
		t.Fatalf("corrupt result error = %v", err)
	}
}

func TestQualityCheckSnapshotRejectsCrossPrefixDuplicateBatchNumber(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	addExportOnlySnapshotBatch(t, root, catalog, "chatgpt-zh-CN-001", []string{"lesson/1"})
	materializeSnapshotSources(t, root, catalog)

	_, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "ambiguous"})
	if err == nil || !strings.Contains(err.Error(), "ambiguous or invalid retranslation batch number 001") {
		t.Fatalf("cross-prefix duplicate error=%v", err)
	}
}

func TestQualityCheckSnapshotUsesFinalRetryAttempt(t *testing.T) {
	root, catalog, batchID := processRetryFixture(t, 1, func(raw string) string {
		return appendBeforeRetranslationArtifactEOF(raw, " `bad`")
	})
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	validRaw, err := os.ReadFile(filepath.Join(batchDir, "inputs", "lesson-1.article"))
	if err != nil {
		t.Fatal(err)
	}
	writeRetryRaw(t, root, batchID, "lesson/1", 2, string(validRaw))
	if _, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"}); err != nil {
		t.Fatal(err)
	}
	materializeSnapshotSources(t, root, catalog)
	manifest, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "retry-final"})
	if err != nil {
		t.Fatal(err)
	}
	unit := snapshotUnit(t, manifest, "lesson/1")
	if unit.Attempt != 2 {
		t.Fatalf("snapshot attempt=%d, want 2", unit.Attempt)
	}
	validationData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(unit.ValidationPath)))
	if err != nil || !strings.Contains(string(validationData), `"raw_response_path": "retries/lesson-1/attempt-002.article"`) {
		t.Fatalf("snapshot validation does not point to final retry: %s err=%v", validationData, err)
	}
}

func TestQualityCheckSnapshotUsesRevalidatedEvidenceWithoutNewAttempt(t *testing.T) {
	root, catalog, batchID, _, _, _ := makeRevalidationFixture(t)
	result, err := RevalidateRetranslationCandidate(root, catalog, RetranslationRevalidateOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"})
	if err != nil {
		t.Fatal(err)
	}
	materializeSnapshotSources(t, root, catalog)
	manifest, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "revalidated"})
	if err != nil {
		t.Fatal(err)
	}
	unit := snapshotUnit(t, manifest, "lesson/1")
	if unit.Attempt != 1 || result.Attempt != 1 {
		t.Fatalf("snapshot attempt=%d revalidation attempt=%d, want 1", unit.Attempt, result.Attempt)
	}
	validation, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(unit.ValidationPath)))
	if err != nil {
		t.Fatal(err)
	}
	if unit.ValidationSHA256 != sum(validation) {
		t.Fatal("snapshot did not bind the final revalidated evidence")
	}
}

func TestQualityCheckSnapshotLatestFailureNeverFallsBack(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1"})
	materializeSnapshotSources(t, root, catalog)
	resultPath := filepath.Join(root, "data", "retranslation-runs", "zh-CN", "chatgpt-zh-CN-002", "result.json")
	var result RetranslationProcessResult
	data, _ := os.ReadFile(resultPath)
	_ = json.Unmarshal(data, &result)
	result.Units[0].Status = "validation_failed"
	if err := writeTranslationJSON(resultPath, result); err != nil {
		t.Fatal(err)
	}
	_, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "must-fail"})
	if err == nil || !strings.Contains(err.Error(), "chatgpt-zh-CN-002") || !strings.Contains(err.Error(), "refusing fallback") {
		t.Fatalf("latest failure error=%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "data", "quality-check-snapshots", "zh-CN", "must-fail")); !os.IsNotExist(statErr) {
		t.Fatalf("failed snapshot left artifact: %v", statErr)
	}
}

func TestQualityCheckSnapshotLatestIdentityMismatchNeverFallsBack(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1"})
	materializeSnapshotSources(t, root, catalog)
	manifestPath := filepath.Join(root, "data", "retranslation-runs", "zh-CN", "chatgpt-zh-CN-002", "manifest.json")
	batchManifest := readRetranslationManifestAt(t, manifestPath)
	batchManifest.Units[0].SourceSHA256 = strings.Repeat("0", 64)
	if err := writeTranslationJSON(manifestPath, batchManifest); err != nil {
		t.Fatal(err)
	}
	_, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "identity-mismatch"})
	if err == nil || !strings.Contains(err.Error(), "chatgpt-zh-CN-002") || !strings.Contains(err.Error(), "source identity") || !strings.Contains(err.Error(), "refusing fallback") {
		t.Fatalf("latest identity mismatch error=%v", err)
	}
}

func TestQualityCheckSnapshotRejectsHashAndIdentityMismatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string, string)
		want   string
	}{
		{"source bytes", func(t *testing.T, root, batchID string) {
			path := filepath.Join(root, "_content", "tour", "lesson.article")
			if err := os.WriteFile(path, []byte("* Different\n\nWrong source.\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}, "source_path section does not match current Catalog source identity"},
		{"input hash", func(t *testing.T, root, batchID string) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "manifest.json")
			manifest := readRetranslationManifestAt(t, path)
			manifest.Units[0].InputSHA256 = strings.Repeat("0", 64)
			if err := writeTranslationJSON(path, manifest); err != nil {
				t.Fatal(err)
			}
		}, "input_sha256 mismatch"},
		{"validation source identity", func(t *testing.T, root, batchID string) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "validation", "lesson-1.json")
			var validation RetranslationValidation
			data, _ := os.ReadFile(path)
			_ = json.Unmarshal(data, &validation)
			validation.SourceSHA256 = strings.Repeat("0", 64)
			if err := writeTranslationJSON(path, validation); err != nil {
				t.Fatal(err)
			}
		}, "validation does not match manifest/result.json"},
		{"candidate restore binding", func(t *testing.T, root, batchID string) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "candidates", "lesson-1.article")
			candidate, _ := os.ReadFile(path)
			candidate = bytes.Replace(candidate, []byte("页面"), []byte("网页"), 1)
			if err := os.WriteFile(path, candidate, 0644); err != nil {
				t.Fatal(err)
			}
		}, "restored candidate does not match saved candidate"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, batchID := processedPromotionFixture(t, 1)
			materializeSnapshotSources(t, root, catalog)
			test.mutate(t, root, batchID)
			_, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "invalid"})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
		})
	}
}

func TestQualityCheckSnapshotManifestRoundTripAndCollision(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	materializeSnapshotSources(t, root, catalog)
	manifest, path, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "round-trip"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	var decoded QualityCheckSnapshotManifest
	if err := json.Unmarshal(data, &decoded); err != nil || !reflect.DeepEqual(*manifest, decoded) {
		t.Fatalf("manifest round trip=%+v err=%v", decoded, err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "round-trip"}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("snapshot collision error=%v", err)
	}
}
