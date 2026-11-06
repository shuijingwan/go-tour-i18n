package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestPackageClosureSurfaceReplacementContract(t *testing.T) {
	root := t.TempDir()
	for _, flags := range [][]string{
		{"--finding", "finding.json"}, {"--apply"}, {"--record"},
		{"--skip-currentness"}, {"--provider", "codex"},
		{"--prior-sha256", "override"}, {"positional"},
	} {
		args := append([]string{"package-closure", "--locale", "en-GB", "--package", "site-v2-shell"}, flags...)
		if err := siteContentCommand(root, args); err == nil {
			t.Fatal("closure accepted authority override", flags)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("invalid invocation mutated repository", err)
	}
	// CLI continues to consume published Generations, not a caller-supplied
	// finding override. The focused lifecycle test exercises both exact deltas.
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "SITE_V2_WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, term := range []string{"唯一的 affected Surface replacement", "shared structured representation successor", "structuredSupersedes", "exact accepted Generation owner", "current StructuredGeneration validation", "没有明确 representation successor", "ordinary delta", "docs → implementation → direct contract tests", "finding owner 的 exact package closure", "所有 scope closure References 仍先验证", "旧 first-real-use verification 不得 carry"} {
		if !strings.Contains(string(doc), term) {
			t.Fatal("canonical closure contract missing", term)
		}
	}
}

// Keep the documented wire names tied to the existing schemas without running
// a lifecycle command or treating Go field names as JSON field names.
func TestSiteWorkflowCommandFieldDocumentation(t *testing.T) {
	b, err := os.ReadFile("../../docs/SITE_V2_WORKFLOW.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	section := func(heading string) string {
		t.Helper()
		_, body, ok := strings.Cut(doc, "### "+heading+"\n")
		if !ok {
			t.Fatal("missing canonical section", heading)
		}
		for _, boundary := range []string{"\n### ", "\n## "} {
			body, _, _ = strings.Cut(body, boundary)
		}
		return body
	}
	closure := section("Package closure canonical command contract")
	scope := section("Surface scope canonical command contract")
	for _, schema := range []struct {
		body   string
		typeOf reflect.Type
		name   string
	}{
		{closure, reflect.TypeOf(sitecontent.PackageClosure{}), sitecontent.ClosureSchema},
		{scope, reflect.TypeOf(sitecontent.IntegratedSurfaceScope{}), sitecontent.IntegratedSurfaceSchema},
	} {
		if !strings.Contains(schema.body, "`"+schema.name+"`") {
			t.Fatal("documented schema differs from implementation", schema.name)
		}
		for i := 0; i < schema.typeOf.NumField(); i++ {
			name, _, _ := strings.Cut(schema.typeOf.Field(i).Tag.Get("json"), ",")
			if name == "" || !strings.Contains(schema.body, "`"+name+"`") {
				t.Fatal("missing exact JSON field", schema.typeOf.Name(), name)
			}
		}
	}
	for _, term := range []string{"JSON string containing the repository-relative newly/currently resolved closure path", "frozen structured baseline", "current exact delta set", "不把已 activation 的自身、前一个 successor 或后续 completion 自动选成 predecessor"} {
		if !strings.Contains(closure, term) {
			t.Fatal("missing closure output/baseline contract", term)
		}
	}
	if !strings.Contains(scope, "**PackageClosure References**") || !strings.Contains(scope, "当前 wire `schema` 为 **`"+sitecontent.IntegratedSurfaceSchema+"`**") {
		t.Fatal("surface-scope input kind or actual stdout schema missing")
	}
	fields := section("Shared re-attestation plan field semantics")
	for _, term := range []string{"`authority.changed_dependencies`", "`authority.validation_inputs`", "`old_currentness_inputs` → `current_currentness_inputs`", "CHANGED_DEPENDENCY_COUNT"} {
		if !strings.Contains(fields, term) {
			t.Fatal("missing re-attestation field distinction", term)
		}
	}
	b, err = os.ReadFile("../../docs/WORKFLOW_HANDOFF.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, term := range []string{"schema 未规定的关系", "structured_baseline_closure must equal previous successor closure", "changed_dependencies must contain every changed re-attestation validation file", "MUTATION=none", "MUTATION=known", "MUTATION=unknown", "POSTCONDITION", "禁止 replay"} {
		if !strings.Contains(string(b), term) {
			t.Fatal("missing verifier/provenance contract", term)
		}
	}
}
