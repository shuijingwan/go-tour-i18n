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

// Full package history matters: five ratings exist only in a sealed checkpoint,
// while the current Snapshot records one B/C/D after a real candidate change.
// A fixture with only direct ratings cannot detect loss of the package scope.
func TestQCRepresentationSuccessorExportWithSealedCarry(t *testing.T) {
	for _, rating := range []string{"B", "C", "D"} {
		t.Run(rating, func(t *testing.T) {
			done := contentidentity.BeginMemoization()
			defer done()
			root, old, current, frozen := replacementPageFixture(t)
			ids := append([]string(nil), old.TranslationScope...)
			initial := surfaceStart(t, root, old, ids)
			if _, _, err := i18n.EstablishCurrentTranslationAuthorization(root, old, "zh-CN", initial.SnapshotID); err != nil {
				t.Fatal(err)
			}
			const finding = "Localize the visible message frozen by the previous representation."
			writeFixture(t, root, "data/locale-surface-reviews/zh-CN/synthetic-surface.md", []byte("decision = failed\nTranslationUnit "+ids[2]+": "+finding+"\n"))
			if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, old, i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "synthetic-surface", PreviousSnapshotID: initial.SnapshotID, SurfaceReviewID: "synthetic-surface", Finding: finding, UnitIDs: ids[2:3]}); err != nil {
				t.Fatal(err)
			}
			e, err := i18n.ExportRetranslationBatch(root, old, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: ids[2:3], AllowReexport: true, PreviousSnapshotID: initial.SnapshotID, SurfaceReopenID: "synthetic-surface"})
			if err != nil {
				t.Fatal(err)
			}
			bundle, manifest, err := i18n.ExportTranslationUnitGenerationBundle(root, old, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
			if err != nil {
				t.Fatal(err)
			}
			files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
			if err != nil {
				t.Fatal(err)
			}
			output := bytes.Replace(files["formal/protected-candidates/"+filepath.Base(manifest.ExpectedOutputs[0].BundlePath)], []byte("Page"), []byte("Changed page"), 1)
			surfaceImport(t, root, old, bundle, map[string][]byte{ids[2]: output})
			if r, err := i18n.ProcessRetranslationBatch(root, old, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || r.ValidationPassed != 1 {
				t.Fatal(r, err)
			}
			const snapshot = "qc-carried-representation"
			if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, old, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
				t.Fatal(err)
			}
			if _, _, err := i18n.ExportQualityCheckReviewerBundle(root, old, i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: initial.SnapshotID}); err != nil {
				t.Fatal(err)
			}
			if _, err := i18n.RecordQualityCheckResults(root, old, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: initial.SnapshotID, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: ids[2:3], Rating: rating, Finding: finding}); err != nil {
				t.Fatal(err)
			}
			historical := freezeReplacementFixture(t, root, frozen)
			resolution, err := resolvePageReplacements(root, current, historical)
			if err != nil {
				t.Fatal(err)
			}
			p := resolution.Units[ids[2]]
			proof := i18n.RepresentationNecessityProof{Reason: "protected-natural-language-unrepresentable", Units: []i18n.RepresentationNecessityUnit{{Replacement: p, Rating: rating, FindingSHA256: digest([]byte(finding))}}}
			data, _ := json.Marshal(proof)
			const proofPath = "data/engineering/necessity.json"
			writeFixture(t, root, proofPath, data)
			o := i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: "quality-check", Locale: "zh-CN", ReopenID: "qc-successor", PreviousSnapshotID: snapshot, PreviousRating: rating, Finding: finding, UnitIDs: []string{p.CurrentUnitID}, HistoricalDocuments: historical, RepresentationProof: i18n.GlossaryArchiveReference{Path: proofPath, SHA256: digest(data)}}
			t.Run("actual-A-cannot-reconcile", func(t *testing.T) {
				pA := resolution.Units[ids[3]]
				badProof := proof
				badProof.Units = []i18n.RepresentationNecessityUnit{{Replacement: pA, Rating: rating, FindingSHA256: digest([]byte(finding))}}
				b, _ := json.Marshal(badProof)
				const path = "data/engineering/false-necessity.json"
				writeFixture(t, root, path, b)
				bad := o
				bad.ReopenID, bad.UnitIDs = "invalid-A", []string{pA.CurrentUnitID}
				bad.RepresentationProof = i18n.GlossaryArchiveReference{Path: path, SHA256: digest(b)}
				if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, current, bad); err == nil || !strings.Contains(err.Error(), "exact QC B/C/D finding required") {
					t.Fatal("sealed A did not fail at the QC authority gate", err)
				}
			})
			receipt, receiptPath, err := i18n.RecordQualityCheckSurfaceReopen(root, current, o)
			if err != nil {
				t.Fatal(err)
			}
			receiptBytes, err := os.ReadFile(filepath.Join(root, receiptPath))
			if err != nil {
				t.Fatal(err)
			}
			// The CLI attaches every registered package before selecting one. Both
			// the explicit canonical template and unselected inventory must work.
			inventory := *current
			inventory.TranslationScope = nil
			if _, err := i18n.CheckQualityCheckSurfaceReopen(root, &inventory, "zh-CN", o.ReopenID); err != nil {
				t.Fatal("unselected inventory lost sealed A", err)
			}
			export := i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, BatchID: "chatgpt-zh-CN-047", UnitKind: i18n.UnitKindPage, UnitIDs: o.UnitIDs, AllowReexport: true, PreviousSnapshotID: snapshot, SurfaceReopenID: o.ReopenID}
			batchPath := filepath.Join(root, "data/retranslation-runs/zh-CN", export.BatchID)
			for _, name := range []string{"old-unit", "snapshot", "receipt", "other-receipt", "finding", "proof", "A", "partial-scope"} {
				t.Run(name, func(t *testing.T) {
					bad := export
					c := inventory
					altered := *receipt
					altered.Units = append([]i18n.QualityCheckSurfaceReopenUnit(nil), receipt.Units...)
					switch name {
					case "old-unit":
						bad.UnitIDs = ids[2:3]
					case "snapshot":
						bad.PreviousSnapshotID = initial.SnapshotID
					case "receipt":
						bad.SurfaceReopenID = "missing-receipt"
					case "other-receipt":
						bad.SurfaceReopenID = "synthetic-surface"
					case "finding":
						altered.Units[0].Finding = "Other finding"
					case "A":
						altered.Units[0].PreviousRating = "A"
					case "proof":
						context := *altered.ReplacementContext
						authority := *context.QualityCheck
						authority.EngineeringProof = i18n.GlossaryArchiveReference{}
						context.QualityCheck = &authority
						altered.ReplacementContext = &context
					case "partial-scope":
						c.TranslationScope = o.UnitIDs
					}
					changed, _ := json.Marshal(altered)
					writeFixture(t, root, receiptPath, changed)
					defer writeFixture(t, root, receiptPath, receiptBytes)
					if _, err := i18n.ExportRetranslationBatch(root, &c, bad); err == nil {
						t.Fatal("invalid successor export accepted")
					}
					if _, err := os.Stat(batchPath); !os.IsNotExist(err) {
						t.Fatal("rejected export created batch", err)
					}
				})
			}
			// Explicit selection uses the published command contract; an unscoped
			// inventory also exercises the original failure path. Neither batch
			// receives output, so the second fixture export consumes no judgment.
			for _, catalog := range []*i18n.Catalog{current, &inventory} {
				e, err = i18n.ExportRetranslationBatch(root, catalog, export)
				if err != nil {
					t.Fatal(err)
				}
				for _, reader := range []*i18n.Catalog{current, &inventory} {
					_, m, err := i18n.ExportTranslationUnitGenerationBundle(root, reader, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
					if err != nil {
						t.Fatal(err)
					}
					if len(m.ExpectedOutputs) != 1 || m.ExpectedOutputs[0].UnitID != p.CurrentUnitID {
						t.Fatal("expanded successor revision scope")
					}
				}
				export.BatchID = "chatgpt-zh-CN-048"
			}
			if len(inventory.TranslationScope) != 0 {
				t.Fatal("mutated caller scope")
			}
			var m i18n.RetranslationBatchManifest
			b, err := os.ReadFile(filepath.Join(batchPath, "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &m); err != nil {
				t.Fatal(err)
			}
			u := m.Units[0]
			if len(m.Units) != 1 || u.UnitID != p.CurrentUnitID || u.PreviousRating != rating || u.PreviousFinding != finding || u.RevisionFeedbackSource != i18n.QCRepresentationFeedback || u.RevisionAuthorizationID != o.ReopenID {
				t.Fatal("lost exact QC revision authority", u)
			}
			got, err := os.ReadFile(filepath.Join(root, receiptPath))
			if err != nil || !bytes.Equal(got, receiptBytes) {
				t.Fatal("export rewrote reconciliation receipt", err)
			}
		})
	}
}
