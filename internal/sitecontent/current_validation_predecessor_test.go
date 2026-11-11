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

func currentValidationSurfacePredecessorFixture(t *testing.T, ordinaryBridge ...bool) (string, *i18n.Catalog, string, string) {
	t.Helper()
	root, c, batch := surfaceCapabilityFixture(t)
	discovery, e := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", batch, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	request := surfaceCapabilityRequest(t, discovery, true)
	transport, e := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", batch, &request, true)
	if e != nil {
		t.Fatal(e)
	}
	bundle, _, e := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: batch})
	if e != nil {
		t.Fatal(e)
	}
	id := c.TranslationScope[0]
	raw := bridgeEOF([]byte(strings.Replace(transport.Authority.Bindings[0].Projection.Template, "function", "函数", 1)))
	surfaceImport(t, root, c, bundle, map[string][]byte{id: raw})
	if p, e := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch}); e != nil || p.ValidationPassed != 1 {
		t.Fatal(p, e)
	}
	snapshot := "surface-current-validation-a"
	if _, _, e = i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot}); e != nil {
		t.Fatal(e)
	}
	previous := requestSnapshot(t, root, batch)
	if len(ordinaryBridge) != 0 && ordinaryBridge[0] {
		// Reuse the same sealed Surface fixture, with an existing immutable
		// ordinary QC bridge between Surface and current-validation invocations.
		cvpredBridgeFixtureContext(t, root, c)
		if _, e = i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: previous, UnitIDs: []string{id}, Rating: "B", Finding: "Improve ordinary prose.", ReviewerSession: "independent-synthetic-reviewer"}); e != nil {
			t.Fatal(e)
		}
		writeFixture(t, root, "engineering/cvpred-review.md", []byte("Independent Reviewer B: Improve ordinary prose.\n"))
		opts := i18n.SameUnitBridgeOptions{Locale: "zh-CN", ContentPackage: c.TranslationPackages[0].ID, BridgeID: "ordinary-before-cvpred", PreviousSnapshotID: snapshot, SnapshotSHA256: digest(mustReadFixture(t, root, "data/quality-check-snapshots/zh-CN/"+snapshot+"/manifest.json")), ReviewerEvidence: i18n.GlossaryArchiveReference{Path: "engineering/cvpred-review.md", SHA256: digest(mustReadFixture(t, root, "engineering/cvpred-review.md"))}, UnitIDs: []string{id}, Record: true}
		bridge, err := i18n.PrepareSameUnitRepresentationBridge(root, c, opts)
		if err != nil {
			t.Fatal("fixture inherited ordinary bridge", err)
		}
		next, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{id}, AllowReexport: true, PreviousSnapshotID: snapshot, RepresentationBridgeID: opts.BridgeID})
		if err != nil {
			t.Fatal(err)
		}
		bundle, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: next.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		surfaceImport(t, root, c, bundle, map[string][]byte{id: []byte(bridge.Receipt.SameUnitBridge.Bindings[0].Template)})
		if p, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: next.BatchID}); err != nil || p.ValidationPassed != 1 {
			t.Fatal("fixture ordinary bridge process", err)
		}
		previous, snapshot, batch = snapshot, "ordinary-before-current-validation-a", next.BatchID
		if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
			t.Fatal(err)
		}
	}
	if _, e = i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: previous, UnitIDs: []string{id}, Rating: "A", ReviewerSession: "independent-synthetic-reviewer"}); e != nil {
		t.Fatal(e)
	}
	if _, _, e = i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot}); e != nil {
		t.Fatal(e)
	}
	b, e := readRegular(root, "locales/zh-CN/locale.json")
	if e != nil {
		t.Fatal(e)
	}
	var metadata i18n.Locale
	if e = json.Unmarshal(b, &metadata); e != nil {
		t.Fatal(e)
	}
	metadata.LanguageAudit.Version++
	metadata.LanguageAudit.Policy.ProseScripts = []string{"Latin", "Han"}
	metadata.LanguageAudit.Policy.ForbiddenClausePunctuation = []string{"."}
	b, _ = json.MarshalIndent(metadata, "", "  ")
	writeFixture(t, root, "locales/zh-CN/locale.json", b)
	assessment, e := i18n.AssessCurrentTranslation(root, c, "zh-CN", snapshot)
	if e != nil || assessment.Affected != 1 || assessment.Stale != 0 || assessment.Unknown != 0 {
		t.Fatalf("current assessment: error=%v compatible=%d affected=%d stale=%d unknown=%d", e, assessment.Compatible, assessment.Affected, assessment.Stale, assessment.Unknown)
	}
	evidence, _ := json.MarshalIndent(assessment, "", "  ")
	writeFixture(t, root, "engineering/current-validator.json", evidence)
	sb, _ := readRegular(root, "data/quality-check-snapshots/zh-CN/"+snapshot+"/manifest.json")
	o := i18n.CurrentValidationReopenOptions{Locale: "zh-CN", ContentPackage: c.TranslationPackages[0].ID, ReopenID: "current-validation-representation", PreviousSnapshotID: snapshot, SnapshotSHA256: digest(sb), Evidence: i18n.GlossaryArchiveReference{Path: "engineering/current-validator.json", SHA256: digest(evidence)}, AssessmentIdentity: assessment.Identity}
	if _, _, e = i18n.RecordQualityCheckSurfaceReopen(root, c, i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: i18n.CurrentValidationAuthority, Locale: o.Locale, ReopenID: o.ReopenID, PreviousSnapshotID: snapshot, CurrentValidation: &o}); e != nil {
		t.Fatal(e)
	}
	revision, e := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: o.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{id}, UnitKind: i18n.UnitKindPage, PreviousSnapshotID: snapshot, SurfaceReopenID: o.ReopenID, AllowReexport: true})
	if e != nil {
		t.Fatal(e)
	}
	return root, c, revision.BatchID, batch
}

