package sitecontent

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const IntegratedSurfaceReviewerBundleSchema = "go-learning/integrated-surface-reviewer-bundle/v1"

// Only this review's semantic contract is normative. Execution/handoff docs
// are linked from it, not made new language-currentness dependencies.
var integratedSurfaceReviewerAuthorities = []string{
	"docs/LANGUAGE_DEFECT_EXHAUSTION.md",
	"docs/INTEGRATED_SURFACE_REVIEW.md", "docs/TRANSLATION_TERMINOLOGY.md", "docs/TERMINOLOGY_GUIDE.md",
}

// Public, first-party composition sources only. Never collect arbitrary runtime
// directories, deployment environment, credentials, or private configuration.
var integratedSurfaceComposition = []string{
	"internal/site/content.go", "internal/site/runtime.go", "internal/site/render.go", "internal/site/shell_render.go",
	"internal/site/tmplfunc/tmpl.go", "internal/site/tmplfunc/func.go",
	"internal/tour/production.go", "internal/tour/seo.go", "internal/tour/tour.go",
}

type IntegratedSurfaceReviewerOptions struct {
	Locale   string
	Closures []Reference
	Scope    []byte // optional canonical formal scope; mutually exclusive with Closures
}

type IntegratedSurfaceReviewerPayload struct {
	Asset          SurfaceAsset               `json:"asset"`
	Role           string                     `json:"role"`
	Source         i18n.TransportBundleFile   `json:"source"`
	Target         i18n.TransportBundleFile   `json:"target"`
	Representation *i18n.TransportBundleFile  `json:"source_representation,omitempty"`
	Generations    []Reference                `json:"structured_generation,omitempty"`
	Provenance     []i18n.TransportBundleFile `json:"provenance"`
}

type IntegratedSurfaceReviewerBundle struct {
	Schema              string                             `json:"schema"`
	Kind                string                             `json:"kind"`
	Version             int                                `json:"version"`
	Lifecycle           string                             `json:"lifecycle"`
	Locale              string                             `json:"locale"`
	ScopeIdentity       string                             `json:"scope_identity_sha256"`
	Scope               i18n.TransportBundleFile           `json:"scope"`
	ReviewRequiredCount int                                `json:"review_required_count"`
	CarriedCount        int                                `json:"carried_count"`
	Closures            []Reference                        `json:"package_closures"`
	Glossary            Reference                          `json:"glossary"`
	UnifiedReview       Reference                          `json:"unified_glossary_review"`
	Runtime             []Reference                        `json:"runtime_public_context"`
	Fresh               []IntegratedSurfaceReviewerPayload `json:"fresh_review_required"`
	Carried             []IntegratedSurfaceReviewerPayload `json:"context_only_carried"`
	Integration         []i18n.TransportBundleFile         `json:"integration_context"`
	Authority           []i18n.TransportBundleFile         `json:"authority"`
	Files               []i18n.TransportBundleFile         `json:"files"`
	Identity            string                             `json:"identity_sha256"`
}

type integratedSurfaceBundleWriter struct {
	v         *closureValidation
	entries   map[string]i18n.TransportBundleEntry
	inventory map[string]i18n.TransportBundleFile
}

