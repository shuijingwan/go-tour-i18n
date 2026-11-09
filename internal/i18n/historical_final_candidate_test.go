package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoricalFinalCandidateVerifier(t *testing.T) {
	for _, mode := range []string{"passed", "raw-tamper", "candidate-tamper", "validation-failed", "active-validator-failure"} {
		t.Run(mode, func(t *testing.T) {
			root, c, batch := processedPromotionFixture(t, 1)
			dir := filepath.Join(root, "data/retranslation-runs/zh-CN", batch)
			m, err := readRetranslationProcessManifest(dir, "zh-CN", batch)
			if err != nil {
				t.Fatal(err)
			}
			result, err := readPromotionResult(dir, "zh-CN", batch, len(m.Units))
			if err != nil {
				t.Fatal(err)
			}
			choice := selectedRetranslationUnit{batchID: batch, batchDir: dir, manifest: m.Units[0], result: result.Units[0]}
			g, err := LoadGlossary(root, "zh-CN")
			if err != nil {
				t.Fatal(err)
			}
			candidate := filepath.Join(dir, choice.result.CandidatePath)
			validation := filepath.Join(dir, choice.result.ValidationPath)
			raw := filepath.Join(dir, "raw-responses", filepath.Base(choice.manifest.InputPath))
			switch mode {
			case "raw-tamper", "candidate-tamper":
				path := raw
				if mode == "candidate-tamper" {
					path = candidate
				}
				b, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(b, []byte("changed\n")...), 0644); err != nil {
					t.Fatal(err)
				}
			case "validation-failed":
				b, err := os.ReadFile(validation)
				if err != nil {
					t.Fatal(err)
				}
				var v RetranslationValidation
				if err := json.Unmarshal(b, &v); err != nil {
					t.Fatal(err)
				}
				v.Status = "validation_failed"
				if err := writeTranslationJSON(validation, v); err != nil {
					t.Fatal(err)
				}
			case "active-validator-failure":
				for _, path := range []string{raw, candidate} {
					b, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					b = []byte(appendBeforeRetranslationArtifactEOF(string(b), " `unexpected`"))
					if err := os.WriteFile(path, b, 0644); err != nil {
						t.Fatal(err)
					}
				}
				bindPromotionFixtureCandidateSHA(t, dir)
			}
			verified, err := verifyHistoricalFinalCandidate(root, c, g, m, choice, nil, protectedTranslation{})
			if mode != "passed" {
				if err == nil {
					t.Fatal("invalid historical final candidate accepted", mode)
				}
				if mode == "active-validator-failure" && !strings.Contains(err.Error(), "protected structure validation failed") {
					t.Fatal("active validator was not exercised", err)
				}
				return
			}
			if err != nil || verified.attempt != 1 || verified.unit.ID != choice.manifest.UnitID {
				t.Fatal(verified, err)
			}
			if want, err := os.ReadFile(candidate); err != nil || !bytes.Equal(want, verified.candidate.data) {
				t.Fatal("candidate observation differs", err)
			}
			// Recovery retains its own import/history requirements after this
			// common verifier: a process-only fixture has no Generation receipt.
			if _, err := recoveryHistoricalFinalUnit(root, c, g, m, choice, GenerationBundleManifest{TaskKind: "translation-unit-revision"}, map[string][]byte{}, nil, protectedTranslation{}, "chatgpt", FormalChatGPTGenerationModel); err == nil || !os.IsNotExist(err) {
				t.Fatal("recovery import/provenance gate changed", err)
			}
		})
	}
}

