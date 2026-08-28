package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
	return regexp.MustCompile(`^(?:chatgpt|codex)-(` + regexp.QuoteMeta(locale) + `)-([0-9]+)$`)
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

func preflightUnifiedRetranslationPromotion(root string, catalog *Catalog, locale string) ([]preparedPromotion, []string, promotionReviewGate, int, int, error) {
	glossary, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, nil, promotionReviewGate{}, 0, 0, err
	}
	latest, err := selectLatestRetranslationUnits(root, catalog, locale)
	if err != nil {
		return nil, nil, promotionReviewGate{}, 0, 0, err
	}
	prepared := make([]preparedPromotion, 0, len(latest.ordered))
	missing := make([]string, 0)
	reviews := promotionReviewGate{missing: []string{}, rejected: []string{}, invalid: []string{}}
	for _, unit := range latest.ordered {
		id := unit.ID
		choice, ok := latest.selectedByID[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		if !selectedRetranslationIdentityMatches(unit, choice) {
			missing = append(missing, id)
			continue
		}
		if choice.result.Status != "passed" {
			missing = append(missing, fmt.Sprintf("%s (latest %s=%s)", id, choice.batchID, choice.result.Status))
			continue
		}
		evidence, err := validateSelectedRetranslationCandidate(root, catalog, locale, glossary, unit, choice)
		if err != nil {
			return nil, nil, promotionReviewGate{}, 0, 0, err
		}
		reviewState, err := checkPromotionReview(choice.batchDir, choice.batchID, locale, unit, choice.manifest, choice.result, evidence.validation, evidence.candidate, evidence.attempt)
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
		canonicalCandidate := evidence.candidate
		if unit.Kind == UnitKindPage {
			canonicalCandidate = canonicalizeCandidateEOF(evidence.candidate)
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
				SourceCandidatePath:    filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, choice.batchID, evidence.candidatePath)),
				CanonicalCandidatePath: canonicalPath, SourceCandidateSHA256: sum(evidence.candidate), CandidateSHA256: sum(canonicalCandidate),
				EOFNormalized: !bytes.Equal(evidence.candidate, canonicalCandidate), Changed: changed,
			},
			candidate: canonicalCandidate, unit: unit, attempt: evidence.attempt,
		})
	}
	return prepared, missing, reviews, latest.pageCount, latest.exampleCount, nil
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
		review.Reviewer == "" || review.ReviewedAt == "" || review.Rubric != TranslationQualityRubric || review.Summary == "" || review.Issues == nil ||
		(review.Decision != "approved" && review.Decision != "rejected") ||
		(review.Rating != "A" && review.Rating != "B" && review.Rating != "C" && review.Rating != "D") {
		return "invalid", nil
	}
	if review.Decision != "approved" || review.Rating != "A" {
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
	if err != nil || (!bytes.Equal(input, []byte(protected.Text)) && !bytes.Equal(input, canonicalizeRetranslationArtifactEOF([]byte(protected.Text)))) || len(protected.Tokens) != manifest.ProtectedTokenCount {
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
	if validation.Attempt != attempt {
		return 0, fmt.Errorf("%s: validation attempt %d does not match final raw response attempt %d", unit.ID, validation.Attempt, attempt)
	}
	raw, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(validation.RawResponsePath)))
	if err != nil {
		return 0, fmt.Errorf("%s: read validation raw response: %w", unit.ID, err)
	}
	restored, failures := protected.restore(string(raw))
	restoredBytes := []byte(restored)
	if len(failures) != 0 || (!bytes.Equal(restoredBytes, candidate) && !bytes.Equal(canonicalizeRetranslationArtifactEOF(restoredBytes), candidate)) {
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
	return canonicalizeRetranslationArtifactEOF(candidate)
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
	if validation.SchemaVersion != retranslationProcessSchemaVersion || validation.BatchID != batchID || validation.Locale != locale || validation.UnitID != manifest.UnitID || validation.UnitKind != manifest.UnitKind || validation.SourceSHA256 != manifest.SourceSHA256 || validation.InputPath != manifest.InputPath || validation.Status != result.Status || validation.CandidatePath != result.CandidatePath {
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
