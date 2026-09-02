package i18n

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func writeGoExampleValidationGlossary(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "locales", "zh-CN")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	body := "mandatory:\n  Go: Go\npreferred:\n  channel: 通道\nforbidden:\n  - 幻灯片\nkeep:\n  - Go\n  - gofmt\n  - GOPATH\n  - URL\n  - API\n  - goroutine\n  - goroutines\n  - Goroutines\n"
	if err := os.WriteFile(filepath.Join(dir, "glossary.yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func goExampleValidationUnit(source string) *TranslationUnit {
	data := []byte(source)
	return &TranslationUnit{
		ID: "example:test/example.go", Kind: UnitKindExample,
		SourcePath: "_content/tour/test/example.go", Source: data, SourceSHA256: sum(data),
	}
}

func TestValidateGoExampleCandidateAllowsNaturalCommentTranslation(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	source := "package main\n\n// Send the value through the channel.\nfunc main() { println(1) }\n"
	candidate := "package main\n\n// 通过通道发送该值。\nfunc main() { println(1) }\n"
	if err := ValidateGoExampleCandidate(root, goExampleValidationUnit(source), "zh-CN", []byte(candidate)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGoExampleCandidateAllowsNumericConstantsShiftVerbTranslation(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := catalog.Unit("example:basics/numeric-constants.go")
	if err != nil {
		t.Fatal(err)
	}
	candidate := strings.Replace(string(unit.Source), "Shift it right again 99 places, so we end up with 1<<1, or 2.", "再向右移位 99 位，最终得到 1<<1，即 2。", 1)
	if candidate == string(unit.Source) {
		t.Fatal("numeric-constants Shift comment was not found")
	}
	if err := ValidateGoExampleCandidate(root, unit, "zh-CN", []byte(candidate)); err != nil {
		t.Fatalf("translated Shift verb candidate: %v", err)
	}
}

func TestValidateGoExampleCandidateRejectsNonCommentChanges(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	tests := []struct {
		name      string
		source    string
		candidate string
	}{
		{"integer", "package main\n// Translate this ordinary comment.\nvar value = 1\n", "package main\n// 翻译此普通注释。\nvar value = 2\n"},
		{"identifier", "package main\n// Translate this ordinary comment.\nvar value = 1\n", "package main\n// 翻译此普通注释。\nvar v = 1\n"},
		{"string", "package main\n// Translate this ordinary comment.\nvar value = \"hello\"\n", "package main\n// 翻译此普通注释。\nvar value = \"你好\"\n"},
		{"raw string", "package main\n// Translate this ordinary comment.\nvar value = `hello`\n", "package main\n// 翻译此普通注释。\nvar value = `你好`\n"},
		{"rune", "package main\n// Translate this ordinary comment.\nvar value = 'a'\n", "package main\n// 翻译此普通注释。\nvar value = '中'\n"},
		{"import", "package main\nimport \"fmt\"\n// Translate this ordinary comment.\nvar _ = fmt.Print\n", "package main\nimport \"log\"\n// 翻译此普通注释。\nvar _ = fmt.Print\n"},
		{"package", "package main\n// Translate this ordinary comment.\nvar value = 1\n", "package demo\n// 翻译此普通注释。\nvar value = 1\n"},
		{"formatting", "package main\n// Translate this ordinary comment.\nvar value = 1\n", "package main\n// 翻译此普通注释。\nvar  value = 1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGoExampleCandidate(root, goExampleValidationUnit(tt.source), "zh-CN", []byte(tt.candidate))
			if err == nil {
				t.Fatal("changed candidate unexpectedly passed")
			}
		})
	}
}

func TestValidateGoExampleCandidateRejectsCommentStructureChanges(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	tests := []struct {
		name      string
		source    string
		candidate string
	}{
		{"deleted", "package main\n// Translate this ordinary comment.\nvar x = 1\n", "package main\nvar x = 1\n"},
		{"added", "package main\nvar x = 1\n", "package main\n// Add this ordinary comment.\nvar x = 1\n"},
		{"moved", "package main\n// Translate this ordinary comment.\nvar x = 1\nvar y = 2\n", "package main\nvar x = 1\n// 翻译此普通注释。\nvar y = 2\n"},
		{"line to block", "package main\n// Translate this ordinary comment.\nvar x = 1\n", "package main\n/* 翻译此普通注释。 */\nvar x = 1\n"},
		{"merged", "package main\n// Translate the first comment.\n// Translate the second comment.\nvar x = 1\n", "package main\n// 翻译第一和第二条注释。\nvar x = 1\n"},
		{"split", "package main\n// Translate this ordinary comment.\nvar x = 1\n", "package main\n// 翻译这条普通注释。\n// 这是拆出的第二条注释。\nvar x = 1\n"},
		{"changed to directive", "package main\n// Translate this ordinary comment.\nfunc main() {}\n", "package main\n//go:noinline\nfunc main() {}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateGoExampleCandidate(root, goExampleValidationUnit(tt.source), "zh-CN", []byte(tt.candidate)); err == nil {
				t.Fatal("comment structure change unexpectedly passed")
			}
		})
	}
}

