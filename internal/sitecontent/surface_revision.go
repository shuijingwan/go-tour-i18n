package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"reflect"
	"strings"
)

// FindingScope is explicit; free prose never mechanically authorizes arbitrary
// replacement files or Page units. This is not a synthetic QC rating.
type SurfaceFinding = i18n.SurfaceLanguageFinding

func validatedSurfaceFinding(root, locale string, ref Reference) (*IntegratedSurfaceReceipt, error) {
	return validatedSurfaceFindingWithScope(root, locale, ref, nil, BuildIntegratedSurfaceScope)
}

func validatedSurfaceFindingWithScope(root, locale string, ref Reference, catalog *i18n.Catalog, build integratedScopeBuilder) (*IntegratedSurfaceReceipt, error) {
	var r IntegratedSurfaceReceipt
	if err := readReference(root, ref, &r); err != nil {
		return nil, err
	}
	copy := r
	copy.Identity = ""
	if r.Identity != identity(copy) || r.Schema != IntegratedSurfaceSchema || r.Scope.Locale != locale || r.Decision != "failed" || len(r.ExactFindings) == 0 {
		return nil, fmt.Errorf("requires exact independent failed Surface finding")
	}
	if ref.Path != "data/locale-language/"+locale+"/surface/"+r.ReviewID+".json" || !component(r.Reviewer) || !component(r.ReviewID) {
		return nil, fmt.Errorf("invalid Surface finding path/session")
	}
	if err := requireIndependentSession(root, locale, r.Reviewer, "reviewer"); err != nil {
		return nil, err
	}
	b, err := readRegular(root, r.Evidence.Path)
	if err != nil || digest(b) != r.Evidence.SHA256 || textBytes(b) != nil {
		return nil, fmt.Errorf("Surface finding evidence changed")
	}
	want, err := build(root, locale, r.Scope.Closures)
	if err != nil || want == nil || !reflect.DeepEqual(r.Scope, *want) {
		if _, successorErr := representationRevisionScope(root, &r); successorErr != nil {
			return nil, fmt.Errorf("Surface finding context no longer current: %w", successorErr)
		}
	}
	for _, f := range r.ExactFindings {
		if f.Class == i18n.SurfaceTranslationUnitLanguage && catalog == nil {
			catalog, err = checkedCatalog(root)
			if err != nil {
				return nil, err
			}
			if err := AttachRegisteredTranslationPackagesReadonly(root, catalog); err != nil {
				return nil, err
			}
			break
		}
	}
	if err := validateIntegratedTUFindings(root, catalog, r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Only machine-owned judgment inputs determine routing. A foreign finding is
// skippable after full receipt validation, never because its ID is unknown.
func reviewedFindingAssets(root string, r *IntegratedSurfaceReceipt) (map[string]SurfaceAsset, error) {
	assets := map[string]SurfaceAsset{}
	for _, a := range r.Scope.Reviewed {
		if _, duplicate := assets[a.ID]; duplicate || !component(a.Package) {
			return nil, fmt.Errorf("ambiguous Surface finding owner")
		}
		assets[a.ID] = a
		if a.SourcePath != "" && a.SourcePath != a.ID {
			if previous, duplicate := assets[a.SourcePath]; duplicate {
				if previous.Package != a.Package {
					return nil, fmt.Errorf("ambiguous Surface finding owner")
				}
				// Native Tour sections can share a physical source path. Their
				// exact TU artifacts remain separately checked by the TU reader.
				previous.CandidateArtifacts = append(append([]i18n.SurfaceCandidateArtifact(nil), previous.CandidateArtifacts...), a.CandidateArtifacts...)
				assets[a.SourcePath] = previous
				continue
			}
			assets[a.SourcePath] = a
		}
	}
	seen := map[string]bool{}
	for _, f := range r.ExactFindings {
		if seen[f.ID] || f.Finding == "" {
			return nil, fmt.Errorf("Surface finding outside reviewed scope")
		}
		if _, ok := assets[f.ID]; !ok {
			for _, a := range r.Scope.Carried {
				if a.ID != f.ID && (a.SourcePath == "" || a.SourcePath != f.ID) {
					continue
				}
				if f.Class == i18n.SurfaceTranslationUnitLanguage && len(a.CandidateArtifacts) > 0 {
					assets[f.ID] = a
				} else if structuredLanguageFinding(f, a) {
					allowed, err := sharedCompositionFindingOwner(root, r.Scope, a)
					if err != nil {
						return nil, err
					}
					if allowed {
						assets[f.ID] = a
					}
				}
			}
		}
		a, ok := assets[f.ID]
		if !ok || ((f.Class == "structured-surface-language" || f.Class == "") && !structuredLanguageFinding(f, a)) {
			return nil, fmt.Errorf("Surface finding outside reviewed scope or class/owner mismatch")
		}
		seen[f.ID] = true
	}
	return assets, nil
}

func structuredLanguageFinding(f SurfaceFinding, a SurfaceAsset) bool {
	// Empty class is the original structured finding wire contract. It cannot
	// claim a formal TU asset or its candidate artifacts.
	return (f.Class == "structured-surface-language" || f.Class == "") && len(f.Artifacts) == 0 && len(a.CandidateArtifacts) == 0
}

func selectStructuredSurfaceRevision(root string, p *StructuredPlan, r *IntegratedSurfaceReceipt) error {
	assets, err := reviewedFindingAssets(root, r)
	if err != nil {
		return err
	}
	selected := map[string]bool{}
	for _, f := range r.ExactFindings {
		if f.Class == i18n.SurfaceTranslationUnitLanguage {
			continue
		}
		owner := assets[f.ID]
		if !structuredLanguageFinding(f, owner) {
			return fmt.Errorf("Surface structured finding class/owner mismatch")
		}
		matches := 0
		for _, a := range p.Assets {
			if a.Package != owner.Package {
				continue
			}
			if (a.Document != nil && a.Document.Path == f.ID) || (a.ID == "tour-ui" && strings.HasPrefix(f.ID, "ui:")) || (a.ID == "article-metadata" && strings.HasPrefix(f.ID, "article:")) {
				selected[a.TargetPath] = true
				matches++
			}
		}
		if matches != 1 {
			return fmt.Errorf("Surface finding requires one exact structured asset")
		}
	}
	p.Expected = []string{}
	for i := range p.Assets {
		a := &p.Assets[i]
		a.State = "context-only"
		if selected[a.TargetPath] {
			p.Expected = append(p.Expected, a.TargetPath)
			a.State = "generation-required"
		}
	}
	p.Expected = uniqueStrings(p.Expected)
	p.Identity = ""
	p.Identity = identity(*p)
	return nil
}

func surfaceRevisionPages(root, locale, pkg string, ref Reference, docs []Document) ([]string, error) {
	r, err := validatedSurfaceFinding(root, locale, ref)
	if err != nil {
		return nil, err
	}
	return selectSurfaceRevisionPages(root, r, pkg, docs)
}

func selectSurfaceRevisionPages(root string, r *IntegratedSurfaceReceipt, pkg string, docs []Document) ([]string, error) {
	assets, err := reviewedFindingAssets(root, r)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, finding := range r.ExactFindings {
		a := assets[finding.ID]
		if a.Package != pkg {
			continue
		}
		if finding.Class != i18n.SurfaceTranslationUnitLanguage {
			if structuredLanguageFinding(finding, a) {
				continue
			}
			return nil, fmt.Errorf("Surface Page finding class/owner mismatch")
		}
		matches := 0
		for _, d := range docs {
			if d.Path == finding.ID {
				if d.Kind != "page" || d.UnitScope != PageContract || len(d.Units) != 1 || len(finding.Artifacts) != 1 || finding.Artifacts[0].Unit.UnitID != d.Units[0].ID || finding.Artifacts[0].Unit.SourceSHA256 != d.SourceSHA || seen[d.Units[0].ID] {
					return nil, fmt.Errorf("Surface finding does not identify one exact formal Page")
				}
				ids, matches = append(ids, d.Units[0].ID), matches+1
				seen[d.Units[0].ID] = true
			}
		}
		if matches != 1 {
			return nil, fmt.Errorf("Surface finding requires one current Page Document")
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no formal Page finding")
	}
	return uniqueStrings(ids), nil
}
