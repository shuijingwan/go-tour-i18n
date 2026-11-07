package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
)

// All publishers here operate in t.TempDir, with synthetic model/Reviewer data.
func TestApprovedIntervalBundleLifecycle(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, c, historical, archive := legacyProvenanceFixture(t, func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("Read "), []byte("请阅读")) })
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: historical.PreviousSnapshotID, UnitIDs: c.TranslationScope, Rating: "A", ReviewerSession: "synthetic-current-reviewer"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: historical.PreviousSnapshotID}); err != nil {
		t.Fatal(err)
	}
	data, _ := readRegular(root, "locales/zh-CN/locale.json")
	var locale i18n.Locale
	json.Unmarshal(data, &locale)
	locale.LanguageAudit.Version++
	locale.LanguageAudit.Policy.ProseScripts = []string{"Latin", "Han"}
	locale.LanguageAudit.Policy.Spacing = []languageaudit.SpacingRule{{ProseScript: "Han", TechnicalScript: "Latin", Kinds: []languageaudit.Kind{languageaudit.Code}}}
	data, _ = json.Marshal(locale)
	writeFixture(t, root, "locales/zh-CN/locale.json", data)
	a, err := i18n.AssessCurrentTranslation(root, c, "zh-CN", historical.PreviousSnapshotID)
	if err != nil || a.Affected != 1 {
		t.Fatal("synthetic typed finding", a, err)
	}
	data, _ = json.Marshal(a)
	writeFixture(t, root, "engineering/approved-finding.json", data)
	o := i18n.CurrentValidationReopenOptions{Locale: "zh-CN", ContentPackage: historical.ContentPackage, ReopenID: "approved-interval-fixture", PreviousSnapshotID: historical.PreviousSnapshotID, SnapshotSHA256: historical.SnapshotSHA256, Evidence: i18n.GlossaryArchiveReference{Path: "engineering/approved-finding.json", SHA256: digest(data)}, AssessmentIdentity: a.Identity}
	if _, _, err = i18n.RecordQualityCheckSurfaceReopen(root, c, i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: i18n.CurrentValidationAuthority, Locale: o.Locale, ReopenID: o.ReopenID, PreviousSnapshotID: o.PreviousSnapshotID, CurrentValidation: &o}); err != nil {
		t.Fatal(err)
	}
	r, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: o.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{historical.CurrentUnitID}, UnitKind: i18n.UnitKindPage, PreviousSnapshotID: o.PreviousSnapshotID, SurfaceReopenID: o.ReopenID, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := i18n.PrepareLegacyGenerationProvenance(root, c, historical, false, "")
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := i18n.PrepareLegacyGenerationProvenance(root, c, historical, true, plan.Identity)
	if err != nil {
		t.Fatal(err)
	}
	ref := &i18n.GlossaryArchiveReference{Path: recorded.ReceiptPath, SHA256: recorded.ReceiptSHA256}
	selection := i18n.ApprovedIntervalSelection{Schema: i18n.ApprovedIntervalSelectionSchema, Locale: o.Locale, BatchID: r.BatchID, ContentPackage: o.ContentPackage, Units: []i18n.ApprovedIntervalSelectionUnit{{CurrentUnitID: historical.CurrentUnitID, Mode: i18n.ApprovedIntervalMode, LegacyProvenance: ref}}}
	data, _ = json.Marshal(selection)
	writeFixture(t, root, "engineering/selection.json", data)
	options := i18n.GenerationBundleOptions{Locale: o.Locale, BatchID: r.BatchID, ApprovedIntervalSelection: "engineering/selection.json"}
	bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, options)
	if err != nil {
		t.Fatal("approved export", err)
	}
	files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var proof i18n.ApprovedIntervalTransport
	if err = json.Unmarshal(files["formal/approved-interval-transport/"+historical.CurrentUnitID+".json"], &proof); err != nil {
		t.Fatal(err)
	}
	if proof.ProvenanceSource != "legacy-recorded" || proof.Projection.Namespace != "GTI18N_AI1/v1" || len(proof.Projection.Findings) == 0 {
		t.Fatal("proof missing", proof.ProvenanceSource)
	}
	var codec approvedPageCodec
	if err = StrictJSON(proof.Projection.Codec, &codec); err != nil {
		t.Fatal(err)
	}
	restored, err := codec.Restore(proof.Projection.Text)
	name := filepath.Base(bm.ExpectedOutputs[0].BundlePath)
	if err != nil || !bytes.Equal(bridgeEOF(restored), files["formal/candidates/"+name]) {
		t.Fatal("inverse restore differs from reviewed candidate", err)
	}
	for _, output := range []string{
		strings.Replace(proof.Projection.Text, proof.Projection.ParentToken, "changed parent", 1),
		proof.Projection.Text + proof.Projection.ParentToken,
		proof.Projection.Text + "⟦P0⟧",
	} {
		if _, err := codec.Restore(output); err == nil {
			t.Fatal("opaque/token mutation accepted")
		}
	}
	for _, change := range []func(map[string][]byte){
		func(f map[string][]byte) {
			f["formal/approved-interval-selection.json"] = []byte(`{"schema":"unknown"}`)
		},
		func(f map[string][]byte) {
			f["formal/approved-interval-transport/"+historical.CurrentUnitID+".json"] = []byte(`{}`)
		},
	} {
		tampered := rewriteSurfaceGenerationBundle(t, bundle, change)
		if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, tampered); err == nil {
			t.Fatal("rehashed tampered request/proof accepted")
		}
	}
	for _, f := range proof.Projection.Findings {
		if f.Candidate.Start < proof.Projection.ParentCandidate.End && proof.Projection.ParentCandidate.Start < f.Candidate.End {
			t.Fatal("finding overlaps opaque parent")
		}
	}
	if err = os.Remove(filepath.Join(root, "engineering/selection.json")); err != nil {
		t.Fatal(err)
	}
	if _, err = i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err != nil {
		t.Fatal("frozen selection reconstruction", err)
	}
	// Archive reappearance must change the graph to the stronger source.
	writeFixture(t, root, "data/generation-archives/zh-CN/"+historical.OriginalBundleSHA256+".zip", archive)
	if _, err = i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err == nil {
		t.Fatal("legacy Bundle survived archive reappearance")
	}
	writeFixture(t, root, "engineering/selection.json", data)
	normal, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, options)
	if err != nil {
		t.Fatal("normal-priority export", err)
	}
	nf, _ := i18n.ReadTransportBundle(normal, 8192, 64<<20)
	var np i18n.ApprovedIntervalTransport
	json.Unmarshal(nf["formal/approved-interval-transport/"+historical.CurrentUnitID+".json"], &np)
	if np.ProvenanceSource != "normal-archive" {
		t.Fatal("legacy forced over normal archive")
	}
	if err = os.Remove(filepath.Join(root, "data/generation-archives/zh-CN/"+historical.OriginalBundleSHA256+".zip")); err != nil {
		t.Fatal(err)
	}
	// Exact baseline restores the original candidate and still FAILS active audit.
	baseline := files["formal/protected-candidates/"+name]
	surfaceImport(t, root, c, bundle, map[string][]byte{historical.CurrentUnitID: baseline})
	ib, err := readRegular(root, "data/retranslation-runs/zh-CN/"+r.BatchID+"/generation-imports/initial-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var imported i18n.GenerationResultBundleManifest
	if json.Unmarshal(ib, &imported) != nil || imported.GenerationBundleSHA256 != digest(bundle) || imported.GenerationBundleSHA256 == historical.OriginalBundleSHA256 || imported.Provider != "chatgpt" || imported.Model != i18n.FormalChatGPTGenerationModel {
		t.Fatal("new model invocation provenance lost")
	}
	startup := *c
	startup.TranslationScope = nil
	p, err := i18n.ProcessRetranslationBatch(root, &startup, i18n.RetranslationProcessOptions{Locale: o.Locale, BatchID: r.BatchID})
	if err != nil || p.RestorePassed != 1 || p.ValidationFailed != 1 {
		t.Fatal("active canonical validation bypassed", p, err)
	}
	retryOptions := i18n.GenerationBundleOptions{Locale: o.Locale, BatchID: r.BatchID, UnitID: historical.CurrentUnitID}
	writeFixture(t, root, "engineering/other-selection.json", append(bytes.Clone(data), '\n'))
	changedOptions := retryOptions
	changedOptions.ApprovedIntervalSelection = "engineering/other-selection.json"
	if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, changedOptions); err == nil {
		t.Fatal("retry replaced frozen selection bytes")
	}
	retry, rm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, retryOptions)
	if err != nil {
		t.Fatal("retry lost proof", err)
	}
	rf, _ := i18n.ReadTransportBundle(retry, 8192, 64<<20)
	if !bytes.Equal(rf["formal/approved-interval-selection.json"], files["formal/approved-interval-selection.json"]) || !bytes.Equal(rf["formal/approved-interval-transport/"+historical.CurrentUnitID+".json"], files["formal/approved-interval-transport/"+historical.CurrentUnitID+".json"]) {
		t.Fatal("retry changed selection/proof")
	}
	repaired := []byte(strings.ReplaceAll(string(baseline), "请阅读", "请阅读 "))
	if bytes.Equal(repaired, baseline) {
		t.Fatal("fixture exposed finding body missing")
	}
	surfaceImport(t, root, c, retry, map[string][]byte{historical.CurrentUnitID: repaired})
	p, err = i18n.ProcessRetranslationRetry(root, &startup, i18n.RetranslationRetryOptions{Locale: o.Locale, BatchID: r.BatchID, UnitID: historical.CurrentUnitID})
	if err != nil || p.ValidationPassed != 1 {
		t.Fatal("retry decoder", p, err)
	}
	if rm.Attempt != 2 {
		t.Fatal("retry attempt identity")
	}
	snapshot, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "approved-after-retry"})
	if err != nil {
		t.Fatal("Snapshot lineage", err)
	}
	found := false
	for _, unit := range snapshot.Units {
		if unit.UnitID == historical.CurrentUnitID {
			found = unit.SelectedBatchID == r.BatchID && unit.Attempt == 2
		}
	}
	if !found {
		t.Fatal("Snapshot lost the new Bundle's selected retry provenance")
	}
	t.Run("installed-proof-stale-recovery-boundary", func(t *testing.T) {
		checkInstalledApprovedProofStaleRecoveryBoundary(t, root, c, o, r.BatchID, historical.CurrentUnitID, options)
	})
}
