package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime/pprof"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func bundleProofCPU() float64 {
	var u syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &u)
	return float64(u.Utime.Sec+u.Stime.Sec) + float64(u.Utime.Usec+u.Stime.Usec)/1e6
}

// Existing retry bytes and complete batch state are copied unchanged to a
// disposable authority root. No export/import/process or formal publication.
func TestGenerationBundleRetryRealAuthorityPerformance(t *testing.T) {
	root, bundlePath, out, label := os.Getenv("GO_TOUR_RETRY_PERF_ROOT"), os.Getenv("GO_TOUR_RETRY_PERF_BUNDLE"), os.Getenv("GO_TOUR_RETRY_PERF_OUT"), os.Getenv("GO_TOUR_RETRY_PERF_LABEL")
	if root == "" || bundlePath == "" || out == "" || label == "" {
		t.Skip("explicit disposable retry proof inputs required")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root || filepath.Base(root) != "authority" || !strings.HasPrefix(filepath.Base(filepath.Dir(root)), "go-tour-retry-current-profile-") || !strings.HasPrefix(out, "/tmp/") || strings.ContainsAny(label, "/\\") {
		t.Fatal("unsafe disposable proof inputs", err)
	}
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(data, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.GenerationBundleManifest
	var formal i18n.RetranslationBatchManifest
	if err := json.Unmarshal(files["manifest.json"], &m); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(files["formal/manifest.json"], &formal); err != nil {
		t.Fatal(err)
	}
	if m.TaskKind != "translation-unit-retry" || len(m.ExpectedOutputs) != 1 || formal.UnitCount < 2 {
		t.Fatal("not a single retry from a complete multi-Unit batch")
	}
	before := authorizationDiagnosisInventory(t, root)
	profile, err := os.Create(filepath.Join(out, label+".cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.StartCPUProfile(profile); err != nil {
		t.Fatal(err)
	}
	finish := contentidentity.BeginMemoization()
	start, cpu := time.Now(), bundleProofCPU()
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
	if err := attachCommandTranslationPackages(root, c, []string{"generation-bundle", "current-check"}); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	c, err = i18n.SelectCurrentValidationReopenPackage(c, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	requests, builds := 0, 0
	resolver := c.BridgeResolver
	if resolver != nil {
		c.BridgeResolver = func(root string, c *i18n.Catalog, mode string, ref i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
			requests++
			return resolver(root, c, mode, ref, func() (*i18n.SameUnitBridgeResolution, error) { builds++; return build() })
		}
	}
	checked, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, data)
	wall, usedCPU := time.Since(start).Seconds(), bundleProofCPU()-cpu
	finish()
	pprof.StopCPUProfile()
	profile.Close()
	metrics := map[string]any{"wall_seconds": wall, "cpu_seconds": usedCPU, "bridge_requests": requests, "bridge_full_builds": builds, "batch_unit_count": formal.UnitCount, "bundle_sha256": contentidentity.SHA256Hex(data), "input_identity_sha256": m.InputIdentitySHA256, "formal_repository_mutation": "none"}
	if err != nil {
		metrics["error"] = err.Error()
	}
	b, _ := json.MarshalIndent(metrics, "", "  ")
	if e := os.WriteFile(filepath.Join(out, label+".json"), append(b, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
	t.Logf("%s", b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(checked, m) {
		t.Fatal("canonical Bundle manifest/identity changed")
	}
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("disposable authority mutated")
	}
}

// One disposable real-authority export + check, never a Generation handoff.
// Both use separate invocation memoization scopes; no PASS or graph is shared.
func TestGenerationBundleRealAuthorityPerformance(t *testing.T) {
	out := os.Getenv("GO_TOUR_REAL_BUNDLE_PROFILE_DIR")
	if out == "" {
		t.Skip("explicit disposable proof destination required")
	}
	setup := time.Now()
	root, temp, c, m, authority, observed := copiedCommentRevisionRealFixture(t, true)
	setupSeconds := time.Since(setup).Seconds()
	if len(c.TranslationScope) != 49 || m.UnitCount != 9 {
		t.Fatal("wrong package/Unit scope")
	}
	slots := 0
	for _, f := range authority.CurrentValidation.Findings {
		slots += len(f.CommentSlots)
	}
	if slots != 41 {
		t.Fatal(slots)
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	var bundle []byte
	var bm i18n.GenerationBundleManifest
	metrics := map[string]any{"fixture": "real exact registered 49 Page package; nine revision Pages; 41 teaching slots; pending output state only in TempDir", "copy_and_catalog_setup_seconds": setupSeconds}
	for _, stage := range []string{"export", "current-check"} {
		f, err := os.Create(filepath.Join(out, "real-"+stage+".cpu.pprof"))
		if err != nil {
			t.Fatal(err)
		}
		if err = pprof.StartCPUProfile(f); err != nil {
			t.Fatal(err)
		}
		done := contentidentity.BeginMemoization()
		start, cpu := time.Now(), bundleProofCPU()
		if stage == "export" {
			bundle, bm, err = i18n.ExportTranslationUnitGenerationBundle(temp, c, i18n.GenerationBundleOptions{Locale: m.Locale, BatchID: m.BatchID})
		} else {
			var checked i18n.GenerationBundleManifest
			checked, err = i18n.VerifyCurrentTranslationUnitGenerationBundle(temp, c, bundle)
			if err == nil && checked.InputIdentitySHA256 != bm.InputIdentitySHA256 {
				t.Fatal("identity changed")
			}
		}
		wall, elapsedCPU := time.Since(start).Seconds(), bundleProofCPU()-cpu
		done()
		pprof.StopCPUProfile()
		f.Close()
		if err != nil {
			t.Fatal(stage, err)
		}
		metrics[stage] = map[string]any{"wall_seconds": wall, "cpu_seconds": elapsedCPU}
		t.Logf("real %s wall=%.3fs cpu=%.3fs", stage, wall, elapsedCPU)
	}
	if len(bm.ExpectedOutputs) != 9 {
		t.Fatal("wrong outputs")
	}
	files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	editable := 0
	for _, r := range m.Units {
		var p i18n.CommentRevisionTransport
		if err = json.Unmarshal(files["formal/comment-revision-transport/"+filepath.Base(r.InputPath)+".json"], &p); err != nil {
			t.Fatal(err)
		}
		editable += len(p.Slots)
	}
	if editable != 41 {
		t.Fatal("editable scope drift", editable)
	}
	path := filepath.Join(out, "disposable-current-generation.zip")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write(bundle); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	for path, want := range observed {
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || contentidentity.SHA256Hex(b) != want {
			t.Fatal("real dependency changed", path)
		}
	}
	metrics["bundle_sha256"], metrics["input_identity_sha256"] = contentidentity.SHA256Hex(bundle), bm.InputIdentitySHA256
	metrics["disposable_bundle_path"], metrics["editable_authorized_slots"] = path, editable
	metrics["formal_repository_mutation"] = "none"
	metrics["generation_handoff"] = false
	b, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(out, "real.metrics.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
