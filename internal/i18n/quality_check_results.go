package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const QualityCheckResultsSchemaVersion = 2
const legacyQualityCheckResultsSchemaVersion = 1
const QualityCheckResultsEvidenceType = "quality_check_results"
const DefaultQualityCheckBatchLimit = 60

type QualityCheckResult struct {
	Index   int    `json:"index"`
	UnitID  string `json:"unit_id"`
	Rating  string `json:"rating"`
	Finding string `json:"finding,omitempty"`
}

// QualityCheckResults is deliberately smaller than TranslationReview. It is
// an immutable-Snapshot-bound record used only to derive later Quality Check
// work. It has no decision field and is never read by promotion.
type QualityCheckResults struct {
	ReviewerSession        string               `json:"reviewer_session,omitempty"`
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
	ReviewerSession    string
	FullRereview       bool
	Locale             string
	SnapshotID         string
	PreviousSnapshotID string
	UnitIDs            []string
	Rating             string
	Finding            string
}

// QualityCheckJudgment is command input, not a receipt or a results schema.
// Stable indexes and all authority bindings are resolved from the Snapshot.
type QualityCheckJudgment struct {
	UnitID  string `json:"unit_id"`
	Rating  string `json:"rating"`
	Finding string `json:"finding,omitempty"`
}

type QualityCheckRecordJudgmentsOptions struct {
	ReviewerSession    string
	FullRereview       bool
	Locale             string
	SnapshotID         string
	PreviousSnapshotID string
	Judgments          []QualityCheckJudgment
}

// Invocation-local instrumentation for deterministic performance regressions.
type qualityCheckRecordingStats struct {
	SnapshotReads           int
	ContextBuilds           int
	RepositoryEvidenceReads int
	WriteAttempts           int
}

type QualityCheckRecordBatchOptions struct {
	ReviewerSession    string
	FullRereview       bool
	Locale             string
	SnapshotID         string
	PreviousSnapshotID string
	StartIndex         int
	Limit              int
	Rating             string
	Finding            string
}

type QualityCheckFindingBackfillOptions struct{ Locale, SnapshotID, UnitID, Finding string }

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
	CompatibilityChain []GlossaryArchiveReference `json:"compatibility_chain,omitempty"`
	Index              int                        `json:"index"`
	UnitID             string                     `json:"unit_id"`
	UnitKind           UnitKind                   `json:"unit_kind"`
	BatchID            string                     `json:"batch_id"`
	Rating             string                     `json:"rating,omitempty"`
	FromSnapshotID     string                     `json:"from_snapshot_id,omitempty"`
	Reason             QualityCheckScopeReason    `json:"reason,omitempty"`
	RequiredAction     QualityCheckRequiredAction `json:"required_action,omitempty"`
}

type QualityCheckScope struct {
	Locale               string `json:"locale"`
	SnapshotID           string `json:"snapshot_id"`
	PreviousSnapshotID   string `json:"previous_snapshot_id,omitempty"`
	UnitCount            int    `json:"unit_count"`
	CurrentResultCount   int    `json:"current_result_count"`
	CarryForwardCount    int    `json:"carry_forward_count"`
	EffectiveCount       int    `json:"effective_result_count"`
	ACount               int    `json:"a_count"`
	BCount               int    `json:"b_count"`
	CCount               int    `json:"c_count"`
	DCount               int    `json:"d_count"`
	PendingCount         int    `json:"pending_count"`
	ReadyForFinalization bool   `json:"ready_for_finalization"`
	// ReadyForFinalReview is a deprecated JSON compatibility alias. New
	// consumers must use ready_for_finalization.
	ReadyForFinalReview bool                    `json:"ready_for_final_review,omitempty"`
	CarryForward        []QualityCheckScopeUnit `json:"carry_forward_units"`
	Pending             []QualityCheckScopeUnit `json:"pending_quality_check_units"`
}

type effectiveQualityCheckResult struct {
	currentReattested  bool
	compatibilityChain []GlossaryArchiveReference
	rating             string
	finding            string
	rubric             string
	snapshotID         string
	unit               QualityCheckSnapshotUnit
}

type effectiveQualityCheckMaterialization struct {
	snapshot *QualityCheckSnapshotManifest
	results  map[string]effectiveQualityCheckResult
}

