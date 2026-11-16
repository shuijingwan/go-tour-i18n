package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSnapshotOutputContextPreservesFullCurrentAndMemberCatalogs(t *testing.T) {
	root, current, o, e := importedCommentProcessFixture(t, true)
	if result, err := ProcessRetranslationBatch(root, current, RetranslationProcessOptions{Locale: o.Locale, BatchID: e.BatchID}); err != nil || result.ValidationPassed != 9 {
		t.Fatal(result, err)
	}
	full := current.TranslationPackages[0]
	child, sibling := full, full
	child.AuthoritySHA256 = sum([]byte("smaller historical member"))
	child.Units = full.Units[:len(full.Units)-2]
	sibling.ID, sibling.AuthoritySHA256, sibling.Units = "member-addition", sum([]byte("independent member")), full.Units[len(full.Units)-2:]
	declaration := TranslationPackageMembership{Predecessors: []TranslationPackagePredecessor{{child.ID, child.AuthoritySHA256}, {sibling.ID, sibling.AuthoritySHA256}}}
	path := "fixtures/output-context-membership.json"
	data := mustJSON(declaration)
	writeSnapshotSource(t, root, path, data)
	declaration.References = []GlossaryArchiveReference{{path, sum(data)}}
	current.TranslationPackages[0].Membership = &declaration
	current.HistoricalTranslationPackages = append(current.HistoricalTranslationPackages, full, child, sibling)
	members, _, err := snapshotMembershipCatalogs(root, current)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, e.BatchPath)
	m, err := readRetranslationProcessManifest(dir, o.Locale, e.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	evidence := members[m.Units[0].UnitID]
	evidence.HistoricalTranslationPackages = nil
	g, err := LoadGlossary(root, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := selectLatestRetranslationUnits(root, current, o.Locale)
	if err != nil {
		t.Fatal(err)
	}
	// Reproduce the production boundary losing its current argument. The child
	// cannot discover the frozen complete package from any hidden registry.
	if _, _, err := prepareSnapshotOutputContext(dir, evidence, evidence, g, m, latest.selectedByID); err == nil {
		t.Fatal("member catalog masqueraded as full current authority")
	}
	outputs, check, err := prepareSnapshotOutputContext(dir, current, evidence, g, m, latest.selectedByID)
	if err != nil || len(outputs) != len(m.Units) {
		t.Fatal("production output context lost full current catalog", err)
	}
	for _, record := range m.Units {
		raw, err := os.ReadFile(filepath.Join(dir, "raw-responses", filepath.Base(record.InputPath)))
		if err != nil {
			t.Fatal(err)
		}
		text, failed := outputs[record.UnitID].restore(string(raw))
		want, err := os.ReadFile(filepath.Join(dir, "candidates", filepath.Base(record.InputPath)))
		if err != nil || len(failed) != 0 || !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(text)), want) {
			t.Fatal("frozen decoder lost exact candidate bytes", failed, err)
		}
	}
	if err := check(); err != nil {
		t.Fatal(err)
	}
	frozen, err := snapshotRevisionEvidenceCatalog(root, current, o.Locale, latest.selectedByID[m.Units[0].UnitID], evidence)
	if err != nil {
		t.Fatal(err)
	}
	_, identity, _, ordered, err := translationAuthorizationScope(frozen)
	if err != nil || identity != full.AuthoritySHA256 || len(ordered) != len(full.Units) || !reflect.DeepEqual(frozen.TranslationScope, current.TranslationScope) {
		t.Fatal("PackageIdentity/OrderedScope differ from complete frozen package", err)
	}
	if len(evidence.TranslationScope) != len(child.Units) || evidence.TranslationPackages[0].AuthoritySHA256 != child.AuthoritySHA256 {
		t.Fatal("decoder lookup replaced historical member authority")
	}
	for _, mode := range []string{"missing-snapshot", "wrong-frozen-authority", "wrong-order"} {
		t.Run(mode, func(t *testing.T) {
			bad := *current
			bad.TranslationPackages = append([]TranslationPackageDescriptor(nil), current.TranslationPackages...)
			bad.HistoricalTranslationPackages = append([]TranslationPackageDescriptor(nil), current.HistoricalTranslationPackages...)
			if mode == "missing-snapshot" {
				p := qualityCheckSnapshotManifestPath(root, o.Locale, o.PreviousSnapshotID)
				saved, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.WriteFile(p, saved, 0644); err != nil {
						t.Error(err)
					}
				}()
			} else {
				for _, descriptors := range [][]TranslationPackageDescriptor{bad.TranslationPackages, bad.HistoricalTranslationPackages} {
					for i, d := range descriptors {
						if d.ID != full.ID || d.AuthoritySHA256 != full.AuthoritySHA256 {
							continue
						}
						if mode == "wrong-frozen-authority" {
							descriptors[i].AuthoritySHA256 = sum([]byte("unrelated same-name authority"))
						} else {
							d.Units = append([]AdaptedTranslationUnit(nil), d.Units...)
							d.Units[0], d.Units[1] = d.Units[1], d.Units[0]
							descriptors[i] = d
						}
					}
				}
			}
			if _, _, err := prepareSnapshotOutputContext(dir, &bad, evidence, g, m, latest.selectedByID); err == nil {
				t.Fatal("wrong frozen package accepted at production output boundary")
			}
		})
	}
}

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
	priorValidation, err := os.ReadFile(filepath.Join(choice.batchDir, choice.result.ValidationPath))
	if err != nil {
		t.Fatal(err)
	}
	v.Attempt = 2
	name := filepath.Base(choice.manifest.InputPath)
	flat := name[:len(name)-len(filepath.Ext(name))]
	v.RawResponsePath = "retries/" + flat + "/attempt-002" + filepath.Ext(name)
	base := "data/retranslation-runs/" + o.Locale + "/" + choice.batchID + "/"
	writeSnapshotSource(t, root, base+"retries/"+flat+"/attempt-001-validation.json", priorValidation)
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
