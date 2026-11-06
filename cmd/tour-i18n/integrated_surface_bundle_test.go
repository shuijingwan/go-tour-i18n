package main

import "testing"

func TestIntegratedSurfaceReviewerBundleCLI(t *testing.T) {
	for _, args := range [][]string{
		{"surface-reviewer-bundle"},
		{"surface-reviewer-bundle", "--locale", "fr-FR", "--output", "review.zip"},
		{"surface-reviewer-bundle", "--input", "scope.json", "--generation", "closure.json", "--output", "review.zip"},
		{"surface-reviewer-bundle-check"},
		{"surface-reviewer-bundle-check", "--locale", "zh-CN"},
		{"surface-reviewer-bundle-check", "--bundle", "bundle.zip", "unexpected"},
	} {
		if err := siteContentCommand(t.TempDir(), args); err == nil {
			t.Fatal("incomplete/mixed/legacy selector accepted", args)
		}
	}
}
