package i18n

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type RetranslationPromoteOptions struct {
	Locale string
	Apply  bool
	Now    func() time.Time
	rename func(string, string) error
}

type RetranslationPromotionPage struct {
	PageID                 string `json:"page_id"`
	BatchID                string `json:"batch_id"`
	SourceCandidatePath    string `json:"source_candidate_path"`
	CanonicalCandidatePath string `json:"canonical_candidate_path"`
	CandidateSHA256        string `json:"candidate_sha256"`
	Changed                bool   `json:"changed"`
}

type RetranslationPromotionPlan struct {
	Locale         string                       `json:"locale"`
	PageCount      int                          `json:"page_count"`
	ChangedCount   int                          `json:"changed_count"`
	UnchangedCount int                          `json:"unchanged_count"`
	CanApply       bool                         `json:"can_apply"`
	Pages          []RetranslationPromotionPage `json:"pages"`
}

type preparedPromotion struct {
	plan      RetranslationPromotionPage
	candidate []byte
	page      Page
}

var promotionBatchRE = regexp.MustCompile(`^chatgpt-(zh-CN)-([0-9]+)$`)

// PromoteRetranslation builds a complete, validated promotion plan and only
// writes canonical files when Apply is explicitly true.
func PromoteRetranslation(root string, catalog *Catalog, options RetranslationPromoteOptions) (*RetranslationPromotionPlan, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if options.Locale == "" {
		return nil, errors.New("retranslation locale is required")
	}
	if options.Locale != "zh-CN" {
		return nil, fmt.Errorf("unsupported locale %q", options.Locale)
	}
	prepared, err := preflightRetranslationPromotion(root, catalog, options.Locale)
	if err != nil {
		return nil, err
	}
	plan := &RetranslationPromotionPlan{Locale: options.Locale, PageCount: len(prepared), CanApply: true, Pages: make([]RetranslationPromotionPage, 0, len(prepared))}
	for _, item := range prepared {
		plan.Pages = append(plan.Pages, item.plan)
		if item.plan.Changed {
			plan.ChangedCount++
		} else {
			plan.UnchangedCount++
		}
	}
	if options.Apply {
		if err := applyRetranslationPromotion(root, catalog, options, prepared); err != nil {
			return nil, err
		}
	}
	return plan, nil
}

