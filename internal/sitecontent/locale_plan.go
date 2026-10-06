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
	ID        string      `json:"id"`
	Package   string      `json:"package"`
	State     string      `json:"state"`
	Reason    string      `json:"reason"`
	SourceSHA string      `json:"source_sha256"`
	TargetSHA string      `json:"target_sha256"`
	Evidence  []Reference `json:"evidence"`
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
	if !component(locale) {
		return nil, fmt.Errorf("invalid locale")
	}
	g, err := LoadCurrent(root)
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
		for _, r := range c.Evidence {
			refs[r.Path] = r
			if r.Path == "locales/"+locale+"/glossary.yaml" {
				oldSHA = r.SHA256
				continue
			}
			b, err := readRegular(root, r.Path)
			if err != nil || digest(b) != r.SHA256 {
				return nil, fmt.Errorf("historical closure evidence changed %s", r.Path)
			}
		}
		reviewID := ""
		for _, r := range c.Evidence {
			if strings.HasSuffix(r.Path, ".a-gate.json") {
				reviewID = strings.TrimSuffix(filepath.Base(r.Path), ".a-gate.json")
			}
		}
		cat, err := checkedCatalog(root)
		if err != nil {
			return nil, err
		}
		if oldSHA != p.GlossarySHA {
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
					if identity(struct {
						Profile                liveProfile
						ValidatedContextSHA256 string
					}{profile, digest(historical)}) != c.ContextIdentity {
						return nil, fmt.Errorf("original Tour context proof changed")
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
		if oldSHA == p.GlossarySHA {
			if err := verifyCarriedLocale(root, g, Locale{Schema: l.Schema, Locale: locale, Packages: []Completion{c}}); err != nil {
				return nil, err
			}
		}
		closureRef, err := ReferenceFile(root, LocalePath(locale))
		if err != nil {
			return nil, err
		}
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
				resolution := i18n.ResolveGlossaryCompatibility(root, locale, oldSHA, p.GlossarySHA, ctx.Scope, cat)
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
	for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
		docs, _, err := PackageDocuments(root, g, pkg)
		if err != nil {
			return nil, err
		}
		for _, d := range docs {
			if c, ok := completed[pkg]; ok {
				if err := checkPackageCompletion(root, g, locale, c); err != nil {
					return nil, err
				}
				_, files, err := CheckPackageClosure(root, c.Evidence[0])
				if err != nil {
					return nil, err
				}
				p.Work = append(p.Work, LanguageWork{ID: d.Path, Package: pkg, State: "carry", Reason: "exact current package closure + prior Surface", SourceSHA: d.SourceSHA, TargetSHA: digest(files[d.Path]), Evidence: c.Evidence})
				continue
			}
			p.Work = append(p.Work, LanguageWork{ID: d.Path, Package: pkg, State: "generation-required", Reason: "new/missing surface", SourceSHA: d.SourceSHA, Evidence: []Reference{}})
		}
	}
	sort.Slice(p.Work, func(i, j int) bool { return p.Work[i].ID < p.Work[j].ID })
	p.Identity = identity(*p)
	return p, nil
}
