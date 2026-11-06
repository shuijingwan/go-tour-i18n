package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Rebuild an immutable adopted corpus from its own exact projection authorities,
// not today's source heads. Only canonical pointer bytes derivable from those
// immutable references are recovered; this never grants current eligibility.
func ReadHistoricalAdoptedCorpusSuccessor(root string, ref Reference, contexts ...*structuredSuccessorValidation) (*CorpusSuccessor, error) {
	v := successorValidation(root, contexts)
	var p CorpusSuccessor
	if err := v.guard.readReference(root, ref, &p); err != nil {
		return nil, err
	}
	if err := v.observe(p); err != nil {
		return nil, err
	}
	copy := p
	copy.Identity = ""
	canonical, _ := Encode(p)
	raw, err := readRegularReference(root, ref)
	if err != nil || p.Schema != CorpusSuccessorSchema || p.State != "pending" || p.Identity != identity(copy) || ref.Path != CorpusSuccessorPath(&p) || !bytes.Equal(canonical, raw) || len(p.ProjectionAuthorities) == 0 {
		return nil, fmt.Errorf("invalid historical adopted corpus")
	}
	oldPath := "data/glossary-corpus-history/" + p.Predecessor.SHA256 + ".json"
	if err := v.guard.bind(Reference{oldPath, p.Predecessor.SHA256}); err != nil {
		return nil, err
	}
	oldBytes, err := readRegularReference(root, Reference{oldPath, p.Predecessor.SHA256})
	if err != nil {
		return nil, err
	}
	old, err := readProjectionCorpus(root, oldPath)
	if err != nil {
		return nil, err
	}
	g := &Global{}
	for _, r := range p.ProjectionAuthorities {
		b, err := readRegularReference(root, r)
		if err != nil {
			return nil, err
		}
		var h struct {
			Schema string `json:"schema"`
		}
		if err := json.Unmarshal(b, &h); err != nil {
			return nil, err
		}
		switch h.Schema {
		case PagePackageAdoptionSchema:
			g.PackageAdoptions = append(g.PackageAdoptions, r)
		case StructuredSourceSuccessorSchema:
			g.StructuredSuccessors = append(g.StructuredSuccessors, r)
		default:
			return nil, fmt.Errorf("unknown historical corpus projection authority")
		}
	}
	want, err := buildAdoptedCorpusAtSources(root, g, old, oldBytes, true, v)
	if err != nil || !reflect.DeepEqual(p, *want) {
		return nil, fmt.Errorf("historical corpus projection mismatch: %v", err)
	}
	if err := v.observe(p); err != nil {
		return nil, err
	}
	if err := v.guard.bind(Reference{oldPath, p.Predecessor.SHA256}); err != nil {
		return nil, err
	}
	return &p, nil
}

// Source-governance/historical projection input only. The enclosing projection
// verifies every resulting physical reference, with old pointer bytes recovered
// exclusively from exact immutable source authorities. Language eligibility
// continues to use i18n.LoadUnifiedGlossaryCorpus/RequireUnifiedGlossaryReview.
func readProjectionCorpus(root, name string) (*i18n.GlossarySourceCorpus, error) {
	b, err := readRegular(root, name)
	if err != nil {
		return nil, err
	}
	var c i18n.GlossarySourceCorpus
	if err := StrictJSON(b, &c); err != nil {
		return nil, err
	}
	if err := i18n.ValidateGlossaryCorpus(&c); err != nil {
		return nil, err
	}
	last := ""
	for _, r := range c.Sources {
		if r.Path <= last || !validPath(r.Path) || !validSHA(r.SHA256) {
			return nil, fmt.Errorf("invalid corpus source references")
		}
		last = r.Path
	}
	return &c, nil
}
