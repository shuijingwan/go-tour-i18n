package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlossaryReviewRequiredByRetranslationExport(t *testing.T) {
	root := t.TempDir()
	locale := "zz-ZZ"
	writeGlossaryReviewTestGlossary(t, root, locale)

	if _, err := ExportRetranslationBatch(root, retranslationTestCatalog(1), RetranslationExportOptions{Locale: locale}); err == nil || !strings.Contains(err.Error(), "Glossary Review gate missing or stale") {
		t.Fatalf("export without Glossary Review error = %v", err)
	}
	if _, _, err := RecordGlossaryReview(root, locale, "review-1", "independent-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := ExportRetranslationBatch(root, retranslationTestCatalog(1), RetranslationExportOptions{Locale: locale}); err != nil {
		t.Fatalf("export with current passed Glossary Review: %v", err)
	}
}

func TestGlossaryReviewBecomesStaleAfterAnyGlossaryByteChange(t *testing.T) {
	root := t.TempDir()
	locale := "zz-ZZ"
	path := writeGlossaryReviewTestGlossary(t, root, locale)
	if _, _, err := RecordGlossaryReview(root, locale, "review-1", "reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	if err := RequireCurrentGlossaryReview(root, locale); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RequireCurrentGlossaryReview(root, locale); err == nil || !strings.Contains(err.Error(), "missing or stale") {
		t.Fatalf("changed glossary error = %v", err)
	}
}

