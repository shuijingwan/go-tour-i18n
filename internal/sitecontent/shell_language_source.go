package sitecontent

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	yaml "github.com/shuijingwan/go-tour-i18n/internal/sitecontent/shellyaml"
	"golang.org/x/net/html"
)

// This parser is a pending source foundation, not a renderer or language gate.
// Its slots are reconstruction coordinates; only index.md exposes a Page TU.
const ShellExtractionContract = "go-learning/shell-language-source/v1"

type ShellLanguageSlot struct {
	Unit     Unit   `json:"unit"`
	Role     string `json:"role"`
	Context  string `json:"context"`
	Encoding string `json:"encoding"`
}

type ShellLanguageSource struct {
	Path          string              `json:"path"`
	Role          string              `json:"role"`
	SourceSHA     string              `json:"source_sha256"`
	StructureSHA  string              `json:"structure_sha256"`
	Slots         []ShellLanguageSlot `json:"slots"`
	LockedVisible []string            `json:"locked_visible_identity_locators"`
	Page          *Document           `json:"formal_page,omitempty"`
}

type shellExtractor struct {
	source      ShellLanguageSource
	raw         []byte
	occurrences map[string]int
	semantic    bool // new Page exports only; frozen foundations retain the old reader
}

func (e *shellExtractor) slot(a, z int, value, role, context, encoding string) error {
	if a < 0 || z > len(e.raw) || a > z {
		return fmt.Errorf("invalid shell slot coordinates: [%d,%d) role=%s context=%s source_size=%d", a, z, role, context, len(e.raw))
	}
	if a == z {
		return nil
	} // Empty accessibility attributes carry no language.
	value = strings.TrimSpace(value)
	text, locked, err := protectBlockPolicy([]byte(value), "html/"+role, true, nil, !e.semantic)
	if err != nil {
		return err
	}
	if !hasLanguage(text) {
		return nil
	}
	return e.protectedSlot(a, z, text, locked, role, context, encoding)
}

func (e *shellExtractor) protectedSlot(a, z int, text string, locked []ProtectedToken, role, context, encoding string) error {
	sh := unitSourceIdentity(text, locked)
	base := e.source.Path + "#" + context + "/" + role + ":" + sh
	e.occurrences[base]++
	locator := base + ":" + strconv.Itoa(e.occurrences[base])
	u := Unit{ID: digest([]byte(ShellExtractionContract + "\n" + locator)), Locator: locator, Source: text, SourceSHA: sh, Start: a, End: z, Protected: locked}
	e.source.Slots = append(e.source.Slots, ShellLanguageSlot{u, role, context, encoding})
	return nil
}

func (e *shellExtractor) finish() (*ShellLanguageSource, error) {
	sort.Slice(e.source.Slots, func(i, j int) bool { return e.source.Slots[i].Unit.Start < e.source.Slots[j].Unit.Start })
	var shape bytes.Buffer
	pos := 0
	for _, s := range e.source.Slots {
		u := s.Unit
		if u.Start < pos {
			return nil, fmt.Errorf("overlapping shell language slots: %s", u.Locator)
		}
		shape.Write(e.raw[pos:u.Start])
		shape.WriteString("\x00" + s.Role + "\x00" + s.Context + "\x00" + s.Encoding + "\x00" + protectionIdentity(u.Protected))
		pos = u.End
	}
	shape.Write(e.raw[pos:])
	e.source.StructureSHA = digest(shape.Bytes())
	if len(e.source.Slots) == 0 {
		return nil, fmt.Errorf("empty shell language source")
	}
	if e.source.Role == "homepage-page" {
		d := Document{Contract: ShellExtractionContract, Path: e.source.Path, Kind: "page", Route: "/", SourceSHA: e.source.SourceSHA, StructureSHA: e.source.StructureSHA, Dependencies: []Dependency{}, Units: []Unit{}}
		for _, s := range e.source.Slots {
			d.Units = append(d.Units, s.Unit)
		}
		page, err := FormalPage(d, e.raw, 1)
		if err != nil {
			return nil, err
		}
		e.source.Page = &page
	}
	return &e.source, nil
}

func ExtractShellLanguageSource(path string, raw []byte) (*ShellLanguageSource, error) {
	return extractShellLanguageSource(path, raw, false)
}

