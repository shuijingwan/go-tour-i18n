package main

import (
	"bytes"
	"os"
	"testing"
	"time"
)

// Opt-in existing-artifact probe. Run the compiled test from the repository
// root; this is the exact read-only CLI current-check, with no package flag.
func TestQualityCheckReviewerBundleExistingCurrent(t *testing.T) {
	path := os.Getenv("GO_TOUR_REVIEWER_CURRENT_BUNDLE")
	if path == "" {
		t.Skip("requires GO_TOUR_REVIEWER_CURRENT_BUNDLE")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	err = run([]string{"quality-check", "reviewer-bundle-check", "--bundle", path})
	t.Logf("CANONICAL_CURRENT_CHECK_WALL=%s", time.Since(started))
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatal("existing bundle changed", readErr)
	}
	if err != nil {
		t.Fatal(err)
	}
}
