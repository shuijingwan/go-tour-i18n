package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Engineering feasibility only. No exporter, importer, recorder, lifecycle
// operation, active validation bypass, or production legacy authority reader.
func TestLegacyGenerationProvenanceFeasibilityReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_LEGACY_PROVENANCE_PROBE") != "1" {
		t.Skip("explicit single-Unit feasibility probe required")
	}
	root := repositoryRoot(t)
	const currentID = "b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c"
	const oldID = "99824b13c26ffe219230ad0171e93d850003d16aa9f780c2f06c41bc46845cb3"
	const base = "data/retranslation-runs/zh-CN/chatgpt-zh-CN-031/"
	const receiptPath = base + "generation-imports/initial-manifest.json"
	const archiveSHA = "6972502acc0ab83f6a9382b9bb6330cca212d74db56e6af29e204e895e37d542"
	refs := map[string]string{}
	read := func(path, sha string) []byte {
		t.Helper()
		b, err := readRegular(root, path)
		if err != nil || (sha != "" && digest(b) != sha) {
			t.Fatal("surviving bytes mismatch", path, err)
		}
		refs[path] = digest(b)
		return b
	}
	exact := func(ref i18n.GlossaryArchiveReference) []byte { return read(ref.Path, ref.SHA256) }
	decode := func(b []byte, dst any) {
		t.Helper()
		if err := StrictJSON(b, dst); err != nil {
			t.Fatal(err)
		}
	}
	var batch i18n.RetranslationBatchManifest
	decode(read("data/retranslation-runs/zh-CN/chatgpt-zh-CN-064/manifest.json", "280338157c956da37f1970f32f80b81ff404fcc1c2a8ec11720f68fb8844bb4b"), &batch)
	var selected i18n.RetranslationBatchUnit
	for _, u := range batch.Units {
		if u.UnitID == currentID {
			selected = u
		}
	}
	var reopen i18n.QualityCheckSurfaceReopen
	decode(read(selected.PreviousReviewPath, selected.PreviousReviewSHA256), &reopen)
	if reopen.AuthoritySource != "current-validation" || reopen.CurrentValidation == nil || selected.RevisionAuthorizationID != reopen.ReopenID {
		t.Fatal("language authority mismatch")
	}
	var snapshot i18n.QualityCheckSnapshotManifest
	decode(exact(reopen.CurrentValidation.Snapshot), &snapshot)
	var q i18n.QualityCheckSnapshotUnit
	for _, u := range snapshot.Units {
		if u.UnitID == currentID {
			q = u
		}
	}
	if q.RepresentationReconciliation == nil || q.SelectedBatchID != "chatgpt-zh-CN-031" || q.Attempt != 1 {
		t.Fatal("Snapshot provenance mismatch")
	}
	var reconciliation i18n.RepresentationReconciliation
	decode(exact(*q.RepresentationReconciliation), &reconciliation)
	var machine i18n.RepresentationMachineUnit
	for _, u := range reconciliation.Units {
		if u.Replacement.CurrentUnitID == currentID {
			machine = u
		}
	}
	if machine.Predecessor.UnitID != oldID || machine.SemanticAuthority == nil || machine.Candidate.Path != q.CandidatePath || machine.Candidate.SHA256 != q.CandidateSHA256 || machine.Validation.Path != q.ValidationPath || machine.Validation.SHA256 != q.ValidationSHA256 {
		t.Fatal("machine/Snapshot mismatch")
	}
	var reviewReceipt i18n.RepresentationReviewReceipt
	decode(exact(*machine.SemanticAuthority), &reviewReceipt)
	files, err := i18n.ReadTransportBundle(exact(reviewReceipt.Bundle), 512, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var review i18n.RepresentationReviewBundle
	decode(files["manifest.json"], &review)
	if err = i18n.ValidateTransportBundleInventory(files, review.Files, true); err != nil {
		t.Fatal(err)
	}
	identity := review.Identity
	review.Identity = ""
	identityBytes, err := json.Marshal(review)
	if err != nil || digest(identityBytes) != identity || reviewReceipt.Decision.BundleIdentity != identity || reviewReceipt.Decision.BundleSHA256 != reviewReceipt.Bundle.SHA256 || reviewReceipt.Decision.Decision != "APPROVE_RECONCILIATION" || !reviewReceipt.Decision.GenerationIndependent || reviewReceipt.Predecessor != machine.Predecessor {
		t.Fatal("semantic receipt/Bundle identity mismatch", err)
	}
	var decision i18n.RepresentationReviewDecision
	decode(exact(reviewReceipt.Evidence), &decision)
	db, _ := json.Marshal(decision)
	rb, _ := json.Marshal(reviewReceipt.Decision)
	if !bytes.Equal(db, rb) {
		t.Fatal("recorded semantic decision differs")
	}
	// The review archive independently freezes these historical bytes. This
	// archive is never described as the missing Generation archive.
	for _, path := range []string{receiptPath, machine.OriginalManifest.Path, machine.OriginalResult.Path, machine.OriginalRaw.Path, machine.Predecessor.CandidatePath, machine.Predecessor.ValidationPath, reconciliation.Options.HistoricalDocuments.Path} {
		b := read(path, "")
		if !bytes.Equal(b, files["inputs/"+path]) {
			t.Fatal("historical review copy differs", path)
		}
	}
	var imported i18n.GenerationResultBundleManifest
	decode(read(receiptPath, "e4d022ff5d8b1079460539bd70394da9f0d456eb6f45ca70900bfc71a8d04d26"), &imported)
	if imported.SchemaVersion != 1 || imported.Kind != "go-tour-i18n/generation-result-bundle" || imported.Provider != "chatgpt" || imported.Model != "gpt-5.6-sol-high" || imported.Locale != "zh-CN" || imported.BatchID != q.SelectedBatchID || imported.Attempt != 1 || imported.TaskKind != "translation-unit-revision" || imported.GenerationBundleSHA256 != archiveSHA || imported.InputIdentitySHA256 != "fdc5e84fc20f2912a2f6a401b176820d3ec06291744684def6c7cc5a03ca62dc" || imported.GenerationManifest.BundlePath != "generation-manifest.json" || imported.GenerationManifest.SHA256 != "55db87dbffd719194483e82f53c7600cd4bcf7a48668f4b9a7cb310474db6b0d" || imported.GenerationManifest.Size != 14024 {
		t.Fatal("trusted import fact mismatch")
	}
	for _, p := range []string{"data/generation-archives/zh-CN/" + archiveSHA + ".zip", base + "generation-imports/import.pending"} {
		if _, err := os.Lstat(filepath.Join(root, p)); !os.IsNotExist(err) {
			t.Fatal("expected absence changed; reassess design", p, err)
		}
	}
	var predecessor i18n.RetranslationBatchManifest
	decode(exact(machine.OriginalManifest), &predecessor)
	var record i18n.RetranslationBatchUnit
	for _, u := range predecessor.Units {
		if u.UnitID == oldID {
			record = u
		}
	}
	var validation i18n.RetranslationValidation
	decode(read(machine.Predecessor.ValidationPath, machine.Predecessor.ValidationSHA256), &validation)
	var result i18n.RetranslationProcessResult
	decode(exact(machine.OriginalResult), &result)
	matchedResult := 0
	for _, u := range result.Units {
		if u.UnitID == oldID && u.Status == "passed" && base+u.CandidatePath == machine.Predecessor.CandidatePath && base+u.ValidationPath == machine.Predecessor.ValidationPath {
			matchedResult++
		}
	}
	if matchedResult != 1 || result.BatchID != imported.BatchID || result.Locale != imported.Locale || record.InputPath != "inputs/"+oldID+".article" || validation.UnitID != oldID || validation.BatchID != imported.BatchID || validation.Locale != imported.Locale || validation.Attempt != 1 || validation.Status != "passed" || validation.InputPath != record.InputPath || validation.RawResponsePath != "raw-responses/"+oldID+".article" || base+validation.RawResponsePath != machine.OriginalRaw.Path || base+validation.CandidatePath != machine.Predecessor.CandidatePath || validation.SourceSHA256 != record.SourceSHA256 {
		t.Fatal("process/input/install-path binding mismatch")
	}
	raw := exact(machine.OriginalRaw)
	seen := map[string]bool{}
	matchedOutput := 0
	for _, o := range imported.Outputs {
		if seen[o.BundlePath] {
			t.Fatal("duplicate output identity")
		}
		seen[o.BundlePath] = true
		if o.BundlePath == "outputs/"+oldID+".article" && o.SHA256 == digest(raw) && o.Size == int64(len(raw)) {
			matchedOutput++
		}
	}
	if matchedOutput != 1 || len(raw) != 8230 {
		t.Fatal("raw differs from imported output")
	}
	input := read(base+record.InputPath, record.InputSHA256)
	source := read(record.SourcePath, record.SourceSHA256)
	var frozen HistoricalPageRepresentations
	decode(exact(reconciliation.Options.HistoricalDocuments), &frozen)
	var old Document
	for _, d := range frozen.Documents {
		if d.Path == record.SourcePath && d.Units[0].ID == oldID {
			old = d
		}
	}
	a := PageAdapter{old, source}
	protected, tokens, err := a.ProtectedInput(nil)
	if err != nil || !bytes.Equal(bridgeEOF([]byte(protected)), input) || len(tokens) != record.ProtectedTokenCount {
		t.Fatal("persisted historical representation/input mismatch", err)
	}
	candidate := read(machine.Predecessor.CandidatePath, machine.Predecessor.CandidateSHA256)
	restored, err := a.Restore(string(raw))
	if err != nil || !bytes.Equal(bridgeEOF(restored), candidate) || !bytes.Equal(candidate, exact(machine.Candidate)) {
		t.Fatal("raw/candidate exact lineage mismatch", err)
	}
	current, err := ParseWorkflowDocument(Source{Path: record.SourcePath, Kind: "page", Route: machine.Replacement.Route, SHA256: record.SourceSHA256}, source, machine.Replacement.StableIndex)
	if err != nil {
		t.Fatal(err)
	}
	ca := PageAdapter{*current, source}
	currentInput, _, err := ca.ProtectedInput(nil)
	if err != nil || current.Units[0].ID != currentID || digest(bridgeEOF([]byte(currentInput))) != selected.InputSHA256 || selected.InputSHA256 != machine.Replacement.CurrentInputSHA256 {
		t.Fatal("current representation identity mismatch", err)
	}
	read("data/retranslation-runs/zh-CN/chatgpt-zh-CN-064/"+selected.InputPath, selected.InputSHA256)
	// Reproduce the recorded transition in its frozen event representation;
	// today's parser descriptor is only the separate current input check above.
	var event Document
	decode(review.Question.CurrentRepresentation, &event)
	ea := PageAdapter{event, source}
	eventInput, _, err := ea.ProtectedInput(nil)
	if err != nil || event.Units[0].ID != currentID || digest(bridgeEOF([]byte(eventInput))) != selected.InputSHA256 {
		t.Fatal("frozen event/current input identity mismatch", err)
	}
	nextRaw, nextCandidate, err := ea.ReviewedReconciliation(a, raw, review.Question)
	if err != nil || !bytes.Equal(nextRaw, exact(machine.CurrentRaw)) || !bytes.Equal(bridgeEOF(nextCandidate), candidate) {
		t.Fatal("recorded semantic materialization not reproducible", err)
	}
	exact(machine.Validation)
	// Recheck every consumed repository byte; print exact evidence for the
	// design matrix. This test does not assert complete lifecycle readiness.
	for path, sha := range refs {
		read(path, sha)
	}
	encoded, _ := json.Marshal(refs)
	t.Logf("surviving_refs=%s", encoded)
	t.Log("raw_size=8230 historical_restore=exact semantic_materialization=exact original_archive=absent legacy_authority_created=false active_validation_not_run=true")
}
