package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlossaryReviewerBundleDeterministicAndComplete(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	first, manifest, err := ExportGlossaryReviewerBundle(root, "uk-UA", catalog)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := ExportGlossaryReviewerBundle(root, "uk-UA", catalog)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("glossary reviewer bundle is not deterministic: %v", err)
	}
	if manifest.UnitCount != 122 || manifest.PageCount != 103 || manifest.ExampleCount != 19 || manifest.GlossarySHA256 == "" {
		t.Fatalf("manifest=%+v", manifest)
	}
	files, err := ReadTransportBundle(first, 128, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"glossary-review.json", "formal/glossary.yaml", "formal/locale.json", "formal/ui-en.json", "formal/tour-pages.tsv", "formal/tour-examples.tsv"} {
		if _, ok := files[required]; !ok {
			t.Fatalf("bundle missing %s", required)
		}
	}
	var context GlossaryReviewerContext
	if err := json.Unmarshal(files["glossary-review.json"], &context); err != nil {
		t.Fatal(err)
	}
	if len(context.SourceUnits) != 122 || context.SourceUnits[0].Source == "" {
		t.Fatalf("incomplete glossary context: %+v", context)
	}
}

func TestGlossaryReviewerBundleCurrentCheckRejectsStaleGlossary(t *testing.T) {
	sourceRoot := repoRoot(t)
	catalog, err := BuildCatalog(sourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	copyBundleAuthority(t, root, glossaryReviewerAuthorityPaths)
	for _, repositoryPath := range []string{
		"locales/uk-UA/glossary.yaml", "locales/uk-UA/locale.json", "internal/tour/ui/en.json",
		"data/tour-pages.tsv", "data/tour-examples.tsv",
	} {
		copyQualityBundleTestFile(t, filepath.Join(sourceRoot, filepath.FromSlash(repositoryPath)), filepath.Join(root, filepath.FromSlash(repositoryPath)))
	}
	bundle, _, err := ExportGlossaryReviewerBundle(root, "uk-UA", catalog)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyCurrentGlossaryReviewerBundle(root, "uk-UA", catalog, bundle); err != nil {
		t.Fatalf("current bundle rejected: %v", err)
	}
	glossaryPath := filepath.Join(root, "locales", "uk-UA", "glossary.yaml")
	glossary, err := os.ReadFile(glossaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(glossaryPath, append(glossary, []byte("\n# changed after review export\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyCurrentGlossaryReviewerBundle(root, "uk-UA", catalog, bundle); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale glossary reviewer bundle accepted: %v", err)
	}
}
