package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func revisionGenerationFixture(t *testing.T) (string, *Catalog, GenerationBundleOptions) {
	t.Helper()
	root, catalog, _ := makeRetranslationReviewBatchFixture(t, 2, "qc-001")
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "B", []string{"lesson/1"})
	recordQualityCheckRatings(t, root, catalog, "qc-001", "", "C", []string{"lesson/2"})
	exported, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", UnitIDs: []string{"lesson/1", "lesson/2"}, AllowReexport: true, PreviousSnapshotID: "qc-001"})
	if err != nil {
		t.Fatal(err)
	}
	return root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: exported.BatchID}
}

func assertRevisionCandidates(t *testing.T, root string, catalog *Catalog, options GenerationBundleOptions, bundle []byte) {
	t.Helper()
	manifest, files, err := readGenerationBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	batch := readRetranslationManifest(t, root, options.BatchID)
	snapshot, err := readQualityCheckSnapshot(root, options.Locale, "qc-001", false, catalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range batch.Units {
		if record.RevisionFeedbackSource != CurrentValidationRevisionFeedback && files[commentTransportPath(record)] != nil {
			t.Fatal("ordinary revision acquired comment transport exception")
		}
		var unit QualityCheckSnapshotUnit
		for _, u := range snapshot.Units {
			if u.UnitID == record.UnitID {
				unit = u
			}
		}
		path := "formal/candidates/" + filepath.Base(record.InputPath)
		want, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(unit.CandidatePath)))
		if err != nil || !bytes.Equal(files[path], want) || sum(files[path]) != unit.CandidateSHA256 {
			t.Fatalf("%s: exact predecessor bytes/hash missing: %v", record.UnitID, err)
		}
		found := false
		for _, f := range manifest.Files {
			if f.BundlePath == path {
				found = f.RepositoryPath == unit.CandidatePath && f.SHA256 == unit.CandidateSHA256
			}
		}
		if !found {
			t.Fatal("candidate inventory is not bound to Snapshot")
		}
		evidence, err := readSnapshotUnitRepositoryEvidence(root, catalog, options.Locale, unit)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(evidence.batchDir, filepath.FromSlash(evidence.validation.RawResponsePath)))
		protectedPath := "formal/protected-candidates/" + filepath.Base(record.InputPath)
		historicalPath := protectedPath
		if record.RevisionFeedbackSource == "surface_review" {
			historicalPath = "formal/historical-protected-candidates/" + filepath.Base(record.InputPath)
		}
		if err != nil || !bytes.Equal(files[historicalPath], raw) {
			t.Fatalf("exact validated raw response missing: %v", err)
		}
		glossary, err := LoadGlossary(root, options.Locale)
		if err != nil {
			t.Fatal(err)
		}
		protected, err := prepareTranslationUnitInput(evidence.unit, glossary)
		if err != nil {
			t.Fatal(err)
		}
		restored, failures := protected.restore(string(files[historicalPath]))
		restoredBytes := []byte(restored)
		predecessorBatch, err := readRetranslationProcessManifest(evidence.batchDir, options.Locale, unit.SelectedBatchID)
		if err != nil {
			t.Fatal(err)
		}
		if predecessorBatch.ArtifactEOF == retranslationArtifactEOFSingleLF {
			restoredBytes = canonicalizeRetranslationArtifactEOF(restoredBytes)
		}
		if len(failures) != 0 || !bytes.Equal(restoredBytes, want) || sum(restoredBytes) != unit.CandidateSHA256 {
			t.Fatalf("protected baseline does not restore to reviewed bytes: %v", failures)
		}
		found = false
		for _, f := range manifest.Files {
			if f.BundlePath == historicalPath {
				found = f.SHA256 == sum(raw) && f.RepositoryPath == filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, unit.SelectedBatchID, evidence.validation.RawResponsePath))
			}
		}
		if !found {
			t.Fatal("protected baseline inventory identity mismatch")
		}
		for _, out := range manifest.ExpectedOutputs {
			if out.BundlePath == path {
				t.Fatal("predecessor candidate became expected output")
			}
		}
		if record.RevisionFeedbackSource == "surface_review" {
			var projection SurfaceTransportProjection
			if err := decodeStrictBundleJSON(files["formal/surface-transport/"+record.UnitID+".json"], &projection); err != nil {
				t.Fatal(err)
			}
			current, err := catalog.Unit(record.UnitID)
			if err != nil {
				t.Fatal(err)
			}
			factory, ok := current.Adapter.(SurfaceTransportAdapter)
			if !ok {
				t.Fatal("Surface fixture lacks source-owned projection")
			}
			decoder, err := factory.ReadSurfaceTransportProjection(projection.Codec)
			if err != nil {
				t.Fatal(err)
			}
			restored, err := decoder.Restore(string(files[protectedPath]))
			if err != nil || !bytes.Equal(files[protectedPath], []byte(projection.Template)) || !bytes.Equal(canonicalizeRetranslationArtifactEOF(restored), want) {
				t.Fatal("Surface executable baseline differs from exact capability/predecessor", err)
			}
			bound := false
			for _, ref := range manifest.Files {
				if ref.BundlePath == protectedPath {
					bound = ref.RepositoryPath == "" && ref.SHA256 == sum([]byte(projection.Template))
				}
			}
			if !bound {
				t.Fatal("Surface executable baseline inventory mismatch")
			}
		}

	}
}

