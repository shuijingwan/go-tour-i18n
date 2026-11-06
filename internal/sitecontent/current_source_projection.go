package sitecontent

import (
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// The existing source-proof owner spans LoadCurrent and corpus reconstruction.
// This node is the complete source projection, not a package/currentness PASS.
// Historical source/catalog observation still runs before every entry; current
// membership and shell representation checks still run after this node.
type currentSourceProjection struct {
	base  string
	graph *Global
}

// A single installed-corpus reconstruction is a read-only phase: it publishes
// nothing and exposes no intermediate authority to its caller. Nested readers
// continue binding their exact references. The owner checks the entire observed
// world freshly on entry and before any successful return, including nil/fallback
// results. It never carries a currentness verdict into another phase/build.
func (v *structuredSuccessorValidation) installedCorpusObservation(work func() (*CorpusSuccessor, error)) (*CorpusSuccessor, error) {
	if v.sharedSourceObservation {
		// The explicit outer readonly bridge owner returns no intermediate
		// authority and closes this entire captured world before its result
		// escapes. Nested readers still bind exact bytes/heads/references.
		return work()
	}
	if err := v.aggregateVerify(); err != nil {
		return nil, err
	}
	graph, phase, readonly := v.currentSource, v.currentSourcePhase, v.readonlyOwner
	v.currentSource, v.currentSourcePhase, v.readonlyOwner = nil, true, true
	defer func() { v.currentSource, v.currentSourcePhase, v.readonlyOwner = graph, phase, readonly }()
	result, err := work()
	if err != nil {
		return nil, err
	}
	if err := v.aggregateVerify(); err != nil {
		return nil, err
	}
	return result, nil
}

func (v *structuredSuccessorValidation) projectCurrentSource(root string, base *Global) (*Global, error) {
	if root != v.guard.root {
		return nil, fmt.Errorf("current source observation root changed")
	}
	// Freeze discovery heads BEFORE reconstruction. Directory membership binds
	// new authorities as well as changes to an already selected pointer.
	for _, dir := range []string{StructuredSourceSuccessorRoot + "current", PagePackageAdoptionRoot + "current"} {
		if err := v.guard.directory(dir); err != nil {
			return nil, err
		}
	}
	for _, path := range []string{ShellAdoptionCurrent, StructuredSourceReplacementPending, i18n.GlossaryCorpusPath} {
		if err := v.guard.optional(path); err != nil {
			return nil, err
		}
	}
	if p := v.currentSource; p != nil && !v.disabled {
		if p.base != base.Identity {
			return nil, fmt.Errorf("current source historical basis changed")
		}
		// Match the existing owner's policy: eager readers check immediately;
		// the installed-corpus phase closes the full guard before returning any
		// authority. Reference/discovery reads above remain fresh in both modes.
		if err := v.verifyObserved(); err != nil {
			return nil, err
		}
		v.countHistorical("current-source-projection-reuse")
		return cloneCurrentSource(p.graph), nil
	}
	heads := *base
	heads.StructuredSuccessors = append([]Reference(nil), base.StructuredSuccessors...)
	heads.PackageAdoptions = append([]Reference(nil), base.PackageAdoptions...)
	if !v.guard.absent[ShellAdoptionCurrent] {
		var ref Reference
		if err := v.readSourceHead(root, ShellAdoptionCurrent, &ref); err != nil {
			return nil, err
		}
		heads.ShellAdoption = &ref
	}
	for _, head := range []struct {
		dir  string
		refs *[]Reference
	}{{StructuredSourceSuccessorRoot + "current", &heads.StructuredSuccessors}, {PagePackageAdoptionRoot + "current", &heads.PackageAdoptions}} {
		for _, name := range v.guard.dirs[head.dir] {
			var ref Reference
			if err := v.readSourceHead(root, head.dir+"/"+name, &ref); err != nil {
				return nil, err
			}
			*head.refs = append(*head.refs, ref)
		}
	}
	g, err := rebuildCurrentSource(root, base, v)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(heads.ShellAdoption, g.ShellAdoption) || !sameSourceHeads(heads.StructuredSuccessors, g.StructuredSuccessors) || !sameSourceHeads(heads.PackageAdoptions, g.PackageAdoptions) {
		return nil, fmt.Errorf("current source discovery changed during construction")
	}
	// Capture the complete typed source closure, including the bodies/proposals
	// declared by adoption receipts. An individual package projection cannot
	// populate this node. Failed construction/guards are never retained.
	if err := v.observe(g); err != nil {
		return nil, err
	}
	if g.ShellAdoption != nil {
		var a ShellAdoption
		if err := v.guard.readReference(root, *g.ShellAdoption, &a); err != nil {
			return nil, err
		}
		if err := v.observe(a); err != nil {
			return nil, err
		}
		var foundation ShellSuccessorAuthority
		if err := v.guard.readReference(root, a.Foundation, &foundation); err != nil {
			return nil, err
		}
		if err := v.observe(foundation); err != nil {
			return nil, err
		}
	}
	for _, ref := range g.PackageAdoptions {
		var a PagePackageAdoption
		if err := v.guard.readReference(root, ref, &a); err != nil {
			return nil, err
		}
		if err := v.observe(a); err != nil {
			return nil, err
		}
	}
	if err := v.verifyObserved(); err != nil {
		return nil, err
	}
	if !v.disabled {
		v.currentSource = &currentSourceProjection{base.Identity, cloneCurrentSource(g)}
	}
	return g, nil
}

func (v *structuredSuccessorValidation) readSourceHead(root, path string, ref *Reference) error {
	b, err := v.guard.readRegular(root, path)
	if err != nil {
		return err
	}
	return StrictJSON(b, ref)
}

func sameSourceHeads(a, b []Reference) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func cloneCurrentSource(g *Global) *Global {
	// Preserve nil/empty distinctions and private ownership, as in the existing
	// source-owned projection copier; no callers can mutate the retained node.
	return cloneReplacementValue(reflect.ValueOf(g)).Interface().(*Global)
}
