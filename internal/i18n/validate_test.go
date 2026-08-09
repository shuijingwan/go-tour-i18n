package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const syntheticSource = `* Source title

Source paragraph with [[https://example.test/path][source link]] and ` + "`inline()`" + `.

  preformatted code

.play basics/packages.go

.image /tour/static/img/tree.png
`

func syntheticCatalog() *Catalog {
	return &Catalog{Pages: []Page{{ID: "synthetic/1", Article: "basics.article", Source: []byte(syntheticSource), SourceSHA256: sum([]byte(syntheticSource))}}}
}

func TestCandidateAllowedChanges(t *testing.T) {
	root := repoRoot(t)
	c := syntheticCatalog()
	candidates := []string{
		syntheticSource,
		strings.Replace(strings.Replace(syntheticSource, "Source title", "测试标题", 1), "Source paragraph", "普通说明文字", 1),
		strings.Replace(syntheticSource, "Source paragraph", "New paragraph.\n\nAnother ordinary paragraph", 1),
		strings.Replace(syntheticSource, "source link", "链接显示文字", 1),
	}
	for i, candidate := range candidates {
		if err := ValidateCandidate(root, c, "synthetic/1", []byte(candidate)); err != nil {
			t.Fatalf("candidate %d: %v", i, err)
		}
	}
}

func TestCandidateProtectedChangesFail(t *testing.T) {
	root := repoRoot(t)
	c := syntheticCatalog()
	tests := map[string]string{
		"missing-play":          strings.Replace(syntheticSource, ".play basics/packages.go\n", "", 1),
		"changed-play":          strings.Replace(syntheticSource, "basics/packages.go", "basics/imports.go", 1),
		"extra-play":            syntheticSource + "\n.play basics/imports.go\n",
		"missing-image":         strings.Replace(syntheticSource, ".image /tour/static/img/tree.png\n", "", 1),
		"changed-image":         strings.Replace(syntheticSource, "tree.png", "gopher.png", 1),
		"changed-url":           strings.Replace(syntheticSource, "https://example.test/path", "https://example.test/other", 1),
		"changed-inline-code":   strings.Replace(syntheticSource, "`inline()`", "`other()`", 1),
		"changed-code-block":    strings.Replace(syntheticSource, "  preformatted code", "  changed code", 1),
		"extra-section":         syntheticSource + "\n* Extra section\n\nText.\n",
		"broken-present":        strings.Replace(syntheticSource, ".play basics/packages.go", ".play", 1),
		"empty":                 "",
		"unsupported-directive": syntheticSource + "\n.video example.mp4\n",
	}
	for name, candidate := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidateCandidate(root, c, "synthetic/1", []byte(candidate)); err == nil {
				t.Fatal("invalid candidate accepted")
			}
		})
	}
	if err := ValidateCandidate(root, c, "missing/1", []byte(syntheticSource)); err == nil {
		t.Fatal("unknown page_id accepted")
	}
}

func TestCandidateRejectsAppengineInSourceOrCandidate(t *testing.T) {
	root := repoRoot(t)
	badSource := syntheticSource + "#appengine: a remote server.\n"
	catalog := &Catalog{Pages: []Page{{ID: "synthetic/1", Article: "basics.article", Source: []byte(badSource), SourceSHA256: sum([]byte(badSource))}}}
	if err := ValidateCandidate(root, catalog, "synthetic/1", []byte(syntheticSource)); err == nil || !strings.Contains(err.Error(), "standalone source contains #appengine") {
		t.Fatalf("bad standalone source error = %v", err)
	}
	if err := ValidateCandidate(root, syntheticCatalog(), "synthetic/1", []byte(syntheticSource+"#appengine: translated remote branch\n")); err == nil || !strings.Contains(err.Error(), "standalone candidate contains #appengine") {
		t.Fatalf("bad standalone candidate error = %v", err)
	}
}

