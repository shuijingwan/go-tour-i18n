package sitecontent

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/languageview"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
)

type LanguageAuditSyntax string

const (
	AuditMarkdown  LanguageAuditSyntax = "markdown"
	AuditHTML      LanguageAuditSyntax = "html"
	AuditTemplate  LanguageAuditSyntax = "template-html"
	AuditShellYAML LanguageAuditSyntax = "registered-shell-yaml"
	AuditData      LanguageAuditSyntax = "formal-data"
)

// ProjectLanguageAudit is a readonly view, not a new extraction/reconstruction
// contract. Markdown uses the existing Goldmark grammar; HTML uses the same
// tokenizer/boundary vocabulary and inline protector as formal content. It also
// exposes social metadata which may be frozen outside editable language slots.
// Rendered Markdown locators are explicitly not source edit coordinates.
func ProjectLanguageAudit(owner string, raw []byte, syntax LanguageAuditSyntax) (languageaudit.Document, error) {
	if err := textBytes(raw); err != nil {
		return languageaudit.Document{}, err
	}
	d := languageaudit.Document{Owner: owner, Blocks: []languageaudit.Block{}}
	if syntax == AuditData {
		rows, err := LearnRenderData(raw)
		if err != nil {
			return d, err
		}
		visible := map[string]map[string]bool{}
		if err := parseLearnData(raw, func(a, z int, context string) {
			parts := strings.Split(context, "/")
			if len(parts) != 3 {
				return
			}
			if visible[parts[1]] == nil {
				visible[parts[1]] = map[string]bool{}
			}
			visible[parts[1]][parts[2]] = true
		}); err != nil {
			return d, err
		}
		for _, row := range rows {
			ownerID := digest([]byte(row["url"]))
			keys := []string{}
			for key := range visible[ownerID] {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				blocks, err := auditHTMLBlocks([]byte(row[key]), false)
				if err != nil {
					return d, err
				}
				for _, b := range blocks {
					b.Locator = "data/" + ownerID + "/" + key + "/" + b.Locator
					b.Start, b.End = -1, -1
					d.Blocks = append(d.Blocks, b)
				}
			}
		}
		return d, nil
	}
	if syntax == AuditShellYAML {
		s, err := ExtractShellLanguageSource(owner, raw)
		if err != nil {
			return d, err
		}
		for _, slot := range s.Slots {
			value, err := restoreBlock(slot.Unit, slot.Unit.Source)
			if err != nil {
				return d, err
			}
			blocks, err := auditHTMLBlocks([]byte(value), false)
			if err != nil {
				return d, err
			}
			for _, b := range blocks {
				b.Locator = slot.Unit.Locator + "/" + b.Locator
				b.Start, b.End = slot.Unit.Start, slot.Unit.End
				d.Blocks = append(d.Blocks, b)
			}
		}
		return d, nil
	}
	if syntax != AuditHTML && syntax != AuditTemplate && syntax != AuditMarkdown {
		return d, fmt.Errorf("unsupported audit syntax %q", syntax)
	}
	metadata := []span{}
	offset := 0
	if syntax != AuditTemplate {
		var err error
		offset, err = parseMetadata(raw, func(a, z int, c string) { metadata = append(metadata, span{a, z, c}) })
		if err != nil {
			return d, err
		}
	}
	for _, s := range metadata {
		pieces, err := auditTextPieces(raw[s.start:s.end], languageaudit.Prose, "")
		if err != nil {
			return d, err
		}
		d.Blocks = append(d.Blocks, languageaudit.Block{Locator: fmt.Sprintf("metadata/%s@%d", s.context, s.start), Context: "metadata", Start: s.start, End: s.end, Pieces: pieces})
	}
	body := raw[offset:]
	blocks, err := auditMarkupBody(body, syntax, syntax == AuditTemplate)
	if err != nil {
		return d, err
	}
	for _, b := range blocks {
		if syntax != AuditMarkdown {
			b.Start += offset
			b.End += offset
		}
		d.Blocks = append(d.Blocks, b)
	}
	return d, nil
}

// Shell owns its metadata grammar and language slots. Runtime metadata values
// are plain text, not HTML; only extractor-declared metadata enters this view.
func projectShellPageLanguageAudit(owner string, raw []byte) (languageaudit.Document, error) {
	d := languageaudit.Document{Owner: owner, Blocks: []languageaudit.Block{}}
	// Establish canonical metadata framing before invoking the slot extractor.
	body, metadata, err := ShellPageRenderInput(raw)
	if err != nil {
		return d, err
	}
	s, err := ExtractShellLanguageSource(owner, raw)
	if err != nil {
		return d, err
	}
	if s.Page == nil {
		return d, fmt.Errorf("formal Shell Page required for language audit")
	}
	metadataEnd := len(raw) - len(body)
	for _, slot := range s.Slots {
		if slot.Unit.End > metadataEnd || !strings.HasPrefix(slot.Role, "metadata/") {
			continue
		}
		value, ok := metadata[strings.TrimPrefix(slot.Role, "metadata/")].(string)
		if !ok {
			return d, fmt.Errorf("Shell language metadata must be text: %s", slot.Role)
		}
		view, err := ProjectLanguageAuditText(owner, slot.Role, "metadata", value, nil)
		if err != nil {
			return d, err
		}
		for _, b := range view.Blocks {
			b.Start, b.End = slot.Unit.Start, slot.Unit.End
			d.Blocks = append(d.Blocks, b)
		}
	}
	// Homepage rendering uses Markdown with inline HTML and template actions.
	// Actions are opaque boundaries, never executed as part of a readonly audit.
	blocks, err := auditMarkupBody(body, AuditMarkdown, true)
	if err != nil {
		return d, err
	}
	d.Blocks = append(d.Blocks, blocks...)
	return d, nil
}

