package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Diagnostic only: no export, publication, policy change, or inferred mapping.
func TestCurrentValidationPredecessorMigrationReadonly(t *testing.T) {
	out := os.Getenv("GO_TOUR_CURRENT_VALIDATION_MIGRATION_PROBE")
	if out == "" {
		t.Skip("explicit /tmp evidence destination required")
	}
	if !strings.HasPrefix(filepath.Clean(out), "/tmp/") {
		t.Fatal("evidence must be in /tmp")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	read := func(p string) []byte {
		t.Helper()
		b, e := readRegular(root, p)
		if e != nil {
			t.Fatal(p, e)
		}
		return b
	}
	decode := func(b []byte, v any) {
		t.Helper()
		if e := json.Unmarshal(b, v); e != nil {
			t.Fatal(e)
		}
	}
	c, e := i18n.BuildSourceCatalog(root)
	if e != nil {
		t.Fatal(e)
	}
	if e = AttachRegisteredTranslationPackagesReadonly(root, c); e != nil {
		t.Fatal(e)
	}
	if e = c.UseTranslationPackage("learn-docs-v1"); e != nil {
		t.Fatal(e)
	}
	var m i18n.RetranslationBatchManifest
	decode(read("data/retranslation-runs/zh-CN/chatgpt-zh-CN-064/manifest.json"), &m)
	// The canonical flattened reader verifies the actual selected A origins,
	// including machine reconciliation, source, input, candidate and raw lineage.
	proof, e := i18n.ReadHistoricalTranslationIntegrity(root, c, m.Locale, m.Units[0].PreviousSnapshotID)
	if e != nil {
		t.Fatal("historical integrity", e)
	}
	readiness, e := i18n.InspectCurrentValidationPredecessors(root, c, m.Locale, m.Units)
	if e != nil {
		t.Fatal("canonical predecessor inspection", e)
	}
	ready := map[string]i18n.CurrentValidationPredecessor{}
	for _, r := range readiness {
		ready[r.UnitID] = r
	}
	var s i18n.QualityCheckSnapshotManifest
	decode(read("data/quality-check-snapshots/"+m.Locale+"/"+m.Units[0].PreviousSnapshotID+"/manifest.json"), &s)
	var impact struct {
		Units []struct {
			SourcePath string          `json:"source_path"`
			Current    string          `json:"current_unit_id"`
			Old        string          `json:"snapshot_representation_unit_id"`
			Labels     map[string]bool `json:"labels"`
		} `json:"units"`
	}
	decode(read("docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json"), &impact)
	var originalRepresentations HistoricalPageRepresentations
	const frozenProfile = "docs/protected-representation-evidence/qc014-historical-representations.json"
	for _, ref := range proof.References {
		if ref.Path == frozenProfile {
			b, err := i18n.ReadHistoricalTranslationReference(root, c, proof, ref)
			if err != nil {
				t.Fatal(err)
			}
			decode(b, &originalRepresentations)
		}
	}
	if len(originalRepresentations.Documents) == 0 {
		t.Fatal("formal historical source representation not in verified lineage")
	}
	var rows []map[string]any
	counts := map[string]int{}
	for _, r := range m.Units {
		row := map[string]any{"unit_id": r.UnitID, "source_path": r.SourcePath, "previous_snapshot_id": r.PreviousSnapshotID, "current_unit_id": r.UnitID, "formal_impact_matched": false, "representation_reconciliation_required": false, "unknown": false, "introduced_machine_identities": []string{}, "exact_blocker": "", "existing_rebase_result": "not-needed"}
		for _, x := range impact.Units {
			if x.SourcePath == r.SourcePath && x.Current == r.UnitID {
				row["formal_impact_matched"] = true
				row["representation_reconciliation_required"] = x.Labels["representation_reconciliation_required"]
				row["unknown"] = x.Labels["unknown"]
				row["formal_impact_snapshot_representation_unit_id"] = x.Old
			}
		}
		var q *i18n.QualityCheckSnapshotUnit
		for i := range s.Units {
			if s.Units[i].UnitID == r.UnitID {
				q = &s.Units[i]
			}
		}
		if q == nil {
			t.Fatal("exact Snapshot Unit missing", r.UnitID)
		}
		row["snapshot_representation_unit_id"] = q.UnitID
		row["source_profile_comparison"] = "formal-impact frozen source profile; selected Snapshot/response codec is separately bound below"
		row["predecessor_batch"] = q.SelectedBatchID
		row["attempt"] = q.Attempt
		row["candidate_sha256"] = q.CandidateSHA256
		candidate := read(q.CandidatePath)
		if digest(candidate) != q.CandidateSHA256 {
			t.Fatal("candidate SHA mismatch")
		}
		vb := read(q.ValidationPath)
		if digest(vb) != q.ValidationSHA256 {
			t.Fatal("validation SHA mismatch")
		}
		var v i18n.RetranslationValidation
		decode(vb, &v)
		base := "data/retranslation-runs/" + m.Locale + "/" + q.SelectedBatchID
		var oldManifest i18n.RetranslationBatchManifest
		decode(read(base+"/manifest.json"), &oldManifest)
		var old i18n.RetranslationBatchUnit
		for _, u := range oldManifest.Units {
			if u.SourcePath == r.SourcePath {
				old = u
			}
		}
		if old.UnitID == "" {
			t.Fatal("original manifest Unit missing")
		}
		row["historical_manifest_unit_id"] = old.UnitID
		rawPath := base + "/" + v.RawResponsePath
		if q.RepresentationReconciliation != nil {
			rb := read(q.RepresentationReconciliation.Path)
			if digest(rb) != q.RepresentationReconciliation.SHA256 {
				t.Fatal("reconciliation SHA mismatch")
			}
			var reconciliation i18n.RepresentationReconciliation
			decode(rb, &reconciliation)
			for _, u := range reconciliation.Units {
				if u.Replacement.CurrentUnitID == r.UnitID {
					rawPath = u.CurrentRaw.Path
					row["original_raw_sha256"] = u.OriginalRaw.SHA256
					row["reconciliation_authority"] = *q.RepresentationReconciliation
					row["unknown_resolved_machine"] = u.UnknownResolvedMachine
					row["semantic_authority"] = u.SemanticAuthority
				}
			}
		}
		raw := read(rawPath)
		row["historical_raw_sha256"] = digest(raw)
		row["historical_raw_path"] = rawPath
		current, e := c.Unit(r.UnitID)
		if e != nil {
			t.Fatal(e)
		}
		a, e := replacementPageAdapter(current.Adapter)
		if e != nil {
			t.Fatal(e)
		}
		input, _, e := a.ProtectedInput(nil)
		if e != nil {
			t.Fatal(e)
		}
		for _, before := range originalRepresentations.Documents {
			if before.Path != r.SourcePath || before.SourceSHA != r.SourceSHA256 {
				continue
			}
			oldPieces, err := protectionPieces(before.Units[0])
			if err != nil {
				t.Fatal(err)
			}
			newPieces, err := protectionPieces(a.Document.Units[0])
			if err != nil {
				t.Fatal(err)
			}
			introduced := []string{}
			for _, atom := range newPieces {
				if atom.token == nil {
					continue
				}
				covered := false
				for _, old := range oldPieces {
					if old.token != nil && atom.start >= old.start && atom.end <= old.end {
						covered = true
						break
					}
				}
				if !covered {
					introduced = append(introduced, atom.token.Raw)
				}
			}
			row["introduced_machine_identities"] = introduced
			row["old_representation_identity"] = identity(before)
			row["current_representation_identity"] = identity(a.Document)
		}
		row["current_input_sha256"] = digest(bridgeEOF([]byte(input)))
		row["old_input_sha256"] = old.InputSHA256
		row["representation_identity_changed"] = digest(bridgeEOF([]byte(input))) != old.InputSHA256
		got, restoreErr := a.Restore(string(raw))
		ordinary := restoreErr == nil && bytes.Equal(bridgeEOF(got), candidate)
		row["current_ordinary_restore"] = ordinary
		importPath := base + "/generation-imports/initial-manifest.json"
		if q.Attempt > 1 {
			importPath = fmt.Sprintf("%s/generation-imports/%s-attempt-%03d-manifest.json", base, strings.TrimSuffix(filepath.Base(old.InputPath), filepath.Ext(old.InputPath)), q.Attempt)
		}
		imported, importErr := readRegular(root, importPath)
		var files map[string][]byte
		if importErr == nil {
			var receipt i18n.GenerationResultBundleManifest
			decode(imported, &receipt)
			archivePath := "data/generation-archives/" + m.Locale + "/" + receipt.GenerationBundleSHA256 + ".zip"
			archive, archiveErr := readRegular(root, archivePath)
			row["import_archive_exists"] = archiveErr == nil
			if archiveErr != nil {
				row["import_archive_observation"] = "not physically present: " + archiveErr.Error()
			} else {
				if digest(archive) != receipt.GenerationBundleSHA256 {
					t.Fatal("archive SHA mismatch")
				}
				files, e = i18n.ReadTransportBundle(archive, 8192, 64<<20)
				if e != nil {
					t.Fatal(e)
				}
				row["import_receipt"] = i18n.GlossaryArchiveReference{Path: importPath, SHA256: digest(imported)}
				row["import_archive"] = i18n.GlossaryArchiveReference{Path: archivePath, SHA256: digest(archive)}
			}
		} else if !os.IsNotExist(importErr) {
			t.Fatal(importErr)
		} else {
			row["import_provenance"] = "legacy-no-import-receipt; canonical frozen A integrity verified"
		}
		class := "UNAFFECTED_CURRENT_RESTORE_PASS"
		if row["exact_blocker"] != "" {
			class = "OTHER_FAIL_CLOSED"
		}
		if !ordinary {
			row["exact_blocker"] = fmt.Sprint(restoreErr)
			class = "OTHER_FAIL_CLOSED"
			member := "formal/surface-transport-authority/" + r.UnitID + ".json"
			if b := files[member]; len(b) > 0 {
				var authority i18n.SurfaceTransportAuthority
				decode(b, &authority)
				for _, binding := range authority.Bindings {
					if binding.UnitID != r.UnitID {
						continue
					}
					decoder, e := a.ReadSurfaceTransportProjection(binding.Projection.Codec)
					if e != nil {
						row["existing_rebase_result"] = e.Error()
						continue
					}
					historical, e := decoder.Restore(string(raw))
					if e != nil || !bytes.Equal(bridgeEOF(historical), candidate) {
						row["existing_rebase_result"] = fmt.Sprintf("sealed restore: %v", e)
						continue
					}
					next, template, _, e := a.BridgeSurfaceRevision(decoder, raw, candidate)
					if e != nil {
						row["existing_rebase_result"] = e.Error()
						continue
					}
					check, e := next.Restore(string(template))
					if e != nil || !bytes.Equal(bridgeEOF(check), candidate) {
						row["existing_rebase_result"] = fmt.Sprintf("bridge exact restore: %v", e)
						continue
					}
					class = "DETERMINISTIC_REBASE_PASS"
					row["existing_rebase_result"] = "source-owned sealed Surface codec composition PASS"
					row["template_sha256"] = digest(template)
					row["transport_kind"] = binding.Projection.Kind
				}
			}
		}
		canonical := ready[r.UnitID]
		row["current_ordinary_restore"] = canonical.OrdinaryRestore
		row["raw_current_page_adapter_restore"] = ordinary
		row["exact_blocker"] = canonical.Blocker
		row["canonical_predecessor_proof"] = canonical
		class = canonical.RecoveryClass
		if canonical.RecoveryClass == "DETERMINISTIC_REBASE_PASS" {
			row["existing_rebase_result"] = "PASS: canonical sealed response lineage + source-owned composition + exact imported raw/attempt binding"
		} else if canonical.RecoveryClass != "UNAFFECTED_CURRENT_RESTORE_PASS" {
			row["existing_rebase_result"] = "BLOCKED: " + canonical.Blocker
		}
		if class == "OTHER_FAIL_CLOSED" && row["unknown"] == true {
			class = "KNOWN_SYS08_UNKNOWN"
		}
		row["recovery_class"] = class
		counts[class]++
		rows = append(rows, row)
	}
	if len(rows) != 32 {
		t.Fatal("expected exact 32-Page diagnostic")
	}
	result := map[string]any{"report_type": "engineering-diagnostic-only", "formal_mutation": "none", "counts": counts, "historical_reference_count": len(proof.References), "units": rows}
	b, e := json.MarshalIndent(result, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(out, append(b, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
	t.Logf("units=%d counts=%v", len(rows), counts)
}
