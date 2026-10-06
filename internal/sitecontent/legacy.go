package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

type liveProfile struct {
	Locale    string `json:"locale"`
	State     string `json:"production_state"`
	Hostname  string `json:"production_hostname"`
	PublicURL string `json:"production_public_url"`
}

// The legacy deployment document retains its own schema. This reader projects
// only deployment facts; it never adds content state to that authority.
func liveProfiles(root string) ([]liveProfile, error) {
	b, err := readRegular(root, "production/identity.json")
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := StrictJSON(b, &raw); err != nil {
		return nil, err
	}
	var schema string
	if err := json.Unmarshal(raw["schema"], &schema); err != nil || schema != "go-tour-i18n/production-identity/v1" {
		return nil, fmt.Errorf("unknown production identity schema")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(raw["locales"], &rows); err != nil {
		return nil, err
	}
	registry := map[string]string{}
	for _, l := range tour.LanguageRegistry() {
		if !l.Official {
			registry[l.Locale] = l.URL
		}
	}
	profiles := []liveProfile{}
	seen := map[string]bool{}
	for _, row := range rows {
		var p liveProfile
		if err := json.Unmarshal(row, &p); err != nil {
			return nil, err
		}
		if err := i18n.ValidateLocaleName(p.Locale); err != nil {
			return nil, err
		}
		if seen[p.Locale] || p.State != "live" || p.Hostname == "" || p.PublicURL != "https://"+p.Hostname+"/" || registry[p.Locale] != p.PublicURL {
			return nil, fmt.Errorf("legacy bootstrap requires unique live registry/deployment identity: %s", p.Locale)
		}
		seen[p.Locale] = true
		profiles = append(profiles, p)
	}
	if len(profiles) != 65 || len(registry) != 65 {
		return nil, fmt.Errorf("legacy bootstrap requires exactly 65 live community locales")
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Locale < profiles[j].Locale })
	return profiles, nil
}

func LiveLocales(root string) ([]string, error) {
	profiles, err := liveProfiles(root)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, p := range profiles {
		out = append(out, p.Locale)
	}
	return out, nil
}

func legacyLocale(root string, g *Global, catalog *i18n.Catalog, profile liveProfile) (*Locale, error) {
	locale := profile.Locale
	if err := i18n.RequireCurrentGlossaryReview(root, locale); err != nil {
		return nil, fmt.Errorf("%s glossary evidence: %w", locale, err)
	}
	gate, surfaceState, err := legacySurfaceReceipt(root, locale, catalog)
	if err != nil {
		return nil, err
	}
	// Reuse the existing formal read-only export validator: complete ready
	// canonical candidates, source/structure, UI, metadata and Course SEO.
	context, coverage, err := i18n.ExportLocaleSurfaceReviewPackage(root, locale, catalog)
	if err != nil {
		return nil, err
	}
	total, pages, _, err := i18n.LocaleWorkflowUnitCounts(catalog)
	if err != nil {
		return nil, err
	}
	if coverage.TranslationUnits != total || coverage.Pages != pages {
		return nil, fmt.Errorf("incomplete legacy Tour closure")
	}
	prefix := "locales/" + locale + "/"
	base := "data/locale-surface-reviews/" + locale + "/" + gate.ReviewID
	markdown, err := readRegular(root, base+".md")
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(markdown)) == 0 {
		return nil, fmt.Errorf("empty historical Markdown evidence")
	}
	paths := []string{base + ".a-gate.json", base + ".md", prefix + "locale.json", prefix + "status.tsv", prefix + "glossary.yaml", prefix + "article-metadata.json", prefix + "course-metadata.json", "internal/tour/ui/" + locale + ".json"}
	if _, err := os.Lstat(filepath.Join(root, base+".registry-baseline.json")); err == nil {
		paths = append(paths, base+".registry-baseline.json")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	statuses, err := i18n.ReadStatuses(filepath.Join(root, prefix+"status.tsv"))
	if err != nil {
		return nil, err
	}
	for _, s := range statuses {
		if s.State != "ready" {
			return nil, fmt.Errorf("non-ready legacy unit %s", s.UnitID)
		}
		paths = append(paths, s.CandidatePath)
	}
	qcPaths, err := legacyQCClosure(root, locale, catalog, statuses, gate.Inputs.GlossarySHA256)
	if err != nil {
		return nil, err
	}
	paths = append(paths, qcPaths...)
	refs, err := sortedReferences(root, paths)
	if err != nil {
		return nil, err
	}
	l := &Locale{Schema: LocaleSchema, Locale: locale, Packages: []Completion{}}
	for _, p := range g.Packages {
		if p.ID != "tour-v1" {
			continue
		}
		c := Completion{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity,
			Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: []string{"/tour/**"}, EvidenceKind: "legacy-tour-closure/v1",
			Evidence: refs, EvidenceIdentity: identity(refs), LegacySurfaceState: surfaceState}
		c.ContextIdentity = identity(struct {
			Profile                liveProfile
			ValidatedContextSHA256 string
		}{profile, digest(context)})
		l.Packages = append(l.Packages, c)
	}
	return l, ValidateLocale(g, *l)
}

