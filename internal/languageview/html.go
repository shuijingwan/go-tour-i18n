// Package languageview shares parser-owned HTML visibility across adapters.
package languageview

import (
	"bytes"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"golang.org/x/net/html"
	"io"
	"strings"
	"unicode"
)

type element struct {
	tag, link    string
	hidden, code bool
}
type Options struct {
	Text      func([]byte, languageaudit.Kind, string) ([]languageaudit.Piece, error)
	Attribute func(string) []languageaudit.Piece
	Boundary  func(string) bool
}

func Boundary(tag string) bool {
	return strings.Contains(" p li h1 h2 h3 h4 h5 h6 td th dt dd figcaption title div section article main ul ol table tr thead tbody blockquote dl header footer nav aside pre script style hr ", " "+tag+" ")
}
func HTML(raw []byte, o Options) ([]languageaudit.Block, error) {
	masked := raw
	textPieces := o.Text
	if textPieces == nil {
		textPieces = Text
	}
	attributePieces := o.Attribute
	if attributePieces == nil {
		attributePieces = Plain
	}
	boundary := o.Boundary
	if boundary == nil {
		boundary = Boundary
	}
	blocks := []languageaudit.Block{}
	stack := []element{}
	current := languageaudit.Block{Context: "paragraph", Start: 0}
	sequence := 0
	flush := func(end int) {
		if len(current.Pieces) > 0 {
			visible := false
			for _, x := range current.Pieces {
				if x.Kind != languageaudit.Opaque && strings.TrimSpace(x.Text) != "" {
					visible = true
				}
			}
			if visible {
				sequence++
				current.Locator = fmt.Sprintf("html/%s/%d", current.Context, sequence)
				current.End = end
				blocks = append(blocks, current)
			}
		}
		current = languageaudit.Block{Context: "paragraph", Start: end}
	}
	appendPieces := func(ps []languageaudit.Piece) {
		for _, p := range ps {
			n := len(current.Pieces)
			if n > 0 && p.Kind != languageaudit.Opaque && current.Pieces[n-1].Kind == p.Kind && current.Pieces[n-1].Binding == p.Binding {
				current.Pieces[n-1].Text += p.Text
			} else {
				current.Pieces = append(current.Pieces, p)
			}
		}
	}
	z := html.NewTokenizer(bytes.NewReader(masked))
	pos := 0
	for {
		typ := z.Next()
		a := pos
		pos += len(z.Raw())
		if typ == html.ErrorToken {
			if z.Err() != io.EOF {
				return nil, z.Err()
			}
			flush(pos)
			break
		}
		hidden, code := false, false
		link := ""
		for _, e := range stack {
			hidden = hidden || e.hidden
			code = code || e.code
			if e.link != "" {
				link = e.link
			}
		}
		switch typ {
		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()
			tag := t.Data
			if boundary(tag) {
				flush(a)
				current.Start = pos
			}
			if len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6' {
				current.Context = "heading"
			}
			if tag == "title" {
				current.Context = "metadata"
			}
			attrs := map[string]string{}
			for _, x := range t.Attr {
				attrs[x.Key] = x.Val
			}
			ownHidden := hidden || attrs["aria-hidden"] == "true" || strings.Contains(attrs["class"], "material-icons") || strings.Contains(" pre script style textarea svg ", " "+tag+" ")
			// Initially hidden panels/dialogs can become visible through interaction.
			if !ownHidden {
				for _, a := range t.Attr {
					if a.Key != "alt" && a.Key != "title" && a.Key != "aria-label" {
						continue
					}
					blocks = append(blocks, languageaudit.Block{Locator: fmt.Sprintf("html/attribute/%s/%s@%d", tag, a.Key, pos-len(z.Raw())), Context: "attribute", Start: pos - len(z.Raw()), End: pos, Pieces: attributePieces(a.Val)})
				}
			}
			if tag == "meta" && !ownHidden {
				name := attrs["property"]
				if name == "" {
					name = attrs["name"]
				}
				switch name {
				case "description", "og:title", "og:description", "twitter:title", "twitter:description":
					ps := attributePieces(attrs["content"])
					blocks = append(blocks, languageaudit.Block{Locator: fmt.Sprintf("html/meta/%s@%d", name, a), Context: "metadata", Start: a, End: pos, Pieces: ps})
				}
			}
			if tag == "br" && !hidden {
				appendPieces([]languageaudit.Piece{{Text: "\n", Kind: languageaudit.Prose}})
			}
			if tag == "wbr" && !hidden {
				appendPieces([]languageaudit.Piece{{Kind: languageaudit.Opaque}})
			}
			if typ != html.SelfClosingTagToken && !strings.Contains(" area base br col embed hr img input link meta param source track wbr ", " "+tag+" ") {
				e := element{tag: tag, hidden: ownHidden, code: strings.Contains(" code var kbd samp tt ", " "+tag+" ")}
				if tag == "a" {
					e.link = attrs["href"]
				}
				stack = append(stack, e)
			}
		case html.EndTagToken:
			tag := z.Token().Data
			if boundary(tag) {
				flush(a)
			}
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].tag == tag {
					stack = stack[:i]
					break
				}
			}
			if boundary(tag) {
				for i := len(stack) - 1; i >= 0; i-- {
					t := stack[i].tag
					if len(t) == 2 && t[0] == 'h' && t[1] >= '1' && t[1] <= '6' {
						current.Context = "heading"
						break
					}
					if t == "title" {
						current.Context = "metadata"
						break
					}
				}
			}
		case html.TextToken:
			if hidden {
				continue
			}
			kind := languageaudit.Prose
			if link != "" {
				kind = languageaudit.Link
			}
			if code {
				kind = languageaudit.Code
			}
			ps, err := textPieces(z.Raw(), kind, link)
			if err != nil {
				return nil, err
			}
			appendPieces(ps)
		}
	}
	return blocks, nil
}
func Plain(s string) []languageaudit.Piece {
	return Identities([]languageaudit.Piece{{Text: s, Kind: languageaudit.Prose}})
}

// Tokenizer-owned text is decoded exactly once. Entity symbols stay visible;
// source-owned template unknowns, when supplied by an adapter, remain opaque.
func Text(raw []byte, k languageaudit.Kind, binding string) ([]languageaudit.Piece, error) {
	out := []languageaudit.Piece{}
	for i, part := range strings.Split(string(raw), "\x1f") {
		if i > 0 {
			out = append(out, languageaudit.Piece{Kind: languageaudit.Opaque})
		}
		for len(part) > 0 {
			at := strings.IndexByte(part, '&')
			if at < 0 {
				out = append(out, languageaudit.Piece{Text: part, Kind: k, Binding: binding})
				break
			}
			if at > 0 {
				out = append(out, languageaudit.Piece{Text: part[:at], Kind: k, Binding: binding})
				part = part[at:]
			}
			end := strings.IndexByte(part, ';')
			if end < 0 || end > 32 {
				out = append(out, languageaudit.Piece{Text: part[:1], Kind: k, Binding: binding})
				part = part[1:]
				continue
			}
			token := part[:end+1]
			v := html.UnescapeString(token)
			kind := k
			if v != token && k == languageaudit.Prose && strings.TrimSpace(v) != "" {
				symbol := true
				for _, r := range v {
					if unicode.IsLetter(r) || unicode.IsDigit(r) {
						symbol = false
					}
				}
				if symbol {
					kind = languageaudit.ProtectedSymbol
				}
			}
			out = append(out, languageaudit.Piece{Text: v, Kind: kind, Binding: binding})
			part = part[end+1:]
		}
	}
	return Identities(out), nil
}
