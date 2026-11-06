package sitecontent

import (
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

// ProjectLanguageAuditText preserves plain visible strings as plain strings.
// In particular neither HTML-like text nor code-looking text becomes markup.
func ProjectLanguageAuditText(owner, locator, context, text string, identities []string) (languageaudit.Document, error) {
	if err := textBytes([]byte(text)); err != nil {
		return languageaudit.Document{}, err
	}
	pieces, err := languageaudit.BindIdentities(auditIdentityPieces([]languageaudit.Piece{{Text: text, Kind: languageaudit.Prose}}), identities)
	if err != nil {
		return languageaudit.Document{}, err
	}
	return languageaudit.Document{Owner: owner, Blocks: []languageaudit.Block{{Locator: locator, Context: context, Start: -1, End: -1, Pieces: pieces}}}, nil
}

func ProjectTourLanguageAudit(root string, c *i18n.Catalog, u *i18n.TranslationUnit, owner string, target []byte) (languageaudit.Document, languageaudit.Document, error) {
	source, localized, err := i18n.NativeTourLanguageAuditDocuments(root, c, u, target)
	source.Owner, localized.Owner = owner, owner
	return source, localized, err
}

func ProjectTourUIMessageAudit(owner, key string, message ui.Message, source ...ui.Message) (languageaudit.Document, error) {
	return i18n.ProjectTourUIMessageAudit(owner, key, message, source...)
}
