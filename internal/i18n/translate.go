package i18n

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrMissingTranslationAPIKey = errors.New("ZHIPU_API_KEY is not set")

type TranslationValidation struct {
	Attempt      int      `json:"attempt"`
	APISuccess   bool     `json:"api_success"`
	TokenValid   bool     `json:"token_valid"`
	PresentValid bool     `json:"present_valid"`
	Failures     []string `json:"failures"`
	Passed       bool     `json:"passed"`
}

type TranslationRunResult struct {
	PageID        string                 `json:"page_id"`
	Locale        string                 `json:"locale"`
	SourceSHA256  string                 `json:"source_sha256"`
	Model         string                 `json:"model"`
	Attempts      int                    `json:"attempts"`
	Status        string                 `json:"status"`
	CandidatePath string                 `json:"candidate_path,omitempty"`
	Validation    *TranslationValidation `json:"validation,omitempty"`
	UpdatedAt     string                 `json:"updated_at"`
}

// TranslationRecoveryResult records an explicit, auditable reset of a formal
// translation window after the exhausted window proved to be infrastructure-only.
type TranslationRecoveryResult struct {
	PageID       string `json:"page_id"`
	Locale       string `json:"locale"`
	SourceSHA256 string `json:"source_sha256"`
	Attempts     int    `json:"attempts"`
	Status       string `json:"status"`
	RecoveryPath string `json:"recovery_path"`
	UpdatedAt    string `json:"updated_at"`
}

type TranslationRevalidationResult struct {
	PageID        string                 `json:"page_id"`
	Locale        string                 `json:"locale"`
	SourceSHA256  string                 `json:"source_sha256"`
	SourceAttempt int                    `json:"source_attempt"`
	Attempts      int                    `json:"attempts"`
	Status        string                 `json:"status"`
	CandidatePath string                 `json:"candidate_path,omitempty"`
	AuditPath     string                 `json:"audit_path"`
	Validation    *TranslationValidation `json:"validation"`
	UpdatedAt     string                 `json:"updated_at"`
}

type responseRevalidationRecord struct {
	SchemaVersion   int                   `json:"schema_version"`
	Locale          string                `json:"locale"`
	PageID          string                `json:"page_id"`
	SourceSHA256    string                `json:"source_sha256"`
	SourceAttempt   int                   `json:"source_attempt"`
	ResponsePath    string                `json:"response_path"`
	ValidationPath  string                `json:"validation_path"`
	RevalidatedAt   string                `json:"revalidated_at"`
	Validation      TranslationValidation `json:"validation"`
	CandidateSHA256 string                `json:"candidate_sha256,omitempty"`
	Passed          bool                  `json:"passed"`
	Failures        []string              `json:"failures,omitempty"`
}

type networkFailureRecoveryRecord struct {
	PageID            string `json:"page_id"`
	Locale            string `json:"locale"`
	SourceSHA256      string `json:"source_sha256"`
	RecoveredAttempts []int  `json:"recovered_attempts"`
	PreviousStatus    string `json:"previous_status"`
	PreviousAttempts  int    `json:"previous_attempts"`
	RecoveredAt       string `json:"recovered_at"`
	RecoveryKind      string `json:"recovery_kind"`
}

type savedTranslationRequest struct {
	PageID       string                `json:"page_id"`
	Locale       string                `json:"locale"`
	SourceSHA256 string                `json:"source_sha256"`
	Body         TranslationAPIRequest `json:"body"`
}

type TranslationRunner struct {
	Root        string
	Catalog     *Catalog
	Client      *TranslationClient
	Now         func() time.Time
	MaxAttempts int
	Dev         bool
}

