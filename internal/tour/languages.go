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
	Locale      string
	EnglishName string
	Autonym     string
	Label       string
	URL         string
	Official    bool
	Current     bool
}

var languageRegistry = []LanguageLink{
	// 展示顺序按英文语言名称字母顺序排列。
	{Locale: "ar", EnglishName: "Arabic", Autonym: "العربية", URL: "https://ar-go-dev.shuijingwanwq.com/"},
	{Locale: "bn-BD", EnglishName: "Bengali", Autonym: "বাংলা", URL: "https://bn-go-dev.shuijingwanwq.com/"},
	{Locale: "pt-BR", EnglishName: "Brazilian Portuguese", Autonym: "Português (Brasil)", URL: "https://pt-go-dev.shuijingwanwq.com/"},
	{Locale: "bg-BG", EnglishName: "Bulgarian", Autonym: "Български", URL: "https://bg-go-dev.shuijingwanwq.com/"},
	{Locale: "cs-CZ", EnglishName: "Czech", Autonym: "Čeština", URL: "https://cs-go-dev.shuijingwanwq.com/"},
	{Locale: "nl-NL", EnglishName: "Dutch", Autonym: "Nederlands", URL: "https://nl-go-dev.shuijingwanwq.com/"},
	{Locale: "en", EnglishName: "English", Autonym: "English", URL: "https://go.dev/tour/", Official: true},
	{Locale: "fil-PH", EnglishName: "Filipino", Autonym: "Filipino", URL: "https://fil-go-dev.shuijingwanwq.com/"},
	{Locale: "fr-FR", EnglishName: "French", Autonym: "Français", URL: "https://fr-go-dev.shuijingwanwq.com/"},
	{Locale: "de-DE", EnglishName: "German", Autonym: "Deutsch", URL: "https://de-go-dev.shuijingwanwq.com/"},
	{Locale: "el-GR", EnglishName: "Greek", Autonym: "Ελληνικά", URL: "https://el-go-dev.shuijingwanwq.com/"},
	{Locale: "hi-IN", EnglishName: "Hindi", Autonym: "हिन्दी", URL: "https://hi-go-dev.shuijingwanwq.com/"},
	{Locale: "hu-HU", EnglishName: "Hungarian", Autonym: "Magyar", URL: "https://hu-go-dev.shuijingwanwq.com/"},
	{Locale: "id-ID", EnglishName: "Indonesian", Autonym: "Bahasa Indonesia", URL: "https://id-go-dev.shuijingwanwq.com/"},
	{Locale: "it-IT", EnglishName: "Italian", Autonym: "Italiano", URL: "https://it-go-dev.shuijingwanwq.com/"},
	{Locale: "ja-JP", EnglishName: "Japanese", Autonym: "日本語", URL: "https://ja-go-dev.shuijingwanwq.com/"},
	{Locale: "kn-IN", EnglishName: "Kannada", Autonym: "ಕನ್ನಡ", URL: "https://kn-go-dev.shuijingwanwq.com/"},
	{Locale: "ko-KR", EnglishName: "Korean", Autonym: "한국어", URL: "https://ko-go-dev.shuijingwanwq.com/"},
	{Locale: "es-419", EnglishName: "Latin American Spanish", Autonym: "Español (Latinoamérica)", URL: "https://es-419-go-dev.shuijingwanwq.com/"},
	{Locale: "ms-MY", EnglishName: "Malay", Autonym: "Bahasa Melayu", URL: "https://ms-go-dev.shuijingwanwq.com/"},
	{Locale: "ml-IN", EnglishName: "Malayalam", Autonym: "മലയാളം", URL: "https://ml-go-dev.shuijingwanwq.com/"},
	{Locale: "mr-IN", EnglishName: "Marathi", Autonym: "मराठी", URL: "https://mr-go-dev.shuijingwanwq.com/"},
	{Locale: "pl-PL", EnglishName: "Polish", Autonym: "Polski", URL: "https://pl-go-dev.shuijingwanwq.com/"},
	{Locale: "ro-RO", EnglishName: "Romanian", Autonym: "Română", URL: "https://ro-go-dev.shuijingwanwq.com/"},
	{Locale: "zh-CN", EnglishName: "Simplified Chinese", Autonym: "简体中文", URL: "https://go-dev.shuijingwanwq.com/"},
	{Locale: "es-ES", EnglishName: "Spanish", Autonym: "Español", URL: "https://es-go-dev.shuijingwanwq.com/"},
	{Locale: "sv-SE", EnglishName: "Swedish", Autonym: "Svenska", URL: "https://sv-go-dev.shuijingwanwq.com/"},
	{Locale: "ta-IN", EnglishName: "Tamil", Autonym: "தமிழ்", URL: "https://ta-go-dev.shuijingwanwq.com/"},
	{Locale: "te-IN", EnglishName: "Telugu", Autonym: "తెలుగు", URL: "https://te-go-dev.shuijingwanwq.com/"},
	{Locale: "th-TH", EnglishName: "Thai", Autonym: "ไทย", URL: "https://th-go-dev.shuijingwanwq.com/"},
	{Locale: "zh-TW", EnglishName: "Traditional Chinese", Autonym: "繁體中文（台灣）", URL: "https://zh-tw-go-dev.shuijingwanwq.com/"},
	{Locale: "tr-TR", EnglishName: "Turkish", Autonym: "Türkçe", URL: "https://tr-go-dev.shuijingwanwq.com/"},
	{Locale: "uk-UA", EnglishName: "Ukrainian", Autonym: "Українська", URL: "https://uk-go-dev.shuijingwanwq.com/"},
	{Locale: "ur-PK", EnglishName: "Urdu", Autonym: "اردو", URL: "https://ur-go-dev.shuijingwanwq.com/"},
	{Locale: "vi-VN", EnglishName: "Vietnamese", Autonym: "Tiếng Việt", URL: "https://vi-go-dev.shuijingwanwq.com/"},
}

