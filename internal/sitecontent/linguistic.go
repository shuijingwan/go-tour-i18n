package sitecontent

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
)

// A model translates one complete linguistic block with opaque locked tokens.
// Paired tokens bind a translated label to its original link/tag destination.
// Atomic code/literal tokens may move with the target language word order.
type ProtectedToken struct {
	Token string `json:"token"`
	Raw   string `json:"raw"`
	Role  string `json:"role"`
	Pair  string `json:"pair,omitempty"`
}

func unitSourceIdentity(source string, locked []ProtectedToken) string {
	return identity(struct {
		Text      string
		Protected []ProtectedToken
	}{source, locked})
}
func protectionIdentity(locked []ProtectedToken) string {
	// Order of separate atomic tokens and complete bindings may change. Binding
	// identity includes both delimiters, never just an unbound URL multiset.
	values := []string{}
	pairs := map[string]string{}
	for _, p := range locked {
		if p.Role == "open" {
			pairs[p.Pair] = p.Raw
		}
	}
	for _, p := range locked {
		switch p.Role {
		case "close":
			values = append(values, "pair\x00"+pairs[p.Pair]+"\x00"+p.Raw)
		case "atom", "suffix":
			values = append(values, p.Role+"\x00"+p.Raw)
		}
	}
	sort.Strings(values)
	return identity(values)
}
func hasLanguage(s string) bool {
	for {
		a := strings.Index(s, "⟦P")
		if a < 0 {
			break
		}
		z := strings.Index(s[a:], "⟧")
		if z < 0 {
			return false
		}
		s = s[:a] + s[a+z+len("⟧"):]
	}
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func restoreBlock(u Unit, target string) (string, error) {
	remaining := target
	positions := map[string]int{}
	for _, p := range u.Protected {
		if strings.Count(target, p.Token) != 1 {
			return "", fmt.Errorf("locked token exact-set changed: %s", p.Token)
		}
		positions[p.Token] = strings.Index(target, p.Token)
		remaining = strings.ReplaceAll(remaining, p.Token, "")
	}
	if strings.ContainsAny(remaining, "⟦⟧") {
		return "", fmt.Errorf("unknown protected token")
	}
	order := append([]ProtectedToken{}, u.Protected...)
	sort.Slice(order, func(i, j int) bool { return positions[order[i].Token] < positions[order[j].Token] })
	stack := []string{}
	for _, p := range order {
		switch p.Role {
		case "open":
			stack = append(stack, p.Pair)
		case "close":
			if len(stack) == 0 || stack[len(stack)-1] != p.Pair {
				return "", fmt.Errorf("protected label/tag binding changed")
			}
			stack = stack[:len(stack)-1]
		case "suffix":
			if strings.TrimSpace(target[positions[p.Token]+len(p.Token):]) != "" {
				return "", fmt.Errorf("heading anchor must remain suffix")
			}
		case "boundary":
			last := -1
			for _, q := range u.Protected {
				if q.Role == "boundary" {
					if positions[q.Token] <= last {
						return "", fmt.Errorf("Page boundary order changed")
					}
					last = positions[q.Token]
				}
			}
		case "atom":
		default:
			return "", fmt.Errorf("unknown protected token role")
		}
	}
	if len(stack) > 0 {
		return "", fmt.Errorf("unclosed protected binding")
	}
	// Replace once in one pass; raw bytes containing token-like text cannot be
	// recursively interpreted as another replacement.
	var out strings.Builder
	pos := 0
	for _, p := range order {
		a := positions[p.Token]
		out.WriteString(target[pos:a])
		out.WriteString(p.Raw)
		pos = a + len(p.Token)
	}
	out.WriteString(target[pos:])
	return out.String(), nil
}

func markdownBlocks(b []byte, offset int, add func(int, int, string)) error {
	tree := goldmark.New(goldmark.WithExtensions(extension.Table)).Parser().Parse(gmtext.NewReader(b))
	return ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		kind := n.Kind().String()
		switch kind {
		case "CodeBlock", "FencedCodeBlock":
			return ast.WalkSkipChildren, nil
		case "HTMLBlock":
			v := n.(*ast.HTMLBlock)
			if v.Lines().Len() > 0 {
				a := v.Lines().At(0).Start
				z := v.Lines().At(v.Lines().Len() - 1).Stop
				if v.HasClosure() {
					z = v.ClosureLine.Stop
				}
				if err := htmlBlocks(b[a:z], offset+a, "html-block", add); err != nil {
					return ast.WalkStop, err
				}
			}
			return ast.WalkSkipChildren, nil
		case "Paragraph", "TextBlock", "Heading", "TableCell":
			if n.Lines().Len() == 0 {
				return ast.WalkContinue, nil
			}
			a := n.Lines().At(0).Start
			z := n.Lines().At(n.Lines().Len() - 1).Stop
			context := "markdown/" + kind
			if n.Parent() != nil && n.Parent().Kind().String() == "ListItem" {
				context = "markdown/list-item"
			}
			add(offset+a, offset+z, context)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
}

func htmlScalar(b []byte, offset int, context string, add func(int, int, string)) error {
	if err := htmlSpans(b, offset, context, func(int, int, string) {}); err != nil {
		return err
	}
	add(offset, offset+len(b), context)
	return nil
}
func mergeDataSpans(spans []span) []span {
	result := []span{}
	for _, s := range spans {
		if len(result) > 0 && result[len(result)-1].context == s.context {
			result[len(result)-1].end = s.end
		} else {
			result = append(result, s)
		}
	}
	return result
}

func htmlBlockTag(tag string) bool {
	return strings.Contains(" p li h1 h2 h3 h4 h5 h6 td th dt dd figcaption title ", " "+tag+" ")
}
func htmlBoundaryTag(tag string) bool {
	return htmlBlockTag(tag) || strings.Contains(" div section article main ul ol table tr thead tbody blockquote dl header footer nav aside pre script style hr ", " "+tag+" ")
}

// HTML's tokenizer establishes tags/attributes. Complete leaf block elements
// are units; text/inline runs outside those elements form complete blocks too.
func htmlBlocks(b []byte, offset int, context string, add func(int, int, string)) error {
	if err := htmlSpans(b, offset, context, func(int, int, string) {}); err != nil {
		return err
	}
	z := html.NewTokenizer(bytes.NewReader(b))
	pos := 0
	type element struct {
		tag       string
		start     int
		listDepth int
	}
	stack := []element{}
	listDepth := 0
	blocks := []span{}
	attributes := []span{}
	runs := []span{}
	run := -1
	flush := func(end int) {
		if run >= 0 {
			runs = append(runs, span{run, end, context + "/text-block"})
			run = -1
		}
	}
	closeBlock := func(end int) {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		blocks = append(blocks, span{e.start, end, context + "/" + e.tag})
	}
	for {
		t := z.Next()
		raw := z.Raw()
		a := pos
		pos += len(raw)
		if t == html.ErrorToken {
			if z.Err() != io.EOF {
				return z.Err()
			}
			break
		}
		if t == html.StartTagToken || t == html.EndTagToken || t == html.SelfClosingTagToken {
			if t != html.EndTagToken {
				if err := htmlAttributeSpans(raw, a, context, func(a, z int, c string) { attributes = append(attributes, span{a, z, c}) }); err != nil {
					return err
				}
			}
			n, _ := z.TagName()
			tag := string(n)
			if t == html.StartTagToken && (tag == "ul" || tag == "ol") {
				listDepth++
			}
			if htmlBoundaryTag(tag) {
				flush(a)
			}
			if htmlBlockTag(tag) {
				if t == html.StartTagToken {
					// HTML5 optional-end elements: upstream includes omitted p ends,
					// and a th followed by a stray </td>. Preserve original bytes.
					if len(stack) > 0 {
						top := stack[len(stack)-1].tag
						if (tag == "p" && top == "p") || ((tag == "td" || tag == "th") && (top == "td" || top == "th")) || (tag == "li" && top == "li" && stack[len(stack)-1].listDepth == listDepth) {
							closeBlock(a)
						}
					}
					stack = append(stack, element{tag, pos, listDepth})
				}
				if t == html.EndTagToken {
					found := -1
					for j := len(stack) - 1; j >= 0; j-- {
						if stack[j].tag == tag {
							found = j
							break
						}
					}
					if found >= 0 {
						for len(stack) > found {
							closeBlock(a)
						}
					} else if tag != "td" && tag != "th" && tag != "p" && tag != "li" {
						return fmt.Errorf("unbalanced HTML linguistic block %s", tag)
					}
				}
			}
			if t == html.EndTagToken && (tag == "ul" || tag == "ol") {
				for len(stack) > 0 && stack[len(stack)-1].listDepth >= listDepth {
					closeBlock(a)
				}
				listDepth--
			}
			if t == html.EndTagToken && (tag == "tr" || tag == "table") {
				for len(stack) > 0 {
					top := stack[len(stack)-1].tag
					if top != "p" && top != "td" && top != "th" {
						break
					}
					closeBlock(a)
				}
			}
			if !htmlBoundaryTag(tag) && run < 0 {
				run = a
			}
		} else if t == html.TextToken {
			if run < 0 {
				run = a
			}
		} else {
			flush(a)
		}
	}
	flush(pos)
	if len(stack) > 0 {
		return fmt.Errorf("unclosed HTML linguistic block")
	}
	protected, err := htmlProtectedRanges(b)
	if err != nil {
		return err
	}
	// Keep inner paragraph units rather than a containing list item's huge body.
	leaves := []span{}
	for _, s := range blocks {
		inner := false
		for _, q := range blocks {
			if q.start > s.start && q.end <= s.end {
				inner = true
				break
			}
		}
		if !inner {
			leaves = append(leaves, s)
			add(offset+s.start, offset+s.end, s.context)
		}
	}
	for _, s := range runs {
		overlap := false
		for _, q := range append(append([]span{}, leaves...), protected...) {
			if s.start < q.end && s.end > q.start {
				overlap = true
				break
			}
		}
		if !overlap && htmlRunLanguage(b[s.start:s.end]) {
			add(offset+s.start, offset+s.end, s.context)
		}
	}
	for _, s := range attributes {
		overlap := false
		for _, q := range append(append([]span{}, leaves...), protected...) {
			if s.start < q.end && s.end > q.start {
				overlap = true
				break
			}
		}
		if !overlap {
			add(offset+s.start, offset+s.end, s.context)
		}
	}
	return nil
}

func htmlRunLanguage(b []byte) bool {
	z := html.NewTokenizer(bytes.NewReader(b))
	for {
		typ := z.Next()
		if typ == html.ErrorToken {
			return false
		}
		if typ == html.TextToken {
			visible, _, err := protectBlock(z.Raw(), "html/text")
			if err == nil && hasLanguage(visible) {
				return true
			}
		}
	}
}

// protectBlock is a bounded inline lexer inside AST/tokenizer-approved blocks,
// not a Markdown document parser. Unsupported inline syntax is rejected. Its
// output is re-parsed after restoration to validate the full block structure.
func protectBlock(b []byte, context string) (string, []ProtectedToken, error) {
	if bytes.ContainsAny(b, "⟦⟧") {
		return "", nil, fmt.Errorf("reserved protected-token delimiter in source")
	}
	locked := []ProtectedToken{}
	var out strings.Builder
	emit := func(raw, role, pair string) string {
		token := fmt.Sprintf("⟦P%d⟧", len(locked))
		locked = append(locked, ProtectedToken{token, raw, role, pair})
		out.WriteString(token)
		return token
	}
	markdown := strings.HasPrefix(context, "markdown/")
	var scan func([]byte) error
	scan = func(text []byte) error {
		for i := 0; i < len(text); {
			if bytes.HasPrefix(text[i:], []byte("{{")) {
				end := bytes.Index(text[i+2:], []byte("}}"))
				if end < 0 {
					return fmt.Errorf("unterminated template action")
				}
				end += i + 4
				emit(string(text[i:end]), "atom", "")
				i = end
				continue
			}
			if bytes.HasPrefix(text[i:], []byte("{#")) {
				end := bytes.IndexByte(text[i:], '}')
				if end < 3 {
					return fmt.Errorf("invalid heading anchor")
				}
				end += i + 1
				for _, r := range string(text[i+2 : end-1]) {
					if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.' {
						return fmt.Errorf("unknown heading anchor")
					}
				}
				if !strings.Contains(context, "Heading") || strings.TrimSpace(string(text[end:])) != "" {
					return fmt.Errorf("anchor outside heading suffix")
				}
				emit(string(text[i:end]), "suffix", "")
				i = end
				continue
			}
			if markdown && text[i] == '`' {
				n := 1
				for i+n < len(text) && text[i+n] == '`' {
					n++
				}
				delim := bytes.Repeat([]byte{'`'}, n)
				end := bytes.Index(text[i+n:], delim)
				if end < 0 {
					return fmt.Errorf("unclosed inline code")
				}
				end += i + 2*n
				emit(string(text[i:end]), "atom", "")
				i = end
				continue
			}
			if markdown && (text[i] == '[' || (text[i] == '!' && i+1 < len(text) && text[i+1] == '[')) {
				label := i + 1
				if text[i] == '!' {
					label++
				}
				end := balancedEnd(text, label-1, '[', ']')
				if end < 0 {
					return fmt.Errorf("unclosed link label")
				}
				finish := end + 1
				if finish < len(text) && text[finish] == '(' {
					q := balancedEnd(text, finish, '(', ')')
					if q < 0 {
						return fmt.Errorf("unclosed link destination")
					}
					finish = q + 1
				} else if finish < len(text) && text[finish] == '[' {
					q := balancedEnd(text, finish, '[', ']')
					if q < 0 {
						return fmt.Errorf("unclosed reference link")
					}
					finish = q + 1
				}
				pair := fmt.Sprintf("binding-%d", len(locked))
				emit(string(text[i:label]), "open", pair)
				if err := scan(text[label:end]); err != nil {
					return err
				}
				emit(string(text[end:finish]), "close", pair)
				i = finish
				continue
			}
			if text[i] == '<' {
				z := html.NewTokenizer(bytes.NewReader(text[i:]))
				typ := z.Next()
				raw := string(z.Raw())
				if typ == html.StartTagToken || typ == html.SelfClosingTagToken {
					tag, _ := z.TagName()
					name := string(tag)
					if !knownHTMLTag(name) {
						return fmt.Errorf("unknown HTML structure %s", name)
					}
					if htmlBoundaryTag(name) && name != "pre" && name != "script" && name != "style" {
						return fmt.Errorf("block HTML inside linguistic block")
					}
					if typ == html.SelfClosingTagToken || strings.Contains(" br img wbr input source meta link hr ", " "+name+" ") {
						emit(raw, "atom", "")
						i += len(raw)
						continue
					}
					// Find matching closing tag with tokenizer nesting, including code.
					consumed := len(raw)
					depth := 1
					closeStart := -1
					for depth > 0 {
						t := z.Next()
						a := consumed
						consumed += len(z.Raw())
						if t == html.ErrorToken {
							return fmt.Errorf("unclosed inline HTML %s", name)
						}
						if t == html.StartTagToken || t == html.EndTagToken {
							n, _ := z.TagName()
							if string(n) == name {
								if t == html.StartTagToken {
									depth++
								} else {
									depth--
									if depth == 0 {
										closeStart = a
									}
								}
							}
						}
					}
					if strings.Contains(" code var kbd samp tt textarea pre script style ", " "+name+" ") {
						emit(string(text[i:i+consumed]), "atom", "")
					} else {
						pair := fmt.Sprintf("binding-%d", len(locked))
						emit(raw, "open", pair)
						if err := scan(text[i+len(raw) : i+closeStart]); err != nil {
							return err
						}
						emit(string(text[i+closeStart:i+consumed]), "close", pair)
					}
					i += consumed
					continue
				}
				if typ == html.CommentToken {
					emit(raw, "atom", "")
					i += len(raw)
					continue
				}
				if typ == html.EndTagToken {
					tag, _ := z.TagName()
					if !knownHTMLTag(string(tag)) {
						return fmt.Errorf("unknown HTML end tag")
					}
					// Known unmatched end tags are inert in HTML5; frozen upstream
					// contains one. Keep the entire original token locked.
					emit(raw, "atom", "")
					i += len(raw)
					continue
				}
				// Markdown autolinks have no editable label.
				if markdown {
					end := bytes.IndexByte(text[i:], '>')
					if end > 0 && strings.ContainsAny(string(text[i:i+end]), ":@") {
						emit(string(text[i:i+end+1]), "atom", "")
						i += end + 1
						continue
					}
				}
			}
			if text[i] == '\n' {
				end := i + 1
				for end < len(text) && (text[end] == ' ' || text[end] == '\t' || (markdown && text[end] == '>')) {
					end++
				}
				emit(string(text[i:end]), "atom", "")
				i = end
				continue
			}
			if text[i] == '&' {
				end := bytes.IndexByte(text[i:], ';')
				if end > 0 && end < 32 {
					emit(string(text[i:i+end+1]), "atom", "")
					i += end + 1
					continue
				}
			}
			if markdown && text[i] == '\\' {
				_, n := utf8.DecodeRune(text[i+1:])
				if n == 0 {
					return fmt.Errorf("dangling escape")
				}
				emit(string(text[i:i+1+n]), "atom", "")
				i += 1 + n
				continue
			}
			if markdown && (text[i] == '*' || text[i] == '_') {
				end := i + 1
				for end < len(text) && text[end] == text[i] {
					end++
				}
				emit(string(text[i:end]), "atom", "")
				i = end
				continue
			}
			// Machine words are locked as complete lexical tokens, never split
			// into independently translated fragments or punctuation units.
			if !unicode.IsSpace(rune(text[i])) {
				end := i
				for end < len(text) && !unicode.IsSpace(rune(text[end])) && !strings.ContainsRune("<>[]{}*`&\\", rune(text[end])) {
					end++
				}
				if end > i {
					word := string(text[i:end])
					trim := strings.Trim(word, ".,;:!?\"'()")
					tech := (len(trim) > 1 && strings.ToUpper(trim) == trim && strings.IndexFunc(trim, func(r rune) bool { return r >= 'A' && r <= 'Z' }) >= 0) || strings.ContainsAny(trim, "/\\_") || strings.Contains(trim, "://") || strings.Contains(trim, ".") || strings.Contains(trim, "=") || strings.HasPrefix(trim, "--") || strings.Contains(word, "()")
					for j := 1; j < len(trim); j++ {
						if trim[j] >= 'A' && trim[j] <= 'Z' && trim[j-1] >= 'a' && trim[j-1] <= 'z' {
							tech = true
						}
					}
					if tech && trim != "" {
						a := strings.Index(word, trim)
						out.WriteString(word[:a])
						emit(trim, "atom", "")
						out.WriteString(word[a+len(trim):])
						i = end
						continue
					}
					out.WriteString(word)
					i = end
					continue
				}
			}
			r, n := utf8.DecodeRune(text[i:])
			out.WriteRune(r)
			i += n
		}
		return nil
	}
	if err := scan(b); err != nil {
		return "", nil, err
	}
	return out.String(), locked, nil
}

func balancedEnd(b []byte, start int, open, close byte) int {
	depth := 0
	for i := start; i < len(b); i++ {
		if b[i] == '\\' {
			i++
			continue
		}
		if b[i] == open {
			depth++
		}
		if b[i] == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
