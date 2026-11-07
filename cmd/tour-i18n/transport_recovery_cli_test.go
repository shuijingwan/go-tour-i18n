package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestTransportRecoveryInstalledConstruction(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	f := approvedIntervalRunScopeFixture(t, 32)
	t.Chdir(f.f.root)
	cli := func(args ...string) []byte {
		t.Helper()
		t.Logf("true run: %s %s", args[0], args[1])
		out, err := authorizationDiagnosisOutput(t, func() error { return run(args) })
		if err != nil {
			t.Fatalf("true run %v: %v\n%s", args, err, out)
		}
		return out
	}
	startup, err := i18n.BuildSourceCatalog(f.f.root)
	if err != nil || len(startup.TranslationScope) != 0 {
		t.Fatal("expected default Tour startup", err)
	}
	cli("generation-bundle", "export", "--content-package", f.f.pkg, "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--approved-interval-selection", "engineering/cli-selection.json", "--output", f.output)
	bundle, err := os.ReadFile(f.output)
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(bundle, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	var gm i18n.GenerationBundleManifest
	if err = json.Unmarshal(files["manifest.json"], &gm); err != nil {
		t.Fatal(err)
	}
	outputs := t.TempDir()
	for _, out := range gm.ExpectedOutputs {
		raw := bytes.ReplaceAll(files["formal/protected-candidates/"+filepath.Base(out.BundlePath)], []byte("说明"), []byte("说明 "))
		courseContextWrite(t, outputs, filepath.Base(out.BundlePath), raw)
	}
	cli("generation-bundle", "import", "--bundle", f.output, "--input-dir", outputs, "--provider", "chatgpt", "--model", i18n.FormalChatGPTGenerationModel)
	processed := cli("retranslation", "process", "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--json")
	var result i18n.RetranslationProcessResult
	if err = json.Unmarshal(processed, &result); err != nil || result.ValidationPassed != 32 {
		t.Fatal("original 32 PASS", err)
	}
	o := i18n.TransportRecoveryOptions{Locale: f.f.locale, ContentPackage: f.f.pkg, PredecessorBatch: f.batch.BatchID}
	plan, err := i18n.PreflightTransportRecovery(f.f.root, f.f.catalog, o)
	if err != nil || plan.Status != "CURRENT" {
		t.Fatalf("installed CURRENT: %#v %v", plan, err)
	}
	path := filepath.Join(f.f.root, "internal/i18n/candidate_snapshot_outputs.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// The supported constructor declaration changes; unrelated encoding and
	// language declarations are preserved. The test never executes this copy.
	needle := []byte("outputs := map[string]protectedTranslation{}")
	if bytes.Count(b, needle) != 1 {
		t.Fatal("constructor fixture declaration changed")
	}
	b = bytes.Replace(b, needle, []byte("outputs := make(map[string]protectedTranslation)"), 1)
	if err = os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}
	plan, err = i18n.PreflightTransportRecovery(f.f.root, f.f.catalog, o)
	if err != nil || plan.Status != "READY" || plan.DirectDriftCount != 1 || plan.RecoveryCount != len(gm.ExpectedOutputs) {
		t.Fatalf("installed READY: %#v %v", plan, err)
	}
	history := authorizationDiagnosisInventory(t, f.f.root)
	readyArgs := []string{"retranslation", "transport-recovery", "--locale", f.f.locale, "--content-package", f.f.pkg, "--predecessor-batch", f.batch.BatchID}
	ready := cli(readyArgs...)
	var direct i18n.TransportRecoveryPlan
	if err = json.Unmarshal(ready, &direct); err != nil || direct.Status != "READY" || direct.DirectDriftCount != 1 || direct.RecoveryCount != 32 || direct.Identity == nil {
		t.Fatal("true-run D=1/R=32 preflight", err, string(ready))
	}
	assertTransportRecoveryCLIRejects(t, readyArgs)
	// Record publication is interrupted after sealing, before the root commit.
	// Only this synthetic repository's directory permission is changed.
	rootDir := filepath.Join(f.f.root, "data/translation-transport-recoveries", f.f.locale, "roots")
	if err = os.MkdirAll(rootDir, 0755); err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(rootDir, contentidentity.SHA256Hex(bundle)+".record.pending")
	recordArgs := append(append([]string{}, readyArgs...), "--record", "--expect-identity", *direct.Identity)
	interrupt := recoveryInterruptAtPending(t, pending, rootDir)
	partial, recordErr := authorizationDiagnosisOutput(t, func() error { return run(recordArgs) })
	interrupt()
	if recordErr == nil || !bytes.Contains(partial, []byte(`"mutation": "unknown"`)) {
		t.Fatal("record interruption must preserve sealed unknown state", recordErr, string(partial))
	}
	sealed := courseContextRead(t, "", pending)
	courseContextWrite(t, "", pending, append(sealed, []byte("corrupt")...))
	if _, err = authorizationDiagnosisOutput(t, func() error { return run(append(append([]string{}, recordArgs...), "--resume")) }); err == nil {
		t.Fatal("corrupt sealed transaction replayed")
	}
	courseContextWrite(t, "", pending, sealed)
	var recorded i18n.TransportRecoveryRecordResult
	if err = json.Unmarshal(cli(append(append([]string{}, recordArgs...), "--resume")...), &recorded); err != nil || !recorded.Recorded {
		t.Fatal("true-run record", err)
	}
	if _, err = authorizationDiagnosisOutput(t, func() error { return run(recordArgs) }); err == nil {
		t.Fatal("duplicate record accepted")
	}
	if _, err = os.Lstat(pending); !os.IsNotExist(err) {
		t.Fatal("record resume did not clear exact pending", err)
	}
	ref := recorded.Authority
	if _, err = authorizationDiagnosisOutput(t, func() error {
		return run(append(append([]string{}, readyArgs...), "--supersedes", ref.Path, "--supersedes-sha256", ref.SHA256))
	}); err == nil {
		t.Fatal("CURRENT recovery leaf superseded without a new epoch")
	}
	checkArgs := []string{"retranslation", "transport-recovery", "--check", "--locale", f.f.locale, "--content-package", f.f.pkg, "--authority", ref.Path, "--authority-sha256", ref.SHA256}
	var phase i18n.TransportRecoveryCheck
	if err = json.Unmarshal(cli(checkArgs...), &phase); err != nil || phase.Status != "READY_FOR_EXPORT" {
		t.Fatal("ready-for-export", phase, err)
	}
	exportArgs := []string{"retranslation", "export", "--locale", f.f.locale, "--content-package", f.f.pkg, "--generator", "chatgpt", "--transport-recovery", ref.Path, "--transport-recovery-sha256", ref.SHA256}
	for _, forbidden := range [][]string{{"--limit", "1"}, {"--id", direct.RecoveryUnits[0]}, {"--unit-id", direct.RecoveryUnits[0]}, {"--unit-kind", "page"}, {"--batch-id", "chosen"}, {"--previous-snapshot-id", "chosen"}, {"--surface-reopen-id", "chosen"}, {"--representation-bridge-id", "chosen"}, {"--glossary-stale"}, {"--allow-reexport"}} {
		if _, err = authorizationDiagnosisOutput(t, func() error { return run(append(append([]string{}, exportArgs...), forbidden...)) }); err == nil {
			t.Fatal("recovery export accepted external scope", forbidden)
		}
	}
	exportPending := filepath.Join(f.f.root, filepath.Dir(ref.Path), ".export.pending")
	interrupt = recoveryInterruptAtPending(t, exportPending, filepath.Join(f.f.root, "data/retranslation-runs", f.f.locale))
	_, exportErr := authorizationDiagnosisOutput(t, func() error { return run(exportArgs) })
	interrupt()
	if exportErr == nil || !strings.Contains(exportErr.Error(), "mutation=unknown") {
		t.Fatal("expected sealed export interruption", exportErr)
	}
	exportSummary := cli(append(append([]string{}, exportArgs...), "--resume")...)
	if _, err = authorizationDiagnosisOutput(t, func() error { return run(exportArgs) }); err == nil {
		t.Fatal("duplicate child export accepted")
	}
	if _, err = os.Lstat(exportPending); !os.IsNotExist(err) {
		t.Fatal("export resume left pending", err)
	}
	batchID := ""
	for _, line := range strings.Split(string(exportSummary), "\n") {
		if strings.HasPrefix(line, "batch_id: ") {
			batchID = strings.TrimPrefix(line, "batch_id: ")
		}
	}
	if batchID == "" || batchID == f.batch.BatchID {
		t.Fatal("new successor batch id missing")
	}
	newBundlePath := filepath.Join(t.TempDir(), "recovery.zip")
	cli("generation-bundle", "export", "--content-package", f.f.pkg, "--locale", f.f.locale, "--batch-id", batchID, "--output", newBundlePath)
	cli("generation-bundle", "current-check", "--content-package", f.f.pkg, "--bundle", newBundlePath)
	newBundle, err := os.ReadFile(newBundlePath)
	if err != nil {
		t.Fatal(err)
	}
	newFiles, err := i18n.ReadTransportBundle(newBundle, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	var newManifest i18n.GenerationBundleManifest
	if err = json.Unmarshal(newFiles["manifest.json"], &newManifest); err != nil || newManifest.TaskKind != i18n.TransportRecoveryTaskKind || len(newManifest.ExpectedOutputs) != 32 || len(newFiles["formal/transport-recovery/construction.json"]) == 0 {
		t.Fatal("full-R current recovery Bundle", err)
	}
	newOutputs, executionPath, executionSHA := syntheticRecoveryModelCall(t, f.f.root, newBundle, newManifest, newFiles, ref, "")
	cli("generation-bundle", "import", "--bundle", newBundlePath, "--input-dir", newOutputs, "--provider", "chatgpt", "--model", i18n.FormalChatGPTGenerationModel, "--generation-execution", executionPath, "--generation-execution-sha256", executionSHA)
	if err = json.Unmarshal(cli("retranslation", "process", "--locale", f.f.locale, "--batch-id", batchID, "--json"), &result); err != nil || result.ValidationPassed != 32 {
		t.Fatal("recovery process 32 PASS", err, result.ValidationPassed)
	}
	if err = json.Unmarshal(cli(checkArgs...), &phase); err != nil || phase.Status != "VALIDATED" {
		t.Fatal("validated recovery check", phase, err)
	}
	var snap struct {
		Manifest i18n.QualityCheckSnapshotManifest `json:"manifest"`
	}
	if err = json.Unmarshal(cli("quality-check", "snapshot", "--locale", f.f.locale, "--snapshot-id", "recovery-successor", "--content-package", f.f.pkg, "--preflight"), &snap); err != nil {
		t.Fatal(err)
	}
	selected := map[string]bool{}
	for _, id := range direct.RecoveryUnits {
		selected[id] = true
	}
	for _, u := range snap.Manifest.Units {
		if selected[u.UnitID] && u.SelectedBatchID != batchID {
			t.Fatal("Snapshot failed to select full successor R", u.UnitID)
		}
	}
	after := authorizationDiagnosisInventory(t, f.f.root)
	for path, sha := range history {
		if after[path] != sha {
			t.Fatal("recovery changed immutable predecessor", path)
		}
	}
	if reflect.DeepEqual(history, after) {
		t.Fatal("synthetic recovery did not publish successor state")
	}
	t.Logf("TRUE_RUN=PASS default_startup=Tour D=1 R=32 new_generation_outputs=32 successor=%s old_invocation_preserved=yes Snapshot_preflight=PASS", batchID)
	assertRecoveryRetryLifecycle(t, cli, f.f.root, f.f.locale, f.f.pkg, batchID, direct.DirectDriftUnits[0])
}

// Interrupt only a TempDir transaction after its sealed owner exists. Reads
// remain possible, but its next publication cannot succeed. No production
// hook or alternative implementation is involved.
func recoveryInterruptAtPending(t *testing.T, pending, deniedDirectory string) func() {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Fatal("transaction interruption test requires ordinary filesystem permissions")
	}
	stop := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		deadline := time.NewTimer(3 * time.Minute)
		defer deadline.Stop()
		for {
			select {
			case <-stop:
				result <- fmt.Errorf("pending transaction was never observed")
				return
			case <-deadline.C:
				result <- fmt.Errorf("pending transaction timeout")
				return
			case <-ticker.C:
				if info, err := os.Lstat(pending); err == nil && info.Mode().IsRegular() {
					result <- os.Chmod(deniedDirectory, 0500)
					return
				}
			}
		}
	}()
	finished := false
	finish := func() {
		if finished {
			return
		}
		finished = true
		close(stop)
		err := <-result
		restoreErr := os.Chmod(deniedDirectory, 0755)
		if err != nil || restoreErr != nil {
			t.Errorf("synthetic transaction interruption: %v; restore: %v", err, restoreErr)
		}
	}
	t.Cleanup(finish)
	return finish
}

