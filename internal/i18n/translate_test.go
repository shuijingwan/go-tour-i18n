package i18n

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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

func uniqueTranslationTokens(text string) []string {
	seen := map[string]bool{}
	var tokens []string
	for _, token := range translationTokenRE.FindAllString(text, -1) {
		if !seen[token] {
			seen[token] = true
			tokens = append(tokens, token)
		}
	}
	return tokens
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
	}
	for name, output := range invalid {
		t.Run(name, func(t *testing.T) {
			if got, failures := p.restore(output); len(failures) == 0 || got != "" {
				t.Fatal("invalid tokens accepted")
			}
		})
	}
	reordered := strings.Replace(strings.Replace(p.Text, p.Tokens[0], "TEMP", 1), p.Tokens[1], p.Tokens[0], 1) + p.Tokens[1]
	if got, failures := p.restore(reordered); len(failures) == 0 || got != "" {
		t.Fatalf("reordered inline sentinels accepted: got=%q failures=%v", got, failures)
	}
}

func TestInlineTokenBoundaryNormalization(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		model  func([]string) string
		want   string
	}{
		{"Chinese period", []string{"`comparable`", "`x`"}, func(tokens []string) string { return tokens[0] + "。" + tokens[1] }, "`comparable`。 `x`"},
		{"Chinese text", []string{"`s`", "`T`"}, func(tokens []string) string { return tokens[0] + "是" + tokens[1] }, "`s` 是 `T`"},
		{"Chinese conjunction", []string{"`==`", "`!=`"}, func(tokens []string) string { return tokens[0] + "和" + tokens[1] }, "`==` 和 `!=`"},
		{"Chinese enumeration", []string{"`f`", "`x`", "`y`"}, func(tokens []string) string { return tokens[0] + "、" + tokens[1] + "、" + tokens[2] }, "`f`、 `x`、 `y`"},
		{"single Chinese punctuation", []string{"`T`"}, func(tokens []string) string { return "调用 " + tokens[0] + "。" }, "调用 `T`。"},
		{"both Chinese text boundaries", []string{"`T`"}, func(tokens []string) string { return "类型" + tokens[0] + "的值" }, "类型 `T` 的值"},
		{"existing spaces", []string{"`s`", "`T`"}, func(tokens []string) string { return tokens[0] + " 是 " + tokens[1] }, "`s` 是 `T`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := protectTranslation([]byte(strings.Join(tt.values, " ")), "12345678", nil)
			if len(p.InlinePairs) != len(tt.values) {
				t.Fatalf("inline pairs = %d, want %d: %+v", len(p.InlinePairs), len(tt.values), p)
			}
			spans := make([]string, len(p.InlinePairs))
			for i, pair := range p.InlinePairs {
				spans[i] = pair.Open + pair.Content + pair.Close
			}
			model := tt.model(spans)
			normalized := normalizeInlineTokenBoundaries(model, p)
			if twice := normalizeInlineTokenBoundaries(normalized, p); twice != normalized {
				t.Fatalf("normalization is not idempotent:\nonce:  %q\ntwice: %q", normalized, twice)
			}
			got, failures := p.restore(model)
			if len(failures) != 0 || got != tt.want {
				t.Fatalf("restore = %q, %v; want %q", got, failures, tt.want)
			}
			codes := presentInlineCodes(got)
			if len(codes) != len(tt.values) {
				t.Fatalf("presentInlineCodes(%q) = %+v, want %d spans", got, codes, len(tt.values))
			}
			for i, code := range codes {
				if code.Raw != tt.values[i] {
					t.Errorf("span %d = %q, want %q", i+1, code.Raw, tt.values[i])
				}
			}
		})
	}
}

func TestInlineCodePairProtectionBasics6Regression(t *testing.T) {
	source := "* Multiple results\n\nThe `swap` function returns two strings.\n"
	p := protectTranslation([]byte(source), strings.Repeat("b", 64), nil)
	if len(p.InlinePairs) != 1 {
		t.Fatalf("inline pairs = %+v", p.InlinePairs)
	}
	pair := p.InlinePairs[0]
	if !strings.Contains(p.Text, pair.Open+"swap"+pair.Close) || strings.Contains(p.Text, "The "+pair.Open+pair.Close+" function") {
		t.Fatalf("swap was not retained between sentinels:\n%s", p.Text)
	}
	valid := "* 多个返回值\n\n调用" + pair.Open + "swap" + pair.Close + "函数会返回两个字符串。\n"
	candidate, failures := p.restore(valid)
	if len(failures) != 0 || !strings.Contains(candidate, "调用 `swap` 函数") {
		t.Fatalf("restore=%q failures=%v", candidate, failures)
	}
	root := repoRoot(t)
	catalog := &Catalog{Pages: []Page{{ID: "synthetic/basics6", Article: "basics.article", Source: []byte(source), SourceSHA256: sum([]byte(source))}}}
	if err := ValidateCandidate(root, catalog, "synthetic/basics6", []byte(candidate)); err != nil {
		t.Fatalf("restored candidate rejected: %v", err)
	}
	for name, model := range map[string]string{
		"changed":    strings.Replace(valid, "swap", "exchange", 1),
		"translated": strings.Replace(valid, "swap", "交换", 1),
		"missing":    strings.Replace(valid, pair.Close, "", 1),
		"duplicate":  valid + pair.Open,
		"extra raw":  "* 多个返回值\n\n`swap` 调用" + pair.Open + "swap" + pair.Close + "函数会返回两个字符串。\n",
	} {
		t.Run(name, func(t *testing.T) {
			got, rejected := p.restore(model)
			if name != "extra raw" {
				if len(rejected) == 0 || got != "" {
					t.Fatalf("mutated inline pair accepted: %q %v", got, rejected)
				}
				return
			}
			if len(rejected) != 0 {
				t.Fatal(rejected)
			}
			if err := ValidateCandidate(root, catalog, "synthetic/basics6", []byte(got)); err == nil {
				t.Fatal("extra raw inline code accepted")
			}
		})
	}
}

func TestInlineCodePairsMayReorderButCannotCrossOrChangeContent(t *testing.T) {
	source := "* Source\n\n`int` and `int`: `:=`, `<-`, `fmt.Println`, `math.Sqrt`, `T comparable`.\n"
	p := protectTranslation([]byte(source), strings.Repeat("c", 64), nil)
	if len(p.InlinePairs) != 6 {
		t.Fatalf("inline pairs=%+v", p.InlinePairs)
	}
	if _, failures := p.restore(p.Text); len(failures) != 0 {
		t.Fatal(failures)
	}
	first, second := p.InlinePairs[0], p.InlinePairs[1]
	crossed := strings.NewReplacer(first.Open, "__open__", first.Close, second.Close, second.Open, first.Open, "__open__", second.Open).Replace(p.Text)
	if _, failures := p.restore(crossed); len(failures) == 0 {
		t.Fatal("crossed inline pairs accepted")
	}
}

func TestInlineTokenBoundaryNormalizationUsesResponsePairOrder(t *testing.T) {
	source := []byte("* Source\n\n`first`、 `second`; each `inner` inside `outer`.\n")
	p := protectTranslation(source, strings.Repeat("d", 64), nil)
	if len(p.InlinePairs) != 4 {
		t.Fatalf("inline pairs=%+v", p.InlinePairs)
	}
	first, second, inner, outer := p.InlinePairs[0], p.InlinePairs[1], p.InlinePairs[2], p.InlinePairs[3]
	model := "* 译文\n\n" +
		first.Open + first.Content + first.Close + "、" + second.Open + second.Content + second.Close +
		"；在 " + outer.Open + outer.Content + outer.Close + " 中的每个 " + inner.Open + inner.Content + inner.Close + "。\n"

	candidate, failures := p.restore(model)
	if len(failures) != 0 {
		t.Fatalf("reordered whole pairs rejected: %v", failures)
	}
	for _, want := range []string{"`first`、 `second`", "`outer`", "`inner`"} {
		if !strings.Contains(candidate, want) {
			t.Errorf("candidate lacks normalized fragment %q:\n%s", want, candidate)
		}
	}
	codes := presentInlineCodes(candidate)
	if len(codes) != 4 {
		t.Fatalf("presentInlineCodes = %+v, want 4 spans\n%s", codes, candidate)
	}

	for name, bad := range map[string]string{
		"missing pair token": strings.Replace(model, inner.Open, "", 1),
		"duplicate token":    model + inner.Open,
		"crossed pairs": strings.NewReplacer(
			inner.Open, "__inner_open__",
			inner.Close, outer.Close,
			outer.Open, inner.Open,
			"__inner_open__", outer.Open,
		).Replace(model),
		"mismatched close": strings.NewReplacer(
			inner.Close, "__inner_close__",
			outer.Close, inner.Close,
			"__inner_close__", outer.Close,
		).Replace(model),
	} {
		t.Run(name, func(t *testing.T) {
			if got, failures := p.restore(bad); len(failures) == 0 || got != "" {
				t.Fatalf("invalid pair accepted: %q %v", got, failures)
			}
		})
	}
}

func TestMoretypes18HistoricalResponsesRestoreNineInlineCodes(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("moretypes/18")
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	protected := protectTranslation(page.Source, page.SourceSHA256, glossary)
	if len(protected.InlinePairs) != 9 {
		t.Fatalf("inline pairs=%d, want 9", len(protected.InlinePairs))
	}
	for attempt := 1; attempt <= 3; attempt++ {
		t.Run(fmt.Sprintf("attempt-%03d", attempt), func(t *testing.T) {
			responsePath := filepath.Join(root, "data", "translation-runs", "zh-CN", "moretypes", "18", "sources", page.SourceSHA256, fmt.Sprintf("attempt-%03d", attempt), "response.json")
			responseBytes, err := os.ReadFile(responsePath)
			if err != nil {
				t.Fatal(err)
			}
			var response TranslationCallResult
			if err := json.Unmarshal(responseBytes, &response); err != nil {
				t.Fatal(err)
			}
			candidate, failures := protected.restore(response.Content)
			if len(failures) != 0 {
				t.Fatalf("historical response restore failures: %v", failures)
			}
			codes := presentInlineCodes(candidate)
			if len(codes) != 9 {
				t.Fatalf("presentInlineCodes = %+v, want 9 spans\n%s", codes, candidate)
			}
			if !strings.Contains(candidate, "`(x+y)/2`、 `x*y`") {
				t.Fatalf("historical response lacks normalized enumeration boundary:\n%s", candidate)
			}
			if err := ValidateCandidateForLocale(root, catalog, "moretypes/18", "zh-CN", []byte(candidate)); err != nil {
				t.Fatalf("historical response candidate rejected: %v\n%s", err, candidate)
			}
		})
	}
}

func TestFlowcontrol6InlinePairsMayReorderAsWholeUnits(t *testing.T) {
	source := "* If with a short statement\n\n(Try using `v` in the last `return` statement.)\n"
	p := protectTranslation([]byte(source), strings.Repeat("f", 64), nil)
	if len(p.InlinePairs) != 2 {
		t.Fatalf("pairs=%+v", p.InlinePairs)
	}
	v, returned := p.InlinePairs[0], p.InlinePairs[1]
	model := "* 带简短语句的 if\n\n（试着在最后一个 " + returned.Open + "return" + returned.Close + " 语句中使用 " + v.Open + "v" + v.Close + "。）\n"
	candidate, failures := p.restore(model)
	if len(failures) != 0 {
		t.Fatalf("whole-pair reorder rejected: %v", failures)
	}
	root := repoRoot(t)
	catalog := &Catalog{Pages: []Page{{ID: "synthetic/flowcontrol6", Article: "flowcontrol.article", Source: []byte(source), SourceSHA256: sum([]byte(source))}}}
	if err := ValidateCandidate(root, catalog, "synthetic/flowcontrol6", []byte(candidate)); err != nil {
		t.Fatalf("reordered candidate rejected: %v\n%s", err, candidate)
	}
	for name, bad := range map[string]string{
		"opening missing": strings.Replace(model, returned.Open, "", 1),
		"closing missing": strings.Replace(model, returned.Close, "", 1),
		"content changed": strings.Replace(model, "return", "returns", 1),
		"duplicate":       model + returned.Open,
	} {
		t.Run(name, func(t *testing.T) {
			if got, failures := p.restore(bad); len(failures) == 0 || got != "" {
				t.Fatalf("invalid pair accepted: %q %v", got, failures)
			}
		})
	}
	extra := "* 带简短语句的 if\n\n（`return` 试着在最后一个 " + returned.Open + "return" + returned.Close + " 语句中使用 " + v.Open + "v" + v.Close + "。）\n"
	got, failures := p.restore(extra)
	if len(failures) != 0 {
		t.Fatal(failures)
	}
	if err := ValidateCandidate(root, catalog, "synthetic/flowcontrol6", []byte(got)); err == nil {
		t.Fatal("extra raw inline code accepted")
	}
}

