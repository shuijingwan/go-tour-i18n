package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Actual receipt read only. No export/record/apply is called on repository state.
func TestQCRepresentationExportReadonlyProbe(t *testing.T) {
	if os.Getenv("GO_TOUR_QC_RECONCILIATION_PROBE") != "1" {
		t.Skip("explicit real-state readonly probe required")
	}
	root := repositoryRoot(t)
	done := contentidentity.BeginMemoization()
	defer done()
	const receiptPath = "data/quality-check-reopens/zh-CN/learn-docs-qc014-representation-successor.json"
	before, err := os.ReadFile(filepath.Join(root, receiptPath))
	if err != nil {
		t.Fatal(err)
	}
	batchPath := filepath.Join(root, "data/retranslation-runs/zh-CN/chatgpt-zh-CN-047")
	if _, err := os.Lstat(batchPath); !os.IsNotExist(err) {
		t.Fatal("probe requires absent batch 047", err)
	}
	defer func() {
		after, err := os.ReadFile(filepath.Join(root, receiptPath))
		if err != nil || !bytes.Equal(before, after) {
			t.Error("receipt changed", err)
		}
		if _, err := os.Lstat(batchPath); !os.IsNotExist(err) {
			t.Error("batch 047 appeared", err)
		}
	}()
	c, err := i18n.BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []string{"unselected", "learn-docs-v1"} {
		clone := *c
		if scope != "unselected" {
			if err := clone.UseTranslationPackage(scope); err != nil {
				t.Fatal(err)
			}
		}
		r, err := i18n.CheckQualityCheckSurfaceReopen(root, &clone, "zh-CN", "learn-docs-qc014-representation-successor")
		if err != nil {
			t.Fatalf("scope=%s: %v", scope, err)
		}
		if r.AuthoritySource != "quality-check" || r.PreviousSnapshotID != "learn-docs-v1-qc-014" || len(r.Units) != 1 || r.Units[0].PreviousRating != "B" || r.Units[0].Snapshot.UnitID != "3d04565df345ae012a9e34fddf1eabc577320d02a2eb124dcf3004bfdfa9132c" || r.Units[0].UnitID != "8e2b3ec2b00595cb7e8ea8587ea6002107a0e616c18f49fc46562702af10f1c3" {
			t.Fatal("unexpected formal authority identity")
		}
		t.Logf("scope=%s: PASS authority_source=%s predecessor=%s units=%d", scope, r.AuthoritySource, r.PreviousSnapshotID, r.UnitCount)
	}
}
