package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func shellTestRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestShellSourceExtractionTemplateIsolation(t *testing.T) {
	raw := []byte(`{{define "gopher info"}}{{$alt := "Go gopher with a wrench"}}{{$src := "/images/English-machine.svg"}}{{if eq .color "English machine selector"}}<img src="{{$src}}" alt="{{$alt}}">{{end}}{{end}}
<header><a href="{{if .children}}#{{else}}{{.url}}{{end}}" aria-label="Open {{.name}} menu">Visible navigation</a><a aria-label={{.name}}>Visible link</a><img alt=""><i class="material-icons">arrow_drop_down</i></header>
<script>let text = "English code";</script><style>.English { color: red; }</style>{{/* This English comment isn't visible }} */}}
<footer>Visible footer</footer>`)
	s, err := ExtractShellLanguageSource("_content/site.tmpl", raw)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ProjectShellSuccessorContributor(*s)
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, x := range c.Contexts {
		text.WriteString(x.Text)
		text.WriteByte('\n')
	}
	for _, want := range []string{"Visible navigation", "Visible footer", "Go gopher with a wrench", "Open menu"} {
		if !strings.Contains(text.String(), want) {
			t.Fatalf("visible text missing: %s\n%s", want, text.String())
		}
	}
	for _, bad := range []string{"English machine", "English code", "English comment", "arrow_drop_down", "/images/"} {
		if strings.Contains(text.String(), bad) {
			t.Fatalf("machine text leaked: %s", bad)
		}
	}
	for _, slot := range s.Slots {
		if slot.Unit.Start < 0 || slot.Unit.End > len(raw) {
			t.Fatal("slot coordinate outside source")
		}
	}
	if _, err := ExtractShellLanguageSource("_content/site.tmpl", []byte(`<p>{{unknownFunction "visible?"}}</p>`)); err == nil {
		t.Fatal("unknown template function accepted")
	}
	if _, err := ExtractShellLanguageSource("_content/site.tmpl", []byte(`<a aria-label=Unquoted>Visible link</a>`)); err == nil {
		t.Fatal("unquoted static accessibility label accepted")
	}
}

