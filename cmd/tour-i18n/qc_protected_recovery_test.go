package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Read real authority; exercise the existing response decoder only in a sparse
// TempDir. No model output is generated and no repository record/export runs.
func TestQC018ProtectedRepresentationRecoveryProof(t *testing.T) {
	out := os.Getenv("GO_TOUR_QC_RECOVERY_PROOF_DIR")
	if out == "" {
		t.Skip("explicit readonly proof destination required")
	}
	phase := recoveryProofTimings(t, out)
	phase("fixture-setup")
	root, temp, c, m051, _, observed := copiedCommentRevisionRealFixture(t, false)
	phase("frozen-authority-and-projection")
	// This multi-Unit proof owns one immutable source Catalog/TempDir, just as
	// one CLI invocation does. Reuse only exact-byte transformations and the
	// source projection; each revalidation still reconstructs its authority and
	// performs its own fresh dependency check before writing copied evidence.
	done := contentidentity.BeginMemoization()
	defer done()
	read := func(path string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(path, err)
		}
		observed[path] = contentidentity.SHA256Hex(b)
		return b
	}
	save := func(name string, v any) {
		t.Helper()
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(out, name), append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	var snapshot i18n.QualityCheckSnapshotManifest
	if err := json.Unmarshal(read("data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-018/manifest.json"), &snapshot); err != nil {
		t.Fatal(err)
	}
	var qc i18n.QualityCheckResults
	if err := json.Unmarshal(read("data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-018/quality-check-results.json"), &qc); err != nil {
		t.Fatal(err)
	}
	var m052 i18n.RetranslationBatchManifest
	if err := json.Unmarshal(read("data/retranslation-runs/zh-CN/chatgpt-zh-CN-052/manifest.json"), &m052); err != nil {
		t.Fatal(err)
	}
	reviewer := read("data/quality-check-review-evidence/zh-CN/20261021-zh-CN-learn-docs-qc-018-re-qc-page-001.md")
	if contentidentity.SHA256Hex(reviewer) != "5abbd2a0709dff73632dc9f0824f5efbb31fc1e2a63bd446aba263d4ed5d9d56" {
		t.Fatal("review evidence drift")
	}
	oldBundle, err := os.ReadFile("/tmp/zh-CN-chatgpt-zh-CN-052-generation.zip")
	if err != nil {
		t.Fatal(err)
	}
	if contentidentity.SHA256Hex(oldBundle) != "809260c21aba4e89c5d949c8b0164f0b0189554fd173299eeef95dbd37ded4b5" {
		t.Fatal("052 ZIP changed")
	}
	files052, err := i18n.ReadTransportBundle(oldBundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	imported := read("data/retranslation-runs/zh-CN/" + m051.BatchID + "/generation-imports/initial-manifest.json")
	var receipt i18n.GenerationResultBundleManifest
	if err = json.Unmarshal(imported, &receipt); err != nil {
		t.Fatal(err)
	}
	archive := read("data/generation-archives/zh-CN/" + receipt.GenerationBundleSHA256 + ".zip")
	if contentidentity.SHA256Hex(archive) != receipt.GenerationBundleSHA256 {
		t.Fatal("imported invocation drift")
	}
	files051, err := i18n.ReadTransportBundle(archive, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var original i18n.GenerationBundleManifest
	if err = json.Unmarshal(files051["manifest.json"], &original); err != nil {
		t.Fatal(err)
	}
	if err = i18n.ValidateTransportBundleInventory(files051, original.Files, true); err != nil {
		t.Fatal(err)
	}
	if contentidentity.SHA256Hex(files051["manifest.json"]) != receipt.GenerationManifest.SHA256 {
		t.Fatal("import provenance manifest drift")
	}
	// Revalidate only copied predecessor Units, never 052 or a real candidate.
	batch := "data/retranslation-runs/zh-CN/" + m051.BatchID
	if err = filepath.WalkDir(filepath.Join(root, batch), func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		b := read(filepath.ToSlash(rel))
		dst := filepath.Join(temp, rel)
		if e = os.MkdirAll(filepath.Dir(dst), 0755); e != nil {
			return e
		}
		return os.WriteFile(dst, b, 0600)
	}); err != nil {
		t.Fatal(err)
	}
	frozen := sitecontent.HistoricalPageRepresentations{Package: "learn-docs-v1"}
	for _, d := range c.TranslationPackages {
		if d.ID == frozen.Package {
			frozen.PackageSHA256 = d.AuthoritySHA256
			for i, b := range d.Units {
				a := b.Adapter.(sitecontent.PageAdapter)
				if snapshot.Units[i].UnitID != b.Unit.ID || snapshot.Units[i].SourceSHA256 != b.Unit.SourceSHA256 {
					t.Fatal("formal representation delta", i)
				}
				frozen.Documents = append(frozen.Documents, a.Document)
			}
		}
	}
	if len(frozen.Documents) != 49 || len(m052.Units) != 4 {
		t.Fatal("exact-set mismatch")
	}
	frozenBytes, _ := json.MarshalIndent(frozen, "", "  ")
	const frozenPath = "engineering/qc018-source-owned-representations.json"
	if err = os.MkdirAll(filepath.Join(temp, "engineering"), 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(temp, frozenPath), frozenBytes, 0600); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachPageReplacementInvocation(temp, c)
	resolution, err := c.ReplacementResolver(temp, c, i18n.GlossaryArchiveReference{Path: frozenPath, SHA256: contentidentity.SHA256Hex(frozenBytes)})
	if err != nil {
		t.Fatal(err)
	}
	save("frozen-source-owned-representations.json", frozen)
	rows := []map[string]any{}
	for i, r := range m052.Units {
		phase("per-unit-binding")
		index := []int{20, 25, 27, 47}[i]
		s := snapshot.Units[index-1]
		if s.UnitID != r.UnitID || s.SelectedBatchID != m051.BatchID || s.Attempt != 1 {
			t.Fatal("snapshot exact identity", index)
		}
		finding := i18n.QualityCheckResult{}
		for _, q := range qc.Results {
			if q.UnitID == r.UnitID {
				finding = q
			}
		}
		if finding.Index != index || finding.Rating != r.PreviousRating || finding.Finding != r.PreviousFinding || finding.Rating != []string{"B", "C", "B", "B"}[i] {
			t.Fatal("QC finding binding", index)
		}
		unit, err := c.Unit(r.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		a := unit.Adapter.(sitecontent.PageAdapter)
		candidate := read(s.CandidatePath)
		validation := read(s.ValidationPath)
		if contentidentity.SHA256Hex(candidate) != s.CandidateSHA256 || contentidentity.SHA256Hex(validation) != s.ValidationSHA256 {
			t.Fatal("selected hashes")
		}
		var v i18n.RetranslationValidation
		if err = json.Unmarshal(validation, &v); err != nil {
			t.Fatal(err)
		}
		rawPath := batch + "/" + v.RawResponsePath
		raw := read(rawPath)
		name := filepath.Base(r.InputPath)
		if !bytes.Equal(raw, files052["formal/protected-candidates/"+name]) || !bytes.Equal(candidate, files052["formal/candidates/"+name]) {
			t.Fatal("Bundle did not freeze selected representations")
		}
		matched := false
		for _, e := range original.ExpectedOutputs {
			if e.UnitID == r.UnitID && e.InstallPath == v.RawResponsePath {
				for _, p := range receipt.Outputs {
					if p.BundlePath == e.BundlePath && p.SHA256 == contentidentity.SHA256Hex(raw) && p.Size == int64(len(raw)) {
						matched = true
					}
				}
			}
		}
		if !matched || v.Status != "passed" || v.Attempt != s.Attempt || v.UnitID != r.UnitID || v.SourceSHA256 != s.SourceSHA256 {
			t.Fatal("raw/attempt/import provenance")
		}
		var p i18n.CommentRevisionTransport
		if err = json.Unmarshal(files051["formal/comment-revision-transport/"+name+".json"], &p); err != nil {
			t.Fatal(err)
		}
		if p.Schema != "go-learning/comment-revision-transport/v1" || p.UnitID != r.UnitID || len(p.Slots) != []int{1, 2, 1, 17}[i] {
			t.Fatal("restricted transport identity")
		}
		opaqueEnd := 0
		slotIDs := []string{}
		var template strings.Builder
		oldCandidate := files051["formal/candidates/"+name]
		for j, slot := range p.Slots {
			if slot.Start < opaqueEnd || slot.End > len(oldCandidate) || p.Values[j] != string(oldCandidate[opaqueEnd:slot.Start]) {
				t.Fatal("opaque coordinate drift")
			}
			template.WriteString(p.Boundaries[j] + slot.Text)
			opaqueEnd = slot.End
			slotIDs = append(slotIDs, slot.SlotID)
		}
		template.WriteString(p.Boundaries[len(p.Slots)])
		if template.String() != p.Text || p.Values[len(p.Slots)] != string(oldCandidate[opaqueEnd:]) {
			t.Fatal("non-comment editable interval")
		}
		generic, _, err := a.ProtectedInput(nil)
		if err != nil {
			t.Fatal(err)
		}
		generic = string(append(bytes.TrimRight([]byte(generic), "\n"), '\n'))
		if contentidentity.SHA256Hex([]byte(generic)) != r.InputSHA256 {
			t.Fatal("formal protected input drift")
		}
		_, ordinaryErr := a.Restore(string(raw))
		if ordinaryErr == nil {
			t.Fatal("restricted raw unexpectedly uses ordinary decoder")
		}
		_, rebaseErr := a.RebaseRevisionPredecessor(a, raw)
		if rebaseErr == nil {
			t.Fatal("legacy replacement unexpectedly accepts transport raw")
		}
		replacement := resolution.Units[r.UnitID]
		if replacement.OldUnitID != replacement.CurrentUnitID || replacement.OldInputSHA256 != replacement.CurrentInputSHA256 {
			t.Fatal("unexpected real replacement delta")
		}
		phase("revalidate-" + r.UnitID)
		revalidated, e := i18n.RevalidateRetranslationCandidate(temp, c, i18n.RetranslationRevalidateOptions{Locale: "zh-CN", BatchID: m051.BatchID, UnitID: r.UnitID})
		phase("historical-raw-scan")
		if e != nil || revalidated.Status != "passed" {
			t.Fatal("copied predecessor exact restore/current validator", index, e)
		}
		copyCandidate, e := os.ReadFile(filepath.Join(temp, s.CandidatePath))
		if e != nil || !bytes.Equal(candidate, copyCandidate) {
			t.Fatal("reviewed target changed")
		}
		alternatives := []map[string]any{}
		ordinaryAlternative := 0
		manifests, e := filepath.Glob(filepath.Join(root, "data/retranslation-runs/zh-CN/*/manifest.json"))
		if e != nil {
			t.Fatal(e)
		}
		for _, path := range manifests {
			b, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			var manifest i18n.RetranslationBatchManifest
			if e = json.Unmarshal(b, &manifest); e != nil {
				t.Fatal(e)
			}
			var record *i18n.RetranslationBatchUnit
			for j, q := range manifest.Units {
				if q.SourcePath == s.SourcePath && q.SourceSHA256 == s.SourceSHA256 {
					record = &manifest.Units[j]
				}
			}
			if record == nil {
				continue
			}
			base := filepath.Dir(path)
			b, e = os.ReadFile(filepath.Join(base, "result.json"))
			if os.IsNotExist(e) {
				continue
			}
			if e != nil {
				t.Fatal(e)
			}
			var result i18n.RetranslationProcessResult
			if e = json.Unmarshal(b, &result); e != nil {
				t.Fatal(e)
			}
			for _, q := range result.Units {
				if q.UnitID != record.UnitID || q.Status != "passed" {
					continue
				}
				b, e = os.ReadFile(filepath.Join(base, q.CandidatePath))
				if e != nil {
					t.Fatal(e)
				}
				row := map[string]any{"batch_id": manifest.BatchID, "unit_id": q.UnitID, "same_candidate": bytes.Equal(b, candidate)}
				if bytes.Equal(b, candidate) {
					vb, e := os.ReadFile(filepath.Join(base, q.ValidationPath))
					if e != nil {
						t.Fatal(e)
					}
					var vv i18n.RetranslationValidation
					if e = json.Unmarshal(vb, &vv); e != nil {
						t.Fatal(e)
					}
					rb, e := os.ReadFile(filepath.Join(base, vv.RawResponsePath))
					if e != nil {
						t.Fatal(e)
					}
					restored, e := a.Restore(string(rb))
					same := e == nil && bytes.Equal(append(bytes.TrimRight(restored, "\n"), '\n'), candidate)
					row["ordinary_decoder_exact_restore"] = same
					row["raw_sha256"] = contentidentity.SHA256Hex(rb)
					row["feedback_source"] = record.RevisionFeedbackSource
					if same {
						ordinaryAlternative++
					}
				}
				alternatives = append(alternatives, row)
			}
		}
		if ordinaryAlternative != 0 {
			t.Fatal("a reusable ordinary representation exists")
		}
		rows = append(rows, map[string]any{"index": index, "snapshot_unit": s, "source_sha256": unit.SourceSHA256, "finding": finding, "reviewer_evidence_sha256": contentidentity.SHA256Hex(reviewer), "predecessor_raw_path": rawPath, "predecessor_raw_sha256": contentidentity.SHA256Hex(raw), "import_provenance_sha256": contentidentity.SHA256Hex(imported), "projection_schema": p.Schema, "editable_comment_slot_ids": slotIDs, "non_comment_editable_bytes": 0, "formal_unit_id_delta": false, "formal_input_sha_delta": false, "ordinary_decoder_error": ordinaryErr.Error(), "legacy_rebase_error": rebaseErr.Error(), "replacement": replacement, "qc_replacement_rejecting_condition": "p.OldUnitID == p.CurrentUnitID", "exact_restore_in_tempdir": "PASS", "active_validation_in_tempdir": "PASS", "alternative_passed_candidates": alternatives, "ordinary_equivalent_raw_count": ordinaryAlternative})
		save("per-unit-representation-proof.json", rows)
		t.Logf("index=%d exact restore/current validation PASS; formal delta=0; no ordinary equivalent raw", index)
	}
	phase("fresh-repository-hash-check")
	for path, sha := range observed {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || contentidentity.SHA256Hex(b) != sha {
			t.Fatal("repository bytes changed", path)
		}
	}
	save("readonly-reconciliation-preflight.json", map[string]any{"result": "FAILED", "recorded": false, "repository_mutation": "none", "formal_representation_delta": 0, "complete_frozen_documents": len(frozen.Documents), "affected_units": rows, "canonical_cli_readonly_available": false, "reason": "existing QC replacement requires unequal formal Unit IDs and input hashes; transport-only projection has neither"})
}

// Test-only phase accounting. CPU labels cover synchronous authority work;
// rusage additionally includes GC/worker CPU. This never changes validation.
func recoveryProofTimings(t *testing.T, out string) func(string) {
	t.Helper()
	type sample struct {
		Phase string  `json:"phase"`
		Wall  float64 `json:"wall_seconds"`
		User  float64 `json:"user_seconds"`
		Sys   float64 `json:"sys_seconds"`
	}
	var rows []sample
	var last time.Time
	var previous syscall.Rusage
	name := ""
	seconds := func(v syscall.Timeval) float64 { return float64(v.Sec) + float64(v.Usec)/1e6 }
	mark := func(next string) {
		now := time.Now()
		var usage syscall.Rusage
		if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
			t.Fatal(err)
		}
		if name != "" {
			row := sample{name, now.Sub(last).Seconds(), seconds(usage.Utime) - seconds(previous.Utime), seconds(usage.Stime) - seconds(previous.Stime)}
			rows = append(rows, row)
			t.Logf("phase=%s wall=%.3fs user=%.3fs sys=%.3fs", name, row.Wall, row.User, row.Sys)
		}
		name, last, previous = next, now, usage
		pprof.SetGoroutineLabels(pprof.WithLabels(context.Background(), pprof.Labels("recovery-phase", next)))
	}
	t.Cleanup(func() {
		mark("")
		pprof.SetGoroutineLabels(context.Background())
		b, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.MkdirAll(out, 0755); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(out, "phase-timings.json"), append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	})
	return mark
}
