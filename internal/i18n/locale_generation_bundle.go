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

const LocaleGenerationBundleSchemaVersion = 1

const (
	LocaleGenerationTaskGlossary           = "glossary"
	LocaleGenerationTaskLocaleAssets       = "locale-assets"
	LocaleGenerationTaskSurfaceReplacement = "surface-replacement"
)

var localeGenerationAuthorityPaths = []string{
	"AGENTS.md",
	"docs/CHATGPT_LANGUAGE_GENERATION.md",
	"docs/CODEX_TRANSLATION.md",
	"docs/GLOSSARY_REVIEW.md",
	"docs/LOCALE_SURFACE_REVIEW.md",
	"docs/NEW_LOCALE_RUNBOOK.md",
	"docs/TERMINOLOGY_GUIDE.md",
	"docs/TRANSLATION_TERMINOLOGY.md",
	"docs/TRANSLATION_WORKFLOW.md",
}

type LocaleGenerationBundleOptions struct {
	Locale   string
	TaskKind string
	ReviewID string
}

type LocaleGenerationContext struct {
	SchemaVersion   int                          `json:"schema_version"`
	Kind            string                       `json:"kind"`
	TaskKind        string                       `json:"task_kind"`
	Locale          string                       `json:"locale"`
	ReviewID        string                       `json:"review_id,omitempty"`
	Instruction     string                       `json:"instruction"`
	ExpectedOutputs []string                     `json:"expected_outputs"`
	SourceUnits     []GlossaryReviewerSourceUnit `json:"source_units,omitempty"`
}

type LocaleGenerationBundleManifest struct {
	SchemaVersion       int                   `json:"schema_version"`
	Kind                string                `json:"kind"`
	TaskKind            string                `json:"task_kind"`
	Locale              string                `json:"locale"`
	ReviewID            string                `json:"review_id,omitempty"`
	InputIdentitySHA256 string                `json:"input_identity_sha256"`
	Files               []TransportBundleFile `json:"files"`
}

