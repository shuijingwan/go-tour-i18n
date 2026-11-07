package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func recoverySchemaFixture() InstalledDerivedTransportRecovery {
	id := sum([]byte("synthetic unit"))
	base := "data/retranslation-runs/fr-FR/chatgpt-fr-FR-001/"
	ref := func(path string) RecoveryFileReference { return recoveryMemberReference(path, []byte(path)) }
	r := RetranslationBatchUnit{UnitID: id, UnitKind: UnitKindPage, SourcePath: "fixtures/page.md", SourceSHA256: sum([]byte("source")), InputPath: "inputs/" + id + ".article", InputSHA256: sum([]byte("input")), ProtectedTokenCount: 1, PreviousSnapshotID: "old-snapshot", RevisionFeedbackSource: CurrentValidationRevisionFeedback, RevisionAuthorizationID: "old-reopen", PreviousRating: "A", PreviousFinding: "synthetic exact finding", PreviousReviewPath: "data/quality-check-reopens/fr-FR/old-reopen.json", PreviousReviewSHA256: sum([]byte("reopen"))}
	archive := ref(importedGenerationArchive("fr-FR", sum([]byte("bundle"))))
	archive.SHA256 = sum([]byte("bundle"))
	a := InstalledDerivedTransportRecovery{Schema: TransportRecoverySchema, Capability: TransportRecoveryCapability, Locale: "fr-FR", ContentPackage: RecoveryPackage{ID: "fixture-docs", AuthoritySHA256: sum([]byte("package")), ScopeIdentitySHA256: sum([]byte("scope")), Units: []RecoverySourceUnit{{UnitID: id, Kind: UnitKindPage, Source: ref("fixtures/page.md")}}},
		Predecessor:       RecoveryInvocation{BatchID: "chatgpt-fr-FR-001", Manifest: ref(base + "manifest.json"), Result: ref(base + "result.json"), TaskKind: "translation-unit-revision", ImportReceipt: ref(base + "generation-imports/initial-manifest.json"), Archive: archive, InputIdentitySHA256: sum([]byte("input identity")), Provider: "chatgpt", Model: FormalChatGPTGenerationModel, FinalUnits: []RecoveryFinalUnit{{UnitID: id, Attempt: 1, Raw: ref(base + "raw-responses/" + id + ".article"), Candidate: ref(base + "candidates/" + id + ".article"), Validation: ref(base + "validation/" + id + ".json")}}},
		LanguageAuthority: RecoveryLanguageAuthority{Reopen: ref(r.PreviousReviewPath), ReopenIdentity: sum([]byte("reopen identity")), Snapshot: ref("data/quality-check-snapshots/fr-FR/old-snapshot/manifest.json"), Scope: []RetranslationBatchUnit{r}},
		RecoveryUnits:     RecoveryClosure{ContextRoot: archive.SHA256, UnitIDs: []string{id}, Edges: []RecoveryContextEdge{{id, archive.SHA256, "complete-installed-decoder-context"}}},
		AllowedAction:     RecoveryAllowedAction{Operation: "retranslation-export-with-transport-recovery", Provider: "chatgpt", Model: FormalChatGPTGenerationModel, NewLanguageGenerationRequired: true}}
	old, new := ref("old.json"), ref("current.json")
	a.DirectDrift = []RecoveryDirectDrift{{UnitID: id, Classification: "construction-dependency-drift", OldProof: old, OldProofIdentity: sum([]byte("old proof")), CurrentProof: new, CurrentProofIdentity: sum([]byte("new proof")), OldConstructionIdentity: sum([]byte("old construction")), CurrentConstructionIdentity: sum([]byte("new construction")), Dependencies: []RecoveryDependencyDelta{{Role: "approved-construction-implementation", Old: ref("internal/i18n/candidate_snapshot_outputs.go"), Current: ref("internal/i18n/candidate_snapshot_outputs.go"), Declarations: []string{"func:snapshotRevisionEvidenceCatalog"}}}}}
	a.DirectDrift[0].Dependencies[0].Old.SHA256 = sum([]byte("old source"))
	a.Construction = ref("construction.json")
	a.Members = []RecoveryFileReference{a.Construction, new, old}
	a.CurrentEpoch = []RecoveryDependency{{Role: "approved-construction-implementation", Ref: ref("internal/i18n/candidate_snapshot_outputs.go")}}
	a.CurrentAssessment = RecoveryAssessment{HistoricalIntegrity: true, LanguageAuthorityIdentity: a.LanguageAuthority.ReopenIdentity, OriginalBaseline: &CurrentTranslationAuthorization{Schema: CurrentTranslationAuthorizationSchema}, CurrentCandidateValidation: []RecoveryValidationAssessment{{id, a.Predecessor.FinalUnits[0].Candidate.SHA256, sum([]byte("validator")), "passed"}}}
	a.Identity = transportRecoveryIdentity(a)
	return a
}