func TestShellSourceExtractionYAMLSchemas(t *testing.T) {
	fixtures := map[string]string{
		"_content/menus.yaml":        "main:\n  - name: Learn\n    url: /learn/\n    socialIconsList: [{name: GitHub, img: github.svg, url: 'https://github.com/golang'}]\nfooter:\n  - name: Get Started\n    url: /learn/\n",
		"_content/resources.yaml":    "- title: Cloud Services\n  description: \"Build services with Go.\"\n  link: /solutions/cloud/\n  icon: /images/cloud.svg\n  iconDark: /images/cloud-dark.svg\n  iconName: Cloud\n  packages:\n    - title: cloud.google.com/go\n      url: https://cloud.google.com/go/home\n",
		"_content/testimonials.yaml": "- name: Jane Example\n  title: Lead Engineer\n  company: Example Corp\n  quote: |-\n    Use <strong>Go</strong> to build services.\n    It works well.\n- name: Other Person\n  title:\n  company: Other Corp\n  quote: 'Go is useful.'\n",
	}
	for path, raw := range fixtures {
		t.Run(path, func(t *testing.T) {
			s, err := ExtractShellLanguageSource(path, []byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			for _, slot := range s.Slots {
				rawSpan := raw[slot.Unit.Start:slot.Unit.End]
				if strings.Contains(slot.Unit.Source, "https://") || strings.Contains(slot.Unit.Source, "cloud.google.com") {
					t.Fatal("machine identity entered language")
				}
				if strings.Contains(slot.Unit.Source, "GitHub") && rawSpan != "GitHub" {
					t.Fatalf("flow YAML span: %q", rawSpan)
				}
			}
		})
	}
	for _, raw := range []string{"main:\n - name: First\n   name: Duplicate\nfooter: []\n", "main: &m []\nfooter: *m\n", "main: []\nfooter: []\nunapproved: English\n", "main:\n - name: true\nfooter: []\n", "main: []\nfooter: []\n---\nmain: []\n"} {
		if _, err := ExtractShellLanguageSource("_content/menus.yaml", []byte(raw)); err == nil {
			t.Fatalf("unsafe YAML accepted: %s", raw)
		}
	}
}

func TestShellSourceExtractionHomepagePhysicalPage(t *testing.T) {
	raw := []byte("---\ntitle: The Go Programming Language\nsummary: Build services with Go.\ntemplate: true\n---\n<h1>Build with Go</h1><textarea>// You can edit this code!\n// Click here and start typing.\npackage main\nfunc main() {}\n</textarea><script>English code</script>")
	s, err := ExtractShellLanguageSource("_content/index.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.Page == nil || s.Page.UnitScope != PageContract || len(s.Page.Units) != 1 || s.Page.Path != "_content/index.md" || s.Page.Route != "/" {
		t.Fatal("physical homepage split into formal slot TUs")
	}
	if len(s.Page.Slots) != len(s.Slots) {
		t.Fatal("Page lost internal slots")
	}
	restored, err := restoreBlock(s.Page.Units[0], s.Page.Units[0].Source)
	if err != nil || !bytes.Equal(raw, []byte(restored)) {
		t.Fatal("formal homepage does not restore source byte-exact", err)
	}
	foundComment := false
	for _, slot := range s.Slots {
		if slot.Role == "teaching-comment" {
			foundComment = true
		}
		if strings.Contains(slot.Unit.Source, "English code") {
			t.Fatal("script treated as language")
		}
	}
	if !foundComment {
		t.Fatal("safe user-visible textarea comment missing")
	}
}

func TestShellSuccessorMappingExactConditions(t *testing.T) {
	u := Unit{ID: "old", Source: "Learn Go", Protected: []ProtectedToken{}}
	old := ShellLanguageSlot{Unit: u, Role: "visible-text", Context: "homepage/page-heading", Encoding: "html-text"}
	next := old
	next.Unit.ID = "next"
	next.Unit.Source = "Learn  Go"
	if !compatibleShellSlot(old, next, "学习 Go") {
		t.Fatal("normalization-compatible role/context/protection denied")
	}
	for _, change := range []string{"source", "role", "context", "encoding", "protection", "target", "empty-target", "new-markup"} {
		t.Run(change, func(t *testing.T) {
			n := next
			target := "学习 Go"
			switch change {
			case "source":
				n.Unit.Source = "Learn Docs"
			case "role":
				n.Role = "attribute/alt"
			case "context":
				n.Context = "footer/link"
			case "encoding":
				n.Encoding = "yaml-plain"
			case "protection":
				n.Unit.Protected = []ProtectedToken{{Token: "⟦P0⟧", Raw: "/learn/", Role: "atom"}}
			case "target":
				target = "学习 ⟦P99⟧"
			case "empty-target":
				target = ""
			case "new-markup":
				target = "<strong>学习</strong> Go"
			}
			if compatibleShellSlot(old, n, target) {
				t.Fatal("inexact or incompatible carry accepted")
			}
		})
	}
	legacy := legacyShellSlot(u)
	newSlot := legacy
	newSlot.Unit.ID = "next"
	oldDoc := Document{Units: []Unit{u}}
	sources := []ShellLanguageSource{{Path: "_content/index.md", Slots: []ShellLanguageSlot{newSlot}}}
	m := mapOldShellHome(oldDoc, sources, map[string]string{"old": "学习 Go"}, []Reference{{Path: "accepted.json", SHA256: strings.Repeat("a", 64)}})
	if len(m) != 1 || m[0].State != "carry" || m[0].Destination == nil {
		t.Fatal("exact provenance-bound mapping missing")
	}
	m = mapOldShellHome(oldDoc, sources, map[string]string{"old": "学习 Go"}, nil)
	if m[0].State != "affected" {
		t.Fatal("carry without accepted provenance")
	}
	sources = append(sources, sources[0])
	m = mapOldShellHome(oldDoc, sources, map[string]string{"old": "学习 Go"}, nil)
	if m[0].State != "affected" {
		t.Fatal("ambiguous destination carried")
	}
}

func TestShellSuccessorTerminologyProjection(t *testing.T) {
	raw := []byte("<header>Learn Go</header><script>machine one</script>")
	s, err := ExtractShellLanguageSource("_content/site.tmpl", raw)
	if err != nil {
		t.Fatal(err)
	}
	one, err := ProjectShellSuccessorContributor(*s)
	if err != nil {
		t.Fatal(err)
	}
	s, err = ExtractShellLanguageSource("_content/site.tmpl", bytes.Replace(raw, []byte("machine one"), []byte("machine TWO!"), 1))
	if err != nil {
		t.Fatal(err)
	}
	two, err := ProjectShellSuccessorContributor(*s)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(one, two) {
		t.Fatal("machine-only script change altered terminology projection")
	}
	s, err = ExtractShellLanguageSource("_content/site.tmpl", bytes.Replace(raw, []byte("Learn Go"), []byte("Learn Modules"), 1))
	if err != nil {
		t.Fatal(err)
	}
	changed, _ := ProjectShellSuccessorContributor(*s)
	if changed.SourceSHA == one.SourceSHA {
		t.Fatal("visible terminology change ignored")
	}
}

func TestShellSuccessorPendingArtifactStrictCurrentness(t *testing.T) {
	// Synthetic wire fixture; never a language/completion/Surface receipt.
	a := &ShellSuccessorAuthority{Schema: ShellSuccessorSchema, Package: "site-v2-shell", State: "pending-source-authority", Sources: []ShellLanguageSource{{Path: "_content/index.md"}}, ParserReferences: []Reference{}}
	a.Identity = identity(*a)
	ref, err := shellAuthorityReference(a)
	if err != nil {
		t.Fatal(err)
	}
	scope := &ShellAffectedScope{Schema: ShellAffectedSchema, Package: "site-v2-shell", State: "pending-language-work", Locale: "zh-CN", SourceAuthority: ref, SourceIdentity: a.Identity, Units: []ShellScopeUnit{{Path: "_content/index.md", State: "generation-required"}}}
	scope.Identity = identity(*scope)
	sourceBytes, _ := Encode(a)
	scopeBytes, _ := Encode(scope)
	if err := checkShellSuccessorWire(scopeBytes, sourceBytes, scope, a); err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"scope-unknown", "scope-duplicate", "scope-tail", "scope-exact-set", "source-state", "source-membership", "source-whitespace"} {
		t.Run(change, func(t *testing.T) {
			b, bound := bytes.Clone(scopeBytes), bytes.Clone(sourceBytes)
			switch change {
			case "scope-unknown":
				b = bytes.Replace(b, []byte(`"schema":`), []byte(`"unknown":true,"schema":`), 1)
			case "scope-duplicate":
				b = bytes.Replace(b, []byte(`"schema":`), []byte(`"schema":"duplicate","schema":`), 1)
			case "scope-tail":
				b = append(b, []byte("{}")...)
			case "scope-exact-set":
				modified := *scope
				modified.Units = []ShellScopeUnit{}
				modified.Identity = ""
				modified.Identity = identity(modified)
				b, _ = Encode(modified)
			case "source-state":
				bound = bytes.Replace(bound, []byte("pending-source-authority"), []byte("complete"), 1)
			case "source-membership":
				bound = bytes.Replace(bound, []byte("_content/index.md"), []byte("_content/extra.md"), 1)
			case "source-whitespace":
				bound = append(bound, '\n')
			}
			if err := checkShellSuccessorWire(b, bound, scope, a); err == nil {
				t.Fatal("mutated pending artifact accepted")
			}
		})
	}
	root := t.TempDir()
	if err := saveImmutable(root, ref.Path, a); err != nil {
		t.Fatal(err)
	}
	if err := saveImmutable(root, ref.Path, a); err != nil {
		t.Fatal("byte-identical reuse failed", err)
	}
	changed := *a
	changed.State = "complete"
	if err := saveImmutable(root, ref.Path, &changed); err == nil {
		t.Fatal("pending authority overwritten")
	}
	got, _ := readRegular(root, ref.Path)
	if !bytes.Equal(got, sourceBytes) {
		t.Fatal("failed mutation changed source artifact")
	}
}

