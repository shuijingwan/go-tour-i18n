package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	GlossaryReviewSchemaVersion = 1
	GlossaryReviewRubric        = "glossary-review/v1"
	glossaryReviewEvidenceType  = "glossary-review-receipt"
	glossaryReviewStage         = "locale-glossary-review"

	legacyGlossaryReviewCoverageSchemaVersion = 1
	legacyGlossaryReviewCoverageEvidenceType  = "legacy-glossary-review-coverage"
	legacyGlossaryReviewCoverageStage         = "pre-glossary-review-live-locale-migration"
	legacyGlossaryReviewCoveragePath          = "data/glossary-review-legacy-coverage.json"
)

var glossaryReviewReceiptNamePattern = regexp.MustCompile(`\.review\.json$`)

// GlossaryReviewReceipt records the formal result of one complete independent
// review of a locale glossary. Hashes are always calculated by the CLI.
type GlossaryReviewReceipt struct {
	SchemaVersion  int      `json:"schema_version"`
	EvidenceType   string   `json:"evidence_type"`
	Locale         string   `json:"locale"`
	ReviewID       string   `json:"review_id"`
	Stage          string   `json:"stage"`
	Decision       string   `json:"decision"`
	Reviewer       string   `json:"reviewer"`
	Rubric         string   `json:"rubric"`
	GlossaryPath   string   `json:"glossary_path"`
	GlossarySHA256 string   `json:"glossary_sha256"`
	Findings       []string `json:"findings,omitempty"`
}

type legacyGlossaryReviewCoverage struct {
	SchemaVersion int                                 `json:"schema_version"`
	EvidenceType  string                              `json:"evidence_type"`
	Stage         string                              `json:"stage"`
	Entries       []legacyGlossaryReviewCoverageEntry `json:"entries"`
}

type legacyGlossaryReviewCoverageEntry struct {
	Locale                  string `json:"locale"`
	GlossaryPath            string `json:"glossary_path"`
	GlossarySHA256          string `json:"glossary_sha256"`
	SurfaceReviewID         string `json:"surface_review_id"`
	SurfaceReviewGatePath   string `json:"surface_review_gate_path"`
	SurfaceReviewGateSHA256 string `json:"surface_review_gate_sha256"`
}

func glossaryReviewGlossaryPath(locale string) string {
	return filepath.ToSlash(filepath.Join("locales", locale, "glossary.yaml"))
}

func GlossaryReviewReceiptPath(root, locale, reviewID string) (string, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return "", err
	}
	if !reviewIDPattern.MatchString(reviewID) {
		return "", fmt.Errorf("invalid review_id %q", reviewID)
	}
	return filepath.Join(root, "data", "glossary-reviews", locale, reviewID+".review.json"), nil
}

func RecordGlossaryReview(root, locale, reviewID, reviewer, decision string, findings []string) (*GlossaryReviewReceipt, string, error) {
	if strings.TrimSpace(reviewer) == "" {
		return nil, "", fmt.Errorf("--reviewer is required")
	}
	if decision != "passed" && decision != "failed" {
		return nil, "", fmt.Errorf("--decision must be passed or failed")
	}
	cleanFindings := make([]string, 0, len(findings))
	for _, finding := range findings {
		if strings.TrimSpace(finding) == "" {
			return nil, "", fmt.Errorf("--finding must not be empty")
		}
		cleanFindings = append(cleanFindings, finding)
	}
	if decision == "failed" && len(cleanFindings) == 0 {
		return nil, "", fmt.Errorf("failed Glossary Review requires at least one --finding")
	}
	if decision == "passed" && len(cleanFindings) != 0 {
		return nil, "", fmt.Errorf("passed Glossary Review must not contain --finding")
	}
	path, err := GlossaryReviewReceiptPath(root, locale, reviewID)
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Lstat(path); err == nil {
		return nil, "", fmt.Errorf("Glossary Review receipt already exists: %s", filepath.ToSlash(path))
	} else if !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("inspect Glossary Review receipt: %w", err)
	}
	glossaryPath := glossaryReviewGlossaryPath(locale)
	if _, err := LoadGlossary(root, locale); err != nil {
		return nil, "", fmt.Errorf("validate Glossary Review input %s: %w", glossaryPath, err)
	}
	glossary, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(glossaryPath)))
	if err != nil {
		return nil, "", fmt.Errorf("read Glossary Review input %s: %w", glossaryPath, err)
	}
	receipt := &GlossaryReviewReceipt{
		SchemaVersion:  GlossaryReviewSchemaVersion,
		EvidenceType:   glossaryReviewEvidenceType,
		Locale:         locale,
		ReviewID:       reviewID,
		Stage:          glossaryReviewStage,
		Decision:       decision,
		Reviewer:       reviewer,
		Rubric:         GlossaryReviewRubric,
		GlossaryPath:   glossaryPath,
		GlossarySHA256: hashBytes(glossary),
		Findings:       cleanFindings,
	}
	data, err := marshalGlossaryReviewJSON(receipt)
	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, "", fmt.Errorf("create Glossary Review directory: %w", err)
	}
	if err := writeGlossaryReviewAtomic(path, data); err != nil {
		return nil, "", err
	}
	return receipt, path, nil
}

