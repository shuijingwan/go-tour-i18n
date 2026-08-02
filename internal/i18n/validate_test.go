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
