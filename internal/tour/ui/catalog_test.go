package ui

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
)

const expectedCatalogMessages = 92

func TestLoadEmbeddedCatalogs(t *testing.T) {
	for _, locale := range []string{"de-DE", "en", "es-ES", "fr-FR", "it-IT", "ja-JP", "ko-KR", "nl-NL", "pt-BR", "tr-TR", "zh-CN"} {
		catalog, err := Load(locale)
		if err != nil {
			t.Fatalf("Load(%q): %v", locale, err)
		}
		if got, want := len(catalog.Messages), expectedCatalogMessages; got != want {
			t.Fatalf("Load(%q) message count = %d, want %d", locale, got, want)
		}
	}
}

func TestLoadFromFSUsesSuppliedCatalogFiles(t *testing.T) {
	en, err := catalogFiles.ReadFile("en.json")
	if err != nil {
		t.Fatal(err)
	}
	target, err := catalogFiles.ReadFile("tr-TR.json")
	if err != nil {
		t.Fatal(err)
	}
	target = bytes.Replace(target, []byte(`"Program sonlandı"`), []byte(`"Program tamamlandı"`), 1)
	loaded, err := LoadFromFS("tr-TR", fstest.MapFS{"en.json": {Data: en}, "tr-TR.json": {Data: target}})
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Messages["execution.exited"].Text; got != "Program tamamlandı" {
		t.Fatalf("filesystem target=%q", got)
	}
	if _, err := LoadFromFS("tr-TR", fstest.MapFS{"en.json": {Data: en}, "tr-TR.json": {Data: []byte("{")}}); err == nil {
		t.Fatal("malformed filesystem catalog accepted")
	}
}

func TestBrazilianPortugueseCatalogMatchesEnglishSource(t *testing.T) {
	source, err := Load("en")
	if err != nil {
		t.Fatal(err)
	}
	portuguese, err := Load("pt-BR")
	if err != nil {
		t.Fatal(err)
	}
	if portuguese.HTMLLang != "pt-BR" {
		t.Fatalf("pt-BR HTMLLang = %q, want pt-BR", portuguese.HTMLLang)
	}
	if got, want := len(portuguese.Messages), expectedCatalogMessages; got != want {
		t.Fatalf("pt-BR message count = %d, want %d", got, want)
	}
	if err := validateCoverage(source, portuguese); err != nil {
		t.Fatalf("pt-BR coverage: %v", err)
	}
	placeholderRE := regexp.MustCompile(`\{[a-z][a-z0-9_]*\}`)
	markupRE := regexp.MustCompile(`<[^>]+>`)
	for key, sourceMessage := range source.Messages {
		message := portuguese.Messages[key]
		if got, want := strings.Join(placeholderRE.FindAllString(message.Text, -1), "\x00"), strings.Join(placeholderRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
			t.Errorf("pt-BR message %q placeholders = %q, want %q", key, got, want)
		}
		if sourceMessage.Kind == "rich" {
			if got, want := strings.Join(markupRE.FindAllString(message.Text, -1), "\x00"), strings.Join(markupRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
				t.Errorf("pt-BR rich message %q markup = %q, want %q", key, got, want)
			}
		}
		if message.Text == sourceMessage.Text && key != "footer.github" {
			t.Errorf("pt-BR message %q duplicates English source text", key)
		}
		if strings.Contains(message.Text, "TODO") {
			t.Errorf("pt-BR message %q retains TODO", key)
		}
	}
	for key, want := range map[string]string{
		"editor.run": "Executar", "editor.format": "Formatar", "editor.reset": "Redefinir",
		"tour.title": "Um Tour por Go",
	} {
		if got := portuguese.Messages[key].Text; got != want {
			t.Errorf("pt-BR message %q = %q, want %q", key, got, want)
		}
	}
}

