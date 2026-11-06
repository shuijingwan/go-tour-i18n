package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestIntegratedSurfaceAggregateReopenAuthority(t *testing.T) {
	root, c, ids := surfaceFixture(t, "learn-docs", 7)
	f := surfaceStart(t, root, c, ids)
	build := surfaceBuilder(t, root, c, &f)
	attachIntegratedSurfaceAuthority(c, build)
	closure := surfaceDescriptorClosure(t, root, c, f.SnapshotID)
	findings := []SurfaceFinding{}
	texts := []string{}
	for i := 0; i < 6; i++ {
		a := i18n.SurfaceCandidateArtifact{SnapshotID: f.SnapshotID, Unit: f.Units[i].Snapshot}
		text := fmt.Sprintf("Correct exact reviewed defect %d.", i+1)
		texts = append(texts, text)
		findings = append(findings, SurfaceFinding{ID: a.Unit.SourcePath, Finding: text, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{a}})
	}
	r, ref := surfaceReceipt(t, root, c, build, []Reference{closure}, "aggregate-failed", "failed", findings)
	o := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "aggregate-reopen", PreviousSnapshotID: f.SnapshotID, SurfaceReviewID: r.ReviewID, Findings: texts, UnitIDs: ids[:6], IntegratedSurface: true}
	reject := func(t *testing.T, bad i18n.QualityCheckSurfaceReopenOptions) {
		t.Helper()
		bad.ReopenID = "rejected-aggregate"
		if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, bad); err == nil {
			t.Fatal("invalid aggregate accepted")
		}
		if _, err := os.Stat(filepath.Join(root, "data/quality-check-reopens/zh-CN", bad.ReopenID+".json")); !os.IsNotExist(err) {
			t.Fatal("rejected aggregate wrote authorization", err)
		}
	}
	for _, test := range []struct {
		name   string
		change func(*i18n.QualityCheckSurfaceReopenOptions)
	}{
		{"missing-unit", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.UnitIDs = ids[:5] }},
		{"extra-unit", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.UnitIDs = ids }},
		{"missing-finding", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.Findings = texts[:5] }},
		{"unknown-finding", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.Findings[5] = "Not in receipt." }},
		{"duplicate-finding", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.Findings[5] = texts[0] }},
		{"empty-finding", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.Findings[5] = "" }},
		{"untrimmed-finding", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.Findings[5] += " " }},
		{"zero-findings", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.Findings = nil }},
		{"duplicate-unit", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.UnitIDs[5] = ids[0] }},
		{"wrong-unit-for-artifact", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.UnitIDs[5] = ids[6] }},
		{"non-integrated", func(o *i18n.QualityCheckSurfaceReopenOptions) { o.IntegratedSurface = false }},
	} {
		t.Run(test.name, func(t *testing.T) {
			bad := o
			bad.Findings = append([]string(nil), o.Findings...)
			bad.UnitIDs = append([]string(nil), o.UnitIDs...)
			test.change(&bad)
			reject(t, bad)
		})
	}
	for _, class := range []string{"navigation", "structured", "runtime"} {
		t.Run(class, func(t *testing.T) {
			fs := append([]SurfaceFinding(nil), findings...)
			fs[5].Class, fs[5].Artifacts = class, nil
			nonTU, _ := surfaceReceipt(t, root, c, build, []Reference{closure}, "aggregate-"+class, "failed", fs)
			bad := o
			bad.SurfaceReviewID = nonTU.ReviewID
			reject(t, bad)
		})
	}
	for name, path := range map[string]string{
		"receipt": ref.Path, "evidence": r.Evidence.Path, "closure": closure.Path,
		"runtime": "production/identity.json", "finalization": "data/quality-check-snapshots/zh-CN/" + f.SnapshotID + "/finalization.json",
		"candidate": f.Units[5].Snapshot.CandidatePath, "validation": f.Units[5].Snapshot.ValidationPath,
	} {
		t.Run("stale-"+name, func(t *testing.T) {
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, path, append(bytes.Clone(before), []byte("changed")...))
			defer writeFixture(t, root, path, before)
			reject(t, o)
		})
	}
	t.Run("stale-current-scope", func(t *testing.T) {
		before := c.TranslationPackages[0].Units[0].Route
		c.TranslationPackages[0].Units[0].Route = "/changed/"
		defer func() { c.TranslationPackages[0].Units[0].Route = before }()
		reject(t, o)
	})
	// Flag order is irrelevant: independently reverse IDs and rotate findings.
	o.UnitIDs = []string{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
	o.Findings = append(append([]string(nil), texts[2:]...), texts[:2]...)
	reopen, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	if reopen.UnitCount != 6 || reopen.SchemaVersion != 1 || reopen.Finding != "" || len(reopen.Findings) != 6 {
		t.Fatal(reopen)
	}
	for i, u := range reopen.Units {
		if u.Index != i+1 || u.UnitID != ids[i] || u.Finding != texts[i] {
			t.Fatal("stable per-unit finding", u)
		}
	}
	exportOptions := i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: ids[:6], AllowReexport: true, PreviousSnapshotID: f.SnapshotID, SurfaceReopenID: o.ReopenID}
	for _, wrong := range [][]string{ids[:5], ids} {
		bad := exportOptions
		bad.UnitIDs = wrong
		if _, err := i18n.ExportRetranslationBatch(root, c, bad); err == nil {
			t.Fatal("export accepted non-exact scope")
		}
	}
	e, err := i18n.ExportRetranslationBatch(root, c, exportOptions)
	if err != nil || e.UnitCount != 6 {
		t.Fatal(e, err)
	}
	b, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil || len(m.ExpectedOutputs) != 6 {
		t.Fatal(m, err)
	}
	files, err := i18n.ReadTransportBundle(b, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var manifest i18n.RetranslationBatchManifest
	if err := StrictJSON(files["formal/manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Units) != 6 {
		t.Fatal(manifest)
	}
	for i, u := range manifest.Units {
		if u.UnitID != ids[i] || u.PreviousFinding != texts[i] || u.RevisionFeedbackSource != "surface_review" || u.RevisionAuthorizationID != o.ReopenID || u.PreviousRating != "A" || u.PreviousReviewDecision != "reopened" || u.PreviousReviewPath != ref.Path || u.PreviousReviewSHA256 != ref.SHA256 {
			t.Fatal("exact manifest finding/provenance", u)
		}
	}
	if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, b); err != nil {
		t.Fatal(err)
	}
}
