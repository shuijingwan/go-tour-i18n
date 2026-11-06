package sitecontent

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	gmtext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type MarkdownReferenceDefinition struct {
	Label       string `json:"label"`
	Start       int    `json:"label_start"`
	End         int    `json:"label_end"`
	Destination string `json:"destination"`
	Title       string `json:"title,omitempty"`
}

func markdownReferences(raw []byte) parser.Context {
	pc := parser.NewContext()
	goldmark.New().Parser().Parse(gmtext.NewReader(raw), parser.WithContext(pc))
	return pc
}

// Goldmark's reference context is the authority. A bounded label substitution
// proves which source coordinates actually own a definition (including quoted
// and list containers); apparent definitions in code never acquire ownership.
func markdownReferenceDefinitions(raw []byte) ([]MarkdownReferenceDefinition, error) {
	pc := markdownReferences(raw)
	if len(pc.References()) == 0 {
		return nil, nil
	}
	const probe = "protected-reference-coordinate-probe"
	if _, ok := pc.Reference(probe); ok {
		return nil, fmt.Errorf("reserved reference probe collision")
	}
	defs := []MarkdownReferenceDefinition{}
	for i := 0; i < len(raw); i++ {
		if raw[i] != '[' {
			continue
		}
		z := balancedEnd(raw, i, '[', ']')
		if z < 0 || z+1 >= len(raw) || raw[z+1] != ':' {
			continue
		}
		ref, ok := pc.Reference(util.ToLinkReference(raw[i+1 : z]))
		if !ok {
			continue
		}
		changed := append(bytes.Clone(raw[:i+1]), []byte(probe)...)
		changed = append(changed, raw[z:]...)
		r, ok := markdownReferences(changed).Reference(probe)
		if !ok || !bytes.Equal(r.Destination(), ref.Destination()) || !bytes.Equal(r.Title(), ref.Title()) {
			continue
		}
		defs = append(defs, MarkdownReferenceDefinition{Label: string(raw[i+1 : z]), Start: i + 1, End: z, Destination: string(ref.Destination()), Title: string(ref.Title())})
		i = z
	}
	return defs, nil
}

func markdownReferenceTitleSpans(raw []byte, add func(int, int, string)) error {
	defs, err := markdownReferenceDefinitions(raw)
	if err != nil {
		return err
	}
	const probe = "protected-title-coordinate-probe"
	for _, def := range defs {
		// Titles may continue on the following line, but never across a blank
		// line. Parser observation below rejects destination/ordinary quotes.
		end := len(raw)
		if n := bytes.Index(raw[def.End:], []byte("\n\n")); n >= 0 {
			end = def.End + n
		}
		found := false
		for a := def.End + 2; a < end && !found; a++ {
			q := raw[a]
			close := q
			if q == '(' {
				close = ')'
			} else if q != '\'' && q != '"' {
				continue
			}
			if a == 0 || !strings.ContainsRune(" \t\n", rune(raw[a-1])) {
				continue
			}
			for z := a + 1; z < end; z++ {
				if raw[z] == '\\' {
					z++
					continue
				}
				if raw[z] != close {
					continue
				}
				changed := append(bytes.Clone(raw[:a+1]), []byte(probe)...)
				changed = append(changed, raw[z:]...)
				r, ok := markdownReferences(changed).Reference(util.ToLinkReference([]byte(def.Label)))
				if ok && string(r.Title()) == probe && string(r.Destination()) == def.Destination {
					add(a+1, z, "markdown/reference-title")
					found = true
				}
				break
			}
		}
	}
	return nil
}

