package i18n

import (
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"regexp"
	"strings"
)

type goExampleCommentKind string

const (
	goExampleCommentNatural         goExampleCommentKind = "natural"
	goExampleCommentNonNatural      goExampleCommentKind = "non-natural"
	goExampleCommentGoDirective     goExampleCommentKind = "go-directive"
	goExampleCommentLineDirective   goExampleCommentKind = "line-directive"
	goExampleCommentLegacyBuild     goExampleCommentKind = "legacy-build"
	goExampleCommentPresentMarker   goExampleCommentKind = "present-marker"
	goExampleCommentTestMarker      goExampleCommentKind = "test-marker"
	goExampleCommentGeneratedMarker goExampleCommentKind = "generated-marker"
)

type goExampleComment struct {
	Start, End   int
	PayloadStart int
	PayloadEnd   int
	Kind         goExampleCommentKind
}

var goExampleShiftVerbRE = regexp.MustCompile(`^Shift\s+it\s+(?:left|right)\b`)

var presentCommentMarkerRE = regexp.MustCompile(`^(?:(?:START|END)\s+[A-Z][A-Z0-9_]*|(?:OMIT|HL|HIGHLIGHT))$`)

func prepareTranslationUnitInput(unit *TranslationUnit, glossary *Glossary) (protectedTranslation, error) {
	if unit == nil {
		return protectedTranslation{}, fmt.Errorf("translation unit is required")
	}
	switch unit.Kind {
	case UnitKindPage:
		return prepareDefaultTranslationInput(unit.Source, unit.SourceSHA256, glossary), nil
	case UnitKindExample:
		return prepareGoExampleTranslationInput(unit.Source, unit.SourceSHA256, glossary)
	default:
		return protectedTranslation{}, fmt.Errorf("unsupported translation unit kind %q", unit.Kind)
	}
}

func prepareGoExampleTranslationInput(source []byte, hash string, glossary *Glossary) (protectedTranslation, error) {
	text := string(source)
	comments, err := scanGoExampleComments(source)
	if err != nil {
		return protectedTranslation{}, err
	}
	var translatable []goExampleComment
	for _, comment := range comments {
		if comment.Kind == goExampleCommentNatural {
			translatable = append(translatable, comment)
		}
	}
	if len(translatable) == 0 {
		result := protectedTranslationFromSpans(text, hash, []protectedSpan{{start: 0, end: len(text), kind: protectedOther}})
		result.RequireTokenOrder = true
		return result, nil
	}

	spans := make([]protectedSpan, 0, len(translatable)+1)
	position := 0
	for _, comment := range translatable {
		if position < comment.PayloadStart {
			spans = append(spans, protectedSpan{start: position, end: comment.PayloadStart, kind: protectedOther})
		}
		position = comment.PayloadEnd
	}
	if position < len(text) {
		spans = append(spans, protectedSpan{start: position, end: len(text), kind: protectedOther})
	}
	spans = append(spans, goExampleKeepProtectionSpans(text, glossary, func(start, end int) bool {
		for _, comment := range translatable {
			if start >= comment.PayloadStart && end <= comment.PayloadEnd {
				return true
			}
		}
		return false
	})...)
	result := protectedTranslationFromSpans(text, hash, spans)
	result.RequireTokenOrder = true
	return result, nil
}

// goExampleKeepProtectionSpans applies the shared Example keep semantics used
// by both input protection and candidate validation. Shift remains protected as
// a keyboard key, while the specific English verb construction "Shift it
// left/right" remains translatable.
func goExampleKeepProtectionSpans(text string, glossary *Glossary, allowed func(start, end int) bool) []protectedSpan {
	spans := translationKeepProtectionSpans(text, glossary, nil, allowed)
	filtered := spans[:0]
	for _, span := range spans {
		if text[span.start:span.end] == "Shift" && goExampleShiftVerbRE.MatchString(text[span.start:]) {
			continue
		}
		filtered = append(filtered, span)
	}
	return filtered
}

func scanGoExampleComments(source []byte) ([]goExampleComment, error) {
	if _, err := parser.ParseFile(token.NewFileSet(), "example.go", source, parser.ParseComments|parser.SkipObjectResolution); err != nil {
		return nil, fmt.Errorf("parse Go example: %w", err)
	}
	fset := token.NewFileSet()
	file := fset.AddFile("example.go", -1, len(source))
	var scanErr error
	var s scanner.Scanner
	s.Init(file, source, func(position token.Position, message string) {
		if scanErr == nil {
			scanErr = fmt.Errorf("scan Go example at %s: %s", position, message)
		}
	}, scanner.ScanComments)
	var comments []goExampleComment
	for {
		position, tok, literal := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok != token.COMMENT {
			continue
		}
		start := file.Offset(position)
		end := start + len(literal)
		payloadStart, payloadEnd, payload := goCommentPayload(source, start, end, literal)
		comments = append(comments, goExampleComment{
			Start: start, End: end, PayloadStart: payloadStart, PayloadEnd: payloadEnd,
			Kind: classifyGoExampleComment(literal, payload),
		})
	}
	if scanErr != nil {
		return nil, scanErr
	}
	return comments, nil
}

func goCommentPayload(source []byte, start, end int, literal string) (int, int, string) {
	payloadStart, payloadEnd := start+2, end
	if strings.HasPrefix(literal, "/*") {
		payloadEnd -= 2
	}
	for payloadStart < payloadEnd && isGoCommentEdgeSpace(source[payloadStart]) {
		payloadStart++
	}
	for payloadEnd > payloadStart && isGoCommentEdgeSpace(source[payloadEnd-1]) {
		payloadEnd--
	}
	return payloadStart, payloadEnd, string(source[payloadStart:payloadEnd])
}

func isGoCommentEdgeSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func classifyGoExampleComment(literal, payload string) goExampleCommentKind {
	trimmed := strings.TrimSpace(payload)
	line := strings.HasPrefix(literal, "//")
	if line && strings.HasPrefix(literal, "//go:") {
		return goExampleCommentGoDirective
	}
	if line && strings.HasPrefix(literal, "//line ") {
		return goExampleCommentLineDirective
	}
	if line && strings.HasPrefix(literal, "// +build ") {
		return goExampleCommentLegacyBuild
	}
	if line && presentCommentMarkerRE.MatchString(trimmed) {
		return goExampleCommentPresentMarker
	}
	if line && (strings.HasPrefix(trimmed, "Output:") || strings.HasPrefix(trimmed, "Unordered output:")) {
		return goExampleCommentTestMarker
	}
	if strings.HasPrefix(trimmed, "Code generated ") && strings.Contains(trimmed, " DO NOT EDIT.") {
		return goExampleCommentGeneratedMarker
	}
	if isNaturalGoExampleComment(trimmed) {
		return goExampleCommentNatural
	}
	return goExampleCommentNonNatural
}

func isNaturalGoExampleComment(text string) bool {
	words := englishWords(strings.TrimSpace(text))
	if len(words) >= 3 {
		return true
	}
	if len(words) != 2 {
		return false
	}
	shortNaturalMarkers := map[string]bool{
		"add": true, "change": true, "create": true, "drop": true,
		"extend": true, "returns": true, "use": true, "works": true,
	}
	return shortNaturalMarkers[strings.ToLower(words[0])]
}

func hasTranslatableGoExampleComment(source []byte) (bool, error) {
	comments, err := scanGoExampleComments(source)
	if err != nil {
		return false, err
	}
	for _, comment := range comments {
		if comment.Kind == goExampleCommentNatural {
			return true, nil
		}
	}
	return false, nil
}
