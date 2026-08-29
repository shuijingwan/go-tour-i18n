// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/assets"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"github.com/shuijingwan/go-tour-i18n/internal/webtest"
)

func TestWeb(t *testing.T) {
	mux := http.NewServeMux()
	if _, err := initTour(mux, "SocketTransport", "en", ""); err != nil {
		t.Fatal(err)
	}
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/_/fmt", fmtHandler)
	fs := http.FileServer(http.FS(contentTour))
	mux.Handle("/favicon.ico", fs)
	mux.Handle("/images/", fs)
	webtest.TestHandler(t, "testdata/*.txt", mux)

	footer := httptest.NewRecorder()
	mux.ServeHTTP(footer, httptest.NewRequest(http.MethodGet, "/tour/footer.html", nil))
	if footer.Code != http.StatusOK {
		t.Fatalf("GET /tour/footer.html: status %d", footer.Code)
	}
	if got := footer.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("GET /tour/footer.html: Content-Type %q", got)
	}
	if !strings.Contains(footer.Body.String(), "github.com/shuijingwan/go-tour-i18n") {
		t.Fatal("GET /tour/footer.html does not contain the shared project links")
	}

	form := url.Values{"body": {"package main\nfunc main(){println(\"ok\")}\n"}}
	req := httptest.NewRequest(http.MethodPost, "/_/fmt", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /_/fmt: status %d, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "package main") {
		t.Fatalf("POST /_/fmt returned unexpected body: %s", rec.Body.String())
	}
}

func TestCourseSEORuntimeHasNoContentFallback(t *testing.T) {
	data, err := fs.ReadFile(contentTour, "tour/static/js/services.js")
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	for _, want := range []string{"config.descriptions[path]", "missing formal course description"} {
		if !strings.Contains(script, want) {
			t.Errorf("course SEO runtime is missing %q", want)
		}
	}
	for _, forbidden := range []string{"plainText", "page.Content", "substring(0, 200)", "querySelectorAll('pre')"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("course SEO runtime retained fallback %q", forbidden)
		}
	}
}

