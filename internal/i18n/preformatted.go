package i18n

import (
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"
	"unicode"
)

type preformattedBlock struct {
	Start int
	End   int
	Text  string
}

type preformattedComment struct {
	Start        int
	End          int
	BodyStart    int
	BodyEnd      int
	Translatable bool
	Identifiers  []preformattedIdentifier
}

type preformattedIdentifier struct {
	Start int
	End   int
	Value string
}

type preformattedAnalysis struct {
	Comments        []preformattedComment
	CodeIdentifiers map[string]bool
	Static          bool
}

// preformattedBlocks applies the same indentation policy used by the
// structural validator. Blank lines belong to the surrounding present
// preformatted block and are therefore protected together with it.
func preformattedBlocks(text string) []preformattedBlock {
	var blocks []preformattedBlock
	for pos := 0; pos < len(text); {
		lineStart := pos
		lineEnd := strings.IndexByte(text[pos:], '\n')
		if lineEnd < 0 {
			lineEnd = len(text)
		} else {
			lineEnd += pos + 1
		}
		line := strings.TrimSuffix(text[lineStart:lineEnd], "\n")
		if !isPreformattedLine(line) {
			pos = lineEnd
			continue
		}

		start := lineStart
		pos = lineEnd
		for pos < len(text) {
			nextEnd := strings.IndexByte(text[pos:], '\n')
			if nextEnd < 0 {
				nextEnd = len(text)
			} else {
				nextEnd += pos + 1
			}
			next := strings.TrimSuffix(text[pos:nextEnd], "\n")
			if next != "" && !isPreformattedLine(next) {
				break
			}
			pos = nextEnd
		}
		blocks = append(blocks, preformattedBlock{Start: start, End: pos, Text: text[start:pos]})
	}
	return blocks
}

func isPreformattedLine(line string) bool {
	return strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "  ")
}

func analyzePreformattedGo(text string) preformattedAnalysis {
	analysis := preformattedAnalysis{
		CodeIdentifiers: map[string]bool{},
		Static:          !isSafeGoFragment(text),
	}
	fset := token.NewFileSet()
	file := fset.AddFile("preformatted.go", -1, len(text))
	var scan scanner.Scanner
	scan.Init(file, []byte(text), func(_ token.Position, _ string) {
		analysis.Static = true
	}, scanner.ScanComments)

	for {
		pos, tok, literal := scan.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.IDENT {
			analysis.CodeIdentifiers[literal] = true
			continue
		}
		if tok != token.COMMENT {
			continue
		}
		if !strings.HasPrefix(literal, "//") {
			// Block comments are deliberately unsupported in the first phase.
			analysis.Static = true
			continue
		}
		start := file.Offset(pos)
		end := start + len(literal)
		bodyStart := start + 2
		for bodyStart < end && (text[bodyStart] == ' ' || text[bodyStart] == '\t') {
			bodyStart++
		}
		bodyEnd := end
		for bodyEnd > bodyStart && (text[bodyEnd-1] == ' ' || text[bodyEnd-1] == '\t' || text[bodyEnd-1] == '\r') {
			bodyEnd--
		}
		body := text[bodyStart:bodyEnd]
		analysis.Comments = append(analysis.Comments, preformattedComment{
			Start:        start,
			End:          end,
			BodyStart:    bodyStart,
			BodyEnd:      bodyEnd,
			Translatable: isTranslatableTeachingComment(body),
		})
	}
	for i := range analysis.Comments {
		comment := &analysis.Comments[i]
		comment.Identifiers = referencedCommentIdentifiers(text[comment.BodyStart:comment.BodyEnd], comment.BodyStart, analysis.CodeIdentifiers)
	}
	return analysis
}

func referencedCommentIdentifiers(body string, base int, codeIdentifiers map[string]bool) []preformattedIdentifier {
	fset := token.NewFileSet()
	file := fset.AddFile("comment.go", -1, len(body))
	var identifiers []preformattedIdentifier
	var scan scanner.Scanner
	scan.Init(file, []byte(body), nil, 0)
	for {
		pos, tok, literal := scan.Scan()
		if tok == token.EOF {
			return identifiers
		}
		if tok != token.IDENT || !codeIdentifiers[literal] {
			continue
		}
		start := base + file.Offset(pos)
		identifiers = append(identifiers, preformattedIdentifier{Start: start, End: start + len(literal), Value: literal})
	}
}

