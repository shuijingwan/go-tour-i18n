package sitecontent

import (
	"fmt"
	"sort"
	"strings"
)

const PageRegistryPath = "data/learn-docs-pages.json"
const PageRegistrySchema = "go-learning/stable-page-index/v1"

type StablePage struct {
	Index   int    `json:"index"`
	Path    string `json:"path"`
	Surface string `json:"surface"`
	Retired bool   `json:"retired"`
}
type PageRegistry struct {
	Schema   string       `json:"schema"`
	Pages    []StablePage `json:"pages"`
	Identity string       `json:"identity_sha256"`
}

func sealPages(r *PageRegistry) { r.Identity = ""; r.Identity = identity(*r) }
func ValidatePageRegistry(r PageRegistry) error {
	copy := r
	sealPages(&copy)
	if r.Schema != PageRegistrySchema || r.Identity != copy.Identity {
		return fmt.Errorf("invalid stable Page registry")
	}
	seen := map[string]bool{}
	for i, p := range r.Pages {
		if p.Index != i+1 || !validPath(p.Path) || seen[p.Path] || !strings.Contains(" learn tutorial database modules security ", " "+p.Surface+" ") {
			return fmt.Errorf("invalid Page index/path/surface")
		}
		seen[p.Path] = true
	}
	return nil
}
func FreezePageRegistry(g *Global) (*PageRegistry, error) {
	order := map[string]int{"learn": 0, "tutorial": 1, "database": 2, "modules": 3, "security": 4}
	pages := []Source{}
	for _, s := range g.Sources {
		if s.Package == "learn-docs-v1" && s.Kind == "page" {
			pages = append(pages, s)
		}
	}
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Surface != pages[j].Surface {
			return order[pages[i].Surface] < order[pages[j].Surface]
		}
		return pages[i].Path < pages[j].Path
	})
	r := &PageRegistry{Schema: PageRegistrySchema, Pages: []StablePage{}}
	for _, p := range pages {
		if _, ok := order[p.Surface]; !ok {
			return nil, fmt.Errorf("unknown Page surface")
		}
		r.Pages = append(r.Pages, StablePage{Index: len(r.Pages) + 1, Path: p.Path, Surface: p.Surface})
	}
	sealPages(r)
	return r, ValidatePageRegistry(*r)
}
func LoadPageRegistry(root string, g *Global) (*PageRegistry, error) {
	b, err := readRegular(root, PageRegistryPath)
	if err != nil {
		return nil, err
	}
	var r PageRegistry
	if err := StrictJSON(b, &r); err != nil {
		return nil, err
	}
	if err := ValidatePageRegistry(r); err != nil {
		return nil, err
	}
	want := map[string]string{}
	for _, s := range g.Sources {
		if s.Package == "learn-docs-v1" && s.Kind == "page" {
			want[s.Path] = s.Surface
		}
	}
	for _, p := range r.Pages {
		if p.Retired {
			if want[p.Path] != "" {
				return nil, fmt.Errorf("retired Page reappeared")
			}
			continue
		}
		if want[p.Path] != p.Surface {
			return nil, fmt.Errorf("Page registry stale %s", p.Path)
		}
		delete(want, p.Path)
	}
	if len(want) != 0 {
		return nil, fmt.Errorf("unregistered new Page requires explicit reconciliation")
	}
	if len(r.Pages) > 60 {
		return nil, fmt.Errorf("stable index 61 requires Maintainer batching authority; active=%d", len(r.Pages))
	}
	return &r, nil
}

// ReconcilePageRegistry never guesses moves and never reuses tombstones.
func ReconcilePageRegistry(old PageRegistry, next []StablePage, moves map[string]string) (*PageRegistry, error) {
	if err := ValidatePageRegistry(old); err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, p := range old.Pages {
		if !p.Retired {
			known[p.Path] = true
		}
	}
	destinations := map[string]bool{}
	for a, b := range moves {
		if !known[a] || !validPath(b) || destinations[b] {
			return nil, fmt.Errorf("ambiguous/unregistered explicit move")
		}
		destinations[b] = true
	}
	future := map[string]StablePage{}
	for _, p := range next {
		if !validPath(p.Path) || future[p.Path].Path != "" {
			return nil, fmt.Errorf("invalid next Page")
		}
		future[p.Path] = p
	}
	r := &PageRegistry{Schema: old.Schema, Pages: append([]StablePage{}, old.Pages...)}
	for i, p := range r.Pages {
		if p.Retired {
			if future[p.Path].Path != "" {
				return nil, fmt.Errorf("tombstone cannot be reused")
			}
			continue
		}
		dest := p.Path
		if m := moves[p.Path]; m != "" {
			dest = m
			if future[p.Path].Path != "" || future[dest].Path == "" {
				return nil, fmt.Errorf("ambiguous explicit move")
			}
		}
		if n, ok := future[dest]; ok {
			r.Pages[i].Path = dest
			r.Pages[i].Surface = n.Surface
			delete(future, dest)
		} else {
			r.Pages[i].Retired = true
		}
	}
	paths := []string{}
	for p := range future {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		n := future[p]
		n.Index = len(r.Pages) + 1
		n.Retired = false
		if n.Index > 60 {
			return nil, fmt.Errorf("index61: Page=%s surface=%s active=%d occupancy=30/30; Maintainer must decide batching", n.Path, n.Surface, len(next))
		}
		r.Pages = append(r.Pages, n)
	}
	sealPages(r)
	return r, ValidatePageRegistry(*r)
}

// ReconcilePageSources is the source-bound future sync hook. The index-only
// primitive never guesses source bytes; this adapter supplies real diagnostics
// when fixed capacity requires a Maintainer decision.
func ReconcilePageSources(old PageRegistry, next []Source, raw map[string][]byte, moves map[string]string) (*PageRegistry, error) {
	pages := []StablePage{}
	documents := map[string]Document{}
	for _, s := range next {
		if s.Kind != "page" || s.Package != "learn-docs-v1" {
			return nil, fmt.Errorf("Page reconciliation requires exact Page source set")
		}
		d, err := ParseDocument(s, raw[s.Path])
		if err != nil {
			return nil, err
		}
		pages = append(pages, StablePage{Path: s.Path, Surface: s.Surface})
		documents[s.Path] = *d
	}
	r, err := ReconcilePageRegistry(old, pages, moves)
	if err == nil || !strings.Contains(err.Error(), "index61:") {
		return r, err
	}
	known := map[string]bool{}
	occupancy := [2]int{}
	for _, p := range old.Pages {
		if p.Retired {
			continue
		}
		dest := p.Path
		if moves[dest] != "" {
			dest = moves[dest]
		}
		known[dest] = true
		if _, ok := documents[dest]; ok && p.Index <= 60 {
			occupancy[(p.Index-1)/30]++
		}
	}
	added := []string{}
	for _, p := range pages {
		if !known[p.Path] {
			added = append(added, p.Path)
		}
	}
	sort.Strings(added)
	for i, path := range added {
		index := len(old.Pages) + i + 1
		if index <= 60 {
			occupancy[(index-1)/30]++
			continue
		}
		d := documents[path]
		page, parseErr := FormalPage(d, raw[path], index)
		if parseErr != nil {
			return nil, parseErr
		}
		return nil, fmt.Errorf("index61: Page=%s source_path=%s source_sha256=%s source_bytes=%d context_bytes=%d active=%d occupancy=%d/%d; Maintainer must decide batching", page.Units[0].ID, path, d.SourceSHA, len(raw[path]), len(page.Units[0].Source), len(pages), occupancy[0], occupancy[1])
	}
	return nil, err
}
