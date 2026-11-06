package sitecontent

import (
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Reuses the existing corpus-successor CAS/archive/pending marker. Adoption is
// source governance; it deliberately makes the old full Review non-current.
func BuildAdoptedCorpusSuccessor(root string) (*CorpusSuccessor, error) {
	old, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		// A recorded consumed replacement intentionally retires one exact
		// corpus pointer reference. Admit only its byte-exact consumed corpus;
		// all other references are checked by the resulting projection below.
		originalErr := err
		old, err = readProjectionCorpus(root, i18n.GlossaryCorpusPath)
		if err != nil {
			return nil, originalErr
		}
		b, err := readRegular(root, i18n.GlossaryCorpusPath)
		if err != nil {
			return nil, err
		}
		g, err := loadSourceCurrent(root)
		if err != nil {
			return nil, err
		}
		allowed := false
		for _, ref := range g.StructuredSuccessors {
			a, err := CheckStructuredSourceSuccessor(root, ref)
			if err != nil {
				return nil, err
			}
			if a.Replacement != nil && a.Replacement.CorpusSHA == digest(b) {
				allowed = true
			}
		}
		if !allowed {
			return nil, originalErr
		}
	}
	b, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		return nil, err
	}
	return buildAdoptedCorpusSuccessorFrom(root, old, b)
}

func buildAdoptedCorpusSuccessorFrom(root string, old *i18n.GlossarySourceCorpus, oldBytes []byte, contexts ...*structuredSuccessorValidation) (*CorpusSuccessor, error) {
	g, err := loadSourceCurrent(root, contexts...)
	if err != nil {
		return nil, err
	}
	return buildAdoptedCorpusAtSources(root, g, old, oldBytes, false, contexts...)
}

func buildAdoptedCorpusAtSources(root string, g *Global, old *i18n.GlossarySourceCorpus, oldBytes []byte, historical bool, contexts ...*structuredSuccessorValidation) (*CorpusSuccessor, error) {
	refs := append(append([]Reference{}, g.PackageAdoptions...), g.StructuredSuccessors...)
	if len(refs) == 0 {
		return nil, fmt.Errorf("explicit current source adoption required")
	}
	current, err := projectAdoptedCorpusAtSources(root, g, old, historical, contexts...)
	if err != nil {
		return nil, err
	}
	p := &CorpusSuccessor{Schema: CorpusSuccessorSchema, State: "pending", ProjectionAuthorities: refs, Predecessor: Reference{i18n.GlossaryCorpusPath, digest(oldBytes)}, PredecessorIdentity: old.Identity,
		Additions: []i18n.LanguageContributor{}, Retirements: []i18n.LanguageContributor{}, NotPresent: []string{}, Result: *current}
	before, after := map[string]i18n.LanguageContributor{}, map[string]i18n.LanguageContributor{}
	for _, c := range old.Contributors {
		before[c.ID] = c
	}
	for _, c := range current.Contributors {
		after[c.ID] = c
	}
	unchanged := []i18n.LanguageContributor{}
	for _, c := range old.Contributors {
		if next, ok := after[c.ID]; ok && reflect.DeepEqual(c, next) {
			unchanged = append(unchanged, c)
		} else {
			p.Retirements = append(p.Retirements, c)
		}
	}
	for _, c := range current.Contributors {
		if previous, ok := before[c.ID]; !ok || !reflect.DeepEqual(previous, c) {
			p.Additions = append(p.Additions, c)
			p.ProposedContexts += len(c.Contexts)
		}
		p.NetContexts += len(c.Contexts)
	}
	for _, c := range old.Contributors {
		p.NetContexts -= len(c.Contexts)
	}
	p.UnchangedCount, p.UnchangedIdentity = len(unchanged), identity(unchanged)
	p.Changed = current.Identity != old.Identity
	if !p.Changed {
		return nil, fmt.Errorf("source adoption corpus has no delta")
	}
	b, err := Encode(current)
	if err != nil {
		return nil, err
	}
	p.ResultSHA = digest(b)
	p.Identity = identity(*p)
	return p, nil
}

