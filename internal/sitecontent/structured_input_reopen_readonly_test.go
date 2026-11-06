package sitecontent

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestStructuredInputReopenReadonlyRepository(t *testing.T) {
	if os.Getenv("GO_TOUR_STRUCTURED_INPUT_REOPEN_READONLY") != "1" {
		t.Skip("explicit real repository readonly acceptance")
	}
	root := repositoryRoot(t)
	fingerprint := func() map[string]string {
		out := map[string]string{}
		for _, name := range []string{"data/locale-language", "data/site-workflow", "data/site-shell", "data/quality-check-snapshots", "data/translation-authorizations", "data/translation-unit-reconciliations", "data/glossary-source-corpus.json", "data/unified-glossary-reviews", "locales/zh-CN"} {
			if err := filepath.WalkDir(filepath.Join(root, name), func(p string, e fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if e.IsDir() {
					return nil
				}
				raw, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				out[p] = digest(raw)
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
			t.Error("formal artifact mutation")
		}
	}()
	done := contentidentity.BeginMemoization()
	defer done()
	r, err := ReopenStructuredInput(root, "zh-CN", "source:_content/learn/books.yaml", false)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Ready || r.Applied || r.RawSourceChanged || !r.GenerationRequired || r.CurrentState != "stale" || r.OldUnits != 10 || r.CurrentUnits != 12 || r.Added != 2 || r.Removed != 0 || r.Changed != 0 || r.Receipt.OldAtomCount != 10 || r.Receipt.CurrentAtomCount != 12 || r.ProjectedState != "generation-required" || !reflect.DeepEqual(r.ProjectedExpected, []string{"locales/zh-CN/structured/_content/learn/books.yaml"}) {
		t.Fatalf("unexpected preflight: %+v", r)
	}
	if !reflect.DeepEqual(r.Receipt.AddedUnitIDs, []string{"82eb40a82fca4565b295bd3d3b5b8cddf4de20420a25ec6679c6ae624b7ef027", "28c82eeb784eff381e75e18f21e7be0ad1235e4a2a7f7525ca9771e07ed5f883"}) || r.Receipt.AddedAtoms[0].Source != "PDF" || r.Receipt.AddedAtoms[1].Source != "GitHub" {
		t.Fatal("unknown real semantic drift")
	}
	if r.Receipt.OldRawSHA != "929aff40ecf99c211788d4edc7bd88697e290a3f7976559f39ba6403dcc8f19a" || r.Receipt.CurrentRawSHA != r.Receipt.OldRawSHA || len(r.Receipt.RemovedUnitIDs) != 0 || len(r.Receipt.ChangedUnitIDs) != 0 {
		t.Fatal("unexpected source/old Unit drift")
	}
	raw, _ := Encode(r)
	if err := os.WriteFile("/tmp/go-tour-structured-input-reopen-preflight.json", raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("READY=true OLD_UNITS=10 CURRENT_UNITS=12 ADDED=PDF,GitHub PROJECTED_EXPECTED=%v RECEIPT=%s", r.ProjectedExpected, r.Reference.Path)
}
