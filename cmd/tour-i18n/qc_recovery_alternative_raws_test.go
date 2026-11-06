package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Include passed retry/revalidation histories, not merely the latest batch
// result. Scope is the four exact physical sources; no other locale is read.
func TestQC018HistoricalPassedRawAlternatives(t *testing.T) {
	out := os.Getenv("GO_TOUR_QC_RECOVERY_PROOF_DIR")
	if out == "" {
		t.Skip("explicit readonly proof destination required")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = sitecontent.AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	if err = c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "per-unit-representation-proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	var prior []struct {
		Snapshot i18n.QualityCheckSnapshotUnit `json:"snapshot_unit"`
	}
	if err = json.Unmarshal(b, &prior); err != nil {
		t.Fatal(err)
	}
	observations := map[string]string{}
	read := func(path string) []byte {
		t.Helper()
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		observations[path] = contentidentity.SHA256Hex(b)
		return b
	}
	manifestPaths, err := filepath.Glob(filepath.Join(root, "data/retranslation-runs/zh-CN/*/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{}
	for _, old := range prior {
		s := old.Snapshot
		u, err := c.Unit(s.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		target := read(filepath.Join(root, s.CandidatePath))
		adapter := u.Adapter.(sitecontent.PageAdapter)
		cases := []map[string]any{}
		seen := map[string]bool{}
		alternatives := 0
		for _, path := range manifestPaths {
			b := read(path)
			var m i18n.RetranslationBatchManifest
			if err = json.Unmarshal(b, &m); err != nil {
				t.Fatal(err)
			}
			var record *i18n.RetranslationBatchUnit
			for j, r := range m.Units {
				if r.SourcePath == s.SourcePath && r.SourceSHA256 == s.SourceSHA256 {
					record = &m.Units[j]
				}
			}
			if record == nil {
				continue
			}
			base := filepath.Dir(path)
			for _, dir := range []string{"validation", "retries", "revalidation-history"} {
				walkErr := filepath.WalkDir(filepath.Join(base, dir), func(path string, d fs.DirEntry, e error) error {
					if os.IsNotExist(e) {
						return nil
					}
					if e != nil {
						return e
					}
					if d.IsDir() || filepath.Ext(path) != ".json" {
						return nil
					}
					b := read(path)
					var v i18n.RetranslationValidation
					if json.Unmarshal(b, &v) != nil || v.UnitID != record.UnitID || v.SourceSHA256 != s.SourceSHA256 || v.Status != "passed" || v.RawResponsePath == "" {
						return nil
					}
					raw := read(filepath.Join(base, v.RawResponsePath))
					key := m.BatchID + ":" + contentidentity.SHA256Hex(raw)
					if seen[key] {
						return nil
					}
					seen[key] = true
					currentID := record.UnitID == s.UnitID
					restored, e := adapter.Restore(string(raw))
					exact := e == nil && bytes.Equal(append(bytes.TrimRight(restored, "\n"), '\n'), append(bytes.TrimRight(target, "\n"), '\n'))
					if currentID && exact {
						alternatives++
					}
					row := map[string]any{"batch": m.BatchID, "validation_path": path, "attempt": v.Attempt, "raw_sha256": contentidentity.SHA256Hex(raw), "same_formal_unit_id": currentID, "ordinary_decoder_exact_candidate": exact}
					if e != nil {
						row["restore_error"] = e.Error()
					}
					cases = append(cases, row)
					return nil
				})
				if walkErr != nil {
					t.Fatal(walkErr)
				}
			}
		}
		if alternatives != 0 {
			t.Fatal("a passed ordinary raw can express this exact reviewed candidate", s.UnitID)
		}
		rows = append(rows, map[string]any{"unit_id": s.UnitID, "source_path": s.SourcePath, "passed_raw_representations": cases, "directly_reusable_ordinary_raws": alternatives})
	}
	// Explicit machine materializations in historical Snapshots have their own
	// candidate identity. None may be substituted for this exact reviewed target.
	snapshots, err := filepath.Glob(filepath.Join(root, "data/quality-check-snapshots/zh-CN/*/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	machineRows := []map[string]any{}
	for _, path := range snapshots {
		var m i18n.QualityCheckSnapshotManifest
		if err = json.Unmarshal(read(path), &m); err != nil {
			t.Fatal(err)
		}
		for _, u := range m.Units {
			if u.RepresentationReconciliation == nil {
				continue
			}
			for _, p := range prior {
				s := p.Snapshot
				if u.SourcePath == s.SourcePath && u.SourceSHA256 == s.SourceSHA256 {
					if u.CandidateSHA256 == s.CandidateSHA256 {
						t.Fatal("an equivalent machine materialization needs explicit investigation", path)
					}
					machineRows = append(machineRows, map[string]any{"snapshot": m.SnapshotID, "source_path": u.SourcePath, "candidate_sha256": u.CandidateSHA256, "same_reviewed_candidate": false, "authority": u.RepresentationReconciliation})
				}
			}
		}
	}
	for path, sha := range observations {
		b, e := os.ReadFile(path)
		if e != nil || contentidentity.SHA256Hex(b) != sha {
			t.Fatal("readonly source changed", path)
		}
	}
	b, err = json.MarshalIndent(map[string]any{"units": rows, "machine_snapshot_candidates": machineRows, "observed_refs_sha256": observations, "result": "PASS", "repository_mutation": "none"}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(out, "historical-alternative-raw-proof.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
