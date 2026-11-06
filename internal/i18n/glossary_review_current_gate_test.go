package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentGlossaryReviewFreshGates(t *testing.T) {
	root := t.TempDir()
	writeGlossaryReviewTestGlossary(t, root, "zh-CN")
	recordSyntheticUnifiedReview(t, root, "current-review-gate")
	if err := RequireCurrentGlossaryReview(root, "zh-CN"); err != nil {
		t.Fatal(err)
	}
	reviews, err := currentUnifiedReviews(root, "zh-CN")
	if err != nil || len(reviews) != 1 {
		t.Fatal(reviews, err)
	}
	for _, name := range []string{
		glossaryReviewGlossaryPath("zh-CN"),
		unifiedReviewerArchive("zh-CN", reviews[0].BundleSHA),
		UnifiedGlossaryReviewPath("zh-CN", reviews[0].ReviewID),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, filepath.FromSlash(name))
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.WriteFile(path, before, 0644); err != nil {
					t.Fatal(err)
				}
			}()
			if err := os.WriteFile(path, append(bytes.Clone(before), '\n'), 0644); err != nil {
				t.Fatal(err)
			}
			// Review JSON whitespace is not an identity change. Mutate its actual
			// bytes into malformed JSON to exercise the same strict reader gate.
			if name == UnifiedGlossaryReviewPath("zh-CN", reviews[0].ReviewID) {
				if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := RequireCurrentGlossaryReview(root, "zh-CN"); err == nil {
				t.Fatal("current glossary/Review/archive mutation accepted")
			}
		})
	}
	if err := RequireCurrentGlossaryReview(root, "zh-CN"); err != nil {
		t.Fatal("restored exact authority", err)
	}
	recordSyntheticUnifiedReview(t, root, "conflicting-current-review-gate")
	if err := RequireCurrentGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("ambiguous current Review accepted")
	}
}
