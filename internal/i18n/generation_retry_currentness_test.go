package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestRetryHandoffReadonlyInputScopeOwnership(t *testing.T) {
	root, c, _, bundles, _ := siblingRetryFixture(t)
	want, _, err := ExportRetryHandoff(root, c, bundles)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	active := false
	c.ReadonlyInputContext = func(r string, catalog *Catalog, work func(*Catalog) error) error {
		if r != root || active {
			t.Fatal("input owner crossed root or nested invocation")
		}
		calls++
		active = true
		defer func() { active = false }()
		return work(catalog)
	}
	got, _, err := ExportRetryHandoff(root, c, bundles)
	if err != nil || !bytes.Equal(want, got) || calls != 1 {
		t.Fatal("member checks did not share one owner or changed bytes", calls, err)
	}
	if _, err := VerifyCurrentRetryHandoff(root, c, got); err != nil || calls != 2 {
		t.Fatal("new command retained a prior owner", calls, err)
	}
	if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, bundles[0]); err != nil || calls != 3 {
		t.Fatal("single check did not close its own phase", calls, err)
	}
	if active || c.ReadonlyInputContext == nil {
		t.Fatal("caller Catalog/lifetime changed")
	}
	c.ReadonlyInputContext = func(_ string, catalog *Catalog, work func(*Catalog) error) error {
		if err := work(catalog); err != nil {
			return err
		}
		return fmt.Errorf("fresh final input guard failed")
	}
	if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, bundles[0]); err == nil {
		t.Fatal("CURRENT escaped a failed final guard")
	}
	if _, _, err := ExportRetryHandoff(root, c, bundles); err == nil {
		t.Fatal("handoff escaped a failed final guard")
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

// Both attempts are formally imported. Attempt 2 fixes one ordinary-text line
// while another remains invalid, so retry must not return to the QC template.
func ordinaryRetryPredecessorFixture(t *testing.T) (string, *Catalog, GenerationBundleOptions) {
	t.Helper()
	root, c, options := revisionGenerationFixture(t)
	bundle, m, err := ExportTranslationUnitGenerationBundle(root, c, options)
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for i, out := range m.ExpectedOutputs {
		data := files["formal/protected-candidates/"+filepath.Base(out.BundlePath)]
		if i == 0 {
			options.UnitID = out.UnitID
			data = []byte(appendBeforeRetranslationArtifactEOF(string(data), "\n第一处修复。  \n第二处修复。  "))
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(out.BundlePath)), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ImportGenerationOutputDirectory(root, c, bundle, "codex", FormalGenerationModel, dir); err != nil {
		t.Fatal(err)
	}
	result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: options.Locale, BatchID: options.BatchID})
	if err != nil || result.ValidationFailed != 1 || result.ValidationPassed != 1 || !strings.HasPrefix(result.Units[0].Error, "candidate has trailing whitespace at line ") {
		t.Fatal("initial failure fixture", result, err)
	}
	retry, m, err := ExportTranslationUnitGenerationBundle(root, c, options)
	if err != nil || m.Attempt != 2 {
		t.Fatal("attempt 2 export", m, err)
	}
	dir = t.TempDir()
	// Read the formal raw rather than assuming the new member exists: this
	// fixture also reproduces the missing-baseline bug in the old exporter.
	rawPath := filepath.Join(root, "data/retranslation-runs", options.Locale, options.BatchID, "raw-responses", filepath.Base(m.ExpectedOutputs[0].BundlePath))
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	data := bytes.Replace(raw, []byte("第一处修复。  \n"), []byte("第一处修复。\n"), 1)
	if bytes.Equal(data, raw) {
		t.Fatal("fixture did not correct the first line")
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.Base(m.ExpectedOutputs[0].BundlePath)), data, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportGenerationOutputDirectory(root, c, retry, "codex", FormalGenerationModel, dir); err != nil {
		t.Fatal(err)
	}
	result, err = ProcessRetranslationRetry(root, c, RetranslationRetryOptions{Locale: options.Locale, BatchID: options.BatchID, UnitID: options.UnitID})
	if err != nil || result.ValidationFailed != 1 || result.ValidationPassed != 1 || result.RetryAttempt != 2 || !strings.HasPrefix(result.Units[0].Error, "candidate has trailing whitespace at line ") {
		t.Fatal("attempt 2 failure fixture", result, err)
	}
	return root, c, options
}

