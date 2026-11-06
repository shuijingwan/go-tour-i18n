package sitecontent

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestShellFrozenTargetValidation(t *testing.T) {
	raw := []byte("---\ntitle: The Go Programming Language\nsummary: Build services with Go.\ntemplate: true\n---\n{{if .company}}<a class=\"study\" href=\"https://go.dev/case\" aria-label=\"View CaseStudy of {{.company}}, (opens in new window)\">Read more</a>{{end}}<textarea>// You can edit this code!\n// Click here and start typing.\npackage main\nfunc main() {}\n</textarea>\n")
	s, err := ExtractShellLanguageSource("_content/index.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	a := ShellPageAdapter{*s, raw}
	page := s.Page.Units[0]
	caseToken, companyToken := "", ""
	for _, p := range page.Protected {
		if p.Raw == "CaseStudy" {
			caseToken = p.Token
		}
		if p.Raw == "{{.company}}" {
			companyToken = p.Token
		}
	}
	if caseToken == "" || companyToken == "" {
		t.Fatal("fixture lost frozen identities")
	}
	for _, fixture := range []struct{ name, comment, label string }{
		{"zh-CN", "点击此处并开始输入。", "查看" + companyToken + "的" + caseToken + "（在新窗口中打开）"},
		{"ja-JP", "ここをクリックして入力を始めてください。", companyToken + "の" + caseToken + "を見る（新しいウィンドウで開く）"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			output := strings.Replace(page.Source, "Click here and start typing.", fixture.comment, 1)
			output = strings.Replace(output, "View "+caseToken+" of "+companyToken+", (opens in new window)", fixture.label, 1)
			candidate, err := a.Restore(output + "\n")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(candidate, []byte(fixture.comment)) || !bytes.Contains(candidate, []byte("package main\nfunc main() {}")) || !bytes.Contains(candidate, []byte(strings.ReplaceAll(strings.ReplaceAll(fixture.label, caseToken, "CaseStudy"), companyToken, "{{.company}}"))) {
				t.Fatal("localized fixture/code mismatch")
			}
			if err := a.Validate(candidate, &i18n.Glossary{Mandatory: map[string]string{}}); err != nil {
				t.Fatal(err)
			}
			_, visible, err := a.GlossaryVisiblePair(candidate)
			if err != nil || !strings.Contains(visible, fixture.comment) || strings.Contains(visible, "CaseStudy") || strings.Contains(visible, "{{.company}}") {
				t.Fatal("frozen visibility lost comment or leaked identity", err)
			}
			for _, drift := range []struct{ name, from, to string }{
				{"CaseStudy", "CaseStudy", "CaseStudies"},
				{"ASCII-extension", "CaseStudy", "CaseStudyX"},
				{"company", "{{.company}}", "{{.other}}"},
				{"template", "{{if .company}}", "{{if .other}}"},
				{"URL", "https://go.dev/case", "https://evil.example/case"},
				{"HTML", "class=\"study\"", "class=\"other\""},
				{"Go-code", "func main() {}", "func main() { panic(1) }"},
				{"markup", fixture.comment, "<script>x</script>" + fixture.comment},
				{"comment-marker", fixture.comment, "/* " + fixture.comment + " */"},
				{"directive", fixture.comment, "go:generate echo bad"},
				{"duplicate-identity", "CaseStudy", "CaseStudy CaseStudy"},
			} {
				t.Run(drift.name, func(t *testing.T) {
					bad := bytes.Replace(candidate, []byte(drift.from), []byte(drift.to), 1)
					if bytes.Equal(bad, candidate) || a.Validate(bad, &i18n.Glossary{Mandatory: map[string]string{}}) == nil {
						t.Fatal("drift accepted")
					}
				})
			}
			for _, change := range []string{strings.Replace(output, caseToken, "CaseStudies", 1), strings.Replace(output, companyToken, "{{.other}}", 1), strings.Replace(output, fixture.comment, "<b>新增</b>", 1)} {
				if _, err := a.Restore(change); err == nil {
					t.Fatal("formal token/markup drift accepted")
				}
			}
		})
	}
}

