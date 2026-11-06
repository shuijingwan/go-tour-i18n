package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const CurrentValidationAuthority = "current-validation"
const CurrentValidationRevisionFeedback = "current_validation"

type CurrentValidationReopenOptions struct {
	ContentPackage                                       string
	Locale, ReopenID, PreviousSnapshotID, SnapshotSHA256 string
	Evidence                                             GlossaryArchiveReference
	AssessmentIdentity                                   string
	UnitIDs                                              []string
}

// SelectCurrentValidationReopenPackage selects native Tour's default complete
// scope or an exact registered package on a copy, never widening explicit scope.
func SelectCurrentValidationReopenPackage(c *Catalog, contentPackage string) (*Catalog, error) {
	if c == nil || contentPackage == "" {
		return nil, fmt.Errorf("explicit content-package required")
	}
	scoped := *c
	if contentPackage == tourNativeTranslationPackage {
		if len(c.TranslationScope) > 0 {
			return nil, fmt.Errorf("canonical Tour-native default complete scope required; explicit scope cannot be widened")
		}
	} else if err := scoped.UseTranslationPackage(contentPackage); err != nil {
		return nil, err
	}
	if len(c.TranslationScope) > 0 && !reflect.DeepEqual(c.TranslationScope, scoped.TranslationScope) {
		return nil, fmt.Errorf("explicit current-validation package scope mismatch")
	}
	pkg, _, _, ordered, err := translationAuthorizationScope(&scoped)
	if err != nil {
		return nil, err
	}
	if pkg != contentPackage || len(ordered) == 0 {
		return nil, fmt.Errorf("current-validation complete package identity/scope mismatch")
	}
	if contentPackage == tourNativeTranslationPackage {
		for _, u := range ordered {
			if u.Adapter != nil {
				return nil, fmt.Errorf("canonical Tour-native scope contains a foreign adapter: %s", u.ID)
			}
		}
	}
	return &scoped, nil
}

type CurrentValidationReopenFinding struct {
	UnitID             string                  `json:"unit_id"`
	ValidationIdentity string                  `json:"current_validation_identity_sha256"`
	Reason             string                  `json:"validator_finding"`
	CommentSlots       []CommentSlotComparison `json:"comment_slots,omitempty"`
}

type CurrentValidationReopenAuthority struct {
	Package                string                           `json:"content_package"`
	Snapshot               GlossaryArchiveReference         `json:"previous_snapshot"`
	Evidence               GlossaryArchiveReference         `json:"reattestation_evidence"`
	EvidenceJSON           string                           `json:"frozen_reattestation_json"`
	AssessmentIdentity     string                           `json:"reattestation_identity_sha256"`
	ImplementationIdentity string                           `json:"authority_dependency_identity_sha256"`
	Findings               []CurrentValidationReopenFinding `json:"findings"`
	Identity               string                           `json:"receipt_identity_sha256"`
}

var currentValidationReopenDependencies = []string{"internal/i18n/current_validation_reopen.go", "internal/i18n/quality_check_reopen.go", "internal/i18n/retranslation_export.go", "internal/i18n/generation_revision_inputs.go", "internal/i18n/preview_failure.go"}

func currentValidationReopenBytes(r *QualityCheckSurfaceReopen) []byte {
	b, _ := json.MarshalIndent(r, "", "  ")
	return append(b, '\n')
}
func currentValidationReopenIdentity(r *QualityCheckSurfaceReopen) string {
	copy := *r
	a := *r.CurrentValidation
	a.Identity = ""
	copy.CurrentValidation = &a
	return sum(mustJSON(copy))
}

func currentValidationFinding(f CurrentValidationReopenFinding) string {
	return "current-validation: " + string(mustJSON(f)) + ". Historical A remains immutable; current eligibility failed under the canonical validator. Revise only the exact affected teaching-comment bodies/coordinates or other listed machine failure and their locally necessary context. Preserve the complete local meaning, code, identifiers, directives, delimiters, structure and unrelated Page prose. Do not retranslate the whole Page. Current validation and independent re-QC are required."
}

