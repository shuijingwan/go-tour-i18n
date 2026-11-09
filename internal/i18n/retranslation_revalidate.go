package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type RetranslationRevalidateOptions struct {
	Locale  string
	BatchID string
	UnitID  string
}

type RetranslationRevalidationResult struct {
	SchemaVersion     int      `json:"schema_version"`
	BatchID           string   `json:"batch_id"`
	Locale            string   `json:"locale"`
	UnitID            string   `json:"unit_id"`
	UnitKind          UnitKind `json:"unit_kind"`
	Attempt           int      `json:"attempt"`
	Revalidation      int      `json:"revalidation"`
	PreviousStatus    string   `json:"previous_status"`
	Status            string   `json:"status"`
	Error             string   `json:"error,omitempty"`
	HistoryPath       string   `json:"history_path"`
	ValidationPath    string   `json:"validation_path"`
	ResultPath        string   `json:"result_path"`
	ResultHistoryPath string   `json:"result_history_path,omitempty"`
	RestorePassed     int      `json:"restore_passed"`
	RestoreFailed     int      `json:"restore_failed"`
	ValidationPassed  int      `json:"validation_passed"`
	ValidationFailed  int      `json:"validation_failed"`
}

// RevalidateRetranslationCandidate runs the current canonical validator over an
// already restored candidate, or restores a failed Unit using the exact original
// imported invocation after an adapter fix. It never generates language or
// changes the raw response/attempt. Previous evidence remains immutable history.
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
	if err := RequireCurrentGlossaryReview(root, options.Locale); err != nil {
		return nil, err
	}
	if err := validateBatchID(options.BatchID); err != nil {
		return nil, err
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", options.Locale, options.BatchID)
	if err := ensureNoGenerationImportPending(batchDir); err != nil {
		return nil, err
	}
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
	// An ordinary current-validation successor is validated as current bytes.
	// Installed approved transport still requires exact decoder/provenance
	// preflight. Other feedback and restore_failed Units also keep that path.
	currentOnly := false
	for _, r := range result.Units {
		if r.UnitID == options.UnitID && (r.Status == "passed" || r.Status == "validation_failed") && manifest.TransportRecovery == nil {
			for _, record := range manifest.Units {
				currentOnly = currentOnly || (record.UnitID == options.UnitID && record.RevisionFeedbackSource == CurrentValidationRevisionFeedback && record.RepresentationBridge == nil)
			}
		}
	}
	if currentOnly {
		approved, err := installedApprovedRevalidationTransport(root, catalog, manifest)
		if err != nil {
			return nil, err
		}
		currentOnly = !approved
	}
	var prepared []preparedRetranslationPage
	if currentOnly {
		if err := validateCurrentValidationCandidateResult(manifest, &result); err != nil {
			return nil, err
		}
		refs, checkErr := currentValidationCandidateLineage(root, catalog, manifest)
		if checkErr != nil {
			return nil, checkErr
		}
		review, err := RequireUnifiedGlossaryReview(root, options.Locale)
		if err != nil {
			return nil, err
		}
		for _, record := range manifest.Units {
			if record.UnitID != options.UnitID {
				continue
			}
			u, err := catalog.Unit(record.UnitID)
			if err != nil {
				return nil, err
			}
			identity, err := currentTranslationAuthority(root, u, options.Locale)
			if err != nil {
				return nil, err
			}
			prepared = append(prepared, preparedRetranslationPage{unit: u, manifest: record, checkOutputContext: func() error {
				if approved, err := installedApprovedRevalidationTransport(root, catalog, manifest); err != nil || approved {
					return fmt.Errorf("current revalidation installed decoder context changed: %v", err)
				}
				fresh, err := currentValidationCandidateLineage(root, catalog, manifest)
				if err != nil || !reflect.DeepEqual(fresh, refs) {
					return fmt.Errorf("current revalidation lineage changed: %v", err)
				}
				got, err := currentTranslationAuthority(root, u, options.Locale)
				if err != nil || got != identity {
					return fmt.Errorf("current revalidation validator authority changed")
				}
				freshGlossary, err := LoadGlossary(root, options.Locale)
				if err != nil || !reflect.DeepEqual(glossary, freshGlossary) {
					return fmt.Errorf("current revalidation glossary changed")
				}
				freshReview, err := RequireUnifiedGlossaryReview(root, options.Locale)
				if err != nil || !reflect.DeepEqual(review, freshReview) {
					return fmt.Errorf("current revalidation full glossary Review changed")
				}
				return nil
			}})
		}
	} else {
		prepared, err = preflightRetranslationProcess(batchDir, catalog, glossary, manifest)
	}
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
	if currentOnly {
		base, err := repositoryRelativePath(root, batchDir)
		if err != nil {
			return nil, err
		}
		input, err := readSnapshotReferencedFile(root, base+"/"+item.manifest.InputPath)
		if err != nil || sum(input) != item.manifest.InputSHA256 {
			return nil, fmt.Errorf("current revalidation saved input hash mismatch")
		}
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
	recoverRestore := unitResult.Status == "restore_failed"
	if !recoverRestore && unitResult.Status != "validation_failed" && unitResult.Status != "passed" {
		return nil, fmt.Errorf("translation unit %q status %q is not revalidatable", options.UnitID, unitResult.Status)
	}
	name := filepath.Base(filepath.FromSlash(item.manifest.InputPath))
	extension := filepath.Ext(name)
	flatID := strings.TrimSuffix(name, extension)
	wantCandidate := filepath.ToSlash(filepath.Join("candidates", retranslationUnitCandidateName(item.unit)))
	wantValidation := filepath.ToSlash(filepath.Join("validation", flatID+".json"))
	existingCandidate := wantCandidate
	if recoverRestore {
		existingCandidate = ""
	}
	if unitResult.CandidatePath != existingCandidate || unitResult.ValidationPath != wantValidation || unitResult.UnitKind != item.unit.Kind {
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
	if current.SchemaVersion != retranslationProcessSchemaVersion || current.BatchID != options.BatchID || current.Locale != options.Locale || current.UnitID != options.UnitID || current.UnitKind != item.unit.Kind || current.SourceSHA256 != item.unit.SourceSHA256 || current.InputPath != item.manifest.InputPath || current.CandidatePath != existingCandidate || current.Status != unitResult.Status || current.Error != unitResult.Error {
		return nil, fmt.Errorf("current validation for %s does not match manifest and result.json", options.UnitID)
	}
	attempt, err := retryValidationAttemptForExtension(current.RawResponsePath, flatID, extension)
	if err != nil || current.Attempt != attempt {
		return nil, fmt.Errorf("%s: validation attempt provenance mismatch", options.UnitID)
	}
	candidatePath := filepath.Join(batchDir, filepath.FromSlash(wantCandidate))
	var candidate []byte
	if recoverRestore {
		if _, err := os.Lstat(candidatePath); !os.IsNotExist(err) {
			return nil, fmt.Errorf("restore recovery candidate already exists or cannot be inspected")
		}
		if current.Error == "" {
			return nil, fmt.Errorf("restore failure evidence is required")
		}
	} else {
		candidate, err = os.ReadFile(candidatePath)
		if err != nil {
			return nil, fmt.Errorf("read restored candidate for %s: %w", options.UnitID, err)
		}
		if current.CandidateSHA256 != "" && current.CandidateSHA256 != sum(candidate) {
			return nil, fmt.Errorf("%s: current validated candidate SHA mismatch", options.UnitID)
		}
	}
	raw, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(current.RawResponsePath)))
	if err != nil {
		return nil, fmt.Errorf("read validation raw response for %s: %w", options.UnitID, err)
	}
	if manifest.ArtifactEOF == retranslationArtifactEOFSingleLF {
		if err := validateRetranslationArtifactEOF(raw); err != nil {
			return nil, err
		}
	}
	if currentOnly {
		// Reuse the invocation-local pre-commit check for the bytes just read;
		// a concurrent candidate/evidence/input replacement cannot be sealed by
		// this validation. No persistent checkpoint or new CAS format is added.
		base, err := repositoryRelativePath(root, batchDir)
		if err != nil {
			return nil, err
		}
		refs := []GlossaryArchiveReference{}
		for path, sha := range map[string]string{
			base + "/manifest.json": "", base + "/result.json": sum(resultData),
			base + "/" + item.manifest.InputPath: item.manifest.InputSHA256,
			base + "/" + wantValidation:          sum(validationData),
			base + "/" + wantCandidate:           sum(candidate),
			base + "/" + current.RawResponsePath: sum(raw),
		} {
			ref, err := successorReference(root, path)
			if err != nil || (sha != "" && ref.SHA256 != sha) {
				return nil, fmt.Errorf("current revalidation evidence changed: %s", path)
			}
			refs = append(refs, ref)
		}
		fresh, err := readRetranslationProcessManifest(batchDir, options.Locale, options.BatchID)
		if err != nil || !reflect.DeepEqual(manifest, fresh) {
			return nil, fmt.Errorf("current revalidation manifest changed")
		}
		lineageCheck := prepared[0].checkOutputContext
		prepared[0].checkOutputContext = func() error {
			if err := lineageCheck(); err != nil {
				return err
			}
			for _, ref := range refs {
				if _, err := readCompatibilityReference(root, ref); err != nil {
					return err
				}
			}
			return nil
		}
	}
	if recoverRestore {
		if err := verifyRestoreRecoveryInvocation(root, batchDir, manifest, *item, current, raw); err != nil {
			return nil, err
		}
	}
	if !currentOnly {
		restored, failures := item.protected.restore(string(raw))
		restoredBytes := []byte(restored)
		if len(failures) != 0 {
			return nil, fmt.Errorf("%s: current restore failed: %s", options.UnitID, strings.Join(failures, "; "))
		}
		if recoverRestore {
			candidate = restoredBytes
			if manifest.ArtifactEOF == retranslationArtifactEOFSingleLF {
				candidate = canonicalizeRetranslationArtifactEOF(candidate)
			}
		} else if !bytes.Equal(restoredBytes, candidate) && !bytes.Equal(canonicalizeRetranslationArtifactEOF(restoredBytes), candidate) {
			return nil, fmt.Errorf("%s: restored candidate does not match saved candidate", options.UnitID)
		}
	}
	number, historyRelative, err := nextRevalidationHistory(batchDir, flatID)
	if err != nil {
		return nil, err
	}
	previousStatus := current.Status
	current.CandidatePath, unitResult.CandidatePath = wantCandidate, wantCandidate
	current.Error = ""
	current.CandidateSHA256 = ""
	if err := ValidateTranslationUnitCandidate(root, catalog, options.UnitID, options.Locale, candidate); err != nil {
		current.Status, current.Error = "validation_failed", err.Error()
	} else {
		current.Status = "passed"
		current.CandidateSHA256 = sum(candidate)
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
	updates := []retranslationFileUpdate{{target: historyPath, data: validationData, requireMissing: true}}
	resultHistory := ""
	if recoverRestore {
		resultHistory = strings.TrimSuffix(historyRelative, "validation.json") + "result.json"
		updates = append(updates, retranslationFileUpdate{target: filepath.Join(batchDir, filepath.FromSlash(resultHistory)), data: resultData, requireMissing: true}, retranslationFileUpdate{target: candidatePath, data: candidate, requireMissing: true})
	}
	updates = append(updates, retranslationFileUpdate{target: validationPath, data: newValidation}, retranslationFileUpdate{target: resultPath, data: newResult})
	if err := checkPreparedRetranslationContext(prepared); err != nil {
		return nil, err
	}
	if err := commitRetranslationFileUpdates(batchDir, updates); err != nil {
		return nil, err
	}
	return &RetranslationRevalidationResult{SchemaVersion: 1, BatchID: options.BatchID, Locale: options.Locale, UnitID: options.UnitID, UnitKind: item.unit.Kind, Attempt: attempt, Revalidation: number, PreviousStatus: previousStatus, Status: current.Status, Error: current.Error, HistoryPath: historyRelative, ResultHistoryPath: resultHistory, ValidationPath: wantValidation, ResultPath: "result.json", RestorePassed: result.RestorePassed, RestoreFailed: result.RestoreFailed, ValidationPassed: result.ValidationPassed, ValidationFailed: result.ValidationFailed}, nil
}

// Use the archived imported bundle's semantic inputs and output hashes, not
// today's validator code as Generation authority. An adapter fix may change
// authority/*; source, input, glossary, invocation and raw bytes must not drift.
func verifyRestoreRecoveryInvocation(root, batchDir string, manifest *RetranslationBatchManifest, item preparedRetranslationPage, current RetranslationValidation, raw []byte) error {
	if item.manifest.GenerationOrigin != nil {
		return verifyGenerationOrigin(root, manifest.Locale, item.unit, item.manifest, raw)
	}
	gm := GenerationBundleManifest{Attempt: current.Attempt}
	if current.Attempt > 1 {
		gm.TaskKind = "translation-unit-retry"
		gm.ExpectedOutputs = []GenerationBundleExpectedOutput{{BundlePath: "outputs/" + filepath.Base(item.manifest.InputPath)}}
	}
	b, err := os.ReadFile(generationImportReceiptPath(batchDir, gm))
	if err != nil {
		return fmt.Errorf("restore recovery requires original import provenance: %w", err)
	}
	var receipt GenerationResultBundleManifest
	if err := contentidentity.StrictJSON(b, &receipt); err != nil {
		return err
	}
	if !validSHA256(receipt.GenerationBundleSHA256) {
		return fmt.Errorf("restore recovery invalid Generation archive identity")
	}
	bundle, err := readCompatibilityFile(root, importedGenerationArchive(manifest.Locale, receipt.GenerationBundleSHA256))
	if err != nil {
		return err
	}
	original, files, err := readGenerationBundle(bundle)
	if err != nil {
		return err
	}
	if err := checkImportedGenerationIdentity(receipt, original, bundle, files); err != nil {
		return err
	}
	if original.Locale != manifest.Locale || original.BatchID != manifest.BatchID || original.Attempt != current.Attempt || original.UnitKind != item.unit.Kind {
		return fmt.Errorf("restore recovery invocation/attempt identity mismatch")
	}
	for _, f := range original.Files {
		if f.RepositoryPath == "" || strings.HasPrefix(f.BundlePath, "authority/") || strings.HasPrefix(f.BundlePath, "formal/retry/") {
			continue
		}
		currentBytes, err := readCompatibilityFile(root, f.RepositoryPath)
		if err != nil || !bytes.Equal(currentBytes, files[f.BundlePath]) {
			return fmt.Errorf("restore recovery stale formal input: %s", f.BundlePath)
		}
	}
	outputs := map[string]TransportBundleFile{}
	for _, p := range receipt.Outputs {
		if _, exists := outputs[p.BundlePath]; exists || !validSHA256(p.SHA256) || p.Size < 1 {
			return fmt.Errorf("restore recovery invalid output provenance")
		}
		outputs[p.BundlePath] = p
	}
	matched := false
	for _, output := range original.ExpectedOutputs {
		p, exists := outputs[output.BundlePath]
		if !exists {
			return fmt.Errorf("restore recovery output provenance exact-set mismatch")
		}
		if output.UnitID == item.unit.ID && output.InstallPath == current.RawResponsePath {
			if p.SHA256 == sum(raw) && p.Size == int64(len(raw)) {
				matched = true
			}
		}
	}
	if matched {
		return nil
	}
	return fmt.Errorf("restore recovery raw response/import provenance mismatch")
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
		if strings.HasSuffix(entry.Name(), "-result.json") {
			validationName := strings.TrimSuffix(entry.Name(), "result.json") + "validation.json"
			if !re.MatchString(validationName) {
				return 0, "", fmt.Errorf("unexpected revalidation history file %q", entry.Name())
			}
			if _, err := os.Stat(filepath.Join(dir, validationName)); err != nil {
				return 0, "", fmt.Errorf("result history lacks validation history: %w", err)
			}
			continue
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

// Classification only: the existing strict preflight owns approved decoding.
// Read the installed archive once at preparation and again before publication;
// malformed/unknown inventory must never acquire current-only eligibility.
func installedApprovedRevalidationTransport(root string, catalog *Catalog, manifest *RetranslationBatchManifest) (bool, error) {
	selection, files, err := importedApprovedIntervalSelection(root, catalog, manifest)
	if err != nil {
		return false, err
	}
	if _, declared := files[approvedIntervalSelectionMember]; declared && selection == nil {
		return false, fmt.Errorf("installed approved interval selection is empty or invalid")
	}
	if selection != nil {
		return true, nil
	}
	for path := range files {
		if strings.HasPrefix(path, "formal/approved-interval-transport/") {
			return true, nil
		}
	}
	return false, nil
}
