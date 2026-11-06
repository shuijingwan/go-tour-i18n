package i18n

import (
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Native workflow membership is also a complete scope authority. Registered
// package selection must not make the native Page/eligible-Example inventory
// invisible when a reader classifies a disjoint finalized Snapshot.
func snapshotMatchesNativeWorkflow(s *QualityCheckSnapshotManifest, c *Catalog) bool {
	native := *c
	native.TranslationScope = nil
	ordered, _, _, err := localeWorkflowUnitList(&native)
	if err != nil || len(ordered) == 0 || len(ordered) != len(s.Units) {
		return false
	}
	for i, u := range ordered {
		x := s.Units[i]
		if x.UnitID != u.ID || x.UnitKind != u.Kind || x.SourcePath != u.SourcePath || x.SourceSHA256 != u.SourceSHA256 {
			return false
		}
		var section *QualityCheckPageSectionIdentity
		if u.Kind == UnitKindPage {
			p, err := native.Page(u.ID)
			if err != nil {
				return false
			}
			section = &QualityCheckPageSectionIdentity{Article: p.Article, SectionNumber: p.SectionNumber, SourceTitle: p.SourceTitle, Route: p.Route}
		}
		if !reflect.DeepEqual(x.PageSection, section) {
			return false
		}
	}
	return true
}

// A fresh machine-successor Snapshot need not declare its predecessor as a
// finalization edge. Its explicit applied reconciliation instead supplies the
// frozen catalog. Coordinates only prefilter candidates; exact historical IDs,
// package authority and the native receipt/candidate/validation proof decide.
func foreignSnapshotMachineCatalog(root string, current *Catalog, endpoint, historical *QualityCheckSnapshotManifest, descriptor TranslationPackageDescriptor) (*Catalog, error) {
	contexts := map[string]*machineSelection{}
	for _, u := range endpoint.Units {
		if u.RepresentationReconciliation == nil {
			continue
		}
		_, machine, err := machineSnapshotEvidence(root, current, endpoint.Locale, u, contexts)
		if err != nil {
			return nil, err
		}
		frozen := machine.ctx.resolution.HistoricalCatalog
		for _, d := range frozen.TranslationPackages {
			if d.ID == descriptor.ID && d.AuthoritySHA256 == descriptor.AuthoritySHA256 && snapshotMatchesTranslationDescriptor(historical, d, true) {
				return frozen, nil
			}
		}
	}
	return nil, nil
}

// SelectFinalizedTranslationPackage scopes a diagnostic to a registered whole
// package using its sealed Snapshot binding, or legacy source coordinates.
// Content drift is assessed later; neither a subset nor a caller's explicitly
// different scope is inferred away.
func SelectFinalizedTranslationPackage(root string, c *Catalog, locale, snapshot string) (*Catalog, bool, error) {
	if c == nil {
		return nil, false, fmt.Errorf("translation catalog required")
	}
	if err := ValidateLocaleName(locale); err != nil {
		return nil, false, err
	}
	if err := validateSnapshotID(snapshot); err != nil {
		return nil, false, err
	}
	b, err := readCompatibilityFile(root, "data/quality-check-snapshots/"+locale+"/"+snapshot+"/finalization.json")
	if err != nil {
		return nil, false, err
	}
	var f QualityCheckFinalization
	if err := contentidentity.StrictJSON(b, &f); err != nil {
		return nil, false, err
	}
	if f.Locale != locale || f.SnapshotID != snapshot {
		return nil, false, fmt.Errorf("finalization diagnostic scope mismatch")
	}
	// Finalization seals the Snapshot, including its explicit package authority.
	// Source coordinates alone cannot distinguish representation predecessors
	// whose physical Page and kind are unchanged.
	s, err := readQualityCheckSnapshot(root, locale, snapshot, false)
	if err != nil {
		return nil, false, err
	}
	manifestPath := "data/quality-check-snapshots/" + locale + "/" + snapshot + "/manifest.json"
	manifest, err := readCompatibilityFile(root, manifestPath)
	if err != nil {
		return nil, false, err
	}
	var bound QualityCheckSnapshotManifest
	if f.SnapshotManifestPath != manifestPath || sum(manifest) != f.SnapshotManifestSHA256 {
		return nil, false, fmt.Errorf("finalization diagnostic Snapshot binding mismatch")
	}
	if err := contentidentity.StrictJSON(manifest, &bound); err != nil {
		return nil, false, err
	}
	if !reflect.DeepEqual(*s, bound) || len(f.Units) != len(s.Units) {
		return nil, false, fmt.Errorf("finalization diagnostic Snapshot binding mismatch")
	}
	units := make([]QualityCheckSnapshotUnit, len(f.Units))
	for i, u := range f.Units {
		if !qualityCheckSnapshotIdentityMatches(u.Snapshot, s.Units[i]) {
			return nil, false, fmt.Errorf("finalization diagnostic Unit binding mismatch")
		}
		units[i] = u.Snapshot
	}
	if s.ContentPackage != nil {
		return selectSnapshotPackageScope(root, c, s)
	}
	return selectTranslationPackageScope(c, units)
}

// An unfinalized successor has no finalization to select from. Its exact
// Snapshot supplies only complete source coordinates for scope selection;
// ordinary current readers must still verify all candidates and lineage.
func selectSnapshotTranslationPackage(root string, c *Catalog, locale, snapshot string) (*Catalog, bool, error) {
	if c == nil {
		return nil, false, fmt.Errorf("translation catalog required")
	}
	if err := ValidateLocaleName(locale); err != nil {
		return nil, false, err
	}
	if err := validateSnapshotID(snapshot); err != nil {
		return nil, false, err
	}
	s, err := readQualityCheckSnapshot(root, locale, snapshot, false)
	if err != nil {
		return nil, false, err
	}
	return selectSnapshotPackageScope(root, c, s)
}

// Both finalized and unfinalized readers consume the same frozen binding.
// Legacy manifests alone retain the coordinate-based ambiguity check.
func selectSnapshotPackageScope(root string, c *Catalog, s *QualityCheckSnapshotManifest) (*Catalog, bool, error) {
	if s.ContentPackage != nil {
		b := s.ContentPackage
		frozen, err := HistoricalTranslationPackageCatalog(c, b.ID, b.AuthoritySHA256)
		if err != nil {
			return nil, false, err
		}
		if len(c.TranslationScope) > 0 && !reflect.DeepEqual(c.TranslationScope, frozen.TranslationScope) {
			return nil, false, fmt.Errorf("explicit/Snapshot package scope mismatch")
		}
		if err := checkSnapshotPackageBinding(root, frozen, s, false); err != nil {
			return nil, false, err
		}
		return frozen, true, nil
	}
	return selectTranslationPackageScope(c, s.Units)
}

func selectTranslationPackageScope(c *Catalog, units []QualityCheckSnapshotUnit) (*Catalog, bool, error) {
	var selected *TranslationPackageDescriptor
	for _, p := range append(append([]TranslationPackageDescriptor{}, c.TranslationPackages...), c.HistoricalTranslationPackages...) {
		if len(p.Units) != len(units) {
			continue
		}
		match := true
		for i, u := range p.Units {
			old := units[i]
			if u.Unit.SourcePath != old.SourcePath || u.Unit.Kind != old.UnitKind {
				match = false
				break
			}
		}
		if match {
			if selected != nil {
				// A historical projection retains its discovery list. The same
				// exact descriptor in both lists is one authority, not ambiguity.
				if selected.ID == p.ID && selected.AuthoritySHA256 == p.AuthoritySHA256 && reflect.DeepEqual(selected.Units, p.Units) {
					continue
				}
				return nil, false, fmt.Errorf("ambiguous finalized package scope")
			}
			p := p
			selected = &p
		}
	}
	copy := *c
	if selected == nil {
		return &copy, false, nil
	}
	historical, err := HistoricalTranslationPackageCatalog(c, selected.ID, selected.AuthoritySHA256)
	if err != nil {
		return nil, false, err
	}
	copy = *historical
	if err := copy.UseTranslationPackage(selected.ID); err != nil {
		return nil, false, err
	}
	if len(c.TranslationScope) > 0 {
		if len(c.TranslationScope) != len(copy.TranslationScope) {
			return nil, false, fmt.Errorf("explicit/finalized package scope mismatch")
		}
		for i, id := range c.TranslationScope {
			if copy.TranslationScope[i] != id {
				return nil, false, fmt.Errorf("explicit/finalized package scope mismatch")
			}
		}
	}
	return &copy, true, nil
}
