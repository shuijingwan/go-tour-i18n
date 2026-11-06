package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const maxVerifiedBytes = 512 << 20

// A validation graph belongs to one read-only invocation, never to a root or
// locale across commands. Only successful immutable nodes are memoized. All
// observed dependencies (including opaque shared Page integrity proofs) remain
// hash-bound and are checked before reuse and before returning the result.
type closureValidation struct {
	readonlySuccessors    *structuredSuccessorValidation // explicit readonly planner owner only
	profile               func(string) func()            // optional test instrumentation
	metrics               map[string]int
	verifiedBytes         map[string][]byte
	verifiedBytesSize     int
	root                  string
	deps                  map[string]string
	dirs                  map[string][]string
	closures              map[Reference]*structuredBaseline
	generations           map[Reference]frozenStructuredValidation
	current               map[currentStructuredKey]currentStructuredValidation
	active                map[Reference]bool
	counts                map[string]int
	reviews               map[string]i18n.GlossaryArchiveReference
	corpusChecked         map[string]bool
	absent                map[string]bool
	modes                 map[string]os.FileMode
	disabled              bool // differential tests only; never shared across invocations
	frames                []*validationProof
	proofs                map[string]*validationProof
	historicalContexts    map[historicalTranslationContextKey]historicalTranslationContext
	global                *Global
	catalog               *i18n.Catalog
	packageClosures       map[Reference]validatedPackageClosure
	reconciliations       map[Reference][]cachedPageReplacement
	bridgeReconstructions map[bridgeReconstructionKey][]cachedBridgeReconstruction
}

func (v *closureValidation) phase(name string) func() {
	if v.profile != nil {
		return v.profile(name)
	}
	return func() {}
}
func (v *closureValidation) count(name string) {
	if v.metrics != nil {
		v.metrics[name]++
	}
}

type validatedPackageClosure struct {
	closure *PackageClosure
	files   map[string][]byte
}

type validationProof struct {
	files              map[string]string
	dirs               map[string][]string
	historicalContexts map[historicalTranslationContextKey]historicalTranslationContext
}

func (v *closureValidation) beginProof() *validationProof {
	p := &validationProof{files: map[string]string{}, dirs: map[string][]string{}}
	v.frames = append(v.frames, p)
	return p
}
func (v *closureValidation) endProof() { v.frames = v.frames[:len(v.frames)-1] }
func (v *closureValidation) reuse(key string) error {
	p := v.proofs[key]
	if p == nil {
		return fmt.Errorf("missing validation graph proof")
	}
	for _, context := range p.historicalContexts {
		if err := v.bindHistoricalTranslationReference(context.catalog, context.proof, context.reference); err != nil {
			return err
		}
	}
	for name, sha := range p.files {
		if err := v.bind(Reference{name, sha}); err != nil {
			return err
		}
	}
	for name := range p.dirs {
		if err := v.directory(name); err != nil {
			return err
		}
	}
	return nil
}

type frozenStructuredValidation struct {
	generation    *StructuredGeneration
	files, inputs map[string][]byte
}
type currentStructuredKey struct {
	ref Reference
	pkg string
}
type currentStructuredValidation struct {
	generation *StructuredGeneration
	files      map[string][]byte
}

func newClosureValidation(root string) *closureValidation {
	return &closureValidation{root: root, deps: map[string]string{}, dirs: map[string][]string{}, closures: map[Reference]*structuredBaseline{}, generations: map[Reference]frozenStructuredValidation{}, current: map[currentStructuredKey]currentStructuredValidation{}, active: map[Reference]bool{}, counts: map[string]int{}, reviews: map[string]i18n.GlossaryArchiveReference{}, corpusChecked: map[string]bool{}, absent: map[string]bool{}, proofs: map[string]*validationProof{}, modes: map[string]os.FileMode{}}
}

