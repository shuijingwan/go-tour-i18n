package sitecontent

import (
	"fmt"
	"os"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// representationRevisionScope is a predecessor proof for revision authorization,
// never a current Surface scope or a closure eligibility result. The ordinary
// builders still require fresh language generation for a changed representation.
// Only the selected structured owners may defer that gate, after their exact
// frozen target and current shared-representation lineage have been proved.
func representationRevisionScope(root string, r *IntegratedSurfaceReceipt) (*IntegratedSurfaceScope, error) {
	owners, err := reviewedFindingAssets(root, r)
	if err != nil {
		return nil, err
	}
	structured := map[string]map[string]SurfaceAsset{}
	for _, f := range r.ExactFindings {
		// A sealed judgment may also contain independent TU/integration work.
		// Its original findings and full scope remain validated, not converted
		// into structured representation authority by this projection.
		if f.Class == i18n.SurfaceTranslationUnitLanguage || f.Class == "surface-integration" {
			continue
		}
		a := owners[f.ID]
		if f.Class != "structured-surface-language" || !structuredLanguageFinding(f, a) || a.UnitID != "" || f.ID != a.ID {
			return nil, fmt.Errorf("representation revision requires exact structured finding owner")
		}
		if structured[a.Package] == nil {
			structured[a.Package] = map[string]SurfaceAsset{}
		}
		structured[a.Package][a.ID] = a
	}
	if len(structured) == 0 {
		return nil, fmt.Errorf("no representation revision owner")
	}
	v := newClosureValidation(root)
	if err := v.currentInputs(r.Scope.Locale); err != nil {
		return nil, err
	}
	var scope *IntegratedSurfaceScope
	err = withReadonlyStructuredValidation(root, func(ctx *structuredSuccessorValidation) error {
		v.readonlySuccessors = ctx
		// Reuse the canonical contributor registry and current Documents for
		// these closure owners. The scope builder below verifies retained Tour
		// and all other packages once; a global Generation plan is not another
		// source of finding authority or a prerequisite for this exact revision.
		if err := v.requireUnifiedReview(root, r.Scope.Locale); err != nil {
			return err
		}
		corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
		if err != nil {
			return err
		}
		groups, err := structuredContributorGroups(corpus.Contributors)
		if err != nil {
			return err
		}
		glossary, err := v.readRegular(root, "locales/"+r.Scope.Locale+"/glossary.yaml")
		if err != nil {
			return err
		}
		review, err := v.unifiedReview(r.Scope.Locale)
		if err != nil {
			return err
		}
		plan := StructuredPlan{Schema: StructuredSchema, Locale: r.Scope.Locale, CorpusSHA: corpus.Identity, GlossarySHA: digest(glossary), Review: Reference{review.Path, review.SHA256}}
		g, err := v.loadCurrent(root)
		if err != nil {
			return err
		}
		selected := map[string]map[string]SurfaceAsset{}
		for pkg, assets := range structured {
			docs, _, err := CurrentPackageDocuments(root, g, pkg, ctx)
			if err != nil {
				return err
			}
			for path, asset := range assets {
				var document *Document
				for i := range docs {
					if docs[i].Path == path {
						if document != nil {
							return fmt.Errorf("ambiguous representation finding Document")
						}
						document = &docs[i]
					}
				}
				if document == nil || document.Contract != ShellExtractionContract || document.Kind != "data" || document.UnitScope != "" {
					// Ordinary structured findings get no representation exception.
					// Complete scope reconstruction still requires them compatible.
					continue
				}
				ok, err := sharedCompositionFindingOwner(root, r.Scope, asset)
				if err != nil || !ok {
					return fmt.Errorf("representation revision requires registered shared composition: %v", err)
				}
				if selected[pkg] == nil {
					selected[pkg] = map[string]SurfaceAsset{}
				}
				selected[pkg][path] = asset
			}
		}
		if len(selected) == 0 {
			return fmt.Errorf("no representation revision owner")
		}
		matched := map[string]bool{}
		check := func(root string, ref Reference) (*PackageClosure, map[string][]byte, error) {
			var c PackageClosure
			if err := v.readReference(root, ref, &c); err != nil {
				return nil, nil, err
			}
			wanted := selected[c.Package]
			if len(wanted) == 0 {
				return v.checkPackageClosure(root, ref)
			}
			baseline, err := v.historicalStructuredClosure(root, ref, map[string]bool{})
			if err != nil {
				return nil, nil, err
			}
			materialize := func(root, locale, pkg string, generations []Reference, frozen ...*Reference) (map[string][]byte, *Reference, bool, error) {
				files := map[string][]byte{}
				seen := map[string]bool{}
				docs, _, err := CurrentPackageDocuments(root, g, pkg, ctx)
				if err != nil {
					return nil, nil, false, err
				}
				for _, d := range docs {
					if d.UnitScope == PageContract {
						continue
					}
					path := d.Path
					if seen[path] {
						return nil, nil, false, fmt.Errorf("ambiguous current structured revision owner")
					}
					seen[path] = true
					owner, ok := baseline.Owners[path]
					if !ok {
						return nil, nil, false, fmt.Errorf("current structured owner absent from reviewed closure: %s", path)
					}
					old := baseline.Generations[owner.Path]
					var a StructuredAsset
					matches := 0
					for _, candidate := range old.Plan.Assets {
						if candidate.Package == pkg && candidate.Document != nil && candidate.Document.Path == path {
							a = candidate
							matches++
						}
					}
					cs := groups[a.ID]
					if matches != 1 || len(cs) == 0 || cs[0].Package != pkg || cs[0].Path != path || a.TargetPath != "locales/"+locale+"/structured/"+path {
						return nil, nil, false, fmt.Errorf("current structured registry owner changed: %s", path)
					}
					a.Document = &d
					a.SourceSHA = identity(cs)
					paths := []string{}
					for _, c := range cs {
						paths = append(paths, c.Path)
					}
					a.SourcePaths = uniqueStrings(paths)
					assessment, err := v.assessStructuredAsset(root, plan, a, baseline)
					if err != nil {
						return nil, nil, false, err
					}
					if owner, selected := wanted[path]; selected {
						// This existing assessment proves package/asset/path/raw source,
						// exact Old -> New slots/releases, the selected head and its
						// current optional dependency re-attestation. Unit IDs may change.
						if assessment.Classification != "affected" || assessment.RepresentationSuccessor == nil || assessment.TargetSHA != owner.TargetSHA || assessment.Generation.Path == "" {
							return nil, nil, false, fmt.Errorf("finding has no exact current representation successor")
						}
						actual, err := compositionStructuredOwner(root, ref, locale, owner, map[Reference]bool{})
						if err != nil || actual != assessment.Generation {
							return nil, nil, false, fmt.Errorf("finding predecessor Generation mismatch: %v", err)
						}
						key := pkg + "\x00" + path
						if matched[key] {
							return nil, nil, false, fmt.Errorf("duplicate representation finding owner")
						}
						matched[key] = true
					} else if assessment.Classification != "compatible" {
						return nil, nil, false, fmt.Errorf("unrelated structured scope drift: %s: %s", path, assessment.Reason)
					}
					// Structured imports store accepted targets in immutable
					// Generation outputs; TargetPath is a virtual output address,
					// not a required installed file. Any physical override must
					// nevertheless agree with that exact predecessor.
					current, err := v.readRegular(root, a.TargetPath)
					if (err != nil && !os.IsNotExist(err)) || (err == nil && digest(current) != digest(baseline.Files[path])) {
						return nil, nil, false, fmt.Errorf("Surface predecessor target changed: %s", path)
					}
					if os.IsNotExist(err) {
						v.absent[a.TargetPath] = true
					}
					files[path] = baseline.Files[path]
				}
				if len(files) != len(baseline.Owners) {
					return nil, nil, false, fmt.Errorf("structured revision membership changed: current=%d historical=%d", len(files), len(baseline.Owners))
				}
				return files, c.StructuredBaseline, true, nil
			}
			// Reuse the complete current Page A-only, package identity, source
			// inventory and membership checks. Only structured materialization
			// has the exact, revision-only exception above.
			want, files, err := v.buildPackageClosureMaterialized(root, c.Locale, c.Package, c.PageFinalization, c.Structured, c.PageComposition, materialize, c.StructuredBaseline)
			if err != nil {
				return nil, nil, err
			}
			if !reflect.DeepEqual(c, *want) {
				return nil, nil, fmt.Errorf("representation revision predecessor closure changed")
			}
			return want, files, nil
		}
		scope, err = v.buildIntegratedSurfaceScopeWithClosures(root, r.Scope.Locale, r.Scope.Closures, check)
		if err != nil {
			return err
		}
		for pkg, paths := range selected {
			for path := range paths {
				if !matched[pkg+"\x00"+path] {
					return fmt.Errorf("representation finding owner not in exact closure")
				}
			}
		}
		// Entire predecessor scope equality remains required: closures,
		// glossary/review, carried/fresh assets, Tour, Pages and runtime context.
		if !reflect.DeepEqual(r.Scope, *scope) {
			return fmt.Errorf("unrelated Surface finding context changed")
		}
		return v.verify()
	})
	return scope, err
}
