package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrdinaryRevisionStillRequiresCompleteInitialQC(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 2, "qc-incomplete")
	if err := catalog.UseTranslationUnitScope([]string{"lesson/1", "lesson/2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordQualityCheckResults(root, catalog, QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "qc-incomplete", ReviewerSession: "synthetic-independent-reviewer", UnitIDs: []string{"lesson/1"}, Rating: "B", Finding: "Correct this message."}); err != nil {
		t.Fatal(err)
	}
	_, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", BatchID: "codex-zh-CN-047", UnitIDs: []string{"lesson/1"}, AllowReexport: true, PreviousSnapshotID: "qc-incomplete"})
	if err == nil || !strings.Contains(err.Error(), "首轮独立质量审核尚未覆盖全部翻译单元") {
		t.Fatalf("ordinary incomplete QC gate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "data/retranslation-runs/zh-CN/codex-zh-CN-047")); !os.IsNotExist(err) {
		t.Fatal("blocked revision created batch", err)
	}
}
