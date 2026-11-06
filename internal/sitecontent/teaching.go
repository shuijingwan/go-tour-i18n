package sitecontent

import (
	"bytes"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
	"io"
	"regexp"
	"strings"
)

func protectTeachingBody(raw []byte, start, end int, body i18n.TeachingCommentBody) (string, []ProtectedToken, error) {
	return protectTeachingBodyPolicy(raw, start, end, body, false)
}

func protectTeachingBodyPolicy(raw []byte, start, end int, body i18n.TeachingCommentBody, legacy bool) (string, []ProtectedToken, error) {
	var out strings.Builder
	locked := []ProtectedToken{}
	part := func(b []byte) error {
		text, tokens, err := protectBlockPolicy(b, "teaching-comment", true, nil, legacy)
		if err != nil {
			return err
		}
		replacements := []string{}
		for _, p := range tokens {
			token := fmt.Sprintf("⟦P%d⟧", len(locked))
			replacements = append(replacements, p.Token, token)
			p.Token = token
			locked = append(locked, p)
		}
		if len(replacements) > 0 {
			text = strings.NewReplacer(replacements...).Replace(text)
		}
		out.WriteString(text)
		return nil
	}
	pos := start
	for _, id := range body.Identifiers {
		if err := part(raw[pos:id.Start]); err != nil {
			return "", nil, err
		}
		token := fmt.Sprintf("⟦P%d⟧", len(locked))
		locked = append(locked, ProtectedToken{Token: token, Raw: string(raw[id.Start:id.End]), Role: "atom"})
		out.WriteString(token)
		pos = id.End
	}
	if err := part(raw[pos:end]); err != nil {
		return "", nil, err
	}
	return out.String(), locked, nil
}

