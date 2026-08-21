package i18n

import (
	"bytes"
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

type RetranslationPromotionUnit struct {
	UnitID                 string   `json:"unit_id"`
	UnitKind               UnitKind `json:"unit_kind"`
	BatchID                string   `json:"batch_id"`
	SourceCandidatePath    string   `json:"source_candidate_path"`
	CanonicalCandidatePath string   `json:"canonical_candidate_path"`
	SourceCandidateSHA256  string   `json:"source_candidate_sha256"`
	CandidateSHA256        string   `json:"candidate_sha256"`
	EOFNormalized          bool     `json:"eof_normalized"`
	Changed                bool     `json:"changed"`
}

type RetranslationPromotionPlan struct {
	Locale              string                       `json:"locale"`
	UnitCount           int                          `json:"unit_count"`
	PageCount           int                          `json:"page_count"`
	ExampleCount        int                          `json:"example_count"`
	ChangedCount        int                          `json:"changed_count"`
	UnchangedCount      int                          `json:"unchanged_count"`
	EOFNormalizedCount  int                          `json:"eof_normalized_count"`
	ReviewApprovedCount int                          `json:"review_approved_count"`
	CanApply            bool                         `json:"can_apply"`
	MissingEvidence     []string                     `json:"missing_evidence,omitempty"`
	MissingReview       []string                     `json:"missing_review"`
	RejectedReview      []string                     `json:"rejected_review"`
	InvalidReview       []string                     `json:"invalid_review"`
	Units               []RetranslationPromotionUnit `json:"units"`
}

type promotionReviewGate struct {
	approved int
	missing  []string
	rejected []string
	invalid  []string
}

type preparedPromotion struct {
	plan      RetranslationPromotionUnit
	candidate []byte
	unit      *TranslationUnit
	attempt   int
}

func promotionBatchRE(locale string) *regexp.Regexp {
	return regexp.MustCompile(`^chatgpt-(` + regexp.QuoteMeta(locale) + `)-([0-9]+)$`)
}

// PromoteRetranslation builds a complete, validated promotion plan and only
// writes canonical files when Apply is explicitly true.
func PromoteRetranslation(root string, catalog *Catalog, options RetranslationPromoteOptions) (*RetranslationPromotionPlan, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if options.Locale == "" {
		return nil, errors.New("retranslation locale is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	prepared, missing, reviews, pages, examples, err := preflightUnifiedRetranslationPromotion(root, catalog, options.Locale)
	if err != nil {
		return nil, err
	}
	plan := &RetranslationPromotionPlan{
		Locale: options.Locale, UnitCount: pages + examples, PageCount: pages, ExampleCount: examples,
		ReviewApprovedCount: reviews.approved,
		CanApply:            len(missing) == 0 && len(reviews.missing) == 0 && len(reviews.rejected) == 0 && len(reviews.invalid) == 0,
		MissingEvidence:     missing, MissingReview: reviews.missing, RejectedReview: reviews.rejected, InvalidReview: reviews.invalid,
		Units: make([]RetranslationPromotionUnit, 0, len(prepared)),
	}
	for _, item := range prepared {
		plan.Units = append(plan.Units, item.plan)
		if item.plan.Changed {
			plan.ChangedCount++
		} else {
			plan.UnchangedCount++
		}
		if item.plan.EOFNormalized {
			plan.EOFNormalizedCount++
		}
	}
	if options.Apply {
		if !plan.CanApply {
			return plan, fmt.Errorf("promotion cannot apply: missing or invalid evidence")
		}
		if err := applyRetranslationPromotion(root, catalog, options, prepared); err != nil {
			return nil, err
		}
	}
	return plan, nil
}

func preflightRetranslationPromotion(root string, catalog *Catalog, locale string) ([]preparedPromotion, error) {
	glossary, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, err
	}
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
		manifest RetranslationBatchUnit
		result   RetranslationUnitResult
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
		match := promotionBatchRE(locale).FindStringSubmatch(entry.Name())
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
		result, err := readPromotionResult(batchDir, locale, entry.Name(), manifest.UnitCount)
		if err != nil {
			return nil, err
		}
		results := map[string]RetranslationUnitResult{}
		for _, record := range result.Units {
			if record.UnitKind != UnitKindPage {
				continue
			}
			if _, exists := results[record.UnitID]; exists {
				return nil, fmt.Errorf("retranslation batch %q has duplicate Page unit_id %q", entry.Name(), record.UnitID)
			}
			results[record.UnitID] = record
		}
		manifestSeen := map[string]bool{}
		for _, record := range manifest.Units {
			if record.UnitKind != UnitKindPage {
				continue
			}
			if manifestSeen[record.UnitID] {
				return nil, fmt.Errorf("retranslation batch %q has duplicate manifest Page unit_id %q", entry.Name(), record.UnitID)
			}
			manifestSeen[record.UnitID] = true
			page, ok := byID[record.UnitID]
			if !ok {
				return nil, fmt.Errorf("retranslation batch %q has extra Page unit_id %q", entry.Name(), record.UnitID)
			}
			if record.SourcePath != filepath.ToSlash(filepath.Join("_content", "tour", page.Article)) || record.SourceSHA256 != page.SourceSHA256 || sum(page.Source) != page.SourceSHA256 {
				return nil, fmt.Errorf("%s: manifest source metadata does not match current Catalog", record.UnitID)
			}
			pageResult, ok := results[record.UnitID]
			if !ok {
				return nil, fmt.Errorf("retranslation batch %q result missing Page unit_id %q", entry.Name(), record.UnitID)
			}
			if current, ok := selectedByID[record.UnitID]; !ok || number > current.number {
				selectedByID[record.UnitID] = selected{number: number, batchID: entry.Name(), batchDir: batchDir, manifest: record, result: pageResult}
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
		if err := validatePromotionEvidence(choice.batchDir, byID[id], choice.manifest, validation, glossary, candidate); err != nil {
			return nil, err
		}
		if bytes.Contains(candidate, []byte("GTI18N")) {
			return nil, fmt.Errorf("%s: candidate contains GTI18N token", id)
		}
		if err := ValidateCandidateForLocale(root, catalog, id, locale, candidate); err != nil {
			return nil, fmt.Errorf("%s: canonical candidate validator: %w", id, err)
		}
		canonicalCandidate := canonicalizeCandidateEOF(candidate)
		if err := ValidateCandidateForLocale(root, catalog, id, locale, canonicalCandidate); err != nil {
			return nil, fmt.Errorf("%s: canonicalized candidate validator: %w", id, err)
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
		pageUnit, err := catalog.Unit(id)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, preparedPromotion{unit: pageUnit, attempt: 1, candidate: canonicalCandidate, plan: RetranslationPromotionUnit{
			UnitID: id, UnitKind: UnitKindPage, BatchID: choice.batchID, SourceCandidatePath: relSource,
			CanonicalCandidatePath: canonical, SourceCandidateSHA256: sum(candidate), CandidateSHA256: sum(canonicalCandidate),
			EOFNormalized: !bytes.Equal(candidate, canonicalCandidate), Changed: readErr != nil || !bytes.Equal(current, canonicalCandidate),
		}})
	}
	return prepared, nil
}

func preflightUnifiedRetranslationPromotion(root string, catalog *Catalog, locale string) ([]preparedPromotion, []string, promotionReviewGate, int, int, error) {
	glossary, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, nil, promotionReviewGate{}, 0, 0, err
	}
	expected, pageCount, exampleCount, err := localeWorkflowUnits(catalog)
	if err != nil {
		return nil, nil, promotionReviewGate{}, 0, 0, err
	}
	for _, unit := range expected {
		if sum(unit.Source) != unit.SourceSHA256 {
			return nil, nil, promotionReviewGate{}, 0, 0, fmt.Errorf("%s: current Catalog source bytes do not match source_sha256", unit.ID)
		}
	}
	type selected struct {
		number   int
		batchID  string
		batchDir string
		manifest RetranslationBatchUnit
		result   RetranslationUnitResult
	}
	selectedByID := make(map[string]selected, len(expected))
	base := filepath.Join(root, "data", "retranslation-runs", locale)
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, nil, promotionReviewGate{}, 0, 0, fmt.Errorf("scan retranslation batches: %w", err)
	}
	seenNumbers := map[int]string{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !entry.IsDir() {
			return nil, nil, promotionReviewGate{}, 0, 0, fmt.Errorf("illegal retranslation batch entry %q", entry.Name())
		}
		match := promotionBatchRE(locale).FindStringSubmatch(entry.Name())
		if match == nil || match[1] != locale {
			return nil, nil, promotionReviewGate{}, 0, 0, fmt.Errorf("illegal retranslation batch %q", entry.Name())
		}
		number, _ := strconv.Atoi(match[2])
		if number < 1 || seenNumbers[number] != "" {
			return nil, nil, promotionReviewGate{}, 0, 0, fmt.Errorf("ambiguous or invalid retranslation batch number %03d", number)
		}
		seenNumbers[number] = entry.Name()
		batchDir := filepath.Join(base, entry.Name())
		manifest, err := readRetranslationProcessManifest(batchDir, locale, entry.Name())
		if err != nil {
			return nil, nil, promotionReviewGate{}, 0, 0, err
		}
		result, err := readPromotionResult(batchDir, locale, entry.Name(), manifest.UnitCount)
		if err != nil {
			return nil, nil, promotionReviewGate{}, 0, 0, err
		}
		results := make(map[string]RetranslationUnitResult, len(result.Units))
		for _, record := range result.Units {
			if _, duplicate := results[record.UnitID]; duplicate {
				return nil, nil, promotionReviewGate{}, 0, 0, fmt.Errorf("batch %q has duplicate result unit_id %q", entry.Name(), record.UnitID)
			}
			results[record.UnitID] = record
		}
		seen := map[string]bool{}
		for _, record := range manifest.Units {
			if seen[record.UnitID] {
				return nil, nil, promotionReviewGate{}, 0, 0, fmt.Errorf("batch %q has duplicate manifest unit_id %q", entry.Name(), record.UnitID)
			}
			seen[record.UnitID] = true
			unit, ok := expected[record.UnitID]
			if !ok {
				continue // Non-workflow historical units do not affect locale promotion.
			}
			if record.UnitKind != unit.Kind || record.SourcePath != unit.SourcePath || record.SourceSHA256 != unit.SourceSHA256 {
				// A manifest is immutable evidence for its own source revision.
				// Older revisions must neither block the current revision nor be
				// eligible as a fallback promotion candidate.
				continue
			}
			unitResult, ok := results[unit.ID]
			if !ok || unitResult.UnitKind != unit.Kind {
				return nil, nil, promotionReviewGate{}, 0, 0, fmt.Errorf("batch %q result missing or mismatching unit %q", entry.Name(), unit.ID)
			}
			if current, exists := selectedByID[unit.ID]; !exists || number > current.number {
				selectedByID[unit.ID] = selected{number: number, batchID: entry.Name(), batchDir: batchDir, manifest: record, result: unitResult}
			}
		}
	}
	ids := make([]string, 0, len(expected))
	for id := range expected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	prepared := make([]preparedPromotion, 0, len(ids))
	missing := make([]string, 0)
	reviews := promotionReviewGate{missing: []string{}, rejected: []string{}, invalid: []string{}}
	for _, id := range ids {
		unit := expected[id]
		choice, ok := selectedByID[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		if choice.result.Status != "passed" {
			missing = append(missing, fmt.Sprintf("%s (latest %s=%s)", id, choice.batchID, choice.result.Status))
			continue
		}
		name := filepath.Base(filepath.FromSlash(choice.manifest.InputPath))
		wantCandidate := filepath.ToSlash(filepath.Join("candidates", retranslationUnitCandidateName(unit)))
		wantValidation := filepath.ToSlash(filepath.Join("validation", strings.TrimSuffix(name, filepath.Ext(name))+".json"))
		if choice.result.CandidatePath != wantCandidate || choice.result.ValidationPath != wantValidation {
			return nil, nil, promotionReviewGate{}, 0, 0, fmt.Errorf("%s: result candidate/validation path mismatch", id)
		}
		validation, err := readPromotionValidation(choice.batchDir, choice.batchID, locale, choice.manifest, choice.result)
		if err != nil {
			return nil, nil, promotionReviewGate{}, 0, 0, err
		}
		candidate, err := os.ReadFile(filepath.Join(choice.batchDir, filepath.FromSlash(wantCandidate)))
		if err != nil {
			return nil, nil, promotionReviewGate{}, 0, 0, fmt.Errorf("%s: read promotion candidate: %w", id, err)
		}
		attempt, err := validateUnifiedPromotionEvidence(root, choice.batchDir, catalog, unit, choice.manifest, validation, glossary, candidate, locale)
		if err != nil {
			return nil, nil, promotionReviewGate{}, 0, 0, err
		}
		reviewState, err := checkPromotionReview(choice.batchDir, choice.batchID, locale, unit, choice.manifest, choice.result, validation, candidate, attempt)
		if err != nil {
			return nil, nil, promotionReviewGate{}, 0, 0, err
		}
		switch reviewState {
		case "missing":
			reviews.missing = append(reviews.missing, id)
			continue
		case "rejected":
			reviews.rejected = append(reviews.rejected, id)
			continue
		case "invalid":
			reviews.invalid = append(reviews.invalid, id)
			continue
		}
		reviews.approved++
		canonicalCandidate := candidate
		if unit.Kind == UnitKindPage {
			canonicalCandidate = canonicalizeCandidateEOF(candidate)
		}
		canonicalPath, err := canonicalTranslationUnitCandidatePath(locale, unit)
		if err != nil {
			return nil, nil, promotionReviewGate{}, 0, 0, err
		}
		existing, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(canonicalPath)))
		changed := readErr != nil || !bytes.Equal(existing, canonicalCandidate)
		prepared = append(prepared, preparedPromotion{
			plan: RetranslationPromotionUnit{
				UnitID: id, UnitKind: unit.Kind, BatchID: choice.batchID,
				SourceCandidatePath:    filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, choice.batchID, wantCandidate)),
				CanonicalCandidatePath: canonicalPath, SourceCandidateSHA256: sum(candidate), CandidateSHA256: sum(canonicalCandidate),
				EOFNormalized: !bytes.Equal(candidate, canonicalCandidate), Changed: changed,
			},
			candidate: canonicalCandidate, unit: unit, attempt: attempt,
		})
	}
	return prepared, missing, reviews, pageCount, exampleCount, nil
}

