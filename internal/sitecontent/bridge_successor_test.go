package sitecontent

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func testBridgeSuccessorContinuity(t *testing.T, parent string, c *i18n.Catalog, snapshot, id string) {
	start := time.Now()
	defer func() { t.Logf("successor continuity wall: %s", time.Since(start)) }()
	root := cloneBridgeSuccessorFixture(t, parent)
	view := *c
	c = &view
	AttachPageReplacementInvocation(root, c)
	const finding = "Second independent QC finding for the revised ordinary prose."
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "independent-bridge-fixture-reviewer", Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: "comment-successor", UnitIDs: []string{id}, Rating: "C", Finding: finding}); err != nil {
		t.Fatal(err)
	}
	exp, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{id}, AllowReexport: true, PreviousSnapshotID: snapshot, UnitKind: i18n.UnitKindPage})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, exp.BatchPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.RetranslationBatchManifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m.Units[0].RepresentationBridge != nil || m.Units[0].PreviousRating != "C" || m.Units[0].PreviousFinding != finding {
		t.Fatal("old bridge finding replaced current QC authority")
	}
	bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: exp.BatchID})
	if err != nil {
		t.Fatal("successor ordinary Bundle must inherit historical response decoder, without new bridge", err)
	}
	if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(bm.ExpectedOutputs[0].BundlePath)
	if !bytes.Contains(files["formal/protected-candidates/"+name], []byte("Locally revised ordinary prose")) {
		t.Fatal("predecessor candidate/template lost")
	}
	// A private opaque-comment scaffold is essential to this regression: the
	// ordinary current source decoder must NOT explain the historical raw.
	u, err := c.Unit(id)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := u.Adapter.Restore(string(files["formal/protected-candidates/"+name])); err == nil && bytes.Equal(append(bytes.TrimRight(got, "\n"), '\n'), files["formal/candidates/"+name]) {
		t.Fatal("fixture did not reproduce private decoder difference")
	}
	var lineage struct {
		Bridge       i18n.GlossaryArchiveReference   `json:"bridge"`
		Dependencies []i18n.GlossaryArchiveReference `json:"dependencies"`
	}
	if err := json.Unmarshal(files["formal/representation-lineage/"+id+".json"], &lineage); err != nil {
		t.Fatal(err)
	}
	// Each changed immutable edge must fail even with the decoder memo warm.
	seen := map[string]bool{}
	for _, ref := range lineage.Dependencies {
		category := ""
		switch {
		case ref.Path == lineage.Bridge.Path:
			category = "bridge"
		case bytes.Contains([]byte(ref.Path), []byte("/validation/")):
			category = "validation"
		case bytes.Contains([]byte(ref.Path), []byte("/candidates/")):
			category = "candidate"
		case bytes.Contains([]byte(ref.Path), []byte("/raw-responses/")):
			category = "raw"
		case bytes.Contains([]byte(ref.Path), []byte("/generation-imports/")):
			category = "provenance"
		case bytes.Contains([]byte(ref.Path), []byte("generation-archives/")):
			category = "archive"
		case bytes.Contains([]byte(ref.Path), []byte("quality-check-snapshots/")):
			category = "snapshot"
		}
		if category == "" || seen[category] {
			continue
		}
		seen[category] = true
		t.Run("tamper-"+category, func(t *testing.T) {
			path := filepath.Join(root, ref.Path)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(bytes.Clone(before), ' '), 0600); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.WriteFile(path, before, 0600); err != nil {
					t.Fatal(err)
				}
			}()
			if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err == nil {
				t.Fatal("historical tamper accepted")
			}
		})
	}
	t.Run("wrong-predecessor-bridge-ref", func(t *testing.T) {
		found := false
		for _, ref := range lineage.Dependencies {
			if !strings.HasSuffix(ref.Path, "/manifest.json") || !strings.Contains(ref.Path, "retranslation-runs/") {
				continue
			}
			path := filepath.Join(root, ref.Path)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var manifest i18n.RetranslationBatchManifest
			if json.Unmarshal(before, &manifest) != nil {
				continue
			}
			for i := range manifest.Units {
				if manifest.Units[i].UnitID == id && manifest.Units[i].RepresentationBridge != nil {
					manifest.Units[i].RepresentationBridge.SHA256 = strings.Repeat("0", 64)
					found = true
				}
			}
			if !found {
				continue
			}
			bad, _ := json.MarshalIndent(manifest, "", "  ")
			writeFixture(t, root, ref.Path, append(bad, '\n'))
			defer writeFixture(t, root, ref.Path, before)
			if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err == nil {
				t.Fatal("wrong historical bridge reference accepted")
			}
			break
		}
		if !found {
			t.Fatal("no selected bridge-backed manifest fixture")
		}
	})
	t.Run("concurrent-historical-dependency", func(t *testing.T) {
		before, err := os.ReadFile(filepath.Join(root, lineage.Bridge.Path))
		if err != nil {
			t.Fatal(err)
		}
		resolver := c.BridgeResolver
		defer func() { c.BridgeResolver = resolver; writeFixture(t, root, lineage.Bridge.Path, before) }()
		c.BridgeResolver = func(root string, c *i18n.Catalog, mode string, ref i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
			if mode != "historical" {
				t.Fatal("old bridge became current authority")
			}
			result, err := build()
			if err == nil {
				writeFixture(t, root, ref.Path, append(bytes.Clone(before), ' '))
			}
			return result, err
		}
		if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err == nil {
			t.Fatal("concurrent bridge mutation accepted")
		}
	})
	if len(seen) != 7 {
		t.Fatal("missing tamper controls", seen)
	}
	// A successor retry inherits the same decoder and current QC finding; it
	// must not fall back to either the source protector or the old bridge QC.
	writeFixture(t, root, filepath.Join(exp.BatchPath, "raw-responses", name), []byte("broken successor response\n"))
	failed, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: exp.BatchID})
	if err != nil || failed.RestoreFailed != 1 {
		t.Fatal("successor failure fixture", failed, err)
	}
	retry, rm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: exp.BatchID, UnitID: id})
	if err != nil || rm.Attempt != 2 {
		t.Fatal("successor retry Bundle", err)
	}
	retryFiles, err := i18n.ReadTransportBundle(retry, 8192, 64<<20)
	if err != nil || !bytes.Equal(retryFiles["formal/representation-lineage/"+id+".json"], files["formal/representation-lineage/"+id+".json"]) || !bytes.Equal(retryFiles["formal/protected-candidates/"+name], files["formal/protected-candidates/"+name]) {
		t.Fatal("retry lineage changed", err)
	}
	staging := t.TempDir()
	revised := bytes.Replace(files["formal/protected-candidates/"+name], []byte("Locally revised ordinary prose"), []byte("Second locally revised prose"), 1)
	writeFixture(t, staging, name, revised)
	if _, err := i18n.ImportGenerationOutputDirectory(root, c, retry, "chatgpt", i18n.FormalChatGPTGenerationModel, staging); err != nil {
		t.Fatal("successor import", err)
	}
	result, err := i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: "zh-CN", BatchID: exp.BatchID, UnitID: id})
	if err != nil || result.ValidationPassed != 1 {
		t.Fatal("successor process", result, err)
	}
	candidate, err := os.ReadFile(filepath.Join(root, exp.BatchPath, result.Units[0].CandidatePath))
	if err != nil || !bytes.Contains(candidate, []byte("Second locally revised prose")) || !bytes.Contains(candidate, []byte("模块路径。")) {
		t.Fatal("successor restored bytes", err)
	}
	const next = "ordinary-second-successor"
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: next}); err != nil {
		t.Fatal("successor Snapshot", err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "independent-bridge-fixture-reviewer", Locale: "zh-CN", SnapshotID: next, PreviousSnapshotID: snapshot, UnitIDs: []string{id}, Rating: "B", Finding: "Third independently bound QC finding."}); err != nil {
		t.Fatal(err)
	}
	third, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{id}, AllowReexport: true, PreviousSnapshotID: next, UnitKind: i18n.UnitKindPage})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: third.BatchID}); err != nil {
		t.Fatal("transitive successor continuity", err)
	}

	// Historical A of an ordinary successor must retain its bridge dependencies
	// without replaying current command documentation as language authority.
	historyRoot := cloneBridgeSuccessorFixture(t, root)
	if err := os.RemoveAll(filepath.Join(historyRoot, third.BatchPath)); err != nil {
		t.Fatal(err)
	}
	historyCatalog := *c
	AttachPageReplacementInvocation(historyRoot, &historyCatalog)
	i18nSnapshot := i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "ordinary-history"}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(historyRoot, &historyCatalog, i18nSnapshot); err != nil {
		t.Fatal("ordinary history Snapshot", err)
	}
	if _, err := i18n.RecordQualityCheckResults(historyRoot, &historyCatalog, i18n.QualityCheckRecordOptions{ReviewerSession: "independent-bridge-fixture-reviewer", Locale: "zh-CN", SnapshotID: i18nSnapshot.SnapshotID, PreviousSnapshotID: next, UnitIDs: []string{id}, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(historyRoot, &historyCatalog, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: i18nSnapshot.SnapshotID}); err != nil {
		t.Fatal("ordinary historical A", err)
	}
	before, err := i18n.VerifyHistoricalQualityCheckFinalization(historyRoot, "zh-CN", i18nSnapshot.SnapshotID, &historyCatalog)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("surface-historical-representation", func(t *testing.T) {
		testSurfaceHistoricalProtectedPredecessor(t, historyRoot, &historyCatalog, &before.Finalization, id, true)
	})
	doc := "docs/TRANSLATION_TASK_SPEC.md"
	original, err := os.ReadFile(filepath.Join(historyRoot, doc))
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, historyRoot, doc, append(original, []byte("\nUnrelated future command documentation.\n")...))
	after, err := i18n.VerifyHistoricalQualityCheckFinalization(historyRoot, "zh-CN", i18nSnapshot.SnapshotID, &historyCatalog)
	if err != nil {
		t.Fatal("historical ordinary successor replayed current policy", err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("historical successor A changed")
	}

}

func cloneBridgeSuccessorFixture(t *testing.T, parent string) string {
	t.Helper()
	root := t.TempDir()
	if err := filepath.WalkDir(parent, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(parent, path)
		dst := filepath.Join(root, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		if d.Type()&os.ModeSymlink != 0 {
			t.Fatal("fixture symlink")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0600)
	}); err != nil {
		t.Fatal(err)
	}

	return root
}
