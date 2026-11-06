package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"path/filepath"
	"sort"
	"strings"
)

const LocalePlanSchema = "go-learning/locale-language-work-plan/v1"

type LanguageWork struct {
	SourcePath string      `json:"source_path,omitempty"`
	Route      string      `json:"canonical_route,omitempty"`
	ID         string      `json:"id"`
	Package    string      `json:"package"`
	State      string      `json:"state"`
	Reason     string      `json:"reason"`
	SourceSHA  string      `json:"source_sha256"`
	TargetSHA  string      `json:"target_sha256"`
	Evidence   []Reference `json:"evidence"`
}
type LocaleLanguagePlan struct {
	Schema      string         `json:"schema"`
	Locale      string         `json:"locale"`
	GlossarySHA string         `json:"glossary_sha256"`
	Work        []LanguageWork `json:"work"`
	Identity    string         `json:"identity_sha256"`
}

// ClassifyLanguageWork is shared by content and structured consumers. It never
// treats a milestone or unrelated package identity as a language invalidator.
func ClassifyLanguageWork(exact, hasTarget, hasA, compatible, retired, ambiguous bool) string {
	if ambiguous {
		return "ambiguous"
	}
	if retired {
		return "retired"
	}
	if !hasTarget {
		return "generation-required"
	}
	if !exact || !compatible {
		return "stale/reopen"
	}
	if !hasA {
		return "review-required"
	}
	return "carry"
}
func PlanLocaleLanguage(root, locale string) (*LocaleLanguagePlan, error) {
	return planLocaleLanguage(root, locale, false)
}

