package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStructuredWorkflowCLIReadOnlyBoundary(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{"structured-plan", "--locale", "zh-CN", "--apply"},
		{"structured-plan", "--locale", "zh-CN", "--output", "unexpected.json"},
		{"structured-plan", "--locale", "../invalid"},
	} {
		if err := siteContentCommand(root, args); err == nil {
			t.Fatal("read-only scope flags accepted", args)
		}
	}
	output := filepath.Join(root, "existing-synthetic.zip")
	before := []byte("existing synthetic transport must remain immutable")
	if err := os.WriteFile(output, before, 0600); err != nil {
		t.Fatal(err)
	}
	if err := exportLocaleGenerationBundleCommand(root, nil, []string{"--locale", "zh-CN", "--task", "locale-assets", "--review-id", "foreign-scope", "--output", output}); err == nil {
		t.Fatal("structured export accepted unrelated Review scope")
	}
	if err := checkLocaleGenerationBundleCommand(root, nil, []string{"--bundle", output}); err == nil {
		t.Fatal("checker accepted malformed transport")
	}
	after, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("CLI rewrote existing transport", err)
	}
}
