package sitecontent

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"golang.org/x/net/html"
)

// BridgeRestrictedRevision composes source-owned representations, never
// re-protects restored target prose. Its caller must verify the original passed
// ordinary raw, the restricted invocation and exact QC authority first.
func (a PageAdapter) BridgeRestrictedRevision(ordinaryRaw, old, reviewed, restrictedRaw []byte, p *i18n.CommentRevisionTransport) (i18n.TranslationUnitAdapter, []byte, error) {
	if p == nil || p.Schema != "go-learning/comment-revision-transport/v1" || p.UnitID != a.Document.Units[0].ID || p.SourceSHA256 != digest(a.Source) || len(p.Boundaries) != len(p.Slots)+1 {
		return nil, nil, fmt.Errorf("bridge restricted identity/shape mismatch")
	}
	before, err := a.Restore(string(ordinaryRaw))
	if err != nil || !bytes.Equal(bridgeEOF(before), bridgeEOF(old)) {
		return nil, nil, fmt.Errorf("bridge ordinary ancestor mismatch: %v", err)
	}
	current, err := ParseWorkflowDocument(documentSource(a.Document), a.Source, a.Document.StableIndex)
	if err != nil {
		return nil, nil, err
	}
	a, ordinaryRaw, err = a.bridgeReleaseNaturalAtoms(PageAdapter{*current, a.Source}, ordinaryRaw)
	if err != nil {
		return nil, nil, err
	}
	sourceBodies, err := a.CommentRevisionBodies(a.Source)
	if err != nil {
		return nil, nil, err
	}
	oldBodies, err := a.CommentRevisionBodies(old)
	if err != nil {
		return nil, nil, err
	}
	targetBodies, err := a.CommentRevisionBodies(reviewed)
	if err != nil {
		return nil, nil, err
	}
	key := func(b i18n.CommentRevisionBody) string { return fmt.Sprintf("%s/%d", b.Coordinate, b.Position) }
	sourceMap, targetMap := map[string]i18n.CommentRevisionBody{}, map[string]i18n.CommentRevisionBody{}
	for _, b := range sourceBodies {
		sourceMap[key(b)] = b
	}
	for _, b := range targetBodies {
		targetMap[key(b)] = b
	}
	type patch struct {
		start, end int
		text       []byte
	}
	var patches []patch
	editable := map[int]string{}
	lastEnd := 0
	for pi, slot := range p.Slots {
		if slot.Start < lastEnd || slot.Start >= slot.End || slot.End > len(old) {
			return nil, nil, fmt.Errorf("bridge slot order/duplicate")
		}
		lastEnd = slot.End
		var b i18n.CommentRevisionBody
		found := false
		for _, q := range oldBodies {
			if q.Start == slot.Start && q.End == slot.End {
				if found {
					return nil, nil, fmt.Errorf("ambiguous bridge coordinate")
				}
				b, found = q, true
			}
		}
		s, ok := sourceMap[key(b)]
		target, ok2 := targetMap[key(b)]
		if !found || !ok || !ok2 {
			return nil, nil, fmt.Errorf("bridge comment is not source-owned")
		}
		ordinarySlot := -1
		for i, u := range a.Document.Slots {
			if u.Start == s.Start && u.End == s.End && strings.Contains(u.Locator, "#teaching-comment:") {
				ordinarySlot = i
			}
		}
		if ordinarySlot < 0 {
			patches = append(patches, patch{s.Start, s.End, bytes.Clone(reviewed[target.Start:target.End])})
			continue
		}
		// Already exposed teaching slots keep the exact token mapping from the
		// passed restricted raw. No lexical search in translated text is permitted.
		body, ok := b.Adapter.(revisionCommentBody)
		if !ok {
			return nil, nil, fmt.Errorf("unknown teaching protector")
		}
		u := a.Document.Slots[ordinarySlot]
		if len(body.unit.Protected) != len(u.Protected) || len(slot.Tokens) != len(u.Protected) {
			return nil, nil, fmt.Errorf("teaching source token exact-set mismatch")
		}
		start := strings.Index(string(restrictedRaw), p.Boundaries[pi]) + len(p.Boundaries[pi])
		end := strings.Index(string(restrictedRaw), p.Boundaries[pi+1])
		if start < 0 || end < start {
			return nil, nil, fmt.Errorf("restricted boundary correspondence")
		}
		value := string(restrictedRaw[start:end])
		pairs := []string{}
		for j, q := range u.Protected {
			oldToken := body.unit.Protected[j]
			if q.Raw != oldToken.Raw || q.Role != oldToken.Role || q.Pair != oldToken.Pair || strings.Count(value, slot.Tokens[j]) != 1 {
				return nil, nil, fmt.Errorf("teaching source token identity changed")
			}
			pairs = append(pairs, slot.Tokens[j], q.Token)
		}
		if len(pairs) > 0 {
			value = strings.NewReplacer(pairs...).Replace(value)
		}
		editable[ordinarySlot] = value
	}
	scaffold := bytes.Clone(a.Source)
	for i := len(patches) - 1; i >= 0; i-- {
		q := patches[i]
		if q.start < 0 || q.end > len(a.Source) || q.start >= q.end || (i > 0 && patches[i-1].end > q.start) {
			return nil, nil, fmt.Errorf("bridge source patch overlap")
		}
		scaffold = append(append(append([]byte{}, scaffold[:q.start]...), q.text...), scaffold[q.end:]...)
	}
	// Retain source-selected internal slots. Only opaque comment payloads enter
	// this private response scaffold; ordinary source prose is unchanged.
	if len(a.Document.ReferenceDefinitions) > 0 && len(patches) > 0 {
		return nil, nil, fmt.Errorf("bridge reference-definition offset proof unavailable")
	}
	doc := a.Document
	doc.SourceSHA = digest(scaffold)
	doc.Units = append([]Unit(nil), doc.Slots...)
	shift := func(at int) int {
		n := at
		for _, q := range patches {
			if q.end <= at {
				n += len(q.text) - (q.end - q.start)
			}
		}
		return n
	}
	for i, s := range doc.Units {
		for _, q := range patches {
			if q.start < s.End && q.end > s.Start {
				return nil, nil, fmt.Errorf("opaque bridge patch overlaps ordinary prose")
			}
		}
		doc.Units[i].Start, doc.Units[i].End = shift(s.Start), shift(s.End)
	}
	formal, err := FormalPage(doc, scaffold, a.Document.StableIndex)
	if err != nil {
		return nil, nil, err
	}
	next := PageAdapter{formal, scaffold}
	boundaries := func(d Document) []ProtectedToken {
		var out []ProtectedToken
		for _, p := range d.Units[0].Protected {
			if p.Role == "boundary" {
				out = append(out, p)
			}
		}
		return out
	}
	ob, nb := boundaries(a.Document), boundaries(formal)
	if len(ob) != len(nb) || len(ob) != len(doc.Units)+1 {
		return nil, nil, fmt.Errorf("bridge normal boundary exact-set mismatch")
	}
	var raw strings.Builder
	for i, s := range a.Document.Slots {
		start := strings.Index(string(ordinaryRaw), ob[i].Token) + len(ob[i].Token)
		end := strings.Index(string(ordinaryRaw), ob[i+1].Token)
		if start > end {
			return nil, nil, fmt.Errorf("ordinary boundary order")
		}
		value := string(ordinaryRaw[start:end])
		local := false
		if v, ok := editable[i]; ok {
			value, local = v, true
		}
		oa, na := 0, 0
		for j, q := range a.Document.Units[0].Protected {
			if q.Token == ob[i].Token {
				oa = j + 1
				break
			}
		}
		for j, q := range formal.Units[0].Protected {
			if q.Token == nb[i].Token {
				na = j + 1
				break
			}
		}
		pairs := []string{}
		for j, q := range s.Protected {
			oldToken := a.Document.Units[0].Protected[oa+j].Token
			if local {
				oldToken = q.Token
			}
			pairs = append(pairs, oldToken, formal.Units[0].Protected[na+j].Token)
		}
		if len(pairs) > 0 {
			value = strings.NewReplacer(pairs...).Replace(value)
		}
		raw.WriteString(nb[i].Token)
		raw.WriteString(value)
	}
	raw.WriteString(nb[len(nb)-1].Token)
	result := bridgeEOF([]byte(raw.String()))
	got, err := next.Restore(string(result))
	if err != nil || !bytes.Equal(bridgeEOF(got), bridgeEOF(reviewed)) {
		return nil, nil, fmt.Errorf("bridge exact reviewed restore failed: %v", err)
	}
	return next, result, nil
}
func bridgeEOF(b []byte) []byte { return append(bytes.TrimRight(b, "\n"), '\n') }

