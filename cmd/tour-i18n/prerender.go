// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/tour"
	"golang.org/x/net/html"
)

const (
	prerenderRuntimeHeadMarker  = `<script id="tour-runtime-head"></script>`
	prerenderChromeWorkerLimit  = 2
	prerenderChromeRouteTimeout = 60 * time.Second
)

func prerenderProductionPagesChrome(contentDir, locale string, expectedPages int) error {
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		return fmt.Errorf("google-chrome is required: %w", err)
	}
	source, err := tour.NewPrerenderSource(os.DirFS(contentDir), locale)
	if err != nil {
		return fmt.Errorf("create prerender source: %w", err)
	}
	if len(source.Routes) != expectedPages {
		return fmt.Errorf("prerender routes=%d, projection pages=%d", len(source.Routes), expectedPages)
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen for prerender source: %w", err)
	}
	server := httptest.NewUnstartedServer(source.Handler)
	server.Listener = listener
	server.Start()
	defer server.Close()
	profileRoot, err := os.MkdirTemp("", "go-tour-prerender-chrome-")
	if err != nil {
		return fmt.Errorf("create Chrome profile root: %w", err)
	}
	defer os.RemoveAll(profileRoot)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobs := make(chan tour.CourseRoute)
	var workers sync.WaitGroup
	var firstErr error
	var errorMu sync.Mutex
	workerCount := min(prerenderChromeWorkerLimit, max(1, runtime.NumCPU()))
	for worker := 0; worker < workerCount; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			for route := range jobs {
				if ctx.Err() != nil {
					return
				}
				profile := filepath.Join(profileRoot, fmt.Sprintf("worker-%d", worker))
				if err := prerenderRouteWithChrome(ctx, chrome, server.URL, profile, contentDir, route); err != nil {
					errorMu.Lock()
					if firstErr == nil {
						firstErr = err
						cancel()
					}
					errorMu.Unlock()
					return
				}
			}
		}(worker)
	}
	for _, route := range source.Routes {
		select {
		case jobs <- route:
		case <-ctx.Done():
			break
		}
		if ctx.Err() != nil {
			break
		}
	}
	close(jobs)
	workers.Wait()
	if firstErr != nil {
		return firstErr
	}
	return prerenderListWithChrome(ctx, chrome, server.URL, filepath.Join(profileRoot, "list"), contentDir, source.List)
}

func prerenderListWithChrome(parent context.Context, chrome, serverURL, profileRoot, contentDir string, route tour.ListRoute) error {
	ctx, cancel := context.WithTimeout(parent, prerenderChromeRouteTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, chrome,
		"--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage",
		"--disable-breakpad", "--disable-crash-reporter", "--disable-background-networking",
		"--disable-default-apps", "--disable-extensions", "--no-first-run", "--noerrdialogs",
		"--user-data-dir="+profileRoot,
		"--host-resolver-rules=MAP assets-go-dev.shuijingwanwq.com ~NOTFOUND, MAP fonts.googleapis.com ~NOTFOUND, MAP pagead2.googlesyndication.com ~NOTFOUND",
		"--run-all-compositor-stages-before-draw", "--virtual-time-budget=5000", "--dump-dom", serverURL+route.Path,
	)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("Chrome %s: %w", route.Path, ctx.Err())
		}
		return fmt.Errorf("Chrome %s: %w: %s", route.Path, err, strings.TrimSpace(stderr.String()))
	}
	output, err = sanitizePrerenderedHTML(output)
	if err != nil {
		return fmt.Errorf("sanitize %s: %w", route.Path, err)
	}
	if err := validateRenderedListPage(output, route); err != nil {
		return fmt.Errorf("validate %s: %w", route.Path, err)
	}
	outputPath := filepath.Join(contentDir, "tour", "prerender", "list.html")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create prerender directory: %w", err)
	}
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("write prerendered list: %w", err)
	}
	return nil
}

func prerenderRouteWithChrome(parent context.Context, chrome, serverURL, profileRoot, contentDir string, route tour.CourseRoute) error {
	ctx, cancel := context.WithTimeout(parent, prerenderChromeRouteTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, chrome,
		"--headless=new",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-breakpad",
		"--disable-crash-reporter",
		"--disable-background-networking",
		"--disable-default-apps",
		"--disable-extensions",
		"--no-first-run",
		"--noerrdialogs",
		"--user-data-dir="+profileRoot,
		"--host-resolver-rules=MAP assets-go-dev.shuijingwanwq.com ~NOTFOUND, MAP fonts.googleapis.com ~NOTFOUND, MAP pagead2.googlesyndication.com ~NOTFOUND",
		"--run-all-compositor-stages-before-draw",
		"--virtual-time-budget=5000",
		"--dump-dom",
		serverURL+route.Path,
	)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("Chrome %s: %w", route.Path, ctx.Err())
		}
		return fmt.Errorf("Chrome %s: %w: %s", route.Path, err, strings.TrimSpace(stderr.String()))
	}
	output, err = sanitizePrerenderedHTML(output)
	if err != nil {
		return fmt.Errorf("sanitize %s: %w", route.Path, err)
	}
	output, err = setPrerenderedEditorSource(output, route)
	if err != nil {
		return fmt.Errorf("set editor source %s: %w", route.Path, err)
	}
	if err := validateRenderedCoursePage(output, route); err != nil {
		return fmt.Errorf("validate %s: %w", route.Path, err)
	}
	outputPath, err := prerenderOutputPath(contentDir, route.Path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create prerender directory: %w", err)
	}
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("write prerendered page: %w", err)
	}
	return nil
}

