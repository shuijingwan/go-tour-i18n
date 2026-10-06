package sitecontent

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"sort"
)

const MaxGenerationUnits = 60
const MaxGenerationTextBytes = 12 << 10
const MaxGenerationContextBytes = 128 << 10
const MaxGenerationOutputBytes = 16 << 20

type WorkingSet struct {
	Index        int      `json:"index"`
	Kind         string   `json:"kind"`
	Selected     []string `json:"selected_units"`
	TextBytes    int      `json:"text_bytes"`
	ContextBytes int      `json:"context_bytes"`
}
type BatchPlan struct {
	Schema    string       `json:"schema"`
	Contract  string       `json:"parser_contract"`
	SourceSHA string       `json:"source_sha256"`
	Sets      []WorkingSet `json:"working_sets"`
	Identity  string       `json:"identity_sha256"`
}

func selectedDocuments(docs []Document, ids []string) []Document {
	selected := map[string]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	result := []Document{}
	for _, d := range docs {
		for _, u := range d.Units {
			if selected[u.ID] {
				result = append(result, d)
				break
			}
		}
	}
	return result
}

// PlanWorkingSets preserves document order and linguistic block order, separates
// Page/Data and cuts before any quality boundary. It never cuts a language unit.
// targets=nil plans Generation; non-nil uses actual Reviewer source+target sizes.
func DiagnosticSlotPlan(docs []Document, raw map[string][]byte, ids []string, targets map[string]string) (*BatchPlan, error) {
	if err := validateDocuments(docs); err != nil {
		return nil, err
	}
	if err := exactIDs(ids, unitsByID(docs)); err != nil {
		return nil, err
	}
	limit, textCap, contextCap := MaxGenerationUnits, MaxGenerationTextBytes, MaxGenerationContextBytes
	if targets != nil {
		limit, textCap, contextCap = 60, 24<<10, 128<<10
	}
	selected := map[string]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	ordered := append([]Document{}, docs...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Kind != ordered[j].Kind {
			return ordered[i].Kind < ordered[j].Kind
		}
		return ordered[i].Path < ordered[j].Path
	})
	p := &BatchPlan{Schema: "go-learning/working-set-plan/v1", Contract: UnitContract, SourceSHA: documentIdentity(selectedDocuments(ordered, ids)), Sets: []WorkingSet{}}
	current := WorkingSet{Selected: []string{}}
	seen := map[string]bool{}
	flush := func() {
		if len(current.Selected) == 0 {
			return
		}
		sort.Strings(current.Selected)
		current.Index = len(p.Sets) + 1
		p.Sets = append(p.Sets, current)
		current = WorkingSet{Selected: []string{}}
		seen = map[string]bool{}
	}
	for _, d := range ordered {
		b, ok := raw[d.Path]
		if !ok || digest(b) != d.SourceSHA {
			return nil, fmt.Errorf("planner context missing/hash mismatch: %s", d.Path)
		}
		for _, u := range d.Units {
			if !selected[u.ID] {
				continue
			}
			size := len(u.Source)
			if targets != nil {
				v, ok := targets[u.ID]
				if !ok {
					return nil, fmt.Errorf("planner missing target")
				}
				size += len(v)
			}
			if size > textCap || len(b) > contextCap {
				return nil, fmt.Errorf("linguistic unit/context exceeds working-set cap: %s", u.ID)
			}
			context := current.ContextBytes
			if !seen[d.Path] {
				context += len(b)
			}
			if len(current.Selected) == limit || current.TextBytes+size > textCap || context > contextCap || (current.Kind != "" && current.Kind != d.Kind) {
				flush()
			}
			current.Kind = d.Kind
			current.Selected = append(current.Selected, u.ID)
			current.TextBytes += size
			if !seen[d.Path] {
				seen[d.Path] = true
				current.ContextBytes += len(b)
			}
		}
	}
	flush()
	p.Identity = identity(*p)
	return p, nil
}

func requireWorkingSet(docs []Document, raw map[string][]byte, ids []string, targets map[string]string) error {
	p, err := PlanPageWorkingSets(docs, raw, ids, targets)
	if err != nil {
		return err
	}
	if len(p.Sets) != 1 {
		return fmt.Errorf("working set exceeds unit/text/context boundary or mixes Page/Data; use deterministic planner")
	}
	return nil
}

