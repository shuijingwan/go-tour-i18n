package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateCourseMetadataCompleteSet(t *testing.T) {
	catalog, targets, glossary, metadata := courseMetadataFixture()
	got, err := validateCourseMetadata(marshalCourseMetadata(t, metadata), "test-LOCALE", catalog, targets, glossary)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Pages) != len(catalog.Pages) {
		t.Fatalf("pages=%d, want %d", len(got.Pages), len(catalog.Pages))
	}
	if got.Pages[0].PageID != "welcome/4" || got.Pages[0].Route != "/welcome/3" {
		t.Fatalf("special welcome identity was not preserved: %+v", got.Pages[0])
	}
}

func TestCourseMetadataCurrentProductionBaselineIs103Pages(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	current, err := BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Pages) != 103 {
		t.Fatalf("current production catalog pages=%d, want 103", len(catalog.Pages))
	}
	if err := HydrateCatalogSources(catalog, current); err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"zh-CN", "ja-JP"} {
		metadata, err := LoadCourseMetadata(root, locale, catalog)
		if err != nil {
			t.Fatalf("load current %s formal course metadata: %v", locale, err)
		}
		if len(metadata.Pages) != len(catalog.Pages) {
			t.Fatalf("%s formal pages=%d, catalog=%d", locale, len(metadata.Pages), len(catalog.Pages))
		}
	}
}

