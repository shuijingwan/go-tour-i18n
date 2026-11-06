package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func (a ShellPageAdapter) GenerationContextReferences(root, locale string) ([]i18n.GlossaryArchiveReference, error) {
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	if g.ShellAdoption == nil {
		return nil, fmt.Errorf("Homepage Generation requires current shell adoption")
	}
	adoption, err := CheckShellAdoption(root, *g.ShellAdoption)
	if err != nil {
		return nil, err
	}
	review, err := i18n.RequireUnifiedGlossaryReview(root, locale)
	if err != nil {
		return nil, err
	}
	paths := []string{ShellAdoptionCurrent, g.ShellAdoption.Path, adoption.Foundation.Path, SourceManifestPath, GlobalPath, i18n.GlossaryCorpusPath, review.Path, "docs/SITE_SHELL_SUCCESSOR_AUTHORITY.md", "docs/SITE_V2_WORKFLOW.md", "docs/SITE_V2_ARCHITECTURE.md", a.Source.Path}
	refs := []i18n.GlossaryArchiveReference{}
	for _, name := range paths {
		ref, err := ReferenceFile(root, name)
		if err != nil {
			return nil, err
		}
		refs = append(refs, i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256})
	}
	return refs, nil
}
