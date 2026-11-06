package sitecontent

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	gmtext "github.com/yuin/goldmark/text"
)

func markdownBindingShape(raw []byte) []byte {
	return markdownBindingShapeWithTitles(raw, true)
}
func markdownBindingShapeWithTitles(raw []byte, titles bool) []byte {
	tree := goldmark.New().Parser().Parse(gmtext.NewReader(raw))
	bindings := []string{}
	_ = ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if enter {
			switch x := n.(type) {
			case *ast.Link:
				title := ""
				if titles {
					title = string(x.Title)
				}
				bindings = append(bindings, identity([]string{"link", string(x.Destination), title}))
			case *ast.Image:
				title := ""
				if titles {
					title = string(x.Title)
				}
				bindings = append(bindings, identity([]string{"image", string(x.Destination), title}))
			}
		}
		return ast.WalkContinue, nil
	})
	sort.Strings(bindings)
	return []byte(strings.Join(bindings, "\n"))
}

func validateMarkdownBindingTitles(d Document, source, target []byte) error {
	// Keep frozen title requirements even in a mixed old/new document. An
	// editable title authorizes its own prose, not removal of another title.
	counts := func(raw []byte) map[string]int {
		out := map[string]int{}
		tree := goldmark.New().Parser().Parse(gmtext.NewReader(raw))
		_ = ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
			if enter {
				switch x := n.(type) {
				case *ast.Link:
					out[identity([]string{"markdown-link", string(x.Destination), string(x.Title)})]++
				case *ast.Image:
					out[identity([]string{"markdown-image", string(x.Destination), string(x.Title)})]++
				}
			}
			return ast.WalkContinue, nil
		})
		return out
	}
	fixed, actual := counts(source), counts(target)
	editableRefTitles := map[string]bool{}
	for _, s := range d.Slots {
		if strings.Contains(s.Locator, "#markdown/reference-title:") {
			owner := -1
			for i, def := range d.ReferenceDefinitions {
				if def.End < s.Start && (owner < 0 || def.End > d.ReferenceDefinitions[owner].End) {
					owner = i
				}
			}
			if owner >= 0 {
				def := d.ReferenceDefinitions[owner]
				editableRefTitles[identity([]string{def.Destination, def.Title})] = true
			}
		}
	}
	for _, u := range d.Units {
		for _, p := range u.Protected {
			b := p.Binding
			if b == nil || p.Role != "close" {
				continue
			}
			if b.EditableTitle || (b.Reference != "" && editableRefTitles[identity([]string{b.Destination, b.Title})]) {
				k := identity([]string{b.Kind, b.Destination, b.Title})
				if fixed[k] > 0 {
					fixed[k]--
				}
			}
		}
	}
	for k, n := range fixed {
		if actual[k] < n {
			return fmt.Errorf("frozen Markdown title binding changed")
		}
	}
	return nil
}

// StructuralBinding v1 is additive protected-token authority. Absent metadata
// retains the historical literal reader. The visible label is never an ID:
// implicit references are expanded to full references only when necessary.
// Definitions (including unused ones) and their URLs stay source-exact.
type StructuralBinding struct {
	Version       int    `json:"version"`
	Kind          string `json:"kind"`
	Destination   string `json:"destination"`
	Title         string `json:"title,omitempty"`
	Reference     string `json:"reference,omitempty"`
	Implicit      bool   `json:"implicit,omitempty"`
	Label         string `json:"source_label,omitempty"`
	EditableTitle bool   `json:"editable_title,omitempty"`
}

func markdownBoundNode(raw []byte, refs []parser.Reference) ast.Node {
	pc := parser.NewContext()
	for _, ref := range refs {
		pc.AddReference(ref)
	}
	tree := goldmark.New().Parser().Parse(gmtext.NewReader(raw), parser.WithContext(pc))
	p := tree.FirstChild()
	if p == nil || p.NextSibling() != nil || p.Kind() != ast.KindParagraph {
		return nil
	}
	n := p.FirstChild()
	if n == nil || n.NextSibling() != nil {
		return nil
	}
	switch n.(type) {
	case *ast.Link, *ast.Image:
		return n
	}
	return nil
}