func TestRenderIndexLocales(t *testing.T) {
	tests := []struct {
		locale string
		want   []string
		absent []string
	}{
		{
			locale: "de-DE",
			want: []string{
				`<html lang="de-DE"`, "Eine Tour durch Go", `aria-label="Design wechseln"`, `alt="Systemdesign"`, `alt="Dunkles Design"`, `alt="Helles Design"`,
			},
			absent: []string{
				`aria-label="Toggle theme"`, `alt="System theme"`, `alt="Dark theme"`, `alt="Light theme"`,
			},
		},
		{
			locale: "en",
			want: []string{
				`<html lang="en"`, "A Tour of Go", `aria-label="Toggle theme"`, `alt="System theme"`, `alt="Dark theme"`, `alt="Light theme"`,
			},
		},
		{
			locale: "zh-CN",
			want: []string{
				`<html lang="zh-CN"`, "Go 语言之旅", `aria-label="切换主题"`, `alt="跟随系统主题"`, `alt="深色主题"`, `alt="浅色主题"`,
			},
			absent: []string{
				`aria-label="Toggle theme"`, `alt="System theme"`, `alt="Dark theme"`, `alt="Light theme"`,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.locale, func(t *testing.T) {
			catalog, err := ui.Load(test.locale)
			if err != nil {
				t.Fatal(err)
			}
			metadata, err := loadSiteMetadata(contentTour)
			if err != nil {
				t.Fatal(err)
			}
			content, err := renderIndex(catalog, metadata)
			if err != nil {
				t.Fatal(err)
			}
			page := string(content)
			for _, want := range test.want {
				if !strings.Contains(page, want) {
					t.Errorf("index for %s does not contain %q", test.locale, want)
				}
			}
			for _, absent := range test.absent {
				if strings.Contains(page, absent) {
					t.Errorf("index for %s unexpectedly contains %q", test.locale, absent)
				}
			}
		})
	}
}

func TestRenderedAssetURLsFollowLocaleAndEnvironment(t *testing.T) {
	development, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	production := development
	production.Development = false
	production.PublishedAt = "2026-08-12T07:23:34Z"

	for _, test := range []struct {
		name, locale string
		metadata     SiteMetadata
		prefix       string
	}{
		{"zh development", "zh-CN", development, ""},
		{"zh production", "zh-CN", production, ""},
		{"ja preview", "ja-JP", development, ""},
		{"ja production", "ja-JP", production, assets.BaseURL},
	} {
		t.Run(test.name, func(t *testing.T) {
			catalog, err := ui.Load(test.locale)
			if err != nil {
				t.Fatal(err)
			}
			metadata := test.metadata
			metadata.Locale = test.locale
			home, err := renderHome(catalog, metadata)
			if err != nil {
				t.Fatal(err)
			}
			index, err := renderIndex(catalog, metadata)
			if err != nil {
				t.Fatal(err)
			}
			pages := string(home) + string(index)
			for _, logicalPath := range []string{
				"tour/static/css/app.css",
				"tour/static/lib/codemirror/lib/codemirror.css",
				"images/site-logo.png",
				"images/site-logo-32.png",
				"images/go-logo-white.svg",
				"images/icons/brightness_6_gm_grey_24dp.svg",
				"images/icons/brightness_2_gm_grey_24dp.svg",
				"images/icons/light_mode_gm_grey_24dp.svg",
			} {
				want := test.prefix + "/" + logicalPath
				if !strings.Contains(pages, want) {
					t.Errorf("rendered pages do not contain asset URL %q", want)
				}
			}
			if !strings.Contains(string(index), `<script src="/tour/script.js"></script>`) {
				t.Error("Tour script is not language-origin relative")
			}
			if strings.Contains(string(index), assets.BaseURL+"/tour/script.js") {
				t.Error("Tour script unexpectedly uses shared assets origin")
			}
			for _, logicalPath := range []string{
				"tour/static/go-dev/course-ad.css",
				"tour/static/go-dev/course-ad.js",
			} {
				want := test.prefix + "/" + logicalPath
				if !strings.Contains(string(index), want) {
					t.Errorf("Tour index does not use locale-selected ad asset URL %q", want)
				}
			}
			if test.locale == "zh-CN" && strings.Contains(string(index), assets.BaseURL+"/tour/static/go-dev/course-ad") {
				t.Error("zh-CN Tour index unexpectedly uses the shared origin for course ad assets")
			}
		})
	}
}

func TestNonSharedRuntimePathsRemainLanguageOrigin(t *testing.T) {
	for path, want := range map[string]string{
		"tour/static/js/app.js":        "templateUrl: '/tour/static/partials/list.html'",
		"tour/static/js/directives.js": "templateUrl: '/tour/static/partials/toc.html'",
		"tour/static/js/services.js":   "$http.get('/tour/lesson/')",
		"tour/template/index.tmpl":     "{{template \"footer\" .}}",
		"tour/concurrency.article":     ".image /tour/static/img/tree.png",
	} {
		data, err := fs.ReadFile(contentTour, path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("%s does not retain %q", path, want)
		}
		if strings.Contains(string(data), assets.BaseURL) {
			t.Errorf("%s unexpectedly references shared assets origin", path)
		}
	}
}

func TestAppCSSUsesRelativeSharedGopher(t *testing.T) {
	data, err := fs.ReadFile(contentTour, "tour/static/css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)
	if !strings.Contains(css, "url(../img/gopher.png)") || strings.Contains(css, "url(/tour/static/img/gopher.png)") {
		t.Fatalf("app.css has unexpected gopher URL")
	}
	for _, base := range []string{"/tour/static/css/", assets.BaseURL + "/tour/static/css/"} {
		baseURL, err := url.Parse(base + "app.css")
		if err != nil {
			t.Fatal(err)
		}
		resolved := baseURL.ResolveReference(&url.URL{Path: "../img/gopher.png"}).String()
		want := strings.TrimSuffix(base, "css/") + "img/gopher.png"
		if resolved != want {
			t.Fatalf("gopher URL resolved to %q, want %q", resolved, want)
		}
	}
}

func TestRenderHomeDistinguishesDevelopmentAndProductionMetadata(t *testing.T) {
	catalog, err := ui.Load("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	development, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	if !development.Development || development.PublishedAt != "" {
		t.Fatalf("source metadata = %+v, want development metadata without published_at", development)
	}
	home, err := renderHome(catalog, development)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(home); !strings.Contains(got, "开发环境") || strings.Contains(got, "最近发布") {
		t.Fatalf("development homepage has unexpected release status: %s", got)
	}

	production := development
	production.Development = false
	production.PublishedAt = "2026-08-12T07:23:34Z"
	wantUpstreamCommitTime, err := production.UpstreamCommitTimeFor(localeProfiles["zh-CN"])
	if err != nil {
		t.Fatal(err)
	}
	home, err = renderHome(catalog, production)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(home); !strings.Contains(got, "最近发布") || !strings.Contains(got, "2026-08-12 15:23:34（北京时间）") || !strings.Contains(got, wantUpstreamCommitTime) || strings.Contains(got, "开发环境") {
		t.Fatalf("production homepage has unexpected release status: %s", got)
	}
}

func TestHomepageLanguageRegistryAndLocaleProfiles(t *testing.T) {
	if got, want := len(languageRegistry), 4; got != want {
		t.Fatalf("language registry length = %d, want %d", got, want)
	}
	for i, want := range []LanguageLink{
		{Locale: "zh-CN", Autonym: "简体中文", URL: "https://go-dev.shuijingwanwq.com/"},
		{Locale: "en", Autonym: "English", URL: "https://go.dev/tour/", Official: true},
		{Locale: "de-DE", Autonym: "Deutsch", URL: "https://de-go-dev.shuijingwanwq.com/"},
		{Locale: "ja-JP", Autonym: "日本語", URL: "https://ja-go-dev.shuijingwanwq.com/"},
	} {
		if got := languageRegistry[i]; got != want {
			t.Errorf("languageRegistry[%d] = %+v, want %+v", i, got, want)
		}
	}
	deLanguages, err := languagesFor("de-DE")
	if err != nil {
		t.Fatal(err)
	}
	deCurrent, err := currentLanguage(deLanguages)
	if err != nil {
		t.Fatal(err)
	}
	if deCurrent != (LanguageLink{Locale: "de-DE", Autonym: "Deutsch", URL: "https://de-go-dev.shuijingwanwq.com/", Current: true}) {
		t.Fatalf("de-DE current language = %+v", deCurrent)
	}
	deProfile := localeProfiles["de-DE"]
	if got, want := deProfile.TimeZone.String(), "Europe/Berlin"; got != want {
		t.Fatalf("de-DE time zone = %q, want %q", got, want)
	}
	if got, want := deProfile.DevelopmentLogURL, "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/"; got != want {
		t.Fatalf("de-DE development log URL = %q, want %q", got, want)
	}

	metadata := SiteMetadata{Locale: "zh-CN", PublishedAt: "2026-08-20T05:56:11Z", UpstreamCommit: FrozenUpstreamCommit, UpstreamCommitTime: FrozenUpstreamCommitTime, Pages: 122, Articles: 122}
	tests := []struct {
		locale, autonym, logURL, published, upstream, currentPublicURL string
	}{
		{"zh-CN", "简体中文", "https://www.shuijingwanwq.com/series/go-tour-chinese-edition-development-series/", "2026-08-20 13:56:11（北京时间）", "2026-08-27 05:55:26（北京时间）", languageRegistry[0].URL},
		{"de-DE", "Deutsch", "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/", "2026-08-20 07:56:11 (Ortszeit)", "2026-08-26 23:55:26 (Ortszeit)", languageRegistry[2].URL},
		{"ja-JP", "日本語", "https://en.shuijingwanwq.com/series/go-tour-chinese-edition-development-series-en/", "2026-08-20 14:56:11（日本時間）", "2026-08-27 06:55:26（日本時間）", languageRegistry[3].URL},
	}
	for _, test := range tests {
		t.Run(test.locale, func(t *testing.T) {
			catalog, err := ui.Load(test.locale)
			if err != nil {
				t.Fatal(err)
			}
			localizedMetadata := metadata
			localizedMetadata.Locale = test.locale
			data, err := newPageTemplateData(catalog, localizedMetadata)
			if err != nil {
				t.Fatal(err)
			}
			if data.CurrentLanguage.Autonym != test.autonym || data.DevelopmentLogURL != test.logURL || data.PublishedAt != test.published || data.UpstreamCommitTime != test.upstream {
				t.Errorf("localized page data = %+v", data)
			}
			homeBytes, err := renderHome(catalog, localizedMetadata)
			if err != nil {
				t.Fatal(err)
			}
			home := string(homeBytes)
			for _, want := range []string{"简体中文", "English", "Deutsch", "日本語", `aria-current="page">` + test.autonym, test.logURL, test.published, test.upstream, "© 2026 永夜", "蜀ICP备13001590号-1", `href="https://beian.miit.gov.cn/"`} {
				if !strings.Contains(home, want) {
					t.Errorf("homepage does not contain %q", want)
				}
			}
			if strings.Contains(home, `href="`+test.currentPublicURL+`">`+test.autonym+`</a>`) {
				t.Errorf("current locale language entry links to public URL %q", test.currentPublicURL)
			}
			if strings.Count(home, `href="`+test.logURL+`"`) != 2 {
				t.Errorf("development log URL is not shared by homepage and footer")
			}
			if strings.Contains(home, "site-card-grid") {
				t.Error("homepage still uses card grid for languages")
			}
		})
	}
}

func TestNewPageTemplateDataUsesMetadataUpstreamCommitTime(t *testing.T) {
	catalog, err := ui.Load("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	data, err := newPageTemplateData(catalog, metadata)
	if err != nil {
		t.Fatal(err)
	}
	want, err := metadata.UpstreamCommitTimeFor(localeProfiles["zh-CN"])
	if err != nil {
		t.Fatal(err)
	}
	if data.UpstreamCommitTime != want {
		t.Fatalf("UpstreamCommitTime = %q, want %q", data.UpstreamCommitTime, want)
	}

	metadata.UpstreamCommitTime = "not-a-time"
	if _, err := newPageTemplateData(catalog, metadata); err == nil || !strings.Contains(err.Error(), "parse upstream_commit_time") {
		t.Fatalf("newPageTemplateData() error = %v, want upstream_commit_time parse error", err)
	}
}

func TestRenderAnalyticsHTML(t *testing.T) {
	catalog, err := ui.Load("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	original := analyticsHTML
	t.Cleanup(func() { analyticsHTML = original })

	for _, test := range []struct {
		name  string
		value template.HTML
		want  string
	}{
		{name: "empty", value: "", want: ""},
		{name: "configured", value: `<script data-test="analytics"></script>`, want: `data-test="analytics"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			analyticsHTML = test.value

			home, err := renderHome(catalog, metadata)
			if err != nil {
				t.Fatalf("renderHome: %v", err)
			}
			tour, err := renderIndex(catalog, metadata)
			if err != nil {
				t.Fatalf("renderIndex: %v", err)
			}
			for name, content := range map[string][]byte{"home": home, "tour": tour} {
				page := string(content)
				if !strings.Contains(page, "<head>") {
					t.Errorf("%s page does not contain a head", name)
				}
				if got := strings.Count(page, test.want); test.want != "" && got != 1 {
					t.Errorf("%s page contains analytics marker %d times, want 1", name, got)
				}
				if test.want == "" && strings.Contains(page, "data-test=\"analytics\"") {
					t.Errorf("%s page contains analytics marker with empty analyticsHTML", name)
				}
			}
		})
	}
}

func TestRenderAdHTML(t *testing.T) {
	catalog, err := ui.Load("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		value template.HTML
		want  string
	}{
		{name: "empty", value: ""},
		{name: "configured", value: `<script data-test="ad"></script>`, want: `data-test="ad"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			adHTML = test.value
			home, err := renderHome(catalog, metadata)
			if err != nil {
				t.Fatal(err)
			}
			tour, err := renderIndex(catalog, metadata)
			if err != nil {
				t.Fatal(err)
			}
			for name, content := range map[string][]byte{"home": home, "tour": tour} {
				page := string(content)
				if test.want == "" && strings.Contains(page, `data-test="ad"`) {
					t.Errorf("%s contains ad when unconfigured", name)
				}
				if test.want != "" && strings.Count(page, test.want) != 1 {
					t.Errorf("%s does not contain exactly one configured ad", name)
				}
			}
		})
	}
}

func TestJSI18nBootstrapLocales(t *testing.T) {
	tests := []struct {
		locale string
		want   []string
		absent []string
	}{
		{
			locale: "de-DE",
			want: []string{
				"\"tour.list_heading\":\"Willkommen zu einer Tour durch Go\"", "\"toc.title\":\"Inhaltsverzeichnis\"", "\"execution.waiting\":\"Warten auf den Remote-Server …\"", "\"execution.exited\":\"Programm beendet\"", "\"execution.vet_failed\":\"Ausführung von go vet fehlgeschlagen.\"", "\"execution.build_failed\":\"Ausführung von go build fehlgeschlagen.\"", "\"execution.communication_error\":\"Fehler bei der Kommunikation mit dem Remote-Server.\"", "\"execution.test_failed\":\"{count} Test fehlgeschlagen.\"", "\"execution.tests_failed\":\"{count} Tests fehlgeschlagen.\"", "\"execution.tests_passed\":\"Alle Tests bestanden.\"", "\"feedback.open\":\"Feedback zu dieser Seite senden\"", "\"feedback.context\":\"Kontext\"",
				"\"editor.syntax\":\"Syntaxhervorhebung\"", "\"editor.imports\":\"Importe\"", "\"editor.run\":\"Ausführen\"", "\"editor.kill\":\"Stoppen\"", "\"editor.format\":\"Formatieren\"", "\"editor.reset\":\"Zurücksetzen\"",
			},
			absent: []string{
				"\"tour.list_heading\":\"Welcome to a tour of Go\"", "\"toc.title\":\"Table of Contents\"", "\"execution.waiting\":\"Waiting for remote server...\"", "\"execution.exited\":\"Program exited\"", "\"execution.vet_failed\":\"Go vet failed.\"", "\"execution.build_failed\":\"Go build failed.\"", "\"execution.communication_error\":\"Error communicating with remote server.\"", "\"execution.test_failed\":\"{count} test failed.\"", "\"execution.tests_failed\":\"{count} tests failed.\"", "\"execution.tests_passed\":\"All tests passed.\"", "\"feedback.open\":\"Send feedback about this page\"", "\"feedback.context\":\"Context\"",
				"\"editor.syntax\":\"Syntax\"", "\"editor.imports\":\"Imports\"", "\"editor.run\":\"Run\"", "\"editor.kill\":\"Kill\"", "\"editor.format\":\"Format\"", "\"editor.reset\":\"Reset\"",
			},
		},
		{
			locale: "en",
			want: []string{
				"\"tour.list_heading\":\"Welcome to a tour of Go\"", "\"toc.title\":\"Table of Contents\"", "\"execution.waiting\":\"Waiting for remote server...\"", "\"execution.exited\":\"Program exited\"", "\"execution.vet_failed\":\"Go vet failed.\"", "\"execution.build_failed\":\"Go build failed.\"", "\"execution.communication_error\":\"Error communicating with remote server.\"", "\"execution.test_failed\":\"{count} test failed.\"", "\"execution.tests_failed\":\"{count} tests failed.\"", "\"execution.tests_passed\":\"All tests passed.\"", "\"feedback.open\":\"Send feedback about this page\"", "\"feedback.context\":\"Context\"",
				"\"editor.syntax\":\"Syntax\"", "\"editor.imports\":\"Imports\"", "\"editor.run\":\"Run\"", "\"editor.kill\":\"Kill\"", "\"editor.format\":\"Format\"", "\"editor.reset\":\"Reset\"",
			},
		},
		{
			locale: "zh-CN",
			want: []string{
				"\"tour.list_heading\":\"欢迎来到 Go 语言之旅\"", "\"toc.title\":\"目录\"", "\"execution.waiting\":\"正在等待远程服务器……\"", "\"execution.exited\":\"程序已退出\"", "\"execution.vet_failed\":\"Go vet 检查失败。\"", "\"execution.build_failed\":\"Go 构建失败。\"", "\"execution.communication_error\":\"与远程服务器通信时出错。\"", "\"execution.test_failed\":\"{count} 个测试失败。\"", "\"execution.tests_failed\":\"{count} 个测试失败。\"", "\"execution.tests_passed\":\"所有测试均已通过。\"", "\"feedback.open\":\"发送本页反馈\"", "\"feedback.context\":\"上下文\"",
				"\"editor.syntax\":\"语法高亮\"", "\"editor.imports\":\"导入\"", "\"editor.run\":\"运行\"", "\"editor.kill\":\"终止\"", "\"editor.format\":\"格式化\"", "\"editor.reset\":\"重置\"",
			},
			absent: []string{
				"\"tour.list_heading\":\"Welcome to a tour of Go\"", "\"toc.title\":\"Table of Contents\"", "\"execution.waiting\":\"Waiting for remote server...\"", "\"execution.exited\":\"Program exited\"", "\"execution.vet_failed\":\"Go vet failed.\"", "\"execution.build_failed\":\"Go build failed.\"", "\"execution.communication_error\":\"Error communicating with remote server.\"", "\"execution.test_failed\":\"{count} test failed.\"", "\"execution.tests_failed\":\"{count} tests failed.\"", "\"execution.tests_passed\":\"All tests passed.\"", "\"feedback.open\":\"Send feedback about this page\"", "\"feedback.context\":\"Context\"",
				"\"editor.syntax\":\"Syntax\"", "\"editor.imports\":\"Imports\"", "\"editor.run\":\"Run\"", "\"editor.kill\":\"Kill\"", "\"editor.format\":\"Format\"", "\"editor.reset\":\"Reset\"",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.locale, func(t *testing.T) {
			catalog, err := ui.Load(test.locale)
			if err != nil {
				t.Fatal(err)
			}
			mux := http.NewServeMux()
			if err := initScript(mux, "", "SocketTransport", "", catalog, map[string]string{}, false); err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tour/script.js", nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("GET /tour/script.js: status %d", recorder.Code)
			}
			text := recorder.Body.String()
			for _, want := range test.want {
				if !strings.Contains(text, want) {
					t.Errorf("bootstrap for %s does not contain %q", test.locale, want)
				}
			}
			for _, absent := range test.absent {
				if strings.Contains(text, absent) {
					t.Errorf("bootstrap for %s unexpectedly contains %q", test.locale, absent)
				}
			}
		})
	}
}

func TestJSRouteMetadataUsesLocaleRegistryOrigins(t *testing.T) {
	for _, test := range []struct {
		locale string
		origin string
		title  string
	}{
		{"zh-CN", "https://go-dev.shuijingwanwq.com", "Go 语言之旅"},
		{"de-DE", "https://de-go-dev.shuijingwanwq.com", "Eine Tour durch Go"},
		{"ja-JP", "https://ja-go-dev.shuijingwanwq.com", "Go 言語ツアー"},
	} {
		t.Run(test.locale, func(t *testing.T) {
			catalog, err := ui.Load(test.locale)
			if err != nil {
				t.Fatal(err)
			}
			bootstrap, err := jsSEOBootstrap(catalog, map[string]string{}, false)
			if err != nil {
				t.Fatal(err)
			}
			text := string(bootstrap)
			for _, want := range []string{`window.__tourSEO = `, `"origin":"` + test.origin + `"`, `"siteTitle":"` + test.title + `"`, `"courseMetadataRequired":false`} {
				if !strings.Contains(text, want) {
					t.Errorf("SEO bootstrap for %s is missing %q: %s", test.locale, want, text)
				}
			}
		})
	}

	controllers, err := fs.ReadFile(contentTour, "tour/static/js/controllers.js")
	if err != nil {
		t.Fatal(err)
	}
	services, err := fs.ReadFile(contentTour, "tour/static/js/services.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"seo.page(l, page", "seo.list()"} {
		if !strings.Contains(string(controllers), want) {
			t.Errorf("controllers.js is missing SPA metadata update %q", want)
		}
	}
	for _, want := range []string{`doc.title = page.Title`, `link[rel="canonical"]`, `meta[name="description"]`, `data-tour-rendered-route`} {
		if !strings.Contains(string(services), want) {
			t.Errorf("services.js is missing route metadata behavior %q", want)
		}
	}
}

func TestLocalPlaygroundConfigurationKeepsRelativeHTTPEndpoints(t *testing.T) {
	catalog, err := ui.Load("en")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	if err := initScript(mux, "ws://127.0.0.1:3999/socket", "SocketTransport", "", catalog, map[string]string{}, false); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tour/script.js", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /tour/script.js: status %d", recorder.Code)
	}
	text := recorder.Body.String()
	for _, want := range []string{
		`window.transport = SocketTransport();`,
		`window.socketAddr = "ws://127.0.0.1:3999/socket";`,
		`window.playgroundBaseURL = "";`,
		`$.ajax(playgroundURL('/_/compile?backend='`,
		`$.ajax(playgroundURL('/_/fmt?backend='`,
		`$http.post(playgroundURL('/_/fmt')`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("local script does not contain %q", want)
		}
	}
	if strings.Contains(text, productionPlaygroundBaseURL) {
		t.Fatalf("local script contains production Playground URL")
	}
}

func TestPlaygroundBaseURLIsJavaScriptStringEscaped(t *testing.T) {
	catalog, err := ui.Load("en")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	baseURL := `https://example.test/";alert(1)//</script>`
	if err := initScript(mux, "", "HTTPTransport", baseURL, catalog, map[string]string{}, false); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tour/script.js", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /tour/script.js: status %d", recorder.Code)
	}
	text := recorder.Body.String()
	if strings.Contains(text, `window.playgroundBaseURL = "https://example.test/";alert(1)//</script>";`) {
		t.Fatal("Playground base URL was injected without JavaScript escaping")
	}
	encoded, err := json.Marshal(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	want := "window.playgroundBaseURL = " + string(encoded) + ";"
	if !strings.Contains(text, want) {
		t.Fatalf("escaped Playground base URL missing from script")
	}
}

