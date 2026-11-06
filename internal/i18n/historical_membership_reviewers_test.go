package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func composedReviewerFixture(t *testing.T) (string, *Catalog, *QualityCheckFinalization) {
	t.Helper()
	root, main, original, plan, _, _ := authorizationMachineFixtureSize(t, 49)
	resolution, err := main.ReplacementResolver(root, main, plan.Receipt.Options.HistoricalDocuments)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, resolution.HistoricalCatalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: original.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	primary := futureSnapshot(t, root, main, "composed-main-final")
	if _, _, err := FinalizeQualityCheck(root, main, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: primary.SnapshotID, PreviousSnapshotID: original.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	current, added := expandedMachinePackage(t, root, main, 2)
	sibling, err := HistoricalTranslationPackageCatalog(current, added.ID, added.AuthoritySHA256)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{added.Units[0].Unit.ID, added.Units[1].Unit.ID}
	checkpointGenerate(t, root, sibling, ids)
	first := futureSnapshot(t, root, sibling, "sibling-qc-001")
	futureRecord(t, root, sibling, first.SnapshotID, "", "A", "", ids[1:])
	second := futureSnapshot(t, root, sibling, "sibling-qc-002")
	futureRecord(t, root, sibling, second.SnapshotID, first.SnapshotID, "A", "", ids[:1])
	if _, _, err := FinalizeQualityCheck(root, sibling, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: second.SnapshotID, PreviousSnapshotID: first.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	declaration := TranslationPackageMembership{Predecessors: []TranslationPackagePredecessor{
		{ID: main.TranslationPackages[0].ID, AuthoritySHA256: main.TranslationPackages[0].AuthoritySHA256},
		{ID: added.ID, AuthoritySHA256: added.AuthoritySHA256},
	}}
	data := mustJSON(declaration)
	writeSnapshotSource(t, root, "fixtures/composed-membership.json", data)
	declaration.References = []GlossaryArchiveReference{{Path: "fixtures/composed-membership.json", SHA256: sum(data)}}
	pointer := []byte("fixture-current-membership-pointer\n")
	writeSnapshotSource(t, root, "data/content-package-memberships/current.json", pointer)
	declaration.References = append(declaration.References, GlossaryArchiveReference{Path: "data/content-package-memberships/current.json", SHA256: sum(pointer)})
	current.TranslationPackages[0].Membership = &declaration
	s, _, err := CreateQualityCheckCandidateSnapshot(root, current, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "composed-current-final", ContentPackage: current.TranslationPackages[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	futureRecord(t, root, current, s.SnapshotID, primary.SnapshotID, "A", "", []string{s.Units[24].UnitID})
	f, _, err := FinalizeQualityCheck(root, current, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: primary.SnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	return root, current, f
}

func TestHistoricalFinalizationComposedReviewerOrigins(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, c, f := composedReviewerFixture(t)
	if len(f.Units) != 51 || f.Units[49].SourceSnapshotID != "sibling-qc-002" || f.Units[50].SourceSnapshotID != "sibling-qc-001" {
		t.Fatal("fixture must retain actual sibling Reviewer origins")
	}
	for _, q := range f.QCResults {
		if strings.HasPrefix(q.SnapshotID, "sibling-") {
			t.Fatal("wire contract freezes primary response lineage")
		}
	}
	before, err := os.ReadFile(qualityCheckFinalizationPath(root, "zh-CN", f.SnapshotID))
	if err != nil {
		t.Fatal(err)
	}
	proof, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", f.SnapshotID, c)
	if err != nil || len(proof.ReviewerGlossaries) != 51 {
		t.Fatal("complete composed history", err)
	}
	for _, id := range []string{"sibling-qc-001", "sibling-qc-002"} {
		for _, name := range []string{"manifest.json", "quality-check-results.json"} {
			found := false
			for _, ref := range proof.References {
				found = found || ref.Path == "data/quality-check-snapshots/zh-CN/"+id+"/"+name
			}
			if !found {
				t.Fatal("flattened proof omitted actual Reviewer dependency", id, name)
			}
		}
	}
	if a, err := AssessCurrentTranslation(root, c, "zh-CN", f.SnapshotID); err != nil || !a.Ready() || a.Compatible != 51 {
		t.Fatal("current authorization cannot consume composed historical proof", a, err)
	}
	tests := []struct {
		name string
		edit func(string, *Catalog)
	}{
		{"missing-source-snapshot", func(r string, _ *Catalog) {
			removeComposedFixture(t, filepath.Join(r, "data/quality-check-snapshots/zh-CN/sibling-qc-001/manifest.json"))
		}},
		{"missing-sibling-result", func(r string, _ *Catalog) {
			removeComposedFixture(t, qualityCheckResultsPath(r, "zh-CN", "sibling-qc-002"))
		}},
		{"tampered-sibling-result", func(r string, _ *Catalog) {
			writeSnapshotSource(t, r, "data/quality-check-snapshots/zh-CN/sibling-qc-002/quality-check-results.json", []byte("{}"))
		}},
		{"non-A-with-rebound-result-sha", func(r string, _ *Catalog) {
			path := "data/quality-check-snapshots/zh-CN/sibling-qc-002/quality-check-results.json"
			var q QualityCheckResults
			readComposedFixtureJSON(t, r, path, &q)
			q.Results[0].Rating = "B"
			b := mustJSON(q)
			writeSnapshotSource(t, r, path, b)
			var child QualityCheckFinalization
			cp := "data/quality-check-snapshots/zh-CN/sibling-qc-002/finalization.json"
			readComposedFixtureJSON(t, r, cp, &child)
			for i := range child.QCResults {
				if child.QCResults[i].Path == path {
					child.QCResults[i].SHA256 = sum(b)
				}
			}
			writeSnapshotSource(t, r, cp, mustJSON(child))
		}},
		{"unrelated-exact-A-snapshot", func(r string, _ *Catalog) {
			var s QualityCheckSnapshotManifest
			var q QualityCheckResults
			readComposedFixtureJSON(t, r, "data/quality-check-snapshots/zh-CN/sibling-qc-002/manifest.json", &s)
			readComposedFixtureJSON(t, r, "data/quality-check-snapshots/zh-CN/sibling-qc-002/quality-check-results.json", &q)
			s.SnapshotID, q.SnapshotID = "unrelated-exact-A", "unrelated-exact-A"
			b := mustJSON(s)
			q.SnapshotManifestSHA256 = sum(b)
			writeSnapshotSource(t, r, "data/quality-check-snapshots/zh-CN/"+s.SnapshotID+"/manifest.json", b)
			writeSnapshotSource(t, r, "data/quality-check-snapshots/zh-CN/"+s.SnapshotID+"/quality-check-results.json", mustJSON(q))
			copy := *f
			copy.Units = append([]QualityCheckFinalizationUnit(nil), f.Units...)
			copy.Units[49].SourceSnapshotID = s.SnapshotID
			writeSnapshotSource(t, r, "data/quality-check-snapshots/zh-CN/"+f.SnapshotID+"/finalization.json", mustJSON(copy))
		}},
		{"candidate-mismatch", func(r string, _ *Catalog) {
			alterComposedChild(t, r, func(f *QualityCheckFinalization) { f.Units[0].Snapshot.CandidateSHA256 = sum([]byte("different")) })
		}},
		{"validation-mismatch", func(r string, _ *Catalog) {
			alterComposedChild(t, r, func(f *QualityCheckFinalization) { f.Units[0].Snapshot.ValidationSHA256 = sum([]byte("different")) })
		}},
		{"attempt-mismatch", func(r string, _ *Catalog) {
			alterComposedChild(t, r, func(f *QualityCheckFinalization) { f.Units[0].Snapshot.Attempt++ })
		}},
		{"membership-not-authorized", func(_ string, c *Catalog) { c.TranslationPackages[0].Membership = nil }},
		{"membership-reference-tamper", func(r string, _ *Catalog) {
			writeSnapshotSource(t, r, "fixtures/composed-membership.json", []byte("{}"))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := cloneCompositionFixture(t, root)
			copy := *c
			copy.TranslationPackages = append([]TranslationPackageDescriptor(nil), c.TranslationPackages...)
			tt.edit(r, &copy)
			if _, err := VerifyHistoricalQualityCheckFinalization(r, "zh-CN", f.SnapshotID, &copy); err == nil {
				t.Fatal("invalid sibling authority accepted")
			}
		})
	}
	t.Run("later-membership-pointer-preserves-history", func(t *testing.T) {
		r := cloneCompositionFixture(t, root)
		writeSnapshotSource(t, r, "data/content-package-memberships/current.json", []byte("later-membership-pointer\n"))
		after, err := VerifyHistoricalQualityCheckFinalization(r, "zh-CN", f.SnapshotID, c)
		if err != nil || !bytes.Equal(mustJSON(proof), mustJSON(after)) {
			t.Fatal("current pointer retroactively changed historical A", err)
		}
	})
	after, err := os.ReadFile(qualityCheckFinalizationPath(root, "zh-CN", f.SnapshotID))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("immutable parent changed", err)
	}
}

func readComposedFixtureJSON(t *testing.T, root, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}

func alterComposedChild(t *testing.T, root string, edit func(*QualityCheckFinalization)) {
	t.Helper()
	path := "data/quality-check-snapshots/zh-CN/sibling-qc-002/finalization.json"
	var f QualityCheckFinalization
	readComposedFixtureJSON(t, root, path, &f)
	edit(&f)
	writeSnapshotSource(t, root, path, mustJSON(f))
}

func removeComposedFixture(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}
