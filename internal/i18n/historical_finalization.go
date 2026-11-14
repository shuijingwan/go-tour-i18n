package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Coupling inventory (authority: docs/CURRENT_TRANSLATION_AUTHORIZATION.md):
// immutable history: QC/Snapshot/candidate/validation/frozen input/glossary;
// historical mutable debt: predecessor compatibility/status/UI/SEO projection;
// current semantics: source, glossary, full Unified Review, adapter contract;
// current machine gate: exact-candidate validation + per-TU impact assessment;
// derived freshness: package/Surface/SEO independently consume their authority;
// runtime identity: coverage/canonical/Production, never a Reviewer A judgment.

// HistoricalFinalizationIntegrity is a flattened immutable audit proof. It
// records actual Reviewer A origins, not a replay of historical CURRENT states.
type HistoricalFinalizationIntegrity struct {
	Finalization       QualityCheckFinalization   `json:"finalization"`
	References         []GlossaryArchiveReference `json:"immutable_references"`
	ReviewerGlossaries []string                   `json:"reviewer_glossary_sha256"`
}

// VerifyHistoricalQualityCheckFinalization verifies the original judgment in
// its frozen world. The optional Catalog only supplies representation-owned
// historical Surface/replacement proof readers, never current glossary lookup.
// Intermediate compatibility logs are not A judgments: current re-attestation
// separately compares each original Reviewer world to the actual current world.
func VerifyHistoricalQualityCheckFinalization(root, locale, snapshotID string, catalogs ...*Catalog) (*HistoricalFinalizationIntegrity, error) {
	var catalog *Catalog
	if len(catalogs) > 0 {
		catalog = catalogs[0]
	}
	return verifyHistoricalQualityCheckFinalization(root, locale, snapshotID, catalog, map[string]bool{})
}

