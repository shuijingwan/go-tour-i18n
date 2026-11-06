package i18n

import (
	"strings"
	"testing"
)

func TestSameUnitBridgeManifestPolicy(t *testing.T) {
	valid := RetranslationBatchUnit{UnitID: "page", UnitKind: UnitKindPage, RevisionFeedbackSource: "quality_check", PreviousSnapshotID: "reviewed", PreviousRating: "B", PreviousFinding: "exact finding", RepresentationBridge: &GlossaryArchiveReference{Path: "data/quality-check-reopens/zh-CN/bridge.json", SHA256: strings.Repeat("a", 64)}}
	check := func(u RetranslationBatchUnit) error {
		return validateRetranslationManifestReexportProvenance(&RetranslationBatchManifest{Units: []RetranslationBatchUnit{u}})
	}
	if err := check(valid); err != nil {
		t.Fatal("valid additive ordinary QC binding", err)
	}
	cases := map[string]func(*RetranslationBatchUnit){"initial": func(u *RetranslationBatchUnit) { u.RevisionFeedbackSource = "" }, "surface": func(u *RetranslationBatchUnit) { u.RevisionFeedbackSource = "surface_review" }, "current-validation": func(u *RetranslationBatchUnit) { u.RevisionFeedbackSource = CurrentValidationRevisionFeedback }, "example": func(u *RetranslationBatchUnit) { u.UnitKind = UnitKindExample }, "missing-predecessor": func(u *RetranslationBatchUnit) { u.PreviousSnapshotID = "" }, "mixed-reopen": func(u *RetranslationBatchUnit) { u.RevisionAuthorizationID = "other" }, "bad-sha": func(u *RetranslationBatchUnit) {
		r := *u.RepresentationBridge
		r.SHA256 = "bad"
		u.RepresentationBridge = &r
	}}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			u := valid
			edit(&u)
			if err := check(u); err == nil {
				t.Fatal("mixed/unknown bridge authority accepted")
			}
		})
	}
	// Legacy ordinary metadata remains unchanged when the additive field is absent.
	legacy := RetranslationBatchUnit{UnitID: "legacy", UnitKind: UnitKindExample}
	if err := check(legacy); err != nil {
		t.Fatal(err)
	}
}