func checkPromotionReview(batchDir, batchID, locale string, unit *TranslationUnit, manifest RetranslationBatchUnit, result RetranslationUnitResult, validation *RetranslationValidation, candidate []byte, attempt int) (string, error) {
	path := filepath.Join(batchDir, "review", retranslationReviewName(unit))
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "missing", nil
	}
	if err != nil {
		return "", fmt.Errorf("%s: read promotion review: %w", unit.ID, err)
	}
	review, err := decodeTranslationReview(b)
	if err != nil {
		return "invalid", nil
	}
	validationBytes, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(result.ValidationPath)))
	if err != nil {
		return "", fmt.Errorf("%s: read promotion validation for review: %w", unit.ID, err)
	}
	if review.SchemaVersion != TranslationReviewSchemaVersion || review.BatchID != batchID || review.Locale != locale ||
		review.UnitID != unit.ID || review.UnitKind != unit.Kind || review.SourceSHA256 != manifest.SourceSHA256 ||
		review.Attempt != attempt || review.Attempt != validation.Attempt || review.CandidatePath != result.CandidatePath ||
		review.CandidateSHA256 != sum(candidate) || review.ValidationPath != result.ValidationPath || review.ValidationSHA256 != sum(validationBytes) ||
		review.Reviewer == "" || review.ReviewedAt == "" || review.Rubric == "" || review.Summary == "" || review.Issues == nil ||
		(review.Decision != "approved" && review.Decision != "rejected") ||
		(review.Rating != "A" && review.Rating != "B" && review.Rating != "C" && review.Rating != "D") {
		return "invalid", nil
	}
	if review.Decision == "rejected" {
		return "rejected", nil
	}
	return "approved", nil
}

