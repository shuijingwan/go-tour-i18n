package main

import (
	"archive/zip"
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

func TestApprovedIntervalRetryRun(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	f := approvedIntervalRunFixture(t)
	t.Chdir(f.f.root)
	startup, err := i18n.ReadCatalog(f.f.root)
	if err != nil || len(startup.TranslationScope) != 0 {
		t.Fatal("startup must remain default Tour", err)
	}
	readBundle := func(path string) ([]byte, i18n.GenerationBundleManifest, map[string][]byte) {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil || len(z.File) == 0 {
			t.Fatal(err)
		}
		for _, member := range z.File {
			if member.Method != zip.Deflate {
				t.Fatal("expected canonical Deflate", member.Name)
			}
		}
		files, err := i18n.ReadTransportBundle(data, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
		if err != nil {
			t.Fatal(err)
		}
		var m i18n.GenerationBundleManifest
		if err := json.Unmarshal(files["manifest.json"], &m); err != nil {
			t.Fatal(err)
		}
		return data, m, files
	}
	export := []string{"generation-bundle", "export", "--content-package", f.f.pkg, "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--approved-interval-selection", "engineering/cli-selection.json", "--output", f.output}
	if _, err := authorizationDiagnosisOutput(t, func() error { return run(export) }); err != nil {
		t.Fatal(err)
	}
	initial, im, initialFiles := readBundle(f.output)
	if im.Attempt != 1 || im.TaskKind != "translation-unit-revision" || len(im.ExpectedOutputs) != 1 {
		t.Fatal("wrong approved revision fixture")
	}
	unit := im.ExpectedOutputs[0]
	name := filepath.Base(unit.BundlePath)
	flat := strings.TrimSuffix(name, filepath.Ext(name))
	var proof i18n.ApprovedIntervalTransport
	proofPath := "formal/approved-interval-transport/" + unit.UnitID + ".json"
	if err := json.Unmarshal(initialFiles[proofPath], &proof); err != nil || proof.Mode != i18n.ApprovedIntervalMode || proof.Record.RevisionFeedbackSource != i18n.CurrentValidationRevisionFeedback {
		t.Fatal("missing actual approved current-validation graph", err)
	}
	baseline := initialFiles["formal/protected-candidates/"+name]
	if len(baseline) == 0 || !bytes.Contains(baseline, []byte(proof.Projection.ParentToken)) {
		t.Fatal("missing opaque baseline")
	}
	outputs := t.TempDir()
	courseContextWrite(t, outputs, name, baseline)
	importArgs := []string{"generation-bundle", "import", "--bundle", f.output, "--input-dir", outputs, "--provider", "chatgpt", "--model", "gpt-5.6-sol-high"}
	if out, err := authorizationDiagnosisOutput(t, func() error { return run(importArgs) }); err != nil || !strings.Contains(string(out), "Generation result imported:") {
		t.Fatal("attempt 1 import through run", err, string(out))
	}
	batch := filepath.Join("data/retranslation-runs", f.f.locale, f.batch.BatchID)
	initialReceiptPath := filepath.Join(batch, "generation-imports/initial-manifest.json")
	initialReceipt := courseContextRead(t, f.f.root, initialReceiptPath)
	initialArchive := "data/generation-archives/" + f.f.locale + "/" + courseContextSHA(initial) + ".zip"
	initialRaw := courseContextRead(t, f.f.root, filepath.Join(batch, unit.InstallPath))
	process := []string{"retranslation", "process", "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--json"}
	out, processErr := authorizationDiagnosisOutput(t, func() error { return run(process) })
	var first i18n.RetranslationProcessResult
	if err := json.Unmarshal(out, &first); err != nil || processErr == nil || first.RestorePassed != 1 || first.ValidationFailed != 1 || first.RestoreFailed != 0 || len(first.Units) != 1 || first.Units[0].Status != "validation_failed" {
		t.Fatal("attempt 1 must have exactly one genuine validation failure", first, processErr, err, string(out))
	}
	validationPath := filepath.Join(batch, first.Units[0].ValidationPath)
	firstValidation := courseContextRead(t, f.f.root, validationPath)
	var v1 i18n.RetranslationValidation
	if err := json.Unmarshal(firstValidation, &v1); err != nil || v1.Attempt != 1 || v1.Status != "validation_failed" {
		t.Fatal(err)
	}
	retryBundlePath := filepath.Join(t.TempDir(), "attempt-002.zip")
	exportRetry := []string{"generation-bundle", "export", "--content-package", f.f.pkg, "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--unit-id", unit.UnitID, "--output", retryBundlePath}
	if _, err := authorizationDiagnosisOutput(t, func() error { return run(exportRetry) }); err != nil {
		t.Fatal("retry export through run", err)
	}
	retryBundle, rm, retryFiles := readBundle(retryBundlePath)
	if rm.Attempt != 2 || rm.TaskKind != "translation-unit-retry" || len(rm.ExpectedOutputs) != 1 || rm.ExpectedOutputs[0].UnitID != unit.UnitID || !bytes.Equal(retryFiles["formal/approved-interval-selection.json"], initialFiles["formal/approved-interval-selection.json"]) || !bytes.Equal(retryFiles[proofPath], initialFiles[proofPath]) {
		t.Fatal("retry did not preserve exact frozen selection/proof")
	}
	repaired := bytes.ReplaceAll(baseline, []byte("说明"), []byte("说明 "))
	if bytes.Equal(repaired, baseline) {
		t.Fatal("fixture must repair editable Han/code adjacency")
	}
	courseContextWrite(t, outputs, name, repaired)
	importRetry := []string{"generation-bundle", "import", "--bundle", retryBundlePath, "--input-dir", outputs, "--provider", "chatgpt", "--model", "gpt-5.6-sol-high"}
	if out, err := authorizationDiagnosisOutput(t, func() error { return run(importRetry) }); err != nil || !strings.Contains(string(out), "Generation result imported:") {
		t.Fatal("attempt 2 import through run", err, string(out))
	}
	retryRawPath := filepath.Join(batch, rm.ExpectedOutputs[0].InstallPath)
	if filepath.ToSlash(rm.ExpectedOutputs[0].InstallPath) != "retries/"+flat+"/attempt-002.article" || !bytes.Equal(courseContextRead(t, f.f.root, retryRawPath), repaired) {
		t.Fatal("wrong exact attempt-002 installation")
	}
	retryReceiptPath := filepath.Join(batch, "generation-imports", flat+"-attempt-002-manifest.json")
	retryReceiptBytes := courseContextRead(t, f.f.root, retryReceiptPath)
	var receipt i18n.GenerationResultBundleManifest
	if err := json.Unmarshal(retryReceiptBytes, &receipt); err != nil || receipt.Attempt != 2 || receipt.Provider != "chatgpt" || receipt.Model != "gpt-5.6-sol-high" || receipt.GenerationBundleSHA256 != courseContextSHA(retryBundle) || receipt.InputIdentitySHA256 != rm.InputIdentitySHA256 || receipt.TaskKind != rm.TaskKind || receipt.Locale != rm.Locale || receipt.BatchID != rm.BatchID || len(receipt.Outputs) != 1 || receipt.Outputs[0] != i18n.NewTransportBundleFile(unit.BundlePath, "", repaired) || receipt.GenerationManifest != i18n.NewTransportBundleFile("generation-manifest.json", "", retryFiles["manifest.json"]) {
		t.Fatal("incorrect actual attempt 2 provenance", receipt, err)
	}
	retryArchive := "data/generation-archives/" + f.f.locale + "/" + courseContextSHA(retryBundle) + ".zip"
	if !bytes.Equal(courseContextRead(t, f.f.root, retryArchive), retryBundle) {
		t.Fatal("retry archive repacked original model input")
	}
	retry := []string{"retranslation", "retry", "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--unit-id", unit.UnitID, "--json"}
	imported := authorizationDiagnosisInventory(t, f.f.root)
	for _, tc := range []struct {
		name   string
		change func(*testing.T)
		want   string
	}{
		{"missing-retry-raw", func(t *testing.T) {
			if err := os.Remove(filepath.Join(f.f.root, retryRawPath)); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { courseContextWrite(t, f.f.root, retryRawPath, repaired) })
		}, "read next retry raw response"},
		{"attempt-sequence", func(t *testing.T) {
			path := filepath.Join(filepath.Dir(retryRawPath), "attempt-003.article")
			courseContextWrite(t, f.f.root, path, repaired)
			t.Cleanup(func() {
				if err := os.Remove(filepath.Join(f.f.root, path)); err != nil {
					t.Error(err)
				}
			})
		}, "unexpected retry attempt 003"},
		{"retry-provenance-mismatch", func(t *testing.T) {
			r := receipt
			r.InputIdentitySHA256 = strings.Repeat("0", 64)
			courseContextWrite(t, f.f.root, retryReceiptPath, courseContextJSON(t, r))
			t.Cleanup(func() { courseContextWrite(t, f.f.root, retryReceiptPath, retryReceiptBytes) })
		}, ""},
		{"missing-retry-archive", func(t *testing.T) {
			if err := os.Remove(filepath.Join(f.f.root, retryArchive)); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { courseContextWrite(t, f.f.root, retryArchive, retryBundle) })
		}, ""},
		{"wrong-retry-archive-sha", func(t *testing.T) {
			courseContextWrite(t, f.f.root, retryArchive, []byte("not the imported retry ZIP"))
			t.Cleanup(func() { courseContextWrite(t, f.f.root, retryArchive, retryBundle) })
		}, ""},
		{"frozen-retry-package-tamper", func(t *testing.T) {
			changed := map[string][]byte{}
			for path, data := range retryFiles {
				changed[path] = bytes.Clone(data)
			}
			var s i18n.ApprovedIntervalSelection
			if err := json.Unmarshal(changed["formal/approved-interval-selection.json"], &s); err != nil {
				t.Fatal(err)
			}
			s.ContentPackage = "tour-v1"
			changed["formal/approved-interval-selection.json"] = courseContextJSON(t, s)
			m := rm
			m.Files = append([]i18n.TransportBundleFile(nil), rm.Files...)
			entries := []i18n.TransportBundleEntry{}
			var identity strings.Builder
			for i, item := range m.Files {
				m.Files[i] = i18n.NewTransportBundleFile(item.BundlePath, item.RepositoryPath, changed[item.BundlePath])
				identity.WriteString(item.BundlePath + "\x00" + m.Files[i].SHA256 + "\x00")
				entries = append(entries, i18n.TransportBundleEntry{Path: item.BundlePath, Data: changed[item.BundlePath]})
			}
			m.InputIdentitySHA256 = courseContextSHA([]byte(identity.String()))
			mb := courseContextJSON(t, m)
			bad, err := i18n.WriteDeterministicTransportBundle(mb, entries)
			if err != nil {
				t.Fatal(err)
			}
			r := receipt
			r.GenerationBundleSHA256 = courseContextSHA(bad)
			r.InputIdentitySHA256 = m.InputIdentitySHA256
			r.GenerationManifest = i18n.NewTransportBundleFile("generation-manifest.json", "", mb)
			path := "data/generation-archives/" + rm.Locale + "/" + r.GenerationBundleSHA256 + ".zip"
			courseContextWrite(t, f.f.root, path, bad)
			t.Cleanup(func() {
				if err := os.Remove(filepath.Join(f.f.root, path)); err != nil {
					t.Error(err)
				}
			})
			courseContextWrite(t, f.f.root, retryReceiptPath, courseContextJSON(t, r))
			t.Cleanup(func() { courseContextWrite(t, f.f.root, retryReceiptPath, retryReceiptBytes) })
		}, ""},
		{"non-retryable-status", func(t *testing.T) {
			path := filepath.Join(batch, "result.json")
			data := courseContextRead(t, f.f.root, path)
			var result i18n.RetranslationProcessResult
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}
			result.Units[0].Status = "passed"
			result.Units[0].Error = ""
			result.ValidationFailed, result.ValidationPassed = 0, 1
			courseContextWrite(t, f.f.root, path, courseContextJSON(t, result))
			t.Cleanup(func() { courseContextWrite(t, f.f.root, path, data) })
		}, "not retryable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.change(t)
			before := authorizationDiagnosisInventory(t, f.f.root)
			if out, err := authorizationDiagnosisOutput(t, func() error { return run(retry) }); err == nil || (tc.want != "" && !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("invalid retry was accepted or rejected at wrong gate: err=%v stdout=%s", err, out)
			}
			if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
				t.Fatal("negative retry mutated state")
			}
		})
		if !reflect.DeepEqual(imported, authorizationDiagnosisInventory(t, f.f.root)) {
			t.Fatal("negative retry did not preserve the imported attempt")
		}
	}
	out, err = authorizationDiagnosisOutput(t, func() error { return run(retry) })
	var result i18n.RetranslationProcessResult
	if err != nil || json.Unmarshal(out, &result) != nil || result.RestoreFailed != 0 || result.ValidationFailed != 0 || result.RestorePassed != 1 || result.ValidationPassed != 1 || result.Units[0].Status != "passed" {
		t.Fatal("true run retry restore/active validation", result, err, string(out))
	}
	var validation i18n.RetranslationValidation
	if err := json.Unmarshal(courseContextRead(t, f.f.root, validationPath), &validation); err != nil || validation.Attempt != 2 || validation.Status != "passed" || validation.RawResponsePath != rm.ExpectedOutputs[0].InstallPath {
		t.Fatal("attempt 2 validation evidence", validation, err)
	}
	history := filepath.Join(batch, "retries", flat, "attempt-001-validation.json")
	if !bytes.Equal(courseContextRead(t, f.f.root, history), firstValidation) {
		t.Fatal("attempt 1 validation not archived byte-exactly")
	}
	for path, expected := range map[string][]byte{initialReceiptPath: initialReceipt, initialArchive: initial, filepath.Join(batch, unit.InstallPath): initialRaw, retryArchive: retryBundle, retryReceiptPath: retryReceiptBytes, retryRawPath: repaired} {
		if !bytes.Equal(courseContextRead(t, f.f.root, path), expected) {
			t.Fatal("retry changed immutable invocation bytes", path)
		}
	}
	var saved i18n.RetranslationProcessResult
	if err := json.Unmarshal(courseContextRead(t, f.f.root, filepath.Join(batch, "result.json")), &saved); err != nil || !reflect.DeepEqual(result, saved) {
		t.Fatal("batch result not updated exactly", err)
	}
	reviewed := initialFiles["formal/candidates/"+name]
	parentRange := proof.Projection.ParentCandidate
	if parentRange.Start < 0 || parentRange.End <= parentRange.Start || parentRange.End > len(reviewed) {
		t.Fatal("invalid typed opaque parent range")
	}
	parent := reviewed[parentRange.Start:parentRange.End]
	// CandidatePath is batch-relative, unlike the repository-relative paths above.
	candidate := courseContextRead(t, filepath.Join(f.f.root, batch), validation.CandidatePath)
	if courseContextSHA(parent) != proof.Projection.ParentSHA256 || !bytes.Contains(candidate, parent) || bytes.Contains(candidate, []byte(proof.Projection.ParentToken)) {
		t.Fatal("opaque parent not restored exactly")
	}
	startup, err = i18n.ReadCatalog(f.f.root)
	if err != nil || len(startup.TranslationScope) != 0 {
		t.Fatal("retry altered default startup scope", err)
	}
	t.Logf("TRUE_RUN_RETRY_ENTRY=yes DEFAULT_STARTUP_SCOPE=Tour FROZEN_PACKAGE=learn-docs-v1 PACKAGE_RECONSTRUCTED_FROM_IMPORTED_ARCHIVE=yes ATTEMPT_1=validation_failed ATTEMPT_2=2 VALIDATION=PASS BUNDLE_SHA256=%s RAW_SHA256=%s", courseContextSHA(retryBundle), courseContextSHA(repaired))
}
