package sitecontent

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Explicit readonly real-receipt probe. The before run uses a Go overlay only
// to print the native gate's exact operands; no formal bytes are changed.
func TestHistoricalReconciliationPackageScopeReadonly(t *testing.T) {
	mode := os.Getenv("GO_TOUR_HISTORICAL_PACKAGE_SCOPE_PROBE")
	if mode == "" {
		t.Skip("explicit readonly historical package scope proof")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	c, err := i18n.BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"learn-docs-qc014-machine-successors", "site-shell-qc001-machine-successors"} {
		t.Run(name, func(t *testing.T) {
			path := "data/translation-unit-reconciliations/zh-CN/" + name + "/receipt.json"
			b, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			var receipt i18n.RepresentationReconciliation
			if err := json.Unmarshal(b, &receipt); err != nil {
				t.Fatal(err)
			}
			scoped := *c
			if err := scoped.UseTranslationPackage(receipt.Options.Package); err != nil {
				t.Fatal(err)
			}
			u := receipt.Units[0].Replacement
			world := &scoped
			r, err := scoped.ReplacementResolver(root, &scoped, receipt.Options.HistoricalDocuments)
			if err != nil {
				t.Fatal(err)
			}
			if r.EventCatalog != nil {
				world = r.EventCatalog
			}
			_, _, err = i18n.NewRepresentationSuccessorVerifier(root, world, "zh-CN").VerifyHistorical(u.OldUnitID, u.CurrentUnitID)
			if mode == "before" && receipt.Options.Package == "learn-docs-v1" {
				if err == nil || !strings.Contains(err.Error(), "complete initial independent QC required for machine reconciliation") {
					t.Fatal("expected exact original gate", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			t.Logf("path=%s package=%s current_scope=%d error=%v", path, receipt.Options.Package, len(scoped.TranslationScope), err)
		})
	}
}
