package site

import (
	"fmt"
	"path"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/assets"
	"github.com/shuijingwan/go-tour-i18n/internal/releasecontract"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// ContentPage supplies concrete, validated localized source to an opt-in
// renderer. It is not an untranslated placeholder or a second locale registry.
type ContentPage struct {
	Route   string
	Path    string
	Format  string
	Content []byte
}
type Content struct {
	baselineRepository string
	baselineCommit     string
	Companies          []map[string]any
	locale             string
	Resolver           *Resolver
	Pages              map[string]ContentPage
	Data               map[string][]byte
	Auxiliary          map[string][]byte
	Resources          map[string][]byte
	breadcrumbSources  map[string]ContentPage
	markdownSources    map[string][]byte
	markdownSourceData map[string][]byte
}

// LoadContent assembles only complete package artifacts. Tour retains its own
// renderer/Playground and can be supplied to Resolver.Handler at its covered
// routes; shell/Docs renderers consume these exact reconstructed bytes.
func LoadContent(root, locale, origin string) (*Content, error) {
	g, l, files, err := sitecontent.LocalCoverageFiles(root, locale)
	if err != nil {
		return nil, err
	}
	return contentFromCoverage(root, locale, origin, g, l, files)
}

func contentFromCoverage(root, locale, origin string, g *sitecontent.Global, l *sitecontent.Locale, payloads map[string]map[string][]byte) (*Content, error) {
	formal, err := sitecontent.LocaleOrigin(root, locale)
	if err != nil {
		return nil, err
	}
	if strings.TrimSuffix(formal, "/") != strings.TrimSuffix(origin, "/") {
		return nil, fmt.Errorf("locale hostname differs from deployment identity")
	}
	aliases, err := sitecontent.RedirectContracts(root, g)
	if err != nil {
		return nil, err
	}
	r, err := newResolver(g, l, origin, aliases)
	if err != nil {
		return nil, err
	}
	result := &Content{locale: locale, Resolver: r, Pages: map[string]ContentPage{}, Data: map[string][]byte{}, baselineRepository: g.UpstreamRepository, baselineCommit: g.UpstreamCommit}
	if g.ShellAdoption != nil {
		result.Companies, err = sitecontent.FrozenHomepageCompanies(root)
		if err != nil {
			return nil, err
		}
	}
	for _, completion := range l.Packages {
		if completion.Package == "tour-v1" {
			continue
		}
		files := payloads[completion.Package]
		docs, _, err := sitecontent.CurrentPackageDocuments(root, g, completion.Package)
		if err != nil {
			return nil, err
		}
		for _, d := range docs {
			b, ok := files[d.Path]
			if !ok {
				return nil, fmt.Errorf("completed source missing reconstructed bytes")
			}
			if d.Kind == "data" {
				result.Data[d.Path] = b
				continue
			}
			if !r.Covered(d.Route) || result.Pages[d.Route].Route != "" {
				return nil, fmt.Errorf("page route lacks exact canonical coverage")
			}
			result.Pages[d.Route] = ContentPage{Route: d.Route, Path: d.Path, Format: strings.TrimPrefix(path.Ext(d.Path), "."), Content: b}
		}
	}
	if err := result.bindMarkdownSources(root); err != nil {
		return nil, err
	}
	if err := result.bindBreadcrumbSources(root); err != nil {
		return nil, err
	}
	result.Auxiliary, err = HomepageAuxiliary(root, result)
	if err != nil {
		return nil, err
	}
	result.Resources = map[string][]byte{}
	paths, err := assets.Paths(root)
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		b, err := releasecontract.ReadRegular(root, "_content/"+p)
		if err != nil {
			return nil, err
		}
		result.Resources["/"+p] = b
	}
	return result, nil
}
