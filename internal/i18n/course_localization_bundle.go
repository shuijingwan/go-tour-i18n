package i18n

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const CourseLocalizationGenerationBundleSchemaVersion = 1

var courseLocalizationGenerationAuthorityPaths = []string{
	"AGENTS.md",
	"docs/CHATGPT_LANGUAGE_GENERATION.md",
	"docs/COURSE_SEO_METADATA.md",
}

type CourseLocalizationGenerationBundleFile struct {
	BundlePath     string `json:"bundle_path"`
	RepositoryPath string `json:"repository_path,omitempty"`
	SHA256         string `json:"sha256"`
}

type CourseLocalizationGenerationPage struct {
	Index                   int    `json:"index"`
	PageID                  string `json:"page_id"`
	Route                   string `json:"route"`
	CanonicalDescription    string `json:"canonical_description"`
	EnglishSource           string `json:"english_source"`
	ReadyTarget             string `json:"ready_target"`
	SourceSHA256            string `json:"source_sha256"`
	SourceDescriptionSHA256 string `json:"source_description_sha256"`
	TargetSHA256            string `json:"target_sha256"`
}

type CourseLocalizationGenerationContext struct {
	SchemaVersion                     int                                `json:"schema_version"`
	Kind                              string                             `json:"kind"`
	Locale                            string                             `json:"locale"`
	GeneratorContract                 string                             `json:"generator_contract"`
	CanonicalSourceDescriptionsSHA256 string                             `json:"canonical_source_descriptions_sha256"`
	CanonicalReviewAuthoritySHA256    string                             `json:"canonical_review_authority_sha256"`
	GlossarySHA256                    string                             `json:"glossary_sha256"`
	LocaleIdentity                    Locale                             `json:"locale_identity"`
	Pages                             []CourseLocalizationGenerationPage `json:"pages"`
}

type CourseLocalizationGenerationBundleManifest struct {
	SchemaVersion      int                                      `json:"schema_version"`
	Kind               string                                   `json:"kind"`
	Locale             string                                   `json:"locale"`
	PageCount          int                                      `json:"page_count"`
	Context            CourseLocalizationGenerationBundleFile   `json:"context"`
	SourceDescriptions CourseLocalizationGenerationBundleFile   `json:"source_descriptions"`
	Glossary           CourseLocalizationGenerationBundleFile   `json:"glossary"`
	LocaleIdentity     CourseLocalizationGenerationBundleFile   `json:"locale_identity"`
	Authority          []CourseLocalizationGenerationBundleFile `json:"authority"`
}

