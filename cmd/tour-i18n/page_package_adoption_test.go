package main

import "testing"

func TestPagePackageAdoptionCLIRequiresExplicitAuthority(t *testing.T) {
	for _, args := range [][]string{
		{"page-package-adopt"},
		{"page-package-adopt", "--input", "data/proposal.json"},
		{"page-package-adopt", "--input-sha256", "missing-input"},
		{"page-package-adopt", "--record"},
		{"page-package-adoption-check", "extra"},
	} {
		if err := pagePackageAdoptionCommand(t.TempDir(), args); err == nil {
			t.Fatal("incomplete authority accepted", args)
		}
	}
}
