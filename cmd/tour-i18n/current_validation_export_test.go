package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
)

type currentValidationExportFixture struct {
	root, locale, pkg, snapshot, reopen, receipt string
	catalog                                      *i18n.Catalog
	ids                                          []string
}

// Only executable/contract inputs are copied. Locale, corpus, independent
// Review, candidates, A proof and recorded reopen are all synthetic TempDir data.
// No Generation Bundle or repository language evidence is used.
func makeCurrentValidationExportFixture(t *testing.T, native bool, locales ...string) currentValidationExportFixture {
	t.Helper()
	f := currentValidationExportFixture{root: authorizationDiagnosisMachineRoot(t), locale: "en-GB", pkg: "tour-v1", snapshot: "export-predecessor", reopen: "export-current-validator", catalog: &i18n.Catalog{}}
	if len(locales) > 0 {
		f.locale = locales[0]
	}
	metadata := installCurrentValidationExportAuthority(t, f)
	d := i18n.TranslationPackageDescriptor{ID: "test-registered-export", AuthoritySHA256: courseContextSHA([]byte("synthetic registered source authority"))}
	var article strings.Builder
	for i, text := range []string{"Read !! this message.", "Read the other message."} {
		id, source := fmt.Sprintf("fixture/%d", i+1), []byte(fmt.Sprintf("* Page %d\n\n%s\n\n", i+1, text))
		if native {
			f.catalog.Pages = append(f.catalog.Pages, i18n.Page{ID: id, Article: "fixture.article", SectionNumber: i + 1, Route: "/" + id, Source: source, SourceSHA256: courseContextSHA(source)})
			article.Write(source)
		} else {
			source = []byte(text + "\n")
			path := fmt.Sprintf("fixtures/export-%d.txt", i+1)
			courseContextWrite(t, f.root, path, source)
			d.Units = append(d.Units, i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: id, Kind: i18n.UnitKindPage, SourcePath: path, Source: source, SourceSHA256: courseContextSHA(source)}, Adapter: &authorizationDiagnosisAdapter{source: string(source)}, StableIndex: i + 1, Route: "/" + id})
		}
		f.ids = append(f.ids, id)
	}
	if native {
		courseContextWrite(t, f.root, "_content/tour/fixture.article", []byte(article.String()))
	} else {
		f.pkg = d.ID
		if err := f.catalog.RegisterTranslationPackage(d); err != nil {
			t.Fatal(err)
		}
		if err := f.catalog.UseTranslationPackage(d.ID); err != nil {
			t.Fatal(err)
		}
	}
	return completeCurrentValidationExportFixture(t, f, metadata)
}