// qualityCheckEvaluation is invocation-local. Snapshot IDs are unique within
// its fixed root/locale/catalog authority, so a completed materialization can
// be reused safely while recursive in-progress IDs remain guarded by seen.
type qualityCheckEvaluation struct {
	originalOnly       bool // historical A integrity only; consumers separately check current compatibility
	root               string
	locale             string
	catalog            *Catalog
	compatibility      *PreparedGlossaryCompatibility
	effective          map[string]effectiveQualityCheckMaterialization
	repositoryEvidence map[string]*snapshotUnitRepositoryEvidence
	machineEvidence    map[string]*machineSelection
	stats              *qualityCheckEvaluationStats
	membership         map[string]*qualityCheckMembershipBaseline
}

type qualityCheckEvaluationStats struct {
	EffectiveMaterializations   map[string]int
	EffectiveCacheHits          int
	RepositoryEvidenceLoads     int
	RepositoryEvidenceCacheHits int
}

func newQualityCheckEvaluation(root, locale string, catalog *Catalog) *qualityCheckEvaluation {
	return newQualityCheckEvaluationWithStats(root, locale, catalog, nil, nil)
}

func newQualityCheckEvaluationWithStats(root, locale string, catalog *Catalog, stats *qualityCheckEvaluationStats, compatibilityStats *glossaryCompatibilityPrepareStats) *qualityCheckEvaluation {
	if stats != nil && stats.EffectiveMaterializations == nil {
		stats.EffectiveMaterializations = map[string]int{}
	}
	return &qualityCheckEvaluation{
		root: root, locale: locale, catalog: catalog,
		compatibility:      newPreparedGlossaryCompatibility(root, locale, catalog, compatibilityStats),
		effective:          map[string]effectiveQualityCheckMaterialization{},
		repositoryEvidence: map[string]*snapshotUnitRepositoryEvidence{},
		machineEvidence:    map[string]*machineSelection{},
		stats:              stats,
	}
}

func (evaluation *qualityCheckEvaluation) readSnapshotUnitRepositoryEvidence(snapshotID string, unit QualityCheckSnapshotUnit) (*snapshotUnitRepositoryEvidence, error) {
	key := snapshotID + "\x00" + unit.UnitID
	if cached, ok := evaluation.repositoryEvidence[key]; ok {
		if evaluation.stats != nil {
			evaluation.stats.RepositoryEvidenceCacheHits++
		}
		return cached, nil
	}
	var evidence *snapshotUnitRepositoryEvidence
	var err error
	if unit.RepresentationReconciliation != nil {
		evidence, _, err = machineSnapshotEvidenceInWorld(evaluation.root, evaluation.catalog, evaluation.locale, unit, evaluation.machineEvidence, evaluation.originalOnly || evaluation.compatibility.historicalSHA != "")
	} else {
		evidence, err = readSnapshotUnitRepositoryEvidence(evaluation.root, evaluation.catalog, evaluation.locale, unit)
	}
	if err != nil {
		return nil, err
	}
	if evaluation.stats != nil {
		evaluation.stats.RepositoryEvidenceLoads++
	}
	evaluation.repositoryEvidence[key] = evidence
	return evidence, nil
}

func cloneEffectiveQualityCheckResults(in map[string]effectiveQualityCheckResult) map[string]effectiveQualityCheckResult {
	out := make(map[string]effectiveQualityCheckResult, len(in))
	for id, result := range in {
		result.compatibilityChain = append([]GlossaryArchiveReference(nil), result.compatibilityChain...)
		out[id] = result
	}
	return out
}

func RecordQualityCheckResults(root string, catalog *Catalog, options QualityCheckRecordOptions) (*QualityCheckRecordResult, error) {
	judgments := make([]QualityCheckJudgment, 0, len(options.UnitIDs))
	for _, id := range options.UnitIDs {
		judgments = append(judgments, QualityCheckJudgment{UnitID: id, Rating: options.Rating, Finding: options.Finding})
	}
	return recordQualityCheckJudgments(root, catalog, options, judgments, false, nil)
}

// RecordQualityCheckJudgments validates one independent Reviewer working set
// and publishes it once. No judgment is committed before the entire set passes.
func RecordQualityCheckJudgments(root string, catalog *Catalog, options QualityCheckRecordJudgmentsOptions) (*QualityCheckRecordResult, error) {
	return recordQualityCheckJudgmentsBulk(root, catalog, options, nil)
}

