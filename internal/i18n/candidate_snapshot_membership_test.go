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
