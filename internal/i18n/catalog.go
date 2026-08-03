package i18n

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/present"
)

var ArticleOrder = []string{
	"welcome.article",
	"basics.article",
	"flowcontrol.article",
	"moretypes.article",
	"methods.article",
	"generics.article",
	"concurrency.article",
}

type Page struct {
	ID            string
	Article       string
	SectionNumber int
	Route         string
	SourceTitle   string
	SourceSHA256  string
	PlayCount     int
	ImageCount    int
	Source        []byte
}

type ConditionalPage struct {
	Article          string
	Condition        string
	ConditionalIndex int
	SourceTitle      string
	SourceSHA256     string
	Source           []byte
}

type Catalog struct {
	Pages       []Page
	Conditional []ConditionalPage
}

type publishedSection struct {
	ID     string
	Source []byte
}

func BuildCatalog(root string) (*Catalog, error) {
	return buildCatalog(root, true)
}

// BuildSourceCatalog parses a prospective upstream tree without assuming the
// fixed baseline page counts. The publication projection assigns known
// persistent IDs where the project has explicitly frozen them; reconciliation
// remains responsible for preserving IDs across later upstream changes.
func BuildSourceCatalog(root string) (*Catalog, error) {
	return buildCatalog(root, false)
}

func buildCatalog(root string, fixedShape bool) (*Catalog, error) {
	var catalog Catalog
	for _, article := range ArticleOrder {
		path := filepath.Join(root, "_content", "tour", article)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", article, err)
		}
		data = normalizeLF(data)
		pages, conditional, err := splitArticle(data, article)
		if err != nil {
			return nil, err
		}
		published, err := projectPublishedSections(article, data, pages, conditional)
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(article, ".article")
		for i, section := range published {
			source := section.Source
			if err := parseSinglePage(root, article, source); err != nil {
				return nil, fmt.Errorf("%s/%d: %w", name, i+1, err)
			}
			catalog.Pages = append(catalog.Pages, Page{
				ID:            section.ID,
				Article:       article,
				SectionNumber: i + 1,
				Route:         fmt.Sprintf("/%s/%d", name, i+1),
				SourceTitle:   pageTitle(source),
				SourceSHA256:  sum(source),
				PlayCount:     countDirective(source, ".play"),
				ImageCount:    countDirective(source, ".image"),
				Source:        source,
			})
		}
		for i, source := range conditional {
			if err := parseSinglePage(root, article, source); err != nil {
				return nil, fmt.Errorf("%s appengine/%d: %w", name, i+1, err)
			}
			catalog.Conditional = append(catalog.Conditional, ConditionalPage{
				Article:          article,
				Condition:        "appengine",
				ConditionalIndex: i + 1,
				SourceTitle:      pageTitle(source),
				SourceSHA256:     sum(source),
				Source:           source,
			})
		}
	}
	if err := validateCatalog(&catalog, fixedShape); err != nil {
		return nil, err
	}
	return &catalog, nil
}

// projectPublishedSections is the single source of truth for the ordered,
// translatable page projection. Conditional sources remain separately audited
// in Catalog.Conditional even when a clean Section is also published here.
func projectPublishedSections(article string, articleSource []byte, standalone, conditional [][]byte) ([]publishedSection, error) {
	name := strings.TrimSuffix(article, ".article")
	if article != "welcome.article" {
		out := make([]publishedSection, 0, len(standalone))
		for i, source := range standalone {
			out = append(out, publishedSection{ID: fmt.Sprintf("%s/%d", name, i+1), Source: source})
		}
		return out, nil
	}
	if len(standalone) != 3 || len(conditional) != 2 {
		return nil, fmt.Errorf("welcome publication projection requires 3 standalone and 2 conditional sections, got %d/%d", len(standalone), len(conditional))
	}
	remoteArticle := projectConditionalContent(articleSource, "appengine")
	remotePages, _, err := splitArticle(remoteArticle, article)
	if err != nil {
		return nil, err
	}
	if len(remotePages) != 5 || bytes.Contains(remotePages[0], []byte("#appengine:")) || bytes.Contains(remotePages[0], []byte("your computer.")) || !bytes.Contains(remotePages[0], []byte("a remote server.")) {
		return nil, fmt.Errorf("welcome remote publication branch is incomplete")
	}
	return []publishedSection{
		{ID: "welcome/1", Source: remotePages[0]},
		{ID: "welcome/2", Source: standalone[1]},
		{ID: "welcome/4", Source: conditional[0]},
		{ID: "welcome/5", Source: conditional[1]},
		{ID: "welcome/3", Source: standalone[2]},
	}, nil
}

