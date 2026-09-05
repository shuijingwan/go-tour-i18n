package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQualityCheckFinalizationRequiresAndBindsCompleteAOnlySnapshot(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 2, "qc-final")
	recordQualityCheckRatings(t, root, catalog, "qc-final", "", "A", []string{"lesson/1", "lesson/2"})
	finalization, path, err := FinalizeQualityCheck(root, catalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-final"})
	if err != nil || len(finalization.Units) != 2 || len(finalization.QCResults) != 1 || path == "" {
		t.Fatalf("finalization=%+v path=%q err=%v", finalization, path, err)
	}
	if _, err := VerifyQualityCheckFinalization(root, catalog, "zh-CN", "qc-final"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, catalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-final"}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("overwrite error=%v", err)
	}
	resultPath := qualityCheckResultsPath(root, "zh-CN", "qc-final")
	b, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, append(b, ' '), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyQualityCheckFinalization(root, catalog, "zh-CN", "qc-final"); err == nil {
		t.Fatal("accepted mutated QC evidence")
	}
}

func TestQualityCheckFinalizationRejectsPendingAndMalformedEvidence(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 2, "qc-pending")
	recordQualityCheckRatings(t, root, catalog, "qc-pending", "", "A", []string{"lesson/1"})
	if _, _, err := FinalizeQualityCheck(root, catalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-pending"}); err == nil || !strings.Contains(err.Error(), "A-only") {
		t.Fatalf("pending error=%v", err)
	}
	p := filepath.Join(root, "data", "quality-check-snapshots", "zh-CN", "qc-pending", "finalization.json")
	if err := os.WriteFile(p, []byte("{bad"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyQualityCheckFinalization(root, catalog, "zh-CN", "qc-pending"); err == nil {
		t.Fatal("accepted malformed finalization")
	}
}