// Read an explicit evidence file without following symlinks. Absolute paths
// are accepted for inspection/import only; the exact bytes are frozen inside
// the shared receipt, never a mutable /tmp dependency of consumed history.
func readCurrentValidationEvidence(root string, ref GlossaryArchiveReference) ([]byte, error) {
	if !validSHA256(ref.SHA256) || ref.Path == "" {
		return nil, fmt.Errorf("exact re-attestation path/SHA required")
	}
	path := ref.Path
	if !filepath.IsAbs(path) {
		return readCompatibilityReference(root, ref)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("regular re-attestation evidence required: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil || sum(b) != ref.SHA256 {
		return nil, fmt.Errorf("re-attestation evidence SHA mismatch: %v", err)
	}
	return b, nil
}

func readCompatibilityReference(root string, ref GlossaryArchiveReference) ([]byte, error) {
	b, err := readCompatibilityFile(root, ref.Path)
	if err != nil || sum(b) != ref.SHA256 {
		return nil, fmt.Errorf("evidence reference stale: %s: %v", ref.Path, err)
	}
	return b, nil
}

func decodeCurrentValidationAssessment(a *CurrentValidationReopenAuthority) (*CurrentTranslationAuthorization, error) {
	if sum([]byte(a.EvidenceJSON)) != a.Evidence.SHA256 {
		return nil, fmt.Errorf("frozen re-attestation evidence hash mismatch")
	}
	var assessment CurrentTranslationAuthorization
	if err := contentidentity.StrictJSON([]byte(a.EvidenceJSON), &assessment); err != nil {
		return nil, err
	}
	if assessment.Identity != a.AssessmentIdentity || assessment.Identity != authorizationIdentity(assessment) || assessment.Schema != CurrentTranslationAuthorizationSchema || assessment.Package != a.Package {
		return nil, fmt.Errorf("re-attestation identity/package mismatch")
	}
	return &assessment, nil
}

// Current selection only: machine validation is expected to fail for the
// affected Units. No historical-success fallback or new candidate is selected.
func requireCurrentValidationBaseline(root string, c *Catalog, locale string, s *QualityCheckSnapshotManifest) error {
	latest, err := selectLatestRetranslationUnits(root, c, locale)
	if err != nil {
		return err
	}
	if len(latest.ordered) != len(s.Units) {
		return fmt.Errorf("current validation baseline exact-set count mismatch")
	}
	for i, u := range latest.ordered {
		old := s.Units[i]
		choice, ok := latest.selectedByID[u.ID]
		if !ok || choice.result.Status != "passed" || u.ID != old.UnitID || u.SourcePath != old.SourcePath || u.SourceSHA256 != old.SourceSHA256 {
			return fmt.Errorf("conflicting current successor/source: %s", u.ID)
		}
		if choice.machine != nil {
			if !qualityCheckSnapshotIdentityMatches(old, choice.machine.snapshot(i+1)) {
				return fmt.Errorf("conflicting machine successor: %s", u.ID)
			}
		} else if !selectedRetranslationIdentityMatches(u, choice) || !choiceMatchesSnapshot(locale, choice, old) {
			return fmt.Errorf("conflicting current candidate: %s", u.ID)
		}
	}
	return nil
}

func expectedCurrentValidationReopen(root string, c *Catalog, o CurrentValidationReopenOptions) (*QualityCheckSurfaceReopen, error) {
	if len(o.UnitIDs) > MaxRetranslationExportLimit {
		return nil, fmt.Errorf("current-validation scope must not exceed %d TranslationUnits", MaxRetranslationExportLimit)
	}
	if ValidateLocaleName(o.Locale) != nil || !reviewIDPattern.MatchString(o.ReopenID) || validateSnapshotID(o.PreviousSnapshotID) != nil || !validSHA256(o.SnapshotSHA256) || !validSHA256(o.AssessmentIdentity) {
		return nil, fmt.Errorf("explicit current-validation identities required")
	}
	c, err := SelectCurrentValidationReopenPackage(c, o.ContentPackage)
	if err != nil {
		return nil, err
	}
	pkg, _, _, ordered, err := translationAuthorizationScope(c)
	if err != nil {
		return nil, err
	}
	ref := GlossaryArchiveReference{Path: "data/quality-check-snapshots/" + o.Locale + "/" + o.PreviousSnapshotID + "/manifest.json", SHA256: o.SnapshotSHA256}
	if _, err := readCompatibilityReference(root, ref); err != nil {
		return nil, err
	}
	s, err := readQualityCheckSnapshot(root, o.Locale, o.PreviousSnapshotID, false)
	if err != nil {
		return nil, err
	}
	if err := requireCurrentValidationBaseline(root, c, o.Locale, s); err != nil {
		return nil, err
	}
	raw, err := readCurrentValidationEvidence(root, o.Evidence)
	if err != nil {
		return nil, err
	}
	a := &CurrentValidationReopenAuthority{Package: pkg, Snapshot: ref, Evidence: o.Evidence, EvidenceJSON: string(raw), AssessmentIdentity: o.AssessmentIdentity}
	frozen, err := decodeCurrentValidationAssessment(a)
	if err != nil {
		return nil, err
	}
	if frozen.Locale != o.Locale || frozen.Historical.Finalization.SnapshotID != o.PreviousSnapshotID {
		return nil, fmt.Errorf("re-attestation predecessor/locale mismatch")
	}
	current, err := AssessCurrentTranslation(root, c, o.Locale, o.PreviousSnapshotID)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(frozen, current) {
		return nil, fmt.Errorf("current re-attestation changed; refusing stale evidence")
	}
	if current.Stale != 0 || current.Unknown != 0 || current.Affected == 0 || current.Compatible+current.Affected != len(ordered) || len(current.Units) != len(ordered) {
		return nil, fmt.Errorf("complete current compatible/affected set without stale/unknown required")
	}
	a.ImplementationIdentity, err = TranslationMachineAuthority(root, currentValidationReopenDependencies)
	if err != nil {
		return nil, err
	}
	r := &QualityCheckSurfaceReopen{SchemaVersion: qualityCheckSurfaceReopenSchemaVersion, EvidenceType: qualityCheckSurfaceReopenEvidenceType, AuthoritySource: CurrentValidationAuthority, Locale: o.Locale, ReopenID: o.ReopenID, PreviousSnapshotID: o.PreviousSnapshotID, CurrentValidation: a, Units: []QualityCheckSurfaceReopenUnit{}}
	finalRef, err := successorReference(root, "data/quality-check-snapshots/"+o.Locale+"/"+o.PreviousSnapshotID+"/finalization.json")
	if err != nil {
		return nil, err
	}
	r.PreviousFinalizationPath, r.PreviousFinalizationSHA256 = finalRef.Path, finalRef.SHA256
	exported, _, err := scanRetranslationBatches(filepath.Join(root, "data/retranslation-runs", o.Locale), o.Locale, c)
	if err != nil {
		return nil, err
	}
	g, err := LoadGlossary(root, o.Locale)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for i, x := range current.Units {
		if x.Classification == "compatible" {
			if x.Validation != "passed" {
				return nil, fmt.Errorf("compatible Unit lacks current validation")
			}
			continue
		}
		if x.Classification != "affected" || x.Validation != "failed" {
			return nil, fmt.Errorf("affected Unit lacks current machine validation failure")
		}
		u := ordered[i]
		old := s.Units[i]
		accepted := current.Historical.Finalization.Units[i]
		if accepted.UnitID != u.ID || accepted.Rating != "A" || !qualityCheckSnapshotIdentityMatches(old, accepted.Snapshot) || !qualityCheckSnapshotIdentityMatches(old, x.Snapshot) {
			return nil, fmt.Errorf("historical A/source exact binding mismatch")
		}
		if e := exported[u.ID]; e.BatchID != "" && e.BatchID != old.SelectedBatchID {
			return nil, fmt.Errorf("Unit already exported to conflicting successor: %s", u.ID)
		}
		candidate, err := readCompatibilityReference(root, GlossaryArchiveReference{Path: old.CandidatePath, SHA256: old.CandidateSHA256})
		if err != nil {
			return nil, err
		}
		validationErr := validateTranslationUnitWithGlossary(root, c, u, o.Locale, candidate, g)
		if validationErr == nil {
			return nil, fmt.Errorf("current machine failure no longer reproducible")
		}
		matched := false
		for _, reason := range x.Reasons {
			matched = matched || reason == "current_validation_failed: "+validationErr.Error()
		}
		if !matched {
			return nil, fmt.Errorf("per-Unit current-validation finding mismatch")
		}
		finding := CurrentValidationReopenFinding{UnitID: u.ID, ValidationIdentity: x.ValidationIdentity, Reason: validationErr.Error()}
		var gate *CommentGateError
		if errors.As(validationErr, &gate) {
			rows, proofErr := ValidateTranslationUnitCommentCoverage(root, c, u, o.Locale, candidate)
			if proofErr == nil || proofErr.Error() != validationErr.Error() {
				return nil, fmt.Errorf("comment finding proof changed")
			}
			for _, row := range rows {
				for _, id := range gate.SlotIDs {
					if row.SlotID == id {
						finding.CommentSlots = append(finding.CommentSlots, row)
					}
				}
			}
			if len(finding.CommentSlots) != len(gate.SlotIDs) {
				return nil, fmt.Errorf("incomplete exact comment failure scope")
			}
		}
		a.Findings = append(a.Findings, finding)
		r.Units = append(r.Units, QualityCheckSurfaceReopenUnit{Index: old.Index, UnitID: u.ID, PreviousRating: "A", Finding: currentValidationFinding(finding), Snapshot: old})
		ids = append(ids, u.ID)
	}
	if len(ids) != current.Affected || len(ids) > MaxRetranslationExportLimit {
		return nil, fmt.Errorf("complete affected scope must contain 1..%d Units", MaxRetranslationExportLimit)
	}
	if err := c.RequireTranslationInvocation(ids); err != nil {
		return nil, err
	}
	if len(o.UnitIDs) > 0 && !reflect.DeepEqual(o.UnitIDs, ids) {
		return nil, fmt.Errorf("requested Unit set/order must exactly match current affected scope")
	}
	r.UnitCount = len(r.Units)
	a.Identity = currentValidationReopenIdentity(r)
	return r, nil
}

// Shared receipt writer; default inspection is mutation-free. Record is only
// reached through the existing explicit RecordQualityCheckSurfaceReopen API.
func prepareCurrentValidationReopen(root string, c *Catalog, o CurrentValidationReopenOptions, record bool) (*QualityCheckSurfaceReopen, string, error) {
	if ValidateLocaleName(o.Locale) != nil || !reviewIDPattern.MatchString(o.ReopenID) {
		return nil, "", fmt.Errorf("safe locale/reopen ID required")
	}
	path := qualityCheckSurfaceReopenPath(root, o.Locale, o.ReopenID)
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("reopen ID already exists or cannot be inspected")
	}
	r, err := expectedCurrentValidationReopen(root, c, o)
	if err != nil {
		return nil, "", err
	}
	relative, err := repositoryRelativePath(root, path)
	if err != nil {
		return nil, "", err
	}
	if record {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, "", err
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return nil, "", err
		}
		_, writeErr := f.Write(currentValidationReopenBytes(r))
		closeErr := f.Close()
		if writeErr != nil {
			return nil, "", writeErr
		}
		if closeErr != nil {
			return nil, "", closeErr
		}
	}
	return r, relative, nil
}