func isSafeGoFragment(text string) bool {
	variants := []string{
		"package p\n" + text,
		"package p\nfunc _() {\n" + text + "\n}\n",
	}
	for _, source := range variants {
		if _, err := parser.ParseFile(token.NewFileSet(), "fragment.go", source, parser.SkipObjectResolution); err == nil {
			return true
		}
	}

	// The tour also uses a reference-style list of Go's predeclared basic
	// types. It is recognizably Go even though it is not a standalone grammar
	// production, so admit only that narrow fragment shape.
	predeclared := map[string]bool{
		"bool": true, "byte": true, "complex128": true, "complex64": true,
		"float32": true, "float64": true, "int": true, "int16": true,
		"int32": true, "int64": true, "int8": true, "rune": true,
		"string": true, "uint": true, "uint16": true, "uint32": true,
		"uint64": true, "uint8": true, "uintptr": true,
	}
	fset := token.NewFileSet()
	file := fset.AddFile("fragment.go", -1, len(text))
	valid := true
	seen := false
	var scan scanner.Scanner
	scan.Init(file, []byte(text), func(_ token.Position, _ string) { valid = false }, scanner.ScanComments)
	for {
		_, tok, literal := scan.Scan()
		switch tok {
		case token.EOF:
			return valid && seen
		case token.COMMENT, token.SEMICOLON:
			continue
		case token.IDENT:
			if !predeclared[literal] {
				valid = false
			}
			seen = true
		default:
			valid = false
		}
	}
}

func isTranslatableTeachingComment(body string) bool {
	body = strings.TrimSpace(body)
	if body == "" || strings.EqualFold(body, "OK") || strings.ContainsAny(body, "=()[]{}<>") {
		return false
	}

	words := englishWords(body)
	if len(words) < 2 {
		return false
	}
	markers := map[string]bool{
		"a": true, "an": true, "and": true, "as": true, "assign": true,
		"alias": true, "block": true, "channel": true, "compile": true,
		"error": true, "for": true, "from": true, "has": true, "here": true,
		"is": true, "match": true, "no": true, "point": true, "pointer": true,
		"read": true, "receive": true, "receiving": true, "represents": true,
		"same": true, "send": true, "set": true, "the": true, "through": true,
		"to": true, "type": true, "unicode": true, "use": true, "value": true,
		"would": true,
	}
	for _, word := range words {
		if markers[strings.ToLower(word)] {
			return true
		}
	}
	return false
}

func englishWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
}

func preformattedProtectionSpans(text string) []protectedSpan {
	var spans []protectedSpan
	for _, block := range preformattedBlocks(text) {
		analysis := analyzePreformattedGo(block.Text)
		if analysis.Static || !hasTranslatableComment(analysis.Comments) {
			spans = append(spans, protectedSpan{start: block.Start, end: staticPreformattedPayloadEnd(text, block), kind: protectedPreformattedStatic})
			continue
		}

		pos := 0
		for _, comment := range analysis.Comments {
			if !comment.Translatable {
				continue
			}
			if pos < comment.BodyStart {
				spans = append(spans, protectedSpan{start: block.Start + pos, end: block.Start + comment.BodyStart, kind: protectedPreformatted})
			}
			for _, identifier := range comment.Identifiers {
				spans = append(spans, protectedSpan{
					start: block.Start + identifier.Start,
					end:   block.Start + identifier.End,
					kind:  protectedPreformattedIdentifier,
				})
			}
			pos = comment.BodyEnd
		}
		if pos < len(block.Text) {
			spans = append(spans, protectedSpan{start: block.Start + pos, end: block.End, kind: protectedPreformatted})
		}
	}
	return spans
}

