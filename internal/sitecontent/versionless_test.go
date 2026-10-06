package sitecontent

import (
	"bytes"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestVersionlessCorpus(t *testing.T) {
	root := repositoryRoot(t)
	c, err := BuildUnifiedCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckUnifiedCorpus(root); err != nil {
		t.Fatal(err)
	}
	again, err := BuildUnifiedCorpus(root)
	if err != nil || identity(c) != identity(again) {
		t.Fatal("nondeterministic", err)
	}
	counts := map[string]int{}
	contexts, size := 0, 0
	for _, v := range c.Contributors {
		counts[v.Parser]++
		for _, x := range v.Contexts {
			contexts++
			size += len(x.Text)
			for _, p := range x.Protected {
				size += len(p)
			}
		}
	}
	if counts["present-visible/v1"] != 103 || counts["go-comment-visible/v1"] != 19 || counts[UnitContract] != 58 || counts["canonical-description/v1"] != 103 {
		t.Fatal(counts)
	}
	t.Logf("corpus contributors=%d contexts=%d projected UTF8 bytes=%d identity=%s categories=%v", len(c.Contributors), contexts, size, c.Identity, counts)
	clone := *c
	clone.Sources = append([]i18n.GlossaryArchiveReference{}, c.Sources...)
	clone.Sources[0].SHA256 = strings.Repeat("a", 64)
	if i18n.CorpusIdentity(clone) != c.Identity {
		t.Fatal("machine byte drift changed projected corpus")
	}
	clone.Contributors = append([]i18n.LanguageContributor{}, c.Contributors...)
	clone.Contributors[0].Contexts = append([]i18n.LanguageContext{}, c.Contributors[0].Contexts...)
	clone.Contributors[0].Contexts[0].Text += " new visible terminology"
	if i18n.CorpusIdentity(clone) == c.Identity {
		t.Fatal("visible text drift ignored")
	}
	clone.Contributors[0].Parser = "unknown"
	clone.Identity = i18n.CorpusIdentity(clone)
	if i18n.ValidateGlossaryCorpus(&clone) == nil {
		t.Fatal("unknown parser accepted")
	}
}
func TestVersionlessCorpusProjection(t *testing.T) {
	project := func(path, kind, text string) i18n.LanguageContributor {
		b := []byte(text)
		d, err := ParseDocument(Source{Path: path, Kind: kind, SHA256: digest(b)}, b)
		if err != nil {
			t.Fatal(err)
		}
		c, err := projectedDocumentContributor(*d, "learn-docs-v1")
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	page := "# Teaching\n\nVisible English prose.\n\n```go\nfunc main() {\n // Create a Context with a timeout.\n Context()\n var count = 1\n}\n```\n"
	c := project("x.md", "page", page)
	machine := project("x.md", "page", strings.Replace(page, "var count = 1", "var count = 2", 1))
	if c.SourceSHA != machine.SourceSHA {
		t.Fatal("machine-only code changed terminology projection")
	}
	visible := project("x.md", "page", strings.Replace(page, "with a timeout", "with a deadline", 1))
	if c.SourceSHA == visible.SourceSHA {
		t.Fatal("teaching comment body missing from corpus")
	}
	for _, x := range c.Contexts {
		if strings.Contains(x.Text, "func main") || strings.Contains(x.Text, "var count") {
			t.Fatal("machine code exposed as corpus language")
		}
	}
	yaml := "- title: Visible book title\n  url: /machine-route\n  credits: 5\n"
	a := project("x.yaml", "data", yaml)
	b := project("x.yaml", "data", strings.Replace(yaml, "/machine-route", "/changed-route", 1))
	if a.SourceSHA != b.SourceSHA || len(a.Contexts) != 1 || strings.TrimSpace(a.Contexts[0].Text) != "Visible book title" {
		t.Fatalf("machine YAML field became terminology context: %+v / %+v", a, b)
	}
}

func TestVersionlessSharedAuthorityCAS(t *testing.T) {
	root := siteFixture(t)
	before, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RefreshSharedCorpusAuthority(root, digest(before), true); err != nil {
		t.Fatal(err)
	}
	after, _ := readRegular(root, i18n.GlossaryCorpusPath)
	if !bytes.Equal(before, after) {
		t.Fatal("same projection mutated static authority")
	}
	if _, err := RefreshSharedCorpusAuthority(root, strings.Repeat("0", 64), true); err == nil {
		t.Fatal("unknown prior mutation replayed")
	}
	after, _ = readRegular(root, i18n.GlossaryCorpusPath)
	if !bytes.Equal(before, after) {
		t.Fatal("failed CAS changed authority")
	}
	var c i18n.GlossarySourceCorpus
	if err := StrictJSON(before, &c); err != nil {
		t.Fatal(err)
	}
	unknown := c.Contributors[0]
	unknown.ID = "zz:unregistered-contributor"
	unknown.SourceSHA = i18n.ContributorIdentity(unknown)
	c.Contributors = append(c.Contributors, unknown)
	c.Identity = i18n.CorpusIdentity(c)
	b, _ := Encode(c)
	writeFixture(t, root, i18n.GlossaryCorpusPath, b)
	if CheckUnifiedCorpus(root) == nil {
		t.Fatal("unknown contributor accepted instead of exact registry set")
	}
}

func TestVersionlessPageRegistry(t *testing.T) {
	root := repositoryRoot(t)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	r, err := LoadPageRegistry(root, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Pages) != 49 {
		t.Fatal(len(r.Pages))
	}
	next := append([]StablePage{}, r.Pages...)
	next = append(next, StablePage{Path: "_content/doc/tutorial/aaa-new.md", Surface: "tutorial"})
	n, err := ReconcilePageRegistry(*r, next, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n.Pages[49].Index != 50 {
		t.Fatal("new Page not appended")
	}
	for i := 0; i < 49; i++ {
		if n.Pages[i] != r.Pages[i] {
			t.Fatal("renumbered")
		}
	}
	deleted := append([]StablePage{}, next[1:]...)
	n, err = ReconcilePageRegistry(*n, deleted, nil)
	if err != nil || !n.Pages[0].Retired {
		t.Fatal("tombstone", err)
	}
	moved := append([]StablePage{}, deleted...)
	old := moved[0].Path
	moved[0].Path = "_content/learn/moved.html"
	n2, err := ReconcilePageRegistry(*n, moved, map[string]string{old: moved[0].Path})
	if err != nil || n2.Pages[1].Path != moved[0].Path {
		t.Fatal("explicit move", err)
	}
	if _, err := ReconcilePageRegistry(*n, moved, map[string]string{old: "missing.md"}); err == nil {
		t.Fatal("ambiguous move accepted")
	}
	for len(moved) < 60 {
		moved = append(moved, StablePage{Path: "_content/doc/tutorial/future-" + string(rune('a'+len(moved))) + ".md", Surface: "tutorial"})
	}
	if _, err := ReconcilePageRegistry(*n2, moved, nil); err == nil {
		t.Fatal("tombstone reused / index61 accepted")
	}
	nextSources := []Source{}
	_, raw, err := PackageDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range g.Sources {
		if s.Package == "learn-docs-v1" && s.Kind == "page" {
			nextSources = append(nextSources, s)
		}
	}
	for i := 0; i < 12; i++ {
		path := fmt.Sprintf("_content/doc/tutorial/aaa-future-%02d.md", i)
		b := []byte("# Future Page\n\nTeaching context.\n")
		raw[path] = b
		nextSources = append(nextSources, Source{Path: path, Kind: "page", Package: "learn-docs-v1", Surface: "tutorial", SHA256: digest(b)})
	}
	if r60, err := ReconcilePageSources(*r, nextSources[:60], raw, nil); err != nil || len(r60.Pages) != 60 || r60.Pages[59].Index != 60 {
		t.Fatal("50–60 capacity", err)
	}
	if _, err := ReconcilePageSources(*r, nextSources, raw, nil); err == nil || !strings.Contains(err.Error(), "source_sha256=") || !strings.Contains(err.Error(), "context_bytes=") || !strings.Contains(err.Error(), "active=61 occupancy=30/30") {
		t.Fatal("index61 source-bound diagnostic", err)
	}
	t.Logf("registry identity=%s active=49 fixed membership=30/19", r.Identity)
}
func TestVersionlessPageProtectionAndFixedPlan(t *testing.T) {
	root := repositoryRoot(t)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 49 {
		t.Fatal(len(docs))
	}
	p, err := PlanPageWorkingSets(docs, raw, selectAll(docs), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Sets) != 2 || len(p.Sets[0].Selected) != 30 || len(p.Sets[1].Selected) != 19 {
		t.Fatal(p)
	}
	for _, d := range docs {
		if len(d.Units) != 1 {
			t.Fatal("slot is formal TU")
		}
		b, err := ReconstructWorkflow(d, raw[d.Path], map[string]string{d.Units[0].ID: d.Units[0].Source})
		if err != nil || !bytes.Equal(b, raw[d.Path]) {
			t.Fatalf("%s roundtrip: %v", d.Path, err)
		}
	}
	t.Logf("fixed Page batches contexts=%d/%d bytes; Page=49 embedded Example=0", p.Sets[0].ContextBytes, p.Sets[1].ContextBytes)
	source := []byte("## Create a folder for your code {#create_folder}\n\nFor other tutorials, see [Tutorials](/doc/tutorial/index).\n\nUse `go mod tidy` to update the module.\n\n```go\nfunc main() {\n // Create a Context with a timeout.\n Context()\n}\n```\n")
	s := Source{Path: "_content/doc/tutorial/test.md", Kind: "page", SHA256: digest(source)}
	d, err := ParseDocument(s, source)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	teaching := false
	for _, u := range d.Units {
		v := u.Source
		if strings.Contains(u.Locator, "teaching-comment") {
			teaching = true
		}
		values[u.ID] = "译文 " + v
	}
	if !teaching {
		t.Fatal("safe teaching comment not exposed")
	}
	b, err := Reconstruct(s, source, values)
	if err != nil {
		t.Fatal(err)
	}
	for _, locked := range []string{"{#create_folder}", "/doc/tutorial/index", "`go mod tidy`", "func main()", "Context()", "// "} {
		if !bytes.Contains(b, []byte(locked)) {
			t.Fatal("protected drift", locked)
		}
	}
	again, err := Reconstruct(s, source, values)
	if err != nil || !bytes.Equal(b, again) {
		t.Fatal("reconstruction nondeterministic")
	}
}
func TestVersionlessUnifiedReviewCoexistence(t *testing.T) {
	root := siteFixture(t)
	if err := i18n.RequireCurrentGlossaryReview(root, "zh-CN"); err != nil {
		t.Fatal("old receipt ambiguity", err)
	}
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err != nil {
		t.Fatal(err)
	}
	b, m, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "generation")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.ExpectedOutputs) != 1 || m.ExpectedOutputs[0] != "locales/zh-CN/glossary.yaml" {
		t.Fatal("split generation")
	}
	if _, err := i18n.CheckUnifiedGlossaryBundle(root, b); err != nil {
		t.Fatal(err)
	}
	rb, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, rb, "conflicting-review", "second-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("two unified authorities accepted")
	}
	if err := os.RemoveAll(root + "/data/unified-glossary-reviews"); err != nil {
		t.Fatal(err)
	}
	if err := i18n.RequireCurrentGlossaryReview(root, "zh-CN"); err != nil {
		t.Fatal("legacy coverage broken", err)
	}
	if _, err := GenerationPlan(root, "zh-CN", "learn-docs-v1", "initial", nil); err == nil {
		t.Fatal("Tour-only review accepted for Site")
	}
}
func TestVersionlessPageGenerationAndQC(t *testing.T) {
	root := siteFixture(t)
	plan, err := GenerationPlan(root, "zh-CN", "learn-docs-v1", "initial", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Sets) != 2 {
		t.Fatal("not two Page invocations")
	}
	refs := []Reference{}
	reviews := []Reference{}
	for _, set := range plan.Sets {
		_, ref, _ := importFixture(t, root, "learn-docs-v1", "fixed-page-"+string(rune('0'+set.Index)), set.Selected)
		refs = append(refs, ref)
		reviews = append(reviews, reviewFixture(t, root, "learn-docs-v1", ref, set.Selected, "A")...)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	sort.Slice(reviews, func(i, j int) bool { return reviews[i].Path < reviews[j].Path })
	f, name, err := FinalizePackage(root, "zh-CN", "learn-docs-v1", refs, reviews, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Targets) != 49 {
		t.Fatal("one rating per Page")
	}
	ref, _ := ReferenceFile(root, name)
	if _, err := CheckFinalization(root, ref); err != nil {
		t.Fatal(err)
	}
	if _, err := ActivationPreflight(root, ref); err == nil {
		t.Fatal("Page-only partial package activated")
	}
	all := append(append([]string{}, plan.Sets[0].Selected...), plan.Sets[1].Selected...)
	sort.Strings(all)
	if _, _, err := ExportPackageGeneration(root, "zh-CN", "learn-docs-v1", "initial", "combined", all, nil); err == nil {
		t.Fatal("combined fixed batches accepted")
	}
	first, err := loadReview(root, reviews[0])
	if err != nil {
		t.Fatal(err)
	}
	first.Identity, first.ReviewID, first.Invocation = "", "synthetic-page-finding", "synthetic-page-finding"
	first.Ratings[0].Rating, first.Ratings[0].Findings = "B", []string{"synthetic exact Page finding"}
	rawFinding, _ := Encode(first)
	_, findingPath, err := RecordPackageReview(root, rawFinding)
	if err != nil {
		t.Fatal(err)
	}
	finding, _ := ReferenceFile(root, findingPath)
	failedReviews := append([]Reference{}, reviews...)
	failedReviews[0] = finding
	sort.Slice(failedReviews, func(i, j int) bool { return failedReviews[i].Path < failedReviews[j].Path })
	if _, _, err := FinalizePackage(root, "zh-CN", "learn-docs-v1", refs, failedReviews, nil); err == nil {
		t.Fatal("B-rated Page finalized")
	}
	revision, batch, err := ExportPackageGeneration(root, "zh-CN", "learn-docs-v1", "revision", "synthetic-page-revision", nil, &finding)
	if err != nil || len(batch.Selected) != 1 || batch.Selected[0] != first.Ratings[0].ID {
		t.Fatal("exact Page revision scope", err)
	}
	files, err := i18n.ReadTransportBundle(revision, 256, 32<<20)
	if err != nil || len(files["context/prior-targets.json"]) == 0 {
		t.Fatal("prior candidate missing from revision context", err)
	}
	out := Output{Schema: WorkflowSchema, BatchSHA: batch.Identity, Targets: []Target{{ID: batch.Selected[0], Text: batch.Documents[0].Units[0].Source}}}
	rawOutput, _ := Encode(out)
	_, path, err := ImportPackageGeneration(root, revision, rawOutput, Provenance{Provider: "codex", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-06T00:02:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	child, _ := ReferenceFile(root, path)
	refs = append(refs, child)
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	failedReviews = append(failedReviews, reviewFixture(t, root, "learn-docs-v1", child, batch.Selected, "A")...)
	sort.Slice(failedReviews, func(i, j int) bool { return failedReviews[i].Path < failedReviews[j].Path })
	if _, _, err := FinalizePackage(root, "zh-CN", "learn-docs-v1", refs, failedReviews, nil); err != nil {
		t.Fatal("re-QC exact Page A / unaffected A carry", err)
	}
}

func TestVersionlessExistingTourCarryPlan(t *testing.T) {
	plan, err := PlanLocaleLanguage(repositoryRoot(t), "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	pages, examples, shared, pending := 0, 0, 0, 0
	for _, w := range plan.Work {
		if w.Package != "tour-v1" {
			if w.State != "generation-required" {
				t.Fatal("new package claim", w)
			}
			pending++
			continue
		}
		if w.State != "carry" {
			t.Fatal("unaffected Tour language reopened", w)
		}
		if strings.HasPrefix(w.ID, "tu:example:") {
			examples++
		} else if strings.HasPrefix(w.ID, "tu:") {
			pages++
		} else {
			shared++
		}
	}
	if pages != 103 || examples != 19 || pending != 58 {
		t.Fatalf("carry pages=%d examples=%d shared=%d pending=%d", pages, examples, shared, pending)
	}
	encoded, _ := Encode(plan)
	if len(encoded) > 1<<20 {
		t.Fatal("historical package closure duplicated per carried row")
	}
	t.Logf("existing exact Tour: Page=%d Example=%d shared=%d carry; new shell/docs=%d pending", pages, examples, shared, pending)
}

func existingTourFixture(t *testing.T) string {
	t.Helper()
	root, repository := siteFixture(t), repositoryRoot(t)
	copyFile := func(path string) {
		b, err := readRegular(repository, path)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path, b)
	}
	g, err := LoadCurrent(repository)
	if err != nil {
		t.Fatal(err)
	}
	l, err := CheckLocale(repository, g, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range l.Packages[0].Evidence {
		copyFile(r.Path)
	}
	copyFile(LocalePath("zh-CN"))
	for _, path := range []string{"internal/tour/tour.go", "internal/tour/production.go", "internal/tour/seo.go", "_content/js/playground.js", "data/glossary-review-legacy-coverage.json"} {
		copyFile(path)
	}
	if err := filepath.WalkDir(filepath.Join(repository, "data/retranslation-runs/zh-CN"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			rel, err := filepath.Rel(repository, path)
			if err != nil {
				return err
			}
			copyFile(filepath.ToSlash(rel))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"data/course-seo", "data/surface-config-history", "data/glossary-history/zh-CN", "data/glossary-reviews/zh-CN"} {
		entries, err := os.ReadDir(filepath.Join(repository, dir))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				copyFile(dir + "/" + e.Name())
			}
		}
	}
	b, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, b, "synthetic-existing-tour-unified", "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestVersionlessIntegratedTourExpansion(t *testing.T) {
	root := existingTourFixture(t)
	p, err := PlanStructuredAssets(root, "zh-CN")
	if err != nil || len(p.Expected) != 9 {
		t.Fatal("existing Tour structured assets not carried", err)
	}
	plan, err := GenerationPlan(root, "zh-CN", "learn-docs-v1", "initial", nil)
	if err != nil {
		t.Fatal(err)
	}
	refs, reviews := []Reference{}, []Reference{}
	for _, set := range plan.Sets {
		_, ref, _ := importFixture(t, root, "learn-docs-v1", fmt.Sprintf("synthetic-expansion-%d", set.Index), set.Selected)
		refs = append(refs, ref)
		reviews = append(reviews, reviewFixture(t, root, "learn-docs-v1", ref, set.Selected, "A")...)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	sort.Slice(reviews, func(i, j int) bool { return reviews[i].Path < reviews[j].Path })
	_, name, err := FinalizePackage(root, "zh-CN", "learn-docs-v1", refs, reviews, nil)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := ReferenceFile(root, name)
	_, structured := importStructuredFixture(t, root)
	_, name, err = FinalizePackageClosure(root, "zh-CN", "learn-docs-v1", &f, []Reference{structured})
	if err != nil {
		t.Fatal(err)
	}
	closure, _ := ReferenceFile(root, name)
	scope, err := BuildIntegratedSurfaceScope(root, "zh-CN", []Reference{closure})
	if err != nil {
		t.Fatal(err)
	}
	oldPages := 0
	for _, a := range scope.Carried {
		if strings.HasPrefix(a.ID, "tu:") {
			oldPages++
		}
	}
	if oldPages != 122 || len(scope.Reviewed) != 56 {
		t.Fatalf("Tour carried=%d new Docs reviewed=%d", oldPages, len(scope.Reviewed))
	}
	for _, a := range scope.Reviewed {
		if a.Package != "learn-docs-v1" {
			t.Fatal("old language scope re-reviewed", a)
		}
	}
	encoded, _ := Encode(scope)
	if len(encoded) > 1<<20 {
		t.Fatal("old closure repeated per carried asset")
	}
	t.Logf("integrated scope: old Tour TU=122 carried, new Docs=56 reviewed; structured pending=9, scope bytes=%d", len(encoded))
}
func importStructuredFixture(t *testing.T, root string) (*StructuredGeneration, Reference) {
	t.Helper()
	b, m, err := ExportStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	o := StructuredOutput{Schema: StructuredSchema, BundleSHA: m.Identity, Outputs: []StructuredTarget{}}
	assets := map[string]StructuredAsset{}
	for _, a := range m.Plan.Assets {
		assets[a.TargetPath] = a
	}
	for _, path := range m.Plan.Expected {
		a := assets[path]
		if a.Document == nil {
			continue
		}
		slots := []Target{}
		for _, u := range a.Document.Units {
			slots = append(slots, Target{u.ID, u.Source})
		}
		o.Outputs = append(o.Outputs, StructuredTarget{Path: path, Slots: slots})
	}
	// Fixture omits Tour shared assets from model work only after a separately
	// checked carried plan; use complete source UI/header outputs for this new locale.
	for _, path := range m.Plan.Expected {
		a := assets[path]
		if a.Document != nil {
			continue
		}
		data := ""
		if a.ID == "tour-ui" {
			b, err := readRegular(root, "internal/tour/ui/en.json")
			if err != nil {
				t.Fatal(err)
			}
			data = strings.Replace(string(b), `"locale": "en"`, `"locale": "zh-CN"`, 1)
			data = strings.Replace(data, `"html_lang": "en"`, `"html_lang": "zh-CN"`, 1)
		} else {
			cs, err := i18n.LoadUnifiedGlossaryCorpus(root)
			if err != nil {
				t.Fatal(err)
			}
			entries := []i18n.ArticleMetadata{}
			for _, c := range cs.Contributors {
				if c.Parser == "article-header/v1" {
					a, b, _ := strings.Cut(c.Contexts[0].Text, "\n")
					entries = append(entries, i18n.ArticleMetadata{Article: strings.TrimPrefix(c.Path, "_content/tour/"), Title: a, Subtitle: b})
				}
			}
			b, _ := Encode(struct {
				Locale   string                 `json:"locale"`
				Articles []i18n.ArticleMetadata `json:"articles"`
			}{"zh-CN", entries})
			data = string(b)
		}
		o.Outputs = append(o.Outputs, StructuredTarget{Path: path, Data: data, Slots: []Target{}})
	}
	sort.Slice(o.Outputs, func(i, j int) bool { return o.Outputs[i].Path < o.Outputs[j].Path })
	raw, _ := Encode(o)
	r, p, err := ImportStructuredAssets(root, b, raw, Provenance{Provider: "codex", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-06T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := ReferenceFile(root, p)
	return r, ref
}
func TestVersionlessStructuredAtomicSurface(t *testing.T) {
	root := siteFixture(t)
	if err := os.Remove(root + "/" + LocalePath("zh-CN")); err != nil {
		t.Fatal(err)
	}
	initial, err := PlanLocaleLanguage(root, "zh-CN")
	if err != nil || len(initial.Work) != 58 {
		t.Fatal("same locale campaign requires no second content init", err)
	}
	p, err := PlanStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Assets) != 11 || len(p.Expected) != 11 {
		t.Fatal("shared structured registry", len(p.Assets), len(p.Expected))
	}
	r, ref := importStructuredFixture(t, root)
	if _, _, err := CheckStructuredGeneration(root, ref); err != nil {
		t.Fatal(err)
	}
	closure, name, err := FinalizePackageClosure(root, "zh-CN", "site-v2-shell", nil, []Reference{ref})
	if err != nil {
		t.Fatal(err)
	}
	cr, _ := ReferenceFile(root, name)
	if len(closure.Files) != 2 {
		t.Fatal("shell nonatomic")
	}
	if _, err := ActivationPreflight(root, cr); err == nil {
		t.Fatal("activated without integrated independent Surface")
	}
	scope, err := BuildIntegratedSurfaceScope(root, "zh-CN", []Reference{cr})
	if err != nil {
		t.Fatal(err)
	}
	if len(scope.Reviewed) != 2 {
		t.Fatal("structured scope")
	}
	md := []byte("# Synthetic isolated independent Surface evidence\n\nPASS.\n")
	writeFixture(t, root, "data/synthetic-surface.md", md)
	receipt := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: "synthetic-surface", Reviewer: "synthetic-independent-reviewer", Decision: "passed", Findings: []string{}, Evidence: Reference{Path: "data/synthetic-surface.md", SHA256: digest(md)}}
	b, _ := Encode(receipt)
	if _, _, err := RecordIntegratedSurface(root, b); err != nil {
		t.Fatal(err)
	}
	plan, err := ActivationPreflight(root, cr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyActivation(root, cr, plan.PriorSHA); err != nil {
		t.Fatal(err)
	}
	g, _ := LoadCurrent(root)
	if _, err := CheckLocale(root, g, "zh-CN"); err != nil {
		t.Fatal("activated closure gate", err)
	}
	carried, err := PlanStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range carried.Assets {
		if a.Package == "site-v2-shell" && a.State != "carry" {
			t.Fatal("active unchanged structured result regenerated", a)
		}
	}
	if len(r.Output.Outputs) != 11 {
		t.Fatal("one composite invocation lost outputs")
	}
}
func TestVersionlessUnifiedStaleAndStrict(t *testing.T) {
	root := siteFixture(t)
	b, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "generation")
	if err != nil {
		t.Fatal(err)
	}
	glossary, _ := readRegular(root, "locales/zh-CN/glossary.yaml")
	writeFixture(t, root, "locales/zh-CN/glossary.yaml", append(glossary, []byte("# changed bytes\n")...))
	if _, err := i18n.CheckUnifiedGlossaryBundle(root, b); err == nil {
		t.Fatal("stale glossary accepted")
	}
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("stale Review accepted")
	}
	writeFixture(t, root, "locales/zh-CN/glossary.yaml", glossary)
	corpus, _ := i18n.LoadUnifiedGlossaryCorpus(root)
	corpus.Contributors[0].Contexts[0].Text += " changed prose"
	corpus.Contributors[0].SourceSHA = i18n.ContributorIdentity(corpus.Contributors[0])
	corpus.Identity = i18n.CorpusIdentity(*corpus)
	raw, _ := Encode(corpus)
	writeFixture(t, root, i18n.GlossaryCorpusPath, raw)
	if _, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN"); err == nil {
		t.Fatal("new corpus reviewed by old receipt")
	}
	if _, err := i18n.CheckUnifiedGlossaryBundle(root, b); err == nil {
		t.Fatal("stale corpus bundle accepted")
	}
	for _, bad := range []string{`{"schema":"x","unknown":true}`, `{"schema":"x","schema":"y"}`, `{} {}`} {
		var c i18n.GlossarySourceCorpus
		if StrictJSON([]byte(bad), &c) == nil {
			t.Fatal("strict JSON", bad)
		}
	}
	if _, _, err := i18n.ExportUnifiedGlossaryBundle(root, "../zh-CN", "generation"); err == nil {
		t.Fatal("traversal locale")
	}
	if err := os.Rename(root+"/locales/zh-CN/glossary.yaml", root+"/locales/zh-CN/glossary-real.yaml"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("glossary-real.yaml", root+"/locales/zh-CN/glossary.yaml"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "generation"); err == nil {
		t.Fatal("symlink glossary")
	}
}
func TestVersionlessDeltaReuseAndUnrelatedPage(t *testing.T) {
	delta, err := i18n.ParsedGlossaryDelta("zh-CN", []byte("locale: zh-CN\nmandatory:\n  synthetic: 测试\n"), []byte("locale: zh-CN\nmandatory:\n  synthetic: 测试\npreferred:\n  SQL transaction: SQL 事务\n"))
	if err != nil {
		t.Fatal(err)
	}
	ok, _ := i18n.ParsedGlossaryImpact("A function returns a value.", "函数返回一个值。", delta)
	if !ok || ClassifyLanguageWork(true, true, true, ok, false, false) != "carry" {
		t.Fatal("Docs-only term stales unrelated Tour")
	}
	delta, err = i18n.ParsedGlossaryDelta("zh-CN", []byte("locale: zh-CN\nmandatory:\n  synthetic: 测试\n"), []byte("locale: zh-CN\nmandatory:\n  synthetic: 测试\npreferred:\n  function: 函数\n"))
	if err != nil {
		t.Fatal(err)
	}
	affected, _ := i18n.ParsedGlossaryImpact("A function returns a value.", "函数返回一个值。", delta)
	if affected || ClassifyLanguageWork(true, true, true, affected, false, false) != "stale/reopen" {
		t.Fatal("affected core term carried")
	}
	if ClassifyLanguageWork(false, true, true, true, false, false) != "stale/reopen" || ClassifyLanguageWork(true, false, false, true, false, false) != "generation-required" || ClassifyLanguageWork(true, true, false, true, false, false) != "review-required" {
		t.Fatal("scope classification")
	}
	root := siteFixture(t)
	global, _ := LoadCurrent(root)
	docs, raw, err := WorkflowDocuments(root, global, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{docs[0].Units[0].ID}
	_, gr, _ := importFixture(t, root, "learn-docs-v1", "prior-page", ids)
	rr := reviewFixture(t, root, "learn-docs-v1", gr, ids, "A")
	before, _ := readRegular(root, rr[0].Path)
	next := append([]Document{}, docs...)
	s := documentSource(next[1])
	changed := append(bytes.Clone(raw[s.Path]), []byte("\nA new unrelated English paragraph.\n")...)
	s.SHA256 = digest(changed)
	d, err := ParseWorkflowDocument(s, changed, next[1].StableIndex)
	if err != nil {
		t.Fatal(err)
	}
	next[1] = *d
	values, err := selections(root, "zh-CN", "learn-docs-v1", []Reference{gr}, next)
	if err != nil || values[ids[0]].Unit.ID != ids[0] {
		t.Fatal("whole package stale coupling", err)
	}
	if _, err := loadReview(root, rr[0]); err != nil {
		t.Fatal("historical A cannot carry", err)
	}
	after, _ := readRegular(root, rr[0].Path)
	if !bytes.Equal(before, after) {
		t.Fatal("rewritten historical A")
	}
	reconciliation, err := ReconcileDocuments(docs, next)
	if err != nil {
		t.Fatal(err)
	}
	carry, stale := 0, 0
	for _, u := range reconciliation.Units {
		if u.State == "carry" {
			carry++
		}
		if u.State == "stale" {
			stale++
		}
	}
	if carry != 48 || stale != 1 {
		t.Fatal("imprecise Page reconcile", carry, stale)
	}
}
func TestVersionlessReviewerWholePagesDedup(t *testing.T) {
	root := siteFixture(t)
	g, ref, _ := importFixture(t, root, "learn-docs-v1", "review-page-batch", nil)
	r, err := ReviewScope(root, "zh-CN", "learn-docs-v1", []Reference{ref}, g.Batch.Selected)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ExportPackageReview(root, r)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckPackageReviewBundle(root, b); err != nil {
		t.Fatal(err)
	}
	fs, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		t.Fatal(err)
	}
	var m PackageReviewerBundle
	if err := StrictJSON(fs["manifest.json"], &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Rows) != 30 || len(m.Documents) != 30 {
		t.Fatal("not one row/context per Page")
	}
	if bytes.Contains(fs["manifest.json"], []byte(`"document":`)) {
		t.Fatal("Document embedded per row")
	}
	t.Logf("30 full Page Reviewer bundle ZIP=%d manifest=%d bytes", len(b), len(fs["manifest.json"]))
}
func TestVersionlessSurfaceExactRevision(t *testing.T) {
	root := siteFixture(t)
	_, gr := importStructuredFixture(t, root)
	_, name, err := FinalizePackageClosure(root, "zh-CN", "site-v2-shell", nil, []Reference{gr})
	if err != nil {
		t.Fatal(err)
	}
	closure, _ := ReferenceFile(root, name)
	scope, err := BuildIntegratedSurfaceScope(root, "zh-CN", []Reference{closure})
	if err != nil {
		t.Fatal(err)
	}
	md := []byte("# Synthetic isolated Surface finding\n\nAn exact shell finding.\n")
	writeFixture(t, root, "data/synthetic-finding.md", md)
	r := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: "shell-finding", Reviewer: "synthetic-independent-reviewer", Decision: "failed", Findings: []string{"synthetic exact language finding"}, ExactFindings: []SurfaceFinding{{ID: "data/site-shell/home.html", Finding: "synthetic finding"}}, Evidence: Reference{"data/synthetic-finding.md", digest(md)}}
	b, _ := Encode(r)
	_, p, err := RecordIntegratedSurface(root, b)
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := ReferenceFile(root, p)
	bundle, m, err := ExportStructuredAssetsRevision(root, "zh-CN", ref)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Plan.Expected) != 1 {
		t.Fatal("unrelated structured asset reopened")
	}
	if _, err := CheckStructuredAssetsBundle(root, bundle); err != nil {
		t.Fatal(err)
	}
	o := StructuredOutput{Schema: StructuredSchema, BundleSHA: m.Identity, Outputs: []StructuredTarget{}}
	for _, a := range m.Plan.Assets {
		if a.State != "generation-required" {
			continue
		}
		targets := []Target{}
		for _, u := range a.Document.Units {
			targets = append(targets, Target{u.ID, u.Source})
		}
		o.Outputs = append(o.Outputs, StructuredTarget{Path: a.TargetPath, Slots: targets})
	}
	raw, _ := Encode(o)
	_, path, err := ImportStructuredAssets(root, bundle, raw, Provenance{Provider: "codex", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-06T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	replacement, _ := ReferenceFile(root, path)
	refs := []Reference{gr, replacement}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	if _, _, err := FinalizePackageClosure(root, "zh-CN", "site-v2-shell", nil, refs); err != nil {
		t.Fatal("exact independent replacement lineage", err)
	}
}