func (r *TranslationRunner) Run(ctx context.Context, pageID, locale, apiKey string) (*TranslationRunResult, error) {
	if apiKey == "" {
		return nil, ErrMissingTranslationAPIKey
	}
	if locale != "zh-CN" {
		return nil, fmt.Errorf("unsupported locale %q", locale)
	}
	if r.Catalog == nil {
		return nil, errors.New("translation catalog is required")
	}
	page, err := r.Catalog.Page(pageID)
	if err != nil {
		return nil, err
	}
	if sum(page.Source) != page.SourceSHA256 {
		return nil, fmt.Errorf("%s: hydrated source hash mismatch", pageID)
	}
	currentStatus, _, err := LoadTranslationResult(r.Root, pageID, locale)
	if err != nil {
		return nil, err
	}
	if !r.Dev && currentStatus.State == "blocked" {
		return nil, fmt.Errorf("%s is blocked; formal translation cannot retry it", pageID)
	}
	glossary, err := LoadGlossary(r.Root, locale)
	if err != nil {
		return nil, err
	}
	protected := protectTranslation(page.Source, page.SourceSHA256, glossary)
	maxAttempts := r.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	client := r.Client
	if client == nil {
		client = NewTranslationClient()
	}
	now := r.Now
	if now == nil {
		now = time.Now
	}
	var last TranslationValidation
	var previous string
	sourceRunDir := filepath.Join(r.Root, "data", "translation-runs", locale, pageID, "sources", page.SourceSHA256)
	firstAttempt, err := nextTranslationAttempt(sourceRunDir)
	if err != nil {
		return nil, err
	}
	windowStart, windowEnd, err := currentFormalAttemptWindow(sourceRunDir, pageID, locale, page.SourceSHA256, maxAttempts)
	if err != nil {
		return nil, err
	}
	if firstAttempt < windowStart {
		return nil, fmt.Errorf("%s: next historical attempt %d precedes current formal window %d-%d", pageID, firstAttempt, windowStart, windowEnd)
	}
	if !r.Dev && firstAttempt > windowEnd {
		updated := now().UTC().Format(time.RFC3339)
		note := fmt.Sprintf("formal attempt window %03d-%03d exhausted before this run", windowStart, windowEnd)
		if err := updateTranslationStatus(r.Root, locale, pageID, "blocked", windowEnd, page.SourceSHA256, "", updated, note); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s formal attempt window %03d-%03d is exhausted", pageID, windowStart, windowEnd)
	}
	lastAttempt := windowEnd
	if r.Dev {
		lastAttempt = firstAttempt
	}
	for attempt := firstAttempt; attempt <= lastAttempt; attempt++ {
		dir := filepath.Join(sourceRunDir, fmt.Sprintf("attempt-%03d", attempt))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		req := makeTranslationRequest(pageID, locale, protected, glossary.PromptRules(pageID), previous)
		if err := writeTranslationJSON(filepath.Join(dir, "request.json"), savedTranslationRequest{pageID, locale, page.SourceSHA256, req}); err != nil {
			return nil, err
		}
		call, callErr := client.Call(ctx, apiKey, req)
		if err := writeTranslationJSON(filepath.Join(dir, "response.json"), call); err != nil {
			return nil, err
		}
		candidate := ""
		last = TranslationValidation{Attempt: attempt, Failures: []string{}}
		if callErr != nil {
			last.Failures = append(last.Failures, callErr.Error())
		} else {
			last.APISuccess = true
			if call.FinishReason != "stop" {
				last.Failures = append(last.Failures, "finish_reason is "+call.FinishReason+", want stop")
			}
			if strings.Contains(call.Content, "```") || !strings.HasPrefix(strings.TrimSpace(call.Content), "* ") {
				last.Failures = append(last.Failures, "model output is fenced, explained, or not a complete section")
			}
			var tokenFailures []string
			candidate, tokenFailures = protected.restore(call.Content)
			last.Failures = append(last.Failures, tokenFailures...)
			last.TokenValid = len(tokenFailures) == 0
			if last.TokenValid {
				if err := ValidateCandidate(r.Root, r.Catalog, pageID, []byte(candidate)); err != nil {
					last.Failures = append(last.Failures, err.Error())
				} else {
					last.PresentValid = true
				}
			}
		}
		last.Passed = last.APISuccess && last.TokenValid && last.PresentValid && len(last.Failures) == 0
		if err := writeTranslationJSON(filepath.Join(dir, "validation.json"), last); err != nil {
			return nil, err
		}
		if last.Passed {
			candidatePath := filepath.ToSlash(filepath.Join("locales", locale, "candidates", strings.ReplaceAll(pageID, "/", "-")+".article"))
			if err := os.MkdirAll(filepath.Join(r.Root, filepath.Dir(candidatePath)), 0755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(r.Root, filepath.FromSlash(candidatePath)), []byte(candidate), 0644); err != nil {
				return nil, err
			}
			updated := now().UTC().Format(time.RFC3339)
			if err := updateTranslationStatus(r.Root, locale, pageID, "ready", attempt, page.SourceSHA256, candidatePath, updated, "GLM-5.2 candidate passed existing validator"); err != nil {
				return nil, err
			}
			return &TranslationRunResult{pageID, locale, page.SourceSHA256, "glm-5.2", attempt, "ready", candidatePath, &last, updated}, nil
		}
		previous = retryFeedback(last.Failures)
	}
	updated := now().UTC().Format(time.RFC3339)
	if r.Dev {
		attempt := firstAttempt
		if err := updateTranslationStatus(r.Root, locale, pageID, "pending", attempt, page.SourceSHA256, "", updated, previous); err != nil {
			return nil, err
		}
		return &TranslationRunResult{pageID, locale, page.SourceSHA256, "glm-5.2", attempt, "pending", "", &last, updated}, nil
	}
	if err := updateTranslationStatus(r.Root, locale, pageID, "blocked", lastAttempt, page.SourceSHA256, "", updated, previous); err != nil {
		return nil, err
	}
	return &TranslationRunResult{pageID, locale, page.SourceSHA256, "glm-5.2", lastAttempt, "blocked", "", &last, updated}, nil
}

