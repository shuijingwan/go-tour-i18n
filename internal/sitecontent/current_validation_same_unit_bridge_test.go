package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func cvpredBridgeFixtureContext(t *testing.T, root string, c *i18n.Catalog) {
	t.Helper()
	data, err := json.Marshal(c.TranslationPackages[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{SourceManifestPath, GlobalPath, PageRegistryPath} {
		if _, err := os.Stat(filepath.Join(root, path)); os.IsNotExist(err) {
			writeFixture(t, root, path, data)
		} else if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCurrentValidationSameUnitBridge(t *testing.T) {
	for _, viaBridge := range []bool{false, true} {
		name := "Surface-derived"
		if viaBridge {
			name = "ordinary-bridge-derived"
		}
		t.Run(name, func(t *testing.T) {
			done := contentidentity.BeginMemoization()
			defer done()
			root, c, batch, _ := currentValidationSurfacePredecessorFixture(t, viaBridge)
			cvpredBridgeFixtureContext(t, root, c)
			writeFixture(t, root, "internal/i18n/current_validation_same_unit_bridge.go", mustReadFixture(t, repositoryRoot(t), "internal/i18n/current_validation_same_unit_bridge.go"))
			bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: batch})
			if err != nil {
				t.Fatal(err)
			}
			files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
			if err != nil {
				t.Fatal(err)
			}
			id := c.TranslationScope[0]
			base := filepath.Base(bm.ExpectedOutputs[0].BundlePath)
			member := "formal/current-validation-predecessor-transport/" + base + ".json"
			var frozen i18n.CurrentValidationPredecessor
			if err := json.Unmarshal(files[member], &frozen); err != nil {
				t.Fatal(err)
			}
			if frozen.RecoveryClass != "DETERMINISTIC_REBASE_PASS" || frozen.OrdinaryRestore {
				t.Fatal("fixture lacks derived predecessor")
			}
			replacement := bridgeEOF([]byte(strings.ReplaceAll(string(files["formal/protected-candidates/"+base]), ".", "。")))
			surfaceImport(t, root, c, bundle, map[string][]byte{id: replacement})
			result, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch})
			if err != nil || result.ValidationPassed != 1 {
				t.Fatal("cvpred process", err)
			}
			const snapshot = "cvpred-reviewed"
			if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
				t.Fatal(err)
			}
			if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: frozen.Selected[0].PreviousSnapshotID, UnitIDs: []string{id}, Rating: "B", Finding: "Improve ordinary prose.", ReviewerSession: "independent-synthetic-reviewer"}); err != nil {
				t.Fatal(err)
			}
			review := []byte("Independent Reviewer B: Improve ordinary prose.\n")
			writeFixture(t, root, "engineering/cvpred-bridge-review.md", review)
			options := i18n.SameUnitBridgeOptions{Locale: "zh-CN", ContentPackage: c.TranslationPackages[0].ID, BridgeID: "cvpred-ordinary", PreviousSnapshotID: snapshot, SnapshotSHA256: digest(mustReadFixture(t, root, "data/quality-check-snapshots/zh-CN/"+snapshot+"/manifest.json")), ReviewerEvidence: i18n.GlossaryArchiveReference{Path: "engineering/cvpred-bridge-review.md", SHA256: digest(review)}, UnitIDs: []string{id}}
			implementation := "internal/i18n/generation_revision_inputs.go"
			writeFixture(t, root, implementation, append(bytes.Clone(mustReadFixture(t, root, implementation)), []byte("\n// synthetic producer-only drift\n")...))
			before := mustReadFixture(t, root, "data/retranslation-runs/zh-CN/"+batch+"/manifest.json")
			plan, err := i18n.PrepareSameUnitRepresentationBridge(root, c, options)
			if err != nil {
				t.Fatal("cvpred bridge preflight", err)
			}
			if plan.Recorded || plan.Mutation != "none" {
				t.Fatal("readonly mutation")
			}
			binding := plan.Receipt.SameUnitBridge.Bindings[0]
			if binding.TransportMember != member || binding.TransportSHA256 != digest(files[member]) || binding.Template != string(replacement) || binding.TemplateSHA256 != digest(replacement) {
				t.Fatal("cvpred member/template binding changed")
			}
			if !bytes.Equal(before, mustReadFixture(t, root, "data/retranslation-runs/zh-CN/"+batch+"/manifest.json")) {
				t.Fatal("old manifest changed")
			}
			if _, err := os.Stat(filepath.Join(root, plan.ReceiptPath)); !os.IsNotExist(err) {
				t.Fatal("preflight published")
			}
			candidate := mustReadFixture(t, root, "data/retranslation-runs/zh-CN/"+batch+"/"+result.Units[0].CandidatePath)
			if err := i18n.ValidateTranslationUnitCandidate(root, c, id, "zh-CN", candidate); err != nil {
				t.Fatal(err)
			}
			for label, path := range map[string]string{"final-raw": binding.Raw.Path, "archive": binding.Archive.Path, "candidate": "data/retranslation-runs/zh-CN/" + batch + "/" + result.Units[0].CandidatePath, "reopen": frozen.Reopen.Path, "historical-raw": frozen.Raw.Path} {
				t.Run(label+"-tamper", func(t *testing.T) {
					before := mustReadFixture(t, root, path)
					defer writeFixture(t, root, path, before)
					writeFixture(t, root, path, append(bytes.Clone(before), ' '))
					if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, options); err == nil {
						t.Fatal("tampered cvpred evidence accepted", label)
					}
				})
			}
			t.Run("current-validator-failure", func(t *testing.T) {
				current := *c
				current.Adapted = append([]i18n.AdaptedTranslationUnit(nil), c.Adapted...)
				current.TranslationPackages = append([]i18n.TranslationPackageDescriptor(nil), c.TranslationPackages...)
				current.TranslationPackages[0].Units = append([]i18n.AdaptedTranslationUnit(nil), c.TranslationPackages[0].Units...)
				for i, unit := range current.Adapted {
					if unit.Unit.ID == id {
						current.Adapted[i].Adapter = approvedBridgeValidatorEpoch{PageAdapter: unit.Adapter.(PageAdapter), canonical: c, rejectedSHA: digest(candidate)}
					}
				}
				for i, unit := range current.TranslationPackages[0].Units {
					if unit.Unit.ID == id {
						current.TranslationPackages[0].Units[i].Adapter = approvedBridgeValidatorEpoch{PageAdapter: unit.Adapter.(PageAdapter), canonical: c, rejectedSHA: digest(candidate)}
					}
				}
				if err := i18n.ValidateTranslationUnitCandidate(root, &current, id, "zh-CN", candidate); err == nil {
					t.Fatal("invalid validator oracle")
				}
				if _, err := i18n.PrepareSameUnitRepresentationBridge(root, &current, options); err == nil || !strings.Contains(err.Error(), "bridge current validation failed") {
					t.Fatal("current validator was not enforced", err)
				}
			})
			options.Record = true
			sealed, err := i18n.PrepareSameUnitRepresentationBridge(root, c, options)
			if err != nil {
				t.Fatal("fixture record", err)
			}
			next, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{id}, AllowReexport: true, PreviousSnapshotID: snapshot, RepresentationBridgeID: options.BridgeID})
			if err != nil {
				t.Fatal(err)
			}
			out, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: next.BatchID})
			if err != nil {
				t.Fatal("frozen cvpred consumer", err)
			}
			nextFiles, err := i18n.ReadTransportBundle(out, 8192, 64<<20)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(nextFiles["formal/protected-candidates/"+base], []byte(sealed.Receipt.SameUnitBridge.Bindings[0].Template)) {
				t.Fatal("frozen template changed")
			}
			surfaceImport(t, root, c, out, map[string][]byte{id: []byte(binding.Template)})
			if p, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: next.BatchID}); err != nil || p.ValidationPassed != 1 {
				t.Fatal("frozen decoder exact restore", err)
			}
		})
	}
}