func TestRevisionGenerationBundleFrozenCandidatesAndReexport(t *testing.T) {
	root, catalog, options := revisionGenerationFixture(t)
	bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	assertRevisionCandidates(t, root, catalog, options, bundle)
	// Simulate the old exporter: no candidate or predecessor inventory entries.
	_, files, _ := readGenerationBundle(bundle)
	oldManifest := manifest
	oldManifest.Files = nil
	var oldEntries []TransportBundleEntry
	var identity strings.Builder
	for _, f := range manifest.Files {
		if strings.HasPrefix(f.BundlePath, "formal/candidates/") || strings.HasPrefix(f.BundlePath, "formal/protected-candidates/") || strings.HasPrefix(f.BundlePath, "formal/predecessors/") {
			continue
		}
		oldManifest.Files = append(oldManifest.Files, f)
		oldEntries = append(oldEntries, TransportBundleEntry{Path: f.BundlePath, Data: files[f.BundlePath]})
		identity.WriteString(f.BundlePath + "\x00" + f.SHA256 + "\x00")
	}
	oldManifest.InputIdentitySHA256 = sum([]byte(identity.String()))
	if oldManifest.InputIdentitySHA256 == manifest.InputIdentitySHA256 {
		t.Fatal("candidate contract did not change aggregate identity")
	}
	data, _ := json.MarshalIndent(oldManifest, "", "  ")
	old, err := WriteDeterministicTransportBundle(append(data, '\n'), oldEntries)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := currentGenerationBundle(root, catalog, old); err == nil {
		t.Fatal("candidate-incomplete old Bundle remained current")
	}
	// Authority evolves without changing the batch authorization or installing outputs.
	path := filepath.Join(root, "docs", "TRANSLATION_TASK_SPEC.md")
	authority, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(authority, []byte("\nUpdated revision input contract.\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := currentGenerationBundle(root, catalog, bundle); err == nil {
		t.Fatal("old authority remained current")
	}
	updated, updatedManifest, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
	if err != nil || updatedManifest.BatchID != options.BatchID || updatedManifest.InputIdentitySHA256 == manifest.InputIdentitySHA256 {
		t.Fatalf("same-batch recovery failed: %v", err)
	}
	if _, _, err := currentGenerationBundle(root, catalog, updated); err != nil {
		t.Fatal(err)
	}
	assertRevisionCandidates(t, root, catalog, options, updated)
	outputs := t.TempDir()
	writeGenerationFixtureOutputs(t, root, options.BatchID, outputs, updatedManifest)
	if _, err := ImportGenerationOutputDirectory(root, catalog, updated, "codex", PreviousCodexGenerationModel, outputs); err != nil {
		t.Fatal(err)
	}
	if result, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: options.Locale, BatchID: options.BatchID}); err != nil || result.ValidationPassed != 2 {
		t.Fatalf("revision process=%+v error=%v", result, err)
	}
}

