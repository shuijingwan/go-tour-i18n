package sitecontent

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"

	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}
func repositoryGlobal(t *testing.T) *Global {
	t.Helper()
	g, err := LoadCurrent(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestStrictJSON(t *testing.T) {
	for _, s := range []string{`{"schema":"x","schema":"y"}`, `{"schema":"x","unknown":1}`, `{"schema":"x"} {}`, `{"packages":[{"state":"x","state":"y"}]}`, `{"schema":`} {
		var l Locale
		if StrictJSON([]byte(s), &l) == nil {
			t.Errorf("accepted %s", s)
		}
	}
	var l Locale
	if err := StrictJSON([]byte(`{"schema":"x","locale":"he","packages":[]}`), &l); err != nil {
		t.Fatal(err)
	}
}

func TestFrozenInventoryCurrent(t *testing.T) {
	g := repositoryGlobal(t)
	counts := map[string]int{}
	rootCounts := map[string]int{}
	redirects := map[string]bool{}
	for _, s := range g.Sources {
		if s.Package != "learn-docs-v1" {
			continue
		}
		counts[s.Kind]++
		for _, root := range learnRoots {
			if strings.HasPrefix(s.Path, root+"/") {
				rootCounts[root]++
			}
		}
		if s.Kind == "data" && s.Route != "" {
			t.Fatal("data route")
		}
		if s.Kind == "redirect" {
			if s.Route != "" || s.ResolvedRedirect == "" {
				t.Fatal("redirect page")
			}
			redirects[s.Path] = true
		}
		if s.Surface == "security" && s.Kind == "page" && !strings.HasPrefix(s.Route, "/doc/security/") {
			t.Fatal("reverse security canonical")
		}
	}
	if !reflect.DeepEqual(counts, map[string]int{"page": 49, "data": 7, "redirect": 6, "asset": 18}) {
		t.Fatalf("counts %v", counts)
	}
	for i, want := range []int{8, 25, 9, 16, 22} {
		if rootCounts[learnRoots[i]] != want {
			t.Fatalf("root %s count %d", learnRoots[i], rootCounts[learnRoots[i]])
		}
	}
	for _, p := range []string{"vulncheck.md", "vuln/vulncheck.md", "vulndb/index.md", "vulndb/api.md", "vulndb/policy.md"} {
		if !redirects["_content/doc/security/"+p] {
			t.Fatalf("missing redirect %s", p)
		}
	}
	if !redirects["_content/doc/modules/pruning.md"] || len(redirects) != 6 {
		t.Fatal("missing JSON metadata redirect")
	}
	for _, s := range g.Sources {
		if s.Path == "_content/doc/modules/pruning.md" && (s.Kind != "redirect" || s.Alias != "/doc/modules/pruning" || s.Redirect != "/doc/modules/managing-dependencies" || s.ResolvedRedirect != s.Redirect) {
			t.Fatalf("incorrect pruning redirect: %+v", s)
		}
	}
	p, _ := packageByID(g, "learn-docs-v1")
	for _, r := range p.Routes {
		if r == "/doc/modules/pruning" {
			t.Fatal("redirect alias published as page")
		}
	}
	b, err := os.ReadFile(filepath.Join(repositoryRoot(t), SnapshotPath))
	if err != nil {
		t.Fatal(err)
	}
	next, err := BuildGlobal(repositoryRoot(t), b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*g, *next) {
		t.Fatal("not deterministic")
	}
}

func TestInventoryDependenciesAndRedirects(t *testing.T) {
	d := dependencyPaths("_content/doc/tutorial/x.md", []byte(`<img src="images/a.png"> https://gstatic.com/a.png <img src="/security/fuzz/example.png"> {{data "a.yaml"}}`))
	want := []Dependency{{Kind: "asset", Path: "_content/doc/security/fuzz/example.png"}, {Kind: "data", Path: "_content/doc/tutorial/a.yaml"}, {Kind: "asset", Path: "_content/doc/tutorial/images/a.png"}}
	if !reflect.DeepEqual(d, want) {
		t.Fatalf("%v", d)
	}
	target, err := redirectMetadata([]byte("---\nredirect: /security/vuln\n---\n"))
	if pageRoute("_content/doc/security/index.md") != "/doc/security/" || err != nil || target != "/security/vuln" {
		t.Fatal("route contract")
	}
	if _, err := redirectsFromSource([]byte(`package redirect; var redirects=map[string]string{"/doc/security/":"/security/"}`)); err == nil {
		t.Fatal("accepted reverse canonical")
	}
}

func TestHTMLCommentJSONRedirectMetadata(t *testing.T) {
	for _, key := range []string{"Redirect", "redirect", "REDIRECT"} {
		target, err := redirectMetadata([]byte(`<!--{"Title":"not a page", "` + key + `":"/doc/modules/managing-dependencies"}-->`))
		if err != nil || target != "/doc/modules/managing-dependencies" {
			t.Fatalf("key %s: target=%q error=%v", key, target, err)
		}
	}
	for _, b := range []string{`<!--{"Redirect":1}-->`, `<!--{"Redirect":"/a","redirect":"/b"}-->`, `<!--{"Redirect":"/a"`, `<!--{invalid}-->`} {
		if _, err := redirectMetadata([]byte(b)); err == nil {
			t.Fatalf("accepted malformed/ambiguous metadata %s", b)
		}
	}
}

func TestTourPublicCanonicalRoutes(t *testing.T) {
	g := repositoryGlobal(t)
	p, _ := packageByID(g, "tour-v1")
	catalog, err := i18n.BuildSourceCatalog(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Routes) != len(catalog.Pages)+2 || len(p.Routes) != 105 {
		t.Fatalf("public routes=%d catalog pages=%d", len(p.Routes), len(catalog.Pages))
	}
	routes := map[string]bool{}
	for _, r := range p.Routes {
		if !strings.HasPrefix(r, "/tour/") {
			t.Fatalf("not a public Tour route: %s", r)
		}
		routes[r] = true
	}
	for _, r := range []string{"/tour/", "/tour/list", "/tour/basics/1", "/tour/welcome/1"} {
		if !routes[r] {
			t.Fatalf("missing %s", r)
		}
	}
	if routes["/basics/1"] || routes["/welcome/1"] {
		t.Fatal("internal route exposed as canonical")
	}
	for _, page := range catalog.Pages {
		if strings.HasPrefix(page.Route, "/tour/") || !routes["/tour"+page.Route] {
			t.Fatal("internal catalog identity changed or public projection missing")
		}
	}
}

func TestGlobalScopeRejectsMutations(t *testing.T) {
	for _, mutation := range []func(*Global){func(g *Global) { g.Schema = "unknown" }, func(g *Global) { g.Packages[1].Routes = append(g.Packages[1].Routes, "/tour/") }, func(g *Global) { g.Sources[0].Package = "unknown" }, func(g *Global) { g.Sources[0].Kind = "unknown" }, func(g *Global) {
		g.Sources[0].Dependencies = append(g.Sources[0].Dependencies, Dependency{Kind: "asset", Path: "../x", SHA256: strings.Repeat("a", 64)})
	}, func(g *Global) {
		for i := range g.Sources {
			if g.Sources[i].Kind == "redirect" {
				g.Sources[i].Route = "/doc/security/fake"
				return
			}
		}
	}} {
		g := repositoryGlobal(t)
		mutation(g)
		if ValidateGlobal(*g) == nil {
			t.Fatal("accepted mutated global")
		}
	}
}

func incompleteState(g *Global) Locale {
	return Locale{Schema: LocaleSchema, Locale: "he", Packages: []Completion{}}
}

func completedTourState(t *testing.T) Locale {
	t.Helper()
	b, err := readRegular(repositoryRoot(t), LocalePath("he"))
	if err != nil {
		t.Fatal(err)
	}
	var l Locale
	if err := StrictJSON(b, &l); err != nil {
		t.Fatal(err)
	}
	return l
}

func TestLocaleScopeFailClosed(t *testing.T) {
	g := repositoryGlobal(t)
	l := incompleteState(g)
	if err := ValidateLocale(g, l); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []func(*Locale){func(l *Locale) { l.Packages[0].Routes = []string{"/tour/"} }, func(l *Locale) { l.Packages[0].EvidenceKind = "forged" }, func(l *Locale) { l.Packages[0].State = "incomplete" }, func(l *Locale) { l.Packages[0].Package = "unknown" }, func(l *Locale) { l.Packages[0].SourceIdentity = "stale" }, func(l *Locale) { l.Packages = append(l.Packages, l.Packages[0]) }} {
		l := completedTourState(t)
		mutation(&l)
		if ValidateLocale(g, l) == nil {
			t.Fatal("accepted unproven locale")
		}
	}
}

func TestSparseCompletionIndependentGlobalPackages(t *testing.T) {
	for _, id := range []string{"learn-docs-v1", "site-v2-shell", "future-v1", "tour-v1"} {
		t.Run(id, func(t *testing.T) {
			g := repositoryGlobal(t)
			l := completedTourState(t)
			before, _ := Encode(l)
			if id == "future-v1" {
				p := Package{ID: id, Activation: "requires-future-gates", Routes: []string{}, Surfaces: []string{}, Dependencies: []Dependency{}, SourceIdentity: strings.Repeat("a", 64)}
				p.Identity = identity(p)
				g.Packages = append(g.Packages, p)
				g.Versions[1].Packages = append(g.Versions[1].Packages, id)
			} else {
				for i := range g.Packages {
					if g.Packages[i].ID == id {
						g.Packages[i].SourceIdentity = strings.Repeat("a", 64)
						g.Packages[i].Identity = ""
						g.Packages[i].Identity = identity(g.Packages[i])
					}
				}
			}
			g.Identity = ""
			g.Identity = identity(g)
			if err := ValidateGlobal(*g); err != nil {
				t.Fatal(err)
			}
			err := ValidateLocale(g, l)
			_, currentErr := CheckLocale(repositoryRoot(t), g, "he")
			if id == "tour-v1" {
				if err == nil || currentErr == nil {
					t.Fatal("changed completed Tour scope did not become stale")
				}
				return
			}
			if err != nil || currentErr != nil {
				t.Fatalf("unfinished/additive package staled Tour: structural=%v current=%v", err, currentErr)
			}
			after, _ := Encode(l)
			if !bytes.Equal(before, after) || len(l.Packages) != 1 {
				t.Fatal("locale progress mechanically rewritten")
			}
			states := PackageStates(g, &l)
			if len(states) != len(g.Packages) || states[len(states)-1].State != "incomplete" {
				t.Fatal("missing package not inferred incomplete")
			}
			// A real global future package is still not an authorized completion.
			p, _ := packageByID(g, id)
			l.Packages = append(l.Packages, Completion{Package: id, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, EvidenceKind: "legacy-tour-closure/v1"})
			if ValidateLocale(g, l) == nil {
				t.Fatal("future/non-Tour completion bypassed gate")
			}
		})
	}
}

func TestSnapshotRejectsUnsafeEntries(t *testing.T) {
	for _, names := range [][]string{{"../x"}, {"x", "x"}} {
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		for _, name := range names {
			h := &zip.FileHeader{Name: name}
			h.SetMode(0644)
			w, err := z.CreateHeader(h)
			if err != nil {
				t.Fatal(err)
			}
			w.Write([]byte("x"))
		}
		z.Close()
		if _, err := snapshotFS(b.Bytes()); err == nil {
			t.Fatal("unsafe snapshot accepted")
		}
	}
}

func TestInventoryChangeClassification(t *testing.T) {
	a := Source{Path: "a", Kind: "page", Route: "/a", SHA256: "old", Dependencies: []Dependency{}}
	b := a
	b.Kind = "data"
	b.Route = ""
	b.SHA256 = "new"
	b.Dependencies = []Dependency{{Kind: "data", Path: "b", SHA256: "x"}}
	want := []string{"content_changed", "role_changed", "route_changed", "dependency_changed"}
	changes := CompareInventory([]Source{a}, []Source{b, {Path: "b"}})
	if !reflect.DeepEqual(changes[0].Changes, want) || changes[1].Changes[0] != "added" {
		t.Fatal(changes)
	}
	if CompareInventory([]Source{a}, nil)[0].Changes[0] != "removed" {
		t.Fatal("removed")
	}
}

func TestLegacyBootstrapPreflightAndNoOverwrite(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "production"), 0755)
	profiles := []liveProfile{}
	for _, l := range tour.LanguageRegistry() {
		if !l.Official {
			profiles = append(profiles, liveProfile{Locale: l.Locale, State: "live", Hostname: strings.TrimSuffix(strings.TrimPrefix(l.URL, "https://"), "/"), PublicURL: l.URL})
		}
	}
	write := func() {
		b, _ := json.Marshal(map[string]any{"schema": "go-tour-i18n/production-identity/v1", "locales": profiles})
		if err := os.WriteFile(filepath.Join(root, "production/identity.json"), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write()
	if _, err := LiveLocales(root); err != nil {
		t.Fatal(err)
	}
	profiles[0].State = "first-production"
	write()
	if _, err := LiveLocales(root); err == nil {
		t.Fatal("accepted non-live")
	}
	profiles[0].State = "live"
	profiles[1] = profiles[0]
	write()
	if _, err := LiveLocales(root); err == nil {
		t.Fatal("accepted duplicate")
	}
	if _, err := BootstrapTour(root, repositoryGlobal(t)); err == nil {
		t.Fatal("fabricated closure")
	}
	if _, err := os.Stat(filepath.Join(root, "locales")); !os.IsNotExist(err) {
		t.Fatal("partial bootstrap")
	}
	if err := WriteNew(root, "new.json", []byte("old")); err != nil {
		t.Fatal(err)
	}
	if WriteNew(root, "new.json", []byte("new")) == nil {
		t.Fatal("overwrote authority")
	}
	b, _ := os.ReadFile(filepath.Join(root, "new.json"))
	if string(b) != "old" {
		t.Fatal("changed")
	}
}

func TestLegacyTourCurrentAndIndependentFutureState(t *testing.T) {
	g := repositoryGlobal(t)
	root := repositoryRoot(t)
	l, err := CheckLocale(root, g, "he")
	if err != nil {
		t.Fatal(err)
	}
	states := PackageStates(g, l)
	if len(l.Packages) != 1 || l.Packages[0].State != "complete" || len(l.Packages[0].Evidence) < 122 || states[1].State != "incomplete" || states[2].State != "incomplete" {
		t.Fatalf("unexpected bootstrap")
	}
	b, err := readRegular(root, LocalePath("he"))
	if err != nil {
		t.Fatal(err)
	}
	var mutated Locale
	StrictJSON(b, &mutated)
	mutated.Packages[0].Evidence[0].SHA256 = "forged"
	if ValidateLocale(g, mutated) == nil {
		t.Fatal("forged evidence")
	}
}

func TestHistoricalLanguageEvidenceBoundary(t *testing.T) {
	a := i18n.LocaleSurfaceReviewAInputs{UILocaleSHA256: "ui", GlossarySHA256: "glossary", ArticleMetadataSHA256: "article", CourseMetadataSHA256: "course", CatalogSourceSHA256: "catalog", CourseSourceDescriptionsSHA256: "descriptions", CourseSourceDescriptionReviewSHA256: "source-review", ProjectConfigSHA256: "project", SEOConfigSHA256: "seo", ProductionPublicIdentitySHA256: "target"}
	current := a
	current.UIEnglishSHA256 = "new-shared-English"
	current.LanguagesConfigSHA256 = "appended-registry"
	if !historicalLanguageInputsMatch(a, current, 3) {
		t.Fatal("historical completion incorrectly becomes new review")
	}
	for _, field := range []string{"UILocaleSHA256", "GlossarySHA256", "ArticleMetadataSHA256", "CourseMetadataSHA256", "CatalogSourceSHA256", "CourseSourceDescriptionsSHA256", "CourseSourceDescriptionReviewSHA256", "ProjectConfigSHA256", "SEOConfigSHA256", "ProductionPublicIdentitySHA256"} {
		changed := current
		reflect.ValueOf(&changed).Elem().FieldByName(field).SetString("changed")
		if historicalLanguageInputsMatch(a, changed, 3) {
			t.Fatalf("unverified %s accepted", field)
		}
	}
	g := repositoryGlobal(t)
	l, err := CheckLocale(repositoryRoot(t), g, "ar")
	if err != nil {
		t.Fatal(err)
	}
	if l.Packages[0].LegacySurfaceState != "historical-verified" {
		t.Fatal("historical receipt misrepresented as current gate")
	}
}

func TestGlossaryCompatibilityFormalSchema(t *testing.T) {
	b, err := readRegular(repositoryRoot(t), "data/glossary-compatibility.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]json.RawMessage
	if err := StrictJSON(b, &schema); err != nil {
		t.Fatal(err)
	}
	var defs map[string]json.RawMessage
	json.Unmarshal(schema["$defs"], &defs)
	for _, name := range []string{"reference", "delta", "context", "scope"} {
		if defs[name] == nil {
			t.Fatalf("missing executable evidence definition %s", name)
		}
	}
	var id string
	json.Unmarshal(schema["$id"], &id)
	if id != i18n.GlossaryCompatibilitySchema {
		t.Fatal("compatibility schema/code identity differs")
	}
	if !bytes.Contains(b, []byte(`"mandatory", "preferred", "forbidden", "keep"`)) {
		t.Fatal("missing terminology categories")
	}
}

func TestLegacyPlanFullQCEvidence(t *testing.T) {
	states, err := LegacyPlan(repositoryRoot(t), repositoryGlobal(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 65 {
		t.Fatal("incomplete cohort")
	}
	for _, l := range states {
		found := false
		for _, r := range l.Packages[0].Evidence {
			if strings.HasSuffix(r.Path, "/finalization.json") {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s missing QC closure", l.Locale)
		}
	}
	full := i18n.QualityCheckFinalization{SchemaVersion: 1, EvidenceType: i18n.QualityCheckFinalizationEvidenceType, Locale: "he", SnapshotID: "qc", GlossarySHA256: "glossary", Units: []i18n.QualityCheckFinalizationUnit{{UnitID: "unit", Rating: "A", Snapshot: i18n.QualityCheckSnapshotUnit{UnitID: "unit", SourceSHA256: "source", CandidateSHA256: "candidate"}}}}
	current := map[string]canonicalQCIdentity{"unit": {"source", "candidate"}}
	if !finalizationMatchesCanonical(full, "he", "qc", "glossary", current) {
		t.Fatal("valid closure rejected")
	}
	full.Units[0].Rating = "B"
	if finalizationMatchesCanonical(full, "he", "qc", "glossary", current) {
		t.Fatal("non A-only accepted")
	}
	full.Units[0].Rating = "A"
	full.Units[0].Snapshot.CandidateSHA256 = "changed"
	if finalizationMatchesCanonical(full, "he", "qc", "glossary", current) {
		t.Fatal("unreviewed canonical candidate accepted")
	}
}

func TestHistoricalPromotionEOFProjection(t *testing.T) {
	root := t.TempDir()
	b := []byte("reviewed\n\n")
	if err := WriteNew(root, "selected.txt", b); err != nil {
		t.Fatal(err)
	}
	f := i18n.QualityCheckFinalization{Units: []i18n.QualityCheckFinalizationUnit{{Snapshot: i18n.QualityCheckSnapshotUnit{CandidatePath: "selected.txt", CandidateSHA256: digest(b)}}}}
	p, err := finalizationWithPromotionEOF(root, f)
	if err != nil {
		t.Fatal(err)
	}
	if p.Units[0].Snapshot.CandidateSHA256 != digest([]byte("reviewed\n")) || f.Units[0].Snapshot.CandidateSHA256 != digest(b) {
		t.Fatal("normalization changed history")
	}
	f.Units[0].Snapshot.CandidateSHA256 = digest([]byte("other\n"))
	if _, err := finalizationWithPromotionEOF(root, f); err == nil {
		t.Fatal("unreviewed bytes accepted")
	}
}