// ExportLocaleGenerationBundle creates provider-neutral, read-only transport
// for locale-level language generation. Formal mutation remains with the
// existing task-specific local workflow; this bundle is not a receipt or gate.
func ExportLocaleGenerationBundle(root string, catalog *Catalog, options LocaleGenerationBundleOptions) ([]byte, LocaleGenerationBundleManifest, error) {
	if catalog == nil {
		return nil, LocaleGenerationBundleManifest{}, fmt.Errorf("locale generation bundle catalog is required")
	}
	if err := ValidateLocaleName(options.Locale); err != nil {
		return nil, LocaleGenerationBundleManifest{}, err
	}
	if options.TaskKind != LocaleGenerationTaskGlossary && options.TaskKind != LocaleGenerationTaskLocaleAssets && options.TaskKind != LocaleGenerationTaskSurfaceReplacement {
		return nil, LocaleGenerationBundleManifest{}, fmt.Errorf("unsupported locale generation task %q", options.TaskKind)
	}
	if options.TaskKind == LocaleGenerationTaskSurfaceReplacement && options.ReviewID == "" {
		return nil, LocaleGenerationBundleManifest{}, fmt.Errorf("surface-replacement requires review_id")
	}
	if options.ReviewID != "" && !reviewIDPattern.MatchString(options.ReviewID) {
		return nil, LocaleGenerationBundleManifest{}, fmt.Errorf("invalid review_id %q", options.ReviewID)
	}
	if options.TaskKind != LocaleGenerationTaskSurfaceReplacement && options.ReviewID != "" {
		return nil, LocaleGenerationBundleManifest{}, fmt.Errorf("review_id is only valid for surface-replacement")
	}
	if options.TaskKind != LocaleGenerationTaskGlossary {
		if err := RequireCurrentGlossaryReview(root, options.Locale); err != nil {
			return nil, LocaleGenerationBundleManifest{}, fmt.Errorf("%s generation requires current Glossary Review coverage: %w", options.TaskKind, err)
		}
	}

	type entry struct {
		path, repositoryPath string
		data                 []byte
	}
	entries := []entry{}
	addRepositoryFile := func(bundlePath, repositoryPath string, required bool) error {
		full := filepath.Join(root, filepath.FromSlash(repositoryPath))
		info, err := os.Lstat(full)
		if os.IsNotExist(err) && !required {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read locale generation input %s: %w", repositoryPath, err)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("locale generation input %s must be a regular non-symlink file", repositoryPath)
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		entries = append(entries, entry{bundlePath, repositoryPath, data})
		return nil
	}

	localeRepo := filepath.ToSlash(filepath.Join("locales", options.Locale, "locale.json"))
	glossaryRepo := filepath.ToSlash(filepath.Join("locales", options.Locale, "glossary.yaml"))
	uiTargetRepo := filepath.ToSlash(filepath.Join("internal", "tour", "ui", options.Locale+".json"))
	articleRepo := filepath.ToSlash(filepath.Join("locales", options.Locale, "article-metadata.json"))
	for _, item := range []struct {
		bundle, repository string
		required           bool
	}{
		{"formal/locale.json", localeRepo, true},
		{"formal/glossary.yaml", glossaryRepo, true},
		{"formal/ui-en.json", "internal/tour/ui/en.json", true},
		{"formal/ui-target.json", uiTargetRepo, false},
		{"formal/article-metadata.json", articleRepo, false},
		{"formal/tour-pages.tsv", "data/tour-pages.tsv", true},
		{"formal/tour-examples.tsv", "data/tour-examples.tsv", true},
	} {
		if err := addRepositoryFile(item.bundle, item.repository, item.required); err != nil {
			return nil, LocaleGenerationBundleManifest{}, err
		}
	}

	context := LocaleGenerationContext{
		SchemaVersion: LocaleGenerationBundleSchemaVersion,
		Kind:          "go-tour-i18n/locale-generation-context",
		TaskKind:      options.TaskKind,
		Locale:        options.Locale,
		ReviewID:      options.ReviewID,
	}
	switch options.TaskKind {
	case LocaleGenerationTaskGlossary:
		context.Instruction = "Generate or revise only the complete glossary.yaml for this locale. Preserve mandatory/preferred/forbidden/keep semantics; do not claim review approval."
		context.ExpectedOutputs = []string{"glossary.yaml"}
		units, _, _, err := localeWorkflowUnitList(catalog)
		if err != nil {
			return nil, LocaleGenerationBundleManifest{}, err
		}
		context.SourceUnits = make([]GlossaryReviewerSourceUnit, 0, len(units))
		for i, unit := range units {
			context.SourceUnits = append(context.SourceUnits, GlossaryReviewerSourceUnit{Index: i + 1, UnitID: unit.ID, UnitKind: unit.Kind, SourcePath: unit.SourcePath, SourceSHA256: unit.SourceSHA256, Source: string(unit.Source)})
		}
	case LocaleGenerationTaskLocaleAssets:
		context.Instruction = "Generate complete target UI and article metadata files from their formal English source/context and the complete reviewed glossary. Do not modify TranslationUnits or Course SEO metadata."
		context.ExpectedOutputs = []string{"ui.json", "article-metadata.json"}
		units, _, _, err := localeWorkflowUnitList(catalog)
		if err != nil {
			return nil, LocaleGenerationBundleManifest{}, err
		}
		for i, unit := range units {
			if unit.Kind == UnitKindPage {
				context.SourceUnits = append(context.SourceUnits, GlossaryReviewerSourceUnit{Index: i + 1, UnitID: unit.ID, UnitKind: unit.Kind, SourcePath: unit.SourcePath, SourceSHA256: unit.SourceSHA256, Source: string(unit.Source)})
			}
		}
	case LocaleGenerationTaskSurfaceReplacement:
		context.Instruction = "Generate only replacements requested by the attached independent Surface Review findings. Do not approve the findings or mutate formal repository state."
		context.ExpectedOutputs = []string{"replacements.json"}
		evidenceRepo := filepath.ToSlash(filepath.Join("data", "locale-surface-reviews", options.Locale, options.ReviewID+".md"))
		if err := addRepositoryFile("formal/review-evidence.md", evidenceRepo, true); err != nil {
			return nil, LocaleGenerationBundleManifest{}, err
		}
		packageData, _, err := ExportLocaleSurfaceReviewPackage(root, options.Locale, catalog)
		if err != nil {
			return nil, LocaleGenerationBundleManifest{}, err
		}
		entries = append(entries, entry{"formal/surface-review.json", "", packageData})
	}
	contextData, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return nil, LocaleGenerationBundleManifest{}, err
	}
	entries = append(entries, entry{"generation-context.json", "", append(contextData, '\n')})
	for _, repoPath := range localeGenerationAuthorityPaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(repoPath)))
		if err != nil {
			return nil, LocaleGenerationBundleManifest{}, fmt.Errorf("read locale generation authority %s: %w", repoPath, err)
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
	manifest := LocaleGenerationBundleManifest{
		SchemaVersion:       LocaleGenerationBundleSchemaVersion,
		Kind:                "go-tour-i18n/locale-generation-bundle",
		TaskKind:            options.TaskKind,
		Locale:              options.Locale,
		ReviewID:            options.ReviewID,
		InputIdentitySHA256: sum([]byte(identity.String())),
		Files:               files,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, LocaleGenerationBundleManifest{}, err
	}
	bundle, err := WriteDeterministicTransportBundle(append(manifestData, '\n'), zipEntries)
	if err != nil {
		return nil, LocaleGenerationBundleManifest{}, err
	}
	return bundle, manifest, nil
}

func VerifyCurrentLocaleGenerationBundle(root string, catalog *Catalog, bundle []byte) (LocaleGenerationBundleManifest, error) {
	files, err := ReadTransportBundle(bundle, 512, 256<<20)
	if err != nil {
		return LocaleGenerationBundleManifest{}, err
	}
	var manifest LocaleGenerationBundleManifest
	if err := decodeStrictBundleJSON(files["manifest.json"], &manifest); err != nil {
		return manifest, fmt.Errorf("parse locale generation bundle manifest: %w", err)
	}
	if manifest.SchemaVersion != LocaleGenerationBundleSchemaVersion || manifest.Kind != "go-tour-i18n/locale-generation-bundle" {
		return manifest, fmt.Errorf("locale generation bundle has incompatible contract")
	}
	if err := ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		return manifest, err
	}
	current, currentManifest, err := ExportLocaleGenerationBundle(root, catalog, LocaleGenerationBundleOptions{Locale: manifest.Locale, TaskKind: manifest.TaskKind, ReviewID: manifest.ReviewID})
	if err != nil {
		return manifest, err
	}
	if !bytes.Equal(current, bundle) {
		return manifest, fmt.Errorf("locale generation bundle is stale or non-canonical")
	}
	return currentManifest, nil
}
