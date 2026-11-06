package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type nativeGenerationFixture struct {
	f        currentValidationExportFixture
	batch    i18n.RetranslationExportResult
	output   string
	manifest i18n.GenerationBundleManifest
}

func invokeGenerationCLI(t *testing.T, root string, c *i18n.Catalog, args []string) ([]byte, error) {
	t.Helper()
	// Handler-level tests intentionally supply a Catalog so they can exercise
	// invalid caller scopes. They do not cover run's loaders/shared dispatcher;
	// generation_bundle_run_test.go covers that public entry point separately.
	copy := *c
	copy.Pages = append([]i18n.Page{}, c.Pages...)
	copy.Examples = append([]i18n.Example{}, c.Examples...)
	copy.TranslationScope = append([]string(nil), c.TranslationScope...)
	return authorizationDiagnosisOutput(t, func() error {
		done := contentidentity.BeginMemoization()
		defer done()
		return generationBundleLifecycleCommand(root, &copy, args)
	})
}

func makeNativeGenerationFixture(t *testing.T, native bool) nativeGenerationFixture {
	t.Helper()
	f := makeCurrentValidationExportFixture(t, native, "zh-CN")
	out, err := authorizationDiagnosisOutput(t, func() error {
		return retranslationExportLifecycleCommand(f.root, f.catalog, currentValidationExportArgs(f))
	})
	if err != nil {
		t.Fatal("synthetic revision export", err)
	}
	var batch i18n.RetranslationExportResult
	if err := json.Unmarshal(out, &batch); err != nil {
		t.Fatal(err)
	}
	fixture := nativeGenerationFixture{f: f, batch: batch, output: filepath.Join(t.TempDir(), "generation.zip")}
	before := authorizationDiagnosisInventory(t, f.root)
	out, err = invokeGenerationCLI(t, f.root, f.catalog, generationExportArgs(fixture, fixture.output))
	if err != nil || !strings.Contains(string(out), "Generation bundle exported:") {
		t.Fatal("Generation Bundle lifecycle dispatch", err, string(out))
	}
	files, err := i18n.ReadTransportBundle(courseContextRead(t, filepath.Dir(fixture.output), filepath.Base(fixture.output)), 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(files["manifest.json"], &fixture.manifest); err != nil {
		t.Fatal(err)
	}
	m := fixture.manifest
	if m.TaskKind != "translation-unit-revision" || m.Locale != "zh-CN" || m.BatchID != batch.BatchID || m.UnitKind != i18n.UnitKindPage || len(m.ExpectedOutputs) != 1 || m.ExpectedOutputs[0].UnitID != f.ids[0] {
		t.Fatal("revision task/exact batch Unit changed")
	}
	var formal i18n.RetranslationBatchManifest
	if err := json.Unmarshal(files["formal/manifest.json"], &formal); err != nil || len(formal.Units) != 1 {
		t.Fatal("formal batch missing", err)
	}
	u := formal.Units[0]
	if u.RevisionFeedbackSource != i18n.CurrentValidationRevisionFeedback || u.RevisionAuthorizationID != f.reopen || u.PreviousRating != "A" || u.PreviousSnapshotID != f.snapshot || u.PreviousReviewSHA256 != before[f.receipt] {
		t.Fatal("recorded current-validation authority lost")
	}
	name := filepath.Base(u.InputPath)
	for _, prefix := range []string{"formal/inputs/", "formal/candidates/", "formal/protected-candidates/"} {
		if len(files[prefix+name]) == 0 {
			t.Fatal("missing formal revision representation", prefix)
		}
	}
	if !strings.Contains(string(files["generation-context.json"]), "formal/comment-revision-transport") || !strings.Contains(string(files["generation-context.json"]), "ONLY the exact authorized teaching-comment bodies") {
		t.Fatal("bounded comment transport contract omitted from Generation context")
	}
	// This native fixture models a prose finding, not a teaching-comment one.
	// No invented comment authority or artificial historical/projection split:
	// protected-candidates must retain the actual passed historical raw bytes.
	var receipt i18n.QualityCheckSurfaceReopen
	if err := json.Unmarshal(courseContextRead(t, f.root, strings.TrimPrefix(f.receipt, f.root+string(filepath.Separator))), &receipt); err != nil {
		t.Fatal(err)
	}
	if len(receipt.CurrentValidation.Findings[0].CommentSlots) != 0 || len(files["formal/comment-revision-transport/"+name+".json"]) != 0 || len(files["formal/historical-protected-candidates/"+name]) != 0 {
		t.Fatal("prose finding acquired unauthorized comment projection")
	}
	old := receipt.Units[0].Snapshot
	var validation i18n.RetranslationValidation
	if err := json.Unmarshal(courseContextRead(t, f.root, old.ValidationPath), &validation); err != nil {
		t.Fatal(err)
	}
	raw := courseContextRead(t, f.root, filepath.ToSlash(filepath.Join("data/retranslation-runs", f.locale, old.SelectedBatchID, validation.RawResponsePath)))
	if !bytes.Equal(files["formal/protected-candidates/"+name], raw) || !bytes.Equal(files["formal/candidates/"+name], courseContextRead(t, f.root, old.CandidatePath)) {
		t.Fatal("historical candidate/raw relation changed")
	}
	if native && (len(f.catalog.TranslationScope) != 0 || len(f.catalog.TranslationPackages) != 0) {
		t.Fatal("native Bundle fabricated registration or changed caller scope")
	}
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.root)) {
		t.Fatal("Bundle export mutated fixture language/lifecycle evidence")
	}
	return fixture
}

