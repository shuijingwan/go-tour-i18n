package i18n

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"testing"
)

func TestExportCourseLocalizationGenerationBundleDeterministicAndComplete(t *testing.T) {
	root, catalog, base := courseV2GenerationBundleFixture(t)
	locale := base.Locale
	first, manifest, err := ExportCourseLocalizationGenerationBundle(root, locale, catalog)
	if err != nil {
		t.Fatal(err)
	}
	second, secondManifest, err := ExportCourseLocalizationGenerationBundle(root, locale, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("Course SEO localization bundle is not deterministic")
	}
	if !reflect.DeepEqual(manifest, secondManifest) {
		t.Fatal("Course SEO localization bundle manifest is not deterministic")
	}
	if manifest.SchemaVersion != CourseLocalizationGenerationBundleSchemaVersion || manifest.Locale != locale {
		t.Fatalf("unexpected bundle identity: %+v", manifest)
	}
	if manifest.PageCount != len(catalog.Pages) {
		t.Fatalf("page_count=%d want %d", manifest.PageCount, len(catalog.Pages))
	}
	reader, err := zip.NewReader(bytes.NewReader(first), int64(len(first)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, file := range reader.File {
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := files[file.Name]; exists {
			t.Fatalf("duplicate ZIP entry %q", file.Name)
		}
		files[file.Name] = data
	}
	for _, name := range []string{"manifest.json", "course-seo-localization.json", "formal/source-descriptions.json", "formal/glossary.yaml", "formal/locale.json"} {
		if _, ok := files[name]; !ok {
			t.Fatalf("bundle missing %s", name)
		}
	}
	if sum(files[manifest.Context.BundlePath]) != manifest.Context.SHA256 {
		t.Fatal("context SHA-256 mismatch")
	}
	if sum(files[manifest.SourceDescriptions.BundlePath]) != manifest.SourceDescriptions.SHA256 {
		t.Fatal("source descriptions SHA-256 mismatch")
	}
	if sum(files[manifest.Glossary.BundlePath]) != manifest.Glossary.SHA256 {
		t.Fatal("glossary SHA-256 mismatch")
	}
	if sum(files[manifest.LocaleIdentity.BundlePath]) != manifest.LocaleIdentity.SHA256 {
		t.Fatal("locale identity SHA-256 mismatch")
	}
	for _, authority := range manifest.Authority {
		data, ok := files[authority.BundlePath]
		if !ok || sum(data) != authority.SHA256 {
			t.Fatalf("authority mismatch: %s", authority.BundlePath)
		}
	}
	var embeddedManifest CourseLocalizationGenerationBundleManifest
	if err := json.Unmarshal(files["manifest.json"], &embeddedManifest); err != nil {
		t.Fatal(err)
	}
	if embeddedManifest.PageCount != manifest.PageCount || embeddedManifest.Context.SHA256 != manifest.Context.SHA256 {
		t.Fatal("embedded manifest does not match returned manifest")
	}
	var context CourseLocalizationGenerationContext
	if err := json.Unmarshal(files["course-seo-localization.json"], &context); err != nil {
		t.Fatal(err)
	}
	if len(context.Pages) != len(catalog.Pages) || context.Locale != locale || context.GeneratorContract != "course-seo-localization-v2" {
		t.Fatalf("unexpected localization context identity: locale=%s pages=%d contract=%s", context.Locale, len(context.Pages), context.GeneratorContract)
	}
	if context.CanonicalSourceDescriptionsSHA256 != manifest.SourceDescriptions.SHA256 {
		t.Fatal("context canonical source-descriptions identity mismatch")
	}
	reviewAuthority, err := RequireCurrentCourseSourceDescriptionReview(root, catalog)
	if err != nil || context.CanonicalReviewAuthoritySHA256 != reviewAuthority {
		t.Fatalf("context canonical review authority mismatch: %v", err)
	}
	if _, err := VerifyCurrentCourseLocalizationGenerationBundle(root, locale, catalog, first); err != nil {
		t.Fatalf("current localization bundle rejected: %v", err)
	}
}

// Reuse the ready-target/glossary fixture and real authority builders. All
// language assets and review receipts belong to t.TempDir, not a live locale.
func courseV2GenerationBundleFixture(t *testing.T) (string, *Catalog, *CourseMetadata) {
	t.Helper()
	root, catalog, predecessor := courseV1MaintenanceFixture(t)
	writeCourseSourceDescriptionAsset(t, root, catalog, nil)
	writeCourseSourceDescriptionReview(t, root, catalog, "synthetic-v2-bundle-source-review")
	data, err := AssembleCourseMetadata(root, catalog, CourseMetadataAssemblyOptions{
		SchemaVersion: CourseMetadataSchemaVersionV2,
		Locale:        predecessor.Locale,
		Provider:      "fixture",
		Model:         "fixture-model",
		GeneratedAt:   "2026-09-15T02:03:04Z",
		Descriptions:  marshalCourseDescriptions(t, catalog),
	})
	if err != nil {
		t.Fatal(err)
	}
	var base CourseMetadata
	if err := decodeStrictBundleJSON(data, &base); err != nil {
		t.Fatal(err)
	}
	writeCourseMetadataFixture(t, root, base.Locale, &base)
	return root, catalog, &base
}
