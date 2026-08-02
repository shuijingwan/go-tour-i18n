package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func mockHTTP(handler roundTripFunc) *http.Client {
	return &http.Client{Transport: handler}
}

func writeTestGlossary(t *testing.T, root string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "locales", "zh-CN", "glossary.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "locales", "zh-CN", "glossary.yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestTranslationProtectionRoundTripAndFailures(t *testing.T) {
	source := []byte("* Hello\n\nUse `Go` and [[/][this link]].\n\n.play welcome/hello.go\n")
	p := protectTranslation(source, sum(source), nil)
	got, failures := p.restore(p.Text)
	if len(failures) != 0 || got != string(source) {
		t.Fatalf("round trip got=%q failures=%v", got, failures)
	}
	invalid := map[string]string{
		"missing":   strings.Replace(p.Text, p.Tokens[0], "", 1),
		"duplicate": p.Text + p.Tokens[0],
		"unknown":   p.Text + "⟪GTI18N_deadbeef_999999⟫",
		"order":     strings.Replace(strings.Replace(p.Text, p.Tokens[0], "TEMP", 1), p.Tokens[1], p.Tokens[0], 1) + p.Tokens[1],
	}
	for name, output := range invalid {
		t.Run(name, func(t *testing.T) {
			if _, failures := p.restore(output); len(failures) == 0 {
				t.Fatal("invalid tokens accepted")
			}
		})
	}
}

func TestMandatoryGlossaryLinkLabelsAreLocked(t *testing.T) {
	glossary := &Glossary{Mandatory: map[string]string{
		"A Tour of Go": "Go 语言之旅",
		"previous":     "上一页",
		"next":         "下一页",
		"Run":          "运行",
		"Format":       "格式化",
		"slides":       "页面",
	}}
	source := []byte("* Links\n\n[[javascript:logo][A Tour of Go]] [[javascript:prev][\"previous\"]] [[javascript:next][\"next\"]] [[javascript:run][Run]] [[javascript:format][Format]]\n\nOrdinary slides remain model-translatable.\n")
	p := protectTranslation(source, sum(source), glossary)
	if strings.Contains(p.Text, "A Tour of Go") || strings.Contains(p.Text, `"previous"`) || strings.Contains(p.Text, "[Run]") {
		t.Fatalf("mandatory labels were not locked:\n%s", p.Text)
	}
	if !strings.Contains(p.Text, "Ordinary slides remain model-translatable.") {
		t.Fatalf("ordinary slides text was locked:\n%s", p.Text)
	}
	restored, failures := p.restore(p.Text)
	if len(failures) != 0 {
		t.Fatal(failures)
	}
	for _, want := range []string{"[Go 语言之旅]", `["上一页"]`, `["下一页"]`, "[运行]", "[格式化]"} {
		if !strings.Contains(restored, want) {
			t.Errorf("restored output missing %q:\n%s", want, restored)
		}
	}
	if !strings.Contains(restored, "Ordinary slides remain model-translatable.") {
		t.Fatal("ordinary slides text changed during restoration")
	}
}

func TestTranslationClientUsesRoundTripperAndCapturesMetadata(t *testing.T) {
	var sawAuth bool
	client := &TranslationClient{Endpoint: "https://example.invalid/completions", HTTP: mockHTTP(func(r *http.Request) (*http.Response, error) {
		sawAuth = r.Header.Get("Authorization") == "Bearer test-secret"
		body := `{"id":"request-123","choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	result, err := client.Call(context.Background(), "test-secret", TranslationAPIRequest{Model: "glm-5.2"})
	if err != nil {
		t.Fatal(err)
	}
	if !sawAuth || result.RequestID != "request-123" || result.FinishReason != "stop" || result.Usage.TotalTokens != 15 {
		t.Fatalf("result=%+v sawAuth=%t", result, sawAuth)
	}
}

func TestTranslationRunnerRetriesAndUsesPersistentPageID(t *testing.T) {
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+
		"welcome/1\tpending\t0\t"+strings.Repeat("a", 64)+"\t\t\t\n")
	writeTestGlossary(t, root)
	source := []byte("* Hello\n\nEnglish text.\n")
	catalog := &Catalog{Pages: []Page{{ID: "welcome/1", Article: "welcome.article", Source: source, SourceSHA256: sum(source)}}}
	// Keep the fixture status consistent with the hydrated source used by the runner.
	if err := updateTranslationStatus(root, "zh-CN", "welcome/1", "pending", 0, sum(source), "", "", ""); err != nil {
		t.Fatal(err)
	}
	legacyAttempt := filepath.Join(root, "data", "translation-runs", "zh-CN", "welcome", "1", "attempt-001")
	if err := os.MkdirAll(legacyAttempt, 0755); err != nil {
		t.Fatal(err)
	}
	legacyMarker := filepath.Join(legacyAttempt, "audit-marker")
	if err := os.WriteFile(legacyMarker, []byte("old source attempt"), 0644); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	httpClient := mockHTTP(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		var request TranslationAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		body := `{"choices":[{"message":{"role":"assistant","content":""},"finish_reason":"length"}],"usage":{}}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	runner := TranslationRunner{Root: root, Catalog: catalog, Client: &TranslationClient{Endpoint: "https://example.invalid", HTTP: httpClient}, Now: func() time.Time { return time.Unix(0, 0) }}
	result, err := runner.Run(context.Background(), "welcome/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "blocked" || result.Attempts != 3 || calls.Load() != 3 {
		t.Fatalf("result=%+v calls=%d", result, calls.Load())
	}
	requestRecord, err := os.ReadFile(filepath.Join(root, "data", "translation-runs", "zh-CN", "welcome", "1", "sources", sum(source), "attempt-001", "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(requestRecord), "test-secret") || strings.Contains(string(requestRecord), "Authorization") {
		t.Fatal("API credential or Authorization header was persisted")
	}
	if marker, err := os.ReadFile(legacyMarker); err != nil || string(marker) != "old source attempt" {
		t.Fatalf("legacy attempt was overwritten: marker=%q err=%v", marker, err)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		path := filepath.Join(root, "data", "translation-runs", "zh-CN", "welcome", "1", "sources", sum(source), fmt.Sprintf("attempt-%03d", attempt), "validation.json")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("new source attempt %d missing: %v", attempt, err)
		}
	}
	status, _, err := LoadTranslationResult(root, "welcome/1", "zh-CN")
	if err != nil || status.State != "blocked" || status.Attempts != 3 {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	before := calls.Load()
	if _, err := runner.Run(context.Background(), "welcome/1", "zh-CN", "test-secret"); err == nil || !strings.Contains(err.Error(), "is blocked") {
		t.Fatalf("formal blocked retry error = %v", err)
	}
	if calls.Load() != before {
		t.Fatal("formal blocked retry called the API")
	}
}

func TestNextTranslationAttemptContinuesCurrentSource(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "attempt-001"), 0755); err != nil {
		t.Fatal(err)
	}
	next, err := nextTranslationAttempt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if next != 2 {
		t.Fatalf("next attempt = %d, want 2", next)
	}
}

func TestDevTranslationContinuesBlockedOneAttemptPerRun(t *testing.T) {
	source := []byte("* Hello\n\nEnglish text.\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+
		"welcome/1\tblocked\t3\t"+hash+"\t\t\tformal attempts exhausted\n")
	writeTestGlossary(t, root)
	sourceDir := filepath.Join(root, "data", "translation-runs", "zh-CN", "welcome", "1", "sources", hash)
	for attempt := 1; attempt <= 3; attempt++ {
		dir := filepath.Join(sourceDir, fmt.Sprintf("attempt-%03d", attempt))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "audit-marker"), []byte(fmt.Sprint(attempt)), 0644); err != nil {
			t.Fatal(err)
		}
	}
	var calls atomic.Int32
	httpClient := mockHTTP(func(r *http.Request) (*http.Response, error) {
		call := calls.Add(1)
		content := ""
		finish := "length"
		if call == 2 {
			var request TranslationAPIRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			user := request.Messages[len(request.Messages)-1].Content
			const marker = "Complete protected page:\n"
			content = user[strings.Index(user, marker)+len(marker):]
			finish = "stop"
		}
		body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": content}, "finish_reason": finish}}, "usage": map[string]int{}})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})
	runner := TranslationRunner{Root: root, Catalog: &Catalog{Pages: []Page{{ID: "welcome/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}, Client: &TranslationClient{Endpoint: "https://example.invalid", HTTP: httpClient}, Dev: true, Now: func() time.Time { return time.Unix(0, 0) }}

	failed, err := runner.Run(context.Background(), "welcome/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != "pending" || failed.Attempts != 4 || calls.Load() != 1 {
		t.Fatalf("failed dev result=%+v calls=%d", failed, calls.Load())
	}
	status, _, err := LoadTranslationResult(root, "welcome/1", "zh-CN")
	if err != nil || status.State != "pending" || status.Attempts != 4 {
		t.Fatalf("failed dev status=%+v err=%v", status, err)
	}

	ready, err := runner.Run(context.Background(), "welcome/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if ready.Status != "ready" || ready.Attempts != 5 || calls.Load() != 2 {
		t.Fatalf("ready dev result=%+v calls=%d", ready, calls.Load())
	}
	for attempt := 1; attempt <= 3; attempt++ {
		marker, err := os.ReadFile(filepath.Join(sourceDir, fmt.Sprintf("attempt-%03d", attempt), "audit-marker"))
		if err != nil || string(marker) != fmt.Sprint(attempt) {
			t.Fatalf("old attempt %d overwritten: marker=%q err=%v", attempt, marker, err)
		}
	}
	for attempt := 4; attempt <= 5; attempt++ {
		if _, err := os.Stat(filepath.Join(sourceDir, fmt.Sprintf("attempt-%03d", attempt), "validation.json")); err != nil {
			t.Fatalf("dev attempt %d missing: %v", attempt, err)
		}
	}
}

func TestTranslationPreflightDoesNotCallHTTP(t *testing.T) {
	var calls atomic.Int32
	client := &TranslationClient{HTTP: mockHTTP(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, nil
	})}
	runner := TranslationRunner{Root: t.TempDir(), Catalog: &Catalog{}, Client: client}
	if _, err := runner.Run(context.Background(), "welcome.hello", "zh-CN", "test-secret"); err == nil {
		t.Fatal("parallel page key was accepted")
	}
	if calls.Load() != 0 {
		t.Fatalf("HTTP calls=%d", calls.Load())
	}
}
