package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func makeRevalidationFixture(t *testing.T) (string, *Catalog, string, string, []byte, []byte) {
	t.Helper()
	root, catalog, batchID := makeRetranslationProcessBatch(t, 1)
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	validationPath := filepath.Join(batchDir, "validation", "lesson-1.json")
	var evidence RetranslationValidation
	b, err := os.ReadFile(validationPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &evidence); err != nil {
		t.Fatal(err)
	}
	evidence.Status, evidence.Error = "validation_failed", "obsolete validator failure"
	if err := writeTranslationJSON(validationPath, evidence); err != nil {
		t.Fatal(err)
	}
	var result RetranslationProcessResult
	resultPath := filepath.Join(batchDir, "result.json")
	b, err = os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatal(err)
	}
	result.Units[0].Status, result.Units[0].Error = evidence.Status, evidence.Error
	recountRetranslationResult(&result)
	if err := writeTranslationJSON(resultPath, result); err != nil {
		t.Fatal(err)
	}
	oldValidation, _ := os.ReadFile(validationPath)
	raw, _ := os.ReadFile(filepath.Join(batchDir, evidence.RawResponsePath))
	candidate, _ := os.ReadFile(filepath.Join(batchDir, evidence.CandidatePath))
	return root, catalog, batchID, string(raw), oldValidation, candidate
}

func TestRetranslationRevalidateArchivesEvidenceAndKeepsAttemptAndArtifacts(t *testing.T) {
	root, catalog, batchID, rawBefore, oldValidation, candidateBefore := makeRevalidationFixture(t)
	got, err := RevalidateRetranslationCandidate(root, catalog, RetranslationRevalidateOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "passed" || got.PreviousStatus != "validation_failed" || got.Attempt != 1 || got.Revalidation != 1 {
		t.Fatalf("result=%+v", got)
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	history, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(got.HistoryPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(history, oldValidation) {
		t.Fatal("history is not the exact previous validation evidence")
	}
	var validation RetranslationValidation
	b, _ := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(got.ValidationPath)))
	if err := json.Unmarshal(b, &validation); err != nil {
		t.Fatal(err)
	}
	if validation.Attempt != 1 || validation.Status != "passed" || validation.RawResponsePath != "raw-responses/lesson-1.article" {
		t.Fatalf("validation=%+v", validation)
	}
	rawAfter, _ := os.ReadFile(filepath.Join(batchDir, validation.RawResponsePath))
	candidateAfter, _ := os.ReadFile(filepath.Join(batchDir, validation.CandidatePath))
	if string(rawAfter) != rawBefore || !bytes.Equal(candidateAfter, candidateBefore) {
		t.Fatal("revalidation changed raw response or candidate")
	}
	var processResult RetranslationProcessResult
	b, _ = os.ReadFile(filepath.Join(batchDir, "result.json"))
	_ = json.Unmarshal(b, &processResult)
	if processResult.Units[0].Status != "passed" || processResult.ValidationPassed != 1 || processResult.ValidationFailed != 0 {
		t.Fatalf("process result=%+v", processResult)
	}
}

func TestRetranslationRevalidateHistoryIsIndependentSequence(t *testing.T) {
	root, catalog, batchID, _, _, _ := makeRevalidationFixture(t)
	options := RetranslationRevalidateOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"}
	first, err := RevalidateRetranslationCandidate(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RevalidateRetranslationCandidate(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	if first.Revalidation != 1 || second.Revalidation != 2 || second.Attempt != 1 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}

func TestRetranslationRevalidateRejectsMissingCandidate(t *testing.T) {
	root, catalog, batchID, _, _, _ := makeRevalidationFixture(t)
	path := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "candidates", "lesson-1.article")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := RevalidateRetranslationCandidate(root, catalog, RetranslationRevalidateOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"}); err == nil {
		t.Fatal("missing candidate was accepted")
	}
}

func TestRetranslationRevalidateCanRemainValidationFailedWithoutRetry(t *testing.T) {
	root, catalog, batchID := processRetryFixture(t, 1, func(raw string) string {
		return appendBeforeRetranslationArtifactEOF(raw, " `bad`")
	})
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	rawPath := filepath.Join(batchDir, "raw-responses", "lesson-1.article")
	candidatePath := filepath.Join(batchDir, "candidates", "lesson-1.article")
	rawBefore, _ := os.ReadFile(rawPath)
	candidateBefore, _ := os.ReadFile(candidatePath)
	got, err := RevalidateRetranslationCandidate(root, catalog, RetranslationRevalidateOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "validation_failed" || got.Attempt != 1 || got.Revalidation != 1 {
		t.Fatalf("result=%+v", got)
	}
	rawAfter, _ := os.ReadFile(rawPath)
	candidateAfter, _ := os.ReadFile(candidatePath)
	if !bytes.Equal(rawBefore, rawAfter) || !bytes.Equal(candidateBefore, candidateAfter) {
		t.Fatal("failed revalidation changed translation artifacts")
	}
	if _, err := os.Stat(filepath.Join(batchDir, "retries", "lesson-1")); !os.IsNotExist(err) {
		t.Fatalf("revalidation created retry artifacts: %v", err)
	}
}
