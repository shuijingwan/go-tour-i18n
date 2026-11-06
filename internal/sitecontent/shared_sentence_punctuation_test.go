package sitecontent

import (
	"bytes"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Pure representation/reconstruction fixtures: no plan, publication, export,
// import or language lifecycle API is invoked, even in a disposable repository.
func TestSharedSentencePunctuationSiteTemplate(t *testing.T) {
	root := repositoryRoot(t)
	raw := mustSharedFixtureBytes(t, root, "_content/site.tmpl")
	before := digest(raw)
	base, err := ExtractShellLanguageSource("_content/site.tmpl", raw)
	if err != nil {
		t.Fatal(err)
	}
	var accessibility string
	for _, s := range base.Slots {
		for _, p := range s.Unit.Protected {
			if p.Raw == "activate/deactivate" {
				accessibility = s.Unit.ID
			}
		}
	}
	prior, _, err := sharedRepresentationDelta(*base, raw, []string{accessibility})
	if err != nil {
		t.Fatal("existing accessibility successor", err)
	}
	periods, err := shellSentencePeriods(*prior, raw)
	if err != nil || len(periods) != 2 {
		t.Fatal("expected exactly the two visible sentences", len(periods), err)
	}
	ids := []string{periods[0].Prefix, periods[1].Prefix}
	sort.Strings(ids)
	next, releases, err := sharedRepresentationRuleDelta(*prior, raw, ids, SharedSentencePunctuationRule)
	if err != nil || len(releases) != 0 || len(next.Slots) != len(prior.Slots)+2 || next.SourceSHA != before || next.StructureSHA == prior.StructureSHA {
		t.Fatal("bounded punctuation topology", err)
	}
	byID := map[string]ShellLanguageSlot{}
	values := map[string]string{}
	for _, s := range next.Slots {
		byID[s.Unit.ID], values[s.Unit.ID] = s, s.Unit.Source
	}
	for _, s := range prior.Slots {
		if !reflect.DeepEqual(s, byID[s.Unit.ID]) {
			t.Fatal("existing slot/protected representation changed", s.Unit.Locator)
		}
	}
	if got, err := reconstructShellSlots(*next, raw, values); err != nil || !bytes.Equal(got, raw) {
		t.Fatal("source must round-trip byte-exact", err)
	}
	for _, end := range []string{"Go Wiki</a>.", "golang.org/x/tools/gopls/doc</a>."} {
		at := bytes.Index(raw, []byte(end))
		if at < 0 || bytes.Count(raw, []byte(end)) != 1 {
			t.Fatal("missing exact source fixture", end)
		}
		at += len(end) - 1
		found := false
		for _, s := range next.Slots {
			if s.Unit.Start == at && s.Unit.End == at+1 && s.Role == "sentence-punctuation" && s.Unit.Source == "." && len(s.Unit.Protected) == 0 {
				values[s.Unit.ID], found = "。", true
			}
		}
		if !found {
			t.Fatal("sentence punctuation remains frozen", end)
		}
		t.Logf("EDITABLE_SENTENCE_END=%s RAW_BYTE_OFFSET=%d", end, at)
	}
	const accessible = "按 Enter 键可激活或停用下拉菜单"
	for _, s := range next.Slots {
		if string(raw[s.Unit.Start:s.Unit.End]) == "Press Enter to activate/deactivate dropdown" {
			values[s.Unit.ID] = accessible
		}
	}
	// A prospective dependency-shaped Document exercises the existing frozen
	// codecs only. It is not a receipt or adoption/currentness authority.
	ref := Reference{SharedRepresentationRoot + "receipts/" + strings.Repeat("a", 64) + ".json", strings.Repeat("b", 64)}
	doc := withSharedDocumentAuthority(shellDocument(*next), ref)
	expected := bytes.Replace(raw, []byte("Press Enter to activate/deactivate dropdown"), []byte(accessible), 1)
	for _, end := range []string{"Go Wiki</a>.", "golang.org/x/tools/gopls/doc</a>."} {
		expected = bytes.Replace(expected, []byte(end), []byte(strings.TrimSuffix(end, ".")+"。"), 1)
	}
	target, err := reconstructFrozenStructuredRepresentation(doc, raw, values)
	if err != nil || !bytes.Equal(target, expected) {
		t.Fatal("punctuation/accessibility reconstruction changed other bytes", err)
	}
	output := StructuredTarget{Path: "locales/fixture/structured/_content/site.tmpl", Slots: []Target{}}
	for _, s := range next.Slots {
		output.Slots = append(output.Slots, Target{ID: s.Unit.ID, Text: values[s.Unit.ID]})
	}
	if got, err := reconstructStructuredTarget(StructuredAsset{Document: &doc}, raw, output, &i18n.Glossary{Mandatory: map[string]string{}}); err != nil || !bytes.Equal(got, expected) {
		t.Fatal("structured slot exact-set/glossary/reconstruction", err)
	}
	if err := validateShellStructure(*next, raw, target); err != nil {
		t.Fatal(err)
	}
	if validateShellStructure(*prior, raw, target) == nil {
		t.Fatal("old representation silently acquired punctuation authority")
	}
	if _, err := reconstructStructuredRepresentation(shellDocument(*next), raw, values); err == nil {
		t.Fatal("legacy extractor implicitly adopted new topology")
	}
	// Selection stays on the exact structured asset; no scope/receipt/export is
	// created. The complete new Document, including punctuation, is its input.
	plan := StructuredPlan{Assets: []StructuredAsset{
		{Package: "site-v2-shell", Document: &doc, TargetPath: "locales/fixture/structured/_content/site.tmpl"},
		{Package: "site-v2-shell", Document: &Document{Path: "_content/menus.yaml"}, TargetPath: "locales/fixture/structured/_content/menus.yaml"},
	}}
	finding := IntegratedSurfaceReceipt{Scope: IntegratedSurfaceScope{Reviewed: []SurfaceAsset{{ID: doc.Path, Package: "site-v2-shell"}}}, ExactFindings: []SurfaceFinding{{ID: doc.Path, Class: "structured-surface-language", Finding: "synthetic sentence punctuation finding"}}}
	if err := selectStructuredSurfaceRevision(root, &plan, &finding); err != nil || !reflect.DeepEqual(plan.Expected, []string{plan.Assets[0].TargetPath}) || !reflect.DeepEqual(plan.Assets[0].Document, &doc) {
		t.Fatal("finding selection lost exact representation", err)
	}
	if got := mustSharedFixtureBytes(t, root, "_content/site.tmpl"); digest(got) != before || !bytes.Equal(got, raw) {
		t.Fatal("raw template changed")
	}
	t.Log("RAW_SITE_TMPL_SHA256=" + before)
}

func TestSharedSentencePunctuationBoundaries(t *testing.T) {
	for name, raw := range map[string]string{
		"no-prose-prefix":        `<p><a href="/wiki/">Go Wiki</a>.</p>`,
		"previous-sentence":      `<p>Reference.<a href="/wiki/">Go Wiki</a>.</p>`,
		"outside-paragraph":      `<div>Read <a href="/wiki/">Go Wiki</a>.</div>`,
		"ellipsis":               `<p>Read <a href="/wiki/">Go Wiki</a>...</p>`,
		"nonterminal-text":       `<p>Read <a href="/wiki/">Go Wiki</a>. suffix</p>`,
		"nonterminal-element":    `<p>Read <a href="/wiki/">Go Wiki</a>.<span>suffix</span></p>`,
		"suffix-action":          `<p>Read <a href="/wiki/">Go Wiki</a>.{{.Suffix}}</p>`,
		"prefix-action":          `<p>Read {{.Prefix}}<a href="/wiki/">Go Wiki</a>.</p>`,
		"href-action":            `<p>Read <a href="{{.URL}}">Go Wiki</a>.</p>`,
		"label-action":           `<p>Read <a href="/wiki/">Go Wiki {{.Name}}</a>.</p>`,
		"code":                   `<p><code>Read <a href="/wiki/">Go Wiki</a>.</code></p>`,
		"code-link":              `<p>Read <a href="/wiki/"><code>json.Marshal</code></a>.</p>`,
		"hidden":                 `<p aria-hidden="true">Read <a href="/wiki/">Go Wiki</a>.</p>`,
		"empty-link":             `<p>Read <a href="/wiki/"></a>.</p>`,
		"icon-link":              `<p>Read <a href="/wiki/"><i class="material-icons">menu</i></a>.</p>`,
		"attribute-period":       `<p title=".">Read <a href="/file.txt?q=1.2">Go Wiki.</a></p>`,
		"domain-identifier":      `<p>Read <a href="https://example.org/v1.2">golang.org/x/tools/gopls/doc.</a></p>`,
		"comment-separator":      `<p>Read <a href="/wiki/">Go Wiki</a><!-- preserve -->.</p>`,
		"structural-punctuation": `<p>Read <a href="/wiki/">Go Wiki</a>;</p>`,
	} {
		t.Run(name, func(t *testing.T) {
			raw = "<header>Independent prose</header>" + raw
			old, err := ExtractShellLanguageSource("_content/site.tmpl", []byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			periods, err := shellSentencePeriods(*old, []byte(raw))
			if err != nil || len(periods) != 0 {
				t.Fatal("unrelated structure/identifier punctuation became editable", periods, err)
			}
		})
	}
}

func TestSharedSentencePunctuationExactSuccessor(t *testing.T) {
	raw := []byte(`<p>Read <a href="https://example.org/v1.2?q=a.b">docs.example.org</a>.</p><p><em>Browse <a href="/other/">Other resources</a>.</em></p><p><a href="/third/">Unrelated link</a>.</p>{{if .Flag}}<span>Other content</span>{{end}}`)
	old, err := ExtractShellLanguageSource("_content/site.tmpl", raw)
	if err != nil {
		t.Fatal(err)
	}
	periods, err := shellSentencePeriods(*old, raw)
	if err != nil || len(periods) != 2 {
		t.Fatal(periods, err)
	}
	first, releases, err := sharedRepresentationRuleDelta(*old, raw, []string{periods[1].Prefix}, SharedSentencePunctuationRule)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := sharedRepresentationRuleDelta(*first, raw, []string{periods[0].Prefix}, SharedSentencePunctuationRule)
	if err != nil {
		t.Fatal("second selective successor", err)
	}
	byID := map[string]ShellLanguageSlot{}
	values := map[string]string{}
	for _, s := range second.Slots {
		byID[s.Unit.ID], values[s.Unit.ID] = s, s.Unit.Source
	}
	for _, s := range first.Slots {
		if !reflect.DeepEqual(s, byID[s.Unit.ID]) {
			t.Fatal("later addition changed already accepted slots")
		}
	}
	for _, ids := range [][]string{nil, {periods[0].Prefix, periods[0].Prefix}, {strings.Repeat("a", 64)}, {periods[1].Prefix}} {
		if _, _, err := sharedRepresentationRuleDelta(*first, raw, ids, SharedSentencePunctuationRule); err == nil {
			t.Fatal("empty/duplicate/unknown/no-op selection accepted", ids)
		}
	}
	r := SharedRepresentationSuccessor{Request: SharedRepresentationRequest{Rule: SharedSentencePunctuationRule, Slots: []string{periods[1].Prefix}}, Old: *old, New: *first, Released: releases}
	if err := checkSharedSentencePunctuation(r, raw); err != nil {
		t.Fatal("sealed topology proof", err)
	}
	for _, name := range []string{"extra-slot", "removed-old-slot", "old-protection", "wrong-selected-prefix", "forged-release"} {
		t.Run(name, func(t *testing.T) {
			b, _ := Encode(r)
			var bad SharedRepresentationSuccessor
			if err := StrictJSON(b, &bad); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "extra-slot":
				bad.New = *second
			case "removed-old-slot":
				bad.New.Slots = bad.New.Slots[1:]
			case "old-protection":
				bad.New.Slots[0].Unit.Protected = []ProtectedToken{{Raw: "forged", Role: "atom"}}
			case "wrong-selected-prefix":
				bad.Request.Slots = []string{periods[0].Prefix}
			case "forged-release":
				bad.Released = []SharedRepresentationRelease{{SlotID: old.Slots[0].Unit.ID, Token: ProtectedToken{Raw: ".", Role: "atom"}}}
			}
			if checkSharedSentencePunctuation(bad, raw) == nil {
				t.Fatal("unbounded/forged successor accepted")
			}
		})
	}
	ref := Reference{SharedRepresentationRoot + "receipts/" + strings.Repeat("a", 64) + ".json", strings.Repeat("b", 64)}
	for _, source := range []*ShellLanguageSource{first, second} {
		doc := withSharedDocumentAuthority(shellDocument(*source), ref)
		v := map[string]string{}
		for _, s := range source.Slots {
			v[s.Unit.ID] = s.Unit.Source
			if s.Role == "sentence-punctuation" {
				v[s.Unit.ID] = "。"
			}
		}
		if _, err := reconstructFrozenStructuredRepresentation(doc, raw, v); err != nil {
			t.Fatal("frozen selected punctuation codec", err)
		}
		for _, drift := range [][2]string{{"example.org/v1.2", "elsewhere.org/v1.2"}, {"docs.example.org", "docs.elsewhere.org"}, {"{{if .Flag}}", "{{if .Other}}"}, {"Unrelated link</a>.", "Unrelated link</a>。"}} {
			if !bytes.Contains(raw, []byte(drift[0])) {
				t.Fatal("missing negative coordinate", drift)
			}
			if validateShellStructure(*source, raw, bytes.Replace(raw, []byte(drift[0]), []byte(drift[1]), 1)) == nil {
				t.Fatal("unrelated href/identifier/template/period mutation accepted", drift)
			}
		}
	}
}