func TestWelcomeCandidateMandatoryGlossary(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("welcome/1")
	if err != nil {
		t.Fatal(err)
	}
	valid := string(page.Source)
	for old, replacement := range map[string]string{
		"[A Tour of Go]": "[Go 语言之旅]",
		`["previous"]`:   `["上一页"]`,
		`["next"]`:       `["下一页"]`,
		"[Run]":          "[运行]",
		"[Format]":       "[格式化]",
		"slides":         "页面",
	} {
		valid = strings.ReplaceAll(valid, old, replacement)
	}
	if err := ValidateCandidate(root, catalog, "welcome/1", []byte(valid)); err != nil {
		t.Fatalf("valid mandatory glossary candidate: %v", err)
	}
	tests := map[string]string{
		"tour label": strings.Replace(valid, "[Go 语言之旅]", "[A Tour of Go]", 1),
		"previous":   strings.Replace(valid, `["上一页"]`, `["previous"]`, 1),
		"next":       strings.Replace(valid, `["下一页"]`, `["next"]`, 1),
		"run":        strings.Replace(valid, "[运行]", "[Run]", 1),
		"format":     strings.Replace(valid, "[格式化]", "[Format]", 1),
		"slides":     strings.Replace(valid, "页面", "幻灯片", 1),
	}
	for name, candidate := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidateCandidate(root, catalog, "welcome/1", []byte(candidate)); err == nil {
				t.Fatal("glossary violation accepted")
			}
		})
	}
}

func TestCandidateRejectsForbiddenTourTranslations(t *testing.T) {
	root := repoRoot(t)
	catalog := syntheticCatalog()
	for name, phrase := range map[string]string{
		"literal tour":      "本之旅",
		"unnatural welcome": "欢迎使用 Go 编程语言之旅",
	} {
		t.Run(name, func(t *testing.T) {
			candidate := strings.Replace(syntheticSource, "Source paragraph", phrase, 1)
			if err := ValidateCandidate(root, catalog, "synthetic/1", []byte(candidate)); err == nil || !strings.Contains(err.Error(), "forbidden zh-CN translation") {
				t.Fatalf("forbidden candidate error = %v", err)
			}
		})
	}
	natural := strings.Replace(syntheticSource, "Source paragraph", "本教程介绍相关内容", 1)
	if err := ValidateCandidate(root, catalog, "synthetic/1", []byte(natural)); err != nil {
		t.Fatalf("natural 本教程 candidate rejected: %v", err)
	}
}

func TestCandidateValidationDoesNotWriteStatus(t *testing.T) {
	path := filepath.Join(repoRoot(t), "locales", "zh-CN", "status.tsv")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateCandidate(repoRoot(t), syntheticCatalog(), "synthetic/1", []byte(syntheticSource)); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("candidate validation modified status.tsv")
	}
}

func TestReorderedProtectedTokensRemainStructurallyValid(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("methods/20 cross-category and inline reorder", func(t *testing.T) {
		page, err := catalog.Page("methods/20")
		if err != nil {
			t.Fatal(err)
		}
		p := protectTranslation(page.Source, page.SourceSHA256, nil)
		if len(p.Tokens) != 16 {
			t.Fatalf("tokens=%d, want 16", len(p.Tokens))
		}
		model := "* 练习：错误\n\n" +
			"从[[" + p.Tokens[1] + "][前一个练习]]中复制你的 " + p.Tokens[0] + " 函数，并修改它使其返回一个 " + p.Tokens[2] + " 值。\n\n" +
			"当传入负数时，" + p.Tokens[3] + " 应当返回一个非 nil 的错误值，因为它不支持复数。\n\n创建一个新类型\n\n" +
			p.Tokens[4] + "并为其实现\n\n" + p.Tokens[6] + "方法，使它成为一个 " + p.Tokens[5] + "，这样 " + p.Tokens[7] + " 就会返回 " + p.Tokens[8] + "。\n\n" +
			"*注意：* 在 " + p.Tokens[10] + " 方法中调用 " + p.Tokens[9] + " 会导致程序陷入无限循环。可以先转换 " + p.Tokens[11] + " 来避免这个问题：" + p.Tokens[12] + "。为什么？\n\n" +
			"修改 " + p.Tokens[13] + " 函数，使其在传入负数时返回一个 " + p.Tokens[14] + " 值。\n\n" + p.Tokens[15] + "\n"
		candidate, failures := p.restore(model)
		if len(failures) != 0 {
			t.Fatal(failures)
		}
		if err := ValidateCandidate(root, catalog, "methods/20", []byte(candidate)); err != nil {
			t.Fatalf("reordered methods/20 candidate rejected: %v\n%s", err, candidate)
		}
	})

	t.Run("generics/1 comparable and T reorder", func(t *testing.T) {
		page, err := catalog.Page("generics/1")
		if err != nil {
			t.Fatal(err)
		}
		p := protectTranslation(page.Source, page.SourceSHA256, nil)
		var comparable, typeT string
		for i, value := range p.Values {
			switch value {
			case "`comparable`":
				comparable = p.Tokens[i]
			case "`T`":
				if typeT == "" {
					typeT = p.Tokens[i]
				}
			}
		}
		model := swapProtectedTokens(p.Text, typeT, comparable)
		candidate, failures := p.restore(model)
		if len(failures) != 0 {
			t.Fatal(failures)
		}
		if err := ValidateCandidate(root, catalog, "generics/1", []byte(candidate)); err != nil {
			t.Fatalf("reordered generics/1 candidate rejected: %v\n%s", err, candidate)
		}
	})
}

