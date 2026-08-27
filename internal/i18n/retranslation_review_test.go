package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func makeRetranslationReviewFixture(t *testing.T) (string, *Catalog, string, string, TranslationReview) {
	t.Helper()
	root, catalog, batchID := makeRetranslationProcessBatch(t, 1)
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	manifest := readRetranslationManifest(t, root, batchID)
	record := manifest.Units[0]
	unit, err := catalog.Unit(record.UnitID)
	if err != nil {
		t.Fatal(err)
	}
	resultData, err := os.ReadFile(filepath.Join(batchDir, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := decodeRetranslationProcessResult(resultData)
	if err != nil {
		t.Fatal(err)
	}
	unitResult := result.Units[0]
	candidate, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(unitResult.CandidatePath)))
	if err != nil {
		t.Fatal(err)
	}
	validation, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(unitResult.ValidationPath)))
	if err != nil {
		t.Fatal(err)
	}
	review := TranslationReview{
		SchemaVersion: TranslationReviewSchemaVersion, BatchID: batchID, Locale: "zh-CN",
		UnitID: record.UnitID, UnitKind: record.UnitKind, SourceSHA256: record.SourceSHA256, Attempt: 1,
		CandidatePath: unitResult.CandidatePath, CandidateSHA256: sum(candidate),
		ValidationPath: unitResult.ValidationPath, ValidationSHA256: sum(validation),
		Decision: "approved", Reviewer: "human@example.com", ReviewedAt: "2026-08-20T12:00:00Z",
		Rubric: TranslationQualityRubric, Rating: "A", Summary: "Accurate and fluent.", Issues: []string{},
	}
	return root, catalog, batchID, filepath.Join(batchDir, "review", retranslationReviewName(unit)), review
}

func writeRetranslationReview(t *testing.T, path string, review TranslationReview) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeTranslationJSON(path, review); err != nil {
		t.Fatal(err)
	}
}

func makeRetranslationReviewBatchFixture(t *testing.T, count int, snapshotID string) (string, *Catalog, string) {
	t.Helper()
	root, catalog, batchID := makeRetranslationProcessBatch(t, count)
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
	materializeSnapshotSources(t, root, catalog)
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshotID}); err != nil {
		t.Fatal(err)
	}
	return root, catalog, batchID
}

func reviewBatchOptions(snapshotID string) RetranslationReviewBatchRecordOptions {
	return RetranslationReviewBatchRecordOptions{
		Locale: "zh-CN", SnapshotID: snapshotID, Rating: "A", Decision: "approved",
		Summary: "Accurate and fluent.", Reviewer: "final-reviewer", Rubric: "translation-quality/v1",
		Now: func() time.Time { return time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC) },
	}
}

func reviewEvidencePath(t *testing.T, root string, catalog *Catalog, batchID, unitID string) string {
	t.Helper()
	unit, err := catalog.Unit(unitID)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "review", retranslationReviewName(unit))
}

func TestCheckRetranslationReviewsValidReview(t *testing.T) {
	root, catalog, batchID, path, review := makeRetranslationReviewFixture(t)
	writeRetranslationReview(t, path, review)
	report, err := CheckRetranslationReviews(root, catalog, RetranslationReviewCheckOptions{Locale: "zh-CN", BatchID: batchID})
	if err != nil {
		t.Fatal(err)
	}
	if report.UnitCount != 1 || report.Approved != 1 || report.Rejected != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestBuildRetranslationReviewScopeFirstLocaleIsFullyPending(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 3, "first-locale")
	scope, err := BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: "zh-CN", SnapshotID: "first-locale"})
	if err != nil || scope.UnitCount != 3 || scope.ReusableCount != 0 || scope.PendingCount != 3 {
		t.Fatalf("scope=%+v err=%v", scope, err)
	}
	for _, pending := range scope.Pending {
		if pending.Reason != ReviewScopeReasonMissingReview || pending.RequiredAction != ReviewScopeActionReviewRequired {
			t.Fatalf("first locale pending unit=%+v", pending)
		}
	}
}

