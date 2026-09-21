package i18n

import (
	"bytes"
	"encoding/json"
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
		})
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
