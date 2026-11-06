package sitecontent

import (
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	gmtext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"sort"
	"strings"
)

// Record the delimiters matched by Goldmark's CommonMark algorithm, including
// flanking, intraword underscores, nesting, escapes and code/link precedence.
// Only matched markup terminates a lexical word; foo_bar remains one atom.
func emphasisDelimiters(b []byte) map[int]bool {
	return matchedEmphasis(b).positions
}

func matchedEmphasis(b []byte) *emphasisPositions {
	p := &emphasisPositions{positions: map[int]bool{}}
	goldmark.New(goldmark.WithParserOptions(parser.WithInlineParsers(util.Prioritized(p, 499)))).Parser().Parse(gmtext.NewReader(b))
	return p
}

type emphasisPositions struct {
	positions      map[int]bool
	opener, closer *parser.Delimiter
	matches        []emphasisMatch
}

type emphasisMatch struct{ open, close int }

func (p *emphasisPositions) Trigger() []byte         { return []byte{'*', '_'} }
func (p *emphasisPositions) IsDelimiter(b byte) bool { return b == '*' || b == '_' }
func (p *emphasisPositions) CanOpenCloser(a, z *parser.Delimiter) bool {
	p.opener, p.closer = a, z
	return a.Char == z.Char
}
func (p *emphasisPositions) OnMatch(consumes int) ast.Node {
	p.matches = append(p.matches, emphasisMatch{p.opener.Segment.Stop, p.closer.Segment.Stop})
	// Goldmark has consumed the matched characters from both delimiter runs.
	for _, d := range []*parser.Delimiter{p.opener, p.closer} {
		for i := d.Segment.Stop; i < d.Segment.Stop+consumes; i++ {
			p.positions[i] = true
		}
	}
	return ast.NewEmphasis(consumes)
}

// Source syntax establishes emphasis bindings. Target-language flanking rules
// do not reclassify those delimiters, but their pairing/nesting stays strict.
func validateEmphasisBindings(s Unit, target string) error {
	source, err := restoreBlock(s, s.Source)
	if err != nil {
		return err
	}
	type protectedRange struct {
		a, z  int
		token string
	}
	ranges := []protectedRange{}
	pos, rawPos := 0, 0
	for _, p := range s.Protected {
		a := strings.Index(s.Source, p.Token)
		rawPos += a - pos
		ranges = append(ranges, protectedRange{rawPos, rawPos + len(p.Raw), p.Token})
		rawPos += len(p.Raw)
		pos = a + len(p.Token)
	}
	tokenAt := func(offset int) string {
		for _, r := range ranges {
			if offset >= r.a && offset < r.z {
				return r.token
			}
		}
		return ""
	}
	type interval struct{ a, z int }
	intervals := []interval{}
	for _, m := range matchedEmphasis([]byte(source)).matches {
		a, z := tokenAt(m.open), tokenAt(m.close)
		if a == "" || z == "" {
			return fmt.Errorf("source emphasis delimiter is not protected")
		}
		x, y := strings.Index(target, a), strings.Index(target, z)
		if x > y {
			return fmt.Errorf("emphasis binding reversed")
		}
		intervals = append(intervals, interval{x, y})
	}
	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i].a == intervals[j].a {
			return intervals[i].z > intervals[j].z
		}
		return intervals[i].a < intervals[j].a
	})
	stack := []int{}
	for _, v := range intervals {
		for len(stack) > 0 && v.a > stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 && v.z > stack[len(stack)-1] {
			return fmt.Errorf("emphasis bindings crossed")
		}
		stack = append(stack, v.z)
	}
	return nil
}
func (p *emphasisPositions) Parse(parent ast.Node, reader gmtext.Reader, pc parser.Context) ast.Node {
	line, segment := reader.PeekLine()
	d := parser.ScanDelimiter(line, reader.PrecendingCharacter(), 1, p)
	if d == nil {
		return nil
	}
	d.Segment = segment.WithStop(segment.Start + d.OriginalLength)
	reader.Advance(d.OriginalLength)
	pc.PushDelimiter(d)
	return d
}
