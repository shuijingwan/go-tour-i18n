package sitecontent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestHomepageInlineRepresentation(t *testing.T) {
	for _, body := range []string{
		`<p>Use <code>x</code> here.</p>`,
		`<p>Read <a href="/fixed">the guide</a> here.</p>`,
		`<p>Use <code>x</code>, then <a href="/fixed">the guide</a>.</p>`,
		`<p>Use <strong><a href="/fixed">the guide</a></strong>, then <code>x</code> and <code>x</code>.</p>`,
		`<p>Read <i class="material-icons" aria-hidden="true">arrow_forward</i><svg><path d="fixed"></svg><a href="/fixed">the guide</a>.</p>`,
		`<p>Read <a href="/first" title="First tooltip">first guide</a> then <a href="/fixed" title="Second tooltip">the guide</a>.</p>`,
	} {
		t.Run(body, func(t *testing.T) {
			raw := []byte("---\ntitle: Example\nsummary: Example summary.\ntemplate: true\n---\n" + body + "\n")
			s, err := ExtractShellPageRepresentation("_content/index.md", raw)
			if err != nil {
				t.Fatal(err)
			}
			a := ShellPageAdapter{*s, raw}
			for _, p := range s.Page.Units[0].Protected {
				if p.Role == "boundary" && (strings.Contains(p.Raw, "<code>") || strings.Contains(p.Raw, "href=")) {
					t.Fatal("inline identity still in boundary")
				}
			}
			values := map[string]string{}
			for _, slot := range s.Slots {
				v := slot.Unit.Source
				if slot.Role == "inline-prose" {
					v = strings.NewReplacer("Use ", "", "Read ", "", "the guide", "指南", " here.", "", ", then ", "，", " and ", "、", ".", "。").Replace(v)
					v = "（" + v + "）请使用。"
					type region struct{ a, z int }
					runs := []region{}
					for _, p := range slot.Unit.Protected {
						if p.Role == "open" && p.Raw == "" {
							for _, q := range slot.Unit.Protected {
								if q.Role == "close" && q.Pair == p.Pair {
									runs = append(runs, region{strings.Index(v, p.Token), strings.Index(v, q.Token) + len(q.Token)})
								}
							}
						}
					}
					if len(runs) == 2 {
						first, second := runs[0], runs[1]
						v = v[:first.a] + v[second.a:second.z] + v[first.z:second.a] + v[first.a:first.z] + v[second.z:]
					}
					// Complete inline atoms may move past surrounding language.
					for _, p := range slot.Unit.Protected {
						if strings.HasPrefix(p.Raw, "<code>") {
							v = strings.Replace(v, p.Token, "", 1) + "先用" + p.Token + "。"
							break
						}
					}
				}
				values[slot.Unit.ID] = v
			}
			target, err := reconstructShellSlots(*s, raw, values)
			if err != nil {
				t.Fatal(err)
			}
			if err = a.Validate(target, &i18n.Glossary{}); err != nil {
				t.Fatal(err)
			}
			ordered := []string{}
			for _, slot := range s.Slots {
				ordered = append(ordered, values[slot.Unit.ID])
			}
			viaAdapter, err := a.Restore(formalTarget(*s.Page, ordered))
			if err != nil || !bytes.Equal(viaAdapter, target) {
				t.Fatal("formal Homepage adapter", err)
			}
			if bytes.Contains(target, []byte("/fixed")) {
				if a.Validate(bytes.Replace(target, []byte("/fixed"), []byte("/evil"), 1), &i18n.Glossary{}) == nil {
					t.Fatal("URL mutation accepted")
				}
			}
			old, err := ExtractShellLanguageSource("_content/index.md", raw)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = (ShellPageAdapter{*old, raw}).Restore(old.Page.Units[0].Source); err != nil {
				t.Fatal("old reader", err)
			}
		})
	}
}

func TestMarkdownInlineTooltipRepresentation(t *testing.T) {
	for _, title := range []string{`"Natural tooltip"`, `'Natural tooltip'`, `(Natural tooltip)`, `"Natural \"tooltip\""`, "\"Natural\ntooltip\""} {
		d, raw := formalFixture(t, "x.md", "Read [the guide](/fixed "+title+").\n")
		if !strings.Contains(d.Units[0].Source, "Natural") {
			t.Fatal("title frozen", title)
		}
		v := strings.NewReplacer("the guide", "指南", "Natural", "自然").Replace(d.Units[0].Source)
		got := checkFormal(t, d, raw, v, true)
		if !bytes.Contains(got, []byte("自然")) {
			t.Fatal(string(got))
		}
		if validatePageStructure(d, raw, bytes.Replace(got, []byte("/fixed"), []byte("/evil"), 1)) == nil {
			t.Fatal("URL mutation accepted")
		}
		checkFormal(t, d, raw, strings.Replace(v, "自然", `bad"') [evil](/evil) "`, 1), false)
	}
}

