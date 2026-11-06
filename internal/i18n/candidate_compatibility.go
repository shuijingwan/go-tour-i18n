package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// acceptedCandidateCarry is a current-eligibility check, not a new invocation
// or Reviewer judgment. A sealed complete authorization supplies the original
// accepted bytes; the shared assessment rechecks them in today's world. Other
// Units may now have authorized successors, so only this exact selected Unit
// must remain compatible. Selection itself never falls back to that history.
func (p *PreparedGlossaryCompatibility) acceptedCandidateCarry(unit *TranslationUnit, batchDir string, record RetranslationBatchUnit, validation *RetranslationValidation, candidate []byte) error {
	if p == nil || p.catalog == nil || p.historicalSHA != "" {
		return fmt.Errorf("current candidate compatibility context required")
	}
	_, _, scope, _, err := translationAuthorizationScope(p.catalog)
	if err != nil {
		return err
	}
	dir := "data/translation-authorizations/" + p.locale + "/" + scope
	path, err := compatibilityPath(p.root, dir+"/.inventory", false)
	if err != nil {
		return fmt.Errorf("formal candidate authorization missing: %w", err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("formal candidate authorization missing: %w", err)
	}
	current, err := readCompatibilityFile(p.root, glossaryReviewGlossaryPath(p.locale))
	if err != nil {
		return err
	}
	validationPath := filepath.ToSlash(filepath.Join(batchDir, "validation", retranslationReviewName(unit)))
	validationData, err := os.ReadFile(validationPath)
	if err != nil {
		return err
	}
	base, err := repositoryRelativePath(p.root, batchDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe candidate authorization member")
		}
		// This reader validates the sealed identity, all original immutable
		// refs and original Review. It does not accept its past current world.
		proof, err := loadSealedTranslationHistory(p.root, p.catalog, p.locale, entry.Name())
		if err != nil {
			return err
		}
		if proof == nil {
			continue
		}
		index := -1
		for i, accepted := range proof.Finalization.Units {
			s := accepted.Snapshot
			if s.UnitID == unit.ID && s.UnitKind == unit.Kind && s.SourcePath == unit.SourcePath && s.SourceSHA256 == unit.SourceSHA256 && s.SelectedBatchID == filepath.Base(batchDir) && s.CandidatePath == base+"/candidates/"+retranslationUnitCandidateName(unit) && s.CandidateSHA256 == sum(candidate) && s.ValidationPath == base+"/validation/"+retranslationReviewName(unit) && s.ValidationSHA256 == sum(validationData) && s.Attempt == validation.Attempt {
				index = i
			}
		}
		if index < 0 {
			continue
		}
		members, err := os.ReadDir(filepath.Join(p.root, filepath.FromSlash(dir), entry.Name()))
		if err != nil {
			return err
		}
		eligible := false
		for _, member := range members {
			b, err := readCompatibilityFile(p.root, dir+"/"+entry.Name()+"/"+member.Name())
			if err != nil {
				return err
			}
			var a CurrentTranslationAuthorization
			if err := contentidentity.StrictJSON(b, &a); err != nil {
				return err
			}
			// loadSealedTranslationHistory already verified these complete
			// records. A stale epoch cannot authorize a new glossary by itself.
			if a.Glossary.SHA256 == sum(current) && index < len(a.Units) && a.Units[index].Classification == "compatible" && a.Units[index].Validation == "passed" {
				eligible = true
			}
		}
		if !eligible {
			return fmt.Errorf("candidate authorization glossary/classification is stale or non-compatible")
		}
		if p.candidateAssessments == nil {
			p.candidateAssessments = map[string]*CurrentTranslationAuthorization{}
		}
		a := p.candidateAssessments[entry.Name()]
		if a == nil {
			a, err = assessCurrentTranslationProof(p.root, p.catalog, p.locale, *proof)
			if err != nil {
				return err
			}
			p.candidateAssessments[entry.Name()] = a
		}
		x := a.Units[index]
		if x.Classification != "compatible" || x.Validation != "passed" {
			return fmt.Errorf("candidate carry blocked: classification=%s validation=%s reasons=%v", x.Classification, x.Validation, x.Reasons)
		}
		return nil
	}
	return fmt.Errorf("no formal authorization for exact selected candidate %s", record.UnitID)
}

// Re-attested whole sets and mixed baselines must enforce the same exact
// formal-input boundary as ordinary archived-candidate selection.
func validateSelectedArchivedInput(unit *TranslationUnit, choice selectedRetranslationUnit, glossary *Glossary) error {
	if choice.manifest.GenerationOrigin == nil {
		return nil
	}
	input, err := os.ReadFile(filepath.Join(choice.batchDir, filepath.FromSlash(choice.manifest.InputPath)))
	if err != nil || sum(input) != choice.manifest.InputSHA256 {
		return fmt.Errorf("%s: archived invocation saved input hash mismatch", unit.ID)
	}
	protected, err := prepareTranslationUnitInput(unit, glossary)
	if err != nil || len(protected.Tokens) != choice.manifest.ProtectedTokenCount || (!bytes.Equal(input, []byte(protected.Text)) && !bytes.Equal(input, canonicalizeRetranslationArtifactEOF([]byte(protected.Text)))) {
		return fmt.Errorf("%s: archived invocation formal input changed; proper recovery required", unit.ID)
	}
	_, err = validateSavedRetranslationInputTokens(unit, input, choice.manifest.ProtectedTokenCount)
	return err
}
