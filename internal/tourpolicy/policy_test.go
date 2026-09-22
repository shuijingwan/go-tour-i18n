package tourpolicy

import (
	"strings"
	"testing"
)

func TestLocalePublicationPolicies(t *testing.T) {
	for _, locale := range []string{"zh-CN", "fr-FR", "de-DE", "ko-KR"} {
		if got := ForLocale(locale); got != GoLocal {
			t.Errorf("ForLocale(%q) = %q, want %q", locale, got, GoLocal)
		}
	}
	for _, locale := range []string{"en", "pt-BR", "nl-NL", "es-ES", "it-IT", "ja-JP", "sv-SE", "tr-TR", "zh-TW", "new-locale"} {
		if got := ForLocale(locale); got != Standard {
			t.Errorf("ForLocale(%q) = %q, want %q", locale, got, Standard)
		}
	}
}

func TestCorrectedPublicationURLs(t *testing.T) {
	localeTargets := map[string]string{
		"https://de-go-dev.shuijingwanwq.com/tour": "https://de-go-dev.shuijingwanwq.com/tour/",
		"https://fr-go-dev.shuijingwanwq.com/tour": "https://fr-go-dev.shuijingwanwq.com/tour/",
		"https://go-dev.shuijingwanwq.com/tour":    "https://go-dev.shuijingwanwq.com/tour/",
		"https://ko-go-dev.shuijingwanwq.com/tour": "https://ko-go-dev.shuijingwanwq.com/tour/",
	}
	if len(tourLocaleLinkCorrections) != len(localeTargets) {
		t.Fatalf("locale link correction count = %d, want %d", len(tourLocaleLinkCorrections), len(localeTargets))
	}
	for source, want := range localeTargets {
		if got, ok := CorrectedPublicationURL(source); !ok || got != want {
			t.Errorf("CorrectedPublicationURL(%q) = %q, %t; want %q, true", source, got, ok, want)
		}
		if got := Classify(source); got != UnknownOwnerTarget {
			t.Errorf("Classify(noncanonical %q) = %q, want %q", source, got, UnknownOwnerTarget)
		}
		if got := Classify(want); got != TourLocale {
			t.Errorf("Classify(canonical %q) = %q, want %q", want, got, TourLocale)
		}
	}
	for source, want := range officialTargets {
		if got, ok := CorrectedPublicationURL(source); !ok || got != want {
			t.Errorf("CorrectedPublicationURL(%q) = %q, %t; want official %q, true", source, got, ok, want)
		}
	}
	if got, ok := CorrectedPublicationURL("https://pt-go-dev.shuijingwanwq.com/tour"); ok {
		t.Errorf("unexpected Standard Tour correction %q", got)
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
		"https://de-go-dev.shuijingwanwq.com/tour/":                                          TourLocale,
		"https://fr-go-dev.shuijingwanwq.com/tour/":                                          TourLocale,
		"https://go-dev.shuijingwanwq.com/tour/":                                             TourLocale,
		"https://ko-go-dev.shuijingwanwq.com/tour/":                                          TourLocale,
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
		"https://marketing-go-dev.shuijingwanwq.com/tour/",
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

func TestReviewedLocaleHomeTargetsAreExact(t *testing.T) {
	targets := []string{
		"https://ar-go-dev.shuijingwanwq.com/",
		"https://bn-go-dev.shuijingwanwq.com/",
		"https://pt-go-dev.shuijingwanwq.com/",
		"https://cs-go-dev.shuijingwanwq.com/",
		"https://nl-go-dev.shuijingwanwq.com/",
		"https://de-go-dev.shuijingwanwq.com/",
		"https://fr-go-dev.shuijingwanwq.com/",
		"https://go-dev.shuijingwanwq.com/",
		"https://hi-go-dev.shuijingwanwq.com/",
		"https://id-go-dev.shuijingwanwq.com/",
		"https://it-go-dev.shuijingwanwq.com/",
		"https://ja-go-dev.shuijingwanwq.com/",
		"https://ko-go-dev.shuijingwanwq.com/",
		"https://pl-go-dev.shuijingwanwq.com/",
		"https://ro-go-dev.shuijingwanwq.com/",
		"https://es-go-dev.shuijingwanwq.com/",
		"https://sv-go-dev.shuijingwanwq.com/",
		"https://ta-go-dev.shuijingwanwq.com/",
		"https://te-go-dev.shuijingwanwq.com/",
		"https://th-go-dev.shuijingwanwq.com/",
		"https://tr-go-dev.shuijingwanwq.com/",
		"https://ur-go-dev.shuijingwanwq.com/",
		"https://uk-go-dev.shuijingwanwq.com/",
		"https://vi-go-dev.shuijingwanwq.com/",
		"https://zh-tw-go-dev.shuijingwanwq.com/",
	}
	if len(siteHomeTargets) != len(targets) {
		t.Fatalf("reviewed locale homepage count = %d, want %d", len(siteHomeTargets), len(targets))
	}
	for _, target := range targets {
		if got := Classify(target); got != SiteHome {
			t.Errorf("Classify(%q) = %q, want %q", target, got, SiteHome)
		}
		for _, unreviewed := range []string{
			strings.TrimSuffix(target, "/"),
			target + "project",
			target + "%2e/",
			target + "?from=header",
			target + "#languages",
			strings.Replace(target, "https://", "http://", 1),
			strings.Replace(target, "https://", "https://user@", 1),
			strings.Replace(target, ".com/", ".com:443/", 1),
		} {
			if got := Classify(unreviewed); got != UnknownOwnerTarget {
				t.Errorf("Classify(%q) = %q, want %q", unreviewed, got, UnknownOwnerTarget)
			}
		}
	}
	for _, target := range []string{
		"https://marketing-go-dev.shuijingwanwq.com/",
		"https://PT-go-dev.shuijingwanwq.com/",
	} {
		if got := Classify(target); got != UnknownOwnerTarget {
			t.Errorf("Classify(%q) = %q, want %q", target, got, UnknownOwnerTarget)
		}
	}
}

func TestReviewedLocaleTourTargetsAreExact(t *testing.T) {
	targets := []string{
		"https://de-go-dev.shuijingwanwq.com/tour/",
		"https://fr-go-dev.shuijingwanwq.com/tour/",
		"https://go-dev.shuijingwanwq.com/tour/",
		"https://ko-go-dev.shuijingwanwq.com/tour/",
	}
	if len(tourLocaleTargets) != len(targets) {
		t.Fatalf("reviewed locale Tour target count = %d, want %d", len(tourLocaleTargets), len(targets))
	}
	for _, target := range targets {
		if got := Classify(target); got != TourLocale {
			t.Errorf("Classify(%q) = %q, want %q", target, got, TourLocale)
		}
		for _, unreviewed := range []string{
			strings.TrimSuffix(target, "/"),
			target + "welcome/1",
			target + "?from=header",
			target + "#languages",
		} {
			if got := Classify(unreviewed); got != UnknownOwnerTarget {
				t.Errorf("Classify(%q) = %q, want %q", unreviewed, got, UnknownOwnerTarget)
			}
		}
	}
	for _, target := range []string{
		"https://pt-go-dev.shuijingwanwq.com/tour/",
		"https://nl-go-dev.shuijingwanwq.com/tour/",
		"https://it-go-dev.shuijingwanwq.com/tour/",
		"https://ja-go-dev.shuijingwanwq.com/tour/",
		"https://pl-go-dev.shuijingwanwq.com/tour/",
		"https://es-go-dev.shuijingwanwq.com/tour/",
		"https://sv-go-dev.shuijingwanwq.com/tour/",
		"https://tr-go-dev.shuijingwanwq.com/tour/",
		"https://zh-tw-go-dev.shuijingwanwq.com/tour/",
	} {
		if got := Classify(target); got != UnknownOwnerTarget {
			t.Errorf("Classify(%q) = %q, want %q", target, got, UnknownOwnerTarget)
		}
	}
}
