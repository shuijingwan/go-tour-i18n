package i18n

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTraditionalChineseArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "zh-TW", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "歡迎！",
		"basics.article":      "套件、變數與函式",
		"flowcontrol.article": "流程控制陳述式：for、if、else、switch 與 defer",
		"moretypes.article":   "更多型別：結構、切片與映射",
		"methods.article":     "方法與介面",
		"generics.article":    "泛型",
		"concurrency.article": "並行處理",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("zh-TW article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("zh-TW article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" || strings.Contains(entry.Title+entry.Subtitle, "TODO") {
			t.Errorf("zh-TW article metadata %s = %+v, want title %q, subtitle, and no TODO", article, entry, title)
		}
	}
}

func TestBrazilianPortugueseArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "pt-BR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "Boas-vindas!",
		"basics.article":      "Pacotes, variáveis e funções",
		"flowcontrol.article": "Instruções de controle de fluxo: for, if, else, switch e defer",
		"moretypes.article":   "Mais tipos: structs, slices e maps",
		"methods.article":     "Métodos e interfaces",
		"generics.article":    "Genéricos",
		"concurrency.article": "Concorrência",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("pt-BR article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("pt-BR article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" || strings.Contains(entry.Title+entry.Subtitle, "TODO") {
			t.Errorf("pt-BR article metadata %s = %+v, want title %q, subtitle, and no TODO", article, entry, title)
		}
	}
}

func TestTurkishArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "tr-TR", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "Hoş geldiniz!",
		"basics.article":      "Paketler, değişkenler ve fonksiyonlar",
		"flowcontrol.article": "Akış denetimi ifadeleri: for, if, else, switch ve defer",
		"moretypes.article":   "Diğer türler: struct'lar, dilimler ve eşlemeler",
		"methods.article":     "Metotlar ve arayüzler",
		"generics.article":    "Jenerikler",
		"concurrency.article": "Eşzamanlılık",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("tr-TR article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("tr-TR article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" || strings.Contains(entry.Title+entry.Subtitle, "TODO") {
			t.Errorf("tr-TR article metadata %s = %+v, want title %q, subtitle, and no TODO", article, entry, title)
		}
	}
}

func TestDutchArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "nl-NL", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "Welkom!",
		"basics.article":      "Pakketten, variabelen en functies",
		"flowcontrol.article": "Besturingsinstructies: for, if, else, switch en defer",
		"moretypes.article":   "Meer typen: structs, slices en maps",
		"methods.article":     "Methoden en interfaces",
		"generics.article":    "Generieke typen",
		"concurrency.article": "Gelijktijdigheid",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("nl-NL article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("nl-NL article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" || strings.Contains(entry.Title+entry.Subtitle, "TODO") {
			t.Errorf("nl-NL article metadata %s = %+v, want title %q, subtitle, and no TODO", article, entry, title)
		}
	}
}

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

func TestSwedishArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "sv-SE", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "Välkommen!",
		"basics.article":      "Paket, variabler och funktioner",
		"flowcontrol.article": "Styrsatser: for, if, else, switch och defer",
		"moretypes.article":   "Fler typer: structar, slices och mappar",
		"methods.article":     "Metoder och gränssnitt",
		"generics.article":    "Generik",
		"concurrency.article": "Samtidighet",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("sv-SE article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("sv-SE article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" {
			t.Errorf("sv-SE article metadata %s = %+v, want title %q and a subtitle", article, entry, title)
		}
		if strings.Contains(entry.Title, "TODO") || strings.Contains(entry.Subtitle, "TODO") {
			t.Errorf("sv-SE article metadata %s retains TODO", article)
		}
	}
}

func TestPolishArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "pl-PL", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "Witamy!",
		"basics.article":      "Pakiety, zmienne i funkcje.",
		"flowcontrol.article": "Instrukcje sterujące: for, if, else, switch i defer",
		"moretypes.article":   "Więcej typów: struktury, wycinki i mapy.",
		"methods.article":     "Metody i interfejsy",
		"generics.article":    "Typy generyczne",
		"concurrency.article": "Współbieżność",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("pl-PL article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("pl-PL article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" || strings.Contains(entry.Title+entry.Subtitle, "TODO") {
			t.Errorf("pl-PL article metadata %s = %+v, want title %q, subtitle, and no TODO", article, entry, title)
		}
	}
}

func TestIndonesianArticleMetadataCoversCatalog(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	catalog, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := LoadArticleMetadata(root, "id-ID", catalog)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := map[string]string{
		"welcome.article":     "Selamat datang!",
		"basics.article":      "Paket, variabel, dan fungsi.",
		"flowcontrol.article": "Pernyataan kontrol alur: for, if, else, switch, dan defer",
		"moretypes.article":   "Tipe lainnya: struct, slice, dan map.",
		"methods.article":     "Method dan interface",
		"generics.article":    "Generik",
		"concurrency.article": "Konkurensi",
	}
	if len(metadata) != len(wantTitles) {
		t.Fatalf("id-ID article metadata count = %d, want %d", len(metadata), len(wantTitles))
	}
	for article, title := range wantTitles {
		entry, ok := metadata[article]
		if !ok {
			t.Errorf("id-ID article metadata is missing %s", article)
			continue
		}
		if entry.Title != title || entry.Subtitle == "" || strings.Contains(entry.Title+entry.Subtitle, "TODO") {
			t.Errorf("id-ID article metadata %s = %+v, want title %q, subtitle, and no TODO", article, entry, title)
		}
	}
}
