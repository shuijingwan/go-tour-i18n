package site

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
)

func headingFixtureHTML(t *testing.T, body string) []byte {
	t.Helper()
	md := goldmark.New(goldmark.WithParserOptions(parser.WithHeadingAttribute(), parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
		goldmark.WithExtensions(extension.Typographer, extension.Linkify, extension.DefinitionList, extension.Table))
	var out bytes.Buffer
	out.WriteString(`<html><head></head><body><header id="shell-static"></header><main>`)
	if err := md.Convert([]byte(body), &out); err != nil {
		t.Fatal(err)
	}
	out.WriteString(`</main></body></html>`)
	return out.Bytes()
}

func headingDOMIDs(t *testing.T, raw []byte) map[string]string {
	t.Helper()
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		for _, a := range n.Attr {
			if a.Key == "id" {
				ids[a.Val] = n.Data
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			visit(ch)
		}
	}
	visit(doc)
	return ids
}

func TestMarkdownSourceHeadingIdentity(t *testing.T) {
	source := "[minimize](#input-minimization) [below](#input-minimization) [external](https://clang.llvm.org/docs/SanitizerCoverage.html#inline-8bit-counters)\n\n## Input minimization\n\nBody.\n"
	target := strings.ReplaceAll(source, "Input minimization", "输入最小化")
	raw := headingFixtureHTML(t, target)
	if headingDOMIDs(t, raw)["heading"] != "h2" {
		t.Fatal("fixture did not reproduce the localized auto-ID defect")
	}
	out, err := preserveMarkdownHeadingIDs([]byte(source), []byte(target), raw)
	if err != nil {
		t.Fatal(err)
	}
	ids := headingDOMIDs(t, out)
	if ids["input-minimization"] != "h2" || ids["shell-static"] != "header" || ids["heading"] != "" {
		t.Fatal(ids)
	}
	if !bytes.Contains(out, []byte(`id="input-minimization">输入最小化</h2>`)) || bytes.Count(out, []byte(`href="#input-minimization"`)) != 2 ||
		!bytes.Contains(out, []byte(`href="https://clang.llvm.org/docs/SanitizerCoverage.html#inline-8bit-counters"`)) {
		t.Fatal(string(out))
	}
}

func TestMarkdownCanonicalHeadingVariants(t *testing.T) {
	for _, fixture := range []struct{ name, source, target string }{
		{"explicit", "## Heading {#EXACT-id}\n", "## 译名 {#EXACT-id}\n"},
		{"duplicate", "## Example\n\n## Example\n", "## 第一项\n\n## 不同项\n"},
		{"punctuation", "## What's new: Go (1.20)?\n", "## 新变化\n"},
		{"inline", "## Using `go test`, *examples* and [links](https://go.dev/)\n", "## 使用 `go test`、*示例* 和 [链接](https://go.dev/)\n"},
		{"unicode-source", "## Café: α and 日本語\n", "## 本地化标题\n"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			// Expected IDs come from the canonical parser, including duplicate
			// suffixes and inline/punctuation/Unicode behavior, never a slugger.
			want, err := canonicalMarkdownHeadings([]byte(fixture.source))
			if err != nil {
				t.Fatal(err)
			}
			out, err := preserveMarkdownHeadingIDs([]byte(fixture.source), []byte(fixture.target), headingFixtureHTML(t, fixture.target))
			if err != nil {
				t.Fatal(err)
			}
			ids := headingDOMIDs(t, out)
			for _, h := range want {
				if ids[h.ID] != "h2" || bytes.Count(out, []byte(`id="`+h.ID+`"`)) != 1 {
					t.Fatal(want, ids)
				}
			}
			if fixture.name == "explicit" && want[0].ID != "EXACT-id" {
				t.Fatal("explicit ID changed", want)
			}
			if fixture.name == "duplicate" && want[0].ID == want[1].ID {
				t.Fatal("duplicate canonical IDs", want)
			}
		})
	}
}

func TestMarkdownHeadingStructureFailsClosed(t *testing.T) {
	source := "## First\n\nParagraph.\n\n### Second\n"
	for _, target := range []string{
		"## 第一\n\nParagraph.\n",                 // count
		"## 第一\n\nParagraph.\n\n## 第二\n",        // level
		"### 第二\n\nParagraph.\n\n## 第一\n",       // hierarchy/order
		"```go\ncode\n```\n\n## 第一\n\n### 第二\n", // protected block correspondence
	} {
		if _, err := preserveMarkdownHeadingIDs([]byte(source), []byte(target), headingFixtureHTML(t, target)); err == nil {
			t.Fatal("unsafe correspondence accepted", target)
		}
	}
	for _, rendered := range []string{
		`<main><h2 id="wrong">第一</h2><h3 id="heading-1">第二</h3></main>`,
		`<main><h2 id="heading">第一</h2><h3 id="heading-1">第二</h3><div id="first"></div></main>`,
	} {
		if _, err := preserveMarkdownHeadingIDs([]byte(source), []byte("## 第一\n\nParagraph.\n\n### 第二\n"), []byte(rendered)); err == nil {
			t.Fatal("missing rendered heading or static-ID collision accepted")
		}
	}
}

