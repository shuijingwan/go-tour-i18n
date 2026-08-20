package i18n

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const defaultRetranslationExportLimit = 10

type RetranslationExportOptions struct {
	Locale        string
	BatchID       string
	UnitIDs       []string
	UnitKind      UnitKind
	Limit         int
	AllowReexport bool
}

type RetranslationBatchUnit struct {
	UnitID              string   `json:"unit_id"`
	UnitKind            UnitKind `json:"unit_kind"`
	SourcePath          string   `json:"source_path"`
	SourceSHA256        string   `json:"source_sha256"`
	InputPath           string   `json:"input_path"`
	InputSHA256         string   `json:"input_sha256"`
	ProtectedTokenCount int      `json:"protected_token_count"`
}

type RetranslationBatchManifest struct {
	SchemaVersion  int                      `json:"schema_version"`
	BatchID        string                   `json:"batch_id"`
	Locale         string                   `json:"locale"`
	ProtectionMode string                   `json:"protection_mode"`
	UnitKind       UnitKind                 `json:"unit_kind"`
	UnitCount      int                      `json:"unit_count"`
	Units          []RetranslationBatchUnit `json:"units"`
}

type RetranslationExportResult struct {
	Locale      string   `json:"locale"`
	BatchID     string   `json:"batch_id,omitempty"`
	BatchPath   string   `json:"batch_path,omitempty"`
	UnitCount   int      `json:"unit_count"`
	UnitIDs     []string `json:"unit_ids,omitempty"`
	AllExported bool     `json:"all_exported"`
}

type preparedRetranslationInput struct {
	unit   *TranslationUnit
	text   string
	path   string
	hash   string
	tokens int
}

type exportedRetranslationUnit struct {
	BatchID      string
	SourceSHA256 string
}

type retranslationStatus struct {
	StaleSource bool
	ReadySource bool
}

