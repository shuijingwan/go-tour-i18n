package main

import (
	"reflect"
	"testing"
)

func TestSameUnitBridgeCanonicalCommand(t *testing.T) {
	args := []string{"--content-package", "registered-fixture", "--locale", "zh-CN", "--bridge-id", "same-unit-bridge", "--previous-snapshot-id", "reviewed-qc", "--previous-snapshot-sha256", "snapshot-sha", "--reviewer-evidence", "engineering/reviewer.md", "--reviewer-evidence-sha256", "review-sha", "--unit-id", "first", "--unit-id", "second"}
	o, err := parseSameUnitBridgeOptions(args)
	if err != nil || o.Record || o.ContentPackage != "registered-fixture" || !reflect.DeepEqual(o.UnitIDs, []string{"first", "second"}) {
		t.Fatal(o, err)
	}
	recorded, err := parseSameUnitBridgeOptions(append(append([]string{}, args...), "--record"))
	if err != nil || !recorded.Record {
		t.Fatal(recorded, err)
	}
	for _, bad := range [][]string{args[2:], append(append([]string{}, args...), "--content-package", "second"), append(append([]string{}, args...), "--apply"), append(append([]string{}, args...), "positional")} {
		if _, err := parseSameUnitBridgeOptions(bad); err == nil {
			t.Fatal("incomplete/mixed command accepted", bad)
		}
	}
	pkg, rest, err := parseTranslationContentPackageArgs(append([]string{"quality-check", "representation-bridge"}, args...))
	if err != nil || pkg != "" || len(rest) != len(args)+2 {
		t.Fatal("required package lost to global parser", err)
	}
	export, _, err := parseRetranslationExportOptions([]string{"--locale", "zh-CN", "--generator", "chatgpt", "--allow-reexport", "--previous-snapshot-id", "reviewed-qc", "--representation-bridge-id", "same-unit-bridge", "--id", "first"})
	if err != nil || export.RepresentationBridgeID != "same-unit-bridge" {
		t.Fatal(export, err)
	}
}

// Surface describes the predecessor representation, not a new feedback mode
// or permission flag. The existing command still takes independent reviewed
// QC Snapshot/evidence coordinates and remains readonly unless --record.
func TestSameUnitBridgeSurfacePredecessorCommandContract(t *testing.T) {
	args := []string{"--content-package", "registered-pages", "--locale", "fixture-locale", "--bridge-id", "independent-qc-bridge", "--previous-snapshot-id", "surface-output-reviewed-by-qc", "--previous-snapshot-sha256", "snapshot-sha", "--reviewer-evidence", "evidence/independent-qc.md", "--reviewer-evidence-sha256", "review-sha", "--unit-id", "same-formal-unit"}
	o, err := parseSameUnitBridgeOptions(args)
	if err != nil || o.Record || o.PreviousSnapshotID != "surface-output-reviewed-by-qc" || o.ReviewerEvidence.Path != "evidence/independent-qc.md" {
		t.Fatal("independent QC command coordinates", o, err)
	}
	for _, flag := range []string{"--surface-reopen-id", "--surface-review-id", "--apply", "--allow-reexport"} {
		if _, err := parseSameUnitBridgeOptions(append(append([]string{}, args...), flag, "old-Surface-authority")); err == nil {
			t.Fatal("Surface authority/implicit publication flag accepted", flag)
		}
	}
	if recorded, err := parseSameUnitBridgeOptions(append(append([]string{}, args...), "--record")); err != nil || !recorded.Record {
		t.Fatal("explicit write-once record mode changed", err)
	}
}
