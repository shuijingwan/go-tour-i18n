package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	CourseMetadataSchemaVersion     = 1
	CourseMetadataGeneratorContract = "course-seo-description-v1"
	CourseMetadataPromptVersion     = "course-seo-description-v1"
	CourseDescriptionMinRunes       = 30
	CourseDescriptionMaxRunes       = 200
)

var (
	htmlTagRE = regexp.MustCompile(`(?i)</?[a-z][^>]*>|<![a-z][^>]*>`)
	urlRE     = regexp.MustCompile(`(?i)(?:[a-z][a-z0-9+.-]*://|www\.|\b[a-z0-9](?:[a-z0-9-]{0,62}\.)+[a-z]{2,}(?:/\S*)?)`)
)

// CourseMetadata is the complete locale-level SEO asset for the formal Page
// catalog. It is separate from TranslationUnit candidates and review evidence.
type CourseMetadata struct {
	SchemaVersion     int                  `json:"schema_version"`
	Locale            string               `json:"locale"`
	GeneratorContract string               `json:"generator_contract"`
	Pages             []CoursePageMetadata `json:"pages"`
}

// CoursePageMetadata binds one generated description to the persistent Page
// identity and every input whose change makes the description stale.
type CoursePageMetadata struct {
	PageID         string                   `json:"page_id"`
	Route          string                   `json:"route"`
	Description    string                   `json:"description"`
	SourceSHA256   string                   `json:"source_sha256"`
	TargetSHA256   string                   `json:"target_sha256"`
	GlossarySHA256 string                   `json:"glossary_sha256"`
	Generation     CourseMetadataGeneration `json:"generation"`
}

type CourseMetadataGeneration struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	PromptVersion string `json:"prompt_version"`
	GeneratedAt   string `json:"generated_at"`
}

// CourseMetadataAssemblyOptions contains generation provenance supplied by the
// offline caller. All content identities are derived from the repository.
type CourseMetadataAssemblyOptions struct {
	Locale       string
	Provider     string
	Model        string
	GeneratedAt  string
	Descriptions []byte
}

type courseDescriptionsFile struct {
	Pages []courseDescriptionEntry `json:"pages"`
}

type courseDescriptionEntry struct {
	PageID      string `json:"page_id"`
	Description string `json:"description"`
}

// LoadCourseMetadata loads and strictly validates the committed formal asset.
// It is intentionally not called by projection or production in phase one.
func LoadCourseMetadata(root, locale string, catalog *Catalog) (*CourseMetadata, error) {
	if catalog == nil {
		return nil, fmt.Errorf("catalog is required")
	}
	metadataPath := filepath.Join(root, "locales", locale, "course-metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("read course metadata: %w", err)
	}
	targets, glossary, err := loadReadyCourseMetadataInputs(root, locale, catalog)
	if err != nil {
		return nil, err
	}
	return validateCourseMetadata(data, locale, catalog, targets, glossary)
}

func loadReadyCourseMetadataInputs(root, locale string, catalog *Catalog) (map[string][]byte, []byte, error) {
	glossary, err := os.ReadFile(filepath.Join(root, "locales", locale, "glossary.yaml"))
	if err != nil {
		return nil, nil, fmt.Errorf("read course metadata glossary: %w", err)
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", locale, "status.tsv"))
	if err != nil {
		return nil, nil, fmt.Errorf("read course metadata status: %w", err)
	}
	statusByID := make(map[string]*Status, len(statuses))
	for i := range statuses {
		if _, exists := statusByID[statuses[i].UnitID]; exists {
			return nil, nil, fmt.Errorf("course metadata status has duplicate unit_id %q", statuses[i].UnitID)
		}
		statusByID[statuses[i].UnitID] = &statuses[i]
	}
	targets := make(map[string][]byte, len(catalog.Pages))
	for _, page := range catalog.Pages {
		status, ok := statusByID[page.ID]
		if !ok {
			return nil, nil, fmt.Errorf("%s: formal locale status is missing", page.ID)
		}
		target, err := loadReadyCandidate(root, catalog, page.ID, locale, status)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: load ready canonical Page target: %w", page.ID, err)
		}
		targets[page.ID] = target
	}
	return targets, glossary, nil
}