func TestShellSourceTeachingDiscoveryRemainsConservative(t *testing.T) {
	code := "// Click here and start typing.\n// 点击此处并开始输入。\n// go:generate echo hello\n// x = 1\npackage main\nfunc main() {}\n"
	bodies := i18n.SafeTeachingCommentBodies(code)
	if len(bodies) != 1 || code[bodies[0].Start:bodies[0].End] != "Click here and start typing." {
		t.Fatalf("source discovery changed: %+v", bodies)
	}
}

func TestShellValidatorFixPreservesFrozenSourceAdoption(t *testing.T) {
	root := shellTestRoot(t)
	var ref Reference
	b, err := readRegular(root, ShellAdoptionCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(b, &ref); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckShellAdoption(root, ref); err != nil {
		t.Fatal(err)
	}
}

func TestFrozenValidatorDependenciesPreserveOnlyDeclaredValidation(t *testing.T) {
	current := []Dependency{{Kind: "contract", Path: "validator.go", SHA256: "new-validator"}, {Kind: "contract", Path: "parser.go", SHA256: "new-parser"}, {Kind: "contract", Path: "runtime.go", SHA256: "new-runtime"}}
	frozen := []Dependency{{Kind: "contract", Path: "validator.go", SHA256: "old-validator"}, {Kind: "contract", Path: "parser.go", SHA256: "old-parser"}, {Kind: "contract", Path: "runtime.go", SHA256: "old-runtime"}}
	got := preserveFrozenValidatorDependencies(current, frozen, []string{"validator.go"})
	if got[0] != frozen[0] || got[1] != current[1] || got[2] != current[2] || reflect.DeepEqual(got, current) || current[0].SHA256 != "new-validator" {
		t.Fatalf("dependencies=%+v current=%+v", got, current)
	}
}

func TestShellFrozenIdentityBindingIsNonrecursive(t *testing.T) {
	u := Unit{Protected: []ProtectedToken{{Token: "⟦P0⟧", Raw: "CaseStudy", Role: "atom"}, {Token: "⟦P1⟧", Raw: "P", Role: "atom"}}}
	bound, err := bindFrozenShellTokens(u, "查看CaseStudy和P。")
	if err != nil || bound != "查看⟦P0⟧和⟦P1⟧。" {
		t.Fatalf("bound=%q error=%v", bound, err)
	}
	for _, bad := range []string{"查看CaseStudy和PP。", "查看CaseStudy和P与P。", "查看CaseStudy。"} {
		if _, err := bindFrozenShellTokens(u, bad); err == nil {
			t.Fatalf("identity drift accepted: %q", bad)
		}
	}
}

func TestShellFrozenTargetFraming(t *testing.T) {
	for _, fixture := range []struct{ path, raw string }{
		{"_content/site.tmpl", `{{define "gopher info"}}{{$alt := "Gopher with a wrench"}}<img alt="A friendly gopher">{{end}}<p>Read more</p>`},
		{"_content/menus.yaml", "main:\n  - name: 'Learn Go'\n    url: /learn/\nfooter: []\n"},
		{"_content/testimonials.yaml", "- name: Jane\n  company: Corp\n  title: \"Lead Engineer\"\n  quote: |-\n    Use <strong>Go</strong> to build services.\n    It works well.\n"},
	} {
		t.Run(fixture.path, func(t *testing.T) {
			raw := []byte(fixture.raw)
			s, err := ExtractShellLanguageSource(fixture.path, raw)
			if err != nil {
				t.Fatal(err)
			}
			values := map[string]string{}
			for _, slot := range s.Slots {
				values[slot.Unit.ID] = slot.Unit.Source
				if !strings.Contains(slot.Unit.Source, "⟦") {
					values[slot.Unit.ID] = "自然な文章"
				}
			}
			b, err := reconstructShellSlots(*s, raw, values)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateShellStructure(*s, raw, b); err != nil {
				t.Fatal(err)
			}
			for _, slot := range s.Slots {
				if len(slot.Unit.Protected) != 0 {
					continue
				}
				for _, bad := range []string{"<b>new markup</b>", "broken\nframing", string([]byte{0xff})} {
					copy := map[string]string{}
					for k, v := range values {
						copy[k] = v
					}
					copy[slot.Unit.ID] = bad
					if _, err := reconstructShellSlots(*s, raw, copy); err == nil {
						t.Fatalf("unsafe target accepted: %q", bad)
					}
				}
			}
		})
	}
}
