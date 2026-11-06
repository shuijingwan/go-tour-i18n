package sitecontent

import (
	"bytes"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Re-attesting an accepted target consumes the current representation authority.
// An adopted canonical Document may deliberately preserve its frozen protector
// while the general parser evolves. Neither raw-source equality nor a saved
// assessment alone authorizes that representation.
func (v *closureValidation) reconstructAssessedStructuredTarget(root string, a StructuredAsset, raw []byte, target StructuredTarget, glossary *i18n.Glossary) ([]byte, error) {
	values, err := structuredTargetValues(a, target, glossary)
	if err != nil {
		return nil, err
	}
	out, parserErr := reconstructAuthorityStructuredRepresentation(*a.Document, raw, values)
	if parserErr == nil {
		return out, nil
	}
	g, err := v.loadCurrent(root)
	if err != nil {
		return nil, err
	}
	// This reader verifies the current/canonical adoption, its frozen foundation
	// and exact source projection. Without such authority, it uses today's
	// parser; a stale Document will therefore fail the same exact comparison.
	docs, sources, err := CurrentPackageDocuments(root, g, a.Package)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if reflect.DeepEqual(d, *a.Document) && bytes.Equal(sources[d.Path], raw) {
			return reconstructFrozenStructuredRepresentation(d, raw, values)
		}
	}
	return nil, parserErr
}
