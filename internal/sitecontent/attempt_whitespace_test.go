package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func whitespacePageFixture(t *testing.T, source string, path string) (PageAdapter, []byte) {
	t.Helper()
	b := []byte(source)
	d, err := ParseWorkflowDocument(Source{Path: path, Kind: "page", SHA256: digest(b), Route: "/doc/whitespace"}, b, 1)
	if err != nil {
		t.Fatal(err)
	}
	a := PageAdapter{*d, b}
	raw := d.Units[0].Source
	for i, p := range d.Units[0].Protected {
		if p.Role != "atom" || !bridgeFormattingWhitespace(p.Raw) || len(p.Raw) < 2 {
			continue
		}
		for _, boundary := range d.Units[0].Protected[i+1:] {
			if boundary.Role == "boundary" || bridgeFormattingWhitespace(boundary.Raw) {
				raw = strings.Replace(raw, p.Token, "", 1)
				raw = strings.Replace(raw, boundary.Token, p.Token+boundary.Token, 1)
				return a, bridgeEOF([]byte(raw))
			}
		}
	}
	t.Fatal("fixture has no source formatting atom")
	return a, nil
}

func TestStructuralWhitespaceSourceProof(t *testing.T) {
	for _, source := range []struct{ path, text string }{
		{"_content/doc/whitespace.md", "# Fixture\n\nThis prose wraps\n    before `API` and [link](https://example.test).\n\nAnother sentence.\n"},
		{"_content/doc/whitespace.html", "<p>This prose wraps\n  before a\n  <a href=\"https://example.test\">link</a>.</p>\n<p>Another sentence.</p>\n"},
	} {
		t.Run(source.path, func(t *testing.T) {
			a, raw := whitespacePageFixture(t, source.text, source.path)
			before := bytes.Clone(raw)
			candidate, err := a.Restore(string(raw))
			if err != nil {
				t.Fatal(err)
			}
			repairs, identity, err := a.InspectStructuralWhitespace(raw)
			if err != nil || identity == "" || len(repairs) != 1 || !repairs[0].OpaqueLine || len(repairs[0].Origins) != 1 {
				t.Fatal(repairs, identity, err)
			}
			r := repairs[0]
			if string(candidate[r.Start:r.End]) != r.Whitespace || strings.Trim(r.Whitespace, " \t") != "" {
				t.Fatal("wrong source/candidate interval")
			}
			if !bytes.Equal(before, raw) {
				t.Fatal("inspection rewrote historical raw")
			}
			clean := []byte(a.Document.Units[0].Source)
			x, _, err := a.InspectStructuralWhitespace(clean)
			if err != nil || len(x) != 0 {
				t.Fatal("canonical clean source changed", x, err)
			}
			if _, _, err := a.InspectStructuralWhitespace(bytes.Replace(raw, []byte("⟦P0⟧"), nil, 1)); err == nil {
				t.Fatal("token tamper accepted")
			}
			bad := a
			bad.Document.Slots = append(append([]Unit(nil), a.Document.Slots...), a.Document.Slots[0])
			if _, _, err := bad.InspectStructuralWhitespace(raw); err == nil {
				t.Fatal("ambiguous source ownership accepted")
			}
		})
	}
}

func TestStructuralWhitespaceRejectsOpaqueOrStyledBytes(t *testing.T) {
	for _, source := range []struct{ path, text string }{
		{"_content/doc/code.md", "# Fixture\n\nText.\n\n```go\nvar s = `first\n  \nlast`\n```\n"},
		{"_content/doc/code.html", "<p>Text.</p>\n<pre>code\n  \ncode</pre>\n"},
		{"_content/doc/styled.html", "<p style=\"white-space:pre-wrap\">This wraps\n  before another\n  word.</p>\n<p>End.</p>\n"},
	} {
		t.Run(source.path, func(t *testing.T) {
			var a PageAdapter
			var raw []byte
			if strings.Contains(source.text, "style=") {
				a, raw = whitespacePageFixture(t, source.text, source.path)
			} else {
				b := []byte(source.text)
				d, e := ParseWorkflowDocument(Source{Path: source.path, Kind: "page", SHA256: digest(b)}, b, 1)
				if e != nil {
					t.Fatal(e)
				}
				a = PageAdapter{*d, b}
				raw = []byte(d.Units[0].Source)
			}
			if _, _, err := a.InspectStructuralWhitespace(raw); err == nil {
				t.Fatal("opaque code/literal or styled whitespace became editable")
			}
		})
	}
}

