package i18n

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestSurfaceCandidateBatchReusesMachinePreparation(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, c, original, plan, ref, _ := authorizationMachineFixture(t)
	s := *original
	s.SnapshotID = "surface-machine-batch"
	s.Units = nil
	artifacts := []SurfaceCandidateArtifact{}
	for i, u := range plan.Receipt.Units {
		unit := (&machineSelection{receipt: plan.Receipt, unit: u, ref: ref}).snapshot(i + 1)
		s.Units = append(s.Units, unit)
		artifacts = append(artifacts, SurfaceCandidateArtifact{SnapshotID: s.SnapshotID, Unit: unit})
	}
	manifest := "data/quality-check-snapshots/zh-CN/" + s.SnapshotID + "/manifest.json"
	writeSnapshotSource(t, root, manifest, mustJSON(s))
	resolve := c.ReplacementResolver
	calls := 0
	c.ReplacementResolver = func(root string, c *Catalog, ref GlossaryArchiveReference) (*TranslationUnitReplacementResolution, error) {
		calls++
		return resolve(root, c, ref)
	}
	if err := ValidateSurfaceCandidateArtifacts(root, c, "zh-CN", artifacts); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("expected one preparation", calls)
	}
	for _, a := range artifacts {
		if err := ValidateSurfaceCandidateArtifact(root, c, "zh-CN", a); err != nil {
			t.Fatal("single/batch differential", err)
		}
	}
	if calls != 1+len(artifacts) {
		t.Fatal("single verifier no longer independent", calls)
	}
	for label, mutate := range map[string]func(*SurfaceCandidateArtifact){
		"unit-id":        func(a *SurfaceCandidateArtifact) { a.Unit.UnitID = artifacts[1].Unit.UnitID },
		"snapshot":       func(a *SurfaceCandidateArtifact) { a.SnapshotID = original.SnapshotID },
		"candidate-hash": func(a *SurfaceCandidateArtifact) { a.Unit.CandidateSHA256 = strings.Repeat("0", 64) },
		"expected-receipt-hash": func(a *SurfaceCandidateArtifact) {
			r := *a.Unit.RepresentationReconciliation
			r.SHA256 = strings.Repeat("0", 64)
			a.Unit.RepresentationReconciliation = &r
		},
	} {
		t.Run(label, func(t *testing.T) {
			bad := append([]SurfaceCandidateArtifact(nil), artifacts...)
			mutate(&bad[len(bad)-1])
			if err := ValidateSurfaceCandidateArtifacts(root, c, "zh-CN", bad); err == nil {
				t.Fatal("accepted invalid artifact after warm preparation")
			}
		})
	}
	// Mutate dependencies after frozen resolution, while preparation continues.
	// The final guard must reject even inputs no longer read by native Unit work.
	for label, name := range map[string]string{
		"historical-documents": plan.Receipt.Options.HistoricalDocuments.Path,
		"formal-impact":        plan.Receipt.Options.Impact.Path,
		"predecessor-snapshot": plan.Receipt.Snapshot.Path,
		"predecessor-qc":       plan.Receipt.QCInputs[0].Path,
		"receipt":              ref.Path,
		"candidate":            artifacts[0].Unit.CandidatePath,
		"validation":           artifacts[0].Unit.ValidationPath,
		"glossary":             glossaryReviewGlossaryPath("zh-CN"),
	} {
		t.Run(label, func(t *testing.T) {
			before, err := readSnapshotReferencedFile(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeSnapshotSource(t, root, name, before)
			c.ReplacementResolver = func(root string, c *Catalog, ref GlossaryArchiveReference) (*TranslationUnitReplacementResolution, error) {
				r, err := resolve(root, c, ref)
				writeSnapshotSource(t, root, name, append(bytes.Clone(before), '\n'))
				return r, err
			}
			if err := ValidateSurfaceCandidateArtifacts(root, c, "zh-CN", artifacts); err == nil {
				t.Fatal("shared preparation concealed mutation")
			}
		})
	}
}
