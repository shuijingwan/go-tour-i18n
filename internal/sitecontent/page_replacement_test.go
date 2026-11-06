package sitecontent

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func replacementPageFixture(t *testing.T) (string, *i18n.Catalog, *i18n.Catalog, HistoricalPageRepresentations) {
	return replacementPageFixtureSize(t, 6, 2, nil)
}
func replacementPageFixtureSize(t *testing.T, count, unchanged int, customize func(int, []byte) []byte) (string, *i18n.Catalog, *i18n.Catalog, HistoricalPageRepresentations) {
	t.Helper()
	root := siteFixture(t, "current-authorization-fresh-fixture", "docs/RETRANSLATION_RUNBOOK.md", "docs/TRANSLATION_QUALITY_REVIEW.md", "docs/TRANSLATION_UNIT_REPLACEMENT.md", "docs/LANGUAGE_DEFECT_EXHAUSTION.md")
	old, current := &i18n.Catalog{}, &i18n.Catalog{}
	pkg := i18n.TranslationPackageDescriptor{ID: "replacement-fixture", AuthoritySHA256: digest([]byte("frozen-package"))}
	prior := pkg
	frozen := HistoricalPageRepresentations{Package: pkg.ID, PackageSHA256: pkg.AuthoritySHA256}
	for i := 1; i <= count; i++ {
		path := fmt.Sprintf("_content/doc/replacement-%d.md", i)
		source := []byte("# Page\n\nRead this message.\n")
		if i > unchanged {
			source = []byte("# Page\n\nSee <img src=\"image.svg\" alt=\"An image description\"> now.\n")
		}
		if customize != nil {
			source = customize(i, source)
		}
		writeFixture(t, root, path, source)
		d, err := ParseWorkflowDocument(Source{Path: path, Kind: "page", Route: fmt.Sprintf("/doc/replacement-%d", i), SHA256: digest(source)}, source, i)
		if err != nil {
			t.Fatal(err)
		}
		before := *d
		if i > unchanged {
			before.Slots = append([]Unit(nil), d.Slots...)
			for j, s := range before.Slots {
				if !strings.Contains(string(source[s.Start:s.End]), "<img") {
					continue
				}
				s.Source = "See ⟦P0⟧ now."
				s.Protected = []ProtectedToken{{Token: "⟦P0⟧", Raw: "<img src=\"image.svg\" alt=\"An image description\">", Role: "atom"}}
				s.SourceSHA = unitSourceIdentity(s.Source, s.Protected)
				prefix, _, _ := strings.Cut(s.Locator, ":")
				s.Locator = prefix + ":" + s.SourceSHA
				s.ID = digest([]byte(UnitContract + "\n" + s.Locator))
				before.Slots[j] = s
			}
			slots := before
			slots.Units = before.Slots
			before, err = FormalPage(slots, source, i)
			if err != nil {
				t.Fatal(err)
			}
		}
		frozen.Documents = append(frozen.Documents, before)
		binding := i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: d.Units[0].ID, Kind: i18n.UnitKindPage, SourcePath: path, SourceSHA256: d.SourceSHA, Source: source}, Adapter: PageAdapter{*d, source}, StableIndex: i, Route: d.Route}
		pkg.Units = append(pkg.Units, binding)
		binding.Unit.ID, binding.Adapter = before.Units[0].ID, PageAdapter{before, source}
		prior.Units = append(prior.Units, binding)
	}
	for _, p := range []struct {
		c *i18n.Catalog
		d i18n.TranslationPackageDescriptor
	}{{old, prior}, {current, pkg}} {
		if err := p.c.RegisterTranslationPackage(p.d); err != nil {
			t.Fatal(err)
		}
		if err := p.c.UseTranslationPackage(p.d.ID); err != nil {
			t.Fatal(err)
		}
		AttachPageReplacementResolver(p.c)
	}
	return root, old, current, frozen
}

func freezeReplacementFixture(t *testing.T, root string, f HistoricalPageRepresentations) i18n.GlossaryArchiveReference {
	t.Helper()
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	path := "data/replacement-fixture/" + digest(b) + ".json"
	writeFixture(t, root, path, b)
	return i18n.GlossaryArchiveReference{Path: path, SHA256: digest(b)}
}

