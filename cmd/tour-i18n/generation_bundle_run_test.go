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
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// A source-only foundation for run's real catalog loading and package loaders.
// No repository locale, batch, Snapshot, reopen or completion receipt is copied.
// The frozen English Learn/Docs snapshot supplies the existing registered
// package; the Tour is an independent, small but complete native source tree.
func generationRunSourceFixture(t *testing.T) (string, *i18n.Catalog) {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	root := authorizationDiagnosisMachineRoot(t)
	for _, p := range []string{"_content/tour/welcome.article", "_content/tour/welcome/hello.go", "_content/tour/welcome/sandbox.go", "data/site-shell/home.html", "data/site-shell/translation.html"} {
		courseContextWrite(t, root, p, courseContextRead(t, repo, p))
	}
	// Preserve the canonical conditional welcome publication branches, while
	// making exactly its first Page affected by a later synthetic policy change.
	welcome := courseContextRead(t, root, "_content/tour/welcome.article")
	welcome = []byte(strings.Replace(string(welcome), "Welcome to a tour", "Read !! this message.\n\nWelcome to a tour", 1))
	courseContextWrite(t, root, "_content/tour/welcome.article", welcome)
	for _, article := range i18n.ArticleOrder {
		if article != "welcome.article" {
			courseContextWrite(t, root, "_content/tour/"+article, []byte(fmt.Sprintf("Fixture %s\nSynthetic source description.\n\nThe Fixture Authors\n\n* Fixture page\n\nRead the other message.\n\n", article)))
		}
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal("real source catalog", err)
	}
	pages, conditional, err := i18n.CatalogBytes(c)
	if err != nil {
		t.Fatal(err)
	}
	examples, err := i18n.ExampleCatalogBytes(c)
	if err != nil {
		t.Fatal(err)
	}
	for p, b := range map[string][]byte{"data/tour-pages.tsv": pages, "data/tour-conditional-pages.tsv": conditional, "data/tour-examples.tsv": examples} {
		courseContextWrite(t, root, p, b)
	}
	snapshot, err := sitecontent.ReadSourceSnapshot(repo)
	if err != nil {
		t.Fatal("frozen English source snapshot", err)
	}
	courseContextWrite(t, root, sitecontent.SnapshotPath, snapshot)
	g, err := sitecontent.BuildGlobal(root, snapshot)
	if err != nil {
		t.Fatal("isolated source scope", err)
	}
	courseContextWrite(t, root, sitecontent.GlobalPath, courseContextJSON(t, g))
	registry, err := sitecontent.FreezePageRegistry(g)
	if err != nil {
		t.Fatal(err)
	}
	courseContextWrite(t, root, sitecontent.PageRegistryPath, courseContextJSON(t, registry))
	// Use the production loader here too, so setup and run independently see
	// the same exact descriptor. In particular, it must never register Tour.
	if err := attachCommandTranslationPackages(root, c, []string{"generation-bundle", "export"}); err != nil {
		t.Fatal("isolated registered package loading", err)
	}
	for _, d := range c.TranslationPackages {
		if d.ID == "tour-v1" {
			t.Fatal("native Tour acquired fake registration")
		}
	}
	if err := c.UseTranslationPackage("tour-v1"); err == nil || !strings.Contains(err.Error(), "unregistered translation package tour-v1") {
		t.Fatal("registered-only API silently accepted native Tour", err)
	}
	return root, c
}

func makeNativeGenerationRunFixture(t *testing.T) nativeGenerationFixture {
	t.Helper()
	root, c := generationRunSourceFixture(t)
	f := currentValidationExportFixture{root: root, locale: "zh-CN", pkg: "tour-v1", snapshot: "run-predecessor", reopen: "run-current-validator", catalog: c}
	for _, p := range c.Pages {
		f.ids = append(f.ids, p.ID)
	}
	for _, e := range c.Examples {
		if e.EligibleTranslation {
			t.Fatal("fixture must not silently omit an eligible native Example")
		}
	}
	metadata := installCurrentValidationExportAuthority(t, f)
	f = completeCurrentValidationExportFixture(t, f, metadata)
	out, err := authorizationDiagnosisOutput(t, func() error {
		return retranslationExportLifecycleCommand(root, c, currentValidationExportArgs(f))
	})
	if err != nil {
		t.Fatal("isolated revision batch", err)
	}
	var batch i18n.RetranslationExportResult
	if err := json.Unmarshal(out, &batch); err != nil {
		t.Fatal(err)
	}
	return nativeGenerationFixture{f: f, batch: batch, output: filepath.Join(t.TempDir(), "generation.zip")}
}