// BridgeResponseFormattingWhitespace is used only after a bridge consumer has
// proved the sealed ordinary decoder and its exact reviewed response. Parsing
// this source-owned private scaffold never parses or protects translated prose.
func (old PageAdapter) BridgeResponseFormattingWhitespace(raw []byte) (i18n.TranslationUnitAdapter, []byte, error) {
	d, err := ParseWorkflowDocument(documentSource(old.Document), old.Source, old.Document.StableIndex)
	if err != nil {
		return nil, nil, err
	}
	return old.bridgeReleaseAtoms(PageAdapter{*d, old.Source}, raw, false)
}

func bridgeFormattingWhitespace(s string) bool {
	if s == "" || s[0] != '\n' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return false
		}
	}
	return true
}

// Exact HTML text-token coordinates, not a whitespace spelling heuristic,
// distinguish prose soft wraps from tag/attribute/code/directive boundaries.
// Markdown (including hard breaks and indentation) is deliberately ineligible.
func bridgeProseFormattingSpans(s Unit, source []byte) (map[[2]int]string, error) {
	allowed := map[[2]int]string{}
	_, locator, _ := strings.Cut(s.Locator, "#")
	context, _, _ := strings.Cut(locator, ":")
	switch context {
	case "html/p", "html/li", "html/td", "html/th", "html/h1", "html/h2", "html/h3", "html/h4", "html/h5", "html/h6", "html/text-block":
	default:
		return allowed, nil
	}
	if s.Start < 0 || s.End > len(source) || s.Start >= s.End || !hasLanguage(s.Source) {
		return nil, fmt.Errorf("bridge formatting slot identity mismatch")
	}
	text, err := restoreBlock(s, s.Source)
	if err != nil || text != string(source[s.Start:s.End]) {
		return nil, fmt.Errorf("bridge formatting source slot mismatch")
	}
	pieces, err := protectionPieces(s)
	if err != nil {
		return nil, err
	}
	// Explicit whitespace styling/space preservation needs a different proof.
	// Do not infer CSS behavior or silently treat that source as normal flow.
	z := html.NewTokenizer(bytes.NewReader(source))
	for typ := z.Next(); typ != html.ErrorToken; typ = z.Next() {
		if typ == html.StartTagToken || typ == html.SelfClosingTagToken {
			for _, attr := range z.Token().Attr {
				if attr.Key == "style" || attr.Key == "xml:space" {
					return allowed, nil
				}
			}
		}
	}
	var spans []span
	if err := htmlSpans(source, 0, "html", func(a, z int, context string) {
		if context == "html" { // Never a visible attribute, pre/code or script.
			spans = append(spans, span{start: a, end: z})
		}
	}); err != nil {
		return nil, err
	}
	for _, p := range pieces {
		value := p.text
		if p.token != nil {
			if p.token.Role != "atom" || p.token.Binding != nil || p.token.Pair != "" || !bridgeFormattingWhitespace(p.token.Raw) {
				continue
			}
			value = p.token.Raw
		}
		for i := 0; i < len(value); i++ {
			if value[i] != '\n' {
				continue
			}
			end := i + 1
			for end < len(value) && (value[end] == ' ' || value[end] == '\t') {
				end++
			}
			a, z := s.Start+p.start+i, s.Start+p.start+end
			for _, sp := range spans {
				if a <= sp.start || z >= sp.end {
					continue
				}
				before, after := false, false
				for _, q := range pieces {
					if q.token != nil {
						continue
					}
					left, right := max(s.Start+q.start, sp.start), min(s.Start+q.end, sp.end)
					if left < min(right, a) {
						before = before || hasLanguage(string(source[left:min(right, a)]))
					}
					if max(left, z) < right {
						after = after || hasLanguage(string(source[max(left, z):right]))
					}
				}
				if before && after {
					allowed[[2]int{a, z}] = value[i:end]
				}
			}
			i = end - 1
		}
	}
	return allowed, nil
}

