package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

const StructuredSchema = "go-learning/locale-structured-assets/v1"

type StructuredAsset struct {
	CarriedSlots []Target              `json:"carried_slots,omitempty"`
	ID           string                `json:"id"`
	Package      string                `json:"package"`
	SourcePaths  []string              `json:"source_paths"`
	SourceSHA    string                `json:"source_sha256"`
	TargetPath   string                `json:"target_path"`
	State        string                `json:"state"`
	TargetSHA    string                `json:"target_sha256,omitempty"`
	Proof        []Reference           `json:"carry_evidence"`
	Document     *Document             `json:"document,omitempty"`
	Assessment   *StructuredAssessment `json:"current_assessment,omitempty"`
}
type StructuredPlan struct {
	Schema      string            `json:"schema"`
	Locale      string            `json:"locale"`
	CorpusSHA   string            `json:"corpus_identity_sha256"`
	GlossarySHA string            `json:"glossary_sha256"`
	Review      Reference         `json:"unified_glossary_review"`
	Assets      []StructuredAsset `json:"assets"`
	Expected    []string          `json:"expected_outputs"`
	Identity    string            `json:"identity_sha256"`
}

// Registry membership is shared with the terminology corpus, not another list.
func PlanStructuredAssets(root, locale string) (*StructuredPlan, error) {
	if !component(locale) {
		return nil, fmt.Errorf("invalid locale")
	}
	v := newClosureValidation(root)
	if err := v.currentInputs(locale); err != nil {
		return nil, err
	}
	var p *StructuredPlan
	err := withReadonlyStructuredValidation(root, func(successors *structuredSuccessorValidation) error {
		v.readonlySuccessors = successors
		var err error
		p, err = v.planStructuredAssets(root, locale)
		if err != nil {
			return err
		}
		return v.verify()
	})
	return p, err
}

