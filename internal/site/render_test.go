package site

import (
	"encoding/xml"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestCompletedSiteComposition(t *testing.T) {
	root, _ := filepath.Abs("../..")
	origin, err := sitecontent.LocaleOrigin(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	c, err := LoadContent(root, "zh-CN", origin)
	if err != nil {
		t.Fatal(err)
	}
	checkSiteComposition(t, c, origin)
}

// Frozen English sources are format fixtures only, never runtime fallback or
// language completion evidence. This exercises every supported page/template
// even when the actual locale's historical completion is currently blocked.
func TestSiteCompositionFrozenFixture(t *testing.T) {
	root, _ := filepath.Abs("../..")
	g, err := sitecontent.LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	c := &Content{Resolver: testResolver(t, "tour-v1", "site-v2-shell", "learn-docs-v1"), Pages: map[string]ContentPage{}, Data: map[string][]byte{}}
	for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
		docs, raw, err := sitecontent.PackageDocuments(root, g, pkg)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range docs {
			if d.Kind == "data" {
				c.Data[d.Path] = raw[d.Path]
				continue
			}
			c.Pages[d.Route] = ContentPage{Route: d.Route, Path: d.Path, Format: strings.TrimPrefix(filepath.Ext(d.Path), "."), Content: raw[d.Path]}
		}
	}
	checkSiteComposition(t, c, "https://locale.example")
}

func checkSiteComposition(t *testing.T, c *Content, origin string) {
	t.Helper()
	tour := http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set("X-Tour", "retained")
		w.Write([]byte("Tour transport"))
	})
	h, err := c.Compose("zh-CN", tour)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Pages) != 51 || len(c.Data) != 7 {
		t.Fatalf("completed pages/data: %d/%d", len(c.Pages), len(c.Data))
	}
	for route, page := range c.Pages {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		body := w.Body.String()
		for _, want := range []string{`<html lang="zh-CN">`, `<link rel="canonical" href="` + strings.TrimSuffix(origin, "/") + route + `">`, "<main>"} {
			if !strings.Contains(body, want) {
				t.Fatalf("%s missing %s", route, want)
			}
		}
		if w.Code != 200 || strings.Count(body, "<main>") != 1 || strings.Contains(body, "site-home") || strings.Contains(body, "ng-app") {
			t.Fatalf("%s wrong renderer/status", route)
		}
		_, title, _, err := sitecontent.PageRenderInput(page.Content)
		if err != nil {
			t.Fatal(err)
		}
		if title != "" && !strings.Contains(body, template.HTMLEscapeString(title)) {
			t.Fatalf("%s lost completed title %s", route, title)
		}
		if route == "/learn/" && (strings.Contains(body, "{{range") || strings.Contains(body, "{{- range") || strings.Contains(body, "{{define")) {
			t.Fatal("Learn template was not executed")
		}
	}
	for _, route := range []string{"/tour/", "/tour/welcome/1", "/tour/script.js", "/tour/footer.html", "/tour/lesson/basics", "/_/compile", "/_/fmt"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Header().Get("X-Tour") != "retained" {
			t.Fatalf("Tour handler lost: %s", route)
		}
	}
	for _, route := range []string{"/unknown", "/translation/missing", "/tour/unknown", "/doc/install", "/doc/security/vuln/vulncheck"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != 404 || w.Header().Get("Location") != "" || w.Header().Get("X-Tour") != "" {
			t.Fatalf("%s not true 404: %d", route, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/security/", nil))
	if w.Code != 301 || w.Header().Get("Location") != "/doc/security/" {
		t.Fatal("alias contract lost")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/sitemap.xml", nil))
	var sitemap struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &sitemap); err != nil {
		t.Fatal(err)
	}
	var expected, actual []string
	for route := range c.Resolver.routes {
		expected = append(expected, strings.TrimSuffix(origin, "/")+route)
	}
	for _, entry := range sitemap.URLs {
		actual = append(actual, entry.Loc)
	}
	sort.Strings(expected)
	sort.Strings(actual)
	if !reflect.DeepEqual(actual, expected) || len(actual) != 156 {
		t.Fatalf("sitemap differs from current authority: %d URLs", len(actual))
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/robots.txt", nil))
	if !strings.Contains(w.Body.String(), strings.TrimSuffix(origin, "/")+"/sitemap.xml") {
		t.Fatal("robots identity")
	}
}

func TestSiteRendererFormatsAndLinks(t *testing.T) {
	c := &Content{Resolver: testResolver(t, "site-v2-shell", "learn-docs-v1"), Data: map[string][]byte{}}
	c.locale = "zh-CN"
	if _, err := c.Compose("fr-FR", nil); err == nil {
		t.Fatal("wrong completed-content locale accepted")
	}
	page := ContentPage{Route: "/doc/tutorial/getting-started", Path: "_content/doc/tutorial/getting-started.md", Format: "md", Content: []byte("---\ntitle: \"正式标题\"\n---\n\n## 标题 {#stable}\n\n**正文** [local](index.html) [English](/doc/install) [relative](../../install) [anchor](#stable)\n\n```go\nfmt.Println(\"hello\")\n```\n\n<table><tr><td>HTML</td></tr></table>\n")}
	b, err := c.renderPage("zh-CN", page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<title>正式标题</title>`, `id="stable"`, `<strong>正文</strong>`, `href="/doc/tutorial/"`, `href="https://go.dev/doc/install"`, `href="#stable"`, `<code class="language-go">`, "<table>"} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("missing rendered format/link: %s\n%s", want, b)
		}
	}
	c.Data["_content/learn/cards.yaml"] = []byte("  - title: 数据标题\n    content: <i>正式内容</i> 第一行\n      第二行\n    url: /doc/tutorial/\n    cta: 本地教程\n")
	page = ContentPage{Route: "/learn/", Path: "_content/learn/index.md", Format: "md", Content: []byte("---\ntitle: 学习\ntemplate: true\n---\n\n{{range first 1 (data \"cards.yaml\")}}<p>{{.title}} {{raw .content}} <a href=\"{{.url}}\">{{.cta}}</a></p>{{end}}\n")}
	b, err = c.renderPage("zh-CN", page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"数据标题", "<i>正式内容</i> 第一行 第二行", `href="/doc/tutorial/"`, "本地教程"} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("lost completed template data: %s", want)
		}
	}
	delete(c.Data, "_content/learn/cards.yaml")
	if _, err := c.renderPage("zh-CN", page); err == nil {
		t.Fatal("missing completed data must fail closed")
	}
	page.Format = "txt"
	page.Content = []byte("<h1>title</h1>")
	if _, err := c.renderPage("zh-CN", page); err == nil {
		t.Fatal("unknown format must fail closed")
	}
}
