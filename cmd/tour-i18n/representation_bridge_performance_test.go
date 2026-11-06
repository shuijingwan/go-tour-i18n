package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Explicit engineering probe only. Normal timing and CPU-profile runs are
// separate processes. Record is refused outside a disposable /tmp repository.
func TestRepresentationBridgePerformance(t *testing.T) {
	root, out := os.Getenv("GO_TOUR_BRIDGE_PERF_ROOT"), os.Getenv("GO_TOUR_BRIDGE_PERF_OUT")
	if root == "" || out == "" {
		t.Skip("explicit isolated performance inputs required")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(out) || !strings.HasPrefix(out, "/tmp/") {
		t.Fatal("absolute root and /tmp evidence required")
	}
	record := os.Getenv("GO_TOUR_BRIDGE_PERF_RECORD") == "1"
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || filepath.Clean(root) != root {
		t.Fatal("canonical existing repository path required", err)
	}
	if record && (!strings.HasPrefix(resolved, "/tmp/") || resolved != root) {
		t.Fatal("record benchmark requires disposable repository")
	}
	ref := os.Getenv("GO_TOUR_BRIDGE_PERF_RECEIPT")
	if ref == "" || filepath.IsAbs(ref) || strings.Contains(ref, "..") {
		t.Fatal("explicit repository-relative existing receipt required")
	}
	b, err := os.ReadFile(filepath.Join(root, ref))
	if err != nil {
		t.Fatal(err)
	}
	var existing i18n.QualityCheckSurfaceReopen
	if err := json.Unmarshal(b, &existing); err != nil || existing.SameUnitBridge == nil {
		t.Fatal("existing exact bridge options required", err)
	}
	// A disposable copy must include every declared current dependency. Catch
	// copy omissions before spending a full graph reconstruction; canonical
	// SHA/currentness verification remains exclusively in the real API below.
	for _, ref := range existing.SameUnitBridge.Dependencies {
		st, err := os.Lstat(filepath.Join(root, filepath.FromSlash(ref.Path)))
		if err != nil || !st.Mode().IsRegular() {
			t.Fatalf("incomplete performance repository copy: %s: %v", ref.Path, err)
		}
	}
	o := existing.SameUnitBridge.Options
	o.BridgeID = os.Getenv("GO_TOUR_BRIDGE_PERF_ID")
	o.Record = record
	if !strings.HasPrefix(o.BridgeID, "perf-") {
		t.Fatal("performance-only bridge identity required")
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	done := contentidentity.BeginMemoization()
	defer done()
	start := time.Now()
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
	if err := attachCommandTranslationPackages(root, c, []string{"quality-check", "representation-bridge"}); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	setup := time.Since(start).Seconds()
	if os.Getenv("GO_TOUR_BRIDGE_PERF_CPU") == "1" {
		finish, err := bridgePerformanceCPU(out)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := finish(); err != nil {
				t.Error(err)
			}
		}()
	}
	start = time.Now()
	cpu := bundleProofCPU()
	plan, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o)
	elapsed, elapsedCPU := time.Since(start).Seconds(), bundleProofCPU()-cpu
	if err != nil {
		t.Fatal(err)
	}
	receipt, _ := json.Marshal(plan.Receipt)
	if err := os.WriteFile(filepath.Join(out, "planned-receipt.json"), receipt, 0600); err != nil {
		t.Fatal(err)
	}
	metrics := map[string]any{"wall_seconds": elapsed, "cpu_seconds": elapsedCPU, "catalog_setup_seconds": setup, "record": record, "receipt_sha256": plan.ReceiptSHA256, "bridge_identity": plan.Receipt.SameUnitBridge.Identity, "package_identity": plan.Receipt.SameUnitBridge.PackageIdentity, "units": plan.Receipt.UnitCount, "recorded": plan.Recorded, "mutation": plan.Mutation}
	data, _ := json.MarshalIndent(metrics, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "timing.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if record != plan.Recorded {
		t.Fatal("publication mode changed")
	}
	after, err := os.ReadFile(filepath.Join(root, ref))
	if err != nil || string(b) != string(after) {
		t.Fatal("existing receipt changed", err)
	}
	t.Logf("BRIDGE_WALL=%.3fs CPU=%.3fs SETUP=%.3fs RECORD=%t IDENTITY=%s", elapsed, elapsedCPU, setup, record, metrics["bridge_identity"])
}

// Complete one-minute segments allow inspection during a long readonly run;
// merging every segment yields the full CPU profile, not a truncated sample.
func bridgePerformanceCPU(out string) (func() error, error) {
	index := 1
	open := func() (*os.File, error) {
		f, err := os.Create(filepath.Join(out, fmt.Sprintf("cpu-%03d.pprof", index)))
		if err != nil {
			return nil, err
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			return nil, err
		}
		return f, nil
	}
	f, err := open()
	if err != nil {
		return nil, err
	}
	stop, done := make(chan struct{}), make(chan error, 1)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				pprof.StopCPUProfile()
				done <- f.Close()
				return
			case <-ticker.C:
				pprof.StopCPUProfile()
				if err := f.Close(); err != nil {
					done <- err
					return
				}
				index++
				var err error
				f, err = open()
				if err != nil {
					done <- err
					return
				}
			}
		}
	}()
	return func() error { close(stop); return <-done }, nil
}
