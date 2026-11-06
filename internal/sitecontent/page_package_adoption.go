package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const PagePackageAdoptionSchema = "go-learning/page-package-source-adoption/v1"
const PagePackageAdoptionRoot = "data/content-package-adoptions/"

// Source governance only. The descriptor below enters the existing shared TU
// lifecycle; this receipt grants no language, QC, completion or activation.
type PagePackageProposal struct {
	Schema  string      `json:"schema"`
	Package string      `json:"package"`
	Surface string      `json:"surface"`
	Sources []Reference `json:"ordered_sources"`
}

type PagePackageAdoption struct {
	Aliases        []AdoptedRouteAlias `json:"aliases,omitempty"`
	Schema         string              `json:"schema"`
	Proposal       Reference           `json:"proposal"`
	BaseIdentity   string              `json:"predecessor_scope_identity_sha256"`
	FrozenManifest Reference           `json:"frozen_source_manifest"`
	Package        Package             `json:"package"`
	Surface        Surface             `json:"surface"`
	Sources        []Source            `json:"sources"`
	Documents      []Document          `json:"documents"`
	Identity       string              `json:"identity_sha256"`
}

type AdoptedRouteAlias struct {
	Alias  string `json:"alias"`
	Target string `json:"target"`
}

func adoptionBase(root string, contexts ...*structuredSuccessorValidation) (*Global, error) {
	g, err := loadHistoricalGlobal(root, contexts...)
	if err != nil {
		return nil, err
	}
	return applyCurrentShellAuthority(root, g, contexts...)
}

func BuildPagePackageAdoption(root string, proposal Reference) (*PagePackageAdoption, error) {
	g, err := adoptionBase(root)
	if err != nil {
		return nil, err
	}
	return buildPagePackageAdoption(root, g, proposal)
}

