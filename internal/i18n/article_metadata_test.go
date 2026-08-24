package i18n

import (
	"path/filepath"
	"testing"
)

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
