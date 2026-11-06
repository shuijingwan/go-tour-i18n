package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
)

// Explicit diagnostic only: no lifecycle methods, generated language or receipts.
// The opt-in output records every registered physical Page, including Tour and
// Homepage. Source counts use the consuming parser, not a repository-wide grep.
func TestRegisteredPageRepresentationAudit(t *testing.T) {
	output := os.Getenv("GO_TOUR_REPRESENTATION_AUDIT")
	if output == "" {
		t.Skip("explicit diagnostic output required")
	}
	root := repositoryRoot(t)
	c, err := i18n.BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	beforePath := os.Getenv("GO_TOUR_REPRESENTATION_BEFORE")
	if beforePath == "" {
		if err := AttachRegisteredTranslationPackages(root, c); err != nil {
			t.Fatal(err)
		}
	}
	type page struct {
		ID, Path, Route, Representation, SourceSHA string
		Counts                                     map[string]int
		Document                                   *Document `json:",omitempty"`
		Code                                       []string  `json:",omitempty"`
	}
	pages := []page{}
	add := func(id, path, route, representation string, raw []byte, d *Document) {
		p := page{ID: id, Path: path, Route: route, Representation: representation, SourceSHA: digest(raw), Counts: map[string]int{}, Document: d}
		markup := bytes.Clone(raw)
		mask := func(a, z int) {
			for i := a; i < z; i++ {
				if markup[i] != '\n' {
					markup[i] = ' '
				}
			}
		}
		if d != nil {
			refs := map[string]int{}
			for _, s := range d.Slots {
				_, tail, _ := strings.Cut(s.Locator, "#")
				parts := strings.Split(tail, ":")
				p.Counts["slot/"+strings.Join(parts[:len(parts)-2], ":")]++
				for _, tok := range s.Protected {
					if strings.HasPrefix(tok.Raw, "{{") {
						p.Counts["template/action"]++
					}
					if tok.Role == "suffix" {
						p.Counts["markdown/explicit-heading-anchor"]++
					}
					if strings.HasPrefix(tok.Raw, "&") && strings.HasSuffix(tok.Raw, ";") {
						p.Counts["entity/atom"]++
					}
					if strings.HasPrefix(tok.Raw, "\\") {
						p.Counts["markdown/escape-atom"]++
					}
					if tok.Role == "atom" && strings.Contains(tok.Raw, "\n") {
						p.Counts["soft-or-hard-wrap"]++
					}
					if tok.Binding != nil {
						b := tok.Binding
						p.Counts["binding/"+b.Kind]++
						if b.Reference != "" {
							refs[b.Reference]++
						}
						if b.Implicit {
							if tok.Raw == "]" {
								p.Counts["reference/actual-shortcut"]++
							} else {
								p.Counts["reference/actual-collapsed"]++
							}
						} else if b.Reference != "" {
							p.Counts["reference/actual-full"]++
						}
					}
				}
			}
			if representation == "Site shell Page" {
				for _, tok := range d.Units[0].Protected {
					if tok.Role == "boundary" {
						p.Counts["shell/inline-code-in-boundary"] += strings.Count(tok.Raw, "<code>")
					}
				}
				z := html.NewTokenizer(bytes.NewReader(raw))
				pos, start := 0, -1
				flush := func(end int) {
					if start < 0 {
						return
					}
					count := 0
					for _, s := range d.Slots {
						if s.Start >= start && s.End <= end && strings.Contains(s.Locator, "/visible-text:") {
							count++
						}
					}
					if count > 1 {
						p.Counts["shell/mixed-inline-paragraph"]++
					}
					start = -1
				}
				for {
					typ := z.Next()
					a := pos
					pos += len(z.Raw())
					if typ == html.ErrorToken {
						flush(pos)
						break
					}
					if typ == html.StartTagToken || typ == html.EndTagToken {
						name, _ := z.TagName()
						if string(name) == "p" {
							flush(a)
							if typ == html.StartTagToken {
								start = pos
							}
						}
					}
				}
			}
			for _, count := range refs {
				if count > 1 {
					p.Counts["reference/shared-definition"]++
				}
			}
			p.Counts["reference/used-definition"] = len(refs)
		}
		if strings.HasSuffix(path, ".md") && representation != "Site shell Page" {
			offset, e := parseMetadata(raw, func(int, int, string) {})
			if e != nil && representation != "Site shell Page" {
				t.Fatal(e)
			}
			body := raw[offset:]
			pc := parser.NewContext()
			tree := goldmark.New(goldmark.WithExtensions(extension.Table)).Parser().Parse(gmtext.NewReader(body), parser.WithContext(pc))
			p.Counts["reference-definition"] = len(pc.References())
			_ = ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
				if !enter {
					return ast.WalkContinue, nil
				}
				p.Counts["markdown/"+n.Kind().String()]++
				if n.Kind() == ast.KindFencedCodeBlock || n.Kind() == ast.KindCodeBlock {
					if n.Lines().Len() > 0 {
						a := n.Lines().At(0).Start
						z := n.Lines().At(n.Lines().Len() - 1).Stop
						p.Code = append(p.Code, string(body[a:z]))
					}
					for i := 0; i < n.Lines().Len(); i++ {
						s := n.Lines().At(i)
						mask(offset+s.Start, offset+s.Stop)
					}
				}
				if n.Kind() == ast.KindCodeSpan {
					for c := n.FirstChild(); c != nil; c = c.NextSibling() {
						if x, ok := c.(*ast.Text); ok {
							mask(offset+x.Segment.Start, offset+x.Segment.Stop)
						}
					}
				}
				if x, ok := n.(*ast.Emphasis); ok {
					if x.Level == 2 {
						p.Counts["markdown/strong"]++
					} else {
						p.Counts["markdown/emphasis"]++
					}
				}
				if x, ok := n.(*ast.FencedCodeBlock); ok {
					p.Counts["fence/"+string(x.Language(body))]++
				}
				if x, ok := n.(*ast.Link); ok && len(x.Title) > 0 {
					p.Counts["markdown/link-title"]++
				}
				if x, ok := n.(*ast.Image); ok && len(x.Title) > 0 {
					p.Counts["markdown/image-title"]++
				}
				if x, ok := n.(*ast.Text); ok {
					if x.SoftLineBreak() {
						p.Counts["markdown/soft-line-break"]++
					}
					if x.HardLineBreak() {
						p.Counts["markdown/hard-line-break"]++
					}
				}
				return ast.WalkContinue, nil
			})
			// The bounded protector owns bracket coordinates. Definitions and
			// code are excluded by iterating only approved Markdown prose slots.
			if d != nil {
				for _, s := range d.Slots {
					if !strings.Contains(s.Locator, "#markdown/") {
						continue
					}
					for _, tok := range s.Protected {
						if tok.Role != "close" || !strings.HasPrefix(tok.Raw, "]") {
							continue
						}
						switch {
						case tok.Raw == "][]":
							p.Counts["reference/collapsed"]++
						case tok.Raw == "]":
							p.Counts["reference/shortcut-or-literal"]++
						case strings.HasPrefix(tok.Raw, "]["):
							p.Counts["reference/full"]++
						case strings.HasPrefix(tok.Raw, "]("):
							p.Counts["link/inline"]++
						}
					}
				}
			}
			for i := 0; i+1 < len(markup); i++ {
				if markup[i] == '\\' {
					if markup[i+1] == '<' {
						mask(i, i+2)
					}
					i++
				}
			}
		}
		if !strings.HasSuffix(path, ".article") {
			z := html.NewTokenizer(bytes.NewReader(markup))
			opaque := 0
			codeRegion := false
			tokenPos := 0
			for {
				typ := z.Next()
				tokenStart := tokenPos
				tokenPos += len(z.Raw())
				if typ == html.ErrorToken {
					break
				}
				if typ == html.TextToken && codeRegion {
					p.Code = append(p.Code, string(raw[tokenStart:tokenPos]))
				}
				if typ == html.EndTagToken {
					tag, _ := z.TagName()
					if strings.Contains(" pre code script style textarea var kbd samp tt ", " "+string(tag)+" ") && opaque > 0 {
						opaque--
						if opaque == 0 {
							codeRegion = false
						}
					}
					continue
				}
				if typ == html.StartTagToken || typ == html.SelfClosingTagToken {
					tag, more := z.TagName()
					if opaque > 0 {
						if typ == html.StartTagToken && strings.Contains(" pre code script style textarea var kbd samp tt ", " "+string(tag)+" ") {
							opaque++
						}
						continue
					}
					p.Counts["html/"+string(tag)]++
					if string(tag) == "pre" || string(tag) == "textarea" {
						codeRegion = true
					}
					if typ == html.StartTagToken && strings.Contains(" pre code script style textarea var kbd samp tt ", " "+string(tag)+" ") {
						opaque++
					}
					for more {
						k, v, m := z.TagAttr()
						more = m
						if string(k) == "alt" || string(k) == "title" || string(k) == "aria-label" {
							if len(v) > 0 {
								p.Counts["html/visible-attribute/"+string(k)]++
							}
						}
					}
				}
			}
		}
		pages = append(pages, p)
	}
	for _, p := range c.Pages {
		u, e := c.Unit(p.ID)
		if e != nil {
			t.Fatal(e)
		}
		add(p.ID, u.SourcePath, "/tour"+p.Route, "Tour present", u.Source, nil)
	}
	if beforePath != "" {
		b, e := os.ReadFile(beforePath)
		if e != nil {
			t.Fatal(e)
		}
		var before []page
		if e = json.Unmarshal(b, &before); e != nil {
			t.Fatal(e)
		}
		for _, p := range before {
			if p.Document == nil {
				u, e := c.Unit(p.ID)
				if e != nil || digest(u.Source) != p.SourceSHA {
					t.Fatalf("Tour canonical inventory drift: %s", p.ID)
				}
				continue
			}
			raw, e := os.ReadFile(filepath.Join(root, p.Path))
			if e != nil {
				t.Fatal(e)
			}
			if digest(raw) != p.SourceSHA {
				t.Fatalf("audit canonical source drift: %s", p.Path)
			}
			var d *Document
			if p.Representation == "Site shell Page" {
				s, e := ExtractShellPageRepresentation(p.Path, raw)
				if e != nil {
					t.Fatal(e)
				}
				d = s.Page
			} else {
				d, e = ParseWorkflowDocument(documentSource(*p.Document), raw, p.Document.StableIndex)
				if e != nil {
					t.Fatal(e)
				}
			}
			got, e := (PageAdapter{*d, raw}).Restore(d.Units[0].Source)
			if p.Representation == "Site shell Page" {
				s, err := ExtractShellPageRepresentation(p.Path, raw)
				if err != nil {
					t.Fatal(err)
				}
				got, e = (ShellPageAdapter{*s, raw}).Restore(d.Units[0].Source)
				if e != nil || !bytes.Equal(got, raw) {
					t.Fatal("Homepage roundtrip", e)
				}
			}
			if e != nil && p.Representation != "Site shell Page" {
				for _, s := range d.Slots {
					if _, e := restoreBlock(s, s.Source); e != nil {
						t.Logf("slot failure: %s %+v %v", s.Source, s.Protected, e)
					}
				}
				t.Fatal(p.Path, e)
			}
			if p.Representation != "Site shell Page" && !bytes.Equal(got, raw) {
				t.Fatal("source roundtrip", p.Path)
			}
			if p.Representation != "Site shell Page" {
				offset, e := parseMetadata(raw, func(int, int, string) {})
				if e != nil {
					t.Fatal(e)
				}
				e = teachingSpans(raw[offset:], offset, strings.HasSuffix(p.Path, ".md"), func(a, z int, _ string) {
					found := false
					for _, s := range d.Slots {
						if a >= s.Start && z <= s.End {
							found = true
						}
					}
					if !found {
						t.Fatalf("teaching body in immutable boundary %s %d:%d", p.Path, a, z)
					}
				}, nil)
				if e != nil {
					t.Fatal(e)
				}
			}
			add(d.Units[0].ID, p.Path, p.Route, p.Representation, raw, d)
		}
		if len(pages) != len(before) {
			t.Fatal("registered inventory changed during audit")
		}
	}
	for _, pkg := range c.TranslationPackages {
		for _, u := range pkg.Units {
			if u.Unit.Kind != i18n.UnitKindPage {
				continue
			}
			switch a := u.Adapter.(type) {
			case PageAdapter:
				d := a.Document
				add(u.Unit.ID, u.Unit.SourcePath, u.Route, "Learn/Docs Page", a.Source, &d)
			case ShellPageAdapter:
				add(u.Unit.ID, u.Unit.SourcePath, u.Route, "Site shell Page", a.Raw, a.Source.Page)
			default:
				t.Fatalf("unclassified registered Page adapter %T", a)
			}
		}
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Path+pages[i].ID < pages[j].Path+pages[j].ID })
	// SYS-08 exact-set closure in the actual complete registered corpus. Test
	// source coordinates, not a search limited to the translated locale.
	for _, fixture := range naturalRoleFixtures(t) {
		matched := false
		for _, p := range pages {
			if p.Path != fixture.Path || p.Document == nil {
				continue
			}
			for si, slot := range p.Document.Slots {
				if slot.Start != fixture.Old.Start || slot.End != fixture.Old.End {
					continue
				}
				matched = true
				for _, token := range slot.Protected {
					if token.Role == "atom" && token.Raw == fixture.Raw {
						t.Fatalf("SYS-08 current corpus: %s: %s", p.Path, fixture.Raw)
					}
				}
				values := make([]string, len(p.Document.Slots))
				for j, s := range p.Document.Slots {
					values[j] = s.Source
				}
				if !strings.Contains(slot.Source, fixture.Raw) {
					t.Fatal("missing editable occurrence", p.Path, fixture.Raw)
				}
				values[si] = strings.Replace(slot.Source, fixture.Raw, "可编辑文本", 1)
				raw, err := os.ReadFile(filepath.Join(root, p.Path))
				if err != nil {
					t.Fatal(err)
				}
				checkFormal(t, *p.Document, raw, formalTarget(*p.Document, values), true)
			}
		}
		if !matched {
			t.Fatal("SYS-08 exact occurrence missing from registered corpus", fixture.Path, fixture.Raw)
		}
	}
	if len(pages) != 153 {
		t.Fatal("registered Page exact-set changed", len(pages))
	}
	t.Log("SOURCE_SPELLING_FALSE_MACHINE_AFTER=0 exact source-coordinate mutations=16/16")
	b, e := json.MarshalIndent(pages, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Clean(output), append(b, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
	t.Logf("TOTAL_REGISTERED_PAGE_UNITS=%d", len(pages))
}
