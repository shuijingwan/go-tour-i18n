package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
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
	promotionNoteRE := regexp.MustCompile(`^ChatGPT retranslation promoted from chatgpt-zh-CN-[0-9]{3}; passed canonical validator$`)
	wantLatestBatch := map[string]string{
		"welcome/1":     "chatgpt-zh-CN-001",
		"concurrency/1": "chatgpt-zh-CN-009",
		"methods/17":    "chatgpt-zh-CN-011",
		"methods/19":    "chatgpt-zh-CN-011",
	}
	for _, s := range statuses {
		if err := validateCommittedStatus(root, c, "zh-CN", s); err != nil {
			t.Fatal(err)
		}
		if s.UpdatedAt == "" {
			t.Fatalf("%s: promoted status has empty updated_at", s.PageID)
		}
		if !promotionNoteRE.MatchString(s.Note) {
			t.Fatalf("%s: unexpected promotion note %q", s.PageID, s.Note)
		}
		if batch := wantLatestBatch[s.PageID]; batch != "" {
			wantNote := "ChatGPT retranslation promoted from " + batch + "; passed canonical validator"
			if s.Note != wantNote {
				t.Fatalf("%s: promotion note = %q, want %q", s.PageID, s.Note, wantNote)
			}
		}
		switch s.PageID {
		case "welcome/1":
			if s.State != "ready" || s.Attempts != 6 || s.SourceSHA256 != "1f581133d7fa40e6490418c6789a60a2f5e1de26c9c86d7eb6120cb58b145857" || s.CandidatePath != "locales/zh-CN/candidates/welcome-1.article" {
				t.Fatalf("welcome/1 status: %+v", s)
			}
		case "welcome/2":
			if s.State != "ready" || s.Attempts != 1 || s.CandidatePath != "locales/zh-CN/candidates/welcome-2.article" {
				t.Fatalf("welcome/2 status: %+v", s)
			}
		case "welcome/3":
			if s.State != "ready" || s.Attempts != 1 || s.CandidatePath != "locales/zh-CN/candidates/welcome-3.article" {
				t.Fatalf("welcome/3 status: %+v", s)
			}
		case "welcome/4":
			if s.State != "ready" || s.Attempts != 1 || s.CandidatePath != "locales/zh-CN/candidates/welcome-4.article" {
				t.Fatalf("welcome/4 status: %+v", s)
			}
		case "welcome/5":
			if s.State != "ready" || s.Attempts != 1 || s.CandidatePath != "locales/zh-CN/candidates/welcome-5.article" {
				t.Fatalf("welcome/5 status: %+v", s)
			}
		case "basics/1":
			if s.State != "ready" || s.Attempts != 1 || s.SourceSHA256 != "f769f12c0a028b2f0cd403d89ff39dd150405e9f2e4155875321522f08619fe0" || s.CandidatePath != "locales/zh-CN/candidates/basics-1.article" {
				t.Fatalf("basics/1 status: %+v", s)
			}
		case "basics/2":
			if s.State != "ready" || s.Attempts != 1 || s.SourceSHA256 != "3329c9bff5f7e2b9b1e161fdebfb3804ff57cf1fb11bd4327d228328bcfb3fd0" || s.CandidatePath != "locales/zh-CN/candidates/basics-2.article" {
				t.Fatalf("basics/2 status: %+v", s)
			}
		case "basics/3":
			if s.State != "ready" || s.Attempts != 1 || s.SourceSHA256 != "38b9c70e49184a24f63f6b12f8ba78e64c0b874ad2a6f9a5fe86267615fd1bf6" || s.CandidatePath != "locales/zh-CN/candidates/basics-3.article" {
				t.Fatalf("basics/3 status: %+v", s)
			}
		case "generics/1":
			if s.State != "ready" || s.Attempts != 5 || s.SourceSHA256 != "01a045105dc8c12fb1709f122d363235c19a6464d5de7587d579524aec270dd6" || s.CandidatePath != "locales/zh-CN/candidates/generics-1.article" {
				t.Fatalf("generics/1 status: %+v", s)
			}
		case "flowcontrol/8":
			if s.State != "ready" || s.Attempts != 1 || s.SourceSHA256 != "d8bbee8455ff59212ef432a08312f7c7703360367325a80df3719157200316e9" || s.CandidatePath != "locales/zh-CN/candidates/flowcontrol-8.article" {
				t.Fatalf("flowcontrol/8 status: %+v", s)
			}
		case "methods/16":
			if s.State != "ready" || s.Attempts != 2 || s.SourceSHA256 != "26e4da09e80d30b06368691c76ee2940139b0f6fc40cad47bb6d1d2947933c27" || s.CandidatePath != "locales/zh-CN/candidates/methods-16.article" {
				t.Fatalf("methods/16 status: %+v", s)
			}
		case "methods/20":
			if s.State != "ready" || s.Attempts != 1 || s.SourceSHA256 != "41f1f73320fde60ee5ff30d5927a19ff22d6da6a336bb776e15f2499e4f421d8" || s.CandidatePath != "locales/zh-CN/candidates/methods-20.article" {
				t.Fatalf("methods/20 status: %+v", s)
			}
		case "methods/24":
			if s.State != "ready" || s.Attempts != 2 || s.SourceSHA256 != "d80f0d46796ad415a7ded4d0cefdef6cdb58deb38d050a61eb64abb25caf27ee" || s.CandidatePath != "locales/zh-CN/candidates/methods-24.article" {
				t.Fatalf("methods/24 status: %+v", s)
			}
		case "concurrency/7":
			if s.State != "ready" || s.Attempts != 1 || s.SourceSHA256 != "7c5d3fc7bb2540285d746242f8a1d16075639648eec8909c1df52239297d2917" || s.CandidatePath != "locales/zh-CN/candidates/concurrency-7.article" {
				t.Fatalf("concurrency/7 status: %+v", s)
			}
		case "concurrency/11":
			if s.State != "ready" || s.Attempts != 4 || s.SourceSHA256 != "45ef131ede663b1355c0a5933634d46c393f66be8a6450b184f86a62e928a64e" || s.CandidatePath != "locales/zh-CN/candidates/concurrency-11.article" {
				t.Fatalf("concurrency/11 status: %+v", s)
			}
		}
	}
}

