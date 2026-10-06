package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"sort"
	"strings"
)

// BuildUnifiedCorpus consumes the registered source closure through each
// surface's parser. The corpus identity excludes raw machine-only source bytes.
func BuildUnifiedCorpus(root string) (*i18n.GlossarySourceCorpus, error) {
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	cat, err := checkedCatalog(root)
	if err != nil {
		return nil, err
	}
	cs, err := i18n.TourLanguageContributors(root, cat)
	if err != nil {
		return nil, err
	}
	sources := map[string]bool{}
	for _, c := range cs {
		sources[c.Path] = true
	}
	sources[GlobalPath] = true
	sources[SnapshotPath] = true
	for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
		docs, _, err := PackageDocuments(root, g, pkg)
		if err != nil {
			return nil, err
		}
		for _, d := range docs {
			c, err := projectedDocumentContributor(d, pkg)
			if err != nil {
				return nil, err
			}
			cs = append(cs, c)
			if pkg == "site-v2-shell" {
				sources[d.Path] = true
			}
		}
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	c := &i18n.GlossarySourceCorpus{Schema: i18n.GlossaryCorpusSchema, Contributors: cs, Sources: []i18n.GlossaryArchiveReference{}}
	paths := []string{}
	for p := range sources {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		b, err := readRegular(root, p)
		if err != nil {
			return nil, err
		}
		c.Sources = append(c.Sources, i18n.GlossaryArchiveReference{Path: p, SHA256: digest(b)})
	}
	c.Identity = i18n.CorpusIdentity(*c)
	if err := i18n.ValidateGlossaryCorpus(c); err != nil {
		return nil, err
	}
	return c, nil
}
func CheckUnifiedCorpus(root string) error {
	want, err := BuildUnifiedCorpus(root)
	if err != nil {
		return err
	}
	current, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return err
	}
	if identity(want) != identity(current) {
		return fmt.Errorf("unified glossary corpus projection stale")
	}
	return nil
}
func requireUnifiedReview(root, locale string) error {
	if err := CheckUnifiedCorpus(root); err != nil {
		return err
	}
	_, err := i18n.RequireUnifiedGlossaryReview(root, locale)
	return err
}

// Projection excludes machine-only gaps; compact locked tokens remain terminology context.
func projectedDocumentContributor(d Document, pkg string) (i18n.LanguageContributor, error) {
	c := i18n.LanguageContributor{ID: "source:" + d.Path, Parser: UnitContract, Package: pkg, Path: d.Path, Route: d.Route, Structured: pkg == "site-v2-shell" || d.Kind == "data", Contexts: []i18n.LanguageContext{}}
	for _, u := range d.Units {
		text := u.Source
		locked := []string{}
		for _, p := range u.Protected {
			text = strings.ReplaceAll(text, p.Token, "")
			if !strings.Contains(p.Raw, "\n") && len(p.Raw) < 256 {
				locked = append(locked, p.Raw)
			}
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		// Context identity is projected text/protection, independent of raw code.
		context := strings.Split(u.Locator, ":")[0]
		if d.Kind == "data" {
			// YAML item URLs locate restoration slots, but are not terminology.
			// Corpus coordinates bind the approved visible field and its language.
			context = "data/" + context[strings.LastIndex(context, "/")+1:]
		}
		id := identity(struct{ Locator, Text string }{context, text})
		c.Contexts = append(c.Contexts, i18n.LanguageContext{ID: id, Text: text, Protected: locked})
	}
	sort.Slice(c.Contexts, func(i, j int) bool { return c.Contexts[i].ID < c.Contexts[j].ID })
	// Repeated teaching prose has occurrence identity within the contributor.
	for i := range c.Contexts {
		if i > 0 && c.Contexts[i].ID <= c.Contexts[i-1].ID {
			c.Contexts[i].ID = c.Contexts[i-1].ID + "x"
		}
	}
	if len(c.Contexts) == 0 {
		return i18n.LanguageContributor{}, fmt.Errorf("empty language contributor %s", d.Path)
	}
	c.SourceSHA = i18n.ContributorIdentity(c)
	return c, nil
}
