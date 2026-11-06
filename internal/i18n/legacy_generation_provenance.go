package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const LegacyGenerationProvenanceSchema = "go-learning/legacy-generation-provenance/v1"
const LegacyGenerationProvenanceCapability = "narrow-per-unit-historical-response-provenance"

// Explicit selectors, never latest discovery or a caller-supplied mapping.
type LegacyGenerationProvenanceOptions struct {
	Locale               string                   `json:"locale"`
	ContentPackage       string                   `json:"content_package"`
	BatchID              string                   `json:"batch_id"`
	UnitID               string                   `json:"historical_unit_id"`
	CurrentUnitID        string                   `json:"current_unit_id"`
	Attempt              int                      `json:"attempt"`
	TaskKind             string                   `json:"task_kind"`
	ImportReceipt        GlossaryArchiveReference `json:"import_receipt"`
	OriginalBundleSHA256 string                   `json:"original_bundle_sha256"`
	PreviousSnapshotID   string                   `json:"previous_snapshot_id"`
	SnapshotSHA256       string                   `json:"previous_snapshot_sha256"`
}

type LegacyProvenanceDependency struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// This schema deliberately has no reconstructed ZIP, input inventory, target
// mapping, language rating, replay permission, or fabricated invocation fields.
type LegacyGenerationProvenance struct {
	Schema                     string                            `json:"schema"`
	Capability                 string                            `json:"capability"`
	ConsumerContract           string                            `json:"consumer_contract"`
	ProofAlgorithm             string                            `json:"proof_algorithm"`
	HistoricalFacts            []string                          `json:"historical_facts"`
	UnknownFacts               []string                          `json:"unknown_original_bundle_facts"`
	OriginalArchiveUnavailable bool                              `json:"original_archive_unavailable_at_record"`
	Options                    LegacyGenerationProvenanceOptions `json:"options"`
	PackageAuthority           string                            `json:"package_authority_sha256"`
	PackageScope               string                            `json:"package_scope_sha256"`
	Provider                   string                            `json:"provider"`
	Model                      string                            `json:"model"`
	InputIdentitySHA256        string                            `json:"recorded_input_identity_sha256"`
	GenerationManifest         TransportBundleFile               `json:"recorded_generation_manifest"`
	Output                     TransportBundleFile               `json:"recorded_output"`
	InstallPath                string                            `json:"install_path"`
	HistoricalManifest         GlossaryArchiveReference          `json:"historical_manifest"`
	HistoricalInput            GlossaryArchiveReference          `json:"historical_input"`
	Raw                        LegacyProvenanceDependency        `json:"raw"`
	Candidate                  GlossaryArchiveReference          `json:"candidate"`
	Validation                 GlossaryArchiveReference          `json:"validation"`
	Result                     GlossaryArchiveReference          `json:"result"`
	Snapshot                   GlossaryArchiveReference          `json:"snapshot"`
	HistoricalSnapshot         GlossaryArchiveReference          `json:"historical_snapshot"`
	Reconciliation             GlossaryArchiveReference          `json:"representation_reconciliation"`
	SemanticReceipt            GlossaryArchiveReference          `json:"semantic_receipt"`
	SemanticBundle             GlossaryArchiveReference          `json:"semantic_bundle"`
	SemanticDecision           GlossaryArchiveReference          `json:"semantic_decision"`
	HistoricalRepresentation   GlossaryArchiveReference          `json:"historical_representation"`
	CurrentSource              GlossaryArchiveReference          `json:"current_source"`
	CurrentInputSHA256         string                            `json:"current_input_sha256"`
	Dependencies               []LegacyProvenanceDependency      `json:"dependencies"`
	AuthorityID                string                            `json:"authority_id"`
	Identity                   string                            `json:"identity"`
}

type LegacyGenerationProvenancePlan struct {
	PlannedReceipt *LegacyGenerationProvenance `json:"planned_receipt"`
	PlannedPath    string                      `json:"planned_path"`
	Identity       string                      `json:"identity"`
	ReceiptPath    string                      `json:"receipt_path,omitempty"`
	ReceiptSHA256  string                      `json:"receipt_sha256,omitempty"`
	Recorded       bool                        `json:"recorded"`
	Mutation       string                      `json:"mutation"`
}

type VerifiedLegacyGenerationProvenance struct {
	Authority        *LegacyGenerationProvenance  `json:"authority"`
	ProvenanceSource string                       `json:"provenance_source"` // legacy-recorded or normal-archive
	Archive          *GlossaryArchiveReference    `json:"archive,omitempty"`
	Dependencies     []LegacyProvenanceDependency `json:"dependencies"`
}

