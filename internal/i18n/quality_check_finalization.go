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
	SchemaVersion           int                             `json:"schema_version"`
	EvidenceType            string                          `json:"evidence_type"`
	Locale                  string                          `json:"locale"`
	SnapshotID              string                          `json:"snapshot_id"`
	PreviousSnapshotID      string                          `json:"previous_snapshot_id,omitempty"`
	PredecessorFinalization *GlossaryArchiveReference       `json:"predecessor_finalization,omitempty"`
	SnapshotManifestPath    string                          `json:"snapshot_manifest_path"`
	SnapshotManifestSHA256  string                          `json:"snapshot_manifest_sha256"`
	GlossaryPath            string                          `json:"glossary_path"`
	GlossarySHA256          string                          `json:"glossary_sha256"`
	Rubric                  string                          `json:"rubric"`
	QCResults               []QualityCheckFinalizationInput `json:"quality_check_results"`
	Units                   []QualityCheckFinalizationUnit  `json:"units"`
	FinalizedAt             string                          `json:"finalized_at"`
}
type QualityCheckFinalizationInput struct {
	SnapshotID string `json:"snapshot_id"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
}
type QualityCheckFinalizationUnit struct {
	CompatibilityChain []GlossaryArchiveReference `json:"compatibility_chain,omitempty"`
	Index              int                        `json:"index"`
	UnitID             string                     `json:"unit_id"`
	SourceSnapshotID   string                     `json:"source_snapshot_id"`
	Rating             string                     `json:"rating"`
	Snapshot           QualityCheckSnapshotUnit   `json:"snapshot"`
}
type QualityCheckFinalizeOptions struct {
	Locale, SnapshotID, PreviousSnapshotID string
	Now                                    func() time.Time
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
	if err := RequireCurrentGlossaryReview(root, options.Locale); err != nil {
		return nil, "", err
	}
	path := qualityCheckFinalizationPath(root, options.Locale, options.SnapshotID)
	snapshot, err := readQualityCheckSnapshot(root, options.Locale, options.SnapshotID, false)
	if err != nil {
		return nil, "", err
	}
	glossary, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(options.Locale))
	if err != nil || sum(glossary) != snapshot.GlossarySHA256 {
		return nil, "", errors.New("new finalization requires a Snapshot of the current complete glossary")
	}
	if _, err := os.Stat(path); err == nil {
		return nil, "", fmt.Errorf("quality-check finalization already exists: %s", filepath.ToSlash(path))
	} else if !os.IsNotExist(err) {
		return nil, "", err
	}
	finalization, err := buildQualityCheckFinalizationWithEvaluation(newQualityCheckEvaluation(root, options.Locale, catalog), options.SnapshotID, options.Now, options.PreviousSnapshotID)
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
	evaluation := newQualityCheckEvaluation(root, locale, catalog)
	return buildQualityCheckFinalizationWithEvaluation(evaluation, snapshotID, now)
}

func buildQualityCheckFinalizationWithEvaluation(evaluation *qualityCheckEvaluation, snapshotID string, now func() time.Time, predecessors ...string) (*QualityCheckFinalization, error) {
	root, catalog, locale := evaluation.root, evaluation.catalog, evaluation.locale
	previous := ""
	var predecessor *GlossaryArchiveReference
	if len(predecessors) > 0 {
		previous = predecessors[0]
	}
	snapshot, err := readQualityCheckSnapshotWithCompatibility(root, locale, snapshotID, true, catalog, evaluation.compatibility)
	if err != nil {
		return nil, err
	}
	scope, err := buildQualityCheckScopeWithEvaluation(evaluation, QualityCheckScopeOptions{Locale: locale, SnapshotID: snapshotID, PreviousSnapshotID: previous})
	if err != nil {
		return nil, err
	}
	if !scope.ReadyForFinalization || scope.ACount != snapshot.UnitCount || scope.BCount != 0 || scope.CCount != 0 || scope.DCount != 0 || scope.PendingCount != 0 {
		return nil, errors.New("quality-check finalization requires complete A-only Quality Check coverage")
	}
	bindPredecessorFinalization := previous != ""
	if bindPredecessorFinalization {
		_, err := os.Lstat(qualityCheckFinalizationPath(root, locale, previous))
		if os.IsNotExist(err) {
			// Ordinary revision carries reviewed A from a mixed-rating Snapshot.
			// Its immutable QC inputs supply lineage; that Snapshot cannot have
			// A-only finalization yet. Zero-review carry still requires one.
			currentResults, e := readQualityCheckResults(root, locale, snapshot)
			if e != nil {
				return nil, e
			}
			if currentResults == nil {
				return nil, fmt.Errorf("zero-review carry requires predecessor finalization")
			}
			bindPredecessorFinalization = false
		} else if err != nil {
			return nil, err
		}
	}
	if bindPredecessorFinalization {
		// The predecessor supplies its historical A proof. Current carry is
		// separately decided above by QC scope/re-attestation, never by asking
		// that predecessor to become a new current judgment.
		_, predecessorErr := VerifyHistoricalQualityCheckFinalization(root, locale, previous, catalog)
		if predecessorErr != nil {
			return nil, fmt.Errorf("predecessor finalization: %w", predecessorErr)
		}
		ref, err := successorReference(root, filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, previous, "finalization.json")))
		if err != nil {
			return nil, err
		}
		predecessor = &ref
	}
	_, effective, err := evaluation.loadEffectiveQualityCheckResults(snapshotID, map[string]bool{})
	if err != nil {
		return nil, err
	}
	for _, carry := range scope.CarryForward {
		u := snapshot.Units[carry.Index-1]
		effective[carry.UnitID] = effectiveQualityCheckResult{rating: carry.Rating, rubric: TranslationQualityRubric, snapshotID: carry.FromSnapshotID, unit: u, compatibilityChain: carry.CompatibilityChain}
	}
	inputs, err := qualityCheckFinalizationInputs(root, locale, snapshotID, map[string]bool{}, previous)
	if err != nil {
		return nil, err
	}
	manifestData, err := os.ReadFile(qualityCheckSnapshotManifestPath(root, locale, snapshotID))
	if err != nil {
		return nil, err
	}
	units := make([]QualityCheckFinalizationUnit, 0, len(snapshot.Units))
	for _, unit := range snapshot.Units {
		if _, err := evaluation.readSnapshotUnitRepositoryEvidence(snapshotID, unit); err != nil {
			return nil, fmt.Errorf("snapshot index %d (%s) identity: %w", unit.Index, unit.UnitID, err)
		}
		result, ok := effective[unit.UnitID]
		if !ok || result.rating != "A" || result.rubric != TranslationQualityRubric || !qualityCheckSnapshotIdentityMatches(unit, result.unit) {
			return nil, fmt.Errorf("snapshot index %d (%s) lacks an identity-matching A Quality Check", unit.Index, unit.UnitID)
		}
		units = append(units, QualityCheckFinalizationUnit{Index: unit.Index, UnitID: unit.UnitID, SourceSnapshotID: result.snapshotID, Rating: result.rating, Snapshot: unit, CompatibilityChain: result.compatibilityChain})
	}
	if now == nil {
		now = time.Now
	}
	return &QualityCheckFinalization{SchemaVersion: QualityCheckFinalizationSchemaVersion, EvidenceType: QualityCheckFinalizationEvidenceType, Locale: locale, SnapshotID: snapshotID, PreviousSnapshotID: previous, PredecessorFinalization: predecessor, SnapshotManifestPath: filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, snapshotID, "manifest.json")), SnapshotManifestSHA256: sum(manifestData), GlossaryPath: snapshot.GlossaryPath, GlossarySHA256: snapshot.GlossarySHA256, Rubric: TranslationQualityRubric, QCResults: inputs, Units: units, FinalizedAt: now().UTC().Format(time.RFC3339Nano)}, nil
}

// Only immutable finalization may supply lineage in the absence of responses.
func finalizedCarryPredecessor(root, locale, snapshotID string) (string, error) {
	b, err := readCompatibilityFile(root, filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, snapshotID, "finalization.json")))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var f QualityCheckFinalization
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &f); err != nil {
		return "", err
	}
	if f.Locale != locale || f.SnapshotID != snapshotID || f.SchemaVersion != QualityCheckFinalizationSchemaVersion || f.EvidenceType != QualityCheckFinalizationEvidenceType {
		return "", errors.New("invalid carry finalization identity")
	}
	if f.PreviousSnapshotID != "" {
		if err := validateSnapshotID(f.PreviousSnapshotID); err != nil {
			return "", err
		}
		if f.PreviousSnapshotID == snapshotID {
			return "", errors.New("quality-check result lineage contains a cycle")
		}
	}
	return f.PreviousSnapshotID, nil
}

func qualityCheckFinalizationInputs(root, locale, snapshotID string, seen map[string]bool, predecessors ...string) ([]QualityCheckFinalizationInput, error) {
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
		previous := ""
		if len(predecessors) > 0 {
			previous = predecessors[0]
		}
		if previous == "" {
			previous, err = finalizedCarryPredecessor(root, locale, snapshotID)
			if err != nil {
				return nil, err
			}
		}
		if previous != "" {
			return qualityCheckFinalizationInputs(root, locale, previous, seen)
		}
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
	original, err := readOriginalQualityCheckResults(root, locale, s)
	if err != nil {
		return nil, err
	}
	_, corrections, err := readQualityCheckCorrections(root, s, original)
	if err != nil {
		return nil, err
	}
	for _, ref := range corrections {
		all = append(all, QualityCheckFinalizationInput{SnapshotID: snapshotID, Path: ref.Path, SHA256: ref.SHA256})
	}
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
	// Compatibility entry point for current consumers. A sealed current proof
	// avoids historical evaluation; the historical API never enters this path.
	if a, err := FindCurrentTranslationAuthorization(root, catalog, locale, snapshotID); err != nil {
		return nil, err
	} else if a != nil {
		return &a.Historical.Finalization, nil
	}
	return verifyLegacyCurrentQualityCheckFinalization(root, catalog, locale, snapshotID)
}

// Retained for unmigrated legacy callers; new registered package consumers use
// RequireCurrentTranslationAuthorization. No existing receipt is rewritten.
func verifyLegacyCurrentQualityCheckFinalization(root string, catalog *Catalog, locale, snapshotID string) (*QualityCheckFinalization, error) {
	if err := RequireCurrentGlossaryReview(root, locale); err != nil {
		return nil, err
	}
	p := qualityCheckFinalizationPath(root, locale, snapshotID)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read quality-check finalization: %w", err)
	}
	var got QualityCheckFinalization
	if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &got); err != nil {
		return nil, fmt.Errorf("parse quality-check finalization: %w", err)
	}
	want, err := buildQualityCheckFinalizationWithEvaluation(newQualityCheckEvaluation(root, locale, catalog), snapshotID, func() time.Time {
		t, e := time.Parse(time.RFC3339Nano, got.FinalizedAt)
		if e != nil {
			return time.Time{}
		}
		return t
	}, got.PreviousSnapshotID)
	if err != nil {
		return nil, err
	}
	if got.FinalizedAt == "" || !bytes.Equal(mustJSON(got), mustJSON(*want)) {
		return nil, errors.New("quality-check finalization does not match current A-only Snapshot evidence")
	}
	return &got, nil
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