// Incremental source governance preserves the frozen unaffected projection,
// rather than silently re-extracting the historical corpus with a new parser.
// Every old and new physical source reference is still checked afresh.
func projectAdoptedCorpus(root string, g *Global, old *i18n.GlossarySourceCorpus) (*i18n.GlossarySourceCorpus, error) {
	return projectAdoptedCorpusAtSources(root, g, old, false)
}

func projectAdoptedCorpusAtSources(root string, g *Global, old *i18n.GlossarySourceCorpus, historical bool, contexts ...*structuredSuccessorValidation) (*i18n.GlossarySourceCorpus, error) {
	v := successorValidation(root, contexts)
	current := *old
	current.Contributors = append([]i18n.LanguageContributor{}, old.Contributors...)
	current.PackageAuthorities = append([]i18n.GlossaryCorpusPackageAuthority(nil), old.PackageAuthorities...)
	refs := map[string]string{}
	add := func(ref Reference) error {
		if prior, ok := refs[ref.Path]; ok && prior != ref.SHA256 {
			return fmt.Errorf("source projection reference conflict: %s", ref.Path)
		}
		refs[ref.Path] = ref.SHA256
		return nil
	}
	for _, ref := range old.Sources {
		refs[ref.Path] = ref.SHA256
	}
	manifest, err := loadSourceManifest(root)
	if err == nil {
		delete(refs, SnapshotPath)
		ref, err := ReferenceFile(root, SourceManifestPath)
		if err != nil {
			return nil, err
		}
		if err = add(ref); err != nil {
			return nil, err
		}
		for _, name := range manifest.Legacy.Paths {
			ref, err := ReferenceFile(root, adoptedSourcePath(name))
			if err != nil {
				return nil, err
			}
			if err = add(ref); err != nil {
				return nil, err
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	for _, ref := range g.PackageAdoptions {
		a, err := CheckPagePackageAdoption(root, ref, v)
		if err != nil {
			return nil, err
		}
		exists := false
		for _, binding := range current.PackageAuthorities {
			if binding.Package == a.Package.ID {
				if binding.Authority != (i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256}) {
					return nil, fmt.Errorf("registered corpus package authority conflict")
				}
				exists = true
			}
		}
		binding := i18n.GlossaryCorpusPackageAuthority{Package: a.Package.ID, Authority: i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256}, SourcePaths: []string{}}
		for _, d := range a.Documents {
			binding.SourcePaths = append(binding.SourcePaths, d.Path)
			if exists {
				continue
			}
			c, err := projectedDocumentContributor(d, a.Package.ID)
			if err != nil {
				return nil, err
			}
			if current.Schema == i18n.GlossaryCorpusSchemaV2 {
				c.Purpose = i18n.ContributorPurposeActiveContent
				c.SourceSHA = i18n.ContributorIdentity(c)
			}
			current.Contributors = append(current.Contributors, c)
		}
		if !exists {
			sort.Strings(binding.SourcePaths)
			current.PackageAuthorities = append(current.PackageAuthorities, binding)
		}
		for _, name := range []string{ref.Path, pagePackageCurrent(a.Package.ID), a.Proposal.Path, a.FrozenManifest.Path} {
			r, err := ReferenceFile(root, name)
			if err != nil {
				return nil, err
			}
			if err = add(r); err != nil {
				return nil, err
			}
		}
		for _, s := range a.Sources {
			if err = add(Reference{s.Path, s.SHA256}); err != nil {
				return nil, err
			}
		}
	}
	frozenPointers := map[string][]byte{}
	for _, ref := range g.StructuredSuccessors {
		a, err := checkStructuredSourceSuccessor(root, ref, 0, v)
		if err != nil {
			return nil, err
		}
		id := "source:" + a.Previous.Path
		index := -1
		for i, c := range current.Contributors {
			if c.ID == id {
				index = i
			}
		}
		if index < 0 {
			return nil, fmt.Errorf("source successor corpus predecessor missing")
		}
		c := current.Contributors[index]
		if c.Package != a.Package || c.Path != a.Previous.Path || c.Parser != UnitContract || !c.Structured || c.Route != a.Old.Route {
			return nil, fmt.Errorf("source successor corpus predecessor binding mismatch")
		}
		// An already registered exact source authority is retained, not replayed.
		if _, registered := refs[ref.Path]; !registered {
			if a.Replacement != nil {
				projection, err := projectedDocumentContributor(a.Current, a.Package)
				if err != nil {
					return nil, err
				}
				projection.Purpose = c.Purpose
				// A consumed replacement establishes a new semantic projection
				// even when a reordering leaves terminology text unchanged.
				for i := range projection.Contexts {
					projection.Contexts[i].ID = a.Identity + "/" + projection.Contexts[i].ID
				}
				projection.SourceSHA = i18n.ContributorIdentity(projection)
				current.Contributors[index] = projection
			} else {
				inserted := a.Current
				inserted.Units = nil
				wanted := map[string]bool{}
				for _, id := range a.AddedIDs {
					wanted[id] = true
				}
				for _, u := range a.Current.Units {
					if wanted[u.ID] {
						inserted.Units = append(inserted.Units, u)
					}
				}
				addition, err := projectedDocumentContributor(inserted, a.Package)
				if err != nil {
					return nil, err
				}
				c.Contexts = append([]i18n.LanguageContext{}, c.Contexts...)
				seen := map[string]bool{}
				for _, ctx := range c.Contexts {
					seen[ctx.ID] = true
				}
				for _, ctx := range addition.Contexts {
					for seen[ctx.ID] {
						ctx.ID += "x"
					}
					seen[ctx.ID] = true
					c.Contexts = append(c.Contexts, ctx)
				}
				sort.Slice(c.Contexts, func(i, j int) bool { return c.Contexts[i].ID < c.Contexts[j].ID })
				c.SourceSHA = i18n.ContributorIdentity(c)
				current.Contributors[index] = c
			}
		}
		pointerName := structuredSourceCurrent(a.Previous.Path)
		pointerBytes, _ := Encode(ref)
		frozenPointers[pointerName] = pointerBytes
		if prior, ok := refs[pointerName]; ok && prior != digest(pointerBytes) {
			if a.Replacement == nil || prior != a.Replacement.PriorPointer.SHA256 {
				return nil, fmt.Errorf("source projection prior pointer mismatch")
			}
			delete(refs, pointerName)
		}
		if err := add(Reference{pointerName, digest(pointerBytes)}); err != nil {
			return nil, err
		}
		for _, name := range []string{ref.Path, a.Proposal.Path, a.Body.Path} {
			r, err := ReferenceFile(root, name)
			if err != nil {
				return nil, err
			}
			if err = add(r); err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(current.Contributors, func(i, j int) bool { return current.Contributors[i].ID < current.Contributors[j].ID })
	sort.Slice(current.PackageAuthorities, func(i, j int) bool {
		return current.PackageAuthorities[i].Package < current.PackageAuthorities[j].Package
	})
	current.Sources = []i18n.GlossaryArchiveReference{}
	for name, sha := range refs {
		b, err := readRegular(root, name)
		if historical {
			if frozen, ok := frozenPointers[name]; ok {
				b, err = frozen, nil
			}
		}
		if err != nil || digest(b) != sha {
			return nil, fmt.Errorf("source projection dependency drift: %s", name)
		}
		current.Sources = append(current.Sources, i18n.GlossaryArchiveReference{Path: name, SHA256: sha})
	}
	sort.Slice(current.Sources, func(i, j int) bool { return current.Sources[i].Path < current.Sources[j].Path })
	current.Identity = i18n.CorpusIdentity(current)
	return &current, i18n.ValidateGlossaryCorpus(&current)
}

func AdoptCorpusProjection(root, prior string, apply bool) (*CorpusSuccessor, error) {
	p, err := BuildAdoptedCorpusSuccessor(root)
	if err != nil {
		return nil, err
	}
	if prior != p.Predecessor.SHA256 || !validSHA(prior) {
		return nil, fmt.Errorf("exact predecessor corpus CAS required")
	}
	if apply {
		fresh, err := BuildAdoptedCorpusSuccessor(root)
		if err != nil || !reflect.DeepEqual(p, fresh) {
			return nil, fmt.Errorf("concurrent source projection mutation")
		}
		if err := installCorpusSuccessor(root, p, prior); err != nil {
			return nil, err
		}
	}
	return p, nil
}
