package sitecontent

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
)

func shellLanguageAuditFixture(t *testing.T) (ShellPageAdapter, []byte) {
	t.Helper()
	raw := []byte("---\ntitle: The Go Programming Language\nsummary: Build services with Go.\ntemplate: true\n---\n" +
		`<p>Read <a href="https://go.dev/a.b">Go</a> and <code>fmt.Println</code> to print.</p>` + "\n\n" +
		"Read [the guide](https://go.dev/guide) with `cache`.\n\n" +
		`{{if .company}}<p title="Read {{.company}}">Read {{.company}}.</p>{{end}}` + "\n" +
		`<script>var machineValue = "English sentence.";</script>` + "\n")
	s, err := ExtractShellLanguageSource("_content/index.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	target := []byte(strings.NewReplacer(
		"The Go Programming Language", "Go 编程语言",
		"Build services with Go.", "使用 Go 构建服务。",
		"Read <a", "请看 <a", "</a> and ", "</a> 与 ", "</code> to print.", "</code> 打印内容。",
		"Read [the guide](https://go.dev/guide) with `cache`.", "请看[指南](https://go.dev/guide)，使用 `cache`。",
		"Read {{.company}}.", "请看 {{.company}}。", "Read {{.company}}", "请看 {{.company}}",
	).Replace(string(raw)))
	return ShellPageAdapter{*s, raw}, target
}

func TestShellPageLanguageAuditProjection(t *testing.T) {
	a, target := shellLanguageAuditFixture(t)
	s, localized, err := a.LanguageAuditDocuments(target)
	if err != nil {
		t.Fatal("canonical source/target metadata rejected", err)
	}
	for _, tc := range []struct {
		name           string
		d              languageaudit.Document
		title, summary string
	}{
		{"source", s, "The Go Programming Language", "Build services with Go."},
		{"target", localized, "Go 编程语言", "使用 Go 构建服务。"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.d.Owner != a.Source.Path {
				t.Fatal("lost owner", tc.d)
			}
			metadata := map[string]string{}
			kinds := map[languageaudit.Kind]bool{}
			for _, b := range tc.d.Blocks {
				var visible strings.Builder
				for _, p := range b.Pieces {
					visible.WriteString(p.Text)
					kinds[p.Kind] = true
				}
				text := visible.String()
				if b.Context == "metadata" {
					metadata[b.Locator] = text
					if b.Start < 0 || b.End <= b.Start {
						t.Fatal("missing canonical metadata slot range", b)
					}
				} else if b.Start != -1 || b.End != -1 || !strings.HasPrefix(b.Locator, "markdown-rendered/") {
					t.Fatal("rendered coordinates claimed as source coordinates", b)
				}
				for _, machine := range []string{"template", "true", ".company", "https://", "machineValue", "English sentence."} {
					if strings.Contains(text, machine) {
						t.Fatal("machine bytes entered visible prose", machine, text)
					}
				}
			}
			if !reflect.DeepEqual(metadata, map[string]string{"metadata/title": tc.title, "metadata/summary": tc.summary}) {
				t.Fatal("metadata must follow exact extractor language slots", metadata)
			}
			for _, kind := range []languageaudit.Kind{languageaudit.Code, languageaudit.Link, languageaudit.Opaque} {
				if !kinds[kind] {
					t.Fatal("markup/template boundary lost", kind)
				}
			}
		})
	}
	s2, t2, err := a.LanguageAuditDocuments(target)
	if err != nil || !reflect.DeepEqual(s, s2) || !reflect.DeepEqual(localized, t2) {
		t.Fatal("projection not deterministic", err)
	}
}

func TestShellPageLanguageAuditMetadataFailClosed(t *testing.T) {
	a, target := shellLanguageAuditFixture(t)
	for _, raw := range []string{"", "---", "---\n", "---\ntitle: Missing end\n"} {
		broken := a
		broken.Raw = []byte(raw)
		if _, _, err := broken.LanguageAuditDocuments(target); err == nil {
			t.Fatal("truncated source metadata accepted", raw)
		}
		if _, _, err := a.LanguageAuditDocuments([]byte(raw)); err == nil {
			t.Fatal("truncated target metadata accepted", raw)
		}
	}
	for name, change := range map[string][2]string{
		"unknown":   {"summary:", "unknown:"},
		"duplicate": {"summary:", "title:"},
		"malformed": {"Build services with Go.", "["},
		"machine":   {"template: true", "template: false"},
		"framing":   {"\n---\n<p>", "\n<p>"},
	} {
		t.Run(name, func(t *testing.T) {
			bad := bytes.Replace(a.Raw, []byte(change[0]), []byte(change[1]), 1)
			if bytes.Equal(a.Raw, bad) {
				t.Fatal("fixture did not change")
			}
			broken := a
			broken.Raw = bad
			if _, _, err := broken.LanguageAuditDocuments(target); err == nil {
				t.Fatal("invalid source metadata accepted")
			}
			if _, _, err := a.LanguageAuditDocuments(bad); err == nil {
				t.Fatal("invalid target metadata accepted")
			}
		})
	}
	// Shell metadata is plain runtime text; HTML-like values are not new markup.
	raw := bytes.Replace(a.Raw, []byte("Build services with Go."), []byte(`"Build <b>services</b> with Go."`), 1)
	_, view, err := a.LanguageAuditDocuments(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range view.Blocks {
		if b.Locator == "metadata/summary" {
			var text strings.Builder
			for _, p := range b.Pieces {
				text.WriteString(p.Text)
			}
			if text.String() != "Build <b>services</b> with Go." {
				t.Fatal("plain metadata interpreted as HTML", text.String())
			}
			return
		}
	}
	t.Fatal("canonical summary omitted")
}

func TestShellPageLanguageAuditCurrentValidationGate(t *testing.T) {
	root := formalAuditFixture(t)
	a, target := shellLanguageAuditFixture(t)
	u := i18n.TranslationUnit{ID: a.Source.Page.Units[0].ID, Kind: i18n.UnitKindPage, SourcePath: a.Source.Path, Source: a.Raw, SourceSHA256: digest(a.Raw)}
	c := &i18n.Catalog{Adapted: []i18n.AdaptedTranslationUnit{{Unit: u, Adapter: a}}}
	if err := i18n.ValidateTranslationUnitCandidate(root, c, u.ID, "zh-CN", target); err != nil {
		t.Fatal("valid Shell candidate rejected", err)
	}
	for _, tc := range []struct{ name, from, to string }{
		{"body", "打印内容。", "打印内容."},
		{"summary", "构建服务。", "构建服务."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := bytes.Replace(target, []byte(tc.from), []byte(tc.to), 1)
			if bytes.Equal(target, bad) {
				t.Fatal("fixture did not change")
			}
			if err := i18n.ValidateTranslationUnitCandidate(root, c, u.ID, "zh-CN", bad); err == nil || !strings.Contains(err.Error(), "sentence-punctuation") {
				t.Fatal("deterministic defect did not block current validation", err)
			}
		})
	}
	for _, tc := range []struct{ name, from, to string }{
		{"href", "https://go.dev/a.b", "https://evil.example"},
		{"identifier", "fmt.Println", "fmt.Print"},
		{"template", "{{.company}}", "{{.other}}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := bytes.Replace(target, []byte(tc.from), []byte(tc.to), 1)
			if err := i18n.ValidateTranslationUnitCandidate(root, c, u.ID, "zh-CN", bad); err == nil {
				t.Fatal("protected structure/identity drift accepted")
			}
		})
	}
}