func buildPagePackageAdoption(root string, base *Global, ref Reference, frozen ...[]Document) (*PagePackageAdoption, error) {
	var p PagePackageProposal
	if err := readReference(root, ref, &p); err != nil {
		return nil, err
	}
	if p.Schema != PagePackageAdoptionSchema || !component(p.Package) || !component(p.Surface) || len(p.Sources) == 0 {
		return nil, fmt.Errorf("complete explicit Page package source proposal required")
	}
	if len(frozen) > 0 {
		if len(frozen) != 1 || len(frozen[0]) != len(p.Sources) {
			return nil, fmt.Errorf("frozen adoption exact document set mismatch")
		}
		if err := validateDocumentRepresentations(frozen[0], false); err != nil {
			return nil, err
		}
	}
	for _, existing := range base.Packages {
		if existing.ID == p.Package {
			return nil, fmt.Errorf("adoption cannot replace an existing package")
		}
	}
	for _, existing := range base.Surfaces {
		if existing.ID == p.Surface {
			return nil, fmt.Errorf("adoption cannot replace an existing surface")
		}
	}
	m, err := loadSourceManifest(root)
	if err != nil {
		return nil, err
	}
	mref, err := ReferenceFile(root, SourceManifestPath)
	if err != nil {
		return nil, err
	}
	a := &PagePackageAdoption{Schema: PagePackageAdoptionSchema, Proposal: ref, BaseIdentity: base.Identity, FrozenManifest: mref,
		Package: Package{ID: p.Package, Activation: "atomic-locale-package", ParserContract: UnitContract, Routes: []string{}, Surfaces: []string{p.Surface}, Dependencies: []Dependency{}},
		Surface: Surface{ID: p.Surface, Package: p.Package, RouteFamily: "", Parser: PageContract, Publication: "requires-independent-completion"}, Sources: []Source{}, Documents: []Document{}}
	seen, routes := map[string]bool{}, map[string]bool{}
	for _, s := range base.Sources {
		seen[s.Path] = true
	}
	for _, p := range base.Packages {
		for _, route := range p.Routes {
			routes[route] = true
		}
	}
	for index, ref := range p.Sources {
		var f *FrozenSourceFile
		for i := range m.Files {
			if m.Files[i].Path == ref.Path {
				f = &m.Files[i]
			}
		}
		if f == nil || f.Classification != "upstream-exact" || f.SHA256 != ref.SHA256 || seen[ref.Path] || (path.Ext(ref.Path) != ".html" && path.Ext(ref.Path) != ".md") {
			return nil, fmt.Errorf("source must be a distinct unregistered frozen upstream Page: %s", ref.Path)
		}
		raw, err := readAdoptedSource(root, *f)
		if err != nil {
			return nil, err
		}
		s := Source{Path: ref.Path, Kind: "page", Origin: "frozen-upstream", Package: p.Package, Surface: p.Surface, Route: pageRoute(ref.Path), SHA256: ref.SHA256, Dependencies: []Dependency{}}
		if !validRoute(s.Route) || routes[s.Route] {
			return nil, fmt.Errorf("canonical route conflicts: %s", s.Route)
		}
		var d *Document
		if len(frozen) == 0 {
			d, err = ParseWorkflowDocument(s, raw, index+1)
		} else {
			// Adoption is a frozen source foundation, not a permanent choice
			// of today's protector. Verify its original source coordinates;
			// current representations are parsed independently by the loader.
			d = &frozen[0][index]
			if d.Path != s.Path || d.Kind != s.Kind || d.Route != s.Route || d.SourceSHA != s.SHA256 || d.StableIndex != index+1 || !reflect.DeepEqual(d.Dependencies, s.Dependencies) {
				return nil, fmt.Errorf("frozen adoption source/membership mismatch")
			}
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ref.Path, err)
		}
		candidate, err := (PageAdapter{*d, raw}).Restore(d.Units[0].Source)
		if err != nil || !bytes.Equal(raw, candidate) {
			return nil, fmt.Errorf("source Page round trip failed: %s", ref.Path)
		}
		seen[ref.Path], routes[s.Route] = true, true
		if strings.HasSuffix(s.Route, "/") && s.Route != "/" {
			alias := strings.TrimSuffix(s.Route, "/")
			if routes[alias] {
				return nil, fmt.Errorf("index route alias conflicts with canonical route")
			}
			a.Aliases = append(a.Aliases, AdoptedRouteAlias{Alias: alias, Target: s.Route})
		}
		a.Sources = append(a.Sources, s)
		a.Documents = append(a.Documents, *d)
		a.Package.Routes = append(a.Package.Routes, s.Route)
	}
	// The standard frozen Page templates remain source dependencies, rather
	// than being translated or inferred from rendered output.
	for _, name := range []string{"_content/site.tmpl", "_content/article.tmpl", "_content/doc/default.tmpl"} {
		var f *FrozenSourceFile
		for i := range m.Files {
			if m.Files[i].Path == name {
				f = &m.Files[i]
			}
		}
		if f == nil {
			return nil, fmt.Errorf("missing frozen template dependency: %s", name)
		}
		if _, err := readAdoptedSource(root, *f); err != nil {
			return nil, err
		}
		a.Package.Dependencies = append(a.Package.Dependencies, Dependency{Kind: "contract", Path: name, SHA256: f.SHA256})
	}
	sort.Strings(a.Package.Routes)
	a.Package.SourceIdentity = identity(struct {
		Sources      []Source
		Dependencies []Dependency
	}{a.Sources, a.Package.Dependencies})
	a.Package.Identity = identity(a.Package)
	a.Identity = identity(*a)
	return a, nil
}

func PagePackageAdoptionPath(a *PagePackageAdoption) string {
	return PagePackageAdoptionRoot + "receipts/" + a.Identity + ".json"
}
func pagePackageCurrent(id string) string { return PagePackageAdoptionRoot + "current/" + id + ".json" }

func CheckPagePackageAdoption(root string, ref Reference, contexts ...*structuredSuccessorValidation) (*PagePackageAdoption, error) {
	if len(contexts) != 0 && contexts[0] != nil && contexts[0].registrationReuse && contexts[0].readonlyOwner {
		var a PagePackageAdoption
		err := contexts[0].registrationAdoption(root, ref, "page", &a, func() (any, error) {
			return checkPagePackageAdoption(root, ref, contexts...)
		})
		return &a, err
	}
	return checkPagePackageAdoption(root, ref, contexts...)
}

