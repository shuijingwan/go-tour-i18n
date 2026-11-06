package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

var currentValidationPredecessorImplementationPaths = []string{
	"internal/i18n/current_validation_predecessor.go", "internal/i18n/generation_revision_inputs.go",
	"internal/i18n/surface_transport.go", "internal/i18n/surface_same_unit_bridge.go",
	"internal/sitecontent/surface_same_unit_bridge.go", "internal/sitecontent/surface_transport.go",
	"internal/sitecontent/page_replacement.go", "docs/TRANSLATION_UNIT_REPLACEMENT.md",
}

// A diagnostic/derived transport, never a language receipt or an A decision.
// Private fields retain the canonical evidence/decoder within this invocation.
type CurrentValidationPredecessor struct {
	Schema                         string                     `json:"schema"`
	Locale                         string                     `json:"locale"`
	ContentPackage                 string                     `json:"content_package"`
	PackageIdentity                string                     `json:"complete_package_identity"`
	OrderedScope                   []string                   `json:"complete_ordered_scope"`
	SnapshotManifest               GlossaryArchiveReference   `json:"snapshot_manifest"`
	ImplementationInputs           []GlossaryArchiveReference `json:"implementation_inputs"`
	ImplementationIdentity         string                     `json:"implementation_identity"`
	CurrentAuthority               string                     `json:"current_adapter_authority"`
	OldRepresentationIdentity      string                     `json:"old_representation_identity"`
	ResponseRepresentationIdentity string                     `json:"response_representation_identity"`
	CurrentRepresentationIdentity  string                     `json:"current_representation_identity"`
	UnitID                         string                     `json:"unit_id"`
	CurrentSource                  GlossaryArchiveReference   `json:"current_source"`
	SourcePath                     string                     `json:"source_path"`
	Snapshot                       QualityCheckSnapshotUnit   `json:"snapshot"`
	Raw                            GlossaryArchiveReference   `json:"historical_raw"`
	OrdinaryRestore                bool                       `json:"current_ordinary_restore"`
	RecoveryClass                  string                     `json:"recovery_class"`
	Blocker                        string                     `json:"exact_blocker,omitempty"`
	Reopen                         GlossaryArchiveReference   `json:"current_validation_reopen"`
	ReopenIdentity                 string                     `json:"current_validation_reopen_identity"`
	Selected                       []RetranslationBatchUnit   `json:"exact_revision_scope"`
	OldRecord                      RetranslationBatchUnit     `json:"historical_record"`
	CurrentInputSHA256             string                     `json:"current_input_sha256"`
	Template                       string                     `json:"protected_template,omitempty"`
	TemplateSHA256                 string                     `json:"protected_template_sha256,omitempty"`
	Dependencies                   []GlossaryArchiveReference `json:"dependencies,omitempty"`
	output                         protectedTranslation
	evidence                       *snapshotUnitRepositoryEvidence
	lineage                        *bridgeResponseLineage
}

// InspectCurrentValidationPredecessors uses the same frozen A evaluation as
// Generation Bundle. Diagnostic failure classes never authorize publication.
func InspectCurrentValidationPredecessors(root string, c *Catalog, locale string, selected []RetranslationBatchUnit) ([]CurrentValidationPredecessor, error) {
	return inspectCurrentValidationPredecessors(root, c, locale, selected)
}

