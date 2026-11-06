package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTourLanguageAuditReaderExactReadyCanonicalMembership(t *testing.T) {
	source := []byte("* Source\n\nRead this paragraph.\n")
	example := []byte("package main\n// Explain this example.\nfunc main() {}\n")
	static := []byte("package main\nfunc main() {}\n")
	c := &Catalog{Pages: []Page{{ID: "fixture/1", Article: "fixture.article", Source: source, SourceSHA256: sum(source)}}, Examples: []Example{
		{ID: "example:fixture.go", SourcePath: "_content/tour/fixture.go", Source: example, SourceSHA256: sum(example)},
		{ID: "example:static.go", SourcePath: "_content/tour/static.go", Source: static, SourceSHA256: sum(static)},
	}}
	header := "unit_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n"
	rows := fmt.Sprintf("fixture/1\tready\t1\t%s\tlocales/zh-CN/candidates/fixture-1.article\t\t\nexample:fixture.go\tready\t1\t%s\tlocales/zh-CN/candidates/fixture.go\t\t\n", sum(source), sum(example))
	// Derive filenames through the same canonical native helper, not assumptions.
	eu, _ := c.Unit("example:fixture.go")
	examplePath, err := canonicalTranslationUnitCandidatePath("zh-CN", eu)
	if err != nil {
		t.Fatal(err)
	}
	rows = strings.ReplaceAll(rows, "locales/zh-CN/candidates/fixture.go", examplePath)
	root := writeStatusFixture(t, header+rows)
	writeGoExampleValidationGlossary(t, root)
	write := func(path, text string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("locales/zh-CN/candidates/fixture-1.article", "* 标题\n\n请阅读此段落。\n")
	write(examplePath, "package main\n// 解释这个示例。\nfunc main() {}\n")
	write("locales/zh-CN/candidates/unrelated.article", "not a formal member")
	got, err := ReadTourLanguageAuditUnits(root, "zh-CN", c)
	if err != nil || len(got) != 2 || got[0].Unit.Kind != UnitKindPage || got[1].Unit.Kind != UnitKindExample {
		t.Fatal(got, err)
	}
	for _, bad := range []string{
		strings.Replace(rows, "\tready\t", "\tcandidate\t", 1),
		strings.Replace(rows, "fixture-1.article", "unrelated.article", 1),
		rows + strings.Split(rows, "\n")[0] + "\n",
		strings.Split(rows, "\n")[0] + "\n",
	} {
		write("locales/zh-CN/status.tsv", header+bad)
		if _, err := ReadTourLanguageAuditUnits(root, "zh-CN", c); err == nil {
			t.Fatal("noncanonical/incomplete/nonready membership accepted", bad)
		}
	}
}