// The legacy context exporter predates the strict shared file reader. Check
// its bounded public roots before invoking it; no symlink may redirect a
// source, canonical candidate, UI/metadata or template read outside the repo.
func guardIntegratedTourContext(v *closureValidation, locale string, catalog *i18n.Catalog) error {
	for _, dir := range []string{"_content/tour", "_content/js", "internal/tour", "locales/" + locale, "data/course-seo"} {
		parts := splitPath(dir)
		current := v.root
		for _, part := range parts {
			current = filepath.Join(current, part)
			st, err := os.Lstat(current)
			if err != nil {
				return err
			}
			if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("unsafe public context directory: %s", dir)
			}
		}
		if err := filepath.WalkDir(current, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink forbidden in public context: %s", dir)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	paths := []string{"internal/tour/ui/en.json", "internal/tour/ui/" + locale + ".json", "locales/" + locale + "/locale.json", "locales/" + locale + "/status.tsv", "locales/" + locale + "/article-metadata.json", "locales/" + locale + "/course-metadata.json"}
	targets, err := i18n.CanonicalTranslationUnitTargetPaths(catalog, locale)
	if err != nil {
		return err
	}
	for name := range targets {
		paths = append(paths, name)
	}
	for _, name := range paths {
		if _, err := v.readRegular(v.root, name); err != nil {
			return err
		}
	}
	return nil
}

func (w *integratedSurfaceBundleWriter) add(name, repo string, b []byte) (i18n.TransportBundleFile, error) {
	if !validPath(name) || name == "manifest.json" {
		return i18n.TransportBundleFile{}, fmt.Errorf("unsafe Surface bundle path")
	}
	if !strings.HasSuffix(name, ".zip") {
		if err := textBytes(b); err != nil {
			return i18n.TransportBundleFile{}, fmt.Errorf("%s: %w", name, err)
		}
	}
	f := i18n.NewTransportBundleFile(name, repo, b)
	if old, ok := w.inventory[name]; ok && old != f {
		return f, fmt.Errorf("conflicting Surface bundle payload: %s", name)
	}
	w.entries[name] = i18n.TransportBundleEntry{Path: name, Data: b}
	w.inventory[name] = f
	return f, nil
}
func (w *integratedSurfaceBundleWriter) reference(ref Reference) (i18n.TransportBundleFile, error) {
	if f, ok := w.inventory["evidence/"+ref.Path]; ok {
		if f.SHA256 != ref.SHA256 {
			return f, fmt.Errorf("conflicting Surface evidence SHA: %s", ref.Path)
		}
		return f, nil // immutable bytes remain in the invocation's final input guard
	}
	b, err := w.v.readRegular(w.v.root, ref.Path)
	if err != nil {
		return i18n.TransportBundleFile{}, err
	}
	if digest(b) != ref.SHA256 {
		return i18n.TransportBundleFile{}, fmt.Errorf("Surface context hash drift: %s", ref.Path)
	}
	return w.add("evidence/"+ref.Path, ref.Path, b)
}
func (w *integratedSurfaceBundleWriter) file(name string) (i18n.TransportBundleFile, error) {
	b, err := w.v.readRegular(w.v.root, name)
	if err != nil {
		return i18n.TransportBundleFile{}, err
	}
	return w.add("integration/"+name, name, b)
}

func integratedSurfaceBundleInputs(o IntegratedSurfaceReviewerOptions) (string, []Reference, *IntegratedSurfaceScope, error) {
	if len(o.Scope) > 0 {
		if len(o.Closures) > 0 {
			return "", nil, nil, fmt.Errorf("scope input and closure selectors are mutually exclusive")
		}
		var s IntegratedSurfaceScope
		if err := textBytes(o.Scope); err != nil {
			return "", nil, nil, err
		}
		if err := StrictJSON(o.Scope, &s); err != nil {
			return "", nil, nil, err
		}
		copy := s
		copy.Identity = ""
		canonical, err := Encode(s)
		if err != nil || s.Schema != IntegratedSurfaceSchema || !component(s.Locale) || s.Identity != identity(copy) || !bytes.Equal(canonical, o.Scope) || (o.Locale != "" && o.Locale != s.Locale) {
			return "", nil, nil, fmt.Errorf("invalid/noncanonical Integrated Surface scope")
		}
		return s.Locale, s.Closures, &s, nil
	}
	if !component(o.Locale) || len(o.Closures) == 0 {
		return "", nil, nil, fmt.Errorf("locale and prospective closures, or a current scope input, required")
	}
	refs := append([]Reference(nil), o.Closures...)
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	return o.Locale, refs, nil, nil
}

// ExportIntegratedSurfaceReviewerBundle uses the same current scope/closure
// readers as surface-record. No language judgment or lifecycle write occurs.
func ExportIntegratedSurfaceReviewerBundle(root string, o IntegratedSurfaceReviewerOptions) ([]byte, *IntegratedSurfaceReviewerBundle, error) {
	locale, closures, input, err := integratedSurfaceBundleInputs(o)
	if err != nil {
		return nil, nil, err
	}
	v := newClosureValidation(root)
	if err := v.currentInputs(locale); err != nil {
		return nil, nil, err
	}
	s, err := v.buildIntegratedSurfaceScope(root, locale, closures)
	if err != nil {
		return nil, nil, err
	}
	scopeBytes, err := Encode(s)
	if err != nil {
		return nil, nil, err
	}
	if input != nil && !bytes.Equal(scopeBytes, o.Scope) {
		return nil, nil, fmt.Errorf("Integrated Surface scope stale")
	}
	w := &integratedSurfaceBundleWriter{v: v, entries: map[string]i18n.TransportBundleEntry{}, inventory: map[string]i18n.TransportBundleFile{}}
	m := &IntegratedSurfaceReviewerBundle{Schema: IntegratedSurfaceReviewerBundleSchema, Kind: "integrated-surface-reviewer", Version: 1, Lifecycle: IntegratedSurfaceSchema, Locale: locale, ScopeIdentity: s.Identity, ReviewRequiredCount: len(s.Reviewed), CarriedCount: len(s.Carried), Closures: s.Closures, Glossary: Reference{"locales/" + locale + "/glossary.yaml", s.GlossarySHA}, UnifiedReview: s.UnifiedReview, Runtime: s.Runtime, Fresh: []IntegratedSurfaceReviewerPayload{}, Carried: []IntegratedSurfaceReviewerPayload{}, Integration: []i18n.TransportBundleFile{}, Files: []i18n.TransportBundleFile{}}
	if m.Scope, err = w.add("integrated-surface-scope.json", "", scopeBytes); err != nil {
		return nil, nil, err
	}
	for _, ref := range append([]Reference{m.Glossary, m.UnifiedReview}, s.Runtime...) {
		f, e := w.reference(ref)
		if e != nil {
			return nil, nil, e
		}
		m.Integration = append(m.Integration, f)
	}
	var review i18n.UnifiedGlossaryReview
	if err := v.readReference(root, s.UnifiedReview, &review); err != nil {
		return nil, nil, err
	}
	if _, err := w.reference(Reference{"data/unified-glossary-reviews/" + locale + "-bundles/" + review.BundleSHA + ".zip", review.BundleSHA}); err != nil {
		return nil, nil, err
	}
	coverage, _ := Encode(s.ExistingCoverage)
	coverageFile, err := w.add("integration/existing-coverage.json", "", coverage)
	if err != nil {
		return nil, nil, err
	}
	m.Integration = append(m.Integration, coverageFile)
	routes, _ := Encode(s.Routes)
	routesFile, err := w.add("integration/canonical-routes.json", "", routes)
	if err != nil {
		return nil, nil, err
	}
	m.Integration = append(m.Integration, routesFile)
	for _, name := range integratedSurfaceComposition {
		f, e := w.file(name)
		if e != nil {
			return nil, nil, e
		}
		m.Integration = append(m.Integration, f)
	}
	localeIdentity, err := w.file("locales/" + locale + "/locale.json")
	if err != nil {
		return nil, nil, err
	}
	m.Integration = append(m.Integration, localeIdentity)
	for _, name := range integratedSurfaceReviewerAuthorities {
		b, e := v.readRegular(root, name)
		if e != nil {
			return nil, nil, e
		}
		f, e := w.add("authority/"+name, name, b)
		if e != nil {
			return nil, nil, e
		}
		m.Authority = append(m.Authority, f)
	}
	// All prospective Page + structured files are useful composition context.
	// Their roles come exclusively from the current scope, never from a caller.
	type projected struct {
		source, target []byte
		document       *Document
		closure        *PackageClosure
		context        *i18n.GlossaryCompatibilityContext
	}
	projectedFiles := map[string]projected{}
	pageFinalizations := map[Reference]i18n.QualityCheckFinalization{}
	glob, err := v.loadCurrent(root)
	if err != nil {
		return nil, nil, err
	}
	// Runtime references include the immutable predecessor scope. Also expose
	// the checked effective registry after source adoption, so reviewers can
	// reason about current Homepage/shared-shell routing without repo discovery.
	effective, _ := Encode(glob)
	effectiveFile, e := w.add("integration/effective-content-scope.json", "", effective)
	if e != nil {
		return nil, nil, e
	}
	m.Integration = append(m.Integration, effectiveFile)
	if glob.ShellAdoption != nil {
		selector, e := w.file(ShellAdoptionCurrent)
		if e != nil {
			return nil, nil, e
		}
		m.Integration = append(m.Integration, selector)
		adoption, e := w.reference(*glob.ShellAdoption)
		if e != nil {
			return nil, nil, e
		}
		m.Integration = append(m.Integration, adoption)
		var a ShellAdoption
		if e := StrictJSON(w.entries[adoption.BundlePath].Data, &a); e != nil {
			return nil, nil, e
		}
		foundation, e := w.reference(a.Foundation)
		if e != nil {
			return nil, nil, e
		}
		m.Integration = append(m.Integration, foundation)
	}
	for _, ref := range s.Closures {
		checked := v.packageClosures[ref]
		if checked.closure == nil {
			return nil, nil, fmt.Errorf("missing checked closure graph node")
		}
		if _, err := w.reference(ref); err != nil {
			return nil, nil, err
		}
		for _, page := range closurePageFinalizations(checked.closure) {
			if !strings.HasPrefix(page.Path, "data/quality-check-snapshots/") {
				continue
			}
			entry, e := w.reference(page)
			if e != nil {
				return nil, nil, e
			}
			var final i18n.QualityCheckFinalization
			if e := StrictJSON(w.entries[entry.BundlePath].Data, &final); e != nil {
				return nil, nil, e
			}
			pageFinalizations[page] = final
		}
		docs, raw, e := CurrentPackageDocuments(root, glob, checked.closure.Package)
		if e != nil {
			return nil, nil, e
		}
		for _, d := range docs {
			d := d
			target, ok := checked.files[d.Path]
			if !ok {
				return nil, nil, fmt.Errorf("missing projected Surface target: %s", d.Path)
			}
			projectedFiles[d.Path] = projected{source: raw[d.Path], target: target, document: &d, closure: checked.closure}
		}
	}
	// Tour contexts already have representation-owned source/target semantics.
	// Support fresh contexts too, without turning the remaining carried Tour
	// inventory into independent fresh rows or guessing physical article slots.
	missingTour := map[string]SurfaceAsset{}
	for _, a := range s.Reviewed {
		if _, ok := projectedFiles[a.ID]; !ok && a.Package == "tour-v1" {
			missingTour[a.ID] = a
		}
	}
	if len(missingTour) > 0 {
		catalog, e := checkedCatalog(root)
		if e != nil {
			return nil, nil, e
		}
		if e := guardIntegratedTourContext(v, locale, catalog); e != nil {
			return nil, nil, e
		}
		contexts, e := i18n.LocaleLanguageContexts(root, locale, catalog)
		if e != nil {
			return nil, nil, e
		}
		for _, ctx := range contexts {
			a, ok := missingTour[ctx.Scope]
			if !ok {
				continue
			}
			if digest([]byte(ctx.Source)) != a.SourceSHA || digest([]byte(ctx.Target)) != a.TargetSHA {
				return nil, nil, fmt.Errorf("Tour semantic scope binding drift: %s", a.ID)
			}
			ctx := ctx
			projectedFiles[a.ID] = projected{source: []byte(ctx.Source), target: []byte(ctx.Target), context: &ctx}
		}
	}
	if e := v.bindLanguageAudit(root, locale); e != nil {
		return nil, nil, e
	}
	attention, err := i18n.NewLanguageAttention(root, locale)
	if err != nil {
		return nil, nil, err
	}
	makePayload := func(a SurfaceAsset, role string) (IntegratedSurfaceReviewerPayload, error) {
		p := IntegratedSurfaceReviewerPayload{Asset: a, Role: role, Provenance: []i18n.TransportBundleFile{}}
		file, ok := projectedFiles[a.ID]
		if !ok {
			return p, fmt.Errorf("Integrated Surface asset lacks full source/target payload: %s", a.ID)
		}
		if digest(file.target) != a.TargetSHA || (file.document != nil && file.document.SourceSHA != digest(file.source)) || (file.context != nil && a.SourceSHA != digest(file.source)) {
			return p, fmt.Errorf("Surface semantic payload binding mismatch: %s", a.ID)
		}
		if a.Package != "tour-v1" {
			if file.document == nil {
				return p, fmt.Errorf("missing formal language Document %s", a.ID)
			}
			result, e := scanFormalLanguageDocument(root, locale, *file.document, file.source, file.target)
			if e != nil {
				return p, e
			}
			if e := attention.Add("integrated-surface", a.ID, a.UnitID, surfaceAttentionUnitKind(a), result); e != nil {
				return p, e
			}
		}
		base := "payloads/" + digest([]byte(a.ID)) + "/"
		sourcePath := ""
		if file.document != nil {
			sourcePath = file.document.Path
		}
		if p.Source, err = w.add(base+"source.txt", sourcePath, file.source); err != nil {
			return p, err
		}
		if p.Target, err = w.add(base+"target.txt", "", file.target); err != nil {
			return p, err
		}
		var representation any = file.document
		if file.context != nil {
			representation = file.context
		}
		doc, _ := Encode(representation)
		rep, e := w.add(base+"source-representation.json", "", doc)
		if e != nil {
			return p, e
		}
		p.Representation = &rep
		refs := append([]Reference(nil), a.Evidence...)
		if file.context != nil {
			for _, ref := range file.context.References {
				refs = append(refs, Reference{ref.Path, ref.SHA256})
			}
		}
		if file.closure != nil && len(a.CandidateArtifacts) > 0 {
			refs = append(refs, closurePageFinalizations(file.closure)...)
		}
		for _, artifact := range a.CandidateArtifacts {
			unit := artifact.Unit
			for _, ref := range []Reference{{unit.CandidatePath, unit.CandidateSHA256}, {unit.ValidationPath, unit.ValidationSHA256}} {
				f, e := w.reference(ref)
				if e != nil {
					return p, e
				}
				p.Provenance = append(p.Provenance, f)
			}
			if digest(file.target) != unit.CandidateSHA256 {
				return p, fmt.Errorf("Page candidate differs from semantic target")
			}
			snapshot := "data/quality-check-snapshots/" + locale + "/" + artifact.SnapshotID + "/manifest.json"
			if file.closure != nil && len(closurePageFinalizations(file.closure)) > 0 {
				var final i18n.QualityCheckFinalization
				matches := 0
				for _, page := range closurePageFinalizations(file.closure) {
					if f, ok := pageFinalizations[page]; ok && f.SnapshotID == artifact.SnapshotID {
						final = f
						matches++
					}
				}
				if matches != 1 || final.SnapshotManifestPath != snapshot {
					return p, fmt.Errorf("Page finalization/Snapshot provenance mismatch")
				}
				refs = append(refs, Reference{snapshot, final.SnapshotManifestSHA256})
			} else {
				b, e := v.readRegular(root, snapshot)
				if e != nil {
					return p, e
				}
				refs = append(refs, Reference{snapshot, digest(b)})
			}
		}
		var generationRefs []Reference
		if file.closure != nil {
			generationRefs = file.closure.Structured
		}
		for _, gen := range generationRefs {
			frozen, ok := v.generations[gen]
			if !ok {
				return p, fmt.Errorf("missing verified generation graph node")
			}
			r, targets := frozen.generation, frozen.files
			for _, asset := range r.Plan.Assets {
				if asset.Document != nil && asset.Document.Path == a.ID && digest(targets[asset.TargetPath]) == a.TargetSHA {
					p.Generations = append(p.Generations, gen)
					refs = append(refs, gen)
				}
			}
		}
		seen := map[Reference]bool{}
		for _, ref := range refs {
			if seen[ref] {
				continue
			}
			seen[ref] = true
			f, e := w.reference(ref)
			if e != nil {
				return p, e
			}
			p.Provenance = append(p.Provenance, f)
		}
		return p, nil
	}
	for _, a := range s.Reviewed {
		p, e := makePayload(a, "fresh_review_required")
		if e != nil {
			return nil, nil, e
		}
		m.Fresh = append(m.Fresh, p)
	}
	for _, a := range s.Carried {
		// These remain context payloads. A fresh Page composition judgment may
		// address their existing shared structured owner via its sealed closure;
		// the recorder/selector derive that relation without changing row roles.
		if _, ok := projectedFiles[a.ID]; ok {
			p, e := makePayload(a, "context_only_carried")
			if e != nil {
				return nil, nil, e
			}
			m.Carried = append(m.Carried, p)
		}
	}
	// Reuse the existing complete Tour semantic/context exporter strictly as
	// carried integration context. Its legacy CLI/schema/authority are untouched.
	for _, c := range s.ExistingCoverage.Packages {
		if c.Package == "tour-v1" {
			catalog, e := checkedCatalog(root)
			if e != nil {
				return nil, nil, e
			}
			if e := guardIntegratedTourContext(v, locale, catalog); e != nil {
				return nil, nil, e
			}
			b, _, e := i18n.ExportLocaleSurfaceReviewPackage(root, locale, catalog)
			if e != nil {
				return nil, nil, e
			}
			var context i18n.LocaleSurfaceReviewPackage
			if e := StrictJSON(b, &context); e != nil {
				return nil, nil, e
			}
			if context.LanguageAttention == nil {
				return nil, nil, fmt.Errorf("Tour context missing language attention")
			}
			attention.Candidates = append(attention.Candidates, context.LanguageAttention.Candidates...)
			for _, other := range context.OtherSurfaces {
				if e := v.bind(Reference{other.Path, other.SHA256}); e != nil {
					return nil, nil, e
				}
			}
			f, e := w.add("integration/context-only-tour.json", "", b)
			if e != nil {
				return nil, nil, e
			}
			m.Integration = append(m.Integration, f)
		}
	}
	if err := attention.Verify(root); err != nil {
		return nil, nil, err
	}
	attentionData, e := Encode(attention)
	if e != nil {
		return nil, nil, e
	}
	af, e := w.add(i18n.LanguageAttentionPath, "", attentionData)
	if e != nil {
		return nil, nil, e
	}
	m.Integration = append(m.Integration, af)
	template, _ := Encode(IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *s, Findings: []string{}, ExactFindings: []SurfaceFinding{}})
	if _, err := w.add("independent-surface-result.template.json", "", template); err != nil {
		return nil, nil, err
	}
	for name := range w.inventory {
		m.Files = append(m.Files, w.inventory[name])
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].BundlePath < m.Files[j].BundlePath })
	m.Identity = identity(*m)
	manifest, err := Encode(m)
	if err != nil {
		return nil, nil, err
	}
	entries := []i18n.TransportBundleEntry{}
	totalBytes := len(manifest)
	for _, entry := range w.entries {
		entries = append(entries, entry)
		totalBytes += len(entry.Data)
	}
	if len(entries)+1 > 8192 || totalBytes > 128<<20 {
		return nil, nil, fmt.Errorf("Integrated Surface Reviewer transport exceeds shared bounded inventory limits")
	}
	if err := v.verify(); err != nil {
		return nil, nil, err
	}
	b, err := i18n.WriteDeterministicTransportBundle(manifest, entries)
	return b, m, err
}

