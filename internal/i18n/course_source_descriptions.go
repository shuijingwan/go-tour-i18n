package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	CourseSourceDescriptionsSchemaVersion      = 1
	CourseSourceDescriptionsGeneratorContract  = "course-seo-source-description-v1"
	CourseSourceDescriptionsPromptVersion      = "course-seo-source-description-v1"
	courseSourceDescriptionReviewSchemaVersion = 1
	courseSourceDescriptionReviewStage         = "canonical-english-course-seo-description-review"
	courseSourceDescriptionsRelativePath       = "data/course-seo/source-descriptions.json"
	courseSourceDescriptionReviewStartMarker   = "<!-- course-source-description-review:start -->"
	courseSourceDescriptionReviewEndMarker     = "<!-- course-source-description-review:end -->"
)

// CourseSourceDescriptions is the repository-controlled semantic source for
// every schema v2 locale course description.
type CourseSourceDescriptions struct {
	SchemaVersion     int                           `json:"schema_version"`
	GeneratorContract string                        `json:"generator_contract"`
	Pages             []CourseSourceDescriptionPage `json:"pages"`
}

type CourseSourceDescriptionPage struct {
	PageID       string                   `json:"page_id"`
	Route        string                   `json:"route"`
	Description  string                   `json:"description"`
	SourceSHA256 string                   `json:"source_sha256"`
	Generation   CourseMetadataGeneration `json:"generation"`
}

type CourseSourceDescriptionAssemblyOptions struct {
	Provider     string
	Model        string
	GeneratedAt  string
	Descriptions []byte
}

type CourseSourceDescriptionReviewInputs struct {
	ArtifactSHA256      string `json:"artifact_sha256"`
	CatalogSourceSHA256 string `json:"catalog_source_sha256"`
	SchemaVersion       int    `json:"source_description_schema_version"`
	GeneratorContract   string `json:"generator_contract"`
	PromptVersion       string `json:"prompt_version"`
}

// CourseSourceDescriptionReviewGate records a completed human review. It is
// independent of TranslationUnit QC and Locale Surface Review evidence.
type CourseSourceDescriptionReviewGate struct {
	SchemaVersion  int                                 `json:"schema_version"`
	ReviewID       string                              `json:"review_id"`
	Stage          string                              `json:"stage"`
	Decision       string                              `json:"decision"`
	Reviewer       string                              `json:"reviewer"`
	EvidenceSHA256 string                              `json:"evidence_sha256"`
	Inputs         CourseSourceDescriptionReviewInputs `json:"inputs"`
}

type courseSourceDescriptionReviewEvidenceBlock struct {
	ReviewID                       string `json:"review_id"`
	ArtifactSHA256                 string `json:"artifact_sha256"`
	CatalogSourceSHA256            string `json:"catalog_source_sha256"`
	SourceDescriptionSchemaVersion int    `json:"source_description_schema_version"`
	GeneratorContract              string `json:"generator_contract"`
	PromptVersion                  string `json:"prompt_version"`
	PageCount                      int    `json:"page_count"`
	Reviewer                       string `json:"reviewer"`
	Decision                       string `json:"decision"`
}

func CourseSourceDescriptionsPath(root string) string {
	return filepath.Join(root, filepath.FromSlash(courseSourceDescriptionsRelativePath))
}

func CourseSourceDescriptionReviewGatePath(root, reviewID string) (string, error) {
	if !reviewIDPattern.MatchString(reviewID) {
		return "", fmt.Errorf("invalid review_id %q", reviewID)
	}
	return filepath.Join(root, "data", "course-seo", "source-description-reviews", reviewID+".gate.json"), nil
}

func CourseSourceDescriptionReviewEvidencePath(root, reviewID string) (string, error) {
	gate, err := CourseSourceDescriptionReviewGatePath(root, reviewID)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(gate, ".gate.json") + ".md", nil
}

