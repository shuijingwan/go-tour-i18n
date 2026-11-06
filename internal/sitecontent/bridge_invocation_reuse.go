package sitecontent

import (
	"fmt"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type bridgeReconstructionKey struct {
	reference Reference
	mode      string
}

type cachedBridgeReconstruction struct {
	input  *i18n.Catalog
	result *i18n.SameUnitBridgeResolution
}

// Share the existing command-owned closure/projection scope. The i18n reader
// validates all disk dependencies before AND after every resolver invocation.
// This function only memoizes a deterministic typed reconstruction result.
func (v *closureValidation) resolveBridgeReconstruction(root string, c *i18n.Catalog, mode string, ref i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
	if root != v.root {
		return nil, fmt.Errorf("bridge invocation root drift")
	}
	// A construction decorator carries a readonly proof lifetime, not a new
	// representation. Keep it in the native callback but exclude that mutable
	// owner from the immutable key, as the Page replacement resolver does.
	construction := c
	c = originalBridgeCatalog(c)
	canonical := func() (*i18n.SameUnitBridgeResolution, error) {
		v.count("bridge-reconstruction-build:" + ref.Path)
		return build()
	}
	adopted := false
	for _, a := range c.Adapted {
		switch page := a.Adapter.(type) {
		case PageAdapter, ShellPageAdapter:
		case adoptedPageAdapter:
			// This closed adapter adds only its immutable root plus the current
			// corpus gate to PageAdapter. The root remains in the exact typed
			// key. Current and historical entries remain separate; foreign roots
			// and unknown domains retain their native reader.
			if (mode != "current" && mode != "historical") || page.root != root {
				return canonical()
			}
			adopted = true
		default:
			return canonical()
		}
	}
	input := *c
	input.BridgeResolver = nil
	input.BridgeConstructionContext = nil
	input.ReplacementResolver, input.SurfaceAuthorityReader, input.HistoricalContextReader, input.PreviewFailureReader = nil, nil, nil, nil
	key := bridgeReconstructionKey{Reference{Path: ref.Path, SHA256: ref.SHA256}, mode}
	for _, cached := range v.bridgeReconstructions[key] {
		if reflect.DeepEqual(&input, cached.input) {
			if adopted {
				// Re-observe the extra adapter condition on EVERY hit. It is not
				// a saved currentness verdict. An unavailable/failed preparation
				// falls back to the native builder, preserving its own result and
				// failure semantics even if this gate was outside its scope.
				v.count("bridge-source-observation:" + ref.Path)
				if err := observeBridgeCorpus(root, construction); err != nil {
					v.count("bridge-source-observation-fallback:" + ref.Path)
					return canonical()
				}
			}
			v.count("bridge-reconstruction-reuse:" + ref.Path)
			return cloneReplacementValue(reflect.ValueOf(cached.result)).Interface().(*i18n.SameUnitBridgeResolution), nil
		}
	}
	r, err := canonical()
	if err != nil {
		return nil, err
	}
	for _, a := range r.Adapters {
		if _, ok := a.(PageAdapter); !ok {
			return r, nil
		}
	}
	if len(v.bridgeReconstructions) < 8 {
		if v.bridgeReconstructions == nil {
			v.bridgeReconstructions = map[bridgeReconstructionKey][]cachedBridgeReconstruction{}
		}
		v.bridgeReconstructions[key] = append(v.bridgeReconstructions[key], cachedBridgeReconstruction{
			cloneReplacementValue(reflect.ValueOf(&input)).Interface().(*i18n.Catalog),
			cloneReplacementValue(reflect.ValueOf(r)).Interface().(*i18n.SameUnitBridgeResolution),
		})
	}
	return r, nil
}

// Preserve the native decorator's observation owner as well as its gate. An
// unwrapped caller remains eager. A wrapped caller may share only the proof
// lifetime already owned/closed by withBridgeConstructionEnvironment; this
// function creates no scope and cannot carry a verdict across invocations.
func observeBridgeCorpus(root string, c *i18n.Catalog) error {
	for _, unit := range c.Adapted {
		page, ok := unit.Adapter.(observedBridgePage)
		if !ok || page.root != root || page.proof == nil || page.proof.guard.root != root {
			continue
		}
		adopted, ok := page.original.(adoptedPageAdapter)
		if ok && adopted.root == root {
			return CheckUnifiedCorpus(root, page.proof)
		}
	}
	return CheckUnifiedCorpus(root)
}
