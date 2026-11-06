package site

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestPreviewBrowserPlanSourceBoundLinks(t *testing.T) {
	for _, shell := range []bool{false, true} {
		for _, shape := range []string{"md", "html", "html-without-metadata", "homepage"} {
			if shape == "homepage" && !shell {
				continue
			}
			format := strings.Split(shape, "-")[0]
			if shape == "homepage" {
				format = "md"
			}
			t.Run(fmt.Sprintf("shell=%v/%s", shell, shape), func(t *testing.T) {
				c := &Content{Pages: map[string]ContentPage{}, Data: map[string][]byte{}, Resolver: &Resolver{origin: "https://locale.example", routes: map[string]bool{"/learn/": true}, canonical: map[string]bool{}, aliases: map[string]string{}}}
				if shell {
					c.Data["_content/site.tmpl"] = []byte(`{{define "entirepage"}}<html><head><title>{{.title}}</title></head><body><header><a href="/learn/">Learn</a></header><main>{{.Content}}</main></body></html>{{end}}`)
					for _, name := range []string{"menus", "resources", "testimonials"} {
						c.Data["_content/"+name+".yaml"] = []byte("[]\n")
					}
				}
				body := `Bare quoted URL: "https://example.invalid/item".

[internal](/learn/) [external](https://example.invalid/explicit)
[reference][ref] <https://example.invalid/angle>
[explicit duplicate](https://example.invalid/shared) and https://example.invalid/shared
[explicit punctuation](https://example.invalid/quoted%22) [fragment](#fixed)

## Section {#fixed}

<a href="https://example.invalid/html">HTML label</a>
{{range data "cards.yaml"}}<a href="{{.url}}">{{.title}}</a>{{end}}

![image](/image.svg)

[ref]: https://example.invalid/reference
[unused]: https://example.invalid/unused
`
				if format == "html" {
					body = `<main><h1 id="fixed">Fixture</h1><p>"https://example.invalid/item"</p><a href="#fixed">Fragment</a><a href="https://example.invalid/html">HTML label</a>{{range data "cards.yaml"}}<a href="{{.url}}">{{.title}}</a>{{end}}</main>`
				}
				c.Data["_content/learn/cards.yaml"] = []byte("- title: Data link\n  url: https://example.invalid/data\n")
				c.Pages["/learn/"] = ContentPage{Route: "/learn/", Path: "_content/learn/index." + format, Format: format, Content: []byte("---\ntitle: Fixture\ntemplate: true\n---\n\n" + body)}
				if shape == "homepage" {
					p := c.Pages["/learn/"]
					p.Path = "_content/index.md"
					p.Content = []byte("---\ntitle: Fixture\nsummary: Homepage-specific metadata\ntemplate: true\n---\n\n" + body)
					c.Pages["/learn/"] = p
					c.Data["_content/cards.yaml"] = c.Data["_content/learn/cards.yaml"]
				}
				if shape == "html-without-metadata" {
					p := c.Pages["/learn/"]
					p.Content = []byte(strings.Split(body, "{{range")[0] + "</main>")
					c.Pages["/learn/"] = p
				}
				l := &sitecontent.Locale{Packages: []sitecontent.Completion{{Routes: []string{"/learn/"}, RouteFamilies: []string{"/learn/**"}}}}
				plan, err := c.previewBrowserPlan("fr-FR", l)
				if err != nil {
					t.Fatal(err)
				}
				p := plan.Pages[0]
				want := []string{"https://example.invalid/html", "#fixed"}
				if !strings.Contains(p.HTML, `href="#fixed"`) {
					t.Fatal("same-document fragment was qualified by rendering", p.HTML)
				}
				if shape != "html-without-metadata" {
					want = append(want, "https://example.invalid/data")
				}
				if format == "md" {
					// The normal renderer still reproduces the real punctuation
					// artifact, while the required set excludes prose/unused refs.
					if !strings.Contains(p.HTML, `href="https://example.invalid/item%22"`) {
						t.Fatal("fixture did not reproduce renderer autolink artifact", p.HTML)
					}
					want = append(want, "/learn/", "https://example.invalid/explicit", "https://example.invalid/reference", "https://example.invalid/angle", "https://example.invalid/shared", "https://example.invalid/quoted%22")
				}
				for _, href := range want {
					if !slices.Contains(p.RequiredLinks, href) {
						t.Errorf("explicit link omitted: %s; got %v", href, p.RequiredLinks)
					}
				}
				for _, href := range p.RequiredLinks {
					if strings.Contains(href, "/item") || strings.Contains(href, "/unused") || strings.Contains(href, "/image.svg") {
						t.Errorf("non-link source became link authority: %s", href)
					}
				}
			})
		}
	}
}

