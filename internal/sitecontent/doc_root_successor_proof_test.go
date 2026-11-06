package sitecontent

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

// Opt-in engineering proof. Writes only source-owned engineering evidence;
// never records a receipt, exports a batch, or changes language artifacts.
func TestDocRootQCRepresentationSuccessorProof(t *testing.T) {
	out := os.Getenv("GO_TOUR_DOC_ROOT_SUCCESSOR_PROOF")
	if out == "" {
		t.Skip("explicit engineering evidence directory required")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	c, err := i18n.BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage("docs-entry-v1"); err != nil {
		t.Fatal(err)
	}
	frozen := HistoricalPageRepresentations{Documents: frozenEntryPages(t)}
	for _, d := range c.TranslationPackages {
		if d.ID == "docs-entry-v1" {
			frozen.Package, frozen.PackageSHA256 = d.ID, d.AuthoritySHA256
		}
	}
	write := func(relative string, value any) i18n.GlossaryArchiveReference {
		t.Helper()
		b, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		b = append(b, '\n')
		if err := writeCompatibilityProofFile(root, relative, b); err != nil {
			t.Fatal(err)
		}
		return i18n.GlossaryArchiveReference{Path: relative, SHA256: digest(b)}
	}
	const prefix = "docs/protected-representation-evidence/docs-entry-v1-doc-root-"
	historical := write(prefix+"historical-pages-20261022.json", frozen)
	r, err := resolvePageReplacements(root, c, historical)
	if err != nil {
		t.Fatal(err)
	}
	const oldID = "31fea3abbbfefa361e467a9ca5e90c93c07561d619a6e268cce60cf4002fb167"
	p := r.Units[oldID]
	if p.OldUnitID != oldID || p.OldUnitID == p.CurrentUnitID || p.OldInputSHA256 == p.CurrentInputSHA256 {
		t.Fatal("real representation successor required", p)
	}
	for _, old := range frozen.Documents[1:] {
		mapping := r.Units[old.Units[0].ID]
		if mapping.OldUnitID != mapping.CurrentUnitID || mapping.OldInputSHA256 != mapping.CurrentInputSHA256 {
			t.Fatal("unrelated A Page changed representation", mapping)
		}
	}
	const base = "data/quality-check-snapshots/zh-CN/docs-entry-v1-qc-001/"
	var s i18n.QualityCheckSnapshotManifest
	manifest, err := readRegular(root, base+"manifest.json")
	if err != nil || json.Unmarshal(manifest, &s) != nil {
		t.Fatal("Snapshot unreadable", err)
	}
	var results i18n.QualityCheckResults
	b, err := readRegular(root, base+"quality-check-results.json")
	if err != nil || digest(b) != "c7d1e57fadf6026c9d9833fdc4f728a0f58edc44cd546961dc643989169d1ad6" || json.Unmarshal(b, &results) != nil {
		t.Fatal("original QC result identity changed", err)
	}
	finding := ""
	for _, result := range results.Results {
		if result.UnitID == oldID && result.Rating == "B" {
			finding = result.Finding
		}
	}
	if finding == "" {
		t.Fatal("original exact B finding required")
	}
	var selected i18n.QualityCheckSnapshotUnit
	for _, u := range s.Units {
		if u.UnitID == oldID {
			selected = u
		}
	}
	vb, err := readRegular(root, selected.ValidationPath)
	var validation i18n.RetranslationValidation
	if err != nil || digest(vb) != selected.ValidationSHA256 || json.Unmarshal(vb, &validation) != nil || validation.Status != "passed" || validation.Attempt != selected.Attempt {
		t.Fatal("exact passed validation/attempt required", err)
	}
	rawPath := "data/retranslation-runs/zh-CN/" + selected.SelectedBatchID + "/" + validation.RawResponsePath
	raw, err := readRegular(root, rawPath)
	if err != nil {
		t.Fatal(err)
	}
	before, err := r.HistoricalCatalog.Unit(oldID)
	if err != nil {
		t.Fatal(err)
	}
	after, err := c.Unit(p.CurrentUnitID)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := after.Adapter.(i18n.ReplacementRevisionAdapter).RebaseRevisionPredecessor(before.Adapter, raw)
	if err != nil {
		t.Fatal("source-coordinate rebase", err)
	}
	oldCandidate, err := before.Adapter.Restore(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	newCandidate, err := after.Adapter.Restore(string(baseline))
	if err != nil || !bytes.Equal(oldCandidate, newCandidate) {
		t.Fatal("reviewed baseline bytes changed", err)
	}
	candidate, err := readRegular(root, selected.CandidatePath)
	if err != nil || digest(candidate) != selected.CandidateSHA256 || !bytes.Equal(candidate, []byte(strings.TrimRight(string(newCandidate), "\n")+"\n")) {
		t.Fatal("Snapshot candidate exact EOF-framed binding failed", err)
	}
	if strings.Count(string(baseline), "Documentation") != 2 {
		t.Fatal("exactly two formerly opaque label components must be editable")
	}
	if err := i18n.ValidateTranslationUnitCandidate(root, c, p.CurrentUnitID, "zh-CN", candidate); err != nil {
		t.Fatal("canonical validation rejected unchanged reviewed baseline", err)
	}
	for _, u := range s.Units {
		if u.UnitID == oldID {
			continue
		}
		mapping := r.Units[u.UnitID]
		accepted, err := readRegular(root, u.CandidatePath)
		if err != nil || digest(accepted) != u.CandidateSHA256 || mapping.CurrentUnitID != u.UnitID {
			t.Fatal("unaffected accepted candidate identity changed", err)
		}
		if err := i18n.ValidateTranslationUnitCandidate(root, c, u.UnitID, "zh-CN", accepted); err != nil {
			t.Fatal("unaffected A no longer passes current automatic validation", err)
		}
	}
	proof := write(prefix+"necessity-20261022.json", i18n.RepresentationNecessityProof{Reason: "protected-natural-language-unrepresentable", Units: []i18n.RepresentationNecessityUnit{{Replacement: p, Rating: "B", FindingSHA256: digest([]byte(finding))}}})
	o := i18n.QualityCheckSurfaceReopenOptions{AuthoritySource: "quality-check", Locale: "zh-CN", ReopenID: "docs-entry-qc001-doc-root-representation", PreviousSnapshotID: s.SnapshotID, PreviousRating: "B", Finding: finding, UnitIDs: []string{p.CurrentUnitID}, HistoricalDocuments: historical, RepresentationProof: proof}
	planned, receiptPath, err := i18n.InspectQualityCheckRepresentationReopen(root, c, o)
	if err != nil {
		t.Fatal("canonical readonly QC successor authorization", err)
	}
	if _, err := os.Lstat(filepath.Join(root, receiptPath)); !os.IsNotExist(err) {
		t.Fatal("readonly inspection recorded a receipt", err)
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"result": "PASS", "mutation": "repository-engineering-only", "formal_language_artifact_mutation": "none", "replacement": p, "historical_documents": historical, "engineering_proof": proof, "planned_receipt_path": receiptPath, "planned_receipt": planned, "snapshot_sha256": digest(manifest), "selected_predecessor": selected, "raw": Reference{rawPath, digest(raw)}, "byte_exact_reviewed_baseline": true, "editable_label_components": 2, "doc_install_representation_unchanged": true}
	data, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "preflight.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "protected-successor-baseline.txt"), baseline, 0644); err != nil {
		t.Fatal(err)
	}
	t.Log("READONLY_SUCCESSOR_PREFLIGHT=PASS", p.CurrentUnitID)
}

func writeCompatibilityProofFile(root, relative string, data []byte) error {
	path := filepath.Join(root, filepath.FromSlash(relative))
	if b, err := os.ReadFile(path); err == nil {
		if bytes.Equal(b, data) {
			return nil
		}
		return os.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func TestDocRootSuccessorCurrentInputReadiness(t *testing.T) {
	out := os.Getenv("GO_TOUR_DOC_ROOT_SUCCESSOR_PROOF")
	if out == "" {
		t.Skip("explicit engineering evidence directory required")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	c, err := i18n.BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	// One source-owned classifier correction must not silently migrate the
	// already finalized package. This checks identities, not language ratings.
	b, err := readRegular(root, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-021/manifest.json")
	var finalized i18n.QualityCheckSnapshotManifest
	if err != nil || json.Unmarshal(b, &finalized) != nil || len(finalized.Units) != 49 {
		t.Fatal("finalized package scope unreadable", err)
	}
	for _, u := range finalized.Units {
		if _, err := c.Unit(u.UnitID); err != nil {
			t.Fatal("unrelated finalized Unit representation drift", u.UnitID, err)
		}
	}
	if err := c.UseTranslationPackage("docs-entry-v1"); err != nil {
		t.Fatal(err)
	}
	var proof struct {
		Replacement i18n.TranslationUnitReplacement
	}
	b, err = os.ReadFile(filepath.Join(out, "preflight.json"))
	if err != nil || json.Unmarshal(b, &proof) != nil {
		t.Fatal(err)
	}
	u, err := c.Unit(proof.Replacement.CurrentUnitID)
	if err != nil {
		t.Fatal(err)
	}
	g, err := i18n.LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	input, _, err := u.Adapter.ProtectedInput(g)
	if err != nil || digest([]byte(strings.TrimRight(input, "\n")+"\n")) != proof.Replacement.CurrentInputSHA256 {
		t.Fatal("actual current input/corpus gate failed", err)
	}
	if err := i18n.RequireNewLanguageGlossaryReview(root, "zh-CN"); err != nil {
		t.Fatal("current Generation full Review gate failed", err)
	}
	report := map[string]any{"result": "PASS", "actual_protected_input_and_corpus_gate": "PASS", "current_full_review_gate": "PASS", "learn_docs_finalized_unit_ids_unchanged": 49, "formal_language_artifact_mutation": "none"}
	b, _ = json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "generation-input-readiness.json"), append(b, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
