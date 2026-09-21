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

const GlossaryReviewerBundleSchemaVersion = 1

var glossaryReviewerAuthorityPaths = []string{
	"AGENTS.md",
	"docs/GLOSSARY_REVIEW.md",
	"docs/NEW_LOCALE_RUNBOOK.md",
	"docs/TERMINOLOGY_GUIDE.md",
	"docs/TRANSLATION_TASK_SPEC.md",
	"docs/TRANSLATION_TERMINOLOGY.md",
}

func VerifyCurrentGlossaryReviewerBundle(root, locale string, catalog *Catalog, bundle []byte) (GlossaryReviewerBundleManifest, error) {
	files, err := ReadTransportBundle(bundle, 512, 128<<20)
	if err != nil {
		return GlossaryReviewerBundleManifest{}, err
	}
	var manifest GlossaryReviewerBundleManifest
	if err := decodeStrictBundleJSON(files["manifest.json"], &manifest); err != nil {
		return manifest, fmt.Errorf("parse glossary reviewer bundle manifest: %w", err)
	}
	if manifest.SchemaVersion != GlossaryReviewerBundleSchemaVersion || manifest.Kind != "go-tour-i18n/glossary-reviewer-bundle" || manifest.Locale != locale {
		return manifest, fmt.Errorf("glossary reviewer bundle identity mismatch")
	}
	if err := ValidateTransportBundleInventory(files, manifest.Files, true); err != nil {
		return manifest, err
	}
	current, currentManifest, err := ExportGlossaryReviewerBundle(root, locale, catalog)
	if err != nil {
		return manifest, err
	}
	if !bytes.Equal(current, bundle) {
		return manifest, fmt.Errorf("glossary reviewer bundle is stale or non-canonical")
	}
	return currentManifest, nil
}

type GlossaryReviewerSourceUnit struct {
	Index        int      `json:"index"`
	UnitID       string   `json:"unit_id"`
	UnitKind     UnitKind `json:"unit_kind"`
	SourcePath   string   `json:"source_path"`
	SourceSHA256 string   `json:"source_sha256"`
	Source       string   `json:"source"`
}

type GlossaryReviewerContext struct {
	SchemaVersion int                          `json:"schema_version"`
	Kind          string                       `json:"kind"`
	Locale        string                       `json:"locale"`
	Instruction   string                       `json:"instruction"`
	UnitCount     int                          `json:"unit_count"`
	PageCount     int                          `json:"page_count"`
	ExampleCount  int                          `json:"example_count"`
	SourceUnits   []GlossaryReviewerSourceUnit `json:"source_units"`
}

type GlossaryReviewerBundleManifest struct {
	SchemaVersion       int                   `json:"schema_version"`
	Kind                string                `json:"kind"`
	Locale              string                `json:"locale"`
	GlossarySHA256      string                `json:"glossary_sha256"`
	UnitCount           int                   `json:"unit_count"`
	PageCount           int                   `json:"page_count"`
	ExampleCount        int                   `json:"example_count"`
	InputIdentitySHA256 string                `json:"input_identity_sha256"`
	Files               []TransportBundleFile `json:"files"`
}

func ExportGlossaryReviewerBundle(root, locale string, catalog *Catalog) ([]byte, GlossaryReviewerBundleManifest, error) {
	if catalog == nil {
		return nil, GlossaryReviewerBundleManifest{}, fmt.Errorf("glossary reviewer bundle catalog is required")
	}
	if err := ValidateLocaleName(locale); err != nil {
		return nil, GlossaryReviewerBundleManifest{}, err
	}
	if _, err := LoadGlossary(root, locale); err != nil {
		return nil, GlossaryReviewerBundleManifest{}, fmt.Errorf("validate glossary reviewer input: %w", err)
	}
	units, pageCount, exampleCount, err := localeWorkflowUnitList(catalog)
	if err != nil {
		return nil, GlossaryReviewerBundleManifest{}, err
	}
	context := GlossaryReviewerContext{
		SchemaVersion: GlossaryReviewerBundleSchemaVersion,
		Kind:          "go-tour-i18n/glossary-review-context", Locale: locale,
		Instruction: "Review the complete glossary and return only passed/failed plus findings. Do not generate replacement glossary content.",
		UnitCount:   len(units), PageCount: pageCount, ExampleCount: exampleCount,
		SourceUnits: make([]GlossaryReviewerSourceUnit, 0, len(units)),
	}
	for i, unit := range units {
		context.SourceUnits = append(context.SourceUnits, GlossaryReviewerSourceUnit{
			Index: i + 1, UnitID: unit.ID, UnitKind: unit.Kind, SourcePath: unit.SourcePath,
			SourceSHA256: unit.SourceSHA256, Source: string(unit.Source),
		})
	}
	contextData, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return nil, GlossaryReviewerBundleManifest{}, err
	}
	contextData = append(contextData, '\n')
	type entry struct {
		path, repositoryPath string
		data                 []byte
	}
	entries := []entry{{"glossary-review.json", "", contextData}}
	formal := []struct{ bundle, repository string }{
		{"formal/glossary.yaml", filepath.ToSlash(filepath.Join("locales", locale, "glossary.yaml"))},
		{"formal/locale.json", filepath.ToSlash(filepath.Join("locales", locale, "locale.json"))},
		{"formal/ui-en.json", "internal/tour/ui/en.json"},
		{"formal/tour-pages.tsv", "data/tour-pages.tsv"},
		{"formal/tour-examples.tsv", "data/tour-examples.tsv"},
	}
	var glossarySHA string
	for _, item := range formal {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(item.repository)))
		if err != nil {
			return nil, GlossaryReviewerBundleManifest{}, fmt.Errorf("read Glossary Review input %s: %w", item.repository, err)
		}
		if item.bundle == "formal/glossary.yaml" {
			glossarySHA = sum(data)
		}
		entries = append(entries, entry{item.bundle, item.repository, data})
	}
	for _, repoPath := range glossaryReviewerAuthorityPaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(repoPath)))
		if err != nil {
			return nil, GlossaryReviewerBundleManifest{}, fmt.Errorf("read Glossary Review authority %s: %w", repoPath, err)
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
	manifest := GlossaryReviewerBundleManifest{
		SchemaVersion: GlossaryReviewerBundleSchemaVersion,
		Kind:          "go-tour-i18n/glossary-reviewer-bundle", Locale: locale, GlossarySHA256: glossarySHA,
		UnitCount: len(units), PageCount: pageCount, ExampleCount: exampleCount,
		InputIdentitySHA256: sum([]byte(identity.String())), Files: files,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, GlossaryReviewerBundleManifest{}, err
	}
	manifestData = append(manifestData, '\n')
	bundle, err := WriteDeterministicTransportBundle(manifestData, zipEntries)
	if err != nil {
		return nil, GlossaryReviewerBundleManifest{}, err
	}
	return bundle, manifest, nil
}
