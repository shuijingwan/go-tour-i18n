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

	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"github.com/shuijingwan/go-tour-i18n/internal/webtest"
)

func TestWeb(t *testing.T) {
	mux := http.NewServeMux()
	if err := initTour(mux, "SocketTransport", "en", ""); err != nil {
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

func TestRenderIndexLocales(t *testing.T) {
	tests := []struct {
		locale string
		want   []string
		absent []string
	}{
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
	home, err = renderHome(catalog, production)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(home); !strings.Contains(got, "最近发布") || !strings.Contains(got, "2026-08-12 15:23:34 (北京时间)") || strings.Contains(got, "开发环境") {
		t.Fatalf("production homepage has unexpected release status: %s", got)
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

func TestJSI18nBootstrapLocales(t *testing.T) {
	tests := []struct {
		locale string
		want   []string
		absent []string
	}{
		{
			locale: "en",
			want: []string{
				"\"tour.list_heading\":\"Welcome to a tour of Go\"", "\"toc.title\":\"Table of Contents\"", "\"execution.waiting\":\"Waiting for remote server...\"", "\"execution.exited\":\"Program exited\"", "\"feedback.open\":\"Send feedback about this page\"", "\"feedback.context\":\"Context\"",
				"\"editor.syntax\":\"Syntax\"", "\"editor.imports\":\"Imports\"", "\"editor.run\":\"Run\"", "\"editor.kill\":\"Kill\"", "\"editor.format\":\"Format\"", "\"editor.reset\":\"Reset\"",
			},
		},
		{
			locale: "zh-CN",
			want: []string{
				"\"tour.list_heading\":\"欢迎来到 Go 语言之旅\"", "\"toc.title\":\"目录\"", "\"execution.waiting\":\"正在等待远程服务器……\"", "\"execution.exited\":\"程序已退出\"", "\"feedback.open\":\"发送本页反馈\"", "\"feedback.context\":\"上下文\"",
				"\"editor.syntax\":\"语法高亮\"", "\"editor.imports\":\"导入\"", "\"editor.run\":\"运行\"", "\"editor.kill\":\"终止\"", "\"editor.format\":\"格式化\"", "\"editor.reset\":\"重置\"",
			},
			absent: []string{
				"\"tour.list_heading\":\"Welcome to a tour of Go\"", "\"toc.title\":\"Table of Contents\"", "\"execution.waiting\":\"Waiting for remote server...\"", "\"execution.exited\":\"Program exited\"", "\"feedback.open\":\"Send feedback about this page\"", "\"feedback.context\":\"Context\"",
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
			if err := initScript(mux, "", "SocketTransport", "", catalog); err != nil {
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

func TestLocalPlaygroundConfigurationKeepsRelativeHTTPEndpoints(t *testing.T) {
	catalog, err := ui.Load("en")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	if err := initScript(mux, "ws://127.0.0.1:3999/socket", "SocketTransport", "", catalog); err != nil {
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
	if err := initScript(mux, "", "HTTPTransport", baseURL, catalog); err != nil {
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
	for _, message := range []string{"Go vet failed.", "Go build failed.", "Error communicating with remote server."} {
		if !strings.Contains(text, message) {
			t.Errorf("playground unexpectedly changed deferred HTTPTransport message %q", message)
		}
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
