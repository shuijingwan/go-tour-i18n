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
	"time"
)

const TranslationReviewSchemaVersion = 1

const DefaultRetranslationReviewBatchLimit = 20

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
	SourcePath       string   `json:"source_path,omitempty"`
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

type RetranslationReviewRecordOptions struct {
	Locale   string
	BatchID  string
	UnitID   string
	Rating   string
	Decision string
	Summary  string
	Issues   []string
	Reviewer string
	Rubric   string
	Now      func() time.Time
}

type RetranslationReviewBatchRecordOptions struct {
	Locale     string
	SnapshotID string
	StartIndex int
	Limit      int
	Rating     string
	Decision   string
	Summary    string
	Issues     []string
	Reviewer   string
	Rubric     string
	Now        func() time.Time
}

type RetranslationReviewBatchRecord struct {
	Index   int    `json:"index"`
	UnitID  string `json:"unit_id"`
	BatchID string `json:"batch_id"`
	Path    string `json:"path"`
}

type RetranslationReviewBatchRecordResult struct {
	Locale        string                           `json:"locale"`
	SnapshotID    string                           `json:"snapshot_id"`
	StartIndex    int                              `json:"start_index"`
	EndIndex      int                              `json:"end_index"`
	Limit         int                              `json:"limit"`
	RecordedCount int                              `json:"recorded_count"`
	Reviews       []RetranslationReviewBatchRecord `json:"reviews"`
}

type preparedRetranslationReview struct {
	review         TranslationReview
	path           string
	repositoryPath string
}

// RecordRetranslationReview creates schema-v1 review evidence from the batch's
// immutable manifest, process result, candidate, and validation files. Only
// the quality-review fields are supplied by the reviewer.
func RecordRetranslationReview(root string, catalog *Catalog, options RetranslationReviewRecordOptions) (*TranslationReview, string, error) {
	prepared, err := prepareRetranslationReview(root, catalog, options)
	if err != nil {
		return nil, "", err
	}
	if err := writePreparedRetranslationReview(prepared); err != nil {
		return nil, "", err
	}
	return &prepared.review, prepared.repositoryPath, nil
}

