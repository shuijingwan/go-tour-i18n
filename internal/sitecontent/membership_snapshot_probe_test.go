package sitecontent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// This opt-in probe prepares a disposable Snapshot in a complete repository
// copy. It never executes a real qc-022 preflight or publishes a Snapshot.
func TestMembershipHistoricalBridgeReaderProbe(t *testing.T) {
	out := os.Getenv("GO_TOUR_MEMBERSHIP_SNAPSHOT_CALLPATH")
	if out == "" {
		t.Skip("explicit readonly historical-reader probe required")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := copiedSurfaceTransportRepository(t)
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	AttachIntegratedSurfaceAuthority(c)
	if err := c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	plan, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "engineering-membership-probe", ContentPackage: "learn-docs-v1"})
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	message := "PASS"
	if err != nil {
		message = err.Error()
	}
	if e := os.WriteFile(filepath.Join(out, "exact-call-path.txt"), []byte(message), 0600); e != nil {
		t.Fatal(e)
	}
	t.Log(message)
	if err != nil {
		t.Fatal(err)
	}
	if plan.UnitCount != 51 || plan.ContentPackage == nil || len(plan.ContentPackage.UnitIDs) != 51 {
		t.Fatal("current complete Snapshot scope changed")
	}
	revised := 0
	for _, u := range plan.Units {
		if u.SelectedBatchID == "chatgpt-zh-CN-060" {
			revised++
		}
	}
	if revised != 5 {
		t.Fatal("latest Surface revisions not selected", revised)
	}
	b, _ := json.MarshalIndent(plan, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "disposable-manifest.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("CURRENT=51; HISTORICAL=49+2; REVISED=5; SNAPSHOT_PUBLICATION=no")
}
