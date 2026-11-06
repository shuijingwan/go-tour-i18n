package main

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Opt-in real-state probe: run the export/current-check contracts entirely in
// memory, with the same inventory and default scope as the CLI. No ZIP or
// formal artifact is written. An already exported batch is the only input.
func TestGenerationBundleReadOnlyPreparation(t *testing.T) {
	root, locale, batch := os.Getenv("GO_TOUR_PREP_ROOT"), os.Getenv("GO_TOUR_PREP_LOCALE"), os.Getenv("GO_TOUR_PREP_BATCH")
	if root == "" || locale == "" || batch == "" {
		t.Skip("requires explicit GO_TOUR_PREP_ROOT/LOCALE/BATCH")
	}
	started := time.Now()
	done := contentidentity.BeginMemoization()
	defer done()
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	c, err := i18n.ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := i18n.HydrateCatalogSources(c, current); err != nil {
		t.Fatal(err)
	}
	if err := sitecontent.AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	b, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: locale, BatchID: batch})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("READ_ONLY_GENERATION_BUNDLE_PREP_WALL=%s outputs=%d identity=%s", time.Since(started), len(m.ExpectedOutputs), m.InputIdentitySHA256)
	if len(c.TranslationScope) != 0 {
		t.Fatal("export changed the CLI default scope")
	}
	checked, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, b)
	if err != nil || !reflect.DeepEqual(m, checked) {
		t.Fatalf("in-memory canonical current-check: %v", err)
	}
	t.Logf("READ_ONLY_CURRENT_CHECK=PASS TOTAL_WALL=%s NO_OUTPUT_WRITTEN=yes", time.Since(started))
}
