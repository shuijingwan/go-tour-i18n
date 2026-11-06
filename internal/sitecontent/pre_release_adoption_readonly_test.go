package sitecontent

import (
	"os"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestPreReleaseAdoptionPreservesFinalizedPageScope(t *testing.T) {
	if os.Getenv("GO_TOUR_ADOPTION_SOURCE_PROOF") != "1" {
		t.Skip("explicit readonly current-authority acceptance fixture")
	}
	root := repositoryRoot(t)
	const base = "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-021/"
	final, err := readRegular(root, base+"finalization.json")
	if err != nil || digest(final) != "60ad7d7e1f6012f0dccdb52bbfd206d46918533272bca9473cad23de9ad29bcc" {
		t.Fatal("finalized predecessor bytes changed", err)
	}
	var f i18n.QualityCheckFinalization
	if err := StrictJSON(final, &f); err != nil {
		t.Fatal(err)
	}
	manifest, err := readRegular(root, base+"manifest.json")
	if err != nil || digest(manifest) != f.SnapshotManifestSHA256 {
		t.Fatal("Snapshot identity changed", err)
	}
	var snapshot i18n.QualityCheckSnapshotManifest
	if err := StrictJSON(manifest, &snapshot); err != nil {
		t.Fatal(err)
	}
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, _, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil || len(docs) != 49 || len(snapshot.Units) != 49 || len(f.Units) != 49 {
		t.Fatal("original exact set changed", err)
	}
	for i, d := range docs {
		u := snapshot.Units[i]
		if d.Units[0].ID != u.UnitID || d.Path != u.SourcePath || d.SourceSHA != u.SourceSHA256 || f.Units[i].UnitID != u.UnitID || f.Units[i].Rating != "A" {
			t.Fatal("original finalized Page membership/source/rating changed", i)
		}
		for _, ref := range []Reference{{u.CandidatePath, u.CandidateSHA256}, {u.ValidationPath, u.ValidationSHA256}} {
			if _, err := readRegularReference(root, ref); err != nil {
				t.Fatal("finalized candidate/validation drift", err)
			}
		}
	}
}
