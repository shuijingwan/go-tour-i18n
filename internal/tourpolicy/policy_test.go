package tourpolicy

import "testing"

func TestLocalePublicationPolicies(t *testing.T) {
	for _, locale := range []string{"zh-CN", "fr-FR", "de-DE", "ko-KR"} {
		if got := ForLocale(locale); got != GoLocal {
			t.Errorf("ForLocale(%q) = %q, want %q", locale, got, GoLocal)
		}
	}
	for _, locale := range []string{"en", "pt-BR", "nl-NL", "es-ES", "it-IT", "ja-JP", "tr-TR", "new-locale"} {
		if got := ForLocale(locale); got != Standard {
			t.Errorf("ForLocale(%q) = %q, want %q", locale, got, Standard)
		}
	}
}

func TestReviewedOfficialTargetsAndFailClosedSiteContent(t *testing.T) {
	for source, want := range officialTargets {
		if got, ok := GoOfficialURL(source); !ok || got != want {
			t.Errorf("GoOfficialURL(%q) = %q, %t; want %q, true", source, got, ok, want)
		}
	}
	for _, target := range []string{"/future-commercial-content", "/other/content"} {
		if got := Classify(target); got != SiteContent {
			t.Errorf("Classify(%q) = %q, want %q", target, got, SiteContent)
		}
	}
	for target, want := range map[string]Class{
		"/tour/welcome/1":            TourLocal,
		"/":                          SiteHome,
		"https://go.dev/doc/install": GoOfficial,
		"https://de-go-dev.shuijingwanwq.com/tour":                                           TourLocale,
		"https://fr-go-dev.shuijingwanwq.com/tour":                                           TourLocale,
		"https://go-dev.shuijingwanwq.com/tour":                                              TourLocale,
		"https://ko-go-dev.shuijingwanwq.com/tour":                                           TourLocale,
		"https://www.shuijingwanwq.com/series/go-tour-chinese-edition-development-series/":   OwnerContent,
		"https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/": OwnerContent,
		"https://en.wikipedia.org/wiki/Go":                                                   External,
		"javascript:click('.next-page')":                                                     Action,
	} {
		if got := Classify(target); got != want {
			t.Errorf("Classify(%q) = %q, want %q", target, got, want)
		}
	}
}

func TestUnknownOwnerTargetsRequireClassification(t *testing.T) {
	for _, target := range []string{
		"https://marketing.shuijingwanwq.com/offer",
		"https://fr-go-dev.shuijingwanwq.com/tour/welcome/1",
		"https://go-dev.shuijingwanwq.com/tour/welcome/1",
	} {
		if got := Classify(target); got != UnknownOwnerTarget {
			t.Errorf("Classify(%q) = %q, want UnknownOwnerTarget", target, got)
		}
	}
	if GoLocal.OwnerContentLinksEnabled() {
		t.Error("go-local publication permits owner-controlled content links")
	}
	if !Standard.OwnerContentLinksEnabled() {
		t.Error("standard publication unexpectedly blocks owner-controlled content links")
	}
}
