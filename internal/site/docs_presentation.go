package site

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/site/tmplfunc"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"golang.org/x/net/html"
)

//go:embed docs_mobile.css
var docsMobileCSS string

// applyDocsPresentation restores the inherited frozen doc/default.tmpl layout
// using site.tmpl's own doclayout. It is a delivery adapter, not a new source or
// language reconstruction contract. Only completed bytes supply visible text.
func (c *Content) applyDocsPresentation(p ContentPage, raw []byte) ([]byte, error) {
	body, _, _, err := sitecontent.PageRenderInput(p.Content)
	if err != nil {
		return nil, err
	}
	params, err := pageMetadata(p)
	if err != nil {
		return nil, err
	}
	if params["breadcrumb"] == true {
		label, err := breadcrumbLabel(params)
		if err != nil {
			return nil, err
		}
		params["breadcrumbTitle"] = label
	}
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	find := func(root *html.Node, predicate func(*html.Node) bool) *html.Node {
		var visit func(*html.Node) *html.Node
		visit = func(n *html.Node) *html.Node {
			if predicate(n) {
				return n
			}
			for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
				if v := visit(ch); v != nil {
					return v
				}
			}
			return nil
		}
		return visit(root)
	}
	main := find(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "main" })
	if main == nil {
		return nil, fmt.Errorf("Docs shell main missing")
	}
	// The base renderer supplies an h1 only when the input has no main. The
	// frozen doclayout supplies that same completed title exactly once.
	source, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if find(source, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "main" }) == nil {
		h1 := main.FirstChild
		for h1 != nil && h1.Type == html.TextNode && strings.TrimSpace(h1.Data) == "" {
			h1 = h1.NextSibling
		}
		if h1 == nil || h1.Type != html.ElementNode || h1.Data != "h1" {
			return nil, fmt.Errorf("Docs shell title boundary missing")
		}
		main.RemoveChild(h1)
	}
	var fragment bytes.Buffer
	for ch := main.FirstChild; ch != nil; ch = ch.NextSibling {
		if err := html.Render(&fragment, ch); err != nil {
			return nil, err
		}
	}
	params["Content"] = template.HTML(fragment.String())
	funcs := c.shellFuncs(p)
	funcs["page"] = c.breadcrumbPage
	t := template.New("docs-delivery").Funcs(funcs)
	if err := tmplfunc.Parse(t, string(c.Data["_content/site.tmpl"])); err != nil {
		return nil, err
	}
	if t.Lookup("doclayout") == nil {
		return nil, fmt.Errorf("completed shell missing Docs layout")
	}
	fragment.Reset()
	if err := t.ExecuteTemplate(&fragment, "doclayout", params); err != nil {
		return nil, err
	}
	part, err := html.Parse(&fragment)
	if err != nil {
		return nil, err
	}
	article := find(part, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "article" })
	if article == nil {
		return nil, fmt.Errorf("Docs layout article missing")
	}
	// Layout-generated navigation follows the same coverage authority as body links.
	if err := c.rewritePresentationLinks(p, article); err != nil {
		return nil, err
	}
	article.Parent.RemoveChild(article)
	for main.FirstChild != nil {
		main.RemoveChild(main.FirstChild)
	}
	main.AppendChild(article)
	// Project-owned presentation follows the frozen stylesheet. Both Preview
	// and publication use this adapter; no frozen resource bytes are rewritten.
	head := find(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "head" })
	if head == nil || find(doc, func(n *html.Node) bool {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == "docs-mobile-presentation" {
				return true
			}
		}
		return false
	}) != nil {
		return nil, fmt.Errorf("Docs presentation style boundary missing or duplicated")
	}
	style := &html.Node{Type: html.ElementNode, Data: "style", Attr: []html.Attribute{{Key: "id", Val: "docs-mobile-presentation"}}}
	style.AppendChild(&html.Node{Type: html.TextNode, Data: docsMobileCSS})
	head.AppendChild(style)
	var out bytes.Buffer
	err = html.Render(&out, doc)
	return out.Bytes(), err
}