func extractShellLanguageSource(path string, raw []byte, semantic bool) (*ShellLanguageSource, error) {
	roles := map[string]string{"_content/index.md": "homepage-page", "_content/site.tmpl": "shared-chrome", "_content/menus.yaml": "menus", "_content/resources.yaml": "resources", "_content/testimonials.yaml": "testimonials"}
	role, ok := roles[path]
	if !ok {
		return nil, fmt.Errorf("unregistered shell language source %s", path)
	}
	if err := textBytes(raw); err != nil {
		return nil, err
	}
	e := &shellExtractor{source: ShellLanguageSource{Path: path, Role: role, SourceSHA: digest(raw), Slots: []ShellLanguageSlot{}, LockedVisible: []string{}}, raw: raw, occurrences: map[string]int{}, semantic: semantic}
	var err error
	if strings.HasSuffix(path, ".yaml") {
		err = e.yaml()
	} else {
		err = e.markup()
	}
	if err != nil {
		return nil, fmt.Errorf("extract %s: %w", path, err)
	}
	return e.finish()
}

// Lex template actions without executing them. Comments and quoted delimiters
// cannot leak into HTML text/attributes. The original bytes remain protected.
func shellTemplateRanges(raw []byte) ([]span, error) {
	out := []span{}
	for pos := 0; pos < len(raw); {
		n := bytes.Index(raw[pos:], []byte("{{"))
		if n < 0 {
			break
		}
		a := pos + n
		i := a + 2
		if i < len(raw) && raw[i] == '-' {
			i++
		}
		for i < len(raw) && unicode.IsSpace(rune(raw[i])) {
			i++
		}
		if bytes.HasPrefix(raw[i:], []byte("/*")) {
			end := bytes.Index(raw[i+2:], []byte("*/"))
			if end < 0 {
				return nil, fmt.Errorf("unterminated template comment")
			}
			i += end + 4
			for i < len(raw) && (unicode.IsSpace(rune(raw[i])) || raw[i] == '-') {
				i++
			}
			if !bytes.HasPrefix(raw[i:], []byte("}}")) {
				return nil, fmt.Errorf("invalid template comment end")
			}
			out = append(out, span{a, i + 2, "template"})
			pos = i + 2
			continue
		}
		quote := byte(0)
		end := -1
		for ; i < len(raw)-1; i++ {
			c := raw[i]
			if quote != 0 {
				if quote == '"' && c == '\\' {
					i++
					continue
				}
				if c == quote {
					quote = 0
				}
				continue
			}
			if c == '"' || c == '\'' || c == '`' {
				quote = c
				continue
			}
			if c == '}' && raw[i+1] == '}' {
				end = i + 2
				break
			}
		}
		if end < 0 || quote != 0 {
			return nil, fmt.Errorf("unterminated template action")
		}
		out = append(out, span{a, end, "template"})
		pos = end
	}
	return out, nil
}

var shellDefine = regexp.MustCompile(`^(define|block)\s+("[^"\n]+")`)
var shellAltAssignment = regexp.MustCompile(`^\$alt\s*(?::=|=)\s*("(?:[^"\\]|\\.)*")$`)
var shellUnquotedVisibleAttribute = regexp.MustCompile(`\b(alt|title|aria-label)\s*=\s*([^'"\s>]+)`)

// Frozen site.tmpl contains aria-label={{.name}}. That is a reference to the
// menus language source, not an unquoted static language value. Hide only this
// entirely dynamic value from the existing quoted-attribute extractor. Static
// unquoted labels still fail closed; all original source bytes stay untouched.
func shellAttributeMask(masked, original []byte) ([]byte, error) {
	result := bytes.Clone(masked)
	for _, m := range shellUnquotedVisibleAttribute.FindAllSubmatchIndex(masked, -1) {
		value := original[m[4]:m[5]]
		ranges, err := shellTemplateRanges(value)
		if err != nil {
			return nil, err
		}
		if len(ranges) == 1 && ranges[0].start == 0 && ranges[0].end == len(value) {
			for i := m[2]; i < m[3]; i++ {
				result[i] = 'x'
			}
		}
	}
	return result, nil
}

func shellAction(b []byte) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(string(b[2:len(b)-2])), "-"), "-"))
}

