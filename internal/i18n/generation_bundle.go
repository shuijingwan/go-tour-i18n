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
	Locale                 string   `json:"locale"`
	BatchID                string   `json:"batch_id"`
	TaskKind               string   `json:"task_kind"`
	Attempt                int      `json:"attempt"`
	Provider               string   `json:"provider"`
	Model                  string   `json:"model"`
	InstalledPaths         []string `json:"installed_paths"`
	InputIdentitySHA256    string   `json:"input_identity_sha256"`
	GenerationBundleSHA256 string   `json:"generation_bundle_sha256"`
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
	current, _, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: manifest.Locale, BatchID: manifest.BatchID, UnitID: retryUnitID(manifest)})
	if err != nil {
		return manifest, nil, fmt.Errorf("generation bundle is stale: %w", err)
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
	glossary, err := LoadGlossary(root, manifest.Locale)
	if err != nil {
		return err
	}
	for _, expected := range manifest.ExpectedOutputs {
		data, ok := outputs[expected.BundlePath]
		if !ok {
			return fmt.Errorf("generation result is missing %s", expected.BundlePath)
		}
		if err := validateGenerationOutputBytes(data); err != nil {
			return fmt.Errorf("%s: %w", expected.BundlePath, err)
		}
		if err := validateTranslationUnitGenerationOutput(root, catalog, glossary, manifest.Locale, manifest.BatchID, expected.UnitID, data); err != nil {
			return fmt.Errorf("%s: %w", expected.BundlePath, err)
		}
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

func installGenerationOutputs(root string, manifest GenerationBundleManifest, generationBundleData, generationManifestData []byte, provider, model string, outputs map[string][]byte) (*GenerationImportResult, error) {
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
	return &GenerationImportResult{
		Locale: manifest.Locale, BatchID: manifest.BatchID, TaskKind: manifest.TaskKind, Attempt: manifest.Attempt,
		Provider: provider, Model: model, InstalledPaths: installedPaths, InputIdentitySHA256: manifest.InputIdentitySHA256, GenerationBundleSHA256: sum(generationBundleData),
	}, nil
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
