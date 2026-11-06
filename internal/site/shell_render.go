package site

import (
	"bytes"
	"fmt"
	"html/template"
	"path"
	"reflect"
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

type shellStrings struct{}

func (shellStrings) HasPrefix(s, p string) bool       { return strings.HasPrefix(s, p) }
func (shellStrings) HasSuffix(s, p string) bool       { return strings.HasSuffix(s, p) }
func (shellStrings) TrimRight(s, p string) string     { return strings.TrimRight(s, p) }
func (shellStrings) Trim(s, p string) string          { return strings.Trim(s, p) }
func (shellStrings) Split(s, p string) []string       { return strings.Split(s, p) }
func (shellStrings) Join(s []string, p string) string { return strings.Join(s, p) }

type shellPaths struct{}

func (shellPaths) Dir(p string) string { return path.Dir(p) }

func (c *Content) shellFuncs(p ContentPage) template.FuncMap {
	return template.FuncMap{
		"data": func(name string) (any, error) {
			key := path.Join(path.Dir(p.Path), name)
			if strings.HasPrefix(name, "/") {
				key = path.Join("_content", name)
			}
			b, ok := c.Data[key]
			if !ok {
				return nil, fmt.Errorf("completed template data missing: %s", key)
			}
			if key == "_content/menus.yaml" || key == "_content/resources.yaml" || key == "_content/testimonials.yaml" {
				return sitecontent.ShellRuntimeData(key, b)
			}
			return sitecontent.LearnRenderData(b)
		},
		"raw":     func(s any) template.HTML { return template.HTML(fmt.Sprint(s)) },
		"strings": func() shellStrings { return shellStrings{} }, "path": func() shellPaths { return shellPaths{} },
		"googleCN": func() bool { return false },
		"add":      func(a, b int) int { return a + b }, "sub": func(a, b int) int { return a - b },
		"yaml": func(s string) (any, error) { return sitecontent.ParseShellRuntimeYAML([]byte(s)) },
		"first": func(n int, list any) (any, error) {
			v := reflect.ValueOf(list)
			if v.Kind() != reflect.Slice {
				return nil, fmt.Errorf("template data sequence required")
			}
			if n < 0 {
				return nil, fmt.Errorf("negative first")
			}
			if n > v.Len() {
				n = v.Len()
			}
			return v.Slice(0, n).Interface(), nil
		},
		"pages": func(pattern string) ([]map[string]any, error) {
			if pattern != "/solutions/*" {
				return nil, fmt.Errorf("unregistered template page context")
			}
			return append([]map[string]any(nil), c.Companies...), nil
		},
		"newest": func(rows []map[string]any) []map[string]any {
			sort.SliceStable(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["date"]) > fmt.Sprint(rows[j]["date"]) })
			return rows
		},
		"page": func(route string) map[string]any { return map[string]any{"URL": route, "title": "", "Parent": nil} },
	}
}

func (c *Content) renderCurrentShellPage(locale string, p ContentPage) ([]byte, error) {
	if p.Path == "data/site-shell/home.html" {
		return nil, fmt.Errorf("historical home cannot be current Homepage")
	}
	for _, key := range []string{"_content/site.tmpl", "_content/menus.yaml", "_content/resources.yaml", "_content/testimonials.yaml"} {
		if len(c.Data[key]) == 0 {
			return nil, fmt.Errorf("completed shared chrome incomplete: %s", key)
		}
	}
	var body []byte
	var title string
	var templated bool
	var err error
	params := map[string]any{"URL": p.Route}
	if p.Path == "_content/index.md" {
		body, params, err = sitecontent.ShellPageRenderInput(p.Content)
		if err != nil {
			return nil, err
		}
		params["URL"] = p.Route
		title, _ = params["title"].(string)
		templated = true
	} else {
		body, title, templated, err = sitecontent.PageRenderInput(p.Content)
		if err != nil {
			return nil, err
		}
		params["title"] = title
	}
	t := template.New("site.tmpl").Funcs(c.shellFuncs(p))
	if err := tmplfunc.Parse(t, string(c.Data["_content/site.tmpl"])); err != nil {
		return nil, err
	}
	if templated {
		render := t.New("page-body")
		if err := tmplfunc.Parse(render, string(body)); err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if err := render.Execute(&out, params); err != nil {
			return nil, err
		}
		body = out.Bytes()
	}
	if p.Format == "md" {
		var out bytes.Buffer
		md := goldmark.New(goldmark.WithParserOptions(parser.WithHeadingAttribute(), parser.WithAutoHeadingID()), goldmark.WithRendererOptions(gmhtml.WithUnsafe()), goldmark.WithExtensions(extension.Typographer, extension.Linkify, extension.DefinitionList, extension.Table))
		if err := md.Convert(body, &out); err != nil {
			return nil, err
		}
		body = out.Bytes()
	} else if p.Format != "html" {
		return nil, fmt.Errorf("unsupported completed page format")
	}
	if p.Path != "_content/index.md" {
		doc, err := html.Parse(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		var fragment bytes.Buffer
		var main func(*html.Node)
		main = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "main" {
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					_ = html.Render(&fragment, ch)
				}
				return
			}
			for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
				main(ch)
			}
		}
		main(doc)
		if fragment.Len() > 0 {
			body = fragment.Bytes()
		} else {
			body = append([]byte("<h1>"+template.HTMLEscapeString(title)+"</h1>"), body...)
		}
	}
	params["Content"] = template.HTML(body)
	var out bytes.Buffer
	if err := t.ExecuteTemplate(&out, "entirepage", params); err != nil {
		return nil, err
	}
	doc, err := html.Parse(bytes.NewReader(out.Bytes()))
	if err != nil {
		return nil, err
	}
	canonical, err := c.Resolver.Canonical(p.Route)
	if err != nil {
		return nil, err
	}
	var walk func(*html.Node) error
	walk = func(n *html.Node) error {
		if n.Type == html.ElementNode {
			if n.Data == "html" {
				for i := range n.Attr {
					if n.Attr[i].Key == "lang" {
						n.Attr[i].Val = locale
					}
				}
			}
			canonicalNode := false
			for _, a := range n.Attr {
				if n.Data == "link" && a.Key == "rel" && a.Val == "canonical" {
					canonicalNode = true
				}
			}
			for i, a := range n.Attr {
				if canonicalNode && a.Key == "href" {
					n.Attr[i].Val = canonical
					continue
				}
				if n.Data == "meta" && a.Key == "content" {
					for _, attr := range n.Attr {
						if attr.Key == "name" && attr.Val == "og:url" {
							n.Attr[i].Val = canonical
						}
					}
				}
				if a.Key == "href" || a.Key == "src" {
					target, err := c.Resolver.ResolveLink(p.Route, a.Val)
					if err != nil {
						return err
					}
					if strings.HasPrefix(target, c.Resolver.origin+"/") {
						target = strings.TrimPrefix(target, c.Resolver.origin)
					}
					n.Attr[i].Val = target
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if err := walk(ch); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(doc); err != nil {
		return nil, err
	}
	out.Reset()
	err = html.Render(&out, doc)
	return out.Bytes(), err
}