func validateCommittedStatus(root string, catalog *Catalog, locale string, status Status) error {
	canonicalCandidate := filepath.ToSlash(filepath.Join("locales", locale, "candidates", strings.ReplaceAll(status.PageID, "/", "-")+".article"))
	switch status.State {
	case "pending":
		if status.CandidatePath != "" {
			return fmt.Errorf("%s: pending status has candidate path %q", status.PageID, status.CandidatePath)
		}
	case "blocked":
		if status.Attempts <= 0 {
			return fmt.Errorf("%s: blocked status has attempts=%d, want > 0", status.PageID, status.Attempts)
		}
		if status.CandidatePath != "" {
			return fmt.Errorf("%s: blocked status has candidate path %q", status.PageID, status.CandidatePath)
		}
	case "ready", "candidate", "published":
		if status.Attempts <= 0 {
			return fmt.Errorf("%s: %s status has attempts=%d, want > 0", status.PageID, status.State, status.Attempts)
		}
		if status.CandidatePath != canonicalCandidate {
			return fmt.Errorf("%s: %s candidate path = %q, want %q", status.PageID, status.State, status.CandidatePath, canonicalCandidate)
		}
		candidate, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(status.CandidatePath)))
		if err != nil {
			return fmt.Errorf("%s: read committed candidate: %w", status.PageID, err)
		}
		if err := ValidateCandidateForLocale(root, catalog, status.PageID, locale, candidate); err != nil {
			return fmt.Errorf("%s: committed candidate validation: %w", status.PageID, err)
		}
	default:
		return fmt.Errorf("%s: unsupported committed status %q", status.PageID, status.State)
	}
	return nil
}

func TestCommittedStateInvariants(t *testing.T) {
	source := []byte("* Source\n\nSource paragraph with `code`.\n")
	root := writeStatusFixture(t, "page_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\nexample/1\tpending\t0\t"+sum(source)+"\t\t\t\n")
	writeTestGlossary(t, root)
	catalog := &Catalog{Pages: []Page{{ID: "example/1", Article: "basics.article", Source: source, SourceSHA256: sum(source)}}}
	canonical := "locales/zh-CN/candidates/example-1.article"
	valid := []byte("* 来源\n\n包含 `code` 的来源段落。\n")
	writeCandidate := func(t *testing.T, content []byte) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(canonical))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeCandidate(t, valid)
	ready := Status{PageID: "example/1", State: "ready", Attempts: 7, SourceSHA256: sum(source), CandidatePath: canonical}
	if err := validateCommittedStatus(root, catalog, "zh-CN", ready); err != nil {
		t.Fatalf("valid ready with historical attempts rejected: %v", err)
	}
	blocked := Status{PageID: "example/1", State: "blocked", Attempts: 7, SourceSHA256: sum(source)}
	if err := validateCommittedStatus(root, catalog, "zh-CN", blocked); err != nil {
		t.Fatalf("valid blocked rejected: %v", err)
	}
	for name, status := range map[string]Status{
		"ready missing candidate": {PageID: "example/1", State: "ready", Attempts: 1, SourceSHA256: sum(source), CandidatePath: "locales/zh-CN/candidates/missing.article"},
		"ready nonstandard path":  {PageID: "example/1", State: "ready", Attempts: 1, SourceSHA256: sum(source), CandidatePath: "other.article"},
		"pending candidate":       {PageID: "example/1", State: "pending", CandidatePath: canonical},
		"blocked no attempts":     {PageID: "example/1", State: "blocked", Attempts: 0},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateCommittedStatus(root, catalog, "zh-CN", status); err == nil {
				t.Fatal("invalid committed status accepted")
			}
		})
	}
	writeCandidate(t, []byte("* 来源\n\n损坏的 `other`。\n"))
	if err := validateCommittedStatus(root, catalog, "zh-CN", ready); err == nil {
		t.Fatal("ready candidate with changed inline code accepted")
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
