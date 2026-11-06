package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"os"
	"sort"
	"strings"
)

// BuildUnifiedCorpus consumes the registered source closure through each
// surface's parser. The corpus identity excludes raw machine-only source bytes.
func BuildUnifiedCorpus(root string, contexts ...*structuredSuccessorValidation) (*i18n.GlossarySourceCorpus, error) {
	manifestExists, snapshotExists, err := terminologyAuthorityFilesExist(root)
	if err != nil {
		return nil, err
	}
	if manifestExists != snapshotExists {
		return nil, fmt.Errorf("terminology discovery authority is incomplete")
	}
	if manifestExists {
		authority, snapshot, err := LoadTerminologyAuthority(root)
		if err != nil {
			return nil, err
		}
		return currentCorpusProjection(root, authority, snapshot, contexts...)
	}
	return currentCorpusProjection(root, nil, nil, contexts...)
}

func buildUnifiedCorpus(root string, terminology *TerminologyDiscoveryAuthority, snapshot []byte) (*i18n.GlossarySourceCorpus, error) {
	g, err := loadSourceCurrent(root)
	if err != nil {
		return nil, err
	}
	cat, err := checkedCatalog(root)
	if err != nil {
		return nil, err
	}
	cs, err := i18n.TourLanguageContributors(root, cat)
	if err != nil {
		return nil, err
	}
	sources := map[string]bool{}
	candidateSources := map[string][]byte{}
	for _, c := range cs {
		sources[c.Path] = true
	}
	sources[GlobalPath] = true
	sources[SnapshotPath] = true
	if len(g.PackageAdoptions) > 0 || len(g.StructuredSuccessors) > 0 {
		manifest, err := loadSourceManifest(root)
		if err == nil {
			// New source authority uses the repo-local frozen files. Historical
			// ZIP references keep their original storage proof in old corpora.
			delete(sources, SnapshotPath)
			sources[SourceManifestPath] = true
			for _, name := range manifest.Legacy.Paths {
				sources[adoptedSourcePath(name)] = true
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	packages := []string{"site-v2-shell", "learn-docs-v1"}
	packageAuthorities := []i18n.GlossaryCorpusPackageAuthority{}
	for _, ref := range g.PackageAdoptions {
		a, err := CheckPagePackageAdoption(root, ref)
		if err != nil {
			return nil, err
		}
		packages = append(packages, a.Package.ID)
		binding := i18n.GlossaryCorpusPackageAuthority{Package: a.Package.ID, Authority: i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256}, SourcePaths: []string{}}
		for _, s := range a.Sources {
			binding.SourcePaths = append(binding.SourcePaths, s.Path)
		}
		sort.Strings(binding.SourcePaths)
		packageAuthorities = append(packageAuthorities, binding)
		sources[ref.Path] = true
		sources[pagePackageCurrent(a.Package.ID)] = true
		sources[a.Proposal.Path] = true
		sources[a.FrozenManifest.Path] = true
		for _, s := range a.Sources {
			sources[s.Path] = true
		}
	}
	for _, ref := range g.StructuredSuccessors {
		a, err := CheckStructuredSourceSuccessor(root, ref)
		if err != nil {
			return nil, err
		}
		for _, name := range []string{ref.Path, structuredSourceCurrent(a.Previous.Path), a.Proposal.Path, a.Body.Path} {
			sources[name] = true
		}
	}
	for _, pkg := range packages {
		docs, raw, err := CurrentPackageDocuments(root, g, pkg)
		if err != nil {
			return nil, err
		}
		for _, d := range docs {
			var c i18n.LanguageContributor
			var err error
			if d.Contract == ShellExtractionContract {
				s, e := ExtractShellLanguageSource(d.Path, raw[d.Path])
				err = e
				if e == nil {
					c, err = ProjectShellSuccessorContributor(*s)
				}
			} else {
				c, err = projectedDocumentContributor(d, pkg)
			}
			if err != nil {
				return nil, err
			}
			cs = append(cs, c)
			if pkg == "site-v2-shell" {
				sources[d.Path] = true
			}
		}
	}
	schema := i18n.GlossaryCorpusSchema
	if g.ShellAdoption != nil {
		schema = i18n.GlossaryCorpusSchemaV2
		for i := range cs {
			cs[i].Purpose = i18n.ContributorPurposeActiveContent
			cs[i].SourceSHA = i18n.ContributorIdentity(cs[i])
		}
	}
	if terminology != nil {
		for i := range cs {
			cs[i].Purpose = i18n.ContributorPurposeActiveContent
			cs[i].SourceSHA = i18n.ContributorIdentity(cs[i])
		}
		discovery, err := ProjectTerminologyContributors(*terminology, snapshot)
		if err != nil {
			return nil, err
		}
		cs = append(cs, discovery...)
		sources[TerminologyManifestPath] = true
		sources[TerminologySnapshotPath] = true
		manifest, err := Encode(terminology)
		if err != nil {
			return nil, err
		}
		candidateSources[TerminologyManifestPath] = manifest
		candidateSources[TerminologySnapshotPath] = snapshot
		schema = i18n.GlossaryCorpusSchemaV2
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	c := &i18n.GlossarySourceCorpus{Schema: schema, Contributors: cs, Sources: []i18n.GlossaryArchiveReference{}}
	if len(packageAuthorities) > 0 {
		c.PackageAuthorities = packageAuthorities
	}
	paths := []string{}
	for p := range sources {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if p == SnapshotPath {
			// Existing corpus refs retain their exact historical path/SHA. The
			// resolution layer verifies proof even when the ZIP is still present.
			if _, err := readRegular(root, i18n.GlossaryCorpusPath); err == nil {
				ref, err := i18n.CorpusStorageReference(root)
				if err != nil {
					return nil, err
				}
				if ref.SHA256 != g.SnapshotSHA256 {
					return nil, fmt.Errorf("corpus/source registered ZIP identity differs")
				}
				c.Sources = append(c.Sources, ref)
				continue
			} else if !os.IsNotExist(err) {
				return nil, err
			}
		}
		b, ok := candidateSources[p]
		if !ok {
			var err error
			b, err = readRegular(root, p)
			if err != nil {
				return nil, err
			}
		}
		c.Sources = append(c.Sources, i18n.GlossaryArchiveReference{Path: p, SHA256: digest(b)})
	}
	c.Identity = i18n.CorpusIdentity(*c)
	if err := i18n.ValidateGlossaryCorpus(c); err != nil {
		return nil, err
	}
	return c, nil
}
func CheckUnifiedCorpus(root string, contexts ...*structuredSuccessorValidation) error {
	want, err := BuildUnifiedCorpus(root, contexts...)
	if err != nil {
		return err
	}
	current, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return err
	}
	if identity(want) != identity(current) {
		return fmt.Errorf("unified glossary corpus projection stale")
	}
	return nil
}
func requireUnifiedReview(root, locale string) error {
	if err := CheckUnifiedCorpus(root); err != nil {
		return err
	}
	_, err := i18n.RequireUnifiedGlossaryReview(root, locale)
	return err
}

// Projection excludes machine-only gaps; compact locked tokens remain terminology context.
func projectedDocumentContributor(d Document, pkg string) (i18n.LanguageContributor, error) {
	c := i18n.LanguageContributor{ID: "source:" + d.Path, Parser: UnitContract, Package: pkg, Path: d.Path, Route: d.Route, Structured: pkg == "site-v2-shell" || d.Kind == "data", Contexts: []i18n.LanguageContext{}}
	for _, u := range d.Units {
		text := u.Source
		locked := []string{}
		for _, p := range u.Protected {
			text = strings.ReplaceAll(text, p.Token, "")
			if !strings.Contains(p.Raw, "\n") && len(p.Raw) < 256 {
				locked = append(locked, p.Raw)
			}
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		// Context identity is projected text/protection, independent of raw code.
		context := strings.Split(u.Locator, ":")[0]
		if d.Kind == "data" {
			// YAML item URLs locate restoration slots, but are not terminology.
			// Corpus coordinates bind the approved visible field and its language.
			context = "data/" + context[strings.LastIndex(context, "/")+1:]
		}
		id := identity(struct{ Locator, Text string }{context, text})
		c.Contexts = append(c.Contexts, i18n.LanguageContext{ID: id, Text: text, Protected: locked})
	}
	sort.Slice(c.Contexts, func(i, j int) bool { return c.Contexts[i].ID < c.Contexts[j].ID })
	// Repeated teaching prose has occurrence identity within the contributor.
	for i := range c.Contexts {
		if i > 0 && c.Contexts[i].ID <= c.Contexts[i-1].ID {
			c.Contexts[i].ID = c.Contexts[i-1].ID + "x"
		}
	}
	if len(c.Contexts) == 0 {
		return i18n.LanguageContributor{}, fmt.Errorf("empty language contributor %s", d.Path)
	}
	c.SourceSHA = i18n.ContributorIdentity(c)
	return c, nil
}
