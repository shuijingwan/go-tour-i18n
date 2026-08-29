package tour

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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
		{"/", http.StatusOK, "Go 语言之旅多语言翻译项目"},
		{"/tour/", http.StatusOK, "Go 语言之旅"},
		{"/tour/list", http.StatusOK, "Go 语言之旅"},
		{"/tour/lesson/welcome", http.StatusOK, `"Title":"Welcome!"`},
		{"/tour/static/css/app.css", http.StatusOK, "body"},
		{"/images/go-logo-white.svg", http.StatusOK, "<svg"},
	}

	home := httptest.NewRecorder()
	handler.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/", nil))
	for _, want := range []string{"永夜维护 · 非官方社区多语言翻译项目", "项目如何工作", "继续在 go.dev 学习", "本站可能展示广告", "GitHub 项目源码", "蜀ICP备13001590号-1", "开发环境", "golang/website@" + FrozenUpstreamCommit[:8]} {
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
	for _, want := range []string{`window.transport = HTTPTransport();`, `window.socketAddr = "";`, `window.playgroundBaseURL = "https://play.go-dev.shuijingwanwq.com:8443";`, `$.ajax(playgroundURL('/_/compile?backend='`, `$.ajax(playgroundURL('/_/fmt?backend='`, `$http.post(playgroundURL('/_/fmt')`} {
		if !strings.Contains(script, want) {
			t.Errorf("production script does not contain %q", want)
		}
	}
	for _, forbidden := range []string{`window.transport = SocketTransport();`, `window.socketAddr = "ws://`, `window.socketAddr = "wss://`} {
		if strings.Contains(script, forbidden) {
			t.Errorf("production script contains %q", forbidden)
		}
	}
	if strings.Contains(script, `window.playgroundBaseURL = "https://go.dev`) {
		t.Fatal("production script still injects the server-side Playground URL")
	}
}

func TestHTTPTransportRuntimeLocalizationInBrowser(t *testing.T) {
	if os.Getenv("GO_TOUR_RUN_BROWSER_TESTS") != "1" {
		t.Skip("set GO_TOUR_RUN_BROWSER_TESTS=1 to run the Chrome integration test")
	}
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Skip("google-chrome is not installed")
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/compile" || r.ParseForm() != nil {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		body := r.Form.Get("body")
		if strings.Contains(body, "runtime-communication-error") {
			http.Error(w, "controlled upstream failure", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(body, "runtime-build-failure"):
			_, _ = io.WriteString(w, `{"Errors":"controlled build failure"}`)
		case strings.Contains(body, "runtime-vet-failure"):
			_, _ = io.WriteString(w, `{"VetErrors":"controlled vet failure","Events":[]}`)
		case strings.Contains(body, "runtime-test-one"):
			_, _ = io.WriteString(w, `{"IsTest":true,"TestsFailed":1,"Events":[]}`)
		case strings.Contains(body, "runtime-test-many"):
			_, _ = io.WriteString(w, `{"IsTest":true,"TestsFailed":2,"Events":[]}`)
		case strings.Contains(body, "runtime-test-pass"):
			_, _ = io.WriteString(w, `{"IsTest":true,"TestsFailed":0,"Events":[]}`)
		default:
			http.Error(w, "unexpected program", http.StatusBadRequest)
		}
	}))
	defer upstream.Close()
	proxy := mustPlaygroundProxy(t, upstream.URL)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	baseURL := "http://" + listener.Addr().String()
	handler, _, err := newTourHandlerWithPlaygroundBase(website.TourOnly(), "zh-CN", proxy, baseURL+"/_", false, false)
	if err != nil {
		t.Fatal(err)
	}
	const route = "/tour/basics/1"
	instrumented := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != route {
			handler.ServeHTTP(w, r)
			return
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, r)
		for key, values := range recorder.Header() {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(recorder.Code)
		script := `<script>
(function waitForRuntime() {
  if (document.documentElement.getAttribute('data-tour-rendered-route') !== '` + route + `' || !document.querySelector('#run') || !document.querySelector('.CodeMirror')) {
    setTimeout(waitForRuntime, 20); return;
  }
  var cases = [['runtime-build-failure', 'Go 构建失败。'], ['runtime-vet-failure', 'Go vet 检查失败。'], ['runtime-communication-error', '与远程服务器通信时出错。'], ['runtime-test-one', '1 个测试失败。'], ['runtime-test-many', '2 个测试失败。'], ['runtime-test-pass', '所有测试均已通过。']];
  var index = 0;
  function runNext() {
    if (index === cases.length) { document.documentElement.setAttribute('data-tour-runtime-i18n', 'PASS'); return; }
    var item = cases[index++];
    document.querySelector('.CodeMirror').CodeMirror.setValue('package main\n// ' + item[0] + '\nfunc main() {}\n');
    setTimeout(function() {
      document.querySelector('#run').click();
      (function waitForOutput() {
        var output = document.querySelector('.output.active');
        if (output && output.textContent.indexOf(item[1]) !== -1) { runNext(); return; }
        setTimeout(waitForOutput, 20);
      }());
    }, 20);
  }
  runNext();
}());
</script>`
		_, _ = w.Write(bytes.Replace(recorder.Body.Bytes(), []byte("</body>"), []byte(script+"</body>"), 1))
	})
	server := httptest.NewUnstartedServer(instrumented)
	server.Listener = listener
	server.Start()
	defer server.Close()
	command := exec.Command(chrome, "--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage", "--disable-breakpad", "--disable-crash-reporter", "--disable-background-networking", "--disable-default-apps", "--disable-extensions", "--no-first-run", "--noerrdialogs", "--user-data-dir="+filepath.Join(t.TempDir(), "chrome-profile"), "--run-all-compositor-stages-before-draw", "--virtual-time-budget=12000", "--dump-dom", server.URL+route)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("google-chrome: %v\n%s", err, output)
	}
	if !bytes.Contains(output, []byte(`data-tour-runtime-i18n="PASS"`)) {
		t.Fatalf("HTTPTransport runtime localization failed:\n%s", output)
	}
}

