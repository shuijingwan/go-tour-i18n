package sitecontent

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func mustReadFixture(t *testing.T, root, path string) []byte {
	t.Helper()
	b, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Snapshot admission separates trusted historical PASS from reproduction drift;
// it grants no installed-invocation recovery or proof refresh capability.
// The caller has just imported, processed, retried and snapshotted a legitimate
// synthetic approved-interval invocation. Every path here is in its TempDir.
func checkInstalledApprovedProofStaleRecoveryBoundary(t *testing.T, root string, c *i18n.Catalog, reopen i18n.CurrentValidationReopenOptions, batch, unit string, export i18n.GenerationBundleOptions) {
	t.Helper()
	// This older domain fixture builds descriptors directly. Supply the same
	// source-owned strict codec reader installed by production registration.
	c.DerivedTransportCodecReader = derivedApprovedCodec
	inventory := func() map[string]string {
		t.Helper()
		out := map[string]string{}
		for _, dir := range []string{"data", "locales", "_content", "production"} {
			if err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
				if os.IsNotExist(err) {
					return nil
				}
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				out[path] = digest(b)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	before := inventory()
	batchPath := "data/retranslation-runs/" + reopen.Locale + "/" + batch
	data, err := readRegular(root, batchPath+"/generation-imports/initial-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var imported i18n.GenerationResultBundleManifest
	if err := json.Unmarshal(data, &imported); err != nil {
		t.Fatal(err)
	}
	bundle, err := readRegular(root, "data/generation-archives/"+reopen.Locale+"/"+imported.GenerationBundleSHA256+".zip")
	if err != nil {
		t.Fatal(err)
	}
	// A decoder dependency successor changes proof identity even if this tiny
	// fixture's logical response still happens to restore the same candidate.
	path := "internal/i18n/candidate_snapshot_outputs.go"
	old, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, path, append(append([]byte(nil), old...), []byte("\n// Synthetic decoder implementation successor.\n")...))
	t.Cleanup(func() { writeFixture(t, root, path, old) })
	t.Run("snapshot-fresh-current-proof-historical-pass", func(t *testing.T) {
		calls := 0
		probe := snapshotProjectionProbeCatalog(c, &calls)
		files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
		if err != nil {
			t.Fatal(err)
		}
		var installed i18n.ApprovedIntervalTransport
		if err := json.Unmarshal(files["formal/approved-interval-transport/"+unit+".json"], &installed); err != nil {
			t.Fatal(err)
		}
		var manifest i18n.RetranslationBatchManifest
		if err := json.Unmarshal(mustReadFixture(t, root, batchPath+"/manifest.json"), &manifest); err != nil {
			t.Fatal(err)
		}
		_, proofs, err := i18n.InspectApprovedIntervalReadiness(root, probe, &manifest, files["formal/approved-interval-selection.json"])
		if err != nil || len(proofs) != 1 || proofs[0].Identity == installed.Identity || !reflect.DeepEqual(proofs[0].Projection, installed.Projection) {
			t.Fatal("fresh proof must differ only outside the semantic projection", proofs, err)
		}
		calls = 0
		if _, err := i18n.PrepareQualityCheckCandidateSnapshot(root, probe, i18n.QualityCheckSnapshotOptions{Locale: reopen.Locale, SnapshotID: "fresh-approved-pass", ContentPackage: reopen.ContentPackage}); err != nil {
			t.Fatal(err)
		}
		if calls < 2 {
			t.Fatal("Snapshot substituted the installed proof for fresh construction")
		}
		if !reflect.DeepEqual(before, inventory()) {
			t.Fatal("Snapshot admission modified historical evidence")
		}
	})
	t.Run("snapshot-admits-historical-pass-with-implementation-drift", func(t *testing.T) {
		s, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: reopen.Locale, SnapshotID: "historical-pass-successor", ContentPackage: reopen.ContentPackage})
		if err != nil {
			t.Fatal("complete historical PASS must remain Snapshot eligible", err)
		}
		found := false
		for _, u := range s.Units {
			if u.UnitID == unit {
				found = u.SelectedBatchID == batch && u.Attempt == 2
			}
		}
		if !found {
			t.Fatal("Snapshot lost exact historical retry selection")
		}
		if !reflect.DeepEqual(before, inventory()) {
			t.Fatal("historical PASS preflight mutated evidence")
		}
	})
	t.Run("pending-overlapping-successor-rejected", func(t *testing.T) {
		var manifest i18n.RetranslationBatchManifest
		if err := json.Unmarshal(mustReadFixture(t, root, batchPath+"/manifest.json"), &manifest); err != nil {
			t.Fatal(err)
		}
		manifest.BatchID = batch + "9"
		path := "data/retranslation-runs/" + reopen.Locale + "/" + manifest.BatchID
		b, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path+"/manifest.json", b)
		defer os.RemoveAll(filepath.Join(root, path))
		_, err = i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: reopen.Locale, SnapshotID: "conflicting-successor", ContentPackage: reopen.ContentPackage})
		if err == nil || !strings.Contains(err.Error(), "conflicting successor") {
			t.Fatalf("pending overlapping successor must block historical admission: %v", err)
		}
	})
	for _, suffix := range []string{".json", ".record.pending"} {
		t.Run("recovery-lineage-conflict"+suffix, func(t *testing.T) {
			path := "data/translation-transport-recoveries/" + reopen.Locale + "/roots/" + imported.GenerationBundleSHA256 + suffix
			writeFixture(t, root, path, []byte("occupied"))
			defer os.Remove(filepath.Join(root, path))
			_, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: reopen.Locale, SnapshotID: "conflicting-recovery", ContentPackage: reopen.ContentPackage})
			if err == nil || !strings.Contains(err.Error(), "conflicts with recovery lineage") {
				t.Fatalf("occupied recovery lineage must block historical admission: %v", err)
			}
		})
	}
	var result i18n.RetranslationProcessResult
	if err := json.Unmarshal(mustReadFixture(t, root, batchPath+"/result.json"), &result); err != nil {
		t.Fatal(err)
	}
	var selected i18n.RetranslationUnitResult
	for _, u := range result.Units {
		if u.UnitID == unit {
			selected = u
		}
	}
	current, err := c.Unit(unit)
	if err != nil {
		t.Fatal(err)
	}
	var finalValidation i18n.RetranslationValidation
	if err := json.Unmarshal(mustReadFixture(t, root, batchPath+"/"+selected.ValidationPath), &finalValidation); err != nil {
		t.Fatal(err)
	}
	t.Run("historical-proof-member-missing", func(t *testing.T) {
		files, err := i18n.ReadTransportBundle(bundle, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
		if err != nil {
			t.Fatal(err)
		}
		var manifest i18n.GenerationBundleManifest
		if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
			t.Fatal(err)
		}
		missing := "formal/approved-interval-transport/" + unit + ".json"
		kept := []i18n.TransportBundleFile{}
		entries := []i18n.TransportBundleEntry{}
		var identity strings.Builder
		for _, f := range manifest.Files {
			if f.BundlePath == missing {
				continue
			}
			kept = append(kept, f)
			entries = append(entries, i18n.TransportBundleEntry{Path: f.BundlePath, Data: files[f.BundlePath]})
			identity.WriteString(f.BundlePath + "\x00" + f.SHA256 + "\x00")
		}
		if len(kept)+1 != len(manifest.Files) {
			t.Fatal("missing-proof perturbation ineffective")
		}
		manifest.Files = kept
		manifest.InputIdentitySHA256 = digest([]byte(identity.String()))
		mb, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		changed, err := i18n.WriteDeterministicTransportBundle(mb, entries)
		if err != nil {
			t.Fatal(err)
		}
		r := imported
		r.GenerationBundleSHA256 = digest(changed)
		r.InputIdentitySHA256 = manifest.InputIdentitySHA256
		r.GenerationManifest = i18n.NewTransportBundleFile("generation-manifest.json", "", mb)
		receiptPath := batchPath + "/generation-imports/initial-manifest.json"
		oldReceipt := mustReadFixture(t, root, receiptPath)
		defer writeFixture(t, root, receiptPath, oldReceipt)
		archivePath := "data/generation-archives/" + reopen.Locale + "/" + r.GenerationBundleSHA256 + ".zip"
		writeFixture(t, root, archivePath, changed)
		defer os.Remove(filepath.Join(root, archivePath))
		rb, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, receiptPath, rb)
		if _, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: reopen.Locale, SnapshotID: "missing-historical-proof", ContentPackage: reopen.ContentPackage}); err == nil {
			t.Fatal("missing frozen proof admitted despite an otherwise readable resealed archive")
		}
	})
	for _, tc := range []struct {
		name, path string
		change     func([]byte) []byte
	}{
		{"snapshot-rejects-stale-proof", "data/generation-archives/" + reopen.Locale + "/" + imported.GenerationBundleSHA256 + ".zip", func(b []byte) []byte { return append([]byte("tampered"), b...) }},
		{"candidate-tamper", batchPath + "/" + selected.CandidatePath, func(b []byte) []byte { return append(b, []byte("changed")...) }},
		{"raw-tamper", batchPath + "/" + finalValidation.RawResponsePath, func(b []byte) []byte { return append(b, []byte("changed")...) }},
		{"validation-tamper", batchPath + "/" + selected.ValidationPath, func(b []byte) []byte { return append(b, '\n') }},
		{"source-semantic-change", current.SourcePath, func(b []byte) []byte { return append(b, []byte("\nchanged source\n")...) }},
		{"incomplete-historical-pass", batchPath + "/result.json", func(b []byte) []byte {
			var r i18n.RetranslationProcessResult
			if err := json.Unmarshal(b, &r); err != nil {
				t.Fatal(err)
			}
			r.ValidationPassed--
			r.ValidationFailed++
			out, err := json.MarshalIndent(r, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			return append(out, '\n')
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := mustReadFixture(t, root, tc.path)
			writeFixture(t, root, tc.path, tc.change(append([]byte{}, old...)))
			defer writeFixture(t, root, tc.path, old)
			if _, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: reopen.Locale, SnapshotID: "invalid-historical-pass", ContentPackage: reopen.ContentPackage}); err == nil {
				t.Fatal("invalid historical PASS admitted")
			}
		})
	}
	for _, tc := range []struct {
		name, want string
		call       func() error
	}{
		{"same-batch-reexport-rejected", "formal raw-responses already exists", func() error {
			_, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, export)
			return err
		}},
		{"passed-unit-retry-export-rejected", "not in retryable", func() error {
			_, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: reopen.Locale, BatchID: batch, UnitID: unit})
			return err
		}},
		{"retry-cannot-refresh-proof", "installed decoder/proof drift", func() error {
			_, err := i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: reopen.Locale, BatchID: batch, UnitID: unit})
			return err
		}},
		{"revalidate-cannot-refresh-proof", "installed decoder/proof drift", func() error {
			_, err := i18n.RevalidateRetranslationCandidate(root, c, i18n.RetranslationRevalidateOptions{Locale: reopen.Locale, BatchID: batch, UnitID: unit})
			return err
		}},
		{"consumed-reopen-not-reusable", "conflicting current candidate", func() error {
			_, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: reopen.Locale, Generator: i18n.RetranslationGeneratorChatGPT, AllowReexport: true, UnitIDs: []string{unit}, UnitKind: i18n.UnitKindPage, PreviousSnapshotID: reopen.PreviousSnapshotID, SurfaceReopenID: reopen.ReopenID})
			return err
		}},
		{"new-reopen-does-not-authorize-replay", "conflicting current candidate", func() error {
			next := reopen
			next.ReopenID = "installed-proof-stale-reopen"
			_, _, err := i18n.InspectCurrentValidationReopen(root, c, next)
			return err
		}},
		{"preimport-recover-cannot-replay-installed-invocation", "formal raw-responses already exists", func() error {
			_, err := i18n.RecoverGenerationOutputs(root, c, bundle, []string{filepath.Join(root, batchPath, "raw-responses")}, filepath.Join(t.TempDir(), "recovered"))
			return err
		}},
		{"import-cannot-replay-installed-invocation", "formal raw-responses already exists", func() error {
			_, err := i18n.ImportGenerationOutputDirectory(root, c, bundle, "chatgpt", i18n.FormalChatGPTGenerationModel, filepath.Join(root, batchPath, "raw-responses"))
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unsafe recovery accepted or wrong failure: %v; want %q", err, tc.want)
			}
			t.Log(err)
			if !reflect.DeepEqual(before, inventory()) {
				t.Fatal("stale recovery attempt changed historical formal state")
			}
		})
	}
}