// A consumer of Tour carry does not replay other packages' current completion.
func planLocaleLanguage(root, locale string, tourOnly bool, validation ...*closureValidation) (*LocaleLanguagePlan, error) {
	if !component(locale) {
		return nil, fmt.Errorf("invalid locale")
	}
	if err := i18n.CheckPromotionTransactionState(root, locale); err != nil {
		return nil, err
	}
	if err := i18n.CheckCourseRefreshTransactionState(root, locale); err != nil {
		return nil, err
	}
	var successors *structuredSuccessorValidation
	if len(validation) > 0 && validation[0] != nil {
		successors = validation[0].readonlySuccessors
	}
	g, err := LoadCurrent(root, successors)
	if err != nil {
		return nil, err
	}
	l, err := readLocaleAuthority(root, g, locale)
	if err != nil {
		return nil, err
	}
	if l.Locale != locale || l.Schema != LocaleSchema {
		return nil, fmt.Errorf("invalid locale authority")
	}
	glossary, err := readRegular(root, "locales/"+locale+"/glossary.yaml")
	if err != nil {
		return nil, err
	}
	p := &LocaleLanguagePlan{Schema: LocalePlanSchema, Locale: locale, GlossarySHA: digest(glossary), Work: []LanguageWork{}}
	for _, c := range l.Packages {
		if c.Package != "tour-v1" {
			continue
		}
		// Full original closure remains the trusted evidence root. Any unverifiable
		// original evidence stops; no synthetic legacy PASS is introduced.
		if err := ValidateLocale(g, *l); err != nil {
			return nil, err
		}
		refs := map[string]Reference{}
		oldSHA := ""
		courseChanged := false
		cat, err := checkedCatalog(root)
		if err != nil {
			return nil, err
		}
		checkpoint, current, err := currentTourCheckpoint(root, g, cat, *l, validation...)
		if err != nil {
			return nil, err
		}
		if current {
			contexts, err := i18n.LocaleLanguageContexts(root, locale, cat)
			if err != nil {
				return nil, err
			}
			for _, ctx := range contexts {
				p.Work = append(p.Work, LanguageWork{ID: ctx.Scope, Package: c.Package, State: "carry", Reason: tourCheckpointReason, SourceSHA: digest([]byte(ctx.Source)), TargetSHA: digest([]byte(ctx.Target)), Evidence: checkpoint})
			}
			continue
		}
		canonicalTargets, err := i18n.CanonicalTranslationUnitTargetPaths(cat, locale)
		if err != nil {
			return nil, err
		}
		// Package discovery belongs to TU selection; it must not alter the
		// catalog/source identity used by the independent full Tour Surface gate.
		if err := AttachRegisteredTranslationPackages(root, cat, successors); err != nil {
			return nil, err
		}
		AttachIntegratedSurfaceAuthority(cat)
		if len(validation) > 0 {
			validation[0].attachReplacementReuse(cat)
		}
		for _, r := range c.Evidence {
			refs[r.Path] = r
			if r.Path == "locales/"+locale+"/glossary.yaml" {
				oldSHA = r.SHA256
				continue
			}
			b, err := readRegular(root, r.Path)
			if err != nil || digest(b) != r.SHA256 {
				if r.Path == "locales/"+locale+"/course-metadata.json" {
					if _, err := i18n.AuthorizeHistoricalCourseMetadataReference(root, locale, r.SHA256, cat); err != nil {
						return nil, fmt.Errorf("historical Course SEO evidence changed: %w", err)
					}
					courseChanged = true
					continue
				}
				if !canonicalTargets[r.Path] && r.Path != "locales/"+locale+"/status.tsv" {
					return nil, fmt.Errorf("historical non-TU evidence changed %s: exact-current Tour Surface checkpoint required", r.Path)
				}
				if err := i18n.AuthorizeHistoricalCanonicalReference(root, locale, r.Path, r.SHA256, cat); err != nil {
					return nil, fmt.Errorf("historical closure evidence changed %s: %w", r.Path, err)
				}
			}
		}
		reviewID := ""
		for _, r := range c.Evidence {
			if strings.HasSuffix(r.Path, ".a-gate.json") {
				reviewID = strings.TrimSuffix(filepath.Base(r.Path), ".a-gate.json")
			}
		}
		if oldSHA != p.GlossarySHA || courseChanged {
			current, _, err := i18n.ExportLocaleSurfaceReviewPackage(root, locale, cat)
			if err != nil {
				return nil, err
			}
			historical, err := i18n.HistoricalTourSurfacePackageForDeltaPlanning(root, locale, reviewID, oldSHA, current, cat)
			if err != nil {
				return nil, err
			}
			profiles, err := liveProfiles(root)
			if err != nil {
				return nil, err
			}
			verified := false
			for _, profile := range profiles {
				if profile.Locale == locale {
					reconstructed := identity(struct {
						Profile                liveProfile
						ValidatedContextSHA256 string
					}{profile, digest(historical)})
					if reconstructed != c.ContextIdentity {
						return nil, fmt.Errorf("original Tour context proof changed: recorded=%s reconstructed=%s", c.ContextIdentity, reconstructed)
					}
					verified = true
				}
			}
			if !verified {
				return nil, fmt.Errorf("unproven historical Tour profile")
			}
		}
		contexts, err := i18n.LocaleLanguageContexts(root, locale, cat)
		if err != nil {
			return nil, err
		}
		if oldSHA == p.GlossarySHA && !courseChanged {
			if err := verifyCarriedLocale(root, g, Locale{Schema: l.Schema, Locale: locale, Packages: []Completion{c}}); err != nil {
				return nil, err
			}
		}
		closureRef, err := ReferenceFile(root, LocalePath(locale))
		if err != nil {
			return nil, err
		}
		compatibility := i18n.PrepareGlossaryCompatibility(root, locale, cat)
		for _, ctx := range contexts {
			exact := true
			for _, ref := range ctx.References {
				old, ok := refs[ref.Path]
				if ok && old.SHA256 != ref.SHA256 {
					exact = false
				}
			}
			compatible := oldSHA == p.GlossarySHA
			reason := "exact original closure"
			proof := []Reference{closureRef}
			if !compatible {
				resolution := compatibility.Resolve(oldSHA, p.GlossarySHA, ctx.Scope)
				compatible = resolution.Status == "compatible" || resolution.Status == "exact"
				reason = resolution.Status
				for _, r := range resolution.Chain {
					proof = append(proof, Reference{Path: r.Path, SHA256: r.SHA256})
				}
			}
			state := ClassifyLanguageWork(exact, true, true, compatible, false, false)
			p.Work = append(p.Work, LanguageWork{ID: ctx.Scope, Package: c.Package, State: state, Reason: reason, SourceSHA: digest([]byte(ctx.Source)), TargetSHA: digest([]byte(ctx.Target)), Evidence: proof})
		}
	}
	completed := map[string]Completion{}
	for _, c := range l.Packages {
		completed[c.Package] = c
	}
	var structured *StructuredPlan
	if !tourOnly {
		structured, err = PlanStructuredAssets(root, locale)
		if err != nil {
			return nil, err
		}
	}
	for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
		if tourOnly {
			continue
		}
		docs, _, err := CurrentPackageDocuments(root, g, pkg)
		if err != nil {
			return nil, err
		}
		c, active := completed[pkg]
		var files map[string][]byte
		pageWork := map[string]i18n.CurrentTranslationAssessmentUnit{}
		if active {
			baseline, e := historicalStructuredCompletion(root, locale, c)
			err = e
			if err != nil {
				return nil, err
			}
			files = baseline.Files
			if len(baseline.Closure.PageComposition) > 0 {
				cat, err := checkedCatalog(root)
				if err != nil {
					return nil, err
				}
				if err := AttachRegisteredTranslationPackages(root, cat, successors); err != nil {
					return nil, err
				}
				AttachIntegratedSurfaceAuthority(cat)
				for _, in := range baseline.Closure.PageComposition {
					h, err := i18n.HistoricalTranslationPackageCatalog(cat, in.Package, in.PackageAuthority)
					if err != nil {
						return nil, err
					}
					var f i18n.QualityCheckFinalization
					if err := readReference(root, Reference{Path: in.Finalization.Path, SHA256: in.Finalization.SHA256}, &f); err != nil {
						return nil, err
					}
					a, err := i18n.AssessCurrentTranslation(root, h, locale, f.SnapshotID)
					if err != nil {
						return nil, err
					}
					for _, u := range a.Units {
						if _, duplicate := pageWork[u.Snapshot.SourcePath]; duplicate {
							return nil, fmt.Errorf("planner overlapping composition")
						}
						pageWork[u.Snapshot.SourcePath] = u
					}
				}
			} else if baseline.Closure.PageFinalization != nil {
				cat, err := checkedCatalog(root)
				if err != nil {
					return nil, err
				}
				if err := AttachRegisteredTranslationPackages(root, cat, successors); err != nil {
					return nil, err
				}
				AttachIntegratedSurfaceAuthority(cat)
				if err := cat.UseTranslationPackage(pkg); err != nil {
					return nil, err
				}
				var baselineFinal i18n.QualityCheckFinalization
				if err := readReference(root, *baseline.Closure.PageFinalization, &baselineFinal); err != nil {
					return nil, err
				}
				inventory := *cat
				inventory.TranslationScope = nil
				if selected, registered, err := i18n.SelectFinalizedTranslationPackage(root, &inventory, locale, baselineFinal.SnapshotID); err != nil {
					return nil, err
				} else if registered {
					cat = selected
				}
				authorization, err := i18n.FindCurrentScopeTranslationAuthorization(root, cat, locale)
				if err != nil || authorization == nil {
					var f i18n.QualityCheckFinalization
					if err := readReference(root, *baseline.Closure.PageFinalization, &f); err != nil {
						return nil, err
					}
					authorization, err = i18n.AssessCurrentTranslation(root, cat, locale, f.SnapshotID)
					if err != nil {
						return nil, err
					}
				}
				for _, u := range authorization.Units {
					pageWork[u.Snapshot.SourcePath] = u
					target, err := readRegular(root, u.Snapshot.CandidatePath)
					if err != nil || digest(target) != u.Snapshot.CandidateSHA256 {
						return nil, fmt.Errorf("current Page planner candidate drift")
					}
					files[u.Snapshot.SourcePath] = target
				}
			}
		}
		for _, d := range docs {
			if active {
				if d.UnitScope == PageContract && pkg == "site-v2-shell" {
					plan, err := PlanHomepage(root, locale)
					if err != nil {
						return nil, err
					}
					for _, u := range plan.Units {
						u.Package = pkg
						p.Work = append(p.Work, u)
					}
					continue
				}
				state, reason := "carry", "historical closure + prior Surface"
				if u, ok := pageWork[d.Path]; ok {
					state, reason = u.Classification, "shared current TU authorization"
					if state == "compatible" {
						state = "carry"
					} else if state == "affected" {
						state = "generation-required"
					}
				}
				for _, a := range structured.Assets {
					if a.Document != nil && a.Document.Path == d.Path {
						state, reason = a.State, "current per-asset structured assessment"
					}
				}
				p.Work = append(p.Work, LanguageWork{ID: d.Path, Package: pkg, State: state, Reason: reason, SourceSHA: d.SourceSHA, TargetSHA: digest(files[d.Path]), Evidence: c.Evidence})
				continue
			}
			p.Work = append(p.Work, LanguageWork{ID: d.Path, Package: pkg, State: "generation-required", Reason: "new/missing surface", SourceSHA: d.SourceSHA, Evidence: []Reference{}})
		}
	}
	sort.Slice(p.Work, func(i, j int) bool { return p.Work[i].ID < p.Work[j].ID })
	p.Identity = identity(*p)
	return p, nil
}
