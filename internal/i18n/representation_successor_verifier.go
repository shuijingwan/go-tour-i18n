package i18n

import (
	"bytes"
	"fmt"
)

// RepresentationSuccessorVerifier is an invocation-local reader of the existing
// applied reconciliation registry. It selects no translation candidate and
// grants no language judgment. Every edge repeats the native machine verifier;
// only its shared preparation/context is reused within this invocation.
type RepresentationSuccessorVerifier struct {
	root, locale string
	catalog      *Catalog
	selections   *latestRetranslationUnits
}

func NewRepresentationSuccessorVerifier(root string, c *Catalog, locale string) *RepresentationSuccessorVerifier {
	return &RepresentationSuccessorVerifier{root: root, catalog: c, locale: locale}
}

// Verify accepts exact identities, never a source path/index or a latest batch.
// A nil mapping means unchanged. A changed identity requires the native applied
// receipt and all its materialized raw/candidate/validation proof artifacts.
func (v *RepresentationSuccessorVerifier) Verify(oldID, currentID string) (*TranslationUnitReplacement, []GlossaryArchiveReference, error) {
	return v.verify(oldID, currentID, false)
}

// VerifyHistorical proves an applied transition in its receipt-bound archived
// glossary world. It grants no current Generation/publication eligibility.
func (v *RepresentationSuccessorVerifier) VerifyHistorical(oldID, currentID string) (*TranslationUnitReplacement, []GlossaryArchiveReference, error) {
	return v.verify(oldID, currentID, true)
}

func (v *RepresentationSuccessorVerifier) verify(oldID, currentID string, historical bool) (*TranslationUnitReplacement, []GlossaryArchiveReference, error) {
	if err := ValidateLocaleName(v.locale); err != nil {
		return nil, nil, err
	}
	if v.catalog == nil || oldID == "" {
		return nil, nil, fmt.Errorf("explicit representation successor catalog/identity required")
	}
	u, err := v.catalog.Unit(currentID)
	if err != nil || sum(u.Source) != u.SourceSHA256 {
		return nil, nil, fmt.Errorf("representation successor current source/identity mismatch: %s", currentID)
	}
	if oldID == currentID {
		return nil, nil, nil
	}
	if v.selections == nil {
		ordered, _, _, err := localeWorkflowUnitList(v.catalog)
		if err != nil {
			return nil, nil, err
		}
		// Reuse the native receipt loader/index, without numeric batch scanning
		// or candidate fallback. Ordinary passed revisions are not replaced.
		index := &latestRetranslationUnits{ordered: ordered, selectedByID: map[string]selectedRetranslationUnit{}}
		if err := attachMachineSelections(v.root, v.catalog, v.locale, index); err != nil {
			return nil, nil, err
		}
		v.selections = index
	}
	s := v.selections.selectedByID[currentID].machine
	if s == nil || s.unit.Replacement.OldUnitID != oldID || s.unit.Replacement.CurrentUnitID != currentID {
		return nil, nil, fmt.Errorf("historical mapping lacks verified current successor: %s", oldID)
	}
	// Native apply publishes this canonical receipt and all artifacts by atomic
	// rename. Options.Apply is normalized to false even in an applied receipt;
	// it is not an applied-state flag. A preflight plan without artifacts fails
	// the native verifier below. Recheck bytes even when the index is warm.
	b, err := readSnapshotReferencedFile(v.root, s.ref.Path)
	if err != nil || sum(b) != s.ref.SHA256 || s.receipt.Options.Apply || !bytes.Equal(b, append(mustJSON(s.receipt), '\n')) {
		return nil, nil, fmt.Errorf("reconciliation receipt hash/canonical identity mismatch")
	}
	if _, err := s.validateInWorld(v.root, v.catalog, historical); err != nil {
		return nil, nil, err
	}
	refs := machineAuthorityReferences(s)
	if historical {
		path, _ := glossaryArchivePath(v.locale, s.receipt.GlossarySHA256)
		refs = append(refs, GlossaryArchiveReference{Path: path, SHA256: s.receipt.GlossarySHA256})
	}
	refs = append(refs,
		GlossaryArchiveReference{Path: s.unit.Predecessor.CandidatePath, SHA256: s.unit.Predecessor.CandidateSHA256},
		GlossaryArchiveReference{Path: s.unit.Predecessor.ValidationPath, SHA256: s.unit.Predecessor.ValidationSHA256})
	for _, q := range s.receipt.QCInputs {
		refs = append(refs, GlossaryArchiveReference{Path: q.Path, SHA256: q.SHA256})
	}
	if historical {
		// Preparation may share derived objects, never an earlier disk PASS.
		// Historical consumers also bind these references into their full proof.
		for _, ref := range refs {
			b, err := readSnapshotReferencedFile(v.root, ref.Path)
			if err != nil || sum(b) != ref.SHA256 {
				return nil, nil, fmt.Errorf("historical reconciliation reference drift: %s", ref.Path)
			}
		}
	}
	p := s.unit.Replacement
	return &p, refs, nil
}
