package sitecontent

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const PageContract = "go-learning/page-translation-unit/v1"

// Internal slots are restoration/impact coordinates. Exactly one Page unit is
// exposed for a complete physical Page and exactly one rating is recorded.
func FormalPage(d Document, raw []byte, index int) (Document, error) {
	if d.Kind != "page" || len(d.Units) == 0 {
		return Document{}, fmt.Errorf("formal TU requires language-bearing Page")
	}
	slots := d.Units
	locked := []ProtectedToken{}
	var text strings.Builder
	pos := 0
	add := func(raw, role, pair string) string {
		token := "⟦P" + strconv.Itoa(len(locked)) + "⟧"
		locked = append(locked, ProtectedToken{Token: token, Raw: raw, Role: role, Pair: pair})
		return token
	}
	for si, s := range slots {
		text.WriteString(add(string(raw[pos:s.Start]), "boundary", ""))
		value := s.Source
		replacements := []string{}
		for _, p := range s.Protected {
			role := p.Role
			if role == "suffix" {
				role = "atom"
			}
			token := add(p.Raw, role, fmt.Sprintf("slot%d/%s", si, p.Pair))
			replacements = append(replacements, p.Token, token)
		}
		if len(replacements) > 0 {
			value = strings.NewReplacer(replacements...).Replace(value)
		}
		text.WriteString(value)
		pos = s.End
	}
	text.WriteString(add(string(raw[pos:]), "boundary", ""))
	source := text.String()
	sh := unitSourceIdentity(source, locked)
	locator := d.Path + "#page:" + sh
	d.Slots = slots
	d.StableIndex = index
	d.UnitScope = PageContract
	d.Units = []Unit{{ID: digest([]byte(UnitContract + "\n" + locator)), Locator: locator, Source: source, SourceSHA: sh, Start: 0, End: len(raw), Protected: locked}}
	return d, nil
}
func ParseWorkflowDocument(s Source, b []byte, index int) (*Document, error) {
	d, err := ParseDocument(s, b)
	if err != nil {
		return nil, err
	}
	v, err := FormalPage(*d, b, index)
	return &v, err
}
func WorkflowDocuments(root string, g *Global, pkg string) ([]Document, map[string][]byte, error) {
	docs, raw, err := PackageDocuments(root, g, pkg)
	if err != nil {
		return nil, nil, err
	}
	out := []Document{}
	indexes := map[string]int{}
	if pkg == "learn-docs-v1" {
		r, err := LoadPageRegistry(root, g)
		if err != nil {
			return nil, nil, err
		}
		for _, p := range r.Pages {
			if !p.Retired {
				indexes[p.Path] = p.Index
			}
		}
	}
	for _, d := range docs {
		if d.Kind != "page" {
			continue
		}
		v, err := FormalPage(d, raw[d.Path], indexes[d.Path])
		if err != nil {
			return nil, nil, err
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StableIndex != out[j].StableIndex {
			return out[i].StableIndex < out[j].StableIndex
		}
		return out[i].Path < out[j].Path
	})
	return out, raw, nil
}
func ReconstructWorkflow(d Document, raw []byte, targets map[string]string) ([]byte, error) {
	if d.UnitScope != PageContract {
		return Reconstruct(documentSource(d), raw, targets)
	}
	if len(d.Units) != 1 || len(targets) != 1 {
		return nil, fmt.Errorf("Page target exact-set")
	}
	u := d.Units[0]
	target, ok := targets[u.ID]
	if !ok {
		return nil, fmt.Errorf("missing Page target")
	}
	if _, err := restoreBlock(u, target); err != nil {
		return nil, err
	}
	boundaries := []ProtectedToken{}
	for _, p := range u.Protected {
		if p.Role == "boundary" {
			boundaries = append(boundaries, p)
		}
	}
	if len(boundaries) != len(d.Slots)+1 {
		return nil, fmt.Errorf("Page slot boundary mismatch")
	}
	values := map[string]string{}
	for i, s := range d.Slots {
		a := strings.Index(target, boundaries[i].Token) + len(boundaries[i].Token)
		z := strings.Index(target, boundaries[i+1].Token)
		if a > z {
			return nil, fmt.Errorf("Page block order changed")
		}
		value := target[a:z]
		// Restore this block independently to retain heading suffix and link binding.
		replacements := []string{}
		at := 0
		for j, p := range u.Protected {
			if p.Token == boundaries[i].Token {
				at = j + 1
				break
			}
		}
		for j, old := range s.Protected {
			q := u.Protected[at+j]
			if q.Raw != old.Raw {
				return nil, fmt.Errorf("Page protected slot binding mismatch")
			}
			replacements = append(replacements, q.Token, old.Token)
		}
		if len(replacements) > 0 {
			value = strings.NewReplacer(replacements...).Replace(value)
		}
		values[s.ID] = value
	}
	if !strings.HasPrefix(target, boundaries[0].Token) || !strings.HasSuffix(target, boundaries[len(boundaries)-1].Token) {
		return nil, fmt.Errorf("text outside Page blocks")
	}
	return Reconstruct(documentSource(d), raw, values)
}
