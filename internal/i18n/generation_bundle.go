package i18n

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	GenerationBundleSchemaVersion       = 1
	GenerationResultBundleSchemaVersion = 1
	FormalGenerationModel               = "gpt-5.6-sol-high"
	FormalChatGPTGenerationModel        = "gpt-5.6-sol-high"
	PreviousCodexGenerationModel        = "gpt-6-luna-high"
)

// TranslationUnitGenerationBundleMaxFiles includes manifest.json. A batch is
// limited to 60 Units, but its source/representation/provenance closure can
// contain many more files. Export and every constituent reader share this
// finite transport budget; it does not change any semantic validation.
const (
	TranslationUnitGenerationBundleMaxFiles       = 8192
	TranslationUnitGenerationBundleMaxBytes int64 = 128 << 20
)

func DefaultGenerationModelForProvider(provider string) (string, error) {
	switch provider {
	case string(RetranslationGeneratorChatGPT):
		return FormalChatGPTGenerationModel, nil
	case string(RetranslationGeneratorCodex):
		return FormalGenerationModel, nil
	default:
		return "", fmt.Errorf("provider must be chatgpt or codex")
	}
}

var translationUnitGenerationAuthorityPaths = []string{
	"docs/LANGUAGE_DEFECT_EXHAUSTION.md",
	"docs/SURFACE_REVISION_TRANSPORT.md",
	"internal/i18n/surface_transport.go",
	"internal/i18n/surface_transport_origin.go",
	"internal/sitecontent/surface_transport.go",
	"internal/i18n/bridge_successor.go",
	"internal/i18n/same_unit_bridge.go",
	"internal/sitecontent/same_unit_bridge.go",
	"internal/i18n/surface_same_unit_bridge.go",
	"internal/sitecontent/surface_same_unit_bridge.go",
	"internal/i18n/comment_revision_transport.go",
	"internal/sitecontent/comment_revision_transport.go",
	"AGENTS.md",
	"docs/SHARED_TRANSLATION_UNIT_LIFECYCLE.md",
	"docs/WORKFLOW_HANDOFF.md",
	"docs/CHATGPT_LANGUAGE_GENERATION.md",
	"docs/CODEX_TRANSLATION.md",
	"docs/RETRANSLATION_RUNBOOK.md",
	"docs/TRANSLATION_TASK_SPEC.md",
	"docs/TRANSLATION_WORKFLOW.md",
}

type GenerationBundleOptions struct {
	Locale                    string
	BatchID                   string
	UnitID                    string
	ApprovedIntervalSelection string
	approvedIntervalSelection []byte // Only canonical reconstruction consumes frozen ZIP bytes.
}

type GenerationBundleExpectedOutput struct {
	BundlePath  string `json:"bundle_path"`
	InstallPath string `json:"install_path"`
	UnitID      string `json:"unit_id"`
}

type GenerationBundleManifest struct {
	SchemaVersion       int                              `json:"schema_version"`
	Kind                string                           `json:"kind"`
	TaskKind            string                           `json:"task_kind"`
	Locale              string                           `json:"locale"`
	BatchID             string                           `json:"batch_id"`
	UnitKind            UnitKind                         `json:"unit_kind"`
	Attempt             int                              `json:"attempt"`
	InputIdentitySHA256 string                           `json:"input_identity_sha256"`
	Files               []TransportBundleFile            `json:"files"`
	ExpectedOutputs     []GenerationBundleExpectedOutput `json:"expected_outputs"`
}

type GenerationResultBundleManifest struct {
	RecoveryExecution      *RecoveryExecutionReceipt `json:"recovery_generation_execution,omitempty"`
	SchemaVersion          int                       `json:"schema_version"`
	Kind                   string                    `json:"kind"`
	TaskKind               string                    `json:"task_kind"`
	Locale                 string                    `json:"locale"`
	BatchID                string                    `json:"batch_id"`
	Attempt                int                       `json:"attempt"`
	GenerationBundleSHA256 string                    `json:"generation_bundle_sha256"`
	InputIdentitySHA256    string                    `json:"input_identity_sha256"`
	Provider               string                    `json:"provider"`
	Model                  string                    `json:"model"`
	GenerationManifest     TransportBundleFile       `json:"generation_manifest"`
	Outputs                []TransportBundleFile     `json:"outputs"`
}

type GenerationImportResult struct {
	RecoveryIdentity            string   `json:"recovery_identity,omitempty"`
	GenerationExecutionIdentity string   `json:"generation_execution_identity,omitempty"`
	Locale                      string   `json:"locale"`
	BatchID                     string   `json:"batch_id"`
	TaskKind                    string   `json:"task_kind"`
	Attempt                     int      `json:"attempt"`
	Provider                    string   `json:"provider"`
	Model                       string   `json:"model"`
	InstalledPaths              []string `json:"installed_paths"`
	InputIdentitySHA256         string   `json:"input_identity_sha256"`
	GenerationBundleSHA256      string   `json:"generation_bundle_sha256"`
}

type generationContext struct {
	CurrentInputs            bool     `json:"current_inputs"`
	RevisionTransportCapable bool     `json:"revision_transport_capable,omitempty"`
	SchemaVersion            int      `json:"schema_version"`
	Kind                     string   `json:"kind"`
	TaskKind                 string   `json:"task_kind"`
	Locale                   string   `json:"locale"`
	BatchID                  string   `json:"batch_id"`
	UnitKind                 UnitKind `json:"unit_kind"`
	Attempt                  int      `json:"attempt"`
	Instruction              string   `json:"instruction"`
	UnitIDs                  []string `json:"unit_ids"`
}

