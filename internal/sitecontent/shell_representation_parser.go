package sitecontent

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"golang.org/x/net/html"
)

const ShellTeachingRepresentation = "go-learning/shell-textarea-teaching-representation/v1"

// This is a prospective, explicitly versioned parser. It does not change the
// default extractor. Only the canonical representation authority may select it.
func prospectiveShellRepresentation(path string, raw []byte, rule string) (*ShellLanguageSource, error) {
	if rule != ShellTeachingRepresentation {
		return nil, fmt.Errorf("unsupported shell representation rule")
	}
	old, err := ExtractShellPageRepresentation(path, raw)
	if err != nil || old.Page == nil {
		return nil, fmt.Errorf("formal Homepage representation required: %v", err)
	}
	body, _, err := ShellPageRenderInput(raw)
	if err != nil {
		return nil, err
	}
	e := shellExtractor{source: *old, raw: raw, semantic: true, occurrences: map[string]int{}}
	e.source.Slots = append([]ShellLanguageSlot{}, old.Slots...)
	z := html.NewTokenizer(bytes.NewReader(body))
	pos, editor := len(raw)-len(body), false
	for {
		t := z.Next()
		start := pos
		pos += len(z.Raw())
		if t == html.ErrorToken {
			if z.Err() != io.EOF {
				return nil, z.Err()
			}
			break
		}
		if t == html.StartTagToken || t == html.EndTagToken {
			tag, _ := z.TagName()
			if string(tag) == "textarea" {
				editor = t == html.StartTagToken
			}
		}
		if t != html.TextToken || !editor {
			continue
		}
		// Encoding must be exact: never guess decoded-to-raw coordinates.
		code := raw[start:pos]
		if !bytes.Equal(code, z.Text()) {
			return nil, fmt.Errorf("textarea entity/raw mapping requires explicit authority")
		}
		block := i18n.AnalyzeCommentCode(string(code), "go", "shell-textarea", false)
		if block.Unsupported {
			return nil, fmt.Errorf("unsupported textarea grammar: %s", block.Diagnostic)
		}
		for _, c := range block.Comments {
			if c.Kind != "teaching-comment" {
				continue
			}
			if c.Style != "line" || !strings.HasPrefix(c.Text, "//") || c.Body == "" {
				return nil, fmt.Errorf("only scanner-owned line teaching bodies supported")
			}
			i := strings.Index(c.Text, c.Body)
			if i < 2 || strings.TrimSpace(c.Text[2:i]) != "" || strings.TrimSpace(c.Text[i+len(c.Body):]) != "" {
				return nil, fmt.Errorf("ambiguous teaching body coordinates")
			}
			a, b := start+c.Start+i, start+c.Start+i+len(c.Body)
			found := false
			for _, s := range old.Slots {
				if s.Unit.Start == a && s.Unit.End == b && s.Role == "teaching-comment" {
					found = true
				} else if a < s.Unit.End && b > s.Unit.Start {
					return nil, fmt.Errorf("new teaching body overlaps existing language authority")
				}
			}
			if found {
				continue
			}
			// Retain the existing identifier binder, using the same Go program.
			// The semantic decision comes from the active source classifier;
			// SafeTeachingCommentBodies itself remains unchanged.
			var ids []i18n.TeachingSpan
			// Scanner-owned body protection preserves identifiers occurring in
			// the program, even when the old exposure heuristic omitted a body.
			bodyInfo, err := i18n.SemanticTeachingCommentBody(string(code), c)
			if err != nil {
				return nil, err
			}
			for _, id := range bodyInfo.Identifiers {
				ids = append(ids, i18n.TeachingSpan{Start: start + id.Start, End: start + id.End})
			}
			bodyInfo.Identifiers = ids
			text, protected, err := protectTeachingBodyPolicy(raw, a, b, bodyInfo, false)
			if err != nil {
				return nil, err
			}
			context := "homepage/textarea/" + block.Coordinate
			if err := e.protectedSlot(a, b, text, protected, "teaching-comment", context, "go-comment-interior"); err != nil {
				return nil, err
			}
		}
	}
	next, err := e.finish()
	if err != nil {
		return nil, err
	}
	if next.SourceSHA != old.SourceSHA || next.Page.SourceSHA != old.Page.SourceSHA || len(next.Page.Units) != 1 {
		return nil, fmt.Errorf("representation changed physical Page/source boundary")
	}
	for _, s := range old.Slots {
		found := false
		for _, n := range next.Slots {
			found = found || reflect.DeepEqual(s, n)
		}
		if !found {
			return nil, fmt.Errorf("existing language/protected slot changed")
		}
	}
	restored, err := (ShellPageAdapter{*next, raw}).Restore(next.Page.Units[0].Source)
	if err != nil || !bytes.Equal(restored, raw) {
		return nil, fmt.Errorf("representation source round trip failed: %v", err)
	}
	return next, nil
}
