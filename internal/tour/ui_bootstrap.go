package tour

import (
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

// UIMessagesBootstrap exposes the exact localized message bootstrap prepended
// to Tour's script.js. Other consumers of the shared Playground runtime must
// use this same key set, strict catalog loader and HTML-safe JSON serializer.
func UIMessagesBootstrap(locale string) ([]byte, error) {
	catalog, err := ui.Load(locale)
	if err != nil {
		return nil, fmt.Errorf("load UI catalog %q: %w", locale, err)
	}
	return jsI18nBootstrap(catalog)
}
