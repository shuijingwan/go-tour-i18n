package i18n

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

type ChangeKind string

const (
	Unchanged      ChangeKind = "unchanged"
	ContentChanged ChangeKind = "content_changed"
	Moved          ChangeKind = "moved"
	Added          ChangeKind = "added"
	Removed        ChangeKind = "removed"
	Ambiguous      ChangeKind = "ambiguous"
)

var ChangeKinds = []ChangeKind{Unchanged, ContentChanged, Moved, Added, Removed, Ambiguous}

type PageChange struct {
	Kind             ChangeKind
	PageID           string
	OldArticle       string
	OldSectionNumber int
	OldRoute         string
	OldSourceTitle   string
	OldSourceSHA256  string
	NewArticle       string
	NewSectionNumber int
	NewRoute         string
	NewSourceTitle   string
	NewSourceSHA256  string
	Reason           string
}

type PreviewReport struct {
	Changes            []PageChange
	ConditionalChanges []PageChange
}

func (r *PreviewReport) Count(kind ChangeKind) int {
	n := 0
	for _, change := range r.Changes {
		if change.Kind == kind {
			n++
		}
	}
	return n
}

func (r *PreviewReport) ConditionalCount(kind ChangeKind) int {
	n := 0
	for _, change := range r.ConditionalChanges {
		if change.Kind == kind {
			n++
		}
	}
	return n
}

func (r *PreviewReport) SafeForCatalogWrite() bool {
	return r.Count(Added) == 0 && r.Count(Removed) == 0 && r.Count(Ambiguous) == 0 &&
		r.ConditionalCount(Added) == 0 && r.ConditionalCount(Removed) == 0 && r.ConditionalCount(Ambiguous) == 0 &&
		r.ConditionalCount(ContentChanged) == 0 && r.ConditionalCount(Moved) == 0
}

func (r *PreviewReport) NeedsManualMapping() bool {
	return r.Count(Added)+r.Count(Removed)+r.Count(Ambiguous) > 0 ||
		r.ConditionalCount(Added)+r.ConditionalCount(Removed)+r.ConditionalCount(Ambiguous)+r.ConditionalCount(ContentChanged)+r.ConditionalCount(Moved) > 0
}

// HydrateCatalogSources attaches current baseline page sources to the committed
// identity catalog. It refuses a catalog that no longer describes that source.
func HydrateCatalogSources(committed, baseline *Catalog) error {
	byRoute := make(map[string]*Page, len(baseline.Pages))
	byHash := make(map[string][]*Page, len(baseline.Pages))
	for i := range baseline.Pages {
		byRoute[baseline.Pages[i].Route] = &baseline.Pages[i]
		byHash[baseline.Pages[i].SourceSHA256] = append(byHash[baseline.Pages[i].SourceSHA256], &baseline.Pages[i])
	}
	for i := range committed.Pages {
		current := byRoute[committed.Pages[i].Route]
		if current == nil || current.SourceSHA256 != committed.Pages[i].SourceSHA256 {
			matches := byHash[committed.Pages[i].SourceSHA256]
			if len(matches) > 0 {
				current = matches[0]
			}
		}
		if current == nil || current.SourceSHA256 != committed.Pages[i].SourceSHA256 {
			return fmt.Errorf("%s: committed catalog does not match current English source; run upstream preview", committed.Pages[i].ID)
		}
		committed.Pages[i].Source = current.Source
	}
	return nil
}

