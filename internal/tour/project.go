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
	"path"
	"path/filepath"
	"strings"
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

type supportAudience string

const (
	supportAudienceMainland      supportAudience = "mainland"
	supportAudienceInternational supportAudience = "international"
)

type supportPaymentMethod struct {
	Identity          string
	Name              string
	Kind              string
	Enabled           bool
	Audience          supportAudience
	Asset             string
	Network           string
	Address           string
	MinimumDeposit    string
	QRAsset           string
	NetworkWarningKey string
}

// ProjectSupport is the single project-level maintenance source for public
// payment details. Locale catalogs contain presentation text only.
var ProjectSupport = struct {
	LocaleReferencePrefix string
	Methods               []supportPaymentMethod
}{
	LocaleReferencePrefix: "go-dev-",
	Methods: []supportPaymentMethod{
		{Identity: "wechat-pay", Name: "微信支付", Kind: "qr", Enabled: true, Audience: supportAudienceMainland, QRAsset: "images/support/wechat.png"},
		{Identity: "alipay", Name: "支付宝", Kind: "qr", Enabled: true, Audience: supportAudienceMainland, QRAsset: "images/support/alipay.png"},
		{Identity: "unionpay", Name: "云闪付", Kind: "qr", Enabled: false, Audience: supportAudienceMainland},
		{Identity: "usdc-base", Name: "USDC", Kind: "crypto", Enabled: true, Audience: supportAudienceInternational, Asset: "USDC", Network: "Base", Address: "0x225f14d54683b1f5bc153bc8a678cad0277096d3", MinimumDeposit: "0.01 USDC", NetworkWarningKey: "support.usdc_network_only"},
		{Identity: "usdt-trc20", Name: "USDT", Kind: "crypto", Enabled: true, Audience: supportAudienceInternational, Asset: "USDT", Network: "Tron (TRC20)", Address: "TF2bM817pLQeN1Ykt3GEecRbTjuSsWtGdK", MinimumDeposit: "0.1 USDT", NetworkWarningKey: "support.usdt_network_only"},
		{Identity: "wise", Name: "Wise", Kind: "link", Enabled: false, Audience: supportAudienceInternational},
		{Identity: "patreon", Name: "Patreon", Kind: "link", Enabled: false, Audience: supportAudienceInternational},
	},
}

type supportView struct {
	Audience  supportAudience
	Reference string
	Methods   []supportPaymentMethod
}

func supportAudienceForLocale(locale string) supportAudience {
	if locale == "zh-CN" {
		return supportAudienceMainland
	}
	return supportAudienceInternational
}

func supportForLocale(locale string) (supportView, error) {
	if err := validateProjectSupport(); err != nil {
		return supportView{}, err
	}
	audience := supportAudienceForLocale(locale)
	view := supportView{
		Audience:  audience,
		Reference: ProjectSupport.LocaleReferencePrefix + locale,
	}
	for _, method := range ProjectSupport.Methods {
		if method.Enabled && method.Audience == audience {
			view.Methods = append(view.Methods, method)
		}
	}
	if len(view.Methods) == 0 {
		return supportView{}, fmt.Errorf("support audience %q has no enabled payment methods", audience)
	}
	return view, nil
}

func validateProjectSupport() error {
	if ProjectSupport.LocaleReferencePrefix != "go-dev-" {
		return fmt.Errorf("invalid locale support reference prefix %q", ProjectSupport.LocaleReferencePrefix)
	}
	seen := make(map[string]bool, len(ProjectSupport.Methods))
	for _, method := range ProjectSupport.Methods {
		if method.Identity == "" || method.Name == "" || method.Kind == "" {
			return fmt.Errorf("support payment method identity, name, and kind are required")
		}
		if seen[method.Identity] {
			return fmt.Errorf("duplicate support payment method %q", method.Identity)
		}
		seen[method.Identity] = true
		if method.Audience != supportAudienceMainland && method.Audience != supportAudienceInternational {
			return fmt.Errorf("support payment method %q has invalid audience %q", method.Identity, method.Audience)
		}
		if !method.Enabled {
			continue
		}
		switch method.Kind {
		case "qr":
			clean := path.Clean(method.QRAsset)
			if method.Audience != supportAudienceMainland || clean != method.QRAsset || !strings.HasPrefix(clean, "images/support/") || path.Ext(clean) != ".png" {
				return fmt.Errorf("enabled QR payment method %q has invalid QR asset", method.Identity)
			}
			if method.Asset != "" || method.Network != "" || method.Address != "" || method.MinimumDeposit != "" || method.NetworkWarningKey != "" {
				return fmt.Errorf("enabled QR payment method %q mixes crypto fields", method.Identity)
			}
		case "crypto":
			if method.Audience != supportAudienceInternational || method.Asset == "" || method.Network == "" || method.Address == "" || method.MinimumDeposit == "" || method.NetworkWarningKey == "" || method.QRAsset != "" {
				return fmt.Errorf("enabled crypto payment method %q is incomplete", method.Identity)
			}
		default:
			return fmt.Errorf("enabled support payment method %q has unsupported kind %q", method.Identity, method.Kind)
		}
	}
	return nil
}

const (
	FrozenUpstreamCommit     = "db076098077c07d3cef1b85a2cf56ff52777f587"
	FrozenUpstreamCommitTime = "2026-09-10T14:36:53Z"
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
	return t.In(profile.TimeZone).Format("2006-01-02 15:04:05") + fmt.Sprintf(profile.TimeLabelFormat, profile.TimeLabel)
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
