package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Readonly real 053 -> qc019 -> 054 authority, with all command output and
// reconstructed state isolated in TempDir. Never create the formal /tmp ZIP.
func TestBridgeSuccessorRealAuthority(t *testing.T) {
	out := os.Getenv("GO_TOUR_BRIDGE_SUCCESSOR_PROOF")
	if out == "" {
		t.Skip("explicit /tmp evidence directory required")
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
	for _, dir := range []string{"data/retranslation-runs/zh-CN", "data/quality-check-snapshots/zh-CN", "data/quality-check-review-evidence/zh-CN"} {
		if err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				rel, _ := filepath.Rel(root, path)
				copyFile(filepath.ToSlash(rel))
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	const bridge = "data/quality-check-reopens/zh-CN/learn-docs-qc018-ordinary-qc-transport.json"
	copyFile(bridge)
	b, _ := os.ReadFile(filepath.Join(temp, bridge))
	var receipt i18n.QualityCheckSurfaceReopen
	if err := json.Unmarshal(b, &receipt); err != nil || receipt.SameUnitBridge == nil {
		t.Fatal(err)
	}
	for _, ref := range receipt.SameUnitBridge.Dependencies {
		copyFile(ref.Path)
	}
	const batch = "data/retranslation-runs/zh-CN/chatgpt-zh-CN-054"
	b, err := os.ReadFile(filepath.Join(temp, batch, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if contentidentity.SHA256Hex(b) != "1d5e91de39980a00d5dfe6972c3320d0515299fdaf2af3aac269af503ca28483" {
		t.Fatal("054 identity changed")
	}
	var m i18n.RetranslationBatchManifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Units) != 3 {
		t.Fatal("054 scope", len(m.Units))
	}
	metrics := map[string]any{}
	sitecontent.AttachPageReplacementInvocation(temp, c)
	requests, builds := 0, 0
	resolve := c.BridgeResolver
	c.BridgeResolver = func(root string, c *i18n.Catalog, mode string, ref i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
		requests++
		if mode != "historical" {
			t.Fatal("old bridge used as current QC authorization")
		}
		return resolve(root, c, mode, ref, func() (*i18n.SameUnitBridgeResolution, error) { builds++; return build() })
	}
	done := contentidentity.BeginMemoization()
	start := time.Now()
	bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(temp, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: m.BatchID})
	metrics["export_seconds"] = time.Since(start).Seconds()
	done()
	if err != nil {
		t.Fatal("real copied 054 export", err)
	}
	if builds != 1 {
		t.Fatal("per-Unit historical reconstruction regression", builds)
	}
	metrics["historical_decoder_requests"] = requests
	metrics["historical_decoder_builds"] = builds
	sitecontent.AttachPageReplacementInvocation(temp, c)
	done = contentidentity.BeginMemoization()
	start = time.Now()
	current, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(temp, c, bundle)
	metrics["current_check_seconds"] = time.Since(start).Seconds()
	done()
	if err != nil || current.InputIdentitySHA256 != bm.InputIdentitySHA256 {
		t.Fatal("real copied current-check", err)
	}
	files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var s i18n.QualityCheckSnapshotManifest
	b, _ = os.ReadFile(filepath.Join(temp, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-019/manifest.json"))
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{}
	for _, r := range m.Units {
		if r.PreviousSnapshotID != "learn-docs-v1-qc-019" || r.RepresentationBridge != nil || r.RevisionFeedbackSource != "quality_check" {
			t.Fatal("semantic authority changed")
		}
		var selected i18n.QualityCheckSnapshotUnit
		for _, q := range s.Units {
			if q.UnitID == r.UnitID {
				selected = q
			}
		}
		if selected.SelectedBatchID != "chatgpt-zh-CN-053" {
			t.Fatal("wrong predecessor")
		}
		candidate, err := os.ReadFile(filepath.Join(temp, selected.CandidatePath))
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(r.InputPath)
		if !bytes.Equal(candidate, files["formal/candidates/"+name]) {
			t.Fatal("reviewed candidate changed")
		}
		u, err := c.Unit(r.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		normal, restoreErr := u.Adapter.Restore(string(files["formal/protected-candidates/"+name]))
		if restoreErr == nil && bytes.Equal(append(bytes.TrimRight(normal, "\n"), '\n'), candidate) {
			t.Fatal("real fixture did not reproduce historical decoder difference")
		}
		if err := i18n.ValidateTranslationUnitCandidate(temp, c, r.UnitID, "zh-CN", candidate); err != nil {
			t.Fatal("current canonical validation", err)
		}
		var validation i18n.RetranslationValidation
		vb, err := os.ReadFile(filepath.Join(temp, selected.ValidationPath))
		if err != nil || json.Unmarshal(vb, &validation) != nil {
			t.Fatal("selected validation", err)
		}
		rawPath := "data/retranslation-runs/zh-CN/" + selected.SelectedBatchID + "/" + validation.RawResponsePath
		selectedRaw, err := os.ReadFile(filepath.Join(temp, rawPath))
		if err != nil || !bytes.Equal(selectedRaw, files["formal/protected-candidates/"+name]) {
			t.Fatal("selected raw/template identity", err)
		}
		var lineage any
		if err := json.Unmarshal(files["formal/representation-lineage/"+r.UnitID+".json"], &lineage); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, map[string]any{"unit_id": r.UnitID, "previous_rating": r.PreviousRating, "previous_finding": r.PreviousFinding, "previous_snapshot": r.PreviousSnapshotID, "selected_predecessor_batch": selected.SelectedBatchID, "candidate_sha256": contentidentity.SHA256Hex(candidate), "ordinary_current_restore_error": fmtError(restoreErr), "ordinary_current_restored_sha256": contentidentity.SHA256Hex(normal), "exact_historical_restore": "PASS", "current_canonical_validation": "PASS", "selected_raw_path": rawPath, "selected_raw_sha256": contentidentity.SHA256Hex(selectedRaw), "lineage": lineage})
	}
	for path, want := range observed {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || contentidentity.SHA256Hex(b) != want {
			t.Fatal("formal input mutated", path)
		}
	}
	if _, err := os.Stat("/tmp/zh-CN-chatgpt-zh-CN-054-generation.zip"); !os.IsNotExist(err) {
		t.Fatal("formal ZIP unexpectedly exists")
	}
	metrics["units"] = rows
	metrics["bundle_identity"] = bm.InputIdentitySHA256
	metrics["formal_mutation"] = "none"
	metrics["observed_sha256"] = observed
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	b, _ = json.MarshalIndent(metrics, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "real-054-temp-proof.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("export=%v current-check=%v; historical decoder full builds=%d", metrics["export_seconds"], metrics["current_check_seconds"], builds)
}
func fmtError(err error) string {
	if err != nil {
		return err.Error()
	}
	return "nil; restored bytes differ"
}
