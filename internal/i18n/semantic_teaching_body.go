package i18n

import (
	"fmt"
	"reflect"
	"sort"
)

// SemanticTeachingCommentBody provides source-bound identifier coordinates to
// explicit representation versions. It does not alter the legacy exposure
// heuristic or authorize any current representation change by itself.
func SemanticTeachingCommentBody(code string, comment CodeComment) (TeachingCommentBody, error) {
	block := AnalyzeCommentCode(code, "go", "semantic-teaching-body", false)
	found := false
	for _, c := range block.Comments {
		found = found || reflect.DeepEqual(c, comment)
	}
	if block.Unsupported || !found || comment.Kind != "teaching-comment" {
		return TeachingCommentBody{}, fmt.Errorf("exact source teaching comment required")
	}
	a := analyzePreformattedGo(code)
	if a.Static {
		return TeachingCommentBody{}, fmt.Errorf("safe Go source required")
	}
	for _, c := range a.Comments {
		if c.Start != comment.Start || c.End != comment.End {
			continue
		}
		body := TeachingCommentBody{TeachingSpan: TeachingSpan{c.BodyStart, c.BodyEnd}}
		for _, id := range c.Identifiers {
			body.Identifiers = append(body.Identifiers, TeachingSpan{id.Start, id.End})
		}
		for id := range a.CodeIdentifiers {
			body.CodeIdentifiers = append(body.CodeIdentifiers, id)
		}
		sort.Strings(body.CodeIdentifiers)
		return body, nil
	}
	return TeachingCommentBody{}, fmt.Errorf("teaching body/source coordinate mismatch")
}
