package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Profiling is test-only. Each stage has a new memoization scope, matching two
// independent commands; filesystem observations and PASS are never shared.
func TestGenerationBundlePerformanceFixture(t *testing.T) {
	dir, label := os.Getenv("GO_TOUR_BUNDLE_PROFILE_DIR"), os.Getenv("GO_TOUR_BUNDLE_PROFILE_LABEL")
	if dir == "" || label == "" {
		t.Skip("explicit profile destination required")
	}
	root, c, o, e := pendingCommentBundleFixture(t)
	counts := &processFixtureCounts{}
	for i, b := range c.Adapted {
		b.Adapter = processFixtureAdapter{b.Adapter.(validationReopenAdapter), counts}
		c.Adapted[i] = b
		c.TranslationPackages[0].Units[i] = b
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	metrics := map[string]any{}
	var finalBundle []byte
	var finalManifest GenerationBundleManifest
	for _, sample := range []string{label + "-before", label + "-after"} {
		var bundle []byte
		var original GenerationBundleManifest
		for _, stage := range []string{"export", "current-check"} {
			*counts = processFixtureCounts{}
			f, err := os.Create(filepath.Join(dir, sample+"-"+stage+".cpu.pprof"))
			if err != nil {
				t.Fatal(err)
			}
			if err = pprof.StartCPUProfile(f); err != nil {
				t.Fatal(err)
			}
			done := contentidentity.BeginMemoization()
			start, cpu := time.Now(), processFixtureCPU()
			if stage == "export" {
				bundle, original, err = ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: o.Locale, BatchID: e.BatchID})
			} else {
				var checked GenerationBundleManifest
				checked, err = VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle)
				if err == nil && !bytes.Equal(mustJSON(checked), mustJSON(original)) {
					t.Fatal("canonical manifest changed")
				}
			}
			wall, elapsedCPU := time.Since(start).Seconds(), processFixtureCPU()-cpu
			done()
			pprof.StopCPUProfile()
			f.Close()
			if err != nil {
				t.Fatal(stage, err)
			}
			metrics[sample+"/"+stage] = map[string]any{"wall_seconds": wall, "cpu_seconds": elapsedCPU, "adapter_call_counts": *counts}
			t.Logf("%s wall=%.6fs cpu=%.6fs counts=%+v", stage, wall, elapsedCPU, *counts)
		}
		if finalBundle != nil && !bytes.Equal(finalBundle, bundle) {
			t.Fatal("unchanged authority produced different transport")
		}
		finalBundle, finalManifest = bundle, original
	}
	metrics["scope"] = "49 historical A Pages / 9 current-validation revision Pages / 41 authorized teaching slots"
	metrics["bundle_sha256"], metrics["input_identity_sha256"] = sum(finalBundle), finalManifest.InputIdentitySHA256
	metrics["formal_repository_mutation"] = "none"
	b, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, label+".metrics.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

func pendingCommentBundleFixture(t *testing.T) (string, *Catalog, CurrentValidationReopenOptions, *RetranslationExportResult) {
	t.Helper()
	root, c, o, ids := currentValidationFixture(t)
	r, _, err := RecordQualityCheckSurfaceReopen(root, c, QualityCheckSurfaceReopenOptions{AuthoritySource: CurrentValidationAuthority, Locale: o.Locale, ReopenID: o.ReopenID, PreviousSnapshotID: o.PreviousSnapshotID, CurrentValidation: &o})
	if err != nil {
		t.Fatal(err)
	}
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: o.Locale, Generator: RetranslationGeneratorChatGPT, UnitIDs: ids[:9], AllowReexport: true, PreviousSnapshotID: o.PreviousSnapshotID, SurfaceReopenID: r.ReopenID})
	if err != nil {
		t.Fatal(err)
	}
	return root, c, o, e
}
