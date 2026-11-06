package sitecontent

import (
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func fixtureLocale(locales []string) string {
	if len(locales) > 0 {
		return locales[0]
	}
	return "zh-CN"
}

// These compatibility tests exercise lifecycle/transport, not translation.
// Source-identical teaching comments are legal only in an English locale; use
// its real metadata and independently construct its complete glossary authority.
func englishCompatibilityLocale(t *testing.T, root string) {
	t.Helper()
	const locale = "en-GB"
	metadata, err := readRegular(repositoryRoot(t), "locales/"+locale+"/locale.json")
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "locales/"+locale+"/locale.json", metadata)
	seedSiteLanguagePolicyFixture(t, root, locale, "Latin")
	writeFixture(t, root, "locales/"+locale+"/glossary.yaml", []byte("locale: "+locale+"\nmandatory:\n  synthetic-test-term: Synthetic test term\n"))
	if _, _, err := i18n.RecordGlossaryReview(root, locale, "fixture-glossary-1", "independent-fixture-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	bundle, _, err := i18n.ExportUnifiedGlossaryBundle(root, locale, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, bundle, "synthetic-unified-review", "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	b, err := Encode(Locale{Schema: LocaleSchema, Locale: locale, Packages: []Completion{}})
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, LocalePath(locale), b)
}
