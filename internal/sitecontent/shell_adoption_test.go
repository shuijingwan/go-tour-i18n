package sitecontent

import (
	"bytes"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestShellReconstructionFrozenRepresentations(t *testing.T) {
	root := repositoryRoot(t)
	for _, p := range []string{"_content/index.md", "_content/site.tmpl", "_content/menus.yaml", "_content/resources.yaml", "_content/testimonials.yaml"} {
		t.Run(p, func(t *testing.T) {
			raw, err := readRegular(root, p)
			if err != nil {
				t.Fatal(err)
			}
			s, err := ExtractShellLanguageSource(p, raw)
			if err != nil {
				t.Fatal(err)
			}
			values := map[string]string{}
			for _, slot := range s.Slots {
				values[slot.Unit.ID] = slot.Unit.Source
			}
			b, err := reconstructShellSlots(*s, raw, values)
			if err != nil || !bytes.Equal(b, raw) {
				t.Fatal("source round trip", err)
			}
			if s.Page != nil {
				a := ShellPageAdapter{*s, raw}
				b, err := a.Restore(s.Page.Units[0].Source + "\n")
				if err != nil || !bytes.Equal(b, raw) {
					t.Fatal("shared Page restoration", err)
				}
				g := &i18n.Glossary{Mandatory: map[string]string{}}
				if err := a.Validate(b, g); err != nil {
					t.Fatal(err)
				}
			}
			for _, slot := range s.Slots {
				if len(slot.Unit.Protected) == 0 && slot.Encoding != "yaml-literal" {
					values[slot.Unit.ID] = "Synthetic localized visible text"
					break
				}
			}
			b, err = reconstructShellSlots(*s, raw, values)
			if err != nil || bytes.Equal(b, raw) {
				t.Fatal("visible delta", err)
			}
			if err := validateShellStructure(*s, raw, b); err != nil {
				t.Fatal(err)
			}
			bad := bytes.Replace(raw, []byte("https://"), []byte("http://"), 1)
			if !bytes.Equal(bad, raw) && validateShellStructure(*s, raw, bad) == nil {
				t.Fatal("machine URL drift accepted")
			}
			first := s.Slots[0].Unit.ID
			delete(values, first)
			if _, err := reconstructShellSlots(*s, raw, values); err == nil {
				t.Fatal("missing target accepted")
			}
			values[first] = "<script>unprotected</script>"
			if _, err := reconstructShellSlots(*s, raw, values); err == nil {
				t.Fatal("new markup accepted")
			}
			if _, err := parseStructuredRepresentation(Document{Contract: "unknown/v1"}, raw); err == nil {
				t.Fatal("unknown parser")
			}
		})
	}
}

func shellAdoptionFixture(t *testing.T, extraFrozenPaths ...string) (string, Reference, Reference, Reference) {
	t.Helper()
	// Bootstrap consumers run before sharedRepresentationFixture can copy its
	// later parser inputs. Install the current declared checker authority first.
	authority := append([]string{"current-authorization-fresh-fixture", "docs/RETRANSLATION_RUNBOOK.md", "docs/TRANSLATION_QUALITY_REVIEW.md", "docs/SITE_SHELL_SUCCESSOR_AUTHORITY.md"}, sharedReattestationValidationPaths...)
	root := siteFixture(t, authority...)
	for _, name := range []string{"internal/site/shell_render.go", "internal/site/tmplfunc/tmpl.go", "internal/site/tmplfunc/func.go"} {
		b, err := readRegular(repositoryRoot(t), name)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, b)
	}
	_, oldGeneration := importStructuredFixture(t, root)
	_, path, err := FinalizePackageClosure(root, "zh-CN", "site-v2-shell", nil, []Reference{oldGeneration})
	if err != nil {
		t.Fatal(err)
	}
	oldClosure, _ := ReferenceFile(root, path)
	scope, err := BuildIntegratedSurfaceScope(root, "zh-CN", []Reference{oldClosure})
	if err != nil {
		t.Fatal(err)
	}
	md := []byte("# Synthetic Surface\n\nPASS\n")
	writeFixture(t, root, "data/shell-adoption-surface.md", md)
	r := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: "shell-predecessor", Reviewer: "synthetic-independent-reviewer", Decision: "passed", Findings: []string{}, Evidence: Reference{"data/shell-adoption-surface.md", digest(md)}}
	raw, _ := Encode(r)
	_, surfacePath, err := RecordIntegratedSurface(root, raw)
	if err != nil {
		t.Fatal(err)
	}
	surface, _ := ReferenceFile(root, surfacePath)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := packageByID(g, "site-v2-shell")
	evidence := []Reference{oldClosure, surface}
	l := Locale{Schema: LocaleSchema, Locale: "zh-CN", Packages: []Completion{{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: routeFamilies(p.ID), EvidenceKind: PackageCompletionKind, Evidence: evidence, EvidenceIdentity: identity(evidence), ContextIdentity: scope.Closures[0].Path}}}
	var closure PackageClosure
	_ = readReference(root, oldClosure, &closure)
	l.Packages[0].ContextIdentity = closure.Identity
	raw, _ = Encode(l)
	writeFixture(t, root, LocalePath("zh-CN"), raw)
	// Frozen source metadata and registered legacy ZIP subset remain exact.
	real := repositoryRoot(t)
	manifest, err := loadSourceManifest(real)
	if err != nil {
		t.Fatal(err)
	}
	keep := map[string]bool{}
	for _, path := range manifest.Legacy.Paths {
		keep[adoptedSourcePath(path)] = true
	}
	for _, path := range []string{"_content/index.md", "_content/site.tmpl", "_content/menus.yaml", "_content/resources.yaml", "_content/testimonials.yaml"} {
		keep[path] = true
	}
	for _, name := range extraFrozenPaths {
		keep[name] = true
	}
	mini := *manifest
	mini.Files = []FrozenSourceFile{}
	for _, file := range manifest.Files {
		if !keep[file.Path] {
			continue
		}
		b, err := readAdoptedSource(real, file)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, file.Path, b)
		mini.Files = append(mini.Files, file)
	}
	sealSourceManifest(&mini)
	raw, _ = Encode(mini)
	writeFixture(t, root, SourceManifestPath, raw)
	a, err := BuildShellSuccessorAuthority(root)
	if err != nil {
		t.Fatal(err)
	}
	foundation, _ := shellAuthorityReference(a)
	if err := saveImmutable(root, foundation.Path, a); err != nil {
		t.Fatal(err)
	}
	corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	contributors := []i18n.LanguageContributor{}
	for _, c := range corpus.Contributors {
		if c.Path == a.HistoricalHome.Path {
			continue
		}
		c.Purpose = i18n.ContributorPurposeActiveContent
		c.SourceSHA = i18n.ContributorIdentity(c)
		contributors = append(contributors, c)
	}
	for _, source := range a.Sources {
		c, err := ProjectShellSuccessorContributor(source)
		if err != nil {
			t.Fatal(err)
		}
		contributors = append(contributors, c)
		corpus.Sources = append(corpus.Sources, i18n.GlossaryArchiveReference{Path: source.Path, SHA256: source.SourceSHA})
	}
	sort.Slice(contributors, func(i, j int) bool { return contributors[i].ID < contributors[j].ID })
	sourceRefs := map[string]i18n.GlossaryArchiveReference{}
	for _, ref := range corpus.Sources {
		sourceRefs[ref.Path] = ref
	}
	corpus.Sources = nil
	for _, ref := range sourceRefs {
		corpus.Sources = append(corpus.Sources, ref)
	}
	sort.Slice(corpus.Sources, func(i, j int) bool { return corpus.Sources[i].Path < corpus.Sources[j].Path })
	corpus.Contributors = contributors
	corpus.Schema = i18n.GlossaryCorpusSchemaV2
	corpus.Identity = i18n.CorpusIdentity(*corpus)
	raw, _ = Encode(corpus)
	writeFixture(t, root, i18n.GlossaryCorpusPath, raw)
	prior, _ := ReferenceFile(root, GlobalPath)
	if _, err := AdoptShellSuccessor(root, foundation, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong CAS adopted")
	}
	after, err := AdoptShellSuccessor(root, foundation, prior.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	adoptionRef, _ := ReferenceFile(root, shellAdoptionPath(after))
	current, err := BuildUnifiedCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = Encode(current)
	writeFixture(t, root, i18n.GlossaryCorpusPath, raw)
	bundle, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, bundle, "shell-current-review", "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	return root, oldClosure, oldGeneration, adoptionRef
}

