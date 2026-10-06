package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const compatibilityFixtureGlossary = "locale: zh-CN\nmandatory:\n  Go: Go\nkeep:\n  - Go\n  - gofmt\n"

func compatibilityFixture(t *testing.T) (string, *Catalog, string) {
	t.Helper()
	root := t.TempDir()
	copyBundleAuthority(t, root, []string{"locales/zh-CN/locale.json"})
	catalog := retranslationTestCatalog(3)
	catalog.Pages[0].Source = []byte("* Page\n\nUse `Go` on this page. Transaction data. Run gofmt afterward.\n")
	catalog.Pages[0].SourceSHA256 = sum(catalog.Pages[0].Source)
	writeSnapshotSource(t, root, "locales/zh-CN/glossary.yaml", []byte(compatibilityFixtureGlossary))
	if _, _, err := RecordGlossaryReview(root, "zh-CN", "original", "independent-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	addProcessedPromotionBatch(t, root, catalog, "codex-zh-CN-001", []string{"lesson/1", "lesson/2", "lesson/3"})
	materializeSnapshotSources(t, root, catalog)
	article, _ := os.ReadFile(filepath.Join(root, "_content/tour/lesson.article"))
	writeSnapshotSource(t, root, "_content/tour/lesson.article", append([]byte("Lesson\nLanguage introduction\n\n"), article...))
	snapshot, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-001"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecordQualityCheckResults(root, catalog, QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "qc-001", UnitIDs: []string{"lesson/1", "lesson/2", "lesson/3"}, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := FinalizeQualityCheck(root, catalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-001"}); err != nil {
		t.Fatal(err)
	}
	for _, l := range []string{"en", "zh-CN"} {
		writeSnapshotSource(t, root, "internal/tour/ui/"+l+".json", []byte(`{"locale":"`+l+`","html_lang":"`+l+`","messages":{"test.one":{"kind":"plain","text":"Welcome"}}}`))
	}
	writeSnapshotSource(t, root, "locales/zh-CN/article-metadata.json", []byte(`{"locale":"zh-CN","articles":[{"article":"lesson.article","title":"Lesson","subtitle":"Language introduction"}]}`))
	metadata := &CourseMetadata{SchemaVersion: 1, Locale: "zh-CN", GeneratorContract: CourseMetadataGeneratorContract}
	statuses := []Status{}
	for _, page := range catalog.Pages {
		s := snapshotUnit(t, snapshot, page.ID)
		target, err := os.ReadFile(filepath.Join(root, s.CandidatePath))
		if err != nil {
			t.Fatal(err)
		}
		path := canonicalCandidatePath("zh-CN", page.ID)
		writeSnapshotSource(t, root, path, target)
		metadata.Pages = append(metadata.Pages, CoursePageMetadata{PageID: page.ID, Route: page.Route, Description: courseDescription(page.ID), SourceSHA256: page.SourceSHA256, TargetSHA256: sum(target), GlossarySHA256: snapshot.GlossarySHA256, Generation: CourseMetadataGeneration{Provider: "fixture", Model: "fixture-model", PromptVersion: CourseMetadataPromptVersion, GeneratedAt: "2026-08-25T12:00:00Z"}})
		statuses = append(statuses, Status{UnitID: page.ID, State: "ready", Attempts: 1, SourceSHA256: page.SourceSHA256, CandidatePath: path})
	}
	writeCourseMetadataFixture(t, root, "zh-CN", metadata)
	if err := writeStatuses(filepath.Join(root, "locales/zh-CN/status.tsv"), statuses); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"internal/tour/languages.go", "internal/tour/tour.go", "internal/tour/production.go", "internal/tour/project.go", "internal/tour/seo.go", "_content/js/playground.js", "_content/tour/static/js/app.js", "_content/tour/template/index.tmpl", "_content/tour/static/partials/index.html"} {
		writeSnapshotSource(t, root, path, []byte("package tour\nconst Config = \"Tour\"\n"))
	}
	return root, catalog, snapshot.GlossarySHA256
}

