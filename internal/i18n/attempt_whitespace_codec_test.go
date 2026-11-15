package i18n

import (
	"bytes"
	"strings"
	"testing"
)

func TestAttemptWhitespaceResponseIsDeletionOnly(t *testing.T) {
	original := []byte("opaque target with API and link\n  \nopaque tail\n")
	a := strings.Index(string(original), "  \n")
	c := attemptWhitespaceCodec{Schema: attemptWhitespaceSchema, Locale: "fixture", BatchID: "batch", Attempt: 2, Record: RetranslationBatchUnit{UnitID: "unit"}, OriginalCandidateSHA256: sum(original), Repairs: []StructuralWhitespaceRepair{{Line: 2, Start: a, End: a + 2, Whitespace: "  ", OpaqueLine: true}}}
	c.RepairedCandidateSHA256 = sum(bytes.Replace(original, []byte("  \n"), []byte("\n"), 1))
	c.Identity = c.identity()
	p, err := whitespaceResponse(c, original, nil)
	if err != nil {
		t.Fatal(err)
	}
	baseline, tokens, err := p.ProtectedInput(nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(baseline, "opaque") || strings.Contains(baseline, "API") {
		t.Fatal("opaque language became editable")
	}
	got, err := p.Restore(baseline + "\n")
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("baseline is not exact historical candidate", err)
	}
	fixed := strings.Replace(baseline, "  ", "", 1)
	got, err = p.Restore(fixed + "\n")
	if err != nil || sum(got) != c.RepairedCandidateSHA256 {
		t.Fatal("exact deletion failed", err)
	}
	for _, bad := range []string{strings.Replace(baseline, "  ", "prose", 1), strings.Replace(baseline, "  ", " ", 1), strings.Replace(baseline, "  ", "\t", 1), strings.Replace(baseline, "  ", "\u00a0", 1), strings.Replace(baseline, tokens[0], "", 1), baseline + tokens[0], tokens[1] + tokens[0], "text" + baseline, baseline + "text", strings.Replace(baseline, tokens[0], "WRONG", 1)} {
		if _, err := p.Restore(bad); err == nil {
			t.Fatal("non-whitespace edit/token tamper accepted", bad)
		}
	}
	for _, change := range []func(*attemptWhitespaceCodec){func(x *attemptWhitespaceCodec) { x.Attempt++ }, func(x *attemptWhitespaceCodec) { x.Locale = "foreign" }, func(x *attemptWhitespaceCodec) { x.Record.UnitID = "foreign" }, func(x *attemptWhitespaceCodec) { x.OriginalRepresentation = "foreign" }} {
		other := c
		change(&other)
		if _, err := whitespaceResponse(other, original, nil); err == nil {
			t.Fatal("codec identity tamper accepted")
		}
		other.Identity = other.identity()
		q, err := whitespaceResponse(other, original, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := q.Restore(fixed + "\n"); err == nil {
			t.Fatal("output reused across attempt/Unit/identity")
		}
	}
	bad := c
	bad.Repairs = append([]StructuralWhitespaceRepair(nil), c.Repairs...)
	bad.Repairs[0].Start--
	bad.Identity = bad.identity()
	if _, err := whitespaceResponse(bad, original, nil); err == nil {
		t.Fatal("wrong protected boundary accepted")
	}
}
