package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestMultiUnitApprovedIntervalBridgeHistoricalReconstruction(t *testing.T) {
	// Reuse the existing passed approved-interval + current-validation predecessor
	// fixture. All artifacts, model outputs and Reviewer decisions are synthetic
	// TempDir data; no real locale lifecycle is read or executed.
	checkApprovedIntervalBundleLifecycle(t, false, func(t *testing.T, root string, c *i18n.Catalog, reopen i18n.CurrentValidationReopenOptions, batch, approvedID string, _ i18n.GenerationBundleOptions) {
		for _, path := range []string{"internal/i18n/approved_interval_same_unit_bridge.go", "internal/sitecontent/approved_interval_same_unit_bridge.go"} {
			writeFixture(t, root, path, mustReadFixture(t, repositoryRoot(t), path))
		}
		var predecessor i18n.RetranslationBatchManifest
		if err := json.Unmarshal(mustReadFixture(t, root, "data/retranslation-runs/zh-CN/"+batch+"/manifest.json"), &predecessor); err != nil {
			t.Fatal(err)
		}
		if len(predecessor.Units) != 2 {
			t.Fatal("fixture requires two distinct restricted predecessors")
		}
		a, b := predecessor.Units[0], predecessor.Units[1]
		if a.InputPath == b.InputPath || a.InputSHA256 == b.InputSHA256 || a.SourcePath == b.SourcePath || a.SourceSHA256 == b.SourceSHA256 {
			t.Fatal("fixture does not distinguish per-Unit formal identities")
		}
		ids := []string{a.UnitID, b.UnitID}
		const finding = "Improve ordinary prose outside the immutable protected regions."
		if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: reopen.Locale, SnapshotID: "approved-after-retry", PreviousSnapshotID: reopen.PreviousSnapshotID, ReviewerSession: "independent-multi-bridge-reviewer", UnitIDs: ids, Rating: "B", Finding: finding}); err != nil {
			t.Fatal(err)
		}
		const evidence = "engineering/multi-bridge-reviewer.md"
		writeFixture(t, root, evidence, []byte("Independent synthetic Reviewer: B. "+finding+"\n"))
		metadata, _ := json.Marshal(c.TranslationPackages[0])
		for _, path := range []string{SourceManifestPath, GlobalPath, PageRegistryPath} {
			if _, err := os.Stat(filepath.Join(root, path)); os.IsNotExist(err) {
				writeFixture(t, root, path, metadata)
			} else if err != nil {
				t.Fatal(err)
			}
		}
		c.DerivedTransportCodecReader = derivedApprovedCodec
		opts := i18n.SameUnitBridgeOptions{Locale: reopen.Locale, ContentPackage: reopen.ContentPackage, BridgeID: "multi-unit-ordinary-qc", PreviousSnapshotID: "approved-after-retry", SnapshotSHA256: digest(mustReadFixture(t, root, "data/quality-check-snapshots/zh-CN/approved-after-retry/manifest.json")), ReviewerEvidence: i18n.GlossaryArchiveReference{Path: evidence, SHA256: digest(mustReadFixture(t, root, evidence))}, UnitIDs: ids, Record: true}
		sealed, err := i18n.PrepareSameUnitRepresentationBridge(root, c, opts)
		if err != nil || !sealed.Recorded || len(sealed.Receipt.SameUnitBridge.Bindings) != 2 {
			t.Fatal("synthetic multi-Unit bridge", err)
		}
		historicalCalls := 0
		c.BridgeResolver = func(_ string, _ *i18n.Catalog, mode string, _ i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
			result, err := build()
			if err != nil || mode != "historical" {
				return result, err
			}
			historicalCalls++
			for _, binding := range result.Receipt.SameUnitBridge.Bindings {
				decoder := result.Adapters[binding.UnitID]
				if decoder == nil {
					t.Fatal("historical reconstruction omitted sibling", binding.UnitID)
				}
				candidate, err := decoder.Restore(binding.Template)
				for _, q := range result.Receipt.Units {
					if q.UnitID == binding.UnitID && (err != nil || !bytes.Equal(bridgeEOF(candidate), mustReadFixture(t, root, q.Snapshot.CandidatePath))) {
						t.Fatal("historical sibling template did not restore its own candidate", binding.UnitID, err)
					}
				}
			}
			return result, nil
		}
		next, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: opts.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: ids, AllowReexport: true, PreviousSnapshotID: opts.PreviousSnapshotID, RepresentationBridgeID: opts.BridgeID})
		if err != nil {
			t.Fatal("synthetic ordinary QC revision", err)
		}
		export := i18n.GenerationBundleOptions{Locale: opts.Locale, BatchID: next.BatchID}
		bundle, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, export)
		if err != nil {
			t.Fatal("multi-Unit historical reconstruction", err)
		}
		if historicalCalls < 2 {
			t.Fatal("both callers must reconstruct the complete bridge", historicalCalls)
		}
		files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
		if err != nil {
			t.Fatal(err)
		}
		approved := false
		for _, binding := range sealed.Receipt.SameUnitBridge.Bindings {
			if !bytes.Equal(files["formal/protected-candidates/"+filepath.Base(binding.ProtectedInput.Path)], []byte(binding.Template)) {
				t.Fatal("wrong per-Unit frozen template", binding.UnitID)
			}
			if binding.UnitID == approvedID {
				approved = strings.HasPrefix(binding.TransportMember, "formal/approved-interval-transport/")
			}
			for _, tc := range []struct{ name, path string }{{"manifest", binding.OriginalManifest.Path}, {"input", binding.ProtectedInput.Path}} {
				t.Run(binding.UnitID+"/tampered-"+tc.name, func(t *testing.T) {
					old := mustReadFixture(t, root, tc.path)
					defer writeFixture(t, root, tc.path, old)
					writeFixture(t, root, tc.path, append(bytes.Clone(old), ' '))
					if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, export); err == nil {
						t.Fatal("tampered per-Unit frozen evidence accepted")
					}
				})
			}
		}
		if !approved {
			t.Fatal("fixture must exercise the approved template/ancestor invariant")
		}
		t.Run("retry-currentness-reuse", func(t *testing.T) {
			testRegisteredBridgeRetryReuse(t, root, c, next, sealed.Receipt.SameUnitBridge.Options.ReviewerEvidence.Path)
		})
	})
}