func ExportTranslationUnitGenerationBundle(root string, catalog *Catalog, options GenerationBundleOptions) ([]byte, GenerationBundleManifest, error) {
	if _, _, err := LoadLanguageAuditPolicy(root, options.Locale, nil); err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	if catalog == nil {
		return nil, GenerationBundleManifest{}, errors.New("generation bundle catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	if err := validateBatchID(options.BatchID); err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	if err := RequireNewLanguageGlossaryReview(root, options.Locale); err != nil {
		return nil, GenerationBundleManifest{}, fmt.Errorf("generation bundle requires current Glossary Review coverage: %w", err)
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", options.Locale, options.BatchID)
	manifest, err := readRetranslationProcessManifest(batchDir, options.Locale, options.BatchID)
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	selection, err := generationApprovedIntervalSelection(root, catalog, manifest, options)
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	// Prove the complete caller-selected package before historical revision
	// readers derive their own predecessor catalogs. A receipt must never turn
	// an explicitly wrong package into the right one or infer scope from a batch.
	_, _, _, ordered, err := translationAuthorizationScope(catalog)
	if err != nil {
		return nil, GenerationBundleManifest{}, fmt.Errorf("generation bundle requires complete package scope: %w", err)
	}
	packageMembers := map[string]bool{}
	for _, unit := range ordered {
		packageMembers[unit.ID] = true
	}
	// Immutable saved inputs are transport evidence, not current source
	// authority. Rebuild every formal input before declaring a bundle current.
	glossary, err := LoadGlossary(root, options.Locale)
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	for _, record := range manifest.Units {
		if !packageMembers[record.UnitID] {
			return nil, GenerationBundleManifest{}, fmt.Errorf("generation batch Unit outside selected complete package scope: %s", record.UnitID)
		}
		unit, err := catalog.Unit(record.UnitID)
		if err != nil {
			return nil, GenerationBundleManifest{}, fmt.Errorf("stale formal unit: %w", err)
		}
		if unit.Kind != record.UnitKind || unit.SourcePath != record.SourcePath || unit.SourceSHA256 != record.SourceSHA256 || sum(unit.Source) != record.SourceSHA256 {
			return nil, GenerationBundleManifest{}, fmt.Errorf("%s: stale formal source identity", record.UnitID)
		}
		protected, err := prepareTranslationUnitInput(unit, glossary)
		if err != nil {
			return nil, GenerationBundleManifest{}, err
		}
		input := []byte(protected.Text)
		if manifest.ArtifactEOF == retranslationArtifactEOFSingleLF {
			input = canonicalizeRetranslationArtifactEOF(input)
		}
		if sum(input) != record.InputSHA256 || len(protected.Tokens) != record.ProtectedTokenCount {
			return nil, GenerationBundleManifest{}, fmt.Errorf("%s: stale protected input", record.UnitID)
		}
	}
	manifestRepo := filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, options.BatchID, "manifest.json"))
	manifestData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(manifestRepo)))
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	if err := checkSurfaceTransportBatch(root, options.Locale, options.BatchID, manifest.Units, true); err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	taskKind := "translation-unit-initial"
	for _, unit := range manifest.Units {
		if unit.ReexportReason == RetranslationReexportReasonGlossaryInputStale {
			taskKind = "translation-unit-glossary-stale-recovery"
			break
		}
		if unit.RevisionFeedbackSource != "" {
			taskKind = "translation-unit-revision"
			break
		}
	}
	originTaskKind := taskKind // Machine retry retains the original semantic task.
	if manifest.TransportRecovery != nil {
		taskKind = TransportRecoveryTaskKind
		originTaskKind = "translation-unit-revision"
	}
	attempt := 1
	selected := manifest.Units
	var retryEvidence []struct {
		bundle, repository string
		data               []byte
	}
	if options.UnitID == "" {
		if _, err := os.Lstat(filepath.Join(batchDir, "raw-responses")); err == nil {
			return nil, GenerationBundleManifest{}, fmt.Errorf("formal raw-responses already exists for batch %s", options.BatchID)
		} else if !os.IsNotExist(err) {
			return nil, GenerationBundleManifest{}, err
		}
	} else {
		taskKind = "translation-unit-retry"
		var record *RetranslationBatchUnit
		for i := range manifest.Units {
			if manifest.Units[i].UnitID == options.UnitID {
				record = &manifest.Units[i]
				break
			}
		}
		if record == nil {
			return nil, GenerationBundleManifest{}, fmt.Errorf("translation unit %q is not in batch %s", options.UnitID, options.BatchID)
		}
		selected = []RetranslationBatchUnit{*record}
		unit, err := catalog.Unit(record.UnitID)
		if err != nil {
			return nil, GenerationBundleManifest{}, err
		}
		validationName := strings.TrimSuffix(filepath.Base(record.InputPath), filepath.Ext(record.InputPath)) + ".json"
		validationRepo := filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, options.BatchID, "validation", validationName))
		validationData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(validationRepo)))
		if err != nil {
			return nil, GenerationBundleManifest{}, fmt.Errorf("read current retry validation: %w", err)
		}
		validation, err := decodeRetranslationValidation(validationData, unit)
		if err != nil || validation.SchemaVersion != retranslationProcessSchemaVersion || validation.BatchID != options.BatchID || validation.Locale != options.Locale || validation.UnitID != options.UnitID ||
			(validation.Status != "restore_failed" && validation.Status != "validation_failed") {
			return nil, GenerationBundleManifest{}, fmt.Errorf("translation unit %q is not in retryable restore_failed/validation_failed state", options.UnitID)
		}
		resultRepo := filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, options.BatchID, "result.json"))
		resultData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(resultRepo)))
		if err != nil {
			return nil, GenerationBundleManifest{}, err
		}
		result, err := decodeRetranslationProcessResult(resultData)
		if err != nil || result.SchemaVersion != retranslationProcessSchemaVersion || result.BatchID != options.BatchID || result.Locale != options.Locale || result.UnitCount != len(result.Units) || result.UnitCount != manifest.UnitCount {
			return nil, GenerationBundleManifest{}, fmt.Errorf("retranslation batch %q has incompatible process result", options.BatchID)
		}
		matches := 0
		for _, item := range result.Units {
			if item.UnitID == options.UnitID {
				matches++
				if item.Status != validation.Status {
					return nil, GenerationBundleManifest{}, fmt.Errorf("current validation for %s does not match result.json", options.UnitID)
				}
			}
		}
		if matches != 1 {
			return nil, GenerationBundleManifest{}, fmt.Errorf("result.json must contain exactly one %s", options.UnitID)
		}
		name := filepath.Base(record.InputPath)
		currentAttempt, err := retryValidationAttemptForExtension(validation.RawResponsePath, strings.TrimSuffix(name, filepath.Ext(name)), filepath.Ext(name))
		if err != nil || currentAttempt != validation.Attempt {
			return nil, GenerationBundleManifest{}, fmt.Errorf("current retry attempt identity mismatch for %s", options.UnitID)
		}
		attempt = currentAttempt + 1
		retryEvidence = append(retryEvidence,
			struct {
				bundle, repository string
				data               []byte
			}{"formal/retry/current-validation.json", validationRepo, validationData},
			struct {
				bundle, repository string
				data               []byte
			}{"formal/retry/result.json", resultRepo, resultData},
		)
	}

	type entry = generationBundleInput
	entries := []entry{{"formal/manifest.json", manifestRepo, manifestData}}
	if originTaskKind == "translation-unit-revision" {
		revisionInputs, err := revisionGenerationInputsCore(root, catalog, options.Locale, manifest.Units, nil, true, selection)
		if err != nil {
			return nil, GenerationBundleManifest{}, err
		}
		if err := validateOrdinaryQCRevisionTemplates(root, catalog, glossary, manifest, revisionInputs); err != nil {
			return nil, GenerationBundleManifest{}, err
		}
		entries = append(entries, revisionInputs...)
	}
	if manifest.TransportRecovery != nil {
		entries, err = recoveryGenerationInputs(root, catalog, manifest, entries)
		if err != nil {
			return nil, GenerationBundleManifest{}, err
		}
	}
	if selection != nil && originTaskKind != "translation-unit-revision" {
		return nil, GenerationBundleManifest{}, fmt.Errorf("approved interval selection requires current-validation revision")
	}
	unitIDs := make([]string, 0, len(selected))
	expected := make([]GenerationBundleExpectedOutput, 0, len(selected))
	for _, record := range manifest.Units { // complete formal batch input is indivisible, including retry.
		data, err := os.ReadFile(filepath.Join(batchDir, filepath.FromSlash(record.InputPath)))
		if err != nil {
			return nil, GenerationBundleManifest{}, err
		}
		if sum(data) != record.InputSHA256 {
			return nil, GenerationBundleManifest{}, fmt.Errorf("%s: input hash mismatch", record.UnitID)
		}
		entries = append(entries, entry{filepath.ToSlash(filepath.Join("formal", record.InputPath)), filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, options.BatchID, record.InputPath)), data})
	}
	for _, record := range selected {
		unitIDs = append(unitIDs, record.UnitID)
		name := filepath.Base(record.InputPath)
		install := filepath.ToSlash(filepath.Join("raw-responses", name))
		if taskKind == "translation-unit-retry" {
			ext := filepath.Ext(name)
			flat := strings.TrimSuffix(name, ext)
			install = filepath.ToSlash(filepath.Join("retries", flat, fmt.Sprintf("attempt-%03d%s", attempt, ext)))
		}
		expected = append(expected, GenerationBundleExpectedOutput{BundlePath: filepath.ToSlash(filepath.Join("outputs", name)), InstallPath: install, UnitID: record.UnitID})
	}
	for _, item := range retryEvidence {
		entries = append(entries, entry{item.bundle, item.repository, item.data})
	}
	glossaryRepo := filepath.ToSlash(filepath.Join("locales", options.Locale, "glossary.yaml"))
	glossaryData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(glossaryRepo)))
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	entries = append(entries, entry{"formal/glossary.yaml", glossaryRepo, glossaryData})
	context := generationContext{
		CurrentInputs: true,
		SchemaVersion: GenerationBundleSchemaVersion, Kind: "go-tour-i18n/generation-context",
		TaskKind: taskKind, Locale: options.Locale, BatchID: options.BatchID, UnitKind: manifest.UnitKind,
		Attempt: attempt, UnitIDs: unitIDs,
		Instruction: "Generate only the exact expected output files. Preserve protected tokens exactly and uniquely; add no explanation or Markdown fence; every output must be UTF-8 and end with exactly one LF.",
	}
	if selection != nil {
		context.Instruction += " Approved-interval mode: formal/approved-interval-selection.json explicitly selects opaque whole approved intervals. Use ONLY the supplied formal/protected-candidates projection and its distinct GTI18N_AI1 tokens, exact/unique and in order. Preserve the entire opaque parent. Source children inventory does not grant target leaf mapping. Never translate or reconstruct opaque bytes or ordinary P tokens. Active canonical validation and independent re-QC remain mandatory."
	}
	if originTaskKind == "translation-unit-revision" {
		for _, r := range manifest.Units {
			if r.RevisionFeedbackSource == "surface_review" {
				context.RevisionTransportCapable = true
				break
			}
		}
		context.Instruction += " Semantic task: translation-unit-revision. formal/inputs contains protected English/source input; formal/candidates contains the exact reviewed restored candidate for semantic, structure and finding closure comparison only. Edit formal/protected-candidates/<input basename>, the validated predecessor protected raw response. Outputs must remain protected raw responses with every protected token exactly and uniquely preserved; never output the restored candidate, reconstruct token mapping, retranslate the whole Unit from English, or fetch another candidate from the repository. Close every authorized finding and its affected locality."
		if context.RevisionTransportCapable {
			context.Instruction += " Surface revision transport capability is separately bound by formal/surface-transport-authority and formal/surface-transport. When kind is surface-source-region-projection, formal/protected-candidates is the supplied revision-only bounded editor, not the old raw: edit ONLY its exposed regions, preserving all boundary tokens exactly and in order. All other prose, comments, code, identifiers, directives, literals and link structure remain opaque. A source-bound inline API descriptor grants only its natural-language payload, never the adjacent API identity or code delimiters. formal/historical-protected-candidates retains exact passed raw. No new representation mapping may be invented. The authority does not approve language quality or grant QC/other-Unit scope."
		}
		context.Instruction += " For current-validation comment authority, formal/comment-revision-transport contains a deterministic candidate-specific projection: formal/protected-candidates exposes ONLY the exact authorized teaching-comment bodies and their source-owned bindings; all unrelated prose, code, delimiters, directives and other comments are opaque boundaries. Edit those exposed bodies, retain every projection token exactly and uniquely in its own slot, and use only this projection token set. formal/historical-protected-candidates preserves the original passed raw for reference. The importer/process restores the complete candidate; do not reconstruct token mappings."
		for _, record := range manifest.Units {
			if record.RepresentationBridge != nil {
				context.Instruction += " An explicit same-Unit representation bridge authorizes the ordinary protected revision template in formal/protected-candidates. It restores to the exact same reviewed candidate under the existing artifact framing; formal/representation-bridge freezes its immutable QC, historical restricted raw, original mapping, ordinary ancestor and machine construction authority. Edit this supplied template only, preserve its tokens exactly and uniquely, and follow each unchanged QC rating/finding locally. Do not reconstruct mappings, re-protect candidate prose, treat the bridge as a language result, or expand the finding scope."
				break
			}
		}
		context.Instruction += " If formal/current-validation-predecessor-transport is present, it is a deterministic representation-only capability: use the supplied formal/protected-candidates template and preserve its exact tokens. The sealed Surface/QC ancestor supplies only the decoder; the unchanged current-validation reopen supplies this revision's A predecessor, exact Unit set and machine finding. Historical raw is preserved separately. Do not infer target-language mappings, reuse an ancestor QC finding as current authority, or treat this transport as candidate approval."

		context.Instruction += " For an explicitly reconciled replacement, formal/protected-candidates is a deterministic source-coordinate rebase that restores byte-exactly to the historical candidate; formal/historical-protected-candidates retains the original raw response. The immutable successor reopen and frozen representation proof are in formal/surface-authority. Use only the current token set for outputs."
		for _, record := range manifest.Units {
			if record.RevisionFeedbackSource == RepresentationRevisionFeedback {
				context.Instruction += " Exception for representation_machine authority: formal/protected-candidates is an explicitly INCOMPLETE revision scaffold, not the reviewed candidate. Only the exact unbindable prose intervals listed in formal/representation-revision-scope.json contain verbatim source text. Use formal/candidates and formal/historical-protected-candidates for the actual frozen reviewed translation. Generate a locally complete revision of those intervals and necessary adjacent grammar, preserving original meaning and unrelated reviewed prose. Do not treat source scaffolding as an approved translation or guess a historical protected-token occurrence. Output normal current protected raw with every current token exact and unique; validation and independent re-QC remain mandatory."
				break
			}
		}
		if taskKind == "translation-unit-retry" {
			context.Instruction += fmt.Sprintf(" This is machine retry attempt %d for only the selected restore_failed/validation_failed Unit; retain the original exact revision semantics and finding closure contract.", attempt)
		}
	}
	contextData, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	contextData = append(contextData, '\n')
	entries = append(entries, entry{"generation-context.json", "", contextData})
	for _, repoPath := range append(append([]string(nil), translationUnitGenerationAuthorityPaths...), "locales/"+options.Locale+"/locale.json") {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(repoPath)))
		if err != nil {
			return nil, GenerationBundleManifest{}, fmt.Errorf("read generation authority %s: %w", repoPath, err)
		}
		entries = append(entries, entry{filepath.ToSlash(filepath.Join("authority", repoPath)), repoPath, data})
	}
	contextRefs := map[string]string{}
	for _, id := range unitIDs {
		unit, err := catalog.Unit(id)
		if err != nil {
			return nil, GenerationBundleManifest{}, err
		}
		context, err := translationGenerationContexts(root, options.Locale, unit.Adapter)
		if err != nil {
			return nil, GenerationBundleManifest{}, err
		}
		{
			for _, ref := range context {
				if sha, seen := contextRefs[ref.Path]; seen {
					if sha != ref.SHA256 {
						return nil, GenerationBundleManifest{}, fmt.Errorf("generation context reference conflict")
					}
					continue
				}
				data, err := readCompatibilityFile(root, ref.Path)
				if err != nil || sum(data) != ref.SHA256 {
					return nil, GenerationBundleManifest{}, fmt.Errorf("generation context hash mismatch: %s", ref.Path)
				}
				contextRefs[ref.Path] = ref.SHA256
				entries = append(entries, entry{"context/" + ref.Path, ref.Path, data})
			}
		}
	}
	entries, err = freezeApprovedConstructionDependencies(root, entries)
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	files := make([]TransportBundleFile, 0, len(entries))
	zipEntries := make([]TransportBundleEntry, 0, len(entries))
	var identity strings.Builder
	for _, item := range entries {
		file := NewTransportBundleFile(item.path, item.repositoryPath, item.data)
		files = append(files, file)
		zipEntries = append(zipEntries, TransportBundleEntry{Path: item.path, Data: item.data})
		identity.WriteString(file.BundlePath + "\x00" + file.SHA256 + "\x00")
	}
	manifestOut := GenerationBundleManifest{
		SchemaVersion: GenerationBundleSchemaVersion, Kind: "go-tour-i18n/generation-bundle",
		TaskKind: taskKind, Locale: options.Locale, BatchID: options.BatchID, UnitKind: manifest.UnitKind,
		Attempt: attempt, InputIdentitySHA256: sum([]byte(identity.String())), Files: files, ExpectedOutputs: expected,
	}
	manifestOutData, err := json.MarshalIndent(manifestOut, "", "  ")
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	manifestOutData = append(manifestOutData, '\n')
	if len(zipEntries)+1 > TranslationUnitGenerationBundleMaxFiles {
		return nil, GenerationBundleManifest{}, fmt.Errorf("generation bundle file count %d exceeds %d", len(zipEntries)+1, TranslationUnitGenerationBundleMaxFiles)
	}
	remaining := TranslationUnitGenerationBundleMaxBytes - int64(len(manifestOutData))
	if remaining < 0 {
		return nil, GenerationBundleManifest{}, fmt.Errorf("generation bundle exceeds %d bytes", TranslationUnitGenerationBundleMaxBytes)
	}
	for _, item := range zipEntries {
		if int64(len(item.Data)) > remaining {
			return nil, GenerationBundleManifest{}, fmt.Errorf("generation bundle exceeds %d bytes", TranslationUnitGenerationBundleMaxBytes)
		}
		remaining -= int64(len(item.Data))
	}
	bundle, err := writeTranslationUnitGenerationBundle(manifestOutData, zipEntries)
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	// Fresh entry/exit proof for Surface capability, including newly selected
	// Review/registered-owner heads; a stored receipt is never a cached PASS.
	checkedSurface := map[string]bool{}
	for _, r := range manifest.Units {
		if r.RevisionFeedbackSource == "surface_review" && !checkedSurface[r.RevisionAuthorizationID] {
			if _, _, err := readSurfaceTransport(root, catalog, options.Locale, r.RevisionAuthorizationID); err != nil {
				return nil, GenerationBundleManifest{}, err
			}
			checkedSurface[r.RevisionAuthorizationID] = true
		}
	}
	if err := checkApprovedIntervalInputs(root, entries); err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	if options.ApprovedIntervalSelection != "" {
		b, err := readCompatibilityFile(root, options.ApprovedIntervalSelection)
		if err != nil || !bytes.Equal(b, selection.bytes) {
			return nil, GenerationBundleManifest{}, fmt.Errorf("approved interval selection changed during export")
		}
	}
	return bundle, manifestOut, nil
}

