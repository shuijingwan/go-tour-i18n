package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const QualityCheckSnapshotSchemaVersion = 1

type QualityCheckSnapshotOptions struct {
	Locale     string
	SnapshotID string
}

type QualityCheckPageSectionIdentity struct {
	Article       string `json:"article"`
	SectionNumber int    `json:"section_number"`
	SourceTitle   string `json:"source_title"`
	Route         string `json:"route"`
}

type QualityCheckSnapshotUnit struct {
	Index            int                              `json:"index"`
	UnitID           string                           `json:"unit_id"`
	UnitKind         UnitKind                         `json:"unit_kind"`
	SelectedBatchID  string                           `json:"selected_batch_id"`
	SourcePath       string                           `json:"source_path"`
	SourceSHA256     string                           `json:"source_sha256"`
	PageSection      *QualityCheckPageSectionIdentity `json:"page_section,omitempty"`
	CandidatePath    string                           `json:"candidate_path"`
	CandidateSHA256  string                           `json:"candidate_sha256"`
	ValidationPath   string                           `json:"validation_path"`
	ValidationSHA256 string                           `json:"validation_sha256"`
	Attempt          int                              `json:"attempt"`
}

type QualityCheckSnapshotManifest struct {
	SchemaVersion  int                        `json:"schema_version"`
	SnapshotID     string                     `json:"snapshot_id"`
	Locale         string                     `json:"locale"`
	GlossaryPath   string                     `json:"glossary_path"`
	GlossarySHA256 string                     `json:"glossary_sha256"`
	UnitCount      int                        `json:"unit_count"`
	PageCount      int                        `json:"page_count"`
	ExampleCount   int                        `json:"example_count"`
	Units          []QualityCheckSnapshotUnit `json:"units"`
}

// CreateQualityCheckCandidateSnapshot freezes the complete current locale
// workflow candidate set after automatic validation and before Quality Check.
// It writes only a manifest whose paths refer to existing repository files.
func CreateQualityCheckCandidateSnapshot(root string, catalog *Catalog, options QualityCheckSnapshotOptions) (*QualityCheckSnapshotManifest, string, error) {
	if catalog == nil {
		return nil, "", errors.New("quality-check snapshot catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, "", err
	}
	if err := validateSnapshotID(options.SnapshotID); err != nil {
		return nil, "", err
	}
	glossaryPath := filepath.ToSlash(filepath.Join("locales", options.Locale, "glossary.yaml"))
	glossaryData, err := readSnapshotReferencedFile(root, glossaryPath)
	if err != nil {
		return nil, "", fmt.Errorf("read snapshot glossary: %w", err)
	}
	glossary, err := LoadGlossary(root, options.Locale)
	if err != nil {
		return nil, "", err
	}
	latest, err := selectLatestRetranslationUnits(root, catalog, options.Locale)
	if err != nil {
		return nil, "", err
	}
	manifest := &QualityCheckSnapshotManifest{
		SchemaVersion: QualityCheckSnapshotSchemaVersion,
		SnapshotID:    options.SnapshotID, Locale: options.Locale,
		GlossaryPath: glossaryPath, GlossarySHA256: sum(glossaryData),
		UnitCount: len(latest.ordered), PageCount: latest.pageCount, ExampleCount: latest.exampleCount,
		Units: make([]QualityCheckSnapshotUnit, 0, len(latest.ordered)),
	}
	for i, unit := range latest.ordered {
		choice, ok := latest.selectedByID[unit.ID]
		if !ok {
			return nil, "", fmt.Errorf("%s: no current-source retranslation candidate", unit.ID)
		}
		evidence, err := validateSelectedRetranslationCandidate(root, catalog, options.Locale, glossary, unit, choice)
		if err != nil {
			return nil, "", err
		}
		sourceData, err := readSnapshotReferencedFile(root, unit.SourcePath)
		if err != nil {
			return nil, "", fmt.Errorf("%s: source_path: %w", unit.ID, err)
		}
		if err := validateSnapshotSourceIdentity(catalog, unit, sourceData); err != nil {
			return nil, "", err
		}
		candidatePath := filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, choice.batchID, evidence.candidatePath))
		validationPath := filepath.ToSlash(filepath.Join("data", "retranslation-runs", options.Locale, choice.batchID, evidence.validationPath))
		record := QualityCheckSnapshotUnit{
			Index: i + 1, UnitID: unit.ID, UnitKind: unit.Kind, SelectedBatchID: choice.batchID,
			SourcePath: unit.SourcePath, SourceSHA256: unit.SourceSHA256,
			CandidatePath: candidatePath, CandidateSHA256: sum(evidence.candidate),
			ValidationPath: validationPath, ValidationSHA256: sum(evidence.validationData), Attempt: evidence.attempt,
		}
		if unit.Kind == UnitKindPage {
			page, err := catalog.Page(unit.ID)
			if err != nil {
				return nil, "", err
			}
			record.PageSection = &QualityCheckPageSectionIdentity{
				Article: page.Article, SectionNumber: page.SectionNumber,
				SourceTitle: page.SourceTitle, Route: page.Route,
			}
		}
		manifest.Units = append(manifest.Units, record)
	}

	base := filepath.Join(root, "data", "quality-check-snapshots", options.Locale)
	finalDir := filepath.Join(base, options.SnapshotID)
	if err := requireMissingSnapshotDirectory(finalDir); err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, "", fmt.Errorf("create quality-check snapshot locale directory: %w", err)
	}
	staging, err := os.MkdirTemp(base, "."+options.SnapshotID+".staging-")
	if err != nil {
		return nil, "", fmt.Errorf("create quality-check snapshot staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := writeTranslationJSON(filepath.Join(staging, "manifest.json"), manifest); err != nil {
		return nil, "", fmt.Errorf("write quality-check snapshot manifest: %w", err)
	}
	if err := requireMissingSnapshotDirectory(finalDir); err != nil {
		return nil, "", err
	}
	if err := os.Rename(staging, finalDir); err != nil {
		return nil, "", fmt.Errorf("commit quality-check snapshot: %w", err)
	}
	committed = true
	manifestPath, err := repositoryRelativePath(root, filepath.Join(finalDir, "manifest.json"))
	if err != nil {
		return nil, "", err
	}
	return manifest, manifestPath, nil
}

