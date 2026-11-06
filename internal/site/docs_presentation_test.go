package site

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestFrozenDocsPresentation(t *testing.T) {
	c := &Content{Resolver: testResolver(t, "site-v2-shell", "learn-docs-v1"), Data: map[string][]byte{}, Pages: map[string]ContentPage{}}
	for _, name := range []string{"site.tmpl", "menus.yaml", "resources.yaml", "testimonials.yaml"} {
		b, err := os.ReadFile("../../_content/" + name)
		if err != nil {
			t.Fatal(err)
		}
		c.Data["_content/"+name] = b
	}
	hub, err := os.ReadFile("../../_content/doc/tutorial/index.html")
	if err != nil {
		t.Fatal(err)
	}
	c.Pages["/doc/tutorial/"] = ContentPage{Route: "/doc/tutorial/", Path: "_content/doc/tutorial/index.html", Format: "html", Content: hub}
	if err := c.bindBreadcrumbSources("../.."); err != nil {
		t.Fatal(err)
	}
	// The inherited frozen Docs default delegates to this exact shell layout.
	def, err := os.ReadFile("../../_content/doc/default.tmpl")
	if err != nil || !bytes.Contains(def, []byte("{{doclayout .}}")) {
		t.Fatal("frozen Docs layout contract", err)
	}
	for _, name := range []string{"tutorial/getting-started.html", "tutorial/handle-errors.html", "security/index.md", "database/index.md", "modules/managing-dependencies.md"} {
		t.Run(name, func(t *testing.T) {
			b, err := os.ReadFile("../../_content/doc/" + name)
			if err != nil {
				t.Fatal(err)
			}
			format := "md"
			if strings.HasSuffix(name, ".html") {
				format = "html"
			}
			route := "/doc/" + strings.TrimSuffix(strings.TrimSuffix(name, ".html"), ".md")
			if strings.HasSuffix(route, "/index") {
				route = strings.TrimSuffix(route, "index")
			}
			p := ContentPage{Path: "_content/doc/" + name, Route: route, Format: format, Content: b}
			out, err := c.renderPage("zh-CN", p)
			if err != nil {
				t.Fatal(err)
			}
			style, err := c.Resolver.ResolveLink(p.Route, "/css/styles.css")
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{`class="Doc Article"`, `class="SiteContent SiteContent--default"`, `href="` + style + `"`} {
				if !bytes.Contains(out, []byte(want)) {
					t.Fatal("missing frozen shell presentation", want)
				}
			}
			if bytes.Count(out, []byte("<h1>")) != 1 {
				t.Fatal("title duplicated or lost")
			}
			if bytes.Count(out, []byte(`id="docs-mobile-presentation"`)) != 1 || !bytes.Contains(out, []byte(docsMobileCSS)) {
				t.Fatal("missing or duplicate project-owned Docs mobile presentation")
			}
			if name == "tutorial/handle-errors.html" && !bytes.Contains(out, []byte("<pre")) {
				t.Fatal("code block lost")
			}
			if !bytes.Equal(b, p.Content) {
				t.Fatal("completed payload mutated")
			}
		})
	}
	css, err := os.ReadFile("../../_content/css/styles.css")
	if err != nil || !bytes.Contains(css, []byte(".Article pre {\n  background: var(--color-background-accented);")) {
		t.Fatal("frozen code background contract", err)
	}
	// Do not fabricate the missing layout from CSS or fall back to raw prose.
	c.Data["_content/site.tmpl"] = []byte(`{{define "entirepage"}}{{.Content}}{{end}}`)
	if _, err := c.renderPage("zh-CN", ContentPage{Path: "_content/doc/future.md", Route: "/doc/future", Format: "md", Content: []byte("---\ntitle: Fixture\n---\n\nBody\n")}); err == nil {
		t.Fatal("missing Docs layout accepted")
	}
}
