package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VerifyCurrentTranslationUnitGenerationBundle checks transport currentness
// without installing outputs or replacing the original invocation identity.
func VerifyCurrentTranslationUnitGenerationBundle(root string, catalog *Catalog, bundle []byte) (GenerationBundleManifest, error) {
	m, _, err := currentGenerationBundle(root, catalog, bundle)
	return m, err
}

// Rebuild the canonical transport using current immutable inputs and the
// original, authenticated batch summary. Only sibling machine state may differ.
func canonicalRetryBundleWithOriginalResult(root string, catalog *Catalog, original GenerationBundleManifest, oldFiles map[string][]byte, current []byte) ([]byte, error) {
	m, files, err := readGenerationBundle(current)
	if err != nil {
		return nil, err
	}
	batchDir := filepath.Join(root, "data/retranslation-runs", m.Locale, m.BatchID)
	if err := ensureNoGenerationImportPending(batchDir); err != nil {
		return nil, err
	}
	if len(original.ExpectedOutputs) != 1 {
		return nil, fmt.Errorf("retry requires exactly one target")
	}
	for _, path := range []string{filepath.Join(batchDir, filepath.FromSlash(m.ExpectedOutputs[0].InstallPath)), generationImportReceiptPath(batchDir, m)} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			return nil, fmt.Errorf("retry output or provenance already exists: %s", path)
		}
	}
	formal, err := readRetranslationProcessManifest(batchDir, m.Locale, m.BatchID)
	if err != nil {
		return nil, err
	}
	glossary, err := LoadGlossary(root, m.Locale)
	if err != nil {
		return nil, err
	}
	if _, err := preflightRetranslationProcess(batchDir, catalog, glossary, formal); err != nil {
		return nil, err
	}
	oldResult, err := verifyRetryBatchResult(batchDir, catalog, formal, oldFiles["formal/retry/result.json"], true)
	if err != nil {
		return nil, fmt.Errorf("original retry result: %w", err)
	}
	currentResult, err := verifyRetryBatchResult(batchDir, catalog, formal, files["formal/retry/result.json"], false)
	if err != nil {
		return nil, fmt.Errorf("current retry result: %w", err)
	}
	for i, item := range oldResult.Units {
		if item.UnitID == retryUnitID(original) && item != currentResult.Units[i] {
			return nil, fmt.Errorf("target retry result changed")
		}
	}
	files["formal/retry/result.json"] = oldFiles["formal/retry/result.json"]
	var identity strings.Builder
	entries := make([]TransportBundleEntry, 0, len(m.Files))
	for i, file := range m.Files {
		data := files[file.BundlePath]
		m.Files[i] = NewTransportBundleFile(file.BundlePath, file.RepositoryPath, data)
		identity.WriteString(file.BundlePath + "\x00" + m.Files[i].SHA256 + "\x00")
		entries = append(entries, TransportBundleEntry{Path: file.BundlePath, Data: data})
	}
	m.InputIdentitySHA256 = sum([]byte(identity.String()))
	manifest, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return writeTranslationUnitGenerationBundle(append(manifest, '\n'), entries)
}

// A historical summary must be a complete canonical result backed by actual
// current validation or immutable retry history, with exact membership/counts.
func verifyRetryBatchResult(batchDir string, catalog *Catalog, manifest *RetranslationBatchManifest, data []byte, historical bool) (*RetranslationProcessResult, error) {
	var result RetranslationProcessResult
	if err := decodeStrictBundleJSON(data, &result); err != nil {
		return nil, err
	}
	if result.SchemaVersion != retranslationProcessSchemaVersion || result.Locale != manifest.Locale || result.BatchID != manifest.BatchID || result.UnitCount != manifest.UnitCount || len(result.Units) != manifest.UnitCount || result.NoPendingBatches {
		return nil, fmt.Errorf("incompatible batch result")
	}
	for i, record := range manifest.Units {
		item := result.Units[i]
		unit, err := catalog.Unit(record.UnitID)
		if err != nil {
			return nil, err
		}
		flat := strings.TrimSuffix(filepath.Base(record.InputPath), filepath.Ext(record.InputPath))
		validationPath := "validation/" + flat + ".json"
		if item.UnitID != record.UnitID || item.UnitKind != record.UnitKind || item.ValidationPath != validationPath {
			return nil, fmt.Errorf("result membership/identity mismatch for %s", record.UnitID)
		}
		paths := []string{filepath.Join(batchDir, filepath.FromSlash(validationPath))}
		if historical {
			history, err := filepath.Glob(filepath.Join(batchDir, "retries", flat, "attempt-???-validation.json"))
			if err != nil {
				return nil, err
			}
			paths = append(paths, history...)
		}
		matched := false
		for _, path := range paths {
			b, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			v, err := decodeRetranslationValidation(b, unit)
			if err != nil {
				return nil, err
			}
			attempt, err := retryValidationAttemptForExtension(v.RawResponsePath, flat, filepath.Ext(record.InputPath))
			if err != nil || attempt != v.Attempt || v.SchemaVersion != result.SchemaVersion || v.Locale != result.Locale || v.BatchID != result.BatchID || v.UnitID != record.UnitID || v.UnitKind != record.UnitKind || v.SourceSHA256 != record.SourceSHA256 || v.InputPath != record.InputPath {
				return nil, fmt.Errorf("validation identity mismatch for %s", record.UnitID)
			}
			if path != paths[0] && filepath.Base(path) != fmt.Sprintf("attempt-%03d-validation.json", attempt) {
				return nil, fmt.Errorf("retry history attempt mismatch")
			}
			if v.Status == item.Status && v.CandidatePath == item.CandidatePath && v.Error == item.Error {
				matched = true
			}
		}
		if !matched || (item.Status != "passed" && item.Status != "restore_failed" && item.Status != "validation_failed") {
			return nil, fmt.Errorf("result has no matching validation evidence for %s", record.UnitID)
		}
	}
	recountRetranslationResult(&result)
	canonical, err := json.MarshalIndent(result, "", "  ")
	if err != nil || !bytes.Equal(append(canonical, '\n'), data) {
		return nil, fmt.Errorf("result counts or canonical bytes mismatch")
	}
	return &result, nil
}
