package sitecontent

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These calls are intentionally canonical while their implementation bytes are
// frozen. Guard the classification evidence, not a locale/batch/source hash.
func TestLegacyFrozenRebuildClassification(t *testing.T) {
	for _, item := range []struct{ file, caller, callee string }{
		{"translation_adapter.go", "AttachPageTranslationUnits", "WorkflowDocuments"},
		{"shell_successor.go", "BuildShellSuccessorAffectedScope", "WorkflowDocuments"},
		{"formal_page.go", "WorkflowDocuments", "PackageDocuments"},
		{"units.go", "PackageDocuments", "ParseDocument"},
	} {
		f, err := parser.ParseFile(token.NewFileSet(), item.file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != item.caller {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if id, ok := call.Fun.(*ast.Ident); ok && id.Name == item.callee {
						found = true
					}
				}
				return true
			})
		}
		if !found {
			t.Fatalf("%s -> %s changed; reassess frozen-boundary classification", item.caller, item.callee)
		}
	}
	for _, item := range []struct{ file, dependency string }{
		{"internal/i18n/same_unit_bridge.go", "internal/sitecontent/translation_adapter.go"},
		{"internal/i18n/same_unit_bridge.go", "internal/sitecontent/formal_page.go"},
		{"internal/sitecontent/translation_adapter.go", "internal/sitecontent/units.go"},
		{"internal/sitecontent/shell_successor.go", "internal/sitecontent/shell_successor.go"},
	} {
		b, err := os.ReadFile(filepath.Join(repositoryRoot(t), item.file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), `"`+item.dependency+`"`) {
			t.Fatalf("%s no longer freezes %s; reassess required rebuild", item.file, item.dependency)
		}
	}
}
