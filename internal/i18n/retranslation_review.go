package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const TranslationReviewSchemaVersion = 1

type RetranslationReviewCheckOptions struct {
	Locale  string
	BatchID string
}

// TranslationReview is human-authored quality-review evidence. It is kept
// separate from RetranslationValidation, which records mechanical validation.
type TranslationReview struct {
	SchemaVersion    int      `json:"schema_version"`
	BatchID          string   `json:"batch_id"`
	Locale           string   `json:"locale"`
	UnitID           string   `json:"unit_id"`
	UnitKind         UnitKind `json:"unit_kind"`
	SourceSHA256     string   `json:"source_sha256"`
	Attempt          int      `json:"attempt"`
	CandidatePath    string   `json:"candidate_path"`
	CandidateSHA256  string   `json:"candidate_sha256"`
	ValidationPath   string   `json:"validation_path"`
	ValidationSHA256 string   `json:"validation_sha256"`
	Decision         string   `json:"decision"`
	Reviewer         string   `json:"reviewer"`
	ReviewedAt       string   `json:"reviewed_at"`
	Rubric           string   `json:"rubric"`
	Rating           string   `json:"rating"`
	Summary          string   `json:"summary"`
	Issues           []string `json:"issues"`
}

type RetranslationReviewCheckResult struct {
	UnitCount int
	Approved  int
	Rejected  int
}

var translationReviewRequiredFields = []string{
	"schema_version", "batch_id", "locale", "unit_id", "unit_kind", "source_sha256", "attempt",
	"candidate_path", "candidate_sha256", "validation_path", "validation_sha256",
	"decision", "reviewer", "reviewed_at", "rubric", "rating", "summary", "issues",
}

