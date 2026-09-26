package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const qualityCheckSurfaceReopenSchemaVersion = 1
const qualityCheckSurfaceReopenEvidenceType = "quality_check_surface_reopen"

type QualityCheckSurfaceReopenUnit struct {
	Index    int                      `json:"index"`
	UnitID   string                   `json:"unit_id"`
	Snapshot QualityCheckSnapshotUnit `json:"snapshot"`
}

type QualityCheckSurfaceReopen struct {
	SchemaVersion              int                             `json:"schema_version"`
	EvidenceType               string                          `json:"evidence_type"`
	Locale                     string                          `json:"locale"`
	ReopenID                   string                          `json:"reopen_id"`
	PreviousSnapshotID         string                          `json:"previous_snapshot_id"`
	PreviousFinalizationPath   string                          `json:"previous_finalization_path"`
	PreviousFinalizationSHA256 string                          `json:"previous_finalization_sha256"`
	SurfaceReviewID            string                          `json:"surface_review_id"`
	SurfaceReviewPath          string                          `json:"surface_review_path"`
	SurfaceReviewSHA256        string                          `json:"surface_review_sha256"`
	Finding                    string                          `json:"finding"`
	UnitCount                  int                             `json:"unit_count"`
	Units                      []QualityCheckSurfaceReopenUnit `json:"units"`
}

type QualityCheckSurfaceReopenOptions struct {
	Locale, ReopenID, PreviousSnapshotID, SurfaceReviewID, Finding string
	UnitIDs                                                        []string
}

func qualityCheckSurfaceReopenPath(root, locale, reopenID string) string {
	return filepath.Join(root, "data", "quality-check-reopens", locale, reopenID+".json")
}