func legacyProvenanceIdentity(r LegacyGenerationProvenance) string {
	r.Identity, r.AuthorityID = "", ""
	return sum(mustJSON(r))
}
func legacyProvenancePath(locale, id string) string {
	return "data/generation-provenance-authorities/" + locale + "/" + id + ".json"
}
func legacyProvenanceGuard(o LegacyGenerationProvenanceOptions) string {
	// One invocation/Unit cannot race two different content-addressed records.
	return "data/generation-provenance-authorities/" + o.Locale + "/." + sum([]byte(o.BatchID+"\x00"+o.UnitID+fmt.Sprint(o.Attempt))) + ".pending"
}

func validateLegacyProvenanceOptions(o LegacyGenerationProvenanceOptions) error {
	if ValidateLocaleName(o.Locale) != nil || validateBatchID(o.BatchID) != nil || !validSHA256(o.UnitID) || !validSHA256(o.CurrentUnitID) || o.ContentPackage == "" || validateSnapshotID(o.PreviousSnapshotID) != nil || !validSHA256(o.SnapshotSHA256) || !validSHA256(o.ImportReceipt.SHA256) || !validSHA256(o.OriginalBundleSHA256) || o.Attempt != 1 || (o.TaskKind != "translation-unit-initial" && o.TaskKind != "translation-unit-revision") {
		return fmt.Errorf("legacy provenance requires exact ordinary Page attempt-1 identifiers and hashes")
	}
	want := "data/retranslation-runs/" + o.Locale + "/" + o.BatchID + "/generation-imports/initial-manifest.json"
	if o.ImportReceipt.Path != want {
		return fmt.Errorf("legacy import receipt must use canonical repository-relative path")
	}
	return nil
}

