package site

import (
	"bytes"
	"fmt"
	"html/template"
	"path"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/site/tmplfunc"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
)

// PreviewBrowserPlan describes acceptance inputs, not completion evidence. The
// preview server still owns all completion/Surface gates. Source-rendered HTML
// supplies structural expectations only; it is never served as locale fallback.
type PreviewBrowserPlan struct {
	Schema   string               `json:"schema"`
	Locale   string               `json:"locale"`
	Families []string             `json:"families"`
	Routes   []string             `json:"canonical_routes"`
	Pages    []PreviewBrowserPage `json:"pages"`
}

type PreviewBrowserPage struct {
	Route  string `json:"route"`
	Family string `json:"family"`
	Format string `json:"format"`
	Source string `json:"source"`
	HTML   string `json:"html"`
	// RequiredLinks comes from explicit parser/template link semantics, not
	// incidental anchors created by the renderer's Linkify extension. Assets
	// and other structural expectations still use the unchanged rendered HTML.
	RequiredLinks []string `json:"required_links"`
}

func LoadPreviewBrowserPlan(root, locale string) (*PreviewBrowserPlan, error) {
	g, err := sitecontent.LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	b, err := sitecontent.ReadArtifact(root, "locales/"+locale+"/content-scope.json")
	if err != nil {
		return nil, err
	}
	var l sitecontent.Locale
	if err := sitecontent.StrictJSON(b, &l); err != nil {
		return nil, err
	}
	if l.Locale != locale {
		return nil, fmt.Errorf("preview plan locale mismatch")
	}
	return previewBrowserPlanFromCoverage(root, locale, g, &l)
}
func previewBrowserPlanFromCoverage(root, locale string, g *sitecontent.Global, l *sitecontent.Locale) (*PreviewBrowserPlan, error) {
	origin, err := sitecontent.LocaleOrigin(root, locale)
	if err != nil {
		return nil, err
	}
	aliases, err := sitecontent.RedirectContracts(root, g)
	if err != nil {
		return nil, err
	}
	r, err := newResolver(g, l, origin, aliases)
	if err != nil {
		return nil, err
	}
	c := &Content{locale: locale, Resolver: r, Pages: map[string]ContentPage{}, Data: map[string][]byte{}}
	if g.ShellAdoption != nil {
		c.Companies, err = sitecontent.FrozenHomepageCompanies(root)
		if err != nil {
			return nil, err
		}
	}
	for _, p := range l.Packages {
		if p.Package == "tour-v1" {
			continue
		}
		docs, raw, err := sitecontent.CurrentPackageDocuments(root, g, p.Package)
		if err != nil {
			return nil, err
		}
		for _, d := range docs {
			if d.Kind == "data" {
				c.Data[d.Path] = raw[d.Path]
				continue
			}
			if c.Pages[d.Route].Route != "" || !r.Covered(d.Route) {
				return nil, fmt.Errorf("non-exact preview page coverage: %s", d.Route)
			}
			c.Pages[d.Route] = ContentPage{Route: d.Route, Path: d.Path, Format: strings.TrimPrefix(path.Ext(d.Path), "."), Content: raw[d.Path]}
		}
	}
	if err := c.bindMarkdownSources(root); err != nil {
		return nil, err
	}
	if err := c.bindBreadcrumbSources(root); err != nil {
		return nil, err
	}
	return c.previewBrowserPlan(locale, l)
}

func routeInFamily(route, family string) bool {
	if strings.HasSuffix(family, "**") {
		return strings.HasPrefix(route, strings.TrimSuffix(family, "**"))
	}
	return route == family
}

func (c *Content) previewBrowserPlan(locale string, l *sitecontent.Locale) (*PreviewBrowserPlan, error) {
	plan := &PreviewBrowserPlan{Schema: "go-learning/preview-browser-plan/v1", Locale: locale, Families: []string{}, Routes: []string{}, Pages: []PreviewBrowserPage{}}
	families := map[string]bool{}
	for _, p := range l.Packages {
		for _, family := range p.RouteFamilies {
			families[family] = true
		}
		plan.Routes = append(plan.Routes, p.Routes...)
	}
	for family := range families {
		plan.Families = append(plan.Families, family)
	}
	sort.Strings(plan.Families)
	sort.Strings(plan.Routes)
	seenFamilies := map[string]bool{}
	for _, route := range plan.Routes {
		family := ""
		for _, f := range plan.Families {
			if routeInFamily(route, f) {
				if family != "" {
					return nil, fmt.Errorf("ambiguous route family: %s", route)
				}
				family = f
			}
		}
		if family == "" {
			return nil, fmt.Errorf("unclassified activated route: %s", route)
		}
		seenFamilies[family] = true
		if strings.HasPrefix(route, "/tour/") {
			continue
		}
		p, ok := c.Pages[route]
		if !ok {
			return nil, fmt.Errorf("missing activated Site page: %s", route)
		}
		html, err := c.renderPage(locale, p)
		if err != nil {
			return nil, err
		}
		html, err = DeliveryResources(route, html, true)
		if err != nil {
			return nil, err
		}
		links, err := c.sourceBoundPageLinks(locale, p, html)
		if err != nil {
			return nil, err
		}
		plan.Pages = append(plan.Pages, PreviewBrowserPage{Route: route, Family: family, Format: p.Format, Source: p.Path, HTML: string(html), RequiredLinks: links})
	}
	if len(seenFamilies) != len(families) {
		return nil, fmt.Errorf("activated family has no route")
	}
	return plan, nil
}

