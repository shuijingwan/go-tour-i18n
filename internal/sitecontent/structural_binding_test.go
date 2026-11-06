package sitecontent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestStructuralBindingFamilies(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"inline", "Read [the guide](/fixed).\n", "Read [指南](/fixed).\n"},
		{"collapsed", "Read [the guide][].\n\n[the guide]: /fixed\n", "Read [指南][].\n\n[指南]: /fixed\n"},
		{"shortcut", "Read [the guide].\n\n[the guide]: /fixed\n", "Read [指南].\n\n[指南]: /fixed\n"},
		{"full", "Read [the guide][fixed].\n\n[fixed]: /fixed\n", "Read [指南][fixed].\n\n[fixed]: /fixed\n"},
		{"shared", "Read [the guide][] and [another guide][the guide].\n\n[the guide]: /fixed\n", "Read [指南][] and [另一份指南][指南].\n\n[指南]: /fixed\n"},
		{"unused", "Read [the guide](/fixed).\n\n[unused name]: /unused\n", "Read [指南](/fixed).\n\n[unused name]: /unused\n"},
		{"image", "See ![the guide][].\n\n[the guide]: /fixed.png\n", "See ![指南][].\n\n[指南]: /fixed.png\n"},
		{"literal", "Read [the guide].\n", "Read [指南].\n"},
		{"pseudo-code", "Read `![the guide][]` and [the guide](/fixed).\n", "Read `![the guide][]` and [指南](/fixed).\n"},
		{"code-bracket-in-label", "Read [the guide `]` text](/fixed).\n", "Read [指南 `]` text](/fixed).\n"},
		{"backtick-runs", "Read `` a ``` [fake][] a `` and [the guide](/fixed).\n", "Read `` a ``` [fake][] a `` and [指南](/fixed).\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, raw := formalFixture(t, "x.md", tc.source)
			checkFormal(t, d, raw, d.Units[0].Source, true)
			v := strings.NewReplacer("another guide", "另一份指南", "the guide", "指南").Replace(d.Units[0].Source)
			got := checkFormal(t, d, raw, v, true)
			if string(got) != tc.want {
				t.Fatalf("got=%q want=%q", got, tc.want)
			}
			for _, p := range d.Units[0].Protected {
				if p.Binding != nil {
					checkFormal(t, d, raw, strings.Replace(v, p.Token, strings.Replace(p.Raw, "fixed", "evil", 1), 1), false)
				}
			}
			if tc.name != "literal" {
				for _, bad := range []string{"指南]outside", "指南][other", "指南\n\n[evil]: /changed"} {
					checkFormal(t, d, raw, strings.Replace(v, "指南", bad, 1), false)
				}
			}
		})
	}
}

func TestStructuralBindingHTMLAttributesAndBlocks(t *testing.T) {
	for _, tag := range []string{"p", "li", "div", "section", "blockquote", "dt", "dd", "td", "th", "figcaption", "summary", "button", "label"} {
		for _, extension := range []string{"html", "md"} {
			source := "<" + tag + ">Read <strong><a href=\"/fixed\" title=\"Natural title\">the guide</a></strong> with <code>x</code>.</" + tag + ">\n"
			d, raw := formalFixture(t, "x."+extension, source)
			if !strings.Contains(d.Units[0].Source, "Natural title") {
				t.Fatalf("locked visible attribute %s/%s", extension, tag)
			}
			v := strings.NewReplacer("Read", "阅读", "the guide", "指南", "Natural title", "自然标题", "with", "使用").Replace(d.Units[0].Source)
			got := checkFormal(t, d, raw, v, true)
			if !bytes.Contains(got, []byte(`title="自然标题"`)) || !bytes.Contains(got, []byte(`<code>x</code>`)) {
				t.Fatal(string(got))
			}
			checkFormal(t, d, raw, strings.Replace(v, "自然标题", `坏" onclick="x`, 1), false)
		}
	}
}

