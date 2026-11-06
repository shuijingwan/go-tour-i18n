package sitecontent

import (
	"fmt"
	"reflect"
)

// Historical inventory follows the adoption transported with that generation,
// not today's registry. Only its exact predecessor-owned old Home is retired.
func (v *closureValidation) applyHistoricalShellRetirement(root string, b *structuredBaseline, visiting map[string]bool) error {
	if b.Closure.StructuredBaseline == nil {
		return nil
	}
	for _, fs := range b.Inputs {
		selector, ok := fs["inputs/"+ShellAdoptionCurrent]
		if !ok {
			continue
		}
		var ref Reference
		if err := StrictJSON(selector, &ref); err != nil {
			return err
		}
		var adoption ShellAdoption
		raw := fs["inputs/"+ref.Path]
		if !validSHA(ref.SHA256) || digest(raw) != ref.SHA256 {
			return fmt.Errorf("historical shell adoption binding mismatch")
		}
		if err := StrictJSON(raw, &adoption); err != nil {
			return err
		}
		copy := adoption
		copy.Identity = ""
		if adoption.Schema != ShellAdoptionSchema || adoption.Identity != identity(copy) || ref.Path != shellAdoptionPath(&adoption) {
			return fmt.Errorf("historical shell adoption identity mismatch")
		}
		if adoption.Package.ID != b.Closure.Package || adoption.Package.Identity != b.Closure.PackageSHA {
			continue
		}
		var foundation ShellSuccessorAuthority
		raw = fs["inputs/"+adoption.Foundation.Path]
		if !validSHA(adoption.Foundation.SHA256) || digest(raw) != adoption.Foundation.SHA256 {
			return fmt.Errorf("historical shell foundation binding mismatch")
		}
		if err := StrictJSON(raw, &foundation); err != nil {
			return err
		}
		fc := foundation
		fc.Identity = ""
		if foundation.Schema != ShellSuccessorSchema || foundation.State != "pending-source-authority" || foundation.Package != adoption.Package.ID || foundation.Identity != identity(fc) || adoption.Foundation.Path != ShellSuccessorRoot+"sources/"+foundation.Identity+".json" || foundation.ExtractionContract != ShellExtractionContract {
			return fmt.Errorf("historical shell foundation identity mismatch")
		}
		path := foundation.HistoricalHome.Path
		owner, exists := b.Owners[path]
		if !exists {
			continue
		}
		for _, source := range adoption.Sources {
			if source.Path == path {
				return fmt.Errorf("historical shell retirement still in adopted inventory")
			}
		}
		parent, err := v.historicalStructuredClosure(root, *b.Closure.StructuredBaseline, visiting)
		if err != nil {
			return err
		}
		if parent.Closure.Locale != b.Closure.Locale || parent.Closure.Package != b.Closure.Package || parent.Closure.PackageSHA != foundation.PredecessorPackageIdentity || parent.Owners[path] != owner || !reflect.DeepEqual(parent.Files[path], b.Files[path]) {
			return fmt.Errorf("historical shell retirement predecessor mismatch")
		}
		if err := v.bind(ref); err != nil {
			return err
		}
		if err := v.bind(adoption.Foundation); err != nil {
			return err
		}
		delete(b.Files, path)
		delete(b.Owners, path)
	}
	return nil
}

func authorizedShellRetirement(root string, g *Global, oldPackage, path string) (bool, error) {
	if g.ShellAdoption == nil {
		return false, nil
	}
	a, err := CheckShellAdoption(root, *g.ShellAdoption)
	if err != nil {
		return false, err
	}
	var foundation ShellSuccessorAuthority
	if err := readReference(root, a.Foundation, &foundation); err != nil {
		return false, err
	}
	return oldPackage == foundation.PredecessorPackageIdentity && path == foundation.HistoricalHome.Path, nil
}

func validateCurrentCompletedPackage(g *Global, c *PackageClosure) error {
	p, err := packageByID(g, c.Package)
	if err != nil {
		return err
	}
	if p.Identity != c.PackageSHA {
		return fmt.Errorf("historical package closure cannot authorize current publication")
	}
	return nil
}
