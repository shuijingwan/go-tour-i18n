package sitecontent

import (
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Finding ownership is a view of existing closure/Generation authority. It
// never rewrites scope bytes, fresh/carried roles, or historical judgments.
func sharedCompositionFindingOwner(root string, scope IntegratedSurfaceScope, asset SurfaceAsset) (bool, error) {
	if len(asset.CandidateArtifacts) != 0 || asset.UnitID != "" || !component(asset.Package) {
		return false, nil
	}
	contains := func(refs []Reference, ref Reference) bool {
		for _, r := range refs {
			if r == ref {
				return true
			}
		}
		return false
	}
	for _, ref := range scope.Closures {
		if !contains(asset.Evidence, ref) {
			continue
		}
		for _, anchor := range scope.Reviewed {
			if anchor.Package != asset.Package || anchor.State != "context-review-required" || len(anchor.CandidateArtifacts) == 0 || !contains(anchor.Evidence, ref) {
				continue
			}
			closure, err := compositionClosure(root, ref, scope.Locale, asset.Package)
			if err != nil {
				return false, err
			}
			if closure.PageFinalization == nil {
				return false, fmt.Errorf("shared composition Page finalization missing")
			}
			var final i18n.QualityCheckFinalization
			if err := readReference(root, *closure.PageFinalization, &final); err != nil {
				return false, err
			}
			anchored := false
			for _, candidate := range anchor.CandidateArtifacts {
				for _, u := range final.Units {
					if final.Locale == scope.Locale && candidate.SnapshotID == final.SnapshotID && candidate.Unit.SourcePath == anchor.ID && candidate.Unit.CandidateSHA256 == anchor.TargetSHA && reflect.DeepEqual(candidate.Unit, u.Snapshot) {
						anchored = true
					}
				}
			}
			if !anchored {
				return false, fmt.Errorf("shared composition Page anchor mismatch")
			}
			owner, err := compositionStructuredOwner(root, ref, scope.Locale, asset, map[Reference]bool{})
			return owner.Path != "", err
		}
	}
	return false, nil
}

func compositionClosure(root string, ref Reference, locale, pkg string) (*PackageClosure, error) {
	var c PackageClosure
	if err := readReference(root, ref, &c); err != nil {
		return nil, err
	}
	copy := c
	copy.Identity = ""
	if c.Schema != ClosureSchema || c.Identity != identity(copy) || c.Locale != locale || c.Package != pkg || ref.Path != workflowPath(c.Locale, c.Package, "closure", c.Identity) {
		return nil, fmt.Errorf("shared composition closure identity mismatch")
	}
	return &c, nil
}

// Resolve only this exact structured target through the frozen closure chain.
// Current callers independently rebuild the complete current scope first;
// historical consumers preserve the sealed chain without replaying a current
// Page eligibility or an unrelated retired inventory against a later package.
func compositionStructuredOwner(root string, ref Reference, locale string, asset SurfaceAsset, visiting map[Reference]bool) (Reference, error) {
	if visiting[ref] {
		return Reference{}, fmt.Errorf("cyclic shared composition baseline")
	}
	visiting[ref] = true
	defer delete(visiting, ref)
	c, err := compositionClosure(root, ref, locale, asset.Package)
	if err != nil {
		return Reference{}, err
	}
	bound := false
	for _, f := range c.Files {
		if f.Path == asset.ID && f.SHA256 == asset.TargetSHA {
			bound = true
		}
	}
	if !bound {
		return Reference{}, fmt.Errorf("shared composition closure/target mismatch")
	}
	var owner Reference
	for _, gen := range c.Structured {
		var r StructuredGeneration
		if err := readReference(root, gen, &r); err != nil {
			return Reference{}, err
		}
		for _, a := range r.Plan.Assets {
			d := a.Document
			// The existing registered shared template/data representation owns
			// these slots. A Page or arbitrary prose/runtime file cannot claim it.
			if a.Package != asset.Package || d == nil || d.Path != asset.ID || d.Contract != ShellExtractionContract || d.Kind != "data" || d.UnitScope == PageContract {
				continue
			}
			produced := false
			for _, output := range r.Output.Outputs {
				produced = produced || output.Path == a.TargetPath
			}
			if !produced {
				continue
			}
			// The existing reader proves frozen corpus, registry, source,
			// representation, output exact-set, restoration and receipt identity.
			_, files, _, err := historicalStructuredGeneration(root, gen)
			if err != nil {
				return Reference{}, err
			}
			if digest(files[a.TargetPath]) != asset.TargetSHA {
				continue
			}
			if owner.Path != "" {
				return Reference{}, fmt.Errorf("ambiguous shared composition structured owner")
			}
			owner = gen
		}
	}
	if owner.Path != "" || c.StructuredBaseline == nil {
		return owner, nil
	}
	return compositionStructuredOwner(root, *c.StructuredBaseline, locale, asset, visiting)
}