func verifyHistoricalQualityCheckFinalization(root, locale, snapshotID string, catalog *Catalog, visiting map[string]bool) (*HistoricalFinalizationIntegrity, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	if err := validateSnapshotID(snapshotID); err != nil {
		return nil, err
	}
	if visiting[snapshotID] {
		return nil, fmt.Errorf("historical finalization lineage cycle")
	}
	visiting[snapshotID] = true
	defer delete(visiting, snapshotID)
	refs := map[string]GlossaryArchiveReference{}
	bind := func(path, expected string) ([]byte, error) {
		b, err := readCompatibilityFile(root, path)
		if err != nil {
			return nil, err
		}
		sha := sum(b)
		if expected != "" && expected != sha {
			return nil, fmt.Errorf("historical hash mismatch: %s", path)
		}
		if old, ok := refs[path]; ok && old.SHA256 != sha {
			return nil, fmt.Errorf("historical reference conflict")
		}
		refs[path] = GlossaryArchiveReference{Path: path, SHA256: sha}
		return b, nil
	}
	finalPath := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, snapshotID, "finalization.json"))
	b, err := bind(finalPath, "")
	if err != nil {
		return nil, err
	}
	var f QualityCheckFinalization
	if err := contentidentity.StrictJSON(b, &f); err != nil {
		return nil, err
	}
	if f.SchemaVersion != QualityCheckFinalizationSchemaVersion || f.EvidenceType != QualityCheckFinalizationEvidenceType || f.Locale != locale || f.SnapshotID != snapshotID || f.Rubric != TranslationQualityRubric {
		return nil, fmt.Errorf("historical finalization identity/rubric mismatch")
	}
	if _, err := time.Parse(time.RFC3339Nano, f.FinalizedAt); err != nil {
		return nil, err
	}
	manifestPath := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, snapshotID, "manifest.json"))
	if f.SnapshotManifestPath != manifestPath {
		return nil, fmt.Errorf("historical Snapshot path mismatch")
	}
	if _, err := bind(manifestPath, f.SnapshotManifestSHA256); err != nil {
		return nil, err
	}
	s, err := readQualityCheckSnapshot(root, locale, snapshotID, false)
	if err != nil {
		return nil, err
	}
	if f.GlossaryPath != s.GlossaryPath || f.GlossarySHA256 != s.GlossarySHA256 || len(f.Units) != s.UnitCount || len(f.QCResults) == 0 {
		return nil, fmt.Errorf("historical finalization complete scope mismatch")
	}
	if _, err := readArchivedGlossary(root, locale, f.GlossarySHA256); err != nil {
		return nil, err
	}
	archivePath, _ := glossaryArchivePath(locale, f.GlossarySHA256)
	if _, err := bind(archivePath, f.GlossarySHA256); err != nil {
		return nil, err
	}
	results := map[string]*QualityCheckResults{}
	snapshots := map[string]*QualityCheckSnapshotManifest{snapshotID: s}
	correctionInputs := map[string]QualityCheckFinalizationInput{}
	for _, ref := range f.QCResults {
		if err := validateSnapshotID(ref.SnapshotID); err != nil {
			return nil, err
		}
		if strings.HasPrefix(ref.Path, qualityCheckCorrectionDir(locale, ref.SnapshotID)+"/") {
			if _, duplicate := correctionInputs[ref.Path]; duplicate {
				return nil, fmt.Errorf("duplicate historical QC correction input")
			}
			if _, err := bind(ref.Path, ref.SHA256); err != nil {
				return nil, err
			}
			correctionInputs[ref.Path] = ref
			continue
		}
		if results[ref.SnapshotID] != nil || ref.Path != filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, ref.SnapshotID, "quality-check-results.json")) {
			return nil, fmt.Errorf("historical QC input exact-set/path mismatch")
		}
		if _, err := bind(ref.Path, ref.SHA256); err != nil {
			return nil, err
		}
		original, err := readQualityCheckSnapshot(root, locale, ref.SnapshotID, false)
		if err != nil {
			return nil, err
		}
		r, err := readQualityCheckResults(root, locale, original)
		if err != nil || r == nil || r.Rubric != TranslationQualityRubric {
			return nil, fmt.Errorf("historical QC missing/invalid: %s: %v", ref.SnapshotID, err)
		}
		if _, err := bind(filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, ref.SnapshotID, "manifest.json")), r.SnapshotManifestSHA256); err != nil {
			return nil, err
		}
		results[ref.SnapshotID], snapshots[ref.SnapshotID] = r, original
		if original.CurrentRevision != nil {
			for _, input := range original.CurrentRevision.Evidence {
				if _, err := bind(input.Path, input.SHA256); err != nil {
					return nil, err
				}
			}
		}
	}
	// Corrections are exact immutable QC inputs, never an unbound A overlay.
	// Old finalizations with no correction keep their original input inventory.
	for id := range results {
		original, err := readOriginalQualityCheckResults(root, locale, snapshots[id])
		if err != nil {
			return nil, err
		}
		_, corrections, err := readQualityCheckCorrections(root, snapshots[id], original)
		if err != nil {
			return nil, err
		}
		for _, ref := range corrections {
			bound, ok := correctionInputs[ref.Path]
			if !ok || bound.SnapshotID != id || bound.SHA256 != ref.SHA256 {
				return nil, fmt.Errorf("historical QC correction not exactly bound")
			}
			delete(correctionInputs, ref.Path)
		}
	}
	if len(correctionInputs) != 0 {
		return nil, fmt.Errorf("orphan historical QC correction input")
	}
	// Validate recorded predecessor graph without any mutable/current lookup.
	zeroReviewLineage := map[string]string{}
	for id := range results {
		seen := map[string]bool{}
		for next := id; next != ""; {
			if seen[next] {
				return nil, fmt.Errorf("historical QC lineage cycle")
			}
			seen[next] = true
			r := results[next]
			if r == nil {
				previous, ok := zeroReviewLineage[next]
				if !ok {
					var err error
					previous, err = verifyHistoricalZeroReviewPredecessor(root, locale, next, catalog, visiting, bind)
					if err != nil {
						return nil, fmt.Errorf("historical QC lineage missing or invalid finalized carry: %s: %w", next, err)
					}
					zeroReviewLineage[next] = previous
				}
				next = previous
				continue
			}
			next = r.PreviousSnapshotID
		}
	}
	if f.PredecessorFinalization != nil {
		if f.PreviousSnapshotID == "" || f.PredecessorFinalization.Path != filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, f.PreviousSnapshotID, "finalization.json")) {
			return nil, fmt.Errorf("historical predecessor identity mismatch")
		}
		if _, err := bind(f.PredecessorFinalization.Path, f.PredecessorFinalization.SHA256); err != nil {
			return nil, err
		}
	}
	proof := &HistoricalFinalizationIntegrity{Finalization: f, ReviewerGlossaries: []string{}}
	// The complete frozen Review and replacement graph are shared by many
	// Units. Validate each exact dependency once in this invocation, while all
	// per-Unit A, Snapshot, source, candidate and validation checks stay below.
	checkedReviews := map[struct{ glossary, reviewer string }]bool{}
	replacements := map[TranslationUnitReplacementContext]*TranslationUnitReplacementResolution{}
	machineContexts := map[string]*machineSelection{}
	successors := NewRepresentationSuccessorVerifier(root, catalog, locale)
	siblings := &historicalMembershipReviewers{root: root, locale: locale, catalog: catalog, snapshot: s, finalization: &f, visiting: visiting, bind: bind, results: results, proofs: map[*Catalog]*HistoricalFinalizationIntegrity{}}
	for i, u := range f.Units {
		var currentSuccessor *TranslationUnitReplacement
		if u.Index != i+1 || u.UnitID != s.Units[i].UnitID || u.Rating != "A" || !qualityCheckSnapshotIdentityMatches(u.Snapshot, s.Units[i]) {
			return nil, fmt.Errorf("historical A-only membership mismatch at %d", i+1)
		}
		r, original := results[u.SourceSnapshotID], snapshots[u.SourceSnapshotID]
		if r == nil || original == nil {
			r, original, err = siblings.resolve(u)
			if err != nil {
				return nil, fmt.Errorf("original Reviewer evidence missing: %s: %w", u.UnitID, err)
			}
		}
		judgmentID, judgmentSnapshot := u.UnitID, u.Snapshot
		if u.Snapshot.RepresentationReconciliation != nil {
			if catalog == nil {
				return nil, fmt.Errorf("explicit representation catalog required for machine successor history")
			}
			_, machine, err := machineSnapshotEvidenceInWorld(root, catalog, locale, u.Snapshot, machineContexts, true)
			if err != nil {
				return nil, err
			}
			// This mapping has passed the receipt, exact candidate/validation
			// and receipt-world machine proof gates. Current eligibility is
			// independently assessed by the current authorization reader.
			currentSuccessor = &machine.unit.Replacement
			for _, ref := range machineAuthorityReferences(machine) {
				if _, err := bind(ref.Path, ref.SHA256); err != nil {
					return nil, err
				}
			}
			// Original A remains an old-ID judgment. The explicit, verified
			// machine mapping above binds it to this byte-identical successor.
			u.UnitID = machine.unit.Predecessor.UnitID
			u.Snapshot = machine.unit.Predecessor
			directCurrent := false
			if original != nil {
				for _, unit := range original.Units {
					if qualityCheckSnapshotIdentityMatches(unit, judgmentSnapshot) {
						directCurrent = true
					}
				}
			}
			if !directCurrent {
				judgmentID, judgmentSnapshot = u.UnitID, u.Snapshot
			}
		}
		judgmentFound := false
		for _, judgment := range r.Results {
			if judgment.UnitID == judgmentID && judgment.Rating == "A" && qualityCheckSnapshotIdentityMatches(original.Units[judgment.Index-1], judgmentSnapshot) {
				judgmentFound = true
			}
		}
		if !judgmentFound {
			return nil, fmt.Errorf("original exact Reviewer A missing: %s", u.UnitID)
		}
		if _, err := readArchivedGlossary(root, locale, original.GlossarySHA256); err != nil {
			return nil, err
		}
		archivePath, _ := glossaryArchivePath(locale, original.GlossarySHA256)
		if _, err := bind(archivePath, original.GlossarySHA256); err != nil {
			return nil, err
		}
		proof.ReviewerGlossaries = append(proof.ReviewerGlossaries, original.GlossarySHA256)
		// Bind a real independent full historical Review when present; original
		// legacy coverage remains a frozen, locale-bound fallback authority.
		reviewKey := struct{ glossary, reviewer string }{original.GlossarySHA256, r.ReviewerSession}
		if !checkedReviews[reviewKey] {
			if err := bindHistoricalGlossaryReview(root, locale, original.GlossarySHA256, r.ReviewerSession, bind); err != nil {
				return nil, err
			}
			checkedReviews[reviewKey] = true
		}
		if _, err := bind(u.Snapshot.CandidatePath, u.Snapshot.CandidateSHA256); err != nil {
			return nil, err
		}
		if _, err := bind(u.Snapshot.ValidationPath, u.Snapshot.ValidationSHA256); err != nil {
			return nil, err
		}
		base := filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, u.Snapshot.SelectedBatchID))
		if _, err := bind(base+"/manifest.json", ""); err != nil {
			return nil, err
		}
		m, err := readRetranslationProcessManifest(filepath.Join(root, filepath.FromSlash(base)), locale, u.Snapshot.SelectedBatchID)
		if err != nil {
			return nil, err
		}
		var record RetranslationBatchUnit
		for _, v := range m.Units {
			if v.UnitID == u.UnitID {
				record = v
			}
		}
		if record.UnitID != u.UnitID || record.SourcePath != u.Snapshot.SourcePath || record.SourceSHA256 != u.Snapshot.SourceSHA256 || record.UnitKind != u.Snapshot.UnitKind {
			return nil, fmt.Errorf("historical source/input authority mismatch: %s", u.UnitID)
		}
		if err := ensureNoGenerationImportPending(filepath.Join(root, filepath.FromSlash(base))); err != nil {
			return nil, err
		}
		if record.RepresentationBridge != nil {
			if _, err := bind(record.RepresentationBridge.Path, record.RepresentationBridge.SHA256); err != nil {
				return nil, err
			}
			receipt, err := readFrozenSameUnitBridge(root, locale, *record.RepresentationBridge)
			if err != nil {
				return nil, err
			}
			matched := false
			for _, q := range receipt.Units {
				matched = matched || (q.UnitID == record.UnitID && record.RevisionFeedbackSource == "quality_check" && q.PreviousRating == record.PreviousRating && q.Finding == record.PreviousFinding && receipt.PreviousSnapshotID == record.PreviousSnapshotID)
			}
			if !matched {
				return nil, fmt.Errorf("historical bridge QC identity mismatch")
			}
			for _, ref := range receipt.SameUnitBridge.FrozenDependencies {
				if _, err := bind(ref.Path, ref.SHA256); err != nil {
					return nil, err
				}
			}
		}
		if record.RepresentationBridge == nil && hasInheritedBridgePredecessor(record) {
			refs, err := frozenSuccessorBridgeReferences(root, locale, record, u.Snapshot.SelectedBatchID, u.Snapshot.Attempt)
			if err != nil {
				return nil, err
			}
			for _, ref := range refs {
				if _, err := bind(ref.Path, ref.SHA256); err != nil {
					return nil, err
				}
			}
		}
		if isReopenFeedback(record.RevisionFeedbackSource) {
			if err := bindHistoricalSurfaceRevision(root, locale, catalog, record, bind, replacements, currentSuccessor, successors); err != nil {
				return nil, err
			}
		}
		if _, err := bind(base+"/"+record.InputPath, record.InputSHA256); err != nil {
			return nil, err
		}
		if _, err := bind(base+"/result.json", ""); err != nil {
			return nil, err
		}
		p, err := readPromotionResult(filepath.Join(root, filepath.FromSlash(base)), locale, m.BatchID, len(m.Units))
		if err != nil {
			return nil, err
		}
		var result RetranslationUnitResult
		for _, v := range p.Units {
			if v.UnitID == u.UnitID {
				result = v
			}
		}
		if result.Status != "passed" || base+"/"+result.CandidatePath != u.Snapshot.CandidatePath || base+"/"+result.ValidationPath != u.Snapshot.ValidationPath {
			return nil, fmt.Errorf("historical machine result mismatch: %s", u.UnitID)
		}
		validation, err := readPromotionValidation(filepath.Join(root, filepath.FromSlash(base)), m.BatchID, locale, record, result)
		if err != nil || validation.Attempt != u.Snapshot.Attempt {
			return nil, fmt.Errorf("historical validation attempt mismatch: %s", u.UnitID)
		}
		if _, err := bind(base+"/"+validation.RawResponsePath, ""); err != nil {
			return nil, err
		}
		if record.GenerationOrigin != nil {
			for _, ref := range []GlossaryArchiveReference{record.GenerationOrigin.Bundle, record.GenerationOrigin.Result} {
				if _, err := bind(ref.Path, ref.SHA256); err != nil {
					return nil, err
				}
			}
		}
	}
	if siblings.check != nil {
		if err := siblings.check(); err != nil {
			return nil, err
		}
	}
	for _, r := range refs {
		// Memoization never accepts a dependency that changed while the graph
		// was being validated, even if its path stayed the same.
		b, err := readCompatibilityFile(root, r.Path)
		if err != nil || sum(b) != r.SHA256 {
			return nil, fmt.Errorf("historical dependency changed during validation: %s", r.Path)
		}
		proof.References = append(proof.References, r)
	}
	sort.Slice(proof.References, func(i, j int) bool { return proof.References[i].Path < proof.References[j].Path })
	return proof, nil
}

