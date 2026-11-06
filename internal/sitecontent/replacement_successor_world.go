package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// The source provider, rather than shared QC code, proves how a frozen event
// descriptor reaches today's canonical descriptor. A shared path/SHA alone is
// never authority to substitute one representation for another.
func shellReplacementSuccessorWorld(root string, current, event *i18n.Catalog) ([]i18n.GlossaryArchiveReference, error) {
	var refs []i18n.GlossaryArchiveReference
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		refs, err = shellReplacementSuccessorWorldAt(root, current, event, g, v)
		return err
	})
	return refs, err
}

func shellReplacementSuccessorWorldAt(root string, current, event *i18n.Catalog, g *Global, v *structuredSuccessorValidation) ([]i18n.GlossaryArchiveReference, error) {
	head, ref, err := currentShellRepresentation(root, g, v, false)
	if err != nil {
		return nil, err
	}
	if head == nil || ref == nil || len(event.TranslationPackages) != 1 {
		return nil, fmt.Errorf("historical reconciliation requires an exact canonical representation successor")
	}
	d := event.TranslationPackages[0]
	if d.ID != head.Basis.Package.ID || !shellReplacementPackageAncestor(g, head.Basis.Package, d.AuthoritySHA256) || len(d.Units) != 1 {
		return nil, fmt.Errorf("historical reconciliation package authority mismatch")
	}
	qualified := representationPackage(head.Basis.Package, *ref)
	var terminal *i18n.TranslationPackageDescriptor
	for i := range current.TranslationPackages {
		p := &current.TranslationPackages[i]
		if p.ID == d.ID {
			if terminal != nil || p.AuthoritySHA256 != qualified.Identity || len(p.Units) != 1 {
				return nil, fmt.Errorf("ambiguous/wrong current representation package")
			}
			terminal = p
		}
	}
	if terminal == nil || !shellReplacementDescriptorMatches(terminal.Units[0], head.New.Source) {
		return nil, fmt.Errorf("current representation descriptor does not match canonical successor")
	}
	refs := []Reference{}
	appendRef := func(r Reference) { refs = append(refs, r) }
	pointer, err := ReferenceFile(root, ShellRepresentationCurrent)
	if err != nil {
		return nil, err
	}
	appendRef(pointer)
	additional, err := shellReattestationContextReferencesAt(root, g, head, v)
	if err != nil {
		return nil, err
	}
	refs = append(refs, additional...)
	// currentShellRepresentation already verified every immutable edge, exact
	// reconstruction, dependency binding, cycle/depth and the selected head.
	// Find the event descriptor in that verified chain, never by source alone.
	node, nodeRef := head, *ref
	found := false
	for depth := 0; ; depth++ {
		if depth > 32 {
			return nil, fmt.Errorf("historical representation successor depth")
		}
		appendRef(nodeRef)
		if shellReplacementDescriptorMatches(d.Units[0], node.Old.Source) {
			found = true
		}
		if node.Predecessor == nil {
			break
		}
		nodeRef = *node.Predecessor
		var prior ShellRepresentationSuccessor
		if err := v.guard.readReference(root, nodeRef, &prior); err != nil {
			return nil, err
		}
		node = &prior
	}
	if !found {
		return nil, fmt.Errorf("reconciliation intermediate descriptor absent from canonical successor lineage")
	}
	refs = append(refs, head.Basis.Source, head.Basis.Adoption, head.Basis.Foundation)
	refs = append(refs, head.Basis.SourceHeads...)
	out := []i18n.GlossaryArchiveReference{}
	seen := map[Reference]bool{}
	for _, r := range refs {
		if !seen[r] {
			out = append(out, i18n.GlossaryArchiveReference{Path: r.Path, SHA256: r.SHA256})
			seen[r] = true
		}
	}
	return out, nil
}

// A structured-source successor may advance the enclosing source package
// while leaving this physical Page's raw bytes and representation unchanged.
// Consume only exact ancestors already validated by loadSourceCurrent, using
// the same source-only contract equality as historical package registration.
func shellReplacementPackageAncestor(g *Global, current Package, authority string) bool {
	if authority == current.Identity {
		return true
	}
	found := false
	for _, prior := range g.HistoricalPackages {
		if prior.ID != current.ID || prior.Identity != authority {
			continue
		}
		p := prior
		p.Identity = ""
		if identity(p) != authority {
			return false
		}
		p.Identity, p.SourceIdentity = current.Identity, current.SourceIdentity
		if !reflect.DeepEqual(p, current) {
			return false
		}
		found = true
	}
	return found
}

func shellReplacementDescriptorMatches(u i18n.AdaptedTranslationUnit, s ShellLanguageSource) bool {
	a, ok := u.Adapter.(ShellPageAdapter)
	return ok && s.Page != nil && len(s.Page.Units) == 1 && u.Unit.ID == s.Page.Units[0].ID &&
		u.Unit.Kind == i18n.UnitKindPage && u.Unit.SourcePath == s.Path && u.Unit.SourceSHA256 == s.SourceSHA &&
		u.Route == s.Page.Route && u.StableIndex == s.Page.StableIndex && digest(u.Unit.Source) == s.SourceSHA &&
		bytes.Equal(a.Raw, u.Unit.Source) && reflect.DeepEqual(a.Source, s)
}