func (v *closureValidation) readRegular(root, name string) ([]byte, error) {
	if root != v.root {
		return nil, fmt.Errorf("validation root changed")
	}
	b, err := readRegular(root, name)
	if err != nil {
		return nil, err
	}
	sha := v.deps[name]
	if previous, ok := v.verifiedBytes[name]; ok && !v.disabled && sha != "" {
		// Re-read through the regular-file safety checks on every access. Exact
		// byte equality to the privately owned, SHA-verified observation is
		// stronger than rehashing it; size/mtime alone never authorize reuse.
		if !bytes.Equal(previous, b) {
			return nil, fmt.Errorf("validation input changed during invocation: %s", name)
		}
	} else {
		sha = digest(b)
		v.counts["sha256:"+name]++
		if !v.disabled && v.verifiedBytesSize+len(b) <= maxVerifiedBytes {
			if v.verifiedBytes == nil {
				v.verifiedBytes = map[string][]byte{}
			}
			v.verifiedBytes[name] = bytes.Clone(b)
			v.verifiedBytesSize += len(b)
		}
	}
	st, err := os.Stat(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return nil, err
	}
	if old, ok := v.modes[name]; ok && old != st.Mode() {
		return nil, fmt.Errorf("validation input mode changed: %s", name)
	}
	v.modes[name] = st.Mode()
	if old, ok := v.deps[name]; ok && old != sha {
		return nil, fmt.Errorf("validation input changed during invocation: %s", name)
	}
	v.deps[name] = sha
	for _, p := range v.frames {
		p.files[name] = sha
	}
	return b, nil
}

func (v *closureValidation) readReference(root string, ref Reference, out any) error {
	v.count("read-reference:" + ref.Path)
	b, err := v.readRegular(root, ref.Path)
	if err != nil {
		return err
	}
	if !validSHA(ref.SHA256) || v.deps[ref.Path] != ref.SHA256 {
		return fmt.Errorf("evidence hash mismatch: %s", ref.Path)
	}
	return StrictJSON(b, out)
}

func (v *closureValidation) bind(ref Reference) error {
	_, err := v.readRegular(v.root, ref.Path)
	if err != nil {
		return err
	}
	if !validSHA(ref.SHA256) || v.deps[ref.Path] != ref.SHA256 {
		return fmt.Errorf("validation dependency hash mismatch: %s", ref.Path)
	}
	return nil
}

func (v *closureValidation) directory(name string) error {
	entries, err := os.ReadDir(filepath.Join(v.root, filepath.FromSlash(name)))
	if os.IsNotExist(err) {
		entries, err = nil, nil
	}
	if err != nil {
		return err
	}
	names := []string{}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			return fmt.Errorf("unsafe validation directory member: %s", name)
		}
		names = append(names, e.Name())
	}
	if old, ok := v.dirs[name]; ok && !reflect.DeepEqual(old, names) {
		return fmt.Errorf("validation directory changed: %s", name)
	}
	v.dirs[name] = names
	for _, p := range v.frames {
		p.dirs[name] = names
	}
	for _, member := range names {
		if _, err := v.readRegular(v.root, filepath.ToSlash(filepath.Join(name, member))); err != nil {
			return err
		}
	}
	return nil
}

func (v *closureValidation) verify() error {
	for _, context := range v.historicalContexts {
		if err := v.bindHistoricalTranslationReference(context.catalog, context.proof, context.reference); err != nil {
			return err
		}
	}
	for name := range v.absent {
		if _, err := readRegular(v.root, name); !os.IsNotExist(err) {
			return fmt.Errorf("validation input appeared/unsafe: %s", name)
		}
	}
	for name, sha := range v.deps {
		_, err := v.readRegular(v.root, name)
		if err != nil || v.deps[name] != sha {
			return fmt.Errorf("validation dependency changed: %s", name)
		}
		st, err := os.Stat(filepath.Join(v.root, filepath.FromSlash(name)))
		if err != nil || st.Mode() != v.modes[name] {
			return fmt.Errorf("validation dependency mode changed: %s", name)
		}
	}
	for name := range v.dirs {
		if err := v.directory(name); err != nil {
			return err
		}
	}
	return nil
}

func (v *closureValidation) optional(name string) error {
	_, err := v.readRegular(v.root, name)
	if os.IsNotExist(err) {
		v.absent[name] = true
		return nil
	}
	return err
}

