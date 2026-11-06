package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The shared lifecycle treats proof as a representation reader, exactly as
// Surface readers. Actual Markdown/HTML proof is tested in internal/site.
func previewFailureFixture(t *testing.T) (string, *Catalog, *PreviewTUFailure) {
	t.Helper()
	root := t.TempDir()
	writeRetranslationTestGlossary(t, root)
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
	p, err := PreparePreviewTUFailure(root, c, "zh-CN", "qc-001", "lesson/1")
	if err != nil {
		t.Fatal(err)
	}
	p.FailureID = "post-pass-proof"
	p.Package = "synthetic-package"
	p.PackageSHA256 = sum([]byte("package"))
	p.Route = "/fixture/page"
	p.PreviewURL = "http://127.0.0.1:3999/"
	p.ContextSHA256 = sum([]byte("current-build"))
	p.ResponseHTML = "<main>synthetic broken link observation</main>"
	p.ResponseSHA256 = sum([]byte(p.ResponseHTML))
	p.MissingLinks = []string{"https://example.invalid/required"}
	c.PreviewFailureReader = func(_ string, _ *Catalog, proof *PreviewTUFailure, current bool) error {
		if !reflect.DeepEqual(proof.MissingLinks, []string{"https://example.invalid/required"}) {
			return fmt.Errorf("unsupported exact proof")
		}
		if current && proof.ContextSHA256 != sum([]byte("current-build")) {
			return fmt.Errorf("stale build")
		}
		return nil
	}
	return root, c, p
}

func TestPreviewFailureSharedReopenRevisionAndIndependentQC(t *testing.T) {
	root, c, p := previewFailureFixture(t)
	// A later machine failure must not rewrite the earlier independent PASS.
	pass := "data/locale-language/zh-CN/surface/old-pass.json"
	writeSnapshotSource(t, root, pass, []byte(`{"decision":"passed","findings":[]}`))
	beforePass, _ := os.ReadFile(filepath.Join(root, pass))
	beforeQC, _ := os.ReadFile(qualityCheckResultsPath(root, "zh-CN", "qc-001"))
	beforeFinal, _ := os.ReadFile(qualityCheckFinalizationPath(root, "zh-CN", "qc-001"))
	path, err := RecordPreviewTUFailure(root, c, *p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecordPreviewTUFailure(root, c, *p); err == nil {
		t.Fatal("immutable failure overwritten")
	}
	b, _ := os.ReadFile(filepath.Join(root, path))
	ref := GlossaryArchiveReference{Path: path, SHA256: sum(b)}
	o := QualityCheckSurfaceReopenOptions{Locale: p.Locale, ReopenID: "preview-fix", PreviousSnapshotID: "qc-001", UnitIDs: []string{"lesson/1"}, PreviewFailure: ref}
	r, _, err := RecordQualityCheckSurfaceReopen(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	if r.UnitCount != 1 || r.IntegratedAuthority != nil || r.SurfaceReviewID != "" || r.revisionFeedbackSource() != PreviewRevisionFeedback {
		t.Fatal("machine proof became a Surface judgment", r)
	}
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: o.UnitIDs, AllowReexport: true, PreviousSnapshotID: "qc-001", SurfaceReopenID: o.ReopenID})
	if err != nil {
		t.Fatal(err)
	}
	m := readRetranslationManifest(t, root, e.BatchID)
	if m.Units[0].RevisionFeedbackSource != PreviewRevisionFeedback || m.Units[0].PreviousFinding != p.Finding() || m.Units[0].PreviousReviewPath != path {
		t.Fatal("machine provenance lost", m.Units[0])
	}
	// Frozen revision/retry does not need the old preview process to stay alive.
	c.PreviewFailureReader = func(_ string, _ *Catalog, _ *PreviewTUFailure, current bool) error {
		if current {
			return fmt.Errorf("preview stopped after export")
		}
		return nil
	}
	stale := o
	stale.ReopenID = "stale-second-reopen"
	if _, _, err := RecordQualityCheckSurfaceReopen(root, c, stale); err == nil {
		t.Fatal("stale preview authorized a second reopen")
	}
	if _, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: o.UnitIDs, AllowReexport: true, PreviousSnapshotID: "qc-001", SurfaceReopenID: o.ReopenID}); err == nil {
		t.Fatal("stale preview authorized another export")
	}
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	copyBundleAuthority(t, root, []string{"docs/PREVIEW_TU_FAILURE.md"})
	exerciseRevisionRetry(t, root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID}, "restore_failed")
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "preview-successor"}); err != nil {
		t.Fatal(err)
	}
	scope, err := BuildQualityCheckScope(root, c, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "preview-successor", PreviousSnapshotID: "qc-001"})
	if err != nil || scope.PendingCount != 1 || scope.ACount != 1 {
		t.Fatalf("successor must re-QC exactly changed TU: %+v %v", scope, err)
	}
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "preview-successor", PreviousSnapshotID: "qc-001"}); err == nil {
		t.Fatal("machine reopen granted successor A")
	}
	copyBundleAuthority(t, root, qualityCheckReviewerAuthorityPaths)
	if _, _, err := ExportQualityCheckReviewerBundle(root, c, QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: "preview-successor", PreviousSnapshotID: "qc-001"}); err != nil {
		t.Fatal(err)
	}
	recordQualityCheckRatings(t, root, c, "preview-successor", "qc-001", "A", []string{"lesson/1"})
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "preview-successor", PreviousSnapshotID: "qc-001"}); err != nil {
		t.Fatal("independently reviewed successor failed finalization", err)
	}
	for name, want := range map[string][]byte{pass: beforePass, filepath.ToSlash(qualityCheckResultsPath("", "zh-CN", "qc-001")): beforeQC, filepath.ToSlash(qualityCheckFinalizationPath("", "zh-CN", "qc-001")): beforeFinal} {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("historical evidence changed", name, err)
		}
	}
}