func readIntegratedSurfaceReviewerBundle(b []byte) (*IntegratedSurfaceReviewerBundle, map[string][]byte, error) {
	files, err := i18n.ReadTransportBundle(b, 8192, 128<<20)
	if err != nil {
		return nil, nil, err
	}
	var m IntegratedSurfaceReviewerBundle
	if err := StrictJSON(files["manifest.json"], &m); err != nil {
		return nil, nil, err
	}
	copy := m
	copy.Identity = ""
	if m.Schema != IntegratedSurfaceReviewerBundleSchema || m.Kind != "integrated-surface-reviewer" || m.Version != 1 || m.Lifecycle != IntegratedSurfaceSchema || !component(m.Locale) || m.Identity != identity(copy) {
		return nil, nil, fmt.Errorf("invalid Integrated Surface bundle identity/schema")
	}
	if err := i18n.ValidateTransportBundleInventory(files, m.Files, true); err != nil {
		return nil, nil, err
	}
	for name, b := range files {
		if !strings.HasSuffix(name, ".zip") {
			if err := textBytes(b); err != nil {
				return nil, nil, err
			}
		}
	}
	var scope IntegratedSurfaceScope
	if err := StrictJSON(files[m.Scope.BundlePath], &scope); err != nil {
		return nil, nil, err
	}
	if m.Scope.BundlePath != "integrated-surface-scope.json" || digest(files[m.Scope.BundlePath]) != m.Scope.SHA256 || scope.Identity != m.ScopeIdentity || scope.Locale != m.Locale || m.ReviewRequiredCount != len(scope.Reviewed) || m.CarriedCount != len(scope.Carried) || len(m.Fresh) != len(scope.Reviewed) {
		return nil, nil, fmt.Errorf("Integrated Surface bundle scope/count mismatch")
	}
	for i, p := range m.Fresh {
		if p.Role != "fresh_review_required" || !surfaceAssetEqual(p.Asset, scope.Reviewed[i]) || digest(files[p.Source.BundlePath]) != p.Source.SHA256 || digest(files[p.Target.BundlePath]) != p.Asset.TargetSHA {
			return nil, nil, fmt.Errorf("Integrated Surface fresh payload mismatch")
		}
	}
	return &m, files, nil
}

func surfaceAssetEqual(a, b SurfaceAsset) bool { return identity(a) == identity(b) }

func CheckIntegratedSurfaceReviewerBundle(root string, b []byte) (*IntegratedSurfaceReviewerBundle, error) {
	m, files, err := readIntegratedSurfaceReviewerBundle(b)
	if err != nil {
		return nil, err
	}
	want, _, err := ExportIntegratedSurfaceReviewerBundle(root, IntegratedSurfaceReviewerOptions{Scope: files[m.Scope.BundlePath]})
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(b, want) {
		return nil, fmt.Errorf("Integrated Surface reviewer bundle stale/noncanonical")
	}
	return m, nil
}

func surfaceAttentionUnitKind(a SurfaceAsset) i18n.UnitKind {
	if len(a.CandidateArtifacts) == 1 {
		return a.CandidateArtifacts[0].Unit.UnitKind
	}
	return ""
}