// Capture the mutable world before validation. This is an input guard, not a
// substitute for LoadCurrent, corpus, review, source or authorization checks.
func (v *closureValidation) currentInputs(locale string) error {
	defer v.phase("currentInputs")()
	key := "current-inputs:" + locale
	if _, ok := v.proofs[key]; ok && !v.disabled {
		return v.reuse(key)
	}
	proof := v.beginProof()
	defer v.endProof()
	for _, name := range []string{GlobalPath, i18n.GlossaryCorpusPath, "locales/" + locale + "/glossary.yaml"} {
		if _, err := v.readRegular(v.root, name); err != nil {
			return err
		}
	}
	for _, name := range []string{SourceManifestPath, SnapshotPath, LocalePath(locale), PackageMembershipPath, PageRegistryPath} {
		if err := v.optional(name); err != nil {
			return err
		}
	}
	if _, missing := v.absent[PackageMembershipPath]; !missing {
		var ref Reference
		if err := v.readReference(v.root, Reference{Path: PackageMembershipPath, SHA256: v.deps[PackageMembershipPath]}, &ref); err != nil {
			return err
		}
		var m PackageMembership
		if err := v.readReference(v.root, ref, &m); err != nil {
			return err
		}
		if err := v.bind(m.PreviousRegistry); err != nil {
			return err
		}
		if err := v.bind(m.Registry); err != nil {
			return err
		}
	}
	var corpus i18n.GlossarySourceCorpus
	if err := v.readReference(v.root, Reference{Path: i18n.GlossaryCorpusPath, SHA256: v.deps[i18n.GlossaryCorpusPath]}, &corpus); err != nil {
		return err
	}
	for _, ref := range corpus.Sources {
		if ref.Path == SnapshotPath {
			continue
		} // resolved by frozen source authority below
		if _, err := v.readRegular(v.root, ref.Path); err != nil {
			return err
		}
	}
	if _, missing := v.absent[SourceManifestPath]; !missing {
		m, err := loadSourceManifest(v.root)
		if err != nil {
			return err
		}
		for _, f := range m.Files {
			if _, err := v.readRegular(v.root, f.Path); err != nil {
				return err
			}
		}
	}
	if err := v.currentReviews(locale); err != nil {
		return err
	}
	v.proofs[key] = proof
	return nil
}

func (v *closureValidation) currentReviews(locale string) error {
	dir := "data/unified-glossary-reviews/" + locale
	if err := v.directory(dir); err != nil {
		return err
	}
	for _, name := range v.dirs[dir] {
		b, err := v.readRegular(v.root, dir+"/"+name)
		if err != nil {
			return err
		}
		var r i18n.UnifiedGlossaryReview
		if err := StrictJSON(b, &r); err != nil {
			return err
		}
		if err := v.bind(Reference{Path: "data/unified-glossary-reviews/" + locale + "-bundles/" + r.BundleSHA + ".zip", SHA256: r.BundleSHA}); err != nil {
			return err
		}
	}
	return nil
}

func (v *closureValidation) unifiedReview(locale string) (i18n.GlossaryArchiveReference, error) {
	key := "review:" + locale
	if ref, ok := v.reviews[locale]; ok && !v.disabled {
		return ref, v.reuse(key)
	}
	proof := v.beginProof()
	defer v.endProof()
	for _, name := range []string{i18n.GlossaryCorpusPath, "locales/" + locale + "/glossary.yaml"} {
		if _, err := v.readRegular(v.root, name); err != nil {
			return i18n.GlossaryArchiveReference{}, err
		}
	}
	if err := v.currentReviews(locale); err != nil {
		return i18n.GlossaryArchiveReference{}, err
	}
	v.counts["current-review"]++
	r, err := i18n.RequireUnifiedGlossaryReview(v.root, locale)
	if err == nil {
		v.reviews[locale] = r
		v.proofs[key] = proof
	}
	return r, err
}

func (v *closureValidation) requireUnifiedReview(root, locale string) error {
	if !v.corpusChecked[locale] || v.disabled {
		v.counts["current-corpus"]++
		if err := CheckUnifiedCorpus(root, v.readonlySuccessors); err != nil {
			return err
		}
		v.corpusChecked[locale] = true
	}
	_, err := v.unifiedReview(locale)
	return err
}

func (v *closureValidation) historicalStructuredClosure(root string, ref Reference, visiting map[string]bool) (*structuredBaseline, error) {
	if root != v.root {
		return nil, fmt.Errorf("validation root changed")
	}
	key := "closure:" + ref.Path + ":" + ref.SHA256
	if visiting[ref.Path] || v.active[ref] {
		return nil, fmt.Errorf("cyclic structured baseline")
	}
	if cached := v.closures[ref]; cached != nil && !v.disabled {
		if err := v.reuse(key); err != nil {
			return nil, err
		}
		copy := *cached
		return &copy, nil
	}
	v.active[ref] = true
	defer delete(v.active, ref)
	proof := v.beginProof()
	defer v.endProof()
	v.counts["closure:"+ref.Path]++
	b, err := v.computeHistoricalStructuredClosure(root, ref, visiting)
	if err == nil {
		copy := *b
		v.closures[ref] = &copy
		v.proofs[key] = proof
	}
	return b, err
}

