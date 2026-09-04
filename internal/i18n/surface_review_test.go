package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocaleSurfaceReviewAGateFailsClosedAndStales(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil || !strings.Contains(err.Error(), "gate missing") {
		t.Fatalf("missing gate error=%v", err)
	}
	gate, path, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "review-1", "reviewer", catalog)
	if err != nil {
		t.Fatal(err)
	}
	if gate.Stage != localeSurfaceReviewAStage || gate.Decision != "passed" || gate.Inputs.CourseMetadataSHA256 == "" || !strings.HasSuffix(filepath.ToSlash(path), "review-1.a-gate.json") {
		t.Fatalf("recorded gate is incomplete: %+v path=%s", gate, path)
	}
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err != nil {
		t.Fatalf("current gate rejected: %v", err)
	}
	for _, path := range []string{
		"internal/tour/ui/en.json", "internal/tour/ui/zz-ZZ.json", "locales/zz-ZZ/glossary.yaml", "locales/zz-ZZ/article-metadata.json", "locales/zz-ZZ/course-metadata.json",
		"internal/tour/languages.go", "internal/tour/project.go", "internal/tour/seo.go",
		"production/identity.json",
	} {
		original, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), append(original, 'x'), 0644); err != nil {
			t.Fatal(err)
		}
		if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil || !strings.Contains(err.Error(), "stale") {
			t.Fatalf("%s did not stale gate: %v", path, err)
		}
		if err := os.WriteFile(filepath.Join(root, path), original, 0644); err != nil {
			t.Fatal(err)
		}
	}
	changed := *catalog
	changed.Pages = append([]Page(nil), catalog.Pages...)
	changed.Pages[0].Source = []byte("changed source")
	if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", &changed); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("catalog/source change did not stale gate: %v", err)
	}
}

func TestLocaleSurfaceReviewAGateRejectsMalformedWrongLocaleAndDecision(t *testing.T) {
	for _, body := range []string{
		"{", `{"schema_version":1,"locale":"other","review_id":"r","stage":"locale-level-language-quality-review","decision":"passed","reviewer":"r"}`,
		`{"schema_version":1,"locale":"zz-ZZ","review_id":"r","stage":"locale-level-language-quality-review","decision":"failed","reviewer":"r"}`,
	} {
		root, catalog := surfaceReviewTestRoot(t)
		dir := filepath.Join(root, "data", "locale-surface-reviews", "zz-ZZ")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "r.a-gate.json"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
		if err := RequireCurrentLocaleSurfaceReviewA(root, "zz-ZZ", catalog); err == nil {
			t.Fatal("invalid gate accepted")
		}
	}
}

func surfaceReviewTestRoot(t *testing.T) (string, *Catalog) {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"internal/tour/ui/en.json": "en", "internal/tour/ui/zz-ZZ.json": "target", "locales/zz-ZZ/glossary.yaml": "glossary", "locales/zz-ZZ/article-metadata.json": "article", "locales/zz-ZZ/course-metadata.json": "course", "internal/tour/languages.go": "languages", "internal/tour/project.go": "project", "internal/tour/seo.go": "seo", "production/identity.json": "identity",
	}
	for path, text := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root, &Catalog{Pages: []Page{{ID: "lesson/1", Source: []byte("source"), SourceSHA256: sum([]byte("source"))}}}
}
