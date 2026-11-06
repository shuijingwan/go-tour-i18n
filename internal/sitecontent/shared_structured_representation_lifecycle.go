package sitecontent

import (
	"fmt"
	"reflect"
)

func (v *closureValidation) verifyArchivedSharedRepresentation(root string, a StructuredAsset, raw []byte, files map[string][]byte) error {
	ref, err := sharedDocumentAuthority(*a.Document)
	if err != nil {
		return err
	}
	if ref == nil {
		return nil
	}
	ctx := successorValidation(root, nil)
	r, err := readSharedRepresentationReceipt(root, *ref, raw, ctx, map[Reference]bool{})
	if err != nil {
		return err
	}
	if r.Request.Package != a.Package || r.Request.AssetID != a.ID || r.RegistrySHA != a.SourceSHA || !reflect.DeepEqual(sharedRepresentationDocument(*r), *a.Document) {
		return fmt.Errorf("archived shared representation owner mismatch")
	}
	for ref := *ref; ; {
		if digest(files["inputs/"+ref.Path]) != ref.SHA256 {
			return fmt.Errorf("shared representation receipt missing from frozen Bundle")
		}
		var node SharedRepresentationSuccessor
		if err := v.readReference(root, ref, &node); err != nil {
			return err
		}
		if node.Previous == nil {
			break
		}
		ref = *node.Previous
	}
	return ctx.aggregateVerify()
}

func (v *closureValidation) sharedRepresentationAssessment(root string, a, old StructuredAsset, raw []byte, current bool) (bool, error) {
	if a.Document == nil || old.Document == nil || !hasSharedDocumentAuthority(*a.Document) || a.ID != old.ID || a.Package != old.Package || a.SourceSHA != old.SourceSHA || a.TargetPath != old.TargetPath || a.Document.Path != old.Document.Path || a.Document.SourceSHA != digest(raw) || old.Document.SourceSHA != digest(raw) {
		return false, nil
	}
	ctx := successorValidation(root, nil)
	ref, err := sharedDocumentAuthority(*a.Document)
	if err != nil {
		return false, err
	}
	r, err := readSharedRepresentationReceipt(root, *ref, raw, ctx, map[Reference]bool{})
	if err != nil {
		return false, err
	}
	if !reflect.DeepEqual(sharedRepresentationDocument(*r), *a.Document) || r.Request.Package != a.Package || r.Request.AssetID != a.ID || r.RegistrySHA != a.SourceSHA {
		return false, fmt.Errorf("shared representation assessment owner mismatch")
	}
	if current {
		g, err := v.loadCurrent(root)
		if err != nil {
			return false, err
		}
		head, _, err := readSharedRepresentationHead(root, a.Package, a.ID, ctx)
		if err != nil {
			return false, err
		}
		if head == nil || *head != *ref {
			return false, fmt.Errorf("shared representation is not exact current head")
		}
		if err := checkSharedRepresentationCurrent(root, g, *r, ctx); err != nil {
			return false, err
		}
	}
	matched := false
	for depth := 0; depth < 32; depth++ {
		previous := shellDocument(r.Old)
		if r.Previous != nil {
			previous = withSharedDocumentAuthority(previous, *r.Previous)
		}
		if reflect.DeepEqual(previous, *old.Document) {
			matched = true
			break
		}
		if r.Previous == nil {
			break
		}
		r, err = readSharedRepresentationReceipt(root, *r.Previous, raw, ctx, map[Reference]bool{})
		if err != nil {
			return false, err
		}
	}
	if err := ctx.aggregateVerify(); err != nil {
		return false, err
	}
	return matched, nil
}