func TestBuildRetranslationReviewScopeReusesOnlyMatchingApprovedEvidence(t *testing.T) {
	root, catalog, batchID := makeRetranslationReviewBatchFixture(t, 3, "all-reviewed")
	options := reviewBatchOptions("all-reviewed")
	options.Limit = 3
	if _, err := RecordRetranslationReviewBatch(root, catalog, options); err != nil {
		t.Fatal(err)
	}
	scope, err := BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: "zh-CN", SnapshotID: "all-reviewed"})
	if err != nil || scope.ReusableCount != 3 || scope.PendingCount != 0 {
		t.Fatalf("all-reviewed scope=%+v err=%v", scope, err)
	}

	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1", "lesson/2"})
	if err := os.RemoveAll(filepath.Join(root, "data", "retranslation-runs", "zh-CN", "chatgpt-zh-CN-002", "review")); err != nil {
		t.Fatal(err)
	}
	materializeSnapshotSources(t, root, catalog)
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "two-revised"}); err != nil {
		t.Fatal(err)
	}
	scope, err = BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: "zh-CN", SnapshotID: "two-revised"})
	if err != nil || scope.ReusableCount != 1 || scope.PendingCount != 2 || scope.Pending[0].UnitID != "lesson/1" || scope.Pending[1].UnitID != "lesson/2" {
		t.Fatalf("two-revised scope=%+v err=%v batch=%s", scope, err, batchID)
	}
}

func TestBuildRetranslationReviewScopeMarksInvalidFinalReviewPending(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*TranslationReview)
		reason RetranslationReviewScopeReason
		action RetranslationReviewRequiredAction
	}{
		{"rejected", func(review *TranslationReview) { review.Decision = "rejected" }, ReviewScopeReasonRejectedReview, ReviewScopeActionRevisionRequired},
		{"rating B", func(review *TranslationReview) { review.Rating = "B" }, ReviewScopeReasonNonAReview, ReviewScopeActionRevisionRequired},
		{"hash mismatch", func(review *TranslationReview) { review.CandidateSHA256 = strings.Repeat("0", 64) }, ReviewScopeReasonIdentityChanged, ReviewScopeActionReviewRequired},
		{"rubric mismatch", func(review *TranslationReview) { review.Rubric = "translation-quality/v2" }, ReviewScopeReasonRubricMismatch, ReviewScopeActionReviewRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, batchID := makeRetranslationReviewBatchFixture(t, 1, "invalid-"+strings.ReplaceAll(test.name, " ", "-"))
			if _, _, err := RecordRetranslationReview(root, catalog, RetranslationReviewRecordOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1", Rating: "A", Decision: "approved", Summary: "Approved.", Reviewer: "reviewer", Rubric: TranslationQualityRubric}); err != nil {
				t.Fatal(err)
			}
			path := reviewEvidencePath(t, root, catalog, batchID, "lesson/1")
			var review TranslationReview
			data, _ := os.ReadFile(path)
			if err := json.Unmarshal(data, &review); err != nil {
				t.Fatal(err)
			}
			test.mutate(&review)
			if err := writeTranslationJSON(path, review); err != nil {
				t.Fatal(err)
			}
			scope, err := BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: "zh-CN", SnapshotID: "invalid-" + strings.ReplaceAll(test.name, " ", "-")})
			if err != nil || scope.ReusableCount != 0 || scope.PendingCount != 1 {
				t.Fatalf("scope=%+v err=%v", scope, err)
			}
			if got := scope.Pending[0]; got.Reason != test.reason || got.RequiredAction != test.action {
				t.Fatalf("pending=%+v, want %s/%s", got, test.reason, test.action)
			}
		})
	}
}

func TestBuildRetranslationReviewScopeRejectsChangedGlossary(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 1, "glossary-guard")
	path := filepath.Join(root, "locales", "zh-CN", "glossary.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("# changed quality policy\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: "zh-CN", SnapshotID: "glossary-guard"})
	if err == nil || !strings.Contains(err.Error(), "glossary hash mismatch") {
		t.Fatalf("glossary guard error=%v", err)
	}
}

