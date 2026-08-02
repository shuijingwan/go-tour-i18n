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
	if len(c.Pages) != 101 || len(c.Conditional) != 2 {
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
		if sum(p.Source) != p.SourceSHA256 {
			t.Fatalf("%s: unstable hash", p.ID)
		}
	}
	if plays != 92 || images != 1 {
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

func TestWelcomePersistentIDAndBaselineShape(t *testing.T) {
	catalog, err := BuildCatalog(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("welcome/1")
	if err != nil {
		t.Fatal(err)
	}
	if page.Route != "/welcome/1" || page.SectionNumber != 1 {
		t.Fatalf("welcome/1 identity changed: %+v", page)
	}
	if len(catalog.Pages) != 101 || len(catalog.Conditional) != 2 {
		t.Fatalf("pages=%d conditional=%d", len(catalog.Pages), len(catalog.Conditional))
	}
}