// projectConditionalContent applies the upstream conditional-line semantics
// without changing the checked-in article. A conditional line replaces the
// next non-blank fallback line; full conditional sections are retained after
// their prefixes are removed.
func projectConditionalContent(data []byte, condition string) []byte {
	prefix := "#" + condition + ":"
	dropFallback := false
	var out strings.Builder
	for _, line := range strings.SplitAfter(string(normalizeLF(data)), "\n") {
		plain := strings.TrimSuffix(line, "\n")
		if strings.HasPrefix(plain, prefix) {
			projected := strings.TrimPrefix(plain, prefix)
			projected = strings.TrimPrefix(projected, " ")
			out.WriteString(projected)
			if strings.HasSuffix(line, "\n") {
				out.WriteByte('\n')
			}
			dropFallback = true
			continue
		}
		if dropFallback {
			dropFallback = false
			if plain != "" {
				continue
			}
		}
		out.WriteString(line)
	}
	return []byte(out.String())
}

func projectPublishedArticle(data []byte, article string) ([]byte, error) {
	data = normalizeLF(data)
	standalone, conditional, err := splitArticle(data, article)
	if err != nil {
		return nil, err
	}
	published, err := projectPublishedSections(article, data, standalone, conditional)
	if err != nil {
		return nil, err
	}
	if article != "welcome.article" {
		return data, nil
	}
	start := bytes.Index(data, []byte("\n* "))
	if start < 0 {
		return nil, fmt.Errorf("%s: first present section not found", article)
	}
	start++
	out := append([]byte(nil), data[:start]...)
	for _, section := range published {
		out = append(out, section.Source...)
	}
	return out, nil
}

func normalizeLF(data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	return bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
}

func splitArticle(data []byte, article string) (pages, conditional [][]byte, err error) {
	lines := strings.SplitAfter(string(data), "\n")
	var current []string
	inConditional := false
	flush := func() {
		if len(current) == 0 {
			return
		}
		source := []byte(strings.Join(current, ""))
		if len(source) == 0 || source[len(source)-1] != '\n' {
			source = append(source, '\n')
		}
		pages = append(pages, source)
		current = nil
	}
	for _, line := range lines {
		plain := strings.TrimSuffix(line, "\n")
		if article == "welcome.article" && strings.HasPrefix(plain, "#appengine:") {
			if strings.HasPrefix(plain, "#appengine: * ") {
				flush()
				inConditional = true
			}
			continue
		}
		if strings.HasPrefix(plain, "* ") {
			inConditional = false
			flush()
			current = append(current, line)
		} else if len(current) > 0 && !inConditional {
			current = append(current, line)
		}
	}
	flush()

	if article == "welcome.article" {
		conditional, err = splitConditional(data)
		if err != nil {
			return nil, nil, err
		}
	}
	return pages, conditional, nil
}

func splitConditional(data []byte) ([][]byte, error) {
	var result [][]byte
	var current []string
	flush := func() {
		if len(current) == 0 {
			return
		}
		source := []byte(strings.Join(current, ""))
		if source[len(source)-1] != '\n' {
			source = append(source, '\n')
		}
		result = append(result, source)
		current = nil
	}
	for _, line := range strings.SplitAfter(string(data), "\n") {
		plain := strings.TrimSuffix(line, "\n")
		if !strings.HasPrefix(plain, "#appengine:") {
			continue
		}
		stripped := strings.TrimPrefix(plain, "#appengine:")
		stripped = strings.TrimPrefix(stripped, " ")
		if strings.HasPrefix(stripped, "* ") {
			flush()
		}
		if len(current) > 0 || strings.HasPrefix(stripped, "* ") {
			current = append(current, stripped+"\n")
		}
	}
	flush()
	return result, nil
}