// Read the same metadata/template context as the source renderer, but use
// Goldmark's explicit Markdown/reference semantics, as structural bindings do.
// Linkify is a presentation extension and is deliberately absent here. This
// reader is plan-only: it does not change renderer or frozen language inputs.
func (c *Content) sourceBoundPageLinks(locale string, p ContentPage, rendered []byte) ([]string, error) {
	var body []byte
	var params map[string]any
	var templated bool
	var err error
	_, shell := c.Data["_content/site.tmpl"]
	if shell && p.Path == "_content/index.md" {
		body, params, err = sitecontent.ShellPageRenderInput(p.Content)
		if err != nil {
			return nil, err
		}
		params["URL"] = p.Route
		templated = true
	} else {
		var title string
		body, title, templated, err = sitecontent.PageRenderInput(p.Content)
		if err != nil {
			return nil, err
		}
		params = map[string]any{"URL": p.Route, "title": title}
	}
	// Rendering an empty page body retains the exact source metadata and the
	// real shared chrome/data links, without generating any prose autolinks.
	blank := p
	blank.Content = append(bytes.Clone(p.Content[:len(p.Content)-len(body)]), []byte("<h1>Source link contract</h1>")...)
	// This synthetic empty body supplies chrome links only; it is not a
	// localized Markdown page and has no source heading correspondence.
	chrome, err := c.renderPageLayout(locale, blank)
	if err != nil {
		return nil, err
	}
	if templated {
		funcs := c.shellFuncs(p)
		funcs["breadcrumbs"] = func(any) template.HTML { return "" }
		t := template.New("site.tmpl").Funcs(funcs)
		if shell {
			if err := tmplfunc.Parse(t, string(c.Data["_content/site.tmpl"])); err != nil {
				return nil, err
			}
		}
		page := t.New("page-body")
		if err := tmplfunc.Parse(page, string(body)); err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if err := page.Execute(&out, params); err != nil {
			return nil, err
		}
		body = out.Bytes()
	}
	if p.Format == "md" {
		md := goldmark.New(goldmark.WithParserOptions(parser.WithHeadingAttribute(), parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
			goldmark.WithExtensions(extension.Typographer, extension.DefinitionList, extension.Table))
		var out bytes.Buffer
		if err := md.Convert(body, &out); err != nil {
			return nil, err
		}
		body = out.Bytes()
	}
	links, err := explicitPageLinks(append(chrome, body...))
	if err != nil {
		return nil, err
	}
	actual, err := explicitPageLinks(rendered)
	if err != nil {
		return nil, err
	}
	served := map[string]bool{}
	normalize := func(link string) (string, error) {
		target, err := c.Resolver.ResolveLink(p.Route, link)
		if err == nil && strings.HasPrefix(target, c.Resolver.origin+"/") {
			target = strings.TrimPrefix(target, c.Resolver.origin)
		}
		return target, err
	}
	for _, link := range actual {
		target, err := normalize(link)
		if err != nil {
			return nil, err
		}
		served[target] = true
	}
	required := map[string]bool{}
	for _, link := range links {
		target, err := normalize(link)
		if err != nil {
			return nil, err
		}
		// Shell composition can discard source framing outside <main>. Only
		// explicit links actually served by the source composition are required.
		if served[target] {
			required[target] = true
		}
	}
	out := []string{}
	for link := range required {
		out = append(out, link)
	}
	sort.Strings(out)
	return out, nil
}

// Goldmark without Linkify uses the same explicit link/reference parser as
// sitecontent's structural bindings. It retains CommonMark angle autolinks,
// raw HTML anchors and evaluated template/data links, including shared chrome.
func explicitPageLinks(body []byte) ([]string, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	links := map[string]bool{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" && a.Val != "" {
					links[a.Val] = true
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	out := []string{}
	for link := range links {
		out = append(out, link)
	}
	sort.Strings(out)
	return out, nil
}
