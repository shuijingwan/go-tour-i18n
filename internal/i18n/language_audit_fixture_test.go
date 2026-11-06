package i18n

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
)

// Reuse the integration profile's real spacing/punctuation rules. Parameters
// are explicit fixture data, never a production locale-name policy fallback.
func testLanguageAuditAuthority(locale, script string, sentenceEnd map[string]string) languageaudit.Authority {
	return languageaudit.Authority{Schema: languageaudit.PolicySchema, Version: 1, Policy: languageaudit.Policy{
		Locale: locale, Profile: "explicit-test-policy", ProseScripts: []string{script},
		Spacing:     []languageaudit.SpacingRule{{ProseScript: script, TechnicalScript: "Latin", Kinds: []languageaudit.Kind{languageaudit.Link, languageaudit.Code, languageaudit.Identity}}},
		SentenceEnd: sentenceEnd, SentenceTerminators: ".!?。！？", ResidualMinWords: 1,
	}}
}

// Simulate completed initialization only for current-validation fixture roots.
// Policy is explicit caller data; existing policy/metadata is never replaced.
func ensureTestCurrentLocaleAuthority(t *testing.T, root, locale string, authority languageaudit.Authority) {
	t.Helper()
	if err := languageAuditFixtureRootError(root, repoRoot(t)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLocaleName(locale); err != nil {
		t.Fatal(err)
	}
	if err := authority.Validate(locale); err != nil {
		t.Fatal(err)
	}
	path := root
	for _, part := range []string{"locales", locale} {
		path = filepath.Join(path, part)
		st, err := os.Lstat(path)
		if os.IsNotExist(err) {
			if err := os.Mkdir(path, 0755); err != nil {
				t.Fatal(err)
			}
		} else if err != nil {
			t.Fatal(err)
		} else if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			t.Fatal("fixture directory must not be a symlink", path)
		}
	}
	path = filepath.Join(path, "locale.json")
	st, err := os.Lstat(path)
	if os.IsNotExist(err) {
		writeInitializableLocale(t, root, locale, Locale{Locale: locale, LanguageName: "Fixture language", EnglishName: "Fixture language", HTMLLang: locale, Phase: "scaffold", TranslationUnit: "present.Section"})
	} else if err != nil {
		t.Fatal(err)
	} else {
		if !st.Mode().IsRegular() {
			t.Fatal("fixture metadata must be regular", path)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(b, &fields); err != nil {
			t.Fatal(err)
		}
		if policy, ok := fields["language_audit"]; ok && string(policy) != "null" {
			if _, _, err := LoadLanguageAuditPolicy(root, locale, nil); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	writeTestLanguageAuditAuthority(t, root, locale, authority)
}

// Native source/candidate provenance is copied exactly; current validation runs
// only inside the fixture. No real locale is initialized or validated here.
func isolatedTourValidationFixture(t *testing.T, locale string, authority languageaudit.Authority) (string, *Catalog) {
	t.Helper()
	source := repoRoot(t)
	catalog, err := BuildCatalog(source)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	paths := map[string]bool{"locales/" + locale + "/locale.json": true, "locales/" + locale + "/glossary.yaml": true}
	for _, p := range catalog.Pages {
		paths[filepath.ToSlash(filepath.Join("_content/tour", p.Article))] = true
	}
	for _, e := range catalog.Examples {
		paths[e.SourcePath] = true
	}
	for path := range paths {
		copyQualityBundleTestFile(t, filepath.Join(source, filepath.FromSlash(path)), filepath.Join(root, filepath.FromSlash(path)))
	}
	ensureTestCurrentLocaleAuthority(t, root, locale, authority)
	return root, catalog
}

// Only update an existing fixture metadata file, preserving every other field.
// Refuse repository roots/children and symlink components before any writes.
// Atomic replacement also keeps a fixture hardlink from mutating its source.
func writeTestLanguageAuditAuthority(t *testing.T, root, locale string, authority languageaudit.Authority) {
	t.Helper()
	if err := languageAuditFixtureRootError(root, repoRoot(t)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLocaleName(locale); err != nil {
		t.Fatal(err)
	}
	if err := authority.Validate(locale); err != nil {
		t.Fatal(err)
	}
	path := root
	for _, part := range []string{"locales", locale, "locale.json"} {
		path = filepath.Join(path, part)
		st, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode()&os.ModeSymlink != 0 {
			t.Fatal("fixture metadata symlink forbidden", path)
		}
	}
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() {
		t.Fatal("fixture metadata must be regular", err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(original, &fields); err != nil {
		t.Fatal(err)
	}
	var actualLocale string
	if err := json.Unmarshal(fields["locale"], &actualLocale); err != nil || actualLocale != locale {
		t.Fatal("fixture locale mismatch", err)
	}
	fields["language_audit"], err = json.Marshal(authority)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	updated = append(updated, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".language-audit-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(st.Mode().Perm()); err != nil {
		_ = tmp.Close()
		t.Fatal(err)
	}
	if _, err := tmp.Write(updated); err != nil {
		_ = tmp.Close()
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		t.Fatal(err)
	}
}

func TestLanguageAuditFixturePolicyPreservesMetadataAndRepository(t *testing.T) {
	source := repoRoot(t)
	before := map[string][]byte{}
	for _, locale := range []string{"zh-CN", "tr-TR"} {
		path := filepath.Join(source, "locales", locale, "locale.json")
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		before[path] = b
	}
	t.Cleanup(func() {
		for path, want := range before {
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(want, got) {
				t.Errorf("real locale metadata changed: %s: %v", path, err)
			}
		}
	})
	root := writeStatusFixtureWithoutLanguageAudit(t, "unit_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\n")
	path := filepath.Join(root, "locales", "zh-CN", "locale.json")
	if _, _, err := LoadLanguageAuditPolicy(root, "zh-CN", nil); err == nil {
		t.Fatal("fixture missing explicit policy unexpectedly accepted")
	}
	var original map[string]json.RawMessage
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &original); err != nil {
		t.Fatal(err)
	}
	authority := testLanguageAuditAuthority("zh-CN", "Han", map[string]string{".": "。"})
	writeTestLanguageAuditAuthority(t, root, "zh-CN", authority)
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var updated map[string]json.RawMessage
	if err := json.Unmarshal(first, &updated); err != nil {
		t.Fatal(err)
	}
	delete(updated, "language_audit")
	if !reflect.DeepEqual(original, updated) {
		t.Fatal("unrelated metadata fields changed")
	}
	writeTestLanguageAuditAuthority(t, root, "zh-CN", authority)
	second, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("test policy writer is not deterministic", err)
	}
	policy, _, err := LoadLanguageAuditPolicy(root, "zh-CN", nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := languageaudit.Document{Owner: "fixture", Blocks: []languageaudit.Block{{Locator: "p1", Context: "paragraph", Pieces: []languageaudit.Piece{{Text: "文字", Kind: languageaudit.Prose}, {Text: "Go", Kind: languageaudit.Code}, {Text: ".", Kind: languageaudit.Prose}}}}}
	r, err := languageaudit.Scan(policy, languageaudit.Document{}, doc)
	if err != nil || languageaudit.Blocking(r) == nil {
		t.Fatal("test profile disabled the deterministic scanner", r, err)
	}
}

func TestLanguageAuditCurrentLocaleFixtureInitializesMissingMetadata(t *testing.T) {
	root := t.TempDir()
	authority := testLanguageAuditAuthority("test-LOCALE", "Latin", map[string]string{"。": "."})
	ensureTestCurrentLocaleAuthority(t, root, "test-LOCALE", authority)
	path := filepath.Join(root, "locales/test-LOCALE/locale.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ensureTestCurrentLocaleAuthority(t, root, "test-LOCALE", authority)
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("current fixture initialization changed metadata on replay", err)
	}
	if _, _, err := LoadLanguageAuditPolicy(root, "test-LOCALE", nil); err != nil {
		t.Fatal(err)
	}
}

func TestLanguageAuditFutureCardFixtureAdapterPreservesCodeBoundary(t *testing.T) {
	card := futureCard{ID: "fixture/card", Href: "https://example.invalid", Heading: "Fixture", Message: "package main\n// Return the value.\nfunc main(){ println(\"ASCII . literal\") }\n"}
	target := card
	target.Message = strings.Replace(card.Message, "Return the value.", "返回值。", 1)
	b, err := xml.Marshal(target)
	if err != nil {
		t.Fatal(err)
	}
	adapter := validationReopenAdapter{futureCardAdapter: futureCardAdapter{source: card}, code: true}
	src, dst, err := adapter.LanguageAuditDocuments(b)
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, block := range dst.Blocks {
		for _, piece := range block.Pieces {
			text.WriteString(piece.Text)
		}
	}
	if !strings.Contains(text.String(), "返回值。") || strings.Contains(text.String(), "func main") || strings.Contains(text.String(), "ASCII . literal") || strings.Contains(text.String(), card.Href) {
		t.Fatal("fixture view lost prose/code boundary", text.String())
	}
	result, err := languageaudit.Scan(testLanguageAuditAuthority("fixture-LOCALE", "Han", map[string]string{".": "。"}).Policy, src, dst)
	if err != nil || languageaudit.Blocking(result) != nil {
		t.Fatal("valid teaching-comment view rejected", result, err)
	}
}