func AssembleCourseSourceDescriptions(catalog *Catalog, options CourseSourceDescriptionAssemblyOptions) ([]byte, error) {
	if catalog == nil {
		return nil, fmt.Errorf("catalog is required")
	}
	descriptions, err := parseCourseDescriptions(options.Descriptions)
	if err != nil {
		return nil, err
	}
	if err := requireExactCourseDescriptionSet(descriptions, catalog, "course source descriptions"); err != nil {
		return nil, err
	}
	asset := CourseSourceDescriptions{
		SchemaVersion:     CourseSourceDescriptionsSchemaVersion,
		GeneratorContract: CourseSourceDescriptionsGeneratorContract,
		Pages:             make([]CourseSourceDescriptionPage, 0, len(catalog.Pages)),
	}
	for _, page := range catalog.Pages {
		asset.Pages = append(asset.Pages, CourseSourceDescriptionPage{
			PageID: page.ID, Route: page.Route, Description: descriptions[page.ID], SourceSHA256: page.SourceSHA256,
			Generation: CourseMetadataGeneration{Provider: options.Provider, Model: options.Model, PromptVersion: CourseSourceDescriptionsPromptVersion, GeneratedAt: options.GeneratedAt},
		})
	}
	data, err := json.MarshalIndent(asset, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode course source descriptions: %w", err)
	}
	data = append(data, '\n')
	if _, err := validateCourseSourceDescriptions(data, catalog); err != nil {
		return nil, fmt.Errorf("validate assembled course source descriptions: %w", err)
	}
	return data, nil
}

func LoadCourseSourceDescriptions(root string, catalog *Catalog) (*CourseSourceDescriptions, error) {
	data, err := os.ReadFile(CourseSourceDescriptionsPath(root))
	if err != nil {
		return nil, fmt.Errorf("read course source descriptions: %w", err)
	}
	return validateCourseSourceDescriptions(data, catalog)
}

func validateCourseSourceDescriptions(data []byte, catalog *Catalog) (*CourseSourceDescriptions, error) {
	if catalog == nil {
		return nil, fmt.Errorf("catalog is required")
	}
	var asset CourseSourceDescriptions
	if err := decodeStrictJSON(data, &asset); err != nil {
		return nil, fmt.Errorf("parse course source descriptions: %w", err)
	}
	if asset.SchemaVersion != CourseSourceDescriptionsSchemaVersion {
		return nil, fmt.Errorf("course source descriptions schema_version=%d, want %d", asset.SchemaVersion, CourseSourceDescriptionsSchemaVersion)
	}
	if asset.GeneratorContract != CourseSourceDescriptionsGeneratorContract {
		return nil, fmt.Errorf("course source descriptions generator_contract %q is stale; want %q", asset.GeneratorContract, CourseSourceDescriptionsGeneratorContract)
	}
	if len(asset.Pages) != len(catalog.Pages) {
		return nil, fmt.Errorf("course source descriptions pages=%d, catalog pages=%d", len(asset.Pages), len(catalog.Pages))
	}
	seenDescriptions := make(map[string]string, len(asset.Pages))
	seenNormalized := make(map[string]string, len(asset.Pages))
	seenRoutes := make(map[string]string, len(asset.Pages))
	for i, entry := range asset.Pages {
		page := catalog.Pages[i]
		if entry.PageID != page.ID {
			return nil, fmt.Errorf("course source descriptions page %d has page_id %q, want catalog page_id %q", i, entry.PageID, page.ID)
		}
		if entry.Route != page.Route {
			return nil, fmt.Errorf("%s: route %q is stale; want %q", entry.PageID, entry.Route, page.Route)
		}
		if prior := seenRoutes[entry.Route]; prior != "" {
			return nil, fmt.Errorf("course source descriptions pages %q and %q have duplicate route %q", prior, entry.PageID, entry.Route)
		}
		seenRoutes[entry.Route] = entry.PageID
		if sum(page.Source) != page.SourceSHA256 || entry.SourceSHA256 != page.SourceSHA256 {
			return nil, fmt.Errorf("%s: source_sha256 is stale", entry.PageID)
		}
		if entry.Generation.PromptVersion != CourseSourceDescriptionsPromptVersion {
			return nil, fmt.Errorf("%s: prompt_version %q is stale; want %q", entry.PageID, entry.Generation.PromptVersion, CourseSourceDescriptionsPromptVersion)
		}
		if err := validateCourseGeneration(entry.PageID, entry.Generation); err != nil {
			return nil, err
		}
		if err := validateCourseDescription(entry.Description); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.PageID, err)
		}
		if prior := seenDescriptions[entry.Description]; prior != "" {
			return nil, fmt.Errorf("course source descriptions pages %q and %q have exact duplicate descriptions", prior, entry.PageID)
		}
		seenDescriptions[entry.Description] = entry.PageID
		normalized := normalizeCourseDescription(entry.Description)
		if prior := seenNormalized[normalized]; prior != "" {
			return nil, fmt.Errorf("course source descriptions pages %q and %q have normalized duplicate descriptions", prior, entry.PageID)
		}
		seenNormalized[normalized] = entry.PageID
	}
	return &asset, nil
}

