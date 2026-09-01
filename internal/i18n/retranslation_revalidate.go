package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type RetranslationRevalidateOptions struct {
	Locale  string
	BatchID string
	UnitID  string
}

type RetranslationRevalidationResult struct {
	SchemaVersion    int      `json:"schema_version"`
	BatchID          string   `json:"batch_id"`
	Locale           string   `json:"locale"`
	UnitID           string   `json:"unit_id"`
	UnitKind         UnitKind `json:"unit_kind"`
	Attempt          int      `json:"attempt"`
	Revalidation     int      `json:"revalidation"`
	PreviousStatus   string   `json:"previous_status"`
	Status           string   `json:"status"`
	Error            string   `json:"error,omitempty"`
	HistoryPath      string   `json:"history_path"`
	ValidationPath   string   `json:"validation_path"`
	ResultPath       string   `json:"result_path"`
	ValidationPassed int      `json:"validation_passed"`
	ValidationFailed int      `json:"validation_failed"`
}

// RevalidateRetranslationCandidate runs the current canonical validator over an
// already restored candidate. It archives validation evidence but never changes
// the raw response, candidate, or translation attempt.
func RevalidateRetranslationCandidate(root string, catalog *Catalog, options RetranslationRevalidateOptions) (*RetranslationRevalidationResult, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if options.Locale == "" || options.BatchID == "" || options.UnitID == "" {
		return nil, errors.New("retranslation revalidate locale, batch_id, and unit_id are required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	if err := validateBatchID(options.BatchID); err != nil {
		return nil, err
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", options.Locale, options.BatchID)
	manifest, err := readRetranslationProcessManifest(batchDir, options.Locale, options.BatchID)
	if err != nil {
		return nil, err
	}
	resultPath := filepath.Join(batchDir, "result.json")
	resultData, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, fmt.Errorf("read retranslation result for %q: %w", options.BatchID, err)
	}
	resultPtr, err := decodeRetranslationProcessResult(resultData)
	if err != nil {
		return nil, fmt.Errorf("parse retranslation result for %q: %w", options.BatchID, err)
	}
	result := *resultPtr
	if result.SchemaVersion != retranslationProcessSchemaVersion || result.BatchID != options.BatchID || result.Locale != options.Locale || result.UnitCount != len(result.Units) || result.UnitCount != manifest.UnitCount {
		return nil, fmt.Errorf("retranslation batch %q has incompatible process result", options.BatchID)
	}
	glossary, err := LoadGlossary(root, options.Locale)
	if err != nil {
		return nil, err
	}
	prepared, err := preflightRetranslationProcess(batchDir, catalog, glossary, manifest)
	if err != nil {
		return nil, err
	}
	var item *preparedRetranslationPage
	for i := range prepared {
		if prepared[i].unit.ID == options.UnitID {
			item = &prepared[i]
			break
		}
	}
	if item == nil {
		return nil, fmt.Errorf("translation unit %q is not in retranslation batch %q", options.UnitID, options.BatchID)
	}
	resultIndex := -1
	for i := range result.Units {
		if result.Units[i].UnitID == options.UnitID {
			if resultIndex != -1 {
				return nil, fmt.Errorf("duplicate result translation unit %q", options.UnitID)
			}
			resultIndex = i
		}
	}
	if resultIndex == -1 {
		return nil, fmt.Errorf("translation unit %q is not in retranslation result", options.UnitID)
	}
	unitResult := &result.Units[resultIndex]
	if unitResult.Status == "restore_failed" || unitResult.CandidatePath == "" {
		return nil, fmt.Errorf("translation unit %q does not have a restored candidate", options.UnitID)
	}
	if unitResult.Status != "validation_failed" && unitResult.Status != "passed" {
		return nil, fmt.Errorf("translation unit %q status %q is not revalidatable", options.UnitID, unitResult.Status)
	}
	name := filepath.Base(filepath.FromSlash(item.manifest.InputPath))
	extension := filepath.Ext(name)
	flatID := strings.TrimSuffix(name, extension)
	wantCandidate := filepath.ToSlash(filepath.Join("candidates", retranslationUnitCandidateName(item.unit)))
	wantValidation := filepath.ToSlash(filepath.Join("validation", flatID+".json"))
	if unitResult.CandidatePath != wantCandidate || unitResult.ValidationPath != wantValidation {
		return nil, fmt.Errorf("translation unit %q result candidate/validation path mismatch", options.UnitID)
	}
	validationPath := filepath.Join(batchDir, filepath.FromSlash(wantValidation))
	validationData, err := os.ReadFile(validationPath)
	if err != nil {
		return nil, fmt.Errorf("read current validation for %s: %w", options.UnitID, err)
	}
	currentPtr, err := decodeRetranslationValidation(validationData, item.unit)
	if err != nil {
		return nil, fmt.Errorf("parse current validation for %s: %w", options.UnitID, err)
	}
	current := *currentPtr
	if current.SchemaVersion != retranslationProcessSchemaVersion || current.BatchID != options.BatchID || current.Locale != options.Locale || current.UnitID != options.UnitID || current.UnitKind != item.unit.Kind || current.SourceSHA256 != item.unit.SourceSHA256 || current.InputPath != item.manifest.InputPath || current.CandidatePath != wantCandidate || current.Status != unitResult.Status {
		return nil, fmt.Errorf("current validation for %s does not match manifest and result.json", options.UnitID)
	}
	attempt, err := retryValidationAttemptForExtension(current.RawResponsePath, flatID, extension)
	if err != nil || current.Attempt != attempt {
		return nil, fmt.Errorf("%s: validation attempt provenance mismatch", options.UnitID)
	}
	candidatePath := filepath.Join(batchDir, filepath.FromSlash(wantCandidate))
	candidate, err := os.ReadFile(candidatePath)
	if err != nil {
		return nil, fmt.Errorf("read restored candidate for %s: %w", options.UnitID, err)
	}
	raw, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(current.RawResponsePath)))
	if err != nil {
		return nil, fmt.Errorf("read validation raw response for %s: %w", options.UnitID, err)
	}
	restored, failures := item.protected.restore(string(raw))
	restoredBytes := []byte(restored)
	if len(failures) != 0 || (!bytes.Equal(restoredBytes, candidate) && !bytes.Equal(canonicalizeRetranslationArtifactEOF(restoredBytes), candidate)) {
		return nil, fmt.Errorf("%s: restored candidate does not match saved candidate", options.UnitID)
	}
	number, historyRelative, err := nextRevalidationHistory(batchDir, flatID)
	if err != nil {
		return nil, err
	}
	previousStatus := current.Status
	current.Error = ""
	if err := ValidateTranslationUnitCandidate(root, catalog, options.UnitID, options.Locale, candidate); err != nil {
		current.Status, current.Error = "validation_failed", err.Error()
	} else {
		current.Status = "passed"
	}
	unitResult.Status, unitResult.Error = current.Status, current.Error
	recountRetranslationResult(&result)
	newValidation, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return nil, err
	}
	newValidation = append(newValidation, '\n')
	newResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	newResult = append(newResult, '\n')
	historyPath := filepath.Join(batchDir, filepath.FromSlash(historyRelative))
	if err := commitRetranslationFileUpdates(batchDir, []retranslationFileUpdate{{target: historyPath, data: validationData, requireMissing: true}, {target: validationPath, data: newValidation}, {target: resultPath, data: newResult}}); err != nil {
		return nil, err
	}
	return &RetranslationRevalidationResult{SchemaVersion: 1, BatchID: options.BatchID, Locale: options.Locale, UnitID: options.UnitID, UnitKind: item.unit.Kind, Attempt: attempt, Revalidation: number, PreviousStatus: previousStatus, Status: current.Status, Error: current.Error, HistoryPath: historyRelative, ValidationPath: wantValidation, ResultPath: "result.json", ValidationPassed: result.ValidationPassed, ValidationFailed: result.ValidationFailed}, nil
}

