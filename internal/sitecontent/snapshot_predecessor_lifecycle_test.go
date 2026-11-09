package sitecontent

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Add one genuinely sealed Surface response beside the approved-interval Unit.
// Its next current-validation revision must rebuild a predecessor transport;
// changing an approved decoder file alone would not exercise that boundary.
func snapshotApprovedPredecessorSibling(t *testing.T, root string, c *i18n.Catalog, old i18n.LegacyGenerationProvenanceOptions) (i18n.LegacyGenerationProvenanceOptions, string) {
	t.Helper()
	var f i18n.QualityCheckFinalization
	if err := json.Unmarshal(mustReadFixture(t, root, "data/quality-check-snapshots/zh-CN/"+old.PreviousSnapshotID+"/finalization.json"), &f); err != nil {
		t.Fatal(err)
	}
	// Seed policy permits both scripts; only the subsequent adjacency rule makes
	// this saved response affected. It is real imported/validated evidence.
	data := mustReadFixture(t, root, "locales/zh-CN/locale.json")
	var locale i18n.Locale
	if err := json.Unmarshal(data, &locale); err != nil {
		t.Fatal(err)
	}
	locale.LanguageAudit.Version++
	locale.LanguageAudit.Policy.ProseScripts = []string{"Latin", "Han"}
	data, _ = json.Marshal(locale)
	writeFixture(t, root, "locales/zh-CN/locale.json", data)
	fp := &f
	build := surfaceBuilder(t, root, c, &fp)
	attachIntegratedSurfaceAuthority(c, build)
	closure := surfaceDescriptorClosure(t, root, c, old.PreviousSnapshotID)
	sibling := c.TranslationScope[0]
	var accepted i18n.QualityCheckSnapshotUnit
	for _, u := range f.Units {
		if u.UnitID == sibling {
			accepted = u.Snapshot
		}
	}
	finding := "Synthetic inline descriptor replacement for predecessor transport regression."
	_, _ = surfaceReceipt(t, root, c, build, []Reference{closure}, "predecessor-sibling-finding", "failed", []SurfaceFinding{{ID: accepted.SourcePath, Finding: finding, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: f.SnapshotID, Unit: accepted}}}})
	// The existing Surface receipt/finding authorizes only this exact sibling.
	reopen := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "predecessor-sibling", PreviousSnapshotID: f.SnapshotID, SurfaceReviewID: "predecessor-sibling-finding", Finding: finding, UnitIDs: []string{sibling}, IntegratedSurface: true}
	if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, reopen); err != nil {
		t.Fatal(err)
	}
	r, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{sibling}, UnitKind: i18n.UnitKindPage, PreviousSnapshotID: f.SnapshotID, SurfaceReopenID: reopen.ReopenID, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	p, err := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", r.BatchID, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	request := p.Authority.Request
	binding := p.Authority.Bindings[0]
	q := i18n.SurfaceTransportUnitRequest{UnitID: sibling, FindingSHA256: binding.FindingSHA256}
	for _, region := range binding.Projection.Available {
		if region.Kind == "inline-api-descriptor" {
			q.Regions = append(q.Regions, region.ID)
			break
		}
	}
	if len(q.Regions) != 1 {
		t.Fatal("source-owned descriptor region missing")
	}
	request.Units = []i18n.SurfaceTransportUnitRequest{q}
	transport, err := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", r.BatchID, &request, true)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: r.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	raw := bridgeEOF([]byte(strings.Replace(transport.Authority.Bindings[0].Projection.Template, "function", "函数", 1)))
	surfaceImport(t, root, c, bundle, map[string][]byte{sibling: raw})
	if result, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: r.BatchID}); err != nil || result.ValidationPassed != 1 {
		t.Fatal("sealed sibling process", result, err)
	}
	next := "approved-with-sealed-sibling"
	surfaceFinalize(t, root, c, next, old.PreviousSnapshotID, []string{sibling})
	old.PreviousSnapshotID = next
	old.SnapshotSHA256 = digest(mustReadFixture(t, root, "data/quality-check-snapshots/zh-CN/"+next+"/manifest.json"))
	return old, sibling
}

func assertSnapshotPredecessorDrift(t *testing.T, root string, c *i18n.Catalog, batch string, files map[string][]byte) {
	t.Helper()
	var m i18n.RetranslationBatchManifest
	if err := json.Unmarshal(mustReadFixture(t, root, "data/retranslation-runs/zh-CN/"+batch+"/manifest.json"), &m); err != nil {
		t.Fatal(err)
	}
	fresh, err := i18n.InspectCurrentValidationPredecessors(root, c, "zh-CN", m.Units)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, p := range fresh {
		if p.RecoveryClass != "DETERMINISTIC_REBASE_PASS" {
			continue
		}
		var frozen i18n.CurrentValidationPredecessor
		var path string
		for _, record := range m.Units {
			if record.UnitID == p.UnitID {
				path = "formal/current-validation-predecessor-transport/" + filepath.Base(record.InputPath) + ".json"
			}
		}
		if err := json.Unmarshal(files[path], &frozen); err != nil {
			t.Fatal("historical typed predecessor missing", err)
		}
		if p.ImplementationIdentity == frozen.ImplementationIdentity || bytes.Equal(mustEncodeFixture(t, p), files[path]) {
			t.Fatal("current predecessor was not freshly rebuilt")
		}
		currentOld := p.ImplementationInputs
		found := false
		for i, ref := range currentOld {
			if ref.Path == "internal/i18n/generation_revision_inputs.go" {
				found = ref.SHA256 != frozen.ImplementationInputs[i].SHA256 && ref.SHA256 == digest(mustReadFixture(t, root, ref.Path))
			}
		}
		if !found {
			t.Fatal("fixture lacks actual predecessor producer SHA drift")
		}
		seen++
	}
	if seen != 1 {
		t.Fatal("expected one freshly reconstructed predecessor", seen)
	}
}

func mustEncodeFixture(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
