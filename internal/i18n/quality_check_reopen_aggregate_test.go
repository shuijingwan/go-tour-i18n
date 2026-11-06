package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This reader fixture isolates shared authorization/serialization. The actual
// Integrated scope/receipt reader is exercised in sitecontent focused tests.
func aggregateSurfaceAuthorityFixture(t *testing.T, root string, c *Catalog, snapshot string) (*SurfaceReviewAuthority, QualityCheckSurfaceReopenOptions) {
	t.Helper()
	// Current eligibility is a separate, mandatory shared authorization gate.
	copyBundleAuthority(t, root, append(append([]string(nil), translationUnitGenerationAuthorityPaths...), "go.mod", "go.sum", "internal/i18n/translation_authorization.go", "internal/i18n/glossary_semantics.go", "internal/i18n/glossary.go", "internal/i18n/glossary_context.go", "internal/i18n/glossary_compatibility.go", "internal/i18n/validate.go", "internal/i18n/go_example_validate.go", "internal/i18n/go_example_protect.go", "internal/i18n/translate_protect.go", "internal/i18n/preformatted.go", "locales/zh-CN/locale.json"))
	copyBundleAuthority(t, root, commentGateAuthorityPaths)
	s, err := readQualityCheckSnapshotForReview(root, "zh-CN", snapshot, c)
	if err != nil {
		t.Fatal(err)
	}
	path := "data/locale-language/zh-CN/surface/aggregate-fixture.json"
	data := []byte("immutable synthetic failed Surface receipt\n")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, path), data, 0644); err != nil {
		t.Fatal(err)
	}
	a := &SurfaceReviewAuthority{Locale: "zh-CN", ReviewID: "aggregate-fixture", Stage: "failed", ReceiptIdentity: "fixture-identity", ScopeIdentity: "fixture-scope", Receipt: GlossaryArchiveReference{Path: path, SHA256: sum(data)}, Evidence: GlossaryArchiveReference{Path: path, SHA256: sum(data)}}
	o := QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "aggregate-fixture", PreviousSnapshotID: snapshot, SurfaceReviewID: a.ReviewID, IntegratedSurface: true}
	for i, u := range s.Units {
		finding := fmt.Sprintf("Correct exact defect %d.", i+1)
		a.Findings = append(a.Findings, SurfaceLanguageFinding{ID: u.SourcePath, Finding: finding, Class: SurfaceTranslationUnitLanguage, Artifacts: []SurfaceCandidateArtifact{{SnapshotID: snapshot, Unit: u}}})
		o.Findings = append(o.Findings, finding)
		o.UnitIDs = append(o.UnitIDs, u.UnitID)
	}
	c.SurfaceAuthorityReader = func(string, string, string, bool, *Catalog) (*SurfaceReviewAuthority, error) { return a, nil }
	return a, o
}