func TestInitialGenerationBundleDoesNotRequireCandidates(t *testing.T) {
	root, catalog, batchID := generationBundleFixture(t, "zh-CN", RetranslationGeneratorCodex)
	bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batchID})
	if err != nil || manifest.TaskKind != "translation-unit-initial" {
		t.Fatalf("initial export: %v", err)
	}
	for _, f := range manifest.Files {
		if strings.HasPrefix(f.BundlePath, "formal/candidates/") || strings.HasPrefix(f.BundlePath, "formal/protected-candidates/") || strings.HasPrefix(f.BundlePath, "formal/predecessors/") {
			t.Fatal("initial unexpectedly requires predecessor inputs")
		}
	}
	if _, _, err := currentGenerationBundle(root, catalog, bundle); err != nil {
		t.Fatal(err)
	}
}

func TestRevisionGenerationBundleCandidateFailures(t *testing.T) {
	for _, mutation := range []string{"tamper", "delete", "finding", "source", "zip", "raw-delete", "raw-tamper", "raw-other-batch", "raw-attempt", "snapshot-delete", "validation-delete"} {
		t.Run(mutation, func(t *testing.T) {
			root, catalog, options := revisionGenerationFixture(t)
			bundle, _, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
			if err != nil {
				t.Fatal(err)
			}
			snapshot, _ := readQualityCheckSnapshot(root, options.Locale, "qc-001", false)
			path := filepath.Join(root, filepath.FromSlash(snapshot.Units[0].CandidatePath))
			switch mutation {
			case "raw-delete", "raw-tamper", "raw-other-batch", "raw-attempt":
				evidence, e := readSnapshotUnitRepositoryEvidence(root, catalog, options.Locale, snapshot.Units[0])
				if e != nil {
					t.Fatal(e)
				}
				rawPath := filepath.Join(evidence.batchDir, evidence.validation.RawResponsePath)
				switch mutation {
				case "raw-delete":
					err = os.Remove(rawPath)
				case "raw-tamper":
					raw, _ := os.ReadFile(rawPath)
					err = os.WriteFile(rawPath, append([]byte("Altered prose\n"), raw...), 0644)
				case "raw-other-batch":
					addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-099", []string{"lesson/1"})
					batch := readRetranslationManifest(t, root, "chatgpt-zh-CN-099")
					other, _ := os.ReadFile(filepath.Join(root, "data", "retranslation-runs", options.Locale, "chatgpt-zh-CN-099", "raw-responses", filepath.Base(batch.Units[0].InputPath)))
					err = os.WriteFile(rawPath, other, 0644)
				case "raw-attempt":
					validation := *evidence.validation
					validation.RawResponsePath = "retries/" + strings.TrimSuffix(filepath.Base(evidence.record.InputPath), ".article") + "/attempt-002.article"
					data, _ := json.MarshalIndent(validation, "", "  ")
					err = os.WriteFile(filepath.Join(root, snapshot.Units[0].ValidationPath), append(data, '\n'), 0644)
				}
			case "snapshot-delete":
				err = os.Remove(filepath.Join(root, "data", "quality-check-snapshots", options.Locale, "qc-001", "manifest.json"))
			case "validation-delete":
				err = os.Remove(filepath.Join(root, snapshot.Units[0].ValidationPath))
			case "tamper":
				err = os.WriteFile(path, []byte("Different candidate\n"), 0644)
			case "delete":
				err = os.Remove(path)
			case "finding", "source":
				batch := readRetranslationManifest(t, root, options.BatchID)
				if mutation == "finding" {
					batch.Units[0].PreviousFinding = "Unreviewed finding"
				} else {
					batch.Units[0].SourceSHA256 = sum([]byte("other source"))
				}
				data, _ := json.MarshalIndent(batch, "", "  ")
				err = os.WriteFile(filepath.Join(root, "data", "retranslation-runs", options.Locale, options.BatchID, "manifest.json"), append(data, '\n'), 0644)
			case "zip":
				_, files, _ := readGenerationBundle(bundle)
				var entries []TransportBundleEntry
				for name, data := range files {
					if name == "manifest.json" {
						continue
					}
					if strings.HasPrefix(name, "formal/candidates/") {
						data = []byte("Tampered ZIP candidate\n")
					}
					entries = append(entries, TransportBundleEntry{Path: name, Data: data})
				}
				bundle, err = WriteDeterministicTransportBundle(files["manifest.json"], entries)
			}
			if err != nil {
				t.Fatal(err)
			}
			if mutation != "zip" {
				if _, _, err := ExportTranslationUnitGenerationBundle(root, catalog, options); err == nil {
					t.Fatal("invalid predecessor/authorization exported")
				}
			}
			if _, _, err := currentGenerationBundle(root, catalog, bundle); err == nil {
				t.Fatal("tampered/deleted/stale input remained current")
			}
		})
	}
}