func TestStructuralBindingSoftBoundaryFreedom(t *testing.T) {
	d, raw := formalFixture(t, "x.md", "Read [the guide](/fixed)\n  with `go test`.\n")
	s := d.Slots[0]
	values := []string{"使用" + s.Protected[3].Token + "，参阅" + s.Protected[0].Token + "指南" + s.Protected[1].Token + "。" + s.Protected[2].Token}
	checkFormal(t, d, raw, formalTarget(d, values), true)
}

func TestStructuralBindingOtherRegisteredMarkdownFamilies(t *testing.T) {
	for _, source := range []string{
		"| Name | Meaning |\n| --- | --- |\n| Entry | Read [the guide](/fixed) |\n",
		"> Read **the guide** with `go test`.\n> More details follow.\n",
		"Read the guide.  \nMore details follow.\n",
		"# Read the guide {#fixed}\n",
	} {
		d, raw := formalFixture(t, "x.md", source)
		v := strings.NewReplacer("Name", "名称", "Meaning", "含义", "Entry", "条目", "Read", "阅读", "the guide", "指南", "More details follow.", "后续详述。").Replace(d.Units[0].Source)
		got := checkFormal(t, d, raw, v, true)
		if strings.Contains(string(got), "the guide") || strings.Contains(string(got), "Name") {
			t.Fatal("language remained opaque", string(got))
		}
	}
}

func TestStructuralBindingHistoricalLiteralReader(t *testing.T) {
	// Absence is the old reader, not an instruction to infer new semantics.
	d, raw := formalFixture(t, "x.md", "Read [the guide][].\n\n[the guide]: /fixed\n")
	for i := range d.Slots {
		for j := range d.Slots[i].Protected {
			d.Slots[i].Protected[j].Binding = nil
		}
	}
	old := d
	old.Units = d.Slots
	old.UnitScope = ""
	d, err := FormalPage(old, raw, 1)
	if err != nil {
		t.Fatal(err)
	}
	v := strings.Replace(d.Units[0].Source, "the guide", "指南", 1)
	got := checkFormal(t, d, raw, v, true)
	if string(got) != "Read [指南][].\n\n[the guide]: /fixed\n" {
		t.Fatal("historical bytes reinterpreted", string(got))
	}
	b, _ := json.Marshal(d)
	if bytes.Contains(b, []byte(`"binding"`)) {
		t.Fatal("old serialization changed")
	}
}

func TestStructuralBindingFrozenFIPS140(t *testing.T) {
	root := repositoryRoot(t)
	path := "_content/doc/security/fips140.md"
	rawBytes, e := os.ReadFile(root + "/" + path)
	if e != nil {
		t.Fatal(e)
	}
	if digest(rawBytes) != "c171065cde5e7a0a9597d89e59bb9e8cb3e614ff096e032b3580ebf2709902e4" {
		t.Fatal("frozen canonical source changed")
	}
	d, e := ParseWorkflowDocument(Source{Path: path, Kind: "page", Route: "/doc/security/fips140", SHA256: digest(rawBytes)}, rawBytes, 40)
	if e != nil {
		t.Fatal(e)
	}
	v := strings.Replace(d.Units[0].Source, "validation certificate", "验证证书", 1)
	got := checkFormal(t, *d, rawBytes, v, true)
	if !bytes.Equal(markdownBindingShapeWithTitles(rawBytes, false), markdownBindingShapeWithTitles(got, false)) {
		t.Fatal("canonical destination changed")
	}
	if !bytes.Contains(got, []byte("[CMVP 验证证书][]")) || !bytes.Contains(got, []byte("[CMVP 验证证书]:")) {
		t.Fatalf("label not independently translatable: %s", got)
	}
}