var teachingASCIIIdentifier = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9]*`)

func validateTeachingBody(s Unit, target string, raw []byte, markdown bool) error {
	var body *i18n.TeachingCommentBody
	offset, err := parseMetadata(raw, func(int, int, string) {})
	if err != nil {
		return err
	}
	// The persisted slot owns eligibility. A historical slot must not acquire
	// the new export-time fence-language rule during literal restoration.
	err = teachingSpansMode(raw[offset:], offset, markdown, false, func(int, int, string) {}, func(start int, c i18n.TeachingCommentBody) {
		if start == s.Start {
			body = &c
		}
	})
	if err != nil {
		return err
	}
	if body == nil {
		return fmt.Errorf("teaching comment lost source binding")
	}
	if strings.TrimSpace(target) == "" || strings.ContainsAny(target, "\n\r") || strings.Contains(target, "//") || strings.Contains(target, "/*") || strings.Contains(target, "*/") {
		return fmt.Errorf("teaching comment body/layout changed")
	}
	want, got := map[string]int{}, map[string]int{}
	ids := map[string]bool{}
	for _, id := range body.CodeIdentifiers {
		ids[id] = true
	}
	for _, id := range body.Identifiers {
		want[string(raw[id.Start:id.End])]++
	}
	for _, id := range teachingASCIIIdentifier.FindAllString(target, -1) {
		if ids[id] {
			got[id]++
		}
	}
	// Opaque tokens protect Unicode Go identifiers as well. Count their exact
	// source bytes without treating adjacent translated text as Go syntax.
	for id := range ids {
		if teachingASCIIIdentifier.FindString(id) != id {
			got[id] = strings.Count(target, id)
		}
	}
	for id, count := range want {
		if got[id] != count {
			return fmt.Errorf("teaching identifier exact count changed: %s", id)
		}
	}
	for id, count := range got {
		if want[id] != count {
			return fmt.Errorf("new teaching identifier: %s", id)
		}
	}
	// The code-identifier count above cannot see names absent from the Go
	// source. General prose classification intentionally does not infer an
	// identity from spelling alone; an already source-bound Go teaching slot
	// must additionally reject new code-like identities in its editable body.
	sourceIdentities := teachingASCIIIdentityCounts(string(raw[s.Start:s.End]))
	for id, count := range teachingASCIIIdentityCounts(target) {
		if count > sourceIdentities[id] {
			return fmt.Errorf("new teaching ASCII identity: %s", id)
		}
	}
	return nil
}

func teachingASCIIIdentityCounts(text string) map[string]int {
	counts := map[string]int{}
	for _, match := range teachingASCIIIdentifier.FindAllStringIndex(text, -1) {
		id := text[match[0]:match[1]]
		codeLike := strings.Contains(id, "_")
		for i := 1; i < len(id); i++ {
			if id[i-1] >= 'a' && id[i-1] <= 'z' && id[i] >= 'A' && id[i] <= 'Z' {
				codeLike = true
			}
		}
		if codeLike {
			counts[id]++
		}
		after := text[match[1]:]
		// Preserve natural technical prose such as "Go (JSON or API)". An
		// immediate call opener or an empty call, including "f ()", has an
		// explicit code role independent of capitalization or an acronym.
		if strings.HasPrefix(after, "(") || strings.HasPrefix(strings.TrimLeft(after, " \t"), "()") {
			counts[id+"()"]++
		}
	}
	return counts
}

func teachingSpans(b []byte, offset int, markdown bool, add func(int, int, string), bind func(int, i18n.TeachingCommentBody)) error {
	return teachingSpansMode(b, offset, markdown, true, add, bind)
}

func teachingSpansMode(b []byte, offset int, markdown, respectFenceLanguage bool, add func(int, int, string), bind func(int, i18n.TeachingCommentBody)) error {
	expose := func(code []byte, a int) {
		for _, s := range i18n.SafeTeachingCommentBodies(string(code)) {
			for i := range s.Identifiers {
				s.Identifiers[i].Start += offset + a
				s.Identifiers[i].End += offset + a
			}
			if bind != nil {
				bind(offset+a+s.Start, s)
			}
			add(offset+a+s.Start, offset+a+s.End, "teaching-comment")
		}
	}
	if markdown {
		tree := goldmark.New().Parser().Parse(gmtext.NewReader(b))
		if err := ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
			if !enter {
				return ast.WalkContinue, nil
			}
			if n.Kind() == ast.KindCodeBlock || n.Kind() == ast.KindFencedCodeBlock {
				if fence, ok := n.(*ast.FencedCodeBlock); ok && respectFenceLanguage {
					// An explicit non-Go fence cannot acquire Go comment semantics
					// merely because its payload happens to parse as a Go fragment.
					switch string(fence.Language(b)) {
					case "", "go", "golang":
					default:
						return ast.WalkSkipChildren, nil
					}
				}
				if n.Lines().Len() > 0 {
					a := n.Lines().At(0).Start
					z := n.Lines().At(n.Lines().Len() - 1).Stop
					expose(b[a:z], a)
				}
				return ast.WalkSkipChildren, nil
			}
			return ast.WalkContinue, nil
		}); err != nil {
			return err
		}
	}
	// Only plain Go text in pre is eligible; HTML entities/markup are protected.
	z := html.NewTokenizer(bytes.NewReader(b))
	pos := 0
	depth := 0
	for {
		typ := z.Next()
		raw := z.Raw()
		a := pos
		pos += len(raw)
		if typ == html.ErrorToken {
			if z.Err() != io.EOF {
				return z.Err()
			}
			break
		}
		if typ == html.StartTagToken {
			tag, _ := z.TagName()
			if string(tag) == "pre" {
				depth++
			}
		}
		if typ == html.EndTagToken {
			tag, _ := z.TagName()
			if string(tag) == "pre" && depth > 0 {
				depth--
			}
		}
		if typ == html.TextToken && depth > 0 && !strings.ContainsAny(string(raw), "&<>") {
			expose(raw, a)
		}
	}
	return nil
}
