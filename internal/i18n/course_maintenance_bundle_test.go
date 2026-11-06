package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCourseRevisionGenerationBundleDeterministicAndCurrent(t *testing.T) {
	root, catalog, base := courseV2GenerationBundleFixture(t)
	pageID := catalog.Pages[0].ID
	findingRepo := "data/locale-surface-reviews/" + base.Locale + "/synthetic-course-seo-finding.md"
	finding := []byte("# Synthetic Locale Surface finding\n\nPage " + pageID + ": clarify the description within the canonical English semantic scope.\n")
	writeSnapshotSource(t, root, findingRepo, finding)
	options := CourseMaintenanceGenerationBundleOptions{
		Locale: base.Locale, TaskKind: "revise", PageIDs: []string{pageID},
		FindingPath: filepath.Join(root, filepath.FromSlash(findingRepo)),
	}
	first, manifest, err := ExportCourseMaintenanceGenerationBundle(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	second, secondManifest, err := ExportCourseMaintenanceGenerationBundle(root, catalog, options)
	if err != nil || !bytes.Equal(first, second) || !reflect.DeepEqual(manifest, secondManifest) {
		t.Fatalf("course maintenance generation bundle is not deterministic: %v", err)
	}
	if manifest.TaskKind != "revise" || len(manifest.SelectedPageIDs) != 1 || manifest.SelectedPageIDs[0] != pageID || manifest.FindingPath != findingRepo {
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
	baseData, err := os.ReadFile(filepath.Join(root, "locales", base.Locale, "course-metadata.json"))
	if err != nil || !bytes.Equal(files["formal/course-metadata.json"], baseData) || !bytes.Equal(files["formal/reviewer-finding"], finding) {
		t.Fatalf("formal base/finding bytes mismatch: %v", err)
	}
	// Keep the original seven-field v2 context byte contract. The optional v1
	// fields must not enter the v2 JSON or change its instruction/output format.
	legacyContext := struct {
		SchemaVersion   int      `json:"schema_version"`
		Kind            string   `json:"kind"`
		TaskKind        string   `json:"task_kind"`
		Locale          string   `json:"locale"`
		SelectedPageIDs []string `json:"selected_page_ids"`
		Instruction     string   `json:"instruction"`
		ExpectedOutput  string   `json:"expected_output"`
	}{CourseMaintenanceGenerationBundleSchemaVersion, "go-tour-i18n/course-seo-maintenance-generation-context", "revise", base.Locale, []string{pageID}, "Generate only the selected page_id/description JSON subset. Canonical English description remains the exclusive semantic-scope authority; do not modify formal course metadata or claim review approval.", "localized-descriptions.json"}
	expectedContext, err := json.MarshalIndent(legacyContext, "", "  ")
	if err != nil || !bytes.Equal(files["generation-context.json"], append(expectedContext, '\n')) {
		t.Fatalf("v2 maintenance context byte contract changed: %v", err)
	}
	localization, _, err := ExportCourseLocalizationGenerationBundle(root, base.Locale, catalog)
	if err != nil {
		t.Fatal(err)
	}
	localizationFiles, err := ReadTransportBundle(localization, 512, 256<<20)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range localizationFiles {
		if name != "manifest.json" && !bytes.Equal(files["localization/"+name], data) {
			t.Fatalf("embedded localization bytes changed: %s", name)
		}
	}
	if _, err := VerifyCurrentCourseMaintenanceGenerationBundle(root, catalog, first); err != nil {
		t.Fatalf("current course maintenance bundle rejected: %v", err)
	}
}

func courseV1MaintenanceFixture(t *testing.T) (string, *Catalog, *CourseMetadata) {
	t.Helper()
	root := t.TempDir()
	catalog, targets, glossary, base := courseMetadataFixture()
	base.Locale = "zh-CN"
	glossary = bytes.Replace(glossary, []byte("test-LOCALE"), []byte("zh-CN"), 1)
	writeSnapshotSource(t, root, "locales/zh-CN/glossary.yaml", glossary)
	statuses := []Status{}
	for i, page := range catalog.Pages {
		path := canonicalCandidatePath("zh-CN", page.ID)
		writeSnapshotSource(t, root, path, targets[page.ID])
		statuses = append(statuses, Status{UnitID: page.ID, State: "ready", Attempts: 1, SourceSHA256: page.SourceSHA256, CandidatePath: path})
		base.Pages[i].GlossarySHA256 = sum(glossary)
	}
	if err := writeStatuses(filepath.Join(root, "locales/zh-CN/status.tsv"), statuses); err != nil {
		t.Fatal(err)
	}
	writeCourseMetadataFixture(t, root, "zh-CN", base)
	copyBundleAuthority(t, root, courseLocalizationGenerationAuthorityPaths)
	recordSyntheticUnifiedReview(t, root, "synthetic-v1-maintenance-review")
	return root, catalog, base
}

func TestCourseV1RefreshGenerationBundleCompleteCurrentAndExactSubset(t *testing.T) {
	for _, kind := range []string{"glossary", "subset"} {
		t.Run(kind, func(t *testing.T) {
			root, cat, base := courseV1MaintenanceFixture(t)
			if kind == "glossary" {
				raw, _ := os.ReadFile(filepath.Join(root, "locales/zh-CN/glossary.yaml"))
				writeSnapshotSource(t, root, "locales/zh-CN/glossary.yaml", append(raw, []byte("# glossary refresh fixture\n")...))
				recordSyntheticUnifiedReview(t, root, "synthetic-v1-maintenance-review-2")
			} else {
				base.Pages[1].SourceSHA256 = strings.Repeat("0", 64)
				writeCourseMetadataFixture(t, root, "zh-CN", base)
			}
			before, _ := os.ReadFile(filepath.Join(root, "locales/zh-CN/course-metadata.json"))
			options := CourseMaintenanceGenerationBundleOptions{Locale: "zh-CN", TaskKind: "refresh"}
			bundle, m, err := ExportCourseMaintenanceGenerationBundle(root, cat, options)
			if err != nil {
				t.Fatal(err)
			}
			repeated, _, err := ExportCourseMaintenanceGenerationBundle(root, cat, options)
			if err != nil || !bytes.Equal(bundle, repeated) {
				t.Fatal("non-deterministic v1 bundle", err)
			}
			// The existing refresh execution returns its selected set without writing.
			_, selected, err := RefreshCourseMetadata(root, cat, CourseMetadataRefreshOptions{Locale: "zh-CN", Provider: "fixture", Model: "fixture", GeneratedAt: "2026-10-12T00:00:00Z", Descriptions: marshalCourseRefreshDescriptions(t, m.SelectedPageIDs)})
			if err != nil || !reflect.DeepEqual(selected, m.SelectedPageIDs) {
				t.Fatal("bundle differs from formal refresh selection", err, selected, m.SelectedPageIDs)
			}
			files, err := ReadTransportBundle(bundle, 1024, 512<<20)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(courseSourceDescriptionsRelativePath))); !os.IsNotExist(err) {
				t.Fatal("fixture unexpectedly has canonical source-description authority")
			}
			for name, data := range files {
				if strings.Contains(name, "source-descriptions.json") || (!strings.HasPrefix(name, "authority/") && (strings.Contains(string(data), `"source_description_sha256"`) || strings.Contains(string(data), `"canonical_review_authority_sha256"`))) {
					t.Fatal("v2-only authority entered v1 inputs", name)
				}
			}
			var context CourseMaintenanceGenerationContext
			if err = decodeStrictBundleJSON(files["generation-context.json"], &context); err != nil {
				t.Fatal(err)
			}
			if context.BaseSchemaVersion != 1 || context.GeneratorContract != CourseMetadataGeneratorContract || len(context.Pages) != len(selected) || context.Constraints == "" {
				t.Fatal("v1 semantic scope missing")
			}
			targets, glossary, err := loadReadyCourseMetadataInputs(root, "zh-CN", cat)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(files["formal/glossary.yaml"], glossary) || !bytes.Equal(files["formal/course-metadata.json"], before) {
				t.Fatal("complete glossary/base missing")
			}
			byID, _ := courseMetadataBaseIndex(base, cat)
			for i, p := range context.Pages {
				page := cat.Pages[catalogPageIndex(cat, p.PageID)]
				if p.PageID != selected[i] || p.Route != page.Route || p.EnglishSource != string(page.Source) || p.ReadyTarget != string(targets[p.PageID]) || !reflect.DeepEqual(p.BaseEntry, byID[p.PageID]) || p.GlossarySHA256 != sum(glossary) {
					t.Fatal("incomplete selected Page context")
				}
			}
			if _, err = VerifyCurrentCourseMaintenanceGenerationBundle(root, cat, bundle); err != nil {
				t.Fatal("v1 current-check failed", err)
			}
			after, _ := os.ReadFile(filepath.Join(root, "locales/zh-CN/course-metadata.json"))
			if !bytes.Equal(before, after) {
				t.Fatal("export changed formal metadata")
			}
			if kind == "subset" && len(selected) != 1 {
				t.Fatal("caller-independent stale subset widened")
			}
		})
	}
}

