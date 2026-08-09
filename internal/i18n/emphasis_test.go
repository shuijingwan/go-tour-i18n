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

func TestEmphasisProtectionRestoresChineseBoundariesAndKeepsContentTranslatable(t *testing.T) {
	source := "* Source\n\nUse _after_ beside `code()` and *important*.\n"
	p := protectTranslation([]byte(source), strings.Repeat("a", 64), nil)
	italicOpen := emphasisToken(t, p, protectedItalicOpen, 0)
	italicClose := emphasisToken(t, p, protectedItalicClose, 0)
	boldOpen := emphasisToken(t, p, protectedBoldOpen, 0)
	boldClose := emphasisToken(t, p, protectedBoldClose, 0)
	code := inlinePairText(t, p, 0)
	model := "* 来源\n\n使用" + italicOpen + "之后" + italicClose + "紧邻" + code + "，并且" + boldOpen + "很重要" + boldClose + "。\n"
	candidate, failures := p.restore(model)
	if len(failures) != 0 {
		t.Fatalf("restore failures: %v", failures)
	}
	if !strings.Contains(candidate, "使用 _之后_ 紧邻 `code()`，并且 *很重要*。") {
		t.Fatalf("emphasis boundary normalization did not restore safe present syntax:\n%s", candidate)
	}
	validateEmphasisCandidate(t, source, candidate, true)
}

func TestEmphasisProtectionSeparatesAdjacentInlineCode(t *testing.T) {
	source := "* Source\n\nUse _after_ `code()`.\n"
	p := protectTranslation([]byte(source), strings.Repeat("d", 64), nil)
	italicOpen := emphasisToken(t, p, protectedItalicOpen, 0)
	italicClose := emphasisToken(t, p, protectedItalicClose, 0)
	code := inlinePairText(t, p, 0)
	model := "* 来源\n\n使用" + italicOpen + "之后" + italicClose + code + "。\n"
	candidate, failures := p.restore(model)
	if len(failures) != 0 {
		t.Fatalf("restore failures: %v", failures)
	}
	if !strings.Contains(candidate, "使用 _之后_ `code()`。") {
		t.Fatalf("adjacent emphasis and inline code were not separated:\n%s", candidate)
	}
	validateEmphasisCandidate(t, source, candidate, true)
}

func TestEmphasisProtectionFailsClosedForSentinelMutations(t *testing.T) {
	source := "* Source\n\nUse _after_ and *important*.\n"
	p := protectTranslation([]byte(source), strings.Repeat("b", 64), nil)
	italicOpen := emphasisToken(t, p, protectedItalicOpen, 0)
	italicClose := emphasisToken(t, p, protectedItalicClose, 0)
	boldOpen := emphasisToken(t, p, protectedBoldOpen, 0)
	for name, model := range map[string]string{
		"missing":   strings.Replace(p.Text, italicOpen, "", 1),
		"duplicate": strings.Replace(p.Text, italicClose, italicClose+italicClose, 1),
		"type swap": strings.Replace(p.Text, italicOpen, boldOpen, 1),
		"wrong order": strings.NewReplacer(
			italicOpen, "__OPEN__",
			italicClose, italicOpen,
			"__OPEN__", italicClose,
		).Replace(p.Text),
	} {
		t.Run(name, func(t *testing.T) {
			if _, failures := p.restore(model); len(failures) == 0 {
				t.Fatal("mutated emphasis sentinels were accepted")
			}
		})
	}
}

func TestBasics4Attempt005EmphasisProtectionRegression(t *testing.T) {
	source := "* Functions\n\nA function can take zero or more arguments.\n\nIn this example, `add` takes two parameters of type `int`.\n\nNotice that the type comes _after_ the variable name.\n\n(For more about why types look the way they do, see the [[/blog/gos-declaration-syntax][article on Go's declaration syntax]].)\n\n.play basics/functions.go\n"
	p := protectTranslation([]byte(source), strings.Repeat("c", 64), nil)
	add := inlinePairText(t, p, 0)
	integer := inlinePairText(t, p, 1)
	italicOpen := emphasisToken(t, p, protectedItalicOpen, 0)
	italicClose := emphasisToken(t, p, protectedItalicClose, 0)
	target := emphasisToken(t, p, protectedLinkTarget, 0)
	goKeep := emphasisToken(t, p, protectedGlossaryOrKeep, 0)
	directive := emphasisToken(t, p, protectedDirective, 0)
	model := "* 函数\n\n函数可以接受零个或多个参数。\n\n在本例中，" + add + " 接受两个 " + integer + " 类型的参数。\n\n注意，类型位于变量名" + italicOpen + "之后" + italicClose + "。\n\n（有关类型为何采用这种形式的更多说明，请参阅 [[" + target + "][关于 " + goKeep + " 声明语法的文章]]。）\n\n" + directive + "\n"
	candidate, failures := p.restore(model)
	if len(failures) != 0 {
		t.Fatalf("restore failures: %v", failures)
	}
	if !strings.Contains(candidate, "变量名 _之后_。") {
		t.Fatalf("attempt-005 regression retained an invalid emphasis boundary:\n%s", candidate)
	}
	validateEmphasisCandidate(t, source, candidate, true)
}

func emphasisToken(t *testing.T, p protectedTranslation, kind protectedTokenKind, occurrence int) string {
	t.Helper()
	seen := 0
	for index, current := range p.Kinds {
		if current != kind {
			continue
		}
		if seen == occurrence {
			return p.Tokens[index]
		}
		seen++
	}
	t.Fatalf("missing protected token kind=%d occurrence=%d", kind, occurrence)
	return ""
}

func inlinePairText(t *testing.T, p protectedTranslation, occurrence int) string {
	t.Helper()
	if occurrence >= len(p.InlinePairs) {
		t.Fatalf("missing inline pair %d", occurrence+1)
	}
	pair := p.InlinePairs[occurrence]
	return pair.Open + pair.Content + pair.Close
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