// Only source-exposed, proved HTML formatting sequences may remain in this
// private response slot. They may move or be deleted, never be invented or
// duplicated. The returned string is syntax-check scratch, not candidate data.
func bridgeFormattingSyntax(s Unit, target, plain string, source []byte) (string, error) {
	allowed, err := bridgeProseFormattingSpans(s, source)
	if err != nil {
		return "", err
	}
	pieces, err := protectionPieces(s)
	if err != nil {
		return "", err
	}
	counts := map[string]int{}
	for coords, value := range allowed {
		for _, p := range pieces {
			if p.token == nil && coords[0] >= s.Start+p.start && coords[1] <= s.Start+p.end {
				counts[value]++
			}
		}
	}
	for _, p := range s.Protected {
		if p.Role != "open" || !strings.HasPrefix(p.Pair, "html-attribute/") {
			continue
		}
		for _, q := range s.Protected {
			if q.Role == "close" && q.Pair == p.Pair && strings.ContainsRune(target[strings.Index(target, p.Token)+len(p.Token):strings.Index(target, q.Token)], '\n') {
				return "", fmt.Errorf("bridge formatting crossed attribute boundary")
			}
		}
	}
	var checked strings.Builder
	for i := 0; i < len(plain); {
		if plain[i] != '\n' {
			checked.WriteByte(plain[i])
			i++
			continue
		}
		end := i + 1
		for end < len(plain) && (plain[end] == ' ' || plain[end] == '\t') {
			end++
		}
		value := plain[i:end]
		if counts[value] == 0 {
			return "", fmt.Errorf("new unprotected line boundary")
		}
		counts[value]--
		checked.WriteByte(' ')
		i = end
	}
	return checked.String(), nil
}