// ExportRetranslationBatch writes one isolated batch of Default protected
// inputs without invoking a model or changing formal translation state.
func ExportRetranslationBatch(root string, catalog *Catalog, options RetranslationExportOptions) (*RetranslationExportResult, error) {
	if catalog == nil {
		return nil, errors.New("retranslation catalog is required")
	}
	if options.Locale == "" {
		return nil, errors.New("retranslation locale is required")
	}
	if options.Locale != "zh-CN" {
		return nil, fmt.Errorf("unsupported locale %q", options.Locale)
	}
	if options.UnitKind != "" && options.UnitKind != UnitKindPage && options.UnitKind != UnitKindExample {
		return nil, fmt.Errorf("不支持的翻译单元类型 %q；只支持 page 或 example", options.UnitKind)
	}
	if options.AllowReexport && len(options.UnitIDs) == 0 {
		return nil, errors.New("--allow-reexport requires at least one --id")
	}
	limit := options.Limit
	if limit == 0 {
		limit = defaultRetranslationExportLimit
	}
	if limit < 1 {
		return nil, errors.New("retranslation export limit must be greater than zero")
	}

	base := filepath.Join(root, "data", "retranslation-runs", options.Locale)
	exported, nextNumber, err := scanRetranslationBatches(base, options.Locale, catalog)
	if err != nil {
		return nil, err
	}
	statuses, err := retranslationStatuses(root, options.Locale, catalog)
	if err != nil {
		return nil, err
	}
	units, err := selectRetranslationUnits(catalog, options.UnitIDs, options.UnitKind, exported, statuses, limit, options.AllowReexport)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return &RetranslationExportResult{Locale: options.Locale, AllExported: true}, nil
	}

	batchID := options.BatchID
	if batchID == "" {
		batchID = fmt.Sprintf("chatgpt-%s-%03d", options.Locale, nextNumber)
	}
	if err := validateBatchID(batchID); err != nil {
		return nil, err
	}
	finalDir := filepath.Join(base, batchID)
	if err := requireMissingBatchDirectory(finalDir); err != nil {
		return nil, err
	}

	glossary, err := LoadGlossary(root, options.Locale)
	if err != nil {
		return nil, err
	}
	prepared := make([]preparedRetranslationInput, 0, len(units))
	for _, unit := range units {
		if sum(unit.Source) != unit.SourceSHA256 {
			return nil, fmt.Errorf("%s: hydrated source hash mismatch", unit.ID)
		}
		if unit.Kind == UnitKindExample {
			hasContent, err := hasTranslatableGoExampleComment(unit.Source)
			if err != nil {
				return nil, fmt.Errorf("%s: 检查可翻译自然语言注释: %w", unit.ID, err)
			}
			if !hasContent {
				return nil, fmt.Errorf("示例翻译单元 %s 没有需要翻译的普通自然语言注释", unit.ID)
			}
		}
		protected, err := prepareTranslationUnitInput(unit, glossary)
		if err != nil {
			return nil, fmt.Errorf("%s: 准备受保护输入: %w", unit.ID, err)
		}
		inputPath := filepath.ToSlash(filepath.Join("inputs", retranslationUnitInputName(unit)))
		prepared = append(prepared, preparedRetranslationInput{
			unit: unit, text: protected.Text, path: inputPath,
			hash: sum([]byte(protected.Text)), tokens: len(protected.Tokens),
		})
	}

	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, fmt.Errorf("create retranslation locale directory: %w", err)
	}
	staging, err := os.MkdirTemp(base, "."+batchID+".staging-")
	if err != nil {
		return nil, fmt.Errorf("create retranslation staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := os.Mkdir(filepath.Join(staging, "inputs"), 0755); err != nil {
		return nil, fmt.Errorf("create retranslation inputs directory: %w", err)
	}
	manifest := RetranslationBatchManifest{
		SchemaVersion: 2, BatchID: batchID, Locale: options.Locale,
		ProtectionMode: "default", UnitKind: prepared[0].unit.Kind,
		UnitCount: len(prepared), Units: make([]RetranslationBatchUnit, 0, len(prepared)),
	}
	unitIDs := make([]string, 0, len(prepared))
	for _, input := range prepared {
		if err := os.WriteFile(filepath.Join(staging, filepath.FromSlash(input.path)), []byte(input.text), 0644); err != nil {
			return nil, fmt.Errorf("write retranslation input for %s: %w", input.unit.ID, err)
		}
		record := RetranslationBatchUnit{
			UnitID: input.unit.ID, UnitKind: input.unit.Kind, SourcePath: input.unit.SourcePath,
			SourceSHA256: input.unit.SourceSHA256, InputPath: input.path,
			InputSHA256: input.hash, ProtectedTokenCount: input.tokens,
		}
		manifest.Units = append(manifest.Units, record)
		unitIDs = append(unitIDs, input.unit.ID)
	}
	if err := writeTranslationJSON(filepath.Join(staging, "manifest.json"), manifest); err != nil {
		return nil, fmt.Errorf("write retranslation manifest: %w", err)
	}
	if err := requireMissingBatchDirectory(finalDir); err != nil {
		return nil, err
	}
	if err := os.Rename(staging, finalDir); err != nil {
		return nil, fmt.Errorf("commit retranslation batch: %w", err)
	}
	committed = true
	batchPath, err := repositoryRelativePath(root, finalDir)
	if err != nil {
		return nil, err
	}
	return &RetranslationExportResult{
		Locale: options.Locale, BatchID: batchID, BatchPath: batchPath,
		UnitCount: len(unitIDs), UnitIDs: unitIDs,
	}, nil
}

