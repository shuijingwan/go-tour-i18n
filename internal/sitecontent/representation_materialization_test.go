package sitecontent

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestRepresentationOnlySuccessorPackageLifecycle(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, old, current, frozen := replacementPageFixtureSize(t, 49, 5, func(i int, b []byte) []byte {
		if i == 1 {
			return bytes.Replace(b, []byte("Read this message."), []byte("Use GlossaryTerm here."), 1)
		}
		return b
	})
	for _, path := range []string{"internal/tour/ui/zh-CN.json"} {
		b, err := os.ReadFile(filepath.Join(repositoryRoot(t), path))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path, b)
	}
	writeFixture(t, root, "locales/zh-CN/article-metadata.json", []byte(`{"locale":"zh-CN","articles":[]}`))
	writeFixture(t, root, "locales/zh-CN/course-metadata.json", []byte(`{"schema_version":1,"locale":"zh-CN","generator_contract":"course-seo-description-v1","pages":[]}`))
	for _, path := range []string{"internal/tour/languages.go", "internal/tour/tour.go", "internal/tour/production.go", "internal/tour/project.go", "internal/tour/seo.go", "_content/js/playground.js", "_content/tour/static/js/app.js", "_content/tour/template/index.tmpl", "_content/tour/static/partials/index.html"} {
		writeFixture(t, root, path, []byte("package tour\nconst Config = \"Tour\"\n"))
	}
	oldIDs := append([]string(nil), old.TranslationScope...)
	currentIDs := append([]string(nil), current.TranslationScope...)
	// All model/Reviewer operations below are synthetic fixture setup in TempDir.
	// The reconciliation itself never invokes or records either role.
	initial, err := i18n.ExportRetranslationBatch(root, old, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: oldIDs})
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := i18n.ExportTranslationUnitGenerationBundle(root, old, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: initial.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(b, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string][]byte{}
	for _, o := range m.ExpectedOutputs {
		outputs[o.UnitID] = files["formal/inputs/"+filepath.Base(o.BundlePath)]
	}
	surfaceImport(t, root, old, b, outputs)
	if p, err := i18n.ProcessRetranslationBatch(root, old, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: initial.BatchID}); err != nil || p.ValidationPassed != 49 {
		t.Fatal(p, err)
	}
	const previous = "qc-old-representation"
	snapshot, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, old, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: previous})
	if err != nil {
		t.Fatal(err)
	}
	const finding = "Localize the visible message."
	as := append(append([]string(nil), oldIDs[:5]...), oldIDs[6:]...)
	for _, r := range []struct {
		ids             []string
		rating, finding string
	}{{as, "A", ""}, {oldIDs[5:6], "B", finding}} {
		if _, err := i18n.RecordQualityCheckResults(root, old, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: previous, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: r.ids, Rating: r.rating, Finding: r.finding}); err != nil {
			t.Fatal(err)
		}
	}
	historical := freezeReplacementFixture(t, root, frozen)
	mapping, err := resolvePageReplacements(root, current, historical)
	if err != nil {
		t.Fatal(err)
	}
	// A real revision transport in the fixture stands for the completed 047.
	p := mapping.Units[oldIDs[5]]
	proof := i18n.RepresentationNecessityProof{Reason: "protected-natural-language-unrepresentable", Units: []i18n.RepresentationNecessityUnit{{Replacement: p, Rating: "B", FindingSHA256: digest([]byte(finding))}}}
	data, _ := json.Marshal(proof)
	writeFixture(t, root, "proof.json", data)
	_, _, err = i18n.RecordQualityCheckSurfaceReopen(root, current, i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: "quality-check", Locale: "zh-CN", ReopenID: "qc-B-successor", PreviousSnapshotID: previous, PreviousRating: "B", Finding: finding, UnitIDs: currentIDs[5:6], HistoricalDocuments: historical, RepresentationProof: i18n.GlossaryArchiveReference{Path: "proof.json", SHA256: digest(data)}})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := i18n.ExportRetranslationBatch(root, current, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, BatchID: "chatgpt-zh-CN-047", UnitIDs: currentIDs[5:6], AllowReexport: true, PreviousSnapshotID: previous, SurfaceReopenID: "qc-B-successor"})
	if err != nil {
		t.Fatal(err)
	}
	b, m, err = i18n.ExportTranslationUnitGenerationBundle(root, current, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: revision.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, err = i18n.ReadTransportBundle(b, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	surfaceImport(t, root, current, b, map[string][]byte{currentIDs[5]: bytes.Replace(files["formal/protected-candidates/"+filepath.Base(m.ExpectedOutputs[0].BundlePath)], []byte("Page"), []byte("Revised page"), 1)})
	if p, err := i18n.ProcessRetranslationBatch(root, current, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: revision.BatchID}); err != nil || p.ValidationPassed != 1 {
		t.Fatal(p, err)
	}
	rows := []map[string]any{}
	for i := 5; i < 49; i++ {
		p := mapping.Units[oldIDs[i]]
		rows = append(rows, map[string]any{"current_unit_id": p.CurrentUnitID, "snapshot_representation_unit_id": p.OldUnitID, "source_path": p.SourcePath, "source_sha256": p.SourceSHA256, "labels": map[string]bool{"representation_reconciliation_required": true, "true_language_defect": i == 5, "unknown": i == 6}})
	}
	impact, _ := json.Marshal(map[string]any{"units": rows})
	writeFixture(t, root, "impact.json", impact)
	manifestPath := "data/quality-check-snapshots/zh-CN/" + previous + "/manifest.json"
	oldManifest, _ := readRegular(root, manifestPath)
	o := i18n.RepresentationReconciliationOptions{Locale: "zh-CN", Package: frozen.Package, ReconciliationID: "machine-successors", PreviousSnapshotID: previous, SnapshotSHA256: digest(oldManifest), Impact: i18n.GlossaryArchiveReference{Path: "impact.json", SHA256: digest(impact)}, HistoricalDocuments: historical}
	before, err := i18n.BuildTranslationUnitProgress(root, current, "zh-CN")
	if err != nil || before.Missing != 43 || before.Passed != 6 {
		t.Fatal(before, err)
	}
	for _, name := range []string{"snapshot", "impact", "frozen", "package", "language", "unknown"} {
		t.Run(name, func(t *testing.T) {
			bad := o
			switch name {
			case "snapshot":
				bad.SnapshotSHA256 = strings.Repeat("0", 64)
			case "impact":
				bad.Impact.SHA256 = strings.Repeat("0", 64)
			case "frozen":
				bad.HistoricalDocuments.SHA256 = strings.Repeat("0", 64)
			case "package":
				bad.Package = "wrong-package"
			case "language", "unknown":
				var copy map[string]any
				json.Unmarshal(impact, &copy)
				row := copy["units"].([]any)[1].(map[string]any)
				if name == "language" {
					row["labels"].(map[string]any)["true_language_defect"] = true
				} else {
					row["source_sha256"] = strings.Repeat("0", 64)
				}
				b, _ := json.Marshal(copy)
				writeFixture(t, root, "bad-impact.json", b)
				bad.Impact = i18n.GlossaryArchiveReference{Path: "bad-impact.json", SHA256: digest(b)}
			}
			bad.Apply = true
			p, err := i18n.ReconcileTranslationUnitRepresentations(root, current, bad)
			if err == nil {
				t.Fatal("untrusted reconciliation accepted", p)
			}
			if _, err := os.Stat(filepath.Join(root, "data/translation-unit-reconciliations/zh-CN/machine-successors")); !os.IsNotExist(err) {
				t.Fatal("rejected apply mutated", err)
			}
		})
	}
	plan, err := i18n.ReconcileTranslationUnitRepresentations(root, current, o)
	if err != nil || plan.Ready != 43 || plan.Excluded != 1 || plan.Blocked != 0 || plan.Applied {
		t.Fatal(plan, err)
	}
	if !plan.Receipt.Units[0].UnknownResolvedMachine {
		t.Fatal("deterministic unknown proof missing")
	}
	// A separate frozen copy starts with the real 5 passed / 43 missing / 1
	// glossary-stale shape before any reconciliation is applied.
	staleRoot := t.TempDir()
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		writeFixture(t, staleRoot, rel, b)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	o.Apply = true
	plan, err = i18n.ReconcileTranslationUnitRepresentations(root, current, o)
	if err != nil || !plan.Applied {
		t.Fatal(plan, err)
	}
	if _, err = i18n.ReconcileTranslationUnitRepresentations(root, current, o); err == nil {
		t.Fatal("repeat apply accepted")
	}
	after, err := i18n.BuildTranslationUnitProgress(root, current, "zh-CN")
	if err != nil || after.Missing != 0 || after.Passed != 49 {
		t.Fatal(after, err)
	}
	successor, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, current, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-current-representation"})
	if err != nil {
		t.Fatal(err)
	}
	if successor.Units[5].SelectedBatchID != "chatgpt-zh-CN-047" || successor.Units[5].RepresentationReconciliation != nil {
		t.Fatal("047 provenance overwritten")
	}
	for _, u := range plan.Receipt.Units {
		found := false
		for _, v := range successor.Units {
			if v.UnitID == u.Replacement.CurrentUnitID {
				found = true
				if v.SelectedBatchID != u.Predecessor.SelectedBatchID || v.Attempt != u.Predecessor.Attempt || v.RepresentationReconciliation == nil {
					t.Fatal("fake attempt provenance")
				}
			}
		}
		if !found {
			t.Fatal("missing successor")
		}
	}
	scope, err := i18n.BuildQualityCheckScope(root, current, i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: successor.SnapshotID, PreviousSnapshotID: previous})
	if err != nil || scope.ACount != 48 || scope.PendingCount != 1 {
		t.Fatal(scope, err)
	}
	if scope.Pending[0].UnitID != currentIDs[5] {
		t.Fatal("revised B inherited A")
	}
	if _, _, err := i18n.ExportQualityCheckReviewerBundle(root, current, i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: successor.SnapshotID, PreviousSnapshotID: previous, StartIndex: 6, Limit: 1}); err != nil {
		t.Fatal(err)
	}
	first := plan.Receipt.Units[0]
	for name, path := range map[string]string{"current-raw": first.CurrentRaw.Path, "candidate": first.Candidate.Path, "validation": first.Validation.Path, "original-raw": first.OriginalRaw.Path, "old-validation": first.Predecessor.ValidationPath, "impact": "impact.json", "old-QC": "data/quality-check-snapshots/zh-CN/" + previous + "/quality-check-results.json"} {
		t.Run("tamper-"+name, func(t *testing.T) {
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, path, append(bytes.Clone(before), ' '))
			defer writeFixture(t, root, path, before)
			if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, current, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-tampered-" + name}); err == nil {
				t.Fatal("tampered machine evidence accepted")
			}
		})
	}
	// Introduce a genuine mandatory glossary delta affecting only the unchanged
	// Unit. Machine successors stay byte-exact; the unrelated stale Unit still
	// blocks a new Snapshot. This is fixture-only glossary authority, not recovery.
	// A synthetic independent re-QC supplies the only new language A. The
	// machine-carried units still name their original old-ID judgments.
	if _, err := i18n.RecordQualityCheckResults(root, current, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: successor.SnapshotID, PreviousSnapshotID: previous, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: currentIDs[5:6], Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	finalized, _, err := i18n.FinalizeQualityCheck(root, current, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: successor.SnapshotID, PreviousSnapshotID: previous})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.VerifyHistoricalQualityCheckFinalization(root, "zh-CN", successor.SnapshotID, current); err != nil {
		t.Fatal("machine A lineage not formally consumable", err)
	}
	// Surface artifacts retain complete package scope when their reconciliation
	// proof is replayed. Inventory-only or partial selection is not authority.
	bindSurface := func(c *i18n.Catalog) error {
		s := &IntegratedSurfaceScope{Locale: "zh-CN", Reviewed: []SurfaceAsset{}}
		for _, u := range finalized.Units {
			s.Reviewed = append(s.Reviewed, SurfaceAsset{ID: u.Snapshot.SourcePath, TargetSHA: u.Snapshot.CandidateSHA256})
		}
		return BindIntegratedSurfaceCandidates(root, c, s, finalized)
	}
	if err := bindSurface(current); err != nil {
		t.Fatal("complete reconciled package Surface binding", err)
	}
	for _, ids := range [][]string{nil, currentIDs[:1]} {
		partial := *current
		partial.TranslationScope = ids
		if err := bindSurface(&partial); err == nil || !strings.Contains(err.Error(), "complete explicit registered package scope required") {
			t.Fatal("incomplete Surface scope accepted or wrong failure", err)
		}
	}
	t.Run("surface-reconciliation-receipt-tamper", func(t *testing.T) {
		name := "data/translation-unit-reconciliations/zh-CN/machine-successors/receipt.json"
		before, err := readRegular(root, name)
		if err != nil {
			t.Fatal(err)
		}
		defer writeFixture(t, root, name, before)
		writeFixture(t, root, name, append(bytes.Clone(before), ' '))
		if err := bindSurface(current); err == nil {
			t.Fatal("Surface accepted tampered reconciliation")
		}
	})
	// A later explicitly authorized correction must receive the reconciled raw,
	// with its genuine original provenance and machine proof in the same Bundle.
	const laterFinding = "Correct this exact synthetic message."
	writeFixture(t, root, "data/locale-surface-reviews/zh-CN/later-fixture.md", []byte("decision = failed\nTranslationUnit "+currentIDs[6]+": "+laterFinding+"\n"))
	if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, current, i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "later-fixture", PreviousSnapshotID: successor.SnapshotID, SurfaceReviewID: "later-fixture", Finding: laterFinding, UnitIDs: currentIDs[6:7]}); err != nil {
		t.Fatal(err)
	}
	later, err := i18n.ExportRetranslationBatch(root, current, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: currentIDs[6:7], AllowReexport: true, PreviousSnapshotID: successor.SnapshotID, SurfaceReopenID: "later-fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, current, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: later.BatchID}); err != nil {
		t.Fatal("machine successor lost its legitimate revision transport", err)
	}
	writeFixture(t, staleRoot, "locales/zh-CN/glossary.yaml", []byte("locale: zh-CN\nmandatory:\n  synthetic-test-term: 测试术语\nforbidden:\n  - GlossaryTerm\n"))
	migrationReview(t, staleRoot, "synthetic-one-unit-glossary-delta", "passed")
	if _, _, err := i18n.AssessGlossaryCompatibility(staleRoot, "zh-CN", snapshot.GlossarySHA256, old); err != nil {
		t.Fatal(err)
	}
	staleBefore, err := i18n.BuildTranslationUnitProgress(staleRoot, current, "zh-CN")
	if err != nil || staleBefore.Passed != 5 || staleBefore.Missing != 43 || staleBefore.Stale != 1 {
		t.Fatal("wrong pre-existing stale fixture", staleBefore, err)
	}
	if p, err := i18n.ReconcileTranslationUnitRepresentations(staleRoot, current, o); err != nil || !p.Applied {
		t.Fatal("unrelated stale Unit blocked exact machine scope", p, err)
	}
	stale, err := i18n.BuildTranslationUnitProgress(staleRoot, current, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if stale.Missing != 0 || stale.Stale != 1 || stale.Passed != 48 {
		t.Fatal("glossary stale was bypassed", stale)
	}
	if stale.Units[0].Status != "stale" {
		t.Fatal("wrong glossary-stale Unit")
	}
	_, _, err = i18n.CreateQualityCheckCandidateSnapshot(staleRoot, current, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-glossary-blocked"})
	if err == nil {
		t.Fatal("glossary stale Snapshot admitted")
	}
	t.Log("expected glossary-stale Snapshot blocker:", err)
	if _, err := os.Stat(filepath.Join(staleRoot, "data/quality-check-snapshots/zh-CN/qc-glossary-blocked")); !os.IsNotExist(err) {
		t.Fatal("failed Snapshot mutated")
	}
	got, _ := readRegular(root, manifestPath)
	if !bytes.Equal(oldManifest, got) {
		t.Fatal("old Snapshot changed")
	}
}

func TestRepresentationNewAtomsDeterministicProof(t *testing.T) {
	for _, tc := range []struct {
		name, source, target string
		pass                 bool
	}{
		{"unique", "Return 404 or 410.", "返回 404 或 410。", true},
		{"missing", "Custom Marshal and Unmarshal", "自定义编组与解组", false},
		{"ambiguous", "Package A depends on package B.", "包 A 依赖包 B；A 与 B 有不同记录。", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("# Test\n\n" + tc.source + "\n")
			d, err := ParseWorkflowDocument(Source{Path: "_content/doc/proof.md", Kind: "page", Route: "/doc/proof", SHA256: digest(source)}, source, 1)
			if err != nil {
				t.Fatal(err)
			}
			// Freeze a real older family: prose-only atoms before the semantic policy.
			old := *d
			old.Slots = append([]Unit(nil), d.Slots...)
			for i, s := range old.Slots {
				if s.Start <= strings.Index(string(source), tc.source) && s.End >= strings.Index(string(source), tc.source)+len(tc.source) {
					s.Source = tc.source
					s.Protected = nil
					s.SourceSHA = unitSourceIdentity(s.Source, nil)
					old.Slots[i] = s
				}
			}
			old.Units = old.Slots
			old, err = FormalPage(old, source, 1)
			if err != nil {
				t.Fatal(err)
			}
			a, b := PageAdapter{*d, source}, PageAdapter{old, source}
			raw := []byte(strings.Replace(old.Units[0].Source, tc.source, tc.target, 1))
			next, err := a.ReconcileRepresentation(b, raw)
			if !tc.pass {
				if err == nil {
					t.Fatal("ambiguous/missing machine identity accepted")
				}
				if tc.name == "ambiguous" {
					if _, err := a.ReconciliationQuestion(b, raw); err == nil {
						t.Fatal("nonliteral semantic authority could guess repeated binding sites")
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			before, _ := b.Restore(string(raw))
			after, err := a.Restore(string(next))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("reviewed bytes changed", err)
			}
		})
	}
}
