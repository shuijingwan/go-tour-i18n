package ui

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadEmbeddedCatalogs(t *testing.T) {
	for _, locale := range []string{"en", "zh-CN"} {
		catalog, err := Load(locale)
		if err != nil {
			t.Fatalf("Load(%q): %v", locale, err)
		}
		if got, want := len(catalog.Messages), 70; got != want {
			t.Fatalf("Load(%q) message count = %d, want %d", locale, got, want)
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
