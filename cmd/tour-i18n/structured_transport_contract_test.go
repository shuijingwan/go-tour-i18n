package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStructuredTransportCommandContract(t *testing.T) {
	b, err := os.ReadFile("../../docs/STRUCTURED_GENERATION_TRANSPORT.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"COMMAND", "OPERATION", "INPUT_KIND", "INPUT_PATH_CONTRACT", "MUTATION_EXPECTATION", "IDEMPOTENCE", "CURRENTNESS_CHECK", "PRIOR_OR_CAS", "SUCCESS_STOP", "FAILURE_STOP", "repository-relative", "validated_repository_inputs", "33554432", "32 MiB"} {
		// Numeric limit is asserted in the reader tests; prose uses 32 MiB.
		if s == "33554432" {
			continue
		}
		if !strings.Contains(string(b), s) {
			t.Fatal("missing contract field", s)
		}
	}
	root := t.TempDir()
	if err = os.WriteFile(filepath.Join(root, "plan.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err = siteContentCommand(root, []string{"structured-check", "--bundle", "plan.json"}); err == nil || !strings.Contains(err.Error(), "open transport bundle") {
		t.Fatal("check accepted Plan instead of ZIP", err)
	}
	if err = siteContentCommand(root, []string{"structured-check", "--bundle", filepath.Join(root, "plan.json")}); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatal("absolute input accepted", err)
	}
	for _, args := range [][]string{
		{"structured-check", "--bundle", "plan.json", "--output", "unexpected.zip"},
		{"structured-check", "--bundle", "plan.json", "--locale", "zh-CN"},
		{"structured-export", "--locale", "zh-CN", "--apply"},
		{"structured-export", "--locale", "zh-CN", "--record"},
	} {
		if err := siteContentCommand(root, args); err == nil {
			t.Fatal("invalid mutation/input flags accepted", args)
		}
	}
}
