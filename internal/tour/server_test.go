// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/webtest"
)

func TestWeb(t *testing.T) {
	mux := http.NewServeMux()
	if err := initTour(mux, "SocketTransport", "en"); err != nil {
		t.Fatal(err)
	}
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/_/fmt", fmtHandler)
	fs := http.FileServer(http.FS(contentTour))
	mux.Handle("/favicon.ico", fs)
	mux.Handle("/images/", fs)
	webtest.TestHandler(t, "testdata/*.txt", mux)

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
			content, err := renderIndex(test.locale)
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
