package sitecontent

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
)

// PageCommentBlocks uses the same Goldmark and HTML coordinate authorities as
// teachingSpans. Internal blocks remain diagnostic slots of one physical Page.
func PageCommentBlocks(path string, raw []byte) ([]i18n.CommentCodeBlock, error) {
	offset, err := parseMetadata(raw, func(int, int, string) {})
	if err != nil {
		return nil, err
	}
	return pageCommentBlocks(path, raw, offset)
}

func (a PageAdapter) CommentSourceBlocks() ([]i18n.CommentCodeBlock, string, error) {
	blocks, err := PageCommentBlocks(a.Document.Path, a.Source)
	if err != nil {
		return nil, "", err
	}
	_, title, _, err := PageRenderInput(a.Source)
	return blocks, title, err
}

func (a PageAdapter) CommentTargetBlocks(candidate []byte) ([]i18n.CommentCodeBlock, error) {
	return PageCommentBlocks(a.Document.Path, candidate)
}

func (a ShellPageAdapter) CommentSourceBlocks() ([]i18n.CommentCodeBlock, string, error) {
	if a.Source.Page == nil {
		return nil, "", fmt.Errorf("formal shell Page required for comment authority")
	}
	return shellPageCommentBlocks(a.Source.Path, a.Raw)
}

func (a ShellPageAdapter) CommentTargetBlocks(candidate []byte) ([]i18n.CommentCodeBlock, error) {
	if a.Source.Page == nil {
		return nil, fmt.Errorf("formal shell Page required for comment authority")
	}
	blocks, _, err := shellPageCommentBlocks(a.Source.Path, candidate)
	return blocks, err
}

// Shell metadata is validated by its canonical extractor, never the narrower
// Learn/Docs Page schema. Runtime parsing supplies coordinates/title only after
// that validation; it cannot authorize unknown or malformed metadata.
func shellPageCommentBlocks(path string, raw []byte) ([]i18n.CommentCodeBlock, string, error) {
	if !bytes.HasPrefix(raw, []byte("---\n")) {
		return nil, "", fmt.Errorf("Homepage metadata required")
	}
	s, err := ExtractShellLanguageSource(path, raw)
	if err != nil {
		return nil, "", err
	}
	if s.Page == nil {
		return nil, "", fmt.Errorf("formal shell Page required for comment authority")
	}
	body, metadata, err := ShellPageRenderInput(raw)
	if err != nil {
		return nil, "", err
	}
	offset := len(raw) - len(body)
	blocks, err := pageCommentBlocks(path, raw, offset)
	if err != nil {
		return nil, "", err
	}
	// Shell display/output pre elements have no code authority without an
	// explicit language binding. Keep declared blocks (including unsupported
	// declarations) subject to the existing language/coverage validator.
	selected := blocks[:0]
	for _, block := range blocks {
		if block.EnclosingKind == "html-pre" && block.DeclaredLanguage == "" {
			continue
		}
		selected = append(selected, block)
	}
	blocks = selected
	// Preserve existing Markdown/pre coordinates. Homepage's editor is a
	// separate, Shell-owned Go surface, not script/style machine content.
	z := html.NewTokenizer(bytes.NewReader(body))
	pos, start, editors := offset, -1, 0
	var code strings.Builder
	for {
		t := z.Next()
		a := pos
		pos += len(z.Raw())
		if t == html.ErrorToken {
			if z.Err() != io.EOF {
				return nil, "", z.Err()
			}
			break
		}
		if t == html.StartTagToken {
			tag, _ := z.TagName()
			if string(tag) == "textarea" {
				start = a
				code.Reset()
			}
		} else if t == html.TextToken && start >= 0 {
			code.Write(z.Text())
		} else if t == html.EndTagToken && start >= 0 {
			tag, _ := z.TagName()
			if string(tag) == "textarea" {
				editors++
				block := i18n.AnalyzeCommentCode(code.String(), "go", fmt.Sprintf("html-textarea/%d", editors), false)
				block.EnclosingKind, block.Start, block.End = "html-textarea", start, pos
				blocks = append(blocks, block)
				start = -1
			}
		}
	}
	if start >= 0 {
		return nil, "", fmt.Errorf("unclosed shell textarea")
	}
	title, _ := metadata["title"].(string)
	return blocks, title, nil
}

// SourceHTMLCommentBlocks inventories frozen upstream HTML before formal Page
// adoption. HTML metadata comments stay opaque to the tokenizer; this does not
// relax the metadata schema used by PageCommentBlocks or create a Page Unit.
func SourceHTMLCommentBlocks(raw []byte) ([]i18n.CommentCodeBlock, error) {
	return pageCommentBlocks("source.html", raw, 0)
}