func validateUnifiedPromotionEvidence(root, batchDir string, catalog *Catalog, unit *TranslationUnit, manifest RetranslationBatchUnit, validation *RetranslationValidation, glossary *Glossary, candidate []byte, locale string) (int, error) {
	input, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(manifest.InputPath)))
	if err != nil || sum(input) != manifest.InputSHA256 {
		return 0, fmt.Errorf("%s: input_sha256 mismatch", unit.ID)
	}
	protected, err := prepareTranslationUnitInput(unit, glossary)
	if err != nil || !bytes.Equal(input, []byte(protected.Text)) || len(protected.Tokens) != manifest.ProtectedTokenCount {
		return 0, fmt.Errorf("%s: regenerated protected input differs from saved input", unit.ID)
	}
	extension := filepath.Ext(filepath.Base(manifest.InputPath))
	flatID := strings.TrimSuffix(filepath.Base(manifest.InputPath), extension)
	attempt, err := retryValidationAttemptForExtension(validation.RawResponsePath, flatID, extension)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", unit.ID, err)
	}
	retryDir := filepath.Join(batchDir, "retries", flatID)
	if attempt == 1 {
		entries, readErr := os.ReadDir(retryDir)
		if readErr != nil && !os.IsNotExist(readErr) {
			return 0, fmt.Errorf("%s: inspect retry history: %w", unit.ID, readErr)
		}
		if readErr == nil && len(entries) != 0 {
			return 0, fmt.Errorf("%s: validation points to attempt-001 but retry history exists", unit.ID)
		}
	} else if err := validateRetryAttemptSequenceForExtension(retryDir, attempt, attempt, extension); err != nil {
		return 0, fmt.Errorf("%s: invalid final retry provenance: %w", unit.ID, err)
	}
	raw, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(validation.RawResponsePath)))
	if err != nil {
		return 0, fmt.Errorf("%s: read validation raw response: %w", unit.ID, err)
	}
	restored, failures := protected.restore(string(raw))
	if len(failures) != 0 || !bytes.Equal([]byte(restored), candidate) {
		return 0, fmt.Errorf("%s: restored candidate does not match saved candidate", unit.ID)
	}
	if bytes.Contains(candidate, []byte("GTI18N")) || containsProtectedTranslationToken(candidate) {
		return 0, fmt.Errorf("%s: candidate contains GTI18N token", unit.ID)
	}
	if err := ValidateTranslationUnitCandidate(root, catalog, unit.ID, locale, candidate); err != nil {
		return 0, fmt.Errorf("%s: canonical validation: %w", unit.ID, err)
	}
	return attempt, nil
}