func compatibilityChange(t *testing.T, root string, data string, id string) string {
	t.Helper()
	writeSnapshotSource(t, root, "locales/zh-CN/glossary.yaml", []byte(data))
	if RequireCurrentGlossaryReview(root, "zh-CN") == nil {
		t.Fatal("old complete Review accepted changed bytes")
	}
	if _, _, err := RecordGlossaryReview(root, "zh-CN", id, "independent-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	return sum([]byte(data))
}

func TestGlossaryCompatibilitySemanticNormalization(t *testing.T) {
	old := []byte(compatibilityFixtureGlossary + "preferred:\n  transaction: 事务\nforbidden:\n  - 错误\n")
	next := []byte("# representation only\nlocale: zh-CN\nforbidden:\n  - 错误\npreferred:\n  transaction: 事务\nkeep:\n  - gofmt\n  - Go\nmandatory:\n  Go: Go\n")
	delta, err := glossarySemanticDelta("zh-CN", old, next)
	if err != nil || len(delta) != 0 {
		t.Fatalf("delta=%v error=%v", delta, err)
	}
	for _, s := range []string{compatibilityFixtureGlossary + "  - Go\n", strings.Replace(compatibilityFixtureGlossary, "zh-CN", "fr-FR", 1), compatibilityFixtureGlossary + "terms:\n  x: y\n", compatibilityFixtureGlossary + "mandatory:\n  Go: Changed\n"} {
		if _, err := normalizeCompatibilityGlossary("zh-CN", []byte(s)); err == nil {
			t.Fatalf("unsafe normalization accepted %q", s)
		}
	}
}

func TestGlossaryCompatibilityImpactVisibility(t *testing.T) {
	value := "new"
	old := "old"
	tests := []struct {
		name, parser, source, target, category, term string
		removed, compatible                          bool
	}{
		{"mandatory one", "present-visible/v1", "* Page\n\nTransaction data.\n", "* 页面\n\n数据。\n", "mandatory", "transaction", false, false},
		{"preferred present", "plain-visible/v1", "Transactions are useful", "事务", "preferred", "transaction", false, false},
		{"docs absent", "plain-visible/v1", "Go program", "Go 程序", "mandatory", "SQL transaction", false, true},
		{"keep present", "plain-visible/v1", "FIPS 140 standard", "标准", "keep", "FIPS 140", false, false},
		{"forbidden visible", "present-visible/v1", "* Page\n\nText\n", "* 页面\n\n错误\n", "forbidden", "错误", false, false},
		{"plain literal backticks", "plain-visible/v1", "Literal label", "`错误`", "forbidden", "错误", false, false},
		{"plain URL", "plain-visible/v1", "URL", "https://example.com/错误", "forbidden", "错误", false, true},
		{"protected", "present-visible/v1", "* Page\n\n`错误` [[https://example.com/错误][Label]]\n\n.code 错误.go\n\n\t错误\n", "* 页面\n\n`错误` [[https://example.com/错误][标签]]\n\n.code 错误.go\n\n\t错误\n", "forbidden", "错误", false, true},
		{"link label", "present-visible/v1", "* Page\n\n[[https://example.com][Label]]\n", "* 页面\n\n[[https://example.com][错误]]\n", "forbidden", "错误", false, false},
		{"Go code", "go-comment-visible/v1", "package main\n// Translate this comment.\nvar 错误 = \"错误\"\n", "package main\n// 翻译注释。\nvar 错误 = \"错误\"\n", "forbidden", "错误", false, true},
		{"Go comment", "go-comment-visible/v1", "package main\n// Translate this comment.\n", "package main\n// 错误。\n", "forbidden", "错误", false, false},
		{"Unicode boundary", "plain-visible/v1", "Word", "badly bad's", "forbidden", "bad", false, true},
		{"removed", "unknown", "anything", "错误", "forbidden", "错误", true, true},
		{"unknown", "unknown", "", "", "preferred", "missing", false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := GlossarySemanticDelta{Category: test.category, Term: test.term, New: &value}
			if test.removed {
				d.Old = &old
				d.New = nil
			}
			ok, reasons := classifyGlossaryContext(GlossaryCompatibilityContext{Parser: test.parser, Source: test.source, Target: test.target}, []GlossarySemanticDelta{d})
			if ok != test.compatible {
				t.Fatalf("compatible=%v reasons=%v", ok, reasons)
			}
		})
	}
}