func validateShellTemplate(raw []byte, ranges []span) error {
	funcs := template.FuncMap{}
	for _, name := range strings.Fields("add sub mul div data page pages file first markdown raw yaml path strings googleCN newest section") {
		funcs[name] = func(...any) any { return nil }
	}
	var normalized strings.Builder
	pos := 0
	for _, r := range ranges {
		normalized.Write(raw[pos:r.start])
		action := shellAction(raw[r.start:r.end])
		if m := shellDefine.FindStringSubmatchIndex(action); m != nil {
			name, err := strconv.Unquote(action[m[4]:m[5]])
			if err != nil {
				return err
			}
			fields := strings.Fields(name)
			if len(fields) == 0 {
				return fmt.Errorf("empty template name")
			}
			funcs[fields[0]] = func(...any) any { return nil }
			action = action[:m[4]] + strconv.Quote(fields[0]) + action[m[5]:]
		}
		normalized.WriteString("{{" + action + "}}")
		pos = r.end
	}
	normalized.Write(raw[pos:])
	_, err := template.New("shell-source-syntax").Funcs(funcs).Parse(normalized.String())
	return err
}

func trimShellSpan(b []byte, a, z int) (int, int) {
	for a < z {
		r, n := utf8.DecodeRune(b[a:z])
		if !unicode.IsSpace(r) {
			break
		}
		a += n
	}
	for z > a {
		r, n := utf8.DecodeLastRune(b[a:z])
		if !unicode.IsSpace(r) {
			break
		}
		z -= n
	}
	return a, z
}

type shellSentencePeriod struct {
	Prefix string // exact frozen visible-prose slot immediately before the link
	Slot   ShellLanguageSlot
}

// Prospective shared representation only. The legacy extractor continues to
// leave punctuation-only nodes frozen. A period is a sentence boundary here
// only after an inline link with an independent prose prefix, at the end of p.
func shellSentencePeriods(old ShellLanguageSource, raw []byte) ([]shellSentencePeriod, error) {
	if old.Role != "shared-chrome" || old.Page != nil || old.SourceSHA != digest(raw) {
		return nil, fmt.Errorf("shared HTML source required for sentence punctuation")
	}
	ranges, err := shellTemplateRanges(raw)
	if err != nil {
		return nil, err
	}
	masked := bytes.Clone(raw)
	for _, r := range ranges {
		for i := r.start; i < r.end; i++ {
			if masked[i] != '\n' {
				masked[i] = 'x'
			}
		}
	}
	type element struct {
		tag       string
		end, para int
		hidden    bool
		visible   bool
		prefix    *ShellLanguageSlot
	}
	stack := []element{}
	paragraph, pos := 0, 0
	var link *element
	out := []shellSentencePeriod{}
	z := html.NewTokenizer(bytes.NewReader(masked))
	for {
		typ := z.Next()
		a := pos
		pos += len(z.Raw())
		if typ == html.ErrorToken {
			if z.Err() != io.EOF {
				return nil, z.Err()
			}
			return out, nil
		}
		if typ == html.TextToken && link != nil {
			start, end := trimShellSpan(raw, a, pos)
			if string(raw[start:end]) == "." && !bytes.Contains(raw[link.prefix.Unit.Start:end], []byte("{{")) {
				closing := []string{}
				for _, n := range stack {
					if n.tag == "p" {
						closing = nil
					}
					closing = append(closing, n.tag)
				}
				if len(closing) > 0 && closing[0] == "p" && shellParagraphEnds(masked[pos:], closing) {
					frame := ShellLanguageSlot{Unit: Unit{Start: start, End: end, Source: ".", Protected: []ProtectedToken{}}, Role: "sentence-punctuation", Context: link.prefix.Context + "/sentence-final-period/p" + strconv.Itoa(link.para), Encoding: "html-text"}
					out = append(out, shellSentencePeriod{Prefix: link.prefix.Unit.ID, Slot: frame})
				}
			}
		}
		if typ == html.TextToken {
			hidden, anchor := false, -1
			for i, n := range stack {
				hidden = hidden || n.hidden
				if n.tag == "a" {
					anchor = i
				}
			}
			if !hidden && anchor >= 0 {
				for _, r := range string(masked[a:pos]) {
					if unicode.IsLetter(r) || unicode.IsDigit(r) {
						stack[anchor].visible = true
					}
				}
			}
		}
		link = nil // must be the very next text node after </a>
		if typ == html.StartTagToken || typ == html.SelfClosingTagToken {
			tok := z.Token()
			n := element{tag: tok.Data, end: pos}
			p := -1
			for i, parent := range stack {
				n.hidden = n.hidden || parent.hidden
				if parent.tag == "p" {
					p, n.para = i, parent.para
				}
			}
			if strings.Contains(" script style pre code textarea var kbd samp tt ", " "+n.tag+" ") {
				n.hidden = true
			}
			for _, attr := range tok.Attr {
				if attr.Key == "hidden" || (attr.Key == "aria-hidden" && attr.Val == "true") || (attr.Key == "class" && strings.Contains(attr.Val, "material-icons")) {
					n.hidden = true
				}
			}
			if n.tag == "p" {
				paragraph++
				n.para = paragraph
			}
			if n.tag == "a" && !n.hidden && p >= 0 {
				for i := range old.Slots {
					s := &old.Slots[i]
					if s.Role != "visible-text" || s.Encoding != "html-text" || s.Unit.Start < stack[p].end || s.Unit.End > a || len(bytes.TrimSpace(raw[s.Unit.End:a])) != 0 || !hasLanguage(s.Unit.Source) {
						continue
					}
					last, _ := utf8.DecodeLastRune(raw[s.Unit.Start:s.Unit.End])
					if unicode.IsLetter(last) {
						n.prefix = s
					}
				}
			}
			if typ == html.StartTagToken && !strings.Contains(" area base br col embed hr img input link meta param source track wbr ", " "+n.tag+" ") {
				stack = append(stack, n)
			}
		} else if typ == html.EndTagToken {
			name, _ := z.TagName()
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].tag == string(name) {
					if string(name) == "a" && stack[i].prefix != nil && stack[i].visible {
						n := stack[i]
						link = &n
					}
					stack = stack[:i]
					break
				}
			}
		}
	}
}