func recordQualityCheckJudgmentsBulk(root string, catalog *Catalog, options QualityCheckRecordJudgmentsOptions, stats *qualityCheckRecordingStats) (*QualityCheckRecordResult, error) {
	if strings.TrimSpace(options.ReviewerSession) == "" {
		return nil, errors.New("独立质量审核记录必须给出 --reviewer-session")
	}
	ids := make([]string, 0, len(options.Judgments))
	for _, j := range options.Judgments {
		ids = append(ids, j.UnitID)
	}
	return recordQualityCheckJudgments(root, catalog, QualityCheckRecordOptions{
		ReviewerSession: options.ReviewerSession, FullRereview: options.FullRereview,
		Locale: options.Locale, SnapshotID: options.SnapshotID, PreviousSnapshotID: options.PreviousSnapshotID, UnitIDs: ids,
	}, options.Judgments, true, stats)
}

func recordQualityCheckJudgments(root string, catalog *Catalog, options QualityCheckRecordOptions, judgments []QualityCheckJudgment, heterogeneous bool, stats *qualityCheckRecordingStats) (*QualityCheckRecordResult, error) {
	// Enforce the bulk contract itself, before role/currentness/evidence work.
	if heterogeneous && len(judgments) > DefaultQualityCheckBatchLimit {
		return nil, fmt.Errorf("quality-check heterogeneous judgment limit must not exceed %d", DefaultQualityCheckBatchLimit)
	}
	if catalog == nil {
		return nil, errors.New("quality-check catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(options.SnapshotID); err != nil {
		return nil, err
	}
	if options.FullRereview && options.PreviousSnapshotID != "" {
		return nil, errors.New("previous_snapshot_id and full_rereview are mutually exclusive")
	}
	if options.PreviousSnapshotID != "" {
		if err := validateSnapshotID(options.PreviousSnapshotID); err != nil {
			return nil, err
		}
		if options.PreviousSnapshotID == options.SnapshotID {
			return nil, errors.New("previous_snapshot_id must differ from snapshot_id")
		}
	}
	if len(judgments) == 0 {
		return nil, errors.New("at least one quality-check unit_id is required")
	}
	byJudgment := make(map[string]QualityCheckJudgment, len(judgments))
	for _, j := range judgments {
		if j.UnitID == "" {
			return nil, fmt.Errorf("empty or duplicate quality-check unit_id %q", j.UnitID)
		}
		if _, duplicate := byJudgment[j.UnitID]; duplicate {
			return nil, fmt.Errorf("empty or duplicate quality-check unit_id %q", j.UnitID)
		}
		if !validQualityRating(j.Rating) {
			return nil, fmt.Errorf("invalid quality-check rating %q", j.Rating)
		}
		j.Finding = strings.TrimSpace(j.Finding)
		if j.Rating != "A" && j.Finding == "" {
			return nil, errors.New("quality-check finding is required for rating B, C, or D")
		}
		byJudgment[j.UnitID] = j
	}
	// Legacy validation order remains unchanged. Bulk resolves Snapshot kinds
	// first so its independent same-kind gate cannot be masked by catalog or
	// initial-working-set checks. All those existing checks still run below.
	if !heterogeneous {
		if err := catalog.RequireTranslationInvocation(options.UnitIDs); err != nil {
			return nil, err
		}
	}
	if len(catalog.TranslationScope) > 0 || heterogeneous {
		if options.ReviewerSession == "" {
			return nil, errors.New("独立质量审核记录必须给出 --reviewer-session")
		}
		if err := CheckUnifiedLanguageRole(root, options.Locale, options.ReviewerSession, "reviewer"); err != nil {
			return nil, err
		}
	}
	if stats != nil {
		stats.SnapshotReads++
	}
	snapshot, err := readQualityCheckSnapshotForReview(root, options.Locale, options.SnapshotID, catalog)
	if err != nil {
		return nil, err
	}
	existing, err := readOriginalQualityCheckResults(root, options.Locale, snapshot)
	if err != nil {
		return nil, err
	}
	if err := requireUncorrectedQualityCheckResults(root, snapshot, existing); err != nil {
		return nil, err
	}
	if existing != nil && options.FullRereview {
		return nil, errors.New("full_rereview may only authorize the first results write")
	}
	if existing == nil && options.PreviousSnapshotID == "" {
		finalized, err := qualityCheckFinalizedSnapshotIDs(root, options.Locale, catalog)
		if err != nil {
			return nil, err
		}
		hasOtherFinalized := false
		for _, id := range finalized {
			if id != options.SnapshotID {
				hasOtherFinalized = true
				break
			}
		}
		if hasOtherFinalized && !options.FullRereview {
			return nil, errors.New("a finalized Quality Check Snapshot already exists: run quality-check preflight and select --previous-snapshot-id, or explicitly pass --full-rereview to accept repeated review")
		}
	}
	previousSnapshotID, err := resolveQualityCheckResultsLineage(options.SnapshotID, existing, options.PreviousSnapshotID)
	if err != nil {
		return nil, err
	}
	if previousSnapshotID == "" && !heterogeneous {
		if err := catalog.RequireInitialTranslationWorkingSet(options.UnitIDs); err != nil {
			return nil, err
		}
	}
	if existing != nil {
		if existing.Rubric != TranslationQualityRubric {
			return nil, fmt.Errorf("quality-check results use obsolete rubric %q; create a new full Snapshot", existing.Rubric)
		}
		if existing.ReviewerSession != options.ReviewerSession {
			return nil, errors.New("independent Reviewer session changed within Snapshot")
		}
	}
	if options.PreviousSnapshotID != "" {
		if _, err := readQualityCheckSnapshot(root, options.Locale, options.PreviousSnapshotID, false); err != nil {
			return nil, fmt.Errorf("previous quality-check snapshot: %w", err)
		}
	}

	byID := make(map[string]QualityCheckSnapshotUnit, len(snapshot.Units))
	for _, unit := range snapshot.Units {
		byID[unit.UnitID] = unit
	}
	// Use the existing current-world reconciliation reader with one shared
	// receipt context. Distinct Units still receive every exact evidence check.
	machines := map[string]*machineSelection{}
	if stats != nil {
		stats.ContextBuilds++
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
		selected = append(selected, unit)
	}
	if heterogeneous {
		if len(selected) == 0 {
			return nil, errors.New("at least one quality-check Snapshot Unit is required")
		}
		for _, unit := range selected[1:] {
			if unit.UnitKind != selected[0].UnitKind {
				return nil, fmt.Errorf("quality-check heterogeneous judgments must have the same UnitKind; mixed %s and %s are not allowed", selected[0].UnitKind, unit.UnitKind)
			}
		}
		if err := catalog.RequireTranslationInvocation(options.UnitIDs); err != nil {
			return nil, err
		}
		if previousSnapshotID == "" {
			if err := catalog.RequireInitialTranslationWorkingSet(options.UnitIDs); err != nil {
				return nil, err
			}
		}
	}
	for _, unit := range selected {
		if stats != nil {
			stats.RepositoryEvidenceReads++
		}
		var evidenceErr error
		if unit.RepresentationReconciliation != nil {
			_, _, evidenceErr = machineSnapshotEvidence(root, catalog, options.Locale, unit, machines)
		} else {
			_, evidenceErr = readSnapshotUnitRepositoryEvidence(root, catalog, options.Locale, unit)
		}
		if evidenceErr != nil {
			return nil, fmt.Errorf("snapshot index %d (%s) identity: %w", unit.Index, unit.UnitID, evidenceErr)
		}
	}

	results := &QualityCheckResults{
		ReviewerSession: options.ReviewerSession,
		SchemaVersion:   QualityCheckResultsSchemaVersion, EvidenceType: QualityCheckResultsEvidenceType,
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
		j := byJudgment[unit.UnitID]
		results.Results = append(results.Results, QualityCheckResult{Index: unit.Index, UnitID: unit.UnitID, Rating: j.Rating, Finding: j.Finding})
	}
	sort.Slice(results.Results, func(i, j int) bool { return results.Results[i].Index < results.Results[j].Index })
	results.ResultCount = len(results.Results)
	manifestData, err := os.ReadFile(qualityCheckSnapshotManifestPath(root, options.Locale, options.SnapshotID))
	if err != nil {
		return nil, err
	}
	results.SnapshotManifestSHA256 = sum(manifestData)
	path := qualityCheckResultsPath(root, options.Locale, options.SnapshotID)
	repositoryPath, err := repositoryRelativePath(root, path)
	if err != nil {
		return nil, err
	}
	if stats != nil {
		stats.WriteAttempts++
	}
	if err := writeQualityCheckResults(path, results); err != nil {
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
		limit = DefaultQualityCheckBatchLimit
	}
	if start < 1 || limit < 1 {
		return nil, errors.New("quality-check start_index and limit must be at least 1")
	}
	if limit > DefaultQualityCheckBatchLimit {
		return nil, fmt.Errorf("quality-check limit must not exceed %d", DefaultQualityCheckBatchLimit)
	}
	snapshot, err := readQualityCheckSnapshotForReview(root, options.Locale, options.SnapshotID, catalog)
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
		ReviewerSession: options.ReviewerSession,
		Locale:          options.Locale, SnapshotID: options.SnapshotID, PreviousSnapshotID: options.PreviousSnapshotID, FullRereview: options.FullRereview,
		UnitIDs: unitIDs, Rating: options.Rating,
		Finding: options.Finding,
	})
}