// Locate a title only after Goldmark has established a complete inline link.
// Verify each delimiter candidate with the same parser, rather than treating
// a quote inside a destination as the start of a title.
func markdownTitleRange(close []byte) (int, int, bool) {
	if !bytes.HasPrefix(close, []byte("](")) {
		return 0, 0, false
	}
	for i := 3; i < len(close); i++ {
		if close[i-1] != ' ' && close[i-1] != '\t' && close[i-1] != '\n' {
			continue
		}
		quote := close[i]
		endQuote := quote
		if quote == '(' {
			endQuote = ')'
		} else if quote != '\'' && quote != '"' {
			continue
		}
		for j := i + 1; j < len(close); j++ {
			if close[j] == '\\' {
				j++
				continue
			}
			if close[j] != endQuote {
				continue
			}
			if strings.TrimSpace(string(close[j+1:])) != ")" {
				break
			}
			probe := append([]byte("[x]"), close[1:i+1]...)
			probe = append(probe, []byte("binding-title-probe")...)
			probe = append(probe, close[j:]...)
			if n, ok := markdownBoundNode(probe, nil).(*ast.Link); ok && string(n.Title) == "binding-title-probe" {
				return i + 1, j, true
			}
			break
		}
	}
	return 0, 0, false
}

func markdownStructuralBinding(raw, label, close []byte, refs []parser.Reference) *StructuralBinding {
	n := markdownBoundNode(raw, refs)
	if n == nil {
		// Persist the fact that bracket prose is unresolved. A translation must
		// not accidentally turn it into a link to an unused source definition.
		return &StructuralBinding{Version: 1, Kind: "markdown-literal"}
	}
	b := &StructuralBinding{Version: 1}
	switch x := n.(type) {
	case *ast.Link:
		b.Kind = "markdown-link"
		b.Destination = string(x.Destination)
		b.Title = string(x.Title)
	case *ast.Image:
		b.Kind = "markdown-image"
		b.Destination = string(x.Destination)
		b.Title = string(x.Title)
	}
	if bytes.Equal(close, []byte("]")) || bytes.Equal(close, []byte("][]")) {
		b.Implicit = true
		b.Reference = string(label)
		b.Label = string(label)
	} else if bytes.HasPrefix(close, []byte("][")) {
		b.Reference = string(close[2 : len(close)-1])
	}
	if b.Reference != "" {
		b.Version = 2
	}
	return b
}

func restoreStructuralBindings(u Unit, target string, positions map[string]int) (map[string]string, error) {
	replacements := map[string]string{}
	// Resolve inner bindings before outer ones. The exact paired tokens already
	// passed the common stack and unique-token checks in restoreBlock.
	for i := 0; i < len(u.Protected); i++ {
		p := u.Protected[i]
		b := p.Binding
		if b == nil {
			continue
		}
		if b.Version == 2 && (b.Kind == "markdown-title" || b.Kind == "markdown-reference-title") && p.Role == "atom" {
			continue
		}
		if (b.Version != 1 && b.Version != 2) || p.Role != "close" {
			return nil, fmt.Errorf("unknown structural binding version/role")
		}
		var open *ProtectedToken
		for j := i - 1; j >= 0; j-- {
			if u.Protected[j].Role == "open" && u.Protected[j].Pair == p.Pair {
				q := u.Protected[j]
				open = &q
				break
			}
		}
		if open == nil {
			return nil, fmt.Errorf("missing structural binding opener")
		}
		a, z := positions[open.Token]+len(open.Token), positions[p.Token]
		label := target[a:z]
		pairs := []string{}
		for _, q := range u.Protected {
			raw := q.Raw
			if v, ok := replacements[q.Token]; ok {
				raw = v
			}
			pairs = append(pairs, q.Token, raw)
		}
		if len(pairs) > 0 {
			label = strings.NewReplacer(pairs...).Replace(label)
		}
		close := p.Raw
		if b.Implicit && label != b.Label {
			close = "][" + b.Reference + "]"
		}
		refs := []parser.Reference{}
		if b.Reference != "" {
			refs = append(refs, parser.NewReference([]byte(b.Reference), []byte(b.Destination), []byte(b.Title)))
		}
		n := markdownBoundNode([]byte(open.Raw+label+close), refs)
		valid := false
		if b.Kind == "markdown-literal" {
			valid = n == nil
		}
		switch x := n.(type) {
		case *ast.Link:
			valid = b.Kind == "markdown-link" && string(x.Destination) == b.Destination && (b.EditableTitle || string(x.Title) == b.Title)
		case *ast.Image:
			valid = b.Kind == "markdown-image" && string(x.Destination) == b.Destination && (b.EditableTitle || string(x.Title) == b.Title)
		}
		if !valid {
			return nil, fmt.Errorf("Markdown structural label/target binding changed")
		}
		replacements[p.Token] = close
	}
	return replacements, nil
}
