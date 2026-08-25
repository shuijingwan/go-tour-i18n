// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	website "github.com/shuijingwan/go-tour-i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
	"golang.org/x/net/html"
)

func TestSanitizePrerenderedHTMLRemovesThirdPartyRuntimeDOM(t *testing.T) {
	input := []byte(`<!doctype html><html><head><script id="tour-runtime-head"></script></head><body>
<iframe src="https://ads.invalid"></iframe>
<ins class="banner adsbygoogle filled"></ins>
<div data-google-query-id="runtime">remove me</div>
<div class="CodeMirror"><span>runtime editor</span></div>
<li class="toc-page ng-scope" style="overflow: hidden; height: 0px;">TOC</li>
<textarea ui-codemirror style="display: none;">package main</textarea>
<div class="go-dev-course-ad" data-go-dev-course-ad data-go-dev-course-ad-mounted="true" role="complementary" aria-label="Advertisement"><span>runtime ad child</span></div>
<main>keep me</main></body></html>`)
	got, err := sanitizePrerenderedHTML(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"<iframe",
		"adsbygoogle",
		"data-google-query-id",
		"runtime ad child",
		"data-go-dev-course-ad-mounted",
		`role="complementary"`,
		`aria-label="Advertisement"`,
		`class="CodeMirror`,
		`style="overflow: hidden; height: 0px;"`,
		`style="display: none;"`,
	} {
		if bytes.Contains(bytes.ToLower(got), []byte(strings.ToLower(forbidden))) {
			t.Errorf("sanitized HTML contains %q: %s", forbidden, got)
		}
	}
	for _, want := range []string{
		prerenderRuntimeHeadMarker,
		"data-go-dev-course-ad",
		"keep me",
		"toc-page",
		"ui-codemirror",
		"package main",
	} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("sanitized HTML lost %q: %s", want, got)
		}
	}
	document, err := html.Parse(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	mounts := findElementsWithAttr(document, "data-go-dev-course-ad")
	if len(mounts) != 1 {
		t.Fatalf("sanitized HTML contains %d course ad mounts, want 1: %s", len(mounts), got)
	}
	mount := mounts[0]
	if mount.FirstChild != nil {
		t.Errorf("course ad mount still has runtime children: %s", got)
	}
	if len(mount.Attr) != 2 || attrValue(mount, "class") != "go-dev-course-ad" {
		t.Errorf("course ad mount attributes=%v, want only static template attributes", mount.Attr)
	}
}

func TestPrerenderOutputPath(t *testing.T) {
	got, err := prerenderOutputPath("/bundle/_content", "/tour/basics/1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/bundle/_content/tour/prerender/basics/1.html" {
		t.Fatalf("output path=%q", got)
	}
	for _, route := range []string{"/tour/list", "/tour/a/b/c", "/tour/../x"} {
		if _, err := prerenderOutputPath("/bundle/_content", route); err == nil {
			t.Errorf("accepted invalid route %q", route)
		}
	}
}

func TestEmbedPrerenderedSourcesPreservesCompleteEscapedGoFiles(t *testing.T) {
	route := tour.CourseRoute{
		Path:  "/tour/test/1",
		Files: []string{"package main\n\nfunc main() { println(\"</pre> & complete\") }\n"},
	}
	input := []byte(`<!doctype html><html><body><div id="editor-container"><div class="CodeMirror"></div></div></body></html>`)
	got, err := embedPrerenderedSources(input, route)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("data-tour-prerender-source")) || !bytes.Contains(got, []byte("&lt;/pre&gt; &amp; complete")) {
		t.Fatalf("embedded source was not safely preserved: %s", got)
	}
	document, err := html.Parse(bytes.NewReader(got))
	if err != nil {
		t.Fatal(err)
	}
	sources := findElementsWithAttr(document, "data-tour-prerender-source")
	if len(sources) != 1 || nodeText(sources[0]) != route.Files[0] {
		t.Fatalf("embedded source round trip failed")
	}
}

