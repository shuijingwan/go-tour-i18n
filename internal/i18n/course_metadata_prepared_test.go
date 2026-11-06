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

func requireSingleCourseCompatibilityPreparation(t *testing.T, stats glossaryCompatibilityPrepareStats) {
	t.Helper()
	want := glossaryCompatibilityPrepareStats{PrepareCalls: 1, FullReviewChecks: 1, ContextInventoryBuilds: 1, EvidenceSetReads: 1, EvidenceVerifications: 1, EdgePreparations: 1}
	if stats != want {
		t.Fatalf("bulk refresh preparation=%+v, want %+v", stats, want)
	}
}

func TestRefreshCourseMetadataSharedPreparedCompatibility(t *testing.T) {
	for _, affected := range []bool{false, true} {
		name := "all-compatible"
		if affected {
			name = "seo-compatible-page-affected"
		}
		t.Run(name, func(t *testing.T) {
			root, cat, old := compatibilityFixture(t)
			data := "# compatible byte change\n" + compatibilityFixtureGlossary
			if affected {
				data += "preferred:\n  transaction: 事务\n"
			}
			current := compatibilityChange(t, root, data, "refresh-shared")
			if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", old, cat); err != nil {
				t.Fatal(err)
			}
			basePath := "locales/zh-CN/course-metadata.json"
			before, _ := readCompatibilityFile(root, basePath)
			if affected {
				p := PrepareGlossaryCompatibility(root, "zh-CN", cat)
				if r := p.Resolve(old, current, "seo:lesson/1"); r.Status != "compatible" {
					t.Fatalf("SEO scope: %+v", r)
				}
				if r := p.Resolve(old, current, "seo-page:lesson/1"); r.Status != "affected" {
					t.Fatalf("Page scope: %+v", r)
				}
			}
			descriptions := []byte(`{"pages":[]}`)
			wantStale := []string{}
			if affected {
				wantStale = []string{"lesson/1"}
				descriptions, _ = json.Marshal(courseDescriptionsFile{Pages: []courseDescriptionEntry{{PageID: "lesson/1", Description: courseDescription("replacement lesson/1")}}})
				stats := glossaryCompatibilityPrepareStats{}
				out, _, err := refreshCourseMetadataWithCompatibilityStats(root, cat, CourseMetadataRefreshOptions{Locale: "zh-CN", Descriptions: []byte(`{"pages":[]}`)}, &stats)
				if err == nil || err.Error() != "course refresh descriptions is missing stale page(s): lesson/1" || out != nil {
					t.Fatalf("incorrect fail-closed subset: %v", err)
				}
				requireSingleCourseCompatibilityPreparation(t, stats)
			}
			stats := glossaryCompatibilityPrepareStats{}
			out, stale, err := refreshCourseMetadataWithCompatibilityStats(root, cat, CourseMetadataRefreshOptions{Locale: "zh-CN", Provider: "fixture", Model: "fixture-new", GeneratedAt: "2026-10-09T00:00:00Z", Descriptions: descriptions}, &stats)
			if err != nil || !reflect.DeepEqual(stale, wantStale) {
				t.Fatalf("stale=%v want=%v err=%v", stale, wantStale, err)
			}
			requireSingleCourseCompatibilityPreparation(t, stats)
			original, err := decodeCourseMetadata(before)
			if err != nil {
				t.Fatal(err)
			}
			refreshed, err := decodeCourseMetadata(out)
			if err != nil {
				t.Fatal(err)
			}
			for i, entry := range original.Pages {
				if affected && entry.PageID == "lesson/1" {
					continue
				}
				if !reflect.DeepEqual(entry, refreshed.Pages[i]) {
					t.Fatalf("compatible Page/provenance changed: %s", entry.PageID)
				}
			}
			after, _ := readCompatibilityFile(root, basePath)
			if !bytes.Equal(before, after) {
				t.Fatal("refresh calculation wrote formal metadata")
			}
			if affected {
				stats := glossaryCompatibilityPrepareStats{}
				out, _, err := refreshCourseMetadataWithCompatibilityStats(root, cat, CourseMetadataRefreshOptions{Locale: "zh-CN", Provider: "fixture", Model: "fixture-new", GeneratedAt: "2026-10-09T00:00:00Z", Descriptions: []byte(`{"pages":[{"page_id":"lesson/1","description":"too short"}]}`)}, &stats)
				if err == nil || !strings.Contains(err.Error(), "validate refreshed course metadata") || out != nil {
					t.Fatalf("final validation bypassed: %v", err)
				}
				requireSingleCourseCompatibilityPreparation(t, stats)
			}
		})
	}
}

func TestRefreshCourseMetadataPreparedExactSHACheapPath(t *testing.T) {
	root, cat, _ := writeCourseMetadataRefreshFixture(t, 3)
	stats := glossaryCompatibilityPrepareStats{}
	_, stale, err := refreshCourseMetadataWithCompatibilityStats(root, cat, CourseMetadataRefreshOptions{Locale: "test-LOCALE", Descriptions: []byte(`{"pages":[]}`)}, &stats)
	if err != nil || len(stale) != 0 {
		t.Fatalf("exact refresh: stale=%v err=%v", stale, err)
	}
	if stats != (glossaryCompatibilityPrepareStats{}) {
		t.Fatalf("exact SHA triggered expensive preparation: %+v", stats)
	}
}

func TestRefreshCourseMetadataPreparedMissingOrStaleEvidence(t *testing.T) {
	for _, kind := range []string{"missing", "stale"} {
		t.Run(kind, func(t *testing.T) {
			root, cat, old := compatibilityFixture(t)
			compatibilityChange(t, root, "# compatible refresh\n"+compatibilityFixtureGlossary, "refresh-missing")
			_, path, err := AssessGlossaryCompatibility(root, "zh-CN", old, cat)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "missing" {
				if err := os.Rename(filepath.Join(root, path), filepath.Join(root, "missing-edge.json")); err != nil {
					t.Fatal(err)
				}
			} else {
				writeSnapshotSource(t, root, "internal/tour/ui/zh-CN.json", []byte(`{"locale":"zh-CN","html_lang":"zh-CN","messages":{"test.one":{"kind":"plain","text":"Modified context"}}}`))
			}
			stats := glossaryCompatibilityPrepareStats{}
			out, _, err := refreshCourseMetadataWithCompatibilityStats(root, cat, CourseMetadataRefreshOptions{Locale: "zh-CN", Descriptions: []byte(`{"pages":[]}`)}, &stats)
			if err == nil || err.Error() != "course refresh descriptions is missing stale page(s): lesson/1, lesson/2, lesson/3" || out != nil {
				t.Fatalf("missing/stale evidence carried: %v", err)
			}
			if stats.PrepareCalls != 1 {
				t.Fatalf("repeated failed preparation: %+v", stats)
			}
		})
	}
}
