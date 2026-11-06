package sitecontent

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Same test-only future card representation as the shared conformance fixture.
// Its adapter knows XML structure only, never Surface/lifecycle policy.
type surfaceCard struct {
	XMLName xml.Name `xml:"card"`
	ID      string   `xml:"id,attr"`
	Href    string   `xml:"href,attr"`
	Heading string   `xml:"heading"`
	Message string   `xml:"message"`
}
type surfaceCardAdapter struct{ card surfaceCard }

// This schema contains prose fields and immutable identity attributes only;
// it has no executable/code representation or teaching-comment coordinates.
func (a surfaceCardAdapter) CommentSourceBlocks() ([]i18n.CommentCodeBlock, string, error) {
	return nil, a.card.Heading, nil
}
func (a surfaceCardAdapter) CommentTargetBlocks(b []byte) ([]i18n.CommentCodeBlock, error) {
	return nil, a.Validate(b, &i18n.Glossary{})
}
func (a surfaceCardAdapter) ReattestationAuthority(root string) (string, error) {
	const source = "internal/sitecontent/surface_authority_test.go"
	// Include the card schema used by the methods, not just the adapter AST.
	return i18n.TranslationAdapterAuthority(root, source, "surfaceCardAdapter", []string{source})
}
func (a surfaceCardAdapter) GlossaryVisiblePair(b []byte) (string, string, error) {
	if err := a.Validate(b, &i18n.Glossary{}); err != nil {
		return "", "", err
	}
	var card surfaceCard
	if err := xml.Unmarshal(b, &card); err != nil {
		return "", "", err
	}
	return a.card.Heading + "\n" + a.card.Message, card.Heading + "\n" + card.Message, nil
}

