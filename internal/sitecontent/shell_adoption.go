package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const ShellAdoptionSchema = "go-learning/shell-current-adoption/v1"
const ShellAdoptionCurrent = ShellSuccessorRoot + "current.json"

// A source adoption binds the existing frozen foundation to current publication
// inventory. It is global source governance, never locale language acceptance.
type ShellAdoption struct {
	Schema      string    `json:"schema"`
	Foundation  Reference `json:"frozen_successor_foundation"`
	Predecessor Reference `json:"historical_global_authority"`
	Sources     []Source  `json:"current_source_inventory"`
	Package     Package   `json:"current_package"`
	Identity    string    `json:"identity_sha256"`
}

func BuildShellAdoption(root string, foundation Reference) (*ShellAdoption, error) {
	if _, err := i18n.LoadUnifiedGlossaryCorpus(root); err != nil {
		return nil, err
	}
	return buildShellAdoption(root, foundation, nil)
}

func buildShellAdoption(root string, foundation Reference, validatorBaseline []Dependency, contexts ...*structuredSuccessorValidation) (*ShellAdoption, error) {
	base, err := loadHistoricalGlobal(root, contexts...)
	if err != nil {
		return nil, err
	}
	old, err := packageByID(base, "site-v2-shell")
	if err != nil {
		return nil, err
	}
	var a ShellSuccessorAuthority
	if err := readReference(root, foundation, &a); err != nil {
		return nil, err
	}
	copy := a
	copy.Identity = ""
	if a.Schema != ShellSuccessorSchema || a.State != "pending-source-authority" || a.Identity != identity(copy) || foundation.Path != ShellSuccessorRoot+"sources/"+a.Identity+".json" || a.PredecessorPackageIdentity != old.Identity || a.ExtractionContract != ShellExtractionContract {
		return nil, fmt.Errorf("invalid frozen shell foundation/baseline")
	}
	if err := checkShellParserDependencies(root, a); err != nil {
		return nil, err
	}
	m, err := loadSourceManifest(root)
	if err != nil || m.Identity != a.FrozenManifestIdentity {
		return nil, fmt.Errorf("shell adoption manifest mismatch")
	}
	// This immutable source proof checks the exact extracted contributors,
	// not current language eligibility after a consumed source transition.
	corpus, err := readProjectionCorpus(root, i18n.GlossaryCorpusPath)
	if err != nil {
		return nil, err
	}
	registered := map[string]i18n.LanguageContributor{}
	for _, c := range corpus.Contributors {
		registered[c.ID] = c
	}
	paths := []string{"_content/index.md", "_content/site.tmpl", "_content/menus.yaml", "_content/resources.yaml", "_content/testimonials.yaml"}
	if len(a.Sources) != len(paths) {
		return nil, fmt.Errorf("shell source inventory non-exact")
	}
	adoption := &ShellAdoption{Schema: ShellAdoptionSchema, Foundation: foundation, Sources: []Source{}, Package: old}
	adoption.Predecessor, err = ReferenceFile(root, GlobalPath)
	if err != nil {
		return nil, err
	}
	for i, s := range a.Sources {
		if s.Path != paths[i] {
			return nil, fmt.Errorf("shell source membership/order mismatch")
		}
		var file *FrozenSourceFile
		for j := range m.Files {
			if m.Files[j].Path == s.Path {
				file = &m.Files[j]
			}
		}
		if file == nil || file.Classification != "upstream-exact" || file.SHA256 != s.SourceSHA {
			return nil, fmt.Errorf("shell frozen source authority mismatch")
		}
		raw, err := readAdoptedSource(root, *file)
		if err != nil {
			return nil, err
		}
		parsed, err := ExtractShellLanguageSource(s.Path, raw)
		if err != nil || !reflect.DeepEqual(s, *parsed) {
			return nil, fmt.Errorf("shell frozen representation/parser mismatch: %s", s.Path)
		}
		contributor, err := ProjectShellSuccessorContributor(s)
		if err != nil || !reflect.DeepEqual(registered[contributor.ID], contributor) {
			return nil, fmt.Errorf("shell source lacks current unified corpus authority")
		}
		kind, route := "data", ""
		if s.Page != nil {
			kind, route = "page", "/"
		}
		adoption.Sources = append(adoption.Sources, Source{Path: s.Path, Kind: kind, Origin: "frozen-upstream", Package: old.ID, Surface: "homepage", Route: route, SHA256: s.SourceSHA, Dependencies: []Dependency{}})
	}
	translation, err := readRegular(root, a.TranslationBody.Path)
	if err != nil || digest(translation) != a.TranslationBody.SourceSHA || a.TranslationBody.Path != "data/site-shell/translation.html" {
		return nil, fmt.Errorf("translation source drift")
	}
	adoption.Sources = append(adoption.Sources, Source{Path: a.TranslationBody.Path, Kind: "page", Origin: "repository", Package: old.ID, Surface: "translation", Route: "/translation/", SHA256: digest(translation), Dependencies: []Dependency{}})
	adoption.Package.ParserContract = ShellExtractionContract
	adoption.Package.Dependencies = append([]Dependency{}, old.Dependencies...)
	for _, name := range []string{"internal/sitecontent/shell_reconstruction.go", "internal/sitecontent/shell_generation_context.go", "internal/sitecontent/shell_runtime_context.go", "internal/site/shell_render.go", "internal/site/tmplfunc/tmpl.go", "internal/site/tmplfunc/func.go"} {
		ref, err := ReferenceFile(root, name)
		if err != nil {
			return nil, err
		}
		adoption.Package.Dependencies = append(adoption.Package.Dependencies, Dependency{Kind: "contract", Path: ref.Path, SHA256: ref.SHA256})
	}
	// Existing source adoption is not a validator-version receipt. Every
	// frozen source/slot/protection field above must still match the current
	// extractor. Keep its original validator dependency identity while current
	// reconstruction and canonical candidate validation run today's code.
	adoption.Package.Dependencies = preserveFrozenValidatorDependencies(adoption.Package.Dependencies, validatorBaseline, []string{"internal/sitecontent/shell_reconstruction.go"})
	adoption.Package.SourceIdentity = identity(struct {
		Sources   []Source
		Contracts []Dependency
	}{adoption.Sources, adoption.Package.Dependencies})
	adoption.Package.Identity = ""
	adoption.Package.Identity = identity(adoption.Package)
	adoption.Identity = identity(*adoption)
	return adoption, nil
}

