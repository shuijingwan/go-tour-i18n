package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"os"
	"reflect"
	"sort"
	"strings"
)

const IntegratedSurfaceSchema = "go-learning/integrated-locale-surface/v1"

type SurfaceAsset struct {
	ID        string      `json:"id"`
	Package   string      `json:"package"`
	State     string      `json:"state"`
	SourceSHA string      `json:"source_sha256"`
	TargetSHA string      `json:"target_sha256"`
	Evidence  []Reference `json:"evidence"`
}
type IntegratedSurfaceScope struct {
	Schema           string         `json:"schema"`
	Locale           string         `json:"locale"`
	GlossarySHA      string         `json:"glossary_sha256"`
	UnifiedReview    Reference      `json:"unified_glossary_review"`
	ExistingCoverage Locale         `json:"existing_coverage"`
	Closures         []Reference    `json:"package_closures"`
	Carried          []SurfaceAsset `json:"carried"`
	Reviewed         []SurfaceAsset `json:"review_required"`
	Runtime          []Reference    `json:"runtime_public_context"`
	Routes           []string       `json:"canonical_routes"`
	Identity         string         `json:"identity_sha256"`
}

// This plan never presents mechanical carry as a fresh language rating.
func BuildIntegratedSurfaceScope(root, locale string, closures []Reference) (*IntegratedSurfaceScope, error) {
	if err := requireUnifiedReview(root, locale); err != nil {
		return nil, err
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	state, err := readLocaleAuthority(root, g, locale)
	if err != nil {
		return nil, err
	}
	pending := map[string]bool{}
	for _, ref := range closures {
		c, _, err := CheckPackageClosure(root, ref)
		if err != nil {
			return nil, err
		}
		pending[c.Package] = true
	}
	l := &Locale{Schema: state.Schema, Locale: state.Locale, Packages: []Completion{}}
	for _, c := range state.Packages {
		if !pending[c.Package] {
			l.Packages = append(l.Packages, c)
		}
	}
	other := *l
	other.Packages = []Completion{}
	for _, c := range l.Packages {
		if c.Package != "tour-v1" {
			other.Packages = append(other.Packages, c)
		}
	}
	if err := verifyCarriedLocale(root, g, other); err != nil {
		return nil, err
	}
	glossary, err := readRegular(root, "locales/"+locale+"/glossary.yaml")
	if err != nil {
		return nil, err
	}
	review, err := i18n.RequireUnifiedGlossaryReview(root, locale)
	if err != nil {
		return nil, err
	}
	s := &IntegratedSurfaceScope{Schema: IntegratedSurfaceSchema, Locale: locale, GlossarySHA: digest(glossary), UnifiedReview: Reference{review.Path, review.SHA256}, ExistingCoverage: *l, Closures: closures, Carried: []SurfaceAsset{}, Reviewed: []SurfaceAsset{}, Runtime: []Reference{}, Routes: []string{}}
	seen := map[string]bool{}
	for _, c := range l.Packages {
		seen[c.Package] = true
		s.Routes = append(s.Routes, c.Routes...)
		if c.Package == "tour-v1" {
			plan, err := PlanLocaleLanguage(root, locale)
			if err != nil {
				return nil, err
			}
			for _, w := range plan.Work {
				if w.Package != "tour-v1" {
					continue
				}
				a := SurfaceAsset{ID: w.ID, Package: w.Package, State: w.State, SourceSHA: w.SourceSHA, TargetSHA: w.TargetSHA, Evidence: w.Evidence}
				if w.State == "carry" {
					s.Carried = append(s.Carried, a)
				} else {
					s.Reviewed = append(s.Reviewed, a)
				}
			}
			continue
		}

		s.Carried = append(s.Carried, SurfaceAsset{ID: c.Package, Package: c.Package, State: "carry", SourceSHA: c.SourceIdentity, TargetSHA: c.ContextIdentity, Evidence: c.Evidence})
	}
	for _, ref := range closures {
		c, _, err := CheckPackageClosure(root, ref)
		if err != nil {
			return nil, err
		}
		if c.Locale != locale || seen[c.Package] {
			return nil, fmt.Errorf("duplicate/wrong prospective coverage")
		}
		seen[c.Package] = true
		p, err := packageByID(g, c.Package)
		if err != nil {
			return nil, err
		}
		s.Routes = append(s.Routes, p.Routes...)
		for _, f := range c.Files {
			s.Reviewed = append(s.Reviewed, SurfaceAsset{ID: f.Path, Package: c.Package, State: "review-required", SourceSHA: p.SourceIdentity, TargetSHA: f.SHA256, Evidence: []Reference{ref}})
		}
	}
	for _, path := range []string{"production/identity.json", "internal/tour/project.go", "internal/tour/languages.go", "internal/tourpolicy/policy.go", GlobalPath} {
		r, err := ReferenceFile(root, path)
		if err != nil {
			return nil, err
		}
		s.Runtime = append(s.Runtime, r)
	}
	sort.Strings(s.Routes)
	sort.Slice(s.Reviewed, func(i, j int) bool { return s.Reviewed[i].ID < s.Reviewed[j].ID })
	s.Identity = identity(*s)
	return s, nil
}

type IntegratedSurfaceReceipt struct {
	Schema        string                 `json:"schema"`
	Scope         IntegratedSurfaceScope `json:"scope"`
	ReviewID      string                 `json:"review_id"`
	Reviewer      string                 `json:"reviewer_session"`
	Decision      string                 `json:"decision"`
	Findings      []string               `json:"findings"`
	ExactFindings []SurfaceFinding       `json:"exact_findings"`
	Evidence      Reference              `json:"reviewer_markdown"`
	Identity      string                 `json:"identity_sha256"`
}

func RecordIntegratedSurface(root string, b []byte) (*IntegratedSurfaceReceipt, string, error) {
	var r IntegratedSurfaceReceipt
	if err := StrictJSON(b, &r); err != nil {
		return nil, "", err
	}
	if r.Identity != "" || r.Schema != IntegratedSurfaceSchema || !component(r.ReviewID) || !component(r.Reviewer) {
		return nil, "", fmt.Errorf("invalid integrated review identity")
	}
	seenFindings := map[string]bool{}
	allowedFindings := map[string]bool{}
	for _, a := range r.Scope.Reviewed {
		allowedFindings[a.ID] = true
	}
	for _, f := range r.ExactFindings {
		if f.Finding == "" || seenFindings[f.ID] || !allowedFindings[f.ID] {
			return nil, "", fmt.Errorf("invalid exact Surface finding scope")
		}
		seenFindings[f.ID] = true
	}
	if r.Decision == "passed" && len(r.ExactFindings) != 0 {
		return nil, "", fmt.Errorf("passed Surface has unresolved exact finding")
	}
	want, err := BuildIntegratedSurfaceScope(root, r.Scope.Locale, r.Scope.Closures)
	if err != nil {
		return nil, "", err
	}
	if !reflect.DeepEqual(r.Scope, *want) {
		return nil, "", fmt.Errorf("integrated scope stale")
	}
	if (r.Decision != "passed" && r.Decision != "failed") || (r.Decision == "passed" && len(r.Findings) > 0) || (r.Decision == "failed" && len(r.Findings) == 0) {
		return nil, "", fmt.Errorf("invalid Surface decision")
	}
	for _, a := range r.Scope.Reviewed {
		if a.Package == "tour-v1" && strings.HasPrefix(a.ID, "tu:") && a.State != "carry" {
			return nil, "", fmt.Errorf("affected Tour TU requires formal revision + independent re-QC before integrated Surface PASS: %s", a.ID)
		}
	}
	if err := requireIndependentSession(root, r.Scope.Locale, r.Reviewer, "reviewer"); err != nil {
		return nil, "", err
	}
	e, err := readRegular(root, r.Evidence.Path)
	if err != nil || digest(e) != r.Evidence.SHA256 {
		return nil, "", fmt.Errorf("Surface evidence hash mismatch")
	}
	if err := textBytes(e); err != nil {
		return nil, "", err
	}
	r.Identity = identity(r)
	p := "data/locale-language/" + r.Scope.Locale + "/surface/" + r.ReviewID + ".json"
	return &r, p, saveImmutable(root, p, &r)
}
func RequireIntegratedSurface(root, locale string, closure Reference) (Reference, error) {
	dir := "data/locale-language/" + locale + "/surface"
	entries, err := os.ReadDir(pathJoin(root, dir))
	if err != nil {
		return Reference{}, err
	}
	matches := []Reference{}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".json") {
			return Reference{}, fmt.Errorf("unsafe Surface receipt")
		}
		ref, err := ReferenceFile(root, dir+"/"+entry.Name())
		if err != nil {
			return Reference{}, err
		}
		var r IntegratedSurfaceReceipt
		if err := readReference(root, ref, &r); err != nil {
			return Reference{}, err
		}
		copy := r
		copy.Identity = ""
		if r.Schema != IntegratedSurfaceSchema || r.Identity != identity(copy) || entry.Name() != r.ReviewID+".json" {
			return Reference{}, fmt.Errorf("Surface receipt identity mismatch")
		}
		included := false
		for _, c := range r.Scope.Closures {
			if c == closure {
				included = true
			}
		}
		if !included {
			continue
		}
		want, err := BuildIntegratedSurfaceScope(root, locale, r.Scope.Closures)
		if err != nil {
			return Reference{}, err
		}
		if !reflect.DeepEqual(r.Scope, *want) {
			return Reference{}, fmt.Errorf("integrated gate stale")
		}
		b, err := readRegular(root, r.Evidence.Path)
		if err != nil || digest(b) != r.Evidence.SHA256 {
			return Reference{}, fmt.Errorf("Surface historical evidence changed")
		}
		if r.Decision == "passed" && len(r.Findings) == 0 {
			matches = append(matches, ref)
		}
	}
	if len(matches) != 1 {
		return Reference{}, fmt.Errorf("integrated current Surface PASS missing/ambiguous")
	}
	return matches[0], nil
}
func verifyCarriedLocale(root string, g *Global, l Locale) error {
	legacy := Locale{Schema: l.Schema, Locale: l.Locale, Packages: []Completion{}}
	for _, c := range l.Packages {
		if c.Package == "tour-v1" {
			legacy.Packages = append(legacy.Packages, c)
		} else {
			if err := checkPackageCompletion(root, g, l.Locale, c); err != nil {
				return err
			}
		}
	}
	if len(legacy.Packages) == 0 {
		return nil
	}
	profiles, err := liveProfiles(root)
	if err != nil {
		return err
	}
	cat, err := checkedCatalog(root)
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if p.Locale == l.Locale {
			want, err := legacyLocale(root, g, cat, p)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(legacy, *want) {
				return compatibleLegacyCompletion(root, g, cat, p, legacy, *want)
			}
			return nil
		}
	}
	return fmt.Errorf("unproven carried Tour identity")
}