func TestGlossaryReviewRejectsInvalidReceipts(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GlossaryReviewReceipt) []byte
		want   string
	}{
		{name: "malformed", mutate: func(*GlossaryReviewReceipt) []byte { return []byte("{") }, want: "malformed"},
		{name: "non-passed", mutate: func(receipt *GlossaryReviewReceipt) []byte {
			receipt.Decision = "failed"
			receipt.Findings = []string{"术语决策不稳定"}
			return marshalGlossaryReviewTestReceipt(t, receipt)
		}, want: "non-passed"},
		{name: "wrong locale", mutate: func(receipt *GlossaryReviewReceipt) []byte {
			receipt.Locale = "other-AA"
			return marshalGlossaryReviewTestReceipt(t, receipt)
		}, want: "wrong locale"},
		{name: "rubric mismatch", mutate: func(receipt *GlossaryReviewReceipt) []byte {
			receipt.Rubric = "glossary-review/v0"
			return marshalGlossaryReviewTestReceipt(t, receipt)
		}, want: "rubric/version mismatch"},
		{name: "schema mismatch", mutate: func(receipt *GlossaryReviewReceipt) []byte {
			receipt.SchemaVersion++
			return marshalGlossaryReviewTestReceipt(t, receipt)
		}, want: "schema/version mismatch"},
		{name: "glossary path mismatch", mutate: func(receipt *GlossaryReviewReceipt) []byte {
			receipt.GlossaryPath = "locales/other-AA/glossary.yaml"
			return marshalGlossaryReviewTestReceipt(t, receipt)
		}, want: "glossary path mismatch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			locale := "zz-ZZ"
			writeGlossaryReviewTestGlossary(t, root, locale)
			receipt, path, err := RecordGlossaryReview(root, locale, "review-1", "reviewer", "passed", nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, test.mutate(receipt), 0644); err != nil {
				t.Fatal(err)
			}
			if err := RequireCurrentGlossaryReview(root, locale); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("invalid receipt error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestGlossaryReviewRejectsAmbiguousCurrentAuthority(t *testing.T) {
	root := t.TempDir()
	locale := "zz-ZZ"
	writeGlossaryReviewTestGlossary(t, root, locale)
	for _, reviewID := range []string{"review-1", "review-2"} {
		if _, _, err := RecordGlossaryReview(root, locale, reviewID, "reviewer", "passed", nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := RequireCurrentGlossaryReview(root, locale); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous receipt error = %v", err)
	}
}

func TestGlossaryReviewRejectsFormalAndLegacyAuthorityTogether(t *testing.T) {
	root := t.TempDir()
	locale := "legacy-AA"
	writeGlossaryReviewTestGlossary(t, root, locale)
	writeLegacyGlossaryReviewTestCoverage(t, root, locale)
	if _, _, err := RecordGlossaryReview(root, locale, "formal-review", "reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	if err := RequireCurrentGlossaryReview(root, locale); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("formal and legacy authority error = %v", err)
	}
}

func TestLegacyGlossaryReviewCoverageIsFixedAndByteBound(t *testing.T) {
	root := t.TempDir()
	locale := "legacy-AA"
	glossaryPath := writeGlossaryReviewTestGlossary(t, root, locale)
	writeLegacyGlossaryReviewTestCoverage(t, root, locale)
	if err := RequireCurrentGlossaryReview(root, locale); err != nil {
		t.Fatalf("current fixed legacy coverage: %v", err)
	}
	if _, err := ExportRetranslationBatch(root, retranslationTestCatalog(1), RetranslationExportOptions{Locale: locale}); err != nil {
		t.Fatalf("legacy locale export: %v", err)
	}
	data, err := os.ReadFile(glossaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(glossaryPath, append(data, '#'), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RequireCurrentGlossaryReview(root, locale); err == nil || !strings.Contains(err.Error(), "missing or stale") {
		t.Fatalf("changed legacy glossary error = %v", err)
	}
}

func TestMatchingSurfaceReviewDoesNotGrantFutureLocaleCoverage(t *testing.T) {
	root := t.TempDir()
	locale := "future-AA"
	writeGlossaryReviewTestGlossary(t, root, locale)
	_, _, _ = writeLegacyGlossaryReviewTestSurfaceGate(t, root, locale)
	if err := RequireCurrentGlossaryReview(root, locale); err == nil || !strings.Contains(err.Error(), "missing or stale") {
		t.Fatalf("future locale with matching Surface Review error = %v", err)
	}
}

func TestRepositoryLegacyGlossaryReviewCoverageIsFrozen(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(legacyGlossaryReviewCoveragePath)))
	if err != nil {
		t.Fatal(err)
	}
	var coverage legacyGlossaryReviewCoverage
	if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &coverage); err != nil {
		t.Fatal(err)
	}
	if len(coverage.Entries) != 18 {
		t.Fatalf("frozen legacy coverage entries = %d, want migration baseline 18", len(coverage.Entries))
	}
	for _, entry := range coverage.Entries {
		glossary, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.GlossaryPath)))
		if err != nil {
			t.Fatalf("read %s glossary: %v", entry.Locale, err)
		}
		if got := hashBytes(glossary); got != entry.GlossarySHA256 {
			t.Fatalf("%s glossary SHA-256 = %s, want frozen %s", entry.Locale, got, entry.GlossarySHA256)
		}
		current, err := currentLegacyGlossaryReviewCoverage(root, entry.Locale, entry.GlossaryPath, entry.GlossarySHA256)
		if err != nil {
			t.Fatalf("%s legacy provenance: %v", entry.Locale, err)
		}
		if !current {
			t.Fatalf("%s fixed legacy coverage is not current", entry.Locale)
		}
	}
}

