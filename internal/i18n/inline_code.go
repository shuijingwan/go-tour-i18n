package i18n

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// presentInlineCode describes a program-font span as interpreted by
// golang.org/x/tools/present. A single backtick inside the outer backticks
// represents a space; doubled backticks represent a literal backtick.
type presentInlineCode struct {
	Start, End int
	Raw        string
	Content    string
}

func presentInlineCodes(text string) []presentInlineCode {
	var result []presentInlineCode
	links := linkRE.FindAllStringIndex(text, -1)
	linkIndex := 0

	for start := 0; start < len(text); {
		_, width := utf8.DecodeRuneInString(text[start:])
		if width == 0 {
			break
		}
		if r, _ := utf8.DecodeRuneInString(text[start:]); unicode.IsSpace(r) {
			start += width
			continue
		}
		end := start + width
		for end < len(text) {
			r, size := utf8.DecodeRuneInString(text[end:])
			if unicode.IsSpace(r) {
				break
			}
			end += size
		}

		word := text[start:end]
		first := strings.IndexByte(word, '`')
		if first >= 0 && (first == 0 || precedingRuneIsPunctuation(word, first)) {
			last := strings.LastIndexByte(word, '`')
			if last > first && (last+1 == len(word) || followingRuneIsPunctuation(word, last+1)) {
				spanStart, spanEnd := start+first, start+last+1
				for linkIndex < len(links) && links[linkIndex][1] <= spanStart {
					linkIndex++
				}
				if linkIndex >= len(links) || links[linkIndex][0] >= spanEnd {
					raw := text[spanStart:spanEnd]
					result = append(result, presentInlineCode{
						Start:   spanStart,
						End:     spanEnd,
						Raw:     raw,
						Content: decodePresentProgramFont(raw[1 : len(raw)-1]),
					})
				}
			}
		}
		start = end
	}
	return result
}

func precedingRuneIsPunctuation(s string, offset int) bool {
	r, _ := utf8.DecodeLastRuneInString(s[:offset])
	return unicode.IsPunct(r)
}

func followingRuneIsPunctuation(s string, offset int) bool {
	r, _ := utf8.DecodeRuneInString(s[offset:])
	return unicode.IsPunct(r)
}

func decodePresentProgramFont(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '`' {
			r, width := utf8.DecodeRuneInString(s[i:])
			out.WriteRune(r)
			i += width
			continue
		}
		if i+1 < len(s) && s[i+1] == '`' {
			out.WriteByte('`')
			i += 2
			continue
		}
		out.WriteByte(' ')
		i++
	}
	return out.String()
}