func TestSupersedeRetranslationReviewArchivesRubricExpiredEvidence(t *testing.T) {
	root, catalog, batchID := makeRetranslationReviewBatchFixture(t, 1, "rubric-renewal")
	if _, _, err := RecordRetranslationReview(root, catalog, RetranslationReviewRecordOptions{
		Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1", Rating: "A", Decision: "approved",
		Summary: "Prior review.", Reviewer: "reviewer", Rubric: TranslationQualityRubric,
	}); err != nil {
		t.Fatal(err)
	}
	currentPath := reviewEvidencePath(t, root, catalog, batchID, "lesson/1")
	var old TranslationReview
	data, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &old); err != nil {
		t.Fatal(err)
	}
	old.Rubric = "translation-quality/old"
	if err := writeTranslationJSON(currentPath, old); err != nil {
		t.Fatal(err)
	}
	oldData, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: "zh-CN", SnapshotID: "rubric-renewal"})
	if err != nil || scope.PendingCount != 1 || scope.Pending[0].Reason != ReviewScopeReasonRubricMismatch || scope.Pending[0].RequiredAction != ReviewScopeActionReviewRequired {
		t.Fatalf("pre-supersede scope=%+v err=%v", scope, err)
	}
	result, err := SupersedeRetranslationReview(root, catalog, RetranslationReviewSupersedeOptions{
		Locale: "zh-CN", SnapshotID: "rubric-renewal", UnitID: "lesson/1", Rating: "A", Decision: "approved",
		Summary: "Reviewed again using the current rubric.", Reviewer: "new-reviewer", Rubric: TranslationQualityRubric,
		Now: func() time.Time { return time.Date(2026, 8, 27, 1, 2, 3, 456000000, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	historyData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(result.HistoryPath)))
	if err != nil || !bytes.Equal(historyData, oldData) {
		t.Fatalf("history was not byte-preserved: err=%v", err)
	}
	newData, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	var current TranslationReview
	if err := json.Unmarshal(newData, &current); err != nil || current.Rubric != TranslationQualityRubric || current.Rating != "A" || current.Decision != "approved" {
		t.Fatalf("current review=%+v err=%v", current, err)
	}
	scope, err = BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: "zh-CN", SnapshotID: "rubric-renewal"})
	if err != nil || scope.ReusableCount != 1 || scope.PendingCount != 0 {
		t.Fatalf("post-supersede scope=%+v err=%v", scope, err)
	}
	// Review validation and promotion consume only the canonical review path;
	// deliberately malformed history must not enter the production gate.
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(result.HistoryPath)), []byte("not current evidence\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckRetranslationReviews(root, catalog, RetranslationReviewCheckOptions{Locale: "zh-CN", BatchID: batchID}); err != nil {
		t.Fatalf("canonical review should remain valid: %v", err)
	}
	writePromotionStatus(t, root, catalog, "old canonical\n")
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil || !plan.CanApply || plan.ReviewApprovedCount != 1 {
		t.Fatalf("promotion must read the renewed canonical review only: plan=%+v err=%v", plan, err)
	}
}

func TestSupersedeRetranslationReviewRejectsRevisionAndIdentityPending(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*TranslationReview)
	}{
		{"non-A review", func(review *TranslationReview) { review.Rating = "B" }},
		{"identity changed", func(review *TranslationReview) { review.CandidateSHA256 = strings.Repeat("0", 64) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, batchID := makeRetranslationReviewBatchFixture(t, 1, "supersede-"+strings.ReplaceAll(test.name, " ", "-"))
			if _, _, err := RecordRetranslationReview(root, catalog, RetranslationReviewRecordOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1", Rating: "A", Decision: "approved", Summary: "Prior review.", Reviewer: "reviewer", Rubric: TranslationQualityRubric}); err != nil {
				t.Fatal(err)
			}
			path := reviewEvidencePath(t, root, catalog, batchID, "lesson/1")
			var review TranslationReview
			data, _ := os.ReadFile(path)
			if err := json.Unmarshal(data, &review); err != nil {
				t.Fatal(err)
			}
			test.mutate(&review)
			if err := writeTranslationJSON(path, review); err != nil {
				t.Fatal(err)
			}
			_, err := SupersedeRetranslationReview(root, catalog, RetranslationReviewSupersedeOptions{Locale: "zh-CN", SnapshotID: "supersede-" + strings.ReplaceAll(test.name, " ", "-"), UnitID: "lesson/1", Rating: "A", Decision: "approved", Summary: "No bypass.", Reviewer: "reviewer", Rubric: TranslationQualityRubric})
			if err == nil || !strings.Contains(err.Error(), "only rubric_mismatch/review_required") {
				t.Fatalf("supersede error=%v", err)
			}
		})
	}
}

