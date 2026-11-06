package sitecontent

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmtext "github.com/yuin/goldmark/text"
)

// Authorized tokens are opaque here. Their exact raw bytes, slot membership,
// pairs and suffixes have already been checked by restoreBlock. Inspect only
// translator-supplied text for new syntax, without lexical/language discovery.
func validatePageSlotSyntax(s Unit, target string, raw []byte) error {
	if err := validateInlineHTMLAttributeBindings(s, target); err != nil {
		return err
	}
	replacements := []string{}
	for _, p := range s.Protected {
		replacements = append(replacements, p.Token, "word")
	}
	plain := target
	for _, p := range s.Protected {
		if p.Binding == nil || p.Binding.Kind != "markdown-title" {
			continue
		}
		for _, close := range s.Protected {
			if close.Role != "close" || close.Pair != p.Pair {
				continue
			}
			a, z := strings.Index(plain, p.Token)+len(p.Token), strings.Index(plain, close.Token)
			if a > z || a < len(p.Token) {
				return fmt.Errorf("Markdown title boundary changed")
			}
			interior := plain[a:z]
			for _, q := range s.Protected {
				if q.Role == "atom" && strings.HasPrefix(q.Raw, "\n") {
					interior = strings.ReplaceAll(interior, q.Token, "")
				}
			}
			if strings.ContainsAny(interior, "\n⟦⟧") {
				return fmt.Errorf("Markdown title cannot cross protected structure")
			}
			plain = plain[:a] + "word" + plain[z:]
		}
	}
	if len(replacements) > 0 {
		plain = strings.NewReplacer(replacements...).Replace(plain)
	}
	if strings.ContainsRune(plain, '\n') {
		return fmt.Errorf("new unprotected line boundary")
	}
	_, tail, _ := strings.Cut(s.Locator, "#")
	context, _, _ := strings.Cut(tail, ":")
	if context == "markdown/reference-title" {
		return nil
	}
	if context == "teaching-comment" {
		restored, err := restoreBlock(s, target)
		if err != nil {
			return err
		}
		if err := validateTeachingBody(s, restored, raw, path.Ext(strings.Split(s.Locator, "#")[0]) == ".md"); err != nil {
			return err
		}
		prose := target
		for _, p := range s.Protected {
			prose = strings.ReplaceAll(prose, p.Token, "")
		}
		_, added, err := protectBlockMode([]byte(prose), context, true)
		if err != nil {
			return err
		}
		if len(added) > 0 {
			return fmt.Errorf("new unprotected teaching technical identity")
		}
	}
	if strings.Contains(context, "/attribute/") && s.Start > 0 {
		quote := raw[s.Start-1]
		if (quote == '\'' || quote == '"') && strings.ContainsRune(plain, rune(quote)) {
			return fmt.Errorf("visible attribute quote boundary changed")
		}
	}
	_, locked, err := protectBlockMode([]byte(plain), context, false)
	if err != nil {
		return err
	}
	if len(locked) != 0 {
		return fmt.Errorf("new unprotected inline structure")
	}
	if strings.HasPrefix(context, "markdown/") {
		return validateEmphasisBindings(s, target)
	}
	return nil
}

// Attribute bindings retain the original opening quote and every inter-value
// markup byte. Adjacent attributes of one tag cannot be separated or reordered.
func validateInlineHTMLAttributeBindings(s Unit, target string) error {
	lastEnd := map[string]int{}
	for i, p := range s.Protected {
		if p.Role != "open" || !strings.HasPrefix(p.Pair, "html-attribute/") {
			continue
		}
		parts := strings.Split(p.Pair, "/")
		if len(parts) != 3 || len(p.Raw) == 0 {
			return fmt.Errorf("invalid inline HTML attribute binding")
		}
		group := parts[1]
		open := strings.Index(target, p.Token)
		start := open + len(p.Token)
		if end, ok := lastEnd[group]; ok && open != end {
			return fmt.Errorf("inline HTML attribute order/markup changed")
		}
		// A paired tag's empty outer opener binds its split start tag to the
		// element body. Prose cannot move ahead of that start tag inside it.
		if i > 0 && s.Protected[i-1].Role == "open" && s.Protected[i-1].Raw == "" {
			outer := s.Protected[i-1]
			if open != strings.Index(target, outer.Token)+len(outer.Token) {
				return fmt.Errorf("inline HTML start tag crossed element body")
			}
		}
		for j := i + 1; j < len(s.Protected); j++ {
			q := s.Protected[j]
			if q.Role != "close" || q.Pair != p.Pair {
				continue
			}
			end := strings.Index(target, q.Token)
			if start > end {
				return fmt.Errorf("inline HTML attribute binding changed")
			}
			value := target[start:end]
			for _, inner := range s.Protected[i+1 : j] {
				if strings.Count(value, inner.Token) != 1 {
					return fmt.Errorf("protected token crossed inline HTML attribute")
				}
				value = strings.ReplaceAll(value, inner.Token, "")
			}
			quote := p.Raw[len(p.Raw)-1]
			if (quote != '\'' && quote != '"') || strings.ContainsRune(value, rune(quote)) || strings.ContainsAny(value, "⟦⟧") {
				return fmt.Errorf("visible attribute quote boundary changed")
			}
			lastEnd[group] = end + len(q.Token)
			break
		}
	}
	return nil
}