// LanguageRegistry returns the homepage language registry in presentation
// order. The returned copy cannot alter the build-time authority.
func LanguageRegistry() []LanguageLink {
	registry := make([]LanguageLink, len(languageRegistry))
	copy(registry, languageRegistry)
	return registry
}

type localeProfile struct {
	DevelopmentLogURL string
	TimeZone          *time.Location
	TimeLabel         string
	TimeLabelFormat   string
	Direction         string
}

var (
	athensTime      = mustLoadLocation("Europe/Athens")
	budapestTime    = mustLoadLocation("Europe/Budapest")
	amsterdamTime   = mustLoadLocation("Europe/Amsterdam")
	berlinTime      = mustLoadLocation("Europe/Berlin")
	madridTime      = mustLoadLocation("Europe/Madrid")
	parisTime       = mustLoadLocation("Europe/Paris")
	pragueTime      = mustLoadLocation("Europe/Prague")
	warsawTime      = mustLoadLocation("Europe/Warsaw")
	romeTime        = mustLoadLocation("Europe/Rome")
	saoPauloTime    = mustLoadLocation("America/Sao_Paulo")
	sofiaTime       = mustLoadLocation("Europe/Sofia")
	stockholmTime   = mustLoadLocation("Europe/Stockholm")
	taipeiTime      = mustLoadLocation("Asia/Taipei")
	istanbulTime    = mustLoadLocation("Europe/Istanbul")
	karachiTime     = mustLoadLocation("Asia/Karachi")
	kyivTime        = mustLoadLocation("Europe/Kyiv")
	jakartaTime     = mustLoadLocation("Asia/Jakarta")
	kualaLumpurTime = mustLoadLocation("Asia/Kuala_Lumpur")
	manilaTime      = mustLoadLocation("Asia/Manila")
	kolkataTime     = mustLoadLocation("Asia/Kolkata")
	hoChiMinhTime   = mustLoadLocation("Asia/Ho_Chi_Minh")
	bangkokTime     = mustLoadLocation("Asia/Bangkok")
	bucharestTime   = mustLoadLocation("Europe/Bucharest")
	dhakaTime       = mustLoadLocation("Asia/Dhaka")
)

