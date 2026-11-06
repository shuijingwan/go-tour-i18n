package sitecontent

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func exactLiteralSites(raw []byte, left, right int, literal string) []int {
	positions := []int{}
	identity := func(b byte) bool {
		return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_'
	}
	for at := left; at < right; {
		n := strings.Index(string(raw[at:right]), literal)
		if n < 0 {
			break
		}
		at += n
		if (at == left || !identity(raw[at-1])) && (at+len(literal) == right || !identity(raw[at+len(literal)])) {
			positions = append(positions, at)
		}
		at += len(literal)
	}
	return positions
}

func (a PageAdapter) UnbindableRevision(adapter i18n.TranslationUnitAdapter, raw []byte) (*i18n.RepresentationBindingFailure, []byte, error) {
	old, ok := adapter.(PageAdapter)
	if !ok || !bytes.Equal(old.Source, a.Source) {
		return nil, nil, fmt.Errorf("revision predecessor source mismatch")
	}
	if _, err := old.Restore(string(raw)); err != nil {
		return nil, nil, err
	}
	op, err := protectionPieces(old.Document.Units[0])
	if err != nil {
		return nil, nil, err
	}
	np, err := protectionPieces(a.Document.Units[0])
	if err != nil {
		return nil, nil, err
	}
	proof := &i18n.RepresentationBindingFailure{Kind: "duplicate-prose-literal", Intervals: []i18n.RepresentationAmbiguousInterval{}}
	type patch struct {
		start, end int
		source     string
	}
	patches := []patch{}
	for i, p := range op {
		if p.token != nil {
			continue
		}
		groups := map[string][]protectedPiece{}
		for _, q := range np {
			if q.token != nil && q.start >= p.start && q.end <= p.end {
				if q.token.Role != "atom" || q.token.Binding != nil || q.token.Pair != "" && !strings.HasSuffix(q.token.Pair, "/") {
					return nil, nil, fmt.Errorf("machine revision cannot select paired/structural/code identities")
				}
				if q.token.Raw == "" {
					return nil, nil, fmt.Errorf("empty literal")
				}
				for _, r := range q.token.Raw {
					if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
						return nil, nil, fmt.Errorf("machine revision only supports new plain-prose identifiers")
					}
				}
				groups[q.token.Raw] = append(groups[q.token.Raw], q)
			}
		}
		if len(groups) == 0 {
			continue
		}
		left, right := 0, len(raw)
		la, ra := "", ""
		if i > 0 {
			la = op[i-1].text
			left = strings.Index(string(raw), la) + len(la)
		}
		if i+1 < len(op) {
			ra = op[i+1].text
			right = strings.Index(string(raw), ra)
		}
		if left < 0 || right < left {
			return nil, nil, fmt.Errorf("revision anchors reordered")
		}
		interval := i18n.RepresentationAmbiguousInterval{SourceStart: p.start, SourceEnd: p.end, SourceText: p.text, ReviewedTarget: string(raw[left:right]), LeftAnchor: la, RightAnchor: ra, Atoms: []i18n.RepresentationAmbiguousAtom{}}
		duplicate := false
		literals := []string{}
		for literal := range groups {
			literals = append(literals, literal)
		}
		sort.Strings(literals)
		for _, literal := range literals {
			sites := groups[literal]
			positions := exactLiteralSites(raw, left, right, literal)
			if len(positions) != len(sites) && (len(sites) != 1 || len(positions) < 2) {
				return nil, nil, fmt.Errorf("unsupported missing literal or distinct source sites; no duplicate-site machine authority")
			}
			if len(positions) > len(sites) {
				duplicate = true
			}
			for _, q := range sites {
				interval.Atoms = append(interval.Atoms, i18n.RepresentationAmbiguousAtom{Literal: literal, SourceStart: q.start, SourceEnd: q.end, TargetSites: positions})
			}
		}
		if duplicate {
			proof.Intervals = append(proof.Intervals, interval)
			patches = append(patches, patch{left, right, p.text})
		}
	}
	if len(patches) == 0 {
		return nil, nil, fmt.Errorf("no duplicate-prose-literal proof; do not invent a machine finding")
	}
	sort.Slice(patches, func(i, j int) bool { return patches[i].start < patches[j].start })
	var scaffold strings.Builder
	pos := 0
	for _, p := range patches {
		if p.start < pos {
			return nil, nil, fmt.Errorf("overlapping revision intervals")
		}
		scaffold.Write(raw[pos:p.start])
		scaffold.WriteString(p.source)
		pos = p.end
	}
	scaffold.Write(raw[pos:])
	// No target occurrence is selected. Every unresolved interval is replaced
	// wholesale by its exact source text; all outside prose/bindings stay frozen.
	derived, err := a.ReconcileRepresentation(old, []byte(scaffold.String()))
	if err != nil {
		return nil, nil, fmt.Errorf("outside-interval proof: %w", err)
	}
	return proof, derived, nil
}