func BackfillQualityCheckFinding(root string, catalog *Catalog, options QualityCheckFindingBackfillOptions) (*QualityCheckRecordResult, error) {
	finding := strings.TrimSpace(options.Finding)
	if finding == "" {
		return nil, errors.New("quality-check finding must be non-empty")
	}
	snapshot, err := readQualityCheckSnapshotForReview(root, options.Locale, options.SnapshotID, catalog)
	if err != nil {
		return nil, err
	}
	results, err := readOriginalQualityCheckResults(root, options.Locale, snapshot)
	if err != nil {
		return nil, err
	}
	if err := requireUncorrectedQualityCheckResults(root, snapshot, results); err != nil {
		return nil, err
	}
	if results == nil {
		return nil, errors.New("quality-check results do not exist")
	}
	for i := range results.Results {
		result := &results.Results[i]
		if result.UnitID != options.UnitID {
			continue
		}
		unit := snapshot.Units[result.Index-1]
		if _, err := readSnapshotUnitRepositoryEvidence(root, catalog, options.Locale, unit); err != nil {
			return nil, fmt.Errorf("snapshot index %d (%s) identity: %w", unit.Index, unit.UnitID, err)
		}
		if result.Rating == "A" {
			return nil, errors.New("finding backfill is only allowed for an existing B, C, or D result")
		}
		if strings.TrimSpace(result.Finding) != "" {
			return nil, fmt.Errorf("quality-check finding already exists for %s", result.UnitID)
		}
		result.Finding = finding
		path := qualityCheckResultsPath(root, options.Locale, options.SnapshotID)
		if err := writeQualityCheckResults(path, results); err != nil {
			return nil, err
		}
		repoPath, err := repositoryRelativePath(root, path)
		if err != nil {
			return nil, err
		}
		return &QualityCheckRecordResult{Locale: options.Locale, SnapshotID: options.SnapshotID, Rating: result.Rating, RecordedCount: 1, ResultCount: results.ResultCount, UnitIDs: []string{result.UnitID}, Path: repoPath}, nil
	}
	return nil, fmt.Errorf("quality-check result does not exist for %s", options.UnitID)
}

