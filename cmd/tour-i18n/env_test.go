package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
