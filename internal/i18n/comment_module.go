package i18n

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
)

// AnalyzeModuleCommentCode requires source-owned go.mod authority at its caller.
// Parse (not ParseLax) owns the whole grammar, quoting and comment coordinates.
func AnalyzeModuleCommentCode(code, coordinate string) CommentCodeBlock {
	b := CommentCodeBlock{Coordinate: coordinate, Language: "gomod", Extractor: "modfile-strict/v1", Code: code, SHA256: sum([]byte(code)), Comments: []CodeComment{}}
	f, err := modfile.Parse("go.mod", []byte(code), nil)
	if err != nil || len(f.Syntax.Stmt) == 0 {
		b.Unsupported = true
		b.Diagnostic = "unsupported concrete go.mod grammar"
		return b
	}
	return analyzeModuleSyntax(b, f)
}

func AnalyzeWorkCommentCode(code, coordinate string) CommentCodeBlock {
	b := CommentCodeBlock{Coordinate: coordinate, Language: "gowork", Extractor: "modfile-work-strict/v1", Code: code, SHA256: sum([]byte(code)), Comments: []CodeComment{}}
	f, err := modfile.ParseWork("go.work", []byte(code), nil)
	if err != nil || len(f.Syntax.Stmt) == 0 {
		b.Unsupported = true
		b.Diagnostic = "unsupported complete go.work grammar"
		return b
	}
	return analyzeModuleSyntax(b, &modfile.File{Syntax: f.Syntax})
}

func analyzeModuleSyntax(b CommentCodeBlock, f *modfile.File) CommentCodeBlock {
	code := b.Code
	machine := map[int]bool{}
	mark := func(c *modfile.Comments) {
		for _, list := range [][]modfile.Comment{c.Before, c.Suffix, c.After} {
			for _, comment := range list {
				machine[comment.Start.Byte] = true
			}
		}
	}
	for _, r := range f.Retract {
		mark(r.Syntax.Comment())
	}
	if f.Module != nil && f.Module.Deprecated != "" {
		mark(f.Module.Syntax.Comment())
	}
	type located struct {
		comment modfile.Comment
		owner   int
	}
	all := []located{}
	seen := map[int]bool{}
	owner := 0
	var walk func(modfile.Expr)
	walk = func(e modfile.Expr) {
		owner++
		c := e.Comment()
		for _, list := range [][]modfile.Comment{c.Before, c.Suffix, c.After} {
			for _, comment := range list {
				if !seen[comment.Start.Byte] {
					seen[comment.Start.Byte] = true
					all = append(all, located{comment, owner})
				}
			}
		}
		switch n := e.(type) {
		case *modfile.FileSyntax:
			for _, s := range n.Stmt {
				walk(s)
			}
		case *modfile.LineBlock:
			walk(&n.LParen)
			for _, line := range n.Line {
				walk(line)
			}
			walk(&n.RParen)
		}
	}
	walk(f.Syntax)
	sort.Slice(all, func(i, j int) bool { return all[i].comment.Start.Byte < all[j].comment.Start.Byte })
	var skeleton strings.Builder
	pos := 0
	for _, item := range all {
		comment := item.comment
		a, z := comment.Start.Byte, comment.Start.Byte+len(comment.Token)
		if a < pos || z > len(code) || code[a:z] != comment.Token {
			b.Unsupported = true
			b.Diagnostic = "go.mod comment coordinate mismatch"
			b.Comments = []CodeComment{}
			return b
		}
		_, _, body := goCommentPayload([]byte(code), a, z, comment.Token)
		kind := "non-language/structural-comment"
		if machine[a] || strings.TrimSpace(body) == "indirect" {
			kind = "machine-semantic-comment"
		} else {
			switch classifyGoExampleComment(comment.Token, body) {
			case goExampleCommentNatural:
				kind = "teaching-comment"
			case goExampleCommentNonNatural:
				if isTranslatableTeachingComment(body) {
					kind = "teaching-comment"
				}
			default:
				kind = "machine-semantic-comment"
			}
		}
		b.Comments = append(b.Comments, CodeComment{Position: len(b.Comments) + 1, TokenPosition: item.owner, Start: a, End: z, Kind: kind, Policy: "protected", Text: code[a:z], Body: body, Style: "line"})
		skeleton.WriteString(code[pos:a])
		skeleton.WriteString("⟦comment:line⟧")
		pos = z
	}
	skeleton.WriteString(code[pos:])
	b.Skeleton = skeleton.String()
	return b
}

func ModuleCommentAuthority(declaredLanguage, metadataTitle string) (string, bool) {
	switch declaredLanguage {
	case "gomod", "go.mod":
		return "explicit fence/class language=" + declaredLanguage, true
	}
	// This metadata explicitly declares a file-format reference. It does not
	// authorize arbitrary command/output blocks: strict full-file parsing above
	// remains required. A path containing "mod" is never language authority.
	if declaredLanguage == "" && metadataTitle == "go.mod file reference" {
		return fmt.Sprintf("frozen Page metadata title=%q", metadataTitle), true
	}
	return "", false
}