func installCurrentValidationExportAuthority(t *testing.T, f currentValidationExportFixture) i18n.Locale {
	t.Helper()
	policy := languageaudit.Authority{Schema: languageaudit.PolicySchema, Version: 1, Policy: languageaudit.Policy{Locale: f.locale, Profile: "explicit-export-fixture", ProseScripts: []string{"Latin"}, SentenceTerminators: ".!?", ResidualMinWords: 1}}
	metadata := i18n.Locale{Locale: f.locale, HTMLLang: f.locale, LanguageName: "English fixture", EnglishName: "English fixture", Phase: "scaffold", TranslationUnit: "present.Section", LanguageAudit: &policy}
	courseContextWrite(t, f.root, "locales/"+f.locale+"/locale.json", courseContextJSON(t, metadata))
	courseContextWrite(t, f.root, "locales/"+f.locale+"/glossary.yaml", []byte("locale: "+f.locale+"\nmandatory:\n  Go: Go\n"))
	corpus := i18n.GlossarySourceCorpus{Schema: i18n.GlossaryCorpusSchema, Sources: []i18n.GlossaryArchiveReference{}, Contributors: []i18n.LanguageContributor{{ID: "synthetic-export", Parser: "present-visible/v1", Package: "tour-v1", Path: "fixtures/source.txt", Contexts: []i18n.LanguageContext{{ID: "synthetic", Text: "Go Read the message", Protected: []string{}}}}}}
	corpus.Contributors[0].SourceSHA = i18n.ContributorIdentity(corpus.Contributors[0])
	corpus.Identity = i18n.CorpusIdentity(corpus)
	courseContextWrite(t, f.root, i18n.GlossaryCorpusPath, courseContextJSON(t, corpus))
	if _, _, err := i18n.RecordGlossaryReview(f.root, f.locale, "synthetic-export", "independent-fixture-reviewer", "passed", nil); err != nil {
		t.Fatal(err)
	}
	review, _, err := i18n.ExportUnifiedGlossaryBundle(f.root, f.locale, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(f.root, review, "synthetic-export", "independent-fixture-reviewer", "separate-fixture-generator", "passed", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RequireUnifiedGlossaryReview(f.root, f.locale); err != nil {
		t.Fatal("fixture lacks current independent Glossary Review", err)
	}
	return metadata
}

// Shared current-validation fixture lifecycle; callers supply either a small
// in-memory catalog or a complete on-disk source/catalog for the real run path.
func completeCurrentValidationExportFixture(t *testing.T, f currentValidationExportFixture, metadata i18n.Locale) currentValidationExportFixture {
	t.Helper()
	e, err := i18n.ExportRetranslationBatch(f.root, f.catalog, i18n.RetranslationExportOptions{Locale: f.locale, Generator: i18n.RetranslationGeneratorCodex, UnitIDs: f.ids})
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.RetranslationBatchManifest
	if err := json.Unmarshal(courseContextRead(t, f.root, e.BatchPath+"/manifest.json"), &m); err != nil {
		t.Fatal(err)
	}
	for _, u := range m.Units {
		courseContextWrite(t, f.root, e.BatchPath+"/raw-responses/"+filepath.Base(u.InputPath), courseContextRead(t, f.root, e.BatchPath+"/"+u.InputPath))
	}
	if p, err := i18n.ProcessRetranslationBatch(f.root, f.catalog, i18n.RetranslationProcessOptions{Locale: f.locale, BatchID: e.BatchID}); err != nil || p.ValidationPassed != len(f.ids) {
		t.Fatal("synthetic baseline validation", p, err)
	}
	if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(f.root, f.catalog, i18n.QualityCheckSnapshotOptions{Locale: f.locale, SnapshotID: f.snapshot}); err != nil {
		t.Fatal(err)
	}
	if _, err := i18n.RecordQualityCheckResults(f.root, f.catalog, i18n.QualityCheckRecordOptions{Locale: f.locale, SnapshotID: f.snapshot, UnitIDs: f.ids, Rating: "A", ReviewerSession: "independent-fixture-reviewer"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.FinalizeQualityCheck(f.root, f.catalog, i18n.QualityCheckFinalizeOptions{Locale: f.locale, SnapshotID: f.snapshot}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.EstablishCurrentTranslationAuthorization(f.root, f.catalog, f.locale, f.snapshot); err != nil {
		t.Fatal(err)
	}
	metadata.LanguageAudit.Version++
	metadata.LanguageAudit.Policy.ForbiddenClausePunctuation = []string{"!!"}
	courseContextWrite(t, f.root, "locales/"+f.locale+"/locale.json", courseContextJSON(t, metadata))
	a, err := i18n.AssessCurrentTranslation(f.root, f.catalog, f.locale, f.snapshot)
	if err != nil {
		t.Fatal("synthetic current assessment", err)
	}
	if a.Compatible != len(f.ids)-1 || a.Affected != 1 || a.Stale != 0 || a.Unknown != 0 || a.Package != f.pkg {
		t.Fatalf("synthetic assessment package=%s compatible=%d affected=%d stale=%d unknown=%d", a.Package, a.Compatible, a.Affected, a.Stale, a.Unknown)
	}
	raw := courseContextJSON(t, a)
	evidencePath := "engineering/current-assessment.json"
	courseContextWrite(t, f.root, evidencePath, raw)
	snapshotSHA := courseContextSHA(courseContextRead(t, f.root, "data/quality-check-snapshots/"+f.locale+"/"+f.snapshot+"/manifest.json"))
	o := i18n.CurrentValidationReopenOptions{ContentPackage: f.pkg, Locale: f.locale, ReopenID: f.reopen, PreviousSnapshotID: f.snapshot, SnapshotSHA256: snapshotSHA, Evidence: i18n.GlossaryArchiveReference{Path: evidencePath, SHA256: courseContextSHA(raw)}, AssessmentIdentity: a.Identity}
	r, path, err := i18n.RecordQualityCheckSurfaceReopen(f.root, f.catalog, i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: i18n.CurrentValidationAuthority, Locale: f.locale, ReopenID: f.reopen, PreviousSnapshotID: f.snapshot, CurrentValidation: &o})
	if err != nil || r.UnitCount != 1 || r.Units[0].UnitID != f.ids[0] {
		t.Fatal("synthetic recorded current-validation receipt", err)
	}
	f.receipt = path
	if !filepath.IsAbs(f.receipt) {
		f.receipt = filepath.Join(f.root, f.receipt)
	}
	f.catalog.TranslationScope = nil // fixture's natural global catalog, before CLI selection
	return f
}

func currentValidationExportArgs(f currentValidationExportFixture) []string {
	return []string{"retranslation", "export", "--content-package", f.pkg, "--locale", f.locale, "--generator", "chatgpt", "--previous-snapshot-id", f.snapshot, "--surface-reopen-id", f.reopen, "--allow-reexport", "--id", f.ids[0], "--unit-kind", "page", "--batch-id", "chatgpt-" + f.locale + "-002", "--json"}
}

func replaceExportCLIFlag(t *testing.T, args []string, name, value string) []string {
	t.Helper()
	args = append([]string{}, args...)
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			args[i+1] = value
			return args
		}
	}
	t.Fatal("fixture flag missing", name)
	return nil
}

// This invokes the full export dispatch used by run after catalog hydration,
// with raw CLI argv. Parsing, complete package selection, current receipt
// reconstruction, all export gates and manifest publication are production code.
func TestTourNativeCurrentValidationExportFullCLI(t *testing.T) {
	f := makeCurrentValidationExportFixture(t, true)
	if len(f.catalog.TranslationPackages) != 0 || len(f.catalog.TranslationScope) != 0 {
		t.Fatal("native fixture must not fake package registration or explicit scope")
	}
	args := currentValidationExportArgs(f)
	// Omitting the formal selector cannot turn the recorded authority into an
	// implicit native export, even though legacy initial exports allow defaults.
	withoutPackage := append(append([]string{}, args[:2]...), args[4:]...)
	before := authorizationDiagnosisInventory(t, f.root)
	if err := retranslationExportLifecycleCommand(f.root, f.catalog, withoutPackage); err == nil || !strings.Contains(err.Error(), "exactly one explicit --content-package") {
		t.Fatal("implicit current-validation package accepted", err)
	}
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.root)) {
		t.Fatal("missing package selection mutated fixture")
	}
	testCurrentValidationExportDispatch(t, f)
	if len(f.catalog.TranslationPackages) != 0 || len(f.catalog.TranslationScope) != 0 {
		t.Fatal("CLI manufactured native registration or mutated scope")
	}
}

