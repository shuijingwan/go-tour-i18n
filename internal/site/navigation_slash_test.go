package site

import (
	"bytes"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func trailingSlashNavigationResolver() *Resolver {
	return &Resolver{
		origin: "https://locale.example",
		routes: map[string]bool{
			"/doc/": true, "/doc/install": true, "/learn/": true,
			"/both": true, "/both/": true, "/explicit/": true,
		},
		canonical: map[string]bool{
			"/doc/": true, "/doc/install": true, "/learn/": true,
			"/global-only/": true,
		},
		aliases: map[string]string{
			"/explicit": "/uncovered", "/legacy/": "/doc/",
			"/alias-target": "/doc", "/external": "https://external.example/fixed",
		},
	}
}

func TestDestinationCoveredTrailingSlashForms(t *testing.T) {
	r := trailingSlashNavigationResolver()
	pages := map[string]http.Handler{}
	for route := range r.routes {
		pages[route] = http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) { w.WriteHeader(http.StatusOK) })
	}
	h, err := r.Handler(pages)
	if err != nil {
		t.Fatal(err)
	}
	// The six exact navigation instances in Integrated Surface findings 006/007.
	cases := []struct{ input, want string }{
		{"/doc/install/", "https://locale.example/doc/install"},
		{"/learn#guided-learning-journeys", "https://locale.example/learn/#guided-learning-journeys"},
		{"/learn#online-learning", "https://locale.example/learn/#online-learning"},
		{"/learn#featured-books", "https://locale.example/learn/#featured-books"},
		{"/learn#self-paced-labs", "https://locale.example/learn/#self-paced-labs"},
		{"/doc", "https://locale.example/doc/"},
		{"/learn?q=a%2Fb&tag=one&tag=two#online%20learning", "https://locale.example/learn/?q=a%2Fb&tag=one&tag=two#online%20learning"},
		{"/doc/install/?download=1#instructions", "https://locale.example/doc/install?download=1#instructions"},
		{"/doc/", "https://locale.example/doc/"},
		{"/doc/install", "https://locale.example/doc/install"},
		{"/learn/", "https://locale.example/learn/"},
		{"/both", "https://locale.example/both"},
		{"/both/", "https://locale.example/both/"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := r.Destination(tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("Destination(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
			}
			returned, err := url.Parse(got)
			if err != nil || !r.Covered(returned.Path) {
				t.Fatalf("returned URL must name an exact covered canonical path: %q, %v", got, err)
			}
			// Use the actual returned URL, not the input or a separately selected
			// route: a local host with the old slash spelling would still be 404.
			// Browsers send query parameters but do not send the fragment.
			returned.Fragment, returned.RawFragment = "", ""
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, returned.String(), nil))
			if w.Code != http.StatusOK || w.Header().Get("Location") != "" {
				t.Fatalf("returned URL %q does not reach its canonical handler directly: status=%d Location=%q", got, w.Code, w.Header().Get("Location"))
			}
		})
	}
	got, err := r.ResolveLink("/", "learn?from=home#guided-learning-journeys")
	if err != nil || got != "https://locale.example/learn/?from=home#guided-learning-journeys" {
		t.Fatalf("relative navigation = %q, %v", got, err)
	}
}

func TestDestinationTrailingSlashCoverageAndAliasBoundaries(t *testing.T) {
	r := trailingSlashNavigationResolver()
	for _, tc := range []struct{ input, want string }{
		{"/global-only?q=1#section", "https://go.dev/global-only?q=1#section"},
		{"/unknown/", "https://go.dev/unknown/"},
		{"/doc/install/extra/", "https://go.dev/doc/install/extra/"},
		{"/", "https://go.dev/"},
		// An explicit alias wins over a locally covered slash alternative.
		{"/explicit?x=1#section", "https://go.dev/uncovered?x=1#section"},
		{"/legacy/child?x=1#section", "https://locale.example/doc/?x=1#section"},
		// Do not apply the new slash rule to an explicit alias's target.
		{"/alias-target", "https://go.dev/doc"},
		{"/external?x=1#section", "https://external.example/fixed"},
		{"https://outside.example/learn#section", "https://outside.example/learn#section"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := r.Destination(tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("Destination(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
			}
		})
	}
	for _, invalid := range []string{"/doc//", "/doc/../learn", "/doc/install//"} {
		if got, err := r.Destination(invalid); err == nil {
			t.Errorf("invalid path %q resolved to %q", invalid, got)
		}
	}
}

func TestDestinationTrailingSlashDoesNotExpandHTTPOrCanonicalCoverage(t *testing.T) {
	r := trailingSlashNavigationResolver()
	beforeRoutes, beforeCanonical, beforeAliases := maps.Clone(r.routes), maps.Clone(r.canonical), maps.Clone(r.aliases)
	beforeSitemap, err := r.Sitemap()
	if err != nil {
		t.Fatal(err)
	}
	pages := map[string]http.Handler{}
	for route := range r.routes {
		pages[route] = http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) { w.WriteHeader(http.StatusOK) })
	}
	for _, route := range []string{"/doc", "/doc/install/", "/learn"} {
		if _, err := r.Destination(route); err != nil {
			t.Fatal(err)
		}
		if r.Covered(route) {
			t.Errorf("navigation registered coverage for %s", route)
		}
		if _, err := r.Canonical(route); err == nil {
			t.Errorf("navigation registered canonical for %s", route)
		}
	}
	h, err := r.Handler(pages)
	if err != nil {
		t.Fatal(err)
	}
	for route, want := range map[string]int{
		"/doc": 404, "/doc/install/": 404, "/learn": 404,
		"/global-only": 404, "/global-only/": 404,
		"/doc/": 200, "/doc/install": 200, "/learn/": 200,
		"/legacy/child": 301,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != want || (want == 404 && w.Header().Get("Location") != "") || (want == 301 && w.Header().Get("Location") != "/doc/") {
			t.Errorf("HTTP %s = %d, Location=%q; want %d", route, w.Code, w.Header().Get("Location"), want)
		}
	}
	// Handler registration must still reject a navigation-only slash variant.
	pages["/learn"] = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	if _, err := r.Handler(pages); err == nil {
		t.Error("handler accepted uncovered slash variant")
	}
	afterSitemap, err := r.Sitemap()
	if err != nil || !bytes.Equal(beforeSitemap, afterSitemap) || !reflect.DeepEqual(beforeRoutes, r.routes) || !reflect.DeepEqual(beforeCanonical, r.canonical) || !reflect.DeepEqual(beforeAliases, r.aliases) {
		t.Fatal("navigation changed coverage, canonical authority, aliases or sitemap", err)
	}
}