func TestValidateGoExampleCandidateRejectsSpecialCommentChanges(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	for _, tt := range []struct{ name, source, candidate string }{
		{"go directive", "//go:build OMIT\n\npackage main\n", "//go:build norun\n\npackage main\n"},
		{"present marker", "package main\n\n// START OMIT\nvar x = 1\n// END OMIT\n", "package main\n\n// START HIDE\nvar x = 1\n// END OMIT\n"},
		{"conservative", "package main\n\n// panic\nvar x = 1\n", "package main\n\n// 恐慌\nvar x = 1\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateGoExampleCandidate(root, goExampleValidationUnit(tt.source), "zh-CN", []byte(tt.candidate)); err == nil || !strings.Contains(err.Error(), "特殊或不可翻译注释") {
				t.Fatalf("special comment error=%v", err)
			}
		})
	}
}

func TestValidateGoExampleCandidateGlossaryRules(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	unit := goExampleValidationUnit("package main\n\n// A goroutine uses Go and gofmt with a channel.\nfunc main() {}\n")
	if err := ValidateGoExampleCandidate(root, unit, "zh-CN", []byte("package main\n\n// goroutine 使用 Go 和 gofmt 以及一个通道。\nfunc main() {}\n")); err != nil {
		t.Fatalf("preferred translation and preserved keep terms: %v", err)
	}
	for _, candidate := range []string{
		"package main\n\n// 协程使用 Go 和 gofmt 以及一个通道。\nfunc main() {}\n",
		"package main\n\n// goroutine 使用围棋和 gofmt 以及一个通道。\nfunc main() {}\n",
	} {
		if err := ValidateGoExampleCandidate(root, unit, "zh-CN", []byte(candidate)); err == nil || !strings.Contains(err.Error(), "glossary.keep") {
			t.Fatalf("keep change error=%v", err)
		}
	}
	forbiddenUnit := goExampleValidationUnit("package main\n\n// Translate this ordinary comment.\nfunc main() {}\n")
	if err := ValidateGoExampleCandidate(root, forbiddenUnit, "zh-CN", []byte("package main\n\n// 这是禁止的幻灯片译法。\nfunc main() {}\n")); err == nil || !strings.Contains(err.Error(), "禁止译法") {
		t.Fatalf("forbidden error=%v", err)
	}
}