// retryFeedback deliberately summarizes audited validation failures for the
// model. Detailed source/candidate diagnostics stay in validation.json and are
// never echoed into a subsequent model request.
func retryFeedback(failures []string) string {
	joined := strings.ToLower(strings.Join(failures, "\n"))
	switch {
	case strings.Contains(joined, "protected token") || strings.Contains(joined, "token "):
		return "上一次输出未能完整、唯一地保留所有受保护 token。每个现有 token 必须原样且恰好出现一次；不得自行重建 token 所代表的代码、directive、链接目标或其他原始结构。请重新翻译完整页面，并保持其他受保护结构不变。"
	case strings.Contains(joined, "inline code"):
		return "上一次输出自行新增、删除或改变了行内代码结构。不得在普通文本中自行添加反引号代码；所有已受保护的行内代码只能通过现有 token 表示。请重新翻译完整页面，并保持其他受保护结构不变。"
	case strings.Contains(joined, "directive"):
		return "上一次输出出现了未受保护的额外 present directive，或改变了 directive 结构。不得自行书写 .play、.image 等 present directive；directive 只能通过已有保护 token 表示。请重新翻译完整页面，并保持其他受保护结构不变。"
	case strings.Contains(joined, "font span") || strings.Contains(joined, "emphasis"):
		return "上一次输出破坏了强调或字体结构。必须保留所有已有强调结构标记；不得自行新增、删除或改变强调类型。请重新翻译完整页面，并保持其他受保护结构不变。"
	case strings.Contains(joined, "present parse"):
		return "上一次输出不是可由 present 解析的完整页面。请只输出完整的 present.Section，并保持既有段落与受保护结构。请重新翻译完整页面，并保持其他受保护结构不变。"
	default:
		return "上一次输出未通过页面结构校验。请只重新翻译普通文本，完整保留所有已有保护 token 和 present 结构，不要自行增删或改写结构。请重新翻译完整页面，并保持其他受保护结构不变。"
	}
}