func TestCheckRetranslationReviewsMissingReview(t *testing.T) {
	root, catalog, batchID, _, _ := makeRetranslationReviewFixture(t)
	_, err := CheckRetranslationReviews(root, catalog, RetranslationReviewCheckOptions{Locale: "zh-CN", BatchID: batchID})
	if err == nil || !strings.Contains(err.Error(), "missing review") {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckRetranslationReviewsHashMismatches(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*TranslationReview)
		want   string
	}{
		{"candidate", func(review *TranslationReview) { review.CandidateSHA256 = strings.Repeat("0", 64) }, "candidate hash mismatch"},
		{"validation", func(review *TranslationReview) { review.ValidationSHA256 = strings.Repeat("0", 64) }, "validation hash mismatch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, batchID, path, review := makeRetranslationReviewFixture(t)
			test.mutate(&review)
			writeRetranslationReview(t, path, review)
			_, err := CheckRetranslationReviews(root, catalog, RetranslationReviewCheckOptions{Locale: "zh-CN", BatchID: batchID})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestCheckRetranslationReviewsInvalidDecisionAndRating(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*TranslationReview)
		want   string
	}{
		{"decision", func(review *TranslationReview) { review.Decision = "pending" }, "invalid decision"},
		{"rating", func(review *TranslationReview) { review.Rating = "E" }, "invalid rating"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, batchID, path, review := makeRetranslationReviewFixture(t)
			test.mutate(&review)
			writeRetranslationReview(t, path, review)
			_, err := CheckRetranslationReviews(root, catalog, RetranslationReviewCheckOptions{Locale: "zh-CN", BatchID: batchID})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestCheckRetranslationReviewsUnitMismatch(t *testing.T) {
	root, catalog, batchID, path, review := makeRetranslationReviewFixture(t)
	review.UnitID = "lesson/2"
	writeRetranslationReview(t, path, review)
	_, err := CheckRetranslationReviews(root, catalog, RetranslationReviewCheckOptions{Locale: "zh-CN", BatchID: batchID})
	if err == nil || !strings.Contains(err.Error(), "unit identity mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckRetranslationReviewsRejectedIsValidEvidence(t *testing.T) {
	root, catalog, batchID, path, review := makeRetranslationReviewFixture(t)
	review.Decision = "rejected"
	review.Rating = "D"
	review.Issues = []string{"Terminology is inconsistent."}
	writeRetranslationReview(t, path, review)
	report, err := CheckRetranslationReviews(root, catalog, RetranslationReviewCheckOptions{Locale: "zh-CN", BatchID: batchID})
	if err != nil || report.Rejected != 1 || report.Approved != 0 {
		t.Fatalf("report=%+v error=%v", report, err)
	}
}

func TestRecordRetranslationReviewGenericsEvidenceIsAccepted(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	source := []byte("* Page\n\nUse `Go` on this page.\n")
	catalog := &Catalog{Pages: []Page{{ID: "generics/1", Article: "generics.article", SectionNumber: 1, Route: "/generics/1", Source: source, SourceSHA256: sum(source)}}}
	exported, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"generics/1"}})
	if err != nil {
		t.Fatal(err)
	}
	batchDir := filepath.Join(root, exported.BatchPath)
	manifest := readRetranslationManifest(t, root, exported.BatchID)
	if err := os.Mkdir(filepath.Join(batchDir, "raw-responses"), 0755); err != nil {
		t.Fatal(err)
	}
	input, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(manifest.Units[0].InputPath)))
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.ReplaceAll(strings.ReplaceAll(string(input), "* Page", "* 页面"), "Use ", "在此页面使用 ")
	if err := os.WriteFile(filepath.Join(batchDir, "raw-responses", "generics-1.article"), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: exported.BatchID}); err != nil {
		t.Fatal(err)
	}
	review, path, err := RecordRetranslationReview(root, catalog, RetranslationReviewRecordOptions{
		Locale: "zh-CN", BatchID: exported.BatchID, UnitID: "generics/1", Rating: "A", Decision: "approved",
		Summary: "Accurate and fluent.", Reviewer: "test-reviewer", Rubric: "translation-quality/v1",
		Now: func() time.Time { return time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if review.SourcePath == "" || review.CandidateSHA256 == "" || review.ValidationSHA256 == "" || review.ReviewedAt != "2026-08-20T12:00:00Z" {
		t.Fatalf("generated review metadata = %+v", review)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
		t.Fatal(err)
	}
	report, err := CheckRetranslationReviews(root, catalog, RetranslationReviewCheckOptions{Locale: "zh-CN", BatchID: exported.BatchID})
	if err != nil || report.Approved != 1 || report.Rejected != 0 {
		t.Fatalf("review check report=%+v error=%v", report, err)
	}
}

func TestRecordRetranslationReviewBatchDefaultsToTwenty(t *testing.T) {
	root, catalog, batchID := makeRetranslationReviewBatchFixture(t, 23, "default-twenty")
	result, err := RecordRetranslationReviewBatch(root, catalog, reviewBatchOptions("default-twenty"))
	if err != nil {
		t.Fatal(err)
	}
	if result.StartIndex != 1 || result.EndIndex != 20 || result.Limit != 20 || result.RecordedCount != 20 || len(result.Reviews) != 20 {
		t.Fatalf("batch result=%+v", result)
	}
	for i := 1; i <= 20; i++ {
		if _, err := os.Stat(reviewEvidencePath(t, root, catalog, batchID, "lesson/"+strconv.Itoa(i))); err != nil {
			t.Fatalf("review %d: %v", i, err)
		}
	}
	if _, err := os.Stat(reviewEvidencePath(t, root, catalog, batchID, "lesson/21")); !os.IsNotExist(err) {
		t.Fatalf("default batch wrote index 21: %v", err)
	}
}

func TestRecordRetranslationReviewBatchUsesSnapshotSelectedBatches(t *testing.T) {
	root, catalog, firstBatch := makeRetranslationProcessBatch(t, 3)
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: firstBatch}); err != nil {
		t.Fatal(err)
	}
	const secondBatch = "chatgpt-zh-CN-002"
	addProcessedPromotionBatch(t, root, catalog, secondBatch, []string{"lesson/1"})
	if err := os.RemoveAll(filepath.Join(root, "data", "retranslation-runs", "zh-CN", secondBatch, "review")); err != nil {
		t.Fatal(err)
	}
	materializeSnapshotSources(t, root, catalog)
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "mixed-batches"}); err != nil {
		t.Fatal(err)
	}
	options := reviewBatchOptions("mixed-batches")
	options.Limit = 3
	result, err := RecordRetranslationReviewBatch(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{result.Reviews[0].BatchID, result.Reviews[1].BatchID, result.Reviews[2].BatchID}; !reflect.DeepEqual(got, []string{secondBatch, firstBatch, firstBatch}) {
		t.Fatalf("selected batches=%v", got)
	}
	for _, review := range result.Reviews {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(review.Path))); err != nil {
			t.Fatalf("review %s: %v", review.UnitID, err)
		}
	}
}

