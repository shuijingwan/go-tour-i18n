package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// This is the existing current-validation successor domain, not a Generation
// decoder. The reopen reader verifies its sealed historical authorization; no
// historical Generation inputs or installed transport are regenerated here.
func currentValidationCandidateLineage(root string, c *Catalog, m *RetranslationBatchManifest) ([]GlossaryArchiveReference, error) {
	if m.TransportRecovery != nil || len(m.Units) == 0 {
		return nil, fmt.Errorf("current candidate binding requires ordinary current-validation successor")
	}
	r, err := readCurrentQualityCheckSurfaceReopen(root, c, m.Locale, m.Units[0].RevisionAuthorizationID)
	if err != nil {
		return nil, err
	}
	if r.AuthoritySource != CurrentValidationAuthority || r.CurrentValidation == nil || r.UnitCount != len(m.Units) {
		return nil, fmt.Errorf("current candidate successor authority/scope mismatch")
	}
	pkg, _, _, _, err := translationAuthorizationScope(c)
	if err != nil || pkg != r.CurrentValidation.Package {
		return nil, fmt.Errorf("current candidate successor package mismatch: %v", err)
	}
	if _, err := SelectCurrentValidationReopenPackage(c, pkg); err != nil {
		return nil, err
	}
	ref := r.revisionEvidence()
	for i, record := range m.Units {
		old := r.Units[i].Snapshot
		if record.RepresentationBridge != nil || record.RevisionFeedbackSource != CurrentValidationRevisionFeedback || record.UnitID != r.Units[i].UnitID || record.UnitKind != old.UnitKind || record.SourcePath != old.SourcePath || record.SourceSHA256 != old.SourceSHA256 || record.PreviousSnapshotID != r.PreviousSnapshotID || record.RevisionAuthorizationID != r.ReopenID || record.PreviousRating != "A" || record.PreviousFinding != r.findingForUnit(record.UnitID) || record.PreviousReviewPath != ref.Path || record.PreviousReviewSHA256 != ref.SHA256 {
			return nil, fmt.Errorf("current candidate successor exact finding/lineage mismatch: %s", record.UnitID)
		}
		u, err := c.Unit(record.UnitID)
		if err != nil || u.Kind != record.UnitKind || u.SourcePath != record.SourcePath || u.SourceSHA256 != record.SourceSHA256 || sum(u.Source) != record.SourceSHA256 {
			return nil, fmt.Errorf("current candidate successor source/scope mismatch: %s", record.UnitID)
		}
		if record.InputPath != "inputs/"+retranslationUnitInputName(u) || !validSHA256(record.InputSHA256) {
			return nil, fmt.Errorf("current candidate successor input identity mismatch")
		}
		source, err := readTranslationUnitSource(root, u)
		if err != nil || validateSnapshotSourceIdentity(c, u, source) != nil {
			return nil, fmt.Errorf("current candidate successor source changed: %s", record.UnitID)
		}
	}
	x, err := decodeCurrentValidationAssessment(r.CurrentValidation)
	if err != nil {
		return nil, err
	}
	refs := append([]GlossaryArchiveReference{}, x.Historical.References...)
	refs = append(refs, r.CurrentValidation.Snapshot, GlossaryArchiveReference{Path: r.PreviousFinalizationPath, SHA256: r.PreviousFinalizationSHA256})
	reopen, err := successorReference(root, "data/quality-check-reopens/"+m.Locale+"/"+r.ReopenID+".json")
	if err != nil {
		return nil, err
	}
	return append(refs, reopen), nil
}

