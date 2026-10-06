package site

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/tourpolicy"
)

func fixtureDigest(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func testResolver(t *testing.T, packages ...string) *Resolver {
	t.Helper()
	root, _ := filepath.Abs("../..")
	g, err := sitecontent.LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	l := &sitecontent.Locale{Schema: sitecontent.LocaleSchema, Locale: "zh-CN", Packages: []sitecontent.Completion{}}
	for _, id := range packages {
		for _, p := range g.Packages {
			if p.ID != id {
				continue
			}
			refs := []sitecontent.Reference{{Path: "synthetic/finalization.json", SHA256: strings.Repeat("1", 64)}}
			families := []string{"/learn/**", "/doc/tutorial/**", "/doc/database/**", "/doc/modules/**", "/doc/security/**"}
			kind := sitecontent.PackageCompletionKind
			legacy := ""
			if id == "site-v2-shell" {
				families = []string{"/", "/translation/"}
			}
			if id == "tour-v1" {
				families = []string{"/tour/**"}
				kind = "legacy-tour-closure/v1"
				legacy = "current"
			} else {
				// In-memory routing fixture follows the unpublished atomic
				// closure + integrated Surface shape; it creates no evidence.
				refs = append(refs, sitecontent.Reference{Path: "synthetic/surface.json", SHA256: strings.Repeat("3", 64)})
			}
			l.Packages = append(l.Packages, sitecontent.Completion{Package: id, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: families, EvidenceKind: kind, Evidence: refs, EvidenceIdentity: fixtureDigest(refs), ContextIdentity: strings.Repeat("2", 64), LegacySurfaceState: legacy})
		}
	}
	aliases, err := sitecontent.RedirectContracts(root, g)
	if err != nil {
		t.Fatal(err)
	}
	r, err := newResolver(g, l, "https://locale.example", aliases)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestSiteFoundationRoutingCoverageAnd404(t *testing.T) {
	r := testResolver(t, "tour-v1")
	cases := map[string]string{"/tour/basics/1": "https://locale.example/tour/basics/1", "/learn/": "https://go.dev/learn/", "/doc/security/": "https://go.dev/doc/security/", "/translation/": "https://go.dev/translation/", "https://gobyexample.com/": "https://gobyexample.com/", "http://example.org/x": "http://example.org/x", "/security/": "https://go.dev/doc/security/"}
	for in, want := range cases {
		got, err := r.Destination(in)
		if err != nil || got != want {
			t.Errorf("%s=%s %v want %s", in, got, err, want)
		}
	}
	pages := map[string]http.Handler{}
	for route := range r.routes {
		pages[route] = http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) { w.WriteHeader(200) })
	}
	h, err := r.Handler(pages)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/", "/translation/", "/learn/", "/doc/security/", "/security/", "/doc/modules/pruning", "/doc/security/vulncheck", "/unknown"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != 404 || w.Header().Get("Location") != "" {
			t.Errorf("%s: status=%d location=%s", route, w.Code, w.Header().Get("Location"))
		}
		if _, err := r.Canonical(route); err == nil {
			t.Fatal("missing local canonical")
		}
	}
	sm, _ := r.Sitemap()
	if strings.Count(string(sm), "<loc>") != 105 || strings.Contains(string(sm), "go.dev") || strings.Contains(string(sm), "/security/") || strings.Contains(string(sm), "/translation/") {
		t.Fatal("sitemap coverage")
	}
}
func TestSiteFoundationAliasesAndSitemap(t *testing.T) {
	r := testResolver(t, "tour-v1", "site-v2-shell", "learn-docs-v1")
	pages := map[string]http.Handler{}
	for route := range r.routes {
		pages[route] = http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) { w.WriteHeader(200) })
	}
	h, err := r.Handler(pages)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{"/security/": "/doc/security/", "/security/vuln/arbitrary-suffix": "/doc/security/vuln/", "/doc/modules/pruning": "/doc/modules/managing-dependencies", "/doc/security/vulncheck": "/doc/security/vuln/"}
	for alias, target := range cases {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", alias, nil))
		if w.Code != 301 || w.Header().Get("Location") != target {
			t.Errorf("alias %s: %d %s", alias, w.Code, w.Header().Get("Location"))
		}
		if _, err := r.Canonical(alias); err == nil {
			t.Fatal("alias canonical")
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/doc/security/vuln/vulncheck", nil))
	if w.Code != 404 {
		t.Fatal("external alias got local fallback")
	}
	sm, _ := r.Sitemap()
	if strings.Count(string(sm), "<loc>") != 156 {
		t.Fatalf("sitemap count: %d", strings.Count(string(sm), "<loc>"))
	}
	for _, excluded := range []string{"https://go.dev", "/security/vulncheck", "/doc/modules/pruning", ".png", ".yaml"} {
		if strings.Contains(string(sm), excluded) {
			t.Fatal("sitemap includes excluded", excluded)
		}
	}
	got, err := r.ResolveLink("/doc/tutorial/index.html", "getting-started.html#run")
	if err != nil || got != "https://locale.example/doc/tutorial/getting-started#run" {
		t.Fatal("relative source link", got, err)
	}
	got, err = r.Destination("/doc/security/vuln/vulncheck")
	if err != nil || got != "https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck" {
		t.Fatal("external redirect destination", got, err)
	}
}
func TestSiteFoundationAdvertising(t *testing.T) {
	p := AdPolicy{Unsupported: map[string]bool{"synthetic-unsupported": true, "zh-CN": true}}
	if p.ForLocale("zh-CN") != AdsUnsupported || p.ForLocale("fr-FR") != GoLocal || p.ForLocale("ja-JP") != Standard {
		t.Fatal("precedence")
	}
	if p.ProductionPreflight() == nil {
		t.Fatal("unfrozen explicit set passed")
	}
	for _, locale := range []string{"synthetic-unsupported", "zh-CN", "fr-FR", "ja-JP"} {
		for _, route := range []string{"/", "/translation/"} {
			if p.Enabled(locale, route, true) {
				t.Fatal("shell ads")
			}
		}
	}
	for _, locale := range []string{"zh-CN", "fr-FR", "de-DE", "ko-KR", "ja-JP", "sw-TZ", "kk-KZ", "fa-IR", "am-ET"} {
		neutral := AdPolicy{}
		if neutral.Enabled(locale, "/tour/basics/1", true) != tourpolicy.ForLocale(locale).TourAdsEnabled() {
			t.Fatal("legacy Tour regression")
		}
	}
	if !p.Enabled("fr-FR", "/doc/security/", true) || !p.Enabled("ja-JP", "/tour/", true) || p.Enabled("synthetic-unsupported", "/doc/security/", true) || p.Enabled("ja-JP", "/doc/security/", false) || p.Enabled("ja-JP", "/security/", true) {
		t.Fatal("route ad policy")
	}
}
