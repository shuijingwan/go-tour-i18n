package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Single-Unit, opt-in diagnosis. Never invokes the 32-Unit readiness probe,
// current assessment, Bundle exporter, reconciliation publisher or Generation.
func TestJSONPredecessorRepresentationReadinessReadonly(t *testing.T) {
	out := os.Getenv("GO_TOUR_JSON_PREDECESSOR_READINESS_EVIDENCE")
	if out == "" {
		t.Skip("explicit single-Unit readonly evidence destination required")
	}
	if !strings.HasPrefix(filepath.Clean(out), "/tmp/") {
		t.Fatal("evidence must be in /tmp")
	}
	if e := os.MkdirAll(out, 0700); e != nil {
		t.Fatal(e)
	}
	resolvedOut, e := filepath.EvalSymlinks(out)
	if e != nil || resolvedOut != filepath.Clean(out) || !strings.HasPrefix(resolvedOut, "/tmp/") {
		t.Fatal("evidence root must not use symlinks or escape /tmp")
	}
	root := repositoryRoot(t)
	const id = "b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c"
	const sourcePath = "_content/doc/tutorial/json.md"
	refs := map[string]i18n.GlossaryArchiveReference{}
	read := func(path string) []byte {
		t.Helper()
		b, e := readRegular(root, path)
		if e != nil {
			t.Fatal(path, e)
		}
		refs[path] = i18n.GlossaryArchiveReference{Path: path, SHA256: digest(b)}
		return b
	}
	exact := func(ref i18n.GlossaryArchiveReference) []byte {
		t.Helper()
		b := read(ref.Path)
		if digest(b) != ref.SHA256 {
			t.Fatal("exact authority SHA mismatch", ref.Path)
		}
		return b
	}
	decode := func(b []byte, v any) {
		t.Helper()
		if e := StrictJSON(b, v); e != nil {
			t.Fatal(e)
		}
	}
	batchBytes := exact(i18n.GlossaryArchiveReference{Path: "data/retranslation-runs/zh-CN/chatgpt-zh-CN-064/manifest.json", SHA256: "280338157c956da37f1970f32f80b81ff404fcc1c2a8ec11720f68fb8844bb4b"})
	var batch i18n.RetranslationBatchManifest
	decode(batchBytes, &batch)
	var selected i18n.RetranslationBatchUnit
	for _, r := range batch.Units {
		if r.UnitID == id {
			selected = r
		}
	}
	if selected.UnitID == "" || selected.SourcePath != sourcePath || selected.RevisionFeedbackSource != i18n.CurrentValidationRevisionFeedback || selected.PreviousRating != "A" {
		t.Fatal("exact JSON revision authority missing")
	}
	reopenBytes := exact(i18n.GlossaryArchiveReference{Path: selected.PreviousReviewPath, SHA256: "338c96b02ab4b1f793a2d08b56e98597c1666ef820fd7d76b4788d6474f7fcbd"})
	var reopen i18n.QualityCheckSurfaceReopen
	decode(reopenBytes, &reopen)
	if reopen.AuthoritySource != i18n.CurrentValidationAuthority || reopen.ReopenID != selected.RevisionAuthorizationID || reopen.CurrentValidation == nil {
		t.Fatal("current-validation authority mismatch")
	}
	var snapshot i18n.QualityCheckSnapshotManifest
	decode(exact(reopen.CurrentValidation.Snapshot), &snapshot)
	var q i18n.QualityCheckSnapshotUnit
	for _, u := range snapshot.Units {
		if u.UnitID == id {
			q = u
		}
	}
	if q.UnitID != id || snapshot.SnapshotID != selected.PreviousSnapshotID || q.RepresentationReconciliation == nil {
		t.Fatal("recorded Snapshot selection mismatch")
	}
	var reconciliation i18n.RepresentationReconciliation
	decode(exact(*q.RepresentationReconciliation), &reconciliation)
	var machine i18n.RepresentationMachineUnit
	for _, u := range reconciliation.Units {
		if u.Replacement.CurrentUnitID == id {
			machine = u
		}
	}
	if machine.Replacement.SourcePath != sourcePath || machine.Candidate.Path != q.CandidatePath || machine.Candidate.SHA256 != q.CandidateSHA256 || machine.SemanticAuthority == nil || q.SelectedBatchID != machine.Predecessor.SelectedBatchID || q.Attempt != machine.Predecessor.Attempt {
		t.Fatal("exact machine predecessor selection missing")
	}
	var predecessor i18n.RetranslationBatchManifest
	decode(exact(machine.OriginalManifest), &predecessor)
	var record i18n.RetranslationBatchUnit
	for _, u := range predecessor.Units {
		if u.UnitID == machine.Predecessor.UnitID {
			record = u
		}
	}
	if record.UnitID == "" || predecessor.BatchID != q.SelectedBatchID {
		t.Fatal("Snapshot does not prove predecessor batch")
	}
	base := "data/retranslation-runs/" + batch.Locale + "/" + predecessor.BatchID + "/"
	oldInput := exact(i18n.GlossaryArchiveReference{Path: base + record.InputPath, SHA256: record.InputSHA256})
	var validation i18n.RetranslationValidation
	decode(exact(i18n.GlossaryArchiveReference{Path: machine.Predecessor.ValidationPath, SHA256: machine.Predecessor.ValidationSHA256}), &validation)
	if record.SourcePath != sourcePath || record.SourceSHA256 != q.SourceSHA256 || validation.SourceSHA256 != record.SourceSHA256 || validation.Attempt != q.Attempt || validation.Status != "passed" || validation.UnitID != record.UnitID || base+validation.RawResponsePath != machine.OriginalRaw.Path {
		t.Fatal("exact attempt/raw binding mismatch")
	}
	raw := exact(machine.OriginalRaw)
	candidate := exact(machine.Candidate)
	materialized := exact(machine.CurrentRaw)
	if !bytes.Equal(candidate, exact(i18n.GlossaryArchiveReference{Path: machine.Predecessor.CandidatePath, SHA256: machine.Predecessor.CandidateSHA256})) {
		t.Fatal("historical candidate bytes changed")
	}
	// Machine reconciliation evidence has its own record shape. It is not
	// a new passed RetranslationValidation or a model attempt.
	var machineValidation struct {
		Kind                                       string
		Replacement                                i18n.TranslationUnitReplacement
		Predecessor                                i18n.QualityCheckSnapshotUnit
		RawSHA256, CandidateSHA256, GlossarySHA256 string
	}
	decode(exact(machine.Validation), &machineValidation)
	if machineValidation.Kind != "go-learning/representation-reconciliation/v1" || machineValidation.Predecessor.Attempt != q.Attempt || machineValidation.Replacement.CurrentUnitID != id || machineValidation.RawSHA256 != digest(materialized) || machineValidation.CandidateSHA256 != digest(candidate) {
		t.Fatal("recorded machine proof mismatch")
	}
	exact(machine.OriginalResult)
	var oldSnapshot i18n.QualityCheckSnapshotManifest
	decode(exact(reconciliation.Snapshot), &oldSnapshot)
	foundOldSnapshot := false
	for _, u := range oldSnapshot.Units {
		if u.UnitID == record.UnitID && u.CandidateSHA256 == machine.Predecessor.CandidateSHA256 && u.SelectedBatchID == q.SelectedBatchID && u.Attempt == q.Attempt {
			foundOldSnapshot = true
		}
	}
	if !foundOldSnapshot {
		t.Fatal("original Snapshot selected predecessor mismatch")
	}
	source := exact(i18n.GlossaryArchiveReference{Path: sourcePath, SHA256: q.SourceSHA256})
	var frozen HistoricalPageRepresentations
	decode(exact(reconciliation.Options.HistoricalDocuments), &frozen)
	var oldDoc Document
	for _, d := range frozen.Documents {
		if d.Path == sourcePath && d.Units[0].ID == record.UnitID {
			oldDoc = d
		}
	}
	if oldDoc.Path == "" {
		t.Fatal("persisted historical mapping missing")
	}
	old := PageAdapter{oldDoc, source}
	ordinary, _, e := old.ProtectedInput(nil)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(bridgeEOF([]byte(ordinary)), oldInput) {
		t.Fatal("historical input mapping cannot be reproduced")
	}
	restored, e := old.Restore(string(raw))
	if e != nil || !bytes.Equal(bridgeEOF(restored), candidate) {
		t.Fatal("historical raw/candidate relation", e)
	}
	currentDoc, e := ParseWorkflowDocument(Source{Path: sourcePath, Kind: "page", Route: machine.Replacement.Route, SHA256: q.SourceSHA256}, source, machine.Replacement.StableIndex)
	if e != nil {
		t.Fatal(e)
	}
	current := PageAdapter{*currentDoc, source}
	input, _, e := current.ProtectedInput(nil)
	if e != nil {
		t.Fatal(e)
	}
	if currentDoc.Units[0].ID != id || digest(bridgeEOF([]byte(input))) != selected.InputSHA256 || selected.InputSHA256 != machine.Replacement.CurrentInputSHA256 {
		t.Fatal("current formal Unit/input delta")
	}
	decodeErr := error(nil)
	if _, decodeErr = current.Restore(string(materialized)); decodeErr == nil || !strings.Contains(decodeErr.Error(), "locked token exact-set changed: ⟦P245⟧") {
		t.Fatal("unresolved current P245 gate not reproduced", decodeErr)
	}
	var receipt i18n.RepresentationReviewReceipt
	decode(exact(*machine.SemanticAuthority), &receipt)
	files, e := i18n.ReadTransportBundle(exact(receipt.Bundle), 512, 64<<20)
	if e != nil {
		t.Fatal(e)
	}
	var review i18n.RepresentationReviewBundle
	decode(files["manifest.json"], &review)
	if e := i18n.ValidateTransportBundleInventory(files, review.Files, true); e != nil {
		t.Fatal(e)
	}
	savedIdentity := review.Identity
	review.Identity = ""
	encoded, e := json.Marshal(review)
	if e != nil || digest(encoded) != savedIdentity {
		t.Fatal("semantic Bundle identity mismatch")
	}
	review.Identity = savedIdentity
	if receipt.Decision.Decision != "APPROVE_RECONCILIATION" || receipt.Decision.CurrentUnitID != id || receipt.Decision.OldUnitID != record.UnitID || receipt.Decision.BundleIdentity != savedIdentity || receipt.Decision.BundleSHA256 != receipt.Bundle.SHA256 || !receipt.Decision.GenerationIndependent {
		t.Fatal("recorded semantic correspondence identity mismatch")
	}
	var decision i18n.RepresentationReviewDecision
	decode(exact(receipt.Evidence), &decision)
	left, _ := json.Marshal(decision)
	right, _ := json.Marshal(receipt.Decision)
	if !bytes.Equal(left, right) {
		t.Fatal("semantic decision bytes/receipt differ")
	}
	question := review.Question
	if !bytes.Equal(files["question.json"], mustProbeJSON(t, question)) {
		t.Fatal("question/manifest mismatch")
	}
	op, e := protectionPieces(oldDoc.Units[0])
	if e != nil {
		t.Fatal(e)
	}
	np, e := protectionPieces(currentDoc.Units[0])
	if e != nil {
		t.Fatal(e)
	}
	var old245, current245, introducedAtom protectedPiece
	var oldProse protectedPiece
	for _, p := range op {
		if p.token != nil && p.token.Token == "⟦P245⟧" {
			old245 = p
		}
		if p.token == nil && p.start == question.SourceStart && p.end == question.SourceEnd {
			oldProse = p
		}
	}
	for _, p := range np {
		if p.token != nil && p.token.Raw == "Marshal" && p.start >= question.SourceStart && p.end <= question.SourceEnd {
			introducedAtom = p
		}
		if p.token != nil && p.token.Token == "⟦P245⟧" {
			current245 = p
		}
	}
	if old245.token == nil || current245.token == nil || introducedAtom.token == nil || old245.token.Raw == current245.token.Raw || oldProse.text != question.SourceText {
		t.Fatal("exact source token inventories or persisted prose interval mismatch")
	}
	coverage := 0
	for _, p := range op {
		if p.token != nil && introducedAtom.start >= p.start && introducedAtom.end <= p.end {
			coverage++
		}
	}
	if coverage != 0 || introducedAtom.token.Binding != nil || introducedAtom.start < oldProse.start || introducedAtom.end > oldProse.end {
		t.Fatal("this site unexpectedly has an old atom/structural child binding")
	}
	if classifyProseTokenInRole("Marshal", "Custom ", " and Unmarshal", true) != technicalIdentifier {
		t.Fatal("current canonical role changed")
	}
	oldLeft, oldRight := false, false
	oldInputStart, oldInputEnd, encodedOffset := 0, 0, 0
	for _, p := range op {
		if p.token != nil && p.token.Token == question.LeftAnchor && p.end == oldProse.start {
			oldLeft = true
		}
		if p.token != nil && p.token.Token == question.RightAnchor && p.start == oldProse.end {
			oldRight = true
		}
		if p.token == nil && p.start == oldProse.start && p.end == oldProse.end {
			oldInputStart = encodedOffset
			oldInputEnd = encodedOffset + len(p.text)
		}
		encodedOffset += len(p.text)
	}
	if !oldLeft || !oldRight || !bytes.Equal(oldInput[oldInputStart:oldInputEnd], []byte(question.SourceText)) {
		t.Fatal("old input/anchor coordinates not source-owned")
	}
	sourceRebaseError := ""
	if _, err := current.RebaseRevisionPredecessor(old, raw); err == nil {
		t.Fatal("unexpected source-only rebase proof; diagnosis must be reconsidered")
	} else {
		sourceRebaseError = err.Error()
	}
	// Determine only the complete historical prose interval, using persisted
	// source-owned anchors. Never search target words or infer a target atom.
	if strings.Count(string(raw), question.LeftAnchor) != 1 || strings.Count(string(raw), question.RightAnchor) != 1 {
		t.Fatal("historical interval anchors not unique")
	}
	rawStart := bytes.Index(raw, []byte(question.LeftAnchor)) + len(question.LeftAnchor)
	rawEnd := bytes.Index(raw, []byte(question.RightAnchor))
	if rawStart > rawEnd || digest(raw[rawStart:rawEnd]) != digest([]byte(question.ReviewedTarget)) {
		t.Fatal("frozen whole-interval binding mismatch")
	}
	pairs := []string{}
	for _, p := range oldDoc.Units[0].Protected {
		pairs = append(pairs, p.Token, p.Raw)
	}
	replace := strings.NewReplacer(pairs...)
	expanded := []byte(replace.Replace(string(raw)))
	if !bytes.Equal(bridgeEOF(expanded), candidate) {
		t.Fatal("exact token expansion is not a valid coordinate projection")
	}
	candidateStart := len(replace.Replace(string(raw[:rawStart])))
	candidateEnd := len(replace.Replace(string(raw[:rawEnd])))
	if !bytes.Equal(candidate[candidateStart:candidateEnd], raw[rawStart:rawEnd]) {
		t.Fatal("candidate whole-interval coordinate mismatch")
	}
	sourceSites := []map[string]any{}
	for off := 0; off < len(source); {
		at := bytes.Index(source[off:], []byte("Marshal"))
		if at < 0 {
			break
		}
		at += off
		end := at + len("Marshal")
		prev, next := rune(0), rune(0)
		if at > 0 {
			prev, _ = utf8.DecodeLastRune(source[:at])
		}
		if end < len(source) {
			next, _ = utf8.DecodeRune(source[end:])
		}
		word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
		if !word(prev) && !word(next) {
			covered := false
			for _, p := range op {
				if p.token != nil && at >= p.start && end <= p.end {
					covered = true
					break
				}
			}
			sourceSites = append(sourceSites, map[string]any{"start": at, "end": end, "covered_by_old_source_atom": covered, "unresolved_site": at == introducedAtom.start && end == introducedAtom.end})
		}
		off = end
	}
	introduced := 0
	for _, s := range sourceSites {
		if s["unresolved_site"] == true {
			introduced++
		}
	}
	if introduced != 1 {
		t.Fatal("new source occurrence not uniquely identified")
	}
	var imported i18n.GenerationResultBundleManifest
	importRef := ReferenceFileForProbe(t, root, base+"generation-imports/initial-manifest.json")
	decode(exact(importRef), &imported)
	if imported.Locale != batch.Locale || imported.BatchID != predecessor.BatchID || imported.Attempt != q.Attempt {
		t.Fatal("selected import receipt mismatch")
	}
	importedRaw := 0
	for _, o := range imported.Outputs {
		if filepath.Base(o.BundlePath) == filepath.Base(record.InputPath) && o.SHA256 == digest(raw) && o.Size == int64(len(raw)) {
			importedRaw++
		}
	}
	if importedRaw != 1 {
		t.Fatal("import receipt does not bind exact raw")
	}
	archivePath := "data/generation-archives/" + batch.Locale + "/" + imported.GenerationBundleSHA256 + ".zip"
	archive, archiveErr := readRegular(root, archivePath)
	if archiveErr != nil && !os.IsNotExist(archiveErr) {
		t.Fatal(archiveErr)
	}
	if archiveErr == nil && digest(archive) != imported.GenerationBundleSHA256 {
		t.Fatal("import archive SHA mismatch")
	}
	// Only this Unit's impact record is consumed, as diagnostic labels, never
	// formal migration authority or a reason to accept target-language binding.
	impactBytes := exact(reconciliation.Options.Impact)
	var impact struct {
		Units []json.RawMessage `json:"units"`
	}
	if e := json.Unmarshal(impactBytes, &impact); e != nil {
		t.Fatal(e)
	}
	var impactRecord json.RawMessage
	for _, r := range impact.Units {
		var x struct {
			Current string `json:"current_unit_id"`
			Path    string `json:"source_path"`
		}
		if json.Unmarshal(r, &x) != nil {
			t.Fatal("impact JSON")
		}
		if x.Current == id && x.Path == sourcePath {
			impactRecord = r
		}
	}
	if len(impactRecord) == 0 {
		t.Fatal("exact impact record missing")
	}
	for path, ref := range refs {
		b, e := readRegular(root, path)
		if e != nil || digest(b) != ref.SHA256 {
			t.Fatal("diagnostic dependency changed", path)
		}
	}
	tokenInterval := func(b []byte, token string) any {
		if bytes.Count(b, []byte(token)) != 1 {
			return nil
		}
		at := bytes.Index(b, []byte(token))
		return []int{at, at + len(token)}
	}
	oldCoverageForErrorToken := []map[string]any{}
	for _, p := range op {
		if p.token != nil && current245.start >= p.start && current245.end <= p.end {
			oldCoverageForErrorToken = append(oldCoverageForErrorToken, map[string]any{"source_start": p.start, "source_end": p.end, "token": p.token})
		}
	}
	evidence := map[string]any{
		"report_type": "engineering-diagnostic-only", "formal_mutation": "none", "scanned_unit_count": 1,
		"HISTORICAL_UNIT_ID": record.UnitID, "CURRENT_UNIT_ID": id, "PREDECESSOR_BATCH": predecessor.BatchID, "ATTEMPT": q.Attempt,
		"OLD_INPUT_SHA256": record.InputSHA256, "HISTORICAL_RAW_SHA256": digest(raw), "MATERIALIZED_RAW_SHA256": digest(materialized), "CANDIDATE_SHA256": digest(candidate), "CURRENT_INPUT_SHA256": selected.InputSHA256, "SOURCE_SHA256": digest(source),
		"INTRODUCED_MACHINE_IDENTITY": "Marshal", "HISTORICAL_TOKEN": "P245", "SOURCE_OCCURRENCE_COUNT": len(sourceSites), "NEW_UNRESOLVED_HEADING_OCCURRENCE_COUNT": introduced, "SOURCE_OCCURRENCE_SCOPE": "full exact-identifier source inventory; machine-bindable count concerns the newly protected heading atom", "MACHINE_BINDABLE_OCCURRENCE_COUNT": coverage, "TARGET_TEXT_USED_FOR_BINDING": "no", "DETERMINISTIC_MAPPING_PROVABLE": "no",
		"source_occurrences": sourceSites, "historical_token": old245, "current_token": current245,
		"source_heading_interval": []int{question.SourceStart, question.SourceEnd}, "old_input_heading_interval": []int{oldInputStart, oldInputEnd},
		"old_raw_whole_heading_interval": []int{rawStart, rawEnd}, "candidate_whole_heading_interval": []int{candidateStart, candidateEnd}, "target_whole_interval_sha256": digest(raw[rawStart:rawEnd]), "target_atom_coordinates": nil,
		"old_protected_atom_coverage": coverage, "source_parser_role": "Markdown heading text; context Custom + Go identifier => technicalIdentifier; no old protected child or resolved Go AST target binding", "current_restore_error": decodeErr.Error(), "source_only_rebase_error": sourceRebaseError, "current_introduced_token": introducedAtom.token.Token,
		"old_input_P245_interval": tokenInterval(oldInput, "⟦P245⟧"), "old_raw_P245_interval": tokenInterval(raw, "⟦P245⟧"), "current_input_P245_interval": tokenInterval([]byte(input), "⟦P245⟧"), "materialized_raw_P245_interval": tokenInterval(materialized, "⟦P245⟧"), "materialized_raw_whole_heading_interval": nil, "materialized_interval_not_inferred": "different historical projection inventory; normal token numbers must not be used as its codec",
		"old_raw_P245_count": bytes.Count(raw, []byte("⟦P245⟧")), "materialized_raw_P245_count": bytes.Count(materialized, []byte("⟦P245⟧")),
		"semantic_authority": *machine.SemanticAuthority, "semantic_decision": receipt.Decision.Decision, "semantic_authority_scope": "whole frozen interval correspondence for historical reconciliation only; not atom identity mapping or current Generation response codec",
		"import_receipt": importRef, "import_archive_expected": i18n.GlossaryArchiveReference{Path: archivePath, SHA256: imported.GenerationBundleSHA256}, "import_archive_exists": archiveErr == nil, "import_archive_missing_is_not_reconstructed": true,
		"current_formal_unit_unchanged": q.UnitID == id, "current_source_unchanged": q.SourceSHA256 == selected.SourceSHA256, "current_formal_input_unchanged": machine.Replacement.CurrentInputSHA256 == selected.InputSHA256,
		"formal_impact_record": impactRecord, "observed_references": refs,
		"conclusion": "No source-only API atom to target-site mapping exists. Unique whole-interval anchors do not create a Marshal target child. Existing semantic approval cannot be expanded into a normal Generation codec. Need a separately specified formal migration capability; no new replacement delta is established.",
	}
	// Unexported pieces are explicitly rendered as coordinates, not empty JSON.
	describe := func(p protectedPiece) any {
		return map[string]any{"source_start": p.start, "source_end": p.end, "protection": p.token}
	}
	evidence["historical_token"] = describe(old245)
	evidence["error_token_current_normal"] = describe(current245)
	evidence["introduced_source_atom"] = describe(introducedAtom)
	evidence["error_token_covered_by_old_source_atoms"] = oldCoverageForErrorToken
	delete(evidence, "current_token")
	b, e := json.MarshalIndent(evidence, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(out, "proof.json"), append(b, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
	t.Logf("units=1 new_source_sites=%d total_source_identifier_sites=%d machine_bindable_sites=%d deterministic_mapping=no archive_exists=%t", introduced, len(sourceSites), coverage, archiveErr == nil)
}

func mustProbeJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func ReferenceFileForProbe(t *testing.T, root, path string) i18n.GlossaryArchiveReference {
	t.Helper()
	ref, e := ReferenceFile(root, path)
	if e != nil {
		t.Fatal(e)
	}
	return i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256}
}
