package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/token"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type surfaceTransportPart struct {
	Token      string `json:"token"`
	End        string `json:"end,omitempty"`
	Value      string `json:"value"`
	Region     string `json:"region,omitempty"`
	Slot       int    `json:"slot"`
	Atom       int    `json:"atom"`
	FormalAtom int    `json:"formal_atom"`
	AtomToken  string `json:"atom_token,omitempty"`
	Prefix     string `json:"prefix,omitempty"`
	Suffix     string `json:"suffix,omitempty"`
}
type surfacePageTransport struct {
	Ordinary              bool                   `json:"ordinary,omitempty"`
	Schema                string                 `json:"schema"`
	UnitID                string                 `json:"unit_id"`
	CanonicalSourceSHA256 string                 `json:"canonical_source_sha256"`
	Origin                PageAdapter            `json:"origin"`
	Parts                 []surfaceTransportPart `json:"parts"`
	Template              string                 `json:"template"`
}

func (a PageAdapter) surfaceTransportPageOrigin() PageAdapter { return a }

// Keep the existing readonly registration guard alive across construction.
// This observes current membership/adoption heads and directory additions;
// it is not a cached registration/currentness PASS.
func (a PageAdapter) WithCurrentSurfaceTransportContext(root string, c *i18n.Catalog, work func(string) error) error {
	return withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		fresh, err := i18n.BuildSourceCatalog(root)
		if err != nil {
			return err
		}
		if err := AttachRegisteredTranslationPackages(root, fresh, v); err != nil {
			return err
		}
		type unit struct {
			ID        string
			Kind      i18n.UnitKind
			Path, SHA string
		}
		type owner struct {
			ID, SHA string
			Units   []unit
		}
		stamp := func(p i18n.TranslationPackageDescriptor) owner {
			r := owner{ID: p.ID, SHA: p.AuthoritySHA256}
			for _, u := range p.Units {
				r.Units = append(r.Units, unit{u.Unit.ID, u.Unit.Kind, u.Unit.SourcePath, u.Unit.SourceSHA256})
			}
			return r
		}
		current := map[string]owner{}
		ordered := []owner{}
		for _, p := range fresh.TranslationPackages {
			s := stamp(p)
			current[p.ID] = s
			ordered = append(ordered, s)
		}
		for _, p := range c.TranslationPackages {
			if expected, ok := current[p.ID]; ok && !reflect.DeepEqual(expected, stamp(p)) {
				return fmt.Errorf("Surface current registered package changed: %s", p.ID)
			}
		}
		return work(identity(ordered))
	})
}