// RecoverNetworkBlockedTranslation explicitly reopens one formal three-attempt
// window only when the exhausted window is fully audited as response-less network
// failures. It never deletes or renumbers attempt records.
func RecoverNetworkBlockedTranslation(root string, catalog *Catalog, pageID, locale string, now func() time.Time) (*TranslationRecoveryResult, error) {
	if locale != "zh-CN" {
		return nil, fmt.Errorf("unsupported locale %q", locale)
	}
	if catalog == nil {
		return nil, errors.New("translation catalog is required")
	}
	page, err := catalog.Page(pageID)
	if err != nil {
		return nil, err
	}
	if sum(page.Source) != page.SourceSHA256 {
		return nil, fmt.Errorf("%s: hydrated source hash mismatch", pageID)
	}
	status, _, err := LoadTranslationResult(root, pageID, locale)
	if err != nil {
		return nil, err
	}
	if status.State != "blocked" {
		return nil, fmt.Errorf("%s is %s, want blocked for network recovery", pageID, status.State)
	}
	if status.SourceSHA256 != page.SourceSHA256 {
		return nil, fmt.Errorf("%s: blocked status source hash does not match current source", pageID)
	}
	const formalWindow = 3
	if status.Attempts < formalWindow {
		return nil, fmt.Errorf("%s: blocked status has only %d attempts, cannot prove an exhausted formal window", pageID, status.Attempts)
	}
	sourceRunDir := filepath.Join(root, "data", "translation-runs", locale, pageID, "sources", page.SourceSHA256)
	recovered := make([]int, 0, formalWindow)
	for attempt := status.Attempts - formalWindow + 1; attempt <= status.Attempts; attempt++ {
		if err := verifyResponseLessNetworkFailure(filepath.Join(sourceRunDir, fmt.Sprintf("attempt-%03d", attempt)), attempt); err != nil {
			return nil, fmt.Errorf("%s: cannot recover formal network window: %w", pageID, err)
		}
		recovered = append(recovered, attempt)
	}
	if now == nil {
		now = time.Now
	}
	updated := now().UTC().Format(time.RFC3339)
	recoveryPath, err := nextNetworkRecoveryPath(sourceRunDir)
	if err != nil {
		return nil, err
	}
	record := networkFailureRecoveryRecord{pageID, locale, page.SourceSHA256, recovered, status.State, status.Attempts, updated, "response-less-network-failure"}
	if err := writeTranslationJSON(recoveryPath, record); err != nil {
		return nil, err
	}
	note := fmt.Sprintf("formal network recovery recorded for response-less attempts %03d-%03d; next formal window starts at attempt-%03d", recovered[0], recovered[len(recovered)-1], status.Attempts+1)
	if err := updateTranslationStatus(root, locale, pageID, "pending", status.Attempts, page.SourceSHA256, "", updated, note); err != nil {
		return nil, err
	}
	return &TranslationRecoveryResult{pageID, locale, page.SourceSHA256, status.Attempts, "pending", filepath.ToSlash(recoveryPath), updated}, nil
}

