package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Reuse the complete imported same-unit bridge fixture, including an ordinary
// QC successor, then change only the new revision's feedback authority.
func testSurfaceHistoricalProtectedPredecessor(t *testing.T, parent string, catalog *i18n.Catalog, f *i18n.QualityCheckFinalization, id string, wantLineage bool) {
	t.Helper()
	root := cloneBridgeSuccessorFixture(t, parent)
	c := *catalog
	AttachPageReplacementInvocation(root, &c)
	if _, _, err := i18n.EstablishCurrentTranslationAuthorization(root, &c, "zh-CN", f.SnapshotID); err != nil {
		t.Fatal("current predecessor authorization", err)
	}
	build := surfaceBuilder(t, root, &c, &f)
	attachIntegratedSurfaceAuthority(&c, build)
	closure := surfaceDescriptorClosure(t, root, &c, f.SnapshotID)
	u := f.Units[0].Snapshot
	if u.UnitID != id {
		t.Fatal("fixture selected a different predecessor")
	}
	const finding = "Correct this exact ordinary prose defect; preserve protected identities."
	r, _ := surfaceReceipt(t, root, &c, build, []Reference{closure}, "protected-predecessor-surface", "failed", []SurfaceFinding{{ID: u.SourcePath, Finding: finding, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: f.SnapshotID, Unit: u}}}})
	o := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "protected-predecessor-reopen", PreviousSnapshotID: f.SnapshotID, SurfaceReviewID: r.ReviewID, Finding: finding, UnitIDs: []string{id}, IntegratedSurface: true}
	if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, &c, o); err != nil {
		t.Fatal(err)
	}
	exp, err := i18n.ExportRetranslationBatch(root, &c, i18n.RetranslationExportOptions{Locale: o.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: o.UnitIDs, AllowReexport: true, PreviousSnapshotID: f.SnapshotID, SurfaceReopenID: o.ReopenID, UnitKind: i18n.UnitKindPage})
	if err != nil {
		t.Fatal(err)
	}
	options := i18n.GenerationBundleOptions{Locale: o.Locale, BatchID: exp.BatchID}
	discovery, err := i18n.PrepareSurfaceRevisionTransport(root, &c, o.Locale, exp.BatchID, nil, false)
	if err != nil {
		t.Fatal("Surface capability discovery", err)
	}
	request := surfaceCapabilityRequest(t, discovery, false)
	capability, err := i18n.PrepareSurfaceRevisionTransport(root, &c, o.Locale, exp.BatchID, &request, true)
	if err != nil {
		t.Fatal("Surface capability fixture authority", err)
	}
	bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, &c, options)
	if err != nil {
		t.Fatal("Surface predecessor Bundle", err)
	}
	checked, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, &c, bundle)
	if err != nil || !reflect.DeepEqual(bm, checked) {
		t.Fatal("independent canonical current-check", err)
	}
	files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(bm.ExpectedOutputs[0].BundlePath)
	baseline := files["formal/protected-candidates/"+name]
	candidate, err := readRegular(root, u.CandidatePath)
	if err != nil || !bytes.Equal(files["formal/candidates/"+name], candidate) {
		t.Fatal("reviewed Snapshot candidate changed", err)
	}
	var v i18n.RetranslationValidation
	validation, err := readRegular(root, u.ValidationPath)
	if err != nil || json.Unmarshal(validation, &v) != nil {
		t.Fatal("validation", err)
	}
	rawPath := "data/retranslation-runs/zh-CN/" + u.SelectedBatchID + "/" + v.RawResponsePath
	raw, err := readRegular(root, rawPath)
	if err != nil || v.Attempt != u.Attempt || !bytes.Equal(files["formal/historical-protected-candidates/"+name], raw) {
		t.Fatal("editable baseline is not exact passed raw/attempt", err)
	}
	projection := capability.Authority.Bindings[0].Projection
	factory := c.Adapted[0].Adapter.(i18n.SurfaceTransportAdapter)
	decoder, err := factory.ReadSurfaceTransportProjection(projection.Codec)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decoder.Restore(string(baseline))
	if err != nil || !bytes.Equal(bridgeEOF(decoded), candidate) {
		t.Fatal("capable baseline lost inherited exact restore", err)
	}
	unit, err := c.Unit(id)
	if err != nil {
		t.Fatal(err)
	}
	restored, restoreErr := unit.Adapter.Restore(string(raw))
	currentRestores := restoreErr == nil && bytes.Equal(append(bytes.TrimRight(restored, "\n"), '\n'), candidate)
	if currentRestores == wantLineage {
		t.Fatal("fixture lost historical/current decoder distinction")
	}
	lineagePath := "formal/representation-lineage/" + id + ".json"
	var lineage struct {
		Bridge       i18n.GlossaryArchiveReference   `json:"bridge"`
		Dependencies []i18n.GlossaryArchiveReference `json:"dependencies"`
	}
	if wantLineage {
		if err := json.Unmarshal(files[lineagePath], &lineage); err != nil || len(lineage.Dependencies) == 0 {
			t.Fatal("missing immutable representation lineage", err)
		}
	} else if files[lineagePath] != nil {
		t.Fatal("ordinary Surface Unit inherited an unrelated bridge")
	}
	for _, path := range []string{"formal/candidates/" + name, "formal/protected-candidates/" + name, lineagePath} {
		if path == lineagePath && !wantLineage {
			continue
		}
		found := false
		for _, entry := range bm.Files {
			if entry.BundlePath == path {
				found = entry.SHA256 == digest(files[path])
			}
		}
		if !found {
			t.Fatal("formal input missing inventory/hash", path)
		}
	}
	if !wantLineage {
		return
	}
	// Immutable lineage, raw, candidate and exact validation/attempt remain
	// mandatory for both export and independent current-check.
	for _, name := range []string{"missing-lineage", "tampered-lineage", "raw", "candidate", "validation-attempt", "wrong-snapshot", "unrelated-unit"} {
		t.Run(name, func(t *testing.T) {
			path := lineage.Bridge.Path
			switch name {
			case "raw":
				path = rawPath
			case "candidate":
				path = u.CandidatePath
			case "validation-attempt":
				path = u.ValidationPath
			case "wrong-snapshot", "unrelated-unit":
				path = filepath.ToSlash(filepath.Join(exp.BatchPath, "manifest.json"))
			}
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, before)
			bad := append(bytes.Clone(before), ' ')
			switch name {
			case "missing-lineage":
				if err := os.Remove(filepath.Join(root, path)); err != nil {
					t.Fatal(err)
				}
			case "validation-attempt":
				changed := v
				changed.Attempt++
				bad, _ = json.Marshal(changed)
			case "wrong-snapshot", "unrelated-unit":
				var m i18n.RetranslationBatchManifest
				if err := json.Unmarshal(before, &m); err != nil {
					t.Fatal(err)
				}
				if name == "wrong-snapshot" {
					m.Units[0].PreviousSnapshotID = "comment-predecessor"
				} else {
					m.Units[0].UnitID = strings.Repeat("a", 64)
				}
				bad, _ = json.Marshal(m)
			}
			if name != "missing-lineage" {
				writeFixture(t, root, path, bad)
			}
			if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, &c, options); err == nil {
				t.Fatal("invalid historical binding exported")
			}
			if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, &c, bundle); err == nil {
				t.Fatal("invalid historical binding remained current")
			}
		})
	}
}

func TestSurfaceOrdinaryProtectedPredecessorBundle(t *testing.T) {
	root, c, ids := surfaceFixture(t, "learn-docs", 1)
	f := surfaceStart(t, root, c, ids)
	testSurfaceHistoricalProtectedPredecessor(t, root, c, f, ids[0], false)
}
