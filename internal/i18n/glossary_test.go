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
		"A Tour of Go":    "Go 语言之旅",
		"previous":        "上一页",
		"next":            "下一页",
		"Run":             "运行",
		"Format":          "格式化",
		"slides":          "页面",
		"Go Playground":   "Go 语言演练场",
		"constraint":      "约束",
		"type switch":     "类型选择",
		"type switches":   "类型选择",
		"type assertion":  "类型断言",
		"type assertions": "类型断言",
		"interface value": "接口值",
		"interface type":  "接口类型",
		"concrete type":   "具体类型",
		"type parameter":  "类型参数",
		"type parameters": "类型参数",
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
		"exported name":        "导出名",
		"unexported name":      "未导出的名称",
		"standard library":     "标准库",
		"iteration":            "迭代",
		"loop condition":       "循环条件",
	}
	for key, value := range wantPreferred {
		if got := glossary.Preferred[key]; got != value {
			t.Errorf("preferred[%q] = %q, want %q", key, got, value)
		}
		if _, ok := glossary.Mandatory[key]; ok {
			t.Errorf("preferred term %q is also mandatory", key)
		}
	}
	rules := glossary.PromptRules("welcome/1")
	for _, text := range []string{
		"A Tour of Go => Go 语言之旅（强制；不得保留对应的英文显示文本）",
		"Go Playground => Go 语言演练场（强制；不得保留对应的英文显示文本）",
		"普通正文中的 tour => 教程（上下文指导；应结合完整页面自然翻译）",
		"普通正文中的 the tour => 本教程（上下文指导；应结合完整页面自然翻译）",
		"普通正文中的 sandbox => 沙箱（上下文指导；应结合完整页面自然翻译）",
		"普通正文中的 deterministic output => 确定性输出（上下文指导；应结合完整页面自然翻译）",
		"禁止使用的 zh-CN 译法：幻灯片",
		"welcome/1 必须将 tour 的含义保留为“之旅”；不得简化或改变该含义",
	} {
		if !strings.Contains(rules, text) {
			t.Errorf("prompt rules missing %q", text)
		}
	}
	for _, old := range []string{"(mandatory;", "ordinary prose ", "forbidden zh-CN translation:", "do not simplify or change"} {
		if strings.Contains(rules, old) {
			t.Errorf("prompt rules retain English control text %q", old)
		}
	}
	ordered := []string{
		"A Tour of Go =>", "Format =>", "Go Playground =>", "Run =>", "next =>", "previous =>", "slides =>",
		"普通正文中的 deterministic output =>", "普通正文中的 exported name =>", "普通正文中的 import path =>",
		"普通正文中的 import statement =>", "普通正文中的 iteration =>", "普通正文中的 loop condition =>",
		"普通正文中的 package =>", "普通正文中的 package name =>", "普通正文中的 sandbox =>",
		"普通正文中的 standard library =>", "普通正文中的 the tour =>", "普通正文中的 tour =>",
		"普通正文中的 unexported name =>", "禁止使用的 zh-CN 译法：幻灯片",
	}
	last := -1
	for _, text := range ordered {
		index := strings.Index(rules, text)
		if index <= last {
			t.Errorf("prompt rule order is wrong at %q:\n%s", text, rules)
		}
		last = index
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