func TestGlossaryCompatibilityByteOnlyLineagePreservesHistory(t *testing.T) {
	root, catalog, g1 := compatibilityFixture(t)
	paths := []string{"data/glossary-reviews/zh-CN/original.review.json", "data/quality-check-snapshots/zh-CN/qc-001/manifest.json", "data/quality-check-snapshots/zh-CN/qc-001/quality-check-results.json", "data/quality-check-snapshots/zh-CN/qc-001/finalization.json"}
	before := map[string][]byte{}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		before[path] = data
	}
	g2 := compatibilityChange(t, root, "# byte change\n"+compatibilityFixtureGlossary, "review-2")
	e, path, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Delta) != 0 || len(e.Affected) != 0 {
		t.Fatalf("byte change: %+v", e)
	}
	if err := CheckGlossaryCompatibilityEvidence(root, "zh-CN", path, catalog); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
		t.Fatal(err)
	}
	scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
	if err != nil || scope.CarryForwardCount != 3 {
		t.Fatalf("scope=%+v err=%v", scope, err)
	}
	// The first new result records lineage without fabricating carried reviewer A.
	if _, err := RecordQualityCheckResults(root, catalog, QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001", UnitIDs: []string{"lesson/1"}, Rating: "A"}); err != nil {
		t.Fatal(err)
	}
	f, _, err := FinalizeQualityCheck(root, catalog, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: "qc-002"})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Units[1].CompatibilityChain) != 1 || f.Units[1].SourceSnapshotID != "qc-001" {
		t.Fatalf("missing carried identity: %+v", f.Units[1])
	}
	g3 := compatibilityChange(t, root, "# another byte change\n"+compatibilityFixtureGlossary, "review-3")
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g2, catalog); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{g1, g3}, {g1, g2}, {g2, g3}} {
		r := ResolveGlossaryCompatibility(root, "zh-CN", pair[0], pair[1], "tu:lesson/2", catalog)
		if r.Status != "compatible" {
			t.Fatalf("chain %v: %+v", pair, r)
		}
	}
	if _, err := VerifyQualityCheckFinalization(root, catalog, "zh-CN", "qc-002"); err != nil {
		t.Fatal(err)
	}
	for path, data := range before {
		after, _ := os.ReadFile(filepath.Join(root, path))
		if !bytes.Equal(data, after) {
			t.Fatalf("history changed: %s", path)
		}
	}
	// Context mutation invalidates every hop; it cannot be carried on a raw absence claim.
	uiPath := filepath.Join(root, "internal/tour/ui/zh-CN.json")
	data, _ := os.ReadFile(uiPath)
	writeSnapshotSource(t, root, "internal/tour/ui/zh-CN.json", bytes.ReplaceAll(data, []byte("Welcome"), []byte("Changed")))
	if CheckGlossaryCompatibilityEvidence(root, "zh-CN", path, catalog) == nil {
		t.Fatal("changed exact set stayed current")
	}
	if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g3, "tu:lesson/2", catalog); r.Status != "stale evidence" {
		t.Fatalf("stale chain=%+v", r)
	}
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g2, catalog); err != nil {
		t.Fatal(err)
	}
	if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g3, "tu:lesson/2", catalog); r.Status != "compatible" || len(r.Chain) != 2 {
		t.Fatalf("re-assessed exact context chain=%+v", r)
	}
}

func TestGlossaryCompatibilityPreciseUnitAndSEO(t *testing.T) {
	for _, category := range []string{"mandatory", "preferred", "keep"} {
		t.Run(category, func(t *testing.T) {
			root, catalog, g1 := compatibilityFixture(t)
			extra := category + ":\n  transaction: 交易\n"
			if category == "keep" {
				extra = "  - Transaction\n"
			}
			newData := compatibilityFixtureGlossary + extra
			if category == "mandatory" {
				newData = strings.Replace(compatibilityFixtureGlossary, "  Go: Go\n", "  Go: Go\n  transaction: 交易\n", 1)
			}
			g2 := compatibilityChange(t, root, newData, "changed")
			e, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog)
			if err != nil {
				t.Fatal(err)
			}
			if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g2, "tu:lesson/1", catalog); r.Status != "affected" {
				t.Fatalf("unit1=%+v", r)
			}
			if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g2, "tu:lesson/2", catalog); r.Status != "compatible" {
				t.Fatalf("unit2=%+v", r)
			}
			if courseGlossaryCompatibility(root, "zh-CN", catalog, g2)(mustCourseEntry(t, root, "lesson/1")) {
				t.Fatal("affected SEO page accepted")
			}
			if !courseGlossaryCompatibility(root, "zh-CN", catalog, g2)(mustCourseEntry(t, root, "lesson/2")) {
				t.Fatal("unaffected SEO page stale")
			}
			if len(e.Affected) == 0 {
				t.Fatal("no affected scopes")
			}
			if category == "keep" {
				if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err == nil {
					t.Fatal("affected keep protection bypassed revision")
				}
				return
			}
			if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
				t.Fatal(err)
			}
			scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
			if err != nil {
				t.Fatal(err)
			}
			if scope.CarryForwardCount != 2 || scope.PendingCount != 1 || scope.Pending[0].UnitID != "lesson/1" {
				t.Fatalf("scope=%+v", scope)
			}
		})
	}
}

