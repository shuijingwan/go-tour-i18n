package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mixedSnapshotFixture(t *testing.T, pkg string) (string, *Catalog, []string, string, string) {
	t.Helper()
	root, c, ids, accepted := frozenCardFinalizedFixture(t, pkg, 2)
	checkpointGlossary(t, root, "preferred:\n  apple: 苹果\n", "mixed-current-review")
	a, err := AssessCurrentTranslation(root, c, "zh-CN", accepted)
	if err != nil || a.Compatible != 1 || a.Affected != 2 {
		t.Fatalf("mixed fixture assessment: %+v %v", a, err)
	}
	futureSnapshot(t, root, c, "mixed-s1")
	futureRecord(t, root, c, "mixed-s1", accepted, "A", "", []string{ids[1]})
	futureRecord(t, root, c, "mixed-s1", accepted, "B", "Close this exact apple finding.", []string{ids[0]})
	e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{ids[0]}, AllowReexport: true, PreviousSnapshotID: "mixed-s1"})
	if err != nil {
		t.Fatal(err)
	}
	b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := readGenerationBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	raw := localizedAppleCardFixtureRevision(t, files["formal/inputs/"+filepath.Base(m.ExpectedOutputs[0].BundlePath)])
	futureImport(t, root, c, b, map[string][]byte{ids[0]: raw})
	if result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || result.ValidationPassed != 1 {
		t.Fatalf("mixed revision process: %+v %v", result, err)
	}
	return root, c, ids, accepted, e.BatchID
}

func TestCurrentTranslationMixedSnapshotRevision(t *testing.T) {
	for _, pkg := range []string{"mixed-cards-alpha", "unrelated-mixed-beta"} {
		t.Run(pkg, func(t *testing.T) {
			root, c, ids, _, batch := mixedSnapshotFixture(t, pkg)
			s := futureSnapshot(t, root, c, "mixed-s2")
			if s.CurrentRevision == nil || len(s.CurrentRevision.Delta) != 1 || s.CurrentRevision.Delta[0].UnitID != ids[0] || s.Units[0].SelectedBatchID != batch {
				t.Fatal("missing exact mixed delta")
			}
			p, err := BuildQualityCheckPreflight(root, c, QualityCheckPreflightOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: "mixed-s1"})
			if err != nil || p.CarryForwardCount != 2 || p.QualityCheckRequiredCount != 1 || p.RevisionRequiredCount != 0 || p.Pending[0].UnitID != ids[0] {
				t.Fatalf("mixed preflight: %+v %v", p, err)
			}
			if _, err := BuildQualityCheckPreflight(root, c, QualityCheckPreflightOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: "frozen-accepted"}); err == nil {
				t.Fatal("wrong planning predecessor accepted")
			}
			b, m, err := ExportQualityCheckReviewerBundle(root, c, QualityCheckReviewerBundleOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: "mixed-s1"})
			if err != nil || m.UnitCount != 1 {
				t.Fatalf("mixed bundle: %+v %v", m, err)
			}
			if _, err := VerifyCurrentQualityCheckReviewerBundle(root, c, b); err != nil {
				t.Fatal(err)
			}
			futureRecord(t, root, c, s.SnapshotID, "mixed-s1", "A", "", []string{ids[0]})
			if _, _, err := FinalizeQualityCheck(root, c, QualityCheckFinalizeOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: "mixed-s1"}); err != nil {
				t.Fatal(err)
			}
			a, path, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", s.SnapshotID)
			if err != nil || !a.Ready() || a.Compatible != 3 {
				t.Fatalf("mixed final current authorization: %+v %v", a, err)
			}
			if _, err := CheckCurrentTranslationAuthorization(root, c, path); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func mixedRewriteJSON(t *testing.T, root, path string, f func(map[string]any)) {
	t.Helper()
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	f(v)
	writeSnapshotSource(t, root, path, append(mustJSON(v), '\n'))
}

