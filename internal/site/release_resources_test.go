package site

import (
	"github.com/shuijingwan/go-tour-i18n/internal/assets"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewCompositionOmitsFrozenGTM(t *testing.T) {
	root, _ := filepath.Abs("../..")
	c := &Content{Resolver: &Resolver{origin: "https://locale.example", routes: map[string]bool{"/": true, "/translation/": true}, canonical: map[string]bool{}, aliases: map[string]string{}}, Pages: map[string]ContentPage{}, Data: map[string][]byte{}}
	for _, name := range []string{"site.tmpl", "menus.yaml", "resources.yaml", "testimonials.yaml"} {
		b, err := os.ReadFile(filepath.Join(root, "_content", name))
		if err != nil {
			t.Fatal(err)
		}
		c.Data["_content/"+name] = b
	}
	for _, route := range []string{"/", "/translation/"} {
		c.Pages[route] = ContentPage{Route: route, Path: "fixture.md", Format: "md", Content: []byte("---\ntitle: Fixture\n---\n\nMeaningful Site preview fixture.\n")}
	}
	for _, locale := range []string{"zh-CN", "fr-FR"} {
		h, err := c.Compose(locale, nil)
		if err != nil {
			t.Fatal(err)
		}
		for route, page := range c.Pages {
			raw, err := c.renderPage(locale, page)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(raw), "'gtm.start'") != 2 || !strings.Contains(string(raw), "/ns.html?") {
				t.Fatal("fixture did not reproduce the real frozen shell telemetry")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
			if w.Code != 200 || strings.Contains(w.Body.String(), "googletagmanager") || strings.Contains(w.Body.String(), "gtm.start") {
				t.Fatal("Preview leaked source GTM", locale, route, w.Code)
			}
			if !strings.Contains(w.Body.String(), `src="/_/site-runtime/js/site.js"`) {
				t.Fatal("download runtime delivery fix lost")
			}
			production, err := DeliveryResources(route, raw)
			if err != nil || strings.Count(string(production), "'gtm.start'") != 2 || !strings.Contains(string(production), "/ns.html?") {
				t.Fatal("Production source analytics contract changed", err)
			}
		}
	}
}

func TestPreviewTelemetryRemovalIsSourceBound(t *testing.T) {
	// Similar domains, ordinary scripts and advertising hooks are not source
	// GTM blocks. Delivery leaves them to the existing browser/ads contracts.
	const outside = `<script src="https://www.googletagmanager.com/unexpected.js"></script><script src="https://example.invalid/required.js"></script><script data-runtime="project-analytics">projectAnalytics()</script><div class="course-ad"></div>`
	got, err := DeliveryResources("/", []byte(outside), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"unexpected.js", "required.js", "projectAnalytics()", `class="course-ad"`} {
		if !strings.Contains(string(got), want) {
			t.Fatal("non-source telemetry or functional/advertising content removed", want)
		}
	}
	root, _ := filepath.Abs("../..")
	frozen, err := os.ReadFile(filepath.Join(root, "_content/site.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	_, tail, opened := strings.Cut(string(frozen), "<script")
	code, _, closed := strings.Cut(tail, "</script>")
	if !opened || !closed {
		t.Fatal("missing frozen bootstrap fixture")
	}
	bootstrap := "<script" + code + "</script>"
	combined := strings.Replace(bootstrap, "</script>", "functionalRuntime();</script>", 1)
	got, err = DeliveryResources("/", []byte(combined), true)
	if err != nil || !strings.Contains(string(got), "functionalRuntime()") || !strings.Contains(string(got), "gtm.start") {
		t.Fatal("script with additional functional code silently removed", err)
	}
	combined = strings.Replace(bootstrap, "<script", `<script onload="functionalRuntime()"`, 1)
	got, err = DeliveryResources("/", []byte(combined), true)
	if err != nil || !strings.Contains(string(got), "functionalRuntime()") {
		t.Fatal("script with functional attributes silently removed", err)
	}
	for _, body := range []string{
		`<!-- Google Tag Manager --><script src="https://example.invalid/required.js"></script><!-- End Google Tag Manager -->`,
		`<!-- Google Tag Manager --><script>functionalRuntime()</script><!-- End Google Tag Manager -->`,
		`<!-- Google Tag Manager -->`,
		`<!-- Google Tag Manager (noscript) --><noscript><iframe src="https://www.googletagmanager.com/functional.html"></iframe></noscript><!-- End Google Tag Manager (noscript) -->`,
	} {
		if _, err := DeliveryResources("/", []byte(body), true); err == nil {
			t.Fatal("unknown or malformed marked content silently removed")
		}
	}
}

func TestReleaseDeliveryResourcesSharedPolicyAcrossSiteFamilies(t *testing.T) {
	for _, route := range []string{"/", "/translation/", "/learn/", "/doc/tutorial/", "/doc/security/"} {
		body := []byte(`<html><head><link rel="stylesheet" href="https://go.dev/css/styles.css?v=current"></head><body><img src="/images/go-logo-white.svg"><script src="https://go.dev/js/index.js"></script><a href="https://go.dev/doc/not-local">external navigation</a></body></html>`)
		got, e := DeliveryResources(route, body)
		if e != nil {
			t.Fatal(e)
		}
		for _, p := range []string{"css/styles.css?v=current", "images/go-logo-white.svg", "js/index.js"} {
			if !strings.Contains(string(got), assets.BaseURL+"/"+p) {
				t.Fatalf("%s resource omitted: %s", route, p)
			}
		}
		if !strings.Contains(string(got), `href="https://go.dev/doc/not-local"`) {
			t.Fatal("navigation fallback became runtime fallback")
		}
	}
}

func TestReleasePreviewResourcesUseFrozenLocalBytes(t *testing.T) {
	body := []byte(`<html><link rel="stylesheet" href="https://go.dev/css/styles.css"><img src="` + assets.BaseURL + `/images/go-logo-white.svg"></html>`)
	got, e := DeliveryResources("/", body, true)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(got), assets.BaseURL) || strings.Contains(string(got), "https://go.dev/css/") {
		t.Fatal("Preview requires undeployed production assets")
	}
	c := &Content{locale: "test", Resolver: &Resolver{origin: "https://locale.example", routes: map[string]bool{"/tour/list": true}, canonical: map[string]bool{"/tour/list": true}, aliases: map[string]string{}}, Resources: map[string][]byte{"/css/styles.css": []byte("frozen exact CSS")}}
	h, e := c.Compose("test", http.NotFoundHandler())
	if e != nil {
		t.Fatal(e)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/css/styles.css", nil))
	if w.Code != 200 || w.Body.String() != "frozen exact CSS" || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/css") {
		t.Fatal("Preview static current payload missing")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/css/missing.css", nil))
	if w.Code != 404 {
		t.Fatal("missing resource fallback")
	}
}

func TestReleaseAuxiliaryNeverDelegatesUncoveredTraversal(t *testing.T) {
	for _, p := range []string{"/tour/static/../../doc/missing", "/tour/lesson/../missing", "/images/../doc/missing"} {
		if tourAuxiliary(p) {
			t.Fatalf("unsafe route delegated to legacy mux: %s", p)
		}
	}
	for _, p := range []string{"/tour/static/js/app.js", "/tour/lesson/basics", "/_/compile", "/_/fmt"} {
		if !tourAuxiliary(p) {
			t.Fatalf("Tour auxiliary lost: %s", p)
		}
	}
}

func TestReleaseResourceQueryAndFragmentRemainExact(t *testing.T) {
	for _, dev := range []bool{false, true} {
		got, e := DeliveryResources("/", []byte(`<img src="https://go.dev/images/go-logo-white.svg?v=current#mark">`), dev)
		if e != nil || !strings.Contains(string(got), "images/go-logo-white.svg?v=current#mark") {
			t.Fatal("resource identity lost query/fragment")
		}
	}
}
