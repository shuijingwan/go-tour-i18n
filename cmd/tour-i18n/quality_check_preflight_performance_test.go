package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Opt-in readonly performance probe, using the existing reviewed transport's
// exact Snapshot/predecessor/package coordinates. Run the compiled test from
// the repository root; no lifecycle output or new bundle is produced.
func TestQualityCheckPreflightExistingPerformance(t *testing.T) {
	path := os.Getenv("GO_TOUR_REVIEWER_CURRENT_BUNDLE")
	if path == "" {
		t.Skip("requires GO_TOUR_REVIEWER_CURRENT_BUNDLE")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(before, 512, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.QualityCheckReviewerBundleManifest
	var snapshot i18n.QualityCheckSnapshotManifest
	var context i18n.QualityCheckReviewerContext
	for file, dst := range map[string]any{"manifest.json": &m, "formal/snapshot-manifest.json": &snapshot, "quality-check.json": &context} {
		if err := json.Unmarshal(files[file], dst); err != nil {
			t.Fatal(file, err)
		}
	}
	if snapshot.ContentPackage == nil {
		t.Fatal("requires explicit registered package Snapshot")
	}
	started, cpu := time.Now(), bundleProofCPU()
	// A full pending scope may exceed a pipe buffer. Use disposable file
	// capture so this profiling-only harness also works for future full scopes.
	outputFile, err := os.CreateTemp(t.TempDir(), "preflight-*.json")
	if err != nil {
		t.Fatal(err)
	}
	savedStdout := os.Stdout
	func() {
		os.Stdout = outputFile
		defer func() { os.Stdout = savedStdout }()
		err = run([]string{"quality-check", "preflight", "--locale", m.Locale, "--snapshot-id", m.SnapshotID, "--previous-snapshot-id", m.PreviousSnapshotID, "--content-package", snapshot.ContentPackage.ID})
	}()
	closeErr := outputFile.Close()
	if err != nil || closeErr != nil {
		t.Fatal(err, closeErr)
	}
	output, err := os.ReadFile(outputFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("CANONICAL_PREFLIGHT_WALL=%s CPU_SECONDS=%.3f", time.Since(started), bundleProofCPU()-cpu)
	var report i18n.QualityCheckPreflight
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatal(err)
	}
	if report.CarryForwardCount != context.Scope.CarryForwardCount || report.QualityCheckRequiredCount != context.Scope.PendingCount || report.RevisionRequiredCount != 0 || report.SelectedPreviousSnapshot != m.PreviousSnapshotID {
		t.Fatal("preflight scope changed", report)
	}
	if len(report.Pending) != len(context.Scope.Pending) {
		t.Fatal("pending exact set changed")
	}
	for i, u := range report.Pending {
		old := context.Scope.Pending[i]
		if u.UnitID != old.UnitID || u.Index != old.Index || u.BatchID != old.BatchID || u.Reason != old.Reason || u.RequiredAction != i18n.QualityCheckActionRequired {
			t.Fatal("pending identity/action changed", u)
		}
	}
	t.Logf("CURRENT_UNITS=%d CARRY_FORWARD=%d PENDING=%d PENDING_EXACT_SET=yes", snapshot.UnitCount, report.CarryForwardCount, len(report.Pending))
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("existing bundle changed", err)
	}
}

// A bounded CPU sample of the real shared evaluator avoids another complete
// baseline run. A timed-out sample is explicitly not a preflight PASS. It must
// run alone in its compiled test process, which exits after stopping the CPU
// profile; the unfinished operation has no write paths or lifecycle output.
func TestQualityCheckPreflightExistingCPUSample(t *testing.T) {
	path, out := os.Getenv("GO_TOUR_REVIEWER_CURRENT_BUNDLE"), os.Getenv("GO_TOUR_QC_PROFILE_DIR")
	if path == "" || out == "" {
		t.Skip("requires existing Bundle and disposable profile directory")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(b, 512, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.QualityCheckReviewerBundleManifest
	var s i18n.QualityCheckSnapshotManifest
	if json.Unmarshal(files["manifest.json"], &m) != nil || json.Unmarshal(files["formal/snapshot-manifest.json"], &s) != nil || s.ContentPackage == nil {
		t.Fatal("invalid profiling coordinates")
	}
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	doneMemo := contentidentity.BeginMemoization()
	defer doneMemo()
	setup := time.Now()
	source, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	c, err := i18n.ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := i18n.HydrateCatalogSources(c, source); err != nil {
		t.Fatal(err)
	}
	if err := attachCommandTranslationPackages(root, c, []string{"quality-check", "preflight"}); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	if err := c.UseTranslationPackage(s.ContentPackage.ID); err != nil {
		t.Fatal(err)
	}
	t.Logf("CATALOG_SETUP_WALL=%s", time.Since(setup))
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(out, "preflight-before.cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	start, cpu := time.Now(), bundleProofCPU()
	finished := make(chan error, 1)
	go func() {
		_, err := i18n.BuildQualityCheckPreflight(root, c, i18n.QualityCheckPreflightOptions{Locale: m.Locale, SnapshotID: m.SnapshotID, PreviousSnapshotID: m.PreviousSnapshotID})
		finished <- err
	}()
	complete := false
	select {
	case err := <-finished:
		if err != nil {
			t.Error(err)
		}
		complete = true
	case <-time.After(60 * time.Second):
	}
	pprof.StopCPUProfile()
	f.Close()
	t.Logf("PREFLIGHT_PROFILE_SAMPLE_WALL=%s CPU_SECONDS=%.3f FULL_PREFLIGHT_COMPLETED=%t", time.Since(start), bundleProofCPU()-cpu, complete)
}
