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
	if firstAttempt > maxAttempts {
		if !r.Dev {
			return nil, fmt.Errorf("%s source %s has exhausted %d attempts", pageID, page.SourceSHA256, maxAttempts)
		}
	}
	lastAttempt := maxAttempts
	if r.Dev {
		lastAttempt = firstAttempt
	}
	for attempt := firstAttempt; attempt <= lastAttempt; attempt++ {
		dir := filepath.Join(sourceRunDir, fmt.Sprintf("attempt-%03d", attempt))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		req := makeTranslationRequest(pageID, locale, protected.Text, glossary.PromptRules(pageID), previous)
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
		previous = strings.Join(last.Failures, "; ")
	}
	updated := now().UTC().Format(time.RFC3339)
	if r.Dev {
		attempt := firstAttempt
		if err := updateTranslationStatus(r.Root, locale, pageID, "pending", attempt, page.SourceSHA256, "", updated, previous); err != nil {
			return nil, err
		}
		return &TranslationRunResult{pageID, locale, page.SourceSHA256, "glm-5.2", attempt, "pending", "", &last, updated}, nil
	}
	if err := updateTranslationStatus(r.Root, locale, pageID, "blocked", maxAttempts, page.SourceSHA256, "", updated, previous); err != nil {
		return nil, err
	}
	return &TranslationRunResult{pageID, locale, page.SourceSHA256, "glm-5.2", maxAttempts, "blocked", "", &last, updated}, nil
}

func makeTranslationRequest(pageID, locale, page, glossaryRules, previous string) TranslationAPIRequest {
	system := "Translate one complete A Tour of Go present.Section from English to Simplified Chinese. Return only complete parseable .article content. Preserve every protection token exactly once and in order. Mandatory glossary translations must be used, corresponding English display text must not remain, and the meaning must not be simplified or changed."
	user := fmt.Sprintf("page_id: %s\nsource_locale: en\ntarget_locale: %s\n\nMandatory glossary rules:\n%s\n\nComplete protected page:\n%s", pageID, locale, glossaryRules, page)
	if previous != "" {
		user += "\n\nPrevious full-page attempt failed validation: " + previous + ". Translate the complete page again."
	}
	return TranslationAPIRequest{Model: "glm-5.2", Stream: false, Thinking: map[string]string{"type": "disabled"}, DoSample: false, MaxTokens: 8192, Messages: []TranslationMessage{{Role: "system", Content: system}, {Role: "user", Content: user}}}
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
