package site

import (
	"bytes"
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/releasecontract"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
)

// Compose binds exact completed Site pages and Tour canonical routes to the
// shared resolver. Auxiliary Tour endpoints are transport/assets, not coverage.
// It is independent of preview/publish and does not activate a release.
func (c *Content) Compose(locale string, tourHandler http.Handler) (http.Handler, error) {
	if c.locale != "" && c.locale != locale {
		return nil, fmt.Errorf("renderer locale differs from completed content")
	}
	pages := map[string]http.Handler{}
	for route := range c.Resolver.routes {
		if strings.HasPrefix(route, "/tour/") {
			if tourHandler == nil {
				return nil, fmt.Errorf("Tour handler required")
			}
			pages[route] = tour.WithSharedChrome(tourHandler, locale, c.Resolver.origin)
			continue
		}
		p, ok := c.Pages[route]
		if !ok || p.Route != route {
			return nil, fmt.Errorf("covered route missing completed page: %s", route)
		}
		body, err := c.renderPage(locale, p)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", route, err)
		}
		body, err = DeliveryResources(route, body, true)
		if err != nil {
			return nil, err
		}
		body, err = tour.ApplySharedChrome(body, locale, c.Resolver.origin)
		if err != nil {
			return nil, err
		}
		pages[route] = http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(body)
		})
	}
	routed, err := c.Resolver.Handler(pages)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if source, ok := runtimeScriptSource(q.URL.Path); ok {
			b, exists := c.Resources["/"+source]
			if !exists {
				http.NotFound(w, q)
				return
			}
			serveRuntimeScript(w, q, c.Resolver, source, b)
			return
		}
		if b, ok := c.Resources[q.URL.Path]; ok {
			kind := mime.TypeByExtension(path.Ext(q.URL.Path))
			if kind == "" {
				kind = http.DetectContentType(b)
			}
			w.Header().Set("Content-Type", kind)
			_, _ = w.Write(b)
			return
		}
		if b, ok := c.Auxiliary[q.URL.Path]; ok {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write(b)
			return
		}
		if q.URL.Path == "/robots.txt" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", c.Resolver.origin)
			return
		}
		if c.Resolver.Covered("/tour/") && tourAuxiliary(q.URL.Path) {
			tourHandler.ServeHTTP(w, q)
			return
		}
		routed.ServeHTTP(w, q)
	}), nil
}

func tourAuxiliary(route string) bool {
	if !releasecontract.ValidRoute(route) {
		return false
	}
	return route == "/tour/script.js" || route == "/tour/footer.html" ||
		strings.HasPrefix(route, "/tour/static/") || strings.HasPrefix(route, "/tour/lesson/") ||
		route == "/_/compile" || route == "/_/fmt" || route == "/favicon.ico" || strings.HasPrefix(route, "/images/")
}

func (c *Content) renderPage(locale string, p ContentPage) ([]byte, error) {
	raw, err := c.renderPageLayout(locale, p)
	if err != nil {
		return nil, err
	}
	return c.applyMarkdownHeadingIDs(p, raw)
}