func TestMarkdownHTMLAndShellIDsUnchanged(t *testing.T) {
	source := "<h2 id=\"raw-html\">Raw</h2>\n\n## Generated\n"
	target := strings.ReplaceAll(source, "Generated", "生成")
	raw := headingFixtureHTML(t, target)
	out, err := preserveMarkdownHeadingIDs([]byte(source), []byte(target), raw)
	if err != nil {
		t.Fatal(err)
	}
	ids := headingDOMIDs(t, out)
	if ids["raw-html"] != "h2" || ids["shell-static"] != "header" || ids["generated"] != "h2" {
		t.Fatal(ids)
	}
	c := &Content{markdownSources: map[string][]byte{}}
	untouched, err := c.applyMarkdownHeadingIDs(ContentPage{Format: "html"}, raw)
	if err != nil || !bytes.Equal(raw, untouched) {
		t.Fatal("HTML page modified", err)
	}
	static := []byte(`<main><h2 id="raw-only">Static</h2></main>`)
	untouched, err = preserveMarkdownHeadingIDs([]byte("<h2 id=\"raw-only\">Static</h2>\n"), []byte("<h2 id=\"raw-only\">静态</h2>\n"), static)
	if err != nil || !bytes.Equal(static, untouched) {
		t.Fatal("raw HTML ID/presentation modified", err)
	}
	if _, err := c.applyMarkdownHeadingIDs(ContentPage{Format: "md", Path: "missing.md"}, raw); err == nil {
		t.Fatal("missing source accepted")
	}
	// Real frozen version-numbers shape: the explicit pre-heading anchor and
	// canonical auto-generated heading ID already coincide in the source.
	source = "<a id=\"in-development\"></a>\n## In development\n"
	target = strings.ReplaceAll(source, "In development", "开发中")
	out, err = preserveMarkdownHeadingIDs([]byte(source), []byte(target), headingFixtureHTML(t, target))
	if err != nil || bytes.Count(out, []byte(`id="in-development"`)) != 2 || !bytes.Contains(out, []byte(`<h2 id="in-development">开发中</h2>`)) {
		t.Fatal("canonical explicit-anchor multiplicity changed", err, string(out))
	}
}

func TestFrozenMarkdownHeadingDelivery(t *testing.T) {
	c := breadcrumbFixture(t)
	c.Resolver = testResolver(t, "site-v2-shell", "learn-docs-v1")
	for _, name := range []string{"site.tmpl", "menus.yaml", "resources.yaml", "testimonials.yaml"} {
		b, err := os.ReadFile("../../_content/" + name)
		if err != nil {
			t.Fatal(err)
		}
		c.Data["_content/"+name] = b
	}
	for _, name := range []string{"security/index.md", "security/fuzz/index.md", "security/fuzz/technical.md", "security/vuln/index.md", "modules/managing-dependencies.md", "modules/version-numbers.md"} {
		b, err := os.ReadFile("../../_content/doc/" + name)
		if err != nil {
			t.Fatal(err)
		}
		route := "/doc/" + strings.TrimSuffix(name, ".md")
		route = strings.TrimSuffix(route, "index")
		p := ContentPage{Path: "_content/doc/" + name, Route: route, Format: "md", Content: b}
		c.Pages[route] = p
	}
	if err := c.bindMarkdownSources("../.."); err != nil {
		t.Fatal(err)
	}
	for _, p := range c.Pages {
		if p.Format != "md" {
			continue
		}
		t.Run(p.Path, func(t *testing.T) {
			source := append([]byte(nil), p.Content...)
			// Test-only visible-heading replacements; formal candidates untouched.
			p.Content = bytes.ReplaceAll(p.Content, []byte("## Input minimization"), []byte("## 输入最小化"))
			p.Content = bytes.ReplaceAll(p.Content, []byte("## Feedback"), []byte("## 反馈"))
			p.Content = bytes.ReplaceAll(p.Content, []byte("## In development"), []byte("## 开发中"))
			out, err := c.renderPage("zh-CN", p)
			if err != nil {
				t.Fatal(err)
			}
			ids := headingDOMIDs(t, out)
			body, err := c.headingMarkdownBody(p, source)
			if err != nil {
				t.Fatal(err)
			}
			headings, err := canonicalMarkdownHeadings(body)
			if err != nil {
				t.Fatal(err)
			}
			for _, h := range headings {
				if ids[h.ID] == "" {
					t.Fatal("source target lost", h)
				}
			}
			if !bytes.Contains(out, []byte(`class="Doc Article"`)) || !bytes.Contains(out, []byte(`class="SiteContent SiteContent--default"`)) {
				t.Fatal("Docs presentation lost")
			}
			if !bytes.Equal(c.markdownSources[p.Path], source) {
				t.Fatal("canonical source bytes mutated")
			}
		})
	}
}

