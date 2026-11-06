package i18n

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Opt-in diagnosis uses the canonical readers against real immutable evidence.
// All output is written to /tmp; it never records authorization or promotion.
func TestTourSuccessorLifecycleReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_SUCCESSOR_LIFECYCLE_READONLY") != "1" {
		t.Skip("explicit real Tour successor diagnosis")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := findCurrentTargetSuccessor(root, "zh-CN", catalog)
	if err != nil || proof == nil {
		t.Fatal("actual current successor", proof, err)
	}
	compatibility, err := historicalSuccessorCompatibility(root, "zh-CN", catalog, proof.oldTargets)
	if err != nil {
		t.Fatal(err)
	}
	f, err := verifySuccessorFinalization(root, "zh-CN", "qc-008", catalog, compatibility)
	if err != nil {
		t.Fatal("actual requested finalization", err)
	}
	original, err := readQualityCheckSnapshot(root, "zh-CN", proof.evidence.SnapshotID, false)
	if err != nil {
		t.Fatal(err)
	}
	type difference struct {
		Position                int
		UnitID                  string
		Fields                  map[string][2]any
		Requested, Installation QualityCheckSnapshotUnit
	}
	diffs := []difference{}
	for i, u := range f.Units {
		prior := original.Units[i]
		if qualityCheckSnapshotIdentityMatches(u.Snapshot, prior) {
			continue
		}
		var a, b map[string]any
		_ = json.Unmarshal(mustJSON(u.Snapshot), &a)
		_ = json.Unmarshal(mustJSON(prior), &b)
		fields := map[string][2]any{}
		for _, name := range []string{"unit_id", "unit_kind", "selected_batch_id", "source_path", "source_sha256", "candidate_path", "candidate_sha256", "validation_path", "validation_sha256", "attempt", "representation_reconciliation"} {
			if !reflect.DeepEqual(a[name], b[name]) {
				fields[name] = [2]any{a[name], b[name]}
			}
		}
		diffs = append(diffs, difference{i + 1, u.UnitID, fields, u.Snapshot, prior})
	}
	diagnostic, err := DiagnoseQualityCheckAuthorization(root, catalog, "zh-CN", "qc-008")
	if err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"proof_snapshot_id": proof.evidence.SnapshotID, "proof_kind": proof.evidence.Kind, "current_requested_snapshot_id": "qc-008", "original_snapshot_id": original.SnapshotID, "differences": diffs, "diagnostic": diagnostic}
	canonical, err := canonicalTranslationUnitCandidatePath("zh-CN", mustSuccessorDiagnosticUnit(t, catalog, "concurrency/11"))
	if err != nil {
		t.Fatal(err)
	}
	installed, err := readCompatibilityFile(root, canonical)
	if err != nil {
		t.Fatal(err)
	}
	report["canonical_path"], report["canonical_sha256"] = canonical, sum(installed)
	for _, d := range diffs {
		if d.UnitID != "concurrency/11" {
			continue
		}
		for name, u := range map[string]QualityCheckSnapshotUnit{"requested": d.Requested, "installation": d.Installation} {
			raw, err := readCompatibilityFile(root, u.CandidatePath)
			if err != nil {
				t.Fatal(err)
			}
			report[name+"_projected_candidate_sha256"] = sum(canonicalizeCandidateEOF(raw))
		}
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	const path = "/tmp/tour-successor-runtime-diagnosis.json"
	if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("proof_snapshot=%s original_snapshot=%s mismatch_count=%d authorized=%t evidence=%s sha256=%s", proof.evidence.SnapshotID, original.SnapshotID, len(diffs), diagnostic.CanonicalAuthorized, path, sum(append(b, '\n')))
}

func mustSuccessorDiagnosticUnit(t *testing.T, c *Catalog, id string) *TranslationUnit {
	t.Helper()
	u, err := c.Unit(id)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
