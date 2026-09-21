package i18n

import (
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
)

var translationUnitGenerationAuthorityPaths = []string{
	"AGENTS.md",
	"docs/CHATGPT_LANGUAGE_GENERATION.md",
	"docs/CODEX_TRANSLATION.md",
	"docs/RETRANSLATION_RUNBOOK.md",
	"docs/TRANSLATION_TASK_SPEC.md",
	"docs/TRANSLATION_WORKFLOW.md",
}

type GenerationBundleOptions struct {
	Locale  string
	BatchID string
	UnitID  string
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
	SchemaVersion          int                   `json:"schema_version"`
	Kind                   string                `json:"kind"`
	TaskKind               string                `json:"task_kind"`
	Locale                 string                `json:"locale"`
	BatchID                string                `json:"batch_id"`
	Attempt                int                   `json:"attempt"`
	GenerationBundleSHA256 string                `json:"generation_bundle_sha256"`
	InputIdentitySHA256    string                `json:"input_identity_sha256"`
	Provider               string                `json:"provider"`
	Model                  string                `json:"model"`
	GenerationManifest     TransportBundleFile   `json:"generation_manifest"`
	Outputs                []TransportBundleFile `json:"outputs"`
}

type GenerationImportResult struct {
	Locale              string   `json:"locale"`
	BatchID             string   `json:"batch_id"`
	TaskKind            string   `json:"task_kind"`
	Attempt             int      `json:"attempt"`
	Provider            string   `json:"provider"`
	Model               string   `json:"model"`
	InstalledPaths      []string `json:"installed_paths"`
	InputIdentitySHA256 string   `json:"input_identity_sha256"`
}

type generationContext struct {
	SchemaVersion int      `json:"schema_version"`
	Kind          string   `json:"kind"`
	TaskKind      string   `json:"task_kind"`
	Locale        string   `json:"locale"`
	BatchID       string   `json:"batch_id"`
	UnitKind      UnitKind `json:"unit_kind"`
	Attempt       int      `json:"attempt"`
	Instruction   string   `json:"instruction"`
	UnitIDs       []string `json:"unit_ids"`
}

func ExportTranslationUnitGenerationBundle(root string, catalog *Catalog, options GenerationBundleOptions) ([]byte, GenerationBundleManifest, error) {
	if catalog == nil {
		return nil, GenerationBundleManifest{}, errors.New("generation bundle catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	if err := validateBatchID(options.BatchID); err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	if err := RequireCurrentGlossaryReview(root, options.Locale); err != nil {
		return nil, GenerationBundleManifest{}, fmt.Errorf("generation bundle requires current Glossary Review coverage: %w", err)
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", options.Locale, options.BatchID)
	manifest, err := readRetranslationProcessManifest(batchDir, options.Locale, options.BatchID)
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	manifestRepo := filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, options.BatchID, "manifest.json"))
	manifestData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(manifestRepo)))
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	taskKind := "translation-unit-initial"
	for _, unit := range manifest.Units {
		if unit.RevisionFeedbackSource != "" {
			taskKind = "translation-unit-revision"
			break
		}
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

	type entry struct {
		path, repositoryPath string
		data                 []byte
	}
	entries := []entry{{"formal/manifest.json", manifestRepo, manifestData}}
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
		SchemaVersion: GenerationBundleSchemaVersion, Kind: "go-tour-i18n/generation-context",
		TaskKind: taskKind, Locale: options.Locale, BatchID: options.BatchID, UnitKind: manifest.UnitKind,
		Attempt: attempt, UnitIDs: unitIDs,
		Instruction: "Generate only the exact expected output files. Preserve protected tokens exactly and uniquely; add no explanation or Markdown fence; every output must be UTF-8 and end with exactly one LF.",
	}
	contextData, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	contextData = append(contextData, '\n')
	entries = append(entries, entry{"generation-context.json", "", contextData})
	for _, repoPath := range translationUnitGenerationAuthorityPaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(repoPath)))
		if err != nil {
			return nil, GenerationBundleManifest{}, fmt.Errorf("read generation authority %s: %w", repoPath, err)
		}
		entries = append(entries, entry{filepath.ToSlash(filepath.Join("authority", repoPath)), repoPath, data})
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
	bundle, err := WriteDeterministicTransportBundle(manifestOutData, zipEntries)
	if err != nil {
		return nil, GenerationBundleManifest{}, err
	}
	return bundle, manifestOut, nil
}