// AssembleCourseMetadata deterministically expands a complete page_id to
// description input into the formal schema. It performs no model calls and
// validates the assembled bytes with the same validator used by the loader.
func AssembleCourseMetadata(root string, catalog *Catalog, options CourseMetadataAssemblyOptions) ([]byte, error) {
	if catalog == nil {
		return nil, fmt.Errorf("catalog is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(options.Descriptions))
	decoder.DisallowUnknownFields()
	var descriptions courseDescriptionsFile
	if err := decoder.Decode(&descriptions); err != nil {
		return nil, fmt.Errorf("parse course descriptions: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("parse course descriptions: multiple JSON values")
		}
		return nil, fmt.Errorf("parse course descriptions: %w", err)
	}
	descriptionByID := make(map[string]string, len(descriptions.Pages))
	for _, entry := range descriptions.Pages {
		if _, exists := descriptionByID[entry.PageID]; exists {
			return nil, fmt.Errorf("course descriptions has duplicate page_id %q", entry.PageID)
		}
		descriptionByID[entry.PageID] = entry.Description
	}
	expected := make(map[string]struct{}, len(catalog.Pages))
	for _, page := range catalog.Pages {
		expected[page.ID] = struct{}{}
	}
	for pageID := range descriptionByID {
		if _, ok := expected[pageID]; !ok {
			return nil, fmt.Errorf("course descriptions has extra page_id %q", pageID)
		}
	}
	var missing []string
	for _, page := range catalog.Pages {
		if _, ok := descriptionByID[page.ID]; !ok {
			missing = append(missing, page.ID)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("course descriptions is missing page(s): %s", strings.Join(missing, ", "))
	}

	targets, glossary, err := loadReadyCourseMetadataInputs(root, options.Locale, catalog)
	if err != nil {
		return nil, err
	}
	metadata := CourseMetadata{
		SchemaVersion: CourseMetadataSchemaVersion, Locale: options.Locale, GeneratorContract: CourseMetadataGeneratorContract,
		Pages: make([]CoursePageMetadata, 0, len(catalog.Pages)),
	}
	for _, page := range catalog.Pages {
		metadata.Pages = append(metadata.Pages, CoursePageMetadata{
			PageID: page.ID, Route: page.Route, Description: descriptionByID[page.ID],
			SourceSHA256: page.SourceSHA256, TargetSHA256: sum(targets[page.ID]), GlossarySHA256: sum(glossary),
			Generation: CourseMetadataGeneration{
				Provider: options.Provider, Model: options.Model, PromptVersion: CourseMetadataPromptVersion, GeneratedAt: options.GeneratedAt,
			},
		})
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode course metadata: %w", err)
	}
	data = append(data, '\n')
	if _, err := validateCourseMetadata(data, options.Locale, catalog, targets, glossary); err != nil {
		return nil, fmt.Errorf("validate assembled course metadata: %w", err)
	}
	return data, nil
}

func validateCourseMetadata(data []byte, locale string, catalog *Catalog, targets map[string][]byte, glossary []byte) (*CourseMetadata, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var metadata CourseMetadata
	if err := decoder.Decode(&metadata); err != nil {
		return nil, fmt.Errorf("parse course metadata: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("parse course metadata: multiple JSON values")
		}
		return nil, fmt.Errorf("parse course metadata: %w", err)
	}
	if metadata.SchemaVersion != CourseMetadataSchemaVersion {
		return nil, fmt.Errorf("course metadata schema_version=%d, want %d", metadata.SchemaVersion, CourseMetadataSchemaVersion)
	}
	if metadata.Locale != locale {
		return nil, fmt.Errorf("course metadata locale %q does not match requested locale %q", metadata.Locale, locale)
	}
	if metadata.GeneratorContract != CourseMetadataGeneratorContract {
		return nil, fmt.Errorf("course metadata generator_contract %q is stale; want %q", metadata.GeneratorContract, CourseMetadataGeneratorContract)
	}

	expected := make(map[string]Page, len(catalog.Pages))
	for _, page := range catalog.Pages {
		expected[page.ID] = page
	}
	seenPages := make(map[string]struct{}, len(metadata.Pages))
	seenRoutes := make(map[string]string, len(metadata.Pages))
	seenDescriptions := make(map[string]string, len(metadata.Pages))
	seenNormalized := make(map[string]string, len(metadata.Pages))
	glossaryHash := sum(glossary)
	for _, entry := range metadata.Pages {
		if _, ok := seenPages[entry.PageID]; ok {
			return nil, fmt.Errorf("course metadata has duplicate page_id %q", entry.PageID)
		}
		seenPages[entry.PageID] = struct{}{}
		page, ok := expected[entry.PageID]
		if !ok {
			return nil, fmt.Errorf("course metadata has extra page_id %q", entry.PageID)
		}
		if prior := seenRoutes[entry.Route]; prior != "" {
			return nil, fmt.Errorf("course metadata pages %q and %q have duplicate route %q", prior, entry.PageID, entry.Route)
		}
		seenRoutes[entry.Route] = entry.PageID
		if entry.Route != page.Route {
			return nil, fmt.Errorf("%s: route %q is stale; want %q", entry.PageID, entry.Route, page.Route)
		}
		if entry.SourceSHA256 != page.SourceSHA256 {
			return nil, fmt.Errorf("%s: source_sha256 is stale", entry.PageID)
		}
		target, ok := targets[entry.PageID]
		if !ok {
			return nil, fmt.Errorf("%s: canonical Page target is missing", entry.PageID)
		}
		if entry.TargetSHA256 != sum(target) {
			return nil, fmt.Errorf("%s: target_sha256 is stale", entry.PageID)
		}
		if entry.GlossarySHA256 != glossaryHash {
			return nil, fmt.Errorf("%s: glossary_sha256 is stale", entry.PageID)
		}
		if entry.Generation.PromptVersion != CourseMetadataPromptVersion {
			return nil, fmt.Errorf("%s: prompt_version %q is stale; want %q", entry.PageID, entry.Generation.PromptVersion, CourseMetadataPromptVersion)
		}
		if strings.TrimSpace(entry.Generation.Provider) == "" || strings.TrimSpace(entry.Generation.Model) == "" {
			return nil, fmt.Errorf("%s: generation provider and model are required", entry.PageID)
		}
		generatedAt, err := time.Parse(time.RFC3339, entry.Generation.GeneratedAt)
		if err != nil || generatedAt.Location() != time.UTC {
			return nil, fmt.Errorf("%s: generated_at must be RFC 3339 UTC", entry.PageID)
		}
		if err := validateCourseDescription(entry.Description); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.PageID, err)
		}
		if prior := seenDescriptions[entry.Description]; prior != "" {
			return nil, fmt.Errorf("course metadata pages %q and %q have exact duplicate descriptions", prior, entry.PageID)
		}
		seenDescriptions[entry.Description] = entry.PageID
		normalized := normalizeCourseDescription(entry.Description)
		if prior := seenNormalized[normalized]; prior != "" {
			return nil, fmt.Errorf("course metadata pages %q and %q have normalized duplicate descriptions", prior, entry.PageID)
		}
		seenNormalized[normalized] = entry.PageID
	}

	var missing []string
	for _, page := range catalog.Pages {
		if _, ok := seenPages[page.ID]; !ok {
			missing = append(missing, page.ID)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("course metadata is missing page(s): %s", strings.Join(missing, ", "))
	}
	if len(metadata.Pages) != len(catalog.Pages) {
		return nil, fmt.Errorf("course metadata pages=%d, catalog pages=%d", len(metadata.Pages), len(catalog.Pages))
	}
	return &metadata, nil
}

func validateCourseDescription(description string) error {
	if strings.TrimSpace(description) == "" {
		return fmt.Errorf("description is empty")
	}
	if description != strings.TrimSpace(description) {
		return fmt.Errorf("description has leading or trailing whitespace")
	}
	for _, r := range description {
		if r == '\n' || r == '\r' || unicode.IsControl(r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) {
			return fmt.Errorf("description must be one paragraph without control characters or line breaks")
		}
	}
	if htmlTagRE.MatchString(description) {
		return fmt.Errorf("description contains an HTML tag")
	}
	if strings.Contains(description, "```") || strings.Contains(description, "~~~") {
		return fmt.Errorf("description contains a Markdown code fence")
	}
	if urlRE.MatchString(description) {
		return fmt.Errorf("description contains a URL")
	}
	length := utf8.RuneCountInString(description)
	if length < CourseDescriptionMinRunes {
		return fmt.Errorf("description length=%d, minimum=%d Unicode code points", length, CourseDescriptionMinRunes)
	}
	if length > CourseDescriptionMaxRunes {
		return fmt.Errorf("description length=%d, maximum=%d Unicode code points", length, CourseDescriptionMaxRunes)
	}
	if normalizeCourseDescription(description) == "" {
		return fmt.Errorf("description has no letters or numbers")
	}
	return nil
}

func normalizeCourseDescription(description string) string {
	var normalized strings.Builder
	for _, r := range strings.ToLower(description) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			normalized.WriteRune(r)
		}
	}
	return normalized.String()
}
