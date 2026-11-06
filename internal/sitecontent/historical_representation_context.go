package sitecontent

import (
	"fmt"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func readHistoricalLanguageContext(root, locale string, ref i18n.GlossaryArchiveReference) ([]byte, error) {
	if strings.HasPrefix(ref.Path, ShellRepresentationReattestationRoot+"current/") {
		return readHistoricalShellReattestationContext(root, locale, ref)
	}
	return readHistoricalActivationContext(root, locale, ref)
}

// Sealed history binds the original pointer SHA. A subsequent re-attestation
// legitimately replaces that pointer, but retains its immutable receipt in the
// exact previous lineage. Recover only the canonical Reference bytes in memory;
// neither the checkpoint nor any filesystem evidence is rewritten. This proves
// historical integrity, never current dependency or language eligibility.
func readHistoricalShellReattestationContext(root, locale string, ref i18n.GlossaryArchiveReference) ([]byte, error) {
	if err := i18n.ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	prefix := ShellRepresentationReattestationRoot + "current/"
	baseID := strings.TrimSuffix(strings.TrimPrefix(ref.Path, prefix), ".json")
	if !validSHA(baseID) || ref.Path != shellReattestationPath(baseID) || !validSHA(ref.SHA256) {
		return nil, fmt.Errorf("not an exact historical Shell re-attestation pointer")
	}
	if err := shellRepresentationReady(root, false); err != nil {
		return nil, err
	}
	var frozen []byte
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		head, _, err := readShellReattestationHead(root, ref.Path, v)
		if err != nil {
			return err
		}
		if head == nil {
			return fmt.Errorf("historical Shell re-attestation head missing")
		}
		var latest ShellRepresentationReattestation
		if err := v.guard.readReference(root, *head, &latest); err != nil {
			return err
		}
		var base ShellRepresentationSuccessor
		if err := v.guard.readReference(root, latest.BaseReceipt, &base); err != nil {
			return err
		}
		copy := base
		copy.Identity = ""
		if base.Schema != ShellRepresentationSchema || base.Identity != baseID || base.Identity != identity(copy) || shellRepresentationReceipt(&base) != latest.BaseReceipt {
			return fmt.Errorf("invalid historical Shell representation base")
		}
		basePointer, err := Encode(latest.BaseReceipt)
		if err != nil {
			return err
		}
		// Use the immutable base, not the head's own assertions, to constrain
		// every node's package/source/representation and old dependency set.
		want := ShellRepresentationReattestation{
			BasePointer: Reference{ShellRepresentationCurrent, digest(basePointer)},
			BaseReceipt: latest.BaseReceipt, BaseIdentity: base.Identity,
			Basis: base.Basis, Representation: base.New, Released: base.Released, OldInputs: base.Inputs,
		}
		if _, err := readShellReattestationLineage(root, *head, &want, v, map[Reference]bool{}, 0); err != nil {
			return err
		}
		// Validate the entire chain before selecting, including ancestors
		// older than the requested node. No orphan/sibling/latest scan.
		for node := head; node != nil; {
			b, err := Encode(*node)
			if err != nil {
				return err
			}
			if digest(b) == ref.SHA256 {
				frozen = b
				return nil
			}
			var r ShellRepresentationReattestation
			if err := v.guard.readReference(root, *node, &r); err != nil {
				return err
			}
			node = r.Previous
		}
		return fmt.Errorf("historical pointer SHA has no exact Shell re-attestation ancestor")
	})
	if err != nil {
		return nil, err
	}
	if err := shellRepresentationReady(root, false); err != nil {
		return nil, err
	}
	return frozen, nil
}
