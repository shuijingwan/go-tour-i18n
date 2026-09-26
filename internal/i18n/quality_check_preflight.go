package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type QualityCheckPreflightOptions struct {
	Locale, SnapshotID, PreviousSnapshotID string
	FullRereview                           bool
}

type QualityCheckPreflight struct {
	Locale                    string                  `json:"locale"`
	SnapshotID                string                  `json:"snapshot_id"`
	Mode                      string                  `json:"mode"`
	ResultsStarted            bool                    `json:"results_started"`
	PersistedPreviousSnapshot string                  `json:"persisted_previous_snapshot_id,omitempty"`
	SelectedPreviousSnapshot  string                  `json:"selected_previous_snapshot_id,omitempty"`
	FinalizedSnapshots        []string                `json:"finalized_snapshots"`
	CarryForwardCount         int                     `json:"carry_forward_count"`
	QualityCheckRequiredCount int                     `json:"quality_check_required_count"`
	RevisionRequiredCount     int                     `json:"revision_required_count"`
	EstimatedReviewerUnits    int                     `json:"estimated_reviewer_units"`
	IntentionalRepeatedReview bool                    `json:"intentional_repeated_review"`
	Pending                   []QualityCheckScopeUnit `json:"pending_units"`
}

func BuildQualityCheckPreflight(root string, catalog *Catalog, options QualityCheckPreflightOptions) (*QualityCheckPreflight, error) {
	if options.FullRereview && options.PreviousSnapshotID != "" {
		return nil, fmt.Errorf("previous_snapshot_id and full_rereview are mutually exclusive")
	}
	scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{
		Locale: options.Locale, SnapshotID: options.SnapshotID, PreviousSnapshotID: options.PreviousSnapshotID,
	})
	if err != nil {
		return nil, err
	}
	snapshot, err := readQualityCheckSnapshotForReview(root, options.Locale, options.SnapshotID)
	if err != nil {
		return nil, err
	}
	results, err := readQualityCheckResults(root, options.Locale, snapshot)
	if err != nil {
		return nil, err
	}
	if options.FullRereview && results != nil {
		return nil, fmt.Errorf("full_rereview may only be selected before results start")
	}
	finalized, err := qualityCheckFinalizedSnapshotIDs(root, options.Locale)
	if err != nil {
		return nil, err
	}
	report := &QualityCheckPreflight{
		Locale: options.Locale, SnapshotID: options.SnapshotID,
		ResultsStarted: results != nil, SelectedPreviousSnapshot: scope.PreviousSnapshotID,
		FinalizedSnapshots: finalized, CarryForwardCount: scope.CarryForwardCount,
		Pending: append([]QualityCheckScopeUnit(nil), scope.Pending...),
	}
	if results != nil {
		report.PersistedPreviousSnapshot = results.PreviousSnapshotID
	}
	switch {
	case scope.PreviousSnapshotID != "":
		report.Mode = "incremental"
	case len(finalized) == 0:
		report.Mode = "initial"
	case options.FullRereview:
		report.Mode = "full-rereview"
		report.IntentionalRepeatedReview = true
	default:
		report.Mode = "predecessor-required"
	}
	for _, pending := range scope.Pending {
		switch pending.RequiredAction {
		case QualityCheckActionRequired:
			report.QualityCheckRequiredCount++
		case QualityCheckActionRevisionRequired:
			report.RevisionRequiredCount++
		}
	}
	report.EstimatedReviewerUnits = report.QualityCheckRequiredCount + report.RevisionRequiredCount
	return report, nil
}

func qualityCheckFinalizedSnapshotIDs(root, locale string) ([]string, error) {
	directory := filepath.Join(root, "data", "quality-check-snapshots", locale)
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := qualityCheckFinalizationPath(root, locale, entry.Name())
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var finalization QualityCheckFinalization
		if err := json.Unmarshal(data, &finalization); err != nil ||
			finalization.SchemaVersion != QualityCheckFinalizationSchemaVersion ||
			finalization.EvidenceType != QualityCheckFinalizationEvidenceType ||
			finalization.Locale != locale || finalization.SnapshotID != entry.Name() {
			return nil, fmt.Errorf("quality-check finalization %s has incompatible identity", entry.Name())
		}
		ids = append(ids, entry.Name())
	}
	sort.Strings(ids)
	return ids, nil
}