func TestMarkdownFrozenSourceIntegrity(t *testing.T) {
	root := t.TempDir()
	name := "_content/doc/security/fuzz/technical.md"
	for _, file := range []string{sourcestorage.ManifestPath, name} {
		b, err := os.ReadFile(filepath.Join("../..", file))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, file)), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, file), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	c := &Content{Pages: map[string]ContentPage{"/fixture": {Path: name, Format: "md"}}}
	if err := c.bindMarkdownSources(root); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), c.markdownSources[name]...)
	if err := os.WriteFile(filepath.Join(root, name), append(before, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if err := c.bindMarkdownSources(root); err == nil {
		t.Fatal("tampered source accepted")
	}
	if reflect.DeepEqual(before, c.markdownSources[name]) {
		t.Fatal("failed binding retained unproven source")
	}
}

func TestMarkdownTemplatedHeadingAuthority(t *testing.T) {
	page := ContentPage{Path: "_content/learn/index.md", Route: "/learn/", Format: "md",
		Content: []byte("---\ntitle: Fixture\ntemplate: true\n---\n{{range data \"cards.yaml\"}}\n## {{.title}}\n{{end}}\n")}
	c := &Content{Data: map[string][]byte{"_content/learn/cards.yaml": []byte("- title: 本地标题\n  url: /learn/\n")},
		markdownSources:    map[string][]byte{page.Path: page.Content},
		markdownSourceData: map[string][]byte{"_content/learn/cards.yaml": []byte("- title: Source heading\n  url: /learn/\n")}}
	target, err := c.headingMarkdownBody(page, page.Content)
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.applyMarkdownHeadingIDs(page, headingFixtureHTML(t, string(target)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte(`id="source-heading">本地标题</h2>`)) {
		t.Fatal("template did not use canonical source data for heading identity", string(out))
	}
}

func TestMarkdownAuthorizedParagraphWrapping(t *testing.T) {
	source := "Original paragraph.\n\n## Stable heading\n\nBody.\n\n### Nested heading\n"
	target := "第一段。\n\n分行后的段落。\n\n## 本地标题\n\n正文。\n\n### 子标题\n"
	out, err := preserveMarkdownHeadingIDs([]byte(source), []byte(target), headingFixtureHTML(t, target))
	if err != nil || headingDOMIDs(t, out)["stable-heading"] != "h2" || headingDOMIDs(t, out)["nested-heading"] != "h3" {
		t.Fatal("authorized prose wrapping changed heading correspondence", err)
	}
	// Current real lifecycle fixture: protected heading IDs/levels are identical,
	// but authorized soft-wrap coordinates produce a different paragraph count.
	name := "_content/doc/database/manage-connections.md"
	frozen, err := os.ReadFile(filepath.Join("../..", name))
	if err != nil {
		t.Fatal(err)
	}
	completed, err := os.ReadFile("../../data/translation-unit-reconciliations/zh-CN/learn-docs-qc014-machine-successors/e1fb550acbf722fc3146f3acdf97ea9fc2a9bb5bfbef7742a40e0a02090d56c2.candidate")
	if err != nil {
		t.Fatal(err)
	}
	c := &Content{}
	p := ContentPage{Path: name, Route: "/doc/database/manage-connections", Format: "md"}
	so, err := c.headingMarkdownBody(p, frozen)
	if err != nil {
		t.Fatal(err)
	}
	to, err := c.headingMarkdownBody(p, completed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := preserveMarkdownHeadingIDs(so, to, headingFixtureHTML(t, string(to))); err != nil {
		t.Fatal(err)
	}
}

func TestFinalizedMarkdownHeadingFixtures(t *testing.T) {
	// Immutable finalized payloads are renderer regression fixtures, not a
	// replacement for current completion validation or the full Preview gate.
	b, err := os.ReadFile("../../data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-017/finalization.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture i18n.QualityCheckFinalization
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	c := &Content{}
	checked := 0
	for _, unit := range fixture.Units {
		u := unit.Snapshot
		if !strings.HasSuffix(u.SourcePath, ".md") {
			continue
		}
		t.Run(u.SourcePath, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("../..", u.SourcePath))
			if err != nil {
				t.Fatal(err)
			}
			target, err := os.ReadFile(filepath.Join("../..", u.CandidatePath))
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprintf("%x", sha256.Sum256(source)) != u.SourceSHA256 || fmt.Sprintf("%x", sha256.Sum256(target)) != u.CandidateSHA256 {
				t.Fatal("fixture identity changed")
			}
			so, _, templated, err := sitecontent.PageRenderInput(source)
			if err != nil {
				t.Fatal(err)
			}
			if templated {
				return // template data authority has a separate regression above
			}
			to, err := c.headingMarkdownBody(ContentPage{Path: u.SourcePath}, target)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := preserveMarkdownHeadingIDs(so, to, headingFixtureHTML(t, string(to))); err != nil {
				t.Fatal(err)
			}
			checked++
		})
	}
	if checked == 0 {
		t.Fatal("no Markdown fixtures checked")
	}
	t.Logf("finalized non-templated Markdown fixtures checked=%d", checked)
}
