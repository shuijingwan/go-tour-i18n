package sitecontent

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// All authority publications in this regression belong to a disposable repo.
// The ordinary source delta predates the finding; the template's accepted owner
// is inherited by the reviewed closure, exactly as in a continuing campaign.
func TestPackageClosureAffectedRepresentationSurface(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, baseline, accepted, page := sharedRepresentationFixture(t)
	unrelated := multiPackageLearnClosureFixture(t, root, accepted)
	land := func(path string, value any) Reference { return landSharedReattestationFixture(t, root, path, value) }
	provenance := Provenance{Provider: "codex", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-31T00:00:00Z"}
	importOutput := func(zip []byte, m *StructuredBundle) Reference {
		t.Helper()
		o := StructuredOutput{Schema: StructuredSchema, BundleSHA: m.Identity, Outputs: []StructuredTarget{}}
		for _, path := range m.Plan.Expected {
			for _, a := range m.Plan.Assets {
				if a.TargetPath != path {
					continue
				}
				target := StructuredTarget{Path: path, Slots: []Target{}}
				carry := map[string]bool{}
				for _, v := range a.CarriedSlots {
					carry[v.ID] = true
				}
				for _, u := range a.Document.Units {
					if !carry[u.ID] {
						target.Slots = append(target.Slots, Target{u.ID, u.Source})
					}
				}
				o.Outputs = append(o.Outputs, target)
			}
		}
		b, _ := Encode(o)
		_, path, err := ImportStructuredAssets(root, zip, b, provenance)
		if err != nil {
			t.Fatal("fixture import", err)
		}
		ref, err := ReferenceFile(root, path)
		if err != nil {
			t.Fatal(err)
		}
		return ref
	}
	const translation = "data/site-shell/translation.html"
	oldRaw := mustSharedFixtureBytes(t, root, translation)
	at := bytes.LastIndex(oldRaw, []byte("</main>"))
	if at < 0 {
		t.Fatal("missing source insertion point")
	}
	body := append(bytes.Clone(oldRaw[:at]), []byte("<section><h2>Synthetic addition</h2><p>Synthetic new source text.</p></section>\n")...)
	body = append(body, oldRaw[at:]...)
	writeFixture(t, root, "data/closure-expanded-source.html", body)
	proposal := land("data/closure-source-proposal.json", StructuredSourceProposal{Schema: StructuredSourceSuccessorSchema, Package: "site-v2-shell", Previous: Reference{translation, digest(oldRaw)}, Body: Reference{"data/closure-expanded-source.html", digest(body)}, InsertAt: at})
	source, err := BuildStructuredSourceSuccessor(root, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptStructuredSourceSuccessor(root, proposal, source.BaseIdentity); err != nil {
		t.Fatal(err)
	}
	corpus := mustSharedFixtureBytes(t, root, i18n.GlossaryCorpusPath)
	if _, err := AdoptCorpusProjection(root, digest(corpus), true); err != nil {
		t.Fatal(err)
	}
	reviewBundle, _, err := i18n.ExportUnifiedGlossaryBundle(root, sharedFixtureLocale, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, reviewBundle, "closure-source-review", "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	// Adopt the Homepage representation only after the source successor, so
	// its source basis is current. A fresh synthetic Page target makes shared
	// composition a mandatory judgment input on the intermediate scope.
	home, err := PlanShellRepresentationSuccessor(root, ShellTeachingRepresentation)
	if err != nil {
		t.Fatal(err)
	}
	homePlan := land("data/closure-homepage-plan.json", home)
	if _, err := ApplyShellRepresentationSuccessor(root, homePlan, home.PriorIdentity, home.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	cat, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackagesReadonly(root, cat); err != nil {
		t.Fatal(err)
	}
	if err := cat.UseTranslationPackage("site-v2-shell"); err != nil {
		t.Fatal(err)
	}
	batch, err := i18n.ExportRetranslationBatch(root, cat, i18n.RetranslationExportOptions{Locale: sharedFixtureLocale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: cat.TranslationScope})
	if err != nil {
		t.Fatal(err)
	}
	homeZip, hm, err := i18n.ExportTranslationUnitGenerationBundle(root, cat, i18n.GenerationBundleOptions{Locale: sharedFixtureLocale, BatchID: batch.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	homeFiles, err := i18n.ReadTransportBundle(homeZip, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	homeOutputs := map[string][]byte{}
	for _, output := range hm.ExpectedOutputs {
		value := homeFiles["formal/inputs/"+filepath.Base(output.BundlePath)]
		changed := bytes.Replace(value, []byte("Easy to learn and great for teams"), []byte("Synthetic accessible learning for teams"), 1)
		if bytes.Equal(value, changed) {
			t.Fatal("missing synthetic Page coordinate")
		}
		homeOutputs[output.UnitID] = changed
	}
	surfaceImport(t, root, cat, homeZip, homeOutputs)
	if result, err := i18n.ProcessRetranslationBatch(root, cat, i18n.RetranslationProcessOptions{Locale: sharedFixtureLocale, BatchID: batch.BatchID}); err != nil || result.ValidationPassed != len(cat.TranslationScope) {
		t.Fatal(result, err)
	}
	const snapshot = "closure-current-page"
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, cat, i18n.QualityCheckSnapshotOptions{Locale: sharedFixtureLocale, SnapshotID: snapshot, ContentPackage: "site-v2-shell"}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, cat, i18n.QualityCheckRecordOptions{Locale: sharedFixtureLocale, SnapshotID: snapshot, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: cat.TranslationScope, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(root, cat, i18n.QualityCheckFinalizeOptions{Locale: sharedFixtureLocale, SnapshotID: snapshot}); err != nil {
		t.Fatal(err)
	}
	pageRef, err := ReferenceFile(root, "data/quality-check-snapshots/"+sharedFixtureLocale+"/"+snapshot+"/finalization.json")
	if err != nil {
		t.Fatal(err)
	}
	page = &pageRef
	t.Log("current Page authority and ordinary source successor constructed")
	zip, manifest, err := ExportStructuredAssets(root, sharedFixtureLocale)
	if err != nil {
		t.Fatal(err)
	}
	ordinary := importOutput(zip, manifest)
	_, middlePath, err := FinalizePackageClosure(root, sharedFixtureLocale, "site-v2-shell", page, []Reference{ordinary})
	if err != nil {
		t.Fatal("ordinary source delta closure", err)
	}
	middle, _ := ReferenceFile(root, middlePath)
	t.Log("ordinary source delta closure PASS")
	scope, err := BuildIntegratedSurfaceScope(root, sharedFixtureLocale, []Reference{unrelated, middle})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(scope.Closures, []Reference{unrelated, middle}) {
		t.Fatal("multi-package scope must retain unrelated package first")
	}
	md := []byte("# Synthetic independent shared language finding\nFrozen natural-language action.\n")
	writeFixture(t, root, "data/closure-shared-finding.md", md)
	r := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: "closure-shared-finding", Reviewer: "synthetic-independent-reviewer", Decision: "failed", Findings: []string{"frozen action"}, ExactFindings: []SurfaceFinding{{ID: "_content/site.tmpl", Class: "structured-surface-language", Finding: "frozen action"}}, Evidence: Reference{"data/closure-shared-finding.md", digest(md)}}
	b, _ := Encode(r)
	_, findingPath, err := RecordIntegratedSurface(root, b)
	if err != nil {
		t.Fatal(err)
	}
	finding, _ := ReferenceFile(root, findingPath)
	frozenFinding := mustSharedFixtureBytes(t, root, findingPath)
	t.Log("independent failed shared composition finding recorded")
	t.Run("compatible-Surface-supersession", func(t *testing.T) {
		// Before representation adoption, this exact finding replaces a
		// compatible inherited owner through the existing merge branch.
		zip, m, err := ExportStructuredAssetsRevision(root, sharedFixtureLocale, finding)
		if err != nil {
			t.Fatal(err)
		}
		compatible := false
		for _, a := range m.Plan.Assets {
			if a.Document != nil && a.Document.Path == "_content/site.tmpl" {
				compatible = a.Assessment != nil && a.Assessment.Classification == "compatible" && a.Assessment.RepresentationSuccessor == nil
			}
		}
		if !compatible {
			t.Fatal("fixture must reach compatible merge branch")
		}
		replacement := importOutput(zip, m)
		refs := []Reference{ordinary, replacement}
		c, _, err := BuildPackageClosure(root, sharedFixtureLocale, "site-v2-shell", page, refs)
		if err != nil || c.StructuredBaseline == nil || *c.StructuredBaseline != baseline || !reflect.DeepEqual(c.Structured, refs) {
			t.Fatal("compatible exact Surface replacement changed", err)
		}
	})
	raw := mustSharedFixtureBytes(t, root, "_content/site.tmpl")
	old, err := ExtractShellLanguageSource("_content/site.tmpl", raw)
	if err != nil {
		t.Fatal(err)
	}
	request := land("data/closure-representation-request.json", SharedRepresentationRequest{Schema: SharedRepresentationSchema + "/request", Package: "site-v2-shell", AssetID: "source:_content/site.tmpl", Path: "_content/site.tmpl", SourceSHA: digest(raw), PreviousIdentity: identity(*old), Generation: accepted, Rule: SharedRepresentationRule, Slots: []string{"0719e14f5f889a9c0dccb5eb291c9fe1de9604b6f5423b3c1497204c4276a85d"}})
	p, err := PlanSharedStructuredRepresentation(root, request)
	if err != nil {
		t.Fatal(err)
	}
	plan := land("data/closure-representation-plan.json", p)
	if _, err := ApplySharedStructuredRepresentation(root, plan, p.PriorIdentity, p.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	parser := sharedRepresentationParserPaths[0]
	writeFixture(t, root, parser, append(mustSharedFixtureBytes(t, root, parser), []byte("\n// audited fixture dependency epoch\n")...))
	rea, err := PlanSharedRepresentationReattestation(root, p.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	reviewed := land("data/closure-reattestation-plan.json", rea)
	if _, err := ApplySharedRepresentationReattestation(root, reviewed, rea.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	t.Log("representation successor and dependency authority current")
	zip, manifest, err = ExportStructuredAssetsRevision(root, sharedFixtureLocale, finding)
	if err != nil {
		t.Fatal("full finding export", err)
	}
	revision := importOutput(zip, manifest)
	t.Log("exact full finding revision current import PASS")
	var generation StructuredGeneration
	if err := readReference(root, revision, &generation); err != nil {
		t.Fatal(err)
	}
	// Re-seal adversarial synthetic transports so negative checks cannot pass
	// merely because their outer Generation or Bundle hash was left stale.
	repackage := func(ref Reference, change func(*StructuredBundle, map[string][]byte)) Reference {
		t.Helper()
		var g StructuredGeneration
		if err := readReference(root, ref, &g); err != nil {
			t.Fatal(err)
		}
		archive := rewriteRecoveryBundle(t, mustSharedFixtureBytes(t, root, g.Bundle.Path), change)
		m, _, err := readStructuredBundle(archive)
		if err != nil {
			t.Fatal(err)
		}
		g.Bundle = Reference{"data/locale-language/" + sharedFixtureLocale + "/structured-bundles/" + digest(archive) + ".zip", digest(archive)}
		writeFixture(t, root, g.Bundle.Path, archive)
		g.Plan, g.Output.BundleSHA = m.Plan, m.Identity
		selected := []StructuredTarget{}
		for _, path := range m.Plan.Expected {
			for _, output := range g.Output.Outputs {
				if output.Path == path {
					selected = append(selected, output)
				}
			}
		}
		g.Output.Outputs = selected
		g.Identity = ""
		g.Identity = identity(g)
		return land("data/locale-language/"+sharedFixtureLocale+"/structured-generation/"+g.Identity+".json", g)
	}
	if !structuredSupersedes(root, &generation, accepted, "_content/site.tmpl") {
		t.Fatal("exact inherited parent not authorized")
	}
	for _, path := range []string{"_content/menus.yaml", translation} {
		if structuredSupersedes(root, &generation, accepted, path) {
			t.Fatal("wrong finding path accepted", path)
		}
	}
	var sibling StructuredGeneration
	if err := readReference(root, accepted, &sibling); err != nil {
		t.Fatal(err)
	}
	sibling.Provenance.GeneratedAt = "2026-10-31T01:00:00Z"
	sibling.Identity = ""
	sibling.Identity = identity(sibling)
	siblingRef := land("data/locale-language/"+sharedFixtureLocale+"/structured-generation/"+sibling.Identity+".json", sibling)
	for _, parent := range []Reference{siblingRef, middle} {
		if structuredSupersedes(root, &generation, parent, "_content/site.tmpl") {
			t.Fatal("wrong/sibling parent accepted", parent)
		}
	}
	refs := []Reference{ordinary, revision}
	t.Run("duplicate-distinct-current-replacements", func(t *testing.T) {
		duplicate := generation
		duplicate.Provenance.GeneratedAt = "2026-10-31T02:00:00Z"
		duplicate.Identity = ""
		duplicate.Identity = identity(duplicate)
		ref := land("data/locale-language/"+sharedFixtureLocale+"/structured-generation/"+duplicate.Identity+".json", duplicate)
		if _, _, err := CheckStructuredGeneration(root, ref); err != nil {
			t.Fatal("duplicate must be independently current-valid", err)
		}
		if _, _, err := BuildPackageClosure(root, sharedFixtureLocale, "site-v2-shell", page, append(append([]Reference{}, refs...), ref)); err == nil || !strings.Contains(err.Error(), "missing/duplicate affected delta: _content/site.tmpl") {
			t.Fatal("two distinct current replacements accepted", err)
		}
	})
	c, path, err := FinalizePackageClosure(root, sharedFixtureLocale, "site-v2-shell", page, refs)
	if err != nil {
		t.Fatal("two-delta closure", err)
	}
	if c.StructuredBaseline == nil || *c.StructuredBaseline != baseline || !reflect.DeepEqual(c.Structured, refs) {
		t.Fatal("wrong exact predecessor/delta set", c)
	}
	closure, _ := ReferenceFile(root, path)
	_, files, err := CheckPackageClosure(root, closure)
	if err != nil {
		t.Fatal("current downstream closure", err)
	}
	history, err := historicalStructuredClosure(root, baseline, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	for path, before := range history.Files {
		// The separately constructed Page authority intentionally differs from
		// the predecessor, to keep mandatory shared composition review current.
		if path != translation && path != "_content/site.tmpl" && path != "_content/index.md" && !bytes.Equal(files[path], before) {
			t.Fatal("unaffected output changed", path)
		}
	}
	t.Run("ordinary-affected-Surface-forbidden", func(t *testing.T) {
		bad := repackage(ordinary, func(m *StructuredBundle, _ map[string][]byte) { m.RevisionFinding = &finding })
		if _, _, _, err := historicalStructuredGeneration(root, bad); err != nil {
			t.Fatal("negative must have valid frozen content", err)
		}
		if _, _, err := BuildPackageClosure(root, sharedFixtureLocale, "site-v2-shell", page, []Reference{bad, revision}); err == nil || !strings.Contains(err.Error(), "glossary delta cannot use Surface finding authority") {
			t.Fatal("ordinary affected state acquired Surface exception", err)
		}
	})
	for _, name := range []string{"sealed-stale-scope", "sealed-wrong-class", "sealed-wrong-path", "sealed-reviewer", "sealed-sibling-parent"} {
		t.Run(name, func(t *testing.T) {
			var bad IntegratedSurfaceReceipt
			_ = StrictJSON(frozenFinding, &bad)
			switch name {
			case "sealed-stale-scope":
				bad.Scope.Closures[0].SHA256 = strings.Repeat("a", 64)
			case "sealed-wrong-class":
				bad.ExactFindings[0].Class = "surface-integration"
			case "sealed-wrong-path":
				bad.ExactFindings[0].ID = "_content/menus.yaml"
			case "sealed-reviewer":
				bad.Reviewer = "synthetic-generation"
			case "sealed-sibling-parent":
				other := history.Closure
				other.Structured = []Reference{siblingRef}
				other.Identity = ""
				other.Identity = identity(other)
				ref := land(workflowPath(other.Locale, other.Package, "closure", other.Identity), other)
				for i := range bad.Scope.Closures {
					if bad.Scope.Closures[i] == middle {
						bad.Scope.Closures[i] = ref
					}
				}
			}
			bad.Scope.Identity = ""
			bad.Scope.Identity = identity(bad.Scope)
			bad.Identity = ""
			bad.Identity = identity(bad)
			badFinding := land(findingPath, bad)
			defer writeFixture(t, root, findingPath, frozenFinding)
			badGen := repackage(revision, func(m *StructuredBundle, fs map[string][]byte) {
				m.RevisionFinding = &badFinding
				fs["inputs/"+findingPath] = mustSharedFixtureBytes(t, root, findingPath)
			})
			if _, _, _, err := historicalStructuredGeneration(root, badGen); err != nil {
				t.Fatal("negative frozen content rejected before finding check", err)
			}
			if _, _, err := BuildPackageClosure(root, sharedFixtureLocale, "site-v2-shell", page, []Reference{ordinary, badGen}); err == nil {
				t.Fatal("sealed invalid finding accepted")
			}
		})
	}
	_, ordinaryManifest, _, err := readStructuredGeneration(root, ordinary)
	if err != nil || ordinaryManifest.RevisionFinding != nil {
		t.Fatal("ordinary delta gained finding authority", err)
	}
	for _, a := range ordinaryManifest.Plan.Assets {
		if a.Document != nil && a.Document.Path == translation && (a.Assessment == nil || a.Assessment.SourceSuccessor == nil || a.Assessment.RepresentationSuccessor != nil) {
			t.Fatal("ordinary source assessment changed")
		}
	}
	t.Run("unconsumed-extra-generation", func(t *testing.T) {
		extra := repackage(accepted, func(m *StructuredBundle, _ map[string][]byte) {
			m.Plan.Expected = nil
			for i := range m.Plan.Assets {
				a := &m.Plan.Assets[i]
				a.State = "context-only"
				if a.ID == "tour-ui" {
					a.State = "generation-required"
					m.Plan.Expected = []string{a.TargetPath}
				}
			}
		})
		if _, _, _, err := historicalStructuredGeneration(root, extra); err != nil {
			t.Fatal("extra must be valid historical Generation", err)
		}
		if _, _, err := BuildPackageClosure(root, sharedFixtureLocale, "site-v2-shell", page, append(append([]Reference{}, refs...), extra)); err == nil || !strings.Contains(err.Error(), "unauthorized extra structured generation") {
			t.Fatal("unconsumed Generation accepted", err)
		}
	})

	// Mutate one authority at a time; current closure validation must reject
	// even when structuredSupersedes alone would still accept the parent edge.
	for _, name := range []string{"identity", "scope", "reviewer", "evidence", "class", "finding-path", "glossary", "review", "stale-generation", "duplicate"} {
		t.Run(name, func(t *testing.T) {
			badRefs := append([]Reference{}, refs...)
			file := findingPath
			value := []byte("{}\n")
			switch name {
			case "identity", "scope", "reviewer", "class", "finding-path":
				var bad IntegratedSurfaceReceipt
				_ = StrictJSON(frozenFinding, &bad)
				switch name {
				case "identity":
					bad.Identity = strings.Repeat("a", 64)
				case "scope":
					bad.Scope.Routes = append(bad.Scope.Routes, "/unrelated/")
				case "reviewer":
					bad.Reviewer = "synthetic-generation"
				case "class":
					bad.ExactFindings[0].Class = "surface-integration"
				case "finding-path":
					bad.ExactFindings[0].ID = "_content/menus.yaml"
				}
				value, _ = Encode(bad)
			case "evidence":
				file = r.Evidence.Path
			case "glossary":
				file = "locales/" + sharedFixtureLocale + "/glossary.yaml"
			case "review":
				file = generation.Plan.Review.Path
			case "stale-generation":
				file = parser
				if !structuredSupersedes(root, &generation, accepted, "_content/site.tmpl") {
					t.Fatal("test must reach current validation with exact supersession")
				}
			case "duplicate":
				badRefs = append(badRefs, revision)
			}
			if name != "duplicate" {
				before := mustSharedFixtureBytes(t, root, file)
				defer writeFixture(t, root, file, before)
				writeFixture(t, root, file, value)
			}
			if _, _, err := BuildPackageClosure(root, sharedFixtureLocale, "site-v2-shell", page, badRefs); err == nil {
				t.Fatal("invalid replacement accepted")
			}
		})
	}
	t.Log("TWO_DELTA_CLOSURE=PASS ordinary source authority preserved; exact Surface inherited owner consumed; unaffected files exact; historical/current closure PASS")
}

// Build a real, current Learn/Docs closure and an unchanged successor whose
// structured owners are inherited. Every publication stays in the test repo.
func multiPackageLearnClosureFixture(t *testing.T, root string, generation Reference) Reference {
	t.Helper()
	cat, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackagesReadonly(root, cat); err != nil {
		t.Fatal(err)
	}
	if err := cat.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	if len(cat.TranslationScope) <= 30 || len(cat.TranslationScope) > 60 {
		t.Fatal("expected two complete Learn/Docs fixture batches", len(cat.TranslationScope))
	}
	sets := [][]string{cat.TranslationScope[:30], cat.TranslationScope[30:]}
	for _, ids := range sets {
		batch, _ := importSyntheticSharedPagesForLocale(t, root, cat, sharedFixtureLocale, ids)
		if r, err := i18n.ProcessRetranslationBatch(root, cat, i18n.RetranslationProcessOptions{Locale: sharedFixtureLocale, BatchID: batch}); err != nil || r.ValidationPassed != len(ids) {
			t.Fatal("unrelated package Page fixture", r, err)
		}
	}
	const snapshot = "multi-package-learn-pages"
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, cat, i18n.QualityCheckSnapshotOptions{Locale: sharedFixtureLocale, SnapshotID: snapshot, ContentPackage: "learn-docs-v1"}); err != nil {
		t.Fatal(err)
	}
	for _, ids := range sets {
		if _, err := i18n.RecordQualityCheckResults(root, cat, i18n.QualityCheckRecordOptions{Locale: sharedFixtureLocale, SnapshotID: snapshot, ReviewerSession: "synthetic-independent-reviewer", UnitIDs: ids, Rating: "A"}); err != nil {
			t.Fatal(err)
		}
	}
	_, finalPath, err := i18n.FinalizeQualityCheck(root, cat, i18n.QualityCheckFinalizeOptions{Locale: sharedFixtureLocale, SnapshotID: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	page, err := ReferenceFile(root, finalPath)
	if err != nil {
		t.Fatal(err)
	}
	_, path, err := FinalizePackageClosure(root, sharedFixtureLocale, "learn-docs-v1", &page, []Reference{generation})
	if err != nil {
		t.Fatal("unrelated baseline closure", err)
	}
	baseline, err := ReferenceFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	var child PackageClosure
	if err := readReference(root, baseline, &child); err != nil {
		t.Fatal(err)
	}
	child.Structured, child.StructuredBaseline = []Reference{}, &baseline
	child.Identity = ""
	child.Identity = identity(child)
	ref := landSharedReattestationFixture(t, root, workflowPath(child.Locale, child.Package, "closure", child.Identity), child)
	if _, _, err := CheckPackageClosure(root, ref); err != nil {
		t.Fatal("unrelated inherited closure must be current-valid", err)
	}
	t.Log("current Learn/Docs closure with inherited structured baseline constructed")
	return ref
}
