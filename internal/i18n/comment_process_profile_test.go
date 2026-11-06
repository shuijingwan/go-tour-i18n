package i18n

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Counters live only in the fixture's adapter, never in production dispatch.
type processFixtureCounts struct {
	ProtectedInputs          int     `json:"protected_inputs"`
	HistoricalRestores       int     `json:"historical_adapter_restores"`
	CandidateValidations     int     `json:"candidate_representation_validations"`
	CommentSourceExtractions int     `json:"comment_source_extractions"`
	CommentTargetExtractions int     `json:"comment_target_extractions"`
	CommentProjectionBuilds  int     `json:"comment_projection_builds"`
	ProtectionSeconds        float64 `json:"protection_seconds"`
	RestoreSeconds           float64 `json:"historical_restore_seconds"`
	ValidationSeconds        float64 `json:"representation_validation_seconds"`
	CommentExtractionSeconds float64 `json:"comment_extraction_seconds"`
}
type processFixtureAdapter struct {
	validationReopenAdapter
	counts *processFixtureCounts
}

func (a processFixtureAdapter) ProtectedInput(g *Glossary) (string, []string, error) {
	start := time.Now()
	a.counts.ProtectedInputs++
	s, t, e := a.validationReopenAdapter.ProtectedInput(g)
	a.counts.ProtectionSeconds += time.Since(start).Seconds()
	return s, t, e
}
func (a processFixtureAdapter) Restore(s string) ([]byte, error) {
	start := time.Now()
	a.counts.HistoricalRestores++
	b, e := a.validationReopenAdapter.Restore(s)
	a.counts.RestoreSeconds += time.Since(start).Seconds()
	return b, e
}
func (a processFixtureAdapter) Validate(b []byte, g *Glossary) error {
	start := time.Now()
	a.counts.CandidateValidations++
	e := a.validationReopenAdapter.Validate(b, g)
	a.counts.ValidationSeconds += time.Since(start).Seconds()
	return e
}
func (a processFixtureAdapter) CommentSourceBlocks() ([]CommentCodeBlock, string, error) {
	start := time.Now()
	a.counts.CommentSourceExtractions++
	b, s, e := a.validationReopenAdapter.CommentSourceBlocks()
	a.counts.CommentExtractionSeconds += time.Since(start).Seconds()
	return b, s, e
}
func (a processFixtureAdapter) CommentTargetBlocks(b []byte) ([]CommentCodeBlock, error) {
	start := time.Now()
	a.counts.CommentTargetExtractions++
	out, e := a.validationReopenAdapter.CommentTargetBlocks(b)
	a.counts.CommentExtractionSeconds += time.Since(start).Seconds()
	return out, e
}
func (a processFixtureAdapter) CommentRevisionBodies(b []byte) ([]CommentRevisionBody, error) {
	a.counts.CommentProjectionBuilds++
	return a.validationReopenAdapter.CommentRevisionBodies(b)
}

func processFixtureCPU() float64 {
	var u syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &u)
	return float64(u.Utime.Sec+u.Stime.Sec) + float64(u.Utime.Usec+u.Stime.Usec)/1e6
}

// Same fixed 49-Page history, exact nine-Page/41-slot revision, real shared
// exporter/importer/process. All lifecycle artifacts are in t.TempDir.
func TestCurrentValidationProcessProfileFixture(t *testing.T) {
	dir, label := os.Getenv("GO_TOUR_PROCESS_PROFILE_DIR"), os.Getenv("GO_TOUR_PROCESS_PROFILE_LABEL")
	if dir == "" || label == "" {
		t.Skip("explicit isolated profile destination required")
	}
	root, c, o, e := importedCommentProcessFixture(t)
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
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	// Match the existing CLI invocation's byte-transform scope. Filesystem
	// reads/currentness are still checked; this adds no new caching behavior.
	done := contentidentity.BeginMemoization()
	start, cpu := time.Now(), processFixtureCPU()
	var result *RetranslationProcessResult
	pprof.Do(context.Background(), pprof.Labels("stage", "process"), func(context.Context) {
		result, err = ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: o.Locale, BatchID: e.BatchID})
	})
	wall, cpuElapsed := time.Since(start).Seconds(), processFixtureCPU()-cpu
	done()
	pprof.StopCPUProfile()
	_ = f.Close()
	if err != nil || result.ValidationPassed != 9 || result.RestorePassed != 9 {
		t.Fatal(result, err)
	}
	metrics := map[string]any{"fixture": "isolated 49 historical A / 9 current-validation Page / 41 teaching slots", "label": label, "wall_seconds": wall, "cpu_seconds": cpuElapsed, "counts": counts, "result": result, "formal_repository_mutation": "none", "catalog_construction_inside_process": 0, "current_assessment_inside_consumed_authority_process": 0}
	b, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, label+".metrics.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wall=%.3fs cpu=%.3fs counts=%+v", wall, cpuElapsed, counts)
}

func importedCommentProcessFixture(t *testing.T) (string, *Catalog, CurrentValidationReopenOptions, *RetranslationExportResult) {
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
	bundle, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: o.Locale, BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string][]byte{}
	for _, expected := range m.ExpectedOutputs {
		name := filepath.Base(expected.BundlePath)
		var p CommentRevisionTransport
		if err := json.Unmarshal(files["formal/comment-revision-transport/"+name+".json"], &p); err != nil {
			t.Fatal(err)
		}
		raw := string(files["formal/protected-candidates/"+name])
		for _, s := range p.Slots {
			raw = strings.Replace(raw, s.Text, "返回教学值。", 1)
		}
		outputs[expected.UnitID] = []byte(raw)
	}
	futureImport(t, root, c, bundle, outputs)
	return root, c, o, e
}