func preflightRetranslationPromotion(root string, catalog *Catalog, locale string) ([]preparedPromotion, error) {
	byID := make(map[string]Page, len(catalog.Pages))
	for _, page := range catalog.Pages {
		if _, exists := byID[page.ID]; exists {
			return nil, fmt.Errorf("duplicate Catalog page_id %q", page.ID)
		}
		byID[page.ID] = page
	}
	base := filepath.Join(root, "data", "retranslation-runs", locale)
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("scan retranslation batches: %w", err)
	}
	type selected struct {
		number   int
		batchID  string
		batchDir string
		manifest RetranslationBatchPage
		result   RetranslationPageResult
	}
	selectedByID := map[string]selected{}
	seenNumbers := map[int]string{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !entry.IsDir() {
			return nil, fmt.Errorf("illegal retranslation batch entry %q", entry.Name())
		}
		match := promotionBatchRE.FindStringSubmatch(entry.Name())
		if match == nil || match[1] != locale {
			return nil, fmt.Errorf("illegal retranslation batch %q", entry.Name())
		}
		number, _ := strconv.Atoi(match[2])
		if number < 1 {
			return nil, fmt.Errorf("illegal retranslation batch %q", entry.Name())
		}
		if prior := seenNumbers[number]; prior != "" {
			return nil, fmt.Errorf("ambiguous retranslation batch number %03d: %q and %q", number, prior, entry.Name())
		}
		seenNumbers[number] = entry.Name()
		batchDir := filepath.Join(base, entry.Name())
		manifest, err := readRetranslationProcessManifest(batchDir, locale, entry.Name())
		if err != nil {
			return nil, err
		}
		result, err := readPromotionResult(batchDir, locale, entry.Name(), manifest.PageCount)
		if err != nil {
			return nil, err
		}
		results := map[string]RetranslationPageResult{}
		for _, record := range result.Pages {
			if _, exists := results[record.PageID]; exists {
				return nil, fmt.Errorf("retranslation batch %q has duplicate result page_id %q", entry.Name(), record.PageID)
			}
			results[record.PageID] = record
		}
		manifestSeen := map[string]bool{}
		for _, record := range manifest.Pages {
			if manifestSeen[record.PageID] {
				return nil, fmt.Errorf("retranslation batch %q has duplicate manifest page_id %q", entry.Name(), record.PageID)
			}
			manifestSeen[record.PageID] = true
			page, ok := byID[record.PageID]
			if !ok {
				return nil, fmt.Errorf("retranslation batch %q has extra page_id %q", entry.Name(), record.PageID)
			}
			if record.Article != page.Article || record.SectionNumber != page.SectionNumber || record.Route != page.Route || record.SourceSHA256 != page.SourceSHA256 || sum(page.Source) != page.SourceSHA256 {
				return nil, fmt.Errorf("%s: manifest source metadata does not match current Catalog", record.PageID)
			}
			pageResult, ok := results[record.PageID]
			if !ok {
				return nil, fmt.Errorf("retranslation batch %q result missing page_id %q", entry.Name(), record.PageID)
			}
			if current, ok := selectedByID[record.PageID]; !ok || number > current.number {
				selectedByID[record.PageID] = selected{number: number, batchID: entry.Name(), batchDir: batchDir, manifest: record, result: pageResult}
			}
		}
		if len(results) != len(manifestSeen) {
			return nil, fmt.Errorf("retranslation batch %q result page set does not match manifest", entry.Name())
		}
	}
	if len(selectedByID) != len(byID) {
		missing := make([]string, 0)
		for id := range byID {
			if _, ok := selectedByID[id]; !ok {
				missing = append(missing, id)
			}
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("retranslation promotion covers %d of %d Catalog pages; missing: %s", len(selectedByID), len(byID), strings.Join(missing, ", "))
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	prepared := make([]preparedPromotion, 0, len(ids))
	for _, id := range ids {
		choice := selectedByID[id]
		if choice.result.Status != "passed" {
			return nil, fmt.Errorf("%s: latest batch %q status %q is not passed; refusing fallback", id, choice.batchID, choice.result.Status)
		}
		name := flattenedPageArticleName(id)
		wantCandidate := filepath.ToSlash(filepath.Join("candidates", name))
		wantValidation := filepath.ToSlash(filepath.Join("validation", strings.TrimSuffix(name, ".article")+".json"))
		if choice.result.CandidatePath != wantCandidate || choice.result.ValidationPath != wantValidation {
			return nil, fmt.Errorf("%s: result candidate/validation path mismatch", id)
		}
		validation, err := readPromotionValidation(choice.batchDir, choice.batchID, locale, choice.manifest, choice.result)
		if err != nil {
			return nil, err
		}
		if validation.Status != "passed" || validation.CandidatePath != wantCandidate {
			return nil, fmt.Errorf("%s: validation/result is not consistently passed", id)
		}
		sourceAbs := filepath.Join(choice.batchDir, filepath.FromSlash(wantCandidate))
		candidate, err := os.ReadFile(sourceAbs)
		if err != nil {
			return nil, fmt.Errorf("%s: read promotion candidate: %w", id, err)
		}
		if bytes.Contains(candidate, []byte("GTI18N")) {
			return nil, fmt.Errorf("%s: candidate contains GTI18N token", id)
		}
		if err := ValidateCandidateForLocale(root, catalog, id, locale, candidate); err != nil {
			return nil, fmt.Errorf("%s: canonical candidate validator: %w", id, err)
		}
		canonical := canonicalCandidatePath(locale, id)
		current, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(canonical)))
		if readErr != nil && !os.IsNotExist(readErr) {
			return nil, fmt.Errorf("%s: read canonical candidate: %w", id, readErr)
		}
		relSource, err := repositoryRelativePath(root, sourceAbs)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, preparedPromotion{page: byID[id], candidate: candidate, plan: RetranslationPromotionPage{
			PageID: id, BatchID: choice.batchID, SourceCandidatePath: relSource,
			CanonicalCandidatePath: canonical, CandidateSHA256: sum(candidate), Changed: readErr != nil || !bytes.Equal(current, candidate),
		}})
	}
	return prepared, nil
}

func readPromotionResult(batchDir, locale, batchID string, pageCount int) (*RetranslationProcessResult, error) {
	b, err := os.ReadFile(filepath.Join(batchDir, "result.json"))
	if err != nil {
		return nil, fmt.Errorf("read retranslation result for %q: %w", batchID, err)
	}
	var result RetranslationProcessResult
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("parse retranslation result for %q: %w", batchID, err)
	}
	if result.SchemaVersion != retranslationProcessSchemaVersion || result.BatchID != batchID || result.Locale != locale || result.PageCount != pageCount || len(result.Pages) != pageCount {
		return nil, fmt.Errorf("retranslation batch %q has incompatible process result", batchID)
	}
	return &result, nil
}

