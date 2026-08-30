package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	retranslationProcessSchemaVersion = 2
	retranslationArtifactEOFSingleLF  = "single_lf"
)

// canonicalizeRetranslationArtifactEOF gives text artifacts exactly one final
// LF without changing any non-EOF bytes.
func canonicalizeRetranslationArtifactEOF(data []byte) []byte {
	trimmed := bytes.TrimRight(data, "\n")
	out := make([]byte, len(trimmed)+1)
	copy(out, trimmed)
	out[len(out)-1] = '\n'
	return out
}

func validateRetranslationArtifactEOF(data []byte) error {
	if !bytes.Equal(data, canonicalizeRetranslationArtifactEOF(data)) {
		return fmt.Errorf("must end with exactly one LF and no blank line at EOF")
	}
	return nil
}

func supportedRetranslationArtifactEOFPolicy(policy string) bool {
	return policy == "" || policy == retranslationArtifactEOFSingleLF
}

type RetranslationProcessOptions struct {
	Locale  string
	BatchID string
}

type RetranslationUnitResult struct {
	UnitID         string   `json:"unit_id"`
	UnitKind       UnitKind `json:"unit_kind"`
	Status         string   `json:"status"`
	CandidatePath  string   `json:"candidate_path,omitempty"`
	ValidationPath string   `json:"validation_path"`
	Error          string   `json:"error,omitempty"`
}

type RetranslationProcessResult struct {
	SchemaVersion    int                       `json:"schema_version"`
	BatchID          string                    `json:"batch_id"`
	Locale           string                    `json:"locale"`
	UnitCount        int                       `json:"unit_count"`
	RestorePassed    int                       `json:"restore_passed"`
	RestoreFailed    int                       `json:"restore_failed"`
	ValidationPassed int                       `json:"validation_passed"`
	ValidationFailed int                       `json:"validation_failed"`
	Units            []RetranslationUnitResult `json:"units"`
	NoPendingBatches bool                      `json:"no_pending_batches,omitempty"`
}

type RetranslationValidation struct {
	SchemaVersion   int      `json:"schema_version"`
	BatchID         string   `json:"batch_id"`
	Locale          string   `json:"locale"`
	UnitID          string   `json:"unit_id"`
	UnitKind        UnitKind `json:"unit_kind"`
	SourceSHA256    string   `json:"source_sha256"`
	Attempt         int      `json:"attempt"`
	Status          string   `json:"status"`
	InputPath       string   `json:"input_path"`
	RawResponsePath string   `json:"raw_response_path"`
	CandidatePath   string   `json:"candidate_path,omitempty"`
	Error           string   `json:"error,omitempty"`
}

type preparedRetranslationPage struct {
	manifest  RetranslationBatchUnit
	unit      *TranslationUnit
	protected protectedTranslation
	raw       []byte
}