func selectRetranslationUnits(catalog *Catalog, requested []string, requestedKind UnitKind, exported map[string]exportedRetranslationUnit, statuses map[string]retranslationStatus, limit int, allowReexport bool) ([]*TranslationUnit, error) {
	if len(requested) != 0 {
		seen := map[string]bool{}
		units := make([]*TranslationUnit, 0, len(requested))
		var kind UnitKind
		for _, unitID := range requested {
			if seen[unitID] {
				return nil, fmt.Errorf("duplicate requested translation unit %q", unitID)
			}
			seen[unitID] = true
			unit, err := catalog.Unit(unitID)
			if err != nil {
				return nil, err
			}
			if kind != "" && unit.Kind != kind {
				return nil, errors.New("一个重译批次不能混合课程页面单元和示例单元")
			}
			if requestedKind != "" && unit.Kind != requestedKind {
				return nil, fmt.Errorf("翻译单元 %s 的类型为 %s，与 --unit-kind %s 不一致", unit.ID, unit.Kind, requestedKind)
			}
			kind = unit.Kind
			if history := exported[unitID]; history.BatchID != "" && !allowReexport {
				return nil, fmt.Errorf("translation unit %q was already exported in batch %q", unitID, history.BatchID)
			}
			units = append(units, unit)
		}
		return units, nil
	}
	if requestedKind == UnitKindExample {
		units := make([]*TranslationUnit, 0, limit)
		for i := range catalog.Examples {
			example := &catalog.Examples[i]
			hasContent, err := hasTranslatableGoExampleComment(example.Source)
			if err != nil {
				return nil, fmt.Errorf("%s: 检查可翻译自然语言注释: %w", example.ID, err)
			}
			if !hasContent {
				continue
			}
			unit, err := catalog.Unit(example.ID)
			if err != nil {
				return nil, err
			}
			if alreadyExportedForCurrentSource(exported[example.ID], unit, statuses[example.ID]) {
				continue
			}
			units = append(units, unit)
			if len(units) == limit {
				break
			}
		}
		return units, nil
	}
	units := make([]*TranslationUnit, 0, limit)
	for _, page := range catalog.Pages {
		unit, err := catalog.Unit(page.ID)
		if err != nil {
			return nil, err
		}
		if alreadyExportedForCurrentSource(exported[page.ID], unit, statuses[page.ID]) {
			continue
		}
		units = append(units, unit)
		if len(units) == limit {
			break
		}
	}
	return units, nil
}

// alreadyExportedForCurrentSource preserves the normal exported-unit guard,
// but a formal status tied to an older source version needs one fresh batch.
func alreadyExportedForCurrentSource(history exportedRetranslationUnit, unit *TranslationUnit, status retranslationStatus) bool {
	if status.ReadySource {
		return true
	}
	if history.BatchID == "" {
		return false
	}
	return !status.StaleSource || history.SourceSHA256 == unit.SourceSHA256
}

func retranslationStatuses(root, locale string, catalog *Catalog) (map[string]retranslationStatus, error) {
	result := map[string]retranslationStatus{}
	statuses, err := ReadStatuses(filepath.Join(root, "locales", locale, "status.tsv"))
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read formal status for retranslation export: %w", err)
	}
	for _, status := range statuses {
		unit, err := catalog.Unit(status.UnitID)
		if err != nil {
			return nil, fmt.Errorf("formal status references unknown translation unit %q", status.UnitID)
		}
		if status.SourceSHA256 != unit.SourceSHA256 {
			result[unit.ID] = retranslationStatus{StaleSource: true}
			continue
		}
		if status.State == "ready" {
			result[unit.ID] = retranslationStatus{ReadySource: true}
		}
	}
	return result, nil
}

func retranslationUnitInputName(unit *TranslationUnit) string {
	if unit.Kind == UnitKindExample {
		name := strings.ReplaceAll(strings.TrimPrefix(unit.ID, "example:"), "/", "-")
		return strings.TrimSuffix(name, filepath.Ext(name)) + ".txt"
	}
	return flattenedPageArticleName(unit.ID)
}

