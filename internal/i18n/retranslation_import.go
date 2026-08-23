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
	"strings"
)

type RetranslationImportOptions struct {
	Locale  string
	BatchID string
	Archive string
}

type RetranslationImportResult struct {
	Locale       string `json:"locale"`
	BatchID      string `json:"batch_id"`
	UnitCount    int    `json:"unit_count"`
	RawDirectory string `json:"raw_directory"`
}

type retranslationResultFile struct {
	BatchID string                        `json:"batch_id"`
	Locale  string                        `json:"locale"`
	Units   []retranslationResultFileUnit `json:"units"`
}

type retranslationResultFileUnit struct {
	UnitID  string `json:"unit_id"`
	Content string `json:"content"`
}

// ImportRetranslationResult imports one or more raw Page responses from a
// translation-result.json artifact. Results may be imported incrementally into
// an otherwise unprocessed batch.
func ImportRetranslationResult(root, resultPath string) (*RetranslationImportResult, error) {
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, fmt.Errorf("read translation result: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var resultFile retranslationResultFile
	if err := decoder.Decode(&resultFile); err != nil {
		return nil, fmt.Errorf("decode translation result: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, fmt.Errorf("decode translation result: %w", err)
	}
	if resultFile.BatchID == "" || resultFile.Locale == "" || len(resultFile.Units) == 0 {
		return nil, errors.New("translation result batch_id, locale, and units are required")
	}
	if err := ValidateLocaleName(resultFile.Locale); err != nil {
		return nil, err
	}
	if err := validateBatchID(resultFile.BatchID); err != nil {
		return nil, err
	}

	batchDir, manifest, err := prepareRetranslationResultImport(root, resultFile.Locale, resultFile.BatchID)
	if err != nil {
		return nil, err
	}
	expected := make(map[string]bool, len(manifest.Units))
	for _, unit := range manifest.Units {
		name := filepath.Base(filepath.FromSlash(unit.InputPath))
		wantInputPath := filepath.ToSlash(filepath.Join("inputs", retranslationUnitInputName(&TranslationUnit{ID: unit.UnitID, Kind: unit.UnitKind})))
		if unit.InputPath != wantInputPath || filepath.Ext(name) != ".article" {
			return nil, fmt.Errorf("%s: retranslation import requires a canonical Page .article input", unit.UnitID)
		}
		expected[strings.TrimSuffix(name, ".article")] = true
	}
	responses := make(map[string]string, len(resultFile.Units))
	for _, unit := range resultFile.Units {
		if unit.UnitID == "" || !expected[unit.UnitID] {
			return nil, fmt.Errorf("unexpected translation result unit_id %q", unit.UnitID)
		}
		if _, duplicate := responses[unit.UnitID]; duplicate {
			return nil, fmt.Errorf("duplicate translation result unit_id %q", unit.UnitID)
		}
		responses[unit.UnitID] = unit.Content
	}

	rawDir := filepath.Join(batchDir, "raw-responses")
	for unitID := range responses {
		if _, err := os.Stat(filepath.Join(rawDir, unitID+".article")); err == nil {
			return nil, fmt.Errorf("retranslation batch %q already has raw response %q", resultFile.BatchID, unitID)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect retranslation raw response: %w", err)
		}
	}
	staging, err := os.MkdirTemp(batchDir, ".result-import-staging-")
	if err != nil {
		return nil, fmt.Errorf("create translation result import staging: %w", err)
	}
	defer os.RemoveAll(staging)
	for unitID, content := range responses {
		if err := os.WriteFile(filepath.Join(staging, unitID+".article"), []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("stage translation result raw response %q: %w", unitID, err)
		}
	}
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		return nil, fmt.Errorf("create retranslation raw response directory: %w", err)
	}
	for unitID := range responses {
		if err := os.Rename(filepath.Join(staging, unitID+".article"), filepath.Join(rawDir, unitID+".article")); err != nil {
			return nil, fmt.Errorf("commit translation result raw response %q: %w", unitID, err)
		}
	}
	rawDirectory, err := repositoryRelativePath(root, rawDir)
	if err != nil {
		return nil, err
	}
	return &RetranslationImportResult{Locale: resultFile.Locale, BatchID: resultFile.BatchID, UnitCount: len(responses), RawDirectory: rawDirectory}, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func prepareRetranslationResultImport(root, locale, batchID string) (string, *RetranslationBatchManifest, error) {
	batchDir := filepath.Join(root, "data", "retranslation-runs", locale, batchID)
	if info, err := os.Stat(batchDir); err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return "", nil, fmt.Errorf("inspect retranslation batch %q: %w", batchID, err)
	}
	manifest, err := readRetranslationProcessManifest(batchDir, locale, batchID)
	if err != nil {
		return "", nil, err
	}
	for _, name := range []string{"result.json", "candidates", "validation"} {
		if _, err := os.Stat(filepath.Join(batchDir, name)); err == nil {
			return "", nil, fmt.Errorf("retranslation batch %q has existing process output %q", batchID, name)
		} else if !os.IsNotExist(err) {
			return "", nil, fmt.Errorf("inspect retranslation batch output: %w", err)
		}
	}
	return batchDir, manifest, nil
}

// ImportRetranslationRawResponses imports a complete Page raw-response ZIP
// artifact into an unprocessed batch. It deliberately does not restore,
// validate, or otherwise process the imported responses.
func ImportRetranslationRawResponses(root string, options RetranslationImportOptions) (*RetranslationImportResult, error) {
	if options.Locale == "" || options.BatchID == "" || options.Archive == "" {
		return nil, errors.New("retranslation import locale, batch_id, and archive are required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	if err := validateBatchID(options.BatchID); err != nil {
		return nil, err
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", options.Locale, options.BatchID)
	if info, err := os.Stat(batchDir); err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return nil, fmt.Errorf("inspect retranslation batch %q: %w", options.BatchID, err)
	}
	manifest, err := readRetranslationProcessManifest(batchDir, options.Locale, options.BatchID)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"result.json", "candidates", "validation", "raw-responses"} {
		if _, err := os.Stat(filepath.Join(batchDir, name)); err == nil {
			if name == "raw-responses" {
				return nil, fmt.Errorf("retranslation batch %q already has raw responses", options.BatchID)
			}
			return nil, fmt.Errorf("retranslation batch %q has existing process output %q", options.BatchID, name)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect retranslation batch output: %w", err)
		}
	}

	expected := make(map[string]bool, len(manifest.Units))
	for _, unit := range manifest.Units {
		name := filepath.Base(filepath.FromSlash(unit.InputPath))
		wantInputPath := filepath.ToSlash(filepath.Join("inputs", retranslationUnitInputName(&TranslationUnit{ID: unit.UnitID, Kind: unit.UnitKind})))
		if unit.InputPath != wantInputPath || filepath.Ext(name) != ".article" {
			return nil, fmt.Errorf("%s: retranslation import requires a canonical Page .article input", unit.UnitID)
		}
		if expected[name] {
			return nil, fmt.Errorf("duplicate manifest raw response %q", name)
		}
		expected[name] = true
	}

	archive, err := zip.OpenReader(options.Archive)
	if err != nil {
		return nil, fmt.Errorf("open retranslation raw-response archive: %w", err)
	}
	defer archive.Close()
	files := make(map[string]*zip.File, len(expected))
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			if file.Name != "raw-responses/" {
				return nil, fmt.Errorf("unexpected archive directory %q", file.Name)
			}
			continue
		}
		const prefix = "raw-responses/"
		if len(file.Name) <= len(prefix) || file.Name[:len(prefix)] != prefix {
			return nil, fmt.Errorf("unexpected archive file %q", file.Name)
		}
		name := file.Name[len(prefix):]
		if filepath.Base(name) != name || !expected[name] {
			return nil, fmt.Errorf("unexpected archive raw response %q", file.Name)
		}
		if files[name] != nil {
			return nil, fmt.Errorf("duplicate archive raw response %q", file.Name)
		}
		files[name] = file
	}
	if len(files) != len(expected) {
		return nil, fmt.Errorf("archive raw response count %d, want %d", len(files), len(expected))
	}
	for name := range expected {
		if files[name] == nil {
			return nil, fmt.Errorf("archive is missing raw response %q", name)
		}
	}

	staging, err := os.MkdirTemp(batchDir, ".import-staging-")
	if err != nil {
		return nil, fmt.Errorf("create retranslation import staging: %w", err)
	}
	defer os.RemoveAll(staging)
	stagedRaw := filepath.Join(staging, "raw-responses")
	if err := os.Mkdir(stagedRaw, 0755); err != nil {
		return nil, err
	}
	for _, unit := range manifest.Units {
		name := filepath.Base(filepath.FromSlash(unit.InputPath))
		in, err := files[name].Open()
		if err != nil {
			return nil, fmt.Errorf("open archived raw response %q: %w", name, err)
		}
		out, err := os.OpenFile(filepath.Join(stagedRaw, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			in.Close()
			return nil, fmt.Errorf("create imported raw response %q: %w", name, err)
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		readCloseErr := in.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("extract archived raw response %q: %w", name, copyErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close imported raw response %q: %w", name, closeErr)
		}
		if readCloseErr != nil {
			return nil, fmt.Errorf("close archived raw response %q: %w", name, readCloseErr)
		}
	}
	rawDir := filepath.Join(batchDir, "raw-responses")
	if err := os.Rename(stagedRaw, rawDir); err != nil {
		return nil, fmt.Errorf("commit retranslation raw responses: %w", err)
	}
	rawDirectory, err := repositoryRelativePath(root, rawDir)
	if err != nil {
		return nil, err
	}
	return &RetranslationImportResult{
		Locale: options.Locale, BatchID: options.BatchID, UnitCount: len(manifest.Units), RawDirectory: rawDirectory,
	}, nil
}
