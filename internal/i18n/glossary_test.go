package i18n

import (
	"os"
	"path/filepath"
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
		"slide":           "页面",
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
	wantKeep := []string{"Go", "gofmt", "PageUp", "PageDown", "Shift", "Enter", "Ctrl", "goroutine", "goroutines", "Goroutines"}
	if len(glossary.Keep) != len(wantKeep) {
		t.Fatalf("keep = %v, want %v", glossary.Keep, wantKeep)
	}
	for i, value := range wantKeep {
		if glossary.Keep[i] != value {
			t.Errorf("keep[%d] = %q, want %q", i, glossary.Keep[i], value)
		}
	}
	wantPreferred := map[string]string{
		"Go programming language": "Go 编程语言",
		"tour":                    "教程",
		"the tour":                "本教程",
		"sandbox":                 "沙箱",
		"deterministic output":    "确定性输出",
		"package":                 "包",
		"import path":             "导入路径",
		"package name":            "包名",
		"import statement":        "导入语句",
		"exported name":           "导出名",
		"unexported name":         "未导出的名称",
		"standard library":        "标准库",
		"iteration":               "迭代",
		"loop condition":          "循环条件",
		"module":                  "模块",
		"exercise":                "练习",
		"syntax highlighting":     "语法高亮",
		"map":                     "映射",
		"maps":                    "映射",
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
		"普通正文中的 map => 映射（上下文指导；应结合完整页面自然翻译）",
		"普通正文中的 maps => 映射（上下文指导；应结合完整页面自然翻译）",
		"Go（保持原样；不得翻译）",
		"goroutine（保持原样；不得翻译）",
		"gofmt（保持原样；不得翻译）",
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
		"A Tour of Go =>", "Format =>", "Go Playground =>", "Run =>", "next =>", "previous =>", "slide =>", "slides =>",
		"普通正文中的 Go programming language =>", "普通正文中的 deterministic output =>", "普通正文中的 exercise =>", "普通正文中的 exported name =>", "普通正文中的 import path =>",
		"普通正文中的 import statement =>", "普通正文中的 iteration =>", "普通正文中的 loop condition =>",
		"普通正文中的 map =>", "普通正文中的 maps =>", "普通正文中的 module =>", "普通正文中的 package =>", "普通正文中的 package name =>", "普通正文中的 sandbox =>",
		"普通正文中的 standard library =>", "普通正文中的 syntax highlighting =>", "普通正文中的 the tour =>", "普通正文中的 tour =>",
		"普通正文中的 unexported name =>", "Ctrl（保持原样；不得翻译）", "Enter（保持原样；不得翻译）",
		"Go（保持原样；不得翻译）", "Goroutines（保持原样；不得翻译）", "PageDown（保持原样；不得翻译）", "PageUp（保持原样；不得翻译）",
		"Shift（保持原样；不得翻译）", "gofmt（保持原样；不得翻译）",
		"goroutine（保持原样；不得翻译）", "goroutines（保持原样；不得翻译）", "禁止使用的 zh-CN 译法：幻灯片",
	}
	last := -1
	for _, text := range ordered {
		index := strings.Index(rules, text)
		if index <= last {
			t.Errorf("prompt rule order is wrong at %q:\n%s", text, rules)
		}
		last = index
	}
	glossaryData, err := os.ReadFile(filepath.Join(repoRoot(t), "locales", "zh-CN", "glossary.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(glossaryData), "\nterms:") {
		t.Fatal("formal zh-CN glossary still contains legacy terms section")
	}
}

func TestDeDEGlossary(t *testing.T) {
	glossary, err := LoadGlossary(repoRoot(t), "de-DE")
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"A Tour of Go":    "Eine Tour durch Go",
		"Go Playground":   "Go Playground",
		"Run":             "Ausführen",
		"Format":          "Formatieren",
		"Reset":           "Zurücksetzen",
		"slide":           "Seite",
		"slides":          "Seiten",
		"constraint":      "Typbeschränkung",
		"type switch":     "Typ-Switch",
		"type assertion":  "Typzusicherung",
		"interface value": "Interface-Wert",
		"type parameter":  "Typparameter",
	} {
		if got := glossary.Mandatory[key]; got != want {
			t.Errorf("mandatory[%q] = %q, want %q", key, got, want)
		}
	}
	for key, want := range map[string]string{
		"Go programming language": "Programmiersprache Go",
		"channel":                 "Kanal",
		"interface":               "Interface",
		"struct":                  "Struct",
		"array":                   "Array",
		"concurrency":             "Nebenläufigkeit",
		"generics":                "Generics",
		"generic programming":     "generische Programmierung",
		"package":                 "Paket",
		"standard library":        "Standardbibliothek",
		"map":                     "Map",
		"slice":                   "Slice",
	} {
		if got := glossary.Preferred[key]; got != want {
			t.Errorf("preferred[%q] = %q, want %q", key, got, want)
		}
	}
	for _, want := range []string{"Folie", "Folien", "Go-Spielplatz", "Typenschalter", "Typbehauptung", "Go-Routine"} {
		if !containsString(glossary.Forbidden, want) {
			t.Errorf("forbidden missing %q: %v", want, glossary.Forbidden)
		}
	}
	for _, want := range []string{"Go", "gofmt", "goroutine", "goroutines", "Goroutines"} {
		if !containsString(glossary.Keep, want) {
			t.Errorf("keep missing %q: %v", want, glossary.Keep)
		}
	}
}

