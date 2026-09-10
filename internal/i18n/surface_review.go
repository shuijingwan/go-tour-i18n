package i18n

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const (
	localeSurfaceReviewASchemaVersionV1 = 1
	localeSurfaceReviewASchemaVersion   = 2
)
const localeSurfaceReviewAStage = "locale-level-language-quality-review"

// LocaleSurfaceReviewAGate is the machine-readable receipt recorded after the
// human locale-level language quality review. It deliberately does not replace
// the paired Markdown review evidence.
type LocaleSurfaceReviewAGate struct {
	SchemaVersion int                        `json:"schema_version"`
	Locale        string                     `json:"locale"`
	ReviewID      string                     `json:"review_id"`
	Stage         string                     `json:"stage"`
	Decision      string                     `json:"decision"`
	Reviewer      string                     `json:"reviewer"`
	Inputs        LocaleSurfaceReviewAInputs `json:"inputs"`
}

// LocaleSurfaceReviewAInputs are hashes calculated by this program, never
// supplied by a reviewer. The config hashes cover the build-time language
// registry, locale profile, public project copy, and SEO origin behavior.
type LocaleSurfaceReviewAInputs struct {
	UIEnglishSHA256       string `json:"ui_english_sha256"`
	UILocaleSHA256        string `json:"ui_locale_sha256"`
	GlossarySHA256        string `json:"glossary_sha256"`
	ArticleMetadataSHA256 string `json:"article_metadata_sha256"`
	CourseMetadataSHA256  string `json:"course_metadata_sha256"`
	CatalogSourceSHA256   string `json:"catalog_source_sha256"`
	LanguagesConfigSHA256 string `json:"languages_config_sha256"`
	ProjectConfigSHA256   string `json:"project_config_sha256"`
	SEOConfigSHA256       string `json:"seo_config_sha256"`
	// ProductionIdentitySHA256 is the v1 whole-file identity input. It remains
	// present so historic receipts retain their original freshness semantics.
	ProductionIdentitySHA256 string `json:"production_identity_sha256,omitempty"`
	// ProductionPublicIdentitySHA256 is the v2 target-locale projection of the
	// public identity used alongside the build-time language registry.
	ProductionPublicIdentitySHA256 string `json:"production_public_identity_sha256,omitempty"`
}

type localeSurfaceReviewProductionIdentity struct {
	Locales []localeSurfaceReviewProductionProfile `json:"locales"`
}

type localeSurfaceReviewProductionProfile struct {
	Locale              string `json:"locale"`
	ProductionHostname  string `json:"production_hostname"`
	ProductionPublicURL string `json:"production_public_url"`
}

type localeSurfaceReviewPublicIdentity struct {
	Locale              string `json:"locale"`
	ProductionHostname  string `json:"production_hostname"`
	ProductionPublicURL string `json:"production_public_url"`
}

var reviewIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func LocaleSurfaceReviewAGatePath(root, locale, reviewID string) (string, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return "", err
	}
	if !reviewIDPattern.MatchString(reviewID) {
		return "", fmt.Errorf("invalid review_id %q", reviewID)
	}
	return filepath.Join(root, "data", "locale-surface-reviews", locale, reviewID+".a-gate.json"), nil
}

func CurrentLocaleSurfaceReviewAInputs(root, locale string, catalog *Catalog) (LocaleSurfaceReviewAInputs, error) {
	return currentLocaleSurfaceReviewAInputs(root, locale, catalog, localeSurfaceReviewASchemaVersion)
}