func generationExportArgs(f nativeGenerationFixture, output string) []string {
	return []string{"generation-bundle", "export", "--content-package", f.f.pkg, "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--output", output}
}

func generationCheckArgs(f nativeGenerationFixture) []string {
	return []string{"generation-bundle", "current-check", "--content-package", f.f.pkg, "--bundle", f.output}
}

func TestTourNativeGenerationBundleExportDispatch(t *testing.T) {
	makeNativeGenerationFixture(t, true)
}

func TestTourNativeGenerationBundleCurrentCheckDispatch(t *testing.T) {
	f := makeNativeGenerationFixture(t, true)
	checkGenerationFixtureCurrent(t, f)
	testGenerationSelectionFailures(t, f)
}

func TestRegisteredGenerationBundleLifecycleDispatch(t *testing.T) {
	f := makeNativeGenerationFixture(t, false)
	checkGenerationFixtureCurrent(t, f)
	testGenerationSelectionFailures(t, f)
}

func checkGenerationFixtureCurrent(t *testing.T, f nativeGenerationFixture) {
	t.Helper()
	before := authorizationDiagnosisInventory(t, f.f.root)
	out, err := invokeGenerationCLI(t, f.f.root, f.f.catalog, generationCheckArgs(f))
	if err != nil || !strings.Contains(string(out), "Generation bundle: CURRENT") || !strings.Contains(string(out), "identity="+f.manifest.InputIdentitySHA256) {
		t.Fatal("independent complete Bundle reconstruction", err, string(out))
	}
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
		t.Fatal("current-check mutated fixture authority")
	}
}