func TestDutchCatalogMatchesEnglishSource(t *testing.T) {
	source, err := Load("en")
	if err != nil {
		t.Fatal(err)
	}
	dutch, err := Load("nl-NL")
	if err != nil {
		t.Fatal(err)
	}
	if dutch.HTMLLang != "nl-NL" {
		t.Fatalf("nl-NL HTMLLang = %q, want nl-NL", dutch.HTMLLang)
	}
	if got, want := len(dutch.Messages), expectedCatalogMessages; got != want {
		t.Fatalf("nl-NL message count = %d, want %d", got, want)
	}
	if err := validateCoverage(source, dutch); err != nil {
		t.Fatalf("nl-NL coverage: %v", err)
	}
	for key, message := range dutch.Messages {
		if strings.Contains(message.Text, "TODO") {
			t.Errorf("nl-NL message %q retains TODO", key)
		}
	}
	for key, want := range map[string]string{
		"editor.run": "Uitvoeren", "editor.format": "Formatteren", "editor.reset": "Herstellen",
		"tour.title": "Een rondleiding door Go",
	} {
		if got := dutch.Messages[key].Text; got != want {
			t.Errorf("nl-NL message %q = %q, want %q", key, got, want)
		}
	}
}

func TestEditorToggleStatesAreLocalizedPerCatalog(t *testing.T) {
	wants := map[string][2]string{
		"en":    {"On", "Off"},
		"de-DE": {"Ein", "Aus"},
		"es-ES": {"Activado", "Desactivado"},
		"fr-FR": {"Activé", "Désactivé"},
		"it-IT": {"Attivato", "Disattivato"},
		"ja-JP": {"オン", "オフ"},
		"ko-KR": {"켜기", "끄기"},
		"nl-NL": {"Aan", "Uit"},
		"pt-BR": {"Ativado", "Desativado"},
		"tr-TR": {"Açık", "Kapalı"},
		"zh-CN": {"开启", "关闭"},
	}
	for locale, want := range wants {
		catalog, err := Load(locale)
		if err != nil {
			t.Fatal(err)
		}
		for index, key := range []string{"editor.on", "editor.off"} {
			got, err := catalog.Plain(key)
			if err != nil {
				t.Fatal(err)
			}
			if got != want[index] {
				t.Errorf("%s %s = %q, want %q", locale, key, got, want[index])
			}
		}
	}
}

func TestTurkishCatalogMatchesEnglishSource(t *testing.T) {
	source, err := Load("en")
	if err != nil {
		t.Fatal(err)
	}
	turkish, err := Load("tr-TR")
	if err != nil {
		t.Fatal(err)
	}
	if turkish.HTMLLang != "tr-TR" {
		t.Fatalf("tr-TR HTMLLang = %q, want tr-TR", turkish.HTMLLang)
	}
	if got, want := len(turkish.Messages), expectedCatalogMessages; got != want {
		t.Fatalf("tr-TR message count = %d, want %d", got, want)
	}
	if err := validateCoverage(source, turkish); err != nil {
		t.Fatalf("tr-TR coverage: %v", err)
	}
	placeholderRE := regexp.MustCompile(`\{[a-z][a-z0-9_]*\}`)
	markupRE := regexp.MustCompile(`<[^>]+>`)
	allowedUntranslatedNames := map[string]bool{
		"footer.github":       true,
		"site.issue_feedback": true,
	}
	for key, sourceMessage := range source.Messages {
		message := turkish.Messages[key]
		if got, want := strings.Join(placeholderRE.FindAllString(message.Text, -1), "\x00"), strings.Join(placeholderRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
			t.Errorf("tr-TR message %q placeholders = %q, want %q", key, got, want)
		}
		if sourceMessage.Kind == "rich" {
			if got, want := strings.Join(markupRE.FindAllString(message.Text, -1), "\x00"), strings.Join(markupRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
				t.Errorf("tr-TR rich message %q markup = %q, want %q", key, got, want)
			}
		}
		if strings.Contains(message.Text, "TODO") {
			t.Errorf("tr-TR message %q retains TODO", key)
		}
		if message.Text == sourceMessage.Text && !allowedUntranslatedNames[key] {
			t.Errorf("tr-TR message %q duplicates English source text", key)
		}
	}
	for key, want := range map[string]string{
		"editor.run": "Çalıştır", "editor.format": "Biçimlendir", "editor.reset": "Sıfırla",
		"tour.title": "Go Turu",
	} {
		if got := turkish.Messages[key].Text; got != want {
			t.Errorf("tr-TR message %q = %q, want %q", key, got, want)
		}
	}
}

