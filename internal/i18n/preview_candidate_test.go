package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildCandidatePreviewDoesNotModifyEnglishSource(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := HydrateCatalogSources(committed, catalog); err != nil {
		t.Fatal(err)
	}
	originalPath := filepath.Join(root, "_content", "tour", "welcome.article")
	original, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := BuildCandidatePreview(root, committed, "welcome/1", "zh-CN", filepath.Join(t.TempDir(), "preview"))
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) {
		t.Fatal("repository English source was modified")
	}
	temporary, err := os.ReadFile(filepath.Join(preview.ContentDir, "tour", "welcome.article"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{[]byte("Go 语言之旅"), []byte("本教程"), []byte("* Go local")} {
		if !bytes.Contains(temporary, want) {
			t.Fatalf("temporary preview missing %q", want)
		}
	}
	if bytes.Contains(original, []byte("Go 语言之旅")) {
		t.Fatal("repository English source unexpectedly contains candidate text")
	}
	if bytes.Contains(temporary, []byte("#appengine:")) {
		t.Fatal("temporary preview contains appengine markers")
	}
	wantTitles := [][]byte{
		[]byte("* Hello, 世界"),
		[]byte("* Go local"),
		[]byte("* Go offline (optional)"),
		[]byte("* The Go Playground"),
		[]byte("* Congratulations"),
	}
	previous := -1
	for _, title := range wantTitles {
		index := bytes.Index(temporary, title)
		if index <= previous {
			t.Fatalf("temporary preview publication order is wrong at %q", title)
		}
		previous = index
	}
}
