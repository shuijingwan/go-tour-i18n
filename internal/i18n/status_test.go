package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommittedStatus(t *testing.T) {
	root := repoRoot(t)
	c, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckStatus(root, "zh-CN", c); err != nil {
		t.Fatal(err)
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", "zh-CN", "status.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range statuses {
		if s.State != "pending" || s.Attempts != 0 || s.CandidatePath != "" {
			t.Fatalf("non-initial status: %+v", s)
		}
	}
}

func TestStatusValidationFailures(t *testing.T) {
	c := &Catalog{Pages: []Page{{ID: "x/1", SourceSHA256: strings.Repeat("a", 64)}, {ID: "x/2", SourceSHA256: strings.Repeat("b", 64)}}}
	valid := "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n" +
		"x/1\tpending\t0\t" + strings.Repeat("a", 64) + "\t\t\t\n" +
		"x/2\tpending\t0\t" + strings.Repeat("b", 64) + "\t\t\t\n"
	tests := map[string]string{
		"missing":          strings.Split(valid, "x/2")[0],
		"extra":            valid + "x/3\tpending\t0\t" + strings.Repeat("c", 64) + "\t\t\t\n",
		"invalid-state":    strings.Replace(valid, "x/1\tpending", "x/1\tunknown", 1),
		"invalid-attempts": strings.Replace(valid, "x/1\tpending\t0", "x/1\tpending\tbad", 1),
		"stale-hash":       strings.Replace(valid, strings.Repeat("a", 64), strings.Repeat("f", 64), 1),
		"invalid-time":     strings.Replace(valid, "\t\t\t\n", "\t\tnot-a-time\t\n", 1),
	}
	for name, status := range tests {
		t.Run(name, func(t *testing.T) {
			root := writeStatusFixture(t, status)
			if err := CheckStatus(root, "zh-CN", c); err == nil {
				t.Fatal("invalid status accepted")
			}
		})
	}
}

func writeStatusFixture(t *testing.T, status string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "locales", "zh-CN")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	locale := Locale{Locale: "zh-CN", LanguageName: "简体中文", EnglishName: "Simplified Chinese", HTMLLang: "zh-CN", Phase: "scaffold", TranslationUnit: "present.Section", DefaultLanguage: true}
	b, err := json.Marshal(locale)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(filepath.Join(dir, "locale.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "status.tsv"), []byte(status), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}
