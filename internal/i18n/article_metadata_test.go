package i18n

import (
	"path/filepath"
	"testing"
)

func TestGermanArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "de-DE", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "Willkommen!",
		"basics.article":      "Pakete, Variablen und Funktionen",
		"flowcontrol.article": "Kontrollflussanweisungen: for, if, else, switch und defer",
		"moretypes.article":   "Weitere Typen: Structs, Slices und Maps",
		"methods.article":     "Methoden und Interfaces",
		"generics.article":    "Generics",
		"concurrency.article": "Nebenläufigkeit",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("de-DE article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("de-DE article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" {
			t.Errorf("de-DE article metadata %s = %+v, want title %q and a subtitle", article, entry, title)
		}
	}
}

func TestJapaneseArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "ja-JP", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantArticles := []string{
		"welcome.article",
		"basics.article",
		"flowcontrol.article",
		"moretypes.article",
		"methods.article",
		"generics.article",
		"concurrency.article",
	}
	if len(metadata) != len(wantArticles) {
		t.Fatalf("ja-JP article metadata count = %d, want %d", len(metadata), len(wantArticles))
	}
	for _, article := range wantArticles {
		if _, ok := metadata[article]; !ok {
			t.Errorf("ja-JP article metadata is missing %s", article)
		}
	}
}
