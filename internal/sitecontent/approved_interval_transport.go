package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
)

type approvedRawRange struct{ Start, End int }
type approvedPageSlot struct {
	Start, End int      `json:"-"`
	RawStart   int      `json:"raw_start"`
	RawEnd     int      `json:"raw_end"`
	Text       string   `json:"text"`
	Tokens     []string `json:"tokens"`
	OldTokens  []string `json:"old_tokens"`
}
type approvedPageCodec struct {
	Schema      string             `json:"schema"`
	Old         PageAdapter        `json:"historical_decoder"`
	Baseline    string             `json:"historical_raw"`
	Boundaries  []string           `json:"boundaries"`
	Values      []string           `json:"frozen_raw_values"`
	Slots       []approvedPageSlot `json:"slots"`
	ParentToken string             `json:"parent_token"`
	Text        string             `json:"text"`
	Tokens      []string           `json:"tokens"`
}

func (p *approvedPageCodec) ProtectedInput(*i18n.Glossary) (string, []string, error) {
	return p.Text, append([]string(nil), p.Tokens...), nil
}
func (p *approvedPageCodec) Validate([]byte, *i18n.Glossary) error {
	return fmt.Errorf("approved interval response transport is not a validator")
}
func (p *approvedPageCodec) Restore(output string) ([]byte, error) {
	output = strings.TrimSuffix(output, "\n")
	if strings.Contains(output, "⟦") || strings.Contains(output, "⟧") {
		return nil, fmt.Errorf("approved interval normal P tokens cannot acquire projection semantics")
	}
	remain := output
	for _, t := range p.Tokens {
		if strings.Count(output, t) != 1 {
			return nil, fmt.Errorf("approved interval token missing/duplicate")
		}
		remain = strings.ReplaceAll(remain, t, "")
	}
	if strings.ContainsAny(remain, "⟪⟫") {
		return nil, fmt.Errorf("approved interval extra projection token")
	}
	if !strings.HasPrefix(output, p.Boundaries[0]) || !strings.HasSuffix(output, p.Boundaries[len(p.Boundaries)-1]) {
		return nil, fmt.Errorf("approved interval edits outside finding footprint")
	}
	var raw strings.Builder
	for i, s := range p.Slots {
		a := strings.Index(output, p.Boundaries[i]) + len(p.Boundaries[i])
		z := strings.Index(output, p.Boundaries[i+1])
		if a > z {
			return nil, fmt.Errorf("approved interval boundary order changed")
		}
		text := output[a:z]
		pairs := []string{}
		for j, t := range s.Tokens {
			if strings.Count(text, t) != 1 {
				return nil, fmt.Errorf("approved interval child moved across finding region")
			}
			pairs = append(pairs, t, s.OldTokens[j])
		}
		if len(pairs) > 0 {
			text = strings.NewReplacer(pairs...).Replace(text)
		}
		raw.WriteString(p.Values[i])
		raw.WriteString(text)
	}
	raw.WriteString(p.Values[len(p.Values)-1])
	// The legitimate saved historical decoder owns source binding/structure,
	// including all unchanged raw tokens. No candidate re-protection occurs.
	return p.Old.Restore(raw.String())
}

type approvedExpansion struct {
	rawStart, rawEnd, targetStart, targetEnd int
	token                                    *ProtectedToken
}

func approvedExpansionTrace(old PageAdapter, raw, candidate []byte) ([]approvedExpansion, error) {
	tokens := old.Document.Units[0].Protected
	positions := []struct {
		at int
		t  ProtectedToken
	}{}
	for _, t := range tokens {
		if strings.Count(string(raw), t.Token) != 1 {
			return nil, fmt.Errorf("historical token inventory not exact")
		}
		positions = append(positions, struct {
			at int
			t  ProtectedToken
		}{strings.Index(string(raw), t.Token), t})
	}
	sort.Slice(positions, func(i, j int) bool { return positions[i].at < positions[j].at })
	var expanded bytes.Buffer
	trace := []approvedExpansion{}
	pos := 0
	for _, p := range positions {
		if p.at < pos {
			return nil, fmt.Errorf("historical token overlap")
		}
		if p.at > pos {
			a := expanded.Len()
			expanded.Write(raw[pos:p.at])
			trace = append(trace, approvedExpansion{pos, p.at, a, expanded.Len(), nil})
		}
		a := expanded.Len()
		expanded.WriteString(p.t.Raw)
		t := p.t
		trace = append(trace, approvedExpansion{p.at, p.at + len(p.t.Token), a, expanded.Len(), &t})
		pos = p.at + len(p.t.Token)
	}
	if pos < len(raw) {
		a := expanded.Len()
		expanded.Write(raw[pos:])
		trace = append(trace, approvedExpansion{pos, len(raw), a, expanded.Len(), nil})
	}
	if !bytes.Equal(bridgeEOF(expanded.Bytes()), bridgeEOF(candidate)) {
		return nil, fmt.Errorf("historical expansion coordinate conversion is not exact; missing codec cannot be guessed")
	}
	return trace, nil
}
func approvedRawCoordinate(trace []approvedExpansion, at int, end bool) (int, int, error) {
	for _, p := range trace {
		if at < p.targetStart || at > p.targetEnd {
			continue
		}
		if p.token == nil {
			return p.rawStart + at - p.targetStart, at, nil
		}
		if at == p.targetStart {
			return p.rawStart, at, nil
		}
		if at == p.targetEnd {
			return p.rawEnd, at, nil
		}
		// Conservatively include the complete immutable source token. Never split
		// it or create a target leaf site; expansion may only enlarge edit footprint.
		if end {
			return p.rawEnd, p.targetEnd, nil
		}
		return p.rawStart, p.targetStart, nil
	}
	return 0, 0, fmt.Errorf("approved interval candidate/raw coordinate conversion unknown")
}