// One graph spans all assets and packages in this invocation. Reused nodes
// retain their exact dependency proofs; the public boundary seals the graph.
func (v *closureValidation) planStructuredAssets(root, locale string) (*StructuredPlan, error) {
	if err := v.bindLanguageAudit(root, locale); err != nil {
		return nil, err
	}
	if !component(locale) {
		return nil, fmt.Errorf("invalid locale")
	}
	if err := v.requireUnifiedReview(root, locale); err != nil {
		return nil, err
	}
	corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return nil, err
	}
	glossary, err := readRegular(root, "locales/"+locale+"/glossary.yaml")
	if err != nil {
		return nil, err
	}
	review, err := v.unifiedReview(locale)
	if err != nil {
		return nil, err
	}
	p := &StructuredPlan{Schema: StructuredSchema, Locale: locale, CorpusSHA: corpus.Identity, GlossarySHA: digest(glossary), Review: Reference{review.Path, review.SHA256}, Assets: []StructuredAsset{}, Expected: []string{}}
	groups, err := structuredContributorGroups(corpus.Contributors)
	if err != nil {
		return nil, err
	}
	g, err := v.loadCurrent(root)
	if err != nil {
		return nil, err
	}
	docs := map[string]Document{}
	for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
		ds, _, err := CurrentPackageDocuments(root, g, pkg, v.readonlySuccessors)
		if err != nil {
			return nil, err
		}
		for _, d := range ds {
			docs[d.Path] = d
		}
	}
	var completion *Completion
	completed := map[string]Completion{}
	b, err := readRegular(root, LocalePath(locale))
	if err == nil {
		var l Locale
		if err := StrictJSON(b, &l); err != nil {
			return nil, err
		}
		if err := ValidateLocale(g, l); err != nil {
			return nil, err
		}
		for i := range l.Packages {
			completed[l.Packages[i].Package] = l.Packages[i]
			if l.Packages[i].Package == "tour-v1" {
				completion = &l.Packages[i]
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	reuse := map[string]string{}
	baselines := map[string]*structuredBaseline{}
	var checkpointEvidence []Reference
	if completion != nil {
		plan, err := planLocaleLanguage(root, locale, true, v)
		if err != nil {
			return nil, err
		}
		for _, w := range plan.Work {
			reuse[w.ID] = w.State
			if w.Package == "tour-v1" && w.State == "carry" && w.Reason == tourCheckpointReason {
				checkpointEvidence = w.Evidence
			}
		}
	}

	ids := []string{}
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		cs := groups[id]
		paths := []string{}
		for _, c := range cs {
			paths = append(paths, c.Path)
		}
		paths = uniqueStrings(paths)
		a := StructuredAsset{ID: id, Package: cs[0].Package, SourcePaths: paths, SourceSHA: identity(cs), State: "generation-required", Proof: []Reference{}}
		switch id {
		case "tour-ui":
			a.TargetPath = "internal/tour/ui/" + locale + ".json"
		case "article-metadata":
			a.TargetPath = "locales/" + locale + "/article-metadata.json"
		default:
			a.TargetPath = "locales/" + locale + "/structured/" + cs[0].Path
			d, ok := docs[cs[0].Path]
			if !ok {
				// A terminology/source foundation is not yet a reconstructable
				// structured lifecycle. Report it and forbid Bundle export.
				a.State = "unknown"
				p.Assets = append(p.Assets, a)
				continue
			}
			a.Document = &d
		}
		if c, ok := completed[a.Package]; ok && a.Document != nil {
			baseline := baselines[a.Package]
			if baseline == nil {
				baseline, err = v.historicalStructuredCompletion(root, locale, c)
				if err != nil {
					return nil, err
				}
				baselines[a.Package] = baseline
			}
			// Source adoption adds genuinely new assets. A predecessor cannot
			// carry a target it never accepted; it still proves translation carry.
			if _, exists := baseline.Owners[a.Document.Path]; !exists {
				if _, e := readRegular(root, a.TargetPath); e == nil {
					a.State = "stale"
				} else if !os.IsNotExist(e) {
					return nil, e
				}
				if a.State == "generation-required" {
					p.Expected = append(p.Expected, a.TargetPath)
				}
				p.Assets = append(p.Assets, a)
				continue
			}
			a.Assessment, err = v.assessStructuredAsset(root, *p, a, baseline)
			if err != nil {
				return nil, err
			}
			a.TargetSHA, a.Proof = a.Assessment.TargetSHA, c.Evidence
			a.CarriedSlots, err = sourceDeltaCarry(root, *p, a, baseline)
			if err != nil {
				return nil, err
			}
			switch a.Assessment.Classification {
			case "compatible":
				a.State = "carry"
			case "affected":
				a.State = "generation-required"
			default:
				a.State = a.Assessment.Classification
			}
			if a.State == "generation-required" {
				p.Expected = append(p.Expected, a.TargetPath)
			}
			p.Assets = append(p.Assets, a)
			continue
		}
		target, err := readRegular(root, a.TargetPath)
		if err == nil {
			a.TargetSHA = digest(target)
			a.State = "stale"
			if a.Package == "tour-v1" && len(checkpointEvidence) > 0 {
				carried := true
				for _, c := range cs {
					if reuse[c.ID] != "carry" {
						carried = false
					}
				}
				if carried {
					a.State, a.Proof = "carry", checkpointEvidence
				}
			} else if completion != nil && a.Package == "tour-v1" {
				for _, r := range completion.Evidence {
					if r.Path == a.TargetPath && r.SHA256 == a.TargetSHA {
						a.Proof = append(a.Proof, r)
					}
				}
				if len(a.Proof) > 0 {
					cat, err := checkedCatalog(root)
					if err != nil {
						return nil, err
					}
					unchanged := true
					oldGlossary := ""
					for _, r := range completion.Evidence {
						if r.Path == "locales/"+locale+"/glossary.yaml" {
							oldGlossary = r.SHA256
						}
					}
					compatibility := i18n.PrepareGlossaryCompatibility(root, locale, cat)
					for _, c := range cs {
						scope := c.ID
						if reuse[scope] != "carry" {
							unchanged = false
						}
						if oldGlossary != p.GlossarySHA {
							result := compatibility.Resolve(oldGlossary, p.GlossarySHA, scope)
							if result.Status != "compatible" && result.Status != "exact" {
								unchanged = false
							}
						}
						for _, r := range completion.Evidence {
							if r.Path == c.Path {
								source, e := readRegular(root, c.Path)
								if e != nil || digest(source) != r.SHA256 {
									unchanged = false
								}
							}
						}
					}
					if unchanged {
						a.State = "carry"
					}
				}
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		} else {
			a.State = "generation-required"
		}
		// New/missing/stale share one exact generation-required output set.
		if a.State != "carry" && a.State != "context-only" {
			p.Expected = append(p.Expected, a.TargetPath)
		}
		p.Assets = append(p.Assets, a)
	}
	finishStructuredPlan(p, g)
	return p, nil
}

func finishStructuredPlan(p *StructuredPlan, g *Global) {
	// Reopening is one exact affected delta, never an implicit initial campaign
	// for unrelated incomplete packages. Their inputs remain context only.
	reopening := false
	for _, a := range p.Assets {
		if a.Assessment != nil && a.Assessment.Classification == "affected" {
			reopening = true
		}
	}
	if reopening {
		p.Expected = []string{}
		for i := range p.Assets {
			a := &p.Assets[i]
			if a.Assessment != nil && a.Assessment.Classification == "affected" {
				p.Expected = append(p.Expected, a.TargetPath)
			} else if a.Assessment == nil && a.State == "generation-required" && !(g.ShellAdoption != nil && a.Package == "site-v2-shell") {
				a.State = "context-only"
			} else if a.State == "generation-required" {
				p.Expected = append(p.Expected, a.TargetPath)
			}
		}
	}
	sort.Strings(p.Expected)
	p.Identity = ""
	p.Identity = identity(*p)
}

func structuredContributorGroups(contributors []i18n.LanguageContributor) (map[string][]i18n.LanguageContributor, error) {
	groups := map[string][]i18n.LanguageContributor{}
	for _, c := range contributors {
		if i18n.EffectiveContributorPurpose(c) == i18n.ContributorPurposeTerminologyDiscovery {
			if c.Structured {
				return nil, fmt.Errorf("terminology-discovery contributor cannot be structured: %s", c.ID)
			}
			continue
		}
		if i18n.EffectiveContributorPurpose(c) != i18n.ContributorPurposeActiveContent {
			return nil, fmt.Errorf("unknown contributor purpose: %s", c.Purpose)
		}
		if !c.Structured {
			continue
		}
		id := c.ID
		if c.Parser == "ui-visible/v1" {
			id = "tour-ui"
		}
		if c.Parser == "article-header/v1" {
			id = "article-metadata"
		}
		groups[id] = append(groups[id], c)
	}
	return groups, nil
}
func uniqueStrings(xs []string) []string {
	sort.Strings(xs)
	out := []string{}
	for _, x := range xs {
		if len(out) == 0 || out[len(out)-1] != x {
			out = append(out, x)
		}
	}
	return out
}

type StructuredBundle struct {
	TransportProfile string                     `json:"transport_profile,omitempty"`
	ValidationInputs []Reference                `json:"validated_repository_inputs,omitempty"`
	AuthorityProfile string                     `json:"authority_profile,omitempty"`
	RevisionFinding  *Reference                 `json:"revision_finding,omitempty"`
	Schema           string                     `json:"schema"`
	Plan             StructuredPlan             `json:"plan"`
	Files            []i18n.TransportBundleFile `json:"files"`
	Identity         string                     `json:"identity_sha256"`
}

func ExportStructuredAssets(root, locale string) ([]byte, *StructuredBundle, error) {
	return exportStructuredAssets(root, locale, nil)
}
func ExportStructuredAssetsRevision(root, locale string, finding Reference) ([]byte, *StructuredBundle, error) {
	return exportStructuredAssets(root, locale, &finding)
}
func exportStructuredAssets(root, locale string, finding *Reference) ([]byte, *StructuredBundle, error) {
	if !component(locale) {
		return nil, nil, fmt.Errorf("invalid locale")
	}
	v := newClosureValidation(root)
	if err := v.currentInputs(locale); err != nil {
		return nil, nil, err
	}
	b, m, err := v.exportStructuredAssetsReadonly(root, locale, finding)
	return b, m, err
}

// Export/check own the same explicit readonly successor lifetime as the public
// planner. Publication/default callers remain eager; no state crosses calls.
func (v *closureValidation) exportStructuredAssetsReadonly(root, locale string, finding *Reference) ([]byte, *StructuredBundle, error) {
	var b []byte
	var m *StructuredBundle
	err := withReadonlyStructuredValidation(root, func(successors *structuredSuccessorValidation) error {
		v.readonlySuccessors = successors
		defer v.phase("structured-export")()
		var err error
		b, m, err = v.exportStructuredAssets(root, locale, finding)
		if err != nil {
			return err
		}
		return v.verify()
	})
	return b, m, err
}

func (v *closureValidation) exportStructuredAssets(root, locale string, finding *Reference) ([]byte, *StructuredBundle, error) {
	p, err := v.planStructuredAssets(root, locale)
	if err != nil {
		return nil, nil, err
	}
	if finding != nil {
		r, err := validatedSurfaceFinding(root, locale, *finding)
		if err != nil {
			return nil, nil, err
		}
		if err := selectStructuredSurfaceRevision(root, p, r); err != nil {
			return nil, nil, err
		}
	}
	if len(p.Expected) == 0 {
		return nil, nil, fmt.Errorf("no generation-required structured assets")
	}
	for _, a := range p.Assets {
		if a.State == "stale" || a.State == "unknown" {
			return nil, nil, fmt.Errorf("structured asset %s is %s", a.ID, a.State)
		}
	}
	m := &StructuredBundle{Schema: StructuredSchema, AuthorityProfile: StructuredAuthorityProfile, RevisionFinding: finding, Plan: *p, Files: []i18n.TransportBundleFile{}}
	entries := []i18n.TransportBundleEntry{}
	paths := []string{i18n.GlossaryCorpusPath, "locales/" + locale + "/glossary.yaml", "locales/" + locale + "/locale.json", p.Review.Path, structuredTransportContract, "docs/LANGUAGE_DEFECT_EXHAUSTION.md"}
	archives := map[Reference]bool{}
	currentAuthority, err := v.loadCurrent(root)
	if err != nil {
		return nil, nil, err
	}
	if currentAuthority.ShellAdoption != nil {
		a, err := CheckShellAdoption(root, *currentAuthority.ShellAdoption, v.readonlySuccessors)
		if err != nil {
			return nil, nil, err
		}
		paths = append(paths, ShellAdoptionCurrent, currentAuthority.ShellAdoption.Path, a.Foundation.Path, SourceManifestPath)
	}
	for _, ref := range currentAuthority.StructuredSuccessors {
		a, err := checkStructuredSourceSuccessor(root, ref, 0, v.readonlySuccessors)
		if err != nil {
			return nil, nil, err
		}
		paths = append(paths, ref.Path, structuredSourceCurrent(a.Previous.Path), a.Proposal.Path, a.Body.Path)
	}
	paths = append(paths, PackageAuthorities...)
	paths = append(paths, StructuredSemanticAuthorities...)
	if finding != nil {
		paths = append(paths, finding.Path)
	}
	archivedGenerations := map[string]*StructuredGeneration{}
	archivedInputs := map[string]map[string][]byte{}
	for _, a := range p.Assets {
		if a.Document != nil && hasSharedDocumentAuthority(*a.Document) {
			authority, err := sharedDocumentAuthority(*a.Document)
			if err != nil {
				return nil, nil, err
			}
			ref := *authority
			for depth := 0; ; depth++ {
				if depth >= 32 {
					return nil, nil, fmt.Errorf("shared representation ancestry limit")
				}
				paths = append(paths, ref.Path)
				var receipt SharedRepresentationSuccessor
				if err := readReference(root, ref, &receipt); err != nil {
					return nil, nil, err
				}
				if depth == 0 {
					contextPaths, err := sharedReattestationContextPaths(root, receipt, successorValidation(root, []*structuredSuccessorValidation{v.readonlySuccessors}))
					if err != nil {
						return nil, nil, err
					}
					paths = append(paths, contextPaths...)
				}
				if receipt.Previous == nil {
					break
				}
				ref = *receipt.Previous
			}
			paths = append(paths, sharedRepresentationHead(a.Package, a.ID))
		}
		if a.Assessment != nil {
			paths = append(paths, a.Assessment.Closure.Path, a.Assessment.Generation.Path, a.Assessment.Surface.Path)
			if a.Assessment.InputReopen != nil {
				paths = append(paths, a.Assessment.InputReopen.Path)
			}
			r := archivedGenerations[a.Assessment.Generation.Path]
			fs := archivedInputs[a.Assessment.Generation.Path]
			if r == nil {
				var err error
				r, _, fs, err = v.historicalStructuredGeneration(root, a.Assessment.Generation)
				if err != nil {
					return nil, nil, err
				}
				archivedGenerations[a.Assessment.Generation.Path], archivedInputs[a.Assessment.Generation.Path] = r, fs
			}
			paths = append(paths, r.Bundle.Path, r.Plan.Review.Path)
			archives[r.Bundle] = true
			var oldReview i18n.UnifiedGlossaryReview
			if err := readReference(root, r.Plan.Review, &oldReview); err != nil {
				return nil, nil, err
			}
			reviewArchive := Reference{"data/unified-glossary-reviews/" + locale + "-bundles/" + oldReview.BundleSHA + ".zip", oldReview.BundleSHA}
			archives[reviewArchive] = true
			paths = append(paths, reviewArchive.Path)
			var surface IntegratedSurfaceReceipt
			if err := readReference(root, a.Assessment.Surface, &surface); err != nil {
				return nil, nil, err
			}
			paths = append(paths, surface.Evidence.Path)
			if a.State == "generation-required" && a.Document != nil {
				for _, t := range r.Output.Outputs {
					if t.Path != a.TargetPath {
						continue
					}
					data, err := Encode(t)
					if err != nil {
						return nil, nil, err
					}
					name := "previous/" + a.Document.Path + ".json"
					entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: data})
					m.Files = append(m.Files, i18n.NewTransportBundleFile(name, "", data))
				}
				name := "baseline/" + a.ID + "/glossary.yaml"
				data := fs["inputs/locales/"+locale+"/glossary.yaml"]
				entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: data})
				m.Files = append(m.Files, i18n.NewTransportBundleFile(name, "", data))
			}
		}
		if a.State != "carry" && a.State != "context-only" {
			for _, path := range a.SourcePaths {
				if a.Document == nil {
					paths = append(paths, path)
				}
			}
		}
	}
	for _, path := range uniqueStrings(paths) {
		b, err := v.readRegular(root, path)
		if err != nil {
			return nil, nil, err
		}
		name := "inputs/" + path
		entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: b})
		m.Files = append(m.Files, i18n.NewTransportBundleFile(name, path, b))
	}
	global, err := v.loadCurrent(root)
	if err != nil {
		return nil, nil, err
	}
	raw := map[string][]byte{}
	neededPackages := map[string]bool{}
	for _, a := range p.Assets {
		if a.Document != nil && a.State != "carry" && a.State != "context-only" {
			neededPackages[a.Package] = true
		}
	}
	for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
		if !neededPackages[pkg] {
			continue
		}
		_, files, err := CurrentPackageDocuments(root, global, pkg, v.readonlySuccessors)
		if err != nil {
			return nil, nil, err
		}
		for k, b := range files {
			raw[k] = b
		}
	}
	for _, a := range p.Assets {
		if (a.State == "carry" || a.State == "context-only") || a.Document == nil {
			continue
		}
		b := raw[a.Document.Path]
		name := "source/" + a.Document.Path
		entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: b})
		repositoryPath := a.Document.Path
		for _, s := range global.Sources {
			if s.Path == a.Document.Path {
				repositoryPath = currentSourceReference(s).Path
			}
		}
		m.Files = append(m.Files, i18n.NewTransportBundleFile(name, repositoryPath, b))
	}
	if finding != nil {
		var review IntegratedSurfaceReceipt
		if err := readReference(root, *finding, &review); err != nil {
			return nil, nil, err
		}
		for _, ref := range review.Scope.Closures {
			// Full finding validation above already proved current eligibility
			// or exact representation-successor revision authority. Previous
			// targets are historical inputs, not a second current closure gate.
			baseline, err := v.historicalStructuredClosure(root, ref, map[string]bool{})
			if err != nil {
				return nil, nil, err
			}
			previous := baseline.Files
			for _, a := range p.Assets {
				if a.State != "generation-required" || a.Document == nil {
					continue
				}
				if b, ok := previous[a.Document.Path]; ok {
					name := "previous/" + a.Document.Path
					entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: b})
					m.Files = append(m.Files, i18n.NewTransportBundleFile(name, "", b))
				}
			}
		}
	}
	entries, err = projectStructuredArchiveTransport(m, entries, archives)
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].BundlePath < m.Files[j].BundlePath })
	m.Identity = identity(*m)
	manifest, err := Encode(m)
	if err != nil {
		return nil, nil, err
	}
	b, err := i18n.WriteDeterministicTransportBundle(manifest, entries)
	if err == nil {
		_, _, err = readStructuredBundle(b) // Never export an unusable/oversized transport.
	}
	return b, m, err
}
func CheckStructuredAssetsBundle(root string, b []byte) (*StructuredBundle, error) {
	m, fs, err := readStructuredBundle(b)
	if err != nil {
		return nil, err
	}
	want, current, err := exportStructuredAssets(root, m.Plan.Locale, m.RevisionFinding)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(want, b) {
		if m.AuthorityProfile != StructuredAuthorityProfile {
			return nil, fmt.Errorf("structured bundle stale/noncanonical")
		}
		_, currentFiles, err := readStructuredBundle(want)
		if err != nil {
			return nil, err
		}
		if err := compatibleStructuredBundles(*m, fs, *current, currentFiles); err != nil {
			return nil, fmt.Errorf("structured bundle stale/noncanonical: %w", err)
		}
	}
	return m, nil
}