func TestValidateCourseMetadataSetAndIdentityFailures(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Catalog, map[string][]byte, []byte, *CourseMetadata) []byte
		want string
	}{
		{"missing page", func(_ *Catalog, _ map[string][]byte, glossary []byte, metadata *CourseMetadata) []byte {
			metadata.Pages = metadata.Pages[:len(metadata.Pages)-1]
			return glossary
		}, "missing page"},
		{"extra page", func(_ *Catalog, _ map[string][]byte, glossary []byte, metadata *CourseMetadata) []byte {
			extra := metadata.Pages[0]
			extra.PageID, extra.Route, extra.Description = "extra/1", "/extra/1", courseDescription("extra")
			metadata.Pages = append(metadata.Pages, extra)
			return glossary
		}, "extra page_id"},
		{"duplicate page id", func(_ *Catalog, _ map[string][]byte, glossary []byte, metadata *CourseMetadata) []byte {
			metadata.Pages[1].PageID = metadata.Pages[0].PageID
			return glossary
		}, "duplicate page_id"},
		{"duplicate route", func(catalog *Catalog, _ map[string][]byte, glossary []byte, metadata *CourseMetadata) []byte {
			catalog.Pages[1].Route = catalog.Pages[0].Route
			metadata.Pages[1].Route = metadata.Pages[0].Route
			return glossary
		}, "duplicate route"},
		{"welcome route mismatch", func(_ *Catalog, _ map[string][]byte, glossary []byte, metadata *CourseMetadata) []byte {
			metadata.Pages[0].Route = "/welcome/4"
			return glossary
		}, "route"},
		{"source stale", func(_ *Catalog, _ map[string][]byte, glossary []byte, metadata *CourseMetadata) []byte {
			metadata.Pages[0].SourceSHA256 = strings.Repeat("0", 64)
			return glossary
		}, "source_sha256 is stale"},
		{"target stale", func(_ *Catalog, _ map[string][]byte, glossary []byte, metadata *CourseMetadata) []byte {
			metadata.Pages[0].TargetSHA256 = strings.Repeat("0", 64)
			return glossary
		}, "target_sha256 is stale"},
		{"glossary stale", func(_ *Catalog, _ map[string][]byte, _ []byte, metadata *CourseMetadata) []byte {
			return []byte("changed glossary\n")
		}, "glossary_sha256 is stale"},
		{"generator contract stale", func(_ *Catalog, _ map[string][]byte, glossary []byte, metadata *CourseMetadata) []byte {
			metadata.GeneratorContract = "course-seo-description-v2"
			return glossary
		}, "generator_contract"},
		{"prompt version stale", func(_ *Catalog, _ map[string][]byte, glossary []byte, metadata *CourseMetadata) []byte {
			metadata.Pages[0].Generation.PromptVersion = "course-seo-description-v2"
			return glossary
		}, "prompt_version"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog, targets, glossary, metadata := courseMetadataFixture()
			glossary = test.edit(catalog, targets, glossary, metadata)
			_, err := validateCourseMetadata(marshalCourseMetadata(t, metadata), "test-LOCALE", catalog, targets, glossary)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func TestValidateCourseMetadataDescriptionFailures(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{"empty", " \t ", "empty"},
		{"too short", "Too short.", "minimum"},
		{"multiline", "This description is deliberately long enough, but it contains\na forbidden newline.", "one paragraph"},
		{"html", "This description is deliberately long enough and contains a <strong>forbidden HTML tag</strong>.", "HTML tag"},
		{"backtick code fence", "This description is deliberately long enough and contains ```a forbidden code fence```.", "code fence"},
		{"tilde code fence", "This description is deliberately long enough and contains ~~~a forbidden code fence~~~.", "code fence"},
		{"url", "This description is deliberately long enough and links to https://example.com/forbidden.", "URL"},
		{"bare domain", "This description is deliberately long enough and contains the forbidden URL go.dev/doc.", "URL"},
		{"www domain", "This description is deliberately long enough and contains the forbidden URL www.example.com.", "URL"},
		{"control", "This description is deliberately long enough but contains a forbidden\u0001 control character.", "control"},
		{"too long", strings.Repeat("界", CourseDescriptionMaxRunes+1), "maximum"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog, targets, glossary, metadata := courseMetadataFixture()
			metadata.Pages[0].Description = test.description
			_, err := validateCourseMetadata(marshalCourseMetadata(t, metadata), "test-LOCALE", catalog, targets, glossary)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func TestValidateCourseDescriptionAllowsGoSelectors(t *testing.T) {
	selectors := []string{
		"io.Reader",
		"io.EOF",
		"fmt.Stringer",
		"image.Image",
		"sync.Mutex",
		"color.RGBA",
	}
	for _, selector := range selectors {
		t.Run(selector, func(t *testing.T) {
			description := "This complete course description explains the Go selector " + selector + " in its lesson context."
			if err := validateCourseDescription(description); err != nil {
				t.Fatalf("validateCourseDescription(%q): %v", description, err)
			}
		})
	}
}

func TestValidateCourseMetadataDuplicateDescriptions(t *testing.T) {
	tests := []struct {
		name string
		edit func(*CourseMetadata)
		want string
	}{
		{"exact", func(metadata *CourseMetadata) {
			metadata.Pages[1].Description = metadata.Pages[0].Description
		}, "exact duplicate"},
		{"normalized whitespace and punctuation", func(metadata *CourseMetadata) {
			metadata.Pages[0].Description = "Learn Go methods, interfaces, and their precise behavior in this complete interactive lesson."
			metadata.Pages[1].Description = "Learn Go methods interfaces and their precise behavior in this complete interactive lesson"
		}, "normalized duplicate"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog, targets, glossary, metadata := courseMetadataFixture()
			test.edit(metadata)
			_, err := validateCourseMetadata(marshalCourseMetadata(t, metadata), "test-LOCALE", catalog, targets, glossary)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func TestValidateCourseMetadataStrictJSON(t *testing.T) {
	catalog, targets, glossary, metadata := courseMetadataFixture()
	data := marshalCourseMetadata(t, metadata)
	data = []byte(strings.Replace(string(data), `"pages":`, `"unknown":true,"pages":`, 1))
	if _, err := validateCourseMetadata(data, "test-LOCALE", catalog, targets, glossary); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error=%v, want unknown field", err)
	}
}

func TestLoadCourseMetadataReadsFormalPaths(t *testing.T) {
	root, catalog, _ := writeCourseMetadataLoaderFixture(t)
	if _, err := LoadCourseMetadata(root, "test-LOCALE", catalog); err != nil {
		t.Fatal(err)
	}
}

func TestLoadCourseMetadataRequiresReadyCanonicalStatus(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Status)
		want string
	}{
		{"not ready", func(status *Status) { status.State = "pending" }, "not a ready candidate"},
		{"noncanonical candidate path", func(status *Status) { status.CandidatePath = "locales/test-LOCALE/candidates/other.article" }, "is not canonical"},
		{"stale source hash", func(status *Status) { status.SourceSHA256 = strings.Repeat("0", 64) }, "source hash does not match current source"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, statuses := writeCourseMetadataLoaderFixture(t)
			test.edit(&statuses[0])
			if err := writeStatuses(filepath.Join(root, "locales", "test-LOCALE", "status.tsv"), statuses); err != nil {
				t.Fatal(err)
			}
			_, err := LoadCourseMetadata(root, "test-LOCALE", catalog)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func TestAssembleCourseMetadataDeterministicValidOutput(t *testing.T) {
	root, catalog, _ := writeCourseMetadataLoaderFixture(t)
	descriptions := marshalCourseDescriptions(t, catalog)
	options := CourseMetadataAssemblyOptions{
		Locale: "test-LOCALE", Provider: "codex", Model: "fixture-model", GeneratedAt: "2026-08-26T01:02:03Z", Descriptions: descriptions,
	}
	first, err := AssembleCourseMetadata(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := AssembleCourseMetadata(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("identical assembly inputs produced different bytes")
	}
	if len(first) == 0 || first[len(first)-1] != '\n' {
		t.Fatal("assembled metadata does not have a fixed trailing newline")
	}
	var metadata CourseMetadata
	if err := json.Unmarshal(first, &metadata); err != nil {
		t.Fatal(err)
	}
	for i, page := range catalog.Pages {
		if metadata.Pages[i].PageID != page.ID || metadata.Pages[i].Route != page.Route {
			t.Fatalf("page %d identity=%s %s, want %s %s", i, metadata.Pages[i].PageID, metadata.Pages[i].Route, page.ID, page.Route)
		}
	}
	output := filepath.Join(root, "locales", "test-LOCALE", "course-metadata.json")
	if err := os.WriteFile(output, first, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCourseMetadata(root, "test-LOCALE", catalog); err != nil {
		t.Fatalf("assembled output did not pass the formal loader: %v", err)
	}
}

func TestAssembleCourseMetadataDescriptionSetFailures(t *testing.T) {
	tests := []struct {
		name string
		edit func(*courseDescriptionsFile)
		want string
	}{
		{"missing page", func(input *courseDescriptionsFile) { input.Pages = input.Pages[:len(input.Pages)-1] }, "missing page"},
		{"extra page", func(input *courseDescriptionsFile) {
			input.Pages = append(input.Pages, courseDescriptionEntry{PageID: "extra/1", Description: courseDescription("extra/1")})
		}, "extra page_id"},
		{"duplicate page id", func(input *courseDescriptionsFile) {
			input.Pages = append(input.Pages, input.Pages[0])
		}, "duplicate page_id"},
		{"invalid description", func(input *courseDescriptionsFile) { input.Pages[0].Description = "short" }, "minimum"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, _ := writeCourseMetadataLoaderFixture(t)
			input := courseDescriptionsForCatalog(catalog)
			test.edit(&input)
			data, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			_, err = AssembleCourseMetadata(root, catalog, CourseMetadataAssemblyOptions{
				Locale: "test-LOCALE", Provider: "codex", Model: "fixture-model", GeneratedAt: "2026-08-26T01:02:03Z", Descriptions: data,
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func TestAssembleCourseMetadataRejectsNonReadyOrStaleTarget(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Status)
		want string
	}{
		{"non-ready", func(status *Status) { status.State = "pending" }, "not a ready candidate"},
		{"stale source identity", func(status *Status) { status.SourceSHA256 = strings.Repeat("0", 64) }, "source hash does not match current source"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, statuses := writeCourseMetadataLoaderFixture(t)
			test.edit(&statuses[0])
			if err := writeStatuses(filepath.Join(root, "locales", "test-LOCALE", "status.tsv"), statuses); err != nil {
				t.Fatal(err)
			}
			_, err := AssembleCourseMetadata(root, catalog, CourseMetadataAssemblyOptions{
				Locale: "test-LOCALE", Provider: "codex", Model: "fixture-model", GeneratedAt: "2026-08-26T01:02:03Z", Descriptions: marshalCourseDescriptions(t, catalog),
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func writeCourseMetadataLoaderFixture(t *testing.T) (string, *Catalog, []Status) {
	t.Helper()
	root := t.TempDir()
	catalog, targets, glossary, metadata := courseMetadataFixture()
	localeDir := filepath.Join(root, "locales", "test-LOCALE")
	if err := os.MkdirAll(filepath.Join(localeDir, "candidates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "course-metadata.json"), marshalCourseMetadata(t, metadata), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "glossary.yaml"), glossary, 0644); err != nil {
		t.Fatal(err)
	}
	var statuses []Status
	for _, page := range catalog.Pages {
		path := canonicalCandidatePath("test-LOCALE", page.ID)
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), targets[page.ID], 0644); err != nil {
			t.Fatal(err)
		}
		statuses = append(statuses, Status{UnitID: page.ID, State: "ready", Attempts: 1, SourceSHA256: page.SourceSHA256, CandidatePath: path})
	}
	if err := writeStatuses(filepath.Join(localeDir, "status.tsv"), statuses); err != nil {
		t.Fatal(err)
	}
	return root, catalog, statuses
}

func courseMetadataFixture() (*Catalog, map[string][]byte, []byte, *CourseMetadata) {
	glossary := []byte("locale: test-LOCALE\nmandatory:\n  slides: slides\n")
	welcomeSource := []byte("* Source welcome\n\nComplete source paragraph for the welcome fixture.\n")
	basicsSource := []byte("* Source basics\n\nComplete source paragraph for the basics fixture.\n")
	methodsSource := []byte("* Source methods\n\nComplete source paragraph for the methods fixture.\n")
	pages := []Page{
		{ID: "welcome/4", Article: "welcome.article", Route: "/welcome/3", Source: welcomeSource, SourceSHA256: sum(welcomeSource)},
		{ID: "basics/1", Article: "basics.article", Route: "/basics/1", Source: basicsSource, SourceSHA256: sum(basicsSource)},
		{ID: "methods/1", Article: "methods.article", Route: "/methods/1", Source: methodsSource, SourceSHA256: sum(methodsSource)},
	}
	targets := map[string][]byte{
		"welcome/4": []byte("* Target welcome\n\nComplete target paragraph for the welcome fixture.\n"),
		"basics/1":  []byte("* Target basics\n\nComplete target paragraph for the basics fixture.\n"),
		"methods/1": []byte("* Target methods\n\nComplete target paragraph for the methods fixture.\n"),
	}
	metadata := &CourseMetadata{
		SchemaVersion: CourseMetadataSchemaVersion, Locale: "test-LOCALE", GeneratorContract: CourseMetadataGeneratorContract,
	}
	for _, page := range pages {
		metadata.Pages = append(metadata.Pages, CoursePageMetadata{
			PageID: page.ID, Route: page.Route, Description: courseDescription(page.ID),
			SourceSHA256: page.SourceSHA256, TargetSHA256: sum(targets[page.ID]), GlossarySHA256: sum(glossary),
			Generation: CourseMetadataGeneration{Provider: "fixture", Model: "fixture-model", PromptVersion: CourseMetadataPromptVersion, GeneratedAt: "2026-08-25T12:00:00Z"},
		})
	}
	return &Catalog{Pages: pages}, targets, glossary, metadata
}

func courseDescription(identity string) string {
	return "A complete and distinct course summary grounded in the target lesson for page " + identity + "."
}

func marshalCourseMetadata(t *testing.T, metadata *CourseMetadata) []byte {
	t.Helper()
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func courseDescriptionsForCatalog(catalog *Catalog) courseDescriptionsFile {
	input := courseDescriptionsFile{Pages: make([]courseDescriptionEntry, 0, len(catalog.Pages))}
	for _, page := range catalog.Pages {
		input.Pages = append(input.Pages, courseDescriptionEntry{PageID: page.ID, Description: courseDescription(page.ID)})
	}
	return input
}

func marshalCourseDescriptions(t *testing.T, catalog *Catalog) []byte {
	t.Helper()
	data, err := json.Marshal(courseDescriptionsForCatalog(catalog))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRefreshCourseMetadataRegeneratesOnlyStalePages(t *testing.T) {
	root, catalog, base := writeCourseMetadataRefreshFixture(t, 103)
	staleIDs := []string{catalog.Pages[0].ID, catalog.Pages[51].ID, catalog.Pages[102].ID}
	for _, pageID := range staleIDs {
		base.Pages[pageIndex(t, base, pageID)].SourceSHA256 = strings.Repeat("0", 64)
	}
	writeCourseMetadataFixture(t, root, "test-LOCALE", base)

	data, stale, err := RefreshCourseMetadata(root, catalog, CourseMetadataRefreshOptions{
		Locale: "test-LOCALE", Provider: "new-provider", Model: "new-model", GeneratedAt: "2026-08-27T06:30:00Z",
		Descriptions: marshalCourseRefreshDescriptions(t, staleIDs),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stale, staleIDs) {
		t.Fatalf("stale=%v, want %v", stale, staleIDs)
	}
	var refreshed CourseMetadata
	if err := json.Unmarshal(data, &refreshed); err != nil {
		t.Fatal(err)
	}
	for _, page := range catalog.Pages {
		old := base.Pages[pageIndex(t, base, page.ID)]
		got := refreshed.Pages[pageIndex(t, &refreshed, page.ID)]
		isStale := page.ID == staleIDs[0] || page.ID == staleIDs[1] || page.ID == staleIDs[2]
		if !isStale {
			if got.Description != old.Description || !reflect.DeepEqual(got.Generation, old.Generation) {
				t.Fatalf("non-stale %s was regenerated: got=%+v old=%+v", page.ID, got, old)
			}
			continue
		}
		if got.Description != courseDescription("refresh "+page.ID) || got.SourceSHA256 != page.SourceSHA256 || got.Route != page.Route ||
			got.Generation.Provider != "new-provider" || got.Generation.Model != "new-model" || got.Generation.GeneratedAt != "2026-08-27T06:30:00Z" {
			t.Fatalf("stale %s was not refreshed from current identity: %+v", page.ID, got)
		}
	}
	writeCourseMetadataFixture(t, root, "test-LOCALE", &refreshed)
	if _, err := LoadCourseMetadata(root, "test-LOCALE", catalog); err != nil {
		t.Fatalf("refreshed metadata did not pass strict validation: %v", err)
	}
}

func TestRefreshCourseMetadataDescriptionSetAndGlossaryGuards(t *testing.T) {
	t.Run("missing stale description", func(t *testing.T) {
		root, catalog, base := writeCourseMetadataRefreshFixture(t, 3)
		base.Pages[0].TargetSHA256 = strings.Repeat("0", 64)
		base.Pages[1].SourceSHA256 = strings.Repeat("0", 64)
		writeCourseMetadataFixture(t, root, "test-LOCALE", base)
		_, _, err := RefreshCourseMetadata(root, catalog, CourseMetadataRefreshOptions{
			Locale: "test-LOCALE", Provider: "provider", Model: "model", GeneratedAt: "2026-08-27T06:30:00Z",
			Descriptions: marshalCourseRefreshDescriptions(t, []string{catalog.Pages[0].ID}),
		})
		if err == nil || !strings.Contains(err.Error(), "missing stale page") {
			t.Fatalf("error=%v", err)
		}
	})
	t.Run("non-stale and extra descriptions", func(t *testing.T) {
		root, catalog, base := writeCourseMetadataRefreshFixture(t, 3)
		base.Pages[0].SourceSHA256 = strings.Repeat("0", 64)
		writeCourseMetadataFixture(t, root, "test-LOCALE", base)
		for _, input := range [][]byte{
			marshalCourseRefreshDescriptions(t, []string{catalog.Pages[0].ID, catalog.Pages[1].ID}),
			[]byte(`{"pages":[{"page_id":"extra/1","description":"A complete and distinct course summary grounded in the target lesson for extra/1."}]}`),
		} {
			_, _, err := RefreshCourseMetadata(root, catalog, CourseMetadataRefreshOptions{Locale: "test-LOCALE", Provider: "provider", Model: "model", GeneratedAt: "2026-08-27T06:30:00Z", Descriptions: input})
			if err == nil || (!strings.Contains(err.Error(), "non-stale") && !strings.Contains(err.Error(), "extra page_id")) {
				t.Fatalf("error=%v", err)
			}
		}
	})
	t.Run("glossary change makes every page stale", func(t *testing.T) {
		root, catalog, _ := writeCourseMetadataRefreshFixture(t, 3)
		path := filepath.Join(root, "locales", "test-LOCALE", "glossary.yaml")
		glossary, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(glossary, []byte("preferred:\n  tour: course\n")...), 0644); err != nil {
			t.Fatal(err)
		}
		ids := []string{catalog.Pages[0].ID, catalog.Pages[1].ID, catalog.Pages[2].ID}
		data, stale, err := RefreshCourseMetadata(root, catalog, CourseMetadataRefreshOptions{Locale: "test-LOCALE", Provider: "provider", Model: "model", GeneratedAt: "2026-08-27T06:30:00Z", Descriptions: marshalCourseRefreshDescriptions(t, ids)})
		if err != nil || !reflect.DeepEqual(stale, ids) {
			t.Fatalf("stale=%v err=%v", stale, err)
		}
		var refreshed CourseMetadata
		if err := json.Unmarshal(data, &refreshed); err != nil {
			t.Fatal(err)
		}
		writeCourseMetadataFixture(t, root, "test-LOCALE", &refreshed)
		if _, err := LoadCourseMetadata(root, "test-LOCALE", catalog); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRefreshCourseMetadataFailsClosedOnCatalogPageSetChange(t *testing.T) {
	root, catalog, _ := writeCourseMetadataRefreshFixture(t, 3)
	catalog.Pages = catalog.Pages[:2]
	_, _, err := RefreshCourseMetadata(root, catalog, CourseMetadataRefreshOptions{
		Locale: "test-LOCALE", Provider: "provider", Model: "model", GeneratedAt: "2026-08-27T06:30:00Z", Descriptions: []byte(`{"pages":[]}`),
	})
	if err == nil || !strings.Contains(err.Error(), "base has extra page_id") {
		t.Fatalf("error=%v", err)
	}
}

func writeCourseMetadataRefreshFixture(t *testing.T, count int) (string, *Catalog, *CourseMetadata) {
	t.Helper()
	root := t.TempDir()
	locale := "test-LOCALE"
	glossary := []byte("locale: test-LOCALE\nmandatory:\n  slides: slides\n")
	catalog := &Catalog{Pages: make([]Page, 0, count)}
	metadata := &CourseMetadata{SchemaVersion: CourseMetadataSchemaVersion, Locale: locale, GeneratorContract: CourseMetadataGeneratorContract, Pages: make([]CoursePageMetadata, 0, count)}
	localeDir := filepath.Join(root, "locales", locale)
	if err := os.MkdirAll(filepath.Join(localeDir, "candidates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "glossary.yaml"), glossary, 0644); err != nil {
		t.Fatal(err)
	}
	statuses := make([]Status, 0, count)
	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("lesson/%03d", i)
		source := []byte(fmt.Sprintf("* Source %03d\n\nComplete source paragraph for lesson %03d.\n", i, i))
		target := []byte(fmt.Sprintf("* Target %03d\n\nComplete target paragraph for lesson %03d.\n", i, i))
		page := Page{ID: id, Article: "lesson.article", Route: "/" + id, Source: source, SourceSHA256: sum(source)}
		catalog.Pages = append(catalog.Pages, page)
		path := canonicalCandidatePath(locale, id)
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), target, 0644); err != nil {
			t.Fatal(err)
		}
		statuses = append(statuses, Status{UnitID: id, State: "ready", Attempts: 1, SourceSHA256: page.SourceSHA256, CandidatePath: path})
		metadata.Pages = append(metadata.Pages, CoursePageMetadata{PageID: id, Route: page.Route, Description: courseDescription(id), SourceSHA256: page.SourceSHA256, TargetSHA256: sum(target), GlossarySHA256: sum(glossary), Generation: CourseMetadataGeneration{Provider: "fixture", Model: "fixture-model", PromptVersion: CourseMetadataPromptVersion, GeneratedAt: "2026-08-25T12:00:00Z"}})
	}
	if err := writeStatuses(filepath.Join(localeDir, "status.tsv"), statuses); err != nil {
		t.Fatal(err)
	}
	writeCourseMetadataFixture(t, root, locale, metadata)
	return root, catalog, metadata
}

func writeCourseMetadataFixture(t *testing.T, root, locale string, metadata *CourseMetadata) {
	t.Helper()
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "locales", locale, "course-metadata.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func marshalCourseRefreshDescriptions(t *testing.T, ids []string) []byte {
	t.Helper()
	input := courseDescriptionsFile{Pages: make([]courseDescriptionEntry, 0, len(ids))}
	for _, id := range ids {
		input.Pages = append(input.Pages, courseDescriptionEntry{PageID: id, Description: courseDescription("refresh " + id)})
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func pageIndex(t *testing.T, metadata *CourseMetadata, pageID string) int {
	t.Helper()
	for i, entry := range metadata.Pages {
		if entry.PageID == pageID {
			return i
		}
	}
	t.Fatalf("metadata has no page %s", pageID)
	return -1
}
