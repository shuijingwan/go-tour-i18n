package site

import (
	"bytes"
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/tour"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"golang.org/x/net/html"
)

//go:embed project_presentation.css
var projectPresentationCSS string

type projectDirectoryGroup struct {
	Path    string
	Covered bool
	Routes  []string
}

// No prefix implies completion. Every directory child comes from the validated
// Resolver exact set; group roots are links only when independently covered.
func projectDirectory(r *Resolver) []projectDirectoryGroup {
	groups := map[string][]string{}
	for route, covered := range r.routes {
		if !covered {
			continue
		}
		parts := strings.Split(strings.Trim(route, "/"), "/")
		key := "/"
		if parts[0] != "" {
			key = "/" + parts[0] + "/"
		}
		if parts[0] == "doc" && len(parts) > 1 {
			key = "/doc/" + parts[1] + "/"
		}
		groups[key] = append(groups[key], route)
	}
	keys := []string{}
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := []projectDirectoryGroup{}
	for _, key := range keys {
		sort.Strings(groups[key])
		out = append(out, projectDirectoryGroup{key, r.Covered(key), groups[key]})
	}
	return out
}

func projectElement(tag, class string) *html.Node {
	n := &html.Node{Type: html.ElementNode, Data: tag}
	if class != "" {
		n.Attr = []html.Attribute{{Key: "class", Val: class}}
	}
	return n
}
func projectAddClass(n *html.Node, class string) {
	for i, a := range n.Attr {
		if a.Key == "class" {
			n.Attr[i].Val += " " + class
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: "class", Val: class})
}
func projectText(tag, text string) *html.Node {
	n := projectElement(tag, "")
	n.AppendChild(&html.Node{Type: html.TextNode, Data: text})
	return n
}
func projectFind(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := projectFind(c, tag); found != nil {
			return found
		}
	}
	return nil
}
func projectMove(parent, child *html.Node) {
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}
	parent.AppendChild(child)
}

func projectClone(n *html.Node) *html.Node {
	v := &html.Node{Type: n.Type, Data: n.Data, DataAtom: n.DataAtom, Attr: append([]html.Attribute(nil), n.Attr...)}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		v.AppendChild(projectClone(c))
	}
	return v
}

func projectStyledDocument(doc, head *html.Node) ([]byte, error) {
	style := projectElement("style", "")
	style.Attr = []html.Attribute{{Key: "id", Val: "project-page-presentation"}}
	style.AppendChild(&html.Node{Type: html.TextNode, Data: projectPresentationCSS})
	head.AppendChild(style)
	var out bytes.Buffer
	err := html.Render(&out, doc)
	return out.Bytes(), err
}