func TestOrdinaryRetryGenerationBundleCurrentPredecessor(t *testing.T) {
	root, c, options := ordinaryRetryPredecessorFixture(t)
	bundle, m, err := ExportTranslationUnitGenerationBundle(root, c, options)
	if err != nil || m.Attempt != 3 || m.TaskKind != "translation-unit-retry" || len(m.ExpectedOutputs) != 1 {
		t.Fatal(m, err)
	}
	_, files, err := readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	var v RetranslationValidation
	if err := json.Unmarshal(files["formal/retry/current-validation.json"], &v); err != nil || v.Attempt != 2 || v.Status != "validation_failed" {
		t.Fatal(v, err)
	}
	rawRepo := filepath.ToSlash(filepath.Join("data/retranslation-runs", options.Locale, options.BatchID, v.RawResponsePath))
	want, err := readCompatibilityFile(root, rawRepo)
	baseline := files["formal/retry/predecessor-response.article"]
	if err != nil || !bytes.Equal(baseline, want) || !bytes.Contains(baseline, []byte("第一处修复。\n第二处修复。  \n")) {
		t.Fatal("current imported correction was lost", err)
	}
	name := filepath.Base(m.ExpectedOutputs[0].BundlePath)
	if bytes.Equal(baseline, files["formal/protected-candidates/"+name]) {
		t.Fatal("fixture failed to distinguish current raw from original revision baseline")
	}
	found := false
	for _, f := range m.Files {
		if f.BundlePath == "formal/retry/predecessor-response.article" {
			found = f == NewTransportBundleFile(f.BundlePath, rawRepo, want)
		}
	}
	if !found || len(files["formal/retry/predecessor-import.json"]) == 0 {
		t.Fatal("current predecessor lacks exact inventory/provenance binding")
	}
	var context generationContext
	if err := json.Unmarshal(files["generation-context.json"], &context); err != nil ||
		!strings.Contains(context.Instruction, "ONLY formal/retry/predecessor-response.article") ||
		!strings.Contains(context.Instruction, "never revert to that older template") ||
		!strings.Contains(context.Instruction, "not language approval") {
		t.Fatal("retry baseline precedence is ambiguous", context, err)
	}
	assertRevisionCandidates(t, root, c, options, bundle)
	if _, codec := files[attemptWhitespaceMember]; codec {
		t.Fatal("ordinary Generation-owned whitespace acquired codec authority")
	}
	codecOptions := options
	codecOptions.ProtectedWhitespaceRepair = true
	if _, _, err := ExportTranslationUnitGenerationBundle(root, c, codecOptions); err == nil {
		t.Fatal("ordinary whitespace admitted as protected-only codec repair")
	}
	if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err != nil {
		t.Fatal(err)
	}
	again, _, err := ExportTranslationUnitGenerationBundle(root, c, options)
	if err != nil || !bytes.Equal(bundle, again) {
		t.Fatal("same-input export is not deterministic", err)
	}
	// Complete only the remaining correction from the new baseline. Existing
	// import/process semantics must accept it without regressing the first fix.
	dir := t.TempDir()
	fixed := bytes.Replace(baseline, []byte("第二处修复。  \n"), []byte("第二处修复。\n"), 1)
	if err := os.WriteFile(filepath.Join(dir, name), fixed, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportGenerationOutputDirectory(root, c, bundle, "codex", FormalGenerationModel, dir); err != nil {
		t.Fatal(err)
	}
	result, err := ProcessRetranslationRetry(root, c, RetranslationRetryOptions{Locale: options.Locale, BatchID: options.BatchID, UnitID: options.UnitID})
	if err != nil || result.ValidationPassed != 2 || result.RetryAttempt != 3 {
		t.Fatal("minimal successor failed existing validation", result, err)
	}
}

func TestOrdinaryRetryGenerationBundlePredecessorDrift(t *testing.T) {
	for _, kind := range []string{"raw-change", "raw-delete", "validation-attempt", "validation-path", "import-delete", "import-change", "archive-delete", "bundle-change", "bundle-delete"} {
		t.Run(kind, func(t *testing.T) {
			root, c, options := ordinaryRetryPredecessorFixture(t)
			bundle, m, err := ExportTranslationUnitGenerationBundle(root, c, options)
			if err != nil {
				t.Fatal(err)
			}
			_, files, err := readGenerationBundle(bundle)
			if err != nil {
				t.Fatal(err)
			}
			member := "formal/retry/predecessor-response.article"
			if strings.HasPrefix(kind, "bundle-") {
				// Re-seal inventory and identity, not merely corrupt ZIP bytes.
				if kind == "bundle-delete" {
					delete(files, member)
				} else {
					files[member] = append([]byte("forged\n"), files[member]...)
				}
				m.Files = nil
				var identity strings.Builder
				var entries []TransportBundleEntry
				original, _, _ := readGenerationBundle(bundle)
				for _, f := range original.Files {
					if data, ok := files[f.BundlePath]; ok {
						f = NewTransportBundleFile(f.BundlePath, f.RepositoryPath, data)
						m.Files = append(m.Files, f)
						entries = append(entries, TransportBundleEntry{Path: f.BundlePath, Data: data})
						identity.WriteString(f.BundlePath + "\x00" + f.SHA256 + "\x00")
					}
				}
				m.InputIdentitySHA256 = sum([]byte(identity.String()))
				bundle, err = writeTranslationUnitGenerationBundle(mustJSON(m), entries)
			} else {
				var v RetranslationValidation
				if err := json.Unmarshal(files["formal/retry/current-validation.json"], &v); err != nil {
					t.Fatal(err)
				}
				paths := map[string]string{}
				for _, f := range m.Files {
					paths[f.BundlePath] = filepath.Join(root, filepath.FromSlash(f.RepositoryPath))
				}
				switch kind {
				case "raw-change":
					err = os.WriteFile(paths[member], append([]byte("changed\n"), files[member]...), 0644)
				case "raw-delete":
					err = os.Remove(paths[member])
				case "validation-attempt", "validation-path":
					if kind == "validation-attempt" {
						v.Attempt++
					} else {
						v.RawResponsePath = "raw-responses/" + filepath.Base(m.ExpectedOutputs[0].BundlePath)
					}
					err = os.WriteFile(paths["formal/retry/current-validation.json"], mustJSON(v), 0644)
				case "import-delete":
					err = os.Remove(paths["formal/retry/predecessor-import.json"])
				case "import-change":
					err = os.WriteFile(paths["formal/retry/predecessor-import.json"], []byte("{}\n"), 0644)
				case "archive-delete":
					var receipt GenerationResultBundleManifest
					if err := json.Unmarshal(files["formal/retry/predecessor-import.json"], &receipt); err != nil {
						t.Fatal(err)
					}
					err = os.Remove(filepath.Join(root, importedGenerationArchive(options.Locale, receipt.GenerationBundleSHA256)))
				}
				if err == nil {
					if _, _, err := ExportTranslationUnitGenerationBundle(root, c, options); err == nil {
						t.Fatal("invalid/missing predecessor exported")
					}
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err == nil {
				t.Fatal("stale/tampered predecessor remained CURRENT")
			}
		})
	}
}
