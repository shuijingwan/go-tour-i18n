package i18n

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestRecordedReconciliationSameGlossaryMembershipSuccessor(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, event, original, plan, ref, receiptBytes := authorizationMachineFixtureSize(t, 49)
	resolution, err := event.ReplacementResolver(root, event, plan.Receipt.Options.HistoricalDocuments)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, resolution.HistoricalCatalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: original.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	prior := futureSnapshot(t, root, event, "recorded-member-predecessor")
	if _, _, err := FinalizeQualityCheck(root, event, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: prior.SnapshotID, PreviousSnapshotID: original.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	current, added := expandedMachinePackage(t, root, event, 2)
	// No bridge/shared-environment owner participates in this reproduction.
	current.BridgeResolver = func(string, *Catalog, string, GlossaryArchiveReference, func() (*SameUnitBridgeResolution, error)) (*SameUnitBridgeResolution, error) {
		t.Fatal("bridge owner must not participate in machine scope selection")
		return nil, nil
	}
	addition, err := HistoricalTranslationPackageCatalog(current, added.ID, added.AuthoritySHA256)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, u := range added.Units {
		ids = append(ids, u.Unit.ID)
	}
	checkpointGenerate(t, root, addition, ids)
	addedSnapshot := futureSnapshot(t, root, addition, "recorded-independent-addition")
	for _, id := range ids {
		futureRecord(t, root, addition, addedSnapshot.SnapshotID, "", "A", "", []string{id})
	}
	if _, _, err := FinalizeQualityCheck(root, addition, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: addedSnapshot.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	declaration := TranslationPackageMembership{Predecessors: []TranslationPackagePredecessor{
		{ID: event.TranslationPackages[0].ID, AuthoritySHA256: event.TranslationPackages[0].AuthoritySHA256},
		{ID: added.ID, AuthoritySHA256: added.AuthoritySHA256},
	}}
	data := mustJSON(declaration)
	writeSnapshotSource(t, root, "fixtures/recorded-membership.json", data)
	declaration.References = []GlossaryArchiveReference{{Path: "fixtures/recorded-membership.json", SHA256: sum(data)}}
	current.TranslationPackages[0].Membership = &declaration
	// This is the first 51-Unit QC lineage. New members have their own complete
	// independent A, rather than retroactively appearing in the original 49.
	first, _, err := CreateQualityCheckCandidateSnapshot(root, current, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "recorded-expanded-qc", ContentPackage: current.TranslationPackages[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	futureRecord(t, root, current, first.SnapshotID, prior.SnapshotID, "A", "", []string{first.Units[24].UnitID})
	next, _, err := CreateQualityCheckCandidateSnapshot(root, current, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "recorded-expanded-successor", ContentPackage: current.TranslationPackages[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := readCompatibilityFile(root, "locales/zh-CN/glossary.yaml")
	if err != nil || sum(glossary) != plan.Receipt.GlossarySHA256 || first.GlossarySHA256 != next.GlossarySHA256 {
		t.Fatal("fixture must keep exact same glossary", err)
	}
	preflight, err := BuildQualityCheckPreflight(root, current, QualityCheckPreflightOptions{Locale: "zh-CN", SnapshotID: next.SnapshotID, PreviousSnapshotID: first.SnapshotID})
	if err != nil || preflight.CarryForwardCount != 51 || preflight.QualityCheckRequiredCount != 0 || preflight.RevisionRequiredCount != 0 {
		t.Fatal("same-glossary 49→51 successor preflight", preflight, err)
	}
	// Creating a NEW reconciliation from a smaller predecessor remains illegal.
	if _, err := prepareMachineReconciliation(root, current, plan.Receipt.Options); err == nil {
		t.Fatal("historical scope authorized new expanded reconciliation")
	}
	selection := &machineSelection{receipt: plan.Receipt, unit: plan.Receipt.Units[0], ref: ref, ctx: &machineReconciliationContext{receipt: plan.Receipt}}
	u := selection.snapshot(1)
	_, selected, err := machineSnapshotEvidence(root, current, "zh-CN", u, nil)
	if err != nil {
		t.Fatal("recorded current reader", err)
	}
	if selected.ctx.historical || selected.ctx.compatibility.historicalSHA != "" || len(selected.ctx.snapshot.Units) != 49 || len(selected.ctx.effective) != 49 || len(selected.ctx.resolution.HistoricalCatalog.TranslationScope) != 49 {
		t.Fatal("historical scope accidentally selected historical glossary/currentness", selected.ctx.historical)
	}
	after, err := readSnapshotReferencedFile(root, ref.Path)
	if err != nil || !bytes.Equal(after, receiptBytes) {
		t.Fatal("immutable receipt changed", err)
	}
	t.Log("historical_snapshot=49 historical_catalog=49 current_catalog=51 effective_QC=49 same_glossary=true historical_glossary_selected=false; successor_QC_carry=51; bridge_calls=0")
}

func TestRecordedReconciliationIncompleteQCWithCurrentGlossary(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	// Keep this original QC unfinalized, so an honestly incomplete result set
	// reaches the completeness gate rather than failing a sealed-proof SHA first.
	root, event, original, plan, ref, _ := authorizationMachineFixtureSize(t, 49)
	current, _ := expandedMachinePackage(t, root, event, 2)
	name := filepath.ToSlash(strings.TrimPrefix(qualityCheckResultsPath(root, "zh-CN", original.SnapshotID), root+string(filepath.Separator)))
	before, err := readSnapshotReferencedFile(root, name)
	if err != nil {
		t.Fatal(err)
	}
	var results QualityCheckResults
	if err := json.Unmarshal(before, &results); err != nil {
		t.Fatal(err)
	}
	results.Results, results.ResultCount = results.Results[:48], 48
	writeSnapshotSource(t, root, name, mustJSON(results))
	selection := &machineSelection{receipt: plan.Receipt, unit: plan.Receipt.Units[0], ref: ref, ctx: &machineReconciliationContext{receipt: plan.Receipt}}
	if _, _, err := machineSnapshotEvidence(root, current, "zh-CN", selection.snapshot(1), nil); err == nil || !strings.Contains(err.Error(), "complete initial independent QC required for machine reconciliation") {
		t.Fatal("49 Snapshot / 48 effective QC must still fail exact gate", err)
	}
	t.Log("historical_snapshot=49 historical_catalog=49 current_catalog=51 effective_QC=48 historical_glossary_selected=false; exact completeness gate rejects")
}
