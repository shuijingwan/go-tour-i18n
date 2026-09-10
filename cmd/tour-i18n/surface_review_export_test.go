package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestExportLocaleSurfaceReviewCommandDoesNotLeavePartialOutput(t *testing.T) {
	workdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := workdir
	if _, err := os.Stat(filepath.Join(root, "data", "tour-pages.tsv")); os.IsNotExist(err) {
		root = filepath.Clean(filepath.Join(workdir, "..", ".."))
	}
	catalog, err := i18n.ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := i18n.HydrateCatalogSources(catalog, current); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing", "package.json")
	if err := exportLocaleSurfaceReviewCommand(root, catalog, []string{"--locale", "tr-TR", "--output", missing}); err == nil {
		t.Fatal("missing output parent accepted")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("failed export left output: %v", err)
	}
}
