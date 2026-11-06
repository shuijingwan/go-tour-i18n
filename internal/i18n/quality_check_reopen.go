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
	"regexp"
	"sort"
	"strings"
)

const qualityCheckSurfaceReopenSchemaVersion = 1
const qualityCheckSurfaceReopenEvidenceType = "quality_check_surface_reopen"

type QualityCheckSurfaceReopenUnit struct {
	PreviousRating string                      `json:"previous_rating,omitempty"`
	Replacement    *TranslationUnitReplacement `json:"replacement,omitempty"`
	Index          int                         `json:"index"`
	UnitID         string                      `json:"unit_id"`
	Finding        string                      `json:"finding,omitempty"`
	Snapshot       QualityCheckSnapshotUnit    `json:"snapshot"`
}

type QualityCheckSurfaceReopen struct {
	SameUnitBridge             *SameUnitBridgeAuthority           `json:"same_unit_bridge,omitempty"`
	CurrentValidation          *CurrentValidationReopenAuthority  `json:"current_validation,omitempty"`
	AuthoritySource            string                             `json:"authority_source,omitempty"`
	PreviewFailure             *GlossaryArchiveReference          `json:"preview_failure,omitempty"`
	ReplacementContext         *TranslationUnitReplacementContext `json:"replacement_context,omitempty"`
	SchemaVersion              int                                `json:"schema_version"`
	EvidenceType               string                             `json:"evidence_type"`
	Locale                     string                             `json:"locale"`
	ReopenID                   string                             `json:"reopen_id"`
	PreviousSnapshotID         string                             `json:"previous_snapshot_id"`
	PreviousFinalizationPath   string                             `json:"previous_finalization_path"`
	PreviousFinalizationSHA256 string                             `json:"previous_finalization_sha256"`
	SurfaceReviewID            string                             `json:"surface_review_id"`
	SurfaceReviewPath          string                             `json:"surface_review_path"`
	SurfaceReviewSHA256        string                             `json:"surface_review_sha256"`
	Finding                    string                             `json:"finding"`
	Findings                   []string                           `json:"findings,omitempty"`
	UnitCount                  int                                `json:"unit_count"`
	Units                      []QualityCheckSurfaceReopenUnit    `json:"units"`
	IntegratedAuthority        *SurfaceReviewAuthority            `json:"integrated_authority,omitempty"`
}

type QualityCheckSurfaceReopenOptions struct {
	CurrentValidation                                              *CurrentValidationReopenOptions
	AuthoritySource                                                string
	PreviousRating                                                 string
	RepresentationProof                                            GlossaryArchiveReference
	PreviewFailure                                                 GlossaryArchiveReference
	PreviousReopenID                                               string
	HistoricalDocuments                                            GlossaryArchiveReference
	Locale, ReopenID, PreviousSnapshotID, SurfaceReviewID, Finding string
	UnitIDs                                                        []string
	Findings                                                       []string
	IntegratedSurface                                              bool
}

func qualityCheckSurfaceReopenPath(root, locale, reopenID string) string {
	return filepath.Join(root, "data", "quality-check-reopens", locale, reopenID+".json")
}

