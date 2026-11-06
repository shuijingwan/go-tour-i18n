package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Engineering-only gates. No production flag, schema or command contract is
// changed. All writers require an explicitly selected disposable repository.
func sharedWorkflowPerformanceGate(seconds float64) string {
	if seconds > 600 {
		return "FAILED"
	}
	if seconds > 540 {
		return "PASS_WITH_LOW_MARGIN"
	}
	return "PASS"
}

func TestSharedWorkflowPerformanceGate(t *testing.T) {
	for _, c := range []struct {
		seconds float64
		want    string
	}{
		{0, "PASS"}, {540, "PASS"}, {540.01, "PASS_WITH_LOW_MARGIN"},
		{600, "PASS_WITH_LOW_MARGIN"}, {600.01, "FAILED"},
	} {
		if got := sharedWorkflowPerformanceGate(c.seconds); got != c.want {
			t.Fatal(c, got)
		}
	}
}

func TestSharedWorkflowEnvironmentPerformance(t *testing.T) {
	root, out, stage := os.Getenv("GO_TOUR_SHARED_PERF_ROOT"), os.Getenv("GO_TOUR_SHARED_PERF_OUT"), os.Getenv("GO_TOUR_SHARED_PERF_STAGE")
	if root == "" || out == "" || stage == "" {
		t.Skip("explicit disposable engineering inputs required")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || root != resolved || !strings.HasPrefix(root, "/tmp/") || !filepath.IsAbs(out) || !strings.HasPrefix(out, "/tmp/") {
		t.Fatal("canonical disposable repository and /tmp evidence required", err)
	}
	rel := os.Getenv("GO_TOUR_SHARED_PERF_MANIFEST")
	if filepath.IsAbs(rel) || strings.Contains(rel, "..") || !strings.HasSuffix(rel, "/manifest.json") {
		t.Fatal("explicit relative batch manifest required")
	}
	before, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	var batch i18n.RetranslationBatchManifest
	if json.Unmarshal(before, &batch) != nil || len(batch.Units) != 1 || batch.Units[0].RepresentationBridge == nil {
		t.Fatal("one recorded-bridge revision required")
	}
	ref := *batch.Units[0].RepresentationBridge
	rb, err := os.ReadFile(filepath.Join(root, ref.Path))
	if err != nil || contentidentity.SHA256Hex(rb) != ref.SHA256 {
		t.Fatal("exact bridge input required", err)
	}
	var receipt i18n.QualityCheckSurfaceReopen
	if json.Unmarshal(rb, &receipt) != nil || receipt.SameUnitBridge == nil {
		t.Fatal("recorded bridge required")
	}
	command := map[string][]string{
		"retranslation-export": {"retranslation", "export"},
		"bundle-export":        {"generation-bundle", "export"},
		"bundle-current-check": {"generation-bundle", "current-check"},
		"bridge-readonly":      {"quality-check", "representation-bridge"},
		"bridge-record":        {"quality-check", "representation-bridge"},
	}[stage]
	if command == nil {
		t.Fatal("unknown engineering stage", stage)
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	resolvedOut, err := filepath.EvalSymlinks(out)
	if err != nil || resolvedOut != out {
		t.Fatal("canonical /tmp evidence directory required", err)
	}
	invocationStart := time.Now()
	// An explicit engineering probe must not consume another 20–30 minutes
	// after its hard gate has already failed. Only disposable roots are admitted
	// above; this deadline is never installed by a production CLI invocation.
	deadline := time.AfterFunc(600*time.Second, func() {
		b, _ := json.MarshalIndent(map[string]any{"stage": stage, "seconds": time.Since(invocationStart).Seconds(), "completed": false, "gate": "FAILED", "error": "600s performance budget exhausted"}, "", "  ")
		_ = os.WriteFile(filepath.Join(out, stage+".json"), append(b, '\n'), 0600)
		os.Exit(1)
	})
	defer deadline.Stop()
	finish := contentidentity.BeginMemoization()
	defer finish()
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
	if err := attachCommandTranslationPackages(root, c, command); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(receipt.SameUnitBridge.Options.ContentPackage); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	setup := time.Since(start).Seconds()
	start = time.Now()
	cpu := bundleProofCPU()
	var payload []byte
	var identity string
	switch stage {
	case "retranslation-export":
		// Delete only this disposable pending copy; reproduce its exact manifest.
		if err := os.RemoveAll(filepath.Join(root, filepath.Dir(rel))); err != nil {
			t.Fatal(err)
		}
		var result *i18n.RetranslationExportResult
		result, err = i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{
			Locale: batch.Locale, BatchID: batch.BatchID, Generator: i18n.RetranslationGeneratorChatGPT,
			UnitKind: batch.UnitKind, UnitIDs: []string{batch.Units[0].UnitID}, AllowReexport: true,
			PreviousSnapshotID: batch.Units[0].PreviousSnapshotID, RepresentationBridgeID: receipt.ReopenID,
		})
		if err == nil && (result.UnitCount != 1 || result.BatchID != batch.BatchID) {
			t.Fatal("exact revision scope changed")
		}
		if err == nil {
			payload, err = os.ReadFile(filepath.Join(root, rel))
			if err == nil && !bytes.Equal(payload, before) {
				t.Fatal("formal manifest bytes changed")
			}
		}
	case "bundle-export":
		var m i18n.GenerationBundleManifest
		payload, m, err = i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: batch.Locale, BatchID: batch.BatchID})
		identity = m.InputIdentitySHA256
		if err == nil {
			err = os.WriteFile(filepath.Join(out, "disposable-generation.zip"), payload, 0600)
		}
	case "bundle-current-check":
		payload, err = os.ReadFile(filepath.Join(out, "disposable-generation.zip"))
		if err == nil {
			var m i18n.GenerationBundleManifest
			m, err = i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, payload)
			identity = m.InputIdentitySHA256
		}
	case "bridge-readonly", "bridge-record":
		o := receipt.SameUnitBridge.Options
		o.BridgeID = "perf-shared-environment-bridge"
		o.Record = stage == "bridge-record"
		var plan *i18n.SameUnitBridgePlan
		plan, err = i18n.PrepareSameUnitRepresentationBridge(root, c, o)
		if err == nil {
			payload, _ = json.Marshal(plan.Receipt)
			identity = plan.Receipt.SameUnitBridge.Identity
			if plan.Recorded != o.Record {
				t.Fatal("record semantics changed")
			}
		}
	}
	seconds := time.Since(start).Seconds()
	metrics := map[string]any{"stage": stage, "seconds": seconds + setup, "completed": true, "operation_seconds": seconds, "setup_seconds": setup, "cpu_seconds": bundleProofCPU() - cpu, "gate": sharedWorkflowPerformanceGate(seconds + setup), "identity": identity, "result_sha256": contentidentity.SHA256Hex(payload)}
	if err != nil {
		metrics["error"] = err.Error()
		metrics["gate"] = "FAILED"
	}
	b, _ := json.MarshalIndent(metrics, "", "  ")
	if e := os.WriteFile(filepath.Join(out, stage+".json"), append(b, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
	t.Logf("STAGE=%s SECONDS=%.6f SETUP=%.6f GATE=%s IDENTITY=%s", stage, seconds, setup, metrics["gate"], identity)
	if err != nil {
		t.Fatal(stage, err)
	}
	if metrics["gate"] == "FAILED" {
		t.Fatal("performance hard gate exceeded: 600s")
	}
	if stage != "retranslation-export" {
		after, e := os.ReadFile(filepath.Join(root, rel))
		if e != nil || !bytes.Equal(before, after) {
			t.Fatal("input batch changed", e)
		}
	}
	after, e := os.ReadFile(filepath.Join(root, ref.Path))
	if e != nil || !bytes.Equal(rb, after) {
		t.Fatal("existing bridge changed", e)
	}
}