func prerenderOutputPath(contentDir, route string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(route, "/tour/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || "/tour/"+strings.Join(parts, "/") != route || path.Clean(route) != route {
		return "", fmt.Errorf("invalid course route %q", route)
	}
	return filepath.Join(contentDir, "tour", "prerender", parts[0], parts[1]+".html"), nil
}

func sanitizePrerenderedHTML(data []byte) ([]byte, error) {
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var clean func(*html.Node)
	clean = func(node *html.Node) {
		for child := node.FirstChild; child != nil; {
			next := child.NextSibling

			// CodeMirror replaces the source textarea with a browser-runtime
			// wrapper. Do not freeze that transient DOM into the release; the
			// surviving ui-codemirror textarea will recreate it at runtime.
			if thirdPartyRuntimeNode(child) ||
				(child.Type == html.ElementNode && hasClass(child, "CodeMirror")) {
				node.RemoveChild(child)
				child = next
				continue
			}

			// TOC collapse animations may be captured at different points by
			// --dump-dom. The inline style is runtime UI state, not content.
			if child.Type == html.ElementNode && hasClass(child, "toc-page") {
				removeAttr(child, "style")
			}

			// CodeMirror hides its source textarea after initialization. Once
			// the runtime wrapper is removed, restore the textarea's static
			// visibility so initial HTML still exposes the editor source.
			if child.Type == html.ElementNode &&
				child.Data == "textarea" &&
				hasAttr(child, "ui-codemirror", "") {
				removeAttr(child, "style")
			}

			if hasAttr(child, "data-go-dev-course-ad", "") {
				for child.FirstChild != nil {
					child.RemoveChild(child.FirstChild)
				}
				for _, attr := range []string{
					"role",
					"aria-label",
					"data-go-dev-course-ad-group",
				} {
					removeAttr(child, attr)
				}
				for _, class := range []string{
					"go-dev-course-ad--max-336",
					"go-dev-course-ad--max-468",
					"go-dev-course-ad--max-728",
				} {
					removeClass(child, class)
				}
			}

			clean(child)
			child = next
		}
	}
	clean(document)
	var output bytes.Buffer
	if err := html.Render(&output, document); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// setPrerenderedEditorSource makes the surviving source textarea the one
// canonical static representation of the default editor file. CodeMirror's
// wrapper is deliberately removed by sanitizePrerenderedHTML.
func setPrerenderedEditorSource(data []byte, route tour.CourseRoute) ([]byte, error) {
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	textarea := findElement(document, "textarea", "ui-codemirror", "")
	if len(route.Files) == 0 {
		if textarea == nil {
			return data, nil
		}
		textarea.Parent.RemoveChild(textarea)
		var output bytes.Buffer
		if err := html.Render(&output, document); err != nil {
			return nil, err
		}
		return output.Bytes(), nil
	}
	if textarea == nil {
		return nil, fmt.Errorf("example page is missing ui-codemirror textarea")
	}
	for textarea.FirstChild != nil {
		textarea.RemoveChild(textarea.FirstChild)
	}
	textarea.AppendChild(&html.Node{Type: html.TextNode, Data: route.Files[0]})
	var output bytes.Buffer
	if err := html.Render(&output, document); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func thirdPartyRuntimeNode(node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}
	if node.Data == "iframe" || hasClass(node, "adsbygoogle") {
		return true
	}
	for _, attr := range node.Attr {
		if attr.Key == "data-google-query-id" {
			return true
		}
	}
	return false
}

func hasAttr(node *html.Node, key, value string) bool {
	for _, attr := range node.Attr {
		if attr.Key == key && (value == "" || attr.Val == value) {
			return true
		}
	}
	return false
}

func removeAttr(node *html.Node, key string) {
	attrs := node.Attr[:0]
	for _, attr := range node.Attr {
		if attr.Key == key {
			continue
		}
		attrs = append(attrs, attr)
	}
	node.Attr = attrs
}

func hasClass(node *html.Node, class string) bool {
	for _, attr := range node.Attr {
		if attr.Key == "class" {
			for _, value := range strings.Fields(attr.Val) {
				if value == class {
					return true
				}
			}
		}
	}
	return false
}

func removeClass(node *html.Node, class string) {
	for i := range node.Attr {
		if node.Attr[i].Key != "class" {
			continue
		}
		classes := strings.Fields(node.Attr[i].Val)
		kept := classes[:0]
		for _, value := range classes {
			if value != class {
				kept = append(kept, value)
			}
		}
		node.Attr[i].Val = strings.Join(kept, " ")
		return
	}
}

func validateRenderedCoursePage(data []byte, route tour.CourseRoute) error {
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if !bytes.Contains(data, []byte(prerenderRuntimeHeadMarker)) {
		return fmt.Errorf("missing runtime head marker")
	}
	if findElement(document, "html", "data-tour-rendered-route", route.Path) == nil {
		return fmt.Errorf("page did not finish rendering")
	}
	title := findElement(document, "title", "", "")
	if title == nil || !strings.Contains(nodeText(title), route.PageTitle) {
		return fmt.Errorf("title does not contain page title %q", route.PageTitle)
	}
	canonical := findElement(document, "link", "rel", "canonical")
	if canonical == nil || attrValue(canonical, "href") != route.Canonical {
		return fmt.Errorf("canonical=%q, want %q", attrValue(canonical, "href"), route.Canonical)
	}
	description := findElement(document, "meta", "name", "description")
	if description == nil || attrValue(description, "content") != route.Description {
		return fmt.Errorf("description=%q, want formal metadata %q", attrValue(description, "content"), route.Description)
	}
	course := findElement(document, "div", "id", "editor-container")
	slide := findElementByClass(document, "slide-content")
	if course == nil || slide == nil || !strings.Contains(nodeText(slide), route.PageTitle) || len(strings.TrimSpace(nodeText(slide))) <= len(strings.TrimSpace(route.PageTitle)) {
		return fmt.Errorf("missing current course title and body")
	}
	if findElementByClass(document, "CodeMirror") != nil {
		return fmt.Errorf("page contains CodeMirror runtime DOM")
	}

	if len(route.Files) == 0 {
		if findElement(document, "textarea", "ui-codemirror", "") != nil {
			return fmt.Errorf("page without an example contains ui-codemirror textarea")
		}
		return nil
	}

	textarea := findElement(document, "textarea", "ui-codemirror", "")
	if textarea == nil {
		return fmt.Errorf("example page is missing ui-codemirror textarea")
	}
	if strings.TrimSpace(attrValue(textarea, "style")) != "" {
		return fmt.Errorf("ui-codemirror textarea contains runtime style")
	}

	if nodeText(textarea) != route.Files[0] {
		return fmt.Errorf("ui-codemirror textarea does not match default lesson file")
	}
	if findElementsWithAttr(document, "data-tour-prerender-source") != nil {
		return fmt.Errorf("page contains deprecated embedded source")
	}
	return nil
}

func validateRenderedListPage(data []byte, route tour.ListRoute) error {
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if !bytes.Contains(data, []byte(prerenderRuntimeHeadMarker)) || findElement(document, "html", "data-tour-rendered-route", route.Path) == nil {
		return fmt.Errorf("page did not finish rendering")
	}
	title := findElement(document, "title", "", "")
	if title == nil || nodeText(title) != route.PageTitle {
		return fmt.Errorf("title=%q, want %q", nodeText(title), route.PageTitle)
	}
	canonical := findElement(document, "link", "rel", "canonical")
	if canonical == nil || attrValue(canonical, "href") != route.Canonical {
		return fmt.Errorf("canonical=%q, want %q", attrValue(canonical, "href"), route.Canonical)
	}
	description := findElement(document, "meta", "name", "description")
	if description == nil || attrValue(description, "content") != route.Description {
		return fmt.Errorf("description=%q, want %q", attrValue(description, "content"), route.Description)
	}
	if heading := findElement(document, "h1", "", ""); heading == nil || nodeText(heading) != route.Heading {
		return fmt.Errorf("missing list heading %q", route.Heading)
	}
	for _, module := range route.Modules {
		if !strings.Contains(nodeText(document), module.Title) || !strings.Contains(nodeText(document), richText(module.Description)) {
			return fmt.Errorf("missing localized module %q", module.Title)
		}
	}
	for _, lesson := range route.Lessons {
		if findElement(document, "a", "href", lesson.Path) == nil || !strings.Contains(nodeText(document), lesson.LessonTitle) || !strings.Contains(nodeText(document), lesson.LessonDescription) {
			return fmt.Errorf("missing localized lesson %q", lesson.Path)
		}
	}
	return nil
}

func findElement(node *html.Node, element, attr, value string) *html.Node {
	if node.Type == html.ElementNode && node.Data == element && (attr == "" || attrValue(node, attr) == value) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findElement(child, element, attr, value); found != nil {
			return found
		}
	}
	return nil
}

func findElementByClass(node *html.Node, class string) *html.Node {
	if node.Type == html.ElementNode && hasClass(node, class) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findElementByClass(child, class); found != nil {
			return found
		}
	}
	return nil
}

func findElementsWithAttr(node *html.Node, attr string) []*html.Node {
	var found []*html.Node
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.ElementNode && hasAttr(current, attr, "") {
			found = append(found, current)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return found
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

func richText(value string) string {
	document, err := html.Parse(strings.NewReader(value))
	if err != nil {
		return value
	}
	return nodeText(document)
}

func nodeText(node *html.Node) string {
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