func (v *closureValidation) historicalStructuredGeneration(root string, ref Reference) (*StructuredGeneration, map[string][]byte, map[string][]byte, error) {
	if root != v.root {
		return nil, nil, nil, fmt.Errorf("validation root changed")
	}
	key := "generation:" + ref.Path + ":" + ref.SHA256
	if v.active[ref] {
		return nil, nil, nil, fmt.Errorf("cyclic structured generation")
	}
	if cached, ok := v.generations[ref]; ok && !v.disabled {
		return cached.generation, cached.files, cached.inputs, v.reuse(key)
	}
	v.active[ref] = true
	defer delete(v.active, ref)
	proof := v.beginProof()
	defer v.endProof()
	if err := v.bind(ref); err != nil {
		return nil, nil, nil, err
	}
	v.counts["generation:"+ref.Path]++
	r, files, inputs, err := v.computeHistoricalStructuredGeneration(root, ref)
	if err == nil {
		v.generations[ref] = frozenStructuredValidation{r, files, inputs}
		v.proofs[key] = proof
	}
	return r, files, inputs, err
}

func (v *closureValidation) checkStructuredGeneration(root string, ref Reference, pkg string) (*StructuredGeneration, map[string][]byte, error) {
	if root != v.root {
		return nil, nil, fmt.Errorf("validation root changed")
	}
	key := currentStructuredKey{ref, pkg}
	proofKey := "current:" + ref.Path + ":" + ref.SHA256 + ":" + pkg
	if cached, ok := v.current[key]; ok && !v.disabled {
		return cached.generation, cached.files, v.reuse(proofKey)
	}
	proof := v.beginProof()
	defer v.endProof()
	v.counts["current-generation:"+ref.Path+":"+pkg]++
	r, files, err := v.computeCurrentStructuredGeneration(root, ref, pkg)
	if err == nil {
		v.current[key] = currentStructuredValidation{r, files}
		v.proofs[proofKey] = proof
	}
	return r, files, err
}

// Existing entry points retain their contracts; recursion shares only this
// explicit local graph. No cache survives a command or accepts a historical
// PASS in place of the first full validation.
func BuildPackageClosure(root, locale, pkg string, page *Reference, structured []Reference) (*PackageClosure, map[string][]byte, error) {
	v := newClosureValidation(root)
	if err := v.currentInputs(locale); err != nil {
		return nil, nil, err
	}
	c, files, err := v.buildPackageClosure(root, locale, pkg, page, structured)
	if err == nil {
		err = v.verify()
	}
	return c, files, err
}
func historicalStructuredClosure(root string, ref Reference, visiting map[string]bool) (*structuredBaseline, error) {
	v := newClosureValidation(root)
	b, err := v.historicalStructuredClosure(root, ref, visiting)
	if err == nil {
		err = v.verify()
	}
	return b, err
}
func historicalStructuredGeneration(root string, ref Reference) (*StructuredGeneration, map[string][]byte, map[string][]byte, error) {
	v := newClosureValidation(root)
	r, files, inputs, err := v.historicalStructuredGeneration(root, ref)
	if err == nil {
		err = v.verify()
	}
	return r, files, inputs, err
}
func validateFrozenStructuredGeneration(root string, r *StructuredGeneration, m *StructuredBundle, inputs map[string][]byte) (*StructuredGeneration, map[string][]byte, map[string][]byte, error) {
	return newClosureValidation(root).validateFrozenStructuredGeneration(root, r, m, inputs)
}
func verifyFrozenStructuredDelta(root string, r *StructuredGeneration, inputs map[string][]byte) error {
	return newClosureValidation(root).verifyFrozenStructuredDelta(root, r, inputs)
}
func validateStructuredDelta(root string, r *StructuredGeneration, m *StructuredBundle) error {
	return newClosureValidation(root).validateStructuredDelta(root, r, m)
}
func mergeStructuredBaselineDelta(root, locale, pkg string, refs []Reference) (map[string][]byte, *Reference, bool, error) {
	return newClosureValidation(root).mergeStructuredBaselineDelta(root, locale, pkg, refs)
}
func checkStructuredGeneration(root string, ref Reference, pkg string) (*StructuredGeneration, map[string][]byte, error) {
	v := newClosureValidation(root)
	var generation StructuredGeneration
	if err := v.readReference(root, ref, &generation); err != nil {
		return nil, nil, err
	}
	if err := v.currentInputs(generation.Plan.Locale); err != nil {
		return nil, nil, err
	}
	r, files, err := v.checkStructuredGeneration(root, ref, pkg)
	if err == nil {
		err = v.verify()
	}
	return r, files, err
}
func assessStructuredAsset(root string, p StructuredPlan, a StructuredAsset, baseline *structuredBaseline) (*StructuredAssessment, error) {
	return newClosureValidation(root).assessStructuredAsset(root, p, a, baseline)
}
func historicalStructuredCompletion(root, locale string, c Completion) (*structuredBaseline, error) {
	return newClosureValidation(root).historicalStructuredCompletion(root, locale, c)
}
func (v *closureValidation) validateHistoricalIntegratedSurface(root, locale string, ref, closure Reference) error {
	if err := validateHistoricalIntegratedSurface(root, locale, ref, closure); err != nil {
		return err
	}
	var r IntegratedSurfaceReceipt
	if err := v.readReference(root, ref, &r); err != nil {
		return err
	}
	return v.bind(r.Evidence)
}

