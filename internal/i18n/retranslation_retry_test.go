package i18n

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func processRetryFixture(t *testing.T, count int, firstRaw func(string) string) (string, *Catalog, string) {
	t.Helper()
	root, catalog, batchID := makeRetranslationProcessBatch(t, count)
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	rawPath := filepath.Join(batchDir, "raw-responses", "lesson-1.article")
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rawPath, []byte(firstRaw(string(raw))), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN"}); err != nil {
		t.Fatal(err)
	}
	return root, catalog, batchID
}

func writeRetryRaw(t *testing.T, root, batchID, pageID string, attempt int, raw string) string {
	t.Helper()
	flat := strings.TrimSuffix(flattenedPageArticleName(pageID), ".article")
	dir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "retries", flat)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "attempt-"+fmtAttempt(attempt)+".article")
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func processExampleRetryFixture(t *testing.T, firstRaw func(string) string) (string, *Catalog, string, string, string) {
	t.Helper()
	root, catalog, batchID, name := makeRetranslationExampleProcessBatch(t)
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	rawPath := filepath.Join(batchDir, "raw-responses", name)
	validRaw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rawPath, []byte(firstRaw(string(validRaw))), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batchID}); err != nil {
		t.Fatal(err)
	}
	return root, catalog, batchID, name, string(validRaw)
}

func writeExampleRetryRaw(t *testing.T, root, batchID, inputName string, attempt int, raw string) string {
	t.Helper()
	flatID := strings.TrimSuffix(inputName, filepath.Ext(inputName))
	dir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "retries", flatID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "attempt-"+fmtAttempt(attempt)+filepath.Ext(inputName))
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func fmtAttempt(attempt int) string {
	return strings.Repeat("0", 3-len(strconv.Itoa(attempt))) + strconv.Itoa(attempt)
}

func TestRetranslationRetryRejectsPassedUnknownAndMissingAttempt(t *testing.T) {
	root, catalog, batchID := processRetryFixture(t, 1, func(raw string) string { return raw })
	if _, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"}); err == nil || !strings.Contains(err.Error(), "not retryable") {
		t.Fatalf("passed retry error = %v", err)
	}
	if _, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "missing/1"}); err == nil || !strings.Contains(err.Error(), "not in retranslation batch") {
		t.Fatalf("unknown page error = %v", err)
	}
	if _, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: "missing-batch", UnitID: "lesson/1"}); err == nil {
		t.Fatal("unknown batch accepted")
	}

	root, catalog, batchID = processRetryFixture(t, 1, func(raw string) string { return raw + " `bad`" })
	if _, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"}); err == nil || !strings.Contains(err.Error(), "attempt-002.article") {
		t.Fatalf("missing attempt error = %v", err)
	}
}

func TestRetranslationRetryUnitIDPage(t *testing.T) {
	root, catalog, batchID := processRetryFixture(t, 1, func(raw string) string { return raw + " `bad`" })
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	validRaw, _ := os.ReadFile(filepath.Join(batchDir, "inputs", "lesson-1.article"))
	writeRetryRaw(t, root, batchID, "lesson/1", 2, string(validRaw))
	result, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"})
	if err != nil || result.ValidationPassed != 1 {
		t.Fatalf("page --unit-id result=%+v err=%v", result, err)
	}
	if _, err := os.Stat(filepath.Join(batchDir, "retries", "lesson-1", "attempt-002.article")); err != nil {
		t.Fatalf("historical Page retry extension changed: %v", err)
	}
}