// ProcessRetranslationBatch restores protected raw responses into isolated
// batch candidates and validates them with the canonical candidate validator.
func ProcessRetranslationBatch(root string, catalog *Catalog, options RetranslationProcessOptions) (*RetranslationProcessResult, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if options.Locale == "" {
		return nil, errors.New("retranslation locale is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	base := filepath.Join(root, "data", "retranslation-runs", options.Locale)
	batchID, noPending, err := selectRetranslationProcessBatch(base, options.Locale, options.BatchID)
	if err != nil {
		return nil, err
	}
	if noPending {
		return &RetranslationProcessResult{Locale: options.Locale, NoPendingBatches: true}, nil
	}
	batchDir := filepath.Join(base, batchID)
	manifest, err := readRetranslationProcessManifest(batchDir, options.Locale, batchID)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(batchDir, "result.json")); err == nil {
		return nil, fmt.Errorf("retranslation batch %q is already processed", batchID)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect retranslation result: %w", err)
	}
	for _, name := range []string{"candidates", "validation"} {
		if _, err := os.Stat(filepath.Join(batchDir, name)); err == nil {
			return nil, fmt.Errorf("retranslation batch %q has incomplete existing process output %q", batchID, name)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect retranslation process output: %w", err)
		}
	}
	glossary, err := LoadGlossary(root, options.Locale)
	if err != nil {
		return nil, err
	}
	prepared, err := preflightRetranslationProcess(batchDir, catalog, glossary, manifest)
	if err != nil {
		return nil, err
	}

	staging, err := os.MkdirTemp(batchDir, ".process-staging-")
	if err != nil {
		return nil, fmt.Errorf("create retranslation process staging: %w", err)
	}
	defer os.RemoveAll(staging)
	if err := os.Mkdir(filepath.Join(staging, "candidates"), 0755); err != nil {
		return nil, err
	}
	if err := os.Mkdir(filepath.Join(staging, "validation"), 0755); err != nil {
		return nil, err
	}
	result := &RetranslationProcessResult{
		SchemaVersion: retranslationProcessSchemaVersion, BatchID: batchID, Locale: options.Locale,
		UnitCount: len(prepared), Units: make([]RetranslationUnitResult, 0, len(prepared)),
	}
	for _, item := range prepared {
		name := filepath.Base(filepath.FromSlash(item.manifest.InputPath))
		rawPath := filepath.ToSlash(filepath.Join("raw-responses", name))
		candidateName := retranslationUnitCandidateName(item.unit)
		candidatePath := filepath.ToSlash(filepath.Join("candidates", candidateName))
		validationPath := filepath.ToSlash(filepath.Join("validation", strings.TrimSuffix(name, filepath.Ext(name))+".json"))
		evidence := RetranslationValidation{
			SchemaVersion: retranslationProcessSchemaVersion, BatchID: batchID, Locale: options.Locale,
			UnitID: item.unit.ID, UnitKind: item.unit.Kind, SourceSHA256: item.unit.SourceSHA256,
			Attempt: 1, InputPath: item.manifest.InputPath, RawResponsePath: rawPath,
		}
		unitResult := RetranslationUnitResult{UnitID: item.unit.ID, UnitKind: item.unit.Kind, ValidationPath: validationPath}
		restored, failures := item.protected.restore(string(item.raw))
		if len(failures) != 0 {
			evidence.Status = "restore_failed"
			evidence.Error = strings.Join(failures, "; ")
			unitResult.Status = evidence.Status
			unitResult.Error = evidence.Error
			result.RestoreFailed++
		} else {
			result.RestorePassed++
			candidate := []byte(restored)
			if manifest.ArtifactEOF == retranslationArtifactEOFSingleLF {
				candidate = canonicalizeRetranslationArtifactEOF(candidate)
			}
			if err := os.WriteFile(filepath.Join(staging, "candidates", candidateName), candidate, 0644); err != nil {
				return nil, fmt.Errorf("write staged candidate for %s: %w", item.unit.ID, err)
			}
			evidence.CandidatePath = candidatePath
			unitResult.CandidatePath = candidatePath
			if err := ValidateTranslationUnitCandidate(root, catalog, item.unit.ID, options.Locale, candidate); err != nil {
				evidence.Status = "validation_failed"
				evidence.Error = err.Error()
				unitResult.Error = evidence.Error
				result.ValidationFailed++
			} else {
				evidence.Status = "passed"
				result.ValidationPassed++
			}
			unitResult.Status = evidence.Status
		}
		if err := writeTranslationJSON(filepath.Join(staging, filepath.FromSlash(validationPath)), evidence); err != nil {
			return nil, fmt.Errorf("write validation for %s: %w", item.unit.ID, err)
		}
		result.Units = append(result.Units, unitResult)
	}
	if err := writeTranslationJSON(filepath.Join(staging, "result.json"), result); err != nil {
		return nil, fmt.Errorf("write retranslation result: %w", err)
	}
	committed := []string{}
	rollback := func() {
		for i := len(committed) - 1; i >= 0; i-- {
			_ = os.RemoveAll(filepath.Join(batchDir, committed[i]))
		}
	}
	for _, name := range []string{"candidates", "validation", "result.json"} {
		if err := os.Rename(filepath.Join(staging, name), filepath.Join(batchDir, name)); err != nil {
			rollback()
			return nil, fmt.Errorf("commit retranslation process output %s: %w", name, err)
		}
		committed = append(committed, name)
	}
	return result, nil
}

func selectRetranslationProcessBatch(base, locale, explicit string) (string, bool, error) {
	if explicit != "" {
		if err := validateBatchID(explicit); err != nil {
			return "", false, err
		}
		if info, err := os.Stat(filepath.Join(base, explicit)); err != nil || !info.IsDir() {
			if err == nil {
				err = errors.New("not a directory")
			}
			return "", false, fmt.Errorf("inspect retranslation batch %q: %w", explicit, err)
		}
		return explicit, false, nil
	}
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return "", true, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("scan retranslation batches: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		batchDir := filepath.Join(base, entry.Name())
		if _, err := readRetranslationProcessManifest(batchDir, locale, entry.Name()); err != nil {
			return "", false, err
		}
		if _, err := os.Stat(filepath.Join(batchDir, "result.json")); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return "", false, err
		}
		return entry.Name(), false, nil
	}
	return "", true, nil
}