func RecordQualityCheckSurfaceReopen(root string, catalog *Catalog, options QualityCheckSurfaceReopenOptions) (*QualityCheckSurfaceReopen, string, error) {
	if catalog == nil {
		return nil, "", errors.New("quality-check catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, "", err
	}
	if !reviewIDPattern.MatchString(options.ReopenID) || !reviewIDPattern.MatchString(options.SurfaceReviewID) {
		return nil, "", errors.New("reopen_id and surface_review_id must be stable identifiers")
	}
	if err := validateSnapshotID(options.PreviousSnapshotID); err != nil {
		return nil, "", err
	}
	finding := strings.TrimSpace(options.Finding)
	if finding == "" || len(options.UnitIDs) == 0 {
		return nil, "", errors.New("surface correction requires a non-empty finding and exact unit_id scope")
	}
	if len(options.UnitIDs) > DefaultRetranslationExportLimit {
		return nil, "", fmt.Errorf("surface correction scope must not exceed %d TranslationUnits", DefaultRetranslationExportLimit)
	}
	finalization, err := VerifyQualityCheckFinalization(root, catalog, options.Locale, options.PreviousSnapshotID)
	if err != nil {
		return nil, "", fmt.Errorf("surface correction requires a current finalized predecessor: %w", err)
	}
	finalizationPath := qualityCheckFinalizationPath(root, options.Locale, options.PreviousSnapshotID)
	finalizationData, err := os.ReadFile(finalizationPath)
	if err != nil {
		return nil, "", err
	}
	surfacePath := filepath.Join(root, "data", "locale-surface-reviews", options.Locale, options.SurfaceReviewID+".md")
	surfaceData, err := os.ReadFile(surfacePath)
	if err != nil {
		return nil, "", fmt.Errorf("read formal Surface Review finding: %w", err)
	}
	if !bytes.Contains(surfaceData, []byte(finding)) {
		return nil, "", errors.New("formal Surface Review evidence does not contain the exact finding")
	}
	snapshot, err := readQualityCheckSnapshotForReview(root, options.Locale, options.PreviousSnapshotID)
	if err != nil {
		return nil, "", err
	}
	byID := make(map[string]QualityCheckSnapshotUnit, len(snapshot.Units))
	for _, unit := range snapshot.Units {
		byID[unit.UnitID] = unit
	}
	seen := map[string]bool{}
	units := make([]QualityCheckSurfaceReopenUnit, 0, len(options.UnitIDs))
	var kind UnitKind
	for _, unitID := range options.UnitIDs {
		unit, ok := byID[unitID]
		if !ok || seen[unitID] {
			return nil, "", fmt.Errorf("surface correction has unknown or duplicate unit_id %q", unitID)
		}
		if !surfaceReviewNamesUnit(surfaceData, unitID) {
			return nil, "", fmt.Errorf("formal Surface Review finding does not name unit_id %s", unitID)
		}
		if kind == "" {
			kind = unit.UnitKind
		} else if kind != unit.UnitKind {
			return nil, "", errors.New("surface correction scope must not mix Page and Example TranslationUnits")
		}
		if _, err := readSnapshotUnitRepositoryEvidence(root, catalog, options.Locale, unit); err != nil {
			return nil, "", fmt.Errorf("surface correction unit %s identity: %w", unitID, err)
		}
		seen[unitID] = true
		units = append(units, QualityCheckSurfaceReopenUnit{Index: unit.Index, UnitID: unit.UnitID, Snapshot: unit})
	}
	sort.Slice(units, func(i, j int) bool { return units[i].Index < units[j].Index })
	receipt := &QualityCheckSurfaceReopen{
		SchemaVersion: qualityCheckSurfaceReopenSchemaVersion, EvidenceType: qualityCheckSurfaceReopenEvidenceType,
		Locale: options.Locale, ReopenID: options.ReopenID, PreviousSnapshotID: options.PreviousSnapshotID,
		PreviousFinalizationPath:   filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", options.Locale, options.PreviousSnapshotID, "finalization.json")),
		PreviousFinalizationSHA256: sum(finalizationData),
		SurfaceReviewID:            options.SurfaceReviewID,
		SurfaceReviewPath:          filepath.ToSlash(filepath.Join("data", "locale-surface-reviews", options.Locale, options.SurfaceReviewID+".md")),
		SurfaceReviewSHA256:        sum(surfaceData), Finding: finding, UnitCount: len(units), Units: units,
	}
	if finalization.SnapshotID != receipt.PreviousSnapshotID {
		return nil, "", errors.New("surface correction finalization identity mismatch")
	}
	path := qualityCheckSurfaceReopenPath(root, options.Locale, options.ReopenID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, "", err
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return nil, "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, "", fmt.Errorf("quality-check surface reopen already exists or cannot be created: %w", err)
	}
	_, writeErr := file.Write(append(data, '\n'))
	closeErr := file.Close()
	if writeErr != nil {
		return nil, "", writeErr
	}
	if closeErr != nil {
		return nil, "", closeErr
	}
	relative, err := repositoryRelativePath(root, path)
	if err != nil {
		return nil, "", err
	}
	return receipt, relative, nil
}

func surfaceReviewNamesUnit(data []byte, unitID string) bool {
	boundary := `[^A-Za-z0-9_.\/-]`
	pattern := `(^|` + boundary + `)` + regexp.QuoteMeta(unitID) + `($|` + boundary + `)`
	return regexp.MustCompile(pattern).Find(data) != nil
}

func readCurrentQualityCheckSurfaceReopen(root string, catalog *Catalog, locale, reopenID string) (*QualityCheckSurfaceReopen, error) {
	if !reviewIDPattern.MatchString(reopenID) {
		return nil, fmt.Errorf("invalid quality-check surface reopen id %q", reopenID)
	}
	path := qualityCheckSurfaceReopenPath(root, locale, reopenID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read quality-check surface reopen: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var receipt QualityCheckSurfaceReopen
	if err := decoder.Decode(&receipt); err != nil {
		return nil, fmt.Errorf("parse quality-check surface reopen: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("parse quality-check surface reopen: multiple JSON values")
	}
	if receipt.SchemaVersion != qualityCheckSurfaceReopenSchemaVersion || receipt.EvidenceType != qualityCheckSurfaceReopenEvidenceType ||
		receipt.Locale != locale || receipt.ReopenID != reopenID || receipt.Finding == "" || receipt.UnitCount < 1 || receipt.UnitCount != len(receipt.Units) {
		return nil, errors.New("quality-check surface reopen has incompatible identity")
	}
	expectedFinalizationPath := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, receipt.PreviousSnapshotID, "finalization.json"))
	expectedSurfacePath := filepath.ToSlash(filepath.Join("data", "locale-surface-reviews", locale, receipt.SurfaceReviewID+".md"))
	if !reviewIDPattern.MatchString(receipt.SurfaceReviewID) || receipt.PreviousFinalizationPath != expectedFinalizationPath || receipt.SurfaceReviewPath != expectedSurfacePath || receipt.UnitCount > DefaultRetranslationExportLimit {
		return nil, errors.New("quality-check surface reopen has unsafe evidence paths or scope")
	}
	if _, err := VerifyQualityCheckFinalization(root, catalog, locale, receipt.PreviousSnapshotID); err != nil {
		return nil, fmt.Errorf("quality-check surface reopen predecessor is not current: %w", err)
	}
	finalizationData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(receipt.PreviousFinalizationPath)))
	if err != nil || sum(finalizationData) != receipt.PreviousFinalizationSHA256 {
		return nil, errors.New("quality-check surface reopen finalization evidence is stale")
	}
	surfaceData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(receipt.SurfaceReviewPath)))
	if err != nil || sum(surfaceData) != receipt.SurfaceReviewSHA256 {
		return nil, errors.New("quality-check surface reopen Surface Review finding is stale")
	}
	if strings.TrimSpace(receipt.Finding) != receipt.Finding || !bytes.Contains(surfaceData, []byte(receipt.Finding)) {
		return nil, errors.New("quality-check surface reopen finding is stale")
	}
	snapshot, err := readQualityCheckSnapshotForReview(root, locale, receipt.PreviousSnapshotID)
	if err != nil {
		return nil, err
	}
	last := 0
	for _, reopened := range receipt.Units {
		if reopened.Index <= last || reopened.Index < 1 || reopened.Index > len(snapshot.Units) ||
			reopened.UnitID != snapshot.Units[reopened.Index-1].UnitID ||
			!qualityCheckSnapshotIdentityMatches(reopened.Snapshot, snapshot.Units[reopened.Index-1]) ||
			!surfaceReviewNamesUnit(surfaceData, reopened.UnitID) {
			return nil, errors.New("quality-check surface reopen exact TranslationUnit scope is stale")
		}
		last = reopened.Index
	}
	return &receipt, nil
}