func assertTransportRecoveryCLIRejects(t *testing.T, base []string) {
	t.Helper()
	for _, extra := range [][]string{
		{"--unit-id", "any"}, {"--id", "any"}, {"--limit", "1"}, {"--batch-id", "any"}, {"--reopen-id", "any"},
		{"--force"}, {"--ignore-drift"}, {"--allow-missing"}, {"--approved-interval-selection", "any.json"},
		{"--content-package", "duplicate"}, {"--authority", "override.json"}, {"--record"}, {"--resume"},
		{"--check", "--record"}, {"--expect-identity", strings.Repeat("0", 64)}, {"--supersedes", "prior.json"},
	} {
		args := append(append([]string{}, base...), extra...)
		if _, err := authorizationDiagnosisOutput(t, func() error { return run(args) }); err == nil {
			t.Fatal("recovery CLI accepted invalid flags", extra)
		}
	}
	for _, args := range [][]string{
		{"retranslation", "transport-recovery", "--locale", base[3], "--predecessor-batch", base[7]},
		{"retranslation", "transport-recovery", "--locale", base[3], "--content-package=", "--predecessor-batch", base[7]},
	} {
		if _, err := authorizationDiagnosisOutput(t, func() error { return run(args) }); err == nil {
			t.Fatal("recovery CLI accepted implicit/empty package")
		}
	}
	t.Log("TRUE_RUN_NEGATIVE_FLAGS=PASS no subset/force/override/implicit package")
}