func mustCourseEntry(t *testing.T, root, id string) CoursePageMetadata {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "locales/zh-CN/course-metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := decodeCourseMetadata(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range c.Pages {
		if e.PageID == id {
			return e
		}
	}
	t.Fatal("missing page")
	return CoursePageMetadata{}
}

func TestGlossaryCompatibilityDocsAddition(t *testing.T) {
	root, catalog, g1 := compatibilityFixture(t)
	g2 := compatibilityChange(t, root, strings.Replace(compatibilityFixtureGlossary, "  Go: Go\n", "  Go: Go\n  SQL transaction: SQL 事务\n", 1), "docs")
	e, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range e.Affected {
		if !strings.HasPrefix(s.ID, "surface:") {
			t.Fatalf("Docs-only affected mapped Tour artifact: %+v", s)
		}
	}
	for _, scope := range []string{"tu:lesson/1", "seo:lesson/2", "seo-page:lesson/2", "ui:test.one", "article:lesson.article"} {
		if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g2, scope, catalog); r.Status != "compatible" {
			t.Fatalf("%s: %+v", scope, r)
		}
	}
	if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g2, "*", catalog); r.Status != "affected" {
		t.Fatal("unmapped Surface silently accepted")
	}
}

func TestGlossaryCompatibilitySEORefreshPreservesUnchangedProvenance(t *testing.T) {
	root, catalog, g1 := compatibilityFixture(t)
	original2 := mustCourseEntry(t, root, "lesson/2")
	original3 := mustCourseEntry(t, root, "lesson/3")
	g2 := compatibilityChange(t, root, strings.Replace(compatibilityFixtureGlossary, "  Go: Go\n", "  Go: Go\n  transaction: 交易\n", 1), "changed")
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog); err != nil {
		t.Fatal(err)
	}
	descriptions, _ := json.Marshal(courseDescriptionsFile{Pages: []courseDescriptionEntry{{PageID: "lesson/1", Description: courseDescription("revised lesson/1")}}})
	data, stale, err := RefreshCourseMetadata(root, catalog, CourseMetadataRefreshOptions{Locale: "zh-CN", Provider: "fixture", Model: "fixture-new-model", GeneratedAt: "2026-10-06T12:00:00Z", Descriptions: descriptions})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stale, []string{"lesson/1"}) {
		t.Fatalf("wrong stale subset %v", stale)
	}
	writeSnapshotSource(t, root, "locales/zh-CN/course-metadata.json", data)
	if !reflect.DeepEqual(original2, mustCourseEntry(t, root, "lesson/2")) || !reflect.DeepEqual(original3, mustCourseEntry(t, root, "lesson/3")) {
		t.Fatal("unaffected historical generation rewritten")
	}
	if entry := mustCourseEntry(t, root, "lesson/1"); entry.GlossarySHA256 != g2 {
		t.Fatal("new replacement did not bind current glossary")
	}
	if _, err := LoadCourseMetadata(root, "zh-CN", catalog); err == nil {
		t.Fatal("changed asset reused old exact inventory")
	}
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCourseMetadata(root, "zh-CN", catalog); err != nil {
		t.Fatalf("mixed page generation SHA rejected: %v", err)
	}
}

func TestGlossaryCompatibilityRemovedKeepPreservesProtectedEvidence(t *testing.T) {
	root, catalog, g1 := compatibilityFixture(t)
	newGlossary := strings.Replace(compatibilityFixtureGlossary, "  - gofmt\n", "", 1)
	compatibilityChange(t, root, newGlossary, "relaxed")
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, catalog, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "qc-002"}); err != nil {
		t.Fatalf("constraint removal discarded trusted protected input: %v", err)
	}
	scope, err := BuildQualityCheckScope(root, catalog, QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "qc-002", PreviousSnapshotID: "qc-001"})
	if err != nil || scope.CarryForwardCount != 3 {
		t.Fatalf("scope=%+v err=%v", scope, err)
	}
}