func CheckRetranslationReviews(root string, catalog *Catalog, options RetranslationReviewCheckOptions) (*RetranslationReviewCheckResult, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if options.Locale == "" || options.BatchID == "" {
		return nil, errors.New("retranslation review locale and batch_id are required")
	}
	if options.Locale != "zh-CN" {
		return nil, fmt.Errorf("unsupported locale %q", options.Locale)
	}
	if err := validateBatchID(options.BatchID); err != nil {
		return nil, err
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", options.Locale, options.BatchID)
	manifest, err := readRetranslationProcessManifest(batchDir, options.Locale, options.BatchID)
	if err != nil {
		return nil, err
	}
	resultData, err := os.ReadFile(filepath.Join(batchDir, "result.json"))
	if err != nil {
		return nil, fmt.Errorf("read retranslation result for %q: %w", options.BatchID, err)
	}
	result, err := decodeRetranslationProcessResult(resultData)
	if err != nil {
		return nil, fmt.Errorf("parse retranslation result for %q: %w", options.BatchID, err)
	}
	if result.BatchID != options.BatchID || result.Locale != options.Locale || result.UnitCount != len(result.Units) || result.UnitCount != len(manifest.Units) {
		return nil, fmt.Errorf("retranslation batch %q has incompatible process result", options.BatchID)
	}
	results := make(map[string]RetranslationUnitResult, len(result.Units))
	for _, unitResult := range result.Units {
		if _, exists := results[unitResult.UnitID]; exists {
			return nil, fmt.Errorf("duplicate process result unit %q", unitResult.UnitID)
		}
		results[unitResult.UnitID] = unitResult
	}

	report := &RetranslationReviewCheckResult{UnitCount: len(manifest.Units)}
	for _, record := range manifest.Units {
		unit, err := catalog.Unit(record.UnitID)
		if err != nil {
			return nil, fmt.Errorf("review %s: unit identity mismatch: %w", record.UnitID, err)
		}
		unitResult, ok := results[record.UnitID]
		if !ok || unitResult.UnitKind != record.UnitKind || unit.Kind != record.UnitKind {
			return nil, fmt.Errorf("review %s: unit identity mismatch", record.UnitID)
		}
		reviewPath := filepath.Join(batchDir, "review", retranslationReviewName(unit))
		reviewData, err := os.ReadFile(reviewPath)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("missing review for %s", record.UnitID)
		}
		if err != nil {
			return nil, fmt.Errorf("read review for %s: %w", record.UnitID, err)
		}
		review, err := decodeTranslationReview(reviewData)
		if err != nil {
			return nil, fmt.Errorf("review %s: invalid JSON schema: %w", record.UnitID, err)
		}
		if review.SchemaVersion != TranslationReviewSchemaVersion {
			return nil, fmt.Errorf("review %s: invalid JSON schema version %d", record.UnitID, review.SchemaVersion)
		}
		if review.BatchID != options.BatchID || review.Locale != options.Locale || review.UnitID != record.UnitID || review.UnitKind != record.UnitKind {
			return nil, fmt.Errorf("review %s: unit identity mismatch", record.UnitID)
		}
		if review.SourceSHA256 != record.SourceSHA256 || review.SourceSHA256 != unit.SourceSHA256 || sum(unit.Source) != review.SourceSHA256 {
			return nil, fmt.Errorf("review %s: source_sha256 mismatch", record.UnitID)
		}
		if review.Decision != "approved" && review.Decision != "rejected" {
			return nil, fmt.Errorf("review %s: invalid decision %q", record.UnitID, review.Decision)
		}
		if review.Rating != "A" && review.Rating != "B" && review.Rating != "C" && review.Rating != "D" {
			return nil, fmt.Errorf("review %s: invalid rating %q", record.UnitID, review.Rating)
		}
		if review.Attempt < 1 || review.Reviewer == "" || review.ReviewedAt == "" || review.Rubric == "" || review.Summary == "" || review.Issues == nil {
			return nil, fmt.Errorf("review %s: invalid JSON schema: empty required value", record.UnitID)
		}
		if err := checkReviewFileHash(batchDir, record.UnitID, "candidate", review.CandidatePath, unitResult.CandidatePath, review.CandidateSHA256); err != nil {
			return nil, err
		}
		if err := checkReviewFileHash(batchDir, record.UnitID, "validation", review.ValidationPath, unitResult.ValidationPath, review.ValidationSHA256); err != nil {
			return nil, err
		}
		validationData, _ := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(review.ValidationPath)))
		validation, err := decodeRetranslationValidation(validationData, unit)
		if err != nil || validation.Attempt != review.Attempt {
			return nil, fmt.Errorf("review %s: attempt does not match validation", record.UnitID)
		}
		if review.Decision == "approved" {
			report.Approved++
		} else {
			report.Rejected++
		}
	}
	return report, nil
}

func decodeTranslationReview(data []byte) (*TranslationReview, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	for _, field := range translationReviewRequiredFields {
		if _, ok := fields[field]; !ok {
			return nil, fmt.Errorf("missing required field %q", field)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var review TranslationReview
	if err := decoder.Decode(&review); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("multiple JSON values")
	}
	return &review, nil
}

func checkReviewFileHash(batchDir, unitID, kind, gotPath, wantPath, wantHash string) error {
	if gotPath != wantPath || gotPath == "" || filepath.IsAbs(gotPath) || strings.Contains(filepath.ToSlash(gotPath), "../") {
		return fmt.Errorf("review %s: %s path mismatch", unitID, kind)
	}
	data, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(gotPath)))
	if os.IsNotExist(err) {
		return fmt.Errorf("review %s: %s file does not exist", unitID, kind)
	}
	if err != nil {
		return fmt.Errorf("review %s: read %s: %w", unitID, kind, err)
	}
	if sum(data) != wantHash {
		return fmt.Errorf("review %s: %s hash mismatch", unitID, kind)
	}
	return nil
}

func retranslationReviewName(unit *TranslationUnit) string {
	return strings.TrimSuffix(retranslationUnitCandidateName(unit), filepath.Ext(retranslationUnitCandidateName(unit))) + ".json"
}
