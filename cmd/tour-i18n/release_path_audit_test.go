package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/assets"
	"github.com/shuijingwan/go-tour-i18n/internal/releasecontract"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func releaseAuditScope(t *testing.T) (string, string, *site.PreviewBrowserPlan) {
	t.Helper()
	root, e := filepath.Abs(filepath.Join("..", ".."))
	if e != nil {
		t.Fatal(e)
	}
	paths, e := filepath.Glob(filepath.Join(root, "locales", "*", "content-scope.json"))
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range paths {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		var l sitecontent.Locale
		if e = json.Unmarshal(b, &l); e != nil {
			t.Fatal(e)
		}
		for _, c := range l.Packages {
			if c.Package == "tour-v1" {
				continue
			}
			plan, e := site.LoadPreviewBrowserPlan(root, l.Locale)
			if e != nil {
				t.Fatal(e)
			}
			return root, l.Locale, plan
		}
	}
	t.Fatal("no activated Site fixture; never skip release coverage")
	return "", "", nil
}

func TestReleaseAuditCurrentCoverageRuntimeAndAssets(t *testing.T) {
	root, locale, plan := releaseAuditScope(t)
	dir := t.TempDir()
	inputs, e := releaseCompilerInputs(root)
	if e != nil {
		t.Fatal(e)
	}
	a, e := site.ExportRelease(root, dir, locale, inputs)
	if e != nil {
		t.Fatal(e)
	}
	if e = checkReleaseCurrent(root, a); e != nil {
		t.Fatal(e)
	}
	if strings.Join(plan.Routes, "\n") != strings.Join(a.Routes, "\n") {
		t.Fatal("Preview/Production route drift")
	}
	tourCalls := map[string]bool{}
	tourHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tourCalls[r.URL.Path] = true
		fmt.Fprintf(w, "<html lang=%q><head><link rel=canonical href=%q></head><body ng-app=tour>Tour</body></html>", locale, a.Origin+r.URL.Path)
	})
	h, e := site.NewReleaseHandler(os.DirFS(dir), locale, tourHandler, false, strings.Repeat("a", 64))
	if e != nil {
		t.Fatal(e)
	}
	for _, route := range a.Routes {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != 200 {
			t.Fatalf("covered %s status=%d", route, w.Code)
		}
		if strings.HasPrefix(route, "/tour/") {
			if !tourCalls[route] {
				t.Fatalf("Tour renderer lost: %s", route)
			}
		} else {
			if strings.Contains(w.Body.String(), "site-home") || strings.Contains(w.Body.String(), "ng-app=") {
				t.Fatalf("wrong renderer: %s", route)
			}
			if !strings.Contains(w.Body.String(), a.Origin+route) {
				t.Fatalf("canonical missing: %s", route)
			}
		}
	}
	for _, family := range a.Families {
		route := strings.TrimRight(strings.TrimSuffix(family, "**"), "/") + "/__release_missing_route__"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != 404 || w.Header().Get("Location") != "" {
			t.Fatalf("false404: %s", route)
		}
	}
	for _, route := range []string{"/socket", "/socket/", "/home.html", "/doc/play/missing.go"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != 404 {
			t.Fatalf("uncovered %s status=%d", route, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/doc/play/hello.go", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "package main") {
		t.Fatal("Homepage toy runtime incomplete")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/sitemap.xml", nil))
	for _, r := range a.Routes {
		if !strings.Contains(w.Body.String(), a.Origin+r+"</loc>") {
			t.Fatalf("sitemap omitted %s", r)
		}
	}
	paths, e := assets.Paths(root)
	if e != nil {
		t.Fatal(e)
	}
	if len(paths) != len(a.SharedAssets) {
		t.Fatal("shared exact inventory drift")
	}
	for _, p := range paths {
		b, e := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		if e != nil || releasecontract.Digest(b) != a.SharedAssets[p] {
			t.Fatalf("asset dependency %s", p)
		}
	}
	for _, locale := range []string{"zh-CN", "fr-FR", "ja-JP"} {
		u, e := assets.URL(locale, false, "tour/static/css/app.css")
		if e != nil || !strings.HasPrefix(u, assets.BaseURL+"/") {
			t.Fatal("legacy locale asset exclusion")
		}
	}
	for p := range a.Files {
		if strings.Contains(p, "site-shell/home.html") {
			t.Fatal("retired Homepage in release")
		}
	}
	// Seal/currentness tests mutate only the disposable output tree.
	p := a.Pages["/"]
	if e = os.WriteFile(filepath.Join(dir, p), []byte("stale"), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e = releasecontract.Read(os.DirFS(dir)); e == nil {
		t.Fatal("stale rendered build accepted")
	}
}

func TestReleaseCompilerInputExactSetCurrentAndStale(t *testing.T) {
	root, e := filepath.Abs("../..")
	if e != nil {
		t.Fatal(e)
	}
	inputs, e := releaseCompilerInputs(root)
	if e != nil {
		t.Fatal(e)
	}
	if e = checkCompilerInputs(root, inputs); e != nil {
		t.Fatal(e)
	}
	for _, p := range []string{"cmd/tour-production/main.go", "cmd/tour-i18n/publish.go", "internal/site/release.go"} {
		original := inputs[p]
		delete(inputs, p)
		if checkCompilerInputs(root, inputs) == nil {
			t.Fatalf("new/missing compiler source accepted: %s", p)
		}
		inputs[p] = strings.Repeat("a", 64)
		if checkCompilerInputs(root, inputs) == nil {
			t.Fatalf("changed compiler source accepted: %s", p)
		}
		inputs[p] = original
	}
}

func TestReleaseSharedHostnameUsesCurrentProductionIdentity(t *testing.T) {
	root, _ := filepath.Abs("../..")
	if e := checkSharedProductionTarget(root, assets.BaseURL); e != nil {
		t.Fatal(e)
	}
	if checkSharedProductionTarget(root, "https://stale-assets.example") == nil {
		t.Fatal("consumer/producer hostname drift accepted")
	}
}

func TestReleaseAuditSearchRejectsUnboundCoverageBeforePOST(t *testing.T) {
	const origin = "https://locale.example"
	posts := 0
	client := indexNowTestClient(func(r *http.Request) (int, string) {
		if r.Method == "POST" {
			posts++
			return 200, ""
		}
		if strings.HasSuffix(r.URL.Path, ".txt") {
			return 200, "index-now-test-key"
		}
		return 200, testIndexNowSitemap(origin+"/", 1)
	})
	profile := indexNowProfile{Locale: "fr-FR", State: "live", Hostname: "locale.example", PublicURL: origin + "/"}
	for _, expected := range [][]string{nil, {origin + "/", origin + "/learn/"}} {
		_, e := bootstrapIndexNow(context.Background(), client, "https://api.indexnow.org/indexnow", profile, "index-now-test-key", expected)
		if e == nil {
			t.Fatal("stale public sitemap submitted")
		}
	}
	if posts != 0 {
		t.Fatal("search mutation before current exact-set proof")
	}
}
