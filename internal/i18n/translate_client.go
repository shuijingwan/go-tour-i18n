package i18n

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultTranslationEndpoint = "https://open.bigmodel.cn/api/paas/v4/chat/completions"

type TranslationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TranslationAPIRequest struct {
	Model     string               `json:"model"`
	Stream    bool                 `json:"stream"`
	Thinking  map[string]string    `json:"thinking"`
	DoSample  bool                 `json:"do_sample"`
	MaxTokens int                  `json:"max_tokens"`
	Messages  []TranslationMessage `json:"messages"`
}

type TranslationUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type TranslationCallResult struct {
	StatusCode   int              `json:"status_code"`
	RequestID    string           `json:"request_id"`
	FinishReason string           `json:"finish_reason"`
	Usage        TranslationUsage `json:"usage"`
	Content      string           `json:"content"`
	Raw          json.RawMessage  `json:"raw"`
	APIError     string           `json:"api_error,omitempty"`
}

type TranslationClient struct {
	Endpoint string
	HTTP     *http.Client
}

func NewTranslationClient() *TranslationClient {
	return &TranslationClient{Endpoint: defaultTranslationEndpoint, HTTP: &http.Client{Timeout: 90 * time.Second}}
}

func (c *TranslationClient) Call(ctx context.Context, key string, req TranslationAPIRequest) (TranslationCallResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TranslationCallResult{}, err
	}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = defaultTranslationEndpoint
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return TranslationCallResult{}, err
	}
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Authorization", "Bearer "+key)
	resp, err := httpClient.Do(hr)
	if err != nil {
		return TranslationCallResult{}, fmt.Errorf("network: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return TranslationCallResult{}, err
	}
	result := TranslationCallResult{StatusCode: resp.StatusCode, RequestID: resp.Header.Get("X-Request-Id"), Raw: append(json.RawMessage(nil), raw...)}
	var parsed struct {
		ID      string `json:"id"`
		Choices []struct {
			Message      TranslationMessage `json:"message"`
			FinishReason string             `json:"finish_reason"`
		} `json:"choices"`
		Usage TranslationUsage `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return result, fmt.Errorf("decode API response: %w", err)
	}
	if result.RequestID == "" {
		result.RequestID = parsed.ID
	}
	result.Usage = parsed.Usage
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.APIError = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return result, fmt.Errorf("api: HTTP %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return result, fmt.Errorf("api: response has no choices")
	}
	result.FinishReason = parsed.Choices[0].FinishReason
	result.Content = parsed.Choices[0].Message.Content
	return result, nil
}
