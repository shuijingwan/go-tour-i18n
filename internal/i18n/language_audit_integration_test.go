package i18n

import (
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"strings"
	"testing"
)

func writeAuditIntegrationPolicy(t *testing.T, root, locale string) {
	t.Helper()
	ensureTestCurrentLocaleAuthority(t, root, locale, testLanguageAuditAuthority(locale, "Han", map[string]string{".": "。"}))
}

func TestLanguageAuditFormalTourCurrentGateAndHistoricalBoundary(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossaryWithoutCurrentAuthority(t, root)
	source := []byte("* Source\n\nRead [[https://go.dev][Go]] guide.\n")
	c := &Catalog{Pages: []Page{{ID: "audit/1", Article: "audit.article", Source: source, SourceSHA256: sum(source)}}}
	u, _ := c.Unit("audit/1")
	bad := []byte("* 标题\n\n查看 [[https://go.dev][Go]]目录。\n")
	g, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateTranslationUnitRepresentation(root, c, u, "zh-CN", bad, g); err != nil {
		t.Fatal("historical structure unexpectedly acquired current policy", err)
	}
	if err := ValidateTranslationUnitCandidate(root, c, u.ID, "zh-CN", bad); err == nil {
		t.Fatal("missing policy accepted")
	}
	writeAuditIntegrationPolicy(t, root, "zh-CN")
	if err := ValidateCandidateForLocale(root, c, u.ID, "zh-CN", bad); err == nil || !strings.Contains(err.Error(), "script-adjacency") {
		t.Fatal("deterministic defect did not block direct Tour validator", err)
	}
	good := []byte("* 标题\n\n查看 [[https://go.dev][Go]] 目录。\n")
	if err := ValidateTranslationUnitCandidate(root, c, u.ID, "zh-CN", good); err != nil {
		t.Fatal(err)
	}
}
func TestLanguageAuditFormalExampleCandidatesNeverBlock(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	writeAuditIntegrationPolicy(t, root, "zh-CN")
	source := "package main\n// Check the issue carefully.\nfunc main(){ println(\"ASCII . -- keyword func\") }\n"
	target := []byte(strings.Replace(source, "Check the issue carefully.", "请查看 issue。", 1))
	u := goExampleValidationUnit(source)
	c := &Catalog{Examples: []Example{{ID: u.ID, SourcePath: u.SourcePath, Source: u.Source, SourceSHA256: u.SourceSHA256}}}
	if err := ValidateTranslationUnitCandidate(root, c, u.ID, "zh-CN", target); err != nil {
		t.Fatal("candidate blocked", err)
	}
	g, _ := LoadGlossary(root, "zh-CN")
	r, err := TranslationUnitLanguageAudit(root, c, u, "zh-CN", target, g)
	if err != nil || len(r.Diagnostics) != 1 || r.Diagnostics[0].Decision != languageaudit.Candidate {
		t.Fatal(r, err)
	}
	bad := []byte(strings.Replace(string(target), "issue。", "issue.", 1))
	if err := ValidateGoExampleCandidate(root, u, "zh-CN", bad); err == nil || !strings.Contains(err.Error(), "sentence-punctuation") {
		t.Fatal(err)
	}
}
