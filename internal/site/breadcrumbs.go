package site

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"text/template/parse"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
)

// pageMetadata reads presentation metadata without supplying translated text.
func pageMetadata(p ContentPage) (map[string]any, error) {
	params := map[string]any{}
	var title string
	var err error
	if bytes.HasPrefix(p.Content, []byte("---\n")) {
		// Use the canonical full-shell YAML reader: the root source also has
		// summary/template metadata, beyond the legacy Docs Page envelope.
		_, params, err = sitecontent.ShellPageRenderInput(p.Content)
		if err == nil {
			title, _ = params["title"].(string)
		}
	} else {
		_, title, _, err = sitecontent.PageRenderInput(p.Content)
		if err == nil && bytes.HasPrefix(p.Content, []byte("<!--{")) {
			var legacy map[string]any
			end := bytes.Index(p.Content, []byte("}-->"))
			err = json.Unmarshal(p.Content[4:end+1], &legacy)
			for key, value := range legacy {
				params[strings.ToLower(key)] = value
			}
		}
	}
	if err != nil {
		return nil, err
	}
	params["URL"], params["title"] = p.Route, title
	return params, nil
}

func breadcrumbLabel(m map[string]any) (string, error) {
	for _, key := range []string{"breadcrumbTitle", "breadcrumbtitle", "title"} {
		if label, ok := m[key].(string); ok && strings.TrimSpace(label) != "" {
			return label, nil
		}
	}
	return "", fmt.Errorf("breadcrumb metadata has no visible title: %v", m["URL"])
}

// Reuse Go's template syntax tree so spacing, trim markers and comments do not
// turn helper dependency discovery into a second template parser.
func declaresBreadcrumbs(p ContentPage, m map[string]any) (bool, error) {
	if m["breadcrumb"] == true {
		return true, nil
	}
	if m["template"] != true {
		return false, nil
	}
	var body []byte
	var err error
	if bytes.HasPrefix(p.Content, []byte("---\n")) {
		body, _, err = sitecontent.ShellPageRenderInput(p.Content)
	} else {
		body, _, _, err = sitecontent.PageRenderInput(p.Content)
	}
	if err != nil {
		return false, err
	}
	tree := parse.New("breadcrumb-dependency")
	tree.Mode = parse.SkipFuncCheck
	if _, err := tree.Parse(string(body), "", "", map[string]*parse.Tree{}); err != nil {
		return false, err
	}
	var calls func(parse.Node) bool
	calls = func(n parse.Node) bool {
		switch n := n.(type) {
		case *parse.ListNode:
			if n != nil {
				for _, x := range n.Nodes {
					if calls(x) {
						return true
					}
				}
			}
		case *parse.ActionNode:
			return calls(n.Pipe)
		case *parse.PipeNode:
			for _, x := range n.Cmds {
				if calls(x) {
					return true
				}
			}
		case *parse.CommandNode:
			for _, x := range n.Args {
				if calls(x) {
					return true
				}
			}
		case *parse.IdentifierNode:
			return n.Ident == "breadcrumbs"
		case *parse.TemplateNode:
			return n.Name == "breadcrumbs"
		case *parse.IfNode:
			return calls(n.Pipe) || calls(n.List) || calls(n.ElseList)
		case *parse.RangeNode:
			return calls(n.Pipe) || calls(n.List) || calls(n.ElseList)
		case *parse.WithNode:
			return calls(n.Pipe) || calls(n.List) || calls(n.ElseList)
		}
		return false
	}
	return calls(tree.Root), nil
}

// The template traverses directory spellings without a trailing slash. Match
// those to the exact completed canonical route; covered-but-missing is an error.
func (c *Content) completedBreadcrumbPage(route string) (ContentPage, bool, error) {
	keys := []string{route}
	if route != "/" {
		keys = append(keys, route+"/")
	}
	for _, key := range keys {
		if c.Resolver.Covered(key) {
			p, ok := c.Pages[key]
			if !ok || p.Route != key {
				return ContentPage{}, true, fmt.Errorf("covered breadcrumb metadata missing: %s", key)
			}
			return p, true, nil
		}
	}
	return ContentPage{}, false, nil
}

// Bind only ancestor metadata needed by actual Docs breadcrumb declarations.
// These verified source bytes are navigation labels, never local Page coverage
// or source-language article fallbacks. Preview and release composition share it.
func (c *Content) bindBreadcrumbSources(root string) error {
	needed := map[string]bool{}
	for _, p := range c.Pages {
		m, err := pageMetadata(p)
		if err != nil {
			return err
		}
		declared, err := declaresBreadcrumbs(p, m)
		if err != nil {
			return err
		}
		if !declared {
			continue
		}
		for route := path.Dir(path.Clean(p.Route)); ; route = path.Dir(route) {
			needed[route] = true
			if route == "/" {
				break
			}
		}
	}
	c.breadcrumbSources = map[string]ContentPage{}
	if len(needed) == 0 {
		return nil
	}
	manifest, err := sourcestorage.LoadManifest(root)
	if err != nil {
		return err
	}
	files := map[string]sourcestorage.FrozenSourceFile{}
	for _, f := range manifest.Files {
		files[f.Path] = f
	}
	for route := range needed {
		// Root's source context supplies the frozen template's exclusion identity,
		// even when the completed Homepage uses a different localized title.
		if route != "/" {
			_, covered, err := c.completedBreadcrumbPage(route)
			if err != nil {
				return err
			}
			if covered {
				continue
			}
		}
		base := "_content" + strings.TrimSuffix(route, "/")
		candidates := []string{base + "/index.md", base + "/index.html"}
		if route != "/" {
			candidates = append(candidates, base+".md", base+".html")
		}
		var source *sourcestorage.FrozenSourceFile
		for _, name := range candidates {
			if f, ok := files[name]; ok && f.Classification == "upstream-exact" {
				if source != nil {
					return fmt.Errorf("ambiguous frozen breadcrumb page: %s", route)
				}
				copy := f
				source = &copy
			}
		}
		if source == nil {
			return fmt.Errorf("frozen breadcrumb metadata missing: %s", route)
		}
		b, err := sourcestorage.ReadAdoptedSource(root, *source)
		if err != nil {
			return err
		}
		p := ContentPage{Route: route, Path: source.Path, Content: b}
		m, err := pageMetadata(p)
		if err != nil {
			return err
		}
		if _, err := breadcrumbLabel(m); err != nil {
			return err
		}
		c.breadcrumbSources[route] = p
	}
	return nil
}

func (c *Content) breadcrumbPage(route string) (map[string]any, error) {
	var p ContentPage
	if route != "/" {
		local, covered, err := c.completedBreadcrumbPage(route)
		if err != nil {
			return nil, err
		}
		if covered {
			p = local
		}
	}
	if p.Route == "" {
		var ok bool
		p, ok = c.breadcrumbSources[route]
		if !ok {
			return nil, fmt.Errorf("breadcrumb metadata missing: %s", route)
		}
	}
	m, err := pageMetadata(p)
	if err != nil {
		return nil, err
	}
	label, err := breadcrumbLabel(m)
	if err != nil {
		return nil, err
	}
	// Legacy JSON folds keys; the frozen template uses breadcrumbTitle.
	m["breadcrumbTitle"] = label
	return m, nil
}
