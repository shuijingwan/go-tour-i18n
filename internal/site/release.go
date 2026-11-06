package site

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/assets"
	"github.com/shuijingwan/go-tour-i18n/internal/releasecontract"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
	"golang.org/x/net/html"
)

// HomepageAuxiliary extracts only the frozen toy files actually selected by
// the completed Homepage. These are runtime inputs, never canonical coverage.
func HomepageAuxiliary(root string, c *Content) (map[string][]byte, error) {
	result := map[string][]byte{}
	p, ok := c.Pages["/"]
	if !ok {
		return result, nil
	}
	b, e := c.renderPage(c.locale, p)
	if e != nil {
		return nil, e
	}
	doc, e := html.Parse(strings.NewReader(string(b)))
	if e != nil {
		return nil, e
	}
	var walk func(*html.Node, bool) error
	walk = func(n *html.Node, toys bool) error {
		if n.Type == html.ElementNode && n.Data == "select" {
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(" "+a.Val+" ", " js-playgroundToysEl ") {
					toys = true
				}
			}
		}
		if toys && n.Type == html.ElementNode && n.Data == "option" {
			for _, a := range n.Attr {
				if a.Key == "value" {
					if !releasecontract.SafePath(a.Val) || path.Base(a.Val) != a.Val || path.Ext(a.Val) != ".go" {
						return fmt.Errorf("unsafe Homepage toy")
					}
					p := "doc/play/" + a.Val
					b, e := sitecontent.ReadArtifact(root, "_content/"+p)
					if e != nil {
						return e
					}
					result["/"+p] = b
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if e := walk(ch, toys); e != nil {
				return e
			}
		}
		return nil
	}
	if e = walk(doc, false); e != nil {
		return nil, e
	}
	return result, nil
}

// ExportRelease renders the same validated completed Content used by Preview,
// into the existing _content tree. It writes no repository language/evidence.
func ExportRelease(root, contentDir, locale string, inputs map[string]string) (*releasecontract.Artifact, error) {
	origin, e := sitecontent.LocaleOrigin(root, locale)
	if e != nil {
		return nil, e
	}
	g, l, files, e := sitecontent.LocalCoverageFiles(root, locale)
	if e != nil {
		return nil, e
	}
	c, e := contentFromCoverage(root, locale, origin, g, l, files)
	if e != nil {
		return nil, e
	}
	if len(l.Packages) == 0 {
		return nil, fmt.Errorf("no activated coverage for release")
	}
	a := &releasecontract.Artifact{Schema: releasecontract.Schema, Locale: locale, Origin: strings.TrimSuffix(origin, "/"), CoverageIdentity: releasecontract.Identity(l), Routes: []string{}, Families: []string{}, Aliases: map[string]string{}, Pages: map[string]string{}, Inputs: inputs, Files: map[string]string{}, AssetPolicy: releasecontract.AssetPolicy, AssetOrigin: assets.BaseURL, SharedAssets: map[string]string{}}
	if a.Inputs == nil {
		a.Inputs = map[string]string{}
	}
	bind := func(p string) error {
		b, e := sitecontent.ReadArtifact(root, p)
		if e != nil {
			return e
		}
		a.Inputs[p] = releasecontract.Digest(b)
		return nil
	}
	if e = bind(sitecontent.GlobalPath); e != nil {
		return nil, e
	}
	if e = bind(sitecontent.LocalePath(locale)); e != nil {
		return nil, e
	}
	families := map[string]bool{}
	for _, completion := range l.Packages {
		a.Routes = append(a.Routes, completion.Routes...)
		for _, f := range completion.RouteFamilies {
			families[f] = true
		}
		for _, ref := range completion.Evidence {
			if e = bind(ref.Path); e != nil {
				return nil, e
			}
		}
		if completion.Package == "tour-v1" {
			continue
		}
		docs, _, e := sitecontent.CurrentPackageDocuments(root, g, completion.Package)
		if e != nil {
			return nil, e
		}
		for _, d := range docs {
			if e = bind(d.Path); e != nil {
				return nil, e
			}
		}
	}
	sort.Strings(a.Routes)
	for f := range families {
		a.Families = append(a.Families, f)
	}
	sort.Strings(a.Families)
	for from, to := range c.Resolver.aliases {
		if c.Resolver.Covered(to) {
			a.Aliases[from] = to
		}
	}
	write := func(p string, b []byte) error {
		if !releasecontract.SafePath(p) {
			return fmt.Errorf("unsafe release output")
		}
		target := filepath.Join(contentDir, filepath.FromSlash(p))
		if e := os.MkdirAll(filepath.Dir(target), 0755); e != nil {
			return e
		}
		return os.WriteFile(target, b, 0644)
	}
	for route, page := range c.Pages {
		if page.Path == "data/site-shell/home.html" {
			return nil, fmt.Errorf("retired Homepage cannot enter release")
		}
		b, e := c.renderPage(locale, page)
		if e != nil {
			return nil, e
		}
		b, e = DeliveryResources(route, b)
		if e != nil {
			return nil, e
		}
		b, e = tour.ApplySharedChrome(b, locale, c.Resolver.origin)
		if e != nil {
			return nil, e
		}
		p := "site/pages/" + releasecontract.Digest([]byte(route)) + ".html"
		if e = write(p, b); e != nil {
			return nil, e
		}
		a.Pages[route] = p
		if strings.HasPrefix(page.Path, "_content/") {
			if e = write(strings.TrimPrefix(page.Path, "_content/"), page.Content); e != nil {
				return nil, e
			}
		}
	}
	for p, b := range c.Data {
		if !strings.HasPrefix(p, "_content/") {
			return nil, fmt.Errorf("unregistered structured output")
		}
		if e = write(strings.TrimPrefix(p, "_content/"), b); e != nil {
			return nil, e
		}
	}
	for p, b := range c.Auxiliary {
		if e = bind("_content" + p); e != nil {
			return nil, e
		}
		if e = write(strings.TrimPrefix(p, "/"), b); e != nil {
			return nil, e
		}
	}
	paths, e := assets.Paths(root)
	if e != nil {
		return nil, e
	}
	for _, p := range paths {
		b, e := releasecontract.ReadRegular(root, "_content/"+p)
		if e != nil {
			return nil, e
		}
		a.SharedAssets[p] = releasecontract.Digest(b)
		if e = bind("_content/" + p); e != nil {
			return nil, e
		}
		if e = write(p, b); e != nil {
			return nil, e
		}
	}
	a.SharedAssetsIdentity = releasecontract.Identity(a.SharedAssets)
	plan, e := previewBrowserPlanFromCoverage(root, locale, g, l)
	if e != nil {
		return nil, e
	}
	// Production adapts only resource delivery. Routes, structural expectations
	// and executable interactions stay on the carried Preview plan authority.
	for i := range plan.Pages {
		body, e := DeliveryResources(plan.Pages[i].Route, []byte(plan.Pages[i].HTML))
		if e != nil {
			return nil, e
		}
		plan.Pages[i].HTML = string(body)
	}
	b, e := json.Marshal(plan)
	if e != nil {
		return nil, e
	}
	if e = write(releasecontract.BrowserPlanPath, append(b, '\n')); e != nil {
		return nil, e
	}
	// The publication timestamp and Chrome prerender are derived publish output,
	// outside the immutable build payload. All source content stays byte-bound.
	e = filepath.WalkDir(contentDir, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if p == contentDir || d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("unsupported build file")
		}
		rel, e := filepath.Rel(contentDir, p)
		if e != nil {
			return e
		}
		rel = filepath.ToSlash(rel)
		if rel == releasecontract.Path || rel == "tour/site-metadata.json" || strings.HasPrefix(rel, "tour/prerender/") {
			return nil
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		a.Files[rel] = releasecontract.Digest(b)
		return nil
	})
	if e != nil {
		return nil, e
	}
	if e = releasecontract.Write(contentDir, a); e != nil {
		return nil, e
	}
	return a, nil
}