func InspectCurrentValidationReopen(root string, c *Catalog, o CurrentValidationReopenOptions) (*QualityCheckSurfaceReopen, string, error) {
	return prepareCurrentValidationReopen(root, c, o, false)
}

func verifyCurrentValidationReopen(root string, c *Catalog, r *QualityCheckSurfaceReopen, current bool) error {
	a := r.CurrentValidation
	if a == nil || r.AuthoritySource != CurrentValidationAuthority || r.PreviewFailure != nil || r.ReplacementContext != nil || r.IntegratedAuthority != nil || r.SurfaceReviewID != "" || r.SurfaceReviewPath != "" || r.SurfaceReviewSHA256 != "" || r.Finding != "" || len(r.Findings) != 0 || r.UnitCount > MaxRetranslationExportLimit || a.Identity != currentValidationReopenIdentity(r) {
		return fmt.Errorf("exclusive exact current-validation receipt required")
	}
	x, err := decodeCurrentValidationAssessment(a)
	if err != nil {
		return err
	}
	if x.Locale != r.Locale || x.Historical.Finalization.Locale != r.Locale || x.Historical.Finalization.SnapshotID != r.PreviousSnapshotID || x.Stale != 0 || x.Unknown != 0 || x.Affected != len(r.Units) || x.Compatible+x.Affected != len(x.Units) || len(a.Findings) != len(r.Units) || len(x.Historical.Finalization.Units) != len(x.Units) || len(x.Historical.ReviewerGlossaries) != len(x.Units) {
		return fmt.Errorf("current-validation receipt exact assessment set mismatch")
	}
	if err := checkFlatHistoricalIntegrity(root, c, &x.Historical); err != nil {
		return err
	}
	if _, err := readCompatibilityReference(root, a.Snapshot); err != nil {
		return err
	}
	if a.Snapshot.Path != "data/quality-check-snapshots/"+r.Locale+"/"+r.PreviousSnapshotID+"/manifest.json" || r.PreviousFinalizationPath != "data/quality-check-snapshots/"+r.Locale+"/"+r.PreviousSnapshotID+"/finalization.json" {
		return fmt.Errorf("current-validation predecessor path mismatch")
	}
	if _, err := readCompatibilityReference(root, GlossaryArchiveReference{Path: r.PreviousFinalizationPath, SHA256: r.PreviousFinalizationSHA256}); err != nil {
		return err
	}
	index := 0
	var kind UnitKind
	for i, u := range x.Units {
		if u.Classification == "compatible" {
			if u.Validation != "passed" {
				return fmt.Errorf("invalid compatible assessment")
			}
			continue
		}
		if index >= len(r.Units) {
			return fmt.Errorf("affected scope incomplete")
		}
		unit, f := r.Units[index], a.Findings[index]
		old := x.Historical.Finalization.Units[i]
		if u.Classification != "affected" || u.Validation != "failed" || old.Rating != "A" || old.UnitID != unit.UnitID || unit.PreviousRating != "A" || unit.Replacement != nil || unit.Index != u.Snapshot.Index || !qualityCheckSnapshotIdentityMatches(unit.Snapshot, u.Snapshot) || !qualityCheckSnapshotIdentityMatches(unit.Snapshot, old.Snapshot) || f.UnitID != unit.UnitID || f.ValidationIdentity != u.ValidationIdentity || unit.Finding != currentValidationFinding(f) {
			return fmt.Errorf("current-validation per-Unit A/finding identity mismatch")
		}
		found := false
		for _, reason := range u.Reasons {
			found = found || reason == "current_validation_failed: "+f.Reason
		}
		if !found || f.Reason == "" {
			return fmt.Errorf("current-validation reason missing")
		}
		if kind != "" && kind != unit.Snapshot.UnitKind {
			return fmt.Errorf("mixed Page/Example current-validation receipt")
		}
		kind = unit.Snapshot.UnitKind
		index++
	}
	if index != len(r.Units) {
		return fmt.Errorf("current-validation scope mismatch")
	}
	if current {
		o := CurrentValidationReopenOptions{ContentPackage: a.Package, Locale: r.Locale, ReopenID: r.ReopenID, PreviousSnapshotID: r.PreviousSnapshotID, SnapshotSHA256: a.Snapshot.SHA256, Evidence: a.Evidence, AssessmentIdentity: a.AssessmentIdentity}
		expected, err := expectedCurrentValidationReopen(root, c, o)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(expected, r) {
			return fmt.Errorf("current-validation authority no longer current")
		}
	}
	return nil
}

func currentValidationPredecessorEvaluation(root string, c *Catalog, locale string) *qualityCheckEvaluation {
	e := newQualityCheckEvaluation(root, locale, c)
	e.originalOnly = true
	return e
}