func (a surfaceCardAdapter) ProtectedInput(_ *i18n.Glossary) (string, []string, error) {
	return "⟬CARD_BEGIN⟭" + a.card.Heading + "⟬CARD_MESSAGE⟭" + a.card.Message + "⟬CARD_END⟭\n", []string{"⟬CARD_BEGIN⟭", "⟬CARD_MESSAGE⟭", "⟬CARD_END⟭"}, nil
}
func (a surfaceCardAdapter) Restore(raw string) ([]byte, error) {
	for _, s := range []string{"⟬CARD_BEGIN⟭", "⟬CARD_MESSAGE⟭", "⟬CARD_END⟭"} {
		if strings.Count(raw, s) != 1 {
			return nil, fmt.Errorf("card token membership")
		}
	}
	raw = strings.TrimSuffix(raw, "\n")
	if !strings.HasPrefix(raw, "⟬CARD_BEGIN⟭") || !strings.HasSuffix(raw, "⟬CARD_END⟭") {
		return nil, fmt.Errorf("card envelope")
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(raw, "⟬CARD_BEGIN⟭"), "⟬CARD_END⟭"), "⟬CARD_MESSAGE⟭")
	card := a.card
	card.Heading, card.Message = parts[0], parts[1]
	b, err := xml.Marshal(card)
	return append(b, '\n'), err
}
func (a surfaceCardAdapter) Validate(b []byte, g *i18n.Glossary) error {
	var card surfaceCard
	if err := xml.Unmarshal(b, &card); err != nil {
		return err
	}
	canonical, _ := xml.Marshal(card)
	if !bytes.Equal(append(canonical, '\n'), b) || card.ID != a.card.ID || card.Href != a.card.Href || card.Heading == "" || card.Message == "" {
		return fmt.Errorf("XML structure identity")
	}
	return i18n.ValidateParsedGlossary(a.card.Heading+"\n"+a.card.Message, card.Heading+"\n"+card.Message, g)
}
func surfaceFixture(t *testing.T, representation string, counts ...int) (string, *i18n.Catalog, []string) {
	t.Helper()
	// Build the complete canonical source/corpus fixture before registering the
	// independent test package. A real corpus copied into a sparse legacy root
	// is not valid current authority for that root.
	root := siteFixture(t, "current-authorization-fresh-fixture", "docs/RETRANSLATION_RUNBOOK.md", "docs/TRANSLATION_QUALITY_REVIEW.md")
	if representation == "xml" {
		const authority = "internal/sitecontent/surface_authority_test.go"
		b, err := readRegular(repositoryRoot(t), authority)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, authority, b)
	}
	if representation == "tour" {
		catalog, err := checkedCatalog(root)
		if err != nil {
			t.Fatal(err)
		}
		// Native Tour has no registered three-Unit package. Keep the complete
		// canonical catalog, including eligible Examples, for current authority.
		progress, err := i18n.BuildTranslationUnitProgress(root, catalog, "zh-CN")
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]string, 0, progress.Total)
		for _, u := range progress.Units {
			ids = append(ids, u.UnitID)
		}
		return root, catalog, ids
	}
	c := &i18n.Catalog{}
	count := 3
	if len(counts) > 0 {
		count = counts[0]
	}
	ids := []string{}
	d := i18n.TranslationPackageDescriptor{ID: "test-surface-" + representation, AuthoritySHA256: digest([]byte("test-only-frozen-surface-inventory/" + representation))}
	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("surface/%s-%d", representation, i)
		ids = append(ids, id)
		route := fmt.Sprintf("/fixture/%s/%d", representation, i)
		var source []byte
		var adapter i18n.TranslationUnitAdapter
		var path string
		if representation == "xml" {
			card := surfaceCard{ID: fmt.Sprintf("card-%d", i), Href: "https://example.invalid/identity", Heading: "A future card", Message: "Read this message."}
			source, _ = xml.Marshal(card)
			source = append(source, '\n')
			adapter = surfaceCardAdapter{card}
			path = fmt.Sprintf("fixtures/future/card-%d.xml", i)
		} else {
			path = fmt.Sprintf("_content/doc/tutorial/surface-%d.md", i)
			source = []byte("# A page\n\nRead this message.\n")
			doc, err := ParseWorkflowDocument(Source{Path: path, Kind: "page", SHA256: digest(source), Route: route}, source, i)
			if err != nil {
				t.Fatal(err)
			}
			id = doc.Units[0].ID
			ids[len(ids)-1] = id
			adapter = PageAdapter{*doc, source}
		}
		writeFixture(t, root, path, source)
		d.Units = append(d.Units, i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: id, Kind: i18n.UnitKindPage, SourcePath: path, SourceSHA256: digest(source), Source: source}, Adapter: adapter, StableIndex: i, Route: route})
	}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	return root, c, ids
}

func TestSurfaceFixtureCompleteRegisteredSources(t *testing.T) {
	root, declared, ids := surfaceFixture(t, "learn-docs")
	catalog, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, catalog); err != nil {
		t.Fatal("fixture lacks canonical registered source inventory", err)
	}
	if len(catalog.TranslationPackages) == 0 {
		t.Fatal("missing canonical packages")
	}
	for _, descriptor := range catalog.TranslationPackages {
		selected := *catalog
		if err := selected.UseTranslationPackage(descriptor.ID); err != nil {
			t.Fatal(err)
		}
		if len(selected.TranslationScope) != len(descriptor.Units) {
			t.Fatal("partial canonical package fixture", descriptor.ID)
		}
		for _, binding := range descriptor.Units {
			b, err := readRegular(root, binding.Unit.SourcePath)
			if err != nil || digest(b) != binding.Unit.SourceSHA256 {
				t.Fatal("canonical source bytes missing/mismatched", binding.Unit.SourcePath, err)
			}
		}
	}
	if len(declared.TranslationPackages) != 1 || len(declared.TranslationScope) != len(declared.TranslationPackages[0].Units) || len(ids) != len(declared.TranslationScope) {
		t.Fatal("partial declared test package")
	}
	for _, binding := range declared.TranslationPackages[0].Units {
		b, err := readRegular(root, binding.Unit.SourcePath)
		if err != nil || digest(b) != binding.Unit.SourceSHA256 {
			t.Fatal("declared source bytes missing/mismatched", binding.Unit.SourcePath, err)
		}
	}
}