// Read-only verification of existing machine finalization; never records QC,
// chooses language ratings, promotes, or rewrites a historical artifact.
type canonicalQCIdentity struct{ SourceSHA256, CandidateSHA256 string }

func legacyQCClosure(root, locale string, catalog *i18n.Catalog, statuses []i18n.Status, glossarySHA string) ([]string, error) {
	current := map[string]canonicalQCIdentity{}
	for _, s := range statuses {
		b, err := readRegular(root, s.CandidatePath)
		if err != nil {
			return nil, err
		}
		current[s.UnitID] = canonicalQCIdentity{s.SourceSHA256, digest(b)}
	}
	base := "data/quality-check-snapshots/" + locale
	entries, err := os.ReadDir(filepath.Join(root, base))
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := base + "/" + e.Name() + "/finalization.json"
		b, err := readRegular(root, p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var f i18n.QualityCheckFinalization
		if err := StrictJSON(b, &f); err != nil {
			return nil, err
		}
		normalized, err := finalizationWithPromotionEOF(root, f)
		if err != nil {
			return nil, err
		}
		if normalized.GlossarySHA256 != glossarySHA {
			compatible := true
			for _, u := range normalized.Units {
				r := i18n.ResolveGlossaryCompatibility(root, locale, normalized.GlossarySHA256, glossarySHA, "tu:"+u.UnitID, catalog)
				if r.Status != "compatible" {
					compatible = false
					break
				}
			}
			if !compatible {
				continue
			}
			normalized.GlossarySHA256 = glossarySHA
		}
		if !finalizationMatchesCanonical(normalized, locale, e.Name(), glossarySHA, current) {
			continue
		}
		verified, err := i18n.VerifyQualityCheckFinalization(root, catalog, locale, e.Name())
		if err != nil {
			return nil, fmt.Errorf("legacy QC finalization %s: %w", p, err)
		}
		paths := []string{p, verified.SnapshotManifestPath}
		for _, q := range verified.QCResults {
			paths = append(paths, q.Path, "data/quality-check-snapshots/"+locale+"/"+q.SnapshotID+"/manifest.json")
		}
		for _, u := range verified.Units {
			paths = append(paths, u.Snapshot.CandidatePath, u.Snapshot.ValidationPath)
		}
		return paths, nil
	}
	return nil, fmt.Errorf("%s: no full A-only QC finalization matches all current canonical candidate/source bytes", locale)
}

// Existing promotion deliberately gives text artifacts exactly one final LF.
// Compare that projection only; preserve the receipt's original raw SHA and
// verify its actual selected bytes before any normalization. No other byte is
// allowed to differ. This does not mutate or fabricate a Snapshot identity.
func finalizationWithPromotionEOF(root string, f i18n.QualityCheckFinalization) (i18n.QualityCheckFinalization, error) {
	projected := f
	projected.Units = append([]i18n.QualityCheckFinalizationUnit{}, f.Units...)
	for i, u := range f.Units {
		b, err := readRegular(root, u.Snapshot.CandidatePath)
		if err != nil {
			return f, err
		}
		if digest(b) != u.Snapshot.CandidateSHA256 {
			return f, fmt.Errorf("historical selected candidate changed: %s", u.UnitID)
		}
		normalized := append(append([]byte{}, bytes.TrimRight(b, "\n")...), '\n')
		projected.Units[i].Snapshot.CandidateSHA256 = digest(normalized)
	}
	return projected, nil
}

func finalizationMatchesCanonical(f i18n.QualityCheckFinalization, locale, snapshot, glossary string, current map[string]canonicalQCIdentity) bool {
	if f.SchemaVersion != i18n.QualityCheckFinalizationSchemaVersion || f.EvidenceType != i18n.QualityCheckFinalizationEvidenceType || f.Locale != locale || f.SnapshotID != snapshot || f.GlossarySHA256 != glossary || len(f.Units) != len(current) || len(current) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, u := range f.Units {
		c, ok := current[u.UnitID]
		if !ok || seen[u.UnitID] || u.Rating != "A" || u.Snapshot.UnitID != u.UnitID || u.Snapshot.SourceSHA256 != c.SourceSHA256 || u.Snapshot.CandidateSHA256 != c.CandidateSHA256 {
			return false
		}
		seen[u.UnitID] = true
	}
	return true
}