// A previous integrated judgment remains immutable evidence for its own exact
// language closure. A new expansion needs a new complete integrated gate, but
// cannot make an unchanged package's previous language judgment disappear.
func validateHistoricalIntegratedSurface(root, locale string, ref, closure Reference) error {
	var r IntegratedSurfaceReceipt
	if err := readReference(root, ref, &r); err != nil {
		return err
	}
	copy := r
	copy.Identity = ""
	s := r.Scope
	s.Identity = ""
	if r.Schema != IntegratedSurfaceSchema || r.Identity != identity(copy) || r.Scope.Identity != identity(s) || r.Scope.Locale != locale || r.Decision != "passed" || len(r.Findings) != 0 || ref.Path != "data/locale-language/"+locale+"/surface/"+r.ReviewID+".json" {
		return fmt.Errorf("invalid historical integrated Surface PASS")
	}
	included := false
	for _, c := range r.Scope.Closures {
		if c == closure {
			included = true
		}
	}
	if !included {
		return fmt.Errorf("historical Surface does not bind package closure")
	}
	b, err := readRegular(root, r.Evidence.Path)
	if err != nil || digest(b) != r.Evidence.SHA256 {
		return fmt.Errorf("historical Surface Markdown changed")
	}
	return requireIndependentSession(root, locale, r.Reviewer, "reviewer")
}
