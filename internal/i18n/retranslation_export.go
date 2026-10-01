package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	// Retranslation export batches are always kind-homogeneous and use the same
	// default and hard limit for automatic, explicit, and revision selection.
	DefaultRetranslationExportLimit = 60
	MaxRetranslationExportLimit     = 60

	RetranslationReexportReasonGlossaryInputStale = "glossary_input_stale"
)

type RetranslationGenerator string

const (
	RetranslationGeneratorCodex   RetranslationGenerator = "codex"
	RetranslationGeneratorChatGPT RetranslationGenerator = "chatgpt"
)

type RetranslationExportOptions struct {
	Locale             string
	BatchID            string
	Generator          RetranslationGenerator
	UnitIDs            []string
	UnitKind           UnitKind
	Limit              int
	AllowReexport      bool
	PreviousSnapshotID string
	SurfaceReopenID    string
	GlossaryStale      bool
}

type RetranslationBatchUnit struct {
	UnitID                  string   `json:"unit_id"`
	UnitKind                UnitKind `json:"unit_kind"`
	SourcePath              string   `json:"source_path"`
	SourceSHA256            string   `json:"source_sha256"`
	InputPath               string   `json:"input_path"`
	InputSHA256             string   `json:"input_sha256"`
	ProtectedTokenCount     int      `json:"protected_token_count"`
	PreviousSnapshotID      string   `json:"previous_snapshot_id,omitempty"`
	RevisionFeedbackSource  string   `json:"revision_feedback_source,omitempty"`
	RevisionAuthorizationID string   `json:"revision_authorization_id,omitempty"`
	PreviousRating          string   `json:"previous_rating,omitempty"`
	PreviousFinding         string   `json:"previous_finding,omitempty"`
	PreviousReviewDecision  string   `json:"previous_review_decision,omitempty"`
	PreviousReviewSummary   string   `json:"previous_review_summary,omitempty"`
	PreviousReviewIssues    []string `json:"previous_review_issues,omitempty"`
	PreviousReviewPath      string   `json:"previous_review_path,omitempty"`
	PreviousReviewSHA256    string   `json:"previous_review_sha256,omitempty"`
	ReexportReason          string   `json:"reexport_reason,omitempty"`
	PreviousBatchID         string   `json:"previous_batch_id,omitempty"`
	PreviousInputPath       string   `json:"previous_input_path,omitempty"`
	PreviousInputSHA256     string   `json:"previous_input_sha256,omitempty"`
	PreviousProtectedTokens int      `json:"previous_protected_token_count,omitempty"`
}

type RetranslationBatchManifest struct {
	SchemaVersion  int                      `json:"schema_version"`
	BatchID        string                   `json:"batch_id"`
	Locale         string                   `json:"locale"`
	ProtectionMode string                   `json:"protection_mode"`
	ArtifactEOF    string                   `json:"artifact_eof,omitempty"`
	UnitKind       UnitKind                 `json:"unit_kind"`
	UnitCount      int                      `json:"unit_count"`
	Units          []RetranslationBatchUnit `json:"units"`
}

type RetranslationExportResult struct {
	Locale      string   `json:"locale"`
	BatchID     string   `json:"batch_id,omitempty"`
	BatchPath   string   `json:"batch_path,omitempty"`
	UnitKind    UnitKind `json:"unit_kind,omitempty"`
	UnitCount   int      `json:"unit_count"`
	UnitIDs     []string `json:"unit_ids,omitempty"`
	AllExported bool     `json:"all_exported"`
}

type preparedRetranslationInput struct {
	unit   *TranslationUnit
	text   string
	path   string
	hash   string
	tokens int
}

type exportedRetranslationUnit struct {
	BatchID      string
	SourceSHA256 string
	batchNumber  int
	numbered     bool
}

type retranslationStatus struct {
	StaleSource bool
	ReadySource bool
}

type revisionFeedback struct {
	source       string
	rating       string
	finding      string
	decision     string
	summary      string
	issues       []string
	reviewPath   string
	reviewSHA256 string
}

