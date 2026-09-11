// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestTourHeaderTitlesAreCenteredOnDesktopAndFitCommonMobileViewports(t *testing.T) {
	if os.Getenv("GO_TOUR_RUN_BROWSER_TESTS") != "1" {
		t.Skip("set GO_TOUR_RUN_BROWSER_TESTS=1 to run the Chrome integration test")
	}
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Skip("google-chrome is not installed")
	}
	css, err := fs.ReadFile(contentTour, "tour/static/css/app.css")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		viewport int
		mobile   bool
	}{
		{name: "desktop", viewport: 1280},
		{name: "mobile-320", viewport: 320, mobile: true},
		{name: "mobile-375", viewport: 375, mobile: true},
		{name: "mobile-414", viewport: 414, mobile: true},
	} {
		for _, title := range []string{"A Tour of Go", "Go Turu", "Eine Tour durch Go", "Go 语言之旅", "Go のツアー"} {
			t.Run(fmt.Sprintf("%s/%s", test.name, title), func(t *testing.T) {
				document := fmt.Sprintf(`<!doctype html><html data-theme="auto"><head><meta name="viewport" content="width=device-width, initial-scale=1"><style>%s</style></head><body>
<div class="bar top-bar"><div class="left"><a href="/"><img class="gopherlogo" alt=""></a><a class="logo" href="/tour/list">%s</a></div><div class="right"><button class="header-toggleTheme"><img data-value="auto" class="go-Icon go-Icon--inverted" height="24" width="24" alt=""></button><span class="nav"><svg viewBox="0 0 24 24" height="100%%" width="100%%"></svg></span><span class="nav"><svg viewBox="0 0 24 24" height="100%%" width="100%%"></svg></span></div></div>
<div id="editor-container"></div>
<script>
function assert(condition, message) { if (!condition) throw new Error(message); }
try {
  var title = document.querySelector('.top-bar .logo');
  var logo = document.querySelector('.top-bar .gopherlogo');
  var header = document.querySelector('.top-bar');
  var editor = document.querySelector('#editor-container');
  var titleBox = title.getBoundingClientRect();
  var headerBox = header.getBoundingClientRect();
  assert(window.innerWidth === %d, 'CSS viewport width is ' + window.innerWidth);
  assert(title.textContent === %q, 'title text changed');
  assert(titleBox.left >= 0 && titleBox.right <= window.innerWidth, 'title is clipped');
  assert(header.scrollWidth <= header.clientWidth, 'header has horizontal overflow');
  assert(document.documentElement.scrollWidth <= document.documentElement.clientWidth, 'page has horizontal overflow');
  assert(editor.getBoundingClientRect().top >= headerBox.bottom, 'course content overlaps header');
  if (!%t) {
    var center = (headerBox.top + headerBox.bottom) / 2;
    var titleCenter = (titleBox.top + titleBox.bottom) / 2;
    var logoBox = logo.getBoundingClientRect();
    var logoCenter = (logoBox.top + logoBox.bottom) / 2;
    assert(headerBox.height >= 48, 'desktop header is shorter than 48px');
    assert(Math.abs(titleCenter - center) <= 1, 'title is not vertically centered');
    assert(Math.abs(logoCenter - center) <= 1, 'logo is not vertically centered');
  }
  document.body.setAttribute('data-tour-header-test', 'PASS');
} catch (error) { document.body.setAttribute('data-tour-header-test', 'FAIL: ' + error.message); }
</script></body></html>`, css, html.EscapeString(title), test.viewport, title, test.mobile)
				path := filepath.Join(t.TempDir(), "tour-header-test.html")
				if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
					t.Fatal(err)
				}
				result := evaluateHeaderAtViewport(t, chrome, "file://"+path, test.viewport, test.mobile)
				if result.Viewport != test.viewport {
					t.Fatalf("CSS viewport width = %d, want %d", result.Viewport, test.viewport)
				}
				t.Logf("window.innerWidth = %d", result.Viewport)
				if result.Status != "PASS" {
					t.Fatalf("tour header browser test failed: %s", result.Status)
				}
			})
		}
	}
}

