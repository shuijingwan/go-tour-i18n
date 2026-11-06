package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func bindingGenerateFixture(t *testing.T, root string, c *i18n.Catalog, ids []string) {
	t.Helper()
	for _, batch := range surfaceFixtureUnitBatches(t, c, ids) {
		e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: batch})
		if err != nil {
			t.Fatal(err)
		}
		bundle, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
		if err != nil {
			t.Fatal(err)
		}
		outputs := map[string][]byte{}
		for _, o := range m.ExpectedOutputs {
			outputs[o.UnitID] = files["formal/inputs/"+filepath.Base(o.BundlePath)]
		}
		surfaceImport(t, root, c, bundle, outputs)
		if r, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || r.ValidationPassed != len(batch) {
			t.Fatal(r, err)
		}
	}
}

func bindingScope(finals ...*i18n.QualityCheckFinalization) *IntegratedSurfaceScope {
	s := &IntegratedSurfaceScope{Locale: "zh-CN"}
	for _, f := range finals {
		for _, u := range f.Units {
			s.Reviewed = append(s.Reviewed, SurfaceAsset{ID: u.Snapshot.SourcePath, TargetSHA: u.Snapshot.CandidateSHA256})
		}
	}
	return s
}

func TestSurfaceBindingHistoricalPackageExpansion(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, old, event, frozen := replacementPageFixtureSize(t, 49, 47, nil)
	previous := surfaceStart(t, root, old, old.TranslationScope)
	historical := freezeReplacementFixture(t, root, frozen)
	mapping, err := resolvePageReplacements(root, event, historical)
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{}
	for _, p := range mapping.Units {
		if p.OldUnitID != p.CurrentUnitID {
			rows = append(rows, map[string]any{"current_unit_id": p.CurrentUnitID, "snapshot_representation_unit_id": p.OldUnitID, "source_path": p.SourcePath, "source_sha256": p.SourceSHA256, "labels": map[string]bool{"representation_reconciliation_required": true, "true_language_defect": false, "unknown": false}})
		}
	}
	impact, _ := json.Marshal(map[string]any{"units": rows})
	writeFixture(t, root, "binding-impact.json", impact)
	snapshot, err := ReferenceFile(root, "data/quality-check-snapshots/zh-CN/"+previous.SnapshotID+"/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := i18n.ReconcileTranslationUnitRepresentations(root, event, i18n.RepresentationReconciliationOptions{Locale: "zh-CN", Package: frozen.Package, ReconciliationID: "binding-machine", PreviousSnapshotID: previous.SnapshotID, SnapshotSHA256: snapshot.SHA256, Impact: i18n.GlossaryArchiveReference{Path: "binding-impact.json", SHA256: digest(impact)}, HistoricalDocuments: historical, Apply: true})
	if err != nil || plan.Ready != 2 {
		t.Fatal("synthetic machine proof", err)
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, event, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "binding-reconciled", ContentPackage: frozen.Package}); err != nil {
		t.Fatal(err)
	}
	frozenFinal, _, err := i18n.FinalizeQualityCheck(root, event, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "binding-reconciled", PreviousSnapshotID: previous.SnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	original := event.TranslationPackages[0]
	added := i18n.TranslationPackageDescriptor{ID: "binding-independent-addition", AuthoritySHA256: digest([]byte("independent-addition"))}
	for i := 0; i < 2; i++ {
		path := fmt.Sprintf("_content/doc/binding-addition-%d.md", i)
		source := []byte("# Additional Page\n\nAn independent message.\n")
		writeFixture(t, root, path, source)
		doc, err := ParseWorkflowDocument(Source{Path: path, Kind: "page", SHA256: digest(source), Route: fmt.Sprintf("/binding-addition/%d", i)}, source, 50+i)
		if err != nil {
			t.Fatal(err)
		}
		added.Units = append(added.Units, i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: doc.Units[0].ID, Kind: i18n.UnitKindPage, SourcePath: path, SourceSHA256: digest(source), Source: source}, Adapter: PageAdapter{*doc, source}, StableIndex: 50 + i, Route: doc.Route})
	}
	addition := &i18n.Catalog{}
	// Discovery retains the historical invocation inventory while scheduling
	// only the independent new package, as a production registered Catalog does.
	seen := map[string]bool{}
	for _, inventory := range []*i18n.Catalog{old, event} {
		for _, u := range inventory.Adapted {
			if !seen[u.Unit.ID] {
				addition.Adapted = append(addition.Adapted, u)
				seen[u.Unit.ID] = true
			}
		}
	}
	if err := addition.RegisterTranslationPackage(added); err != nil {
		t.Fatal(err)
	}
	if err := addition.UseTranslationPackage(added.ID); err != nil {
		t.Fatal(err)
	}
	bindingGenerateFixture(t, root, addition, addition.TranslationScope)
	addedFinal := surfaceFinalize(t, root, addition, "binding-added", "", addition.TranslationScope)
	current := *event
	merged := original
	merged.AuthoritySHA256 = digest([]byte("current-complete-membership"))
	merged.Units = append(append([]i18n.AdaptedTranslationUnit{}, original.Units...), added.Units...)
	current.TranslationPackages = []i18n.TranslationPackageDescriptor{merged}
	current.HistoricalTranslationPackages = []i18n.TranslationPackageDescriptor{original, added}
	current.Adapted = append(append([]i18n.AdaptedTranslationUnit{}, event.Adapted...), added.Units...)
	if err := current.UseTranslationPackage(merged.ID); err != nil {
		t.Fatal(err)
	}
	bind := func() error {
		return BindIntegratedSurfaceCandidates(root, &current, bindingScope(frozenFinal, addedFinal), frozenFinal, addedFinal)
	}
	if err := bind(); err != nil {
		t.Fatal("complete 51 with frozen 49 and independent 2", err)
	}
	if err := BindIntegratedSurfaceCandidates(root, &current, bindingScope(frozenFinal), frozenFinal); err == nil {
		t.Fatal("historical 49 authorized missing new Units")
	}
	missing := bindingScope(frozenFinal, addedFinal)
	missing.Reviewed = missing.Reviewed[:50]
	if err := BindIntegratedSurfaceCandidates(root, &current, missing, frozenFinal, addedFinal); err == nil {
		t.Fatal("missing presented new Page accepted")
	}
	for label, path := range map[string]string{"receipt": "data/translation-unit-reconciliations/zh-CN/binding-machine/receipt.json", "candidate": plan.Receipt.Units[0].Candidate.Path, "validation": plan.Receipt.Units[0].Validation.Path, "provenance": plan.Receipt.Units[0].OriginalRaw.Path, "historical-qc": "data/quality-check-snapshots/zh-CN/" + previous.SnapshotID + "/quality-check-results.json", "archived-glossary": "data/glossary-history/zh-CN/" + plan.Receipt.GlossarySHA256 + ".yaml"} {
		t.Run(label, func(t *testing.T) {
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, before)
			writeFixture(t, root, path, append(bytes.Clone(before), ' '))
			if err := bind(); err == nil {
				t.Fatal("tampered lineage accepted", label)
			}
		})
	}
	oldGlossary, err := readRegular(root, "locales/zh-CN/glossary.yaml")
	if err != nil {
		t.Fatal(err)
	}
	structuredGlossarySuccessor(t, root, "Unrelated source term")
	if now, _ := readRegular(root, "locales/zh-CN/glossary.yaml"); bytes.Equal(oldGlossary, now) {
		t.Fatal("fixture did not advance current glossary")
	}
	if err := bind(); err != nil {
		t.Fatal("current glossary contaminated frozen reconciliation", err)
	}
	currentGlossary, _ := readRegular(root, "locales/zh-CN/glossary.yaml")
	writeFixture(t, root, "locales/zh-CN/glossary.yaml", append(bytes.Clone(currentGlossary), []byte("# unreviewed successor\n")...))
	if err := bind(); err == nil {
		t.Fatal("frozen glossary substituted for current full Review")
	}
	t.Log("current=51; historical provenance/QC=49; new 2 independent; missing new target and tampered lineage rejected")
}