func pageCommentBlocks(path string, raw []byte, offset int) ([]i18n.CommentCodeBlock, error) {
	b := raw[offset:]
	out := []i18n.CommentCodeBlock{}
	covered := [][2]int{}
	if filepath.Ext(path) == ".md" {
		tree := goldmark.New().Parser().Parse(gmtext.NewReader(b))
		err := ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
			if !enter || (n.Kind() != ast.KindCodeBlock && n.Kind() != ast.KindFencedCodeBlock) {
				return ast.WalkContinue, nil
			}
			lang := ""
			if f, ok := n.(*ast.FencedCodeBlock); ok {
				lang = string(f.Language(b))
			}
			var code strings.Builder
			for i := 0; i < n.Lines().Len(); i++ {
				s := n.Lines().At(i)
				code.Write(s.Value(b))
			}
			if n.Lines().Len() > 0 {
				covered = append(covered, [2]int{n.Lines().At(0).Start, n.Lines().At(n.Lines().Len() - 1).Stop})
			}
			block := i18n.AnalyzeCommentCode(code.String(), lang, fmt.Sprintf("markdown/%d", len(out)+1), false)
			if prev := n.PreviousSibling(); prev != nil && (prev.Kind() == ast.KindParagraph || prev.Kind() == ast.KindTextBlock) {
				var text strings.Builder
				_ = ast.Walk(prev, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
					if enter {
						if node, ok := n.(*ast.Text); ok {
							text.Write(node.Segment.Value(b))
							if node.SoftLineBreak() || node.HardLineBreak() {
								text.WriteByte(' ')
							}
						}
					}
					return ast.WalkContinue, nil
				})
				block.SourceContext = text.String()
			}
			block.EnclosingKind = "markdown-indented-code"
			if n.Kind() == ast.KindFencedCodeBlock {
				block.EnclosingKind = "markdown-fenced-code"
			}
			if n.Lines().Len() > 0 {
				block.Start = offset + n.Lines().At(0).Start
				block.End = offset + n.Lines().At(n.Lines().Len()-1).Stop
			}
			out = append(out, block)
			return ast.WalkSkipChildren, nil
		})
		if err != nil {
			return nil, err
		}
	}
	// HTML code is tokenized, so attributes/escaped delimiters cannot masquerade
	// as comments. Nested markup is decoded to the actual code text.
	z := html.NewTokenizer(bytes.NewReader(b))
	pos, depth, start := 0, 0, 0
	var code strings.Builder
	lang := ""
	var context strings.Builder
	blockContext := ""
	variableMarkup := false
	for {
		t := z.Next()
		a := pos
		pos += len(z.Raw())
		if t == html.ErrorToken {
			if z.Err() != io.EOF {
				return nil, z.Err()
			}
			break
		}
		skip := false
		for _, c := range covered {
			if a >= c[0] && a < c[1] {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if t == html.StartTagToken {
			tok := z.Token()
			if depth == 0 && (tok.Data == "p" || tok.Data == "li") {
				context.Reset()
			}
			if tok.Data == "pre" {
				if depth == 0 {
					code.Reset()
					start = a
					lang = ""
					blockContext = context.String()
					variableMarkup = false
				}
				depth++
			}
			if depth > 0 {
				if tok.Data == "var" {
					variableMarkup = true
				}
				for _, attr := range tok.Attr {
					if attr.Key == "class" {
						for _, c := range strings.Fields(attr.Val) {
							if strings.HasPrefix(c, "language-") {
								lang = strings.TrimPrefix(c, "language-")
							}
						}
					}
				}
			}
		} else if t == html.EndTagToken {
			tag, _ := z.TagName()
			if string(tag) == "pre" && depth > 0 {
				depth--
				if depth == 0 {
					block := i18n.AnalyzeCommentCode(code.String(), lang, fmt.Sprintf("html-pre/%d", len(out)+1), false)
					block.EnclosingKind, block.Start, block.End = "html-pre", offset+start, offset+pos
					block.SourceContext, block.VariableMarkup = blockContext, variableMarkup
					out = append(out, block)
					context.Reset()
				}
			}
		} else if t == html.TextToken && depth > 0 {
			code.Write(z.Text())
		} else if t == html.TextToken {
			context.Write(z.Text())
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unclosed pre block")
	}
	return out, nil
}