// ExportCourseLocalizationGenerationBundle creates a deterministic ZIP transport
// for schema-v2 Course SEO generation. It does not change Course SEO authority,
// metadata, status, or any review gate.
func ExportCourseLocalizationGenerationBundle(root, locale string, catalog *Catalog) ([]byte, CourseLocalizationGenerationBundleManifest, error) {
	if catalog == nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, fmt.Errorf("course localization bundle catalog is required")
	}
	if err := ValidateLocaleName(locale); err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, err
	}
	if _, err := localeWorkflowDirectory(root, locale); err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, err
	}
	if _, err := LoadGlossary(root, locale); err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, err
	}

	sourceDescriptionsData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(courseSourceDescriptionsRelativePath)))
	if err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, fmt.Errorf("read canonical source descriptions: %w", err)
	}
	sourceDescriptions, err := LoadCourseSourceDescriptions(root, catalog)
	if err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, err
	}
	reviewAuthority, err := RequireCurrentCourseSourceDescriptionReview(root, catalog)
	if err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, err
	}
	targets, glossaryData, err := loadReadyCourseMetadataInputs(root, locale, catalog)
	if err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, err
	}
	if len(sourceDescriptions.Pages) != len(catalog.Pages) {
		return nil, CourseLocalizationGenerationBundleManifest{}, fmt.Errorf("course source-description page count mismatch")
	}

	localeRepo := filepath.ToSlash(filepath.Join("locales", locale, "locale.json"))
	localeData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(localeRepo)))
	if err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, fmt.Errorf("read locale identity: %w", err)
	}
	var localeIdentity Locale
	if err := json.Unmarshal(localeData, &localeIdentity); err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, fmt.Errorf("parse locale identity: %w", err)
	}
	if localeIdentity.Locale != locale {
		return nil, CourseLocalizationGenerationBundleManifest{}, fmt.Errorf("locale identity mismatch: %q", localeIdentity.Locale)
	}

	context := CourseLocalizationGenerationContext{
		SchemaVersion:                     CourseLocalizationGenerationBundleSchemaVersion,
		Kind:                              "go-tour-i18n/course-seo-localization-context",
		Locale:                            locale,
		GeneratorContract:                 "course-seo-localization-v2",
		CanonicalSourceDescriptionsSHA256: sum(sourceDescriptionsData),
		CanonicalReviewAuthoritySHA256:    reviewAuthority,
		GlossarySHA256:                    sum(glossaryData),
		LocaleIdentity:                    localeIdentity,
		Pages:                             make([]CourseLocalizationGenerationPage, 0, len(catalog.Pages)),
	}
	for i, page := range catalog.Pages {
		canonical := sourceDescriptions.Pages[i]
		if canonical.PageID != page.ID || canonical.Route != page.Route || canonical.SourceSHA256 != page.SourceSHA256 {
			return nil, CourseLocalizationGenerationBundleManifest{}, fmt.Errorf("course source-description identity mismatch at page %d", i+1)
		}
		target, ok := targets[page.ID]
		if !ok {
			return nil, CourseLocalizationGenerationBundleManifest{}, fmt.Errorf("%s: ready target missing", page.ID)
		}
		context.Pages = append(context.Pages, CourseLocalizationGenerationPage{
			Index:                   i + 1,
			PageID:                  page.ID,
			Route:                   page.Route,
			CanonicalDescription:    canonical.Description,
			EnglishSource:           string(page.Source),
			ReadyTarget:             string(target),
			SourceSHA256:            page.SourceSHA256,
			SourceDescriptionSHA256: sum([]byte(canonical.Description)),
			TargetSHA256:            sum(target),
		})
	}

	contextData, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, err
	}
	contextData = append(contextData, '\n')

	glossaryRepo := filepath.ToSlash(filepath.Join("locales", locale, "glossary.yaml"))
	sourceDescriptionsRepo := filepath.ToSlash(courseSourceDescriptionsRelativePath)
	manifest := CourseLocalizationGenerationBundleManifest{
		SchemaVersion:      CourseLocalizationGenerationBundleSchemaVersion,
		Kind:               "go-tour-i18n/course-seo-localization-generation-bundle",
		Locale:             locale,
		PageCount:          len(context.Pages),
		Context:            CourseLocalizationGenerationBundleFile{BundlePath: "course-seo-localization.json", SHA256: sum(contextData)},
		SourceDescriptions: CourseLocalizationGenerationBundleFile{BundlePath: "formal/source-descriptions.json", RepositoryPath: sourceDescriptionsRepo, SHA256: sum(sourceDescriptionsData)},
		Glossary:           CourseLocalizationGenerationBundleFile{BundlePath: "formal/glossary.yaml", RepositoryPath: glossaryRepo, SHA256: sum(glossaryData)},
		LocaleIdentity:     CourseLocalizationGenerationBundleFile{BundlePath: "formal/locale.json", RepositoryPath: localeRepo, SHA256: sum(localeData)},
		Authority:          make([]CourseLocalizationGenerationBundleFile, 0, len(courseLocalizationGenerationAuthorityPaths)),
	}

	type entry struct {
		path string
		data []byte
	}
	entries := []entry{
		{path: "course-seo-localization.json", data: contextData},
		{path: "formal/source-descriptions.json", data: sourceDescriptionsData},
		{path: "formal/glossary.yaml", data: glossaryData},
		{path: "formal/locale.json", data: localeData},
	}
	for _, repoPath := range courseLocalizationGenerationAuthorityPaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(repoPath)))
		if err != nil {
			return nil, CourseLocalizationGenerationBundleManifest{}, fmt.Errorf("read Course SEO authority %s: %w", repoPath, err)
		}
		bundlePath := filepath.ToSlash(filepath.Join("authority", repoPath))
		manifest.Authority = append(manifest.Authority, CourseLocalizationGenerationBundleFile{
			BundlePath:     bundlePath,
			RepositoryPath: repoPath,
			SHA256:         sum(data),
		})
		entries = append(entries, entry{path: bundlePath, data: data})
	}

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, err
	}
	manifestData = append(manifestData, '\n')

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	if err := writeLocaleSurfaceReviewReviewerBundleEntry(writer, "manifest.json", manifestData); err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, err
	}
	for _, item := range entries {
		if err := writeLocaleSurfaceReviewReviewerBundleEntry(writer, item.path, item.data); err != nil {
			return nil, CourseLocalizationGenerationBundleManifest{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, CourseLocalizationGenerationBundleManifest{}, err
	}
	return buffer.Bytes(), manifest, nil
}
