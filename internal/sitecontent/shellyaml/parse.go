// Package shellyaml parses the explicitly supported frozen shell YAML subset.
// It is not a general YAML reader. Unsupported syntax is rejected, never
// silently normalized or interpreted as a different data type.
package shellyaml

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Kind uint8
type Style uint8

const (
	MappingNode Kind = iota + 1
	SequenceNode
	ScalarNode
	AliasNode
)
const (
	FlowStyle Style = 1 << iota
	LiteralStyle
	FoldedStyle
)

type Node struct {
	Kind         Kind
	Style        Style
	Tag          string
	Value        string
	Content      []*Node
	Line, Column int
	Anchor       string
}
type sourceLine struct {
	text           string
	offset, indent int
}
type parser struct {
	raw   string
	lines []sourceLine
	row   int
}

func Parse(raw []byte) (*Node, error) {
	if !utf8.Valid(raw) || len(raw) > 1<<20 || strings.ContainsAny(string(raw), "\r\x00\ufeff") {
		return nil, fmt.Errorf("invalid frozen YAML bytes")
	}
	p := &parser{raw: string(raw)}
	pos := 0
	for _, line := range strings.SplitAfter(p.raw, "\n") {
		text := strings.TrimSuffix(line, "\n")
		indent := len(text) - len(strings.TrimLeft(text, " "))
		if strings.Contains(text[:indent], "\t") || strings.HasPrefix(text[indent:], "\t") {
			return nil, fmt.Errorf("YAML tab indentation forbidden")
		}
		p.lines = append(p.lines, sourceLine{text, pos, indent})
		pos += len(line)
	}
	p.skip()
	if p.row == len(p.lines) {
		return nil, fmt.Errorf("empty YAML")
	}
	n, err := p.block(p.lines[p.row].indent, 0)
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.row != len(p.lines) {
		return nil, fmt.Errorf("unsupported YAML document/tail at line %d", p.row+1)
	}
	return n, nil
}

func (p *parser) skip() {
	for p.row < len(p.lines) {
		s := strings.TrimSpace(p.lines[p.row].text)
		if s != "" && !strings.HasPrefix(s, "#") {
			break
		}
		p.row++
	}
}
func (p *parser) node(kind Kind, offset int) *Node {
	row := 0
	for row+1 < len(p.lines) && p.lines[row+1].offset <= offset {
		row++
	}
	return &Node{Kind: kind, Line: row + 1, Column: utf8.RuneCountInString(p.raw[p.lines[row].offset:offset]) + 1, Content: []*Node{}}
}
func sequenceLine(s string) bool { return s == "-" || strings.HasPrefix(s, "- ") }

func (p *parser) block(indent, depth int) (*Node, error) {
	if depth > 64 {
		return nil, fmt.Errorf("YAML nesting limit")
	}
	p.skip()
	if p.row >= len(p.lines) {
		return nil, fmt.Errorf("missing YAML block")
	}
	l := p.lines[p.row]
	if l.indent != indent {
		return nil, fmt.Errorf("unexpected YAML indentation at line %d", p.row+1)
	}
	s := l.text[indent:]
	if strings.HasPrefix(s, "[") || strings.HasPrefix(s, "{") {
		n, end, err := p.flow(l.offset+indent, depth)
		if err != nil {
			return nil, err
		}
		return n, p.consumeFlowTail(end)
	}
	if sequenceLine(s) {
		return p.sequence(indent, depth)
	}
	return p.mapping(indent, depth, nil)
}

func (p *parser) sequence(indent, depth int) (*Node, error) {
	n := p.node(SequenceNode, p.lines[p.row].offset+indent)
	for {
		p.skip()
		if p.row >= len(p.lines) || p.lines[p.row].indent < indent {
			break
		}
		l := p.lines[p.row]
		s := l.text[l.indent:]
		if l.indent != indent || !sequenceLine(s) {
			break
		}
		a := l.offset + indent + 1
		for a < l.offset+len(l.text) && p.raw[a] == ' ' {
			a++
		}
		if a == l.offset+len(l.text) {
			p.row++
			p.skip()
			if p.row >= len(p.lines) {
				return nil, fmt.Errorf("empty YAML sequence entry")
			}
			child, err := p.block(p.lines[p.row].indent, depth+1)
			if err != nil {
				return nil, err
			}
			n.Content = append(n.Content, child)
			continue
		}
		if strings.HasPrefix(p.raw[a:], "{") || strings.HasPrefix(p.raw[a:], "[") {
			child, end, err := p.flow(a, depth+1)
			if err != nil {
				return nil, err
			}
			if err = p.consumeFlowTail(end); err != nil {
				return nil, err
			}
			n.Content = append(n.Content, child)
			continue
		}
		// The first mapping key occurs on the same line as its sequence marker.
		child := p.node(MappingNode, a)
		if err := p.entry(child, a, l.offset+len(l.text), indent+2, depth+1); err != nil {
			return nil, err
		}
		child, err := p.mapping(indent+2, depth+1, child)
		if err != nil {
			return nil, err
		}
		n.Content = append(n.Content, child)
	}
	return n, nil
}

