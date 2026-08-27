package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

func TestLoadProjectEnvMissingFile(t *testing.T) {
	if err := loadProjectEnv(t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestLoadProjectEnvLoadsUnsetVariable(t *testing.T) {
	root := t.TempDir()
	key := "GO_TOUR_I18N_TEST_DOTENV_LOAD"
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(key+"=fake-local-value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Unsetenv(key) })
	if err := loadProjectEnv(root); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv(key); got != "fake-local-value" {
		t.Fatalf("loaded value = %q", got)
	}
}

func TestLoadProjectEnvDoesNotOverrideEnvironment(t *testing.T) {
	root := t.TempDir()
	key := "GO_TOUR_I18N_TEST_DOTENV_PRECEDENCE"
	t.Setenv(key, "system-value")
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(key+"=fake-file-value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadProjectEnv(root); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv(key); got != "system-value" {
		t.Fatalf("environment was overwritten: %q", got)
	}
}

func TestLoadProjectEnvRejectsMalformedFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("MALFORMED='unterminated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	err := loadProjectEnv(root)
	if err == nil || !strings.Contains(err.Error(), "load project .env") {
		t.Fatalf("error = %v", err)
	}
}

func TestParsePreviewOptionsSupportsSingleAndCompletePreview(t *testing.T) {
	single, err := parsePreviewOptions([]string{"--locale", "zh-CN", "--id", "methods/17"})
	if err != nil {
		t.Fatal(err)
	}
	if single.Locale != "zh-CN" || single.ID != "methods/17" || single.HTTPAddr != "127.0.0.1:3999" {
		t.Fatalf("single preview options = %+v", single)
	}
	complete, err := parsePreviewOptions([]string{"--locale", "zh-CN", "--http", "127.0.0.1:4999"})
	if err != nil {
		t.Fatal(err)
	}
	if complete.Locale != "zh-CN" || complete.ID != "" || complete.HTTPAddr != "127.0.0.1:4999" {
		t.Fatalf("complete preview options = %+v", complete)
	}
	if _, err := parsePreviewOptions([]string{"--id", "methods/17"}); err == nil {
		t.Fatal("preview without locale was accepted")
	}
}

func TestBuildLocaleCommandRequiresCompleteWorkflow(t *testing.T) {
	root, catalog, pendingUnit := incompletePublishTestCatalog(t)
	output := filepath.Join(t.TempDir(), "cli-projection")
	err := buildLocale(root, catalog, []string{"--locale", "zh-CN", "--output", output})
	if err == nil || !strings.Contains(err.Error(), "workflow translation units") || !strings.Contains(err.Error(), pendingUnit+"=pending") {
		t.Fatalf("CLI projection error = %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("blocked CLI projection created output: %v", statErr)
	}
}

func TestCompletePreviewHandlerUsesProductionStyleSurface(t *testing.T) {
	root, catalog := publishTestCatalog(t)
	projection, err := i18n.BuildLocaleProjection(root, catalog, "zh-CN", filepath.Join(t.TempDir(), "projection"))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := tour.NewPreviewHandler(os.DirFS(projection.ContentDir), "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	server := newIPv4TestServer(t, handler)

	get := func(path string) (int, string, string) {
		t.Helper()
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, response.Header.Get("Content-Type"), string(body)
	}
	if status, contentType, body := get("/robots.txt"); status != http.StatusOK || contentType != "text/plain; charset=utf-8" || !strings.Contains(body, "Sitemap: https://go-dev.shuijingwanwq.com/sitemap.xml") || strings.Contains(body, "<!doctype html") {
		t.Fatalf("robots = status=%d content-type=%q body=%q", status, contentType, body)
	}
	if status, contentType, body := get("/sitemap.xml"); status != http.StatusOK || contentType != "application/xml; charset=utf-8" || !strings.Contains(body, "<loc>https://go-dev.shuijingwanwq.com/tour/welcome/1</loc>") || strings.Contains(body, "<!doctype html") {
		t.Fatalf("sitemap = status=%d content-type=%q body=%q", status, contentType, body)
	}
	if status, _, body := get("/"); status != http.StatusOK || !strings.Contains(body, `<meta name="description" content="这是一个由社区维护的 Go 官方学习内容翻译项目。当前首先提供简体中文，后续可自然扩展至其他语言和其他 Go 内容。">`) || !strings.Contains(body, `<link rel="canonical" href="https://go-dev.shuijingwanwq.com/">`) {
		t.Fatalf("homepage = status=%d body=%q", status, body)
	}
	if status, _, body := get("/tour/script.js"); status != http.StatusOK || !strings.Contains(body, "window.transport = HTTPTransport();") || !strings.Contains(body, `window.socketAddr = "";`) || !strings.Contains(body, `window.playgroundBaseURL = "";`) || strings.Contains(body, "window.transport = SocketTransport();") || strings.Contains(body, "https://play.go-dev.shuijingwanwq.com:8443") {
		t.Fatalf("preview script is not HTTPTransport: status=%d body=%q", status, body)
	}
	for _, path := range []string{"/socket", "/socket/", "/_/compile", "/_/fmt"} {
		status, _, body := get(path)
		if path == "/socket" || path == "/socket/" {
			if status != http.StatusNotFound {
				t.Errorf("GET %s: status=%d body=%q, want 404", path, status, body)
			}
		} else if status == http.StatusOK || strings.Contains(body, "<!doctype html") {
			t.Errorf("GET %s did not reach HTTP Playground path: status=%d body=%q", path, status, body)
		}
	}
}
