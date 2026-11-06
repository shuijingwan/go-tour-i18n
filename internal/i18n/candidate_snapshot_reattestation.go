package i18n

import (
	"fmt"
	"path/filepath"
	"reflect"
)

// Discovery is only an identity selection. A matching accepted set must pass
// shared historical integrity and current re-attestation before it can replace
// fresh Generation validation. No compatibility inventory is consulted.
func selectedSnapshotReattestation(root string, c *Catalog, locale string, latest *latestRetranslationUnits) (*CurrentTranslationAuthorization, error) {
	// Unmigrated Tour retains its legacy reader. Registered scopes, including
	// future representations, all use this same shared selection primitive.
	if len(c.TranslationScope) == 0 {
		return nil, nil
	}
	ids, err := qualityCheckFinalizedSnapshotIDs(root, locale, c)
	if err != nil {
		return nil, err
	}
	for i := len(ids) - 1; i >= 0; i-- {
		s, err := readQualityCheckSnapshot(root, locale, ids[i], false)
		if err != nil {
			return nil, err
		}
		if len(s.Units) != len(latest.ordered) {
			continue
		}
		match := true
		for j, old := range s.Units {
			u := latest.ordered[j]
			choice, ok := latest.selectedByID[u.ID]
			base := filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, choice.batchID))
			if !ok || !selectedRetranslationIdentityMatches(u, choice) || choice.result.Status != "passed" || old.UnitID != u.ID || old.SourceSHA256 != u.SourceSHA256 || old.SelectedBatchID != choice.batchID || old.CandidatePath != base+"/"+choice.result.CandidatePath || old.ValidationPath != base+"/"+choice.result.ValidationPath {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		a, err := AssessCurrentTranslation(root, c, locale, ids[i])
		if err != nil {
			return nil, err
		}
		if err := requireSnapshotReattestation(a); err != nil {
			return nil, err
		}
		return a, nil
	}
	return nil, nil
}

func requireSnapshotReattestation(a *CurrentTranslationAuthorization) error {
	if a.Schema != CurrentTranslationAuthorizationSchema || a.Identity != authorizationIdentity(*a) || len(a.Units) == 0 || a.Compatible+a.Affected != len(a.Units) || a.Stale != 0 || a.Unknown != 0 || len(a.Canonical) != 0 {
		return fmt.Errorf("Snapshot current re-attestation is not a complete machine-valid reviewable scope")
	}
	for _, u := range a.Units {
		if (u.Classification != "compatible" && u.Classification != "affected") || u.Validation != "passed" {
			return fmt.Errorf("Snapshot current re-attestation rejected: unit=%s classification=%s validation=%s", u.Snapshot.UnitID, u.Classification, u.Validation)
		}
	}
	return nil
}

// This optional Snapshot binding reuses the assessment wire format; it is not
// stored as an authorization. Historic readers check its immutable structure;
// current review/scope/bundle readers also independently revalidate it today.
func checkSnapshotReattestation(root string, c *Catalog, s *QualityCheckSnapshotManifest, current bool) error {
	a := s.CurrentReattestation
	if a == nil {
		if s.CurrentRevision != nil {
			return fmt.Errorf("mixed Snapshot lacks sealed current assessment")
		}
		return nil
	}
	if err := requireSnapshotReattestation(a); err != nil {
		return err
	}
	if a.Locale != s.Locale || a.Glossary.Path != s.GlossaryPath || a.Glossary.SHA256 != s.GlossarySHA256 || a.Historical.Finalization.SnapshotID == s.SnapshotID || len(a.Units) != len(s.Units) {
		return fmt.Errorf("Snapshot current re-attestation binding mismatch")
	}
	if s.CurrentRevision != nil {
		return checkMixedSnapshot(root, c, s, current, map[string]bool{})
	}
	for i, u := range a.Units {
		if !reflect.DeepEqual(u.Snapshot, s.Units[i]) {
			return fmt.Errorf("Snapshot current re-attestation candidate set mismatch")
		}
	}
	if !current || c == nil {
		return nil
	}
	if err := checkFlatHistoricalIntegrity(root, c, &a.Historical); err != nil {
		return err
	}
	want, err := assessCurrentTranslationProof(root, c, s.Locale, a.Historical)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(a, want) {
		return fmt.Errorf("Snapshot current re-attestation stale")
	}
	return requireSnapshotReattestation(want)
}
