package i18n

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var translationTokenRE = regexp.MustCompile(`⟪GTI18N_[0-9a-f]{8}_[0-9]{6}⟫`)
var translationKeepRE = regexp.MustCompile(`\b(?:Go|gofmt|PageUp|PageDown|Shift|Enter|Ctrl)\b`)
var whereToGoPrefixRE = regexp.MustCompile(`(?i)\bwhere[\t \r\n]+to[\t \r\n]+$`)
var directiveLineRE = regexp.MustCompile(`(?m)^\.(?:play|image)\s+[^\n]+$`)

type protectedTranslation struct {
	Text   string
	Tokens []string
	Values []string
	Kinds  []protectedTokenKind
}

type protectedTokenKind uint8

const (
	protectedOther protectedTokenKind = iota
	protectedInlineCode
	protectedDirective
	protectedLinkTarget
	protectedGlossaryOrKeep
	protectedPreformatted
	protectedPreformattedStatic
	protectedPreformattedIdentifier
)

type protectedSpan struct {
	start, end int
	restore    string
	kind       protectedTokenKind
}

func protectTranslation(source []byte, hash string, glossary *Glossary) protectedTranslation {
	text := string(source)
	spans := preformattedProtectionSpans(text)
	for _, m := range directiveLineRE.FindAllStringIndex(text, -1) {
		spans = append(spans, protectedSpan{start: m[0], end: m[1], kind: protectedDirective})
	}
	for _, m := range linkRE.FindAllStringSubmatchIndex(text, -1) {
		spans = append(spans, protectedSpan{start: m[2], end: m[3], kind: protectedLinkTarget})
		if glossary != nil {
			label := text[m[4]:m[5]]
			if key, wrapper := glossaryKeyForLabel(label, glossary.Mandatory); key != "" && key != "slides" {
				spans = append(spans, protectedSpan{start: m[4], end: m[5], restore: wrapper + glossary.Mandatory[key] + wrapper, kind: protectedGlossaryOrKeep})
			}
		}
	}
	for _, code := range presentInlineCodes(text) {
		spans = append(spans, protectedSpan{start: code.Start, end: code.End, kind: protectedInlineCode})
	}
	for _, m := range translationKeepRE.FindAllStringIndex(text, -1) {
		if !shouldProtectTranslationKeep(text, m[0], m[1]) {
			continue
		}
		spans = append(spans, protectedSpan{start: m[0], end: m[1], kind: protectedGlossaryOrKeep})
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start == spans[j].start {
			return spans[i].end > spans[j].end
		}
		return spans[i].start < spans[j].start
	})
	filtered := spans[:0]
	end := -1
	for _, span := range spans {
		if span.start < end {
			continue
		}
		filtered = append(filtered, span)
		end = span.end
	}
	prefix := hash
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	var out strings.Builder
	result := protectedTranslation{}
	pos := 0
	for i, span := range filtered {
		token := fmt.Sprintf("⟪GTI18N_%s_%06d⟫", prefix, i+1)
		out.WriteString(text[pos:span.start])
		out.WriteString(token)
		result.Tokens = append(result.Tokens, token)
		restore := span.restore
		if restore == "" {
			restore = text[span.start:span.end]
		}
		result.Values = append(result.Values, restore)
		result.Kinds = append(result.Kinds, span.kind)
		pos = span.end
	}
	out.WriteString(text[pos:])
	result.Text = out.String()
	return result
}

// shouldProtectTranslationKeep keeps all literal keep matches by default. A
// capitalized Go is not necessarily the Go language: English title case turns
// the ordinary verb in "Where to Go" into "Go". That interrogative infinitive
// construction is sufficiently specific to leave translatable; weaker clues
// such as "to Go" or "Go from" are deliberately not enough because they can
// name the language (for example, "migrate to Go from another language").
func shouldProtectTranslationKeep(text string, start, end int) bool {
	if text[start:end] != "Go" {
		return true
	}
	return !whereToGoPrefixRE.MatchString(text[:start])
}

func (p protectedTranslation) restore(output string) (string, []string) {
	var failures []string
	found := translationTokenRE.FindAllString(output, -1)
	if len(found) != len(p.Tokens) {
		failures = append(failures, fmt.Sprintf("protected token count = %d, want %d", len(found), len(p.Tokens)))
	}
	known := map[string]bool{}
	for _, token := range p.Tokens {
		known[token] = true
	}
	for _, token := range found {
		if !known[token] {
			failures = append(failures, "unknown protected token: "+token)
		}
	}
	for i, token := range p.Tokens {
		if n := strings.Count(output, token); n != 1 {
			failures = append(failures, fmt.Sprintf("token %d occurrence count = %d, want 1", i+1, n))
		}
	}
	if len(failures) != 0 {
		return "", failures
	}
	restored := normalizeInlineTokenBoundaries(output, p)
	values := make(map[string]string, len(p.Tokens))
	for i, token := range p.Tokens {
		values[token] = p.Values[i]
	}
	for _, token := range found {
		restored = strings.Replace(restored, token, values[token], 1)
	}
	if translationTokenRE.MatchString(restored) {
		return "", []string{"protected token remains after restoration"}
	}
	return restored, nil
}

// normalizeInlineTokenBoundaries adds only the whitespace required for each
// independently protected inline-code span to remain a distinct legacy present
// word. It runs while token identity is still available and never examines or
// changes the backticks inside a token's restoration value.
func normalizeInlineTokenBoundaries(output string, p protectedTranslation) string {
	found := translationTokenRE.FindAllString(output, -1)
	if len(found) != len(p.Tokens) {
		return output // restore calls this helper only after strict token validation.
	}
	kinds := make(map[string]protectedTokenKind, len(p.Tokens))
	for i, token := range p.Tokens {
		kinds[token] = p.Kinds[i]
	}
	segments := make([]string, len(found)+1)
	pos := 0
	for i, token := range found {
		offset := strings.Index(output[pos:], token)
		if offset < 0 {
			return output // restore calls this helper only after strict token validation.
		}
		segments[i] = output[pos : pos+offset]
		pos += offset + len(token)
	}
	segments[len(p.Tokens)] = output[pos:]

	for i, original := range segments {
		segment := original
		previousInline := i > 0 && kinds[found[i-1]] == protectedInlineCode
		nextInline := i < len(found) && kinds[found[i]] == protectedInlineCode

		if previousInline && startsWithNonBoundary(segment) {
			segment = " " + segment
		}
		if nextInline && endsWithNonBoundary(segment) {
			segment += " "
		}
		if previousInline && nextInline && !containsUnicodeSpace(original) && !endsWithUnicodeSpace(segment) {
			segment += " "
		}
		segments[i] = segment
	}

	var normalized strings.Builder
	for i, token := range found {
		normalized.WriteString(segments[i])
		normalized.WriteString(token)
	}
	normalized.WriteString(segments[len(p.Tokens)])
	return normalized.String()
}

func startsWithNonBoundary(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return !unicode.IsSpace(r) && !unicode.IsPunct(r)
}

func endsWithNonBoundary(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return !unicode.IsSpace(r) && !unicode.IsPunct(r)
}

func containsUnicodeSpace(s string) bool {
	return strings.IndexFunc(s, unicode.IsSpace) >= 0
}

func endsWithUnicodeSpace(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return unicode.IsSpace(r)
}