func TestCurrentValidationSealedPredecessorTransport(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, c, batch, oldBatch := currentValidationSurfacePredecessorFixture(t)
	options := i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: batch}
	bundle, m, e := i18n.ExportTranslationUnitGenerationBundle(root, c, options)
	if e != nil {
		t.Fatal("current-validation recovery export", e)
	}
	if _, e = i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); e != nil {
		t.Fatal("independent current-check", e)
	}
	files, e := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if e != nil {
		t.Fatal(e)
	}
	name := filepath.Base(m.ExpectedOutputs[0].BundlePath)
	var binding i18n.CurrentValidationPredecessor
	if e = json.Unmarshal(files["formal/current-validation-predecessor-transport/"+name+".json"], &binding); e != nil {
		t.Fatal(e)
	}
	if binding.Schema != "go-learning/current-validation-predecessor-transport/v1" || binding.Locale != "zh-CN" || binding.RecoveryClass != "DETERMINISTIC_REBASE_PASS" || binding.Snapshot.SelectedBatchID != oldBatch || binding.ReopenIdentity == "" || binding.ImplementationIdentity == "" || len(binding.Selected) != 1 || binding.Selected[0].RevisionFeedbackSource != i18n.CurrentValidationRevisionFeedback || binding.Selected[0].PreviousRating != "A" {
		t.Fatal("independent representation/language bindings lost")
	}
	oldRaw, e := readRegular(root, binding.Raw.Path)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Equal(oldRaw, files["formal/protected-candidates/"+name]) || !bytes.Equal(oldRaw, files["formal/historical-protected-candidates/"+name]) {
		t.Fatal("restricted predecessor raw was not preserved separately")
	}
	// Exact add-a-test family regression: Surface's P0 is not an ordinary Page
	// token set. The complete sealed source codec, not target prose, resolves it.
	if _, e := c.Adapted[0].Adapter.Restore(string(oldRaw)); e == nil || !strings.Contains(e.Error(), "locked token exact-set changed: ⟦P0⟧") {
		t.Fatal("fixture did not reproduce P0", e)
	}
	if len(files["formal/comment-revision-transport/"+name+".json"]) != 0 {
		t.Fatal("prose finding invented comment authority")
	}
	mb, _ := readRegular(root, "data/retranslation-runs/zh-CN/"+batch+"/manifest.json")
	var manifest i18n.RetranslationBatchManifest
	if json.Unmarshal(mb, &manifest) != nil {
		t.Fatal("manifest")
	}
	// Wrong language authority/scope must fail before any representation recovery.
	for _, mode := range []string{"wrong-reopen", "wrong-snapshot", "wrong-unit"} {
		t.Run(mode, func(t *testing.T) {
			bad := manifest
			bad.Units = append([]i18n.RetranslationBatchUnit{}, manifest.Units...)
			switch mode {
			case "wrong-reopen":
				bad.Units[0].RevisionAuthorizationID = "unrelated"
			case "wrong-snapshot":
				bad.Units[0].PreviousSnapshotID = "unrelated"
			case "wrong-unit":
				bad.Units[0].UnitID = "unrelated"
			}
			if _, e := i18n.InspectCurrentValidationPredecessors(root, c, "zh-CN", bad.Units); e == nil {
				t.Fatal("unrelated authority accepted")
			}
		})
	}
	paths := map[string]string{"raw": binding.Raw.Path, "candidate": binding.Snapshot.CandidatePath, "validation": binding.Snapshot.ValidationPath, "snapshot": binding.SnapshotManifest.Path, "reopen": binding.Reopen.Path, "manifest": "data/retranslation-runs/zh-CN/" + oldBatch + "/manifest.json", "import": "data/retranslation-runs/zh-CN/" + oldBatch + "/generation-imports/initial-manifest.json", "implementation": "internal/i18n/current_validation_predecessor.go"}
	for _, ref := range binding.Dependencies {
		if strings.HasPrefix(ref.Path, "data/generation-archives/") {
			paths["archive"] = ref.Path
		}
		if strings.HasPrefix(ref.Path, "data/surface-revision-transports/") {
			paths["codec-authority"] = ref.Path
		}
	}
	if paths["archive"] == "" {
		t.Fatal("sealed archive not bound")
	}
	for name, path := range paths {
		t.Run("tamper/"+name, func(t *testing.T) {
			b, e := readRegular(root, path)
			if e != nil {
				t.Fatal(e)
			}
			writeFixture(t, root, path, append(bytes.Clone(b), ' '))
			defer writeFixture(t, root, path, b)
			if _, e := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); e == nil {
				t.Fatal("tampered dependency remained CURRENT")
			}
		})
	}
	t.Run("changed-current-representation", func(t *testing.T) {
		copy := *c
		copy.Adapted = append([]i18n.AdaptedTranslationUnit{}, c.Adapted...)
		a := copy.Adapted[0].Adapter.(PageAdapter)
		a.Document.Units = append([]Unit{}, a.Document.Units...)
		a.Document.Units[0].Source += " new protection"
		copy.Adapted[0].Adapter = a
		if _, e := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, &copy, bundle); e == nil {
			t.Fatal("changed current representation remained CURRENT")
		}
	})
	t.Run("wrong-attempt", func(t *testing.T) {
		b, e := readRegular(root, binding.Snapshot.ValidationPath)
		if e != nil {
			t.Fatal(e)
		}
		var v i18n.RetranslationValidation
		if json.Unmarshal(b, &v) != nil {
			t.Fatal("validation")
		}
		v.Attempt++
		changed, _ := json.Marshal(v)
		writeFixture(t, root, binding.Snapshot.ValidationPath, changed)
		defer writeFixture(t, root, binding.Snapshot.ValidationPath, b)
		if _, e := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); e == nil {
			t.Fatal("wrong attempt remained CURRENT")
		}
	})
	// Import/process must choose the same recovered decoder, not merely succeed
	// at ZIP construction. Only the synthetic punctuation finding is repaired.
	replacement := bridgeEOF([]byte(strings.ReplaceAll(string(files["formal/protected-candidates/"+name]), ".", "。")))
	surfaceImport(t, root, c, bundle, map[string][]byte{c.TranslationScope[0]: replacement})
	if p, e := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch}); e != nil || p.ValidationPassed != 1 {
		t.Fatal("recovered response did not process", p, e)
	}
	if _, _, e := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "recovered-independent-qc-pending"}); e != nil {
		t.Fatal("recovered current candidate Snapshot", e)
	}
	if _, e := os.Stat(filepath.Join(root, "data/retranslation-runs/zh-CN", oldBatch, "manifest.json")); e != nil {
		t.Fatal(e)
	}
}

// This recovery capability never promotes even a unique target occurrence to
// authority. Newly protected prose atoms need separate source-owned lineage.
func TestCurrentValidationPredecessorUnsealedNewProtectionFailClosed(t *testing.T) {
	for _, tc := range []struct{ name, source, target string }{
		{"404", "Return HTTP 404 or 410.", "返回 HTTP 404 或 410。"},
		{"Marshal", "Custom Marshal and Unmarshal", "自定义 Marshal 和 Unmarshal"},
		{"package-identity", "Package A depends on package B.", "包 A 依赖包 B。"},
		{"ambiguous-package", "Package A depends on package B.", "包 A 依赖包 B；A 与 B 有不同记录。"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, after, raw := unboundPageFixture(t, tc.source, tc.target)
			candidate, e := before.Restore(string(raw))
			if e != nil {
				t.Fatal("valid historical fixture", e)
			}
			if _, e := after.RebaseRevisionPredecessor(before, raw); e == nil {
				t.Fatal("new prose identity silently bound")
			}
			if _, _, _, e := after.BridgeSurfaceRevision(before, raw, candidate); e == nil {
				t.Fatal("unsealed ordinary prose treated as Surface codec")
			}
		})
	}
}