// A missing response is bridged only by a fully verified immutable zero-review
// finalization. Reuse the historical verifier, never current eligibility or a
// permissive skip over a missing QC input.
func verifyHistoricalZeroReviewPredecessor(root, locale, snapshotID string, catalog *Catalog, visiting map[string]bool, bind func(string, string) ([]byte, error)) (string, error) {
	previous, err := finalizedCarryPredecessor(root, locale, snapshotID)
	if err != nil {
		return "", err
	}
	if previous == "" {
		return "", fmt.Errorf("zero-review carry requires finalized predecessor")
	}
	resultPath := filepath.ToSlash(filepath.Join("data", "quality-check-snapshots", locale, snapshotID, "quality-check-results.json"))
	if _, err := readCompatibilityFile(root, resultPath); !os.IsNotExist(err) {
		return "", fmt.Errorf("zero-review carry requires absent QC response: %s: %v", snapshotID, err)
	}
	carry, err := verifyHistoricalQualityCheckFinalization(root, locale, snapshotID, catalog, visiting)
	if err != nil {
		return "", err
	}
	if carry.Finalization.PreviousSnapshotID != previous || carry.Finalization.PredecessorFinalization == nil {
		return "", fmt.Errorf("zero-review carry requires exact predecessor finalization")
	}
	predecessor, err := verifyHistoricalQualityCheckFinalization(root, locale, previous, catalog, visiting)
	if err != nil {
		return "", err
	}
	if !reflect.DeepEqual(carry.Finalization.QCResults, predecessor.Finalization.QCResults) || len(carry.Finalization.Units) != len(predecessor.Finalization.Units) {
		return "", fmt.Errorf("zero-review carry changed complete frozen Reviewer scope")
	}
	for _, proof := range []*HistoricalFinalizationIntegrity{carry, predecessor} {
		for _, ref := range proof.References {
			if _, err := bind(ref.Path, ref.SHA256); err != nil {
				return "", err
			}
		}
	}
	return previous, nil
}

