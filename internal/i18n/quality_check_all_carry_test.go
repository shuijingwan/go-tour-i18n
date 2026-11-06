package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func allCarryFixture(t *testing.T, affected bool) (string, *Catalog, string) {
	t.Helper()
	root, cat, old := compatibilityFixture(t)
	data := "# compatible refresh\n" + compatibilityFixtureGlossary
	if affected {
		data += "preferred:\n  transaction: 事务\n"
	}
	compatibilityChange(t, root, data, "carry-refresh")
	_, path, err := AssessGlossaryCompatibility(root, "zh-CN", old, cat)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, cat, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	return root, cat, path
}

func TestQualityCheckZeroReviewAllCarryFinalization(t *testing.T) {
	root, cat, compat := allCarryFixture(t, false)
	scope, err := BuildQualityCheckScope(root, cat, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
	if err != nil || scope.CarryForwardCount != 3 || scope.CurrentResultCount != 0 || !scope.ReadyForFinalization {
		t.Fatalf("scope: %+v %v", scope, err)
	}
	before, _ := os.ReadFile(qualityCheckFinalizationPath(root, "zh-CN", "qc-001"))
	f, _, err := FinalizeQualityCheck(root, cat, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
	if err != nil {
		t.Fatal(err)
	}
	if f.PreviousSnapshotID != "qc-001" || f.PredecessorFinalization == nil || f.PredecessorFinalization.SHA256 != sum(before) || len(f.QCResults) != 1 || f.QCResults[0].SnapshotID != "qc-001" {
		t.Fatalf("false current response lineage: %+v", f)
	}
	for _, u := range f.Units {
		if u.SourceSnapshotID != "qc-001" || len(u.CompatibilityChain) != 1 || u.CompatibilityChain[0].Path != compat {
			t.Fatalf("carry provenance: %+v", u)
		}
	}
	if _, err := os.Stat(qualityCheckResultsPath(root, "zh-CN", "qc-002")); !os.IsNotExist(err) {
		t.Fatal("fake current QC result")
	}
	after, _ := os.ReadFile(qualityCheckFinalizationPath(root, "zh-CN", "qc-001"))
	if !bytes.Equal(before, after) {
		t.Fatal("rewrote predecessor")
	}
	if _, err := VerifyQualityCheckFinalization(root, cat, "zh-CN", "qc-002"); err != nil {
		t.Fatal(err)
	}
	// A subsequent all-carry Snapshot must inherit without synthesizing responses.
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, cat, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-003"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, cat, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-003", PreviousSnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	// Even a still-valid predecessor with a different finalization timestamp
	// cannot silently replace the exact authority bound by the successor.
	var altered QualityCheckFinalization
	if err := decodeStrictCourseSourceDescriptionReviewJSON(before, &altered); err != nil {
		t.Fatal(err)
	}
	altered.FinalizedAt = "2000-01-01T00:00:00Z"
	writeSnapshotSource(t, root, "data/quality-check-snapshots/zh-CN/qc-001/finalization.json", mustJSON(altered))
	if _, err := VerifyQualityCheckFinalization(root, cat, "zh-CN", "qc-002"); err == nil {
		t.Fatal("tampered predecessor accepted")
	}
}

func TestQualityCheckZeroReviewAllCarryRejectsBrokenAuthority(t *testing.T) {
	for _, kind := range []string{"affected", "missing-compatibility", "stale-compatibility", "unfinalized", "non-a", "conflicting-lineage", "source", "review"} {
		t.Run(kind, func(t *testing.T) {
			root, cat, compat := allCarryFixture(t, kind == "affected")
			switch kind {
			case "missing-compatibility":
				if err := os.Rename(filepath.Join(root, compat), filepath.Join(root, "missing-compatibility.json")); err != nil {
					t.Fatal(err)
				}
			case "stale-compatibility":
				writeSnapshotSource(t, root, "internal/tour/ui/zh-CN.json", []byte(`{"locale":"zh-CN","html_lang":"zh-CN","messages":{"test.one":{"kind":"plain","text":"Changed"}}}`))
			case "unfinalized":
				if err := os.Rename(qualityCheckFinalizationPath(root, "zh-CN", "qc-001"), filepath.Join(root, "missing-finalization.json")); err != nil {
					t.Fatal(err)
				}
			case "non-a":
				b, _ := os.ReadFile(qualityCheckResultsPath(root, "zh-CN", "qc-001"))
				writeSnapshotSource(t, root, "data/quality-check-snapshots/zh-CN/qc-001/quality-check-results.json", bytes.ReplaceAll(b, []byte(`"rating": "A"`), []byte(`"rating": "B"`)))
			case "conflicting-lineage":
				if _, err := RecordQualityCheckResults(root, cat, QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "qc-002", FullRereview: true, Rating: "A", UnitIDs: []string{"lesson/1", "lesson/2", "lesson/3"}}); err != nil {
					t.Fatal(err)
				}
			case "source":
				writeSnapshotSource(t, root, "_content/tour/lesson.article", []byte("changed source\n"))
			case "review":
				writeSnapshotSource(t, root, "locales/zh-CN/glossary.yaml", []byte("# unreviewed\n"+compatibilityFixtureGlossary))
			}
			if _, _, err := FinalizeQualityCheck(root, cat, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"}); err == nil {
				t.Fatal("broken authority finalized")
			}
			if _, err := os.Stat(qualityCheckFinalizationPath(root, "zh-CN", "qc-002")); !os.IsNotExist(err) {
				t.Fatal("failed finalization wrote artifact")
			}
		})
	}
}