func currentCourseSourceDescriptionReviewInputs(root string, catalog *Catalog) (CourseSourceDescriptionReviewInputs, error) {
	data, err := os.ReadFile(CourseSourceDescriptionsPath(root))
	if err != nil {
		return CourseSourceDescriptionReviewInputs{}, fmt.Errorf("read course source descriptions: %w", err)
	}
	if _, err := validateCourseSourceDescriptions(data, catalog); err != nil {
		return CourseSourceDescriptionReviewInputs{}, err
	}
	catalogIdentity, err := coursePageCatalogIdentity(catalog)
	if err != nil {
		return CourseSourceDescriptionReviewInputs{}, err
	}
	return CourseSourceDescriptionReviewInputs{
		ArtifactSHA256: hashBytes(data), CatalogSourceSHA256: catalogIdentity,
		SchemaVersion: CourseSourceDescriptionsSchemaVersion, GeneratorContract: CourseSourceDescriptionsGeneratorContract,
		PromptVersion: CourseSourceDescriptionsPromptVersion,
	}, nil
}

func BuildCourseSourceDescriptionReviewGate(root, reviewID, reviewer string, catalog *Catalog) ([]byte, string, error) {
	if strings.TrimSpace(reviewer) == "" {
		return nil, "", fmt.Errorf("--reviewer is required")
	}
	path, err := CourseSourceDescriptionReviewGatePath(root, reviewID)
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Lstat(path); err == nil {
		return nil, "", fmt.Errorf("course source-description review gate already exists: %s", filepath.ToSlash(path))
	} else if !os.IsNotExist(err) {
		return nil, "", err
	}
	evidencePath, err := CourseSourceDescriptionReviewEvidencePath(root, reviewID)
	if err != nil {
		return nil, "", err
	}
	evidence, err := os.ReadFile(evidencePath)
	if err != nil {
		return nil, "", fmt.Errorf("course source-description review evidence must exist before recording the gate: %w", err)
	}
	inputs, err := currentCourseSourceDescriptionReviewInputs(root, catalog)
	if err != nil {
		return nil, "", err
	}
	block, err := parseCourseSourceDescriptionReviewEvidence(evidence)
	if err != nil {
		return nil, "", err
	}
	if err := validateCourseSourceDescriptionReviewEvidenceBlock(block, reviewID, reviewer, inputs, len(catalog.Pages)); err != nil {
		return nil, "", err
	}
	gate := CourseSourceDescriptionReviewGate{SchemaVersion: courseSourceDescriptionReviewSchemaVersion, ReviewID: reviewID, Stage: courseSourceDescriptionReviewStage, Decision: "passed", Reviewer: reviewer, EvidenceSHA256: hashBytes(evidence), Inputs: inputs}
	data, err := json.MarshalIndent(gate, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("encode course source-description review gate: %w", err)
	}
	return append(data, '\n'), path, nil
}

