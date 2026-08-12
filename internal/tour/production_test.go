package tour

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	website "github.com/shuijingwan/go-tour-i18n"
)

func TestProductionPlaygroundUsesGoDev(t *testing.T) {
	if productionPlaygroundURL != "https://go.dev/_" {
		t.Fatalf("production Playground URL = %q, want https://go.dev/_", productionPlaygroundURL)
	}
	proxy := mustPlaygroundProxy(t, productionPlaygroundURL)
	if proxy.compileURL != "https://go.dev/_/compile" {
		t.Fatalf("compile upstream URL = %q, want https://go.dev/_/compile", proxy.compileURL)
	}
	if proxy.formatURL != "https://go.dev/_/fmt" {
		t.Fatalf("format upstream URL = %q, want https://go.dev/_/fmt", proxy.formatURL)
	}
	if strings.Contains(proxy.compileURL, "play.golang.org") || strings.Contains(proxy.formatURL, "play.golang.org") {
		t.Fatal("production Playground proxy still references play.golang.org")
	}
}

func TestProductionHandlerUsesHTTPTransportAndServesTour(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compile":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"Events":[{"Message":"ok\n","Kind":"stdout","Delay":0}]}`)
		case "/fmt":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"Body":"package main\n","Error":""}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	handler := productionTestHandler(t, upstream.URL)

	tests := []struct {
		path       string
		wantStatus int
		want       string
	}{
		{"/", http.StatusOK, "A Tour of Go 多语言翻译项目"},
		{"/tour/", http.StatusOK, "Go 语言之旅"},
		{"/tour/list", http.StatusOK, "Go 语言之旅"},
		{"/tour/lesson/welcome", http.StatusOK, `"Title":"Welcome!"`},
		{"/tour/static/css/app.css", http.StatusOK, "body"},
		{"/images/go-logo-white.svg", http.StatusOK, "<svg"},
	}

	home := httptest.NewRecorder()
	handler.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/", nil))
	for _, want := range []string{"非官方社区多语言翻译项目", "GitHub 项目源码", "蜀ICP备13001590号-1", "开发环境", "golang/website@e11dacba"} {
		if !strings.Contains(home.Body.String(), want) {
			t.Errorf("homepage does not contain %q", want)
		}
	}
	if strings.Contains(home.Body.String(), "最近发布") {
		t.Error("development homepage renders a production release timestamp")
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, test.path, nil))
			if rec.Code != test.wantStatus {
				t.Fatalf("GET %s: status=%d body=%s", test.path, rec.Code, rec.Body.String())
			}
			if test.want != "" && !strings.Contains(rec.Body.String(), test.want) {
				t.Errorf("GET %s does not contain %q", test.path, test.want)
			}
		})
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tour/script.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /tour/script.js: status=%d", rec.Code)
	}
	script := rec.Body.String()
	for _, want := range []string{`window.transport = HTTPTransport();`, `window.socketAddr = "";`} {
		if !strings.Contains(script, want) {
			t.Errorf("production script does not contain %q", want)
		}
	}
	for _, forbidden := range []string{`window.transport = SocketTransport();`, `window.socketAddr = "ws://`, `window.socketAddr = "wss://`} {
		if strings.Contains(script, forbidden) {
			t.Errorf("production script contains %q", forbidden)
		}
	}
}

func TestProductionHandlerRejectsSocketAndReservedRoutes(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	defer upstream.Close()
	handler := productionTestHandler(t, upstream.URL)

	tests := []struct {
		name    string
		path    string
		upgrade bool
	}{
		{"socket", "/socket", false},
		{"socket upgrade", "/socket", true},
		{"socket subtree", "/socket/anything", true},
		{"share", "/_/share", false},
		{"unknown reserved", "/_/anything", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			if test.upgrade {
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("GET %s: status=%d body=%s", test.path, rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
				t.Errorf("GET %s fell through to SPA HTML", test.path)
			}
		})
	}
}