func TestStructuralWhitespaceAttemptWorkflow(t *testing.T) {
	for _, transport := range []string{"directory", "result-zip", "ordinary-editable"} {
		t.Run(transport, func(t *testing.T) {
			root := siteFixture(t, "docs/RETRANSLATION_RUNBOOK.md", "docs/TRANSLATION_QUALITY_REVIEW.md", "docs/LANGUAGE_DEFECT_EXHAUSTION.md", "docs/SURFACE_REVISION_TRANSPORT.md", "current-authorization-fresh-fixture")
			a, raw := whitespacePageFixture(t, "# Fixture\n\nThis prose wraps\n    before `API` and [link](https://example.test).\n\nAnother sentence.\n", "_content/doc/whitespace.md")
			if transport == "ordinary-editable" {
				last := a.Document.Units[0].Protected[len(a.Document.Units[0].Protected)-1].Token
				raw = bridgeEOF([]byte(strings.Replace(a.Document.Units[0].Source, last, " "+last, 1)))
			}
			writeFixture(t, root, a.Document.Path, a.Source)
			id := a.Document.Units[0].ID
			c := &i18n.Catalog{}
			desc := i18n.TranslationPackageDescriptor{ID: "synthetic-whitespace", AuthoritySHA256: digest([]byte("synthetic-whitespace")), Units: []i18n.AdaptedTranslationUnit{{Unit: i18n.TranslationUnit{ID: id, Kind: i18n.UnitKindPage, SourcePath: a.Document.Path, Source: a.Source, SourceSHA256: digest(a.Source)}, Adapter: a, StableIndex: 1, Route: a.Document.Route, WorkingSet: "batch-1", WorkingSetLimit: 30}}}
			if err := c.RegisterTranslationPackage(desc); err != nil {
				t.Fatal(err)
			}
			if err := c.UseTranslationUnitScope([]string{id}); err != nil {
				t.Fatal(err)
			}
			e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: []string{id}})
			if err != nil {
				t.Fatal(err)
			}
			bundle, initial, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			name := filepath.Base(initial.ExpectedOutputs[0].BundlePath)
			writeFixture(t, dir, name, raw)
			if _, err := i18n.ImportGenerationOutputDirectory(root, c, bundle, "chatgpt", i18n.FormalChatGPTGenerationModel, dir); err != nil {
				t.Fatal(err)
			}
			result, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID})
			if result == nil || result.ValidationFailed != 1 {
				t.Fatal(result, err)
			}
			batchDir := filepath.Join(root, e.BatchPath)
			validationPath := filepath.Join(batchDir, "validation", id+".json")
			validation, err := os.ReadFile(validationPath)
			if err != nil {
				t.Fatal(err)
			}
			if transport == "ordinary-editable" {
				retry, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id})
				if err != nil {
					t.Fatal("ordinary editable retry rejected", err)
				}
				files, err := i18n.ReadTransportBundle(retry, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
				if err != nil {
					t.Fatal(err)
				}
				if _, present := files["formal/whitespace-attempt/codec.json"]; present {
					t.Fatal("editable whitespace acquired a codec")
				}
				if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id, ProtectedWhitespaceRepair: true}); err == nil {
					t.Fatal("codec admitted without protected-only failure")
				}
				clean := t.TempDir()
				writeFixture(t, clean, name, bridgeEOF([]byte(a.Document.Units[0].Source)))
				if _, err := i18n.ImportGenerationOutputDirectory(root, c, retry, "chatgpt", i18n.FormalChatGPTGenerationModel, clean); err != nil {
					t.Fatal("ordinary directory retry import", err)
				}
				if r, err := i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id}); err != nil || r.ValidationPassed != 1 {
					t.Fatal("ordinary retry changed", r, err)
				}
				return
			}
			if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id}); err == nil || !strings.Contains(err.Error(), "protected-response representation cannot express this validation repair") {
				t.Fatal("impossible ordinary retry admitted", err)
			}
			retry, gm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id, ProtectedWhitespaceRepair: true})
			if err != nil {
				t.Fatal(err)
			}
			if gm.Attempt != 2 || gm.TaskKind != "translation-unit-retry" || len(gm.ExpectedOutputs) != 1 || gm.ExpectedOutputs[0].UnitID != id {
				t.Fatal("attempt/scope drift", gm)
			}
			if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, retry); err != nil {
				t.Fatal(err)
			}
			files, err := i18n.ReadTransportBundle(retry, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
			if err != nil {
				t.Fatal(err)
			}
			template := files["formal/whitespace-attempt/response.article"]
			if bytes.Contains(template, []byte("This prose")) || !bytes.Contains(template, []byte("GTI18N_WS1_")) {
				t.Fatal("codec exposed language")
			}
			fixed := bytes.ReplaceAll(template, []byte(" "), nil)
			staging := t.TempDir()
			writeFixture(t, staging, name, fixed)
			// Baseline retains exact failure; it is never eligible to import as PASS.
			bad := t.TempDir()
			writeFixture(t, bad, name, template)
			if _, err := i18n.ImportGenerationOutputDirectory(root, c, retry, "chatgpt", i18n.FormalChatGPTGenerationModel, bad); err == nil {
				t.Fatal("unrepaired baseline imported")
			}
			for _, badRaw := range [][]byte{append(bytes.Clone(fixed), []byte("prose")...), bytes.Replace(fixed, []byte("⟪GTI18N_WS1_"), []byte("⟪WRONG_"), 1)} {
				writeFixture(t, bad, name, badRaw)
				if _, err := i18n.ImportGenerationOutputDirectory(root, c, retry, "chatgpt", i18n.FormalChatGPTGenerationModel, bad); err == nil {
					t.Fatal("token/prose modification gained authority")
				}
			}
			if transport == "result-zip" {
				packed, _, err := i18n.PackGenerationResultBundle(root, c, retry, "chatgpt", i18n.FormalChatGPTGenerationModel, staging)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := i18n.ImportGenerationResultBundle(root, c, retry, packed); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := i18n.ImportGenerationOutputDirectory(root, c, retry, "chatgpt", i18n.FormalChatGPTGenerationModel, staging); err != nil {
					t.Fatal(err)
				}
			}
			result, err = i18n.ProcessRetranslationRetry(root, c, i18n.RetranslationRetryOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id})
			if err != nil || result.ValidationPassed != 1 {
				t.Fatal(result, err)
			}
			oldRaw, err := os.ReadFile(filepath.Join(batchDir, "raw-responses", name))
			if err != nil || !bytes.Equal(oldRaw, raw) {
				t.Fatal("old raw overwritten", err)
			}
			history, err := os.ReadFile(filepath.Join(batchDir, "retries", id, "attempt-001-validation.json"))
			if err != nil || !bytes.Equal(history, validation) {
				t.Fatal("old validation not exact immutable history", err)
			}
			var v i18n.RetranslationValidation
			b, err := os.ReadFile(validationPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &v); err != nil || v.Attempt != 2 || v.Status != "passed" || v.CandidateSHA256 == "" {
				t.Fatal(v, err)
			}
			candidate, err := os.ReadFile(filepath.Join(batchDir, v.CandidatePath))
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range bytes.Split(candidate, []byte{'\n'}) {
				if len(line) > 0 && bytes.ContainsAny(line[len(line)-1:], " \t\r") {
					t.Fatal("repair retained trailing whitespace")
				}
			}
			if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "whitespace-repaired"}); err != nil {
				t.Fatal("exact attempt historical reader", err)
			}
			revalidation, err := i18n.RevalidateRetranslationCandidate(root, c, i18n.RetranslationRevalidateOptions{Locale: "zh-CN", BatchID: e.BatchID, UnitID: id})
			if err != nil || revalidation.Status != "passed" || revalidation.Attempt != 2 {
				t.Fatal("exact-attempt revalidation", revalidation, err)
			}
			// A newer unrelated receipt cannot replace the exact selected attempt.
			writeFixture(t, batchDir, "generation-imports/"+id+"-attempt-003-manifest.json", []byte("not authority"))
			if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "whitespace-after-revalidate"}); err != nil {
				t.Fatal("reader searched mutable latest invocation", err)
			}
			scope, err := i18n.BuildQualityCheckScope(root, c, i18n.QualityCheckScopeOptions{Locale: "zh-CN", SnapshotID: "whitespace-repaired"})
			if err != nil {
				t.Fatal("QC reader", err)
			}
			if scope.PendingCount != 1 || scope.ACount != 0 || scope.ReadyForFinalization {
				t.Fatal("codec generated language approval", scope)
			}
			if transport == "directory" {
				var codec map[string]json.RawMessage
				if err := json.Unmarshal(files["formal/whitespace-attempt/codec.json"], &codec); err != nil {
					t.Fatal(err)
				}
				var oldIdentity string
				if err := json.Unmarshal(codec["identity_sha256"], &oldIdentity); err != nil {
					t.Fatal(err)
				}
				var ranges []i18n.StructuralWhitespaceRepair
				if err := json.Unmarshal(codec["repairs"], &ranges); err != nil {
					t.Fatal(err)
				}
				tamper := map[string][2]string{
					"unit":     {"\"unit_id\":\"" + id + "\"", "\"unit_id\":\"foreign\""},
					"attempt":  {"\"attempt\":2", "\"attempt\":3"},
					"source":   {"\"source_sha256\":\"" + digest(a.Source) + "\"", "\"source_sha256\":\"" + strings.Repeat("0", 64) + "\""},
					"input":    {"\"input_sha256\":\"" + digest(files["formal/inputs/"+name]) + "\"", "\"input_sha256\":\"" + strings.Repeat("0", 64) + "\""},
					"boundary": {fmt.Sprintf("\"source_start\":%d", ranges[0].Origins[0].SourceStart), fmt.Sprintf("\"source_start\":%d", ranges[0].Origins[0].SourceStart+1)},
					"schema":   {"go-tour-i18n/protected-structural-whitespace-attempt/v1", "foreign-schema"},
				}
				selected := filepath.Join(batchDir, "generation-imports", id+"-attempt-002-manifest.json")
				originalReceipt, err := os.ReadFile(selected)
				if err != nil {
					t.Fatal(err)
				}
				for kind, pair := range tamper {
					t.Run("historical-"+kind, func(t *testing.T) {
						badBundle := rewriteSurfaceGenerationBundle(t, retry, func(fs map[string][]byte) {
							key := "formal/whitespace-attempt/codec.json"
							b := bytes.Replace(fs[key], []byte(pair[0]), []byte(pair[1]), 1)
							if bytes.Equal(b, fs[key]) {
								t.Fatal("tamper fixture did not change field", kind)
							}
							b = bytes.Replace(b, []byte("\"identity_sha256\":\""+oldIdentity+"\""), []byte("\"identity_sha256\":\"\""), 1)
							newIdentity := digest(b)
							fs[key] = bytes.Replace(b, []byte("\"identity_sha256\":\"\""), []byte("\"identity_sha256\":\""+newIdentity+"\""), 1)
						})
						fs, err := i18n.ReadTransportBundle(badBundle, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
						if err != nil {
							t.Fatal(err)
						}
						var badGM i18n.GenerationBundleManifest
						if err := json.Unmarshal(fs["manifest.json"], &badGM); err != nil {
							t.Fatal(err)
						}
						var receipt i18n.GenerationResultBundleManifest
						if err := json.Unmarshal(originalReceipt, &receipt); err != nil {
							t.Fatal(err)
						}
						receipt.GenerationBundleSHA256 = digest(badBundle)
						receipt.InputIdentitySHA256 = badGM.InputIdentitySHA256
						receipt.GenerationManifest = i18n.NewTransportBundleFile("generation-manifest.json", "", fs["manifest.json"])
						data, err := json.MarshalIndent(receipt, "", "  ")
						if err != nil {
							t.Fatal(err)
						}
						writeFixture(t, root, "data/generation-archives/zh-CN/"+receipt.GenerationBundleSHA256+".zip", badBundle)
						writeFixture(t, batchDir, "generation-imports/"+id+"-attempt-002-manifest.json", append(data, '\n'))
						defer writeFixture(t, batchDir, "generation-imports/"+id+"-attempt-002-manifest.json", originalReceipt)
						if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "wrong-" + kind}); err == nil {
							t.Fatal("resealed wrong codec accepted by historical reader", kind)
						}
					})
				}
			}
			// Corrupt the selected archive: a newer receipt is never a fallback.
			var imported i18n.GenerationResultBundleManifest
			b, err = os.ReadFile(filepath.Join(batchDir, "generation-imports", id+"-attempt-002-manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(b, &imported); err != nil {
				t.Fatal(err)
			}
			archivePath := filepath.Join(root, "data/generation-archives/zh-CN", imported.GenerationBundleSHA256+".zip")
			archive, err := os.ReadFile(archivePath)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, filepath.ToSlash(strings.TrimPrefix(archivePath, root+string(filepath.Separator))), append(bytes.Clone(archive), 0))
			if _, _, err := i18n.CreateQualityCheckCandidateSnapshot(root, c, i18n.QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "whitespace-tampered"}); err == nil {
				t.Fatal("tampered selected archive accepted/latest fallback")
			}
		})
	}
}

