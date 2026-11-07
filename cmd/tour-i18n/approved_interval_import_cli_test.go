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

func TestApprovedIntervalDeflateDirectoryImportRun(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	f := approvedIntervalRunFixture(t)
	t.Chdir(f.f.root)
	export := []string{"generation-bundle", "export", "--content-package", f.f.pkg, "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--approved-interval-selection", "engineering/cli-selection.json", "--output", f.output}
	if _, err := authorizationDiagnosisOutput(t, func() error { return run(export) }); err != nil {
		t.Fatal("synthetic approved export through run", err)
	}
	bundle, err := os.ReadFile(f.output)
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil || len(z.File) == 0 {
		t.Fatal("synthetic ZIP", err)
	}
	for _, member := range z.File {
		if member.Method != zip.Deflate {
			t.Fatal("non-Deflate canonical member", member.Name)
		}
	}
	files, err := i18n.ReadTransportBundle(bundle, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	var manifest i18n.GenerationBundleManifest
	if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.TaskKind != "translation-unit-revision" || manifest.Attempt != 1 || len(manifest.ExpectedOutputs) != len(f.f.ids) {
		t.Fatal("wrong revision fixture", manifest)
	}
	var selection i18n.ApprovedIntervalSelection
	if err := json.Unmarshal(files["formal/approved-interval-selection.json"], &selection); err != nil || selection.ContentPackage != f.f.pkg || len(selection.Units) != len(f.f.ids) {
		t.Fatal("missing frozen explicit approved scope", err)
	}
	outputs := t.TempDir()
	wantRaw := map[string][]byte{}
	for _, expected := range manifest.ExpectedOutputs {
		var proof i18n.ApprovedIntervalTransport
		if err := json.Unmarshal(files["formal/approved-interval-transport/"+expected.UnitID+".json"], &proof); err != nil || proof.Mode != i18n.ApprovedIntervalMode || proof.Record.RevisionFeedbackSource != i18n.CurrentValidationRevisionFeedback {
			t.Fatal("not an approved-interval/current-validation lifecycle", err)
		}
		name := filepath.Base(expected.BundlePath)
		raw := files["formal/protected-candidates/"+name]
		if len(raw) == 0 || !bytes.Contains(raw, []byte(proof.Projection.ParentToken)) {
			t.Fatal("missing immutable parent/template")
		}
		// The fixture deliberately creates Han/code adjacency in editable prose.
		// Repair that synthetic machine finding; opaque tokens remain verbatim.
		raw = bytes.ReplaceAll(raw, []byte("说明"), []byte("说明 "))
		raw = append(bytes.TrimRight(raw, "\n"), '\n')
		courseContextWrite(t, outputs, name, raw)
		wantRaw[expected.BundlePath] = raw
	}
	entries, err := os.ReadDir(outputs)
	if err != nil || len(entries) != len(manifest.ExpectedOutputs) {
		t.Fatal("not an exact output set", err)
	}
	check := []string{"generation-bundle", "current-check", "--content-package", f.f.pkg, "--bundle", f.output}
	if _, err := authorizationDiagnosisOutput(t, func() error { return run(check) }); err != nil {
		t.Fatal("preimport canonical reconstruction", err)
	}
	before := authorizationDiagnosisInventory(t, f.f.root)
	args := []string{"generation-bundle", "import", "--bundle", f.output, "--input-dir", outputs, "--provider", "chatgpt", "--model", "gpt-5.6-sol-high"}
	for _, tc := range []struct {
		name string
		argv []string
		want string
	}{
		{"external-package-override", append(append([]string{}, args...), "--content-package", "tour-v1"), "import does not accept --content-package"},
		{"missing-model", args[:8], "--provider and --model are required"},
		{"missing-provider", append(append([]string{}, args[:6]...), args[8:]...), "--provider and --model are required"},
		{"mutually-exclusive-result", append(append([]string{}, args...), "--result", "unused-result.zip"), "usage: generation-bundle import"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := authorizationDiagnosisOutput(t, func() error { return run(tc.argv) }); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal("real run accepted invalid import argv or failed at wrong gate", err)
			}
			if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
				t.Fatal("rejected import changed authority/state")
			}
		})
	}
	out, err := authorizationDiagnosisOutput(t, func() error { return run(args) })
	if err != nil || !strings.Contains(string(out), "Generation result imported: ") {
		t.Fatal("true run approved Deflate directory import", err, string(out))
	}
	batchDir := filepath.Join("data/retranslation-runs", f.f.locale, f.batch.BatchID)
	allowed := map[string]bool{}
	for _, expected := range manifest.ExpectedOutputs {
		path := filepath.Join(batchDir, filepath.FromSlash(expected.InstallPath))
		if !bytes.Equal(courseContextRead(t, f.f.root, path), wantRaw[expected.BundlePath]) {
			t.Fatal("installed raw bytes differ", expected.UnitID)
		}
		allowed[filepath.Join(f.f.root, path)] = true
	}
	receiptPath := filepath.Join(batchDir, "generation-imports", "initial-manifest.json")
	var receipt i18n.GenerationResultBundleManifest
	if err := json.Unmarshal(courseContextRead(t, f.f.root, receiptPath), &receipt); err != nil {
		t.Fatal(err)
	}
	sha := courseContextSHA(bundle)
	if receipt.SchemaVersion != i18n.GenerationResultBundleSchemaVersion || receipt.Kind != "go-tour-i18n/generation-result-bundle" || receipt.TaskKind != manifest.TaskKind || receipt.Locale != manifest.Locale || receipt.BatchID != manifest.BatchID || receipt.Attempt != 1 || receipt.Provider != "chatgpt" || receipt.Model != "gpt-5.6-sol-high" || receipt.GenerationBundleSHA256 != sha || receipt.InputIdentitySHA256 != manifest.InputIdentitySHA256 || len(receipt.Outputs) != len(wantRaw) {
		t.Fatal("wrong actual import provenance", receipt)
	}
	if receipt.GenerationManifest != i18n.NewTransportBundleFile("generation-manifest.json", "", files["manifest.json"]) {
		t.Fatal("receipt lost exact frozen manifest")
	}
	seen := map[string]bool{}
	for _, output := range receipt.Outputs {
		raw, ok := wantRaw[output.BundlePath]
		if !ok || seen[output.BundlePath] || output != i18n.NewTransportBundleFile(output.BundlePath, "", raw) {
			t.Fatal("wrong exact output identity", output)
		}
		seen[output.BundlePath] = true
	}
	archive := filepath.Join("data/generation-archives", manifest.Locale, sha+".zip")
	if !bytes.Equal(courseContextRead(t, f.f.root, archive), bundle) {
		t.Fatal("archive repacked the actual Deflate model input")
	}
	allowed[filepath.Join(f.f.root, archive)] = true
	allowed[filepath.Join(f.f.root, receiptPath)] = true
	if _, err := os.Lstat(filepath.Join(f.f.root, batchDir, "generation-imports", "import.pending")); !os.IsNotExist(err) {
		t.Fatal("pending marker remains", err)
	}
	after := authorizationDiagnosisInventory(t, f.f.root)
	for path, hash := range before {
		if after[path] != hash {
			t.Fatal("import rewrote existing authority/state", path)
		}
	}
	for path := range after {
		if _, exists := before[path]; !exists && !allowed[path] {
			t.Fatal("unexpected formal install", path)
		}
	}
	for _, argv := range [][]string{check, args} {
		if _, err := authorizationDiagnosisOutput(t, func() error { return run(argv) }); err == nil || !strings.Contains(err.Error(), "raw-responses already exists") {
			t.Fatal("imported revision remained pending CURRENT or replayable", err)
		}
		if !reflect.DeepEqual(after, authorizationDiagnosisInventory(t, f.f.root)) {
			t.Fatal("postimport current-check/replay changed state")
		}
	}
	t.Logf("TRUE_RUN_IMPORT=PASS approved_interval_package=%s provider=chatgpt model=gpt-5.6-sol-high attempt=1 bundle_sha256=%s input_identity=%s outputs=%d archive_exact=yes pending_clear=yes", f.f.pkg, sha, manifest.InputIdentitySHA256, len(wantRaw))
	assertApprovedIntervalProcessRun(t, f, manifest, files, bundle, receiptPath, archive, after)
}