func TestTransportRecoveryPreflightBoundaries(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	f := approvedIntervalRunFixture(t)
	t.Chdir(f.f.root)
	cli := func(args ...string) []byte {
		t.Helper()
		out, err := authorizationDiagnosisOutput(t, func() error { return run(args) })
		if err != nil {
			t.Fatal(args, err, string(out))
		}
		return out
	}
	cli("generation-bundle", "export", "--content-package", f.f.pkg, "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--approved-interval-selection", "engineering/cli-selection.json", "--output", f.output)
	bundle := courseContextRead(t, "", f.output)
	files, err := i18n.ReadTransportBundle(bundle, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	var gm i18n.GenerationBundleManifest
	if err = json.Unmarshal(files["manifest.json"], &gm); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, o := range gm.ExpectedOutputs {
		courseContextWrite(t, dir, filepath.Base(o.BundlePath), bytes.ReplaceAll(files["formal/protected-candidates/"+filepath.Base(o.BundlePath)], []byte("说明"), []byte("说明 ")))
	}
	cli("generation-bundle", "import", "--bundle", f.output, "--input-dir", dir, "--provider", "chatgpt", "--model", i18n.FormalChatGPTGenerationModel)
	cli("retranslation", "process", "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--json")
	o := i18n.TransportRecoveryOptions{Locale: f.f.locale, ContentPackage: f.f.pkg, PredecessorBatch: f.batch.BatchID}
	p, err := i18n.PreflightTransportRecovery(f.f.root, f.f.catalog, o)
	if err != nil || p.Status != "CURRENT" || p.Identity != nil || p.PlannedAuthority != nil {
		t.Fatal("CURRENT must grant no recovery", err)
	}
	if _, err = i18n.RecordTransportRecovery(f.f.root, f.f.catalog, o, strings.Repeat("0", 64), false); err == nil {
		t.Fatal("CURRENT invocation recorded recovery")
	}
	t.Run("encoding-only-proof-envelope", func(t *testing.T) {
		// Represent a historical producer whose only difference was JSON
		// spelling. Re-seal its synthetic archive/receipt consistently, so the
		// decision is made on typed construction, not a corrupt SHA/ZIP.
		changed := gm
		changed.Files = append([]i18n.TransportBundleFile(nil), gm.Files...)
		entries := []i18n.TransportBundleEntry{}
		var identity strings.Builder
		for i, f := range changed.Files {
			b := files[f.BundlePath]
			if strings.HasPrefix(f.BundlePath, "formal/approved-interval-transport/") {
				var encoded bytes.Buffer
				if err := json.Indent(&encoded, b, "", "  "); err != nil {
					t.Fatal(err)
				}
				if bytes.Equal(b, encoded.Bytes()) {
					t.Fatal("proof encoding perturbation ineffective")
				}
				b = encoded.Bytes()
			}
			changed.Files[i] = i18n.NewTransportBundleFile(f.BundlePath, f.RepositoryPath, b)
			identity.WriteString(f.BundlePath + "\x00" + changed.Files[i].SHA256 + "\x00")
			entries = append(entries, i18n.TransportBundleEntry{Path: f.BundlePath, Data: b})
		}
		changed.InputIdentitySHA256 = courseContextSHA([]byte(identity.String()))
		mb := courseContextJSON(t, changed)
		archive, err := i18n.WriteDeterministicTransportBundle(mb, entries)
		if err != nil {
			t.Fatal(err)
		}
		path := "data/generation-archives/" + o.Locale + "/" + courseContextSHA(archive) + ".zip"
		receiptPath := "data/retranslation-runs/" + o.Locale + "/" + o.PredecessorBatch + "/generation-imports/initial-manifest.json"
		receiptBytes := courseContextRead(t, f.f.root, receiptPath)
		var receipt i18n.GenerationResultBundleManifest
		if err = json.Unmarshal(receiptBytes, &receipt); err != nil {
			t.Fatal(err)
		}
		receipt.GenerationBundleSHA256 = courseContextSHA(archive)
		receipt.InputIdentitySHA256 = changed.InputIdentitySHA256
		receipt.GenerationManifest = i18n.NewTransportBundleFile("generation-manifest.json", "", mb)
		courseContextWrite(t, f.f.root, path, archive)
		defer os.Remove(filepath.Join(f.f.root, path))
		courseContextWrite(t, f.f.root, receiptPath, courseContextJSON(t, receipt))
		defer courseContextWrite(t, f.f.root, receiptPath, receiptBytes)
		plan, err := i18n.PreflightTransportRecovery(f.f.root, f.f.catalog, o)
		if err == nil || plan.Status != "BLOCKED" || plan.Identity != nil || plan.PlannedAuthority != nil || len(plan.Blockers) != 1 || plan.Blockers[0].Code != "ENCODING_ONLY_MISMATCH" {
			t.Fatal("encoding-only mismatch granted recovery or bypassed typed predicate", err, plan.Blockers)
		}
	})
	sourcePath := "internal/i18n/candidate_snapshot_outputs.go"
	original := courseContextRead(t, f.f.root, sourcePath)
	drift := bytes.Replace(original, []byte("outputs := map[string]protectedTranslation{}"), []byte("outputs := make(map[string]protectedTranslation)"), 1)
	courseContextWrite(t, f.f.root, sourcePath, drift)
	p, err = i18n.PreflightTransportRecovery(f.f.root, f.f.catalog, o)
	if err != nil || p.Status != "READY" {
		t.Fatal("negative fixture READY", err)
	}
	base := "data/retranslation-runs/" + o.Locale + "/" + o.PredecessorBatch + "/"
	unit, err := f.f.catalog.Unit(gm.ExpectedOutputs[0].UnitID)
	if err != nil {
		t.Fatal(err)
	}
	archive := "data/generation-archives/" + o.Locale + "/" + courseContextSHA(bundle) + ".zip"
	blocked := func(t *testing.T, options i18n.TransportRecoveryOptions) {
		t.Helper()
		before := authorizationDiagnosisInventory(t, f.f.root)
		plan, err := i18n.PreflightTransportRecovery(f.f.root, f.f.catalog, options)
		if err == nil || plan.Status != "BLOCKED" || plan.Identity != nil || plan.PlannedAuthority != nil || plan.Mutation != "none" {
			t.Fatal("failure created recovery rights", err, plan)
		}
		if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
			t.Fatal("negative preflight mutated formal state")
		}
	}
	for _, tc := range []struct {
		name, path string
		change     func([]byte) []byte
	}{
		{"archive-wrong-sha", archive, func([]byte) []byte { return []byte("not original archive") }},
		{"receipt-identity", base + "generation-imports/initial-manifest.json", func(b []byte) []byte {
			var r i18n.GenerationResultBundleManifest
			_ = json.Unmarshal(b, &r)
			r.InputIdentitySHA256 = strings.Repeat("0", 64)
			return courseContextJSON(t, r)
		}},
		{"old-validation-nonpass", base + "result.json", func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte(`"status": "passed"`), []byte(`"status": "validation_failed"`))
		}},
		{"raw-drift", base + "raw-responses/" + filepath.Base(gm.ExpectedOutputs[0].BundlePath), func(b []byte) []byte { return append(b, []byte("changed")...) }},
		{"glossary-drift", "locales/" + o.Locale + "/glossary.yaml", func(b []byte) []byte { return append(b, []byte("\n# changed glossary bytes\n")...) }},
		{"source-drift", unit.SourcePath, func(b []byte) []byte { return append(b, []byte("\nchanged canonical source\n")...) }},
		{"unclassified-producer", sourcePath, func(b []byte) []byte { return append(b, []byte("\nfunc unrelatedRecoveryFixture() {}\n")...) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := courseContextRead(t, f.f.root, tc.path)
			changed := tc.change(bytes.Clone(b))
			if bytes.Equal(b, changed) {
				t.Fatal("negative perturbation ineffective")
			}
			courseContextWrite(t, f.f.root, tc.path, changed)
			defer courseContextWrite(t, f.f.root, tc.path, b)
			blocked(t, o)
		})
	}
	t.Run("missing-archive", func(t *testing.T) {
		b := courseContextRead(t, f.f.root, archive)
		if err := os.Remove(filepath.Join(f.f.root, archive)); err != nil {
			t.Fatal(err)
		}
		defer courseContextWrite(t, f.f.root, archive, b)
		blocked(t, o)
	})
	for _, tc := range []struct {
		name    string
		options i18n.TransportRecoveryOptions
	}{
		{"implicit-package", i18n.TransportRecoveryOptions{Locale: o.Locale, PredecessorBatch: o.PredecessorBatch}},
		{"wrong-package", i18n.TransportRecoveryOptions{Locale: o.Locale, ContentPackage: "tour-v1", PredecessorBatch: o.PredecessorBatch}},
		{"wrong-predecessor", i18n.TransportRecoveryOptions{Locale: o.Locale, ContentPackage: o.ContentPackage, PredecessorBatch: "chatgpt-" + o.Locale + "-999"}},
	} {
		t.Run(tc.name, func(t *testing.T) { blocked(t, tc.options) })
	}
	t.Run("expect-CAS-change", func(t *testing.T) {
		changed := bytes.Replace(drift, []byte("make(map[string]protectedTranslation)"), []byte("make(map[string]protectedTranslation, 0)"), 1)
		courseContextWrite(t, f.f.root, sourcePath, changed)
		defer courseContextWrite(t, f.f.root, sourcePath, drift)
		before := authorizationDiagnosisInventory(t, f.f.root)
		r, err := i18n.RecordTransportRecovery(f.f.root, f.f.catalog, o, *p.Identity, false)
		if err == nil || r.Recorded || r.Mutation != "none" {
			t.Fatal("changed epoch accepted old expect identity", err)
		}
		if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
			t.Fatal("failed CAS published bytes")
		}
	})
	t.Run("overlapping-pending", func(t *testing.T) {
		pending := "data/translation-transport-recoveries/" + o.Locale + "/roots/" + courseContextSHA(bundle) + ".record.pending"
		courseContextWrite(t, f.f.root, pending, []byte("unknown sealed state"))
		defer os.Remove(filepath.Join(f.f.root, pending))
		blocked(t, o)
	})
}