func TestHistoricalFinalCandidateRecoveryOutputProvenance(t *testing.T) {
	root, c, batch := generationBundleFixture(t, "zh-CN", RetranslationGeneratorChatGPT)
	bundle, initial, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	for _, expected := range initial.ExpectedOutputs {
		input := files["formal/inputs/"+filepath.Base(expected.BundlePath)]
		raw := strings.ReplaceAll(string(input), "* Page", "* 页面")
		raw = strings.ReplaceAll(raw, "Use ", "在此页面使用 ")
		raw = strings.ReplaceAll(raw, " on this page.", "。")
		if err := os.WriteFile(filepath.Join(output, filepath.Base(expected.BundlePath)), []byte(raw), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ImportGenerationOutputDirectory(root, c, bundle, "chatgpt", FormalChatGPTGenerationModel, output); err != nil {
		t.Fatal(err)
	}
	result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil || result.ValidationPassed != 2 {
		t.Fatal(result, err)
	}
	dir := filepath.Join(root, "data/retranslation-runs/zh-CN", batch)
	m, err := readRetranslationProcessManifest(dir, "zh-CN", batch)
	if err != nil {
		t.Fatal(err)
	}
	choice := selectedRetranslationUnit{batchID: batch, batchDir: dir, manifest: m.Units[0], result: result.Units[0]}
	g, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	f, err := recoveryHistoricalFinalUnit(root, c, g, m, choice, initial, files, nil, protectedTranslation{}, "chatgpt", FormalChatGPTGenerationModel)
	if err != nil || f.Attempt != 1 || len(f.History) != 2 {
		t.Fatal(f, err)
	}
	t.Run("snapshot-frozen-approved-predecessor", func(t *testing.T) {
		checkSnapshotFrozenApprovedPredecessor(t, root, c, g, m, initial, files)
	})
	path := filepath.Join(dir, "generation-imports/initial-manifest.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var receipt GenerationResultBundleManifest
	if err := json.Unmarshal(b, &receipt); err != nil {
		t.Fatal(err)
	}
	receipt.Outputs[0].SHA256 = sum([]byte("not the imported raw"))
	if err := writeTranslationJSON(path, receipt); err != nil {
		t.Fatal(err)
	}
	if _, err := recoveryHistoricalFinalUnit(root, c, g, m, choice, initial, files, nil, protectedTranslation{}, "chatgpt", FormalChatGPTGenerationModel); err == nil || !strings.Contains(err.Error(), "historical output provenance mismatch") {
		t.Fatal("recovery output provenance gate changed", err)
	}
}

func TestSnapshotApprovedSemanticEquivalenceBoundary(t *testing.T) {
	type codec struct {
		Schema string `json:"schema"`
	}
	type child struct {
		Role string `json:"role"`
	}
	c := &Catalog{}
	c.DerivedTransportCodecReader = func(_ TranslationUnitAdapter, b, inventory []byte) (any, any, TranslationUnitAdapter, error) {
		var p codec
		var children []child
		if err := json.Unmarshal(b, &p); err != nil {
			return nil, nil, nil, err
		}
		if err := json.Unmarshal(inventory, &children); err != nil {
			return nil, nil, nil, err
		}
		return p, children, nil, nil
	}
	u := &TranslationUnit{ID: "fixture"}
	old := ApprovedIntervalTransport{Schema: ApprovedIntervalTransportSchema, Mode: ApprovedIntervalMode, CurrentAuthority: sum([]byte("old validator")), Dependencies: []GlossaryArchiveReference{{Path: approvedIntervalImplementationPaths[0], SHA256: sum([]byte("old implementation"))}, {Path: "locales/fr-FR/glossary.yaml", SHA256: sum([]byte("same language"))}}, Projection: ApprovedIntervalProjection{Codec: json.RawMessage(`{"schema":"fixture-codec/v1"}`), Children: json.RawMessage(`[{"role":"source-child"}]`)}}
	current := old
	current.Dependencies = append([]GlossaryArchiveReference(nil), old.Dependencies...)
	current.CurrentAuthority = sum([]byte("current validator"))
	current.Dependencies[0].SHA256 = sum([]byte("current implementation"))
	if err := snapshotApprovedProofEquivalent(c, u, &old, &current); err != nil {
		t.Fatal("code-owned drift must remain admissible", err)
	}
	current.Projection.ParentSHA256 = sum([]byte("changed semantic parent"))
	if err := snapshotApprovedProofEquivalent(c, u, &old, &current); err == nil {
		t.Fatal("semantic body drift accepted")
	}
	current.Projection = old.Projection
	current.Dependencies[1].SHA256 = sum([]byte("changed language evidence"))
	if err := snapshotApprovedProofEquivalent(c, u, &old, &current); err == nil {
		t.Fatal("non-implementation dependency drift accepted")
	}
}
