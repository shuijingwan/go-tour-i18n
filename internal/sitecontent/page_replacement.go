package sitecontent

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Frozen representations are explicit inputs, never discovered by latest/path
// guessing. Their protected IDs and source bytes are checked again against the
// immutable predecessor Snapshot by shared successor authorization.
type HistoricalPageRepresentations struct {
	Package       string     `json:"package"`
	PackageSHA256 string     `json:"package_sha256"`
	Documents     []Document `json:"documents"`
}

func AttachPageReplacementResolver(c *i18n.Catalog) {
	c.ReplacementResolver = resolvePageReplacements
}

func resolvePageReplacements(root string, c *i18n.Catalog, ref i18n.GlossaryArchiveReference) (*i18n.TranslationUnitReplacementResolution, error) {
	b, err := readRegular(root, ref.Path)
	if err != nil || digest(b) != ref.SHA256 {
		return nil, fmt.Errorf("historical Page representation reference stale")
	}
	frozen, err := readReplacementRepresentations(root, b, ref)
	if err != nil {
		return nil, err
	}
	var descriptor *i18n.TranslationPackageDescriptor
	for i := range c.TranslationPackages {
		if c.TranslationPackages[i].ID == frozen.Package && c.TranslationPackages[i].AuthoritySHA256 == frozen.PackageSHA256 {
			descriptor = &c.TranslationPackages[i]
		}
	}
	if descriptor == nil {
		historical, err := i18n.HistoricalTranslationPackageCatalog(c, frozen.Package, frozen.PackageSHA256)
		if err != nil {
			return nil, err
		}
		if frozen.Package != "site-v2-shell" {
			return resolvePageReplacements(root, historical, ref)
		}
		// A projection must not retain competing current representations of
		// the same physical package. Its exact descriptor supplies every Unit.
		historical.Adapted = append([]i18n.AdaptedTranslationUnit(nil), historical.TranslationPackages[0].Units...)
		refs, err := shellReplacementSuccessorWorld(root, c, historical)
		if err != nil {
			return nil, err
		}
		r, err := resolvePageReplacements(root, historical, ref)
		if err != nil {
			return nil, err
		}
		r.EventCatalog, r.SuccessorReferences = historical, refs
		return r, nil
	}
	if descriptor == nil || descriptor.AuthoritySHA256 != frozen.PackageSHA256 || len(descriptor.Units) != len(frozen.Documents) {
		return nil, fmt.Errorf("replacement package descriptor mismatch")
	}
	next := []PageAdapter{}
	for _, u := range descriptor.Units {
		a, err := replacementPageAdapter(u.Adapter)
		if err != nil || a.Document.StableIndex != u.StableIndex || a.Document.Route != u.Route {
			return nil, fmt.Errorf("replacement requires unique formal Page adapter")
		}
		next = append(next, a)
	}
	projection, err := sharedHistoricalPageProjection(frozen, descriptor, next)
	if err != nil {
		return nil, err
	}
	historical := *c
	historical.TranslationPackages = nil
	historical.Adapted = nil
	historical.TranslationScope = append([]string(nil), c.TranslationScope...)
	for _, p := range c.TranslationPackages {
		if p.ID != frozen.Package {
			historical.TranslationPackages = append(historical.TranslationPackages, p)
			historical.Adapted = append(historical.Adapted, p.Units...)
		}
	}
	d := projection.Descriptor
	out := &i18n.TranslationUnitReplacementResolution{HistoricalCatalog: &historical, Units: projection.Units}
	for i, current := range descriptor.Units {
		for j, id := range historical.TranslationScope {
			if id == current.Unit.ID {
				historical.TranslationScope[j] = frozen.Documents[i].Units[0].ID
			}
		}
	}
	if err := historical.RegisterTranslationPackage(d); err != nil {
		return nil, err
	}
	return out, nil
}

