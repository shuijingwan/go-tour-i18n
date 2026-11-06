package i18n

import "testing"

func TestInheritedBridgePredecessorEligibility(t *testing.T) {
	for _, source := range []string{"quality_check", "surface_review", "", PreviewRevisionFeedback, QCRepresentationFeedback, RepresentationRevisionFeedback, CurrentValidationRevisionFeedback} {
		r := RetranslationBatchUnit{PreviousSnapshotID: "predecessor", RevisionFeedbackSource: source}
		want := source == "quality_check" || source == "surface_review"
		if hasInheritedBridgePredecessor(r) != want {
			t.Fatal("expanded representation inheritance to another revision class", source)
		}
		if !want {
			if _, lineage, err := inheritedBridgeOutput(t.TempDir(), nil, nil, "zh-CN", r); err != nil || lineage != nil {
				t.Fatal("unrelated revision class entered inherited bridge reader", source, err)
			}
		}
		r.PreviousSnapshotID = ""
		if hasInheritedBridgePredecessor(r) {
			t.Fatal("missing exact predecessor accepted", source)
		}
	}
}
