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
	Text             string
	Tokens           []string
	Values           []string
	Kinds            []protectedTokenKind
	InlineBoundaries []bool
	InlinePairs      []protectedInlinePair
	EmphasisTokens   []string
}

type protectedTokenKind uint8

const (
	protectedOther protectedTokenKind = iota
	protectedInlineCodeOpen
	protectedInlineCodeClose
	protectedDirective
	protectedLinkTarget
	protectedGlossaryOrKeep
	protectedPreformatted
	protectedPreformattedStatic
	protectedPreformattedIdentifier
	protectedItalicOpen
	protectedItalicClose
	protectedBoldOpen
	protectedBoldClose
)

type protectedSpan struct {
	start, end       int
	restore          string
	kind             protectedTokenKind
	inlineBoundaries bool
	inlinePair       int
	inlineContent    string
}

// protectedInlinePair keeps code visible to the model while requiring its
// exact source bytes between a pair of structural sentinels.
type protectedInlinePair struct {
	Open, Close string
	Content     string
	Boundaries  bool
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
	inlinePair := 0
	inlineCodes := append([]presentInlineCode{}, presentInlineCodes(text)...)
	for _, code := range inlineCodes {
		inlinePair++
		spans = append(spans,
			protectedSpan{start: code.Start, end: code.Start + 1, restore: "`", kind: protectedInlineCodeOpen, inlineBoundaries: true, inlinePair: inlinePair, inlineContent: text[code.Start+1 : code.End-1]},
			protectedSpan{start: code.End - 1, end: code.End, restore: "`", kind: protectedInlineCodeClose, inlineBoundaries: true, inlinePair: inlinePair, inlineContent: text[code.Start+1 : code.End-1]},
		)
	}
	linkCodes := linkLabelInlineCodes(text)
	inlineCodes = append(inlineCodes, linkCodes...)
	for _, code := range linkCodes {
		inlinePair++
		spans = append(spans,
			protectedSpan{start: code.Start, end: code.Start + 1, restore: "`", kind: protectedInlineCodeOpen, inlinePair: inlinePair, inlineContent: text[code.Start+1 : code.End-1]},
			protectedSpan{start: code.End - 1, end: code.End, restore: "`", kind: protectedInlineCodeClose, inlinePair: inlinePair, inlineContent: text[code.Start+1 : code.End-1]},
		)
	}
	spans = append(spans, presentEmphasisDelimiterSpans(text)...)
	for _, m := range translationKeepRE.FindAllStringIndex(text, -1) {
		if withinInlineCode(m[0], m[1], inlineCodes) {
			continue
		}
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
		result.InlineBoundaries = append(result.InlineBoundaries, span.inlineBoundaries)
		if span.kind == protectedInlineCodeOpen {
			result.InlinePairs = append(result.InlinePairs, protectedInlinePair{Open: token, Content: span.inlineContent, Boundaries: span.inlineBoundaries})
		}
		if span.kind == protectedInlineCodeClose {
			for pair := len(result.InlinePairs) - 1; pair >= 0; pair-- {
				if result.InlinePairs[pair].Close == "" {
					result.InlinePairs[pair].Close = token
					break
				}
			}
		}
		if isEmphasisTokenKind(span.kind) {
			result.EmphasisTokens = append(result.EmphasisTokens, token)
		}
		pos = span.end
	}
	out.WriteString(text[pos:])
	result.Text = out.String()
	return result
}

