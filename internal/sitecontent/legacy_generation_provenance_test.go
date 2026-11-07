package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func legacyProvenanceFixture(t *testing.T, outputTransforms ...func([]byte) []byte) (string, *i18n.Catalog, i18n.LegacyGenerationProvenanceOptions, []byte) {
	t.Helper()
	root, c, o := semanticReviewFixture(t, outputTransforms...)
	b, m, err := i18n.ExportRepresentationReviewBundle(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	d := i18n.RepresentationReviewDecision{ReviewID: o.ReviewID, BundleIdentity: m.Identity, BundleSHA256: digest(b), Locale: o.Reconciliation.Locale, Package: o.Reconciliation.Package, OldUnitID: m.Replacement.OldUnitID, CurrentUnitID: o.UnitID, PreviousSnapshotID: o.Reconciliation.PreviousSnapshotID, FormalImpactSHA256: o.Reconciliation.Impact.SHA256, GenerationIndependent: true, ReviewerSession: "synthetic-independent-semantic", Provider: "chatgpt", Model: i18n.FormalChatGPTGenerationModel, Decision: "APPROVE_RECONCILIATION", Finding: "Synthetic whole interval approval only."}
	db, _ := json.Marshal(d)
	if _, _, err = i18n.RecordRepresentationReview(root, c, b, db); err != nil {
		t.Fatal(err)
	}
	ro := o.Reconciliation
	ro.LanguageReviewIDs = []string{o.ReviewID}
	ro.Apply = true
	if _, err = i18n.ReconcileTranslationUnitRepresentations(root, c, ro); err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "legacy-current"})
	if err != nil {
		t.Fatal(err)
	}
	sb, err := os.ReadFile(filepath.Join(root, "data/quality-check-snapshots/zh-CN/legacy-current/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var q i18n.QualityCheckSnapshotUnit
	for _, u := range snapshot.Units {
		if u.UnitID == o.UnitID {
			q = u
		}
	}
	path := "data/retranslation-runs/zh-CN/" + q.SelectedBatchID + "/generation-imports/initial-manifest.json"
	ib, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	var imported i18n.GenerationResultBundleManifest
	if err = json.Unmarshal(ib, &imported); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, "data/generation-archives/zh-CN", imported.GenerationBundleSHA256+".zip")
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(archivePath); err != nil {
		t.Fatal(err)
	}
	return root, c, i18n.LegacyGenerationProvenanceOptions{Locale: "zh-CN", ContentPackage: o.Reconciliation.Package, BatchID: q.SelectedBatchID, UnitID: m.Replacement.OldUnitID, CurrentUnitID: o.UnitID, Attempt: 1, TaskKind: imported.TaskKind, ImportReceipt: i18n.GlossaryArchiveReference{Path: path, SHA256: digest(ib)}, OriginalBundleSHA256: imported.GenerationBundleSHA256, PreviousSnapshotID: snapshot.SnapshotID, SnapshotSHA256: digest(sb)}, archive
}