func parseSinglePage(root, article string, source []byte) error {
	present.PlayEnabled = true
	ctx := &present.Context{ReadFile: func(name string) ([]byte, error) {
		clean := filepath.Clean(filepath.FromSlash(name))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("unsafe referenced path %q", name)
		}
		return os.ReadFile(filepath.Join(root, "_content", "tour", clean))
	}}
	wrapped := append([]byte("Tour page\n\n"), source...)
	doc, err := ctx.Parse(bytes.NewReader(wrapped), article, 0)
	if err != nil {
		return fmt.Errorf("present parse: %w", err)
	}
	if len(doc.Sections) != 1 {
		return fmt.Errorf("top-level sections = %d, want 1", len(doc.Sections))
	}
	return nil
}

func pageTitle(source []byte) string {
	line, _, _ := bytes.Cut(source, []byte("\n"))
	return strings.TrimPrefix(string(line), "* ")
}

func countDirective(source []byte, directive string) int {
	n := 0
	s := bufio.NewScanner(bytes.NewReader(source))
	for s.Scan() {
		if strings.HasPrefix(s.Text(), directive+" ") {
			n++
		}
	}
	return n
}

func sum(source []byte) string {
	h := sha256.Sum256(source)
	return hex.EncodeToString(h[:])
}

func validateCatalog(c *Catalog, fixedShape bool) error {
	if fixedShape && len(c.Pages) != 103 {
		return fmt.Errorf("published pages = %d, want 103", len(c.Pages))
	}
	if fixedShape && len(c.Conditional) != 2 {
		return fmt.Errorf("conditional pages = %d, want 2", len(c.Conditional))
	}
	ids, routes := map[string]bool{}, map[string]bool{}
	plays, images := 0, 0
	for _, p := range c.Pages {
		if strings.ContainsAny(p.SourceTitle, "\t\r\n") {
			return fmt.Errorf("%s: source_title contains a tab or newline", p.ID)
		}
		if !sha256RE.MatchString(p.SourceSHA256) {
			return fmt.Errorf("%s: invalid source_sha256", p.ID)
		}
		if ids[p.ID] || routes[p.Route] {
			return fmt.Errorf("duplicate page_id or route: %s %s", p.ID, p.Route)
		}
		ids[p.ID], routes[p.Route] = true, true
		plays += p.PlayCount
		images += p.ImageCount
	}
	for _, p := range c.Conditional {
		if strings.ContainsAny(p.SourceTitle, "\t\r\n") || !sha256RE.MatchString(p.SourceSHA256) {
			return fmt.Errorf("conditional %d: invalid title or source_sha256", p.ConditionalIndex)
		}
	}
	if fixedShape && (plays != 93 || images != 1) {
		return fmt.Errorf("directive totals: play=%d image=%d, want 93/1", plays, images)
	}
	return nil
}

var sha256RE = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (c *Catalog) Page(id string) (*Page, error) {
	for i := range c.Pages {
		if c.Pages[i].ID == id {
			return &c.Pages[i], nil
		}
	}
	return nil, fmt.Errorf("unknown page_id %q", id)
}

var pageHeader = []string{"page_id", "article", "section_number", "route", "source_title", "source_sha256", "play_count", "image_count"}
var conditionalHeader = []string{"article", "condition", "conditional_index", "source_title", "source_sha256"}

