package main

import (
	"strings"
	"testing"
)

func TestShellSuccessorCLIRejectsScopeMutations(t *testing.T) {
	for _, args := range [][]string{
		{"shell-successor"},
		{"shell-successor", "--input", "elsewhere.json"},
		{"shell-successor", "--package", "site-v3"},
		{"shell-successor-check", "--locale", "zh-CN"},
		{"shell-successor-check", "--apply"},
		{"shell-successor-check"},
		{"shell-successor", "--locale", "zh-CN", "unexpected"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if err := siteContentCommand(t.TempDir(), args); err == nil {
				t.Fatal("unapproved successor flags accepted")
			}
		})
	}
}
