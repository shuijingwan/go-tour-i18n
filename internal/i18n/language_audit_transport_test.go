package i18n

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func copyTestLanguageAuditMachineAuthority(t *testing.T, root string) {
	t.Helper()
	source := repoRoot(t)
	if err := languageAuditFixtureRootError(root, source); err != nil {
		t.Fatal(err)
	}
	paths := append(append([]string(nil), LanguageAuditAuthorityPaths...), "go.mod", "go.sum", "internal/i18n/glossary.go", "internal/i18n/glossary_context.go", "internal/i18n/glossary_compatibility.go", "internal/i18n/glossary_semantics.go", "internal/i18n/translation_authorization.go", "internal/i18n/bridge_successor.go")
	for _, p := range paths {
		b, err := os.ReadFile(filepath.Join(source, p))
		if err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func seedLanguageAuditTransportFixture(t *testing.T, root, locale, script string) {
	t.Helper()
	source := repoRoot(t)
	if err := languageAuditFixtureRootError(root, source); err != nil {
		t.Fatal(err)
	}

	copyTestLanguageAuditMachineAuthority(t, root)
	ensureTestCurrentLocaleAuthority(t, root, locale, testLanguageAuditAuthority(locale, script, map[string]string{"。": "."}))
}
func TestLanguageAttentionTypedCandidatesAndFinalGuard(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "locales", "fr-FR")
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, "glossary.yaml"), []byte("locale: fr-FR\nmandatory:\n  Go: Go\nkeep:\n  - Go\n"), 0644)
	seedLanguageAuditTransportFixture(t, root, "fr-FR", "Latin")
	a, err := NewLanguageAttention(root, "fr-FR")
	if err != nil {
		t.Fatal(err)
	}
	candidate := languageaudit.Result{Diagnostics: []languageaudit.Diagnostic{{Class: "source-prose-overlap", Decision: languageaudit.Candidate, Locator: "p/1", Evidence: "Command Prompt"}}}
	if err := a.Add("future-surface", "owner", "unit", UnitKindPage, candidate); err != nil {
		t.Fatal(err)
	}
	if len(a.Candidates) != 1 || a.Candidates[0].Classification != languageaudit.Candidate || a.Candidates[0].Context != "Command Prompt" {
		t.Fatal(a)
	}
	candidate.Diagnostics[0].Decision = languageaudit.Defect
	if err := a.Add("s", "owner", "", "", candidate); err == nil {
		t.Fatal("defect entered Reviewer transport")
	}
	if err := a.Verify(root); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(root, "internal/languageaudit/audit.go"), []byte("implementation drift"), 0644)
	if err := a.Verify(root); err == nil {
		t.Fatal("implementation drift not detected")
	}
}

func isolatedLanguageSurfaceFixture(t *testing.T) string {
	t.Helper()
	return copyTestLocaleSurfaceTree(t, "tr-TR")
}

func copyTestLocaleSurfaceTree(t *testing.T, locale string) string {
	t.Helper()
	source := repoRoot(t)
	root := t.TempDir()
	for _, dir := range []string{"_content", "locales/" + locale, "internal/tour", "production"} {
		copyQualityBundleTestTree(t, filepath.Join(source, dir), filepath.Join(root, dir))
	}
	for _, name := range []string{"data/tour-pages.tsv", "data/tour-examples.tsv", "data/tour-conditional-pages.tsv", "data/course-seo/source-descriptions.json"} {
		if _, err := os.Stat(filepath.Join(source, name)); os.IsNotExist(err) {
			continue
		}
		copyQualityBundleTestFile(t, filepath.Join(source, name), filepath.Join(root, name))
	}
	copyQualityBundleTestTree(t, filepath.Join(source, "data/course-seo/source-description-reviews"), filepath.Join(root, "data/course-seo/source-description-reviews"))
	copyBundleAuthority(t, root, localeSurfaceReviewReviewerAuthorityPaths)
	return root
}

func languageAuditFixtureRootError(root, repository string) error {
	actual, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	source, err := filepath.EvalSymlinks(repository)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(source, actual)
	if err != nil {
		return err
	}
	if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		return fmt.Errorf("fixture authority writes require isolated temporary root")
	}
	return nil
}
func TestLanguageAuditTransportFixtureRejectsRepositoryRoot(t *testing.T) {
	root := repoRoot(t)
	if languageAuditFixtureRootError(root, root) == nil {
		t.Fatal("real root accepted")
	}
	if languageAuditFixtureRootError(filepath.Join(root, "internal"), root) == nil {
		t.Fatal("repo child accepted")
	}
	alias := filepath.Join(t.TempDir(), "repo-alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	if languageAuditFixtureRootError(alias, root) == nil {
		t.Fatal("repository symlink alias accepted")
	}
	if err := languageAuditFixtureRootError(t.TempDir(), root); err != nil {
		t.Fatal(err)
	}
}
