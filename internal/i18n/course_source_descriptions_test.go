package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCourseSourceDescriptionsStrictValidationAndStaleness(t *testing.T) {
	_, catalog, _ := writeCourseMetadataRefreshFixture(t, 3)
	data := assembleCourseSourceDescriptionFixture(t, catalog, nil)
	asset, err := validateCourseSourceDescriptions(data, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(asset.Pages) != len(catalog.Pages) || asset.Pages[0].PageID != catalog.Pages[0].ID {
		t.Fatalf("unexpected canonical source-description asset: %+v", asset)
	}

	withUnknown := []byte(strings.Replace(string(data), `"pages":`, `"unknown":true,"pages":`, 1))
	if _, err := validateCourseSourceDescriptions(withUnknown, catalog); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error=%v", err)
	}

	reordered := *asset
	reordered.Pages = append([]CourseSourceDescriptionPage(nil), asset.Pages...)
	reordered.Pages[0], reordered.Pages[1] = reordered.Pages[1], reordered.Pages[0]
	reorderedData, _ := json.Marshal(reordered)
	if _, err := validateCourseSourceDescriptions(reorderedData, catalog); err == nil || !strings.Contains(err.Error(), "catalog page_id") {
		t.Fatalf("out-of-order asset error=%v", err)
	}

	changed := *catalog
	changed.Pages = append([]Page(nil), catalog.Pages...)
	changed.Pages[0].Source = []byte("changed complete English source")
	changed.Pages[0].SourceSHA256 = sum(changed.Pages[0].Source)
	if _, err := validateCourseSourceDescriptions(data, &changed); err == nil || !strings.Contains(err.Error(), "source_sha256 is stale") {
		t.Fatalf("English source change error=%v", err)
	}
}

