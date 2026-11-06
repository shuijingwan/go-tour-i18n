package i18n

import (
	"go/scanner"
	"go/token"
	"strings"
	"text/template"
	"text/template/parse"
)

// AnalyzeTemplateGoComments accepts only the website's raw string action,
// through Go's template AST. Padding preserves every comment byte coordinate;
// the original template/code bytes remain in the immutable skeleton.
func AnalyzeTemplateGoComments(code, coordinate string) (CommentCodeBlock, bool) {
	t, err := template.New("code").Funcs(template.FuncMap{"raw": func(s string) string { return s }}).Parse(code)
	if err != nil {
		return CommentCodeBlock{}, false
	}
	var projected strings.Builder
	pos, actions := 0, 0
	type span struct{ start, end int }
	protected := []span{}
	fset := token.NewFileSet()
	file := fset.AddFile("source", -1, len(code))
	var scan scanner.Scanner
	scan.Init(file, []byte(code), func(token.Position, string) {}, scanner.ScanComments)
	for {
		p, tok, lit := scan.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.STRING || tok == token.CHAR || tok == token.COMMENT {
			a := file.Offset(p)
			protected = append(protected, span{a, a + len(lit)})
		}
	}
	for _, node := range t.Tree.Root.Nodes {
		switch n := node.(type) {
		case *parse.TextNode:
			projected.Write(n.Text)
			pos += len(n.Text)
		case *parse.ActionNode:
			if len(n.Pipe.Decl) != 0 || len(n.Pipe.Cmds) != 1 {
				return CommentCodeBlock{}, false
			}
			args := n.Pipe.Cmds[0].Args
			if len(args) != 2 {
				return CommentCodeBlock{}, false
			}
			id, ok := args[0].(*parse.IdentifierNode)
			if !ok || id.Ident != "raw" {
				return CommentCodeBlock{}, false
			}
			value, ok := args[1].(*parse.StringNode)
			if !ok || !strings.Contains("<>&", value.Text) || len(value.Text) != 1 {
				return CommentCodeBlock{}, false
			}
			end := strings.Index(code[pos:], "}}")
			if end < 0 {
				return CommentCodeBlock{}, false
			}
			end += pos + 2
			for _, e := range protected {
				if e.start <= pos && e.end > pos {
					return CommentCodeBlock{}, false
				}
			}
			projected.WriteString(value.Text)
			projected.WriteString(strings.Repeat(" ", end-pos-len(value.Text)))
			pos = end
			actions++
		default:
			return CommentCodeBlock{}, false
		}
	}
	if actions == 0 || pos != len(code) || !isSafeGoFragment(projected.String()) {
		return CommentCodeBlock{}, false
	}
	b := AnalyzeCommentCode(projected.String(), "go", coordinate, false)
	if b.Unsupported {
		return CommentCodeBlock{}, false
	}
	b.Code, b.SHA256, b.Extractor = code, sum([]byte(code)), "tour-go-template-literal/v1"
	var skeleton strings.Builder
	pos = 0
	for _, c := range b.Comments {
		skeleton.WriteString(code[pos:c.Start])
		skeleton.WriteString("⟦comment:" + c.Style + "⟧")
		pos = c.End
	}
	skeleton.WriteString(code[pos:])
	b.Skeleton = skeleton.String()
	return b, true
}
