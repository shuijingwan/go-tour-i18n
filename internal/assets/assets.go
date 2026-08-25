// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package assets defines the first shared, locale-neutral asset set.
package assets

import (
	"fmt"
	"path"
	"strings"
)

const BaseURL = "https://assets-go-dev.shuijingwanwq.com"

// SharedPaths is the complete assets origin allowlist. Paths are
// relative to the repository's _content directory and retain their public URL
// layout. Keep this list small: adding a file changes the public export.
var sharedPaths = []string{
	"images/go-logo-white.svg",
	"images/icons/brightness_2_gm_grey_24dp.svg",
	"images/icons/brightness_6_gm_grey_24dp.svg",
	"images/icons/light_mode_gm_grey_24dp.svg",
	"images/site-logo-32.png",
	"images/site-logo.png",
	"tour/static/css/app.css",
	"tour/static/go-dev/course-ad.css",
	"tour/static/go-dev/course-ad.js",
	"tour/static/img/gopher.png",
	"tour/static/lib/codemirror/lib/codemirror.css",
}

var sharedPathSet = func() map[string]bool {
	result := make(map[string]bool, len(sharedPaths))
	for _, logicalPath := range sharedPaths {
		result[logicalPath] = true
	}
	return result
}()

// SharedPaths returns a copy of the complete allowlist.
func SharedPaths() []string {
	return append([]string(nil), sharedPaths...)
}

// URL resolves a logical asset path for one build-selected site. Development
// and zh-CN production remain same-origin; other production locales use the
// shared assets origin.
func URL(locale string, development bool, logicalPath string) (string, error) {
	clean := path.Clean(logicalPath)
	if clean == "." || strings.HasPrefix(logicalPath, "/") || clean != logicalPath || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid asset path %q", logicalPath)
	}
	if !sharedPathSet[clean] {
		return "", fmt.Errorf("asset path %q is not in the shared allowlist", logicalPath)
	}
	local := "/" + clean
	if development || locale == "zh-CN" {
		return local, nil
	}
	return BaseURL + local, nil
}
