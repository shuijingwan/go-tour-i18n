package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type selectedRetranslationUnit struct {
	number   int
	batchID  string
	batchDir string
	manifest RetranslationBatchUnit
	result   RetranslationUnitResult
}

type latestRetranslationUnits struct {
	ordered      []*TranslationUnit
	selectedByID map[string]selectedRetranslationUnit
	pageCount    int
	exampleCount int
}

type validatedRetranslationCandidate struct {
	candidatePath  string
	validationPath string
	candidate      []byte
	validationData []byte
	validation     *RetranslationValidation
	attempt        int
}

// selectLatestRetranslationUnits is the shared selection boundary for Candidate
// Snapshot and promotion. For each current workflow unit it selects the batch
// with the greatest numeric batch suffix before checking source identity. A
// newer failed or identity-mismatching result therefore remains selected and
// callers cannot fall back to older successful evidence.
func selectLatestRetranslationUnits(root string, catalog *Catalog, locale string) (*latestRetranslationUnits, error) {
	ordered, pageCount, exampleCount, err := localeWorkflowUnitList(catalog)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*TranslationUnit, len(ordered))
	for _, unit := range ordered {
		if sum(unit.Source) != unit.SourceSHA256 {
			return nil, fmt.Errorf("%s: current Catalog source bytes do not match source_sha256", unit.ID)
		}
		byID[unit.ID] = unit
	}

	base := filepath.Join(root, "data", "retranslation-runs", locale)
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("scan retranslation batches: %w", err)
	}
	selectedByID := make(map[string]selectedRetranslationUnit, len(ordered))
	seenNumbers := map[int]string{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !entry.IsDir() {
			return nil, fmt.Errorf("illegal retranslation batch entry %q", entry.Name())
		}
		match := promotionBatchRE(locale).FindStringSubmatch(entry.Name())
		if match == nil || match[1] != locale {
			return nil, fmt.Errorf("illegal retranslation batch %q", entry.Name())
		}
		number, _ := strconv.Atoi(match[2])
		if number < 1 || seenNumbers[number] != "" {
			return nil, fmt.Errorf("ambiguous or invalid retranslation batch number %03d", number)
		}
		seenNumbers[number] = entry.Name()
		batchDir := filepath.Join(base, entry.Name())
		manifest, err := readRetranslationProcessManifest(batchDir, locale, entry.Name())
		if err != nil {
			return nil, err
		}
		result, err := readPromotionResult(batchDir, locale, entry.Name(), manifest.UnitCount)
		if err != nil {
			return nil, err
		}
		results := make(map[string]RetranslationUnitResult, len(result.Units))
		for _, record := range result.Units {
			if _, duplicate := results[record.UnitID]; duplicate {
				return nil, fmt.Errorf("batch %q has duplicate result unit_id %q", entry.Name(), record.UnitID)
			}
			results[record.UnitID] = record
		}
		seen := map[string]bool{}
		for _, record := range manifest.Units {
			if seen[record.UnitID] {
				return nil, fmt.Errorf("batch %q has duplicate manifest unit_id %q", entry.Name(), record.UnitID)
			}
			seen[record.UnitID] = true
			unit, ok := byID[record.UnitID]
			if !ok {
				continue // Non-workflow historical units do not affect the current locale workflow.
			}
			unitResult, ok := results[unit.ID]
			if !ok || unitResult.UnitKind != record.UnitKind {
				return nil, fmt.Errorf("batch %q result missing or mismatching unit %q", entry.Name(), unit.ID)
			}
			if current, exists := selectedByID[unit.ID]; !exists || number > current.number {
				selectedByID[unit.ID] = selectedRetranslationUnit{
					number: number, batchID: entry.Name(), batchDir: batchDir,
					manifest: record, result: unitResult,
				}
			}
		}
	}
	return &latestRetranslationUnits{
		ordered: ordered, selectedByID: selectedByID,
		pageCount: pageCount, exampleCount: exampleCount,
	}, nil
}

// validateSelectedRetranslationCandidate verifies all immutable evidence for
// the selected final attempt. It deliberately does not inspect Quality Check
// review records, so Candidate Snapshot can run before that gate.
func validateSelectedRetranslationCandidate(root string, catalog *Catalog, locale string, glossary *Glossary, unit *TranslationUnit, choice selectedRetranslationUnit) (*validatedRetranslationCandidate, error) {
	if !selectedRetranslationIdentityMatches(unit, choice) {
		return nil, fmt.Errorf("%s: latest batch %s manifest source identity does not match current Catalog; refusing fallback", unit.ID, choice.batchID)
	}
	if choice.result.Status != "passed" {
		return nil, fmt.Errorf("%s: latest batch %s status %q is not passed; refusing fallback", unit.ID, choice.batchID, choice.result.Status)
	}
	name := filepath.Base(filepath.FromSlash(choice.manifest.InputPath))
	wantCandidate := filepath.ToSlash(filepath.Join("candidates", retranslationUnitCandidateName(unit)))
	wantValidation := filepath.ToSlash(filepath.Join("validation", strings.TrimSuffix(name, filepath.Ext(name))+".json"))
	if choice.result.CandidatePath != wantCandidate || choice.result.ValidationPath != wantValidation {
		return nil, fmt.Errorf("%s: result candidate/validation path mismatch", unit.ID)
	}
	validation, err := readPromotionValidation(choice.batchDir, choice.batchID, locale, choice.manifest, choice.result)
	if err != nil {
		return nil, err
	}
	validationData, err := os.ReadFile(filepath.Join(choice.batchDir, filepath.FromSlash(wantValidation)))
	if err != nil {
		return nil, fmt.Errorf("%s: read selected validation: %w", unit.ID, err)
	}
	candidate, err := os.ReadFile(filepath.Join(choice.batchDir, filepath.FromSlash(wantCandidate)))
	if err != nil {
		return nil, fmt.Errorf("%s: read selected candidate: %w", unit.ID, err)
	}
	attempt, err := validateUnifiedPromotionEvidence(root, choice.batchDir, catalog, unit, choice.manifest, validation, glossary, candidate, locale)
	if err != nil {
		return nil, err
	}
	return &validatedRetranslationCandidate{
		candidatePath: wantCandidate, validationPath: wantValidation,
		candidate: candidate, validationData: validationData,
		validation: validation, attempt: attempt,
	}, nil
}

func selectedRetranslationIdentityMatches(unit *TranslationUnit, choice selectedRetranslationUnit) bool {
	return choice.manifest.UnitID == unit.ID && choice.manifest.UnitKind == unit.Kind &&
		choice.manifest.SourcePath == unit.SourcePath && choice.manifest.SourceSHA256 == unit.SourceSHA256
}