func deriveHistoricalPageProjection(frozen *HistoricalPageRepresentations, descriptor *i18n.TranslationPackageDescriptor, next []PageAdapter) (*historicalPageProjection, error) {
	reconciliation, err := reconcileReplacementRepresentations(frozen.Documents, next)
	if err != nil {
		return nil, err
	}
	changes := map[string]UnitChange{}
	for _, change := range reconciliation.Units {
		changes[change.ID] = change
	}
	d := i18n.TranslationPackageDescriptor{ID: descriptor.ID, AuthoritySHA256: descriptor.AuthoritySHA256}
	out := &historicalPageProjection{Units: map[string]i18n.TranslationUnitReplacement{}}
	for i, old := range frozen.Documents {
		current := descriptor.Units[i]
		a := next[i]
		u := old.Units[0]
		change := changes[u.ID]
		if (u.ID == current.Unit.ID && change.State != "carry") || (u.ID != current.Unit.ID && (change.State != "stale" || change.Replacement != current.Unit.ID || u.SourceSHA == a.Document.Units[0].SourceSHA)) {
			return nil, fmt.Errorf("reconciliation does not prove exact replacement")
		}
		binding := current
		binding.Unit.ID = u.ID
		binding.Adapter, err = historicalReplacementAdapter(current.Adapter, old)
		if err != nil {
			return nil, err
		}
		d.Units = append(d.Units, binding)
		out.Units[u.ID] = i18n.TranslationUnitReplacement{OldUnitID: u.ID, CurrentUnitID: current.Unit.ID, Package: d.ID, PackageSHA256: d.AuthoritySHA256, SourcePath: old.Path, SourceSHA256: old.SourceSHA, StableIndex: old.StableIndex, Route: old.Route, OldInputSHA256: digest([]byte(strings.TrimRight(u.Source, "\n") + "\n")), CurrentInputSHA256: digest([]byte(strings.TrimRight(a.Document.Units[0].Source, "\n") + "\n")), ReconciliationSHA256: identity(reconciliation)}
	}
	out.Descriptor = d
	return out, nil
}

type protectedPiece struct {
	start, end int
	text       string
	token      *ProtectedToken
}

func protectionPieces(u Unit) ([]protectedPiece, error) {
	if _, err := restoreBlock(u, u.Source); err != nil {
		return nil, err
	}
	tokens := append([]ProtectedToken(nil), u.Protected...)
	sort.Slice(tokens, func(i, j int) bool {
		return strings.Index(u.Source, tokens[i].Token) < strings.Index(u.Source, tokens[j].Token)
	})
	pieces := []protectedPiece{}
	pos, rawPos := 0, 0
	for i := range tokens {
		p := &tokens[i]
		at := strings.Index(u.Source, p.Token)
		if at < pos {
			return nil, fmt.Errorf("overlapping protected tokens")
		}
		if at > pos {
			text := u.Source[pos:at]
			pieces = append(pieces, protectedPiece{rawPos, rawPos + len(text), text, nil})
			rawPos += len(text)
		}
		pieces = append(pieces, protectedPiece{rawPos, rawPos + len(p.Raw), p.Token, p})
		rawPos += len(p.Raw)
		pos = at + len(p.Token)
	}
	if pos < len(u.Source) {
		text := u.Source[pos:]
		pieces = append(pieces, protectedPiece{rawPos, rawPos + len(text), text, nil})
	}
	return pieces, nil
}

// Rebase only source-coordinate tokens, never parse translated prose to infer
// a token mapping. Newly released text comes from the formerly opaque source
// atom. The result must restore byte-exactly to the reviewed old candidate.
func (a PageAdapter) RebaseRevisionPredecessor(adapter i18n.TranslationUnitAdapter, raw []byte) ([]byte, error) {
	return a.rebasePredecessor(adapter, raw, false)
}

// ReconcileRepresentation may additionally lock an exact technical literal
// already present in a uniquely bounded historical prose interval. Missing or
// extra occurrences are ambiguous; never infer translated words or nearest text.
func (a PageAdapter) ReconcileRepresentation(adapter i18n.TranslationUnitAdapter, raw []byte) ([]byte, error) {
	return a.rebasePredecessor(adapter, raw, true)
}

