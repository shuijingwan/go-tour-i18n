package i18n

import "fmt"

// acceptedResponseCatalog scopes only frozen response reconstruction. The
// caller's current source/protector/validator still uses its complete Catalog.
// Call only after the inherited-lineage selector proves a same-ID decoder, or
// in the explicit current-validation transport branch. A replacement successor
// may legitimately have no same-ID predecessor and must not enter this owner.
// An explicit scope can cross a package era only through formal membership.
func acceptedResponseCatalog(root string, current *Catalog, locale string, record RetranslationBatchUnit) (*Catalog, func() error, error) {
	if current == nil || len(current.TranslationScope) == 0 {
		return current, nil, nil // Preserve the native legacy inventory boundary.
	}
	if _, _, _, _, err := translationAuthorizationScope(current); err != nil {
		return nil, nil, err
	}
	if err := ValidateLocaleName(locale); err != nil {
		return nil, nil, err
	}
	if err := validateSnapshotID(record.PreviousSnapshotID); err != nil {
		return nil, nil, err
	}
	path := "data/quality-check-snapshots/" + locale + "/" + record.PreviousSnapshotID + "/manifest.json"
	before, err := readSnapshotReferencedFile(root, path)
	if err != nil {
		return nil, nil, err
	}
	frozen, err := readQualityCheckSnapshot(root, locale, record.PreviousSnapshotID, false)
	if err != nil {
		return nil, nil, err
	}
	found := false
	for _, u := range frozen.Units {
		found = found || (u.UnitID == record.UnitID && u.UnitKind == record.UnitKind && u.SourcePath == record.SourcePath && u.SourceSHA256 == record.SourceSHA256)
	}
	if !found {
		return nil, nil, fmt.Errorf("accepted response lacks exact historical Unit/source")
	}
	historical := current
	var membershipCheck func() error
	if !acceptedResponseSourceScope(current, frozen) {
		members, check, err := snapshotMembershipCatalogs(root, current)
		if err != nil {
			return nil, nil, err
		}
		historical = members[record.UnitID]
		if historical == nil || check == nil || !acceptedResponseSourceScope(historical, frozen) {
			return nil, nil, fmt.Errorf("accepted response historical scope outside explicit package membership")
		}
		membershipCheck = check
	}
	// Keep an already explicit historical world, rather than rediscovering it
	// among retained representation eras with the same source coordinates.
	// The decoder's existing complete-ID/mapping gates remain unchanged.
	if err := checkSnapshotPackageBinding(root, historical, frozen, false); err != nil {
		return nil, nil, err
	}
	u, err := historical.Unit(record.UnitID)
	if err != nil || u.Kind != record.UnitKind || u.SourcePath != record.SourcePath || u.SourceSHA256 != record.SourceSHA256 {
		return nil, nil, fmt.Errorf("accepted response historical Unit/source identity mismatch")
	}
	check := func() error {
		if membershipCheck != nil {
			if err := membershipCheck(); err != nil {
				return err
			}
		}
		after, err := readSnapshotReferencedFile(root, path)
		if err != nil || sum(after) != sum(before) {
			return fmt.Errorf("accepted response predecessor Snapshot changed")
		}
		return nil
	}
	if err := check(); err != nil {
		return nil, nil, err
	}
	return historical, check, nil
}

func acceptedResponseSourceScope(c *Catalog, s *QualityCheckSnapshotManifest) bool {
	_, _, _, ordered, err := translationAuthorizationScope(c)
	if err != nil || len(ordered) != len(s.Units) {
		return false
	}
	for i, u := range ordered {
		old := s.Units[i]
		if u.Kind != old.UnitKind || u.SourcePath != old.SourcePath || u.SourceSHA256 != old.SourceSHA256 {
			return false
		}
	}
	return true
}