func TestSectionStructureProtection(t *testing.T) {
	root := repoRoot(t)
	source := "* Root\n\n  first block\n\n** Child\n\n  second block\n\n.play basics/packages.go\n"
	catalog := &Catalog{Pages: []Page{{ID: "synthetic/sections", Article: "basics.article", Source: []byte(source), SourceSHA256: sum([]byte(source))}}}
	tests := map[string]string{
		"preformatted moved to child": "* Root\n\n** Child\n\n  first block\n\n  second block\n\n.play basics/packages.go\n",
		"directive moved to root":     "* Root\n\n.play basics/packages.go\n\n** Child\n\n  second block\n",
	}
	for name, candidate := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidateCandidate(root, catalog, "synthetic/sections", []byte(candidate)); err == nil {
				t.Fatalf("invalid section move accepted:\n%s", candidate)
			}
		})
	}
}

func TestTerminalDirectiveCannotMoveEarlier(t *testing.T) {
	root := repoRoot(t)
	source := "* Root\n\nText.\n\n.play basics/packages.go\n"
	catalog := &Catalog{Pages: []Page{{ID: "synthetic/directive", Article: "basics.article", Source: []byte(source), SourceSHA256: sum([]byte(source))}}}
	if err := ValidateCandidate(root, catalog, "synthetic/directive", []byte(source)); err != nil {
		t.Fatalf("terminal directive in its original position rejected: %v", err)
	}
	candidate := "* Root\n\n.play basics/packages.go\n\nText.\n"
	if err := ValidateCandidate(root, catalog, "synthetic/directive", []byte(candidate)); err == nil {
		t.Fatalf("moved terminal directive accepted:\n%s", candidate)
	}
}

func TestNonTerminalDirectivePlacement(t *testing.T) {
	root := repoRoot(t)
	source := "* Root\n\nBefore image.\n\n.image /tour/static/img/tree.png\n\nAfter image.\n\n  static code\n\nAfter code.\n"
	catalog := &Catalog{Pages: []Page{{ID: "synthetic/nonterminal-directive", Article: "basics.article", Source: []byte(source), SourceSHA256: sum([]byte(source))}}}
	tests := []struct {
		name      string
		candidate string
		wantValid bool
	}{
		{
			name:      "image remains between prose regions",
			candidate: "* Root\n\n图片前的译文。\n\n.image /tour/static/img/tree.png\n\n图片后的译文。\n\n另一段译文。\n\n  static code\n\n代码后的译文。\n",
			wantValid: true,
		},
		{
			name:      "image moved to section start",
			candidate: "* Root\n\n.image /tour/static/img/tree.png\n\nBefore image.\n\nAfter image.\n\n  static code\n\nAfter code.\n",
		},
		{
			name:      "image moved to section end",
			candidate: "* Root\n\nBefore image.\n\nAfter image.\n\n  static code\n\nAfter code.\n\n.image /tour/static/img/tree.png\n",
		},
		{
			name:      "image directive changed",
			candidate: strings.Replace(source, "tree.png", "gopher.png", 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCandidate(root, catalog, "synthetic/nonterminal-directive", []byte(tt.candidate))
			if tt.wantValid && err != nil {
				t.Fatalf("valid candidate rejected: %v", err)
			}
			if !tt.wantValid && err == nil {
				t.Fatalf("invalid candidate accepted:\n%s", tt.candidate)
			}
		})
	}
}

