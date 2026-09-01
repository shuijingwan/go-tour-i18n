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
