package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCurrentValidationProjectionBoundHistoricalProof(t *testing.T) {
	root, c, options, _ := currentValidationFixture(t)
	r, _, err := InspectCurrentValidationReopen(root, c, options)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyCurrentValidationReopen(root, c, r, false); err != nil {
		t.Fatal(err)
	}
	assessment, err := decodeCurrentValidationAssessment(r.CurrentValidation)
	if err != nil || assessment.Ready() || assessment.Affected != 9 {
		t.Fatal("original A must not become current authorization", err)
	}
	// Differential comparison with the unchanged canonical historical reader.
	eager := currentValidationPredecessorEvaluation(root, c, options.Locale)
	wantSnapshot, wantResults, err := eager.loadEffectiveQualityCheckResults(options.PreviousSnapshotID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	reused := currentValidationPredecessorEvaluation(root, c, options.Locale)
	reused.stats = &qualityCheckEvaluationStats{}
	if err := reused.materializeOriginalFinalization(&assessment.Historical); err != nil {
		t.Fatal(err)
	}
	gotSnapshot, gotResults, err := reused.loadEffectiveQualityCheckResults(options.PreviousSnapshotID, map[string]bool{})
	if err != nil || !reflect.DeepEqual(wantSnapshot, gotSnapshot) || !reflect.DeepEqual(wantResults, gotResults) || reused.stats.EffectiveCacheHits != 1 {
		t.Fatal("bound proof changed original-A materialization", err)
	}
	if len(gotResults) != 49 || assessment.Ready() {
		t.Fatal("historical A reused as current PASS")
	}
	for _, mutate := range []func(*HistoricalFinalizationIntegrity){
		func(p *HistoricalFinalizationIntegrity) { p.Finalization.Units = p.Finalization.Units[:48] },
		func(p *HistoricalFinalizationIntegrity) { p.Finalization.Units[0].Rating = "B" },
		func(p *HistoricalFinalizationIntegrity) { p.Finalization.Locale = "fr-FR" },
		func(p *HistoricalFinalizationIntegrity) { p.Finalization.Units[0].Snapshot.Attempt++ },
	} {
		var bad HistoricalFinalizationIntegrity
		if err := decodeStrictCourseSourceDescriptionReviewJSON(mustJSON(assessment.Historical), &bad); err != nil {
			t.Fatal(err)
		}
		mutate(&bad)
		if err := currentValidationPredecessorEvaluation(root, c, options.Locale).materializeOriginalFinalization(&bad); err == nil {
			t.Fatal("invalid original-A scope/identity accepted")
		}
	}
	current := newQualityCheckEvaluation(root, options.Locale, c)
	if err := current.materializeOriginalFinalization(&assessment.Historical); err == nil {
		t.Fatal("current evaluation accepted historical-only materialization")
	}
	// Materialization is invocation-local. The existing final guard still
	// verifies ALL frozen refs, including unselected QC evidence. Preserve size
	// and mtime to prove that metadata alone cannot authorize the reuse.
	path := assessment.Historical.Finalization.QCResults[0].Path
	full := filepath.Join(root, path)
	before, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(full)
	if err != nil {
		t.Fatal(err)
	}
	bad := bytes.Clone(before)
	bad[len(bad)/2] ^= 1
	writeSnapshotSource(t, root, path, bad)
	if err := os.Chtimes(full, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := verifyCurrentValidationReopen(root, c, r, false); err == nil {
		t.Fatal("final frozen authority recheck lost same-size/mtime QC tamper")
	}
	writeSnapshotSource(t, root, path, before)
	if err := verifyCurrentValidationReopen(root, c, r, false); err != nil {
		t.Fatal("unchanged frozen authority rejected", err)
	}
	if len(currentValidationPredecessorEvaluation(root, c, options.Locale).effective) != 0 {
		t.Fatal("materialization survived into another invocation")
	}
	t.Log("exact original-A differential PASS; current assessment remains not Ready; original-only scope and fresh all-reference final guard retained")
}