// Only the existing parser-owned ordinary prose slots and tightly bound inline
// API descriptors are eligible. Code blocks/comments/directives/literals and
// arbitrary portions of an atomic token are never offered as coordinates.
func (a PageAdapter) SurfaceTransportProjection(origin i18n.TranslationUnitAdapter, raw []byte, regions []string) (i18n.SurfaceTransportProjection, i18n.TranslationUnitAdapter, error) {
	provider, ok := origin.(interface{ surfaceTransportPageOrigin() PageAdapter })
	if !ok {
		return i18n.SurfaceTransportProjection{}, nil, fmt.Errorf("unknown Surface ordinary source mapping")
	}
	o := provider.surfaceTransportPageOrigin()
	if len(o.Document.Units) != 1 || len(a.Document.Units) != 1 || len(o.Document.Slots) != len(a.Document.Slots) || a.Document.Path != o.Document.Path || digest(o.Source) != o.Document.SourceSHA {
		return i18n.SurfaceTransportProjection{}, nil, fmt.Errorf("Surface ordinary source shape mismatch")
	}
	if _, err := o.Restore(string(raw)); err != nil {
		return i18n.SurfaceTransportProjection{}, nil, err
	}
	uid := a.Document.Units[0].ID
	type bound struct {
		index, pos, slot, local int
		p                       ProtectedToken
	}
	var boundTokens []bound
	si, local := -1, 0
	for i, p := range o.Document.Units[0].Protected {
		if strings.Count(string(raw), p.Token) != 1 {
			return i18n.SurfaceTransportProjection{}, nil, fmt.Errorf("Surface source token exact-set mismatch")
		}
		if p.Role == "boundary" {
			si++
			local = 0
			boundTokens = append(boundTokens, bound{i, strings.Index(string(raw), p.Token), si, -1, p})
			continue
		}
		boundTokens = append(boundTokens, bound{i, strings.Index(string(raw), p.Token), si, local, p})
		local++
	}
	regionID := func(slot, atom int, kind string) string {
		return digest([]byte(fmt.Sprintf("surface-source-region/v1/%s/%s/%d/%d/%s", uid, a.Document.SourceSHA, slot, atom, kind)))
	}
	available := []i18n.SurfaceTransportRegion{}
	prose := map[int]string{}
	for i, s := range a.Document.Slots {
		if strings.Contains(s.Locator, "#teaching-comment:") {
			continue
		}
		id := regionID(i, -1, "ordinary-prose")
		prose[i] = id
		available = append(available, i18n.SurfaceTransportRegion{ID: id, Kind: "ordinary-prose", SourceCoordinate: s.Locator, Preview: string(a.Source[s.Start:s.End])})
	}
	// Display-only target context follows the already verified source token and
	// boundary correspondence. It is never searched to construct a mapping.
	for i := range available {
		for slot, id := range prose {
			if id != available[i].ID {
				continue
			}
			start, end := -1, -1
			for _, b := range boundTokens {
				if b.p.Role == "boundary" && b.slot == slot {
					start = b.pos + len(b.p.Token)
				}
				if b.p.Role == "boundary" && b.slot == slot+1 {
					end = b.pos
				}
			}
			if start < 0 || end < start {
				return i18n.SurfaceTransportProjection{}, nil, fmt.Errorf("Surface source boundary correspondence missing")
			}
			value := string(raw[start:end])
			pairs := []string{}
			for _, b := range boundTokens {
				if b.p.Role != "boundary" && b.slot == slot {
					pairs = append(pairs, b.p.Token, b.p.Raw)
				}
			}
			if len(pairs) > 0 {
				value = strings.NewReplacer(pairs...).Replace(value)
			}
			available[i].RestoredPreview = value
		}
	}
	type descriptor struct{ id, prefix, word, suffix string }
	descriptors := map[int]descriptor{}
	for _, b := range boundTokens {
		if b.local < 0 || b.slot < 0 || b.slot >= len(a.Document.Slots) || prose[b.slot] == "" {
			continue
		}
		s := a.Document.Slots[b.slot]
		if b.local >= len(s.Protected) || s.Protected[b.local].Raw != b.p.Raw {
			continue
		}
		prefix, word, suffix, ok := surfaceInlineDescriptor(s, b.local)
		if !ok {
			continue
		}
		id := regionID(b.slot, b.local, "inline-api-descriptor")
		descriptors[b.index] = descriptor{id, prefix, word, suffix}
		available = append(available, i18n.SurfaceTransportRegion{ID: id, Kind: "inline-api-descriptor", SourceCoordinate: fmt.Sprintf("%s/protected/%d/descriptor", s.Locator, b.local), Preview: b.p.Raw})
	}
	wanted := map[string]bool{}
	for _, id := range regions {
		if wanted[id] {
			return i18n.SurfaceTransportProjection{}, nil, fmt.Errorf("duplicate Surface editable coordinate")
		}
		wanted[id] = true
	}
	for _, r := range available {
		if wanted[r.ID] {
			delete(wanted, r.ID)
		}
	}
	if len(wanted) != 0 {
		return i18n.SurfaceTransportProjection{}, nil, fmt.Errorf("Surface editable scope outside source-owned regions")
	}
	for _, id := range regions {
		wanted[id] = true
	}
	p := &surfacePageTransport{Schema: "go-learning/surface-page-codec/v1", UnitID: uid, CanonicalSourceSHA256: a.Document.SourceSHA, Origin: o}
	next := 0
	newToken := func() string {
		v := fmt.Sprintf("⟪GTI18N_%s_%06d⟫", digest(append([]byte(uid), raw...))[:8], next)
		next++
		return v
	}
	var text strings.Builder
	add := func(value, region string, slot, atom, formal int, atomToken, prefix, suffix string) {
		v := surfaceTransportPart{Token: newToken(), Value: value, Region: region, Slot: slot, Atom: atom, FormalAtom: formal, AtomToken: atomToken, Prefix: prefix, Suffix: suffix}
		text.WriteString(v.Token)
		if region != "" {
			v.End = newToken()
			text.WriteString(value)
			text.WriteString(v.End)
		}
		p.Parts = append(p.Parts, v)
	}
	sort.Slice(boundTokens, func(i, j int) bool { return boundTokens[i].pos < boundTokens[j].pos })
	pos, current := 0, -1
	for _, b := range boundTokens {
		if b.pos < pos {
			return i18n.SurfaceTransportProjection{}, nil, fmt.Errorf("Surface overlapping source tokens")
		}
		id := ""
		if wanted[prose[current]] {
			id = prose[current]
		}
		add(string(raw[pos:b.pos]), id, current, -1, -1, "", "", "")
		if d, ok := descriptors[b.index]; ok && wanted[d.id] {
			add(d.word, d.id, b.slot, b.local, b.index, b.p.Token, d.prefix, d.suffix)
		} else {
			add(b.p.Token, "", b.slot, -1, b.index, "", "", "")
		}
		pos = b.pos + len(b.p.Token)
		if b.p.Role == "boundary" {
			current = b.slot
		}
	}
	add(string(raw[pos:]), "", -1, -1, -1, "", "", "")
	p.Template = text.String()
	ordinary := len(regions) > 0 && reflect.DeepEqual(o.Document, a.Document) && bytes.Equal(o.Source, a.Source)
	for _, d := range descriptors {
		if wanted[d.id] {
			ordinary = false
		}
	}
	if ordinary {
		p.Ordinary = true
		p.Parts = nil
		p.Template = string(raw)
	}
	codec, err := json.Marshal(p)
	if err != nil {
		return i18n.SurfaceTransportProjection{}, nil, err
	}
	editable := append([]string(nil), regions...)
	sort.Strings(editable)
	result := i18n.SurfaceTransportProjection{Kind: "surface-source-region-projection", Available: available, Editable: editable, Template: p.Template, Codec: codec}
	if ordinary {
		result.Kind = "ordinary"
	}
	if len(regions) == 0 {
		result.Template = ""
		result.Codec = nil
	}
	return result, p, nil
}

