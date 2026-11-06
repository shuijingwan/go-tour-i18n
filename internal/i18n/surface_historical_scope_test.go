package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestSurfaceHistoricalReconciliationIncompleteQC(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, c, original, _, _, _ := authorizationMachineFixtureSize(t, 49)
	s := futureSnapshot(t, root, c, "surface-historical-incomplete")
	artifacts := []SurfaceCandidateArtifact{}
	for _, u := range s.Units {
		artifacts = append(artifacts, SurfaceCandidateArtifact{SnapshotID: s.SnapshotID, Unit: u})
	}
	path := qualityCheckResultsPath(root, "zh-CN", original.SnapshotID)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var q QualityCheckResults
	if err := json.Unmarshal(b, &q); err != nil {
		t.Fatal(err)
	}
	q.Results, q.ResultCount = q.Results[:48], 48
	writeSnapshotSource(t, root, filepath.ToSlash(strings.TrimPrefix(path, root+string(os.PathSeparator))), mustJSON(q))
	if err := ValidateSurfaceCandidatePackageArtifacts(root, c, "zh-CN", artifacts); err == nil || !strings.Contains(err.Error(), "complete initial independent QC required for machine reconciliation") {
		t.Fatal("historical 49 Snapshot / 48 QC accepted or wrong gate", err)
	}
}

func TestSurfaceCandidateExplicitPackageSnapshot(t *testing.T) {
	root, c, _ := packageSnapshotFixture(t)
	s, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "surface-explicit-package", ContentPackage: c.TranslationPackages[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	a := SurfaceCandidateArtifact{SnapshotID: s.SnapshotID, Unit: s.Units[0]}
	if err := ValidateSurfaceCandidatePackageArtifacts(root, c, "zh-CN", []SurfaceCandidateArtifact{a}); err != nil {
		t.Fatal("complete 1-to-1 package metadata reader", err)
	}
	if err := ValidateSurfaceCandidateArtifact(root, c, "zh-CN", a); err != nil {
		t.Fatal("current artifact reader metadata regression", err)
	}
	s.ContentPackage.AuthoritySHA256 = sum([]byte("unregistered historical package"))
	writeSnapshotSource(t, root, "data/quality-check-snapshots/zh-CN/"+s.SnapshotID+"/manifest.json", mustJSON(s))
	if err := ValidateSurfaceCandidatePackageArtifacts(root, c, "zh-CN", []SurfaceCandidateArtifact{a}); err == nil {
		t.Fatal("unregistered package-era authority accepted")
	}
}
