package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// These are independent disk observations even while the CLI's exact-byte
// transform memo is warm. No lifecycle state is written outside t.TempDir.
func TestCurrentValidationGenerationBundleCurrentnessMatrix(t *testing.T) {
	root, c, o, e := pendingCommentBundleFixture(t)
	options := GenerationBundleOptions{Locale: o.Locale, BatchID: e.BatchID}
	bundle, bm, err := ExportTranslationUnitGenerationBundle(root, c, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(bm.ExpectedOutputs) != 9 || len(c.TranslationScope) != 49 {
		t.Fatal("package exact set")
	}
	dir := filepath.Join(root, "data/retranslation-runs", o.Locale, e.BatchID)
	m, err := readRetranslationProcessManifest(dir, o.Locale, e.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	r, err := readCurrentQualityCheckSurfaceReopen(root, c, o.Locale, o.ReopenID)
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range m.Units {
		name := filepath.Base(record.InputPath)
		for _, path := range []string{"formal/candidates/" + name, "formal/protected-candidates/" + name, "formal/historical-protected-candidates/" + name, "formal/comment-revision-transport/" + name + ".json"} {
			if len(files[path]) == 0 {
				t.Fatal("authorized teaching-comment transport member omitted", path)
			}
		}
	}
	old := r.Units[0].Snapshot
	validation, err := os.ReadFile(filepath.Join(root, old.ValidationPath))
	if err != nil {
		t.Fatal(err)
	}
	var v RetranslationValidation
	if err = json.Unmarshal(validation, &v); err != nil {
		t.Fatal(err)
	}
	review, err := RequireUnifiedGlossaryReview(root, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	rootRel := func(path string) string {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		return filepath.ToSlash(rel)
	}
	paths := map[string]string{
		"glossary-bytes": glossaryReviewGlossaryPath(o.Locale), "unified-review": review.Path,
		"predecessor-snapshot":  "data/quality-check-snapshots/" + o.Locale + "/" + o.PreviousSnapshotID + "/manifest.json",
		"predecessor-candidate": old.CandidatePath, "predecessor-validation": old.ValidationPath,
		"reopen-receipt":           rootRel(qualityCheckSurfaceReopenPath(root, o.Locale, o.ReopenID)),
		"historical-protected-raw": "data/retranslation-runs/" + o.Locale + "/" + old.SelectedBatchID + "/" + v.RawResponsePath,
		"generation-input":         rootRel(filepath.Join(dir, m.Units[0].InputPath)),
	}
	done := contentidentity.BeginMemoization()
	defer done()
	reject := func(t *testing.T, cat *Catalog, b []byte) {
		t.Helper()
		if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, cat, b); err == nil {
			t.Fatal("drift incorrectly CURRENT")
		}
	}
	for name, path := range paths {
		t.Run(name, func(t *testing.T) {
			full := filepath.Join(root, path)
			before, err := os.ReadFile(full)
			if err != nil {
				t.Fatal(err)
			}
			writeSnapshotSource(t, root, path, append(bytes.Clone(before), '\n'))
			defer writeSnapshotSource(t, root, path, before)
			reject(t, c, bundle)
		})
	}
	t.Run("package-partial", func(t *testing.T) {
		copy := *c
		copy.TranslationScope = append([]string(nil), c.TranslationScope[:9]...)
		reject(t, &copy, bundle)
	})
	t.Run("wrong-package-scope", func(t *testing.T) {
		copy := *c
		copy.TranslationScope = []string{"other-package/unit"}
		reject(t, &copy, bundle)
	})
	t.Run("source-bytes-fresh-catalog", func(t *testing.T) {
		// CLI reconstructs/hydrates source authority on every command. Represent
		// that exact new observation instead of passing an obsolete in-memory TU.
		copy := *c
		copy.Adapted = append([]AdaptedTranslationUnit(nil), c.Adapted...)
		b := copy.Adapted[0]
		path := b.Unit.SourcePath
		original, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		changed := append(bytes.Clone(original), '\n')
		writeSnapshotSource(t, root, path, changed)
		defer writeSnapshotSource(t, root, path, original)
		b.Unit.Source, b.Unit.SourceSHA256 = changed, sum(changed)
		copy.Adapted[0] = b
		reject(t, &copy, bundle)
	})
	for _, name := range []string{"wrong-predecessor", "wrong-reopen"} {
		t.Run(name, func(t *testing.T) {
			before, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			var changed RetranslationBatchManifest
			if err = json.Unmarshal(before, &changed); err != nil {
				t.Fatal(err)
			}
			if name == "wrong-predecessor" {
				changed.Units[0].PreviousSnapshotID = "wrong-predecessor"
			} else {
				changed.Units[0].RevisionAuthorizationID = "wrong-reopen"
			}
			writeSnapshotSource(t, root, rootRel(filepath.Join(dir, "manifest.json")), append(mustJSON(changed), '\n'))
			defer writeSnapshotSource(t, root, rootRel(filepath.Join(dir, "manifest.json")), before)
			reject(t, c, bundle)
		})
	}
	t.Run("authorized-comment-slot-set", func(t *testing.T) {
		path := qualityCheckSurfaceReopenPath(root, o.Locale, o.ReopenID)
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var changed QualityCheckSurfaceReopen
		if err = json.Unmarshal(before, &changed); err != nil {
			t.Fatal(err)
		}
		changed.CurrentValidation.Findings[0].CommentSlots = changed.CurrentValidation.Findings[0].CommentSlots[:1]
		writeSnapshotSource(t, root, rootRel(path), currentValidationReopenBytes(&changed))
		defer writeSnapshotSource(t, root, rootRel(path), before)
		reject(t, c, bundle)
	})
	for _, name := range []string{"comment-projection-mapping", "files-inventory", "aggregate-identity"} {
		t.Run(name, func(t *testing.T) {
			manifest, files, err := readGenerationBundle(bundle)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "comment-projection-mapping":
				path := commentTransportPath(m.Units[0])
				var p CommentRevisionTransport
				if err = json.Unmarshal(files[path], &p); err != nil {
					t.Fatal(err)
				}
				p.Values[0] += "unauthorized"
				files[path] = mustJSON(p)
				var identity strings.Builder
				for i, f := range manifest.Files {
					manifest.Files[i] = NewTransportBundleFile(f.BundlePath, f.RepositoryPath, files[f.BundlePath])
					identity.WriteString(f.BundlePath + "\x00" + manifest.Files[i].SHA256 + "\x00")
				}
				manifest.InputIdentitySHA256 = sum([]byte(identity.String()))
			case "files-inventory":
				manifest.Files[0].SHA256 = strings.Repeat("0", 64)
			case "aggregate-identity":
				manifest.InputIdentitySHA256 = strings.Repeat("0", 64)
			}
			entries := []TransportBundleEntry{}
			for path, b := range files {
				if path != "manifest.json" {
					entries = append(entries, TransportBundleEntry{Path: path, Data: b})
				}
			}
			data, err := json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			edited, err := WriteDeterministicTransportBundle(append(data, '\n'), entries)
			if err != nil {
				t.Fatal(err)
			}
			reject(t, c, edited)
		})
	}
	if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err != nil {
		t.Fatal("unchanged transport not CURRENT", err)
	}
}