func TestPrerenderRepresentativeCoursePagesInBrowser(t *testing.T) {
	chrome := browserTestChrome(t)

	for _, locale := range []string{"zh-CN", "ja-JP"} {
		t.Run(locale, func(t *testing.T) {
			source, err := tour.NewPrerenderSource(website.TourOnly(), locale)
			if err != nil {
				t.Fatal(err)
			}
			server := newIPv4TestServer(t, source.Handler)
			outputRoot := t.TempDir()
			var pages [][]byte
			for _, routePath := range []string{"/tour/welcome/2", "/tour/basics/1"} {
				route, ok := courseRoute(source.Routes, routePath)
				if !ok {
					t.Fatalf("route %s not found", routePath)
				}
				profile := filepath.Join(t.TempDir(), "chrome-profile")
				if err := prerenderRouteWithChrome(t.Context(), chrome, server.URL, profile, outputRoot, route); err != nil {
					t.Fatal(err)
				}
				path, err := prerenderOutputPath(outputRoot, route.Path)
				if err != nil {
					t.Fatal(err)
				}
				page, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				pages = append(pages, page)
				for _, want := range []string{route.PageTitle, route.Canonical, `name="description"`, `id="editor-container"`} {
					if !bytes.Contains(page, []byte(want)) {
						t.Errorf("%s output missing %q", route.Path, want)
					}
				}
				if bytes.Contains(page, []byte(`class="CodeMirror`)) {
					t.Errorf("%s contains CodeMirror runtime DOM", route.Path)
				}
				if len(route.Files) > 0 && !bytes.Contains(page, []byte("ui-codemirror")) {
					t.Errorf("%s does not contain ui-codemirror textarea", route.Path)
				}
				if got := bytes.Count(page, []byte("data-tour-prerender-source=")); got != len(route.Files) {
					t.Errorf("%s embedded sources=%d, want %d", route.Path, got, len(route.Files))
				}
			}
			if bytes.Equal(pages[0], pages[1]) {
				t.Fatal("two course URLs produced identical initial HTML")
			}
		})
	}
}

func TestSPACourseNavigationUpdatesMetadataInBrowser(t *testing.T) {
	chrome := browserTestChrome(t)
	var err error
	source, err := tour.NewPrerenderSource(website.TourOnly(), "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	const initialPath = "/tour/basics/2"
	const nextPath = "/tour/basics/3"
	instrumented := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != initialPath {
			source.Handler.ServeHTTP(w, r)
			return
		}
		recorder := httptest.NewRecorder()
		source.Handler.ServeHTTP(recorder, r)
		for key, values := range recorder.Header() {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(recorder.Code)
		script := `<script>
(function waitForInitialPage() {
  if (document.documentElement.getAttribute('data-tour-rendered-route') !== '` + initialPath + `' || !document.querySelector('.next-page')) {
    setTimeout(waitForInitialPage, 20); return;
  }
  var oldTitle = document.title;
  var oldDescription = document.querySelector('meta[name="description"]').content;
  document.querySelector('.next-page').click();
  (function waitForNextPage() {
    var canonical = document.querySelector('link[rel="canonical"]').href;
    var description = document.querySelector('meta[name="description"]').content;
    if (location.pathname === '` + nextPath + `' &&
        document.documentElement.getAttribute('data-tour-rendered-route') === '` + nextPath + `' &&
        canonical === 'https://go-dev.shuijingwanwq.com` + nextPath + `' &&
        document.title !== oldTitle && description && description !== oldDescription) {
      document.documentElement.setAttribute('data-tour-seo-navigation', 'PASS'); return;
    }
    setTimeout(waitForNextPage, 20);
  }());
}());
</script>`
		body := bytes.Replace(recorder.Body.Bytes(), []byte("</body>"), []byte(script+"</body>"), 1)
		_, _ = w.Write(body)
	})
	server := newIPv4TestServer(t, instrumented)
	output := browserDumpDOM(t, chrome, server.URL+initialPath, 7000)
	if !bytes.Contains(output, []byte(`data-tour-seo-navigation="PASS"`)) {
		t.Fatalf("SPA navigation did not update route metadata: %s", output)
	}
}

