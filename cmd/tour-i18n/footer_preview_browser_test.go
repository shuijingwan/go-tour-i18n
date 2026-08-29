package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
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

func TestDEPreviewListFooterGeometryInBrowser(t *testing.T) {
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
	projection, err := i18n.BuildLocaleProjection(root, catalog, "de-DE", filepath.Join(t.TempDir(), "projection"))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := tour.NewPreviewHandler(os.DirFS(projection.ContentDir), "de-DE")
	if err != nil {
		t.Fatal(err)
	}
	instrumented := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/tour/list" {
			handler.ServeHTTP(w, request)
			return
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		for key, values := range recorder.Header() {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(recorder.Code)
		script := `<script>
(function inspectFooter() {
  var footer = document.querySelector('.site-footer');
  var list = document.querySelector('.list-wrapper');
  if (!footer || !list) { setTimeout(inspectFooter, 20); return; }
  footer.scrollIntoView({block: 'end'});
  function inspect(node) {
    var cs = getComputedStyle(node), r = node.getBoundingClientRect();
    return {tag: node.tagName, className: node.className, rect: {left:r.left,right:r.right,top:r.top,bottom:r.bottom,width:r.width,height:r.height}, clientWidth:node.clientWidth, scrollWidth:node.scrollWidth, offsetLeft:node.offsetLeft, scrollLeft:node.scrollLeft, overflow:cs.overflow, overflowX:cs.overflowX, width:cs.width, maxWidth:cs.maxWidth, position:cs.position, left:cs.left, right:cs.right, transform:cs.transform, whiteSpace:cs.whiteSpace, textAlign:cs.textAlign};
  }
  var chain = [], node = footer;
  while (node) { chain.push(inspect(node)); if (node === document.documentElement) break; node = node.parentElement; }
  var range = document.createRange(); range.selectNodeContents(footer.firstChild);
  var text = range.getBoundingClientRect();
  var data = {text: footer.textContent.trim(), chain: chain, firstText: {left:text.left,right:text.right,top:text.top,bottom:text.bottom,width:text.width,height:text.height}, document: {scrollWidth:document.documentElement.scrollWidth,clientWidth:document.documentElement.clientWidth,scrollLeft:document.documentElement.scrollLeft}, body: {scrollWidth:document.body.scrollWidth,clientWidth:document.body.clientWidth,scrollLeft:document.body.scrollLeft}};
  document.documentElement.setAttribute('data-footer-geometry', btoa(unescape(encodeURIComponent(JSON.stringify(data)))));
}());
</script>`
		_, _ = w.Write(bytes.Replace(recorder.Body.Bytes(), []byte("</body>"), []byte(script+"</body>"), 1))
	})
	server := newIPv4TestServer(t, instrumented)

	for _, viewport := range []int{320, 375} {
		t.Run(fmt.Sprintf("%d", viewport), func(t *testing.T) {
			var geometry map[string]any
			if err := json.Unmarshal(chromeFooterGeometry(t, chrome, server.URL+"/tour/list", viewport), &geometry); err != nil {
				t.Fatal(err)
			}
			assertPreviewFooterGeometry(t, geometry, viewport)
		})
	}
}

func assertPreviewFooterGeometry(t *testing.T, geometry map[string]any, viewport int) {
	t.Helper()
	text, _ := geometry["text"].(string)
	for _, want := range []string{"Yongye", "Inoffizielles mehrsprachiges Community-Übersetzungsprojekt", "Startseite", "GitHub", "Entwicklungsprotokoll", "© 2026"} {
		if !strings.Contains(text, want) {
			t.Errorf("footer text = %q, missing %q", text, want)
		}
	}
	actualViewport := geometry["viewport"].(map[string]any)
	if got := int(actualViewport["width"].(float64)); got != viewport {
		t.Errorf("CSS viewport width = %d, want %d", got, viewport)
	}
	footer := geometry["chain"].([]any)[0].(map[string]any)
	rect := footer["rect"].(map[string]any)
	if got := footer["scrollWidth"].(float64); got != footer["clientWidth"].(float64) {
		t.Errorf("footer scrollWidth/clientWidth = %v/%v", got, footer["clientWidth"])
	}
	if rect["left"].(float64) < 0 || rect["right"].(float64) > float64(viewport) || rect["top"].(float64) < 0 || rect["bottom"].(float64) > 801 {
		t.Errorf("footer rect = %+v, viewport=%d", rect, viewport)
	}
	first := geometry["firstText"].(map[string]any)
	if first["left"].(float64) < 0 || first["right"].(float64) > float64(viewport) || first["top"].(float64) < 0 || first["bottom"].(float64) > 801 {
		t.Errorf("footer first line container = %+v, viewport=%d", first, viewport)
	}
	for _, property := range []string{"transform", "overflowX", "position", "scrollLeft"} {
		if property == "transform" && footer[property] != "none" {
			t.Errorf("footer %s = %v, want none", property, footer[property])
		}
		if property == "scrollLeft" && footer[property].(float64) != 0 {
			t.Errorf("footer %s = %v, want 0", property, footer[property])
		}
	}
	container := geometry["container"].(map[string]any)["rect"].(map[string]any)
	if container["bottom"].(float64) > rect["top"].(float64)+0.1 {
		t.Errorf("container bottom %v overlaps footer top %v", container["bottom"], rect["top"])
	}
	stack := geometry["stack"].([]any)
	if len(stack) < 2 || stack[0].(map[string]any)["className"] != "" || stack[1].(map[string]any)["className"] != "site-footer" {
		t.Errorf("elementFromPoint on footer first line = %v, want footer content", stack)
	}
}

