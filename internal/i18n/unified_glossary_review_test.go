package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Synthetic registered source authority and independent receipt exist only in
// t.TempDir. They migrate language-work fixtures, never repository evidence.
func recordSyntheticUnifiedReview(t *testing.T, root, id string) {
	t.Helper()
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
	b, _, err := ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecordUnifiedGlossaryReview(root, b, id, "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
}
