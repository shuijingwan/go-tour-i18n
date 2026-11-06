package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"strings"
)

func readCommentPredecessor(raw []byte) ([]commentDispositionRecord, error) {
	var envelope struct {
		Schema    string            `json:"schema"`
		Taxonomy  map[string]int    `json:"taxonomy"`
		Languages map[string]int    `json:"language_counts"`
		Residuals map[string]int    `json:"residual_classes"`
		Blocks    []json.RawMessage `json:"blocks"`
	}
	// RawMessage lets the shared strict walker validate UTF-8, duplicate keys
	// and framing without pretending its struct reflection flattens embedding.
	if err := sitecontent.StrictJSON(raw, &envelope); err != nil {
		return nil, err
	}
	if envelope.Schema != "go-learning/code-comment-inventory/v1" {
		return nil, fmt.Errorf("unknown comment predecessor schema")
	}
	records := []commentDispositionRecord{}
	for _, block := range envelope.Blocks {
		var members map[string]json.RawMessage
		if err := json.Unmarshal(block, &members); err != nil {
			return nil, err
		}
		for _, value := range members {
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return nil, fmt.Errorf("null predecessor scalar")
			}
		}
		var record commentDispositionRecord
		decoder := json.NewDecoder(bytes.NewReader(block))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func compareCommentPredecessor(previous, current []commentDispositionRecord) (map[string]int, error) {
	if len(previous) != len(current) {
		return nil, fmt.Errorf("comment predecessor source exact-set mismatch")
	}
	counts := map[string]int{"RESIDUAL_INPUT": 0, "PRESERVED_PROVEN_BLOCKS": 0, "TERMINAL_TRANSCRIPT_BLOCKS": 0, "NON_CODE_OUTPUT_OR_LITERAL_BLOCKS": 0, "GO_STRICT_BLOCKS": 0, "GOMOD_BLOCKS": 0, "MACHINE_OPAQUE_BLOCKS": 0, "OTHER_PROVEN_BLOCKS": 0, "CURRENT_AMBIGUOUS_UNSUPPORTED_BLOCKS": 0}
	seen := map[string]bool{}
	for i, p := range previous {
		c := current[i]
		key := p.Package + "/" + string(p.ParentKind) + "/" + p.ParentID + "/" + p.Coordinate
		if seen[key] || p.Package != c.Package || p.ParentKind != c.ParentKind || p.ParentID != c.ParentID || p.Coordinate != c.Coordinate || p.SourcePath != c.SourcePath || p.SourceSHA != c.SourceSHA || p.BlockSHA != c.BlockSHA || p.Code != c.Code || p.StableIndex != c.StableIndex || p.Route != c.Route || p.EnclosingKind != c.EnclosingKind || p.DeclaredLanguage != c.DeclaredLanguage {
			return nil, fmt.Errorf("comment predecessor identity/order drift: %s", key)
		}
		seen[key] = true
		if p.Disposition != "ambiguous-unsupported" {
			if p.Disposition != c.Disposition || p.Language != c.Language || p.Authority != c.Authority {
				return nil, fmt.Errorf("previously proved source disposition changed: %s", key)
			}
			counts["PRESERVED_PROVEN_BLOCKS"]++
			continue
		}
		counts["RESIDUAL_INPUT"]++
		switch {
		case c.Disposition == "ambiguous-unsupported":
			counts["CURRENT_AMBIGUOUS_UNSUPPORTED_BLOCKS"]++
		case c.Disposition == "terminal-transcript":
			counts["TERMINAL_TRANSCRIPT_BLOCKS"]++
		case c.Disposition == "non-code-output-or-literal":
			counts["NON_CODE_OUTPUT_OR_LITERAL_BLOCKS"]++
		case c.Disposition == "machine-semantic-opaque":
			counts["MACHINE_OPAQUE_BLOCKS"]++
		case strings.HasPrefix(c.ResolvedExtractor, "modfile-"):
			counts["GOMOD_BLOCKS"]++
		case strings.HasPrefix(c.ResolvedExtractor, "tour-go-"):
			counts["GO_STRICT_BLOCKS"]++
		default:
			counts["OTHER_PROVEN_BLOCKS"]++
		}
	}
	return counts, nil
}