func RecordQualityCheckSurfaceReopen(root string, catalog *Catalog, options QualityCheckSurfaceReopenOptions) (*QualityCheckSurfaceReopen, string, error) {
	if options.AuthoritySource == CurrentValidationAuthority {
		if options.CurrentValidation == nil || options.Locale != options.CurrentValidation.Locale || options.ReopenID != options.CurrentValidation.ReopenID || options.PreviousSnapshotID != options.CurrentValidation.PreviousSnapshotID || len(options.UnitIDs) != 0 || options.Finding != "" || len(options.Findings) != 0 || options.PreviewFailure != (GlossaryArchiveReference{}) || options.HistoricalDocuments != (GlossaryArchiveReference{}) || options.RepresentationProof != (GlossaryArchiveReference{}) || options.PreviousRating != "" || options.SurfaceReviewID != "" || options.IntegratedSurface || options.PreviousReopenID != "" {
			return nil, "", fmt.Errorf("exclusive current-validation authority options required")
		}
		return prepareCurrentValidationReopen(root, catalog, *options.CurrentValidation, true)
	}
	if options.CurrentValidation != nil {
		return nil, "", fmt.Errorf("current-validation evidence requires its authority source")
	}
	if options.AuthoritySource == "quality-check" {
		return recordQualityCheckReplacement(root, catalog, options)
	}
	if options.AuthoritySource != "" {
		return nil, "", fmt.Errorf("unknown reconciliation authority source")
	}
	if options.PreviewFailure.Path != "" {
		return recordPreviewReopen(root, catalog, options)
	}
	if options.PreviousReopenID != "" || options.HistoricalDocuments.Path != "" {
		return recordReplacementSurfaceReopen(root, catalog, options)
	}
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
	// Keep the legacy single-finding wire representation. Multiple exact
	// findings require Integrated authority, which supplies the pairing.
	findings, err := surfaceReopenSelectedFindings(options)
	if err != nil {
		return nil, "", err
	}
	options.Finding, options.Findings = "", nil
	if len(findings) == 1 {
		options.Finding = findings[0]
	} else {
		options.Findings = findings
	}
	finding := options.Finding
	if len(options.UnitIDs) == 0 {
		return nil, "", errors.New("surface correction requires a non-empty finding and exact unit_id scope")
	}
	if len(options.UnitIDs) > MaxRetranslationExportLimit {
		return nil, "", fmt.Errorf("surface correction scope must not exceed %d TranslationUnits", MaxRetranslationExportLimit)
	}
	var authority *SurfaceReviewAuthority
	var unitFindings map[string]string
	var finalization *QualityCheckFinalization
	if options.IntegratedSurface {
		resolved, err := resolveSurfaceReopenAuthority(root, catalog, options, true)
		if err != nil {
			return nil, "", err
		}
		authority, unitFindings = resolved.authority, resolved.unitFindings
		catalog, finalization = resolved.catalog, &resolved.history.Finalization
	} else {
		finalization, err = VerifyQualityCheckFinalization(root, catalog, options.Locale, options.PreviousSnapshotID)
		if err != nil {
			return nil, "", fmt.Errorf("surface correction requires a current finalized predecessor: %w", err)
		}
	}
	finalizationPath := qualityCheckFinalizationPath(root, options.Locale, options.PreviousSnapshotID)
	finalizationData, err := os.ReadFile(finalizationPath)
	if err != nil {
		return nil, "", err
	}
	surfaceRelative := filepath.ToSlash(filepath.Join("data", "locale-surface-reviews", options.Locale, options.SurfaceReviewID+".md"))
	if authority != nil {
		surfaceRelative = authority.Receipt.Path
	}
	surfacePath := filepath.Join(root, filepath.FromSlash(surfaceRelative))
	surfaceData, err := os.ReadFile(surfacePath)
	if err != nil {
		return nil, "", fmt.Errorf("read formal Surface Review finding: %w", err)
	}
	if authority == nil && !bytes.Contains(surfaceData, []byte(finding)) {
		return nil, "", errors.New("formal Surface Review evidence does not contain the exact finding")
	}
	snapshot, err := readQualityCheckSnapshot(root, options.Locale, options.PreviousSnapshotID, authority == nil, catalog)
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
		if authority == nil && !surfaceReviewNamesUnit(surfaceData, unitID) {
			return nil, "", fmt.Errorf("formal Surface Review finding does not name unit_id %s", unitID)
		}
		if kind == "" {
			kind = unit.UnitKind
		} else if kind != unit.UnitKind {
			return nil, "", errors.New("surface correction scope must not mix Page and Example TranslationUnits")
		}
		if authority == nil {
			if _, err := readSnapshotUnitRepositoryEvidence(root, catalog, options.Locale, unit); err != nil {
				return nil, "", fmt.Errorf("surface correction unit %s identity: %w", unitID, err)
			}
		}
		seen[unitID] = true
		reopened := QualityCheckSurfaceReopenUnit{Index: unit.Index, UnitID: unit.UnitID, Snapshot: unit}
		if len(options.Findings) > 0 {
			reopened.Finding = unitFindings[unitID]
		}
		units = append(units, reopened)
	}
	sort.Slice(units, func(i, j int) bool { return units[i].Index < units[j].Index })
	receipt := &QualityCheckSurfaceReopen{
		SchemaVersion: qualityCheckSurfaceReopenSchemaVersion, EvidenceType: qualityCheckSurfaceReopenEvidenceType,
		Locale: options.Locale, ReopenID: options.ReopenID, PreviousSnapshotID: options.PreviousSnapshotID,
		PreviousFinalizationPath:   filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", options.Locale, options.PreviousSnapshotID, "finalization.json")),
		PreviousFinalizationSHA256: sum(finalizationData),
		SurfaceReviewID:            options.SurfaceReviewID,
		SurfaceReviewPath:          surfaceRelative,
		SurfaceReviewSHA256:        sum(surfaceData), Finding: finding, UnitCount: len(units), Units: units,
		IntegratedAuthority: authority,
		Findings:            options.Findings,
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

// CheckQualityCheckSurfaceReopen performs the same read-only authority and
// unconsumed-output preflight used by revision export. It never records a reopen
// or creates a batch, and supports each existing reopen authority family.
func CheckQualityCheckSurfaceReopen(root string, catalog *Catalog, locale, reopenID string) (*QualityCheckSurfaceReopen, error) {
	return readCurrentQualityCheckSurfaceReopen(root, catalog, locale, reopenID, true)
}

func readCurrentQualityCheckSurfaceReopen(root string, catalog *Catalog, locale, reopenID string, requireUnconsumedCurrent ...bool) (*QualityCheckSurfaceReopen, error) {
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
		receipt.Locale != locale || receipt.ReopenID != reopenID || receipt.UnitCount < 1 || receipt.UnitCount != len(receipt.Units) {
		return nil, errors.New("quality-check surface reopen has incompatible identity")
	}
	if receipt.SameUnitBridge != nil || receipt.AuthoritySource == SameUnitBridgeAuthoritySource {
		r, _, err := readSameUnitBridgeForInvocation(root, catalog, locale, GlossaryArchiveReference{Path: filepath.ToSlash(filepath.Join("data/quality-check-reopens", locale, reopenID+".json")), SHA256: sum(data)})
		return r, err
	}
	if receipt.CurrentValidation != nil || receipt.AuthoritySource == CurrentValidationAuthority {
		current := len(requireUnconsumedCurrent) > 0 && requireUnconsumedCurrent[0]
		if err := verifyCurrentValidationReopen(root, catalog, &receipt, current); err != nil {
			return nil, err
		}
		if !bytes.Equal(data, currentValidationReopenBytes(&receipt)) {
			return nil, fmt.Errorf("current-validation reopen encoding changed")
		}
		return &receipt, nil
	}
	if receipt.ReplacementContext != nil {
		if receipt.PreviewFailure != nil {
			return nil, errors.New("mixed preview/replacement authority")
		}
		if err := verifyReplacementSurfaceReopen(root, catalog, &receipt); err != nil {
			return nil, err
		}
		if len(requireUnconsumedCurrent) > 0 && requireUnconsumedCurrent[0] {
			if err := replacementAuthorizationUnconsumed(root, catalog, &receipt); err != nil {
				return nil, err
			}
		}
		return &receipt, nil
	}
	if receipt.AuthoritySource != "" {
		return nil, errors.New("reconciliation authority lacks replacement context")
	}
	if receipt.PreviewFailure != nil {
		current := len(requireUnconsumedCurrent) > 0 && requireUnconsumedCurrent[0]
		if err := verifyPreviewReopen(root, catalog, &receipt, current); err != nil {
			return nil, err
		}
		return &receipt, nil
	}
	for _, u := range receipt.Units {
		if u.Replacement != nil {
			return nil, errors.New("replacement lacks successor context")
		}
	}
	options := QualityCheckSurfaceReopenOptions{Locale: locale, PreviousSnapshotID: receipt.PreviousSnapshotID, SurfaceReviewID: receipt.SurfaceReviewID, Finding: receipt.Finding, Findings: receipt.Findings, IntegratedSurface: receipt.IntegratedAuthority != nil}
	if _, err := surfaceReopenSelectedFindings(options); err != nil {
		return nil, err
	}
	expectedFinalizationPath := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, receipt.PreviousSnapshotID, "finalization.json"))
	expectedSurfacePath := filepath.ToSlash(filepath.Join("data", "locale-surface-reviews", locale, receipt.SurfaceReviewID+".md"))
	if receipt.IntegratedAuthority != nil {
		expectedSurfacePath = filepath.ToSlash(filepath.Join("data", "locale-language", locale, "surface", receipt.SurfaceReviewID+".json"))
	}
	if !reviewIDPattern.MatchString(receipt.SurfaceReviewID) || receipt.PreviousFinalizationPath != expectedFinalizationPath || receipt.SurfaceReviewPath != expectedSurfacePath || receipt.UnitCount > MaxRetranslationExportLimit {
		return nil, errors.New("quality-check surface reopen has unsafe evidence paths or scope")
	}
	var resolved *surfaceReopenResolution
	if receipt.IntegratedAuthority != nil {
		for _, u := range receipt.Units {
			options.UnitIDs = append(options.UnitIDs, u.UnitID)
		}
		current := len(requireUnconsumedCurrent) > 0 && requireUnconsumedCurrent[0]
		resolved, err = resolveSurfaceReopenAuthority(root, catalog, options, current)
		if err != nil {
			return nil, err
		}
		catalog = resolved.catalog
	} else if _, err := VerifyQualityCheckFinalization(root, catalog, locale, receipt.PreviousSnapshotID); err != nil {
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
	if strings.TrimSpace(receipt.Finding) != receipt.Finding || (receipt.IntegratedAuthority == nil && !bytes.Contains(surfaceData, []byte(receipt.Finding))) {
		return nil, errors.New("quality-check surface reopen finding is stale")
	}
	snapshot, err := readQualityCheckSnapshot(root, locale, receipt.PreviousSnapshotID, receipt.IntegratedAuthority == nil, catalog)
	if err != nil {
		return nil, err
	}
	last := 0
	var kind UnitKind
	for _, reopened := range receipt.Units {
		if reopened.Index <= last || reopened.Index < 1 || reopened.Index > len(snapshot.Units) ||
			reopened.UnitID != snapshot.Units[reopened.Index-1].UnitID ||
			!qualityCheckSnapshotIdentityMatches(reopened.Snapshot, snapshot.Units[reopened.Index-1]) ||
			(receipt.IntegratedAuthority == nil && !surfaceReviewNamesUnit(surfaceData, reopened.UnitID)) {
			return nil, errors.New("quality-check surface reopen exact TranslationUnit scope is stale")
		}
		if kind == "" {
			kind = reopened.Snapshot.UnitKind
		} else if kind != reopened.Snapshot.UnitKind {
			return nil, errors.New("surface correction scope must not mix Page and Example TranslationUnits")
		}
		if len(receipt.Findings) == 0 && reopened.Finding != "" && reopened.Finding != receipt.Finding {
			return nil, errors.New("quality-check surface reopen per-unit finding mismatch")
		}
		last = reopened.Index
	}
	if receipt.IntegratedAuthority != nil {
		if !reflect.DeepEqual(resolved.authority, receipt.IntegratedAuthority) {
			return nil, errors.New("Integrated Surface reopen authority changed")
		}
		for _, u := range receipt.Units {
			if receipt.findingForUnit(u.UnitID) != resolved.unitFindings[u.UnitID] {
				return nil, errors.New("Integrated Surface reopen per-unit finding changed or missing")
			}
		}
	}
	return &receipt, nil
}