func TestRecordRetranslationReviewBatchRangeBoundaries(t *testing.T) {
	t.Run("final partial chunk", func(t *testing.T) {
		root, catalog, _ := makeRetranslationReviewBatchFixture(t, 23, "final-partial")
		options := reviewBatchOptions("final-partial")
		options.StartIndex = 22
		result, err := RecordRetranslationReviewBatch(root, catalog, options)
		if err != nil {
			t.Fatal(err)
		}
		if result.StartIndex != 22 || result.EndIndex != 23 || result.RecordedCount != 2 {
			t.Fatalf("final chunk=%+v", result)
		}
	})

	t.Run("start outside snapshot", func(t *testing.T) {
		root, catalog, batchID := makeRetranslationReviewBatchFixture(t, 3, "outside")
		options := reviewBatchOptions("outside")
		options.StartIndex = 4
		if _, err := RecordRetranslationReviewBatch(root, catalog, options); err == nil || !strings.Contains(err.Error(), "outside recordable review range") {
			t.Fatalf("range error=%v", err)
		}
		if _, err := os.Stat(reviewEvidencePath(t, root, catalog, batchID, "lesson/1")); !os.IsNotExist(err) {
			t.Fatalf("out-of-range batch wrote evidence: %v", err)
		}
	})
}

