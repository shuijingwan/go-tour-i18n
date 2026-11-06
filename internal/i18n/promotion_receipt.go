package i18n

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Native receipt is prepared against exact pre-apply canonical bytes. It is
// staged before installation and published exclusively after canonical AND
// status installation. A publication failure rolls back both installations.
func prepareNativePromotionReceipt(root string, catalog *Catalog, options RetranslationPromoteOptions, prepared []preparedPromotion) (*TargetSuccessorEvidence, error) {
	if err := requireMissingNativePromotionReceipt(root, options.Locale, options.SnapshotID); err != nil {
		return nil, err
	}
	path := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", options.Locale, options.SnapshotID, "finalization.json"))
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return nil, err
	}
	var f QualityCheckFinalization
	if err := contentidentity.StrictJSON(b, &f); err != nil {
		return nil, err
	}
	if len(f.Units) != len(prepared) {
		return nil, fmt.Errorf("native receipt membership mismatch")
	}
	e := &TargetSuccessorEvidence{Schema: NativePromotionReceiptSchema, Kind: "native-promotion-receipt", Reason: "formal-promotion-apply", Locale: options.Locale, SnapshotID: options.SnapshotID, Finalization: GlossaryArchiveReference{path, sum(b)}, GlossarySHA: f.GlossarySHA256, QCResults: f.QCResults, Membership: []string{}, Units: []TargetSuccessorUnit{}}
	e.Snapshot, err = successorReference(root, f.SnapshotManifestPath)
	if err != nil {
		return nil, err
	}
	previous, err := successorReference(root, "locales/"+options.Locale+"/status.tsv")
	if err != nil {
		return nil, err
	}
	e.PreviousStatusSHA = previous.SHA256
	e.ProofInputs, err = successorProofInputs(root, catalog, options.Locale, &f)
	if err != nil {
		return nil, err
	}
	byID := map[string]preparedPromotion{}
	for _, item := range prepared {
		byID[item.plan.UnitID] = item
	}
	for _, fu := range f.Units {
		u := fu.Snapshot
		item, ok := byID[u.UnitID]
		if !ok || fu.Rating != "A" || item.plan.SourceCandidateSHA256 != u.CandidateSHA256 || item.plan.BatchID != u.SelectedBatchID {
			return nil, fmt.Errorf("native receipt finalized candidate mismatch %s", u.UnitID)
		}
		old, err := readCompatibilityFile(root, item.plan.CanonicalCandidatePath)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		present := err == nil
		if present && len(old) == 0 {
			return nil, fmt.Errorf("empty old canonical target %s", u.UnitID)
		}
		oldSHA := ""
		if present {
			oldSHA = sum(old)
		}
		e.Membership = append(e.Membership, u.UnitID)
		e.Units = append(e.Units, TargetSuccessorUnit{UnitID: u.UnitID, Kind: u.UnitKind, SourcePath: u.SourcePath, SourceSHA: u.SourceSHA256, BatchID: u.SelectedBatchID, Candidate: GlossaryArchiveReference{u.CandidatePath, u.CandidateSHA256}, Validation: GlossaryArchiveReference{u.ValidationPath, u.ValidationSHA256}, CanonicalPath: item.plan.CanonicalCandidatePath, OldSHA: oldSHA, OldPresent: present, NewSHA: item.plan.CandidateSHA256, Changed: oldSHA != item.plan.CandidateSHA256, OldBytes: string(old)})
	}
	e.PlanIdentity = sum(mustJSON(e.Units))
	return e, nil
}

func stageNativePromotionReceipt(e *TargetSuccessorEvidence, stagedReceipt, statusPath string) error {
	b, err := os.ReadFile(statusPath)
	if err != nil {
		return err
	}
	e.Status = GlossaryArchiveReference{"locales/" + e.Locale + "/status.tsv", sum(b)}
	e.StatusBytes = string(b)
	e.Identity = successorIdentity(*e)
	if _, err := successorPath(e); err != nil {
		return err
	}
	return writeNewGenerationFile(stagedReceipt, append(mustJSON(e), '\n'))
}

// Publication is the commit point. Recheck the installed output and complete
// sealed artifact before the exclusive link; never publish from partial output.
func checkInstalledNativePromotion(root, stagedReceipt string, e *TargetSuccessorEvidence) error {
	refs := append([]GlossaryArchiveReference{e.Status}, successorCanonicalReferences(e)...)
	refs = append(refs, e.ProofInputs...)
	refs = append(refs, e.Snapshot, e.Finalization, GlossaryArchiveReference{glossaryReviewGlossaryPath(e.Locale), e.GlossarySHA})
	for _, r := range refs {
		b, err := readCompatibilityFile(root, r.Path)
		if err != nil || sum(b) != r.SHA256 {
			return fmt.Errorf("promotion installed output/input changed %s", r.Path)
		}
	}
	b, err := os.ReadFile(stagedReceipt)
	if err != nil {
		return err
	}
	var staged TargetSuccessorEvidence
	if err := contentidentity.StrictJSON(b, &staged); err != nil {
		return err
	}
	if string(mustJSON(staged)) != string(mustJSON(e)) || staged.Identity != successorIdentity(staged) {
		return fmt.Errorf("staged promotion receipt changed")
	}
	return nil
}

func checkNativePromotionPrecommit(root string, e *TargetSuccessorEvidence) error {
	for _, r := range append(append([]GlossaryArchiveReference{}, e.ProofInputs...), e.Snapshot, e.Finalization, GlossaryArchiveReference{"locales/" + e.Locale + "/status.tsv", e.PreviousStatusSHA}, GlossaryArchiveReference{glossaryReviewGlossaryPath(e.Locale), e.GlossarySHA}) {
		b, err := readCompatibilityFile(root, r.Path)
		if err != nil || sum(b) != r.SHA256 {
			return fmt.Errorf("promotion precommit input changed %s", r.Path)
		}
	}
	for _, u := range e.Units {
		b, err := readCompatibilityFile(root, u.CanonicalPath)
		if !u.OldPresent && os.IsNotExist(err) {
			continue
		}
		if err != nil || !u.OldPresent || sum(b) != u.OldSHA {
			return fmt.Errorf("promotion precommit old target changed %s", u.UnitID)
		}
	}
	return nil
}