func TestSurfaceReplacementMixedAggregateRevisionAndRecovery(t *testing.T) {
	root, old, current, frozen := replacementPageFixture(t)
	ids := append([]string(nil), old.TranslationScope...)
	f := surfaceStart(t, root, old, ids)
	if _, _, err := BuildTranslationPackageClosure(root, old, "zh-CN", frozen.Package, f.SnapshotID); err != nil {
		t.Fatal("ordinary finalized descriptor closure", err)
	}
	// Isolate replacement lifecycle from unrelated corpus/production planning.
	context, err := ReferenceFile(root, "data/unified-glossary-reviews/zh-CN/synthetic-unified-review.review.json")
	if err != nil {
		t.Fatal(err)
	}
	s := IntegratedSurfaceScope{Schema: IntegratedSurfaceSchema, Locale: "zh-CN", UnifiedReview: context}
	for _, u := range f.Units {
		s.Reviewed = append(s.Reviewed, SurfaceAsset{ID: u.Snapshot.SourcePath, SourcePath: u.Snapshot.SourcePath, CandidateArtifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: f.SnapshotID, Unit: u.Snapshot}}})
	}
	s.Identity = identity(s)
	build := func(string, string, []Reference) (*IntegratedSurfaceScope, error) { copy := s; return &copy, nil }
	attachIntegratedSurfaceAuthority(old, build)
	attachIntegratedSurfaceAuthority(current, build)
	findings := []SurfaceFinding{}
	texts := []string{}
	for _, u := range f.Units {
		text := fmt.Sprintf("Correct exact message %d.", u.Index)
		texts = append(texts, text)
		findings = append(findings, SurfaceFinding{ID: u.Snapshot.SourcePath, Finding: text, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: f.SnapshotID, Unit: u.Snapshot}}})
	}
	r, ref := surfaceReceipt(t, root, old, build, nil, "replacement-surface", "failed", findings)
	o := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "replacement-old", PreviousSnapshotID: f.SnapshotID, SurfaceReviewID: r.ReviewID, Findings: texts, UnitIDs: ids, IntegratedSurface: true}
	_, oldPath, err := i18n.RecordQualityCheckSurfaceReopen(root, old, o)
	if err != nil {
		t.Fatal(err)
	}
	protectedPaths := []string{oldPath, ref.Path, r.Evidence.Path, context.Path, f.SnapshotManifestPath, "data/quality-check-snapshots/zh-CN/" + f.SnapshotID + "/finalization.json"}
	immutable := map[string][]byte{}
	for _, p := range protectedPaths {
		immutable[p], err = readRegular(root, p)
		if err != nil {
			t.Fatal(err)
		}
	}
	artifact := findings[2].Artifacts[0]
	if err := i18n.ValidateHistoricalSurfaceCandidateArtifact(root, old, "zh-CN", artifact); err != nil {
		t.Fatal(err)
	}
	if err := i18n.ValidateSurfaceCandidateArtifact(root, current, "zh-CN", artifact); err == nil {
		t.Fatal("historical artifact impersonated current Unit")
	}
	o.PreviousReopenID, o.ReopenID = "replacement-old", "replacement-successor"
	o.HistoricalDocuments = freezeReplacementFixture(t, root, frozen)
	o.UnitIDs = current.TranslationScope
	successor, successorPath, err := i18n.RecordQualityCheckSurfaceReopen(root, current, o)
	if err != nil {
		t.Fatal(err)
	}
	if successor.UnitCount != 6 {
		t.Fatal("mixed scope split")
	}
	for _, p := range []string{successorPath, o.HistoricalDocuments.Path} {
		immutable[p], err = readRegular(root, p)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i, u := range successor.Units {
		if (u.Replacement == nil) != (i < 2) || u.Snapshot.UnitID != ids[i] || u.UnitID != current.TranslationScope[i] {
			t.Fatal("mixed explicit identity mismatch")
		}
	}
	if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, current, o); err == nil {
		t.Fatal("successor overwritten")
	}
	export := i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: o.UnitIDs, PreviousSnapshotID: f.SnapshotID, SurfaceReopenID: o.ReopenID, AllowReexport: true}
	e, err := i18n.ExportRetranslationBatch(root, current, export)
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := i18n.ExportTranslationUnitGenerationBundle(root, current, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(b, 512, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	for i, out := range m.ExpectedOutputs {
		name := filepath.Base(out.BundlePath)
		candidate, err := readRegular(root, f.Units[i].Snapshot.CandidatePath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(files["formal/candidates/"+name], candidate) {
			t.Fatal("lost historical semantic predecessor")
		}
		u, _ := current.Unit(out.UnitID)
		restored, err := u.Adapter.Restore(string(files["formal/protected-candidates/"+name]))
		if err != nil || !bytes.Equal(restored, candidate) {
			t.Fatal("rebased baseline not byte-exact", err)
		}
		if i >= 2 && len(files["formal/historical-protected-candidates/"+name]) == 0 {
			t.Fatal("lost historical raw")
		}
	}
	archive := filepath.Join(t.TempDir(), "partial.zip")
	var zipBytes bytes.Buffer
	z := zip.NewWriter(&zipBytes)
	for i := 0; i < 6; i++ {
		name := ids[i] + ".article"
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		u, _ := old.Unit(ids[i])
		input, _, _ := u.Adapter.ProtectedInput(&i18n.Glossary{})
		if _, err := w.Write([]byte(input + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, zipBytes.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	recovery, err := i18n.RecoverGenerationOutputs(root, current, b, []string{archive}, filepath.Join(t.TempDir(), "recovered"))
	if err != nil || len(recovery.Recovered) != 2 || len(recovery.Missing) != 4 {
		t.Fatal("trusted unchanged recovery", recovery, err)
	}
	for i := 0; i < 2; i++ {
		got, err := os.ReadFile(filepath.Join(recovery.OutputDir, ids[i]+".article"))
		u, _ := old.Unit(ids[i])
		input, _, _ := u.Adapter.ProtectedInput(&i18n.Glossary{})
		if err != nil || string(got) != input+"\n" {
			t.Fatal("trusted bytes changed")
		}
	}
	// Complete only this isolated fixture, then prove current-ID validation/QC
	// does not inherit A for any of the replaced or revised candidates.
	outputs := map[string][]byte{}
	for _, out := range m.ExpectedOutputs {
		name := filepath.Base(out.BundlePath)
		outputs[out.UnitID] = bytes.Replace(files["formal/protected-candidates/"+name], []byte("Page"), []byte("Revised page"), 1)
	}
	surfaceImport(t, root, current, b, outputs)
	result, err := i18n.ProcessRetranslationBatch(root, current, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil || result.ValidationPassed != 6 {
		t.Fatal(result, err)
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, current, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "replacement-current"}); err != nil {
		t.Fatal(err)
	}
	scope, err := i18n.BuildQualityCheckScope(root, current, i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "replacement-current", PreviousSnapshotID: f.SnapshotID})
	if err != nil || scope.PendingCount != 6 || scope.ACount != 0 {
		t.Fatal("replacement must receive new QC", scope, err)
	}
	if _, _, err := i18n.ExportQualityCheckReviewerBundle(root, current, i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: "replacement-current", PreviousSnapshotID: f.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, current, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "replacement-current", PreviousSnapshotID: f.SnapshotID, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: current.TranslationScope, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(root, current, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "replacement-current", PreviousSnapshotID: f.SnapshotID}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.VerifyQualityCheckFinalization(root, current, "zh-CN", "replacement-current"); err != nil {
		t.Fatal(err)
	}
	t.Run("package-closure-replacement-authority", func(t *testing.T) {
		assertReplacementPackageClosure(t, root, current, frozen.Package, "replacement-current", context, ref, oldPath, successorPath, o.HistoricalDocuments.Path)
	})
	export.BatchID = "chatgpt-zh-CN-099"
	if _, err := i18n.ExportRetranslationBatch(root, current, export); err == nil {
		t.Fatal("consumed successor authorized another batch")
	}
	for p, want := range immutable {
		got, err := readRegular(root, p)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("immutable evidence modified", p)
		}
	}
	// Canonical bridge current-check rejects changed representation evidence.
	writeFixture(t, root, o.HistoricalDocuments.Path, []byte("stale"))
	if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, current, b); err == nil {
		t.Fatal("stale replacement proof remained current")
	}
}

func TestReconcilePageReplacementFailsClosed(t *testing.T) {
	root, _, c, frozen := replacementPageFixture(t)
	for _, name := range []string{"ambiguous", "path-move", "source-change", "registry-change", "route-change", "package-change", "unrelated-unit", "forged-old-slot"} {
		t.Run(name, func(t *testing.T) {
			b, _ := json.Marshal(frozen)
			var bad HistoricalPageRepresentations
			_ = json.Unmarshal(b, &bad)
			switch name {
			case "ambiguous":
				bad.Documents[3] = bad.Documents[2]
			case "path-move":
				bad.Documents[2].Path = "_content/doc/moved.md"
			case "source-change":
				bad.Documents[2].SourceSHA = digest([]byte("changed source"))
			case "registry-change":
				bad.Documents[2].StableIndex = 4
			case "route-change":
				bad.Documents[2].Route = "/moved"
			case "package-change":
				bad.Package = "other-package"
			case "unrelated-unit":
				bad.Documents[2] = bad.Documents[0]
			case "forged-old-slot":
				bad.Documents[2].Slots[0].End++
			}
			if _, err := resolvePageReplacements(root, c, freezeReplacementFixture(t, root, bad)); err == nil {
				t.Fatal("invalid replacement accepted")
			}
		})
	}
	t.Run("stale-current-proof", func(t *testing.T) {
		copy := *c
		copy.TranslationPackages = append([]i18n.TranslationPackageDescriptor(nil), c.TranslationPackages...)
		copy.TranslationPackages[0].Units = append([]i18n.AdaptedTranslationUnit(nil), c.TranslationPackages[0].Units...)
		u := &copy.TranslationPackages[0].Units[2]
		a := u.Adapter.(PageAdapter)
		a.Document = frozen.Documents[2]
		u.Adapter = a
		if _, err := resolvePageReplacements(root, &copy, freezeReplacementFixture(t, root, frozen)); err == nil {
			t.Fatal("stale current parser proof accepted")
		}
	})
}

func TestFrozenPageReplacementReconciliation(t *testing.T) {
	oldPath := os.Getenv("GO_TOUR_REPLACEMENT_BEFORE_DOCUMENTS")
	bundlePath := os.Getenv("GO_TOUR_INLINE_HTML_BEFORE_BUNDLE")
	if oldPath == "" || bundlePath == "" {
		t.Skip("explicit read-only frozen representation/Bundle fixture")
	}
	b, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	var old []Document
	if err := json.Unmarshal(b, &old); err != nil {
		t.Fatal(err)
	}
	root := repositoryRoot(t)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	next, raw, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	r, err := ReconcileHistoricalPages(old, next)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(bundlePath)
	if err != nil || digest(bundle) != "fab3d6a3a4587201ed0642a1800d73a4ebd58f4463d7acead3fc05b04522fc83" {
		t.Fatal("old Bundle fixture identity", err)
	}
	files, err := i18n.ReadTransportBundle(bundle, 512, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var batch i18n.RetranslationBatchManifest
	if err := StrictJSON(files["formal/manifest.json"], &batch); err != nil {
		t.Fatal(err)
	}
	var snapshot i18n.QualityCheckSnapshotManifest
	if err := StrictJSON(files["formal/predecessors/learn-docs-v1-qc-009/manifest.json"], &snapshot); err != nil {
		t.Fatal(err)
	}
	preservedSnapshot, err := os.ReadFile(filepath.Join(root, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-009/manifest.json"))
	if err != nil || !bytes.Equal(preservedSnapshot, files["formal/predecessors/learn-docs-v1-qc-009/manifest.json"]) {
		t.Fatal("immutable real Snapshot differs from preserved Bundle", err)
	}
	changes := map[string]UnitChange{}
	for _, u := range r.Units {
		changes[u.ID] = u
	}
	checked := 0
	for _, record := range batch.Units {
		for i, d := range old {
			if d.Path != record.SourcePath {
				continue
			}
			checked++
			n := next[i]
			if d.Units[0].ID != record.UnitID || snapshot.Units[d.StableIndex-1].UnitID != record.UnitID || d.SourceSHA != record.SourceSHA256 || d.StableIndex != n.StableIndex || d.Route != n.Route {
				t.Fatal("frozen predecessor identity mismatch")
			}
			input := []byte(stringsTrimLF(n.Units[0].Source) + "\n")
			before := files["formal/"+record.InputPath]
			if d.StableIndex == 10 || d.StableIndex == 29 {
				if n.Units[0].ID != record.UnitID || !bytes.Equal(input, before) {
					t.Fatal("trusted Unit changed")
				}
			} else if n.Units[0].ID == record.UnitID || bytes.Equal(input, before) || changes[record.UnitID].Replacement != n.Units[0].ID {
				t.Fatal("missing unique replacement")
			}
			baseline := files["formal/protected-candidates/"+filepath.Base(record.InputPath)]
			a := PageAdapter{n, raw[n.Path]}
			rebased, err := a.RebaseRevisionPredecessor(PageAdapter{d, raw[d.Path]}, baseline)
			if err != nil {
				t.Fatal("real predecessor rebase", d.Path, err)
			}
			candidate, err := a.Restore(string(rebased))
			if err != nil || !bytes.Equal(append(bytes.TrimRight(candidate, "\n"), '\n'), files["formal/candidates/"+filepath.Base(record.InputPath)]) {
				t.Fatal("real candidate binding", err)
			}
			t.Logf("stable=%d old=%s current=%s physical_sha=%s mapping=%s", d.StableIndex, record.UnitID, n.Units[0].ID, n.SourceSHA, changes[record.UnitID].State)
		}
	}
	if checked != 6 {
		t.Fatal("real exact six-page regression fixture incomplete", checked)
	}
}
