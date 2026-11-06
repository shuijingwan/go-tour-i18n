package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestIntegratedSurfaceHistoricalPredecessor(t *testing.T) {
	root, c, ids, snapshot := checkpointFixture(t)
	a, options := aggregateSurfaceAuthorityFixture(t, root, c, snapshot)
	finalPath := qualityCheckFinalizationPath(root, options.Locale, snapshot)
	before, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	checkpointGlossary(t, root, "preferred:\n  unused-surface-term: 无关术语\n", "surface-current-review")
	if _, err := VerifyQualityCheckFinalization(root, c, options.Locale, snapshot); err == nil {
		t.Fatal("ordinary current-glossary verifier was weakened")
	}
	legacy := options
	legacy.IntegratedSurface, legacy.Findings, legacy.Finding = false, nil, options.Findings[0]
	legacy.UnitIDs, legacy.ReopenID, legacy.SurfaceReviewID = ids[:1], "legacy-historical-rejected", "legacy-failed"
	writeSnapshotSource(t, root, "data/locale-surface-reviews/zh-CN/legacy-failed.md", []byte(ids[0]+": "+legacy.Finding))
	if _, _, err := RecordQualityCheckSurfaceReopen(root, c, legacy); err == nil {
		t.Fatal("legacy Markdown reopen lost its ordinary current-finalization gate")
	}
	current, path, err := EstablishCurrentTranslationAuthorization(root, c, options.Locale, snapshot)
	if err != nil || !current.Ready() || current.Glossary.SHA256 == current.Historical.Finalization.GlossarySHA256 {
		t.Fatal("historical/current fixture", current, err)
	}
	reject := func(t *testing.T, o QualityCheckSurfaceReopenOptions) {
		t.Helper()
		o.ReopenID = "rejected-historical"
		if _, _, err := RecordQualityCheckSurfaceReopen(root, c, o); err == nil {
			t.Fatal("invalid authority accepted")
		}
		if _, err := os.Stat(qualityCheckSurfaceReopenPath(root, o.Locale, o.ReopenID)); !os.IsNotExist(err) {
			t.Fatal("rejection published authorization", err)
		}
	}
	for _, name := range []string{"subset", "superset", "duplicate", "wrong-unit", "structured", "wrong-attempt"} {
		t.Run(name, func(t *testing.T) {
			o := options
			o.UnitIDs = append([]string(nil), options.UnitIDs...)
			original := a.Findings[0]
			defer func() { a.Findings[0] = original }()
			switch name {
			case "subset":
				o.UnitIDs = o.UnitIDs[:2]
			case "superset":
				o.UnitIDs = append(o.UnitIDs, "unknown")
			case "duplicate":
				o.UnitIDs[1] = o.UnitIDs[0]
			case "wrong-unit":
				o.UnitIDs[0] = "unknown"
			case "structured":
				a.Findings[0].Class = "structured-surface-language"
			case "wrong-attempt":
				a.Findings[0].Artifacts = append([]SurfaceCandidateArtifact(nil), original.Artifacts...)
				a.Findings[0].Artifacts[0].Unit.Attempt++
			}
			reject(t, o)
		})
	}
	for name, ref := range map[string]string{
		"sealed-authorization": path, "current-review": current.UnifiedReview.Path,
		"finalization":        "data/quality-check-snapshots/" + options.Locale + "/" + snapshot + "/finalization.json",
		"historical-glossary": current.Historical.References[0].Path,
		"original-reviewer-a": current.Historical.Finalization.QCResults[0].Path,
		"candidate":           current.Units[0].Snapshot.CandidatePath,
		"validation":          current.Units[0].Snapshot.ValidationPath,
		"current-source":      current.Units[0].Snapshot.SourcePath,
		"surface-receipt":     a.Receipt.Path,
	} {
		t.Run("tampered-"+name, func(t *testing.T) {
			full := filepath.Join(root, filepath.FromSlash(ref))
			data, err := os.ReadFile(full)
			if err != nil {
				t.Fatal(err)
			}
			writeSnapshotSource(t, root, ref, append(bytes.Clone(data), []byte("tampered")...))
			defer writeSnapshotSource(t, root, ref, data)
			reject(t, options)
		})
	}
	t.Run("missing-current-authorization", func(t *testing.T) {
		// Without a valid full current Review neither a checkpoint nor a fresh
		// shared assessment may authorize this old A.
		data, _ := os.ReadFile(filepath.Join(root, current.UnifiedReview.Path))
		if err := os.Remove(filepath.Join(root, current.UnifiedReview.Path)); err != nil {
			t.Fatal(err)
		}
		defer writeSnapshotSource(t, root, current.UnifiedReview.Path, data)
		reject(t, options)
	})
	t.Run("wrong-package-checkpoint", func(t *testing.T) {
		bad := *current
		bad.Package = "different-package"
		bad.Identity = authorizationIdentity(bad)
		if err := writeTranslationJSON(filepath.Join(root, path), bad); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := writeTranslationJSON(filepath.Join(root, path), current); err != nil {
				t.Fatal(err)
			}
		}()
		reject(t, options)
	})
	t.Run("authority-first", func(t *testing.T) {
		reader := c.SurfaceAuthorityReader
		c.SurfaceAuthorityReader = nil
		defer func() { c.SurfaceAuthorityReader = reader }()
		reject(t, options)
	})
	t.Run("expanded-package-and-explicit-owner", func(t *testing.T) {
		expanded, _ := expandedMachinePackage(t, root, c, 2)
		o := options
		o.ReopenID = "historical-expanded"
		r, _, err := RecordQualityCheckSurfaceReopen(root, expanded, o)
		if err != nil || r.UnitCount != len(ids) || len(expanded.TranslationScope) != len(ids)+2 {
			t.Fatal("historical scope replaced current package membership", r, err)
		}
		wrong := *expanded
		wrong.TranslationPackages = append([]TranslationPackageDescriptor(nil), expanded.TranslationPackages...)
		wrong.TranslationPackages[0].ID = "different-current-owner"
		o.ReopenID = "wrong-explicit-package"
		if _, _, err := RecordQualityCheckSurfaceReopen(root, &wrong, o); err == nil {
			t.Fatal("explicit current package did not constrain historical scope")
		}
		o.ReopenID = "new-units-not-authorized"
		o.UnitIDs = append(append([]string(nil), options.UnitIDs...), expanded.TranslationScope[len(ids):]...)
		if _, _, err := RecordQualityCheckSurfaceReopen(root, expanded, o); err == nil {
			t.Fatal("historical Surface findings authorized added Units")
		}
	})
	r, _, err := RecordQualityCheckSurfaceReopen(root, c, options)
	if err != nil || r == nil || r.UnitCount != len(ids) || r.PreviousFinalizationSHA256 != sum(before) {
		t.Fatal("historical predecessor reopen", r, err)
	}
	if _, err := CheckQualityCheckSurfaceReopen(root, c, options.Locale, options.ReopenID); err != nil {
		t.Fatal("current reopen consumer replayed old glossary gate", err)
	}
	if _, _, err := RecordQualityCheckSurfaceReopen(root, c, options); err == nil {
		t.Fatal("immutable reopen overwritten")
	}
	if after, _ := os.ReadFile(finalPath); !bytes.Equal(before, after) {
		t.Fatal("historical finalization changed")
	}
	// A newly selected candidate is a conflicting successor, never a substitute
	// for the exact artifact actually reviewed by Surface.
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: options.Locale, Generator: RetranslationGeneratorChatGPT, UnitIDs: ids, AllowReexport: true, PreviousSnapshotID: snapshot, SurfaceReopenID: options.ReopenID})
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: options.Locale, BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string][]byte{}
	for _, output := range m.ExpectedOutputs {
		outputs[output.UnitID] = files["formal/inputs/"+filepath.Base(output.BundlePath)]
	}
	futureImport(t, root, c, b, outputs)
	if result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: options.Locale, BatchID: e.BatchID}); err != nil || result.ValidationPassed != len(ids) {
		t.Fatal(result, err)
	}
	if _, err := CheckQualityCheckSurfaceReopen(root, c, options.Locale, options.ReopenID); err == nil {
		t.Fatal("consumed/conflicting successor authorization accepted")
	}
	if _, err := readCurrentQualityCheckSurfaceReopen(root, c, options.Locale, options.ReopenID); err != nil {
		t.Fatal("frozen invocation lost immutable original authority", err)
	}
}
