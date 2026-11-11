package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestGenerationBundleImportPackageParser(t *testing.T) {
	base := []string{"generation-bundle", "import", "--bundle", "generation.zip", "--input-dir", "outputs", "--provider", "chatgpt", "--model", "gpt-5.6-sol-high"}
	args := append(append([]string{}, base[:2]...), "--content-package", "learn-docs-v1")
	args = append(args, base[2:]...)
	pkg, remaining, err := parseTranslationContentPackageArgs(args)
	if err != nil || pkg != "learn-docs-v1" || !reflect.DeepEqual(remaining, base) {
		t.Fatal("shared parser lost explicit import package or handler flags", pkg, remaining, err)
	}
	for _, extra := range [][]string{{"--content-package"}, {"--content-package", ""}, {"--content-package", "learn-docs-v1", "--content-package", "learn-docs-v1"}} {
		if _, _, err := parseTranslationContentPackageArgs(append(append([]string{}, base...), extra...)); err == nil {
			t.Fatal("invalid package selector accepted", extra)
		}
	}
}

// Reuse the real registered Learn/Docs source fixture. Only synthetic locale,
// batch and outputs are installed, all under TempDir; run owns catalog loading.
func TestGenerationBundleImportPackageRun(t *testing.T) {
	for _, pkg := range []string{"learn-docs-v1", "tour-v1"} {
		for _, transport := range []string{"directory", "result-zip"} {
			t.Run(pkg+"/"+transport, func(t *testing.T) {
				root, c := generationRunSourceFixture(t)
				f := currentValidationExportFixture{root: root, locale: "en-GB", pkg: pkg, catalog: c}
				installCurrentValidationExportAuthority(t, f)
				scoped, err := i18n.SelectCurrentValidationReopenPackage(c, pkg)
				if err != nil {
					t.Fatal(err)
				}
				id := c.Pages[0].ID
				selector := []string{}
				if pkg != "tour-v1" {
					id = scoped.TranslationScope[0]
					selector = []string{"--content-package", pkg}
				}
				batch, err := i18n.ExportRetranslationBatch(root, scoped, i18n.RetranslationExportOptions{Locale: f.locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{id}})
				if err != nil {
					t.Fatal(err)
				}
				t.Chdir(root)
				bundlePath := filepath.Join(t.TempDir(), "generation.zip")
				for _, argv := range [][]string{
					append([]string{"generation-bundle", "export", "--locale", f.locale, "--batch-id", batch.BatchID, "--output", bundlePath}, selector...),
					append([]string{"generation-bundle", "current-check", "--bundle", bundlePath}, selector...),
				} {
					if _, err := authorizationDiagnosisOutput(t, func() error { return run(argv) }); err != nil {
						t.Fatal("export/current-check with same package", err)
					}
				}
				bundle := courseContextRead(t, filepath.Dir(bundlePath), filepath.Base(bundlePath))
				files, err := i18n.ReadTransportBundle(bundle, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
				if err != nil {
					t.Fatal(err)
				}
				var manifest i18n.GenerationBundleManifest
				if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
					t.Fatal(err)
				}
				outputs := t.TempDir()
				for _, expected := range manifest.ExpectedOutputs {
					name := filepath.Base(expected.BundlePath)
					courseContextWrite(t, outputs, name, append(bytes.TrimRight(files["formal/inputs/"+name], "\n"), '\n'))
				}
				args := []string{"generation-bundle", "import", "--bundle", bundlePath}
				if transport == "directory" {
					args = append(args, "--input-dir", outputs, "--provider", "chatgpt", "--model", "gpt-5.6-sol-high")
				} else {
					result := filepath.Join(t.TempDir(), "result.zip")
					pack := []string{"generation-bundle", "result-pack", "--bundle", bundlePath, "--input-dir", outputs, "--provider", "chatgpt", "--model", "gpt-5.6-sol-high", "--output", result}
					if pkg != "tour-v1" {
						before := authorizationDiagnosisInventory(t, root)
						if _, err := authorizationDiagnosisOutput(t, func() error { return run(pack) }); err == nil || !strings.Contains(err.Error(), "outside selected complete package scope") {
							t.Fatal("registered result-pack accepted default scope", err)
						}
						if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
							t.Fatal("rejected result-pack mutated state")
						}
						if _, err := os.Lstat(result); !os.IsNotExist(err) {
							t.Fatal("rejected result-pack published output", err)
						}
					}
					if _, err := authorizationDiagnosisOutput(t, func() error { return run(append(pack, selector...)) }); err != nil {
						t.Fatal("result-pack with same package", err)
					}
					args = append(args, "--result", result)
				}
				before := authorizationDiagnosisInventory(t, root)
				if pkg != "tour-v1" {
					// The production source fixture registers only Learn/Docs. Add a
					// disjoint synthetic descriptor to exercise the shared selector
					// and import handler with a valid, foreign registered scope.
					foreign := i18n.TranslationPackageDescriptor{ID: "foreign-import-test", AuthoritySHA256: courseContextSHA([]byte("foreign source authority")), Units: []i18n.AdaptedTranslationUnit{{
						Unit:    i18n.TranslationUnit{ID: "foreign-page", Kind: i18n.UnitKindPage, SourcePath: "fixtures/foreign.txt", Source: []byte("Foreign source.\n"), SourceSHA256: courseContextSHA([]byte("Foreign source.\n"))},
						Adapter: &authorizationDiagnosisAdapter{source: "Foreign source.\n"}, StableIndex: 1, Route: "/foreign",
					}}}
					if err := c.RegisterTranslationPackage(foreign); err != nil {
						t.Fatal(err)
					}
					foreignCatalog, err := i18n.SelectCurrentValidationReopenPackage(c, foreign.ID)
					if err != nil {
						t.Fatal("foreign fixture must be a valid complete registered scope", err)
					}
					if err := importGenerationResultBundleCommand(root, foreignCatalog, args[2:], foreign.ID); err == nil || !strings.Contains(err.Error(), "outside selected complete package scope") {
						t.Fatal("foreign registered scope accepted", err)
					}
					if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
						t.Fatal("rejected foreign registered import mutated state")
					}
					for _, tc := range []struct{ selector, want string }{
						{"", "outside selected complete package scope"},
						{"tour-v1", "outside selected complete package scope"},
						{"unknown-package", "unregistered translation package"},
					} {
						argv := append([]string{}, args...)
						if tc.selector != "" {
							argv = append(argv, "--content-package", tc.selector)
						}
						if _, err := authorizationDiagnosisOutput(t, func() error { return run(argv) }); err == nil || !strings.Contains(err.Error(), tc.want) {
							t.Fatal("wrong/missing selector accepted or wrong failure", tc.selector, err)
						}
						if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
							t.Fatal("rejected import mutated state", tc.selector)
						}
					}
				}
				args = append(args, selector...)
				if pkg != "tour-v1" {
					assertPackageImportRefusesFormalState(t, root, batch.BatchPath, args)
				}
				out, err := authorizationDiagnosisOutput(t, func() error { return run(args) })
				if err != nil || !strings.Contains(string(out), "Generation result imported:") {
					t.Fatal("import with caller-selected scope", err, string(out))
				}
				var receipt i18n.GenerationResultBundleManifest
				if err := json.Unmarshal(courseContextRead(t, root, batch.BatchPath+"/generation-imports/initial-manifest.json"), &receipt); err != nil {
					t.Fatal(err)
				}
				if receipt.Provider != "chatgpt" || receipt.Model != "gpt-5.6-sol-high" || receipt.GenerationBundleSHA256 != courseContextSHA(bundle) || receipt.InputIdentitySHA256 != manifest.InputIdentitySHA256 || receipt.BatchID != batch.BatchID || receipt.Locale != f.locale || receipt.Attempt != 1 || len(receipt.Outputs) != len(manifest.ExpectedOutputs) {
					t.Fatal("import provenance changed", receipt)
				}
				for _, expected := range manifest.ExpectedOutputs {
					if !bytes.Equal(courseContextRead(t, root, batch.BatchPath+"/"+expected.InstallPath), courseContextRead(t, outputs, filepath.Base(expected.BundlePath))) {
						t.Fatal("installed bytes changed")
					}
				}
				if _, err := os.Lstat(filepath.Join(root, batch.BatchPath, "generation-imports/import.pending")); !os.IsNotExist(err) {
					t.Fatal("pending marker remains", err)
				}
				after := authorizationDiagnosisInventory(t, root)
				for path, sha := range before {
					if after[path] != sha {
						t.Fatal("existing authority/state rewritten", path)
					}
				}
				if _, err := authorizationDiagnosisOutput(t, func() error { return run(args) }); err == nil || !strings.Contains(err.Error(), "already exists") {
					t.Fatal("import replay/overwrite accepted", err)
				}
				if !reflect.DeepEqual(after, authorizationDiagnosisInventory(t, root)) {
					t.Fatal("rejected replay mutated state")
				}
			})
		}
	}
}

func assertPackageImportRefusesFormalState(t *testing.T, root, batchPath string, args []string) {
	t.Helper()
	for _, tc := range []struct {
		path, want string
		directory  bool
	}{
		{"raw-responses", "already exists", true},
		{"candidates", "already exists", true},
		{"validation", "already exists", true},
		{"result.json", "already exists", false},
		{"generation-imports/initial-manifest.json", "provenance already exists", false},
		{"generation-imports/import.pending", "pending or unknown mutation state", false},
	} {
		t.Run("refuse-"+tc.path, func(t *testing.T) {
			path := filepath.Join(root, batchPath, tc.path)
			if tc.directory {
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			} else {
				courseContextWrite(t, root, batchPath+"/"+tc.path, []byte("synthetic existing state\n"))
			}
			before := authorizationDiagnosisInventory(t, root)
			if _, err := authorizationDiagnosisOutput(t, func() error { return run(args) }); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal("existing formal state did not block registered import", err)
			}
			if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
				t.Fatal("refused import changed state")
			}
			if _, err := os.Lstat(path); err != nil {
				t.Fatal("refused import removed existing state", err)
			}
			// Remove only this synthetic TempDir obstruction for the next case.
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}