func currentLocaleSurfaceReviewAInputs(root, locale string, catalog *Catalog, schemaVersion int) (LocaleSurfaceReviewAInputs, error) {
	if catalog == nil {
		return LocaleSurfaceReviewAInputs{}, fmt.Errorf("Locale Surface Review A requires a catalog")
	}
	if err := ValidateLocaleName(locale); err != nil {
		return LocaleSurfaceReviewAInputs{}, err
	}
	hashFile := func(path string) (string, error) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return "", fmt.Errorf("read Locale Surface Review A input %s: %w", path, err)
		}
		return hashBytes(data), nil
	}
	uiEN, err := hashFile("internal/tour/ui/en.json")
	if err != nil {
		return LocaleSurfaceReviewAInputs{}, err
	}
	uiLocale, err := hashFile("internal/tour/ui/" + locale + ".json")
	if err != nil {
		return LocaleSurfaceReviewAInputs{}, err
	}
	glossary, err := hashFile("locales/" + locale + "/glossary.yaml")
	if err != nil {
		return LocaleSurfaceReviewAInputs{}, err
	}
	article, err := hashFile("locales/" + locale + "/article-metadata.json")
	if err != nil {
		return LocaleSurfaceReviewAInputs{}, err
	}
	course, err := hashFile("locales/" + locale + "/course-metadata.json")
	if err != nil {
		return LocaleSurfaceReviewAInputs{}, err
	}
	languages, err := hashFile("internal/tour/languages.go")
	if err != nil {
		return LocaleSurfaceReviewAInputs{}, err
	}
	project, err := hashFile("internal/tour/project.go")
	if err != nil {
		return LocaleSurfaceReviewAInputs{}, err
	}
	seo, err := hashFile("internal/tour/seo.go")
	if err != nil {
		return LocaleSurfaceReviewAInputs{}, err
	}
	encoded, err := json.Marshal(catalog)
	if err != nil {
		return LocaleSurfaceReviewAInputs{}, fmt.Errorf("encode current catalog/source identity: %w", err)
	}
	inputs := LocaleSurfaceReviewAInputs{
		UIEnglishSHA256:       uiEN,
		UILocaleSHA256:        uiLocale,
		GlossarySHA256:        glossary,
		ArticleMetadataSHA256: article,
		CourseMetadataSHA256:  course,
		CatalogSourceSHA256:   hashBytes(encoded),
		LanguagesConfigSHA256: languages,
		ProjectConfigSHA256:   project,
		SEOConfigSHA256:       seo,
	}
	switch schemaVersion {
	case localeSurfaceReviewASchemaVersionV1:
		productionIdentity, err := hashFile("production/identity.json")
		if err != nil {
			return LocaleSurfaceReviewAInputs{}, err
		}
		inputs.ProductionIdentitySHA256 = productionIdentity
	case localeSurfaceReviewASchemaVersion:
		productionPublicIdentity, err := localeSurfaceReviewPublicIdentityHash(root, locale)
		if err != nil {
			return LocaleSurfaceReviewAInputs{}, err
		}
		inputs.ProductionPublicIdentitySHA256 = productionPublicIdentity
	default:
		return LocaleSurfaceReviewAInputs{}, fmt.Errorf("unsupported Locale Surface Review A gate schema version %d", schemaVersion)
	}
	return inputs, nil
}

func localeSurfaceReviewPublicIdentityHash(root, locale string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	if err != nil {
		return "", fmt.Errorf("read Locale Surface Review A production identity: %w", err)
	}
	var identity localeSurfaceReviewProductionIdentity
	if err := json.Unmarshal(data, &identity); err != nil {
		return "", fmt.Errorf("parse Locale Surface Review A production identity: %w", err)
	}
	profiles := make([]localeSurfaceReviewProductionProfile, 0, 1)
	for _, profile := range identity.Locales {
		if profile.Locale == locale {
			profiles = append(profiles, profile)
		}
	}
	if len(profiles) != 1 {
		return "", fmt.Errorf("Locale Surface Review A production identity requires exactly one profile for %s", locale)
	}
	profile := profiles[0]
	if profile.Locale == "" || profile.ProductionHostname == "" || profile.ProductionPublicURL == "" {
		return "", fmt.Errorf("Locale Surface Review A production identity profile for %s is missing public identity fields", locale)
	}
	encoded, err := json.Marshal(localeSurfaceReviewPublicIdentity{
		Locale:              profile.Locale,
		ProductionHostname:  profile.ProductionHostname,
		ProductionPublicURL: profile.ProductionPublicURL,
	})
	if err != nil {
		return "", fmt.Errorf("encode Locale Surface Review A public identity: %w", err)
	}
	return hashBytes(encoded), nil
}

func RecordLocaleSurfaceReviewA(root, locale, reviewID, reviewer string, catalog *Catalog) (*LocaleSurfaceReviewAGate, string, error) {
	if reviewer == "" {
		return nil, "", fmt.Errorf("--reviewer is required")
	}
	path, err := LocaleSurfaceReviewAGatePath(root, locale, reviewID)
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Lstat(path); err == nil {
		return nil, "", fmt.Errorf("Locale Surface Review A gate already exists: %s", filepath.ToSlash(path))
	} else if !os.IsNotExist(err) {
		return nil, "", err
	}
	inputs, err := CurrentLocaleSurfaceReviewAInputs(root, locale, catalog)
	if err != nil {
		return nil, "", err
	}
	gate := &LocaleSurfaceReviewAGate{SchemaVersion: localeSurfaceReviewASchemaVersion, Locale: locale, ReviewID: reviewID, Stage: localeSurfaceReviewAStage, Decision: "passed", Reviewer: reviewer, Inputs: inputs}
	data, err := json.MarshalIndent(gate, "", "  ")
	if err != nil {
		return nil, "", err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, "", err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, "", err
	}
	return gate, path, nil
}

// RequireCurrentLocaleSurfaceReviewA fails closed before a complete locale
// preview. It never interprets human Markdown evidence.
func RequireCurrentLocaleSurfaceReviewA(root, locale string, catalog *Catalog) error {
	_, err := CurrentLocaleSurfaceReviewAGate(root, locale, catalog)
	return err
}

// CurrentLocaleSurfaceReviewAGate returns the current passed A gate. It is the
// single freshness lookup used by consumers that also need its review identity.
func CurrentLocaleSurfaceReviewAGate(root, locale string, catalog *Catalog) (LocaleSurfaceReviewAGate, error) {
	gates, err := currentLocaleSurfaceReviewAGates(root, locale, catalog)
	if err != nil {
		return LocaleSurfaceReviewAGate{}, err
	}
	return gates[0], nil
}

