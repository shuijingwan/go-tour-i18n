package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// QC feedback on a successor whose revised candidates came from a machine
// reopen. Every lifecycle artifact belongs to t.TempDir, never the repository.
func nestedQCRevisionFixture(t *testing.T) (string, *Catalog, RetranslationExportOptions) {
	t.Helper()
	root, c, o, e := importedCommentProcessFixture(t)
	if r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: o.Locale, BatchID: e.BatchID}); err != nil || r.ValidationPassed != 9 {
		t.Fatal(r, err)
	}
	const snapshot = "qc-feedback-successor"
	s := futureSnapshot(t, root, c, snapshot)
	for i, u := range s.Units[:9] {
		rating, finding := "A", ""
		if i >= 4 && i < 8 {
			rating = []string{"B", "C", "B", "B"}[i-4]
			finding = fmt.Sprintf("Exact independent finding for %s", u.UnitID)
		}
		futureRecord(t, root, c, snapshot, o.PreviousSnapshotID, rating, finding, []string{u.UnitID})
	}
	ids := []string{}
	for _, u := range s.Units[4:8] {
		ids = append(ids, u.UnitID)
	}
	return root, c, RetranslationExportOptions{Locale: o.Locale, Generator: RetranslationGeneratorChatGPT, PreviousSnapshotID: snapshot, AllowReexport: true, UnitIDs: ids}
}

func TestQCRevisionTransportPerformanceFixture(t *testing.T) {
	dir, label := os.Getenv("GO_TOUR_QC_PROFILE_DIR"), os.Getenv("GO_TOUR_QC_PROFILE_LABEL")
	if dir == "" || label == "" {
		t.Skip("explicit isolated profile destination required")
	}
	root, c, o := nestedQCRevisionFixture(t)
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
	var e *RetranslationExportResult
	var bundle []byte
	var m GenerationBundleManifest
	for _, stage := range []string{"retranslation-export", "bundle-export", "bundle-check"} {
		*counts = processFixtureCounts{}
		f, err := os.Create(filepath.Join(dir, label+"-"+stage+".cpu.pprof"))
		if err != nil {
			t.Fatal(err)
		}
		if err = pprof.StartCPUProfile(f); err != nil {
			t.Fatal(err)
		}
		done := contentidentity.BeginMemoization()
		start, cpu := time.Now(), processFixtureCPU()
		switch stage {
		case "retranslation-export":
			e, err = ExportRetranslationBatch(root, c, o)
		case "bundle-export":
			bundle, m, err = ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: o.Locale, BatchID: e.BatchID})
		case "bundle-check":
			var checked GenerationBundleManifest
			checked, err = VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle)
			if err == nil && !bytes.Equal(mustJSON(checked), mustJSON(m)) {
				t.Fatal("manifest changed")
			}
		}
		wall, cpuElapsed := time.Since(start).Seconds(), processFixtureCPU()-cpu
		done()
		pprof.StopCPUProfile()
		f.Close()
		if err != nil {
			t.Fatal(stage, err)
		}
		metrics[stage] = map[string]any{"wall_seconds": wall, "cpu_seconds": cpuElapsed, "counts": *counts}
		t.Logf("%s wall=%.3fs cpu=%.3fs counts=%+v", stage, wall, cpuElapsed, counts)
	}
	batch := readRetranslationManifest(t, root, e.BatchID)
	if len(batch.Units) != 4 {
		t.Fatal("wrong concentrated scope")
	}
	for i, r := range batch.Units {
		if r.UnitID != o.UnitIDs[i] || r.RevisionFeedbackSource != "quality_check" || r.PreviousRating != []string{"B", "C", "B", "B"}[i] || r.PreviousFinding != fmt.Sprintf("Exact independent finding for %s", r.UnitID) {
			t.Fatal("feedback drift", r)
		}
	}
	metrics["manifest"] = batch
	metrics["bundle_sha256"] = sum(bundle)
	metrics["bundle_identity"] = m.InputIdentitySHA256
	data, _ := json.MarshalIndent(metrics, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, label+".metrics.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
