package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Only timestamp recovery can consume this bridge. Neither the normal Bundle
// checker nor generic carry-forward recognizes it as an inventory exception.
type TimestampCurrentnessBridge struct {
	CurrentBundle   Reference                       `json:"current_bundle"`
	CurrentIdentity string                          `json:"current_bundle_identity_sha256"`
	Reattestations  []TimestampReattestationLineage `json:"reattestation_lineage"`
}

type TimestampReattestationLineage struct {
	Base                   Reference   `json:"representation_successor"`
	RepresentationIdentity string      `json:"representation_identity_sha256"`
	Old                    Reference   `json:"old_reattestation"`
	Current                Reference   `json:"current_reattestation"`
	Lineage                []Reference `json:"current_to_origin"`
}

func timestampContextMember(m StructuredBundle, fs map[string][]byte, path string) ([]byte, error) {
	b, ok := fs["inputs/"+path]
	matches := 0
	for _, f := range m.Files {
		if f.BundlePath == "inputs/"+path {
			if f.RepositoryPath != path || f.SHA256 != digest(b) || f.Size != int64(len(b)) {
				return nil, fmt.Errorf("invalid re-attestation context member: %s", path)
			}
			matches++
		}
	}
	if !ok || matches != 1 {
		return nil, fmt.Errorf("missing/duplicate re-attestation context member: %s", path)
	}
	return b, nil
}

// Every node is present in the frozen transport AND matches its immutable
// repository Reference. Existing family validation proves all base/source/
// topology/dependency invariants, including the complete Previous chain.
func timestampSnapshotLineage(root string, m StructuredBundle, fs map[string][]byte, head Reference, base Reference, want *SharedRepresentationReattestation, ctx *structuredSuccessorValidation) ([]Reference, error) {
	if _, err := readSharedReattestationLineage(root, head, want, ctx, map[Reference]bool{}, 0); err != nil {
		return nil, err
	}
	refs := []Reference{}
	seen := map[Reference]bool{}
	for ref := head; ; {
		if seen[ref] || len(refs) > 32 {
			return nil, fmt.Errorf("timestamp re-attestation cycle/depth")
		}
		seen[ref] = true
		b, err := timestampContextMember(m, fs, ref.Path)
		if err != nil {
			return nil, err
		}
		if digest(b) != ref.SHA256 {
			return nil, fmt.Errorf("frozen re-attestation SHA mismatch")
		}
		var r SharedRepresentationReattestation
		if err := ctx.guard.readReference(root, ref, &r); err != nil {
			return nil, err
		}
		if r.BaseReceipt != base {
			return nil, fmt.Errorf("different timestamp re-attestation base")
		}
		refs = append(refs, ref)
		if r.Previous == nil {
			break
		}
		ref = *r.Previous
	}
	return refs, nil
}

