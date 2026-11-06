package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFailedUnifiedGlossaryReviewPreservesFindingAndBlocksGate(t *testing.T) {
	root := t.TempDir()
	path := writeGlossaryReviewTestGlossary(t, root, "zh-CN")
	recordSyntheticUnifiedReview(t, root, "prior-unified-review")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A byte change makes the previous judgment historical, not current.
	current := append(bytes.Clone(before), '\n')
	if err := os.WriteFile(path, current, 0644); err != nil {
		t.Fatal(err)
	}
	bundle, manifest, err := ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	finding := "Finding category: missing-explicit-decision\nTerm: fixture term\nObserved instances: fixture/context\nRevision scope: only this term\nOut of scope: unrelated glossary rules"
	receipt, receiptPath, err := RecordUnifiedGlossaryReview(root, bundle, "failed-unified-review", "synthetic-independent-reviewer", "synthetic-generation", "failed", []string{finding})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Decision != "failed" || len(receipt.Findings) != 1 || receipt.Findings[0] != finding || receipt.InputSHA != manifest.Identity || receipt.BundleSHA != sum(bundle) {
		t.Fatalf("failed judgment/finding/bundle binding lost: %+v", receipt)
	}
	if receiptPath != UnifiedGlossaryReviewPath("zh-CN", "failed-unified-review") {
		t.Fatal("wrong failed receipt path", receiptPath)
	}
	if err := validateUnifiedReviewBundle(root, *receipt); err != nil {
		t.Fatal("failed receipt historical proof", err)
	}
	reviews, err := currentUnifiedReviews(root, "zh-CN")
	if err != nil || len(reviews) != 1 || reviews[0].Decision != "failed" {
		t.Fatalf("current failed judgment not readable: %+v, %v", reviews, err)
	}
	if _, err := RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("FAILED judgment granted language eligibility")
	}
	// Glossary revision input remains available without granting a language PASS.
	generation, _, err := ExportUnifiedGlossaryBundle(root, "zh-CN", "generation")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CheckUnifiedGlossaryBundle(root, generation); err != nil {
		t.Fatal("glossary generation input unavailable after FAILED", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(current, after) {
		t.Fatal("record/export changed glossary", err)
	}
}

// Synthetic registered source authority and independent receipt exist only in
// t.TempDir. They migrate language-work fixtures, never repository evidence.
func recordSyntheticUnifiedReview(t *testing.T, root, id string, locales ...string) {
	t.Helper()
	if err := languageAuditFixtureRootError(root, repoRoot(t)); err != nil {
		t.Fatal(err)
	}
	locale := "zh-CN"
	if len(locales) > 0 {
		locale = locales[0]
	}
	if _, err := os.Lstat(filepath.Join(root, "locales", locale, "locale.json")); os.IsNotExist(err) {
		copyBundleAuthority(t, root, []string{"locales/" + locale + "/locale.json"})
	} else if err != nil {
		t.Fatal(err)
	}
	ensureTestCurrentLocaleAuthority(t, root, locale, testLanguageAuditAuthority(locale, "Han", map[string]string{".": "。"}))
	paths := append([]string{}, glossaryReviewerAuthorityPaths...)
	paths = append(paths, "docs/SITE_V2_WORKFLOW.md", "docs/SITE_V2_ARCHITECTURE.md", "docs/GLOSSARY_COMPATIBILITY.md", "docs/TRANSLATION_WORKFLOW.md")
	paths = append(paths, "docs/CODEX_TRANSLATION.md", "docs/CHATGPT_LANGUAGE_GENERATION.md")
	copyBundleAuthority(t, root, paths)
	path := filepath.Join(root, filepath.FromSlash(GlossaryCorpusPath))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		c := GlossarySourceCorpus{Schema: GlossaryCorpusSchema, Sources: []GlossaryArchiveReference{}, Contributors: []LanguageContributor{{ID: "tour:page:lesson/1", Parser: "present-visible/v1", Package: "tour-v1", Path: "_content/tour/lesson.article", Contexts: []LanguageContext{{ID: "synthetic-language", Text: "Go programming language teaching fixture", Protected: []string{}}}}}}
		c.Contributors[0].SourceSHA = ContributorIdentity(c.Contributors[0])
		c.Identity = CorpusIdentity(c)
		b, _ := json.Marshal(c)
		writeSnapshotSource(t, root, GlossaryCorpusPath, append(b, '\n'))
	} else if err != nil {
		t.Fatal(err)
	}
	b, _, err := ExportUnifiedGlossaryBundle(root, locale, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecordUnifiedGlossaryReview(root, b, id, "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
}
