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
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestTourHeaderTitlesFitCommonMobileViewports(t *testing.T) {
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

	for _, viewport := range []int{320, 375, 414} {
		for _, title := range []string{"A Tour of Go", "Eine Tour durch Go", "Go 语言之旅", "Go のツアー"} {
			t.Run(fmt.Sprintf("%d/%s", viewport, title), func(t *testing.T) {
				document := fmt.Sprintf(`<!doctype html><html data-theme="auto"><head><meta name="viewport" content="width=device-width, initial-scale=1"><style>%s</style></head><body>
<div class="bar top-bar"><div class="left"><a href="/"><img class="gopherlogo" alt=""></a><a class="logo" href="/tour/list">%s</a></div><div class="right"><button class="header-toggleTheme"><img data-value="auto" class="go-Icon go-Icon--inverted" height="24" width="24" alt=""></button><span class="nav"><svg viewBox="0 0 24 24" height="100%%" width="100%%"></svg></span><span class="nav"><svg viewBox="0 0 24 24" height="100%%" width="100%%"></svg></span></div></div>
<div id="editor-container"></div>
<script>
function assert(condition, message) { if (!condition) throw new Error(message); }
try {
  var title = document.querySelector('.top-bar .logo');
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
  document.body.setAttribute('data-tour-header-test', 'PASS');
} catch (error) { document.body.setAttribute('data-tour-header-test', 'FAIL: ' + error.message); }
</script></body></html>`, css, html.EscapeString(title), viewport, title)
				path := filepath.Join(t.TempDir(), "tour-header-test.html")
				if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
					t.Fatal(err)
				}
				result := evaluateHeaderAtViewport(t, chrome, "file://"+path, viewport)
				if result.Viewport != viewport {
					t.Fatalf("CSS viewport width = %d, want %d", result.Viewport, viewport)
				}
				t.Logf("window.innerWidth = %d", result.Viewport)
				if result.Status != "PASS" {
					t.Fatalf("tour header browser test failed: %s", result.Status)
				}
			})
		}
	}
}

type headerViewportResult struct {
	Status   string `json:"status"`
	Viewport int    `json:"viewport"`
}

func evaluateHeaderAtViewport(t *testing.T, chrome, target string, viewport int) headerViewportResult {
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
	headerCDPCall(t, ws, 1, "Emulation.setDeviceMetricsOverride", map[string]any{"width": viewport, "height": 800, "deviceScaleFactor": 1, "mobile": true})
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