func checkPagePackageAdoption(root string, ref Reference, contexts ...*structuredSuccessorValidation) (*PagePackageAdoption, error) {
	var a PagePackageAdoption
	if err := readReference(root, ref, &a); err != nil {
		return nil, err
	}
	base, err := adoptionBase(root, contexts...)
	if err != nil {
		return nil, err
	}
	want, err := buildPagePackageAdoption(root, base, a.Proposal, a.Documents)
	if err != nil {
		return nil, err
	}
	if ref.Path != PagePackageAdoptionPath(want) || !reflect.DeepEqual(a, *want) {
		return nil, fmt.Errorf("Page package adoption stale/noncanonical")
	}
	return &a, nil
}

// The pointer is the sole publication boundary. Orphan content-addressed
// receipts are inert and safe to reuse; exact identical replay is idempotent.
func AdoptPagePackage(root string, proposal Reference, prior string) (*PagePackageAdoption, error) {
	a, err := BuildPagePackageAdoption(root, proposal)
	if err != nil {
		return nil, err
	}
	if !validSHA(prior) || prior != a.BaseIdentity {
		return nil, fmt.Errorf("exact predecessor scope identity CAS required")
	}
	lock, err := os.OpenFile(pathJoin(root, "data/.content-package-adoption.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	if err = lock.Close(); err != nil {
		return nil, err
	}
	defer os.Remove(lock.Name())
	if err = saveImmutable(root, PagePackageAdoptionPath(a), a); err != nil {
		return nil, err
	}
	ref, err := ReferenceFile(root, PagePackageAdoptionPath(a))
	if err != nil {
		return nil, err
	}
	pointer, _ := Encode(ref)
	current, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	if existing, e := readRegular(root, pagePackageCurrent(a.Package.ID)); e == nil {
		if !bytes.Equal(existing, pointer) {
			return nil, fmt.Errorf("conflicting adoption pointer")
		}
		return a, nil
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	if _, err := projectPagePackageAdoption(current, *a, ref); err != nil {
		return nil, err
	}
	fresh, err := BuildPagePackageAdoption(root, proposal)
	if err != nil || !reflect.DeepEqual(a, fresh) {
		return nil, fmt.Errorf("concurrent source adoption dependency mutation")
	}
	return a, SaveArtifact(root, pagePackageCurrent(a.Package.ID), pointer)
}

func projectPagePackageAdoption(base *Global, a PagePackageAdoption, ref Reference) (*Global, error) {
	for _, alias := range a.Aliases {
		for _, p := range base.Packages {
			for _, route := range p.Routes {
				if alias.Alias == route {
					return nil, fmt.Errorf("adopted alias shadows current canonical route")
				}
			}
		}
	}
	next := *base
	next.PackageAdoptions = append(append([]Reference{}, base.PackageAdoptions...), ref)
	next.Packages = append(append([]Package{}, base.Packages...), a.Package)
	next.Surfaces = append(append([]Surface{}, base.Surfaces...), a.Surface)
	next.Sources = append(append([]Source{}, base.Sources...), a.Sources...)
	next.Versions = append([]Version{}, base.Versions...)
	next.Versions[1].Packages = append(append([]string{}, base.Versions[1].Packages...), a.Package.ID)
	next.Identity = ""
	next.Identity = identity(next)
	return &next, ValidateGlobal(next)
}

func applyPagePackageAdoptions(root string, base *Global, contexts ...*structuredSuccessorValidation) (*Global, error) {
	entries, err := os.ReadDir(pathJoin(root, PagePackageAdoptionRoot+"current"))
	if os.IsNotExist(err) {
		return base, nil
	}
	if err != nil {
		return nil, err
	}
	next := base
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return nil, fmt.Errorf("unsafe package adoption pointer")
		}
		var ref Reference
		b, err := readRegular(root, PagePackageAdoptionRoot+"current/"+entry.Name())
		if err != nil {
			return nil, err
		}
		if err = StrictJSON(b, &ref); err != nil {
			return nil, err
		}
		a, err := CheckPagePackageAdoption(root, ref, contexts...)
		if err != nil {
			return nil, err
		}
		if entry.Name() != a.Package.ID+".json" {
			return nil, fmt.Errorf("package adoption pointer membership mismatch")
		}
		next, err = projectPagePackageAdoption(next, *a, ref)
		if err != nil {
			return nil, err
		}
	}
	return next, nil
}

