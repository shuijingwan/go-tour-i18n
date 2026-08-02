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
	if err := initTour(mux, "SocketTransport"); err != nil {
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