func (v *closureValidation) review(locale string, ref Reference) error {
	if err := v.directory("data/unified-glossary-reviews/" + locale); err != nil {
		return err
	}
	var review struct {
		BundleSHA string `json:"reviewer_bundle_sha256"`
	}
	// Full schema/role/archive validation is still performed by the canonical
	// reader. This observer only binds its transitive filesystem dependencies.
	b, err := v.readRegular(v.root, ref.Path)
	if err != nil {
		return err
	}
	if digest(b) != ref.SHA256 {
		return fmt.Errorf("Review hash mismatch")
	}
	if err := json.Unmarshal(b, &review); err != nil {
		return err
	}
	return v.bind(Reference{Path: "data/unified-glossary-reviews/" + locale + "-bundles/" + review.BundleSHA + ".zip", SHA256: review.BundleSHA})
}

func (v *closureValidation) loadCurrent(root string) (*Global, error) {
	// The world was captured before validation and is verified at invocation
	// return. Reuse this graph node, not a saved PASS from another command.
	if _, err := v.readRegular(root, GlobalPath); err != nil {
		return nil, err
	}
	if v.global != nil && !v.disabled {
		return v.global, nil
	}
	v.counts["current-global"]++
	g, err := LoadCurrent(root, v.readonlySuccessors)
	if err == nil {
		// Observe physical repository dependencies. Upstream logical paths (for
		// example internal/web/page.go) live in the already guarded source
		// manifest/ZIP, not necessarily at that path in this repository.
		for _, pkg := range g.Packages {
			for _, dep := range pkg.Dependencies {
				if err := v.optional(dep.Path); err != nil {
					return nil, err
				}
			}
		}
		for _, source := range g.Sources {
			for _, dep := range source.Dependencies {
				if err := v.optional(dep.Path); err != nil {
					return nil, err
				}
			}
		}
		if g.ShellAdoption != nil {
			var a ShellAdoption
			if err := v.readReference(root, *g.ShellAdoption, &a); err != nil {
				return nil, err
			}
			if err := v.bind(a.Foundation); err != nil {
				return nil, err
			}
		}
		v.global = g
	}
	return g, err
}

func (v *closureValidation) packageCatalog(pkg string) (*i18n.Catalog, error) {
	defer v.phase("packageCatalog:" + pkg)()
	v.count("package-catalog:" + pkg)
	c, err := v.inventoryCatalog()
	if err != nil {
		return nil, err
	}
	if err := c.UseTranslationPackage(pkg); err != nil {
		return nil, err
	}
	return c, nil
}

func (v *closureValidation) inventoryCatalog() (*i18n.Catalog, error) {
	defer v.phase("inventoryCatalog")()
	v.count("inventory-catalog-call")
	if v.catalog == nil || v.disabled {
		v.counts["registered-catalog"]++
		c, err := checkedCatalog(v.root)
		if err != nil {
			return nil, err
		}
		register := func(successors *structuredSuccessorValidation) error {
			defer v.phase("AttachRegisteredTranslationPackages")()
			return AttachRegisteredTranslationPackages(v.root, c, successors)
		}
		if v.readonlySuccessors != nil {
			err = register(v.readonlySuccessors)
		} else {
			// Standalone inventory readers own the same bounded readonly lifetime
			// as the planner; public registration and publication stay eager.
			err = withReadonlyStructuredValidation(v.root, register)
		}
		if err != nil {
			return nil, err
		}
		AttachIntegratedSurfaceAuthority(c)
		v.attachReplacementReuse(c)
		v.catalog = c
	}
	c := *v.catalog
	return &c, nil
}

