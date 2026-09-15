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
	"unicode"
	"unicode/utf8"
)

const (
	CourseMetadataSchemaVersion       = 1
	CourseMetadataGeneratorContract   = "course-seo-description-v1"
	CourseMetadataPromptVersion       = "course-seo-description-v1"
	CourseMetadataSchemaVersionV2     = 2
	CourseMetadataGeneratorContractV2 = "course-seo-localization-v2"
	CourseMetadataPromptVersionV2     = "course-seo-localization-v2"
	CourseDescriptionMinRunes         = 30
	CourseDescriptionMaxRunes         = 200
)

var (
	htmlTagRE       = regexp.MustCompile(`(?i)</?[a-z][^>]*>|<![a-z][^>]*>`)
	schemeURLRE     = regexp.MustCompile(`(?i)[a-z][a-z0-9+.-]*://`)
	wwwURLRE        = regexp.MustCompile(`(?i)\bwww\.`)
	bareDomainURLRE = regexp.MustCompile(`(?i)\b[a-z0-9](?:[a-z0-9-]{0,62}\.)+[a-z]{2,}(?:/\S*)?`)
	goSelectorRE    = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[A-Z][A-Za-z0-9_]*$`)
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
	PageID                  string                   `json:"page_id"`
	Route                   string                   `json:"route"`
	Description             string                   `json:"description"`
	SourceSHA256            string                   `json:"source_sha256"`
	SourceDescriptionSHA256 string                   `json:"source_description_sha256,omitempty"`
	TargetSHA256            string                   `json:"target_sha256"`
	GlossarySHA256          string                   `json:"glossary_sha256"`
	Generation              CourseMetadataGeneration `json:"generation"`
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
	SchemaVersion int
	Locale        string
	Provider      string
	Model         string
	GeneratedAt   string
	Descriptions  []byte
}

// CourseMetadataRefreshOptions supplies only descriptions that must be
// regenerated. The existing formal metadata asset is the immutable base for
// every non-stale Page entry.
type CourseMetadataRefreshOptions struct {
	SchemaVersion int
	Locale        string
	Provider      string
	Model         string
	GeneratedAt   string
	Descriptions  []byte
}

type courseDescriptionsFile struct {
	Pages []courseDescriptionEntry `json:"pages"`
}

type courseDescriptionEntry struct {
	PageID      string `json:"page_id"`
	Description string `json:"description"`
}

// LoadCourseMetadata loads and strictly validates the committed formal asset.
func LoadCourseMetadata(root, locale string, catalog *Catalog) (*CourseMetadata, error) {
	if catalog == nil {
		return nil, fmt.Errorf("catalog is required")
	}
	metadataPath := filepath.Join(root, "locales", locale, "course-metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("read course metadata: %w", err)
	}
	decoded, err := decodeCourseMetadata(data)
	if err != nil {
		return nil, fmt.Errorf("parse course metadata: %w", err)
	}
	targets, glossary, err := loadReadyCourseMetadataInputs(root, locale, catalog)
	if err != nil {
		return nil, err
	}
	var sourceDescriptions *CourseSourceDescriptions
	if decoded.SchemaVersion == CourseMetadataSchemaVersionV2 {
		sourceDescriptions, err = LoadCourseSourceDescriptions(root, catalog)
		if err != nil {
			return nil, err
		}
		if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err != nil {
			return nil, err
		}
	}
	return validateCourseMetadataWithSourceDescriptions(data, locale, catalog, targets, glossary, sourceDescriptions)
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
	if options.SchemaVersion == 0 {
		options.SchemaVersion = CourseMetadataSchemaVersion
	}
	contract, prompt, err := courseMetadataContract(options.SchemaVersion)
	if err != nil {
		return nil, err
	}
	descriptionByID, err := parseCourseDescriptions(options.Descriptions)
	if err != nil {
		return nil, err
	}
	if err := requireExactCourseDescriptionSet(descriptionByID, catalog, "course descriptions"); err != nil {
		return nil, err
	}

	targets, glossary, err := loadReadyCourseMetadataInputs(root, options.Locale, catalog)
	if err != nil {
		return nil, err
	}
	var sourceDescriptions *CourseSourceDescriptions
	var sourceByID map[string]CourseSourceDescriptionPage
	if options.SchemaVersion == CourseMetadataSchemaVersionV2 {
		sourceDescriptions, err = LoadCourseSourceDescriptions(root, catalog)
		if err != nil {
			return nil, err
		}
		if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err != nil {
			return nil, err
		}
		sourceByID = courseSourceDescriptionsByID(sourceDescriptions)
	}
	metadata := CourseMetadata{
		SchemaVersion: options.SchemaVersion, Locale: options.Locale, GeneratorContract: contract,
		Pages: make([]CoursePageMetadata, 0, len(catalog.Pages)),
	}
	for _, page := range catalog.Pages {
		entry := CoursePageMetadata{
			PageID: page.ID, Route: page.Route, Description: descriptionByID[page.ID],
			SourceSHA256: page.SourceSHA256, TargetSHA256: sum(targets[page.ID]), GlossarySHA256: sum(glossary),
			Generation: CourseMetadataGeneration{
				Provider: options.Provider, Model: options.Model, PromptVersion: prompt, GeneratedAt: options.GeneratedAt,
			},
		}
		if options.SchemaVersion == CourseMetadataSchemaVersionV2 {
			entry.SourceDescriptionSHA256 = sum([]byte(sourceByID[page.ID].Description))
		}
		metadata.Pages = append(metadata.Pages, entry)
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode course metadata: %w", err)
	}
	data = append(data, '\n')
	if _, err := validateCourseMetadataWithSourceDescriptions(data, options.Locale, catalog, targets, glossary, sourceDescriptions); err != nil {
		return nil, fmt.Errorf("validate assembled course metadata: %w", err)
	}
	return data, nil
}

// RefreshCourseMetadata produces a complete current metadata asset while
// requiring new AI descriptions only for Pages whose existing metadata
// identity is stale. It never treats the previous description as new output.
func RefreshCourseMetadata(root string, catalog *Catalog, options CourseMetadataRefreshOptions) ([]byte, []string, error) {
	if catalog == nil {
		return nil, nil, fmt.Errorf("catalog is required")
	}
	basePath := filepath.Join(root, "locales", options.Locale, "course-metadata.json")
	baseData, err := os.ReadFile(basePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read course metadata refresh base: %w", err)
	}
	base, err := decodeCourseMetadata(baseData)
	if err != nil {
		return nil, nil, fmt.Errorf("parse course metadata refresh base: %w", err)
	}
	if _, _, err := courseMetadataContract(base.SchemaVersion); err != nil {
		return nil, nil, fmt.Errorf("course metadata refresh base: %w", err)
	}
	if options.SchemaVersion != 0 && base.SchemaVersion != options.SchemaVersion {
		return nil, nil, fmt.Errorf("course metadata refresh base schema_version=%d, requested %d", base.SchemaVersion, options.SchemaVersion)
	}
	if base.Locale != options.Locale {
		return nil, nil, fmt.Errorf("course metadata refresh base locale %q does not match requested locale %q", base.Locale, options.Locale)
	}
	baseByID, err := courseMetadataBaseIndex(base, catalog)
	if err != nil {
		return nil, nil, err
	}
	targets, glossary, err := loadReadyCourseMetadataInputs(root, options.Locale, catalog)
	if err != nil {
		return nil, nil, err
	}
	glossaryHash := sum(glossary)
	contract, prompt, _ := courseMetadataContract(base.SchemaVersion)
	var sourceDescriptions *CourseSourceDescriptions
	var sourceByID map[string]CourseSourceDescriptionPage
	if base.SchemaVersion == CourseMetadataSchemaVersionV2 {
		sourceDescriptions, err = LoadCourseSourceDescriptions(root, catalog)
		if err != nil {
			return nil, nil, err
		}
		if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err != nil {
			return nil, nil, err
		}
		sourceByID = courseSourceDescriptionsByID(sourceDescriptions)
	}
	stale := make([]string, 0)
	staleByID := make(map[string]bool, len(catalog.Pages))
	for _, page := range catalog.Pages {
		entry := baseByID[page.ID]
		if courseMetadataEntryIsStale(base, entry, page, targets[page.ID], glossaryHash, sourceByID) {
			stale = append(stale, page.ID)
			staleByID[page.ID] = true
		}
	}
	descriptionByID, err := parseCourseDescriptions(options.Descriptions)
	if err != nil {
		return nil, nil, err
	}
	for pageID := range descriptionByID {
		if _, ok := baseByID[pageID]; !ok {
			return nil, nil, fmt.Errorf("course refresh descriptions has extra page_id %q", pageID)
		}
		if !staleByID[pageID] {
			return nil, nil, fmt.Errorf("course refresh descriptions has non-stale page_id %q", pageID)
		}
	}
	if missing := missingCourseDescriptionIDs(stale, descriptionByID); len(missing) > 0 {
		return nil, nil, fmt.Errorf("course refresh descriptions is missing stale page(s): %s", strings.Join(missing, ", "))
	}

	metadata := CourseMetadata{
		SchemaVersion: base.SchemaVersion, Locale: options.Locale, GeneratorContract: contract,
		Pages: make([]CoursePageMetadata, 0, len(catalog.Pages)),
	}
	for _, page := range catalog.Pages {
		entry := baseByID[page.ID]
		if staleByID[page.ID] {
			entry = CoursePageMetadata{
				PageID: page.ID, Route: page.Route, Description: descriptionByID[page.ID],
				SourceSHA256: page.SourceSHA256, TargetSHA256: sum(targets[page.ID]), GlossarySHA256: glossaryHash,
				Generation: CourseMetadataGeneration{Provider: options.Provider, Model: options.Model, PromptVersion: prompt, GeneratedAt: options.GeneratedAt},
			}
			if base.SchemaVersion == CourseMetadataSchemaVersionV2 {
				entry.SourceDescriptionSHA256 = sum([]byte(sourceByID[page.ID].Description))
			}
		}
		metadata.Pages = append(metadata.Pages, entry)
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("encode refreshed course metadata: %w", err)
	}
	data = append(data, '\n')
	if _, err := validateCourseMetadataWithSourceDescriptions(data, options.Locale, catalog, targets, glossary, sourceDescriptions); err != nil {
		return nil, nil, fmt.Errorf("validate refreshed course metadata: %w", err)
	}
	return data, stale, nil
}

func parseCourseDescriptions(data []byte) (map[string]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
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
	return descriptionByID, nil
}

func decodeCourseMetadata(data []byte) (*CourseMetadata, error) {
	var metadata CourseMetadata
	if err := decodeStrictJSON(data, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

func courseMetadataContract(schemaVersion int) (string, string, error) {
	switch schemaVersion {
	case CourseMetadataSchemaVersion:
		return CourseMetadataGeneratorContract, CourseMetadataPromptVersion, nil
	case CourseMetadataSchemaVersionV2:
		return CourseMetadataGeneratorContractV2, CourseMetadataPromptVersionV2, nil
	default:
		return "", "", fmt.Errorf("course metadata schema_version=%d is unsupported", schemaVersion)
	}
}

func courseSourceDescriptionsByID(source *CourseSourceDescriptions) map[string]CourseSourceDescriptionPage {
	byID := make(map[string]CourseSourceDescriptionPage, len(source.Pages))
	for _, page := range source.Pages {
		byID[page.PageID] = page
	}
	return byID
}

func courseMetadataBaseIndex(metadata *CourseMetadata, catalog *Catalog) (map[string]CoursePageMetadata, error) {
	expected := make(map[string]struct{}, len(catalog.Pages))
	for _, page := range catalog.Pages {
		expected[page.ID] = struct{}{}
	}
	entries := make(map[string]CoursePageMetadata, len(metadata.Pages))
	for i, entry := range metadata.Pages {
		if metadata.SchemaVersion == CourseMetadataSchemaVersionV2 && i < len(catalog.Pages) && entry.PageID != catalog.Pages[i].ID {
			return nil, fmt.Errorf("course metadata refresh base page %d has page_id %q, want catalog page_id %q", i, entry.PageID, catalog.Pages[i].ID)
		}
		if _, ok := expected[entry.PageID]; !ok {
			return nil, fmt.Errorf("course metadata refresh base has extra page_id %q", entry.PageID)
		}
		if _, exists := entries[entry.PageID]; exists {
			return nil, fmt.Errorf("course metadata refresh base has duplicate page_id %q", entry.PageID)
		}
		entries[entry.PageID] = entry
	}
	missing := make([]string, 0)
	for _, page := range catalog.Pages {
		if _, ok := entries[page.ID]; !ok {
			missing = append(missing, page.ID)
		}
	}
	if len(missing) > 0 || len(entries) != len(catalog.Pages) {
		sort.Strings(missing)
		return nil, fmt.Errorf("course metadata refresh base page set does not match current catalog; missing page(s): %s", strings.Join(missing, ", "))
	}
	return entries, nil
}

func courseMetadataEntryIsStale(metadata *CourseMetadata, entry CoursePageMetadata, page Page, target []byte, glossaryHash string, sourceByID map[string]CourseSourceDescriptionPage) bool {
	contract, prompt, err := courseMetadataContract(metadata.SchemaVersion)
	if err != nil {
		return true
	}
	stale := metadata.GeneratorContract != contract || entry.Route != page.Route ||
		entry.SourceSHA256 != page.SourceSHA256 || entry.TargetSHA256 != sum(target) ||
		entry.GlossarySHA256 != glossaryHash || entry.Generation.PromptVersion != prompt
	if metadata.SchemaVersion == CourseMetadataSchemaVersionV2 {
		source, ok := sourceByID[page.ID]
		stale = stale || !ok || entry.SourceDescriptionSHA256 != sum([]byte(source.Description))
	}
	return stale
}

func missingCourseDescriptionIDs(stale []string, descriptions map[string]string) []string {
	missing := make([]string, 0)
	for _, pageID := range stale {
		if _, ok := descriptions[pageID]; !ok {
			missing = append(missing, pageID)
		}
	}
	return missing
}

func validateCourseMetadata(data []byte, locale string, catalog *Catalog, targets map[string][]byte, glossary []byte) (*CourseMetadata, error) {
	return validateCourseMetadataWithSourceDescriptions(data, locale, catalog, targets, glossary, nil)
}

func validateCourseMetadataWithSourceDescriptions(data []byte, locale string, catalog *Catalog, targets map[string][]byte, glossary []byte, sourceDescriptions *CourseSourceDescriptions) (*CourseMetadata, error) {
	var metadata CourseMetadata
	if err := decodeStrictJSON(data, &metadata); err != nil {
		return nil, fmt.Errorf("parse course metadata: %w", err)
	}
	contract, prompt, err := courseMetadataContract(metadata.SchemaVersion)
	if err != nil {
		return nil, err
	}
	if metadata.Locale != locale {
		return nil, fmt.Errorf("course metadata locale %q does not match requested locale %q", metadata.Locale, locale)
	}
	if metadata.GeneratorContract != contract {
		return nil, fmt.Errorf("course metadata generator_contract %q is stale; want %q", metadata.GeneratorContract, contract)
	}
	var sourceByID map[string]CourseSourceDescriptionPage
	if metadata.SchemaVersion == CourseMetadataSchemaVersionV2 {
		if sourceDescriptions == nil {
			return nil, fmt.Errorf("course metadata schema v2 requires current canonical English source descriptions")
		}
		sourceByID = courseSourceDescriptionsByID(sourceDescriptions)
	}
	if metadata.SchemaVersion == CourseMetadataSchemaVersionV2 && len(metadata.Pages) != len(catalog.Pages) {
		return nil, fmt.Errorf("course metadata pages=%d, catalog pages=%d", len(metadata.Pages), len(catalog.Pages))
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
	for i, entry := range metadata.Pages {
		if _, ok := seenPages[entry.PageID]; ok {
			return nil, fmt.Errorf("course metadata has duplicate page_id %q", entry.PageID)
		}
		seenPages[entry.PageID] = struct{}{}
		page, ok := expected[entry.PageID]
		if !ok {
			return nil, fmt.Errorf("course metadata has extra page_id %q", entry.PageID)
		}
		if metadata.SchemaVersion == CourseMetadataSchemaVersionV2 && entry.PageID != catalog.Pages[i].ID {
			return nil, fmt.Errorf("course metadata page %d has page_id %q, want catalog page_id %q", i, entry.PageID, catalog.Pages[i].ID)
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
		if metadata.SchemaVersion == CourseMetadataSchemaVersion {
			if entry.SourceDescriptionSHA256 != "" {
				return nil, fmt.Errorf("%s: source_description_sha256 is not valid in schema v1", entry.PageID)
			}
		} else {
			source, ok := sourceByID[entry.PageID]
			if !ok || entry.SourceDescriptionSHA256 != sum([]byte(source.Description)) {
				return nil, fmt.Errorf("%s: source_description_sha256 is stale", entry.PageID)
			}
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
		if entry.Generation.PromptVersion != prompt {
			return nil, fmt.Errorf("%s: prompt_version %q is stale; want %q", entry.PageID, entry.Generation.PromptVersion, prompt)
		}
		if err := validateCourseGeneration(entry.PageID, entry.Generation); err != nil {
			return nil, err
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
	if containsCourseDescriptionURL(description) {
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

func containsCourseDescriptionURL(description string) bool {
	if schemeURLRE.MatchString(description) || wwwURLRE.MatchString(description) {
		return true
	}
	for _, match := range bareDomainURLRE.FindAllString(description, -1) {
		// A lowercase package qualifier followed by an exported Go identifier,
		// such as io.Reader, is a selector rather than a bare domain.
		if goSelectorRE.MatchString(match) {
			continue
		}
		return true
	}
	return false
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