func TestRetranslationRetryExampleLifecycleUsesGoAttempts(t *testing.T) {
	root, catalog, batchID, name, validRaw := processExampleRetryFixture(t, func(raw string) string {
		return strings.Replace(raw, "通过通道发送该值。", "使用了幻灯片。", 1)
	})
	unitID := catalog.Examples[0].ID
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	statusPath := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	canonicalPath := filepath.Join(root, "locales", "zh-CN", "candidates", "sentinel.go")
	if err := os.WriteFile(statusPath, []byte("status sentinel\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(canonicalPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(canonicalPath, []byte("canonical sentinel\n"), 0644); err != nil {
		t.Fatal(err)
	}

	invalidAttempt := strings.Replace(validRaw, "通过通道发送该值。", "仍然使用幻灯片。", 1)
	attempt2 := writeExampleRetryRaw(t, root, batchID, name, 2, invalidAttempt)
	result, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: unitID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Units[0].Status != "validation_failed" || result.ValidationFailed != 1 || filepath.Ext(attempt2) != ".txt" {
		t.Fatalf("example attempt 2 result=%+v path=%s", result, attempt2)
	}
	flatID := strings.TrimSuffix(name, filepath.Ext(name))
	validationPath := filepath.Join(batchDir, "validation", flatID+".json")
	validation, _ := os.ReadFile(validationPath)
	if !strings.Contains(string(validation), `"attempt": 2`) || !strings.Contains(string(validation), `"raw_response_path": "retries/`+flatID+`/attempt-002.txt"`) || !strings.Contains(string(validation), "禁止译法") {
		t.Fatalf("example attempt 2 validation=%s", validation)
	}

	writeExampleRetryRaw(t, root, batchID, name, 3, validRaw)
	result, err = ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: unitID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Units[0].Status != "passed" || result.ValidationPassed != 1 {
		t.Fatalf("example attempt 3 result=%+v", result)
	}
	candidateName := retranslationUnitCandidateName(&TranslationUnit{ID: unitID, Kind: UnitKindExample})
	candidate, err := os.ReadFile(filepath.Join(batchDir, "candidates", candidateName))
	if err != nil || !strings.Contains(string(candidate), "通过通道发送该值。") {
		t.Fatalf("example retry candidate=%q err=%v", candidate, err)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := os.Stat(filepath.Join(batchDir, "retries", flatID, "attempt-"+fmtAttempt(attempt)+"-validation.json")); err != nil {
			t.Fatalf("missing example validation history %d: %v", attempt, err)
		}
	}
	for path, want := range map[string]string{statusPath: "status sentinel\n", canonicalPath: "canonical sentinel\n"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("formal data changed at %s: %q err=%v", path, got, err)
		}
	}
}

func TestRetranslationRetryExampleRestoreFailureAndPreflightSafety(t *testing.T) {
	t.Run("restore failure", func(t *testing.T) {
		root, catalog, batchID, name, validRaw := processExampleRetryFixture(t, func(raw string) string {
			return strings.Replace(raw, "通过通道发送该值。", "使用了幻灯片。", 1)
		})
		broken := strings.Replace(validRaw, translationTokenRE.FindString(validRaw), "", 1)
		writeExampleRetryRaw(t, root, batchID, name, 2, broken)
		result, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: catalog.Examples[0].ID})
		if err != nil || result.Units[0].Status != "restore_failed" {
			t.Fatalf("example restore retry result=%+v err=%v", result, err)
		}
	})
	for _, tt := range []struct {
		name   string
		mutate func(*RetranslationBatchManifest)
		want   string
	}{
		{"source hash", func(m *RetranslationBatchManifest) { m.Units[0].SourceSHA256 = strings.Repeat("0", 64) }, "source metadata"},
		{"input hash", func(m *RetranslationBatchManifest) { m.Units[0].InputSHA256 = strings.Repeat("0", 64) }, "input_sha256"},
		{"unit kind", func(m *RetranslationBatchManifest) { m.Units[0].UnitKind = UnitKindPage }, "unit_kind"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root, catalog, batchID, name, validRaw := processExampleRetryFixture(t, func(raw string) string {
				return strings.Replace(raw, "通过通道发送该值。", "使用了幻灯片。", 1)
			})
			writeExampleRetryRaw(t, root, batchID, name, 2, validRaw)
			rewriteProcessManifest(t, root, batchID, tt.mutate)
			_, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: catalog.Examples[0].ID})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("preflight error=%v, want %q", err, tt.want)
			}
			flatID := strings.TrimSuffix(name, filepath.Ext(name))
			if _, statErr := os.Stat(filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID, "retries", flatID, "attempt-001-validation.json")); !os.IsNotExist(statErr) {
				t.Fatalf("unsafe retry evidence created: %v", statErr)
			}
		})
	}
}

