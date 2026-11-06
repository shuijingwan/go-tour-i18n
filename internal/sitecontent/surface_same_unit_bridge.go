package sitecontent

import (
	"bytes"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// The sealed codec already names every source boundary/atom. Compose its
// passed response back into an ordinary source-owned response, retaining the
// revised opaque atom values as opaque. No target text is protected or searched.
func (a PageAdapter) BridgeSurfaceRevision(decoder i18n.TranslationUnitAdapter, raw, reviewed []byte) (i18n.TranslationUnitAdapter, []byte, []byte, error) {
	p, ok := decoder.(*surfacePageTransport)
	if !ok || p.Ordinary || len(a.Document.Units) != 1 || p.UnitID != a.Document.Units[0].ID || p.CanonicalSourceSHA256 != a.Document.SourceSHA || p.Origin.Document.Path != a.Document.Path || len(p.Origin.Document.Units) != 1 || len(p.Origin.Document.Slots) != len(a.Document.Slots) || digest(p.Origin.Source) != p.Origin.Document.SourceSHA {
		return nil, nil, nil, fmt.Errorf("Surface bridge ordinary source mapping mismatch")
	}
	got, err := p.Restore(string(raw))
	if err != nil || !bytes.Equal(bridgeEOF(got), bridgeEOF(reviewed)) {
		return nil, nil, nil, fmt.Errorf("Surface bridge old response mismatch: %v", err)
	}
	origin, ordinary, err := p.ordinaryResponse(string(raw))
	if err != nil {
		return nil, nil, nil, err
	}
	template := bridgeEOF([]byte(ordinary))
	got, err = origin.Restore(string(template))
	if err != nil || !bytes.Equal(bridgeEOF(got), bridgeEOF(reviewed)) {
		return nil, nil, nil, fmt.Errorf("Surface bridge exact ordinary restore failed: %v", err)
	}
	oldOrigin, oldOrdinary, err := p.ordinaryResponse(p.Template)
	if err != nil {
		return nil, nil, nil, err
	}
	old, err := oldOrigin.Restore(oldOrdinary)
	baseline, baselineErr := p.Restore(p.Template)
	if err != nil || baselineErr != nil || !bytes.Equal(bridgeEOF(old), bridgeEOF(baseline)) {
		return nil, nil, nil, fmt.Errorf("Surface bridge ordinary ancestor mismatch")
	}
	return origin, template, bridgeEOF([]byte(oldOrdinary)), nil
}
