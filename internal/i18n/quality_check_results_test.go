package i18n

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func recordQualityCheckRatings(t *testing.T, root string, catalog *Catalog, snapshotID, previousSnapshotID, rating string, unitIDs []string) {
	t.Helper()
	if _, err := RecordQualityCheckResults(root, catalog, QualityCheckRecordOptions{
		Locale: "zh-CN", SnapshotID: snapshotID, PreviousSnapshotID: previousSnapshotID,
		Rating: rating, UnitIDs: unitIDs,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestQualityCheckScopeFreshLocaleIsFullyPending(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 3, "qc-001")
	scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-001"})
	if err != nil || scope.UnitCount != 3 || scope.CurrentResultCount != 0 || scope.CarryForwardCount != 0 || scope.PendingCount != 3 || scope.ReadyForFinalReview {
		t.Fatalf("fresh scope=%+v err=%v", scope, err)
	}
	for _, pending := range scope.Pending {
		if pending.Reason != QualityCheckScopeReasonMissing || pending.RequiredAction != QualityCheckActionRequired {
			t.Fatalf("fresh pending=%+v", pending)
		}
	}
}

func TestQualityCheckScopeCarriesOnlyIdentityMatchingA(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 4, "qc-001")
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "A", []string{"lesson/1"})
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "B", []string{"lesson/2"})
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "C", []string{"lesson/3"})
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "D", []string{"lesson/4"})
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
	if err != nil || scope.CarryForwardCount != 1 || scope.PendingCount != 3 || scope.ACount != 1 || scope.BCount != 1 || scope.CCount != 1 || scope.DCount != 1 {
		t.Fatalf("incremental scope=%+v err=%v", scope, err)
	}
	if scope.CarryForward[0].UnitID != "lesson/1" || scope.CarryForward[0].FromSnapshotID != "qc-001" {
		t.Fatalf("carry-forward=%+v", scope.CarryForward)
	}
	for _, pending := range scope.Pending {
		if pending.Reason != QualityCheckScopeReasonNonA || pending.RequiredAction != QualityCheckActionRevisionRequired {
			t.Fatalf("non-A pending=%+v", pending)
		}
	}
}

func TestQualityCheckSnapshotIdentityIncludesAllCarryForwardFields(t *testing.T) {
	base := QualityCheckSnapshotUnit{
		UnitID: "lesson/1", UnitKind: UnitKindPage, SelectedBatchID: "chatgpt-zh-CN-001",
		SourcePath: "_content/tour/lesson.article", SourceSHA256: strings.Repeat("1", 64),
		CandidatePath: "data/retranslation-runs/zh-CN/chatgpt-zh-CN-001/candidates/lesson-1.article", CandidateSHA256: strings.Repeat("2", 64),
		ValidationPath: "data/retranslation-runs/zh-CN/chatgpt-zh-CN-001/validation/lesson-1.json", ValidationSHA256: strings.Repeat("3", 64),
		Attempt: 1,
	}
	tests := []struct {
		name   string
		mutate func(*QualityCheckSnapshotUnit)
	}{
		{"source", func(unit *QualityCheckSnapshotUnit) { unit.SourceSHA256 = strings.Repeat("4", 64) }},
		{"candidate", func(unit *QualityCheckSnapshotUnit) { unit.CandidateSHA256 = strings.Repeat("4", 64) }},
		{"validation", func(unit *QualityCheckSnapshotUnit) { unit.ValidationSHA256 = strings.Repeat("4", 64) }},
		{"attempt", func(unit *QualityCheckSnapshotUnit) { unit.Attempt = 2 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := base
			test.mutate(&changed)
			if qualityCheckSnapshotIdentityMatches(changed, base) {
				t.Fatalf("%s identity change was reusable", test.name)
			}
		})
	}
}