// Independently reconstruct the historical graph. No existing Generation or
// current-validation consumer calls this function or gains an archive bypass.
func buildLegacyGenerationProvenance(root string, c *Catalog, o LegacyGenerationProvenanceOptions) (*LegacyGenerationProvenance, error) {
	if err := validateLegacyProvenanceOptions(o); err != nil {
		return nil, err
	}
	scoped, err := SelectCurrentValidationReopenPackage(c, o.ContentPackage)
	if err != nil {
		return nil, err
	}
	c = scoped
	pkg, authority, scope, ordered, err := translationAuthorizationScope(c)
	if err != nil || pkg != o.ContentPackage {
		return nil, fmt.Errorf("legacy complete package mismatch: %v", err)
	}
	refs := map[string]LegacyProvenanceDependency{}
	read := func(ref GlossaryArchiveReference) ([]byte, error) {
		if !validSHA256(ref.SHA256) {
			return nil, fmt.Errorf("legacy dependency requires exact SHA: %s", ref.Path)
		}
		b, e := readCompatibilityFile(root, ref.Path)
		if e != nil {
			return nil, e
		}
		if sum(b) != ref.SHA256 {
			return nil, fmt.Errorf("legacy dependency hash mismatch: %s", ref.Path)
		}
		d := LegacyProvenanceDependency{ref.Path, sum(b), int64(len(b))}
		if old, ok := refs[ref.Path]; ok && old != d {
			return nil, fmt.Errorf("legacy dependency changed: %s", ref.Path)
		}
		refs[ref.Path] = d
		return b, nil
	}
	bind := func(path string) (GlossaryArchiveReference, error) {
		b, e := readCompatibilityFile(root, path)
		if e != nil {
			return GlossaryArchiveReference{}, e
		}
		ref := GlossaryArchiveReference{path, sum(b)}
		_, e = read(ref)
		return ref, e
	}
	for _, unit := range ordered {
		ref, e := bind(unit.SourcePath)
		if e != nil || ref.SHA256 != unit.SourceSHA256 {
			return nil, fmt.Errorf("legacy complete package source drift: %s: %v", unit.SourcePath, e)
		}
	}
	receiptBytes, err := read(o.ImportReceipt)
	if err != nil {
		return nil, err
	}
	var imported GenerationResultBundleManifest
	if err = contentidentity.StrictJSON(receiptBytes, &imported); err != nil {
		return nil, err
	}
	if imported.SchemaVersion != GenerationResultBundleSchemaVersion || imported.Kind != "go-tour-i18n/generation-result-bundle" || imported.Locale != o.Locale || imported.BatchID != o.BatchID || imported.Attempt != o.Attempt || imported.TaskKind != o.TaskKind || imported.GenerationBundleSHA256 != o.OriginalBundleSHA256 || !validSHA256(imported.InputIdentitySHA256) || imported.GenerationManifest.BundlePath != "generation-manifest.json" || !validSHA256(imported.GenerationManifest.SHA256) || imported.GenerationManifest.Size <= 0 || imported.GenerationManifest.RepositoryPath != "" {
		return nil, fmt.Errorf("legacy imported invocation identity mismatch")
	}
	if err = requireFormalGenerationIdentity(imported.Provider, imported.Model, imported.BatchID); err != nil {
		return nil, err
	}
	if err = ensureNoGenerationImportPending(filepath.Join(root, "data/retranslation-runs", o.Locale, o.BatchID)); err != nil {
		return nil, err
	}
	snapshotRef := GlossaryArchiveReference{"data/quality-check-snapshots/" + o.Locale + "/" + o.PreviousSnapshotID + "/manifest.json", o.SnapshotSHA256}
	sb, err := read(snapshotRef)
	if err != nil {
		return nil, err
	}
	var snapshot QualityCheckSnapshotManifest
	if err = contentidentity.StrictJSON(sb, &snapshot); err != nil {
		return nil, err
	}
	if snapshot.Locale != o.Locale || snapshot.SnapshotID != o.PreviousSnapshotID || snapshot.SchemaVersion != QualityCheckSnapshotSchemaVersion {
		return nil, fmt.Errorf("legacy Snapshot identity mismatch")
	}
	var q QualityCheckSnapshotUnit
	n := 0
	for _, u := range snapshot.Units {
		if u.UnitID == o.CurrentUnitID {
			q = u
			n++
		}
	}
	if n != 1 || q.UnitKind != UnitKindPage || q.SelectedBatchID != o.BatchID || q.Attempt != o.Attempt || q.RepresentationReconciliation == nil {
		return nil, fmt.Errorf("legacy exact Snapshot/Unit/attempt selection required")
	}
	if _, err = read(*q.RepresentationReconciliation); err != nil {
		return nil, err
	}
	// Existing historical machine verifier retains original QC, glossary world,
	// replacement uniqueness, semantic decision, and byte-preserving transition.
	evidence, selection, err := machineSnapshotEvidenceInWorld(root, c, o.Locale, q, nil, true)
	if err != nil {
		return nil, err
	}
	m := selection.unit
	if m.Predecessor.UnitID != o.UnitID || m.Predecessor.Attempt != o.Attempt || m.Predecessor.SelectedBatchID != o.BatchID || m.SemanticAuthority == nil || evidence.record.GenerationOrigin != nil || evidence.record.RepresentationBridge != nil {
		return nil, fmt.Errorf("unsupported legacy historical response scope/codec")
	}
	for _, ref := range machineAuthorityReferences(selection) {
		if _, err = read(ref); err != nil {
			return nil, err
		}
	}
	var semantic RepresentationReviewReceipt
	b, err := read(*m.SemanticAuthority)
	if err != nil {
		return nil, err
	}
	if err = contentidentity.StrictJSON(b, &semantic); err != nil {
		return nil, err
	}
	if semantic.Schema != representationReviewSchema || semantic.Identity != reviewReceiptIdentity(semantic) || semantic.Predecessor != m.Predecessor {
		return nil, fmt.Errorf("legacy semantic receipt identity mismatch")
	}
	rb, err := read(semantic.Bundle)
	if err != nil {
		return nil, err
	}
	review, files, err := readLegacySemanticBundle(rb)
	if err != nil {
		return nil, err
	}
	if semantic.Decision.BundleIdentity != review.Identity || semantic.Decision.BundleSHA256 != sum(rb) || semantic.Decision.Decision != "APPROVE_RECONCILIATION" || !semantic.Decision.GenerationIndependent || semantic.Decision.OldUnitID != o.UnitID || semantic.Decision.CurrentUnitID != o.CurrentUnitID {
		return nil, fmt.Errorf("legacy semantic scope mismatch")
	}
	if _, err = read(semantic.Evidence); err != nil {
		return nil, err
	}
	// Check every persisted historical review copy, never today's glossary in
	// place of the review-time bytes. Store the archived glossary reference.
	for _, f := range review.Files {
		if f.RepositoryPath == "" {
			continue
		}
		path := f.RepositoryPath
		if path == glossaryReviewGlossaryPath(o.Locale) {
			path, err = glossaryArchivePath(o.Locale, f.SHA256)
			if err != nil {
				return nil, err
			}
		}
		actual, e := read(GlossaryArchiveReference{path, f.SHA256})
		if e != nil {
			return nil, e
		}
		if int64(len(actual)) != f.Size || !bytes.Equal(actual, files[f.BundlePath]) {
			return nil, fmt.Errorf("legacy historical review copy mismatch: %s", path)
		}
	}
	if !bytes.Equal(receiptBytes, files["inputs/"+o.ImportReceipt.Path]) {
		return nil, fmt.Errorf("legacy trusted import not frozen by recorded review")
	}
	base := "data/retranslation-runs/" + o.Locale + "/" + o.BatchID + "/"
	manifestBytes, err := read(m.OriginalManifest)
	if err != nil {
		return nil, err
	}
	manifest, err := readRetranslationProcessManifest(filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(base, "/"))), o.Locale, o.BatchID)
	if err != nil {
		return nil, err
	}
	if m.OriginalManifest.Path != base+"manifest.json" || !bytes.Equal(manifestBytes, files["inputs/"+base+"manifest.json"]) {
		return nil, fmt.Errorf("legacy batch manifest lineage mismatch")
	}
	record := evidence.record
	if record.UnitID != o.UnitID || record.InputPath != "inputs/"+o.UnitID+".article" || (o.TaskKind == "translation-unit-revision") != (record.RevisionFeedbackSource != "") {
		return nil, fmt.Errorf("legacy batch task/input mismatch")
	}
	historicalInput := GlossaryArchiveReference{base + record.InputPath, record.InputSHA256}
	input, err := read(historicalInput)
	if err != nil {
		return nil, err
	}
	v := evidence.validation
	install := "raw-responses/" + o.UnitID + ".article"
	if v.RawResponsePath != install || v.Attempt != 1 || v.Status != "passed" || v.InputPath != record.InputPath || m.OriginalRaw.Path != base+install {
		return nil, fmt.Errorf("legacy raw install lineage mismatch")
	}
	raw, err := read(m.OriginalRaw)
	if err != nil {
		return nil, err
	}
	if err = validateGenerationOutputBytes(raw); err != nil {
		return nil, err
	}
	var output TransportBundleFile
	seen := map[string]bool{}
	matches := 0
	for _, out := range imported.Outputs {
		if validateGenerationInstallPath(out.BundlePath) != nil || !strings.HasPrefix(out.BundlePath, "outputs/") || seen[out.BundlePath] || !validSHA256(out.SHA256) || out.Size <= 0 || out.RepositoryPath != "" {
			return nil, fmt.Errorf("legacy invalid/duplicate imported output")
		}
		seen[out.BundlePath] = true
		if out.BundlePath == "outputs/"+o.UnitID+".article" {
			output = out
			matches++
		}
	}
	if matches != 1 || output.SHA256 != sum(raw) || output.Size != int64(len(raw)) || len(imported.Outputs) != manifest.UnitCount {
		return nil, fmt.Errorf("legacy imported raw hash/size/output mismatch")
	}
	// The surviving source-owned historical adapter, never Protect(candidate),
	// must reproduce the saved input and restore the exact accepted candidate.
	old, err := selection.ctx.resolution.HistoricalCatalog.Unit(o.UnitID)
	if err != nil {
		return nil, err
	}
	p, err := prepareTranslationUnitInput(old, selection.ctx.glossary)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(p.Text)), input) || len(p.Tokens) != record.ProtectedTokenCount {
		return nil, fmt.Errorf("legacy historical protected input mismatch")
	}
	restored, failed := p.restore(string(raw))
	if len(failed) != 0 || !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(restored)), evidence.candidate) {
		return nil, fmt.Errorf("legacy exact historical restore failed")
	}
	current, err := c.Unit(o.CurrentUnitID)
	if err != nil {
		return nil, err
	}
	source, err := bind(current.SourcePath)
	if err != nil {
		return nil, err
	}
	if source.SHA256 != q.SourceSHA256 || current.SourceSHA256 != source.SHA256 || !bytes.Equal(current.Source, old.Source) {
		return nil, fmt.Errorf("legacy current source mismatch")
	}
	cp, err := prepareTranslationUnitInput(current, &Glossary{})
	if err != nil {
		return nil, err
	}
	currentInput := sum(canonicalizeRetranslationArtifactEOF([]byte(cp.Text)))
	if currentInput != m.Replacement.CurrentInputSHA256 {
		return nil, fmt.Errorf("legacy current input mismatch")
	}
	// Freeze exact QC Snapshot dependencies as well as their result files. These
	// are proof dependencies, not an expansion of this authority's Unit scope.
	for _, qc := range selection.receipt.QCInputs {
		if _, err = read(GlossaryArchiveReference{qc.Path, qc.SHA256}); err != nil {
			return nil, err
		}
		sp := "data/quality-check-snapshots/" + o.Locale + "/" + qc.SnapshotID + "/manifest.json"
		ref, e := bind(sp)
		if e != nil {
			return nil, e
		}
		data, e := read(ref)
		if e != nil {
			return nil, e
		}
		var s QualityCheckSnapshotManifest
		if e = contentidentity.StrictJSON(data, &s); e != nil {
			return nil, e
		}
		for _, u := range s.Units {
			for _, r := range []GlossaryArchiveReference{{u.CandidatePath, u.CandidateSHA256}, {u.ValidationPath, u.ValidationSHA256}} {
				if _, e = read(r); e != nil {
					return nil, e
				}
			}
			if u.RepresentationReconciliation != nil {
				// This v1 proof supports ordinary historical QC inputs only.
				// Nested representation graphs need an explicit future contract.
				return nil, fmt.Errorf("legacy QC proof requires unsupported nested reconciliation")
			}
			ubase := "data/retranslation-runs/" + o.Locale + "/" + u.SelectedBatchID + "/"
			um, e := readRetranslationProcessManifest(filepath.Join(root, filepath.FromSlash(ubase)), o.Locale, u.SelectedBatchID)
			if e != nil {
				return nil, e
			}
			if _, e = bind(ubase + "manifest.json"); e != nil {
				return nil, e
			}
			if _, e = bind(ubase + "result.json"); e != nil {
				return nil, e
			}
			for _, ur := range um.Units {
				if ur.UnitID != u.UnitID {
					continue
				}
				if _, e = read(GlossaryArchiveReference{ubase + ur.InputPath, ur.InputSHA256}); e != nil {
					return nil, e
				}
				vb, e := read(GlossaryArchiveReference{u.ValidationPath, u.ValidationSHA256})
				if e != nil {
					return nil, e
				}
				var uv RetranslationValidation
				if e = contentidentity.StrictJSON(vb, &uv); e != nil {
					return nil, e
				}
				if _, e = bind(ubase + uv.RawResponsePath); e != nil {
					return nil, e
				}
			}
		}
		archived, e := glossaryArchivePath(o.Locale, s.GlossarySHA256)
		if e != nil {
			return nil, e
		}
		if _, e = read(GlossaryArchiveReference{archived, s.GlossarySHA256}); e != nil {
			return nil, e
		}
	}
	// The machine reader also validates archived accepted glossary bytes.
	gp, err := glossaryArchivePath(o.Locale, selection.receipt.GlossarySHA256)
	if err != nil {
		return nil, err
	}
	if _, err = read(GlossaryArchiveReference{gp, selection.receipt.GlossarySHA256}); err != nil {
		return nil, err
	}
	candidate := GlossaryArchiveReference{m.Predecessor.CandidatePath, m.Predecessor.CandidateSHA256}
	validation := GlossaryArchiveReference{m.Predecessor.ValidationPath, m.Predecessor.ValidationSHA256}
	r := &LegacyGenerationProvenance{
		Schema: LegacyGenerationProvenanceSchema, Capability: LegacyGenerationProvenanceCapability,
		ConsumerContract: "historical-response/v1", ProofAlgorithm: "ordinary-page-surviving-import/v1",
		HistoricalFacts:            []string{"import-provider-model-task-batch-attempt", "exact-imported-raw", "formal-source-input", "accepted-candidate-lineage", "recorded-reconciliation"},
		UnknownFacts:               []string{"original-generation-manifest-bytes", "original-member-inventory", "original-authority-context-bytes", "original-glossary-bytes", "original-input-member-equality", "original-editable-predecessor-baseline"},
		OriginalArchiveUnavailable: true, Options: o, PackageAuthority: authority, PackageScope: scope,
		Provider: imported.Provider, Model: imported.Model, InputIdentitySHA256: imported.InputIdentitySHA256,
		GenerationManifest: imported.GenerationManifest, Output: output, InstallPath: install,
		HistoricalManifest: m.OriginalManifest, HistoricalInput: historicalInput, Raw: refs[m.OriginalRaw.Path],
		Candidate: candidate, Validation: validation, Result: m.OriginalResult,
		Snapshot: snapshotRef, HistoricalSnapshot: selection.receipt.Snapshot, Reconciliation: *q.RepresentationReconciliation,
		SemanticReceipt: *m.SemanticAuthority, SemanticBundle: semantic.Bundle, SemanticDecision: semantic.Evidence,
		HistoricalRepresentation: selection.receipt.Options.HistoricalDocuments,
		CurrentSource:            source, CurrentInputSHA256: currentInput,
	}
	for _, ref := range []GlossaryArchiveReference{candidate, validation} {
		if _, err = read(ref); err != nil {
			return nil, err
		}
	}
	for _, d := range refs {
		r.Dependencies = append(r.Dependencies, d)
	}
	sort.Slice(r.Dependencies, func(i, j int) bool { return r.Dependencies[i].Path < r.Dependencies[j].Path })
	r.Identity = legacyProvenanceIdentity(*r)
	r.AuthorityID = r.Identity
	if err = verifyLegacyDependencies(root, r); err != nil {
		return nil, err
	}
	return r, nil
}

