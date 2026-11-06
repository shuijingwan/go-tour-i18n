package sitecontent

import (
	"fmt"
	"os"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Keep the original source-adoption context file byte-identical. Canonical
// representation is a separate, explicitly checked additive Bundle authority.
func (a ShellPageAdapter) RepresentationContextReferences(root string) ([]i18n.GlossaryArchiveReference, error) {
	if _, err := os.Lstat(pathJoin(root, ShellRepresentationCurrent)); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	s, r, err := selectShellRepresentation(root, a.Raw)
	if err != nil {
		return nil, err
	}
	if r == nil || !reflect.DeepEqual(*s, a.Source) {
		return nil, fmt.Errorf("Generation adapter does not match current representation authority")
	}
	var refs []i18n.GlossaryArchiveReference
	for _, name := range []string{ShellRepresentationCurrent, shellRepresentationReceipt(r).Path, "docs/SHELL_REPRESENTATION_SUCCESSOR.md"} {
		ref, err := ReferenceFile(root, name)
		if err != nil {
			return nil, err
		}
		refs = append(refs, i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256})
	}
	additional, err := shellReattestationContextReferences(root, r)
	if err != nil {
		return nil, err
	}
	for _, ref := range additional {
		refs = append(refs, i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256})
	}
	return refs, nil
}
