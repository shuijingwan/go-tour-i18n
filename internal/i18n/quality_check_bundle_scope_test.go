package i18n

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Reuse the exact preview-reopen fixture. Everything created below is confined
// to t.TempDir: one finalized package, one revised Unit, and an unrelated
// unfinalized Tour Snapshot between predecessor and successor.
func TestQualityCheckReviewerBundleExactPackageScope(t *testing.T) {
	root, c, options, predecessor := previewRevisionPackageFixture(t)
	b, m, err := ExportTranslationUnitGenerationBundle(root, c, options)
	if err != nil {
		t.Fatal(err)
	}
	_, generationFiles, err := readGenerationBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(m.ExpectedOutputs[0].BundlePath)
	raw := generationFiles["formal/protected-candidates/"+name]
	revised := bytes.Replace(raw, []byte("unique"), []byte("revised"), 1)
	if bytes.Equal(raw, revised) {
		t.Fatal("fixture did not revise the candidate")
	}
	futureImport(t, root, c, b, map[string][]byte{predecessor.UnitID: revised})
	if result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: options.BatchID}); err != nil || result.ValidationPassed != 1 {
		t.Fatalf("fixture revision: %+v %v", result, err)
	}
	inventory := *c
	inventory.TranslationScope = nil
	materializeSnapshotSources(t, root, &inventory)
	addProcessedPromotionBatch(t, root, &inventory, "codex-zh-CN-090", []string{inventory.Pages[0].ID})
	abandoned := futureSnapshot(t, root, &inventory, "scope-qc-013")
	if abandoned.UnitCount != 1 || abandoned.Units[0].UnitID == predecessor.UnitID {
		t.Fatal("fixture did not create unrelated Tour scope")
	}
	successor := futureSnapshot(t, root, c, "scope-qc-014")
	if successor.CurrentRevision == nil || len(successor.CurrentRevision.Delta) != 1 {
		t.Fatal("fixture lost preview revision proof")
	}
	copyBundleAuthority(t, root, qualityCheckReviewerAuthorityPaths)
	previous := readRetranslationManifest(t, root, options.BatchID).Units[0].PreviousSnapshotID
	bundle, manifest, err := ExportQualityCheckReviewerBundle(root, c, QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: successor.SnapshotID, PreviousSnapshotID: previous, StartIndex: predecessor.Index, Limit: 60})
	if err != nil || manifest.UnitCount != 1 {
		t.Fatalf("scoped reviewer export: %+v %v", manifest, err)
	}
	files, err := ReadTransportBundle(bundle, 512, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	var context QualityCheckReviewerContext
	if err := json.Unmarshal(files["quality-check.json"], &context); err != nil {
		t.Fatal(err)
	}
	if context.Scope.UnitCount != 3 || context.Scope.CarryForwardCount != 2 || context.Scope.PendingCount != 1 || context.Units[0].UnitID != predecessor.UnitID || context.PreviousSnapshotID != previous {
		t.Fatalf("scope changed: %+v", context.Scope)
	}
	for _, memo := range []bool{false, true} {
		t.Run(map[bool]string{false: "memo-off", true: "memo-on"}[memo], func(t *testing.T) {
			if memo {
				done := contentidentity.BeginMemoization()
				defer done()
			}
			checked, err := VerifyCurrentQualityCheckReviewerBundle(root, &inventory, bundle)
			if err != nil || checked.InputIdentitySHA256 != manifest.InputIdentitySHA256 {
				t.Fatalf("unscoped canonical rebuild: identity=%s %v", checked.InputIdentitySHA256, err)
			}
			if len(inventory.TranslationScope) != 0 {
				t.Fatal("current-check mutated caller scope")
			}
		})
	}
	// Even a numerically newer unrelated Snapshot cannot become this lineage.
	futureSnapshot(t, root, &inventory, "scope-qc-099")
	if _, err := VerifyCurrentQualityCheckReviewerBundle(root, &inventory, bundle); err != nil {
		t.Fatal("new unrelated Snapshot polluted currentness", err)
	}
	for _, id := range []string{"scope-qc-013", "scope-qc-099"} {
		if _, err := os.Stat(qualityCheckSnapshotManifestPath(root, "zh-CN", id)); err != nil {
			t.Fatal("abandoned Snapshot removed", err)
		}
		for _, path := range []string{qualityCheckResultsPath(root, "zh-CN", id), qualityCheckFinalizationPath(root, "zh-CN", id)} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("abandoned Snapshot entered QC/finalization", err)
			}
		}
	}
	t.Run("explicit-wrong-scope", func(t *testing.T) {
		wrong := inventory
		wrong.TranslationScope = []string{inventory.Pages[0].ID}
		if _, err := VerifyCurrentQualityCheckReviewerBundle(root, &wrong, bundle); err == nil {
			t.Fatal("explicit wrong scope inferred away")
		}
	})
	t.Run("wrong-predecessor", func(t *testing.T) {
		wrong := manifest
		wrong.PreviousSnapshotID = "scope-qc-013"
		data, _ := json.Marshal(wrong)
		var entries []TransportBundleEntry
		for path, data := range files {
			if path != "manifest.json" {
				entries = append(entries, TransportBundleEntry{Path: path, Data: data})
			}
		}
		tampered, err := WriteDeterministicTransportBundle(data, entries)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := VerifyCurrentQualityCheckReviewerBundle(root, &inventory, tampered); err == nil {
			t.Fatal("wrong predecessor accepted")
		}
	})
	t.Run("canonical-byte-comparison", func(t *testing.T) {
		var altered bytes.Buffer
		writer := zip.NewWriter(&altered)
		if err := writer.SetComment("non-canonical transport with unchanged inputs"); err != nil {
			t.Fatal(err)
		}
		for path, data := range files {
			h := &zip.FileHeader{Name: path, Method: zip.Store}
			h.SetMode(0644)
			w, err := writer.CreateHeader(h)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(data); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := VerifyCurrentQualityCheckReviewerBundle(root, &inventory, altered.Bytes()); err == nil || !strings.Contains(err.Error(), "stale or non-canonical") {
			t.Fatalf("canonical comparison weakened: %v", err)
		}
	})
}