func testGenerationSelectionFailures(t *testing.T, f nativeGenerationFixture) {
	t.Helper()
	for _, operation := range []string{"export", "current-check"} {
		for _, name := range []string{"partial", "different-order", "foreign-adapter", "unknown-package", "wrong-package", "missing-package", "duplicate-package", "batch-reopen", "batch-predecessor", "reopen-currentness"} {
			t.Run(operation+"/"+name, func(t *testing.T) {
				c := *f.f.catalog
				output := filepath.Join(t.TempDir(), "rejected.zip")
				args := generationExportArgs(f, output)
				if operation == "current-check" {
					args = generationCheckArgs(f)
				}
				want := ""
				switch name {
				case "partial", "different-order":
					ids := f.f.ids[:1]
					if name == "different-order" {
						ids = []string{f.f.ids[1], f.f.ids[0]}
					}
					if err := c.UseTranslationUnitScope(ids); err != nil {
						t.Fatal(err)
					}
					want = "scope"
				case "foreign-adapter":
					if f.f.pkg != "tour-v1" {
						t.Skip("registered scope legitimately has adapters")
					}
					u, err := c.Unit(f.f.ids[0])
					if err != nil {
						t.Fatal(err)
					}
					c.Adapted = append([]i18n.AdaptedTranslationUnit{}, c.Adapted...)
					c.Adapted = append(c.Adapted, i18n.AdaptedTranslationUnit{Unit: *u, Adapter: &authorizationDiagnosisAdapter{source: string(u.Source)}})
					want = "foreign adapter"
				case "unknown-package":
					args = replaceExportCLIFlag(t, args, "--content-package", "unknown-package")
					want = "unregistered translation package"
				case "missing-package":
					if f.f.pkg != "tour-v1" {
						t.Skip("unscoped registered catalog has no native batch membership")
					}
					args = append(append([]string{}, args[:2]...), args[4:]...)
					want = "exactly one explicit --content-package"
				case "duplicate-package":
					args = append(args, "--content-package", f.f.pkg)
					want = "invalid --content-package"
				case "wrong-package":
					raw := []byte("Foreign source.\n")
					d := i18n.TranslationPackageDescriptor{ID: "foreign-package", AuthoritySHA256: courseContextSHA(raw), Units: []i18n.AdaptedTranslationUnit{{Unit: i18n.TranslationUnit{ID: "foreign/unit", Kind: i18n.UnitKindPage, SourcePath: "fixtures/foreign.txt", Source: raw, SourceSHA256: courseContextSHA(raw)}, Adapter: &authorizationDiagnosisAdapter{source: string(raw)}, Route: "/foreign", StableIndex: 1}}}
					if err := c.RegisterTranslationPackage(d); err != nil {
						t.Fatal(err)
					}
					args = replaceExportCLIFlag(t, args, "--content-package", d.ID)
				case "batch-reopen", "batch-predecessor":
					path := f.batch.BatchPath + "/manifest.json"
					original := courseContextRead(t, f.f.root, path)
					var m i18n.RetranslationBatchManifest
					if err := json.Unmarshal(original, &m); err != nil {
						t.Fatal(err)
					}
					if name == "batch-reopen" {
						m.Units[0].RevisionAuthorizationID = "wrong-reopen"
					} else {
						m.Units[0].PreviousSnapshotID = "wrong-predecessor"
					}
					courseContextWrite(t, f.f.root, path, courseContextJSON(t, m))
					defer courseContextWrite(t, f.f.root, path, original)
				case "reopen-currentness":
					path := strings.TrimPrefix(f.f.receipt, f.f.root+string(filepath.Separator))
					original := courseContextRead(t, f.f.root, path)
					var r i18n.QualityCheckSurfaceReopen
					if err := json.Unmarshal(original, &r); err != nil {
						t.Fatal(err)
					}
					r.CurrentValidation.Findings[0].Reason += " changed"
					courseContextWrite(t, f.f.root, path, courseContextJSON(t, r))
					defer courseContextWrite(t, f.f.root, path, original)
				}
				_, err := invokeGenerationCLI(t, f.f.root, &c, args)
				if err == nil || (want != "" && !strings.Contains(err.Error(), want)) {
					t.Fatal("invalid scope/currentness passed or wrong failure class", err, want)
				}
				if name == "wrong-package" && strings.Contains(err.Error(), "unregistered translation package") {
					t.Fatal("fixture failed registration rather than exact authority matching", err)
				}
				if name == "wrong-package" && !strings.Contains(err.Error(), "outside selected complete package scope") {
					t.Fatal("wrong package did not reach the canonical batch membership guard", err)
				}
				if _, err := os.Stat(output); !os.IsNotExist(err) {
					t.Fatal("rejected export published a transport", err)
				}
			})
		}
	}
	checkGenerationFixtureCurrent(t, f)
}