func TestQualityCheckScopeRequiresNewQCWhenAnyIdentityFieldChanges(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*QualityCheckSnapshotUnit)
	}{
		{"source", func(unit *QualityCheckSnapshotUnit) { unit.SourceSHA256 = strings.Repeat("4", 64) }},
		{"candidate", func(unit *QualityCheckSnapshotUnit) { unit.CandidateSHA256 = strings.Repeat("4", 64) }},
		{"validation", func(unit *QualityCheckSnapshotUnit) { unit.ValidationSHA256 = strings.Repeat("4", 64) }},
		{"attempt", func(unit *QualityCheckSnapshotUnit) { unit.Attempt++ }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, _ := makeRetranslationReviewBatchFixture(t, 1, "qc-001")
			recordQualityCheckRatings(t, root, catalog, "qc-001", "", "A", []string{"lesson/1"})
			if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
				t.Fatal(err)
			}
			manifest, err := readQualityCheckSnapshot(root, "zh-CN", "qc-002", false)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(&manifest.Units[0])
			if err := writeTranslationJSON(qualityCheckSnapshotManifestPath(root, "zh-CN", "qc-002"), manifest); err != nil {
				t.Fatal(err)
			}
			scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
			if err != nil || scope.CarryForwardCount != 0 || scope.ACount != 0 || scope.PendingCount != 1 {
				t.Fatalf("%s scope=%+v err=%v", test.name, scope, err)
			}
			if pending := scope.Pending[0]; pending.Reason != QualityCheckScopeReasonIdentityChanged || pending.RequiredAction != QualityCheckActionRequired {
				t.Fatalf("%s pending=%+v", test.name, pending)
			}
		})
	}
}

func TestQualityCheckScopeDoesNotCarryAcrossGlossaryChange(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 1, "qc-001")
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "A", []string{"lesson/1"})
	glossaryPath := filepath.Join(root, "locales", "zh-CN", "glossary.yaml")
	data, err := os.ReadFile(glossaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(glossaryPath, append(data, []byte("# quality policy changed\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
	if err != nil || scope.CarryForwardCount != 0 || scope.PendingCount != 1 {
		t.Fatalf("glossary scope=%+v err=%v", scope, err)
	}
	if pending := scope.Pending[0]; pending.Reason != QualityCheckScopeReasonGlossaryChanged || pending.RequiredAction != QualityCheckActionRequired {
		t.Fatalf("glossary pending=%+v", pending)
	}
}

func TestQualityCheckScopeDoesNotCarryAcrossRubricChange(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 1, "qc-001")
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "A", []string{"lesson/1"})
	snapshot, err := readQualityCheckSnapshot(root, "zh-CN", "qc-001", false)
	if err != nil {
		t.Fatal(err)
	}
	results, err := readQualityCheckResults(root, "zh-CN", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	results.Rubric = "translation-quality/old"
	if err := writeQualityCheckResults(qualityCheckResultsPath(root, "zh-CN", "qc-001"), results); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
	if err != nil || scope.CarryForwardCount != 0 || scope.PendingCount != 1 {
		t.Fatalf("rubric scope=%+v err=%v", scope, err)
	}
	if pending := scope.Pending[0]; pending.Reason != QualityCheckScopeReasonRubricChanged || pending.RequiredAction != QualityCheckActionRequired {
		t.Fatalf("rubric pending=%+v", pending)
	}
}

func TestQualityCheckScopeRevisionCarries119AndRequiresExactlyThree(t *testing.T) {
	root, catalog, firstBatch := makeRetranslationReviewBatchFixture(t, 122, "qc-001")
	revised := []string{"lesson/17", "lesson/37", "lesson/94"}
	revisedSet := map[string]bool{"lesson/17": true, "lesson/37": true, "lesson/94": true}
	approved := make([]string, 0, 119)
	for _, page := range catalog.Pages {
		if !revisedSet[page.ID] {
			approved = append(approved, page.ID)
		}
	}
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "A", approved)
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "B", revised)

	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-006", revised)
	if err := os.RemoveAll(filepath.Join(root, "data", "retranslation-runs", "zh-CN", "chatgpt-zh-CN-006", "review")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
	if err != nil || scope.CarryForwardCount != 119 || scope.PendingCount != 3 || scope.ACount != 119 || scope.BCount != 0 || scope.ReadyForFinalReview {
		t.Fatalf("119+3 scope=%+v err=%v first_batch=%s", scope, err, firstBatch)
	}
	gotPending := []string{scope.Pending[0].UnitID, scope.Pending[1].UnitID, scope.Pending[2].UnitID}
	if !reflect.DeepEqual(gotPending, revised) {
		t.Fatalf("pending=%v, want %v", gotPending, revised)
	}
	for _, pending := range scope.Pending {
		if pending.Reason != QualityCheckScopeReasonIdentityChanged || pending.RequiredAction != QualityCheckActionRequired {
			t.Fatalf("revised pending=%+v", pending)
		}
	}

	finalScope, err := BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002"})
	if err != nil || finalScope.ReusableCount != 0 || finalScope.PendingCount != 122 {
		t.Fatalf("Final Review scope was polluted by QC results: scope=%+v err=%v", finalScope, err)
	}
	for _, pending := range finalScope.Pending {
		if pending.Reason != ReviewScopeReasonMissingReview {
			t.Fatalf("Final Review pending=%+v", pending)
		}
	}

	recordQualityCheckRatings(t, root, catalog, "qc-002", "qc-001", "A", revised)
	completed, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002"})
	if err != nil || completed.ACount != 122 || completed.CarryForwardCount != 119 || completed.CurrentResultCount != 3 || completed.PendingCount != 0 || !completed.ReadyForFinalReview {
		t.Fatalf("completed QC scope=%+v err=%v", completed, err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-003"}); err != nil {
		t.Fatal(err)
	}
	third, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-003", PreviousSnapshotID: "qc-002"})
	if err != nil || third.ACount != 122 || third.CarryForwardCount != 122 || third.PendingCount != 0 || !third.ReadyForFinalReview {
		t.Fatalf("multi-revision lineage scope=%+v err=%v", third, err)
	}
}

func TestReviewRecordBatchesRejectThirtyOneAndPageExampleMix(t *testing.T) {
	root, catalog := complete122PromotionFixture(t)
	materializeSnapshotSources(t, root, catalog)
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "batch-boundaries"}); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordQualityCheckResultBatch(root, catalog, QualityCheckRecordBatchOptions{
		Locale: "zh-CN", SnapshotID: "batch-boundaries", StartIndex: 1, Limit: 31, Rating: "A",
	}); err == nil {
		t.Fatal("Quality Check limit 31 was accepted")
	}
	if _, err := RecordQualityCheckResultBatch(root, catalog, QualityCheckRecordBatchOptions{
		Locale: "zh-CN", SnapshotID: "batch-boundaries", StartIndex: 103, Limit: 2, Rating: "A",
	}); err == nil || !strings.Contains(err.Error(), "must not mix") {
		t.Fatalf("Quality Check Page/Example mixed range error=%v", err)
	}
	if _, err := RecordRetranslationReviewBatch(root, catalog, RetranslationReviewBatchRecordOptions{
		Locale: "zh-CN", SnapshotID: "batch-boundaries", StartIndex: 103, Limit: 2,
		Rating: "A", Decision: "approved", Summary: "reviewed", Reviewer: "test", Rubric: TranslationQualityRubric,
	}); err == nil || !strings.Contains(err.Error(), "must not mix") {
		t.Fatalf("Final Review Page/Example mixed range error=%v", err)
	}
}