// RevalidateSavedTranslationResponse replays no model work: it applies the
// current protection/restore and candidate validator to one immutable, audited
// successful response from a blocked page.
func RevalidateSavedTranslationResponse(root string, catalog *Catalog, pageID, locale string, attempt int, now func() time.Time) (*TranslationRevalidationResult, error) {
	if attempt <= 0 {
		return nil, fmt.Errorf("attempt must be a positive integer")
	}
	if catalog == nil {
		return nil, errors.New("translation catalog is required")
	}
	page, err := catalog.Page(pageID)
	if err != nil {
		return nil, err
	}
	if sum(page.Source) != page.SourceSHA256 {
		return nil, fmt.Errorf("%s: hydrated source hash mismatch", pageID)
	}
	status, _, err := LoadTranslationResult(root, pageID, locale)
	if err != nil {
		return nil, err
	}
	if status.State != "blocked" {
		return nil, fmt.Errorf("%s is %s, want blocked for response revalidation", pageID, status.State)
	}
	if status.SourceSHA256 != page.SourceSHA256 {
		return nil, fmt.Errorf("%s: blocked status source hash does not match current source", pageID)
	}
	glossary, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, err
	}
	sourceRunDir := filepath.Join(root, "data", "translation-runs", locale, pageID, "sources", page.SourceSHA256)
	attemptDir := filepath.Join(sourceRunDir, fmt.Sprintf("attempt-%03d", attempt))
	requestPath, responsePath := filepath.Join(attemptDir, "request.json"), filepath.Join(attemptDir, "response.json")
	validationPath := filepath.Join(attemptDir, "validation.json")
	requestBytes, err := os.ReadFile(requestPath)
	if err != nil {
		return nil, fmt.Errorf("attempt-%03d request audit: %w", attempt, err)
	}
	responseBytes, err := os.ReadFile(responsePath)
	if err != nil {
		return nil, fmt.Errorf("attempt-%03d response audit: %w", attempt, err)
	}
	validationBytes, err := os.ReadFile(validationPath)
	if err != nil {
		return nil, fmt.Errorf("attempt-%03d validation audit: %w", attempt, err)
	}
	var request savedTranslationRequest
	var response TranslationCallResult
	var historical TranslationValidation
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		return nil, fmt.Errorf("attempt-%03d invalid request audit: %w", attempt, err)
	}
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("attempt-%03d invalid response audit: %w", attempt, err)
	}
	if err := json.Unmarshal(validationBytes, &historical); err != nil {
		return nil, fmt.Errorf("attempt-%03d invalid validation audit: %w", attempt, err)
	}
	if request.PageID != pageID || request.Locale != locale || request.SourceSHA256 != page.SourceSHA256 {
		return nil, fmt.Errorf("attempt-%03d audit identity does not match page, locale, and source hash", attempt)
	}
	if historical.Attempt != attempt || !historical.APISuccess {
		return nil, fmt.Errorf("attempt-%03d is not an audited successful API response", attempt)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || response.APIError != "" || response.FinishReason != "stop" || strings.TrimSpace(response.Content) == "" {
		return nil, fmt.Errorf("attempt-%03d has no successful stop response with model content", attempt)
	}
	if now == nil {
		now = time.Now
	}
	updated := now().UTC().Format(time.RFC3339)
	validation := TranslationValidation{Attempt: attempt, APISuccess: true}
	if strings.Contains(response.Content, "```") || !strings.HasPrefix(strings.TrimSpace(response.Content), "* ") {
		validation.Failures = append(validation.Failures, "model output is fenced, explained, or not a complete section")
	}
	protected := protectTranslation(page.Source, page.SourceSHA256, glossary)
	candidate, failures := protected.restore(response.Content)
	validation.Failures = append(validation.Failures, failures...)
	validation.TokenValid = len(failures) == 0
	if validation.TokenValid {
		if err := ValidateCandidateForLocale(root, catalog, pageID, locale, []byte(candidate)); err != nil {
			validation.Failures = append(validation.Failures, err.Error())
		} else {
			validation.PresentValid = true
		}
	}
	validation.Passed = validation.APISuccess && validation.TokenValid && validation.PresentValid && len(validation.Failures) == 0
	responseAuditPath, err := repositoryRelativePath(root, responsePath)
	if err != nil {
		return nil, err
	}
	validationAuditPath, err := repositoryRelativePath(root, validationPath)
	if err != nil {
		return nil, err
	}
	record := responseRevalidationRecord{1, locale, pageID, page.SourceSHA256, attempt, responseAuditPath, validationAuditPath, updated, validation, "", validation.Passed, validation.Failures}
	if validation.Passed {
		record.CandidateSHA256 = sum([]byte(candidate))
	}
	auditPath, err := nextResponseRevalidationPath(sourceRunDir)
	if err != nil {
		return nil, err
	}
	if err := writeTranslationJSON(auditPath, record); err != nil {
		return nil, err
	}
	result := &TranslationRevalidationResult{pageID, locale, page.SourceSHA256, attempt, status.Attempts, "blocked", "", filepath.ToSlash(auditPath), &validation, updated}
	if !validation.Passed {
		return result, fmt.Errorf("%s attempt-%03d revalidation failed: %s", pageID, attempt, strings.Join(validation.Failures, "; "))
	}
	candidatePath := filepath.ToSlash(filepath.Join("locales", locale, "candidates", strings.ReplaceAll(pageID, "/", "-")+".article"))
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(candidatePath)), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(candidatePath)), []byte(candidate), 0644); err != nil {
		return nil, err
	}
	if err := updateTranslationStatus(root, locale, pageID, "ready", status.Attempts, page.SourceSHA256, candidatePath, updated, fmt.Sprintf("historical GLM-5.2 response attempt-%03d passed current restore and validator", attempt)); err != nil {
		return nil, err
	}
	result.Status, result.CandidatePath = "ready", candidatePath
	return result, nil
}