func prepareRetranslationReview(root string, catalog *Catalog, options RetranslationReviewRecordOptions) (*preparedRetranslationReview, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if options.Locale == "" || options.BatchID == "" || options.UnitID == "" {
		return nil, errors.New("retranslation review locale, batch_id, and unit_id are required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	if err := validateBatchID(options.BatchID); err != nil {
		return nil, err
	}
	if options.Rating != "A" && options.Rating != "B" && options.Rating != "C" && options.Rating != "D" {
		return nil, fmt.Errorf("invalid rating %q", options.Rating)
	}
	if options.Decision != "approved" && options.Decision != "rejected" {
		return nil, fmt.Errorf("invalid decision %q", options.Decision)
	}
	if strings.TrimSpace(options.Summary) == "" || strings.TrimSpace(options.Reviewer) == "" || strings.TrimSpace(options.Rubric) == "" {
		return nil, errors.New("summary, reviewer, and rubric are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}

	batchDir := filepath.Join(root, "data", "retranslation-runs", options.Locale, options.BatchID)
	manifest, err := readRetranslationProcessManifest(batchDir, options.Locale, options.BatchID)
	if err != nil {
		return nil, err
	}
	var record RetranslationBatchUnit
	for _, candidate := range manifest.Units {
		if candidate.UnitID == options.UnitID {
			record = candidate
			break
		}
	}
	if record.UnitID == "" {
		return nil, fmt.Errorf("unit %q is not in batch %q", options.UnitID, options.BatchID)
	}
	unit, err := catalog.Unit(record.UnitID)
	if err != nil {
		return nil, err
	}
	resultData, err := os.ReadFile(filepath.Join(batchDir, "result.json"))
	if err != nil {
		return nil, fmt.Errorf("read retranslation result: %w", err)
	}
	result, err := decodeRetranslationProcessResult(resultData)
	if err != nil {
		return nil, err
	}
	var unitResult RetranslationUnitResult
	for _, candidate := range result.Units {
		if candidate.UnitID == options.UnitID {
			unitResult = candidate
			break
		}
	}
	if unitResult.UnitID == "" || unitResult.UnitKind != record.UnitKind {
		return nil, fmt.Errorf("unit %q is missing or has incompatible process result", options.UnitID)
	}
	candidateData, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(unitResult.CandidatePath)))
	if err != nil {
		return nil, fmt.Errorf("read candidate for %s: %w", options.UnitID, err)
	}
	validationData, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(unitResult.ValidationPath)))
	if err != nil {
		return nil, fmt.Errorf("read validation for %s: %w", options.UnitID, err)
	}
	validation, err := decodeRetranslationValidation(validationData, unit)
	if err != nil {
		return nil, fmt.Errorf("validation for %s: %w", options.UnitID, err)
	}
	if validation.BatchID != options.BatchID || validation.Locale != options.Locale || validation.UnitID != options.UnitID || validation.UnitKind != record.UnitKind {
		return nil, fmt.Errorf("validation for %s has incompatible identity", options.UnitID)
	}
	if record.SourceSHA256 != unit.SourceSHA256 || sum(unit.Source) != record.SourceSHA256 {
		return nil, fmt.Errorf("source metadata mismatch for %s", options.UnitID)
	}
	review := TranslationReview{
		SchemaVersion: TranslationReviewSchemaVersion, BatchID: options.BatchID, Locale: options.Locale,
		UnitID: record.UnitID, UnitKind: record.UnitKind, SourcePath: record.SourcePath, SourceSHA256: record.SourceSHA256,
		Attempt: validation.Attempt, CandidatePath: unitResult.CandidatePath, CandidateSHA256: sum(candidateData),
		ValidationPath: unitResult.ValidationPath, ValidationSHA256: sum(validationData), Decision: options.Decision,
		Reviewer: options.Reviewer, ReviewedAt: options.Now().UTC().Format(time.RFC3339), Rubric: options.Rubric,
		Rating: options.Rating, Summary: options.Summary, Issues: append([]string(nil), options.Issues...),
	}
	if review.Issues == nil {
		review.Issues = []string{}
	}
	path := filepath.Join(batchDir, "review", retranslationReviewName(unit))
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("review already exists for %s", options.UnitID)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect review for %s: %w", options.UnitID, err)
	}
	return &preparedRetranslationReview{
		review:         review,
		path:           path,
		repositoryPath: filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, options.BatchID, "review", retranslationReviewName(unit))),
	}, nil
}

func writePreparedRetranslationReview(prepared *preparedRetranslationReview) error {
	if err := os.MkdirAll(filepath.Dir(prepared.path), 0755); err != nil {
		return fmt.Errorf("create review directory: %w", err)
	}
	data, err := json.MarshalIndent(prepared.review, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal review for %s: %w", prepared.review.UnitID, err)
	}
	file, err := os.OpenFile(prepared.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return fmt.Errorf("review already exists for %s", prepared.review.UnitID)
	}
	if err != nil {
		return fmt.Errorf("write review for %s: %w", prepared.review.UnitID, err)
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(prepared.path)
		}
	}()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write review for %s: %w", prepared.review.UnitID, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close review for %s: %w", prepared.review.UnitID, err)
	}
	remove = false
	return nil
}