func TestValidateGoExampleCandidateNormalizesUppercaseTechnicalPluralKeepIdentity(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	tests := []struct {
		name, source, candidate string
	}{
		{"URL with Korean object particle", "Fetch URLs in parallel.", "URL을 병렬로 가져오세요."},
		{"URL with Korean plural and object particles", "Fetch URLs in parallel.", "URL들을 병렬로 가져오세요."},
		{"URL with Korean quantifier", "Fetch URLs in parallel.", "여러 URL을 병렬로 가져오세요."},
		{"API identity", "Use APIs in this example.", "이 예제에서 API를 사용하세요."},
		{"GOPATH identity", "Compare GOPATHs in this example.", "이 예제에서 GOPATH를 비교하세요."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := "package main\n\n// " + tt.source + "\nfunc main() {}\n"
			candidate := "package main\n\n// " + tt.candidate + "\nfunc main() {}\n"
			if err := ValidateGoExampleCandidate(root, goExampleValidationUnit(source), "zh-CN", []byte(candidate)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestValidateGoExampleCandidateRejectsInvalidUppercaseTechnicalPluralKeepIdentity(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	source := "package main\n\n// Fetch URLs in parallel.\nfunc main() {}\n"
	unit := goExampleValidationUnit(source)
	for _, tt := range []struct{ name, candidate string }{
		{"embedded suffix", "URLsafe 값을 병렬로 가져오세요."},
		{"embedded prefix", "myURLs 값을 병렬로 가져오세요."},
		{"embedded following identifier", "URLsWorker 값을 병렬로 가져오세요."},
		{"underscore extension", "URL_s 값을 병렬로 가져오세요."},
		{"wrong identity case", "Url을 병렬로 가져오세요."},
		{"uppercase plural suffix", "URLS를 병렬로 가져오세요."},
		{"two lowercase suffixes", "URLss를 병렬로 가져오세요."},
		{"es suffix", "URLes를 병렬로 가져오세요."},
		{"ies suffix", "URLies를 병렬로 가져오세요."},
		{"missing identity", "값을 병렬로 가져오세요."},
		{"translated identity", "주소를 병렬로 가져오세요."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			candidate := "package main\n\n// " + tt.candidate + "\nfunc main() {}\n"
			if err := ValidateGoExampleCandidate(root, unit, "zh-CN", []byte(candidate)); err == nil || !strings.Contains(err.Error(), "glossary.keep") {
				t.Fatalf("keep identity error=%v", err)
			}
		})
	}
}

func TestValidateGoExampleCandidateKeepsExistingExactKeepFormsIndependent(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	source := "package main\n\n// A goroutine uses Go, gofmt, and GOPATH.\nfunc main() {}\n"
	candidate := "package main\n\n// goroutine은 Go, gofmt, GOPATH를 사용합니다.\nfunc main() {}\n"
	if err := ValidateGoExampleCandidate(root, goExampleValidationUnit(source), "zh-CN", []byte(candidate)); err != nil {
		t.Fatal(err)
	}

	pluralSource := "package main\n\n// Both goroutines use Go in parallel.\nfunc main() {}\n"
	singularCandidate := "package main\n\n// goroutine은 Go를 병렬로 사용합니다.\nfunc main() {}\n"
	if err := ValidateGoExampleCandidate(root, goExampleValidationUnit(pluralSource), "zh-CN", []byte(singularCandidate)); err == nil || !strings.Contains(err.Error(), "glossary.keep") {
		t.Fatalf("explicit goroutine surface forms were merged: %v", err)
	}

	exact := []string{"Go", "gofmt", "GOPATH", "goroutine", "goroutines", "Goroutines"}
	items := goExampleKeepItems(strings.Join(exact, " "), &Glossary{Keep: exact})
	want := append([]string(nil), exact...)
	sort.Strings(want)
	if !equalStrings(items, want) {
		t.Fatalf("keep items = %v, want %v", items, want)
	}
}

func TestGoExamplePluralKeepIdentityNormalizationDoesNotChangeProtection(t *testing.T) {
	source := []byte("package main\n\n// Fetch URLs in parallel.\nfunc main() {}\n")
	protected, err := prepareGoExampleTranslationInput(source, sum(source), &Glossary{Keep: []string{"URL"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(protected.Text, "URLs") || containsString(protected.Values, "URL") {
		t.Fatalf("plural identity changed protected input: text=%q values=%q", protected.Text, protected.Values)
	}
}

func TestValidateGoExampleCandidateIgnoresCommentTextInsideStrings(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	source := "package main\n\n// Translate this ordinary comment.\nvar interpreted = \"// hello\"\nvar raw = `// hello`\n"
	candidate := "package main\n\n// 翻译此普通注释。\nvar interpreted = \"// hello\"\nvar raw = `// hello`\n"
	if err := ValidateGoExampleCandidate(root, goExampleValidationUnit(source), "zh-CN", []byte(candidate)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGoExampleCandidateRejectsParseFailureAndSourceHashMismatch(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	unit := goExampleValidationUnit("package main\n\n// Translate this ordinary comment.\nfunc main() {}\n")
	if err := ValidateGoExampleCandidate(root, unit, "zh-CN", []byte("package main\nfunc {")); err == nil || !strings.Contains(err.Error(), "候选 Go 文件无法解析") {
		t.Fatalf("parse error=%v", err)
	}
	unit.SourceSHA256 = strings.Repeat("0", 64)
	if err := ValidateGoExampleCandidate(root, unit, "zh-CN", unit.Source); err == nil || !strings.Contains(err.Error(), "完整源版本哈希不匹配") {
		t.Fatalf("source hash error=%v", err)
	}
}

func TestValidateGoExampleCandidateCorpusBaseline(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	translatable := 0
	for i := range catalog.Examples {
		example := &catalog.Examples[i]
		unit, err := catalog.Unit(example.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateGoExampleCandidate(root, unit, "zh-CN", unit.Source); err != nil {
			t.Fatalf("%s source baseline: %v", unit.ID, err)
		}
		hasContent, err := hasTranslatableGoExampleComment(unit.Source)
		if err != nil {
			t.Fatal(err)
		}
		if hasContent {
			translatable++
		}
	}
	if len(catalog.Examples) != 93 || translatable != 19 {
		t.Fatalf("corpus baseline examples=%d translatable=%d, want 93/19", len(catalog.Examples), translatable)
	}
}

func TestValidateTranslationUnitCandidateDispatchesExample(t *testing.T) {
	root := t.TempDir()
	writeGoExampleValidationGlossary(t, root)
	example := retranslationTestExample("example:demo/main.go", "_content/tour/demo/main.go", "package main\n\n// Translate this ordinary comment.\nfunc main() {}\n")
	catalog := &Catalog{Examples: []Example{example}}
	if err := ValidateTranslationUnitCandidate(root, catalog, example.ID, "zh-CN", example.Source); err != nil {
		t.Fatal(err)
	}
}