func TestItalianCatalogMatchesEnglishSource(t *testing.T) {
	source, err := Load("en")
	if err != nil {
		t.Fatal(err)
	}
	italian, err := Load("it-IT")
	if err != nil {
		t.Fatal(err)
	}
	if italian.HTMLLang != "it-IT" {
		t.Fatalf("it-IT HTMLLang = %q, want it-IT", italian.HTMLLang)
	}
	if got, want := len(italian.Messages), expectedCatalogMessages; got != want {
		t.Fatalf("it-IT message count = %d, want %d", got, want)
	}
	if err := validateCoverage(source, italian); err != nil {
		t.Fatalf("it-IT coverage: %v", err)
	}
	for key, message := range italian.Messages {
		if strings.Contains(message.Text, "TODO") {
			t.Errorf("it-IT message %q retains TODO", key)
		}
	}
}

func TestFrenchCatalogMatchesEnglishSource(t *testing.T) {
	source, err := Load("en")
	if err != nil {
		t.Fatal(err)
	}
	french, err := Load("fr-FR")
	if err != nil {
		t.Fatal(err)
	}
	if french.HTMLLang != "fr-FR" {
		t.Fatalf("fr-FR HTMLLang = %q, want fr-FR", french.HTMLLang)
	}
	if got, want := len(french.Messages), expectedCatalogMessages; got != want {
		t.Fatalf("fr-FR message count = %d, want %d", got, want)
	}
	if err := validateCoverage(source, french); err != nil {
		t.Fatalf("fr-FR coverage: %v", err)
	}
	placeholderRE := regexp.MustCompile(`\{[a-z][a-z0-9_]*\}`)
	markupRE := regexp.MustCompile(`<[^>]+>`)
	allowedUntranslatedNames := map[string]bool{
		"site.issue_feedback": true,
		"footer.github":       true,
	}
	for key, sourceMessage := range source.Messages {
		frenchMessage := french.Messages[key]
		if got, want := strings.Join(placeholderRE.FindAllString(frenchMessage.Text, -1), "\x00"), strings.Join(placeholderRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
			t.Errorf("fr-FR message %q placeholders = %q, want %q", key, got, want)
		}
		if sourceMessage.Kind == "rich" {
			if got, want := strings.Join(markupRE.FindAllString(frenchMessage.Text, -1), "\x00"), strings.Join(markupRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
				t.Errorf("fr-FR rich message %q markup = %q, want %q", key, got, want)
			}
		}
		if frenchMessage.Text == sourceMessage.Text && !allowedUntranslatedNames[key] {
			t.Errorf("fr-FR message %q duplicates English source text", key)
		}
	}
}