func writeTranslationUnitGenerationBundle(manifest []byte, entries []TransportBundleEntry) ([]byte, error) {
	return writeDeterministicTransportBundle(manifest, entries, zip.Deflate)
}

func decodeGenerationBundle(data []byte) (GenerationBundleManifest, map[string][]byte, error) {
	files, err := ReadTransportBundle(data, TranslationUnitGenerationBundleMaxFiles, TranslationUnitGenerationBundleMaxBytes)
	if err != nil {
		return GenerationBundleManifest{}, nil, err
	}
	var manifest GenerationBundleManifest
	if err := decodeStrictBundleJSON(files["manifest.json"], &manifest); err != nil {
		return manifest, nil, fmt.Errorf("parse generation bundle manifest: %w", err)
	}
	if manifest.SchemaVersion != GenerationBundleSchemaVersion || manifest.Kind != "go-tour-i18n/generation-bundle" || manifest.Locale == "" || manifest.BatchID == "" || len(manifest.ExpectedOutputs) == 0 {
		return manifest, nil, errors.New("generation bundle has incompatible contract")
	}
	if err := ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		return manifest, nil, err
	}
	seen := map[string]bool{}
	for _, output := range manifest.ExpectedOutputs {
		if err := validateTransportBundlePath(output.BundlePath); err != nil {
			return manifest, nil, err
		}
		if !strings.HasPrefix(output.BundlePath, "outputs/") || output.UnitID == "" || output.InstallPath == "" || seen[output.BundlePath] {
			return manifest, nil, errors.New("generation bundle has invalid expected output contract")
		}
		seen[output.BundlePath] = true
	}
	return manifest, files, nil
}

