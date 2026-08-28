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

// TranslationQualityRubric is the current production rubric. Changing the
// identifier deliberately invalidates prior review evidence for reuse.
const TranslationQualityRubric = "translation-quality/v1"

type RetranslationReviewScopeReason string

const (
	ReviewScopeReasonMissingReview   RetranslationReviewScopeReason = "missing_review"
	ReviewScopeReasonIdentityChanged RetranslationReviewScopeReason = "identity_changed"
	ReviewScopeReasonRubricMismatch  RetranslationReviewScopeReason = "rubric_mismatch"
	ReviewScopeReasonNonAReview      RetranslationReviewScopeReason = "non_a_review"
	ReviewScopeReasonRejectedReview  RetranslationReviewScopeReason = "rejected_review"
)

type RetranslationReviewRequiredAction string

const (
	ReviewScopeActionReviewRequired   RetranslationReviewRequiredAction = "review_required"
	ReviewScopeActionRevisionRequired RetranslationReviewRequiredAction = "revision_required"
)

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

// RetranslationReviewSupersedeOptions is deliberately separate from normal
// record options: it is only for renewing an otherwise-identical A + approved
// review whose rubric has expired.
type RetranslationReviewSupersedeOptions struct {
	Locale     string
	SnapshotID string
	UnitID     string
	Rating     string
	Decision   string
	Summary    string
	Issues     []string
	Reviewer   string
	Rubric     string
	Now        func() time.Time
}

type RetranslationReviewSupersedeResult struct {
	Locale      string `json:"locale"`
	SnapshotID  string `json:"snapshot_id"`
	UnitID      string `json:"unit_id"`
	BatchID     string `json:"batch_id"`
	CurrentPath string `json:"current_path"`
	HistoryPath string `json:"history_path"`
}

// RetranslationReviewScope is the review work derived from a complete,
// immutable Candidate Snapshot. The Snapshot is never narrowed; only the
// reviewer work set excludes valid, reusable Final Review evidence.
type RetranslationReviewScope struct {
	Locale        string                         `json:"locale"`
	SnapshotID    string                         `json:"snapshot_id"`
	UnitCount     int                            `json:"unit_count"`
	ReusableCount int                            `json:"reusable_count"`
	PendingCount  int                            `json:"pending_count"`
	Reusable      []RetranslationReviewScopeUnit `json:"-"`
	Pending       []RetranslationReviewScopeUnit `json:"pending_review_units"`
}

type RetranslationReviewScopeUnit struct {
	Index          int                               `json:"index"`
	UnitID         string                            `json:"unit_id"`
	UnitKind       UnitKind                          `json:"unit_kind"`
	BatchID        string                            `json:"batch_id"`
	Reason         RetranslationReviewScopeReason    `json:"reason,omitempty"`
	RequiredAction RetranslationReviewRequiredAction `json:"required_action,omitempty"`
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
	return prepareRetranslationReviewAllowExisting(root, catalog, options, false)
}