func TestFrFRGlossary(t *testing.T) {
	glossary, err := LoadGlossary(repoRoot(t), "fr-FR")
	if err != nil {
		t.Fatal(err)
	}
	if glossary.Locale != "fr-FR" {
		t.Fatalf("glossary locale = %q, want fr-FR", glossary.Locale)
	}
	for key, want := range map[string]string{
		"A Tour of Go":    "Un tour de Go",
		"Go Playground":   "Go Playground",
		"Run":             "Exécuter",
		"Format":          "Formater",
		"Reset":           "Réinitialiser",
		"slide":           "page",
		"slides":          "pages",
		"constraint":      "contrainte",
		"type switch":     "switch de type",
		"type assertion":  "assertion de type",
		"interface value": "valeur d’interface",
		"type parameter":  "paramètre de type",
	} {
		if got := glossary.Mandatory[key]; got != want {
			t.Errorf("mandatory[%q] = %q, want %q", key, got, want)
		}
	}
	for key, want := range map[string]string{
		"Go programming language": "langage de programmation Go",
		"channel":                 "canal",
		"interface":               "interface",
		"struct":                  "structure",
		"array":                   "tableau",
		"concurrency":             "concurrence",
		"generics":                "génériques",
		"generic programming":     "programmation générique",
		"package":                 "paquet",
		"standard library":        "bibliothèque standard",
		"map":                     "map",
		"slice":                   "slice",
	} {
		if got := glossary.Preferred[key]; got != want {
			t.Errorf("preferred[%q] = %q, want %q", key, got, want)
		}
	}
	for _, want := range []string{"diapositive", "diapositives", "terrain de jeu de Go", "affirmation de type", "routine Go"} {
		if !containsString(glossary.Forbidden, want) {
			t.Errorf("forbidden missing %q: %v", want, glossary.Forbidden)
		}
	}
	for _, want := range []string{"Go", "gofmt", "goroutine", "goroutines", "Goroutines"} {
		if !containsString(glossary.Keep, want) {
			t.Errorf("keep missing %q: %v", want, glossary.Keep)
		}
	}
	rules := glossary.PromptRules("welcome/1")
	if !strings.Contains(rules, "禁止使用的 fr-FR 译法：diapositive") {
		t.Errorf("fr-FR prompt rules do not identify the locale:\n%s", rules)
	}
	for _, unwanted := range []string{"禁止使用的 zh-CN 译法", "必须将 tour 的含义保留为“之旅”"} {
		if strings.Contains(rules, unwanted) {
			t.Errorf("fr-FR prompt rules contain zh-CN-only instruction %q:\n%s", unwanted, rules)
		}
	}
}

func TestKoKRGlossary(t *testing.T) {
	glossary, err := LoadGlossary(repoRoot(t), "ko-KR")
	if err != nil {
		t.Fatal(err)
	}
	if glossary.Locale != "ko-KR" {
		t.Fatalf("glossary locale = %q, want ko-KR", glossary.Locale)
	}
	for key, want := range map[string]string{
		"A Tour of Go":    "Go 언어 투어",
		"Go Playground":   "Go 플레이그라운드",
		"Run":             "실행",
		"Format":          "포맷",
		"Reset":           "초기화",
		"slide":           "페이지",
		"constraint":      "타입 제약",
		"type switch":     "타입 스위치",
		"type assertion":  "타입 단언",
		"interface value": "인터페이스 값",
		"type parameter":  "타입 매개변수",
	} {
		if got := glossary.Mandatory[key]; got != want {
			t.Errorf("mandatory[%q] = %q, want %q", key, got, want)
		}
	}
	for key, want := range map[string]string{
		"Go programming language": "Go 프로그래밍 언어",
		"channel":                 "채널",
		"interface":               "인터페이스",
		"struct":                  "구조체",
		"array":                   "배열",
		"concurrency":             "동시성",
		"generics":                "제네릭",
		"package":                 "패키지",
		"standard library":        "표준 라이브러리",
		"map":                     "맵",
		"slice":                   "슬라이스",
		"pointer":                 "포인터",
		"receiver":                "리시버",
		"zero value":              "제로 값",
	} {
		if got := glossary.Preferred[key]; got != want {
			t.Errorf("preferred[%q] = %q, want %q", key, got, want)
		}
	}
	for _, want := range []string{"Go 놀이터", "고루틴", "Go 루틴", "타입 주장", "슬라이드", "Golang", "골랭"} {
		if !containsString(glossary.Forbidden, want) {
			t.Errorf("forbidden missing %q: %v", want, glossary.Forbidden)
		}
	}
	for _, want := range []string{"Go", "gofmt", "GOPATH", "URL", "API", "goroutine", "goroutines", "Goroutines"} {
		if !containsString(glossary.Keep, want) {
			t.Errorf("keep missing %q: %v", want, glossary.Keep)
		}
	}
	rules := glossary.PromptRules("welcome/1")
	for _, want := range []string{
		"A Tour of Go => Go 언어 투어（强制；不得保留对应的英文显示文本）",
		"Go Playground => Go 플레이그라운드（强制；不得保留对应的英文显示文本）",
		"普通正文中的 channel => 채널（上下文指导；应结合完整页面自然翻译）",
		"goroutine（保持原样；不得翻译）",
		"禁止使用的 ko-KR 译法：고루틴",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("prompt rules missing %q", want)
		}
	}
}

