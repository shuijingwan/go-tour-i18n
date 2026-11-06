package sitecontent

import (
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Prepared adoption bodies belong to the existing readonly owner and guard.
// The type and exact immutable reference distinguish independent authorities.
type registrationAdoptionKey struct {
	kind string
	ref  Reference
}

// AttachRegisteredTranslationPackagesReadonly gives catalog construction one
// explicit owner. Every dependency remains guarded until the aggregate check
// before return. The context ends here, before a caller may import/process;
// default/public attachment and publication retain their eager semantics.
func AttachRegisteredTranslationPackagesReadonly(root string, catalog *i18n.Catalog) error {
	return withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		return AttachRegisteredTranslationPackages(root, catalog, v)
	})
}

func (v *structuredSuccessorValidation) registrationAdoption(root string, ref Reference, kind string, out any, build func() (any, error)) error {
	if root != v.guard.root || !v.readonlyOwner || !v.registrationReuse {
		return fmt.Errorf("readonly registration owner required")
	}
	key := registrationAdoptionKey{kind, ref}
	if b, ok := v.registrationAdoptions[key]; ok && !v.disabled {
		// Every hit still reads the exact receipt. All observed inputs remain
		// guarded until the owner's aggregate validation before success return.
		if err := v.guard.bind(ref); err != nil {
			return err
		}
		return StrictJSON(b, out) // private copy, never caller-owned cached data
	}
	if err := v.guard.readReference(root, ref, out); err != nil {
		return err
	}
	if err := v.observe(out); err != nil {
		return err
	}
	if err := v.guard.optional(i18n.GlossaryCorpusPath); err != nil {
		return err
	}
	// The normal eager checker performs every identity, source, parser,
	// package and canonical reconstruction check on the first exact authority.
	v.countHistorical("registration-" + kind + "-adoption-build")
	a, err := build()
	if err != nil {
		return err
	}
	if err := v.observe(a); err != nil {
		return err
	}
	b, err := Encode(a)
	if err != nil {
		return err
	}
	if err := StrictJSON(b, out); err != nil {
		return err
	}
	if v.registrationAdoptions == nil {
		v.registrationAdoptions = map[registrationAdoptionKey][]byte{}
	}
	v.registrationAdoptions[key] = b
	return nil
}