func TestSurfaceAggregateReopenBindingAndReadFailClosed(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	glossary, _ := os.ReadFile(filepath.Join(root, "locales/zh-CN/glossary.yaml"))
	writeSnapshotSource(t, root, "locales/zh-CN/glossary.yaml", append([]byte("locale: zh-CN\n"), glossary...))
	recordSyntheticUnifiedReview(t, root, "synthetic-generation-review")
	c := retranslationTestCatalog(2)
	for i := range c.Pages {
		c.Pages[i].Article = fmt.Sprintf("lesson-%d.article", i+1)
		c.Pages[i].SectionNumber = 1
	}
	addProcessedPromotionBatch(t, root, c, "codex-zh-CN-001", []string{"lesson/1", "lesson/2"})
	materializeSnapshotSources(t, root, c)
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-001"}); err != nil {
		t.Fatal(err)
	}
	recordQualityCheckRatings(t, root, c, "qc-001", "", "A", []string{"lesson/1", "lesson/2"})
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-001"}); err != nil {
		t.Fatal(err)
	}
	a, o := aggregateSurfaceAuthorityFixture(t, root, c, "qc-001")
	for _, name := range []string{"duplicate-artifact", "conflicting-findings", "wrong-source", "wrong-snapshot"} {
		t.Run(name, func(t *testing.T) {
			before := a.Findings[1]
			defer func() { a.Findings[1] = before }()
			copy := before
			copy.Artifacts = append([]SurfaceCandidateArtifact(nil), before.Artifacts...)
			switch name {
			case "duplicate-artifact":
				copy.Artifacts = append(copy.Artifacts, copy.Artifacts[0])
			case "conflicting-findings":
				copy.ID, copy.Artifacts = a.Findings[0].ID, a.Findings[0].Artifacts
			case "wrong-source":
				copy.ID = "wrong/path"
			case "wrong-snapshot":
				copy.Artifacts[0].SnapshotID = "other-snapshot"
			}
			a.Findings[1] = copy
			if _, _, err := RecordQualityCheckSurfaceReopen(root, c, o); err == nil {
				t.Fatal("invalid artifact binding accepted")
			}
		})
	}
	r, path, err := RecordQualityCheckSurfaceReopen(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readCurrentQualityCheckSurfaceReopen(root, c, o.Locale, o.ReopenID, true); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"missing-unit-finding", "swapped-unit-findings", "missing-selected-finding", "duplicate-selected-finding", "extra-global-finding", "duplicate-unit", "mixed-kind"} {
		t.Run(name, func(t *testing.T) {
			bad := *r
			bad.Units = append([]QualityCheckSurfaceReopenUnit(nil), r.Units...)
			bad.Findings = append([]string(nil), r.Findings...)
			switch name {
			case "missing-unit-finding":
				bad.Units[1].Finding = ""
			case "swapped-unit-findings":
				bad.Units[0].Finding, bad.Units[1].Finding = bad.Units[1].Finding, bad.Units[0].Finding
			case "missing-selected-finding":
				bad.Findings = bad.Findings[:1]
			case "duplicate-selected-finding":
				bad.Findings[1] = bad.Findings[0]
			case "extra-global-finding":
				bad.Finding = r.Findings[0]
			case "duplicate-unit":
				bad.Units[1] = bad.Units[0]
			case "mixed-kind":
				bad.Units[1].Snapshot.UnitKind = UnitKindExample
			}
			if err := writeTranslationJSON(filepath.Join(root, path), bad); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := writeTranslationJSON(filepath.Join(root, path), r); err != nil {
					t.Fatal(err)
				}
			}()
			if _, err := readCurrentQualityCheckSurfaceReopen(root, c, o.Locale, o.ReopenID); err == nil {
				t.Fatal("tampered reopen accepted")
			}
		})
	}
}

func TestSurfaceAggregateReopenLimit(t *testing.T) {
	ids, findings := []string{}, []string{}
	for i := 0; i < 61; i++ {
		ids = append(ids, fmt.Sprintf("lesson/%d", i+1))
		findings = append(findings, fmt.Sprintf("Exact finding %d", i+1))
	}
	root := t.TempDir()
	o := QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "too-large", PreviousSnapshotID: "qc-001", SurfaceReviewID: "failed", IntegratedSurface: true, Findings: findings, UnitIDs: ids}
	if _, _, err := RecordQualityCheckSurfaceReopen(root, retranslationTestCatalog(61), o); err == nil || !strings.Contains(err.Error(), "must not exceed 60") {
		t.Fatal(err)
	}
	if _, err := os.Stat(qualityCheckSurfaceReopenPath(root, o.Locale, o.ReopenID)); !os.IsNotExist(err) {
		t.Fatal("oversized scope wrote receipt", err)
	}
}

func TestSurfaceAggregateReopenMixedKinds(t *testing.T) {
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
	glossary, _ := os.ReadFile(filepath.Join(root, "locales/zh-CN/glossary.yaml"))
	writeSnapshotSource(t, root, "locales/zh-CN/glossary.yaml", append([]byte("locale: zh-CN\n"), glossary...))
	recordSyntheticUnifiedReview(t, root, "synthetic-generation-review")
	c := retranslationTestCatalog(1)
	c.Examples = retranslationTestExampleCatalog(1).Examples
	ids := []string{c.Pages[0].ID, c.Examples[0].ID}
	addProcessedPromotionBatch(t, root, c, "codex-zh-CN-001", ids[:1])
	addProcessedPromotionBatch(t, root, c, "codex-zh-CN-002", ids[1:])
	materializeSnapshotSources(t, root, c)
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-001"}); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		recordQualityCheckRatings(t, root, c, "qc-001", "", "A", []string{id})
	}
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-001"}); err != nil {
		t.Fatal(err)
	}
	_, o := aggregateSurfaceAuthorityFixture(t, root, c, "qc-001")
	if _, _, err := RecordQualityCheckSurfaceReopen(root, c, o); err == nil || !strings.Contains(err.Error(), "must not mix Page and Example") {
		t.Fatal("mixed scope accepted or wrong rejection", err)
	}
	if _, err := os.Stat(qualityCheckSurfaceReopenPath(root, o.Locale, o.ReopenID)); !os.IsNotExist(err) {
		t.Fatal("mixed scope wrote receipt", err)
	}
}
