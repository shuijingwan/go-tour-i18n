package i18n

import (
	"encoding/json"
	"fmt"
)

// HistoricalLocaleSurfaceLanguageInputsCompatible extends the existing Site
// v1 historical language closure, not the registry/current publication gate.
// Every non-glossary/config locale-owned identity remains exact.
func HistoricalLocaleSurfaceLanguageInputsCompatible(root, locale string, gate LocaleSurfaceReviewAGate, gateData []byte, current LocaleSurfaceReviewAInputs, catalog *Catalog) bool {
	a := gate.Inputs
	if gate.SchemaVersion >= localeSurfaceReviewASchemaVersionV4 {
		if !recordedTourSurfaceConfigProjectionValid(root, a) {
			return false
		}
		if current.TourLanguageContextSHA256 == "" {
			sha, err := currentTourLanguageContextSHA256(root, locale, catalog)
			if err != nil {
				return false
			}
			current.TourLanguageContextSHA256 = sha
		}
		if !validSHA256(a.TourLanguageContextSHA256) || a.TourLanguageContextSHA256 != current.TourLanguageContextSHA256 {
			return false
		}
		if current.TourConfigProjection == nil {
			p, err := currentTourSurfaceConfigProjections(root)
			if err != nil {
				return false
			}
			current.TourConfigProjection = p
		}
		if a.TourConfigProjection == nil || !tourConfigProjectionCompatible(a.TourConfigProjection.Project, current.TourConfigProjection.Project) || !tourConfigProjectionCompatible(a.TourConfigProjection.SEO, current.TourConfigProjection.SEO) {
			return false
		}
	}
	if a.GlossarySHA256 != current.GlossarySHA256 {
		if !glossaryScopeCompatible(root, locale, a.GlossarySHA256, current.GlossarySHA256, "*", catalog) {
			return false
		}
		a.GlossarySHA256 = current.GlossarySHA256
	}
	if a.ProjectConfigSHA256 != current.ProjectConfigSHA256 || a.SEOConfigSHA256 != current.SEOConfigSHA256 {
		if gate.SchemaVersion >= localeSurfaceReviewASchemaVersionV4 {
			if a.TourConfigProjection == nil || current.TourConfigProjection == nil || !tourConfigProjectionCompatible(a.TourConfigProjection.Project, current.TourConfigProjection.Project) || !tourConfigProjectionCompatible(a.TourConfigProjection.SEO, current.TourConfigProjection.SEO) {
				return false
			}
		} else if !historicalTourSurfaceConfigCompatible(root, locale, gate, gateData) {
			return false
		}
		a.ProjectConfigSHA256 = current.ProjectConfigSHA256
		a.SEOConfigSHA256 = current.SEOConfigSHA256
	}
	return a.UILocaleSHA256 == current.UILocaleSHA256 && a.GlossarySHA256 == current.GlossarySHA256 && a.ArticleMetadataSHA256 == current.ArticleMetadataSHA256 && a.CourseMetadataSHA256 == current.CourseMetadataSHA256 && a.CatalogSourceSHA256 == current.CatalogSourceSHA256 && a.CourseSourceDescriptionsSHA256 == current.CourseSourceDescriptionsSHA256 && a.CourseSourceDescriptionReviewSHA256 == current.CourseSourceDescriptionReviewSHA256 && a.ProjectConfigSHA256 == current.ProjectConfigSHA256 && a.SEOConfigSHA256 == current.SEOConfigSHA256 && (gate.SchemaVersion < 2 || a.ProductionPublicIdentitySHA256 == current.ProductionPublicIdentitySHA256)
}

