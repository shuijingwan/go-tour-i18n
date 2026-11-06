package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Opt-in recorded historical fixture: verify one complete replacement→machine
// branch, without AssessCurrentTranslation or the authorization CLI. Immutable
// repository evidence is read only; output is confined to the supplied /tmp dir.
func TestRecordedReplacementResponseMachineProof(t *testing.T) {
	dir := os.Getenv("GO_TOUR_RESPONSE_BRANCH_FIXTURE")
	if dir == "" {
		t.Skip("requires explicit recorded response-branch fixture")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	var fixture struct {
		Current         string `json:"MACHINE_CURRENT_UNIT_ID"`
		Old             string `json:"MACHINE_PREDECESSOR_UNIT_ID"`
		Path            string `json:"MACHINE_REPLACEMENT_SOURCE_PATH"`
		Snapshot        string `json:"FINALIZED_SNAPSHOT_ID"`
		FinalizationSHA string `json:"FINALIZATION_SHA256"`
	}
	b, err := os.ReadFile(filepath.Join(dir, "trigger-proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	sources, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	c, err := i18n.ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := i18n.HydrateCatalogSources(c, sources); err != nil {
		t.Fatal(err)
	}
	if err := attachCommandTranslationPackages(root, c, []string{"quality-check", "authorization"}); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	// Select from the immutable fixture Unit's registered package, not a CLI
	// authorization result or a cache's earlier currentness verdict.
	found := false
	for _, d := range c.TranslationPackages {
		for _, u := range d.Units {
			if u.Unit.ID == fixture.Current {
				if found {
					t.Fatal("ambiguous fixture package")
				}
				if err := c.UseTranslationPackage(d.ID); err != nil {
					t.Fatal(err)
				}
				found = true
			}
		}
	}
	if !found {
		t.Fatal("fixture current Unit missing")
	}
	mapping, refs, err := i18n.NewRepresentationSuccessorVerifier(root, c, "zh-CN").VerifyHistorical(fixture.Old, fixture.Current)
	if err != nil {
		t.Fatal("recorded replacement→machine historical proof", err)
	}
	if mapping == nil || mapping.OldUnitID != fixture.Old || mapping.CurrentUnitID != fixture.Current || mapping.SourcePath != fixture.Path || len(refs) == 0 {
		t.Fatal("fixture mapping/provenance lost")
	}
	for _, r := range refs {
		b, err := os.ReadFile(filepath.Join(root, r.Path))
		if err != nil || contentidentity.SHA256Hex(b) != r.SHA256 {
			t.Fatal("fixture reference drift", r.Path, err)
		}
	}
	if fixture.Snapshot != "" {
		path := filepath.Join(root, "data/quality-check-snapshots/zh-CN", fixture.Snapshot, "finalization.json")
		before, err := os.ReadFile(path)
		if err != nil || contentidentity.SHA256Hex(before) != fixture.FinalizationSHA {
			t.Fatal("recorded finalization identity", err)
		}
		proof, err := i18n.VerifyHistoricalQualityCheckFinalization(root, "zh-CN", fixture.Snapshot, c)
		if err != nil || len(proof.Finalization.Units) != len(c.TranslationScope) {
			t.Fatal("complete recorded historical response graph", err)
		}
		t.Logf("complete frozen historical graph PASS: units=%d; current assessment not invoked", len(proof.Finalization.Units))
	}
	t.Log("replacement→machine complete historical proof PASS; no authorization CLI invoked")
}
