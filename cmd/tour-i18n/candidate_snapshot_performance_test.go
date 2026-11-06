package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Snapshot only in an isolated copy of exact completed candidates. Never
// create, remove, rename, or revalidate a real lifecycle artifact.
func TestCandidateSnapshotRealAuthorityPerformance(t *testing.T) {
	out := os.Getenv("GO_TOUR_REAL_SNAPSHOT_PROFILE_DIR")
	if out == "" {
		t.Skip("explicit disposable proof destination required")
	}
	root, temp, c, m, _, observed := copiedCommentRevisionRealFixture(t, false)
	const current = "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-018/manifest.json"
	old, err := os.ReadFile(filepath.Join(root, current))
	if err != nil {
		t.Fatal(err)
	}
	if contentidentity.SHA256Hex(old) != "c09283fa2f0edfba9c88ad85fce406610d214c4c23f074ad7489b09946fcfe30" {
		t.Fatal("qc018 identity changed")
	}
	observed[current] = contentidentity.SHA256Hex(old)
	var accepted i18n.QualityCheckSnapshotManifest
	if err = json.Unmarshal(old, &accepted); err != nil {
		t.Fatal(err)
	}
	batch := "data/retranslation-runs/" + m.Locale + "/" + m.BatchID
	for _, dir := range []string{batch + "/candidates", batch + "/validation", batch + "/revalidation-history"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			b, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			observed[filepath.ToSlash(rel)] = contentidentity.SHA256Hex(b)
			dst := filepath.Join(temp, rel)
			if e = os.MkdirAll(filepath.Dir(dst), 0755); e != nil {
				return e
			}
			return os.WriteFile(dst, b, 0600)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err := os.ReadFile(filepath.Join(root, batch, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	observed[batch+"/result.json"] = contentidentity.SHA256Hex(result)
	if err = os.WriteFile(filepath.Join(temp, batch, "result.json"), result, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(out, "real-after.cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err = pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	done := contentidentity.BeginMemoization()
	start, cpu := time.Now(), bundleProofCPU()
	s, path, err := i18n.CreateQualityCheckCandidateSnapshot(temp, c, i18n.QualityCheckSnapshotOptions{Locale: m.Locale, SnapshotID: "disposable-snapshot-performance"})
	wall, cpuElapsed := time.Since(start).Seconds(), bundleProofCPU()-cpu
	done()
	pprof.StopCPUProfile()
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if s.UnitCount != 49 || s.PageCount != 49 || s.ExampleCount != 0 {
		t.Fatal("exact package scope changed")
	}
	originalUnits, _ := json.Marshal(accepted.Units)
	selectedUnits, _ := json.Marshal(s.Units)
	if !bytes.Equal(originalUnits, selectedUnits) {
		t.Fatal("selected Unit/candidate/source/validation/attempt differs from qc018")
	}
	revised := 0
	for _, u := range s.Units {
		if u.SelectedBatchID == m.BatchID {
			revised++
		}
	}
	if revised != 9 {
		t.Fatal("wrong selected revision delta", revised)
	}
	for path, sha := range observed {
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || contentidentity.SHA256Hex(b) != sha {
			t.Fatal("real formal bytes mutated", path)
		}
	}
	data, err := json.MarshalIndent(map[string]any{"wall_seconds": wall, "cpu_seconds": cpuElapsed, "snapshot": s, "selected_units_equal_qc018": true, "revised": revised, "carried": 49 - revised, "disposable_snapshot_path": path, "observed_real_input_sha256": observed, "formal_repository_mutation": "none", "qc018_mutated": false}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(out, "real-after.metrics.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("real equivalent 49-Page Snapshot wall=%.3fs cpu=%.3fs; selected Units exactly equal qc018", wall, cpuElapsed)
}
