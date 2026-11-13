package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Opt-in measurement of an already existing b49 attempt-002 invocation, never
// 393 attempt-003. Each test process measures one fresh command invocation.
// The disposable root restores only exact pre-existing validation/result bytes;
// all writes below are engineering ZIPs/metrics outside that authority root.
func TestGenerationBundleCombinedRealAuthorityPerformance(t *testing.T) {
	root, out, stage := os.Getenv("GO_TOUR_COMBINED_PERF_ROOT"), os.Getenv("GO_TOUR_COMBINED_PERF_OUT"), os.Getenv("GO_TOUR_COMBINED_PERF_STAGE")
	if root == "" || out == "" || stage == "" {
		t.Skip("explicit disposable authority root and stage required")
	}
	resolved, err := filepath.EvalSymlinks(root)
	original, originalErr := filepath.Abs("../..")
	if err != nil || originalErr != nil || resolved != root || filepath.Base(root) != "go-tour-combined-export-real-authority" || root == original || !filepath.IsAbs(out) || !strings.HasPrefix(out, "/tmp/") {
		t.Fatal("disposable authority root and /tmp engineering outputs required", err)
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	const id = "b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c"
	const batch = "chatgpt-zh-CN-071"
	baseline := filepath.Join(out, "engineering-standalone.zip")
	path := baseline
	args := []string{"generation-bundle", "export", "--content-package", "learn-docs-v1", "--locale", "zh-CN", "--batch-id", batch, "--unit-id", id, "--output", path}
	switch stage {
	case "export":
	case "current-check":
		args = []string{"generation-bundle", "current-check", "--content-package", "learn-docs-v1", "--bundle", baseline}
	case "combined":
		path = filepath.Join(out, "engineering-combined.zip")
		args[len(args)-1] = path
		args = append(args, "--current-check")
	default:
		t.Fatal("unknown stage")
	}
	before := authorizationDiagnosisInventory(t, root)
	start, cpu := time.Now(), bundleProofCPU()
	finish := contentidentity.BeginMemoization()
	// Mirror run's source/catalog/registration setup, permitting only counters
	// around the existing bridge resolver. No builder or eligibility is replaced.
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
	if err := attachCommandTranslationPackages(root, c, args); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	requests, builds := 0, 0
	resolve := c.BridgeResolver
	if resolve == nil {
		t.Fatal("missing invocation-owned bridge resolver")
	}
	c.BridgeResolver = func(root string, input *i18n.Catalog, mode string, ref i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
		requests++
		return resolve(root, input, mode, ref, func() (*i18n.SameUnitBridgeResolution, error) { builds++; return build() })
	}
	stdout, err := authorizationDiagnosisOutput(t, func() error { return generationBundleLifecycleCommand(root, c, args) })
	wall, elapsedCPU := time.Since(start).Seconds(), bundleProofCPU()-cpu
	finish()
	metrics := map[string]any{"stage": stage, "wall_seconds": wall, "cpu_seconds": elapsedCPU, "bridge_requests": requests, "bridge_full_builds": builds, "generation_handoff": false}
	if err != nil {
		metrics["error"] = err.Error()
	}
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("authority fixture mutated")
	}
	metrics["formal_repository_mutation"] = "none"
	if err == nil {
		data, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		files, e := i18n.ReadTransportBundle(data, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
		if e != nil {
			t.Fatal(e)
		}
		var m i18n.GenerationBundleManifest
		if e = json.Unmarshal(files["manifest.json"], &m); e != nil {
			t.Fatal(e)
		}
		if m.Attempt != 2 || m.TaskKind != "translation-unit-retry" || len(m.ExpectedOutputs) != 1 || m.ExpectedOutputs[0].UnitID != id {
			t.Fatal("wrong disposable retry/attempt")
		}
		metrics["bundle_sha256"], metrics["input_identity_sha256"] = contentidentity.SHA256Hex(data), m.InputIdentitySHA256
		// The measurement reuses an existing real invocation, not a newly
		// invented retry. Its exact archive must also remain byte-identical.
		archived, e := os.ReadFile(filepath.Join(root, "data/generation-archives/zh-CN/1c83c7499e3703d4d815e20176fb8c4b0dee1565bede430b793c4da3293fbbdf.zip"))
		if e != nil || !bytes.Equal(archived, data) {
			t.Fatal("existing real retry Bundle identity drift", e)
		}
		if stage != "export" && !strings.Contains(string(stdout), "Generation bundle: CURRENT") {
			t.Fatal("missing CURRENT", string(stdout))
		}
		if stage == "combined" {
			original, e := os.ReadFile(baseline)
			if e != nil || !bytes.Equal(original, data) {
				t.Fatal("combined bytes differ", e)
			}
		}
	}
	b, _ := json.MarshalIndent(metrics, "", "  ")
	if e := os.WriteFile(filepath.Join(out, stage+".json"), append(b, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
	t.Logf("%s", b)
	if err != nil {
		t.Fatal(err)
	}
}