// A formatting atom must belong to exactly one complete slot in both source
// representations. A partial overlap or duplicate owner cannot release it.
func bridgeFormattingOwner(d Document, p protectedPiece) (Unit, bool) {
	var owner Unit
	count := 0
	for _, s := range d.Slots {
		if s.Start < p.end && s.End > p.start {
			if s.Start > p.start || s.End < p.end {
				return Unit{}, false
			}
			owner, count = s, count+1
		}
	}
	return owner, count == 1
}

// Release only historical atoms whose exact canonical-source interval is now
// natural-language prose. Keep all other historical protection (including an
// approved opaque parent); never parse or protect translated candidate bytes.
// This private response Page does not replace the caller's formal Unit/input.
func (old PageAdapter) bridgeReleaseNaturalAtoms(current PageAdapter, raw []byte) (PageAdapter, []byte, error) {
	return old.bridgeReleaseAtoms(current, raw, true)
}

func (old PageAdapter) bridgeReleaseAtoms(current PageAdapter, raw []byte, natural bool) (PageAdapter, []byte, error) {
	if !bytes.Equal(old.Source, current.Source) || len(old.Document.Units) != 1 || len(current.Document.Units) != 1 {
		return PageAdapter{}, nil, fmt.Errorf("bridge atom release source mismatch")
	}
	prior, err := protectionPieces(old.Document.Units[0])
	if err != nil {
		return PageAdapter{}, nil, err
	}
	now, err := protectionPieces(current.Document.Units[0])
	if err != nil {
		return PageAdapter{}, nil, err
	}
	released := map[[2]int]bool{}
	for _, p := range prior {
		if p.token == nil || p.token.Role != "atom" || p.token.Binding != nil {
			continue
		}
		if bridgeFormattingWhitespace(p.token.Raw) {
			before, ok := bridgeFormattingOwner(old.Document, p)
			after, ok2 := bridgeFormattingOwner(current.Document, p)
			if !ok || !ok2 || before.Start != after.Start || before.End != after.End {
				continue
			}
			for _, s := range []Unit{before, after} {
				spans, err := bridgeProseFormattingSpans(s, old.Source)
				if err != nil {
					return PageAdapter{}, nil, err
				}
				if spans[[2]int{p.start, p.end}] != p.token.Raw {
					ok = false
				}
			}
			if ok {
				released[[2]int{p.start, p.end}] = true
			}
			continue
		}
		if !natural || !hasLanguage(p.token.Raw) {
			continue
		}
		for _, q := range now {
			if q.token == nil && p.start >= q.start && p.end <= q.end && p.start < p.end && q.text[p.start-q.start:p.end-q.start] == p.token.Raw && classifyProseToken(p.token.Raw, string(old.Source[max(0, p.start-96):p.start]), string(old.Source[p.end:min(len(old.Source), p.end+48)])) == naturalLanguage {
				released[[2]int{p.start, p.end}] = true
			}
		}
	}
	if len(released) == 0 {
		return old, raw, nil
	}
	doc := old.Document
	doc.Units = append([]Unit(nil), doc.Slots...)
	for i, slot := range doc.Units {
		pieces, err := protectionPieces(slot)
		if err != nil {
			return PageAdapter{}, nil, err
		}
		remove := map[string]bool{}
		for _, p := range pieces {
			if p.token != nil && released[[2]int{slot.Start + p.start, slot.Start + p.end}] {
				slot.Source = strings.Replace(slot.Source, p.token.Token, p.token.Raw, 1)
				remove[p.token.Token] = true
			}
		}
		slot.Protected = nil
		for _, p := range doc.Units[i].Protected {
			if !remove[p.Token] {
				slot.Protected = append(slot.Protected, p)
			}
		}
		slot.SourceSHA = unitSourceIdentity(slot.Source, slot.Protected)
		doc.Units[i] = slot
	}
	doc, err = FormalPage(doc, old.Source, old.Document.StableIndex)
	if err != nil {
		return PageAdapter{}, nil, err
	}
	next := PageAdapter{doc, old.Source}
	projected, err := next.RebaseRevisionPredecessor(old, raw)
	return next, projected, err
}

// Package metadata is source authority, distinct from language history. These
// are the existing shared Site Page source/registry references, never locale
// target bytes or a guessed route allowlist.
func (a PageAdapter) BridgeContextReferences(root string) ([]i18n.GlossaryArchiveReference, error) {
	refs := []i18n.GlossaryArchiveReference{}
	for _, path := range []string{SourceManifestPath, GlobalPath, PageRegistryPath} {
		ref, err := ReferenceFile(root, path)
		if err != nil {
			return nil, err
		}
		refs = append(refs, i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256})
	}
	return refs, nil
}
