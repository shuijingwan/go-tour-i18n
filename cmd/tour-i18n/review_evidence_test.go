package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReviewEvidenceFormat(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		wantLine   string
		wantReason string
	}{
		{name: "empty", data: nil, wantLine: "line 1", wantReason: "file is empty"},
		{name: "normal multi-paragraph Markdown", data: []byte("# Review\n\nFirst paragraph.\n\n- decision: `A`\n")},
		{name: "two final LFs", data: []byte("# Review\n\n"), wantLine: "line 2", wantReason: "extra blank line"},
		{name: "trailing space", data: []byte("# Review \n"), wantLine: "line 1", wantReason: "trailing space or tab"},
		{name: "trailing tab", data: []byte("# Review\nFinding\t\n"), wantLine: "line 2", wantReason: "trailing space or tab"},
		{name: "missing final LF", data: []byte("# Review"), wantLine: "line 1", wantReason: "exactly one LF"},
		{name: "BOM", data: []byte("\xef\xbb\xbf# Review\n"), wantLine: "line 1", wantReason: "BOM"},
		{name: "invalid UTF-8", data: []byte("# Review\n\xff\n"), wantLine: "line 2", wantReason: "invalid UTF-8"},
		{name: "CRLF", data: []byte("# Review\r\n"), wantLine: "line 1", wantReason: "CRLF"},
		{name: "lone CR", data: []byte("# Review\rtext\n"), wantLine: "line 1", wantReason: "carriage return"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateReviewEvidence("draft.md", test.data)
			if test.wantReason == "" {
				if err != nil {
					t.Fatalf("valid Markdown rejected: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid Markdown accepted")
			}
			for _, want := range []string{"draft.md", test.wantLine, test.wantReason} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
		})
	}
}

func TestReviewEvidenceSaveRejectsInvalidInputWithoutOutput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "draft.md")
	outputDir := filepath.Join(root, "formal")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("invalid trailing space \n"), 0644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(outputDir, "review.md")
	if _, err := saveReviewEvidence(input, output); err == nil {
		t.Fatal("save accepted invalid input")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("invalid save left output behind: %v", err)
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("invalid save left staging files behind: %v", entries)
	}
}

func TestReviewEvidenceSaveDoesNotOverwriteExistingOutput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "draft.md")
	output := filepath.Join(root, "formal.md")
	want := []byte("existing receipt-bound evidence\n")
	if err := os.WriteFile(input, []byte("# New review\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, want, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := saveReviewEvidence(input, output); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("save did not reject existing output: %v", err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q want %q", got, want)
	}
}

func TestReviewEvidenceSavePreservesValidatedBytes(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "draft.md")
	output := filepath.Join(root, "formal.md")
	want := []byte("# Review\n\nA paragraph.\n")
	if err := os.WriteFile(input, want, 0600); err != nil {
		t.Fatal(err)
	}
	path, err := saveReviewEvidence(input, output)
	if err != nil {
		t.Fatal(err)
	}
	if path != output {
		t.Fatalf("saved path=%q want %q", path, output)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("saved bytes changed: got %q want %q", got, want)
	}
}

func TestReviewEvidenceCheckDoesNotModifyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "untracked-review.md")
	want := []byte("# Review\n\nNo repository state is required.\n")
	if err := os.WriteFile(path, want, 0644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := reviewEvidenceCommand([]string{"check", "--file", path}); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) || !after.ModTime().Equal(before.ModTime()) || after.Mode() != before.Mode() {
		t.Fatalf("check modified file: bytes_equal=%t modtime_before=%s modtime_after=%s mode_before=%s mode_after=%s", bytes.Equal(got, want), before.ModTime(), after.ModTime(), before.Mode(), after.Mode())
	}
}
