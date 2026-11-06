package sitecontent

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"io"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
)

// Bind decoded parser coordinates back to identical raw comment bytes. Markup
// or entity boundaries inside a payload are deliberately unsupported, rather
// than guessed. Entities elsewhere in the code do not affect the mapping.
func commentRawCoordinate(raw []byte, block i18n.CommentCodeBlock, c i18n.CodeComment) (int, error) {
	if block.Start < 0 || block.End > len(raw) || block.Start > block.End {
		return 0, fmt.Errorf("invalid comment block range")
	}
	b := raw[block.Start:block.End]
	if block.EnclosingKind != "html-pre" {
		if !bytes.Equal(b, []byte(block.Code)) {
			return markdownCommentRawCoordinate(raw, block, c)
		}
		return block.Start + c.Start, nil
	}
	z := html.NewTokenizer(bytes.NewReader(b))
	pos, decoded := 0, 0
	for {
		t := z.Next()
		start := pos
		text := string(z.Raw())
		pos += len(text)
		if t == html.ErrorToken {
			if z.Err() != io.EOF {
				return 0, z.Err()
			}
			break
		}
		if t != html.TextToken {
			continue
		}
		value := string(z.Text())
		if c.Start >= decoded && c.End <= decoded+len(value) {
			found := -1
			for at := 0; at <= len(text); {
				i := strings.Index(text[at:], c.Text)
				if i < 0 {
					break
				}
				i += at
				if len(stdhtml.UnescapeString(text[:i])) == c.Start-decoded && stdhtml.UnescapeString(text[i:i+len(c.Text)]) == c.Text {
					if found >= 0 {
						return 0, fmt.Errorf("ambiguous raw comment mapping")
					}
					found = i
				}
				at = i + 1
			}
			if found < 0 {
				return 0, fmt.Errorf("comment payload crosses encoded/markup boundary")
			}
			return block.Start + start + found, nil
		}
		decoded += len(value)
	}
	return 0, fmt.Errorf("comment coordinate not present in raw code")
}

// Goldmark's code lines can omit per-line indentation. Reuse its exact segment
// table, including padding, rather than searching for a repeated comment.
func markdownCommentRawCoordinate(raw []byte, block i18n.CommentCodeBlock, c i18n.CodeComment) (int, error) {
	offset, err := parseMetadata(raw, func(int, int, string) {})
	if err != nil {
		return 0, err
	}
	b := raw[offset:]
	tree := goldmark.New().Parser().Parse(gmtext.NewReader(b))
	index, found := 0, -1
	err = ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter || (n.Kind() != ast.KindCodeBlock && n.Kind() != ast.KindFencedCodeBlock) {
			return ast.WalkContinue, nil
		}
		index++
		if block.Coordinate != fmt.Sprintf("markdown/%d", index) {
			return ast.WalkSkipChildren, nil
		}
		var code strings.Builder
		for i := 0; i < n.Lines().Len(); i++ {
			s := n.Lines().At(i)
			code.Write(s.Value(b))
		}
		if code.String() != block.Code {
			return ast.WalkStop, fmt.Errorf("Markdown code segment projection changed")
		}
		decoded := 0
		for i := 0; i < n.Lines().Len(); i++ {
			s := n.Lines().At(i)
			value := s.Value(b)
			if c.Start >= decoded && c.End <= decoded+len(value) {
				a := s.Start + c.Start - decoded - s.Padding
				z := a + len(c.Text)
				if a < s.Start || z > s.Stop || !bytes.Equal(b[a:z], []byte(c.Text)) {
					return ast.WalkStop, fmt.Errorf("comment crosses Markdown synthetic padding or line boundary")
				}
				found = offset + a
			}
			decoded += len(value)
		}
		return ast.WalkStop, nil
	})
	if err != nil {
		return 0, err
	}
	if found < 0 {
		return 0, fmt.Errorf("comment has no exact Markdown segment coordinate")
	}
	return found, nil
}

type revisionCommentBody struct {
	unit     Unit
	original string
}

