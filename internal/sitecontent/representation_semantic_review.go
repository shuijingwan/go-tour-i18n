package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// The question is an entire frozen prose interval, never a guessed Chinese
// substring or a selected occurrence of a repeated technical literal.
func (a PageAdapter) ReconciliationQuestion(adapter i18n.TranslationUnitAdapter, raw []byte) (*i18n.RepresentationSemanticQuestion, error) {
	old, ok := adapter.(PageAdapter)
	if !ok || !bytes.Equal(old.Source, a.Source) {
		return nil, fmt.Errorf("semantic source mismatch")
	}
	if _, err := old.Restore(string(raw)); err != nil {
		return nil, err
	}
	op, err := protectionPieces(old.Document.Units[0])
	if err != nil {
		return nil, err
	}
	np, err := protectionPieces(a.Document.Units[0])
	if err != nil {
		return nil, err
	}
	var question *i18n.RepresentationSemanticQuestion
	for i, p := range op {
		if p.token != nil {
			continue
		}
		atoms := []ProtectedToken{}
		for _, q := range np {
			if q.token == nil || q.start < p.start || q.end > p.end {
				continue
			}
			if q.token.Role != "atom" || q.token.Binding != nil {
				return nil, fmt.Errorf("semantic authority cannot release structural/URL/code bindings")
			}
			// A nonliteral semantic question is permitted only for new prose
			// identifiers, not source markup/code, flags, paths or destinations.
			for _, r := range q.token.Raw {
				if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
					return nil, fmt.Errorf("new atom is not a prose identifier")
				}
			}
			atoms = append(atoms, *q.token)
		}
		if len(atoms) == 0 {
			continue
		}
		left, right := "", ""
		if i > 0 {
			left = op[i-1].text
		}
		if i+1 < len(op) {
			right = op[i+1].text
		}
		start, end := 0, len(raw)
		if left != "" {
			start = strings.Index(string(raw), left) + len(left)
		}
		if right != "" {
			end = strings.Index(string(raw), right)
		}
		if start < 0 || end < start {
			return nil, fmt.Errorf("semantic interval anchors reordered")
		}
		target := string(raw[start:end])
		missing := false
		for _, atom := range atoms {
			if !strings.Contains(target, atom.Raw) {
				missing = true
			}
		}
		if !missing {
			continue
		} // Repeated literal ambiguity remains machine-blocked.
		if question != nil {
			return nil, fmt.Errorf("multiple unresolved semantic intervals require separate review scopes")
		}
		changes, _ := json.Marshal(atoms)
		type coordinate struct {
			Start      int             `json:"start"`
			End        int             `json:"end"`
			Text       string          `json:"text"`
			Protection *ProtectedToken `json:"protection,omitempty"`
		}
		inventory := func(pieces []protectedPiece) []coordinate {
			out := []coordinate{}
			for _, p := range pieces {
				out = append(out, coordinate{p.start, p.end, p.text, p.token})
			}
			return out
		}
		allChanges, _ := json.Marshal(struct {
			Old     []coordinate `json:"old_source_coordinates"`
			Current []coordinate `json:"current_source_coordinates"`
		}{inventory(op), inventory(np)})
		before, _ := json.Marshal(old.Document)
		after, _ := json.Marshal(a.Document)
		question = &i18n.RepresentationSemanticQuestion{SourceStart: p.start, SourceEnd: p.end, SourceText: p.text, ReviewedTarget: target, LeftAnchor: left, RightAnchor: right, ChangedAtoms: changes, RepresentationChanges: allChanges, OldRepresentation: before, CurrentRepresentation: after}
	}
	if question == nil {
		return nil, fmt.Errorf("no uniquely scoped nonliteral semantic question; repeated target bindings remain unresolved")
	}
	projection, err := a.semanticProjection(*question)
	if err != nil {
		return nil, err
	}
	// All bindings OUTSIDE this question must already have deterministic proof.
	if _, err = projection.ReconcileRepresentation(old, raw); err != nil {
		return nil, fmt.Errorf("outside-question machine proof: %w", err)
	}
	return question, nil
}

// The projection is private machine evidence, with its own protected inventory.
// It is never registered as canonical source or passed off as a model response.
func (a PageAdapter) semanticProjection(q i18n.RepresentationSemanticQuestion) (PageAdapter, error) {
	d := a.Document
	d.Slots = append([]Unit(nil), d.Slots...)
	removed := 0
	for i, s := range d.Slots {
		pieces, err := protectionPieces(s)
		if err != nil {
			return PageAdapter{}, err
		}
		keep := []ProtectedToken{}
		for _, p := range pieces {
			if p.token == nil {
				continue
			}
			if s.Start+p.start >= q.SourceStart && s.Start+p.end <= q.SourceEnd {
				if p.token.Role != "atom" || p.token.Binding != nil {
					return PageAdapter{}, fmt.Errorf("semantic projection cannot remove a structural binding")
				}
				s.Source = strings.Replace(s.Source, p.text, p.token.Raw, 1)
				removed++
			} else {
				keep = append(keep, *p.token)
			}
		}
		s.Protected = keep
		s.SourceSHA = unitSourceIdentity(s.Source, s.Protected)
		d.Slots[i] = s
	}
	if removed == 0 {
		return PageAdapter{}, fmt.Errorf("semantic projection has no scoped identifier")
	}
	d.Units = d.Slots
	next, err := FormalPage(d, a.Source, d.StableIndex)
	if err != nil {
		return PageAdapter{}, err
	}
	return PageAdapter{next, a.Source}, nil
}

func (a PageAdapter) ReviewedReconciliation(old i18n.TranslationUnitAdapter, raw []byte, q i18n.RepresentationSemanticQuestion) ([]byte, []byte, error) {
	want, err := a.ReconciliationQuestion(old, raw)
	if err != nil {
		return nil, nil, err
	}
	if !reflect.DeepEqual(q, *want) {
		return nil, nil, fmt.Errorf("reviewed semantic question changed")
	}
	p, err := a.semanticProjection(q)
	if err != nil {
		return nil, nil, err
	}
	next, err := p.ReconcileRepresentation(old, raw)
	if err != nil {
		return nil, nil, err
	}
	restored, err := p.Restore(string(next))
	return next, restored, err
}