// RequireCurrentCourseSourceDescriptionReview returns a deterministic identity
// of all current passed receipts. Missing, malformed, or unknown receipts fail
// closed; stale well-formed receipts do not count as current authority.
func RequireCurrentCourseSourceDescriptionReview(root string, catalog *Catalog) (string, error) {
	current, err := currentCourseSourceDescriptionReviewInputs(root, catalog)
	if err != nil {
		return "", err
	}
	directory := filepath.Join(root, "data", "course-seo", "source-description-reviews")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("course source-description review gate missing; complete the full source-description review and record a current gate")
	}
	if err != nil {
		return "", fmt.Errorf("read course source-description review gates: %w", err)
	}
	var identities []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gate.json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return "", fmt.Errorf("read course source-description review gate %s: %w", entry.Name(), err)
		}
		var gate CourseSourceDescriptionReviewGate
		if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &gate); err != nil {
			return "", fmt.Errorf("course source-description review gate is malformed: %s: %w", entry.Name(), err)
		}
		wantName := gate.ReviewID + ".gate.json"
		if gate.SchemaVersion != courseSourceDescriptionReviewSchemaVersion || gate.Stage != courseSourceDescriptionReviewStage || gate.Decision != "passed" || strings.TrimSpace(gate.Reviewer) == "" || !reviewIDPattern.MatchString(gate.ReviewID) || gate.EvidenceSHA256 == "" || entry.Name() != wantName {
			return "", fmt.Errorf("course source-description review gate is invalid: %s", entry.Name())
		}
		if gate.Inputs != current {
			continue
		}
		evidencePath, err := CourseSourceDescriptionReviewEvidencePath(root, gate.ReviewID)
		if err != nil {
			return "", err
		}
		evidence, err := os.ReadFile(evidencePath)
		if err != nil {
			return "", fmt.Errorf("read current course source-description review evidence %s: %w", filepath.Base(evidencePath), err)
		}
		if hashBytes(evidence) != gate.EvidenceSHA256 {
			return "", fmt.Errorf("course source-description review evidence SHA-256 does not match current gate: %s", filepath.Base(evidencePath))
		}
		block, err := parseCourseSourceDescriptionReviewEvidence(evidence)
		if err != nil {
			return "", err
		}
		if err := validateCourseSourceDescriptionReviewEvidenceBlock(block, gate.ReviewID, gate.Reviewer, current, len(catalog.Pages)); err != nil {
			return "", err
		}
		identities = append(identities, entry.Name()+":"+hashBytes(data))
	}
	if len(identities) == 0 {
		return "", fmt.Errorf("course source-description review gate is stale; complete the full source-description review and record a current gate")
	}
	sort.Strings(identities)
	return hashBytes([]byte(strings.Join(identities, "\n") + "\n")), nil
}

func parseCourseSourceDescriptionReviewEvidence(evidence []byte) (courseSourceDescriptionReviewEvidenceBlock, error) {
	startMarker := []byte(courseSourceDescriptionReviewStartMarker)
	endMarker := []byte(courseSourceDescriptionReviewEndMarker)
	if bytes.Count(evidence, startMarker) != 1 || bytes.Count(evidence, endMarker) != 1 {
		return courseSourceDescriptionReviewEvidenceBlock{}, fmt.Errorf("course source-description review evidence must contain exactly one machine-readable review identity block")
	}
	start := bytes.Index(evidence, startMarker) + len(startMarker)
	end := bytes.Index(evidence, endMarker)
	if start > end {
		return courseSourceDescriptionReviewEvidenceBlock{}, fmt.Errorf("course source-description review evidence markers are out of order")
	}
	body := bytes.TrimSpace(evidence[start:end])
	if len(body) == 0 {
		return courseSourceDescriptionReviewEvidenceBlock{}, fmt.Errorf("course source-description review evidence identity block is empty")
	}
	var block courseSourceDescriptionReviewEvidenceBlock
	if err := decodeStrictCourseSourceDescriptionReviewJSON(body, &block); err != nil {
		return courseSourceDescriptionReviewEvidenceBlock{}, fmt.Errorf("course source-description review evidence identity block is malformed: %w", err)
	}
	return block, nil
}

func decodeStrictCourseSourceDescriptionReviewJSON(data []byte, value any) error {
	if err := rejectDuplicateCourseSourceDescriptionReviewJSONMembers(data); err != nil {
		return err
	}
	return decodeStrictJSON(data, value)
}

func rejectDuplicateCourseSourceDescriptionReviewJSONMembers(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	return validateUniqueCourseSourceDescriptionReviewJSONMembers(decoder)
}