func TestNonTerminalPlayDirectivePlacementIsGeneric(t *testing.T) {
	root := repoRoot(t)
	source := "* Root\n\nBefore program.\n\n.play basics/packages.go\n\nAfter program.\n\n  static code\n"
	catalog := &Catalog{Pages: []Page{{ID: "synthetic/nonterminal-play", Article: "basics.article", Source: []byte(source), SourceSHA256: sum([]byte(source))}}}
	candidate := "* Root\n\nBefore program.\n\nAfter program.\n\n  static code\n\n.play basics/packages.go\n"
	if err := ValidateCandidate(root, catalog, "synthetic/nonterminal-play", []byte(candidate)); err == nil {
		t.Fatalf("moved nonterminal .play accepted:\n%s", candidate)
	}
}

func TestMultipleDirectivesCannotMoveAsAnOrderedGroup(t *testing.T) {
	root := repoRoot(t)
	source := "* Root\n\nBefore first directive.\n\n.play basics/packages.go\n\nBetween directives.\n\n  static code\n\n.image /tour/static/img/tree.png\n\nAfter second directive.\n"
	catalog := &Catalog{Pages: []Page{{ID: "synthetic/multiple-directives", Article: "basics.article", Source: []byte(source), SourceSHA256: sum([]byte(source))}}}
	candidate := "* Root\n\n.play basics/packages.go\n\n.image /tour/static/img/tree.png\n\nBefore first directive.\n\nBetween directives.\n\n  static code\n\nAfter second directive.\n"
	if err := ValidateCandidate(root, catalog, "synthetic/multiple-directives", []byte(candidate)); err == nil {
		t.Fatalf("ordered directives moved as a group accepted:\n%s", candidate)
	}
}

func TestStructuralPayloadMutationsFail(t *testing.T) {
	root := repoRoot(t)
	tests := []struct {
		name, source, candidate string
	}{
		{
			name:      "legacy program span changed",
			source:    "* Root\n\nUse `package`rand`.\n",
			candidate: "* Root\n\nUse `package`math`.\n",
		},
		{
			name:      "link target outside link",
			source:    "* Root\n\n[[https://one.test][one]]\n",
			candidate: "* Root\n\n[[changed][one]] https://one.test\n",
		},
		{
			name:      "link targets swapped",
			source:    "* Root\n\n[[https://one.test][one]] [[https://two.test][two]]\n",
			candidate: "* Root\n\n[[https://two.test][one]] [[https://one.test][two]]\n",
		},
		{
			name:      "preformatted blocks swapped",
			source:    "* Root\n\n  first block\n\n  second block\n",
			candidate: "* Root\n\n  second block\n\n  first block\n",
		},
		{
			name:      "preformatted block changed",
			source:    "* Root\n\n  first block\n",
			candidate: "* Root\n\n  changed block\n",
		},
		{
			name:      "directive content changed",
			source:    "* Root\n\n.play basics/packages.go\n",
			candidate: "* Root\n\n.play basics/imports.go\n",
		},
		{
			name:      "directive order changed",
			source:    "* Root\n\n.play basics/packages.go\n\n.image /tour/static/img/tree.png\n",
			candidate: "* Root\n\n.image /tour/static/img/tree.png\n\n.play basics/packages.go\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := &Catalog{Pages: []Page{{ID: "synthetic/mutation", Article: "basics.article", Source: []byte(tt.source), SourceSHA256: sum([]byte(tt.source))}}}
			if err := ValidateCandidate(root, catalog, "synthetic/mutation", []byte(tt.candidate)); err == nil {
				t.Fatalf("invalid structural mutation accepted:\n%s", tt.candidate)
			}
		})
	}
}

func swapProtectedTokens(text, first, second string) string {
	const placeholder = "__PROTECTED_TOKEN_SWAP__"
	text = strings.Replace(text, first, placeholder, 1)
	text = strings.Replace(text, second, first, 1)
	return strings.Replace(text, placeholder, second, 1)
}