func readRetranslationProcessManifest(batchDir, locale, batchID string) (*RetranslationBatchManifest, error) {
	data, err := os.ReadFile(filepath.Join(batchDir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("read retranslation manifest for %q: %w", batchID, err)
	}
	manifest, err := decodeRetranslationManifest(data)
	if err != nil {
		return nil, fmt.Errorf("parse retranslation manifest for %q: %w", batchID, err)
	}
	if manifest.SchemaVersion != 2 || manifest.BatchID != batchID || manifest.Locale != locale || manifest.ProtectionMode != "default" || !supportedRetranslationArtifactEOFPolicy(manifest.ArtifactEOF) || (manifest.UnitKind != UnitKindPage && manifest.UnitKind != UnitKindExample) {
		return nil, fmt.Errorf("retranslation batch %q has incompatible manifest metadata", batchID)
	}
	if manifest.UnitCount < 1 || manifest.UnitCount != len(manifest.Units) {
		return nil, fmt.Errorf("retranslation batch %q unit_count %d does not match units %d", batchID, manifest.UnitCount, len(manifest.Units))
	}
	return manifest, nil
}

// legacyRetranslationManifest is the single compatibility boundary for the
// immutable Page-only Batch 001-011 audit evidence.
type legacyRetranslationManifest struct {
	SchemaVersion   int    `json:"schema_version"`
	BatchID         string `json:"batch_id"`
	Locale          string `json:"locale"`
	ProtectionMode  string `json:"protection_mode"`
	TranslationUnit string `json:"translation_unit"`
	PageCount       int    `json:"page_count"`
	Pages           []struct {
		PageID              string `json:"page_id"`
		Article             string `json:"article"`
		SourceSHA256        string `json:"source_sha256"`
		InputPath           string `json:"input_path"`
		InputSHA256         string `json:"input_sha256"`
		ProtectedTokenCount int    `json:"protected_token_count"`
	} `json:"pages"`
}

func decodeRetranslationManifest(data []byte) (*RetranslationBatchManifest, error) {
	var current RetranslationBatchManifest
	if err := json.Unmarshal(data, &current); err != nil {
		return nil, err
	}
	if current.SchemaVersion == 2 {
		return &current, nil
	}
	var legacy legacyRetranslationManifest
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}
	if legacy.SchemaVersion != 1 || legacy.TranslationUnit != "present.Section" || legacy.PageCount != len(legacy.Pages) {
		return nil, errors.New("unsupported legacy retranslation manifest")
	}
	normalized := &RetranslationBatchManifest{
		SchemaVersion: 2, BatchID: legacy.BatchID, Locale: legacy.Locale,
		ProtectionMode: legacy.ProtectionMode, UnitKind: UnitKindPage,
		UnitCount: legacy.PageCount, Units: make([]RetranslationBatchUnit, 0, legacy.PageCount),
	}
	for _, page := range legacy.Pages {
		normalized.Units = append(normalized.Units, RetranslationBatchUnit{
			UnitID: page.PageID, UnitKind: UnitKindPage,
			SourcePath:   filepath.ToSlash(filepath.Join("_content", "tour", page.Article)),
			SourceSHA256: page.SourceSHA256, InputPath: page.InputPath,
			InputSHA256: page.InputSHA256, ProtectedTokenCount: page.ProtectedTokenCount,
		})
	}
	return normalized, nil
}

