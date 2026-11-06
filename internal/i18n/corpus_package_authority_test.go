package i18n

import (
	"strings"
	"testing"
)

func TestCorpusExplicitRegisteredPagePackageAuthority(t *testing.T) {
	ref := GlossaryArchiveReference{Path: "data/content-package-adoptions/receipts/" + strings.Repeat("a", 64) + ".json", SHA256: strings.Repeat("b", 64)}
	c := GlossarySourceCorpus{Schema: GlossaryCorpusSchema, PackageAuthorities: []GlossaryCorpusPackageAuthority{{Package: "future-pages", Authority: ref, SourcePaths: []string{"_content/new/page.html"}}}, Sources: []GlossaryArchiveReference{ref}, Contributors: []LanguageContributor{{ID: "source:_content/new/page.html", Parser: "go-learning/content-units/v2", Package: "future-pages", Path: "_content/new/page.html", Route: "/new/page", Contexts: []LanguageContext{{ID: "context", Text: "Synthetic source prose."}}}}}
	c.Contributors[0].SourceSHA = ContributorIdentity(c.Contributors[0])
	c.Identity = CorpusIdentity(c)
	if err := ValidateGlossaryCorpus(&c); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*GlossarySourceCorpus){
		func(c *GlossarySourceCorpus) { c.PackageAuthorities = nil },
		func(c *GlossarySourceCorpus) { c.Sources = nil },
		func(c *GlossarySourceCorpus) {
			c.PackageAuthorities[0].SourcePaths = []string{"_content/different.html"}
		},
		func(c *GlossarySourceCorpus) { c.Contributors[0].Structured = true },
		func(c *GlossarySourceCorpus) {
			c.PackageAuthorities[0].SourcePaths = append(append([]string{}, c.PackageAuthorities[0].SourcePaths...), "_content/new/unrepresented.html")
		},
	} {
		copy := c
		copy.PackageAuthorities = append([]GlossaryCorpusPackageAuthority{}, c.PackageAuthorities...)
		copy.Contributors = append([]LanguageContributor{}, c.Contributors...)
		change(&copy)
		copy.Contributors[0].SourceSHA = ContributorIdentity(copy.Contributors[0])
		copy.Identity = CorpusIdentity(copy)
		if err := ValidateGlossaryCorpus(&copy); err == nil {
			t.Fatal("unbound source package accepted")
		}
	}
}
