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

func TestInlineCodePairOrderAndContentAreStrict(t *testing.T) {
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
			p := protectTranslation([]byte(tt.source), "12345678", nil)
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
			p := protectTranslation([]byte(keep), "12345678", nil)
			if len(p.Values) != 1 || p.Values[0] != keep || p.Kinds[0] != protectedGlossaryOrKeep {
				t.Fatalf("protection = %+v, want %q as glossary/keep", p, keep)
			}
		})
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
		"恰好输出一次",
		"不得复制",
		"不得复用",
		"完整保持所属语义单元和结构关系",
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

func TestTranslationRequestExplainsDynamicPairProtocol(t *testing.T) {
	source := []byte("* Source\n\nThe `swap` is _after_ the value.\n\n.play hidden/example.go\n")
	protected := protectTranslation(source, sum(source), nil)
	request := makeTranslationRequest("example/1", "zh-CN", protected, "- glossary rule", "")
	user := request.Messages[1].Content
	inline := protected.InlinePairs[0]
	emphasis := emphasisPairsForPrompt(protected)[0]
	for _, want := range []string{
		inline.Open, inline.Close, "两者之间当前可见的代码内容必须逐字原样保留在同一 pair 内", "不得翻译、改写、增删、移出 pair", "不得自行添加反引号", "反引号由程序恢复",
		emphasis.Open, emphasis.Close, "两者之间的自然语言允许翻译", "译文必须始终留在同一 pair 内", "不得将成对结构 token 拆散、独立移动",
		"directive、链接 target、keep-word 等单 token 结构仍只能原样、唯一保留",
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
			failure:   `basics/6: protected structure validation failed: present directives mismatch at index 1: expected ".play basics/multiple-results.go", actual ".play basics/multiple-results.go 7,9"`,
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
			name:      "font",
			failure:   "methods/24: protected structure validation failed: font span count mismatch: expected 10, actual 9",
			wants:     []string{"强调或字体结构", "不得自行新增、删除或改变强调类型"},
			forbidden: []string{"directive", ".play", "inline code", "expected 10"},
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