// RecordRetranslationReviewBatch records one stable Candidate Snapshot index
// range. Every unit is fully preflighted through the single-unit review path
// and checked against the snapshot before any review evidence is written.
func RecordRetranslationReviewBatch(root string, catalog *Catalog, options RetranslationReviewBatchRecordOptions) (*RetranslationReviewBatchRecordResult, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(options.SnapshotID); err != nil {
		return nil, err
	}
	startIndex := options.StartIndex
	if startIndex == 0 {
		startIndex = 1
	}
	if startIndex < 1 {
		return nil, fmt.Errorf("start_index must be at least 1, got %d", options.StartIndex)
	}
	limit := options.Limit
	if limit == 0 {
		limit = DefaultRetranslationReviewBatchLimit
	}
	if limit < 1 {
		return nil, fmt.Errorf("limit must be at least 1, got %d", options.Limit)
	}

	snapshot, err := readQualityCheckSnapshotForReview(root, options.Locale, options.SnapshotID)
	if err != nil {
		return nil, err
	}
	if startIndex > snapshot.UnitCount {
		return nil, fmt.Errorf("start_index %d is outside snapshot range 1-%d", startIndex, snapshot.UnitCount)
	}
	endIndex := startIndex + limit - 1
	if endIndex > snapshot.UnitCount || endIndex < startIndex {
		endIndex = snapshot.UnitCount
	}
	selected := snapshot.Units[startIndex-1 : endIndex]

	reviewedAt := time.Now()
	if options.Now != nil {
		reviewedAt = options.Now()
	}
	fixedNow := func() time.Time { return reviewedAt }
	prepared := make([]*preparedRetranslationReview, 0, len(selected))
	result := &RetranslationReviewBatchRecordResult{
		Locale: options.Locale, SnapshotID: options.SnapshotID,
		StartIndex: startIndex, EndIndex: endIndex, Limit: limit,
		RecordedCount: len(selected), Reviews: make([]RetranslationReviewBatchRecord, 0, len(selected)),
	}
	for _, snapshotUnit := range selected {
		item, err := prepareRetranslationReview(root, catalog, RetranslationReviewRecordOptions{
			Locale: options.Locale, BatchID: snapshotUnit.SelectedBatchID, UnitID: snapshotUnit.UnitID,
			Rating: options.Rating, Decision: options.Decision, Summary: options.Summary,
			Issues: options.Issues, Reviewer: options.Reviewer, Rubric: options.Rubric, Now: fixedNow,
		})
		if err != nil {
			return nil, fmt.Errorf("snapshot index %d (%s) preflight: %w", snapshotUnit.Index, snapshotUnit.UnitID, err)
		}
		if err := checkPreparedReviewAgainstSnapshot(options.Locale, snapshotUnit, item); err != nil {
			return nil, fmt.Errorf("snapshot index %d (%s) preflight: %w", snapshotUnit.Index, snapshotUnit.UnitID, err)
		}
		prepared = append(prepared, item)
		result.Reviews = append(result.Reviews, RetranslationReviewBatchRecord{
			Index: snapshotUnit.Index, UnitID: item.review.UnitID, BatchID: item.review.BatchID, Path: item.repositoryPath,
		})
	}

	written := make([]string, 0, len(prepared))
	for _, item := range prepared {
		if err := writePreparedRetranslationReview(item); err != nil {
			rollbackErrors := []error{err}
			for i := len(written) - 1; i >= 0; i-- {
				if removeErr := os.Remove(written[i]); removeErr != nil && !os.IsNotExist(removeErr) {
					rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback %s: %w", filepath.ToSlash(written[i]), removeErr))
				}
			}
			return nil, fmt.Errorf("commit review batch: %w", errors.Join(rollbackErrors...))
		}
		written = append(written, item.path)
	}
	return result, nil
}

