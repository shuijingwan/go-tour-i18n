package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const QualityCheckResultsSchemaVersion = 1
const QualityCheckResultsEvidenceType = "quality_check_results"

type QualityCheckResult struct {
	Index  int    `json:"index"`
	UnitID string `json:"unit_id"`
	Rating string `json:"rating"`
}

// QualityCheckResults is deliberately smaller than TranslationReview. It is
// an immutable-Snapshot-bound record used only to derive later Quality Check
// work. It has no decision field and is never read by promotion.
type QualityCheckResults struct {
	SchemaVersion          int                  `json:"schema_version"`
	EvidenceType           string               `json:"evidence_type"`
	Locale                 string               `json:"locale"`
	SnapshotID             string               `json:"snapshot_id"`
	SnapshotManifestSHA256 string               `json:"snapshot_manifest_sha256"`
	PreviousSnapshotID     string               `json:"previous_snapshot_id,omitempty"`
	Rubric                 string               `json:"rubric"`
	ResultCount            int                  `json:"result_count"`
	Results                []QualityCheckResult `json:"results"`
}

type QualityCheckRecordOptions struct {
	Locale             string
	SnapshotID         string
	PreviousSnapshotID string
	UnitIDs            []string
	Rating             string
}

type QualityCheckRecordBatchOptions struct {
	Locale             string
	SnapshotID         string
	PreviousSnapshotID string
	StartIndex         int
	Limit              int
	Rating             string
}

type QualityCheckRecordResult struct {
	Locale             string   `json:"locale"`
	SnapshotID         string   `json:"snapshot_id"`
	PreviousSnapshotID string   `json:"previous_snapshot_id,omitempty"`
	Rating             string   `json:"rating"`
	RecordedCount      int      `json:"recorded_count"`
	ResultCount        int      `json:"result_count"`
	UnitIDs            []string `json:"unit_ids"`
	Path               string   `json:"path"`
}

type QualityCheckScopeReason string

const (
	QualityCheckScopeReasonMissing         QualityCheckScopeReason = "missing_quality_check"
	QualityCheckScopeReasonIdentityChanged QualityCheckScopeReason = "identity_changed"
	QualityCheckScopeReasonGlossaryChanged QualityCheckScopeReason = "glossary_changed"
	QualityCheckScopeReasonRubricChanged   QualityCheckScopeReason = "rubric_changed"
	QualityCheckScopeReasonNonA            QualityCheckScopeReason = "non_a_quality_check"
)

type QualityCheckRequiredAction string

const (
	QualityCheckActionRequired         QualityCheckRequiredAction = "quality_check_required"
	QualityCheckActionRevisionRequired QualityCheckRequiredAction = "revision_required"
)

type QualityCheckScopeOptions struct {
	Locale             string
	SnapshotID         string
	PreviousSnapshotID string
}

type QualityCheckScopeUnit struct {
	Index          int                        `json:"index"`
	UnitID         string                     `json:"unit_id"`
	UnitKind       UnitKind                   `json:"unit_kind"`
	BatchID        string                     `json:"batch_id"`
	Rating         string                     `json:"rating,omitempty"`
	FromSnapshotID string                     `json:"from_snapshot_id,omitempty"`
	Reason         QualityCheckScopeReason    `json:"reason,omitempty"`
	RequiredAction QualityCheckRequiredAction `json:"required_action,omitempty"`
}

type QualityCheckScope struct {
	Locale              string                  `json:"locale"`
	SnapshotID          string                  `json:"snapshot_id"`
	PreviousSnapshotID  string                  `json:"previous_snapshot_id,omitempty"`
	UnitCount           int                     `json:"unit_count"`
	CurrentResultCount  int                     `json:"current_result_count"`
	CarryForwardCount   int                     `json:"carry_forward_count"`
	EffectiveCount      int                     `json:"effective_result_count"`
	ACount              int                     `json:"a_count"`
	BCount              int                     `json:"b_count"`
	CCount              int                     `json:"c_count"`
	DCount              int                     `json:"d_count"`
	PendingCount        int                     `json:"pending_count"`
	ReadyForFinalReview bool                    `json:"ready_for_final_review"`
	CarryForward        []QualityCheckScopeUnit `json:"carry_forward_units"`
	Pending             []QualityCheckScopeUnit `json:"pending_quality_check_units"`
}

type effectiveQualityCheckResult struct {
	rating     string
	rubric     string
	snapshotID string
	unit       QualityCheckSnapshotUnit
}