type glossaryStaleProvenance struct {
	batchID             string
	inputPath           string
	inputSHA256         string
	protectedTokenCount int
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
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	generator := options.Generator
	if generator == "" {
		generator = RetranslationGeneratorCodex
	}
	if err := ValidateRetranslationGenerator(generator); err != nil {
		return nil, err
	}
	if options.UnitKind != "" && options.UnitKind != UnitKindPage && options.UnitKind != UnitKindExample {
		return nil, fmt.Errorf("不支持的翻译单元类型 %q；只支持 page 或 example", options.UnitKind)
	}
	if options.AllowReexport && len(options.UnitIDs) == 0 {
		return nil, errors.New("--allow-reexport requires at least one --id")
	}
	if options.GlossaryStale && !options.AllowReexport {
		return nil, errors.New("--glossary-stale requires --allow-reexport")
	}
	if options.GlossaryStale && (options.PreviousSnapshotID != "" || options.SurfaceReopenID != "") {
		return nil, errors.New("--glossary-stale cannot be combined with Quality Check or Surface Review revision authorization")
	}
	if options.PreviousSnapshotID != "" && !options.AllowReexport {
		return nil, errors.New("--previous-snapshot-id requires --allow-reexport revision mode")
	}
	if options.SurfaceReopenID != "" && (options.PreviousSnapshotID == "" || !options.AllowReexport) {
		return nil, errors.New("--surface-reopen-id requires --allow-reexport and --previous-snapshot-id")
	}
	limit := options.Limit
	if limit == 0 {
		limit = DefaultRetranslationExportLimit
	}
	if limit < 1 {
		return nil, errors.New("retranslation export limit must be greater than zero")
	}
	if limit > MaxRetranslationExportLimit {
		return nil, fmt.Errorf("retranslation export limit must not exceed %d", MaxRetranslationExportLimit)
	}
	if len(options.UnitIDs) > MaxRetranslationExportLimit {
		return nil, fmt.Errorf("a retranslation batch must not contain more than %d TranslationUnits", MaxRetranslationExportLimit)
	}
	if err := RequireCurrentGlossaryReview(root, options.Locale); err != nil {
		return nil, fmt.Errorf("retranslation export requires current Glossary Review coverage: %w", err)
	}

	base := filepath.Join(root, "data", "retranslation-runs", options.Locale)
	exported, nextNumber, err := scanRetranslationBatches(base, options.Locale, catalog)
	if err != nil {
		return nil, err
	}
	statuses, err := retranslationStatuses(root, options.Locale, catalog)
	if err != nil {
		return nil, err
	}
	units, err := selectRetranslationUnits(catalog, options.UnitIDs, options.UnitKind, exported, statuses, limit, options.AllowReexport)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return &RetranslationExportResult{Locale: options.Locale, AllExported: true}, nil
	}
	revisionFeedbackByID := map[string]revisionFeedback{}
	var surfaceReopen *QualityCheckSurfaceReopen
	if options.SurfaceReopenID != "" {
		surfaceReopen, err = readCurrentQualityCheckSurfaceReopen(root, catalog, options.Locale, options.SurfaceReopenID)
		if err != nil {
			return nil, err
		}
		if surfaceReopen.PreviousSnapshotID != options.PreviousSnapshotID {
			return nil, errors.New("surface reopen predecessor does not match --previous-snapshot-id")
		}
		want := map[string]bool{}
		for _, unit := range surfaceReopen.Units {
			want[unit.UnitID] = true
		}
		if len(want) != len(units) {
			return nil, errors.New("revision unit set must exactly match surface reopen scope")
		}
		for _, unit := range units {
			if !want[unit.ID] {
				return nil, errors.New("revision unit set must exactly match surface reopen scope")
			}
		}
	}
	if options.PreviousSnapshotID != "" {
		snapshot, err := readQualityCheckSnapshotForReview(root, options.Locale, options.PreviousSnapshotID)
		if err != nil {
			return nil, fmt.Errorf("previous Quality Check Snapshot: %w", err)
		}
		effectiveSnapshot, effectiveResults, err := loadEffectiveQualityCheckResults(root, options.Locale, options.PreviousSnapshotID, map[string]bool{})
		if err != nil {
			return nil, err
		}
		if effectiveSnapshot.SnapshotID != snapshot.SnapshotID || effectiveSnapshot.GlossarySHA256 != snapshot.GlossarySHA256 {
			return nil, errors.New("effective Quality Check Snapshot identity mismatch")
		}
		byID := map[string]QualityCheckSnapshotUnit{}
		for _, u := range snapshot.Units {
			byID[u.UnitID] = u
		}
		for _, unit := range units {
			snapshotUnit, ok := byID[unit.ID]
			if !ok {
				return nil, fmt.Errorf("revision unit %s is absent from previous Snapshot", unit.ID)
			}
			if _, err := readSnapshotUnitRepositoryEvidence(root, catalog, options.Locale, snapshotUnit); err != nil {
				return nil, fmt.Errorf("previous Snapshot unit %s identity: %w", unit.ID, err)
			}
			result, ok := effectiveResults[unit.ID]
			if !ok {
				return nil, fmt.Errorf("revision unit %s has no effective Quality Check result in previous Snapshot %s", unit.ID, options.PreviousSnapshotID)
			}
			if result.rubric != TranslationQualityRubric || !qualityCheckSnapshotIdentityMatches(snapshotUnit, result.unit) {
				return nil, fmt.Errorf("revision unit %s has no current, identity-matching effective Quality Check result in previous Snapshot %s", unit.ID, options.PreviousSnapshotID)
			}
			if result.rating != "A" {
				if surfaceReopen != nil {
					return nil, fmt.Errorf("surface reopen unit %s is already non-A and must use ordinary Quality Check revision feedback", unit.ID)
				}
				if strings.TrimSpace(result.finding) == "" {
					return nil, fmt.Errorf("revision unit %s has no finding in previous Snapshot %s; backfill it first", unit.ID, options.PreviousSnapshotID)
				}
				revisionFeedbackByID[unit.ID] = revisionFeedback{source: "quality_check", rating: result.rating, finding: result.finding}
				continue
			}
			if surfaceReopen != nil {
				revisionFeedbackByID[unit.ID] = revisionFeedback{source: "surface_review", rating: "A", finding: surfaceReopen.Finding, decision: "reopened", summary: surfaceReopen.ReopenID, reviewPath: surfaceReopen.SurfaceReviewPath, reviewSHA256: surfaceReopen.SurfaceReviewSHA256}
				continue
			}
			return nil, fmt.Errorf("revision unit %s is not eligible from previous Snapshot %s: Quality Check is already A; new revision export requires Quality Check B/C/D feedback", unit.ID, options.PreviousSnapshotID)
		}
	}

	batchID := options.BatchID
	if batchID == "" {
		batchID = fmt.Sprintf("%s-%s-%03d", generator, options.Locale, nextNumber)
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
	var latest *latestRetranslationUnits
	if options.GlossaryStale {
		latest, err = selectLatestRetranslationUnits(root, catalog, options.Locale)
		if err != nil {
			return nil, fmt.Errorf("glossary-stale recovery latest candidate selection: %w", err)
		}
	}
	prepared := make([]preparedRetranslationInput, 0, len(units))
	glossaryStaleByID := map[string]glossaryStaleProvenance{}
	for _, unit := range units {
		if sum(unit.Source) != unit.SourceSHA256 {
			return nil, fmt.Errorf("%s: hydrated source hash mismatch", unit.ID)
		}
		if unit.Kind == UnitKindExample {
			hasContent, err := hasTranslatableGoExampleComment(unit.Source)
			if err != nil {
				return nil, fmt.Errorf("%s: 检查可翻译自然语言注释: %w", unit.ID, err)
			}
			if !hasContent {
				return nil, fmt.Errorf("示例翻译单元 %s 没有需要翻译的普通自然语言注释", unit.ID)
			}
		}
		protected, err := prepareTranslationUnitInput(unit, glossary)
		if err != nil {
			return nil, fmt.Errorf("%s: 准备受保护输入: %w", unit.ID, err)
		}
		inputPath := filepath.ToSlash(filepath.Join("inputs", retranslationUnitInputName(unit)))
		input := canonicalizeRetranslationArtifactEOF([]byte(protected.Text))
		if options.GlossaryStale {
			provenance, err := validateGlossaryStaleRetranslationUnit(root, options.Locale, unit, input, latest)
			if err != nil {
				return nil, err
			}
			glossaryStaleByID[unit.ID] = provenance
		}
		prepared = append(prepared, preparedRetranslationInput{
			unit: unit, text: string(input), path: inputPath,
			hash: sum(input), tokens: len(protected.Tokens),
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
		SchemaVersion: 2, BatchID: batchID, Locale: options.Locale,
		ProtectionMode: "default", ArtifactEOF: retranslationArtifactEOFSingleLF, UnitKind: prepared[0].unit.Kind,
		UnitCount: len(prepared), Units: make([]RetranslationBatchUnit, 0, len(prepared)),
	}
	unitIDs := make([]string, 0, len(prepared))
	for _, input := range prepared {
		if err := os.WriteFile(filepath.Join(staging, filepath.FromSlash(input.path)), []byte(input.text), 0644); err != nil {
			return nil, fmt.Errorf("write retranslation input for %s: %w", input.unit.ID, err)
		}
		record := RetranslationBatchUnit{
			UnitID: input.unit.ID, UnitKind: input.unit.Kind, SourcePath: input.unit.SourcePath,
			SourceSHA256: input.unit.SourceSHA256, InputPath: input.path,
			InputSHA256: input.hash, ProtectedTokenCount: input.tokens,
		}
		if prior, ok := revisionFeedbackByID[input.unit.ID]; ok {
			record.PreviousSnapshotID = options.PreviousSnapshotID
			record.RevisionFeedbackSource = prior.source
			record.RevisionAuthorizationID = options.SurfaceReopenID
			record.PreviousRating = prior.rating
			record.PreviousFinding = prior.finding
			record.PreviousReviewDecision = prior.decision
			record.PreviousReviewSummary = prior.summary
			record.PreviousReviewIssues = append([]string(nil), prior.issues...)
			record.PreviousReviewPath = prior.reviewPath
			record.PreviousReviewSHA256 = prior.reviewSHA256
		}
		if prior, ok := glossaryStaleByID[input.unit.ID]; ok {
			record.ReexportReason = RetranslationReexportReasonGlossaryInputStale
			record.PreviousBatchID = prior.batchID
			record.PreviousInputPath = prior.inputPath
			record.PreviousInputSHA256 = prior.inputSHA256
			record.PreviousProtectedTokens = prior.protectedTokenCount
		}
		manifest.Units = append(manifest.Units, record)
		unitIDs = append(unitIDs, input.unit.ID)
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
		UnitKind: prepared[0].unit.Kind, UnitCount: len(unitIDs), UnitIDs: unitIDs,
	}, nil
}

func validateGlossaryStaleRetranslationUnit(root, locale string, unit *TranslationUnit, currentInput []byte, latest *latestRetranslationUnits) (glossaryStaleProvenance, error) {
	if latest == nil {
		return glossaryStaleProvenance{}, errors.New("glossary-stale recovery latest candidate selection is required")
	}
	choice, ok := latest.selectedByID[unit.ID]
	if !ok {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: glossary-stale recovery requires a latest processed result", unit.ID)
	}
	if !selectedRetranslationIdentityMatches(unit, choice) {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: latest processed batch %s source identity does not match current Catalog", unit.ID, choice.batchID)
	}
	if choice.result.Status != "passed" {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: latest processed batch %s status %q is not passed", unit.ID, choice.batchID, choice.result.Status)
	}
	if choice.artifactEOF != retranslationArtifactEOFSingleLF {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: latest processed batch %s does not have an unambiguous single-LF protected-input identity", unit.ID, choice.batchID)
	}
	wantInputPath := filepath.ToSlash(filepath.Join("inputs", retranslationUnitInputName(unit)))
	if choice.manifest.InputPath != wantInputPath {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: latest processed batch %s has non-canonical input_path %q", unit.ID, choice.batchID, choice.manifest.InputPath)
	}
	savedInput, err := os.ReadFile(filepath.Join(choice.batchDir, filepath.FromSlash(choice.manifest.InputPath)))
	if err != nil {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: read latest saved protected input: %w", unit.ID, err)
	}
	if sum(savedInput) != choice.manifest.InputSHA256 {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: latest saved input hash does not match manifest", unit.ID)
	}
	tokens := translationTokenRE.FindAll(savedInput, -1)
	seenTokens := map[string]bool{}
	for _, token := range tokens {
		value := string(token)
		if seenTokens[value] {
			return glossaryStaleProvenance{}, fmt.Errorf("%s: latest saved input contains duplicate protected token %s", unit.ID, value)
		}
		seenTokens[value] = true
	}
	if len(tokens) != choice.manifest.ProtectedTokenCount {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: latest saved input protected token count %d does not match manifest %d", unit.ID, len(tokens), choice.manifest.ProtectedTokenCount)
	}
	name := filepath.Base(filepath.FromSlash(choice.manifest.InputPath))
	wantCandidate := filepath.ToSlash(filepath.Join("candidates", retranslationUnitCandidateName(unit)))
	wantValidation := filepath.ToSlash(filepath.Join("validation", strings.TrimSuffix(name, filepath.Ext(name))+".json"))
	if choice.result.CandidatePath != wantCandidate || choice.result.ValidationPath != wantValidation {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: latest processed result candidate/validation path mismatch", unit.ID)
	}
	validation, err := readPromotionValidation(choice.batchDir, choice.batchID, locale, choice.manifest, choice.result)
	if err != nil {
		return glossaryStaleProvenance{}, err
	}
	if validation.Status != "passed" {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: latest validation status %q is not passed", unit.ID, validation.Status)
	}
	if _, err := os.ReadFile(filepath.Join(choice.batchDir, filepath.FromSlash(wantCandidate))); err != nil {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: read latest candidate: %w", unit.ID, err)
	}
	if bytes.Equal(savedInput, currentInput) {
		return glossaryStaleProvenance{}, fmt.Errorf("%s: current protected input has no glossary-induced drift from latest processed batch %s", unit.ID, choice.batchID)
	}
	return glossaryStaleProvenance{
		batchID: choice.batchID, inputPath: choice.manifest.InputPath,
		inputSHA256: choice.manifest.InputSHA256, protectedTokenCount: choice.manifest.ProtectedTokenCount,
	}, nil
}

func readFinalReviewRevisionFeedback(root, locale string, snapshot QualityCheckSnapshotUnit, evidence *snapshotUnitRepositoryEvidence) (revisionFeedback, error) {
	reviewPath := filepath.Join(evidence.batchDir, "review", retranslationReviewName(evidence.unit))
	reviewData, err := os.ReadFile(reviewPath)
	if os.IsNotExist(err) {
		return revisionFeedback{}, errors.New("Final Review evidence is missing")
	}
	if err != nil {
		return revisionFeedback{}, fmt.Errorf("read Final Review evidence: %w", err)
	}
	review, err := decodeTranslationReview(reviewData)
	if err != nil {
		return revisionFeedback{}, fmt.Errorf("invalid Final Review evidence: %w", err)
	}
	if !reviewMatchesSnapshotIdentity(locale, snapshot, *review) {
		return revisionFeedback{}, errors.New("Final Review identity does not match Candidate Snapshot")
	}
	if review.Rubric != TranslationQualityRubric {
		return revisionFeedback{}, errors.New("Final Review rubric is not current")
	}
	if (review.Rating != "B" && review.Rating != "C" && review.Rating != "D") || review.Decision != "rejected" {
		return revisionFeedback{}, fmt.Errorf("Final Review must be rated B, C, or D with decision rejected (got %s + %s)", review.Rating, review.Decision)
	}
	repositoryPath, err := repositoryRelativePath(root, reviewPath)
	if err != nil {
		return revisionFeedback{}, err
	}
	return revisionFeedback{
		source: "final_review", rating: review.Rating, decision: review.Decision,
		summary: review.Summary, issues: append([]string(nil), review.Issues...),
		reviewPath: repositoryPath, reviewSHA256: sum(reviewData),
	}, nil
}

func selectRetranslationUnits(catalog *Catalog, requested []string, requestedKind UnitKind, exported map[string]exportedRetranslationUnit, statuses map[string]retranslationStatus, limit int, allowReexport bool) ([]*TranslationUnit, error) {
	if len(requested) != 0 {
		seen := map[string]bool{}
		units := make([]*TranslationUnit, 0, len(requested))
		var kind UnitKind
		for _, unitID := range requested {
			if seen[unitID] {
				return nil, fmt.Errorf("duplicate requested translation unit %q", unitID)
			}
			seen[unitID] = true
			unit, err := catalog.Unit(unitID)
			if err != nil {
				return nil, err
			}
			if kind != "" && unit.Kind != kind {
				return nil, errors.New("一个重译批次不能混合课程页面单元和示例单元")
			}
			if requestedKind != "" && unit.Kind != requestedKind {
				return nil, fmt.Errorf("翻译单元 %s 的类型为 %s，与 --unit-kind %s 不一致", unit.ID, unit.Kind, requestedKind)
			}
			kind = unit.Kind
			if history := exported[unitID]; history.BatchID != "" && !allowReexport {
				return nil, fmt.Errorf("translation unit %q was already exported in batch %q", unitID, history.BatchID)
			}
			units = append(units, unit)
		}
		return units, nil
	}
	if requestedKind == UnitKindExample {
		units := make([]*TranslationUnit, 0, limit)
		for i := range catalog.Examples {
			example := &catalog.Examples[i]
			hasContent, err := hasTranslatableGoExampleComment(example.Source)
			if err != nil {
				return nil, fmt.Errorf("%s: 检查可翻译自然语言注释: %w", example.ID, err)
			}
			if !hasContent {
				continue
			}
			unit, err := catalog.Unit(example.ID)
			if err != nil {
				return nil, err
			}
			if alreadyExportedForCurrentSource(exported[example.ID], unit, statuses[example.ID]) {
				continue
			}
			units = append(units, unit)
			if len(units) == limit {
				break
			}
		}
		return units, nil
	}
	units := make([]*TranslationUnit, 0, limit)
	for _, page := range catalog.Pages {
		unit, err := catalog.Unit(page.ID)
		if err != nil {
			return nil, err
		}
		if alreadyExportedForCurrentSource(exported[page.ID], unit, statuses[page.ID]) {
			continue
		}
		units = append(units, unit)
		if len(units) == limit {
			break
		}
	}
	return units, nil
}

// alreadyExportedForCurrentSource preserves the normal exported-unit guard,
// but a formal status tied to an older source version needs one fresh batch.
func alreadyExportedForCurrentSource(history exportedRetranslationUnit, unit *TranslationUnit, status retranslationStatus) bool {
	if status.ReadySource {
		return true
	}
	if history.BatchID == "" {
		return false
	}
	return !status.StaleSource || history.SourceSHA256 == unit.SourceSHA256
}

func retranslationStatuses(root, locale string, catalog *Catalog) (map[string]retranslationStatus, error) {
	result := map[string]retranslationStatus{}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", locale, "status.tsv"))
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read formal status for retranslation export: %w", err)
	}
	for _, status := range statuses {
		unit, err := catalog.Unit(status.UnitID)
		if err != nil {
			return nil, fmt.Errorf("formal status references unknown translation unit %q", status.UnitID)
		}
		if status.SourceSHA256 != unit.SourceSHA256 {
			result[unit.ID] = retranslationStatus{StaleSource: true}
			continue
		}
		if status.State == "ready" {
			result[unit.ID] = retranslationStatus{ReadySource: true}
		}
	}
	return result, nil
}

func retranslationUnitInputName(unit *TranslationUnit) string {
	if unit.Kind == UnitKindExample {
		name := strings.ReplaceAll(strings.TrimPrefix(unit.ID, "example:"), "/", "-")
		return strings.TrimSuffix(name, filepath.Ext(name)) + ".txt"
	}
	return flattenedPageArticleName(unit.ID)
}

func retranslationUnitCandidateName(unit *TranslationUnit) string {
	if unit.Kind == UnitKindExample {
		return strings.ReplaceAll(strings.TrimPrefix(unit.ID, "example:"), "/", "-")
	}
	return flattenedPageArticleName(unit.ID)
}

func scanRetranslationBatches(base, locale string, catalog *Catalog) (map[string]exportedRetranslationUnit, int, error) {
	exported := map[string]exportedRetranslationUnit{}
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return exported, 1, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("scan retranslation batches: %w", err)
	}
	known := make(map[string]UnitKind, len(catalog.Pages)+len(catalog.Examples))
	for _, page := range catalog.Pages {
		known[page.ID] = UnitKindPage
	}
	for _, example := range catalog.Examples {
		known[example.ID] = UnitKindExample
	}
	nextNumber := 1
	seenNumbers := map[int]string{}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		match := promotionBatchRE(locale).FindStringSubmatch(entry.Name())
		batchNumber := 0
		if match != nil {
			var err error
			batchNumber, err = strconv.Atoi(match[2])
			if err != nil || batchNumber < 1 || seenNumbers[batchNumber] != "" {
				return nil, 0, fmt.Errorf("ambiguous or invalid retranslation batch number %03d", batchNumber)
			}
			seenNumbers[batchNumber] = entry.Name()
			if batchNumber >= nextNumber {
				nextNumber = batchNumber + 1
			}
		}
		manifestPath := filepath.Join(base, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if os.IsNotExist(err) && match == nil {
			continue
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read retranslation manifest %s: %w", filepath.ToSlash(manifestPath), err)
		}
		manifest, err := decodeRetranslationManifest(data)
		if err != nil {
			return nil, 0, fmt.Errorf("parse retranslation manifest %s: %w", filepath.ToSlash(manifestPath), err)
		}
		if manifest.BatchID != entry.Name() {
			return nil, 0, fmt.Errorf("retranslation manifest batch_id %q does not match directory %q", manifest.BatchID, entry.Name())
		}
		if manifest.Locale != locale {
			return nil, 0, fmt.Errorf("retranslation batch %q locale %q does not match %q", entry.Name(), manifest.Locale, locale)
		}
		if manifest.SchemaVersion != 2 || manifest.ProtectionMode != "default" || !supportedRetranslationArtifactEOFPolicy(manifest.ArtifactEOF) || (manifest.UnitKind != UnitKindPage && manifest.UnitKind != UnitKindExample) {
			return nil, 0, fmt.Errorf("retranslation batch %q has incompatible manifest metadata", entry.Name())
		}
		if manifest.UnitCount == 0 {
			return nil, 0, fmt.Errorf("retranslation batch %q has no translation units", entry.Name())
		}
		if manifest.UnitCount != len(manifest.Units) {
			return nil, 0, fmt.Errorf("retranslation batch %q unit_count %d does not match units %d", entry.Name(), manifest.UnitCount, len(manifest.Units))
		}
		if err := validateRetranslationManifestReexportProvenance(manifest); err != nil {
			return nil, 0, fmt.Errorf("retranslation batch %q provenance: %w", entry.Name(), err)
		}
		for _, record := range manifest.Units {
			unitID, unitKind := record.UnitID, record.UnitKind
			wantKind, ok := known[unitID]
			if !ok {
				return nil, 0, fmt.Errorf("retranslation batch %q has unknown translation unit %q", entry.Name(), unitID)
			}
			if unitKind != wantKind || manifest.UnitKind != unitKind {
				return nil, 0, fmt.Errorf("retranslation batch %q translation unit metadata mismatch for %q", entry.Name(), unitID)
			}
			current, exists := exported[unitID]
			// Formal chatgpt/codex batches share one numeric namespace. Their
			// numeric suffix, not provider-prefix lexical order, decides which
			// export is latest. Preserve the historical lexical behavior when
			// an explicitly named non-formal batch is involved.
			if exists && current.numbered && match != nil && batchNumber < current.batchNumber {
				continue
			}
			exported[unitID] = exportedRetranslationUnit{
				BatchID: entry.Name(), SourceSHA256: record.SourceSHA256,
				batchNumber: batchNumber, numbered: match != nil,
			}
		}
	}
	return exported, nextNumber, nil
}

func ValidateRetranslationGenerator(generator RetranslationGenerator) error {
	switch generator {
	case RetranslationGeneratorCodex, RetranslationGeneratorChatGPT:
		return nil
	default:
		return fmt.Errorf("unsupported retranslation generator %q; use codex or chatgpt", generator)
	}
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