func bindHistoricalGlossaryReview(root, locale, sha, reviewer string, bind func(string, string) ([]byte, error)) error {
	for _, dir := range []string{"data/unified-glossary-reviews/" + locale, "data/glossary-reviews/" + locale} {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".review.json") {
				continue
			}
			path := dir + "/" + e.Name()
			b, err := readCompatibilityFile(root, path)
			if err != nil {
				return err
			}
			if strings.HasPrefix(dir, "data/unified") {
				var r UnifiedGlossaryReview
				if err := contentidentity.StrictJSON(b, &r); err != nil {
					return err
				}
				if r.GlossarySHA != sha || r.Decision != "passed" {
					continue
				}
				if err := validateCompatibilityReview(root, locale, sha, GlossaryArchiveReference{path, sum(b)}); err != nil {
					return err
				}
				if reviewer != "" && reviewer == r.GenerationSession {
					return fmt.Errorf("original generation session cannot supply Reviewer A")
				}
				if _, err := bind(path, sum(b)); err != nil {
					return err
				}
				_, err = bind(unifiedReviewerArchive(locale, r.BundleSHA), r.BundleSHA)
				return err
			}
			var r GlossaryReviewReceipt
			if err := contentidentity.StrictJSON(b, &r); err != nil {
				return err
			}
			if r.GlossarySHA256 != sha || r.Decision != "passed" {
				continue
			}
			if err := validateCompatibilityReview(root, locale, sha, GlossaryArchiveReference{path, sum(b)}); err != nil {
				return err
			}
			_, err = bind(path, sum(b))
			return err
		}
	}
	b, err := readCompatibilityFile(root, legacyGlossaryReviewCoveragePath)
	if err != nil {
		return fmt.Errorf("original full Glossary Review missing: %s", sha)
	}
	// The fixed legacy migration table is historical authority. Its existing
	// validator takes an explicit frozen SHA and verifies the original gate;
	// it does not read today's glossary/config/runtime projections.
	covered, err := currentLegacyGlossaryReviewCoverage(root, locale, glossaryReviewGlossaryPath(locale), sha)
	if err != nil || !covered {
		return fmt.Errorf("original legacy Glossary Review invalid: %s: %v", sha, err)
	}
	var legacy legacyGlossaryReviewCoverage
	if err := contentidentity.StrictJSON(b, &legacy); err != nil {
		return err
	}
	for _, e := range legacy.Entries {
		if e.Locale == locale && e.GlossarySHA256 == sha {
			if _, err := bind(legacyGlossaryReviewCoveragePath, sum(b)); err != nil {
				return err
			}
			_, err := bind(e.SurfaceReviewGatePath, e.SurfaceReviewGateSHA256)
			return err
		}
	}
	return fmt.Errorf("original full Glossary Review missing: %s", sha)
}

