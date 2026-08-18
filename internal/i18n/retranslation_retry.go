package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type RetranslationRetryOptions struct {
	Locale  string
	BatchID string
	PageID  string
}

var retryRawPathRE = regexp.MustCompile(`^retries/([^/]+)/attempt-([0-9]{3})\.article$`)

// ProcessRetranslationRetry processes the next pre-existing protected retry
// response for one failed page. It never writes a raw response itself.
func ProcessRetranslationRetry(root string, catalog *Catalog, options RetranslationRetryOptions) (*RetranslationProcessResult, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if options.Locale == "" || options.BatchID == "" || options.PageID == "" {
		return nil, errors.New("retranslation retry locale, batch_id, and page_id are required")
	}
	if options.Locale != "zh-CN" {
		return nil, fmt.Errorf("unsupported locale %q", options.Locale)
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
	var result RetranslationProcessResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("parse retranslation result for %q: %w", options.BatchID, err)
	}
	if result.SchemaVersion != retranslationProcessSchemaVersion || result.BatchID != options.BatchID || result.Locale != options.Locale || result.PageCount != len(result.Pages) || result.PageCount != manifest.PageCount {
		return nil, fmt.Errorf("retranslation batch %q has incompatible process result", options.BatchID)
	}
	resultIndex := -1
	for i := range result.Pages {
		if result.Pages[i].PageID == options.PageID {
			if resultIndex != -1 {
				return nil, fmt.Errorf("duplicate result page_id %q", options.PageID)
			}
			resultIndex = i
		}
	}
	if resultIndex == -1 {
		return nil, fmt.Errorf("page_id %q is not in retranslation batch %q", options.PageID, options.BatchID)
	}
	if result.Pages[resultIndex].Status != "restore_failed" && result.Pages[resultIndex].Status != "validation_failed" {
		return nil, fmt.Errorf("page_id %q status %q is not retryable", options.PageID, result.Pages[resultIndex].Status)
	}

	validationPath := filepath.Join(batchDir, "validation", strings.TrimSuffix(flattenedPageArticleName(options.PageID), ".article")+".json")
	validationData, err := os.ReadFile(validationPath)
	if err != nil {
		return nil, fmt.Errorf("read current validation for %s: %w", options.PageID, err)
	}
	var current RetranslationValidation
	if err := json.Unmarshal(validationData, &current); err != nil {
		return nil, fmt.Errorf("parse current validation for %s: %w", options.PageID, err)
	}
	if current.SchemaVersion != retranslationProcessSchemaVersion || current.BatchID != options.BatchID || current.Locale != options.Locale || current.PageID != options.PageID || current.Status != result.Pages[resultIndex].Status {
		return nil, fmt.Errorf("current validation for %s does not match result.json", options.PageID)
	}
	flatID := strings.TrimSuffix(flattenedPageArticleName(options.PageID), ".article")
	currentAttempt, err := retryValidationAttempt(current.RawResponsePath, flatID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", options.PageID, err)
	}
	nextAttempt := currentAttempt + 1
	retryDirRelative := filepath.ToSlash(filepath.Join("retries", flatID))
	retryRawRelative := filepath.ToSlash(filepath.Join(retryDirRelative, fmt.Sprintf("attempt-%03d.article", nextAttempt)))
	retryRawPath := filepath.Join(batchDir, filepath.FromSlash(retryRawRelative))
	retryRaw, err := os.ReadFile(retryRawPath)
	if err != nil {
		return nil, fmt.Errorf("read next retry raw response %s: %w", retryRawRelative, err)
	}
	if err := validateRetryAttemptSequence(filepath.Join(batchDir, filepath.FromSlash(retryDirRelative)), currentAttempt, nextAttempt); err != nil {
		return nil, err
	}
	historyRelative := filepath.ToSlash(filepath.Join(retryDirRelative, fmt.Sprintf("attempt-%03d-validation.json", currentAttempt)))
	historyPath := filepath.Join(batchDir, filepath.FromSlash(historyRelative))
	if _, err := os.Stat(historyPath); err == nil {
		return nil, fmt.Errorf("retry validation history already exists: %s", historyRelative)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect retry validation history: %w", err)
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
		if prepared[i].page.ID == options.PageID {
			item = &prepared[i]
			break
		}
	}
	if item == nil {
		return nil, fmt.Errorf("page_id %q is not in manifest", options.PageID)
	}

	candidateRelative := filepath.ToSlash(filepath.Join("candidates", flattenedPageArticleName(options.PageID)))
	evidence := RetranslationValidation{
		SchemaVersion: retranslationProcessSchemaVersion, BatchID: options.BatchID, Locale: options.Locale,
		PageID: options.PageID, InputPath: item.manifest.InputPath, RawResponsePath: retryRawRelative,
	}
	pageResult := &result.Pages[resultIndex]
	pageResult.CandidatePath = ""
	restored, failures := item.protected.restore(string(retryRaw))
	var candidate []byte
	if len(failures) != 0 {
		evidence.Status = "restore_failed"
		evidence.Error = strings.Join(failures, "; ")
		pageResult.Status = evidence.Status
	} else {
		candidate = []byte(restored)
		evidence.CandidatePath = candidateRelative
		pageResult.CandidatePath = candidateRelative
		if err := ValidateCandidate(root, catalog, options.PageID, candidate); err != nil {
			evidence.Status = "validation_failed"
			evidence.Error = err.Error()
		} else {
			evidence.Status = "passed"
		}
		pageResult.Status = evidence.Status
	}
	recountRetranslationResult(&result)
	newValidation, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return nil, err
	}
	newValidation = append(newValidation, '\n')
	newResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	newResult = append(newResult, '\n')
	updates := []retryFileUpdate{
		{target: historyPath, data: validationData, requireMissing: true},
	}
	if candidate != nil {
		updates = append(updates, retryFileUpdate{target: filepath.Join(batchDir, filepath.FromSlash(candidateRelative)), data: candidate})
	}
	updates = append(updates,
		retryFileUpdate{target: validationPath, data: newValidation},
		retryFileUpdate{target: resultPath, data: newResult},
	)
	if err := commitRetryFileUpdates(batchDir, updates); err != nil {
		return nil, err
	}
	return &result, nil
}