func TestPlaygroundUsesBootstrappedExitedMessage(t *testing.T) {
	data, err := fs.ReadFile(contentTour, "js/playground.js")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `window.__tourUIMessages['execution.exited']`) {
		t.Fatal("playground does not read execution.exited from the shared UI messages")
	}
	if strings.Contains(text, "Program exited") {
		t.Fatal("playground still hard-codes Program exited")
	}
	if !strings.Contains(text, `(m ? ': ' + m : '.')`) {
		t.Fatal("playground changed end-event status/reason formatting")
	}
	httpTransportStart := strings.Index(text, "function HTTPTransport")
	socketTransportStart := strings.Index(text, "function SocketTransport")
	if httpTransportStart < 0 || socketTransportStart < 0 || socketTransportStart <= httpTransportStart {
		t.Fatal("playground does not contain the expected HTTPTransport boundary")
	}
	httpTransport := text[httpTransportStart:socketTransportStart]
	for _, key := range []string{"execution.vet_failed", "execution.build_failed", "execution.communication_error", "execution.test_failed", "execution.tests_failed", "execution.tests_passed"} {
		if !strings.Contains(httpTransport, `window.__tourUIMessages['`+key+`']`) {
			t.Errorf("playground does not read %q from the shared UI messages", key)
		}
	}
	for _, message := range []string{"Go vet failed.", "Go build failed.", "Error communicating with remote server.", "All tests passed.", " test failed."} {
		if strings.Contains(httpTransport, message) {
			t.Errorf("playground still hard-codes runtime message %q", message)
		}
	}
	for _, want := range []string{`'\n' + window.__tourUIMessages['execution.vet_failed'] + '\n\n'`, `testsFailed == 1`, `execution.test_failed`, `execution.tests_failed`} {
		if !strings.Contains(httpTransport, want) {
			t.Errorf("playground changed required runtime formatting or plural branch %q", want)
		}
	}
}