func TestProductionHandlerAdHTMLConfiguration(t *testing.T) {
	original := adHTML
	t.Cleanup(func() { adHTML = original })
	const marker = `<script data-test="ad"></script>`

	for _, test := range []struct {
		name  string
		value string
		count int
	}{
		{name: "disabled", count: 0},
		{name: "enabled", value: marker, count: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("TOUR_AD_HTML", test.value)
			handler := productionTestHandler(t, "http://127.0.0.1:1")
			for _, requestPath := range []string{"/", "/tour/list", "/tour/welcome/1"} {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, requestPath, nil))
				if rec.Code != http.StatusOK {
					t.Fatalf("GET %s: status %d", requestPath, rec.Code)
				}
				if got := strings.Count(rec.Body.String(), marker); got != test.count {
					t.Errorf("GET %s contains ad HTML %d times, want %d", requestPath, got, test.count)
				}
			}
		})
	}
}

func TestPlaygroundURLConfiguration(t *testing.T) {
	if got := playgroundURLForTest("", "/_/compile?backend="); got != "/_/compile?backend=" {
		t.Fatalf("empty Playground base URL = %q", got)
	}
	if got := playgroundURLForTest(productionPlaygroundBaseURL, "/_/compile?backend=gc"); got != productionPlaygroundBaseURL+"/compile?backend=gc" {
		t.Fatalf("production compile URL = %q", got)
	}
	if got := playgroundURLForTest(productionPlaygroundBaseURL, "/_/fmt"); got != productionPlaygroundBaseURL+"/fmt" {
		t.Fatalf("production format URL = %q", got)
	}
	if got := playgroundURLForTest(productionPlaygroundBaseURL, "/_/compile?backend=https://attacker.invalid"); !strings.HasPrefix(got, productionPlaygroundBaseURL+"/compile?") {
		t.Fatalf("backend changed Playground host: %q", got)
	}
}

