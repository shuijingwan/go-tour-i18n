package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentValidationSnapshotCommitChecks(t *testing.T) {
	root, c, o, e := importedCommentProcessFixture(t)
	r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: o.Locale, BatchID: e.BatchID})
	if err != nil || r.ValidationPassed != 9 {
		t.Fatal(r, err)
	}
	options := QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "isolated-commit-proof"}
	var checks []func() error
	s, err := prepareQualityCheckCandidateSnapshot(root, c, options, &checks)
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) == 0 || s.UnitCount != 49 {
		t.Fatal("missing reusable context or exact set")
	}
	// The complete frozen input produces the same manifest, including stable order.
	again, err := prepareQualityCheckCandidateSnapshot(root, c, options)
	if err != nil || !bytes.Equal(mustJSON(s), mustJSON(again)) {
		t.Fatal("manifest identity changed", err)
	}
	if len(s.Units) != 49 || s.PageCount != 49 || s.ExampleCount != 0 {
		t.Fatal("wrong exact-set")
	}
	revision := 0
	for _, u := range s.Units {
		if u.SelectedBatchID == e.BatchID {
			revision++
		}
	}
	if revision != 9 {
		t.Fatal("predecessor carry/revised delta selection", revision)
	}
	base := "data/retranslation-runs/" + o.Locale + "/" + e.BatchID + "/"
	review, err := RequireUnifiedGlossaryReview(root, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]string{"candidate": s.Units[0].CandidatePath, "validation": s.Units[0].ValidationPath, "source": s.Units[0].SourcePath,
		"glossary": glossaryReviewGlossaryPath(o.Locale), "full-review": review.Path,
		"batch-manifest": base + "manifest.json", "batch-result": base + "result.json",
		"predecessor-snapshot":     "data/quality-check-snapshots/" + o.Locale + "/" + o.PreviousSnapshotID + "/manifest.json",
		"predecessor-finalization": "data/quality-check-snapshots/" + o.Locale + "/" + o.PreviousSnapshotID + "/finalization.json",
		"reopen":                   "data/quality-check-reopens/" + o.Locale + "/" + o.ReopenID + ".json",
		"validator":                "internal/i18n/go_example_validate.go"}
	m, err := readRetranslationProcessManifest(filepath.Join(root, base), o.Locale, e.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	paths["raw"] = base + "raw-responses/" + filepath.Base(m.Units[0].InputPath)
	paths["input"] = base + m.Units[0].InputPath
	for name, path := range paths {
		t.Run(name, func(t *testing.T) {
			before, err := os.ReadFile(filepath.Join(root, path))
			if err != nil {
				t.Fatal(err)
			}
			writeSnapshotSource(t, root, path, append(bytes.Clone(before), '\n'))
			defer writeSnapshotSource(t, root, path, before)
			if _, _, err := writeQualityCheckCandidateSnapshot(root, options, s, checks...); err == nil {
				t.Fatal("concurrent dependency mutation committed")
			}
			if _, err := os.Lstat(filepath.Join(root, "data/quality-check-snapshots", o.Locale, options.SnapshotID)); !os.IsNotExist(err) {
				t.Fatal("partial publication", err)
			}
		})
	}
	t.Run("package-membership", func(t *testing.T) {
		saved := c.TranslationScope
		c.TranslationScope = append([]string(nil), saved[:9]...)
		defer func() { c.TranslationScope = saved }()
		if _, _, err := writeQualityCheckCandidateSnapshot(root, options, s, checks...); err == nil {
			t.Fatal("partial package committed")
		}
	})
	if _, _, err := writeQualityCheckCandidateSnapshot(root, options, s, checks...); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "data/quality-check-snapshots", o.Locale, options.SnapshotID, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := writeQualityCheckCandidateSnapshot(root, options, s, checks...); err == nil {
		t.Fatal("existing snapshot overwritten")
	}
	after, err := os.ReadFile(filepath.Join(root, "data/quality-check-snapshots", o.Locale, options.SnapshotID, "manifest.json"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("write-once bytes changed", err)
	}
}

// Selection capture follows the strict validation's raw-response coordinate,
// including retry paths. Full retry sequence validation remains in the normal
// candidate validator, exercised by TestQualityCheckSnapshotUsesFinalRetryAttempt.
func TestSnapshotPreparedOutputBindsSelectedRetryResponse(t *testing.T) {
	root, c, o, e := importedCommentProcessFixture(t)
	r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: o.Locale, BatchID: e.BatchID})
	if err != nil || r.ValidationPassed != 9 {
		t.Fatal(r, err)
	}
	latest, err := selectLatestRetranslationUnits(root, c, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	u := latest.ordered[0]
	choice := latest.selectedByID[u.ID]
	v, err := readPromotionValidation(choice.batchDir, choice.batchID, o.Locale, choice.manifest, choice.result)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(choice.batchDir, v.RawResponsePath))
	if err != nil {
		t.Fatal(err)
	}
	v.Attempt = 2
	name := filepath.Base(choice.manifest.InputPath)
	flat := name[:len(name)-len(filepath.Ext(name))]
	v.RawResponsePath = "retries/" + flat + "/attempt-002" + filepath.Ext(name)
	base := "data/retranslation-runs/" + o.Locale + "/" + choice.batchID + "/"
	writeSnapshotSource(t, root, base+choice.result.ValidationPath, append(mustJSON(v), '\n'))
	writeSnapshotSource(t, root, base+v.RawResponsePath, raw)
	g, err := LoadGlossary(root, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	_, checks, err := prepareSnapshotRevisionOutputs(root, c, o.Locale, g, latest)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, base+v.RawResponsePath, append(bytes.Clone(raw), '\n'))
	if err := checks[0](); err == nil {
		t.Fatal("changed selected retry raw was accepted")
	}
}