func inspectCurrentValidationPredecessors(root string, c *Catalog, locale string, selected []RetranslationBatchUnit) ([]CurrentValidationPredecessor, error) {
	if c == nil || ValidateLocaleName(locale) != nil || len(selected) == 0 || len(selected) > MaxRetranslationExportLimit {
		return nil, fmt.Errorf("exact current-validation revision scope required")
	}
	r0 := selected[0]
	historical, reopen, err := surfaceRevisionCatalog(root, c, locale, r0.RevisionAuthorizationID)
	if err != nil {
		return nil, err
	}
	if reopen.AuthoritySource != CurrentValidationAuthority || reopen.CurrentValidation == nil || len(reopen.Units) != len(selected) {
		return nil, fmt.Errorf("exclusive complete current-validation predecessor authority required")
	}
	pkg, _, _, _, err := translationAuthorizationScope(c)
	if err != nil || pkg != reopen.CurrentValidation.Package {
		return nil, fmt.Errorf("current-validation predecessor package mismatch")
	}
	assessment, err := decodeCurrentValidationAssessment(reopen.CurrentValidation)
	if err != nil {
		return nil, err
	}
	ev := currentValidationPredecessorEvaluation(root, historical, locale)
	if err = ev.materializeOriginalFinalization(&assessment.Historical); err != nil {
		return nil, err
	}
	g, err := LoadGlossary(root, locale)
	if err != nil {
		return nil, err
	}
	var rows []CurrentValidationPredecessor
	for i, r := range selected {
		q := reopen.Units[i]
		ref := reopen.revisionEvidence()
		if r.RevisionFeedbackSource != CurrentValidationRevisionFeedback || r.RepresentationBridge != nil || r.RevisionAuthorizationID != reopen.ReopenID || r.UnitID != q.UnitID || r.PreviousSnapshotID != reopen.PreviousSnapshotID || r.PreviousRating != "A" || r.PreviousFinding != q.Finding || r.PreviousReviewPath != ref.Path || r.PreviousReviewSHA256 != ref.SHA256 {
			return nil, fmt.Errorf("current-validation predecessor exact scope/finding mismatch")
		}
		e, err := ev.readSnapshotUnitRepositoryEvidence(reopen.PreviousSnapshotID, q.Snapshot)
		if err != nil {
			return nil, err
		}
		row, err := inspectCurrentValidationPredecessor(root, c, historical, g, locale, selected, r, reopen, q.Snapshot, e)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func inspectCurrentValidationPredecessor(root string, current, historical *Catalog, g *Glossary, locale string, selected []RetranslationBatchUnit, r RetranslationBatchUnit, reopen *QualityCheckSurfaceReopen, s QualityCheckSnapshotUnit, e *snapshotUnitRepositoryEvidence) (CurrentValidationPredecessor, error) {
	row := CurrentValidationPredecessor{Schema: "go-learning/current-validation-predecessor-transport/v1", Locale: locale, SnapshotManifest: reopen.CurrentValidation.Snapshot, UnitID: r.UnitID, SourcePath: r.SourcePath, Snapshot: s, Selected: selected, OldRecord: e.record, Reopen: reopen.revisionEvidence(), ReopenIdentity: reopen.CurrentValidation.Identity, evidence: e, CurrentInputSHA256: r.InputSHA256}
	u, err := current.Unit(r.UnitID)
	if err != nil {
		return row, err
	}
	disk, err := readTranslationUnitSource(root, u)
	if err != nil || validateSnapshotSourceIdentity(current, u, disk) != nil {
		return row, fmt.Errorf("current-validation recovery source changed")
	}
	sourceFile, err := readSnapshotReferencedFile(root, u.SourcePath)
	if err != nil {
		return row, err
	}
	row.CurrentSource = GlossaryArchiveReference{Path: u.SourcePath, SHA256: sum(sourceFile)}
	normal, err := prepareTranslationUnitInput(u, g)
	if err != nil {
		return row, err
	}
	if r.UnitKind != s.UnitKind || r.UnitKind != u.Kind || r.SourcePath != s.SourcePath || r.SourcePath != u.SourcePath || r.SourceSHA256 != s.SourceSHA256 || r.SourceSHA256 != u.SourceSHA256 || sum(canonicalizeRetranslationArtifactEOF([]byte(normal.Text))) != r.InputSHA256 || len(normal.Tokens) != r.ProtectedTokenCount {
		return row, fmt.Errorf("current-validation predecessor current source/input mismatch")
	}
	encoded, err := json.Marshal(normal)
	if err != nil {
		return row, err
	}
	row.CurrentRepresentationIdentity = sum(encoded)
	pkg, pkgIdentity, _, ordered, err := translationAuthorizationScope(current)
	if err != nil || pkg != reopen.CurrentValidation.Package {
		return row, fmt.Errorf("current-validation recovery package mismatch")
	}
	row.ContentPackage, row.PackageIdentity = pkg, pkgIdentity
	for _, unit := range ordered {
		row.OrderedScope = append(row.OrderedScope, unit.ID)
	}
	name := filepath.Base(e.record.InputPath)
	attempt, err := retryValidationAttemptForExtension(e.validation.RawResponsePath, strings.TrimSuffix(name, filepath.Ext(name)), filepath.Ext(name))
	if err != nil || attempt != s.Attempt || e.validation.Status != "passed" {
		return row, fmt.Errorf("predecessor exact raw attempt mismatch")
	}
	rawPath := filepath.ToSlash(filepath.Join("data/retranslation-runs", locale, s.SelectedBatchID, e.validation.RawResponsePath))
	if e.machine != nil {
		rawPath = e.machine.unit.CurrentRaw.Path
	}
	raw, err := readSnapshotReferencedFile(root, rawPath)
	if err != nil {
		return row, err
	}
	row.Raw = GlossaryArchiveReference{Path: rawPath, SHA256: sum(raw)}
	batch, err := readRetranslationProcessManifest(e.batchDir, locale, s.SelectedBatchID)
	if err != nil {
		return row, err
	}
	frame := func(b []byte) []byte {
		if batch.ArtifactEOF == retranslationArtifactEOFSingleLF {
			return canonicalizeRetranslationArtifactEOF(b)
		}
		return b
	}
	if batch.ArtifactEOF == retranslationArtifactEOFSingleLF {
		if err := validateRetranslationArtifactEOF(raw); err != nil {
			return row, err
		}
	}

	p, err := prepareTranslationUnitInput(e.unit, g)
	if err != nil {
		return row, err
	}
	restored, failures := p.restore(string(raw))
	row.OrdinaryRestore = len(failures) == 0 && bytes.Equal(frame([]byte(restored)), e.candidate)
	row.RecoveryClass = "UNAFFECTED_CURRENT_RESTORE_PASS"
	if row.OrdinaryRestore {
		return row, nil
	}
	row.RecoveryClass = "OTHER_FAIL_CLOSED"
	row.Blocker = strings.Join(failures, "; ")
	if row.Blocker == "" {
		row.Blocker = "ordinary raw restores different candidate bytes"
	}
	// Historical response decoding does not consume this revision's authority.
	// Keep the original Surface/QC record, finding, import and bridge exclusive.
	decoder, lineage, err := surfaceTransportOutput(root, historical, locale, e.record)
	surface := err == nil && lineage != nil
	if err == nil && lineage == nil {
		decoder, lineage, err = inheritedBridgeOutput(root, historical, g, locale, e.record)
	}
	if err != nil {
		row.Blocker = err.Error()
		return row, nil
	}
	if lineage == nil {
		return row, nil
	}
	got, failed := decoder.restore(string(raw))
	if len(failed) != 0 || !bytes.Equal(frame([]byte(got)), e.candidate) {
		row.Blocker = "sealed historical decoder does not restore exact Snapshot candidate"
		return row, nil
	}
	if u.Kind != r.UnitKind || u.SourcePath != r.SourcePath || u.SourceSHA256 != r.SourceSHA256 || sum(canonicalizeRetranslationArtifactEOF([]byte(normal.Text))) != r.InputSHA256 || e.record.UnitID != r.UnitID || e.record.InputSHA256 != r.InputSHA256 {
		row.RecoveryClass = "REPLACEMENT_LINEAGE_REQUIRED"
		row.Blocker = "formal source/input/Unit delta requires explicit replacement lineage"
		return row, nil
	}
	encoded, err = json.Marshal(decoder)
	if err != nil {
		return row, err
	}
	row.OldRepresentationIdentity = sum(encoded)
	imported, err := currentValidationPredecessorImport(root, locale, s, e, raw)
	if err != nil {
		row.Blocker = err.Error()
		return row, nil
	}

	var template []byte
	if surface {
		factory, ok := u.Adapter.(SurfaceSameUnitBridgeAdapter)
		if !ok {
			return row, nil
		}
		adapter, text, _, err := factory.BridgeSurfaceRevision(decoder.Adapter, raw, e.candidate)
		if err != nil {
			row.Blocker = err.Error()
			return row, nil
		}
		textInput, tokens, err := adapter.ProtectedInput(g)
		if err != nil {
			return row, err
		}
		decoder = protectedTranslation{Text: textInput, Tokens: tokens, Adapter: adapter}
		template = text
	} else {
		template = raw
	}
	got, failed = decoder.restore(string(template))
	if len(failed) != 0 || !bytes.Equal(frame([]byte(got)), e.candidate) {
		row.Blocker = "composed representation does not restore exact Snapshot candidate"
		return row, nil
	}
	if err = lineage.check(root, locale); err != nil {
		return row, err
	}
	for _, path := range currentValidationPredecessorImplementationPaths {
		b, err := readSnapshotReferencedFile(root, path)
		if err != nil {
			return row, err
		}
		row.ImplementationInputs = append(row.ImplementationInputs, GlossaryArchiveReference{Path: path, SHA256: sum(b)})
	}
	row.ImplementationIdentity, err = TranslationMachineAuthority(root, currentValidationPredecessorImplementationPaths)
	if err != nil {
		return row, err
	}
	row.CurrentAuthority, err = currentTranslationAuthority(root, u, locale)
	if err != nil {
		return row, err
	}
	encoded, err = json.Marshal(normal)
	if err != nil {
		return row, err
	}
	row.CurrentRepresentationIdentity = sum(encoded)
	for _, ref := range row.ImplementationInputs {
		b, err := readSnapshotReferencedFile(root, ref.Path)
		if err != nil || sum(b) != ref.SHA256 {
			return row, fmt.Errorf("recovery implementation changed")
		}
	}
	encoded, err = json.Marshal(decoder)
	if err != nil {
		return row, err
	}
	row.ResponseRepresentationIdentity = sum(encoded)
	row.RecoveryClass = "DETERMINISTIC_REBASE_PASS"
	row.Template = string(template)
	row.TemplateSHA256 = sum(template)
	row.Dependencies = append(append(append([]GlossaryArchiveReference(nil), lineage.Dependencies...), imported...), row.CurrentSource)
	row.output = decoder
	row.lineage = lineage
	return row, nil
}

// A representation recovery requires the actual imported invocation, not just
// a locally plausible raw/candidate pair. Ordinary legacy proofs do not enter
// this recovery-only gate or acquire a fabricated import archive.
func currentValidationPredecessorImport(root, locale string, s QualityCheckSnapshotUnit, e *snapshotUnitRepositoryEvidence, raw []byte) ([]GlossaryArchiveReference, error) {
	probe := GenerationBundleManifest{Attempt: s.Attempt}
	if s.Attempt > 1 {
		probe.TaskKind = "translation-unit-retry"
		probe.ExpectedOutputs = []GenerationBundleExpectedOutput{{BundlePath: "outputs/" + filepath.Base(e.record.InputPath)}}
	}
	importPath, err := repositoryRelativePath(root, generationImportReceiptPath(e.batchDir, probe))
	if err != nil {
		return nil, err
	}
	b, err := readSnapshotReferencedFile(root, importPath)
	if err != nil {
		return nil, fmt.Errorf("predecessor recovery requires sealed import: %w", err)
	}
	var imported GenerationResultBundleManifest
	if err := decodeStrictBundleJSON(b, &imported); err != nil {
		return nil, err
	}
	archivePath := importedGenerationArchive(locale, imported.GenerationBundleSHA256)
	archive, err := readSnapshotReferencedFile(root, archivePath)
	if err != nil {
		return nil, err
	}
	original, files, err := readGenerationBundle(archive)
	if err != nil {
		return nil, err
	}
	if err := checkImportedGenerationIdentity(imported, original, archive, files); err != nil {
		return nil, err
	}
	if original.Locale != locale || original.BatchID != s.SelectedBatchID || original.Attempt != s.Attempt {
		return nil, fmt.Errorf("predecessor imported batch/attempt mismatch")
	}
	matched := 0
	for _, expected := range original.ExpectedOutputs {
		if expected.UnitID == e.record.UnitID && expected.InstallPath == e.validation.RawResponsePath {
			for _, out := range imported.Outputs {
				if out.BundlePath == expected.BundlePath && out.SHA256 == sum(raw) && out.Size == int64(len(raw)) {
					matched++
				}
			}
		}
	}
	if matched != 1 {
		return nil, fmt.Errorf("predecessor imported raw/attempt exact binding mismatch")
	}
	base, err := repositoryRelativePath(root, e.batchDir)
	if err != nil {
		return nil, err
	}
	manifest, err := readSnapshotReferencedFile(root, base+"/manifest.json")
	if err != nil || !bytes.Equal(manifest, files["formal/manifest.json"]) {
		return nil, fmt.Errorf("predecessor imported formal manifest mismatch")
	}
	input, err := readSnapshotReferencedFile(root, base+"/"+e.record.InputPath)
	if err != nil || sum(input) != e.record.InputSHA256 || !bytes.Equal(input, files["formal/inputs/"+filepath.Base(e.record.InputPath)]) {
		return nil, fmt.Errorf("predecessor imported formal input mismatch")
	}
	return []GlossaryArchiveReference{{Path: importPath, SHA256: sum(b)}, {Path: archivePath, SHA256: sum(archive)}, {Path: base + "/manifest.json", SHA256: sum(manifest)}, {Path: base + "/" + e.record.InputPath, SHA256: sum(input)}}, nil
}