func PreviewCatalog(old, next *Catalog) (*PreviewReport, error) {
	report := &PreviewReport{}
	oldUsed := make([]bool, len(old.Pages))
	newUsed := make([]bool, len(next.Pages))
	oldInitialRoutes := indexPages(old.Pages, func(p Page) string { return p.Route })
	newInitialRoutes := indexPages(next.Pages, func(p Page) string { return p.Route })
	// An unchanged route disambiguates byte-identical duplicate lesson pages;
	// it is not used when the content differs.
	for route, oldIndexes := range oldInitialRoutes {
		newIndexes := newInitialRoutes[route]
		if len(oldIndexes) == 1 && len(newIndexes) == 1 && old.Pages[oldIndexes[0]].SourceSHA256 == next.Pages[newIndexes[0]].SourceSHA256 {
			oi, ni := oldIndexes[0], newIndexes[0]
			report.Changes = append(report.Changes, makeChange(Unchanged, &old.Pages[oi], &next.Pages[ni], "source hash and current location are unchanged"))
			oldUsed[oi], newUsed[ni] = true, true
		}
	}
	oldHashes := indexPages(old.Pages, func(p Page) string { return p.SourceSHA256 })
	newHashes := indexPages(next.Pages, func(p Page) string { return p.SourceSHA256 })

	// A unique byte-identical source is the only automatic move signal.
	for hash, oldIndexes := range oldHashes {
		newIndexes := newHashes[hash]
		if len(oldIndexes) != 1 || len(newIndexes) != 1 {
			continue
		}
		oi, ni := oldIndexes[0], newIndexes[0]
		if oldUsed[oi] || newUsed[ni] {
			continue
		}
		kind, reason := Unchanged, "source hash and current location are unchanged"
		if !sameLocation(old.Pages[oi], next.Pages[ni]) {
			kind, reason = Moved, "unique source hash matched at a different route or position"
		}
		report.Changes = append(report.Changes, makeChange(kind, &old.Pages[oi], &next.Pages[ni], reason))
		oldUsed[oi], newUsed[ni] = true, true
	}

	oldRoutes := unmatchedIndex(old.Pages, oldUsed, func(p Page) string { return p.Route })
	newRoutes := unmatchedIndex(next.Pages, newUsed, func(p Page) string { return p.Route })
	oldSignatureCounts := unmatchedSignatureCounts(old.Pages, oldUsed)
	newSignatureCounts := unmatchedSignatureCounts(next.Pages, newUsed)
	for route, oldIndexes := range oldRoutes {
		newIndexes := newRoutes[route]
		if len(oldIndexes) != 1 || len(newIndexes) != 1 {
			continue
		}
		oi, ni := oldIndexes[0], newIndexes[0]
		compatible, err := compatibleStructure(old.Pages[oi].Source, next.Pages[ni].Source)
		if err != nil {
			return nil, err
		}
		kind, reason := Ambiguous, "current route remains, but protected structure changed"
		if compatible && oldSignatureCounts[signatureKey(old.Pages[oi].Source)] == 1 && newSignatureCounts[signatureKey(next.Pages[ni].Source)] == 1 {
			kind, reason = ContentChanged, "current route remains and protected structure is uniquely compatible"
		} else if isStandaloneConditionalProjection(old.Pages[oi].Source, next.Pages[ni].Source) {
			kind, reason = ContentChanged, "current route remains and source differs only by standalone conditional projection"
		}
		report.Changes = append(report.Changes, makeChange(kind, &old.Pages[oi], &next.Pages[ni], reason))
		oldUsed[oi], newUsed[ni] = true, true
	}

	// A structure-only match away from the old route could be a move plus edit,
	// but is deliberately never migrated automatically.
	for oi := range old.Pages {
		if oldUsed[oi] {
			continue
		}
		matches := []int{}
		for ni := range next.Pages {
			if newUsed[ni] {
				continue
			}
			compatible, err := compatibleStructure(old.Pages[oi].Source, next.Pages[ni].Source)
			if err != nil {
				return nil, err
			}
			if compatible {
				matches = append(matches, ni)
			}
		}
		if len(matches) == 1 && uniqueUnmatchedSignature(old.Pages, oldUsed, oi) {
			ni := matches[0]
			report.Changes = append(report.Changes, makeChange(Ambiguous, &old.Pages[oi], &next.Pages[ni], "possible move with content change requires manual mapping"))
			oldUsed[oi], newUsed[ni] = true, true
		}
	}
	for oi := range old.Pages {
		if !oldUsed[oi] {
			report.Changes = append(report.Changes, makeChange(Removed, &old.Pages[oi], nil, "old page has no conservative match"))
		}
	}
	for ni := range next.Pages {
		if !newUsed[ni] {
			report.Changes = append(report.Changes, makeChange(Added, nil, &next.Pages[ni], "new page has no conservative match"))
		}
	}
	report.ConditionalChanges = previewConditional(old.Conditional, next.Conditional)
	sortChanges(report.Changes)
	sortChanges(report.ConditionalChanges)
	return report, nil
}

func isStandaloneConditionalProjection(old, next []byte) bool {
	return bytes.Contains(old, []byte("#appengine:")) && bytes.Equal(projectStandaloneConditionalContent(old, "appengine"), next)
}

func ReconcileCatalog(committed, next *Catalog, report *PreviewReport) (*Catalog, error) {
	if !report.SafeForCatalogWrite() {
		return nil, fmt.Errorf("upstream changes are not safe for catalog write; run upstream preview and map added, removed, or ambiguous pages explicitly")
	}
	ids := map[string]string{}
	for _, change := range report.Changes {
		ids[change.NewRoute] = change.PageID
	}
	out := &Catalog{Conditional: append([]ConditionalPage(nil), next.Conditional...)}
	for _, page := range next.Pages {
		id := ids[page.Route]
		if id == "" {
			return nil, fmt.Errorf("%s: no persistent page_id mapping", page.Route)
		}
		page.ID = id
		out.Pages = append(out.Pages, page)
	}
	return out, nil
}

