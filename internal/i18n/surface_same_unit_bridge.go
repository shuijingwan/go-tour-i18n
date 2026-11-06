package i18n

import (
	"bytes"
	"fmt"
)

func surfaceBridgeMember(id string) string {
	return "formal/surface-transport-authority/" + id + ".json"
}

// Compose only from the actual imported Surface projection. Its authority is
// historical representation evidence; buildSameUnitBridge owns the new QC
// finding/currentness checks. The observer freezes every dependency afresh.
func surfaceSameUnitBridgeOutput(root string, c *Catalog, locale string, r RetranslationBatchUnit, raw, reviewed []byte, files map[string][]byte, observe func(string) ([]byte, error)) (TranslationUnitAdapter, []byte, []byte, error) {
	if r.RevisionFeedbackSource != "surface_review" || r.UnitKind != UnitKindPage {
		return nil, nil, nil, fmt.Errorf("Surface bridge requires exact Page predecessor")
	}
	member := surfaceBridgeMember(r.UnitID)
	var authority SurfaceTransportAuthority
	if err := decodeStrictCourseSourceDescriptionReviewJSON(files[member], &authority); err != nil || !bytes.Equal(files[member], surfaceTransportBytes(&authority)) {
		return nil, nil, nil, fmt.Errorf("Surface bridge imported mapping unavailable")
	}
	p, lineage, err := surfaceTransportOutput(root, c, locale, r)
	if err != nil || lineage == nil || lineage.Bridge.Path != surfaceTransportPath(locale, r.RevisionAuthorizationID) || lineage.Bridge.SHA256 != sum(files[member]) {
		return nil, nil, nil, fmt.Errorf("Surface bridge sealed lineage mismatch: %v", err)
	}
	for _, ref := range append(append([]GlossaryArchiveReference{}, lineage.Dependencies...), authority.Reopen) {
		b, err := observe(ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			return nil, nil, nil, fmt.Errorf("Surface bridge historical dependency drift: %s", ref.Path)
		}
	}
	var reopen QualityCheckSurfaceReopen
	b, err := observe(authority.Reopen.Path)
	if err != nil || authority.Reopen.Path != "data/quality-check-reopens/"+locale+"/"+r.RevisionAuthorizationID+".json" || decodeStrictCourseSourceDescriptionReviewJSON(b, &reopen) != nil || reopen.Locale != locale || reopen.ReopenID != r.RevisionAuthorizationID || reopen.PreviousSnapshotID != r.PreviousSnapshotID || reopen.revisionFeedbackSource() != "surface_review" {
		return nil, nil, nil, fmt.Errorf("Surface bridge unrelated reopen")
	}
	matched := 0
	for _, binding := range authority.Bindings {
		if binding.UnitID != r.UnitID {
			continue
		}
		if binding.Predecessor.UnitID != r.UnitID || binding.Predecessor.UnitKind != r.UnitKind || binding.Predecessor.SourcePath != r.SourcePath || binding.Predecessor.SourceSHA256 != r.SourceSHA256 {
			return nil, nil, nil, fmt.Errorf("Surface bridge predecessor formal identity drift")
		}
		for _, q := range reopen.Units {
			if q.UnitID == r.UnitID && qualityCheckSnapshotIdentityMatches(q.Snapshot, binding.Predecessor) && reopen.findingForUnit(r.UnitID) == r.PreviousFinding {
				matched++
			}
		}
	}
	if matched != 1 {
		return nil, nil, nil, fmt.Errorf("Surface bridge exact predecessor/reopen mismatch")
	}
	got, failures := p.restore(string(raw))
	if len(failures) != 0 || !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(got)), reviewed) {
		return nil, nil, nil, fmt.Errorf("Surface bridge passed raw/candidate mismatch")
	}
	u, err := c.Unit(r.UnitID)
	if err != nil {
		return nil, nil, nil, err
	}
	factory, ok := u.Adapter.(SurfaceSameUnitBridgeAdapter)
	if !ok || u.Kind != r.UnitKind || u.SourcePath != r.SourcePath || u.SourceSHA256 != r.SourceSHA256 {
		return nil, nil, nil, fmt.Errorf("Surface bridge source adapter mismatch")
	}
	return factory.BridgeSurfaceRevision(p.Adapter, raw, reviewed)
}
