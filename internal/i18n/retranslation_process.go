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

const retranslationProcessSchemaVersion = 1

type RetranslationProcessOptions struct {
	Locale  string
	BatchID string
}

type RetranslationPageResult struct {
	PageID         string `json:"page_id"`
	Status         string `json:"status"`
	CandidatePath  string `json:"candidate_path,omitempty"`
	ValidationPath string `json:"validation_path"`
}

type RetranslationProcessResult struct {
	SchemaVersion    int                       `json:"schema_version"`
	BatchID          string                    `json:"batch_id"`
	Locale           string                    `json:"locale"`
	PageCount        int                       `json:"page_count"`
	RestorePassed    int                       `json:"restore_passed"`
	RestoreFailed    int                       `json:"restore_failed"`
	ValidationPassed int                       `json:"validation_passed"`
	ValidationFailed int                       `json:"validation_failed"`
	Pages            []RetranslationPageResult `json:"pages"`
	NoPendingBatches bool                      `json:"no_pending_batches,omitempty"`
}

type RetranslationValidation struct {
	SchemaVersion   int    `json:"schema_version"`
	BatchID         string `json:"batch_id"`
	Locale          string `json:"locale"`
	PageID          string `json:"page_id"`
	Status          string `json:"status"`
	InputPath       string `json:"input_path"`
	RawResponsePath string `json:"raw_response_path"`
	CandidatePath   string `json:"candidate_path,omitempty"`
	Error           string `json:"error,omitempty"`
}

type preparedRetranslationPage struct {
	manifest  RetranslationBatchPage
	page      Page
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
	if options.Locale != "zh-CN" {
		return nil, fmt.Errorf("unsupported locale %q", options.Locale)
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
		PageCount: len(prepared), Pages: make([]RetranslationPageResult, 0, len(prepared)),
	}
	for _, item := range prepared {
		name := flattenedPageArticleName(item.page.ID)
		rawPath := filepath.ToSlash(filepath.Join("raw-responses", name))
		candidatePath := filepath.ToSlash(filepath.Join("candidates", name))
		validationPath := filepath.ToSlash(filepath.Join("validation", strings.TrimSuffix(name, ".article")+".json"))
		evidence := RetranslationValidation{
			SchemaVersion: retranslationProcessSchemaVersion, BatchID: batchID, Locale: options.Locale,
			PageID: item.page.ID, InputPath: item.manifest.InputPath, RawResponsePath: rawPath,
		}
		pageResult := RetranslationPageResult{PageID: item.page.ID, ValidationPath: validationPath}
		restored, failures := item.protected.restore(string(item.raw))
		if len(failures) != 0 {
			evidence.Status = "restore_failed"
			evidence.Error = strings.Join(failures, "; ")
			pageResult.Status = evidence.Status
			result.RestoreFailed++
		} else {
			result.RestorePassed++
			if err := os.WriteFile(filepath.Join(staging, "candidates", name), []byte(restored), 0644); err != nil {
				return nil, fmt.Errorf("write staged candidate for %s: %w", item.page.ID, err)
			}
			evidence.CandidatePath = candidatePath
			pageResult.CandidatePath = candidatePath
			if err := ValidateCandidate(root, catalog, item.page.ID, []byte(restored)); err != nil {
				evidence.Status = "validation_failed"
				evidence.Error = err.Error()
				result.ValidationFailed++
			} else {
				evidence.Status = "passed"
				result.ValidationPassed++
			}
			pageResult.Status = evidence.Status
		}
		if err := writeTranslationJSON(filepath.Join(staging, filepath.FromSlash(validationPath)), evidence); err != nil {
			return nil, fmt.Errorf("write validation for %s: %w", item.page.ID, err)
		}
		result.Pages = append(result.Pages, pageResult)
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
	var manifest RetranslationBatchManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse retranslation manifest for %q: %w", batchID, err)
	}
	if manifest.SchemaVersion != 1 || manifest.BatchID != batchID || manifest.Locale != locale || manifest.ProtectionMode != "default" || manifest.TranslationUnit != "present.Section" {
		return nil, fmt.Errorf("retranslation batch %q has incompatible manifest metadata", batchID)
	}
	if manifest.PageCount < 1 || manifest.PageCount != len(manifest.Pages) {
		return nil, fmt.Errorf("retranslation batch %q page_count %d does not match pages %d", batchID, manifest.PageCount, len(manifest.Pages))
	}
	return &manifest, nil
}

func preflightRetranslationProcess(batchDir string, catalog *Catalog, glossary *Glossary, manifest *RetranslationBatchManifest) ([]preparedRetranslationPage, error) {
	rawDir := filepath.Join(batchDir, "raw-responses")
	entries, err := os.ReadDir(rawDir)
	if err != nil {
		return nil, fmt.Errorf("read raw responses: %w", err)
	}
	expectedRaw := make(map[string]bool, len(manifest.Pages))
	seenPages := map[string]bool{}
	prepared := make([]preparedRetranslationPage, 0, len(manifest.Pages))
	for _, record := range manifest.Pages {
		if seenPages[record.PageID] {
			return nil, fmt.Errorf("duplicate manifest page_id %q", record.PageID)
		}
		seenPages[record.PageID] = true
		page, err := catalog.Page(record.PageID)
		if err != nil {
			return nil, fmt.Errorf("manifest page_id %q: %w", record.PageID, err)
		}
		if page.Article != record.Article || page.SectionNumber != record.SectionNumber || page.Route != record.Route || page.SourceSHA256 != record.SourceSHA256 || sum(page.Source) != record.SourceSHA256 {
			return nil, fmt.Errorf("%s: manifest source metadata does not match current Catalog", record.PageID)
		}
		name := flattenedPageArticleName(record.PageID)
		wantInputPath := filepath.ToSlash(filepath.Join("inputs", name))
		if record.InputPath != wantInputPath {
			return nil, fmt.Errorf("%s: input_path %q, want %q", record.PageID, record.InputPath, wantInputPath)
		}
		input, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(record.InputPath)))
		if err != nil {
			return nil, fmt.Errorf("%s: read saved input: %w", record.PageID, err)
		}
		if sum(input) != record.InputSHA256 {
			return nil, fmt.Errorf("%s: input_sha256 mismatch", record.PageID)
		}
		protected := prepareDefaultTranslationInput(page.Source, page.SourceSHA256, glossary)
		if !bytes.Equal([]byte(protected.Text), input) {
			return nil, fmt.Errorf("%s: regenerated Default protected input differs from saved input", record.PageID)
		}
		if len(protected.Tokens) != record.ProtectedTokenCount {
			return nil, fmt.Errorf("%s: protected_token_count %d, regenerated %d", record.PageID, record.ProtectedTokenCount, len(protected.Tokens))
		}
		rawName := name
		expectedRaw[rawName] = true
		raw, err := os.ReadFile(filepath.Join(rawDir, rawName))
		if err != nil {
			return nil, fmt.Errorf("%s: read raw response: %w", record.PageID, err)
		}
		prepared = append(prepared, preparedRetranslationPage{manifest: record, page: *page, protected: protected, raw: raw})
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
