package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

type projectionTestAdapter struct{ input string }

func (a projectionTestAdapter) ProtectedInput(*Glossary) (string, []string, error) {
	return a.input, nil, nil
}
func (a projectionTestAdapter) Restore(s string) ([]byte, error) { return []byte(s), nil }
func (a projectionTestAdapter) Validate([]byte, *Glossary) error { return nil }

func successorProjectionFixture(t *testing.T) (string, *Catalog, string) {
	t.Helper()
	root := t.TempDir()
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	writeRetranslationTestGlossary(t, root)
	recordSyntheticUnifiedReview(t, root, "projection-review", "zh-CN")
	source := []byte("immutable physical source\n")
	u := TranslationUnit{ID: "old-projection", Kind: UnitKindPage, SourcePath: "_content/doc/fixture.html", Source: source, SourceSHA256: sum(source)}
	c := &Catalog{Adapted: []AdaptedTranslationUnit{{Unit: u, Adapter: projectionTestAdapter{"Old prose\n"}, StableIndex: 1, WorkingSet: "pages", WorkingSetLimit: 30}}}
	if err := c.UseTranslationUnitScope([]string{u.ID}); err != nil {
		t.Fatal(err)
	}
	r, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT})
	if err != nil {
		t.Fatal(err)
	}
	return root, c, r.BatchID
}

func TestGenerationBundleRejectsCurrentProjectionDrift(t *testing.T) {
	for _, kind := range []string{"input", "unit", "source"} {
		t.Run(kind, func(t *testing.T) {
			root, c, batch := successorProjectionFixture(t)
			bundle, _, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: batch})
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "input":
				c.Adapted[0].Adapter = projectionTestAdapter{"Newly exposed prose\n"}
			case "unit":
				c.Adapted[0].Unit.ID = "new-projection"
			case "source":
				c.Adapted[0].Unit.Source = []byte("changed source\n")
				c.Adapted[0].Unit.SourceSHA256 = sum(c.Adapted[0].Unit.Source)
			}
			if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err == nil {
				t.Fatal("stale bundle accepted")
			}
			if _, _, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: batch}); err == nil {
				t.Fatal("stale export accepted")
			}
			if _, err := os.Stat(filepath.Join(root, "data/retranslation-runs/zh-CN", batch, "raw-responses")); !os.IsNotExist(err) {
				t.Fatal("current-check mutated output")
			}
		})
	}
}

func TestProjectionSuccessorExportPreservesHistoricalEvidence(t *testing.T) {
	root, c, batch := successorProjectionFixture(t)
	manifestPath := filepath.Join(root, "data/retranslation-runs/zh-CN", batch, "manifest.json")
	old, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	c.Adapted[0].Unit.ID = "new-projection"
	c.Adapted[0].Adapter = projectionTestAdapter{"Newly exposed prose\n"}
	if err := c.UseTranslationUnitScope([]string{"new-projection"}); err != nil {
		t.Fatal(err)
	}
	r, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{"new-projection"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.UnitCount != 1 || r.BatchID == batch {
		t.Fatal(r)
	}
	if b, err := os.ReadFile(manifestPath); err != nil || string(b) != string(old) {
		t.Fatal("history changed")
	}
	m := readRetranslationManifest(t, root, r.BatchID)
	if m.Units[0].PreviousRating != "" || m.Units[0].PreviousSnapshotID != "" {
		t.Fatal("new ID inherited old QC")
	}
	// A source mismatch is not an adapter projection successor; arbitrary
	// unknown histories continue to fail closed rather than being ignored.
	c.Adapted[0].Unit.SourceSHA256 = sum([]byte("different bytes"))
	if _, _, err := scanRetranslationBatches(filepath.Join(root, "data/retranslation-runs/zh-CN"), "zh-CN", c); err == nil {
		t.Fatal("unrelated unknown history accepted")
	}
}

func TestQualityCheckProjectionSuccessorRequiresIndependentQC(t *testing.T) {
	root, c, _ := makeRetranslationReviewBatchFixture(t, 1, "qc-001")
	recordQualityCheckRatings(t, root, c, "qc-001", "", "A", []string{"lesson/1"})
	c.Pages = append(c.Pages, c.Pages[0]) // historical lookup remains readable
	c.Pages[0].ID = "lesson/new"
	if err := c.UseTranslationUnitScope([]string{"lesson/new"}); err != nil {
		t.Fatal(err)
	}
	addProcessedPromotionBatch(t, root, c, "chatgpt-zh-CN-006", []string{"lesson/new"})
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	scope, err := BuildQualityCheckScope(root, c, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
	if err != nil {
		t.Fatal(err)
	}
	if scope.CarryForwardCount != 0 || scope.ACount != 0 || scope.PendingCount != 1 || scope.Pending[0].Reason != QualityCheckScopeReasonIdentityChanged || scope.Pending[0].RequiredAction != QualityCheckActionRequired {
		t.Fatalf("old A rebound: %+v", scope)
	}
}
