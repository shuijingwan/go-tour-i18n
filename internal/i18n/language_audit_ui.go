package i18n

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/languageview"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"
)

func PlainLanguageAuditDocument(owner, locator, context, text string, ids []string) (languageaudit.Document, error) {
	if !utf8.ValidString(text) || strings.ContainsAny(text, "\r\x00") || strings.HasPrefix(text, "\ufeff") {
		return languageaudit.Document{}, fmt.Errorf("invalid visible language text")
	}
	ps, err := languageaudit.BindIdentities(languageview.Plain(text), ids)
	if err != nil {
		return languageaudit.Document{}, err
	}
	return languageaudit.Document{Owner: owner, Blocks: []languageaudit.Block{{Locator: locator, Context: context, Start: -1, End: -1, Pieces: ps}}}, nil
}
func ProjectTourUIMessageAudit(owner, key string, message ui.Message, source ...ui.Message) (languageaudit.Document, error) {
	if _, err := ui.VisibleMessageText(message); err != nil {
		return languageaudit.Document{}, err
	}
	original := message
	if len(source) > 1 {
		return languageaudit.Document{}, fmt.Errorf("one source message required")
	}
	if len(source) == 1 {
		original = source[0]
	}
	if !reflect.DeepEqual(auditUIPlaceholder.FindAllString(original.Text, -1), auditUIPlaceholder.FindAllString(message.Text, -1)) {
		return languageaudit.Document{}, fmt.Errorf("UI source/target placeholder mismatch")
	}
	var d languageaudit.Document
	var err error
	switch message.Kind {
	case "plain":
		d, err = PlainLanguageAuditDocument(owner, "message/"+key, "ui-message", message.Text, nil)
	case "rich":
		d.Owner = owner
		d.Blocks, err = languageview.HTML([]byte(message.Text), languageview.Options{})
		for i := range d.Blocks {
			d.Blocks[i].Locator = "message/" + key + "/" + d.Blocks[i].Locator
			d.Blocks[i].Start = -1
			d.Blocks[i].End = -1
		}
	default:
		return languageaudit.Document{}, fmt.Errorf("unsupported UI kind %q", message.Kind)
	}
	for i := range d.Blocks {
		ps := []languageaudit.Piece{}
		for _, p := range d.Blocks[i].Pieces {
			if p.Kind != languageaudit.Prose && p.Kind != languageaudit.Link {
				ps = append(ps, p)
				continue
			}
			pos := 0
			for _, m := range auditUIPlaceholder.FindAllStringIndex(p.Text, -1) {
				if pos < m[0] {
					q := p
					q.Text = p.Text[pos:m[0]]
					ps = append(ps, q)
				}
				ps = append(ps, languageaudit.Piece{Kind: languageaudit.Opaque})
				pos = m[1]
			}
			if pos < len(p.Text) {
				q := p
				q.Text = p.Text[pos:]
				ps = append(ps, q)
			}
		}
		d.Blocks[i].Pieces = ps
	}
	return d, err
}

// Existing UI catalog contract tests freeze this placeholder grammar; dynamic
// values are opaque to a readonly language audit, never literal English prose.
var auditUIPlaceholder = regexp.MustCompile(`\{[a-z][a-z0-9_]*\}`)