func TestProductionCompileProxy(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/compile" || r.Method != http.MethodPost {
			t.Fatalf("upstream request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("upstream content type = %q", got)
		}
		if got := r.Header.Get("X-User-Header"); got != "" {
			t.Errorf("user header was forwarded: %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("version") != "2" || r.Form.Get("body") != "package main\nfunc main(){}\n" || r.Form.Get("withVet") != "true" {
			t.Errorf("compile form = %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"Errors":"","Events":[{"Message":"ok\n","Kind":"stdout","Delay":0}],"VetErrors":""}`)
	}))
	defer upstream.Close()

	proxy := mustPlaygroundProxy(t, upstream.URL)
	form := url.Values{
		"version": {"2"},
		"body":    {"package main\nfunc main(){}\n"},
		"withVet": {"true"},
	}
	req := formRequest(http.MethodPost, "/_/compile?backend=attacker.invalid", form)
	req.Header.Set("X-User-Header", "secret")
	rec := httptest.NewRecorder()
	proxy.compile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("compile status=%d body=%s", rec.Code, rec.Body.String())
	}
	if calls.Load() != 1 || !strings.Contains(rec.Body.String(), `"Message":"ok\n"`) {
		t.Fatalf("compile calls=%d body=%s", calls.Load(), rec.Body.String())
	}
}

func TestProductionCompileProxyRejectsUnsafeRequestsAndResponses(t *testing.T) {
	t.Run("method", func(t *testing.T) {
		var calls atomic.Int32
		upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
		defer upstream.Close()
		proxy := mustPlaygroundProxy(t, upstream.URL)
		rec := httptest.NewRecorder()
		proxy.compile(rec, httptest.NewRequest(http.MethodGet, "/_/compile", nil))
		if rec.Code != http.StatusMethodNotAllowed || calls.Load() != 0 {
			t.Fatalf("status=%d calls=%d", rec.Code, calls.Load())
		}
	})

	t.Run("request too large", func(t *testing.T) {
		var calls atomic.Int32
		upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
		defer upstream.Close()
		proxy := mustPlaygroundProxy(t, upstream.URL)
		form := url.Values{"body": {strings.Repeat("x", playgroundRequestLimit)}}
		rec := httptest.NewRecorder()
		proxy.compile(rec, formRequest(http.MethodPost, "/_/compile", form))
		if rec.Code != http.StatusRequestEntityTooLarge || calls.Load() != 0 {
			t.Fatalf("status=%d calls=%d", rec.Code, calls.Load())
		}
	})

	t.Run("upstream error", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "failure", http.StatusInternalServerError)
		}))
		defer upstream.Close()
		proxy := mustPlaygroundProxy(t, upstream.URL)
		rec := httptest.NewRecorder()
		proxy.compile(rec, formRequest(http.MethodPost, "/_/compile", url.Values{"body": {"package main"}}))
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("response too large", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, strings.Repeat("x", playgroundResponseLimit+1))
		}))
		defer upstream.Close()
		proxy := mustPlaygroundProxy(t, upstream.URL)
		rec := httptest.NewRecorder()
		proxy.compile(rec, formRequest(http.MethodPost, "/_/compile", url.Values{"body": {"package main"}}))
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("redirect rejected", func(t *testing.T) {
		var escaped atomic.Int32
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			escaped.Add(1)
			_, _ = io.WriteString(w, `{}`)
		}))
		defer target.Close()
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL, http.StatusFound)
		}))
		defer upstream.Close()
		proxy := mustPlaygroundProxy(t, upstream.URL)
		rec := httptest.NewRecorder()
		proxy.compile(rec, formRequest(http.MethodPost, "/_/compile", url.Values{"body": {"package main"}}))
		if rec.Code != http.StatusBadGateway || escaped.Load() != 0 {
			t.Fatalf("status=%d escaped=%d", rec.Code, escaped.Load())
		}
	})
}

