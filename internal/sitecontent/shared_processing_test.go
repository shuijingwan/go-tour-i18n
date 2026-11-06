package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func sharedPageFixture(t *testing.T, count int) (string, *i18n.Catalog, []string) {
	t.Helper()
	root := siteFixture(t, "docs/RETRANSLATION_RUNBOOK.md", "docs/TRANSLATION_QUALITY_REVIEW.md", "current-authorization-fresh-fixture")
	c := &i18n.Catalog{}
	descriptor := i18n.TranslationPackageDescriptor{ID: "synthetic-shared-pages", AuthoritySHA256: digest([]byte(fmt.Sprintf("synthetic-shared-pages/%d", count)))}
	ids := []string{}
	for i := 1; i <= count; i++ {
		p := fmt.Sprintf("_content/doc/tutorial/synthetic-%02d.md", i)
		raw := []byte("# Synthetic page\n\nUse `Go` to learn.\n")
		d, err := ParseWorkflowDocument(Source{Path: p, Kind: "page", SHA256: digest(raw), Surface: "tutorial", Route: fmt.Sprintf("/doc/tutorial/synthetic-%02d", i)}, raw, i)
		if err != nil {
			t.Fatal(err)
		}
		u := d.Units[0]
		writeFixture(t, root, p, raw)
		descriptor.Units = append(descriptor.Units, i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: u.ID, Kind: i18n.UnitKindPage, SourcePath: p, Source: raw, SourceSHA256: d.SourceSHA}, Adapter: PageAdapter{*d, raw}, StableIndex: i, Route: d.Route, WorkingSet: fmt.Sprintf("batch-%d", (i-1)/30+1), WorkingSetLimit: 30})
		ids = append(ids, u.ID)
	}
	if err := c.RegisterTranslationPackage(descriptor); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationUnitScope(ids); err != nil {
		t.Fatal(err)
	}
	return root, c, ids
}

func importSyntheticSharedPages(t *testing.T, root string, c *i18n.Catalog, ids []string, broken ...int) (string, map[string][]byte) {
	t.Helper()
	return importSyntheticSharedPagesForLocale(t, root, c, "zh-CN", ids, broken...)
}

