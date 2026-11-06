package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"
)

// A branch-boundary fixture, not a Surface/A authorization shortcut: native
// passed old/current invocations have different IDs and protected inputs at
// the same source. The historical record's replacement metadata is test-only;
// mapping integrity is independently checked below by the native mapping reader.
func replacementResponseFixture(t *testing.T) (string, *Catalog, *Catalog, *TranslationUnit, QualityCheckSnapshotUnit, QualityCheckSnapshotUnit, RetranslationBatchUnit, []byte) {
	t.Helper()
	root, base, _ := futureContentFixture(t)
	d := base.TranslationPackages[0]
	d.AuthoritySHA256 = sum([]byte("replacement response fixture source authority"))
	d.Units = append([]AdaptedTranslationUnit(nil), d.Units[:1]...)
	card := d.Units[0].Adapter.(futureCardAdapter)
	d.Units[0].Adapter = authorizationMachineCard{card}
	old := &Catalog{}
	if err := old.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := old.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	oldID := d.Units[0].Unit.ID
	checkpointGenerate(t, root, old, []string{oldID})
	prior := futureSnapshot(t, root, old, "replacement-response-old")
	futureRecord(t, root, old, prior.SnapshotID, "", "A", "", []string{oldID})
	if _, _, err := FinalizeQualityCheck(root, old, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: prior.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	d.Units = append([]AdaptedTranslationUnit(nil), d.Units...)
	d.Units[0].Unit.ID = sum([]byte(oldID + "-replacement"))
	d.Units[0].Adapter = frozenCardAdapter{card}
	current := &Catalog{}
	if err := current.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := current.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	u, err := current.Unit(d.Units[0].Unit.ID)
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := readCompatibilityFile(root, glossaryReviewGlossaryPath("zh-CN"))
	if err != nil {
		t.Fatal(err)
	}
	text, _, err := u.Adapter.ProtectedInput(nil)
	if err != nil {
		t.Fatal(err)
	}
	invocation := mustJSON(frozenCardInvocation{u.SourceSHA256, glossary, text})
	writeSnapshotSource(t, root, "fixtures/replacement-invocation.json", invocation)
	writeSnapshotSource(t, root, "fixtures/replacement-output.txt", []byte(text))
	origin := TranslationUnitGenerationOrigin{GlossaryArchiveReference{"fixtures/replacement-invocation.json", sum(invocation)}, GlossaryArchiveReference{"fixtures/replacement-output.txt", sum([]byte(text))}}
	e, err := ExportRetranslationBatch(root, current, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{u.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if err := LandExistingGenerationBatch(root, current, "zh-CN", e.BatchID, map[string]TranslationUnitGenerationOrigin{u.ID: origin}, map[string][]byte{u.ID: []byte(text)}); err != nil {
		t.Fatal(err)
	}
	if r, err := ProcessRetranslationBatch(root, current, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || r.ValidationPassed != 1 {
		t.Fatal(r, err)
	}
	accepted := futureSnapshot(t, root, current, "replacement-response-event").Units[0]
	batchDir := filepath.Join(root, e.BatchPath)
	m, err := readRetranslationProcessManifest(batchDir, "zh-CN", e.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	r := &m.Units[0]
	r.PreviousSnapshotID = prior.SnapshotID
	r.RevisionFeedbackSource = "surface_review"
	r.RevisionAuthorizationID = "synthetic-replacement-response-event"
	r.PreviousRating = "A"
	r.PreviousFinding = "synthetic exact finding"
	r.PreviousReviewPath = "fixtures/replacement-review.json"
	r.PreviousReviewSHA256 = sum([]byte("synthetic independent Surface evidence"))
	writeSnapshotSource(t, root, e.BatchPath+"/manifest.json", mustJSON(m))
	candidate, err := readSnapshotReferencedFile(root, accepted.CandidatePath)
	if err != nil {
		t.Fatal(err)
	}
	return root, old, current, u, prior.Units[0], accepted, *r, candidate
}

func TestReplacementResponseDoesNotRequireSameIDInheritance(t *testing.T) {
	root, old, current, u, prior, accepted, record, candidate := replacementResponseFixture(t)
	g, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if prior.UnitID == record.UnitID || !hasInheritedBridgePredecessor(record) {
		t.Fatal("fixture lacks actual non-same-ID Surface boundary")
	}
	current.BridgeResolver = func(string, *Catalog, string, GlossaryArchiveReference, func() (*SameUnitBridgeResolution, error)) (*SameUnitBridgeResolution, error) {
		t.Fatal("non-same-ID record tried an arbitrary bridge history")
		return nil, nil
	}
	current.ReplacementResolver = func(string, *Catalog, GlossaryArchiveReference) (*TranslationUnitReplacementResolution, error) {
		t.Fatal("response selector tried an unrelated replacement history")
		return nil, nil
	}
	if _, _, err := acceptedResponseCatalog(root, current, "zh-CN", record); err == nil || !bytes.Contains([]byte(err.Error()), []byte("accepted response lacks exact historical Unit/source")) {
		t.Fatal("rejected eager predicate not reproduced", err)
	}
	if _, lineage, err := inheritedBridgeOutput(root, current, g, "zh-CN", record); err != nil || lineage != nil {
		t.Fatal("non-same-ID record acquired arbitrary inherited history", lineage, err)
	}
	if err := verifyAcceptedFrozenInput(root, "zh-CN", u, accepted, candidate, sum(mustReadResponseGlossary(t, root)), current); err != nil {
		t.Fatal("exact invocation codec was replaced by eager same-ID check", err)
	}
	oldEvidence, err := readSnapshotUnitRepositoryEvidence(root, old, "zh-CN", prior)
	if err != nil {
		t.Fatal(err)
	}
	event := TranslationUnitReplacement{OldUnitID: prior.UnitID, CurrentUnitID: u.ID, Package: old.TranslationPackages[0].ID, PackageSHA256: old.TranslationPackages[0].AuthoritySHA256, SourcePath: record.SourcePath, SourceSHA256: record.SourceSHA256, StableIndex: 1, Route: old.Adapted[0].Route, OldInputSHA256: oldEvidence.record.InputSHA256, CurrentInputSHA256: record.InputSHA256, ReconciliationSHA256: sum([]byte("synthetic exact replacement mapping"))}
	next := event
	next.OldUnitID = u.ID
	next.OldInputSHA256 = record.InputSHA256
	next.CurrentUnitID = sum([]byte(u.ID + "-machine"))
	next.CurrentInputSHA256 = sum([]byte("later machine protected representation"))
	latest := event
	latest.CurrentUnitID = next.CurrentUnitID
	latest.CurrentInputSHA256 = next.CurrentInputSHA256
	reopen := &QualityCheckSurfaceReopen{Units: []QualityCheckSurfaceReopenUnit{{Index: 1, UnitID: u.ID, Snapshot: prior, Replacement: &event}}}
	resolution := &TranslationUnitReplacementResolution{HistoricalCatalog: old, Units: map[string]TranslationUnitReplacement{prior.UnitID: latest}}
	bind := func(path, sha string) ([]byte, error) {
		b, e := readSnapshotReferencedFile(root, path)
		if e != nil || sha != "" && sum(b) != sha {
			return nil, fmt.Errorf("fixture immutable reference mismatch: %s", path)
		}
		return b, nil
	}
	if err := verifyHistoricalSurfaceMapping(root, "zh-CN", record, reopen, resolution, &next, bind); err != nil {
		t.Fatal("replacement then machine mapping topology", err)
	}
	bad := next
	bad.OldUnitID = "unrelated"
	if err := verifyHistoricalSurfaceMapping(root, "zh-CN", record, reopen, resolution, &bad, bind); err == nil {
		t.Fatal("arbitrary mapping accepted")
	}
	for _, name := range []string{"raw", "input", "candidate", "validation", "missing-required-snapshot"} {
		t.Run(name, func(t *testing.T) {
			r := cloneCompositionFixture(t, root)
			path := accepted.CandidatePath
			switch name {
			case "raw":
				path = "fixtures/replacement-output.txt"
			case "input":
				path = "data/retranslation-runs/zh-CN/" + accepted.SelectedBatchID + "/" + record.InputPath
			case "validation":
				path = accepted.ValidationPath
			case "missing-required-snapshot":
				path = "data/quality-check-snapshots/zh-CN/" + record.PreviousSnapshotID + "/manifest.json"
			}
			if name == "missing-required-snapshot" {
				removeComposedFixture(t, filepath.Join(r, path))
			} else {
				b, e := readSnapshotReferencedFile(r, path)
				if e != nil {
					t.Fatal(e)
				}
				writeSnapshotSource(t, r, path, append(b, ' '))
			}
			observedCandidate := candidate
			if name == "candidate" {
				var err error
				observedCandidate, err = readSnapshotReferencedFile(r, accepted.CandidatePath)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := verifyAcceptedFrozenInput(r, "zh-CN", u, accepted, observedCandidate, sum(mustReadResponseGlossary(t, r)), current); err == nil {
				t.Fatal("tampered provenance accepted", name)
			}
		})
	}
}