func repositoryRelativePath(root, path string) (string, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("make audit path relative to repository: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("audit path %q is outside repository root", path)
	}
	return filepath.ToSlash(relative), nil
}

func nextResponseRevalidationPath(sourceRunDir string) (string, error) {
	entries, err := os.ReadDir(sourceRunDir)
	if err != nil {
		return "", err
	}
	max := 0
	for _, entry := range entries {
		var n int
		if entry.Type().IsRegular() {
			if _, err := fmt.Sscanf(entry.Name(), "revalidation-%03d.json", &n); err == nil && n > max {
				max = n
			}
		}
	}
	return filepath.Join(sourceRunDir, fmt.Sprintf("revalidation-%03d.json", max+1)), nil
}

func verifyResponseLessNetworkFailure(dir string, attempt int) error {
	for _, name := range []string{"request.json", "response.json", "validation.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("attempt-%03d audit is incomplete: %s: %w", attempt, name, err)
		}
	}
	validationBytes, err := os.ReadFile(filepath.Join(dir, "validation.json"))
	if err != nil {
		return err
	}
	var validation TranslationValidation
	if err := json.Unmarshal(validationBytes, &validation); err != nil {
		return fmt.Errorf("attempt-%03d invalid validation audit: %w", attempt, err)
	}
	if validation.Attempt != attempt || validation.APISuccess || validation.TokenValid || validation.PresentValid || validation.Passed || len(validation.Failures) == 0 {
		return fmt.Errorf("attempt-%03d is not a response-less network failure", attempt)
	}
	for _, failure := range validation.Failures {
		if !strings.HasPrefix(failure, "network: ") {
			return fmt.Errorf("attempt-%03d has non-network failure %q", attempt, failure)
		}
	}
	responseBytes, err := os.ReadFile(filepath.Join(dir, "response.json"))
	if err != nil {
		return err
	}
	var response TranslationCallResult
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return fmt.Errorf("attempt-%03d invalid response audit: %w", attempt, err)
	}
	if response.StatusCode != 0 || response.RequestID != "" || response.FinishReason != "" || response.Content != "" || hasResponseRaw(response.Raw) || response.APIError != "" || response.Usage != (TranslationUsage{}) {
		return fmt.Errorf("attempt-%03d has an API response and is not recoverable as network-only", attempt)
	}
	return nil
}

func hasResponseRaw(raw json.RawMessage) bool {
	return len(raw) != 0 && string(raw) != "null"
}

func nextNetworkRecoveryPath(sourceRunDir string) (string, error) {
	entries, err := os.ReadDir(sourceRunDir)
	if err != nil {
		return "", err
	}
	max := 0
	for _, entry := range entries {
		var recovery int
		_, scanErr := fmt.Sscanf(entry.Name(), "network-recovery-%03d.json", &recovery)
		if entry.Type().IsRegular() && scanErr == nil && recovery > max {
			max = recovery
		}
	}
	return filepath.Join(sourceRunDir, fmt.Sprintf("network-recovery-%03d.json", max+1)), nil
}

// currentFormalAttemptWindow derives the only formal window that may run for a
// source. A recovery audit advances the window; invoking translate run again
// never does.
func currentFormalAttemptWindow(sourceRunDir, pageID, locale, sourceSHA256 string, width int) (int, int, error) {
	if width <= 0 {
		return 0, 0, fmt.Errorf("formal attempt window width must be positive")
	}
	entries, err := os.ReadDir(sourceRunDir)
	if os.IsNotExist(err) {
		return 1, width, nil
	}
	if err != nil {
		return 0, 0, err
	}
	var latest networkFailureRecoveryRecord
	latestNumber := 0
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		var number int
		if _, err := fmt.Sscanf(entry.Name(), "network-recovery-%03d.json", &number); err != nil || number == 0 {
			continue
		}
		bytes, err := os.ReadFile(filepath.Join(sourceRunDir, entry.Name()))
		if err != nil {
			return 0, 0, err
		}
		var record networkFailureRecoveryRecord
		if err := json.Unmarshal(bytes, &record); err != nil {
			return 0, 0, fmt.Errorf("invalid network recovery audit %s: %w", entry.Name(), err)
		}
		if record.PageID != pageID || record.Locale != locale || record.SourceSHA256 != sourceSHA256 || record.RecoveryKind != "response-less-network-failure" || record.PreviousStatus != "blocked" || len(record.RecoveredAttempts) != width || record.PreviousAttempts < width {
			return 0, 0, fmt.Errorf("invalid network recovery audit %s", entry.Name())
		}
		for index, attempt := range record.RecoveredAttempts {
			if attempt != record.PreviousAttempts-width+1+index {
				return 0, 0, fmt.Errorf("invalid recovered attempt range in %s", entry.Name())
			}
		}
		if number > latestNumber {
			latestNumber = number
			latest = record
		}
	}
	if latestNumber == 0 {
		return 1, width, nil
	}
	start := latest.PreviousAttempts + 1
	return start, start + width - 1, nil
}