func chromeFooterGeometry(t *testing.T, chrome, target string, viewport int) []byte {
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
		"--host-resolver-rules=MAP assets-go-dev.shuijingwanwq.com ~NOTFOUND, MAP fonts.googleapis.com ~NOTFOUND",
		"about:blank")
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
	cdpCall(t, ws, 1, "Emulation.setDeviceMetricsOverride", map[string]any{"width": viewport, "height": 800, "deviceScaleFactor": 1, "mobile": true})
	cdpCall(t, ws, 2, "Page.navigate", map[string]any{"url": target})
	time.Sleep(500 * time.Millisecond)
	result := cdpCall(t, ws, 3, "Runtime.evaluate", map[string]any{
		"expression": `(async () => {
  for (let n = 0; n < 600; n++) { if (document.querySelector('.site-footer') && document.querySelector('.list-wrapper') && document.querySelectorAll('.lesson').length > 0) break; await new Promise(resolve => setTimeout(resolve, 20)); }
  await new Promise(resolve => setTimeout(resolve, 500));
  const footer = document.querySelector('.site-footer'); if (!footer) throw new Error('footer did not render');
  const container = document.querySelector('.list-wrapper > .container'), list = document.querySelector('.list-wrapper');
  footer.scrollIntoView({block: 'end'}); await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
  const inspect = node => { const cs = getComputedStyle(node), r = node.getBoundingClientRect(); return {tag: node.tagName, className: node.className, rect: {left:r.left,right:r.right,top:r.top,bottom:r.bottom,width:r.width,height:r.height}, clientWidth:node.clientWidth, scrollWidth:node.scrollWidth, offsetLeft:node.offsetLeft, scrollLeft:node.scrollLeft, overflow:cs.overflow, overflowX:cs.overflowX, width:cs.width, maxWidth:cs.maxWidth, position:cs.position, left:cs.left, right:cs.right, transform:cs.transform, whiteSpace:cs.whiteSpace, textAlign:cs.textAlign}; };
  const chain = []; for (let node = footer; node; node = node.parentElement) { chain.push(inspect(node)); if (node === document.documentElement) break; }
  const first = footer.firstElementChild, r = first.getBoundingClientRect();
  const stack = (x, y) => document.elementsFromPoint(x, y).map(node => ({tag:node.tagName, className:node.className}));
  const overflow = [...document.querySelectorAll('*')].map(node => { const r = node.getBoundingClientRect(); return {tag:node.tagName, className:node.className, left:r.left, right:r.right, scrollWidth:node.scrollWidth, clientWidth:node.clientWidth}; }).filter(node => node.right > innerWidth || node.left < 0);
  return {viewport: {width: innerWidth, height: innerHeight}, text: footer.textContent.trim(), chain, container:inspect(container), list:inspect(list), view:inspect(list.parentElement), bodyChildren:[...document.body.children].map(node => ({tag:node.tagName,className:node.className})), order:list.compareDocumentPosition(footer), firstText: {left:r.left,right:r.right,top:r.top,bottom:r.bottom,width:r.width,height:r.height}, stack:stack(r.left + 16, r.top + 10), document: {scrollWidth:document.documentElement.scrollWidth,clientWidth:document.documentElement.clientWidth,scrollLeft:document.documentElement.scrollLeft}, body: {scrollWidth:document.body.scrollWidth,clientWidth:document.body.clientWidth,scrollLeft:document.body.scrollLeft}, overflow};
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
	if len(envelope.Result.Value) == 0 {
		t.Fatalf("DevTools footer evaluation returned no value: %s", result)
	}
	cdpCall(t, ws, 4, "Browser.close", map[string]any{})
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	closed = true
	return envelope.Result.Value
}

func cdpCall(t *testing.T, ws *websocket.Conn, id int, method string, params any) json.RawMessage {
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
