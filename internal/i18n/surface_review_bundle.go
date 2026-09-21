package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const localeSurfaceReviewReviewerBundleSchemaVersion = 1

var localeSurfaceReviewReviewerAuthorityPaths = []string{
	"AGENTS.md",
	"docs/CHATGPT_LANGUAGE_GENERATION.md",
	"docs/COURSE_SEO_METADATA.md",
	"docs/GLOSSARY_REVIEW.md",
	"docs/LOCALE_SURFACE_REVIEW.md",
	"docs/TRANSLATION_QUALITY_REVIEW.md",
}

type LocaleSurfaceReviewReviewerBundleFile = TransportBundleFile

type LocaleSurfaceReviewReviewerBundleManifest struct {
	SchemaVersion int                                     `json:"schema_version"`
	Kind          string                                  `json:"kind"`
	Locale        string                                  `json:"locale"`
	ReviewPackage LocaleSurfaceReviewReviewerBundleFile   `json:"review_package"`
	Coverage      LocaleSurfaceReviewPackageCoverage      `json:"coverage"`
	Authority     []LocaleSurfaceReviewReviewerBundleFile `json:"authority"`
}

// ExportLocaleSurfaceReviewReviewerBundle returns a deterministic ZIP transport
// for an external Reviewer session. The existing surface-review package remains
// the formal review input; the ZIP only co-locates that package with the current
// Stage A authority documents and a hash manifest so the Reviewer does not need
// to re-read the same repository files through a remote filesystem tool.
func ExportLocaleSurfaceReviewReviewerBundle(root, locale string, catalog *Catalog) ([]byte, LocaleSurfaceReviewReviewerBundleManifest, error) {
	packageData, coverage, err := ExportLocaleSurfaceReviewPackage(root, locale, catalog)
	if err != nil {
		return nil, LocaleSurfaceReviewReviewerBundleManifest{}, err
	}

	entries := []TransportBundleEntry{{Path: "surface-review.json", Data: packageData}}
	authority := make([]LocaleSurfaceReviewReviewerBundleFile, 0, len(localeSurfaceReviewReviewerAuthorityPaths))
	for _, repositoryPath := range localeSurfaceReviewReviewerAuthorityPaths {
		path := filepath.Join(root, filepath.FromSlash(repositoryPath))
		info, err := os.Stat(path)
		if err != nil {
			return nil, LocaleSurfaceReviewReviewerBundleManifest{}, fmt.Errorf("stat Surface Review authority %s: %w", repositoryPath, err)
		}
		if !info.Mode().IsRegular() {
			return nil, LocaleSurfaceReviewReviewerBundleManifest{}, fmt.Errorf("Surface Review authority %s is not a regular file", repositoryPath)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, LocaleSurfaceReviewReviewerBundleManifest{}, fmt.Errorf("read Surface Review authority %s: %w", repositoryPath, err)
		}
		bundlePath := filepath.ToSlash(filepath.Join("authority", repositoryPath))
		authority = append(authority, LocaleSurfaceReviewReviewerBundleFile{
			BundlePath: bundlePath, RepositoryPath: repositoryPath, SHA256: sum(data),
		})
		entries = append(entries, TransportBundleEntry{Path: bundlePath, Data: data})
	}

	manifest := LocaleSurfaceReviewReviewerBundleManifest{
		SchemaVersion: localeSurfaceReviewReviewerBundleSchemaVersion,
		Kind:          "go-tour-i18n/locale-surface-review-reviewer-bundle",
		Locale:        locale,
		ReviewPackage: LocaleSurfaceReviewReviewerBundleFile{BundlePath: "surface-review.json", SHA256: sum(packageData)},
		Coverage:      coverage,
		Authority:     authority,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, LocaleSurfaceReviewReviewerBundleManifest{}, err
	}
	manifestData = append(manifestData, '\n')

	bundle, err := WriteDeterministicTransportBundle(manifestData, entries)
	if err != nil {
		return nil, LocaleSurfaceReviewReviewerBundleManifest{}, err
	}
	return bundle, manifest, nil
}
