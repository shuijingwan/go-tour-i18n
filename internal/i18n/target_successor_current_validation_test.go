package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeSuccessorCurrentValidationIncrementalAuthorization(t *testing.T) {
	const locale, previous, next = "zh-CN", "native-original-a", "native-incremental-a"
	root, _, _ := futureContentFixtureSize(t, 1)
	c := retranslationTestCatalog(3) // Minimal equivalent of N-1 carry + one fresh A.
	c.Pages[0].Source = bytes.ReplaceAll(c.Pages[0].Source, []byte(" on this page."), []byte(" on this page!!"))
	c.Pages[0].SourceSHA256 = sum(c.Pages[0].Source)
	copyTestLanguageAuditMachineAuthority(t, root)
	copyBundleAuthority(t, root, currentValidationReopenDependencies)
	copyBundleAuthority(t, root, append(append([]string{}, commentGateAuthorityPaths...), "internal/i18n/validate.go", "internal/i18n/go_example_validate.go", "internal/i18n/go_example_protect.go", "internal/i18n/translate_protect.go", "internal/i18n/preformatted.go"))
	ids := []string{c.Pages[0].ID, c.Pages[1].ID, c.Pages[2].ID}
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: locale, Generator: RetranslationGeneratorChatGPT, UnitIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: locale, BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string][]byte{}
	for _, u := range manifest.ExpectedOutputs {
		outputs[u.UnitID] = []byte(strings.NewReplacer("* Page", "* 页面", "Use ", "本页使用 ", " on this page.", "。", " on this page!!", "！！").Replace(string(files["formal/inputs/"+filepath.Base(u.BundlePath)])))
	}
	futureImport(t, root, c, bundle, outputs)
	if result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: locale, BatchID: e.BatchID}); err != nil || result.ValidationPassed != len(ids) {
		t.Fatal("initial passed invocation", result, err)
	}
	materializeSnapshotSources(t, root, c)
	futureSnapshot(t, root, c, previous)
	recordQualityCheckRatings(t, root, c, previous, "", "A", ids)
	if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: locale, SnapshotID: previous}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := EstablishCurrentTranslationAuthorization(root, c, locale, previous); err != nil {
		t.Fatal(err)
	}
	baseline, err := readQualityCheckSnapshot(root, locale, previous, false)
	if err != nil {
		t.Fatal(err)
	}
	writePromotionStatus(t, root, c, "")
	for _, u := range baseline.Units {
		b, err := readCompatibilityFile(root, u.CandidatePath)
		if err != nil {
			t.Fatal(err)
		}
		writeSnapshotSource(t, root, canonicalCandidatePath(locale, u.UnitID), canonicalizeCandidateEOF(b))
	}
	// A real active policy failure affects only one accepted candidate. A
	// byte-only glossary epoch leaves the other original judgments eligible
	// through direct current re-attestation, without a compatibility edge log.
	glossary, err := readCompatibilityFile(root, "locales/"+locale+"/glossary.yaml")
	if err != nil {
		t.Fatal(err)
	}
	compatibilityChange(t, root, "# current endpoint\n"+string(glossary), "native-current-endpoint")
	metadata, err := os.ReadFile(filepath.Join(root, "locales", locale, "locale.json"))
	if err != nil {
		t.Fatal(err)
	}
	var policy Locale
	if err := json.Unmarshal(metadata, &policy); err != nil {
		t.Fatal(err)
	}
	policy.LanguageAudit.Version++
	policy.LanguageAudit.Policy.SentenceEnd = map[string]string{"！": "。"}
	writeSnapshotSource(t, root, "locales/"+locale+"/locale.json", mustJSON(policy))
	a, err := AssessCurrentTranslation(root, c, locale, previous)
	if err != nil {
		t.Fatal(err)
	}
	if a.Compatible != 2 || a.Affected != 1 || a.Stale != 0 || a.Unknown != 0 {
		t.Fatalf("current failure scope: compatible=%d affected=%d stale=%d unknown=%d", a.Compatible, a.Affected, a.Stale, a.Unknown)
	}
	o := assessmentReopenOptions(t, root, a)
	r, _, err := RecordQualityCheckSurfaceReopen(root, c, QualityCheckSurfaceReopenOptions{AuthoritySource: CurrentValidationAuthority, Locale: locale, ReopenID: o.ReopenID, PreviousSnapshotID: previous, CurrentValidation: &o})
	if err != nil || r.UnitCount != 1 || r.Units[0].UnitID != ids[0] || r.Units[0].PreviousRating != "A" {
		t.Fatal("current-validation authority", err)
	}
	e, err = ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: locale, Generator: RetranslationGeneratorChatGPT, UnitIDs: ids[:1], AllowReexport: true, PreviousSnapshotID: previous, SurfaceReopenID: r.ReopenID})
	if err != nil {
		t.Fatal(err)
	}
	bundle, manifest, err = ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: locale, BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	_, files, err = readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	raw := files["formal/protected-candidates/"+filepath.Base(manifest.ExpectedOutputs[0].BundlePath)]
	if !bytes.Contains(raw, []byte("！！")) {
		t.Fatal("fixture lost the exact active failure")
	}
	futureImport(t, root, c, bundle, map[string][]byte{ids[0]: bytes.ReplaceAll(raw, []byte("！！"), []byte("。"))})
	if result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: locale, BatchID: e.BatchID}); err != nil || result.ValidationPassed != 1 {
		t.Fatal("successor active validation", result, err)
	}
	futureSnapshot(t, root, c, next)
	if scope, err := BuildQualityCheckScope(root, c, QualityCheckScopeOptions{Locale: locale, SnapshotID: next, PreviousSnapshotID: previous}); err != nil || scope.CarryForwardCount != 2 || scope.PendingCount != 1 {
		t.Fatal("reopen must not invent a fresh Reviewer A", scope, err)
	}
	recordQualityCheckRatings(t, root, c, next, previous, "A", ids[:1])
	if scope, err := BuildQualityCheckScope(root, c, QualityCheckScopeOptions{Locale: locale, SnapshotID: next, PreviousSnapshotID: previous}); err != nil || scope.CurrentResultCount != 1 || scope.CarryForwardCount != 2 || scope.ACount != 3 || !scope.ReadyForFinalization {
		t.Fatal("incremental A-only scope", scope, err)
	}
	final, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: locale, SnapshotID: next, PreviousSnapshotID: previous})
	if err != nil {
		t.Fatal(err)
	}
	for i, u := range final.Units {
		origin := previous
		if i == 0 {
			origin = next
		}
		if u.SourceSnapshotID != origin || u.Rating != "A" {
			t.Fatal("fresh QC and original Reviewer judgments conflated", u.UnitID)
		}
	}
	if plan, err := PromoteRetranslation(root, c, RetranslationPromoteOptions{Locale: locale, SnapshotID: next, Apply: true}); err != nil || plan.ChangedCount != 1 || plan.UnchangedCount != 2 {
		t.Fatal("native incremental installation", plan, err)
	}
	d, err := DiagnoseQualityCheckAuthorization(root, c, locale, next)
	if err != nil || !d.CanonicalAuthorized || d.ReviewerACount != 3 || d.SuccessorKind != "native-promotion-receipt" {
		t.Fatal("incremental successor historical A-only authorization", d, err)
	}
	a, err = AssessCurrentTranslation(root, c, locale, next)
	if err != nil || !a.Ready() {
		t.Fatal("current eligibility independently required", err)
	}
	qc := "data/quality-check-snapshots/" + locale + "/" + next + "/quality-check-results.json"
	for _, path := range []string{qc, final.QCResults[0].Path, baseline.Units[1].CandidatePath, final.Units[0].Snapshot.ValidationPath, final.Units[0].Snapshot.SourcePath, canonicalCandidatePath(locale, ids[0])} {
		original, err := readCompatibilityFile(root, path)
		if err != nil {
			t.Fatal(err)
		}
		writeSnapshotSource(t, root, path, append(bytes.Clone(original), ' '))
		if d, err := DiagnoseQualityCheckAuthorization(root, c, locale, next); err == nil && d.CanonicalAuthorized {
			t.Fatal("tampered immutable evidence authorized", path)
		}
		writeSnapshotSource(t, root, path, original)
	}
	qcBytes, err := readCompatibilityFile(root, qc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, qc)); err != nil {
		t.Fatal(err)
	}
	if d, err := DiagnoseQualityCheckAuthorization(root, c, locale, next); err == nil && d.CanonicalAuthorized {
		t.Fatal("missing fresh Reviewer evidence replaced by reopen authority")
	}
	writeSnapshotSource(t, root, qc, qcBytes)
	// Historical A remains provable after an active validator change, but it
	// cannot stand in for the separate current eligibility assessment.
	policy.LanguageAudit.Version++
	policy.LanguageAudit.Policy.SentenceEnd = map[string]string{"。": "."}
	writeSnapshotSource(t, root, "locales/"+locale+"/locale.json", mustJSON(policy))
	p, err := historicalSuccessorCompatibility(root, locale, c, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifySuccessorFinalization(root, locale, next, c, p); err != nil {
		t.Fatal("today's policy revoked original A", err)
	}
	a, err = AssessCurrentTranslation(root, c, locale, next)
	if err != nil || a.Ready() || a.Affected != 3 {
		t.Fatal("historical A authorized current canonical validation failure", err)
	}
}