// Compare all non-reattestation semantics with the existing exact comparator.
// The only removable members are mechanically enumerated by validated changed
// heads and their complete chains. Unknown context files remain ordinary inputs.
func compatibleTimestampLineage(root string, old StructuredBundle, oldFiles map[string][]byte, current StructuredBundle, currentFiles map[string][]byte, fresh bool) ([]TimestampReattestationLineage, []Reference, error) {
	if old.AuthorityProfile != StructuredAuthorityProfile || current.AuthorityProfile != StructuredAuthorityProfile || !reflect.DeepEqual(old.Plan, current.Plan) || !reflect.DeepEqual(old.RevisionFinding, current.RevisionFinding) {
		return nil, nil, fmt.Errorf("timestamp recovery semantic Plan/Document/Expected/finding drift")
	}
	for _, path := range StructuredSemanticAuthorities {
		if !bytes.Equal(oldFiles["inputs/"+path], currentFiles["inputs/"+path]) {
			return nil, nil, fmt.Errorf("timestamp recovery semantic authority drift: %s", path)
		}
	}
	baseAssets := map[Reference]StructuredAsset{}
	for _, a := range old.Plan.Assets {
		if a.Document == nil {
			continue
		}
		ref, err := sharedDocumentAuthority(*a.Document)
		if err != nil {
			return nil, nil, err
		}
		if ref != nil {
			if previous, ok := baseAssets[*ref]; ok && (previous.ID != a.ID || previous.Package != a.Package) {
				return nil, nil, fmt.Errorf("ambiguous timestamp bridge owner")
			}
			baseAssets[*ref] = a
		}
	}
	bases := []Reference{}
	for ref := range baseAssets {
		bases = append(bases, ref)
	}
	sort.Slice(bases, func(i, j int) bool { return bases[i].Path < bases[j].Path })
	removeOld, removeCurrent := map[string]bool{}, map[string]bool{}
	proofs := []TimestampReattestationLineage{}
	refs := []Reference{}
	err := withReadonlyStructuredValidation(root, func(ctx *structuredSuccessorValidation) error {
		for _, baseRef := range bases {
			a := baseAssets[baseRef]
			baseBytes, err := timestampContextMember(old, oldFiles, baseRef.Path)
			if err != nil {
				return err
			}
			currentBase, err := timestampContextMember(current, currentFiles, baseRef.Path)
			if err != nil {
				return err
			}
			if !bytes.Equal(baseBytes, currentBase) || digest(baseBytes) != baseRef.SHA256 {
				return fmt.Errorf("timestamp bridge representation successor changed")
			}
			var header SharedRepresentationSuccessor
			if err := StrictJSON(baseBytes, &header); err != nil {
				return err
			}
			path := sharedReattestationPath(header.Identity)
			// Unchanged heads need no exception, including their inventory.
			ob, ook := oldFiles["inputs/"+path]
			cb, cok := currentFiles["inputs/"+path]
			if !ook && !cok {
				continue
			}
			if bytes.Equal(ob, cb) {
				continue
			}
			ob, err = timestampContextMember(old, oldFiles, path)
			if err != nil {
				return err
			}
			cb, err = timestampContextMember(current, currentFiles, path)
			if err != nil {
				return err
			}
			var oldHead, currentHead Reference
			if err := StrictJSON(ob, &oldHead); err != nil {
				return err
			}
			if err := StrictJSON(cb, &currentHead); err != nil {
				return err
			}
			if oldHead == currentHead {
				return fmt.Errorf("re-attestation head bytes changed without lineage change")
			}
			raw, err := ctx.guard.readRegular(root, a.Document.Path)
			if err != nil {
				return err
			}
			base, err := readSharedRepresentationReceipt(root, baseRef, raw, ctx, map[Reference]bool{})
			if err != nil {
				return err
			}
			if base.Request.Package != a.Package || base.Request.AssetID != a.ID || !reflect.DeepEqual(sharedRepresentationDocument(*base), *a.Document) {
				return fmt.Errorf("timestamp bridge representation/owner mismatch")
			}
			var node SharedRepresentationReattestation
			if err := ctx.guard.readReference(root, currentHead, &node); err != nil {
				return err
			}
			bp := sharedRepresentationHead(a.Package, a.ID)
			pointer, err := timestampContextMember(current, currentFiles, bp)
			if err != nil {
				return err
			}
			var selectedBase Reference
			if err := StrictJSON(pointer, &selectedBase); err != nil {
				return err
			}
			if selectedBase != baseRef || node.BaseReceipt != baseRef || node.BaseIdentity != base.Identity || node.RepresentationIdentity != base.NewIdentity || !reflect.DeepEqual(node.Base, *base) || node.BasePointer != (Reference{bp, digest(pointer)}) || !reflect.DeepEqual(node.OldInputs, base.Inputs) {
				return fmt.Errorf("timestamp bridge different representation/base authority")
			}
			if fresh {
				g, err := loadSourceCurrent(root, ctx)
				if err != nil {
					return err
				}
				if err := checkSharedRepresentationCurrent(root, g, *base, ctx); err != nil {
					return err
				}
				live, err := ctx.guard.readRegular(root, path)
				if err != nil {
					return err
				}
				if !bytes.Equal(live, cb) {
					return fmt.Errorf("timestamp bridge current re-attestation head changed")
				}
			}
			chain, err := timestampSnapshotLineage(root, current, currentFiles, currentHead, baseRef, &node, ctx)
			if err != nil {
				return err
			}
			oldChain, err := timestampSnapshotLineage(root, old, oldFiles, oldHead, baseRef, &node, ctx)
			if err != nil {
				return err
			}
			index := -1
			for i, ref := range chain {
				if ref == oldHead {
					index = i
					break
				}
			}
			if index < 1 || !reflect.DeepEqual(chain[index:], oldChain) {
				return fmt.Errorf("old re-attestation is not an exact current ancestor")
			}
			removeOld["inputs/"+path] = true
			removeCurrent["inputs/"+path] = true
			for _, ref := range oldChain {
				removeOld["inputs/"+ref.Path] = true
			}
			for _, ref := range chain {
				removeCurrent["inputs/"+ref.Path] = true
			}
			proofs = append(proofs, TimestampReattestationLineage{Base: baseRef, RepresentationIdentity: base.NewIdentity, Old: oldHead, Current: currentHead, Lineage: chain})
			refs = append(refs, baseRef)
			refs = append(refs, chain...)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(proofs) == 0 {
		return nil, nil, fmt.Errorf("no exact descendant shared re-attestation bridge")
	}
	filter := func(m StructuredBundle, remove map[string]bool) StructuredBundle {
		m.Files = append([]i18n.TransportBundleFile{}, m.Files...)
		kept := m.Files[:0]
		for _, f := range m.Files {
			if !remove[f.BundlePath] {
				kept = append(kept, f)
			}
		}
		m.Files = kept
		return m
	}
	if err := compatibleStructuredBundles(filter(old, removeOld), oldFiles, filter(current, removeCurrent), currentFiles); err != nil {
		return nil, nil, err
	}
	return proofs, refs, nil
}

// Canonical export performs the same complete current graph validation as the
// normal checker. Only this recovery compares a failed ordinary comparison via
// the additional exact lineage proof; no original bytes/identities are changed.
func timestampCurrentBundle(root string, old StructuredBundle, oldBytes []byte, oldFiles map[string][]byte) (*TimestampCurrentnessBridge, []byte, error) {
	b, current, err := exportStructuredAssets(root, old.Plan.Locale, old.RevisionFinding)
	if err != nil {
		return nil, nil, err
	}
	if bytes.Equal(b, oldBytes) {
		return nil, nil, nil
	}
	if old.AuthorityProfile != StructuredAuthorityProfile {
		return nil, nil, fmt.Errorf("timestamp recovery original Bundle profile is noncanonical")
	}
	_, files, err := readStructuredBundle(b)
	if err != nil {
		return nil, nil, err
	}
	if err := compatibleStructuredBundles(old, oldFiles, *current, files); err == nil {
		return nil, nil, nil
	}
	proof, _, err := compatibleTimestampLineage(root, old, oldFiles, *current, files, true)
	if err != nil {
		return nil, nil, err
	}
	return &TimestampCurrentnessBridge{CurrentBundle: structuredBundleRef(old.Plan.Locale, b), CurrentIdentity: current.Identity, Reattestations: proof}, b, nil
}