func canonicalizeCandidateEOF(candidate []byte) []byte {
	canonical := bytes.TrimRight(candidate, "\n")
	out := make([]byte, len(canonical)+1)
	copy(out, canonical)
	out[len(out)-1] = '\n'
	return out
}

func validatePromotionEvidence(batchDir string, page Page, manifest RetranslationBatchUnit, validation *RetranslationValidation, glossary *Glossary, candidate []byte) error {
	name := flattenedPageArticleName(page.ID)
	wantInputPath := filepath.ToSlash(filepath.Join("inputs", name))
	if manifest.InputPath != wantInputPath {
		return fmt.Errorf("%s: input_path %q, want %q", page.ID, manifest.InputPath, wantInputPath)
	}
	savedInput, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(manifest.InputPath)))
	if err != nil {
		return fmt.Errorf("%s: read saved input: %w", page.ID, err)
	}
	if sum(savedInput) != manifest.InputSHA256 {
		return fmt.Errorf("%s: input_sha256 mismatch", page.ID)
	}
	protected := prepareDefaultTranslationInput(page.Source, page.SourceSHA256, glossary)
	if !bytes.Equal([]byte(protected.Text), savedInput) {
		return fmt.Errorf("%s: regenerated Default protected input differs from saved input", page.ID)
	}
	if len(protected.Tokens) != manifest.ProtectedTokenCount {
		return fmt.Errorf("%s: protected_token_count %d, regenerated %d", page.ID, manifest.ProtectedTokenCount, len(protected.Tokens))
	}
	flatID := strings.TrimSuffix(name, ".article")
	currentAttempt, err := retryValidationAttempt(validation.RawResponsePath, flatID)
	if err != nil {
		return fmt.Errorf("%s: %w", page.ID, err)
	}
	retryDir := filepath.Join(batchDir, "retries", flatID)
	if currentAttempt == 1 {
		entries, err := os.ReadDir(retryDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("%s: inspect retry history: %w", page.ID, err)
		}
		if err == nil && len(entries) != 0 {
			return fmt.Errorf("%s: validation points to attempt-001 but retry history exists", page.ID)
		}
	} else if err := validateRetryAttemptSequence(retryDir, currentAttempt, currentAttempt); err != nil {
		return fmt.Errorf("%s: invalid final retry provenance: %w", page.ID, err)
	}
	raw, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(validation.RawResponsePath)))
	if err != nil {
		return fmt.Errorf("%s: read validation raw response: %w", page.ID, err)
	}
	restored, failures := protected.restore(string(raw))
	if len(failures) != 0 {
		return fmt.Errorf("%s: validation raw response restore failed: %s", page.ID, strings.Join(failures, "; "))
	}
	if !bytes.Equal([]byte(restored), candidate) {
		return fmt.Errorf("%s: restored candidate does not match saved candidate", page.ID)
	}
	return nil
}