func TestGlossaryCompatibilityStrictEvidenceAndArchive(t *testing.T) {
	root, catalog, g1 := compatibilityFixture(t)
	g2 := compatibilityChange(t, root, "# new\n"+compatibilityFixtureGlossary, "next")
	e, path, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, path))
	for _, bad := range [][]byte{append(append([]byte{}, data...), []byte("{}")...), bytes.Replace(data, []byte(`"schema":`), []byte(`"unknown":0,"schema":`), 1), bytes.Replace(data, []byte(`"schema":`), []byte(`"schema":"duplicate","schema":`), 1), bytes.Replace(data, []byte(`"locale": "zh-CN"`), []byte(`"locale": "fr-FR"`), 1)} {
		writeSnapshotSource(t, root, path, bad)
		if CheckGlossaryCompatibilityEvidence(root, "zh-CN", path, catalog) == nil {
			t.Fatal("malformed evidence accepted")
		}
		if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g2, "tu:lesson/2", catalog); r.Status == "compatible" {
			t.Fatal("malformed resolver accepted")
		}
	}
	writeSnapshotSource(t, root, path, data)
	if err := writeCompatibilityImmutable(root, path, data); err != nil {
		t.Fatal(err)
	}
	if err := writeCompatibilityImmutable(root, path, []byte("different")); err == nil {
		t.Fatal("immutable overwrite allowed")
	}
	if _, err := readCompatibilityFile(root, "../outside"); err == nil {
		t.Fatal("traversal allowed")
	}
	link := filepath.Join(root, "data/link")
	if err := os.Symlink(filepath.Join(root, "locales"), link); err != nil {
		t.Fatal(err)
	}
	if _, err := readCompatibilityFile(root, "data/link/zh-CN/glossary.yaml"); err == nil {
		t.Fatal("parent symlink allowed")
	}
	old, _ := readArchivedGlossary(root, "zh-CN", g1)
	if !bytes.Equal(old, []byte(compatibilityFixtureGlossary)) {
		t.Fatal("old bytes unavailable")
	}
	// Tampering with reasons is detected even if JSON remains structurally valid.
	e.Compatible[0].Reasons = []string{"invented"}
	bad, _ := json.Marshal(e)
	writeSnapshotSource(t, root, path, bad)
	if CheckGlossaryCompatibilityEvidence(root, "zh-CN", path, catalog) == nil {
		t.Fatal("unproved classification accepted")
	}
}

func TestGlossaryCompatibilityConfigProjection(t *testing.T) {
	base := []byte("package tour\nconst Canonical = \"/tour/\"\nfunc route() string { return Canonical }\n")
	old, err := projectTourConfig(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"\n// unrelated comment\n", "\nfunc learnRoute() string { return \"/learn/\" }\n", "\nconst DocsLabel = \"Docs\"\n"} {
		p, err := projectTourConfig(append(append([]byte{}, base...), []byte(suffix)...))
		if err != nil || !tourConfigProjectionCompatible(old, p) {
			t.Fatalf("unrelated change stale: %v", err)
		}
	}
	for _, data := range []string{strings.Replace(string(base), "/tour/", "/learn/", 1), string(base) + "\nfunc init() {}\n", string(base) + "\nvar Runtime = 1\n", string(base) + "\n//go:linkname x y\nfunc x() {}\n"} {
		p, err := projectTourConfig([]byte(data))
		if err == nil && tourConfigProjectionCompatible(old, p) {
			t.Fatalf("semantic/unknown config accepted: %s", data)
		}
	}
	root, catalog := surfaceReviewTestRoot(t)
	gate, _, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "config-review", "reviewer", catalog)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise an immutable historical v3 gate and its independent baseline.
	gate.SchemaVersion = localeSurfaceReviewASchemaVersionV3
	gate.Inputs.TourConfigProjection = nil
	gate.Inputs.TourLanguageContextSHA256 = ""
	gatePath, _ := LocaleSurfaceReviewAGatePath(root, "zz-ZZ", gate.ReviewID)
	gateData, _ := marshalGlossaryReviewJSON(gate)
	if err := os.WriteFile(gatePath, gateData, 0644); err != nil {
		t.Fatal(err)
	}
	path, err := RecordTourSurfaceConfigBaseline(root, "zz-ZZ", gate.ReviewID)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := RecordTourSurfaceConfigBaseline(root, "zz-ZZ", gate.ReviewID); err != nil || second != path {
		t.Fatalf("baseline idempotency %s %v", second, err)
	}
	oldProject, _ := os.ReadFile(filepath.Join(root, "internal/tour/project.go"))
	writeSnapshotSource(t, root, "internal/tour/project.go", append(oldProject, []byte("func unrelated() string { return \"Learn\" }\n")...))
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zz-ZZ", gate.ReviewID, catalog); err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, "internal/tour/seo.go", []byte("package tour\nconst TourRoute = \"/learn/\"\n"))
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zz-ZZ", gate.ReviewID, catalog); err == nil {
		t.Fatal("canonical projection change accepted")
	}
	after, _ := os.ReadFile(gatePath)
	if !reflect.DeepEqual(gateData, after) {
		t.Fatal("historical gate rewritten")
	}
}

