package sitecontent

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSameUnitBridgeSourceComposition(t *testing.T) {
	for _, code := range []string{
		"```go\npackage p\n// Explain the return value.\n//go:generate echo keep\nfunc f() string { return \"literal\" }\n```\n",
		"<pre>\n{\n  // The module path.\n  \"path\": \"example.com/module\"\n}\n</pre>\n",
	} {
		t.Run(code[:3], func(t *testing.T) {
			source := []byte("# Heading\n\nKeep ordinary prose with `API` and [link](https://example.test).\n\n" + code)
			d, err := ParseWorkflowDocument(Source{Path: "_content/doc/bridge.md", Kind: "page", Route: "/doc/bridge", SHA256: digest(source)}, source, 1)
			if err != nil {
				t.Fatal(err)
			}
			a := PageAdapter{*d, source}
			raw := []byte(strings.Replace(d.Units[0].Source, "Keep ordinary prose", "保留正文", 1))
			old, err := a.Restore(string(raw))
			if err != nil {
				t.Fatal(err)
			}
			bodies, err := a.CommentRevisionBodies(old)
			if err != nil || len(bodies) != 1 {
				t.Fatal(len(bodies), err)
			}
			body := bodies[0]
			reviewed := append(append(append([]byte{}, old[:body.Start]...), []byte("说明返回内容。")...), old[body.End:]...)
			p := &i18n.CommentRevisionTransport{Schema: "go-learning/comment-revision-transport/v1", UnitID: d.Units[0].ID, SourceSHA256: d.SourceSHA, Slots: []i18n.CommentRevisionTransportSlot{{SlotID: "source-owned-test-coordinate", Start: body.Start, End: body.End}}, Boundaries: []string{"⟪GTI18N_aaaaaaaa_000001⟫", "⟪GTI18N_aaaaaaaa_000002⟫"}}
			restricted := []byte(p.Boundaries[0] + "说明返回内容。" + p.Boundaries[1])
			decoder, bridge, err := a.BridgeRestrictedRevision(raw, old, reviewed, restricted, p)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decoder.Restore(string(bridge))
			if err != nil || !bytes.Equal(bridgeEOF(got), bridgeEOF(reviewed)) {
				t.Fatal(err)
			}
			if !bytes.Contains(bridge, []byte("保留正文")) {
				t.Fatal("ordinary prose hidden")
			}
			revised := bytes.Replace(bridge, []byte("保留正文"), []byte("修订正文"), 1)
			got, err = decoder.Restore(string(revised))
			if err != nil || !bytes.Contains(got, []byte("修订正文")) {
				t.Fatal(err)
			}
			_, tokens, _ := decoder.ProtectedInput(nil)
			for _, bad := range []string{strings.Replace(string(bridge), tokens[0], "", 1), string(bridge) + tokens[0], strings.Replace(string(bridge), tokens[0], "⟦WRONG⟧", 1)} {
				if _, err := decoder.Restore(bad); err == nil {
					t.Fatal("token mutation accepted")
				}
			}
			for _, bad := range [][]byte{bytes.Replace(reviewed, []byte("保留正文"), []byte("越权修改"), 1), bytes.Replace(reviewed, []byte("example.test"), []byte("changed.test"), 1)} {
				if _, _, err := a.BridgeRestrictedRevision(raw, old, bad, restricted, p); err == nil {
					t.Fatal("unbound target change accepted")
				}
			}
			for _, pair := range [][2]string{{"//go:generate echo keep", "//go:generate echo changed"}, {"return \"literal\"", "return \"changed\""}, {"func f()", "func renamed()"}, {"example.com/module", "changed.test/module"}} {
				if !bytes.Contains(reviewed, []byte(pair[0])) {
					continue
				}
				bad := bytes.Replace(reviewed, []byte(pair[0]), []byte(pair[1]), 1)
				if _, _, err := a.BridgeRestrictedRevision(raw, old, bad, restricted, p); err == nil {
					t.Fatal("changed protected identity accepted", pair[0])
				}
			}
			p.Slots = append(p.Slots, p.Slots[0])
			p.Boundaries = append(p.Boundaries, p.Boundaries[1])
			if _, _, err := a.BridgeRestrictedRevision(raw, old, reviewed, restricted, p); err == nil {
				t.Fatal("duplicate comment coordinate accepted")
			}
		})
	}
}

