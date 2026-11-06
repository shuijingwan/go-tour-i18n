package sitecontent

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestStructuredPlanScopeReadonlyRepository(t *testing.T) {
	if os.Getenv("GO_TOUR_STRUCTURED_PLAN_SCOPE_READONLY") != "1" {
		t.Skip("explicit repository read-only structured-plan acceptance")
	}
	root := repositoryRoot(t)
	fingerprint := func() map[string]string {
		t.Helper()
		out := map[string]string{}
		for _, name := range []string{"data/quality-check-snapshots", "data/translation-authorizations", "data/translation-unit-reconciliations", "data/locale-language", "data/site-workflow", "data/site-shell", "data/glossary-source-corpus.json", "data/unified-glossary-reviews", "locales/zh-CN"} {
			if err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(name)), func(path string, e fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if e.IsDir() {
					return nil
				}
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				out[path] = digest(b)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	before := fingerprint()
	defer func() {
		if !reflect.DeepEqual(before, fingerprint()) {
			t.Error("FORMAL_ARTIFACT_MUTATION detected")
		}
	}()
	done := contentidentity.BeginMemoization()
	defer done()
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	count, _, _, err := i18n.LocaleWorkflowUnitCounts(c)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("CURRENT_CATALOG_UNIT_COUNT=%d CURRENT_SCOPE=Tour-native-default", count)
	for _, d := range c.TranslationPackages {
		t.Logf("REGISTERED_SCOPE=%s COUNT=%d AUTHORITY=%s", d.ID, len(d.Units), d.AuthoritySHA256)
	}
	const snapshot = "learn-docs-v1-qc-015"
	const finalizationPath = "data/quality-check-snapshots/zh-CN/" + snapshot + "/finalization.json"
	var finalized i18n.QualityCheckFinalization
	if err := readReference(root, Reference{finalizationPath, "530374a7aa563c103f975940b480f998fbd315541c56438e21d2557e22275b1d"}, &finalized); err != nil {
		t.Fatal(err)
	}
	var manifest i18n.QualityCheckSnapshotManifest
	if err := readReference(root, Reference{finalized.SnapshotManifestPath, finalized.SnapshotManifestSHA256}, &manifest); err != nil {
		t.Fatal(err)
	}
	selected, registered, err := i18n.SelectFinalizedTranslationPackage(root, c, "zh-CN", snapshot)
	if err != nil || !registered || len(selected.TranslationScope) != 49 || len(manifest.Units) != 49 || len(finalized.Units) != 49 {
		t.Fatal("qc-015 exact complete registered scope", registered, err)
	}
	for i, id := range selected.TranslationScope {
		u, err := selected.Unit(id)
		x := manifest.Units[i]
		if err != nil || x.UnitID != id || finalized.Units[i].UnitID != id || x.SourcePath != u.SourcePath || x.SourceSHA256 != u.SourceSHA256 || x.UnitKind != u.Kind || x.Index != i+1 {
			t.Fatal("qc-015 exact current membership/order/source", i+1, err)
		}
	}
	t.Log("QC015_EXACT_MEMBERSHIP=PASS QC015_UNIT_COUNT=49 FINALIZATION_SHA=unchanged")
	if os.Getenv("GO_TOUR_STRUCTURED_SCOPE_INVENTORY_ONLY") == "1" {
		return
	}
	p, err := PlanStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Log("STRUCTURED_PLAN_READONLY=FAILED", err)
		if os.Getenv("GO_TOUR_STRUCTURED_PLAN_EXPECT_PASS") == "1" {
			t.Fatal(err)
		}
		return
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/go-tour-structured-plan-scope.json", append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("STRUCTURED_PLAN_READONLY=PASS PLAN_EXPECTED_COUNT=%d PLAN_EXPECTED=%v", len(p.Expected), p.Expected)
	for _, a := range p.Assets {
		if a.State != "carry" {
			t.Logf("ASSET=%s STATE=%s ASSESSMENT=%+v", a.ID, a.State, a.Assessment)
		}
		if a.ID == "source:_content/learn/books.yaml" {
			t.Logf("BOOKS_STATE=%s BOOKS_CURRENT_UNIT_COUNT=%d", a.State, len(a.Document.Units))
			for _, u := range a.Document.Units {
				if u.Source == "PDF" || u.Source == "GitHub" {
					t.Logf("BOOKS_NEW_UNIT=%s SOURCE=%s", u.ID, u.Source)
				}
			}
		}
	}
}