func readPromotionValidation(batchDir, batchID, locale string, manifest RetranslationBatchPage, result RetranslationPageResult) (*RetranslationValidation, error) {
	b, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(result.ValidationPath)))
	if err != nil {
		return nil, fmt.Errorf("%s: read promotion validation: %w", manifest.PageID, err)
	}
	var validation RetranslationValidation
	if err := json.Unmarshal(b, &validation); err != nil {
		return nil, fmt.Errorf("%s: parse promotion validation: %w", manifest.PageID, err)
	}
	if validation.SchemaVersion != retranslationProcessSchemaVersion || validation.BatchID != batchID || validation.Locale != locale || validation.PageID != manifest.PageID || validation.InputPath != manifest.InputPath || validation.Status != result.Status || validation.CandidatePath != result.CandidatePath {
		return nil, fmt.Errorf("%s: validation does not match manifest/result.json", manifest.PageID)
	}
	return &validation, nil
}

func applyRetranslationPromotion(root string, catalog *Catalog, options RetranslationPromoteOptions, prepared []preparedPromotion) error {
	localeDir := filepath.Join(root, "locales", options.Locale)
	statusPath := filepath.Join(localeDir, "status.tsv")
	statuses, err := ReadStatuses(statusPath)
	if err != nil {
		return fmt.Errorf("read canonical status: %w", err)
	}
	if len(statuses) != len(catalog.Pages) {
		return fmt.Errorf("status entries = %d, want %d", len(statuses), len(catalog.Pages))
	}
	statusByID := map[string]int{}
	for i, status := range statuses {
		if _, exists := statusByID[status.PageID]; exists {
			return fmt.Errorf("duplicate status page_id %q", status.PageID)
		}
		page, err := catalog.Page(status.PageID)
		if err != nil || status.SourceSHA256 != page.SourceSHA256 {
			return fmt.Errorf("%s: invalid canonical status source", status.PageID)
		}
		statusByID[status.PageID] = i
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	updated := now().UTC().Format(time.RFC3339)
	staging, err := os.MkdirTemp(localeDir, ".promotion-staging-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	stagedCandidates := filepath.Join(staging, "candidates")
	canonicalCandidates := filepath.Join(localeDir, "candidates")
	if err := copyDirectory(canonicalCandidates, stagedCandidates); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(stagedCandidates, 0755); err != nil {
		return err
	}
	for _, item := range prepared {
		target := filepath.Join(stagedCandidates, filepath.Base(filepath.FromSlash(item.plan.CanonicalCandidatePath)))
		if err := os.WriteFile(target, item.candidate, 0644); err != nil {
			return err
		}
		i, ok := statusByID[item.plan.PageID]
		if !ok {
			return fmt.Errorf("status missing page_id %q", item.plan.PageID)
		}
		statuses[i].State = "ready"
		statuses[i].SourceSHA256 = item.page.SourceSHA256
		statuses[i].CandidatePath = item.plan.CanonicalCandidatePath
		statuses[i].UpdatedAt = updated
		statuses[i].Note = fmt.Sprintf("ChatGPT retranslation promoted from %s; passed canonical validator", item.plan.BatchID)
	}
	stagedStatus := filepath.Join(staging, "status.tsv")
	if err := writeStatuses(stagedStatus, statuses); err != nil {
		return err
	}
	rename := options.rename
	if rename == nil {
		rename = os.Rename
	}
	backupCandidates := filepath.Join(staging, "old-candidates")
	backupStatus := filepath.Join(staging, "old-status.tsv")
	candidatesBackedUp := false
	if _, err := os.Stat(canonicalCandidates); err == nil {
		if err := rename(canonicalCandidates, backupCandidates); err != nil {
			return fmt.Errorf("backup canonical candidates: %w", err)
		}
		candidatesBackedUp = true
	} else if !os.IsNotExist(err) {
		return err
	}
	rollbackCandidates := func() {
		_ = os.RemoveAll(canonicalCandidates)
		if candidatesBackedUp {
			_ = os.Rename(backupCandidates, canonicalCandidates)
		}
	}
	if err := rename(stagedCandidates, canonicalCandidates); err != nil {
		rollbackCandidates()
		return fmt.Errorf("install canonical candidates: %w", err)
	}
	if err := rename(statusPath, backupStatus); err != nil {
		rollbackCandidates()
		return fmt.Errorf("backup canonical status: %w", err)
	}
	if err := rename(stagedStatus, statusPath); err != nil {
		_ = os.Rename(backupStatus, statusPath)
		rollbackCandidates()
		return fmt.Errorf("install canonical status: %w", err)
	}
	return nil
}

func copyDirectory(source, target string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return fmt.Errorf("unexpected directory in canonical candidates: %s", entry.Name())
		}
		in, err := os.Open(filepath.Join(source, entry.Name()))
		if err != nil {
			return err
		}
		out, err := os.OpenFile(filepath.Join(target, entry.Name()), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			_, err = io.Copy(out, in)
		}
		closeOut := error(nil)
		if out != nil {
			closeOut = out.Close()
		}
		_ = in.Close()
		if err != nil {
			return err
		}
		if closeOut != nil {
			return closeOut
		}
	}
	return nil
}