func importSyntheticSharedPagesForLocale(t *testing.T, root string, c *i18n.Catalog, locale string, ids []string, broken ...int) (string, map[string][]byte) {
	t.Helper()
	exported, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	bundle, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: locale, BatchID: exported.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	good := map[string][]byte{}
	for i, e := range m.ExpectedOutputs {
		b, err := os.ReadFile(filepath.Join(root, exported.BatchPath, "inputs", filepath.Base(e.BundlePath)))
		if err != nil {
			t.Fatal(err)
		}
		good[e.UnitID] = bytes.Clone(b)
		for _, index := range broken {
			if i == index {
				b = bytes.Replace(b, []byte("⟦P0⟧"), []byte("⟦BROKEN⟧"), 1)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(e.BundlePath)), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := i18n.ImportGenerationOutputDirectory(root, c, bundle, "chatgpt", "gpt-5.6-sol-high", dir); err != nil {
		t.Fatal("transport must retain semantic failures", err)
	}
	return exported.BatchID, good
}

func TestSharedPageProcessingPreserves29AndRetriesExactlyOne(t *testing.T) {
	root, c, ids := sharedPageFixture(t, 51)
	if _, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{ids[0], ids[30]}}); err == nil {
		t.Fatal("explicit initial export combined fixed groups")
	}
	batch, good := importSyntheticSharedPages(t, root, c, ids[:30], 29)
	r, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil {
		t.Fatal(err)
	}
	if r.ValidationPassed != 29 || r.RestoreFailed != 1 || r.ValidationFailed != 0 {
		t.Fatalf("29/1: %+v", r)
	}
	if _, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", UnitIDs: ids[30:]}); err == nil {
		t.Fatal("second Generation allowed before first batch valid")
	}
	failed := r.Units[29].UnitID
	retained := map[string][]byte{}
	for _, u := range r.Units[:29] {
		b, err := os.ReadFile(filepath.Join(root, "data/retranslation-runs/zh-CN", batch, u.CandidatePath))
		if err != nil {
			t.Fatal(err)
		}
		retained[u.CandidatePath] = b
	}
	bundle, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: batch, UnitID: failed})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.ExpectedOutputs) != 1 || m.ExpectedOutputs[0].UnitID != failed || m.Attempt != 2 {
		t.Fatal("retry not exact")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, filepath.Base(m.ExpectedOutputs[0].BundlePath)), good[failed], 0600); err != nil {
		t.Fatal(err)
	}
	recovery, err := i18n.RecoverGenerationOutputs(root, c, bundle, []string{dir}, filepath.Join(t.TempDir(), "recovered-page"))
	if err != nil || len(recovery.Recovered) != 1 || len(recovery.Missing) != 0 {
		t.Fatalf("shared Page recovery: %+v %v", recovery, err)
	}
	if _, err := i18n.ImportGenerationOutputDirectory(root, c, bundle, "chatgpt", "gpt-5.6-sol-high", recovery.OutputDir); err != nil {
		t.Fatal(err)
	}
	r, err = i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: "zh-CN", BatchID: batch, UnitID: failed})
	if err != nil {
		t.Fatal(err)
	}
	if r.ValidationPassed != 30 || r.RestoreFailed+r.ValidationFailed != 0 {
		t.Fatal("retry did not converge", r)
	}
	for p, b := range retained {
		now, _ := os.ReadFile(filepath.Join(root, "data/retranslation-runs/zh-CN", batch, p))
		if !bytes.Equal(now, b) {
			t.Fatal("successful Unit changed", p)
		}
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "synthetic-before-batch2"}); err == nil {
		t.Fatal("QC snapshot allowed before batch2")
	}
	second, _ := importSyntheticSharedPages(t, root, c, ids[30:])
	if _, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: second}); err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "synthetic-full"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.UnitCount != 51 {
		t.Fatal("incomplete snapshot")
	}
	scope, err := i18n.BuildQualityCheckScope(root, c, i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "synthetic-full"})
	if err != nil {
		t.Fatal(err)
	}
	if scope.PendingCount != 51 || scope.ACount != 0 {
		t.Fatal("machine PASS became reviewer A")
	}
	if err := i18n.CheckUnifiedLanguageRole(root, "zh-CN", "synthetic-generation", "reviewer"); err == nil {
		t.Fatal("generation accepted as reviewer")
	}
	_, mReview, err := i18n.ExportQualityCheckReviewerBundle(root, c, i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: "synthetic-full"})
	if err != nil {
		t.Fatal(err)
	}
	if mReview.UnitCount != 30 {
		t.Fatal("review combined fixed batches")
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "synthetic-independent-reviewer", Locale: "zh-CN", SnapshotID: "synthetic-full", UnitIDs: []string{ids[0], ids[30]}, Rating: "A"}); err == nil {
		t.Fatal("initial record combined fixed groups")
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "synthetic-independent-reviewer", Locale: "zh-CN", SnapshotID: "synthetic-full", UnitIDs: ids[:2], Rating: "B", Finding: "synthetic fixture finding"}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "synthetic-independent-reviewer", Locale: "zh-CN", SnapshotID: "synthetic-full", UnitIDs: ids[2:30], Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	options := i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: ids[:1], AllowReexport: true, PreviousSnapshotID: "synthetic-full"}
	if _, err := i18n.ExportRetranslationBatch(root, c, options); err == nil || !strings.Contains(err.Error(), "全部初审") {
		t.Fatal("revision before second initial QC", err)
	}
	if _, m, err := i18n.ExportQualityCheckReviewerBundle(root, c, i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: "synthetic-full"}); err != nil || m.UnitCount != 21 {
		t.Fatal("second initial QC must contain exactly 21 Page", err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "synthetic-independent-reviewer", Locale: "zh-CN", SnapshotID: "synthetic-full", UnitIDs: ids[30:32], Rating: "C", Finding: "second group finding"}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "synthetic-independent-reviewer", Locale: "zh-CN", SnapshotID: "synthetic-full", UnitIDs: ids[32:], Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	options.UnitIDs = []string{ids[0], ids[30], ids[2]}
	if _, err := i18n.ExportRetranslationBatch(root, c, options); err == nil || !strings.Contains(err.Error(), "already A") {
		t.Fatal("revision included unaffected A", err)
	}
	options.UnitIDs = []string{ids[0], ids[1], ids[30], ids[31]}
	revision, err := i18n.ExportRetranslationBatch(root, c, options)
	if err != nil {
		t.Fatal("concentrated exact revision", err)
	}
	revisionBundle, revisionManifest, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: revision.BatchID})
	if err != nil || len(revisionManifest.ExpectedOutputs) != 4 {
		t.Fatal("revision scope", err)
	}
	revisionDir := t.TempDir()
	for _, output := range revisionManifest.ExpectedOutputs {
		// Each output must retain its own Page boundary/protected identity.
		unit, err := c.Unit(output.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		input, _, err := unit.Adapter.ProtectedInput(nil)
		if err != nil {
			t.Fatal(err)
		}
		replacement := append(bytes.Replace([]byte(input), []byte("Use "), []byte("Try "), 1), '\n')
		if err := os.WriteFile(filepath.Join(revisionDir, filepath.Base(output.BundlePath)), replacement, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := i18n.ImportGenerationOutputDirectory(root, c, revisionBundle, "chatgpt", "gpt-5.6-sol-high", revisionDir); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: revision.BatchID}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "synthetic-revised"}); err != nil {
		t.Fatal(err)
	}
	reQC, err := i18n.BuildQualityCheckScope(root, c, i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "synthetic-revised", PreviousSnapshotID: "synthetic-full"})
	if err != nil || reQC.PendingCount != 4 || len(reQC.CarryForward) != 47 {
		t.Fatalf("exact re-QC with 47 carried A: %+v %v", reQC, err)
	}
	for i, pending := range reQC.Pending {
		if pending.Index != []int{1, 2, 31, 32}[i] || pending.RequiredAction != i18n.QualityCheckActionRequired {
			t.Fatal("revised scope is not stable quality_check_required", pending)
		}
	}
	reviewOptions := i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: "synthetic-revised", PreviousSnapshotID: "synthetic-full"}
	_, reviewManifest, err := i18n.ExportQualityCheckReviewerBundle(root, c, reviewOptions)
	if err != nil || reviewManifest.UnitCount != 4 {
		t.Fatal("re-QC split initial groups", reviewManifest, err)
	}
	// Deliberately pre-export a future set in the fixture to prove JIT rejection.
	reviewOptions.StartIndex = 2
	future, _, err := i18n.ExportQualityCheckReviewerBundle(root, c, reviewOptions)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.VerifyCurrentQualityCheckReviewerBundle(root, c, future); err != nil {
		t.Fatal("fixture future Bundle was not initially current", err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "synthetic-independent-reviewer", Locale: "zh-CN", SnapshotID: "synthetic-revised", PreviousSnapshotID: "synthetic-full", UnitIDs: options.UnitIDs, Rating: "B"}); err == nil {
		t.Fatal("re-QC accepted non-A without finding")
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "synthetic-independent-reviewer", Locale: "zh-CN", SnapshotID: "synthetic-revised", PreviousSnapshotID: "synthetic-full", UnitIDs: []string{ids[0], ids[30]}, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.VerifyCurrentQualityCheckReviewerBundle(root, c, future); err == nil {
		t.Fatal("future re-QC Bundle remained current after record")
	}
	// Omitted CLI predecessor must use the persisted revision lineage too.
	if _, m, err := i18n.ExportQualityCheckReviewerBundle(root, c, i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: "synthetic-revised"}); err != nil || m.UnitCount != 2 {
		t.Fatal("resumed re-QC inherited initial partition", m, err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "synthetic-independent-reviewer", Locale: "zh-CN", SnapshotID: "synthetic-revised", UnitIDs: []string{ids[1], ids[31]}, Rating: "A"}); err != nil {
		t.Fatal("persisted re-QC lineage rejected cross-group record", err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "synthetic-revised", PreviousSnapshotID: "synthetic-full"}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.VerifyQualityCheckFinalization(root, c, "zh-CN", "synthetic-revised"); err != nil {
		t.Fatal(err)
	}
	finding := "Correct " + ids[0] + " and " + ids[30]
	options.UnitIDs = []string{ids[0], ids[30]}
	writeFixture(t, root, "data/locale-surface-reviews/zh-CN/synthetic-surface.md", []byte(finding+"\n"))
	if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "synthetic-reopen", PreviousSnapshotID: "synthetic-revised", SurfaceReviewID: "synthetic-surface", Finding: finding, UnitIDs: options.UnitIDs}); err != nil {
		t.Fatal(err)
	}
	options.PreviousSnapshotID, options.SurfaceReopenID = "synthetic-revised", "synthetic-reopen"
	if _, err := i18n.ExportRetranslationBatch(root, c, options); err != nil {
		t.Fatal("cross-group Surface reopen revision", err)
	}
}