func TestGermanCatalogMatchesEnglishSource(t *testing.T) {
	source, err := Load("en")
	if err != nil {
		t.Fatal(err)
	}
	german, err := Load("de-DE")
	if err != nil {
		t.Fatal(err)
	}
	if german.HTMLLang != "de-DE" {
		t.Fatalf("de-DE HTMLLang = %q, want de-DE", german.HTMLLang)
	}
	if got, want := len(german.Messages), expectedCatalogMessages; got != want {
		t.Fatalf("de-DE message count = %d, want %d", got, want)
	}
	if err := validateCoverage(source, german); err != nil {
		t.Fatalf("de-DE coverage: %v", err)
	}
	placeholderRE := regexp.MustCompile(`\{[a-z][a-z0-9_]*\}`)
	markupRE := regexp.MustCompile(`<[^>]+>`)
	allowedUntranslatedNames := map[string]bool{
		"module.generics.title": true,
		"site.issue_feedback":   true,
		"footer.github":         true,
	}
	for key, sourceMessage := range source.Messages {
		germanMessage := german.Messages[key]
		if got, want := strings.Join(placeholderRE.FindAllString(germanMessage.Text, -1), "\x00"), strings.Join(placeholderRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
			t.Errorf("de-DE message %q placeholders = %q, want %q", key, got, want)
		}
		if sourceMessage.Kind == "rich" {
			if got, want := strings.Join(markupRE.FindAllString(germanMessage.Text, -1), "\x00"), strings.Join(markupRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
				t.Errorf("de-DE rich message %q markup = %q, want %q", key, got, want)
			}
		}
		if germanMessage.Text == sourceMessage.Text && !allowedUntranslatedNames[key] {
			t.Errorf("de-DE message %q duplicates English source text", key)
		}
	}
}

func TestJapaneseCatalogMatchesEnglishSource(t *testing.T) {
	source, err := Load("en")
	if err != nil {
		t.Fatal(err)
	}
	japanese, err := Load("ja-JP")
	if err != nil {
		t.Fatal(err)
	}
	if japanese.HTMLLang != "ja-JP" {
		t.Fatalf("ja-JP HTMLLang = %q, want ja-JP", japanese.HTMLLang)
	}
	if got, want := len(japanese.Messages), expectedCatalogMessages; got != want {
		t.Fatalf("ja-JP message count = %d, want %d", got, want)
	}
	if err := validateCoverage(source, japanese); err != nil {
		t.Fatalf("ja-JP coverage: %v", err)
	}
	allowedUntranslatedNames := map[string]bool{
		"site.issue_feedback": true,
		"footer.github":       true,
	}
	for key, sourceMessage := range source.Messages {
		if japanese.Messages[key].Text == sourceMessage.Text && !allowedUntranslatedNames[key] {
			t.Errorf("ja-JP message %q duplicates English source text", key)
		}
	}
}

func TestKoreanCatalogMatchesEnglishSource(t *testing.T) {
	source, err := Load("en")
	if err != nil {
		t.Fatal(err)
	}
	korean, err := Load("ko-KR")
	if err != nil {
		t.Fatal(err)
	}
	if korean.HTMLLang != "ko-KR" {
		t.Fatalf("ko-KR HTMLLang = %q, want ko-KR", korean.HTMLLang)
	}
	if got, want := len(korean.Messages), expectedCatalogMessages; got != want {
		t.Fatalf("ko-KR message count = %d, want %d", got, want)
	}
	if err := validateCoverage(source, korean); err != nil {
		t.Fatalf("ko-KR coverage: %v", err)
	}
	placeholderRE := regexp.MustCompile(`\{[a-z][a-z0-9_]*\}`)
	markupRE := regexp.MustCompile(`<[^>]+>`)
	allowedUntranslatedNames := map[string]bool{
		"site.issue_feedback": true,
		"footer.github":       true,
	}
	for key, sourceMessage := range source.Messages {
		koreanMessage := korean.Messages[key]
		if got, want := strings.Join(placeholderRE.FindAllString(koreanMessage.Text, -1), "\x00"), strings.Join(placeholderRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
			t.Errorf("ko-KR message %q placeholders = %q, want %q", key, got, want)
		}
		if sourceMessage.Kind == "rich" {
			if got, want := strings.Join(markupRE.FindAllString(koreanMessage.Text, -1), "\x00"), strings.Join(markupRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
				t.Errorf("ko-KR rich message %q markup = %q, want %q", key, got, want)
			}
		}
		if koreanMessage.Text == sourceMessage.Text && !allowedUntranslatedNames[key] {
			t.Errorf("ko-KR message %q duplicates English source text", key)
		}
		if strings.Contains(koreanMessage.Text, "TODO") {
			t.Errorf("ko-KR message %q retains TODO", key)
		}
	}
}