func readPromotionResult(batchDir, locale, batchID string, unitCount int) (*RetranslationProcessResult, error) {
	b, err := os.ReadFile(filepath.Join(batchDir, "result.json"))
	if err != nil {
		return nil, fmt.Errorf("read retranslation result for %q: %w", batchID, err)
	}
	result, err := decodeRetranslationProcessResult(b)
	if err != nil {
		return nil, fmt.Errorf("parse retranslation result for %q: %w", batchID, err)
	}
	if result.SchemaVersion != retranslationProcessSchemaVersion || result.BatchID != batchID || result.Locale != locale || result.UnitCount != unitCount || len(result.Units) != unitCount {
		return nil, fmt.Errorf("retranslation batch %q has incompatible process result", batchID)
	}
	return result, nil
}

func readPromotionValidation(batchDir, batchID, locale string, manifest RetranslationBatchUnit, result RetranslationUnitResult) (*RetranslationValidation, error) {
	b, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(result.ValidationPath)))
	if err != nil {
		return nil, fmt.Errorf("%s: read promotion validation: %w", manifest.UnitID, err)
	}
	unit := &TranslationUnit{ID: manifest.UnitID, Kind: manifest.UnitKind, SourceSHA256: manifest.SourceSHA256}
	validation, err := decodeRetranslationValidation(b, unit)
	if err != nil {
		return nil, fmt.Errorf("%s: parse promotion validation: %w", manifest.UnitID, err)
	}
	if validation.SchemaVersion != retranslationProcessSchemaVersion || validation.BatchID != batchID || validation.Locale != locale || validation.UnitID != manifest.UnitID || validation.UnitKind != manifest.UnitKind || validation.InputPath != manifest.InputPath || validation.Status != result.Status || validation.CandidatePath != result.CandidatePath {
		return nil, fmt.Errorf("%s: validation does not match manifest/result.json", manifest.UnitID)
	}
	return validation, nil
}