func TestGlossaryCompatibilityConfigProjectionRequiresArchivedBytes(t *testing.T) {
	root, catalog := surfaceReviewTestRoot(t)
	gate, path, err := RecordLocaleSurfaceReviewA(root, "zz-ZZ", "archived-config", "reviewer", catalog)
	if err != nil {
		t.Fatal(err)
	}
	originalGate, _ := os.ReadFile(path)
	if gate.SchemaVersion != localeSurfaceReviewASchemaVersionV4 || !recordedTourSurfaceConfigProjectionValid(root, gate.Inputs) {
		t.Fatal("new gate lacks verified original config bytes")
	}
	project, _ := os.ReadFile(filepath.Join(root, "internal/tour/project.go"))
	writeSnapshotSource(t, root, "internal/tour/project.go", append(project, []byte("func docsRoute() string { return \"/docs/\" }\n")...))
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zz-ZZ", gate.ReviewID, catalog); err != nil {
		t.Fatalf("unrelated additive config invalidates v4 projection: %v", err)
	}
	// Corrupt only a temporary fixture archive: an embedded projection cannot
	// substitute for the exact original config bytes.
	archive := filepath.Join(root, "data/surface-config-history", gate.Inputs.ProjectConfigSHA256+".go")
	if err := os.WriteFile(archive, []byte("package tour\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zz-ZZ", gate.ReviewID, catalog); err == nil {
		t.Fatal("embedded projection accepted corrupted original config archive")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(originalGate, after) {
		t.Fatal("config compatibility rewrote original gate")
	}
}

func TestGlossaryCompatibilitySurfaceAndGenerationCurrentness(t *testing.T) {
	root, catalog, g1 := compatibilityFixture(t)
	registryRoot, _ := surfaceReviewTestRoot(t)
	for _, path := range []string{"internal/tour/languages.go", "production/identity.json"} {
		data, err := os.ReadFile(filepath.Join(registryRoot, path))
		if err != nil {
			t.Fatal(err)
		}
		writeSnapshotSource(t, root, path, bytes.ReplaceAll(data, []byte("zz-ZZ"), []byte("zh-CN")))
	}
	gate, gatePath, err := RecordLocaleSurfaceReviewA(root, "zh-CN", "surface-original", "independent-reviewer", catalog)
	if err != nil {
		t.Fatal(err)
	}
	gateBytes, _ := os.ReadFile(gatePath)
	originalPackage, _, err := ExportLocaleSurfaceReviewPackage(root, "zh-CN", catalog)
	if err != nil {
		t.Fatal(err)
	}
	// New Generation remains bound to the entire current glossary even though
	// the prior reviewed language becomes compatible through a separate chain.
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	if _, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", BatchID: "codex-zh-CN-002", UnitIDs: []string{"lesson/2"}, Limit: 1, AllowReexport: true}); err != nil {
		t.Fatal(err)
	}
	bundle, _, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: "codex-zh-CN-002"})
	if err != nil {
		t.Fatal(err)
	}
	g2 := compatibilityChange(t, root, "# representation\n"+compatibilityFixtureGlossary, "next")
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zh-CN", gate.ReviewID, catalog); err != nil {
		t.Fatal(err)
	}
	currentPackage, _, err := ExportLocaleSurfaceReviewPackage(root, "zh-CN", catalog)
	if err != nil {
		t.Fatal(err)
	}
	historicalPackage, err := HistoricalTourSurfacePackage(root, "zh-CN", gate.ReviewID, g1, currentPackage, catalog)
	if err != nil || !bytes.Equal(originalPackage, historicalPackage) {
		t.Fatalf("historical closure context reconstruction error=%v", err)
	}
	if _, _, err := currentGenerationBundle(root, catalog, bundle); err == nil {
		t.Fatal("old Generation input accepted through downstream compatibility")
	}
	if _, _, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: "codex-zh-CN-002"}); err != nil {
		t.Fatalf("new full current Generation bundle rejected: %v", err)
	}
	if _, err := LoadCourseMetadata(root, "zh-CN", catalog); err != nil {
		t.Fatalf("all compatible SEO loader stale: %v", err)
	}
	g3 := compatibilityChange(t, root, strings.Replace(compatibilityFixtureGlossary, "  Go: Go\n", "  Go: Go\n  transaction: 交易\n", 1), "affected")
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g2, catalog); err != nil {
		t.Fatal(err)
	}
	if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g3, "tu:lesson/1", catalog); r.Status != "affected" {
		t.Fatalf("affected intermediate bypassed: %+v", r)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zh-CN", gate.ReviewID, catalog); err == nil {
		t.Fatal("affected surface accepted")
	}
	after, _ := os.ReadFile(gatePath)
	if !bytes.Equal(gateBytes, after) {
		t.Fatal("historical Surface gate modified")
	}
}