// HistoricalTourSurfacePackage reconstructs ONLY the glossary/config inputs
// proved compatible with immutable historical authority. The caller must still
// compare the entire reconstructed package identity with its original digest.
// All other content, key sets, deployment facts and source bytes stay current.
func HistoricalTourSurfacePackage(root, locale, reviewID, glossarySHA string, data []byte, catalog *Catalog) ([]byte, error) {
	path, err := LocaleSurfaceReviewAGatePath(root, locale, reviewID)
	if err != nil {
		return nil, err
	}
	relative, err := repositoryRelativePath(root, path)
	if err != nil {
		return nil, err
	}
	gateData, err := readCompatibilityFile(root, relative)
	if err != nil {
		return nil, err
	}
	var gate LocaleSurfaceReviewAGate
	if err := decodeStrictCourseSourceDescriptionReviewJSON(gateData, &gate); err != nil {
		return nil, err
	}
	if err := validateLocaleSurfaceReviewAGate(gate, locale); err != nil {
		return nil, err
	}
	var pkg LocaleSurfaceReviewPackage
	if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &pkg); err != nil {
		return nil, err
	}
	if pkg.Locale != locale || gate.ReviewID != reviewID || !HistoricalLocaleSurfaceLanguageInputsCompatible(root, locale, gate, gateData, pkg.Inputs, catalog) {
		return nil, fmt.Errorf("historical language/config projection is stale")
	}
	if glossarySHA != pkg.Glossary.SHA256 {
		if !glossaryScopeCompatible(root, locale, glossarySHA, pkg.Glossary.SHA256, "*", catalog) {
			return nil, fmt.Errorf("historical glossary is affected")
		}
		bytes, err := readArchivedGlossary(root, locale, glossarySHA)
		if err != nil {
			return nil, err
		}
		pkg.Glossary.SHA256 = glossarySHA
		pkg.Glossary.Text = string(bytes)
		pkg.Inputs.GlossarySHA256 = glossarySHA
	}
	if gate.Inputs.ProjectConfigSHA256 != pkg.Inputs.ProjectConfigSHA256 || gate.Inputs.SEOConfigSHA256 != pkg.Inputs.SEOConfigSHA256 {
		var b TourSurfaceConfigBaseline
		if gate.SchemaVersion == localeSurfaceReviewASchemaVersionV4 {
			if !recordedTourSurfaceConfigProjectionValid(root, gate.Inputs) {
				return nil, fmt.Errorf("historical config bytes unavailable")
			}
			b.Project = GlossaryArchiveReference{"data/surface-config-history/" + gate.Inputs.ProjectConfigSHA256 + ".go", gate.Inputs.ProjectConfigSHA256}
			b.SEO = GlossaryArchiveReference{"data/surface-config-history/" + gate.Inputs.SEOConfigSHA256 + ".go", gate.Inputs.SEOConfigSHA256}
		} else {
			if !historicalTourSurfaceConfigCompatible(root, locale, gate, gateData) {
				return nil, fmt.Errorf("historical config bytes unavailable")
			}
			bytes, err := readCompatibilityFile(root, surfaceConfigBaselinePath(locale, reviewID))
			if err != nil {
				return nil, err
			}
			if err := decodeStrictCourseSourceDescriptionReviewJSON(bytes, &b); err != nil {
				return nil, err
			}
		}
		for _, input := range []struct {
			original string
			ref      GlossaryArchiveReference
		}{{"internal/tour/project.go", b.Project}, {"internal/tour/seo.go", b.SEO}} {
			ref, original := input.ref, input.original
			bytes, err := readCompatibilityFile(root, ref.Path)
			if err != nil {
				return nil, err
			}
			found := false
			for i := range pkg.OtherSurfaces {
				if pkg.OtherSurfaces[i].Path == original {
					pkg.OtherSurfaces[i].SHA256 = ref.SHA256
					pkg.OtherSurfaces[i].SourceText = string(bytes)
					found = true
				}
			}
			if !found {
				return nil, fmt.Errorf("missing exact historical config context")
			}
		}
		pkg.Inputs.ProjectConfigSHA256 = gate.Inputs.ProjectConfigSHA256
		pkg.Inputs.SEOConfigSHA256 = gate.Inputs.SEOConfigSHA256
	}
	output, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(output, '\n'), nil
}
