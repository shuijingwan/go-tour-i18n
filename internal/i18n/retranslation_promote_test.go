package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func processedPromotionFixture(t *testing.T, count int) (string, *Catalog, string) {
	t.Helper()
	root, catalog, batchID := makeRetranslationProcessBatch(t, count)
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
	writeApprovedPromotionReviews(t, root, catalog, batchID)
	return root, catalog, batchID
}

func TestPromotionBatchREIsLocaleAwareAndKeepsZHCNCompatibility(t *testing.T) {
	if !promotionBatchRE("zh-CN").MatchString("chatgpt-zh-CN-013") {
		t.Fatal("zh-CN historical batch was rejected")
	}
	if !promotionBatchRE("zh-CN").MatchString("codex-zh-CN-014") {
		t.Fatal("Codex batch was rejected")
	}
	if !promotionBatchRE("ja-JP").MatchString("chatgpt-ja-JP-001") {
		t.Fatal("ja-JP batch was rejected")
	}
	if promotionBatchRE("ja-JP").MatchString("chatgpt-zh-CN-001") {
		t.Fatal("ja-JP matcher accepted a different locale")
	}
}

func addProcessedPromotionBatch(t *testing.T, root string, catalog *Catalog, batchID string, ids []string) {
	t.Helper()
	result, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", BatchID: batchID, UnitIDs: ids, Limit: len(ids), AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	batchDir := filepath.Join(root, result.BatchPath)
	if err := os.Mkdir(filepath.Join(batchDir, "raw-responses"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := readRetranslationManifest(t, root, batchID)
	for _, record := range manifest.Units {
		input, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(record.InputPath)))
		if err != nil {
			t.Fatal(err)
		}
		translated := strings.ReplaceAll(string(input), "* Page", "* 新页面")
		translated = strings.ReplaceAll(translated, "Use ", "新批次使用 ")
		translated = strings.ReplaceAll(translated, " on this page.", "。")
		translated = strings.ReplaceAll(translated, "Translate this comment.", "翻译这条注释。")
		if err := os.WriteFile(filepath.Join(batchDir, "raw-responses", retranslationUnitInputName(&TranslationUnit{ID: record.UnitID, Kind: record.UnitKind})), []byte(translated), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
	writeApprovedPromotionReviews(t, root, catalog, batchID)
}

func writeApprovedPromotionReviews(t *testing.T, root string, catalog *Catalog, batchID string) {
	t.Helper()
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	manifest := readRetranslationManifest(t, root, batchID)
	resultBytes, err := os.ReadFile(filepath.Join(batchDir, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := decodeRetranslationProcessResult(resultBytes)
	if err != nil {
		t.Fatal(err)
	}
	results := make(map[string]RetranslationUnitResult, len(result.Units))
	for _, record := range result.Units {
		results[record.UnitID] = record
	}
	for _, record := range manifest.Units {
		unit, err := catalog.Unit(record.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		unitResult := results[record.UnitID]
		candidate, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(unitResult.CandidatePath)))
		if err != nil {
			t.Fatal(err)
		}
		validationBytes, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(unitResult.ValidationPath)))
		if err != nil {
			t.Fatal(err)
		}
		validation, err := decodeRetranslationValidation(validationBytes, unit)
		if err != nil {
			t.Fatal(err)
		}
		review := TranslationReview{
			SchemaVersion: TranslationReviewSchemaVersion, BatchID: batchID, Locale: "zh-CN",
			UnitID: unit.ID, UnitKind: unit.Kind, SourceSHA256: unit.SourceSHA256, Attempt: validation.Attempt,
			CandidatePath: unitResult.CandidatePath, CandidateSHA256: sum(candidate),
			ValidationPath: unitResult.ValidationPath, ValidationSHA256: sum(validationBytes),
			Decision: "approved", Reviewer: "promotion-test", ReviewedAt: "2026-08-20T12:00:00Z",
			Rubric: TranslationQualityRubric, Rating: "A", Summary: "Approved for promotion.", Issues: []string{},
		}
		writeRetranslationReview(t, filepath.Join(batchDir, "review", retranslationReviewName(unit)), review)
	}
}

func writePromotionStatus(t *testing.T, root string, catalog *Catalog, canonical string) {
	writePromotionStatusForLocale(t, root, "zh-CN", catalog, canonical)
}

func writePromotionStatusForLocale(t *testing.T, root, locale string, catalog *Catalog, canonical string) {
	t.Helper()
	localeDir := filepath.Join(root, "locales", locale)
	if err := os.MkdirAll(filepath.Join(localeDir, "candidates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "locale.json"), []byte(`{"locale":"`+locale+`","phase":"scaffold","translation_unit":"present.Section"}`), 0644); err != nil {
		t.Fatal(err)
	}
	statuses := make([]Status, 0, len(catalog.Pages))
	for _, page := range catalog.Pages {
		path := ""
		state := "pending"
		if canonical != "" {
			path = canonicalCandidatePath(locale, page.ID)
			state = "ready"
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte(canonical), 0644); err != nil {
				t.Fatal(err)
			}
		}
		statuses = append(statuses, Status{UnitID: page.ID, State: state, Attempts: 7, SourceSHA256: page.SourceSHA256, CandidatePath: path})
	}
	for i := range catalog.Examples {
		example := &catalog.Examples[i]
		eligible, err := hasTranslatableGoExampleComment(example.Source)
		if err != nil {
			t.Fatal(err)
		}
		if eligible {
			statuses = append(statuses, Status{UnitID: example.ID, State: "pending", SourceSHA256: example.SourceSHA256})
		}
	}
	if err := writeStatuses(filepath.Join(localeDir, "status.tsv"), statuses); err != nil {
		t.Fatal(err)
	}
}

func TestRetranslationPromotionSupportsLocaleAwareWorkflow(t *testing.T) {
	const locale = "ja-JP"
	root, catalog, batchID := makeRetranslationProcessBatchForLocale(t, locale, 1)
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: locale, BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecordRetranslationReview(root, catalog, RetranslationReviewRecordOptions{
		Locale: locale, BatchID: batchID, UnitID: "lesson/1", Rating: "A", Decision: "approved",
		Summary: "Valid fixture.", Reviewer: "test", Rubric: TranslationQualityRubric, Now: func() time.Time { return time.Unix(0, 0) },
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckRetranslationReviews(root, catalog, RetranslationReviewCheckOptions{Locale: locale, BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
	writePromotionStatusForLocale(t, root, locale, catalog, "")
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: locale, Apply: true})
	if err != nil || !plan.CanApply || plan.Locale != locale {
		t.Fatalf("promotion plan=%+v err=%v", plan, err)
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", locale, "status.tsv"))
	if err != nil || len(statuses) != 1 || statuses[0].State != "ready" || !strings.HasPrefix(statuses[0].CandidatePath, "locales/ja-JP/") {
		t.Fatalf("ja-JP statuses=%+v err=%v", statuses, err)
	}
}

func complete122PromotionFixture(t *testing.T) (string, *Catalog) {
	t.Helper()
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	catalog := retranslationTestCatalog(103)
	for i := 0; i < 19; i++ {
		id := fmt.Sprintf("example:demo/example-%02d.go", i+1)
		source := []byte("package main\n\n// Translate this comment.\nfunc main() {}\n")
		catalog.Examples = append(catalog.Examples, Example{ID: id, SourcePath: "_content/tour/demo/" + filepath.Base(strings.TrimPrefix(id, "example:demo/")), Source: source, SourceSHA256: sum(source)})
	}
	pageIDs := make([]string, 0, 103)
	for _, page := range catalog.Pages {
		pageIDs = append(pageIDs, page.ID)
	}
	exampleIDs := make([]string, 0, 19)
	for _, example := range catalog.Examples {
		exampleIDs = append(exampleIDs, example.ID)
	}
	for start, number := 0, 1; start < len(pageIDs); start, number = start+DefaultRetranslationReviewBatchLimit, number+1 {
		end := start + DefaultRetranslationReviewBatchLimit
		if end > len(pageIDs) {
			end = len(pageIDs)
		}
		addProcessedPromotionBatch(t, root, catalog, fmt.Sprintf("chatgpt-zh-CN-%03d", number), pageIDs[start:end])
	}
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-005", exampleIDs)
	writePromotionStatus(t, root, catalog, "")
	return root, catalog
}

func TestRetranslationPromoteComplete122UnitWorkflow(t *testing.T) {
	root, catalog := complete122PromotionFixture(t)
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil || !plan.CanApply || plan.UnitCount != 122 || plan.PageCount != 103 || plan.ExampleCount != 19 || plan.ReviewApprovedCount != 122 || len(plan.Units) != 122 {
		t.Fatalf("122-unit plan=%+v err=%v", plan, err)
	}
	for _, unit := range plan.Units {
		if unit.UnitKind == UnitKindExample && unit.EOFNormalized {
			t.Fatalf("Example %s was EOF-normalized", unit.UnitID)
		}
	}
	plan, err = PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true})
	if err != nil || !plan.CanApply {
		t.Fatalf("122-unit apply plan=%+v err=%v", plan, err)
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", "zh-CN", "status.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	readyExamples := 0
	for _, status := range statuses {
		if strings.HasPrefix(status.UnitID, "example:") {
			readyExamples++
			if status.State != "ready" || status.Attempts != 1 || filepath.Ext(status.CandidatePath) != ".go" {
				t.Fatalf("promoted Example status=%+v", status)
			}
			candidate, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(status.CandidatePath)))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(candidate, []byte("package main\n\n// 翻译这条注释。\nfunc main() {}\n")) {
				t.Fatalf("promoted Example %s bytes=%q", status.UnitID, candidate)
			}
		}
	}
	if readyExamples != 19 {
		t.Fatalf("ready Examples=%d, want 19", readyExamples)
	}
}

func TestRetranslationPromoteComplete122UnitApplyRollback(t *testing.T) {
	root, catalog := complete122PromotionFixture(t)
	statusPath := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	statusBefore, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	candidatesDir := filepath.Join(root, "locales", "zh-CN", "candidates")
	calls := 0
	rename := func(old, new string) error {
		calls++
		if calls == 4 {
			return errors.New("injected full-workflow failure")
		}
		return os.Rename(old, new)
	}
	if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true, rename: rename}); err == nil || !strings.Contains(err.Error(), "injected full-workflow failure") {
		t.Fatalf("error = %v", err)
	}
	statusAfter, _ := os.ReadFile(statusPath)
	if !bytes.Equal(statusBefore, statusAfter) {
		t.Fatal("failed 122-unit apply changed status")
	}
	entries, err := os.ReadDir(candidatesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed 122-unit apply left %d canonical candidates", len(entries))
	}
}

