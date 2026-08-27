// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Project holds stable public project configuration. URLs and ownership data
// live here so templates and release generation do not duplicate them.
var Project = struct {
	GitHubURL, GitHubIssuesURL                      string
	UpstreamURL, ICPURL, ICPNumber, CopyrightHolder string
}{
	GitHubURL:       "https://github.com/shuijingwan/go-tour-i18n",
	GitHubIssuesURL: "https://github.com/shuijingwan/go-tour-i18n/issues",
	UpstreamURL:     "https://github.com/golang/website",
	ICPURL:          "https://beian.miit.gov.cn/",
	ICPNumber:       "蜀ICP备13001590号-1",
	CopyrightHolder: "永夜",
}

const (
	FrozenUpstreamCommit     = "b3fc6537086f09e88cb3c1ecd09bd47c31c54241"
	FrozenUpstreamCommitTime = "2026-08-26T21:55:26Z"
)

// SiteMetadata is read from the selected content tree at startup. Source-tree
// metadata is explicitly development-only; publish writes production metadata
// with a required RFC 3339 UTC publication time.
type SiteMetadata struct {
	Development        bool   `json:"development,omitempty"`
	Locale             string `json:"locale"`
	PublishedAt        string `json:"published_at"`
	UpstreamCommit     string `json:"upstream_commit"`
	UpstreamCommitTime string `json:"upstream_commit_time"`
	Pages              int    `json:"pages"`
	Articles           int    `json:"articles"`
}

func (m SiteMetadata) PublishedAtFor(profile localeProfile) (string, error) {
	if m.Development {
		return "", fmt.Errorf("development metadata has no published_at")
	}
	t, err := time.Parse(time.RFC3339, m.PublishedAt)
	if err != nil {
		return "", fmt.Errorf("parse published_at: %w", err)
	}
	return formatSiteTime(t, profile), nil
}

func (m SiteMetadata) UpstreamCommitTimeFor(profile localeProfile) (string, error) {
	t, err := time.Parse(time.RFC3339, m.UpstreamCommitTime)
	if err != nil {
		return "", fmt.Errorf("parse upstream_commit_time: %w", err)
	}
	return formatSiteTime(t, profile), nil
}

func formatSiteTime(t time.Time, profile localeProfile) string {
	return t.In(profile.TimeZone).Format("2006-01-02 15:04:05") + "（" + profile.TimeLabel + "）"
}

func loadSiteMetadata(content fs.FS) (SiteMetadata, error) {
	data, err := fs.ReadFile(content, "tour/site-metadata.json")
	if err != nil {
		return SiteMetadata{}, fmt.Errorf("read site metadata: %w", err)
	}
	var metadata SiteMetadata
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&metadata); err != nil {
		return SiteMetadata{}, fmt.Errorf("parse site metadata: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return SiteMetadata{}, fmt.Errorf("parse site metadata: multiple JSON values")
	}
	if metadata.Locale == "" || metadata.Pages < 1 || metadata.Articles < 1 || metadata.UpstreamCommit != FrozenUpstreamCommit || metadata.UpstreamCommitTime != FrozenUpstreamCommitTime {
		return SiteMetadata{}, fmt.Errorf("invalid site metadata")
	}
	if metadata.Development {
		if metadata.PublishedAt != "" {
			return SiteMetadata{}, fmt.Errorf("development site metadata must not contain published_at")
		}
	} else if _, err := time.Parse(time.RFC3339, metadata.PublishedAt); err != nil {
		return SiteMetadata{}, fmt.Errorf("invalid site metadata published_at: %w", err)
	}
	return metadata, nil
}

// WriteSiteMetadata writes the bundle-local metadata consumed by the public
// homepage. The caller supplies values calculated by the publish projection.
func WriteSiteMetadata(contentDir string, metadata SiteMetadata) error {
	if metadata.Development {
		return fmt.Errorf("production site metadata cannot be development metadata")
	}
	if _, err := time.Parse(time.RFC3339, metadata.PublishedAt); err != nil {
		return fmt.Errorf("parse published_at: %w", err)
	}
	if metadata.Locale == "" || metadata.Pages < 1 || metadata.Articles < 1 || metadata.UpstreamCommit != FrozenUpstreamCommit || metadata.UpstreamCommitTime != FrozenUpstreamCommitTime {
		return fmt.Errorf("invalid site metadata")
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encode site metadata: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(contentDir, "tour", "site-metadata.json"), data, 0644); err != nil {
		return fmt.Errorf("write site metadata: %w", err)
	}
	return nil
}