func TestFormalGlossaryReviewDoesNotDependOnUnrelatedLegacyGate(t *testing.T) {
	root := t.TempDir()
	futureLocale := "future-AA"
	legacyLocale := "legacy-AA"
	writeGlossaryReviewTestGlossary(t, root, futureLocale)
	if _, _, err := RecordGlossaryReview(root, futureLocale, "formal-review", "reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	writeGlossaryReviewTestGlossary(t, root, legacyLocale)
	writeLegacyGlossaryReviewTestCoverage(t, root, legacyLocale)
	legacyGate, err := LocaleSurfaceReviewAGatePath(root, legacyLocale, "historic-review")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(legacyGate); err != nil {
		t.Fatal(err)
	}

	if err := RequireCurrentGlossaryReview(root, futureLocale); err != nil {
		t.Fatalf("formal future locale was coupled to unrelated legacy gate: %v", err)
	}
	if err := RequireCurrentGlossaryReview(root, legacyLocale); err == nil || !strings.Contains(err.Error(), "read legacy coverage Surface Review gate") {
		t.Fatalf("legacy locale with missing bound gate error = %v", err)
	}
}

func TestRepositoryLegacyGlossaryReviewCoverageAllowsExport(t *testing.T) {
	repository := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(repository, filepath.FromSlash(legacyGlossaryReviewCoveragePath)))
	if err != nil {
		t.Fatal(err)
	}
	var coverage legacyGlossaryReviewCoverage
	if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &coverage); err != nil {
		t.Fatal(err)
	}
	var selected legacyGlossaryReviewCoverageEntry
	for _, entry := range coverage.Entries {
		if entry.Locale == "zh-CN" {
			selected = entry
			break
		}
	}
	if selected.Locale == "" {
		t.Fatal("zh-CN is absent from fixed legacy coverage")
	}

	root := t.TempDir()
	copyGlossaryReviewTestFile(t, repository, root, selected.GlossaryPath)
	copyGlossaryReviewTestFile(t, repository, root, selected.SurfaceReviewGatePath)
	coverage.Entries = []legacyGlossaryReviewCoverageEntry{selected}
	coverageData, err := json.MarshalIndent(coverage, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	coveragePath := filepath.Join(root, filepath.FromSlash(legacyGlossaryReviewCoveragePath))
	if err := os.MkdirAll(filepath.Dir(coveragePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(coveragePath, append(coverageData, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ExportRetranslationBatch(root, retranslationTestCatalog(1), RetranslationExportOptions{Locale: selected.Locale}); err != nil {
		t.Fatalf("actual legacy live locale export: %v", err)
	}
}

func writeGlossaryReviewTestGlossary(t *testing.T, root, locale string) string {
	t.Helper()
	path := filepath.Join(root, "locales", locale, "glossary.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("mandatory:\n  Go: Go\nkeep:\n  - Go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func marshalGlossaryReviewTestReceipt(t *testing.T, receipt *GlossaryReviewReceipt) []byte {
	t.Helper()
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func writeLegacyGlossaryReviewTestCoverage(t *testing.T, root, locale string) {
	t.Helper()
	gatePath, gateData, glossarySHA := writeLegacyGlossaryReviewTestSurfaceGate(t, root, locale)
	coverage := legacyGlossaryReviewCoverage{
		SchemaVersion: legacyGlossaryReviewCoverageSchemaVersion,
		EvidenceType:  legacyGlossaryReviewCoverageEvidenceType,
		Stage:         legacyGlossaryReviewCoverageStage,
		Entries: []legacyGlossaryReviewCoverageEntry{{
			Locale: locale, GlossaryPath: glossaryReviewGlossaryPath(locale), GlossarySHA256: glossarySHA,
			SurfaceReviewID: "historic-review", SurfaceReviewGatePath: gatePath, SurfaceReviewGateSHA256: hashBytes(gateData),
		}},
	}
	data, err := json.MarshalIndent(coverage, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(legacyGlossaryReviewCoveragePath))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeLegacyGlossaryReviewTestSurfaceGate(t *testing.T, root, locale string) (string, []byte, string) {
	t.Helper()
	glossary, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(glossaryReviewGlossaryPath(locale))))
	if err != nil {
		t.Fatal(err)
	}
	gate := LocaleSurfaceReviewAGate{
		SchemaVersion: localeSurfaceReviewASchemaVersion,
		Locale:        locale,
		ReviewID:      "historic-review",
		Stage:         localeSurfaceReviewAStage,
		Decision:      "passed",
		Reviewer:      "historic-reviewer",
		Inputs:        LocaleSurfaceReviewAInputs{GlossarySHA256: hashBytes(glossary)},
	}
	data, err := json.MarshalIndent(gate, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	relative := filepath.ToSlash(filepath.Join("data", "locale-surface-reviews", locale, "historic-review.a-gate.json"))
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return relative, data, hashBytes(glossary)
}

func copyGlossaryReviewTestFile(t *testing.T, sourceRoot, destinationRoot, relative string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(destinationRoot, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
