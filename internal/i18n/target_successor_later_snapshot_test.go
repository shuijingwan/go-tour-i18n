package i18n

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestSuccessorLaterSnapshotExactIdentity(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, catalog, _ := historicalRecoveryFixture(t)
	if _, _, err := RecoverHistoricalPromotion(root, catalog, "zh-CN", "qc-002", true); err != nil {
		t.Fatal(err)
	}
	assertAuthorized := func(snapshot string) {
		t.Helper()
		d, err := DiagnoseQualityCheckAuthorization(root, catalog, "zh-CN", snapshot)
		if err != nil || !d.CanonicalAuthorized {
			t.Fatal("exact installed/later Snapshot rejected", snapshot, d, err)
		}
	}
	assertAuthorized("qc-002")
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-003"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, catalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-003", PreviousSnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	assertAuthorized("qc-003")
	manifest := "data/quality-check-snapshots/zh-CN/qc-003/manifest.json"
	before, err := readCompatibilityFile(root, manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"candidate SHA", "validation SHA", "source SHA", "representation reconciliation"} {
		t.Run(kind, func(t *testing.T) {
			var s QualityCheckSnapshotManifest
			if err := contentidentity.StrictJSON(before, &s); err != nil {
				t.Fatal(err)
			}
			u := &s.Units[0]
			switch kind {
			case "candidate SHA":
				u.CandidateSHA256 = strings.Repeat("0", 64)
			case "validation SHA":
				u.ValidationSHA256 = strings.Repeat("0", 64)
			case "source SHA":
				u.SourceSHA256 = strings.Repeat("0", 64)
			case "representation reconciliation":
				u.RepresentationReconciliation = &GlossaryArchiveReference{Path: "data/unauthorized-reconciliation.json", SHA256: strings.Repeat("0", 64)}
			}
			defer writeSnapshotSource(t, root, manifest, bytes.Clone(before))
			writeSnapshotSource(t, root, manifest, mustJSON(s))
			if d, err := DiagnoseQualityCheckAuthorization(root, catalog, "zh-CN", "qc-003"); err == nil && d.CanonicalAuthorized {
				t.Fatal("changed identity borrowed installation authority", kind)
			}
		})
	}
	if d, err := DiagnoseQualityCheckAuthorization(root, catalog, "zh-CN", "qc-001"); err == nil && d.CanonicalAuthorized {
		t.Fatal("wrong predecessor borrowed current installation authority")
	}
	assertAuthorized("qc-003")
}

func TestSuccessorSupersededSnapshotCannotAuthorizeCurrentCandidate(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, catalog, _, _, _ := nativeMultiEdgeLineageFixture(t)
	d, err := DiagnoseQualityCheckAuthorization(root, catalog, "zh-CN", "qc-003")
	if err != nil || !d.CanonicalAuthorized {
		t.Fatal("current native installation rejected", d, err)
	}
	d, err = DiagnoseQualityCheckAuthorization(root, catalog, "zh-CN", "qc-002")
	if err != nil || d.CanonicalAuthorized || d.Reason != "successor Snapshot identity mismatch: lesson/2" {
		t.Fatal("superseded accepted candidate must not authorize newer canonical bytes", d, err)
	}
}