func shellAdoptionPath(a *ShellAdoption) string {
	return ShellSuccessorRoot + "adoptions/" + a.Identity + ".json"
}
func CheckShellAdoption(root string, ref Reference, contexts ...*structuredSuccessorValidation) (*ShellAdoption, error) {
	if len(contexts) != 0 && contexts[0] != nil && contexts[0].registrationReuse && contexts[0].readonlyOwner {
		var a ShellAdoption
		err := contexts[0].registrationAdoption(root, ref, "shell", &a, func() (any, error) {
			return checkShellAdoption(root, ref, contexts...)
		})
		return &a, err
	}
	return checkShellAdoption(root, ref, contexts...)
}

func checkShellAdoption(root string, ref Reference, contexts ...*structuredSuccessorValidation) (*ShellAdoption, error) {
	var a ShellAdoption
	if err := readReference(root, ref, &a); err != nil {
		return nil, err
	}
	want, err := buildShellAdoption(root, a.Foundation, a.Package.Dependencies, contexts...)
	if err != nil {
		return nil, err
	}
	if ref.Path != shellAdoptionPath(want) || !reflect.DeepEqual(a, *want) {
		return nil, fmt.Errorf("shell adoption stale/noncanonical")
	}
	return &a, nil
}

// Validator implementations can evolve without replacing frozen semantic
// source authority. The owner declares only validation dependencies; all other
// dependency hashes remain current and strict. No history or receipt is edited.
func preserveFrozenValidatorDependencies(current, frozen []Dependency, paths []string) []Dependency {
	result := append([]Dependency{}, current...)
	for i, d := range result {
		for _, p := range paths {
			if d.Path != p {
				continue
			}
			for _, old := range frozen {
				if old.Path == d.Path && old.Kind == d.Kind {
					result[i] = old
					break
				}
			}
		}
	}
	return result
}

