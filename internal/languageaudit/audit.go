// Package languageaudit provides advisory, deterministic enumeration over a
// parser-owned visible representation. It creates no language judgment or gate.
package languageaudit

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Kind string

const (
	Prose           Kind = "prose"
	Link            Kind = "link-label"
	TechnicalLink   Kind = "technical-link-label"
	Code            Kind = "inline-code"
	Identity        Kind = "technical-identity"
	ProtectedWord   Kind = "protected-visible-word"
	ProtectedSymbol Kind = "protected-visible-symbol"
	Opaque          Kind = "opaque"
)

type Piece struct {
	Text    string
	Kind    Kind
	Binding string
}
type Block struct {
	Locator, Context string
	Start, End       int
	Pieces           []Piece
}
type Document struct {
	Owner  string
	Blocks []Block
}
type SpacingRule struct {
	ProseScript     string `json:"prose_script"`
	TechnicalScript string `json:"technical_script"`
	Kinds           []Kind `json:"kinds"`
}

// Policy is explicit locale data; identities are injected from existing authority.
type Policy struct {
	Locale                           string            `json:"locale"`
	Profile                          string            `json:"profile"`
	ProseScripts                     []string          `json:"prose_scripts"`
	Spacing                          []SpacingRule     `json:"spacing"`
	SentenceEnd                      map[string]string `json:"sentence_end"`
	ClosingQuotes                    string            `json:"closing_quotes"`
	SentenceTerminators              string            `json:"sentence_terminators"`
	ForbiddenClausePunctuation       []string          `json:"forbidden_clause_punctuation"`
	HeadingSymbols                   string            `json:"heading_symbols"`
	ProtectedIdentities              []string          `json:"protected_identities,omitempty"`
	ResidualMinWords                 int               `json:"residual_min_words"`
	PunctuationOnlyParagraphIsDefect bool              `json:"punctuation_only_paragraph_is_defect"`
}
type Decision string

const (
	Defect    Decision = "deterministic-defect"
	Candidate Decision = "review-candidate"
)

type Diagnostic struct {
	Owner, Locator, Context  string
	Start, End               int // parser block source coordinates; -1 denotes rendered coordinates
	VisibleStart, VisibleEnd int // rune offsets in the visible block, not edit offsets
	Class                    string
	Decision                 Decision
	Evidence                 string
	Profile                  string
}
type Result struct{ Diagnostics []Diagnostic }

func (p Policy) validate() error {
	if p.Locale == "" || p.Profile == "" || len(p.ProseScripts) == 0 || p.ResidualMinWords < 1 {
		return fmt.Errorf("explicit locale/profile/scripts and residual word threshold required")
	}
	names := append([]string(nil), p.ProseScripts...)
	for _, r := range p.Spacing {
		names = append(names, r.ProseScript, r.TechnicalScript)
		if len(r.Kinds) == 0 {
			return fmt.Errorf("spacing rule requires explicit representation kinds")
		}
		for _, k := range r.Kinds {
			if !technical(k) {
				return fmt.Errorf("spacing rules require technical representation kinds")
			}
		}
	}
	for _, n := range names {
		if unicode.Scripts[n] == nil {
			return fmt.Errorf("unknown Unicode script %q", n)
		}
	}
	for from, to := range p.SentenceEnd {
		if utf8.RuneCountInString(from) != 1 || to == "" {
			return fmt.Errorf("invalid punctuation rule")
		}
	}
	for _, s := range p.ForbiddenClausePunctuation {
		if s == "" {
			return fmt.Errorf("empty punctuation sequence")
		}
	}
	return nil
}
func script(r rune, name string) bool { return unicode.Is(unicode.Scripts[name], r) }
func technical(k Kind) bool           { return k == Code || k == Identity || k == Link || k == TechnicalLink }
func linkKind(k Kind) bool            { return k == Link || k == TechnicalLink }
func wordRune(r rune) bool            { return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) }
func runeScript(r rune) string {
	for name, table := range unicode.Scripts {
		if unicode.Is(table, r) {
			return name
		}
	}
	return ""
}
func identityNeighbor(r, edge rune) bool {
	return r == '_' || unicode.IsMark(r) || unicode.IsDigit(r) || (unicode.IsLetter(r) && runeScript(r) == runeScript(edge))
}
func normalized(in []Piece, p Policy) []Piece {
	out := []Piece{}
	terms := append([]string(nil), p.ProtectedIdentities...)
	sort.SliceStable(terms, func(i, j int) bool { return utf8.RuneCountInString(terms[i]) > utf8.RuneCountInString(terms[j]) })
	for _, x := range in {
		if x.Kind != Prose && x.Kind != Link {
			out = append(out, x)
			continue
		}
		rs := []rune(x.Text)
		start := 0
		emit := func(i, z int) {
			if start < i {
				out = append(out, Piece{Text: string(rs[start:i]), Kind: x.Kind, Binding: x.Binding})
			}
			out = append(out, Piece{Text: string(rs[i:z]), Kind: Identity, Binding: x.Binding})
			start = z
		}
		for i := 0; i < len(rs); {
			matched := 0
			for _, term := range terms {
				tr := []rune(term)
				if len(tr) == 0 || i+len(tr) > len(rs) || string(rs[i:i+len(tr)]) != term {
					continue
				}
				if i > 0 && identityNeighbor(rs[i-1], tr[0]) {
					continue
				}
				end := i + len(tr)
				if end < len(rs) && identityNeighbor(rs[end], tr[len(tr)-1]) {
					continue
				}
				matched = len(tr)
				break
			}
			if matched > 0 {
				emit(i, i+matched)
				i += matched
				continue
			}
			i++
		}
		if start < len(rs) {
			out = append(out, Piece{Text: string(rs[start:]), Kind: x.Kind, Binding: x.Binding})
		}
	}
	return out
}