func TestRetranslationRetryLifecyclePreservesEvidenceAndOtherPages(t *testing.T) {
	root, catalog, batchID := processRetryFixture(t, 2, func(raw string) string { return raw + " `bad`" })
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	originalRawPath := filepath.Join(batchDir, "raw-responses", "lesson-1.article")
	originalRaw, _ := os.ReadFile(originalRawPath)
	originalValidation, _ := os.ReadFile(filepath.Join(batchDir, "validation", "lesson-1.json"))
	otherCandidatePath := filepath.Join(batchDir, "candidates", "lesson-2.article")
	otherValidationPath := filepath.Join(batchDir, "validation", "lesson-2.json")
	otherCandidate, _ := os.ReadFile(otherCandidatePath)
	otherValidation, _ := os.ReadFile(otherValidationPath)
	canonicalPath := filepath.Join(root, "locales", "zh-CN", "candidates", "lesson-1.article")
	statusPath := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	if err := os.MkdirAll(filepath.Dir(canonicalPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(canonicalPath, []byte("canonical\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statusPath, []byte("status\n"), 0644); err != nil {
		t.Fatal(err)
	}
	validRaw, _ := os.ReadFile(filepath.Join(batchDir, "raw-responses", "lesson-2.article"))

	missingToken := strings.Replace(string(validRaw), translationTokenRE.FindString(string(validRaw)), "", 1)
	attempt2 := writeRetryRaw(t, root, batchID, "lesson/1", 2, missingToken)
	attempt2Before, _ := os.ReadFile(attempt2)
	result, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Units[0].Status != "restore_failed" || result.RestoreFailed != 1 || result.ValidationPassed != 1 {
		t.Fatalf("attempt 2 result = %+v", result)
	}
	history1, _ := os.ReadFile(filepath.Join(batchDir, "retries", "lesson-1", "attempt-001-validation.json"))
	if string(history1) != string(originalValidation) {
		t.Fatal("attempt-001 validation history differs")
	}
	if _, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"}); err == nil || !strings.Contains(err.Error(), "attempt-003.article") {
		t.Fatalf("repeated attempt error = %v", err)
	}
	attempt2After, _ := os.ReadFile(attempt2)
	if string(attempt2After) != string(attempt2Before) {
		t.Fatal("attempt-002 was overwritten")
	}

	invalidRaw := string(validRaw) + " `bad`"
	writeRetryRaw(t, root, batchID, "lesson/1", 3, invalidRaw)
	result, err = ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Units[0].Status != "validation_failed" || result.ValidationFailed != 1 {
		t.Fatalf("attempt 3 result = %+v", result)
	}

	writeRetryRaw(t, root, batchID, "lesson/1", 4, string(validRaw))
	result, err = ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.RestorePassed != 2 || result.RestoreFailed != 0 || result.ValidationPassed != 2 || result.ValidationFailed != 0 || result.Units[0].Status != "passed" {
		t.Fatalf("attempt 4 result = %+v", result)
	}
	validation, _ := os.ReadFile(filepath.Join(batchDir, "validation", "lesson-1.json"))
	if !strings.Contains(string(validation), `"raw_response_path": "retries/lesson-1/attempt-004.article"`) {
		t.Fatalf("final validation = %s", validation)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := os.Stat(filepath.Join(batchDir, "retries", "lesson-1", "attempt-"+fmtAttempt(attempt)+"-validation.json")); err != nil {
			t.Fatalf("missing validation history %d: %v", attempt, err)
		}
	}
	for path, before := range map[string][]byte{otherCandidatePath: otherCandidate, otherValidationPath: otherValidation} {
		after, _ := os.ReadFile(path)
		if string(after) != string(before) {
			t.Fatalf("other page changed: %s", path)
		}
	}
	afterRaw, _ := os.ReadFile(originalRawPath)
	if string(afterRaw) != string(originalRaw) {
		t.Fatal("original raw response changed")
	}
	for path, want := range map[string]string{canonicalPath: "canonical\n", statusPath: "status\n"} {
		got, _ := os.ReadFile(path)
		if string(got) != want {
			t.Fatalf("formal data changed: %s", path)
		}
	}
}

func TestRetranslationRetryAcceptsInitialRestoreFailureAndRequiresOriginalInput(t *testing.T) {
	root, catalog, batchID := processRetryFixture(t, 1, func(raw string) string {
		return strings.Replace(raw, translationTokenRE.FindString(raw), "", 1)
	})
	batchDir := filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	validRaw, _ := os.ReadFile(filepath.Join(batchDir, "inputs", "lesson-1.article"))
	writeRetryRaw(t, root, batchID, "lesson/1", 2, string(validRaw))
	result, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"})
	if err != nil || result.ValidationPassed != 1 {
		t.Fatalf("restore retry result=%+v err=%v", result, err)
	}

	root, catalog, batchID = processRetryFixture(t, 1, func(raw string) string { return raw + " `bad`" })
	batchDir = filepath.Join(root, "data", "retranslation-runs", "zh-CN", batchID)
	inputPath := filepath.Join(batchDir, "inputs", "lesson-1.article")
	input, _ := os.ReadFile(inputPath)
	input = append(input, 'x')
	if err := os.WriteFile(inputPath, input, 0644); err != nil {
		t.Fatal(err)
	}
	rewriteProcessManifest(t, root, batchID, func(m *RetranslationBatchManifest) { m.Units[0].InputSHA256 = sum(input) })
	writeRetryRaw(t, root, batchID, "lesson/1", 2, string(validRaw))
	if _, err := ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batchID, UnitID: "lesson/1"}); err == nil || !strings.Contains(err.Error(), "regenerated Default protected input differs") {
		t.Fatalf("input mismatch error = %v", err)
	}
}