func readLegacySemanticBundle(b []byte) (*RepresentationReviewBundle, map[string][]byte, error) {
	files, err := ReadTransportBundle(b, 512, 64<<20)
	if err != nil {
		return nil, nil, err
	}
	var m RepresentationReviewBundle
	if err = contentidentity.StrictJSON(files["manifest.json"], &m); err != nil {
		return nil, nil, err
	}
	if err = ValidateTransportBundleInventory(files, m.Files, true); err != nil {
		return nil, nil, err
	}
	id := m.Identity
	m.Identity = ""
	if id != sum(mustJSON(m)) {
		return nil, nil, fmt.Errorf("legacy semantic Bundle identity mismatch")
	}
	m.Identity = id
	return &m, files, nil
}
func verifyLegacyDependencies(root string, r *LegacyGenerationProvenance) error {
	for _, d := range r.Dependencies {
		b, e := readCompatibilityFile(root, d.Path)
		if e != nil || sum(b) != d.SHA256 || int64(len(b)) != d.Size {
			return fmt.Errorf("legacy dependency drift: %s: %v", d.Path, e)
		}
	}
	return nil
}

// Return stronger evidence when available. The receipt's historical absence
// observation never masks an archive that has since appeared.
func compareLegacyArchive(root string, r *LegacyGenerationProvenance) (*GlossaryArchiveReference, error) {
	path := importedGenerationArchive(r.Options.Locale, r.Options.OriginalBundleSHA256)
	b, err := readCompatibilityFile(root, path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if sum(b) != r.Options.OriginalBundleSHA256 {
		return nil, fmt.Errorf("legacy canonical archive SHA mismatch")
	}
	m, files, err := readGenerationBundle(b)
	if err != nil {
		return nil, err
	}
	receiptBytes, err := readCompatibilityFile(root, r.Options.ImportReceipt.Path)
	if err != nil {
		return nil, err
	}
	if sum(receiptBytes) != r.Options.ImportReceipt.SHA256 {
		return nil, fmt.Errorf("legacy/archive trusted import receipt drift")
	}
	var imported GenerationResultBundleManifest
	if err = contentidentity.StrictJSON(receiptBytes, &imported); err != nil {
		return nil, err
	}
	if err = checkImportedGenerationIdentity(imported, m, b, files); err != nil {
		return nil, err
	}
	if imported.InputIdentitySHA256 != r.InputIdentitySHA256 || !reflect.DeepEqual(imported.GenerationManifest, r.GenerationManifest) {
		return nil, fmt.Errorf("legacy/archive recorded invocation conflict")
	}
	outputs := 0
	for _, out := range imported.Outputs {
		if out == r.Output {
			outputs++
		}
	}
	if outputs != 1 {
		return nil, fmt.Errorf("legacy/archive recorded output conflict")
	}
	matched := 0
	for _, out := range m.ExpectedOutputs {
		if out.UnitID == r.Options.UnitID && out.BundlePath == r.Output.BundlePath && out.InstallPath == r.InstallPath {
			matched++
		}
	}
	if matched != 1 {
		return nil, fmt.Errorf("legacy/archive selected output conflict")
	}
	for _, x := range []struct {
		member string
		ref    GlossaryArchiveReference
	}{{"formal/manifest.json", r.HistoricalManifest}, {"formal/" + strings.TrimPrefix(r.HistoricalInput.Path, "data/retranslation-runs/"+r.Options.Locale+"/"+r.Options.BatchID+"/"), r.HistoricalInput}} {
		actual, e := readCompatibilityFile(root, x.ref.Path)
		if e != nil || sum(actual) != x.ref.SHA256 || !bytes.Equal(actual, files[x.member]) {
			return nil, fmt.Errorf("legacy/archive surviving input conflict: %s", x.member)
		}
	}
	// Any original member which claims a repository dependency covered by this
	// receipt must agree. No claim is added for unknown original docs/glossary.
	deps := map[string]LegacyProvenanceDependency{}
	for _, d := range r.Dependencies {
		deps[d.Path] = d
	}
	for _, f := range m.Files {
		if d, ok := deps[f.RepositoryPath]; ok && (f.SHA256 != d.SHA256 || f.Size != d.Size) {
			return nil, fmt.Errorf("legacy/archive covered fact conflict: %s", f.RepositoryPath)
		}
	}
	return &GlossaryArchiveReference{path, sum(b)}, nil
}

// Fresh typed reader, with explicit requested scope and exact receipt SHA.
// It intentionally does not expose a representation decoder or bypass token gates.
func VerifyLegacyGenerationProvenance(root string, c *Catalog, ref GlossaryArchiveReference, want LegacyGenerationProvenanceOptions) (*VerifiedLegacyGenerationProvenance, error) {
	if err := validateLegacyProvenanceOptions(want); err != nil {
		return nil, err
	}
	if err := legacyNoPending(root, want); err != nil {
		return nil, err
	}
	b, err := readCompatibilityFile(root, ref.Path)
	if err != nil {
		return nil, err
	}
	if !validSHA256(ref.SHA256) || sum(b) != ref.SHA256 {
		return nil, fmt.Errorf("legacy authority receipt SHA mismatch")
	}
	var r LegacyGenerationProvenance
	if err = contentidentity.StrictJSON(b, &r); err != nil {
		return nil, err
	}
	if r.Options != want || r.Identity != legacyProvenanceIdentity(r) || r.AuthorityID != r.Identity || ref.Path != legacyProvenancePath(want.Locale, r.Identity) || !bytes.Equal(b, append(mustJSON(r), '\n')) {
		return nil, fmt.Errorf("legacy authority path/identity/scope mismatch")
	}
	// Check the stronger evidence first, then reconstruct every surviving fact.
	archive, err := compareLegacyArchive(root, &r)
	if err != nil {
		return nil, err
	}
	fresh, err := buildLegacyGenerationProvenance(root, c, want)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(fresh, &r) {
		return nil, fmt.Errorf("legacy recorded authority graph changed")
	}
	if err = legacyConflicts(root, &r, false); err != nil {
		return nil, err
	}
	if err = verifyLegacyDependencies(root, &r); err != nil {
		return nil, err
	}
	archive, err = compareLegacyArchive(root, &r)
	if err != nil {
		return nil, err
	}
	source := "legacy-recorded"
	deps := append([]LegacyProvenanceDependency(nil), r.Dependencies...)
	if archive != nil {
		source = "normal-archive"
		ab, e := readCompatibilityFile(root, archive.Path)
		if e != nil || sum(ab) != archive.SHA256 {
			return nil, fmt.Errorf("legacy/archive changed before return: %v", e)
		}
		deps = append(deps, LegacyProvenanceDependency{archive.Path, archive.SHA256, int64(len(ab))})
	}
	if err = verifyLegacyDependencies(root, &r); err != nil {
		return nil, err
	}
	return &VerifiedLegacyGenerationProvenance{&r, source, archive, deps}, nil
}
func legacyNoPending(root string, o LegacyGenerationProvenanceOptions) error {
	guard := legacyProvenanceGuard(o)
	_, err := compatibilityPath(root, filepath.ToSlash(filepath.Dir(guard))+"/sentinel", false)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = os.Lstat(filepath.Join(root, filepath.FromSlash(guard))); os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("legacy provenance mutation=unknown: pending transaction; diagnose before replay")
}
func legacyConflicts(root string, r *LegacyGenerationProvenance, record bool) error {
	dir := filepath.Dir(legacyProvenancePath(r.Options.Locale, r.Identity))
	// Exact namespace conflict checking is not authority selection: no record
	// found here can supply missing inputs or become a latest fallback.
	full, err := compatibilityPath(root, dir+"/sentinel", false)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Dir(full))
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") || !e.Type().IsRegular() {
			return fmt.Errorf("invalid legacy authority namespace entry")
		}
		path := dir + "/" + e.Name()
		b, err := readCompatibilityFile(root, path)
		if err != nil {
			return err
		}
		var old LegacyGenerationProvenance
		if err = contentidentity.StrictJSON(b, &old); err != nil {
			return err
		}
		if old.Schema != LegacyGenerationProvenanceSchema || old.Capability != LegacyGenerationProvenanceCapability || validateLegacyProvenanceOptions(old.Options) != nil || old.Identity != legacyProvenanceIdentity(old) || old.AuthorityID != old.Identity || path != legacyProvenancePath(old.Options.Locale, old.Identity) {
			return fmt.Errorf("conflicting invalid recorded legacy authority")
		}
		if old.Options.BatchID == r.Options.BatchID && old.Options.UnitID == r.Options.UnitID && old.Options.Attempt == r.Options.Attempt {
			if record || !reflect.DeepEqual(old, *r) {
				return fmt.Errorf("duplicate or conflicting already-recorded legacy authority")
			}
		}
	}
	return nil
}

