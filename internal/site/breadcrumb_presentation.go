package site

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/site/tmplfunc"
	"golang.org/x/net/html"
)

// Existing templated Site pages (for example Learn) invoke breadcrumbs directly
// rather than through doclayout. Give that same frozen template the same context.
func (c *Content) applyBreadcrumbPresentation(p ContentPage, raw []byte) ([]byte, error) {
	if !bytes.Contains(raw, []byte("SiteBreadcrumb")) {
		return raw, nil
	}
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	var crumbs []*html.Node
	var collect func(*html.Node)
	collect = func(n *html.Node) {
		for _, a := range n.Attr {
			if a.Key == "class" {
				for _, class := range strings.Fields(a.Val) {
					if class == "SiteBreadcrumb" {
						crumbs = append(crumbs, n)
						return
					}
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			collect(ch)
		}
	}
	collect(doc)
	if len(crumbs) == 0 {
		return raw, nil
	}
	m, err := pageMetadata(p)
	if err != nil {
		return nil, err
	}
	label, err := breadcrumbLabel(m)
	if err != nil {
		return nil, err
	}
	m["breadcrumbTitle"] = label
	funcs := c.shellFuncs(p)
	funcs["page"] = c.breadcrumbPage
	t := template.New("site-breadcrumb-delivery").Funcs(funcs)
	if err := tmplfunc.Parse(t, string(c.Data["_content/site.tmpl"])); err != nil {
		return nil, err
	}
	for _, old := range crumbs {
		var out bytes.Buffer
		if err := t.ExecuteTemplate(&out, "breadcrumbs", m); err != nil {
			return nil, err
		}
		part, err := html.Parse(&out)
		if err != nil {
			return nil, err
		}
		crumbs = nil
		collect(part)
		if len(crumbs) != 1 {
			return nil, fmt.Errorf("breadcrumb template requires one container")
		}
		nav := crumbs[0]
		if err := c.rewritePresentationLinks(p, nav); err != nil {
			return nil, err
		}
		nav.Parent.RemoveChild(nav)
		old.Parent.InsertBefore(nav, old)
		old.Parent.RemoveChild(old)
	}
	var out bytes.Buffer
	err = html.Render(&out, doc)
	return out.Bytes(), err
}

func (c *Content) rewritePresentationLinks(p ContentPage, n *html.Node) error {
	for i, a := range n.Attr {
		if a.Key == "href" || a.Key == "src" {
			dest, err := c.Resolver.ResolveLink(p.Route, a.Val)
			if err != nil {
				return err
			}
			if strings.HasPrefix(dest, c.Resolver.origin+"/") {
				dest = strings.TrimPrefix(dest, c.Resolver.origin)
			}
			n.Attr[i].Val = dest
		}
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if err := c.rewritePresentationLinks(p, ch); err != nil {
			return err
		}
	}
	return nil
}