// Presentation-only migration from the mature legacy home.tmpl body model.
// All prose stays owned by the accepted translation body; section labels are
// reused from the existing UI catalog. No localized text is authored here.
func (c *Content) applyProjectPresentation(locale string, p ContentPage, raw []byte) ([]byte, error) {
	if p.Path != "data/site-shell/translation.html" || c.Resolver == nil {
		return nil, fmt.Errorf("project presentation authority missing")
	}
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	main, head := projectFind(doc, "main"), projectFind(doc, "head")
	if main == nil || head == nil {
		return nil, fmt.Errorf("project shell boundary missing")
	}
	// A future formally accepted rich body owns its own prose and sections.
	// Apply the same presentation without forcing it back into the predecessor's
	// five paragraphs. Future applications/regions/maintenance sections remain
	// source-owned HTML, with no guessed links or locally authored labels.
	for n := main.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.ElementNode && n.Data == "section" {
			projectAddClass(main, "ProjectPage")
			for section := main.FirstChild; section != nil; section = section.NextSibling {
				if section.Type == html.ElementNode && section.Data == "section" {
					projectAddClass(section, "ProjectSection")
				}
			}
			return projectStyledDocument(doc, head)
		}
	}
	catalog, err := ui.Load(locale)
	if err != nil {
		return nil, err
	}
	label := func(key string) (string, error) { return catalog.Plain(key) }
	var title, links *html.Node
	paragraphs := []*html.Node{}
	for n := main.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.TextNode && strings.TrimSpace(n.Data) == "" {
			continue
		}
		if n.Type != html.ElementNode {
			return nil, fmt.Errorf("unknown project body node")
		}
		switch n.Data {
		case "h1":
			if title != nil {
				return nil, fmt.Errorf("duplicate project title")
			}
			title = n
		case "p":
			paragraphs = append(paragraphs, n)
		case "nav":
			if links != nil {
				return nil, fmt.Errorf("duplicate project links")
			}
			links = n
		default:
			return nil, fmt.Errorf("project successor requires explicit presentation binding for %s", n.Data)
		}
	}
	if title == nil || links == nil || len(paragraphs) != 5 {
		return nil, fmt.Errorf("project accepted body structure changed")
	}
	for main.FirstChild != nil {
		main.RemoveChild(main.FirstChild)
	}
	root := projectElement("div", "ProjectPage")
	main.AppendChild(root)
	hero := projectElement("section", "ProjectHero")
	hero.Attr = append(hero.Attr, html.Attribute{Key: "data-project-section", Val: "introduction"})
	root.AppendChild(hero)
	projectMove(hero, title)
	projectMove(hero, paragraphs[0])
	section := func(id, key string) (*html.Node, error) {
		text, err := label(key)
		if err != nil {
			return nil, err
		}
		n := projectElement("section", "ProjectSection")
		n.Attr = append(n.Attr, html.Attribute{Key: "id", Val: "project-" + id}, html.Attribute{Key: "data-project-section", Val: id})
		n.AppendChild(projectText("h2", text))
		root.AppendChild(n)
		return n, nil
	}
	projects, err := section("coverage", "site.current_projects")
	if err != nil {
		return nil, err
	}
	grid := projectElement("div", "ProjectDirectory")
	grid.Attr = append(grid.Attr, html.Attribute{Key: "data-project-coverage", Val: ""})
	projects.AppendChild(grid)
	for _, g := range projectDirectory(c.Resolver) {
		details := projectElement("details", "ProjectCard")
		summary := projectElement("summary", "")
		if g.Covered {
			a := projectText("a", g.Path)
			a.Attr = []html.Attribute{{Key: "href", Val: g.Path}}
			summary.AppendChild(a)
		} else {
			summary.AppendChild(projectText("span", g.Path))
		}
		summary.AppendChild(projectText("span", fmt.Sprintf(" (%d)", len(g.Routes))))
		details.AppendChild(summary)
		list := projectElement("ul", "")
		for _, route := range g.Routes {
			li := projectElement("li", "")
			a := projectText("a", route)
			a.Attr = []html.Attribute{{Key: "href", Val: route}}
			li.AppendChild(a)
			list.AppendChild(li)
		}
		details.AppendChild(list)
		grid.AppendChild(details)
	}
	status, err := section("status", "site.project_status")
	if err != nil {
		return nil, err
	}
	projectMove(status, paragraphs[4])
	if c.baselineCommit != "" {
		text, err := label("site.upstream_baseline")
		if err != nil {
			return nil, err
		}
		dl := projectElement("dl", "")
		dl.AppendChild(projectText("dt", text))
		dd := projectElement("dd", "")
		a := projectText("a", c.baselineRepository+"@"+c.baselineCommit)
		a.Attr = []html.Attribute{{Key: "href", Val: "https://github.com/" + c.baselineRepository + "/tree/" + c.baselineCommit}}
		dd.AppendChild(a)
		dl.AppendChild(dd)
		status.AppendChild(dl)
	}
	languages, err := section("languages", "site.language_versions")
	if err != nil {
		return nil, err
	}
	model, err := tour.NewSharedChromeModel(locale, c.Resolver.origin)
	if err != nil {
		return nil, err
	}
	list := projectElement("ul", "ProjectLanguages")
	languages.AppendChild(list)
	for _, l := range model.Languages {
		li := projectElement("li", "")
		if l.Current {
			n := projectText("span", l.Label)
			n.Attr = []html.Attribute{{Key: "aria-current", Val: "page"}}
			li.AppendChild(n)
		} else {
			n := projectText("a", l.Label)
			n.Attr = []html.Attribute{{Key: "href", Val: l.URL}}
			li.AppendChild(n)
		}
		list.AppendChild(li)
	}
	projectLinks, err := section("links", "site.project_links")
	if err != nil {
		return nil, err
	}
	projectMove(projectLinks, links)
	validation, err := section("validation", "site.translation_validation")
	if err != nil {
		return nil, err
	}
	projectMove(validation, paragraphs[1])
	workflow, err := section("workflow", "site.how_it_works")
	if err != nil {
		return nil, err
	}
	projectMove(workflow, paragraphs[2])
	support, err := section("support", "support.section_title")
	if err != nil {
		return nil, err
	}
	projectMove(support, paragraphs[3])
	learning, err := section("learning", "site.continue_learning_title")
	if err != nil {
		return nil, err
	}
	// The existing official-source link is presented as the final learning CTA,
	// retaining its exact accepted label and href; no replacement prose.
	for n := links.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" && a.Val == "https://go.dev/" {
					learning.AppendChild(projectClone(n))
				}
			}
		}
	}
	return projectStyledDocument(doc, head)
}