func (p *parser) mapping(indent, depth int, n *Node) (*Node, error) {
	if n == nil {
		n = p.node(MappingNode, p.lines[p.row].offset+indent)
	}
	for {
		p.skip()
		if p.row >= len(p.lines) || p.lines[p.row].indent < indent {
			break
		}
		l := p.lines[p.row]
		if l.indent != indent {
			return nil, fmt.Errorf("unexpected mapping indentation at line %d", p.row+1)
		}
		if sequenceLine(l.text[indent:]) {
			break
		}
		if err := p.entry(n, l.offset+indent, l.offset+len(l.text), indent, depth); err != nil {
			return nil, err
		}
	}
	return n, nil
}

func validKey(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if c != '_' && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (i == 0 || c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func (p *parser) entry(n *Node, a, end, indent, depth int) error {
	colon := strings.IndexByte(p.raw[a:end], ':')
	if colon < 0 {
		return fmt.Errorf("expected YAML mapping key at line %d", p.row+1)
	}
	key := strings.TrimSpace(p.raw[a : a+colon])
	if !validKey(key) {
		return fmt.Errorf("unsupported YAML key %q", key)
	}
	for i := 0; i < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return fmt.Errorf("duplicate YAML key %s", key)
		}
	}
	k := p.node(ScalarNode, a)
	k.Tag = "!!str"
	k.Value = key
	value := a + colon + 1
	if value < end && p.raw[value] != ' ' {
		return fmt.Errorf("mapping colon requires separation")
	}
	for value < end && p.raw[value] == ' ' {
		value++
	}
	var v *Node
	var err error
	s := strings.TrimSpace(p.raw[value:end])
	if s == "" || strings.HasPrefix(s, "#") {
		p.row++
		p.skip()
		// YAML permits an indentless sequence immediately below a mapping key.
		if p.row < len(p.lines) && (p.lines[p.row].indent > indent || (p.lines[p.row].indent == indent && sequenceLine(p.lines[p.row].text[indent:]))) {
			v, err = p.block(p.lines[p.row].indent, depth+1)
		} else {
			v = p.node(ScalarNode, value)
			v.Tag = "!!null"
		}
	} else if s == "|" || s == "|-" {
		v, err = p.literal(value, indent, s == "|-")
	} else if strings.HasPrefix(s, "|") || strings.HasPrefix(s, ">") {
		return fmt.Errorf("unsupported block scalar header %q", s)
	} else if s[0] == '[' || s[0] == '{' {
		var finish int
		v, finish, err = p.flow(value, depth+1)
		if err == nil {
			err = p.consumeFlowTail(finish)
		}
	} else {
		v, err = p.scalar(value, end, false)
		p.row++
	}
	if err != nil {
		return err
	}
	n.Content = append(n.Content, k, v)
	return nil
}

func (p *parser) scalar(a, end int, flow bool) (*Node, error) {
	n := p.node(ScalarNode, a)
	n.Tag = "!!str"
	s := strings.TrimSpace(p.raw[a:end])
	if s == "" {
		n.Tag = "!!null"
		return n, nil
	}
	if strings.ContainsAny(s, "\n\t") {
		return nil, fmt.Errorf("multiline/tab scalar unsupported")
	}
	if strings.ContainsRune("&*!%?@`", rune(s[0])) || s == "---" || s == "..." {
		return nil, fmt.Errorf("unsupported YAML scalar syntax %q", s)
	}
	if s[0] == '"' || s[0] == '\'' {
		q := s[0]
		finish := -1
		for i := 1; i < len(s); i++ {
			if q == '"' && s[i] == '\\' {
				i++
				continue
			}
			if s[i] == q {
				if q == '\'' && i+1 < len(s) && s[i+1] == q {
					i++
					continue
				}
				finish = i + 1
				break
			}
		}
		if finish < 0 {
			return nil, fmt.Errorf("unterminated YAML quote")
		}
		tail := strings.TrimSpace(s[finish:])
		if tail != "" && !strings.HasPrefix(tail, "#") {
			return nil, fmt.Errorf("quoted YAML scalar tail")
		}
		if q == '"' {
			if err := json.Unmarshal([]byte(s[:finish]), &n.Value); err != nil {
				return nil, fmt.Errorf("unsupported quoted YAML escape: %w", err)
			}
		} else {
			n.Value = strings.ReplaceAll(s[1:finish-1], "''", "'")
		}
		return n, nil
	}
	if i := strings.Index(s, " #"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if strings.Contains(s, ": ") || (!flow && strings.ContainsAny(s, "[]{}")) {
		return nil, fmt.Errorf("ambiguous/unsupported plain YAML scalar")
	}
	n.Value = s
	switch s {
	case "true", "false":
		n.Tag = "!!bool"
	case "null", "~":
		n.Tag = "!!null"
		n.Value = ""
	default:
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			n.Tag = "!!float"
		}
	}
	return n, nil
}

