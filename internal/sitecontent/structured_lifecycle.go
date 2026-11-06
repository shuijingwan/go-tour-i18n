package sitecontent

import (
	"bytes"
	"fmt"
	"io/fs"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

// Assessment extends the existing structured plan; it is machine authority,
// never a replacement language judgment or a synthetic Surface finding.
type StructuredAssessment struct {
	RepresentationSuccessor *Reference `json:"shared_representation_successor,omitempty"`
	SourceSuccessor         *Reference `json:"structured_source_successor,omitempty"`
	Classification          string     `json:"classification"`
	Reason                  string     `json:"reason"`
	Closure                 Reference  `json:"baseline_closure"`
	Generation              Reference  `json:"baseline_generation"`
	Surface                 Reference  `json:"baseline_surface"`
	OldGlossarySHA          string     `json:"old_glossary_sha256"`
	GlossarySHA             string     `json:"current_glossary_sha256"`
	Review                  Reference  `json:"current_unified_review"`
	SourceSHA               string     `json:"source_sha256"`
	TargetSHA               string     `json:"accepted_target_sha256"`
	InputReopen             *Reference `json:"structured_input_reopen,omitempty"`
	Identity                string     `json:"identity_sha256"`
}

type structuredBaseline struct {
	Closure             PackageClosure
	ClosureRef, Surface Reference
	Files               map[string][]byte
	Owners              map[string]Reference
	Generations         map[string]*StructuredGeneration
	Inputs              map[string]map[string][]byte
}

func (validation *closureValidation) historicalStructuredCompletion(root, locale string, c Completion) (*structuredBaseline, error) {
	if c.EvidenceKind != PackageCompletionKind || len(c.Evidence) != 2 || c.State != "complete" || c.EvidenceIdentity != identity(c.Evidence) {
		return nil, fmt.Errorf("invalid historical structured completion")
	}
	b, err := validation.historicalStructuredClosure(root, c.Evidence[0], map[string]bool{})
	if err != nil {
		return nil, err
	}
	if b.Closure.Locale != locale || b.Closure.Package != c.Package || b.Closure.PackageSHA != c.PackageIdentity || b.Closure.Identity != c.ContextIdentity {
		return nil, fmt.Errorf("historical completion/closure binding mismatch")
	}
	if err := validation.validateHistoricalIntegratedSurface(root, locale, c.Evidence[1], c.Evidence[0]); err != nil {
		return nil, err
	}
	b.Surface = c.Evidence[1]
	return b, nil
}

// Rebuild from frozen model inputs, original outputs and old glossary. No
// current eligibility is required, and every localized file is hash checked.
func (validation *closureValidation) computeHistoricalStructuredClosure(root string, ref Reference, visiting map[string]bool) (*structuredBaseline, error) {
	if visiting[ref.Path] {
		return nil, fmt.Errorf("cyclic structured baseline")
	}
	visiting[ref.Path] = true
	defer delete(visiting, ref.Path)
	var c PackageClosure
	if err := validation.readReference(root, ref, &c); err != nil {
		return nil, err
	}
	copy := c
	copy.Identity = ""
	if c.Schema != ClosureSchema || c.Identity != identity(copy) || ref.Path != workflowPath(c.Locale, c.Package, "closure", c.Identity) {
		return nil, fmt.Errorf("historical closure identity mismatch")
	}
	b := &structuredBaseline{Closure: c, ClosureRef: ref, Files: map[string][]byte{}, Owners: map[string]Reference{}, Generations: map[string]*StructuredGeneration{}, Inputs: map[string]map[string][]byte{}}
	for fi, page := range closurePageFinalizations(&c) {
		if !strings.HasPrefix(page.Path, "data/quality-check-snapshots/") {
			return nil, fmt.Errorf("historical structured baseline requires shared Page integrity")
		}
		var f i18n.QualityCheckFinalization
		if err := validation.readReference(root, page, &f); err != nil {
			return nil, err
		}
		if f.Locale != c.Locale || page.Path != "data/quality-check-snapshots/"+c.Locale+"/"+f.SnapshotID+"/finalization.json" {
			return nil, fmt.Errorf("historical Page finalization scope/path mismatch")
		}
		cat, err := validation.inventoryCatalog()
		if err != nil {
			return nil, err
		}
		pkg, authority := c.Package, c.PackageSHA
		if len(c.PageComposition) > 0 {
			if c.PageFinalization != nil {
				return nil, fmt.Errorf("mixed historical finalization modes")
			}
			pkg, authority = c.PageComposition[fi].Package, c.PageComposition[fi].PackageAuthority
		}
		cat, err = i18n.HistoricalTranslationPackageCatalog(cat, pkg, authority)
		if err != nil {
			return nil, err
		}
		h, err := i18n.ReadHistoricalTranslationIntegrity(root, cat, c.Locale, f.SnapshotID)
		if err != nil {
			return nil, err
		}
		for _, ref := range h.References {
			if err := validation.bindHistoricalTranslationReference(cat, h, ref); err != nil {
				return nil, err
			}
		}
		for _, u := range h.Finalization.Units {
			v, err := validation.readRegular(root, u.Snapshot.CandidatePath)
			if err != nil || digest(v) != u.Snapshot.CandidateSHA256 {
				return nil, fmt.Errorf("historical Page target drift")
			}
			b.Files[u.Snapshot.SourcePath] = v
		}
	}
	// A successor transports its current deltas, not a flattened ancestry.
	// Inherit only structured owners; Page evidence is independently verified
	// above against this closure's own finalization.
	if c.StructuredBaseline != nil {
		parent, err := validation.historicalStructuredClosure(root, *c.StructuredBaseline, visiting)
		if err != nil {
			return nil, err
		}
		if parent.Closure.Locale != c.Locale || parent.Closure.Package != c.Package {
			return nil, fmt.Errorf("historical structured baseline scope mismatch")
		}
		for path, owner := range parent.Owners {
			b.Files[path], b.Owners[path] = parent.Files[path], owner
		}
		for path, generation := range parent.Generations {
			b.Generations[path], b.Inputs[path] = generation, parent.Inputs[path]
		}
	}
	seen := map[string]bool{}
	for _, gen := range c.Structured {
		if seen[gen.Path] {
			return nil, fmt.Errorf("duplicate historical generation")
		}
		seen[gen.Path] = true
		r, files, fs, err := validation.historicalStructuredGeneration(root, gen)
		if err != nil {
			return nil, err
		}
		if r.Plan.Locale != c.Locale {
			return nil, fmt.Errorf("historical generation locale mismatch")
		}
		b.Generations[gen.Path], b.Inputs[gen.Path] = r, fs
		for _, a := range r.Plan.Assets {
			if a.Package != c.Package || a.Document == nil {
				continue
			}
			v, ok := files[a.TargetPath]
			if !ok {
				continue
			}
			p := a.Document.Path
			if old, exists := b.Owners[p]; exists {
				if old == gen {
					continue // Compatible with older, flattened closure inventories.
				}
				if a.Assessment != nil && a.Assessment.Classification == "affected" && a.Assessment.Generation == old {
					parent, err := validation.historicalStructuredClosure(root, a.Assessment.Closure, visiting)
					if err != nil || parent.Closure.Package != c.Package || parent.Owners[p] != old || digest(parent.Files[p]) != a.Assessment.TargetSHA {
						return nil, fmt.Errorf("historical delta baseline mismatch")
					}
				} else if structuredSupersedes(root, r, old, p) {
				} else if structuredSupersedes(root, b.Generations[old.Path], gen, p) {
					continue
				} else {
					return nil, fmt.Errorf("ambiguous historical structured replacement")
				}
			}
			b.Files[p], b.Owners[p] = v, gen
		}
	}
	if err := validation.applyHistoricalShellRetirement(root, b, visiting); err != nil {
		return nil, err
	}
	if len(b.Files) != len(c.Files) {
		expectedOnly, actualOnly := []string{}, []string{}
		expected := map[string]bool{}
		for _, f := range c.Files {
			expected[f.Path] = true
			if _, ok := b.Files[f.Path]; !ok {
				expectedOnly = append(expectedOnly, f.Path)
			}
		}
		for path := range b.Files {
			if !expected[path] {
				actualOnly = append(actualOnly, path)
			}
		}
		sort.Strings(expectedOnly)
		sort.Strings(actualOnly)
		return nil, fmt.Errorf("historical closure inventory mismatch: closure=%s package=%s expected=%d actual=%d expected_only=%v actual_only=%v", ref.Path, c.Package, len(c.Files), len(b.Files), expectedOnly, actualOnly)
	}
	for i, f := range c.Files {
		if i > 0 && c.Files[i-1].Path >= f.Path {
			return nil, fmt.Errorf("non-exact historical inventory")
		}
		v, ok := b.Files[f.Path]
		if !ok || digest(v) != f.SHA256 {
			return nil, fmt.Errorf("historical closure target mismatch: %s", f.Path)
		}
	}
	return b, nil
}

func (validation *closureValidation) computeHistoricalStructuredGeneration(root string, ref Reference) (*StructuredGeneration, map[string][]byte, map[string][]byte, error) {
	r, m, fs, err := readStructuredGeneration(root, ref)
	if err != nil {
		return nil, nil, nil, err
	}
	provenanceRefs, err := validateStructuredProvenance(root, r)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, ref := range provenanceRefs {
		if err := validation.bind(ref); err != nil {
			return nil, nil, nil, err
		}
	}
	if err := validation.bind(r.Bundle); err != nil {
		return nil, nil, nil, err
	}
	if r.CarryForward != nil {
		for _, dep := range []Reference{r.CarryForward.CurrentBundle, r.CarryForward.OriginalOutput} {
			if err := validation.bind(dep); err != nil {
				return nil, nil, nil, err
			}
		}
	}
	if err := validation.review(r.Plan.Locale, r.Plan.Review); err != nil {
		return nil, nil, nil, err
	}
	return validation.validateFrozenStructuredGeneration(root, r, m, fs)
}

// Shared frozen validator also checks prepared recovery before it publishes
// archives. A preflight therefore proves the same integrity later consumers use.
func (validation *closureValidation) validateFrozenStructuredGeneration(root string, r *StructuredGeneration, m *StructuredBundle, fs map[string][]byte) (*StructuredGeneration, map[string][]byte, map[string][]byte, error) {
	r, files, fs, err := validation.validateFrozenStructuredContent(root, r, m, fs)
	if err != nil {
		return nil, nil, nil, err
	}
	if m.RevisionFinding == nil {
		if err := validation.verifyFrozenStructuredDelta(root, r, fs); err != nil {
			return nil, nil, nil, err
		}
	}
	return r, files, fs, nil
}

// Frozen transport, registry, complete output restoration and provenance prove
// source consumption. They do not grant language eligibility: that additionally
// requires the historical assessment/closure checks in the wrapper above.
func (validation *closureValidation) validateFrozenStructuredContent(root string, r *StructuredGeneration, m *StructuredBundle, fs map[string][]byte) (*StructuredGeneration, map[string][]byte, map[string][]byte, error) {
	for _, ref := range m.ValidationInputs {
		if err := validation.bind(ref); err != nil {
			return nil, nil, nil, err
		}
	}
	old := fs["inputs/locales/"+r.Plan.Locale+"/glossary.yaml"]
	if digest(old) != r.Plan.GlossarySHA {
		return nil, nil, nil, fmt.Errorf("historical glossary binding mismatch")
	}
	glossary, err := i18n.ParseGlossaryBytes(r.Plan.Locale, old)
	if err != nil {
		return nil, nil, nil, err
	}
	if r.Plan.Review.Path == "" || digest(fs["inputs/"+r.Plan.Review.Path]) != r.Plan.Review.SHA256 {
		return nil, nil, nil, fmt.Errorf("historical Unified Review binding mismatch")
	}
	if err := i18n.CheckHistoricalUnifiedGlossaryReview(root, r.Plan.Locale, r.Plan.GlossarySHA, r.Plan.CorpusSHA, i18n.GlossaryArchiveReference{Path: r.Plan.Review.Path, SHA256: r.Plan.Review.SHA256}); err != nil {
		return nil, nil, nil, err
	}
	var corpus i18n.GlossarySourceCorpus
	if err := StrictJSON(fs["inputs/"+i18n.GlossaryCorpusPath], &corpus); err != nil {
		return nil, nil, nil, err
	}
	if corpus.Identity != r.Plan.CorpusSHA || corpus.Identity != i18n.CorpusIdentity(corpus) {
		return nil, nil, nil, fmt.Errorf("historical corpus identity mismatch")
	}
	if err := i18n.ValidateGlossaryCorpus(&corpus); err != nil {
		return nil, nil, nil, err
	}
	groups, err := structuredContributorGroups(corpus.Contributors)
	if err != nil {
		return nil, nil, nil, err
	}
	wanted := []string{}
	assets := map[string]StructuredAsset{}
	ids := map[string]bool{}
	if len(r.Plan.Assets) != len(groups) {
		return nil, nil, nil, fmt.Errorf("historical asset registry non-exact inventory")
	}
	for _, a := range r.Plan.Assets {
		cs := groups[a.ID]
		if a.Document != nil && hasSharedDocumentAuthority(*a.Document) && a.State != "carry" && a.State != "context-only" {
			if err := validation.verifyArchivedSharedRepresentation(root, a, fs["source/"+a.Document.Path], fs); err != nil {
				return nil, nil, nil, err
			}
		}
		paths := []string{}
		for _, c := range cs {
			paths = append(paths, c.Path)
		}
		if len(cs) == 0 || ids[a.ID] || a.Package != cs[0].Package || a.SourceSHA != identity(cs) || assets[a.TargetPath].ID != "" || !reflect.DeepEqual(a.SourcePaths, uniqueStrings(paths)) {
			return nil, nil, nil, fmt.Errorf("historical structured source/input mismatch")
		}
		ids[a.ID] = true
		assets[a.TargetPath] = a
		if a.State != "carry" && a.State != "context-only" {
			wanted = append(wanted, a.TargetPath)
		}
	}
	sort.Strings(wanted)
	if !reflect.DeepEqual(wanted, r.Plan.Expected) || len(r.Output.Outputs) != len(wanted) {
		return nil, nil, nil, fmt.Errorf("historical structured exact-set mismatch")
	}
	files := map[string][]byte{}
	for i, t := range r.Output.Outputs {
		if t.Path != wanted[i] {
			return nil, nil, nil, fmt.Errorf("historical output inventory mismatch")
		}
		a := assets[t.Path]
		if a.Document == nil {
			if len(t.Slots) != 0 || textBytes([]byte(t.Data)) != nil {
				return nil, nil, nil, fmt.Errorf("historical complete-file output invalid")
			}
			switch a.ID {
			case "tour-ui":
				if err := ui.ValidateLocalizedBytes(r.Plan.Locale, fs["inputs/internal/tour/ui/en.json"], []byte(t.Data)); err != nil {
					return nil, nil, nil, err
				}
			case "article-metadata":
				cat := &i18n.Catalog{}
				for _, c := range groups[a.ID] {
					cat.Pages = append(cat.Pages, i18n.Page{Article: strings.TrimPrefix(c.Path, "_content/tour/")})
				}
				if _, err := i18n.ValidateArticleMetadataBytes(r.Plan.Locale, cat, []byte(t.Data)); err != nil {
					return nil, nil, nil, err
				}
			default:
				return nil, nil, nil, fmt.Errorf("unknown historical structured validator")
			}
			files[t.Path] = []byte(t.Data)
			continue
		}
		raw := fs["source/"+a.Document.Path]
		if err := verifyFrozenStructuredRepresentation(*a.Document, raw); err != nil {
			return nil, nil, nil, fmt.Errorf("historical parser/source mismatch: %s: %w", a.ID, err)
		}
		values, err := structuredTargetValues(a, t, glossary)
		if err != nil {
			return nil, nil, nil, err
		}
		v, err := reconstructFrozenStructuredRepresentation(*a.Document, raw, values)
		if err != nil {
			return nil, nil, nil, err
		}
		files[t.Path] = v
	}
	if r.CarryForward != nil && !reflect.DeepEqual(r.CarryForward.ValidatedFiles, structuredValidatedFiles(files)) {
		return nil, nil, nil, fmt.Errorf("structured carry-forward reconstructed validation identity changed")
	}
	var originalReview i18n.UnifiedGlossaryReview
	if err := validation.readReference(root, r.Plan.Review, &originalReview); err != nil {
		return nil, nil, nil, err
	}
	if originalReview.GenerationSession != r.Provenance.Session {
		return nil, nil, nil, fmt.Errorf("historical Generation role/provenance mismatch")
	}
	if err := i18n.CheckUnifiedLanguageRole(root, r.Plan.Locale, r.Provenance.Session, "generation"); err != nil {
		return nil, nil, nil, err
	}
	return r, files, fs, nil
}

// Read only the requested structured documents through the existing source
// authority. Unrelated Page parsers do not belong to a structured validator.
func currentStructuredDocuments(root string, g *Global, assets []StructuredAsset) (map[string]Document, map[string][]byte, error) {
	docs := map[string]Document{}
	raw := map[string][]byte{}
	var frozen fs.FS
	for _, a := range assets {
		if a.Document == nil {
			continue
		}
		if _, ok := docs[a.Document.Path]; ok {
			continue
		}
		var source *Source
		for i := range g.Sources {
			if g.Sources[i].Path == a.Document.Path && g.Sources[i].Package == a.Package {
				source = &g.Sources[i]
			}
		}
		if source == nil {
			return nil, nil, fmt.Errorf("structured current source unregistered")
		}
		var b []byte
		var err error
		if source.Body != nil || a.Package == "site-v2-shell" {
			b, err = readRegularReference(root, currentSourceReference(*source))
		} else {
			if frozen == nil {
				snapshot, e := ReadSourceSnapshot(root)
				if e != nil {
					return nil, nil, e
				}
				frozen, err = snapshotFS(snapshot)
				if err != nil {
					return nil, nil, err
				}
			}
			b, err = fs.ReadFile(frozen, source.Path)
		}
		if err != nil {
			return nil, nil, err
		}
		var d *Document
		if source.Body != nil {
			current, _, e := CurrentPackageDocuments(root, g, a.Package)
			if e != nil {
				return nil, nil, e
			}
			for _, doc := range current {
				if doc.Path == a.Document.Path && reflect.DeepEqual(doc, *a.Document) {
					value := doc
					d = &value
				}
			}
			if d == nil {
				return nil, nil, fmt.Errorf("source successor canonical Document mismatch")
			}
			err = verifyFrozenStructuredRepresentation(*d, b)
		} else {
			d, err = parseAuthorityStructuredRepresentation(*a.Document, b)
		}
		if err != nil {
			return nil, nil, err
		}
		docs[source.Path], raw[source.Path] = *d, b
	}
	return docs, raw, nil
}

func readStructuredGeneration(root string, ref Reference) (*StructuredGeneration, *StructuredBundle, map[string][]byte, error) {
	var r StructuredGeneration
	if err := readReference(root, ref, &r); err != nil {
		return nil, nil, nil, err
	}
	copy := r
	copy.Identity = ""
	pc := r.Plan
	pc.Identity = ""
	if r.Schema != StructuredSchema || r.Plan.Schema != StructuredSchema || r.Output.Schema != StructuredSchema || r.Identity != identity(copy) || r.Plan.Identity != identity(pc) || ref.Path != "data/locale-language/"+r.Plan.Locale+"/structured-generation/"+r.Identity+".json" {
		return nil, nil, nil, fmt.Errorf("invalid structured evidence identity")
	}
	if _, err := validateStructuredProvenance(root, &r); err != nil {
		return nil, nil, nil, err
	}
	b, err := readRegularReference(root, r.Bundle)
	if err != nil {
		return nil, nil, nil, err
	}
	if r.Bundle.Path != "data/locale-language/"+r.Plan.Locale+"/structured-bundles/"+digest(b)+".zip" {
		return nil, nil, nil, fmt.Errorf("structured bundle path mismatch")
	}
	m, fs, err := readStructuredBundle(b)
	if err != nil {
		return nil, nil, nil, err
	}
	if !reflect.DeepEqual(m.Plan, r.Plan) || r.Output.BundleSHA != m.Identity {
		return nil, nil, nil, fmt.Errorf("structured provenance/input identity mismatch")
	}
	if r.CarryForward != nil {
		current, currentFiles, err := readStructuredCarryForward(root, &r, *m, fs)
		return &r, current, currentFiles, err
	}
	return &r, m, fs, nil
}

func readRegularReference(root string, ref Reference) ([]byte, error) {
	b, err := readRegular(root, ref.Path)
	if err != nil || digest(b) != ref.SHA256 {
		return nil, fmt.Errorf("structured evidence hash drift: %s", ref.Path)
	}
	return b, nil
}

func reconstructStructuredTarget(a StructuredAsset, raw []byte, t StructuredTarget, glossary *i18n.Glossary) ([]byte, error) {
	values, err := structuredTargetValues(a, t, glossary)
	if err != nil {
		return nil, err
	}
	if a.Assessment != nil && a.Assessment.SourceSuccessor != nil {
		return reconstructFrozenStructuredRepresentation(*a.Document, raw, values)
	}
	return reconstructAuthorityStructuredRepresentation(*a.Document, raw, values)
}

func structuredTargetValues(a StructuredAsset, t StructuredTarget, glossary *i18n.Glossary) (map[string]string, error) {
	if a.Document == nil || t.Data != "" || len(t.Slots)+len(a.CarriedSlots) != len(a.Document.Units) {
		return nil, fmt.Errorf("structured parser slot exact-set required")
	}
	values := map[string]string{}
	if len(a.CarriedSlots) > 0 && (a.Assessment == nil || a.Assessment.SourceSuccessor == nil) {
		return nil, fmt.Errorf("carried slots require exact source successor authority")
	}
	for _, v := range append(append([]Target{}, a.CarriedSlots...), t.Slots...) {
		if _, ok := values[v.ID]; ok {
			return nil, fmt.Errorf("duplicate structured slot")
		}
		values[v.ID] = v.Text
	}
	for _, u := range a.Document.Units {
		if _, ok := values[u.ID]; !ok {
			return nil, fmt.Errorf("missing structured slot")
		}
		if err := i18n.ValidateParsedGlossary(u.Source, values[u.ID], glossary); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func (validation *closureValidation) assessStructuredAsset(root string, p StructuredPlan, a StructuredAsset, baseline *structuredBaseline) (*StructuredAssessment, error) {
	x := &StructuredAssessment{Classification: "unknown", Closure: baseline.ClosureRef, Surface: baseline.Surface, GlossarySHA: p.GlossarySHA, Review: p.Review, SourceSHA: a.SourceSHA}
	finish := func(state, reason string) (*StructuredAssessment, error) {
		x.Classification, x.Reason = state, reason
		x.Identity = identity(*x)
		return x, nil
	}
	if a.Document == nil || (a.Document.Contract != UnitContract && a.Document.Contract != ShellExtractionContract) || a.Package != baseline.Closure.Package || p.Locale != baseline.Closure.Locale {
		return finish("unknown", "baseline scope/parser mismatch")
	}
	gen, ok := baseline.Owners[a.Document.Path]
	if !ok {
		return finish("unknown", "baseline asset missing")
	}
	x.Generation, x.TargetSHA = gen, digest(baseline.Files[a.Document.Path])
	r := baseline.Generations[gen.Path]
	x.OldGlossarySHA = r.Plan.GlossarySHA
	var oldAsset *StructuredAsset
	var target *StructuredTarget
	for i := range r.Plan.Assets {
		if r.Plan.Assets[i].ID == a.ID {
			oldAsset = &r.Plan.Assets[i]
		}
	}
	for i := range r.Output.Outputs {
		if r.Output.Outputs[i].Path == a.TargetPath {
			target = &r.Output.Outputs[i]
		}
	}
	if oldAsset == nil || target == nil || oldAsset.Package != a.Package {
		return finish("unknown", "historical asset/input missing")
	}
	oldCorpus := baseline.Inputs[gen.Path]["inputs/"+i18n.GlossaryCorpusPath]
	currentCorpus, err := validation.readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		return nil, err
	}
	if (oldAsset.SourceSHA != a.SourceSHA && !sameStructuredContributorInput(oldCorpus, currentCorpus, a.ID, a.SourceSHA)) || !reflect.DeepEqual(oldAsset.Document, a.Document) {
		if hasSharedDocumentAuthority(*a.Document) {
			matched, err := validation.sharedRepresentationAssessment(root, a, *oldAsset, baseline.Inputs[gen.Path]["source/"+a.Document.Path], true)
			if err != nil {
				return nil, err
			}
			if matched {
				ref, err := sharedDocumentAuthority(*a.Document)
				if err != nil {
					return nil, err
				}
				x.RepresentationSuccessor = ref
				return finish("affected", "explicit shared structured representation successor requires fresh language generation")
			}
		}
		if matched, err := validation.sourceExpansionAssessment(root, p, a, baseline, x); err != nil {
			return nil, err
		} else if matched {
			return x, nil
		}
		a.Assessment = x
		reopen, err := validation.findInputReopen(root, p, a, baseline)
		if err != nil {
			return nil, err
		}
		if reopen != nil {
			applyInputReopenAssessment(x, *reopen)
			return x, nil
		}
		return finish("stale", "current exact source/input drift")
	}
	current, err := validation.readRegular(root, "locales/"+p.Locale+"/glossary.yaml")
	if err != nil {
		return nil, err
	}
	if digest(current) != p.GlossarySHA {
		return finish("unknown", "current glossary binding mismatch")
	}
	review, err := validation.unifiedReview(p.Locale)
	if err != nil {
		return nil, err
	}
	if p.Review != (Reference{review.Path, review.SHA256}) {
		return finish("unknown", "current full Unified Review mismatch")
	}
	old := baseline.Inputs[gen.Path]["inputs/locales/"+p.Locale+"/glossary.yaml"]
	delta, err := i18n.ParsedGlossaryDelta(p.Locale, old, current)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, v := range target.Slots {
		values[v.ID] = v.Text
	}
	affected := false
	for _, u := range a.Document.Units {
		compatible, _ := i18n.ParsedGlossaryImpact(u.Source, values[u.ID], delta)
		if !compatible {
			affected = true
		}
	}
	glossary, err := i18n.ParseGlossaryBytes(p.Locale, current)
	if err != nil {
		return nil, err
	}
	raw := baseline.Inputs[gen.Path]["source/"+a.Document.Path]
	// Reconstruction is always checked, even if current terminology requires a
	// replacement. A glossary failure is authorized only by actual impact.
	v, err := validation.reconstructAssessedStructuredTarget(root, a, raw, *target, glossary)
	if err != nil && !affected {
		return finish("unknown", "current validation failed: "+err.Error())
	}
	if err == nil && !bytes.Equal(v, baseline.Files[a.Document.Path]) {
		return finish("unknown", "current reconstruction mismatch")
	}
	if affected {
		return finish("affected", "current parsed glossary impact")
	}
	return finish("compatible", "exact source/accepted target + current validation")
}

func (validation *closureValidation) verifyFrozenStructuredDelta(root string, r *StructuredGeneration, fs map[string][]byte) error {
	baselines := map[string]*structuredBaseline{}
	for _, a := range r.Plan.Assets {
		if a.Assessment == nil {
			continue
		}
		x := a.Assessment
		copy := *x
		copy.Identity = ""
		if x.Identity != identity(copy) || x.GlossarySHA != r.Plan.GlossarySHA || x.Review != r.Plan.Review || x.SourceSHA != a.SourceSHA {
			return fmt.Errorf("historical delta assessment identity/binding mismatch")
		}
		b := baselines[x.Closure.Path]
		if b == nil {
			var err error
			b, err = validation.historicalStructuredClosure(root, x.Closure, map[string]bool{})
			if err != nil {
				return err
			}
			if err := validation.validateHistoricalIntegratedSurface(root, r.Plan.Locale, x.Surface, x.Closure); err != nil {
				return err
			}
			baselines[x.Closure.Path] = b
		}
		if a.Document == nil || b.Closure.Package != a.Package || b.Closure.Locale != r.Plan.Locale || b.Owners[a.Document.Path] != x.Generation || digest(b.Files[a.Document.Path]) != x.TargetSHA {
			return fmt.Errorf("historical delta baseline/target binding mismatch")
		}
		old := b.Generations[x.Generation.Path]
		if old == nil || old.Plan.GlossarySHA != x.OldGlossarySHA {
			return fmt.Errorf("historical delta old glossary mismatch")
		}
		if x.InputReopen != nil {
			b.Surface = x.Surface
			if err := validation.verifyFrozenInputReopen(root, r.Plan, a, b, fs); err != nil {
				return err
			}
			continue
		}
		if x.RepresentationSuccessor != nil {
			matched := false
			for _, oldAsset := range old.Plan.Assets {
				if oldAsset.ID != a.ID {
					continue
				}
				var err error
				matched, err = validation.sharedRepresentationAssessment(root, a, oldAsset, b.Inputs[x.Generation.Path]["source/"+a.Document.Path], false)
				if err != nil {
					return err
				}
			}
			authority, err := sharedDocumentAuthority(*a.Document)
			if err != nil {
				return err
			}
			if !matched || authority == nil || *x.RepresentationSuccessor != *authority || x.Classification != "affected" || a.State != "generation-required" || len(a.CarriedSlots) != 0 {
				return fmt.Errorf("historical shared representation assessment mismatch")
			}
			continue
		}
		if x.SourceSuccessor != nil {
			var source StructuredSourceSuccessor
			if err := readReference(root, *x.SourceSuccessor, &source); err != nil {
				return err
			}
			copy := source
			copy.Identity = ""
			if source.Schema != StructuredSourceSuccessorSchema || source.Identity != identity(copy) || x.SourceSuccessor.Path != StructuredSourceSuccessorPath(&source) || !reflect.DeepEqual(a.Document, &source.Current) || x.Classification != "affected" || a.State != "generation-required" {
				return fmt.Errorf("historical source successor identity/scope mismatch")
			}
			if source.Replacement == nil {
				if _, _, err := sourceExpansionSlots(source.Old, source.Current); err != nil {
					return err
				}
			} else {
				if _, err := CheckStructuredSourceSuccessor(root, *x.SourceSuccessor); err != nil {
					return err
				}
			}
			currentRaw := fs["source/"+source.Current.Path]
			if digest(currentRaw) != source.Body.SHA256 {
				return fmt.Errorf("historical source successor body mismatch")
			}
			d, err := sourceDeltaFromBaseline(root, r.Plan.Locale, source, b)
			if err != nil || !reflect.DeepEqual(d.Carry, a.CarriedSlots) {
				return fmt.Errorf("historical source successor carry scope mismatch")
			}
			continue
		}
		matched := false
		for _, asset := range old.Plan.Assets {
			if asset.ID == a.ID && asset.Package == a.Package && (asset.SourceSHA == a.SourceSHA || sameStructuredContributorInput(b.Inputs[x.Generation.Path]["inputs/"+i18n.GlossaryCorpusPath], fs["inputs/"+i18n.GlossaryCorpusPath], a.ID, a.SourceSHA)) && reflect.DeepEqual(asset.Document, a.Document) {
				matched = true
			}
		}
		if !matched {
			return fmt.Errorf("historical delta source/input mismatch")
		}
		delta, err := i18n.ParsedGlossaryDelta(r.Plan.Locale, b.Inputs[x.Generation.Path]["inputs/locales/"+r.Plan.Locale+"/glossary.yaml"], fs["inputs/locales/"+r.Plan.Locale+"/glossary.yaml"])
		if err != nil {
			return err
		}
		affected := false
		for _, target := range old.Output.Outputs {
			if target.Path != a.TargetPath {
				continue
			}
			values := map[string]string{}
			for _, v := range target.Slots {
				values[v.ID] = v.Text
			}
			for _, u := range a.Document.Units {
				ok, _ := i18n.ParsedGlossaryImpact(u.Source, values[u.ID], delta)
				if !ok {
					affected = true
				}
			}
		}
		if affected {
			if x.Classification != "affected" || a.State != "generation-required" {
				return fmt.Errorf("historical affected false carry")
			}
		} else if x.Classification != "compatible" || a.State != "carry" {
			return fmt.Errorf("historical compatible arbitrary replacement")
		}
	}
	return nil
}

func (validation *closureValidation) validateStructuredDelta(root string, r *StructuredGeneration, m *StructuredBundle) error {
	if m.RevisionFinding != nil {
		return nil
	}
	baselines := map[string]*structuredBaseline{}
	expected := []string{}
	for _, a := range r.Plan.Assets {
		if a.Assessment == nil {
			if a.State != "carry" && a.State != "context-only" {
				expected = append(expected, a.TargetPath)
			}
			continue
		}
		x := a.Assessment
		b := baselines[x.Closure.Path]
		if b == nil {
			var err error
			b, err = validation.historicalStructuredClosure(root, x.Closure, map[string]bool{})
			if err != nil {
				return err
			}
			if err := validation.validateHistoricalIntegratedSurface(root, r.Plan.Locale, x.Surface, x.Closure); err != nil {
				return err
			}
			b.Surface = x.Surface
			baselines[x.Closure.Path] = b
		}
		want, err := validation.assessStructuredAsset(root, r.Plan, a, b)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(want, x) {
			return fmt.Errorf("structured delta baseline/glossary/input assessment mismatch")
		}
		carry, err := sourceDeltaCarry(root, r.Plan, a, b)
		if err != nil || !reflect.DeepEqual(carry, a.CarriedSlots) {
			return fmt.Errorf("structured source carry authority mismatch")
		}
		switch x.Classification {
		case "compatible":
			if a.State != "carry" {
				return fmt.Errorf("compatible arbitrary replacement")
			}
		case "affected":
			if a.State != "generation-required" {
				return fmt.Errorf("affected false carry")
			}
			expected = append(expected, a.TargetPath)
		default:
			return fmt.Errorf("structured delta %s asset", x.Classification)
		}
	}
	sort.Strings(expected)
	if !reflect.DeepEqual(expected, r.Plan.Expected) {
		return fmt.Errorf("structured delta non-exact output authority")
	}
	return nil
}

// Shared primitive for all structured packages: immutable accepted inventory
// plus a fully authorized current exact affected set. Reference order is free.
func (validation *closureValidation) mergeStructuredBaselineDelta(root, locale, pkg string, refs []Reference, frozenBaseline ...*Reference) (map[string][]byte, *Reference, bool, error) {
	var baseline *structuredBaseline
	var completion *Completion
	glob, err := validation.loadCurrent(root)
	if err != nil {
		return nil, nil, false, err
	}
	l, err := readLocaleAuthority(root, glob, locale)
	if err != nil {
		return nil, nil, false, err
	}
	// A saved closure fixes its own predecessor. Locale coverage may already
	// contain this closure or a later one; it cannot choose that epoch for us.
	if len(frozenBaseline) > 0 && frozenBaseline[0] != nil {
		baseline, err = validation.historicalStructuredClosure(root, *frozenBaseline[0], map[string]bool{})
		if err != nil {
			return nil, nil, false, err
		}
		if baseline.Closure.Locale != locale || baseline.Closure.Package != pkg {
			return nil, nil, false, fmt.Errorf("frozen structured baseline scope mismatch")
		}
	}
	for i := range l.Packages {
		if l.Packages[i].Package == pkg && l.Packages[i].EvidenceKind == PackageCompletionKind {
			completion = &l.Packages[i]
		}
	}
	seen := map[string]bool{}
	generations := map[string]*StructuredGeneration{}
	outputs := map[string]map[string][]byte{}
	for _, ref := range refs {
		if seen[ref.Path] {
			return nil, nil, false, fmt.Errorf("duplicate conflicting structured delta")
		}
		seen[ref.Path] = true
		r, files, _, err := validation.historicalStructuredGeneration(root, ref)
		if err != nil {
			return nil, nil, false, err
		}
		if r.Plan.Locale != locale {
			return nil, nil, false, fmt.Errorf("structured locale mismatch")
		}
		generations[ref.Path], outputs[ref.Path] = r, files
		for _, a := range r.Plan.Assets {
			if a.Package != pkg || a.Assessment == nil || a.State != "generation-required" {
				continue
			}
			if baseline != nil && baseline.ClosureRef != a.Assessment.Closure {
				return nil, nil, false, fmt.Errorf("conflicting structured baseline closures")
			}
			baseline, err = validation.historicalStructuredClosure(root, a.Assessment.Closure, map[string]bool{})
			if err != nil {
				return nil, nil, false, err
			}
			baseline.Surface = a.Assessment.Surface
			if baseline.Closure.Locale != locale || baseline.Closure.Package != pkg {
				return nil, nil, false, fmt.Errorf("cross-package baseline misuse")
			}
			if err := validation.validateHistoricalIntegratedSurface(root, locale, baseline.Surface, baseline.ClosureRef); err != nil {
				return nil, nil, false, err
			}
		}
	}
	if baseline == nil && completion != nil {
		baseline, err = validation.historicalStructuredCompletion(root, locale, *completion)
		if err != nil {
			return nil, nil, false, err
		}
	}
	if baseline == nil {
		return nil, nil, false, nil
	}
	if completion != nil && baseline.ClosureRef != completion.Evidence[0] {
		// An already activated successor may still be checked against its bound
		// predecessor; unrelated baselines are never accepted.
		var active PackageClosure
		if err := validation.readReference(root, completion.Evidence[0], &active); err != nil {
			return nil, nil, false, err
		}
		if active.StructuredBaseline == nil || *active.StructuredBaseline != baseline.ClosureRef {
			return nil, nil, false, fmt.Errorf("wrong baseline closure")
		}
	}
	corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return nil, nil, false, err
	}
	groups, err := structuredContributorGroups(corpus.Contributors)
	if err != nil {
		return nil, nil, false, err
	}
	gs, err := validation.readRegular(root, "locales/"+locale+"/glossary.yaml")
	if err != nil {
		return nil, nil, false, err
	}
	review, err := validation.unifiedReview(locale)
	if err != nil {
		return nil, nil, false, err
	}
	p := StructuredPlan{Locale: locale, CorpusSHA: corpus.Identity, GlossarySHA: digest(gs), Review: Reference{review.Path, review.SHA256}}
	docs, _, err := CurrentPackageDocuments(root, glob, pkg)
	if err != nil {
		return nil, nil, false, err
	}
	byPath := map[string]Document{}
	for _, d := range docs {
		byPath[d.Path] = d
	}
	out := map[string][]byte{}
	changed := false
	usedDelta := map[string]bool{}
	baselineRefs := map[string]bool{}
	inherited := false
	for _, ref := range baseline.Closure.Structured {
		baselineRefs[ref.Path] = true
		if !seen[ref.Path] {
			inherited = true
		}
	}
	for path, owner := range baseline.Owners {
		if _, current := byPath[path]; !current {
			retired, err := authorizedShellRetirement(root, glob, baseline.Closure.PackageSHA, path)
			if err != nil {
				return nil, nil, false, err
			}
			if retired {
				continue
			}
			return nil, nil, false, fmt.Errorf("unreconciled baseline asset retirement")
		}
		if !seen[owner.Path] {
			inherited = true
		}
		old := baseline.Generations[owner.Path]
		var a StructuredAsset
		for _, asset := range old.Plan.Assets {
			if asset.Document != nil && asset.Document.Path == path {
				a = asset
			}
		}
		cs := groups[a.ID]
		if len(cs) == 0 || cs[0].Package != pkg {
			return nil, nil, false, fmt.Errorf("structured registry inventory mismatch")
		}
		d, ok := byPath[path]
		if !ok {
			return nil, nil, false, fmt.Errorf("structured source missing")
		}
		a.Document, a.SourceSHA = &d, identity(cs)
		x, err := validation.assessStructuredAsset(root, p, a, baseline)
		if err != nil {
			return nil, nil, false, err
		}
		replacements := []Reference{}
		for _, ref := range refs {
			if baselineRefs[ref.Path] {
				continue
			}
			if _, ok := outputs[ref.Path][a.TargetPath]; ok {
				replacements = append(replacements, ref)
			}
		}
		if x.Classification == "compatible" {
			if len(replacements) == 1 && structuredSupersedes(root, generations[replacements[0].Path], owner, path) {
				_, current, err := validation.checkStructuredGeneration(root, replacements[0], pkg)
				if err != nil {
					return nil, nil, false, err
				}
				out[path] = current[a.TargetPath]
				usedDelta[replacements[0].Path], changed = true, true
				continue
			}
			if len(replacements) != 0 {
				return nil, nil, false, fmt.Errorf("compatible arbitrary replacement")
			}
			out[path] = baseline.Files[path]
			continue
		}
		if x.Classification != "affected" {
			return nil, nil, false, fmt.Errorf("structured %s: %s", x.Classification, path)
		}
		if len(replacements) != 1 {
			return nil, nil, false, fmt.Errorf("missing/duplicate affected delta: %s", path)
		}
		ref := replacements[0]
		r := generations[ref.Path]
		_, manifest, _, err := readStructuredGeneration(root, ref)
		if err != nil {
			return nil, nil, false, err
		}
		if manifest.RevisionFinding != nil {
			// A representation successor can require fresh language under an
			// already recorded exact Surface finding. It is not a glossary/source
			// delta: the independent finding must supersede this accepted owner.
			if x.RepresentationSuccessor == nil {
				return nil, nil, false, fmt.Errorf("glossary delta cannot use Surface finding authority")
			}
			if r.Plan.GlossarySHA != p.GlossarySHA || r.Plan.Review != p.Review {
				return nil, nil, false, fmt.Errorf("wrong delta glossary binding")
			}
			if !structuredSupersedes(root, r, owner, path) {
				return nil, nil, false, fmt.Errorf("unauthorized representation Surface replacement")
			}
			_, current, err := validation.checkStructuredGeneration(root, ref, pkg)
			if err != nil {
				return nil, nil, false, err
			}
			out[path] = current[a.TargetPath]
			usedDelta[ref.Path], changed = true, true
			continue
		}
		if r.Plan.GlossarySHA != p.GlossarySHA || r.Plan.Review != p.Review {
			return nil, nil, false, fmt.Errorf("wrong delta glossary binding")
		}
		bound := false
		for _, asset := range r.Plan.Assets {
			if asset.TargetPath == a.TargetPath && reflect.DeepEqual(asset.Assessment, x) {
				bound = true
			}
		}
		if !bound {
			return nil, nil, false, fmt.Errorf("unauthorized structured replacement")
		}
		_, current, err := validation.checkStructuredGeneration(root, ref, pkg)
		if err != nil {
			return nil, nil, false, err
		}
		out[path] = current[a.TargetPath]
		usedDelta[ref.Path] = true
		changed = true
	}
	for _, d := range docs {
		if d.UnitScope == PageContract {
			continue
		}
		if _, exists := baseline.Owners[d.Path]; exists {
			continue
		}
		var asset StructuredAsset
		for _, cs := range groups {
			if cs[0].Package == pkg && cs[0].Path == d.Path {
				asset = StructuredAsset{ID: cs[0].ID, Package: pkg, SourcePaths: []string{d.Path}, SourceSHA: identity(cs), TargetPath: "locales/" + locale + "/structured/" + d.Path, Document: &d}
			}
		}
		if asset.ID == "" {
			if d.Kind == "page" {
				continue
			} // Shared Page finalization owns non-structured Pages.
			return nil, nil, false, fmt.Errorf("new structured asset outside current registry")
		}
		candidates := []Reference{}
		for _, ref := range refs {
			if baselineRefs[ref.Path] {
				continue
			}
			if _, ok := outputs[ref.Path][asset.TargetPath]; ok {
				candidates = append(candidates, ref)
			}
		}
		if len(candidates) != 1 {
			return nil, nil, false, fmt.Errorf("new structured asset requires unique validated generation: %s", d.Path)
		}
		ref := candidates[0]
		_, files, err := validation.checkStructuredGeneration(root, ref, pkg)
		if err != nil {
			return nil, nil, false, err
		}
		out[d.Path] = files[asset.TargetPath]
		usedDelta[ref.Path], changed = true, true
	}
	for _, ref := range refs {
		if !baselineRefs[ref.Path] && !usedDelta[ref.Path] {
			return nil, nil, false, fmt.Errorf("unauthorized extra structured generation")
		}
		for _, a := range generations[ref.Path].Plan.Assets {
			if a.Package != pkg || a.Document == nil {
				continue
			}
			if _, ok := outputs[ref.Path][a.TargetPath]; ok {
				if _, exists := baseline.Owners[a.Document.Path]; !exists && !usedDelta[ref.Path] {
					return nil, nil, false, fmt.Errorf("unauthorized extra delta")
				}
			}
		}
	}
	if changed || inherited {
		return out, &baseline.ClosureRef, true, nil
	}
	return out, nil, true, nil
}

func (validation *closureValidation) structuredSurfaceCarrySet(root string, c *PackageClosure) (map[string]bool, []Reference, error) {
	carried := map[string]bool{}
	if c.StructuredBaseline == nil {
		return carried, nil, nil
	}
	b, err := validation.historicalStructuredClosure(root, *c.StructuredBaseline, map[string]bool{})
	if err != nil {
		return nil, nil, err
	}
	if b.Closure.Locale != c.Locale || b.Closure.Package != c.Package {
		return nil, nil, fmt.Errorf("Surface baseline scope mismatch")
	}
	changed := map[string]bool{}
	proof := []Reference{*c.StructuredBaseline}
	for _, ref := range c.Structured {
		r, _, _, err := validation.historicalStructuredGeneration(root, ref)
		if err != nil {
			return nil, nil, err
		}
		for _, a := range r.Plan.Assets {
			if a.Package == c.Package && a.Document != nil && a.Assessment != nil && a.State == "generation-required" {
				changed[a.Document.Path] = true
				if a.Assessment.Surface.Path != "" {
					proof = append(proof, a.Assessment.Surface)
				}
			}
		}
	}
	for _, f := range c.Files {
		v, ok := b.Files[f.Path]
		if ok && !changed[f.Path] && digest(v) == f.SHA256 {
			carried[f.Path] = true
		}
	}
	return carried, proof, nil
}
