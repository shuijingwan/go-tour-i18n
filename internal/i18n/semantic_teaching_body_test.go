package i18n

import "testing"

func TestSemanticTeachingBodyExplicitCoordinates(t *testing.T) {
	code := "package main\nvar x int\n// Review x carefully.\n"
	b := AnalyzeCommentCode(code, "go", "explicit-version", false)
	if b.Unsupported || len(b.Comments) != 1 || b.Comments[0].Kind != "teaching-comment" {
		t.Fatal(b)
	}
	body, err := SemanticTeachingCommentBody(code, b.Comments[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(body.Identifiers) != 1 || code[body.Identifiers[0].Start:body.Identifiers[0].End] != "x" {
		t.Fatal("identifier binding lost", body)
	}
	if len(SafeTeachingCommentBodies(code)) != 0 {
		t.Fatal("legacy conservative exposure changed")
	}
	altered := b.Comments[0]
	altered.Body = "injected text"
	if _, err := SemanticTeachingCommentBody(code, altered); err == nil {
		t.Fatal("unbound comment accepted")
	}
}
