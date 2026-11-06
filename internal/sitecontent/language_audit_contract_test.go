package sitecontent

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// Entry contract coverage guards the high-risk distinction between current
// acceptance and immutable historical decoding. Semantic behavior is tested
// separately with native Tour/registered/structured inputs, not mocked PASS.
func TestLanguageAuditCurrentEntryContracts(t *testing.T) {
	for _, check := range []struct{ file, function, callee string }{
		{"structured.go", "validateStructuredOutput", "structuredLanguageAttention"},
		{"structured.go", "computeCurrentStructuredGeneration", "bindLanguageAudit"},
		{"integrated_surface_bundle.go", "ExportIntegratedSurfaceReviewerBundle", "NewLanguageAttention"},
	} {
		f, err := parser.ParseFile(token.NewFileSet(), check.file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		foundFunction, foundCall := false, false
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name.Name != check.function {
				continue
			}
			foundFunction = true
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch x := call.Fun.(type) {
				case *ast.Ident:
					foundCall = foundCall || x.Name == check.callee
				case *ast.SelectorExpr:
					foundCall = foundCall || x.Sel.Name == check.callee
				}
				return true
			})
		}
		if !foundFunction || !foundCall {
			t.Fatal("current entry lost formal audit boundary", check)
		}
	}
	for _, file := range []string{"docs/LANGUAGE_DEFECT_EXHAUSTION.md", "docs/NEW_LOCALE_RUNBOOK.md", "docs/TRANSLATION_QUALITY_REVIEW.md", "docs/INTEGRATED_SURFACE_REVIEW.md"} {
		b, err := os.ReadFile(repositoryRoot(t) + "/" + file)
		if err != nil || !strings.Contains(string(b), "language") {
			t.Fatal("canonical contract absent", file, err)
		}
	}
}