func PackGenerationResultBundle(root string, catalog *Catalog, generationBundleData []byte, provider, model, inputDir string) ([]byte, GenerationResultBundleManifest, error) {
	manifest, files, err := readGenerationBundle(generationBundleData)
	if err != nil {
		return nil, GenerationResultBundleManifest{}, err
	}
	if err := requireFormalGenerationIdentity(provider, model, manifest.BatchID); err != nil {
		return nil, GenerationResultBundleManifest{}, err
	}
	if _, _, err := currentGenerationBundle(root, catalog, generationBundleData); err != nil {
		return nil, GenerationResultBundleManifest{}, err
	}
	var batch RetranslationBatchManifest
	if err := decodeStrictBundleJSON(files["formal/manifest.json"], &batch); err != nil {
		return nil, GenerationResultBundleManifest{}, err
	}
	if batch.TransportRecovery != nil {
		return nil, GenerationResultBundleManifest{}, fmt.Errorf("recovery only supports directory import with Generation execution evidence")
	}
	info, err := os.Lstat(inputDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, GenerationResultBundleManifest{}, fmt.Errorf("result input must be a real directory")
	}
	directoryEntries, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, GenerationResultBundleManifest{}, err
	}
	wantNames := map[string]GenerationBundleExpectedOutput{}
	for _, expected := range manifest.ExpectedOutputs {
		wantNames[filepath.Base(expected.BundlePath)] = expected
	}
	if len(directoryEntries) != len(wantNames) {
		return nil, GenerationResultBundleManifest{}, fmt.Errorf("result file set mismatch: got %d entries, want %d", len(directoryEntries), len(wantNames))
	}
	outputs := make([]TransportBundleFile, 0, len(wantNames))
	zipEntries := []TransportBundleEntry{{Path: "generation-manifest.json", Data: files["manifest.json"]}}
	glossary, err := LoadGlossary(root, manifest.Locale)
	if err != nil {
		return nil, GenerationResultBundleManifest{}, err
	}
	for _, dirEntry := range directoryEntries {
		expected, ok := wantNames[dirEntry.Name()]
		if !ok || dirEntry.IsDir() || dirEntry.Type()&os.ModeSymlink != 0 {
			return nil, GenerationResultBundleManifest{}, fmt.Errorf("unexpected or non-regular result entry %q", dirEntry.Name())
		}
		info, err := dirEntry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil, GenerationResultBundleManifest{}, fmt.Errorf("result entry %q is not a regular file", dirEntry.Name())
		}
		data, err := os.ReadFile(filepath.Join(inputDir, dirEntry.Name()))
		if err != nil {
			return nil, GenerationResultBundleManifest{}, err
		}
		if err := validateGenerationOutputBytes(data); err != nil {
			return nil, GenerationResultBundleManifest{}, fmt.Errorf("%s: %w", dirEntry.Name(), err)
		}
		protected, surface, err := preimportSurfaceGenerationOutput(catalog, manifest, files, expected.UnitID)
		if err != nil {
			return nil, GenerationResultBundleManifest{}, fmt.Errorf("%s: %w", dirEntry.Name(), err)
		}
		if surface {
			err = validatePreparedGenerationOutput(root, catalog, manifest.Locale, expected.UnitID, data, protected, batch.ArtifactEOF)
		} else {
			err = validateTranslationUnitGenerationOutput(root, catalog, glossary, manifest.Locale, manifest.BatchID, expected.UnitID, data)
		}
		if err != nil {
			return nil, GenerationResultBundleManifest{}, fmt.Errorf("%s: %w", dirEntry.Name(), err)
		}
		file := NewTransportBundleFile(expected.BundlePath, "", data)
		outputs = append(outputs, file)
		zipEntries = append(zipEntries, TransportBundleEntry{Path: expected.BundlePath, Data: data})
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].BundlePath < outputs[j].BundlePath })
	resultManifest := GenerationResultBundleManifest{
		SchemaVersion: GenerationResultBundleSchemaVersion, Kind: "go-tour-i18n/generation-result-bundle",
		TaskKind: manifest.TaskKind, Locale: manifest.Locale, BatchID: manifest.BatchID, Attempt: manifest.Attempt,
		GenerationBundleSHA256: sum(generationBundleData), InputIdentitySHA256: manifest.InputIdentitySHA256,
		Provider: provider, Model: model,
		GenerationManifest: NewTransportBundleFile("generation-manifest.json", "", files["manifest.json"]), Outputs: outputs,
	}
	manifestData, err := json.MarshalIndent(resultManifest, "", "  ")
	if err != nil {
		return nil, GenerationResultBundleManifest{}, err
	}
	manifestData = append(manifestData, '\n')
	bundle, err := WriteDeterministicTransportBundle(manifestData, zipEntries)
	if err != nil {
		return nil, GenerationResultBundleManifest{}, err
	}
	if err := checkPreimportSurfaceGenerationContext(root, catalog, manifest, files); err != nil {
		return nil, GenerationResultBundleManifest{}, err
	}
	return bundle, resultManifest, nil
}

