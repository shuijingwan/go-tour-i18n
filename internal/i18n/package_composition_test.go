package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func finalizedCompositionFixture(t *testing.T) (string, *Catalog, []TranslationFinalizationInput) {
	t.Helper()
	root, c, ids := futureContentFixture(t)
	checkpointGenerate(t, root, c, ids)
	old := c.TranslationPackages[0]
	old.AuthoritySHA256 = sum([]byte("original-source-registry"))
	a, b := old, old
	a.Units = append([]AdaptedTranslationUnit{}, old.Units[:2]...)
	b.ID = "separate-source-adoption"
	b.AuthoritySHA256 = sum([]byte("adopted-source-registry"))
	b.Units = append([]AdaptedTranslationUnit{}, old.Units[2:]...)
	c.TranslationPackages = []TranslationPackageDescriptor{a, b}
	inputs := []TranslationFinalizationInput{}
	for i, d := range c.TranslationPackages {
		if err := c.UseTranslationPackage(d.ID); err != nil {
			t.Fatal(err)
		}
		snapshot := fmt.Sprintf("composition-predecessor-%d", i+1)
		futureSnapshot(t, root, c, snapshot)
		for _, u := range d.Units {
			futureRecord(t, root, c, snapshot, "", "A", "", []string{u.Unit.ID})
		}
		_, path, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot})
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, TranslationFinalizationInput{Package: d.ID, PackageAuthority: d.AuthoritySHA256, Finalization: GlossaryArchiveReference{Path: path, SHA256: sum(data)}})
	}
	c.HistoricalTranslationPackages = append([]TranslationPackageDescriptor{}, c.TranslationPackages...)
	old.AuthoritySHA256 = sum([]byte("unified-membership-successor"))
	c.TranslationPackages = []TranslationPackageDescriptor{old}
	if err := c.UseTranslationPackage(old.ID); err != nil {
		t.Fatal(err)
	}
	return root, c, inputs
}

func cloneCompositionFixture(t *testing.T, root string) string {
	t.Helper()
	out := t.TempDir()
	if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(out, rel), 0700)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(out, rel), b, 0600)
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestFinalizedPackageComposition(t *testing.T) {
	root, c, inputs := finalizedCompositionFixture(t)
	proof, files, err := ComposeFinalizedTranslationPackage(root, c, "zh-CN", inputs)
	if err != nil || len(files) != 3 || len(proof.Units) != 3 {
		t.Fatal("valid disjoint finalized histories", err)
	}
	again, _, err := ComposeFinalizedTranslationPackage(root, c, "zh-CN", inputs)
	if err != nil || !reflect.DeepEqual(proof, again) {
		t.Fatal("deterministic proof", err)
	}
	for _, a := range proof.Assessments {
		if !a.Ready() {
			t.Fatal("not current")
		}
	}
	// Composition emits no new Snapshot, A, finalization or checkpoint.
	entries, err := os.ReadDir(filepath.Join(root, "data/quality-check-snapshots/zh-CN"))
	if err != nil || len(entries) != 2 {
		t.Fatal("composition changed historical reviews", err)
	}
	tests := []struct {
		name   string
		mutate func(*testing.T, string, *Catalog, []TranslationFinalizationInput)
	}{
		{"missing-finalization", func(t *testing.T, r string, c *Catalog, in []TranslationFinalizationInput) {
			if err := os.Remove(filepath.Join(r, in[1].Finalization.Path)); err != nil {
				t.Fatal(err)
			}
		}},
		{"wrong-sha", func(t *testing.T, r string, c *Catalog, in []TranslationFinalizationInput) {
			in[1].Finalization.SHA256 = strings.Repeat("a", 64)
		}},
		{"wrong-descriptor", func(t *testing.T, r string, c *Catalog, in []TranslationFinalizationInput) {
			in[1].PackageAuthority = strings.Repeat("b", 64)
		}},
		{"duplicate-predecessor", func(t *testing.T, r string, c *Catalog, in []TranslationFinalizationInput) { in[1] = in[0] }},
		{"coverage-gap", func(t *testing.T, r string, c *Catalog, in []TranslationFinalizationInput) {
			d := c.TranslationPackages[0]
			u := d.Units[0]
			u.Unit.ID = "new-not-reviewed"
			u.StableIndex = 4
			d.Units = append(d.Units, u)
			c.TranslationPackages = []TranslationPackageDescriptor{d}
			c.Adapted = append(c.Adapted, u)
			c.TranslationScope = append(c.TranslationScope, u.Unit.ID)
		}},
		{"extra-unit", func(t *testing.T, r string, c *Catalog, in []TranslationFinalizationInput) {
			d := c.TranslationPackages[0]
			d.Units = d.Units[:2]
			c.TranslationPackages = []TranslationPackageDescriptor{d}
			c.TranslationScope = c.TranslationScope[:2]
		}},
		{"stale-source", func(t *testing.T, r string, c *Catalog, in []TranslationFinalizationInput) {
			p := c.Adapted[0].Unit.SourcePath
			data, _ := os.ReadFile(filepath.Join(r, p))
			writeSnapshotSource(t, r, p, append(data, ' '))
		}},
		{"candidate-drift", func(t *testing.T, r string, c *Catalog, in []TranslationFinalizationInput) {
			p := proof.Units[0].CandidatePath
			data, _ := os.ReadFile(filepath.Join(r, p))
			writeSnapshotSource(t, r, p, append(data, ' '))
		}},
		{"validation-drift", func(t *testing.T, r string, c *Catalog, in []TranslationFinalizationInput) {
			p := proof.Units[0].ValidationPath
			data, _ := os.ReadFile(filepath.Join(r, p))
			writeSnapshotSource(t, r, p, append(data, ' '))
		}},
		{"incompatible-current-glossary", func(t *testing.T, r string, c *Catalog, in []TranslationFinalizationInput) {
			writeSnapshotSource(t, r, "locales/zh-CN/glossary.yaml", []byte("locale: zh-CN\nterms: []\n"))
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := cloneCompositionFixture(t, root)
			copy := *c
			in := append([]TranslationFinalizationInput{}, inputs...)
			copy.Adapted = append([]AdaptedTranslationUnit{}, c.Adapted...)
			copy.TranslationScope = append([]string{}, c.TranslationScope...)
			tc.mutate(t, r, &copy, in)
			if _, _, err := ComposeFinalizedTranslationPackage(r, &copy, "zh-CN", in); err == nil {
				t.Fatal("invalid composition authorized")
			}
		})
	}
}

