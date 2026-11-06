package sitecontent

import (
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestIntegratedSurfaceHistoricalReopenCurrentReader(t *testing.T) {
	root, c, ids := surfaceFixture(t, "learn-docs", 2)
	f := surfaceStart(t, root, c, ids)
	structuredGlossarySuccessor(t, root, "unused-surface-reopen-term")
	if _, err := i18n.VerifyQualityCheckFinalization(root, c, "zh-CN", f.SnapshotID); err == nil {
		t.Fatal("ordinary finalization verifier accepted historical glossary")
	}
	if _, _, err := i18n.EstablishCurrentTranslationAuthorization(root, c, "zh-CN", f.SnapshotID); err != nil {
		t.Fatal(err)
	}
	build := surfaceBuilder(t, root, c, &f)
	attachIntegratedSurfaceAuthority(c, build)
	closure := surfaceDescriptorClosure(t, root, c, f.SnapshotID)
	artifact := i18n.SurfaceCandidateArtifact{SnapshotID: f.SnapshotID, Unit: f.Units[0].Snapshot}
	const finding = "Correct the exact reviewed Page language."
	r, _ := surfaceReceipt(t, root, c, build, []Reference{closure}, "historical-reopen-failed", "failed", []SurfaceFinding{{ID: artifact.Unit.SourcePath, Finding: finding, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{artifact}}})
	o := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "historical-page-reopen", PreviousSnapshotID: f.SnapshotID, SurfaceReviewID: r.ReviewID, Finding: finding, UnitIDs: ids[:1], IntegratedSurface: true}
	reopened, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, o)
	if err != nil || reopened == nil || reopened.Units[0].Snapshot != artifact.Unit {
		t.Fatal("actual Integrated Surface reader rejected historical Page predecessor", err)
	}
	if _, err := i18n.CheckQualityCheckSurfaceReopen(root, c, o.Locale, o.ReopenID); err != nil {
		t.Fatal("current consumption of historical predecessor", err)
	}
}
