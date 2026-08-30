package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDeDELocaleIdentity(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "locales", "de-DE", "locale.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got Locale
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := Locale{
		Locale:          "de-DE",
		LanguageName:    "Deutsch",
		EnglishName:     "German",
		HTMLLang:        "de-DE",
		Phase:           "scaffold",
		TranslationUnit: "present.Section",
		DefaultLanguage: false,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("de-DE locale identity = %+v, want %+v", got, want)
	}
}

func TestFrFRLocaleIdentity(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "locales", "fr-FR", "locale.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got Locale
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := Locale{
		Locale:          "fr-FR",
		LanguageName:    "Français",
		EnglishName:     "French",
		HTMLLang:        "fr-FR",
		Phase:           "scaffold",
		TranslationUnit: "present.Section",
		DefaultLanguage: false,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fr-FR locale identity = %+v, want %+v", got, want)
	}
}

func TestInitializeLocaleStatusFromStableWorkflowInventory(t *testing.T) {
	pageTwo := []byte("* Two\n\nSecond Page.\n")
	pageOne := []byte("* One\n\nFirst Page.\n")
	ineligible := []byte("package main\n\nfunc main() {}\n")
	eligible := []byte("package main\n\n// Explain this example.\nfunc main() {}\n")
	catalog := &Catalog{
		Pages: []Page{
			{ID: "lesson/2", Article: "lesson.article", Source: pageTwo, SourceSHA256: sum(pageTwo)},
			{ID: "lesson/1", Article: "lesson.article", Source: pageOne, SourceSHA256: sum(pageOne)},
		},
		Examples: []Example{
			{ID: "example:demo/ineligible.go", SourcePath: "_content/tour/demo/ineligible.go", Source: ineligible, SourceSHA256: sum(ineligible)},
			{ID: "example:demo/eligible.go", SourcePath: "_content/tour/demo/eligible.go", Source: eligible, SourceSHA256: sum(eligible)},
		},
	}

	initialize := func(t *testing.T) (string, []byte) {
		t.Helper()
		root := t.TempDir()
		writeInitializableLocale(t, root, "de-DE", Locale{
			Locale: "de-DE", LanguageName: "Deutsch", EnglishName: "German", HTMLLang: "de-DE",
			Phase: "scaffold", TranslationUnit: "present.Section",
		})
		result, err := InitializeLocaleStatus(root, "de-DE", catalog)
		if err != nil {
			t.Fatal(err)
		}
		if *result != (StatusInitializationResult{Total: 3, Pages: 2, Examples: 1}) {
			t.Fatalf("initialization result = %+v", result)
		}
		path := filepath.Join(root, "locales", "de-DE", "status.tsv")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := bytes.Count(data, []byte("\n")); got != 4 {
			t.Fatalf("status.tsv lines = %d, want 4", got)
		}
		statuses, err := ReadStatuses(path)
		if err != nil {
			t.Fatal(err)
		}
		want := []Status{
			{UnitID: "lesson/2", State: "pending", Attempts: 0, SourceSHA256: sum(pageTwo)},
			{UnitID: "lesson/1", State: "pending", Attempts: 0, SourceSHA256: sum(pageOne)},
			{UnitID: "example:demo/eligible.go", State: "pending", Attempts: 0, SourceSHA256: sum(eligible)},
		}
		if !reflect.DeepEqual(statuses, want) {
			t.Fatalf("statuses = %#v, want %#v", statuses, want)
		}
		for _, status := range statuses {
			if status.CandidatePath != "" || status.UpdatedAt != "" || status.Note != "" {
				t.Fatalf("initial status contains workflow output: %+v", status)
			}
			if status.UnitID == "example:demo/ineligible.go" {
				t.Fatal("non-eligible Example was initialized")
			}
		}
		if err := CheckStatus(root, "de-DE", catalog); err != nil {
			t.Fatalf("initialized status failed CheckStatus: %v", err)
		}
		entries, err := os.ReadDir(filepath.Dir(path))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("locale directory contains temporary initialization files: %v", entries)
		}
		return root, data
	}

	_, first := initialize(t)
	_, second := initialize(t)
	if !bytes.Equal(first, second) {
		t.Fatal("same catalog produced different status.tsv bytes")
	}
}