func adoptedPageDocuments(root string, g *Global, pkg string, contexts ...*structuredSuccessorValidation) ([]Document, map[string][]byte, error) {
	for _, ref := range g.PackageAdoptions {
		a, err := CheckPagePackageAdoption(root, ref, contexts...)
		if err != nil {
			return nil, nil, err
		}
		if a.Package.ID != pkg {
			continue
		}
		raw := map[string][]byte{}
		for _, s := range a.Sources {
			b, err := readRegular(root, s.Path)
			if err != nil || digest(b) != s.SHA256 {
				return nil, nil, fmt.Errorf("adopted Page source drift: %s", s.Path)
			}
			raw[s.Path] = b
		}
		docs := make([]Document, 0, len(a.Sources))
		for index, s := range a.Sources {
			d, err := ParseWorkflowDocument(s, raw[s.Path], index+1)
			if err != nil {
				return nil, nil, err
			}
			docs = append(docs, *d)
		}
		return docs, raw, nil
	}
	return nil, nil, fmt.Errorf("unregistered adopted Page package: %s", pkg)
}

// Registry data only: adapters and every lifecycle transition are shared.
func AttachAdoptedPageTranslationUnits(root string, c *i18n.Catalog, contexts ...*structuredSuccessorValidation) ([]string, error) {
	g, err := loadSourceCurrent(root, contexts...)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, ref := range g.PackageAdoptions {
		a, err := CheckPagePackageAdoption(root, ref, contexts...)
		if err != nil {
			return nil, err
		}
		docs, raw, err := adoptedPageDocuments(root, g, a.Package.ID, contexts...)
		if err != nil {
			return nil, err
		}
		descriptor := i18n.TranslationPackageDescriptor{ID: a.Package.ID, AuthoritySHA256: a.Package.Identity}
		for _, d := range docs {
			u := d.Units[0]
			adapter := adoptedPageAdapter{PageAdapter: PageAdapter{d, bytes.Clone(raw[d.Path])}, root: root}
			descriptor.Units = append(descriptor.Units, i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: u.ID, Kind: i18n.UnitKindPage, SourcePath: d.Path, Source: raw[d.Path], SourceSHA256: d.SourceSHA}, Adapter: adapter, StableIndex: d.StableIndex, Route: d.Route, WorkingSet: fmt.Sprintf("page-batch-%d", (d.StableIndex-1)/30+1), WorkingSetLimit: 30})
			ids = append(ids, u.ID)
		}
		if err = c.RegisterTranslationPackage(descriptor); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

type adoptedPageAdapter struct {
	PageAdapter
	root string
}

func (a adoptedPageAdapter) ReattestationAuthority(root string) (string, error) {
	base, err := a.PageAdapter.ReattestationAuthority(root)
	if err != nil {
		return "", err
	}
	registration, err := i18n.TranslationAdapterAuthority(root, "internal/sitecontent/page_package_adoption.go", "adoptedPageAdapter", []string{"internal/sitecontent/corpus.go", "internal/sitecontent/adopted_corpus.go"})
	if err != nil {
		return "", err
	}
	return identity([]string{base, registration}), nil
}

func (a adoptedPageAdapter) ProtectedInput(glossary *i18n.Glossary) (string, []string, error) {
	// Source registration must not let a pre-adoption corpus/Review silently
	// authorize new content. Read current projection before exporting inputs.
	if err := CheckUnifiedCorpus(a.root); err != nil {
		return "", nil, err
	}
	return a.PageAdapter.ProtectedInput(glossary)
}