func TestChineseHomeProjectCardUsesTourTitle(t *testing.T) {
	catalog, err := ui.Load("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	home, err := renderHome(catalog, metadata)
	if err != nil {
		t.Fatal(err)
	}
	text := string(home)
	if !strings.Contains(text, "<h3>Go 语言之旅</h3>") {
		t.Fatal("Chinese homepage project card does not use tour.title")
	}
	if strings.Contains(text, "<h3>A Tour of Go</h3>") {
		t.Fatal("Chinese homepage project card still hard-codes A Tour of Go")
	}
}

func TestRenderHomeUsesLocaleSEOIdentity(t *testing.T) {
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		locale, description, canonical, otherCanonical string
	}{
		{"zh-CN", "这是一个由社区维护的 Go 官方学习内容翻译项目。当前首先提供简体中文，后续可自然扩展至其他语言和其他 Go 内容。", "https://go-dev.shuijingwanwq.com/", "https://ja-go-dev.shuijingwanwq.com/"},
		{"de-DE", "Ein von der Community betreutes Übersetzungsprojekt für offizielle Go-Lerninhalte. Als erste Sprache wurde vereinfachtes Chinesisch bereitgestellt; die Struktur lässt sich auf weitere Sprachen und Go-Inhalte erweitern.", "https://de-go-dev.shuijingwanwq.com/", "https://ja-go-dev.shuijingwanwq.com/"},
		{"ja-JP", "Go の公式学習コンテンツをコミュニティで翻訳・維持するプロジェクトです。最初に利用できる言語は簡体字中国語で、今後さらに多くの言語や Go コンテンツに対応できる構成になっています。", "https://ja-go-dev.shuijingwanwq.com/", "https://go-dev.shuijingwanwq.com/"},
	} {
		t.Run(test.locale, func(t *testing.T) {
			catalog, err := ui.Load(test.locale)
			if err != nil {
				t.Fatal(err)
			}
			home, err := renderHome(catalog, metadata)
			if err != nil {
				t.Fatal(err)
			}
			text := string(home)
			if !strings.Contains(text, `<meta name="description" content="`+test.description+`">`) {
				t.Errorf("homepage description does not use locale UI catalog: %s", text)
			}
			if !strings.Contains(text, `<link rel="canonical" href="`+test.canonical+`">`) || strings.Contains(text, `<link rel="canonical" href="`+test.otherCanonical+`">`) {
				t.Errorf("homepage canonical has wrong locale identity: %s", text)
			}
		})
	}
}

