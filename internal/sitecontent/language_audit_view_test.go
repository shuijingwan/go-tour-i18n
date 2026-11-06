package sitecontent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
)

func auditFixturePolicy(locale, script string) languageaudit.Policy {
	return languageaudit.Policy{Locale: locale, Profile: "explicit-audit-fixture", ProseScripts: []string{script}, Spacing: []languageaudit.SpacingRule{{ProseScript: script, TechnicalScript: "Latin", Kinds: []languageaudit.Kind{languageaudit.Code, languageaudit.Link, languageaudit.Identity}}}, SentenceEnd: map[string]string{".": "。"}, ClosingQuotes: "\"'”’»」』)", SentenceTerminators: ".!?。！？؟", ForbiddenClausePunctuation: []string{" -- "}, HeadingSymbols: "&", ProtectedIdentities: []string{"Go"}, ResidualMinWords: 1}
}
func auditClassCount(r languageaudit.Result, class string, d languageaudit.Decision) int {
	n := 0
	for _, x := range r.Diagnostics {
		if x.Class == class && x.Decision == d {
			n++
		}
	}
	return n
}
func TestLanguageAuditParserBoundaries(t *testing.T) {
	p := auditFixturePolicy("zh-CN", "Han")
	for _, tc := range []struct {
		syntax LanguageAuditSyntax
		raw    string
	}{{AuditHTML, `<p>文字<a href="https://example.org/a.b?x=1">GOPATH</a>目录 <code>bin</code>目录</p>`}, {AuditMarkdown, "文字[GOPATH](https://example.org/a.b?x=1)目录 `bin`目录\n"}} {
		view, err := ProjectLanguageAudit("fixture", []byte(tc.raw), tc.syntax)
		if err != nil {
			t.Fatal(err)
		}
		r, err := languageaudit.Scan(p, languageaudit.Document{}, view)
		if err != nil || auditClassCount(r, "script-adjacency", languageaudit.Defect) != 3 {
			t.Fatal(tc.syntax, r, err)
		}
		for _, b := range view.Blocks {
			for _, x := range b.Pieces {
				if x.Text == "https://example.org/a.b?x=1" {
					t.Fatal("href entered visible language")
				}
			}
		}
	}
	raw := []byte(`<p>文字 <code>cmd -- flag.</code> 正常。</p><pre>文字x. -- wrong</pre><script>English sentence.</script><p aria-hidden="true">隐藏x.</p><p>LatinКириллица</p>`)
	view, err := ProjectLanguageAudit("fixture", raw, AuditHTML)
	if err != nil {
		t.Fatal(err)
	}
	r, err := languageaudit.Scan(p, languageaudit.Document{}, view)
	if err != nil || len(r.Diagnostics) != 0 {
		t.Fatal(r, err)
	}
}
func TestLanguageAuditTemplateMetadataAndSymbols(t *testing.T) {
	raw := []byte(`<html lang="en"><head><meta name="og:title" content="{{if .Title}}{{.Title}}{{end}} - The Go Programming Language"><link rel="canonical" href="https://example.org/foo"></head><body><h1>支持 &amp; 成本</h1><p>
<i>文字 <a href="fixed">Go Wiki</a>.</i>
</p><p>按 Enter 键可激活或停用下拉菜单</p></body></html>`)
	view, err := ProjectLanguageAudit("fixture", raw, AuditTemplate)
	if err != nil {
		t.Fatal(err)
	}
	r, err := languageaudit.Scan(auditFixturePolicy("zh-CN", "Han"), view, view)
	if err != nil || auditClassCount(r, "sentence-punctuation", languageaudit.Defect) != 1 || auditClassCount(r, "protected-prose-symbol", languageaudit.Candidate) != 1 {
		t.Fatal(r, err)
	}
	meta := false
	for _, d := range r.Diagnostics {
		if d.Context == "metadata" && d.Decision == languageaudit.Candidate {
			meta = true
		}
		if d.Class == "source-prose-overlap" && d.Evidence == "Title" {
			t.Fatal("template action leaked")
		}
	}
	if !meta {
		t.Fatal("social metadata not exposed")
	}
	second, err := ProjectLanguageAudit("fixture", raw, AuditTemplate)
	if err != nil || !reflect.DeepEqual(view, second) {
		t.Fatal("non-deterministic projection", err)
	}
}
func TestLanguageAuditMarkdownParagraphTopology(t *testing.T) {
	p := auditFixturePolicy("zh-CN", "Han")
	source, err := ProjectLanguageAudit("fixture", []byte("See [the guide](/guide).\n"), AuditMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	soft, err := ProjectLanguageAudit("fixture", []byte("请参阅\n[指南](/guide)。\n"), AuditMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	split, err := ProjectLanguageAudit("fixture", []byte("请参阅\n\n[指南](/guide)。\n"), AuditMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	a, err := languageaudit.Scan(p, source, soft)
	if err != nil {
		t.Fatal(err)
	}
	b, err := languageaudit.Scan(p, source, split)
	if err != nil {
		t.Fatal(err)
	}
	if auditClassCount(a, "paragraph-link-split", languageaudit.Candidate) != 0 || auditClassCount(b, "paragraph-link-split", languageaudit.Candidate) != 1 {
		t.Fatal(a, b)
	}
	if _, err := ProjectLanguageAudit("fixture", []byte("<p><code>unclosed"), AuditHTML); err == nil {
		t.Fatal("unsupported malformed markup silently projected")
	}
}

func TestLanguageAuditVisibleAttributesPanelsAndIdentityAuthority(t *testing.T) {
	raw := []byte(`<html><head><meta name="og:title" content="{{.Title}} - The Go Programming Language"></head><body>
<p hidden role="tabpanel">Command Prompt</p><span aria-hidden="true">Not language</span>
<img alt="2 &lt; 3 &amp; tooltip" src="fixed">
<p title={{.Title}} aria-label="按 Enter 键可激活或停用下拉菜单">API HTTP GOPATH schema issue Command Prompt x_y a.b</p>
</body></html>`)
	view, err := ProjectLanguageAudit("fixture", raw, AuditTemplate)
	if err != nil {
		t.Fatal(err)
	}
	r, err := languageaudit.Scan(auditFixturePolicy("zh-CN", "Han"), view, view)
	if err != nil {
		t.Fatal(err)
	}
	foundPrompt, foundAttr, foundAccess := false, false, false
	for _, b := range view.Blocks {
		for _, x := range b.Pieces {
			if x.Text == "2 < 3 & tooltip" {
				foundAttr = true
			}
			if x.Text == "按 Enter 键可激活或停用下拉菜单" {
				foundAccess = true
			}
		}
	}
	for _, d := range r.Diagnostics {
		if d.Decision == languageaudit.Defect {
			t.Fatal("invented language defect", d)
		}
		if strings.Contains(d.Evidence, "Command Prompt") {
			foundPrompt = true
		}
		if strings.Contains(d.Evidence, "HTTP") || strings.Contains(d.Evidence, "GOPATH") || strings.Contains(d.Evidence, "API") || strings.Contains(d.Evidence, "Not language") || strings.Contains(d.Evidence, "Title") {
			t.Fatal("proven namespace or hidden/action text entered residuals", d)
		}
	}
	if !foundPrompt || !foundAttr || !foundAccess {
		t.Fatal("lost visible conditional/attribute/accessibility content", foundPrompt, foundAttr, foundAccess)
	}
	for _, word := range []string{"schema", "issue", "Command", "Prompt", "x_y", "a.b"} {
		ps := auditIdentityPieces([]languageaudit.Piece{{Text: word, Kind: languageaudit.Prose}})
		if len(ps) != 1 || ps[0].Kind != languageaudit.Prose {
			t.Fatal("no positive identity authority", word, ps)
		}
	}
}

func TestLanguageAuditProvenTechnicalLinkStillHasBinding(t *testing.T) {
	p := auditFixturePolicy("zh-CN", "Han")
	source, err := ProjectLanguageAudit("fixture", []byte(`<p>See <a href="/api">HTTP</a>.</p>`), AuditHTML)
	if err != nil {
		t.Fatal(err)
	}
	target, err := ProjectLanguageAudit("fixture", []byte(`<p>请参阅</p><p><a href="/api">HTTP</a>。</p>`), AuditHTML)
	if err != nil {
		t.Fatal(err)
	}
	r, err := languageaudit.Scan(p, source, target)
	if err != nil || auditClassCount(r, "source-prose-overlap", languageaudit.Candidate) != 0 || auditClassCount(r, "paragraph-link-split", languageaudit.Candidate) != 1 {
		t.Fatal(r, err)
	}
	domain, err := ProjectLanguageAudit("fixture", []byte(`<p><a href="https://go.dev">go.dev</a> <a href="/file">go.mod</a> <a href="/code">context.Context</a> <a href="/prose">schema issue</a></p>`), AuditHTML)
	if err != nil {
		t.Fatal(err)
	}
	r, err = languageaudit.Scan(p, domain, domain)
	if err != nil || auditClassCount(r, "source-prose-overlap", languageaudit.Candidate) != 1 {
		t.Fatal(r, err)
	}
	for _, d := range r.Diagnostics {
		if d.Evidence != "schema issue" {
			t.Fatal("proven technical link residual", d)
		}
	}
}