func TestMarkdownReferenceTooltipRepresentation(t *testing.T) {
	for _, title := range []string{`"Natural tooltip"`, `'Natural tooltip'`, `(Natural tooltip)`, `"Natural \"tooltip\""`, "\"Natural\ntooltip\""} {
		for _, usage := range []string{"[the guide][]", "[the guide]", "[the guide][fixed]"} {
			key := "the guide"
			if strings.Contains(usage, "[fixed]") {
				key = "fixed"
			}
			d, raw := formalFixture(t, "x.md", "Read "+usage+".\n\n["+key+"]: /fixed "+title+"\n")
			if !strings.Contains(d.Units[0].Source, "Natural") {
				t.Fatal("reference title frozen", title)
			}
			v := strings.NewReplacer("the guide", "指南", "Natural", "自然").Replace(d.Units[0].Source)
			got := checkFormal(t, d, raw, v, true)
			if !bytes.Contains(got, []byte("自然")) {
				t.Fatal(string(got))
			}
			if validatePageStructure(d, raw, bytes.Replace(got, []byte("/fixed"), []byte("/evil"), 1)) == nil {
				t.Fatal("URL mutation accepted")
			}
			checkFormal(t, d, raw, strings.Replace(v, "自然", `bad"') [evil](/evil) "`, 1), false)
		}
	}
}

func TestHistoricalTooltipAndPB1Readers(t *testing.T) {
	for _, tc := range []struct {
		source, close string
		binding       *StructuralBinding
	}{
		{"Read [the guide](/fixed \"Natural tooltip\").\n", `](/fixed "Natural tooltip")`, nil},
		{"Read [the guide][].\n\n[the guide]: /fixed \"Natural tooltip\"\n", "][]", nil},
		{"Read [the guide][].\n\n[the guide]: /fixed\n", "][]", &StructuralBinding{Version: 1, Kind: "markdown-link", Reference: "the guide", Label: "the guide", Implicit: true, Destination: "/fixed"}},
	} {
		raw := []byte(tc.source)
		u := Unit{Source: "Read ⟦P0⟧the guide⟦P1⟧.", Start: 0, End: len("Read [the guide" + tc.close + "."), Protected: []ProtectedToken{{Token: "⟦P0⟧", Raw: "[", Role: "open", Pair: "link"}, {Token: "⟦P1⟧", Raw: tc.close, Role: "close", Pair: "link", Binding: tc.binding}}}
		u.SourceSHA = unitSourceIdentity(u.Source, u.Protected)
		u.Locator = "x.md#markdown/Paragraph:" + u.SourceSHA
		u.ID = digest([]byte(UnitContract + "\n" + u.Locator))
		d, err := FormalPage(Document{Contract: UnitContract, Path: "x.md", Kind: "page", SourceSHA: digest(raw), Units: []Unit{u}}, raw, 1)
		if err != nil {
			t.Fatal(err)
		}
		old := PageAdapter{d, raw}
		got, err := old.Restore(d.Units[0].Source)
		if err != nil || !bytes.Equal(got, raw) {
			t.Fatal("old reader changed", err)
		}
		current, err := ParseWorkflowDocument(Source{Path: "x.md", Kind: "page", SHA256: digest(raw)}, raw, 1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = (PageAdapter{*current, raw}).RebaseRevisionPredecessor(old, []byte(d.Units[0].Source+"\n")); err != nil {
			t.Fatal("byte-exact old predecessor", err)
		}
	}
}

func TestReferenceLabelsMayShareTranslatedSpelling(t *testing.T) {
	d, raw := formalFixture(t, "x.md", "Read [first guide][] and [second guide][].\n\n[first guide]: /first\n[second guide]: /second\n")
	v := strings.NewReplacer("first guide", "指南", "second guide", "指南").Replace(d.Units[0].Source)
	got := checkFormal(t, d, raw, v, true)
	if !bytes.Contains(got, []byte("[指南][first guide]")) || !bytes.Contains(got, []byte("[指南][second guide]")) {
		t.Fatal("colliding language labels were bound together", string(got))
	}
}