func TestCurrentTranslationMixedSnapshotFailClosed(t *testing.T) {
	for _, kind := range []string{"baseline-stale", "baseline-tampered", "source", "unauthorized", "finding", "wrong-predecessor", "validation", "unknown", "provenance", "output-provenance", "input-identity", "candidate-hash", "arbitrary-passed"} {
		t.Run(kind, func(t *testing.T) {
			root, c, ids, accepted, batch := mixedSnapshotFixture(t, "negative-mixed-cards")
			s := futureSnapshot(t, root, c, "mixed-s2")
			base := "data/retranslation-runs/zh-CN/" + batch + "/"
			switch kind {
			case "baseline-stale":
				checkpointGlossary(t, root, "# new glossary identity\n", "mixed-next-review")
			case "baseline-tampered":
				b, err := readCompatibilityFile(root, "data/quality-check-snapshots/zh-CN/mixed-s1/manifest.json")
				if err != nil {
					t.Fatal(err)
				}
				var p QualityCheckSnapshotManifest
				if err := json.Unmarshal(b, &p); err != nil {
					t.Fatal(err)
				}
				p.CurrentReattestation.Units[0].AuthoritySHA = strings.Repeat("1", 64)
				p.CurrentReattestation.Identity = authorizationIdentity(*p.CurrentReattestation)
				writeSnapshotSource(t, root, "data/quality-check-snapshots/zh-CN/mixed-s1/manifest.json", append(mustJSON(p), '\n'))
			case "source":
				writeSnapshotSource(t, root, c.Adapted[0].Unit.SourcePath, append(c.Adapted[0].Unit.Source, '\n'))
			case "candidate-hash":
				writeSnapshotSource(t, root, s.Units[2].CandidatePath, []byte("unaccepted drift\n"))
			case "unknown":
				c.Adapted[0].Adapter = struct{ TranslationUnitAdapter }{c.Adapted[0].Adapter}
				c.TranslationPackages[0].Units[0] = c.Adapted[0]
			case "unauthorized":
				mixedRewriteJSON(t, root, base+"manifest.json", func(m map[string]any) { m["units"].([]any)[0].(map[string]any)["revision_feedback_source"] = "" })
			case "wrong-predecessor":
				mixedRewriteJSON(t, root, base+"manifest.json", func(m map[string]any) { m["units"].([]any)[0].(map[string]any)["previous_snapshot_id"] = accepted })
			case "finding":
				mixedRewriteJSON(t, root, "data/quality-check-snapshots/zh-CN/mixed-s1/quality-check-results.json", func(m map[string]any) {
					for _, r := range m["results"].([]any) {
						v := r.(map[string]any)
						if v["unit_id"] == ids[0] {
							delete(v, "finding")
						}
					}
				})
			case "validation":
				mixedRewriteJSON(t, root, s.Units[0].ValidationPath, func(m map[string]any) { m["status"] = "validation_failed" })
			case "provenance":
				mixedRewriteJSON(t, root, base+"generation-imports/initial-manifest.json", func(m map[string]any) { m["provider"] = "unknown" })
			case "input-identity":
				mixedRewriteJSON(t, root, base+"generation-imports/initial-manifest.json", func(m map[string]any) { m["input_identity_sha256"] = strings.Repeat("1", 64) })
			case "output-provenance":
				mixedRewriteJSON(t, root, base+"generation-imports/initial-manifest.json", func(m map[string]any) { m["outputs"].([]any)[0].(map[string]any)["sha256"] = strings.Repeat("1", 64) })
			case "arbitrary-passed":
				e, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{ids[0]}, AllowReexport: true})
				if err != nil {
					t.Fatal(err)
				}
				b, m, err := ExportTranslationUnitGenerationBundle(root, c, GenerationBundleOptions{Locale: "zh-CN", BatchID: e.BatchID})
				if err != nil {
					t.Fatal(err)
				}
				_, files, _ := readGenerationBundle(b)
				futureImport(t, root, c, b, map[string][]byte{ids[0]: files["formal/inputs/"+filepath.Base(m.ExpectedOutputs[0].BundlePath)]})
				if r, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: "zh-CN", BatchID: e.BatchID}); err != nil || r.ValidationPassed != 1 {
					t.Fatalf("arbitrary passed fixture: %+v %v", r, err)
				}
			}

			_, _, snapshotErr := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "rejected-mixed"})
			if snapshotErr == nil {
				t.Fatal("unsafe mixed Snapshot accepted")
			}
			wantSnapshot := map[string]string{
				"baseline-stale":    `quality-check snapshot "mixed-s1" current re-attestation glossary is stale`,
				"baseline-tampered": "Snapshot current re-attestation stale",
				"source":            "Snapshot current re-attestation stale",
				"unauthorized":      "unauthorized latest replacement: " + ids[0],
				"finding":           "require finding for rating B at index 1",
				"wrong-predecessor": "latest candidate replacement lacks exact current baseline revision authorization",
				"validation":        ids[0] + ": validation does not match manifest/result.json",
				"unknown":           "Snapshot current re-attestation stale",
				"provenance":        "provider must be chatgpt or codex",
				"output-provenance": "revision output provenance mismatch",
				"input-identity":    "imported Generation provenance/input identity mismatch",
				"candidate-hash":    "checkpoint historical evidence missing/changed: " + s.Units[2].CandidatePath,
				"arbitrary-passed":  "latest candidate replacement lacks exact current baseline revision authorization",
			}[kind]
			if wantSnapshot == "" || !strings.Contains(snapshotErr.Error(), wantSnapshot) {
				t.Fatalf("%s reached wrong Snapshot failure class: got %v; want %q", kind, snapshotErr, wantSnapshot)
			}
			t.Logf("MIXED_SNAPSHOT_REJECTION kind=%s error=%v", kind, snapshotErr)
			if _, err := os.Stat(filepath.Join(root, "data/quality-check-snapshots/zh-CN/rejected-mixed")); !os.IsNotExist(err) {
				t.Fatal("unsafe mixed Snapshot mutated storage")
			}
			if kind != "arbitrary-passed" {

				_, preflightErr := BuildQualityCheckPreflight(root, c, QualityCheckPreflightOptions{Locale: "zh-CN", SnapshotID: s.SnapshotID, PreviousSnapshotID: "mixed-s1"})
				if preflightErr == nil {
					t.Fatal("unchecked mixed JSON accepted downstream")
				}
				wantPreflight := map[string]string{
					"baseline-stale":    `quality-check snapshot "mixed-s2" current re-attestation glossary is stale`,
					"baseline-tampered": "mixed Snapshot evidence changed: data/quality-check-snapshots/zh-CN/mixed-s1/manifest.json",
					"source":            "Snapshot current re-attestation stale",
					"unknown":           "Snapshot current re-attestation stale",
					"candidate-hash":    "checkpoint historical evidence missing/changed: " + s.Units[2].CandidatePath,
				}[kind]
				if wantPreflight == "" {
					path := map[string]string{
						"unauthorized": base + "manifest.json", "wrong-predecessor": base + "manifest.json",
						"finding":           "data/quality-check-snapshots/zh-CN/mixed-s1/quality-check-results.json",
						"validation":        s.Units[0].ValidationPath,
						"provenance":        base + "generation-imports/initial-manifest.json",
						"output-provenance": base + "generation-imports/initial-manifest.json",
						"input-identity":    base + "generation-imports/initial-manifest.json",
					}[kind]
					if path == "" {
						t.Fatal("missing expected preflight failure class", kind)
					}
					wantPreflight = "mixed Snapshot evidence changed: " + path
				}
				if !strings.Contains(preflightErr.Error(), wantPreflight) {
					t.Fatalf("%s reached wrong preflight failure class: got %v; want %q", kind, preflightErr, wantPreflight)
				}
				t.Logf("MIXED_PREFLIGHT_REJECTION kind=%s error=%v", kind, preflightErr)
			}
		})
	}
}