func TestSharedPageInitialAutomaticWorkingSet(t *testing.T) {
	root, c, ids := sharedPageFixture(t, 51)
	r, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN"})
	if err != nil || r.UnitCount != 30 || r.UnitIDs[29] != ids[29] {
		t.Fatal("automatic initial export crossed fixed group", r, err)
	}
	// Glossary-stale is a separate recovery path and keeps its fixed partition.
	if _, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", AllowReexport: true, GlossaryStale: true, UnitIDs: []string{ids[0], ids[30]}}); err == nil || !strings.Contains(err.Error(), "不得合并固定") {
		t.Fatal("glossary-stale partition changed", err)
	}
}

func TestSharedPageProcessingReportsAllIndependentFailures(t *testing.T) {
	root, c, ids := sharedPageFixture(t, 30)
	batch, _ := importSyntheticSharedPages(t, root, c, ids, 0, 15, 29)
	r, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Units) != 30 || r.RestoreFailed != 3 || r.ValidationPassed != 27 {
		t.Fatal("batch stopped at first failure", r)
	}
	for _, i := range []int{0, 15, 29} {
		if r.Units[i].Status != "restore_failed" || r.Units[i].Error == "" {
			t.Fatal("missing exact failure", i)
		}
	}
}

func TestSharedPageAOnlyFinalizationUsesExistingEvidence(t *testing.T) {
	root, c, ids := sharedPageFixture(t, 30)
	batch, _ := importSyntheticSharedPages(t, root, c, ids)
	if _, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "synthetic-a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "synthetic-a", ReviewerSession: "synthetic-generation", UnitIDs: ids, Rating: "A"}); err == nil {
		t.Fatal("generation session accepted as independent reviewer")
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "synthetic-a", ReviewerSession: "synthetic-independent-reviewer", UnitIDs: ids, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "synthetic-a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.VerifyQualityCheckFinalization(root, c, "zh-CN", "synthetic-a"); err != nil {
		t.Fatal(err)
	}
}