func TestLegacyGenerationProvenanceMissingArchive(t *testing.T) {
	root, c, o, archive := legacyProvenanceFixture(t)
	p, err := i18n.PrepareLegacyGenerationProvenance(root, c, o, false, "")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := i18n.PrepareLegacyGenerationProvenance(root, c, o, false, "")
	if err != nil || p.Identity != p2.Identity || p.Recorded || p.Mutation != "none" {
		t.Fatal("inspection not deterministic", err)
	}
	if _, err = os.Stat(filepath.Join(root, "data/generation-provenance-authorities")); !os.IsNotExist(err) {
		t.Fatal("inspection wrote authority", err)
	}
	r := p.PlannedReceipt
	if !r.OriginalArchiveUnavailable || r.Capability != i18n.LegacyGenerationProvenanceCapability || len(r.UnknownFacts) != 6 {
		t.Fatal("invented original Bundle facts", r)
	}
	for _, d := range r.Dependencies {
		b, e := os.ReadFile(filepath.Join(root, d.Path))
		if e != nil || digest(b) != d.SHA256 || int64(len(b)) != d.Size {
			t.Fatal("dependency mismatch", d, e)
		}
	}
	qcPath := "data/quality-check-snapshots/zh-CN/semantic-predecessor/quality-check-results.json"
	qcBefore, _ := os.ReadFile(filepath.Join(root, qcPath))
	recorded, err := i18n.PrepareLegacyGenerationProvenance(root, c, o, true, p.Identity)
	if err != nil || !recorded.Recorded || recorded.Mutation != "known" {
		t.Fatal(recorded, err)
	}
	ref := i18n.GlossaryArchiveReference{Path: recorded.ReceiptPath, SHA256: recorded.ReceiptSHA256}
	verified, err := i18n.VerifyLegacyGenerationProvenance(root, c, ref, o)
	if err != nil || verified.ProvenanceSource != "legacy-recorded" || verified.Authority.Identity != p.Identity {
		t.Fatal(verified, err)
	}
	if _, err = i18n.PrepareLegacyGenerationProvenance(root, c, o, true, p.Identity); err == nil {
		t.Fatal("duplicate record allowed")
	}
	wrong := o
	wrong.CurrentUnitID = c.TranslationScope[0]
	if _, err = i18n.VerifyLegacyGenerationProvenance(root, c, ref, wrong); err == nil {
		t.Fatal("another Unit reused authority")
	}
	data, _ := os.ReadFile(filepath.Join(root, ref.Path))
	var widened i18n.LegacyGenerationProvenance
	json.Unmarshal(data, &widened)
	widened.Capability = "whole-batch"
	widened.Identity, widened.AuthorityID = "", ""
	identityBytes, _ := json.Marshal(widened)
	widened.Identity = digest(identityBytes)
	widened.AuthorityID = widened.Identity
	bad, _ := json.Marshal(widened)
	bad = append(bad, '\n')
	badRef := i18n.GlossaryArchiveReference{Path: "data/generation-provenance-authorities/zh-CN/" + widened.Identity + ".json", SHA256: digest(bad)}
	writeFixture(t, root, badRef.Path, bad)
	if _, err = i18n.VerifyLegacyGenerationProvenance(root, c, badRef, o); err == nil {
		t.Fatal("scope widening allowed")
	}
	if err = os.Remove(filepath.Join(root, badRef.Path)); err != nil {
		t.Fatal(err)
	}
	// Unknown target-mapping fields are rejected even with an exact file SHA.
	var unknown map[string]any
	json.Unmarshal(data, &unknown)
	unknown["target_mapping"] = map[string]string{"invented": "forbidden"}
	bad, _ = json.Marshal(unknown)
	writeFixture(t, root, ref.Path, bad)
	unknownRef := ref
	unknownRef.SHA256 = digest(bad)
	if _, err = i18n.VerifyLegacyGenerationProvenance(root, c, unknownRef, o); err == nil {
		t.Fatal("target mapping accepted")
	}
	writeFixture(t, root, ref.Path, data)
	archivePath := "data/generation-archives/zh-CN/" + o.OriginalBundleSHA256 + ".zip"
	writeFixture(t, root, archivePath, archive)
	verified, err = i18n.VerifyLegacyGenerationProvenance(root, c, ref, o)
	if err != nil || verified.ProvenanceSource != "normal-archive" || verified.Archive == nil {
		t.Fatal("normal archive not preferred", verified, err)
	}
	if _, err = i18n.PrepareLegacyGenerationProvenance(root, c, o, false, ""); err == nil || !strings.Contains(err.Error(), "normal archive") {
		t.Fatal("archive-present legacy accepted", err)
	}
	writeFixture(t, root, archivePath, []byte("wrong archive"))
	if _, err = i18n.VerifyLegacyGenerationProvenance(root, c, ref, o); err == nil {
		t.Fatal("wrong archive SHA accepted")
	}
	after, _ := os.ReadFile(filepath.Join(root, qcPath))
	if !bytes.Equal(after, qcBefore) {
		t.Fatal("language authority changed")
	}
}

