package i18n

import (
	"fmt"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// QCResults freezes the primary response lineage. A membership carry retains
// the sibling's actual SourceSnapshotID, including an unfinalized partial QC
// response. Resolve it only through that member's complete accepted history.
type historicalMembershipReviewers struct {
	root, locale string
	catalog      *Catalog
	snapshot     *QualityCheckSnapshotManifest
	finalization *QualityCheckFinalization
	visiting     map[string]bool
	bind         func(string, string) ([]byte, error)
	results      map[string]*QualityCheckResults
	members      map[string]*Catalog
	check        func() error
	proofs       map[*Catalog]*HistoricalFinalizationIntegrity
}

func (h *historicalMembershipReviewers) prepare() error {
	if h.members != nil {
		return nil
	}
	if h.catalog == nil || h.snapshot.ContentPackage == nil {
		return fmt.Errorf("frozen package membership required for sibling Reviewer evidence")
	}
	b := h.snapshot.ContentPackage
	c, err := HistoricalTranslationPackageCatalog(h.catalog, b.ID, b.AuthoritySHA256)
	if err != nil {
		return err
	}
	if err := checkSnapshotPackageBinding(h.root, c, h.snapshot, false); err != nil {
		return err
	}
	// Source discovery already resolved the exact historical descriptor. Its
	// immutable receipt/registry references remain authority; today's pointer
	// can advance without changing the frozen predecessor relationship.
	historical := *c
	historical.TranslationPackages = append([]TranslationPackageDescriptor(nil), c.TranslationPackages...)
	for i, d := range historical.TranslationPackages {
		if d.Membership == nil {
			continue
		}
		declaration := *d.Membership
		declaration.References = nil
		for _, ref := range d.Membership.References {
			if ref.Path != "data/content-package-memberships/current.json" {
				declaration.References = append(declaration.References, ref)
			}
		}
		historical.TranslationPackages[i].Membership = &declaration
	}
	c = &historical
	members, check, err := snapshotMembershipCatalogs(h.root, c)
	if err != nil {
		return err
	}
	if len(members) == 0 || check == nil {
		return fmt.Errorf("sibling Reviewer evidence outside formal membership")
	}
	// The caller's live registry cannot add a relationship absent from the
	// finalized Snapshot. Bind the exact declaration already frozen there.
	for _, d := range c.TranslationPackages {
		if d.ID != b.ID || d.AuthoritySHA256 != b.AuthoritySHA256 || d.Membership == nil {
			continue
		}
		for _, ref := range d.Membership.References {
			found := false
			for _, frozen := range b.AuthorityReferences {
				found = found || frozen == ref
			}
			if !found {
				return fmt.Errorf("sibling membership authority not frozen in Snapshot: %s", ref.Path)
			}
			if _, err := h.bind(ref.Path, ref.SHA256); err != nil {
				return err
			}
		}
	}
	h.members, h.check = members, check
	return nil
}

func (h *historicalMembershipReviewers) resolve(u QualityCheckFinalizationUnit) (*QualityCheckResults, *QualityCheckSnapshotManifest, error) {
	if err := h.prepare(); err != nil {
		return nil, nil, err
	}
	member := h.members[u.UnitID]
	if member == nil {
		return nil, nil, fmt.Errorf("Reviewer Unit outside ordered membership")
	}
	proof := h.proofs[member]
	if proof == nil {
		// Discovery is not proof. Every missing origin for this member must
		// match the same complete finalized accepted set, never a latest/path
		// fallback or a mixture of arbitrary partial Snapshot judgments.
		ids, err := qualityCheckFinalizedSnapshotIDs(h.root, h.locale, member)
		if err != nil {
			return nil, nil, err
		}
		for _, id := range ids {
			if h.visiting[id] {
				continue
			}
			path := filepath.ToSlash(qualityCheckFinalizationPath("", h.locale, id))
			data, err := readCompatibilityFile(h.root, path)
			if err != nil {
				return nil, nil, err
			}
			var f QualityCheckFinalization
			if err := contentidentity.StrictJSON(data, &f); err != nil {
				return nil, nil, err
			}
			matches := true
			for _, wanted := range h.finalization.Units {
				if h.members[wanted.UnitID] != member || h.results[wanted.SourceSnapshotID] != nil {
					continue
				}
				found := false
				for _, accepted := range f.Units {
					found = found || (accepted.UnitID == wanted.UnitID && accepted.SourceSnapshotID == wanted.SourceSnapshotID && accepted.Rating == "A" && qualityCheckSnapshotIdentityMatches(accepted.Snapshot, wanted.Snapshot))
				}
				matches = matches && found
			}
			if !matches {
				continue
			}
			// Explicit package bindings, when present, must also select the
			// declared historical member. Legacy manifests use the canonical
			// complete ordered source-coordinate selector, not their name.
			inventory := *member
			inventory.TranslationScope = nil
			selected, registered, err := selectSnapshotTranslationPackage(h.root, &inventory, h.locale, id)
			if err != nil {
				return nil, nil, err
			}
			pkg, authority, scope, ordered, err := translationAuthorizationScope(member)
			if err != nil {
				return nil, nil, err
			}
			gotPkg, gotAuthority, gotScope, _, err := translationAuthorizationScope(selected)
			if err != nil || !registered || pkg != gotPkg || authority != gotAuthority || scope != gotScope || len(f.Units) != len(ordered) {
				return nil, nil, fmt.Errorf("sibling finalization package mismatch")
			}
			for i, old := range f.Units {
				current := ordered[i]
				if old.UnitID != current.ID || old.Snapshot.UnitKind != current.Kind || old.Snapshot.SourcePath != current.SourcePath || old.Snapshot.SourceSHA256 != current.SourceSHA256 {
					return nil, nil, fmt.Errorf("sibling finalization source membership mismatch")
				}
			}
			if _, err := h.bind(path, sum(data)); err != nil {
				return nil, nil, err
			}
			proof, err = verifyHistoricalQualityCheckFinalization(h.root, h.locale, id, member, h.visiting)
			if err != nil {
				return nil, nil, err // Never hide a failed matching proof by fallback.
			}
			for _, ref := range proof.References {
				if _, err := h.bind(ref.Path, ref.SHA256); err != nil {
					return nil, nil, err
				}
			}
			h.proofs[member] = proof
			break
		}
	}
	if proof == nil {
		return nil, nil, fmt.Errorf("complete membership-authorized sibling finalization required")
	}
	// The sibling proof may carry A from an earlier, unfinalized response.
	// Its QCResults binds both that response SHA and its Snapshot manifest SHA.
	for _, ref := range proof.Finalization.QCResults {
		if ref.SnapshotID != u.SourceSnapshotID {
			continue
		}
		if !validSHA256(ref.SHA256) {
			return nil, nil, fmt.Errorf("sibling Reviewer result lacks frozen SHA")
		}
		data, err := h.bind(ref.Path, ref.SHA256)
		if err != nil {
			return nil, nil, err
		}
		var r QualityCheckResults
		if err := contentidentity.StrictJSON(data, &r); err != nil {
			return nil, nil, err
		}
		path := filepath.ToSlash(qualityCheckSnapshotManifestPath("", h.locale, ref.SnapshotID))
		data, err = h.bind(path, r.SnapshotManifestSHA256)
		if err != nil {
			return nil, nil, err
		}
		var s QualityCheckSnapshotManifest
		if err := contentidentity.StrictJSON(data, &s); err != nil {
			return nil, nil, err
		}
		if r.Locale != h.locale || s.Locale != h.locale || r.SnapshotID != ref.SnapshotID || s.SnapshotID != ref.SnapshotID || r.Rubric != TranslationQualityRubric {
			return nil, nil, fmt.Errorf("sibling original Reviewer result identity mismatch")
		}
		// Both objects came from the exact bytes already structurally validated
		// by the complete sibling proof. No unbound reread supplies their fields.
		return &r, &s, nil // Caller still verifies the exact per-Unit A judgment.
	}
	return nil, nil, fmt.Errorf("sibling proof lacks exact Reviewer source Snapshot")
}