func TestSharedPageRetryBundlesRemainCurrentAcrossSiblingSuccess(t *testing.T) {
	root, c, ids := sharedPageFixture(t, 3)
	batch, good := importSyntheticSharedPages(t, root, c, ids, 0, 1, 2)
	r, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil || r.RestoreFailed != 3 {
		t.Fatalf("process=%+v err=%v", r, err)
	}
	bundles := [][]byte{}
	manifests := []i18n.GenerationBundleManifest{}
	for _, id := range ids {
		b, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: batch, UnitID: id})
		if err != nil {
			t.Fatal(err)
		}
		bundles = append(bundles, b)
		manifests = append(manifests, m)
	}
	// Process in a different order from export.
	for step, index := range []int{2, 0, 1} {
		if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundles[index]); err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		name := filepath.Base(manifests[index].ExpectedOutputs[0].BundlePath)
		if err := os.WriteFile(filepath.Join(dir, name), good[ids[index]], 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := i18n.ImportGenerationOutputDirectory(root, c, bundles[index], "chatgpt", "gpt-5.6-sol-high", dir); err != nil {
			t.Fatal(err)
		}
		r, err = i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: "zh-CN", BatchID: batch, UnitID: ids[index]})
		if err != nil || r.ValidationPassed != step+1 || r.RestoreFailed != 2-step || r.ValidationFailed != 0 {
			t.Fatalf("retry=%+v err=%v", r, err)
		}
	}
}