func TestQualityCheckResultsNeverSatisfyFinalReviewOrPromotion(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 2, "qc-001")
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "A", []string{"lesson/1", "lesson/2"})
	qcScope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-001"})
	if err != nil || !qcScope.ReadyForFinalReview || qcScope.ACount != 2 {
		t.Fatalf("QC scope=%+v err=%v", qcScope, err)
	}
	finalScope, err := BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: "zh-CN", SnapshotID: "qc-001"})
	if err != nil || finalScope.ReusableCount != 0 || finalScope.PendingCount != 2 {
		t.Fatalf("Final Review scope=%+v err=%v", finalScope, err)
	}
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil || plan.CanApply || plan.ReviewApprovedCount != 0 || !reflect.DeepEqual(plan.MissingReview, []string{"lesson/1", "lesson/2"}) {
		t.Fatalf("promotion accepted QC results: plan=%+v err=%v", plan, err)
	}
}

func TestQualityCheckRecordRejectsOverwriteAndSnapshotManifestMutation(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 1, "qc-001")
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "A", []string{"lesson/1"})
	_, err := RecordQualityCheckResults(root, catalog, QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "qc-001", UnitIDs: []string{"lesson/1"}, Rating: "B"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("overwrite error=%v", err)
	}
	manifestPath := qualityCheckSnapshotManifestPath(root, "zh-CN", "qc-001")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(data, ' '), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-001"})
	if err == nil || !strings.Contains(err.Error(), "incompatible identity") {
		t.Fatalf("snapshot binding error=%v", err)
	}
}
