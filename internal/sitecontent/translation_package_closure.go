package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"reflect"
	"sort"
)

// Descriptor projection after the existing shared finalization gate. This is
// the same PackageClosure artifact, not a package-specific acceptance policy.
func BuildTranslationPackageClosure(root string, catalog *i18n.Catalog, locale, pkg, snapshot string) (*PackageClosure, map[string][]byte, error) {
	if catalog == nil {
		return nil, nil, fmt.Errorf("registered package catalog required")
	}
	c := *catalog
	if err := c.UseTranslationPackage(pkg); err != nil {
		return nil, nil, err
	}
	files, err := i18n.FinalizedTranslationPackageCandidates(root, &c, locale, snapshot)
	if err != nil {
		return nil, nil, err
	}
	var descriptor i18n.TranslationPackageDescriptor
	for _, p := range c.TranslationPackages {
		if p.ID == pkg {
			descriptor = p
		}
	}
	if !validSHA(descriptor.AuthoritySHA256) {
		return nil, nil, fmt.Errorf("package closure requires frozen descriptor authority SHA")
	}
	final, err := ReferenceFile(root, "data/quality-check-snapshots/"+locale+"/"+snapshot+"/finalization.json")
	if err != nil {
		return nil, nil, err
	}
	closure := &PackageClosure{Schema: ClosureSchema, Locale: locale, Package: pkg, PackageSHA: descriptor.AuthoritySHA256, PageFinalization: &final, Structured: []Reference{}, Files: []Reference{}}
	for path, b := range files {
		closure.Files = append(closure.Files, Reference{Path: path, SHA256: digest(b)})
	}
	sort.Slice(closure.Files, func(i, j int) bool { return closure.Files[i].Path < closure.Files[j].Path })
	closure.Identity = identity(*closure)
	return closure, files, nil
}
func CheckTranslationPackageClosure(root string, c *i18n.Catalog, ref Reference) (*PackageClosure, map[string][]byte, error) {
	var closure PackageClosure
	if err := readReference(root, ref, &closure); err != nil {
		return nil, nil, err
	}
	if len(closure.PageComposition) > 0 {
		return CheckPackageClosure(root, ref)
	}
	if closure.PageFinalization == nil || ref.Path != workflowPath(closure.Locale, closure.Package, "closure", closure.Identity) {
		return nil, nil, fmt.Errorf("descriptor closure identity mismatch")
	}
	var f i18n.QualityCheckFinalization
	if err := readReference(root, *closure.PageFinalization, &f); err != nil {
		return nil, nil, err
	}
	want, files, err := BuildTranslationPackageClosure(root, c, closure.Locale, closure.Package, f.SnapshotID)
	if err != nil {
		return nil, nil, err
	}
	if !reflect.DeepEqual(closure, *want) {
		return nil, nil, fmt.Errorf("descriptor package closure stale")
	}
	return &closure, files, nil
}

// Inventory composition uses the existing Integrated Surface scope schema.
// Fixtures can register a future descriptor without touching public registry.
func BuildIntegratedSurfaceScopeForCatalog(root string, c *i18n.Catalog, locale string, closures []Reference) (*IntegratedSurfaceScope, error) {
	scope, err := BuildIntegratedSurfaceScope(root, locale, nil)
	if err != nil {
		return nil, err
	}
	return integratedSurfaceScopeForCatalog(root, c, *scope, closures)
}

func integratedSurfaceScopeForCatalog(root string, c *i18n.Catalog, base IntegratedSurfaceScope, closures []Reference) (*IntegratedSurfaceScope, error) {
	scope := &base
	locale := scope.Locale
	scope.Reviewed = append([]SurfaceAsset(nil), base.Reviewed...)
	scope.Routes = append([]string(nil), base.Routes...)
	scope.Closures = closures
	seen := map[string]bool{}
	for _, carried := range scope.ExistingCoverage.Packages {
		seen[carried.Package] = true
	}
	for _, ref := range closures {
		closure, _, err := CheckTranslationPackageClosure(root, c, ref)
		if err != nil {
			return nil, err
		}
		if closure.Locale != locale || seen[closure.Package] {
			return nil, fmt.Errorf("duplicate/wrong descriptor coverage")
		}
		seen[closure.Package] = true
		for _, file := range closure.Files {
			scope.Reviewed = append(scope.Reviewed, SurfaceAsset{ID: file.Path, SourcePath: file.Path, Package: closure.Package, State: "review-required", SourceSHA: closure.PackageSHA, TargetSHA: file.SHA256, Evidence: []Reference{ref}})
		}
		for _, d := range c.TranslationPackages {
			if d.ID == closure.Package {
				for _, u := range d.Units {
					scope.Routes = append(scope.Routes, u.Route)
				}
			}
		}
		for _, page := range closurePageFinalizations(closure) {
			var f i18n.QualityCheckFinalization
			if err := readReference(root, page, &f); err != nil {
				return nil, err
			}
			if err := BindIntegratedSurfaceCandidates(root, c, scope, &f); err != nil {
				return nil, err
			}
		}
	}
	sort.Strings(scope.Routes)
	sort.Slice(scope.Reviewed, func(i, j int) bool { return scope.Reviewed[i].ID < scope.Reviewed[j].ID })
	scope.Identity = ""
	scope.Identity = identity(*scope)
	return scope, nil
}