func TestInitializeLocaleStatusRefusesExistingFileWithoutChangingBytes(t *testing.T) {
	root := t.TempDir()
	writeInitializableLocale(t, root, "de-DE", Locale{Locale: "de-DE", Phase: "scaffold", TranslationUnit: "present.Section"})
	path := filepath.Join(root, "locales", "de-DE", "status.tsv")
	before := []byte("established formal workflow state\n")
	if err := os.WriteFile(path, before, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitializeLocaleStatus(root, "de-DE", &Catalog{}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing status error = %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("existing status changed: got %q, want %q", after, before)
	}
}

func TestInitializeLocaleStatusRejectsMissingOrInvalidMetadataWithoutOutput(t *testing.T) {
	tests := []struct {
		name string
		body []byte
	}{
		{name: "missing"},
		{name: "malformed JSON", body: []byte("{")},
		{name: "locale mismatch", body: []byte(`{"locale":"de","phase":"scaffold","translation_unit":"present.Section"}`)},
		{name: "phase mismatch", body: []byte(`{"locale":"de-DE","phase":"production","translation_unit":"present.Section"}`)},
		{name: "translation unit mismatch", body: []byte(`{"locale":"de-DE","phase":"scaffold","translation_unit":"other"}`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if test.body != nil {
				dir := filepath.Join(root, "locales", "de-DE")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "locale.json"), test.body, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := InitializeLocaleStatus(root, "de-DE", &Catalog{}); err == nil {
				t.Fatal("invalid locale metadata was accepted")
			}
			if _, err := os.Lstat(filepath.Join(root, "locales", "de-DE", "status.tsv")); !os.IsNotExist(err) {
				t.Fatalf("failed initialization left status.tsv: %v", err)
			}
		})
	}
}

func TestInitializeLocaleStatusRejectsInvalidLocale(t *testing.T) {
	root := t.TempDir()
	if _, err := InitializeLocaleStatus(root, "../de-DE", &Catalog{}); err == nil || !strings.Contains(err.Error(), "invalid locale") {
		t.Fatalf("invalid locale error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "status.tsv")); !os.IsNotExist(err) {
		t.Fatalf("invalid locale created status.tsv outside locale directory: %v", err)
	}
}

func TestCommittedDeDEStatusUsesStableWorkflowOrder(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckStatus(root, "de-DE", catalog); err != nil {
		t.Fatal(err)
	}
	units, pages, examples, err := localeWorkflowUnitList(catalog)
	if err != nil {
		t.Fatal(err)
	}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", "de-DE", "status.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 122 || len(units) != 122 || pages != 103 || examples != 19 {
		t.Fatalf("de-DE workflow counts: statuses=%d units=%d pages=%d examples=%d", len(statuses), len(units), pages, examples)
	}
	for i := range units {
		if statuses[i].UnitID != units[i].ID {
			t.Fatalf("de-DE status row %d unit_id=%q, want stable workflow unit %q", i+1, statuses[i].UnitID, units[i].ID)
		}
	}
}

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
	total, pages, examples, err := LocaleWorkflowUnitCounts(c)
	if err != nil {
		t.Fatal(err)
	}
	if total != 122 || pages != 103 || examples != 19 {
		t.Fatalf("workflow counts: total=%d pages=%d examples=%d", total, pages, examples)
	}
	readyCount, pageCount, exampleCount := 0, 0, 0
	for _, s := range statuses {
		if err := validateCommittedStatus(root, c, "zh-CN", s); err != nil {
			t.Fatal(err)
		}
		unit, err := c.Unit(s.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		if s.State != "ready" || s.Attempts <= 0 || s.SourceSHA256 != unit.SourceSHA256 || s.CandidatePath == "" {
			t.Fatalf("workflow unit is not complete: %+v", s)
		}
		readyCount++
		switch unit.Kind {
		case UnitKindExample:
			exampleCount++
			if filepath.Ext(s.CandidatePath) != ".go" {
				t.Fatalf("eligible Example candidate is not canonical Go source: %+v", s)
			}
		case UnitKindPage:
			pageCount++
			if filepath.Ext(s.CandidatePath) != ".article" {
				t.Fatalf("Page candidate is not canonical article source: %+v", s)
			}
		}
	}
	if len(statuses) != total || readyCount != total || pageCount != pages || exampleCount != examples {
		t.Fatalf("status counts: total=%d ready=%d pages=%d examples=%d", len(statuses), readyCount, pageCount, exampleCount)
	}
}

func validateCommittedStatus(root string, catalog *Catalog, locale string, status Status) error {
	unit, err := catalog.Unit(status.UnitID)
	if err != nil {
		return err
	}
	canonicalCandidate, err := canonicalTranslationUnitCandidatePath(locale, unit)
	if err != nil {
		return err
	}
	switch status.State {
	case "pending":
		if status.CandidatePath != "" {
			return fmt.Errorf("%s: pending status has candidate path %q", status.UnitID, status.CandidatePath)
		}
	case "blocked":
		if status.Attempts <= 0 {
			return fmt.Errorf("%s: blocked status has attempts=%d, want > 0", status.UnitID, status.Attempts)
		}
		if status.CandidatePath != "" {
			return fmt.Errorf("%s: blocked status has candidate path %q", status.UnitID, status.CandidatePath)
		}
	case "ready", "candidate", "published":
		if status.Attempts <= 0 {
			return fmt.Errorf("%s: %s status has attempts=%d, want > 0", status.UnitID, status.State, status.Attempts)
		}
		if status.CandidatePath != canonicalCandidate {
			return fmt.Errorf("%s: %s candidate path = %q, want %q", status.UnitID, status.State, status.CandidatePath, canonicalCandidate)
		}
		candidate, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(status.CandidatePath)))
		if err != nil {
			return fmt.Errorf("%s: read committed candidate: %w", status.UnitID, err)
		}
		if err := ValidateTranslationUnitCandidate(root, catalog, status.UnitID, locale, candidate); err != nil {
			return fmt.Errorf("%s: committed candidate validation: %w", status.UnitID, err)
		}
	default:
		return fmt.Errorf("%s: unsupported committed status %q", status.UnitID, status.State)
	}
	return nil
}