func (c *Content) renderPageLayout(locale string, p ContentPage) ([]byte, error) {
	if _, current := c.Data["_content/site.tmpl"]; current {
		raw, err := c.renderCurrentShellPage(locale, p)
		if err != nil {
			return raw, err
		}
		if p.Route == "/translation/" {
			return c.applyProjectPresentation(locale, p, raw)
		}
		if strings.HasPrefix(p.Path, "_content/doc/") {
			return c.applyDocsPresentation(p, raw)
		}
		return c.applyBreadcrumbPresentation(p, raw)
	}
	body, title, templated, err := sitecontent.PageRenderInput(p.Content)
	if err != nil {
		return nil, err
	}
	if templated {
		funcs := template.FuncMap{
			"breadcrumbs": func(any) template.HTML { return "" },
			"raw":         func(s string) template.HTML { return template.HTML(s) },
			"first": func(n int, rows []map[string]string) []map[string]string {
				if n < len(rows) {
					return rows[:n]
				}
				return rows
			},
			"data": func(name string) ([]map[string]string, error) {
				b, ok := c.Data[path.Join(path.Dir(p.Path), name)]
				if !ok {
					return nil, fmt.Errorf("completed template data missing: %s", name)
				}
				return sitecontent.LearnRenderData(b)
			},
		}
		t, err := template.New(p.Path).Funcs(funcs).Parse(string(body))
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if err := t.Execute(&out, nil); err != nil {
			return nil, err
		}
		body = out.Bytes()
	}
	switch p.Format {
	case "md":
		// Match frozen web Markdown's headings, raw HTML and block extensions.
		md := goldmark.New(goldmark.WithParserOptions(parser.WithHeadingAttribute(), parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
			goldmark.WithExtensions(extension.Typographer, extension.Linkify, extension.DefinitionList, extension.Table))
		var out bytes.Buffer
		if err := md.Convert(body, &out); err != nil {
			return nil, err
		}
		body = out.Bytes()
	case "html":
	default:
		return nil, fmt.Errorf("unsupported completed page format: %s", p.Format)
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var walk func(*html.Node) error
	mains := 0
	walk = func(n *html.Node) error {
		if n.Type == html.ElementNode {
			if n.Data == "main" {
				mains++
			}
			if title == "" && n.Data == "h1" {
				title = nodeText(n)
			}
			for i, a := range n.Attr {
				if a.Key != "href" && a.Key != "src" {
					continue
				}
				u, err := url.Parse(a.Val)
				if err != nil {
					return err
				}
				if u.Scheme == "" && u.Host == "" && u.Path == "" {
					continue
				}
				target, err := c.Resolver.ResolveLink(p.Route, a.Val)
				if err != nil {
					return err
				}
				// Local navigation stays on the serving origin (also in preview).
				if strings.HasPrefix(target, c.Resolver.origin+"/") {
					target = strings.TrimPrefix(target, c.Resolver.origin)
				}
				n.Attr[i].Val = target
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(doc); err != nil {
		return nil, err
	}
	if title == "" || mains > 1 {
		return nil, fmt.Errorf("completed page requires title and at most one main")
	}
	var fragment bytes.Buffer
	var bodyNode *html.Node
	var findBody func(*html.Node)
	findBody = func(n *html.Node) {
		if n.Data == "body" {
			bodyNode = n
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			findBody(ch)
		}
	}
	findBody(doc)
	for ch := bodyNode.FirstChild; ch != nil; ch = ch.NextSibling {
		if err := html.Render(&fragment, ch); err != nil {
			return nil, err
		}
	}
	content := fragment.String()
	if mains == 0 {
		content = "<main><h1>" + template.HTMLEscapeString(title) + "</h1>" + content + "</main>"
	}
	canonical, err := c.Resolver.Canonical(p.Route)
	if err != nil {
		return nil, err
	}
	languages := tour.LanguageRegistry()
	for i := range languages {
		languages[i].Current = languages[i].Locale == locale
	}
	t, err := template.New("site-page").Parse(sitePageTemplate)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	err = t.Execute(&out, struct {
		Locale, Title, Canonical string
		Body                     template.HTML
		Languages                []tour.LanguageLink
	}{locale, title, canonical, template.HTML(content), languages})
	return out.Bytes(), err
}

func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(nodeText(child))
	}
	return strings.TrimSpace(b.String())
}

// Chrome uses existing technical identity and registry labels only. Pending
// Homepage successor/shared-language assets are never renderer inputs.
const sitePageTemplate = `<!doctype html>
<html lang="{{.Locale}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>{{.Title}}</title><link rel="canonical" href="{{.Canonical}}">
<style>body{margin:0;font-family:system-ui,sans-serif;line-height:1.6;color:#20252b}header,main,footer{padding:1rem max(1rem,calc((100% - 70rem)/2))}header{background:#007d9c}header a{color:white}a{overflow-wrap:anywhere}pre{overflow:auto;padding:1rem;background:#f3f5f7}img{max-width:100%;height:auto}table{display:block;overflow:auto}.site-language-list{display:flex;flex-wrap:wrap;gap:.5rem 1.5rem;padding:0;list-style:none}footer{border-top:1px solid #ddd}main{overflow-wrap:anywhere}</style></head>
<body><header class="top-bar"><a href="/">Go</a></header>{{.Body}}
<footer class="site-footer"><ul class="site-language-list">{{range .Languages}}<li>{{if .Current}}<span aria-current="page">{{.EnglishName}} — {{.Autonym}}</span>{{else}}<a href="{{.URL}}">{{.EnglishName}} — {{.Autonym}}</a>{{end}}</li>{{end}}</ul></footer></body></html>
`