func surfaceImport(t *testing.T, root string, c *i18n.Catalog, b []byte, outputs map[string][]byte) {
	t.Helper()
	dir := t.TempDir()
	files, err := i18n.ReadTransportBundle(b, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.GenerationBundleManifest
	if err := StrictJSON(files["manifest.json"], &m); err != nil {
		t.Fatal(err)
	}
	for _, out := range m.ExpectedOutputs {
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(out.BundlePath)), outputs[out.UnitID], 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := i18n.ImportGenerationOutputDirectory(root, c, b, "chatgpt", i18n.FormalChatGPTGenerationModel, dir); err != nil {
		t.Fatal(err)
	}
}
func surfaceFinalize(t *testing.T, root string, c *i18n.Catalog, snapshot, previous string, ids []string, locales ...string) *i18n.QualityCheckFinalization {
	t.Helper()
	locale := fixtureLocale(locales)
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: locale, SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	for _, batch := range surfaceFixtureUnitBatches(t, c, ids) {
		if _, _, err := i18n.ExportQualityCheckReviewerBundle(root, c, i18n.QualityCheckReviewerBundleOptions{Locale: locale, SnapshotID: snapshot, PreviousSnapshotID: previous}); err != nil {
			t.Fatal(err)
		}
		if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: locale, SnapshotID: snapshot, PreviousSnapshotID: previous, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: batch, Rating: "A"}); err != nil {
			t.Fatal(err)
		}
	}
	f, _, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: locale, SnapshotID: snapshot, PreviousSnapshotID: previous})
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func surfaceStart(t *testing.T, root string, c *i18n.Catalog, ids []string, locales ...string) *i18n.QualityCheckFinalization {
	t.Helper()
	locale := fixtureLocale(locales)
	for _, batch := range surfaceFixtureUnitBatches(t, c, ids) {
		u, err := c.Unit(batch[0])
		if err != nil {
			t.Fatal(err)
		}
		e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: batch, UnitKind: u.Kind})
		if err != nil {
			t.Fatal(err)
		}
		b, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: locale, BatchID: e.BatchID})
		if err != nil {
			t.Fatal(err)
		}
		files, _ := i18n.ReadTransportBundle(b, 8192, 64<<20)
		outputs := map[string][]byte{}
		for _, o := range m.ExpectedOutputs {
			outputs[o.UnitID] = files["formal/inputs/"+filepath.Base(o.BundlePath)]
		}
		surfaceImport(t, root, c, b, outputs)
		if r, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: locale, BatchID: e.BatchID}); err != nil || r.ValidationPassed != len(batch) {
			t.Fatal(r, err)
		}
	}
	return surfaceFinalize(t, root, c, "surface-initial", "", ids, locale)
}

