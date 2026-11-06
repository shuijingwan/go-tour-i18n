package main

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestFullLocalePreviewSiteComposition(t *testing.T) {
	root, catalog := publishTestCatalog(t)
	projection, err := i18n.BuildLocaleProjection(root, catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
	if err != nil {
		t.Fatal(err)
	}
	h, err := newFullLocalePreviewHandler(root, projection.ContentDir, projection.Locale)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/", "/translation/", "/learn/", "/doc/tutorial/getting-started", "/tour/", "/tour/basics/1"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		body := w.Body.String()
		if w.Code != 200 || !strings.Contains(body, `lang="zh-CN"`) {
			t.Fatalf("%s status/lang", route)
		}
		isTour := strings.HasPrefix(route, "/tour/")
		if strings.Contains(body, `ng-app="tour"`) != isTour {
			t.Fatalf("%s wrong renderer", route)
		}
		if !isTour && (strings.Contains(body, "site-home") || strings.Count(body, "<main>") != 1 || !strings.Contains(body, `href="https://go-dev.shuijingwanwq.com`+route+`"`)) {
			t.Fatalf("%s wrong Site identity", route)
		}
	}
	for _, route := range []string{"/missing", "/translation/missing", "/tour/missing", "/tour/basics/999", "/socket", "/socket/", "/tour/lesson/missing"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != 404 || w.Header().Get("Location") != "" {
			t.Fatalf("%s should be true 404: %d", route, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/tour/script.js", nil))
	for _, want := range []string{"window.transport = HTTPTransport();", `window.playgroundBaseURL = "";`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("lost Tour transport: %s", want)
		}
	}
	for _, route := range []string{"/_/compile", "/_/fmt"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != 405 {
			t.Fatalf("transport %s status=%d", route, w.Code)
		}
	}
}
