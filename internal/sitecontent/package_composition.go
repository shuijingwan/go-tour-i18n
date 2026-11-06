package sitecontent

import (
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type PackageCompositionPlan struct {
	Membership  Reference                           `json:"membership_successor"`
	Composition *i18n.TranslationPackageComposition `json:"composition"`
	Mutation    string                              `json:"mutation"`
}

// Readonly preflight: neither a new A nor a persisted authorization. A future
// ordinary package closure consumes the same finalization references and
// rebuilds this proof, including the full current eligibility gates.
func InspectPackageComposition(root, locale, pkg string, refs []Reference) (*PackageCompositionPlan, error) {
	v := newClosureValidation(root)
	if err := v.currentInputs(locale); err != nil {
		return nil, err
	}
	inputs, err := v.compositionInputs(root, pkg, refs)
	if err != nil {
		return nil, err
	}
	c, err := v.packageCatalog(pkg)
	if err != nil {
		return nil, err
	}
	p, _, err := v.composePackage(root, c, locale, inputs)
	if err != nil {
		return nil, err
	}
	g, err := v.loadCurrent(root)
	if err != nil {
		return nil, err
	}
	if err := v.verify(); err != nil {
		return nil, err
	}
	return &PackageCompositionPlan{Membership: *g.Membership, Composition: p, Mutation: "none"}, nil
}

func (v *closureValidation) compositionInputs(root, pkg string, refs []Reference) ([]i18n.TranslationFinalizationInput, error) {
	g, err := v.loadCurrent(root)
	if err != nil {
		return nil, err
	}
	if g.Membership == nil {
		return nil, fmt.Errorf("composition requires current membership successor")
	}
	var m PackageMembership
	if err := v.readReference(root, *g.Membership, &m); err != nil {
		return nil, err
	}
	if m.Package != pkg || len(refs) != len(m.Predecessors) {
		return nil, fmt.Errorf("composition predecessor exact-set mismatch")
	}
	inputs := []i18n.TranslationFinalizationInput{}
	for i, old := range m.Predecessors {
		inputs = append(inputs, i18n.TranslationFinalizationInput{Package: old.Package, PackageAuthority: old.Authority, Finalization: i18n.GlossaryArchiveReference{Path: refs[i].Path, SHA256: refs[i].SHA256}})
	}
	return inputs, nil
}

func (v *closureValidation) composePackage(root string, c *i18n.Catalog, locale string, inputs []i18n.TranslationFinalizationInput) (*i18n.TranslationPackageComposition, map[string][]byte, error) {
	refs := []Reference{}
	for _, in := range inputs {
		refs = append(refs, Reference{Path: in.Finalization.Path, SHA256: in.Finalization.SHA256})
	}
	pkg := ""
	for _, d := range c.TranslationPackages {
		if len(d.Units) == len(c.TranslationScope) {
			match := true
			for i, u := range d.Units {
				if u.Unit.ID != c.TranslationScope[i] {
					match = false
				}
			}
			if match {
				if pkg != "" {
					return nil, nil, fmt.Errorf("ambiguous current composition descriptor")
				}
				pkg = d.ID
			}
		}
	}
	expected, err := v.compositionInputs(root, pkg, refs)
	if err != nil || !reflect.DeepEqual(expected, inputs) {
		return nil, nil, fmt.Errorf("composition frozen descriptor binding mismatch: %v", err)
	}
	p, files, err := i18n.ComposeFinalizedTranslationPackage(root, c, locale, inputs)
	if err != nil {
		return nil, nil, err
	}
	for _, a := range p.Assessments {
		for _, r := range append(append([]i18n.GlossaryArchiveReference{}, a.Historical.References...), a.Glossary, a.UnifiedReview) {
			if err := v.bind(Reference{Path: r.Path, SHA256: r.SHA256}); err != nil {
				return nil, nil, err
			}
		}
	}
	return p, files, nil
}

func closurePageFinalizations(c *PackageClosure) []Reference {
	refs := []Reference{}
	if c.PageFinalization != nil {
		refs = append(refs, *c.PageFinalization)
	}
	for _, in := range c.PageComposition {
		refs = append(refs, Reference{Path: in.Finalization.Path, SHA256: in.Finalization.SHA256})
	}
	return refs
}

func BuildComposedPackageClosure(root, locale, pkg string, finalizations, structured []Reference) (*PackageClosure, map[string][]byte, error) {
	v := newClosureValidation(root)
	if err := v.currentInputs(locale); err != nil {
		return nil, nil, err
	}
	inputs, err := v.compositionInputs(root, pkg, finalizations)
	if err != nil {
		return nil, nil, err
	}
	c, files, err := v.buildPackageClosureInputs(root, locale, pkg, nil, structured, inputs)
	if err == nil {
		err = v.verify()
	}
	return c, files, err
}

// Same immutable closure publisher; no composition-specific state machine.
func FinalizeComposedPackageClosure(root, locale, pkg string, finalizations, structured []Reference) (*PackageClosure, string, error) {
	c, _, err := BuildComposedPackageClosure(root, locale, pkg, finalizations, structured)
	if err != nil {
		return nil, "", err
	}
	p := workflowPath(locale, pkg, "closure", c.Identity)
	return c, p, saveImmutable(root, p, c)
}
