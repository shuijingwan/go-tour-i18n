package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
)

// These observations belong to one readonly invocation. They are neither a
// persisted protocol nor permission to select a candidate or recover a batch.
type historicalCandidateFile struct {
	path string
	data []byte
}

type verifiedHistoricalFinalCandidate struct {
	unit           *TranslationUnit
	validation     *RetranslationValidation
	attempt        int
	candidate      historicalCandidateFile
	validationFile historicalCandidateFile
	raw            historicalCandidateFile
}

// Verify explicitly selected facts with the supplied decoder, independently of
// installed-proof currentness. The caller still owns scope, provenance/history,
// full-batch eligibility and publication checks. The active validator is never
// optional, including when a frozen inverse is used to restore the old raw.
func verifyHistoricalFinalCandidate(root string, c *Catalog, g *Glossary, m *RetranslationBatchManifest, choice selectedRetranslationUnit, frozen TranslationUnitAdapter, current protectedTranslation) (*verifiedHistoricalFinalCandidate, error) {
	u, err := c.Unit(choice.manifest.UnitID)
	if err != nil {
		return nil, err
	}
	if choice.batchID != m.BatchID || choice.result.UnitID != choice.manifest.UnitID || choice.result.UnitKind != choice.manifest.UnitKind || choice.manifest.UnitKind != u.Kind || choice.manifest.SourcePath != u.SourcePath || choice.manifest.SourceSHA256 != u.SourceSHA256 {
		return nil, fmt.Errorf("historical final Unit/batch/source identity mismatch")
	}
	v, err := readPromotionValidation(choice.batchDir, m.BatchID, m.Locale, choice.manifest, choice.result)
	if err != nil {
		return nil, err
	}
	base, err := repositoryRelativePath(root, choice.batchDir)
	if err != nil {
		return nil, err
	}
	observe := func(relative string) (historicalCandidateFile, error) {
		path := base + "/" + relative
		b, err := readCompatibilityFile(root, path)
		return historicalCandidateFile{path, b}, err
	}
	candidate, err := observe(choice.result.CandidatePath)
	if err != nil {
		return nil, err
	}
	validation, err := observe(choice.result.ValidationPath)
	if err != nil {
		return nil, err
	}
	observedValidation, err := decodeRetranslationValidation(validation.data, u)
	if err != nil || !reflect.DeepEqual(v, observedValidation) {
		return nil, fmt.Errorf("historical final validation changed during observation: %v", err)
	}
	if v.CandidateSHA256 != "" && v.CandidateSHA256 != sum(candidate.data) {
		return nil, fmt.Errorf("historical final validated candidate SHA mismatch")
	}
	raw, err := observe(v.RawResponsePath)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(choice.manifest.InputPath)
	attempt, err := retryValidationAttemptForExtension(v.RawResponsePath, name[:len(name)-len(filepath.Ext(name))], filepath.Ext(name))
	if err != nil {
		return nil, err
	}
	if attempt != v.Attempt {
		return nil, fmt.Errorf("historical final validation attempt mismatch")
	}
	p, err := prepareTranslationUnitInput(u, g)
	if err != nil {
		return nil, err
	}
	if frozen != nil {
		p.Adapter = frozen
	} else if current.Adapter != nil {
		p = current
	}
	restored, failures := p.restore(string(raw.data))
	if len(failures) != 0 || !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(restored)), candidate.data) || v.Status != "passed" {
		return nil, fmt.Errorf("historical final inverse/validation mismatch")
	}
	if err := validateTranslationUnitWithGlossary(root, c, u, m.Locale, candidate.data, g); err != nil {
		return nil, err
	}
	return &verifiedHistoricalFinalCandidate{unit: u, validation: v, attempt: attempt, candidate: candidate, validationFile: validation, raw: raw}, nil
}