func TestSpanishCatalogMatchesEnglishSource(t *testing.T) {
	source, err := Load("en")
	if err != nil {
		t.Fatal(err)
	}
	spanish, err := Load("es-ES")
	if err != nil {
		t.Fatal(err)
	}
	if spanish.HTMLLang != "es-ES" {
		t.Fatalf("es-ES HTMLLang = %q, want es-ES", spanish.HTMLLang)
	}
	if got, want := len(spanish.Messages), expectedCatalogMessages; got != want {
		t.Fatalf("es-ES message count = %d, want %d", got, want)
	}
	if err := validateCoverage(source, spanish); err != nil {
		t.Fatalf("es-ES coverage: %v", err)
	}
	placeholderRE := regexp.MustCompile(`\{[a-z][a-z0-9_]*\}`)
	markupRE := regexp.MustCompile(`<[^>]+>`)
	allowedUntranslatedNames := map[string]bool{
		"site.issue_feedback": true,
		"footer.github":       true,
	}
	for key, sourceMessage := range source.Messages {
		spanishMessage := spanish.Messages[key]
		if got, want := strings.Join(placeholderRE.FindAllString(spanishMessage.Text, -1), "\x00"), strings.Join(placeholderRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
			t.Errorf("es-ES message %q placeholders = %q, want %q", key, got, want)
		}
		if sourceMessage.Kind == "rich" {
			if got, want := strings.Join(markupRE.FindAllString(spanishMessage.Text, -1), "\x00"), strings.Join(markupRE.FindAllString(sourceMessage.Text, -1), "\x00"); got != want {
				t.Errorf("es-ES rich message %q markup = %q, want %q", key, got, want)
			}
		}
		if spanishMessage.Text == sourceMessage.Text && !allowedUntranslatedNames[key] {
			t.Errorf("es-ES message %q duplicates English source text", key)
		}
		if strings.Contains(spanishMessage.Text, "TODO") {
			t.Errorf("es-ES message %q retains TODO", key)
		}
	}
}

func TestCatalogPlainIsSafeForTemplateUse(t *testing.T) {
	catalog := Catalog{Messages: map[string]Message{
		"message.value": {Kind: "plain", Text: `<script>alert("x")</script>`},
		"message.rich":  {Kind: "rich", Text: "<p>text</p>"},
	}}
	tmpl := template.Must(template.New("test").Funcs(template.FuncMap{"ui": catalog.Plain}).Parse(`{{ui "message.value"}}`))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), `&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`; got != want {
		t.Fatalf("template output = %q, want %q", got, want)
	}
	if _, err := catalog.Plain("missing.key"); err == nil {
		t.Error("Plain(missing.key) succeeded, want error")
	}
	if got, err := catalog.Rich("message.rich"); err != nil || got != "<p>text</p>" {
		t.Errorf("Rich(message.rich) = %q, %v", got, err)
	}
	for _, key := range []string{"missing.key", "message.value"} {
		if _, err := catalog.Rich(key); err == nil {
			t.Errorf("Rich(%q) succeeded, want error", key)
		}
	}
}

