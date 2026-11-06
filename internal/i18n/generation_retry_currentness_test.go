package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func siblingRetryFixture(t *testing.T) (string, *Catalog, string, [][]byte, []GenerationBundleManifest) {
	t.Helper()
	root := t.TempDir()
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	writeRetranslationTestGlossaryForLocale(t, root, "zh-CN")
	recordSyntheticUnifiedReview(t, root, "synthetic-generation-review", "zh-CN")
	c := retranslationTestCatalog(3)
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorCodex, Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, e.BatchPath)
	m := readRetranslationManifest(t, root, e.BatchID)
	if err := os.Mkdir(filepath.Join(dir, "raw-responses"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, u := range m.Units {
		b, err := os.ReadFile(filepath.Join(dir, u.InputPath))
		if err != nil {
			t.Fatal(err)
		}
		b = bytes.Replace(b, []byte("⟪GTI18N_"), []byte("⟪BROKEN_"), 1)
		if err := os.WriteFile(filepath.Join(dir, "raw-responses", filepath.Base(u.InputPath)), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil || r.RestoreFailed != 3 {
		t.Fatalf("process=%+v err=%v", r, err)
	}
	bundles := [][]byte{}
	manifests := []GenerationBundleManifest{}
	for _, u := range m.Units {
		b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: u.UnitID})
		if err != nil {
			t.Fatal(err)
		}
		bundles = append(bundles, b)
		manifests = append(manifests, m)
	}
	return root, c, e.BatchID, bundles, manifests
}

func TestRetryGenerationBundleSiblingCurrentness(t *testing.T) {
	root, c, batch, bundles, manifests := siblingRetryFixture(t)
	for i, b := range bundles {
		if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, b); err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		writeGenerationFixtureOutputs(t, root, batch, dir, manifests[i])
		if i == 1 {
			packed, _, err := PackGenerationResultBundle(root, c, b, "codex", FormalGenerationModel, dir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ImportGenerationResultBundle(root, c, b, packed); err != nil {
				t.Fatal(err)
			}
		} else {
			if _, err := ImportGenerationOutputDirectory(root, c, b, "codex", FormalGenerationModel, dir); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, b); err == nil {
			t.Fatal("installed retry still current")
		}
		r, err := ProcessRetranslationRetry(root, c, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batch, UnitID: manifests[i].ExpectedOutputs[0].UnitID})
		if err != nil || r.ValidationPassed != i+1 {
			t.Fatalf("retry=%+v err=%v", r, err)
		}
		for _, pending := range bundles[i+1:] {
			if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, pending); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestRetryGenerationBundleRejectsTargetDriftAndTampering(t *testing.T) {
	for _, kind := range []string{"validation", "attempt", "status", "source", "input", "protected", "glossary", "manifest", "install", "result", "result-resealed", "authority", "raw", "provenance"} {
		t.Run(kind, func(t *testing.T) {
			root, c, batch, bundles, manifests := siblingRetryFixture(t)
			b := bundles[1]
			m := manifests[1]
			dir := filepath.Join(root, "data/retranslation-runs/zh-CN", batch)
			_, files, err := readGenerationBundle(b)
			if err != nil {
				t.Fatal(err)
			}
			mutate := func(path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(data, ' '), 0644); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "validation", "attempt", "status":
				vpath := ""
				for _, f := range m.Files {
					if f.BundlePath == "formal/retry/current-validation.json" {
						vpath = filepath.Join(root, f.RepositoryPath)
					}
				}
				var v RetranslationValidation
				json.Unmarshal(files["formal/retry/current-validation.json"], &v)
				if kind == "validation" {
					v.Error += " changed"
				}
				if kind == "attempt" {
					v.Attempt++
					v.RawResponsePath = m.ExpectedOutputs[0].InstallPath
				}
				if kind == "status" {
					v.Status = "passed"
				}
				data, _ := json.MarshalIndent(v, "", "  ")
				os.WriteFile(vpath, append(data, '\n'), 0644)
			case "source":
				c.Pages[1].Source = append(c.Pages[1].Source, []byte(" changed")...)
			case "input":
				mutate(filepath.Join(dir, "inputs", filepath.Base(m.ExpectedOutputs[0].BundlePath)))
			case "protected":
				formal := readRetranslationManifest(t, root, batch)
				formal.Units[1].ProtectedTokenCount++
				data, _ := json.MarshalIndent(formal, "", "  ")
				os.WriteFile(filepath.Join(dir, "manifest.json"), append(data, '\n'), 0644)
			case "glossary":
				mutate(filepath.Join(root, "locales/zh-CN/glossary.yaml"))
			case "manifest":
				mutate(filepath.Join(dir, "manifest.json"))
			case "raw":
				path := filepath.Join(dir, m.ExpectedOutputs[0].InstallPath)
				os.MkdirAll(filepath.Dir(path), 0755)
				os.WriteFile(path, []byte("existing\n"), 0644)
			case "provenance":
				path := generationImportReceiptPath(dir, m)
				os.MkdirAll(filepath.Dir(path), 0755)
				os.WriteFile(path, []byte("existing\n"), 0644)
			default:
				if kind == "install" {
					m.ExpectedOutputs[0].InstallPath = strings.Replace(m.ExpectedOutputs[0].InstallPath, "002", "003", 1)
				} else {
					name := "formal/retry/result.json"
					if kind == "authority" {
						name = "authority/AGENTS.md"
					}
					if kind == "result-resealed" {
						var r RetranslationProcessResult
						json.Unmarshal(files[name], &r)
						r.Units[0].Error += " forged"
						data, _ := json.MarshalIndent(r, "", "  ")
						files[name] = append(data, '\n')
					} else {
						files[name] = append(files[name], ' ')
					}
				}
				// Re-seal hashes and identity to ensure semantic/canonical checks also fail.
				var identity strings.Builder
				entries := []TransportBundleEntry{}
				for i, f := range m.Files {
					m.Files[i] = NewTransportBundleFile(f.BundlePath, f.RepositoryPath, files[f.BundlePath])
					identity.WriteString(f.BundlePath + "\x00" + m.Files[i].SHA256 + "\x00")
					entries = append(entries, TransportBundleEntry{Path: f.BundlePath, Data: files[f.BundlePath]})
				}
				m.InputIdentitySHA256 = sum([]byte(identity.String()))
				data, _ := json.MarshalIndent(m, "", "  ")
				b, err = WriteDeterministicTransportBundle(append(data, '\n'), entries)
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, b); err == nil {
				t.Fatal("stale or tampered retry accepted")
			}
		})
	}
}
