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

// Extend the real run export/import test, reusing its same TempDir invocation.
// No selected Catalog is passed to process: run reloads its default Tour scope.
func assertApprovedIntervalProcessRun(t *testing.T, f nativeGenerationFixture, manifest i18n.GenerationBundleManifest, files map[string][]byte, bundle []byte, receiptPath, archive string, imported map[string]string) {
	t.Helper()
	startup, err := i18n.ReadCatalog(f.f.root)
	if err != nil || len(startup.TranslationScope) != 0 {
		t.Fatal("fixture startup must remain default Tour", err)
	}
	args := []string{"retranslation", "process", "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--json"}
	batchDir := filepath.Join(f.f.root, "data/retranslation-runs", f.f.locale, f.batch.BatchID)
	receiptBytes := courseContextRead(t, f.f.root, receiptPath)
	var receipt i18n.GenerationResultBundleManifest
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		t.Fatal(err)
	}
	writeReceipt := func(t *testing.T, r i18n.GenerationResultBundleManifest) {
		t.Helper()
		courseContextWrite(t, f.f.root, receiptPath, courseContextJSON(t, r))
		t.Cleanup(func() { courseContextWrite(t, f.f.root, receiptPath, receiptBytes) })
	}
	installChangedArchive := func(t *testing.T, data []byte, r i18n.GenerationResultBundleManifest) {
		t.Helper()
		r.GenerationBundleSHA256 = courseContextSHA(data)
		path := "data/generation-archives/" + manifest.Locale + "/" + r.GenerationBundleSHA256 + ".zip"
		if _, err := os.Lstat(filepath.Join(f.f.root, path)); !os.IsNotExist(err) {
			t.Fatal("negative fixture must create a new archive", err)
		}
		courseContextWrite(t, f.f.root, path, data)
		t.Cleanup(func() {
			if err := os.Remove(filepath.Join(f.f.root, path)); err != nil {
				t.Error(err)
			}
		})
		writeReceipt(t, r)
	}
	mutateFrozen := func(t *testing.T, change func(map[string][]byte)) {
		t.Helper()
		changed := map[string][]byte{}
		for path, data := range files {
			changed[path] = bytes.Clone(data)
		}
		change(changed)
		m := manifest
		m.Files = append([]i18n.TransportBundleFile(nil), manifest.Files...)
		entries := []i18n.TransportBundleEntry{}
		var identity strings.Builder
		for i, item := range m.Files {
			data := changed[item.BundlePath]
			m.Files[i] = i18n.NewTransportBundleFile(item.BundlePath, item.RepositoryPath, data)
			identity.WriteString(item.BundlePath + "\x00" + m.Files[i].SHA256 + "\x00")
			entries = append(entries, i18n.TransportBundleEntry{Path: item.BundlePath, Data: data})
		}
		m.InputIdentitySHA256 = courseContextSHA([]byte(identity.String()))
		mb := courseContextJSON(t, m)
		// A readable historical encoding with fully resealed inventory/receipt.
		// Rejection must come from the frozen authority/proof, not broken ZIP CRC.
		data, err := i18n.WriteDeterministicTransportBundle(mb, entries)
		if err != nil {
			t.Fatal(err)
		}
		r := receipt
		r.InputIdentitySHA256 = m.InputIdentitySHA256
		r.GenerationManifest = i18n.NewTransportBundleFile("generation-manifest.json", "", mb)
		installChangedArchive(t, data, r)
	}
	for _, tc := range []struct {
		name   string
		change func(*testing.T)
		want   string
	}{
		{"missing-archive", func(t *testing.T) {
			if err := os.Remove(filepath.Join(f.f.root, archive)); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { courseContextWrite(t, f.f.root, archive, bundle) })
		}, "generation-archives"},
		{"wrong-archive-sha", func(t *testing.T) {
			courseContextWrite(t, f.f.root, archive, []byte("corrupted original archive"))
			t.Cleanup(func() { courseContextWrite(t, f.f.root, archive, bundle) })
		}, "archive SHA256 mismatch"},
		{"corrupt-archive-with-matching-sha", func(t *testing.T) {
			installChangedArchive(t, []byte("not a ZIP"), receipt)
		}, "open transport bundle"},
		{"receipt-archive-identity", func(t *testing.T) {
			r := receipt
			r.InputIdentitySHA256 = strings.Repeat("0", 64)
			writeReceipt(t, r)
		}, "provenance/input identity mismatch"},
		{"receipt-batch-identity", func(t *testing.T) {
			r := receipt
			r.BatchID += "-other"
			writeReceipt(t, r)
		}, "receipt identity mismatch"},
		{"frozen-package-tamper", func(t *testing.T) {
			mutateFrozen(t, func(changed map[string][]byte) {
				var s i18n.ApprovedIntervalSelection
				if err := json.Unmarshal(changed["formal/approved-interval-selection.json"], &s); err != nil {
					t.Fatal(err)
				}
				s.ContentPackage = "tour-v1"
				changed["formal/approved-interval-selection.json"] = courseContextJSON(t, s)
			})
		}, ""},
		{"frozen-proof-tamper", func(t *testing.T) {
			mutateFrozen(t, func(changed map[string][]byte) {
				changed["formal/approved-interval-transport/"+f.f.ids[0]+".json"] = []byte("{}\n")
			})
		}, "decoder/proof drift"},
	} {
		t.Run("process/"+tc.name, func(t *testing.T) {
			tc.change(t)
			before := authorizationDiagnosisInventory(t, f.f.root)
			if _, err := authorizationDiagnosisOutput(t, func() error { return run(args) }); err == nil || (tc.want != "" && !strings.Contains(err.Error(), tc.want)) {
				t.Fatal("invalid imported graph not rejected at expected gate", err)
			}
			if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
				t.Fatal("failed preflight mutated formal state")
			}
			for _, name := range []string{"result.json", "candidates", "validation"} {
				if _, err := os.Lstat(filepath.Join(batchDir, name)); !os.IsNotExist(err) {
					t.Fatal("failed preflight produced process output", name, err)
				}
			}
			if paths, err := filepath.Glob(filepath.Join(batchDir, ".process-staging-*")); err != nil || len(paths) != 0 {
				t.Fatal("failed preflight created staging", paths, err)
			}
		})
		if !reflect.DeepEqual(imported, authorizationDiagnosisInventory(t, f.f.root)) {
			t.Fatal("negative fixture failed to restore exact imported state")
		}
	}
	out, err := authorizationDiagnosisOutput(t, func() error { return run(args) })
	if err != nil {
		t.Fatal("true run process from default Tour scope", err, string(out))
	}
	var result i18n.RetranslationProcessResult
	if err := json.Unmarshal(out, &result); err != nil || result.BatchID != manifest.BatchID || result.RestorePassed != len(manifest.ExpectedOutputs) || result.ValidationPassed != len(manifest.ExpectedOutputs) || result.RestoreFailed != 0 || result.ValidationFailed != 0 {
		t.Fatal("restore/current validation did not pass", result, err, string(out))
	}
	var saved i18n.RetranslationProcessResult
	if err := json.Unmarshal(courseContextRead(t, batchDir, "result.json"), &saved); err != nil || !reflect.DeepEqual(saved, result) {
		t.Fatal("result evidence differs", err)
	}
	for _, unit := range result.Units {
		var validation i18n.RetranslationValidation
		if err := json.Unmarshal(courseContextRead(t, batchDir, unit.ValidationPath), &validation); err != nil || validation.Status != "passed" || validation.Attempt != 1 || validation.UnitID != unit.UnitID {
			t.Fatal("validation evidence", validation, err)
		}
		candidate := courseContextRead(t, batchDir, unit.CandidatePath)
		var proof i18n.ApprovedIntervalTransport
		if err := json.Unmarshal(files["formal/approved-interval-transport/"+unit.UnitID+".json"], &proof); err != nil {
			t.Fatal(err)
		}
		var name string
		for _, expected := range manifest.ExpectedOutputs {
			if expected.UnitID == unit.UnitID {
				name = filepath.Base(expected.BundlePath)
			}
		}
		reviewed := files["formal/candidates/"+name]
		parentRange := proof.Projection.ParentCandidate
		if parentRange.Start < 0 || parentRange.End <= parentRange.Start || parentRange.End > len(reviewed) {
			t.Fatal("invalid typed parent coordinates")
		}
		parent := reviewed[parentRange.Start:parentRange.End]
		if courseContextSHA(parent) != proof.Projection.ParentSHA256 || !bytes.Contains(candidate, parent) || bytes.Contains(candidate, []byte(proof.Projection.ParentToken)) {
			t.Fatal("whole opaque parent not restored exactly")
		}
	}
	after := authorizationDiagnosisInventory(t, f.f.root)
	for path, hash := range imported {
		if after[path] != hash {
			t.Fatal("process rewrote import receipt/archive/raw or authority", path)
		}
	}
	if !bytes.Equal(courseContextRead(t, f.f.root, archive), bundle) || !bytes.Equal(courseContextRead(t, f.f.root, receiptPath), receiptBytes) {
		t.Fatal("process changed original imported invocation")
	}
	currentStartup, err := i18n.ReadCatalog(f.f.root)
	if err != nil || len(currentStartup.TranslationScope) != 0 {
		t.Fatal("process changed startup scope", err)
	}
	t.Log("TRUE_RUN_PROCESS_ENTRY=yes DEFAULT_STARTUP_SCOPE=Tour FROZEN_PACKAGE=learn-docs-v1 PACKAGE_RECONSTRUCTED_FROM_IMPORTED_ARCHIVE=yes RESTORE=PASS ACTIVE_VALIDATION=PASS OPAQUE_PARENT_EXACT=yes IMPORT_STATE_UNCHANGED=yes")
}