func (a PageAdapter) rebasePredecessor(adapter i18n.TranslationUnitAdapter, raw []byte, allowNewAtoms bool) ([]byte, error) {
	old, ok := adapter.(PageAdapter)
	if !ok || !bytes.Equal(old.Source, a.Source) {
		return nil, fmt.Errorf("replacement predecessor source mismatch")
	}
	before, err := old.Restore(string(raw))
	if err != nil {
		return nil, err
	}
	op, err := protectionPieces(old.Document.Units[0])
	if err != nil {
		return nil, err
	}
	np, err := protectionPieces(a.Document.Units[0])
	if err != nil {
		return nil, err
	}
	substitutions := []string{}
	used := map[int]bool{}
	for _, p := range op {
		if p.token == nil {
			continue
		}
		var text strings.Builder
		for j, q := range np {
			if used[j] {
				continue
			}
			if q.token != nil {
				inside := q.start >= p.start && q.end <= p.end
				if q.start == q.end {
					inside = (q.start == p.start && p.start == p.end && q.token.Role == p.token.Role) || (p.start < p.end && q.start > p.start && q.start <= p.end && q.token.Role == "close")
					if q.token.Binding != nil && q.token.Binding.Version == 2 && q.token.Binding.Kind == "markdown-reference-title" && q.start >= p.start && q.start < p.end {
						inside = true
					}
				}
				if inside {
					text.WriteString(q.text)
					used[j] = true
				}
			} else {
				start, end := q.start, q.end
				if start < p.start {
					start = p.start
				}
				if end > p.end {
					end = p.end
				}
				if start < end {
					text.WriteString(q.text[start-q.start : end-q.start])
				}
			}
		}
		substitutions = append(substitutions, p.text, text.String())
	}
	if allowNewAtoms {
		var err error
		raw, substitutions, err = bindNewProseAtoms(op, np, used, raw, substitutions)
		if err != nil {
			return nil, err
		}
	}
	for j, p := range np {
		if p.token != nil && !used[j] {
			return nil, fmt.Errorf("replacement introduces protection outside historical source atoms")
		}
	}
	result := []byte(strings.NewReplacer(substitutions...).Replace(string(raw)))
	after, err := a.Restore(string(result))
	if err != nil || !bytes.Equal(before, after) {
		return nil, fmt.Errorf("replacement baseline does not restore to exact predecessor: %v", err)
	}
	return result, nil
}

func bindNewProseAtoms(op, np []protectedPiece, used map[int]bool, raw []byte, substitutions []string) ([]byte, []string, error) {
	patches := []protectedPiece{}
	for i, p := range op {
		if p.token != nil {
			continue
		}
		groups := map[string][]int{}
		for j, q := range np {
			if q.token != nil && !used[j] && q.start >= p.start && q.end <= p.end {
				if q.token.Role != "atom" || q.token.Binding != nil || q.token.Raw == "" {
					return nil, nil, fmt.Errorf("new protection requires an exact literal atom")
				}
				groups[q.token.Raw] = append(groups[q.token.Raw], j)
			}
		}
		if len(groups) == 0 {
			continue
		}
		left, right := 0, len(raw)
		if i > 0 {
			left = strings.Index(string(raw), op[i-1].text) + len(op[i-1].text)
		}
		if i+1 < len(op) {
			right = strings.Index(string(raw), op[i+1].text)
		}
		if left < 0 || right < left {
			return nil, nil, fmt.Errorf("historical prose anchors reordered")
		}
		literals := []string{}
		for literal := range groups {
			literals = append(literals, literal)
		}
		sort.Strings(literals)
		for _, literal := range literals {
			indexes := groups[literal]
			positions := []int{}
			for at := left; at < right; {
				n := strings.Index(string(raw[at:right]), literal)
				if n < 0 {
					break
				}
				at += n
				// ASCII identifier boundaries prevent A from binding inside API.
				identity := func(b byte) bool {
					return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_'
				}
				if (at == left || !identity(raw[at-1])) && (at+len(literal) == right || !identity(raw[at+len(literal)])) {
					positions = append(positions, at)
				}
				at += len(literal)
			}
			if len(positions) != len(indexes) {
				return nil, nil, fmt.Errorf("new machine atom %q has non-unique historical prose mapping: source=%d target=%d", literal, len(indexes), len(positions))
			}
			for k, j := range indexes {
				marker := fmt.Sprintf("\x00reconcile-%d\x00", j)
				patches = append(patches, protectedPiece{start: positions[k], end: positions[k] + len(literal), text: marker})
				substitutions = append(substitutions, marker, np[j].text)
				used[j] = true
			}
		}
	}
	sort.Slice(patches, func(i, j int) bool { return patches[i].start < patches[j].start })
	var out strings.Builder
	pos := 0
	for _, p := range patches {
		if p.start < pos {
			return nil, nil, fmt.Errorf("overlapping machine mapping")
		}
		out.Write(raw[pos:p.start])
		out.WriteString(p.text)
		pos = p.end
	}
	out.Write(raw[pos:])
	return []byte(out.String()), substitutions, nil
}
