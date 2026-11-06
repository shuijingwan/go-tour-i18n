package i18n

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

// TourCommentFreeProjection proves a narrow diagnostic fragment against an
// actual Go Example referenced by this parent Page. It never infers a language
// from prose, identifiers, a filename, or a localized candidate.
func TourCommentFreeProjection(code string, examples []Example) (string, bool) {
	events, err := scanGoExampleLexicalEvents([]byte(code))
	if err != nil || len(events) == 0 {
		return "", false
	}
	for _, e := range events {
		if e.token == token.COMMENT {
			return "", false
		}
	}
	render := func(n ast.Node) string {
		var b bytes.Buffer
		if format.Node(&b, token.NewFileSet(), n) != nil {
			return ""
		}
		return b.String()
	}
	params := func(fields *ast.FieldList) string {
		parts := []string{}
		if fields == nil {
			return ""
		}
		for _, f := range fields.List {
			if len(f.Names) == 0 {
				return ""
			}
			for _, name := range f.Names {
				parts = append(parts, name.Name+":"+render(f.Type))
			}
		}
		return strings.Join(parts, ";")
	}
	parameterShape := ""
	if f, err := parser.ParseFile(token.NewFileSet(), "parameters.go", "package p\nfunc _("+strings.TrimSpace(code)+") {}", parser.SkipObjectResolution); err == nil && len(f.Decls) == 1 {
		parameterShape = params(f.Decls[0].(*ast.FuncDecl).Type.Params)
	}
	rangeShape := func(r *ast.RangeStmt) string {
		copy := *r
		copy.Body = &ast.BlockStmt{}
		if id, ok := copy.Value.(*ast.Ident); ok && id.Name == "_" {
			copy.Value = nil
		}
		return render(&copy)
	}
	ranges := []string{}
	// Only headers on complete physical lines are supported. Strings/characters
	// can own newlines, so this projection deliberately rejects them.
	for _, e := range events {
		if e.token == token.STRING || e.token == token.CHAR {
			ranges = nil
			break
		}
	}
	if ranges != nil {
		for _, line := range strings.Split(strings.TrimSpace(code), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), "range.go", "package p\nfunc _(){\n"+line+" {}\n}", parser.SkipObjectResolution)
			if err != nil || len(f.Decls) != 1 {
				ranges = nil
				break
			}
			body := f.Decls[0].(*ast.FuncDecl).Body
			if len(body.List) != 1 {
				ranges = nil
				break
			}
			r, ok := body.List[0].(*ast.RangeStmt)
			if !ok {
				ranges = nil
				break
			}
			ranges = append(ranges, rangeShape(r))
		}
	}
	for _, example := range examples {
		if sum(example.Source) != example.SourceSHA256 {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), example.SourcePath, example.Source, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		paramMatch := false
		sourceRanges := map[string]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.FuncType:
				if parameterShape != "" && params(v.Params) == parameterShape {
					paramMatch = true
				}
			case *ast.RangeStmt:
				sourceRanges[rangeShape(v)] = true
			}
			return true
		})
		if paramMatch {
			return "Go parameter AST projection of " + example.SourcePath + "@" + example.SourceSHA256, true
		}
		if len(ranges) > 0 {
			all := true
			for _, r := range ranges {
				if !sourceRanges[r] {
					all = false
				}
			}
			if all {
				return "Go range-header AST projection of " + example.SourcePath + "@" + example.SourceSHA256 + "; blank value normalized by Go range assignment semantics", true
			}
		}
	}
	return "", false
}
