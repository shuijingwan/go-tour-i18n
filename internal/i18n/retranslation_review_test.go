package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		Rubric: "translation-quality-v1", Rating: "A", Summary: "Accurate and fluent.", Issues: []string{},
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
