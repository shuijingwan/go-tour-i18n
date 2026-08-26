// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"golang.org/x/net/html"
)

const runtimeHeadMarker = `<script id="tour-runtime-head"></script>`

type prerenderedPage struct {
	route string
	html  []byte
}

func prerenderPath(route string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(route, "/tour/"), "/")
	if !strings.HasPrefix(route, "/tour/") || len(parts) != 2 || parts[0] == "" || parts[1] == "" || path.Clean(route) != route {
		return "", fmt.Errorf("invalid course route %q", route)
	}
	return path.Join("tour", "prerender", parts[0], parts[1]+".html"), nil
}

func loadPrerenderedPages(content fs.FS, routes []CourseRoute) ([]prerenderedPage, error) {
	pages := make([]prerenderedPage, 0, len(routes))
	for _, route := range routes {
		file, err := prerenderPath(route.Path)
		if err != nil {
			return nil, err
		}
		data, err := fs.ReadFile(content, file)
		if err != nil {
			return nil, fmt.Errorf("read prerendered page %s: %w", route.Path, err)
		}
		if err := validatePrerenderedPage(data, route); err != nil {
			return nil, fmt.Errorf("validate prerendered page %s: %w", route.Path, err)
		}
		pages = append(pages, prerenderedPage{route: route.Path, html: data})
	}
	return pages, nil
}

func validatePrerenderedPage(data []byte, route CourseRoute) error {
	checks := []struct {
		value []byte
		name  string
	}{
		{[]byte(runtimeHeadMarker), "runtime head marker"},
		{[]byte(`data-tour-rendered-route="` + route.Path + `"`), "render completion marker"},
		{[]byte(`<link rel="canonical" href="` + route.Canonical + `"`), "canonical"},
		{[]byte("<title>"), "title"},
		{[]byte(route.PageTitle), "page title"},
		{[]byte(`id="editor-container"`), "course body"},
	}
	for _, check := range checks {
		if !bytes.Contains(data, check.value) {
			return fmt.Errorf("missing %s", check.name)
		}
	}
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("parse HTML: %w", err)
	}
	description := findElement(document, "meta", "name", "description")
	if description == nil || attrValue(description, "content") != route.Description {
		return fmt.Errorf("description=%q, want formal metadata %q", attrValue(description, "content"), route.Description)
	}
	for _, forbidden := range [][]byte{
		[]byte("<iframe"),
		[]byte("adsbygoogle"),
		[]byte("data-google-query-id"),
		[]byte(`class="CodeMirror`),
	} {
		if bytes.Contains(bytes.ToLower(data), bytes.ToLower(forbidden)) {
			return fmt.Errorf("contains forbidden runtime DOM %q", forbidden)
		}
	}
	textarea := findElement(document, "textarea", "ui-codemirror", "")
	if len(route.Files) == 0 {
		if textarea != nil {
			return fmt.Errorf("page without an example contains ui-codemirror textarea")
		}
		return nil
	}
	if textarea == nil {
		return fmt.Errorf("page example is missing ui-codemirror textarea")
	}
	if nodeText(textarea) != route.Files[0] {
		return fmt.Errorf("ui-codemirror textarea does not match default lesson file")
	}
	if bytes.Contains(data, []byte("data-tour-prerender-source=")) {
		return fmt.Errorf("page contains deprecated embedded source")
	}
	return nil
}

func findElement(node *html.Node, element, attr, value string) *html.Node {
	if node.Type == html.ElementNode && node.Data == element {
		if attr == "" || attrValue(node, attr) == value {
			return node
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findElement(child, element, attr, value); found != nil {
			return found
		}
	}
	return nil
}

func attrValue(node *html.Node, key string) string {
	if node == nil {
		return ""
	}
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func nodeText(node *html.Node) string {
	if node == nil {
		return ""
	}
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
	return text.String()
}

func registerPrerenderedPages(mux *http.ServeMux, pages []prerenderedPage) {
	runtimeHead := string(analyticsHTML) + string(adHTML)
	for _, page := range pages {
		page := page
		mux.HandleFunc(page.route, func(w http.ResponseWriter, _ *http.Request) {
			content := page.html
			if runtimeHead != "" {
				content = bytes.Replace(content, []byte(runtimeHeadMarker), []byte(runtimeHeadMarker+string(runtimeHead)), 1)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(content)
		})
	}
}