func TestCanonicalizeCandidateEOF(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"two LF", "abc\n\n", "abc\n"},
		{"three LF", "abc\n\n\n", "abc\n"},
		{"one LF unchanged", "abc\n", "abc\n"},
		{"missing LF", "abc", "abc\n"},
		{"middle blank line", "abc\n\ndef\n\n", "abc\n\ndef\n"},
		{"trailing space preserved", "abc \n\n", "abc \n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalizeCandidateEOF([]byte(tt.in))
			if !bytes.Equal(got, []byte(tt.want)) {
				t.Fatalf("canonicalizeCandidateEOF(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRetranslationPromoteDryRunDoesNotModifyCanonicalAndLatestWins(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 2)
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-007", []string{"lesson/1"})
	addProcessedPromotionBatch(t, root, catalog, "codex-zh-CN-008", []string{"lesson/1"})
	writePromotionStatus(t, root, catalog, "old canonical\n")
	statusPath := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	statusBefore, _ := os.ReadFile(statusPath)
	canonicalPath := filepath.Join(root, filepath.FromSlash(canonicalCandidatePath("zh-CN", "lesson/1")))
	canonicalBefore, _ := os.ReadFile(canonicalPath)
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.UnitCount != 2 || plan.ChangedCount != 2 || plan.UnchangedCount != 0 || !plan.CanApply || plan.Units[0].BatchID != "codex-zh-CN-008" {
		t.Fatalf("plan = %+v", plan)
	}
	statusAfter, _ := os.ReadFile(statusPath)
	canonicalAfter, _ := os.ReadFile(canonicalPath)
	if !bytes.Equal(statusBefore, statusAfter) || !bytes.Equal(canonicalBefore, canonicalAfter) {
		t.Fatal("dry-run modified canonical data")
	}
}

func TestRetranslationPromoteLatestFailureDoesNotFallback(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1"})
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", "chatgpt-zh-CN-002")
	var result RetranslationProcessResult
	b, _ := os.ReadFile(filepath.Join(batchDir, "result.json"))
	_ = json.Unmarshal(b, &result)
	result.Units[0].Status = "validation_failed"
	if err := writeTranslationJSON(filepath.Join(batchDir, "result.json"), result); err != nil {
		t.Fatal(err)
	}
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil || plan.CanApply || len(plan.MissingEvidence) != 1 || !strings.Contains(plan.MissingEvidence[0], "validation_failed") {
		t.Fatalf("plan=%+v error=%v", plan, err)
	}
}

func TestRetranslationPromoteLatestMissingReviewDoesNotFallback(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1"})
	path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", "chatgpt-zh-CN-002", "review", "lesson-1.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil || plan.CanApply || !reflect.DeepEqual(plan.MissingReview, []string{"lesson/1"}) || plan.ReviewApprovedCount != 0 {
		t.Fatalf("plan=%+v error=%v", plan, err)
	}
}

func TestRetranslationPromoteReviewGate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string, *TranslationReview)
		field  func(*RetranslationPromotionPlan) []string
	}{
		{"missing review", func(t *testing.T, path string, review *TranslationReview) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		}, func(plan *RetranslationPromotionPlan) []string { return plan.MissingReview }},
		{"rejected", func(t *testing.T, path string, review *TranslationReview) {
			review.Decision = "rejected"
			writeRetranslationReview(t, path, *review)
		}, func(plan *RetranslationPromotionPlan) []string { return plan.RejectedReview }},
		{"approved rating B", func(t *testing.T, path string, review *TranslationReview) {
			review.Rating = "B"
			writeRetranslationReview(t, path, *review)
		}, func(plan *RetranslationPromotionPlan) []string { return plan.RejectedReview }},
		{"candidate hash mismatch", func(t *testing.T, path string, review *TranslationReview) {
			review.CandidateSHA256 = strings.Repeat("0", 64)
			writeRetranslationReview(t, path, *review)
		}, func(plan *RetranslationPromotionPlan) []string { return plan.InvalidReview }},
		{"validation hash mismatch", func(t *testing.T, path string, review *TranslationReview) {
			review.ValidationSHA256 = strings.Repeat("0", 64)
			writeRetranslationReview(t, path, *review)
		}, func(plan *RetranslationPromotionPlan) []string { return plan.InvalidReview }},
		{"unit mismatch", func(t *testing.T, path string, review *TranslationReview) {
			review.UnitID = "lesson/2"
			writeRetranslationReview(t, path, *review)
		}, func(plan *RetranslationPromotionPlan) []string { return plan.InvalidReview }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, batch := processedPromotionFixture(t, 1)
			writePromotionStatus(t, root, catalog, "old canonical\n")
			canonicalPath := filepath.Join(root, filepath.FromSlash(canonicalCandidatePath("zh-CN", "lesson/1")))
			before, _ := os.ReadFile(canonicalPath)
			reviewPath := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch, "review", "lesson-1.json")
			var review TranslationReview
			b, err := os.ReadFile(reviewPath)
			if err != nil || json.Unmarshal(b, &review) != nil {
				t.Fatalf("read review: %v", err)
			}
			test.mutate(t, reviewPath, &review)
			plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true})
			if err == nil || plan == nil || plan.CanApply || !reflect.DeepEqual(test.field(plan), []string{"lesson/1"}) {
				t.Fatalf("plan=%+v error=%v", plan, err)
			}
			after, _ := os.ReadFile(canonicalPath)
			if !bytes.Equal(before, after) {
				t.Fatal("blocked review gate changed canonical candidate")
			}
		})
	}
}

