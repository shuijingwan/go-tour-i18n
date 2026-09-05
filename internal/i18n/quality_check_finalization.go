package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const QualityCheckFinalizationSchemaVersion = 1
const QualityCheckFinalizationEvidenceType = "quality_check_finalization"

// QualityCheckFinalization is the sole promotion authority for new work. It
// records a mechanical proof that the complete Snapshot resolves to QC A.
type QualityCheckFinalization struct {
	SchemaVersion          int                             `json:"schema_version"`
	EvidenceType           string                          `json:"evidence_type"`
	Locale                 string                          `json:"locale"`
	SnapshotID             string                          `json:"snapshot_id"`
	SnapshotManifestPath   string                          `json:"snapshot_manifest_path"`
	SnapshotManifestSHA256 string                          `json:"snapshot_manifest_sha256"`
	GlossaryPath           string                          `json:"glossary_path"`
	GlossarySHA256         string                          `json:"glossary_sha256"`
	Rubric                 string                          `json:"rubric"`
	QCResults              []QualityCheckFinalizationInput `json:"quality_check_results"`
	Units                  []QualityCheckFinalizationUnit  `json:"units"`
	FinalizedAt            string                          `json:"finalized_at"`
}
type QualityCheckFinalizationInput struct {
	SnapshotID string `json:"snapshot_id"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
}
type QualityCheckFinalizationUnit struct {
	Index            int                      `json:"index"`
	UnitID           string                   `json:"unit_id"`
	SourceSnapshotID string                   `json:"source_snapshot_id"`
	Rating           string                   `json:"rating"`
	Snapshot         QualityCheckSnapshotUnit `json:"snapshot"`
}
type QualityCheckFinalizeOptions struct {
	Locale, SnapshotID string
	Now                func() time.Time
}

func qualityCheckFinalizationPath(root, locale, snapshotID string) string {
	return filepath.Join(root, "data", "quality-check-snapshots", locale, snapshotID, "finalization.json")
}

func FinalizeQualityCheck(root string, catalog *Catalog, options QualityCheckFinalizeOptions) (*QualityCheckFinalization, string, error) {
	if catalog == nil {
		return nil, "", errors.New("quality-check catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, "", err
	}
	if err := validateSnapshotID(options.SnapshotID); err != nil {
		return nil, "", err
	}
	path := qualityCheckFinalizationPath(root, options.Locale, options.SnapshotID)
	if _, err := os.Stat(path); err == nil {
		return nil, "", fmt.Errorf("quality-check finalization already exists: %s", filepath.ToSlash(path))
	} else if !os.IsNotExist(err) {
		return nil, "", err
	}
	finalization, err := buildQualityCheckFinalization(root, catalog, options.Locale, options.SnapshotID, options.Now)
	if err != nil {
		return nil, "", err
	}
	if err := writeQualityCheckFinalization(path, finalization); err != nil {
		return nil, "", err
	}
	rel, err := repositoryRelativePath(root, path)
	if err != nil {
		return nil, "", err
	}
	return finalization, rel, nil
}

func buildQualityCheckFinalization(root string, catalog *Catalog, locale, snapshotID string, now func() time.Time) (*QualityCheckFinalization, error) {
	snapshot, err := readQualityCheckSnapshotForReview(root, locale, snapshotID)
	if err != nil {
		return nil, err
	}
	scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: locale, SnapshotID: snapshotID})
	if err != nil {
		return nil, err
	}
	if !scope.ReadyForFinalization || scope.ACount != snapshot.UnitCount || scope.BCount != 0 || scope.CCount != 0 || scope.DCount != 0 || scope.PendingCount != 0 {
		return nil, errors.New("quality-check finalization requires complete A-only Quality Check coverage")
	}
	_, effective, err := loadEffectiveQualityCheckResults(root, locale, snapshotID, map[string]bool{})
	if err != nil {
		return nil, err
	}
	inputs, err := qualityCheckFinalizationInputs(root, locale, snapshotID, map[string]bool{})
	if err != nil {
		return nil, err
	}
	manifestData, err := os.ReadFile(qualityCheckSnapshotManifestPath(root, locale, snapshotID))
	if err != nil {
		return nil, err
	}
	units := make([]QualityCheckFinalizationUnit, 0, len(snapshot.Units))
	for _, unit := range snapshot.Units {
		if _, err := readSnapshotUnitRepositoryEvidence(root, catalog, locale, unit); err != nil {
			return nil, fmt.Errorf("snapshot index %d (%s) identity: %w", unit.Index, unit.UnitID, err)
		}
		result, ok := effective[unit.UnitID]
		if !ok || result.rating != "A" || result.rubric != TranslationQualityRubric || !qualityCheckSnapshotIdentityMatches(unit, result.unit) {
			return nil, fmt.Errorf("snapshot index %d (%s) lacks an identity-matching A Quality Check", unit.Index, unit.UnitID)
		}
		units = append(units, QualityCheckFinalizationUnit{Index: unit.Index, UnitID: unit.UnitID, SourceSnapshotID: result.snapshotID, Rating: result.rating, Snapshot: unit})
	}
	if now == nil {
		now = time.Now
	}
	return &QualityCheckFinalization{SchemaVersion: QualityCheckFinalizationSchemaVersion, EvidenceType: QualityCheckFinalizationEvidenceType, Locale: locale, SnapshotID: snapshotID, SnapshotManifestPath: filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, snapshotID, "manifest.json")), SnapshotManifestSHA256: sum(manifestData), GlossaryPath: snapshot.GlossaryPath, GlossarySHA256: snapshot.GlossarySHA256, Rubric: TranslationQualityRubric, QCResults: inputs, Units: units, FinalizedAt: now().UTC().Format(time.RFC3339Nano)}, nil
}

func qualityCheckFinalizationInputs(root, locale, snapshotID string, seen map[string]bool) ([]QualityCheckFinalizationInput, error) {
	if seen[snapshotID] {
		return nil, errors.New("quality-check result lineage contains a cycle")
	}
	seen[snapshotID] = true
	defer delete(seen, snapshotID)
	s, err := readQualityCheckSnapshot(root, locale, snapshotID, false)
	if err != nil {
		return nil, err
	}
	r, err := readQualityCheckResults(root, locale, s)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("quality-check results are missing for snapshot %q", snapshotID)
	}
	all := []QualityCheckFinalizationInput{}
	if r.PreviousSnapshotID != "" {
		all, err = qualityCheckFinalizationInputs(root, locale, r.PreviousSnapshotID, seen)
		if err != nil {
			return nil, err
		}
	}
	p := qualityCheckResultsPath(root, locale, snapshotID)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	rel, err := repositoryRelativePath(root, p)
	if err != nil {
		return nil, err
	}
	all = append(all, QualityCheckFinalizationInput{SnapshotID: snapshotID, Path: rel, SHA256: sum(b)})
	return all, nil
}

func writeQualityCheckFinalization(path string, v *QualityCheckFinalization) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func VerifyQualityCheckFinalization(root string, catalog *Catalog, locale, snapshotID string) (*QualityCheckFinalization, error) {
	p := qualityCheckFinalizationPath(root, locale, snapshotID)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read quality-check finalization: %w", err)
	}
	var got QualityCheckFinalization
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&got); err != nil {
		return nil, fmt.Errorf("parse quality-check finalization: %w", err)
	}
	want, err := buildQualityCheckFinalization(root, catalog, locale, snapshotID, func() time.Time {
		t, e := time.Parse(time.RFC3339Nano, got.FinalizedAt)
		if e != nil {
			return time.Time{}
		}
		return t
	})
	if err != nil {
		return nil, err
	}
	if got.FinalizedAt == "" || !bytes.Equal(mustJSON(got), mustJSON(*want)) {
		return nil, errors.New("quality-check finalization does not match current A-only Snapshot evidence")
	}
	return &got, nil
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