func validateUniqueCourseSourceDescriptionReviewJSONMembers(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("expected JSON object member name")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON object member name %q", key)
			}
			seen[key] = struct{}{}
			if err := validateUniqueCourseSourceDescriptionReviewJSONMembers(decoder); err != nil {
				return err
			}
		}
		return requireCourseSourceDescriptionReviewJSONDelimiter(decoder, '}')
	case '[':
		for decoder.More() {
			if err := validateUniqueCourseSourceDescriptionReviewJSONMembers(decoder); err != nil {
				return err
			}
		}
		return requireCourseSourceDescriptionReviewJSONDelimiter(decoder, ']')
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

func requireCourseSourceDescriptionReviewJSONDelimiter(decoder *json.Decoder, want json.Delim) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != want {
		return fmt.Errorf("expected JSON delimiter %q", want)
	}
	return nil
}

func validateCourseSourceDescriptionReviewEvidenceBlock(block courseSourceDescriptionReviewEvidenceBlock, reviewID, reviewer string, current CourseSourceDescriptionReviewInputs, pageCount int) error {
	if block.ReviewID != reviewID {
		return fmt.Errorf("course source-description review evidence review_id %q does not match %q", block.ReviewID, reviewID)
	}
	if block.Reviewer != reviewer {
		return fmt.Errorf("course source-description review evidence reviewer %q does not match %q", block.Reviewer, reviewer)
	}
	if block.Decision != "passed" {
		return fmt.Errorf("course source-description review evidence decision must be passed, got %q", block.Decision)
	}
	if block.ArtifactSHA256 != current.ArtifactSHA256 || block.CatalogSourceSHA256 != current.CatalogSourceSHA256 ||
		block.SourceDescriptionSchemaVersion != current.SchemaVersion || block.GeneratorContract != current.GeneratorContract || block.PromptVersion != current.PromptVersion {
		return fmt.Errorf("course source-description review evidence identity is stale")
	}
	if block.PageCount != pageCount {
		return fmt.Errorf("course source-description review evidence page_count=%d, want %d", block.PageCount, pageCount)
	}
	return nil
}

func coursePageCatalogIdentity(catalog *Catalog) (string, error) {
	if catalog == nil {
		return "", fmt.Errorf("catalog is required")
	}
	type identityPage struct {
		PageID       string `json:"page_id"`
		Route        string `json:"route"`
		SourceSHA256 string `json:"source_sha256"`
	}
	pages := make([]identityPage, 0, len(catalog.Pages))
	for _, page := range catalog.Pages {
		if sum(page.Source) != page.SourceSHA256 {
			return "", fmt.Errorf("%s: catalog source identity does not match complete English Page source", page.ID)
		}
		pages = append(pages, identityPage{PageID: page.ID, Route: page.Route, SourceSHA256: page.SourceSHA256})
	}
	data, err := json.Marshal(pages)
	if err != nil {
		return "", fmt.Errorf("encode course Page catalog identity: %w", err)
	}
	return hashBytes(data), nil
}

func decodeStrictJSON(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func validateCourseGeneration(pageID string, generation CourseMetadataGeneration) error {
	if strings.TrimSpace(generation.Provider) == "" || strings.TrimSpace(generation.Model) == "" {
		return fmt.Errorf("%s: generation provider and model are required", pageID)
	}
	generatedAt, err := time.Parse(time.RFC3339, generation.GeneratedAt)
	if err != nil || generatedAt.Location() != time.UTC {
		return fmt.Errorf("%s: generated_at must be RFC 3339 UTC", pageID)
	}
	return nil
}

func requireExactCourseDescriptionSet(descriptions map[string]string, catalog *Catalog, label string) error {
	expected := make(map[string]struct{}, len(catalog.Pages))
	for _, page := range catalog.Pages {
		expected[page.ID] = struct{}{}
	}
	for pageID := range descriptions {
		if _, ok := expected[pageID]; !ok {
			return fmt.Errorf("%s has extra page_id %q", label, pageID)
		}
	}
	var missing []string
	for _, page := range catalog.Pages {
		if _, ok := descriptions[page.ID]; !ok {
			missing = append(missing, page.ID)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%s is missing page(s): %s", label, strings.Join(missing, ", "))
	}
	return nil
}
