package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestQualityCheckRepresentationReconciliation(t *testing.T) {
	root, old, current, frozen := replacementPageFixture(t)
	ids := append([]string{}, old.TranslationScope[:3]...)
	if err := old.UseTranslationUnitScope(ids); err != nil {
		t.Fatal(err)
	}
	if err := current.UseTranslationUnitScope(current.TranslationScope[:3]); err != nil {
		t.Fatal(err)
	}
	e, err := i18n.ExportRetranslationBatch(root, old, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := i18n.ExportTranslationUnitGenerationBundle(root, old, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, _ := i18n.ReadTransportBundle(b, 8192, 64<<20)
	outputs := map[string][]byte{}
	for _, out := range m.ExpectedOutputs {
		outputs[out.UnitID] = files["formal/inputs/"+filepath.Base(out.BundlePath)]
	}
	surfaceImport(t, root, old, b, outputs)
	if r, err := i18n.ProcessRetranslationBatch(root, old, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || r.ValidationPassed != len(ids) {
		t.Fatal(r, err)
	}
	const snapshot = "qc-representation"
	if _, _, err = i18n.CreateQualityCheckCandidateSnapshot(root, old, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = i18n.ExportQualityCheckReviewerBundle(root, old, i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	const finding = "Localize the visible message frozen by the previous representation."
	if _, err = i18n.RecordQualityCheckResults(root, old, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: ids[2:3], Rating: "B", Finding: finding}); err != nil {
		t.Fatal(err)
	}
	others := append(append([]string{}, ids[:2]...), ids[3:]...)
	if _, err = i18n.RecordQualityCheckResults(root, old, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: others, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	historical := freezeReplacementFixture(t, root, frozen)
	resolution, err := resolvePageReplacements(root, current, historical)
	if err != nil {
		t.Fatal(err)
	}
	p := resolution.Units[ids[2]]
	proof := i18n.RepresentationNecessityProof{Reason: "protected-natural-language-unrepresentable", Units: []i18n.RepresentationNecessityUnit{{Replacement: p, Rating: "B", FindingSHA256: digest([]byte(finding))}}}
	data, _ := json.Marshal(proof)
	proofPath := "data/engineering/necessity.json"
	writeFixture(t, root, proofPath, data)
	o := i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: "quality-check", Locale: "zh-CN", ReopenID: "qc-representation-successor", PreviousSnapshotID: snapshot, PreviousRating: "B", Finding: finding, UnitIDs: []string{p.CurrentUnitID}, HistoricalDocuments: historical, RepresentationProof: i18n.GlossaryArchiveReference{Path: proofPath, SHA256: digest(data)}}
	immutable := map[string][]byte{}
	for _, path := range []string{"data/quality-check-snapshots/zh-CN/" + snapshot + "/manifest.json", "data/quality-check-snapshots/zh-CN/" + snapshot + "/quality-check-results.json", "data/retranslation-runs/zh-CN/" + e.BatchID + "/manifest.json"} {
		immutable[path], err = readRegular(root, path)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"rating", "snapshot", "unit", "finding", "surface"} {
		t.Run(name, func(t *testing.T) {
			bad := o
			switch name {
			case "rating":
				bad.PreviousRating = "A"
			case "snapshot":
				bad.PreviousSnapshotID = "qc-wrong"
			case "unit":
				bad.UnitIDs = ids[:1]
			case "finding":
				bad.Finding = "Other finding"
			case "surface":
				bad.IntegratedSurface = true
			}
			if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, current, bad); err == nil {
				t.Fatal("unbound authority accepted")
			}
		})
	}
	planned, plannedPath, err := i18n.InspectQualityCheckRepresentationReopen(root, current, o)
	if err != nil {
		t.Fatal("readonly authorization", err)
	}
	if _, err := os.Lstat(filepath.Join(root, plannedPath)); !os.IsNotExist(err) {
		t.Fatal("inspection created a receipt", err)
	}
	r, _, err := i18n.RecordQualityCheckSurfaceReopen(root, current, o)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(planned, r) {
		t.Fatal("inspection/record authority mismatch")
	}
	if _, _, err := i18n.InspectQualityCheckRepresentationReopen(root, current, o); err == nil {
		t.Fatal("duplicate receipt accepted")
	}
	if r.AuthoritySource != "quality-check" || r.PreviousFinalizationPath != "" || r.SurfaceReviewID != "" || r.Units[0].PreviousRating != "B" {
		t.Fatal("mixed authority", r)
	}
	export := i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: o.UnitIDs, AllowReexport: true, PreviousSnapshotID: snapshot, SurfaceReopenID: o.ReopenID}
	revision, err := i18n.ExportRetranslationBatch(root, current, export)
	if err != nil {
		t.Fatal(err)
	}
	b, m, err = i18n.ExportTranslationUnitGenerationBundle(root, current, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: revision.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, _ = i18n.ReadTransportBundle(b, 8192, 64<<20)
	outputs = map[string][]byte{}
	for _, out := range m.ExpectedOutputs {
		outputs[out.UnitID] = bytes.Replace(files["formal/protected-candidates/"+filepath.Base(out.BundlePath)], []byte("Page"), []byte("Revised page"), 1)
	}
	surfaceImport(t, root, current, b, outputs)
	if result, err := i18n.ProcessRetranslationBatch(root, current, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: revision.BatchID}); err != nil || result.ValidationPassed != 1 {
		t.Fatal(result, err)
	}
	if _, _, err = i18n.CreateQualityCheckCandidateSnapshot(root, current, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-representation-revision"}); err != nil {
		t.Fatal(err)
	}
	scope, err := i18n.BuildQualityCheckScope(root, current, i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-representation-revision", PreviousSnapshotID: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if scope.PendingCount != 1 || scope.ACount != 2 {
		t.Fatal("replacement inherited language rating", scope)
	}
	if _, _, err = i18n.ExportQualityCheckReviewerBundle(root, current, i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: "qc-representation-revision", PreviousSnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	if _, err = i18n.RecordQualityCheckResults(root, current, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "qc-representation-revision", PreviousSnapshotID: snapshot, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: o.UnitIDs, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	for path, want := range immutable {
		got, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("old evidence mutated", path)
		}
	}
}
