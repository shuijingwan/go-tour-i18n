package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmphasisItalicTranslationPassesCandidateValidator(t *testing.T) {
	source := "* Source\n\nA _type_switch_ construct.\n"
	candidate := "* 来源\n\n一种 _类型选择_ 结构。\n"
	validateEmphasisCandidate(t, source, candidate, true)
}

func TestEmphasisStructuralMutationsFailCandidateValidator(t *testing.T) {
	source := "* Source\n\nA _type_switch_ construct.\n"
	tests := map[string]string{
		"italic to bold": "* 来源\n\n一种 *类型选择* 结构。\n",
		"deleted":        "* 来源\n\n一种类型选择结构。\n",
		"added italic":   "* 来源\n\n一种 _类型选择_ _语法_ 结构。\n",
		"added bold":     "* 来源\n\n一种 _类型选择_ *语法* 结构。\n",
	}
	for name, candidate := range tests {
		t.Run(name, func(t *testing.T) {
			validateEmphasisCandidate(t, source, candidate, false)
		})
	}
}

func TestEmphasisBoldTranslationPassesCandidateValidator(t *testing.T) {
	source := "* Source\n\nThis is *important*.\n"
	candidate := "* 来源\n\n这一点 *很重要*。\n"
	validateEmphasisCandidate(t, source, candidate, true)
}

func TestEmphasisSplitAndMergeFailCandidateValidator(t *testing.T) {
	tests := []struct {
		name, source, candidate string
	}{
		{"split", "* Source\n\n_one_span_\n", "* 来源\n\n_一个_ _范围_\n"},
		{"merge", "* Source\n\n_one_ _two_\n", "* 来源\n\n_一_二_\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateEmphasisCandidate(t, tt.source, tt.candidate, false)
		})
	}
}

func TestFontMixedEmphasisAndInlineCodeOrder(t *testing.T) {
	source := "* Source\n\nUse _first_, *second*, and `code()`.\n"
	valid := "* 来源\n\n使用 _第一_、 *第二* 和 `code()`。\n"
	validateEmphasisCandidate(t, source, valid, true)

	tests := map[string]string{
		"types swapped":  "* 来源\n\n使用 *第一*、 _第二_ 和 `code()`。\n",
		"code reordered": "* 来源\n\n使用 _第一_、 `code()` 和 *第二*。\n",
	}
	for name, candidate := range tests {
		t.Run(name, func(t *testing.T) {
			validateEmphasisCandidate(t, source, candidate, false)
		})
	}
}

func TestEmphasisMethods16CandidatePassesValidator(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "locales", "zh-CN", "candidates", "methods-16.article")
	candidate, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(candidate), "_类型选择_") || strings.Contains(string(candidate), "*类型选择*") {
		t.Fatalf("methods/16 candidate does not use italic present syntax:\n%s", candidate)
	}
	if err := ValidateCandidate(root, catalog, "methods/16", candidate); err != nil {
		t.Fatalf("methods/16 candidate rejected: %v", err)
	}
}

func validateEmphasisCandidate(t *testing.T, source, candidate string, wantValid bool) {
	t.Helper()
	root := repoRoot(t)
	catalog := &Catalog{Pages: []Page{{
		ID:            "synthetic/emphasis",
		Article:       "basics.article",
		SourceSHA256:  sum([]byte(source)),
		Source:        []byte(source),
		SectionNumber: 1,
	}}}
	err := ValidateCandidate(root, catalog, "synthetic/emphasis", []byte(candidate))
	if wantValid && err != nil {
		t.Fatalf("valid candidate rejected: %v\n%s", err, candidate)
	}
	if !wantValid && err == nil {
		t.Fatalf("invalid candidate accepted:\n%s", candidate)
	}
}