func retranslationUnitCandidateName(unit *TranslationUnit) string {
	if unit.Kind == UnitKindExample {
		return strings.ReplaceAll(strings.TrimPrefix(unit.ID, "example:"), "/", "-")
	}
	return flattenedPageArticleName(unit.ID)
}

func scanRetranslationBatches(base, locale string, catalog *Catalog) (map[string]exportedRetranslationUnit, int, error) {
	exported := map[string]exportedRetranslationUnit{}
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return exported, 1, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("scan retranslation batches: %w", err)
	}
	known := make(map[string]UnitKind, len(catalog.Pages)+len(catalog.Examples))
	for _, page := range catalog.Pages {
		known[page.ID] = UnitKindPage
	}
	for _, example := range catalog.Examples {
		known[example.ID] = UnitKindExample
	}
	prefix := "chatgpt-" + locale + "-"
	batchPattern := regexp.MustCompile("^" + regexp.QuoteMeta(prefix) + `([0-9]+)$`)
	nextNumber := 1
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if match := batchPattern.FindStringSubmatch(entry.Name()); match != nil {
			var number int
			if _, err := fmt.Sscanf(match[1], "%d", &number); err == nil && number >= nextNumber {
				nextNumber = number + 1
			}
		}
		manifestPath := filepath.Join(base, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, 0, fmt.Errorf("read retranslation manifest %s: %w", filepath.ToSlash(manifestPath), err)
		}
		manifest, err := decodeRetranslationManifest(data)
		if err != nil {
			return nil, 0, fmt.Errorf("parse retranslation manifest %s: %w", filepath.ToSlash(manifestPath), err)
		}
		if manifest.BatchID != entry.Name() {
			return nil, 0, fmt.Errorf("retranslation manifest batch_id %q does not match directory %q", manifest.BatchID, entry.Name())
		}
		if manifest.Locale != locale {
			return nil, 0, fmt.Errorf("retranslation batch %q locale %q does not match %q", entry.Name(), manifest.Locale, locale)
		}
		if manifest.SchemaVersion != 2 || manifest.ProtectionMode != "default" || (manifest.UnitKind != UnitKindPage && manifest.UnitKind != UnitKindExample) {
			return nil, 0, fmt.Errorf("retranslation batch %q has incompatible manifest metadata", entry.Name())
		}
		if manifest.UnitCount == 0 {
			return nil, 0, fmt.Errorf("retranslation batch %q has no translation units", entry.Name())
		}
		if manifest.UnitCount != len(manifest.Units) {
			return nil, 0, fmt.Errorf("retranslation batch %q unit_count %d does not match units %d", entry.Name(), manifest.UnitCount, len(manifest.Units))
		}
		for _, record := range manifest.Units {
			unitID, unitKind := record.UnitID, record.UnitKind
			wantKind, ok := known[unitID]
			if !ok {
				return nil, 0, fmt.Errorf("retranslation batch %q has unknown translation unit %q", entry.Name(), unitID)
			}
			if unitKind != wantKind || manifest.UnitKind != unitKind {
				return nil, 0, fmt.Errorf("retranslation batch %q translation unit metadata mismatch for %q", entry.Name(), unitID)
			}
			// Entries are sorted by batch ID, so retain the latest export for a
			// unit. Its source hash decides whether a stale formal status still
			// needs a fresh batch.
			exported[unitID] = exportedRetranslationUnit{BatchID: entry.Name(), SourceSHA256: record.SourceSHA256}
		}
	}
	return exported, nextNumber, nil
}

func validateBatchID(batchID string) error {
	if batchID == "" || strings.HasPrefix(batchID, ".") || filepath.Base(batchID) != batchID || batchID == "." || strings.ContainsAny(batchID, `/\\`) {
		return fmt.Errorf("invalid retranslation batch_id %q", batchID)
	}
	return nil
}

func requireMissingBatchDirectory(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("retranslation batch directory already exists: %s", filepath.ToSlash(path))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect retranslation batch directory: %w", err)
	}
	return nil
}
