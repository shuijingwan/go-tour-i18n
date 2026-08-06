package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/tools/present"
)

type fontSpanKind string

const (
	fontItalic  fontSpanKind = "italic"
	fontBold    fontSpanKind = "bold"
	fontProgram fontSpanKind = "program"
)

func parsedFontSpans(root, article string, source []byte) ([]fontSpanKind, error) {
	doc, err := parsePresentPage(root, article, source)
	if err != nil {
		return nil, err
	}
	var spans []fontSpanKind
	for _, section := range doc.Sections {
		if err := collectSectionFontSpans(section, &spans); err != nil {
			return nil, err
		}
	}
	return spans, nil
}

func parsePresentPage(root, article string, source []byte) (*present.Doc, error) {
	present.PlayEnabled = true
	ctx := &present.Context{ReadFile: func(name string) ([]byte, error) {
		clean := filepath.Clean(filepath.FromSlash(name))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("unsafe referenced path %q", name)
		}
		return os.ReadFile(filepath.Join(root, "_content", "tour", clean))
	}}
	wrapped := append([]byte("Tour page\n\n"), source...)
	doc, err := ctx.Parse(bytes.NewReader(wrapped), article, 0)
	if err != nil {
		return nil, fmt.Errorf("present parse: %w", err)
	}
	if len(doc.Sections) != 1 {
		return nil, fmt.Errorf("top-level sections = %d, want 1", len(doc.Sections))
	}
	return doc, nil
}

func collectSectionFontSpans(section present.Section, spans *[]fontSpanKind) error {
	if err := collectStyledFontSpans(section.Title, spans); err != nil {
		return err
	}
	for _, elem := range section.Elem {
		switch value := elem.(type) {
		case present.Text:
			if value.Pre {
				continue
			}
			for _, line := range value.Lines {
				if err := collectStyledFontSpans(line, spans); err != nil {
					return err
				}
			}
		case present.List:
			for _, item := range value.Bullet {
				if err := collectStyledFontSpans(item, spans); err != nil {
					return err
				}
			}
		case present.Section:
			if err := collectSectionFontSpans(value, spans); err != nil {
				return err
			}
		}
	}
	return nil
}

func collectStyledFontSpans(text string, spans *[]fontSpanKind) error {
	styled := string(present.Style(text))
	tokenizer := html.NewTokenizer(strings.NewReader(styled))
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			if err := tokenizer.Err(); err != nil && !errors.Is(err, io.EOF) {
				return err
			}
			return nil
		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := tokenizer.TagName()
			switch string(name) {
			case "i":
				*spans = append(*spans, fontItalic)
			case "b":
				*spans = append(*spans, fontBold)
			case "code":
				*spans = append(*spans, fontProgram)
			}
		}
	}
}

func compareFontSpans(expected, actual []fontSpanKind) error {
	limit := len(expected)
	if len(actual) < limit {
		limit = len(actual)
	}
	for i := 0; i < limit; i++ {
		if expected[i] != actual[i] {
			return fmt.Errorf("font span mismatch at index %d: expected %s, actual %s", i+1, expected[i], actual[i])
		}
	}
	if len(expected) != len(actual) {
		return fmt.Errorf("font span count mismatch: expected %d, actual %d; first difference index %d", len(expected), len(actual), limit+1)
	}
	return nil
}
