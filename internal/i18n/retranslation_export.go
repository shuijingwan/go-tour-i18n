package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const defaultRetranslationExportLimit = 10

type RetranslationExportOptions struct {
	Locale  string
	BatchID string
	PageIDs []string
	Limit   int
}

type RetranslationBatchPage struct {
	PageID              string `json:"page_id"`
	Article             string `json:"article"`
	SectionNumber       int    `json:"section_number"`
	Route               string `json:"route"`
	SourceSHA256        string `json:"source_sha256"`
	InputPath           string `json:"input_path"`
	InputSHA256         string `json:"input_sha256"`
	ProtectedTokenCount int    `json:"protected_token_count"`
}

type RetranslationBatchManifest struct {
	SchemaVersion   int                      `json:"schema_version"`
	BatchID         string                   `json:"batch_id"`
	Locale          string                   `json:"locale"`
	ProtectionMode  string                   `json:"protection_mode"`
	TranslationUnit string                   `json:"translation_unit"`
	PageCount       int                      `json:"page_count"`
	Pages           []RetranslationBatchPage `json:"pages"`
}

type RetranslationExportResult struct {
	Locale      string   `json:"locale"`
	BatchID     string   `json:"batch_id,omitempty"`
	BatchPath   string   `json:"batch_path,omitempty"`
	PageCount   int      `json:"page_count"`
	PageIDs     []string `json:"page_ids,omitempty"`
	AllExported bool     `json:"all_exported"`
}

type preparedRetranslationInput struct {
	page   Page
	text   string
	path   string
	hash   string
	tokens int
}

// ExportRetranslationBatch writes one isolated batch of Default protected
// inputs without invoking a model or changing formal translation state.
func ExportRetranslationBatch(root string, catalog *Catalog, options RetranslationExportOptions) (*RetranslationExportResult, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if options.Locale == "" {
		return nil, errors.New("retranslation locale is required")
	}
	if options.Locale != "zh-CN" {
		return nil, fmt.Errorf("unsupported locale %q", options.Locale)
	}
	limit := options.Limit
	if limit == 0 {
		limit = defaultRetranslationExportLimit
	}
	if limit < 1 {
		return nil, errors.New("retranslation export limit must be greater than zero")
	}

	base := filepath.Join(root, "data", "retranslation-runs", options.Locale)
	exported, nextNumber, err := scanRetranslationBatches(base, options.Locale, catalog)
	if err != nil {
		return nil, err
	}
	pages, err := selectRetranslationPages(catalog, options.PageIDs, exported, limit)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return &RetranslationExportResult{Locale: options.Locale, AllExported: true}, nil
	}

	batchID := options.BatchID
	if batchID == "" {
		batchID = fmt.Sprintf("chatgpt-%s-%03d", options.Locale, nextNumber)
	}
	if err := validateBatchID(batchID); err != nil {
		return nil, err
	}
	finalDir := filepath.Join(base, batchID)
	if err := requireMissingBatchDirectory(finalDir); err != nil {
		return nil, err
	}

	glossary, err := LoadGlossary(root, options.Locale)
	if err != nil {
		return nil, err
	}
	prepared := make([]preparedRetranslationInput, 0, len(pages))
	for _, page := range pages {
		if sum(page.Source) != page.SourceSHA256 {
			return nil, fmt.Errorf("%s: hydrated source hash mismatch", page.ID)
		}
		protected := prepareDefaultTranslationInput(page.Source, page.SourceSHA256, glossary)
		inputPath := filepath.ToSlash(filepath.Join("inputs", flattenedPageArticleName(page.ID)))
		prepared = append(prepared, preparedRetranslationInput{
			page: page, text: protected.Text, path: inputPath,
			hash: sum([]byte(protected.Text)), tokens: len(protected.Tokens),
		})
	}

	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, fmt.Errorf("create retranslation locale directory: %w", err)
	}
	staging, err := os.MkdirTemp(base, "."+batchID+".staging-")
	if err != nil {
		return nil, fmt.Errorf("create retranslation staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := os.Mkdir(filepath.Join(staging, "inputs"), 0755); err != nil {
		return nil, fmt.Errorf("create retranslation inputs directory: %w", err)
	}
	manifest := RetranslationBatchManifest{
		SchemaVersion: 1, BatchID: batchID, Locale: options.Locale,
		ProtectionMode: "default", TranslationUnit: "present.Section",
		PageCount: len(prepared), Pages: make([]RetranslationBatchPage, 0, len(prepared)),
	}
	pageIDs := make([]string, 0, len(prepared))
	for _, input := range prepared {
		if err := os.WriteFile(filepath.Join(staging, filepath.FromSlash(input.path)), []byte(input.text), 0644); err != nil {
			return nil, fmt.Errorf("write retranslation input for %s: %w", input.page.ID, err)
		}
		manifest.Pages = append(manifest.Pages, RetranslationBatchPage{
			PageID: input.page.ID, Article: input.page.Article, SectionNumber: input.page.SectionNumber,
			Route: input.page.Route, SourceSHA256: input.page.SourceSHA256, InputPath: input.path,
			InputSHA256: input.hash, ProtectedTokenCount: input.tokens,
		})
		pageIDs = append(pageIDs, input.page.ID)
	}
	if err := writeTranslationJSON(filepath.Join(staging, "manifest.json"), manifest); err != nil {
		return nil, fmt.Errorf("write retranslation manifest: %w", err)
	}
	if err := requireMissingBatchDirectory(finalDir); err != nil {
		return nil, err
	}
	if err := os.Rename(staging, finalDir); err != nil {
		return nil, fmt.Errorf("commit retranslation batch: %w", err)
	}
	committed = true
	batchPath, err := repositoryRelativePath(root, finalDir)
	if err != nil {
		return nil, err
	}
	return &RetranslationExportResult{
		Locale: options.Locale, BatchID: batchID, BatchPath: batchPath,
		PageCount: len(pageIDs), PageIDs: pageIDs,
	}, nil
}