// Both a representation failure and a semantic machine failure must keep the
// original revision baseline on retry, while outputs stay exact single-Unit.
func exerciseRevisionRetry(t *testing.T, root string, catalog *Catalog, options GenerationBundleOptions, mode string, repair ...func([]byte) []byte) {
	t.Helper()
	bundle, manifest, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	_, files, _ := readGenerationBundle(bundle)
	dir := t.TempDir()
	for _, out := range manifest.ExpectedOutputs {
		name := filepath.Base(out.BundlePath)
		data := files["formal/candidates/"+name]
		if mode == "validation_failed" {
			data = []byte(appendBeforeRetranslationArtifactEOF(string(files["formal/protected-candidates/"+name]), "额外 `bad` 代码"))
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "codex", PreviousCodexGenerationModel, dir); err != nil {
		t.Fatal(err)
	}
	result, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: options.Locale, BatchID: options.BatchID})
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range result.Units {
		if u.Status != mode {
			t.Fatalf("expected %s from invalid output, got %+v", mode, u)
		}
	}
	for _, out := range manifest.ExpectedOutputs {
		options.UnitID = out.UnitID
		retry, m, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
		if err != nil || m.TaskKind != "translation-unit-retry" || m.Attempt != 2 || len(m.ExpectedOutputs) != 1 {
			t.Fatalf("revision retry: %+v %v", m, err)
		}
		assertRevisionCandidates(t, root, catalog, options, retry)
		_, inputs, _ := readGenerationBundle(retry)
		for _, path := range []string{"formal/retry/current-validation.json", "formal/retry/result.json", "formal/manifest.json", "formal/predecessors/qc-001/manifest.json"} {
			if len(inputs[path]) == 0 {
				t.Fatalf("retry missing %s", path)
			}
		}
		if !bytes.Contains(inputs["generation-context.json"], []byte("Semantic task: translation-unit-revision")) || !bytes.Contains(inputs["generation-context.json"], []byte("machine retry attempt 2")) {
			t.Fatal("retry lost revision instruction")
		}
		dir := t.TempDir()
		name := filepath.Base(m.ExpectedOutputs[0].BundlePath)
		data := []byte(appendBeforeRetranslationArtifactEOF(string(inputs["formal/protected-candidates/"+name]), "局部补充说明。"))
		if len(repair) > 0 {
			data = repair[0](inputs["formal/protected-candidates/"+name])
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := ImportGenerationOutputDirectory(root, catalog, retry, "codex", PreviousCodexGenerationModel, dir); err != nil {
			t.Fatal(err)
		}
		result, err = ProcessRetranslationRetry(root, catalog, RetranslationRetryOptions{Locale: options.Locale, BatchID: options.BatchID, UnitID: options.UnitID})
		if err != nil {
			t.Fatal(err)
		}
	}
	if result.ValidationPassed != len(manifest.ExpectedOutputs) {
		t.Fatalf("protected revision recovery failed: %+v", result)
	}
}

func TestRevisionGenerationBundleProtectedOutputAndRetry(t *testing.T) {
	for _, status := range []string{"restore_failed", "validation_failed"} {
		t.Run(status, func(t *testing.T) {
			root, catalog, options := revisionGenerationFixture(t)
			exerciseRevisionRetry(t, root, catalog, options, status)
		})
	}
}