func PrepareLegacyGenerationProvenance(root string, c *Catalog, o LegacyGenerationProvenanceOptions, record bool, expectedIdentity string) (*LegacyGenerationProvenancePlan, error) {
	return prepareLegacyGenerationProvenance(root, c, o, record, expectedIdentity, nil)
}

// beforePublish is an invocation-local test seam; no production hook/global
// mutation flag can weaken reconstruction or publication checks.
func prepareLegacyGenerationProvenance(root string, c *Catalog, o LegacyGenerationProvenanceOptions, record bool, expectedIdentity string, beforePublish func()) (*LegacyGenerationProvenancePlan, error) {
	if err := validateLegacyProvenanceOptions(o); err != nil {
		return nil, err
	}
	if (!record && expectedIdentity != "") || (record && !validSHA256(expectedIdentity)) {
		return nil, fmt.Errorf("expected-identity required exclusively for record")
	}
	if err := legacyNoPending(root, o); err != nil {
		return nil, err
	}
	r, err := buildLegacyGenerationProvenance(root, c, o)
	if err != nil {
		return nil, err
	}
	archive, err := compareLegacyArchive(root, r)
	if err != nil {
		return nil, err
	}
	if archive != nil {
		return nil, fmt.Errorf("original archive exists: normal archive path required; legacy record rejected")
	}
	p := &LegacyGenerationProvenancePlan{PlannedReceipt: r, PlannedPath: legacyProvenancePath(o.Locale, r.Identity), Identity: r.Identity, Mutation: "none"}
	if err = legacyConflicts(root, r, record); err != nil {
		return p, err
	}
	if !record {
		return p, nil
	}
	if expectedIdentity != r.Identity {
		return p, fmt.Errorf("legacy expected-identity mismatch")
	}
	return publishLegacyGenerationProvenance(root, p, func() (*LegacyGenerationProvenance, error) {
		return buildLegacyGenerationProvenance(root, c, o)
	}, beforePublish)
}

