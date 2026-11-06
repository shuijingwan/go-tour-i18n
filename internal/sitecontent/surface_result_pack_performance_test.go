package sitecontent

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// The actual inputs/outputs are readonly. All repository operations and the
// returned Result ZIP belong exclusively to a disposable repository copy.
func TestSurfaceResultPackRealPerformance(t *testing.T) {
	out := os.Getenv("GO_TOUR_SURFACE_RESULT_PACK_REAL_PROFILE")
	if out == "" {
		t.Skip("explicit disposable real-repository profile required")
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	root := copiedSurfaceTransportRepository(t)
	done := contentidentity.BeginMemoization()
	defer done()
	start := time.Now()
	f, err := os.Create(filepath.Join(out, "catalog.cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	fresh, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	c, err := i18n.ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := i18n.HydrateCatalogSources(c, fresh); err != nil {
		t.Fatal(err)
	}
	// Match the production result-pack caller, including eager registration.
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	AttachIntegratedSurfaceAuthority(c)
	pprof.StopCPUProfile()
	f.Close()
	catalogSeconds := time.Since(start).Seconds()
	t.Logf("DISPOSABLE_CATALOG_READY wall_seconds=%.3f", catalogSeconds)
	bundle, err := os.ReadFile("/tmp/zh-CN-chatgpt-zh-CN-060-generation-20261027-131729.zip")
	if err != nil {
		t.Fatal(err)
	}
	if digest(bundle) != "f9d850603415bc7aec37034e7d5130000511e45c9c71f35e81b11adba571f470" {
		t.Fatal("unexpected real Generation input")
	}
	outputs, err := os.ReadFile("/home/wangqiang/下载/zh-CN-chatgpt-zh-CN-060-outputs.zip")
	if err != nil {
		t.Fatal(err)
	}
	if digest(outputs) != "dd7d5d51cd12cee655fb4056f9da0f1b48ea593adfb9a851eb39ea1ffde9d593" {
		t.Fatal("unexpected real generated outputs")
	}
	z, err := zip.NewReader(bytes.NewReader(outputs), int64(len(outputs)))
	if err != nil {
		t.Fatal(err)
	}
	if len(z.File) != 5 {
		t.Fatal("real output exact-set changed")
	}
	dir := t.TempDir()
	for _, entry := range z.File {
		if entry.Name != "outputs/"+filepath.Base(entry.Name) || !entry.Mode().IsRegular() {
			t.Fatal("non-flat output transport")
		}
		r, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, dir, filepath.Base(entry.Name), b)
	}
	f, err = os.Create(filepath.Join(out, "result-pack.cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	start = time.Now()
	result, manifest, packErr := i18n.PackGenerationResultBundle(root, c, bundle, "chatgpt", i18n.FormalChatGPTGenerationModel, dir)
	seconds := time.Since(start).Seconds()
	pprof.StopCPUProfile()
	f.Close()
	metrics := map[string]any{"catalog_wall_seconds": catalogSeconds, "result_pack_wall_seconds": seconds, "outputs": len(manifest.Outputs), "generation_bundle_sha256": digest(bundle), "generation_outputs_sha256": digest(outputs), "result_zip_sha256": digest(result), "original_repository_mutation": "none", "error": ""}
	if packErr != nil {
		metrics["error"] = packErr.Error()
	}
	b, _ := json.MarshalIndent(metrics, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "metrics.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("DISPOSABLE_RESULT_PACK wall_seconds=%.3f outputs=%d err=%v", seconds, len(manifest.Outputs), packErr)
	if packErr != nil {
		t.Fatal(packErr)
	}
	if len(manifest.Outputs) != 5 {
		t.Fatal("incomplete real Result")
	}
	if _, err := os.Stat(filepath.Join(root, "data/retranslation-runs/zh-CN/chatgpt-zh-CN-060/generation-imports/initial-manifest.json")); !os.IsNotExist(err) {
		t.Fatal("pack installed lifecycle state", err)
	}
	if err := os.WriteFile(filepath.Join(out, "disposable-result.zip"), result, 0600); err != nil {
		t.Fatal(err)
	}
}
