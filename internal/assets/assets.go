// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package assets defines the shared project static asset set.
package assets

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const BaseURL = "https://assets-go-dev.shuijingwanwq.com"

// sharedPaths is the compatibility baseline. Paths are
// relative to the repository's _content directory and retain their public URL
// layout. Paths(root) adds the current frozen Site static namespaces.
var sharedPaths = []string{
	"images/go-logo-white.svg",
	"images/icons/brightness_2_gm_grey_24dp.svg",
	"images/icons/brightness_6_gm_grey_24dp.svg",
	"images/icons/light_mode_gm_grey_24dp.svg",
	"images/site-logo-32.png",
	"images/site-logo.png",
	"images/support/alipay.png",
	"images/support/wechat.png",
	"tour/static/css/app.css",
	"tour/static/go-dev/course-ad.css",
	"tour/static/go-dev/course-ad.js",
	"tour/static/img/gopher.png",
	"tour/static/js/support.js",
	"tour/static/lib/codemirror/lib/codemirror.css",
}

var sharedPathSet = func() map[string]bool {
	result := make(map[string]bool, len(sharedPaths))
	for _, logicalPath := range sharedPaths {
		result[logicalPath] = true
	}
	return result
}()

// SharedPaths returns a copy of the compatibility baseline.
func SharedPaths() []string {
	return append([]string(nil), sharedPaths...)
}

// StaticPath is the shared static namespace contract. It never admits HTML,
// translated sources, Tour partials/script.js, or executable Go examples.
var staticPathPattern = regexp.MustCompile(`^[A-Za-z0-9._/&-]+$`)

func StaticPath(p string) bool {
	if !staticPathPattern.MatchString(p) {
		return false
	}
	if path.Clean(p) != p || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\\\x00?#") || strings.HasPrefix(p, "../") {
		return false
	}
	if sharedPathSet[p] {
		return true
	}
	for _, prefix := range []string{"css/", "js/", "fonts/", "images/", "doc/", "learn/"} {
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		switch strings.ToLower(path.Ext(p)) {
		case ".css", ".js", ".woff", ".woff2", ".ttf", ".eot", ".otf", ".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico":
			return true
		}
	}
	return false
}

// Paths derives the exact export set from frozen repository static namespaces.
// This is one global authority for single and batch deployment, all locales.
func Paths(root string) ([]string, error) {
	set := map[string]bool{}
	for _, p := range SharedPaths() {
		set[p] = true
	}
	for _, prefix := range []string{"css", "js", "fonts", "images", "doc", "learn"} {
		base := filepath.Join(root, "_content", prefix)
		if _, e := os.Stat(base); os.IsNotExist(e) {
			continue
		}
		e := filepath.WalkDir(base, func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			if !d.Type().IsRegular() {
				return fmt.Errorf("static source contains unsupported entry: %s", p)
			}
			rel, e := filepath.Rel(filepath.Join(root, "_content"), p)
			if e != nil {
				return e
			}
			rel = filepath.ToSlash(rel)
			if StaticPath(rel) {
				set[rel] = true
			}
			return nil
		})
		if e != nil {
			return nil, e
		}
	}
	result := make([]string, 0, len(set))
	for p := range set {
		result = append(result, p)
	}
	sort.Strings(result)
	return result, nil
}

// URL resolves a logical asset path for one build-selected site. Development
// remains same-origin; all production locales use the shared assets origin.
func URL(locale string, development bool, logicalPath string) (string, error) {
	clean := path.Clean(logicalPath)
	if clean == "." || strings.HasPrefix(logicalPath, "/") || clean != logicalPath || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid asset path %q", logicalPath)
	}
	if !StaticPath(clean) {
		return "", fmt.Errorf("asset path %q is not in the shared allowlist", logicalPath)
	}
	local := "/" + clean
	if development {
		return local, nil
	}
	return BaseURL + local, nil
}