// Synthetic setup obeys the same size/kind boundaries as the real workflow.
func surfaceFixtureUnitBatches(t *testing.T, c *i18n.Catalog, ids []string) [][]string {
	t.Helper()
	var batches [][]string
	var kind i18n.UnitKind
	for _, id := range ids {
		u, err := c.Unit(id)
		if err != nil {
			t.Fatal(err)
		}
		if len(batches) == 0 || len(batches[len(batches)-1]) == i18n.MaxRetranslationExportLimit || kind != u.Kind {
			batches = append(batches, nil)
			kind = u.Kind
		}
		batches[len(batches)-1] = append(batches[len(batches)-1], id)
	}
	return batches
}
func surfaceDescriptorClosure(t *testing.T, root string, c *i18n.Catalog, snapshot string) Reference {
	t.Helper()
	closure, _, err := BuildTranslationPackageClosure(root, c, "zh-CN", c.TranslationPackages[0].ID, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	path := workflowPath(closure.Locale, closure.Package, "closure", closure.Identity)
	if err := saveImmutable(root, path, closure); err != nil {
		t.Fatal(err)
	}
	ref, err := ReferenceFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
func surfaceBuilder(t *testing.T, root string, c *i18n.Catalog, f **i18n.QualityCheckFinalization, locales ...string) integratedScopeBuilder {
	t.Helper()
	base, err := BuildIntegratedSurfaceScope(root, fixtureLocale(locales), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.TranslationPackages) > 0 {
		return func(root, locale string, refs []Reference) (*IntegratedSurfaceScope, error) {
			return integratedSurfaceScopeForCatalog(root, c, *base, refs)
		}
	}
	// Tour's section context is an inventory representation adapter. It identifies
	// the section explicitly; shared exact mapping/authorization remains identical.
	return func(root, locale string, refs []Reference) (*IntegratedSurfaceScope, error) {
		copy := *base
		s := &copy
		s.Reviewed = append([]SurfaceAsset(nil), base.Reviewed...)
		s.Carried = append([]SurfaceAsset(nil), base.Carried...)
		for _, u := range (*f).Units {
			s.Carried = append(s.Carried, SurfaceAsset{ID: "tu:" + u.UnitID, UnitID: u.UnitID, SourcePath: u.Snapshot.SourcePath, State: "carry", SourceSHA: u.Snapshot.SourceSHA256, TargetSHA: u.Snapshot.CandidateSHA256})
		}
		if err := BindIntegratedSurfaceCandidates(root, c, s, *f); err != nil {
			return nil, err
		}
		s.Identity = ""
		s.Identity = identity(*s)
		return s, nil
	}
}
func surfaceReceipt(t *testing.T, root string, c *i18n.Catalog, build integratedScopeBuilder, closures []Reference, review string, decision string, findings []SurfaceFinding, locales ...string) (*IntegratedSurfaceReceipt, Reference) {
	t.Helper()
	scope, err := build(root, fixtureLocale(locales), closures)
	if err != nil {
		t.Fatal(err)
	}
	evidence := Reference{Path: "data/fixture-surface/" + review + ".txt"}
	body := []byte("Independent exact reviewed canonical artifacts\n")
	for _, f := range findings {
		body = append(body, []byte(f.ID+": "+f.Finding+"\n")...)
	}
	evidence.SHA256 = digest(body)
	writeFixture(t, root, evidence.Path, body)
	r := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: review, Reviewer: "synthetic-independent-reviewer", Decision: decision, Findings: []string{}, ExactFindings: findings, Evidence: evidence}
	if decision == "failed" {
		r.Findings = []string{"explicit fixture defect"}
	}
	b, _ := Encode(r)
	stored, path, err := recordIntegratedSurface(root, b, c, build)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := ReferenceFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return stored, ref
}

func TestIntegratedSurfaceSharedAuthorityRepresentations(t *testing.T) {
	for _, representation := range []string{"tour", "learn-docs", "xml"} {
		t.Run(representation, func(t *testing.T) {
			root, c, ids := surfaceFixture(t, representation)
			locale := "zh-CN"
			if representation == "tour" {
				locale = "en-GB"
				englishCompatibilityLocale(t, root)
			}
			f := surfaceStart(t, root, c, ids, locale)
			build := surfaceBuilder(t, root, c, &f, locale)
			attachIntegratedSurfaceAuthority(c, build)
			closures := []Reference{}
			if len(c.TranslationPackages) > 0 {
				closures = append(closures, surfaceDescriptorClosure(t, root, c, f.SnapshotID))
			}
			finding := "Correct the reviewed message."
			artifact := i18n.SurfaceCandidateArtifact{SnapshotID: f.SnapshotID, Unit: f.Units[1].Snapshot}
			selected := ids[1:2]
			findings := []SurfaceFinding{{ID: artifact.Unit.SourcePath, Finding: finding, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{artifact}}}
			if representation == "xml" {
				selected = ids[1:3]
				second := i18n.SurfaceCandidateArtifact{SnapshotID: f.SnapshotID, Unit: f.Units[2].Snapshot}
				findings = append(findings, SurfaceFinding{ID: second.Unit.SourcePath, Finding: finding, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{second}})
			}
			r, _ := surfaceReceipt(t, root, c, build, closures, "surface-failed", "failed", findings, locale)
			o := i18n.QualityCheckSurfaceReopenOptions{Locale: locale, ReopenID: "integrated-reopen", PreviousSnapshotID: f.SnapshotID, SurfaceReviewID: r.ReviewID, Finding: finding, UnitIDs: selected, IntegratedSurface: true}
			reopened, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, o)
			if err != nil {
				t.Fatal(err)
			}
			if reopened.IntegratedAuthority == nil || reopened.Units[0].Snapshot.CandidateSHA256 != artifact.Unit.CandidateSHA256 {
				t.Fatal("canonical artifact not bound")
			}
			e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: selected, AllowReexport: true, PreviousSnapshotID: f.SnapshotID, SurfaceReopenID: o.ReopenID})
			if err != nil {
				t.Fatal(err)
			}
			bundleOptions := i18n.GenerationBundleOptions{Locale: locale, BatchID: e.BatchID}
			if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, bundleOptions); err == nil || !strings.Contains(err.Error(), "capability authority required") {
				t.Fatal("Surface handoff without recorded capability accepted", err)
			}
			discovery, err := i18n.PrepareSurfaceRevisionTransport(root, c, locale, e.BatchID, nil, false)
			if representation != "learn-docs" {
				// Shared receipt/reopen mapping is representation-independent.
				// Current executable transport is deliberately Page-only; native
				// Tour and this XML adapter must not inherit a Page codec.
				if err == nil || !strings.Contains(err.Error(), "Surface current source context unavailable") {
					t.Fatal("unsupported representation received Surface transport", err)
				}
				if _, err := os.Stat(filepath.Join(root, "data/surface-revision-transports", locale, o.ReopenID+".json")); !os.IsNotExist(err) {
					t.Fatal("unsupported transport manufactured authority", err)
				}
				t.Log("exact shared receipt/artifacts -> reopen PASS; unsupported transport rejected before Generation")
				return
			}
			if err != nil {
				t.Fatal("Page transport discovery", err)
			}
			request := discovery.Authority.Request
			for _, binding := range discovery.Authority.Bindings {
				source, err := c.Unit(binding.UnitID)
				if err != nil {
					t.Fatal(err)
				}
				page, ok := source.Adapter.(PageAdapter)
				if !ok || len(page.Document.Slots) != 2 {
					t.Fatal("unexpected synthetic Page source shape")
				}
				unit := i18n.SurfaceTransportUnitRequest{UnitID: binding.UnitID, FindingSHA256: binding.FindingSHA256}
				for _, region := range binding.Projection.Available {
					// The fixture explicitly assigns its body slot to this finding;
					// neither the exporter nor finding prose infers editable scope.
					if region.Kind == "ordinary-prose" && region.SourceCoordinate == page.Document.Slots[1].Locator {
						unit.Regions = []string{region.ID}
					}
				}
				if len(unit.Regions) != 1 {
					t.Fatalf("missing exact fixture body coordinate: %+v", binding.Projection.Available)
				}
				request.Units = append(request.Units, unit)
			}
			if plan, err := i18n.PrepareSurfaceRevisionTransport(root, c, locale, e.BatchID, &request, true); err != nil || !plan.Recorded || !plan.RevisionTransportCapable {
				t.Fatal("isolated exact Page capability authority", err)
			}
			b, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
			if err != nil {
				t.Fatal(err)
			}
			files, _ := i18n.ReadTransportBundle(b, 8192, 64<<20)
			good, bad := map[string][]byte{}, map[string][]byte{}
			for _, out := range m.ExpectedOutputs {
				good[out.UnitID] = bytes.Replace(files["formal/protected-candidates/"+filepath.Base(out.BundlePath)], []byte("message"), []byte("corrected message"), 1)
				bad[out.UnitID] = []byte("broken protected response\n")
			}
			// A deliberately invalid protected output forces the same machine retry.
			surfaceImport(t, root, c, b, bad)
			result, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID})
			if err != nil || result.RestoreFailed != len(selected) {
				t.Fatal(result, err)
			}
			bundles := [][]byte{}
			for _, id := range selected {
				retry, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id})
				if err != nil {
					t.Fatal(err)
				}
				retryFiles, _ := i18n.ReadTransportBundle(retry, 8192, 64<<20)
				for path, data := range files {
					if strings.HasPrefix(path, "formal/surface-authority/") && !bytes.Equal(data, retryFiles[path]) {
						t.Fatal("origin Surface authority lost in retry")
					}
				}
				bundles = append(bundles, retry)
			}
			aggregate, _, err := i18n.ExportRetryHandoff(root, c, bundles)
			if err != nil {
				t.Fatal(err)
			}
			_, originals, err := i18n.ReadRetryHandoff(aggregate)
			if err != nil {
				t.Fatal(err)
			}
			for i, retry := range originals {
				id := selected[i]
				if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, retry); err != nil {
					t.Fatal("Integrated Surface sibling retry independence", err)
				}
				surfaceImport(t, root, c, retry, map[string][]byte{id: good[id]})
				result, err = i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id})
				if err != nil || result.ValidationPassed != i+1 {
					t.Fatal(result, err)
				}
				if _, err := i18n.VerifyCurrentRetryHandoff(root, c, aggregate); err == nil {
					t.Fatal("consumed aggregate stayed current")
				}
			}
			// Original failed receipt cannot authorize a new invocation after candidate drift.
			o.ReopenID = "stale-second-reopen"
			if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, o); err == nil {
				t.Fatal("stale receipt reopened")
			}
			if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "surface-successor"}); err != nil {
				t.Fatal(err)
			}
			scope, err := i18n.BuildQualityCheckScope(root, c, i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "surface-successor", PreviousSnapshotID: f.SnapshotID})
			if err != nil || scope.PendingCount != len(selected) || scope.ACount != len(ids)-len(selected) {
				t.Fatal("unaffected A carry", scope, err)
			}
			if _, _, err := i18n.ExportQualityCheckReviewerBundle(root, c, i18n.QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: "surface-successor", PreviousSnapshotID: f.SnapshotID}); err != nil {
				t.Fatal(err)
			}
			if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "surface-successor", PreviousSnapshotID: f.SnapshotID, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: selected, Rating: "A"}); err != nil {
				t.Fatal(err)
			}
			f, _, err = i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "surface-successor", PreviousSnapshotID: f.SnapshotID})
			if err != nil {
				t.Fatal(err)
			}
			if len(c.TranslationPackages) > 0 {
				closures = []Reference{surfaceDescriptorClosure(t, root, c, f.SnapshotID)}
			}
			_, pass := surfaceReceipt(t, root, c, build, closures, "surface-pass", "passed", nil)
			if len(closures) > 0 {
				gate, err := requireIntegratedSurface(root, "zh-CN", closures[0], build)
				if err != nil || gate != pass {
					t.Fatal("Integrated Surface/package closure", gate, err)
				}
				if _, _, err := CheckTranslationPackageClosure(root, c, closures[0]); err != nil {
					t.Fatal(err)
				}
			}
			t.Logf("same shared receipt/artifact mapping → reopen → revision → origin retry → successor re-QC/A-only → Integrated closure; unaffected A=%d", len(ids)-len(selected))
		})
	}
}