func (p *parser) literal(a, parentIndent int, strip bool) (*Node, error) {
	n := p.node(ScalarNode, a)
	n.Tag = "!!str"
	n.Style = LiteralStyle
	p.row++
	start := p.row
	indent := -1
	for p.row < len(p.lines) {
		l := p.lines[p.row]
		if strings.TrimSpace(l.text) != "" {
			if l.indent <= parentIndent {
				break
			}
			if indent < 0 {
				indent = l.indent
			}
			if l.indent < indent {
				return nil, fmt.Errorf("invalid literal scalar indentation")
			}
		}
		p.row++
	}
	var b strings.Builder
	for i := start; i < p.row; i++ {
		line := p.lines[i].text
		if strings.TrimSpace(line) != "" {
			b.WriteString(line[indent:])
		}
		if p.lines[i].offset+len(line) < len(p.raw) {
			b.WriteByte('\n')
		}
	}
	n.Value = strings.TrimRight(b.String(), "\n")
	if !strip && b.Len() > 0 {
		n.Value += "\n"
	}
	return n, nil
}

func (p *parser) flow(a, depth int) (*Node, int, error) {
	if depth > 64 {
		return nil, 0, fmt.Errorf("YAML nesting limit")
	}
	if a >= len(p.raw) {
		return nil, 0, fmt.Errorf("missing flow value")
	}
	kind, close := SequenceNode, byte(']')
	if p.raw[a] == '{' {
		kind, close = MappingNode, '}'
	} else if p.raw[a] != '[' {
		return nil, 0, fmt.Errorf("expected flow container")
	}
	n := p.node(kind, a)
	n.Style = FlowStyle
	pos := a + 1
	space := func() {
		for pos < len(p.raw) {
			if strings.ContainsRune(" \n", rune(p.raw[pos])) {
				pos++
				continue
			}
			if p.raw[pos] == '#' {
				for pos < len(p.raw) && p.raw[pos] != '\n' {
					pos++
				}
				continue
			}
			break
		}
	}
	for {
		space()
		if pos >= len(p.raw) {
			return nil, 0, fmt.Errorf("unterminated flow YAML")
		}
		if p.raw[pos] == close {
			return n, pos + 1, nil
		}
		if kind == MappingNode {
			start := pos
			for pos < len(p.raw) && p.raw[pos] != ':' && p.raw[pos] != close {
				pos++
			}
			if pos == len(p.raw) || p.raw[pos] != ':' {
				return nil, 0, fmt.Errorf("invalid flow mapping key")
			}
			key := strings.TrimSpace(p.raw[start:pos])
			if !validKey(key) {
				return nil, 0, fmt.Errorf("unsupported flow key")
			}
			for i := 0; i < len(n.Content); i += 2 {
				if n.Content[i].Value == key {
					return nil, 0, fmt.Errorf("duplicate flow key")
				}
			}
			k := p.node(ScalarNode, start)
			k.Tag = "!!str"
			k.Value = key
			n.Content = append(n.Content, k)
			pos++
			space()
		}
		var value *Node
		var err error
		if pos < len(p.raw) && (p.raw[pos] == '[' || p.raw[pos] == '{') {
			value, pos, err = p.flow(pos, depth+1)
		} else {
			start := pos
			quote := byte(0)
			for pos < len(p.raw) {
				c := p.raw[pos]
				if quote != 0 {
					if quote == '"' && c == '\\' {
						pos += 2
						continue
					}
					if c == quote {
						if quote == '\'' && pos+1 < len(p.raw) && p.raw[pos+1] == quote {
							pos += 2
							continue
						}
						quote = 0
					}
					pos++
					continue
				}
				if c == '"' || c == '\'' {
					quote = c
					pos++
					continue
				}
				if c == ',' || c == ']' || c == '}' || c == '\n' {
					break
				}
				pos++
			}
			if quote != 0 {
				return nil, 0, fmt.Errorf("unterminated flow scalar")
			}
			value, err = p.scalar(start, pos, true)
		}
		if err != nil {
			return nil, 0, err
		}
		n.Content = append(n.Content, value)
		space()
		if pos >= len(p.raw) {
			return nil, 0, fmt.Errorf("unterminated flow YAML")
		}
		if p.raw[pos] == close {
			return n, pos + 1, nil
		}
		if p.raw[pos] != ',' {
			return nil, 0, fmt.Errorf("flow YAML requires comma")
		}
		pos++
	}
}

func (p *parser) consumeFlowTail(end int) error {
	row := 0
	for row+1 < len(p.lines) && p.lines[row+1].offset < end {
		row++
	}
	lineEnd := p.lines[row].offset + len(p.lines[row].text)
	tail := strings.TrimSpace(p.raw[end:lineEnd])
	if tail != "" && !strings.HasPrefix(tail, "#") {
		return fmt.Errorf("unsupported flow YAML tail")
	}
	p.row = row + 1
	return nil
}