func TestRegisteredCurrentValidationExportFullCLI(t *testing.T) {
	testCurrentValidationExportDispatch(t, makeCurrentValidationExportFixture(t, false))
}

func testCurrentValidationExportDispatch(t *testing.T, f currentValidationExportFixture) {
	t.Helper()
	args := currentValidationExportArgs(f)
	before := authorizationDiagnosisInventory(t, f.root)
	for _, name := range []string{"wrong-package", "wrong-reopen", "wrong-predecessor", "wrong-unit", "superset", "duplicate-package", "partial-scope", "different-scope"} {
		t.Run(name, func(t *testing.T) {
			bad, c, want := append([]string{}, args...), *f.catalog, ""
			switch name {
			case "wrong-package":
				bad, want = replaceExportCLIFlag(t, bad, "--content-package", "unknown-package"), "unregistered translation package"
			case "wrong-reopen":
				bad, want = replaceExportCLIFlag(t, bad, "--surface-reopen-id", "missing-reopen"), "missing-reopen"
			case "wrong-predecessor":
				bad, want = replaceExportCLIFlag(t, bad, "--previous-snapshot-id", "wrong-predecessor"), "predecessor does not match"
			case "wrong-unit":
				bad, want = replaceExportCLIFlag(t, bad, "--id", f.ids[1]), "unit set must exactly match"
			case "superset":
				bad, want = append(bad, "--id", f.ids[1]), "unit set must exactly match"
			case "duplicate-package":
				bad, want = append(bad, "--content-package", f.pkg), "invalid --content-package"
			case "partial-scope", "different-scope":
				ids := f.ids[:1]
				if name == "different-scope" {
					ids = f.ids[1:]
				}
				if err := c.UseTranslationUnitScope(ids); err != nil {
					t.Fatal(err)
				}
				want = "scope"
			}
			if err := retranslationExportLifecycleCommand(f.root, &c, bad); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatal("wrong CLI failure class", err, "want", want)
			}
			if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.root)) {
				t.Fatal("rejected export mutated fixture lifecycle artifacts")
			}
		})
	}
	out, err := authorizationDiagnosisOutput(t, func() error { return retranslationExportLifecycleCommand(f.root, f.catalog, args) })
	if err != nil {
		t.Fatal("full current-validation export dispatch", err)
	}
	var result i18n.RetranslationExportResult
	if err := json.Unmarshal(out, &result); err != nil || result.UnitCount != 1 || result.UnitKind != i18n.UnitKindPage || result.BatchID != "chatgpt-en-GB-002" || !reflect.DeepEqual(result.UnitIDs, f.ids[:1]) {
		t.Fatal("CLI stdout result contract", string(out), err)
	}
	var manifest i18n.RetranslationBatchManifest
	if err := json.Unmarshal(courseContextRead(t, f.root, result.BatchPath+"/manifest.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.UnitCount != 1 || len(manifest.Units) != 1 {
		t.Fatal("export scope expanded")
	}
	u := manifest.Units[0]
	receiptRelative, err := filepath.Rel(f.root, f.receipt)
	if err != nil {
		t.Fatal(err)
	}
	if u.UnitID != f.ids[0] || u.RevisionFeedbackSource != i18n.CurrentValidationRevisionFeedback || u.PreviousRating != "A" || u.PreviousSnapshotID != f.snapshot || u.RevisionAuthorizationID != f.reopen || u.PreviousReviewSHA256 != before[f.receipt] || u.PreviousReviewPath != filepath.ToSlash(receiptRelative) || !strings.Contains(u.PreviousFinding, "clause-punctuation") {
		t.Fatal("current receipt/machine finding lineage lost", u)
	}
	for path, sha := range before {
		if courseContextSHA(courseContextRead(t, f.root, strings.TrimPrefix(path, f.root+string(filepath.Separator)))) != sha {
			t.Fatal("export rewrote historical fixture evidence", path)
		}
	}
	if _, err := os.Stat(filepath.Join(f.root, result.BatchPath, "raw-responses")); !os.IsNotExist(err) {
		t.Fatal("export created Generation output", err)
	}
}
