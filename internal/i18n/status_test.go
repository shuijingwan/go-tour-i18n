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