func applyRetranslationPromotion(root string, catalog *Catalog, options RetranslationPromoteOptions, prepared []preparedPromotion) error {
	localeDir := filepath.Join(root, "locales", options.Locale)
	statusPath := filepath.Join(localeDir, "status.tsv")
	statuses, err := ReadStatuses(statusPath)
	if err != nil {
		return fmt.Errorf("read canonical status: %w", err)
	}
	allowStaleSource := make(map[string]bool, len(prepared))
	for _, item := range prepared {
		if item.unit == nil || item.unit.ID != item.plan.UnitID || item.unit.SourceSHA256 == "" || sum(item.unit.Source) != item.unit.SourceSHA256 {
			return fmt.Errorf("invalid current-source promotion preparation for %q", item.plan.UnitID)
		}
		allowStaleSource[item.unit.ID] = true
	}
	if err := checkPromotionStatus(root, options.Locale, catalog, statuses, allowStaleSource); err != nil {
		return fmt.Errorf("check canonical status: %w", err)
	}
	statusByID := map[string]int{}
	for i, status := range statuses {
		if _, exists := statusByID[status.UnitID]; exists {
			return fmt.Errorf("duplicate status unit_id %q", status.UnitID)
		}
		statusByID[status.UnitID] = i
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
		i, ok := statusByID[item.plan.UnitID]
		if !ok {
			return fmt.Errorf("status missing unit_id %q", item.plan.UnitID)
		}
		statuses[i].State = "ready"
		statuses[i].Attempts = item.attempt
		statuses[i].SourceSHA256 = item.unit.SourceSHA256
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

// checkPromotionStatus retains CheckStatus's complete locale/status validation
// while allowing only preflighted units to carry an old source hash until this
// atomic promotion installs their current-source candidate and status row.
func checkPromotionStatus(root, locale string, catalog *Catalog, statuses []Status, allowStaleSource map[string]bool) error {
	shadow := *catalog
	shadow.Pages = append([]Page(nil), catalog.Pages...)
	shadow.Examples = append([]Example(nil), catalog.Examples...)
	pageIndex := make(map[string]int, len(shadow.Pages))
	exampleIndex := make(map[string]int, len(shadow.Examples))
	for i := range shadow.Pages {
		pageIndex[shadow.Pages[i].ID] = i
	}
	for i := range shadow.Examples {
		exampleIndex[shadow.Examples[i].ID] = i
	}
	for _, status := range statuses {
		unit, err := catalog.Unit(status.UnitID)
		if err != nil {
			return err
		}
		if status.SourceSHA256 == unit.SourceSHA256 {
			continue
		}
		if !allowStaleSource[status.UnitID] {
			return fmt.Errorf("%s: stale source_sha256 without current-source promotion evidence", status.UnitID)
		}
		switch unit.Kind {
		case UnitKindPage:
			shadow.Pages[pageIndex[unit.ID]].SourceSHA256 = status.SourceSHA256
		case UnitKindExample:
			shadow.Examples[exampleIndex[unit.ID]].SourceSHA256 = status.SourceSHA256
		default:
			return fmt.Errorf("%s: unsupported promotion unit kind %q", unit.ID, unit.Kind)
		}
	}
	return CheckStatus(root, locale, &shadow)
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
