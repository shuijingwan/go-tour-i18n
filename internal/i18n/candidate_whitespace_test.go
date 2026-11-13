package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetranslationCandidateArtifactRejectsTrailingWhitespace(t *testing.T) {
	root, c, batch := makeRetranslationProcessBatch(t, 1)
	p, err := prepareTranslationUnitInput(&TranslationUnit{ID: c.Pages[0].ID, Kind: UnitKindPage, Source: c.Pages[0].Source, SourceSHA256: c.Pages[0].SourceSHA256}, mustWhitespaceGlossary(t, root))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "data/retranslation-runs/zh-CN", batch, "raw-responses/lesson-1.article"))
	if err != nil {
		t.Fatal(err)
	}
	restored, failures := p.restore(string(raw))
	if len(failures) != 0 {
		t.Fatal(failures)
	}
	good := canonicalizeRetranslationArtifactEOF([]byte(restored))
	if err := ValidateTranslationUnitCandidate(root, c, "lesson/1", "zh-CN", good); err != nil {
		t.Fatal("clean candidate rejected", err)
	}
	if err := validateRetranslationCandidateArtifact(good); err != nil {
		t.Fatal("clean artifact rejected", err)
	}
	// Artifact eligibility must not alter the semantic validator's policy.
	if err := ValidateTranslationUnitCandidate(root, c, "lesson/1", "zh-CN", bytes.Replace(good, []byte("\n"), []byte(" \n"), 1)); err != nil {
		t.Fatal("workflow hygiene leaked into translation semantics", err)
	}
	for _, suffix := range []string{" ", "\t", "\r", "  ", " \t"} {
		t.Run("line-ending-"+strings.ReplaceAll(suffix, " ", "space"), func(t *testing.T) {
			bad := bytes.Replace(good, []byte("\n"), []byte(suffix+"\n"), 1)
			before := bytes.Clone(bad)
			if err := validateRetranslationCandidateArtifact(bad); err == nil || !strings.Contains(err.Error(), "trailing whitespace at line 1") {
				t.Fatal("candidate with Git trailing whitespace accepted", err)
			}
			if !bytes.Equal(before, bad) {
				t.Fatal("validator normalized candidate bytes")
			}
		})
	}
}

