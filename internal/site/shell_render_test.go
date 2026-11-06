package site

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestSuccessorHomepageRuntimeComposition(t *testing.T) {
	root, _ := filepath.Abs("../..")
	for _, locale := range []string{"zh-CN", "fr-FR"} {
		t.Run(locale, func(t *testing.T) {
			c := &Content{locale: locale, Resolver: testResolver(t, "site-v2-shell", "learn-docs-v1", "tour-v1"), Pages: map[string]ContentPage{}, Data: map[string][]byte{}}
			for _, name := range []string{"_content/site.tmpl", "_content/menus.yaml", "_content/resources.yaml", "_content/testimonials.yaml", "_content/learn/training.yaml"} {
				b, err := os.ReadFile(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				c.Data[name] = b
			}
			body, err := os.ReadFile(filepath.Join(root, "_content/index.md"))
			if err != nil {
				t.Fatal(err)
			}
			// Marker stands for test-only completed bytes. No locale target is written.
			body = []byte(strings.Replace(string(body), "Build simple, secure, scalable systems with Go", "SYNTHETIC COMPLETED HOMEPAGE", 1))
			p := ContentPage{Route: "/", Path: "_content/index.md", Format: "md", Content: body}
			c.Pages["/"] = p
			c.Companies, err = sitecontent.FrozenHomepageCompanies(root)
			if err != nil {
				t.Fatal(err)
			}
			b, err := c.renderPage(locale, p)
			if err != nil {
				t.Fatal(err)
			}
			html := string(b)
			for _, want := range []string{"SYNTHETIC COMPLETED HOMEPAGE", "Hero-gridContainer", "WhoUsesCaseStudyList", "TestimonialsGo", "WhyGo", "GettingStartedGo", "Site-header", "Site-footer", `lang="` + locale + `"`, `href="https://locale.example/"`, `href="https://go.dev/dl"`} {
				if !strings.Contains(html, want) {
					t.Fatalf("missing %s", want)
				}
			}
			if strings.Contains(html, "<header class=\"top-bar\">") || strings.Contains(html, "Go Learning &amp; Documentation Translations") {
				t.Fatal("historical simplified homepage rendered")
			}
			if strings.Contains(html, "{{range") || strings.Contains(html, "{{- range") {
				t.Fatal("template not executed")
			}
			if strings.Count(html, `<main `) != 1 {
				t.Fatal("shared chrome main framing")
			}
			if _, err := c.renderPage(locale, ContentPage{Path: "data/site-shell/home.html", Route: "/", Format: "html", Content: []byte("<main>old home</main>")}); err == nil {
				t.Fatal("arbitrary old home accepted")
			}
			for _, key := range []string{"_content/site.tmpl", "_content/menus.yaml", "_content/resources.yaml", "_content/testimonials.yaml"} {
				before := c.Data[key]
				delete(c.Data, key)
				if key == "_content/site.tmpl" {
					_, err = c.renderCurrentShellPage(locale, p)
				} else {
					_, err = c.renderPage(locale, p)
				}
				c.Data[key] = before
				if err == nil {
					t.Fatal("missing chrome accepted", key)
				}
			}
			// True 404 and independent Tour handler remain resolver contracts.
			// A smaller covered resolver keeps this focused on routing composition.
			c.Resolver.routes = map[string]bool{"/": true, "/tour/": true}
			c.Pages = map[string]ContentPage{"/": p}
			h, err := c.Compose(locale, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("X-Tour", "independent") }))
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/not-covered", nil))
			if w.Code != 404 {
				t.Fatal("coverage expanded")
			}
			w = httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/tour/", nil))
			if w.Header().Get("X-Tour") != "independent" {
				t.Fatal("Tour renderer lost")
			}
		})
	}
}