func readGenerationBundle(data []byte) (GenerationBundleManifest, map[string][]byte, error) {
	files, err := ReadTransportBundle(data, 512, 64<<20)
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
	current, _, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: manifest.Locale, BatchID: manifest.BatchID, UnitID: retryUnitID(manifest)})
	if err != nil || !bytes.Equal(current, generationBundleData) {
		if err != nil {
			return nil, GenerationResultBundleManifest{}, fmt.Errorf("generation bundle is stale: %w", err)
		}
		return nil, GenerationResultBundleManifest{}, errors.New("generation bundle is stale or non-canonical")
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
		if err := validateTranslationUnitGenerationOutput(root, catalog, glossary, manifest.Locale, manifest.BatchID, expected.UnitID, data); err != nil {
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
	return bundle, resultManifest, nil
}

func ImportGenerationResultBundle(root string, catalog *Catalog, generationBundleData, resultBundleData []byte) (*GenerationImportResult, error) {
	manifest, generationFiles, err := readGenerationBundle(generationBundleData)
	if err != nil {
		return nil, err
	}
	current, _, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: manifest.Locale, BatchID: manifest.BatchID, UnitID: retryUnitID(manifest)})
	if err != nil || !bytes.Equal(current, generationBundleData) {
		if err != nil {
			return nil, fmt.Errorf("generation bundle is stale: %w", err)
		}
		return nil, errors.New("generation bundle is stale or non-canonical")
	}
	resultFiles, err := ReadTransportBundle(resultBundleData, 256, 64<<20)
	if err != nil {
		return nil, err
	}
	var resultManifest GenerationResultBundleManifest
	if err := decodeStrictBundleJSON(resultFiles["manifest.json"], &resultManifest); err != nil {
		return nil, fmt.Errorf("parse generation result manifest: %w", err)
	}
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
	outputByPath := map[string]TransportBundleFile{}
	for _, output := range resultManifest.Outputs {
		outputByPath[output.BundlePath] = output
	}
	if len(outputByPath) != len(manifest.ExpectedOutputs) {
		return nil, errors.New("generation result bundle output set mismatch")
	}
	glossary, err := LoadGlossary(root, manifest.Locale)
	if err != nil {
		return nil, err
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", manifest.Locale, manifest.BatchID)
	installed := make([]string, 0, len(manifest.ExpectedOutputs))
	for _, expected := range manifest.ExpectedOutputs {
		if _, ok := outputByPath[expected.BundlePath]; !ok {
			return nil, fmt.Errorf("generation result bundle is missing %s", expected.BundlePath)
		}
		data := resultFiles[expected.BundlePath]
		if err := validateGenerationOutputBytes(data); err != nil {
			return nil, fmt.Errorf("%s: %w", expected.BundlePath, err)
		}
		if err := validateTranslationUnitGenerationOutput(root, catalog, glossary, manifest.Locale, manifest.BatchID, expected.UnitID, data); err != nil {
			return nil, fmt.Errorf("%s: %w", expected.BundlePath, err)
		}
		destination := filepath.Join(batchDir, filepath.FromSlash(expected.InstallPath))
		if !pathWithinRoot(batchDir, destination) {
			return nil, fmt.Errorf("unsafe install path %q", expected.InstallPath)
		}
		if _, err := os.Lstat(destination); err == nil {
			return nil, fmt.Errorf("formal generation output already exists: %s", expected.InstallPath)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if manifest.TaskKind == "translation-unit-retry" {
		expected := manifest.ExpectedOutputs[0]
		destination := filepath.Join(batchDir, filepath.FromSlash(expected.InstallPath))
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return nil, err
		}
		temporary, err := os.CreateTemp(filepath.Dir(destination), ".generation-import-*")
		if err != nil {
			return nil, err
		}
		temporaryPath := temporary.Name()
		defer os.Remove(temporaryPath)
		if _, err = temporary.Write(resultFiles[expected.BundlePath]); err == nil {
			err = temporary.Chmod(0644)
		}
		if closeErr := temporary.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return nil, err
		}
		if err := os.Link(temporaryPath, destination); err != nil {
			if os.IsExist(err) {
				return nil, fmt.Errorf("formal generation output already exists: %s", expected.InstallPath)
			}
			return nil, err
		}
		installed = append(installed, filepath.ToSlash(filepath.Join("data", "retranslation-runs", manifest.Locale, manifest.BatchID, expected.InstallPath)))
	} else {
		finalDir := filepath.Join(batchDir, "raw-responses")
		staging, err := os.MkdirTemp(batchDir, ".raw-responses.import-*")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(staging)
		for _, expected := range manifest.ExpectedOutputs {
			name := filepath.Base(expected.InstallPath)
			if err := os.WriteFile(filepath.Join(staging, name), resultFiles[expected.BundlePath], 0644); err != nil {
				return nil, err
			}
			installed = append(installed, filepath.ToSlash(filepath.Join("data", "retranslation-runs", manifest.Locale, manifest.BatchID, expected.InstallPath)))
		}
		if err := os.Rename(staging, finalDir); err != nil {
			if os.IsExist(err) {
				return nil, fmt.Errorf("formal raw-responses already exists for batch %s", manifest.BatchID)
			}
			return nil, err
		}
	}
	sort.Strings(installed)
	return &GenerationImportResult{
		Locale: manifest.Locale, BatchID: manifest.BatchID, TaskKind: manifest.TaskKind, Attempt: manifest.Attempt,
		Provider: resultManifest.Provider, Model: resultManifest.Model, InstalledPaths: installed, InputIdentitySHA256: manifest.InputIdentitySHA256,
	}, nil
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
	if model != FormalGenerationModel {
		return fmt.Errorf("formal generation model must be %s", FormalGenerationModel)
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
	protected, err := prepareTranslationUnitInput(unit, glossary)
	if err != nil {
		return err
	}
	restored, failures := protected.restore(string(data))
	if len(failures) != 0 {
		return fmt.Errorf("protected output validation failed: %s", strings.Join(failures, "; "))
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", locale, batchID)
	batchManifest, err := readRetranslationProcessManifest(batchDir, locale, batchID)
	if err != nil {
		return err
	}
	candidate := []byte(restored)
	if batchManifest.ArtifactEOF == retranslationArtifactEOFSingleLF {
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
