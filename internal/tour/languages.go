// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"fmt"
	"time"
)

// LanguageLink describes one build-time language site exposed on the
// homepage. Registry order is presentation order and is shared by every
// locale build.
type LanguageLink struct {
	Locale   string
	Autonym  string
	URL      string
	Official bool
	Current  bool
}

var languageRegistry = []LanguageLink{
	{Locale: "zh-CN", Autonym: "简体中文", URL: "https://go-dev.shuijingwanwq.com/tour/welcome/1"},
	{Locale: "en", Autonym: "English", URL: "https://go.dev/tour/", Official: true},
	{Locale: "ja-JP", Autonym: "日本語", URL: "https://ja-go-dev.shuijingwanwq.com/tour/welcome/1"},
}

type localeProfile struct {
	DevelopmentLogURL string
	TimeZone          *time.Location
	TimeLabel         string
}

var localeProfiles = map[string]localeProfile{
	"zh-CN": {
		DevelopmentLogURL: "https://www.shuijingwanwq.com/series/go-tour-chinese-edition-development-series/",
		TimeZone:          time.FixedZone("UTC+8", 8*60*60),
		TimeLabel:         "北京时间",
	},
	"ja-JP": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          time.FixedZone("UTC+9", 9*60*60),
		TimeLabel:         "日本時間",
	},
	// English is the catalog source and remains renderable for development,
	// although the English language entry points to the official Tour.
	"en": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          time.UTC,
		TimeLabel:         "UTC",
	},
}

func languagesFor(locale string) ([]LanguageLink, error) {
	if _, ok := localeProfiles[locale]; !ok {
		return nil, fmt.Errorf("unsupported site locale %q", locale)
	}
	languages := make([]LanguageLink, len(languageRegistry))
	copy(languages, languageRegistry)
	for i := range languages {
		languages[i].Current = languages[i].Locale == locale
	}
	return languages, nil
}

func currentLanguage(languages []LanguageLink) (LanguageLink, error) {
	for _, language := range languages {
		if language.Current {
			return language, nil
		}
	}
	return LanguageLink{}, fmt.Errorf("language registry has no current locale")
}
