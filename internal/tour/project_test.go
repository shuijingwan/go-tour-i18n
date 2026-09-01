// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadSiteMetadataDistinguishesDevelopmentAndProduction(t *testing.T) {
	const fields = `"locale":"zh-CN","upstream_commit":"` + FrozenUpstreamCommit + `","upstream_commit_time":"` + FrozenUpstreamCommitTime + `","pages":103,"articles":7`
	tests := []struct {
		name    string
		json    string
		wantDev bool
		wantErr string
	}{
		{"development", `{"development":true,` + fields + `}`, true, ""},
		{"production", `{"published_at":"2026-08-12T07:23:34Z",` + fields + `}`, false, ""},
		{"development with publication", `{"development":true,"published_at":"2026-08-12T07:23:34Z",` + fields + `}`, false, "must not contain published_at"},
		{"production without publication", `{` + fields + `}`, false, "invalid site metadata published_at"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := loadSiteMetadata(fstest.MapFS{"tour/site-metadata.json": {Data: []byte(test.json)}})
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("loadSiteMetadata error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if metadata.Development != test.wantDev {
				t.Fatalf("Development = %t, want %t", metadata.Development, test.wantDev)
			}
		})
	}
}

func TestWriteSiteMetadataRejectsDevelopmentMetadata(t *testing.T) {
	metadata := SiteMetadata{
		Development:        true,
		Locale:             "zh-CN",
		UpstreamCommit:     FrozenUpstreamCommit,
		UpstreamCommitTime: FrozenUpstreamCommitTime,
		Pages:              103,
		Articles:           7,
	}
	contentDir := t.TempDir()
	if err := WriteSiteMetadata(contentDir, metadata); err == nil {
		t.Fatal("WriteSiteMetadata accepted development metadata")
	}
	if _, err := os.Stat(filepath.Join(contentDir, "tour", "site-metadata.json")); !os.IsNotExist(err) {
		t.Fatalf("development metadata file exists or unexpected stat error: %v", err)
	}
}

func TestSiteMetadataTimesAreLocaleAware(t *testing.T) {
	metadata := SiteMetadata{UpstreamCommitTime: "2026-08-20T05:56:11Z"}
	for locale, want := range map[string]string{
		"zh-CN": "2026-08-20 13:56:11（北京时间）",
		"de-DE": "2026-08-20 07:56:11 (Ortszeit)",
		"fr-FR": "2026-08-20 07:56:11 (heure locale)",
		"ja-JP": "2026-08-20 14:56:11（日本時間）",
		"ko-KR": "2026-08-20 14:56:11 (한국 표준시)",
	} {
		got, err := metadata.UpstreamCommitTimeFor(localeProfiles[locale])
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("UpstreamCommitTimeFor(%s) = %q, want %q", locale, got, want)
		}
	}
}

func TestFrenchSiteTimeObservesDaylightSavingTime(t *testing.T) {
	for _, test := range []struct {
		name, source, want string
	}{
		{"winter", "2026-01-15T12:00:00Z", "2026-01-15 13:00:00 (heure locale)"},
		{"summer", "2026-07-15T12:00:00Z", "2026-07-15 14:00:00 (heure locale)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := SiteMetadata{UpstreamCommitTime: test.source}
			got, err := metadata.UpstreamCommitTimeFor(localeProfiles["fr-FR"])
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("UpstreamCommitTimeFor(fr-FR) = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGermanSiteTimeObservesDaylightSavingTime(t *testing.T) {
	for _, test := range []struct {
		name, source, want string
	}{
		{"winter", "2026-01-15T12:00:00Z", "2026-01-15 13:00:00 (Ortszeit)"},
		{"summer", "2026-07-15T12:00:00Z", "2026-07-15 14:00:00 (Ortszeit)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := SiteMetadata{UpstreamCommitTime: test.source}
			got, err := metadata.UpstreamCommitTimeFor(localeProfiles["de-DE"])
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("UpstreamCommitTimeFor(de-DE) = %q, want %q", got, test.want)
			}
		})
	}
}