func TestPreviewBrowserPlanRouteFamilies(t *testing.T) {
	c := &Content{Pages: map[string]ContentPage{}, Data: map[string][]byte{}, Resolver: &Resolver{origin: "https://locale.example", routes: map[string]bool{}, canonical: map[string]bool{}, aliases: map[string]string{}}}
	l := &sitecontent.Locale{Packages: []sitecontent.Completion{{Routes: []string{"/learn/", "/doc/tutorial/sample", "/tour/"}, RouteFamilies: []string{"/learn/**", "/doc/tutorial/**", "/tour/**"}}}}
	for _, route := range []string{"/learn/", "/doc/tutorial/sample"} {
		c.Resolver.routes[route] = true
		c.Pages[route] = ContentPage{Route: route, Path: "_content/sample.md", Format: "md", Content: []byte("---\ntitle: Sample\n---\n\nA source fixture with a static code block.\n\n```go\nprintln(1)\n```\n")}
	}
	plan, err := c.previewBrowserPlan("fr-FR", l)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Pages) != 2 || len(plan.Families) != 3 || len(plan.Routes) != 3 || plan.Locale != "fr-FR" {
		t.Fatalf("non-exact plan: %+v", plan)
	}
	for _, p := range plan.Pages {
		if !strings.Contains(p.HTML, "<pre><code") || !strings.Contains(p.HTML, `lang="fr-FR"`) || !strings.Contains(p.HTML, "https://locale.example"+p.Route) {
			t.Fatalf("renderer contract lost: %+v", p)
		}
	}
	for _, families := range [][]string{{"/learn/**"}, {"/**", "/learn/**"}, {"/learn/**", "/doc/tutorial/**", "/tour/**", "/unused/**"}} {
		l.Packages[0].RouteFamilies = families
		if _, err := c.previewBrowserPlan("fr-FR", l); err == nil {
			t.Fatalf("invalid families accepted: %v", families)
		}
	}
}

func TestPreviewBrowserPlanMissingPage(t *testing.T) {
	c := &Content{Pages: map[string]ContentPage{}}
	l := &sitecontent.Locale{Packages: []sitecontent.Completion{{Routes: []string{"/learn/"}, RouteFamilies: []string{"/learn/**"}}}}
	if _, err := c.previewBrowserPlan("de-DE", l); err == nil {
		t.Fatal("missing renderer accepted")
	}
}

func TestPreviewBrowserPlanHomepageRuntimeBoundary(t *testing.T) {
	// Exercise the current composition registry without starting a preview or
	// reading completion evidence. The Homepage's toy bootstrap is not routed;
	// compile/fmt continue to use the existing Tour HTTP proxy.
	c := &Content{Pages: map[string]ContentPage{"/": {Route: "/", Path: "fixture.md", Format: "md", Content: []byte("---\ntitle: Fixture\n---\n\nFixture home content.\n")}}, Data: map[string][]byte{}, Resolver: &Resolver{origin: "https://locale.example", routes: map[string]bool{"/": true, "/tour/": true}, canonical: map[string]bool{}, aliases: map[string]string{}}}
	h, err := c.Compose("fr-FR", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("X-Fixture-Tour", "true") }))
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/doc/play/hello.go", nil))
	if w.Code != 404 {
		t.Fatalf("unexpected toy route: %d", w.Code)
	}
	for _, route := range []string{"/_/compile", "/_/fmt"} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", route, nil))
		if w.Header().Get("X-Fixture-Tour") != "true" {
			t.Fatalf("proxy lost: %s", route)
		}
	}
}
