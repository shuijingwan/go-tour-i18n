package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type authorizationMachineCard struct{ futureCardAdapter }

const authorizationMachineBoundary = "⟬CARD_REPRESENTATION⟭"

func (a authorizationMachineCard) ProtectedInput(g *Glossary) (string, []string, error) {
	s, tokens, err := a.futureCardAdapter.ProtectedInput(g)
	return authorizationMachineBoundary + s, append(tokens, authorizationMachineBoundary), err
}
func (a authorizationMachineCard) Restore(raw string) ([]byte, error) {
	if strings.Count(raw, authorizationMachineBoundary) != 1 || !strings.HasPrefix(raw, authorizationMachineBoundary) {
		return nil, fmt.Errorf("machine boundary mismatch")
	}
	return a.futureCardAdapter.Restore(strings.TrimPrefix(raw, authorizationMachineBoundary))
}
func (a authorizationMachineCard) ReconcileRepresentation(_ TranslationUnitAdapter, raw []byte) ([]byte, error) {
	return append([]byte(authorizationMachineBoundary), raw...), nil
}

// A three-Unit synthetic representation proof exercises the actual receipt
// reader and machine verifier with a warm pure-transform scope. All fixture
// generation/QC and mutations are confined to t.TempDir, never repository data.
func authorizationMachineFixture(t *testing.T) (string, *Catalog, *QualityCheckSnapshotManifest, *RepresentationReconciliationPlan, GlossaryArchiveReference, []byte) {
	t.Helper()
	return authorizationMachineFixtureSize(t, 3)
}