// Page boundaries and code/pre bytes are restored from the source verbatim.
// Block hierarchy is checked independently of linguistic slots, teaching-comment
// eligibility, soft wrapping and the order of complete inline bindings.
func validatePageStructure(d Document, source, target []byte) error {
	if err := textBytes(target); err != nil {
		return err
	}
	so, err := parseMetadata(source, func(int, int, string) {})
	if err != nil {
		return err
	}
	to, err := parseMetadata(target, func(int, int, string) {})
	if err != nil {
		return err
	}
	if path.Ext(d.Path) == ".md" {
		if !bytes.Equal(pageBlockShape(source[so:]), pageBlockShape(target[to:])) {
			return fmt.Errorf("Page Markdown block structure changed")
		}
		bound := false
		for _, u := range d.Units {
			for _, p := range u.Protected {
				if p.Binding != nil {
					bound = true
					break
				}
			}
		}
		titleEditable := false
		for _, u := range d.Units {
			for _, p := range u.Protected {
				if p.Binding != nil && p.Binding.EditableTitle {
					titleEditable = true
				}
			}
		}
		if bound && !bytes.Equal(markdownBindingShapeWithTitles(source[so:], !titleEditable), markdownBindingShapeWithTitles(target[to:], !titleEditable)) {
			return fmt.Errorf("Page Markdown link/image destination binding changed")
		}
		if titleEditable {
			if err := validateMarkdownBindingTitles(d, source[so:], target[to:]); err != nil {
				return err
			}
			shape := func(b []byte) string {
				values := []string{}
				for _, r := range markdownReferences(b).References() {
					values = append(values, string(r.Destination()))
				}
				sort.Strings(values)
				return strings.Join(values, "\x00")
			}
			if shape(source[so:]) != shape(target[to:]) {
				return fmt.Errorf("Markdown reference definition destination changed")
			}
		}
	}
	return nil
}

func pageBlockShape(b []byte) []byte {
	tree := goldmark.New(goldmark.WithExtensions(extension.Table)).Parser().Parse(gmtext.NewReader(b))
	var out bytes.Buffer
	_ = ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if n.Type() == ast.TypeInline {
			if v, ok := n.(*ast.Text); ok && enter && v.HardLineBreak() {
				out.WriteString("(HardLineBreak)")
			}
			return ast.WalkContinue, nil
		}
		// Prose paragraphs/TextBlocks and list tightness depend on where the
		// authorized soft-wrap tokens land, and on empty coordinates. New raw
		// line boundaries are rejected before restoration; these are not new
		// source-protected markup. Preserve the enclosing hierarchy instead.
		if n.Kind() == ast.KindParagraph || n.Kind() == ast.KindTextBlock {
			return ast.WalkContinue, nil
		}
		if enter {
			fmt.Fprintf(&out, "(%s", n.Kind())
			switch v := n.(type) {
			case *ast.Heading:
				fmt.Fprintf(&out, ":%d", v.Level)
			case *ast.List:
				fmt.Fprintf(&out, ":%c:%d", v.Marker, v.Start)
			}
		} else {
			out.WriteByte(')')
		}
		return ast.WalkContinue, nil
	})
	return out.Bytes()
}
