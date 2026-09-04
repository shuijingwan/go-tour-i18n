package i18n

import (
	"path/filepath"
	"strings"
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

func TestFrenchArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "fr-FR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "Bienvenue !",
		"basics.article":      "Paquets, variables et fonctions",
		"flowcontrol.article": "Instructions de contrôle du flux : for, if, else, switch et defer",
		"moretypes.article":   "Autres types : structures, slices et maps",
		"methods.article":     "Méthodes et interfaces",
		"generics.article":    "Génériques",
		"concurrency.article": "Concurrence",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("fr-FR article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("fr-FR article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" {
			t.Errorf("fr-FR article metadata %s = %+v, want title %q and a subtitle", article, entry, title)
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

func TestKoreanArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "ko-KR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "환영합니다!",
		"basics.article":      "패키지, 변수, 함수",
		"flowcontrol.article": "흐름 제어문: for, if, else, switch, defer",
		"moretypes.article":   "더 다양한 타입: 구조체, 슬라이스, 맵",
		"methods.article":     "메서드와 인터페이스",
		"generics.article":    "제네릭",
		"concurrency.article": "동시성",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("ko-KR article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("ko-KR article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" {
			t.Errorf("ko-KR article metadata %s = %+v, want title %q and a subtitle", article, entry, title)
		}
		if strings.Contains(entry.Title, "TODO") || strings.Contains(entry.Subtitle, "TODO") {
			t.Errorf("ko-KR article metadata %s retains TODO", article)
		}
	}
}

func TestSpanishArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "es-ES", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "¡Te damos la bienvenida!",
		"basics.article":      "Paquetes, variables y funciones.",
		"flowcontrol.article": "Sentencias de control de flujo: for, if, else, switch y defer",
		"moretypes.article":   "Más tipos: estructuras, slices y mapas.",
		"methods.article":     "Métodos e interfaces",
		"generics.article":    "Genéricos",
		"concurrency.article": "Concurrencia",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("es-ES article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("es-ES article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" {
			t.Errorf("es-ES article metadata %s = %+v, want title %q and a subtitle", article, entry, title)
		}
		if strings.Contains(entry.Title, "TODO") || strings.Contains(entry.Subtitle, "TODO") {
			t.Errorf("es-ES article metadata %s retains TODO", article)
		}
	}
}
