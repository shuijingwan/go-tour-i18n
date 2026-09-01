// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	website "github.com/shuijingwan/go-tour-i18n"
)

func TestPrerenderRoutesReuseSitemapLessonDataForEveryLocale(t *testing.T) {
	for _, test := range []struct {
		locale string
		origin string
	}{
		{"zh-CN", "https://go-dev.shuijingwanwq.com"},
		{"ja-JP", "https://ja-go-dev.shuijingwanwq.com"},
		{"ko-KR", "https://ko-go-dev.shuijingwanwq.com"},
	} {
		t.Run(test.locale, func(t *testing.T) {
			source, err := NewPrerenderSource(website.TourOnly(), test.locale)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(source.Routes); got != 103 {
				t.Fatalf("course routes=%d, want 103", got)
			}
			seen := map[string]bool{}
			for _, route := range source.Routes {
				if seen[route.Path] {
					t.Fatalf("duplicate course route %q", route.Path)
				}
				seen[route.Path] = true
				if route.Canonical != test.origin+route.Path {
					t.Fatalf("canonical=%q, want %q", route.Canonical, test.origin+route.Path)
				}
			}
			if !seen["/tour/welcome/2"] || !seen["/tour/basics/1"] {
				t.Fatalf("representative routes are missing")
			}
		})
	}
}

func TestLoadPrerenderedPagesFailsClosed(t *testing.T) {
	route := CourseRoute{
		Path:        "/tour/basics/1",
		Canonical:   "https://example.test/tour/basics/1",
		PageTitle:   "Packages",
		LessonTitle: "Basics",
		Files:       []string{"package main"},
	}
	if _, err := loadPrerenderedPages(fstest.MapFS{}, []CourseRoute{route}); err == nil || !strings.Contains(err.Error(), "read prerendered page") {
		t.Fatalf("missing prerender error=%v", err)
	}
	invalid := fstest.MapFS{
		"tour/prerender/basics/1.html": {Data: []byte("<!doctype html><title>same SPA shell</title>")},
	}
	if _, err := loadPrerenderedPages(invalid, []CourseRoute{route}); err == nil || !strings.Contains(err.Error(), "missing runtime head marker") {
		t.Fatalf("invalid prerender error=%v", err)
	}
}

func TestValidatePrerenderedPageRequiresExactFormalDescription(t *testing.T) {
	route := CourseRoute{Path: "/tour/basics/1", Canonical: "https://example.test/tour/basics/1", PageTitle: "Packages", Description: "the formal description"}
	html := []byte(`<!doctype html><html data-tour-rendered-route="/tour/basics/1"><head>` + runtimeHeadMarker + `<title>Packages</title><link rel="canonical" href="https://example.test/tour/basics/1"><meta name="description" content="different"></head><body><div id="editor-container"></div></body></html>`)
	if err := validatePrerenderedPage(html, route); err == nil || !strings.Contains(err.Error(), "want formal metadata") {
		t.Fatalf("mismatched description error=%v", err)
	}
}

func TestValidatePrerenderedPageRequiresDefaultTextareaSourceOnly(t *testing.T) {
	route := CourseRoute{
		Path:        "/tour/basics/1",
		Canonical:   "https://example.test/tour/basics/1",
		PageTitle:   "Packages",
		Description: "the formal description",
		Files:       []string{"package main\n", "package second\n"},
	}
	base := `<!doctype html><html data-tour-rendered-route="/tour/basics/1"><head>` + runtimeHeadMarker + `<title>Packages</title><link rel="canonical" href="https://example.test/tour/basics/1"><meta name="description" content="the formal description"></head><body><div id="editor-container"><textarea ui-codemirror>` + route.Files[0] + `</textarea></div></body></html>`
	if err := validatePrerenderedPage([]byte(base), route); err != nil {
		t.Fatalf("valid default textarea source: %v", err)
	}
	withHiddenSource := strings.Replace(base, "</body>", `<pre hidden data-tour-prerender-source="example-2.go">`+route.Files[1]+`</pre></body>`, 1)
	if err := validatePrerenderedPage([]byte(withHiddenSource), route); err == nil || !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("hidden source error=%v", err)
	}
	wrongDefault := strings.Replace(base, route.Files[0], route.Files[1], 1)
	if err := validatePrerenderedPage([]byte(wrongDefault), route); err == nil || !strings.Contains(err.Error(), "default lesson file") {
		t.Fatalf("wrong textarea error=%v", err)
	}
}

