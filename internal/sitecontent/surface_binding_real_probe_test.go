package sitecontent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Explicit readonly proof of the production binding arguments. Closure/current
// eligibility is checked again by the full Surface builder, not inferred here.
func TestSurfaceCandidateBindingRealReadonly(t *testing.T) {
	out := os.Getenv("GO_TOUR_SURFACE_BINDING_READONLY")
	if out == "" {
		t.Skip("explicit real closure binding proof")
	}
	start := time.Now()
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	v := newClosureValidation(root)
	results := map[string]any{}
	err := withReadonlyStructuredValidation(root, func(owner *structuredSuccessorValidation) error {
		v.readonlySuccessors = owner
		catalog, err := v.inventoryCatalog()
		if err != nil {
			return err
		}
		tour := *catalog
		tour.TranslationScope = nil
		f, _, err := i18n.RequireCurrentCanonicalQualityCheck(root, "zh-CN", &tour)
		if err != nil {
			return err
		}
		s := &IntegratedSurfaceScope{Locale: "zh-CN"}
		for _, u := range f.Units {
			s.Carried = append(s.Carried, SurfaceAsset{ID: "tu:" + u.UnitID, SourcePath: u.Snapshot.SourcePath, UnitID: u.UnitID, TargetSHA: u.Snapshot.CandidateSHA256})
		}
		if err := BindIntegratedSurfaceCandidates(root, &tour, s, f); err != nil {
			return fmt.Errorf("Tour binding: %w", err)
		}
		results["tour"] = map[string]any{"result": "PASS", "units": len(f.Units)}
		for _, path := range []string{
			"data/site-workflow/zh-CN/learn-docs-v1/closure/a7ac3c4a6f9673a02784b593ca13216f76b6e8f3cb1f83c9a9b277c9940dcf2a.json",
			"data/site-workflow/zh-CN/site-v2-shell/closure/68cd5c0142c1a73e06375d45dce5e45a2f8441f6d682bdf7db58b7b4336d4e42.json",
		} {
			ref, err := ReferenceFile(root, path)
			if err != nil {
				return err
			}
			var closure PackageClosure
			if err := v.readReference(root, ref, &closure); err != nil {
				return err
			}
			c, err := v.packageCatalog(closure.Package)
			if err != nil {
				return err
			}
			s := &IntegratedSurfaceScope{Locale: "zh-CN"}
			for _, f := range closure.Files {
				s.Reviewed = append(s.Reviewed, SurfaceAsset{ID: f.Path, Package: closure.Package, TargetSHA: f.SHA256})
			}
			finals := []*i18n.QualityCheckFinalization{}
			for _, ref := range closurePageFinalizations(&closure) {
				var f i18n.QualityCheckFinalization
				if err := v.readReference(root, ref, &f); err != nil {
					return err
				}
				finals = append(finals, &f)
			}
			begin := time.Now()
			if err := BindIntegratedSurfaceCandidates(root, c, s, finals...); err != nil {
				results[closure.Package] = map[string]any{"result": "FAILED", "error": err.Error(), "current_scope": len(c.TranslationScope)}
				return fmt.Errorf("%s binding: %w", closure.Package, err)
			}
			count := 0
			for _, a := range s.Reviewed {
				count += len(a.CandidateArtifacts)
			}
			if count != len(c.TranslationScope) {
				return fmt.Errorf("binding omitted current Units: %d/%d", count, len(c.TranslationScope))
			}
			results[closure.Package] = map[string]any{"result": "PASS", "current_scope": len(c.TranslationScope), "bound_artifacts": count, "finalizations": len(finals), "seconds": time.Since(begin).Seconds()}
		}
		return v.verify()
	})
	results["seconds"] = time.Since(start).Seconds()
	if err != nil {
		results["error"] = err.Error()
	}
	b, _ := json.MarshalIndent(results, "", "  ")
	if writeErr := os.WriteFile(filepath.Join(out, "targeted-real-probe.json"), append(b, '\n'), 0600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Learn/Shell/Tour bindings PASS; readonly probe seconds=%.2f", time.Since(start).Seconds())
}
