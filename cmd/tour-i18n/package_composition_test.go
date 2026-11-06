package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageCompositionPreservesPredecessorOrder(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"z-first.json", "a-second.json"} {
		if err := os.WriteFile(filepath.Join(root, p), []byte("{}\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	refs, err := siteContentReferencesInOrder(root, []string{"z-first.json", "a-second.json"})
	if err != nil || len(refs) != 2 || refs[0].Path != "z-first.json" || refs[1].Path != "a-second.json" {
		t.Fatal("membership predecessor order was sorted", err)
	}
}

func TestPackageCompositionCommandRejectsMutationFlags(t *testing.T) {
	for _, flag := range []string{"--apply", "--record", "--input", "--generation"} {
		t.Run(flag, func(t *testing.T) {
			args := []string{"package-composition", flag}
			if flag == "--input" || flag == "--generation" {
				args = append(args, "untrusted.json")
			}
			if err := siteContentCommand(t.TempDir(), args); err == nil {
				t.Fatal("readonly composition accepted foreign flag")
			}
		})
	}
}

func TestPackageClosureRejectsMixedFinalizationModes(t *testing.T) {
	err := siteContentCommand(t.TempDir(), []string{"package-closure", "--locale", "zh-CN", "--package", "generic-fixture", "--input", "one.json", "--page-finalization", "two.json"})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatal("mixed Snapshot and composition accepted", err)
	}
}
