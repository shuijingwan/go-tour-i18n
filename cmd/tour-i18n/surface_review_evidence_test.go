package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestSurfaceReviewEvidenceScaffoldKeepsLifecycleConclusionInFinalizableBlock(t *testing.T) {
	manifest := i18n.LocaleSurfaceReviewReviewerBundleManifest{
		ReviewPackage: i18n.LocaleSurfaceReviewReviewerBundleFile{SHA256: strings.Repeat("b", 64)},
		Coverage: i18n.LocaleSurfaceReviewPackageCoverage{
			Pages: 103, UI: 10, Articles: 7, TranslationUnits: 122, OtherSurfaces: 3,
		},
	}
	profile := finalizeProfile{Locale: "zz-ZZ", Hostname: "zz.example", PublicURL: "https://zz.example/", State: "first-production"}
	first := renderLocaleSurfaceReviewEvidenceScaffold("zz-ZZ", "review-1", "reviewer", "2026-09-21", strings.Repeat("a", 64), manifest, profile)
	second := renderLocaleSurfaceReviewEvidenceScaffold("zz-ZZ", "review-1", "reviewer", "2026-09-21", strings.Repeat("a", 64), manifest, profile)
	if !bytes.Equal(first, second) {
		t.Fatal("evidence scaffold is not deterministic")
	}
	if err := validateFinalizationPlaceholder(first); err != nil {
		t.Fatalf("generated first-production evidence is not finalizable: %v", err)
	}
	withoutBlock := bytes.Replace(first, []byte(finalizationPlaceholder), nil, 1)
	if bytes.Contains(withoutBlock, []byte("`PENDING`")) || bytes.Contains(bytes.ToLower(withoutBlock), []byte("later independent gate")) {
		t.Fatal("generated evidence contains stale lifecycle wording outside finalization block")
	}
	for _, want := range []string{"reviewer bundle SHA-256", "coverage: pages=103", "production state at scaffold time: `first-production`", "REVIEWER_TO_COMPLETE"} {
		if !bytes.Contains(first, []byte(want)) {
			t.Fatalf("generated evidence is missing %q", want)
		}
	}
}