func TestGlossaryStaleRetryDoesNotInheritRevisionSemantics(t *testing.T) {
	root := t.TempDir()
	catalog := glossaryStaleTestCatalog()
	copyBundleAuthority(t, root, translationUnitGenerationAuthorityPaths)
	writeGlossaryStaleTestGlossary(t, root, "old-review", true)
	initial, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT})
	if err != nil {
		t.Fatal(err)
	}
	processGlossaryStaleTestBatch(t, root, catalog, initial.BatchID)
	writeGlossaryStaleTestGlossary(t, root, "new-review", false)
	recovery, err := ExportRetranslationBatch(root, catalog, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT, UnitIDs: []string{"lesson/1"}, GlossaryStale: true, AllowReexport: true})
	if err != nil {
		t.Fatal(err)
	}
	options := GenerationBundleOptions{Locale: "zh-CN", BatchID: recovery.BatchID}
	bundle, m, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
	if err != nil || m.TaskKind != "translation-unit-glossary-stale-recovery" {
		t.Fatalf("glossary recovery export: %v", err)
	}
	dir := t.TempDir()
	writeGenerationFixtureOutputs(t, root, recovery.BatchID, dir, m)
	path := filepath.Join(dir, filepath.Base(m.ExpectedOutputs[0].BundlePath))
	data, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(appendBeforeRetranslationArtifactEOF(string(data), "额外 `bad` 代码")), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "chatgpt", FormalChatGPTGenerationModel, dir); err != nil {
		t.Fatal(err)
	}
	result, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: options.Locale, BatchID: options.BatchID})
	if err != nil || result.ValidationFailed != 1 {
		t.Fatalf("glossary recovery failure fixture: %+v %v", result, err)
	}
	options.UnitID = "lesson/1"
	retry, m, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
	if err != nil || m.TaskKind != "translation-unit-retry" || m.Attempt != 2 {
		t.Fatalf("glossary retry: %+v %v", m, err)
	}
	for _, f := range m.Files {
		if strings.HasPrefix(f.BundlePath, "formal/candidates/") || strings.HasPrefix(f.BundlePath, "formal/protected-candidates/") || strings.HasPrefix(f.BundlePath, "formal/predecessors/") {
			t.Fatal("glossary retry acquired QC revision semantics")
		}
	}
	if _, _, err := currentGenerationBundle(root, catalog, retry); err != nil {
		t.Fatal(err)
	}
}

func TestRevisionGenerationBundleFrozenInputTransportTamper(t *testing.T) {
	for _, prefix := range []string{"formal/candidates/", "formal/protected-candidates/", "formal/predecessors/qc-001/manifest.json", "formal/predecessors/qc-001/validation/", "formal/manifest.json"} {
		for _, remove := range []bool{false, true} {
			t.Run(prefix+fmtInt(map[bool]int{false: 0, true: 1}[remove]), func(t *testing.T) {
				root, catalog, options := revisionGenerationFixture(t)
				bundle, _, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
				if err != nil {
					t.Fatal(err)
				}
				_, files, _ := readGenerationBundle(bundle)
				var entries []TransportBundleEntry
				for path, data := range files {
					if path == "manifest.json" {
						continue
					}
					if strings.HasPrefix(path, prefix) {
						if remove {
							continue
						}
						data = append(data, []byte("changed\n")...)
					}
					entries = append(entries, TransportBundleEntry{Path: path, Data: data})
				}
				changed, err := WriteDeterministicTransportBundle(files["manifest.json"], entries)
				if err != nil {
					t.Fatal(err)
				}
				if _, _, err := currentGenerationBundle(root, catalog, changed); err == nil {
					t.Fatal("changed frozen input remained current")
				}
			})
		}
	}
}

func TestRevisionGenerationBundleDoesNotSelectLatestCandidate(t *testing.T) {
	root, catalog, options := revisionGenerationFixture(t)
	// A separate later processed batch exists; the reviewed Snapshot stays immutable.
	addProcessedPromotionBatch(t, root, catalog, "chatgpt-zh-CN-099", []string{"lesson/1"})
	snapshot, err := readQualityCheckSnapshot(root, options.Locale, "qc-001", false)
	if err != nil {
		t.Fatal(err)
	}
	predecessor, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(snapshot.Units[0].CandidatePath)))
	if err != nil {
		t.Fatal(err)
	}
	otherDir := filepath.Join(root, "data", "retranslation-runs", options.Locale, "chatgpt-zh-CN-099")
	result, err := readPromotionResult(otherDir, options.Locale, "chatgpt-zh-CN-099", 1)
	if err != nil {
		t.Fatal(err)
	}
	other, err := os.ReadFile(filepath.Join(otherDir, filepath.FromSlash(result.Units[0].CandidatePath)))
	if err != nil || bytes.Equal(other, predecessor) {
		t.Fatalf("fixture must contain a different newer candidate: %v", err)
	}
	bundle, _, err := ExportTranslationUnitGenerationBundle(root, catalog, options)
	if err != nil {
		t.Fatal(err)
	}
	assertRevisionCandidates(t, root, catalog, options, bundle)
}
