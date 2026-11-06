package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoricalFinalizationZeroReviewBridge(t *testing.T) {
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 1, "reviewed-first")
	recordQualityCheckRatings(t, root, catalog, "reviewed-first", "", "A", []string{"lesson/1"})
	if _, _, err := FinalizeQualityCheck(root, catalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "reviewed-first"}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"zero-carry", "reviewed-later"} {
		if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: id}); err != nil {
			t.Fatal(err)
		}
		previous := "reviewed-first"
		if id == "reviewed-later" {
			previous = "zero-carry"
			recordQualityCheckRatings(t, root, catalog, id, previous, "A", []string{"lesson/1"})
		}
		if _, _, err := FinalizeQualityCheck(root, catalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: id, PreviousSnapshotID: previous}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(qualityCheckResultsPath(root, "zh-CN", "zero-carry")); !os.IsNotExist(err) {
		t.Fatal("zero-review fixture created QC response", err)
	}
	proof, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", "reviewed-later", catalog)
	if err != nil {
		t.Fatal("reviewed -> zero-review -> reviewed history", err)
	}
	carryPath := qualityCheckFinalizationPath(root, "zh-CN", "zero-carry")
	carryBytes, err := os.ReadFile(carryPath)
	if err != nil {
		t.Fatal(err)
	}
	boundCarry := false
	for _, ref := range proof.References {
		boundCarry = boundCarry || ref.Path == filepath.ToSlash(strings.TrimPrefix(carryPath, root+string(filepath.Separator))) && ref.SHA256 == sum(carryBytes)
	}
	if !boundCarry {
		t.Fatal("zero-review finalization omitted from immutable dependency proof")
	}
	for _, label := range []string{"missing-finalization", "missing-predecessor-ref", "wrong-predecessor-sha", "wrong-previous-identity", "not-A-only", "incomplete-membership", "cycle", "unbound-QC-response"} {
		t.Run(label, func(t *testing.T) {
			defer func() {
				if err := os.WriteFile(carryPath, carryBytes, 0644); err != nil {
					t.Fatal(err)
				}
			}()
			var carry QualityCheckFinalization
			if err := json.Unmarshal(carryBytes, &carry); err != nil {
				t.Fatal(err)
			}
			switch label {
			case "missing-finalization":
				if err := os.Remove(carryPath); err != nil {
					t.Fatal(err)
				}
			case "missing-predecessor-ref":
				carry.PredecessorFinalization = nil
			case "wrong-predecessor-sha":
				carry.PredecessorFinalization.SHA256 = strings.Repeat("0", 64)
			case "wrong-previous-identity":
				carry.PreviousSnapshotID = "different-snapshot"
			case "not-A-only":
				carry.Units[0].Rating = "B"
			case "incomplete-membership":
				carry.Units = nil
			case "cycle":
				carry.PreviousSnapshotID = "reviewed-later"
				b, err := os.ReadFile(qualityCheckFinalizationPath(root, "zh-CN", "reviewed-later"))
				if err != nil {
					t.Fatal(err)
				}
				carry.PredecessorFinalization = &GlossaryArchiveReference{Path: "data/quality-check-snapshots/zh-CN/reviewed-later/finalization.json", SHA256: sum(b)}
			case "unbound-QC-response":
				path := qualityCheckResultsPath(root, "zh-CN", "zero-carry")
				if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
				defer os.Remove(path)
			}
			if label != "missing-finalization" && label != "unbound-QC-response" {
				b, err := json.Marshal(carry)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(carryPath, b, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", "reviewed-later", catalog); err == nil {
				t.Fatal("invalid zero-review bridge accepted")
			}
		})
	}
}