func TestProductionFormatProxy(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/fmt" || r.Method != http.MethodPost {
			t.Fatalf("upstream request = %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("body") != "package main\nfunc main(){}" || r.Form.Get("imports") != "true" {
			t.Errorf("upstream form = %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"Body":"package main\n\nfunc main() {}\n","Error":""}`)
	}))
	defer upstream.Close()
	proxy := mustPlaygroundProxy(t, upstream.URL)
	rec := httptest.NewRecorder()
	proxy.format(rec, formRequest(http.MethodPost, "/_/fmt?backend=attacker.invalid", url.Values{
		"body": {"package main\nfunc main(){}"}, "imports": {"true"},
	}))
	if rec.Code != http.StatusOK || calls.Load() != 1 || !strings.Contains(rec.Body.String(), "func main() {}") {
		t.Fatalf("status=%d calls=%d body=%s", rec.Code, calls.Load(), rec.Body.String())
	}
}

func TestProductionFormatProxyRejectsUnsafeRequestsAndResponses(t *testing.T) {
	t.Run("method, request size, and upstream error", func(t *testing.T) {
		var calls atomic.Int32
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			http.Error(w, "failure", http.StatusInternalServerError)
		}))
		defer upstream.Close()
		proxy := mustPlaygroundProxy(t, upstream.URL)

		rec := httptest.NewRecorder()
		proxy.format(rec, httptest.NewRequest(http.MethodGet, "/_/fmt", nil))
		if rec.Code != http.StatusMethodNotAllowed || calls.Load() != 0 {
			t.Fatalf("GET status=%d calls=%d", rec.Code, calls.Load())
		}

		rec = httptest.NewRecorder()
		proxy.format(rec, formRequest(http.MethodPost, "/_/fmt", url.Values{"body": {strings.Repeat("x", playgroundRequestLimit)}}))
		if rec.Code != http.StatusRequestEntityTooLarge || calls.Load() != 0 {
			t.Fatalf("large status=%d calls=%d", rec.Code, calls.Load())
		}

		rec = httptest.NewRecorder()
		proxy.format(rec, formRequest(http.MethodPost, "/_/fmt", url.Values{"body": {"package main"}}))
		if rec.Code != http.StatusBadGateway || calls.Load() != 1 {
			t.Fatalf("upstream status=%d calls=%d", rec.Code, calls.Load())
		}
	})

	t.Run("response too large", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, strings.Repeat("x", playgroundResponseLimit+1))
		}))
		defer upstream.Close()
		proxy := mustPlaygroundProxy(t, upstream.URL)
		rec := httptest.NewRecorder()
		proxy.format(rec, formRequest(http.MethodPost, "/_/fmt", url.Values{"body": {"package main"}}))
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("redirect rejected", func(t *testing.T) {
		var escaped atomic.Int32
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			escaped.Add(1)
			_, _ = io.WriteString(w, `{}`)
		}))
		defer target.Close()
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL, http.StatusFound)
		}))
		defer upstream.Close()
		proxy := mustPlaygroundProxy(t, upstream.URL)
		rec := httptest.NewRecorder()
		proxy.format(rec, formRequest(http.MethodPost, "/_/fmt", url.Values{"body": {"package main"}}))
		if rec.Code != http.StatusBadGateway || escaped.Load() != 0 {
			t.Fatalf("status=%d escaped=%d", rec.Code, escaped.Load())
		}
	})
}

func productionTestHandler(t *testing.T, upstreamURL string) http.Handler {
	t.Helper()
	proxy := mustPlaygroundProxy(t, upstreamURL)
	handler, err := newProductionHandler(website.TourOnly(), "zh-CN", proxy)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func mustPlaygroundProxy(t *testing.T, baseURL string) *playgroundProxy {
	t.Helper()
	proxy, err := newPlaygroundProxy(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	return proxy
}

func formRequest(method, target string, form url.Values) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}
