package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestReconciledCurrentAssessmentMembershipAppend(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, event, original, plan, ref, receiptBytes := authorizationMachineFixtureSize(t, 49)
	resolution, err := event.ReplacementResolver(root, event, plan.Receipt.Options.HistoricalDocuments)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, resolution.HistoricalCatalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: original.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	accepted := futureSnapshot(t, root, event, "reconciled-current-assessment")
	if _, _, err := FinalizeQualityCheck(root, event, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: accepted.SnapshotID, PreviousSnapshotID: original.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	recordedScopeCompatibilityInputs(t, root)
	checkpointGlossary(t, root, "preferred:\n  AbsentFixtureGlossaryTerm: 固定术语\n", "current-assessment-append-review")
	current, added := expandedMachinePackage(t, root, event, 2)
	addition, err := HistoricalTranslationPackageCatalog(current, added.ID, added.AuthoritySHA256)
	if err != nil {
		t.Fatal(err)
	}
	for _, unit := range added.Units {
		checkpointGenerate(t, root, addition, []string{unit.Unit.ID})
	}
	// A valid edge with the right endpoints but the wrong TU inventory is NOT
	// our authority. The legacy resolver must continue refusing that request.
	_, edgePath, err := AssessGlossaryCompatibility(root, "zh-CN", plan.Receipt.GlossarySHA256, addition)
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := readCompatibilityFile(root, "locales/zh-CN/glossary.yaml")
	if err != nil {
		t.Fatal(err)
	}
	member := plan.Receipt.Units[0]
	legacy := PrepareGlossaryCompatibility(root, "zh-CN", resolution.HistoricalCatalog).Resolve(plan.Receipt.GlossarySHA256, sum(glossary), "tu:"+member.Predecessor.UnitID)
	if legacy.Status == "compatible" || legacy.Status == "exact" {
		t.Fatal("foreign edge widened to historical TU", legacy)
	}
	a, err := AssessCurrentTranslation(root, event, "zh-CN", accepted.SnapshotID)
	if err != nil || !a.Ready() || a.Compatible != 49 {
		t.Fatal("canonical complete 49-Unit assessment", a, err)
	}
	selection := &machineSelection{receipt: plan.Receipt, unit: member, ref: ref, ctx: &machineReconciliationContext{receipt: plan.Receipt}}
	u := selection.snapshot(1)
	if _, _, err := machineSnapshotEvidence(root, current, "zh-CN", u, nil); err != nil {
		t.Fatal("49 historical A / 51 current scope must use current assessment", err)
	}
	if a, err := AssessCurrentTranslation(root, current, "zh-CN", accepted.SnapshotID); err == nil && a.Ready() {
		t.Fatal("individual eligibility authorized expanded complete package")
	}
	if err := os.Remove(filepath.Join(root, edgePath)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := machineSnapshotEvidence(root, current, "zh-CN", u, nil); err != nil {
		t.Fatal("canonical current assessment depended on legacy edge", err)
	}
	if _, err := os.Stat(filepath.Join(root, "data/translation-authorizations")); !os.IsNotExist(err) {
		t.Fatal("readonly assessment wrote an authorization", err)
	}
	// A genuine terminology impact is rejected even if current mechanical
	// validation can restore the unchanged candidate successfully.
	checkpointGlossary(t, root, "  message: 消息\n", "current-assessment-append-impact")
	if _, _, err := machineSnapshotEvidence(root, current, "zh-CN", u, nil); err == nil || !strings.Contains(err.Error(), "affected") || !strings.Contains(err.Error(), "preferred_source_term_present") {
		t.Fatal("true semantic impact lost", err)
	}
	after, err := readSnapshotReferencedFile(root, ref.Path)
	if err != nil || !bytes.Equal(after, receiptBytes) {
		t.Fatal("historical receipt changed", err)
	}
	t.Log("historical=49 current=51; legacy foreign-scope edge remains invalid; canonical assessment compatible=49; semantic impact rejected; no partial authorization")
}

func TestReconciliationCurrentInputPhaseGuards(t *testing.T) {
	root, c, _ := makeRetranslationReviewBatchFixture(t, 1, "input-phase")
	input := &reconciliationCurrentInputs{}
	if err := input.prepare(root, "zh-CN"); err != nil {
		t.Fatal(err)
	}
	first := input.glossary
	if err := input.prepare(root, "zh-CN"); err != nil || input.glossary != first {
		t.Fatal("common inputs rebuilt", err)
	}
	if err := input.prepare(t.TempDir(), "zh-CN"); err == nil {
		t.Fatal("input scope crossed roots")
	}
	if err := input.prepare(root, "en-GB"); err == nil {
		t.Fatal("input scope crossed locales")
	}
	if err := input.check(); err != nil {
		t.Fatal(err)
	}
	reviewData, err := readCompatibilityFile(root, input.review.Path)
	if err != nil {
		t.Fatal(err)
	}
	var review UnifiedGlossaryReview
	if err := decodeStrictCourseSourceDescriptionReviewJSON(reviewData, &review); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{glossaryReviewGlossaryPath("zh-CN"), input.review.Path, unifiedReviewerArchive("zh-CN", review.BundleSHA), GlossaryCorpusPath} {
		t.Run(path, func(t *testing.T) {
			before, err := readCompatibilityFile(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeSnapshotSource(t, root, path, before)
			writeSnapshotSource(t, root, path, append(bytes.Clone(before), 'x'))
			if err := input.check(); err == nil {
				t.Fatal("phase accepted changed language inputs", path)
			}
		})
	}
	eval := newQualityCheckEvaluation(root, "zh-CN", c)
	if _, _, err := eval.loadEffectiveQualityCheckResults("input-phase", map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if eval.reconciliationInputs != nil {
		t.Fatal("input phase escaped success")
	}
	if _, _, err := eval.loadEffectiveQualityCheckResults("missing-phase", map[string]bool{}); err == nil {
		t.Fatal("missing Snapshot accepted")
	}
	if eval.reconciliationInputs != nil {
		t.Fatal("input phase escaped failure")
	}
}
