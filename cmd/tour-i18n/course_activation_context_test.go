package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func courseContextSHA(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func courseContextJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func courseContextWrite(t *testing.T, root, path string, b []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, b, 0644); err != nil {
		t.Fatal(err)
	}
}

func courseContextRead(t *testing.T, root, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Build isolated evidence with the production export/process/QC/finalization
// and historical-promotion builders. No real locale asset is written.
func courseActivationContextFixture(t *testing.T) (string, *i18n.Catalog, i18n.GlossaryArchiveReference, string, []byte) {
	t.Helper()
	root := t.TempDir()
	repository, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"AGENTS.md", "docs/WORKFLOW_HANDOFF.md", "docs/GLOSSARY_REVIEW.md", "docs/NEW_LOCALE_RUNBOOK.md",
		"docs/TERMINOLOGY_GUIDE.md", "docs/TRANSLATION_TASK_SPEC.md", "docs/TRANSLATION_TERMINOLOGY.md",
		"docs/SITE_V2_WORKFLOW.md", "docs/SITE_V2_ARCHITECTURE.md", "docs/GLOSSARY_COMPATIBILITY.md",
		"docs/TRANSLATION_WORKFLOW.md", "docs/CODEX_TRANSLATION.md", "docs/CHATGPT_LANGUAGE_GENERATION.md",
		"locales/zh-CN/locale.json", sitecontent.GlobalPath,
	} {
		courseContextWrite(t, root, path, courseContextRead(t, repository, path))
	}
	source := []byte("* Page\n\nUse `Go` on this page.\n")
	catalog := &i18n.Catalog{Pages: []i18n.Page{{ID: "lesson/1", Article: "lesson.article", SectionNumber: 1, Route: "/lesson/1", Source: source, SourceSHA256: courseContextSHA(source)}}}
	courseContextWrite(t, root, "_content/tour/lesson.article", append([]byte("Lesson\nLanguage introduction\n\n"), source...))
	glossary := []byte("locale: zh-CN\nmandatory:\n  Go: Go\nkeep:\n  - Go\n  - gofmt\n")
	courseContextWrite(t, root, "locales/zh-CN/glossary.yaml", glossary)
	corpus := i18n.GlossarySourceCorpus{Schema: i18n.GlossaryCorpusSchema, Sources: []i18n.GlossaryArchiveReference{}, Contributors: []i18n.LanguageContributor{{ID: "tour:page:lesson/1", Parser: "present-visible/v1", Package: "tour-v1", Path: "_content/tour/lesson.article", Contexts: []i18n.LanguageContext{{ID: "synthetic", Text: "Go programming language", Protected: []string{}}}}}}
	corpus.Contributors[0].SourceSHA = i18n.ContributorIdentity(corpus.Contributors[0])
	corpus.Identity = i18n.CorpusIdentity(corpus)
	courseContextWrite(t, root, i18n.GlossaryCorpusPath, courseContextJSON(t, corpus))
	if _, _, err := i18n.RecordGlossaryReview(root, "zh-CN", "synthetic", "synthetic-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	bundle, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, bundle, "synthetic-unified", "synthetic-reviewer", "synthetic-generator", "passed", nil); err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"en", "zh-CN"} {
		courseContextWrite(t, root, "internal/tour/ui/"+locale+".json", []byte(`{"locale":"`+locale+`","html_lang":"`+locale+`","messages":{"test.one":{"kind":"plain","text":"Welcome"}}}`))
	}
	courseContextWrite(t, root, "locales/zh-CN/article-metadata.json", []byte(`{"locale":"zh-CN","articles":[{"article":"lesson.article","title":"Lesson","subtitle":"Language introduction"}]}`))
	for _, path := range []string{"internal/tour/languages.go", "internal/tour/tour.go", "internal/tour/production.go", "internal/tour/project.go", "internal/tour/seo.go", "_content/js/playground.js", "_content/tour/static/js/app.js", "_content/tour/template/index.tmpl", "_content/tour/static/partials/index.html"} {
		courseContextWrite(t, root, path, []byte("package tour\nconst Config = \"Tour\"\n"))
	}
	makeSnapshot := func(batch, snapshot, previous, title string) *i18n.QualityCheckSnapshotManifest {
		t.Helper()
		result, err := i18n.ExportRetranslationBatch(root, catalog, i18n.RetranslationExportOptions{Locale: "zh-CN", BatchID: batch, UnitIDs: []string{"lesson/1"}, AllowReexport: true})
		if err != nil {
			t.Fatal(err)
		}
		var manifest i18n.RetranslationBatchManifest
		if err := json.Unmarshal(courseContextRead(t, root, result.BatchPath+"/manifest.json"), &manifest); err != nil {
			t.Fatal(err)
		}
		input := courseContextRead(t, root, result.BatchPath+"/"+manifest.Units[0].InputPath)
		courseContextWrite(t, root, result.BatchPath+"/raw-responses/lesson-1.article", bytes.Replace(input, []byte("* Page"), []byte(title), 1))
		if _, err := i18n.ProcessRetranslationBatch(root, catalog, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch}); err != nil {
			t.Fatal(err)
		}
		s, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, catalog, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: snapshot})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := i18n.RecordQualityCheckResults(root, catalog, i18n.QualityCheckRecordOptions{Locale: "zh-CN", SnapshotID: snapshot, PreviousSnapshotID: previous, UnitIDs: []string{"lesson/1"}, Rating: "A"}); err != nil {
			t.Fatal(err)
		}
		if _, _, err := i18n.FinalizeQualityCheck(root, catalog, i18n.QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: snapshot}); err != nil {
			t.Fatal(err)
		}
		candidate := courseContextRead(t, root, s.Units[0].CandidatePath)
		courseContextWrite(t, root, "locales/zh-CN/candidates/lesson-1.article", candidate)
		courseContextWrite(t, root, "locales/zh-CN/status.tsv", []byte("unit_id\tstatus\tattempts\tsource_sha256\tcandidate_path\tupdated_at\tnote\nlesson/1\tready\t1\t"+catalog.Pages[0].SourceSHA256+"\tlocales/zh-CN/candidates/lesson-1.article\t\t\n"))
		return s
	}
	s := makeSnapshot("codex-zh-CN-001", "qc-001", "", "* Initial fixture")
	old, err := i18n.AssembleCourseMetadata(root, catalog, i18n.CourseMetadataAssemblyOptions{SchemaVersion: 1, Locale: "zh-CN", Provider: "fixture", Model: "fixture", GeneratedAt: "2026-10-09T00:00:00Z", Descriptions: []byte(`{"pages":[{"page_id":"lesson/1","description":"A complete formal description grounded in the initial lesson content."}]}`)})
	if err != nil {
		t.Fatal(err)
	}
	courseContextWrite(t, root, "locales/zh-CN/course-metadata.json", old)
	paths := []string{"locales/zh-CN/course-metadata.json", "locales/zh-CN/glossary.yaml", "locales/zh-CN/status.tsv", "internal/tour/ui/zh-CN.json", "data/quality-check-snapshots/zh-CN/qc-001/manifest.json", "data/quality-check-snapshots/zh-CN/qc-001/quality-check-results.json", "data/quality-check-snapshots/zh-CN/qc-001/finalization.json", "locales/zh-CN/candidates/lesson-1.article", s.Units[0].CandidatePath, s.Units[0].ValidationPath}
	sort.Strings(paths)
	refs := []sitecontent.Reference{}
	for _, path := range paths {
		refs = append(refs, sitecontent.Reference{Path: path, SHA256: courseContextSHA(courseContextRead(t, root, path))})
	}
	var global sitecontent.Global
	if err := sitecontent.StrictJSON(courseContextRead(t, root, sitecontent.GlobalPath), &global); err != nil {
		t.Fatal(err)
	}
	packages := map[string]sitecontent.Package{}
	for _, p := range global.Packages {
		packages[p.ID] = p
	}
	tour := packages["tour-v1"]
	baseline := sitecontent.Locale{Schema: sitecontent.LocaleSchema, Locale: "zh-CN", Packages: []sitecontent.Completion{{Package: tour.ID, State: "complete", PackageIdentity: tour.Identity, SourceIdentity: tour.SourceIdentity, Surfaces: tour.Surfaces, Routes: tour.Routes, RouteFamilies: []string{"/tour/**"}, EvidenceKind: "legacy-tour-closure/v1", Evidence: refs, EvidenceIdentity: courseContextSHA(courseContextJSON(t, refs)), ContextIdentity: courseContextSHA([]byte("synthetic legacy context")), LegacySurfaceState: "historical-verified"}}}
	baselineBytes, err := sitecontent.Encode(baseline)
	if err != nil {
		t.Fatal(err)
	}
	path := sitecontent.LocalePath("zh-CN")
	courseContextWrite(t, root, path, baselineBytes)
	makeSnapshot("codex-zh-CN-002", "qc-002", "qc-001", "* Revised fixture")
	if _, _, err := i18n.RecoverHistoricalPromotion(root, catalog, "zh-CN", "qc-002", true); err != nil {
		t.Fatal(err)
	}
	// Assemble the exact immutable receipt/closure bindings consumed by the
	// non-recursive activation proof; package language generation is outside
	// this isolated historical-context regression.
	p := packages["site-v2-shell"]
	closure := sitecontent.PackageClosure{Schema: sitecontent.ClosureSchema, Locale: "zh-CN", Package: p.ID, PackageSHA: p.Identity}
	closure.Identity = courseContextSHA(courseContextJSON(t, closure))
	closurePath := "data/site-workflow/zh-CN/site-v2-shell/closure/" + closure.Identity + ".json"
	courseContextWrite(t, root, closurePath, courseContextJSON(t, closure))
	closureRef := sitecontent.Reference{Path: closurePath, SHA256: courseContextSHA(courseContextRead(t, root, closurePath))}
	markdown := "data/locale-language/zh-CN/surface/synthetic-activation.md"
	courseContextWrite(t, root, markdown, []byte("# Synthetic independent activation evidence\n"))
	unifiedRef, err := i18n.RequireUnifiedGlossaryReview(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	scope := sitecontent.IntegratedSurfaceScope{Schema: sitecontent.IntegratedSurfaceSchema, Locale: "zh-CN", GlossarySHA: courseContextSHA(glossary), UnifiedReview: sitecontent.Reference{Path: unifiedRef.Path, SHA256: unifiedRef.SHA256}, ExistingCoverage: baseline, Closures: []sitecontent.Reference{closureRef}, Runtime: []sitecontent.Reference{{Path: path, SHA256: courseContextSHA(baselineBytes)}, {Path: sitecontent.GlobalPath, SHA256: courseContextSHA(courseContextRead(t, root, sitecontent.GlobalPath))}}}
	scope.Identity = courseContextSHA(courseContextJSON(t, scope))
	receipt := sitecontent.IntegratedSurfaceReceipt{Schema: sitecontent.IntegratedSurfaceSchema, Scope: scope, ReviewID: "synthetic-activation", Reviewer: "synthetic-reviewer", Decision: "passed", Evidence: sitecontent.Reference{Path: markdown, SHA256: courseContextSHA(courseContextRead(t, root, markdown))}}
	receipt.Identity = courseContextSHA(courseContextJSON(t, receipt))
	receiptPath := "data/locale-language/zh-CN/surface/synthetic-activation.json"
	courseContextWrite(t, root, receiptPath, courseContextJSON(t, receipt))
	activationRefs := []sitecontent.Reference{closureRef, {Path: receiptPath, SHA256: courseContextSHA(courseContextRead(t, root, receiptPath))}}
	current := baseline
	current.Packages = append(append([]sitecontent.Completion{}, baseline.Packages...), sitecontent.Completion{Package: p.ID, State: "complete", PackageIdentity: p.Identity, SourceIdentity: p.SourceIdentity, Surfaces: p.Surfaces, Routes: p.Routes, RouteFamilies: []string{"/", "/translation/"}, EvidenceKind: sitecontent.PackageCompletionKind, Evidence: activationRefs, EvidenceIdentity: courseContextSHA(courseContextJSON(t, activationRefs)), ContextIdentity: closure.Identity})
	currentBytes, err := sitecontent.Encode(current)
	if err != nil {
		t.Fatal(err)
	}
	courseContextWrite(t, root, path, currentBytes)
	descriptions := "replacement.json"
	courseContextWrite(t, root, descriptions, []byte(`{"pages":[{"page_id":"lesson/1","description":"A distinct formal replacement grounded in the revised lesson content."}]}`))
	return root, catalog, i18n.GlossaryArchiveReference{Path: path, SHA256: courseContextSHA(baselineBytes)}, receiptPath, old
}

