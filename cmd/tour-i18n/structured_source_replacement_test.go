package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStructuredSourceReplacementCommandBoundaries(t *testing.T) {
	root := t.TempDir()
	raw := []byte("{}\n")
	if err := os.WriteFile(filepath.Join(root, "proposal.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	base := []string{"structured-source-replace", "--input", "proposal.json", "--input-sha256", fmt.Sprintf("%x", sha256.Sum256(raw))}
	for _, tc := range []struct {
		flags []string
		want  string
	}{
		{[]string{"--apply"}, "requires exact"},
		{[]string{"--prior-identity", "unused"}, "CAS flags"},
		{[]string{"--locale", "zh-CN"}, "locale independent"},
	} {
		if err := siteContentCommand(root, append(append([]string{}, base...), tc.flags...)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatal("CLI boundary", tc, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(err) {
		t.Fatal("invalid command mutated repository")
	}
}