func ImportGenerationResultBundle(root string, catalog *Catalog, generationBundleData, resultBundleData []byte) (*GenerationImportResult, error) {
	manifest, generationFiles, err := currentGenerationBundle(root, catalog, generationBundleData)
	if err != nil {
		return nil, err
	}
	resultFiles, err := ReadTransportBundle(resultBundleData, 256, 64<<20)
	if err != nil {
		return nil, err
	}
	var resultManifest GenerationResultBundleManifest
	if err := decodeStrictBundleJSON(resultFiles["manifest.json"], &resultManifest); err != nil {
		return nil, fmt.Errorf("parse generation result manifest: %w", err)
	}
	outputs, err := validateGenerationResult(root, catalog, generationBundleData, manifest, generationFiles, resultFiles, resultManifest)
	if err != nil {
		return nil, err
	}
	return installGenerationOutputs(root, manifest, generationBundleData, generationFiles["manifest.json"], resultManifest.Provider, resultManifest.Model, outputs)
}

// ImportGenerationOutputDirectory validates and installs local generation
// outputs without creating an intermediate Result Bundle ZIP.
func ImportGenerationOutputDirectory(root string, catalog *Catalog, generationBundleData []byte, provider, model, inputDir string) (*GenerationImportResult, error) {
	manifest, generationFiles, err := currentGenerationBundle(root, catalog, generationBundleData)
	if err != nil {
		return nil, err
	}
	if err := requireFormalGenerationIdentity(provider, model, manifest.BatchID); err != nil {
		return nil, err
	}
	outputBytes, err := readExactGenerationOutputDirectory(inputDir, manifest.ExpectedOutputs)
	if err != nil {
		return nil, err
	}
	outputs := make(map[string][]byte, len(outputBytes))
	for _, expected := range manifest.ExpectedOutputs {
		outputs[expected.BundlePath] = outputBytes[filepath.Base(expected.BundlePath)]
	}
	if err := validateGenerationOutputSet(root, catalog, manifest, outputs); err != nil {
		return nil, err
	}
	result, err := installGenerationOutputs(root, manifest, generationBundleData, generationFiles["manifest.json"], provider, model, outputs)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func currentGenerationBundle(root string, catalog *Catalog, generationBundleData []byte) (GenerationBundleManifest, map[string][]byte, error) {
	manifest, files, err := readGenerationBundle(generationBundleData)
	if err != nil {
		return manifest, nil, err
	}
	current, _, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: manifest.Locale, BatchID: manifest.BatchID, UnitID: retryUnitID(manifest), approvedIntervalSelection: files[approvedIntervalSelectionMember]})
	if err != nil {
		return manifest, nil, fmt.Errorf("generation bundle is stale: %w", err)
	}
	if manifest.TaskKind == "translation-unit-retry" {
		current, err = canonicalRetryBundleWithOriginalResult(root, catalog, manifest, files, current)
		if err != nil {
			return manifest, nil, fmt.Errorf("generation bundle is stale: %w", err)
		}
	}
	if !bytes.Equal(current, generationBundleData) {
		return manifest, nil, errors.New("generation bundle is stale or non-canonical")
	}
	return manifest, files, nil
}

