package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

const liveLocalesStart = "<!-- live-locales:start -->"
const liveLocalesEnd = "<!-- live-locales:end -->"

type readmeIdentity struct {
	Locales []readmeProfile `json:"locales"`
}
type readmeProfile struct {
	Locale string `json:"locale"`
	State  string `json:"production_state"`
	URL    string `json:"production_public_url"`
}

// projectLiveLocales derives README's community list from the candidate
// identity and the homepage registry; neither authority is duplicated here.
func projectLiveLocales(readme, identity []byte, registry []tour.LanguageLink) ([]byte, error) {
	if bytes.Count(readme, []byte(liveLocalesStart)) != 1 || bytes.Count(readme, []byte(liveLocalesEnd)) != 1 {
		return nil, fmt.Errorf("README must contain exactly one live-locales start and end marker")
	}
	start, end := bytes.Index(readme, []byte(liveLocalesStart)), bytes.Index(readme, []byte(liveLocalesEnd))
	if start > end {
		return nil, fmt.Errorf("README live-locales markers are out of order")
	}
	var parsed readmeIdentity
	if err := json.Unmarshal(identity, &parsed); err != nil || parsed.Locales == nil {
		return nil, fmt.Errorf("malformed production identity for README projection")
	}
	profiles := map[string]readmeProfile{}
	for _, profile := range parsed.Locales {
		if profile.Locale == "" || profile.State == "" || profile.URL == "" {
			return nil, fmt.Errorf("production identity has incomplete locale for README projection")
		}
		if _, ok := profiles[profile.Locale]; ok {
			return nil, fmt.Errorf("production identity has duplicate locale %s for README projection", profile.Locale)
		}
		profiles[profile.Locale] = profile
	}
	registered := map[string]tour.LanguageLink{}
	for _, language := range registry {
		if language.Locale == "" || language.EnglishName == "" || language.Autonym == "" || language.URL == "" {
			return nil, fmt.Errorf("language registry has incomplete entry for README projection")
		}
		if _, ok := registered[language.Locale]; ok {
			return nil, fmt.Errorf("language registry has duplicate locale %s for README projection", language.Locale)
		}
		registered[language.Locale] = language
	}
	var lines []string
	for _, language := range registry {
		profile, ok := profiles[language.Locale]
		if !ok || profile.State != "live" {
			continue
		}
		if language.Official {
			return nil, fmt.Errorf("live production locale %s is incorrectly marked Official in language registry", language.Locale)
		}
		if language.URL != profile.URL {
			return nil, fmt.Errorf("language registry URL differs from production identity for %s", language.Locale)
		}
		label := language.EnglishName
		if language.EnglishName != language.Autonym {
			label += " — " + language.Autonym
		}
		lines = append(lines, "- ["+label+"]("+profile.URL+")")
	}
	for locale, profile := range profiles {
		if profile.State != "live" {
			continue
		}
		language, ok := registered[locale]
		if !ok {
			return nil, fmt.Errorf("live production locale %s is missing from language registry", locale)
		}
		if language.Official {
			return nil, fmt.Errorf("live production locale %s is incorrectly marked Official in language registry", locale)
		}
	}
	block := liveLocalesStart + "\n" + strings.Join(lines, "\n") + "\n" + liveLocalesEnd
	result := append([]byte{}, readme[:start]...)
	result = append(result, block...)
	return append(result, readme[end+len(liveLocalesEnd):]...), nil
}

func projectRootREADME(root string, identity []byte) ([]byte, error) {
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		return nil, fmt.Errorf("read README: %w", err)
	}
	return projectLiveLocales(readme, identity, tour.LanguageRegistry())
}