type legacyRetranslationProcessResult struct {
	SchemaVersion    int    `json:"schema_version"`
	BatchID          string `json:"batch_id"`
	Locale           string `json:"locale"`
	PageCount        int    `json:"page_count"`
	RestorePassed    int    `json:"restore_passed"`
	RestoreFailed    int    `json:"restore_failed"`
	ValidationPassed int    `json:"validation_passed"`
	ValidationFailed int    `json:"validation_failed"`
	Pages            []struct {
		PageID         string `json:"page_id"`
		Status         string `json:"status"`
		CandidatePath  string `json:"candidate_path"`
		ValidationPath string `json:"validation_path"`
	} `json:"pages"`
}

func decodeRetranslationProcessResult(data []byte) (*RetranslationProcessResult, error) {
	var current RetranslationProcessResult
	if err := json.Unmarshal(data, &current); err != nil {
		return nil, err
	}
	if current.SchemaVersion == retranslationProcessSchemaVersion {
		return &current, nil
	}
	var legacy legacyRetranslationProcessResult
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}
	if legacy.SchemaVersion != 1 || legacy.PageCount != len(legacy.Pages) {
		return nil, errors.New("unsupported legacy retranslation process result")
	}
	normalized := &RetranslationProcessResult{
		SchemaVersion: retranslationProcessSchemaVersion, BatchID: legacy.BatchID, Locale: legacy.Locale,
		UnitCount: legacy.PageCount, RestorePassed: legacy.RestorePassed, RestoreFailed: legacy.RestoreFailed,
		ValidationPassed: legacy.ValidationPassed, ValidationFailed: legacy.ValidationFailed,
		Units: make([]RetranslationUnitResult, 0, legacy.PageCount),
	}
	for _, page := range legacy.Pages {
		normalized.Units = append(normalized.Units, RetranslationUnitResult{
			UnitID: page.PageID, UnitKind: UnitKindPage, Status: page.Status,
			CandidatePath: page.CandidatePath, ValidationPath: page.ValidationPath,
		})
	}
	return normalized, nil
}

type legacyRetranslationValidation struct {
	SchemaVersion   int    `json:"schema_version"`
	BatchID         string `json:"batch_id"`
	Locale          string `json:"locale"`
	PageID          string `json:"page_id"`
	Status          string `json:"status"`
	InputPath       string `json:"input_path"`
	RawResponsePath string `json:"raw_response_path"`
	CandidatePath   string `json:"candidate_path"`
	Error           string `json:"error"`
}

func decodeRetranslationValidation(data []byte, unit *TranslationUnit) (*RetranslationValidation, error) {
	var current RetranslationValidation
	if err := json.Unmarshal(data, &current); err != nil {
		return nil, err
	}
	if current.SchemaVersion == retranslationProcessSchemaVersion {
		return &current, nil
	}
	var legacy legacyRetranslationValidation
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}
	if legacy.SchemaVersion != 1 || unit == nil || legacy.PageID != unit.ID || unit.Kind != UnitKindPage {
		return nil, errors.New("unsupported legacy retranslation validation")
	}
	name := retranslationUnitInputName(unit)
	extension := filepath.Ext(name)
	flatID := strings.TrimSuffix(name, extension)
	attempt, err := retryValidationAttemptForExtension(legacy.RawResponsePath, flatID, extension)
	if err != nil {
		return nil, fmt.Errorf("legacy retranslation validation: %w", err)
	}
	return &RetranslationValidation{
		SchemaVersion: retranslationProcessSchemaVersion, BatchID: legacy.BatchID, Locale: legacy.Locale,
		UnitID: unit.ID, UnitKind: unit.Kind, SourceSHA256: unit.SourceSHA256, Attempt: attempt,
		Status: legacy.Status, InputPath: legacy.InputPath, RawResponsePath: legacy.RawResponsePath,
		CandidatePath: legacy.CandidatePath, Error: legacy.Error,
	}, nil
}

