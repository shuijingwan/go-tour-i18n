package i18n

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"sort"
	"strings"
)

type goExampleLexicalEvent struct {
	token   token.Token
	literal string
}

// ValidateTranslationUnitCandidate dispatches candidate validation by the
// translation unit kind without changing the existing page validation policy.
func ValidateTranslationUnitCandidate(root string, catalog *Catalog, unitID, locale string, candidate []byte) error {
	if catalog == nil {
		return fmt.Errorf("translation catalog is required")
	}
	unit, err := catalog.Unit(unitID)
	if err != nil {
		return err
	}
	switch unit.Kind {
	case UnitKindPage:
		return ValidateCandidateForLocale(root, catalog, unit.ID, locale, candidate)
	case UnitKindExample:
		return ValidateGoExampleCandidate(root, unit, locale, candidate)
	default:
		return fmt.Errorf("翻译单元 %s 的类型 %q 不受支持", unit.ID, unit.Kind)
	}
}

// ValidateGoExampleCandidate permits changes only inside ordinary natural
// language comment payloads. Every other source byte and lexical event remains
// independently checked against the complete upstream Go source.
func ValidateGoExampleCandidate(root string, unit *TranslationUnit, locale string, candidate []byte) error {
	if unit == nil || unit.Kind != UnitKindExample {
		return fmt.Errorf("Go 示例校验器需要 example 翻译单元")
	}
	if sum(unit.Source) != unit.SourceSHA256 {
		return fmt.Errorf("示例翻译单元 %s 的完整源版本哈希不匹配", unit.ID)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), unit.SourcePath, unit.Source, parser.ParseComments|parser.SkipObjectResolution); err != nil {
		return fmt.Errorf("示例翻译单元 %s 的源 Go 文件无法解析: %w", unit.ID, err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), unit.SourcePath, candidate, parser.ParseComments|parser.SkipObjectResolution); err != nil {
		return fmt.Errorf("示例翻译单元 %s 的候选 Go 文件无法解析: %w", unit.ID, err)
	}
	if _, err := format.Source(candidate); err != nil {
		return fmt.Errorf("示例翻译单元 %s 的候选无法通过 gofmt 健全性检查: %w", unit.ID, err)
	}

	sourceEvents, err := scanGoExampleLexicalEvents(unit.Source)
	if err != nil {
		return fmt.Errorf("示例翻译单元 %s 的源 lexical event 扫描失败: %w", unit.ID, err)
	}
	candidateEvents, err := scanGoExampleLexicalEvents(candidate)
	if err != nil {
		return fmt.Errorf("示例翻译单元 %s 的候选 lexical event 扫描失败: %w", unit.ID, err)
	}
	if len(sourceEvents) != len(candidateEvents) {
		return fmt.Errorf("示例翻译单元 %s 的 Go token/comment event 数量发生变化: source=%d candidate=%d", unit.ID, len(sourceEvents), len(candidateEvents))
	}
	for i := range sourceEvents {
		sourceEvent, candidateEvent := sourceEvents[i], candidateEvents[i]
		if sourceEvent.token != candidateEvent.token {
			return fmt.Errorf("示例翻译单元 %s 的 lexical event %d 类型发生变化: source=%s candidate=%s", unit.ID, i+1, sourceEvent.token, candidateEvent.token)
		}
		if sourceEvent.token != token.COMMENT {
			if sourceEvent.literal != candidateEvent.literal {
				return fmt.Errorf("示例翻译单元 %s 的非注释 Go token %d 发生变化: source=%q candidate=%q", unit.ID, i+1, sourceEvent.literal, candidateEvent.literal)
			}
			continue
		}
		if goCommentStyle(sourceEvent.literal) != goCommentStyle(candidateEvent.literal) {
			return fmt.Errorf("示例翻译单元 %s 的注释 %d 类型发生变化", unit.ID, i+1)
		}
		_, _, payload := goCommentPayload([]byte(sourceEvent.literal), 0, len(sourceEvent.literal), sourceEvent.literal)
		sourceKind := classifyGoExampleComment(sourceEvent.literal, payload)
		_, _, candidatePayload := goCommentPayload([]byte(candidateEvent.literal), 0, len(candidateEvent.literal), candidateEvent.literal)
		candidateKind := classifyGoExampleComment(candidateEvent.literal, candidatePayload)
		if sourceKind != goExampleCommentNatural && sourceEvent.literal != candidateEvent.literal {
			return fmt.Errorf("示例翻译单元 %s 的特殊或不可翻译注释 %d 发生变化", unit.ID, i+1)
		}
		if sourceKind == goExampleCommentNatural && candidateKind != goExampleCommentNatural && candidateKind != goExampleCommentNonNatural {
			return fmt.Errorf("示例翻译单元 %s 的普通注释 %d 被改成特殊注释", unit.ID, i+1)
		}
	}

	sourceComments, err := scanGoExampleComments(unit.Source)
	if err != nil {
		return err
	}
	candidateComments, err := scanGoExampleComments(candidate)
	if err != nil {
		return err
	}
	if len(sourceComments) != len(candidateComments) {
		return fmt.Errorf("示例翻译单元 %s 的注释数量发生变化: source=%d candidate=%d", unit.ID, len(sourceComments), len(candidateComments))
	}
	if err := compareGoExampleImmutableBytes(unit.ID, unit.Source, candidate, sourceComments, candidateComments); err != nil {
		return err
	}
	glossary, err := LoadGlossary(root, locale)
	if err != nil {
		return err
	}
	return validateGoExampleGlossary(unit.ID, unit.Source, candidate, sourceComments, candidateComments, glossary)
}