func TestStructuralWhitespaceRealSourceProof(t *testing.T) {
	if os.Getenv("GO_TOUR_WHITESPACE_DIAGNOSIS") != "1" {
		t.Skip("explicit readonly source/raw diagnosis")
	}
	root := repositoryRoot(t)
	dir := filepath.Join(root, "data/retranslation-runs/zh-CN/chatgpt-zh-CN-073")
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest i18n.RetranslationBatchManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	ids := map[string]int{"704fb9e2968f11379508a1f22a9837e154dd94ef14e4f781ecd7d7ed701a13a3": 9, "36346dcd0f2901eadeb8f4ec8c70dc4c0e709457e72ed30880c9e471e18c398a": 77, "c3dfdb3d129df54ca09e79e21054c0afa5d3ec1bbd4e6cbeb227476bd8dcb0c3": 99, "61c3a8166079f2411ded5981623c0e2c7945ee164f8ac11b17d4f956b214174b": 21}
	count := 0
	for _, r := range manifest.Units {
		line, ok := ids[r.UnitID]
		if !ok {
			continue
		}
		count++
		source, err := os.ReadFile(filepath.Join(root, r.SourcePath))
		if err != nil || digest(source) != r.SourceSHA256 {
			t.Fatal("source drift", err)
		}
		d, err := ParseWorkflowDocument(Source{Path: r.SourcePath, Kind: "page", SHA256: digest(source)}, source, 1)
		if err != nil || d.Units[0].ID != r.UnitID {
			t.Fatal("formal identity drift", err)
		}
		raw, err := os.ReadFile(filepath.Join(dir, "raw-responses", r.UnitID+".article"))
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := os.ReadFile(filepath.Join(dir, "candidates", r.UnitID+".article"))
		if err != nil {
			t.Fatal(err)
		}
		a := PageAdapter{*d, source}
		restored, err := a.Restore(string(raw))
		if err != nil || !bytes.Equal(bridgeEOF(restored), candidate) {
			t.Fatal("original restore drift", err)
		}
		repairs, _, err := a.InspectStructuralWhitespace(raw)
		if err != nil {
			t.Fatal(r.UnitID, err)
		}
		found := false
		for _, repair := range repairs {
			found = found || repair.OpaqueLine && repair.Line == line
		}
		if !found {
			t.Fatal("exact blocked line has no bounded proof", r.UnitID)
		}
		t.Logf("unit=%s first_blocked_line=%d exact_origin=PASS repair_ranges=%d", r.UnitID, line, len(repairs))
	}
	if count != 4 {
		t.Fatal("wrong exact diagnostic scope", count)
	}
}