func mustWhitespaceGlossary(t *testing.T, root string) *Glossary {
	t.Helper()
	g, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestCandidateWhitespaceRejectsWithoutChangingSemantics(t *testing.T) {
	for name, text := range map[string]string{
		"markdown-hard-break": "first line  \nsecond line\n",
		"preformatted-code":   "```go\nprintln(1) \n```\n",
		"html":                "<p>text \n</p>\n",
		"blank-line":          "text\n \t\ntext\n",
		"last-line":           "text ",
	} {
		t.Run(name, func(t *testing.T) {
			original := []byte(text)
			if err := validateRetranslationCandidateArtifact(original); err == nil {
				t.Fatal("semantic whitespace was silently accepted")
			}
			if string(original) != text {
				t.Fatal("semantic whitespace was rewritten")
			}
		})
	}
	// A meaningful non-ASCII space is outside Git's default blank-at-eol rule.
	if err := validateRetranslationCandidateArtifact([]byte("text\u00a0\n")); err != nil {
		t.Fatal("format gate expanded into Unicode whitespace normalization", err)
	}
	if got := canonicalizeRetranslationArtifactEOF([]byte("text  \n\n")); !bytes.Equal(got, []byte("text  \n")) {
		t.Fatal("single_lf changed non-EOF bytes", string(got))
	}
}

func TestProcessRejectsCandidateArtifactTrailingWhitespace(t *testing.T) {
	for name, suffix := range map[string]string{"space": " ", "tab": "\t", "CR": "\r", "two-spaces": "  "} {
		t.Run(name, func(t *testing.T) {
			testProcessCandidateArtifactWhitespace(t, suffix)
		})
	}
}

func testProcessCandidateArtifactWhitespace(t *testing.T, suffix string) {
	t.Helper()
	root, c, batch := makeRetranslationProcessBatch(t, 2)
	dir := filepath.Join(root, "data/retranslation-runs/zh-CN", batch)
	rawPath := filepath.Join(dir, "raw-responses/lesson-1.article")
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	bad := bytes.Replace(raw, []byte("\n"), []byte(suffix+"\n"), 1)
	if err := os.WriteFile(rawPath, bad, 0644); err != nil {
		t.Fatal(err)
	}
	result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil || result.ValidationFailed != 1 || result.ValidationPassed != 1 || result.RestorePassed != 2 {
		t.Fatal("process candidate eligibility", result, err)
	}
	var validation RetranslationValidation
	b, err := os.ReadFile(filepath.Join(dir, "validation/lesson-1.json"))
	if err != nil || json.Unmarshal(b, &validation) != nil || validation.Status != "validation_failed" || validation.CandidateSHA256 != "" {
		t.Fatal("whitespace defect received a PASS binding", validation, err)
	}
	if after, err := os.ReadFile(rawPath); err != nil || !bytes.Equal(after, bad) {
		t.Fatal("process rewrote raw response", err)
	}
}

func TestRetryRejectsCandidateArtifactWhitespace(t *testing.T) {
	root, c, batch := processRetryFixture(t, 1, func(raw string) string {
		return string(bytes.Replace([]byte(raw), []byte("\n"), []byte(" \n"), 1))
	})
	dir := filepath.Join(root, "data/retranslation-runs/zh-CN", batch)
	raw, err := os.ReadFile(filepath.Join(dir, "raw-responses/lesson-1.article"))
	if err != nil {
		t.Fatal(err)
	}
	badRetry := bytes.Replace(raw, []byte(" \n"), []byte("\t\n"), 1)
	path := writeRetryRaw(t, root, batch, "lesson/1", 2, string(badRetry))
	got, err := ProcessRetranslationRetry(root, c, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batch, UnitID: "lesson/1"})
	if err != nil || got.ValidationFailed != 1 || !strings.Contains(got.Units[0].Error, "trailing whitespace at line 1") {
		t.Fatal("retry accepted invalid candidate artifact", got, err)
	}
	var validation RetranslationValidation
	b, err := os.ReadFile(filepath.Join(dir, got.Units[0].ValidationPath))
	if err != nil || json.Unmarshal(b, &validation) != nil || validation.Attempt != 2 || validation.Status != "validation_failed" || validation.CandidateSHA256 != "" {
		t.Fatal("retry hygiene failure granted PASS or changed attempt", validation, err)
	}
	if b, err := os.ReadFile(path); err != nil || !bytes.Equal(b, badRetry) {
		t.Fatal("retry normalized response bytes", err)
	}
}

