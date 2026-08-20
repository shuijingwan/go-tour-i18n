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
	const fields = `"locale":"zh-CN","upstream_commit":"645042eb697eaf69e33a9af00c6b5b3fffdead5a","upstream_commit_time":"2026-08-20T05:56:11Z","pages":103,"articles":7`
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