func TestRetranslationPromoteRejectsIncompleteAndInvalidEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string, *Catalog, string)
		want   string
	}{
		{"illegal batch", func(t *testing.T, root string, catalog *Catalog, batch string) {
			if err := os.MkdirAll(filepath.Join(root, "data", "retranslation-runs", "zh-CN", "bad"), 0755); err != nil {
				t.Fatal(err)
			}
		}, "illegal retranslation batch"},
		{"ambiguous batch number", func(t *testing.T, root string, catalog *Catalog, batch string) {
			addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-001", []string{"lesson/1"})
		}, "ambiguous or invalid retranslation batch number"},
		{"result validation mismatch", func(t *testing.T, root string, catalog *Catalog, batch string) {
			path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch, "validation", "lesson-1.json")
			var v RetranslationValidation
			b, _ := os.ReadFile(path)
			_ = json.Unmarshal(b, &v)
			v.Status = "validation_failed"
			if err := writeTranslationJSON(path, v); err != nil {
				t.Fatal(err)
			}
		}, "validation does not match"},
		{"validator failure", func(t *testing.T, root string, catalog *Catalog, batch string) {
			batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch)
			for _, relative := range []string{"candidates/lesson-1.article", "raw-responses/lesson-1.article"} {
				path := filepath.Join(batchDir, filepath.FromSlash(relative))
				b, _ := os.ReadFile(path)
				b = append(b, []byte(" `unexpected`\n")...)
				if err := os.WriteFile(path, b, 0644); err != nil {
					t.Fatal(err)
				}
			}
		}, "canonical validation"},
		{"GTI18N token", func(t *testing.T, root string, catalog *Catalog, batch string) {
			batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch)
			for _, relative := range []string{"candidates/lesson-1.article", "raw-responses/lesson-1.article"} {
				path := filepath.Join(batchDir, filepath.FromSlash(relative))
				b, _ := os.ReadFile(path)
				b = append(b, []byte(" GTI18N_leftover\n")...)
				if err := os.WriteFile(path, b, 0644); err != nil {
					t.Fatal(err)
				}
			}
		}, "contains GTI18N token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, catalog, batch := processedPromotionFixture(t, 1)
			tt.mutate(t, root, catalog, batch)
			if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"}); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestRetranslationPromoteMissingWorkflowUnitEvidenceCannotApply(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	source := []byte("* Missing\n\nMissing source.\n")
	catalog.Pages = append(catalog.Pages, Page{ID: "missing/1", Article: "missing.article", Source: source, SourceSHA256: sum(source)})
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil || plan.CanApply || plan.UnitCount != 2 || !reflect.DeepEqual(plan.MissingEvidence, []string{"missing/1"}) {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestRetranslationPromoteBindsInputRawAndCandidateEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string, string)
		want   string
	}{
		{"validator-valid candidate differs from restore", func(t *testing.T, root, batchDir string) {
			path := filepath.Join(batchDir, "candidates", "lesson-1.article")
			candidate, _ := os.ReadFile(path)
			candidate = bytes.Replace(candidate, []byte("页面"), []byte("网页"), 1)
			if err := ValidateCandidateForLocale(root, retranslationTestCatalog(1), "lesson/1", "zh-CN", candidate); err != nil {
				t.Fatalf("mutated candidate must remain validator-valid: %v", err)
			}
			if err := os.WriteFile(path, candidate, 0644); err != nil {
				t.Fatal(err)
			}
		}, "restored candidate does not match saved candidate"},
		{"saved input content", func(t *testing.T, root, batchDir string) {
			path := filepath.Join(batchDir, "inputs", "lesson-1.article")
			input, _ := os.ReadFile(path)
			input = append(input, 'x')
			if err := os.WriteFile(path, input, 0644); err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(batchDir, "manifest.json")
			manifest := readRetranslationManifestAt(t, manifestPath)
			manifest.Units[0].InputSHA256 = sum(input)
			if err := writeTranslationJSON(manifestPath, manifest); err != nil {
				t.Fatal(err)
			}
		}, "regenerated protected input differs"},
		{"input sha", func(t *testing.T, root, batchDir string) {
			manifestPath := filepath.Join(batchDir, "manifest.json")
			manifest := readRetranslationManifestAt(t, manifestPath)
			manifest.Units[0].InputSHA256 = strings.Repeat("0", 64)
			if err := writeTranslationJSON(manifestPath, manifest); err != nil {
				t.Fatal(err)
			}
		}, "input_sha256 mismatch"},
		{"illegal raw path", func(t *testing.T, root, batchDir string) {
			path := filepath.Join(batchDir, "validation", "lesson-1.json")
			var validation RetranslationValidation
			b, _ := os.ReadFile(path)
			_ = json.Unmarshal(b, &validation)
			validation.RawResponsePath = "../raw-responses/lesson-1.article"
			if err := writeTranslationJSON(path, validation); err != nil {
				t.Fatal(err)
			}
		}, "not a recognized attempt"},
		{"wrong retry attempt", func(t *testing.T, root, batchDir string) {
			retryDir := filepath.Join(batchDir, "retries", "lesson-1")
			if err := os.MkdirAll(retryDir, 0755); err != nil {
				t.Fatal(err)
			}
			raw, _ := os.ReadFile(filepath.Join(batchDir, "raw-responses", "lesson-1.article"))
			raw = bytes.Replace(raw, []byte("页面"), []byte("错误尝试"), 1)
			if err := os.WriteFile(filepath.Join(retryDir, "attempt-002.article"), raw, 0644); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(batchDir, "validation", "lesson-1.json")
			var validation RetranslationValidation
			b, _ := os.ReadFile(path)
			_ = json.Unmarshal(b, &validation)
			validation.RawResponsePath = "retries/lesson-1/attempt-002.article"
			if err := writeTranslationJSON(path, validation); err != nil {
				t.Fatal(err)
			}
		}, "missing retry attempt-001-validation.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, catalog, batch := processedPromotionFixture(t, 1)
			batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch)
			tt.mutate(t, root, batchDir)
			if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"}); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestRetranslationPromoteRejectsIncompleteOrRewoundRetryProvenance(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string)
		want   string
	}{
		{"arbitrary attempt-999", func(t *testing.T, batchDir string) {
			raw, _ := os.ReadFile(filepath.Join(batchDir, "raw-responses", "lesson-1.article"))
			retryDir := filepath.Join(batchDir, "retries", "lesson-1")
			if err := os.MkdirAll(retryDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(retryDir, "attempt-999.article"), raw, 0644); err != nil {
				t.Fatal(err)
			}
			setPromotionValidationRawPath(t, batchDir, "retries/lesson-1/attempt-999.article")
		}, "missing retry attempt-002.article"},
		{"attempt-002 missing history", func(t *testing.T, batchDir string) {
			raw, _ := os.ReadFile(filepath.Join(batchDir, "raw-responses", "lesson-1.article"))
			retryDir := filepath.Join(batchDir, "retries", "lesson-1")
			if err := os.MkdirAll(retryDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(retryDir, "attempt-002.article"), raw, 0644); err != nil {
				t.Fatal(err)
			}
			setPromotionValidationRawPath(t, batchDir, "retries/lesson-1/attempt-002.article")
		}, "missing retry attempt-001-validation.json"},
		{"rewound to attempt-001", func(t *testing.T, batchDir string) {
			retryDir := filepath.Join(batchDir, "retries", "lesson-1")
			if err := os.MkdirAll(retryDir, 0755); err != nil {
				t.Fatal(err)
			}
			raw, _ := os.ReadFile(filepath.Join(batchDir, "raw-responses", "lesson-1.article"))
			if err := os.WriteFile(filepath.Join(retryDir, "attempt-002.article"), raw, 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(retryDir, "attempt-001-validation.json"), []byte("{}\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}, "validation points to attempt-001 but retry history exists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, catalog, batch := processedPromotionFixture(t, 1)
			batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch)
			tt.mutate(t, batchDir)
			if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"}); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func setPromotionValidationRawPath(t *testing.T, batchDir, rawPath string) {
	t.Helper()
	path := filepath.Join(batchDir, "validation", "lesson-1.json")
	var validation RetranslationValidation
	b, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(b, &validation) != nil {
		t.Fatalf("read validation: %v", err)
	}
	validation.RawResponsePath = rawPath
	if err := writeTranslationJSON(path, validation); err != nil {
		t.Fatal(err)
	}
}

func readRetranslationManifestAt(t *testing.T, path string) RetranslationBatchManifest {
	t.Helper()
	var manifest RetranslationBatchManifest
	b, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(b, &manifest) != nil {
		t.Fatalf("read manifest %s: %v", path, err)
	}
	return manifest
}

func copyTestTree(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(target, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, info.Mode())
	})
}

func TestRetranslationPromoteSelectsRetryFinalCandidate(t *testing.T) {
	root, catalog, batch := processRetryFixture(t, 1, func(raw string) string {
		return appendBeforeRetranslationArtifactEOF(raw, " `bad`")
	})
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch)
	valid, _ := os.ReadFile(filepath.Join(batchDir, "raw-responses", "lesson-1.article"))
	valid = []byte(strings.Replace(string(valid), " `bad`", "", 1))
	writeRetryRaw(t, root, batch, "lesson/1", 2, string(valid))
	if _, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batch, UnitID: "lesson/1"}); err != nil {
		t.Fatal(err)
	}
	validationPath := filepath.Join(batchDir, "validation", "lesson-1.json")
	var validation RetranslationValidation
	validationData, _ := os.ReadFile(validationPath)
	_ = json.Unmarshal(validationData, &validation)
	if validation.RawResponsePath != "retries/lesson-1/attempt-002.article" {
		t.Fatalf("final raw_response_path = %q", validation.RawResponsePath)
	}
	if err := os.WriteFile(filepath.Join(batchDir, "raw-responses", "lesson-1.article"), []byte("corrupted initial attempt"), 0644); err != nil {
		t.Fatal(err)
	}
	writeApprovedPromotionReviews(t, root, catalog, batch)
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil || plan.UnitCount != 1 || plan.Units[0].BatchID != batch {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestRetranslationPromoteApplyUpdatesCanonicalAndPreservesAttempts(t *testing.T) {
	root, catalog, batch := processedPromotionFixture(t, 2)
	writePromotionStatus(t, root, catalog, "old\n")
	now := time.Date(2026, 8, 18, 3, 4, 5, 0, time.FixedZone("test", 8*60*60))
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true, Now: func() time.Time { return now }})
	if err != nil || plan.PageCount != 2 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", "zh-CN", "status.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range statuses {
		if status.State != "ready" || status.Attempts != 1 || status.SourceSHA256 == "" || status.CandidatePath != canonicalCandidatePath("zh-CN", status.UnitID) || status.UpdatedAt != "2026-08-17T19:04:05Z" || status.Note != "retranslation promoted from "+batch+"; passed canonical validator" {
			t.Fatalf("status = %+v", status)
		}
		candidate, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(status.CandidatePath)))
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateCandidateForLocale(root, catalog, status.UnitID, "zh-CN", candidate); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRetranslationPromoteUsesCurrentSourceRevisionAndRepairsStaleStatus(t *testing.T) {
	root, catalog, oldBatch := processedPromotionFixture(t, 1)
	writePromotionStatus(t, root, catalog, "old canonical\n")
	oldSource := catalog.Pages[0].SourceSHA256
	catalog.Pages[0].Source = []byte("* Page\n\nUse the updated `Go` source on this page.\n")
	catalog.Pages[0].SourceSHA256 = sum(catalog.Pages[0].Source)
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-002", []string{"lesson/1"})

	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil || !plan.CanApply || len(plan.Units) != 1 || plan.Units[0].BatchID != "chatgpt-zh-CN-002" {
		t.Fatalf("current revision plan=%+v err=%v", plan, err)
	}
	if plan.Units[0].BatchID == oldBatch {
		t.Fatal("promotion fell back to the old source revision")
	}
	if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true}); err != nil {
		t.Fatal(err)
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", "zh-CN", "status.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if statuses[0].State != "ready" || statuses[0].SourceSHA256 != catalog.Pages[0].SourceSHA256 || statuses[0].CandidatePath != canonicalCandidatePath("zh-CN", "lesson/1") || statuses[0].Note != "retranslation promoted from chatgpt-zh-CN-002; passed canonical validator" {
		t.Fatalf("promoted stale status=%+v old_source=%s", statuses[0], oldSource)
	}
	if err := CheckStatus(root, "zh-CN", catalog); err != nil {
		t.Fatalf("post-promotion status: %v", err)
	}
}

