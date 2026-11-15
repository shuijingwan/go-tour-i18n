package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"golang.org/x/net/html"
)

// InspectStructuralWhitespace traces exact source-token expansions. It does
// not protect target prose, infer translations or change any representation.
func (a PageAdapter) InspectStructuralWhitespace(raw []byte) ([]i18n.StructuralWhitespaceRepair, string, error) {
	d, err := ParseWorkflowDocument(documentSource(a.Document), a.Source, a.Document.StableIndex)
	if err != nil || !reflect.DeepEqual(d.Units, a.Document.Units) || !reflect.DeepEqual(d.Slots, a.Document.Slots) {
		return nil, "", fmt.Errorf("protected whitespace requires exact canonical source mapping")
	}
	restored, err := a.Restore(string(raw))
	if err != nil {
		return nil, "", err
	}
	pieces, err := protectionPieces(a.Document.Units[0])
	if err != nil {
		return nil, "", err
	}
	byToken := map[string]protectedPiece{}
	for _, p := range pieces {
		if p.token != nil {
			byToken[p.token.Token] = p
		}
	}
	tokens := append([]ProtectedToken(nil), a.Document.Units[0].Protected...)
	sort.Slice(tokens, func(i, j int) bool {
		return strings.Index(string(raw), tokens[i].Token) < strings.Index(string(raw), tokens[j].Token)
	})
	var expanded strings.Builder
	origins := []int{}
	starts := make([]int, len(tokens))
	pos := 0
	for i, token := range tokens {
		at := strings.Index(string(raw), token.Token)
		if at < pos {
			return nil, "", fmt.Errorf("protected whitespace token order/identity mismatch")
		}
		expanded.Write(raw[pos:at])
		for range at - pos {
			origins = append(origins, -1)
		}
		starts[i] = expanded.Len()
		expanded.WriteString(token.Raw)
		for range len(token.Raw) {
			origins = append(origins, i)
		}
		pos = at + len(token.Token)
	}
	expanded.Write(raw[pos:])
	for range len(raw) - pos {
		origins = append(origins, -1)
	}
	candidate := bridgeEOF(restored)
	if !bytes.Equal(bridgeEOF([]byte(expanded.String())), candidate) {
		return nil, "", fmt.Errorf("protected whitespace expansion is not exact original restore")
	}
	// Whitespace-preserving HTML needs a separate semantic proof.
	styled := false
	z := html.NewTokenizer(bytes.NewReader(a.Source))
	for typ := z.Next(); typ != html.ErrorToken; typ = z.Next() {
		if typ == html.StartTagToken || typ == html.SelfClosingTagToken {
			for _, attr := range z.Token().Attr {
				styled = styled || attr.Key == "style" || attr.Key == "xml:space"
			}
		}
	}
	var repairs []i18n.StructuralWhitespaceRepair
	offset := 0
	for li, line := range bytes.Split(candidate, []byte{'\n'}) {
		end := len(line)
		start := end
		for start > 0 && (line[start-1] == ' ' || line[start-1] == '\t' || line[start-1] == '\r') {
			start--
		}
		if start != end {
			r := i18n.StructuralWhitespaceRepair{Line: li + 1, Start: offset + start, End: offset + end, Whitespace: string(line[start:]), OpaqueLine: start == 0}
			for at := r.Start; at < r.End; {
				if origins[at] < 0 {
					r.OpaqueLine = false
					at++
					continue
				}
				i := origins[at]
				stop := at + 1
				for stop < r.End && origins[stop] == i {
					stop++
				}
				token, p := tokens[i], byToken[tokens[i].Token]
				owner, unique := bridgeFormattingOwner(a.Document, p)
				slotIndex := -1
				for si, slot := range a.Document.Slots {
					if slot.Start == owner.Start && slot.End == owner.End {
						slotIndex = si
					}
				}
				_, locator, _ := strings.Cut(owner.Locator, "#")
				context, _, _ := strings.Cut(locator, ":")
				prose := false
				switch context {
				case "markdown/Paragraph", "markdown/TextBlock", "markdown/list-item", "html/p", "html/li", "html/td", "html/th", "html/h1", "html/h2", "html/h3", "html/h4", "html/h5", "html/h6", "html/text-block":
					prose = true
				}
				// Keep LF opaque. Only the indentation after that LF belongs to
				// this repair; never a hard break, pair, code or boundary payload.
				if !unique || !prose || styled || token.Role != "atom" || token.Binding != nil || token.Pair != fmt.Sprintf("slot%d/", slotIndex) || !bridgeFormattingWhitespace(token.Raw) || at-starts[i] < 1 {
					return nil, "", fmt.Errorf("representation/recovery blocker: trailing whitespace at line %d is not proved prose formatting (%s)", li+1, token.Token)
				}
				r.Origins = append(r.Origins, i18n.StructuralWhitespaceOrigin{Token: token.Token, RawSHA256: digest([]byte(token.Raw)), SourceStart: p.start, SourceEnd: p.end, TokenStart: at - starts[i], TokenEnd: stop - starts[i], Owner: owner.Locator})
				at = stop
			}
			repairs = append(repairs, r)
		}
		offset += len(line) + 1
	}
	fingerprint := identity(struct {
		Source       string
		Units, Slots []Unit
	}{digest(a.Source), a.Document.Units, a.Document.Slots})
	return repairs, fingerprint, nil
}
