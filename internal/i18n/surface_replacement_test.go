package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSurfaceReplacementRejectsUnboundClaims(t *testing.T) {
	root, c, _ := makeRetranslationReviewBatchFixture(t, 2, "qc-001")
	ids := []string{"lesson/1", "lesson/2"}
	recordQualityCheckRatings(t, root, c, "qc-001", "", "A", ids)
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-001"}); err != nil {
		t.Fatal(err)
	}
	// Legacy textual single-finding authorization remains byte-compatible.
	finding := "Correct reviewed lesson/1."
	body := []byte(finding + "\nunit_id: lesson/1\n")
	path := "data/locale-surface-reviews/zh-CN/replacement-original.md"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, path), body, 0600); err != nil {
		t.Fatal(err)
	}
	o := QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "original", PreviousSnapshotID: "qc-001", SurfaceReviewID: "replacement-original", Finding: finding, UnitIDs: ids[:1]}
	r, p, err := RecordQualityCheckSurfaceReopen(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"replacement-without-context", "changed-current-id", "forged-context"} {
		t.Run(name, func(t *testing.T) {
			bad := *r
			bad.Units = append([]QualityCheckSurfaceReopenUnit(nil), r.Units...)
			switch name {
			case "replacement-without-context":
				bad.Units[0].Replacement = &TranslationUnitReplacement{OldUnitID: ids[0], CurrentUnitID: ids[1]}
			case "changed-current-id":
				bad.Units[0].UnitID = ids[1]
			case "forged-context":
				bad.ReplacementContext = &TranslationUnitReplacementContext{HistoricalDocuments: GlossaryArchiveReference{Path: path, SHA256: sum(body)}}
			}
			if err := writeTranslationJSON(filepath.Join(root, p), bad); err != nil {
				t.Fatal(err)
			}
			if _, err := readCurrentQualityCheckSurfaceReopen(root, c, "zh-CN", "original"); err == nil {
				t.Fatal("unbound replacement claim accepted")
			}
			if err := writeTranslationJSON(filepath.Join(root, p), r); err != nil {
				t.Fatal(err)
			}
		})
	}
	if _, err := readCurrentQualityCheckSurfaceReopen(root, c, "zh-CN", "original"); err != nil {
		t.Fatal("legacy read regressed", err)
	}
}
