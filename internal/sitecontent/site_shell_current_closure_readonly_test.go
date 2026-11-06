package sitecontent

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Opt-in Maintainer handoff: exercise the canonical current checker on the
// exact newly created closure, without creating evidence or changing coverage.
func TestMaintainerCurrentShellClosureReadonly(t *testing.T) {
	name := os.Getenv("GO_TOUR_CURRENT_SHELL_CLOSURE")
	if name == "" {
		t.Skip("Maintainer exact current shell closure check")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	ref := Reference{name, os.Getenv("GO_TOUR_CURRENT_SHELL_CLOSURE_SHA256")}
	c, files, err := CheckPackageClosure(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	old := Reference{"data/site-workflow/zh-CN/site-v2-shell/closure/41ec108c37469562da5f8870b4d3fcc84ce1afefda148da1d8bad091ca9717d3.json", "dcb7fcafd1708c4198747356996b51776c2e1c874bd86ff7d0110c31135e8ae9"}
	page := Reference{"data/quality-check-snapshots/zh-CN/site-v2-shell-qc-002/finalization.json", "fc5c325871ba9e29dc0b6e8693fbdb270038bc7d27a24f7a6a5cba641bd56ffb"}
	if c.Schema != ClosureSchema || c.Locale != "zh-CN" || c.Package != "site-v2-shell" || ref == old || c.PageFinalization == nil || *c.PageFinalization != page || c.StructuredBaseline == nil || *c.StructuredBaseline != old || len(c.Structured) != 0 {
		t.Fatal("current Page / inherited structured baseline binding mismatch")
	}
	// The old closure is historical here. Never run its CURRENT Page gate.
	var prior PackageClosure
	if err := readReference(root, old, &prior); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.Files, prior.Files) || len(files) != len(c.Files) {
		t.Fatal("recovery changed localized bytes/inventory")
	}
	for _, f := range c.Files {
		if digest(files[f.Path]) != f.SHA256 {
			t.Fatal("projected bytes differ", f.Path)
		}
	}
	if digest(files["data/site-shell/translation.html"]) != "ad273e0c1bf67de01d26009580e3eebb106dd673c58c57088dd733537562b8ad" {
		t.Fatal("translation carry changed")
	}
	report := map[string]any{"result": "PASS", "closure": ref, "identity_sha256": c.Identity, "page_finalization": c.PageFinalization, "structured_baseline_closure": c.StructuredBaseline, "inherited_structured_generations": prior.Structured, "localized_file_count": len(files), "localized_bytes_unchanged": true, "translation_carry": "PASS"}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("GO_TOUR_CURRENT_SHELL_CLOSURE_REPORT"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
