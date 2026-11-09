package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestSnapshotReconciledHistoricalInvocation(t *testing.T) {
	checkApprovedIntervalBundleLifecycle(t, true)
}

func checkSnapshotReconciledHistoricalInvocation(t *testing.T, root string, c *i18n.Catalog, o i18n.CurrentValidationReopenOptions, batch string, historical i18n.LegacyGenerationProvenanceOptions, bundle []byte) {
	t.Helper()
	if historical.UnitID == historical.CurrentUnitID {
		t.Fatal("fixture lacks old/current reconciliation")
	}
	prefix := "data/retranslation-runs/" + o.Locale + "/"
	var oldManifest, currentManifest i18n.RetranslationBatchManifest
	if err := json.Unmarshal(mustReadFixture(t, root, prefix+historical.BatchID+"/manifest.json"), &oldManifest); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range oldManifest.Units {
		if r.UnitID == historical.CurrentUnitID {
			t.Fatal("current Unit was fabricated in historical invocation")
		}
		if r.UnitID == historical.UnitID {
			found = true
		}
	}
	if !found {
		t.Fatal("old invocation identity missing")
	}
	archivePath := "data/generation-archives/" + o.Locale + "/" + historical.OriginalBundleSHA256 + ".zip"
	if _, err := os.Lstat(filepath.Join(root, archivePath)); !os.IsNotExist(err) {
		t.Fatal("fixture fabricated missing old ZIP", err)
	}
	var previous i18n.QualityCheckSnapshotManifest
	previousBytes := mustReadFixture(t, root, "data/quality-check-snapshots/"+o.Locale+"/"+o.PreviousSnapshotID+"/manifest.json")
	if err := json.Unmarshal(previousBytes, &previous); err != nil {
		t.Fatal(err)
	}
	var reconciliationPath string
	for _, u := range previous.Units {
		if u.UnitID == historical.CurrentUnitID {
			if u.SelectedBatchID != historical.BatchID || u.RepresentationReconciliation == nil {
				t.Fatal("previous Snapshot lost explicit historical reconciliation")
			}
			reconciliationPath = u.RepresentationReconciliation.Path
		}
	}
	if reconciliationPath == "" {
		t.Fatal("reconciliation receipt missing")
	}
	receiptBytes := mustReadFixture(t, root, reconciliationPath)
	if err := json.Unmarshal(mustReadFixture(t, root, prefix+batch+"/manifest.json"), &currentManifest); err != nil {
		t.Fatal(err)
	}
	var receipt i18n.RepresentationReconciliation
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		t.Fatal(err)
	}
	mapped := false
	for _, item := range receipt.Units {
		if item.Replacement.CurrentUnitID == historical.CurrentUnitID {
			if item.Replacement.OldUnitID != historical.UnitID || item.Predecessor.SelectedBatchID != historical.BatchID {
				t.Fatal("receipt does not bind exact old invocation")
			}
			for _, r := range oldManifest.Units {
				if r.UnitID == historical.UnitID && r.InputSHA256 != item.Replacement.OldInputSHA256 {
					t.Fatal("old formal input binding changed")
				}
			}
			for _, r := range currentManifest.Units {
				if r.UnitID == historical.CurrentUnitID {
					mapped = r.InputSHA256 == item.Replacement.CurrentInputSHA256 && r.RevisionFeedbackSource == i18n.CurrentValidationRevisionFeedback
				}
			}
		}
	}
	if !mapped {
		t.Fatal("current successor is not bound by the existing receipt")
	}
	var result i18n.RetranslationProcessResult
	if err := json.Unmarshal(mustReadFixture(t, root, prefix+batch+"/result.json"), &result); err != nil {
		t.Fatal(err)
	}
	if result.ValidationPassed != len(currentManifest.Units) || result.RestorePassed != len(currentManifest.Units) || len(result.Units) != 3 {
		t.Fatal("historical batch not complete PASS")
	}
	for _, r := range result.Units {
		var v i18n.RetranslationValidation
		if err := json.Unmarshal(mustReadFixture(t, root, prefix+batch+"/"+r.ValidationPath), &v); err != nil {
			t.Fatal(err)
		}
		if v.CandidateSHA256 != "" || v.Status != "passed" || r.Status != "passed" {
			t.Fatal("fixture must exercise historical PASS for every Unit")
		}
	}
	files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	proofPath := "formal/approved-interval-transport/" + historical.CurrentUnitID + ".json"
	proof := files[proofPath]
	if len(proof) == 0 || len(files["formal/approved-interval-selection.json"]) == 0 {
		t.Fatal("installed approved invocation missing")
	}
	path := "internal/i18n/generation_revision_inputs.go"
	code := mustReadFixture(t, root, path)
	writeFixture(t, root, path, append(bytes.Clone(code), []byte("\n// Synthetic current decoder producer drift.\n")...))
	defer writeFixture(t, root, path, code)
	var strictStack string
	observeCaller := func() {
		stack := make([]byte, 32768)
		stack = stack[:runtime.Stack(stack, false)]
		if strings.Contains(string(stack), "validateUnifiedPromotionEvidence") && strings.Contains(string(stack), "prepareInvocationRetranslationOutput") {
			strictStack = string(stack)
		}
	}
	probe := *c
	probe.ReplacementResolver = func(root string, catalog *i18n.Catalog, ref i18n.GlossaryArchiveReference) (*i18n.TranslationUnitReplacementResolution, error) {
		observeCaller()
		return c.ReplacementResolver(root, catalog, ref)
	}
	probe.DerivedTransportCodecReader = func(a i18n.TranslationUnitAdapter, codec, inventory []byte) (any, any, i18n.TranslationUnitAdapter, error) {
		observeCaller()
		return derivedApprovedCodec(a, codec, inventory)
	}
	s, err := i18n.PrepareQualityCheckCandidateSnapshot(root, &probe, i18n.QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "reconciled-historical-invocation", ContentPackage: o.ContentPackage})
	if err != nil {
		t.Log("ordinary strict caller trace:", strictStack)
		t.Fatal("Snapshot must keep old invocation identity and current reconciled eligibility separate:", err)
	}
	if strictStack != "" {
		t.Fatal("Snapshot candidate validation re-entered ordinary approved reader", strictStack)
	}
	found = false
	for _, u := range s.Units {
		if u.UnitID == historical.CurrentUnitID {
			found = u.SelectedBatchID == batch && u.Attempt == 2
		}
	}
	if !found {
		t.Fatal("current selected successor lost")
	}
	if !bytes.Equal(previousBytes, mustReadFixture(t, root, "data/quality-check-snapshots/"+o.Locale+"/"+o.PreviousSnapshotID+"/manifest.json")) || !bytes.Equal(receiptBytes, mustReadFixture(t, root, reconciliationPath)) {
		t.Fatal("historical mapping evidence changed")
	}
	_, err = i18n.RevalidateRetranslationCandidate(root, c, i18n.RetranslationRevalidateOptions{Locale: o.Locale, BatchID: batch, UnitID: historical.CurrentUnitID})
	if err == nil || !strings.Contains(err.Error(), "approved interval installed decoder/proof drift: "+proofPath) {
		t.Fatal("ordinary revalidation lost strict proof currentness", err)
	}
	// Supplying an ordinary prepared decoder cannot bypass candidate integrity
	// or turn an explicit reconciliation into a guessed source-path mapping.
	for _, tc := range []struct{ name, path string }{
		{"receipt-tamper", reconciliationPath},
		{"ordinary-candidate-tamper", prefix + batch + "/" + result.Units[1].CandidatePath},
		{"source-semantic-drift", currentManifest.Units[1].SourcePath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := mustReadFixture(t, root, tc.path)
			writeFixture(t, root, tc.path, append(bytes.Clone(original), []byte("changed")...))
			defer writeFixture(t, root, tc.path, original)
			if _, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "invalid-reconciled-historical-invocation", ContentPackage: o.ContentPackage}); err == nil {
				t.Fatal("changed evidence accepted")
			}
		})
	}
	filesAfter, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil || !bytes.Equal(proof, filesAfter[proofPath]) {
		t.Fatal("historical proof changed", err)
	}
}
