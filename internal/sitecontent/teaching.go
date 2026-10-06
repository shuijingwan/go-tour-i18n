package sitecontent

import (
	"bytes"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
	"io"
	"strings"
)

func teachingSpans(b []byte, offset int, markdown bool, add func(int, int, string)) error {
	expose := func(code []byte, a int) {
		for _, s := range i18n.SafeTeachingCommentSpans(string(code)) {
			add(offset+a+s.Start, offset+a+s.End, "teaching-comment")
		}
	}
	if markdown {
		tree := goldmark.New().Parser().Parse(gmtext.NewReader(b))
		if err := ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
			if !enter {
				return ast.WalkContinue, nil
			}
			if n.Kind() == ast.KindCodeBlock || n.Kind() == ast.KindFencedCodeBlock {
				if n.Lines().Len() > 0 {
					a := n.Lines().At(0).Start
					z := n.Lines().At(n.Lines().Len() - 1).Stop
					expose(b[a:z], a)
				}
				return ast.WalkSkipChildren, nil
			}
			return ast.WalkContinue, nil
		}); err != nil {
			return err
		}
	}
	// Only plain Go text in pre is eligible; HTML entities/markup are protected.
	z := html.NewTokenizer(bytes.NewReader(b))
	pos := 0
	depth := 0
	for {
		typ := z.Next()
		raw := z.Raw()
		a := pos
		pos += len(raw)
		if typ == html.ErrorToken {
			if z.Err() != io.EOF {
				return z.Err()
			}
			break
		}
		if typ == html.StartTagToken {
			tag, _ := z.TagName()
			if string(tag) == "pre" {
				depth++
			}
		}
		if typ == html.EndTagToken {
			tag, _ := z.TagName()
			if string(tag) == "pre" && depth > 0 {
				depth--
			}
		}
		if typ == html.TextToken && depth > 0 && !strings.ContainsAny(string(raw), "&<>") {
			expose(raw, a)
		}
	}
	return nil
}
