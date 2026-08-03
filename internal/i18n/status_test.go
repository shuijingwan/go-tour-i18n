package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
		switch s.PageID {
		case "welcome/1":
			if s.State != "ready" || s.Attempts != 6 || s.SourceSHA256 != "1f581133d7fa40e6490418c6789a60a2f5e1de26c9c86d7eb6120cb58b145857" || s.CandidatePath != "locales/zh-CN/candidates/welcome-1.article" || s.UpdatedAt != "2026-08-03T11:01:01Z" || s.Note != "发布投影切换为远程执行分支，人工同步后的 candidate 已通过现有 validator" {
				t.Fatalf("welcome/1 status: %+v", s)
			}
		case "welcome/2":
			if s.State != "ready" || s.Attempts != 1 || s.CandidatePath != "locales/zh-CN/candidates/welcome-2.article" || s.UpdatedAt != "2026-08-03T09:16:52Z" || s.Note != "人工评审修订后的 candidate 已通过现有 validator" {
				t.Fatalf("welcome/2 status: %+v", s)
			}
		case "welcome/3":
			if s.State != "ready" || s.Attempts != 1 || s.CandidatePath != "locales/zh-CN/candidates/welcome-3.article" || s.UpdatedAt != "2026-08-03T09:45:27Z" || s.Note != "人工评审修订后的 candidate 已通过现有 validator" {
				t.Fatalf("welcome/3 status: %+v", s)
			}
		case "welcome/4":
			if s.State != "ready" || s.Attempts != 1 || s.CandidatePath != "locales/zh-CN/candidates/welcome-4.article" || s.UpdatedAt != "2026-08-03T10:17:53Z" || s.Note != "人工评审修订后的 candidate 已通过现有 validator" {
				t.Fatalf("welcome/4 status: %+v", s)
			}
		case "welcome/5":
			if s.State != "ready" || s.Attempts != 1 || s.CandidatePath != "locales/zh-CN/candidates/welcome-5.article" || s.UpdatedAt != "2026-08-03T10:50:33Z" || s.Note != "人工评审修订后的 candidate 已通过现有 validator" {
				t.Fatalf("welcome/5 status: %+v", s)
			}
		case "basics/1":
			if s.State != "ready" || s.Attempts != 1 || s.SourceSHA256 != "f769f12c0a028b2f0cd403d89ff39dd150405e9f2e4155875321522f08619fe0" || s.CandidatePath != "locales/zh-CN/candidates/basics-1.article" || s.UpdatedAt != "2026-08-03T11:10:32Z" || s.Note != "GLM-5.2 candidate passed existing validator" {
				t.Fatalf("basics/1 status: %+v", s)
			}
		case "basics/2":
			if s.State != "ready" || s.Attempts != 1 || s.SourceSHA256 != "3329c9bff5f7e2b9b1e161fdebfb3804ff57cf1fb11bd4327d228328bcfb3fd0" || s.CandidatePath != "locales/zh-CN/candidates/basics-2.article" || s.UpdatedAt != "2026-08-03T13:30:03Z" || s.Note != "人工评审修订后的 candidate 已通过现有 validator" {
				t.Fatalf("basics/2 status: %+v", s)
			}
		case "basics/3":
			if s.State != "ready" || s.Attempts != 1 || s.SourceSHA256 != "38b9c70e49184a24f63f6b12f8ba78e64c0b874ad2a6f9a5fe86267615fd1bf6" || s.CandidatePath != "locales/zh-CN/candidates/basics-3.article" || s.UpdatedAt != "2026-08-03T13:37:50Z" || s.Note != "人工评审修订后的 candidate 已通过现有 validator" {
				t.Fatalf("basics/3 status: %+v", s)
			}
		default:
			if s.State != "pending" || s.Attempts != 0 || s.CandidatePath != "" {
				t.Fatalf("non-initial status: %+v", s)
			}
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

func TestStatusUsesPersistentIDNotCatalogPositionOrRoute(t *testing.T) {
	a := strings.Repeat("a", 64)
	b := strings.Repeat("b", 64)
	status := "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n" +
		"x/1\tpending\t0\t" + a + "\t\t\t\n" +
		"x/2\tpending\t0\t" + b + "\t\t\t\n"
	root := writeStatusFixture(t, status)
	catalog := &Catalog{Pages: []Page{
		{ID: "x/2", Route: "/other/3", SectionNumber: 3, SourceSHA256: b},
		{ID: "x/1", Route: "/other/8", SectionNumber: 8, SourceSHA256: a},
	}}
	if err := CheckStatus(root, "zh-CN", catalog); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateTranslationStatusWritesCanonicalTSV(t *testing.T) {
	a := strings.Repeat("a", 64)
	b := strings.Repeat("b", 64)
	c := strings.Repeat("c", 64)
	before := "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n" +
		"x/1\tready\t2\t" + a + "\tlocales/zh-CN/candidates/x-1.article\t2026-01-02T03:04:05Z\texisting ready\n" +
		"x/2\tpending\t0\t" + b + "\t\"\"\t\"\"\t\"\"\n" +
		"x/3\tpending\t0\t" + c + "\t\"\"\t\"\"\t\"\"\n"
	root := writeStatusFixture(t, before)

	if err := updateTranslationStatus(root, "zh-CN", "x/2", "ready", 1, b, "locales/zh-CN/candidates/x-2.article", "2026-02-03T04:05:06Z", "candidate passed"); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, "locales", "zh-CN", "status.tsv")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(before,
		"x/2\tpending\t0\t"+b+"\t\"\"\t\"\"\t\"\"",
		"x/2\tready\t1\t"+b+"\tlocales/zh-CN/candidates/x-2.article\t2026-02-03T04:05:06Z\tcandidate passed", 1)
	if string(got) != want {
		t.Fatalf("status.tsv mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
	for lineNumber, line := range strings.Split(strings.TrimSuffix(string(got), "\n"), "\n") {
		if strings.TrimRight(line, " \t") != line {
			t.Errorf("line %d has trailing whitespace: %q", lineNumber+1, line)
		}
	}

	statuses, err := ReadStatuses(path)
	if err != nil {
		t.Fatal(err)
	}
	wantStatuses := []Status{
		{"x/1", "ready", 2, a, "locales/zh-CN/candidates/x-1.article", "2026-01-02T03:04:05Z", "existing ready"},
		{"x/2", "ready", 1, b, "locales/zh-CN/candidates/x-2.article", "2026-02-03T04:05:06Z", "candidate passed"},
		{"x/3", "pending", 0, c, "", "", ""},
	}
	if !reflect.DeepEqual(statuses, wantStatuses) {
		t.Fatalf("round trip statuses = %#v, want %#v", statuses, wantStatuses)
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
