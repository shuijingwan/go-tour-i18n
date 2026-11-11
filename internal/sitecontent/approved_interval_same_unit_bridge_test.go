package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// A synthetic active validator epoch. All representation/source operations
// remain the real Page adapter, and historical evidence uses its source-owned
// canonical catalog. Only the current validator rejects this exact candidate.
type approvedBridgeValidatorEpoch struct {
	PageAdapter
	canonical   *i18n.Catalog
	rejectedSHA string
}

func (a approvedBridgeValidatorEpoch) ApprovedIntervalBridgeCatalog(*i18n.Catalog) *i18n.Catalog {
	return a.canonical
}
func (a approvedBridgeValidatorEpoch) ReattestationAuthority(root string) (string, error) {
	identity, err := a.PageAdapter.ReattestationAuthority(root)
	return digest([]byte(identity + "/current-validator-reject/" + a.rejectedSHA)), err
}
func (a approvedBridgeValidatorEpoch) Validate(candidate []byte, glossary *i18n.Glossary) error {
	if err := a.PageAdapter.Validate(candidate, glossary); err != nil {
		return err
	}
	if digest(candidate) == a.rejectedSHA {
		return fmt.Errorf("synthetic current Page validator rejects reviewed candidate")
	}
	return nil
}

func TestApprovedIntervalSameUnitBridge(t *testing.T) {
	checkApprovedIntervalBundleLifecycle(t, false, func(t *testing.T, root string, c *i18n.Catalog, reopen i18n.CurrentValidationReopenOptions, batch, id string, export i18n.GenerationBundleOptions) {
		for _, path := range []string{"internal/i18n/approved_interval_same_unit_bridge.go", "internal/sitecontent/approved_interval_same_unit_bridge.go"} {
			writeFixture(t, root, path, mustReadFixture(t, repositoryRoot(t), path))
		}
		manifestPath := "data/retranslation-runs/zh-CN/" + batch + "/manifest.json"
		var m i18n.RetranslationBatchManifest
		if err := json.Unmarshal(mustReadFixture(t, root, manifestPath), &m); err != nil {
			t.Fatal(err)
		}
		var reviewedIDs []string
		for _, r := range m.Units {
			reviewedIDs = append(reviewedIDs, r.UnitID)
		}
		if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: reopen.Locale, SnapshotID: "approved-after-retry", PreviousSnapshotID: reopen.PreviousSnapshotID, ReviewerSession: "independent-approved-bridge-reviewer", UnitIDs: reviewedIDs, Rating: "B", Finding: "Improve ordinary prose outside the immutable approved parent."}); err != nil {
			t.Fatal(err)
		}
		const review = "engineering/approved-bridge-reviewer.md"
		evidence := []byte("Independent Reviewer: B. Improve ordinary prose outside the immutable approved parent.\n")
		writeFixture(t, root, review, evidence)
		o := i18n.SameUnitBridgeOptions{Locale: reopen.Locale, ContentPackage: reopen.ContentPackage, BridgeID: "approved-ordinary-qc", PreviousSnapshotID: "approved-after-retry", SnapshotSHA256: digest(mustReadFixture(t, root, "data/quality-check-snapshots/zh-CN/approved-after-retry/manifest.json")), ReviewerEvidence: i18n.GlossaryArchiveReference{Path: review, SHA256: digest(evidence)}, UnitIDs: []string{id}}
		path := "internal/i18n/generation_revision_inputs.go"
		before := mustReadFixture(t, root, path)
		writeFixture(t, root, path, append(bytes.Clone(before), []byte("\n// synthetic code-owned implementation drift\n")...))
		if _, err := i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: reopen.Locale, BatchID: batch, UnitID: id}); err == nil || !strings.Contains(err.Error(), "installed decoder/proof drift") {
			t.Fatal("ordinary current-check must remain strict", err)
		}
		var record i18n.RetranslationBatchUnit
		for _, r := range m.Units {
			if r.UnitID == id {
				record = r
			}
		}
		oldReceiptPath := "data/retranslation-runs/zh-CN/" + batch + "/generation-imports/initial-manifest.json"
		var imported i18n.GenerationResultBundleManifest
		if err := json.Unmarshal(mustReadFixture(t, root, oldReceiptPath), &imported); err != nil {
			t.Fatal(err)
		}
		archivePath := "data/generation-archives/zh-CN/" + imported.GenerationBundleSHA256 + ".zip"
		archive := mustReadFixture(t, root, archivePath)
		files, err := i18n.ReadTransportBundle(archive, 8192, 64<<20)
		if err != nil {
			t.Fatal(err)
		}
		member := "formal/approved-interval-transport/" + id + ".json"
		proofBytes := bytes.Clone(files[member])
		// The Catalog is a directly registered synthetic package. Its exact
		// descriptor supplies the fixture's context bytes; production source
		// publication/registry parsers are not exercised by this fixture.
		metadata, _ := json.Marshal(c.TranslationPackages[0])
		for _, path := range []string{SourceManifestPath, GlobalPath, PageRegistryPath} {
			if _, err := os.Stat(filepath.Join(root, path)); os.IsNotExist(err) {
				writeFixture(t, root, path, metadata)
			} else if err != nil {
				t.Fatal(err)
			}
		}
		c.DerivedTransportCodecReader = derivedApprovedCodec
		plan, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o)
		if err != nil {
			t.Fatal("approved bridge preflight", err)
		}
		if plan.Recorded || plan.Mutation != "none" || len(plan.Receipt.SameUnitBridge.Bindings) != 1 {
			t.Fatal("readonly scope", plan)
		}
		decorated, err := i18n.PrepareSameUnitRepresentationBridge(root, BridgeRepresentationCatalog(root, c), o)
		if err != nil || decorated.ReceiptSHA256 != plan.ReceiptSHA256 {
			t.Fatal("decorated source-owned bridge changed", err)
		}
		b := plan.Receipt.SameUnitBridge.Bindings[0]
		if b.TransportMember != member || b.TransportSHA256 != digest(proofBytes) || !bytes.Equal(archive, mustReadFixture(t, root, archivePath)) {
			t.Fatal("historical proof/archive changed")
		}
		if _, err := os.Stat(filepath.Join(root, plan.ReceiptPath)); !os.IsNotExist(err) {
			t.Fatal("preflight wrote receipt")
		}
		// Fresh owned adapter construction must exactly restore and allow ordinary
		// prose editing, with the unbindable approved parent remaining opaque.
		var p i18n.ApprovedIntervalTransport
		if err := json.Unmarshal(proofBytes, &p); err != nil {
			t.Fatal(err)
		}
		var result i18n.RetranslationProcessResult
		if err := json.Unmarshal(mustReadFixture(t, root, "data/retranslation-runs/zh-CN/"+batch+"/result.json"), &result); err != nil {
			t.Fatal(err)
		}
		var final i18n.RetranslationUnitResult
		for _, r := range result.Units {
			if r.UnitID == id {
				final = r
			}
		}
		var validation i18n.RetranslationValidation
		if err := json.Unmarshal(mustReadFixture(t, root, "data/retranslation-runs/zh-CN/"+batch+"/"+final.ValidationPath), &validation); err != nil {
			t.Fatal(err)
		}
		rawPath := "data/retranslation-runs/zh-CN/" + batch + "/" + validation.RawResponsePath
		raw := mustReadFixture(t, root, rawPath)
		candidatePath := "data/retranslation-runs/zh-CN/" + batch + "/" + final.CandidatePath
		candidate := mustReadFixture(t, root, candidatePath)
		u, _ := c.Unit(id)
		factory := u.Adapter.(i18n.ApprovedIntervalSameUnitBridgeAdapter)
		decoder, template, _, err := factory.BridgeApprovedIntervalRevision(p.Projection, raw, candidate)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decoder.Restore(string(template))
		if err != nil || !bytes.Equal(bridgeEOF(got), candidate) {
			t.Fatal("new inverse", err)
		}
		if !bytes.Equal(template, []byte(b.Template)) {
			t.Fatal("fresh source-owned template differs")
		}
		if !bytes.Contains(template, []byte("请阅读")) {
			t.Fatal("fixture ordinary prose missing")
		}
		edited := bytes.Replace(template, []byte("请阅读"), []byte("请仔细阅读"), 1)
		revised, err := decoder.Restore(string(edited))
		if err != nil || bytes.Equal(bridgeEOF(revised), candidate) {
			t.Fatal("ordinary prose remained restricted", err)
		}
		if err := i18n.ValidateTranslationUnitCandidate(root, c, id, reopen.Locale, bridgeEOF(revised)); err != nil {
			t.Fatal("ordinary edited candidate validator", err)
		}
		for name, rel := range map[string]string{"candidate": candidatePath, "raw": rawPath, "archive": archivePath, "final-archive": b.Archive.Path, "import": b.Import.Path, "validation": "data/retranslation-runs/zh-CN/" + batch + "/" + final.ValidationPath, "input": "data/retranslation-runs/zh-CN/" + batch + "/" + record.InputPath, "source": record.SourcePath, "Reviewer": review} {
			t.Run(name+"-tamper", func(t *testing.T) {
				old := mustReadFixture(t, root, rel)
				changed := append(bytes.Clone(old), ' ')
				if name == "import" {
					var receipt i18n.GenerationResultBundleManifest
					if err := json.Unmarshal(old, &receipt); err != nil {
						t.Fatal(err)
					}
					receipt.Attempt++ // A bound identity change, not insignificant JSON whitespace.
					changed, err = json.Marshal(receipt)
					if err != nil {
						t.Fatal(err)
					}
				}
				writeFixture(t, root, rel, changed)
				defer writeFixture(t, root, rel, old)
				if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o); err == nil {
					t.Fatal("tamper accepted")
				}
			})
		}
		t.Run("current-validator-failure", func(t *testing.T) {
			// Changing locale policy changes the approved finding too, so it is
			// not an isolated validator oracle. Keep that semantic evidence exact
			// and exercise the canonical validation dispatcher with a stricter
			// active Page validator, independently of historical reconstruction.
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
			const cause = "synthetic current Page validator rejects reviewed candidate"
			if err := i18n.ValidateTranslationUnitCandidate(root, &current, id, reopen.Locale, candidate); err == nil || !strings.Contains(err.Error(), cause) {
				t.Fatal("fixture must fail active canonical validation", err)
			}
			if _, err := i18n.PrepareSameUnitRepresentationBridge(root, &current, o); err == nil || !strings.Contains(err.Error(), "bridge current validation failed: "+cause) {
				t.Fatal("expected current validator gate", err)
			}
		})
		t.Run("coordinate-and-inverse", func(t *testing.T) {
			bad := p.Projection
			bad.ParentSource.Start++
			if _, _, _, err := factory.BridgeApprovedIntervalRevision(bad, raw, candidate); err == nil {
				t.Fatal("unbound parent accepted")
			}
			bad = p.Projection
			bad.CurrentRepresentationSHA256 = strings.Repeat("0", 64)
			if _, _, _, err := factory.BridgeApprovedIntervalRevision(bad, raw, candidate); err == nil {
				t.Fatal("representation changed")
			}
			if _, _, _, err := factory.BridgeApprovedIntervalRevision(p.Projection, raw, append(bytes.Clone(candidate), ' ')); err == nil {
				t.Fatal("candidate mismatch")
			}
			_, tokens, _ := decoder.ProtectedInput(nil)
			if _, err := decoder.Restore(strings.Replace(string(template), tokens[len(tokens)-1], "", 1)); err == nil {
				t.Fatal("opaque parent deletion accepted")
			}
		})
		// Fixture-only publication exercises the existing shared consumer. The
		// Generation Bundle receives the new ordinary response, never a guessed
		// parent mapping or an old approved proof declared CURRENT.
		o.Record = true
		sealed, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o)
		if err != nil || !sealed.Recorded {
			t.Fatal("fixture record", err)
		}
		next, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: o.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: o.UnitIDs, AllowReexport: true, PreviousSnapshotID: o.PreviousSnapshotID, RepresentationBridgeID: o.BridgeID})
		if err != nil {
			t.Fatal("fixture ordinary bridge revision", err)
		}
		bundle, manifest, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: o.Locale, BatchID: next.BatchID})
		if err != nil {
			t.Fatal("fixture bridge Bundle", err)
		}
		newFiles, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(manifest.ExpectedOutputs[0].BundlePath)
		if !bytes.Equal(newFiles["formal/protected-candidates/"+name], template) {
			t.Fatal("Generation did not receive exact ordinary template")
		}
		surfaceImport(t, root, c, bundle, map[string][]byte{id: template})
		processed, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: o.Locale, BatchID: next.BatchID})
		if err != nil || processed.ValidationPassed != 1 {
			t.Fatal("fixture ordinary bridge process", err)
		}
		t.Run("partial-successor-preserves-untouched-historical-pass-sibling", func(t *testing.T) {
			snapshot, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: o.Locale, ContentPackage: o.ContentPackage, SnapshotID: "approved-bridge-successor"})
			if err != nil {
				t.Fatal("frozen bridge reader", err)
			}
			for _, r := range m.Units {
				found := false
				for _, chosen := range snapshot.Units {
					if chosen.UnitID != r.UnitID {
						continue
					}
					found = true
					wantBatch := batch
					if r.UnitID == id {
						wantBatch = next.BatchID
					}
					if chosen.SelectedBatchID != wantBatch {
						t.Fatalf("old fallback or lost sibling: %s selected %s, want %s", r.UnitID, chosen.SelectedBatchID, wantBatch)
					}
					if wantBatch == batch {
						var original i18n.RetranslationUnitResult
						for _, old := range result.Units {
							if old.UnitID == r.UnitID {
								original = old
							}
						}
						if chosen.CandidateSHA256 != digest(mustReadFixture(t, root, "data/retranslation-runs/zh-CN/"+batch+"/"+original.CandidatePath)) {
							t.Fatal("trusted sibling bytes changed")
						}
					}
				}
				if !found {
					t.Fatal("lost selected Unit", r.UnitID)
				}
			}
			if !bytes.Equal(archive, mustReadFixture(t, root, archivePath)) {
				t.Fatal("partial admission rewrote historical archive")
			}
			t.Run("pending-successor-on-admitted-sibling-rejected", func(t *testing.T) {
				pending := m
				pending.BatchID = next.BatchID + "9"
				pending.Units = nil
				for _, r := range m.Units {
					if r.UnitID != id {
						pending.Units = append(pending.Units, r)
					}
				}
				if len(pending.Units) != 1 {
					t.Fatal("fixture requires exactly one admitted sibling")
				}
				pending.UnitCount = len(pending.Units)
				path := "data/retranslation-runs/zh-CN/" + pending.BatchID
				data, err := json.MarshalIndent(pending, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, path+"/manifest.json", data)
				defer os.RemoveAll(filepath.Join(root, path))
				if _, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: o.Locale, ContentPackage: o.ContentPackage, SnapshotID: "pending-admitted-sibling"}); err == nil || !strings.Contains(err.Error(), "conflicting successor") {
					t.Fatal("pending admitted-Unit successor must block", err)
				}
			})
			t.Run("newer-selection-corruption-does-not-fall-back", func(t *testing.T) {
				if len(processed.Units) != 1 {
					t.Fatal("fixture successor scope")
				}
				path := "data/retranslation-runs/zh-CN/" + next.BatchID + "/" + processed.Units[0].CandidatePath
				before := mustReadFixture(t, root, path)
				defer writeFixture(t, root, path, before)
				writeFixture(t, root, path, append(bytes.Clone(before), ' '))
				if _, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: o.Locale, ContentPackage: o.ContentPackage, SnapshotID: "no-old-fallback"}); err == nil {
					t.Fatal("corrupt newer candidate fell back to historical selection")
				}
			})
			for _, original := range result.Units {
				if original.UnitID != id {
					continue
				} // No longer admitted; its full historical PASS must still hold.
				for _, rel := range []string{original.CandidatePath, original.ValidationPath} {
					t.Run("unselected-historical-integrity-"+filepath.Base(rel), func(t *testing.T) {
						path := "data/retranslation-runs/zh-CN/" + batch + "/" + rel
						before := mustReadFixture(t, root, path)
						defer writeFixture(t, root, path, before)
						writeFixture(t, root, path, append(bytes.Clone(before), ' '))
						if _, err := i18n.PrepareQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: o.Locale, ContentPackage: o.ContentPackage, SnapshotID: "full-historical-integrity"}); err == nil {
							t.Fatal("partial admission lost full-batch historical integrity")
						}
					})
				}
			}
		})

	})
}