func playgroundURLForTest(base, path string) string {
	if base == "" {
		return path
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimPrefix(path, "/_/")
}

func TestProductionHandlerServesLocaleSpecificSEODocuments(t *testing.T) {
	for _, test := range []struct {
		locale, origin, otherOrigin string
	}{
		{"zh-CN", "https://go-dev.shuijingwanwq.com", "https://ja-go-dev.shuijingwanwq.com"},
		{"ja-JP", "https://ja-go-dev.shuijingwanwq.com", "https://go-dev.shuijingwanwq.com"},
	} {
		t.Run(test.locale, func(t *testing.T) {
			handler := productionTestHandlerLocale(t, "http://127.0.0.1:1", test.locale)

			robots := httptest.NewRecorder()
			robotsRequest := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
			robotsRequest.Host = "untrusted-host.invalid"
			handler.ServeHTTP(robots, robotsRequest)
			if robots.Code != http.StatusOK || robots.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Fatalf("robots response = %d %q", robots.Code, robots.Header().Get("Content-Type"))
			}
			if strings.Contains(robots.Body.String(), "<!doctype html") || !strings.Contains(robots.Body.String(), "Sitemap: "+test.origin+"/sitemap.xml") || strings.Contains(robots.Body.String(), test.otherOrigin) {
				t.Fatalf("invalid robots.txt: %q", robots.Body.String())
			}

			sitemap := httptest.NewRecorder()
			sitemapRequest := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
			sitemapRequest.Host = "untrusted-host.invalid"
			handler.ServeHTTP(sitemap, sitemapRequest)
			if sitemap.Code != http.StatusOK || sitemap.Header().Get("Content-Type") != "application/xml; charset=utf-8" {
				t.Fatalf("sitemap response = %d %q", sitemap.Code, sitemap.Header().Get("Content-Type"))
			}
			body := sitemap.Body.String()
			if !strings.HasPrefix(body, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") || !strings.Contains(body, "<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">") || !strings.HasSuffix(strings.TrimSpace(body), "</urlset>") {
				t.Fatal("sitemap is not well-formed XML")
			}
			locs := strings.Split(body, "<loc>")[1:]
			seen := map[string]bool{}
			for _, part := range locs {
				u := strings.SplitN(part, "</loc>", 2)[0]
				if seen[u] {
					t.Fatalf("duplicate sitemap URL %q", u)
				}
				seen[u] = true
				if !strings.HasPrefix(u, test.origin+"/") || strings.Contains(u, test.otherOrigin) || strings.Contains(u, "go-tour.shuijingwanwq.com") {
					t.Fatalf("unexpected sitemap host in %q", u)
				}
			}
			if len(locs) != 105 {
				t.Fatalf("sitemap URLs = %d, want 105 (homepage + list + 103 pages)", len(locs))
			}
			if !seen[test.origin+"/"] || !seen[test.origin+"/tour/list"] {
				t.Fatalf("sitemap is missing homepage or tour list: %v", seen)
			}
			if first, second := strings.SplitN(locs[0], "</loc>", 2)[0], strings.SplitN(locs[1], "</loc>", 2)[0]; first != test.origin+"/" || second != test.origin+"/tour/list" {
				t.Fatalf("sitemap starts with %q, %q", first, second)
			}
		})
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
	if requestID := rec.Header().Get("X-Request-ID"); requestID == "" {
		t.Fatal("compile response is missing X-Request-ID")
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
	if requestID := rec.Header().Get("X-Request-ID"); requestID == "" {
		t.Fatal("format response is missing X-Request-ID")
	}
}

func TestProductionPlaygroundFailureLoggingIsMinimalAndSilentOnSuccess(t *testing.T) {
	var logs bytes.Buffer
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"Events":[]}`)
	}))
	defer upstream.Close()

	proxy := mustPlaygroundProxy(t, upstream.URL)
	proxy.logger = log.New(&logs, "", 0)
	form := url.Values{"version": {"2"}, "body": {"package main\nfunc main(){}"}}
	rec := httptest.NewRecorder()
	proxy.compile(rec, formRequest(http.MethodPost, "/_/compile", form))
	if rec.Code != http.StatusOK || logs.Len() != 0 {
		t.Fatalf("successful compile status=%d logs=%q", rec.Code, logs.String())
	}

	proxy.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, io.ErrUnexpectedEOF
	})
	rec = httptest.NewRecorder()
	proxy.compile(rec, formRequest(http.MethodPost, "/_/compile", form))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("failed compile status=%d body=%s", rec.Code, rec.Body.String())
	}
	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" || !strings.Contains(logs.String(), "request_id="+requestID) || !strings.Contains(logs.String(), "operation=compile") || !strings.Contains(logs.String(), "unexpected EOF") {
		t.Fatalf("failure log=%q request_id=%q", logs.String(), requestID)
	}
	if strings.Contains(logs.String(), "package main") {
		t.Fatalf("failure log contains source code: %q", logs.String())
	}
	if got := strings.Count(logs.String(), "playground "); got != 1 {
		t.Fatalf("failure log entries=%d logs=%q", got, logs.String())
	}
}

func TestProductionPlaygroundInvalidJSONLogsStatusAndKeepsResponse(t *testing.T) {
	var logs bytes.Buffer
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, "not json")
	}))
	defer upstream.Close()

	proxy := mustPlaygroundProxy(t, upstream.URL)
	proxy.logger = log.New(&logs, "", 0)
	rec := httptest.NewRecorder()
	proxy.compile(rec, formRequest(http.MethodPost, "/_/compile", url.Values{"version": {"2"}, "body": {"package main"}}))
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "invalid Playground compile response") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" || !strings.Contains(logs.String(), "request_id="+requestID) || !strings.Contains(logs.String(), "upstream_status=200") || !strings.Contains(logs.String(), "invalid Playground compile response") {
		t.Fatalf("invalid JSON log=%q request_id=%q", logs.String(), requestID)
	}
}

func TestProductionFormatFailureLogsRequestIDAndStatus(t *testing.T) {
	var logs bytes.Buffer
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary failure", http.StatusServiceUnavailable)
	}))
	defer upstream.Close()

	proxy := mustPlaygroundProxy(t, upstream.URL)
	proxy.logger = log.New(&logs, "", 0)
	rec := httptest.NewRecorder()
	proxy.format(rec, formRequest(http.MethodPost, "/_/fmt", url.Values{"body": {"package main"}}))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" || !strings.Contains(logs.String(), "request_id="+requestID) || !strings.Contains(logs.String(), "operation=format") || !strings.Contains(logs.String(), "upstream_status=503") {
		t.Fatalf("format failure log=%q request_id=%q", logs.String(), requestID)
	}
	if got := strings.Count(logs.String(), "playground "); got != 1 {
		t.Fatalf("failure log entries=%d logs=%q", got, logs.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
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
	return productionTestHandlerLocale(t, upstreamURL, "zh-CN")
}

func productionTestHandlerLocale(t *testing.T, upstreamURL, locale string) http.Handler {
	t.Helper()
	proxy := mustPlaygroundProxy(t, upstreamURL)
	handler, err := newProductionHandler(website.TourOnly(), locale, proxy)
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