func TestCourseV1RefreshGenerationBundleStaleAndFailClosed(t *testing.T) {
	for _, kind := range []string{"glossary", "source", "target", "base", "inventory", "missing-member", "selected-set"} {
		t.Run(kind, func(t *testing.T) {
			root, cat, base := courseV1MaintenanceFixture(t)
			base.Pages[0].SourceSHA256 = strings.Repeat("0", 64)
			writeCourseMetadataFixture(t, root, "zh-CN", base)
			bundle, _, err := ExportCourseMaintenanceGenerationBundle(root, cat, CourseMaintenanceGenerationBundleOptions{Locale: "zh-CN", TaskKind: "refresh"})
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "glossary":
				raw, _ := os.ReadFile(filepath.Join(root, "locales/zh-CN/glossary.yaml"))
				writeSnapshotSource(t, root, "locales/zh-CN/glossary.yaml", append(raw, []byte("# later refresh\n")...))
				recordSyntheticUnifiedReview(t, root, "synthetic-v1-maintenance-review-3")
			case "source":
				cat.Pages[0].Source = append(cat.Pages[0].Source, []byte("New source context.\n")...)
				cat.Pages[0].SourceSHA256 = sum(cat.Pages[0].Source)
				statuses, _ := ReadStatuses(filepath.Join(root, "locales/zh-CN/status.tsv"))
				statuses[0].SourceSHA256 = cat.Pages[0].SourceSHA256
				if err = writeStatuses(filepath.Join(root, "locales/zh-CN/status.tsv"), statuses); err != nil {
					t.Fatal(err)
				}
			case "target":
				path := canonicalCandidatePath("zh-CN", cat.Pages[0].ID)
				raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
				writeSnapshotSource(t, root, path, append(raw, []byte("Additional complete target context.\n")...))
			case "base":
				base.Pages[0].Description = courseDescription("changed base")
				writeCourseMetadataFixture(t, root, "zh-CN", base)
			case "inventory", "missing-member", "selected-set":
				files, _ := ReadTransportBundle(bundle, 1024, 512<<20)
				var m CourseMaintenanceGenerationBundleManifest
				json.Unmarshal(files["manifest.json"], &m)
				if kind == "selected-set" {
					m.SelectedPageIDs = append(m.SelectedPageIDs, cat.Pages[2].ID)
				} else if kind == "missing-member" {
					delete(files, "formal/glossary.yaml")
				} else {
					files["extra.txt"] = []byte("extra")
				}
				manifest, _ := json.MarshalIndent(m, "", "  ")
				entries := []TransportBundleEntry{}
				for name, data := range files {
					if name != "manifest.json" {
						entries = append(entries, TransportBundleEntry{Path: name, Data: data})
					}
				}
				bundle, err = WriteDeterministicTransportBundle(append(manifest, '\n'), entries)
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err = VerifyCurrentCourseMaintenanceGenerationBundle(root, cat, bundle); err == nil {
				t.Fatal("stale/tampered v1 bundle accepted")
			}
		})
	}
	root, cat, _ := courseV1MaintenanceFixture(t)
	if _, _, err := ExportCourseMaintenanceGenerationBundle(root, cat, CourseMaintenanceGenerationBundleOptions{Locale: "zh-CN", TaskKind: "refresh"}); err == nil || !strings.Contains(err.Error(), "no stale Pages") {
		t.Fatal("empty refresh exported", err)
	}
	for _, opts := range []CourseMaintenanceGenerationBundleOptions{{Locale: "zh-CN", TaskKind: "refresh", PageIDs: []string{cat.Pages[0].ID}}, {Locale: "zh-CN", TaskKind: "revise", PageIDs: []string{cat.Pages[0].ID}, FindingPath: "finding"}} {
		if _, _, err := ExportCourseMaintenanceGenerationBundle(root, cat, opts); err == nil {
			t.Fatal("v1 selection/revise restriction relaxed")
		}
	}
}
