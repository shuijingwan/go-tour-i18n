package i18n

import (
	"strings"
	"testing"
)

// Frozen scope projection is confined to the Snapshot evidence consumer. The
// ordinary current/historical manifest reader never admits arbitrary partials.
func TestSnapshotMembershipDoesNotRelaxCompleteReader(t *testing.T) {
	root, old, _ := packageSnapshotFixture(t)
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, old, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "legacy-complete"}); err != nil {
		t.Fatal(err)
	}
	_, inventory, _ := futureContentFixture(t)
	d := inventory.TranslationPackages[0]
	d.ID = old.TranslationPackages[0].ID
	d.AuthoritySHA256 = sum([]byte("uncomposed-current-source"))
	d.Units = d.Units[:2]
	current := &Catalog{}
	if err := current.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	current.HistoricalTranslationPackages = old.TranslationPackages
	if err := current.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	for _, today := range []bool{false, true} {
		if _, err := readQualityCheckSnapshot(root, "zh-CN", "legacy-complete", today, current); err == nil || !strings.Contains(err.Error(), "两批页面生成及机器校验全部完成前禁止独立质量审核") {
			t.Fatal("ordinary complete-scope invariant weakened", today, err)
		}
	}
	if members, guard, err := snapshotMembershipCatalogs(root, current); err != nil || len(members) != 0 || guard != nil {
		t.Fatal("uncomposed package acquired membership projection", err)
	}
}

func TestSnapshotMembershipCarryAssessmentClassification(t *testing.T) {
	root, c, ids, snapshot := checkpointFixture(t)
	assessment, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot)
	if err != nil || !assessment.Ready() {
		t.Fatal(assessment, err)
	}
	latest, err := selectLatestRetranslationUnits(root, c, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	g, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	current := *c
	u, err := c.Unit(ids[0])
	if err != nil {
		t.Fatal(err)
	}
	members := map[string]*Catalog{ids[0]: c}
	for _, tc := range []struct {
		name, class, validation string
		pass                    bool
	}{
		{"compatible", "compatible", "passed", true},
		{"affected", "affected", "passed", false},
		{"stale", "stale", "not_run", false},
		{"unknown", "unknown", "not_run", false},
		{"validation-failed", "compatible", "failed", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			local := *assessment
			local.Units = append([]CurrentTranslationAssessmentUnit(nil), assessment.Units...)
			local.Units[0].Classification = tc.class
			local.Units[0].Validation = tc.validation
			local.Identity = authorizationIdentity(local)
			_, err := validateSnapshotMembershipCandidate(root, &current, "zh-CN", g, u, latest.selectedByID[u.ID], nil, members, map[*Catalog]*CurrentTranslationAuthorization{c: &local}, nil)
			if tc.pass && err != nil {
				t.Fatal(err)
			}
			if !tc.pass && (err == nil || !strings.Contains(err.Error(), "carry ineligible")) {
				t.Fatalf("carry must reject per-Unit assessment: %v", err)
			}
		})
	}
}

func TestSnapshotMembershipAssessmentCommitRechecksAffected(t *testing.T) {
	root, c, ids, snapshot := checkpointFixture(t)
	checkpointGlossary(t, root, "preferred:\n  apple: 苹果\n", "member-affected-review")
	latest, err := selectLatestRetranslationUnits(root, c, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	members := map[string]*Catalog{}
	for _, id := range ids {
		members[id] = c
	}
	assessments, check, err := snapshotMembershipAssessments(root, "zh-CN", members, latest, map[*Catalog]string{c: snapshot})
	if err != nil {
		t.Fatal("affected predecessor is assessment evidence, not whole authorization", err)
	}
	if a := assessments[c]; a == nil || a.Affected != 1 || a.Compatible != 2 || a.Ready() {
		t.Fatal("expected non-Ready complete assessment", a)
	}
	if err := check(); err != nil {
		t.Fatal("unchanged non-Ready assessment must recheck", err)
	}
	checkpointGlossary(t, root, "# changed after preflight\n", "member-commit-review")
	if err := check(); err == nil {
		t.Fatal("commit accepted changed current assessment")
	}
}
