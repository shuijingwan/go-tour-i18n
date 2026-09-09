// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tourpolicy holds the publication policy shared by locale projection
// and the Tour runtime. It intentionally classifies audited link targets by
// their meaning instead of treating a URL prefix as proof of its destination.
package tourpolicy

import (
	"net/url"
	"strings"
)

// Publication selects the public relationship between a locale and go.dev.
type Publication string

const (
	Standard Publication = "standard"
	GoLocal  Publication = "go-local"
)

// ForLocale returns the explicit publication policy for a locale. Locales not
// named here deliberately default to Standard; registry tests ensure every
// supported locale has that default verified.
func ForLocale(locale string) Publication {
	switch locale {
	case "zh-CN", "fr-FR", "de-DE", "ko-KR":
		return GoLocal
	default:
		return Standard
	}
}

func (p Publication) TourAdsEnabled() bool { return p == Standard }

// OwnerContentLinksEnabled reports whether a Tour publication may link to
// owner-controlled content sites. This is independent of whether a target has
// the current locale hostname: go-local Tours must not expose a direct path to
// either kind of owner-controlled commercial content.
func (p Publication) OwnerContentLinksEnabled() bool { return p == Standard }

// Class is the semantic class of a clickable Tour target.
type Class string

const (
	TourLocal          Class = "tour-local"
	SiteHome           Class = "site-home"
	GoOfficial         Class = "go-official"
	External           Class = "external"
	SiteContent        Class = "site-content"
	OwnerContent       Class = "owner-content"
	UnknownOwnerTarget Class = "unknown-owner-target"
	Action             Class = "action"
)

const ownerControlledDomain = "shuijingwanwq.com"

// ownerContentTargets contains only the owner-controlled commercial/content
// destinations that have been audited. Do not add a host here merely because
// it shares the project domain; unknown project-owned hosts must instead be
// reviewed through UnknownOwnerTarget.
var ownerContentTargets = map[string]bool{
	"https://www.shuijingwanwq.com/series/go-tour-chinese-edition-development-series/":   true,
	"https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/": true,
}

// officialTargets is the reviewed inventory of legacy official links inherited
// from the upstream Tour. These are exact targets, not prefix rules: adding a
// new same-site target requires an explicit semantic decision and the go-local
// gate will reject it until then.
var officialTargets = map[string]string{
	"/":                                   "https://go.dev/",
	"/blog/":                              "https://go.dev/blog/",
	"/blog/defer-panic-and-recover":       "https://go.dev/blog/defer-panic-and-recover",
	"/blog/go-slices-usage-and-internals": "https://go.dev/blog/go-slices-usage-and-internals",
	"/blog/gos-declaration-syntax":        "https://go.dev/blog/gos-declaration-syntax",
	"/blog/playground":                    "https://go.dev/blog/playground",
	"/cmd/go/#hdr-GOPATH_and_Modules":     "https://go.dev/cmd/go/#hdr-GOPATH_and_Modules",
	"/cmd/gofmt/":                         "https://go.dev/cmd/gofmt/",
	"/doc/":                               "https://go.dev/doc/",
	"/doc/articles/wiki/":                 "https://go.dev/doc/articles/wiki/",
	"/doc/code":                           "https://go.dev/doc/code",
	"/doc/codewalk/functions/":            "https://go.dev/doc/codewalk/functions/",
	"/doc/codewalk/sharemem/":             "https://go.dev/doc/codewalk/sharemem/",
	"/doc/contribute#check_tracker":       "https://go.dev/doc/contribute#check_tracker",
	"/doc/install":                        "https://go.dev/doc/install",
	"/doc/install/":                       "https://go.dev/doc/install/",
	"/pkg/":                               "https://go.dev/pkg/",
	"/pkg/builtin/#append":                "https://go.dev/pkg/builtin/#append",
	"/pkg/compress/gzip/#NewReader":       "https://go.dev/pkg/compress/gzip/#NewReader",
	"/pkg/fmt/":                           "https://go.dev/pkg/fmt/",
	"/pkg/fmt/#Stringer":                  "https://go.dev/pkg/fmt/#Stringer",
	"/pkg/image/#Image":                   "https://go.dev/pkg/image/#Image",
	"/pkg/image/#Rectangle":               "https://go.dev/pkg/image/#Rectangle",
	"/pkg/image/color/":                   "https://go.dev/pkg/image/color/",
	"/pkg/io/#Reader":                     "https://go.dev/pkg/io/#Reader",
	"/pkg/math/#Sqrt":                     "https://go.dev/pkg/math/#Sqrt",
	"/pkg/strings/#Fields":                "https://go.dev/pkg/strings/#Fields",
	"/pkg/strings/#Reader":                "https://go.dev/pkg/strings/#Reader",
	"/pkg/sync/":                          "https://go.dev/pkg/sync/",
	"/pkg/sync/#Mutex":                    "https://go.dev/pkg/sync/#Mutex",
	"/ref/spec":                           "https://go.dev/ref/spec",
	"/talks/2012/concurrency.slide":       "https://go.dev/talks/2012/concurrency.slide",
	"/talks/2012/simple.slide":            "https://go.dev/talks/2012/simple.slide",
	"/talks/2013/advconc.slide":           "https://go.dev/talks/2013/advconc.slide",
	"https://play.golang.org/":            "https://go.dev/play/",
}

// GoOfficialURL returns the official destination for one reviewed target.
func GoOfficialURL(target string) (string, bool) {
	url, ok := officialTargets[target]
	return url, ok
}

// Classify classifies the current target. Root-relative targets deliberately
// remain SiteContent unless their reviewed meaning is known.
func Classify(target string) Class {
	if strings.HasPrefix(target, "javascript:") || target == "#" {
		return Action
	}
	if target == "https://go.dev" || strings.HasPrefix(target, "https://go.dev/") {
		return GoOfficial
	}
	if target == "/" {
		return SiteHome
	}
	if strings.HasPrefix(target, "/tour/") || target == "/tour" {
		return TourLocal
	}
	if _, ok := GoOfficialURL(target); ok {
		return GoOfficial
	}
	if strings.HasPrefix(target, "/") {
		return SiteContent
	}
	if ownerContentTargets[target] {
		return OwnerContent
	}
	if parsed, err := url.Parse(target); err == nil && isOwnerControlledHost(parsed.Hostname()) {
		return UnknownOwnerTarget
	}
	return External
}

// isOwnerControlledHost identifies targets requiring an explicit ownership
// review. It does not assign their content semantics: that is reserved for
// ownerContentTargets above.
func isOwnerControlledHost(host string) bool {
	host = strings.ToLower(host)
	return host == ownerControlledDomain || strings.HasSuffix(host, "."+ownerControlledDomain)
}