// Choose a shared reference key from the first source-owned implicit usage.
// The model supplies only visible prose. Every definition and explicit usage
// of that identity follows deterministically; unequal visible labels become
// full references to the same key. No proximity matching or translation occurs.
func restorePageReferenceBindings(d Document, target string) ([]byte, error) {
	u := d.Units[0]
	keys := map[string]string{}
	labels := map[string]string{}
	changedLabel := false
	positions := map[string]int{}
	for _, p := range u.Protected {
		positions[p.Token] = strings.Index(target, p.Token)
	}
	for _, p := range u.Protected {
		b := p.Binding
		if b == nil || b.Version != 2 || !b.Implicit || b.Reference == "" {
			continue
		}
		var open ProtectedToken
		for _, q := range u.Protected {
			if q.Role == "open" && q.Pair == p.Pair {
				open = q
				break
			}
		}
		a, z := positions[open.Token]+len(open.Token), positions[p.Token]
		if a > z {
			return nil, fmt.Errorf("reference usage binding changed")
		}
		label := target[a:z]
		replacements := []string{}
		for _, q := range u.Protected {
			replacements = append(replacements, q.Token, q.Raw)
		}
		if len(replacements) > 0 {
			label = strings.NewReplacer(replacements...).Replace(label)
		}
		key := util.ToLinkReference([]byte(b.Reference))
		if _, ok := keys[key]; !ok {
			keys[key] = label
		}
		labels[p.Token] = label
		changedLabel = changedLabel || label != b.Label
	}
	if len(keys) == 0 || !changedLabel {
		v, err := restoreBlock(u, target)
		return []byte(v), err
	}
	validKey := func(key string) bool {
		normal := util.ToLinkReference([]byte(key))
		keyContext := parser.NewContext()
		keyTree := goldmark.New().Parser().Parse(gmtext.NewReader([]byte("["+key+"]: /binding-key-probe\n")), parser.WithContext(keyContext))
		keyRef, keyOK := keyContext.Reference(normal)
		empty := true
		for n := keyTree.FirstChild(); n != nil; n = n.NextSibling() {
			if (n.Kind().String() != "Paragraph" && n.Kind().String() != "TextBlock") || n.FirstChild() != nil {
				empty = false
			}
		}
		return normal != "" && keyOK && len(keyContext.References()) == 1 && empty && string(keyRef.Destination()) == "/binding-key-probe"
	}
	canonical := map[string]string{}
	for _, def := range d.ReferenceDefinitions {
		old := util.ToLinkReference([]byte(def.Label))
		if _, ok := canonical[old]; !ok {
			canonical[old] = def.Label
		}
		if key, ok := keys[old]; ok && !validKey(key) {
			keys[old] = canonical[old]
		}
	}
	// Labels may legitimately collide across different destinations. Preserve
	// their source keys and emit full references in that case; never constrain
	// the translator to choose different words to repair machine identity.
	for pass := 0; pass <= len(keys); pass++ {
		owners := map[string]string{}
		collisions := map[string]bool{}
		for _, def := range d.ReferenceDefinitions {
			old := util.ToLinkReference([]byte(def.Label))
			key := def.Label
			if v, ok := keys[old]; ok {
				key = v
			}
			normal := util.ToLinkReference([]byte(key))
			if prior, ok := owners[normal]; ok && prior != old {
				collisions[prior] = true
				collisions[old] = true
			}
			owners[normal] = old
		}
		changed := false
		for old := range collisions {
			if _, ok := keys[old]; ok && keys[old] != canonical[old] {
				keys[old] = canonical[old]
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	overrides, err := restoreStructuralBindings(u, target, positions)
	if err != nil {
		return nil, err
	}
	for _, p := range u.Protected {
		b := p.Binding
		if b == nil || b.Version != 2 || b.Reference == "" {
			continue
		}
		key, ok := keys[util.ToLinkReference([]byte(b.Reference))]
		if !ok {
			continue
		}
		if b.Implicit && labels[p.Token] == key {
			if p.Raw == "]" {
				overrides[p.Token] = "]"
			} else {
				overrides[p.Token] = "][]"
			}
		} else {
			overrides[p.Token] = "][" + key + "]"
		}
	}
	pieces, err := protectionPieces(u)
	if err != nil {
		return nil, err
	}
	for _, piece := range pieces {
		if piece.token == nil {
			continue
		}
		changes := []MarkdownReferenceDefinition{}
		for _, def := range d.ReferenceDefinitions {
			if _, ok := keys[util.ToLinkReference([]byte(def.Label))]; ok && def.Start >= piece.start && def.End <= piece.end {
				changes = append(changes, def)
			}
		}
		if len(changes) == 0 {
			continue
		}
		if piece.token.Role != "boundary" {
			return nil, fmt.Errorf("reference definition not owned by frozen boundary")
		}
		raw := piece.token.Raw
		for i := len(changes) - 1; i >= 0; i-- {
			def := changes[i]
			raw = raw[:def.Start-piece.start] + keys[util.ToLinkReference([]byte(def.Label))] + raw[def.End-piece.start:]
		}
		overrides[piece.token.Token] = raw
	}
	order := append([]ProtectedToken{}, u.Protected...)
	sort.Slice(order, func(i, j int) bool { return positions[order[i].Token] < positions[order[j].Token] })
	var out strings.Builder
	pos := 0
	for _, p := range order {
		a := positions[p.Token]
		out.WriteString(target[pos:a])
		v := p.Raw
		if r, ok := overrides[p.Token]; ok {
			v = r
		}
		out.WriteString(v)
		pos = a + len(p.Token)
	}
	out.WriteString(target[pos:])
	return []byte(out.String()), nil
}
