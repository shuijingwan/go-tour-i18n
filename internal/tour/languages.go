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
	// 展示顺序按英文语言名称字母顺序排列。
	{Locale: "zh-CN", Autonym: "简体中文", URL: "https://go-dev.shuijingwanwq.com/"},
	{Locale: "en", Autonym: "English", URL: "https://go.dev/tour/", Official: true},
	{Locale: "fr-FR", Autonym: "Français", URL: "https://fr-go-dev.shuijingwanwq.com/"},
	{Locale: "de-DE", Autonym: "Deutsch", URL: "https://de-go-dev.shuijingwanwq.com/"},
	{Locale: "ja-JP", Autonym: "日本語", URL: "https://ja-go-dev.shuijingwanwq.com/"},
}

type localeProfile struct {
	DevelopmentLogURL string
	TimeZone          *time.Location
	TimeLabel         string
	TimeLabelFormat   string
}

var (
	berlinTime = mustLoadLocation("Europe/Berlin")
	parisTime  = mustLoadLocation("Europe/Paris")
)

var localeProfiles = map[string]localeProfile{
	"zh-CN": {
		DevelopmentLogURL: "https://www.shuijingwanwq.com/series/go-tour-chinese-edition-development-series/",
		TimeZone:          time.FixedZone("UTC+8", 8*60*60),
		TimeLabel:         "北京时间",
		TimeLabelFormat:   "（%s）",
	},
	"de-DE": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          berlinTime,
		TimeLabel:         "Ortszeit",
		TimeLabelFormat:   " (%s)",
	},
	"fr-FR": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          parisTime,
		TimeLabel:         "heure locale",
		TimeLabelFormat:   " (%s)",
	},
	"ja-JP": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          time.FixedZone("UTC+9", 9*60*60),
		TimeLabel:         "日本時間",
		TimeLabelFormat:   "（%s）",
	},
	// English is the catalog source and remains renderable for development,
	// although the English language entry points to the official Tour.
	"en": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          time.UTC,
		TimeLabel:         "UTC",
		TimeLabelFormat:   "（%s）",
	},
}

func mustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		panic(fmt.Sprintf("load time zone %q: %v", name, err))
	}
	return location
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
