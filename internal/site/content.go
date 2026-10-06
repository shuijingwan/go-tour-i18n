package site

import (
	"fmt"
	"path"
	"strings"

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
	Resolver *Resolver
	Pages    map[string]ContentPage
	Data     map[string][]byte
}

// LoadContent assembles only complete package artifacts. Tour retains its own
// renderer/Playground and can be supplied to Resolver.Handler at its covered
// routes; shell/Docs renderers consume these exact reconstructed bytes.
func LoadContent(root, locale, origin string) (*Content, error) {
	r, err := Load(root, locale, origin)
	if err != nil {
		return nil, err
	}
	g, l, err := sitecontent.LocalCoverage(root, locale)
	if err != nil {
		return nil, err
	}
	result := &Content{Resolver: r, Pages: map[string]ContentPage{}, Data: map[string][]byte{}}
	for _, completion := range l.Packages {
		if completion.Package == "tour-v1" {
			continue
		}
		files, err := sitecontent.CompletedPackageFiles(root, locale, completion.Package)
		if err != nil {
			return nil, err
		}
		docs, _, err := sitecontent.PackageDocuments(root, g, completion.Package)
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
	return result, nil
}