func readQualityCheckSnapshotForReview(root, locale, snapshotID string) (*QualityCheckSnapshotManifest, error) {
	path := filepath.Join(root, "data", "quality-check-snapshots", locale, snapshotID, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read quality-check snapshot %q: %w", snapshotID, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest QualityCheckSnapshotManifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("parse quality-check snapshot %q: %w", snapshotID, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("parse quality-check snapshot %q: multiple JSON values", snapshotID)
	}
	if manifest.SchemaVersion != QualityCheckSnapshotSchemaVersion || manifest.SnapshotID != snapshotID || manifest.Locale != locale {
		return nil, fmt.Errorf("quality-check snapshot %q has incompatible identity", snapshotID)
	}
	if manifest.UnitCount < 1 || manifest.UnitCount != len(manifest.Units) || manifest.PageCount+manifest.ExampleCount != manifest.UnitCount {
		return nil, fmt.Errorf("quality-check snapshot %q has incompatible unit counts", snapshotID)
	}
	wantGlossaryPath := filepath.ToSlash(filepath.Join("locales", locale, "glossary.yaml"))
	if manifest.GlossaryPath != wantGlossaryPath || manifest.GlossarySHA256 == "" {
		return nil, fmt.Errorf("quality-check snapshot %q has incompatible glossary identity", snapshotID)
	}
	glossaryData, err := readSnapshotReferencedFile(root, manifest.GlossaryPath)
	if err != nil {
		return nil, fmt.Errorf("quality-check snapshot %q glossary: %w", snapshotID, err)
	}
	if sum(glossaryData) != manifest.GlossarySHA256 {
		return nil, fmt.Errorf("quality-check snapshot %q glossary hash mismatch", snapshotID)
	}
	seen := make(map[string]bool, len(manifest.Units))
	pageCount, exampleCount := 0, 0
	for i, unit := range manifest.Units {
		if unit.Index != i+1 {
			return nil, fmt.Errorf("quality-check snapshot %q has unstable index %d at position %d", snapshotID, unit.Index, i+1)
		}
		if unit.UnitID == "" || seen[unit.UnitID] {
			return nil, fmt.Errorf("quality-check snapshot %q has empty or duplicate unit_id %q", snapshotID, unit.UnitID)
		}
		seen[unit.UnitID] = true
		switch unit.UnitKind {
		case UnitKindPage:
			pageCount++
		case UnitKindExample:
			exampleCount++
		default:
			return nil, fmt.Errorf("quality-check snapshot %q unit %s has invalid unit_kind %q", snapshotID, unit.UnitID, unit.UnitKind)
		}
		if err := validateBatchID(unit.SelectedBatchID); err != nil {
			return nil, fmt.Errorf("quality-check snapshot %q unit %s: %w", snapshotID, unit.UnitID, err)
		}
		if unit.SourcePath == "" || unit.SourceSHA256 == "" || unit.CandidatePath == "" || unit.CandidateSHA256 == "" ||
			unit.ValidationPath == "" || unit.ValidationSHA256 == "" || unit.Attempt < 1 {
			return nil, fmt.Errorf("quality-check snapshot %q unit %s has empty evidence identity", snapshotID, unit.UnitID)
		}
	}
	if pageCount != manifest.PageCount || exampleCount != manifest.ExampleCount {
		return nil, fmt.Errorf("quality-check snapshot %q has incompatible kind counts", snapshotID)
	}
	return &manifest, nil
}

func checkPreparedReviewAgainstSnapshot(locale string, snapshot QualityCheckSnapshotUnit, prepared *preparedRetranslationReview) error {
	review := prepared.review
	wantCandidatePath := filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, review.BatchID, review.CandidatePath))
	wantValidationPath := filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, review.BatchID, review.ValidationPath))
	if review.BatchID != snapshot.SelectedBatchID || review.UnitID != snapshot.UnitID || review.UnitKind != snapshot.UnitKind ||
		review.SourcePath != snapshot.SourcePath || review.SourceSHA256 != snapshot.SourceSHA256 || review.Attempt != snapshot.Attempt {
		return errors.New("review identity does not match Candidate Snapshot")
	}
	if wantCandidatePath != snapshot.CandidatePath || review.CandidateSHA256 != snapshot.CandidateSHA256 {
		return errors.New("candidate path or hash does not match Candidate Snapshot")
	}
	if wantValidationPath != snapshot.ValidationPath || review.ValidationSHA256 != snapshot.ValidationSHA256 {
		return errors.New("validation path or hash does not match Candidate Snapshot")
	}
	return nil
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
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
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
