package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
)

const ApprovedIntervalSelectionSchema = "go-learning/approved-interval-selection/v1"
const ApprovedIntervalTransportSchema = "go-learning/approved-interval-transport/v1"
const ApprovedIntervalMode = "approved-interval-opaque"
const approvedIntervalSelectionMember = "formal/approved-interval-selection.json"

var approvedIntervalImplementationPaths = []string{
	"internal/i18n/approved_interval_transport.go", "internal/i18n/current_validation_approved_interval.go",
	"internal/i18n/generation_revision_inputs.go", "internal/i18n/generation_bundle.go",
	"internal/i18n/retranslation_process_context.go", "internal/i18n/generation_import_archive.go",
	"internal/i18n/bridge_invocation.go", "internal/i18n/candidate_snapshot_outputs.go", "internal/i18n/retranslation_promote.go",
	"internal/sitecontent/approved_interval_transport.go", "internal/sitecontent/approved_interval_coordinates.go",
}

type ApprovedIntervalSelectionUnit struct {
	CurrentUnitID    string                    `json:"current_unit_id"`
	Mode             string                    `json:"mode"`
	LegacyProvenance *GlossaryArchiveReference `json:"legacy_provenance,omitempty"`
}
type ApprovedIntervalSelection struct {
	Schema         string                          `json:"schema"`
	Locale         string                          `json:"locale"`
	BatchID        string                          `json:"batch_id"`
	ContentPackage string                          `json:"content_package"`
	Units          []ApprovedIntervalSelectionUnit `json:"units"`
	bytes          []byte
}

// Parsing a request grants no authority. Evidence is independently reconstructed.
func ParseApprovedIntervalSelection(root, path string, c *Catalog, m *RetranslationBatchManifest) (*ApprovedIntervalSelection, error) {
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return nil, err
	}
	return parseApprovedIntervalSelectionBytes(root, b, c, m)
}
func parseApprovedIntervalSelectionBytes(root string, b []byte, c *Catalog, m *RetranslationBatchManifest) (*ApprovedIntervalSelection, error) {
	var s ApprovedIntervalSelection
	if err := contentidentity.StrictJSON(b, &s); err != nil {
		return nil, err
	}
	pkg, _, _, _, err := translationAuthorizationScope(c)
	if err != nil || s.Schema != ApprovedIntervalSelectionSchema || s.Locale != m.Locale || s.BatchID != m.BatchID || s.ContentPackage != pkg || len(s.Units) == 0 {
		return nil, fmt.Errorf("approved interval selection identity/scope mismatch")
	}
	pos := map[string]int{}
	for i, u := range m.Units {
		pos[u.UnitID] = i
	}
	last := -1
	for _, u := range s.Units {
		p, ok := pos[u.CurrentUnitID]
		if !ok || p <= last || u.Mode != ApprovedIntervalMode {
			return nil, fmt.Errorf("approved interval selection duplicate/order/mode/Unit scope mismatch")
		}
		last = p
		r := m.Units[p]
		if r.RevisionFeedbackSource != CurrentValidationRevisionFeedback || r.RepresentationBridge != nil || r.UnitKind != UnitKindPage {
			return nil, fmt.Errorf("approved interval requires exact current-validation Page scope")
		}
		if u.LegacyProvenance != nil {
			if validateGenerationInstallPath(u.LegacyProvenance.Path) != nil || !validSHA256(u.LegacyProvenance.SHA256) {
				return nil, fmt.Errorf("approved interval legacy reference requires exact repository path/SHA")
			}
			if _, err = readCompatibilityFile(root, u.LegacyProvenance.Path); err != nil {
				return nil, err
			}
		}
	}
	s.bytes = bytes.Clone(b)
	return &s, nil
}
func (s *ApprovedIntervalSelection) entry(id string) *ApprovedIntervalSelectionUnit {
	if s == nil {
		return nil
	}
	for i := range s.Units {
		if s.Units[i].CurrentUnitID == id {
			return &s.Units[i]
		}
	}
	return nil
}