func TestProductionHandlerRequiresCompletePrerenderTree(t *testing.T) {
	t.Cleanup(func() {
		if err := useContent(website.TourOnly()); err != nil {
			t.Errorf("restore embedded content: %v", err)
		}
	})
	contentDir := filepath.Join(t.TempDir(), "_content")
	if err := os.CopyFS(contentDir, website.TourOnly()); err != nil {
		t.Fatal(err)
	}
	metadata := SiteMetadata{
		Locale:             "zh-CN",
		PublishedAt:        "2026-08-25T00:00:00Z",
		UpstreamCommit:     FrozenUpstreamCommit,
		UpstreamCommitTime: FrozenUpstreamCommitTime,
		Pages:              103,
		Articles:           7,
	}
	if err := WriteSiteMetadata(contentDir, metadata); err != nil {
		t.Fatal(err)
	}
	if _, err := NewProductionHandler(os.DirFS(contentDir), "zh-CN"); err == nil || !strings.Contains(err.Error(), "read prerendered page") {
		t.Fatalf("production handler missing-prerender error=%v", err)
	}
}

func TestPrerenderedPageHandlerInjectsRuntimeHeadWithoutChangingBody(t *testing.T) {
	originalAnalytics, originalAd := analyticsHTML, adHTML
	t.Cleanup(func() { analyticsHTML, adHTML = originalAnalytics, originalAd })
	analyticsHTML = `<script data-runtime="analytics"></script>`
	adHTML = `<script data-runtime="ad"></script>`
	page := prerenderedPage{
		route: "/tour/basics/1",
		html:  []byte("<!doctype html><head>" + runtimeHeadMarker + "<title>Packages</title></head><body>body</body>"),
	}
	mux := http.NewServeMux()
	registerPrerenderedPages(mux, []prerenderedPage{page})
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, page.route, nil))
	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("response=%d %q", recorder.Code, recorder.Header().Get("Content-Type"))
	}
	body := recorder.Body.String()
	for _, want := range []string{`data-runtime="analytics"`, `data-runtime="ad"`, "<body>body</body>"} {
		if strings.Count(body, want) != 1 {
			t.Errorf("response does not contain exactly one %q: %s", want, body)
		}
	}
}

func TestPrerenderedPagesServeDifferentRouteSpecificInitialHTML(t *testing.T) {
	pages := []prerenderedPage{
		{
			route: "/tour/welcome/2",
			html:  []byte(`<!doctype html><title>本地安装 — Go 语言之旅</title><link rel="canonical" href="https://go-dev.shuijingwanwq.com/tour/welcome/2"><meta name="description" content="本地安装正文"><body><h2>本地安装</h2><p>本地安装正文</p></body>`),
		},
		{
			route: "/tour/basics/1",
			html:  []byte(`<!doctype html><title>包 — Go 语言之旅</title><link rel="canonical" href="https://go-dev.shuijingwanwq.com/tour/basics/1"><meta name="description" content="每个 Go 程序都由包构成"><body><h2>包</h2><p>每个 Go 程序都由包构成</p><pre>package main</pre></body>`),
		},
	}
	mux := http.NewServeMux()
	registerPrerenderedPages(mux, pages)
	responses := make([]string, len(pages))
	for i, page := range pages {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, page.route, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s: status=%d", page.route, recorder.Code)
		}
		responses[i] = recorder.Body.String()
		if !strings.Contains(responses[i], page.route) {
			t.Errorf("GET %s is missing its canonical", page.route)
		}
	}
	if responses[0] == responses[1] {
		t.Fatal("two course routes returned identical initial HTML")
	}
	if strings.Contains(responses[0], "package main") || !strings.Contains(responses[1], "package main") {
		t.Fatal("no-example and example initial HTML are not distinct")
	}
}

func TestPrerenderPathMatchesBundleLayout(t *testing.T) {
	got, err := prerenderPath("/tour/moretypes/12")
	if err != nil {
		t.Fatal(err)
	}
	if got != "tour/prerender/moretypes/12.html" {
		t.Fatalf("path=%q", got)
	}
	for _, route := range []string{"/tour/list", "/tour/../x/1", "/tour/a/b/c"} {
		if _, err := prerenderPath(route); err == nil {
			t.Errorf("accepted invalid route %q", route)
		}
	}
}
