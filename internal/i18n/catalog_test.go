package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCatalogBaseline(t *testing.T) {
	root := repoRoot(t)
	c, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Pages) != 103 || len(c.Conditional) != 2 {
		t.Fatalf("pages=%d conditional=%d", len(c.Pages), len(c.Conditional))
	}
	plays, images := 0, 0
	perArticle := map[string][]int{}
	for _, p := range c.Pages {
		plays += p.PlayCount
		images += p.ImageCount
		perArticle[p.Article] = append(perArticle[p.Article], p.SectionNumber)
		if err := parseSinglePage(root, p.Article, p.Source); err != nil {
			t.Fatalf("%s: %v", p.ID, err)
		}
		if bytes.Contains(p.Source, []byte("#appengine:")) {
			t.Fatalf("%s: standalone source contains condition marker", p.ID)
		}
		if sum(p.Source) != p.SourceSHA256 {
			t.Fatalf("%s: unstable hash", p.ID)
		}
	}
	if plays != 93 || images != 1 {
		t.Fatalf("play/image=%d/%d", plays, images)
	}
	for article, numbers := range perArticle {
		for i, n := range numbers {
			if n != i+1 {
				t.Fatalf("%s section numbers=%v", article, numbers)
			}
		}
	}
	wantOrder := []string{"welcome.article", "basics.article", "flowcontrol.article", "moretypes.article", "methods.article", "generics.article", "concurrency.article"}
	var gotOrder []string
	for _, p := range c.Pages {
		if len(gotOrder) == 0 || gotOrder[len(gotOrder)-1] != p.Article {
			gotOrder = append(gotOrder, p.Article)
		}
	}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("article order=%v", gotOrder)
	}
	if err := CheckCatalogFiles(root, c); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogGenerationDeterministic(t *testing.T) {
	c, err := BuildCatalog(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	p1, c1, err := CatalogBytes(c)
	if err != nil {
		t.Fatal(err)
	}
	p2, c2, err := CatalogBytes(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p1, p2) || !bytes.Equal(c1, c2) {
		t.Fatal("catalog output is not deterministic")
	}
	e1, err := ExampleCatalogBytes(c)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "data"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "tour-pages.tsv"), p1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "tour-conditional-pages.tsv"), c1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "tour-examples.tsv"), e1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckCatalogFiles(dir, c); err != nil {
		t.Fatal(err)
	}
	p1[10] ^= 1
	if err := os.WriteFile(filepath.Join(dir, "data", "tour-pages.tsv"), p1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckCatalogFiles(dir, c); err == nil {
		t.Fatal("modified catalog was accepted")
	}
}

func TestSplitConditionalPages(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "_content", "tour", "welcome.article"))
	if err != nil {
		t.Fatal(err)
	}
	pages, conditional, err := splitArticle(b, "welcome.article")
	if err != nil {
		t.Fatal(err)
	}
	if len(conditional) != 2 {
		t.Fatalf("conditional=%d", len(conditional))
	}
	if len(pages) == 0 || !bytes.Contains(pages[0], []byte("your computer.")) {
		t.Fatal("welcome/1 does not contain the standalone computer branch")
	}
	if bytes.Contains(pages[0], []byte("#appengine:")) || bytes.Contains(pages[0], []byte("a remote server.")) {
		t.Fatalf("welcome/1 contains appengine content:\n%s", pages[0])
	}
	if pageTitle(conditional[0]) != "Go offline (optional)" || pageTitle(conditional[1]) != "The Go Playground" {
		t.Fatalf("conditional titles=%q/%q", pageTitle(conditional[0]), pageTitle(conditional[1]))
	}
}

func TestStandaloneConditionalProjection(t *testing.T) {
	input := []byte("* Root\n\n#appengine: App Engine only.\nFallback.\n\n#appengine: [[/install][installing Go]]\nVisible.\n")
	want := []byte("* Root\n\nFallback.\n\nVisible.\n")
	if got := projectStandaloneConditionalContent(input, "appengine"); !bytes.Equal(got, want) {
		t.Fatalf("standalone projection:\ngot:\n%s\nwant:\n%s", got, want)
	}
	plain := []byte("* Root\n\nOrdinary content.\n")
	if got := projectStandaloneConditionalContent(plain, "appengine"); !bytes.Equal(got, plain) {
		t.Fatalf("unconditional content changed:\ngot:\n%s\nwant:\n%s", got, plain)
	}
}