func assertRecoveryRetryLifecycle(t *testing.T, cli func(...string) []byte, root, locale, pkg, predecessor, failedUnit string) {
	t.Helper()
	before := authorizationDiagnosisInventory(t, root)
	path := "internal/i18n/candidate_snapshot_outputs.go"
	b := courseContextRead(t, root, path)
	needle := []byte("outputs := make(map[string]protectedTranslation)")
	if bytes.Count(b, needle) != 1 {
		t.Fatal("second supported constructor epoch fixture missing")
	}
	courseContextWrite(t, root, path, bytes.Replace(b, needle, []byte("outputs := make(map[string]protectedTranslation, 0)"), 1))
	args := []string{"retranslation", "transport-recovery", "--locale", locale, "--content-package", pkg, "--predecessor-batch", predecessor}
	var plan i18n.TransportRecoveryPlan
	if err := json.Unmarshal(cli(args...), &plan); err != nil || plan.Status != "READY" || plan.DirectDriftCount != 1 || plan.RecoveryCount != 32 || plan.Identity == nil {
		t.Fatal("installed recovery child must support a fresh full-R recovery epoch", err)
	}
	var record i18n.TransportRecoveryRecordResult
	if err := json.Unmarshal(cli(append(append([]string{}, args...), "--record", "--expect-identity", *plan.Identity)...), &record); err != nil || !record.Recorded {
		t.Fatal("second recovery record", err)
	}
	ref := record.Authority
	text := cli("retranslation", "export", "--locale", locale, "--content-package", pkg, "--generator", "chatgpt", "--transport-recovery", ref.Path, "--transport-recovery-sha256", ref.SHA256)
	batch := ""
	for _, line := range strings.Split(string(text), "\n") {
		if strings.HasPrefix(line, "batch_id: ") {
			batch = strings.TrimPrefix(line, "batch_id: ")
		}
	}
	if batch == "" || batch == predecessor {
		t.Fatal("second recovery requires a new invocation")
	}
	bundlePath := filepath.Join(t.TempDir(), "recovery-retry-initial.zip")
	cli("generation-bundle", "export", "--content-package", pkg, "--locale", locale, "--batch-id", batch, "--output", bundlePath)
	bundle := courseContextRead(t, "", bundlePath)
	files, err := i18n.ReadTransportBundle(bundle, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	var gm i18n.GenerationBundleManifest
	if err = json.Unmarshal(files["manifest.json"], &gm); err != nil || len(gm.ExpectedOutputs) != 32 {
		t.Fatal("second full-R Generation", err)
	}
	dir, execution, sha := syntheticRecoveryModelCall(t, root, bundle, gm, files, ref, failedUnit)
	cli("generation-bundle", "import", "--bundle", bundlePath, "--input-dir", dir, "--provider", "chatgpt", "--model", i18n.FormalChatGPTGenerationModel, "--generation-execution", execution, "--generation-execution-sha256", sha)
	out, processErr := authorizationDiagnosisOutput(t, func() error {
		return run([]string{"retranslation", "process", "--locale", locale, "--batch-id", batch, "--json"})
	})
	var result i18n.RetranslationProcessResult
	if err = json.Unmarshal(out, &result); err != nil || processErr == nil || result.ValidationPassed != 31 || result.ValidationFailed != 1 || result.RestorePassed != 32 {
		t.Fatal("one genuine successor validation failure required", err, processErr, result.ValidationPassed, result.ValidationFailed)
	}
	for _, u := range result.Units {
		if u.Status == "validation_failed" && u.UnitID != failedUnit {
			t.Fatal("unexpected retry scope")
		}
	}
	if _, err := authorizationDiagnosisOutput(t, func() error {
		return run([]string{"quality-check", "snapshot", "--locale", locale, "--snapshot-id", "recovery-failed-no-fallback", "--content-package", pkg, "--preflight"})
	}); err == nil {
		t.Fatal("failed successor fell back to prior PASS")
	}
	passed := map[string][]byte{}
	base := "data/retranslation-runs/" + locale + "/" + batch + "/"
	for _, u := range result.Units {
		if u.Status == "passed" {
			passed[base+u.CandidatePath] = courseContextRead(t, root, base+u.CandidatePath)
			passed[base+u.ValidationPath] = courseContextRead(t, root, base+u.ValidationPath)
		}
	}
	validation := courseContextRead(t, root, base+"validation/"+failedUnit+".json")
	retryPath := filepath.Join(t.TempDir(), "recovery-attempt-002.zip")
	cli("generation-bundle", "export", "--content-package", pkg, "--locale", locale, "--batch-id", batch, "--unit-id", failedUnit, "--output", retryPath)
	retry := courseContextRead(t, "", retryPath)
	rf, err := i18n.ReadTransportBundle(retry, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	var rm i18n.GenerationBundleManifest
	if err = json.Unmarshal(rf["manifest.json"], &rm); err != nil || rm.Attempt != 2 || len(rm.ExpectedOutputs) != 1 || rm.ExpectedOutputs[0].UnitID != failedUnit {
		t.Fatal("exact attempt-2 Generation", err)
	}
	dir, execution, sha = syntheticRecoveryModelCall(t, root, retry, rm, rf, ref, "")
	cli("generation-bundle", "import", "--bundle", retryPath, "--input-dir", dir, "--provider", "chatgpt", "--model", i18n.FormalChatGPTGenerationModel, "--generation-execution", execution, "--generation-execution-sha256", sha)
	if err = json.Unmarshal(cli("retranslation", "retry", "--locale", locale, "--batch-id", batch, "--unit-id", failedUnit, "--json"), &result); err != nil || result.ValidationPassed != 32 || result.ValidationFailed != 0 {
		t.Fatal("recovery retry full PASS", err, result.ValidationPassed)
	}
	for path, b := range passed {
		if !bytes.Equal(b, courseContextRead(t, root, path)) {
			t.Fatal("retry changed a successful sibling", path)
		}
	}
	if !bytes.Equal(validation, courseContextRead(t, root, base+"retries/"+failedUnit+"/attempt-001-validation.json")) {
		t.Fatal("retry validation history lost exact bytes")
	}
	var snapshot struct {
		Manifest i18n.QualityCheckSnapshotManifest `json:"manifest"`
	}
	if err = json.Unmarshal(cli("quality-check", "snapshot", "--locale", locale, "--snapshot-id", "recovery-retried-successor", "--content-package", pkg, "--preflight"), &snapshot); err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, id := range plan.RecoveryUnits {
		ids[id] = true
	}
	for _, u := range snapshot.Manifest.Units {
		if ids[u.UnitID] && u.SelectedBatchID != batch {
			t.Fatal("retry successor R not selected")
		}
		if u.UnitID == failedUnit && u.Attempt != 2 {
			t.Fatal("Snapshot did not select retry attempt 2")
		}
	}
	after := authorizationDiagnosisInventory(t, root)
	for path, sha := range before {
		if after[path] != sha {
			t.Fatal("nested recovery changed prior immutable invocation", path)
		}
	}
	t.Log("RECOVERY_RETRY=PASS R=32 first_attempt_passed=31 retry_outputs=1 final_passed=32 prior_invocations_preserved=yes")
}

func syntheticRecoveryModelCall(t *testing.T, root string, bundle []byte, m i18n.GenerationBundleManifest, files map[string][]byte, authority i18n.TransportRecoveryReference, failUnit string) (string, string, string) {
	t.Helper()
	base := fmt.Sprintf("engineering/model-calls/%s/attempt-%03d", m.BatchID, m.Attempt)
	inputPath := base + "/input.zip"
	courseContextWrite(t, root, inputPath, bundle)
	ref := func(path string, b []byte) i18n.RecoveryFileReference {
		return i18n.RecoveryFileReference{Path: path, SHA256: courseContextSHA(b), Size: int64(len(b))}
	}
	call := i18n.TransportRecoveryModelInvocation{Session: "synthetic-generation-role", Invocation: base, BundleSHA256: courseContextSHA(bundle), InputIdentitySHA256: m.InputIdentitySHA256, TaskKind: m.TaskKind, Attempt: m.Attempt, Provider: "chatgpt", Model: i18n.FormalChatGPTGenerationModel, InputEvidence: ref(inputPath, bundle)}
	for _, out := range m.ExpectedOutputs {
		raw := files["formal/protected-candidates/"+filepath.Base(out.BundlePath)]
		if out.UnitID != failUnit {
			raw = bytes.ReplaceAll(raw, []byte("说明"), []byte("说明 "))
		}
		path := base + "/outputs/" + filepath.Base(out.BundlePath)
		courseContextWrite(t, root, path, raw)
		evidence := ref(path, raw)
		call.Outputs = append(call.Outputs, i18n.TransportRecoveryExecutionOutput{Output: i18n.NewTransportBundleFile(out.BundlePath, "", raw), Evidence: evidence})
		call.Evidence = append(call.Evidence, evidence)
	}
	e := i18n.TransportRecoveryGenerationExecution{Schema: i18n.TransportRecoveryExecutionSchema, Kind: "model-generation", Locale: m.Locale, ContentPackage: "learn-docs-v1", BatchID: m.BatchID, TaskKind: m.TaskKind, Attempt: m.Attempt, Authority: authority, BundleSHA256: courseContextSHA(bundle), InputIdentitySHA256: m.InputIdentitySHA256, Provider: "chatgpt", Model: i18n.FormalChatGPTGenerationModel, Invocations: []i18n.TransportRecoveryModelInvocation{call}}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	e.Identity = courseContextSHA(b)
	b, err = json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	path := base + "/execution.json"
	courseContextWrite(t, root, path, b)
	return filepath.Join(root, base, "outputs"), path, courseContextSHA(b)
}
