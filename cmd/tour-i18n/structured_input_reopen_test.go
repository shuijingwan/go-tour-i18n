package main

import (
	"os"
	"testing"
)

func TestStructuredInputReopenCommandScope(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{}, {"--locale", "zh-CN"}, {"--asset", "source:_content/learn/books.yaml"},
		{"--locale", "zh-CN", "--asset", "source:_content/learn/books.yaml", "unexpected"},
		{"--locale", "zh-CN", "--asset", "source:_content/learn/books.yaml", "--input", "arbitrary.json"},
		{"--locale", "../escape", "--asset", "source:_content/learn/books.yaml"},
	} {
		if err := siteContentCommand(root, append([]string{"structured-input-reopen"}, args...)); err == nil {
			t.Fatal("invalid/external authority accepted", args)
		}
	}
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 0 {
		t.Fatal("invalid preflight mutated root", err)
	}
}