// RequireCurrentGlossaryReview accepts exactly one current authority: either a
// passed formal receipt or one fixed legacy migration entry. It never derives
// legacy coverage dynamically from arbitrary Surface Review gates.
func RequireCurrentGlossaryReview(root, locale string) error {
	if err := ValidateLocaleName(locale); err != nil {
		return err
	}
	glossaryPath := glossaryReviewGlossaryPath(locale)
	if _, err := LoadGlossary(root, locale); err != nil {
		return fmt.Errorf("validate Glossary Review input %s: %w", glossaryPath, err)
	}
	glossary, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(glossaryPath)))
	if err != nil {
		return fmt.Errorf("read Glossary Review input %s: %w", glossaryPath, err)
	}
	currentSHA := hashBytes(glossary)

	receipts, err := currentGlossaryReviewReceipts(root, locale, glossaryPath, currentSHA)
	if err != nil {
		return err
	}
	legacy, err := currentLegacyGlossaryReviewCoverage(root, locale, glossaryPath, currentSHA)
	if err != nil {
		return err
	}
	authorities := len(receipts)
	if legacy {
		authorities++
	}
	if authorities == 0 {
		return fmt.Errorf("Glossary Review gate missing or stale for %s; complete the full Glossary Review and record a current passed receipt", locale)
	}
	if authorities != 1 {
		return fmt.Errorf("Glossary Review authority is ambiguous for %s: found %d current authorities", locale, authorities)
	}
	if len(receipts) == 1 && receipts[0].Decision != "passed" {
		return fmt.Errorf("Glossary Review gate is non-passed for %s review_id=%s", locale, receipts[0].ReviewID)
	}
	return nil
}

func currentGlossaryReviewReceipts(root, locale, glossaryPath, glossarySHA string) ([]GlossaryReviewReceipt, error) {
	directory := filepath.Join(root, "data", "glossary-reviews", locale)
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Glossary Review receipts: %w", err)
	}
	var current []GlossaryReviewReceipt
	for _, entry := range entries {
		if entry.IsDir() || !glossaryReviewReceiptNamePattern.MatchString(entry.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read Glossary Review receipt %s: %w", entry.Name(), err)
		}
		var receipt GlossaryReviewReceipt
		if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &receipt); err != nil {
			return nil, fmt.Errorf("Glossary Review receipt is malformed: %s: %w", entry.Name(), err)
		}
		if err := validateGlossaryReviewReceipt(receipt, locale, glossaryPath, entry.Name()); err != nil {
			return nil, err
		}
		if receipt.GlossarySHA256 == glossarySHA {
			current = append(current, receipt)
		}
	}
	sort.Slice(current, func(i, j int) bool { return current[i].ReviewID < current[j].ReviewID })
	return current, nil
}

func validateGlossaryReviewReceipt(receipt GlossaryReviewReceipt, locale, glossaryPath, filename string) error {
	if receipt.SchemaVersion != GlossaryReviewSchemaVersion {
		return fmt.Errorf("Glossary Review receipt schema/version mismatch: %s", filename)
	}
	if receipt.EvidenceType != glossaryReviewEvidenceType || receipt.Stage != glossaryReviewStage {
		return fmt.Errorf("Glossary Review receipt evidence/stage mismatch: %s", filename)
	}
	if receipt.Locale != locale {
		return fmt.Errorf("Glossary Review receipt has wrong locale %q, want %q: %s", receipt.Locale, locale, filename)
	}
	if !reviewIDPattern.MatchString(receipt.ReviewID) || filename != receipt.ReviewID+".review.json" {
		return fmt.Errorf("Glossary Review receipt review identity/path mismatch: %s", filename)
	}
	if strings.TrimSpace(receipt.Reviewer) == "" {
		return fmt.Errorf("Glossary Review receipt has empty reviewer: %s", filename)
	}
	if receipt.Rubric != GlossaryReviewRubric {
		return fmt.Errorf("Glossary Review receipt rubric/version mismatch: %s", filename)
	}
	if receipt.GlossaryPath != glossaryPath {
		return fmt.Errorf("Glossary Review receipt glossary path mismatch: %s", filename)
	}
	if !validSHA256(receipt.GlossarySHA256) {
		return fmt.Errorf("Glossary Review receipt has invalid glossary SHA-256: %s", filename)
	}
	if receipt.Decision != "passed" && receipt.Decision != "failed" {
		return fmt.Errorf("Glossary Review receipt has invalid decision %q: %s", receipt.Decision, filename)
	}
	if receipt.Decision == "failed" && len(receipt.Findings) == 0 {
		return fmt.Errorf("failed Glossary Review receipt has no findings: %s", filename)
	}
	if receipt.Decision == "passed" && len(receipt.Findings) != 0 {
		return fmt.Errorf("passed Glossary Review receipt unexpectedly has findings: %s", filename)
	}
	for _, finding := range receipt.Findings {
		if strings.TrimSpace(finding) == "" {
			return fmt.Errorf("Glossary Review receipt has empty finding: %s", filename)
		}
	}
	return nil
}