func TestGlossaryCompatibilityLineageMissingStaleAndAmbiguous(t *testing.T) {
	root, catalog, g1 := compatibilityFixture(t)
	g2 := compatibilityChange(t, root, "# next\n"+compatibilityFixtureGlossary, "next")
	if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g2, "tu:lesson/2", catalog); r.Status != "missing evidence" {
		t.Fatalf("missing=%+v", r)
	}
	e, path, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(filepath.Join(root, path))
	g3 := compatibilityChange(t, root, "# third\n"+compatibilityFixtureGlossary, "third")
	next, nextPath, err := AssessGlossaryCompatibility(root, "zh-CN", g2, catalog)
	if err != nil {
		t.Fatal(err)
	}
	// Corrupt an actual predecessor: strict current-check must fail closed.
	writeSnapshotSource(t, root, path, []byte("{}"))
	if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g3, "tu:lesson/2", catalog); r.Status != "stale evidence" {
		t.Fatalf("stale=%+v", r)
	}
	writeSnapshotSource(t, root, path, original)
	// A missing referenced predecessor cannot be repaired by guessing a route.
	wrong := *next
	wrong.Lineage[0].Path = "data/glossary-compatibility/zh-CN/" + strings.Repeat("0", 64) + ".json"
	wrong.Identity = ""
	wrong.Identity = sum(mustJSON(wrong))
	wrongPath, _ := glossaryCompatibilityEvidencePath("zh-CN", wrong.Identity)
	bad, _ := marshalGlossaryReviewJSON(wrong)
	writeSnapshotSource(t, root, wrongPath, bad)
	if CheckGlossaryCompatibilityEvidence(root, "zh-CN", wrongPath, catalog) == nil {
		t.Fatal("missing predecessor accepted")
	}
	if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g3, "tu:lesson/2", catalog); r.Status != "stale evidence" {
		t.Fatalf("invalid referenced evidence ignored: %+v", r)
	}
	// Remove only the deliberately malformed fixture to isolate branch behavior.
	if err := os.Remove(filepath.Join(root, wrongPath)); err != nil {
		t.Fatal(err)
	}
	// A different target branch is always ambiguous.
	branch := *e
	branch.NewGlossary = next.NewGlossary
	branch.NewFullReview = next.NewFullReview
	branch.Lineage = []GlossaryArchiveReference{}
	rebuilt, err := buildGlossaryCompatibilityEvidence(root, "zh-CN", g1, g3, branch.NewFullReview, e.Contexts, nil)
	if err != nil {
		t.Fatal(err)
	}
	branchPath, _ := glossaryCompatibilityEvidencePath("zh-CN", rebuilt.Identity)
	bytes, _ := marshalGlossaryReviewJSON(rebuilt)
	writeSnapshotSource(t, root, branchPath, bytes)
	if r := ResolveGlossaryCompatibility(root, "zh-CN", g1, g3, "tu:lesson/2", catalog); r.Status != "stale evidence" || r.Reason != "ambiguous_lineage" {
		t.Fatalf("branch=%+v", r)
	}
	_ = nextPath
}

