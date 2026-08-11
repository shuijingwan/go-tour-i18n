package i18n

import (
	"bytes"
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
	if projection.PageCount != 3 || projection.ArticleCount != 2 || projection.Ready != 3 || projection.Pending != 0 || projection.Blocked != 0 {
		t.Fatalf("projection counts = %+v", projection)
	}
	alpha, err := os.ReadFile(filepath.Join(projection.ContentDir, "tour", "alpha.article"))
	if err != nil {
		t.Fatal(err)
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
	wantProgram, err := os.ReadFile(filepath.Join(fixture.root, "_content", "tour", "alpha", "one.go"))
	if err != nil {
		t.Fatal(err)
	}
	gotProgram, err := os.ReadFile(filepath.Join(projection.ContentDir, "tour", "alpha", "one.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotProgram, wantProgram) {
		t.Fatal("copied program source changed")
	}
	static, err := os.ReadFile(filepath.Join(projection.ContentDir, "tour", "static.txt"))
	if err != nil || string(static) != "shared static asset\n" {
		t.Fatalf("shared static asset was not preserved: data=%q err=%v", static, err)
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
			if err == nil || !strings.Contains(err.Error(), "complete projection requires every catalog page to be ready") {
				t.Fatalf("error = %v, want non-ready rejection", err)
			}
		})
	}
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
		fixture.statuses = append(fixture.statuses, Status{PageID: "extra/1", State: "pending", SourceSHA256: strings.Repeat("a", 64)})
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
	pageID := fixture.statuses[2].PageID
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

func TestBuildLocaleProjectionPreservesWelcomePublicationSemantics(t *testing.T) {
	root := repoRoot(t)
	current, err := BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := HydrateCatalogSources(catalog, current); err != nil {
		t.Fatal(err)
	}
	projection, err := BuildLocaleProjection(root, catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
	if err != nil {
		t.Fatal(err)
	}
	welcome, err := os.ReadFile(filepath.Join(projection.ContentDir, "tour", "welcome.article"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(welcome, []byte("#appengine:")) {
		t.Fatal("welcome projection retained appengine prefixes")
	}
	sections, _, err := splitArticle(welcome, "welcome.article")
	if err != nil {
		t.Fatal(err)
	}
	for _, pageID := range []string{"welcome/1", "welcome/4", "welcome/5"} {
		page, err := catalog.Page(pageID)
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(canonicalCandidatePath("zh-CN", pageID))))
		if err != nil {
			t.Fatal(err)
		}
		if !sameProjectedSection(sections[page.SectionNumber-1], candidate) {
			t.Errorf("%s does not match its canonical candidate", pageID)
		}
	}
	if !bytes.Contains(sections[0], []byte("在远程服务器上编译并运行该程序")) || bytes.Contains(sections[0], []byte("在你的电脑上编译并运行该程序")) {
		t.Fatal("welcome/1 did not retain the translated remote execution branch")
	}
}

type projectionFixture struct {
	root       string
	catalog    *Catalog
	statuses   []Status
	candidates map[string][]byte
}

func newProjectionFixture(t *testing.T) *projectionFixture {
	t.Helper()
	root := t.TempDir()
	alphaOne := []byte("* One\n\nEnglish first paragraph.\n.play alpha/one.go\n")
	alphaTwo := []byte("* Two\n\nEnglish second paragraph with `code`.\n")
	betaOne := []byte("* Three\n\nEnglish third paragraph.\n")
	writeFixtureFile(t, filepath.Join(root, "_content", "tour", "alpha.article"), appendArticle("Alpha", alphaOne, alphaTwo))
	writeFixtureFile(t, filepath.Join(root, "_content", "tour", "beta.article"), appendArticle("Beta", betaOne))
	writeFixtureFile(t, filepath.Join(root, "_content", "tour", "alpha", "one.go"), []byte("package main\n"))
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
	fixture := &projectionFixture{root: root, catalog: &Catalog{Pages: pages}, candidates: candidates}
	for _, page := range pages {
		path := canonicalCandidatePath("zh-CN", page.ID)
		writeFixtureFile(t, filepath.Join(root, filepath.FromSlash(path)), candidates[page.ID])
		fixture.statuses = append(fixture.statuses, Status{PageID: page.ID, State: "ready", Attempts: 1, SourceSHA256: page.SourceSHA256, CandidatePath: path})
	}
	fixture.writeStatuses(t)
	return fixture
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
