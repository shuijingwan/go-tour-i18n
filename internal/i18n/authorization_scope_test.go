package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFinalizedPackageAuthorizationScopeIsolation(t *testing.T) {
	root, c, _, snapshot := checkpointFixture(t)
	a, path, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(filepath.Join(root, path))
	foreign := c.TranslationPackages[0].Units[0]
	foreign.Unit.ID, foreign.Unit.SourcePath, foreign.Route = "other/home", "fixtures/other/home.xml", "/other/"
	writeSnapshotSource(t, root, foreign.Unit.SourcePath, foreign.Unit.Source)
	if err := c.RegisterTranslationPackage(TranslationPackageDescriptor{ID: "other-shell", AuthoritySHA256: sum([]byte("other-authority")), Units: []AdaptedTranslationUnit{foreign}}); err != nil {
		t.Fatal(err)
	}
	c.TranslationScope = nil
	scoped, registered, err := SelectFinalizedTranslationPackage(root, c, "zh-CN", snapshot)
	if err != nil || !registered || len(c.TranslationScope) != 0 {
		t.Fatalf("scope inference: %v %v", registered, err)
	}
	checked, err := RequireCurrentTranslationAuthorization(root, scoped, "zh-CN", snapshot)
	if err != nil || !reflect.DeepEqual(a, checked) {
		t.Fatalf("foreign Homepage changed checkpoint: %v", err)
	}
	for _, u := range checked.Units {
		if u.Snapshot.UnitID == foreign.Unit.ID {
			t.Fatal("foreign Homepage entered scope")
		}
	}
	after, _ := os.ReadFile(filepath.Join(root, path))
	if !bytes.Equal(original, after) {
		t.Fatal("trusted authorization rewritten")
	}
	if err := c.UseTranslationPackage("other-shell"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := SelectFinalizedTranslationPackage(root, c, "zh-CN", snapshot); err == nil {
		t.Fatal("explicit wrong package ignored")
	}
	// Exact membership includes order and source coordinates, not just counts.
	for _, mutation := range []string{"missing", "order", "foreign-coordinate"} {
		t.Run(mutation, func(t *testing.T) {
			copy := *scoped
			copy.TranslationPackages = append([]TranslationPackageDescriptor(nil), scoped.TranslationPackages...)
			p := &copy.TranslationPackages[0]
			p.Units = append([]AdaptedTranslationUnit(nil), p.Units...)
			switch mutation {
			case "missing":
				p.Units = p.Units[1:]
			case "order":
				p.Units[0], p.Units[1] = p.Units[1], p.Units[0]
			case "foreign-coordinate":
				p.Units[0].Unit.SourcePath = foreign.Unit.SourcePath
				copy.Adapted = append([]AdaptedTranslationUnit(nil), scoped.Adapted...)
				copy.Adapted[0].Unit.SourcePath = foreign.Unit.SourcePath
			}
			_, registered, err := SelectFinalizedTranslationPackage(root, &copy, "zh-CN", snapshot)
			if err == nil && registered {
				t.Fatal("nonexact package inferred")
			}
			if _, err := RequireCurrentTranslationAuthorization(root, &copy, "zh-CN", snapshot); err == nil {
				t.Fatal("nonexact membership authorized")
			}
		})
	}
}

// Model a representation successor: one physical Page/kind, distinct frozen
// authorities and Unit IDs, independent A judgments and a finalized old edge.
func finalizedBoundSuccessorFixture(t *testing.T) (string, *Catalog, string, string) {
	t.Helper()
	root, source, _ := futureContentFixture(t)
	d := source.TranslationPackages[0]
	d.ID, d.AuthoritySHA256, d.Units = "future-selector-package", sum([]byte("old-selector-authority")), d.Units[:1]
	old := &Catalog{}
	if err := old.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := old.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	oldID := d.Units[0].Unit.ID
	checkpointGenerate(t, root, old, []string{oldID})
	const previous, snapshot = "selector-historical", "selector-current"
	futureSnapshot(t, root, old, previous)
	futureRecord(t, root, old, previous, "", "A", "", []string{oldID})
	if _, _, err := FinalizeQualityCheck(root, old, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: previous}); err != nil {
		t.Fatal(err)
	}
	d.Units = append([]AdaptedTranslationUnit(nil), d.Units...)
	d.AuthoritySHA256 = sum([]byte("current-selector-authority"))
	d.Units[0].Unit.ID = sum([]byte(oldID + "-new-representation"))
	current := &Catalog{HistoricalTranslationPackages: old.TranslationPackages}
	if err := current.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := current.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	id := d.Units[0].Unit.ID
	checkpointGenerate(t, root, current, []string{id})
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, current, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot, ContentPackage: d.ID}); err != nil {
		t.Fatal(err)
	}
	futureRecord(t, root, current, snapshot, previous, "A", "", []string{id})
	if _, _, err := FinalizeQualityCheck(root, current, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: previous}); err != nil {
		t.Fatal(err)
	}
	return root, current, snapshot, previous
}

