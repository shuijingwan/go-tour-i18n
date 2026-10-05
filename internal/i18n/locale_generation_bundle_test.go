package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLocaleGlossaryGenerationBundleDeterministicAndComplete(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	options := LocaleGenerationBundleOptions{Locale: "uk-UA", TaskKind: LocaleGenerationTaskGlossary}
	first, manifest, err := ExportLocaleGenerationBundle(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := ExportLocaleGenerationBundle(root, catalog, options)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("locale generation bundle is not deterministic: %v", err)
	}
	if manifest.Locale != "uk-UA" || manifest.TaskKind != LocaleGenerationTaskGlossary || manifest.InputIdentitySHA256 == "" {
		t.Fatalf("manifest=%+v", manifest)
	}
	files, err := ReadTransportBundle(first, 512, 256<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		t.Fatal(err)
	}
	var context LocaleGenerationContext
	if err := json.Unmarshal(files["generation-context.json"], &context); err != nil {
		t.Fatal(err)
	}
	if len(context.SourceUnits) != 122 || len(context.ExpectedOutputs) != 1 || context.ExpectedOutputs[0] != "glossary.yaml" {
		t.Fatalf("incomplete locale generation context: %+v", context)
	}
	if _, err := VerifyCurrentLocaleGenerationBundle(root, catalog, first); err != nil {
		t.Fatalf("current locale generation bundle rejected: %v", err)
	}
}

func TestLocaleGenerationBundleTaskContracts(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		task, reviewID string
		outputs        []string
		required       string
	}{
		{task: LocaleGenerationTaskLocaleAssets, outputs: []string{"ui.json", "article-metadata.json"}, required: "formal/ui-en.json"},
		{task: LocaleGenerationTaskSurfaceReplacement, reviewID: "20260921-first-production", outputs: []string{"replacements.json"}, required: "formal/surface-review.json"},
	} {
		t.Run(test.task, func(t *testing.T) {
			bundle, manifest, err := ExportLocaleGenerationBundle(root, catalog, LocaleGenerationBundleOptions{Locale: "uk-UA", TaskKind: test.task, ReviewID: test.reviewID})
			if err != nil {
				t.Fatal(err)
			}
			files, err := ReadTransportBundle(bundle, 512, 256<<20)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := files[test.required]; !ok {
				t.Fatalf("bundle missing %s", test.required)
			}
			var context LocaleGenerationContext
			if err := json.Unmarshal(files["generation-context.json"], &context); err != nil {
				t.Fatal(err)
			}
			if context.TaskKind != test.task || !localeBundleStringsEqual(context.ExpectedOutputs, test.outputs) || manifest.ReviewID != test.reviewID {
				t.Fatalf("context=%+v manifest=%+v", context, manifest)
			}
			if test.task == LocaleGenerationTaskLocaleAssets {
				expectedArticles := catalogArticleSet(catalog)
				if len(context.ArticleSources) != len(expectedArticles) {
					t.Fatalf("article sources=%d want=%d", len(context.ArticleSources), len(expectedArticles))
				}
				seen := make(map[string]bool, len(context.ArticleSources))
				for _, source := range context.ArticleSources {
					if _, ok := expectedArticles[source.Article]; !ok {
						t.Fatalf("unexpected article source %+v", source)
					}
					if seen[source.Article] {
						t.Fatalf("duplicate article source %q", source.Article)
					}
					seen[source.Article] = true
					wantPath := filepath.ToSlash(filepath.Join("_content", "tour", source.Article))
					wantTitle, wantSubtitle, err := sourceArticleHeader(filepath.Join(root, filepath.FromSlash(wantPath)))
					if err != nil {
						t.Fatal(err)
					}
					if source.SourcePath != wantPath || source.SourceTitle != wantTitle || source.SourceSubtitle != wantSubtitle {
						t.Fatalf("article source mismatch: got=%+v want_path=%q want_title=%q want_subtitle=%q", source, wantPath, wantTitle, wantSubtitle)
					}
					if source.Article == "methods.article" && source.SourceTitle == "Methods" {
						t.Fatalf("article root title was replaced by Section title: %+v", source)
					}
				}
			}
		})
	}
}

func TestLocaleGenerationBundleLocaleAssetsArticleHeaderChangeMakesOldBundleStale(t *testing.T) {
	sourceRoot := repoRoot(t)
	catalog, err := BuildCatalog(sourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	options := LocaleGenerationBundleOptions{Locale: "uk-UA", TaskKind: LocaleGenerationTaskLocaleAssets}
	bundle, manifest, err := ExportLocaleGenerationBundle(sourceRoot, catalog, options)
	if err != nil {
		t.Fatal(err)
	}

	fixtureRoot := t.TempDir()
	for _, file := range manifest.Files {
		if file.RepositoryPath != "" {
			copyLocaleGenerationTestFile(t, sourceRoot, fixtureRoot, file.RepositoryPath)
		}
	}
	receipts, err := filepath.Glob(filepath.Join(sourceRoot, "data", "glossary-reviews", "uk-UA", "*.review.json"))
	if err != nil || len(receipts) == 0 {
		t.Fatalf("find uk-UA Glossary Review receipt: files=%v err=%v", receipts, err)
	}
	for _, receipt := range receipts {
		relative, err := filepath.Rel(sourceRoot, receipt)
		if err != nil {
			t.Fatal(err)
		}
		copyLocaleGenerationTestFile(t, sourceRoot, fixtureRoot, filepath.ToSlash(relative))
	}
	for article := range catalogArticleSet(catalog) {
		copyLocaleGenerationTestFile(t, sourceRoot, fixtureRoot, filepath.ToSlash(filepath.Join("_content", "tour", article)))
	}
	if _, err := VerifyCurrentLocaleGenerationBundle(fixtureRoot, catalog, bundle); err != nil {
		t.Fatalf("copied current locale-assets bundle rejected: %v", err)
	}

	methodsPath := filepath.Join(fixtureRoot, "_content", "tour", "methods.article")
	data, err := os.ReadFile(methodsPath)
	if err != nil {
		t.Fatal(err)
	}
	_, subtitle, err := sourceArticleHeader(methodsPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(data, []byte(subtitle), []byte(subtitle+" Changed for bundle identity test."), 1)
	if bytes.Equal(changed, data) {
		t.Fatal("failed to change article root subtitle")
	}
	if err := os.WriteFile(methodsPath, changed, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyCurrentLocaleGenerationBundle(fixtureRoot, catalog, bundle); err == nil {
		t.Fatal("old locale-assets bundle remained current after article root subtitle changed")
	}
}

func copyLocaleGenerationTestFile(t *testing.T, sourceRoot, targetRoot, relative string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("read fixture source %s: %v", relative, err)
	}
	target := filepath.Join(targetRoot, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatalf("create fixture directory for %s: %v", relative, err)
	}
	if err := os.WriteFile(target, data, 0644); err != nil {
		t.Fatalf("write fixture %s: %v", relative, err)
	}
}

func localeBundleStringsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
