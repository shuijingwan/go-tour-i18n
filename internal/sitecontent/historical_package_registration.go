package sitecontent

import (
	"fmt"
	"os"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// A structured-only source successor changes the enclosing package identity,
// not its formal Page inventory. Preserve the exact validated predecessor
// descriptor for historical readers; never register it as a current scope.
func attachStructuredHistoricalPackages(root string, c *i18n.Catalog, contexts ...*structuredSuccessorValidation) error {
	g, err := loadSourceCurrent(root, contexts...)
	if err != nil || len(g.StructuredSuccessors) == 0 {
		return err
	}
	base, err := adoptionBase(root, contexts...)
	if err != nil {
		return err
	}
	// Only exact packages named by validated structured ancestry qualify.
	// HistoricalPackages also contains unrelated membership/adoption history.
	var predecessors []Package
	for _, ref := range g.StructuredSuccessors {
		a, err := checkStructuredSourceSuccessor(root, ref, 0, contexts...)
		if err != nil {
			return err
		}
		for a.Replacement != nil {
			predecessors = append(predecessors, a.Replacement.PredecessorPackage)
			a, err = checkStructuredSourceSuccessor(root, a.Replacement.Predecessor, 0, contexts...)
			if err != nil {
				return err
			}
		}
	}
	// Canonical representation successors have a separate package authority.
	// Structured-only history still registers the exact original Page inventory.
	if _, err := readRegular(root, ShellRepresentationCurrent); err == nil {
		v := successorValidation(root, contexts)
		r, _, err := currentShellRepresentation(root, g, v, false)
		if err != nil {
			return err
		}
		for r.Predecessor != nil {
			r, err = checkShellRepresentation(root, g, *r.Predecessor, v, map[Reference]bool{}, 0)
			if err != nil {
				return err
			}
		}
		shadow := *c
		shadow.TranslationPackages = append([]i18n.TranslationPackageDescriptor{}, c.TranslationPackages...)
		shadow.HistoricalTranslationPackages = nil
		for i, d := range shadow.TranslationPackages {
			if d.ID != r.Basis.Package.ID {
				continue
			}
			old := r.Old.Source
			u := old.Page.Units[0]
			raw, err := readRegular(root, old.Path)
			if err != nil {
				return err
			}
			d.AuthoritySHA256 = r.Basis.Package.Identity
			d.Units = []i18n.AdaptedTranslationUnit{{Unit: i18n.TranslationUnit{ID: u.ID, Kind: i18n.UnitKindPage, SourcePath: old.Path, Source: raw, SourceSHA256: old.SourceSHA}, Adapter: ShellPageAdapter{old, raw}, StableIndex: 1, Route: "/", WorkingSet: "homepage-initial", WorkingSetLimit: 1}}
			shadow.TranslationPackages[i] = d
		}
		if err := registerStructuredHistoricalPackages(base, g, &shadow, predecessors...); err != nil {
			return err
		}
		c.HistoricalTranslationPackages = append(c.HistoricalTranslationPackages, shadow.HistoricalTranslationPackages...)
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return registerStructuredHistoricalPackages(base, g, c, predecessors...)
}

func registerStructuredHistoricalPackages(base, current *Global, c *i18n.Catalog, ancestors ...Package) error {
	var registrations []i18n.TranslationPackageDescriptor
	for _, d := range c.TranslationPackages {
		before, err := packageByID(base, d.ID)
		if err != nil {
			continue
		} // A later source adoption has no predecessor here.
		after, err := packageByID(current, d.ID)
		if err != nil || after.Identity != d.AuthoritySHA256 {
			return fmt.Errorf("current descriptor authority mismatch: %s", d.ID)
		}
		if before.Identity == after.Identity {
			continue
		}
		registered := false
		for _, p := range current.HistoricalPackages {
			registered = registered || reflect.DeepEqual(p, before)
		}
		// Only source identity may change in this source-only transition.
		want := before
		want.Identity, want.SourceIdentity = after.Identity, after.SourceIdentity
		if !registered || !reflect.DeepEqual(want, after) {
			return fmt.Errorf("unproven structured package predecessor: %s", d.ID)
		}
		for _, u := range d.Units {
			found := false
			for _, s := range base.Sources {
				if s.Package != d.ID || s.Path != u.Unit.SourcePath {
					continue
				}
				for _, now := range current.Sources {
					if reflect.DeepEqual(s, now) && s.SHA256 == u.Unit.SourceSHA256 && digest(u.Unit.Source) == s.SHA256 {
						found = true
					}
				}
			}
			if !found {
				return fmt.Errorf("structured successor changed formal Page source: %s", u.Unit.SourcePath)
			}
		}
		seen := map[string]Package{}
		for _, p := range append([]Package{before}, ancestors...) {
			if p.ID != d.ID {
				continue
			}
			if prior, ok := seen[p.Identity]; ok {
				if !reflect.DeepEqual(prior, p) {
					return fmt.Errorf("ambiguous structured package predecessor: %s", d.ID)
				}
				continue
			}
			found := false
			for _, historical := range current.HistoricalPackages {
				if historical.ID == p.ID && historical.Identity == p.Identity {
					if !reflect.DeepEqual(historical, p) {
						return fmt.Errorf("conflicting structured package predecessor: %s", d.ID)
					}
					found = true
				}
			}
			if !found {
				return fmt.Errorf("missing structured package predecessor: %s", d.ID)
			}
			contract := p
			contract.Identity, contract.SourceIdentity = after.Identity, after.SourceIdentity
			if !reflect.DeepEqual(contract, after) {
				return fmt.Errorf("historical structured package contract mismatch")
			}
			historical := d
			historical.AuthoritySHA256 = p.Identity
			registrations = append(registrations, historical)
			seen[p.Identity] = p
		}
	}
	c.HistoricalTranslationPackages = append(c.HistoricalTranslationPackages, registrations...)
	return nil
}
