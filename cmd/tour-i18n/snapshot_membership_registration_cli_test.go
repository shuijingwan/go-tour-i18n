package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
)

// Only frozen English source is shared with the real repository. Adoption,
// membership, locale, generation and Reviewer evidence are new TempDir fixtures.
func snapshotMembershipRunFixture(t *testing.T) (string, *i18n.Catalog) {
	t.Helper()
	root, _ := generationRunSourceFixture(t)
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// Adoption enables the canonical adopted-source reader. Materialize only
	// its frozen English legacy/source inputs, then refresh the native lock.
	var manifest sitecontent.FrozenSourceManifest
	if err := json.Unmarshal(courseContextRead(t, repo, sitecontent.SourceManifestPath), &manifest); err != nil {
		t.Fatal(err)
	}
	keep := map[string]bool{}
	for _, name := range manifest.Legacy.Paths {
		path := name
		if !strings.HasPrefix(path, "_content/") {
			path = sourcestorage.ContractRoot + path
		}
		keep[path] = true
	}
	for _, f := range manifest.Files {
		if !keep[f.Path] {
			continue
		}
		data, err := sourcestorage.ReadAdoptedSource(repo, f)
		if err != nil {
			t.Fatal(err)
		}
		courseContextWrite(t, root, f.Path, data)
	}
	// Adopted docs-entry protection binds the English course descriptions to
	// the complete native source catalog, not the foundation's tiny Tour.
	if err := filepath.WalkDir(filepath.Join(repo, "_content/tour"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(repo, path)
			if err != nil {
				return err
			}
			courseContextWrite(t, root, filepath.ToSlash(rel), courseContextRead(t, repo, filepath.ToSlash(rel)))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	native, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	pages, conditional, err := i18n.CatalogBytes(native)
	if err != nil {
		t.Fatal(err)
	}
	examples, err := i18n.ExampleCatalogBytes(native)
	if err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string][]byte{"data/tour-pages.tsv": pages, "data/tour-conditional-pages.tsv": conditional, "data/tour-examples.tsv": examples} {
		courseContextWrite(t, root, path, data)
	}
	for _, path := range []string{sitecontent.SourceManifestPath, "internal/tour/ui/en.json", "data/course-seo/source-descriptions.json", "_content/doc/index.html", "_content/doc/install.html", "_content/site.tmpl", "_content/article.tmpl", "_content/doc/default.tmpl"} {
		courseContextWrite(t, root, path, courseContextRead(t, repo, path))
	}
	snapshot, err := sitecontent.ReadSourceSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	global, err := sitecontent.BuildGlobal(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	courseContextWrite(t, root, sitecontent.GlobalPath, courseContextJSON(t, global))
	proposal := sitecontent.PagePackageProposal{Schema: sitecontent.PagePackageAdoptionSchema, Package: "docs-entry-v1", Surface: "docs-entry"}
	for _, path := range []string{"_content/doc/index.html", "_content/doc/install.html"} {
		proposal.Sources = append(proposal.Sources, sitecontent.Reference{Path: path, SHA256: courseContextSHA(courseContextRead(t, root, path))})
	}
	proposalPath := "engineering/docs-entry-proposal.json"
	data := courseContextJSON(t, proposal)
	courseContextWrite(t, root, proposalPath, data)
	ref := sitecontent.Reference{Path: proposalPath, SHA256: courseContextSHA(data)}
	adoption, err := sitecontent.BuildPagePackageAdoption(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sitecontent.AdoptPagePackage(root, ref, adoption.BaseIdentity); err != nil {
		t.Fatal(err)
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := sitecontent.AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	var old, sibling i18n.TranslationPackageDescriptor
	for _, d := range c.TranslationPackages {
		if d.ID == "learn-docs-v1" {
			old = d
		}
		if d.ID == "docs-entry-v1" {
			sibling = d
		}
	}
	if len(old.Units) != 49 || len(sibling.Units) != 2 {
		t.Fatal("source authorities must be 49+2")
	}
	seal := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return courseContextSHA(b)
	}
	registryBytes := courseContextRead(t, root, sitecontent.PageRegistryPath)
	var registry sitecontent.PageRegistry
	if err := json.Unmarshal(registryBytes, &registry); err != nil {
		t.Fatal(err)
	}
	previous := "engineering/previous-page-registry.json"
	courseContextWrite(t, root, previous, registryBytes)
	m := sitecontent.PackageMembership{Schema: sitecontent.PackageMembershipSchema, Package: old.ID, PreviousRegistry: sitecontent.Reference{Path: previous, SHA256: courseContextSHA(registryBytes)}}
	for _, d := range []i18n.TranslationPackageDescriptor{old, sibling} {
		p := sitecontent.MembershipPredecessor{Package: d.ID, Authority: d.AuthoritySHA256}
		for _, u := range d.Units {
			courseContextWrite(t, root, u.Unit.SourcePath, u.Unit.Source)
			p.Sources = append(p.Sources, sitecontent.Reference{Path: u.Unit.SourcePath, SHA256: u.Unit.SourceSHA256})
			if d.ID == sibling.ID {
				registry.Pages = append(registry.Pages, sitecontent.StablePage{Index: len(registry.Pages) + 1, Path: u.Unit.SourcePath, Surface: proposal.Surface})
			}
		}
		m.Predecessors = append(m.Predecessors, p)
	}
	registry.Identity = ""
	registry.Identity = seal(registry)
	data = courseContextJSON(t, registry)
	current := "engineering/current-page-registry.json"
	courseContextWrite(t, root, current, data)
	courseContextWrite(t, root, sitecontent.PageRegistryPath, data)
	m.Registry = sitecontent.Reference{Path: current, SHA256: courseContextSHA(data)}
	m.Identity = seal(m)
	data = courseContextJSON(t, m)
	ref = sitecontent.Reference{Path: "data/content-package-memberships/receipts/" + m.Identity + ".json", SHA256: courseContextSHA(data)}
	courseContextWrite(t, root, ref.Path, data)
	courseContextWrite(t, root, sitecontent.PackageMembershipPath, courseContextJSON(t, ref))
	c, err = i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := sitecontent.AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("learn-docs-v1"); err != nil || len(c.TranslationScope) != 51 {
		t.Fatal("current complete package", err)
	}
	return root, c
}

func TestSnapshotMembershipHistoricalRegistrationRun(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, c := snapshotMembershipRunFixture(t)
	const locale = "en-GB"
	f := currentValidationExportFixture{root: root, locale: locale, pkg: "learn-docs-v1", catalog: c, snapshot: "membership-finalized"}
	metadata := i18n.Locale{Locale: locale, HTMLLang: locale, LanguageName: "English fixture", EnglishName: "English fixture", Phase: "scaffold", TranslationUnit: "present.Section", LanguageAudit: &languageaudit.Authority{Schema: languageaudit.PolicySchema, Version: 1, Policy: languageaudit.Policy{Locale: locale, Profile: "membership-fixture", ProseScripts: []string{"Latin", "Han"}, SentenceTerminators: ".!?", ResidualMinWords: 1}}}
	courseContextWrite(t, root, "locales/"+locale+"/locale.json", courseContextJSON(t, metadata))
	courseContextWrite(t, root, "locales/"+locale+"/glossary.yaml", []byte("locale: "+locale+"\nmandatory:\n  Go: Go\n"))
	g, err := sitecontent.LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []string{"site-v2-shell", f.pkg} {
		_, raw, err := sitecontent.CurrentPackageDocuments(root, g, pkg)
		if err != nil {
			t.Fatal(err)
		}
		for path, data := range raw {
			courseContextWrite(t, root, path, data)
		}
	}
	corpus, err := sitecontent.BuildUnifiedCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	courseContextWrite(t, root, i18n.GlossaryCorpusPath, courseContextJSON(t, corpus))
	if _, _, err := i18n.RecordGlossaryReview(root, locale, "membership-glossary", "independent-membership-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	review, _, err := i18n.ExportUnifiedGlossaryBundle(root, locale, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, review, "membership-unified", "independent-membership-reviewer", "separate-membership-generator", "passed", nil); err != nil {
		t.Fatal(err)
	}
	target := ""
	for _, d := range c.TranslationPackages {
		if d.ID == f.pkg {
			for _, u := range d.Units {
				if u.Unit.SourcePath == "_content/doc/tutorial/json.md" {
					target = u.Unit.ID
				}
			}
		}
	}
	if target == "" {
		t.Fatal("missing target source")
	}
	land := func(batch *i18n.RetranslationExportResult, unitID string, rawOverride []byte, attempt int) []byte {
		t.Helper()
		var formal i18n.RetranslationBatchManifest
		if err := json.Unmarshal(courseContextRead(t, root, batch.BatchPath+"/manifest.json"), &formal); err != nil {
			t.Fatal(err)
		}
		o := i18n.GenerationBundleOptions{Locale: locale, BatchID: batch.BatchID}
		if attempt > 1 {
			o.UnitID = unitID
		}
		bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, o)
		if err != nil {
			t.Fatal(err)
		}
		files, err := i18n.ReadTransportBundle(bundle, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
		if err != nil {
			t.Fatal(err)
		}
		outputs := t.TempDir()
		var targetRaw []byte
		for _, out := range bm.ExpectedOutputs {
			name := filepath.Base(out.BundlePath)
			raw := files["formal/inputs/"+name]
			if formal.Units[0].RevisionFeedbackSource != "" {
				raw = files["formal/protected-candidates/"+name]
			}
			if out.UnitID == target {
				raw = bytes.ReplaceAll(raw, []byte("The "), []byte("说明"))
				if rawOverride != nil {
					raw = rawOverride
				}
				targetRaw = bytes.Clone(raw)
			}
			courseContextWrite(t, outputs, name, raw)
		}
		if _, err := i18n.ImportGenerationOutputDirectory(root, c, bundle, "chatgpt", i18n.FormalChatGPTGenerationModel, outputs); err != nil {
			t.Fatal(err)
		}
		return targetRaw
	}
	for start := 0; start < len(c.TranslationScope); start += 30 {
		end := start + 30
		if end > len(c.TranslationScope) {
			end = len(c.TranslationScope)
		}
		batch, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: c.TranslationScope[start:end]})
		if err != nil {
			t.Fatal(err)
		}
		land(batch, "", nil, 1)
		p, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: locale, BatchID: batch.BatchID})
		if err != nil || p.ValidationPassed != end-start {
			t.Fatal("initial current package", p, err)
		}
	}
	old, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: locale, SnapshotID: f.snapshot, ContentPackage: f.pkg})
	if err != nil {
		t.Fatal(err)
	}
	for start := 0; start < 51; start += 30 {
		end := start + 30
		if end > 51 {
			end = 51
		}
		if _, err := i18n.RecordQualityCheckResults(root, c, i18n.QualityCheckRecordOptions{Locale: locale, SnapshotID: f.snapshot, UnitIDs: c.TranslationScope[start:end], Rating: "A", ReviewerSession: "synthetic-independent-membership-reviewer"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := i18n.FinalizeQualityCheck(root, c, i18n.QualityCheckFinalizeOptions{Locale: locale, SnapshotID: f.snapshot}); err != nil {
		t.Fatal(err)
	}
	metadata.LanguageAudit.Version++
	metadata.LanguageAudit.Policy.Spacing = []languageaudit.SpacingRule{{ProseScript: "Han", TechnicalScript: "Latin", Kinds: []languageaudit.Kind{languageaudit.Code}}}
	courseContextWrite(t, root, "locales/"+locale+"/locale.json", courseContextJSON(t, metadata))
	a, err := i18n.AssessCurrentTranslation(root, c, locale, f.snapshot)
	if err != nil || a.Affected != 1 || a.Unknown != 0 || a.Stale != 0 {
		t.Fatal("exact current failure", a, err)
	}
	data := courseContextJSON(t, a)
	path := "engineering/membership-assessment.json"
	courseContextWrite(t, root, path, data)
	opts := i18n.CurrentValidationReopenOptions{Locale: locale, ContentPackage: f.pkg, ReopenID: "membership-reopen", PreviousSnapshotID: f.snapshot, SnapshotSHA256: courseContextSHA(courseContextRead(t, root, "data/quality-check-snapshots/"+locale+"/"+f.snapshot+"/manifest.json")), Evidence: i18n.GlossaryArchiveReference{Path: path, SHA256: courseContextSHA(data)}, AssessmentIdentity: a.Identity}
	if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: i18n.CurrentValidationAuthority, Locale: locale, ReopenID: opts.ReopenID, PreviousSnapshotID: f.snapshot, CurrentValidation: &opts}); err != nil {
		t.Fatal(err)
	}
	batch, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{target}, UnitKind: i18n.UnitKindPage, PreviousSnapshotID: f.snapshot, SurfaceReopenID: opts.ReopenID, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	baseline := land(batch, target, nil, 1)
	p, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: locale, BatchID: batch.BatchID})
	if err != nil || p.ValidationFailed != 1 {
		t.Fatal("retry setup", p, err)
	}
	land(batch, target, bytes.ReplaceAll(baseline, []byte("说明"), []byte("说明 ")), 2)
	if p, err := i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: locale, BatchID: batch.BatchID, UnitID: target}); err != nil || p.ValidationPassed != 1 {
		t.Fatal("retry candidate", p, err)
	}
	before := authorizationDiagnosisInventory(t, root)
	t.Chdir(root)
	argv := []string{"quality-check", "snapshot", "--locale", locale, "--snapshot-id", "membership-successor", "--content-package", f.pkg, "--preflight"}
	out, err := authorizationDiagnosisOutput(t, func() error { return run(argv) })
	if err != nil {
		t.Fatal("real Snapshot preflight", err, string(out))
	}
	var result struct {
		Manifest *i18n.QualityCheckSnapshotManifest `json:"manifest"`
		Recorded bool                               `json:"recorded"`
		Mutation string                             `json:"mutation"`
	}
	if err := json.Unmarshal(out, &result); err != nil || result.Manifest == nil || result.Recorded || result.Mutation != "none" {
		t.Fatal("preflight contract", err, string(out))
	}
	s := result.Manifest
	if s.ContentPackage == nil || s.ContentPackage.ID != f.pkg || s.ContentPackage.AuthoritySHA256 != old.ContentPackage.AuthoritySHA256 || s.UnitCount != 51 || s.PageCount != 51 || s.ExampleCount != 0 || !reflect.DeepEqual(s.ContentPackage.UnitIDs, c.TranslationScope) {
		t.Fatal("current 51-page scope changed")
	}
	found := false
	for _, u := range s.Units {
		if u.UnitID == target {
			found = u.SelectedBatchID == batch.BatchID && u.Attempt == 2
			v := courseContextRead(t, root, u.ValidationPath)
			if !bytes.Contains(v, []byte(`"attempt": 2`)) {
				t.Fatal("retry validation not selected")
			}
		}
	}
	if !found {
		t.Fatal("latest attempt2 candidate was overwritten by historical scope")
	}
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("Snapshot preflight mutated state")
	}
	if _, err := os.Stat(filepath.Join(root, "data/quality-check-snapshots", locale, "membership-successor")); !os.IsNotExist(err) {
		t.Fatal("preflight created Snapshot")
	}
	t.Log("SNAPSHOT_PREFLIGHT_TRUE_RUN=PASS CURRENT_SCOPE=51 RETRY_ATTEMPT=2 HISTORICAL_AUTHORITY=exact NEW_SNAPSHOT=no")
}
