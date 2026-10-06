package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestGlossaryCompatibilityCLIRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown"}, {"archive"}, {"archive", "--all", "--locale", "zh-CN"}, {"assess", "--all"}, {"check", "--locale", "zh-CN", "--evidence", "../outside"}, {"check", "--locale", "zh-CN", "--evidence", "/tmp/outside"}, {"status", "--locale", "zh-CN", "--review-id", "other"}, {"archive", "--locale", "zh-CN", "--scope", "tu:one"}, {"config-baseline", "--all", "--review-id", "one"}, {"archive", "--locale", "../escape"}, {"archive", "--locale", "zh-CN", "trailing"}} {
		if glossaryCompatibilityCommand(t.TempDir(), args) == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestGlossaryCompatibilityCLIArchiveIsImmutableAndReviewed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "locales/zh-CN/glossary.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	original := []byte("locale: zh-CN\nmandatory:\n  Go: Go\n")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	args := []string{"archive", "--locale", "zh-CN"}
	if glossaryCompatibilityCommand(root, args) == nil {
		t.Fatal("unreviewed glossary archived")
	}
	if _, _, err := i18n.RecordGlossaryReview(root, "zh-CN", "original", "independent-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := glossaryCompatibilityCommand(root, args); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, append(original, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if glossaryCompatibilityCommand(root, args) == nil {
		t.Fatal("byte-stale Review allowed archive/current")
	}
	entries, err := os.ReadDir(filepath.Join(root, "data/glossary-history/zh-CN"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("archive=%v err=%v", entries, err)
	}
}
