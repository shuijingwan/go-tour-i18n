package i18n

import (
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const GlossaryCorpusSchema = "go-learning/glossary-source-corpus/v1"
const GlossaryCorpusSchemaV2 = "go-learning/glossary-source-corpus/v2"
const GlossaryCorpusPath = "data/glossary-source-corpus.json"

type ContributorPurpose string

const (
	ContributorPurposeActiveContent        ContributorPurpose = "active-content"
	ContributorPurposeTerminologyDiscovery ContributorPurpose = "terminology-discovery"
)

type LanguageContext struct {
	ID        string   `json:"id"`
	Text      string   `json:"text"`
	Protected []string `json:"protected"`
}
type LanguageContributor struct {
	ID         string             `json:"id"`
	Parser     string             `json:"parser"`
	Package    string             `json:"package"`
	Path       string             `json:"path"`
	Route      string             `json:"route"`
	Structured bool               `json:"structured"`
	Purpose    ContributorPurpose `json:"purpose,omitempty"`
	SourceSHA  string             `json:"projected_source_sha256"`
	Contexts   []LanguageContext  `json:"contexts"`
}
type GlossarySourceCorpus struct {
	PackageAuthorities []GlossaryCorpusPackageAuthority `json:"package_authorities,omitempty"`
	Schema             string                           `json:"schema"`
	Contributors       []LanguageContributor            `json:"contributors"`
	Sources            []GlossaryArchiveReference       `json:"source_files"`
	Identity           string                           `json:"identity_sha256"`
}

// Source registration only; candidate/QC/finalization remain in the shared
// TranslationPackageDescriptor lifecycle. The current source projector must
// reproduce this binding, and the corpus inventory freezes its exact receipt.
type GlossaryCorpusPackageAuthority struct {
	Package     string                   `json:"package"`
	Authority   GlossaryArchiveReference `json:"authority"`
	SourcePaths []string                 `json:"source_paths"`
}

func CorpusIdentity(c GlossarySourceCorpus) string {
	c.Sources = nil
	c.Identity = ""
	return sum(mustJSON(c))
}
func ContributorIdentity(c LanguageContributor) string { c.SourceSHA = ""; return sum(mustJSON(c)) }
func EffectiveContributorPurpose(c LanguageContributor) ContributorPurpose {
	if c.Purpose == "" {
		return ContributorPurposeActiveContent
	}
	return c.Purpose
}
func ValidateGlossaryCorpus(c *GlossarySourceCorpus) error {
	if (c.Schema != GlossaryCorpusSchema && c.Schema != GlossaryCorpusSchemaV2) || len(c.Contributors) == 0 || c.Identity != CorpusIdentity(*c) {
		return fmt.Errorf("invalid unified corpus identity")
	}
	last := ""
	registered := map[string]map[string]bool{}
	for _, p := range c.PackageAuthorities {
		if p.Package == "" || validateGenerationInstallPath(p.Package) != nil || strings.ContainsAny(p.Package, "/\\\x00") || p.Package <= last || p.Package == "tour-v1" || p.Package == "site-v2-shell" || p.Package == "learn-docs-v1" || len(p.SourcePaths) == 0 || validateGenerationInstallPath(p.Authority.Path) != nil || !strings.HasPrefix(p.Authority.Path, "data/content-package-adoptions/receipts/") || !validSHA256(p.Authority.SHA256) {
			return fmt.Errorf("invalid source package corpus authority")
		}
		last = p.Package
		found := false
		for _, ref := range c.Sources {
			found = found || ref == p.Authority
		}
		if !found {
			return fmt.Errorf("source package receipt missing from corpus inventory")
		}
		paths := map[string]bool{}
		previous := ""
		for _, name := range p.SourcePaths {
			if name <= previous || validateGenerationInstallPath(name) != nil {
				return fmt.Errorf("invalid source package exact paths")
			}
			paths[name] = true
			previous = name
		}
		registered[p.Package] = paths
	}
	last = ""
	used := map[string]map[string]bool{}
	for _, v := range c.Contributors {
		if v.ID <= last || v.SourceSHA != ContributorIdentity(v) || len(v.Contexts) == 0 {
			return fmt.Errorf("invalid corpus contributor")
		}
		last = v.ID
		purpose := EffectiveContributorPurpose(v)
		if c.Schema == GlossaryCorpusSchema {
			if v.Purpose != "" || purpose != ContributorPurposeActiveContent {
				return fmt.Errorf("v1 corpus contributor purpose must remain implicit active-content")
			}
		} else if v.Purpose == "" {
			return fmt.Errorf("v2 corpus contributor purpose is required")
		}
		switch purpose {
		case ContributorPurposeActiveContent:
			switch v.Parser {
			case "present-visible/v1", "go-comment-visible/v1", "ui-visible/v1", "article-header/v1", "canonical-description/v1", "go-learning/content-units/v2":
			case "go-learning/shell-language-source/v1":
				page := v.Path == "_content/index.md"
				structured := v.Path == "_content/site.tmpl" || v.Path == "_content/menus.yaml" || v.Path == "_content/resources.yaml" || v.Path == "_content/testimonials.yaml"
				if v.Package != "site-v2-shell" || (!page && !structured) || v.Structured != structured || (page && v.Route != "/") || (structured && v.Route != "") {
					return fmt.Errorf("invalid shell successor contributor")
				}
			default:
				return fmt.Errorf("unknown active-content corpus parser %s", v.Parser)
			}
			if v.Package != "tour-v1" && v.Package != "site-v2-shell" && v.Package != "learn-docs-v1" {
				if !registered[v.Package][v.Path] || v.ID != "source:"+v.Path || v.Parser != "go-learning/content-units/v2" || v.Structured || v.Route == "" {
					return fmt.Errorf("unknown active-content corpus package %s", v.Package)
				}
				if used[v.Package] == nil {
					used[v.Package] = map[string]bool{}
				}
				used[v.Package][v.Path] = true
			}
		case ContributorPurposeTerminologyDiscovery:
			if c.Schema != GlossaryCorpusSchemaV2 || v.Structured || v.Route != "" || v.Package != "terminology-discovery" {
				return fmt.Errorf("invalid terminology-discovery contributor isolation")
			}
			switch v.Parser {
			case "markdown-visible/v1", "html-visible/v1", "go-doc-visible/v1":
			default:
				return fmt.Errorf("unknown terminology-discovery corpus parser %s", v.Parser)
			}
		default:
			return fmt.Errorf("unknown corpus contributor purpose %s", v.Purpose)
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
	for pkg, paths := range registered {
		for name := range paths {
			if !used[pkg][name] {
				return fmt.Errorf("registered package contributor exact set incomplete")
			}
		}
	}
	return nil
}
func LoadUnifiedGlossaryCorpus(root string) (*GlossarySourceCorpus, error) {
	return LoadGlossaryCorpusAt(root, GlossaryCorpusPath)
}

// LoadGlossaryCorpusAt verifies both current and immutable archived corpora.
// Only the retired source ZIP has a storage-resolution layer; every other
// source keeps its original physical path and exact SHA validation.
func LoadGlossaryCorpusAt(root, corpusPath string) (*GlossarySourceCorpus, error) {
	if _, err := os.Lstat(filepath.Join(root, "data/.corpus-successor-pending")); !os.IsNotExist(err) {
		return nil, fmt.Errorf("mutation=unknown: corpus successor publication pending")
	}
	c, b, err := readGlossaryCorpusAt(root, corpusPath)
	if err != nil {
		return nil, err
	}
	corpusRef := contentidentity.Reference{Path: corpusPath, SHA256: sum(b)}
	last := ""
	for _, r := range c.Sources {
		if r.Path <= last || !validSHA256(r.SHA256) {
			return nil, fmt.Errorf("invalid corpus source registry")
		}
		last = r.Path
		if r.Path == sourcestorage.LegacyZIPPath {
			required, err := CorpusStorageRequirements(c)
			if err != nil {
				return nil, err
			}
			if err := sourcestorage.CheckCorpusReference(root, corpusRef, c.Identity, contentidentity.Reference{Path: r.Path, SHA256: r.SHA256}, required); err != nil {
				return nil, err
			}
			continue
		}
		b, err := readCompatibilityFile(root, r.Path)
		if err != nil {
			return nil, err
		}
		if sum(b) != r.SHA256 {
			return nil, fmt.Errorf("corpus source stale: %s", r.Path)
		}
	}
	return c, nil
}

func readGlossaryCorpusAt(root, corpusPath string) (*GlossarySourceCorpus, []byte, error) {
	b, err := readCompatibilityFile(root, corpusPath)
	if err != nil {
		return nil, nil, err
	}
	if !utf8.Valid(b) {
		return nil, nil, fmt.Errorf("invalid corpus UTF-8")
	}
	var c GlossarySourceCorpus
	if err := contentidentity.StrictJSON(b, &c); err != nil {
		return nil, nil, err
	}
	if err := ValidateGlossaryCorpus(&c); err != nil {
		return nil, nil, err
	}
	if corpusPath != GlossaryCorpusPath && corpusPath != "data/glossary-corpus-history/"+sum(b)+".json" {
		return nil, nil, fmt.Errorf("non-canonical archived corpus path")
	}
	return &c, b, nil
}

// CorpusStorageRequirements follows the registered parser/source contract,
// never directory globs or the presence of unrelated mirror files.
func CorpusStorageRequirements(c *GlossarySourceCorpus) ([]string, error) {
	if err := ValidateGlossaryCorpus(c); err != nil {
		return nil, err
	}
	paths := []string{}
	seen := map[string]bool{}
	for _, v := range c.Contributors {
		if v.Package != "learn-docs-v1" {
			continue
		}
		if v.Parser != "go-learning/content-units/v2" || EffectiveContributorPurpose(v) != ContributorPurposeActiveContent || seen[v.Path] || !strings.HasPrefix(v.Path, "_content/") {
			return nil, fmt.Errorf("ambiguous corpus storage consumer")
		}
		seen[v.Path] = true
		paths = append(paths, v.Path)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("ZIP corpus reference has no exact registered consumer set")
	}
	sort.Strings(paths)
	return paths, nil
}

// CorpusStorageReference returns the declared historical SHA after strict
// storage verification. It never invents a new reference or semantic identity.
func CorpusStorageReference(root string) (GlossaryArchiveReference, error) {
	c, b, err := readGlossaryCorpusAt(root, GlossaryCorpusPath)
	if err != nil {
		return GlossaryArchiveReference{}, err
	}
	for _, ref := range c.Sources {
		if ref.Path == sourcestorage.LegacyZIPPath {
			required, err := CorpusStorageRequirements(c)
			if err != nil {
				return GlossaryArchiveReference{}, err
			}
			if err := sourcestorage.CheckCorpusReference(root, contentidentity.Reference{Path: GlossaryCorpusPath, SHA256: sum(b)}, c.Identity, contentidentity.Reference{Path: ref.Path, SHA256: ref.SHA256}, required); err != nil {
				return GlossaryArchiveReference{}, err
			}
			return ref, nil
		}
	}
	return GlossaryArchiveReference{}, fmt.Errorf("current corpus has no registered ZIP storage reference")
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

// SafeTeachingCommentBodies exposes complete comment coordinates and their
// source-bound identifiers. A Page keeps one linguistic slot per comment so
// atomic identifiers can follow the target language's word order.
type TeachingCommentBody struct {
	TeachingSpan
	Identifiers     []TeachingSpan
	CodeIdentifiers []string
}

func SafeTeachingCommentBodies(code string) []TeachingCommentBody {
	a := analyzePreformattedGo(code)
	if a.Static {
		return nil
	}
	ids := []string{}
	for id := range a.CodeIdentifiers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := []TeachingCommentBody{}
	for _, c := range a.Comments {
		if !c.Translatable {
			continue
		}
		body := TeachingCommentBody{TeachingSpan: TeachingSpan{c.BodyStart, c.BodyEnd}, CodeIdentifiers: ids}
		for _, id := range c.Identifiers {
			body.Identifiers = append(body.Identifiers, TeachingSpan{id.Start, id.End})
		}
		out = append(out, body)
	}
	return out
}

func SafeTeachingCommentSpans(code string) []TeachingSpan {
	out := []TeachingSpan{}
	for _, c := range SafeTeachingCommentBodies(code) {
		pos := c.Start
		for _, i := range c.Identifiers {
			if pos < i.Start {
				out = append(out, TeachingSpan{pos, i.Start})
			}
			pos = i.End
		}
		if pos < c.End {
			out = append(out, TeachingSpan{pos, c.End})
		}
	}
	return out
}