func TestCourseMetadataHistoricalActivationCommandWiring(t *testing.T) {
	root, catalog, historical, receiptPath, old := courseActivationContextFixture(t)
	protected := map[string][]byte{}
	for _, path := range []string{historical.Path, receiptPath, "data/quality-check-snapshots/zh-CN/qc-001/finalization.json", "data/quality-check-snapshots/zh-CN/qc-002/finalization.json", "locales/zh-CN/candidates/lesson-1.article"} {
		protected[path] = courseContextRead(t, root, path)
	}
	options := i18n.CourseMetadataRefreshOptions{SchemaVersion: 1, Locale: "zh-CN", Provider: "fixture", Model: "fixture", GeneratedAt: "2026-10-09T02:29:24Z", Descriptions: courseContextRead(t, root, "replacement.json")}
	if _, _, err := i18n.ApplyCourseMetadataRefresh(root, catalog, options); err == nil || !strings.Contains(err.Error(), "historical context reference changed: "+historical.Path) {
		t.Fatalf("unwired current scope passed historical SHA: %v", err)
	}
	assertCourseMetadataOutput(t, filepath.Join(root, "locales/zh-CN/course-metadata.json"), old)
	if _, err := os.Stat(filepath.Join(root, "data/course-seo/history")); !os.IsNotExist(err) {
		t.Fatal("unwired failure archived history before proof")
	}
	if err := refreshCourseMetadata(root, catalog, []string{"--locale", "zh-CN", "--schema-version", "1", "--descriptions", filepath.Join(root, "replacement.json"), "--provider", "fixture", "--model", "fixture", "--generated-at", options.GeneratedAt, "--output", filepath.Join(root, "locales/zh-CN/course-metadata.json")}); err != nil {
		t.Fatal("native command could not use formal historical activation proof", err)
	}
	restored, err := catalog.HistoricalContextReader(root, "zh-CN", historical)
	if err != nil || courseContextSHA(restored) != historical.SHA256 || bytes.Equal(restored, courseContextRead(t, root, historical.Path)) {
		t.Fatalf("not the exact historical baseline: %v", err)
	}
	// A fresh catalog tests the separate recover-history command registration.
	recoveryCatalog := &i18n.Catalog{Pages: catalog.Pages}
	courseContextWrite(t, root, "old.json", old)
	output := captureStdout(t, func() error {
		return recoverCourseMetadataHistory(root, recoveryCatalog, []string{"--locale", "zh-CN", "--input", filepath.Join(root, "old.json")})
	})
	var recovery struct {
		Applied bool                     `json:"applied"`
		Path    string                   `json:"evidence_path"`
		Proof   i18n.CourseSEOTransition `json:"evidence"`
	}
	if err := json.Unmarshal(output, &recovery); err != nil || recovery.Applied || recovery.Proof.Schema != i18n.CourseHistoryRecoverySchema || recovery.Proof.TargetSuccessor == nil {
		t.Fatalf("wrong recovery preflight authority: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, recovery.Path)); !os.IsNotExist(err) {
		t.Fatal("recovery preflight applied evidence")
	}
	for path, before := range protected {
		assertCourseMetadataOutput(t, filepath.Join(root, path), before)
	}
	if len(catalog.TranslationPackages) != 0 || len(recoveryCatalog.TranslationPackages) != 0 {
		t.Fatal("Course SEO command expanded translation package inventory")
	}
}

func TestCourseMetadataHistoricalActivationFailClosed(t *testing.T) {
	for _, mutation := range []string{"wrong-historical-sha", "missing-receipt", "tampered-receipt", "malformed-receipt", "stale-runtime"} {
		t.Run(mutation, func(t *testing.T) {
			root, catalog, historical, receiptPath, old := courseActivationContextFixture(t)
			switch mutation {
			case "wrong-historical-sha":
				entries, err := filepath.Glob(filepath.Join(root, "data/promotion-recoveries/zh-CN/*.json"))
				if err != nil || len(entries) != 1 {
					t.Fatal("missing fixture target successor", err)
				}
				var e i18n.TargetSuccessorEvidence
				if err := json.Unmarshal(courseContextRead(t, root, strings.TrimPrefix(entries[0], root+"/")), &e); err != nil {
					t.Fatal(err)
				}
				e.HistoricalClosure.SHA256 = strings.Repeat("0", 64)
				e.Identity = ""
				e.Identity = courseContextSHA(courseContextJSON(t, e))
				if err := os.Remove(entries[0]); err != nil {
					t.Fatal(err)
				}
				encoded, err := sitecontent.Encode(e)
				if err != nil {
					t.Fatal(err)
				}
				courseContextWrite(t, root, "data/promotion-recoveries/zh-CN/"+e.Identity+".json", encoded)
			case "missing-receipt":
				if err := os.Remove(filepath.Join(root, receiptPath)); err != nil {
					t.Fatal(err)
				}
			case "tampered-receipt":
				courseContextWrite(t, root, receiptPath, append(courseContextRead(t, root, receiptPath), '\n'))
			case "malformed-receipt":
				courseContextWrite(t, root, receiptPath, []byte("{}"))
			case "stale-runtime":
				courseContextWrite(t, root, sitecontent.GlobalPath, append(courseContextRead(t, root, sitecontent.GlobalPath), '\n'))
			}
			err := refreshCourseMetadata(root, catalog, []string{"--locale", "zh-CN", "--schema-version", "1", "--descriptions", filepath.Join(root, "replacement.json"), "--provider", "fixture", "--model", "fixture", "--generated-at", "2026-10-09T02:29:24Z", "--output", filepath.Join(root, "locales/zh-CN/course-metadata.json")})
			if err == nil || !strings.Contains(err.Error(), "historical context reference changed: "+historical.Path) {
				t.Fatalf("invalid historical activation authority accepted: %v", err)
			}
			assertCourseMetadataOutput(t, filepath.Join(root, "locales/zh-CN/course-metadata.json"), old)
			if _, err := os.Stat(filepath.Join(root, "data/course-seo/history")); !os.IsNotExist(err) {
				t.Fatal("failed proof wrote Course SEO history")
			}
			// Exercise recover-history with the same invalid activation proof,
			// using only a read-only refresh projection as its current temp asset.
			current, _, err := i18n.RefreshCourseMetadata(root, catalog, i18n.CourseMetadataRefreshOptions{SchemaVersion: 1, Locale: "zh-CN", Provider: "fixture", Model: "fixture", GeneratedAt: "2026-10-09T02:29:24Z", Descriptions: courseContextRead(t, root, "replacement.json")})
			if err != nil {
				t.Fatal(err)
			}
			courseContextWrite(t, root, "locales/zh-CN/course-metadata.json", current)
			courseContextWrite(t, root, "old.json", old)
			err = recoverCourseMetadataHistory(root, &i18n.Catalog{Pages: catalog.Pages}, []string{"--locale", "zh-CN", "--input", filepath.Join(root, "old.json")})
			if err == nil || !strings.Contains(err.Error(), "historical context reference changed: "+historical.Path) {
				t.Fatalf("recovery accepted invalid historical activation proof: %v", err)
			}
			assertCourseMetadataOutput(t, filepath.Join(root, "locales/zh-CN/course-metadata.json"), current)
		})
	}
}