func authorizationMachineFixtureSize(t *testing.T, count int) (string, *Catalog, *QualityCheckSnapshotManifest, *RepresentationReconciliationPlan, GlossaryArchiveReference, []byte) {
	t.Helper()
	root, old, ids := futureContentFixture(t)
	if count != len(ids) {
		d := old.TranslationPackages[0]
		base := d.Units
		d.Units, ids = nil, nil
		for i := 0; i < count; i++ {
			u := base[i%len(base)]
			u.Unit.ID = fmt.Sprintf("expanded-card-%d", i+1)
			u.Unit.SourcePath = fmt.Sprintf("fixtures/expanded-card-%d.xml", i+1)
			u.StableIndex, u.Route = i+1, fmt.Sprintf("/fixture/expanded/%d", i+1)
			u.WorkingSet = fmt.Sprintf("initial-%d", i+1)
			writeSnapshotSource(t, root, u.Unit.SourcePath, u.Unit.Source)
			d.Units, ids = append(d.Units, u), append(ids, u.Unit.ID)
		}
		old = &Catalog{}
		if err := old.RegisterTranslationPackage(d); err != nil {
			t.Fatal(err)
		}
		if err := old.UseTranslationPackage(d.ID); err != nil {
			t.Fatal(err)
		}
	}
	old.TranslationPackages[0].AuthoritySHA256 = sum([]byte("machine fixture package authority"))
	checkpointGenerate(t, root, old, ids)
	const snapshot = "authorization-machine-original"
	s := futureSnapshot(t, root, old, snapshot)
	for _, id := range ids {
		futureRecord(t, root, old, snapshot, "", "A", "", []string{id})
	}
	current := &Catalog{}
	d := old.TranslationPackages[0]
	d.Units = append([]AdaptedTranslationUnit(nil), d.Units...)
	mappings := map[string]TranslationUnitReplacement{}
	for i := range d.Units {
		u := &d.Units[i]
		before, _, _ := u.Adapter.ProtectedInput(nil)
		u.Unit.ID = sum([]byte(u.Unit.ID + "-representation"))
		u.Adapter = authorizationMachineCard{u.Adapter.(futureCardAdapter)}
		after, _, _ := u.Adapter.ProtectedInput(nil)
		mappings[ids[i]] = TranslationUnitReplacement{OldUnitID: ids[i], CurrentUnitID: u.Unit.ID, Package: d.ID, PackageSHA256: d.AuthoritySHA256, SourcePath: u.Unit.SourcePath, SourceSHA256: u.Unit.SourceSHA256, StableIndex: u.StableIndex, Route: u.Route, OldInputSHA256: sum(canonicalizeRetranslationArtifactEOF([]byte(before))), CurrentInputSHA256: sum(canonicalizeRetranslationArtifactEOF([]byte(after))), ReconciliationSHA256: sum([]byte("exact fixture representation relation"))}
	}
	if err := current.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := current.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	frozen := mustJSON(mappings)
	writeSnapshotSource(t, root, "frozen.json", frozen)
	frozenRef := GlossaryArchiveReference{"frozen.json", sum(frozen)}
	current.ReplacementResolver = func(root string, _ *Catalog, ref GlossaryArchiveReference) (*TranslationUnitReplacementResolution, error) {
		b, err := readSnapshotReferencedFile(root, ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			return nil, fmt.Errorf("fixture frozen authority mismatch")
		}
		var units map[string]TranslationUnitReplacement
		if err := json.Unmarshal(b, &units); err != nil {
			return nil, err
		}
		return &TranslationUnitReplacementResolution{HistoricalCatalog: old, Units: units}, nil
	}
	rows := []representationImpactUnit{}
	for _, id := range ids {
		p := mappings[id]
		rows = append(rows, representationImpactUnit{CurrentUnitID: p.CurrentUnitID, OldUnitID: id, SourcePath: p.SourcePath, SourceSHA256: p.SourceSHA256, Labels: map[string]bool{"representation_reconciliation_required": true, "true_language_defect": false, "unknown": false}})
	}
	impact := mustJSON(representationImpact{Units: rows})
	writeSnapshotSource(t, root, "impact.json", impact)
	manifest, err := readSnapshotReferencedFile(root, filepath.ToSlash(filepath.Join("data/quality-check-snapshots/zh-CN", snapshot, "manifest.json")))
	if err != nil {
		t.Fatal(err)
	}
	options := RepresentationReconciliationOptions{Locale: "zh-CN", Package: d.ID, ReconciliationID: "authorization-machine", PreviousSnapshotID: snapshot, SnapshotSHA256: sum(manifest), HistoricalDocuments: frozenRef, Impact: GlossaryArchiveReference{"impact.json", sum(impact)}, Apply: true}
	plan, err := ReconcileTranslationUnitRepresentations(root, current, options)
	if err != nil || !plan.Applied || plan.Ready != len(ids) {
		t.Fatal("fixture machine proof", plan, err)
	}
	receiptPath := machineReceiptPath(options)
	receiptBytes, err := readSnapshotReferencedFile(root, receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	ref := GlossaryArchiveReference{receiptPath, sum(receiptBytes)}
	return root, current, s, plan, ref, receiptBytes
}

func TestAuthorizationMachineMemoFailClosed(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, current, s, plan, ref, receiptBytes := authorizationMachineFixture(t)
	receiptPath := ref.Path
	selected := machineSelection{receipt: plan.Receipt, unit: plan.Receipt.Units[0], ref: ref}
	unit := selected.snapshot(1)
	verify := func(u QualityCheckSnapshotUnit) error {
		_, _, err := machineSnapshotEvidence(root, current, "zh-CN", u, nil)
		return err
	}
	if err := verify(unit); err != nil {
		t.Fatal("initial proof", err)
	}
	for _, kind := range []string{"unchanged", "expected SHA", "receipt bytes", "old mapping", "current mapping", "validation bytes", "validation identity", "candidate bytes", "historical candidate"} {
		t.Run(kind, func(t *testing.T) {
			u := unit
			newRef := ref
			u.RepresentationReconciliation = &newRef
			path := receiptPath
			original := receiptBytes
			changed := append(bytes.Clone(original), '\n')
			switch kind {
			case "expected SHA":
				newRef.SHA256 = strings.Repeat("0", 64)
			case "old mapping", "current mapping", "validation identity":
				var r RepresentationReconciliation
				if err := json.Unmarshal(receiptBytes, &r); err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "old mapping":
					r.Units[0].Replacement.OldUnitID = "wrong-old"
				case "current mapping":
					r.Units[0].Replacement.CurrentUnitID = "wrong-current"
				case "validation identity":
					r.Units[0].Validation.SHA256 = strings.Repeat("0", 64)
				}
				r.Identity = machineReceiptIdentity(r)
				changed = mustJSON(r)
				newRef.SHA256 = sum(changed)
			case "validation bytes":
				path = unit.ValidationPath
			case "candidate bytes":
				path = unit.CandidatePath
			case "historical candidate":
				path = s.Units[0].CandidatePath
			}
			if path != receiptPath {
				var err error
				original, err = readSnapshotReferencedFile(root, path)
				if err != nil {
					t.Fatal(err)
				}
				changed = append(bytes.Clone(original), '\n')
			}
			if kind != "unchanged" && kind != "expected SHA" {
				writeSnapshotSource(t, root, path, changed)
				defer writeSnapshotSource(t, root, path, original)
			}
			err := verify(u)
			if (err == nil) != (kind == "unchanged") {
				t.Fatalf("kind=%s err=%v", kind, err)
			}
		})
	}
}
