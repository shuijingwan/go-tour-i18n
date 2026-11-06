package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// Explicit read-only inspection of an already-exported batch. The newly built
// ZIP remains in memory: no re-export file, output, receipt or candidate lands.
func TestCommentRevisionTransportAcceptedBatchProof(t *testing.T) {
	output := os.Getenv("GO_TOUR_COMMENT_TRANSPORT_PROOF")
	if output == "" {
		t.Skip("explicit read-only proof output required")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	const locale, pkg, batch = "zh-CN", "learn-docs-v1", "chatgpt-zh-CN-051"
	const receiptPath = "data/quality-check-reopens/zh-CN/learn-docs-qc017-current-validation-code-comments.json"
	const manifestPath = "data/retranslation-runs/zh-CN/chatgpt-zh-CN-051/manifest.json"
	const previousPath = "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-017/manifest.json"
	const oldZIP = "/tmp/zh-CN-chatgpt-zh-CN-051-generation.zip"
	before := map[string]string{}
	read := func(path, sha string) []byte {
		t.Helper()
		full := path
		if !filepath.IsAbs(full) {
			full = filepath.Join(root, full)
		}
		b, err := os.ReadFile(full)
		if err != nil {
			t.Fatal(err)
		}
		actual := contentidentity.SHA256Hex(b)
		if sha != "" && actual != sha {
			t.Fatal("immutable input SHA mismatch", path, actual)
		}
		before[path] = actual
		return b
	}
	var r i18n.QualityCheckSurfaceReopen
	if err := json.Unmarshal(read(receiptPath, "c0568b10d39e91217dbaf689b9002311282f1aaeb8a24e5bdef2ffdbbd4f74ff"), &r); err != nil {
		t.Fatal(err)
	}
	var m i18n.RetranslationBatchManifest
	if err := json.Unmarshal(read(manifestPath, "f81fece6b28560ae99e03ff7b33b0192371dc012bcf9ff6adaccdcd442b0e2d8"), &m); err != nil {
		t.Fatal(err)
	}
	read(previousPath, "bfc5c76cab7966bdc8bf7eeb99d8da2fd06e447fc39ac26c194de22422da8cd2")
	old := read(oldZIP, "73dc3e9c7db76267baece62dd1b24a90ec1b5a161f2d32da8bfd5c68b1aa0460")
	if r.AuthoritySource != i18n.CurrentValidationAuthority || r.CurrentValidation.Package != pkg || r.UnitCount != 9 || m.UnitCount != 9 || len(m.Units) != 9 || m.BatchID != batch || m.Locale != locale {
		t.Fatal("wrong real authority scope")
	}
	for _, state := range []string{"raw-responses", "result.json", "candidates", "validation"} {
		if _, err := os.Lstat(filepath.Join(root, "data/retranslation-runs", locale, batch, state)); !os.IsNotExist(err) {
			t.Fatal("formal output already exists or cannot be inspected", state, err)
		}
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := sitecontent.AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	if err := c.UseTranslationPackage(pkg); err != nil {
		t.Fatal(err)
	}
	if len(c.TranslationScope) != 49 {
		t.Fatal("physical Page package boundary changed")
	}
	// Fail promptly on a raw-coordinate limitation before the full historical
	// authority reader runs. The exporter below still validates every receipt.
	for _, old := range r.Units {
		u, err := c.Unit(old.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		a, ok := u.Adapter.(i18n.CommentRevisionAdapter)
		if !ok {
			t.Fatal("missing Page body coordinates")
		}
		candidate := read(old.Snapshot.CandidatePath, old.Snapshot.CandidateSHA256)
		if _, err := a.CommentRevisionBodies(candidate); err != nil {
			t.Fatal(err)
		}
	}
	bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: locale, BatchID: batch})
	if err != nil {
		t.Fatal(err)
	}
	files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	oldFiles, err := i18n.ReadTransportBundle(old, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var oldManifest i18n.GenerationBundleManifest
	if err := json.Unmarshal(oldFiles["manifest.json"], &oldManifest); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(bundle, old) || oldManifest.InputIdentitySHA256 == bm.InputIdentitySHA256 {
		t.Fatal("obsolete transport remained canonical")
	}
	if bm.TaskKind != "translation-unit-revision" || len(bm.ExpectedOutputs) != 9 {
		t.Fatal("formal output scope changed")
	}
	pages := []map[string]any{}
	total, protected := 0, 0
	for i, record := range m.Units {
		if record.UnitID != r.Units[i].UnitID || record.UnitKind != i18n.UnitKindPage || record.PreviousRating != "A" || record.RevisionFeedbackSource != i18n.CurrentValidationRevisionFeedback || record.PreviousReviewSHA256 != before[receiptPath] {
			t.Fatal("finding lineage lost")
		}
		name := filepath.Base(record.InputPath)
		var p i18n.CommentRevisionTransport
		if err := json.Unmarshal(files["formal/comment-revision-transport/"+name+".json"], &p); err != nil {
			t.Fatal(err)
		}
		candidate := files["formal/candidates/"+name]
		if p.UnitID != record.UnitID || p.SourceSHA256 != record.SourceSHA256 || p.CandidateSHA256 != contentidentity.SHA256Hex(candidate) || p.Reopen.SHA256 != before[receiptPath] {
			t.Fatal("projection identity drift")
		}
		read(record.SourcePath, record.SourceSHA256)
		read(r.Units[i].Snapshot.CandidatePath, p.CandidateSHA256)
		if len(files["formal/historical-protected-candidates/"+name]) == 0 || !bytes.Equal(files["formal/protected-candidates/"+name], []byte(p.Text+"\n")) || len(p.Boundaries) != len(p.Slots)+1 {
			t.Fatal("missing protected revision representation")
		}
		var finding *i18n.CurrentValidationReopenFinding
		for j := range r.CurrentValidation.Findings {
			if r.CurrentValidation.Findings[j].UnitID == record.UnitID {
				finding = &r.CurrentValidation.Findings[j]
			}
		}
		if finding == nil || len(finding.CommentSlots) != len(p.Slots) {
			t.Fatal("hidden authorized slot")
		}
		wanted := map[string]i18n.CommentSlotComparison{}
		for _, row := range finding.CommentSlots {
			wanted[row.SlotID] = row
		}
		gotIDs := []string{}
		pos := 0
		for j, s := range p.Slots {
			row, ok := wanted[s.SlotID]
			if !ok || row.SemanticKind != "teaching-comment" || !row.ExactSourceCarry || row.ProtectedSemantics || s.Start < pos || s.End > len(candidate) {
				t.Fatal("unauthorized editable slot", s.SlotID)
			}
			left, right := i18n.CommentPayloadBounds(row.TargetComment)
			if string(candidate[s.Start:s.End]) != row.TargetComment[left:right] || p.Values[j] != string(candidate[pos:s.Start]) {
				t.Fatal("non-body bytes exposed")
			}
			start := strings.Index(p.Text, p.Boundaries[j]) + len(p.Boundaries[j])
			end := strings.Index(p.Text, p.Boundaries[j+1])
			if start > end || p.Text[start:end] != s.Text || strings.TrimSpace(s.Text) == "" {
				t.Fatal("authorized body lacks editable location")
			}
			editable := s.Text
			for _, token := range s.Tokens {
				editable = strings.ReplaceAll(editable, token, "")
			}
			if strings.TrimSpace(editable) == "" {
				t.Fatal("authorized language text still hidden behind tokens", s.SlotID)
			}
			delete(wanted, s.SlotID)
			gotIDs = append(gotIDs, s.SlotID)
			pos = s.End
		}
		if len(wanted) != 0 || p.Values[len(p.Values)-1] != string(candidate[pos:]) {
			t.Fatal("projection does not exactly cover candidate")
		}
		u, err := c.Unit(record.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		rows, err := i18n.ValidateTranslationUnitCommentCoverage(root, c, u, locale, candidate)
		if err == nil {
			t.Fatal("baseline no longer has the frozen gate failure")
		}
		visible := map[string]bool{}
		for _, id := range gotIDs {
			visible[id] = true
		}
		for _, row := range rows {
			if row.ProtectedSemantics {
				protected++
				if visible[row.SlotID] {
					t.Fatal("protected comment exposed")
				}
			}
			if visible[row.SlotID] && !row.ReviewRequired {
				t.Fatal("unaffected comment exposed")
			}
		}
		total += len(gotIDs)
		pages = append(pages, map[string]any{"unit_id": record.UnitID, "source_path": record.SourcePath, "source_sha256": record.SourceSHA256, "candidate_sha256": p.CandidateSHA256, "authorized_editable_slot_ids": gotIDs, "transport_sha256": contentidentity.SHA256Hex(files["formal/comment-revision-transport/"+name+".json"]), "remaining_content_protected": true})
	}
	if total != 41 {
		t.Fatal("real teaching scope changed", total)
	}
	after := map[string]string{}
	for path, sha := range before {
		full := path
		if !filepath.IsAbs(full) {
			full = filepath.Join(root, full)
		}
		b, err := os.ReadFile(full)
		if err != nil {
			t.Fatal(err)
		}
		after[path] = contentidentity.SHA256Hex(b)
		if after[path] != sha {
			t.Fatal("formal input mutated", path)
		}
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("immutable inputs changed")
	}
	evidence := map[string]any{"result": "PASS", "stage": "current-validation-comment-revision-transport-fix", "locale": locale, "content_package": pkg, "batch_id": batch, "formal_pages": 9, "authorized_comment_slots": total, "editable_authorized_comment_slots": total, "hidden_authorized_comment_slots": 0, "unauthorized_editable_comment_slots": 0, "protected_comment_exposure": 0, "protected_comments_checked": protected, "formal_representation_delta": 0, "formal_tu_boundary_unchanged": true, "generation_reconstructs_token_mapping": false, "same_batch_reexport_supported": true, "old_bundle_reusable": false, "old_bundle_identity": oldManifest.InputIdentitySHA256, "new_in_memory_bundle_identity": bm.InputIdentitySHA256, "new_in_memory_bundle_sha256": contentidentity.SHA256Hex(bundle), "bundle_written": false, "formal_language_artifact_mutation": "none", "immutable_input_references": before, "pages": pages}
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := f.Write(append(data, '\n'))
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatal(writeErr, closeErr)
	}
}