// GenerationPlan is read-only and shares exactly the export authorization gate.
func GenerationPlan(root, locale, pkg, task string, finding *Reference) (*BatchPlan, error) {
	_, _, docs, raw, glossary, err := currentPackage(root, locale, pkg)
	if err != nil {
		return nil, err
	}
	if _, err := i18n.CurrentParsedGlossaryReview(root, locale, digest(glossary)); err != nil {
		return nil, err
	}
	ids, err := generationEligible(root, locale, pkg, task, finding, glossary, docs)
	if err != nil {
		return nil, err
	}
	return PlanPageWorkingSets(docs, raw, ids, nil)
}
func ReviewerPlan(root, locale, pkg string, refs []Reference) (*BatchPlan, error) {
	_, _, docs, raw, glossary, err := currentPackage(root, locale, pkg)
	if err != nil {
		return nil, err
	}
	values, err := selections(root, locale, pkg, refs, docs)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	targets := map[string]string{}
	for id, s := range values {
		if s.Generation.Batch.GlossarySHA != digest(glossary) {
			return nil, fmt.Errorf("new review plan requires current glossary")
		}
		ids = append(ids, id)
		targets[id] = s.Target.Text
	}
	sort.Strings(ids)
	return PlanPageWorkingSets(docs, raw, ids, targets)
}

// Fixed Page membership is authority, not a byte-packing heuristic. Transport
// safety is a generous hard limit; exceeding it stops instead of splitting.
func PlanPageWorkingSets(docs []Document, raw map[string][]byte, ids []string, targets map[string]string) (*BatchPlan, error) {
	if err := validateDocuments(docs); err != nil {
		return nil, err
	}
	if err := exactIDs(ids, unitsByID(docs)); err != nil {
		return nil, err
	}
	selected := map[string]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	sets := map[int]*WorkingSet{}
	p := &BatchPlan{Schema: "go-learning/fixed-page-plan/v1", Contract: PageContract, SourceSHA: documentIdentity(selectedDocuments(docs, ids)), Sets: []WorkingSet{}}
	for _, d := range docs {
		for _, u := range d.Units {
			if !selected[u.ID] {
				continue
			}
			if d.UnitScope != PageContract || len(d.Units) != 1 || d.Kind != "page" {
				return nil, fmt.Errorf("internal slots/data are not formal TranslationUnits")
			}
			n := 1
			if d.StableIndex > 0 {
				if d.StableIndex > 60 {
					return nil, fmt.Errorf("index61 requires Maintainer authority: %s", d.Path)
				}
				n = (d.StableIndex-1)/30 + 1
			}
			if sets[n] == nil {
				sets[n] = &WorkingSet{Index: n, Kind: "page", Selected: []string{}}
			}
			s := sets[n]
			s.Selected = append(s.Selected, u.ID)
			s.TextBytes += len(u.Source)
			if targets != nil {
				t, ok := targets[u.ID]
				if !ok {
					return nil, fmt.Errorf("missing Page target")
				}
				s.TextBytes += len(t)
			}
			b, ok := raw[d.Path]
			if !ok || digest(b) != d.SourceSHA {
				return nil, fmt.Errorf("Page context mismatch")
			}
			s.ContextBytes += len(b)
		}
	}
	for n := 1; n <= 2; n++ {
		s := sets[n]
		if s == nil {
			continue
		}
		if len(s.Selected) > 30 || s.TextBytes > 16<<20 || s.ContextBytes > 16<<20 {
			return nil, fmt.Errorf("fixed Page batch %d hard transport limit: pages=%d text=%d context=%d; no auto-split", n, len(s.Selected), s.TextBytes, s.ContextBytes)
		}
		sort.Strings(s.Selected)
		p.Sets = append(p.Sets, *s)
	}
	p.Identity = identity(*p)
	return p, nil
}

// PlanWorkingSets is retained solely for internal slot diagnostics.
func PlanWorkingSets(d []Document, r map[string][]byte, i []string, t map[string]string) (*BatchPlan, error) {
	return DiagnosticSlotPlan(d, r, i, t)
}