// First adoption uses exact predecessor CAS and atomic no-overwrite pointer.
// Replays are idempotent only for precisely the same already-current artifact.
func AdoptShellSuccessor(root string, foundation Reference, prior string) (*ShellAdoption, error) {
	a, err := BuildShellAdoption(root, foundation)
	if err != nil {
		return nil, err
	}
	if !validSHA(prior) || prior != a.Predecessor.SHA256 {
		return nil, fmt.Errorf("shell adoption predecessor CAS mismatch")
	}
	if err := saveImmutable(root, shellAdoptionPath(a), a); err != nil {
		return nil, err
	}
	ref, err := ReferenceFile(root, shellAdoptionPath(a))
	if err != nil {
		return nil, err
	}
	b, err := Encode(ref)
	if err != nil {
		return nil, err
	}
	current, err := readRegular(root, ShellAdoptionCurrent)
	if err == nil {
		if !bytes.Equal(current, b) {
			return nil, fmt.Errorf("another shell adoption is current")
		}
		return a, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	actual, err := ReferenceFile(root, GlobalPath)
	if err != nil || actual != a.Predecessor {
		return nil, fmt.Errorf("concurrent source authority mutation")
	}
	return a, SaveArtifact(root, ShellAdoptionCurrent, b)
}

func applyCurrentShellAuthority(root string, base *Global, contexts ...*structuredSuccessorValidation) (*Global, error) {
	b, err := readRegular(root, ShellAdoptionCurrent)
	if os.IsNotExist(err) {
		return base, nil
	}
	if err != nil {
		return nil, err
	}
	var ref Reference
	if err := StrictJSON(b, &ref); err != nil {
		return nil, err
	}
	a, err := CheckShellAdoption(root, ref, contexts...)
	if err != nil {
		return nil, err
	}
	next := *base
	next.Sources = []Source{}
	next.Packages = append([]Package(nil), base.Packages...)
	next.Surfaces = append([]Surface(nil), base.Surfaces...)
	for _, s := range base.Sources {
		if s.Package != a.Package.ID {
			next.Sources = append(next.Sources, s)
		}
	}
	next.Sources = append(next.Sources, a.Sources...)
	for i, p := range next.Packages {
		if p.ID == a.Package.ID {
			next.HistoricalPackages = []Package{p}
			next.Packages[i] = a.Package
		}
	}
	for i := range next.Surfaces {
		if next.Surfaces[i].ID == "homepage" {
			next.Surfaces[i].Parser = PageContract
		}
	}
	next.ShellAdoption = &ref
	next.Identity = ""
	next.Identity = identity(next)
	return &next, ValidateGlobal(next)
}

func CurrentPackageDocuments(root string, g *Global, pkg string, contexts ...*structuredSuccessorValidation) ([]Document, map[string][]byte, error) {
	docs, raw, err := baseCurrentPackageDocuments(root, g, pkg, contexts...)
	if err != nil {
		return nil, nil, err
	}
	return selectSharedStructuredDocuments(root, g, pkg, docs, raw, contexts...)
}

func baseCurrentPackageDocuments(root string, g *Global, pkg string, contexts ...*structuredSuccessorValidation) ([]Document, map[string][]byte, error) {
	if pkg != "site-v2-shell" || g.ShellAdoption == nil {
		docs, raw, err := sharedPackageRepresentation(root, g, pkg, contexts...)
		if err != nil {
			return nil, nil, err
		}
		return applyStructuredDocumentSuccessors(root, g, pkg, docs, raw, contexts...)
	}
	a, err := CheckShellAdoption(root, *g.ShellAdoption, contexts...)
	if err != nil {
		return nil, nil, err
	}
	var foundation ShellSuccessorAuthority
	if err := readReference(root, a.Foundation, &foundation); err != nil {
		return nil, nil, err
	}
	docs := []Document{}
	raw := map[string][]byte{}
	for _, s := range foundation.Sources {
		b, err := readRegular(root, s.Path)
		if err != nil || digest(b) != s.SourceSHA {
			return nil, nil, fmt.Errorf("current shell source drift")
		}
		d := shellDocument(s)
		if s.Page != nil && g.ShellRepresentation != nil {
			selected, _, err := selectShellRepresentation(root, b, contexts...)
			if err != nil {
				return nil, nil, err
			}
			d = *selected.Page
		}
		docs = append(docs, d)
		raw[s.Path] = b
	}
	b, err := readRegular(root, foundation.TranslationBody.Path)
	if err != nil || digest(b) != foundation.TranslationBody.SourceSHA {
		return nil, nil, fmt.Errorf("translation source drift")
	}
	docs = append(docs, foundation.TranslationBody)
	raw[foundation.TranslationBody.Path] = b
	return applyStructuredDocumentSuccessors(root, g, pkg, docs, raw, contexts...)
}

func AttachShellPageTranslationUnits(root string, c *i18n.Catalog, contexts ...*structuredSuccessorValidation) ([]string, error) {
	g, err := LoadCurrent(root, contexts...)
	if err != nil {
		return nil, err
	}
	if g.ShellAdoption == nil {
		return nil, nil
	}
	// A structured representation head owns no Page TU. Page registration,
	// including historical Snapshot integrity, must not depend on its current
	// parser/dependency epoch; structured readers validate those heads separately.
	docs, raw, err := baseCurrentPackageDocuments(root, g, "site-v2-shell", contexts...)
	if err != nil {
		return nil, err
	}
	p, err := packageByID(g, "site-v2-shell")
	if err != nil {
		return nil, err
	}
	descriptor := i18n.TranslationPackageDescriptor{ID: p.ID, AuthoritySHA256: p.Identity}
	ids := []string{}
	for _, d := range docs {
		if d.UnitScope != PageContract {
			continue
		}
		s, representation, err := selectShellRepresentation(root, raw[d.Path], contexts...)
		if err != nil {
			return nil, err
		}
		if (g.ShellRepresentation == nil) != (representation == nil) {
			return nil, fmt.Errorf("canonical representation changed during catalog construction")
		}
		if representation != nil {
			ref := shellRepresentationReceipt(representation)
			if ref != *g.ShellRepresentation || representationPackage(representation.Basis.Package, ref).Identity != p.Identity {
				return nil, fmt.Errorf("canonical representation/package authority drift")
			}
		}
		u := s.Page.Units[0]
		if representation != nil {
			priorPackage := representation.Basis.Package
			if representation.Predecessor != nil {
				priorPackage = representationPackage(priorPackage, *representation.Predecessor)
			}
			old := representation.Old.Source
			oldUnit := old.Page.Units[0]
			c.HistoricalTranslationPackages = append(c.HistoricalTranslationPackages, i18n.TranslationPackageDescriptor{ID: p.ID, AuthoritySHA256: priorPackage.Identity, Units: []i18n.AdaptedTranslationUnit{{Unit: i18n.TranslationUnit{ID: oldUnit.ID, Kind: i18n.UnitKindPage, SourcePath: d.Path, Source: bytes.Clone(raw[d.Path]), SourceSHA256: d.SourceSHA}, Adapter: ShellPageAdapter{old, bytes.Clone(raw[d.Path])}, StableIndex: 1, Route: "/", WorkingSet: "homepage-initial", WorkingSetLimit: 1}}})
		}
		ids = append(ids, u.ID)
		descriptor.Units = append(descriptor.Units, i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: u.ID, Kind: i18n.UnitKindPage, SourcePath: d.Path, Source: raw[d.Path], SourceSHA256: d.SourceSHA}, Adapter: ShellPageAdapter{*s, raw[d.Path]}, StableIndex: 1, Route: "/", WorkingSet: "homepage-initial", WorkingSetLimit: 1})
	}
	if len(ids) != 1 {
		return nil, fmt.Errorf("current Homepage requires exactly one physical Page TU")
	}
	return ids, c.RegisterTranslationPackage(descriptor)
}