// ReadCatalog reads the committed catalog, whose page_id column is the source
// of truth for persistent page identity.
func ReadCatalog(root string) (*Catalog, error) {
	pages, err := readTSV(filepath.Join(root, "data", "tour-pages.tsv"), pageHeader)
	if err != nil {
		return nil, err
	}
	conditional, err := readTSV(filepath.Join(root, "data", "tour-conditional-pages.tsv"), conditionalHeader)
	if err != nil {
		return nil, err
	}
	c := &Catalog{}
	for line, record := range pages {
		section, err := strconv.Atoi(record[2])
		if err != nil {
			return nil, fmt.Errorf("page catalog line %d: invalid section_number: %w", line+2, err)
		}
		play, err := strconv.Atoi(record[6])
		if err != nil {
			return nil, fmt.Errorf("page catalog line %d: invalid play_count: %w", line+2, err)
		}
		image, err := strconv.Atoi(record[7])
		if err != nil {
			return nil, fmt.Errorf("page catalog line %d: invalid image_count: %w", line+2, err)
		}
		c.Pages = append(c.Pages, Page{ID: record[0], Article: record[1], SectionNumber: section, Route: record[3], SourceTitle: record[4], SourceSHA256: record[5], PlayCount: play, ImageCount: image})
	}
	for line, record := range conditional {
		index, err := strconv.Atoi(record[2])
		if err != nil {
			return nil, fmt.Errorf("conditional catalog line %d: invalid conditional_index: %w", line+2, err)
		}
		c.Conditional = append(c.Conditional, ConditionalPage{Article: record[0], Condition: record[1], ConditionalIndex: index, SourceTitle: record[3], SourceSHA256: record[4]})
	}
	if err := validateCatalog(c, false); err != nil {
		return nil, fmt.Errorf("committed catalog: %w", err)
	}
	return c, nil
}

func readTSV(path string, header []string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = '\t'
	r.FieldsPerRecord = len(header)
	got, err := r.Read()
	if err != nil {
		return nil, err
	}
	for i := range header {
		if got[i] != header[i] {
			return nil, fmt.Errorf("%s: header column %d=%q, want %q", path, i+1, got[i], header[i])
		}
	}
	var records [][]string
	for {
		record, err := r.Read()
		if err == io.EOF {
			return records, nil
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
}

func WriteCatalog(c *Catalog, pages io.Writer, conditional io.Writer) error {
	w := csv.NewWriter(pages)
	w.Comma = '\t'
	if err := w.Write(pageHeader); err != nil {
		return err
	}
	for _, p := range c.Pages {
		if err := w.Write([]string{p.ID, p.Article, strconv.Itoa(p.SectionNumber), p.Route, p.SourceTitle, p.SourceSHA256, strconv.Itoa(p.PlayCount), strconv.Itoa(p.ImageCount)}); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	w = csv.NewWriter(conditional)
	w.Comma = '\t'
	if err := w.Write(conditionalHeader); err != nil {
		return err
	}
	for _, p := range c.Conditional {
		if err := w.Write([]string{p.Article, p.Condition, strconv.Itoa(p.ConditionalIndex), p.SourceTitle, p.SourceSHA256}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func CatalogBytes(c *Catalog) ([]byte, []byte, error) {
	var pages, conditional bytes.Buffer
	if err := WriteCatalog(c, &pages, &conditional); err != nil {
		return nil, nil, err
	}
	return pages.Bytes(), conditional.Bytes(), nil
}

func CheckCatalogFiles(root string, c *Catalog) error {
	wantPages, wantConditional, err := CatalogBytes(c)
	if err != nil {
		return err
	}
	return compareFile(filepath.Join(root, "data", "tour-pages.tsv"), wantPages, "page catalog", filepath.Join(root, "data", "tour-conditional-pages.tsv"), wantConditional, "conditional catalog")
}

func compareFile(path1 string, want1 []byte, label1, path2 string, want2 []byte, label2 string) error {
	var failures []string
	for _, item := range []struct {
		path, label string
		want        []byte
	}{{path1, label1, want1}, {path2, label2, want2}} {
		got, err := os.ReadFile(item.path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", item.label, err))
			continue
		}
		if !bytes.Equal(got, item.want) {
			failures = append(failures, fmt.Sprintf("%s differs from generated content", item.label))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}