func indexPages(pages []Page, key func(Page) string) map[string][]int {
	out := map[string][]int{}
	for i, page := range pages {
		out[key(page)] = append(out[key(page)], i)
	}
	return out
}

func unmatchedIndex(pages []Page, used []bool, key func(Page) string) map[string][]int {
	out := map[string][]int{}
	for i, page := range pages {
		if !used[i] {
			out[key(page)] = append(out[key(page)], i)
		}
	}
	return out
}

func sameLocation(a, b Page) bool {
	return a.Route == b.Route && a.Article == b.Article && a.SectionNumber == b.SectionNumber
}

func compatibleStructure(a, b []byte) (bool, error) {
	sa, err := structuralSignature(a)
	if err != nil {
		return false, err
	}
	sb, err := structuralSignature(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(signatureBytes(sa), signatureBytes(sb)), nil
}

func signatureBytes(s signature) []byte {
	return []byte(strings.Join(s.Directives, "\x00") + "\x01" + strings.Join(s.LinkTargets, "\x00") + "\x02" + strings.Join(s.InlineCode, "\x00") + "\x03" + strings.Join(s.Preformatted, "\x00"))
}

func uniqueUnmatchedSignature(pages []Page, used []bool, target int) bool {
	want, err := structuralSignature(pages[target].Source)
	if err != nil {
		return false
	}
	wantBytes := signatureBytes(want)
	n := 0
	for i := range pages {
		if used[i] {
			continue
		}
		got, err := structuralSignature(pages[i].Source)
		if err == nil && bytes.Equal(wantBytes, signatureBytes(got)) {
			n++
		}
	}
	return n == 1
}

func unmatchedSignatureCounts(pages []Page, used []bool) map[string]int {
	out := map[string]int{}
	for i := range pages {
		if !used[i] {
			out[signatureKey(pages[i].Source)]++
		}
	}
	return out
}

func signatureKey(source []byte) string {
	sig, err := structuralSignature(source)
	if err != nil {
		return "error:" + err.Error()
	}
	return string(signatureBytes(sig))
}

func makeChange(kind ChangeKind, old, next *Page, reason string) PageChange {
	c := PageChange{Kind: kind, Reason: reason}
	if old != nil {
		c.PageID, c.OldArticle, c.OldSectionNumber, c.OldRoute = old.ID, old.Article, old.SectionNumber, old.Route
		c.OldSourceTitle, c.OldSourceSHA256 = old.SourceTitle, old.SourceSHA256
	}
	if next != nil {
		c.NewArticle, c.NewSectionNumber, c.NewRoute = next.Article, next.SectionNumber, next.Route
		c.NewSourceTitle, c.NewSourceSHA256 = next.SourceTitle, next.SourceSHA256
	}
	return c
}

func previewConditional(old, next []ConditionalPage) []PageChange {
	used := make([]bool, len(next))
	var changes []PageChange
	for i := range old {
		match := -1
		for j := range next {
			if !used[j] && old[i].SourceSHA256 == next[j].SourceSHA256 {
				if match != -1 {
					match = -2
					break
				}
				match = j
			}
		}
		if match >= 0 {
			used[match] = true
			o, n := conditionalPage(old[i]), conditionalPage(next[match])
			kind, reason := Unchanged, "conditional source hash is unchanged"
			if !sameLocation(o, n) {
				kind, reason = Moved, "conditional source hash matched at a different position"
			}
			changes = append(changes, makeChange(kind, &o, &n, reason))
			continue
		}
		o := conditionalPage(old[i])
		changes = append(changes, makeChange(Removed, &o, nil, "conditional page has no exact match"))
	}
	for i := range next {
		if !used[i] {
			n := conditionalPage(next[i])
			changes = append(changes, makeChange(Added, nil, &n, "new conditional page has no exact match"))
		}
	}
	return changes
}

func conditionalPage(p ConditionalPage) Page {
	return Page{Article: p.Article, SectionNumber: p.ConditionalIndex, Route: fmt.Sprintf("#%s/%d", p.Condition, p.ConditionalIndex), SourceTitle: p.SourceTitle, SourceSHA256: p.SourceSHA256, Source: p.Source}
}

func sortChanges(changes []PageChange) {
	sort.SliceStable(changes, func(i, j int) bool {
		a, b := changes[i], changes[j]
		if a.NewArticle != b.NewArticle {
			return a.NewArticle < b.NewArticle
		}
		if a.NewSectionNumber != b.NewSectionNumber {
			return a.NewSectionNumber < b.NewSectionNumber
		}
		if a.OldArticle != b.OldArticle {
			return a.OldArticle < b.OldArticle
		}
		if a.OldSectionNumber != b.OldSectionNumber {
			return a.OldSectionNumber < b.OldSectionNumber
		}
		return a.PageID < b.PageID
	})
}
