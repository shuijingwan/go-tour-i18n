package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestInitializeLocaleCreatesMechanicalSkeleton(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{filepath.Join(root, "locales"), filepath.Join(root, "internal", "tour", "ui")} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	sourceUI := `{"locale":"en","html_lang":"en","messages":{"editor.run":{"kind":"plain","text":"Run"},"execution.test_failed":{"kind":"plain","text":"{count} test failed."},"module.description":{"kind":"rich","text":"<p>Learn <a href=\"https://go.dev\">Go</a>.</p>"}}}`
	if err := os.WriteFile(filepath.Join(root, "internal", "tour", "ui", "en.json"), []byte(sourceUI), 0644); err != nil {
		t.Fatal(err)
	}
	catalog := &i18n.Catalog{Pages: []i18n.Page{{
		ID: "welcome/1", Article: "welcome.article", Route: "/welcome/1", SourceSHA256: strings.Repeat("a", 64),
	}}}
	result, err := initializeLocale(root, catalog, localeInitOptions{
		Locale: "it-IT", LanguageName: "Italiano", EnglishName: "Italian", HTMLLang: "it-IT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.UnitCount != 1 || result.PageCount != 1 || result.ExampleCount != 0 {
		t.Fatalf("result=%+v", result)
	}
	for _, path := range []string{
		"locales/it-IT/locale.json", "locales/it-IT/glossary.yaml", "locales/it-IT/article-metadata.json",
		"locales/it-IT/course-metadata.todo.json", "locales/it-IT/status.tsv", "locales/it-IT/.locale-init-incomplete",
		"internal/tour/ui/it-IT.json",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "locales", "it-IT", "course-metadata.json")); !os.IsNotExist(err) {
		t.Fatalf("locale init created formal course metadata: %v", err)
	}
	if err := requireLocaleInitializationComplete(root, "it-IT"); err == nil {
		t.Fatal("incomplete locale passed complete build/publish gate")
	}
	if err := os.Remove(filepath.Join(root, "locales", "it-IT", localeInitIncompleteMarker)); err != nil {
		t.Fatal(err)
	}
	if err := requireLocaleInitializationComplete(root, "it-IT"); err != nil {
		t.Fatalf("completed locale gate: %v", err)
	}
	if err := i18n.CheckStatus(root, "it-IT", catalog); err != nil {
		t.Fatalf("generated status: %v", err)
	}
	uiData, err := os.ReadFile(filepath.Join(root, "internal", "tour", "ui", "it-IT.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ui localeInitUICatalog
	if err := json.Unmarshal(uiData, &ui); err != nil {
		t.Fatal(err)
	}
	if ui.Messages["editor.run"].Text != "TODO: editor.run" || !strings.Contains(ui.Messages["execution.test_failed"].Text, "{count}") {
		t.Fatalf("plain placeholders=%+v", ui.Messages)
	}
	rich := ui.Messages["module.description"].Text
	if !strings.Contains(rich, `<a href="https://go.dev">`) || strings.Contains(rich, "Learn") {
		t.Fatalf("rich placeholder did not preserve markup identity without copying prose: %q", rich)
	}
}

func TestInitializeLocaleFailsClosedWhenLocaleExists(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "locales", "it-IT"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "tour", "ui"), 0755); err != nil {
		t.Fatal(err)
	}
	_, err := initializeLocale(root, &i18n.Catalog{}, localeInitOptions{
		Locale: "it-IT", LanguageName: "Italiano", EnglishName: "Italian", HTMLLang: "it-IT",
	})
	if err == nil || !strings.Contains(err.Error(), "拒绝覆盖") {
		t.Fatalf("existing locale error=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "tour", "ui", "it-IT.json")); !os.IsNotExist(err) {
		t.Fatalf("failed init left UI catalog: %v", err)
	}
}

func TestInitializeLocaleFailsClosedWhenUICatalogExists(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{filepath.Join(root, "locales"), filepath.Join(root, "internal", "tour", "ui")} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	uiPath := filepath.Join(root, "internal", "tour", "ui", "it-IT.json")
	if err := os.WriteFile(uiPath, []byte("existing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := initializeLocale(root, &i18n.Catalog{}, localeInitOptions{
		Locale: "it-IT", LanguageName: "Italiano", EnglishName: "Italian", HTMLLang: "it-IT",
	})
	if err == nil || !strings.Contains(err.Error(), "拒绝覆盖") {
		t.Fatalf("existing UI error=%v", err)
	}
	if data, err := os.ReadFile(uiPath); err != nil || string(data) != "existing\n" {
		t.Fatalf("existing UI changed: data=%q err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(root, "locales", "it-IT")); !os.IsNotExist(err) {
		t.Fatalf("failed init left locale directory: %v", err)
	}
}

func TestInitializeLocaleRollsBackWhenStatusInitializationFails(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{filepath.Join(root, "locales"), filepath.Join(root, "internal", "tour", "ui")} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "tour", "ui", "en.json"), []byte(`{"locale":"en","html_lang":"en","messages":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	page := i18n.Page{ID: "welcome/1", Article: "welcome.article", Route: "/welcome/1", SourceSHA256: strings.Repeat("a", 64)}
	_, err := initializeLocale(root, &i18n.Catalog{Pages: []i18n.Page{page, page}}, localeInitOptions{
		Locale: "it-IT", LanguageName: "Italiano", EnglishName: "Italian", HTMLLang: "it-IT",
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate catalog translation unit") {
		t.Fatalf("status initialization error=%v", err)
	}
	for _, path := range []string{filepath.Join(root, "locales", "it-IT"), filepath.Join(root, "internal", "tour", "ui", "it-IT.json")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("failed init left target %s: %v", path, err)
		}
	}
}