func TestHistoricalPackageSelectionAfterMembershipSuccessor(t *testing.T) {
	root, c, inputs := finalizedCompositionFixture(t)
	for _, in := range inputs {
		data, _ := os.ReadFile(filepath.Join(root, in.Finalization.Path))
		var f QualityCheckFinalization
		if err := json.Unmarshal(data, &f); err != nil {
			t.Fatal(err)
		}
		inventory := *c
		inventory.TranslationScope = nil
		h, registered, err := SelectFinalizedTranslationPackage(root, &inventory, "zh-CN", f.SnapshotID)
		if err != nil || !registered || len(h.TranslationScope) != len(f.Units) {
			t.Fatal("historical complete scope selection", err)
		}
		if _, err := ReadHistoricalTranslationIntegrity(root, h, "zh-CN", f.SnapshotID); err != nil {
			t.Fatal(err)
		}
		if _, registered, err := SelectFinalizedTranslationPackage(root, h, "zh-CN", f.SnapshotID); err != nil || !registered {
			t.Fatal("nested historical reader duplicated exact descriptor", err)
		}
	}
	if err := c.UseTranslationPackage(inputs[1].Package); err == nil {
		t.Fatal("historical identity scheduled")
	}
}

type compositionMutationAdapter struct {
	futureCardAdapter
	mutate func()
}

func (a compositionMutationAdapter) Validate(candidate []byte, g *Glossary) error {
	if err := a.futureCardAdapter.Validate(candidate, g); err != nil {
		return err
	}
	a.mutate()
	return nil
}

func TestFinalizedPackageCompositionRejectsConcurrentHistoryMutation(t *testing.T) {
	root, c, inputs := finalizedCompositionFixture(t)
	p := inputs[0].Finalization.Path
	before, err := os.ReadFile(filepath.Join(root, p))
	if err != nil {
		t.Fatal(err)
	}
	changed := false
	adapter := c.HistoricalTranslationPackages[1].Units[0].Adapter.(futureCardAdapter)
	c.HistoricalTranslationPackages = append([]TranslationPackageDescriptor{}, c.HistoricalTranslationPackages...)
	d := c.HistoricalTranslationPackages[1]
	d.Units = append([]AdaptedTranslationUnit{}, d.Units...)
	d.Units[0].Adapter = compositionMutationAdapter{futureCardAdapter: adapter, mutate: func() { changed = true; writeSnapshotSource(t, root, p, append(append([]byte{}, before...), ' ')) }}
	c.HistoricalTranslationPackages[1] = d
	if _, _, err := ComposeFinalizedTranslationPackage(root, c, "zh-CN", inputs); err == nil {
		t.Fatal("concurrent history mutation authorized")
	}
	if !changed {
		t.Fatal("test did not reach mutation boundary")
	}
}