func retryValidationAttempt(rawPath, flatID string) (int, error) {
	if rawPath == filepath.ToSlash(filepath.Join("raw-responses", flatID+".article")) {
		return 1, nil
	}
	match := retryRawPathRE.FindStringSubmatch(rawPath)
	if match == nil || match[1] != flatID {
		return 0, fmt.Errorf("validation raw_response_path %q is not a recognized attempt", rawPath)
	}
	attempt, err := strconv.Atoi(match[2])
	if err != nil || attempt < 2 {
		return 0, fmt.Errorf("invalid retry attempt in raw_response_path %q", rawPath)
	}
	return attempt, nil
}

func validateRetryAttemptSequence(dir string, current, next int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read retry directory: %w", err)
	}
	articleRE := regexp.MustCompile(`^attempt-([0-9]{3})\.article$`)
	validationRE := regexp.MustCompile(`^attempt-([0-9]{3})-validation\.json$`)
	found := map[int]bool{}
	history := map[int]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			return fmt.Errorf("unexpected retry directory %q", entry.Name())
		}
		match := articleRE.FindStringSubmatch(entry.Name())
		if match == nil {
			validationMatch := validationRE.FindStringSubmatch(entry.Name())
			if validationMatch == nil {
				return fmt.Errorf("unexpected retry file %q", entry.Name())
			}
			attempt, _ := strconv.Atoi(validationMatch[1])
			if attempt < 1 || attempt >= current {
				return fmt.Errorf("unexpected retry validation history attempt %03d", attempt)
			}
			history[attempt] = true
			continue
		}
		attempt, _ := strconv.Atoi(match[1])
		if attempt < 2 || attempt > next {
			return fmt.Errorf("unexpected retry attempt %03d; next expected attempt is %03d", attempt, next)
		}
		found[attempt] = true
	}
	for attempt := 2; attempt <= next; attempt++ {
		if !found[attempt] {
			return fmt.Errorf("missing retry attempt-%03d.article", attempt)
		}
	}
	for attempt := 1; attempt < current; attempt++ {
		if !history[attempt] {
			return fmt.Errorf("missing retry attempt-%03d-validation.json", attempt)
		}
	}
	return nil
}

func recountRetranslationResult(result *RetranslationProcessResult) {
	result.RestorePassed = 0
	result.RestoreFailed = 0
	result.ValidationPassed = 0
	result.ValidationFailed = 0
	for _, page := range result.Pages {
		switch page.Status {
		case "restore_failed":
			result.RestoreFailed++
		case "validation_failed":
			result.RestorePassed++
			result.ValidationFailed++
		case "passed":
			result.RestorePassed++
			result.ValidationPassed++
		}
	}
}

type retryFileUpdate struct {
	target         string
	data           []byte
	requireMissing bool
}

func commitRetryFileUpdates(batchDir string, updates []retryFileUpdate) error {
	staging, err := os.MkdirTemp(batchDir, ".retry-staging-")
	if err != nil {
		return fmt.Errorf("create retry staging: %w", err)
	}
	defer os.RemoveAll(staging)
	type committedUpdate struct {
		target, backup string
	}
	var committed []committedUpdate
	rollback := func() {
		for i := len(committed) - 1; i >= 0; i-- {
			_ = os.Remove(committed[i].target)
			if committed[i].backup != "" {
				_ = os.Rename(committed[i].backup, committed[i].target)
			}
		}
	}
	for i, update := range updates {
		if err := os.MkdirAll(filepath.Dir(update.target), 0755); err != nil {
			rollback()
			return err
		}
		if update.requireMissing {
			if _, err := os.Stat(update.target); err == nil {
				rollback()
				return fmt.Errorf("retry evidence already exists: %s", update.target)
			} else if !os.IsNotExist(err) {
				rollback()
				return err
			}
		}
		staged := filepath.Join(staging, fmt.Sprintf("new-%03d", i))
		if err := os.WriteFile(staged, update.data, 0644); err != nil {
			rollback()
			return err
		}
		backup := ""
		if _, err := os.Stat(update.target); err == nil {
			backup = filepath.Join(staging, fmt.Sprintf("old-%03d", i))
			if err := os.Rename(update.target, backup); err != nil {
				rollback()
				return err
			}
		} else if !os.IsNotExist(err) {
			rollback()
			return err
		}
		if err := os.Rename(staged, update.target); err != nil {
			if backup != "" {
				_ = os.Rename(backup, update.target)
			}
			rollback()
			return err
		}
		committed = append(committed, committedUpdate{target: update.target, backup: backup})
	}
	return nil
}