func prepareRetranslationReviewAllowExisting(root string, catalog *Catalog, options RetranslationReviewRecordOptions, allowExisting bool) (*preparedRetranslationReview, error) {
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
	if strings.TrimSpace(options.Summary) == "" || strings.TrimSpace(options.Reviewer) == "" || options.Rubric != TranslationQualityRubric {
		return nil, fmt.Errorf("summary and reviewer are required; rubric must be %q", TranslationQualityRubric)
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
		if !allowExisting {
			return nil, fmt.Errorf("review already exists for %s", options.UnitID)
		}
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

// SupersedeRetranslationReview renews exactly one rubric-expired A + approved
// review after a reviewer has performed a new Final Review. It is not a
// revision path: all candidate identity fields must still match the Snapshot.
func SupersedeRetranslationReview(root string, catalog *Catalog, options RetranslationReviewSupersedeOptions) (*RetranslationReviewSupersedeResult, error) {
	if options.Locale == "" || options.SnapshotID == "" || options.UnitID == "" {
		return nil, errors.New("retranslation review supersede locale, snapshot_id, and unit_id are required")
	}
	if options.Rating != "A" || options.Decision != "approved" {
		return nil, errors.New("retranslation review supersede requires rating A and decision approved")
	}
	scope, err := BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: options.Locale, SnapshotID: options.SnapshotID})
	if err != nil {
		return nil, err
	}
	var pending *RetranslationReviewScopeUnit
	for i := range scope.Pending {
		if scope.Pending[i].UnitID == options.UnitID {
			pending = &scope.Pending[i]
			break
		}
	}
	if pending == nil {
		return nil, fmt.Errorf("unit %q is not pending review in snapshot %q", options.UnitID, options.SnapshotID)
	}
	if pending.Reason != ReviewScopeReasonRubricMismatch || pending.RequiredAction != ReviewScopeActionReviewRequired {
		return nil, fmt.Errorf("unit %q is %s/%s; only rubric_mismatch/review_required may be superseded", options.UnitID, pending.Reason, pending.RequiredAction)
	}
	snapshotUnit, err := snapshotUnitForReviewScope(root, options.Locale, options.SnapshotID, options.UnitID)
	if err != nil {
		return nil, err
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", options.Locale, snapshotUnit.SelectedBatchID)
	unit, err := catalog.Unit(options.UnitID)
	if err != nil {
		return nil, err
	}
	currentPath := filepath.Join(batchDir, "review", retranslationReviewName(unit))
	oldData, err := os.ReadFile(currentPath)
	if err != nil {
		return nil, fmt.Errorf("read current review for supersede: %w", err)
	}
	old, err := decodeTranslationReview(oldData)
	if err != nil || !reviewMatchesSnapshotIdentity(options.Locale, snapshotUnit, *old) || old.Rating != "A" || old.Decision != "approved" || old.Rubric == TranslationQualityRubric {
		return nil, errors.New("current review is not an identity-matching rubric-expired A + approved review")
	}
	prepared, err := prepareRetranslationReviewAllowExisting(root, catalog, RetranslationReviewRecordOptions{
		Locale: options.Locale, BatchID: snapshotUnit.SelectedBatchID, UnitID: options.UnitID,
		Rating: options.Rating, Decision: options.Decision, Summary: options.Summary, Issues: options.Issues,
		Reviewer: options.Reviewer, Rubric: options.Rubric, Now: options.Now,
	}, true)
	if err != nil {
		return nil, err
	}
	if err := checkPreparedReviewAgainstSnapshot(options.Locale, snapshotUnit, prepared); err != nil {
		return nil, err
	}
	when := time.Now()
	if options.Now != nil {
		when = options.Now()
	}
	historyName := strings.TrimSuffix(filepath.Base(currentPath), ".json") + ".superseded-" + when.UTC().Format("20060102T150405.000000000Z") + ".json"
	historyPath := filepath.Join(batchDir, "review", "history", historyName)
	if err := os.MkdirAll(filepath.Dir(historyPath), 0755); err != nil {
		return nil, fmt.Errorf("create review history directory: %w", err)
	}
	history, err := os.OpenFile(historyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, fmt.Errorf("archive current review: %w", err)
	}
	if _, err := history.Write(oldData); err != nil {
		_ = history.Close()
		_ = os.Remove(historyPath)
		return nil, fmt.Errorf("archive current review: %w", err)
	}
	if err := history.Close(); err != nil {
		return nil, fmt.Errorf("archive current review: %w", err)
	}
	newData, err := json.MarshalIndent(prepared.review, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal superseding review: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(currentPath), ".review-supersede-*")
	if err != nil {
		return nil, fmt.Errorf("create superseding review: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(append(newData, '\n')); err != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("write superseding review: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("close superseding review: %w", err)
	}
	if err := os.Rename(temporaryPath, currentPath); err != nil {
		return nil, fmt.Errorf("replace superseding review: %w", err)
	}
	return &RetranslationReviewSupersedeResult{
		Locale: options.Locale, SnapshotID: options.SnapshotID, UnitID: options.UnitID, BatchID: snapshotUnit.SelectedBatchID,
		CurrentPath: prepared.repositoryPath,
		HistoryPath: filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, snapshotUnit.SelectedBatchID, "review", "history", historyName)),
	}, nil
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
		return nil, fmt.Errorf("start_index %d is outside Candidate Snapshot range 1-%d", startIndex, snapshot.UnitCount)
	}
	endIndex := startIndex + limit - 1
	if endIndex > snapshot.UnitCount || endIndex < startIndex {
		endIndex = snapshot.UnitCount
	}
	selected := snapshot.Units[startIndex-1 : endIndex]

	scope, err := BuildRetranslationReviewScope(root, catalog, RetranslationReviewScopeOptions{Locale: options.Locale, SnapshotID: options.SnapshotID})
	if err != nil {
		return nil, err
	}
	pendingByIndex := make(map[int]RetranslationReviewScopeUnit, len(scope.Pending))
	for _, pending := range scope.Pending {
		pendingByIndex[pending.Index] = pending
	}

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
		pending, isPending := pendingByIndex[snapshotUnit.Index]
		if !isPending {
			return nil, fmt.Errorf("snapshot index %d (%s) is not recordable: valid Final Review evidence already exists", snapshotUnit.Index, snapshotUnit.UnitID)
		}
		// A rubric renewal needs an explicit supersede; B/C/D and rejected
		// evidence require a revision instead of another Final Review record.
		if pending.Reason != ReviewScopeReasonMissingReview && pending.Reason != ReviewScopeReasonIdentityChanged {
			return nil, fmt.Errorf("snapshot index %d (%s) is not recordable: %s/%s", snapshotUnit.Index, snapshotUnit.UnitID, pending.Reason, pending.RequiredAction)
		}
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