func shellParagraphEnds(raw []byte, closing []string) bool {
	z := html.NewTokenizer(bytes.NewReader(raw))
	for len(closing) > 0 {
		switch z.Next() {
		case html.TextToken:
			if len(bytes.TrimSpace(z.Raw())) != 0 {
				return false
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			if string(name) != closing[len(closing)-1] {
				return false
			}
			closing = closing[:len(closing)-1]
		default:
			return false
		}
	}
	return true
}

func (e *shellExtractor) markup() error {
	offset := 0
	if e.source.Role == "homepage-page" {
		end := bytes.Index(e.raw[4:], []byte("\n---\n"))
		if !bytes.HasPrefix(e.raw, []byte("---\n")) || end < 0 {
			return fmt.Errorf("homepage requires frozen metadata")
		}
		meta := e.raw[4 : 4+end]
		n, err := shellYAMLNode(meta)
		if err != nil {
			return err
		}
		if n.Kind != yaml.MappingNode {
			return fmt.Errorf("invalid homepage metadata")
		}
		for i := 0; i < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			switch k.Value {
			case "title", "summary":
				a, z, enc, err := shellScalarRange(meta, v)
				if err != nil {
					return err
				}
				if err = e.slot(a+4, z+4, string(meta[a:z]), "metadata/"+k.Value, "homepage", enc); err != nil {
					return err
				}
			case "template":
				if v.Tag != "!!bool" || v.Value != "true" {
					return fmt.Errorf("homepage must be templated")
				}
			default:
				return fmt.Errorf("unknown homepage metadata %s", k.Value)
			}
		}
		offset = 4 + end + 5
	}
	ranges, err := shellTemplateRanges(e.raw)
	if err != nil {
		return err
	}
	if err = validateShellTemplate(e.raw, ranges); err != nil {
		return err
	}
	masked := bytes.Clone(e.raw)
	for _, r := range ranges {
		for i := r.start; i < r.end; i++ {
			if masked[i] != '\n' {
				masked[i] = 'x'
			}
		}
	}
	// Alt literals are the only frozen assigned strings reaching a visible sink.
	// Compare/dispatch strings, classes, paths and template signatures stay code.
	for _, r := range ranges {
		action := shellAction(e.raw[r.start:r.end])
		if m := shellAltAssignment.FindStringSubmatchIndex(action); m != nil {
			lit := action[m[2]:m[3]]
			value, err := strconv.Unquote(lit)
			if err != nil {
				return err
			}
			if strings.Contains(lit, `\`) {
				return fmt.Errorf("unsupported escaped alt literal")
			}
			a := r.start + bytes.Index(e.raw[r.start:r.end], []byte(lit)) + 1
			if err = e.slot(a, a+len(value), value, "attribute/alt", "template/gopher/$alt", "go-string-interior"); err != nil {
				return err
			}
		}
	}
	return e.markupSpans(masked, offset)
}

func (e *shellExtractor) markupSpans(masked []byte, offset int) error {
	type elem struct {
		tag, context string
		hidden       bool
	}
	stack := []elem{}
	context := func() string {
		parts := []string{}
		for _, v := range stack {
			parts = append(parts, v.context)
		}
		return strings.Join(parts, "/")
	}
	hidden := func() bool {
		for _, v := range stack {
			if v.hidden {
				return true
			}
		}
		return false
	}
	z := html.NewTokenizer(bytes.NewReader(masked[offset:]))
	pos := offset
	for {
		t := z.Next()
		a := pos
		pos += len(z.Raw())
		if t == html.ErrorToken {
			if z.Err() != io.EOF {
				return z.Err()
			}
			break
		}
		if t == html.TextToken {
			if hidden() {
				if e.source.Role == "homepage-page" && len(stack) > 0 && stack[len(stack)-1].tag == "textarea" {
					for _, body := range i18n.SafeTeachingCommentBodies(string(e.raw[a:pos])) {
						for i := range body.Identifiers {
							body.Identifiers[i].Start += a
							body.Identifiers[i].End += a
						}
						s, end := a+body.Start, a+body.End
						text, locked, err := protectTeachingBodyPolicy(e.raw, s, end, body, !e.semantic)
						if err != nil {
							return err
						}
						if err = e.protectedSlot(s, end, text, locked, "teaching-comment", context(), "go-comment-interior"); err != nil {
							return err
						}
					}
				}
				continue
			}
			a, b := trimShellSpan(e.raw, a, pos)
			if a == b {
				continue
			}
			role := "visible-text"
			if len(stack) > 0 && stack[len(stack)-1].tag == "title" {
				role = "metadata/title"
			}
			if err := e.slot(a, b, string(e.raw[a:b]), role, context(), "html-text"); err != nil {
				return err
			}
			continue
		}
		if t == html.StartTagToken || t == html.SelfClosingTagToken {
			tok := z.Token()
			tag := tok.Data
			if !knownHTMLTag(tag) {
				return fmt.Errorf("unknown shell HTML tag %s", tag)
			}
			c := tag
			isHidden := hidden()
			for _, attr := range tok.Attr {
				if attr.Key == "class" {
					c += "." + attr.Val
					if strings.Contains(attr.Val, "material-icons") {
						isHidden = true
					}
				}
				if attr.Key == "id" {
					c += "#" + attr.Val
				}
				if attr.Key == "href" {
					c += "[href=" + attr.Val + "]"
				}
			}
			// Accessibility attributes remain visible language even on controls.
			if tag != "script" && tag != "style" {
				var slotErr error
				attributes, err := shellAttributeMask(masked[a:pos], e.raw[a:pos])
				if err != nil {
					return err
				}
				err = htmlAttributeSpans(attributes, a, "html", func(s, end int, role string) {
					if slotErr == nil {
						slotErr = e.slot(s, end, string(e.raw[s:end]), strings.TrimPrefix(role, "html/"), context()+"/"+c, "html-attribute")
					}
				})
				if err != nil {
					return err
				}
				if slotErr != nil {
					return slotErr
				}
			}
			switch tag {
			case "script", "style", "pre", "code", "textarea", "var", "kbd", "samp", "tt":
				isHidden = true
			}
			if t == html.StartTagToken && !strings.Contains(" area base br col embed hr img input link meta param source track wbr ", " "+tag+" ") {
				stack = append(stack, elem{tag, c, isHidden})
			}
			continue
		}
		if t == html.EndTagToken {
			tok := z.Token()
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].tag == tok.Data {
					stack = stack[:i]
					break
				}
			}
		}
	}
	return nil
}

func shellYAMLNode(b []byte) (*yaml.Node, error) {
	n, err := yaml.Parse(b)
	if err != nil {
		return nil, err
	}
	var check func(*yaml.Node) error
	check = func(n *yaml.Node) error {
		if n.Kind == yaml.AliasNode || n.Anchor != "" {
			return fmt.Errorf("YAML aliases/anchors forbidden")
		}
		if n.Kind == yaml.MappingNode {
			seen := map[string]bool{}
			for i := 0; i < len(n.Content); i += 2 {
				k := n.Content[i]
				if k.Kind != yaml.ScalarNode || k.Tag != "!!str" || seen[k.Value] {
					return fmt.Errorf("duplicate/non-string YAML key")
				}
				seen[k.Value] = true
			}
		}
		for _, c := range n.Content {
			if err := check(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := check(n); err != nil {
		return nil, err
	}
	return n, nil
}

// YAML coordinates retain the exact scalar encoding; semantic source values
// are decoded by the YAML parser, never by a line-based approximate reader.
func shellScalarRange(b []byte, n *yaml.Node) (int, int, string, error) {
	lines := bytes.SplitAfter(b, []byte("\n"))
	a := 0
	if n.Line < 1 || n.Line > len(lines) {
		return 0, 0, "", fmt.Errorf("invalid YAML source coordinate")
	}
	for i := 0; i < n.Line-1; i++ {
		a += len(lines[i])
	}
	// YAML columns count Unicode code points, not bytes.
	line := lines[n.Line-1]
	col := 0
	for i := 0; i < n.Column-1; i++ {
		_, size := utf8.DecodeRune(line[col:])
		col += size
	}
	a += col
	if a >= len(b) {
		return 0, 0, "", fmt.Errorf("empty YAML scalar")
	}
	if n.Style&yaml.FoldedStyle != 0 {
		return 0, 0, "", fmt.Errorf("folded YAML scalar unsupported")
	}
	if n.Style&yaml.LiteralStyle != 0 {
		z := a + len(line) - col
		indent := -1
		for i := n.Line; i < len(lines); i++ {
			l := lines[i]
			trim := bytes.TrimSpace(l)
			spaces := len(l) - len(bytes.TrimLeft(l, " "))
			if len(trim) > 0 {
				if indent < 0 {
					indent = spaces
				}
				if spaces < indent {
					break
				}
			}
			z += len(l)
		}
		return a, z, "yaml-literal", nil
	}
	if b[a] == '"' || b[a] == '\'' {
		q := b[a]
		for i := a + 1; i < len(b); i++ {
			if b[i] == '\n' {
				return 0, 0, "", fmt.Errorf("multiline quoted YAML unsupported")
			}
			if q == '"' && b[i] == '\\' {
				i++
				continue
			}
			if b[i] == q {
				if q == '\'' && i+1 < len(b) && b[i+1] == q {
					i++
					continue
				}
				return a, i + 1, "yaml-quoted", nil
			}
		}
		return 0, 0, "", fmt.Errorf("unterminated YAML string")
	}
	z := a
	for z < len(b) && b[z] != '\n' {
		if n.Style&yaml.FlowStyle != 0 && strings.ContainsRune(",]}", rune(b[z])) {
			break
		}
		if b[z] == '#' && (z == a || b[z-1] == ' ') {
			break
		}
		z++
	}
	_, z = trimShellSpan(b, a, z)
	return a, z, "yaml-plain", nil
}

func (e *shellExtractor) yaml() error {
	n, err := shellYAMLNode(e.raw)
	if err != nil {
		return err
	}
	var walk func(*yaml.Node, string, string) error
	walk = func(n *yaml.Node, scope, loc string) error {
		if scope == "menus-root" {
			if n.Kind != yaml.MappingNode {
				return fmt.Errorf("menus requires mapping")
			}
			for i := 0; i < len(n.Content); i += 2 {
				k := n.Content[i].Value
				if k != "main" && k != "footer" {
					return fmt.Errorf("unknown menu root %s", k)
				}
				if err := walk(n.Content[i+1], "menu-list", k); err != nil {
					return err
				}
			}
			return nil
		}
		if strings.HasSuffix(scope, "-list") {
			if n.Kind != yaml.SequenceNode {
				return fmt.Errorf("expected YAML sequence at %s", loc)
			}
			for i, c := range n.Content {
				if err := walk(c, strings.TrimSuffix(scope, "-list"), fmt.Sprintf("%s/%d", loc, i)); err != nil {
					return err
				}
			}
			return nil
		}
		if n.Kind != yaml.MappingNode {
			return fmt.Errorf("expected YAML mapping at %s", loc)
		}
		for i := 0; i < len(n.Content); i += 2 {
			k, v := n.Content[i].Value, n.Content[i+1]
			child := loc + "/" + k
			visible, locked, nested := false, false, ""
			switch scope {
			case "menu":
				switch k {
				case "name", "explanation":
					visible = true
				case "url", "external":
					locked = true
				case "children":
					nested = "menu-list"
				case "socialIconsList":
					nested = "social-list"
				}
			case "social":
				switch k {
				case "name":
					visible = true
				case "url", "img", "darkImg":
					locked = true
				}
			case "resource":
				switch k {
				case "title", "description", "iconName":
					visible = true
				case "link", "icon", "iconDark":
					locked = true
				case "packages":
					nested = "package-list"
				}
			case "package":
				switch k {
				case "title", "url":
					locked = true
				}
			case "testimonial":
				switch k {
				case "title", "quote":
					visible = true
				case "name", "company":
					locked = true
				}
			}
			if nested != "" {
				if err := walk(v, nested, child); err != nil {
					return err
				}
				continue
			}
			if !visible && !locked {
				return fmt.Errorf("unknown %s field %s", scope, k)
			}
			if v.Kind != yaml.ScalarNode {
				return fmt.Errorf("non-scalar %s", child)
			}
			if visible {
				if v.Tag == "!!null" && k == "title" && scope == "testimonial" {
					continue
				}
				if v.Tag != "!!str" {
					return fmt.Errorf("non-string visible scalar %s", child)
				}
				coordinate := *v
				coordinate.Style |= n.Style & yaml.FlowStyle
				a, z, enc, err := shellScalarRange(e.raw, &coordinate)
				if err != nil {
					return err
				}
				if err = e.slot(a, z, v.Value, "structured/"+k, scope+"/"+loc, enc); err != nil {
					return err
				}
			} else {
				if k == "external" {
					if v.Tag != "!!bool" {
						return fmt.Errorf("external must be boolean")
					}
				} else if v.Tag != "!!str" {
					return fmt.Errorf("identity scalar must be string: %s", child)
				}
				if (scope == "package" && k == "title") || (scope == "testimonial" && (k == "name" || k == "company")) {
					e.source.LockedVisible = append(e.source.LockedVisible, child)
				}
			}
		}
		return nil
	}
	scope, loc := "menus-root", ""
	if e.source.Role == "resources" {
		scope, loc = "resource-list", "resources"
	}
	if e.source.Role == "testimonials" {
		scope, loc = "testimonial-list", "testimonials"
	}
	return walk(n, scope, loc)
}

// Proposed contributors are projected independently of raw machine gaps. They
// are not inserted into the current corpus by this foundation task.
func ProjectShellSuccessorContributor(s ShellLanguageSource) (i18n.LanguageContributor, error) {
	projections := []LanguageProjection{}
	for _, slot := range s.Slots {
		text := slot.Unit.Source
		locked := []string{}
		for _, p := range slot.Unit.Protected {
			text = strings.ReplaceAll(text, p.Token, "")
			if !strings.HasPrefix(p.Raw, "{{") {
				locked = append(locked, p.Raw)
			}
		}
		text = html.UnescapeString(text)
		projections = append(projections, LanguageProjection{strings.Join(strings.Fields(text), " "), locked})
	}
	c := i18n.LanguageContributor{ID: "source:" + s.Path, Parser: ShellExtractionContract, Package: "site-v2-shell", Path: s.Path, Route: "", Structured: s.Role != "homepage-page", Purpose: i18n.ContributorPurposeActiveContent, Contexts: finalizeLanguageContexts(projections)}
	if s.Page != nil {
		c.Route = "/"
	}
	if len(c.Contexts) == 0 {
		return c, fmt.Errorf("empty proposed shell contributor")
	}
	c.SourceSHA = i18n.ContributorIdentity(c)
	return c, nil
}
