package i18n

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

type replacementThirdCard struct{ authorizationMachineCard }

func (a replacementThirdCard) ProtectedInput(g *Glossary) (string, []string, error) {
	s, tokens, err := a.authorizationMachineCard.ProtectedInput(g)
	return "⟬THIRD⟭" + s, append(tokens, "⟬THIRD⟭"), err
}
func (a replacementThirdCard) Restore(s string) ([]byte, error) {
	if len(s) < len("⟬THIRD⟭") || s[:len("⟬THIRD⟭")] != "⟬THIRD⟭" {
		return nil, fmt.Errorf("third boundary missing")
	}
	return a.authorizationMachineCard.Restore(s[len("⟬THIRD⟭"):])
}

// The shared reader receives an explicit provider-verified B→C bridge. The
// source provider's actual canonical Shell lineage is tested separately.
func TestHistoricalReconciliationEventWorldFinalization(t *testing.T) {
	root, b, original, plan, ref, receiptBytes := authorizationMachineFixture(t)
	r, err := b.ReplacementResolver(root, b, plan.Receipt.Options.HistoricalDocuments)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = FinalizeQualityCheck(root, r.HistoricalCatalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: original.SnapshotID}); err != nil {
		t.Fatal("A", err)
	}
	second := futureSnapshot(t, root, b, "event-qc002")
	if _, _, err = FinalizeQualityCheck(root, b, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: second.SnapshotID, PreviousSnapshotID: original.SnapshotID}); err != nil {
		t.Fatal("B", err)
	}
	baseline, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", second.SnapshotID, b)
	if err != nil {
		t.Fatal(err)
	}
	d := b.TranslationPackages[0]
	d.AuthoritySHA256 = sum([]byte("third-package"))
	d.Units = append([]AdaptedTranslationUnit(nil), d.Units...)
	ids := []string{}
	for i := range d.Units {
		u := &d.Units[i]
		u.Unit.ID = sum([]byte(u.Unit.ID + "-third"))
		u.Adapter = replacementThirdCard{u.Adapter.(authorizationMachineCard)}
		ids = append(ids, u.Unit.ID)
	}
	c := &Catalog{}
	if err = c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err = c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	bridge := []byte("exact provider-verified B to C authority\n")
	writeSnapshotSource(t, root, "bridge.json", bridge)
	bridgeRef := GlossaryArchiveReference{"bridge.json", sum(bridge)}
	originalResolver := b.ReplacementResolver
	c.ReplacementResolver = func(root string, today *Catalog, ref GlossaryArchiveReference) (*TranslationUnitReplacementResolution, error) {
		out, err := originalResolver(root, b, ref)
		if err != nil {
			return nil, err
		}
		// Provider validation is deliberately strict even for same-source impostors.
		if !reflect.DeepEqual(today.TranslationPackages, c.TranslationPackages) {
			return nil, fmt.Errorf("wrong terminal descriptor")
		}
		event := *b
		event.Adapted = append([]AdaptedTranslationUnit(nil), b.Adapted...)
		out.EventCatalog = &event
		out.SuccessorReferences = []GlossaryArchiveReference{bridgeRef}
		return out, nil
	}
	a, err := replacementResolutionInWorld(root, c, &TranslationUnitReplacementContext{HistoricalDocuments: plan.Receipt.Options.HistoricalDocuments}, true)
	if err != nil {
		t.Fatal(err)
	}
	a2, err := replacementResolutionInWorld(root, c, &TranslationUnitReplacementContext{HistoricalDocuments: plan.Receipt.Options.HistoricalDocuments}, true)
	if err != nil || a.EventCatalog == a2.EventCatalog {
		t.Fatal("invocation world not isolated", err)
	}
	if _, err = replacementResolution(root, c, &TranslationUnitReplacementContext{HistoricalDocuments: plan.Receipt.Options.HistoricalDocuments}); err == nil {
		t.Fatal("current reconciliation accepted historical B as C")
	}
	after, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", second.SnapshotID, c)
	if err != nil || !bytes.Equal(mustJSON(baseline.Finalization), mustJSON(after.Finalization)) {
		t.Fatal("B immutable proof changed", err)
	}
	checkpointGenerate(t, root, c, ids)
	third := futureSnapshot(t, root, c, "event-qc004")
	if _, _, err = FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: third.SnapshotID, PreviousSnapshotID: second.SnapshotID}); err == nil {
		t.Fatal("old A silently carried onto fresh C without independent QC")
	}
	futureRecord(t, root, c, third.SnapshotID, second.SnapshotID, "A", "", ids)
	for _, path := range []string{third.Units[0].CandidatePath, third.Units[0].ValidationPath} {
		before, readErr := readSnapshotReferencedFile(root, path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		writeSnapshotSource(t, root, path, append(bytes.Clone(before), ' '))
		_, _, staleErr := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: third.SnapshotID, PreviousSnapshotID: second.SnapshotID})
		writeSnapshotSource(t, root, path, before)
		if staleErr == nil {
			t.Fatal("C active validation accepted changed candidate/validation", path)
		}
	}
	if _, _, err = FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: third.SnapshotID, PreviousSnapshotID: second.SnapshotID}); err != nil {
		t.Fatal("C with B predecessor", err)
	}
	for _, kind := range []string{"bridge missing", "bridge bytes", "machine receipt", "source identity", "wrong intermediate", "wrong package"} {
		t.Run(kind, func(t *testing.T) {
			resolver := c.ReplacementResolver
			c.ReplacementResolver = func(root string, today *Catalog, frozen GlossaryArchiveReference) (*TranslationUnitReplacementResolution, error) {
				out, err := resolver(root, today, frozen)
				if err != nil {
					return nil, err
				}
				switch kind {
				case "bridge missing":
					out.SuccessorReferences = nil
				case "source identity", "wrong intermediate", "wrong package":
					for id, p := range out.Units {
						switch kind {
						case "source identity":
							p.SourceSHA256 = sum([]byte("drift"))
						case "wrong intermediate":
							p.CurrentUnitID = ids[0]
						case "wrong package":
							p.PackageSHA256 = sum([]byte("wrong"))
						}
						out.Units[id] = p
						break
					}
				}
				return out, nil
			}
			defer func() { c.ReplacementResolver = resolver }()
			if kind == "bridge bytes" {
				writeSnapshotSource(t, root, "bridge.json", []byte("tamper"))
				defer writeSnapshotSource(t, root, "bridge.json", bridge)
			}
			if kind == "machine receipt" {
				writeSnapshotSource(t, root, ref.Path, append(bytes.Clone(receiptBytes), ' '))
				defer writeSnapshotSource(t, root, ref.Path, receiptBytes)
			}
			if _, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", second.SnapshotID, c); err == nil {
				t.Fatal("accepted", kind)
			}
		})
	}
}
