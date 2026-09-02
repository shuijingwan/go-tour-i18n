package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func retryOutputResult(status, failure string) *i18n.RetranslationProcessResult {
	result := &i18n.RetranslationProcessResult{
		SchemaVersion: 2, BatchID: "codex-ko-KR-001", Locale: "ko-KR", UnitCount: 2, RetryAttempt: 2,
		Units: []i18n.RetranslationUnitResult{
			{UnitID: "basics/11", UnitKind: i18n.UnitKindPage, Status: status, CandidatePath: "candidates/basics-11.article", ValidationPath: "validation/basics-11.json", Error: failure},
			{UnitID: "basics/12", UnitKind: i18n.UnitKindPage, Status: "passed", CandidatePath: "candidates/basics-12.article", ValidationPath: "validation/basics-12.json"},
		},
	}
	if status == "passed" {
		result.RestorePassed = 2
		result.ValidationPassed = 2
	} else {
		result.RestorePassed = 2
		result.ValidationPassed = 1
		result.ValidationFailed = 1
	}
	return result
}

func TestRetranslationRetryDefaultPassSummaryOmitsUnitsJSON(t *testing.T) {
	var output bytes.Buffer
	if err := writeRetranslationRetryOutput(&output, retryOutputResult("passed", ""), "basics/11", false); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{
		"重译重试：PASS\n", "batch_id: codex-ko-KR-001\n", "locale: ko-KR\n", "unit_id: basics/11\n",
		"attempt: 2\n", "status: passed\n", "restore_passed: 2\n", "validation_passed: 2\n", "validation_failed: 0\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("PASS summary missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{`"units"`, "basics/12", "candidate_path=", "validation_path=", "reason="} {
		if strings.Contains(got, unwanted) {
			t.Errorf("PASS summary contains %q:\n%s", unwanted, got)
		}
	}
}

func TestRetranslationRetryDefaultFailedSummaryShowsCurrentEvidence(t *testing.T) {
	var output bytes.Buffer
	result := retryOutputResult("validation_failed", "current validation failure")
	if err := writeRetranslationRetryOutput(&output, result, "basics/11", false); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{
		"重译重试：FAILED\n", "attempt: 2\n", "status: validation_failed\n", "validation_failed: 1\n",
		"失败 Unit：unit_id=basics/11 status=validation_failed validation_path=validation/basics-11.json",
		"candidate_path=candidates/basics-11.article", `reason="current validation failure"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FAILED summary missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `"units"`) || strings.Contains(got, "basics/12") {
		t.Fatalf("FAILED summary expanded full batch:\n%s", got)
	}
}

func TestRetranslationRetryJSONKeepsFullProcessResultSchema(t *testing.T) {
	var output bytes.Buffer
	result := retryOutputResult("passed", "")
	if err := writeRetranslationRetryOutput(&output, result, "basics/11", true); err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(output.Bytes(), &document); err != nil {
		t.Fatalf("retry --json output is not JSON: %v\n%s", err, output.String())
	}
	units, ok := document["units"].([]any)
	if !ok || len(units) != 2 {
		t.Fatalf("retry --json units = %#v", document["units"])
	}
	if document["batch_id"] != result.BatchID || document["locale"] != result.Locale || document["schema_version"] != float64(result.SchemaVersion) {
		t.Fatalf("retry --json identity = %#v", document)
	}
	if _, exists := document["retry_attempt"]; exists {
		t.Fatalf("presentation-only retry attempt changed JSON schema: %#v", document)
	}
	if strings.Contains(output.String(), "重译重试：") {
		t.Fatalf("retry --json mixed human summary with JSON:\n%s", output.String())
	}
}

func TestRetranslationRevalidationOutput(t *testing.T) {
	result := &i18n.RetranslationRevalidationResult{SchemaVersion: 1, BatchID: "codex-ko-KR-004", Locale: "ko-KR", UnitID: "concurrency/2", UnitKind: i18n.UnitKindPage, Attempt: 1, Revalidation: 1, PreviousStatus: "validation_failed", Status: "passed", HistoryPath: "revalidation-history/concurrency-2/revalidation-001-validation.json", ValidationPath: "validation/concurrency-2.json", ResultPath: "result.json", ValidationPassed: 2, ValidationFailed: 1}
	var output bytes.Buffer
	if err := writeRetranslationRevalidationOutput(&output, result, false); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"重译重新验证：PASS", "attempt: 1", "revalidation: 1", "previous_status: validation_failed", "status: passed", result.HistoryPath} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("summary missing %q:\n%s", want, output.String())
		}
	}
	output.Reset()
	if err := writeRetranslationRevalidationOutput(&output, result, true); err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(output.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document["attempt"] != float64(1) || document["revalidation"] != float64(1) {
		t.Fatalf("json=%#v", document)
	}
}

func promotionOutputPlan() *i18n.RetranslationPromotionPlan {
	return &i18n.RetranslationPromotionPlan{
		Locale: "ko-KR", UnitCount: 122, PageCount: 103, ExampleCount: 19,
		ChangedCount: 2, UnchangedCount: 120, EOFNormalizedCount: 1,
		ReviewApprovedCount: 122, CanApply: true,
		MissingReview: []string{}, RejectedReview: []string{}, InvalidReview: []string{},
		Units: []i18n.RetranslationPromotionUnit{{
			UnitID: "flowcontrol/1", UnitKind: i18n.UnitKindPage, BatchID: "codex-ko-KR-001",
			SourceCandidatePath:    "data/retranslation-runs/ko-KR/codex-ko-KR-001/candidates/flowcontrol-1.article",
			CanonicalCandidatePath: "locales/ko-KR/candidates/flowcontrol/1.article",
			SourceCandidateSHA256:  strings.Repeat("a", 64), CandidateSHA256: strings.Repeat("b", 64), Changed: true,
		}},
	}
}

func TestRetranslationPromotionDefaultReadySummaryOmitsUnitDetails(t *testing.T) {
	var output bytes.Buffer
	if err := writeRetranslationPromotionOutput(&output, promotionOutputPlan(), false, false); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{"重译提升：READY\n", "locale: ko-KR\n", "mode: dry-run\n", "unit_count: 122（103 Page，19 Example）", "review_approved_count: 122", "changed: 2", "unchanged: 120", "eof_normalized: 1", "can_apply: true", "--apply"} {
		if !strings.Contains(got, want) {
			t.Errorf("READY summary missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{`"units"`, "flowcontrol/1", "codex-ko-KR-001", strings.Repeat("a", 64), "candidate_path"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("READY summary contains unit detail %q:\n%s", unwanted, got)
		}
	}
}

func TestRetranslationPromotionDefaultBlockedSummaryShowsOnlyActionableFailures(t *testing.T) {
	plan := promotionOutputPlan()
	plan.CanApply = false
	plan.ReviewApprovedCount = 118
	plan.MissingEvidence = []string{"basics/1"}
	plan.MissingReview = []string{"basics/2"}
	plan.RejectedReview = []string{"basics/3"}
	plan.InvalidReview = []string{"basics/4"}
	var output bytes.Buffer
	if err := writeRetranslationPromotionOutput(&output, plan, false, false); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{"重译提升：BLOCKED", "can_apply: false", "missing_evidence (1):\n- basics/1", "missing_review (1):\n- basics/2", "rejected_review (1):\n- basics/3", "invalid_review (1):\n- basics/4"} {
		if !strings.Contains(got, want) {
			t.Errorf("BLOCKED summary missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "flowcontrol/1") || strings.Contains(got, `"units"`) || strings.Contains(got, "--apply") {
		t.Fatalf("BLOCKED summary contains normal unit details or apply hint:\n%s", got)
	}
}

func TestRetranslationPromotionJSONKeepsFullPlan(t *testing.T) {
	plan := promotionOutputPlan()
	var output bytes.Buffer
	if err := writeRetranslationPromotionOutput(&output, plan, false, true); err != nil {
		t.Fatal(err)
	}
	var decoded i18n.RetranslationPromotionPlan
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("promote --json output is not JSON: %v\n%s", err, output.String())
	}
	if len(decoded.Units) != 1 || decoded.Units[0].UnitID != "flowcontrol/1" || decoded.Units[0].SourceCandidateSHA256 != strings.Repeat("a", 64) {
		t.Fatalf("promote --json lost full units: %#v", decoded.Units)
	}
	if strings.Contains(output.String(), "重译提升：") || strings.Contains(output.String(), "mode: dry-run") {
		t.Fatalf("promote --json mixed human summary with JSON:\n%s", output.String())
	}
}

func TestRetranslationPromotionApplySuccessSummary(t *testing.T) {
	var output bytes.Buffer
	if err := writeRetranslationPromotionOutput(&output, promotionOutputPlan(), true, false); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{"重译提升：APPLIED", "mode: apply", "applied: true", "can_apply: true"} {
		if !strings.Contains(got, want) {
			t.Errorf("apply summary missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `"units"`) || strings.Contains(got, "flowcontrol/1") || strings.Contains(got, "--apply") {
		t.Fatalf("apply summary expanded units or suggested apply again:\n%s", got)
	}
}