func validateGenerationResult(root string, catalog *Catalog, generationBundleData []byte, manifest GenerationBundleManifest, generationFiles, resultFiles map[string][]byte, resultManifest GenerationResultBundleManifest) (map[string][]byte, error) {
	if resultManifest.SchemaVersion != GenerationResultBundleSchemaVersion || resultManifest.Kind != "go-tour-i18n/generation-result-bundle" ||
		resultManifest.TaskKind != manifest.TaskKind || resultManifest.Locale != manifest.Locale || resultManifest.BatchID != manifest.BatchID || resultManifest.Attempt != manifest.Attempt ||
		resultManifest.GenerationBundleSHA256 != sum(generationBundleData) || resultManifest.InputIdentitySHA256 != manifest.InputIdentitySHA256 {
		return nil, errors.New("generation result bundle identity does not match generation bundle")
	}
	if err := requireFormalGenerationIdentity(resultManifest.Provider, resultManifest.Model, manifest.BatchID); err != nil {
		return nil, err
	}
	if resultManifest.GenerationManifest.BundlePath != "generation-manifest.json" || resultManifest.GenerationManifest.SHA256 != sum(generationFiles["manifest.json"]) || resultManifest.GenerationManifest.Size != int64(len(generationFiles["manifest.json"])) {
		return nil, errors.New("generation result bundle embedded manifest identity mismatch")
	}
	inventory := append([]TransportBundleFile{resultManifest.GenerationManifest}, resultManifest.Outputs...)
	if err := ValidateTransportBundleInventory(resultFiles, inventory, true); err != nil {
		return nil, err
	}
	if !bytes.Equal(resultFiles["generation-manifest.json"], generationFiles["manifest.json"]) {
		return nil, errors.New("generation result bundle embeds a different generation manifest")
	}
	outputByPath := make(map[string][]byte, len(resultManifest.Outputs))
	for _, output := range resultManifest.Outputs {
		outputByPath[output.BundlePath] = resultFiles[output.BundlePath]
	}
	if err := validateGenerationOutputSet(root, catalog, manifest, outputByPath); err != nil {
		return nil, err
	}
	return outputByPath, nil
}

func validateGenerationOutputSet(root string, catalog *Catalog, manifest GenerationBundleManifest, outputs map[string][]byte) error {
	if len(outputs) != len(manifest.ExpectedOutputs) {
		return errors.New("generation result bundle output set mismatch")
	}
	for _, expected := range manifest.ExpectedOutputs {
		data, ok := outputs[expected.BundlePath]
		if !ok {
			return fmt.Errorf("generation result is missing %s", expected.BundlePath)
		}
		if err := validateGenerationOutputBytes(data); err != nil {
			return fmt.Errorf("%s: %w", expected.BundlePath, err)
		}
		// Semantic failures belong to process/retry, where every independent
		// Unit is retained and receives its own validation evidence.
		if err := validateGenerationInstallPath(expected.InstallPath); err != nil {
			return err
		}
	}
	return nil
}

func readExactGenerationOutputDirectory(inputDir string, expected []GenerationBundleExpectedOutput) (map[string][]byte, error) {
	info, err := os.Lstat(inputDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("generation input must be a real directory")
	}
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, err
	}
	want := make(map[string]bool, len(expected))
	for _, item := range expected {
		name := filepath.Base(item.BundlePath)
		if name == "." || name == string(filepath.Separator) || want[name] {
			return nil, errors.New("generation bundle has invalid expected output names")
		}
		want[name] = true
	}
	if len(entries) != len(want) {
		return nil, fmt.Errorf("generation output file set mismatch: got %d entries, want %d", len(entries), len(want))
	}
	outputs := make(map[string][]byte, len(want))
	for _, entry := range entries {
		name := entry.Name()
		if !want[name] || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("unexpected or non-regular generation output entry %q", name)
		}
		path := filepath.Join(inputDir, name)
		before, err := os.Lstat(path)
		if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("generation output %q is not a regular non-symlink file", name)
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		opened, statErr := file.Stat()
		if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
			file.Close()
			return nil, fmt.Errorf("generation output %q changed while opening", name)
		}
		data, readErr := io.ReadAll(file)
		after, afterErr := file.Stat()
		closeErr := file.Close()
		if readErr != nil {
			return nil, readErr
		}
		if afterErr != nil || closeErr != nil || !os.SameFile(opened, after) || after.Size() != int64(len(data)) {
			return nil, fmt.Errorf("generation output %q changed while reading", name)
		}
		outputs[name] = data
	}
	return outputs, nil
}

func validateGenerationInstallPath(path string) error {
	if path == "" || filepath.IsAbs(filepath.FromSlash(path)) || filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))) != path || strings.Contains(path, "\\") {
		return fmt.Errorf("unsafe install path %q", path)
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("unsafe install path %q", path)
		}
	}
	return nil
}

