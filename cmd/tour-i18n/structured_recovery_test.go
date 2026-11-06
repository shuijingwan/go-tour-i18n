package main

import (
	"strings"
	"testing"
)

func TestStructuredRecoveryCLIRejectsIncompleteOrUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{{"structured-recover"}, {"structured-recover", "--input", "output.json"}, {"structured-recover", "--locale", "zh-CN"}, {"structured-recover", "--bundle", "input.zip", "extra"}} {
		if err := siteContentCommand(t.TempDir(), args); err == nil {
			t.Fatal("invalid recovery arguments accepted", args)
		}
	}
	args := []string{"structured-recover", "--bundle", "missing.zip", "--input", "missing.json", "--bundle-sha256", strings.Repeat("a", 64), "--output-sha256", strings.Repeat("b", 64), "--provider", "chatgpt", "--model", "gpt-5.6-sol-high", "--session", "original-session", "--generated-at", "2026-10-09T06:30:15Z"}
	if err := siteContentCommand(t.TempDir(), args); err == nil || !strings.Contains(err.Error(), "missing.zip") {
		t.Fatalf("read-only entry error=%v", err)
	}
}