func TestCourseSourceDescriptionReviewGateCurrentStaleAndMalformed(t *testing.T) {
	root, catalog, _ := writeCourseMetadataRefreshFixture(t, 3)
	writeCourseSourceDescriptionAsset(t, root, catalog, nil)
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err == nil || !strings.Contains(err.Error(), "gate missing") {
		t.Fatalf("missing source-description review gate accepted: %v", err)
	}
	if _, _, err := BuildCourseSourceDescriptionReviewGate(root, "missing-evidence", "reviewer", catalog); err == nil || !strings.Contains(err.Error(), "evidence must exist") {
		t.Fatalf("review gate without human evidence accepted: %v", err)
	}
	writeCourseSourceDescriptionReview(t, root, catalog, "review-1")
	firstAuthority, err := RequireCurrentCourseSourceDescriptionReview(root, catalog)
	if err != nil || firstAuthority == "" {
		t.Fatalf("current source-description review rejected: identity=%q err=%v", firstAuthority, err)
	}
	evidencePath, _ := CourseSourceDescriptionReviewEvidencePath(root, "review-1")
	evidence, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidencePath, append(evidence, []byte("\nmodified after gate\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("modified review evidence accepted: %v", err)
	}
	if err := os.WriteFile(evidencePath, evidence, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err != nil {
		t.Fatalf("restored current evidence rejected: %v", err)
	}
	if err := os.Remove(evidencePath); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err == nil || !strings.Contains(err.Error(), "read current") {
		t.Fatalf("deleted review evidence accepted: %v", err)
	}
	if err := os.WriteFile(evidencePath, evidence, 0644); err != nil {
		t.Fatal(err)
	}
	changedCatalog := *catalog
	changedCatalog.Pages = append([]Page(nil), catalog.Pages...)
	changedCatalog.Pages[0].Source = []byte("changed complete English Page source")
	changedCatalog.Pages[0].SourceSHA256 = sum(changedCatalog.Pages[0].Source)
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, &changedCatalog); err == nil || !strings.Contains(err.Error(), "source_sha256 is stale") {
		t.Fatalf("English source change did not stale canonical asset and review gate: %v", err)
	}

	writeCourseSourceDescriptionAsset(t, root, catalog, map[string]string{catalog.Pages[0].ID: "changed"})
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("changed source-description asset did not stale review gate: %v", err)
	}
	writeCourseSourceDescriptionReview(t, root, catalog, "review-2")
	secondAuthority, err := RequireCurrentCourseSourceDescriptionReview(root, catalog)
	if err != nil || secondAuthority == firstAuthority {
		t.Fatalf("new review authority identity=%q first=%q err=%v", secondAuthority, firstAuthority, err)
	}

	bad := filepath.Join(root, "data", "course-seo", "source-description-reviews", "malformed.gate.json")
	if err := os.WriteFile(bad, []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("malformed review gate accepted: %v", err)
	}
	if err := os.WriteFile(bad, []byte(`{"unknown":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown review gate field accepted: %v", err)
	}
	validGatePath, err := CourseSourceDescriptionReviewGatePath(root, "review-2")
	if err != nil {
		t.Fatal(err)
	}
	validGate, err := os.ReadFile(validGatePath)
	if err != nil {
		t.Fatal(err)
	}
	duplicateDecision := strings.Replace(string(validGate), `"decision": "passed"`, `"decision": "failed",`+"\n  "+`"decision": "passed"`, 1)
	if err := os.WriteFile(bad, []byte(duplicateDecision), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err == nil || !strings.Contains(err.Error(), `duplicate JSON object member name "decision"`) {
		t.Fatalf("duplicate top-level review gate decision accepted: %v", err)
	}
	duplicateArtifact := strings.Replace(string(validGate), `"artifact_sha256": "`, `"artifact_sha256": "stale",`+"\n    "+`"artifact_sha256": "`, 1)
	if err := os.WriteFile(bad, []byte(duplicateArtifact), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err == nil || !strings.Contains(err.Error(), `duplicate JSON object member name "artifact_sha256"`) {
		t.Fatalf("duplicate nested review gate artifact_sha256 accepted: %v", err)
	}
}

func TestBuildCourseSourceDescriptionReviewGateRejectsInvalidEvidence(t *testing.T) {
	tests := []struct {
		name     string
		reviewID string
		reviewer string
		evidence func(string, *Catalog, CourseSourceDescriptionReviewInputs) []byte
		want     string
	}{
		{
			name: "missing block", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(string, *Catalog, CourseSourceDescriptionReviewInputs) []byte { return nil },
			want:     "exactly one",
		},
		{
			name: "arbitrary non-empty Markdown", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(string, *Catalog, CourseSourceDescriptionReviewInputs) []byte {
				return []byte("# Review\n\nLooks good.\n")
			},
			want: "exactly one",
		},
		{
			name: "duplicate block", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(reviewID string, catalog *Catalog, inputs CourseSourceDescriptionReviewInputs) []byte {
				block := validCourseSourceDescriptionReviewEvidence(t, reviewID, "reviewer", "passed", catalog, inputs)
				return append(block, block...)
			},
			want: "exactly one",
		},
		{
			name: "malformed block", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(string, *Catalog, CourseSourceDescriptionReviewInputs) []byte {
				return []byte(courseSourceDescriptionReviewStartMarker + "\n{\n" + courseSourceDescriptionReviewEndMarker + "\n")
			},
			want: "malformed",
		},
		{
			name: "failed decision", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(reviewID string, catalog *Catalog, inputs CourseSourceDescriptionReviewInputs) []byte {
				return validCourseSourceDescriptionReviewEvidence(t, reviewID, "reviewer", "failed", catalog, inputs)
			},
			want: "decision must be passed",
		},
		{
			name: "duplicate decision", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(reviewID string, catalog *Catalog, inputs CourseSourceDescriptionReviewInputs) []byte {
				evidence := validCourseSourceDescriptionReviewEvidence(t, reviewID, "reviewer", "passed", catalog, inputs)
				return []byte(strings.Replace(string(evidence), `"decision": "passed"`, `"decision": "failed",`+"\n  "+`"decision": "passed"`, 1))
			},
			want: `duplicate JSON object member name "decision"`,
		},
		{
			name: "duplicate artifact SHA", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(reviewID string, catalog *Catalog, inputs CourseSourceDescriptionReviewInputs) []byte {
				evidence := validCourseSourceDescriptionReviewEvidence(t, reviewID, "reviewer", "passed", catalog, inputs)
				return []byte(strings.Replace(string(evidence), `"artifact_sha256": "`, `"artifact_sha256": "stale",`+"\n  "+`"artifact_sha256": "`, 1))
			},
			want: `duplicate JSON object member name "artifact_sha256"`,
		},
		{
			name: "stale artifact", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(reviewID string, catalog *Catalog, inputs CourseSourceDescriptionReviewInputs) []byte {
				inputs.ArtifactSHA256 = strings.Repeat("0", 64)
				return validCourseSourceDescriptionReviewEvidence(t, reviewID, "reviewer", "passed", catalog, inputs)
			},
			want: "identity is stale",
		},
		{
			name: "stale catalog", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(reviewID string, catalog *Catalog, inputs CourseSourceDescriptionReviewInputs) []byte {
				inputs.CatalogSourceSHA256 = strings.Repeat("0", 64)
				return validCourseSourceDescriptionReviewEvidence(t, reviewID, "reviewer", "passed", catalog, inputs)
			},
			want: "identity is stale",
		},
		{
			name: "review id mismatch", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(_ string, catalog *Catalog, inputs CourseSourceDescriptionReviewInputs) []byte {
				return validCourseSourceDescriptionReviewEvidence(t, "review-2", "reviewer", "passed", catalog, inputs)
			},
			want: "review_id",
		},
		{
			name: "reviewer mismatch", reviewID: "review-1", reviewer: "reviewer",
			evidence: func(reviewID string, catalog *Catalog, inputs CourseSourceDescriptionReviewInputs) []byte {
				return validCourseSourceDescriptionReviewEvidence(t, reviewID, "other-reviewer", "passed", catalog, inputs)
			},
			want: "reviewer",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, catalog, _ := writeCourseMetadataRefreshFixture(t, 3)
			writeCourseSourceDescriptionAsset(t, root, catalog, nil)
			inputs, err := currentCourseSourceDescriptionReviewInputs(root, catalog)
			if err != nil {
				t.Fatal(err)
			}
			evidencePath, err := CourseSourceDescriptionReviewEvidencePath(root, test.reviewID)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(evidencePath), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(evidencePath, test.evidence(test.reviewID, catalog, inputs), 0644); err != nil {
				t.Fatal(err)
			}
			if _, _, err := BuildCourseSourceDescriptionReviewGate(root, test.reviewID, test.reviewer, catalog); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func assembleCourseSourceDescriptionFixture(t *testing.T, catalog *Catalog, changed map[string]string) []byte {
	t.Helper()
	input := courseDescriptionsForCatalog(catalog)
	for i := range input.Pages {
		identity := input.Pages[i].PageID
		if suffix := changed[identity]; suffix != "" {
			input.Pages[i].Description = "A changed canonical English semantic summary grounded only in the complete lesson for page " + identity + " " + suffix + "."
		} else {
			input.Pages[i].Description = "A canonical English semantic summary grounded only in the complete lesson for page " + identity + "."
		}
	}
	descriptions, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	data, err := AssembleCourseSourceDescriptions(catalog, CourseSourceDescriptionAssemblyOptions{Provider: "fixture", Model: "fixture-model", GeneratedAt: "2026-09-15T01:02:03Z", Descriptions: descriptions})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeCourseSourceDescriptionAsset(t *testing.T, root string, catalog *Catalog, changed map[string]string) {
	t.Helper()
	path := CourseSourceDescriptionsPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, assembleCourseSourceDescriptionFixture(t, catalog, changed), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeCourseSourceDescriptionReview(t *testing.T, root string, catalog *Catalog, reviewID string) {
	t.Helper()
	evidencePath, err := CourseSourceDescriptionReviewEvidencePath(root, reviewID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(evidencePath), 0755); err != nil {
		t.Fatal(err)
	}
	inputs, err := currentCourseSourceDescriptionReviewInputs(root, catalog)
	if err != nil {
		t.Fatal(err)
	}
	evidence := validCourseSourceDescriptionReviewEvidence(t, reviewID, "reviewer", "passed", catalog, inputs)
	if err := os.WriteFile(evidencePath, evidence, 0644); err != nil {
		t.Fatal(err)
	}
	data, path, err := BuildCourseSourceDescriptionReviewGate(root, reviewID, "reviewer", catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func validCourseSourceDescriptionReviewEvidence(t *testing.T, reviewID, reviewer, decision string, catalog *Catalog, inputs CourseSourceDescriptionReviewInputs) []byte {
	t.Helper()
	block := courseSourceDescriptionReviewEvidenceBlock{
		ReviewID: reviewID, ArtifactSHA256: inputs.ArtifactSHA256, CatalogSourceSHA256: inputs.CatalogSourceSHA256,
		SourceDescriptionSchemaVersion: inputs.SchemaVersion, GeneratorContract: inputs.GeneratorContract,
		PromptVersion: inputs.PromptVersion, PageCount: len(catalog.Pages), Reviewer: reviewer, Decision: decision,
	}
	data, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return []byte("# Canonical English course source-description review\n\n" + courseSourceDescriptionReviewStartMarker + "\n" + string(data) + "\n" + courseSourceDescriptionReviewEndMarker + "\n")
}
