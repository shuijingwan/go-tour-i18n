// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"golang.org/x/net/html"
)

func TestRenderedRootDirectionFollowsLocaleProfile(t *testing.T) {
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		locale, catalogLocale, direction string
	}{
		{locale: "ar", catalogLocale: "en", direction: "rtl"},
		{locale: "de-DE", catalogLocale: "de-DE", direction: "ltr"},
	} {
		t.Run(test.locale, func(t *testing.T) {
			catalog, err := ui.Load(test.catalogLocale)
			if err != nil {
				t.Fatal(err)
			}
			catalog.Locale = test.locale
			catalog.HTMLLang = test.locale
			localizedMetadata := metadata
			localizedMetadata.Locale = test.locale
			for name, render := range map[string]func(ui.Catalog, SiteMetadata) ([]byte, error){
				"home": renderHome,
				"tour": renderIndex,
			} {
				page, err := render(catalog, localizedMetadata)
				if err != nil {
					t.Fatalf("render %s: %v", name, err)
				}
				want := `<html lang="` + test.locale + `" dir="` + test.direction + `"`
				if !strings.Contains(string(page), want) {
					t.Errorf("%s root does not contain %q", name, want)
				}
			}
		})
	}
}

func TestWritingDirectionFailsClosed(t *testing.T) {
	profile := localeProfiles["ar"]
	invalid := profile
	invalid.Direction = ""
	localeProfiles["ar"] = invalid
	t.Cleanup(func() { localeProfiles["ar"] = profile })

	catalog, err := ui.Load("en")
	if err != nil {
		t.Fatal(err)
	}
	catalog.Locale = "ar"
	catalog.HTMLLang = "ar"
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newPageTemplateData(catalog, metadata); err == nil || !strings.Contains(err.Error(), "unsupported writing direction") {
		t.Fatalf("missing writing direction error = %v", err)
	}
}

func TestRTLLayoutKeepsProgrammingSurfacesLTR(t *testing.T) {
	css, err := fs.ReadFile(contentTour, "tour/static/css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	styles := string(css)

	if _, ok := cssDeclarations(styles, "[dir='rtl'] ul"); ok {
		t.Fatal("app.css must not apply RTL indentation to every ul")
	}
	for _, selector := range []string{"[dir='rtl'] .container ul", "[dir='rtl'] .slide-content ul"} {
		declarations := mustCSSDeclarations(t, styles, selector)
		assertCSSDeclaration(t, selector, declarations, "padding-right", "32px")
		assertCSSDeclaration(t, selector, declarations, "padding-left", "0")
	}

	menu := mustCSSDeclarations(t, styles, ".header-language-menu ul")
	assertCSSDeclaration(t, ".header-language-menu ul", menu, "padding", "4px 0 0")
	toc := mustCSSDeclarations(t, styles, ".toc *")
	assertCSSDeclaration(t, ".toc *", toc, "padding", "0")

	ltrList := mustCSSDeclarations(t, styles, "ul")
	assertCSSDeclaration(t, "ul", ltrList, "padding-left", "32px")
	if got := ltrList["padding-right"]; got != "" {
		t.Errorf("LTR ul padding-right = %q, want unset", got)
	}

	for _, selector := range []string{
		"pre",
		"code",
		"#file-editor",
		"#file-editor > textarea",
		"#file-editor .CodeMirror",
		"#file-editor .CodeMirror-scroll",
		"#file-editor .CodeMirror-lines",
		"#file-editor .CodeMirror-gutters",
		".output > pre",
	} {
		declarations := mustCSSDeclarations(t, styles, selector)
		assertCSSDeclaration(t, selector, declarations, "direction", "ltr")
		assertCSSDeclaration(t, selector, declarations, "text-align", "left")
		assertCSSDeclaration(t, selector, declarations, "unicode-bidi", "isolate")
	}
	for _, selector := range []string{
		"[dir='rtl'] .header-language-menu",
		"[dir='rtl'] .toc",
		"[dir='rtl'] .site-links",
		"[dir='rtl'] .output .system",
	} {
		if _, ok := cssDeclarations(styles, selector); !ok {
			t.Errorf("app.css is missing component rule %q", selector)
		}
	}

	values, err := fs.ReadFile(contentTour, "tour/static/js/values.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(values), "direction: 'ltr'") {
		t.Error("CodeMirror configuration does not force LTR editing")
	}

	directives, err := fs.ReadFile(contentTour, "tour/static/js/directives.js")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(directives), "document.documentElement.dir === 'rtl' ? 'left' : 'right'"); got != 2 {
		t.Fatalf("RTL-aware TOC slide direction occurrences = %d, want 2", got)
	}

	editor, err := fs.ReadFile(contentTour, "tour/static/partials/editor.html")
	if err != nil {
		t.Fatal(err)
	}
	document, err := html.Parse(strings.NewReader(string(editor)))
	if err != nil {
		t.Fatal(err)
	}
	textarea := findHTMLElement(document, "textarea", "")
	if textarea == nil || htmlAttribute(textarea, "dir") != "ltr" || !hasHTMLAttribute(textarea, "ui-codemirror") {
		t.Error("editor textarea does not declare LTR CodeMirror source direction")
	}
	previous := findHTMLElement(document, "a", "prev-page")
	next := findHTMLElement(document, "a", "next-page")
	if previous == nil || next == nil {
		t.Fatal("editor previous/current/next navigation is incomplete")
	}
	if got := rtlTestNodeText(previous); got != "<" || htmlAttribute(previous, "ng-click") != "prevPageClick($event)" {
		t.Errorf("previous navigation = text %q action %q", got, htmlAttribute(previous, "ng-click"))
	}
	if got := rtlTestNodeText(next); got != ">" || htmlAttribute(next, "ng-click") != "nextPageClick($event)" {
		t.Errorf("next navigation = text %q action %q", got, htmlAttribute(next, "ng-click"))
	}
}

