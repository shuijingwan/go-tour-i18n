package site

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"

	"github.com/shuijingwan/go-tour-i18n/internal/site/tmplfunc"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
)

// These inputs are invocation-local source semantics, not locale Page coverage.
// Their bytes are checked against the registered frozen manifest before use.
func (c *Content) bindMarkdownSources(root string) error {
	c.markdownSources = map[string][]byte{}
	c.markdownSourceData = map[string][]byte{}
	manifest, err := sourcestorage.LoadManifest(root)
	if err != nil {
		return err
	}
	files := map[string]sourcestorage.FrozenSourceFile{}
	for _, f := range manifest.Files {
		files[f.Path] = f
	}
	read := func(name string) ([]byte, error) {
		f, ok := files[name]
		if !ok {
			return nil, fmt.Errorf("Markdown source not registered: %s", name)
		}
		return sourcestorage.ReadAdoptedSource(root, f)
	}
	for _, p := range c.Pages {
		if p.Format != "md" {
			continue
		}
		b, err := read(p.Path)
		if err != nil {
			return err
		}
		c.markdownSources[p.Path] = b
	}
	for name := range c.Data {
		b, err := read(name)
		if err != nil {
			return err
		}
		c.markdownSourceData[name] = b
	}
	return nil
}

// Match the existing renderer's metadata/template stage before parsing headings.
// For canonical source execution Data is the verified frozen source data; no
// translated heading is allowed to redefine source navigation identity.
func (c *Content) headingMarkdownBody(p ContentPage, raw []byte) ([]byte, error) {
	var body []byte
	var params map[string]any
	var templated bool
	var err error
	if p.Path == "_content/index.md" {
		body, params, err = sitecontent.ShellPageRenderInput(raw)
		templated = true
	} else {
		var title string
		body, title, templated, err = sitecontent.PageRenderInput(raw)
		params = map[string]any{"title": title}
	}
	if err != nil {
		return nil, err
	}
	params["URL"] = p.Route
	if !templated {
		return body, nil
	}
	t := template.New("heading-context").Funcs(c.shellFuncs(p))
	if shell, ok := c.Data["_content/site.tmpl"]; ok {
		if err := tmplfunc.Parse(t, string(shell)); err != nil {
			return nil, err
		}
	}
	part := t.New("heading-body")
	if err := tmplfunc.Parse(part, string(body)); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := part.Execute(&out, params); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

type markdownHeading struct {
	Level        int
	ID, Position string
}

func headingMarkdownParser() goldmark.Markdown {
	// Exact current Site Markdown options; ID generation belongs to Goldmark.
	return goldmark.New(goldmark.WithParserOptions(parser.WithHeadingAttribute(), parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
		goldmark.WithExtensions(extension.Typographer, extension.Linkify, extension.DefinitionList, extension.Table))
}

func canonicalMarkdownHeadings(body []byte) ([]markdownHeading, error) {
	md := headingMarkdownParser()
	tree := md.Parser().Parse(gmtext.NewReader(body))
	headings := []markdownHeading{}
	var walk func(ast.Node, string) error
	walk = func(n ast.Node, position string) error {
		if h, ok := n.(*ast.Heading); ok {
			id, ok := h.AttributeString("id")
			if !ok {
				return fmt.Errorf("canonical Markdown heading has no ID")
			}
			b, ok := id.([]byte)
			if !ok || len(b) == 0 {
				return fmt.Errorf("invalid canonical heading ID")
			}
			headings = append(headings, markdownHeading{h.Level, string(b), position})
		}
		i := 0
		for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
			// Match sitecontent.pageBlockShape: authorized soft-wrap coordinates
			// can split/join prose paragraphs and TextBlocks. Current completion
			// already verifies protected Page slots/order and block hierarchy.
			if ch.Type() == ast.TypeInline || ch.Kind() == ast.KindParagraph || ch.Kind() == ast.KindTextBlock {
				continue
			}
			if err := walk(ch, position+"/"+ch.Kind().String()+":"+strconv.Itoa(i)); err != nil {
				return err
			}
			i++
		}
		return nil
	}
	err := walk(tree, "")
	return headings, err
}

func preserveMarkdownHeadingIDs(source, target, rendered []byte) ([]byte, error) {
	want, err := canonicalMarkdownHeadings(source)
	if err != nil {
		return nil, err
	}
	got, err := canonicalMarkdownHeadings(target)
	if err != nil {
		return nil, err
	}
	if len(want) != len(got) {
		return nil, fmt.Errorf("Markdown heading count mismatch: source=%d localized=%d", len(want), len(got))
	}
	for i := range want {
		if want[i].Level != got[i].Level || want[i].Position != got[i].Position {
			return nil, fmt.Errorf("Markdown heading structure mismatch at %d: source=%+v localized=%+v", i, want[i], got[i])
		}
	}
	if len(want) == 0 {
		return rendered, nil
	}
	doc, err := html.Parse(bytes.NewReader(rendered))
	if err != nil {
		return nil, err
	}
	var nodes []*html.Node
	var collect func(*html.Node, bool)
	collect = func(n *html.Node, main bool) {
		main = main || n.Type == html.ElementNode && n.Data == "main"
		if main && n.Type == html.ElementNode && len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
			nodes = append(nodes, n)
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			collect(ch, main)
		}
	}
	collect(doc, false)
	selected := make([]*html.Node, len(got))
	last := -1
	for i, h := range got {
		matches := 0
		position := -1
		for j, n := range nodes {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == h.ID {
					matches++
					selected[i] = n
					position = j
				}
			}
		}
		if matches != 1 || position <= last || selected[i].Data != "h"+strconv.Itoa(h.Level) {
			return nil, fmt.Errorf("rendered Markdown heading mismatch: %+v matches=%d", h, matches)
		}
		last = position
	}
	for i, n := range selected {
		for j, a := range n.Attr {
			if a.Key == "id" {
				n.Attr[j].Val = want[i].ID
			}
		}
	}
	// Preserve canonical multiplicity: some frozen pages deliberately have an
	// explicit <a id> before a heading with the same generated ID. Do not alter
	// that source representation, or introduce additional static/shell collisions.
	var canonical bytes.Buffer
	if err := headingMarkdownParser().Convert(source, &canonical); err != nil {
		return nil, err
	}
	sourceDOM, err := html.Parse(&canonical)
	if err != nil {
		return nil, err
	}
	wantCounts, counts := elementIDCounts(sourceDOM), elementIDCounts(doc)
	for _, h := range want {
		if counts[h.ID] != wantCounts[h.ID] {
			return nil, fmt.Errorf("source heading ID collision: %s source=%d rendered=%d", h.ID, wantCounts[h.ID], counts[h.ID])
		}
	}
	var out bytes.Buffer
	err = html.Render(&out, doc)
	return out.Bytes(), err
}

func elementIDCounts(doc *html.Node) map[string]int {
	counts := map[string]int{}
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		for _, a := range n.Attr {
			if a.Key == "id" {
				counts[a.Val]++
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			visit(ch)
		}
	}
	visit(doc)
	return counts
}

func (c *Content) applyMarkdownHeadingIDs(p ContentPage, raw []byte) ([]byte, error) {
	// Legacy synthetic Content fixtures predate source-bound localized delivery.
	// Both production Content construction and the source plan bind this map.
	if p.Format != "md" || c.markdownSources == nil {
		return raw, nil
	}
	source, ok := c.markdownSources[p.Path]
	if !ok {
		return nil, fmt.Errorf("Markdown heading source missing: %s", p.Path)
	}
	src := *c
	src.Data = c.markdownSourceData
	if src.Data == nil {
		src.Data = c.Data
	}
	want, err := src.headingMarkdownBody(p, source)
	if err != nil {
		return nil, err
	}
	got, err := c.headingMarkdownBody(p, p.Content)
	if err != nil {
		return nil, err
	}
	b, err := preserveMarkdownHeadingIDs(want, got, raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p.Path, err)
	}
	return b, nil
}