func TestStandaloneConditionalProjectionIsRecognizedForCatalogMigration(t *testing.T) {
	oldSource := []byte("* Root\n\n#appengine: [[/install][installing Go]]\nFallback.\n")
	newSource := []byte("* Root\n\nFallback.\n")
	old := &Catalog{Pages: []Page{{ID: "example/1", Article: "example.article", SectionNumber: 1, Route: "/example/1", SourceTitle: "Root", SourceSHA256: sum(oldSource), Source: oldSource}}}
	next := &Catalog{Pages: []Page{{Article: "example.article", SectionNumber: 1, Route: "/example/1", SourceTitle: "Root", SourceSHA256: sum(newSource), Source: newSource}}}
	report, err := PreviewCatalog(old, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 1 || report.Changes[0].Kind != ContentChanged || !report.SafeForCatalogWrite() {
		t.Fatalf("projection report = %+v", report)
	}
	reconciled, err := ReconcileCatalog(old, next, report)
	if err != nil {
		t.Fatal(err)
	}
	if len(reconciled.Pages) != 1 || reconciled.Pages[0].ID != "example/1" || reconciled.Pages[0].SourceSHA256 != sum(newSource) {
		t.Fatalf("reconciled catalog = %+v", reconciled.Pages)
	}
}

func TestStandaloneProjectionCoversNonWelcomeArticles(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("concurrency/11")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "_content", "tour", "concurrency.article"))
	if err != nil {
		t.Fatal(err)
	}
	start := bytes.Index(raw, []byte("* Where to Go from here...\n"))
	if start < 0 {
		t.Fatal("concurrency/11 source section not found")
	}
	want := raw[start:]
	for _, line := range [][]byte{
		[]byte("#appengine: You can get started by\n"),
		[]byte("#appengine: [[/doc/install/][installing Go]].\n"),
		[]byte("#appengine: Once you have Go installed, the\n"),
		[]byte("#appengine: continue.\n"),
	} {
		want = bytes.ReplaceAll(want, line, nil)
	}
	if !bytes.Equal(page.Source, want) {
		t.Fatalf("concurrency/11 standalone projection:\ngot:\n%s\nwant:\n%s", page.Source, want)
	}
	for _, forbidden := range [][]byte{[]byte("#appengine:"), []byte("installing Go"), []byte("Once you have Go installed"), []byte("continue.")} {
		if bytes.Contains(page.Source, forbidden) {
			t.Fatalf("concurrency/11 source retained %q:\n%s", forbidden, page.Source)
		}
	}
	if !bytes.Contains(page.Source, []byte("The\n[[/doc/][Go Documentation]] is a great place to\nstart.")) {
		t.Fatalf("concurrency/11 fallback content missing:\n%s", page.Source)
	}
	for _, id := range []string{"concurrency/11", "flowcontrol/10"} {
		p, err := catalog.Page(id)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(p.Source, []byte("#appengine:")) {
			t.Fatalf("%s source retained condition marker", id)
		}
	}
	candidate := bytes.ReplaceAll(page.Source, []byte("[slides]"), []byte("[页面]"))
	if err := ValidateCandidate(root, catalog, "concurrency/11", candidate); err != nil {
		t.Fatalf("projected concurrency/11 candidate rejected: %v", err)
	}
}

func TestWelcomePersistentIDAndBaselineShape(t *testing.T) {
	catalog, err := BuildCatalog(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		id, route, title, hash string
	}{
		{"welcome/1", "/welcome/1", "Hello, 世界", "1f581133d7fa40e6490418c6789a60a2f5e1de26c9c86d7eb6120cb58b145857"},
		{"welcome/2", "/welcome/2", "Go local", "c1c8c34dfe10b11b5cc72e1daae4336290d613bbbe5445012be949ab904fd3fa"},
		{"welcome/4", "/welcome/3", "Go offline (optional)", "bb517d4b577fdb446fd029c2998d2426663b94a68f4b4e494c59af421508f683"},
		{"welcome/5", "/welcome/4", "The Go Playground", "19e6d7da57ca1d191c485754f3dd4ac87775c651b8c0ae8e05c03cbd9b225897"},
		{"welcome/3", "/welcome/5", "Congratulations", "9a6983b2e50b2fa78ff4f65683210714aea1c1575418e36796c156950ec6330d"},
	}
	for section, expected := range want {
		page, err := catalog.Page(expected.id)
		if err != nil {
			t.Fatal(err)
		}
		if page.Route != expected.route || page.SectionNumber != section+1 || page.SourceTitle != expected.title || page.SourceSHA256 != expected.hash {
			t.Fatalf("%s projection = %+v", expected.id, page)
		}
		if bytes.Contains(page.Source, []byte("#appengine:")) {
			t.Fatalf("%s projected source contains condition marker", expected.id)
		}
		if expected.id == "welcome/1" && (bytes.Contains(page.Source, []byte("your computer.")) || !bytes.Contains(page.Source, []byte("a remote server."))) {
			t.Fatal("welcome/1 did not select only the remote execution branch")
		}
		if err := parseSinglePage(repoRoot(t), page.Article, page.Source); err != nil {
			t.Fatalf("%s projected source: %v", expected.id, err)
		}
		if expected.id == "welcome/4" && !bytes.Contains(page.Source, []byte("go install golang.org/x/website/tour@latest")) {
			t.Fatal("welcome/4 projected source is incomplete")
		}
		if expected.id == "welcome/5" && !bytes.Contains(page.Source, []byte("The playground uses the latest stable release of Go.")) {
			t.Fatal("welcome/5 projected source is incomplete")
		}
		if expected.id == "welcome/4" || expected.id == "welcome/5" {
			candidate := page.Source
			if expected.id == "welcome/5" {
				candidate = bytes.Replace(candidate, []byte("[Go Playground]"), []byte("[Go 语言演练场]"), 1)
			}
			if err := ValidateCandidate(repoRoot(t), catalog, expected.id, candidate); err != nil {
				t.Fatalf("%s projected source is not candidate-valid: %v", expected.id, err)
			}
		}
	}
	if len(catalog.Pages) != 103 || len(catalog.Conditional) != 2 {
		t.Fatalf("pages=%d conditional=%d", len(catalog.Pages), len(catalog.Conditional))
	}
	if _, err := catalog.Page("welcome/appengine/1"); err == nil {
		t.Fatal("unprojected conditional identifier was accepted")
	}
}