func TestPreviewFailureSharedFailClosed(t *testing.T) {
	root, c, p := previewFailureFixture(t)
	for _, name := range []string{"candidate", "source", "validation", "snapshot", "non-tu", "stale-build", "arbitrary-text"} {
		t.Run(name, func(t *testing.T) {
			bad := *p
			switch name {
			case "candidate":
				bad.Artifact.Unit.CandidateSHA256 = sum([]byte("other"))
			case "source":
				bad.Artifact.Unit.SourceSHA256 = sum([]byte("other"))
			case "validation":
				bad.Artifact.Unit.ValidationSHA256 = sum([]byte("other"))
			case "snapshot":
				bad.Artifact.SnapshotID = "other"
			case "non-tu":
				bad.Check = "runtime-exception"
			case "stale-build":
				bad.ContextSHA256 = sum([]byte("old-build"))
			case "arbitrary-text":
				bad.MissingLinks = []string{"Please fix this sentence"}
			}
			if _, err := RecordPreviewTUFailure(root, c, bad); err == nil {
				t.Fatal("unproven machine failure accepted")
			}
		})
	}
	path, err := RecordPreviewTUFailure(root, c, *p)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, path))
	ref := GlossaryArchiveReference{Path: path, SHA256: sum(b)}
	base := QualityCheckSurfaceReopenOptions{Locale: p.Locale, ReopenID: "preview-fix", PreviousSnapshotID: "qc-001", UnitIDs: []string{"lesson/1"}, PreviewFailure: ref}
	for _, name := range []string{"finding", "surface", "replacement", "wrong-unit", "extra-unit", "predecessor", "locale"} {
		t.Run(name, func(t *testing.T) {
			o := base
			switch name {
			case "finding":
				o.Finding = "pretend Reviewer finding"
			case "surface":
				o.SurfaceReviewID = "old-pass"
			case "replacement":
				o.HistoricalDocuments.Path = "frozen.json"
			case "wrong-unit":
				o.UnitIDs = []string{"lesson/2"}
			case "extra-unit":
				o.UnitIDs = []string{"lesson/1", "lesson/2"}
			case "predecessor":
				o.PreviousSnapshotID = "different"
			case "locale":
				o.Locale = "fr-FR"
			}
			if _, _, err := RecordQualityCheckSurfaceReopen(root, c, o); err == nil {
				t.Fatal("invalid reopen scope/authority accepted")
			}
		})
	}
	writeSnapshotSource(t, root, "data/preview-tu-failures/zh-CN/plain.json", []byte("a user says preview failed"))
	base.PreviewFailure = GlossaryArchiveReference{Path: "data/preview-tu-failures/zh-CN/plain.json", SHA256: sum([]byte("a user says preview failed"))}
	if _, _, err := RecordQualityCheckSurfaceReopen(root, c, base); err == nil {
		t.Fatal("free text accepted as machine authority")
	}
}
