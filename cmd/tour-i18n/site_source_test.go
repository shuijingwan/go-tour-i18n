package main

import "testing"

func TestSourceAuthorityCLIRejectsMutationFlagsOnReadOnlyCommands(t *testing.T) {
	for _, args := range [][]string{
		{"source-check", "--apply=false"},
		{"source-check", "--website-root="},
		{"source-retirement-check", "--apply"},
		{"source-retirement-check", "--website-root=/tmp"},
		{"source-storage-compatibility", "--website-root=/tmp"},
		{"source-storage-compatibility", "--apply", "extra"},
		{"source-migrate", "--apply"},
		{"source-migrate", "--website-root=/tmp", "extra"},
	} {
		if err := siteSourceCommand(t.TempDir(), args); err == nil {
			t.Fatal("unsafe command flags accepted", args)
		}
	}
}