func BuildQualityCheckScope(root string, catalog *Catalog, options QualityCheckScopeOptions) (*QualityCheckScope, error) {
	evaluation := newQualityCheckEvaluation(root, options.Locale, catalog)
	return buildQualityCheckScopeWithEvaluation(evaluation, options)
}

func buildQualityCheckScopeWithEvaluation(evaluation *qualityCheckEvaluation, options QualityCheckScopeOptions) (*QualityCheckScope, error) {
	root, catalog := evaluation.root, evaluation.catalog
	if catalog == nil {
		return nil, errors.New("quality-check catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(options.SnapshotID); err != nil {
		return nil, err
	}
	current, err := readQualityCheckSnapshotWithCompatibility(root, options.Locale, options.SnapshotID, true, catalog, evaluation.compatibility)
	if err != nil {
		return nil, err
	}
	currentResults, err := readQualityCheckResults(root, options.Locale, current)
	if err != nil {
		return nil, err
	}
	previousSnapshotID := options.PreviousSnapshotID
	if previousSnapshotID == "" && currentResults == nil {
		previousSnapshotID, err = finalizedCarryPredecessor(root, options.Locale, options.SnapshotID)
		if err != nil {
			return nil, err
		}
		if previousSnapshotID == "" {
			previousSnapshotID, err = machineBoundPredecessor(root, current)
			if err != nil {
				return nil, err
			}
		}
	}
	if previousSnapshotID != "" {
		if err := validateSnapshotID(previousSnapshotID); err != nil {
			return nil, err
		}
		if previousSnapshotID == options.SnapshotID {
			return nil, errors.New("previous_snapshot_id must differ from snapshot_id")
		}
	}
	if currentResults != nil {
		previousSnapshotID, err = resolveQualityCheckResultsLineage(options.SnapshotID, currentResults, previousSnapshotID)
		if err != nil {
			return nil, err
		}
	}
	if current.CurrentRevision != nil && previousSnapshotID != "" && previousSnapshotID != filepath.Base(filepath.Dir(current.CurrentRevision.Baseline.Path)) {
		return nil, errors.New("mixed Snapshot requires its bound revision predecessor")
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
		previous, previousEffective, err = evaluation.previousQualityCheckResults(current, previousSnapshotID, map[string]bool{})
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
		if _, err := evaluation.readSnapshotUnitRepositoryEvidence(options.SnapshotID, unit); err != nil {
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
		if unit.RepresentationReconciliation != nil && evaluation.membership[options.SnapshotID] == nil {
			prior, ok, err = evaluation.machineCarry(unit, previousSnapshotID, previousEffective)
			if err != nil {
				return nil, err
			}
		}
		if !ok {
			base.Reason = QualityCheckScopeReasonMissing
			// A changed content-addressed Unit ID is a new input, even when
			// the same physical page occupies the predecessor's stable index.
			// Report the lineage without inheriting its rating or finding.
			if previous != nil {
				for _, old := range previous.Units {
					if old.Index == unit.Index && old.UnitKind == unit.UnitKind && old.SourcePath == unit.SourcePath && old.UnitID != unit.UnitID {
						base.Reason = QualityCheckScopeReasonIdentityChanged
					}
				}
			}
			base.RequiredAction = QualityCheckActionRequired
			scope.Pending = append(scope.Pending, base)
			continue
		}
		resolution := GlossaryCompatibilityResolution{Status: "exact"}
		if previous != nil && !prior.currentReattested {
			resolution = evaluation.compatibility.Resolve(previous.GlossarySHA256, current.GlossarySHA256, "tu:"+unit.UnitID)
		}
		if resolution.Status != "exact" && resolution.Status != "compatible" {
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
		base.CompatibilityChain = append(append([]GlossaryArchiveReference(nil), prior.compatibilityChain...), resolution.Chain...)
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
	scope.ReadyForFinalization = scope.ACount == scope.UnitCount && scope.PendingCount == 0
	scope.ReadyForFinalReview = scope.ReadyForFinalization
	if err := evaluation.membershipChecks(); err != nil {
		return nil, err
	}
	return scope, nil
}

func resolveQualityCheckResultsLineage(snapshotID string, existing *QualityCheckResults, requestedPreviousSnapshotID string) (string, error) {
	if existing == nil {
		return requestedPreviousSnapshotID, nil
	}
	persisted := existing.PreviousSnapshotID
	if requestedPreviousSnapshotID == "" || requestedPreviousSnapshotID == persisted {
		return persisted, nil
	}
	return "", fmt.Errorf(
		"quality-check Snapshot %q results lineage was fixed at the first results write: persisted previous_snapshot_id=%q, requested previous_snapshot_id=%q; resume with the persisted lineage or restart Quality Check recording from the correct Snapshot workflow evidence; do not edit quality-check-results.json",
		snapshotID, persisted, requestedPreviousSnapshotID,
	)
}

func loadEffectiveQualityCheckResults(root, locale, snapshotID string, seen map[string]bool, catalogs ...*Catalog) (*QualityCheckSnapshotManifest, map[string]effectiveQualityCheckResult, error) {
	var catalog *Catalog
	if len(catalogs) > 0 {
		catalog = catalogs[0]
	}
	return newQualityCheckEvaluation(root, locale, catalog).loadEffectiveQualityCheckResults(snapshotID, seen)
}

// Materialize only an already verified immutable original-A proof. This never
// supplies current eligibility. Callers retain their exact dependency rechecks.
func (evaluation *qualityCheckEvaluation) materializeOriginalFinalization(proof *HistoricalFinalizationIntegrity) error {
	f := proof.Finalization
	if !evaluation.originalOnly || f.Locale != evaluation.locale || f.Rubric != TranslationQualityRubric {
		return fmt.Errorf("original-only finalization materialization required")
	}
	s, err := readQualityCheckSnapshot(evaluation.root, evaluation.locale, f.SnapshotID, false)
	if err != nil {
		return err
	}
	if len(f.Units) != s.UnitCount || len(proof.ReviewerGlossaries) != s.UnitCount {
		return fmt.Errorf("original finalization materialization scope mismatch")
	}
	effective := map[string]effectiveQualityCheckResult{}
	for i, accepted := range f.Units {
		if accepted.Rating != "A" || accepted.UnitID != s.Units[i].UnitID || !qualityCheckSnapshotIdentityMatches(accepted.Snapshot, s.Units[i]) {
			return fmt.Errorf("original finalization materialization identity mismatch")
		}
		effective[accepted.UnitID] = effectiveQualityCheckResult{rating: "A", rubric: TranslationQualityRubric, snapshotID: accepted.SourceSnapshotID, unit: accepted.Snapshot}
	}
	evaluation.effective[f.SnapshotID] = effectiveQualityCheckMaterialization{snapshot: s, results: effective}
	return nil
}

func (evaluation *qualityCheckEvaluation) loadEffectiveQualityCheckResults(snapshotID string, seen map[string]bool) (*QualityCheckSnapshotManifest, map[string]effectiveQualityCheckResult, error) {
	if seen[snapshotID] {
		return nil, nil, errors.New("quality-check result lineage contains a cycle")
	}
	if cached, ok := evaluation.effective[snapshotID]; ok {
		if evaluation.stats != nil {
			evaluation.stats.EffectiveCacheHits++
		}
		return cached.snapshot, cloneEffectiveQualityCheckResults(cached.results), nil
	}
	// A sealed accepted scope is an explicit boundary, not another predecessor
	// to traverse. Assess today's exact candidates once; affected/unknown units
	// are omitted from carry and remain pending in the ordinary QC workflow.
	proof, proofErr := loadSealedTranslationHistory(evaluation.root, evaluation.catalog, evaluation.locale, snapshotID)
	if proofErr == nil && proof == nil && evaluation.catalog != nil && len(evaluation.catalog.TranslationScope) > 0 {
		// First migration may expose affected units, so a complete checkpoint
		// cannot exist yet. The same historical/current primitive still carries
		// exact eligible A units into ordinary affected-only QC; it never writes
		// a partial authorization or consults global compatibility inventory.
		if _, err := os.Lstat(qualityCheckFinalizationPath(evaluation.root, evaluation.locale, snapshotID)); err == nil {
			proof, proofErr = VerifyHistoricalQualityCheckFinalization(evaluation.root, evaluation.locale, snapshotID, evaluation.catalog)
		} else if !os.IsNotExist(err) {
			proofErr = err
		}
	}
	if proofErr != nil {
		return nil, nil, proofErr
	} else if proof != nil {
		if evaluation.originalOnly {
			if err := evaluation.materializeOriginalFinalization(proof); err != nil {
				return nil, nil, err
			}
			materialized := evaluation.effective[snapshotID]
			return materialized.snapshot, cloneEffectiveQualityCheckResults(materialized.results), nil
		}
		a, err := assessCurrentTranslationProof(evaluation.root, evaluation.catalog, evaluation.locale, *proof)
		if err != nil {
			return nil, nil, err
		}
		s, err := readQualityCheckSnapshot(evaluation.root, evaluation.locale, snapshotID, false)
		if err != nil {
			return nil, nil, err
		}
		effective := map[string]effectiveQualityCheckResult{}
		for i, u := range a.Units {
			if u.Classification == "compatible" && u.Validation == "passed" {
				f := proof.Finalization.Units[i]
				effective[f.UnitID] = effectiveQualityCheckResult{rating: "A", rubric: TranslationQualityRubric, snapshotID: f.SourceSnapshotID, unit: f.Snapshot, currentReattested: true}
			}
		}
		evaluation.effective[snapshotID] = effectiveQualityCheckMaterialization{snapshot: s, results: effective}
		return s, cloneEffectiveQualityCheckResults(effective), nil
	}
	seen[snapshotID] = true
	defer delete(seen, snapshotID)
	snapshot, err := readQualityCheckSnapshot(evaluation.root, evaluation.locale, snapshotID, false)
	if err != nil {
		return nil, nil, err
	}
	results, err := readQualityCheckResults(evaluation.root, evaluation.locale, snapshot)
	if err != nil {
		return nil, nil, err
	}
	effective := map[string]effectiveQualityCheckResult{}
	if results == nil {
		previous, err := finalizedCarryPredecessor(evaluation.root, evaluation.locale, snapshotID)
		if err != nil {
			return nil, nil, err
		}
		results = &QualityCheckResults{PreviousSnapshotID: previous}
		if previous == "" {
			results.PreviousSnapshotID, err = machineBoundPredecessor(evaluation.root, snapshot)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	if results.PreviousSnapshotID != "" {
		previous, inherited, err := evaluation.previousQualityCheckResults(snapshot, results.PreviousSnapshotID, seen)
		if err != nil {
			return nil, nil, err
		}
		for _, unit := range snapshot.Units {
			prior, ok := inherited[unit.UnitID]
			if unit.RepresentationReconciliation != nil && evaluation.membership[snapshot.SnapshotID] == nil {
				prior, ok, err = evaluation.machineCarry(unit, results.PreviousSnapshotID, inherited)
				if err != nil {
					return nil, nil, err
				}
			}
			resolution := GlossaryCompatibilityResolution{Status: "exact"}
			if !prior.currentReattested {
				compatibility := evaluation.compatibility
				if evaluation.originalOnly && compatibility.historicalSHA == "" {
					compatibility = PrepareGlossaryCompatibility(evaluation.root, evaluation.locale, evaluation.catalog)
					compatibility.historicalSHA = snapshot.GlossarySHA256
				}
				resolution = compatibility.Resolve(previous.GlossarySHA256, snapshot.GlossarySHA256, "tu:"+unit.UnitID)
			}
			if ok && qualityCheckSnapshotIdentityMatches(unit, prior.unit) && (resolution.Status == "exact" || (prior.rating == "A" && resolution.Status == "compatible")) {
				prior.compatibilityChain = append(append([]GlossaryArchiveReference(nil), prior.compatibilityChain...), resolution.Chain...)
				effective[unit.UnitID] = prior
			}
		}
	}
	for _, result := range results.Results {
		unit := snapshot.Units[result.Index-1]
		effective[result.UnitID] = effectiveQualityCheckResult{rating: result.Rating, finding: result.Finding, rubric: results.Rubric, snapshotID: snapshotID, unit: unit}
	}
	if evaluation.stats != nil {
		evaluation.stats.EffectiveMaterializations[snapshotID]++
	}
	if err := evaluation.membershipChecks(); err != nil {
		return nil, nil, err
	}
	evaluation.effective[snapshotID] = effectiveQualityCheckMaterialization{snapshot: snapshot, results: cloneEffectiveQualityCheckResults(effective)}
	return snapshot, effective, nil
}

func readOriginalQualityCheckResults(root, locale string, snapshot *QualityCheckSnapshotManifest) (*QualityCheckResults, error) {
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
	if (results.SchemaVersion != QualityCheckResultsSchemaVersion && results.SchemaVersion != legacyQualityCheckResultsSchemaVersion) || results.EvidenceType != QualityCheckResultsEvidenceType ||
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
		if results.SchemaVersion >= QualityCheckResultsSchemaVersion && result.Rating != "A" && strings.TrimSpace(result.Finding) == "" {
			return nil, fmt.Errorf("quality-check results %q require finding for rating %s at index %d", snapshot.SnapshotID, result.Rating, result.Index)
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
		current.Attempt == prior.Attempt && reflect.DeepEqual(current.RepresentationReconciliation, prior.RepresentationReconciliation)
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
