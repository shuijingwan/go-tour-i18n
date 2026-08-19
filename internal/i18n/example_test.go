package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExampleCatalogBaseline(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(catalog.Examples), 93; got != want {
		t.Fatalf("examples=%d, want %d", got, want)
	}
	seen := map[string]bool{}
	for _, example := range catalog.Examples {
		if seen[example.SourcePath] {
			t.Fatalf("duplicate example source path %q", example.SourcePath)
		}
		seen[example.SourcePath] = true
		if sum(example.Source) != example.SourceSHA256 {
			t.Fatalf("%s: source hash is not the complete file hash", example.ID)
		}
	}
	if seen["_content/tour/solutions/loops.go"] {
		t.Fatal("unreferenced solution was included in example catalog")
	}
	if err := CheckCatalogFiles(root, catalog); err != nil {
		t.Fatal(err)
	}
}

func TestExampleDiscoveryDeduplicatesReferences(t *testing.T) {
	root := t.TempDir()
	writeExampleFixture(t, root, "demo/main.go", "package main\n")
	pages := []Page{
		{ID: "lesson/1", Article: "lesson.article", Source: []byte("* One\n\n.play demo/main.go\n")},
		{ID: "lesson/2", Article: "lesson.article", Source: []byte("* Two\n\n.play demo/main.go\n")},
	}
	examples, err := discoverExamples(root, pages)
	if err != nil {
		t.Fatal(err)
	}
	if len(examples) != 1 {
		t.Fatalf("examples=%d, want 1", len(examples))
	}
	if got := strings.Join(examples[0].ReferencedBy, ","); got != "lesson/1,lesson/2" {
		t.Fatalf("referenced_by=%q", got)
	}
}

func TestExampleDiscoveryRejectsUnsafeAndMissingPaths(t *testing.T) {
	for _, test := range []struct {
		name, directive, want string
	}{
		{"unsafe", "../outside.go", "unsafe referenced path"},
		{"missing", "demo/missing.go", "no such file"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			pages := []Page{{ID: "lesson/1", Article: "lesson.article", Source: []byte("* One\n\n.play " + test.directive + "\n")}}
			_, err := discoverExamples(root, pages)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func TestExampleHashCoversCommentsAndCode(t *testing.T) {
	base := []byte("package main\n\n// Print a greeting.\nfunc main() {}\n")
	commentChanged := bytes.Replace(base, []byte("greeting"), []byte("message"), 1)
	codeChanged := bytes.Replace(base, []byte("main()"), []byte("run()"), 1)
	if sum(base) == sum(commentChanged) {
		t.Fatal("changing a comment did not change the complete-file hash")
	}
	if sum(base) == sum(codeChanged) {
		t.Fatal("changing code did not change the complete-file hash")
	}
}

func TestCatalogUnitResolvesPageAndExample(t *testing.T) {
	catalog, err := BuildCatalog(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Unit("basics/1")
	if err != nil {
		t.Fatal(err)
	}
	originalPage, _ := catalog.Page("basics/1")
	if page.Kind != UnitKindPage || page.ID != originalPage.ID || page.SourceSHA256 != originalPage.SourceSHA256 || !bytes.Equal(page.Source, originalPage.Source) {
		t.Fatalf("page unit=%+v", page)
	}
	example, err := catalog.Unit("example:basics/packages.go")
	if err != nil {
		t.Fatal(err)
	}
	if example.Kind != UnitKindExample || example.ID != "example:basics/packages.go" || example.SourcePath != "_content/tour/basics/packages.go" || sum(example.Source) != example.SourceSHA256 {
		t.Fatalf("example unit=%+v", example)
	}
}

func TestExampleManifestHashMismatchFails(t *testing.T) {
	root := t.TempDir()
	source := "package main\n\n// Original comment.\nfunc main() {}\n"
	writeExampleFixture(t, root, "demo/main.go", source)
	manifest := "upstream_path\tlocal_path\tmode\tupstream_sha256\tlocal_sha256\tnote\n" +
		fmt.Sprintf("_content/tour/demo/main.go\t_content/tour/demo/main.go\texact\t%s\t%s\ttest\n", sum([]byte(source)), strings.Repeat("0", 64))
	if err := os.WriteFile(filepath.Join(root, "UPSTREAM_MANIFEST.tsv"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	pages := []Page{{ID: "lesson/1", Article: "lesson.article", Source: []byte("* One\n\n.play demo/main.go\n")}}
	_, err := discoverExamples(root, pages)
	if err == nil || !strings.Contains(err.Error(), "does not match UPSTREAM_MANIFEST.tsv") {
		t.Fatalf("error=%v", err)
	}
}

func writeExampleFixture(t *testing.T, root, path, source string) {
	t.Helper()
	absolute := filepath.Join(root, "_content", "tour", filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(absolute), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absolute, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
}