func (v *closureValidation) checkPackageClosure(root string, ref Reference) (*PackageClosure, map[string][]byte, error) {
	defer v.phase("closure:" + ref.Path)()
	v.count("closure-call:" + ref.Path)
	if root != v.root {
		return nil, nil, fmt.Errorf("validation root changed")
	}
	key := "current-closure:" + ref.Path + ":" + ref.SHA256
	if c, ok := v.packageClosures[ref]; ok && !v.disabled {
		return c.closure, c.files, v.reuse(key)
	}
	proof := v.beginProof()
	defer v.endProof()
	v.counts[key]++
	c, files, err := v.computePackageClosure(root, ref)
	if err == nil {
		if v.packageClosures == nil {
			v.packageClosures = map[Reference]validatedPackageClosure{}
		}
		v.packageClosures[ref] = validatedPackageClosure{c, files}
		v.proofs[key] = proof
	}
	return c, files, err
}

func CheckPackageClosure(root string, ref Reference) (*PackageClosure, map[string][]byte, error) {
	v := newClosureValidation(root)
	out, files, err := v.checkedPackageClosure(root, ref)
	if err == nil {
		err = v.verify()
	}
	return out, files, err
}

// Preserve one graph across coverage eligibility and payload projection. Every
// use still checks the captured bytes/directories; the caller verifies at exit.
func (v *closureValidation) checkedPackageClosure(root string, ref Reference) (*PackageClosure, map[string][]byte, error) {
	// Preserve historical readers' independence of current language eligibility.
	var c PackageClosure
	if err := v.readReference(root, ref, &c); err != nil {
		return nil, nil, err
	}
	g, err := v.loadCurrent(root)
	if err != nil {
		return nil, nil, err
	}
	historical := false
	for _, p := range g.HistoricalPackages {
		historical = historical || c.Package == p.ID && c.PackageSHA == p.Identity
	}
	if !historical {
		if err := v.currentInputs(c.Locale); err != nil {
			return nil, nil, err
		}
	}
	return v.checkPackageClosure(root, ref)
}

func BuildIntegratedSurfaceScope(root, locale string, closures []Reference) (*IntegratedSurfaceScope, error) {
	v := newClosureValidation(root)
	if err := v.currentInputs(locale); err != nil {
		return nil, err
	}
	return v.buildIntegratedSurfaceScopeReadonly(root, locale, closures)
}

// Reuse the existing source-lineage graph for this readonly invocation. Both
// graphs retain their final byte/mode/absence/directory guards before success.
func (v *closureValidation) buildIntegratedSurfaceScopeReadonly(root, locale string, closures []Reference) (*IntegratedSurfaceScope, error) {
	var scope *IntegratedSurfaceScope
	err := withReadonlyStructuredValidation(root, func(successors *structuredSuccessorValidation) error {
		v.readonlySuccessors = successors
		defer v.phase("integrated-surface-scope")()
		var err error
		scope, err = v.buildIntegratedSurfaceScope(root, locale, closures)
		if err != nil {
			return err
		}
		return v.verify()
	})
	return scope, err
}

func structuredSurfaceCarrySet(root string, c *PackageClosure) (map[string]bool, []Reference, error) {
	return newClosureValidation(root).structuredSurfaceCarrySet(root, c)
}

func (v *closureValidation) projectSharedPackageFinalization(root, locale, pkg string, ref Reference, out map[string][]byte) error {
	defer v.phase("Page-finalization:" + pkg)()
	v.count("quality-finalization:" + ref.Path)
	var f i18n.QualityCheckFinalization
	if err := v.readReference(root, ref, &f); err != nil {
		return err
	}
	if f.Locale != locale || ref.Path != "data/quality-check-snapshots/"+locale+"/"+f.SnapshotID+"/finalization.json" {
		return fmt.Errorf("shared finalization scope/path mismatch")
	}
	c, err := v.packageCatalog(pkg)
	if err != nil {
		return err
	}
	files, err := i18n.FinalizedTranslationPackageCandidates(root, c, locale, f.SnapshotID)
	if err != nil {
		return err
	}
	for name, b := range files {
		out[name] = b
	}
	return nil
}