func preflightRetranslationProcess(batchDir string, catalog *Catalog, glossary *Glossary, manifest *RetranslationBatchManifest) ([]preparedRetranslationPage, error) {
	rawDir := filepath.Join(batchDir, "raw-responses")
	entries, err := os.ReadDir(rawDir)
	if err != nil {
		return nil, fmt.Errorf("read raw responses: %w", err)
	}
	expectedRaw := make(map[string]bool, len(manifest.Units))
	seenUnits := map[string]bool{}
	prepared := make([]preparedRetranslationPage, 0, len(manifest.Units))
	for _, record := range manifest.Units {
		unitID, unitKind := record.UnitID, record.UnitKind
		if unitID == "" {
			return nil, errors.New("retranslation manifest has empty translation unit id")
		}
		if seenUnits[unitID] {
			return nil, fmt.Errorf("duplicate manifest translation unit %q", unitID)
		}
		seenUnits[unitID] = true
		unit, err := catalog.Unit(unitID)
		if err != nil {
			return nil, fmt.Errorf("manifest translation unit %q: %w", unitID, err)
		}
		if unit.Kind != unitKind || manifest.UnitKind != unitKind {
			return nil, fmt.Errorf("%s: manifest unit_kind %q does not match Catalog kind %q", unitID, unitKind, unit.Kind)
		}
		if unit.SourceSHA256 != record.SourceSHA256 || sum(unit.Source) != record.SourceSHA256 {
			return nil, fmt.Errorf("%s: manifest source metadata does not match current Catalog", unitID)
		}
		if record.SourcePath != unit.SourcePath {
			return nil, fmt.Errorf("%s: manifest source_path %q does not match %q", unitID, record.SourcePath, unit.SourcePath)
		}
		name := filepath.Base(filepath.FromSlash(record.InputPath))
		wantInputPath := filepath.ToSlash(filepath.Join("inputs", retranslationUnitInputName(unit)))
		if name == "." || record.InputPath != wantInputPath {
			return nil, fmt.Errorf("%s: unsafe or non-canonical input_path %q", unitID, record.InputPath)
		}
		input, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(record.InputPath)))
		if err != nil {
			return nil, fmt.Errorf("%s: read saved input: %w", unitID, err)
		}
		if sum(input) != record.InputSHA256 {
			return nil, fmt.Errorf("%s: input_sha256 mismatch", unitID)
		}
		protected, err := prepareTranslationUnitInput(unit, glossary)
		if err != nil {
			return nil, fmt.Errorf("%s: prepare protected input: %w", unitID, err)
		}
		expectedInput := []byte(protected.Text)
		if manifest.ArtifactEOF == retranslationArtifactEOFSingleLF {
			expectedInput = canonicalizeRetranslationArtifactEOF(expectedInput)
		}
		if !bytes.Equal(expectedInput, input) {
			if unit.Kind == UnitKindPage {
				return nil, fmt.Errorf("%s: regenerated Default protected input differs from saved input", unitID)
			}
			return nil, fmt.Errorf("%s: regenerated protected input differs from saved input", unitID)
		}
		if len(protected.Tokens) != record.ProtectedTokenCount {
			return nil, fmt.Errorf("%s: protected_token_count %d, regenerated %d", unitID, record.ProtectedTokenCount, len(protected.Tokens))
		}
		rawName := name
		expectedRaw[rawName] = true
		raw, err := os.ReadFile(filepath.Join(rawDir, rawName))
		if err != nil {
			return nil, fmt.Errorf("%s: read raw response: %w", unitID, err)
		}
		if manifest.ArtifactEOF == retranslationArtifactEOFSingleLF {
			if err := validateRetranslationArtifactEOF(raw); err != nil {
				return nil, fmt.Errorf("%s: raw response %s %w", unitID, rawName, err)
			}
		}
		prepared = append(prepared, preparedRetranslationPage{manifest: record, unit: unit, protected: protected, raw: raw})
	}
	actual := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("unexpected raw response directory %q", entry.Name())
		}
		if actual[entry.Name()] {
			return nil, fmt.Errorf("duplicate raw response %q", entry.Name())
		}
		actual[entry.Name()] = true
		if !expectedRaw[entry.Name()] {
			return nil, fmt.Errorf("unexpected raw response %q", entry.Name())
		}
	}
	if len(actual) != len(expectedRaw) {
		return nil, fmt.Errorf("raw response count %d, want %d", len(actual), len(expectedRaw))
	}
	return prepared, nil
}
