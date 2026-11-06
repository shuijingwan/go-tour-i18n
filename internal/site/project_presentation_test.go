package site

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

func TestTranslationProjectPresentation(t *testing.T) {
	body, err := os.ReadFile("../../data/site-shell/translation.html")
	if err != nil {
		t.Fatal(err)
	}
	c := &Content{Resolver: &Resolver{origin: "https://locale.example", routes: map[string]bool{"/": true, "/translation/": true, "/tour/": true, "/tour/welcome/1": true, "/doc/tutorial/": true, "/doc/tutorial/test": true, "/doc/modules/developing": true}}}
	p := ContentPage{Route: "/translation/", Path: "data/site-shell/translation.html", Content: body}
	p.Format = "html"
	c.Data = map[string][]byte{}
	for _, path := range []string{"_content/site.tmpl", "_content/menus.yaml", "_content/resources.yaml", "_content/testimonials.yaml"} {
		b, err := os.ReadFile("../../" + path)
		if err != nil {
			t.Fatal(err)
		}
		c.Data[path] = b
	}
	rendered, err := c.renderPage("zh-CN", p)
	if err != nil || bytes.Count(rendered, []byte(`data-project-section="`)) != 9 {
		t.Fatal("actual current-shell composition", err)
	}
	for _, locale := range []string{"zh-CN", "fr-FR"} {
		raw := append([]byte(`<html><head><link rel="stylesheet" href="/site.css"><meta name="viewport" content="width=device-width, initial-scale=1"></head><body><header class="Site-header"><div class="Header-rightContent"></div></header>`), body...)
		raw = append(raw, []byte(`<footer class="Site-footer"><div class="Footer-links"></div><div class="Footer-bottom"></div></footer></body></html>`)...)
		result, err := c.applyProjectPresentation(locale, p, raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range []string{"introduction", "coverage", "status", "languages", "links", "validation", "workflow", "support", "learning"} {
			if bytes.Count(result, []byte(`data-project-section="`+id+`"`)) != 1 {
				t.Fatal("missing/duplicate", id)
			}
		}
		if bytes.Contains(result, []byte(`href="/doc/modules/"`)) || bytes.Contains(result, []byte("PayPal")) {
			t.Fatal("uncovered hub or missing payment authority invented")
		}
		if !bytes.Contains(result, []byte(`href="/doc/modules/developing"`)) {
			t.Fatal("coverage child missing")
		}
		if bytes.Count(result, []byte(`aria-current="page"`)) != 1 {
			t.Fatal("current registry marker")
		}
		// All five accepted paragraphs survive byte-for-byte in the derived DOM.
		for _, part := range strings.Split(string(body), "<p>")[1:] {
			text := strings.SplitN(part, "</p>", 2)[0]
			if bytes.Count(result, []byte(text)) != 1 {
				t.Fatal("accepted prose changed", text)
			}
		}
		composed, err := tour.ApplySharedChrome(result, locale, c.Resolver.origin)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Count(composed, []byte(`data-project-footer="`)) != 1 || !bytes.Contains(composed, []byte(`class="Site-footer"`)) || !bytes.Contains(composed, []byte(`class="Site-header"`)) {
			t.Fatal("shared upstream chrome lost")
		}
		// Keep the real composition available for the focused browser fixture.
		if dir := os.Getenv("GO_TOUR_PROJECT_PRESENTATION_FIXTURE_DIR"); dir != "" && locale == "zh-CN" {
			if err := os.WriteFile(dir+"/project.html", composed, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := c.applyProjectPresentation("zh-CN", p, []byte("<html><head></head><body><main><h1>changed</h1></main></body></html>")); err == nil {
		t.Fatal("unknown source shape accepted")
	}
	for _, id := range []string{"source-baseline", "applications", "regions", "independent-qc", "revision", "closure", "surface", "activation", "preview-human", "production", "maintenance"} {
		raw := []byte(`<html><head></head><body><main><h1>FORMAL SUCCESSOR</h1><section data-project-section="` + id + `"><h2>FORMAL SECTION</h2><p>ACCEPTED SUCCESSOR TEXT</p></section></main></body></html>`)
		got, err := c.applyProjectPresentation("zh-CN", p, raw)
		if err != nil || !bytes.Contains(got, []byte("ACCEPTED SUCCESSOR TEXT")) {
			t.Fatal(id, err)
		}
	}
}

func TestProjectCoverageDirectoryExactSet(t *testing.T) {
	r := &Resolver{routes: map[string]bool{"/": true, "/doc/modules/developing": true, "/doc/install": false, "/tour/welcome/1": true}}
	groups := projectDirectory(r)
	found := map[string]bool{}
	last := ""
	for _, g := range groups {
		if g.Path <= last {
			t.Fatal("unstable groups")
		}
		last = g.Path
		if g.Path == "/doc/modules/" && g.Covered {
			t.Fatal("prefix fabricated root")
		}
		for _, route := range g.Routes {
			if !r.Covered(route) || found[route] {
				t.Fatal(route)
			}
			found[route] = true
		}
	}
	if len(found) != 3 {
		t.Fatal(found)
	}
}
