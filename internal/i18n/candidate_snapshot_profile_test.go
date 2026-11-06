package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Real shared Snapshot creator over a fixed 49-Page successor with nine revised
// Pages / 41 teaching slots. Lifecycle state exists only in t.TempDir.
func TestCandidateSnapshotPerformanceFixture(t *testing.T) {
	dir, label := os.Getenv("GO_TOUR_SNAPSHOT_PROFILE_DIR"), os.Getenv("GO_TOUR_SNAPSHOT_PROFILE_LABEL")
	if dir == "" || label == "" {
		t.Skip("explicit isolated profile destination required")
	}
	root, c, o, e := importedCommentProcessFixture(t)
	if r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: o.Locale, BatchID: e.BatchID}); err != nil || r.ValidationPassed != 9 {
		t.Fatal(r, err)
	}
	counts := &processFixtureCounts{}
	for i, b := range c.Adapted {
		b.Adapter = processFixtureAdapter{b.Adapter.(validationReopenAdapter), counts}
		c.Adapted[i] = b
		c.TranslationPackages[0].Units[i] = b
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, label+".cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err = pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	done := contentidentity.BeginMemoization()
	start, cpu := time.Now(), processFixtureCPU()
	s, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "disposable-performance"})
	wall, cpuElapsed := time.Since(start).Seconds(), processFixtureCPU()-cpu
	done()
	pprof.StopCPUProfile()
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if s.UnitCount != 49 || s.PageCount != 49 || s.ExampleCount != 0 {
		t.Fatal("wrong exact scope")
	}
	revised := 0
	for _, u := range s.Units {
		if u.SelectedBatchID == e.BatchID {
			revised++
		}
	}
	if revised != 9 {
		t.Fatal("wrong revision selection", revised)
	}
	metrics := map[string]any{"fixture": "49 Page successor, nine revised Pages, 41 teaching slots", "wall_seconds": wall, "cpu_seconds": cpuElapsed, "counts": counts, "unit_count": s.UnitCount, "revised": revised, "current_revision": s.CurrentRevision != nil, "current_reattestation": s.CurrentReattestation != nil, "manifest": s, "formal_repository_mutation": "none"}
	b, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, label+".metrics.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wall=%.3fs cpu=%.3fs counts=%+v", wall, cpuElapsed, counts)
}