func TestGlossaryCompatibilityArchiveCurrentCohort(t *testing.T) {
	root := repoRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "data/glossary-history"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 65 {
		t.Fatalf("archive locale count=%d", len(entries))
	}
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			locale := entry.Name()
			current, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
			if err != nil {
				t.Fatal(err)
			}
			archived, err := readArchivedGlossary(root, locale, sum(current))
			if err != nil || !bytes.Equal(current, archived) {
				t.Fatalf("archive bytes error=%v", err)
			}
			if err := RequireCurrentGlossaryReview(root, locale); err != nil {
				t.Fatal(err)
			}
			if _, err := normalizeCompatibilityGlossary(locale, current); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGlossaryCompatibilityConfigBaselineCohort(t *testing.T) {
	root := repoRoot(t)
	paths, err := filepath.Glob(filepath.Join(root, "data/locale-surface-reviews/*/*.config-baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 83 {
		t.Fatalf("config baseline count=%d", len(paths))
	}
	locales := map[string]bool{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var baseline TourSurfaceConfigBaseline
		if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &baseline); err != nil {
			t.Fatal(err)
		}
		gateData, err := readCompatibilityFile(root, baseline.Gate.Path)
		if err != nil {
			t.Fatal(err)
		}
		var gate LocaleSurfaceReviewAGate
		if err := decodeStrictCourseSourceDescriptionReviewJSON(gateData, &gate); err != nil {
			t.Fatal(err)
		}
		if !historicalTourSurfaceConfigCompatible(root, baseline.Locale, gate, gateData) {
			t.Fatalf("invalid config baseline %s", path)
		}
		locales[baseline.Locale] = true
	}
	if len(locales) != 65 {
		t.Fatalf("config baseline locale count=%d", len(locales))
	}
}

func TestGlossaryCompatibilityHistoricalSurfaceCannotApproveModifiedOtherText(t *testing.T) {
	root, catalog, g1 := compatibilityFixture(t)
	registryRoot, _ := surfaceReviewTestRoot(t)
	for _, path := range []string{"internal/tour/languages.go", "production/identity.json"} {
		data, err := os.ReadFile(filepath.Join(registryRoot, path))
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.ReplaceAll(data, []byte("zz-ZZ"), []byte("zh-CN"))
		data = bytes.ReplaceAll(data, []byte("first-production"), []byte("live"))
		writeSnapshotSource(t, root, path, data)
	}
	gate, gatePath, err := RecordLocaleSurfaceReviewA(root, "zh-CN", "historical-surface", "independent-reviewer", catalog)
	if err != nil {
		t.Fatal(err)
	}
	gate.SchemaVersion = localeSurfaceReviewASchemaVersionV3
	gate.Inputs.TourConfigProjection = nil
	gate.Inputs.TourLanguageContextSHA256 = ""
	gateData, _ := marshalGlossaryReviewJSON(gate)
	writeSnapshotSource(t, root, "data/locale-surface-reviews/zh-CN/"+gate.ReviewID+".a-gate.json", gateData)
	originalPackage, _, err := ExportLocaleSurfaceReviewPackage(root, "zh-CN", catalog)
	if err != nil {
		t.Fatal(err)
	}
	refs := []contentidentity.Reference{}
	paths := []string{"data/locale-surface-reviews/zh-CN/" + gate.ReviewID + ".a-gate.json", "locales/zh-CN/glossary.yaml", "locales/zh-CN/status.tsv", "data/quality-check-snapshots/zh-CN/qc-001/finalization.json"}
	for _, page := range catalog.Pages {
		paths = append(paths, canonicalCandidatePath("zh-CN", page.ID))
	}
	sort.Strings(paths)
	for _, path := range paths {
		data, err := readCompatibilityFile(root, path)
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, contentidentity.Reference{Path: path, SHA256: sum(data)})
	}
	profile := struct {
		Locale    string `json:"locale"`
		State     string `json:"production_state"`
		Hostname  string `json:"production_hostname"`
		PublicURL string `json:"production_public_url"`
	}{"zh-CN", "live", "zz.example", "https://zz.example/"}
	closure := contentidentity.Completion{Package: "tour-v1", State: "complete", PackageIdentity: sum([]byte("fixture-package")), SourceIdentity: sum([]byte("fixture-source")), EvidenceKind: "legacy-tour-closure/v1", Evidence: refs, EvidenceIdentity: sum(mustJSON(refs)), ContextIdentity: sum(mustJSON(struct {
		Profile                any
		ValidatedContextSHA256 string
	}{profile, sum(originalPackage)}))}
	state := contentidentity.Locale{Schema: contentidentity.LocaleSchema, Locale: "zh-CN", Packages: []contentidentity.Completion{closure}}
	writeSnapshotSource(t, root, "locales/zh-CN/content-scope.json", mustJSON(state))
	compatibilityChange(t, root, "# changed bytes\n"+compatibilityFixtureGlossary, "next")
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zh-CN", gate.ReviewID, catalog); err != nil {
		t.Fatalf("historical full closure proof rejected: %v", err)
	}
	writeSnapshotSource(t, root, "_content/js/playground.js", []byte("changed visible runtime message"))
	if _, _, err := AssessGlossaryCompatibility(root, "zh-CN", g1, catalog); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireCurrentLocaleSurfaceReviewAByReviewID(root, "zh-CN", gate.ReviewID, catalog); err == nil {
		t.Fatal("re-assessment approved later other-surface text with old Surface A")
	}
	after, _ := os.ReadFile(gatePath)
	if !bytes.Equal(after, gateData) {
		t.Fatal("historical gate changed")
	}
}