func installGenerationOutputs(root string, manifest GenerationBundleManifest, generationBundleData, generationManifestData []byte, provider, model string, outputs map[string][]byte, executions ...*recoveryExecutionInstall) (*GenerationImportResult, error) {
	_, frozen, err := readGenerationBundle(generationBundleData)
	if err != nil {
		return nil, err
	}
	batch, err := decodeRetranslationManifest(frozen["formal/manifest.json"])
	if err != nil {
		return nil, err
	}
	if (manifest.TaskKind == TransportRecoveryTaskKind && batch.TransportRecovery == nil) || (batch.TransportRecovery != nil && manifest.TaskKind != TransportRecoveryTaskKind && manifest.TaskKind != "translation-unit-retry") {
		return nil, fmt.Errorf("recovery task/authority mismatch")
	}
	var execution *recoveryExecutionInstall
	if len(executions) == 1 {
		execution = executions[0]
	}
	if execution != nil && execution.Check == nil {
		return nil, fmt.Errorf("recovery import requires publication CAS guard")
	}
	if (batch.TransportRecovery != nil) != (execution != nil) || len(executions) > 1 {
		return nil, fmt.Errorf("recovery import requires exact new Generation execution provenance")
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", manifest.Locale, manifest.BatchID)
	if err := ensureNoGenerationImportPending(batchDir); err != nil {
		return nil, err
	}
	batchInfo, err := os.Lstat(batchDir)
	if err != nil || !batchInfo.IsDir() || batchInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("generation batch install root must be a real directory")
	}
	if manifest.TaskKind != "translation-unit-retry" {
		for _, name := range []string{"raw-responses", "candidates", "validation", "result.json"} {
			path := filepath.Join(batchDir, name)
			if _, err := os.Lstat(path); err == nil {
				return nil, fmt.Errorf("formal generation batch state %q already exists; refusing import", name)
			} else if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}
	for _, expected := range manifest.ExpectedOutputs {
		if err := validateGenerationInstallPath(expected.InstallPath); err != nil {
			return nil, err
		}
		if err := checkGenerationInstallParents(batchDir, expected.InstallPath); err != nil {
			return nil, err
		}
		destination := filepath.Join(batchDir, filepath.FromSlash(expected.InstallPath))
		if _, err := os.Lstat(destination); err == nil {
			return nil, fmt.Errorf("formal generation output already exists: %s", expected.InstallPath)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	receipt := GenerationResultBundleManifest{
		SchemaVersion: GenerationResultBundleSchemaVersion, Kind: "go-tour-i18n/generation-result-bundle", TaskKind: manifest.TaskKind,
		Locale: manifest.Locale, BatchID: manifest.BatchID, Attempt: manifest.Attempt,
		GenerationBundleSHA256: sum(generationBundleData), InputIdentitySHA256: manifest.InputIdentitySHA256,
		Provider: provider, Model: model,
		GenerationManifest: NewTransportBundleFile("generation-manifest.json", "", generationManifestData),
	}
	if execution != nil {
		receipt.RecoveryExecution = &execution.Receipt
		for path := range execution.Members {
			if err := recoveryRequireAbsent(root, path); err != nil {
				return nil, err
			}
		}
	}
	for _, expected := range manifest.ExpectedOutputs {
		receipt.Outputs = append(receipt.Outputs, NewTransportBundleFile(expected.BundlePath, "", outputs[expected.BundlePath]))
	}
	sort.Slice(receipt.Outputs, func(i, j int) bool { return receipt.Outputs[i].BundlePath < receipt.Outputs[j].BundlePath })
	receiptData, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return nil, err
	}
	receiptData = append(receiptData, '\n')
	receiptPath := generationImportReceiptPath(batchDir, manifest)
	if _, err := os.Lstat(receiptPath); err == nil {
		return nil, fmt.Errorf("generation import provenance already exists: %s; inspect batch state before retrying", filepath.Base(receiptPath))
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := writeCompatibilityImmutable(root, importedGenerationArchive(manifest.Locale, sum(generationBundleData)), generationBundleData); err != nil {
		return nil, err
	}
	var installed []string
	var installedRawInfo os.FileInfo
	if err := ensureGenerationImportDirectory(batchDir); err != nil {
		return nil, err
	}
	pendingPath := filepath.Join(batchDir, "generation-imports", "import.pending")
	if err := writeNewGenerationFile(pendingPath, receiptData); err != nil {
		return nil, fmt.Errorf("another generation import may be in progress; inspect batch state: %w", err)
	}
	keepPending := false
	defer func() {
		if !keepPending {
			_ = os.Remove(pendingPath)
		}
	}()
	if execution != nil {
		// This evidence and the outputs share the existing pending marker. Until
		// the final receipt commits, no process/retry consumer may observe a
		// successful import; an interrupted install is retained for diagnosis.
		keepPending = true
		var paths []string
		for path := range execution.Members {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			if err := publishRecoverySealedFile(root, path, execution.Members[path], false); err != nil {
				return nil, err
			}
		}
		if err := execution.Check(); err != nil {
			return nil, fmt.Errorf("mutation=unknown: recovery import CAS failed; pending=%s: %w", pendingPath, err)
		}
	}
	rollback := func() error {
		if manifest.TaskKind != "translation-unit-retry" {
			finalDir := filepath.Join(batchDir, "raw-responses")
			info, err := os.Lstat(finalDir)
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil || installedRawInfo == nil || !os.SameFile(installedRawInfo, info) || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("formal raw-responses changed after install; refusing unsafe rollback")
			}
			entries, err := os.ReadDir(finalDir)
			if err != nil || len(entries) != len(manifest.ExpectedOutputs) {
				return errors.New("formal raw-responses inventory changed after install; refusing unsafe rollback")
			}
			for _, expected := range manifest.ExpectedOutputs {
				name := filepath.Base(expected.InstallPath)
				fileInfo, err := os.Lstat(filepath.Join(finalDir, name))
				if err != nil || !fileInfo.Mode().IsRegular() || fileInfo.Mode()&os.ModeSymlink != 0 {
					return errors.New("formal raw-response file changed after install; refusing unsafe rollback")
				}
				data, err := os.ReadFile(filepath.Join(finalDir, name))
				if err != nil || !bytes.Equal(data, outputs[expected.BundlePath]) {
					return errors.New("formal raw-response bytes changed after install; refusing unsafe rollback")
				}
			}
			return os.RemoveAll(finalDir)
		}
		for _, path := range installed {
			fullPath := filepath.Join(batchDir, filepath.FromSlash(path))
			info, err := os.Lstat(fullPath)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("retry output %s changed after install; refusing unsafe rollback", path)
			}
			data, err := os.ReadFile(fullPath)
			if err != nil || !bytes.Equal(data, outputs[manifest.ExpectedOutputs[0].BundlePath]) {
				return fmt.Errorf("retry output %s bytes changed after install; refusing unsafe rollback", path)
			}
			if err := os.Remove(fullPath); err != nil {
				return err
			}
		}
		return nil
	}
	if manifest.TaskKind == "translation-unit-retry" {
		expected := manifest.ExpectedOutputs[0]
		destination := filepath.Join(batchDir, filepath.FromSlash(expected.InstallPath))
		if err := createGenerationInstallParents(batchDir, expected.InstallPath); err != nil {
			return nil, err
		}
		if err := writeNewGenerationFile(destination, outputs[expected.BundlePath]); err != nil {
			return nil, err
		}
		installed = append(installed, expected.InstallPath)
	} else {
		finalDir := filepath.Join(batchDir, "raw-responses")
		staging, err := os.MkdirTemp(batchDir, ".raw-responses.import-*")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(staging)
		for _, expected := range manifest.ExpectedOutputs {
			name := filepath.Base(expected.InstallPath)
			if err := writeNewGenerationFile(filepath.Join(staging, name), outputs[expected.BundlePath]); err != nil {
				return nil, err
			}
			installed = append(installed, expected.InstallPath)
		}
		installed = installed[:0]
		if _, err := os.Lstat(finalDir); err == nil {
			return nil, fmt.Errorf("formal raw-responses already exists for batch %s", manifest.BatchID)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		if err := renameGenerationDirNoReplace(staging, finalDir); err != nil {
			if os.IsExist(err) {
				return nil, fmt.Errorf("formal raw-responses already exists for batch %s", manifest.BatchID)
			}
			return nil, err
		}
		installedRawInfo, err = os.Lstat(finalDir)
		if err != nil {
			keepPending = true
			return nil, fmt.Errorf("raw-responses installed but its state could not be confirmed; inspect before retrying: %w", err)
		}
		for _, expected := range manifest.ExpectedOutputs {
			installed = append(installed, expected.InstallPath)
		}
	}
	if execution != nil {
		if err := execution.Check(); err != nil {
			return nil, fmt.Errorf("mutation=unknown: recovery import CAS failed; pending=%s: %w", pendingPath, err)
		}
	}
	if err := writeNewGenerationFile(receiptPath, receiptData); err != nil {
		if rollbackErr := rollback(); rollbackErr != nil {
			keepPending = true
			return nil, fmt.Errorf("install provenance failed (%v); formal output rollback is incomplete; inspect pending import state: %w", err, rollbackErr)
		}
		return nil, fmt.Errorf("install generation provenance: %w", err)
	}
	if err := os.Remove(pendingPath); err != nil {
		keepPending = true
		return nil, fmt.Errorf("generation outputs and provenance were installed but pending marker remains; inspect actual batch state before retrying: %w", err)
	}
	keepPending = true // already removed; suppress deferred cleanup.
	installedPaths := make([]string, 0, len(installed))
	for _, path := range installed {
		installedPaths = append(installedPaths, filepath.ToSlash(filepath.Join("data", "retranslation-runs", manifest.Locale, manifest.BatchID, path)))
	}
	sort.Strings(installedPaths)
	imported := &GenerationImportResult{
		Locale: manifest.Locale, BatchID: manifest.BatchID, TaskKind: manifest.TaskKind, Attempt: manifest.Attempt,
		Provider: provider, Model: model, InstalledPaths: installedPaths, InputIdentitySHA256: manifest.InputIdentitySHA256, GenerationBundleSHA256: sum(generationBundleData),
	}
	if execution != nil {
		imported.RecoveryIdentity = batch.TransportRecovery.Identity
		imported.GenerationExecutionIdentity = execution.Receipt.Identity
	}
	return imported, nil
}

func ensureNoGenerationImportPending(batchDir string) error {
	pending := filepath.Join(batchDir, "generation-imports", "import.pending")
	if _, err := os.Lstat(pending); err == nil {
		return errors.New("generation import has pending or unknown mutation state; inspect raw responses, retries, and provenance before continuing")
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func ensureGenerationImportDirectory(batchDir string) error {
	directory := filepath.Join(batchDir, "generation-imports")
	if err := os.Mkdir(directory, 0755); err != nil && !os.IsExist(err) {
		return err
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("generation import provenance path must be a real directory")
	}
	return nil
}

func generationImportReceiptPath(batchDir string, manifest GenerationBundleManifest) string {
	if manifest.TaskKind == "translation-unit-retry" {
		expected := manifest.ExpectedOutputs[0]
		name := filepath.Base(filepath.FromSlash(expected.BundlePath))
		flatID := strings.TrimSuffix(name, filepath.Ext(name))
		return filepath.Join(batchDir, "generation-imports", fmt.Sprintf("%s-attempt-%03d-manifest.json", flatID, manifest.Attempt))
	}
	return filepath.Join(batchDir, "generation-imports", "initial-manifest.json")
}

func checkGenerationInstallParents(batchDir, relative string) error {
	parts := strings.Split(filepath.ToSlash(filepath.Dir(filepath.FromSlash(relative))), "/")
	current := batchDir
	for _, part := range parts {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe install parent %q", current)
		}
	}
	return nil
}

func createGenerationInstallParents(batchDir, relative string) error {
	parts := strings.Split(filepath.ToSlash(filepath.Dir(filepath.FromSlash(relative))), "/")
	current := batchDir
	for _, part := range parts {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		if err := os.Mkdir(current, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe install parent %q", current)
		}
	}
	return nil
}

func writeNewGenerationFile(destination string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".generation-import-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err = temporary.Write(data); err == nil {
		err = temporary.Chmod(0644)
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Link(temporaryPath, destination); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("formal generation output already exists: %s", destination)
		}
		return err
	}
	return nil
}

func retryUnitID(manifest GenerationBundleManifest) string {
	if manifest.TaskKind == "translation-unit-retry" && len(manifest.ExpectedOutputs) == 1 {
		return manifest.ExpectedOutputs[0].UnitID
	}
	return ""
}

func requireFormalGenerationIdentity(provider, model, batchID string) error {
	if provider != string(RetranslationGeneratorChatGPT) && provider != string(RetranslationGeneratorCodex) {
		return fmt.Errorf("provider must be chatgpt or codex")
	}
	modelAccepted := false
	switch provider {
	case string(RetranslationGeneratorChatGPT):
		modelAccepted = model == FormalChatGPTGenerationModel
	case string(RetranslationGeneratorCodex):
		modelAccepted = model == FormalGenerationModel || model == PreviousCodexGenerationModel
	}
	if !modelAccepted {
		return fmt.Errorf("model %s is not an accepted formal model for provider %s", model, provider)
	}
	if strings.HasPrefix(batchID, "chatgpt-") && provider != "chatgpt" || strings.HasPrefix(batchID, "codex-") && provider != "codex" {
		return fmt.Errorf("provider %s does not match batch provenance %s", provider, batchID)
	}
	return nil
}

func validateGenerationOutputBytes(data []byte) error {
	if !utf8.Valid(data) || bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return errors.New("output must be UTF-8 without BOM")
	}
	if bytes.ContainsRune(data, '\x00') {
		return errors.New("output must not contain NUL")
	}
	if err := validateRetranslationArtifactEOF(data); err != nil {
		return err
	}
	return nil
}