func TestRevalidateSavedTranslationResponseFlowcontrol6OrderRegression(t *testing.T) {
	source := []byte("* If with a short statement\n\n(Try using `v` in the last `return` statement.)\n")
	root := writeRevalidationFixture(t, "flowcontrol/6", source, 3, "blocked")
	catalog := &Catalog{Pages: []Page{{ID: "flowcontrol/6", Article: "flowcontrol.article", Source: source, SourceSHA256: sum(source)}}}
	p := protectTranslation(source, sum(source), nil)
	v, returned := p.InlinePairs[0], p.InlinePairs[1]
	content := "* 带简短语句的 if\n\n（试着在最后一个 " + returned.Open + "return" + returned.Close + " 语句中使用 " + v.Open + "v" + v.Close + "。）\n"
	writeRevalidationAttempt(t, root, "flowcontrol/6", "zh-CN", sum(source), 3, content, "stop", true)
	result, err := RevalidateSavedTranslationResponse(root, catalog, "flowcontrol/6", "zh-CN", 3, func() time.Time { return time.Unix(0, 0) })
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ready" || result.Attempts != 3 || result.SourceAttempt != 3 {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(filepath.Join(root, result.CandidatePath)); err != nil {
		t.Fatal(err)
	}
	status, _, err := LoadTranslationResult(root, "flowcontrol/6", "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "ready" || status.Attempts != 3 {
		t.Fatalf("status=%+v", status)
	}
	for _, name := range []string{"request.json", "response.json", "validation.json"} {
		if _, err := os.Stat(filepath.Join(root, "data", "translation-runs", "zh-CN", "flowcontrol", "6", "sources", sum(source), "attempt-003", name)); err != nil {
			t.Fatal(err)
		}
	}
	auditPath := filepath.Join(root, "data", "translation-runs", "zh-CN", "flowcontrol", "6", "sources", sum(source), "revalidation-001.json")
	if _, err := os.Stat(auditPath); err != nil {
		t.Fatal(err)
	}
	var audit responseRevalidationRecord
	auditBytes, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(auditBytes, &audit); err != nil {
		t.Fatal(err)
	}
	for _, got := range []string{audit.ResponsePath, audit.ValidationPath} {
		if filepath.IsAbs(got) || strings.Contains(got, root) {
			t.Fatalf("audit path is not repository-relative: %q", got)
		}
	}
	wantDir := filepath.ToSlash(filepath.Join("data", "translation-runs", "zh-CN", "flowcontrol", "6", "sources", sum(source), "attempt-003"))
	if audit.ResponsePath != wantDir+"/response.json" || audit.ValidationPath != wantDir+"/validation.json" {
		t.Fatalf("audit paths = %q, %q", audit.ResponsePath, audit.ValidationPath)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "translation-runs", "zh-CN", "flowcontrol", "6", "sources", sum(source), "attempt-004")); !os.IsNotExist(err) {
		t.Fatalf("attempt-004 unexpectedly exists: %v", err)
	}
}

func TestRevalidateSavedTranslationResponseFailsClosedAndAppendsAudit(t *testing.T) {
	source := []byte("* Source\n\nUse `code`.\n")
	root := writeRevalidationFixture(t, "example/1", source, 3, "blocked")
	catalog := &Catalog{Pages: []Page{{ID: "example/1", Article: "basics.article", Source: source, SourceSHA256: sum(source)}}}
	p := protectTranslation(source, sum(source), nil)
	bad := "* 来源\n\n使用 " + p.InlinePairs[0].Open + "changed" + p.InlinePairs[0].Close + "。\n"
	writeRevalidationAttempt(t, root, "example/1", "zh-CN", sum(source), 3, bad, "stop", true)
	if _, err := RevalidateSavedTranslationResponse(root, catalog, "example/1", "zh-CN", 3, func() time.Time { return time.Unix(0, 0) }); err == nil {
		t.Fatal("bad restore accepted")
	}
	if _, err := RevalidateSavedTranslationResponse(root, catalog, "example/1", "zh-CN", 3, func() time.Time { return time.Unix(1, 0) }); err == nil {
		t.Fatal("second bad restore accepted")
	}
	status, _, _ := LoadTranslationResult(root, "example/1", "zh-CN")
	if status.State != "blocked" || status.Attempts != 3 {
		t.Fatalf("status=%+v", status)
	}
	dir := filepath.Join(root, "data", "translation-runs", "zh-CN", "example", "1", "sources", sum(source))
	for _, name := range []string{"revalidation-001.json", "revalidation-002.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRevalidateSavedTranslationResponseEligibility(t *testing.T) {
	source := []byte("* Source\n\nUse `code`.\n")
	validCatalog := &Catalog{Pages: []Page{{ID: "example/1", Article: "basics.article", Source: source, SourceSHA256: sum(source)}}}
	validResponse := func(p protectedTranslation) string {
		return "* 来源\n\n使用 " + p.InlinePairs[0].Open + "code" + p.InlinePairs[0].Close + "。\n"
	}
	for _, tt := range []struct {
		name, state     string
		attempt         int
		content, finish string
		api             bool
	}{
		{"not blocked", "pending", 3, "", "", false},
		{"attempt missing", "blocked", 3, "", "", false},
		{"empty content", "blocked", 3, "", "stop", true},
		{"non stop", "blocked", 3, "ignored", "length", true},
		{"network attempt", "blocked", 3, "ignored", "stop", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := writeRevalidationFixture(t, "example/1", source, 3, tt.state)
			if tt.name != "attempt missing" {
				writeRevalidationAttempt(t, root, "example/1", "zh-CN", sum(source), tt.attempt, tt.content, tt.finish, tt.api)
			}
			if tt.name == "empty content" { /* deliberately empty */
			} else if tt.content == "ignored" {
				p := protectTranslation(source, sum(source), nil)
				writeRevalidationAttempt(t, root, "example/1", "zh-CN", sum(source), tt.attempt, validResponse(p), tt.finish, tt.api)
			}
			if _, err := RevalidateSavedTranslationResponse(root, validCatalog, "example/1", "zh-CN", tt.attempt, nil); err == nil {
				t.Fatal("ineligible response accepted")
			}
			status, _, _ := LoadTranslationResult(root, "example/1", "zh-CN")
			if status.State != tt.state || status.Attempts != 3 {
				t.Fatalf("status changed: %+v", status)
			}
		})
	}
	t.Run("source hash changed", func(t *testing.T) {
		root := writeRevalidationFixture(t, "example/1", source, 3, "blocked")
		changed := []byte("* Source\n\nUse `other`.\n")
		changedCatalog := &Catalog{Pages: []Page{{ID: "example/1", Article: "basics.article", Source: changed, SourceSHA256: sum(changed)}}}
		if _, err := RevalidateSavedTranslationResponse(root, changedCatalog, "example/1", "zh-CN", 3, nil); err == nil {
			t.Fatal("stale source hash accepted")
		}
	})
}

func TestRevalidateSavedTranslationResponseUsesRequestedLocale(t *testing.T) {
	source := []byte("* Source\n\nUse `code`.\n")
	root := writeRevalidationFixture(t, "example/1", source, 3, "blocked")
	locale := "test-locale"
	cloneRevalidationLocale(t, root, locale)
	catalog := &Catalog{Pages: []Page{{ID: "example/1", Article: "basics.article", Source: source, SourceSHA256: sum(source)}}}
	p := protectTranslation(source, sum(source), nil)
	content := "* 来源\n\n使用 " + p.InlinePairs[0].Open + "code" + p.InlinePairs[0].Close + "。\n"
	writeRevalidationAttempt(t, root, "example/1", locale, sum(source), 3, content, "stop", true)
	if _, err := RevalidateSavedTranslationResponse(root, catalog, "example/1", locale, 3, nil); err != nil {
		t.Fatal(err)
	}
	root = writeRevalidationFixture(t, "example/1", source, 3, "blocked")
	cloneRevalidationLocale(t, root, locale)
	writeRevalidationAttempt(t, root, "example/1", locale, sum(source), 3, content, "stop", true)
	path := filepath.Join(root, "data", "translation-runs", locale, "example", "1", "sources", sum(source), "attempt-003", "request.json")
	var request savedTranslationRequest
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &request); err != nil {
		t.Fatal(err)
	}
	request.Locale = "other-locale"
	if err := writeTranslationJSON(path, request); err != nil {
		t.Fatal(err)
	}
	if _, err := RevalidateSavedTranslationResponse(root, catalog, "example/1", locale, 3, nil); err == nil {
		t.Fatal("mismatched request locale accepted")
	}
	if _, err := RevalidateSavedTranslationResponse(root, catalog, "example/1", "missing-locale", 3, nil); err == nil {
		t.Fatal("invalid locale accepted")
	}
}

