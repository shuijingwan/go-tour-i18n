package sitecontent

import (
	"encoding/json"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func formalAuditFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := repositoryRoot(t)
	paths := append(append([]string(nil), i18n.LanguageAuditAuthorityPaths...), "go.mod", "go.sum", "internal/i18n/glossary.go", "internal/i18n/glossary_context.go", "internal/i18n/glossary_compatibility.go", "internal/i18n/glossary_semantics.go", "internal/i18n/translation_authorization.go", "internal/i18n/bridge_successor.go")
	paths = append(paths, siteLanguageAuditPaths...)
	for _, p := range paths {
		b, err := os.ReadFile(filepath.Join(source, p))
		if err != nil {
			t.Fatal(err)
		}
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	policy := auditFixturePolicy("zh-CN", "Han")
	policy.ProtectedIdentities = nil
	m := i18n.Locale{Locale: "zh-CN", HTMLLang: "zh-CN", LanguageAudit: &languageaudit.Authority{Schema: languageaudit.PolicySchema, Version: 1, Policy: policy}}
	b, _ := json.Marshal(m)
	dir := filepath.Join(root, "locales", "zh-CN")
	_ = os.MkdirAll(dir, 0755)
	if err := os.WriteFile(filepath.Join(dir, "locale.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "glossary.yaml"), []byte("locale: zh-CN\nmandatory:\n  Go: Go\npreferred:\nforbidden:\nkeep:\n  - Go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}
func TestLanguageAuditFormalLearnDocsGate(t *testing.T) {
	root := formalAuditFixture(t)
	path := "_content/doc/audit.html"
	source := []byte(`<p>Read the issue with <code>cache</code>.</p>`)
	d, err := ParseWorkflowDocument(Source{Path: path, Kind: "page", Route: "/doc/audit", SHA256: digest(source)}, source, 1)
	if err != nil {
		t.Fatal(err)
	}
	a := PageAdapter{Document: *d, Source: source}
	u := i18n.TranslationUnit{ID: "audit-page", Kind: i18n.UnitKindPage, SourcePath: path, Source: source, SourceSHA256: digest(source)}
	c := &i18n.Catalog{Adapted: []i18n.AdaptedTranslationUnit{{Unit: u, Adapter: a}}}
	good := []byte(`<p>查看 issue 与 <code>cache</code>。</p>`)
	if err := i18n.ValidateTranslationUnitCandidate(root, c, u.ID, "zh-CN", good); err != nil {
		t.Fatal("review candidate blocked", err)
	}
	bad := []byte(strings.Replace(string(good), "code>。", "code>.", 1))
	if err := i18n.ValidateTranslationUnitCandidate(root, c, u.ID, "zh-CN", bad); err == nil || !strings.Contains(err.Error(), "sentence-punctuation") {
		t.Fatal("missing deterministic gate", err)
	}
}
func TestLanguageAuditFormalStructuredWholeRepresentationAndAttention(t *testing.T) {
	root := formalAuditFixture(t)
	path := "_content/site.tmpl"
	targetPath := "locales/zh-CN/structured/_content/site.tmpl"
	p := StructuredPlan{Locale: "zh-CN", Assets: []StructuredAsset{{ID: "source:" + path, TargetPath: targetPath, Document: &Document{Path: path}}}}
	source := []byte(`<p><i>Read the issue at <a href="https://go.dev">Go</a>.</i></p>`)
	good := []byte(`<p><i>请查看 issue，访问 <a href="https://go.dev">Go</a>。</i></p>`)
	r, err := structuredLanguageAttention(root, p, map[string][]byte{path: source}, map[string][]byte{targetPath: good})
	if err != nil || len(r.Candidates) == 0 {
		t.Fatal("candidate transport", r, err)
	}
	for _, c := range r.Candidates {
		if c.Classification != languageaudit.Candidate || c.Locale != "zh-CN" || c.Owner != path || c.Locator == "" {
			t.Fatal(c)
		}
	}
	bad := []byte(strings.Replace(string(good), "a>。", "a>.", 1))
	if _, err := structuredLanguageAttention(root, p, map[string][]byte{path: source}, map[string][]byte{targetPath: bad}); err == nil || !strings.Contains(err.Error(), "sentence-punctuation") {
		t.Fatal("frozen structure punctuation escaped current gate", err)
	}
	if err := os.WriteFile(filepath.Join(root, "locales/zh-CN/glossary.yaml"), []byte("drift"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := r.Verify(root); err == nil {
		t.Fatal("final dependency guard lost glossary drift")
	}
}
func TestLanguageAuditFormalStructuredPolicyDependencyGuard(t *testing.T) {
	root := formalAuditFixture(t)
	v := newClosureValidation(root)
	if err := v.bindLanguageAudit(root, "zh-CN"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "locales/zh-CN/locale.json")
	b, _ := os.ReadFile(path)
	b = append(b, '\n')
	_ = os.WriteFile(path, b, 0644)
	if err := v.verify(); err == nil {
		t.Fatal("current policy dependency mutation was cached")
	}
}

// Complete formal fixtures declare their own explicit policy before generation;
// this never edits the real locale or manufactures historical machine evidence.
func seedSiteLanguagePolicyFixture(t *testing.T, root, locale, script string) {
	t.Helper()
	source := repositoryRoot(t)
	real, _ := filepath.EvalSymlinks(source)
	actual, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(real, actual)
	if err != nil || rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		t.Fatal("fixture policy may not write repository authority")
	}
	paths := append(append(append([]string(nil), i18n.LanguageAuditAuthorityPaths...), siteLanguageAuditPaths...), "docs/LANGUAGE_DEFECT_EXHAUSTION.md")
	for _, p := range paths {
		b, err := os.ReadFile(filepath.Join(source, p))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, p, b)
	}
	path := "locales/" + locale + "/locale.json"
	var m i18n.Locale
	b, err := os.ReadFile(filepath.Join(root, path))
	if err == nil {
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
	}
	m.Locale = locale
	m.HTMLLang = locale
	m.LanguageAudit = &languageaudit.Authority{Schema: languageaudit.PolicySchema, Version: 1, Policy: languageaudit.Policy{Locale: locale, Profile: "explicit-synthetic-lifecycle", ProseScripts: []string{script}, SentenceTerminators: ".!?", ResidualMinWords: 1}}
	b, err = json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, path, b)
}
func TestLanguageAuditFormalDataGrammarUsesNativeVisibleFields(t *testing.T) {
	root := formalAuditFixture(t)
	source := []byte("- title: Read the issue\n  description: See the guide\n  url: https://go.dev/literal.example\n  thumbnail: /images/static.png\n")
	target := []byte("- title: 查看 issue。\n  description: 请看指南.\n  url: https://go.dev/literal.example\n  thumbnail: /images/static.png\n")
	d := Document{Path: "arbitrary-data.yaml", Kind: "data", Contract: UnitContract}
	r, err := scanFormalLanguageDocument(root, "zh-CN", d, source, target)
	if err != nil {
		t.Fatal(err)
	}
	if languageaudit.Blocking(r) == nil {
		t.Fatal("rendered data scalar punctuation omitted")
	}
	for _, h := range r.Diagnostics {
		if strings.Contains(h.Evidence, "static.png") || strings.Contains(h.Evidence, "literal.example") {
			t.Fatal("machine data value entered prose", h)
		}
	}
}
