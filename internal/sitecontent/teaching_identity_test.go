package sitecontent

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompleteCodeTeachingASCIIIdentityControls(t *testing.T) {
	code := "package main\nvar message, name, formats string\nfunc main() {\n // Associate message with name using Go, JSON, API and RFC3339.\n // Read the OpenAPI description.\n}\n"
	for _, enclosure := range []struct{ name, path, before, after string }{
		{"markdown-fence", "x.md", "```go\n", "```\n"},
		{"html-pre", "x.html", "<pre>\n", "</pre>\n"},
		{"indented-code", "x.md", "", ""},
	} {
		t.Run(enclosure.name, func(t *testing.T) {
			body := code
			if enclosure.before == "" {
				body = "    " + strings.ReplaceAll(strings.TrimSuffix(code, "\n"), "\n", "\n    ") + "\n"
			}
			d, raw := formalFixture(t, enclosure.path, enclosure.before+body+enclosure.after)
			index := -1
			values := []string{}
			for i, s := range d.Slots {
				values = append(values, s.Source)
				if strings.Contains(s.Source, "Associate") {
					index = i
				}
			}
			if index < 0 {
				t.Fatal("teaching slot missing")
			}
			s := d.Slots[index]
			tokens := map[string]string{}
			for _, p := range s.Protected {
				tokens[p.Raw] = p.Token
			}
			for _, id := range []string{"message", "name", "JSON", "API"} {
				if tokens[id] == "" {
					t.Fatal("source binding missing", id, s)
				}
			}
			check := func(t *testing.T, value string, pass bool) []byte {
				t.Helper()
				v := append([]string{}, values...)
				v[index] = value
				return checkFormal(t, d, raw, formalTarget(d, v), pass)
			}
			positive := "将" + tokens["name"] + "与" + tokens["message"] + "关联，使用 Go、" + tokens["JSON"] + "、" + tokens["API"] + " 和 RFC3339。"
			for name, value := range map[string]string{
				"identifier-reorder-and-tech-terms": positive,
				"source-exact-bindings":             s.Source,
				"ordinary-chinese-prose":            "使用" + tokens["JSON"] + "和" + tokens["API"] + "将" + tokens["name"] + "与" + tokens["message"] + "关联。",
				"new-ordinary-ascii-prose":          "Use these values with Go: " + tokens["name"] + " " + tokens["message"] + " " + tokens["JSON"] + " " + tokens["API"] + ".",
			} {
				t.Run(name, func(t *testing.T) {
					got := check(t, value, true)
					if name == "identifier-reorder-and-tech-terms" && !bytes.Contains(got, []byte("将name与message关联")) {
						t.Fatal("identifier order lost", string(got))
					}
				})
			}
			for name, value := range map[string]string{
				"newName":              positive + " newName",
				"new_identifier":       positive + " new_identifier",
				"unknown-call":         positive + " unknown()",
				"unknown-spaced-call":  positive + " unknown ()",
				"other-slot-identity":  positive + " OpenAPI",
				"extended-token":       strings.Replace(positive, tokens["name"], tokens["name"]+"X", 1),
				"identifier-duplicate": positive + " name",
				"identifier-injection": positive + " formats",
				"comment-delimiter":    positive + " // extra",
				"block-comment":        positive + " /* extra */",
				"newline":              positive + "\n",
				"empty-body":           "",
			} {
				t.Run(name, func(t *testing.T) { check(t, value, false) })
			}
			// A code-like spelling explicitly present in this source slot may be
			// retained, but must not authorize a different slot or extra copies.
			for i, slot := range d.Slots {
				if !strings.Contains(slot.Source, "OpenAPI") {
					continue
				}
				v := append([]string{}, values...)
				v[i] = "阅读 OpenAPI 描述。"
				checkFormal(t, d, raw, formalTarget(d, v), true)
				v[i] += " OpenAPI"
				checkFormal(t, d, raw, formalTarget(d, v), false)
			}
		})
	}
}
