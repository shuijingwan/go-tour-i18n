package i18n

import "testing"

func TestRepresentationHistoricalAIsSeparateFromCurrentGlossary(t *testing.T) {
	root, c, ids, snapshot := checkpointFixture(t)
	if _, _, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot); err != nil {
		t.Fatal(err)
	}
	checkpointGlossary(t, root, "forbidden:\n  - apple\n", "machine-historical-review")
	normal := newQualityCheckEvaluation(root, "zh-CN", c)
	_, current, err := normal.loadEffectiveQualityCheckResults(snapshot, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := current[ids[0]]; ok {
		t.Fatal("current glossary-invalid Unit carried A")
	}
	original := newQualityCheckEvaluation(root, "zh-CN", c)
	original.originalOnly = true
	_, historical, err := original.loadEffectiveQualityCheckResults(snapshot, map[string]bool{})
	if err != nil || len(historical) != len(ids) {
		t.Fatal("lost trusted historical judgments", historical, err)
	}
	if historical[ids[0]].rating != "A" {
		t.Fatal("historical A rewritten")
	}
	a, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot)
	if err != nil || a.Affected != 1 || a.Compatible != len(ids)-1 {
		t.Fatal("current gate bypassed", a, err)
	}
}
