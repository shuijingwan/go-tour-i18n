// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package website exports the static content needed by the standalone Tour.
package website

import (
	"embed"
	"io/fs"
)

// TourOnly returns the content needed by the standalone Tour.
func TourOnly() fs.FS {
	return subdir(tourOnly, "_content")
}

// This list is adapted from golang.org/x/website.TourOnly. It intentionally
// embeds only Tour resources and includes every favicon used by index.tmpl.
//
//go:embed _content/favicon.ico
//go:embed _content/images/go-logo-white.svg
//go:embed _content/images/site-logo.png
//go:embed _content/images/site-logo-32.png
//go:embed _content/images/icons
//go:embed _content/images/favicon-gopher.png
//go:embed _content/images/favicon-gopher-plain.png
//go:embed _content/images/favicon-gopher.svg
//go:embed _content/js/playground.js
//go:embed _content/tour
var tourOnly embed.FS

func subdir(fsys fs.FS, path string) fs.FS {
	s, err := fs.Sub(fsys, path)
	if err != nil {
		panic(err)
	}
	return s
}
