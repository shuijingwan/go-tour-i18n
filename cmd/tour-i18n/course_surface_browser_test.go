package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
	"golang.org/x/net/websocket"
)

func TestFrenchPreviewEditorStatesAndLessonPreGeometryInBrowser(t *testing.T) {
	chrome := browserTestChrome(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := i18n.ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := i18n.HydrateCatalogSources(catalog, current); err != nil {
		t.Fatal(err)
	}
	projection, err := i18n.BuildLocaleProjection(root, catalog, "fr-FR", filepath.Join(t.TempDir(), "projection"))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := tour.NewPreviewHandler(os.DirFS(projection.ContentDir), "fr-FR")
	if err != nil {
		t.Fatal(err)
	}
	server := newIPv4TestServer(t, handler)

	for _, route := range []string{"/tour/basics/11", "/tour/moretypes/1"} {
		t.Run("mobile"+route, func(t *testing.T) {
			result := courseSurfaceInBrowser(t, chrome, server.URL+route, 375, false)
			if result.Status != "PASS" {
				t.Fatalf("mobile course surface failed: %s", result.Status)
			}
		})
	}
	t.Run("desktop editor interaction", func(t *testing.T) {
		result := courseSurfaceInBrowser(t, chrome, server.URL+"/tour/basics/11", 1280, true)
		if result.Status != "PASS" {
			t.Fatalf("desktop editor surface failed: %s", result.Status)
		}
	})
}

type courseSurfaceResult struct {
	Status string `json:"status"`
}

func courseSurfaceInBrowser(t *testing.T, chrome, target string, viewport int, desktop bool) courseSurfaceResult {
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
		"--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage", "--disable-breakpad", "--disable-crash-reporter", "--disable-background-networking", "--disable-default-apps", "--disable-extensions", "--no-first-run", "--noerrdialogs",
		"--remote-debugging-address=127.0.0.1", "--remote-debugging-port="+fmt.Sprint(port), "--remote-allow-origins=*",
		"--user-data-dir="+filepath.Join(t.TempDir(), "chrome-profile"),
		"--host-resolver-rules=MAP assets-go-dev.shuijingwanwq.com ~NOTFOUND, MAP fonts.googleapis.com ~NOTFOUND", "about:blank")
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
		request, requestErr := http.NewRequest(http.MethodPut, debugURL, nil)
		if requestErr != nil {
			t.Fatal(requestErr)
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
	cdpCall(t, ws, 1, "Emulation.setDeviceMetricsOverride", map[string]any{"width": viewport, "height": 800, "deviceScaleFactor": 1, "mobile": !desktop})
	ready := false
	callID := 2
	for attempt := 0; attempt < 3 && !ready; attempt++ {
		cdpCall(t, ws, callID, "Page.navigate", map[string]any{"url": target})
		callID++
		// Use short, separate evaluations so the document parser can continue
		// loading the old parser-blocking Angular bundle between checks.
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
			probe := cdpCall(t, ws, callID, "Runtime.evaluate", map[string]any{"expression": "typeof window.angular", "returnByValue": true})
			callID++
			if strings.Contains(string(probe), `"value":"object"`) {
				ready = true
				break
			}
		}
	}
	if !ready {
		t.Fatal("Angular did not load after three navigations")
	}
	result := cdpCall(t, ws, callID, "Runtime.evaluate", map[string]any{
		"expression": `(async () => {
  const wait = async selector => { for (let n = 0; n < 600; n++) { const node = document.querySelector(selector); if (node) return node; await new Promise(resolve => setTimeout(resolve, 20)); } const lesson = await fetch('/tour/lesson/'); throw new Error('missing ' + selector + ' at ' + location.href + '; angular=' + typeof window.angular + '; lesson=' + lesson.status + '; ' + document.body.textContent.trim().slice(0, 160)); };
  try {
    const syntax = await wait('.syntax-checkbox'), imports = await wait('.imports-checkbox');
    await wait('.slide-content pre');
    const pre = [...document.querySelectorAll('.slide-content pre')].sort((a, b) => Math.max(...b.textContent.split('\n').map(line => line.length)) - Math.max(...a.textContent.split('\n').map(line => line.length)))[0];
    await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
    const assert = (condition, message) => { if (!condition) throw new Error(message); };
    assert(document.documentElement.scrollWidth <= document.documentElement.clientWidth, 'document has horizontal overflow');
    assert(document.querySelector('#left-side').scrollWidth <= document.querySelector('#left-side').clientWidth, 'lesson container has horizontal overflow');
    const original = pre.textContent;
    if (!` + fmt.Sprint(desktop) + `) {
      const style = getComputedStyle(pre), rect = pre.getBoundingClientRect(), content = pre.textContent;
      const logicalLines = content.split('\n').length;
      const lineHeight = parseFloat(style.lineHeight);
      const verticalExtras = parseFloat(style.paddingTop) + parseFloat(style.paddingBottom) + parseFloat(style.borderTopWidth) + parseFloat(style.borderBottomWidth);
      const unwrappedHeight = logicalLines * lineHeight + verticalExtras;
      assert(rect.width <= pre.parentElement.getBoundingClientRect().width + 0.5, 'lesson pre exceeds available content width');
      assert(pre.scrollWidth <= pre.clientWidth, 'lesson pre has horizontal scrolling');
      assert(style.whiteSpace === 'pre-wrap', 'lesson pre does not preserve whitespace while wrapping');
      assert(pre.scrollHeight > unwrappedHeight + lineHeight * 0.5, 'long pre content did not visually wrap');
      await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
      assert(pre.textContent === original, 'lesson pre textContent changed during visual wrapping');
    }
    assert(syntax.innerText.trim().endsWith('Désactivé'), 'French syntax state is ' + syntax.innerText.trim());
    assert(imports.innerText.trim().endsWith('Désactivé'), 'French imports state is ' + imports.innerText.trim());
    if (` + fmt.Sprint(desktop) + `) {
      assert(document.querySelector('.CodeMirror') && document.querySelector('#run') && document.querySelector('.next-page'), 'desktop editor/navigation controls are missing');
      syntax.click(); await new Promise(resolve => setTimeout(resolve, 50));
      assert(syntax.classList.contains('active') && syntax.innerText.trim().endsWith('Activé'), 'syntax toggle interaction regressed');
      imports.click(); await new Promise(resolve => setTimeout(resolve, 50));
      assert(imports.classList.contains('active') && imports.innerText.trim().endsWith('Activé'), 'imports toggle interaction regressed');
    }
    return {status:'PASS'};
  } catch (error) { return {status:'FAIL: ' + error.message}; }
})()`,
		"awaitPromise": true, "returnByValue": true,
	})
	var envelope struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		t.Fatal(err)
	}
	callID++
	cdpCall(t, ws, callID, "Browser.close", map[string]any{})
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	closed = true
	var value courseSurfaceResult
	if err := json.Unmarshal(envelope.Result.Value, &value); err != nil {
		t.Fatalf("decode browser result %s: %v", envelope.Result.Value, err)
	}
	return value
}