type HomepagePlan struct {
	Locale    string         `json:"locale"`
	Package   string         `json:"package"`
	PageCount int            `json:"formal_page_count"`
	Units     []LanguageWork `json:"units"`
	Identity  string         `json:"identity_sha256"`
}

func PlanHomepage(root, locale string) (*HomepagePlan, error) {
	if err := requireUnifiedReview(root, locale); err != nil {
		return nil, err
	}
	c, err := checkedCatalog(root)
	if err != nil {
		return nil, err
	}
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		return nil, err
	}
	if err := c.UseTranslationPackage("site-v2-shell"); err != nil {
		return nil, err
	}
	p, err := i18n.BuildTranslationUnitProgress(root, c, locale)
	if err != nil {
		return nil, err
	}
	plan := &HomepagePlan{Locale: locale, Package: "site-v2-shell", PageCount: len(p.Units), Units: []LanguageWork{}}
	authorization, err := i18n.FindCurrentScopeTranslationAuthorization(root, c, locale)
	if err != nil {
		return nil, err
	}
	accepted := map[string]bool{}
	if authorization != nil && authorization.Ready() {
		for _, u := range authorization.Units {
			accepted[u.Snapshot.UnitID] = u.Classification == "compatible"
		}
	}
	for _, u := range p.Units {
		state := u.Status
		if state == "missing" {
			state = "generation-required"
		}
		if state == "passed" {
			state = "review-required"
			if accepted[u.UnitID] {
				state = "carry"
			}
		}
		unit, err := c.Unit(u.UnitID)
		if err != nil {
			return nil, err
		}
		plan.Units = append(plan.Units, LanguageWork{ID: u.UnitID, Package: "site-v2-shell", SourcePath: unit.SourcePath, Route: "/", State: state, Reason: "shared TranslationUnit progress", SourceSHA: unit.SourceSHA256})
	}
	sort.Slice(plan.Units, func(i, j int) bool { return plan.Units[i].ID < plan.Units[j].ID })
	plan.Identity = identity(*plan)
	return plan, nil
}
