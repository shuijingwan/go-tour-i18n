package i18n

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestCourseRevisionGenerationBundleDeterministicAndCurrent(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	options := CourseMaintenanceGenerationBundleOptions{
		Locale: "uk-UA", TaskKind: "revise", PageIDs: []string{"welcome/1"},
		FindingPath: filepath.Join(root, "data", "locale-surface-reviews", "uk-UA", "20260921-first-production.md"),
	}
	first, manifest, err := ExportCourseMaintenanceGenerationBundle(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := ExportCourseMaintenanceGenerationBundle(root, catalog, options)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("course maintenance generation bundle is not deterministic: %v", err)
	}
	if manifest.TaskKind != "revise" || len(manifest.SelectedPageIDs) != 1 || manifest.SelectedPageIDs[0] != "welcome/1" || manifest.FindingPath == "" {
		t.Fatalf("manifest=%+v", manifest)
	}
	files, err := ReadTransportBundle(first, 1024, 512<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"generation-context.json", "formal/course-metadata.json", "formal/reviewer-finding", "localization/course-seo-localization.json"} {
		if _, ok := files[required]; !ok {
			t.Fatalf("bundle missing %s", required)
		}
	}
	if _, err := VerifyCurrentCourseMaintenanceGenerationBundle(root, catalog, first); err != nil {
		t.Fatalf("current course maintenance bundle rejected: %v", err)
	}
}