func TestCommittedWorkflowCanonicalCandidatePathsAreUnique(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	units, pages, examples, err := localeWorkflowUnits(catalog)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]string, len(units))
	for _, unit := range units {
		path, err := canonicalTranslationUnitCandidatePath("zh-CN", unit)
		if err != nil {
			t.Fatal(err)
		}
		if previous := seen[path]; previous != "" {
			t.Fatalf("canonical path %q collides for %s and %s", path, previous, unit.ID)
		}
		seen[path] = unit.ID
		if unit.Kind == UnitKindPage && filepath.Ext(path) != ".article" {
			t.Errorf("Page %s canonical path = %q", unit.ID, path)
		}
		if unit.Kind == UnitKindExample && filepath.Ext(path) != ".go" {
			t.Errorf("Example %s canonical path = %q", unit.ID, path)
		}
	}
	if pages != 103 || examples != 19 || len(units) != 122 || len(seen) != 122 {
		t.Fatalf("workflow/canonical counts: units=%d pages=%d examples=%d paths=%d", len(units), pages, examples, len(seen))
	}
	for _, page := range catalog.Pages {
		unit, err := catalog.Unit(page.ID)
		if err != nil {
			t.Fatal(err)
		}
		got, err := canonicalTranslationUnitCandidatePath("zh-CN", unit)
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.ToSlash(filepath.Join("locales", "zh-CN", "candidates", strings.ReplaceAll(page.ID, "/", "-")+".article")); got != want {
			t.Fatalf("Page %s canonical path = %q, want %q", page.ID, got, want)
		}
	}
}