func validateSnapshotSourceIdentity(catalog *Catalog, unit *TranslationUnit, sourceData []byte) error {
	if unit.Kind == UnitKindExample {
		if !bytes.Equal(sourceData, unit.Source) || sum(sourceData) != unit.SourceSHA256 {
			return fmt.Errorf("%s: source_path bytes do not match current Catalog source identity", unit.ID)
		}
		return nil
	}
	page, err := catalog.Page(unit.ID)
	if err != nil {
		return err
	}
	data := normalizeLF(sourceData)
	standaloneData := data
	if page.Article != "welcome.article" {
		standaloneData = projectStandaloneConditionalContent(data, "appengine")
	}
	standalone, _, err := splitArticle(standaloneData, page.Article)
	if err != nil {
		return fmt.Errorf("%s: split source article: %w", unit.ID, err)
	}
	var conditional [][]byte
	if page.Article == "welcome.article" {
		conditional, err = splitConditional(data)
		if err != nil {
			return fmt.Errorf("%s: split conditional source article: %w", unit.ID, err)
		}
	}
	published, err := projectPublishedSections(page.Article, data, standalone, conditional)
	if err != nil {
		return fmt.Errorf("%s: project source article: %w", unit.ID, err)
	}
	if page.SectionNumber < 1 || page.SectionNumber > len(published) ||
		!bytes.Equal(published[page.SectionNumber-1].Source, unit.Source) || sum(unit.Source) != unit.SourceSHA256 {
		return fmt.Errorf("%s: source_path section does not match current Catalog source identity", unit.ID)
	}
	return nil
}

func validateSnapshotID(snapshotID string) error {
	if snapshotID == "" || strings.HasPrefix(snapshotID, ".") || filepath.Base(snapshotID) != snapshotID || snapshotID == "." || strings.ContainsAny(snapshotID, `/\\`) {
		return fmt.Errorf("invalid quality-check snapshot_id %q", snapshotID)
	}
	return nil
}

func readSnapshotReferencedFile(root, repositoryPath string) ([]byte, error) {
	if repositoryPath == "" || filepath.IsAbs(repositoryPath) || filepath.Clean(repositoryPath) == "." || strings.HasPrefix(filepath.ToSlash(filepath.Clean(repositoryPath)), "../") {
		return nil, fmt.Errorf("unsafe repository path %q", repositoryPath)
	}
	path := filepath.Join(root, filepath.FromSlash(repositoryPath))
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("repository path is not a regular file: %s", repositoryPath)
	}
	return os.ReadFile(path)
}

func requireMissingSnapshotDirectory(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("quality-check snapshot directory already exists: %s", filepath.ToSlash(path))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect quality-check snapshot directory: %w", err)
	}
	return nil
}