func TestEsESGlossary(t *testing.T) {
	glossary, err := LoadGlossary(repoRoot(t), "es-ES")
	if err != nil {
		t.Fatal(err)
	}
	if glossary.Locale != "es-ES" {
		t.Fatalf("glossary locale = %q, want es-ES", glossary.Locale)
	}
	for key, want := range map[string]string{
		"A Tour of Go":    "Un tour por Go",
		"Go Playground":   "Go Playground",
		"Run":             "Ejecutar",
		"Format":          "Formatear",
		"Reset":           "Restablecer",
		"slide":           "página",
		"constraint":      "restricción",
		"type switch":     "switch de tipo",
		"type assertion":  "aserción de tipo",
		"interface value": "valor de interfaz",
		"type parameter":  "parámetro de tipo",
	} {
		if got := glossary.Mandatory[key]; got != want {
			t.Errorf("mandatory[%q] = %q, want %q", key, got, want)
		}
	}
	for key, want := range map[string]string{
		"Go programming language": "lenguaje de programación Go",
		"channel":                 "canal",
		"interface":               "interfaz",
		"struct":                  "estructura",
		"array":                   "array",
		"concurrency":             "concurrencia",
		"generics":                "genéricos",
		"generic programming":     "programación genérica",
		"standard library":        "biblioteca estándar",
		"map":                     "mapa",
		"slice":                   "slice",
		"zero value":              "valor cero",
	} {
		if got := glossary.Preferred[key]; got != want {
			t.Errorf("preferred[%q] = %q, want %q", key, got, want)
		}
	}
	for _, want := range []string{"Golang", "patio de juegos de Go", "rutina Go", "corrutina", "afirmación de tipo", "diapositiva"} {
		if !containsString(glossary.Forbidden, want) {
			t.Errorf("forbidden missing %q: %v", want, glossary.Forbidden)
		}
	}
	for _, want := range []string{"Go", "go vet", "gofmt", "GOPATH", "URL", "API", "ASCII", "CPU", "UTC", "goroutine", "goroutines", "Goroutines"} {
		if !containsString(glossary.Keep, want) {
			t.Errorf("keep missing %q: %v", want, glossary.Keep)
		}
	}
	rules := glossary.PromptRules("welcome/1")
	for _, want := range []string{
		"A Tour of Go => Un tour por Go（强制；不得保留对应的英文显示文本）",
		"Go Playground => Go Playground（强制；不得保留对应的英文显示文本）",
		"普通正文中的 channel => canal（上下文指导；应结合完整页面自然翻译）",
		"goroutine（保持原样；不得翻译）",
		"禁止使用的 es-ES 译法：Golang",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("prompt rules missing %q", want)
		}
	}
}

func TestGlossaryKeepValidationAndLegacyTerms(t *testing.T) {
	tests := []struct {
		name, body, want string
	}{
		{
			name: "duplicate keep",
			body: "mandatory:\n  slides: 页面\nkeep:\n  - Go\n  - Go\n",
			want: "duplicate value \"Go\"",
		},
		{
			name: "empty keep",
			body: "mandatory:\n  slides: 页面\nkeep:\n  -\n",
			want: "value is empty",
		},
		{
			name: "unknown section",
			body: "mandatory:\n  slides: 页面\nunknown:\n  value: ignored\n",
			want: "unknown glossary section \"unknown\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeGlossaryFixture(t, tt.body)
			_, err := LoadGlossary(root, "zh-CN")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadGlossary error = %v, want %q", err, tt.want)
			}
		})
	}

	root := writeGlossaryFixture(t, "mandatory:\n  slides: 页面\nterms:\n  exercise: 练习\nkeep:\n  - PageDown\n  - Go\n")
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if len(glossary.Keep) != 2 || glossary.Keep[0] != "PageDown" || glossary.Keep[1] != "Go" {
		t.Fatalf("keep order = %v, want [PageDown Go]", glossary.Keep)
	}
	if strings.Contains(glossary.PromptRules("example/1"), "exercise") {
		t.Fatal("legacy terms unexpectedly became active")
	}
}

func writeGlossaryFixture(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "locales", "zh-CN")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "glossary.yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
