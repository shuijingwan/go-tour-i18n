package i18n

import "fmt"

type AcceptedCommentTarget struct {
	UnitID       string                   `json:"unit_id"`
	Kind         UnitKind                 `json:"kind"`
	Candidate    GlossaryArchiveReference `json:"candidate"`
	SourceSHA256 string                   `json:"source_sha256"`
	Bytes        []byte                   `json:"-"`
}

// CurrentAcceptedCommentTargets consumes existing current authorization or the
// canonical finalized Tour boundary. Directory presence, latest filename, and
// a historical A alone never authorize a target. This function writes nothing.
func CurrentAcceptedCommentTargets(root string, c *Catalog, locale string) ([]AcceptedCommentTarget, []GlossaryArchiveReference, error) {
	var f *QualityCheckFinalization
	refs := []GlossaryArchiveReference{}
	if len(c.TranslationScope) > 0 {
		ids, err := qualityCheckFinalizedSnapshotIDs(root, locale, c)
		if err != nil {
			return nil, nil, err
		}
		var lastErr error
		for _, id := range ids {
			if err := finalizationMatchesLatestPromotionCandidates(root, c, locale, id); err != nil {
				lastErr = err
				continue
			}
			a, err := RequireCurrentTranslationAuthorization(root, c, locale, id)
			if err != nil {
				lastErr = err
				continue
			}
			if f != nil {
				return nil, nil, fmt.Errorf("ambiguous current finalized target")
			}
			f = &a.Historical.Finalization
			ref, err := successorReference(root, "data/quality-check-snapshots/"+locale+"/"+id+"/finalization.json")
			if err != nil {
				return nil, nil, err
			}
			refs = append(refs, ref)
		}
		if f == nil {
			return nil, nil, fmt.Errorf("CURRENT_TARGET_UNAVAILABLE: %v", lastErr)
		}
	} else {
		var err error
		f, refs, err = RequireCurrentCanonicalQualityCheck(root, locale, c)
		if err != nil {
			return nil, nil, err
		}
	}
	out := []AcceptedCommentTarget{}
	for _, r := range f.Units {
		u, err := c.Unit(r.UnitID)
		if err != nil {
			return nil, nil, err
		}
		if r.Snapshot.SourceSHA256 != u.SourceSHA256 {
			return nil, nil, fmt.Errorf("current comment source stale: %s", u.ID)
		}
		b, err := readSnapshotReferencedFile(root, r.Snapshot.CandidatePath)
		if err != nil || sum(b) != r.Snapshot.CandidateSHA256 {
			return nil, nil, fmt.Errorf("current comment candidate drift: %s", u.ID)
		}
		out = append(out, AcceptedCommentTarget{u.ID, u.Kind, GlossaryArchiveReference{r.Snapshot.CandidatePath, r.Snapshot.CandidateSHA256}, u.SourceSHA256, b})
	}
	return out, refs, nil
}