func validateTranslationUnitGenerationOutput(root string, catalog *Catalog, glossary *Glossary, locale, batchID, unitID string, data []byte) error {
	unit, err := catalog.Unit(unitID)
	if err != nil {
		return err
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", locale, batchID)
	batchManifest, err := readRetranslationProcessManifest(batchDir, locale, batchID)
	if err != nil {
		return err
	}
	var record *RetranslationBatchUnit
	for i := range batchManifest.Units {
		if batchManifest.Units[i].UnitID == unit.ID {
			record = &batchManifest.Units[i]
		}
	}
	if record == nil {
		return fmt.Errorf("output Unit absent from batch")
	}
	protected, err := prepareInvocationRetranslationOutput(root, catalog, glossary, locale, batchManifest.Units, *record, batchID)
	if err != nil {
		return err
	}
	return validatePreparedGenerationOutput(root, catalog, locale, unitID, data, protected, batchManifest.ArtifactEOF)
}

func validatePreparedGenerationOutput(root string, catalog *Catalog, locale, unitID string, data []byte, protected protectedTranslation, artifactEOF string) error {
	restored, failures := protected.restore(string(data))
	if len(failures) != 0 {
		return fmt.Errorf("protected output validation failed: %s", strings.Join(failures, "; "))
	}
	candidate := []byte(restored)
	if artifactEOF == retranslationArtifactEOFSingleLF {
		candidate = canonicalizeRetranslationArtifactEOF(candidate)
	}
	if err := ValidateTranslationUnitCandidate(root, catalog, unitID, locale, candidate); err != nil {
		return fmt.Errorf("staged output machine validation failed: %w", err)
	}
	return nil
}

func decodeStrictBundleJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("multiple JSON values")
	}
	return nil
}

func pathWithinRoot(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
