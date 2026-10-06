package i18n

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const GlossaryCorpusSchema = "go-learning/glossary-source-corpus/v1"
const GlossaryCorpusPath = "data/glossary-source-corpus.json"

type LanguageContext struct {
	ID        string   `json:"id"`
	Text      string   `json:"text"`
	Protected []string `json:"protected"`
}
type LanguageContributor struct {
	ID         string            `json:"id"`
	Parser     string            `json:"parser"`
	Package    string            `json:"package"`
	Path       string            `json:"path"`
	Route      string            `json:"route"`
	Structured bool              `json:"structured"`
	SourceSHA  string            `json:"projected_source_sha256"`
	Contexts   []LanguageContext `json:"contexts"`
}
type GlossarySourceCorpus struct {
	Schema       string                     `json:"schema"`
	Contributors []LanguageContributor      `json:"contributors"`
	Sources      []GlossaryArchiveReference `json:"source_files"`
	Identity     string                     `json:"identity_sha256"`
}

func CorpusIdentity(c GlossarySourceCorpus) string {
	c.Sources = nil
	c.Identity = ""
	return sum(mustJSON(c))
}
func ContributorIdentity(c LanguageContributor) string { c.SourceSHA = ""; return sum(mustJSON(c)) }
func ValidateGlossaryCorpus(c *GlossarySourceCorpus) error {
	if c.Schema != GlossaryCorpusSchema || len(c.Contributors) == 0 || c.Identity != CorpusIdentity(*c) {
		return fmt.Errorf("invalid unified corpus identity")
	}
	last := ""
	for _, v := range c.Contributors {
		if v.ID <= last || v.SourceSHA != ContributorIdentity(v) || len(v.Contexts) == 0 {
			return fmt.Errorf("invalid corpus contributor")
		}
		last = v.ID
		switch v.Parser {
		case "present-visible/v1", "go-comment-visible/v1", "ui-visible/v1", "article-header/v1", "canonical-description/v1", "go-learning/content-units/v2":
		default:
			return fmt.Errorf("unknown corpus parser %s", v.Parser)
		}
		if err := validateGenerationInstallPath(v.Path); err != nil {
			return err
		}
		prev := ""
		for _, x := range v.Contexts {
			if x.ID <= prev || strings.TrimSpace(x.Text) == "" || !utf8.ValidString(x.Text) || strings.ContainsRune(x.Text, 0) {
				return fmt.Errorf("invalid corpus context")
			}
			prev = x.ID
		}
	}
	return nil
}
func LoadUnifiedGlossaryCorpus(root string) (*GlossarySourceCorpus, error) {
	b, err := readCompatibilityFile(root, GlossaryCorpusPath)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(b) {
		return nil, fmt.Errorf("invalid corpus UTF-8")
	}
	var c GlossarySourceCorpus
	if err := contentidentity.StrictJSON(b, &c); err != nil {
		return nil, err
	}
	if err := ValidateGlossaryCorpus(&c); err != nil {
		return nil, err
	}
	last := ""
	for _, r := range c.Sources {
		if r.Path <= last || !validSHA256(r.SHA256) {
			return nil, fmt.Errorf("invalid corpus source registry")
		}
		last = r.Path
		b, err := readCompatibilityFile(root, r.Path)
		if err != nil {
			return nil, err
		}
		if sum(b) != r.SHA256 {
			return nil, fmt.Errorf("corpus source stale: %s", r.Path)
		}
	}
	return &c, nil
}

// Tour keeps its own source parsers. This adapter projects language only;
// code never becomes an eligible Example context.
func TourLanguageContributors(root string, catalog *Catalog) ([]LanguageContributor, error) {
	units, _, _, err := localeWorkflowUnitList(catalog)
	if err != nil {
		return nil, err
	}
	out := []LanguageContributor{}
	add := func(id, parser, path, route, text string, structured bool) {
		lines := []string{}
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				lines = append(lines, line)
			}
		}
		c := LanguageContributor{ID: id, Parser: parser, Package: "tour-v1", Path: path, Route: route, Structured: structured, Contexts: []LanguageContext{{ID: id, Text: strings.Join(lines, "\n"), Protected: []string{}}}}
		c.SourceSHA = ContributorIdentity(c)
		out = append(out, c)
	}
	for _, u := range units {
		p := "present-visible/v1"
		if u.Kind == UnitKindExample {
			p = "go-comment-visible/v1"
		}
		text, _, err := compatibilityVisiblePair(GlossaryCompatibilityContext{Parser: p, Source: string(u.Source), Target: string(u.Source)})
		if err != nil {
			return nil, err
		}
		route := ""
		if u.Kind == UnitKindPage {
			page, err := catalog.Page(u.ID)
			if err != nil {
				return nil, err
			}
			route = "/tour" + page.Route
		}
		add("tour:"+string(u.Kind)+":"+u.ID, p, u.SourcePath, route, text, false)
	}
	catalogUI, err := ui.LoadFromFS("en", os.DirFS(filepath.Join(root, "internal/tour/ui")))
	if err != nil {
		return nil, err
	}
	keys := []string{}
	for key := range catalogUI.Messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		text, err := ui.VisibleMessageText(catalogUI.Messages[key])
		if err != nil {
			return nil, err
		}
		add("ui:"+key, "ui-visible/v1", "internal/tour/ui/en.json", "", text, true)
	}
	articles := catalogArticleSet(catalog)
	names := []string{}
	for name := range articles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := "_content/tour/" + name
		t, s, err := sourceArticleHeader(filepath.Join(root, p))
		if err != nil {
			return nil, err
		}
		add("article:"+name, "article-header/v1", p, "", t+"\n"+s, true)
	}
	seo, err := LoadCourseSourceDescriptions(root, catalog)
	if err != nil {
		return nil, err
	}
	for _, p := range seo.Pages {
		add("seo:"+p.PageID, "canonical-description/v1", courseSourceDescriptionsRelativePath, p.Route, p.Description, false)
	}
	return out, nil
}

// SafeTeachingCommentSpans reuses Tour's conservative Go-fragment analyzer.
// Unknown fragments remain entirely protected, including block comments.
type TeachingSpan struct{ Start, End int }

func SafeTeachingCommentSpans(code string) []TeachingSpan {
	a := analyzePreformattedGo(code)
	out := []TeachingSpan{}
	if a.Static {
		return out
	}
	for _, c := range a.Comments {
		if !c.Translatable {
			continue
		}
		pos := c.BodyStart
		for _, i := range c.Identifiers {
			if pos < i.Start {
				out = append(out, TeachingSpan{pos, i.Start})
			}
			pos = i.End
		}
		if pos < c.BodyEnd {
			out = append(out, TeachingSpan{pos, c.BodyEnd})
		}
	}
	return out
}
