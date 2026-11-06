package site

import (
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

func TestSharedChromeSiteFamiliesAndFuturePackage(t *testing.T) {
	root, _ := filepath.Abs("../..")
	c := &Content{locale: "zh-CN", Resolver: testResolver(t, "site-v2-shell", "learn-docs-v1"), Pages: map[string]ContentPage{}, Data: map[string][]byte{}}
	for _, name := range []string{"site.tmpl", "menus.yaml", "resources.yaml", "testimonials.yaml", "learn/quickstart.yaml", "learn/guided.yaml", "learn/cloud.yaml", "learn/tutorials.yaml", "learn/training.yaml", "learn/books.yaml"} {
		b, err := os.ReadFile(filepath.Join(root, "_content", name))
		if err != nil {
			t.Fatal(err)
		}
		c.Data["_content/"+name] = b
	}
	var err error
	c.Companies, err = sitecontent.FrozenHomepageCompanies(root)
	if err != nil {
		t.Fatal(err)
	}
	c.Resolver.origin = "https://go-dev.shuijingwanwq.com"
	c.Resolver.routes = map[string]bool{}
	for route, path := range map[string]string{"/": "index.md", "/learn/": "learn/index.md", "/doc/tutorial/generics": "doc/tutorial/generics.md", "/doc/tutorial/": "doc/tutorial/index.html", "/doc/security/": "doc/security/index.md"} {
		b, err := os.ReadFile(filepath.Join(root, "_content", path))
		if err != nil {
			t.Fatal(err)
		}
		c.Pages[route] = ContentPage{Route: route, Path: "_content/" + path, Format: "md", Content: b}
		c.Resolver.routes[route] = true
	}
	c.Pages["/future/"] = ContentPage{Route: "/future/", Path: "_content/future/index.md", Format: "md", Content: []byte("---\ntitle: Future package fixture\n---\n\nFuture package substantive text.\n")}
	c.Resolver.routes["/future/"] = true
	if err := c.bindBreadcrumbSources(root); err != nil {
		t.Fatal(err)
	}
	h, err := c.Compose("zh-CN", nil)
	if err != nil {
		t.Fatal(err)
	}
	for route := range c.Pages {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		b := w.Body.Bytes()
		if w.Code != 200 || bytes.Count(b, []byte(`data-project-footer="`)) != 1 || bytes.Count(b, []byte(`data-project-nav="`)) != 1 {
			t.Fatal("missing/duplicate shared chrome", route, w.Code)
		}
		if bytes.Count(b, []byte(`data-locale="`)) != len(tour.LanguageRegistry()) {
			t.Fatal("registry mismatch", route)
		}
		primary, project, legal := bytes.Index(b, []byte(`class="Footer-links"`)), bytes.Index(b, []byte(`data-project-footer=`)), bytes.Index(b, []byte(`class="Footer-bottom"`))
		if !(primary >= 0 && primary < project && project < legal) {
			t.Fatal("project footer placement", route)
		}
		for _, want := range []string{`data-project-footer-row="identity"`, `data-project-footer-row="legal"`, `href="https://go-dev.shuijingwanwq.com/"`, `href="https://beian.miit.gov.cn/"`, `© 2026 永夜`} {
			if !strings.Contains(string(b), want) {
				t.Fatal(route, want)
			}
		}
	}
}