var localeProfiles = map[string]localeProfile{
	"ar": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          time.UTC,
		TimeLabel:         "UTC",
		TimeLabelFormat:   " (%s)",
		Direction:         "rtl",
	},
	"bg-BG": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          sofiaTime,
		TimeLabel:         "местно време",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"bn-BD": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          dhakaTime,
		TimeLabel:         "স্থানীয় সময়",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"pt-BR": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          saoPauloTime,
		TimeLabel:         "horário local",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"cs-CZ": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          pragueTime,
		TimeLabel:         "místní čas",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"nl-NL": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          amsterdamTime,
		TimeLabel:         "lokale tijd",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"zh-CN": {
		DevelopmentLogURL: "https://www.shuijingwanwq.com/series/go-tour-chinese-edition-development-series/",
		TimeZone:          time.FixedZone("UTC+8", 8*60*60),
		TimeLabel:         "北京时间",
		TimeLabelFormat:   "（%s）",
		Direction:         "ltr",
	},
	"de-DE": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          berlinTime,
		TimeLabel:         "Ortszeit",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"el-GR": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          athensTime,
		TimeLabel:         "τοπική ώρα",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"fil-PH": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          manilaTime,
		TimeLabel:         "lokal na oras",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"fr-FR": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          parisTime,
		TimeLabel:         "heure locale",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"hi-IN": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          kolkataTime,
		TimeLabel:         "स्थानीय समय",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"hu-HU": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          budapestTime,
		TimeLabel:         "helyi idő",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"kn-IN": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          kolkataTime,
		TimeLabel:         "ಸ್ಥಳೀಯ ಸಮಯ",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"ml-IN": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          kolkataTime,
		TimeLabel:         "പ്രാദേശിക സമയം",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"mr-IN": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          kolkataTime,
		TimeLabel:         "स्थानिक वेळ",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"ms-MY": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          kualaLumpurTime,
		TimeLabel:         "waktu tempatan",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"id-ID": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          jakartaTime,
		TimeLabel:         "waktu setempat",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"vi-VN": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          hoChiMinhTime,
		TimeLabel:         "giờ địa phương",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"it-IT": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          romeTime,
		TimeLabel:         "ora locale",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"es-419": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          time.UTC,
		TimeLabel:         "UTC",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"es-ES": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          madridTime,
		TimeLabel:         "hora local",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"pl-PL": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          warsawTime,
		TimeLabel:         "czas lokalny",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"ro-RO": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          bucharestTime,
		TimeLabel:         "ora locală",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"sv-SE": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          stockholmTime,
		TimeLabel:         "lokal tid",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"ta-IN": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          kolkataTime,
		TimeLabel:         "உள்ளூர் நேரம்",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"te-IN": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          kolkataTime,
		TimeLabel:         "స్థానిక సమయం",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"th-TH": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          bangkokTime,
		TimeLabel:         "เวลาท้องถิ่น",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"zh-TW": {
		DevelopmentLogURL: "https://www.shuijingwanwq.com/series/go-tour-chinese-edition-development-series/",
		TimeZone:          taipeiTime,
		TimeLabel:         "台灣時間",
		TimeLabelFormat:   "（%s）",
		Direction:         "ltr",
	},
	"tr-TR": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          istanbulTime,
		TimeLabel:         "yerel saat",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"uk-UA": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          kyivTime,
		TimeLabel:         "місцевий час",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	"ur-PK": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          karachiTime,
		TimeLabel:         "مقامی وقت",
		TimeLabelFormat:   " (%s)",
		Direction:         "rtl",
	},
	"ja-JP": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          time.FixedZone("UTC+9", 9*60*60),
		TimeLabel:         "日本時間",
		TimeLabelFormat:   "（%s）",
		Direction:         "ltr",
	},
	"ko-KR": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          time.FixedZone("UTC+9", 9*60*60),
		TimeLabel:         "한국 표준시",
		TimeLabelFormat:   " (%s)",
		Direction:         "ltr",
	},
	// English is the catalog source and remains renderable for development,
	// although the English language entry points to the official Tour.
	"en": {
		DevelopmentLogURL: "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/",
		TimeZone:          time.UTC,
		TimeLabel:         "UTC",
		TimeLabelFormat:   "（%s）",
		Direction:         "ltr",
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
		languages[i].Label = languages[i].EnglishName
		if languages[i].EnglishName != languages[i].Autonym {
			languages[i].Label += " — " + languages[i].Autonym
		}
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