func TestPublishedBundlePrerenderEndToEndInBrowser(t *testing.T) {
	chrome := browserTestChrome(t)
	root, catalog := publishTestCatalog(t)
	bundle := filepath.Join(t.TempDir(), "production-bundle")
	if err := publishBundle(root, catalog, publishOptions{
		Locale:      "zh-CN",
		Output:      bundle,
		PublishedAt: testPublishedAt,
	}); err != nil {
		t.Fatalf("publish production bundle: %v", err)
	}

	playgroundProxyURL, rootCertificate := newFakePlaygroundProxy(t)
	binaryURL, stopBinary, binaryLogs := startPublishedTourBinary(t, bundle, playgroundProxyURL, rootCertificate)
	t.Cleanup(stopBinary)

	const initialPath = "/tour/basics/1"
	const nextPath = "/tour/basics/2"
	initialResponse, err := http.Get(binaryURL + initialPath)
	if err != nil {
		t.Fatal(err)
	}
	initialHTML, readErr := io.ReadAll(initialResponse.Body)
	closeErr := initialResponse.Body.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read initial production HTML: read=%v close=%v", readErr, closeErr)
	}
	if initialResponse.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status=%d body=%s", initialPath, initialResponse.StatusCode, initialHTML)
	}
	for _, want := range []string{
		`data-tour-rendered-route="` + initialPath + `"`,
		`href="https://go-dev.shuijingwanwq.com` + initialPath + `"`,
		"每个 Go 程序都是由包组成的",
	} {
		if !bytes.Contains(initialHTML, []byte(want)) {
			t.Errorf("initial production HTML missing %q", want)
		}
	}

	productionTarget, err := url.Parse(binaryURL)
	if err != nil {
		t.Fatal(err)
	}
	var browserOrigin string
	reverseProxy := httputil.NewSingleHostReverseProxy(productionTarget)
	reverseProxy.ModifyResponse = func(response *http.Response) error {
		if response.Request.URL.Path != "/tour/script.js" {
			return nil
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			return err
		}
		body = bytes.ReplaceAll(body,
			[]byte(`window.playgroundBaseURL = "https://play.go-dev.shuijingwanwq.com:8443";`),
			[]byte(`window.playgroundBaseURL = "`+browserOrigin+`/_";`),
		)
		response.Body = io.NopCloser(bytes.NewReader(body))
		response.ContentLength = int64(len(body))
		response.Header.Del("Content-Length")
		return nil
	}
	browserHandler := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != initialPath {
			reverseProxy.ServeHTTP(w, request)
			return
		}
		recorder := httptest.NewRecorder()
		reverseProxy.ServeHTTP(recorder, request)
		for key, values := range recorder.Header() {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(recorder.Code)
		script := `<script>
(function waitForBootstrap() {
  if (!window.angular || !angular.element(document.documentElement).injector() ||
      document.documentElement.getAttribute('data-tour-rendered-route') !== '` + initialPath + `' ||
      document.querySelectorAll('#editor-container').length !== 1 || !document.querySelector('.next-page')) {
    setTimeout(waitForBootstrap, 20); return;
  }
  var oldTitle = document.title;
  var oldDescription = document.querySelector('meta[name="description"]').content;
  document.querySelector('.next-page').click();
  (function waitForNextPage() {
    var canonical = document.querySelector('link[rel="canonical"]').href;
    var description = document.querySelector('meta[name="description"]').content;
    if (location.pathname !== '` + nextPath + `' ||
        document.documentElement.getAttribute('data-tour-rendered-route') !== '` + nextPath + `' ||
        canonical !== 'https://go-dev.shuijingwanwq.com` + nextPath + `' ||
        document.title === oldTitle || !description || description === oldDescription ||
        document.querySelectorAll('#editor-container').length !== 1) {
      setTimeout(waitForNextPage, 20); return;
    }
    document.querySelector('#run').click();
    (function waitForRunOutput() {
      var output = document.querySelector('.output.active pre');
      if (output && output.textContent.indexOf('Now you have 2.6457513110645907 problems.') !== -1) {
        document.documentElement.setAttribute('data-tour-bundle-e2e', 'PASS'); return;
      }
      setTimeout(waitForRunOutput, 20);
    }());
  }());
}());
</script>`
		body := bytes.Replace(recorder.Body.Bytes(), []byte("</body>"), []byte(script+"</body>"), 1)
		_, _ = w.Write(body)
	})
	browserServer := newIPv4TestServer(t, browserHandler)
	browserOrigin = browserServer.URL
	output := browserDumpDOM(t, chrome, browserServer.URL+initialPath, 12000)
	if !bytes.Contains(output, []byte(`data-tour-bundle-e2e="PASS"`)) {
		stopBinary()
		t.Fatalf("published bundle browser E2E failed; binary logs:\n%s\noutput=%s", binaryLogs(), browserOutputSummary(output))
	}
}

func courseRoute(routes []tour.CourseRoute, path string) (tour.CourseRoute, bool) {
	for _, route := range routes {
		if route.Path == path {
			return route, true
		}
	}
	return tour.CourseRoute{}, false
}

func newIPv4TestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return server
}

func browserTestChrome(t *testing.T) string {
	t.Helper()
	if os.Getenv("GO_TOUR_RUN_BROWSER_TESTS") != "1" {
		t.Skip("set GO_TOUR_RUN_BROWSER_TESTS=1 to run the Chrome integration test")
	}
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Skip("google-chrome is not installed")
	}
	return chrome
}

