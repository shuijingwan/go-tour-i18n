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
	system := `Translate one complete A Tour of Go present.Section from English to Simplified Chinese. Return only complete parseable .article content. Preserve every protection token exactly once and in order. Mandatory glossary translations must be used, corresponding English display text must not remain, and the meaning must not be simplified or changed.

中文表达要求：
1. 翻译前先理解完整 present.Section 的页面用途和上下文，不要逐词翻译或机械照搬英文语序。
2. 页面标题应简洁、自然并准确概括页面主题。简短、含双关或依赖上下文的标题，应根据完整页面含义翻译，不能生硬拼接字面译法。
3. 正文采用自然、清楚、简洁的中国大陆简体中文技术教程风格。在不遗漏、不增加、不改变技术含义的前提下，可以调整语序，并可在同一段落内合理拆分或合并句子。
4. 避免明显的机器翻译表达，例如机械使用“它们”“你自己”“来进行……”，以及不自然的被动语态或英语式定语顺序。
5. 准确区分用户操作：按钮应点击；链接应点击；键盘按键应按或按下；命令应执行；文本内容应输入。不得把按键描述为输入字符串，也不得混淆按钮、命令和链接。
6. 技术词语应根据当前语境选择准确、自然的中文译法，不要仅按固定字典逐词替换。
7. 可以润色普通中文文本，但不得改变、增删或重新标记任何受保护结构。尤其不得自行新增或删除行内代码反引号、预格式化代码、present directive、链接及链接 target、HTML 或特殊 present 语法；受保护内容的数量、顺序和形式必须保持一致。
8. 不得增加原文没有的解释、提示、结论或标题。
9. 输出前静默自检：标题是否自然；是否存在英文语序或机器翻译腔；操作说明是否符合真实操作；技术含义和信息量是否与原文一致；是否无意增删了行内代码、链接、directive 或其他结构。

只输出最终完整的 present.Section，不输出分析、说明或修改过程。`
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
