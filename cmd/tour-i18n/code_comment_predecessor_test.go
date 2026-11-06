package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCommentResidualPredecessor(t *testing.T) {
	p := commentDispositionRecord{commentBlockRecord: commentBlockRecord{Package: "fixture", ParentID: "1", Coordinate: "1", Code: "source"}, Language: "go", Disposition: "comment-free-by-grammar", Authority: "source parser"}
	unknown := p
	unknown.ParentID = "2"
	unknown.Disposition = "ambiguous-unsupported"
	resolved := unknown
	resolved.Disposition = "terminal-transcript"
	counts, err := compareCommentPredecessor([]commentDispositionRecord{p, unknown}, []commentDispositionRecord{p, resolved})
	if err != nil || counts["PRESERVED_PROVEN_BLOCKS"] != 1 || counts["RESIDUAL_INPUT"] != 1 || counts["TERMINAL_TRANSCRIPT_BLOCKS"] != 1 {
		t.Fatal(counts, err)
	}
	bad := p
	bad.Code = "changed"
	if _, err := compareCommentPredecessor([]commentDispositionRecord{p}, []commentDispositionRecord{bad}); err == nil {
		t.Fatal("source drift accepted")
	}
	bad = p
	bad.Disposition = "non-code-output-or-literal"
	if _, err := compareCommentPredecessor([]commentDispositionRecord{p}, []commentDispositionRecord{bad}); err == nil {
		t.Fatal("proved disposition rewritten")
	}
	raw, err := json.Marshal(map[string]any{"schema": "go-learning/code-comment-inventory/v1", "blocks": []commentDispositionRecord{p}})
	if err != nil {
		t.Fatal(err)
	}
	read, err := readCommentPredecessor(raw)
	if err != nil || len(read) != 1 || read[0].Package != "fixture" {
		t.Fatal(read, err)
	}
	for _, invalid := range []string{strings.Replace(string(raw), `"package":"fixture"`, `"package":"fixture","package":"duplicate"`, 1), strings.Replace(string(raw), `"package":"fixture"`, `"package":null`, 1), strings.Replace(string(raw), `"package":"fixture"`, `"unknown":1,"package":"fixture"`, 1), string(raw) + `{}`} {
		if _, err := readCommentPredecessor([]byte(invalid)); err == nil {
			t.Fatal("invalid JSON transport accepted", invalid)
		}
	}
}