func scanGoExampleLexicalEvents(source []byte) ([]goExampleLexicalEvent, error) {
	fset := token.NewFileSet()
	file := fset.AddFile("example.go", -1, len(source))
	var scanErr error
	var s scanner.Scanner
	s.Init(file, source, func(position token.Position, message string) {
		if scanErr == nil {
			scanErr = fmt.Errorf("%s: %s", position, message)
		}
	}, scanner.ScanComments)
	var events []goExampleLexicalEvent
	for {
		_, tok, literal := s.Scan()
		if tok == token.EOF {
			break
		}
		events = append(events, goExampleLexicalEvent{token: tok, literal: literal})
	}
	return events, scanErr
}

func goCommentStyle(literal string) string {
	if strings.HasPrefix(literal, "//") {
		return "line"
	}
	return "block"
}

func compareGoExampleImmutableBytes(unitID string, source, candidate []byte, sourceComments, candidateComments []goExampleComment) error {
	sourceCursor, candidateCursor := 0, 0
	for i := range sourceComments {
		sourceComment, candidateComment := sourceComments[i], candidateComments[i]
		if sourceComment.Kind != goExampleCommentNatural {
			continue
		}
		if !bytes.Equal(source[sourceCursor:sourceComment.PayloadStart], candidate[candidateCursor:candidateComment.PayloadStart]) {
			return fmt.Errorf("示例翻译单元 %s 的普通注释 %d 位置或不可翻译源码发生变化", unitID, i+1)
		}
		sourceCursor = sourceComment.PayloadEnd
		candidateCursor = candidateComment.PayloadEnd
	}
	if !bytes.Equal(source[sourceCursor:], candidate[candidateCursor:]) {
		return fmt.Errorf("示例翻译单元 %s 的普通注释位置或不可翻译源码发生变化", unitID)
	}
	return nil
}

func validateGoExampleGlossary(unitID string, source, candidate []byte, sourceComments, candidateComments []goExampleComment, glossary *Glossary) error {
	for i, sourceComment := range sourceComments {
		if sourceComment.Kind != goExampleCommentNatural {
			continue
		}
		sourcePayload := string(source[sourceComment.PayloadStart:sourceComment.PayloadEnd])
		candidateComment := candidateComments[i]
		candidatePayload := string(candidate[candidateComment.PayloadStart:candidateComment.PayloadEnd])
		sourceKeep := goExampleKeepItems(sourcePayload, glossary)
		candidateKeep := goExampleKeepItems(candidatePayload, glossary)
		if !equalStrings(sourceKeep, candidateKeep) {
			return fmt.Errorf("示例翻译单元 %s 的普通注释 %d glossary.keep 发生变化: source=%v candidate=%v", unitID, i+1, sourceKeep, candidateKeep)
		}
		for _, forbidden := range glossary.Forbidden {
			if strings.Contains(candidatePayload, forbidden) {
				return fmt.Errorf("示例翻译单元 %s 的普通注释 %d 包含禁止译法 %q", unitID, i+1, forbidden)
			}
		}
	}
	return nil
}

func goExampleKeepItems(payload string, glossary *Glossary) []string {
	spans := goExampleKeepProtectionSpans(payload, glossary, nil)
	items := make([]string, 0, len(spans))
	for _, span := range spans {
		items = append(items, payload[span.start:span.end])
	}
	sort.Strings(items)
	return items
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
