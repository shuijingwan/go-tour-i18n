package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func localeGenerationBundleFixture(t *testing.T) (string, *Catalog) {
	t.Helper()
	root := copyTestLocaleSurfaceTree(t, "uk-UA")
	ensureTestCurrentLocaleAuthority(t, root, "uk-UA", testLanguageAuditAuthority("uk-UA", "Cyrillic", map[string]string{"。": "."}))
	copyTestLanguageAuditMachineAuthority(t, root)
	copyBundleAuthority(t, root, localeGenerationAuthorityPaths)
	copyQualityBundleTestFile(t, filepath.Join(repoRoot(t), "data/locale-surface-reviews/uk-UA/20260921-first-production.md"), filepath.Join(root, "data/locale-surface-reviews/uk-UA/20260921-first-production.md"))
	recordSyntheticUnifiedReview(t, root, "synthetic-locale-generation-review", "uk-UA")
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, catalog
}

func TestLocaleGlossaryGenerationBundleDeterministicAndComplete(t *testing.T) {
	root, catalog := localeGenerationBundleFixture(t)
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
	// Glossary tasks consume the unified corpus transport, not the retired
	// Tour-only generation-context.json. Verify complete current JSON inputs.
	var embedded UnifiedGlossaryBundle
	if err := decodeStrictBundleJSON(files["manifest.json"], &embedded); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 2 || embedded.Schema != "go-learning/unified-glossary-bundle/v1" || embedded.Role != "generation" || embedded.Locale != options.Locale || embedded.Identity != manifest.InputIdentitySHA256 || !reflect.DeepEqual(embedded.ExpectedOutputs, []string{glossaryReviewGlossaryPath(options.Locale)}) {
		t.Fatalf("incomplete unified glossary contract: %+v", embedded)
	}
	corpus, err := LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	var archived GlossarySourceCorpus
	corpusPath := "inputs/" + GlossaryCorpusPath
	if len(files[corpusPath]) == 0 {
		t.Fatalf("missing complete corpus JSON: %s", corpusPath)
	}
	if err := decodeStrictBundleJSON(files[corpusPath], &archived); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&archived, corpus) || embedded.CorpusSHA != corpus.Identity {
		t.Fatal("unified generation corpus was truncated or changed")
	}
	for _, path := range []string{GlossaryCorpusPath, glossaryReviewGlossaryPath(options.Locale), "locales/" + options.Locale + "/locale.json"} {
		original, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(files["inputs/"+path], original) {
			t.Fatalf("generation input incomplete: %s", path)
		}
	}
	if _, err := CheckUnifiedGlossaryBundle(root, first); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyCurrentLocaleGenerationBundle(root, catalog, first); err != nil {
		t.Fatalf("current locale generation bundle rejected: %v", err)
	}
}

func TestLocaleGenerationBundleTaskContracts(t *testing.T) {
	root, catalog := localeGenerationBundleFixture(t)
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
	sourceRoot, catalog := localeGenerationBundleFixture(t)
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
	// Carry the current synthetic review and its immutable archived input,
	// rather than the retired glossary-review receipt family.
	copyLocaleGenerationTestFile(t, sourceRoot, fixtureRoot, GlossaryCorpusPath)
	for _, dir := range []string{"data/unified-glossary-reviews/uk-UA", "data/unified-glossary-reviews/uk-UA-bundles"} {
		copyQualityBundleTestTree(t, filepath.Join(sourceRoot, dir), filepath.Join(fixtureRoot, dir))
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
