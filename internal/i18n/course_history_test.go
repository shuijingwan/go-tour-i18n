package i18n

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func courseHistoryFixture(t *testing.T) (string, *Catalog, []byte, []byte, string) {
	t.Helper()
	root, cat, _ := historicalRecoveryFixture(t)
	if _, _, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true); err != nil {
		t.Fatal(err)
	}
	old, err := readCompatibilityFile(root, courseMetadataRelativePath("zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	g1 := readSnapshotGlossarySHA(t, root, "qc-001")
	_, edge, err := AssessGlossaryCompatibility(root, "zh-CN", g1, cat)
	if err != nil {
		t.Fatal(err)
	}
	current, stale, err := RefreshCourseMetadata(root, cat, CourseMetadataRefreshOptions{Locale: "zh-CN", Provider: "fixture", Model: "fixture", GeneratedAt: "2026-10-09T02:29:24Z", Descriptions: []byte(`{"pages":[{"page_id":"lesson/1","description":"A refreshed and distinct description covering the lesson one concepts accurately."}]}`)})
	if err != nil || !reflect.DeepEqual(stale, []string{"lesson/1"}) {
		t.Fatalf("refresh: %v %v", stale, err)
	}
	writeSnapshotSource(t, root, courseMetadataRelativePath("zh-CN"), current)
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, cat); err != nil {
		t.Fatal(err)
	}
	return root, cat, old, current, edge
}

func TestCourseHistoryRecoveryExactProof(t *testing.T) {
	root, cat, old, current, _ := courseHistoryFixture(t)
	closure, _ := readCompatibilityFile(root, "locales/zh-CN/content-scope.json")
	if err := AuthorizeHistoricalCanonicalReference(root, "zh-CN", courseMetadataRelativePath("zh-CN"), sum(old), cat); err == nil {
		t.Fatal("TU authority authorized SEO")
	}
	e, path, err := RecoverCourseMetadataHistory(root, cat, "zh-CN", old, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.StalePages) != 1 || e.StalePages[0] != "lesson/1" {
		t.Fatal(e.StalePages)
	}
	if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
		t.Fatal("dry run mutation")
	}
	e2, path2, err := RecoverCourseMetadataHistory(root, cat, "zh-CN", old, false)
	if err != nil || path != path2 || !reflect.DeepEqual(e, e2) {
		t.Fatal("not deterministic", err)
	}
	if _, _, err := RecoverCourseMetadataHistory(root, cat, "zh-CN", old, true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecoverCourseMetadataHistory(root, cat, "zh-CN", old, true); err == nil {
		t.Fatal("overwrote receipt")
	}
	if _, err := AuthorizeHistoricalCourseMetadataReference(root, "zh-CN", sum(old), cat); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCourseMetadata(root, "zh-CN", cat); err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeHistoricalCanonicalReference(root, "zh-CN", courseMetadataRelativePath("zh-CN"), sum(old), cat); err == nil {
		t.Fatal("expanded TU authority")
	}
	after, _ := readCompatibilityFile(root, courseMetadataRelativePath("zh-CN"))
	if !bytes.Equal(after, current) {
		t.Fatal("recovery modified current metadata")
	}
	after, _ = readCompatibilityFile(root, "locales/zh-CN/content-scope.json")
	if !bytes.Equal(after, closure) {
		t.Fatal("closure changed")
	}
}

