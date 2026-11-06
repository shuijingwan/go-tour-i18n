package i18n

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/tools/present"
)

// TourAuditUnit is a current ready canonical TU, selected by the existing
// workflow membership/status readers. It is not a new authorization or judgment.
type TourAuditUnit struct {
	Unit   *TranslationUnit
	Owner  string
	Target []byte
}

func ReadTourLanguageAuditUnits(root, locale string, catalog *Catalog) ([]TourAuditUnit, error) {
	if catalog == nil || len(catalog.Adapted) != 0 || len(catalog.TranslationScope) != 0 {
		return nil, fmt.Errorf("complete native Tour catalog required")
	}
	dir, err := localeWorkflowDirectory(root, locale)
	if err != nil {
		return nil, err
	}
	statuses, err := ReadStatuses(filepath.Join(dir, "status.tsv"))
	if err != nil {
		return nil, err
	}
	if err := checkStatuses(locale, catalog, statuses); err != nil {
		return nil, err
	}
	byID := map[string]*Status{}
	for i := range statuses {
		byID[statuses[i].UnitID] = &statuses[i]
	}
	units, _, _, err := localeWorkflowUnitList(catalog)
	if err != nil {
		return nil, err
	}
	out := []TourAuditUnit{}
	for _, u := range units {
		target, err := loadReadyTranslationUnitCandidate(root, catalog, u, locale, byID[u.ID])
		if err != nil {
			return nil, err
		}
		out = append(out, TourAuditUnit{u, byID[u.ID].CandidatePath, target})
	}
	return out, nil
}

// TourAuditFragment contains either renderer-owned HTML or a source-selected
// teaching-comment payload. Coordinates refer to the owning TU; rendered prose
// has -1 coordinates. Identities come from source Go syntax/comment bindings.
type TourAuditFragment struct {
	Locator, Context, HTML, Text string
	Start, End                   int
	Identities                   []string
}

// TourLanguageAuditFragments reuses the locked Present parser/Style semantics
// and source-selected comment inventory. Target language never chooses whether
// a code comment is natural language, a directive, or protected code.
func TourLanguageAuditFragments(root string, catalog *Catalog, u *TranslationUnit, target []byte) ([]TourAuditFragment, []TourAuditFragment, error) {
	if u == nil || catalog == nil || u.Adapter != nil || sum(u.Source) != u.SourceSHA256 || !utf8.Valid(u.Source) || !utf8.Valid(target) {
		return nil, nil, fmt.Errorf("native Tour source/representation identity required")
	}
	actual, err := catalog.Unit(u.ID)
	if err != nil || actual.Adapter != nil || actual.Kind != u.Kind || actual.SourcePath != u.SourcePath || actual.SourceSHA256 != u.SourceSHA256 || !bytes.Equal(actual.Source, u.Source) {
		return nil, nil, fmt.Errorf("Tour audit unit is not the exact catalog member: %s", u.ID)
	}
	var sourceBlocks, targetBlocks []CommentCodeBlock
	sourceView, targetView := []TourAuditFragment{}, []TourAuditFragment{}
	switch u.Kind {
	case UnitKindPage:
		page, err := catalog.Page(u.ID)
		if err != nil {
			return nil, nil, err
		}
		for i, raw := range [][]byte{u.Source, target} {
			doc, err := parsePresentPage(root, page.Article, raw)
			if err != nil {
				return nil, nil, err
			}
			view, err := tourAuditSection(doc.Sections[0], "section/1")
			if err != nil {
				return nil, nil, err
			}
			if i == 0 {
				sourceView = view
			} else {
				targetView = view
			}
		}
		sourceBlocks, targetBlocks = TourPageCommentBlocks(u.Source), TourPageCommentBlocks(target)
	case UnitKindExample:
		// The native validator proves full immutable Go code/delimiters independently
		// of comment prose. Empty glossary adds no locale terminology assumptions.
		if err := validateGoExampleWithGlossary(root, u, "", target, &Glossary{}); err != nil {
			return nil, nil, err
		}
		sourceBlocks = []CommentCodeBlock{AnalyzeCommentCode(string(u.Source), "go", "example/1", true)}
		targetBlocks = []CommentCodeBlock{AnalyzeCommentCode(string(target), "go", "example/1", true)}
	default:
		return nil, nil, fmt.Errorf("unsupported Tour unit kind %q", u.Kind)
	}
	examples := []Example{}
	for _, e := range catalog.Examples {
		for _, id := range e.ReferencedBy {
			if id == u.ID {
				examples = append(examples, e)
				break
			}
		}
	}
	resolved := make([]CommentCodeBlock, len(sourceBlocks))
	for i, b := range sourceBlocks {
		resolved[i] = SourceCommentBlockDisposition(b, "", examples)
	}
	if err := ValidateSourceCommentBlockSet(sourceBlocks, resolved); err != nil {
		return nil, nil, err
	}
	targetBlocks = SourceSelectedTargetCommentBlocks(resolved, targetBlocks)
	rows, failures := CompareCommentBlocks("audit", "", resolved, targetBlocks)
	if err := ValidateCommentCoverage(rows, failures, false); err != nil {
		return nil, nil, err
	}
	for i, s := range resolved {
		analysis := analyzePreformattedGo(s.Code)
		for j, c := range s.Comments {
			if c.Kind != "teaching-comment" || c.Policy != "translatable" {
				continue
			}
			tc := targetBlocks[i].Comments[j]
			ids := []string{}
			for _, id := range referencedCommentIdentifiers(c.Body, 0, analysis.CodeIdentifiers) {
				ids = append(ids, id.Value)
			}
			sort.Strings(ids)
			for k, pair := range []struct {
				b CommentCodeBlock
				c CodeComment
			}{{s, c}, {targetBlocks[i], tc}} {
				f := TourAuditFragment{Locator: fmt.Sprintf("%s/comment/%d", s.Coordinate, j+1), Context: "teaching-comment", Text: pair.c.Body, Start: pair.b.Start + pair.c.Start, End: pair.b.Start + pair.c.End, Identities: ids}
				if k == 0 {
					sourceView = append(sourceView, f)
				} else {
					targetView = append(targetView, f)
				}
			}
		}
	}
	return sourceView, targetView, nil
}