// Shared-lifecycle acceptance fixture, called after the existing comment
// transport fixture has created a reviewed, restricted predecessor in TempDir.
func testSameUnitBridgeLifecycle(t *testing.T, root string, c *i18n.Catalog, snapshot, id string) {
	t.Helper()
	meta, _ := json.Marshal(c.TranslationPackages[0])
	writeFixture(t, root, "engineering/bridge-package.json", meta)
	const finding = "Revise the ordinary prose locally; preserve all protected identities."
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "independent-bridge-fixture-reviewer", Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: "comment-predecessor", UnitIDs: []string{id}, Rating: "B", Finding: finding}); err != nil {
		t.Fatal(err)
	}
	snapshotPath := "data/quality-check-snapshots/zh-CN/" + snapshot + "/manifest.json"
	snap, err := os.ReadFile(filepath.Join(root, snapshotPath))
	if err != nil {
		t.Fatal(err)
	}
	evidence := []byte("Independent fixture Reviewer: " + id + "\nB: " + finding + "\n")
	writeFixture(t, root, "engineering/bridge-reviewer.md", evidence)
	o := i18n.SameUnitBridgeOptions{Locale: "zh-CN", ContentPackage: c.TranslationPackages[0].ID, BridgeID: "same-unit-qc-bridge", PreviousSnapshotID: snapshot, SnapshotSHA256: digest(snap), ReviewerEvidence: i18n.GlossaryArchiveReference{Path: "engineering/bridge-reviewer.md", SHA256: digest(evidence)}, UnitIDs: []string{id}}
	noBridge, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: o.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: o.UnitIDs, AllowReexport: true, PreviousSnapshotID: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: o.Locale, BatchID: noBridge.BatchID}); !errors.Is(err, i18n.ErrRevisionTemplateIncapable) {
		t.Fatal("restricted predecessor accepted without bridge", err)
	}
	plan, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o)
	if err != nil {
		t.Fatal("bridge preflight", err)
	}
	if plan.Recorded || plan.Mutation != "none" || len(plan.Receipt.Units) != 1 || plan.Receipt.Units[0].UnitID != id || plan.Receipt.Units[0].PreviousRating != "B" || plan.Receipt.Units[0].Finding != finding {
		t.Fatal("bridge changed formal/QC identity")
	}
	if _, err := os.Stat(filepath.Join(root, plan.ReceiptPath)); !os.IsNotExist(err) {
		t.Fatal("preflight wrote authority")
	}
	again, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o)
	if err != nil || again.ReceiptSHA256 != plan.ReceiptSHA256 {
		t.Fatal("nondeterministic bridge", err)
	}
	// The epoch changes validation only. Its PageAdapter factory must give
	// the same receipt with the real bound transport's public data codec.
	epoch := c.Adapted[0].Adapter.(transportEpochPage)
	if !epoch.enabled {
		t.Fatal("current fixture validator must be enabled")
	}
	decorated := *c
	decorated.Adapted = append([]i18n.AdaptedTranslationUnit(nil), c.Adapted...)
	decorated.TranslationPackages = append([]i18n.TranslationPackageDescriptor(nil), c.TranslationPackages...)
	decorated.TranslationPackages[0].Units = append([]i18n.AdaptedTranslationUnit(nil), c.TranslationPackages[0].Units...)
	decorated.Adapted[0].Adapter = observedBridgePage{PageAdapter: epoch.PageAdapter, original: epoch}
	decorated.TranslationPackages[0].Units[0].Adapter = decorated.Adapted[0].Adapter
	sharedPlan, err := i18n.PrepareSameUnitRepresentationBridge(root, &decorated, o)
	if err != nil || sharedPlan.ReceiptSHA256 != plan.ReceiptSHA256 {
		t.Fatal("comment shared bridge changed receipt", err)
	}
	wrong := o
	wrong.SnapshotSHA256 = strings.Repeat("0", 64)
	if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, wrong); err == nil {
		t.Fatal("wrong Snapshot accepted")
	}
	wrong = o
	wrong.ContentPackage = "wrong-package"
	if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, wrong); err == nil {
		t.Fatal("wrong package accepted")
	}
	wrong = o
	wrong.UnitIDs = append(wrong.UnitIDs, id)
	if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, wrong); err == nil {
		t.Fatal("duplicate scope accepted")
	}
	t.Run("concurrent-mutation", func(t *testing.T) {
		original := c.Adapted[0]
		wrapped := original
		wrapped.Adapter = concurrentBridgePage{transportEpochPage: original.Adapter.(transportEpochPage), mutate: func() { writeFixture(t, root, o.ReviewerEvidence.Path, append(bytes.Clone(evidence), ' ')) }}
		c.Adapted[0], c.TranslationPackages[0].Units[0] = wrapped, wrapped
		defer func() {
			c.Adapted[0], c.TranslationPackages[0].Units[0] = original, original
			writeFixture(t, root, o.ReviewerEvidence.Path, evidence)
		}()
		if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o); err == nil || !strings.Contains(err.Error(), "bridge dependency drift") {
			t.Fatal("concurrent change was not caught at fresh boundary", err)
		}
	})
	o.Record = true
	recorded, err := i18n.PrepareSameUnitRepresentationBridge(root, &decorated, o)
	if err != nil || !recorded.Recorded {
		t.Fatal("TempDir record", err)
	}
	if recorded.ReceiptSHA256 != plan.ReceiptSHA256 {
		t.Fatal("second fresh comment bridge receipt changed")
	}
	AttachPageReplacementInvocation(root, c)
	if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o); err == nil {
		t.Fatal("duplicate receipt overwritten")
	}
	t.Run("warm-reconstruction-dependency-matrix", func(t *testing.T) {
		// Deliberately return the same reconstruction to test the i18n reader's
		// fresh disk boundary independently of the owned Page projection memo.
		oldResolver := c.BridgeResolver
		defer func() { c.BridgeResolver = oldResolver }()
		var result *i18n.SameUnitBridgeResolution
		builds := 0
		c.BridgeResolver = func(_ string, _ *i18n.Catalog, _ string, _ i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
			if result == nil {
				var err error
				builds++
				result, err = build()
				if err != nil {
					return nil, err
				}
			}
			return result, nil
		}
		if _, err := i18n.CheckQualityCheckSurfaceReopen(root, c, o.Locale, o.BridgeID); err != nil {
			t.Fatal("warm baseline", err)
		}
		// Every frozen dependency is read fresh even with an invocation-local memo.
		for _, ref := range recorded.Receipt.SameUnitBridge.Dependencies {
			t.Run("drift-"+strings.ReplaceAll(ref.Path, "/", "_"), func(t *testing.T) {
				b, err := os.ReadFile(filepath.Join(root, ref.Path))
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, ref.Path, append(bytes.Clone(b), ' '))
				defer writeFixture(t, root, ref.Path, b)
				if _, err := i18n.CheckQualityCheckSurfaceReopen(root, c, o.Locale, o.BridgeID); err == nil {
					t.Fatal("warm authority accepted dependency drift", ref.Path)
				}
			})
		}

		if _, err := i18n.CheckQualityCheckSurfaceReopen(root, c, o.Locale, o.BridgeID); err != nil {
			t.Fatal("restored warm baseline", err)
		}
		if builds != 1 {
			t.Fatal("warm matrix rebuilt graph", builds)
		}
		// A lookup-time mutation must also fail at the after-read boundary,
		// including when the resolver returns an already reconstructed object.
		baseline, err := os.ReadFile(filepath.Join(root, o.ReviewerEvidence.Path))
		if err != nil {
			t.Fatal(err)
		}
		c.BridgeResolver = func(_ string, _ *i18n.Catalog, _ string, _ i18n.GlossaryArchiveReference, _ func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
			writeFixture(t, root, o.ReviewerEvidence.Path, append(bytes.Clone(baseline), ' '))
			return result, nil
		}
		_, err = i18n.CheckQualityCheckSurfaceReopen(root, c, o.Locale, o.BridgeID)
		writeFixture(t, root, o.ReviewerEvidence.Path, baseline)
		if err == nil {
			t.Fatal("warm lookup concurrent mutation accepted")
		}
	})
	t.Run("bridge-receipt-drift", func(t *testing.T) {
		path := filepath.Join(root, recorded.ReceiptPath)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		defer writeFixture(t, root, recorded.ReceiptPath, b)
		changed := *recorded.Receipt
		authority := *changed.SameUnitBridge
		authority.Bindings = append([]i18n.SameUnitBridgeBinding(nil), authority.Bindings...)
		authority.Bindings[0].Template += "unbound prose"
		changed.SameUnitBridge = &authority
		v, _ := json.Marshal(changed)
		writeFixture(t, root, recorded.ReceiptPath, append(v, '\n'))
		if _, err := i18n.CheckQualityCheckSurfaceReopen(root, c, o.Locale, o.BridgeID); err == nil {
			t.Fatal("tampered bridge accepted")
		}
	})
	exp, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: o.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: o.UnitIDs, AllowReexport: true, PreviousSnapshotID: snapshot, RepresentationBridgeID: o.BridgeID})
	if err != nil {
		t.Fatal("bridge-backed ordinary export", err)
	}
	bundle, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: o.Locale, BatchID: exp.BatchID})
	if err != nil {
		t.Fatal("bridge Bundle", err)
	}
	if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err != nil {
		t.Fatal("bridge current-check", err)
	}
	files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(m.ExpectedOutputs[0].BundlePath)
	baseline := files["formal/protected-candidates/"+name]
	if !bytes.Contains(baseline, []byte("Keep unrelated prose")) || bytes.Contains(baseline, []byte("const message")) {
		t.Fatal("normal prose/code capability wrong")
	}
	var manifest i18n.RetranslationBatchManifest
	if err := json.Unmarshal(files["formal/manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Units[0].RevisionFeedbackSource != "quality_check" || manifest.Units[0].PreviousFinding != finding || manifest.Units[0].PreviousRating != "B" || manifest.Units[0].RepresentationBridge == nil {
		t.Fatal("QC feedback or bridge binding lost")
	}
	// Invalid output is tested through the real importer boundary; a legal
	// synthetic local prose edit then uses the normal import/process lifecycle.
	dir := t.TempDir()
	write := func(b []byte) { writeFixture(t, dir, name, b) }
	for _, bad := range [][]byte{bytes.Replace(baseline, []byte("Keep unrelated prose"), []byte("Changed `newIdentity` prose"), 1), append(bytes.Clone(baseline), []byte("new outside prose\n")...)} {
		write(bad)
		if _, _, err := i18n.PackGenerationResultBundle(root, c, bundle, "chatgpt", i18n.FormalChatGPTGenerationModel, dir); err == nil {
			t.Fatal("unauthorized syntax accepted")
		}
	}
	// A restore failure preserves the bridge for a single-Unit retry. Result-pack
	// correctly rejects this response; install only in this isolated failure fixture.
	writeFixture(t, root, filepath.Join(exp.BatchPath, "raw-responses", name), []byte("broken response\n"))
	processed, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: o.Locale, BatchID: exp.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	if processed.RestoreFailed != 1 {
		t.Fatal("expected synthetic failure", processed)
	}
	retry, rm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: o.Locale, BatchID: exp.BatchID, UnitID: id})
	if err != nil {
		t.Fatal("bridge retry Bundle", err)
	}
	retryFiles, err := i18n.ReadTransportBundle(retry, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	if rm.Attempt != 2 || !bytes.Equal(retryFiles["formal/protected-candidates/"+name], baseline) || !bytes.Equal(retryFiles["formal/representation-bridge/"+id+".json"], files["formal/representation-bridge/"+id+".json"]) {
		t.Fatal("retry changed bridge authority")
	}
	revised := bytes.Replace(baseline, []byte("Keep unrelated prose"), []byte("Locally revised ordinary prose"), 1)
	write(revised)
	if _, err := i18n.ImportGenerationOutputDirectory(root, c, retry, "chatgpt", i18n.FormalChatGPTGenerationModel, dir); err != nil {
		t.Fatal("bridge retry import", err)
	}
	processed, err = i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: o.Locale, BatchID: exp.BatchID, UnitID: id})
	if err != nil || processed.ValidationPassed != 1 {
		t.Fatal("bridge retry/process", processed, err)
	}
	candidate, err := os.ReadFile(filepath.Join(root, exp.BatchPath, processed.Units[0].CandidatePath))
	if err != nil || !bytes.Contains(candidate, []byte("Locally revised ordinary prose")) || !bytes.Contains(candidate, []byte("将name与message关联")) {
		t.Fatal("bridge retry lost reviewed comments/prose", err)
	}
	if err := i18n.ValidateTranslationUnitCandidate(root, c, id, o.Locale, candidate); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "bridge-revised-successor"}); err != nil {
		t.Fatal("bridge downstream evidence reader", err)
	}
	t.Run("successor-ordinary-continuity", func(t *testing.T) { testBridgeSuccessorContinuity(t, root, c, "bridge-revised-successor", id) })
	// Historical A consumes the frozen bridge; it must not become a replay of
	// old mutable command docs. New generation authorization remains current.
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "independent-bridge-fixture-reviewer", Locale: o.Locale, SnapshotID: "bridge-revised-successor", PreviousSnapshotID: snapshot, UnitIDs: []string{id}, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: o.Locale, SnapshotID: "bridge-revised-successor"}); err != nil {
		t.Fatal("bridge finalized history", err)
	}
	before, err := i18n.VerifyHistoricalQualityCheckFinalization(root, o.Locale, "bridge-revised-successor", c)
	if err != nil {
		t.Fatal("bridge historical integrity", err)
	}
	doc := "docs/TRANSLATION_TASK_SPEC.md"
	oldDoc, err := os.ReadFile(filepath.Join(root, doc))
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, doc, append(bytes.Clone(oldDoc), []byte("\nGeneric command documentation evolution.\n")...))
	defer writeFixture(t, root, doc, oldDoc)
	if _, err := i18n.CheckQualityCheckSurfaceReopen(root, c, o.Locale, o.BridgeID); err == nil {
		t.Fatal("new bridge authorization accepted docs drift")
	}
	after, err := i18n.VerifyHistoricalQualityCheckFinalization(root, o.Locale, "bridge-revised-successor", c)
	if err != nil {
		t.Fatal("historical A replayed mutable docs", err)
	}
	beforeBytes, _ := json.Marshal(before)
	afterBytes, _ := json.Marshal(after)
	if !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatal("historical A changed after current policy evolution")
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: o.Locale, SnapshotID: "bridge-current-policy-successor"}); err != nil {
		t.Fatal("passed bridge candidate current validation", err)
	}
	for _, path := range []string{recorded.ReceiptPath, recorded.Receipt.SameUnitBridge.Bindings[0].Raw.Path} {
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path, append(bytes.Clone(b), ' '))
		_, err = i18n.VerifyHistoricalQualityCheckFinalization(root, o.Locale, "bridge-revised-successor", c)
		writeFixture(t, root, path, b)
		if err == nil {
			t.Fatal("historical bridge accepted immutable drift", path)
		}
	}
}

type concurrentBridgePage struct {
	transportEpochPage
	mutate func()
}

func (a concurrentBridgePage) BridgeRestrictedRevision(ordinary, old, reviewed, restricted []byte, p *i18n.CommentRevisionTransport) (i18n.TranslationUnitAdapter, []byte, error) {
	a.mutate()
	return a.PageAdapter.BridgeRestrictedRevision(ordinary, old, reviewed, restricted, p)
}

// The synthetic registered package supplies its own explicit metadata authority;
// it does not claim to be a member of the repository's frozen Site registry.
func (a transportEpochPage) BridgeContextReferences(root string) ([]i18n.GlossaryArchiveReference, error) {
	ref, err := ReferenceFile(root, "engineering/bridge-package.json")
	if err != nil {
		return nil, err
	}
	return []i18n.GlossaryArchiveReference{{Path: ref.Path, SHA256: ref.SHA256}}, nil
}