type RetranslationReviewScopeOptions struct {
	Locale     string
	SnapshotID string
}

// BuildRetranslationReviewScope separates a complete Snapshot into valid
// reusable Final Reviews and the units that still require Final Review or a
// revision. Quality Check results never enter this scope. It writes nothing.
func BuildRetranslationReviewScope(root string, catalog *Catalog, options RetranslationReviewScopeOptions) (*RetranslationReviewScope, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(options.SnapshotID); err != nil {
		return nil, err
	}
	snapshot, err := readQualityCheckSnapshotForReview(root, options.Locale, options.SnapshotID)
	if err != nil {
		return nil, err
	}
	scope := &RetranslationReviewScope{Locale: options.Locale, SnapshotID: options.SnapshotID, UnitCount: snapshot.UnitCount,
		Reusable: []RetranslationReviewScopeUnit{}, Pending: []RetranslationReviewScopeUnit{}}
	for _, snapshotUnit := range snapshot.Units {
		unit := RetranslationReviewScopeUnit{Index: snapshotUnit.Index, UnitID: snapshotUnit.UnitID, UnitKind: snapshotUnit.UnitKind, BatchID: snapshotUnit.SelectedBatchID}
		if reason, action := reviewScopePendingDisposition(root, catalog, options.Locale, snapshotUnit); reason != "" {
			unit.Reason = reason
			unit.RequiredAction = action
			scope.Pending = append(scope.Pending, unit)
			continue
		}
		scope.Reusable = append(scope.Reusable, unit)
	}
	scope.ReusableCount = len(scope.Reusable)
	scope.PendingCount = len(scope.Pending)
	return scope, nil
}

func snapshotUnitForReviewScope(root, locale, snapshotID, unitID string) (QualityCheckSnapshotUnit, error) {
	snapshot, err := readQualityCheckSnapshotForReview(root, locale, snapshotID)
	if err != nil {
		return QualityCheckSnapshotUnit{}, err
	}
	for _, unit := range snapshot.Units {
		if unit.UnitID == unitID {
			return unit, nil
		}
	}
	return QualityCheckSnapshotUnit{}, fmt.Errorf("snapshot %q has no pending unit %q", snapshotID, unitID)
}