func TestShellSuccessorAffectedScopeCurrentRepository(t *testing.T) {
	root := shellTestRoot(t)
	before, err := os.ReadFile(filepath.Join(root, LocalePath("zh-CN")))
	if err != nil {
		t.Fatal(err)
	}
	if digest(before) != "9bcdf339dfe64d182003bb77e49befb3ac4ca72e38a1210ffd7f3adb0de77879" {
		t.Fatal("formal completion baseline differs")
	}
	a, err := BuildShellSuccessorAuthority(root)
	if err != nil {
		t.Fatal(err)
	}
	if a.State != "pending-source-authority" || len(a.Sources) != 5 || a.Sources[0].Page == nil || len(a.Sources[0].Page.Units) != 1 {
		t.Fatal("pending successor inventory")
	}
	s, err := BuildShellSuccessorAffectedScope(root, "zh-CN", a)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, u := range s.Units {
		counts[u.Role]++
		if (u.Role == "translation-body" || u.Role == "learn-docs-page") && (u.State != "carry" || len(u.GenerationSlots) != 0 || u.TargetSHA == "" || len(u.Evidence) == 0) {
			t.Fatal("unchanged accepted body/Page requires Generation")
		}
	}
	if counts["homepage-page"] != 1 || counts["learn-docs-page"] != 49 || counts["translation-body"] != 1 || len(s.Units) != 55 {
		t.Fatal("affected scope exact-set changed", counts)
	}
	if len(s.OldHomeMappings) != 5 {
		t.Fatal("historical homepage slots changed")
	}
	for _, m := range s.OldHomeMappings {
		if m.State != "affected" || m.Destination != nil || m.TargetSHA == "" {
			t.Fatal("unproven old-home carry")
		}
	}
	if !s.Terminology.Affected || s.Terminology.NewSources != 5 || s.Terminology.NewContexts != 235 || len(s.Terminology.Contributors) != 5 {
		t.Fatal("terminology additions not projected")
	}
	after, _ := os.ReadFile(filepath.Join(root, LocalePath("zh-CN")))
	if !bytes.Equal(before, after) {
		t.Fatal("foundation mutated completion")
	}
	a.Sources[0].SourceSHA = strings.Repeat("0", 64)
	if _, err := BuildShellSuccessorAffectedScope(root, "zh-CN", a); err == nil {
		t.Fatal("source authority mismatch accepted")
	}
	t.Logf("source_identity=%s scope_identity=%s new_contexts=%d", s.SourceIdentity, s.Identity, s.Terminology.NewContexts)
	for _, u := range s.Units {
		if u.State == "generation-required" {
			t.Logf("%s: slots=%d formal_page=%t", u.Path, len(u.GenerationSlots), u.UnitID != "")
		}
	}
}
