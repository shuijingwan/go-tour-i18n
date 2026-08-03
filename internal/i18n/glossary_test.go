package i18n

import (
	"strings"
	"testing"
)

func TestZhCNMandatoryGlossary(t *testing.T) {
	glossary, err := LoadGlossary(repoRoot(t), "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"A Tour of Go":  "Go 语言之旅",
		"previous":      "上一页",
		"next":          "下一页",
		"Run":           "运行",
		"Format":        "格式化",
		"slides":        "页面",
		"Go Playground": "Go 语言演练场",
	}
	for key, value := range want {
		if got := glossary.Mandatory[key]; got != value {
			t.Errorf("mandatory[%q] = %q, want %q", key, got, value)
		}
	}
	if len(glossary.Forbidden) != 3 {
		t.Fatalf("forbidden = %v", glossary.Forbidden)
	}
	for _, value := range []string{"幻灯片", "本之旅", "欢迎使用 Go 编程语言之旅"} {
		if !containsString(glossary.Forbidden, value) {
			t.Errorf("forbidden missing %q: %v", value, glossary.Forbidden)
		}
	}
	wantPreferred := map[string]string{
		"tour":                 "教程",
		"the tour":             "本教程",
		"sandbox":              "沙箱",
		"deterministic output": "确定性输出",
		"package":              "包",
		"import path":          "导入路径",
		"package name":         "包名",
		"import statement":     "导入语句",
	}
	for key, value := range wantPreferred {
		if got := glossary.Preferred[key]; got != value {
			t.Errorf("preferred[%q] = %q, want %q", key, got, value)
		}
	}
	rules := glossary.PromptRules("welcome/1")
	for _, text := range []string{"mandatory", "do not retain the English display text", "A Tour of Go => Go 语言之旅", "Go Playground => Go 语言演练场", "ordinary prose tour => 教程", "ordinary prose the tour => 本教程", "ordinary prose sandbox => 沙箱", "ordinary prose deterministic output => 确定性输出", "do not simplify or change"} {
		if !strings.Contains(rules, text) {
			t.Errorf("prompt rules missing %q", text)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