func selectRetranslationPages(catalog *Catalog, requested []string, exported map[string]string, limit int) ([]Page, error) {
	byID := make(map[string]Page, len(catalog.Pages))
	for _, page := range catalog.Pages {
		byID[page.ID] = page
	}
	if len(requested) != 0 {
		seen := map[string]bool{}
		pages := make([]Page, 0, len(requested))
		for _, pageID := range requested {
			if seen[pageID] {
				return nil, fmt.Errorf("duplicate requested page_id %q", pageID)
			}
			seen[pageID] = true
			page, ok := byID[pageID]
			if !ok {
				return nil, fmt.Errorf("unknown page_id %q", pageID)
			}
			if batch := exported[pageID]; batch != "" {
				return nil, fmt.Errorf("page_id %q was already exported in batch %q", pageID, batch)
			}
			pages = append(pages, page)
		}
		return pages, nil
	}
	pages := make([]Page, 0, limit)
	for _, page := range catalog.Pages {
		if exported[page.ID] != "" {
			continue
		}
		pages = append(pages, page)
		if len(pages) == limit {
			break
		}
	}
	return pages, nil
}

func scanRetranslationBatches(base, locale string, catalog *Catalog) (map[string]string, int, error) {
	exported := map[string]string{}
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return exported, 1, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("scan retranslation batches: %w", err)
	}
	known := make(map[string]bool, len(catalog.Pages))
	for _, page := range catalog.Pages {
		known[page.ID] = true
	}
	prefix := "chatgpt-" + locale + "-"
	batchPattern := regexp.MustCompile("^" + regexp.QuoteMeta(prefix) + `([0-9]+)$`)
	nextNumber := 1
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if match := batchPattern.FindStringSubmatch(entry.Name()); match != nil {
			var number int
			if _, err := fmt.Sscanf(match[1], "%d", &number); err == nil && number >= nextNumber {
				nextNumber = number + 1
			}
		}
		manifestPath := filepath.Join(base, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, 0, fmt.Errorf("read retranslation manifest %s: %w", filepath.ToSlash(manifestPath), err)
		}
		var manifest RetranslationBatchManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, 0, fmt.Errorf("parse retranslation manifest %s: %w", filepath.ToSlash(manifestPath), err)
		}
		if manifest.BatchID != entry.Name() {
			return nil, 0, fmt.Errorf("retranslation manifest batch_id %q does not match directory %q", manifest.BatchID, entry.Name())
		}
		if manifest.Locale != locale {
			return nil, 0, fmt.Errorf("retranslation batch %q locale %q does not match %q", entry.Name(), manifest.Locale, locale)
		}
		if manifest.SchemaVersion != 1 || manifest.ProtectionMode != "default" || manifest.TranslationUnit != "present.Section" {
			return nil, 0, fmt.Errorf("retranslation batch %q has incompatible manifest metadata", entry.Name())
		}
		if manifest.PageCount == 0 {
			return nil, 0, fmt.Errorf("retranslation batch %q has no pages", entry.Name())
		}
		if manifest.PageCount != len(manifest.Pages) {
			return nil, 0, fmt.Errorf("retranslation batch %q page_count %d does not match pages %d", entry.Name(), manifest.PageCount, len(manifest.Pages))
		}
		for _, page := range manifest.Pages {
			if !known[page.PageID] {
				return nil, 0, fmt.Errorf("retranslation batch %q has unknown page_id %q", entry.Name(), page.PageID)
			}
			if previous := exported[page.PageID]; previous != "" {
				return nil, 0, fmt.Errorf("page_id %q appears in multiple retranslation batches %q and %q", page.PageID, previous, entry.Name())
			}
			exported[page.PageID] = entry.Name()
		}
	}
	return exported, nextNumber, nil
}

func validateBatchID(batchID string) error {
	if batchID == "" || strings.HasPrefix(batchID, ".") || filepath.Base(batchID) != batchID || batchID == "." || strings.ContainsAny(batchID, `/\\`) {
		return fmt.Errorf("invalid retranslation batch_id %q", batchID)
	}
	return nil
}

func requireMissingBatchDirectory(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("retranslation batch directory already exists: %s", filepath.ToSlash(path))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect retranslation batch directory: %w", err)
	}
	return nil
}
