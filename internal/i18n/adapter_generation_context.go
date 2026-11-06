package i18n

// TranslationGenerationContextAdapter adds representation-owned frozen source
// authority to the same provider-neutral Bundle. It adds no lifecycle state.
type TranslationGenerationContextAdapter interface {
	GenerationContextReferences(root, locale string) ([]GlossaryArchiveReference, error)
}

// TranslationRepresentationContextAdapter adds an independently adopted
// canonical representation authority without rewriting an older source
// adoption's frozen context provider or granting any language acceptance.
type TranslationRepresentationContextAdapter interface {
	RepresentationContextReferences(root string) ([]GlossaryArchiveReference, error)
}

func translationGenerationContexts(root, locale string, adapter TranslationUnitAdapter) ([]GlossaryArchiveReference, error) {
	var refs []GlossaryArchiveReference
	if a, ok := adapter.(TranslationGenerationContextAdapter); ok {
		r, err := a.GenerationContextReferences(root, locale)
		if err != nil {
			return nil, err
		}
		refs = append(refs, r...)
	}
	if a, ok := adapter.(TranslationRepresentationContextAdapter); ok {
		r, err := a.RepresentationContextReferences(root)
		if err != nil {
			return nil, err
		}
		refs = append(refs, r...)
	}
	return refs, nil
}