func TestCurrentTranslationMixedImportedArchiveRecovery(t *testing.T) {
	root, c, _, _, batch := mixedSnapshotFixture(t, "archive-recovery-cards")
	base := "data/retranslation-runs/zh-CN/" + batch + "/"
	b, err := readCompatibilityFile(root, base+"generation-imports/initial-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var receipt GenerationResultBundleManifest
	if err := json.Unmarshal(b, &receipt); err != nil {
		t.Fatal(err)
	}
	path := importedGenerationArchive("zh-CN", receipt.GenerationBundleSHA256)
	original, err := readCompatibilityFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, path)); err != nil {
		t.Fatal(err)
	}
	// Current docs may evolve after invocation. Archive recovery authenticates
	// actual old transport against its receipt and never re-exports inputs.
	writeSnapshotSource(t, root, "docs/SHARED_TRANSLATION_UNIT_LIFECYCLE.md", []byte("current docs changed after original invocation\n"))
	if _, _, err := CreateQualityCheckCandidateSnapshot(root, c, QualityCheckSnapshotOptions{Locale: "zh-CN", SnapshotID: "missing-archive"}); err == nil {
		t.Fatal("missing original invocation accepted")
	}
	if got, err := ArchiveImportedGenerationBundle(root, original); err != nil || got != path {
		t.Fatalf("original archive recovery: %s %v", got, err)
	}
	after, err := readCompatibilityFile(root, base+"generation-imports/initial-manifest.json")
	if err != nil || !bytes.Equal(b, after) {
		t.Fatal("recovery changed existing import receipt")
	}
	futureSnapshot(t, root, c, "recovered-mixed")
}