// nil means legacy/unbound evidence: the caller must retain its original
// strict restoration path. A malformed or mismatching binding is never a
// reason to fall back. This helper grants no Reviewer rating or A carry.
func snapshotBoundCurrentValidationCandidate(root string, c *Catalog, locale string, u *TranslationUnit, choice selectedRetranslationUnit) (*validatedRetranslationCandidate, []GlossaryArchiveReference, error) {
	if choice.machine != nil || choice.manifest.RevisionFeedbackSource != CurrentValidationRevisionFeedback || choice.manifest.RepresentationBridge != nil {
		return nil, nil, nil
	}
	m, err := readRetranslationProcessManifest(choice.batchDir, locale, choice.batchID)
	if err != nil {
		return nil, nil, err
	}
	if m.TransportRecovery != nil {
		return nil, nil, nil
	}
	v, err := readPromotionValidation(choice.batchDir, choice.batchID, locale, choice.manifest, choice.result)
	if err != nil {
		return nil, nil, err
	}
	if !selectedRetranslationIdentityMatches(u, choice) || choice.result.Status != "passed" || choice.result.Error != "" || v.Error != "" {
		return nil, nil, fmt.Errorf("current candidate binding source/result mismatch")
	}
	if v.CandidateSHA256 == "" {
		return nil, nil, nil
	}
	refs, err := currentValidationCandidateLineage(root, c, m)
	if err != nil {
		return nil, nil, err
	}
	result, err := readPromotionResult(choice.batchDir, locale, choice.batchID, m.UnitCount)
	if err != nil {
		return nil, nil, err
	}
	if err := validateCurrentValidationCandidateResult(m, result); err != nil {
		return nil, nil, err
	}
	found := false
	for i, record := range m.Units {
		if record.UnitID == u.ID {
			found = reflect.DeepEqual(record, choice.manifest) && reflect.DeepEqual(result.Units[i], choice.result)
		}
	}
	if !found {
		return nil, nil, fmt.Errorf("current candidate binding selected Unit mismatch")
	}
	name := filepath.Base(choice.manifest.InputPath)
	ext := filepath.Ext(name)
	flat := strings.TrimSuffix(name, ext)
	if choice.result.CandidatePath != "candidates/"+retranslationUnitCandidateName(u) || choice.result.ValidationPath != "validation/"+flat+".json" {
		return nil, nil, fmt.Errorf("current candidate binding paths mismatch")
	}
	attempt, err := retryValidationAttemptForExtension(v.RawResponsePath, flat, ext)
	if err != nil || attempt != v.Attempt {
		return nil, nil, fmt.Errorf("current candidate binding attempt mismatch")
	}
	retryDir := filepath.Join(choice.batchDir, "retries", flat)
	if attempt == 1 {
		entries, err := os.ReadDir(retryDir)
		if (err != nil && !os.IsNotExist(err)) || len(entries) != 0 {
			return nil, nil, fmt.Errorf("current candidate binding attempt-001 has retry history")
		}
	} else if err := validateRetryAttemptSequenceForExtension(retryDir, attempt, attempt, ext); err != nil {
		return nil, nil, err
	}
	base, err := repositoryRelativePath(root, choice.batchDir)
	if err != nil {
		return nil, nil, err
	}
	candidate, err := readSnapshotReferencedFile(root, base+"/"+v.CandidatePath)
	if err != nil || sum(candidate) != v.CandidateSHA256 {
		return nil, nil, fmt.Errorf("current validated candidate SHA mismatch")
	}
	if err := ValidateTranslationUnitCandidate(root, c, u.ID, locale, candidate); err != nil {
		return nil, nil, err
	}
	data, err := readSnapshotReferencedFile(root, base+"/"+choice.result.ValidationPath)
	if err != nil {
		return nil, nil, err
	}
	again, err := decodeRetranslationValidation(data, u)
	if err != nil || !reflect.DeepEqual(v, again) {
		return nil, nil, fmt.Errorf("current candidate validation changed during verification")
	}
	for _, path := range []string{base + "/manifest.json", base + "/result.json", base + "/" + choice.manifest.InputPath, base + "/" + v.CandidatePath, base + "/" + choice.result.ValidationPath, base + "/" + v.RawResponsePath} {
		ref, err := successorReference(root, path)
		if err != nil {
			return nil, nil, err
		}
		if path == base+"/"+choice.manifest.InputPath && ref.SHA256 != choice.manifest.InputSHA256 {
			return nil, nil, fmt.Errorf("current candidate saved input hash mismatch")
		}
		refs = append(refs, ref)
	}
	// Stable refs form the existing Snapshot's evidence inventory, not a new
	// receipt/identity family. Its canonical manifest hash includes the binding.
	byPath := map[string]GlossaryArchiveReference{}
	for _, ref := range refs {
		if old, ok := byPath[ref.Path]; ok && old != ref {
			return nil, nil, fmt.Errorf("current candidate evidence reference conflict")
		}
		byPath[ref.Path] = ref
	}
	refs = refs[:0]
	for _, ref := range byPath {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	return &validatedRetranslationCandidate{candidatePath: v.CandidatePath, validationPath: choice.result.ValidationPath, candidate: candidate, validationData: data, validation: v, attempt: attempt}, refs, nil
}

func validateCurrentValidationCandidateResult(m *RetranslationBatchManifest, result *RetranslationProcessResult) error {
	if result.NoPendingBatches || result.UnitCount != len(m.Units) || len(result.Units) != len(m.Units) || result.RestorePassed+result.RestoreFailed != m.UnitCount || result.ValidationPassed+result.ValidationFailed != result.RestorePassed {
		return fmt.Errorf("current candidate binding result counts mismatch")
	}
	copy := *result
	recountRetranslationResult(&copy)
	if !reflect.DeepEqual(copy, *result) {
		return fmt.Errorf("current candidate binding result status/count mismatch")
	}
	seen := map[string]bool{}
	for i, record := range m.Units {
		got := result.Units[i]
		if seen[record.UnitID] || record.UnitID != got.UnitID || record.UnitKind != got.UnitKind {
			return fmt.Errorf("current candidate binding manifest/result exact set mismatch")
		}
		seen[record.UnitID] = true
	}
	return nil
}
