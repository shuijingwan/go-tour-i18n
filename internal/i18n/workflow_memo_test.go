package i18n

import (
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Exercise the actual gate readers with the CLI's pure-transform memo enabled,
// including mutations between checks in the same invocation. No saved PASS is
// memoized and every fixture is confined to t.TempDir.
func TestWorkflowMemoFailClosedGates(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"preview-currentness", TestPreviewFailureSharedFailClosed},
		{"one-tu-revision-and-independent-qc", TestPreviewFailureSharedReopenRevisionAndIndependentQC},
		{"integrated-surface", TestSurfaceAggregateReopenBindingAndReadFailClosed},
		{"surface-currentness", TestSurfaceReviewReopenFailsClosedOnScopeAndEvidence},
		{"historical-direct-surface", TestSurfaceReviewFindingCanExplicitlyReopenFinalizedAWithoutChangingHistory},
		{"current-authorization-mismatch", TestCurrentTranslationReattestationFailClosedMatrix},
		{"snapshot-mismatch", TestCurrentTranslationSnapshotMigrationFailClosed},
		{"historical-compatibility", TestCurrentTranslationMissingHistoricalIntermediateCompatibility},
	} {
		t.Run(tc.name, func(t *testing.T) { done := contentidentity.BeginMemoization(); defer done(); tc.run(t) })
	}
}