func TestLegacyGenerationProvenanceDependencyDrift(t *testing.T) {
	root, c, o, _ := legacyProvenanceFixture(t)
	p, err := i18n.PrepareLegacyGenerationProvenance(root, c, o, false, "")
	if err != nil {
		t.Fatal(err)
	}
	r := p.PlannedReceipt
	paths := map[string]string{"import": o.ImportReceipt.Path, "manifest": r.HistoricalManifest.Path, "input": r.HistoricalInput.Path, "raw": r.Raw.Path, "candidate": r.Candidate.Path, "validation": r.Validation.Path, "Snapshot": r.Snapshot.Path, "reconciliation": r.Reconciliation.Path, "semantic-receipt": r.SemanticReceipt.Path, "semantic-Bundle": r.SemanticBundle.Path, "semantic-decision": r.SemanticDecision.Path, "historical-representation": r.HistoricalRepresentation.Path, "source": r.CurrentSource.Path, "result": r.Result.Path}
	for name, path := range paths {
		t.Run(name, func(t *testing.T) {
			original, e := os.ReadFile(filepath.Join(root, path))
			if e != nil {
				t.Fatal(e)
			}
			if e = os.Chmod(filepath.Join(root, path), 0644); e != nil {
				t.Fatal(e)
			}
			defer writeFixture(t, root, path, original)
			writeFixture(t, root, path, append(append([]byte{}, original...), byte(' ')))
			if _, e = i18n.PrepareLegacyGenerationProvenance(root, c, o, false, ""); e == nil {
				t.Fatal("changed dependency accepted")
			}
		})
	}
	for _, name := range []string{"missing-import", "missing-raw", "provider", "model", "batch", "attempt", "task", "raw-hash", "raw-size", "install-path", "wrong-current-unit", "wrong-historical-unit", "wrong-package", "wrong-current-input"} {
		t.Run(name, func(t *testing.T) {
			changed := o
			path := o.ImportReceipt.Path
			b, _ := os.ReadFile(filepath.Join(root, path))
			var receipt i18n.GenerationResultBundleManifest
			json.Unmarshal(b, &receipt)
			defer writeFixture(t, root, path, b)
			switch name {
			case "missing-import":
				os.Remove(filepath.Join(root, path))
			case "missing-raw":
				raw, _ := os.ReadFile(filepath.Join(root, r.Raw.Path))
				defer writeFixture(t, root, r.Raw.Path, raw)
				os.Remove(filepath.Join(root, r.Raw.Path))
			case "provider":
				receipt.Provider = "codex"
			case "model":
				receipt.Model = "invented-model"
			case "batch":
				changed.BatchID = "chatgpt-zh-CN-999"
			case "attempt":
				changed.Attempt = 2
			case "task":
				changed.TaskKind = "translation-unit-retry"
			case "raw-hash":
				for i := range receipt.Outputs {
					if receipt.Outputs[i].BundlePath == r.Output.BundlePath {
						receipt.Outputs[i].SHA256 = strings.Repeat("0", 64)
					}
				}
			case "raw-size":
				for i := range receipt.Outputs {
					if receipt.Outputs[i].BundlePath == r.Output.BundlePath {
						receipt.Outputs[i].Size++
					}
				}
			case "install-path":
				v, _ := os.ReadFile(filepath.Join(root, r.Validation.Path))
				defer writeFixture(t, root, r.Validation.Path, v)
				writeFixture(t, root, r.Validation.Path, bytes.ReplaceAll(v, []byte("raw-responses/"), []byte("elsewhere/")))
			case "wrong-current-unit":
				changed.CurrentUnitID = c.TranslationScope[0]
			case "wrong-historical-unit":
				changed.UnitID = c.TranslationScope[0]
			case "wrong-package":
				changed.ContentPackage = "missing-package"
			case "wrong-current-input":
				u, _ := c.Unit(o.CurrentUnitID)
				a := u.Adapter.(PageAdapter)
				a.Document.Units = append([]Unit(nil), a.Document.Units...)
				a.Document.Units[0].Source += "tampered"
				// Mutate the explicit source file; catalog cannot substitute its cached source.
				source, _ := os.ReadFile(filepath.Join(root, r.CurrentSource.Path))
				defer writeFixture(t, root, r.CurrentSource.Path, source)
				writeFixture(t, root, r.CurrentSource.Path, append(source, byte('x')))
			}
			if name == "provider" || name == "model" || name == "raw-hash" || name == "raw-size" {
				b, _ = json.Marshal(receipt)
				writeFixture(t, root, path, b)
				changed.ImportReceipt.SHA256 = digest(b)
			}
			if _, e := i18n.PrepareLegacyGenerationProvenance(root, c, changed, false, ""); e == nil {
				t.Fatal("unsafe provenance accepted")
			}
		})
	}
}

func TestLegacyGenerationProvenanceJSONReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_LEGACY_PROVENANCE_READER_PROBE") != "1" {
		t.Skip("explicit real single-Unit readonly regression")
	}
	root := repositoryRoot(t)
	c, err := i18n.BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	o := i18n.LegacyGenerationProvenanceOptions{Locale: "zh-CN", ContentPackage: "learn-docs-v1", BatchID: "chatgpt-zh-CN-031", UnitID: "99824b13c26ffe219230ad0171e93d850003d16aa9f780c2f06c41bc46845cb3", CurrentUnitID: "b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c", Attempt: 1, TaskKind: "translation-unit-revision", ImportReceipt: i18n.GlossaryArchiveReference{Path: "data/retranslation-runs/zh-CN/chatgpt-zh-CN-031/generation-imports/initial-manifest.json", SHA256: "e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26"}, OriginalBundleSHA256: "6972502acc0ab83f6a9382b9bb6330cca212d74db56e6af29e204e895e37d542", PreviousSnapshotID: "learn-docs-v1-qc-023", SnapshotSHA256: "2fdde982c05eb53d54ff4c353780e07055b9e4ea19ae28105f52f4abdaa7c7d9"}
	p, err := i18n.PrepareLegacyGenerationProvenance(root, c, o, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Recorded || p.Mutation != "none" {
		t.Fatal("readonly regression wrote authority")
	}
	t.Logf("planned_identity=%s dependencies=%d no_record=true no_consumer_integration=true", p.Identity, len(p.PlannedReceipt.Dependencies))
}