func nextRevalidationHistory(batchDir, flatID string) (int, string, error) {
	dirRelative := filepath.ToSlash(filepath.Join("revalidation-history", flatID))
	dir := filepath.Join(batchDir, filepath.FromSlash(dirRelative))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 1, filepath.ToSlash(filepath.Join(dirRelative, "revalidation-001-validation.json")), nil
	}
	if err != nil {
		return 0, "", fmt.Errorf("read revalidation history: %w", err)
	}
	re := regexp.MustCompile(`^revalidation-([0-9]{3})-validation\.json$`)
	found := map[int]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			return 0, "", fmt.Errorf("unexpected revalidation history directory %q", entry.Name())
		}
		m := re.FindStringSubmatch(entry.Name())
		if m == nil {
			return 0, "", fmt.Errorf("unexpected revalidation history file %q", entry.Name())
		}
		n, _ := strconv.Atoi(m[1])
		if n < 1 || found[n] {
			return 0, "", fmt.Errorf("invalid revalidation history sequence")
		}
		found[n] = true
	}
	for n := 1; n <= len(found); n++ {
		if !found[n] {
			return 0, "", fmt.Errorf("missing revalidation history %03d", n)
		}
	}
	next := len(found) + 1
	return next, filepath.ToSlash(filepath.Join(dirRelative, fmt.Sprintf("revalidation-%03d-validation.json", next))), nil
}
