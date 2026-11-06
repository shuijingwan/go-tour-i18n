package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Opt-in real-authority performance proof. Every write targets TempDir or the
// explicitly selected /tmp evidence directory, never formal repository bytes.
func TestBridgeBackedLifecyclePerformance(t *testing.T) {
	out, mode := os.Getenv("GO_TOUR_BRIDGE_PROFILE_DIR"), os.Getenv("GO_TOUR_BRIDGE_PROFILE_MODE")
	if out == "" {
		t.Skip("explicit isolated profiling destination required")
	}
	root, temp, c, _, _, observed := copiedCommentRevisionRealFixture(t, false)
	copyFile := func(path string) {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		observed[path] = contentidentity.SHA256Hex(b)
		dst := filepath.Join(temp, path)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	const receiptPath = "data/quality-check-reopens/zh-CN/learn-docs-qc018-ordinary-qc-transport.json"
	copyFile(receiptPath)
	b, _ := os.ReadFile(filepath.Join(temp, receiptPath))
	var receipt i18n.QualityCheckSurfaceReopen
	if err := json.Unmarshal(b, &receipt); err != nil || receipt.SameUnitBridge == nil {
		t.Fatal(err)
	}
	for _, ref := range receipt.SameUnitBridge.Dependencies {
		copyFile(ref.Path)
	}
	const batch = "data/retranslation-runs/zh-CN/chatgpt-zh-CN-053"
	entries, err := filepath.Glob(filepath.Join(root, batch, "inputs/*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range entries {
		rel, _ := filepath.Rel(root, path)
		copyFile(filepath.ToSlash(rel))
	}
	copyFile(batch + "/manifest.json")
	var m i18n.RetranslationBatchManifest
	b, _ = os.ReadFile(filepath.Join(temp, batch, "manifest.json"))
	if err := json.Unmarshal(b, &m); err != nil || len(m.Units) != 4 {
		t.Fatal(err)
	}
	// Already-generated formal responses are immutable readonly fixtures. The
	// disposable batch starts pending, so import/process exercise real writers.
	staging := t.TempDir()
	for _, u := range m.Units {
		name := filepath.Base(u.InputPath)
		path := batch + "/raw-responses/" + name
		copyFile(path)
		raw, err := os.ReadFile(filepath.Join(temp, path))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(staging, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{"raw-responses", "generation-imports", "candidates", "validation"} {
		if err := os.RemoveAll(filepath.Join(temp, batch, dir)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(temp, batch, "result.json")); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	var bundle, result []byte
	metrics := map[string]any{}
	stages := []string{"export", "current-check", "result-pack", "import", "process"}
	if mode == "authority" {
		stages = []string{"authority"}
	}
	for _, stage := range stages {
		// Each command gets its own invocation context; nested calls may share it.
		sitecontent.AttachPageReplacementInvocation(temp, c)
		requests, builds := 0, 0
		resolve := c.BridgeResolver
		c.BridgeResolver = func(root string, c *i18n.Catalog, mode string, ref i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
			requests++
			return resolve(root, c, mode, ref, func() (*i18n.SameUnitBridgeResolution, error) { builds++; return build() })
		}
		done := contentidentity.BeginMemoization()
		f, err := os.Create(filepath.Join(out, mode+"-"+stage+".cpu.pprof"))
		if err != nil {
			t.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			t.Fatal(err)
		}
		start, cpu := time.Now(), bundleProofCPU()
		switch stage {
		case "authority":
			_, err = i18n.CheckQualityCheckSurfaceReopen(temp, c, "zh-CN", receipt.ReopenID)
		case "export":
			bundle, _, err = i18n.ExportTranslationUnitGenerationBundle(temp, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: m.BatchID})
			if err == nil {
				b, e := os.ReadFile(filepath.Join(root, batch, "generation-imports/initial-manifest.json"))
				if e != nil {
					t.Fatal(e)
				}
				var original i18n.GenerationResultBundleManifest
				if e = json.Unmarshal(b, &original); e != nil {
					t.Fatal(e)
				}
				b, e = os.ReadFile(filepath.Join(root, "data/generation-archives/zh-CN", original.GenerationBundleSHA256+".zip"))
				if e != nil {
					t.Fatal(e)
				}
				if !bytes.Equal(bundle, b) {
					t.Fatal("performance-only change altered serialized formal Bundle")
				}
			}
		case "current-check":
			_, err = i18n.VerifyCurrentTranslationUnitGenerationBundle(temp, c, bundle)
		case "result-pack":
			result, _, err = i18n.PackGenerationResultBundle(temp, c, bundle, "chatgpt", i18n.FormalChatGPTGenerationModel, staging)
		case "import":
			_, err = i18n.ImportGenerationResultBundle(temp, c, bundle, result)
		case "process":
			var r *i18n.RetranslationProcessResult
			r, err = i18n.ProcessRetranslationBatch(temp, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: m.BatchID})
			if err == nil && (r.RestorePassed != 4 || r.ValidationPassed != 4) {
				t.Fatal(r)
			}
			if err == nil {
				for _, u := range r.Units {
					got, e := os.ReadFile(filepath.Join(temp, batch, u.CandidatePath))
					if e != nil {
						t.Fatal(e)
					}
					want, e := os.ReadFile(filepath.Join(root, batch, u.CandidatePath))
					if e != nil {
						t.Fatal(e)
					}
					if !bytes.Equal(got, want) {
						t.Fatal("optimized restore changed formal candidate", u.UnitID)
					}
				}
			}
		}
		elapsed, cpuTime := time.Since(start).Seconds(), bundleProofCPU()-cpu
		pprof.StopCPUProfile()
		f.Close()
		done()
		if err != nil {
			t.Fatal(stage, err)
		}
		if mode != "authority" && builds != 1 {
			t.Fatal("expected exactly one complete reconstruction per command", stage, requests, builds)
		}
		metrics[stage] = map[string]any{"wall_seconds": elapsed, "cpu_seconds": cpuTime, "bridge_requests": requests, "bridge_full_reconstructions": builds}
		t.Logf("%s wall=%.3fs cpu=%.3fs bridge requests=%d full builds=%d", stage, elapsed, cpuTime, requests, builds)
		partial, _ := json.MarshalIndent(metrics, "", "  ")
		if err := os.WriteFile(filepath.Join(out, mode+".metrics.json"), append(partial, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for path, want := range observed {
		// Engineering test source is not formal language authority; a separate
		// focused test may legitimately update it during an isolated proof.
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		got, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || contentidentity.SHA256Hex(got) != want {
			t.Fatal("formal input mutated", path, err)
		}
	}
	metrics["formal_artifact_mutation"] = "none"
	b, _ = json.MarshalIndent(metrics, "", "  ")
	if err := os.WriteFile(filepath.Join(out, mode+".metrics.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