func TestCommittedStateInvariants(t *testing.T) {
	source := []byte("* Source\n\nSource paragraph with `code`.\n")
	root := writeStatusFixture(t, "unit_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\nexample/1\tpending\t0\t"+sum(source)+"\t\t\t\n")
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
	ready := Status{UnitID: "example/1", State: "ready", Attempts: 7, SourceSHA256: sum(source), CandidatePath: canonical}
	if err := validateCommittedStatus(root, catalog, "zh-CN", ready); err != nil {
		t.Fatalf("valid ready with historical attempts rejected: %v", err)
	}
	blocked := Status{UnitID: "example/1", State: "blocked", Attempts: 7, SourceSHA256: sum(source)}
	if err := validateCommittedStatus(root, catalog, "zh-CN", blocked); err != nil {
		t.Fatalf("valid blocked rejected: %v", err)
	}
	for name, status := range map[string]Status{
		"ready missing candidate": {UnitID: "example/1", State: "ready", Attempts: 1, SourceSHA256: sum(source), CandidatePath: "locales/zh-CN/candidates/missing.article"},
		"ready nonstandard path":  {UnitID: "example/1", State: "ready", Attempts: 1, SourceSHA256: sum(source), CandidatePath: "other.article"},
		"pending candidate":       {UnitID: "example/1", State: "pending", CandidatePath: canonical},
		"blocked no attempts":     {UnitID: "example/1", State: "blocked", Attempts: 0},
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
	valid := "unit_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n" +
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

func TestUnifiedStatusExpectedSet(t *testing.T) {
	pageSource := []byte("* Page\n\nPage source.\n")
	eligibleSource := []byte("package main\n\n// Explain this example.\nfunc main() {}\n")
	ineligibleSource := []byte("package main\n\nfunc main() {}\n")
	catalog := &Catalog{
		Pages: []Page{{ID: "page/1", Source: pageSource, SourceSHA256: sum(pageSource)}},
		Examples: []Example{
			{ID: "example:eligible.go", SourcePath: "_content/tour/eligible.go", Source: eligibleSource, SourceSHA256: sum(eligibleSource)},
			{ID: "example:ineligible.go", SourcePath: "_content/tour/ineligible.go", Source: ineligibleSource, SourceSHA256: sum(ineligibleSource)},
		},
	}
	valid := []Status{
		{UnitID: "page/1", State: "pending", SourceSHA256: sum(pageSource)},
		{UnitID: "example:eligible.go", State: "pending", SourceSHA256: sum(eligibleSource)},
	}
	check := func(t *testing.T, statuses []Status) error {
		t.Helper()
		root := writeStatusFixture(t, "unit_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n")
		if err := writeStatuses(filepath.Join(root, "locales", "zh-CN", "status.tsv"), statuses); err != nil {
			t.Fatal(err)
		}
		return CheckStatus(root, "zh-CN", catalog)
	}
	if err := check(t, valid); err != nil {
		t.Fatal(err)
	}
	tests := map[string][]Status{
		"missing page":             {valid[1]},
		"missing eligible example": {valid[0]},
		"non-eligible example":     {valid[0], {UnitID: "example:ineligible.go", State: "pending", SourceSHA256: sum(ineligibleSource)}},
		"unknown ID":               {valid[0], {UnitID: "example:unknown.go", State: "pending", SourceSHA256: sum(eligibleSource)}},
		"duplicate page":           {valid[0], valid[0]},
		"duplicate example":        {valid[1], valid[1]},
		"Example hash mismatch":    {valid[0], {UnitID: valid[1].UnitID, State: "pending", SourceSHA256: strings.Repeat("f", 64)}},
		"Page hash mismatch":       {{UnitID: valid[0].UnitID, State: "pending", SourceSHA256: strings.Repeat("f", 64)}, valid[1]},
	}
	for name, statuses := range tests {
		t.Run(name, func(t *testing.T) {
			if err := check(t, statuses); err == nil {
				t.Fatal("invalid unified status accepted")
			}
		})
	}
	t.Run("ready Example canonical path", func(t *testing.T) {
		statuses := append([]Status(nil), valid...)
		statuses[1].State = "ready"
		statuses[1].Attempts = 1
		statuses[1].CandidatePath = "locales/zh-CN/candidates/eligible.go"
		if err := check(t, statuses); err != nil {
			t.Fatal(err)
		}
		statuses[1].CandidatePath = "locales/zh-CN/candidates/eligible.article"
		if err := check(t, statuses); err == nil {
			t.Fatal("noncanonical Example candidate path accepted")
		}
	})
}

func TestStatusUsesPersistentIDNotCatalogPositionOrRoute(t *testing.T) {
	a := strings.Repeat("a", 64)
	b := strings.Repeat("b", 64)
	status := "unit_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n" +
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
	before := "unit_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n" +
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
	if err := writeStatuses(path, statuses); err != nil {
		t.Fatal(err)
	}
	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rewritten, got) {
		t.Fatal("writing the same statuses twice produced a different status.tsv")
	}
}

func writeStatusFixture(t *testing.T, status string) string {
	t.Helper()
	// Older TranslationRunner tests focus on Page behavior rather than the
	// status schema; normalize their inline fixture header to the current one.
	status = strings.Replace(status, "page_id\tstatus\t", "unit_id\tstatus\t", 1)
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

func writeInitializableLocale(t *testing.T, root, localeName string, locale Locale) {
	t.Helper()
	dir := filepath.Join(root, "locales", localeName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(locale)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "locale.json"), append(b, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