func TestIntegratedSurfaceAuthorityFailClosed(t *testing.T) {
	root, c, ids := surfaceFixture(t, "xml")
	f := surfaceStart(t, root, c, ids)
	build := surfaceBuilder(t, root, c, &f)
	attachIntegratedSurfaceAuthority(c, build)
	closure := surfaceDescriptorClosure(t, root, c, f.SnapshotID)
	artifact := i18n.SurfaceCandidateArtifact{SnapshotID: f.SnapshotID, Unit: f.Units[1].Snapshot}
	finding := "Correct the reviewed message."
	receipt, ref := surfaceReceipt(t, root, c, build, []Reference{closure}, "negative-receipt", "failed", []SurfaceFinding{{ID: artifact.Unit.SourcePath, Finding: finding, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{artifact}}})
	options := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", PreviousSnapshotID: f.SnapshotID, SurfaceReviewID: receipt.ReviewID, Finding: finding, UnitIDs: ids[1:2], IntegratedSurface: true}
	reject := func(t *testing.T, id string) {
		t.Helper()
		o := options
		o.ReopenID = id
		if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, o); err == nil {
			t.Fatal("invalid authority authorized reopen")
		}
		if _, err := os.Stat(filepath.Join(root, "data/quality-check-reopens/zh-CN", id+".json")); !os.IsNotExist(err) {
			t.Fatal("failed check mutated reopen state", err)
		}
	}
	t.Run("zero-source-match", func(t *testing.T) {
		before := c.Adapted[1].Unit.SourcePath
		c.Adapted[1].Unit.SourcePath = "no/source.xml"
		defer func() { c.Adapted[1].Unit.SourcePath = before }()
		reject(t, "zero-map")
	})
	t.Run("multiple-source-matches", func(t *testing.T) {
		u := c.Adapted[1]
		u.Unit.ID = "ambiguous/other"
		u.StableIndex = 4
		c.Adapted = append(c.Adapted, u)
		defer func() { c.Adapted = c.Adapted[:3] }()
		reject(t, "multiple-map")
	})
	t.Run("current-source-sha-changed", func(t *testing.T) {
		before := c.Adapted[1].Unit.SourceSHA256
		c.Adapted[1].Unit.SourceSHA256 = strings.Repeat("a", 64)
		defer func() { c.Adapted[1].Unit.SourceSHA256 = before }()
		reject(t, "source-stale")
	})
	t.Run("unit-kind-changed", func(t *testing.T) {
		before := c.Adapted[1].Unit.Kind
		c.Adapted[1].Unit.Kind = i18n.UnitKindExample
		defer func() { c.Adapted[1].Unit.Kind = before }()
		reject(t, "kind-stale")
	})
	t.Run("candidate-sha-changed", func(t *testing.T) {
		path := artifact.Unit.CandidatePath
		before, _ := readRegular(root, path)
		writeFixture(t, root, path, append(bytes.Clone(before), []byte("changed")...))
		defer writeFixture(t, root, path, before)
		reject(t, "candidate-stale")
	})
	t.Run("validation-sha-changed", func(t *testing.T) {
		path := artifact.Unit.ValidationPath
		before, _ := readRegular(root, path)
		writeFixture(t, root, path, []byte("changed"))
		defer writeFixture(t, root, path, before)
		reject(t, "validation-stale")
	})
	t.Run("scope-runtime-drift", func(t *testing.T) {
		path := "production/identity.json"
		before, _ := readRegular(root, path)
		writeFixture(t, root, path, []byte("changed"))
		defer writeFixture(t, root, path, before)
		reject(t, "scope-stale")
	})
	t.Run("closure-drift", func(t *testing.T) {
		before, _ := readRegular(root, closure.Path)
		writeFixture(t, root, closure.Path, []byte("changed"))
		defer writeFixture(t, root, closure.Path, before)
		reject(t, "closure-stale")
	})
	t.Run("reviewer-artifact-drift", func(t *testing.T) {
		before, _ := readRegular(root, receipt.Evidence.Path)
		writeFixture(t, root, receipt.Evidence.Path, []byte("changed"))
		defer writeFixture(t, root, receipt.Evidence.Path, before)
		reject(t, "review-stale")
	})
	t.Run("receipt-identity-drift", func(t *testing.T) {
		before, _ := readRegular(root, ref.Path)
		bad := *receipt
		bad.Reviewer = "other-reviewer"
		b, _ := Encode(bad)
		writeFixture(t, root, ref.Path, b)
		defer writeFixture(t, root, ref.Path, before)
		reject(t, "receipt-stale")
	})
	t.Run("exact-affected-scope", func(t *testing.T) {
		before := options.UnitIDs
		options.UnitIDs = ids
		defer func() { options.UnitIDs = before }()
		reject(t, "expanded-scope")
	})
	t.Run("non-TU-language-is-blocker", func(t *testing.T) {
		r, _ := surfaceReceipt(t, root, c, build, []Reference{closure}, "non-tu-receipt", "failed", []SurfaceFinding{{ID: artifact.Unit.SourcePath, Finding: finding, Class: "navigation"}})
		before := options.SurfaceReviewID
		options.SurfaceReviewID = r.ReviewID
		defer func() { options.SurfaceReviewID = before }()
		reject(t, "non-tu-blocker")
		if _, err := requireIntegratedSurface(root, "zh-CN", closure, build); err == nil {
			t.Fatal("non-TU blocker yielded Surface PASS")
		}
	})
	t.Run("reviewed-byte-binding", func(t *testing.T) {
		scope, err := build(root, "zh-CN", []Reference{closure})
		if err != nil {
			t.Fatal(err)
		}
		badArtifact := artifact
		badArtifact.Unit.CandidateSHA256 = strings.Repeat("a", 64)
		r := *receipt
		r.Identity = ""
		r.ReviewID = "unreviewed-artifact"
		r.Scope = *scope
		r.ExactFindings = []SurfaceFinding{{ID: artifact.Unit.SourcePath, Finding: finding, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{badArtifact}}}
		b, _ := Encode(r)
		if _, _, err := recordIntegratedSurface(root, b, c, build); err == nil {
			t.Fatal("unreviewed candidate bytes entered receipt")
		}
	})
	t.Run("scope-drift-before-first-export", func(t *testing.T) {
		o := options
		o.ReopenID = "before-export"
		if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, o); err != nil {
			t.Fatal(err)
		}
		before := c.TranslationPackages[0].Units[0].Route
		c.TranslationPackages[0].Units[0].Route = "/fixture/changed-route"
		defer func() { c.TranslationPackages[0].Units[0].Route = before }()
		if _, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: ids[1:2], AllowReexport: true, PreviousSnapshotID: f.SnapshotID, SurfaceReopenID: o.ReopenID}); err == nil {
			t.Fatal("stale scope authorized first revision export")
		}
	})
}

func TestRegisteredPageDescriptorSurfaceSourceMapping(t *testing.T) {
	root := siteFixture(t)
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := AttachPageTranslationUnits(root, c)
	if err != nil || len(ids) != 49 {
		t.Fatal("current registered Learn/Docs inventory", err)
	}
	var g Global
	b, err := readRegular(root, GlobalPath)
	if err != nil || StrictJSON(b, &g) != nil {
		t.Fatal(err)
	}
	p, err := packageByID(&g, "learn-docs-v1")
	if err != nil || c.TranslationPackages[0].AuthoritySHA256 != p.Identity {
		t.Fatal("descriptor did not bind frozen Content Scope authority", err)
	}
	for _, id := range ids {
		u, err := c.Unit(id)
		if err != nil {
			t.Fatal(err)
		}
		mapped, err := c.ExactSourceTranslationUnit(u.SourcePath, u.Kind, u.SourceSHA256)
		if err != nil || mapped.ID != id {
			t.Fatal("registered Page source mapping", id, err)
		}
	}
}
