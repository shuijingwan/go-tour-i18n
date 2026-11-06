package site

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/site/tmplfunc"
	"html/template"
)

func breadcrumbFixture(t *testing.T) *Content {
	t.Helper()
	c := &Content{Resolver: &Resolver{origin: "https://locale.example", routes: map[string]bool{"/": true, "/doc/tutorial/": true}, canonical: map[string]bool{}, aliases: map[string]string{}}, Pages: map[string]ContentPage{}, Data: map[string][]byte{}}
	c.Pages["/"] = ContentPage{Route: "/", Content: []byte("---\ntitle: Localized root fixture\n---\n")}
	c.Pages["/doc/tutorial/"] = ContentPage{Route: "/doc/tutorial/", Path: "_content/doc/tutorial/index.html", Content: []byte("<!--{\"Title\":\"LOCALIZED TUTORIAL HUB\",\"Breadcrumb\":true}-->\nBody")}
	if err := c.bindBreadcrumbSources("../.."); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestBreadcrumbAuthorityAndFutureCoverage(t *testing.T) {
	c := breadcrumbFixture(t)
	local, err := c.breadcrumbPage("/doc/tutorial")
	if err != nil {
		t.Fatal(err)
	}
	if local["title"] != "LOCALIZED TUTORIAL HUB" || local["URL"] != "/doc/tutorial/" {
		t.Fatal(local)
	}
	source, err := c.breadcrumbPage("/doc")
	if err != nil {
		t.Fatal(err)
	}
	if source["title"] != "Documentation" || source["URL"] != "/doc" {
		t.Fatal(source)
	}
	url, err := c.Resolver.ResolveLink("/doc/tutorial/example", source["URL"].(string))
	if err != nil || url != "https://go.dev/doc" {
		t.Fatal(url, err)
	}
	root, err := c.breadcrumbPage("/")
	if err != nil {
		t.Fatal(err)
	}
	if root["title"] != "The Go Programming Language" {
		t.Fatal(root)
	}
	// A future completed parent takes over both label and exact canonical href;
	// source metadata must never mask a missing/invalid completed projection.
	c.Resolver.routes["/doc/"] = true
	c.Pages["/doc/"] = ContentPage{Route: "/doc/", Content: []byte("---\ntitle: FUTURE LOCAL DOCS\nbreadcrumbTitle: LOCAL DOCS NAV\n---\n")}
	future, err := c.breadcrumbPage("/doc")
	if err != nil {
		t.Fatal(err)
	}
	if future["title"] != "FUTURE LOCAL DOCS" || future["breadcrumbTitle"] != "LOCAL DOCS NAV" || future["URL"] != "/doc/" {
		t.Fatal(future)
	}
	url, err = c.Resolver.ResolveLink("/doc/tutorial/example", future["URL"].(string))
	if err != nil || url != "https://locale.example/doc/" {
		t.Fatal(url, err)
	}
}

func TestBreadcrumbMetadataFailsClosed(t *testing.T) {
	for _, state := range []string{"missing source", "empty source", "missing completed", "empty completed", "missing root"} {
		t.Run(state, func(t *testing.T) {
			c := breadcrumbFixture(t)
			route := "/doc"
			switch state {
			case "missing source":
				delete(c.breadcrumbSources, route)
			case "empty source":
				c.breadcrumbSources[route] = ContentPage{Route: route, Content: []byte("---\ntitle: ' '\n---\n")}
			case "missing completed":
				delete(c.Pages, "/doc/tutorial/")
				route = "/doc/tutorial"
			case "empty completed":
				p := c.Pages["/doc/tutorial/"]
				p.Content = []byte("<!--{\"Title\":\"  \\t\"}-->\n")
				c.Pages["/doc/tutorial/"] = p
				route = "/doc/tutorial"
			case "missing root":
				delete(c.breadcrumbSources, "/")
				route = "/"
			}
			if _, err := c.breadcrumbPage(route); err == nil {
				t.Fatal("invalid ancestor accepted", state)
			}
		})
	}
}

func TestFrozenTutorialBreadcrumbTemplates(t *testing.T) {
	c := breadcrumbFixture(t)
	raw, err := os.ReadFile("../../_content/site.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"getting-started", "handle-errors"} {
		t.Run(name, func(t *testing.T) {
			b, err := os.ReadFile("../../_content/doc/tutorial/" + name + ".html")
			if err != nil {
				t.Fatal(err)
			}
			p := ContentPage{Route: "/doc/tutorial/" + name, Content: b}
			m, err := pageMetadata(p)
			if err != nil {
				t.Fatal(err)
			}
			m["title"] = "LOCALIZED CURRENT TITLE"
			funcs := c.shellFuncs(p)
			funcs["page"] = c.breadcrumbPage
			tmpl := template.New("breadcrumb-fixture").Funcs(funcs)
			if err := tmplfunc.Parse(tmpl, string(raw)); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := tmpl.ExecuteTemplate(&out, "breadcrumbs", m); err != nil {
				t.Fatal(err)
			}
			text := out.String()
			for _, want := range []string{`href="/doc"`, `href="/doc/tutorial/"`, "Documentation", "LOCALIZED TUTORIAL HUB", "LOCALIZED CURRENT TITLE"} {
				if !strings.Contains(text, want) {
					t.Fatal(want, text)
				}
			}
			if strings.Contains(text, `href="/"`) || strings.Contains(text, "Localized root fixture") {
				t.Fatal("root exclusion lost", text)
			}
			if strings.Count(text, "<li ") != 3 {
				t.Fatal("ancestor hierarchy lost", text)
			}
		})
	}
}

func TestBreadcrumbFrozenSourceIntegrity(t *testing.T) {
	c := breadcrumbFixture(t)
	root := t.TempDir()
	for _, name := range []string{"data/site-source-manifest.json", "_content/index.md", "_content/doc/index.html"} {
		b, err := os.ReadFile(filepath.Join("../..", name))
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.bindBreadcrumbSources(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_content/doc/index.html"), []byte("tampered metadata"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := c.bindBreadcrumbSources(root); err == nil || !strings.Contains(err.Error(), "adopted frozen source changed") {
		t.Fatal("unverified source metadata accepted", err)
	}
}

func TestDeclaredBreadcrumbCallsAndSiteAdapter(t *testing.T) {
	c := breadcrumbFixture(t)
	for _, body := range []string{`{{breadcrumbs .}}`, `{{- breadcrumbs . -}}`, `{{if true}}{{breadcrumbs .}}{{end}}`} {
		p := ContentPage{Route: "/learn/", Path: "_content/learn/index.md", Content: []byte("---\ntitle: COMPLETED LEARN\nbreadcrumbTitle: LEARN NAV\ntemplate: true\n---\n" + body)}
		m, err := pageMetadata(p)
		if err != nil {
			t.Fatal(err)
		}
		declared, err := declaresBreadcrumbs(p, m)
		if err != nil || !declared {
			t.Fatal(body, declared, err)
		}
		b, err := os.ReadFile("../../_content/site.tmpl")
		if err != nil {
			t.Fatal(err)
		}
		c.Data["_content/site.tmpl"] = b
		out, err := c.applyBreadcrumbPresentation(p, []byte(`<html><body><main><ol class="SiteBreadcrumb"><li><a href="/"> </a></li></ol></main></body></html>`))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(out, []byte("LEARN NAV")) || bytes.Contains(out, []byte(`href="/"`)) {
			t.Fatal(string(out))
		}
	}
	p := ContentPage{Content: []byte("---\ntitle: Fixture\ntemplate: true\n---\n{{/* breadcrumbs */}}{{printf \"breadcrumbs\"}}")}
	m, _ := pageMetadata(p)
	if declared, err := declaresBreadcrumbs(p, m); err != nil || declared {
		t.Fatal("non-call treated as dependency", declared, err)
	}
}
