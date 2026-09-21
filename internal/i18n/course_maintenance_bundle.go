package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const CourseMaintenanceGenerationBundleSchemaVersion = 1

type CourseMaintenanceGenerationBundleOptions struct {
	Locale      string
	TaskKind    string
	PageIDs     []string
	FindingPath string
}

type CourseMaintenanceGenerationContext struct {
	SchemaVersion   int      `json:"schema_version"`
	Kind            string   `json:"kind"`
	TaskKind        string   `json:"task_kind"`
	Locale          string   `json:"locale"`
	SelectedPageIDs []string `json:"selected_page_ids"`
	Instruction     string   `json:"instruction"`
	ExpectedOutput  string   `json:"expected_output"`
}

type CourseMaintenanceGenerationBundleManifest struct {
	SchemaVersion       int                   `json:"schema_version"`
	Kind                string                `json:"kind"`
	TaskKind            string                `json:"task_kind"`
	Locale              string                `json:"locale"`
	SelectedPageIDs     []string              `json:"selected_page_ids"`
	FindingPath         string                `json:"finding_path,omitempty"`
	InputIdentitySHA256 string                `json:"input_identity_sha256"`
	Files               []TransportBundleFile `json:"files"`
}

// ExportCourseMaintenanceGenerationBundle transports the complete schema-v2
// localization context plus the current base asset for refresh/revise. The
// existing refresh/revise commands remain the only formal mutation path.
func ExportCourseMaintenanceGenerationBundle(root string, catalog *Catalog, options CourseMaintenanceGenerationBundleOptions) ([]byte, CourseMaintenanceGenerationBundleManifest, error) {
	if options.TaskKind != "refresh" && options.TaskKind != "revise" {
		return nil, CourseMaintenanceGenerationBundleManifest{}, fmt.Errorf("course maintenance generation task must be refresh or revise")
	}
	localizationBundle, _, err := ExportCourseLocalizationGenerationBundle(root, options.Locale, catalog)
	if err != nil {
		return nil, CourseMaintenanceGenerationBundleManifest{}, err
	}
	localizationFiles, err := ReadTransportBundle(localizationBundle, 512, 256<<20)
	if err != nil {
		return nil, CourseMaintenanceGenerationBundleManifest{}, err
	}
	baseRepo := filepath.ToSlash(filepath.Join("locales", options.Locale, "course-metadata.json"))
	baseData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(baseRepo)))
	if err != nil {
		return nil, CourseMaintenanceGenerationBundleManifest{}, fmt.Errorf("read course metadata maintenance base: %w", err)
	}
	base, err := decodeCourseMetadata(baseData)
	if err != nil || base.SchemaVersion != CourseMetadataSchemaVersionV2 || base.Locale != options.Locale {
		return nil, CourseMaintenanceGenerationBundleManifest{}, fmt.Errorf("course maintenance generation requires a schema-v2 base for %s", options.Locale)
	}

	selected := []string{}
	if options.TaskKind == "refresh" {
		if len(options.PageIDs) != 0 || options.FindingPath != "" {
			return nil, CourseMaintenanceGenerationBundleManifest{}, fmt.Errorf("course refresh generation derives the exact stale subset; page IDs and finding are not accepted")
		}
		selected, err = courseMetadataStalePageIDs(root, catalog, options.Locale, base)
		if err != nil {
			return nil, CourseMaintenanceGenerationBundleManifest{}, err
		}
		if len(selected) == 0 {
			return nil, CourseMaintenanceGenerationBundleManifest{}, fmt.Errorf("course metadata has no stale Pages to refresh")
		}
	} else {
		if len(options.PageIDs) == 0 || options.FindingPath == "" {
			return nil, CourseMaintenanceGenerationBundleManifest{}, fmt.Errorf("course revise generation requires page IDs and a repository finding path")
		}
		if _, err := LoadCourseMetadata(root, options.Locale, catalog); err != nil {
			return nil, CourseMaintenanceGenerationBundleManifest{}, fmt.Errorf("course revise generation base is not current: %w", err)
		}
		known := make(map[string]bool, len(catalog.Pages))
		for _, page := range catalog.Pages {
			known[page.ID] = true
		}
		seen := map[string]bool{}
		for _, pageID := range options.PageIDs {
			if !known[pageID] || seen[pageID] {
				return nil, CourseMaintenanceGenerationBundleManifest{}, fmt.Errorf("invalid or duplicate course revise page_id %q", pageID)
			}
			seen[pageID] = true
			selected = append(selected, pageID)
		}
		sort.Slice(selected, func(i, j int) bool {
			return catalogPageIndex(catalog, selected[i]) < catalogPageIndex(catalog, selected[j])
		})
	}

	type entry struct {
		path, repositoryPath string
		data                 []byte
	}
	entries := []entry{{"formal/course-metadata.json", baseRepo, baseData}}
	for name, data := range localizationFiles {
		if name != "manifest.json" {
			entries = append(entries, entry{filepath.ToSlash(filepath.Join("localization", name)), "", data})
		}
	}
	findingRepo := ""
	if options.FindingPath != "" {
		findingFull, err := filepath.Abs(options.FindingPath)
		if err != nil {
			return nil, CourseMaintenanceGenerationBundleManifest{}, err
		}
		rootAbs, err := filepath.Abs(root)
		if err != nil || !pathWithinRoot(rootAbs, findingFull) {
			return nil, CourseMaintenanceGenerationBundleManifest{}, fmt.Errorf("course revise finding must be a file inside the repository")
		}
		info, err := os.Lstat(findingFull)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return nil, CourseMaintenanceGenerationBundleManifest{}, fmt.Errorf("course revise finding must be a regular non-symlink file")
		}
		findingData, err := os.ReadFile(findingFull)
		if err != nil {
			return nil, CourseMaintenanceGenerationBundleManifest{}, err
		}
		findingRepo, _ = filepath.Rel(rootAbs, findingFull)
		findingRepo = filepath.ToSlash(findingRepo)
		entries = append(entries, entry{"formal/reviewer-finding", findingRepo, findingData})
	}
	context := CourseMaintenanceGenerationContext{
		SchemaVersion:   CourseMaintenanceGenerationBundleSchemaVersion,
		Kind:            "go-tour-i18n/course-seo-maintenance-generation-context",
		TaskKind:        options.TaskKind,
		Locale:          options.Locale,
		SelectedPageIDs: selected,
		ExpectedOutput:  "localized-descriptions.json",
		Instruction:     "Generate only the selected page_id/description JSON subset. Canonical English description remains the exclusive semantic-scope authority; do not modify formal course metadata or claim review approval.",
	}
	contextData, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return nil, CourseMaintenanceGenerationBundleManifest{}, err
	}
	entries = append(entries, entry{"generation-context.json", "", append(contextData, '\n')})
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
	manifest := CourseMaintenanceGenerationBundleManifest{
		SchemaVersion:       CourseMaintenanceGenerationBundleSchemaVersion,
		Kind:                "go-tour-i18n/course-seo-maintenance-generation-bundle",
		TaskKind:            options.TaskKind,
		Locale:              options.Locale,
		SelectedPageIDs:     selected,
		FindingPath:         findingRepo,
		InputIdentitySHA256: sum([]byte(identity.String())),
		Files:               files,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, CourseMaintenanceGenerationBundleManifest{}, err
	}
	bundle, err := WriteDeterministicTransportBundle(append(manifestData, '\n'), zipEntries)
	if err != nil {
		return nil, CourseMaintenanceGenerationBundleManifest{}, err
	}
	return bundle, manifest, nil
}

