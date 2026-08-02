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
