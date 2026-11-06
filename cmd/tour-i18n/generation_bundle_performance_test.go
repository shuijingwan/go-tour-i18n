package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/pprof"
	"syscall"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func bundleProofCPU() float64 {
	var u syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &u)
	return float64(u.Utime.Sec+u.Stime.Sec) + float64(u.Utime.Usec+u.Stime.Usec)/1e6
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