func browserDumpDOM(t *testing.T, chrome, target string, virtualTimeBudget int) []byte {
	t.Helper()
	command := exec.Command(chrome,
		"--headless=new",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-breakpad",
		"--disable-crash-reporter",
		"--disable-background-networking",
		"--disable-default-apps",
		"--disable-extensions",
		"--no-first-run",
		"--noerrdialogs",
		"--user-data-dir="+filepath.Join(t.TempDir(), "chrome-profile"),
		"--host-resolver-rules=MAP assets-go-dev.shuijingwanwq.com ~NOTFOUND, MAP fonts.googleapis.com ~NOTFOUND, MAP pagead2.googlesyndication.com ~NOTFOUND",
		"--run-all-compositor-stages-before-draw",
		fmt.Sprintf("--virtual-time-budget=%d", virtualTimeBudget),
		"--dump-dom",
		target,
	)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("Chrome %s: %v: %s", target, err, strings.TrimSpace(stderr.String()))
	}
	return output
}

func newFakePlaygroundProxy(t *testing.T) (string, string) {
	t.Helper()
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "go-tour browser test root"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "go.dev"},
		DNSNames:     []string{"go.dev"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, rootTemplate, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(leafKey)}),
	)
	if err != nil {
		t.Fatal(err)
	}
	playground := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/_/compile" {
			http.NotFound(w, request)
			return
		}
		if err := request.ParseForm(); err != nil || !strings.Contains(request.Form.Get("body"), "math.Sqrt(7)") {
			http.Error(w, "unexpected program", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"Events":[{"Message":"Now you have 2.6457513110645907 problems.\n","Kind":"stdout","Delay":0}]}`)
	}))
	playground.Listener.Close()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	playground.Listener = listener
	playground.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}
	playground.StartTLS()
	t.Cleanup(playground.Close)

	connectProxy := newIPv4TestServer(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodConnect {
			http.Error(w, "CONNECT required", http.StatusMethodNotAllowed)
			return
		}
		upstream, err := net.Dial("tcp4", playground.Listener.Addr().String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			upstream.Close()
			http.Error(w, "hijacking unsupported", http.StatusInternalServerError)
			return
		}
		client, buffered, err := hijacker.Hijack()
		if err != nil {
			upstream.Close()
			return
		}
		defer client.Close()
		defer upstream.Close()
		_, _ = buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
		if err := buffered.Flush(); err != nil {
			return
		}
		done := make(chan struct{})
		go func() {
			_, _ = io.Copy(upstream, client)
			_ = upstream.(*net.TCPConn).CloseWrite()
			close(done)
		}()
		_, _ = io.Copy(client, upstream)
		<-done
	}))
	rootPath := filepath.Join(t.TempDir(), "browser-test-root.pem")
	if err := os.WriteFile(rootPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER}), 0600); err != nil {
		t.Fatal(err)
	}
	return connectProxy.URL, rootPath
}

func startPublishedTourBinary(t *testing.T, bundle, proxyURL, rootCertificate string) (string, func(), func() string) {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(bundle, "bin", "tour"), "-http", address)
	command.Env = environmentWithOverrides(os.Environ(), map[string]string{
		"HTTPS_PROXY":    proxyURL,
		"HTTP_PROXY":     "",
		"NO_PROXY":       "127.0.0.1,localhost",
		"SSL_CERT_FILE":  rootCertificate,
		"TOUR_ANALYTICS": "",
		"TOUR_AD_HTML":   "",
	})
	var logs bytes.Buffer
	command.Stdout = &logs
	command.Stderr = &logs
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	wait := make(chan error, 1)
	go func() { wait <- command.Wait() }()
	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			_ = command.Process.Kill()
			select {
			case <-wait:
			case <-time.After(5 * time.Second):
				t.Errorf("production binary did not stop")
			}
		})
	}
	client := &http.Client{Timeout: 500 * time.Millisecond}
	baseURL := "http://" + address
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(baseURL + "/tour/basics/1")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return baseURL, stop, logs.String
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	stop()
	t.Fatalf("production binary did not become ready:\n%s", logs.String())
	return "", func() {}, logs.String
}

func browserOutputSummary(output []byte) string {
	document, err := html.Parse(bytes.NewReader(output))
	if err != nil {
		return string(output)
	}
	return fmt.Sprintf("route=%q title=%q canonical=%q output=%q",
		attrValue(findElement(document, "html", "", ""), "data-tour-rendered-route"),
		nodeText(findElement(document, "title", "", "")),
		attrValue(findElement(document, "link", "rel", "canonical"), "href"),
		nodeText(findElementByClass(document, "output")),
	)
}

func environmentWithOverrides(environment []string, overrides map[string]string) []string {
	result := make([]string, 0, len(environment)+len(overrides))
	for _, value := range environment {
		key := strings.SplitN(value, "=", 2)[0]
		if _, overridden := overrides[key]; !overridden {
			result = append(result, value)
		}
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}
