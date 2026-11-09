package sitecontent

import "github.com/shuijingwan/go-tour-i18n/internal/i18n"

// Observe the real, source-owned typed inverse reader without changing adapter
// types or manufacturing a projection. Both historical and current comparison
// sides still pass through the production strict codec reader.
func snapshotProjectionProbeCatalog(c *i18n.Catalog, calls *int) *i18n.Catalog {
	out := *c
	out.DerivedTransportCodecReader = func(a i18n.TranslationUnitAdapter, codec, inventory []byte) (any, any, i18n.TranslationUnitAdapter, error) {
		*calls++
		return derivedApprovedCodec(a, codec, inventory)
	}
	return &out
}
