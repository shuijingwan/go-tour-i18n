package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReconciledFrozenInputCurrentAssessment(t *testing.T) {
	root, c, _, plan, ref, receiptBytes := authorizationMachineFixture(t)
	const snapshot = "accepted-reconciliation"
	futureSnapshot(t, root, c, snapshot)
	for _, id := range c.TranslationScope {
		futureRecord(t, root, c, snapshot, "", "A", "", []string{id})
	}
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	history, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", snapshot, c)
	if err != nil {
		t.Fatal(err)
	}
	checkpointGlossary(t, root, "preferred:\n  Go local: Go 本地化\n", "reconciliation-current-review")
	// An unbound later compatibility artifact must not enter the frozen world.
	// Shared current assessment uses the canonical per-TU semantic primitive;
	// it does not extend this unrelated legacy inventory's coverage.
	writeSnapshotSource(t, root, "data/glossary-compatibility/zh-CN/unbound-current.json", []byte("{unbound\n"))
	again, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", snapshot, c)
	if err != nil || !reflect.DeepEqual(history, again) {
		t.Fatal("later current evidence changed frozen proof", err)
	}
	a, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot)
	if err != nil || !a.Ready() || a.Compatible != len(plan.Receipt.Units) {
		t.Fatalf("canonical current assessment replayed legacy compatibility: %+v: %v", a, err)
	}
	for i, x := range a.Units {
		if x.Snapshot.UnitID == plan.Receipt.Units[i].Predecessor.UnitID || x.Validation != "passed" || x.Snapshot.RepresentationReconciliation == nil || !reflect.DeepEqual(x.Reasons, []string{"changed_source_term_absent"}) {
			t.Fatal("lost explicit old/current mapping or current validation", x)
		}
	}
	if _, err := RequireCurrentTranslationAuthorization(root, c, "zh-CN", snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "data/translation-authorizations")); !os.IsNotExist(err) {
		t.Fatal("readonly assessment saved checkpoint", err)
	}
	for _, scope := range []string{"seo:test", "ui:test", "structured:test", "*"} {
		p := PrepareGlossaryCompatibility(root, "zh-CN", c)
		p.tuOnly = true
		if r := p.Resolve(plan.Receipt.GlossarySHA256, a.Glossary.SHA256, scope); r.Status == "compatible" || r.Status == "exact" {
			t.Fatal("TU proof authorized derived consumer", scope, r)
		}
	}
	for _, kind := range []string{"missing-receipt", "receipt-tamper", "wrong-glossary", "wrong-mapping", "candidate-tamper"} {
		t.Run(kind, func(t *testing.T) {
			path, before := ref.Path, receiptBytes
			bad := *plan.Receipt
			bad.Units = append([]RepresentationMachineUnit(nil), bad.Units...)
			switch kind {
			case "missing-receipt":
				if err := os.Remove(filepath.Join(root, path)); err != nil {
					t.Fatal(err)
				}
			case "receipt-tamper":
				writeSnapshotSource(t, root, path, append(bytes.Clone(before), ' '))
			case "wrong-glossary":
				bad.GlossarySHA256 = a.Glossary.SHA256
				bad.Identity = machineReceiptIdentity(bad)
				writeSnapshotSource(t, root, path, append(mustJSON(bad), '\n'))
			case "wrong-mapping":
				bad.Units[0].Replacement.OldUnitID = bad.Units[0].Replacement.CurrentUnitID
				bad.Identity = machineReceiptIdentity(bad)
				writeSnapshotSource(t, root, path, append(mustJSON(bad), '\n'))
			case "candidate-tamper":
				path = a.Units[0].Snapshot.CandidatePath
				var err error
				before, err = readSnapshotReferencedFile(root, path)
				if err != nil {
					t.Fatal(err)
				}
				writeSnapshotSource(t, root, path, append(bytes.Clone(before), ' '))
			}
			defer writeSnapshotSource(t, root, path, before)
			if got, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot); err == nil && got.Ready() {
				t.Fatal("invalid successor authorized", kind)
			}
		})
	}
	// A real semantic impact stays affected even with an exact historical proof.
	checkpointGlossary(t, root, "  message: 消息\n", "reconciliation-semantic-review")
	semantic, err := AssessCurrentTranslation(root, c, "zh-CN", snapshot)
	if err != nil || semantic.Ready() || semantic.Affected != len(a.Units) {
		t.Fatal("semantic impact bypassed", semantic, err)
	}
	for _, u := range semantic.Units {
		if !strings.Contains(strings.Join(u.Reasons, " "), "preferred_source_term_present") {
			t.Fatal("missing actual semantic reason", u)
		}
	}
	after, err := readSnapshotReferencedFile(root, ref.Path)
	if err != nil || !bytes.Equal(after, receiptBytes) {
		t.Fatal("historical receipt changed", err)
	}
}