func cloneRevalidationLocale(t *testing.T, root, locale string) {
	t.Helper()
	dir := filepath.Join(root, "locales", locale)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"status.tsv", "glossary.yaml"} {
		b, err := os.ReadFile(filepath.Join(root, "locales", "zh-CN", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func writeRevalidationFixture(t *testing.T, pageID string, source []byte, attempts int, state string) string {
	t.Helper()
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+pageID+"\t"+state+"\t"+fmt.Sprint(attempts)+"\t"+sum(source)+"\t\t\t\n")
	writeTestGlossary(t, root)
	return root
}

func writeRevalidationAttempt(t *testing.T, root, pageID, locale, hash string, attempt int, content, finish string, apiSuccess bool) {
	t.Helper()
	dir := filepath.Join(root, "data", "translation-runs", locale, pageID, "sources", hash, fmt.Sprintf("attempt-%03d", attempt))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeTranslationJSON(filepath.Join(dir, "request.json"), savedTranslationRequest{pageID, locale, hash, TranslationAPIRequest{Model: "glm-5.2"}}); err != nil {
		t.Fatal(err)
	}
	if err := writeTranslationJSON(filepath.Join(dir, "response.json"), TranslationCallResult{StatusCode: 200, RequestID: "audit", FinishReason: finish, Content: content, Raw: json.RawMessage(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if err := writeTranslationJSON(filepath.Join(dir, "validation.json"), TranslationValidation{Attempt: attempt, APISuccess: apiSuccess, Failures: []string{"old validator failed"}}); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyInlineCodeRemainsOneProtectedSpan(t *testing.T) {
	tests := []struct {
		raw, content string
	}{
		{"`package`rand`", "package rand"},
		{"`for`i`:=`range`c`", "for i := range c"},
		{"`Same(tree.New(1),`tree.New(1))`", "Same(tree.New(1), tree.New(1))"},
		{"`\"cannot`Sqrt`negative`number:`-2\"`", "\"cannot Sqrt negative number: -2\""},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			p := protectTranslation([]byte(tt.raw), "12345678", nil)
			rawContent := tt.raw[1 : len(tt.raw)-1]
			if len(p.InlinePairs) != 1 || p.InlinePairs[0].Content != rawContent || !strings.Contains(p.Text, rawContent) {
				t.Fatalf("protection = %+v, want one visible inline-code pair", p)
			}
			normalized := normalizeInlineTokenBoundaries(p.Text, p)
			if normalized != p.Text {
				t.Fatalf("normalization changed isolated token: %q => %q", p.Text, normalized)
			}
			got, failures := p.restore(p.Text)
			if len(failures) != 0 || got != tt.raw {
				t.Fatalf("restore = %q, %v; want %q", got, failures, tt.raw)
			}
			codes := presentInlineCodes(got)
			if len(codes) != 1 || codes[0].Raw != tt.raw || codes[0].Content != tt.content {
				t.Fatalf("presentInlineCodes(%q) = %+v; want raw %q content %q", got, codes, tt.raw, tt.content)
			}
		})
	}
}

func TestNonInlineTokenKindsAreNotBoundaryNormalized(t *testing.T) {
	glossary := &Glossary{Mandatory: map[string]string{"A Tour of Go": "Go 语言之旅"}}
	source := []byte("Go [[/target][A Tour of Go]]\n\n.play example.go")
	p := protectTranslation(source, "12345678", glossary)
	wantKinds := map[protectedTokenKind]bool{
		protectedDirective:      false,
		protectedLinkTarget:     false,
		protectedGlossaryOrKeep: false,
	}
	for _, kind := range p.Kinds {
		if kind == protectedInlineCodeOpen || kind == protectedInlineCodeClose {
			t.Fatalf("unexpected inline token in %+v", p)
		}
		if _, ok := wantKinds[kind]; ok {
			wantKinds[kind] = true
		}
	}
	for kind, found := range wantKinds {
		if !found {
			t.Errorf("protected kind %v not assigned: %+v", kind, p)
		}
	}
	if got := normalizeInlineTokenBoundaries(p.Text, p); got != p.Text {
		t.Fatalf("non-inline tokens changed:\ngot:  %q\nwant: %q", got, p.Text)
	}
}

func TestLinkLabelProgramSpanProtectionRoundTrip(t *testing.T) {
	source := []byte("* Link\n\n[[/pkg/][Use `pkg.Type` here]].\n")
	p := protectTranslation(source, "12345678", nil)
	if len(p.Tokens) != 3 || p.Kinds[0] != protectedLinkTarget || len(p.InlinePairs) != 1 || p.InlinePairs[0].Boundaries {
		t.Fatalf("protection = %+v, want target then inline-code pair", p)
	}
	wantProtected := "[[" + p.Tokens[0] + "][Use " + inlinePairText(t, p, 0) + " here]]"
	if !strings.Contains(p.Text, wantProtected) {
		t.Fatalf("link label program span not independently protected:\n%s", p.Text)
	}
	model := "* 链接\n\n[[" + p.Tokens[0] + "][使用 " + inlinePairText(t, p, 0) + "]]。\n"
	got, failures := p.restore(model)
	if len(failures) != 0 || got != "* 链接\n\n[[/pkg/][使用 `pkg.Type`]]。\n" {
		t.Fatalf("restore = %q, failures=%v", got, failures)
	}
}

func TestMethods24ProtectionIncludesLinkLabelProgramSpan(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("methods/24")
	if err != nil {
		t.Fatal(err)
	}
	p := protectTranslation(page.Source, page.SourceSHA256, nil)
	counts := map[protectedTokenKind]int{}
	for _, kind := range p.Kinds {
		counts[kind]++
	}
	if counts[protectedLinkTarget] != 4 || counts[protectedDirective] != 1 || counts[protectedPreformattedStatic] != 1 || counts[protectedInlineCodeOpen] != 9 || counts[protectedInlineCodeClose] != 9 || counts[protectedBoldOpen] != 1 || counts[protectedBoldClose] != 1 || len(p.Tokens) != 26 {
		t.Fatalf("protected counts = %+v, total=%d; want links=4 directive=1 static=1 inline-open=9 inline-close=9 bold-open=1 bold-close=1 total=26", counts, len(p.Tokens))
	}
	target, code := "", ""
	for i, value := range p.Values {
		switch value {
		case "/pkg/image/#Rectangle":
			target = p.Tokens[i]
		}
	}
	for pair := range p.InlinePairs {
		if p.InlinePairs[pair].Content == "image.Rectangle" {
			code = inlinePairText(t, p, pair)
		}
	}
	if target == "" || code == "" || !strings.Contains(p.Text, "[["+target+"]["+code+"]]") {
		t.Fatalf("methods/24 linked program protection missing: target=%q code=%q\n%s", target, code, p.Text)
	}
}

func TestInlineTokenBoundaryNormalizationForLegacyPresent(t *testing.T) {
	root := repoRoot(t)
	source := "* Root\n\nUse `code` here.\n"
	catalog := &Catalog{Pages: []Page{{ID: "synthetic/inline-boundary", Article: "basics.article", Source: []byte(source), SourceSHA256: sum([]byte(source))}}}
	tests := []struct {
		name, prefix, suffix, want string
	}{
		{"full-width colon before", "说明：", " 方法。", "说明： `code` 方法。"},
		{"full-width comma before", "说明，", " 方法。", "说明， `code` 方法。"},
		{"full-width period before", "说明。", " 方法。", "说明。 `code` 方法。"},
		{"full-width opening parenthesis", "说明（", "）", "说明（ `code`）"},
		{"full-width closing parenthesis after", "调用 ", "）。", "调用 `code`）。"},
		{"full-width comma after", "调用 ", "，继续。", "调用 `code`，继续。"},
		{"full-width period after", "调用 ", "。", "调用 `code`。"},
		{"han characters on both sides", "使用", "的结果。", "使用 `code` 的结果。"},
		{"existing ASCII spaces", "use ", " here.", "use `code` here."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := protectTranslation([]byte(source), "12345678", nil)
			candidate, failures := p.restore("* 根\n\n" + tt.prefix + inlinePairText(t, p, 0) + tt.suffix + "\n")
			if len(failures) != 0 {
				t.Fatal(failures)
			}
			if !strings.Contains(candidate, tt.want) {
				t.Fatalf("restore = %q, want fragment %q", candidate, tt.want)
			}
			if err := ValidateCandidate(root, catalog, "synthetic/inline-boundary", []byte(candidate)); err != nil {
				t.Fatalf("legacy present did not recognize inline code: %v\n%s", err, candidate)
			}
		})
	}
}

func TestMethods24HistoricalResponseWithoutEmphasisSentinelsIsRejected(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("methods/24")
	if err != nil {
		t.Fatal(err)
	}
	p := protectTranslation(page.Source, page.SourceSHA256, nil)
	record, err := os.ReadFile(filepath.Join(root, "data", "translation-runs", "zh-CN", "methods", "24", "sources", page.SourceSHA256, "attempt-001", "response.json"))
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(record, &response); err != nil {
		t.Fatal(err)
	}
	if _, failures := p.restore(response.Content); len(failures) == 0 {
		t.Fatal("historical response without new emphasis sentinels was accepted")
	}
}

func TestTranslationKeepSkipsOnlyHighConfidenceOrdinaryGoVerb(t *testing.T) {
	glossary, err := LoadGlossary(repoRoot(t), "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	keepOnly := &Glossary{Keep: glossary.Keep}
	tests := []struct {
		name, source string
		wantGo       bool
	}{
		{"where to Go", "* Where to Go from here...\n", false},
		{"where to Go with whitespace and case variation", "* WHERE\tto  Go next\n", false},
		{"Go language subject", "Go has only one looping construct.\n", true},
		{"Go language possessive", "Go's switch is like C's.\n", true},
		{"Go language object", "Programs written in Go are portable.\n", true},
		{"install Go", "Install Go before continuing.\n", true},
		{"Go Documentation", "[[/doc/][Go Documentation]]\n", true},
		{"A Tour of Go", "[[/tour/][A Tour of Go]]\n", true},
		{"migrate to Go from another language", "Migrate to Go from another language.\n", true},
		{"Go local remains conservative", "* Go local\n", true},
		{"Go offline remains conservative", "* Go offline (optional)\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := protectTranslation([]byte(tt.source), "12345678", keepOnly)
			gotGo := false
			for i, value := range p.Values {
				if value != "Go" {
					continue
				}
				if p.Kinds[i] != protectedGlossaryOrKeep {
					t.Fatalf("Go token kind = %v, want glossary/keep", p.Kinds[i])
				}
				gotGo = true
			}
			if gotGo != tt.wantGo {
				t.Fatalf("Go protected = %t, want %t; values=%q", gotGo, tt.wantGo, p.Values)
			}
		})
	}

	for _, keep := range []string{"gofmt", "PageUp", "PageDown", "Shift", "Enter", "Ctrl"} {
		t.Run("other keep "+keep, func(t *testing.T) {
			p := protectTranslation([]byte(keep), "12345678", keepOnly)
			if len(p.Values) != 1 || p.Values[0] != keep || p.Kinds[0] != protectedGlossaryOrKeep {
				t.Fatalf("protection = %+v, want %q as glossary/keep", p, keep)
			}
		})
	}
}

func TestTranslationKeepProtectsGoroutineButNotMap(t *testing.T) {
	glossary, err := LoadGlossary(repoRoot(t), "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	source := []byte("Go gofmt PageUp PageDown Shift Enter Ctrl goroutine goroutines Goroutines map maps `Go`\n")
	p := protectTranslation(source, "12345678", glossary)
	values := strings.Join(p.Values, "\n")
	for _, keep := range glossary.Keep {
		if !strings.Contains(values, keep) {
			t.Errorf("keep %q was not protected: %q", keep, p.Values)
		}
	}
	for _, visible := range []string{"map", "maps"} {
		if !strings.Contains(p.Text, visible) {
			t.Errorf("%q was unexpectedly protected: %s", visible, p.Text)
		}
		if containsString(p.Values, visible) {
			t.Errorf("%q entered keep protection: values=%q", visible, p.Values)
		}
	}
	for _, keep := range []string{"goroutine", "goroutines", "Goroutines"} {
		if strings.Contains(p.Text, keep) {
			t.Errorf("%q remained visible instead of being protected: %s", keep, p.Text)
		}
		if !containsString(p.Values, keep) {
			t.Errorf("%q missing from protected values: %q", keep, p.Values)
		}
	}
	goValues := 0
	for _, value := range p.Values {
		if value == "Go" {
			goValues++
		}
	}
	if goValues != 1 {
		t.Fatalf("Go protected values = %d, want 1 outside inline code; values=%q", goValues, p.Values)
	}
	restored, failures := p.restore(p.Text)
	if len(failures) != 0 {
		t.Fatalf("restore failures: %v", failures)
	}
	if restored != string(source) {
		t.Fatalf("restored = %q, want %q", restored, source)
	}
}

func TestTranslationKeepBoundariesAllowPresentEmphasis(t *testing.T) {
	glossary, err := LoadGlossary(repoRoot(t), "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"goroutine", "(goroutine)", "goroutine.", "goroutines", "Goroutines"} {
		t.Run(source, func(t *testing.T) {
			p := protectTranslation([]byte(source), "12345678", glossary)
			if strings.Contains(p.Text, "goroutine") || strings.Contains(p.Text, "Goroutines") {
				t.Fatalf("keep remained visible: %s", p.Text)
			}
		})
	}

	p := protectTranslation([]byte("_goroutine_"), "12345678", glossary)
	if !reflect.DeepEqual(p.Values, []string{"_", "goroutine", "_"}) {
		t.Fatalf("protected values = %q, want emphasis/keep/emphasis", p.Values)
	}
	if p.Text != strings.Join(p.Tokens, "") {
		t.Fatalf("protected emphasis = %q, want three adjacent tokens", p.Text)
	}
	restored, failures := p.restore(p.Text)
	if len(failures) != 0 || restored != "_goroutine_" {
		t.Fatalf("restore = %q, failures=%v", restored, failures)
	}

	for _, source := range []string{"mygoroutine", "goroutineWorker", "foo_goroutine_bar", "map", "maps"} {
		t.Run("visible "+source, func(t *testing.T) {
			p := protectTranslation([]byte(source), "12345678", glossary)
			if p.Text != source || len(p.Values) != 0 {
				t.Fatalf("%q was unexpectedly protected: text=%q values=%q", source, p.Text, p.Values)
			}
		})
	}
}

func TestConcurrency1ProtectsEmphasizedGoroutine(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("concurrency/1")
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	p := prepareDefaultTranslationInput(page.Source, page.SourceSHA256, glossary)
	if strings.Contains(p.Text, "goroutine") {
		t.Fatalf("concurrency/1 left emphasized goroutine visible:\n%s", p.Text)
	}
	found := false
	for i, value := range p.Values {
		if value == "goroutine" && i > 0 && i+1 < len(p.Values) && p.Values[i-1] == "_" && p.Values[i+1] == "_" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("concurrency/1 did not produce emphasis/keep/emphasis spans: values=%q", p.Values)
	}
}

func TestConcurrency11ProjectedProtectionSkipsOnlyTitleGo(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("concurrency/11")
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	p := protectTranslation(page.Source, page.SourceSHA256, glossary)
	var links, keeps int
	for _, kind := range p.Kinds {
		switch kind {
		case protectedLinkTarget:
			links++
		case protectedGlossaryOrKeep:
			keeps++
		}
	}
	if links != 15 || keeps != 11 || len(p.Tokens) != 26 {
		t.Fatalf("protected counts links=%d keeps=%d total=%d, want 15/11/26", links, keeps, len(p.Tokens))
	}
	if strings.Contains(p.Text, "Where to ⟪GTI18N_") || !strings.Contains(p.Text, "* Where to Go from here...") {
		t.Fatalf("title Go should remain translatable:\n%s", p.Text)
	}
	if n := strings.Count(strings.Join(p.Values, "\n"), "Go"); n != 11 {
		t.Fatalf("protected Go values = %d, want 11; values=%q", n, p.Values)
	}
}

func TestGenericsInlineBoundaryNormalizationPassesValidator(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("generics/1")
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	p := protectTranslation(page.Source, page.SourceSHA256, glossary)
	comparable, x := "", ""
	for _, pair := range p.InlinePairs {
		if pair.Content == "comparable" && comparable == "" {
			comparable = pair.Open + pair.Content + pair.Close
		}
		if pair.Content == "x" && x == "" {
			x = pair.Open + pair.Content + pair.Close
		}
	}
	if comparable == "" || x == "" {
		t.Fatalf("target inline pairs not found: %+v", p.InlinePairs)
	}
	model := strings.Replace(p.Text, comparable+". "+x, comparable+"。"+x, 1)
	if model == p.Text {
		t.Fatal("attempt-003 equivalent protected text was not constructed")
	}
	candidate, failures := p.restore(model)
	if len(failures) != 0 {
		t.Fatalf("restore failures: %v", failures)
	}
	if !strings.Contains(candidate, "`comparable`。 `x`") {
		t.Fatalf("candidate lacks normalized boundary:\n%s", candidate)
	}
	if err := ValidateCandidate(root, catalog, "generics/1", []byte(candidate)); err != nil {
		t.Fatalf("normalized candidate validation: %v\n%s", err, candidate)
	}
}

func TestPresentInlineCodeProtection(t *testing.T) {
	tests := []struct {
		name, source, raw, content string
	}{
		{"ordinary", "Use `main`.", "`main`", "main"},
		{"legacy space", "statement `package`rand`.", "`package`rand`", "package rand"},
		{"multiple legacy spaces", "call `go`test`./...`.", "`go`test`./...`", "go test ./..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes := presentInlineCodes(tt.source)
			if len(codes) != 1 || codes[0].Raw != tt.raw || codes[0].Content != tt.content {
				t.Fatalf("presentInlineCodes(%q) = %+v", tt.source, codes)
			}
			if strings.HasSuffix(codes[0].Raw, ".") {
				t.Fatalf("trailing period included in code span: %q", codes[0].Raw)
			}
			p := protectTranslation([]byte(tt.source), sum([]byte(tt.source)), nil)
			if len(p.InlinePairs) != 1 || p.InlinePairs[0].Content != tt.raw[1:len(tt.raw)-1] {
				t.Fatalf("inline pairs = %+v, want visible %q", p.InlinePairs, tt.raw[1:len(tt.raw)-1])
			}
			restored, failures := p.restore(p.Text)
			if len(failures) != 0 || restored != tt.source {
				t.Fatalf("restore = %q, %v; want %q", restored, failures, tt.source)
			}
		})
	}
}

func TestBasicsPackagesLegacyInlineCodeIsFullyProtected(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("basics/1")
	if err != nil {
		t.Fatal(err)
	}
	p := protectTranslation(page.Source, page.SourceSHA256, nil)
	var pair protectedInlinePair
	for _, current := range p.InlinePairs {
		if current.Content == "package`rand" {
			pair = current
			break
		}
	}
	if pair.Open == "" {
		t.Fatalf("legacy inline-code pair absent: %+v", p.InlinePairs)
	}
	want := "statement " + pair.Open + "package`rand" + pair.Close + "."
	if !strings.Contains(p.Text, want) || !strings.Contains(p.Text, "package`rand") {
		t.Fatalf("protected basics/1 does not contain %q cleanly:\n%s", want, p.Text)
	}
	candidate, err := os.ReadFile(filepath.Join(root, "locales", "zh-CN", "candidates", "basics-1.article"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateCandidate(root, catalog, "basics/1", candidate); err != nil {
		t.Fatalf("basics/1 candidate validation: %v", err)
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

func TestTranslationRequestIncludesNaturalChineseGuidance(t *testing.T) {
	source := []byte("* Contextual title\n\nGo uses `T`.\n\n.play example/example.go\n")
	protected := protectTranslation(source, sum(source), nil)
	request := makeTranslationRequest("example/1", "zh-CN", protected, "- glossary rule", "")
	if len(request.Messages) != 2 || request.Messages[0].Role != "system" || request.Messages[1].Role != "user" {
		t.Fatalf("messages = %+v", request.Messages)
	}
	system := request.Messages[0].Content
	wants := []string{
		"请将一个完整的《Go 语言之旅》present.Section 从英文翻译为中国大陆简体中文。",
		"只返回完整且可由 present 解析的 .article 内容。",
		"原样出现、恰好出现一次；不得修改、删除、复制或伪造",
		"为适应目标语言自然语序可以调整 token 位置",
		"应当翻译的英文显示文本不得残留",
		"不得简化、遗漏或改变原文含义",
		"理解完整 present.Section 的页面用途和上下文",
		"页面标题应简洁、自然并准确概括页面主题",
		"中国大陆简体中文技术教程风格",
		"按钮应点击；链接应点击；键盘按键应按或按下；命令应执行；文本内容应输入",
		"不得自行新增或删除行内代码反引号、预格式化代码、present directive、链接及链接 target、HTML 或特殊 present 语法",
		"只输出最终完整的 present.Section，不输出分析、说明或修改过程",
	}
	for _, want := range wants {
		if !strings.Contains(system, want) {
			t.Errorf("system prompt missing %q", want)
		}
	}
	for _, removed := range []string{
		"Translate one complete A Tour of Go present.Section",
		"Preserve every protection token exactly once and in order.",
		"Mandatory glossary translations must be used",
	} {
		if strings.Contains(system, removed) {
			t.Errorf("system prompt retains English fixed instruction %q", removed)
		}
	}
	user := request.Messages[1].Content
	for _, want := range []string{
		"唯一占位符",
		"输出中也必须恰好包含",
		"不得复制",
		"带有结构角色",
		"始终保持原有结构角色和所属关系",
		fmt.Sprintf("本页共有 %d 个保护 token，输出中也必须恰好包含 %d 个。", len(protected.Tokens), len(protected.Tokens)),
	} {
		if !strings.Contains(user, want) {
			t.Errorf("user prompt missing protected token rule %q", want)
		}
	}
	if !strings.Contains(user, "强制术语表与译法规则：\n- glossary rule") || !strings.Contains(user, "需要翻译的完整受保护页面：\n* Contextual title") {
		t.Fatalf("user prompt lost glossary or protected page:\n%s", user)
	}
	for _, removed := range []string{"Mandatory glossary rules:", "Complete protected page:", "TOKEN_T", "comparable"} {
		if strings.Contains(user, removed) {
			t.Errorf("user prompt contains forbidden fixed text %q", removed)
		}
	}

	retry := makeTranslationRequest("example/1", "zh-CN", protected, "- glossary rule", retryFeedback([]string{"protected token order mismatch at 1"}))
	retryUser := retry.Messages[1].Content
	for _, want := range []string{"上一次完整页面翻译未通过校验：上一次输出未能完整、唯一地保留所有受保护 token。", "每个现有 token 必须原样且恰好出现一次", "请重新翻译完整页面"} {
		if !strings.Contains(retryUser, want) {
			t.Errorf("retry prompt missing %q:\n%s", want, retryUser)
		}
	}
	if strings.Contains(retryUser, "Previous full-page attempt failed validation:") || strings.Contains(retryUser, "Translate the complete page again.") {
		t.Errorf("retry prompt retains English fixed instruction:\n%s", retryUser)
	}
}

func TestRawInputTranslationRequestUsesOriginalPageWithoutTokenInstructions(t *testing.T) {
	source := []byte("* Title\n\nSee [[/target][a link]] and `code`.\n\n.play welcome/hello.go\n")
	request := makeTranslationRequestForMode("example/1", "zh-CN", source, nil, "- glossary rule", "")
	if len(request.Messages) != 2 {
		t.Fatalf("messages = %+v", request.Messages)
	}
	all := request.Messages[0].Content + "\n" + request.Messages[1].Content
	if translationTokenRE.MatchString(all) || strings.Contains(all, "保护 token") || strings.Contains(all, "占位符") {
		t.Fatalf("raw request contains generated-token content:\n%s", all)
	}
	user := request.Messages[1].Content
	for _, want := range []string{
		"需要翻译的完整原始页面：\n" + string(source),
		"[[/target][a link]]",
		"`code`",
		".play welcome/hello.go",
		"逐字保留原有行内代码、预格式化代码、present directive、链接及链接 target",
	} {
		if !strings.Contains(user, want) {
			t.Errorf("raw request missing %q:\n%s", want, user)
		}
	}
}

func TestDefaultStaticContextRequestOnlyAddsReadOnlyAppendix(t *testing.T) {
	source := []byte("* Type parameters\n\nSome text about `T`.\n\n  func Index[T comparable](s []T, x T) int\n\n.play generics/index.go\n")
	protected := protectTranslation(source, sum(source), nil)
	before := protected
	blocks := staticContextBlocks(protected)
	if len(blocks) != 1 {
		t.Fatalf("static blocks = %+v, want one", blocks)
	}
	wantCode := "  func Index[T comparable](s []T, x T) int"
	if blocks[0].Code != wantCode || protected.Kinds[slices.Index(protected.Tokens, blocks[0].Token)] != protectedPreformattedStatic {
		t.Fatalf("static block = %+v, want code %q and static kind", blocks[0], wantCode)
	}

	plain := makeTranslationRequestForMode("generics/1", "zh-CN", source, &protected, "- glossary rule", "")
	withContext := makeTranslationRequestForModeOptions("generics/1", "zh-CN", source, &protected, "- glossary rule", "", translationRequestOptions{IncludeStaticContext: true})
	assertProtectedTranslationMetadataEqual(t, before, protected)
	if plain.Messages[0].Content != withContext.Messages[0].Content {
		t.Fatal("static context changed the system prompt")
	}
	if strings.Count(withContext.Messages[1].Content, "只读技术上下文（不属于输出页面）：") != 1 {
		t.Fatalf("static appendix count is not one:\n%s", withContext.Messages[1].Content)
	}
	appendix := staticContextAppendix(protected)
	if appendix == "" || !strings.Contains(appendix, "对应 token："+blocks[0].Token+"\n<static-code>\n"+wantCode+"\n</static-code>") {
		t.Fatalf("static appendix does not contain exact token/code mapping:\n%s", appendix)
	}
	if got := strings.Replace(withContext.Messages[1].Content, "\n\n"+appendix, "", 1); got != plain.Messages[1].Content {
		t.Fatalf("removing deterministic appendix did not reproduce default user message\ngot:\n%s\nwant:\n%s", got, plain.Messages[1].Content)
	}
	for i, token := range protected.Tokens {
		if protected.Kinds[i] != protectedPreformattedStatic && strings.Contains(appendix, protected.Values[i]) {
			t.Fatalf("appendix contains non-static protected value for token %s: %q", token, protected.Values[i])
		}
	}
	if protocol := protectedStructureProtocol(protected); strings.Count(plain.Messages[1].Content, protocol) != 1 || strings.Count(withContext.Messages[1].Content, protocol) != 1 {
		t.Fatal("static context changed or duplicated the protected structure protocol")
	}
}

func TestDefaultStaticContextIncludesMultipleBlocksInSourceOrder(t *testing.T) {
	source := []byte("* Exercise\n\nFirst:\n\n\tz := 1.0\n\nSecond:\n\n\tz := float64(1)\n\n.play example.go\n")
	protected := protectTranslation(source, sum(source), nil)
	blocks := staticContextBlocks(protected)
	if len(blocks) != 2 {
		t.Fatalf("static blocks = %+v, want two", blocks)
	}
	wantCodes := []string{"\tz := 1.0", "\tz := float64(1)"}
	appendix := staticContextAppendix(protected)
	previousEnd := -1
	for i, block := range blocks {
		if block.Code != wantCodes[i] {
			t.Errorf("block %d code = %q, want %q", i+1, block.Code, wantCodes[i])
		}
		entry := "对应 token：" + block.Token + "\n<static-code>\n" + block.Code + "\n</static-code>"
		if strings.Count(appendix, entry) != 1 {
			t.Fatalf("block %d mapping count is not one:\n%s", i+1, appendix)
		}
		at := strings.Index(appendix, entry)
		if at <= previousEnd {
			t.Fatalf("block %d is out of source order:\n%s", i+1, appendix)
		}
		previousEnd = at + len(entry)
	}
}

func TestDefaultStaticContextWithoutStaticBlockLeavesRequestIdentical(t *testing.T) {
	source := []byte("* Next\n\nRead [[/doc/][the documentation]].\n")
	protected := protectTranslation(source, sum(source), nil)
	if blocks := staticContextBlocks(protected); len(blocks) != 0 {
		t.Fatalf("static blocks = %+v, want none", blocks)
	}
	plain := makeTranslationRequestForMode("example/1", "zh-CN", source, &protected, "- glossary rule", "")
	withContext := makeTranslationRequestForModeOptions("example/1", "zh-CN", source, &protected, "- glossary rule", "", translationRequestOptions{IncludeStaticContext: true})
	if !reflect.DeepEqual(plain, withContext) {
		t.Fatalf("no-static request changed\nplain=%+v\nwith context=%+v", plain, withContext)
	}
}

func TestDefaultStaticContextRetryDiffersOnlyByAppendix(t *testing.T) {
	source := []byte("* Type parameters\n\nText about `T`.\n\n  func Index[T comparable](s []T, x T) int\n")
	protected := protectTranslation(source, sum(source), nil)
	feedback := retryFeedbackForMode([]string{"inline code count mismatch: expected 1, actual 2"}, false, false)
	plain := makeTranslationRequestForMode("generics/1", "zh-CN", source, &protected, "- glossary rule", feedback)
	withContext := makeTranslationRequestForModeOptions("generics/1", "zh-CN", source, &protected, "- glossary rule", feedback, translationRequestOptions{IncludeStaticContext: true})
	if plain.Messages[0].Content != withContext.Messages[0].Content {
		t.Fatal("retry static context changed the system prompt")
	}
	retrySuffix := "\n\n上一次完整页面翻译未通过校验：" + feedback
	for name, user := range map[string]string{"default": plain.Messages[1].Content, "static context": withContext.Messages[1].Content} {
		if !strings.HasSuffix(user, retrySuffix) {
			t.Errorf("%s retry feedback is not the last paragraph:\n%s", name, user)
		}
	}
	appendix := staticContextAppendix(protected)
	appendixAt := strings.Index(withContext.Messages[1].Content, "\n\n"+appendix)
	retryAt := strings.LastIndex(withContext.Messages[1].Content, retrySuffix)
	if appendixAt < 0 || retryAt < 0 || appendixAt >= retryAt {
		t.Fatalf("appendix/retry order is wrong:\n%s", withContext.Messages[1].Content)
	}
	if got := strings.Replace(withContext.Messages[1].Content, "\n\n"+appendix, "", 1); got != plain.Messages[1].Content {
		t.Fatal("retry request differs by more than the deterministic static appendix")
	}
}

func TestTranslationRunnerDevStaticContextModeConstraints(t *testing.T) {
	tests := []struct {
		name   string
		runner TranslationRunner
		want   string
	}{
		{"requires dev", TranslationRunner{DevStaticContext: true}, "--dev-static-context requires --dev"},
		{"rejects raw", TranslationRunner{Dev: true, DevStaticContext: true, RawInput: true}, "--dev-static-context cannot be used with --raw-input"},
		{"rejects minimal", TranslationRunner{Dev: true, DevStaticContext: true, MinimalProtect: true}, "--dev-static-context cannot be used with --minimal-protect"},
		{"allows dev default", TranslationRunner{Dev: true, DevStaticContext: true}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.runner.validateModes()
			if tt.want == "" {
				if err != nil {
					t.Fatalf("validateModes() error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tt.want {
				t.Fatalf("validateModes() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestTranslationRequestOptionsZeroValuePreservesExistingModes(t *testing.T) {
	source := []byte("* Source\n\n*Note*: use `Bounds`.\n\n  static := true\n\n.play welcome/hello.go\n")
	full := protectTranslation(source, sum(source), nil)
	minimal := protectPlayDirectives(source, sum(source))
	tests := []struct {
		name      string
		protected *protectedTranslation
	}{
		{"default", &full},
		{"raw", nil},
		{"minimal", &minimal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plain := makeTranslationRequestForMode("example/1", "zh-CN", source, tt.protected, "- glossary rule", "retry feedback")
			zeroOptions := makeTranslationRequestForModeOptions("example/1", "zh-CN", source, tt.protected, "- glossary rule", "retry feedback", translationRequestOptions{})
			if !reflect.DeepEqual(plain, zeroOptions) {
				t.Fatalf("zero request options changed %s mode", tt.name)
			}
		})
	}
}

func assertProtectedTranslationMetadataEqual(t *testing.T, want, got protectedTranslation) {
	t.Helper()
	if want.Text != got.Text || !slices.Equal(want.Tokens, got.Tokens) || !slices.Equal(want.Values, got.Values) || !slices.Equal(want.Kinds, got.Kinds) || !slices.Equal(want.InlineBoundaries, got.InlineBoundaries) || !slices.Equal(want.InlinePairs, got.InlinePairs) || !slices.Equal(want.EmphasisTokens, got.EmphasisTokens) || want.MinimalProtect != got.MinimalProtect {
		t.Fatalf("protected translation changed\nwant=%+v\ngot=%+v", want, got)
	}
}

func TestRawInputRunnerUsesResponseDirectlyAndStillValidatesCandidate(t *testing.T) {
	source := []byte("* Title\n\nSee [[/target][a link]] and `code`.\n\n.play welcome/hello.go\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+
		"example/1\tpending\t0\t"+hash+"\t\t\t\n")
	writeTestGlossary(t, root)
	if err := os.MkdirAll(filepath.Join(root, "_content", "tour", "welcome"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_content", "tour", "welcome", "hello.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var captured TranslationAPIRequest
	client := &TranslationClient{Endpoint: "https://example.invalid", HTTP: mockHTTP(func(r *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": string(source)}, "finish_reason": "stop"}}, "usage": map[string]int{}})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})}
	runner := TranslationRunner{Root: root, Catalog: &Catalog{Pages: []Page{{ID: "example/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}, Client: client, RawInput: true, Now: func() time.Time { return time.Unix(0, 0) }}
	result, err := runner.Run(context.Background(), "example/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ready" || !result.Validation.TokenValid || !result.Validation.PresentValid {
		t.Fatalf("result = %+v", result)
	}
	requestText := captured.Messages[0].Content + "\n" + captured.Messages[1].Content
	if translationTokenRE.MatchString(requestText) || strings.Contains(requestText, "保护 token") {
		t.Fatalf("raw runner generated token request content:\n%s", requestText)
	}
	for _, want := range []string{"[[/target][a link]]", "`code`", ".play welcome/hello.go"} {
		if !strings.Contains(requestText, want) {
			t.Errorf("raw request changed source structure, missing %q:\n%s", want, requestText)
		}
	}
	candidate, err := os.ReadFile(filepath.Join(root, result.CandidatePath))
	if err != nil || string(candidate) != string(source) {
		t.Fatalf("candidate=%q err=%v, want direct response %q", candidate, err, source)
	}
}

func TestMinimalProtectMethods24ProtectsPlayAndEmphasisDelimiters(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("methods/24")
	if err != nil {
		t.Fatal(err)
	}
	protected := protectPlayDirectives(page.Source, page.SourceSHA256)
	wantKinds := []protectedTokenKind{protectedBoldOpen, protectedBoldClose, protectedDirective}
	wantValues := []string{"*", "*", ".play methods/images.go"}
	if !protected.MinimalProtect || !slices.Equal(protected.Kinds, wantKinds) || !slices.Equal(protected.Values, wantValues) {
		t.Fatalf("minimal protection = %+v", protected)
	}
	if strings.Contains(protected.Text, ".play methods/images.go") || !strings.Contains(protected.Text, "Note") || strings.Contains(protected.Text, "*Note*") {
		t.Fatalf("play or emphasis delimiters were not protected as expected:\n%s", protected.Text)
	}
	for _, want := range []string{
		"[[/pkg/image/#Image][Package image]]",
		"`Image`",
		"[[/pkg/image/#Rectangle][`image.Rectangle`]]",
		"\tpackage image",
		"Note",
	} {
		if !strings.Contains(protected.Text, want) {
			t.Errorf("unprotected source structure missing %q:\n%s", want, protected.Text)
		}
	}
	if got := translationTokenRE.FindAllString(protected.Text, -1); !slices.Equal(got, protected.Tokens) {
		t.Fatalf("tokens = %q, want %q", got, protected.Tokens)
	}
	request := makeTranslationRequestForMode(page.ID, "zh-CN", page.Source, &protected, "- glossary rule", "")
	system := request.Messages[0].Content
	for _, want := range []string{"相邻字体结构必须保持 legacy present 可以分别解析的边界", "推荐“*注意*： `Bounds`”", "避免“*注意*：`Bounds`”", "不是中文标点后的普遍空格规则"} {
		if !strings.Contains(system, want) {
			t.Errorf("minimal system prompt missing font boundary rule %q:\n%s", want, system)
		}
	}
	user := request.Messages[1].Content
	if !strings.Contains(user, "分别代表完整 .play directive 或 emphasis delimiter") || !strings.Contains(user, protected.Tokens[0]) || strings.Contains(user, ".play methods/images.go") {
		t.Fatalf("minimal request does not match protected input:\n%s", user)
	}

	candidate, failures := protected.restore(protected.Text)
	if len(failures) != 0 || candidate != string(page.Source) {
		t.Fatalf("restore = %q, failures=%v", candidate, failures)
	}
	if err := ValidateCandidate(root, catalog, page.ID, []byte(candidate)); err != nil {
		t.Fatalf("restored source rejected by shared validator: %v", err)
	}
	for name, response := range map[string]string{
		"missing":   strings.Replace(protected.Text, protected.Tokens[0], "", 1),
		"duplicate": protected.Text + protected.Tokens[0],
		"modified":  strings.Replace(protected.Text, protected.Tokens[0], "⟪GTI18N_deadbeef_000001⟫", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if got, failures := protected.restore(response); got != "" || len(failures) == 0 {
				t.Fatalf("invalid minimal placeholder accepted: got=%q failures=%v", got, failures)
			}
		})
	}
}

func TestMinimalProtectPolicyUsesExistingEmphasisSpans(t *testing.T) {
	source := []byte("* Source\n\n(*Note:* Use `code` and [[/target][link]].)\n\n.play welcome/hello.go\n")
	protected := protectPlayDirectives(source, "12345678")
	wantKinds := []protectedTokenKind{protectedBoldOpen, protectedBoldClose, protectedDirective}
	wantValues := []string{"*", "*", ".play welcome/hello.go"}
	if !protected.MinimalProtect || !slices.Equal(protected.Kinds, wantKinds) || !slices.Equal(protected.Values, wantValues) {
		t.Fatalf("minimal protection = %+v, want kinds %v values %q", protected, wantKinds, wantValues)
	}
	if len(protected.Tokens) != 3 || len(protected.EmphasisTokens) != 2 || len(protected.InlinePairs) != 0 {
		t.Fatalf("minimal token roles = %+v", protected)
	}
	for _, visible := range []string{"Note:", "`code`", "[[/target][link]]"} {
		if !strings.Contains(protected.Text, visible) {
			t.Errorf("minimal input hid %q:\n%s", visible, protected.Text)
		}
	}
	if strings.Contains(protected.Text, ".play welcome/hello.go") || strings.Contains(protected.Text, "*Note:*") {
		t.Fatalf("minimal input left protected structure visible:\n%s", protected.Text)
	}
	if restored, failures := protected.restore(protected.Text); len(failures) != 0 || restored != string(source) {
		t.Fatalf("restore = %q, failures=%v, want %q", restored, failures, source)
	}
}

func TestDirectiveProtectionPoliciesShareCollector(t *testing.T) {
	source := []byte("* Directives\n\n.play examples/one.go\n\n.image images/two.png\n")
	full := protectTranslation(source, "12345678", nil)
	minimal := protectPlayDirectives(source, "12345678")

	if len(full.Tokens) != 2 || len(full.Values) != 2 || full.Values[0] != ".play examples/one.go" || full.Values[1] != ".image images/two.png" {
		t.Fatalf("default directive protection = %+v, want play and image", full)
	}
	if len(minimal.Tokens) != 1 || len(minimal.Values) != 1 || minimal.Values[0] != ".play examples/one.go" {
		t.Fatalf("minimal directive protection = %+v, want only play", minimal)
	}
	if strings.Contains(full.Text, ".play examples/one.go") || strings.Contains(full.Text, ".image images/two.png") {
		t.Fatalf("default protection left a directive visible:\n%s", full.Text)
	}
	if strings.Contains(minimal.Text, ".play examples/one.go") || !strings.Contains(minimal.Text, ".image images/two.png") {
		t.Fatalf("minimal protection did not select only play:\n%s", minimal.Text)
	}
	if restored, failures := full.restore(full.Text); len(failures) != 0 || restored != string(source) {
		t.Fatalf("default restore = %q, failures=%v", restored, failures)
	}
	if restored, failures := minimal.restore(minimal.Text); len(failures) != 0 || restored != string(source) {
		t.Fatalf("minimal restore = %q, failures=%v", restored, failures)
	}
}

func TestMinimalProtectLeavesLinkTargetUnprotected(t *testing.T) {
	source := []byte("* Link\n\n[[/target][label]]\n\n.play examples/one.go\n")
	full := protectTranslation(source, "12345678", nil)
	minimal := protectPlayDirectives(source, "12345678")

	if len(full.Tokens) != 2 || full.Kinds[0] != protectedLinkTarget || full.Values[0] != "/target" || full.Kinds[1] != protectedDirective {
		t.Fatalf("default link/directive protection = %+v", full)
	}
	if len(minimal.Tokens) != 1 || minimal.Kinds[0] != protectedDirective || !strings.Contains(minimal.Text, "[[/target][label]]") {
		t.Fatalf("minimal protection unexpectedly protected link target: %+v", minimal)
	}
}

func TestMinimalFontBoundaryPromptDoesNotAffectOtherModes(t *testing.T) {
	source := []byte("* Source\n\n*Note*: use `Bounds`.\n\n.play welcome/hello.go\n")
	minimal := protectPlayDirectives(source, sum(source))
	full := protectTranslation(source, sum(source), nil)
	requests := map[string]TranslationAPIRequest{
		"minimal": makeTranslationRequestForMode("example/1", "zh-CN", source, &minimal, "- glossary rule", ""),
		"raw":     makeTranslationRequestForMode("example/1", "zh-CN", source, nil, "- glossary rule", ""),
		"default": makeTranslationRequestForMode("example/1", "zh-CN", source, &full, "- glossary rule", ""),
	}
	marker := "相邻字体结构必须保持 legacy present 可以分别解析的边界"
	for mode, request := range requests {
		system := request.Messages[0].Content
		for _, want := range []string{"Section/标题行", "“* ”", "ASCII 空格", "不得写成“*标题”", "应保持为“* 标题”"} {
			if !strings.Contains(system, want) {
				t.Errorf("%s prompt missing shared Section rule %q", mode, want)
			}
		}
		got := strings.Contains(system, marker)
		if got != (mode == "minimal") {
			t.Errorf("%s prompt contains minimal font boundary rule = %t", mode, got)
		}
	}
}

func TestMinimalRetryFeedbackClassifiesFontBeforeGenericFallback(t *testing.T) {
	fontWants := []string{"没有被 present 正确解析", "marker 本身存在并不一定意味着结构有效", "相邻 font constructs 必须保留可独立解析的 whitespace 边界", "*注意*：`Bounds`", "*注意*： `Bounds`", "不得新增、删除、翻译或改变原有程序字体内容", "不得改变强调类型"}
	for _, failure := range []string{"font span count mismatch: expected 2, actual 1", "emphasis sentinel order mismatch at 1"} {
		feedback := retryFeedbackForMode([]string{failure}, false, true)
		for _, want := range fontWants {
			if !strings.Contains(feedback, want) {
				t.Errorf("minimal font feedback for %q missing %q:\n%s", failure, want, feedback)
			}
		}
		if strings.Contains(feedback, "唯一的 .play directive 占位符") {
			t.Errorf("minimal font failure used generic feedback:\n%s", feedback)
		}
	}

	link := retryFeedbackForMode([]string{"link inline code at link index 1 count mismatch"}, false, true)
	if !strings.Contains(link, "不要给原本是普通文本的链接标签添加反引号") || strings.Contains(link, "相邻 font constructs") {
		t.Errorf("link feedback lost priority:\n%s", link)
	}
	inline := retryFeedbackForMode([]string{"inline code count mismatch: expected 3, actual 4"}, false, true)
	for _, want := range []string{"新增、删除或改变了原文的行内代码结构", "不得因中文改写而复制并再次添加反引号", "调整中文句式来避免重复", "span 的数量、内容和结构一致"} {
		if !strings.Contains(inline, want) {
			t.Errorf("minimal inline feedback missing %q:\n%s", want, inline)
		}
	}
	if strings.Contains(inline, "所有 minimal-protect 占位符") || strings.Contains(inline, "链接显示文本") {
		t.Errorf("minimal inline failure used the wrong feedback:\n%s", inline)
	}
	generic := retryFeedbackForMode([]string{"section topology mismatch"}, false, true)
	if !strings.Contains(generic, "所有 minimal-protect 占位符") || strings.Contains(generic, "相邻 font constructs") {
		t.Errorf("unknown minimal failure did not use generic fallback:\n%s", generic)
	}
}

func TestDevAttemptsRetriesMinimalProtectWithFontBoundaryFeedback(t *testing.T) {
	source := []byte("* Source\n\n*Note*: use `Bounds`.\n\n.play welcome/hello.go\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\nexample/1\tpending\t0\t"+hash+"\t\t\t\n")
	writeTestGlossary(t, root)
	if err := os.MkdirAll(filepath.Join(root, "_content", "tour", "welcome"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_content", "tour", "welcome", "hello.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	client := &TranslationClient{Endpoint: "https://example.invalid", HTTP: mockHTTP(func(r *http.Request) (*http.Response, error) {
		call := calls.Add(1)
		var request TranslationAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		all := request.Messages[0].Content + "\n" + request.Messages[1].Content
		for _, want := range []string{"相邻字体结构必须保持 legacy present 可以分别解析的边界", "推荐“*注意*： `Bounds`”"} {
			if !strings.Contains(all, want) {
				t.Errorf("attempt %d missing initial font boundary rule %q:\n%s", call, want, all)
			}
		}
		tokens := uniqueTranslationTokens(request.Messages[1].Content)
		if len(tokens) != 3 {
			t.Fatalf("attempt %d tokens=%q, want emphasis pair and .play", call, tokens)
		}
		content := "* 来源\n\n" + tokens[0] + "注意" + tokens[1] + "： `Bounds`。 *额外*。\n\n" + tokens[2] + "\n"
		if call == 1 {
			if strings.Contains(request.Messages[1].Content, "上一次完整页面翻译未通过校验") {
				t.Fatalf("first request unexpectedly contains retry feedback:\n%s", request.Messages[1].Content)
			}
		} else {
			for _, want := range []string{"没有被 present 正确解析", "marker 本身存在并不一定意味着结构有效", "*注意*：`Bounds`", "*注意*： `Bounds`"} {
				if !strings.Contains(request.Messages[1].Content, want) {
					t.Errorf("second request missing font feedback %q:\n%s", want, request.Messages[1].Content)
				}
			}
			content = "* 来源\n\n" + tokens[0] + "注意" + tokens[1] + "： `Bounds`。\n\n" + tokens[2] + "\n"
		}
		body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": content}, "finish_reason": "stop"}}, "usage": map[string]int{}})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})}
	runner := TranslationRunner{Root: root, Catalog: &Catalog{Pages: []Page{{ID: "example/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}, Client: client, Dev: true, DevAttempts: 2, MinimalProtect: true, Now: func() time.Time { return time.Unix(0, 0) }}
	result, err := runner.Run(context.Background(), "example/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || result.Status != "ready" || result.Attempts != 2 || !result.Validation.PresentValid {
		t.Fatalf("result=%+v calls=%d", result, calls.Load())
	}
	firstValidationPath := filepath.Join(root, "data", "translation-runs", "zh-CN", "example", "1", "sources", hash, "attempt-001", "validation.json")
	firstValidationBytes, err := os.ReadFile(firstValidationPath)
	if err != nil {
		t.Fatal(err)
	}
	var firstValidation TranslationValidation
	if err := json.Unmarshal(firstValidationBytes, &firstValidation); err != nil {
		t.Fatal(err)
	}
	if firstValidation.Passed || firstValidation.PresentValid || !strings.Contains(strings.Join(firstValidation.Failures, "\n"), "font span count mismatch") {
		t.Fatalf("first validation = %+v, want real font span rejection", firstValidation)
	}
}

func TestMinimalProtectNormalizesEmphasisBoundaryBeforeValidation(t *testing.T) {
	source := []byte("* Test\n\n(*Note:* If you are interested.)\n\n.play welcome/hello.go\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\nexample/1\tpending\t0\t"+hash+"\t\t\t\n")
	writeTestGlossary(t, root)
	if err := os.MkdirAll(filepath.Join(root, "_content", "tour", "welcome"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_content", "tour", "welcome", "hello.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	catalog := &Catalog{Pages: []Page{{ID: "example/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}
	protected := protectPlayDirectives(source, hash)
	if !slices.Equal(protected.Kinds, []protectedTokenKind{protectedBoldOpen, protectedBoldClose, protectedDirective}) {
		t.Fatalf("minimal token kinds = %v", protected.Kinds)
	}
	response := "* 测试\n\n（" + protected.Tokens[0] + "注意：" + protected.Tokens[1] + "如果你感兴趣。）\n\n" + protected.Tokens[2] + "\n"
	candidate, failures := protected.restore(response)
	if len(failures) != 0 {
		t.Fatalf("restore failures = %v", failures)
	}
	if !strings.Contains(candidate, "（*注意：* 如果你感兴趣。）") {
		t.Fatalf("restore did not add the required emphasis boundary:\n%s", candidate)
	}
	if err := ValidateCandidate(root, catalog, "example/1", []byte(candidate)); err != nil {
		t.Fatalf("normalized candidate failed shared validator: %v\n%s", err, candidate)
	}
	expectedFonts, err := parsedFontSpans(root, "welcome.article", source)
	if err != nil {
		t.Fatal(err)
	}
	actualFonts, err := parsedFontSpans(root, "welcome.article", []byte(candidate))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(expectedFonts, []fontSpanKind{fontBold}) || !slices.Equal(actualFonts, expectedFonts) {
		t.Fatalf("font spans: source=%v candidate=%v", expectedFonts, actualFonts)
	}
}

func TestDevAttemptsRetriesMinimalProtectWithInlineCodeFeedback(t *testing.T) {
	source := []byte("* Generic\n\nThis declaration means that `s` is a slice of any type `T` that fulfills the built-in constraint `comparable`.\n\n.play welcome/hello.go\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\nexample/1\tpending\t0\t"+hash+"\t\t\t\n")
	writeTestGlossary(t, root)
	if err := os.MkdirAll(filepath.Join(root, "_content", "tour", "welcome"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_content", "tour", "welcome", "hello.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	client := &TranslationClient{Endpoint: "https://example.invalid", HTTP: mockHTTP(func(r *http.Request) (*http.Response, error) {
		call := calls.Add(1)
		var request TranslationAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		user := request.Messages[1].Content
		tokens := uniqueTranslationTokens(user)
		if len(tokens) != 1 {
			t.Fatalf("attempt %d tokens=%q, want only .play", call, tokens)
		}
		content := "* 泛型\n\n这个声明表示，`s` 是元素类型为 `T` 的切片，而 `T` 满足内置约束 `comparable`。\n\n" + tokens[0] + "\n"
		if call == 1 {
			if strings.Contains(user, "上一次完整页面翻译未通过校验") {
				t.Fatalf("first request unexpectedly contains retry feedback:\n%s", user)
			}
		} else {
			for _, want := range []string{"新增、删除或改变了原文的行内代码结构", "不得因中文改写而复制并再次添加反引号", "调整中文句式来避免重复", "span 的数量、内容和结构一致"} {
				if !strings.Contains(user, want) {
					t.Errorf("second request missing inline feedback %q:\n%s", want, user)
				}
			}
			if strings.Contains(user, "所有 minimal-protect 占位符必须") {
				t.Errorf("second request used generic minimal feedback:\n%s", user)
			}
			content = "* 泛型\n\n这个声明表示，`s` 是一个元素类型为 `T` 且满足内置约束 `comparable` 的切片。\n\n" + tokens[0] + "\n"
		}
		body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": content}, "finish_reason": "stop"}}, "usage": map[string]int{}})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})}
	runner := TranslationRunner{Root: root, Catalog: &Catalog{Pages: []Page{{ID: "example/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}, Client: client, Dev: true, DevAttempts: 2, MinimalProtect: true, Now: func() time.Time { return time.Unix(0, 0) }}
	result, err := runner.Run(context.Background(), "example/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || result.Status != "ready" || result.Attempts != 2 || !result.Validation.Passed {
		t.Fatalf("result=%+v calls=%d", result, calls.Load())
	}
	firstValidationPath := filepath.Join(root, "data", "translation-runs", "zh-CN", "example", "1", "sources", hash, "attempt-001", "validation.json")
	firstValidationBytes, err := os.ReadFile(firstValidationPath)
	if err != nil {
		t.Fatal(err)
	}
	var firstValidation TranslationValidation
	if err := json.Unmarshal(firstValidationBytes, &firstValidation); err != nil {
		t.Fatal(err)
	}
	wantFailure := "inline code count mismatch: expected 3, actual 4"
	if firstValidation.Passed || firstValidation.PresentValid || !strings.Contains(strings.Join(firstValidation.Failures, "\n"), wantFailure) {
		t.Fatalf("first validation = %+v, want %q", firstValidation, wantFailure)
	}
}

func TestTranslationRunnerRejectsMutuallyExclusiveRawModes(t *testing.T) {
	runner := TranslationRunner{RawInput: true, MinimalProtect: true}
	if _, err := runner.Run(context.Background(), "example/1", "zh-CN", "test-secret"); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("Run error = %v, want mutually exclusive modes", err)
	}
}

func TestDevAttemptsRetriesMinimalProtectWithLinkInlineCodeFeedback(t *testing.T) {
	source := []byte("* Source\n\n[[/target][Package image]]\n\n.play welcome/hello.go\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\nexample/1\tpending\t0\t"+hash+"\t\t\t\n")
	writeTestGlossary(t, root)
	if err := os.MkdirAll(filepath.Join(root, "_content", "tour", "welcome"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_content", "tour", "welcome", "hello.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	client := &TranslationClient{Endpoint: "https://example.invalid", HTTP: mockHTTP(func(r *http.Request) (*http.Response, error) {
		call := calls.Add(1)
		var request TranslationAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		user := request.Messages[1].Content
		tokens := uniqueTranslationTokens(user)
		if len(tokens) != 1 {
			t.Fatalf("tokens=%q", tokens)
		}
		content := "* 来源\n\n[[/target][`image` 包]]\n\n" + tokens[0] + "\n"
		if call == 2 {
			for _, want := range []string{"链接显示文本中新增了源页面不存在的行内代码格式", "不要给原本是普通文本的链接标签添加反引号", "普通链接显示文字仍可正常翻译"} {
				if !strings.Contains(user, want) {
					t.Errorf("second request missing feedback %q:\n%s", want, user)
				}
			}
			for _, forbidden := range []string{"image/color", "link inline code at link index"} {
				if strings.Contains(user, forbidden) {
					t.Errorf("second request leaked %q:\n%s", forbidden, user)
				}
			}
			content = "* 来源\n\n[[/target][图像包]]\n\n" + tokens[0] + "\n"
		}
		body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": content}, "finish_reason": "stop"}}, "usage": map[string]int{}})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})}
	runner := TranslationRunner{Root: root, Catalog: &Catalog{Pages: []Page{{ID: "example/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}, Client: client, Dev: true, DevAttempts: 2, MinimalProtect: true, Now: func() time.Time { return time.Unix(0, 0) }}
	result, err := runner.Run(context.Background(), "example/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || result.Status != "ready" || result.Attempts != 2 {
		t.Fatalf("result=%+v calls=%d", result, calls.Load())
	}
}

func TestDevAttemptsFailureRecordsLastAttempt(t *testing.T) {
	source := []byte("* Source\n\n[[/target][Package image]]\n\n.play welcome/hello.go\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\nexample/1\tpending\t0\t"+hash+"\t\t\t\n")
	writeTestGlossary(t, root)
	if err := os.MkdirAll(filepath.Join(root, "_content", "tour", "welcome"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_content", "tour", "welcome", "hello.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	client := &TranslationClient{Endpoint: "https://example.invalid", HTTP: mockHTTP(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		var request TranslationAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		token := translationTokenRE.FindAllString(request.Messages[1].Content, -1)[0]
		body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": "* 来源\n\n[[/target][`image` 包]]\n\n" + token + "\n"}, "finish_reason": "stop"}}, "usage": map[string]int{}})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})}
	runner := TranslationRunner{Root: root, Catalog: &Catalog{Pages: []Page{{ID: "example/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}, Client: client, Dev: true, DevAttempts: 2, MinimalProtect: true, Now: func() time.Time { return time.Unix(0, 0) }}
	result, err := runner.Run(context.Background(), "example/1", "zh-CN", "test-secret")
	if err != nil || calls.Load() != 2 || result.Status != "pending" || result.Attempts != 2 {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, calls.Load())
	}
	status, _, err := LoadTranslationResult(root, "example/1", "zh-CN")
	if err != nil || status.Attempts != 2 || status.State != "pending" {
		t.Fatalf("status=%+v err=%v", status, err)
	}
}

func TestValidateDevAttempts(t *testing.T) {
	for _, tt := range []struct {
		dev, wantErr bool
		attempts     int
		want         int
	}{
		{true, false, 0, 1}, {true, false, 1, 1}, {true, false, 2, 2}, {true, false, 3, 3},
		{false, false, 0, 1}, {false, true, 1, 0}, {false, true, 2, 0}, {true, true, 4, 0}, {true, true, -1, 0},
	} {
		got, err := validateDevAttempts(tt.dev, tt.attempts)
		if (err != nil) != tt.wantErr || (!tt.wantErr && got != tt.want) {
			t.Errorf("validateDevAttempts(%t, %d) = %d, %v; want %d, error=%t", tt.dev, tt.attempts, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestTranslationRequestExplainsDynamicPairProtocol(t *testing.T) {
	source := []byte("* Source\n\nThe `swap` is _after_ the value.\n\n.play hidden/example.go\n")
	protected := protectTranslation(source, sum(source), nil)
	request := makeTranslationRequest("example/1", "zh-CN", protected, "- glossary rule", "")
	user := request.Messages[1].Content
	inline := protected.InlinePairs[0]
	emphasis := emphasisPairsForPrompt(protected)[0]
	for _, want := range []string{
		inline.Open, inline.Close, "两者之间当前可见的代码内容必须逐字原样保留在同一 pair 内", "不得翻译、改写、增删、移出 pair", "不得自行添加反引号", "反引号由程序恢复", "必须继续作为行内结构存在", "不得插入、贴入或跨越预格式化 block、directive 或其他块级结构边界",
		emphasis.Open, emphasis.Close, "两者之间的自然语言允许翻译", "译文必须始终留在同一 pair 内", "不得将成对结构 token 拆散或独立移动", "不同的完整 pair 可随各自的语义单元整体换位",
		"其他单 token 仍须原样、唯一地留在所属结构角色中",
	} {
		if !strings.Contains(user, want) {
			t.Errorf("pair protocol missing %q:\n%s", want, user)
		}
	}
	for _, forbidden := range []string{"hidden/example.go", ".play hidden"} {
		if strings.Contains(user, forbidden) {
			t.Errorf("opaque directive leaked %q:\n%s", forbidden, user)
		}
	}
	retry := makeTranslationRequest("example/1", "zh-CN", protected, "- glossary rule", retryFeedback([]string{"inline code count mismatch"}))
	for _, want := range []string{inline.Open, inline.Close, emphasis.Open, emphasis.Close, "不得自行添加反引号"} {
		if !strings.Contains(retry.Messages[1].Content, want) {
			t.Errorf("retry lacks pair protocol %q:\n%s", want, retry.Messages[1].Content)
		}
	}
}

func TestTranslationRequestExplainsStaticBlocksAndDirectivesByRole(t *testing.T) {
	source := []byte("* Source\n\n\tstatic := true\n\nUse `inline`.\n\n.play hidden/example.go\n")
	protected := protectTranslation(source, sum(source), nil)
	request := makeTranslationRequest("example/1", "zh-CN", protected, "- glossary rule", "")
	user := request.Messages[1].Content
	static := protectedTokensOfKind(protected, protectedPreformattedStatic)
	directive := protectedTokensOfKind(protected, protectedDirective)
	if len(static) != 1 || len(directive) != 1 {
		t.Fatalf("static=%v directive=%v", static, directive)
	}
	for _, want := range []string{
		"静态预格式化 block：", static[0], "完整、独立的预格式化代码块", "不得嵌入普通段落、标题、列表或 directive 行", "不得与相邻自然语言合并",
		"present directive：", directive[0], "完整 present directive 行", "不得嵌入普通文本或预格式化代码块", "不得自行手写新的 .play、.image 等 directive",
	} {
		if !strings.Contains(user, want) {
			t.Errorf("role protocol missing %q:\n%s", want, user)
		}
	}
	for _, forbidden := range []string{"hidden/example.go", ".play hidden", "所有 protected token 必须保持原始全局顺序", "不得调整 token 相对顺序"} {
		if strings.Contains(user, forbidden) {
			t.Errorf("role protocol contains forbidden %q:\n%s", forbidden, user)
		}
	}
}

func TestTranslationRequestExplainsPreformattedIdentifierRole(t *testing.T) {
	source := []byte("* Source\n\n\tfmt.Println(p) // read p\n\n")
	protected := protectTranslation(source, sum(source), nil)
	request := makeTranslationRequest("example/1", "zh-CN", protected, "- glossary rule", "")
	user := request.Messages[1].Content
	identifiers := protectedTokensOfKind(protected, protectedPreformattedIdentifier)
	if len(identifiers) != 1 {
		t.Fatalf("preformatted identifier tokens = %v", identifiers)
	}
	for _, want := range []string{
		"教学注释中的 Go 标识符：", identifiers[0], "教学注释中引用的 Go 源码标识符", "词法上独立的 Go 标识符识别", "不得翻译、删除、替换、改变拼写", "相邻中文、英文字母、数字、下划线等字符拼接", "整条注释可按自然中文语序翻译",
	} {
		if !strings.Contains(user, want) {
			t.Errorf("identifier protocol missing %q:\n%s", want, user)
		}
	}
	for _, forbidden := range []string{"所有 protected token 必须保持原始全局顺序", "不得调整 token 相对顺序", "保持原绝对位置"} {
		if strings.Contains(user, forbidden) {
			t.Errorf("identifier protocol contains forbidden %q:\n%s", forbidden, user)
		}
	}
}

func TestTranslationRequestOmitsAbsentBlockRoleProtocols(t *testing.T) {
	source := []byte("* Source\n\nUse `inline`.\n")
	protected := protectTranslation(source, sum(source), nil)
	request := makeTranslationRequest("example/1", "zh-CN", protected, "- glossary rule", "")
	user := request.Messages[1].Content
	for _, forbidden := range []string{"静态预格式化 block：", "教学注释中的 Go 标识符：", "present directive："} {
		if strings.Contains(user, forbidden) {
			t.Errorf("unexpected role protocol %q:\n%s", forbidden, user)
		}
	}
}

func TestRetryFeedbackClassifiesFailuresWithoutEchoingDiagnostics(t *testing.T) {
	tests := []struct {
		name, failure string
		wants         []string
		forbidden     []string
	}{
		{
			name:      "inline code",
			failure:   "basics/6: protected structure validation failed: inline code count mismatch: expected 1, actual 2; check the named directive or protected content near the first difference",
			wants:     []string{"不得在普通文本中自行添加反引号代码", "行内代码"},
			forbidden: []string{"directive", ".play", "expected 1", "actual 2"},
		},
		{
			name:      "directive",
			failure:   `basics/6: protected structure validation failed: present directives mismatch at index 1: expected ".play basics/multiple-results.go", actual ".play basics/multiple-results.go 7,9"; check the named directive or protected content near the first difference`,
			wants:     []string{"不得自行书写 .play、.image 等 present directive", "directive 只能通过已有保护 token 表示"},
			forbidden: []string{"multiple-results.go", "7,9", "expected", "actual"},
		},
		{
			name:      "token",
			failure:   "token 4 occurrence count = 0, want 1",
			wants:     []string{"每个现有 token 必须原样且恰好出现一次", "不得自行重建 token 所代表的代码"},
			forbidden: []string{"occurrence count", "token 4", "want 1"},
		},
		{
			name:      "preformatted static block with diagnostic suffix",
			failure:   "preformatted code block mismatch at index 1: static preformatted block changed; check the named directive or protected content near the first difference",
			wants:     []string{"预格式化代码或教学注释结构", "静态预格式化代码块必须保持独立 block"},
			forbidden: []string{"directive", ".play", "static preformatted block changed"},
		},
		{
			name:      "preformatted teaching comment identifier with diagnostic suffix",
			failure:   "preformatted code block mismatch at index 3: line comment mismatch at index 1: referenced Go identifier count mismatch: expected 1, actual 0; check the named directive or protected content near the first difference",
			wants:     []string{"预格式化代码或教学注释结构", "教学注释中的受保护 Go 标识符必须保持独立"},
			forbidden: []string{"directive", ".play", "referenced Go identifier"},
		},
		{
			name:      "font with diagnostic suffix",
			failure:   "font span count mismatch: expected 5, actual 3; first difference index 4; check the named directive or protected content near the first difference",
			wants:     []string{"强调或字体结构", "不得自行新增、删除或改变强调类型"},
			forbidden: []string{"directive", ".play", "inline code", "expected 5", "actual 3"},
		},
		{
			name:      "present parse",
			failure:   "present parse failed: malformed section",
			wants:     []string{"不是可由 present 解析的完整页面"},
			forbidden: []string{"directive 只能通过已有保护 token 表示"},
		},
		{
			name:      "generic fallback",
			failure:   "rendered page validation failed",
			wants:     []string{"只重新翻译普通文本"},
			forbidden: []string{"directive 只能通过已有保护 token 表示"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feedback := retryFeedback([]string{tt.failure})
			for _, want := range tt.wants {
				if !strings.Contains(feedback, want) {
					t.Errorf("feedback missing %q:\n%s", want, feedback)
				}
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(feedback, forbidden) {
					t.Errorf("feedback leaked %q:\n%s", forbidden, feedback)
				}
			}
		})
	}
}

func TestRetryFeedbackKeepsRawValidationAuditOutOfNextRequest(t *testing.T) {
	source := []byte("* Root\n\nUse `code`.\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+
		"welcome/1\tpending\t0\t"+hash+"\t\t\t\n")
	writeTestGlossary(t, root)
	var secondUser string
	var calls atomic.Int32
	client := &TranslationClient{Endpoint: "https://example.invalid", HTTP: mockHTTP(func(r *http.Request) (*http.Response, error) {
		call := calls.Add(1)
		var request TranslationAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		user := request.Messages[len(request.Messages)-1].Content
		tokens := translationTokenRE.FindAllString(user, -1)
		if len(tokens) < 2 {
			t.Fatalf("inline sentinel tokens = %v", tokens)
		}
		content := "* 根\n\n使用 " + tokens[0] + "code" + tokens[1] + "。\n"
		if call == 1 {
			content = "* 根\n\n`额外` 和 " + tokens[0] + "code" + tokens[1] + "。\n"
		} else {
			secondUser = user
		}
		body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": content}, "finish_reason": "stop"}}, "usage": map[string]int{}})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})}
	runner := TranslationRunner{Root: root, Catalog: &Catalog{Pages: []Page{{ID: "welcome/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}, Client: client, Now: func() time.Time { return time.Unix(0, 0) }}
	result, err := runner.Run(context.Background(), "welcome/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ready" || calls.Load() != 2 {
		t.Fatalf("result=%+v calls=%d", result, calls.Load())
	}
	for _, want := range []string{"不得在普通文本中自行添加反引号代码", "所有已受保护的行内代码只能通过现有 token 表示"} {
		if !strings.Contains(secondUser, want) {
			t.Errorf("second request missing safe inline feedback %q:\n%s", want, secondUser)
		}
	}
	for _, forbidden := range []string{"inline code count mismatch", "expected 1", "actual 2", "check the named directive", ".play"} {
		if strings.Contains(secondUser, forbidden) {
			t.Errorf("second request leaked raw validation diagnostic %q:\n%s", forbidden, secondUser)
		}
	}
	validationPath := filepath.Join(root, "data", "translation-runs", "zh-CN", "welcome", "1", "sources", hash, "attempt-001", "validation.json")
	validation, err := os.ReadFile(validationPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(validation), "inline code count mismatch: expected 1, actual 2") {
		t.Fatalf("raw validation audit was summarized or lost:\n%s", validation)
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

func TestRecoverNetworkBlockedTranslationRestoresFormalWindowWithoutOverwritingAudits(t *testing.T) {
	source := []byte("* Hello\n\nEnglish text.\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+
		"welcome/1\tblocked\t3\t"+hash+"\t\t\tformal attempts exhausted\n")
	writeTestGlossary(t, root)
	sourceDir := filepath.Join(root, "data", "translation-runs", "zh-CN", "welcome", "1", "sources", hash)
	for attempt := 1; attempt <= 3; attempt++ {
		writeNetworkFailureAudit(t, sourceDir, attempt)
	}
	catalog := &Catalog{Pages: []Page{{ID: "welcome/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}
	recovered, err := RecoverNetworkBlockedTranslation(root, catalog, "welcome/1", "zh-CN", func() time.Time { return time.Unix(0, 0) })
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status != "pending" || recovered.Attempts != 3 || !strings.HasSuffix(recovered.RecoveryPath, "network-recovery-001.json") {
		t.Fatalf("recovery=%+v", recovered)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := os.Stat(filepath.Join(sourceDir, fmt.Sprintf("attempt-%03d", attempt), "validation.json")); err != nil {
			t.Fatalf("old attempt %d missing after recovery: %v", attempt, err)
		}
	}
	if _, err := os.Stat(recovered.RecoveryPath); err != nil {
		t.Fatalf("recovery audit missing: %v", err)
	}

	var calls atomic.Int32
	runner := TranslationRunner{Root: root, Catalog: catalog, Client: &TranslationClient{Endpoint: "https://example.invalid", HTTP: mockHTTP(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("temporary network outage")
	})}, Now: func() time.Time { return time.Unix(1, 0) }}
	result, err := runner.Run(context.Background(), "welcome/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "blocked" || result.Attempts != 6 || calls.Load() != 3 {
		t.Fatalf("result=%+v calls=%d, want blocked at attempt 6 after three recovered formal attempts", result, calls.Load())
	}
	for attempt := 4; attempt <= 6; attempt++ {
		if _, err := os.Stat(filepath.Join(sourceDir, fmt.Sprintf("attempt-%03d", attempt), "validation.json")); err != nil {
			t.Fatalf("recovered formal attempt %d missing: %v", attempt, err)
		}
	}
}

func TestRecoverNetworkBlockedTranslationFailsClosed(t *testing.T) {
	source := []byte("* Hello\n\nEnglish text.\n")
	hash := sum(source)
	catalog := &Catalog{Pages: []Page{{ID: "welcome/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}
	for name, mutate := range map[string]func(t *testing.T, sourceDir string){
		"validator failure": func(t *testing.T, sourceDir string) {
			writeNetworkFailureAudit(t, sourceDir, 1)
			writeNetworkFailureAudit(t, sourceDir, 2)
			writeNetworkFailureAudit(t, sourceDir, 3)
			path := filepath.Join(sourceDir, "attempt-003", "validation.json")
			validation := TranslationValidation{Attempt: 3, APISuccess: true, TokenValid: true, Failures: []string{"directive mismatch"}}
			if err := writeTranslationJSON(path, validation); err != nil {
				t.Fatal(err)
			}
		},
		"valid API response": func(t *testing.T, sourceDir string) {
			writeNetworkFailureAudit(t, sourceDir, 1)
			writeNetworkFailureAudit(t, sourceDir, 2)
			writeNetworkFailureAudit(t, sourceDir, 3)
			response := TranslationCallResult{StatusCode: 200, RequestID: "request-id", FinishReason: "stop", Content: "* model response", Raw: json.RawMessage(`{"choices":[]}`)}
			if err := writeTranslationJSON(filepath.Join(sourceDir, "attempt-003", "response.json"), response); err != nil {
				t.Fatal(err)
			}
		},
		"incomplete audit": func(t *testing.T, sourceDir string) {
			writeNetworkFailureAudit(t, sourceDir, 1)
			writeNetworkFailureAudit(t, sourceDir, 2)
			if err := os.MkdirAll(filepath.Join(sourceDir, "attempt-003"), 0755); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+
				"welcome/1\tblocked\t3\t"+hash+"\t\t\tformal attempts exhausted\n")
			sourceDir := filepath.Join(root, "data", "translation-runs", "zh-CN", "welcome", "1", "sources", hash)
			mutate(t, sourceDir)
			if _, err := RecoverNetworkBlockedTranslation(root, catalog, "welcome/1", "zh-CN", nil); err == nil {
				t.Fatal("network recovery unexpectedly succeeded")
			}
		})
	}

	t.Run("non-blocked", func(t *testing.T) {
		root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+
			"welcome/1\tpending\t0\t"+hash+"\t\t\t\n")
		if _, err := RecoverNetworkBlockedTranslation(root, catalog, "welcome/1", "zh-CN", nil); err == nil {
			t.Fatal("non-blocked page recovered")
		}
	})
}

func TestFormalTranslationResumesOnlyRemainingInitialWindowAttempts(t *testing.T) {
	source := []byte("* Hello\n\nEnglish text.\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+
		"welcome/1\tpending\t1\t"+hash+"\t\t\tinterrupted after attempt 1\n")
	writeTestGlossary(t, root)
	sourceDir := filepath.Join(root, "data", "translation-runs", "zh-CN", "welcome", "1", "sources", hash)
	writeNetworkFailureAudit(t, sourceDir, 1)
	catalog := &Catalog{Pages: []Page{{ID: "welcome/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}
	var calls atomic.Int32
	runner := newNetworkFailureRunner(root, catalog, &calls)
	result, err := runner.Run(context.Background(), "welcome/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "blocked" || result.Attempts != 3 || calls.Load() != 2 {
		t.Fatalf("result=%+v calls=%d, want only attempts 2 and 3", result, calls.Load())
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "attempt-004")); !os.IsNotExist(err) {
		t.Fatalf("attempt-004 exists after initial window resume: %v", err)
	}
	if _, err := runner.Run(context.Background(), "welcome/1", "zh-CN", "test-secret"); err == nil || !strings.Contains(err.Error(), "is blocked") {
		t.Fatalf("blocked initial window run error = %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("blocked run made additional API calls: %d", calls.Load())
	}
}

func TestFormalTranslationResumesOnlyRemainingRecoveredWindowAttempts(t *testing.T) {
	source := []byte("* Hello\n\nEnglish text.\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+
		"welcome/1\tblocked\t3\t"+hash+"\t\t\tformal attempts exhausted\n")
	writeTestGlossary(t, root)
	sourceDir := filepath.Join(root, "data", "translation-runs", "zh-CN", "welcome", "1", "sources", hash)
	for attempt := 1; attempt <= 3; attempt++ {
		writeNetworkFailureAudit(t, sourceDir, attempt)
	}
	catalog := &Catalog{Pages: []Page{{ID: "welcome/1", Article: "welcome.article", Source: source, SourceSHA256: hash}}}
	if _, err := RecoverNetworkBlockedTranslation(root, catalog, "welcome/1", "zh-CN", func() time.Time { return time.Unix(0, 0) }); err != nil {
		t.Fatal(err)
	}
	writeNetworkFailureAudit(t, sourceDir, 4) // Simulate interruption after the first recovered-window attempt.
	var calls atomic.Int32
	runner := newNetworkFailureRunner(root, catalog, &calls)
	result, err := runner.Run(context.Background(), "welcome/1", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "blocked" || result.Attempts != 6 || calls.Load() != 2 {
		t.Fatalf("result=%+v calls=%d, want only attempts 5 and 6", result, calls.Load())
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "attempt-007")); !os.IsNotExist(err) {
		t.Fatalf("attempt-007 exists after recovered window resume: %v", err)
	}
}

func newNetworkFailureRunner(root string, catalog *Catalog, calls *atomic.Int32) TranslationRunner {
	return TranslationRunner{Root: root, Catalog: catalog, Client: &TranslationClient{Endpoint: "https://example.invalid", HTTP: mockHTTP(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("temporary network outage")
	})}, Now: func() time.Time { return time.Unix(1, 0) }}
}

func writeNetworkFailureAudit(t *testing.T, sourceDir string, attempt int) {
	t.Helper()
	dir := filepath.Join(sourceDir, fmt.Sprintf("attempt-%03d", attempt))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeTranslationJSON(filepath.Join(dir, "request.json"), savedTranslationRequest{}); err != nil {
		t.Fatal(err)
	}
	if err := writeTranslationJSON(filepath.Join(dir, "response.json"), TranslationCallResult{}); err != nil {
		t.Fatal(err)
	}
	validation := TranslationValidation{Attempt: attempt, Failures: []string{"network: dial udp: socket: operation not permitted"}}
	if err := writeTranslationJSON(filepath.Join(dir, "validation.json"), validation); err != nil {
		t.Fatal(err)
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
			const marker = "需要翻译的完整受保护页面：\n"
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

func TestDevTranslationAllowsAnotherStandalonePage(t *testing.T) {
	source := []byte("* Go local\n\nEnglish text.\n")
	hash := sum(source)
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"+
		"welcome/2\tpending\t0\t"+hash+"\t\t\t\n")
	writeTestGlossary(t, root)

	var calls atomic.Int32
	client := &TranslationClient{Endpoint: "https://example.invalid", HTTP: mockHTTP(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		body := `{"choices":[{"message":{"role":"assistant","content":""},"finish_reason":"length"}],"usage":{}}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	runner := TranslationRunner{
		Root:    root,
		Catalog: &Catalog{Pages: []Page{{ID: "welcome/2", Article: "welcome.article", Source: source, SourceSHA256: hash}}},
		Client:  client,
		Dev:     true,
		Now:     func() time.Time { return time.Unix(0, 0) },
	}

	result, err := runner.Run(context.Background(), "welcome/2", "zh-CN", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.PageID != "welcome/2" || result.Status != "pending" || result.Attempts != 1 || calls.Load() != 1 {
		t.Fatalf("result=%+v calls=%d", result, calls.Load())
	}
}

func TestTranslationPreflightDoesNotCallHTTP(t *testing.T) {
	var calls atomic.Int32
	client := &TranslationClient{HTTP: mockHTTP(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, nil
	})}
	runner := TranslationRunner{Root: t.TempDir(), Catalog: &Catalog{
		Conditional: []ConditionalPage{{Article: "welcome.article", Condition: "appengine", ConditionalIndex: 1}},
	}, Client: client}
	for _, pageID := range []string{"missing/1", "welcome/appengine/1"} {
		if _, err := runner.Run(context.Background(), pageID, "zh-CN", "test-secret"); err == nil || !strings.Contains(err.Error(), "unknown page_id") {
			t.Errorf("Run(%q) error = %v, want unknown page_id", pageID, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("HTTP calls=%d", calls.Load())
	}
}