func bindHistoricalSurfaceRevision(root, locale string, c *Catalog, record RetranslationBatchUnit, bind func(string, string) ([]byte, error), replacements map[TranslationUnitReplacementContext]*TranslationUnitReplacementResolution, currentSuccessor *TranslationUnitReplacement, successors *RepresentationSuccessorVerifier) error {
	path := "data/quality-check-reopens/" + locale + "/" + record.RevisionAuthorizationID + ".json"
	b, err := bind(path, "")
	if err != nil {
		return err
	}
	var reopen QualityCheckSurfaceReopen
	if err := contentidentity.StrictJSON(b, &reopen); err != nil {
		return err
	}
	ref := reopen.revisionEvidence()
	if reopen.SchemaVersion != qualityCheckSurfaceReopenSchemaVersion || reopen.EvidenceType != qualityCheckSurfaceReopenEvidenceType || reopen.Locale != locale || reopen.ReopenID != record.RevisionAuthorizationID || reopen.PreviousSnapshotID != record.PreviousSnapshotID || reopen.UnitCount != len(reopen.Units) || reopen.UnitCount < 1 || reopen.UnitCount > DefaultQualityCheckBatchLimit || reopen.findingForUnit(record.UnitID) != record.PreviousFinding || ref.Path != record.PreviousReviewPath || ref.SHA256 != record.PreviousReviewSHA256 || record.RevisionFeedbackSource != reopen.revisionFeedbackSource() {
		return fmt.Errorf("historical Surface revision identity/finding mismatch")
	}
	if reopen.AuthoritySource == "quality-check" {
		if err := verifyQualityCheckReplacement(root, c, &reopen); err != nil {
			return err
		}
		a := reopen.ReplacementContext.QualityCheck
		for _, ref := range []GlossaryArchiveReference{a.Snapshot, a.Results, a.EngineeringProof, reopen.ReplacementContext.HistoricalDocuments} {
			if _, err := bind(ref.Path, ref.SHA256); err != nil {
				return err
			}
		}
		return nil
	}
	if reopen.AuthoritySource == RepresentationMachineAuthority {
		world, err := historicalRepresentationRevisionWorld(root, c, &reopen)
		if err != nil {
			return err
		}
		for _, ref := range []GlossaryArchiveReference{world.snapshot, world.glossary} {
			if _, err := bind(ref.Path, ref.SHA256); err != nil {
				return err
			}
		}
		if err := verifyRepresentationRevisionInWorld(root, c, &reopen, world); err != nil {
			return err
		}
		a := reopen.ReplacementContext.Machine
		refs := []GlossaryArchiveReference{a.Options.Impact, a.Options.HistoricalDocuments, a.OriginalRaw, a.OriginalManifest, a.OriginalResult, {Path: a.Predecessor.CandidatePath, SHA256: a.Predecessor.CandidateSHA256}, {Path: a.Predecessor.ValidationPath, SHA256: a.Predecessor.ValidationSHA256}, {Path: filepath.ToSlash(filepath.Join("data/quality-check-snapshots", locale, a.Options.PreviousSnapshotID, "manifest.json")), SHA256: a.Options.SnapshotSHA256}}
		for _, q := range a.QCInputs {
			refs = append(refs, GlossaryArchiveReference{Path: q.Path, SHA256: q.SHA256})
		}
		for _, ref := range refs {
			if _, err := bind(ref.Path, ref.SHA256); err != nil {
				return err
			}
		}
		return nil
	}
	if _, err := bind(reopen.PreviousFinalizationPath, reopen.PreviousFinalizationSHA256); err != nil {
		return err
	}
	if _, err := bind(ref.Path, ref.SHA256); err != nil {
		return err
	}
	if reopen.PreviewFailure != nil {
		return verifyPreviewReopen(root, c, &reopen, false)
	}
	historicalCatalog := c
	if reopen.ReplacementContext != nil {
		for _, ref := range []GlossaryArchiveReference{reopen.ReplacementContext.PredecessorReopen, reopen.ReplacementContext.HistoricalDocuments} {
			if _, err := bind(ref.Path, ref.SHA256); err != nil {
				return err
			}
		}
		priorBytes, err := bind(reopen.ReplacementContext.PredecessorReopen.Path, reopen.ReplacementContext.PredecessorReopen.SHA256)
		if err != nil {
			return err
		}
		if err := verifyHistoricalReopenPredecessor(locale, &reopen, priorBytes); err != nil {
			return err
		}
		resolution := replacements[*reopen.ReplacementContext]
		if resolution == nil {
			var err error
			resolution, err = replacementResolution(root, c, reopen.ReplacementContext)
			if err != nil {
				return err
			}
			replacements[*reopen.ReplacementContext] = resolution
		}
		historicalCatalog = resolution.HistoricalCatalog
		// Old finalized Snapshots predate reconciliation and therefore have no
		// machine Snapshot ref. Resolve only their exact recorded event identity
		// to the resolver's current identity through the shared native verifier.
		if currentSuccessor == nil {
			for _, u := range reopen.Units {
				p, ok := resolution.Units[u.Snapshot.UnitID]
				if u.UnitID != record.UnitID || !ok || (p.CurrentUnitID == record.UnitID && p.CurrentInputSHA256 == record.InputSHA256) {
					continue
				}
				var refs []GlossaryArchiveReference
				var err error
				currentSuccessor, refs, err = successors.VerifyHistorical(record.UnitID, p.CurrentUnitID)
				if err != nil {
					return err
				}
				for _, ref := range refs {
					if _, err := bind(ref.Path, ref.SHA256); err != nil {
						return err
					}
				}
				break
			}
		}
		if err := verifyHistoricalSurfaceMapping(root, locale, record, &reopen, resolution, currentSuccessor, bind); err != nil {
			return err
		}
	}
	if reopen.IntegratedAuthority != nil {
		if historicalCatalog == nil || historicalCatalog.SurfaceAuthorityReader == nil {
			return fmt.Errorf("Integrated Surface authority reader required")
		}
		a, err := historicalCatalog.SurfaceAuthorityReader(root, locale, reopen.SurfaceReviewID, false, historicalCatalog)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(a, reopen.IntegratedAuthority) {
			return fmt.Errorf("original Surface authority changed")
		}
		for _, ref := range append([]GlossaryArchiveReference{a.Receipt, a.Evidence}, a.Context...) {
			// Current projections are retained inside the immutable receipt, not
			// installed as permanent historical filesystem equality constraints.
			if strings.HasPrefix(ref.Path, "data/") && ref.Path != "data/site-content-scope.json" {
				if _, err := bind(ref.Path, ref.SHA256); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