func VerifyCurrentCourseMaintenanceGenerationBundle(root string, catalog *Catalog, bundle []byte) (CourseMaintenanceGenerationBundleManifest, error) {
	files, err := ReadTransportBundle(bundle, 1024, 512<<20)
	if err != nil {
		return CourseMaintenanceGenerationBundleManifest{}, err
	}
	var manifest CourseMaintenanceGenerationBundleManifest
	if err := decodeStrictBundleJSON(files["manifest.json"], &manifest); err != nil {
		return manifest, err
	}
	if manifest.SchemaVersion != CourseMaintenanceGenerationBundleSchemaVersion || manifest.Kind != "go-tour-i18n/course-seo-maintenance-generation-bundle" {
		return manifest, fmt.Errorf("course maintenance generation bundle has incompatible contract")
	}
	if err := ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		return manifest, err
	}
	finding := ""
	if manifest.FindingPath != "" {
		finding = filepath.Join(root, filepath.FromSlash(manifest.FindingPath))
	}
	pageIDs := manifest.SelectedPageIDs
	if manifest.TaskKind == "refresh" {
		pageIDs = nil
	}
	current, currentManifest, err := ExportCourseMaintenanceGenerationBundle(root, catalog, CourseMaintenanceGenerationBundleOptions{Locale: manifest.Locale, TaskKind: manifest.TaskKind, PageIDs: pageIDs, FindingPath: finding})
	if err != nil {
		return manifest, err
	}
	if !bytes.Equal(current, bundle) {
		return manifest, fmt.Errorf("course maintenance generation bundle is stale or non-canonical")
	}
	return currentManifest, nil
}

func courseMetadataStalePageIDs(root string, catalog *Catalog, locale string, base *CourseMetadata) ([]string, error) {
	baseByID, err := courseMetadataBaseIndex(base, catalog)
	if err != nil {
		return nil, err
	}
	targets, glossary, err := loadReadyCourseMetadataInputs(root, locale, catalog)
	if err != nil {
		return nil, err
	}
	sourceDescriptions, err := LoadCourseSourceDescriptions(root, catalog)
	if err != nil {
		return nil, err
	}
	if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err != nil {
		return nil, err
	}
	sourceByID := courseSourceDescriptionsByID(sourceDescriptions)
	stale := []string{}
	for _, page := range catalog.Pages {
		if courseMetadataEntryIsStale(base, baseByID[page.ID], page, targets[page.ID], sum(glossary), sourceByID) {
			stale = append(stale, page.ID)
		}
	}
	return stale, nil
}

func catalogPageIndex(catalog *Catalog, pageID string) int {
	for index, page := range catalog.Pages {
		if page.ID == pageID {
			return index
		}
	}
	return len(catalog.Pages)
}