type StructuredTarget struct {
	Path  string   `json:"path"`
	Data  string   `json:"data"`
	Slots []Target `json:"slots"`
}
type StructuredOutput struct {
	Schema    string             `json:"schema"`
	BundleSHA string             `json:"bundle_identity_sha256"`
	Outputs   []StructuredTarget `json:"outputs"`
}
type StructuredGeneration struct {
	TimestampRecovery *Reference              `json:"timestamp_recovery,omitempty"`
	CarryForward      *StructuredCarryForward `json:"carry_forward,omitempty"`
	Schema            string                  `json:"schema"`
	Plan              StructuredPlan          `json:"plan"`
	Output            StructuredOutput        `json:"output"`
	Provenance        Provenance              `json:"generation"`
	Bundle            Reference               `json:"bundle"`
	Identity          string                  `json:"identity_sha256"`
}

func validateStructuredOutput(root string, p StructuredPlan, o StructuredOutput) (map[string][]byte, error) {
	if o.Schema != StructuredSchema || len(o.Outputs) != len(p.Expected) {
		return nil, fmt.Errorf("structured output exact-set mismatch")
	}
	all := map[string]StructuredAsset{}
	for _, a := range p.Assets {
		all[a.TargetPath] = a
	}
	files := map[string][]byte{}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	selected := []StructuredAsset{}
	for _, t := range o.Outputs {
		selected = append(selected, all[t.Path])
	}
	docs, raw, err := currentStructuredDocuments(root, g, selected)
	if err != nil {
		return nil, err
	}
	glossary, err := i18n.LoadGlossary(root, p.Locale)
	if err != nil {
		return nil, err
	}
	for i, t := range o.Outputs {
		if t.Path != p.Expected[i] {
			return nil, fmt.Errorf("structured output outside generation-required exact set")
		}
		a := all[t.Path]
		if a.Document != nil {
			if identity(docs[a.Document.Path]) != identity(*a.Document) {
				return nil, fmt.Errorf("structured current exact input mismatch")
			}
			values, err := structuredTargetValues(a, t, glossary)
			if err != nil {
				return nil, err
			}
			var b []byte
			if a.Assessment != nil && a.Assessment.SourceSuccessor != nil {
				b, err = reconstructFrozenStructuredRepresentation(*a.Document, raw[a.Document.Path], values)
			} else {
				b, err = reconstructAuthorityStructuredRepresentation(*a.Document, raw[a.Document.Path], values)
			}
			if err != nil {
				return nil, err
			}
			files[t.Path] = b
		} else {
			if len(t.Slots) != 0 {
				return nil, fmt.Errorf("Tour structured asset is complete file")
			}
			b := []byte(t.Data)
			if err := textBytes(b); err != nil {
				return nil, err
			}
			switch a.ID {
			case "tour-ui":
				en, err := readRegular(root, "internal/tour/ui/en.json")
				if err != nil {
					return nil, err
				}
				if err := ui.ValidateLocalizedBytes(p.Locale, en, b); err != nil {
					return nil, err
				}
			case "article-metadata":
				cat, err := checkedCatalog(root)
				if err != nil {
					return nil, err
				}
				if _, err := i18n.ValidateArticleMetadataBytes(p.Locale, cat, b); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unknown structured validator")
			}
			files[t.Path] = b
		}
	}
	if _, err := structuredLanguageAttention(root, p, raw, files); err != nil {
		return nil, err
	}
	return files, nil
}
func ImportStructuredAssets(root string, bundle, output []byte, provenance Provenance) (*StructuredGeneration, string, error) {
	m, err := CheckStructuredAssetsBundle(root, bundle)
	if err != nil {
		return nil, "", err
	}
	if err := validateProvenance(provenance); err != nil {
		return nil, "", err
	}
	if err := requireIndependentSession(root, m.Plan.Locale, provenance.Session, "generation"); err != nil {
		return nil, "", err
	}
	var o StructuredOutput
	if err := StrictJSON(output, &o); err != nil {
		return nil, "", err
	}
	if o.BundleSHA != m.Identity {
		return nil, "", fmt.Errorf("structured bundle identity mismatch")
	}
	if _, err := validateStructuredOutput(root, m.Plan, o); err != nil {
		return nil, "", err
	}
	ref := Reference{Path: "data/locale-language/" + m.Plan.Locale + "/structured-bundles/" + digest(bundle) + ".zip", SHA256: digest(bundle)}
	r := &StructuredGeneration{Schema: StructuredSchema, Plan: m.Plan, Output: o, Provenance: provenance, Bundle: ref}
	r.Identity = identity(*r)
	p := "data/locale-language/" + m.Plan.Locale + "/structured-generation/" + r.Identity + ".json"
	if err := SaveArtifact(root, ref.Path, bundle); err != nil {
		return nil, "", err
	}
	if err := saveImmutable(root, p, r); err != nil {
		return nil, "", err
	}
	return r, p, nil
}
func CheckStructuredGeneration(root string, ref Reference) (*StructuredGeneration, map[string][]byte, error) {
	return checkStructuredGeneration(root, ref, "")
}
func (validation *closureValidation) computeCurrentStructuredGeneration(root string, ref Reference, pkg string) (*StructuredGeneration, map[string][]byte, error) {
	if _, _, _, err := validation.historicalStructuredGeneration(root, ref); err != nil {
		return nil, nil, err
	}
	var r StructuredGeneration
	if err := validation.readReference(root, ref, &r); err != nil {
		return nil, nil, err
	}
	copy := r
	copy.Identity = ""
	if r.Schema != StructuredSchema || r.Identity != identity(copy) || ref.Path != "data/locale-language/"+r.Plan.Locale+"/structured-generation/"+r.Identity+".json" {
		return nil, nil, fmt.Errorf("invalid structured evidence identity")
	}
	if _, err := validateStructuredProvenance(root, &r); err != nil {
		return nil, nil, err
	}
	_, m, fs, err := readStructuredGeneration(root, ref)
	if err != nil {
		return nil, nil, err
	}
	if m.AuthorityProfile != "" {
		paths := append([]string(nil), StructuredSemanticAuthorities...)
		for _, f := range m.Files {
			if strings.HasPrefix(f.RepositoryPath, "docs/") && !structuredExecutionAuthority(f.RepositoryPath) {
				paths = append(paths, f.RepositoryPath)
			}
		}
		for _, name := range uniqueStrings(paths) {
			if _, err := validation.readRegular(root, name); err != nil {
				return nil, nil, err
			}
		}
	}
	if err := checkStructuredSemanticAuthority(root, *m, fs); err != nil {
		return nil, nil, err
	}
	if err := validation.bindLanguageAudit(root, r.Plan.Locale); err != nil {
		return nil, nil, err
	}
	p := r.Plan
	o := r.Output
	if err := validation.requireUnifiedReview(root, p.Locale); err != nil {
		return nil, nil, err
	}
	if err := validation.validateStructuredDelta(root, &r, m); err != nil {
		return nil, nil, err
	}
	glob, err := validation.loadCurrent(root)
	if err != nil {
		return nil, nil, err
	}
	selected := []StructuredAsset{}
	for _, a := range p.Assets {
		if a.Document == nil || (pkg != "" && a.Package != pkg) {
			continue
		}
		for _, t := range o.Outputs {
			if t.Path == a.TargetPath {
				selected = append(selected, a)
			}
		}
	}
	currentDocs, _, err := currentStructuredDocuments(root, glob, selected)
	if err != nil {
		return nil, nil, err
	}
	for _, a := range p.Assets {
		if a.Document == nil || (pkg != "" && a.Package != pkg) {
			continue
		}
		for _, t := range o.Outputs {
			if t.Path == a.TargetPath && identity(currentDocs[a.Document.Path]) != identity(*a.Document) {
				return nil, nil, fmt.Errorf("structured current source drift: %s", a.ID)
			}
		}
	}
	if pkg != "" {
		assets := []StructuredAsset{}
		wanted := map[string]bool{}
		for _, a := range p.Assets {
			if a.Package == pkg {
				assets = append(assets, a)
				wanted[a.TargetPath] = true
			}
		}
		p.Assets = assets
		p.Expected = []string{}
		o.Outputs = []StructuredTarget{}
		for _, t := range r.Output.Outputs {
			if wanted[t.Path] {
				p.Expected = append(p.Expected, t.Path)
				o.Outputs = append(o.Outputs, t)
			}
		}
	}
	// Per-asset semantic reuse; corpus growth elsewhere is not a language gate.
	current, err := validation.readRegular(root, "locales/"+p.Locale+"/glossary.yaml")
	if err != nil {
		return nil, nil, err
	}
	if digest(current) != p.GlossarySHA {
		old := fs["inputs/locales/"+p.Locale+"/glossary.yaml"]
		delta, err := i18n.ParsedGlossaryDelta(p.Locale, old, current)
		if err != nil {
			return nil, nil, err
		}
		for _, a := range p.Assets {
			if a.Document == nil {
				return nil, nil, fmt.Errorf("Tour structured carry requires original V2-B scope proof")
			}
			for _, t := range o.Outputs {
				if t.Path != a.TargetPath {
					continue
				}
				target := map[string]string{}
				for _, v := range t.Slots {
					target[v.ID] = v.Text
				}
				for _, u := range a.Document.Units {
					ok, _ := i18n.ParsedGlossaryImpact(u.Source, target[u.ID], delta)
					if !ok {
						return nil, nil, fmt.Errorf("affected structured asset %s", a.ID)
					}
				}
			}
		}
	}
	files, err := validateStructuredOutput(root, p, o)
	return &r, files, err
}
func structuredSupersedes(root string, g *StructuredGeneration, parent Reference, path string) bool {
	if g != nil && g.CarryForward != nil {
		original, _, _, err := readStructuredGeneration(root, parent)
		if err == nil && original.CarryForward == nil && original.Bundle == g.Bundle && reflect.DeepEqual(original.Plan, g.Plan) && reflect.DeepEqual(original.Output, g.Output) && original.Provenance == g.Provenance {
			for _, a := range g.Plan.Assets {
				if a.Document != nil && a.Document.Path == path {
					for _, t := range g.Output.Outputs {
						if t.Path == a.TargetPath {
							return true
						}
					}
				}
			}
		}
	}
	b, err := readRegular(root, g.Bundle.Path)
	if err != nil {
		return false
	}
	fs, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		return false
	}
	var m StructuredBundle
	if StrictJSON(fs["manifest.json"], &m) != nil || m.RevisionFinding == nil {
		return false
	}
	var r IntegratedSurfaceReceipt
	if readReference(root, *m.RevisionFinding, &r) != nil || r.Decision != "failed" {
		return false
	}
	copy := r
	copy.Identity = ""
	scope := r.Scope
	scope.Identity = ""
	if r.Schema != IntegratedSurfaceSchema || r.Identity != identity(copy) || r.Scope.Identity != identity(scope) || r.Scope.Locale != g.Plan.Locale {
		return false
	}
	if _, err := readRegularReference(root, r.Evidence); err != nil {
		return false
	}
	if err := requireIndependentSession(root, r.Scope.Locale, r.Reviewer, "reviewer"); err != nil {
		return false
	}
	exact := false
	owners, err := reviewedFindingAssets(root, &r)
	if err != nil {
		return false
	}
	asset, ok := owners[path]
	if !ok {
		return false
	}
	for _, f := range r.ExactFindings {
		if f.ID == path && structuredLanguageFinding(f, asset) {
			exact = true
		}
	}
	if !exact {
		return false
	}
	// Validate every scope reference before an owner match can return success.
	// Only the finding owner's exact package participates in owner resolution.
	closures := map[Reference]PackageClosure{}
	for _, ref := range r.Scope.Closures {
		var c PackageClosure
		if readReference(root, ref, &c) != nil {
			return false
		}
		if c.Package == asset.Package {
			closures[ref] = c
		}
	}
	for _, ref := range r.Scope.Closures {
		c, ok := closures[ref]
		if !ok {
			continue
		}
		for _, old := range c.Structured {
			if old == parent {
				return true
			}
		}
		// A mandatory shared composition finding may name a carried asset whose
		// Generation owner is inherited through the closure's frozen baseline.
		// Resolve the exact path's accepted owner, never an arbitrary ancestor.
		if c.StructuredBaseline != nil {
			owner, err := compositionStructuredOwner(root, ref, r.Scope.Locale, asset, map[Reference]bool{})
			if err != nil {
				return false
			}
			if owner == parent {
				return true
			}
		}
	}
	return false
}