// This new, explicit revision-only source rule recognizes a natural-language
// descriptor beside an independently bound Go/API identity. It does not change
// canonical Page parsing, formal Unit identity or the original protected input.
func surfaceInlineDescriptor(s Unit, index int) (string, string, string, bool) {
	p := s.Protected[index]
	if p.Role != "atom" {
		return "", "", "", false
	}
	raw := p.Raw
	open, close := "", ""
	if strings.HasPrefix(raw, "`") && strings.HasSuffix(raw, "`") && !strings.Contains(raw[1:len(raw)-1], "`") {
		open, close = "`", "`"
	} else if strings.HasPrefix(raw, "<code>") && strings.HasSuffix(raw, "</code>") {
		open, close = "<code>", "</code>"
	} else {
		return "", "", "", false
	}
	body := raw[len(open) : len(raw)-len(close)]
	technical, word, ok := strings.Cut(body, " ")
	if !ok || (word != "method" && word != "function" && word != "package") {
		return "", "", "", false
	}
	for _, part := range strings.Split(technical, ".") {
		if !token.IsIdentifier(part) {
			return "", "", "", false
		}
	}
	bound := standardGoIdentity(technical)
	if word == "package" {
		bound = standardGoPackages[technical]
	}
	if !bound {
		stack := []string{}
		for _, q := range s.Protected[:index] {
			if q.Role == "open" {
				stack = append(stack, q.Pair)
			}
			if q.Role == "close" && len(stack) > 0 && stack[len(stack)-1] == q.Pair {
				stack = stack[:len(stack)-1]
			}
		}
		if len(stack) == 0 {
			return "", "", "", false
		}
		last := technical
		for _, part := range strings.Split(technical, ".") {
			last = part
		}
		for j := index + 1; j < len(s.Protected); j++ {
			q := s.Protected[j]
			if q.Role != "close" {
				continue
			}
			if q.Pair != stack[len(stack)-1] {
				return "", "", "", false
			}
			if q.Binding != nil && (strings.HasSuffix(q.Binding.Destination, "#"+last) || strings.HasSuffix(q.Binding.Destination, "."+last)) {
				bound = true
			}
			if strings.Contains(q.Raw, "#"+last+")") || strings.Contains(q.Raw, "."+last+")") {
				bound = true
			}
			break
		}
	}
	if !bound {
		return "", "", "", false
	}
	return open + technical + " ", word, close, true
}

