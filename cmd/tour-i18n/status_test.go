package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestInitializeLocaleStatusCommand(t *testing.T) {
	root := t.TempDir()
	localeDir := filepath.Join(root, "locales", "de-DE")
	if err := os.MkdirAll(localeDir, 0755); err != nil {
		t.Fatal(err)
	}
	metadata := []byte(`{"locale":"de-DE","phase":"scaffold","translation_unit":"present.Section"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "locale.json"), metadata, 0644); err != nil {
		t.Fatal(err)
	}
	catalog := &i18n.Catalog{Pages: []i18n.Page{{ID: "lesson/1", SourceSHA256: strings.Repeat("a", 64)}}}

	locale, result, err := initializeLocaleStatusCommand(root, catalog, []string{"--locale", "de-DE"})
	if err != nil {
		t.Fatal(err)
	}
	if locale != "de-DE" || *result != (i18n.StatusInitializationResult{Total: 1, Pages: 1, Examples: 0}) {
		t.Fatalf("locale=%q result=%+v", locale, result)
	}
	if err := i18n.CheckStatus(root, "de-DE", catalog); err != nil {
		t.Fatalf("CLI initialization failed status check: %v", err)
	}
}

func TestInitializeLocaleStatusCommandRejectsInvalidArguments(t *testing.T) {
	if _, _, err := initializeLocaleStatusCommand(t.TempDir(), &i18n.Catalog{}, nil); err == nil || !strings.Contains(err.Error(), "--locale is required") {
		t.Fatalf("missing locale error = %v", err)
	}
	if _, _, err := initializeLocaleStatusCommand(t.TempDir(), &i18n.Catalog{}, []string{"--locale", "de-DE", "extra"}); err == nil || !strings.Contains(err.Error(), "unexpected status init arguments") {
		t.Fatalf("unexpected argument error = %v", err)
	}
}