func (a PageAdapter) ApprovedIntervalProjection(adapter i18n.TranslationUnitAdapter, raw, candidate []byte, q i18n.RepresentationSemanticQuestion, audit languageaudit.Result) (i18n.ApprovedIntervalProjection, i18n.TranslationUnitAdapter, error) {
	var empty i18n.ApprovedIntervalProjection
	old, ok := adapter.(PageAdapter)
	if !ok || !bytes.Equal(a.Source, old.Source) {
		return empty, nil, fmt.Errorf("approved interval frozen source adapter mismatch")
	}
	if len(a.Document.Units) != 1 || len(old.Document.Units) != 1 {
		return empty, nil, fmt.Errorf("approved interval requires complete Page")
	}
	// Prove the saved whole question from source coordinates, not target words.
	var oldSaved, currentSaved Document
	if StrictJSON(q.OldRepresentation, &oldSaved) != nil || StrictJSON(q.CurrentRepresentation, &currentSaved) != nil {
		return empty, nil, fmt.Errorf("approved interval saved representation missing")
	}
	if !reflect.DeepEqual(oldSaved, old.Document) || currentSaved.Units[0].ID != a.Document.Units[0].ID || !reflect.DeepEqual(currentSaved.Units[0], a.Document.Units[0]) {
		return empty, nil, fmt.Errorf("approved interval child inventory/representation drift")
	}
	op, err := protectionPieces(old.Document.Units[0])
	if err != nil {
		return empty, nil, err
	}
	np, err := protectionPieces(a.Document.Units[0])
	if err != nil {
		return empty, nil, err
	}
	parentFound := false
	for _, p := range op {
		if p.token == nil && p.start == q.SourceStart && p.end == q.SourceEnd && p.text == q.SourceText {
			parentFound = true
		}
	}
	if !parentFound {
		return empty, nil, fmt.Errorf("approved interval is not the complete saved prose interval")
	}
	children := []map[string]any{}
	for _, p := range np {
		if p.token == nil {
			continue
		}
		if p.start >= q.SourceStart && p.end <= q.SourceEnd {
			children = append(children, map[string]any{"source_start": p.start, "source_end": p.end, "identity": p.token.Token, "role": p.token.Role, "protection": p.token})
			continue
		}
		// Outside the whole parent every current protected source child must be
		// carried by an existing immutable historical source token, never target inference.
		covered := false
		for _, before := range op {
			if before.token != nil && p.start >= before.start && p.end <= before.end && bytes.Equal(a.Source[p.start:p.end], old.Source[p.start:p.end]) {
				covered = true
			}
		}
		if !covered {
			return empty, nil, fmt.Errorf("approved interval unproved protection outside parent")
		}
	}
	if len(children) == 0 {
		return empty, nil, fmt.Errorf("approved interval no current protected child inventory")
	}
	// Exact saved old anchors select the WHOLE parent only, not a target atom.
	if q.LeftAnchor == "" || q.RightAnchor == "" || strings.Count(string(raw), q.LeftAnchor) != 1 || strings.Count(string(raw), q.RightAnchor) != 1 {
		return empty, nil, fmt.Errorf("approved interval whole parent anchors ambiguous")
	}
	parentRawStart := strings.Index(string(raw), q.LeftAnchor) + len(q.LeftAnchor)
	parentRawEnd := strings.Index(string(raw), q.RightAnchor)
	if parentRawStart > parentRawEnd || string(raw[parentRawStart:parentRawEnd]) != q.ReviewedTarget {
		return empty, nil, fmt.Errorf("approved interval whole frozen target differs")
	}
	trace, err := approvedExpansionTrace(old, raw, candidate)
	if err != nil {
		return empty, nil, err
	}
	parentStart, parentEnd := -1, -1
	for _, p := range trace {
		if p.token == nil && parentRawStart >= p.rawStart && parentRawEnd <= p.rawEnd {
			parentStart = p.targetStart + parentRawStart - p.rawStart
			parentEnd = p.targetStart + parentRawEnd - p.rawStart
		}
	}
	if parentStart < 0 || !bytes.Equal(candidate[parentStart:parentEnd], []byte(q.ReviewedTarget)) {
		return empty, nil, fmt.Errorf("approved interval parent inverse coordinates unknown")
	}
	regions, err := a.approvedFindingCoordinates(candidate, audit)
	if err != nil {
		return empty, nil, err
	}
	ranges := []approvedRawRange{}
	for i := range regions {
		region := &regions[i]
		rs, ts, err := approvedRawCoordinate(trace, region.Candidate.Start, false)
		if err != nil {
			return empty, nil, err
		}
		re, te, err := approvedRawCoordinate(trace, region.Candidate.End, true)
		if err != nil {
			return empty, nil, err
		}
		if ts < parentEnd && te > parentStart {
			return empty, nil, fmt.Errorf("approved interval finding/opaque parent overlap")
		}
		region.Candidate.Start, region.Candidate.End = ts, te
		ranges = append(ranges, approvedRawRange{rs, re})
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].Start < ranges[j].Start })
	merged := []approvedRawRange{}
	for _, r := range ranges {
		if r.End <= r.Start {
			return empty, nil, fmt.Errorf("approved interval incomplete editable footprint")
		}
		if len(merged) > 0 && r.Start <= merged[len(merged)-1].End {
			if r.End > merged[len(merged)-1].End {
				merged[len(merged)-1].End = r.End
			}
		} else {
			merged = append(merged, r)
		}
	}
	codec := &approvedPageCodec{Schema: "go-learning/approved-interval-response/v1", Old: old, Baseline: string(raw)}
	seed := digest([]byte(identity(a.Document) + identity(old.Document) + digest(candidate) + identity(q) + identity(regions)))
	next := 0
	fresh := func() string { next++; return fmt.Sprintf("⟪GTI18N_AI1_%s_%06d⟫", seed[:16], next) }
	pos := 0
	for _, region := range merged {
		boundary := fresh()
		codec.Boundaries = append(codec.Boundaries, boundary)
		codec.Values = append(codec.Values, string(raw[pos:region.Start]))
		codec.Tokens = append(codec.Tokens, boundary)
		codec.Text += boundary
		s := approvedPageSlot{RawStart: region.Start, RawEnd: region.End, Text: string(raw[region.Start:region.End])}
		pairs := []string{}
		for _, p := range trace {
			if p.token != nil && p.rawStart >= region.Start && p.rawEnd <= region.End {
				token := fresh()
				s.Tokens = append(s.Tokens, token)
				s.OldTokens = append(s.OldTokens, p.token.Token)
				codec.Tokens = append(codec.Tokens, token)
				pairs = append(pairs, p.token.Token, token)
			}
		}
		if len(pairs) > 0 {
			s.Text = strings.NewReplacer(pairs...).Replace(s.Text)
		}
		codec.Text += s.Text
		codec.Slots = append(codec.Slots, s)
		pos = region.End
	}
	boundary := fresh()
	codec.Boundaries = append(codec.Boundaries, boundary)
	codec.Values = append(codec.Values, string(raw[pos:]))
	codec.Tokens = append(codec.Tokens, boundary)
	codec.Text += boundary
	// Every frozen chunk is opaque, including the complete parent. Bind its
	// owning boundary explicitly; no child target position is invented.
	for i, value := range codec.Values {
		start := 0
		if i > 0 {
			start = merged[i-1].End
		}
		end := start + len(value)
		if parentRawStart >= start && parentRawEnd <= end {
			codec.ParentToken = codec.Boundaries[i]
		}
	}
	if codec.ParentToken == "" {
		return empty, nil, fmt.Errorf("approved interval parent is not opaque")
	}
	got, err := codec.Restore(codec.Text)
	if err != nil || !bytes.Equal(bridgeEOF(got), bridgeEOF(candidate)) {
		return empty, nil, fmt.Errorf("approved interval baseline restore mismatch: %v", err)
	}
	childJSON, _ := json.Marshal(children)
	encoded, _ := json.Marshal(codec)
	p := i18n.ApprovedIntervalProjection{Namespace: "GTI18N_AI1/v1", UnitID: a.Document.Units[0].ID, SourceSHA256: digest(a.Source), OldRepresentationSHA256: identity(old.Document), CurrentRepresentationSHA256: identity(a.Document), CandidateSHA256: digest(candidate), ParentSource: i18n.ApprovedIntervalCoordinate{Space: "canonical-source", Unit: "byte", Start: q.SourceStart, End: q.SourceEnd}, ParentCandidate: i18n.ApprovedIntervalCoordinate{Space: "candidate-markdown", Unit: "byte", Start: parentStart, End: parentEnd}, ParentSHA256: digest([]byte(q.ReviewedTarget)), ParentToken: codec.ParentToken, Children: childJSON, Findings: regions, Text: bridgeStringEOF(codec.Text), Tokens: codec.Tokens, Codec: encoded}
	return p, codec, nil
}
func bridgeStringEOF(s string) string { return string(bridgeEOF([]byte(s))) }
