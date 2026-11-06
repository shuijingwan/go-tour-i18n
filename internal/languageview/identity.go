package languageview

import (
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Only positive evidence roles from the existing semantic classifier establish
// identity. Capitalization, a foreign script or punctuation alone never does.
// Heading/named-symbol inference is disabled: Command Prompt remains prose.
func Identities(in []languageaudit.Piece) []languageaudit.Piece {
	out := []languageaudit.Piece{}
	for _, p := range in {
		if p.Kind != languageaudit.Prose && p.Kind != languageaudit.Link {
			out = append(out, p)
			continue
		}
		s := p.Text
		start := 0
		for i := 0; i < len(s); {
			r, size := utf8.DecodeRuneInString(s[i:])
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				i += size
				continue
			}
			active := identityRuneScript(r)
			end := i + size
			for end < len(s) {
				next, n := utf8.DecodeRuneInString(s[end:])
				if unicode.IsSpace(next) || strings.ContainsRune("<>()[]{}*,;!?\"'&", next) {
					break
				}
				if unicode.IsLetter(next) {
					name := identityRuneScript(next)
					if active != "" && name != active {
						break
					}
					active = name
				} else if !unicode.IsDigit(next) && !unicode.IsMark(next) && !strings.ContainsRune("._/#:+-=", next) {
					break
				}
				end += n
			}
			word := strings.TrimRight(s[i:end], ".:;!?,")
			role := classifyProseTokenInRole(word, s[max(0, i-96):i], s[end:min(len(s), end+48)], false)
			if role == knownTechnicalAtom || role == technicalIdentifier || role == urlOrTarget || role == codeOrLiteral {
				if start < i {
					out = append(out, languageaudit.Piece{Text: s[start:i], Kind: p.Kind, Binding: p.Binding})
				}
				kind := languageaudit.Identity
				if p.Kind == languageaudit.Link {
					kind = languageaudit.TechnicalLink
				}
				out = append(out, languageaudit.Piece{Text: word, Kind: kind, Binding: p.Binding})
				start = i + len(word)
			}
			i = end
		}
		if start < len(s) {
			out = append(out, languageaudit.Piece{Text: s[start:], Kind: p.Kind, Binding: p.Binding})
		}
	}
	return out
}

func identityRuneScript(r rune) string {
	if !unicode.IsLetter(r) {
		return ""
	}
	for name, table := range unicode.Scripts {
		if unicode.Is(table, r) {
			return name
		}
	}
	return ""
}