func TestRetranslationPromoteDoesNotUseOldSourceRevisionEvidence(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	writePromotionStatus(t, root, catalog, "old canonical\n")
	catalog.Pages[0].Source = []byte("* Page\n\nUse the updated `Go` source on this page.\n")
	catalog.Pages[0].SourceSHA256 = sum(catalog.Pages[0].Source)
	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil || plan.CanApply || !reflect.DeepEqual(plan.MissingEvidence, []string{"lesson/1"}) {
		t.Fatalf("old revision fallback plan=%+v err=%v", plan, err)
	}
	if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true}); err == nil || !strings.Contains(err.Error(), "promotion cannot apply") {
		t.Fatalf("old revision apply error=%v", err)
	}
}

func TestRetranslationPromoteCanonicalizesEOFWithoutChangingHistoricalCandidate(t *testing.T) {
	root, catalog, batch := processedPromotionFixture(t, 1)
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batch)
	sourcePath := filepath.Join(batchDir, "candidates", "lesson-1.article")
	rawPath := filepath.Join(batchDir, "raw-responses", "lesson-1.article")
	for _, path := range []string{sourcePath, rawPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeApprovedPromotionReviews(t, root, catalog, batch)
	sourceBefore, _ := os.ReadFile(sourcePath)
	canonicalized := canonicalizeCandidateEOF(sourceBefore)
	writePromotionStatus(t, root, catalog, "old canonical\n")
	canonicalPath := filepath.Join(root, filepath.FromSlash(canonicalCandidatePath("zh-CN", "lesson/1")))
	canonicalBefore, _ := os.ReadFile(canonicalPath)

	plan, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	page := plan.Units[0]
	if plan.EOFNormalizedCount != 1 || !page.EOFNormalized || page.SourceCandidateSHA256 != sum(sourceBefore) || page.CandidateSHA256 != sum(canonicalized) || !page.Changed {
		t.Fatalf("dry-run plan=%+v page=%+v", plan, page)
	}
	canonicalAfterDryRun, _ := os.ReadFile(canonicalPath)
	sourceAfterDryRun, _ := os.ReadFile(sourcePath)
	if !bytes.Equal(canonicalBefore, canonicalAfterDryRun) || !bytes.Equal(sourceBefore, sourceAfterDryRun) {
		t.Fatal("dry-run changed canonical or historical candidate")
	}

	if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true}); err != nil {
		t.Fatal(err)
	}
	canonicalAfterApply, _ := os.ReadFile(canonicalPath)
	sourceAfterApply, _ := os.ReadFile(sourcePath)
	if !bytes.Equal(canonicalAfterApply, canonicalized) {
		t.Fatalf("canonical candidate = %q, want %q", canonicalAfterApply, canonicalized)
	}
	if !bytes.Equal(sourceAfterApply, sourceBefore) {
		t.Fatal("apply changed historical batch candidate")
	}
	recheck, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if recheck.ChangedCount != 0 || recheck.UnchangedCount != 1 || recheck.EOFNormalizedCount != 1 || recheck.Units[0].Changed {
		t.Fatalf("post-apply plan = %+v", recheck)
	}
}

func TestRetranslationPromoteApplyRollbackOnCommitFailure(t *testing.T) {
	root, catalog, _ := processedPromotionFixture(t, 1)
	writePromotionStatus(t, root, catalog, "old\n")
	statusPath := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	candidatePath := filepath.Join(root, filepath.FromSlash(canonicalCandidatePath("zh-CN", "lesson/1")))
	statusBefore, _ := os.ReadFile(statusPath)
	candidateBefore, _ := os.ReadFile(candidatePath)
	calls := 0
	rename := func(old, new string) error {
		calls++
		if calls == 4 {
			return errors.New("injected failure")
		}
		return os.Rename(old, new)
	}
	if _, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", Apply: true, rename: rename}); err == nil || !strings.Contains(err.Error(), "injected failure") {
		t.Fatalf("error = %v", err)
	}
	statusAfter, _ := os.ReadFile(statusPath)
	candidateAfter, _ := os.ReadFile(candidatePath)
	if !bytes.Equal(statusBefore, statusAfter) || !bytes.Equal(candidateBefore, candidateAfter) {
		t.Fatal("failed apply left partial canonical state")
	}
}