func withinInlineCode(start, end int, codes []presentInlineCode) bool {
	for _, code := range codes {
		if start >= code.Start && end <= code.End {
			return true
		}
	}
	return false
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
	if len(failures) == 0 {
		if err := p.validateInlinePairs(output); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) == 0 {
		if err := p.validateEmphasisTokenOrder(output); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) != 0 {
		return "", failures
	}
	restored := normalizeInlineTokenBoundaries(output, p)
	restored = normalizeEmphasisTokenBoundaries(restored, p)
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

func (p protectedTranslation) validateInlinePairs(output string) error {
	if len(p.InlinePairs) == 0 {
		return nil
	}
	pos := 0
	for i, pair := range p.InlinePairs {
		open := strings.Index(output[pos:], pair.Open)
		if open < 0 {
			return fmt.Errorf("inline code sentinel %d opening marker missing", i+1)
		}
		open += pos
		contentStart := open + len(pair.Open)
		closeOffset := strings.Index(output[contentStart:], pair.Close)
		if closeOffset < 0 {
			return fmt.Errorf("inline code sentinel %d closing marker missing", i+1)
		}
		close := contentStart + closeOffset
		if got := output[contentStart:close]; got != pair.Content {
			return fmt.Errorf("inline code sentinel %d content changed", i+1)
		}
		pos = close + len(pair.Close)
	}
	return nil
}

func (p protectedTranslation) validateEmphasisTokenOrder(output string) error {
	if len(p.EmphasisTokens) == 0 {
		return nil
	}
	var actual []string
	for _, token := range translationTokenRE.FindAllString(output, -1) {
		if isTokenIn(token, p.EmphasisTokens) {
			actual = append(actual, token)
		}
	}
	if len(actual) != len(p.EmphasisTokens) {
		return fmt.Errorf("emphasis sentinel count = %d, want %d", len(actual), len(p.EmphasisTokens))
	}
	for i := range actual {
		if actual[i] != p.EmphasisTokens[i] {
			return fmt.Errorf("emphasis sentinel order mismatch at %d", i+1)
		}
	}
	return nil
}

func isEmphasisTokenKind(kind protectedTokenKind) bool {
	return kind == protectedItalicOpen || kind == protectedItalicClose || kind == protectedBoldOpen || kind == protectedBoldClose
}

// presentEmphasisDelimiterSpans protects only the delimiters. The natural-language
// contents remain in the model input and can be translated normally.
func presentEmphasisDelimiterSpans(text string) []protectedSpan {
	var spans []protectedSpan
	for start := 0; start < len(text); {
		for start < len(text) {
			r, width := utf8.DecodeRuneInString(text[start:])
			if !unicode.IsSpace(r) {
				break
			}
			start += width
		}
		end := start
		for end < len(text) {
			r, width := utf8.DecodeRuneInString(text[end:])
			if unicode.IsSpace(r) {
				break
			}
			end += width
		}
		if start == end {
			break
		}
		word := text[start:end]
		if len(word) < 2 {
			start = end
			continue
		}
		first := strings.IndexAny(word, "_*`")
		if first < 0 || word[first] == '`' {
			start = end
			continue
		}
		if first != 0 {
			r, _ := utf8.DecodeLastRuneInString(word[:first])
			if !unicode.IsPunct(r) {
				start = end
				continue
			}
		}
		marker := word[first]
		styled := word[first:]
		last := first + strings.LastIndex(styled, styled[:1])
		if last == first {
			start = end
			continue
		}
		if last+1 != len(word) {
			r, _ := utf8.DecodeRuneInString(word[last+1:])
			if !unicode.IsPunct(r) {
				start = end
				continue
			}
		}
		openKind, closeKind := protectedItalicOpen, protectedItalicClose
		if marker == '*' {
			openKind, closeKind = protectedBoldOpen, protectedBoldClose
		}
		spans = append(spans,
			protectedSpan{start: start + first, end: start + first + 1, restore: string(marker), kind: openKind},
			protectedSpan{start: start + last, end: start + last + 1, restore: string(marker), kind: closeKind},
		)
		start = end
	}
	return spans
}

// normalizeInlineTokenBoundaries adds only the whitespace required for each
// protected inline-code pair to remain a distinct legacy present word. It runs
// after strict pair/content validation and before sentinel restoration.
func normalizeInlineTokenBoundaries(output string, p protectedTranslation) string {
	if len(p.InlinePairs) == 0 {
		return output
	}
	type pairPosition struct{ open, close int }
	positions := make([]pairPosition, len(p.InlinePairs))
	pos := 0
	for i, pair := range p.InlinePairs {
		open := strings.Index(output[pos:], pair.Open)
		if open < 0 {
			return output
		}
		open += pos
		closeOffset := strings.Index(output[open+len(pair.Open):], pair.Close)
		if closeOffset < 0 {
			return output
		}
		close := open + len(pair.Open) + closeOffset + len(pair.Close)
		positions[i] = pairPosition{open, close}
		pos = close
	}
	insert := map[int]bool{}
	for i, pair := range p.InlinePairs {
		if !pair.Boundaries {
			continue
		}
		at := positions[i]
		previousIsInline := i > 0 && positions[i-1].close == at.open && p.InlinePairs[i-1].Boundaries
		nextIsInline := i+1 < len(positions) && at.close == positions[i+1].open && p.InlinePairs[i+1].Boundaries
		if !previousIsInline && endsWithNonBoundary(output[:at.open]) {
			insert[at.open] = true
		}
		if !nextIsInline && startsWithNonBoundary(output[at.close:]) {
			insert[at.close] = true
		}
		if nextIsInline {
			insert[at.close] = true
		}
	}
	var normalized strings.Builder
	for i := 0; i <= len(output); i++ {
		if insert[i] {
			normalized.WriteByte(' ')
		}
		if i < len(output) {
			normalized.WriteByte(output[i])
		}
	}
	return normalized.String()
}

// normalizeEmphasisTokenBoundaries gives present's emphasis parser an explicit
// outer boundary without changing the translated span content. It deliberately
// runs before sentinel restoration, while each opening and closing role is known.
func normalizeEmphasisTokenBoundaries(output string, p protectedTranslation) string {
	if len(p.EmphasisTokens) == 0 {
		return output
	}
	found := translationTokenRE.FindAllString(output, -1)
	var actual []string
	for _, token := range found {
		if isTokenIn(token, p.EmphasisTokens) {
			actual = append(actual, token)
		}
	}
	if len(actual) != len(p.EmphasisTokens) {
		return output // token validation above will reject this.
	}
	for i := range actual {
		if actual[i] != p.EmphasisTokens[i] {
			return output // restore turns this into a fail-closed ordering failure below.
		}
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
			return output
		}
		segments[i] = output[pos : pos+offset]
		pos += offset + len(token)
	}
	segments[len(found)] = output[pos:]
	for i, token := range found {
		switch kinds[token] {
		case protectedItalicOpen, protectedBoldOpen:
			previousStructural := i > 0 && (kinds[found[i-1]] == protectedInlineCodeClose || kinds[found[i-1]] == protectedItalicClose || kinds[found[i-1]] == protectedBoldClose)
			if endsWithFontNonBoundary(segments[i]) || (segments[i] == "" && previousStructural) {
				segments[i] += " "
			}
		case protectedItalicClose, protectedBoldClose:
			nextStructural := i+1 < len(found) && (kinds[found[i+1]] == protectedInlineCodeOpen || kinds[found[i+1]] == protectedItalicOpen || kinds[found[i+1]] == protectedBoldOpen)
			if startsWithFontNonBoundary(segments[i+1]) || (segments[i+1] == "" && nextStructural) {
				segments[i+1] = " " + segments[i+1]
			}
		}
	}
	var normalized strings.Builder
	for i, token := range found {
		normalized.WriteString(segments[i])
		normalized.WriteString(token)
	}
	normalized.WriteString(segments[len(found)])
	return normalized.String()
}

func isTokenIn(token string, tokens []string) bool {
	for _, want := range tokens {
		if token == want {
			return true
		}
	}
	return false
}

func endsWithFontNonBoundary(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return !unicode.IsSpace(r) && !unicode.IsPunct(r)
}

func startsWithFontNonBoundary(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return !unicode.IsSpace(r) && !unicode.IsPunct(r)
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
	return !isLegacyInlineBoundary(r)
}

// present's legacy program-font parser accepts ASCII punctuation before a
// backtick span, but does not consistently recognize full-width punctuation.
// Insert structural whitespace before an inline token only when the preceding
// rune is neither whitespace nor such an ASCII boundary.
func isLegacyInlineBoundary(r rune) bool {
	return unicode.IsSpace(r) || (r <= unicode.MaxASCII && unicode.IsPunct(r))
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
