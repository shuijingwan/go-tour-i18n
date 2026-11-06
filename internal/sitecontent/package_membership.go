package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const PackageMembershipPath = "data/content-package-memberships/current.json"
const PackageMembershipSchema = "go-learning/package-membership-successor/v1"

// Membership changes source governance, never the judgment of a language.
// Ordered complete predecessors supply append-only membership. Their original
// descriptors remain available exclusively to immutable historical readers.
type PackageMembership struct {
	Schema           string                  `json:"schema"`
	Package          string                  `json:"package"`
	Predecessors     []MembershipPredecessor `json:"predecessors"`
	PreviousRegistry Reference               `json:"previous_page_registry"`
	Registry         Reference               `json:"current_page_registry"`
	Identity         string                  `json:"identity_sha256"`
}
type MembershipPredecessor struct {
	Package   string      `json:"package"`
	Authority string      `json:"package_identity_sha256"`
	Sources   []Reference `json:"ordered_page_sources"`
}

func readPackageMembership(root string) (*PackageMembership, error) {
	b, err := readRegular(root, PackageMembershipPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ref Reference
	if err := StrictJSON(b, &ref); err != nil {
		return nil, err
	}
	var m PackageMembership
	if err := readReference(root, ref, &m); err != nil {
		return nil, err
	}
	copy := m
	copy.Identity = ""
	if m.Schema != PackageMembershipSchema || !component(m.Package) || len(m.Predecessors) < 2 || m.Identity != identity(copy) || ref.Path != "data/content-package-memberships/receipts/"+m.Identity+".json" {
		return nil, fmt.Errorf("invalid package membership successor")
	}
	return &m, nil
}

func applyPackageMembership(root string, base *Global) (*Global, error) {
	m, err := readPackageMembership(root)
	if err != nil || m == nil {
		return base, err
	}
	var before, after PageRegistry
	if err := readReference(root, m.PreviousRegistry, &before); err != nil {
		return nil, err
	}
	if err := readReference(root, m.Registry, &after); err != nil {
		return nil, err
	}
	currentRegistry, err := readRegular(root, PageRegistryPath)
	if err != nil || digest(currentRegistry) != m.Registry.SHA256 {
		return nil, fmt.Errorf("current Page registry differs from membership successor")
	}
	if ValidatePageRegistry(before) != nil || ValidatePageRegistry(after) != nil || len(after.Pages) <= len(before.Pages) || !reflect.DeepEqual(before.Pages, after.Pages[:len(before.Pages)]) {
		return nil, fmt.Errorf("membership requires exact append-only Page registry")
	}
	predecessors, paths := map[string]bool{}, map[string]bool{}
	next := *base
	next.Packages = nil
	next.HistoricalPackages = append([]Package{}, base.HistoricalPackages...)
	next.Sources = append([]Source{}, base.Sources...)
	next.Surfaces = append([]Surface{}, base.Surfaces...)
	next.Versions = append([]Version{}, base.Versions...)
	merged := Package{ID: m.Package, Activation: "atomic-locale-package", ParserContract: UnitContract, Routes: []string{}, Surfaces: []string{}, Dependencies: []Dependency{}}
	deps := map[string]Dependency{}
	ordered := []StablePage{}
	for pi, old := range m.Predecessors {
		p, err := packageByID(base, old.Package)
		if err != nil || p.Identity != old.Authority || predecessors[old.Package] || (pi == 0 && old.Package != m.Package) || len(old.Sources) == 0 {
			return nil, fmt.Errorf("membership predecessor authority mismatch: %s", old.Package)
		}
		predecessors[old.Package] = true
		next.HistoricalPackages = append(next.HistoricalPackages, p)
		want := map[string]Source{}
		for _, s := range base.Sources {
			if s.Package == old.Package && s.Kind == "page" {
				want[s.Path] = s
			}
		}
		if len(want) != len(old.Sources) {
			return nil, fmt.Errorf("membership predecessor exact-set mismatch")
		}
		for _, ref := range old.Sources {
			s, ok := want[ref.Path]
			if !ok || paths[ref.Path] || s.SHA256 != ref.SHA256 {
				return nil, fmt.Errorf("duplicate/wrong membership source")
			}
			b, err := readRegular(root, ref.Path)
			if err != nil || digest(b) != ref.SHA256 {
				return nil, fmt.Errorf("membership source stale: %s", ref.Path)
			}
			paths[ref.Path] = true
			ordered = append(ordered, StablePage{Index: len(ordered) + 1, Path: s.Path, Surface: s.Surface})
		}
		merged.Routes = append(merged.Routes, p.Routes...)
		merged.Surfaces = append(merged.Surfaces, p.Surfaces...)
		for _, d := range p.Dependencies {
			if previous, ok := deps[d.Path]; ok && previous != d {
				return nil, fmt.Errorf("conflicting membership dependencies")
			}
			deps[d.Path] = d
		}
	}
	if !reflect.DeepEqual(ordered, after.Pages) {
		return nil, fmt.Errorf("membership ordered registry exact-set mismatch")
	}
	for _, p := range base.Packages {
		if !predecessors[p.ID] {
			next.Packages = append(next.Packages, p)
		}
	}
	for i := range next.Sources {
		if predecessors[next.Sources[i].Package] {
			next.Sources[i].Package = m.Package
		}
	}
	for i := range next.Surfaces {
		if predecessors[next.Surfaces[i].Package] {
			next.Surfaces[i].Package = m.Package
		}
	}
	for i, v := range base.Versions {
		next.Versions[i].Packages = nil
		for _, id := range v.Packages {
			if id == m.Package || !predecessors[id] {
				next.Versions[i].Packages = append(next.Versions[i].Packages, id)
			}
		}
	}
	for _, d := range deps {
		merged.Dependencies = append(merged.Dependencies, d)
	}
	sort.Slice(merged.Dependencies, func(i, j int) bool { return merged.Dependencies[i].Path < merged.Dependencies[j].Path })
	sort.Strings(merged.Routes)
	sort.Strings(merged.Surfaces)
	merged.SourceIdentity = m.Identity
	merged.Identity = identity(merged)
	next.Packages = append(next.Packages, merged)
	var ref Reference
	pointer, err := readRegular(root, PackageMembershipPath)
	if err != nil {
		return nil, err
	}
	if err := StrictJSON(pointer, &ref); err != nil {
		return nil, err
	}
	var frozen PackageMembership
	if err := readReference(root, ref, &frozen); err != nil || !reflect.DeepEqual(*m, frozen) {
		return nil, fmt.Errorf("membership changed during projection")
	}
	next.Membership = &ref
	next.Identity = ""
	next.Identity = identity(next)
	return &next, ValidateGlobal(next)
}

// This transformation runs only after ordinary source adapters are registered.
// Identical Units are reused; no source/protected input/candidate is regenerated.
func attachPackageMembership(root string, c *i18n.Catalog, contexts ...*structuredSuccessorValidation) error {
	m, err := readPackageMembership(root)
	if err != nil || m == nil {
		return err
	}
	g, err := LoadCurrent(root, contexts...)
	if err != nil {
		return err
	}
	return projectPackageMembershipCatalog(root, c, m, g)
}

func projectPackageMembershipCatalog(root string, c *i18n.Catalog, m *PackageMembership, g *Global) error {
	p, err := packageByID(g, m.Package)
	if err != nil {
		return err
	}
	if g.Membership == nil || g.Membership.Path != "data/content-package-memberships/receipts/"+m.Identity+".json" {
		return fmt.Errorf("membership catalog authority mismatch")
	}
	pointer, err := readRegular(root, PackageMembershipPath)
	if err != nil {
		return err
	}
	var live Reference
	if StrictJSON(pointer, &live) != nil || live != *g.Membership {
		return fmt.Errorf("membership changed during catalog projection")
	}
	proof := &i18n.TranslationPackageMembership{}
	for _, ref := range []Reference{{Path: PackageMembershipPath, SHA256: digest(pointer)}, *g.Membership, m.PreviousRegistry, m.Registry, {Path: PageRegistryPath, SHA256: m.Registry.SHA256}} {
		proof.References = append(proof.References, i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256})
	}
	merged := i18n.TranslationPackageDescriptor{ID: m.Package, AuthoritySHA256: p.Identity, Membership: proof}
	removed := map[string]bool{}
	for _, old := range m.Predecessors {
		var found *i18n.TranslationPackageDescriptor
		for i := range c.TranslationPackages {
			d := &c.TranslationPackages[i]
			if d.ID == old.Package && d.AuthoritySHA256 == old.Authority {
				found = d
			}
		}
		if found == nil || len(found.Units) != len(old.Sources) {
			return fmt.Errorf("missing membership descriptor")
		}
		proof.Predecessors = append(proof.Predecessors, i18n.TranslationPackagePredecessor{ID: old.Package, AuthoritySHA256: old.Authority})
		for i, u := range found.Units {
			raw, err := readRegular(root, old.Sources[i].Path)
			if err != nil || digest(raw) != old.Sources[i].SHA256 || u.Unit.SourcePath != old.Sources[i].Path || u.Unit.SourceSHA256 != old.Sources[i].SHA256 || !bytes.Equal(u.Unit.Source, raw) {
				return fmt.Errorf("membership Unit/source mismatch")
			}
			u.StableIndex = len(merged.Units) + 1
			switch a := u.Adapter.(type) {
			case PageAdapter:
				a.Document.StableIndex = u.StableIndex
				u.Adapter = a
			case adoptedPageAdapter:
				a.Document.StableIndex = u.StableIndex
				u.Adapter = a
			}
			u.WorkingSet = fmt.Sprintf("page-batch-%d", (u.StableIndex-1)/30+1)
			u.WorkingSetLimit = 30
			merged.Units = append(merged.Units, u)
		}
		c.HistoricalTranslationPackages = append(c.HistoricalTranslationPackages, *found)
		removed[old.Package] = true
	}
	kept := []i18n.TranslationPackageDescriptor{}
	for _, d := range c.TranslationPackages {
		if !removed[d.ID] {
			kept = append(kept, d)
		}
	}
	c.TranslationPackages = append(kept, merged)
	memberIDs := map[string]bool{}
	for _, u := range merged.Units {
		memberIDs[u.Unit.ID] = true
	}
	adapted := []i18n.AdaptedTranslationUnit{}
	for _, u := range c.Adapted {
		if !memberIDs[u.Unit.ID] {
			adapted = append(adapted, u)
		}
	}
	c.Adapted = append(adapted, merged.Units...)
	return nil
}