func currentLegacyGlossaryReviewCoverage(root, locale, glossaryPath, glossarySHA string) (bool, error) {
	path := filepath.Join(root, filepath.FromSlash(legacyGlossaryReviewCoveragePath))
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read legacy Glossary Review coverage: %w", err)
	}
	var coverage legacyGlossaryReviewCoverage
	if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &coverage); err != nil {
		return false, fmt.Errorf("legacy Glossary Review coverage is malformed: %w", err)
	}
	if coverage.SchemaVersion != legacyGlossaryReviewCoverageSchemaVersion || coverage.EvidenceType != legacyGlossaryReviewCoverageEvidenceType || coverage.Stage != legacyGlossaryReviewCoverageStage {
		return false, fmt.Errorf("legacy Glossary Review coverage schema/evidence/stage mismatch")
	}
	seen := map[string]bool{}
	var selected *legacyGlossaryReviewCoverageEntry
	for _, entry := range coverage.Entries {
		if err := ValidateLocaleName(entry.Locale); err != nil {
			return false, fmt.Errorf("legacy Glossary Review coverage has invalid locale %q", entry.Locale)
		}
		if seen[entry.Locale] {
			return false, fmt.Errorf("legacy Glossary Review coverage has duplicate locale %s", entry.Locale)
		}
		seen[entry.Locale] = true
		if entry.GlossaryPath != glossaryReviewGlossaryPath(entry.Locale) || !validSHA256(entry.GlossarySHA256) || !reviewIDPattern.MatchString(entry.SurfaceReviewID) || !validSHA256(entry.SurfaceReviewGateSHA256) {
			return false, fmt.Errorf("legacy Glossary Review coverage has invalid identity for %s", entry.Locale)
		}
		wantGatePath := filepath.ToSlash(filepath.Join("data", "locale-surface-reviews", entry.Locale, entry.SurfaceReviewID+".a-gate.json"))
		if entry.SurfaceReviewGatePath != wantGatePath {
			return false, fmt.Errorf("legacy Glossary Review coverage Surface Review path mismatch for %s", entry.Locale)
		}
		if entry.Locale == locale {
			entryCopy := entry
			selected = &entryCopy
		}
	}
	if selected == nil {
		return false, nil
	}
	gateData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(selected.SurfaceReviewGatePath)))
	if err != nil {
		return false, fmt.Errorf("read legacy coverage Surface Review gate for %s: %w", selected.Locale, err)
	}
	if hashBytes(gateData) != selected.SurfaceReviewGateSHA256 {
		return false, fmt.Errorf("legacy coverage Surface Review gate SHA-256 mismatch for %s", selected.Locale)
	}
	var gate LocaleSurfaceReviewAGate
	if err := decodeStrictCourseSourceDescriptionReviewJSON(gateData, &gate); err != nil {
		return false, fmt.Errorf("legacy coverage Surface Review gate is malformed for %s: %w", selected.Locale, err)
	}
	if err := validateLocaleSurfaceReviewAGate(gate, selected.Locale); err != nil || gate.ReviewID != selected.SurfaceReviewID || gate.Inputs.GlossarySHA256 != selected.GlossarySHA256 {
		return false, fmt.Errorf("legacy coverage Surface Review gate identity mismatch for %s", selected.Locale)
	}
	return selected.GlossaryPath == glossaryPath && selected.GlossarySHA256 == glossarySHA, nil
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func marshalGlossaryReviewJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode Glossary Review receipt: %w", err)
	}
	return append(data, '\n'), nil
}

func writeGlossaryReviewAtomic(path string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".glossary-review-*")
	if err != nil {
		return fmt.Errorf("create temporary Glossary Review receipt: %w", err)
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(data); err == nil {
		err = temp.Chmod(0644)
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write Glossary Review receipt: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("commit Glossary Review receipt: %w", err)
	}
	return nil
}
