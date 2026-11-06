package i18n

import "strings"

// AnalyzeJSONDocumentation reuses the existing scanner/schema grammar. The
// sole extensions are an object-member excerpt and a grammar-position ellipsis
// denoting omitted members. Neither changes original bytes or string semantics.
func AnalyzeJSONDocumentation(code, coordinate string) (CommentCodeBlock, bool) {
	text := strings.TrimSpace(code)
	if text == "" || !strings.ContainsAny(text[:1], "{[\"") {
		return CommentCodeBlock{}, false
	}
	wrapped := strings.HasPrefix(text, "\"")
	input := code
	if wrapped {
		input = "{" + code + "}"
	}
	b := analyzeCommentCodeMode(input, "json", coordinate, false, true)
	if b.Unsupported {
		return CommentCodeBlock{}, false
	}
	if wrapped {
		for i := range b.Comments {
			b.Comments[i].Start--
			b.Comments[i].End--
		}
	}
	b.Code, b.SHA256, b.Extractor = code, sum([]byte(code)), "tour-scanner-json-documentation/v1"
	var skeleton strings.Builder
	pos := 0
	for _, c := range b.Comments {
		if c.Start < pos || c.End > len(code) {
			return CommentCodeBlock{}, false
		}
		skeleton.WriteString(code[pos:c.Start])
		skeleton.WriteString("⟦comment:" + c.Style + "⟧")
		pos = c.End
	}
	skeleton.WriteString(code[pos:])
	b.Skeleton = skeleton.String()
	return b, true
}