func TestTransportRecoveryStrictProtocols(t *testing.T) {
	a := recoverySchemaFixture()
	if _, err := decodeRecoveryAuthority(mustJSON(a)); err != nil {
		t.Fatal("valid typed schema fixture", err)
	}
	for _, tc := range []struct {
		name   string
		change func(*InstalledDerivedTransportRecovery)
	}{
		{"unknown-schema", func(a *InstalledDerivedTransportRecovery) {
			a.Schema = "go-learning/installed-derived-transport-recovery/v2"
		}},
		{"unknown-capability", func(a *InstalledDerivedTransportRecovery) { a.Capability = "copy-old-raw" }},
		{"empty-D", func(a *InstalledDerivedTransportRecovery) { a.DirectDrift = nil }},
		{"incomplete-R", func(a *InstalledDerivedTransportRecovery) { a.RecoveryUnits.UnitIDs = nil }},
		{"foreign-D", func(a *InstalledDerivedTransportRecovery) { a.DirectDrift[0].UnitID = "unselected" }},
		{"duplicate-D", func(a *InstalledDerivedTransportRecovery) { a.DirectDrift = append(a.DirectDrift, a.DirectDrift[0]) }},
		{"native-Tour", func(a *InstalledDerivedTransportRecovery) { a.ContentPackage.ID = tourNativeTranslationPackage }},
		{"old-raw-replay", func(a *InstalledDerivedTransportRecovery) { a.AllowedAction.OldRawReplayAllowed = true }},
		{"provider-switch", func(a *InstalledDerivedTransportRecovery) { a.AllowedAction.Provider = "codex" }},
		{"wrong-language-origin", func(a *InstalledDerivedTransportRecovery) {
			a.LanguageAuthority.Scope[0].RevisionFeedbackSource = "quality_check"
		}},
		{"escaping-member", func(a *InstalledDerivedTransportRecovery) { a.Members[0].Path = "../escape" }},
		{"missing-context-edge", func(a *InstalledDerivedTransportRecovery) { a.RecoveryUnits.Edges = nil }},
		{"wrong-context-root", func(a *InstalledDerivedTransportRecovery) { a.RecoveryUnits.ContextRoot = sum([]byte("wrong archive")) }},
		{"extra-unaffected-unit", func(a *InstalledDerivedTransportRecovery) {
			a.RecoveryUnits.NonDirectUnitIDs = append(a.RecoveryUnits.NonDirectUnitIDs, a.DirectDrift[0].UnitID)
		}},
		{"mixed-unit-kind", func(a *InstalledDerivedTransportRecovery) { a.LanguageAuthority.Scope[0].UnitKind = UnitKindExample }},
		{"scope-over-sixty", func(a *InstalledDerivedTransportRecovery) {
			for len(a.RecoveryUnits.UnitIDs) <= MaxRetranslationExportLimit {
				a.RecoveryUnits.UnitIDs = append(a.RecoveryUnits.UnitIDs, sum([]byte(fmt.Sprint(len(a.RecoveryUnits.UnitIDs)))))
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var x InstalledDerivedTransportRecovery
			if err := json.Unmarshal(mustJSON(a), &x); err != nil {
				t.Fatal(err)
			}
			tc.change(&x)
			x.Identity = transportRecoveryIdentity(x)
			if _, err := decodeRecoveryAuthority(mustJSON(x)); err == nil {
				t.Fatal("invalid recovery schema accepted")
			}
		})
	}
	for _, bad := range [][]byte{bytes.Replace(mustJSON(a), []byte("{schema"), []byte("{"), 1), append([]byte(`{"unrecognized":1,`), mustJSON(a)[1:]...), append([]byte(`{"schema":"duplicate",`), mustJSON(a)[1:]...)} {
		if bytes.Equal(bad, mustJSON(a)) {
			continue
		}
		if _, err := decodeRecoveryAuthority(bad); err == nil {
			t.Fatal("unknown/duplicate field accepted")
		}
	}
	for _, path := range []string{"/absolute.json", "../escape.json", "safe/../../escape.json", `safe\authority.json`} {
		if validRecoveryFileReference(RecoveryFileReference{path, sum(nil), 0}) {
			t.Fatal("unsafe reference accepted", path)
		}
	}
}

func TestTransportRecoveryTypedConstructionEncoding(t *testing.T) {
	// This closed fixture reader represents the source adapter's typed inverse;
	// the production construction decoder must invoke it before hashing identity.
	type codec struct {
		Schema string `json:"schema"`
		Text   string `json:"text"`
	}
	type child struct {
		Start int    `json:"start"`
		Role  string `json:"role"`
	}
	c := &Catalog{Pages: []Page{{ID: "unit"}}}
	c.DerivedTransportCodecReader = func(a TranslationUnitAdapter, b, inventory []byte) (any, any, TranslationUnitAdapter, error) {
		var v codec
		var children []child
		if err := contentidentity.StrictJSON(b, &v); err != nil {
			return nil, nil, nil, err
		}
		if err := contentidentity.StrictJSON(inventory, &children); err != nil {
			return nil, nil, nil, err
		}
		if v.Schema != "fixture-codec/v1" {
			return nil, nil, nil, fmt.Errorf("unknown codec version")
		}
		return v, children, nil, nil
	}
	x := DerivedTransportConstruction{Schema: DerivedTransportConstructionSchema,
		Producer:        DerivedTransportProducer{ID: ApprovedIntervalTransportSchema, Version: 2},
		SemanticContext: DerivedTransportSemanticContext{OrderedScope: []RecoverySourceUnit{{UnitID: "unit"}}},
		Decoders:        []DerivedDecoderConstruction{{UnitID: "unit", Variant: "approved-interval", Approved: &DerivedApprovedConstruction{Codec: codec{"fixture-codec/v1", "原文 "}, Children: []child{{1, "literal"}}}}}}
	x.Producer.Identity = sum(mustJSON(x.Producer.Implementation))
	x.Identity = derivedConstructionIdentity(x)
	b := mustJSON(x)
	var compact bytes.Buffer
	if err := json.Compact(&compact, b); err != nil {
		t.Fatal(err)
	}
	for _, wire := range [][]byte{b, compact.Bytes(), bytes.ReplaceAll(compact.Bytes(), []byte(`"schema":"fixture-codec/v1","text":"原文 "`), []byte(`"text":"\u539f\u6587 ","schema":"fixture-codec/v1"`))} {
		v, err := decodeDerivedTransportConstruction(c, wire)
		if err != nil || v.Identity != x.Identity {
			t.Fatal("encoding created a construction epoch", err)
		}
	}
	for _, wire := range [][]byte{
		bytes.ReplaceAll(compact.Bytes(), []byte("原文 "), []byte("原文")),
		bytes.ReplaceAll(compact.Bytes(), []byte("fixture-codec/v1"), []byte("fixture-codec/v9")),
		bytes.ReplaceAll(compact.Bytes(), []byte(`"schema":"fixture-codec/v1"`), []byte(`"unrecognized":true,"schema":"fixture-codec/v1"`)),
		bytes.ReplaceAll(compact.Bytes(), []byte(`"start":1`), []byte(`"start":1,"extra":1`)),
	} {
		if _, err := decodeDerivedTransportConstruction(c, wire); err == nil {
			t.Fatal("semantic/unknown inverse payload accepted with old identity")
		}
	}
	old := &DerivedTransportConstruction{SemanticContext: x.SemanticContext, Decoders: x.Decoders}
	current := *old
	current.Decoders = append([]DerivedDecoderConstruction(nil), old.Decoders...)
	current.Decoders[0].ProtectedBaseline = "different actual response contract"
	if _, err := classifyRecoveryConstructionDifference(old, &current, nil, nil, ""); err == nil {
		t.Fatal("unclassified decoder payload change authorized recovery")
	}
}

func TestTransportRecoveryConstructionDeltaClassification(t *testing.T) {
	path := "internal/i18n/candidate_snapshot_outputs.go"
	old := []byte("package i18n\nfunc snapshotRevisionEvidenceCatalog() int { return 1 }\nfunc unrelated() int { return 3 }\n")
	current := bytes.Replace(old, []byte("return 1"), []byte("return 2"), 1)
	decls, err := classifyApprovedDependencyDelta(path, old, current)
	if err != nil || len(decls) != 1 || decls[0] != "func:snapshotRevisionEvidenceCatalog" {
		t.Fatal("owned typed construction delta", decls, err)
	}
	for _, bad := range [][]byte{append(append([]byte{}, old...), []byte("// encoding-only comment\n")...), bytes.Replace(current, []byte("return 3"), []byte("return 4"), 1), append([]byte("//go:build linux\n\n"), current...)} {
		if _, err := classifyApprovedDependencyDelta(path, old, bad); err == nil {
			t.Fatal("encoding-only or mixed-role change authorized recovery")
		}
	}
	if _, err := classifyApprovedDependencyDelta("unregistered.go", old, current); err == nil {
		t.Fatal("path spelling granted a producer role")
	}
	if _, err := recoveryArchiveReference(map[string][]byte{"current.go": current}, GlossaryArchiveReference{path, sum(old)}); err == nil {
		t.Fatal("current source filled missing old construction facts")
	}
}

func TestTransportRecoveryNamespaceAndImmutablePublication(t *testing.T) {
	root := t.TempDir()
	unlock, err := lockRetranslationBatchNamespace(root)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := lockRetranslationBatchNamespace(root); err == nil {
		other()
		t.Fatal("parallel exporter admitted")
	}
	unlock()
	unlock, err = lockRetranslationBatchNamespace(root)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("namespace lock changed repository bytes", err)
	}
	path := "data/authority/member.json"
	b := []byte("sealed bytes")
	if err := publishRecoverySealedFile(root, path, b, false); err != nil {
		t.Fatal(err)
	}
	if err := publishRecoverySealedFile(root, path, b, false); err == nil {
		t.Fatal("duplicate publication was a no-op")
	}
	if err := publishRecoverySealedFile(root, path, b, true); err != nil {
		t.Fatal("exact sealed continuation rejected", err)
	}
	if err := publishRecoverySealedFile(root, path, []byte("changed"), true); err == nil {
		t.Fatal("resume overwrote sealed bytes")
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := publishRecoverySealedFile(root, "escape/authority.json", b, false); err == nil {
		t.Fatal("symlink parent accepted")
	}
}

func TestTransportRecoveryCommittedAuthorityIntegrity(t *testing.T) {
	root := t.TempDir()
	a := recoverySchemaFixture()
	base := transportRecoveryDirectory(a.Locale, a.Identity)
	for _, m := range a.Members {
		if err := publishRecoverySealedFile(root, base+"/"+m.Path, []byte(m.Path), false); err != nil {
			t.Fatal(err)
		}
	}
	ab := mustJSON(a)
	ref := recoveryAuthorityRef(&a, ab)
	if err := publishRecoverySealedFile(root, ref.Path, ab, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readRecoveryExactAuthority(root, ref); err == nil {
		t.Fatal("uncommitted authority directory consumed")
	}
	l := recoveryLink{Schema: "go-learning/transport-recovery-link/v1", Kind: "root", Locale: a.Locale, RootInvocationSHA256: a.Predecessor.Archive.SHA256, Next: ref}
	l.Identity = recoveryLinkIdentity(l)
	if err := publishRecoverySealedFile(root, recoveryRootLinkPath(&a), mustJSON(l), false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readRecoveryExactAuthority(root, ref); err != nil {
		t.Fatal("exact committed authority", err)
	}
	if _, err := ReadTransportRecoveryReference(root, ref.Path, sum([]byte("wrong SHA"))); err == nil {
		t.Fatal("authority SHA override accepted")
	}
	if _, err := verifyRecoveryRecordPrior(root, &a); err == nil {
		t.Fatal("second root accepted")
	}
	next := a
	next.Supersedes = &ref
	next.Identity = transportRecoveryIdentity(next)
	if _, err := verifyRecoveryRecordPrior(root, &next); err == nil {
		t.Fatal("current authority superseded without epoch delta")
	}
	next.Predecessor.Archive.SHA256 = sum([]byte("another root"))
	if _, err := verifyRecoveryRecordPrior(root, &next); err == nil {
		t.Fatal("wrong supersedes root accepted")
	}
	member := filepath.Join(root, base, a.Members[0].Path)
	if err := os.WriteFile(member, []byte("changed member"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readRecoveryExactAuthority(root, ref); err == nil {
		t.Fatal("member tamper accepted")
	}
	if err := os.WriteFile(member, []byte(a.Members[0].Path), 0644); err != nil {
		t.Fatal(err)
	}
	if err := publishRecoverySealedFile(root, base+"/untracked-extra.json", []byte("extra"), false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readRecoveryExactAuthority(root, ref); err == nil {
		t.Fatal("extra authority member accepted")
	}
}

func TestTransportRecoveryGenerationExecutionOrigins(t *testing.T) {
	a := recoverySchemaFixture()
	authority := recoveryAuthorityRef(&a, mustJSON(a))
	bundle := []byte("synthetic current input")
	m := GenerationBundleManifest{Locale: a.Locale, BatchID: "chatgpt-fr-FR-002", TaskKind: TransportRecoveryTaskKind, Attempt: 1, InputIdentitySHA256: sum([]byte("new input")), ExpectedOutputs: []GenerationBundleExpectedOutput{{BundlePath: "outputs/one.article", UnitID: "one"}, {BundlePath: "outputs/two.article", UnitID: "two"}}}
	outputs := map[string][]byte{"outputs/one.article": []byte("one\n"), "outputs/two.article": []byte("two\n")}
	material := map[string][]byte{"engineering/new-input.zip": bundle, "engineering/one.article": outputs["outputs/one.article"], "engineering/two.article": outputs["outputs/two.article"]}
	call := TransportRecoveryModelInvocation{Session: "synthetic new model session", Invocation: "synthetic call 1", BundleSHA256: sum(bundle), InputIdentitySHA256: m.InputIdentitySHA256, TaskKind: m.TaskKind, Attempt: 1, Provider: a.Predecessor.Provider, Model: a.Predecessor.Model, InputEvidence: recoveryMemberReference("engineering/new-input.zip", bundle)}
	for _, name := range []string{"one", "two"} {
		r := recoveryMemberReference("engineering/"+name+".article", outputs["outputs/"+name+".article"])
		call.Outputs = append(call.Outputs, TransportRecoveryExecutionOutput{NewTransportBundleFile("outputs/"+name+".article", "", outputs["outputs/"+name+".article"]), r})
		call.Evidence = append(call.Evidence, r)
	}
	e := TransportRecoveryGenerationExecution{Schema: TransportRecoveryExecutionSchema, Kind: "model-generation", Locale: m.Locale, ContentPackage: a.ContentPackage.ID, BatchID: m.BatchID, TaskKind: m.TaskKind, Attempt: 1, Authority: authority, BundleSHA256: sum(bundle), InputIdentitySHA256: m.InputIdentitySHA256, Provider: call.Provider, Model: call.Model, Invocations: []TransportRecoveryModelInvocation{call}}
	verify := func(x TransportRecoveryGenerationExecution) error {
		x.Identity = recoveryExecutionIdentity(x)
		_, err := validateRecoveryExecution("", &x, &a, authority, m, bundle, call.Provider, call.Model, outputs, func(r RecoveryFileReference) ([]byte, error) { return material[r.Path], nil })
		return err
	}
	if err := verify(e); err != nil {
		t.Fatal("new execution", err)
	}
	for _, tc := range []struct {
		name   string
		change func(*TransportRecoveryGenerationExecution)
	}{
		{"old-invocation", func(e *TransportRecoveryGenerationExecution) { e.BundleSHA256 = sum([]byte("old Bundle")) }},
		{"missing-evidence", func(e *TransportRecoveryGenerationExecution) { e.Invocations[0].Evidence = nil }},
		{"missing-origin", func(e *TransportRecoveryGenerationExecution) { e.Invocations[0].Outputs = e.Invocations[0].Outputs[:1] }},
		{"duplicate-origin", func(e *TransportRecoveryGenerationExecution) {
			e.Invocations[0].Outputs = append(e.Invocations[0].Outputs, e.Invocations[0].Outputs[0])
		}},
		{"provider-model", func(e *TransportRecoveryGenerationExecution) { e.Invocations[0].Model = "not-the-selected-model" }},
		{"old-raw-evidence", func(e *TransportRecoveryGenerationExecution) {
			e.Invocations[0].Outputs[0].Evidence.Path = "data/retranslation-runs/fr-FR/chatgpt-fr-FR-001/raw-responses/one.article"
		}},
		{"self-declaration", func(e *TransportRecoveryGenerationExecution) {
			e.Invocations[0].InputEvidence = recoveryMemberReference("engineering/statement.txt", []byte("I regenerated"))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var x TransportRecoveryGenerationExecution
			if err := json.Unmarshal(mustJSON(e), &x); err != nil {
				t.Fatal(err)
			}
			tc.change(&x)
			if err := verify(x); err == nil {
				t.Fatal("invalid execution origin accepted")
			}
		})
	}
	var split TransportRecoveryGenerationExecution
	if err := json.Unmarshal(mustJSON(e), &split); err != nil {
		t.Fatal(err)
	}
	second := split.Invocations[0]
	second.Invocation = "synthetic actual missing-output call 2"
	second.Outputs = second.Outputs[1:]
	split.Invocations[0].Outputs = split.Invocations[0].Outputs[:1]
	split.Invocations = append(split.Invocations, second)
	if err := verify(split); err != nil {
		t.Fatal("interrupted attempt with two actual new calls", err)
	}
}

func TestTransportRecoveryExecutionOriginCannotBeRebound(t *testing.T) {
	root := t.TempDir()
	a := recoverySchemaFixture()
	old := TransportRecoveryGenerationExecution{Schema: TransportRecoveryExecutionSchema, Kind: "model-generation", Locale: a.Locale, BatchID: a.Predecessor.BatchID, Attempt: 1, BundleSHA256: a.Predecessor.Archive.SHA256, Invocations: []TransportRecoveryModelInvocation{{Session: "actual synthetic old session", Invocation: "actual synthetic old call"}}}
	old.Identity = recoveryExecutionIdentity(old)
	b := mustJSON(old)
	ref := recoveryMemberReference("engineering/prior-execution.json", b)
	if err := publishRecoverySealedFile(root, ref.Path, b, false); err != nil {
		t.Fatal(err)
	}
	r := GenerationResultBundleManifest{Locale: a.Locale, BatchID: a.Predecessor.BatchID, Attempt: 1, GenerationBundleSHA256: a.Predecessor.Archive.SHA256, RecoveryExecution: &RecoveryExecutionReceipt{Identity: old.Identity, Object: ref}}
	if err := publishRecoverySealedFile(root, a.Predecessor.ImportReceipt.Path, mustJSON(r), false); err != nil {
		t.Fatal(err)
	}
	current := old
	current.BatchID = "chatgpt-fr-FR-002"
	current.BundleSHA256 = sum([]byte("new current Bundle"))
	current.Identity = recoveryExecutionIdentity(current)
	if err := rejectRecoveryExecutionOriginReplay(root, &current, &a); err == nil {
		t.Fatal("old actual call rebound by changing new Bundle identity")
	}
	current.Invocations = append([]TransportRecoveryModelInvocation(nil), old.Invocations...)
	current.Invocations[0].Invocation = "actual synthetic new call"
	current.Identity = recoveryExecutionIdentity(current)
	if err := rejectRecoveryExecutionOriginReplay(root, &current, &a); err != nil {
		t.Fatal("new actual call rejected", err)
	}
}

func TestTransportRecoveryRejectsOldRawDirectoryAndInode(t *testing.T) {
	root := t.TempDir()
	a := recoverySchemaFixture()
	old := a.Predecessor.FinalUnits[0].Raw.Path
	if err := publishRecoverySealedFile(root, old, []byte("same possible language bytes\n"), false); err != nil {
		t.Fatal(err)
	}
	m := GenerationBundleManifest{ExpectedOutputs: []GenerationBundleExpectedOutput{{BundlePath: "outputs/response.article"}}}
	if err := rejectHistoricalRecoveryOutputDirectory(root, filepath.Join(root, filepath.Dir(old)), &a, m); err == nil {
		t.Fatal("old formal raw directory accepted as new output")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "response.article")
	if err := os.Link(filepath.Join(root, old), path); err != nil {
		t.Fatal(err)
	}
	if err := rejectHistoricalRecoveryOutputDirectory(root, dir, &a, m); err == nil {
		t.Fatal("hardlinked old raw accepted as new model response")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("same possible language bytes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := rejectHistoricalRecoveryOutputDirectory(root, dir, &a, m); err != nil {
		t.Fatal("fresh evidence-backed call may legitimately return identical language bytes", err)
	}
}

func TestTransportRecoveryChildBindsCompleteAuthorityReference(t *testing.T) {
	a := recoverySchemaFixture()
	ref := recoveryAuthorityRef(&a, mustJSON(a))
	r := a.LanguageAuthority.Scope[0]
	r.ReexportReason = transportRecoveryReason
	r.PreviousBatchID = a.Predecessor.BatchID
	r.PreviousInputPath = r.InputPath
	r.PreviousInputSHA256 = r.InputSHA256
	r.PreviousProtectedTokens = r.ProtectedTokenCount
	m := RetranslationBatchManifest{TransportRecovery: &ref, Locale: a.Locale, BatchID: "chatgpt-fr-FR-002", UnitCount: 1, Units: []RetranslationBatchUnit{r}}
	if err := validateRecoveryChildManifest(&a, &m); err != nil {
		t.Fatal("canonical complete reference", err)
	}
	for _, change := range []func(*TransportRecoveryReference){
		func(r *TransportRecoveryReference) { r.SHA256 = sum([]byte("wrong authority bytes")) },
		func(r *TransportRecoveryReference) { r.Size++ },
		func(r *TransportRecoveryReference) { r.Path = "data/elsewhere/authority.json" },
		func(r *TransportRecoveryReference) { r.Identity = sum([]byte("wrong identity")) },
	} {
		bad := ref
		change(&bad)
		m.TransportRecovery = &bad
		if err := validateRecoveryChildManifest(&a, &m); err == nil {
			t.Fatal("child partial authority match accepted", bad)
		}
	}
}