// A historical completion is not a current Surface A gate or permission for
// new publication. V2-A migrates the trusted live Site v1 achievement without
// rewriting old receipts or treating unrelated shared-config bytes as a new
// language review. Locale-owned language artifacts must still match exactly.
func legacySurfaceReceipt(root, locale string, catalog *i18n.Catalog) (i18n.LocaleSurfaceReviewAGate, string, error) {
	if gate, err := i18n.CurrentLocaleSurfaceReviewAGate(root, locale, catalog); err == nil {
		return gate, "current", nil
	}
	current, err := i18n.CurrentLocaleSurfaceReviewAInputs(root, locale, catalog)
	if err != nil {
		return i18n.LocaleSurfaceReviewAGate{}, "", err
	}
	dir := "data/locale-surface-reviews/" + locale
	entries, err := os.ReadDir(filepath.Join(root, dir))
	if err != nil {
		return i18n.LocaleSurfaceReviewAGate{}, "", err
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".a-gate.json") {
			continue
		}
		b, err := readRegular(root, dir+"/"+e.Name())
		if err != nil {
			return i18n.LocaleSurfaceReviewAGate{}, "", err
		}
		var g i18n.LocaleSurfaceReviewAGate
		if err := StrictJSON(b, &g); err != nil {
			return g, "", err
		}
		if g.Locale != locale || g.ReviewID+".a-gate.json" != e.Name() || g.Stage != "locale-level-language-quality-review" || g.Decision != "passed" || g.Reviewer == "" || g.SchemaVersion < 1 || g.SchemaVersion > 4 {
			return g, "", fmt.Errorf("invalid historical Surface receipt %s", e.Name())
		}
		a, c := g.Inputs, current
		if (g.SchemaVersion == 4 || !historicalLanguageInputsMatch(a, c, g.SchemaVersion)) && !i18n.HistoricalLocaleSurfaceLanguageInputsCompatible(root, locale, g, b, current, catalog) {
			continue
		}
		md, err := readRegular(root, dir+"/"+g.ReviewID+".md")
		if err != nil {
			return g, "", err
		}
		if len(bytes.TrimSpace(md)) == 0 {
			return g, "", fmt.Errorf("missing historical Markdown evidence")
		}
		return g, "historical-verified", nil
	}
	return i18n.LocaleSurfaceReviewAGate{}, "", fmt.Errorf("%s has no historical passed Surface receipt matching current locale-owned language/source artifacts", locale)
}

func historicalLanguageInputsMatch(a, c i18n.LocaleSurfaceReviewAInputs, schema int) bool {
	return a.UILocaleSHA256 == c.UILocaleSHA256 && a.GlossarySHA256 == c.GlossarySHA256 && a.ArticleMetadataSHA256 == c.ArticleMetadataSHA256 && a.CourseMetadataSHA256 == c.CourseMetadataSHA256 && a.CatalogSourceSHA256 == c.CatalogSourceSHA256 && a.CourseSourceDescriptionsSHA256 == c.CourseSourceDescriptionsSHA256 && a.CourseSourceDescriptionReviewSHA256 == c.CourseSourceDescriptionReviewSHA256 && a.ProjectConfigSHA256 == c.ProjectConfigSHA256 && a.SEOConfigSHA256 == c.SEOConfigSHA256 && (schema < 2 || a.ProductionPublicIdentitySHA256 == c.ProductionPublicIdentitySHA256)
}

func checkedCatalog(root string) (*i18n.Catalog, error) {
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return nil, err
	}
	if err := i18n.CheckCatalogFiles(root, current); err != nil {
		return nil, err
	}
	c, err := i18n.ReadCatalog(root)
	if err != nil {
		return nil, err
	}
	if err := i18n.HydrateCatalogSources(c, current); err != nil {
		return nil, err
	}
	return c, nil
}