func (a revisionCommentBody) ProtectedInput(*i18n.Glossary) (string, []string, error) {
	tokens := []string{}
	for _, p := range a.unit.Protected {
		tokens = append(tokens, p.Token)
	}
	return a.unit.Source, tokens, nil
}
func (a revisionCommentBody) Validate([]byte, *i18n.Glossary) error {
	return fmt.Errorf("internal comment body only")
}
func (a revisionCommentBody) Restore(text string) ([]byte, error) {
	raw, err := restoreBlock(a.unit, text)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" || strings.ContainsAny(raw, "\r\n\x00⟦⟧⟪⟫<>&") || strings.Contains(raw, "//") || strings.Contains(raw, "/*") || strings.Contains(raw, "*/") {
		return nil, fmt.Errorf("comment payload syntax/layout changed")
	}
	want := teachingASCIIIdentityCounts(a.original)
	for id, n := range teachingASCIIIdentityCounts(raw) {
		if n > want[id] {
			return nil, fmt.Errorf("new teaching ASCII identity: %s", id)
		}
	}
	// Reuse the same lexical contract as formal teaching slots, with source
	// bindings removed before classifying newly supplied prose.
	plain := text
	for _, p := range a.unit.Protected {
		plain = strings.ReplaceAll(plain, p.Token, "")
	}
	_, added, err := protectBlockMode([]byte(plain), "teaching-comment", true)
	if err != nil {
		return nil, err
	}
	for _, p := range added {
		// A closed, canonical technical namespace is ordinary teaching prose
		// authority too. English plural morphology ("IDs") can legitimately
		// become the registered atom "ID" in another language. This does not
		// authorize a new code identity, literal, path or structural binding.
		if p.Role != "atom" || p.Binding != nil || classifyProseToken(p.Raw, "", "") != knownTechnicalAtom {
			return nil, fmt.Errorf("new unprotected teaching identity: %s", p.Raw)
		}
	}
	// Existing source-bound atoms may move, but cannot be duplicated as bare
	// prose alongside their required unique token.
	counts := func(s string) map[string]int {
		m := map[string]int{}
		for _, id := range teachingASCIIIdentifier.FindAllString(s, -1) {
			m[id]++
		}
		return m
	}
	wantAtoms, gotAtoms := counts(a.original), counts(raw)
	for _, p := range a.unit.Protected {
		if p.Raw != "" && teachingASCIIIdentifier.FindString(p.Raw) == p.Raw && gotAtoms[p.Raw] != wantAtoms[p.Raw] {
			return nil, fmt.Errorf("teaching binding count changed: %s", p.Raw)
		}
	}
	return []byte(raw), nil
}

func (a PageAdapter) CommentRevisionBodies(candidate []byte) ([]i18n.CommentRevisionBody, error) {
	source, title, err := a.CommentSourceBlocks()
	if err != nil {
		return nil, err
	}
	for i, b := range source {
		source[i] = i18n.SourceCommentBlockDisposition(b, title, nil)
	}
	target, err := a.CommentTargetBlocks(candidate)
	if err != nil {
		return nil, err
	}
	target = i18n.SourceSelectedTargetCommentBlocks(source, target)
	out := []i18n.CommentRevisionBody{}
	for bi, b := range target {
		if bi >= len(source) || b.Unsupported {
			return nil, fmt.Errorf("comment revision source-selected grammar unavailable")
		}
		for ci, c := range b.Comments {
			if ci >= len(source[bi].Comments) || source[bi].Comments[ci].Kind != "teaching-comment" {
				continue
			}
			start, err := commentRawCoordinate(candidate, b, c)
			if err != nil {
				return nil, fmt.Errorf("%s %s comment %d: %w", a.Document.Path, b.Coordinate, c.Position, err)
			}
			left, right := i18n.CommentPayloadBounds(c.Text)
			body := i18n.TeachingCommentBody{TeachingSpan: i18n.TeachingSpan{Start: 0, End: right - left}}
			for _, s := range i18n.SafeTeachingCommentBodies(b.Code) {
				if s.Start == c.Start+left && s.End == c.Start+right {
					for _, id := range s.Identifiers {
						body.Identifiers = append(body.Identifiers, i18n.TeachingSpan{Start: id.Start - s.Start, End: id.End - s.Start})
					}
					break
				}
			}
			original := string(candidate[start+left : start+right])
			text, locked, err := protectTeachingBody([]byte(original), 0, len(original), body)
			if err != nil {
				return nil, err
			}
			u := Unit{Source: text, Protected: locked}
			out = append(out, i18n.CommentRevisionBody{Coordinate: b.Coordinate, Position: c.Position, Start: start + left, End: start + right, Adapter: revisionCommentBody{unit: u, original: original}})
		}
	}
	return out, nil
}
