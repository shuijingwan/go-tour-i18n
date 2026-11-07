package i18n

import (
	"bytes"
	"fmt"
	"sort"
)

// Producer 2 adds a closed recovery construction reader. The legacy producer's
// absence of these bindings is a known version fact, not fabricated old source.
// A recorded v2 producer must retain its actual old implementation bytes.
var approvedConstructionReaderDeclarationsV2 = map[string][]string{
	"internal/i18n/transport_recovery_construction.go": {"func:typedApprovedConstruction", "func:deriveRecoveryConstruction", "func:decodeDerivedTransportConstruction", "func:derivedConstructionIdentity"},
	"internal/i18n/transport_recovery_context.go":      {"func:derivedRecoverySelection", "func:recoveryGenerationInputs", "func:currentRecoveryInvocationAuthority"},
	"internal/sitecontent/transport_recovery_codec.go": {"func:derivedApprovedCodec"},
}

func classifyRecoveryConstructionDifference(old, current *DerivedTransportConstruction, oldFiles, currentFiles map[string][]byte, root string, materials ...*historicalConstructionResolver) ([]RecoveryDependencyDelta, error) {
	if !bytes.Equal(mustJSON(old.SemanticContext), mustJSON(current.SemanticContext)) || !bytes.Equal(mustJSON(old.Decoders), mustJSON(current.Decoders)) {
		return nil, recoveryTypedBlocker{"UNCLASSIFIED_PROOF_MISMATCH", "semantic scope or decoder payload changed"}
	}
	prior := map[string]RecoveryDependency{}
	for _, r := range old.Producer.Implementation {
		prior[r.Role+"\x00"+r.Ref.Path] = r
	}
	var deltas []RecoveryDependencyDelta
	for _, r := range current.Producer.Implementation {
		x, ok := prior[r.Role+"\x00"+r.Ref.Path]
		if !ok {
			if old.Producer.Version == 1 && current.Producer.Version == 2 && r.Role == "approved-construction-reader" {
				continue
			}
			return nil, recoveryTypedBlocker{"UNCLASSIFIED_PROOF_MISMATCH", "unknown producer dependency addition"}
		}
		delete(prior, r.Role+"\x00"+r.Ref.Path)
		if x.Ref == r.Ref {
			continue
		}
		ob, err := recoveryArchiveReference(oldFiles, GlossaryArchiveReference{x.Ref.Path, x.Ref.SHA256})
		if len(materials) > 0 {
			ob, err = materials[0].ReadProducer(GlossaryArchiveReference{x.Ref.Path, x.Ref.SHA256}, r.Role, old.Producer.Version)
		}
		if err != nil {
			return nil, err
		}
		nb, err := readCompatibilityReference(root, GlossaryArchiveReference{r.Ref.Path, r.Ref.SHA256})
		if err != nil {
			return nil, err
		}
		var names []string
		if r.Role == "approved-construction-implementation" {
			names, err = classifyApprovedDependencyDelta(r.Ref.Path, ob, nb)
		} else {
			allowed, ok := approvedConstructionReaderDeclarationsV2[r.Ref.Path]
			if !ok {
				return nil, recoveryTypedBlocker{"UNCLASSIFIED_PROOF_MISMATCH", "non-construction recovery implementation changed"}
			}
			a, ae := recoveryGoDeclarations(r.Ref.Path, ob)
			b, be := recoveryGoDeclarations(r.Ref.Path, nb)
			if ae != nil || be != nil {
				return nil, fmt.Errorf("producer declaration parse failed")
			}
			keys := map[string]bool{}
			for k := range a {
				keys[k] = true
			}
			for k := range b {
				keys[k] = true
			}
			for k := range keys {
				if a[k] == b[k] {
					continue
				}
				owned := false
				for _, name := range allowed {
					if k == name {
						owned = true
					}
				}
				if !owned {
					return nil, recoveryTypedBlocker{"UNCLASSIFIED_PROOF_MISMATCH", "mixed-role recovery reader declaration changed"}
				}
				names = append(names, k)
			}
			if len(names) == 0 {
				err = fmt.Errorf("no construction declaration delta")
			}
		}
		if err != nil {
			return nil, recoveryTypedBlocker{"UNCLASSIFIED_PROOF_MISMATCH", err.Error()}
		}
		sort.Strings(names)
		deltas = append(deltas, RecoveryDependencyDelta{Role: r.Role, Old: x.Ref, Current: r.Ref, Declarations: names})
	}
	if len(prior) != 0 {
		return nil, recoveryTypedBlocker{"UNCLASSIFIED_PROOF_MISMATCH", "producer dependency removed"}
	}
	return deltas, nil
}