// RequireUniqueCurrentLocaleSurfaceReviewAGate requires exactly one current A
// gate where an operation must bind its evidence to an unambiguous review ID.
func RequireUniqueCurrentLocaleSurfaceReviewAGate(root, locale string, catalog *Catalog) (LocaleSurfaceReviewAGate, error) {
	gates, err := currentLocaleSurfaceReviewAGates(root, locale, catalog)
	if err != nil {
		return LocaleSurfaceReviewAGate{}, err
	}
	if len(gates) != 1 {
		return LocaleSurfaceReviewAGate{}, fmt.Errorf("Locale Surface Review A gate is ambiguous for %s: found %d current gates; retain exactly one current review before first-production", locale, len(gates))
	}
	return gates[0], nil
}

// RequireCurrentLocaleSurfaceReviewAByReviewID verifies the named gate itself
// is current; it never accepts a different current gate for the same locale.
func RequireCurrentLocaleSurfaceReviewAByReviewID(root, locale, reviewID string, catalog *Catalog) (LocaleSurfaceReviewAGate, error) {
	path, err := LocaleSurfaceReviewAGatePath(root, locale, reviewID)
	if err != nil {
		return LocaleSurfaceReviewAGate{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return LocaleSurfaceReviewAGate{}, fmt.Errorf("Locale Surface Review A gate missing for %s review_id=%s", locale, reviewID)
	}
	if err != nil {
		return LocaleSurfaceReviewAGate{}, fmt.Errorf("read Locale Surface Review A gate %s: %w", reviewID, err)
	}
	var gate LocaleSurfaceReviewAGate
	if json.Unmarshal(data, &gate) != nil {
		return LocaleSurfaceReviewAGate{}, fmt.Errorf("language review evidence/gate stale: malformed Locale Surface Review A gate %s", filepath.Base(path))
	}
	if err := validateLocaleSurfaceReviewAGate(gate, locale); err != nil {
		return LocaleSurfaceReviewAGate{}, err
	}
	current, err := currentLocaleSurfaceReviewAInputs(root, locale, catalog, gate.SchemaVersion)
	if err != nil {
		return LocaleSurfaceReviewAGate{}, err
	}
	if gate.Inputs != current {
		return LocaleSurfaceReviewAGate{}, fmt.Errorf("language review evidence/gate stale for %s review_id=%s; complete Locale Surface Review A again and record the current A gate", locale, reviewID)
	}
	return gate, nil
}

func currentLocaleSurfaceReviewAGates(root, locale string, catalog *Catalog) ([]LocaleSurfaceReviewAGate, error) {
	directory := filepath.Join(root, "data", "locale-surface-reviews", locale)
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Locale Surface Review A gate missing for %s; complete Locale Surface Review A and record the current A gate", locale)
	}
	if err != nil {
		return nil, fmt.Errorf("read Locale Surface Review A gates: %w", err)
	}
	var currentGates []LocaleSurfaceReviewAGate
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || !regexp.MustCompile(`\.a-gate\.json$`).MatchString(entry.Name()) {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(directory, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("language review evidence/gate stale: read gate %s: %w", entry.Name(), readErr)
		}
		var gate LocaleSurfaceReviewAGate
		if json.Unmarshal(data, &gate) != nil {
			return nil, fmt.Errorf("language review evidence/gate stale: malformed Locale Surface Review A gate %s", entry.Name())
		}
		if err := validateLocaleSurfaceReviewAGate(gate, locale); err != nil {
			return nil, err
		}
		current, err := currentLocaleSurfaceReviewAInputs(root, locale, catalog, gate.SchemaVersion)
		if err != nil {
			return nil, err
		}
		if gate.Inputs != current {
			continue
		}
		currentGates = append(currentGates, gate)
	}
	if len(currentGates) == 0 {
		return nil, fmt.Errorf("language review evidence/gate stale for %s; complete Locale Surface Review A again and record the current A gate", locale)
	}
	return currentGates, nil
}

func validateLocaleSurfaceReviewAGate(gate LocaleSurfaceReviewAGate, locale string) error {
	if gate.Locale != locale || gate.Stage != localeSurfaceReviewAStage || gate.Decision != "passed" || gate.ReviewID == "" || gate.Reviewer == "" {
		return fmt.Errorf("language review evidence/gate stale: invalid Locale Surface Review A gate for %s", locale)
	}
	if gate.SchemaVersion != localeSurfaceReviewASchemaVersionV1 && gate.SchemaVersion != localeSurfaceReviewASchemaVersion {
		return fmt.Errorf("language review evidence/gate stale: unsupported Locale Surface Review A gate schema version %d", gate.SchemaVersion)
	}
	return nil
}

func hashBytes(data []byte) string { value := sha256.Sum256(data); return hex.EncodeToString(value[:]) }
