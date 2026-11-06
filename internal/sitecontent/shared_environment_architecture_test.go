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

// Guard ownership, not machine wall time. Consumers use the shared builder;
// only its observation/cache owner may enter the canonical cold derivation.
// Cross-process tests separately assert cold=1/warm=0 for actual calculations.
func TestSharedHeavyEnvironmentOwnership(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	owners := map[string]string{"buildGlobalCanonical": "observedGlobalSource", "deriveGlobalSource": "observedGlobalSource", "deriveHistoricalPageProjection": "historicalPageProjectionWithCounter", "derivePackageRepresentation": "sharedPackageRepresentation", "deriveHistoricalStructuredGraph": "sharedHistoricalStructuredGraph"}
	required := map[string]string{"BuildGlobal": "observedGlobalSource", "resolvePageReplacements": "sharedHistoricalPageProjection", "CurrentPackageDocuments": "sharedPackageRepresentation", "withBridgeConstructionEnvironment": "withReadonlyStructuredValidation", "BridgeRepresentationCatalog": "bridgeRepresentationCatalog", "resolveBridgeReconstruction": "originalBridgeCatalog", "reconstructFrozenStructuredRepresentation": "sharedHistoricalStructuredGraph", "observe": "observeSourcePath"}
	seen := map[string]bool{}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if assignment, ok := n.(*ast.AssignStmt); ok {
					for _, lhs := range assignment.Lhs {
						if field, ok := lhs.(*ast.SelectorExpr); ok && field.Sel.Name == "sharedSourceObservation" && fn.Name.Name != "withBridgeConstructionEnvironment" {
							t.Errorf("%s: %s extends the readonly bridge observation lifetime", name, fn.Name.Name)
						}
					}
				}
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if method, ok := call.Fun.(*ast.SelectorExpr); ok && required[fn.Name.Name] == method.Sel.Name {
					seen[fn.Name.Name] = true
				}
				id, ok := call.Fun.(*ast.Ident)
				if !ok {
					return true
				}
				if owner, guarded := owners[id.Name]; guarded && fn.Name.Name != owner {
					t.Errorf("%s: %s bypasses %s owner for %s", name, fn.Name.Name, owner, id.Name)
				}
				if id.Name == "WorkflowDocuments" && fn.Name.Name != "sharedWorkflowDocuments" && fn.Name.Name != "AttachPageTranslationUnits" && fn.Name.Name != "BuildShellSuccessorAffectedScope" {
					t.Errorf("%s: %s bypasses shared workflow owner", name, fn.Name.Name)
				}
				if required[fn.Name.Name] == id.Name {
					seen[fn.Name.Name] = true
				}
				return true
			})
		}
	}
	for caller, owner := range required {
		if !seen[caller] {
			t.Errorf("%s no longer consumes %s", caller, owner)
		}
	}
	b, err := os.ReadFile(filepath.Join(repositoryRoot(t), "cmd/tour-i18n/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), "main.go", b, 0)
	if err != nil {
		t.Fatal(err)
	}
	owned := false
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "run" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if ok && pkg.Name == "contentidentity" && sel.Sel.Name == "BeginMemoization" {
				owned = true
			}
			return true
		})
	}
	if !owned {
		t.Fatal("formal CLI invocation lost its shared deterministic environment scope")
	}
	b, err = os.ReadFile(filepath.Join(repositoryRoot(t), "cmd/tour-i18n/same_unit_bridge.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "sitecontent.BridgeRepresentationCatalog(root, c)") {
		t.Fatal("bridge CLI lost its eager construction data view")
	}
}