func TestRecordRetranslationReviewBatchDoesNotOverwriteExistingReview(t *testing.T) {
	root, catalog, batchID := makeRetranslationReviewBatchFixture(t, 3, "existing-review")
	if _, _, err := RecordRetranslationReview(root, catalog, RetranslationReviewRecordOptions{
		Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/2", Rating: "A", Decision: "approved",
		Summary: "Existing evidence.", Reviewer: "first-reviewer", Rubric: "translation-quality/v1",
		Now: func() time.Time { return time.Date(2026, 8, 24, 11, 0, 0, 0, time.UTC) },
	}); err != nil {
		t.Fatal(err)
	}
	existingPath := reviewEvidencePath(t, root, catalog, batchID, "lesson/2")
	existingBefore, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecordRetranslationReview(root, catalog, RetranslationReviewRecordOptions{
		Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/2", Rating: "A", Decision: "approved",
		Summary: "Must not replace existing evidence.", Reviewer: "second-reviewer", Rubric: TranslationQualityRubric,
	}); err == nil || !strings.Contains(err.Error(), "review already exists") {
		t.Fatalf("ordinary record overwrite error=%v", err)
	}
	options := reviewBatchOptions("existing-review")
	options.Limit = 3
	result, err := RecordRetranslationReviewBatch(root, catalog, options)
	if err != nil || result.RecordedCount != 2 || result.Reviews[0].UnitID != "lesson/1" || result.Reviews[1].UnitID != "lesson/3" {
		t.Fatalf("pending-only batch result=%+v error=%v", result, err)
	}
	if _, err := os.Stat(reviewEvidencePath(t, root, catalog, batchID, "lesson/1")); err != nil {
		t.Fatalf("pending lesson/1 was not reviewed: %v", err)
	}
	existingAfter, err := os.ReadFile(existingPath)
	if err != nil || !bytes.Equal(existingBefore, existingAfter) {
		t.Fatalf("existing review changed: err=%v", err)
	}
}

func TestRecordRetranslationReviewBatchPreflightPreventsPartialWritesOnIdentityAndHashMismatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string, string)
		want   string
	}{
		{
			name: "validation identity mismatch",
			mutate: func(t *testing.T, root, batchID string) {
				path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "validation", "lesson-2.json")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var validation RetranslationValidation
				if err := json.Unmarshal(data, &validation); err != nil {
					t.Fatal(err)
				}
				validation.UnitID = "lesson/1"
				if err := writeTranslationJSON(path, validation); err != nil {
					t.Fatal(err)
				}
			},
			want: "incompatible identity",
		},
		{
			name: "candidate hash mismatch",
			mutate: func(t *testing.T, root, batchID string) {
				path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "candidates", "lesson-2.article")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
					t.Fatal(err)
				}
			},
			want: "candidate path or hash",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, batchID := makeRetranslationReviewBatchFixture(t, 3, "mismatch")
			test.mutate(t, root, batchID)
			options := reviewBatchOptions("mismatch")
			options.Limit = 3
			if _, err := RecordRetranslationReviewBatch(root, catalog, options); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("mismatch error=%v, want %q", err, test.want)
			}
			for _, unitID := range []string{"lesson/1", "lesson/2", "lesson/3"} {
				if _, err := os.Stat(reviewEvidencePath(t, root, catalog, batchID, unitID)); !os.IsNotExist(err) {
					t.Fatalf("failed preflight wrote %s review: %v", unitID, err)
				}
			}
		})
	}
}