func tourAuditSection(s present.Section, locator string) ([]TourAuditFragment, error) {
	// The actual action template escapes section titles without applying Style.
	out := []TourAuditFragment{{Locator: locator + "/title", Context: "heading", HTML: "<h2>" + html.EscapeString(s.Title) + "</h2>", Start: -1, End: -1}}
	for i, e := range s.Elem {
		loc := fmt.Sprintf("%s/element/%d", locator, i+1)
		value := ""
		switch v := e.(type) {
		case present.Text:
			if v.Pre {
				continue
			} // source-owned comment inventory handles only language payloads
			lines := []string{}
			for _, line := range v.Lines {
				lines = append(lines, string(present.Style(line)))
			}
			value = "<p>" + strings.Join(lines, "\n") + "</p>"
		case present.List:
			value = "<ul>"
			for _, b := range v.Bullet {
				value += "<li>" + string(present.Style(b)) + "</li>"
			}
			value += "</ul>"
		case present.Section:
			nested, err := tourAuditSection(v, loc)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
			continue
		case present.Link:
			value = `<p><a href="` + html.EscapeString(v.URL.String()) + `">` + string(present.Style(v.Label)) + `</a></p>`
		case present.HTML:
			value = string(v.HTML)
		case present.Code, present.Image, present.Video, present.Iframe:
			continue // code/media addresses and machine attributes are not language
		default:
			return nil, fmt.Errorf("unsupported Tour audit element %T", e)
		}
		out = append(out, TourAuditFragment{Locator: loc, HTML: value, Start: -1, End: -1})
	}
	return out, nil
}

// ReadTourArticleAuditHeaders uses the same formal article membership and
// normalized source header lines as the localized article projection.
func ReadTourArticleAuditHeaders(root string, c *Catalog) (map[string]ArticleMetadata, error) {
	out := map[string]ArticleMetadata{}
	for article := range catalogArticleSet(c) {
		raw, err := os.ReadFile(filepath.Join(root, "_content", "tour", article))
		if err != nil {
			return nil, err
		}
		lines := strings.SplitN(string(normalizeLF(raw)), "\n", 3)
		if len(lines) < 3 {
			return nil, fmt.Errorf("incomplete article header %s", article)
		}
		out[article] = ArticleMetadata{Article: article, Title: lines[0], Subtitle: lines[1]}
	}
	return out, nil
}