func TestEditorTemplateUsesLocalizedPlainTextBindings(t *testing.T) {
	data, err := fs.ReadFile(contentTour, "tour/static/partials/editor.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, binding := range []string{"editorUI.syntax", "editorUI.imports", "editorUI.run", "editorUI.kill", "editorUI.format", "editorUI.reset"} {
		if !strings.Contains(text, "{{"+binding+"}}") {
			t.Errorf("editor template is missing binding %q", binding)
		}
	}
	for _, literal := range []string{">Syntax<", ">Imports<", ">Run<", ">Kill<", ">Format<", ">Reset<"} {
		if strings.Contains(text, literal) {
			t.Errorf("editor template still hard-codes %q", literal)
		}
	}
}

func TestCourseAdMountFollowsModuleBar(t *testing.T) {
	data, err := fs.ReadFile(contentTour, "tour/static/partials/editor.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	mount := `<div class="go-dev-course-ad" data-go-dev-course-ad course-ad></div>`
	if strings.Count(text, mount) != 1 {
		t.Fatalf("course ad mount count = %d, want 1", strings.Count(text, mount))
	}
	moduleStart := strings.Index(text, `<div class="bar module-bar">`)
	mountStart := strings.Index(text, mount)
	if moduleStart < 0 || mountStart < 0 {
		t.Fatal("course ad mount or module bar is missing")
	}
	moduleEnd := strings.Index(text[moduleStart:], "</div>")
	if moduleEnd < 0 {
		t.Fatal("module bar has no closing tag")
	}
	between := text[moduleStart+moduleEnd+len("</div>") : mountStart]
	if strings.TrimSpace(between) != "" {
		t.Fatalf("course ad mount does not immediately follow module bar: %q", between)
	}
}

func TestCourseAdAssetsUseAngularViewLifecycle(t *testing.T) {
	cssData, err := fs.ReadFile(contentTour, "tour/static/go-dev/course-ad.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(cssData)
	for _, forbidden := range []string{"position: fixed", "position: absolute", "position: sticky", "height:", "min-height:", "vh", "viewport"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("course ad CSS contains forbidden layout behavior %q", forbidden)
		}
	}
	mobileStart := strings.Index(css, "@media (max-width: 600px)")
	if mobileStart < 0 {
		t.Fatal("course ad CSS has no mobile media query")
	}
	mobileCSS := css[mobileStart:]
	if !strings.Contains(mobileCSS, "margin-top: 24px") || strings.Contains(mobileCSS, "margin-top: auto") {
		t.Error("mobile course ad does not keep ordinary document-flow spacing")
	}
	if strings.Contains(css, "max-width: 620px") {
		t.Error("course ad CSS retains the deprecated desktop width cap")
	}
	for _, want := range []string{
		"#left-side > .relative-content",
		"display: flex",
		"flex-direction: column",
		"width: calc(100% - 80px)",
		"margin-top: auto",
		"margin-bottom: 24px",
		".go-dev-course-ad--max-336",
		"max-width: 336px",
		".go-dev-course-ad--max-468",
		"max-width: 468px",
		".go-dev-course-ad--max-728",
		"max-width: 728px",
		"@media (max-width: 600px)",
		"margin-top: 24px",
		`ins.adsbygoogle[data-ad-status="unfilled"]`,
		"display: none !important",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("course ad CSS is missing %q", want)
		}
	}

	jsData, err := fs.ReadFile(contentTour, "tour/static/go-dev/course-ad.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(jsData)
	for _, want := range []string{
		"function removeAdSenseHeightOverrides(element)",
		"function protectLayoutForMount(element)",
		"function stopProtectingLayout(element)",
		"new MutationObserver",
		"attributeFilter: ['style']",
		"element.style.getPropertyValue('height') === 'auto'",
		"element.style.getPropertyValue('min-height') === '0px'",
		"document.createElement('ins')",
		"ad.className = 'adsbygoogle'",
		"data-ad-client', 'ca-pub-8392190980622725'",
		"experimentStorageKey = 'goDevCourseAdExperimentGroup'",
		"{name: 'A', slot: '3362554728', maxWidth: 336}",
		"{name: 'B', slot: '1260537939', maxWidth: 468}",
		"{name: 'C', slot: '4220340824', maxWidth: 728}",
		"{name: 'D', slot: '4728596962'}",
		"Math.floor(Math.random() * experimentGroups.length)",
		"window.sessionStorage.getItem(experimentStorageKey)",
		"window.sessionStorage.setItem(experimentStorageKey, group.name)",
		"data-go-dev-course-ad-group",
		"data-ad-format', 'auto'",
		"data-full-width-responsive', 'true'",
		"function unmount(element)",
		"window.goDevCourseAd",
		"try {",
		"console.warn('course ad request failed', error)",
		"(window.adsbygoogle = window.adsbygoogle || []).push({})",
	} {
		if !strings.Contains(js, want) {
			t.Errorf("course ad JS is missing %q", want)
		}
	}
	for _, forbidden := range []string{"test ad", "googlesyndication", "pagead2", "document.body", "mountall", "mountwithin", "currentmount", "layoutobserver", "setinterval", "settimeout"} {
		if strings.Contains(strings.ToLower(js), forbidden) {
			t.Errorf("course ad JS contains forbidden dependency %q", forbidden)
		}
	}
	if got := strings.Count(js, ".push({})"); got != 1 {
		t.Errorf("course ad JS push count = %d, want 1", got)
	}
	for _, forbidden := range []string{"970", "data-ad-width", "data-ad-height"} {
		if strings.Contains(js, forbidden) {
			t.Errorf("course ad JS contains unsupported experiment behavior %q", forbidden)
		}
	}
	for _, forbidden := range []string{"removeAttribute('style')", `setAttribute('style', '')`} {
		if strings.Contains(js, forbidden) {
			t.Errorf("course ad JS contains broad style cleanup %q", forbidden)
		}
	}
	directivesData, err := fs.ReadFile(contentTour, "tour/static/js/directives.js")
	if err != nil {
		t.Fatal(err)
	}
	directives := string(directivesData)
	for _, want := range []string{"directive('courseAd'", "lifecycle.mount(elm[0])", "scope.$on('$destroy'", "lifecycle.unmount(elm[0])"} {
		if !strings.Contains(directives, want) {
			t.Errorf("course ad directive is missing %q", want)
		}
	}
}

func TestJSModuleBootstrapLocales(t *testing.T) {
	tests := []struct {
		locale string
		want   []string
		absent []string
	}{
		{
			locale: "en",
			want: []string{
				`"mechanics":{"title":"Using the tour","description":"\u003cp\u003eWelcome to a tour of the \u003ca href=\"https://go.dev\"\u003eGo programming language\u003c/a\u003e. The tour covers the most important features of the language, mainly:\u003c/p\u003e"}`,
				`"basics":{"title":"Basics"`, `"methods":{"title":"Methods and interfaces"`, `"generics":{"title":"Generics"`, `"concurrency":{"title":"Concurrency"`,
				"The starting point, learn all the basics of the language.\\u003c/p\\u003e\\u003cp\\u003eDeclaring variables,", "Learn how to define methods on types, how to declare interfaces, and how to put everything together.\\u003c/p\\u003e", "Learn how to use type parameters in Go functions and structs.\\u003c/p\\u003e", "Go provides concurrency features as part of the core language.\\u003c/p\\u003e\\u003cp\\u003eThis module goes over goroutines",
			},
		},
		{
			locale: "zh-CN",
			want: []string{
				`"mechanics":{"title":"使用本教程","description":"\u003cp\u003e欢迎来到 \u003ca href=\"https://go.dev\"\u003eGo 编程语言\u003c/a\u003e之旅。本教程主要介绍该语言最重要的特性：\u003c/p\u003e"}`,
				`"basics":{"title":"基础"`, `"methods":{"title":"方法和接口"`, `"generics":{"title":"泛型"`, `"concurrency":{"title":"并发"`,
				"从这里开始，学习这门语言的基础知识。\\u003c/p\\u003e\\u003cp\\u003e声明变量、调用函数，以及进入下一课前需要了解的一切。", "学习如何为类型定义方法、如何声明接口，以及如何将它们组合起来。\\u003c/p\\u003e", "学习如何在 Go 函数和结构体中使用类型参数。\\u003c/p\\u003e", "Go 在语言层面提供了并发支持。\\u003c/p\\u003e\\u003cp\\u003e本模块介绍 goroutine 和 channel，以及如何使用它们实现不同的并发模式。",
			},
			absent: []string{
				`"mechanics":{"title":"Using the tour"`, `"basics":{"title":"Basics"`, `"methods":{"title":"Methods and interfaces"`, `"generics":{"title":"Generics"`, `"concurrency":{"title":"Concurrency"`,
				"The starting point, learn all the basics of the language.\\u003c/p\\u003e\\u003cp\\u003eDeclaring variables,", "Learn how to define methods on types, how to declare interfaces, and how to put everything together.\\u003c/p\\u003e", "Learn how to use type parameters in Go functions and structs.\\u003c/p\\u003e", "Go provides concurrency features as part of the core language.\\u003c/p\\u003e\\u003cp\\u003eThis module goes over goroutines",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.locale, func(t *testing.T) {
			catalog, err := ui.Load(test.locale)
			if err != nil {
				t.Fatal(err)
			}
			bootstrap, err := jsModuleBootstrap(catalog)
			if err != nil {
				t.Fatal(err)
			}
			text := string(bootstrap)
			for _, want := range test.want {
				if strings.Contains(want, "channel") {
					continue // The Chinese catalog intentionally uses the localized term.
				}
				if !strings.Contains(text, want) {
					t.Errorf("module bootstrap for %s does not contain %q", test.locale, want)
				}
			}
			for _, absent := range test.absent {
				if strings.Contains(text, absent) {
					t.Errorf("module bootstrap for %s unexpectedly contains %q", test.locale, absent)
				}
			}
		})
	}
}

func TestEnglishModuleDescriptionsMatchFrozenValues(t *testing.T) {
	source, err := exec.Command("git", "show", "519cc38:_content/tour/static/js/values.js").Output()
	if err != nil {
		t.Fatalf("read frozen values.js: %v", err)
	}
	catalog, err := ui.Load("en")
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]string{"mechanics": "module.using_tour.description", "basics": "module.basics.description", "methods": "module.methods.description", "generics": "module.generics.description", "concurrency": "module.concurrency.description"}
	entries := regexp.MustCompile(`(?m)^\s*'id': '([^']+)',\n\s*'title': '[^']*',\n\s*'description': '([^']*)',$`).FindAllStringSubmatch(string(source), -1)
	if len(entries) != 5 {
		t.Fatalf("frozen values.js description entries = %d, want 5", len(entries))
	}
	for _, entry := range entries {
		key := keys[entry[1]]
		message, ok := catalog.Messages[key]
		if !ok || message.Kind != "rich" || message.Text != entry[2] {
			t.Errorf("%s does not match frozen rich description", key)
		}
	}
	if !strings.Contains(catalog.Messages["module.using_tour.description"].Text, "https://go.dev") {
		t.Error("frozen mechanics description lost go.dev link")
	}
}

func TestChineseModuleDescriptionsAreRich(t *testing.T) {
	catalog, err := ui.Load("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"module.using_tour.description":  `<p>欢迎来到 <a href="https://go.dev">Go 编程语言</a>之旅。本教程主要介绍该语言最重要的特性：</p>`,
		"module.basics.description":      `<p>从这里开始，学习这门语言的基础知识。</p><p>声明变量、调用函数，以及进入下一课前需要了解的一切。</p>`,
		"module.methods.description":     `<p>学习如何为类型定义方法、如何声明接口，以及如何将它们组合起来。</p>`,
		"module.generics.description":    `<p>学习如何在 Go 函数和结构体中使用类型参数。</p>`,
		"module.concurrency.description": `<p>Go 在语言层面提供了并发支持。</p><p>本模块介绍 goroutine 和通道，以及如何使用它们实现不同的并发模式。</p>`,
	}
	for key, text := range want {
		message, ok := catalog.Messages[key]
		if !ok || message.Kind != "rich" || message.Text != text {
			t.Errorf("%s = %#v, want rich %q", key, message, text)
		}
	}
}

func TestJSModuleBootstrapRejectsMissingOrNonRichDescription(t *testing.T) {
	catalog, err := ui.Load("en")
	if err != nil {
		t.Fatal(err)
	}
	delete(catalog.Messages, "module.basics.description")
	if _, err := jsModuleBootstrap(catalog); err == nil || !strings.Contains(err.Error(), "is missing") {
		t.Fatalf("missing rich message error = %v", err)
	}

	catalog, err = ui.Load("en")
	if err != nil {
		t.Fatal(err)
	}
	catalog.Messages["module.basics.description"] = ui.Message{Kind: "plain", Text: "Basics"}
	if _, err := jsModuleBootstrap(catalog); err == nil || !strings.Contains(err.Error(), "not rich") {
		t.Fatalf("non-rich message error = %v", err)
	}
}

func TestTableOfContentsNavigationStructure(t *testing.T) {
	data, err := fs.ReadFile(contentTour, "tour/static/js/values.js")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	fragments := []string{
		"'id': 'mechanics',", "'lessons': ['welcome']",
		"'id': 'basics',", "'lessons': ['basics', 'flowcontrol', 'moretypes']",
		"'id': 'methods',", "'lessons': ['methods']",
		"'id': 'generics',", "'lessons': ['generics']",
		"'id': 'concurrency',", "'lessons': ['concurrency']",
	}
	position := 0
	for _, fragment := range fragments {
		next := strings.Index(text[position:], fragment)
		if next < 0 {
			t.Fatalf("tableOfContents is missing %q", fragment)
		}
		position += next + len(fragment)
	}
}

func TestListDescriptionUsesFrozenRichBinding(t *testing.T) {
	data, err := fs.ReadFile(contentTour, "tour/static/partials/list.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `ng-bind-html-unsafe="m.description"`) {
		t.Fatal("list description does not use frozen rich binding")
	}
}

func TestJSI18nBootstrapEscapesJSON(t *testing.T) {
	catalog, err := ui.Load("en")
	if err != nil {
		t.Fatal(err)
	}
	catalog.Messages["feedback.open"] = ui.Message{Kind: "plain", Text: `<>&"`}
	bootstrap, err := jsI18nBootstrap(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(bootstrap); strings.ContainsAny(got, "<>&") || !strings.Contains(got, `\u003c\u003e\u0026\"`) {
		t.Fatalf("bootstrap does not safely JSON-escape message: %q", got)
	}
}

func TestJSI18nBootstrapRejectsInvalidMessageKind(t *testing.T) {
	catalog, err := ui.Load("en")
	if err != nil {
		t.Fatal(err)
	}
	catalog.Messages["toc.title"] = ui.Message{Kind: "unexpected", Text: "Table of Contents"}
	if _, err := jsI18nBootstrap(catalog); err == nil || !strings.Contains(err.Error(), "not plain") {
		t.Fatalf("jsI18nBootstrap error = %v, want plain-message failure", err)
	}
}

func TestRegisterHandlersLocale(t *testing.T) {
	tests := []struct {
		locale string
		want   string
	}{
		{locale: "de-DE", want: `aria-label="Design wechseln"`},
		{locale: "en", want: `aria-label="Toggle theme"`},
		{locale: "zh-CN", want: `aria-label="切换主题"`},
	}
	for _, test := range tests {
		t.Run(test.locale, func(t *testing.T) {
			if err := RegisterHandlersLocale(http.NewServeMux(), test.locale); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(uiContent), test.want) {
				t.Errorf("HTTPTransport UI for %s does not contain %q", test.locale, test.want)
			}
		})
	}

	if err := RegisterHandlersLocale(http.NewServeMux(), "ja"); err == nil || !strings.Contains(err.Error(), "unknown UI locale") {
		t.Fatalf("RegisterHandlersLocale(ja) error = %v, want unknown locale", err)
	}
}