func makeTranslationRequest(pageID, locale string, protected protectedTranslation, glossaryRules, previous string) TranslationAPIRequest {
	system := `请将一个完整的《Go 语言之旅》present.Section 从英文翻译为中国大陆简体中文。

只返回完整且可由 present 解析的 .article 内容。必须保留每个保护 token，使其原样出现、恰好出现一次；不得修改、删除、复制或伪造。为适应目标语言自然语序可以调整 token 位置，但不得破坏其所属的链接、代码、directive、预格式化等结构关系。必须使用术语表中的强制译法；对应的、应当翻译的英文显示文本不得残留；不得简化、遗漏或改变原文含义。

中文表达要求：
1. 翻译前先理解完整 present.Section 的页面用途和上下文，不要逐词翻译或机械照搬英文语序。
2. 页面标题应简洁、自然并准确概括页面主题。简短、含双关或依赖上下文的标题，应根据完整页面含义翻译，不能生硬拼接字面译法。
3. 正文采用自然、清楚、简洁的中国大陆简体中文技术教程风格。在不遗漏、不增加、不改变技术含义的前提下，可以调整语序，并可在同一段落内合理拆分或合并句子。
4. 避免明显的机器翻译表达，例如机械使用“它们”“你自己”“来进行……”，以及不自然的被动语态或英语式定语顺序。
5. 准确区分用户操作：按钮应点击；链接应点击；键盘按键应按或按下；命令应执行；文本内容应输入。不得把按键描述为输入字符串，也不得混淆按钮、命令和链接。
6. 技术词语应根据当前语境选择准确、自然的中文译法，不要仅按固定字典逐词替换。
7. 可以润色普通中文文本，但不得改变、增删或重新标记任何受保护结构。尤其不得自行新增或删除行内代码反引号、预格式化代码、present directive、链接及链接 target、HTML 或特殊 present 语法；所有保护 token 必须完整且唯一，结构关系和形式必须保持一致。
8. 不得增加原文没有的解释、提示、结论或标题。
9. 输出前静默自检：标题是否自然；是否存在英文语序或机器翻译腔；操作说明是否符合真实操作；技术含义和信息量是否与原文一致；是否无意增删了行内代码、链接、directive 或其他结构。

只输出最终完整的 present.Section，不输出分析、说明或修改过程。`
	user := fmt.Sprintf(`page_id: %s
source_locale: en
target_locale: %s

强制术语表与译法规则：
%s

重要：下文每个形如 ⟪GTI18N_...⟫ 的保护 token 都是唯一占位符。
本页共有 %d 个保护 token，输出中也必须恰好包含 %d 个。
每个 token 必须原样输出且恰好输出一次。
不得复制、不得复用、不得删除、不得改写或伪造任何 token。
可以仅在完整保持所属语义单元和结构关系的前提下调整位置；不得将成对结构 token 拆散或独立移动，也不得将任何内容移入或移出 pair。不同的完整 pair 可为自然语序随各自的语义单元整体换位。

%s

需要翻译的完整受保护页面：
%s`, pageID, locale, glossaryRules, len(protected.Tokens), len(protected.Tokens), protectedStructureProtocol(protected), protected.Text)
	if previous != "" {
		user += "\n\n上一次完整页面翻译未通过校验：" + previous
	}
	return TranslationAPIRequest{Model: "glm-5.2", Stream: false, Thinking: map[string]string{"type": "disabled"}, DoSample: false, MaxTokens: 8192, Messages: []TranslationMessage{{Role: "system", Content: system}, {Role: "user", Content: user}}}
}