type visible struct {
	chars  []rune
	kinds  []Kind
	ids    []int
	pieces []Piece
}

func flatten(in []Piece, p Policy) visible {
	v := visible{pieces: normalized(in, p)}
	for i, x := range v.pieces {
		text := x.Text
		if x.Kind == Opaque {
			text = "\ufffc"
		}
		for _, r := range text {
			v.chars = append(v.chars, r)
			v.kinds = append(v.kinds, x.Kind)
			v.ids = append(v.ids, i)
		}
	}
	return v
}
func edgeScript(v visible, i int, name string) bool {
	if unicode.IsLetter(v.chars[i]) {
		return script(v.chars[i], name)
	}
	if unicode.IsMark(v.chars[i]) || unicode.IsDigit(v.chars[i]) {
		id := v.ids[i]
		for j := i; j >= 0 && v.ids[j] == id; j-- {
			if unicode.IsLetter(v.chars[j]) {
				return script(v.chars[j], name)
			}
		}
		for j := i + 1; j < len(v.chars) && v.ids[j] == id; j++ {
			if unicode.IsLetter(v.chars[j]) {
				return script(v.chars[j], name)
			}
		}
	}
	return false
}
func proseScript(v visible, i int, name string) bool {
	if v.kinds[i] != Prose && v.kinds[i] != Link && v.kinds[i] != ProtectedWord {
		return false
	}
	return edgeScript(v, i, name)
}
func technicalScript(v visible, i int, r SpacingRule) bool {
	allowed := false
	for _, k := range r.Kinds {
		if v.kinds[i] == k || (k == Link && v.kinds[i] == TechnicalLink) {
			allowed = true
		}
	}
	return allowed && edgeScript(v, i, r.TechnicalScript)
}
func punctuationOnly(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsPunct(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
func hasProseScript(v visible, a, z int, names []string) bool {
	for i := a; i < z; i++ {
		if v.kinds[i] != Prose && v.kinds[i] != ProtectedWord {
			continue
		}
		for _, n := range names {
			if unicode.IsLetter(v.chars[i]) && script(v.chars[i], n) {
				return true
			}
		}
	}
	return false
}

func endsSentence(s string, p Policy) bool {
	s = strings.TrimRight(strings.TrimSpace(s), p.ClosingQuotes)
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return strings.ContainsRune(p.SentenceTerminators, r)
}
func sourceLinkContinuation(source Document, b Block) bool {
	for _, x := range b.Pieces {
		if !linkKind(x.Kind) || x.Binding == "" {
			continue
		}
		for _, s := range source.Blocks {
			prose := false
			match := false
			for _, q := range s.Pieces {
				if q.Kind == Prose && strings.TrimSpace(q.Text) != "" {
					prose = true
				}
				if linkKind(q.Kind) && q.Binding == x.Binding {
					match = true
				}
			}
			if prose && match {
				return true
			}
		}
	}
	return false
}

type word struct {
	text        string
	start, end  int
	eligible    bool
	protected   bool
	breakBefore bool
}

func words(v visible) []word {
	out := []word{}
	for i := 0; i < len(v.chars); {
		if !unicode.IsLetter(v.chars[i]) {
			i++
			continue
		}
		active := runeScript(v.chars[i])
		z := i + 1
		for z < len(v.chars) && wordRune(v.chars[z]) {
			if unicode.IsLetter(v.chars[z]) && runeScript(v.chars[z]) != active {
				break
			}
			z++
		}
		eligible := true
		protected := false
		for k := i; k < z; k++ {
			if v.kinds[k] == Code || v.kinds[k] == Identity || v.kinds[k] == TechnicalLink || v.kinds[k] == Opaque {
				eligible = false
			}
			if v.kinds[k] == ProtectedWord {
				protected = true
			}
		}
		barrier := false
		if len(out) > 0 {
			for k := out[len(out)-1].end; k < i; k++ {
				if v.kinds[k] == Opaque || v.kinds[k] == Code || v.kinds[k] == Identity || v.kinds[k] == TechnicalLink {
					barrier = true
				}
			}
		}
		out = append(out, word{strings.ToLower(string(v.chars[i:z])), i, z, eligible, protected, barrier})
		i = z
	}
	return out
}

// Scan returns facts under the selected audit policy, never a formal FAILED.
// Residual overlap and paragraph-link split are always review candidates.
func Scan(p Policy, source, target Document) (Result, error) {
	if err := p.validate(); err != nil {
		return Result{}, err
	}
	for _, d := range []Document{source, target} {
		for _, b := range d.Blocks {
			for _, x := range b.Pieces {
				if !utf8.ValidString(x.Text) {
					return Result{}, fmt.Errorf("invalid UTF-8")
				}
				switch x.Kind {
				case Prose, Link, TechnicalLink, Code, Identity, ProtectedWord, ProtectedSymbol, Opaque:
				default:
					return Result{}, fmt.Errorf("unknown representation kind %q", x.Kind)
				}
			}
		}
	}
	result := Result{Diagnostics: []Diagnostic{}}
	add := func(b Block, v visible, a, z int, class string, decision Decision) {
		evidence := string(v.chars[a:z])
		left, right := a-24, z+24
		if left < 0 {
			left = 0
		}
		if right > len(v.chars) {
			right = len(v.chars)
		}
		if a == z || class == "script-adjacency" {
			evidence = string(v.chars[left:right])
		}
		result.Diagnostics = append(result.Diagnostics, Diagnostic{target.Owner, b.Locator, b.Context, b.Start, b.End, a, z, class, decision, evidence, p.Profile})
	}
	// A source-text overlap index works for same-script locales too. This is
	// candidate enumeration, not language identification or a translation rule.
	sourceRuns := map[string][][]string{}
	for _, b := range source.Blocks {
		ws := words(flatten(b.Pieces, p))
		for i, w := range ws {
			if !w.eligible {
				continue
			}
			run := []string{}
			for j := i; j < len(ws) && ws[j].eligible && (j == i || !ws[j].breakBefore); j++ {
				run = append(run, ws[j].text)
			}
			sourceRuns[w.text] = append(sourceRuns[w.text], run)
		}
	}
	var previous *Block
	for _, b := range target.Blocks {
		v := flatten(b.Pieces, p)
		rs := v.chars
		if len(rs) == 0 {
			continue
		}
		for i := 0; i+1 < len(rs); i++ {
			if v.ids[i] == v.ids[i+1] || !wordRune(rs[i]) || !wordRune(rs[i+1]) {
				continue
			}
			for _, r := range p.Spacing {
				if (proseScript(v, i, r.ProseScript) && technicalScript(v, i+1, r)) || (technicalScript(v, i, r) && proseScript(v, i+1, r.ProseScript)) {
					add(b, v, i, i+2, "script-adjacency", Defect)
					break
				}
			}
		}
		local := hasProseScript(v, 0, len(rs), p.ProseScripts)
		if local {
			for i, r := range rs {
				if v.kinds[i] != Prose && v.kinds[i] != ProtectedSymbol {
					continue
				}
				if _, bad := p.SentenceEnd[string(r)]; !bad {
					continue
				}
				if r == '.' && ((i > 0 && rs[i-1] == '.') || (i+1 < len(rs) && rs[i+1] == '.')) {
					continue
				}
				rest := strings.TrimSpace(string(rs[i+1:]))
				rest = strings.Trim(rest, p.ClosingQuotes)
				if strings.TrimSpace(rest) == "" {
					add(b, v, i, i+1, "sentence-punctuation", Defect)
				}
			}
			for _, seq := range p.ForbiddenClausePunctuation {
				needle := []rune(seq)
				for i := 0; i+len(needle) <= len(rs); i++ {
					if string(rs[i:i+len(needle)]) != seq {
						continue
					}
					prose := true
					for j := i; j < i+len(needle); j++ {
						if v.kinds[j] != Prose {
							prose = false
						}
					}
					if prose && hasProseScript(v, 0, i, p.ProseScripts) && hasProseScript(v, i+len(needle), len(rs), p.ProseScripts) {
						add(b, v, i, i+len(needle), "clause-punctuation", Defect)
					}
				}
			}
			if b.Context == "heading" {
				for i, r := range rs {
					if strings.ContainsRune(p.HeadingSymbols, r) && v.kinds[i] != Code && v.kinds[i] != Identity {
						add(b, v, i, i+1, "protected-prose-symbol", Candidate)
					}
				}
			}
		}
		if b.Context == "paragraph" {
			literal := false
			for _, k := range v.kinds {
				if k == Opaque || k == Code || k == Identity || k == TechnicalLink {
					literal = true
				}
			}
			if !literal && punctuationOnly(string(rs)) {
				decision := Candidate
				if p.PunctuationOnlyParagraphIsDefect {
					decision = Defect
				}
				add(b, v, 0, len(rs), "punctuation-only-paragraph", decision)
			}
			if previous != nil && !endsSentence(blockText(*previous, p), p) && sourceLinkContinuation(source, b) {
				onlyLink := true
				hasLink := false
				for _, x := range b.Pieces {
					if linkKind(x.Kind) {
						hasLink = true
					} else if strings.TrimSpace(x.Text) != "" && !punctuationOnly(x.Text) {
						onlyLink = false
					}
				}
				if onlyLink && hasLink {
					add(b, v, 0, len(rs), "paragraph-link-split", Candidate)
				}
			}
			saved := b
			previous = &saved
		} else {
			previous = nil
		}
		ws := words(v)
		for i := 0; i < len(ws); {
			w := ws[i]
			best := 0
			if w.eligible {
				for _, run := range sourceRuns[w.text] {
					n := 0
					for n < len(run) && i+n < len(ws) && ws[i+n].eligible && (n == 0 || !ws[i+n].breakBefore) && ws[i+n].text == run[n] {
						n++
					}
					if n > best {
						best = n
					}
				}
			}
			if best >= p.ResidualMinWords {
				class := "source-prose-overlap"
				for j := i; j < i+best; j++ {
					if ws[j].protected {
						class = "protected-language-candidate"
					}
				}
				add(b, v, ws[i].start, ws[i+best-1].end, class, Candidate)
				i += best
			} else {
				i++
			}
		}
	}
	sort.SliceStable(result.Diagnostics, func(i, j int) bool {
		a, b := result.Diagnostics[i], result.Diagnostics[j]
		if a.Locator != b.Locator {
			return a.Locator < b.Locator
		}
		if a.VisibleStart != b.VisibleStart {
			return a.VisibleStart < b.VisibleStart
		}
		if a.Class != b.Class {
			return a.Class < b.Class
		}
		return a.Decision < b.Decision
	})
	return result, nil
}
func blockText(b Block, p Policy) string { return string(flatten(b.Pieces, p).chars) }

// BindIdentities exposes the existing exact, Unicode-boundary-aware identity
// mapping to adapters with source-owned lexical bindings (for example code
// comment identifiers). This is representation metadata, not a locale policy.
func BindIdentities(in []Piece, identities []string) ([]Piece, error) {
	for _, id := range identities {
		if id == "" || !utf8.ValidString(id) {
			return nil, fmt.Errorf("invalid bound identity")
		}
	}
	for _, p := range in {
		if !utf8.ValidString(p.Text) {
			return nil, fmt.Errorf("invalid UTF-8")
		}
	}
	return normalized(in, Policy{ProtectedIdentities: identities}), nil
}