// NewReleaseHandler never opens a repository or falls through to a Tour SPA.
// All content comes from the publish artifact verified at startup.
func NewReleaseHandler(content fs.FS, locale string, tourHandler http.Handler, runtimeHead bool, bundleIdentity ...string) (http.Handler, error) {
	a, e := releasecontract.Read(content)
	if e != nil {
		return nil, e
	}
	if a.Locale != locale {
		return nil, fmt.Errorf("release locale mismatch")
	}
	if e = tour.CheckReleaseCanonicalOrigin(locale, a.Origin); e != nil {
		return nil, e
	}
	r := &Resolver{origin: a.Origin, routes: map[string]bool{}, canonical: map[string]bool{}, aliases: a.Aliases}
	pages := map[string]http.Handler{}
	for _, route := range a.Routes {
		r.routes[route] = true
		r.canonical[route] = true
		if strings.HasPrefix(route, "/tour/") {
			if tourHandler == nil {
				return nil, fmt.Errorf("Tour handler required")
			}
			pages[route] = tour.WithSharedChrome(tourHandler, locale, a.Origin)
			continue
		}
		b, e := fs.ReadFile(content, a.Pages[route])
		if e != nil {
			return nil, e
		}
		if runtimeHead {
			head := os.Getenv("TOUR_ANALYTICS")
			if (AdPolicy{}).Enabled(locale, route, true) {
				head += os.Getenv("TOUR_AD_HTML")
			}
			b = []byte(strings.Replace(string(b), "</head>", head+"</head>", 1))
		}
		pages[route] = http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(b)
		})
	}
	h, e := r.Handler(pages)
	if e != nil {
		return nil, e
	}
	return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if source, ok := runtimeScriptSource(q.URL.Path); ok {
			// Only sealed shared inputs may feed a derived runtime script.
			if a.SharedAssets[source] == "" || a.Files[source] != a.SharedAssets[source] {
				http.NotFound(w, q)
				return
			}
			b, err := fs.ReadFile(content, source)
			if err != nil || releasecontract.Digest(b) != a.SharedAssets[source] {
				http.NotFound(w, q)
				return
			}
			serveRuntimeScript(w, q, r, source, b)
			return
		}
		if q.URL.Path == "/robots.txt" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", a.Origin)
			return
		}
		if q.URL.Path == "/site-release.json" {
			w.Header().Set("Content-Type", "application/json")
			input := ""
			if len(bundleIdentity) == 1 {
				input = bundleIdentity[0]
			}
			_ = json.NewEncoder(w).Encode(struct {
				Identity string `json:"identity"`
				Coverage string `json:"coverage_identity"`
				Assets   string `json:"shared_assets_identity"`
				Input    string `json:"input_identity"`
			}{a.Identity, a.CoverageIdentity, a.SharedAssetsIdentity, input})
			return
		}
		p := strings.TrimPrefix(q.URL.Path, "/")
		if strings.HasPrefix(p, "doc/play/") && a.Files[p] != "" && path.Ext(p) == ".go" {
			b, e := fs.ReadFile(content, p)
			if e != nil {
				http.NotFound(w, q)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write(b)
			return
		}
		if r.Covered("/tour/") && tourAuxiliary(q.URL.Path) {
			tourHandler.ServeHTTP(w, q)
			return
		}
		h.ServeHTTP(w, q)
	}), nil
}
