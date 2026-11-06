package sitecontent

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type surfaceBridgeContextPage struct{ PageAdapter }

func (a surfaceBridgeContextPage) BridgeContextReferences(root string) ([]i18n.GlossaryArchiveReference, error) {
	r, err := ReferenceFile(root, "engineering/surface-bridge-package.json")
	return []i18n.GlossaryArchiveReference{{Path: r.Path, SHA256: r.SHA256}}, err
}

func TestSurfaceBridgeSourceComposition(t *testing.T) {
	source := []byte("# Source-owned bridge\n\nRead the `sql.OpenDB function` with [link](https://example.test).\n\nKeep unrelated prose.\n\n```go\n//go:generate echo keep\nconst value = \"opaque\"\n```\n")
	d, err := ParseWorkflowDocument(Source{Path: "_content/doc/example.md", Kind: "page", Route: "/doc/example", SHA256: digest(source)}, source, 1)
	if err != nil {
		t.Fatal(err)
	}
	a := PageAdapter{*d, source}
	text, _, err := a.ProtectedInput(nil)
	if err != nil {
		t.Fatal(err)
	}
	discovery, _, err := a.SurfaceTransportProjection(a, []byte(text), nil)
	if err != nil {
		t.Fatal(err)
	}
	var region string
	for _, r := range discovery.Available {
		if r.Kind == "inline-api-descriptor" {
			region = r.ID
		}
	}
	projection, adapter, err := a.SurfaceTransportProjection(a, []byte(text), []string{region})
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(strings.Replace(projection.Template, "function", "procedure", 1))
	reviewed, err := adapter.Restore(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(adapter)
	decoder, template, ancestor, err := a.BridgeSurfaceRevision(adapter, raw, reviewed)
	if err != nil {
		t.Fatal(err)
	}
	old, oldErr := a.Restore(string(ancestor))
	oldCandidate, baselineErr := adapter.Restore(projection.Template)
	if len(ancestor) == 0 || oldErr != nil || baselineErr != nil || !bytes.Equal(bridgeEOF(old), bridgeEOF(oldCandidate)) {
		t.Fatal("ordinary ancestor did not preserve the old source mapping", oldErr, baselineErr)
	}
	got, err := decoder.Restore(string(template))
	if err != nil || !bytes.Equal(bridgeEOF(got), bridgeEOF(reviewed)) {
		t.Fatal("ordinary exact restore", err)
	}
	if !bytes.Contains(template, []byte("Keep unrelated prose.")) || bytes.Contains(template, []byte("procedure")) || bytes.Contains(template, []byte("sql.OpenDB")) {
		t.Fatal("ordinary prose/opaque descriptor scope")
	}
	if _, err := adapter.Restore(string(template)); err == nil {
		t.Fatal("ordinary template was accepted by old Surface editor")
	}
	if _, err := decoder.Restore(strings.Replace(string(template), "⟦P", "⟦wrong", 1)); err == nil {
		t.Fatal("protected identity change accepted")
	}
	for _, name := range []string{"wrong-unit", "wrong-source", "wrong-path", "changed-origin", "ordinary-already", "missing-token", "candidate-delta", "unknown-decoder"} {
		t.Run(name, func(t *testing.T) {
			var p surfacePageTransport
			if err := json.Unmarshal(before, &p); err != nil {
				t.Fatal(err)
			}
			var candidate = i18n.TranslationUnitAdapter(&p)
			r := bytes.Clone(raw)
			target := bytes.Clone(reviewed)
			switch name {
			case "wrong-unit":
				p.UnitID = "other"
			case "wrong-source":
				p.CanonicalSourceSHA256 = strings.Repeat("0", 64)
			case "wrong-path":
				p.Origin.Document.Path = "other.md"
			case "changed-origin":
				p.Origin.Source = append(p.Origin.Source, 'x')
			case "ordinary-already":
				p.Ordinary = true
			case "missing-token":
				r = []byte("broken")
			case "candidate-delta":
				target = append(target, 'x')
			case "unknown-decoder":
				candidate = a
			}
			if _, _, _, err := a.BridgeSurfaceRevision(candidate, r, target); err == nil {
				t.Fatal("invalid composition accepted", name)
			}
		})
	}
	after, _ := json.Marshal(adapter)
	if !bytes.Equal(before, after) || !bytes.Equal(source, a.Source) || a.Document.Units[0].ID != d.Units[0].ID {
		t.Fatal("Surface codec/source/formal Unit mutated")
	}
}

func TestSurfaceToOrdinaryQCBridgeContract(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	finish := contentidentity.BeginMemoization()
	defer finish()
	root, c, batch := surfaceCapabilityFixture(t)
	metadata, _ := json.Marshal(c.TranslationPackages[0])
	writeFixture(t, root, "engineering/surface-bridge-package.json", metadata)
	adapted := c.Adapted[0]
	adapted.Adapter = surfaceBridgeContextPage{adapted.Adapter.(PageAdapter)}
	c.Adapted[0], c.TranslationPackages[0].Units[0] = adapted, adapted
	id := adapted.Unit.ID
	discovery, err := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", batch, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	request := surfaceCapabilityRequest(t, discovery, true)
	transport, err := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", batch, &request, true)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil {
		t.Fatal(err)
	}
	raw := bridgeEOF([]byte(strings.Replace(transport.Authority.Bindings[0].Projection.Template, "function", "函数", 1)))
	surfaceImport(t, root, c, bundle, map[string][]byte{id: raw})
	processed, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil || processed.ValidationPassed != 1 {
		t.Fatal("Surface predecessor", processed, err)
	}
	const snapshot = "surface-independent-qc"
	s, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	const finding = "Revise the unrelated prose locally; preserve all protected code and API identities."
	if digest([]byte(finding)) == transport.Authority.Request.Units[0].FindingSHA256 {
		t.Fatal("fixture findings must differ")
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{ReviewerSession: "independent-later-qc-reviewer", Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: requestSnapshot(t, root, batch), UnitIDs: []string{id}, Rating: "B", Finding: finding}); err != nil {
		t.Fatal(err)
	}
	evidence := []byte("Independent later QC: " + id + "\nB: " + finding + "\n")
	writeFixture(t, root, "engineering/later-review.md", evidence)
	snapshotPath := "data/quality-check-snapshots/zh-CN/" + snapshot + "/manifest.json"
	snapshotBytes, _ := readRegular(root, snapshotPath)
	o := i18n.SameUnitBridgeOptions{Locale: "zh-CN", ContentPackage: c.TranslationPackages[0].ID, BridgeID: "surface-independent-qc-bridge", PreviousSnapshotID: snapshot, SnapshotSHA256: digest(snapshotBytes), ReviewerEvidence: i18n.GlossaryArchiveReference{Path: "engineering/later-review.md", SHA256: digest(evidence)}, UnitIDs: []string{id}}
	noBridge, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{id}, PreviousSnapshotID: snapshot, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	failedManifest, _ := readRegular(root, noBridge.BatchPath+"/manifest.json")
	if b, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: noBridge.BatchID}); !errors.Is(err, i18n.ErrRevisionTemplateIncapable) || len(b) != 0 {
		t.Fatal("Surface bounded editor became generic QC authority", err)
	}
	observe := func() map[string]string {
		t.Helper()
		paths := map[string]string{}
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				paths[path] = digest(b)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return paths
	}
	beforePreflight := observe()
	plan, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o)
	if err != nil {
		t.Fatal("Surface -> independent QC readonly preflight", err)
	}
	// Explicitly opt this fixture's context-only adapter into the ordinary
	// factory decorator. Unknown production adapters remain on their original
	// path; this fixture delegates all representation semantics to PageAdapter.
	decorated := *c
	decorated.Adapted = append([]i18n.AdaptedTranslationUnit(nil), c.Adapted...)
	decorated.TranslationPackages = append([]i18n.TranslationPackageDescriptor(nil), c.TranslationPackages...)
	decorated.TranslationPackages[0].Units = append([]i18n.AdaptedTranslationUnit(nil), c.TranslationPackages[0].Units...)
	page := adapted.Adapter.(surfaceBridgeContextPage)
	decorated.Adapted[0].Adapter = observedBridgePage{PageAdapter: page.PageAdapter, original: page}
	decorated.TranslationPackages[0].Units[0].Adapter = decorated.Adapted[0].Adapter
	sharedPlan, err := i18n.PrepareSameUnitRepresentationBridge(root, &decorated, o)
	if err != nil || sharedPlan.ReceiptSHA256 != plan.ReceiptSHA256 || !reflect.DeepEqual(sharedPlan.Receipt, plan.Receipt) {
		t.Fatal("shared bridge construction changed canonical receipt", err)
	}
	if plan.Recorded || plan.Mutation != "none" || plan.Receipt.Units[0].PreviousRating != "B" || plan.Receipt.Units[0].Finding != finding {
		t.Fatal("Surface finding replaced independent QC authority")
	}
	if !reflect.DeepEqual(beforePreflight, observe()) {
		t.Fatal("preflight mutated fixture path set or bytes")
	}
	if _, err := os.Stat(filepath.Join(root, plan.ReceiptPath)); !os.IsNotExist(err) {
		t.Fatal("preflight published receipt")
	}
	binding := plan.Receipt.SameUnitBridge.Bindings[0]
	var manifest i18n.RetranslationBatchManifest
	mb, _ := readRegular(root, "data/retranslation-runs/zh-CN/"+batch+"/manifest.json")
	if json.Unmarshal(mb, &manifest) != nil || binding.FormalInputSHA256 == "" || binding.FormalInputSHA256 != manifest.Units[0].InputSHA256 || binding.TransportMember != "formal/surface-transport-authority/"+id+".json" {
		t.Fatal("preflight temporary formal input was not bound")
	}
	paths := map[string]string{
		"source": adapted.Unit.SourcePath, "input": binding.ProtectedInput.Path,
		"candidate": s.Units[0].CandidatePath, "validation": s.Units[0].ValidationPath,
		"raw": binding.Raw.Path, "archive": binding.Archive.Path, "import": binding.Import.Path,
		"ordinary-ancestor":         transport.Authority.Bindings[0].Raw.Path,
		"Surface-transport-mapping": transport.Path, "Surface-reopen": transport.Authority.Reopen.Path,
		"QC-results": "data/quality-check-snapshots/zh-CN/" + snapshot + "/quality-check-results.json",
	}
	for name, path := range paths {
		t.Run("drift-"+name, func(t *testing.T) {
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, before)
			changed := append(bytes.Clone(before), ' ')
			if name == "import" {
				var r i18n.GenerationResultBundleManifest
				if err := json.Unmarshal(before, &r); err != nil {
					t.Fatal(err)
				}
				r.Attempt++
				changed, _ = json.Marshal(r)
			} else if name == "QC-results" {
				var r i18n.QualityCheckResults
				if err := json.Unmarshal(before, &r); err != nil {
					t.Fatal(err)
				}
				r.Results[0].Rating = "A"
				changed, _ = json.Marshal(r)
			}
			writeFixture(t, root, path, changed)
			if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o); err == nil {
				t.Fatal("drift accepted", name)
			}
		})
	}
	for _, field := range []string{"unit", "kind", "source-path", "source-sha", "input-path", "input-sha", "unrelated-Surface"} {
		t.Run("manifest-"+field, func(t *testing.T) {
			path := "data/retranslation-runs/zh-CN/" + batch + "/manifest.json"
			defer writeFixture(t, root, path, mb)
			var wrong i18n.RetranslationBatchManifest
			json.Unmarshal(mb, &wrong)
			r := &wrong.Units[0]
			switch field {
			case "unit":
				r.UnitID = "different-unit"
			case "kind":
				r.UnitKind = i18n.UnitKindExample
			case "source-path":
				r.SourcePath = "_content/other.md"
			case "source-sha":
				r.SourceSHA256 = strings.Repeat("0", 64)
			case "input-sha":
				r.InputSHA256 = strings.Repeat("0", 64)
			case "input-path":
				r.InputPath = "inputs/another.article"
			case "unrelated-Surface":
				r.RevisionAuthorizationID = "another-Surface-reopen"
			}
			b, _ := json.Marshal(wrong)
			writeFixture(t, root, path, b)
			if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o); err == nil {
				t.Fatal("wrong manifest accepted", field)
			}
		})
	}
	for _, field := range []string{"candidate-sha", "attempt", "selected-batch"} {
		t.Run("snapshot-"+field, func(t *testing.T) {
			defer writeFixture(t, root, snapshotPath, snapshotBytes)
			var wrong i18n.QualityCheckSnapshotManifest
			json.Unmarshal(snapshotBytes, &wrong)
			switch field {
			case "candidate-sha":
				wrong.Units[0].CandidateSHA256 = strings.Repeat("0", 64)
			case "attempt":
				wrong.Units[0].Attempt++
			case "selected-batch":
				wrong.Units[0].SelectedBatchID = "wrong-selected-batch"
			}
			b, _ := json.Marshal(wrong)
			writeFixture(t, root, snapshotPath, b)
			bad := o
			bad.SnapshotSHA256 = digest(b)
			if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, bad); err == nil {
				t.Fatal("wrong Snapshot accepted", field)
			}
		})
	}
	o.Record = true
	recorded, err := i18n.PrepareSameUnitRepresentationBridge(root, &decorated, o)
	if err != nil || !recorded.Recorded {
		t.Fatal("fixture bridge record", err)
	}
	if recorded.ReceiptSHA256 != plan.ReceiptSHA256 {
		t.Fatal("second fresh shared construction changed receipt identity")
	}
	AttachPageReplacementInvocation(root, c)
	if _, err := i18n.PrepareSameUnitRepresentationBridge(root, c, o); err == nil {
		t.Fatal("duplicate bridge record accepted")
	}
	for _, path := range []string{recorded.ReceiptPath, transport.Path, binding.Import.Path, paths["QC-results"]} {
		t.Run("recorded-drift-"+filepath.Base(path), func(t *testing.T) {
			before, _ := readRegular(root, path)
			defer writeFixture(t, root, path, before)
			writeFixture(t, root, path, append(bytes.Clone(before), ' '))
			if _, err := i18n.CheckQualityCheckSurfaceReopen(root, c, "zh-CN", o.BridgeID); err == nil {
				t.Fatal("recorded proof drift accepted")
			}
		})
	}
	exp, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{id}, PreviousSnapshotID: snapshot, AllowReexport: true, RepresentationBridgeID: o.BridgeID})
	if err != nil {
		t.Fatal("new ordinary QC batch", err)
	}
	next, gm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: exp.BatchID})
	if err != nil {
		t.Fatal("bridge Bundle", err)
	}
	if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, next); err != nil {
		t.Fatal("independent Bundle reconstruction", err)
	}
	files, err := i18n.ReadTransportBundle(next, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(gm.ExpectedOutputs[0].BundlePath)
	template := files["formal/protected-candidates/"+name]
	if !bytes.Equal(template, []byte(binding.Template)) || !bytes.Contains(template, []byte("Unrelated prose remains unchanged.")) || bytes.Contains(template, []byte("函数")) || bytes.Contains(template, []byte("const s")) {
		t.Fatal("ordinary/opaque boundaries changed")
	}
	var ordinaryManifest i18n.RetranslationBatchManifest
	if json.Unmarshal(files["formal/manifest.json"], &ordinaryManifest) != nil || len(ordinaryManifest.Units) != 1 || len(gm.ExpectedOutputs) != 1 || ordinaryManifest.Units[0].PreviousFinding != finding || ordinaryManifest.Units[0].RevisionFeedbackSource != "quality_check" || ordinaryManifest.Units[0].RevisionAuthorizationID != "" || ordinaryManifest.Units[0].PreviousSnapshotID != snapshot || ordinaryManifest.Units[0].RepresentationBridge == nil {
		t.Fatal("independent QC feedback lost")
	}
	newRaw := bytes.Replace(template, []byte("Unrelated prose remains unchanged."), []byte("Later QC revised prose."), 1)
	t.Run("retry-keeps-Surface-history-and-independent-QC", func(t *testing.T) {
		rr := cloneBridgeSuccessorFixture(t, root)
		view := *c
		c := &view
		AttachPageReplacementInvocation(rr, c)
		writeFixture(t, rr, exp.BatchPath+"/raw-responses/"+name, []byte("broken response\n"))
		failed, err := i18n.ProcessRetranslationBatch(rr, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: exp.BatchID})
		if err != nil || failed.RestoreFailed != 1 {
			t.Fatal("isolated retry predecessor", failed, err)
		}
		retry, rm, err := i18n.ExportTranslationUnitGenerationBundle(rr, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: exp.BatchID, UnitID: id})
		if err != nil || rm.Attempt != 2 {
			t.Fatal("Surface-derived ordinary bridge retry", err)
		}
		rf, err := i18n.ReadTransportBundle(retry, 8192, 64<<20)
		if err != nil || !bytes.Equal(rf["formal/protected-candidates/"+name], template) || !bytes.Equal(rf["formal/representation-bridge/"+id+".json"], files["formal/representation-bridge/"+id+".json"]) {
			t.Fatal("retry changed template/authority", err)
		}
		dir := t.TempDir()
		writeFixture(t, dir, name, newRaw)
		if _, err := i18n.ImportGenerationOutputDirectory(rr, c, retry, "chatgpt", i18n.FormalChatGPTGenerationModel, dir); err != nil {
			t.Fatal("retry import", err)
		}
		passed, err := i18n.ProcessRetranslationRetry(rr, c, i18n.RetranslationRetryOptions{Locale: "zh-CN", BatchID: exp.BatchID, UnitID: id})
		if err != nil || passed.ValidationPassed != 1 {
			t.Fatal("retry decoder", passed, err)
		}
		if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(rr, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "retry-bridge-successor"}); err != nil {
			t.Fatal("retry historical bridge reader", err)
		}
	})
	surfaceImport(t, root, c, next, map[string][]byte{id: newRaw})
	processed, err = i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: exp.BatchID})
	if err != nil || processed.ValidationPassed != 1 {
		t.Fatal("ordinary bridge process", processed, err)
	}
	candidate, _ := readRegular(root, exp.BatchPath+"/"+processed.Units[0].CandidatePath)
	if !bytes.Contains(candidate, []byte("sql.OpenDB 函数")) || !bytes.Contains(candidate, []byte("Later QC revised prose.")) || !bytes.Contains(candidate, []byte("//go:generate echo keep")) {
		t.Fatal("reviewed opaque payload or code changed")
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "ordinary-bridge-successor"}); err != nil {
		t.Fatal("frozen Surface bridge decoder", err)
	}
	after, _ := readRegular(root, noBridge.BatchPath+"/manifest.json")
	if !bytes.Equal(after, failedManifest) {
		t.Fatal("old zero-output batch changed")
	}
	t.Log("Surface representation evidence remained separate from independent QC B; exact ordinary restore/current-check/process/Snapshot PASS")
}