func cssDeclarations(styles, target string) (map[string]string, bool) {
	declarations := make(map[string]string)
	found := false
	for offset := 0; offset < len(styles); {
		open := strings.IndexByte(styles[offset:], '{')
		if open < 0 {
			break
		}
		open += offset
		start := strings.LastIndexAny(styles[:open], "{}") + 1
		header := strings.TrimSpace(styles[start:open])
		close := strings.IndexByte(styles[open+1:], '}')
		if close < 0 {
			break
		}
		close += open + 1
		body := styles[open+1 : close]
		if !strings.Contains(body, "{") {
			for _, selector := range strings.Split(header, ",") {
				if strings.TrimSpace(selector) != target {
					continue
				}
				found = true
				for _, declaration := range strings.Split(body, ";") {
					name, value, ok := strings.Cut(declaration, ":")
					if ok {
						declarations[strings.TrimSpace(name)] = strings.TrimSpace(value)
					}
				}
			}
		}
		offset = open + 1
	}
	return declarations, found
}

func mustCSSDeclarations(t *testing.T, styles, selector string) map[string]string {
	t.Helper()
	declarations, ok := cssDeclarations(styles, selector)
	if !ok {
		t.Fatalf("app.css rule %q not found", selector)
	}
	return declarations
}

func assertCSSDeclaration(t *testing.T, selector string, declarations map[string]string, property, want string) {
	t.Helper()
	if got := declarations[property]; got != want {
		t.Errorf("app.css %s %s = %q, want %q", selector, property, got, want)
	}
}

func findHTMLElement(node *html.Node, element, class string) *html.Node {
	if node.Type == html.ElementNode && node.Data == element && (class == "" || hasHTMLClass(node, class)) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findHTMLElement(child, element, class); found != nil {
			return found
		}
	}
	return nil
}

func hasHTMLClass(node *html.Node, class string) bool {
	for _, candidate := range strings.Fields(htmlAttribute(node, "class")) {
		if candidate == class {
			return true
		}
	}
	return false
}

func htmlAttribute(node *html.Node, name string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val
		}
	}
	return ""
}

func hasHTMLAttribute(node *html.Node, name string) bool {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return true
		}
	}
	return false
}

func rtlTestNodeText(node *html.Node) string {
	var text strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			text.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.TrimSpace(text.String())
}