func auditMarkupBody(body []byte, syntax LanguageAuditSyntax, templated bool) ([]languageaudit.Block, error) {
	if syntax == AuditMarkdown {
		if templated {
			var err error
			body, err = auditTemplateMask(body)
			if err != nil {
				return nil, err
			}
		}
		var rendered bytes.Buffer
		// Unsafe here means preserve source-owned inline HTML in a readonly buffer;
		// it is never executed, rendered in a browser, or written to repository data.
		md := goldmark.New(goldmark.WithParserOptions(parser.WithHeadingAttribute(), parser.WithAutoHeadingID()), goldmark.WithRendererOptions(gmhtml.WithUnsafe()), goldmark.WithExtensions(extension.Typographer, extension.Linkify, extension.DefinitionList, extension.Table))
		if err := md.Convert(body, &rendered); err != nil {
			return nil, err
		}
		body = rendered.Bytes()
	}
	blocks, err := auditHTMLBlocks(body, templated && syntax != AuditMarkdown)
	if err != nil {
		return nil, err
	}
	for i := range blocks {
		if syntax == AuditMarkdown {
			blocks[i].Locator = "markdown-rendered/" + blocks[i].Locator
			blocks[i].Start, blocks[i].End = -1, -1
		}
	}
	return blocks, nil
}

func auditHTMLBlocks(raw []byte, templated bool) ([]languageaudit.Block, error) {
	masked := bytes.Clone(raw)
	if templated {
		var err error
		masked, err = auditTemplateMask(raw)
		if err != nil {
			return nil, err
		}
	}
	// Retain the canonical HTML content grammar/unsupported-syntax boundary.
	if err := htmlBlocksMode(masked, 0, "html", func(int, int, string) {}, false); err != nil {
		return nil, err
	}
	return languageview.HTML(masked, languageview.Options{Text: auditTextPieces, Attribute: auditAttributePieces, Boundary: htmlBoundaryTag})
}

func auditTemplateMask(raw []byte) ([]byte, error) {
	ranges, err := shellTemplateRanges(raw)
	if err != nil {
		return nil, err
	}
	if err := validateShellTemplate(raw, ranges); err != nil {
		return nil, err
	}
	masked := bytes.Clone(raw)
	// Equal byte length retains tokenizer coordinates; an opaque control sentinel
	// is never a fabricated visible word or an inferred separator.
	for _, s := range ranges {
		for i := s.start; i < s.end; i++ {
			if masked[i] != '\n' {
				masked[i] = 0x1f
			}
		}
	}
	return shellAttributeMask(masked, raw)
}

func auditTextPieces(raw []byte, kind languageaudit.Kind, binding string) ([]languageaudit.Piece, error) {
	// Reuse opaque token/entity semantics, rather than recognizing raw markup
	// with a regex. Template sentinels explicitly break unknown composition.
	out := []languageaudit.Piece{}
	for i, part := range bytes.Split(raw, []byte{0x1f}) {
		if i > 0 && (len(out) == 0 || out[len(out)-1].Kind != languageaudit.Opaque) {
			out = append(out, languageaudit.Piece{Kind: languageaudit.Opaque})
		}
		if len(part) == 0 {
			continue
		}
		text, tokens, err := protectBlockMode(part, "html/text", false)
		if err != nil {
			return nil, err
		}
		pieces, err := protectionPieces(Unit{Source: text, Protected: tokens})
		if err != nil {
			return nil, err
		}
		for _, p := range pieces {
			value := p.text
			k := kind
			if p.token != nil {
				value = p.token.Raw
				if strings.HasPrefix(value, "{{") || strings.HasPrefix(value, "<!--") {
					out = append(out, languageaudit.Piece{Kind: languageaudit.Opaque})
					continue
				}
				decoded := html.UnescapeString(value)
				if strings.TrimSpace(value) == "" {
					// The native protector locks source line breaks for restoration.
					// In a visible audit view they remain whitespace, not unknown output.
				} else if strings.HasPrefix(value, "&") && decoded != value {
					value = decoded
					if kind == languageaudit.Prose && strings.TrimSpace(value) != "" {
						allSymbols := true
						for _, r := range value {
							if unicode.IsLetter(r) || unicode.IsDigit(r) {
								allSymbols = false
							}
						}
						if allSymbols {
							k = languageaudit.ProtectedSymbol
						}
					}
				} else {
					out = append(out, languageaudit.Piece{Kind: languageaudit.Opaque})
					continue
				}
			} else {
				value = html.UnescapeString(value)
			}
			out = append(out, languageaudit.Piece{Text: value, Kind: k, Binding: binding})
		}
	}
	return auditIdentityPieces(out), nil
}

// HTML attribute values are already entity-decoded by the tokenizer. They are
// plain visible strings, never recursively interpreted as markup.
func auditAttributePieces(value string) []languageaudit.Piece {
	out := []languageaudit.Piece{}
	for i, part := range strings.Split(value, "\x1f") {
		if i > 0 && (len(out) == 0 || out[len(out)-1].Kind != languageaudit.Opaque) {
			out = append(out, languageaudit.Piece{Kind: languageaudit.Opaque})
		}
		if part != "" {
			out = append(out, languageaudit.Piece{Text: part, Kind: languageaudit.Prose})
		}
	}
	return auditIdentityPieces(out)
}

func auditIdentityPieces(p []languageaudit.Piece) []languageaudit.Piece {
	return languageview.Identities(p)
}