func reviewScopePendingDisposition(root string, catalog *Catalog, locale string, snapshot QualityCheckSnapshotUnit) (RetranslationReviewScopeReason, RetranslationReviewRequiredAction) {
	evidence, err := readSnapshotUnitRepositoryEvidence(root, catalog, locale, snapshot)
	if err != nil {
		return ReviewScopeReasonIdentityChanged, ReviewScopeActionReviewRequired
	}
	reviewPath := filepath.Join(evidence.batchDir, "review", retranslationReviewName(evidence.unit))
	reviewData, err := os.ReadFile(reviewPath)
	if os.IsNotExist(err) {
		return ReviewScopeReasonMissingReview, ReviewScopeActionReviewRequired
	}
	if err != nil {
		return ReviewScopeReasonIdentityChanged, ReviewScopeActionReviewRequired
	}
	review, err := decodeTranslationReview(reviewData)
	if err != nil || !reviewMatchesSnapshotIdentity(locale, snapshot, *review) {
		return ReviewScopeReasonIdentityChanged, ReviewScopeActionReviewRequired
	}
	if review.Rating != "A" {
		return ReviewScopeReasonNonAReview, ReviewScopeActionRevisionRequired
	}
	if review.Decision != "approved" {
		return ReviewScopeReasonRejectedReview, ReviewScopeActionRevisionRequired
	}
	if review.Rubric != TranslationQualityRubric {
		return ReviewScopeReasonRubricMismatch, ReviewScopeActionReviewRequired
	}
	state, err := checkPromotionReview(evidence.batchDir, snapshot.SelectedBatchID, locale, evidence.unit, evidence.record, evidence.result, evidence.validation, evidence.candidate, snapshot.Attempt)
	if err != nil {
		return ReviewScopeReasonIdentityChanged, ReviewScopeActionReviewRequired
	}
	switch state {
	case "approved":
		return "", ""
	case "missing":
		return ReviewScopeReasonMissingReview, ReviewScopeActionReviewRequired
	case "rejected":
		return ReviewScopeReasonRejectedReview, ReviewScopeActionRevisionRequired
	default:
		return ReviewScopeReasonIdentityChanged, ReviewScopeActionReviewRequired
	}
}

type snapshotUnitRepositoryEvidence struct {
	unit       *TranslationUnit
	candidate  []byte
	batchDir   string
	record     RetranslationBatchUnit
	result     RetranslationUnitResult
	validation *RetranslationValidation
}