func TestValidationFailures(t *testing.T) {
	source := mustCatalog(t, `{"locale":"en","html_lang":"en","messages":{"message.one":{"kind":"plain","text":"One"},"message.two":{"kind":"rich","text":"<p>Two</p>"}}}`)
	cases := []struct {
		name       string
		data       string
		want       string
		parseFails bool
	}{
		{"missing key", `{"locale":"zh-CN","html_lang":"zh-CN","messages":{"message.one":{"kind":"plain","text":"一"}}}`, "missing keys: message.two", false},
		{"extra key", `{"locale":"zh-CN","html_lang":"zh-CN","messages":{"message.one":{"kind":"plain","text":"一"},"message.two":{"kind":"rich","text":"<p>二</p>"},"message.extra":{"kind":"plain","text":"额外"}}}`, "unknown keys: message.extra", false},
		{"kind mismatch", `{"locale":"zh-CN","html_lang":"zh-CN","messages":{"message.one":{"kind":"rich","text":"<p>一</p>"},"message.two":{"kind":"rich","text":"<p>二</p>"}}}`, "message kind mismatch: message.one", false},
		{"malformed JSON", `{`, "EOF", true},
		{"duplicate key", `{"locale":"zh-CN","html_lang":"zh-CN","messages":{"message.one":{"kind":"plain","text":"一"},"message.one":{"kind":"plain","text":"壹"}}}`, "duplicate message key \"message.one\"", true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			catalog, err := parseCatalog([]byte(test.data))
			if test.parseFails {
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("parseCatalog error = %v, want %q", err, test.want)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCatalog: %v", err)
			}
			err = validateCoverage(source, catalog)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateCoverage error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRichMarkupWhitelist(t *testing.T) {
	valid := `{"locale":"en","html_lang":"en","messages":{"message.rich":{"kind":"rich","text":"<p>Go <a href=\"https://go.dev\">Tour</a></p>"}}}`
	if _, err := parseCatalog([]byte(valid)); err != nil {
		t.Fatalf("parse valid rich catalog: %v", err)
	}
	for _, invalid := range []string{
		`{"locale":"en","html_lang":"en","messages":{"message.rich":{"kind":"rich","text":"<p><strong>Tour</strong></p>"}}}`,
		`{"locale":"en","html_lang":"en","messages":{"message.rich":{"kind":"rich","text":"<p><a href=\"https://example.com\">Tour</a></p>"}}}`,
	} {
		if _, err := parseCatalog([]byte(invalid)); err == nil || !strings.Contains(err.Error(), "unsupported markup") {
			t.Fatalf("parse invalid rich catalog error = %v", err)
		}
	}
}

func TestUnknownLocale(t *testing.T) {
	if _, err := Load("ja"); err == nil || !strings.Contains(err.Error(), "unknown UI locale") {
		t.Fatalf("Load(ja) error = %v, want unknown locale", err)
	}
}

func TestLocaleSyntax(t *testing.T) {
	for _, locale := range []string{"en", "ja", "ko", "zh-CN", "zh-Hant", "zh-Hans", "pt-BR"} {
		if !localePattern.MatchString(locale) {
			t.Errorf("localePattern does not accept %q", locale)
		}
	}
	for _, locale := range []string{"", "/", `\\`, "..", "zh/../CN", ".zh", "zh.", "-zh", "zh-", "zh--Hant"} {
		if localePattern.MatchString(locale) {
			t.Errorf("localePattern accepts unsafe locale %q", locale)
		}
	}
}

func TestLoadDiscoversEmbeddedStyleLocaleFile(t *testing.T) {
	files := fstest.MapFS{
		"en.json":      {Data: []byte(`{"locale":"en","html_lang":"en","messages":{"message.one":{"kind":"plain","text":"One"}}}`)},
		"zh-Hant.json": {Data: []byte(`{"locale":"zh-Hant","html_lang":"zh-Hant","messages":{"message.one":{"kind":"plain","text":"一"}}}`)},
	}
	catalog, err := load("zh-Hant", files)
	if err != nil {
		t.Fatalf("load auto-discovered zh-Hant.json: %v", err)
	}
	if got := catalog.Messages["message.one"].Text; got != "一" {
		t.Fatalf("zh-Hant message = %q, want 一", got)
	}
}

func mustCatalog(t *testing.T, text string) Catalog {
	t.Helper()
	catalog, err := parseCatalog([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