func CheckLocale(root string, g *Global, locale string) (*Locale, error) {
	if err := i18n.ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	b, err := readRegular(root, LocalePath(locale))
	if err != nil {
		return nil, err
	}
	var got Locale
	if err := StrictJSON(b, &got); err != nil {
		return nil, err
	}
	if got.Locale != locale {
		return nil, fmt.Errorf("locale scope identity mismatch")
	}
	if err := ValidateLocale(g, got); err != nil {
		return nil, err
	}
	profiles, err := liveProfiles(root)
	if err != nil {
		return nil, err
	}
	catalog, err := checkedCatalog(root)
	if err != nil {
		return nil, err
	}
	for _, p := range profiles {
		if p.Locale == locale {
			if len(got.Packages) == 0 {
				return &got, nil // no completion claim, therefore no evidence to accept
			}
			want, err := legacyLocale(root, g, catalog, p)
			if err != nil {
				return nil, err
			}
			if !reflect.DeepEqual(got, *want) {
				if err := compatibleLegacyCompletion(root, g, catalog, p, got, *want); err != nil {
					return nil, fmt.Errorf("%s content completion STALE: %w", locale, err)
				}
			}
			return &got, nil
		}
	}
	return nil, fmt.Errorf("locale is not a verified legacy live locale")
}

func compatibleLegacyCompletion(root string, g *Global, catalog *i18n.Catalog, profile liveProfile, recorded, current Locale) error {
	if len(recorded.Packages) != 1 || len(current.Packages) != 1 {
		return fmt.Errorf("legacy package exact-set changed")
	}
	old, next := recorded.Packages[0], current.Packages[0]
	refs := map[string]Reference{}
	for _, r := range next.Evidence {
		refs[r.Path] = r
	}
	oldGlossary := ""
	reviewID := ""
	for _, r := range old.Evidence {
		got, ok := refs[r.Path]
		if !ok {
			return fmt.Errorf("historical evidence set changed")
		}
		if r.Path == "locales/"+profile.Locale+"/glossary.yaml" {
			oldGlossary = r.SHA256
		} else if r.SHA256 != got.SHA256 {
			return fmt.Errorf("historical language evidence bytes changed: %s", r.Path)
		}
		if strings.HasSuffix(r.Path, ".a-gate.json") {
			reviewID = strings.TrimSuffix(filepath.Base(r.Path), ".a-gate.json")
		}
		delete(refs, r.Path)
	}
	if len(refs) != 0 || oldGlossary == "" || reviewID == "" {
		return fmt.Errorf("historical evidence exact-set changed")
	}
	context, _, err := i18n.ExportLocaleSurfaceReviewPackage(root, profile.Locale, catalog)
	if err != nil {
		return err
	}
	historical, err := i18n.HistoricalTourSurfacePackage(root, profile.Locale, reviewID, oldGlossary, context, catalog)
	if err != nil {
		return err
	}
	restoredIdentity := identity(struct {
		Profile                liveProfile
		ValidatedContextSHA256 string
	}{profile, digest(historical)})
	if restoredIdentity != old.ContextIdentity {
		return fmt.Errorf("historical full context digest changed")
	}
	next.ContextIdentity = restoredIdentity
	next.Evidence = old.Evidence
	next.EvidenceIdentity = old.EvidenceIdentity
	// The same exact original passed gate can become compatible current language
	// evidence; retain the historical completion's original state in this view.
	next.LegacySurfaceState = old.LegacySurfaceState
	if !reflect.DeepEqual(old, next) {
		return fmt.Errorf("historical package identity changed")
	}
	return ValidateLocale(g, recorded)
}

// BootstrapTour preflights every closure and output before creating any state.
// Existing identical state is idempotent; existing different state is never overwritten.
func LegacyPlan(root string, g *Global) ([]*Locale, error) {
	profiles, err := liveProfiles(root)
	if err != nil {
		return nil, err
	}
	catalog, err := checkedCatalog(root)
	if err != nil {
		return nil, err
	}
	states := []*Locale{}
	for _, p := range profiles {
		l, err := legacyLocale(root, g, catalog, p)
		if err != nil {
			return nil, fmt.Errorf("bootstrap %s: %w", p.Locale, err)
		}
		states = append(states, l)
	}
	return states, nil
}

func BootstrapTour(root string, g *Global) (int, error) {
	states, err := LegacyPlan(root, g)
	if err != nil {
		return 0, err
	}
	type output struct {
		path string
		data []byte
	}
	pending := []output{}
	for _, l := range states {
		b, err := Encode(l)
		if err != nil {
			return 0, err
		}
		target := LocalePath(l.Locale)
		old, err := readRegular(root, target)
		if err == nil {
			if string(old) != string(b) {
				return 0, fmt.Errorf("refuse to overwrite %s", target)
			}
			continue
		}
		if !os.IsNotExist(err) {
			return 0, err
		}
		pending = append(pending, output{target, b})
	}
	for _, p := range pending {
		if err := WriteNew(root, p.path, p.data); err != nil {
			return 0, err
		}
	}
	return len(states), nil
}