func TestStructuralBindingHistoricalCorpus(t *testing.T) {
	p := os.Getenv("GO_TOUR_REPRESENTATION_BEFORE")
	if p == "" {
		p = "testdata/protected-page-before.json.gz"
	}
	b, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	if strings.HasSuffix(p, ".gz") {
		z, e := gzip.NewReader(bytes.NewReader(b))
		if e != nil {
			t.Fatal(e)
		}
		b, e = io.ReadAll(z)
		if e != nil {
			t.Fatal(e)
		}
		if e = z.Close(); e != nil {
			t.Fatal(e)
		}
	}
	var pages []struct{ Document *Document }
	if e = json.Unmarshal(b, &pages); e != nil {
		t.Fatal(e)
	}
	count := 0
	root := repositoryRoot(t)
	var snapshot i18n.QualityCheckSnapshotManifest
	if os.Getenv("GO_TOUR_REPRESENTATION_BEFORE") != "" {
		b, e = os.ReadFile(filepath.Join(root, "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-014/manifest.json"))
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(b, &snapshot); e != nil {
			t.Fatal(e)
		}
	}
	rebased := 0
	matched := 0
	for _, p := range pages {
		if p.Document == nil || !strings.Contains(p.Document.Path, "/doc/") && p.Document.Path != "_content/learn/index.md" {
			continue
		}
		d := *p.Document
		text, e := restoreBlock(d.Units[0], d.Units[0].Source)
		if e != nil {
			t.Fatal(e)
		}
		a := PageAdapter{d, []byte(text)}
		got, e := a.Restore(d.Units[0].Source)
		if e != nil || !bytes.Equal(got, []byte(text)) {
			t.Fatalf("historical %s: %v", d.Path, e)
		}
		if e := validateDocumentRepresentations([]Document{d}, false); e != nil {
			t.Fatal(e)
		}
		if _, _, e := a.ProtectedInput(&i18n.Glossary{}); e != nil {
			t.Fatal(e)
		}
		for _, s := range snapshot.Units {
			if s.UnitID != d.Units[0].ID {
				continue
			}
			matched++
			vb, e := os.ReadFile(filepath.Join(root, s.ValidationPath))
			if e != nil || digest(vb) != s.ValidationSHA256 {
				t.Fatal("frozen validation SHA", e)
			}
			var v struct {
				RawResponsePath string `json:"raw_response_path"`
				Attempt         int    `json:"attempt"`
				Status          string `json:"status"`
			}
			if e = json.Unmarshal(vb, &v); e != nil || v.Attempt != s.Attempt || v.Status != "passed" {
				t.Fatal("frozen validation attempt", e)
			}
			raw, e := os.ReadFile(filepath.Join(root, "data/retranslation-runs/zh-CN", s.SelectedBatchID, v.RawResponsePath))
			if e != nil {
				t.Fatal(e)
			}
			candidate, e := os.ReadFile(filepath.Join(root, s.CandidatePath))
			if e != nil || digest(candidate) != s.CandidateSHA256 {
				t.Fatal("frozen candidate SHA", e)
			}
			got, e := a.Restore(string(raw))
			if e != nil || !bytes.Equal(append(bytes.TrimRight(got, "\n"), '\n'), candidate) {
				t.Fatalf("old raw/candidate %s: %v", s.UnitID, e)
			}
			next, e := ParseWorkflowDocument(documentSource(d), a.Source, d.StableIndex)
			if e != nil {
				t.Fatal(e)
			}
			if next.Units[0].ID != s.UnitID {
				if digest([]byte(next.Units[0].Source)) == digest([]byte(d.Units[0].Source)) {
					t.Fatal("replacement lacks protected-input byte drift", d.Path)
				}
				current := PageAdapter{*next, a.Source}
				baseline, e := current.RebaseRevisionPredecessor(a, raw)
				if e != nil {
					t.Fatalf("rebase %s: %v", d.Path, e)
				}
				after, e := current.Restore(string(baseline))
				if e != nil || !bytes.Equal(got, after) {
					t.Fatal("rebase changed reviewed bytes", d.Path, e)
				}
				rebased++
			}
		}
		count++
	}
	if count != 49 {
		t.Fatal(count)
	}
	if len(snapshot.Units) > 0 && matched != 49 {
		t.Fatal("historical exact-set not matched", matched)
	}
	t.Logf("OLD_REPRESENTATION_COMPATIBILITY=PASS historical source=%d qc-014 raw/candidate=%d rebased=%d", count, len(snapshot.Units), rebased)
}