// staticPreformattedPayloadEnd leaves the source separator after a static
// preformatted block visible to the model. preformattedBlocks deliberately
// owns those trailing blank lines for present validation, but replacing them
// with the static token would otherwise make the token run directly into the
// next prose or directive. At EOF there is no following structure to
// separate, so retain the complete block payload.
func staticPreformattedPayloadEnd(text string, block preformattedBlock) int {
	if block.End == len(text) {
		return block.End
	}
	end := block.End
	for end > block.Start {
		lineEnd := end
		if text[lineEnd-1] == '\n' {
			lineEnd--
		}
		lineStart := block.Start
		if previous := strings.LastIndexByte(text[block.Start:lineEnd], '\n'); previous >= 0 {
			lineStart += previous + 1
		}
		if lineStart != lineEnd {
			return lineEnd
		}
		end = lineStart
	}
	return block.End
}

func hasTranslatableComment(comments []preformattedComment) bool {
	for _, comment := range comments {
		if comment.Translatable {
			return true
		}
	}
	return false
}

func comparePreformatted(source, candidate string) error {
	expected := preformattedBlocks(source)
	actual := preformattedBlocks(candidate)
	if len(expected) != len(actual) {
		return protectedCountError("preformatted code block", len(expected), len(actual))
	}
	for i := range expected {
		if err := comparePreformattedBlock(expected[i].Text, actual[i].Text); err != nil {
			return preformattedBlockError(i+1, err)
		}
	}
	return nil
}

func comparePreformattedBlock(source, candidate string) error {
	expected := analyzePreformattedGo(source)
	if expected.Static || !hasTranslatableComment(expected.Comments) {
		return compareExactPreformatted(source, candidate)
	}
	actual := analyzePreformattedGo(candidate)
	if actual.Static {
		return preformattedComparisonError("candidate is not a safely recognized //-comment Go block")
	}
	if len(expected.Comments) != len(actual.Comments) {
		return protectedCountError("line comment", len(expected.Comments), len(actual.Comments))
	}

	expectedPos, actualPos := 0, 0
	for i := range expected.Comments {
		want := expected.Comments[i]
		got := actual.Comments[i]
		if source[expectedPos:want.BodyStart] != candidate[actualPos:got.BodyStart] {
			return preformattedCommentError(i+1, "non-comment bytes or // position changed")
		}
		wantBody := source[want.BodyStart:want.BodyEnd]
		gotBody := candidate[got.BodyStart:got.BodyEnd]
		if want.Translatable {
			if strings.TrimSpace(gotBody) == "" {
				return preformattedCommentError(i+1, "translatable comment body is empty")
			}
			if strings.Contains(gotBody, "//") || strings.Contains(gotBody, "/*") || strings.Contains(gotBody, "*/") {
				return preformattedCommentError(i+1, "translated comment body contains a comment marker")
			}
			gotIdentifiers := referencedCommentIdentifiers(gotBody, 0, expected.CodeIdentifiers)
			if err := compareCommentIdentifiers(want.Identifiers, gotIdentifiers); err != nil {
				return preformattedCommentError(i+1, err.Error())
			}
		} else if wantBody != gotBody {
			return preformattedCommentError(i+1, "non-translatable comment body changed")
		}
		expectedPos = want.BodyEnd
		actualPos = got.BodyEnd
	}
	if source[expectedPos:] != candidate[actualPos:] {
		return preformattedComparisonError("non-comment bytes, indentation, or line endings changed")
	}
	return nil
}

func compareCommentIdentifiers(expected, actual []preformattedIdentifier) error {
	if len(expected) != len(actual) {
		return protectedCountError("referenced Go identifier", len(expected), len(actual))
	}
	for i := range expected {
		if expected[i].Value != actual[i].Value {
			return fmt.Errorf("referenced Go identifier mismatch at index %d: expected %q, actual %q", i+1, expected[i].Value, actual[i].Value)
		}
	}
	return nil
}

func compareExactPreformatted(source, candidate string) error {
	if source != candidate {
		return preformattedComparisonError("static preformatted block changed")
	}
	return nil
}
