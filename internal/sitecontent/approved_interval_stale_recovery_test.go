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

// This is a rejection proof, not an installed-invocation recovery capability.
// The caller has just imported, processed, retried and snapshotted a legitimate
// synthetic approved-interval invocation. Every path here is in its TempDir.
func checkInstalledApprovedProofStaleRecoveryBoundary(t *testing.T, root string, c *i18n.Catalog, reopen i18n.CurrentValidationReopenOptions, batch, unit string, export i18n.GenerationBundleOptions) {
	t.Helper()
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
	for _, tc := range []struct {
		name, want string
		call       func() error
	}{
		{"snapshot-rejects-stale-proof", "installed decoder/proof drift", func() error {
			_, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: reopen.Locale, SnapshotID: "stale-proof-successor", ContentPackage: reopen.ContentPackage})
			return err
		}},
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