// The frozen shell reader retains its old inventory. New exports use the
// semantic-role policy tested separately; this is historical compatibility.
func TestRepresentationKnownBlockers(t *testing.T) {
	raw := []byte("---\ntitle: Example\nsummary: Example summary.\ntemplate: true\n---\n<p>Move <code>x</code> using <code>y</code>.</p>\n")
	s, e := ExtractShellLanguageSource("_content/index.md", raw)
	if e != nil {
		t.Fatal(e)
	}
	codeBoundaries := 0
	for _, p := range s.Page.Units[0].Protected {
		if p.Role == "boundary" && strings.Contains(p.Raw, "<code>") {
			codeBoundaries++
		}
	}
	if codeBoundaries != 2 {
		t.Fatal("shell inline-boundary blocker changed; reassess audit", codeBoundaries)
	}
}

func TestTeachingFenceLanguageAuthority(t *testing.T) {
	for _, language := range []string{"", "go", "golang", "sh", "shell", "bash", "json", "yaml", "text"} {
		source := "# Example\n\n```" + language + "\n// Teach the reader about this operation.\nvar value = 1\n```\n"
		d, raw := formalFixture(t, "x.md", source)
		goFence := language == "" || language == "go" || language == "golang"
		if strings.Contains(d.Units[0].Source, "Teach the reader") != goFence {
			t.Fatal("fence comment language misclassified", language)
		}
		if got := checkFormal(t, d, raw, d.Units[0].Source, true); !bytes.Equal(got, raw) {
			t.Fatal("source bytes changed", language)
		}
	}
}

func TestTeachingFenceHistoricalEligibility(t *testing.T) {
	raw := []byte("```sh\n// Teach the reader about this operation.\nvar value = 1\n```\n")
	var slot Unit
	e := teachingSpansMode(raw, 0, true, false, func(a, z int, _ string) {
		text, tokens, e := protectBlock(raw[a:z], "teaching-comment")
		if e != nil {
			t.Fatal(e)
		}
		sh := unitSourceIdentity(text, tokens)
		locator := "x.md#teaching-comment:" + sh + ":1"
		slot = Unit{ID: digest([]byte(UnitContract + "\n" + locator)), Locator: locator, Source: text, SourceSHA: sh, Start: a, End: z, Protected: tokens}
	}, nil)
	if e != nil || slot.Source == "" {
		t.Fatal("missing historical comment eligibility", e)
	}
	d, e := FormalPage(Document{Contract: UnitContract, Path: "x.md", Kind: "page", SourceSHA: digest(raw), Units: []Unit{slot}}, raw, 1)
	if e != nil {
		t.Fatal(e)
	}
	got := checkFormal(t, d, raw, formalTarget(d, []string{"为读者讲解此操作。"}), true)
	if !bytes.Contains(got, []byte("// 为读者讲解此操作。")) {
		t.Fatal("historical eligibility reinterpreted")
	}
}

func TestStructuralBindingStandaloneReferenceRoundtrip(t *testing.T) {
	raw := []byte("Read [the guide][].\n\n[the guide]: /fixed\n")
	s := Source{Path: "x.md", Kind: "page", SHA256: digest(raw)}
	d, e := ParseDocument(s, raw)
	if e != nil {
		t.Fatal(e)
	}
	_, e = Reconstruct(s, raw, map[string]string{d.Units[0].ID: strings.Replace(d.Units[0].Source, "the guide", "指南", 1)})
	if e != nil {
		t.Fatal(e)
	}
}

func TestStructuralBindingCandidateBindingValidation(t *testing.T) {
	d, raw := formalFixture(t, "x.md", "Read [the guide][].\n\n[the guide]: /fixed\n")
	good := []byte("阅读[指南][the guide]。\n\n[the guide]: /fixed\n")
	if e := validatePageStructure(d, raw, good); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{
		"阅读[指南][]。\n\n[the guide]: /fixed\n",
		"阅读[指南][other]。\n\n[the guide]: /fixed\n",
		"阅读[指南][the guide]。\n\n[the guide]: /evil\n",
	} {
		if validatePageStructure(d, raw, []byte(bad)) == nil {
			t.Fatal("candidate binding drift accepted", bad)
		}
	}
	// An unused definition must not capture previously literal bracket prose.
	d, raw = formalFixture(t, "x.md", "Read [unavailable].\n\n[available]: /fixed\n")
	checkFormal(t, d, raw, strings.Replace(d.Units[0].Source, "unavailable", "available", 1), false)
}

