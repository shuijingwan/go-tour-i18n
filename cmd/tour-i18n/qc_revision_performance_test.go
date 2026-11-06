package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
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

// All exports below target the sparse TempDir. The real 052 directory and ZIP
// are only read and checked for byte preservation after the measurements.
func TestQCRevisionRealAuthorityPerformance(t *testing.T) {
	out, label := os.Getenv("GO_TOUR_REAL_QC_PROFILE_DIR"), os.Getenv("GO_TOUR_REAL_QC_PROFILE_LABEL")
	if out == "" || label == "" {
		t.Skip("explicit isolated profile destination required")
	}
	root, temp, c, old, _, observed := copiedCommentRevisionRealFixture(t, false)
	const batch = "data/retranslation-runs/zh-CN/chatgpt-zh-CN-052"
	const snapshot = "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-018"
	copied := map[string]bool{}
	var file func(string)
	var visit func(any)
	visit = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for _, v := range x {
				visit(v)
			}
		case []any:
			for _, v := range x {
				visit(v)
			}
		case string:
			if strings.HasPrefix(x, "{") {
				var v any
				if json.Unmarshal([]byte(x), &v) == nil {
					visit(v)
				}
				return
			}
			if filepath.IsAbs(x) || strings.Contains(x, "..") || strings.ContainsAny(x, "\r\n") {
				return
			}
			for _, prefix := range []string{"data/", "locales/", "docs/", "internal/", "_content/", "content/"} {
				if strings.HasPrefix(x, prefix) {
					if s, e := os.Stat(filepath.Join(root, x)); e == nil && s.Mode().IsRegular() {
						file(x)
					}
					return
				}
			}
		}
	}
	file = func(path string) {
		if copied[path] {
			return
		}
		copied[path] = true
		b, e := os.ReadFile(filepath.Join(root, path))
		if e != nil {
			t.Fatal(path, e)
		}
		observed[path] = contentidentity.SHA256Hex(b)
		dst := filepath.Join(temp, path)
		if e = os.MkdirAll(filepath.Dir(dst), 0755); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(dst, b, 0600); e != nil {
			t.Fatal(e)
		}
		if strings.HasSuffix(path, ".json") {
			var v any
			if json.Unmarshal(b, &v) == nil {
				visit(v)
			}
		}
	}
	tree := func(dir string) {
		t.Helper()
		if e := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if !d.IsDir() {
				rel, _ := filepath.Rel(root, path)
				file(filepath.ToSlash(rel))
			}
			return nil
		}); e != nil {
			t.Fatal(e)
		}
	}
	tree("data/retranslation-runs/zh-CN/" + old.BatchID)
	tree(snapshot)
	formal, e := os.ReadFile(filepath.Join(root, batch, "manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	if contentidentity.SHA256Hex(formal) != "b40b2c3c646437bb0cbd2bc03fb971f79123e6557e0240d389d8823ce7906a83" {
		t.Fatal("formal manifest changed")
	}
	var expected i18n.RetranslationBatchManifest
	if e = json.Unmarshal(formal, &expected); e != nil {
		t.Fatal(e)
	}
	// Remove only the disposable copy of 052, making the exact export pending.
	if e = os.RemoveAll(filepath.Join(temp, batch)); e != nil {
		t.Fatal(e)
	}
	ids := []string{}
	for _, u := range expected.Units {
		ids = append(ids, u.UnitID)
	}
	if e = os.MkdirAll(out, 0755); e != nil {
		t.Fatal(e)
	}
	metrics := map[string]any{}
	var exported *i18n.RetranslationExportResult
	var bundle []byte
	var bm i18n.GenerationBundleManifest
	for _, stage := range []string{"retranslation-export", "bundle-export", "bundle-check"} {
		// Three separate CLI invocations have three fresh Catalog projections.
		sitecontent.AttachPageReplacementInvocation(temp, c)
		f, e := os.Create(filepath.Join(out, label+"-"+stage+".cpu.pprof"))
		if e != nil {
			t.Fatal(e)
		}
		if e = pprof.StartCPUProfile(f); e != nil {
			t.Fatal(e)
		}
		done := contentidentity.BeginMemoization()
		start, cpu := time.Now(), bundleProofCPU()
		switch stage {
		case "retranslation-export":
			exported, e = i18n.ExportRetranslationBatch(temp, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, BatchID: expected.BatchID, PreviousSnapshotID: "learn-docs-v1-qc-018", AllowReexport: true, UnitKind: i18n.UnitKindPage, UnitIDs: ids})
		case "bundle-export":
			bundle, bm, e = i18n.ExportTranslationUnitGenerationBundle(temp, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: exported.BatchID})
		case "bundle-check":
			var m i18n.GenerationBundleManifest
			m, e = i18n.VerifyCurrentTranslationUnitGenerationBundle(temp, c, bundle)
			if e == nil && m.InputIdentitySHA256 != bm.InputIdentitySHA256 {
				t.Fatal("identity changed")
			}
		}
		wall, cpuElapsed := time.Since(start).Seconds(), bundleProofCPU()-cpu
		done()
		pprof.StopCPUProfile()
		f.Close()
		if e != nil {
			t.Fatal(stage, e)
		}
		metrics[stage] = map[string]any{"wall_seconds": wall, "cpu_seconds": cpuElapsed}
		t.Logf("%s wall=%.3fs cpu=%.3fs", stage, wall, cpuElapsed)
	}
	got, e := os.ReadFile(filepath.Join(temp, batch, "manifest.json"))
	if e != nil || !bytes.Equal(got, formal) {
		t.Fatal("exact real export manifest differs", e)
	}
	realBundle, e := os.ReadFile("/tmp/zh-CN-chatgpt-zh-CN-052-generation.zip")
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(bundle, realBundle) {
		t.Fatal("serialized Bundle differs from formal transport")
	}
	for p, want := range observed {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		b, e := os.ReadFile(filepath.Join(root, p))
		if e != nil || contentidentity.SHA256Hex(b) != want {
			t.Fatal("real input mutated", p)
		}
	}
	metrics["formal_repository_mutation"] = "none"
	metrics["formal_bundle_equal"] = true
	metrics["manifest_equal"] = true
	metrics["bundle_identity"] = bm.InputIdentitySHA256
	metrics["observed_real_input_sha256"] = observed
	b, _ := json.MarshalIndent(metrics, "", "  ")
	if e = os.WriteFile(filepath.Join(out, label+".metrics.json"), append(b, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
}