func TestHomepageLanguageListFitsMobileViewport(t *testing.T) {
	if os.Getenv("GO_TOUR_RUN_BROWSER_TESTS") != "1" {
		t.Skip("set GO_TOUR_RUN_BROWSER_TESTS=1 to run the Chrome integration test")
	}
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Skip("google-chrome is not installed")
	}
	css, err := fs.ReadFile(contentTour, "tour/static/css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	document := fmt.Sprintf(`<!doctype html><html data-theme="auto"><head><meta name="viewport" content="width=device-width, initial-scale=1"><style>%s</style></head><body class="site-home">
<main class="site-main"><section class="site-section"><h2>Versions linguistiques</h2><ul class="site-language-list">
<li><a href="https://go-dev.shuijingwanwq.com/">Simplified Chinese — 简体中文</a></li>
<li><a href="https://go.dev/tour/">English</a></li>
<li><span aria-current="page">French — Français</span></li>
<li><a href="https://de-go-dev.shuijingwanwq.com/">German — Deutsch</a></li>
<li><a href="https://ja-go-dev.shuijingwanwq.com/">Japanese — 日本語</a></li>
<li><a href="https://ko-go-dev.shuijingwanwq.com/">Korean — 한국어</a></li>
</ul></section></main>
<script>
function assert(condition, message) { if (!condition) throw new Error(message); }
try {
  const list = document.querySelector('.site-language-list'), items = [...list.children], current = document.querySelector('[aria-current="page"]');
  assert(innerWidth === 375, 'CSS viewport width is ' + innerWidth);
  assert(items.length === 6, 'language count changed');
  assert(items.every((item, index) => index === 0 || item.getBoundingClientRect().top > items[index - 1].getBoundingClientRect().top), 'languages are not one per line');
  assert(parseInt(getComputedStyle(current).fontWeight, 10) >= 700, 'current language is not emphasized');
  assert(list.scrollWidth <= list.clientWidth, 'language list has horizontal overflow');
  assert(document.documentElement.scrollWidth <= document.documentElement.clientWidth, 'homepage has horizontal overflow');
  document.body.setAttribute('data-tour-header-test', 'PASS');
} catch (error) { document.body.setAttribute('data-tour-header-test', 'FAIL: ' + error.message); }
</script></body></html>`, css)
	path := filepath.Join(t.TempDir(), "homepage-language-list-test.html")
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	result := evaluateHeaderAtViewport(t, chrome, "file://"+path, 375, true)
	if result.Status != "PASS" {
		t.Fatalf("homepage language list browser test failed: %s", result.Status)
	}
}

func TestHomepageTitlesAreGeometricallyCenteredOnDesktop(t *testing.T) {
	if os.Getenv("GO_TOUR_RUN_BROWSER_TESTS") != "1" {
		t.Skip("set GO_TOUR_RUN_BROWSER_TESTS=1 to run the Chrome integration test")
	}
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Skip("google-chrome is not installed")
	}
	css, err := fs.ReadFile(contentTour, "tour/static/css/app.css")
	if err != nil {
		t.Fatal(err)
	}

	for _, title := range []string{"Go Turu Çok Dilli Çeviri Projesi", "Go 语言之旅多语言翻译项目", "Mehrsprachiges Übersetzungsprojekt für A Tour of Go"} {
		t.Run(title, func(t *testing.T) {
			document := fmt.Sprintf(`<!doctype html><html data-theme="auto"><head><meta name="viewport" content="width=device-width, initial-scale=1"><style>%s</style></head><body class="site-home">
<header class="bar top-bar site-header"><a href="/"><img class="site-logo" alt=""></a><a class="logo" href="/">%s</a></header>
<script>
function assert(condition, message) { if (!condition) throw new Error(message); }
try {
  var header = document.querySelector('.site-header'), logo = document.querySelector('.site-logo'), title = document.querySelector('.site-header .logo');
  var headerBox = header.getBoundingClientRect(), logoBox = logo.getBoundingClientRect(), titleBox = title.getBoundingClientRect();
  var center = (headerBox.top + headerBox.bottom) / 2;
  assert(headerBox.height >= 48, 'homepage header is shorter than 48px');
  assert(Math.abs(((logoBox.top + logoBox.bottom) / 2) - center) <= 1, 'homepage logo is not vertically centered');
  assert(Math.abs(((titleBox.top + titleBox.bottom) / 2) - center) <= 1, 'homepage title is not vertically centered');
  assert(document.documentElement.scrollWidth <= document.documentElement.clientWidth, 'homepage has horizontal overflow');
  document.body.setAttribute('data-tour-header-test', 'PASS');
} catch (error) { document.body.setAttribute('data-tour-header-test', 'FAIL: ' + error.message); }
</script></body></html>`, css, html.EscapeString(title))
			path := filepath.Join(t.TempDir(), "homepage-header-test.html")
			if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
				t.Fatal(err)
			}
			result := evaluateHeaderAtViewport(t, chrome, "file://"+path, 1280, false)
			if result.Status != "PASS" {
				t.Fatalf("homepage header browser test failed: %s", result.Status)
			}
		})
	}
}

func TestPlaygroundOutputWrapsWithoutHorizontalOverflowAtMobileAndDesktop(t *testing.T) {
	if os.Getenv("GO_TOUR_RUN_BROWSER_TESTS") != "1" {
		t.Skip("set GO_TOUR_RUN_BROWSER_TESTS=1 to run the Chrome integration test")
	}
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Skip("google-chrome is not installed")
	}
	css, err := fs.ReadFile(contentTour, "tour/static/css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	outputPrefix := "first line\n  preserved spaces\n"
	longToken := strings.Repeat("unbroken-program-output-", 40)
	longOutput := outputPrefix + longToken

	for _, test := range []struct {
		name     string
		viewport int
		mobile   bool
	}{
		{name: "mobile", viewport: 375, mobile: true},
		{name: "desktop", viewport: 1280},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := fmt.Sprintf(`<!doctype html><html data-theme="auto"><head><meta name="viewport" content="width=device-width, initial-scale=1"><style>%s</style></head><body>
<div class="output active"><pre><span class="stdout">%s</span><span class="system">%s</span></pre></div>
<script>
function assert(condition, message) { if (!condition) throw new Error(message); }
try {
  var output = document.querySelector('.output'), pre = output.querySelector('pre'), spans = [...pre.children], style = getComputedStyle(pre);
  assert(spans.length === 2 && spans.every((span) => span.tagName === 'SPAN'), 'output does not use the real PlaygroundOutput span DOM');
  assert(pre.textContent === %q, 'output text changed');
  assert(style.whiteSpace === 'pre-wrap', 'output does not preserve whitespace while wrapping');
  assert(spans.every((span) => getComputedStyle(span).whiteSpace === 'pre-wrap'), 'output spans did not inherit whitespace wrapping');
  assert(spans.every((span) => getComputedStyle(span).overflowWrap === 'anywhere'), 'output spans did not inherit long-token wrapping');
  assert(pre.scrollWidth <= pre.clientWidth + 2, 'output pre has horizontal overflow');
  assert(output.scrollWidth <= output.clientWidth + 2, 'output container has horizontal overflow');
  assert(document.documentElement.scrollWidth <= document.documentElement.clientWidth + 2, 'page has horizontal overflow');
  assert(document.body.scrollWidth <= document.body.clientWidth + 2, 'body has horizontal overflow');
  assert(pre.scrollHeight > parseFloat(style.lineHeight) * 3, 'long output did not wrap');
  document.body.setAttribute('data-tour-header-test', 'PASS');
} catch (error) { document.body.setAttribute('data-tour-header-test', 'FAIL: ' + error.message); }
</script></body></html>`, css, html.EscapeString(outputPrefix), html.EscapeString(longToken), longOutput)
			path := filepath.Join(t.TempDir(), "playground-output-test.html")
			if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
				t.Fatal(err)
			}
			result := evaluateHeaderAtViewport(t, chrome, "file://"+path, test.viewport, test.mobile)
			if result.Status != "PASS" {
				t.Fatalf("playground output browser test failed: %s", result.Status)
			}
		})
	}
}

type headerViewportResult struct {
	Status   string `json:"status"`
	Viewport int    `json:"viewport"`
}

func evaluateHeaderAtViewport(t *testing.T, chrome, target string, viewport int, mobile bool) headerViewportResult {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(chrome,
		"--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage", "--disable-breakpad", "--disable-crash-reporter", "--noerrdialogs",
		"--remote-debugging-address=127.0.0.1", "--remote-debugging-port="+fmt.Sprint(port), "--remote-allow-origins=*",
		"--user-data-dir="+filepath.Join(t.TempDir(), "chrome-profile"), "about:blank")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	closed := false
	t.Cleanup(func() {
		if !closed {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})

	debugURL := fmt.Sprintf("http://127.0.0.1:%d/json/new?about:blank", port)
	var response *http.Response
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		request, err := http.NewRequest(http.MethodPut, debugURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err = http.DefaultClient.Do(request)
		if err == nil {
			break
		}
	}
	if response == nil {
		t.Fatal("Chrome DevTools endpoint did not start")
	}
	defer response.Body.Close()
	var page struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	ws, err := websocket.Dial(page.WebSocketDebuggerURL, "", "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	headerCDPCall(t, ws, 1, "Emulation.setDeviceMetricsOverride", map[string]any{"width": viewport, "height": 800, "deviceScaleFactor": 1, "mobile": mobile})
	headerCDPCall(t, ws, 2, "Page.navigate", map[string]any{"url": target})
	time.Sleep(500 * time.Millisecond)
	result := headerCDPCall(t, ws, 3, "Runtime.evaluate", map[string]any{
		"expression":    "JSON.stringify({status: document.body.getAttribute('data-tour-header-test') || 'FAIL: test did not run', viewport: window.innerWidth})",
		"returnByValue": true,
	})
	var envelope struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		t.Fatal(err)
	}
	headerCDPCall(t, ws, 4, "Browser.close", map[string]any{})
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	closed = true
	var value headerViewportResult
	if err := json.Unmarshal([]byte(envelope.Result.Value), &value); err != nil {
		t.Fatalf("decode header browser result %q: %v", envelope.Result.Value, err)
	}
	return value
}

func headerCDPCall(t *testing.T, ws *websocket.Conn, id int, method string, params any) json.RawMessage {
	t.Helper()
	message, err := json.Marshal(map[string]any{"id": id, "method": method, "params": params})
	if err != nil {
		t.Fatal(err)
	}
	if err := websocket.Message.Send(ws, string(message)); err != nil {
		t.Fatal(err)
	}
	for {
		var response struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		var received string
		if err := websocket.Message.Receive(ws, &received); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(received), &response); err != nil {
			t.Fatal(err)
		}
		if response.ID != id {
			continue
		}
		if len(response.Error) != 0 && string(response.Error) != "null" {
			t.Fatalf("DevTools %s: %s", method, response.Error)
		}
		return response.Result
	}
}
