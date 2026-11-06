package sitecontent

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
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
