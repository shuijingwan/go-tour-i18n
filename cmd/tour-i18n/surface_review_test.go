package main

import (
	"strings"
	"testing"
)

func TestFullLocalePreviewRequiresCurrentSurfaceReviewAGate(t *testing.T) {
	root, catalog := publishTestCatalog(t)
	if err := requireFullLocalePreviewGate(root, catalog, "es-ES"); err == nil || !strings.Contains(err.Error(), "Locale Surface Review A gate missing") {
		t.Fatalf("full preview gate error=%v", err)
	}
}

func TestCandidatePreviewIsNotSubjectToFullLocaleGate(t *testing.T) {
	options := previewOptions{Locale: "es-ES", ID: "welcome/1"}
	if previewRequiresLocaleSurfaceReviewAGate(options) {
		t.Fatal("candidate preview would be incorrectly gated")
	}
}
