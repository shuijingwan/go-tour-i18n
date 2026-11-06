package sitecontent

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Structural rejection only. These hash-bound headers never stand in for
// Page integrity/current eligibility or frozen structured reconstruction.
// The ordinary full validators still run after this preflight succeeds.
func (v *closureValidation) preflightClosureInputs(locale, pkg string, page *Reference, structured []Reference, composition []i18n.TranslationFinalizationInput, frozenBaseline ...*Reference) error {
	defer v.phase("structural-preflight")()
	g, err := v.loadCurrent(v.root)
	if err != nil {
		return err
	}
	if _, err := packageByID(g, pkg); err != nil {
		return err
	}
	if page != nil && len(composition) > 0 {
		return fmt.Errorf("mixed closure Page finalization modes")
	}
	if page != nil {
		if err := v.preflightPageFinalization(locale, *page); err != nil {
			return err
		}
	}
	if len(composition) > 0 {
		refs := []Reference{}
		for _, in := range composition {
			refs = append(refs, Reference{in.Finalization.Path, in.Finalization.SHA256})
		}
		want, err := v.compositionInputs(v.root, pkg, refs)
		if err != nil || !reflect.DeepEqual(want, composition) {
			return fmt.Errorf("composition descriptor binding mismatch: %v", err)
		}
		for _, in := range composition {
			if err := v.preflightPageFinalization(locale, Reference{in.Finalization.Path, in.Finalization.SHA256}); err != nil {
				return err
			}
		}
	}
	l, err := readLocaleAuthority(v.root, g, locale)
	if err != nil {
		return err
	}
	// Traversal sets belong only to this structural walk, not a second proof
	// cache. Every reference is observed by the existing invocation graph.
	seen, active := map[Reference]bool{}, map[Reference]bool{}
	var closure func(Reference) error
	var generation func(Reference) error
	visit := func(ref Reference, read func() error) error {
		if active[ref] {
			return fmt.Errorf("cyclic closure authority")
		}
		if seen[ref] {
			return v.bind(ref)
		}
		active[ref] = true
		defer delete(active, ref)
		if err := read(); err != nil {
			return err
		}
		seen[ref] = true
		return nil
	}
	generation = func(ref Reference) error {
		return visit(ref, func() error {
			var r StructuredGeneration
			if err := v.readReference(v.root, ref, &r); err != nil {
				return err
			}
			for _, a := range r.Plan.Assets {
				if a.Assessment != nil {
					if err := closure(a.Assessment.Closure); err != nil {
						return err
					}
				}
			}
			return nil
		})
	}
	closure = func(ref Reference) error {
		return visit(ref, func() error {
			var c PackageClosure
			if err := v.readReference(v.root, ref, &c); err != nil {
				return err
			}
			copy := c
			copy.Identity = ""
			if c.Schema != ClosureSchema || c.Identity != identity(copy) || c.Locale != locale || ref.Path != workflowPath(c.Locale, c.Package, "closure", c.Identity) {
				return fmt.Errorf("historical closure identity mismatch")
			}
			finals := closurePageFinalizations(&c)
			if c.PageFinalization != nil && len(c.PageComposition) > 0 {
				return fmt.Errorf("mixed historical Page finalization modes")
			}
			if len(finals) > 0 {
				catalog, err := v.inventoryCatalog()
				if err != nil {
					return err
				}
				for i, ref := range finals {
					id, sha := c.Package, c.PackageSHA
					if len(c.PageComposition) > 0 {
						id, sha = c.PageComposition[i].Package, c.PageComposition[i].PackageAuthority
					}
					if _, err := i18n.HistoricalTranslationPackageCatalog(catalog, id, sha); err != nil {
						return err
					}
					if err := v.preflightPageFinalization(locale, ref); err != nil {
						return err
					}
				}
			}
			if c.StructuredBaseline != nil {
				if err := closure(*c.StructuredBaseline); err != nil {
					return err
				}
			}
			for _, ref := range c.Structured {
				if err := generation(ref); err != nil {
					return err
				}
			}
			return nil
		})
	}
	if len(frozenBaseline) > 0 && frozenBaseline[0] != nil {
		if err := closure(*frozenBaseline[0]); err != nil {
			return err
		}
	}
	for _, c := range l.Packages {
		if c.Package == pkg && c.State == "complete" && c.EvidenceKind == PackageCompletionKind {
			if err := closure(c.Evidence[0]); err != nil {
				return err
			}
		}
	}
	for _, ref := range structured {
		if err := generation(ref); err != nil {
			return err
		}
	}
	return nil
}

func (v *closureValidation) preflightPageFinalization(locale string, ref Reference) error {
	if !strings.HasPrefix(ref.Path, "data/quality-check-snapshots/") {
		return nil
	} // legacy has its own complete validator
	var f i18n.QualityCheckFinalization
	if err := v.readReference(v.root, ref, &f); err != nil {
		return err
	}
	if f.SchemaVersion != i18n.QualityCheckFinalizationSchemaVersion || f.EvidenceType != i18n.QualityCheckFinalizationEvidenceType || f.Locale != locale || ref.Path != "data/quality-check-snapshots/"+locale+"/"+f.SnapshotID+"/finalization.json" {
		return fmt.Errorf("Page finalization scope/path/schema mismatch")
	}
	return nil
}
