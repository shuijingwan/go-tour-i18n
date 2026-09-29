package i18n

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const generationRecoveryMaxCandidateSize int64 = 64 << 20

type GenerationRecoveryIssue struct {
	Source string `json:"source"`
	Output string `json:"output,omitempty"`
	Reason string `json:"reason"`
}

type GenerationRecoveryResult struct {
	Locale                 string                    `json:"locale"`
	BatchID                string                    `json:"batch_id"`
	TaskKind               string                    `json:"task_kind"`
	Attempt                int                       `json:"attempt"`
	OutputDir              string                    `json:"output_dir"`
	Recovered              []string                  `json:"recovered"`
	Missing                []string                  `json:"missing"`
	Issues                 []GenerationRecoveryIssue `json:"issues,omitempty"`
	InputIdentitySHA256    string                    `json:"input_identity_sha256"`
	GenerationBundleSHA256 string                    `json:"generation_bundle_sha256"`
}

type generationRecoveryCandidate struct {
	source string
	data   []byte
}

// RecoverGenerationOutputs validates independently recoverable outputs from
// interrupted local staging/transport artifacts and writes only trusted bytes
// into a new non-formal staging directory. It never installs formal outputs.
func RecoverGenerationOutputs(root string, catalog *Catalog, generationBundleData []byte, sources []string, outputDir string) (*GenerationRecoveryResult, error) {
	if len(sources) == 0 {
		return nil, errors.New("generation recovery requires at least one source")
	}
	if outputDir == "" {
		return nil, errors.New("generation recovery requires an output directory")
	}
	manifest, _, err := currentGenerationBundle(root, catalog, generationBundleData)
	if err != nil {
		return nil, err
	}
	batchDir := filepath.Join(root, "data", "retranslation-runs", manifest.Locale, manifest.BatchID)
	if err := ensureNoGenerationImportPending(batchDir); err != nil {
		return nil, err
	}
	if manifest.TaskKind != "translation-unit-retry" {
		for _, name := range []string{"raw-responses", "candidates", "validation", "result.json"} {
			if _, err := os.Lstat(filepath.Join(batchDir, name)); err == nil {
				return nil, fmt.Errorf("formal generation batch state %q already exists; recover from the formal lifecycle instead of regenerating", name)
			} else if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}
	receiptPath := generationImportReceiptPath(batchDir, manifest)
	if _, err := os.Lstat(receiptPath); err == nil {
		return nil, fmt.Errorf("generation import provenance already exists: %s; inspect formal batch state before recovery", filepath.Base(receiptPath))
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	for _, expected := range manifest.ExpectedOutputs {
		if err := validateGenerationInstallPath(expected.InstallPath); err != nil {
			return nil, err
		}
		destination := filepath.Join(batchDir, filepath.FromSlash(expected.InstallPath))
		if _, err := os.Lstat(destination); err == nil {
			return nil, fmt.Errorf("formal generation output already exists: %s; recover from the formal lifecycle instead of regenerating", expected.InstallPath)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	expectedByName := make(map[string]GenerationBundleExpectedOutput, len(manifest.ExpectedOutputs))
	for _, expected := range manifest.ExpectedOutputs {
		name := filepath.Base(filepath.FromSlash(expected.BundlePath))
		if name == "." || name == string(filepath.Separator) || expectedByName[name].BundlePath != "" {
			return nil, errors.New("generation bundle has invalid recovery output names")
		}
		expectedByName[name] = expected
	}
	candidates := make(map[string][]generationRecoveryCandidate, len(expectedByName))
	var issues []GenerationRecoveryIssue
	for _, source := range sources {
		if source == "" {
			return nil, errors.New("generation recovery source must not be empty")
		}
		absolute, err := filepath.Abs(source)
		if err != nil {
			return nil, err
		}
		sourceIssues, err := collectGenerationRecoverySource(absolute, expectedByName, candidates)
		if err != nil {
			return nil, err
		}
		issues = append(issues, sourceIssues...)
	}
	glossary, err := LoadGlossary(root, manifest.Locale)
	if err != nil {
		return nil, err
	}
	recoveredBytes := make(map[string][]byte, len(expectedByName))
	recoveredSource := make(map[string]string, len(expectedByName))
	for _, expected := range manifest.ExpectedOutputs {
		name := filepath.Base(filepath.FromSlash(expected.BundlePath))
		for _, candidate := range candidates[name] {
			if err := validateGenerationOutputBytes(candidate.data); err != nil {
				issues = append(issues, GenerationRecoveryIssue{Source: candidate.source, Output: expected.BundlePath, Reason: err.Error()})
				continue
			}
			if err := validateTranslationUnitGenerationOutput(root, catalog, glossary, manifest.Locale, manifest.BatchID, expected.UnitID, candidate.data); err != nil {
				issues = append(issues, GenerationRecoveryIssue{Source: candidate.source, Output: expected.BundlePath, Reason: err.Error()})
				continue
			}
			if existing, ok := recoveredBytes[name]; ok {
				if !bytes.Equal(existing, candidate.data) {
					return nil, fmt.Errorf("conflicting valid recovery candidates for %s from %s and %s", expected.BundlePath, recoveredSource[name], candidate.source)
				}
				continue
			}
			recoveredBytes[name] = candidate.data
			recoveredSource[name] = candidate.source
		}
	}
	outputPath, err := writeRecoveredGenerationDirectory(outputDir, recoveredBytes)
	if err != nil {
		return nil, err
	}
	result := &GenerationRecoveryResult{
		Locale: manifest.Locale, BatchID: manifest.BatchID, TaskKind: manifest.TaskKind,
		Attempt: manifest.Attempt, OutputDir: outputPath, Issues: issues,
		InputIdentitySHA256: manifest.InputIdentitySHA256, GenerationBundleSHA256: sum(generationBundleData),
	}
	for _, expected := range manifest.ExpectedOutputs {
		name := filepath.Base(filepath.FromSlash(expected.BundlePath))
		if _, ok := recoveredBytes[name]; ok {
			result.Recovered = append(result.Recovered, expected.BundlePath)
		} else {
			result.Missing = append(result.Missing, expected.BundlePath)
		}
	}
	return result, nil
}

func collectGenerationRecoverySource(source string, expected map[string]GenerationBundleExpectedOutput, candidates map[string][]generationRecoveryCandidate) ([]GenerationRecoveryIssue, error) {
	info, err := os.Lstat(source)
	if err != nil {
		return nil, fmt.Errorf("inspect generation recovery source %s: %w", source, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("generation recovery source must not be a symlink: %s", source)
	}
	if info.IsDir() {
		return collectGenerationRecoveryDirectory(source, expected, candidates)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("generation recovery source must be a regular file or real directory: %s", source)
	}
	if _, ok := expected[filepath.Base(source)]; ok {
		data, err := readGenerationRecoveryFile(source)
		if err != nil {
			return []GenerationRecoveryIssue{{Source: source, Output: expected[filepath.Base(source)].BundlePath, Reason: err.Error()}}, nil
		}
		name := filepath.Base(source)
		candidates[name] = append(candidates[name], generationRecoveryCandidate{source: source, data: data})
		return nil, nil
	}
	return collectGenerationRecoveryArchive(source, expected, candidates)
}
func collectGenerationRecoveryDirectory(source string, expected map[string]GenerationBundleExpectedOutput, candidates map[string][]generationRecoveryCandidate) ([]GenerationRecoveryIssue, error) {
	entries, err := os.ReadDir(source)
	if err != nil {
		return nil, err
	}
	var issues []GenerationRecoveryIssue
	for _, entry := range entries {
		expectedOutput, ok := expected[entry.Name()]
		if !ok {
			continue
		}
		path := filepath.Join(source, entry.Name())
		data, err := readGenerationRecoveryFile(path)
		if err != nil {
			issues = append(issues, GenerationRecoveryIssue{Source: path, Output: expectedOutput.BundlePath, Reason: err.Error()})
			continue
		}
		candidates[entry.Name()] = append(candidates[entry.Name()], generationRecoveryCandidate{source: path, data: data})
	}
	return issues, nil
}

func readGenerationRecoveryFile(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("recovery candidate must be a regular non-symlink file")
	}
	if before.Size() > generationRecoveryMaxCandidateSize {
		return nil, fmt.Errorf("recovery candidate exceeds %d bytes", generationRecoveryMaxCandidateSize)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		file.Close()
		return nil, errors.New("recovery candidate changed while opening")
	}
	data, readErr := io.ReadAll(io.LimitReader(file, generationRecoveryMaxCandidateSize+1))
	after, statErr := file.Stat()
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if int64(len(data)) > generationRecoveryMaxCandidateSize {
		return nil, fmt.Errorf("recovery candidate exceeds %d bytes", generationRecoveryMaxCandidateSize)
	}
	if statErr != nil || closeErr != nil || !os.SameFile(opened, after) || after.Size() != int64(len(data)) {
		return nil, errors.New("recovery candidate changed while reading")
	}
	return data, nil
}

func collectGenerationRecoveryArchive(source string, expected map[string]GenerationBundleExpectedOutput, candidates map[string][]generationRecoveryCandidate) ([]GenerationRecoveryIssue, error) {
	before, err := os.Lstat(source)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("generation recovery archive must be a regular non-symlink file")
	}
	file, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, errors.New("generation recovery archive changed while opening")
	}
	prefix := make([]byte, 2)
	n, readErr := io.ReadFull(file, prefix)
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		return nil, readErr
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	var reader io.Reader = file
	var compressed *gzip.Reader
	if n == 2 && prefix[0] == 0x1f && prefix[1] == 0x8b {
		compressed, err = gzip.NewReader(file)
		if err != nil {
			return []GenerationRecoveryIssue{{Source: source, Reason: "open gzip recovery artifact: " + err.Error()}}, nil
		}
		defer compressed.Close()
		reader = compressed
	}
	archive := tar.NewReader(reader)
	var issues []GenerationRecoveryIssue
	archiveFailed := false
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			issues = append(issues, GenerationRecoveryIssue{Source: source, Reason: "read interrupted tar artifact: " + err.Error()})
			archiveFailed = true
			break
		}
		clean, ok := safeGenerationRecoveryArchivePath(header.Name)
		if !ok {
			issues = append(issues, GenerationRecoveryIssue{Source: source, Reason: fmt.Sprintf("unsafe archive member %q", header.Name)})
			continue
		}
		name := path.Base(clean)
		expectedOutput, wanted := expected[name]
		if !wanted {
			continue
		}
		memberSource := source + "!" + clean
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			issues = append(issues, GenerationRecoveryIssue{Source: memberSource, Output: expectedOutput.BundlePath, Reason: "expected archive member is not a regular file"})
			continue
		}
		if header.Size < 0 || header.Size > generationRecoveryMaxCandidateSize {
			issues = append(issues, GenerationRecoveryIssue{Source: memberSource, Output: expectedOutput.BundlePath, Reason: fmt.Sprintf("archive member exceeds %d bytes", generationRecoveryMaxCandidateSize)})
			continue
		}
		data, err := io.ReadAll(archive)
		if err != nil || int64(len(data)) != header.Size {
			if err == nil {
				err = io.ErrUnexpectedEOF
			}
			issues = append(issues, GenerationRecoveryIssue{Source: memberSource, Output: expectedOutput.BundlePath, Reason: "read complete archive member: " + err.Error()})
			archiveFailed = true
			break
		}
		candidates[name] = append(candidates[name], generationRecoveryCandidate{source: memberSource, data: data})
	}
	if compressed != nil && !archiveFailed {
		if _, err := io.Copy(io.Discard, compressed); err != nil {
			issues = append(issues, GenerationRecoveryIssue{Source: source, Reason: "gzip integrity check after recoverable tar members: " + err.Error()})
		}
	}
	after, statErr := file.Stat()
	if statErr != nil || !os.SameFile(opened, after) || after.Size() != opened.Size() {
		return nil, errors.New("generation recovery archive changed while reading")
	}
	return issues, nil
}
func safeGenerationRecoveryArchivePath(name string) (string, bool) {
	if name == "" || strings.Contains(name, "\\") || path.IsAbs(name) {
		return "", false
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

func writeRecoveredGenerationDirectory(outputDir string, outputs map[string][]byte) (string, error) {
	target, err := filepath.Abs(outputDir)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(target); err == nil {
		return "", fmt.Errorf("generation recovery output already exists: %s", target)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", err
	}
	parentInfo, err := os.Lstat(parent)
	if err != nil || !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("generation recovery output parent must be a real directory")
	}
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(target)+".recovery-*")
	if err != nil {
		return "", err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(staging)
		}
	}()
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if filepath.Base(name) != name || name == "." || name == string(filepath.Separator) {
			return "", fmt.Errorf("unsafe recovered generation filename %q", name)
		}
		if err := writeNewGenerationFile(filepath.Join(staging, name), outputs[name]); err != nil {
			return "", err
		}
	}
	if err := renameGenerationDirNoReplace(staging, target); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("generation recovery output already exists: %s", target)
		}
		return "", err
	}
	keep = true
	return target, nil
}