func TestShellCurrentAdoptionSharedLifecycle(t *testing.T) {
	root, oldClosure, oldGen, adoption := shellAdoptionFixture(t)
	if _, _, err := CheckPackageClosure(root, oldClosure); err != nil {
		t.Fatal("historical home closure", err)
	}
	if _, err := BuildIntegratedSurfaceScope(root, "zh-CN", []Reference{oldClosure}); err == nil {
		t.Fatal("historical home became current Surface")
	}
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, _, err := CurrentPackageDocuments(root, g, "site-v2-shell")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 6 {
		t.Fatal("current atomic inventory", len(docs))
	}
	pages := 0
	for _, d := range docs {
		if d.Path == "data/site-shell/home.html" {
			t.Fatal("historical source current")
		}
		if d.UnitScope == PageContract {
			pages++
			if d.Path != "_content/index.md" || len(d.Units) != 1 || d.Route != "/" {
				t.Fatal(d)
			}
		}
	}
	if pages != 1 {
		t.Fatal(pages)
	}
	if _, _, err := BuildPackageClosure(root, "zh-CN", "site-v2-shell", nil, []Reference{oldGen}); err == nil {
		t.Fatal("missing Homepage finalization")
	}
	plan, err := PlanStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	newAssets := 0
	for _, a := range plan.Assets {
		if a.Document != nil && a.Document.Contract == ShellExtractionContract {
			newAssets++
			if a.State != "generation-required" {
				t.Fatal(a)
			}
		}
		if a.ID == "source:data/site-shell/translation.html" && a.State != "carry" {
			t.Fatal("translation not carried")
		}
	}
	if newAssets != 4 {
		t.Fatal(newAssets)
	}
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	AttachIntegratedSurfaceAuthority(c)
	if err := c.UseTranslationPackage("site-v2-shell"); err != nil {
		t.Fatal(err)
	}
	descriptor := c.TranslationPackages[len(c.TranslationPackages)-1]
	if len(descriptor.Units) != 1 {
		t.Fatal("Page slots became TUs")
	}
	id := descriptor.Units[0].Unit.ID
	e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorCodex, UnitIDs: []string{id}})
	if err != nil {
		t.Fatal(err)
	}
	// Test-only synthetic raw response flows through the real shared lifecycle.
	batchPath := e.BatchPath
	adapter := descriptor.Units[0].Adapter
	glossary, _ := i18n.LoadGlossary(root, "zh-CN")
	input, _, err := adapter.ProtectedInput(glossary)
	if err != nil {
		t.Fatal(err)
	}
	batchBundle, m, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	transport, err := i18n.ReadTransportBundle(batchBundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"context/" + ShellAdoptionCurrent, "context/" + adoption.Path, "context/docs/SITE_SHELL_SUCCESSOR_AUTHORITY.md", "context/_content/index.md", "context/" + i18n.GlossaryCorpusPath} {
		if len(transport[name]) == 0 {
			t.Fatal("Homepage bundle lacks current frozen authority", name)
		}
	}
	if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, batchBundle); err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	name := m.ExpectedOutputs[0].BundlePath
	name = name[strings.LastIndex(name, "/")+1:]
	if err := os.WriteFile(outDir+"/"+name, []byte(input+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.ImportGenerationOutputDirectory(root, c, batchBundle, "codex", "gpt-5.6-sol-high", outDir); err != nil {
		t.Fatal(err)
	}
	r, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil || r.ValidationPassed != 1 {
		t.Fatal(batchPath, r, err)
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "homepage-synthetic"}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "homepage-synthetic", ReviewerSession: "synthetic-independent-reviewer", UnitIDs: []string{id}, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	_, finalPath, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "homepage-synthetic"})
	if err != nil {
		t.Fatal(err)
	}
	page, _ := ReferenceFile(root, finalPath)
	if _, _, err := i18n.EstablishCurrentTranslationAuthorization(root, c, "zh-CN", "homepage-synthetic"); err != nil {
		t.Fatal(err)
	}
	ready, err := PlanHomepage(root, "zh-CN")
	if err != nil || ready.Units[0].State != "carry" {
		t.Fatal("A-only Homepage carry", ready, err)
	}
	_, newGen := importStructuredFixture(t, root)
	closure, files, err := BuildPackageClosure(root, "zh-CN", "site-v2-shell", &page, []Reference{oldGen, newGen})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 6 || len(closure.Files) != 6 {
		t.Fatal("incomplete closure")
	}
	if _, ok := files["data/site-shell/home.html"]; ok {
		t.Fatal("historical home projected")
	}
	_, closurePath, err := FinalizePackageClosure(root, "zh-CN", "site-v2-shell", &page, []Reference{oldGen, newGen})
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := ReferenceFile(root, closurePath)
	scope, err := BuildIntegratedSurfaceScope(root, "zh-CN", []Reference{ref})
	if err != nil {
		t.Fatal(err)
	}
	carriedTranslation := false
	for _, a := range scope.Carried {
		if a.ID == "data/site-shell/translation.html" {
			carriedTranslation = true
		}
	}
	if !carriedTranslation {
		t.Fatal("translation re-reviewed")
	}
	if len(scope.Reviewed) != 5 {
		t.Fatal("Homepage context + four structured", len(scope.Reviewed))
	}
	for _, a := range scope.Reviewed {
		if a.ID == "_content/index.md" && (a.State != "context-review-required" || len(a.CandidateArtifacts) != 1) {
			t.Fatal("Homepage context must reuse shared QC", a)
		}
	}
	// Each missing member of the four-source generation is rejected by the
	// same import exact-set gate, without accepting the other outputs partially.
	var generation StructuredGeneration
	_ = readReference(root, newGen, &generation)
	for _, p := range []string{"_content/site.tmpl", "_content/menus.yaml", "_content/resources.yaml", "_content/testimonials.yaml"} {
		bad := generation.Output
		bad.Outputs = []StructuredTarget{}
		for _, o := range generation.Output.Outputs {
			if o.Path != "locales/zh-CN/structured/"+p {
				bad.Outputs = append(bad.Outputs, o)
			}
		}
		if _, err := validateStructuredOutput(root, generation.Plan, bad); err == nil {
			t.Fatal("missing structured source accepted", p)
		}
	}
	if _, _, err := BuildPackageClosure(root, "zh-CN", "site-v2-shell", &page, []Reference{oldGen}); err == nil {
		t.Fatal("missing structured sources")
	}
	for _, tc := range []struct{ name, path string }{{"adoption", adoption.Path}, {"translation-source", "data/site-shell/translation.html"}, {"translation-target", oldGen.Path}, {"Homepage-source", "_content/index.md"}} {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := readRegular(root, tc.path)
			writeFixture(t, root, tc.path, append(bytes.Clone(b), []byte("tamper")...))
			defer writeFixture(t, root, tc.path, b)
			if _, _, err := CheckPackageClosure(root, ref); err == nil {
				t.Fatal("tamper accepted")
			}
		})
	}
	// Source governance is global; a second locale registers the same one TU.
	identityBytes, _ := readRegular(root, "locales/zh-CN/locale.json")
	identityBytes = bytes.ReplaceAll(identityBytes, []byte("zh-CN"), []byte("fr-FR"))
	writeFixture(t, root, "locales/fr-FR/locale.json", identityBytes)
	writeFixture(t, root, "locales/fr-FR/glossary.yaml", []byte("locale: fr-FR\nmandatory:\n  synthetic-test-term: terme\n"))
	ub, _, err := i18n.ExportUnifiedGlossaryBundle(root, "fr-FR", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, ub, "french-synthetic", "independent-french-reviewer", "french-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	hp, err := PlanHomepage(root, "fr-FR")
	if err != nil || hp.PageCount != 1 || hp.Units[0].ID != id || hp.Units[0].State != "generation-required" {
		t.Fatal("second locale", hp, err)
	}
	t.Log("historical/current adoption/one shared TU/A-only/current authorization/atomic six-source closure/Surface/second locale PASS")
}
