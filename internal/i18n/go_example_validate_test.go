package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGoExampleValidationGlossary(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "locales", "zh-CN")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	body := "mandatory:\n  Go: Go\npreferred:\n  channel: 通道\nforbidden:\n  - 幻灯片\nkeep:\n  - Go\n  - gofmt\n  - goroutine\n"
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
