package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestScopedSuccessorFinalizationKeepsHistoricalWorld(t *testing.T) {
	root, catalog, _, snapshot := checkpointFixture(t)
	if _, err := InitializeLocaleStatus(root, "zh-CN", catalog); err != nil {
		t.Fatal(err)
	}
	_, err := PromoteRetranslation(root, catalog, RetranslationPromoteOptions{Locale: "zh-CN", SnapshotID: snapshot, Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(root, "locales/zh-CN/candidates/.promotion-*.json"))
	if err != nil || len(paths) != 1 {
		t.Fatal("native target successor missing", paths, err)
	}
	path, _ := filepath.Rel(root, paths[0])
	path = filepath.ToSlash(path)
	baseline, err := verifyTargetSuccessor(root, path, catalog)
	if err != nil {
		t.Fatal(err)
	}
	receipt := baseline.evidence
	historyBefore, err := ReadHistoricalTranslationIntegrity(root, catalog, "zh-CN", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	checkpointGlossary(t, root, "\n# next compatibility world\n", "scoped-world-next")
	historyAfter, err := ReadHistoricalTranslationIntegrity(root, catalog, "zh-CN", snapshot)
	if err != nil || !bytes.Equal(mustJSON(historyBefore), mustJSON(historyAfter)) {
		t.Fatal("frozen integrity result changed across glossary worlds", err)
	}
	scoped := *catalog
	// The event was valid in its original current glossary world. A later
	// glossary must not change its bytes or make its scoped reader re-enter
	// current compatibility while rebuilding that frozen event.
	got, err := verifyTargetSuccessor(root, path, &scoped)
	if err != nil || !bytes.Equal(mustJSON(got.evidence), mustJSON(baseline.evidence)) {
		t.Fatalf("scoped event changed or re-entered current validation: %v", err)
	}
	if !got.compatibility.disableSuccessor || !got.compatibility.tuOnly || got.compatibility.historicalSHA != receipt.GlossarySHA {
		t.Fatal("nested scoped reader lost the explicit frozen world")
	}
	if got.compatibility == baseline.compatibility {
		t.Fatal("independent invocation shared a compatibility world")
	}
	// Scope remains exact; the world is no permission to accept foreign Units.
	wrong := scoped
	wrong.TranslationScope = wrong.TranslationScope[:len(wrong.TranslationScope)-1]
	if _, err := verifyTargetSuccessor(root, path, &wrong); err == nil {
		t.Fatal("wrong selected scope accepted")
	}
	before, err := readCompatibilityFile(root, receipt.Finalization.Path)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, receipt.Finalization.Path, append(bytes.Clone(before), ' '))
	if _, err := verifyTargetSuccessor(root, path, &scoped); err == nil {
		t.Fatal("new invocation accepted changed finalization bytes")
	}
}

func TestMachineSnapshotHistoricalWorldIsolation(t *testing.T) {
	root, c, _, plan, ref, _ := authorizationMachineFixture(t)
	selection := &machineSelection{receipt: plan.Receipt, unit: plan.Receipt.Units[0], ref: ref, ctx: &machineReconciliationContext{receipt: plan.Receipt}}
	u := selection.snapshot(1)
	before, _, err := machineSnapshotEvidence(root, c, "zh-CN", u, nil)
	if err != nil {
		t.Fatal(err)
	}
	// A real current glossary constraint affects this Unit, but cannot rewrite
	// its immutable representation transition. No target successor is guessed.
	checkpointGlossary(t, root, "forbidden:\n  - message\n", "machine-world-next")
	contexts := map[string]*machineSelection{}
	after, selected, err := machineSnapshotEvidenceInWorld(root, c, "zh-CN", u, contexts, true)
	if err != nil || !bytes.Equal(before.candidate, after.candidate) || !reflect.DeepEqual(selected.unit, selection.unit) {
		t.Fatalf("historical machine bytes/identity changed: %v", err)
	}
	world := selected.ctx
	if !world.historical || world.currentGlossarySHA != plan.Receipt.GlossarySHA256 || world.compatibility.historicalSHA != plan.Receipt.GlossarySHA256 {
		t.Fatal("receipt world not pinned")
	}
	if _, again, err := machineSnapshotEvidenceInWorld(root, c, "zh-CN", u, contexts, true); err != nil || again.ctx != world || again.ctx.compatibility != world.compatibility {
		t.Fatal("nested machine check created another world", err)
	}
	_, independent, err := machineSnapshotEvidenceInWorld(root, c, "zh-CN", u, nil, true)
	if err != nil || independent.ctx == world || independent.ctx.compatibility == world.compatibility {
		t.Fatal("independent invocation shared world", err)
	}
	if _, _, err := machineSnapshotEvidence(root, c, "zh-CN", u, nil); err == nil {
		t.Fatal("historical world authorized affected current validation")
	}
	data, err := os.ReadFile(filepath.Join(root, selected.unit.Candidate.Path))
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, selected.unit.Candidate.Path, append(data, ' '))
	if _, _, err := machineSnapshotEvidenceInWorld(root, c, "zh-CN", u, nil, true); err == nil {
		t.Fatal("new historical invocation accepted candidate tamper")
	}
}

func TestScopedSuccessorFinalizationRejectsCycle(t *testing.T) {
	root, catalog, _, snapshot := checkpointFixture(t)
	s, err := readQualityCheckSnapshot(root, "zh-CN", snapshot, false)
	if err != nil {
		t.Fatal(err)
	}
	r, err := readQualityCheckResults(root, "zh-CN", s)
	if err != nil {
		t.Fatal(err)
	}
	r.PreviousSnapshotID = r.SnapshotID
	resultBytes := mustJSON(r)
	writeSnapshotSource(t, root, filepath.ToSlash(qualityCheckResultsPath("", "zh-CN", s.SnapshotID)), resultBytes)
	finalPath := filepath.ToSlash(qualityCheckFinalizationPath("", "zh-CN", s.SnapshotID))
	finalBytes, err := readCompatibilityFile(root, finalPath)
	if err != nil {
		t.Fatal(err)
	}
	var final QualityCheckFinalization
	if err := decodeStrictCourseSourceDescriptionReviewJSON(finalBytes, &final); err != nil {
		t.Fatal(err)
	}
	for i := range final.QCResults {
		if final.QCResults[i].SnapshotID == s.SnapshotID {
			final.QCResults[i].SHA256 = sum(resultBytes)
		}
	}
	writeSnapshotSource(t, root, finalPath, mustJSON(final))
	p, _ := historicalSuccessorCompatibility(root, "zh-CN", catalog, nil)
	if _, err := verifySuccessorFinalization(root, "zh-CN", s.SnapshotID, catalog, p); err == nil || !(strings.Contains(err.Error(), "cycle") || strings.Contains(err.Error(), "invalid previous_snapshot_id")) {
		t.Fatal("cycle was not checked through its rebound evidence", err)
	}
}
