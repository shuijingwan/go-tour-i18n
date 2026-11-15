package i18n

import (
	"bytes"
	"fmt"
	"strings"
)

const attemptWhitespaceSchema = "go-tour-i18n/protected-structural-whitespace-attempt/v1"
const attemptWhitespaceMember = "formal/whitespace-attempt/codec.json"
const attemptWhitespacePrefix = "formal/whitespace-attempt/"

// Source-owned coordinates; these are not a general validator repair model.
type StructuralWhitespaceOrigin struct {
	Token       string `json:"token"`
	RawSHA256   string `json:"raw_sha256"`
	SourceStart int    `json:"source_start"`
	SourceEnd   int    `json:"source_end"`
	TokenStart  int    `json:"token_start"`
	TokenEnd    int    `json:"token_end"`
	Owner       string `json:"owner"`
}

type StructuralWhitespaceRepair struct {
	Line       int                          `json:"line"`
	Start      int                          `json:"start"`
	End        int                          `json:"end"`
	Whitespace string                       `json:"whitespace"`
	OpaqueLine bool                         `json:"opaque_line"`
	Origins    []StructuralWhitespaceOrigin `json:"origins"`
}

// Implementations must prove exact expansion and exclude code, literals,
// links, attributes, directives and structural boundaries from editable bytes.
type StructuralWhitespaceInspector interface {
	InspectStructuralWhitespace([]byte) ([]StructuralWhitespaceRepair, string, error)
}

type attemptWhitespaceCodec struct {
	Schema                   string                       `json:"schema"`
	Locale                   string                       `json:"locale"`
	BatchID                  string                       `json:"batch_id"`
	Attempt                  int                          `json:"attempt"`
	Record                   RetranslationBatchUnit       `json:"record"`
	ManifestSHA256           string                       `json:"manifest_sha256"`
	GlossarySHA256           string                       `json:"glossary_sha256"`
	OriginalRepresentation   string                       `json:"original_representation_sha256"`
	OriginImport             GlossaryArchiveReference     `json:"origin_import"`
	OriginArchive            GlossaryArchiveReference     `json:"origin_archive"`
	PreviousImport           GlossaryArchiveReference     `json:"previous_import"`
	PreviousArchive          GlossaryArchiveReference     `json:"previous_archive"`
	OriginalRawSHA256        string                       `json:"original_raw_sha256"`
	OriginalCandidateSHA256  string                       `json:"original_candidate_sha256"`
	OriginalValidationSHA256 string                       `json:"original_validation_sha256"`
	PreviousRawSHA256        string                       `json:"previous_raw_sha256"`
	PreviousValidationSHA256 string                       `json:"previous_validation_sha256"`
	Repairs                  []StructuralWhitespaceRepair `json:"repairs"`
	RepairedCandidateSHA256  string                       `json:"repaired_candidate_sha256"`
	Identity                 string                       `json:"identity_sha256"`
}

func (c attemptWhitespaceCodec) identity() string {
	c.Identity = ""
	return sum(mustJSON(c))
}

// All target-language bytes are opaque copied slices. Only the exact original
// trailing ASCII whitespace or its deletion is representable in each slot.
type attemptWhitespaceResponse struct {
	codec     attemptWhitespaceCodec
	candidate []byte
	segments  []string
	tokens    []string
	adapter   TranslationUnitAdapter
}

func whitespaceResponse(c attemptWhitespaceCodec, candidate []byte, adapter TranslationUnitAdapter) (*attemptWhitespaceResponse, error) {
	if c.Schema != attemptWhitespaceSchema || c.Identity != c.identity() || sum(candidate) != c.OriginalCandidateSHA256 || len(c.Repairs) == 0 {
		return nil, fmt.Errorf("invalid protected structural-whitespace attempt codec")
	}
	p := &attemptWhitespaceResponse{codec: c, candidate: bytes.Clone(candidate), adapter: adapter}
	pos, opaque := 0, false
	for _, r := range c.Repairs {
		if r.Start < pos || r.End <= r.Start || r.End >= len(candidate) || candidate[r.End] != '\n' || string(candidate[r.Start:r.End]) != r.Whitespace || strings.Trim(r.Whitespace, " \t\r") != "" {
			return nil, fmt.Errorf("invalid whitespace repair coordinates")
		}
		p.segments = append(p.segments, string(candidate[pos:r.Start]))
		pos, opaque = r.End, opaque || r.OpaqueLine
	}
	if !opaque {
		return nil, fmt.Errorf("codec requires a proved protected-only whitespace line")
	}
	p.segments = append(p.segments, string(candidate[pos:]))
	if sum([]byte(strings.Join(p.segments, ""))) != c.RepairedCandidateSHA256 {
		return nil, fmt.Errorf("whitespace repair expected bytes mismatch")
	}
	for i := range p.segments {
		t := fmt.Sprintf("⟪GTI18N_WS1_%s_%06d⟫", c.Identity, i)
		if bytes.Contains(candidate, []byte(t)) {
			return nil, fmt.Errorf("whitespace codec token collision")
		}
		p.tokens = append(p.tokens, t)
	}
	return p, nil
}

func (p *attemptWhitespaceResponse) ProtectedInput(*Glossary) (string, []string, error) {
	var out strings.Builder
	for i, token := range p.tokens {
		out.WriteString(token)
		if i < len(p.codec.Repairs) {
			out.WriteString(p.codec.Repairs[i].Whitespace)
		}
	}
	return out.String(), append([]string(nil), p.tokens...), nil
}

func (p *attemptWhitespaceResponse) Restore(raw string) ([]byte, error) {
	raw = strings.TrimSuffix(raw, "\n") // shared single-LF transport framing
	var out strings.Builder
	pos := 0
	for i, token := range p.tokens {
		if strings.Count(raw, token) != 1 || !strings.HasPrefix(raw[pos:], token) {
			return nil, fmt.Errorf("whitespace codec protected token exact/unique/order changed")
		}
		pos += len(token)
		out.WriteString(p.segments[i])
		if i < len(p.codec.Repairs) {
			end := strings.Index(raw[pos:], p.tokens[i+1])
			if end < 0 {
				return nil, fmt.Errorf("whitespace codec missing boundary")
			}
			value := raw[pos : pos+end]
			if value != "" && value != p.codec.Repairs[i].Whitespace {
				return nil, fmt.Errorf("whitespace codec permits only exact structural-whitespace deletion")
			}
			out.WriteString(value)
			pos += end
		}
	}
	if pos != len(raw) {
		return nil, fmt.Errorf("text outside whitespace codec boundaries")
	}
	return []byte(out.String()), nil
}

func (p *attemptWhitespaceResponse) Validate(candidate []byte, g *Glossary) error {
	return p.adapter.Validate(candidate, g)
}