// Old schema-v1 receipts inherit the single global finding; aggregate receipts
// must supply an exact per-unit finding and never fall back to another Unit.
func (r *QualityCheckSurfaceReopen) findingForUnit(id string) string {
	for _, u := range r.Units {
		if u.UnitID == id {
			if len(r.Findings) > 0 || u.Finding != "" {
				return u.Finding
			}
			return r.Finding
		}
	}
	return ""
}

func surfaceReopenSelectedFindings(o QualityCheckSurfaceReopenOptions) ([]string, error) {
	findings := append([]string(nil), o.Findings...)
	if len(findings) == 0 {
		finding := o.Finding
		if !o.IntegratedSurface {
			finding = strings.TrimSpace(finding)
		}
		findings = []string{finding}
	} else if o.Finding != "" {
		return nil, errors.New("surface correction cannot combine Finding and Findings")
	}
	seen := map[string]bool{}
	for _, f := range findings {
		if f == "" || strings.TrimSpace(f) != f || seen[f] {
			return nil, errors.New("surface correction requires non-empty trimmed unique exact findings")
		}
		seen[f] = true
	}
	if len(findings) > 1 && !o.IntegratedSurface {
		return nil, errors.New("multiple Surface findings require Integrated Surface authority")
	}
	sort.Strings(findings)
	return findings, nil
}