func publishLegacyGenerationProvenance(root string, p *LegacyGenerationProvenancePlan, rebuild func() (*LegacyGenerationProvenance, error), beforePublish func()) (_ *LegacyGenerationProvenancePlan, resultErr error) {
	r := p.PlannedReceipt
	o := r.Options
	destination, err := compatibilityPath(root, p.PlannedPath, true)
	if err != nil {
		return p, err
	}
	guard := filepath.Join(root, filepath.FromSlash(legacyProvenanceGuard(o)))
	if err = os.Mkdir(guard, 0700); err != nil {
		p.Mutation = "unknown"
		return p, fmt.Errorf("legacy pending transaction: mutation=unknown; diagnose: %w", err)
	}
	keep := false
	defer func() {
		if !keep {
			if cleanupErr := os.RemoveAll(guard); cleanupErr != nil {
				p.Mutation = "unknown"
				resultErr = fmt.Errorf("legacy staging cleanup mutation=unknown; diagnose: %v; original: %v", cleanupErr, resultErr)
			}
		}
	}()
	data := append(mustJSON(r), '\n')
	staged := filepath.Join(guard, "receipt.json")
	if err = os.WriteFile(staged, data, 0600); err != nil {
		return p, err
	}
	if beforePublish != nil {
		beforePublish()
	}
	fresh, err := rebuild()
	if err != nil {
		return p, err
	}
	if !reflect.DeepEqual(fresh, r) {
		return p, fmt.Errorf("legacy dependencies changed before publication")
	}
	if err = legacyConflicts(root, r, true); err != nil {
		return p, err
	}
	if err = verifyLegacyDependencies(root, r); err != nil {
		return p, err
	}
	archive, err := compareLegacyArchive(root, r)
	if err != nil {
		return p, err
	}
	if archive != nil {
		return p, fmt.Errorf("archive appeared before legacy publication; normal archive required")
	}
	// Recheck every path component after all proof work, immediately before link.
	if _, err = compatibilityPath(root, p.PlannedPath, false); err != nil {
		return p, err
	}
	if err = os.Link(staged, destination); err != nil {
		if !os.IsExist(err) {
			keep, p.Mutation = true, "unknown"
			return p, fmt.Errorf("legacy publication mutation=unknown; preserve and diagnose: %w", err)
		}
		return p, err
	}
	keep = true
	p.Mutation = "unknown"
	// Once linked, any missing readback or cleanup outcome must preserve the
	// pending evidence. Never delete/rewrite a possibly committed receipt.
	actual, err := readCompatibilityFile(root, p.PlannedPath)
	if err != nil || !bytes.Equal(actual, data) {
		return p, fmt.Errorf("legacy publication mutation=unknown; preserve and diagnose: %v", err)
	}
	p.ReceiptPath, p.ReceiptSHA256 = p.PlannedPath, sum(data)
	p.Recorded, p.Mutation = true, "known"
	if err = os.Remove(staged); err != nil {
		return p, fmt.Errorf("legacy authority committed; mutation=known; pending cleanup diagnosis: %w", err)
	}
	if err = os.Remove(guard); err != nil {
		return p, fmt.Errorf("legacy authority committed; mutation=known; pending cleanup diagnosis: %w", err)
	}
	return p, nil
}