// readSnapshotUnitRepositoryEvidence checks that a Snapshot unit still points
// to the exact current source, candidate, validation, and final attempt. Both
// Quality Check and Final Review scopes use this shared identity boundary.
func readSnapshotUnitRepositoryEvidence(root string, catalog *Catalog, locale string, snapshot QualityCheckSnapshotUnit) (*snapshotUnitRepositoryEvidence, error) {
	unit, err := catalog.Unit(snapshot.UnitID)
	if err != nil || unit.Kind != snapshot.UnitKind || unit.SourcePath != snapshot.SourcePath || unit.SourceSHA256 != snapshot.SourceSHA256 {
		return nil, errors.New("source identity does not match Candidate Snapshot")
	}
	candidate, err := readSnapshotReferencedFile(root, snapshot.CandidatePath)
	if err != nil || sum(candidate) != snapshot.CandidateSHA256 {
		return nil, errors.New("candidate identity does not match Candidate Snapshot")
	}
	validation, err := readSnapshotReferencedFile(root, snapshot.ValidationPath)
	if err != nil || sum(validation) != snapshot.ValidationSHA256 {
		return nil, errors.New("validation identity does not match Candidate Snapshot")
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", locale, snapshot.SelectedBatchID)
	manifest, err := readRetranslationProcessManifest(batchDir, locale, snapshot.SelectedBatchID)
	if err != nil {
		return nil, err
	}
	var record RetranslationBatchUnit
	for _, item := range manifest.Units {
		if item.UnitID == snapshot.UnitID {
			record = item
			break
		}
	}
	if record.UnitID == "" || record.UnitKind != snapshot.UnitKind || record.SourcePath != snapshot.SourcePath || record.SourceSHA256 != snapshot.SourceSHA256 {
		return nil, errors.New("batch manifest identity does not match Candidate Snapshot")
	}
	result, err := readPromotionResult(batchDir, locale, snapshot.SelectedBatchID, len(manifest.Units))
	if err != nil {
		return nil, err
	}
	var unitResult RetranslationUnitResult
	for _, item := range result.Units {
		if item.UnitID == snapshot.UnitID {
			unitResult = item
			break
		}
	}
	if unitResult.UnitID == "" || unitResult.Status != "passed" ||
		filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, snapshot.SelectedBatchID, unitResult.CandidatePath)) != snapshot.CandidatePath ||
		filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, snapshot.SelectedBatchID, unitResult.ValidationPath)) != snapshot.ValidationPath {
		return nil, errors.New("batch result identity does not match Candidate Snapshot")
	}
	parsedValidation, err := readPromotionValidation(batchDir, snapshot.SelectedBatchID, locale, record, unitResult)
	if err != nil || parsedValidation.Attempt != snapshot.Attempt {
		return nil, errors.New("final attempt identity does not match Candidate Snapshot")
	}
	return &snapshotUnitRepositoryEvidence{
		unit: unit, candidate: candidate, batchDir: batchDir, record: record, result: unitResult, validation: parsedValidation,
	}, nil
}

// reviewMatchesSnapshotIdentity intentionally does not require source_path:
// schema-v1 legacy evidence did not require it, while source hash and the
// selected batch identity remain immutable and sufficient for comparison.
func reviewMatchesSnapshotIdentity(locale string, snapshot QualityCheckSnapshotUnit, review TranslationReview) bool {
	return review.SchemaVersion == TranslationReviewSchemaVersion && review.BatchID == snapshot.SelectedBatchID &&
		review.Locale == locale && review.UnitID == snapshot.UnitID && review.UnitKind == snapshot.UnitKind &&
		review.SourceSHA256 == snapshot.SourceSHA256 && review.Attempt == snapshot.Attempt &&
		filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, review.BatchID, review.CandidatePath)) == snapshot.CandidatePath &&
		review.CandidateSHA256 == snapshot.CandidateSHA256 &&
		filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, review.BatchID, review.ValidationPath)) == snapshot.ValidationPath &&
		review.ValidationSHA256 == snapshot.ValidationSHA256 && review.Reviewer != "" && review.ReviewedAt != "" &&
		review.Summary != "" && review.Issues != nil
}

func readQualityCheckSnapshotForReview(root, locale, snapshotID string) (*QualityCheckSnapshotManifest, error) {
	return readQualityCheckSnapshot(root, locale, snapshotID, true)
}

func readQualityCheckSnapshot(root, locale, snapshotID string, verifyCurrentGlossary bool) (*QualityCheckSnapshotManifest, error) {
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
	if verifyCurrentGlossary {
		glossaryData, err := readSnapshotReferencedFile(root, manifest.GlossaryPath)
		if err != nil {
			return nil, fmt.Errorf("quality-check snapshot %q glossary: %w", snapshotID, err)
		}
		if sum(glossaryData) != manifest.GlossarySHA256 {
			return nil, fmt.Errorf("quality-check snapshot %q glossary hash mismatch", snapshotID)
		}
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
		if review.Attempt < 1 || review.Reviewer == "" || review.ReviewedAt == "" || review.Rubric != TranslationQualityRubric || review.Summary == "" || review.Issues == nil {
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