func TestCourseHistoryRecoveryFailClosed(t *testing.T) {
	for _, kind := range []string{"old-sha", "non-stale-description", "non-stale-provenance", "current-edit", "missing-edge", "stale-edge", "canonical", "source", "closure", "schema", "locale", "receipt", "archive", "missing-receipt", "fork"} {
		t.Run(kind, func(t *testing.T) {
			root, cat, old, current, edge := courseHistoryFixture(t)
			_, path, err := RecoverCourseMetadataHistory(root, cat, "zh-CN", old, true)
			if err != nil {
				t.Fatal(err)
			}
			recoverAgain := false
			switch kind {
			case "old-sha":
				old = append(old, ' ')
				recoverAgain = true
			case "non-stale-description", "non-stale-provenance", "current-edit":
				m, _ := decodeCourseMetadata(current)
				if kind == "non-stale-provenance" {
					m.Pages[1].Generation.Model = "manual"
				} else if kind == "non-stale-description" {
					m.Pages[1].Description = courseDescription("manual non-stale")
				} else {
					m.Pages[0].Description = courseDescription("manual stale")
				}
				writeSnapshotSource(t, root, courseMetadataRelativePath("zh-CN"), marshalCourseMetadata(t, m))
				recoverAgain = kind != "current-edit"
			case "missing-edge":
				if err := os.Rename(filepath.Join(root, edge), filepath.Join(root, "removed-edge.json")); err != nil {
					t.Fatal(err)
				}
				recoverAgain = true
			case "stale-edge":
				writeSnapshotSource(t, root, "internal/tour/ui/zh-CN.json", []byte(`{"locale":"zh-CN","html_lang":"zh-CN","messages":{"test.one":{"kind":"plain","text":"Modified"}}}`))
				recoverAgain = true
			case "canonical":
				writeSnapshotSource(t, root, canonicalCandidatePath("zh-CN", "lesson/1"), []byte("* Manual\n"))
				recoverAgain = true
			case "source":
				cat.Pages[0].SourceSHA256 = strings.Repeat("0", 64)
				recoverAgain = true
			case "closure":
				writeSnapshotSource(t, root, "locales/zh-CN/content-scope.json", []byte("{}"))
			case "schema", "locale":
				m, _ := decodeCourseMetadata(current)
				if kind == "schema" {
					m.SchemaVersion = 99
				} else {
					m.Locale = "wrong"
				}
				writeSnapshotSource(t, root, courseMetadataRelativePath("zh-CN"), marshalCourseMetadata(t, m))
				recoverAgain = true
			case "receipt":
				b, _ := readCompatibilityFile(root, path)
				writeSnapshotSource(t, root, path, append(b, []byte("{}")...))
			case "archive":
				writeSnapshotSource(t, root, courseHistoryPath("zh-CN", sum(old)), []byte("{}"))
			case "missing-receipt":
				if err := os.Rename(filepath.Join(root, path), filepath.Join(root, "removed-receipt.json")); err != nil {
					t.Fatal(err)
				}
			case "fork":
				b, _ := readCompatibilityFile(root, path)
				var e CourseSEOTransition
				if err := decodeStrictCourseSourceDescriptionReviewJSON(b, &e); err != nil {
					t.Fatal(err)
				}
				e.ReconstructionSHA = strings.Repeat("1", 64)
				e.Identity = courseTransitionIdentity(e)
				writeSnapshotSource(t, root, courseTransitionPath(&e), mustJSON(e))
			}
			if recoverAgain {
				if _, _, err := RecoverCourseMetadataHistory(root, cat, "zh-CN", old, false); err == nil {
					t.Fatal("broken recovery accepted")
				}
			} else if _, err := AuthorizeHistoricalCourseMetadataReference(root, "zh-CN", sum(old), cat); err == nil {
				t.Fatal("broken transition accepted")
			}
		})
	}
}

func TestCourseRefreshNativeReceiptAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "publication-failure"}[fail], func(t *testing.T) {
			root, cat, _ := historicalRecoveryFixture(t)
			if _, _, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true); err != nil {
				t.Fatal(err)
			}
			old, _ := readCompatibilityFile(root, courseMetadataRelativePath("zh-CN"))
			g1 := readSnapshotGlossarySHA(t, root, "qc-001")
			if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, cat); err != nil {
				t.Fatal(err)
			}
			publish := os.Link
			if fail {
				publish = func(string, string) error { return errors.New("fixture publication failure") }
			}
			e, _, err := applyCourseMetadataRefresh(root, cat, CourseMetadataRefreshOptions{Locale: "zh-CN", Provider: "fixture", Model: "fixture", GeneratedAt: "2026-10-09T02:29:24Z", Descriptions: []byte(`{"pages":[{"page_id":"lesson/1","description":"A distinct formal replacement description covering the lesson one content accurately."}]}`)}, publish)
			if fail {
				if err == nil {
					t.Fatal("failure ignored")
				}
				after, _ := readCompatibilityFile(root, courseMetadataRelativePath("zh-CN"))
				if !bytes.Equal(after, old) {
					t.Fatal("rollback failed")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if e.Schema != CourseRefreshReceiptSchema || e.Old.SHA256 != sum(old) {
					t.Fatal("wrong native authority")
				}
				if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, cat); err != nil {
					t.Fatal(err)
				}
				if _, err := AuthorizeHistoricalCourseMetadataReference(root, "zh-CN", sum(old), cat); err != nil {
					t.Fatal(err)
				}
			}
			if err := CheckCourseRefreshTransactionState(root, "zh-CN"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCourseRefreshNativeV2AndCrashGuard(t *testing.T) {
	t.Run("schema-v2", func(t *testing.T) {
		root, cat, _ := compatibilityFixture(t)
		writeCourseSourceDescriptionAsset(t, root, cat, nil)
		writeCourseSourceDescriptionReview(t, root, cat, "source-v2")
		base, err := AssembleCourseMetadata(root, cat, CourseMetadataAssemblyOptions{SchemaVersion: 2, Locale: "zh-CN", Provider: "fixture", Model: "fixture", GeneratedAt: "2026-10-09T00:00:00Z", Descriptions: marshalCourseDescriptions(t, cat)})
		if err != nil {
			t.Fatal(err)
		}
		writeSnapshotSource(t, root, courseMetadataRelativePath("zh-CN"), base)
		writeCourseSourceDescriptionAsset(t, root, cat, map[string]string{cat.Pages[1].ID: "changed canonical description"})
		writeCourseSourceDescriptionReview(t, root, cat, "source-v2-new")
		e, stale, err := ApplyCourseMetadataRefresh(root, cat, CourseMetadataRefreshOptions{Locale: "zh-CN", Provider: "fixture", Model: "fixture", GeneratedAt: "2026-10-09T02:29:24Z", Descriptions: []byte(`{"pages":[{"page_id":"lesson/2","description":"A faithful and distinct localized description for the updated second lesson source."}]}`)})
		if err != nil || len(stale) != 1 || e.SchemaVersion != 2 {
			t.Fatalf("v2 %v %v", stale, err)
		}
		if _, err := AuthorizeHistoricalCourseMetadataReference(root, "zh-CN", sum(base), cat); err != nil {
			t.Fatal(err)
		}
	})
	for _, kind := range []string{"panic", "rollback-failure"} {
		t.Run(kind, func(t *testing.T) {
			root, cat, _ := historicalRecoveryFixture(t)
			if _, _, err := RecoverHistoricalPromotion(root, cat, "zh-CN", "qc-002", true); err != nil {
				t.Fatal(err)
			}
			if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", readSnapshotGlossarySHA(t, root, "qc-001"), cat); err != nil {
				t.Fatal(err)
			}
			publish := func(string, string) error {
				if kind == "panic" {
					panic("fixture process interruption")
				}
				backup := filepath.Join(root, "locales/zh-CN/.course-refresh-staging-transaction/old.json")
				if err := os.Rename(backup, filepath.Join(root, "retained-backup.json")); err != nil {
					t.Fatal(err)
				}
				return errors.New("fixture publication and rollback failure")
			}
			func() {
				defer func() {
					if value := recover(); value != nil && kind != "panic" {
						t.Fatal(value)
					}
				}()
				_, _, err := applyCourseMetadataRefresh(root, cat, CourseMetadataRefreshOptions{Locale: "zh-CN", Provider: "fixture", Model: "fixture", GeneratedAt: "2026-10-09T02:29:24Z", Descriptions: []byte(`{"pages":[{"page_id":"lesson/1","description":"A distinct complete description for the formally refreshed lesson one content."}]}`)}, publish)
				if kind == "rollback-failure" && (err == nil || !strings.Contains(err.Error(), "mutation=unknown")) {
					t.Fatal(err)
				}
			}()
			if err := CheckCourseRefreshTransactionState(root, "zh-CN"); err == nil {
				t.Fatal("missing crash guard")
			}
			if _, err := LoadCourseMetadata(root, "zh-CN", cat); err == nil {
				t.Fatal("loader passed interrupted transaction")
			}
		})
	}
}