func protectedStructureProtocol(protected protectedTranslation) string {
	var rules []string
	if len(protected.InlinePairs) != 0 {
		rules = append(rules, "行内代码成对结构（反引号由程序恢复，pair 内的英文或标识符不是应翻译的英文显示文本）：")
		for i, pair := range protected.InlinePairs {
			rules = append(rules, fmt.Sprintf("- 行内代码 pair %d：%s 是 opening token，%s 是 closing token。两者之间当前可见的代码内容必须逐字原样保留在同一 pair 内；不得翻译、改写、增删、移出 pair 或移入其他内容；不得自行添加反引号。", i+1, pair.Open, pair.Close))
		}
	}
	emphasisPairs := emphasisPairsForPrompt(protected)
	if len(emphasisPairs) != 0 {
		rules = append(rules, "强调成对结构：")
		for i, pair := range emphasisPairs {
			kind := "italic"
			if pair.Kind == protectedBoldOpen {
				kind = "bold"
			}
			rules = append(rules, fmt.Sprintf("- %s pair %d：%s 是 opening token，%s 是 closing token。两者之间的自然语言允许翻译，但译文必须始终留在同一 pair 内；不得拆散、交换 token，也不得将内容移出或移入 pair。", kind, i+1, pair.Open, pair.Close))
		}
	}
	if len(rules) == 0 {
		return "本页没有成对结构 token；所有单 token 结构仍必须原样、唯一保留。"
	}
	rules = append(rules, "除上述 pair 外，directive、链接 target、keep-word 等单 token 结构仍只能原样、唯一保留；不得自行重建其隐藏的原始结构。")
	return strings.Join(rules, "\n")
}

type promptEmphasisPair struct {
	Open, Close string
	Kind        protectedTokenKind
}

func emphasisPairsForPrompt(protected protectedTranslation) []promptEmphasisPair {
	var pairs []promptEmphasisPair
	for i, token := range protected.Tokens {
		switch protected.Kinds[i] {
		case protectedItalicOpen, protectedBoldOpen:
			pairs = append(pairs, promptEmphasisPair{Open: token, Kind: protected.Kinds[i]})
		case protectedItalicClose, protectedBoldClose:
			for j := len(pairs) - 1; j >= 0; j-- {
				if pairs[j].Close == "" {
					pairs[j].Close = token
					break
				}
			}
		}
	}
	return pairs
}

func nextTranslationAttempt(sourceRunDir string) (int, error) {
	entries, err := os.ReadDir(sourceRunDir)
	if os.IsNotExist(err) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	max := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		var attempt int
		if _, err := fmt.Sscanf(entry.Name(), "attempt-%03d", &attempt); err == nil && attempt > max {
			max = attempt
		}
	}
	return max + 1, nil
}

func LoadTranslationResult(root, pageID, locale string) (*Status, string, error) {
	statuses, err := ReadStatuses(filepath.Join(root, "locales", locale, "status.tsv"))
	if err != nil {
		return nil, "", err
	}
	for i := range statuses {
		if statuses[i].PageID != pageID {
			continue
		}
		candidate := ""
		if statuses[i].CandidatePath != "" {
			b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(statuses[i].CandidatePath)))
			if err != nil {
				return nil, "", err
			}
			candidate = string(b)
		}
		return &statuses[i], candidate, nil
	}
	return nil, "", fmt.Errorf("unknown page_id %q", pageID)
}

func updateTranslationStatus(root, locale, pageID, state string, attempts int, hash, candidate, updated, note string) error {
	path := filepath.Join(root, "locales", locale, "status.tsv")
	statuses, err := ReadStatuses(path)
	if err != nil {
		return err
	}
	found := false
	for i := range statuses {
		if statuses[i].PageID == pageID {
			statuses[i] = Status{pageID, state, attempts, hash, candidate, updated, note}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unknown page_id %q", pageID)
	}
	return writeStatuses(path, statuses)
}

func writeTranslationJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}