func TestStructuralBindingAdapterAuthorityDependency(t *testing.T) {
	root := repositoryRoot(t)
	tmp := t.TempDir()
	for _, path := range []string{"go.mod", "go.sum", "internal/i18n/glossary.go", "internal/i18n/glossary_context.go", "internal/i18n/glossary_compatibility.go", "internal/i18n/glossary_semantics.go", "internal/i18n/translation_authorization.go", "internal/i18n/bridge_successor.go"} {
		b, e := os.ReadFile(filepath.Join(root, path))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.MkdirAll(filepath.Dir(filepath.Join(tmp, path)), 0755); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(tmp, path), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	for _, name := range []string{"comment_inventory.go", "render_input.go", "shell_page_replacement.go", "translation_adapter.go", "shell_reconstruction.go", "shell_language_source.go", "shell_inline.go", "page_replacement.go", "shellyaml/parse.go", "formal_page.go", "units.go", "linguistic.go", "semantic_role.go", "semantic_go_packages.go", "page_structure.go", "structural_binding.go", "markdown_references.go", "teaching.go", "emphasis.go"} {
		path := filepath.Join("internal/sitecontent", name)
		b, e := os.ReadFile(filepath.Join(root, path))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.MkdirAll(filepath.Dir(filepath.Join(tmp, path)), 0755); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(tmp, path), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	a := func() (string, string) {
		p, e := (PageAdapter{}).ReattestationAuthority(tmp)
		if e != nil {
			t.Fatal(e)
		}
		s, e := (ShellPageAdapter{}).ReattestationAuthority(tmp)
		if e != nil {
			t.Fatal(e)
		}
		return p, s
	}
	p, s := a()
	path := filepath.Join(tmp, "internal/sitecontent/structural_binding.go")
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, append(b, []byte("\n// Dependency mutation fixture.\n")...), 0600); e != nil {
		t.Fatal(e)
	}
	np, ns := a()
	if p == np || s == ns {
		t.Fatal("shared binding mutation absent from adapter authority")
	}
	path = filepath.Join(tmp, "internal/i18n/bridge_successor.go")
	b, e = os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, append(b, []byte("\n// Historical decoder policy mutation fixture.\n")...), 0600); e != nil {
		t.Fatal(e)
	}
	bp, bs := a()
	if bp == np || bs == ns {
		t.Fatal("historical bridge decoder mutation absent from machine authority")
	}
}

func BenchmarkStructuralBindingFIPS140Restore(b *testing.B) {
	root, e := filepath.Abs("../..")
	if e != nil {
		b.Fatal(e)
	}
	path := "_content/doc/security/fips140.md"
	raw, e := os.ReadFile(filepath.Join(root, path))
	if e != nil {
		b.Fatal(e)
	}
	d, e := ParseWorkflowDocument(Source{Path: path, Kind: "page", SHA256: digest(raw)}, raw, 40)
	if e != nil {
		b.Fatal(e)
	}
	f, e := os.Open("testdata/protected-page-before.json.gz")
	if e != nil {
		b.Fatal(e)
	}
	defer f.Close()
	z, e := gzip.NewReader(f)
	if e != nil {
		b.Fatal(e)
	}
	defer z.Close()
	var pages []struct {
		Path     string
		Document *Document
	}
	if e = json.NewDecoder(z).Decode(&pages); e != nil {
		b.Fatal(e)
	}
	var old *Document
	for _, p := range pages {
		if p.Path == path {
			old = p.Document
		}
	}
	if old == nil {
		b.Fatal("missing frozen before representation")
	}
	for _, tc := range []struct {
		name string
		doc  *Document
	}{{"old", old}, {"new", d}} {
		b.Run(tc.name, func(b *testing.B) {
			a := PageAdapter{*tc.doc, raw}
			text := tc.doc.Units[0].Source
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, e := a.Restore(text); e != nil {
					b.Fatal(e)
				}
			}
		})
	}
}