type ApprovedIntervalCoordinate struct {
	Space string `json:"space"`
	Unit  string `json:"offset_unit"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}
type ApprovedIntervalFindingRegion struct {
	Diagnostic       languageaudit.Diagnostic   `json:"diagnostic"`
	Visible          ApprovedIntervalCoordinate `json:"visible"`
	Rendered         ApprovedIntervalCoordinate `json:"rendered"`
	Candidate        ApprovedIntervalCoordinate `json:"candidate"`
	ConversionSHA256 string                     `json:"conversion_sha256"`
}
type ApprovedIntervalProjection struct {
	Namespace                   string                          `json:"namespace"`
	UnitID                      string                          `json:"unit_id"`
	SourceSHA256                string                          `json:"source_sha256"`
	OldRepresentationSHA256     string                          `json:"old_representation_sha256"`
	CurrentRepresentationSHA256 string                          `json:"current_representation_sha256"`
	CandidateSHA256             string                          `json:"candidate_sha256"`
	ParentSource                ApprovedIntervalCoordinate      `json:"parent_source"`
	ParentCandidate             ApprovedIntervalCoordinate      `json:"parent_candidate"`
	ParentSHA256                string                          `json:"parent_target_sha256"`
	ParentToken                 string                          `json:"parent_token"`
	Children                    json.RawMessage                 `json:"current_children_source_inventory"`
	Findings                    []ApprovedIntervalFindingRegion `json:"typed_finding_regions"`
	Text                        string                          `json:"text"`
	Tokens                      []string                        `json:"complete_projection_token_inventory"`
	Codec                       json.RawMessage                 `json:"inverse_restore_contract"`
}

// The adapter operates only on verified, source-owned historical mapping.
// The supplied diagnostics are typed machine facts, never parsed finding prose.
type ApprovedIntervalAdapter interface {
	ApprovedIntervalProjection(TranslationUnitAdapter, []byte, []byte, RepresentationSemanticQuestion, languageaudit.Result) (ApprovedIntervalProjection, TranslationUnitAdapter, error)
}
type ApprovedIntervalTransport struct {
	Schema                 string                              `json:"schema"`
	Mode                   string                              `json:"mode"`
	SelectionSHA256        string                              `json:"selection_sha256"`
	Record                 RetranslationBatchUnit              `json:"revision_record"`
	Reopen                 GlossaryArchiveReference            `json:"language_authority"`
	Snapshot               QualityCheckSnapshotUnit            `json:"snapshot"`
	SemanticAuthority      GlossaryArchiveReference            `json:"semantic_authority"`
	ProvenanceSource       string                              `json:"provenance_source"`
	LegacyReference        *GlossaryArchiveReference           `json:"legacy_reference,omitempty"`
	HistoricalVerification *VerifiedLegacyGenerationProvenance `json:"historical_verification,omitempty"`
	RequestedOptions       LegacyGenerationProvenanceOptions   `json:"requested_options"`
	Dependencies           []GlossaryArchiveReference          `json:"dependencies"`
	CurrentAuthority       string                              `json:"current_authority"`
	Projection             ApprovedIntervalProjection          `json:"projection"`
	Identity               string                              `json:"identity"`
	output                 protectedTranslation
}

func approvedIntervalIdentity(p ApprovedIntervalTransport) string {
	p.Identity = ""
	return sum(mustJSON(p))
}
func approvedIntervalProofMember(id string) string {
	return "formal/approved-interval-transport/" + id + ".json"
}

func buildApprovedIntervalTransport(root string, c *Catalog, g *Glossary, locale string, r RetranslationBatchUnit, reopen *QualityCheckSurfaceReopen, s QualityCheckSnapshotUnit, e *snapshotUnitRepositoryEvidence, selection *ApprovedIntervalSelection) (*ApprovedIntervalTransport, error) {
	entry := selection.entry(r.UnitID)
	if entry == nil || reopen == nil || reopen.AuthoritySource != CurrentValidationAuthority || e.machine == nil || e.machine.unit.SemanticAuthority == nil {
		return nil, fmt.Errorf("approved interval requires explicit selection and recorded semantic reconciliation")
	}
	if selection.ContentPackage != reopen.CurrentValidation.Package || r.RevisionAuthorizationID != reopen.ReopenID || r.PreviousSnapshotID != reopen.PreviousSnapshotID || r.PreviousReviewSHA256 != reopen.revisionEvidence().SHA256 {
		return nil, fmt.Errorf("approved interval language scope mismatch")
	}
	machine := e.machine
	review, ref, err := machine.ctx.semanticAuthority(root, c, r.UnitID)
	if err != nil {
		return nil, err
	}
	old, err := machine.ctx.resolution.HistoricalCatalog.Unit(machine.unit.Predecessor.UnitID)
	if err != nil {
		return nil, err
	}
	raw, err := readCompatibilityFile(root, machine.unit.OriginalRaw.Path)
	if err != nil || sum(raw) != machine.unit.OriginalRaw.SHA256 {
		return nil, fmt.Errorf("approved interval historical raw drift: %v", err)
	}
	restored, err := old.Adapter.Restore(string(raw))
	if err != nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF(restored), e.candidate) {
		return nil, fmt.Errorf("approved interval historical restore not exact: %v", err)
	}
	// Import identity is taken from the immutable semantic review inventory copy.
	srBytes, err := readCompatibilityFile(root, ref.Path)
	if err != nil {
		return nil, err
	}
	var sr RepresentationReviewReceipt
	if err = contentidentity.StrictJSON(srBytes, &sr); err != nil {
		return nil, err
	}
	rb, err := readCompatibilityFile(root, sr.Bundle.Path)
	if err != nil || sum(rb) != sr.Bundle.SHA256 {
		return nil, fmt.Errorf("approved interval semantic Bundle drift")
	}
	_, files, err := readLegacySemanticBundle(rb)
	if err != nil {
		return nil, err
	}
	base := "data/retranslation-runs/" + locale + "/" + s.SelectedBatchID + "/"
	// Surviving formal input independently proves the saved historical codec.
	// This consumes formal-source-input, not unknown original ZIP member bytes
	// or a predecessor template inferred from the candidate.
	historicalInput, _, err := old.Adapter.ProtectedInput(nil)
	if err != nil {
		return nil, err
	}
	savedInput, err := readCompatibilityFile(root, base+e.record.InputPath)
	if err != nil || sum(savedInput) != e.record.InputSHA256 || !bytes.Equal(canonicalizeRetranslationArtifactEOF([]byte(historicalInput)), savedInput) {
		return nil, fmt.Errorf("approved interval historical codec does not reproduce formal input")
	}
	ip := base + "generation-imports/initial-manifest.json"
	ib := files["inputs/"+ip]
	var imported GenerationResultBundleManifest
	if err = contentidentity.StrictJSON(ib, &imported); err != nil {
		return nil, err
	}
	options := LegacyGenerationProvenanceOptions{Locale: locale, ContentPackage: selection.ContentPackage, BatchID: s.SelectedBatchID, UnitID: old.ID, CurrentUnitID: r.UnitID, Attempt: s.Attempt, TaskKind: imported.TaskKind, ImportReceipt: GlossaryArchiveReference{ip, sum(ib)}, OriginalBundleSHA256: imported.GenerationBundleSHA256, PreviousSnapshotID: r.PreviousSnapshotID, SnapshotSHA256: reopen.CurrentValidation.Snapshot.SHA256}
	proof := &ApprovedIntervalTransport{Schema: ApprovedIntervalTransportSchema, Mode: ApprovedIntervalMode, SelectionSHA256: sum(selection.bytes), Record: r, Reopen: reopen.revisionEvidence(), Snapshot: s, SemanticAuthority: *ref, LegacyReference: entry.LegacyProvenance, RequestedOptions: options}
	proof.Dependencies = append(proof.Dependencies, GlossaryArchiveReference{base + e.record.InputPath, sum(savedInput)})
	if entry.LegacyProvenance != nil {
		verified, err := VerifyLegacyGenerationProvenance(root, c, *entry.LegacyProvenance, options)
		if err != nil {
			return nil, err
		}
		if verified.Authority.ConsumerContract != "historical-response/v1" || verified.Authority.Capability != LegacyGenerationProvenanceCapability {
			return nil, fmt.Errorf("approved interval unsupported legacy facts contract")
		}
		proof.ProvenanceSource = verified.ProvenanceSource
		proof.HistoricalVerification = verified
		for _, d := range verified.Dependencies {
			proof.Dependencies = append(proof.Dependencies, GlossaryArchiveReference{d.Path, d.SHA256})
		}
		proof.Dependencies = append(proof.Dependencies, *entry.LegacyProvenance)
	} else {
		proof.ProvenanceSource = "normal-archive"
	}
	archivePath := importedGenerationArchive(locale, imported.GenerationBundleSHA256)
	archive, archiveErr := readCompatibilityFile(root, archivePath)
	if archiveErr == nil {
		original, archiveFiles, err := readGenerationBundle(archive)
		if err != nil {
			return nil, err
		}
		if err = checkImportedGenerationIdentity(imported, original, archive, archiveFiles); err != nil {
			return nil, err
		}
		matched := 0
		for _, out := range original.ExpectedOutputs {
			if out.UnitID == old.ID && out.InstallPath == e.validation.RawResponsePath {
				for _, o := range imported.Outputs {
					if o.BundlePath == out.BundlePath && o.SHA256 == sum(raw) && o.Size == int64(len(raw)) {
						matched++
					}
				}
			}
		}
		if matched != 1 || !bytes.Equal(archiveFiles["formal/manifest.json"], files["inputs/"+base+"manifest.json"]) {
			return nil, fmt.Errorf("approved interval normal archive invocation mismatch")
		}
		input, err := readCompatibilityFile(root, base+e.record.InputPath)
		if err != nil || sum(input) != e.record.InputSHA256 || !bytes.Equal(input, archiveFiles["formal/"+e.record.InputPath]) {
			return nil, fmt.Errorf("approved interval normal archive input mismatch")
		}
		proof.ProvenanceSource = "normal-archive"
		proof.Dependencies = append(proof.Dependencies, GlossaryArchiveReference{archivePath, sum(archive)})
	} else if !os.IsNotExist(archiveErr) {
		return nil, archiveErr
	} else if entry.LegacyProvenance == nil || proof.ProvenanceSource != "legacy-recorded" {
		return nil, fmt.Errorf("approved interval absent archive requires explicit recorded legacy reference")
	}
	if proof.HistoricalVerification != nil && proof.HistoricalVerification.ProvenanceSource != proof.ProvenanceSource {
		return nil, fmt.Errorf("approved interval verified provenance/archive state changed")
	}
	u, err := c.Unit(r.UnitID)
	if err != nil {
		return nil, err
	}
	factory, ok := u.Adapter.(ApprovedIntervalAdapter)
	if !ok {
		return nil, fmt.Errorf("approved interval adapter lacks source-owned projection")
	}
	// Reproduce the persisted finding by exact error identity, while using only
	// typed diagnostics to compute edit scope. No substring/keyword extraction.
	audit, err := TranslationUnitLanguageAudit(root, c, u, locale, e.candidate, g)
	if err != nil {
		return nil, err
	}
	blocking := languageaudit.Blocking(audit)
	activeErr := validateTranslationUnitWithGlossary(root, c, u, locale, e.candidate, g)
	var finding *CurrentValidationReopenFinding
	for i := range reopen.CurrentValidation.Findings {
		f := &reopen.CurrentValidation.Findings[i]
		if f.UnitID == r.UnitID {
			if finding != nil {
				return nil, fmt.Errorf("duplicate approved interval finding")
			}
			finding = f
		}
	}
	if finding == nil || len(finding.CommentSlots) != 0 || blocking == nil || activeErr == nil || activeErr.Error() != blocking.Error() || finding.Reason != activeErr.Error() {
		return nil, fmt.Errorf("approved interval current finding cannot be reproduced as complete typed language diagnostics")
	}
	projection, adapter, err := factory.ApprovedIntervalProjection(old.Adapter, raw, e.candidate, review.Question, audit)
	if err != nil {
		return nil, err
	}
	got, err := adapter.Restore(projection.Text)
	if err != nil || !bytes.Equal(canonicalizeRetranslationArtifactEOF(got), e.candidate) {
		return nil, fmt.Errorf("approved interval projection baseline restore mismatch: %v", err)
	}
	proof.Projection = projection
	proof.CurrentAuthority, err = currentTranslationAuthority(root, u, locale)
	if err != nil {
		return nil, err
	}
	proof.Dependencies = append(proof.Dependencies, machineAuthorityReferences(machine)...)
	proof.Dependencies = append(proof.Dependencies, sr.Bundle, sr.Evidence, options.ImportReceipt, proof.Reopen, reopen.CurrentValidation.Snapshot, GlossaryArchiveReference{u.SourcePath, u.SourceSHA256})
	for _, path := range append(append([]string(nil), approvedIntervalImplementationPaths...), "locales/"+locale+"/glossary.yaml", "locales/"+locale+"/locale.json") {
		b, err := readCompatibilityFile(root, path)
		if err != nil {
			return nil, err
		}
		proof.Dependencies = append(proof.Dependencies, GlossaryArchiveReference{path, sum(b)})
	}
	if err = RequireCurrentGlossaryReview(root, locale); err != nil {
		return nil, err
	}
	for _, d := range proof.Dependencies {
		b, err := readCompatibilityFile(root, d.Path)
		if err != nil || sum(b) != d.SHA256 {
			return nil, fmt.Errorf("approved interval dependency drift: %s: %v", d.Path, err)
		}
	}
	proof.Identity = approvedIntervalIdentity(*proof)
	text, tokens, err := adapter.ProtectedInput(g)
	if err != nil {
		return nil, err
	}
	proof.output = protectedTranslation{Text: text, Tokens: tokens, Adapter: adapter}
	return proof, nil
}

// Read only the exact initial imported archive, never discover an authority by
// Unit or latest receipt. Actual new Bundle bytes carry the frozen request.
func importedApprovedIntervalSelection(root string, c *Catalog, m *RetranslationBatchManifest) (*ApprovedIntervalSelection, map[string][]byte, error) {
	path := "data/retranslation-runs/" + m.Locale + "/" + m.BatchID + "/generation-imports/initial-manifest.json"
	b, err := readCompatibilityFile(root, path)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var imported GenerationResultBundleManifest
	if err = contentidentity.StrictJSON(b, &imported); err != nil {
		return nil, nil, err
	}
	archive, err := readCompatibilityFile(root, importedGenerationArchive(m.Locale, imported.GenerationBundleSHA256))
	if err != nil {
		return nil, nil, err
	}
	gm, files, err := readGenerationBundle(archive)
	if err != nil {
		return nil, nil, err
	}
	if err = checkImportedGenerationIdentity(imported, gm, archive, files); err != nil {
		return nil, nil, err
	}
	if gm.BatchID != m.BatchID || gm.Locale != m.Locale || gm.Attempt != 1 {
		return nil, nil, fmt.Errorf("approved interval installed invocation mismatch")
	}
	disk, err := readCompatibilityFile(root, filepath.ToSlash(filepath.Join("data/retranslation-runs", m.Locale, m.BatchID, "manifest.json")))
	if err != nil || !bytes.Equal(disk, files["formal/manifest.json"]) {
		return nil, nil, fmt.Errorf("approved interval installed batch manifest mismatch")
	}
	if len(files[approvedIntervalSelectionMember]) == 0 {
		return nil, files, nil
	}
	s, err := parseApprovedIntervalSelectionBytes(root, files[approvedIntervalSelectionMember], c, m)
	return s, files, err
}
func compareInstalledApprovedProofs(inputs []generationBundleInput, files map[string][]byte) error {
	expected := map[string]bool{}
	for _, in := range inputs {
		if !strings.HasPrefix(in.path, "formal/approved-interval-transport/") {
			continue
		}
		if !bytes.Equal(in.data, files[in.path]) {
			return fmt.Errorf("approved interval installed decoder/proof drift: %s", in.path)
		}
		expected[in.path] = true
	}
	for path := range files {
		if strings.HasPrefix(path, "formal/approved-interval-transport/") && !expected[path] {
			return fmt.Errorf("unselected approved interval proof: %s", path)
		}
	}
	return nil
}

func generationApprovedIntervalSelection(root string, c *Catalog, m *RetranslationBatchManifest, options GenerationBundleOptions) (*ApprovedIntervalSelection, error) {
	if m.TransportRecovery != nil {
		if options.ApprovedIntervalSelection != "" {
			return nil, fmt.Errorf("recovery derives approved selection; external selection forbidden")
		}
		s, err := derivedRecoverySelection(root, c, m)
		if err != nil {
			return nil, err
		}
		if len(options.approvedIntervalSelection) > 0 && !bytes.Equal(options.approvedIntervalSelection, s.bytes) {
			return nil, fmt.Errorf("frozen recovery selection differs from canonical derivation")
		}
		return s, nil
	}
	if options.ApprovedIntervalSelection != "" && len(options.approvedIntervalSelection) != 0 {
		return nil, fmt.Errorf("conflicting approved interval requests")
	}
	var s *ApprovedIntervalSelection
	var err error
	if options.ApprovedIntervalSelection != "" {
		s, err = ParseApprovedIntervalSelection(root, options.ApprovedIntervalSelection, c, m)
	}
	if len(options.approvedIntervalSelection) != 0 {
		s, err = parseApprovedIntervalSelectionBytes(root, options.approvedIntervalSelection, c, m)
	}
	if err != nil {
		return nil, err
	}
	if options.UnitID != "" {
		original, _, err := importedApprovedIntervalSelection(root, c, m)
		if err != nil {
			return nil, err
		}
		if s != nil && (original == nil || !bytes.Equal(s.bytes, original.bytes)) {
			return nil, fmt.Errorf("retry must preserve original approved interval selection")
		}
		s = original
	}
	return s, nil
}

// The real imported initial Bundle is the sole persisted selector. No external
// file, latest receipt, or caller mode can replace it for downstream decoding.
func approvedResponseInputs(root string, c *Catalog, m *RetranslationBatchManifest, transports ...map[string]protectedTranslation) ([]generationBundleInput, error) {
	if m.TransportRecovery != nil {
		if err := verifyRecoveryInstalledExecution(root, c, m); err != nil {
			return nil, err
		}
	}
	s, files, err := importedApprovedIntervalSelection(root, c, m)
	if err != nil {
		return nil, err
	}
	if m.TransportRecovery!=nil {
		expected,err:=derivedRecoverySelection(root,c,m)
		if err!=nil || s==nil || !bytes.Equal(s.bytes,expected.bytes){return nil,fmt.Errorf("installed recovery selection differs from authority derivation: %v",err)}
	}
	inputs, err := revisionGenerationInputsCore(root, c, m.Locale, m.Units, transports, false, s)
	if err != nil {
		return nil, err
	}
	inputs, err = applySurfaceTransportInWorld(root, c, m.Locale, m.Units, inputs, true)
	if err != nil {
		return nil, err
	}
	if err = compareInstalledApprovedProofs(inputs, files); err != nil {
		return nil, err
	}
	if m.TransportRecovery != nil {
		inputs, err = recoveryGenerationInputs(root, c, m, inputs)
		if err != nil {
			return nil, err
		}
		for _, in := range inputs {
			if in.path == transportRecoveryConstructionMember && !bytes.Equal(in.data, files[in.path]) {
				return nil, fmt.Errorf("installed recovery construction drift")
			}
		}
	}
	return inputs, nil
}

// Recheck exact immutable dependencies and observed original archive state at
// return/publication, including archive reappearance after a legacy decision.
func checkApprovedIntervalInputs(root string, inputs []generationBundleInput) error {
	approved := false
	for _, in := range inputs {
		if !strings.HasPrefix(in.path, "formal/approved-interval-transport/") {
			continue
		}
		approved = true
		var p ApprovedIntervalTransport
		if err := contentidentity.StrictJSON(in.data, &p); err != nil {
			return err
		}
		if p.Schema != ApprovedIntervalTransportSchema || p.Mode != ApprovedIntervalMode || p.Identity != approvedIntervalIdentity(p) {
			return fmt.Errorf("invalid approved interval proof")
		}
		for _, ref := range p.Dependencies {
			b, err := readCompatibilityFile(root, ref.Path)
			if err != nil || sum(b) != ref.SHA256 {
				return fmt.Errorf("approved interval dependency changed: %s", ref.Path)
			}
		}
		path := importedGenerationArchive(p.RequestedOptions.Locale, p.RequestedOptions.OriginalBundleSHA256)
		_, err := readCompatibilityFile(root, path)
		if p.ProvenanceSource == "legacy-recorded" {
			if !os.IsNotExist(err) {
				return fmt.Errorf("approved interval historical archive state changed")
			}
		} else if p.ProvenanceSource != "normal-archive" || err != nil {
			return fmt.Errorf("approved interval normal archive unavailable")
		}
	}
	if approved {
		for _, in := range inputs {
			if in.repositoryPath == "" {
				continue
			}
			b, err := readCompatibilityFile(root, in.repositoryPath)
			if err != nil || !bytes.Equal(b, in.data) {
				return fmt.Errorf("approved interval frozen input changed: %s", in.repositoryPath)
			}
		}
	}
	return nil
}
