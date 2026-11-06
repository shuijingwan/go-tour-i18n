package i18n

import "testing"

func TestCommentBlockDispositionExactSet(t *testing.T) {
	blocks := []CommentCodeBlock{
		SourceCommentBlockDisposition(AnalyzeCommentCode("var x int\n", "go", "1", false), "", nil),
		SourceCommentBlockDisposition(AnalyzeCommentCode("var x int // The integer value.\n", "go", "2", false), "", nil),
		SourceCommentBlockDisposition(AnalyzeCommentCode("unknown source notation ???\n", "", "3", false), "", nil),
	}
	if err := ValidateCommentBlockDispositions(blocks); err != nil {
		t.Fatal(err)
	}
	if blocks[0].Disposition != "comment-free-by-grammar" || blocks[1].Disposition != "comment-extracted" || blocks[2].Disposition != "ambiguous-unsupported" {
		t.Fatal(blocks)
	}
	for _, bad := range []func(*CommentCodeBlock){
		func(b *CommentCodeBlock) { b.Unsupported = true },
		func(b *CommentCodeBlock) {
			b.Disposition = "non-code-output-or-literal"
			b.Comments = []CodeComment{{Kind: "teaching-comment"}}
		},
		func(b *CommentCodeBlock) { b.Disposition = "guessed-shell" },
		func(b *CommentCodeBlock) { b.DispositionEvidence = "" },
	} {
		copy := append([]CommentCodeBlock{}, blocks...)
		bad(&copy[0])
		if ValidateCommentBlockDispositions(copy) == nil {
			t.Fatal("invalid claim accepted", copy[0])
		}
	}
	if ValidateCommentBlockDispositions(append(blocks, blocks[0])) == nil {
		t.Fatal("duplicate coordinate silently accepted")
	}
	if err := ValidateSourceCommentBlockSet(blocks, blocks); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]CommentCodeBlock{blocks[:2], append(append([]CommentCodeBlock{}, blocks...), blocks[0]), {blocks[1], blocks[0], blocks[2]}} {
		if ValidateSourceCommentBlockSet(blocks, bad) == nil {
			t.Fatal("dropped/extra/reordered source block accepted")
		}
	}
	changed := append([]CommentCodeBlock{}, blocks...)
	changed[0].Code += " "
	if ValidateSourceCommentBlockSet(blocks, changed) == nil {
		t.Fatal("source bytes rewritten by disposition")
	}
	b := SourceCommentBlockDisposition(AnalyzeCommentCode("go 1.18\n", "", "4", false), "go.mod file reference", nil)
	if b.Unsupported || b.Language != "gomod" || b.Disposition != "comment-free-by-grammar" {
		t.Fatal(b)
	}
	b = SourceCommentBlockDisposition(AnalyzeCommentCode("$ echo hi\n", "shell", "5", false), "go.mod file reference", nil)
	if !b.Unsupported {
		t.Fatal("explicit unsupported language overwritten by metadata")
	}
}

func TestCommentBlockDispositionSourceSelectedTarget(t *testing.T) {
	source := []CommentCodeBlock{SourceCommentBlockDisposition(AnalyzeCommentCode("require example.com/a v1.0.0 // indirect\n", "", "1", false), "go.mod file reference", nil)}
	for _, fixture := range []struct {
		code string
		pass bool
	}{
		{"require example.com/a v1.0.0 // indirect\n", true},
		{"require example.com/a v1.0.0 // changed\n", false},
		{"require example.com/a v1.0.0\n", false},
		{"require example.com/a v1.0.0 // indirect\n// Extra comment.\n", false},
	} {
		target := []CommentCodeBlock{AnalyzeCommentCode(fixture.code, "", "1", false)}
		target = SourceSelectedTargetCommentBlocks(source, target)
		rows, failures := CompareCommentBlocks("fixture", "zh", source, target)
		if pass := ValidateCommentCoverage(rows, failures, true) == nil; pass != fixture.pass {
			t.Fatal(fixture, rows, failures)
		}
	}
	unknown := []CommentCodeBlock{SourceCommentBlockDisposition(AnalyzeCommentCode("$ echo unknown\n", "", "1", false), "", nil)}
	target := SourceSelectedTargetCommentBlocks(unknown, source)
	if !target[0].Unsupported || len(target[0].Comments) != 0 {
		t.Fatal("target supplied source authority")
	}
	declared := []CommentCodeBlock{AnalyzeCommentCode(source[0].Code, "shell", "1", false)}
	if !SourceSelectedTargetCommentBlocks(source, declared)[0].Unsupported {
		t.Fatal("target language declaration mutation accepted")
	}
}