func (a PageAdapter) ReadSurfaceTransportProjection(codec json.RawMessage) (i18n.TranslationUnitAdapter, error) {
	if len(a.Document.Units) != 1 {
		return nil, fmt.Errorf("Surface source adapter is not one formal Page")
	}
	var p surfacePageTransport
	if err := StrictJSON(codec, &p); err != nil {
		return nil, err
	}
	if p.Schema != "go-learning/surface-page-codec/v1" || p.UnitID != a.Document.Units[0].ID || p.CanonicalSourceSHA256 != a.Document.SourceSHA || (!p.Ordinary && len(p.Parts) == 0) {
		return nil, fmt.Errorf("Surface codec exact Unit/source mismatch")
	}
	return &p, nil
}
func (p *surfacePageTransport) ProtectedInput(*i18n.Glossary) (string, []string, error) {
	if p.Ordinary {
		_, tokens, err := p.Origin.ProtectedInput(nil)
		return p.Template, tokens, err
	}
	tokens := []string{}
	for _, q := range p.Parts {
		tokens = append(tokens, q.Token)
		if q.End != "" {
			tokens = append(tokens, q.End)
		}
	}
	return p.Template, tokens, nil
}
func (p *surfacePageTransport) Validate([]byte, *i18n.Glossary) error {
	return fmt.Errorf("Surface transport is not a formal validator")
}
func (p *surfacePageTransport) UnrestrictedRevisionTransport() bool { return p.Ordinary }
func (p *surfacePageTransport) ordinaryResponse(output string) (PageAdapter, string, error) {
	if p.Ordinary {
		return p.Origin, output, nil
	}
	output = strings.TrimSuffix(output, "\n")
	_, tokens, _ := p.ProtectedInput(nil)
	for _, t := range tokens {
		if strings.Count(output, t) != 1 {
			return PageAdapter{}, "", fmt.Errorf("Surface token missing/duplicate")
		}
	}
	// Copy all source-owned mappings; only an explicitly granted descriptor's
	// payload can differ. Canonical source/documents are never modified.
	var origin PageAdapter
	b, err := json.Marshal(p.Origin)
	if err != nil {
		return PageAdapter{}, "", err
	}
	if err := json.Unmarshal(b, &origin); err != nil {
		return PageAdapter{}, "", err
	}
	var ordinary strings.Builder
	for _, q := range p.Parts {
		if !strings.HasPrefix(output, q.Token) {
			return PageAdapter{}, "", fmt.Errorf("Surface opaque boundary moved or outside text")
		}
		output = output[len(q.Token):]
		if q.End == "" {
			ordinary.WriteString(q.Value)
			continue
		}
		end := strings.Index(output, q.End)
		if end < 0 {
			return PageAdapter{}, "", fmt.Errorf("Surface editable boundary missing")
		}
		v := output[:end]
		output = output[end+len(q.End):]
		if strings.ContainsAny(v, "⟪⟫⟦⟧\r\n\x00") {
			return PageAdapter{}, "", fmt.Errorf("Surface editable text crossed protected boundary")
		}
		if q.Atom < 0 {
			ordinary.WriteString(v)
			continue
		}
		if len(v) == 0 || len(v) > 256 {
			return PageAdapter{}, "", fmt.Errorf("Surface inline descriptor must be nonempty bounded prose")
		}
		for _, r := range v {
			if !unicode.IsLetter(r) && r != ' ' {
				return PageAdapter{}, "", fmt.Errorf("Surface descriptor cannot introduce code/markup")
			}
		}
		if q.Slot < 0 || q.Slot >= len(origin.Document.Slots) || q.Atom >= len(origin.Document.Slots[q.Slot].Protected) || q.FormalAtom < 0 || q.FormalAtom >= len(origin.Document.Units[0].Protected) {
			return PageAdapter{}, "", fmt.Errorf("Surface descriptor coordinate invalid")
		}
		raw := q.Prefix + v + q.Suffix
		origin.Document.Slots[q.Slot].Protected[q.Atom].Raw = raw
		origin.Document.Units[0].Protected[q.FormalAtom].Raw = raw
		ordinary.WriteString(q.AtomToken)
	}
	if output != "" {
		return PageAdapter{}, "", fmt.Errorf("Surface extra output text")
	}
	return origin, ordinary.String(), nil
}

func (p *surfacePageTransport) Restore(output string) ([]byte, error) {
	if p.Ordinary {
		return p.Origin.Restore(output)
	}
	origin, ordinary, err := p.ordinaryResponse(output)
	if err != nil {
		return nil, err
	}
	result, err := origin.Restore(ordinary)
	if err != nil {
		return nil, err
	}
	if err := p.Origin.Validate(result, &i18n.Glossary{}); err != nil {
		return nil, err
	}
	return bytes.Clone(result), nil
}