func TestCandidateArtifactHygieneDoesNotChangeTranslationAuthority(t *testing.T) {
	root, c, _ := makeRetranslationProcessBatch(t, 1)
	copyTestLanguageAuditMachineAuthority(t, root)
	copyBundleAuthority(t, root, append(append([]string{}, commentGateAuthorityPaths...), "internal/i18n/validate.go", "internal/i18n/go_example_validate.go", "internal/i18n/go_example_protect.go", "internal/i18n/translate_protect.go", "internal/i18n/preformatted.go"))
	u, err := c.Unit("lesson/1")
	if err != nil {
		t.Fatal(err)
	}
	before, err := currentTranslationAuthority(root, u, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"candidate_artifact_hygiene.go", "retranslation_process.go", "retranslation_retry.go", "retranslation_revalidate.go"} {
		// Change only workflow implementation bytes in this isolated fixture.
		// The bridge's translation authority must remain a separate identity.
		if err := os.WriteFile(filepath.Join(root, "internal/i18n", path), []byte("package i18n\n// synthetic workflow-only change\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	after, err := currentTranslationAuthority(root, u, "zh-CN")
	if err != nil || before != after {
		t.Fatal("workflow hygiene changed translation authority", before, after, err)
	}
}

func TestCandidateArtifactProcessPreservesSingleLFContract(t *testing.T) {
	root, c, batch := makeRetranslationProcessBatch(t, 1)
	dir := filepath.Join(root, "data/retranslation-runs/zh-CN", batch)
	path := filepath.Join(dir, "raw-responses/lesson-1.article")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(bytes.Clone(raw), '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch}); err == nil || !strings.Contains(err.Error(), "exactly one LF") {
		t.Fatal("noncanonical raw EOF accepted", err)
	}
	for _, name := range []string{"candidates", "validation", "result.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatal("EOF failure published formal output", name, err)
		}
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil || got.ValidationPassed != 1 {
		t.Fatal("clean single_lf candidate rejected", got, err)
	}
	candidate, err := os.ReadFile(filepath.Join(dir, got.Units[0].CandidatePath))
	if err != nil || validateRetranslationArtifactEOF(candidate) != nil {
		t.Fatal("candidate EOF contract changed", err)
	}
}

func TestRevalidatePassedWhitespaceCandidateAllowsRetry(t *testing.T) {
	root, c, batch := makeRetranslationProcessBatch(t, 1)
	dir := filepath.Join(root, "data/retranslation-runs/zh-CN", batch)
	read := func(path string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	cleanRaw := read("raw-responses/lesson-1.article")
	badRaw := bytes.Replace(cleanRaw, []byte("\n"), []byte(" \n"), 1)
	if err := os.WriteFile(filepath.Join(dir, "raw-responses/lesson-1.article"), badRaw, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch}); err != nil {
		t.Fatal(err)
	}
	// Model evidence emitted by the old validator which missed this defect.
	// These synthetic TempDir bytes are not a new canonical validation PASS.
	candidate := read("candidates/lesson-1.article")
	var old RetranslationValidation
	if err := json.Unmarshal(read("validation/lesson-1.json"), &old); err != nil {
		t.Fatal(err)
	}
	old.Status, old.Error, old.CandidateSHA256 = "passed", "", sum(candidate)
	if err := writeTranslationJSON(filepath.Join(dir, "validation/lesson-1.json"), old); err != nil {
		t.Fatal(err)
	}
	var result RetranslationProcessResult
	if err := json.Unmarshal(read("result.json"), &result); err != nil {
		t.Fatal(err)
	}
	result.Units[0].Status, result.Units[0].Error = "passed", ""
	recountRetranslationResult(&result)
	if err := writeTranslationJSON(filepath.Join(dir, "result.json"), result); err != nil {
		t.Fatal(err)
	}
	previousValidation, previousResult := read("validation/lesson-1.json"), read("result.json")
	got, err := RevalidateRetranslationCandidate(root, c, RetranslationRevalidateOptions{Locale: "zh-CN", BatchID: batch, UnitID: "lesson/1"})
	if err != nil || got.PreviousStatus != "passed" || got.Status != "validation_failed" || got.Attempt != 1 || !strings.Contains(got.Error, "trailing whitespace") {
		t.Fatal("old SHA-bound PASS was not revalidated as a defect", got, err)
	}
	if got.ResultHistoryPath != "" || !bytes.Equal(read(got.HistoryPath), previousValidation) {
		t.Fatal("ordinary revalidation history contract changed")
	}
	resultHistory := strings.TrimSuffix(got.HistoryPath, "validation.json") + "result.json"
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(resultHistory))); !os.IsNotExist(err) {
		t.Fatal("ordinary revalidation created result history", err)
	}
	currentResult := read("result.json")
	if err := json.Unmarshal(currentResult, &result); err != nil || bytes.Equal(currentResult, previousResult) || result.Units[0].Status != "validation_failed" || result.ValidationFailed != 1 || result.ValidationPassed != 0 {
		t.Fatal("ordinary revalidation did not update current result", result, err)
	}
	var failed RetranslationValidation
	if err := json.Unmarshal(read("validation/lesson-1.json"), &failed); err != nil || failed.CandidateSHA256 != "" || failed.Attempt != old.Attempt || failed.RawResponsePath != old.RawResponsePath {
		t.Fatal("failed revalidation changed attempt/provenance or granted SHA binding", failed, err)
	}
	if !bytes.Equal(candidate, read("candidates/lesson-1.article")) || !bytes.Equal(badRaw, read("raw-responses/lesson-1.article")) {
		t.Fatal("revalidation rewrote SHA-bound candidate or raw bytes")
	}
	writeRetryRaw(t, root, batch, "lesson/1", 2, string(cleanRaw))
	retried, err := ProcessRetranslationRetry(root, c, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batch, UnitID: "lesson/1"})
	if err != nil || retried.ValidationPassed != 1 || retried.Units[0].Status != "passed" {
		t.Fatal("ordinary attempt-002 retry unavailable", retried, err)
	}
	var final RetranslationValidation
	if err := json.Unmarshal(read("validation/lesson-1.json"), &final); err != nil || final.Attempt != 2 || final.CandidateSHA256 != sum(read(final.CandidatePath)) {
		t.Fatal("retry did not bind corrected attempt-002 candidate", final, err)
	}
}
