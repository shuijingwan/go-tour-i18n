package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildLocaleProjectionReplacesMultipleSectionsAndArticles(t *testing.T) {
	fixture := newProjectionFixture(t)
	output := filepath.Join(t.TempDir(), "projection")
	projection, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", output)
	if err != nil {
		t.Fatal(err)
	}
	if projection.UnitCount != 4 || projection.PageCount != 3 || projection.ExampleCount != 1 || projection.ArticleCount != 2 || projection.Ready != 4 || projection.Pending != 0 || projection.Blocked != 0 {
		t.Fatalf("projection counts = %+v", projection)
	}
	alpha, err := os.ReadFile(filepath.Join(projection.ContentDir, "tour", "alpha.article"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"甲课程", "甲课程说明"} {
		if !bytes.Contains(alpha, []byte(want)) {
			t.Errorf("projected alpha.article missing localized metadata %q", want)
		}
	}
	for _, want := range []string{"* 第一页", "第一段译文。", "* 第二页", "包含 `code` 的第二段译文。", ".play alpha/one.go"} {
		if !bytes.Contains(alpha, []byte(want)) {
			t.Errorf("projected alpha.article missing %q", want)
		}
	}
	for _, fallback := range []string{"English first paragraph.", "English second paragraph."} {
		if bytes.Contains(alpha, []byte(fallback)) {
			t.Errorf("projected alpha.article retained English fallback %q", fallback)
		}
	}
	beta, err := os.ReadFile(filepath.Join(projection.ContentDir, "tour", "beta.article"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(beta, []byte("第三段译文。")) || bytes.Contains(beta, []byte("English third paragraph.")) {
		t.Fatalf("beta projection did not replace its candidate:\n%s", beta)
	}
	wantProgram := fixture.candidates["example:alpha/one.go"]
	gotProgram, err := os.ReadFile(filepath.Join(projection.ContentDir, "tour", "alpha", "one.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotProgram, wantProgram) {
		t.Fatal("eligible Example candidate was not projected")
	}
	upstreamStaticExample, _ := os.ReadFile(filepath.Join(fixture.root, "_content", "tour", "alpha", "static.go"))
	projectedStaticExample, _ := os.ReadFile(filepath.Join(projection.ContentDir, "tour", "alpha", "static.go"))
	if !bytes.Equal(upstreamStaticExample, projectedStaticExample) {
		t.Fatal("non-eligible Example changed during projection")
	}
	static, err := os.ReadFile(filepath.Join(projection.ContentDir, "tour", "static.txt"))
	if err != nil || string(static) != "shared static asset\n" {
		t.Fatalf("shared static asset was not preserved: data=%q err=%v", static, err)
	}
}

func TestBuildLocaleProjectionRejectsIncompleteArticleMetadata(t *testing.T) {
	tests := []struct {
		name string
		edit func([]ArticleMetadata) []ArticleMetadata
		want string
	}{
		{"missing", func(entries []ArticleMetadata) []ArticleMetadata { return entries[:1] }, "missing article"},
		{"extra", func(entries []ArticleMetadata) []ArticleMetadata {
			return append(entries, ArticleMetadata{Article: "extra.article", Title: "额外", Subtitle: "额外说明"})
		}, "extra article"},
		{"duplicate", func(entries []ArticleMetadata) []ArticleMetadata { return append(entries, entries[0]) }, "duplicate article"},
		{"empty title", func(entries []ArticleMetadata) []ArticleMetadata { entries[0].Title = " "; return entries }, "empty title"},
		{"empty subtitle", func(entries []ArticleMetadata) []ArticleMetadata { entries[0].Subtitle = " "; return entries }, "empty subtitle"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newProjectionFixture(t)
			fixture.writeMetadata(t, test.edit(fixture.metadata))
			_, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestBuildLocaleProjectionRejectsMissingOrUnparseableArticleMetadata(t *testing.T) {
	for _, test := range []struct {
		name    string
		prepare func(t *testing.T, fixture *projectionFixture)
	}{
		{"missing", func(t *testing.T, fixture *projectionFixture) {
			t.Helper()
			if err := os.Remove(fixture.metadataPath()); err != nil {
				t.Fatal(err)
			}
		}},
		{"unparseable", func(t *testing.T, fixture *projectionFixture) {
			t.Helper()
			writeFixtureFile(t, fixture.metadataPath(), []byte("not JSON"))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newProjectionFixture(t)
			test.prepare(t, fixture)
			_, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
			if err == nil || !strings.Contains(err.Error(), "article metadata") {
				t.Fatalf("error = %v, want article metadata rejection", err)
			}
		})
	}
}

func TestBuildLocaleProjectionRejectsNonReadyPages(t *testing.T) {
	for _, state := range []string{"pending", "blocked"} {
		t.Run(state, func(t *testing.T) {
			fixture := newProjectionFixture(t)
			fixture.statuses[1].State = state
			fixture.statuses[1].CandidatePath = ""
			if state == "pending" {
				fixture.statuses[1].Attempts = 0
			}
			fixture.writeStatuses(t)
			_, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
			if err == nil || !strings.Contains(err.Error(), "workflow translation units to be ready") {
				t.Fatalf("error = %v, want non-ready rejection", err)
			}
		})
	}
}

func TestBuildLocaleProjectionRequiresReadyValidExample(t *testing.T) {
	t.Run("pending", func(t *testing.T) {
		fixture := newProjectionFixture(t)
		status := &fixture.statuses[len(fixture.statuses)-1]
		status.State, status.Attempts, status.CandidatePath = "pending", 0, ""
		fixture.writeStatuses(t)
		_, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
		if err == nil || !strings.Contains(err.Error(), "example:alpha/one.go=pending") {
			t.Fatalf("pending Example error=%v", err)
		}
	})
	t.Run("missing candidate", func(t *testing.T) {
		fixture := newProjectionFixture(t)
		status := fixture.statuses[len(fixture.statuses)-1]
		if err := os.Remove(filepath.Join(fixture.root, filepath.FromSlash(status.CandidatePath))); err != nil {
			t.Fatal(err)
		}
		_, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
		if err == nil || !strings.Contains(err.Error(), "read canonical candidate") {
			t.Fatalf("missing Example candidate error=%v", err)
		}
	})
	t.Run("invalid candidate", func(t *testing.T) {
		fixture := newProjectionFixture(t)
		status := fixture.statuses[len(fixture.statuses)-1]
		if err := os.WriteFile(filepath.Join(fixture.root, filepath.FromSlash(status.CandidatePath)), []byte("package changed\n"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
		if err == nil || !strings.Contains(err.Error(), "candidate validation") {
			t.Fatalf("invalid Example candidate error=%v", err)
		}
	})
}

func TestBuildLocaleProjectionRejectsMissingCandidateWithoutFallback(t *testing.T) {
	fixture := newProjectionFixture(t)
	missing := filepath.Join(fixture.root, filepath.FromSlash(fixture.statuses[0].CandidatePath))
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "projection")
	_, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", output)
	if err == nil || !strings.Contains(err.Error(), "read canonical candidate") {
		t.Fatalf("error = %v, want missing canonical candidate rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(output, "_content")); !os.IsNotExist(statErr) {
		t.Fatalf("projection content exists after rejected candidate: %v", statErr)
	}
}

func TestBuildLocaleProjectionRejectsCatalogStatusSetMismatch(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		fixture := newProjectionFixture(t)
		fixture.statuses = fixture.statuses[:2]
		fixture.writeStatuses(t)
		if _, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection")); err == nil {
			t.Fatal("missing status page was accepted")
		}
	})
	t.Run("extra", func(t *testing.T) {
		fixture := newProjectionFixture(t)
		fixture.statuses = append(fixture.statuses, Status{UnitID: "extra/1", State: "pending", SourceSHA256: strings.Repeat("a", 64)})
		fixture.writeStatuses(t)
		if _, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection")); err == nil {
			t.Fatal("extra status page was accepted")
		}
	})
}

func TestBuildLocaleProjectionRequiresCanonicalCandidatePath(t *testing.T) {
	fixture := newProjectionFixture(t)
	oldPath := filepath.Join(fixture.root, filepath.FromSlash(fixture.statuses[0].CandidatePath))
	noncanonical := "locales/zh-CN/candidates/alternate.article"
	data, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture.root, filepath.FromSlash(noncanonical)), data, 0644); err != nil {
		t.Fatal(err)
	}
	fixture.statuses[0].CandidatePath = noncanonical
	fixture.writeStatuses(t)
	_, err = BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
	if err == nil || !strings.Contains(err.Error(), "is not canonical") {
		t.Fatalf("error = %v, want canonical path rejection", err)
	}
}

func TestBuildLocaleProjectionRejectsResidualProtectedToken(t *testing.T) {
	fixture := newProjectionFixture(t)
	pageID := fixture.statuses[2].UnitID
	candidate := append([]byte(nil), fixture.candidates[pageID]...)
	candidate = bytes.Replace(candidate, []byte("第三段译文。"), []byte("第三段译文。⟪GTI18N_deadbeef_000001⟫"), 1)
	if err := os.WriteFile(filepath.Join(fixture.root, filepath.FromSlash(fixture.statuses[2].CandidatePath)), candidate, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
	if err == nil || !strings.Contains(err.Error(), "contains a protected token") {
		t.Fatalf("error = %v, want protected token rejection", err)
	}
}

func TestBuildLocaleProjectionRequiresAllEligibleExamplesReady(t *testing.T) {
	fixture := newProjectionFixture(t)
	status := &fixture.statuses[len(fixture.statuses)-1]
	status.State, status.Attempts, status.CandidatePath, status.UpdatedAt, status.Note = "pending", 0, "", "", ""
	fixture.writeStatuses(t)
	_, err := BuildLocaleProjection(fixture.root, fixture.catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
	if err == nil || !strings.Contains(err.Error(), "all 4 workflow translation units") || !strings.Contains(err.Error(), "example:alpha/one.go=pending") {
		t.Fatalf("incomplete workflow projection error=%v", err)
	}
}

type projectionFixture struct {
	root       string
	catalog    *Catalog
	statuses   []Status
	candidates map[string][]byte
	metadata   []ArticleMetadata
}

func newProjectionFixture(t *testing.T) *projectionFixture {
	t.Helper()
	root := t.TempDir()
	alphaOne := []byte("* One\n\nEnglish first paragraph.\n.play alpha/one.go\n")
	alphaTwo := []byte("* Two\n\nEnglish second paragraph with `code`.\n")
	betaOne := []byte("* Three\n\nEnglish third paragraph.\n")
	writeFixtureFile(t, filepath.Join(root, "_content", "tour", "alpha.article"), appendArticle("Alpha", alphaOne, alphaTwo))
	writeFixtureFile(t, filepath.Join(root, "_content", "tour", "beta.article"), appendArticle("Beta", betaOne))
	exampleSource := []byte("package main\n\n// Print a greeting.\nfunc main() {}\n")
	exampleCandidate := []byte("package main\n\n// 打印问候语。\nfunc main() {}\n")
	staticExample := []byte("package main\n\nfunc helper() {}\n")
	writeFixtureFile(t, filepath.Join(root, "_content", "tour", "alpha", "one.go"), exampleSource)
	writeFixtureFile(t, filepath.Join(root, "_content", "tour", "alpha", "static.go"), staticExample)
	writeFixtureFile(t, filepath.Join(root, "_content", "tour", "static.txt"), []byte("shared static asset\n"))
	writeFixtureFile(t, filepath.Join(root, "locales", "zh-CN", "locale.json"), []byte(`{"locale":"zh-CN","language_name":"简体中文","english_name":"Simplified Chinese","html_lang":"zh-CN","phase":"scaffold","translation_unit":"present.Section","default_language":true}`))
	writeFixtureFile(t, filepath.Join(root, "locales", "zh-CN", "glossary.yaml"), []byte("mandatory:\n  slides: 幻灯片\n"))

	pages := []Page{
		{ID: "alpha/1", Article: "alpha.article", SectionNumber: 1, Route: "/alpha/1", SourceTitle: "One", SourceSHA256: sum(alphaOne), PlayCount: 1, Source: alphaOne},
		{ID: "alpha/2", Article: "alpha.article", SectionNumber: 2, Route: "/alpha/2", SourceTitle: "Two", SourceSHA256: sum(alphaTwo), Source: alphaTwo},
		{ID: "beta/1", Article: "beta.article", SectionNumber: 1, Route: "/beta/1", SourceTitle: "Three", SourceSHA256: sum(betaOne), Source: betaOne},
	}
	candidates := map[string][]byte{
		"alpha/1": []byte("* 第一页\n\n第一段译文。\n.play alpha/one.go\n"),
		"alpha/2": []byte("* 第二页\n\n包含 `code` 的第二段译文。\n"),
		"beta/1":  []byte("* 第三页\n\n第三段译文。\n"),
	}
	fixture := &projectionFixture{
		root: root, catalog: &Catalog{Pages: pages, Examples: []Example{
			{ID: "example:alpha/one.go", SourcePath: "_content/tour/alpha/one.go", Source: exampleSource, SourceSHA256: sum(exampleSource)},
			{ID: "example:alpha/static.go", SourcePath: "_content/tour/alpha/static.go", Source: staticExample, SourceSHA256: sum(staticExample)},
		}}, candidates: candidates,
		metadata: []ArticleMetadata{
			{Article: "alpha.article", Title: "甲课程", Subtitle: "甲课程说明"},
			{Article: "beta.article", Title: "乙课程", Subtitle: "乙课程说明"},
		},
	}
	for _, page := range pages {
		path := canonicalCandidatePath("zh-CN", page.ID)
		writeFixtureFile(t, filepath.Join(root, filepath.FromSlash(path)), candidates[page.ID])
		fixture.statuses = append(fixture.statuses, Status{UnitID: page.ID, State: "ready", Attempts: 1, SourceSHA256: page.SourceSHA256, CandidatePath: path})
	}
	examplePath, err := canonicalTranslationUnitCandidatePath("zh-CN", &TranslationUnit{ID: "example:alpha/one.go", Kind: UnitKindExample})
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(root, filepath.FromSlash(examplePath)), exampleCandidate)
	fixture.candidates["example:alpha/one.go"] = exampleCandidate
	fixture.statuses = append(fixture.statuses, Status{UnitID: "example:alpha/one.go", State: "ready", Attempts: 1, SourceSHA256: sum(exampleSource), CandidatePath: examplePath})
	fixture.writeStatuses(t)
	fixture.writeMetadata(t, fixture.metadata)
	return fixture
}

func (f *projectionFixture) metadataPath() string {
	return filepath.Join(f.root, "locales", "zh-CN", "article-metadata.json")
}

func (f *projectionFixture) writeMetadata(t *testing.T, entries []ArticleMetadata) {
	t.Helper()
	data, err := json.Marshal(articleMetadataFile{Locale: "zh-CN", Articles: entries})
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, f.metadataPath(), data)
}

func (f *projectionFixture) writeStatuses(t *testing.T) {
	t.Helper()
	if err := writeStatuses(filepath.Join(f.root, "locales", "zh-CN", "status.tsv"), f.statuses); err != nil {
		t.Fatal(err)
	}
}

func appendArticle(title string, sections ...[]byte) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "%s\nFixture subtitle\n\nFixture Author\nhttps://example.test\n\n", title)
	for i, section := range sections {
		if i > 0 {
			out.WriteByte('\n')
		}
		out.Write(section)
	}
	return out.Bytes()
}

func writeFixtureFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