func TestFinalizedPackageExplicitBindingWithHistoricalPredecessor(t *testing.T) {
	root, c, snapshot, previous := finalizedBoundSuccessorFixture(t)
	current, registered, err := SelectFinalizedTranslationPackage(root, c, "zh-CN", snapshot)
	if err != nil || !registered || current.TranslationPackages[0].AuthoritySHA256 != c.TranslationPackages[0].AuthoritySHA256 {
		t.Fatal("explicit current authority", err)
	}
	a, err := AssessCurrentTranslation(root, current, "zh-CN", snapshot)
	if err != nil || !a.Ready() || a.Compatible != 1 {
		t.Fatal("current authorization assessment", err)
	}
	d := c.HistoricalTranslationPackages[0]
	historical, err := HistoricalTranslationPackageCatalog(c, d.ID, d.AuthoritySHA256)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", previous, historical)
	if err != nil || len(proof.Finalization.Units) != 1 || proof.Finalization.Units[0].UnitID != d.Units[0].Unit.ID {
		t.Fatal("predecessor historical integrity", err)
	}
	if a.PackageAuthoritySHA == d.AuthoritySHA256 {
		t.Fatal("old authority leaked into current authorization")
	}
	// A legacy predecessor still has no explicit scope. Its original frozen
	// catalog resolves it; the mixed current inventory must remain ambiguous.
	if _, registered, err := SelectFinalizedTranslationPackage(root, historical, "zh-CN", previous); err != nil || !registered {
		t.Fatal("frozen predecessor selection", err)
	}
	if _, _, err := SelectFinalizedTranslationPackage(root, c, "zh-CN", previous); err == nil || !strings.Contains(err.Error(), "ambiguous finalized package scope") {
		t.Fatal("arbitrary legacy ambiguity accepted", err)
	}
	// Duplicate discovery of the SAME descriptor is still one authority.
	duplicate := *c
	duplicate.HistoricalTranslationPackages = append(append([]TranslationPackageDescriptor{}, c.HistoricalTranslationPackages...), c.TranslationPackages[0])
	if _, _, err := SelectFinalizedTranslationPackage(root, &duplicate, "zh-CN", snapshot); err != nil {
		t.Fatal("exact duplicate discovery", err)
	}
}

func TestFinalizedPackageExplicitBindingFailClosed(t *testing.T) {
	root, c, snapshot, _ := finalizedBoundSuccessorFixture(t)
	manifestPath := "data/quality-check-snapshots/zh-CN/" + snapshot + "/manifest.json"
	finalPath := "data/quality-check-snapshots/zh-CN/" + snapshot + "/finalization.json"
	manifest, _ := os.ReadFile(filepath.Join(root, manifestPath))
	final, _ := os.ReadFile(filepath.Join(root, finalPath))
	for _, name := range []string{"unknown-authority", "wrong-unit-ids", "source-path", "source-sha", "unrelated-historical-authority", "manifest-hash", "finalization-unit"} {
		t.Run(name, func(t *testing.T) {
			defer writeSnapshotSource(t, root, manifestPath, manifest)
			defer writeSnapshotSource(t, root, finalPath, final)
			var s QualityCheckSnapshotManifest
			var f QualityCheckFinalization
			if err := json.Unmarshal(manifest, &s); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(final, &f); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "unknown-authority":
				s.ContentPackage.AuthoritySHA256 = sum([]byte("unregistered"))
			case "wrong-unit-ids":
				s.ContentPackage.UnitIDs[0] = "unrelated-id"
			case "source-path":
				s.Units[0].SourcePath = "fixtures/wrong.xml"
			case "source-sha":
				s.Units[0].SourceSHA256 = sum([]byte("wrong source"))
			case "unrelated-historical-authority":
				s.ContentPackage.AuthoritySHA256 = c.HistoricalTranslationPackages[0].AuthoritySHA256
			case "finalization-unit":
				f.Units[0].Snapshot.UnitID = "unrelated-finalization-id"
			}
			b := mustJSON(s)
			writeSnapshotSource(t, root, manifestPath, b)
			if name != "manifest-hash" {
				f.SnapshotManifestSHA256 = sum(b)
			}
			// Keep all other frozen coordinates consistent so malformed binding
			// is rejected by the real selector, not only an outer file hash.
			if name != "finalization-unit" {
				f.Units[0].Snapshot = s.Units[0]
			}
			writeSnapshotSource(t, root, finalPath, mustJSON(f))
			if _, _, err := SelectFinalizedTranslationPackage(root, c, "zh-CN", snapshot); err == nil {
				t.Fatal("invalid explicit/finalization binding selected", name)
			}
		})
	}
}