func assertRunGenerationBundle(t *testing.T, f nativeGenerationFixture, revision bool) {
	t.Helper()
	files, err := i18n.ReadTransportBundle(courseContextRead(t, filepath.Dir(f.output), filepath.Base(f.output)), 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.GenerationBundleManifest
	if err := json.Unmarshal(files["manifest.json"], &m); err != nil || m.BatchID != f.batch.BatchID || m.Locale != f.f.locale || len(m.ExpectedOutputs) != 1 || m.ExpectedOutputs[0].UnitID != f.f.ids[0] {
		t.Fatal("run changed exact batch membership", err)
	}
	if !revision {
		return
	}
	if m.TaskKind != "translation-unit-revision" {
		t.Fatal("revision task lost")
	}
	var formal i18n.RetranslationBatchManifest
	if err := json.Unmarshal(files["formal/manifest.json"], &formal); err != nil || len(formal.Units) != 1 {
		t.Fatal("formal batch missing", err)
	}
	u := formal.Units[0]
	if u.RevisionFeedbackSource != i18n.CurrentValidationRevisionFeedback || u.RevisionAuthorizationID != f.f.reopen || u.PreviousSnapshotID != f.f.snapshot || u.PreviousRating != "A" || u.PreviousReviewSHA256 != courseContextSHA(courseContextRead(t, f.f.root, strings.TrimPrefix(f.f.receipt, f.f.root+string(filepath.Separator)))) {
		t.Fatal("run lost exact recorded current-validation authority")
	}
	if len(files["formal/current-validation-predecessor-transport/"+filepath.Base(u.InputPath)+".json"]) != 0 {
		t.Fatal("ordinary exact predecessor acquired unnecessary recovery authority")
	}
	for _, prefix := range []string{"formal/inputs/", "formal/candidates/", "formal/protected-candidates/"} {
		if len(files[prefix+filepath.Base(u.InputPath)]) == 0 {
			t.Fatal("run omitted predecessor representation", prefix)
		}
	}
	if !strings.Contains(string(files["generation-context.json"]), "ONLY the exact authorized teaching-comment bodies") {
		t.Fatal("run omitted bounded comment transport instructions")
	}
}

func runGenerationExport(t *testing.T, f nativeGenerationFixture) {
	t.Helper()
	before := authorizationDiagnosisInventory(t, f.f.root)
	out, err := authorizationDiagnosisOutput(t, func() error {
		// This is deliberately the actual public entry point, not the lifecycle
		// helper: load/hydrate -> attach packages -> shared selector -> handler.
		return run([]string{"generation-bundle", "export", "--content-package", f.f.pkg, "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--output", f.output})
	})
	if err != nil || !strings.Contains(string(out), "Generation bundle exported:") {
		t.Fatal("real run export", err, string(out))
	}
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
		t.Fatal("run export mutated fixture authority")
	}
}

func runGenerationCurrentCheck(t *testing.T, f nativeGenerationFixture) {
	t.Helper()
	before := authorizationDiagnosisInventory(t, f.f.root)
	out, err := authorizationDiagnosisOutput(t, func() error {
		return run([]string{"generation-bundle", "current-check", "--content-package", f.f.pkg, "--bundle", f.output})
	})
	if err != nil || !strings.Contains(string(out), "Generation bundle: CURRENT") {
		t.Fatal("real run independent current-check", err, string(out))
	}
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
		t.Fatal("run current-check mutated fixture authority")
	}
}

// run owns its catalog: these tests pass only argv and an isolated working
// directory. No supplied/pre-scoped Catalog can bypass the shared dispatcher.
func TestTourNativeGenerationBundleExportRun(t *testing.T) {
	f := makeNativeGenerationRunFixture(t)
	t.Chdir(f.f.root)
	runGenerationExport(t, f)
	assertRunGenerationBundle(t, f, true)
}

func TestTourNativeGenerationBundleCurrentCheckRun(t *testing.T) {
	f := makeNativeGenerationRunFixture(t)
	t.Chdir(f.f.root)
	runGenerationExport(t, f)
	runGenerationCurrentCheck(t, f)
	assertRunGenerationBundle(t, f, true)
	for _, pkg := range []string{"unknown-package", "learn-docs-v1"} {
		for _, mode := range []string{"export", "current-check"} {
			args := generationCheckArgs(f)
			output := filepath.Join(t.TempDir(), "rejected.zip")
			if mode == "export" {
				args = generationExportArgs(f, output)
			}
			args = replaceExportCLIFlag(t, args, "--content-package", pkg)
			want := "unregistered translation package"
			if pkg == "learn-docs-v1" {
				want = "outside selected complete package scope"
			}
			if _, err := authorizationDiagnosisOutput(t, func() error { return run(args) }); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatal("real run wrong package failure", mode, pkg, err)
			}
			if _, err := os.Lstat(output); !os.IsNotExist(err) {
				t.Fatal("rejected real run published a bundle")
			}
		}
	}
}

func TestRegisteredGenerationBundleLifecycleRun(t *testing.T) {
	root, c := generationRunSourceFixture(t)
	f := currentValidationExportFixture{root: root, locale: "en-GB", pkg: "learn-docs-v1", catalog: c}
	installCurrentValidationExportAuthority(t, f)
	scoped, err := i18n.SelectCurrentValidationReopenPackage(c, f.pkg)
	if err != nil {
		t.Fatal(err)
	}
	f.ids = append(f.ids, scoped.TranslationScope[0])
	batch, err := i18n.ExportRetranslationBatch(root, scoped, i18n.RetranslationExportOptions{Locale: f.locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: f.ids})
	if err != nil {
		t.Fatal("registered initial fixture", err)
	}
	fixture := nativeGenerationFixture{f: f, batch: *batch, output: filepath.Join(t.TempDir(), "registered.zip")}
	t.Chdir(root)
	runGenerationExport(t, fixture)
	runGenerationCurrentCheck(t, fixture)
	assertRunGenerationBundle(t, fixture, false)
}
