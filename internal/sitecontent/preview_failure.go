package sitecontent

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
)

// RenderedExternalLinks uses anchor semantics, never reference definitions or
// URL-looking code/text. It matches the verifier's document-wide anchor set.
func RenderedExternalLinks(b []byte) (map[string]bool, error) {
	d, err := html.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	links := map[string]bool{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					u, e := url.Parse(a.Val)
					if e == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" {
						links[a.Val] = true
					}
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(d)
	return links, nil
}

// MissingPageSourceLinks is a pure representation proof shared by live capture
// and historical finalization/closure readers. It has no lifecycle mutations.
func MissingPageSourceLinks(source, target []byte, format string) ([]string, error) {
	render := func(b []byte) ([]byte, error) {
		body, _, templated, err := PageRenderInput(b)
		if err != nil {
			return nil, err
		}
		if templated {
			return nil, fmt.Errorf("templated TU link proof unsupported")
		}
		if format == "html" {
			return body, nil
		}
		if format != "md" {
			return nil, fmt.Errorf("unsupported preview TU representation")
		}
		md := goldmark.New(goldmark.WithParserOptions(parser.WithHeadingAttribute(), parser.WithAutoHeadingID()), goldmark.WithRendererOptions(gmhtml.WithUnsafe()), goldmark.WithExtensions(extension.Typographer, extension.Linkify, extension.DefinitionList, extension.Table))
		var out bytes.Buffer
		err = md.Convert(body, &out)
		return out.Bytes(), err
	}
	s, err := render(source)
	if err != nil {
		return nil, err
	}
	t, err := render(target)
	if err != nil {
		return nil, err
	}
	sl, err := RenderedExternalLinks(s)
	if err != nil {
		return nil, err
	}
	tl, err := RenderedExternalLinks(t)
	if err != nil {
		return nil, err
	}
	missing := []string{}
	for href := range sl {
		if !tl[href] {
			missing = append(missing, href)
		}
	}
	sort.Strings(missing)
	if len(missing) == 0 {
		return nil, fmt.Errorf("no machine-provable TU content link loss")
	}
	return missing, nil
}

// VerifyFrozenPreviewTUFailure lets all shared QC/package consumers verify a
// consumed invocation without importing the preview server or reviving it.
func (a PageAdapter) VerifyFrozenPreviewTUFailure(target []byte, p *i18n.PreviewTUFailure) error {
	if a.Document.Path != p.Artifact.Unit.SourcePath || a.Document.Route != p.Route || digest(a.Source) != p.Artifact.Unit.SourceSHA256 {
		return fmt.Errorf("frozen preview source/route mismatch")
	}
	missing, err := MissingPageSourceLinks(a.Source, target, strings.TrimPrefix(path.Ext(a.Document.Path), "."))
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(missing, p.MissingLinks) {
		return fmt.Errorf("preview exact TU link defect differs from receipt")
	}
	actual, err := RenderedExternalLinks([]byte(p.ResponseHTML))
	if err != nil {
		return err
	}
	for _, href := range missing {
		if actual[href] {
			return fmt.Errorf("preview response contains claimed missing link")
		}
	}
	return nil
}
