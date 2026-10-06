package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// This identity binds actual public language bytes/source coverage. Registry,
// project and SEO code have their own semantic compatibility contracts and
// must not reintroduce whole-file freshness through this projection.
func currentTourLanguageContextSHA256(root, locale string, catalog *Catalog) (string, error) {
	if catalog == nil {
		return "", fmt.Errorf("surface context catalog required")
	}
	refs := map[string]string{}
	add := func(path string) error {
		data, err := readCompatibilityFile(root, path)
		if os.IsNotExist(err) {
			refs[path] = "missing"
			return nil
		}
		if err != nil {
			return err
		}
		refs[path] = sum(data)
		return nil
	}
	units, _, _, err := localeWorkflowUnits(catalog)
	if err != nil {
		return "", err
	}
	for _, unit := range units {
		if err := add(unit.SourcePath); err != nil {
			return "", err
		}
		path, err := canonicalTranslationUnitCandidatePath(locale, unit)
		if err != nil {
			return "", err
		}
		if err := add(path); err != nil {
			return "", err
		}
	}
	for _, path := range []string{"internal/tour/tour.go", "internal/tour/production.go", "_content/js/playground.js"} {
		if err := add(path); err != nil {
			return "", err
		}
	}
	for directory, extension := range map[string]string{"_content/tour/static/js": ".js", "_content/tour/template": ".tmpl", "_content/tour/static/partials": ".html"} {
		path, err := compatibilityPath(root, directory+"/.inventory", false)
		if os.IsNotExist(err) {
			refs[directory] = "missing"
			continue
		}
		if err != nil {
			return "", err
		}
		entries, err := os.ReadDir(filepath.Dir(path))
		if os.IsNotExist(err) {
			refs[directory] = "missing"
			continue
		}
		if err != nil {
			return "", err
		}
		refs[directory] = "directory"
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == extension {
				if !entry.Type().IsRegular() {
					return "", fmt.Errorf("unsafe surface context member")
				}
				if err := add(directory + "/" + entry.Name()); err != nil {
					return "", err
				}
			}
		}
	}
	ordered := []GlossaryArchiveReference{}
	for path, sha := range refs {
		ordered = append(ordered, GlossaryArchiveReference{path, sha})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	// Explicit missing identities preserve fail-closed changes from absence to
	// presence. A real compatibility assessment separately requires the complete
	// formal context inventory; missing input never proves compatible language.
	return sum(mustJSON(struct {
		Version string
		Inputs  []GlossaryArchiveReference
	}{"tour-language-context/v1", ordered})), nil
}

// Historical v1/v2/v3 has no source-context field. The immutable V2-A closure
// supplies the exact original refs and full package digest; without that proof
// a historic gate cannot cross glossary SHA on the basis of a new assessment.
func historicalCompletedTourLanguageContextCurrent(root, locale string, gate LocaleSurfaceReviewAGate, gateData []byte, catalog *Catalog) bool {
	if catalog == nil {
		var err error
		catalog, err = compatibilityCatalog(root)
		if err != nil {
			return false
		}
	}
	data, err := readCompatibilityFile(root, "locales/"+locale+"/content-scope.json")
	if err != nil {
		return false
	}
	var scope contentidentity.Locale
	if decodeStrictCourseSourceDescriptionReviewJSON(data, &scope) != nil || scope.Schema != contentidentity.LocaleSchema || scope.Locale != locale || len(scope.Packages) != 1 {
		return false
	}
	c := scope.Packages[0]
	if c.Package != "tour-v1" || c.State != "complete" || c.EvidenceKind != "legacy-tour-closure/v1" || c.EvidenceIdentity != sum(mustJSON(c.Evidence)) || !validSHA256(c.ContextIdentity) || !validSHA256(c.PackageIdentity) || !validSHA256(c.SourceIdentity) {
		return false
	}
	wantGate := "data/locale-surface-reviews/" + locale + "/" + gate.ReviewID + ".a-gate.json"
	found := false
	glossarySHA := ""
	for i, ref := range c.Evidence {
		if !validSHA256(ref.SHA256) || validateGenerationInstallPath(ref.Path) != nil || (i > 0 && c.Evidence[i-1].Path >= ref.Path) {
			return false
		}
		if ref.Path == glossaryReviewGlossaryPath(locale) {
			glossarySHA = ref.SHA256
			continue
		}
		bytes, err := readCompatibilityFile(root, ref.Path)
		if err != nil || sum(bytes) != ref.SHA256 {
			return false
		}
		if ref.Path == wantGate && ref.SHA256 == sum(gateData) {
			found = true
		}
	}
	if !found || glossarySHA == "" {
		return false
	}
	current, _, err := ExportLocaleSurfaceReviewPackage(root, locale, catalog)
	if err != nil {
		return false
	}
	original, err := HistoricalTourSurfacePackage(root, locale, gate.ReviewID, glossarySHA, current, catalog)
	if err != nil {
		return false
	}
	production, err := readCompatibilityFile(root, "production/identity.json")
	if err != nil {
		return false
	}
	var identity localeSurfaceReviewProductionIdentity
	if json.Unmarshal(production, &identity) != nil {
		return false
	}
	profiles := []localeSurfaceReviewProductionProfile{}
	for _, p := range identity.Locales {
		if p.Locale == locale {
			profiles = append(profiles, p)
		}
	}
	if len(profiles) != 1 {
		return false
	}
	p := profiles[0]
	if p.ProductionState != "live" {
		return false
	}
	// Field order/tags are the unchanged Site v2-A live profile wire projection.
	profile := struct {
		Locale    string `json:"locale"`
		State     string `json:"production_state"`
		Hostname  string `json:"production_hostname"`
		PublicURL string `json:"production_public_url"`
	}{p.Locale, p.ProductionState, p.ProductionHostname, p.ProductionPublicURL}
	return c.ContextIdentity == sum(mustJSON(struct {
		Profile                any
		ValidatedContextSHA256 string
	}{profile, sum(original)}))
}
