package sitecontent

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

// Frozen source foundations retain ExtractShellLanguageSource. A newly
// authorized Page representation groups complete inline runs using the shared
// HTML protector; no structured asset adopts these slots implicitly.
func ExtractShellPageRepresentation(path string, raw []byte) (*ShellLanguageSource, error) {
	s, err := extractShellLanguageSource(path, raw, true)
	if err != nil || s.Page == nil {
		return s, err
	}
	masked := bytes.Clone(raw)
	ranges, err := shellTemplateRanges(raw)
	if err != nil {
		return nil, err
	}
	for _, r := range ranges {
		for i := r.start; i < r.end; i++ {
			if masked[i] != '\n' {
				masked[i] = 'x'
			}
		}
	}
	offset := bytes.Index(raw[4:], []byte("\n---\n")) + 9
	groups := []span{}
	zr := html.NewTokenizer(bytes.NewReader(masked[offset:]))
	pos, start := offset, offset
	flush := func(end int) {
		if start < end {
			groups = append(groups, span{start, end, "html/inline-run"})
		}
	}
	for {
		typ := zr.Next()
		a := pos
		pos += len(zr.Raw())
		if typ == html.ErrorToken {
			flush(pos)
			break
		}
		if typ == html.StartTagToken || typ == html.EndTagToken || typ == html.SelfClosingTagToken {
			name, _ := zr.TagName()
			if htmlBoundaryTag(string(name)) {
				flush(a)
				start = pos
			}
		} else if typ == html.CommentToken {
			flush(a)
			start = pos
		}
	}
	e := &shellExtractor{source: *s, raw: raw, occurrences: map[string]int{}}
	e.source.Slots = nil
	used := map[int]bool{}
	for _, g := range groups {
		a, z := trimShellSpan(raw, g.start, g.end)
		if a == z {
			continue
		}
		members := []int{}
		eligible := true
		for i, slot := range s.Slots {
			u := slot.Unit
			if u.Start < z && u.End > a {
				if u.Start < a || u.End > z || (slot.Encoding != "html-text" && slot.Encoding != "html-attribute") {
					eligible = false
					break
				}
				members = append(members, i)
			}
		}
		if !eligible || len(members) == 0 || !bytes.Contains(raw[a:z], []byte("<")) {
			continue
		}
		text, locked, err := protectBlockMode(raw[a:z], g.context, true)
		if err != nil {
			// Inline runs cut by a block boundary (e.g. an anchor wrapping a
			// div) retain the frozen block framework. They are not movable
			// inline children. The original extractor already validated them.
			continue
		}
		if err = e.protectedSlot(a, z, text, locked, "inline-prose", g.context, "html-text"); err != nil {
			return nil, err
		}
		for _, i := range members {
			used[i] = true
		}
	}
	for i, slot := range s.Slots {
		if !used[i] {
			e.source.Slots = append(e.source.Slots, slot)
		}
	}
	return e.finish()
}

// Rebind only parser-established structure, never infer machine ownership from
// target spelling. Identical repeated structures are interchangeable; distinct
// destinations retain their paired source identity even when reordered.
func bindInlineShellTokens(u Unit, value string) (string, error) {
	text, tokens, err := protectBlockMode([]byte(value), "html/inline-run", false)
	if err != nil {
		return "", err
	}
	structural := []ProtectedToken{}
	prebound := map[string]bool{}
	// Source coordinates preserve hidden icon/text roles and repeated raws.
	// Parsing individual raw atoms would discard their enclosing DOM context.
	original, err := restoreBlock(u, u.Source)
	if err != nil {
		return "", err
	}
	sourceText, syntax, err := protectBlockMode([]byte(original), "html/inline-run", false)
	if err != nil {
		return "", err
	}
	sp, err := protectionPieces(Unit{Source: sourceText, Protected: syntax})
	if err != nil {
		return "", err
	}
	up, err := protectionPieces(u)
	if err != nil {
		return "", err
	}
	for _, p := range up {
		if p.token == nil {
			continue
		}
		for _, q := range sp {
			if q.token != nil && p.start == q.start && p.end == q.end && p.token.Raw == q.token.Raw && p.token.Role == q.token.Role {
				structural = append(structural, *p.token)
				prebound[p.token.Token] = true
				break
			}
		}
	}
	if protectionIdentity(tokens) != protectionIdentity(structural) {
		return "", fmt.Errorf("inline machine identity/binding changed")
	}
	key := func(ps []ProtectedToken) map[string][]ProtectedToken {
		opens := map[string]string{}
		for i, p := range ps {
			if p.Role == "open" {
				opens[p.Pair] = p.Raw
				if p.Raw == "" && i+1 < len(ps) && strings.HasPrefix(ps[i+1].Pair, "html-attribute/") {
					// Empty outer openers own a split start tag. Include its
					// machine framing so two titled links follow their URLs.
					for _, q := range ps[i+1:] {
						if strings.HasPrefix(q.Pair, "html-attribute/") {
							opens[p.Pair] += q.Raw
							if q.Role == "open" {
								opens[p.Pair] += "\x00attribute\x00"
							}
							if q.Role == "close" && strings.HasSuffix(q.Raw, ">") {
								break
							}
						}
					}
				}
			}
		}
		out := map[string][]ProtectedToken{}
		for _, p := range ps {
			k := p.Role + "\x00" + p.Raw
			if p.Role == "close" {
				k += "\x00" + opens[p.Pair]
			}
			out[k] = append(out[k], p)
		}
		return out
	}
	source, target := key(structural), key(tokens)
	pairs := []string{}
	// Assign whole pairs by their closing identity first; their opener follows.
	mapped := map[string]bool{}
	for k, ts := range target {
		ss := source[k]
		if len(ts) != len(ss) {
			return "", fmt.Errorf("inline token inventory changed")
		}
		if len(ts) == 0 || ts[0].Role != "close" {
			continue
		}
		for i, p := range ts {
			q := ss[i]
			pairs = append(pairs, p.Token, q.Token)
			mapped[p.Token] = true
			for _, op := range tokens {
				if op.Role == "open" && op.Pair == p.Pair {
					for _, oq := range u.Protected {
						if oq.Role == "open" && oq.Pair == q.Pair {
							pairs = append(pairs, op.Token, oq.Token)
							mapped[op.Token] = true
						}
					}
				}
			}
		}
	}
	keys := []string{}
	for k := range target {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for i, p := range target[k] {
			if !mapped[p.Token] {
				pairs = append(pairs, p.Token, source[k][i].Token)
			}
		}
	}
	if len(pairs) > 0 {
		text = strings.NewReplacer(pairs...).Replace(text)
	}
	return bindFrozenShellTokensMode(u, text, prebound)
}