func RecordQualityCheckResults(root string, catalog *Catalog, options QualityCheckRecordOptions) (*QualityCheckRecordResult, error) {
	if catalog == nil {
		return nil, errors.New("quality-check catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(options.SnapshotID); err != nil {
		return nil, err
	}
	if options.PreviousSnapshotID != "" {
		if err := validateSnapshotID(options.PreviousSnapshotID); err != nil {
			return nil, err
		}
		if options.PreviousSnapshotID == options.SnapshotID {
			return nil, errors.New("previous_snapshot_id must differ from snapshot_id")
		}
		if _, err := readQualityCheckSnapshot(root, options.Locale, options.PreviousSnapshotID, false); err != nil {
			return nil, fmt.Errorf("previous quality-check snapshot: %w", err)
		}
	}
	if !validQualityRating(options.Rating) {
		return nil, fmt.Errorf("invalid quality-check rating %q", options.Rating)
	}
	if len(options.UnitIDs) == 0 {
		return nil, errors.New("at least one quality-check unit_id is required")
	}
	snapshot, err := readQualityCheckSnapshotForReview(root, options.Locale, options.SnapshotID)
	if err != nil {
		return nil, err
	}
	existing, err := readQualityCheckResults(root, options.Locale, snapshot)
	if err != nil {
		return nil, err
	}
	previousSnapshotID := options.PreviousSnapshotID
	if existing != nil {
		if existing.Rubric != TranslationQualityRubric {
			return nil, fmt.Errorf("quality-check results use obsolete rubric %q; create a new full Snapshot", existing.Rubric)
		}
		if existing.PreviousSnapshotID != "" && previousSnapshotID != "" && previousSnapshotID != existing.PreviousSnapshotID {
			return nil, fmt.Errorf("previous_snapshot_id %q does not match existing result lineage %q", previousSnapshotID, existing.PreviousSnapshotID)
		}
		if existing.PreviousSnapshotID != "" {
			previousSnapshotID = existing.PreviousSnapshotID
		}
	}

	byID := make(map[string]QualityCheckSnapshotUnit, len(snapshot.Units))
	for _, unit := range snapshot.Units {
		byID[unit.UnitID] = unit
	}
	seenRequested := map[string]bool{}
	selected := make([]QualityCheckSnapshotUnit, 0, len(options.UnitIDs))
	for _, unitID := range options.UnitIDs {
		if unitID == "" || seenRequested[unitID] {
			return nil, fmt.Errorf("empty or duplicate quality-check unit_id %q", unitID)
		}
		seenRequested[unitID] = true
		unit, ok := byID[unitID]
		if !ok {
			return nil, fmt.Errorf("snapshot %q has no unit %q", options.SnapshotID, unitID)
		}
		if _, err := readSnapshotUnitRepositoryEvidence(root, catalog, options.Locale, unit); err != nil {
			return nil, fmt.Errorf("snapshot index %d (%s) identity: %w", unit.Index, unit.UnitID, err)
		}
		selected = append(selected, unit)
	}

	results := &QualityCheckResults{
		SchemaVersion: QualityCheckResultsSchemaVersion, EvidenceType: QualityCheckResultsEvidenceType,
		Locale: options.Locale, SnapshotID: options.SnapshotID, PreviousSnapshotID: previousSnapshotID,
		Rubric: TranslationQualityRubric, Results: []QualityCheckResult{},
	}
	if existing != nil {
		*results = *existing
		results.Results = append([]QualityCheckResult(nil), existing.Results...)
		results.PreviousSnapshotID = previousSnapshotID
	}
	existingByID := make(map[string]bool, len(results.Results))
	for _, result := range results.Results {
		existingByID[result.UnitID] = true
	}
	for _, unit := range selected {
		if existingByID[unit.UnitID] {
			return nil, fmt.Errorf("quality-check result already exists for %s", unit.UnitID)
		}
		results.Results = append(results.Results, QualityCheckResult{Index: unit.Index, UnitID: unit.UnitID, Rating: options.Rating})
	}
	sort.Slice(results.Results, func(i, j int) bool { return results.Results[i].Index < results.Results[j].Index })
	results.ResultCount = len(results.Results)
	manifestData, err := os.ReadFile(qualityCheckSnapshotManifestPath(root, options.Locale, options.SnapshotID))
	if err != nil {
		return nil, err
	}
	results.SnapshotManifestSHA256 = sum(manifestData)
	path := qualityCheckResultsPath(root, options.Locale, options.SnapshotID)
	if err := writeQualityCheckResults(path, results); err != nil {
		return nil, err
	}
	repositoryPath, err := repositoryRelativePath(root, path)
	if err != nil {
		return nil, err
	}
	unitIDs := make([]string, 0, len(selected))
	for _, unit := range selected {
		unitIDs = append(unitIDs, unit.UnitID)
	}
	return &QualityCheckRecordResult{
		Locale: options.Locale, SnapshotID: options.SnapshotID, PreviousSnapshotID: previousSnapshotID,
		Rating: options.Rating, RecordedCount: len(selected), ResultCount: results.ResultCount,
		UnitIDs: unitIDs, Path: repositoryPath,
	}, nil
}

func RecordQualityCheckResultBatch(root string, catalog *Catalog, options QualityCheckRecordBatchOptions) (*QualityCheckRecordResult, error) {
	start := options.StartIndex
	if start == 0 {
		start = 1
	}
	limit := options.Limit
	if limit == 0 {
		limit = DefaultRetranslationReviewBatchLimit
	}
	if start < 1 || limit < 1 {
		return nil, errors.New("quality-check start_index and limit must be at least 1")
	}
	if limit > DefaultRetranslationReviewBatchLimit {
		return nil, fmt.Errorf("quality-check limit must not exceed %d", DefaultRetranslationReviewBatchLimit)
	}
	snapshot, err := readQualityCheckSnapshotForReview(root, options.Locale, options.SnapshotID)
	if err != nil {
		return nil, err
	}
	if start > len(snapshot.Units) {
		return nil, fmt.Errorf("start_index %d is outside Candidate Snapshot range 1-%d", start, len(snapshot.Units))
	}
	end := start + limit - 1
	if end > len(snapshot.Units) || end < start {
		end = len(snapshot.Units)
	}
	unitIDs := make([]string, 0, end-start+1)
	selected := snapshot.Units[start-1 : end]
	for _, unit := range selected[1:] {
		if unit.UnitKind != selected[0].UnitKind {
			return nil, fmt.Errorf("quality-check record-batch range must not mix %s and %s TranslationUnits", selected[0].UnitKind, unit.UnitKind)
		}
	}
	for _, unit := range selected {
		unitIDs = append(unitIDs, unit.UnitID)
	}
	return RecordQualityCheckResults(root, catalog, QualityCheckRecordOptions{
		Locale: options.Locale, SnapshotID: options.SnapshotID, PreviousSnapshotID: options.PreviousSnapshotID,
		UnitIDs: unitIDs, Rating: options.Rating,
	})
}

func BuildQualityCheckScope(root string, catalog *Catalog, options QualityCheckScopeOptions) (*QualityCheckScope, error) {
	if catalog == nil {
		return nil, errors.New("quality-check catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(options.SnapshotID); err != nil {
		return nil, err
	}
	current, err := readQualityCheckSnapshotForReview(root, options.Locale, options.SnapshotID)
	if err != nil {
		return nil, err
	}
	currentResults, err := readQualityCheckResults(root, options.Locale, current)
	if err != nil {
		return nil, err
	}
	previousSnapshotID := options.PreviousSnapshotID
	if previousSnapshotID != "" {
		if err := validateSnapshotID(previousSnapshotID); err != nil {
			return nil, err
		}
		if previousSnapshotID == options.SnapshotID {
			return nil, errors.New("previous_snapshot_id must differ from snapshot_id")
		}
	}
	if currentResults != nil {
		if currentResults.PreviousSnapshotID != "" && previousSnapshotID != "" && previousSnapshotID != currentResults.PreviousSnapshotID {
			return nil, fmt.Errorf("previous_snapshot_id %q does not match recorded lineage %q", previousSnapshotID, currentResults.PreviousSnapshotID)
		}
		if currentResults.PreviousSnapshotID != "" {
			previousSnapshotID = currentResults.PreviousSnapshotID
		}
	}

	currentByID := map[string]QualityCheckResult{}
	if currentResults != nil {
		for _, result := range currentResults.Results {
			currentByID[result.UnitID] = result
		}
	}
	var previous *QualityCheckSnapshotManifest
	previousEffective := map[string]effectiveQualityCheckResult{}
	if previousSnapshotID != "" {
		if previousSnapshotID == options.SnapshotID {
			return nil, errors.New("quality-check result lineage contains a cycle")
		}
		previous, previousEffective, err = loadEffectiveQualityCheckResults(root, options.Locale, previousSnapshotID, map[string]bool{})
		if err != nil {
			return nil, err
		}
	}

	scope := &QualityCheckScope{
		Locale: options.Locale, SnapshotID: options.SnapshotID, PreviousSnapshotID: previousSnapshotID,
		UnitCount: current.UnitCount, CurrentResultCount: len(currentByID),
		CarryForward: []QualityCheckScopeUnit{}, Pending: []QualityCheckScopeUnit{},
	}
	for _, unit := range current.Units {
		base := QualityCheckScopeUnit{Index: unit.Index, UnitID: unit.UnitID, UnitKind: unit.UnitKind, BatchID: unit.SelectedBatchID}
		if _, err := readSnapshotUnitRepositoryEvidence(root, catalog, options.Locale, unit); err != nil {
			base.Reason = QualityCheckScopeReasonIdentityChanged
			base.RequiredAction = QualityCheckActionRequired
			scope.Pending = append(scope.Pending, base)
			continue
		}
		if direct, ok := currentByID[unit.UnitID]; ok {
			if currentResults.Rubric != TranslationQualityRubric {
				base.Reason = QualityCheckScopeReasonRubricChanged
				base.RequiredAction = QualityCheckActionRequired
				scope.Pending = append(scope.Pending, base)
				continue
			}
			base.Rating = direct.Rating
			addQualityCheckRating(scope, direct.Rating)
			if direct.Rating != "A" {
				base.Reason = QualityCheckScopeReasonNonA
				base.RequiredAction = QualityCheckActionRevisionRequired
				scope.Pending = append(scope.Pending, base)
			}
			continue
		}
		prior, ok := previousEffective[unit.UnitID]
		if !ok {
			base.Reason = QualityCheckScopeReasonMissing
			base.RequiredAction = QualityCheckActionRequired
			scope.Pending = append(scope.Pending, base)
			continue
		}
		if previous != nil && previous.GlossarySHA256 != current.GlossarySHA256 {
			base.Reason = QualityCheckScopeReasonGlossaryChanged
			base.RequiredAction = QualityCheckActionRequired
			scope.Pending = append(scope.Pending, base)
			continue
		}
		if !qualityCheckSnapshotIdentityMatches(unit, prior.unit) {
			base.Reason = QualityCheckScopeReasonIdentityChanged
			base.RequiredAction = QualityCheckActionRequired
			scope.Pending = append(scope.Pending, base)
			continue
		}
		if prior.rubric != TranslationQualityRubric {
			base.Reason = QualityCheckScopeReasonRubricChanged
			base.RequiredAction = QualityCheckActionRequired
			scope.Pending = append(scope.Pending, base)
			continue
		}
		base.Rating = prior.rating
		base.FromSnapshotID = prior.snapshotID
		addQualityCheckRating(scope, prior.rating)
		if prior.rating == "A" {
			scope.CarryForward = append(scope.CarryForward, base)
			continue
		}
		base.Reason = QualityCheckScopeReasonNonA
		base.RequiredAction = QualityCheckActionRevisionRequired
		scope.Pending = append(scope.Pending, base)
	}
	scope.CarryForwardCount = len(scope.CarryForward)
	scope.PendingCount = len(scope.Pending)
	scope.EffectiveCount = scope.ACount + scope.BCount + scope.CCount + scope.DCount
	scope.ReadyForFinalReview = scope.ACount == scope.UnitCount && scope.PendingCount == 0
	return scope, nil
}

func loadEffectiveQualityCheckResults(root, locale, snapshotID string, seen map[string]bool) (*QualityCheckSnapshotManifest, map[string]effectiveQualityCheckResult, error) {
	if seen[snapshotID] {
		return nil, nil, errors.New("quality-check result lineage contains a cycle")
	}
	seen[snapshotID] = true
	defer delete(seen, snapshotID)
	snapshot, err := readQualityCheckSnapshot(root, locale, snapshotID, false)
	if err != nil {
		return nil, nil, err
	}
	results, err := readQualityCheckResults(root, locale, snapshot)
	if err != nil {
		return nil, nil, err
	}
	effective := map[string]effectiveQualityCheckResult{}
	if results == nil {
		return snapshot, effective, nil
	}
	if results.PreviousSnapshotID != "" {
		previous, inherited, err := loadEffectiveQualityCheckResults(root, locale, results.PreviousSnapshotID, seen)
		if err != nil {
			return nil, nil, err
		}
		if previous.GlossarySHA256 == snapshot.GlossarySHA256 {
			for _, unit := range snapshot.Units {
				prior, ok := inherited[unit.UnitID]
				if ok && qualityCheckSnapshotIdentityMatches(unit, prior.unit) {
					effective[unit.UnitID] = prior
				}
			}
		}
	}
	for _, result := range results.Results {
		unit := snapshot.Units[result.Index-1]
		effective[result.UnitID] = effectiveQualityCheckResult{rating: result.Rating, rubric: results.Rubric, snapshotID: snapshotID, unit: unit}
	}
	return snapshot, effective, nil
}

func readQualityCheckResults(root, locale string, snapshot *QualityCheckSnapshotManifest) (*QualityCheckResults, error) {
	path := qualityCheckResultsPath(root, locale, snapshot.SnapshotID)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read quality-check results %q: %w", snapshot.SnapshotID, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var results QualityCheckResults
	if err := decoder.Decode(&results); err != nil {
		return nil, fmt.Errorf("parse quality-check results %q: %w", snapshot.SnapshotID, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("parse quality-check results %q: multiple JSON values", snapshot.SnapshotID)
	}
	manifestData, err := os.ReadFile(qualityCheckSnapshotManifestPath(root, locale, snapshot.SnapshotID))
	if err != nil {
		return nil, err
	}
	if results.SchemaVersion != QualityCheckResultsSchemaVersion || results.EvidenceType != QualityCheckResultsEvidenceType ||
		results.Locale != locale || results.SnapshotID != snapshot.SnapshotID || results.SnapshotManifestSHA256 != sum(manifestData) ||
		results.Rubric == "" || results.ResultCount != len(results.Results) {
		return nil, fmt.Errorf("quality-check results %q have incompatible identity", snapshot.SnapshotID)
	}
	if results.PreviousSnapshotID != "" {
		if err := validateSnapshotID(results.PreviousSnapshotID); err != nil || results.PreviousSnapshotID == snapshot.SnapshotID {
			return nil, fmt.Errorf("quality-check results %q have invalid previous_snapshot_id", snapshot.SnapshotID)
		}
	}
	seen := map[string]bool{}
	lastIndex := 0
	for _, result := range results.Results {
		if result.Index <= lastIndex || result.Index < 1 || result.Index > len(snapshot.Units) || seen[result.UnitID] {
			return nil, fmt.Errorf("quality-check results %q have unstable or duplicate result identity", snapshot.SnapshotID)
		}
		unit := snapshot.Units[result.Index-1]
		if result.UnitID != unit.UnitID || !validQualityRating(result.Rating) {
			return nil, fmt.Errorf("quality-check results %q do not match Candidate Snapshot at index %d", snapshot.SnapshotID, result.Index)
		}
		seen[result.UnitID] = true
		lastIndex = result.Index
	}
	return &results, nil
}

func writeQualityCheckResults(path string, results *QualityCheckResults) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".quality-check-results-*")
	if err != nil {
		return fmt.Errorf("create quality-check results: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("chmod quality-check results: %w", err)
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write quality-check results: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close quality-check results: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("commit quality-check results: %w", err)
	}
	return nil
}

func qualityCheckSnapshotManifestPath(root, locale, snapshotID string) string {
	return filepath.Join(root, "data", "quality-check-snapshots", locale, snapshotID, "manifest.json")
}

func qualityCheckResultsPath(root, locale, snapshotID string) string {
	return filepath.Join(root, "data", "quality-check-snapshots", locale, snapshotID, "quality-check-results.json")
}

func qualityCheckSnapshotIdentityMatches(current, prior QualityCheckSnapshotUnit) bool {
	return current.UnitID == prior.UnitID && current.UnitKind == prior.UnitKind && current.SelectedBatchID == prior.SelectedBatchID &&
		current.SourcePath == prior.SourcePath && current.SourceSHA256 == prior.SourceSHA256 &&
		current.CandidatePath == prior.CandidatePath && current.CandidateSHA256 == prior.CandidateSHA256 &&
		current.ValidationPath == prior.ValidationPath && current.ValidationSHA256 == prior.ValidationSHA256 &&
		current.Attempt == prior.Attempt
}

func validQualityRating(rating string) bool {
	return rating == "A" || rating == "B" || rating == "C" || rating == "D"
}

func addQualityCheckRating(scope *QualityCheckScope, rating string) {
	switch rating {
	case "A":
		scope.ACount++
	case "B":
		scope.BCount++
	case "C":
		scope.CCount++
	case "D":
		scope.DCount++
	}
}
