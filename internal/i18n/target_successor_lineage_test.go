package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Keep an earlier changed-only recovery in the same inventory as a later
// native receipt. The second revision changes a different Unit, so matching
// just the first recovery's transition targets cannot establish currentness.
func nativeMultiEdgeLineageFixture(t *testing.T) (string, *Catalog, *TargetSuccessorEvidence, string, map[string][]byte) {
	t.Helper()
	root, cat, _ := historicalRecoveryFixture(t, func(c *Catalog) {
		c.Pages[1].Source = append(c.Pages[1].Source, []byte("Iteration details.\n")...)
		c.Pages[1].SourceSHA256 = sum(c.Pages[1].Source)
	})
	oldReceipt, oldPath, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(oldReceipt.Units) != 1 || oldReceipt.Units[0].UnitID != "lesson/1" {
		t.Fatal("fixture must have one earlier changed transition")
	}
	oldGlossary := oldReceipt.GlossarySHA
	next := strings.Replace(compatibilityFixtureGlossary, "  Go: Go\n", "  Go: Go\n  Iteration: 迭代\n", 1)
	compatibilityChange(t, root, next, "next-iteration")
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", oldGlossary, cat); err != nil {
		t.Fatal(err)
	}
	batch, err := ExportRetranslationBatch(root, cat, RetranslationExportOptions{Locale: "zh-CN", BatchID: "codex-zh-CN-003", UnitIDs: []string{"lesson/2"}, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	m := readRetranslationManifest(t, root, batch.BatchID)
	input, err := readCompatibilityFile(root, batch.BatchPath+"/"+m.Units[0].InputPath)
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.NewReplacer("* Page", "* 修订第二页", "Use ", "本页使用 ", " on this page.", "。", "Iteration", "迭代").Replace(string(input))
	writeSnapshotSource(t, root, batch.BatchPath+"/raw-responses/lesson-2.article", []byte(raw))
	if _, err := ProcessRetranslationBatch(root, cat, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch.BatchID}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", oldGlossary, cat); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, cat, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-003"}); err != nil {
		t.Fatal(err)
	}
	recordQualityCheckRatings(t, root, cat, "qc-003", "qc-002", "A", []string{"lesson/2"})
	f, _, err := FinalizeQualityCheck(root, cat, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-003", PreviousSnapshotID: "qc-002"})
	if err != nil {
		t.Fatal(err)
	}
	protected := map[string][]byte{}
	for _, path := range []string{oldPath, "data/quality-check-snapshots/zh-CN/qc-001/finalization.json", "data/quality-check-snapshots/zh-CN/qc-002/finalization.json", "data/quality-check-snapshots/zh-CN/qc-003/finalization.json", "data/quality-check-snapshots/zh-CN/qc-003/quality-check-results.json"} {
		protected[path], err = readCompatibilityFile(root, path)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range f.Units {
		for _, ref := range u.CompatibilityChain {
			protected[ref.Path], err = readCompatibilityFile(root, ref.Path)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(f.Units[2].CompatibilityChain) < 2 {
		t.Fatal("fixture must carry A through multiple glossary edges")
	}
	before, err := readCompatibilityFile(root, canonicalCandidatePath("zh-CN", "lesson/2"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PromoteRetranslation(root, cat, RetranslationPromoteOptions{Locale: "zh-CN", SnapshotID: "qc-003", Apply: true}); err != nil {
		t.Fatal(err)
	}
	after, err := readCompatibilityFile(root, canonicalCandidatePath("zh-CN", "lesson/2"))
	if err != nil || bytes.Equal(before, after) {
		t.Fatal("fixture promotion did not change canonical target", err)
	}
	// A later full-world assessment must not replace any frozen chain refs.
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", oldGlossary, cat); err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(root, "locales/zh-CN/candidates/.promotion-*.json"))
	if err != nil || len(paths) != 1 {
		t.Fatal("fixture missing native receipt", err)
	}
	path, err := filepath.Rel(root, paths[0])
	if err != nil {
		t.Fatal(err)
	}
	path = filepath.ToSlash(path)
	proof, err := verifyTargetSuccessor(root, path, cat)
	if err != nil {
		t.Fatal("native event lost its immutable multi-edge A-only lineage", err)
	}
	protected[path], err = readCompatibilityFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return root, cat, proof.evidence, path, protected
}

func TestNativePromotionHistoricalMultiEdgeLineage(t *testing.T) {
	root, cat, receipt, path, protected := nativeMultiEdgeLineageFixture(t)
	scope, err := BuildQualityCheckScope(root, cat, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-003"})
	if err != nil || scope.CurrentResultCount != 1 || scope.CarryForwardCount != 2 || scope.ACount != 3 || !scope.ReadyForFinalization {
		t.Fatalf("legitimate carry lost: %+v %v", scope, err)
	}
	d, err := DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-003")
	if err != nil || !d.CanonicalAuthorized || d.ReviewerACount != 3 || d.SuccessorKind != "native-promotion-receipt" {
		t.Fatalf("native authorization failed: %+v %v", d, err)
	}
	changed := []string{}
	for _, u := range receipt.Units {
		if u.Changed {
			changed = append(changed, u.UnitID)
		}
	}
	if len(changed) != 1 || changed[0] != "lesson/2" {
		t.Fatal("native changed-unit exact-set differs", changed)
	}
	ui := "internal/tour/ui/zh-CN.json"
	original, err := readCompatibilityFile(root, ui)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, ui, append(bytes.Clone(original), '\n'))
	proof, err := verifyTargetSuccessor(root, path, cat)
	if err != nil {
		t.Fatal("non-TU context revoked historical TU event", err)
	}
	if err := requireSuccessorCurrentLanguage(root, cat, proof); err != nil {
		t.Fatal("non-TU context revoked current TU authorization", err)
	}
	oldGlossary := readSnapshotGlossarySHA(t, root, "qc-002")
	if r := proof.compatibility.Resolve(oldGlossary, receipt.GlossarySHA, "tu:lesson/2"); r.Status != "affected" {
		t.Fatal("historical rebinding bypassed the affected TU classification", r)
	}
	for _, consumer := range []string{"ui:test.one", "seo:lesson/2", "seo-page:lesson/2", "*"} {
		if r := proof.compatibility.Resolve(receipt.GlossarySHA, receipt.GlossarySHA, consumer); r.Status == "exact" || r.Status == "compatible" {
			t.Fatal("TU history authorized derived consumer", consumer)
		}
	}
	if err := AuthorizeHistoricalCanonicalReference(root, "zh-CN", ui, sum(original), cat); err == nil {
		t.Fatal("TU history authorized stale UI")
	}
	if r := ResolveGlossaryCompatibility(root, "zh-CN", oldGlossary, receipt.GlossarySHA, "ui:test.one", cat); r.Status == "compatible" {
		t.Fatal("ordinary derived-consumer resolver accepted stale non-TU inventory")
	}
	for file, before := range protected {
		after, err := readCompatibilityFile(root, file)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("immutable evidence changed", file, err)
		}
	}
}

func TestNativePromotionHistoricalMultiEdgeLineageFailClosed(t *testing.T) {
	root, cat, receipt, path, _ := nativeMultiEdgeLineageFixture(t)
	f, err := readCompatibilityFile(root, receipt.Finalization.Path)
	if err != nil {
		t.Fatal(err)
	}
	var finalization QualityCheckFinalization
	if err := decodeStrictCourseSourceDescriptionReviewJSON(f, &finalization); err != nil {
		t.Fatal(err)
	}
	inputs := []string{receipt.Finalization.Path, finalization.Units[2].CompatibilityChain[0].Path, receipt.Units[1].Candidate.Path, finalization.Units[2].CompatibilityChain[1].Path}
	priorBytes, err := readCompatibilityFile(root, "data/quality-check-snapshots/zh-CN/qc-002/finalization.json")
	if err != nil {
		t.Fatal(err)
	}
	var prior QualityCheckFinalization
	if err := decodeStrictCourseSourceDescriptionReviewJSON(priorBytes, &prior); err != nil {
		t.Fatal(err)
	}
	var priorEdge GlossaryCompatibilityEvidence
	b, err := readCompatibilityFile(root, prior.Units[2].CompatibilityChain[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &priorEdge); err != nil {
		t.Fatal(err)
	}
	for _, c := range priorEdge.Contexts {
		if c.Scope == "tu:lesson/2" {
			for _, ref := range c.References {
				if strings.HasPrefix(ref.Path, "data/retranslation-runs/") {
					inputs = append(inputs, ref.Path)
				}
			}
		}
	}
	for _, input := range inputs {
		for _, mutation := range []string{"tamper", "missing"} {
			t.Run(mutation+"/"+input, func(t *testing.T) {
				before, err := readCompatibilityFile(root, input)
				if err != nil {
					t.Fatal(err)
				}
				defer writeSnapshotSource(t, root, input, before)
				if mutation == "missing" {
					if err := os.Remove(filepath.Join(root, input)); err != nil {
						t.Fatal(err)
					}
				} else {
					writeSnapshotSource(t, root, input, append(bytes.Clone(before), '\n'))
				}
				if _, err := verifyTargetSuccessor(root, path, cat); err == nil {
					t.Fatal("tampered immutable TU evidence accepted", input)
				}
				if d, err := DiagnoseQualityCheckAuthorization(root, cat, "zh-CN", "qc-003"); err == nil && d.CanonicalAuthorized {
					t.Fatal("tampered immutable TU evidence authorized canonical", input)
				}
			})
		}
	}
}
