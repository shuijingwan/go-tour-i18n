package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestWriteCourseMetadataAtomicCreatesOutput(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "course-metadata.json")
	want := []byte("new complete metadata\n")
	if err := writeCourseMetadataAtomic(output, want); err != nil {
		t.Fatal(err)
	}
	assertCourseMetadataOutput(t, output, want)
	assertNoCourseMetadataStaging(t, dir, filepath.Base(output))
}

func TestWriteCourseMetadataAtomicReplacesOutput(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "course-metadata.json")
	if err := os.WriteFile(output, []byte("old metadata\n"), 0644); err != nil {
		t.Fatal(err)
	}
	want := []byte("new complete metadata\n")
	if err := writeCourseMetadataAtomic(output, want); err != nil {
		t.Fatal(err)
	}
	assertCourseMetadataOutput(t, output, want)
	assertNoCourseMetadataStaging(t, dir, filepath.Base(output))
}

func TestWriteCourseMetadataAtomicRejectsInvalidParent(t *testing.T) {
	tests := []struct {
		name   string
		output func(string) string
	}{
		{"missing", func(root string) string { return filepath.Join(root, "missing", "course-metadata.json") }},
		{"not directory", func(root string) string {
			parent := filepath.Join(root, "parent-file")
			if err := os.WriteFile(parent, []byte("not a directory"), 0644); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(parent, "course-metadata.json")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			output := test.output(root)
			if err := writeCourseMetadataAtomic(output, []byte("metadata\n")); err == nil {
				t.Fatal("write with invalid parent succeeded")
			}
			if _, err := os.Lstat(output); err == nil {
				t.Fatal("target exists after failed write")
			}
			assertNoCourseMetadataStaging(t, root, filepath.Base(output))
		})
	}
}

func TestAssembleCourseMetadataValidationFailurePreservesOutput(t *testing.T) {
	root := t.TempDir()
	descriptions := filepath.Join(root, "descriptions.json")
	if err := os.WriteFile(descriptions, []byte(`{"pages":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "course-metadata.json")
	old := []byte("existing complete metadata\n")
	if err := os.WriteFile(output, old, 0644); err != nil {
		t.Fatal(err)
	}
	catalog := &i18n.Catalog{Pages: []i18n.Page{{ID: "lesson/1", Route: "/lesson/1"}}}
	err := assembleCourseMetadata(root, catalog, []string{
		"--locale", "test-LOCALE",
		"--descriptions", descriptions,
		"--provider", "codex",
		"--model", "fixture-model",
		"--generated-at", "2026-08-26T01:02:03Z",
		"--output", output,
	})
	if err == nil {
		t.Fatal("assemble with incomplete descriptions succeeded")
	}
	assertCourseMetadataOutput(t, output, old)
	assertNoCourseMetadataStaging(t, root, filepath.Base(output))
}

func assertCourseMetadataOutput(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("output=%q, want exact bytes %q", got, want)
	}
}

func assertNoCourseMetadataStaging(t *testing.T, dir, outputBase string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "."+outputBase+".staging-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("staging files remain: %v", matches)
	}
}
