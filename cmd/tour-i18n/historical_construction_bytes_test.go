package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// The initial archive is a readable historical producer fixture, not a pending
// current export. Material absent from that producer is omitted BEFORE install;
// no installed archive is ever patched by this lifecycle.
func TestHistoricalConstructionBytesRunLifecycle(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	f := approvedIntervalRunFixture(t)
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"internal/contentidentity/strict_json.go", "cmd/tour-i18n/main.go", "cmd/tour-i18n/historical_construction_bytes.go"} {
		courseContextWrite(t, f.f.root, path, courseContextRead(t, repo, path))
	}
	t.Chdir(f.f.root)
	cli := func(args ...string) []byte {
		t.Helper()
		b, e := authorizationDiagnosisOutput(t, func() error { return run(args) })
		if e != nil {
			t.Fatal(args, e, string(b))
		}
		return b
	}
	cli("generation-bundle", "export", "--content-package", f.f.pkg, "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--approved-interval-selection", "engineering/cli-selection.json", "--output", f.output)
	bundle := courseContextRead(t, "", f.output)
	files, e := i18n.ReadTransportBundle(bundle, 8192, 134217728)
	if e != nil {
		t.Fatal(e)
	}
	var gm i18n.GenerationBundleManifest
	if e = json.Unmarshal(files["manifest.json"], &gm); e != nil {
		t.Fatal(e)
	}
	logical := "internal/i18n/candidate_snapshot_outputs.go"
	old := courseContextRead(t, f.f.root, logical)
	oldSHA := courseContextSHA(old)
	entries := []i18n.TransportBundleEntry{}
	kept := []i18n.TransportBundleFile{}
	var identity strings.Builder
	removed := 0
	for _, r := range gm.Files {
		b := files[r.BundlePath]
		if courseContextSHA(b) == oldSHA {
			removed++
			continue
		}
		kept = append(kept, r)
		entries = append(entries, i18n.TransportBundleEntry{Path: r.BundlePath, Data: b})
		identity.WriteString(r.BundlePath + "\x00" + r.SHA256 + "\x00")
	}
	if removed == 0 {
		t.Fatal("fixture did not omit historical material")
	}
	gm.Files = kept
	gm.InputIdentitySHA256 = courseContextSHA([]byte(identity.String()))
	mb := courseContextJSON(t, gm)
	historical, e := i18n.WriteDeterministicTransportBundle(mb, entries)
	if e != nil {
		t.Fatal(e)
	}
	batchBase := "data/retranslation-runs/" + f.f.locale + "/" + f.batch.BatchID + "/"
	archive := "data/generation-archives/" + f.f.locale + "/" + courseContextSHA(historical) + ".zip"
	receipt := i18n.GenerationResultBundleManifest{SchemaVersion: i18n.GenerationResultBundleSchemaVersion, Kind: "go-tour-i18n/generation-result-bundle", TaskKind: gm.TaskKind, Locale: gm.Locale, BatchID: gm.BatchID, Attempt: 1, Provider: "chatgpt", Model: i18n.FormalChatGPTGenerationModel, GenerationBundleSHA256: courseContextSHA(historical), InputIdentitySHA256: gm.InputIdentitySHA256, GenerationManifest: i18n.NewTransportBundleFile("generation-manifest.json", "", mb)}
	for _, out := range gm.ExpectedOutputs {
		raw := bytes.ReplaceAll(files["formal/protected-candidates/"+filepath.Base(out.BundlePath)], []byte("说明"), []byte("说明 "))
		courseContextWrite(t, f.f.root, batchBase+out.InstallPath, raw)
		receipt.Outputs = append(receipt.Outputs, i18n.NewTransportBundleFile(out.BundlePath, "", raw))
	}
	courseContextWrite(t, f.f.root, archive, historical)
	receiptPath := batchBase + "generation-imports/initial-manifest.json"
	courseContextWrite(t, f.f.root, receiptPath, courseContextJSON(t, receipt))
	cli("retranslation", "process", "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--json")
	immutable := authorizationDiagnosisInventory(t, f.f.root)
	// A supported independent producer change establishes the same old typed
	// construction through surviving bytes before the missing dependency changes.
	other := "internal/i18n/current_validation_approved_interval.go"
	before := courseContextRead(t, f.f.root, other)
	changed := bytes.Replace(before, []byte("return rows, err"), []byte("return (rows), err"), 1)
	if bytes.Equal(before, changed) {
		t.Fatal("independent producer fixture")
	}
	courseContextWrite(t, f.f.root, other, changed)
	recoveryArgs := []string{"retranslation", "transport-recovery", "--locale", f.f.locale, "--content-package", f.f.pkg, "--predecessor-batch", f.batch.BatchID}
	var baseline i18n.TransportRecoveryPlan
	if e = json.Unmarshal(cli(recoveryArgs...), &baseline); e != nil || baseline.Status != "READY" {
		t.Fatal("surviving-material baseline", e, baseline)
	}
	changed = bytes.Replace(old, []byte("outputs := map[string]protectedTranslation{}"), []byte("outputs := make(map[string]protectedTranslation)"), 1)
	if bytes.Equal(changed, old) {
		t.Fatal("target producer fixture")
	}
	courseContextWrite(t, f.f.root, logical, changed)
	b, e := authorizationDiagnosisOutput(t, func() error { return run(recoveryArgs) })
	if e == nil || !bytes.Contains(b, []byte("missing declared historical construction bytes")) {
		t.Fatal("missing material not BLOCKED", e, string(b))
	}
	acquisition := t.TempDir()
	source := filepath.Join(acquisition, "source-before.go")
	evidence := filepath.Join(acquisition, "inventory.json")
	courseContextWrite(t, "", source, old)
	eb := []byte(`{"kind":"synthetic-source-before","sha256":"` + oldSHA + `"}`)
	courseContextWrite(t, "", evidence, eb)
	args := []string{"generation-bundle", "historical-construction-bytes", "--locale", f.f.locale, "--content-package", f.f.pkg, "--batch-id", f.batch.BatchID, "--unit-id", gm.ExpectedOutputs[0].UnitID, "--logical-dependency-path", logical, "--source", source, "--source-kind", "codex-artifact", "--acquisition-evidence", evidence, "--acquisition-evidence-sha256", courseContextSHA(eb)}
	var plan i18n.HistoricalBytesPlan
	if e = json.Unmarshal(cli(args...), &plan); e != nil || plan.Status != "READY" || plan.CAS == nil || plan.AuthorityIdentity == nil || plan.PlannedAuthority == nil {
		t.Fatal("adoption preflight", e, plan)
	}
	if plan.Declaration.Size != nil || plan.PlannedAuthority.Statement.RecoveredSize != int64(len(old)) {
		t.Fatal("legacy size fabricated")
	}
	for _, tc := range []struct {
		name  string
		at    int
		value string
	}{{"wrong-locale", 3, "fr-FR"}, {"wrong-package", 5, "tour-v1"}, {"wrong-batch", 7, "chatgpt-en-GB-999"}, {"wrong-unit", 9, strings.Repeat("0", 64)}, {"undeclared-path", 11, "internal/i18n/undeclared.go"}} {
		t.Run(tc.name, func(t *testing.T) {
			bad := append([]string{}, args...)
			bad[tc.at] = tc.value
			b, e := authorizationDiagnosisOutput(t, func() error { return run(bad) })
			if e == nil || !bytes.Contains(b, []byte(`"status": "BLOCKED"`)) {
				t.Fatal("unproven historical declaration accepted", e, string(b))
			}
		})
	}

	t.Logf("PREFLIGHT_READY K=%s A=%s S=%s", *plan.DeclarationKey, *plan.AuthorityIdentity, oldSHA)
	// Audit-only machine paths/kind do not participate in A or K.
	source2 := filepath.Join(acquisition, "backup.go")
	courseContextWrite(t, "", source2, old)
	alternate := append([]string{}, args...)
	alternate[13] = source2
	alternate[15] = "backup"
	var alt i18n.HistoricalBytesPlan
	if e = json.Unmarshal(cli(alternate...), &alt); e != nil || *alt.AuthorityIdentity != *plan.AuthorityIdentity || *alt.DeclarationKey != *plan.DeclarationKey || *alt.CAS == *plan.CAS {
		t.Fatal("acquisition affected identity or not CAS", e, alt)
	}
	for _, bad := range [][]string{append(append([]string{}, args...), "--source", source), append(append([]string{}, args...), "--old-sha256", oldSHA), append(append([]string{}, args...), "--unit-id="), append(append([]string{}, args...), "--record"), append(append([]string{}, args...), "--resume"), append(append([]string{}, args...), "--check"), append(append([]string{}, args...), "--force")} {
		if _, e := authorizationDiagnosisOutput(t, func() error { return run(bad) }); e == nil {
			t.Fatal("invalid raw argv accepted", bad)
		}
	}
	record := append(append([]string{}, args...), "--record", "--expect-identity", *plan.AuthorityIdentity, "--expect-cas", *plan.CAS)
	// Wrong identity/CAS and changed acquisition reject before sealing.
	wrong := append([]string{}, record...)
	wrong[len(wrong)-1] = strings.Repeat("0", 64)
	if _, e := authorizationDiagnosisOutput(t, func() error { return run(wrong) }); e == nil {
		t.Fatal("wrong CAS accepted")
	}
	wrong = append([]string{}, record...)
	wrong[len(wrong)-3] = strings.Repeat("0", 64)
	if _, e := authorizationDiagnosisOutput(t, func() error { return run(wrong) }); e == nil {
		t.Fatal("wrong authority identity accepted")
	}

	courseContextWrite(t, "", source, append(append([]byte{}, old...), '\n'))
	if _, e := authorizationDiagnosisOutput(t, func() error { return run(record) }); e == nil {
		t.Fatal("changed/normalized source accepted")
	}
	courseContextWrite(t, "", source, old)
	// Rewriting the external inode updates its observation, so acquire fresh CAS.
	if e = json.Unmarshal(cli(args...), &plan); e != nil {
		t.Fatal(e)
	}
	record = append(append([]string{}, args...), "--record", "--expect-identity", *plan.AuthorityIdentity, "--expect-cas", *plan.CAS)
	key := *plan.DeclarationKey
	root := "data/historical-construction-bytes/"
	pending := root + "transactions/" + f.f.locale + "/" + key + ".pending"
	authorityParent := root + "authorities/" + f.f.locale
	indexParent := root + "declarations/" + f.f.locale
	txParent := root + "transactions/" + f.f.locale
	for _, p := range []string{authorityParent, indexParent, txParent} {
		if e = os.MkdirAll(filepath.Join(f.f.root, p), 0755); e != nil {
			t.Fatal(e)
		}
		p := p
		t.Cleanup(func() { _ = os.Chmod(filepath.Join(f.f.root, p), 0755) })
	}
	if e = os.Chmod(filepath.Join(f.f.root, authorityParent), 0500); e != nil {
		t.Fatal(e)
	}
	b, e = authorizationDiagnosisOutput(t, func() error { return run(record) })
	if e == nil {
		t.Fatal("expected publication interruption")
	}
	t.Log("blob/authority interrupted:", e, string(b))
	sealed := courseContextRead(t, f.f.root, pending)
	if !bytes.Equal(courseContextRead(t, f.f.root, plan.Blob.Path), old) {
		t.Fatal("blob not exact")
	}
	if _, e = os.Stat(filepath.Join(f.f.root, *plan.AuthorityPath)); !os.IsNotExist(e) {
		t.Fatal("authority unexpectedly committed", e)
	}
	_ = os.Chmod(filepath.Join(f.f.root, authorityParent), 0755)
	resume := append(append([]string{}, record...), "--resume")
	courseContextWrite(t, f.f.root, pending, []byte("{}\n"))
	if _, e := authorizationDiagnosisOutput(t, func() error { return run(resume) }); e == nil {
		t.Fatal("corrupt sealed transaction accepted")
	}
	courseContextWrite(t, f.f.root, pending, sealed)
	wrong = append([]string{}, resume...)
	wrong[15] = "backup"
	if _, e := authorizationDiagnosisOutput(t, func() error { return run(wrong) }); e == nil {
		t.Fatal("wrong sealed owner accepted")
	}
	if e = os.Remove(source); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(evidence); e != nil {
		t.Fatal(e)
	}
	_ = os.Chmod(filepath.Join(f.f.root, indexParent), 0500)
	b, e = authorizationDiagnosisOutput(t, func() error { return run(resume) })
	if e == nil {
		t.Fatal("index publication interruption expected")
	}
	t.Log("authority/index interrupted:", e, string(b))
	if _, e = os.Stat(filepath.Join(f.f.root, *plan.AuthorityPath)); e != nil {
		t.Fatal("authority absent before index", e)
	}
	index := indexParent + "/" + key + ".json"
	if _, e = os.Stat(filepath.Join(f.f.root, index)); !os.IsNotExist(e) {
		t.Fatal("index committed unexpectedly", e)
	}
	_ = os.Chmod(filepath.Join(f.f.root, indexParent), 0755)
	_ = os.Chmod(filepath.Join(f.f.root, txParent), 0500)
	b, e = authorizationDiagnosisOutput(t, func() error { return run(resume) })
	if e == nil {
		t.Fatal("cleanup interruption expected")
	}
	var committed i18n.HistoricalBytesPlan
	_ = json.Unmarshal(b, &committed)
	if committed.Mutation != "known" || committed.Index == nil {
		t.Fatal("commit falsely unknown", e, string(b))
	}
	_ = os.Chmod(filepath.Join(f.f.root, txParent), 0755)
	ab := courseContextRead(t, f.f.root, *plan.AuthorityPath)
	check := []string{"generation-bundle", "historical-construction-bytes", "--check", "--locale", f.f.locale, "--content-package", f.f.pkg, "--authority", *plan.AuthorityPath, "--authority-sha256", courseContextSHA(ab)}
	var verified i18n.HistoricalBytesPlan
	if e = json.Unmarshal(cli(check...), &verified); e != nil || verified.Status != "VERIFIED_HISTORICAL_EVIDENCE" || verified.TransportRecoveryAuthorized || verified.Mutation != "none" {
		t.Fatal("independent historical check", e, verified)
	}
	var recorded i18n.HistoricalBytesPlan
	if e = json.Unmarshal(cli(resume...), &recorded); e != nil || recorded.Status != "RECORDED" {
		t.Fatal("sealed resume", e, recorded)
	}
	if _, e = os.Stat(filepath.Join(f.f.root, pending)); !os.IsNotExist(e) {
		t.Fatal("pending survived cleanup", e)
	}
	// A new call cannot reuse cleanup authority after the journal has gone.
	if _, e := authorizationDiagnosisOutput(t, func() error { return run(resume) }); e == nil {
		t.Fatal("resume without pending accepted")
	}
	courseContextWrite(t, "", source, old)
	courseContextWrite(t, "", evidence, eb)
	if _, e := authorizationDiagnosisOutput(t, func() error { return run(record) }); e == nil {
		t.Fatal("duplicate adoption accepted")
	}
	// Exact physical refs are mandatory even for encoding-only authority changes.
	for _, ref := range []i18n.RecoveryFileReference{*recorded.Authority, *recorded.Blob} {
		path := filepath.Join(f.f.root, ref.Path)
		original := courseContextRead(t, f.f.root, ref.Path)
		if e = os.Chmod(path, 0644); e != nil {
			t.Fatal(e)
		}
		courseContextWrite(t, f.f.root, ref.Path, append(append([]byte{}, original...), '\n'))
		_ = os.Chmod(path, 0444)
		b, e := authorizationDiagnosisOutput(t, func() error { return run(check) })
		if e == nil || !bytes.Contains(b, []byte(`"status": "BLOCKED"`)) {
			t.Fatal("tampered recorded material accepted", ref.Path, e, string(b))
		}
		_ = os.Chmod(path, 0644)
		courseContextWrite(t, f.f.root, ref.Path, original)
		_ = os.Chmod(path, 0444)
	}

	var after i18n.TransportRecoveryPlan
	if e = json.Unmarshal(cli(recoveryArgs...), &after); e != nil || after.Status != "READY" {
		t.Fatal("supplement did not unblock recovery", e, after)
	}
	if baseline.PlannedAuthority.DirectDrift[0].OldConstructionIdentity != after.PlannedAuthority.DirectDrift[0].OldConstructionIdentity {
		t.Fatal("supplement changed old typed construction")
	}
	for path, hash := range immutable {
		if courseContextSHA(courseContextRead(t, "", path)) != hash {
			t.Fatal("historical state changed", path)
		}
	}
	t.Log("TRUE_RUN_LIFECYCLE=PASS BEFORE=BLOCKED AFTER=READY ARCHIVE_RECEIPT_PROOF_RAW_VALIDATION_UNCHANGED=yes SEALED_RESUME=PASS EXTERNAL_SOURCE_NOT_REQUIRED_AFTER_SEAL=yes")
}
