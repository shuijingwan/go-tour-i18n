package sitecontent

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarkdownInlineHTMLVisibleAttributes(t *testing.T) {
	for _, source := range []string{
		`Text before <img src="x.png" alt="Natural description"> text after.`,
		`<img src="x.png" alt="Natural description">`,
		`Text before <img src="x.png" alt="Natural description" /> text after.`,
		`Text before <span title="Natural title">visible text</span> text after.`,
		`<span title="Natural title">visible text</span>`,
		`Text before <span aria-label="Natural label">visible text</span> text after.`,
		`<span aria-label="Natural label">visible text</span>`,
		`Text before <img  id='fixed' src="x.png" alt='Natural description' title = "Natural title" aria-label="Natural label" data-x="fixed" /> text after.`,
	} {
		t.Run(source, func(t *testing.T) {
			d, raw := formalFixture(t, "x.md", source+"\n")
			replace := strings.NewReplacer("Text before", "前文", "text after.", "后文。", "Natural description", "自然说明", "Natural title", "自然标题", "Natural label", "自然标签", "visible text", "可见文本")
			values := []string{}
			for _, s := range d.Slots {
				values = append(values, replace.Replace(s.Source))
			}
			got := checkFormal(t, d, raw, formalTarget(d, values), true)
			if !bytes.Equal(got, []byte(replace.Replace(string(raw)))) {
				t.Fatalf("markup/attribute bytes changed: %q", got)
			}
			if err := validatePageStructure(d, raw, got); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMarkdownInlineHTMLAttributeInjectionRejected(t *testing.T) {
	d, raw := formalFixture(t, "x.md", `Text <img src="x.png" alt="Natural description" title="Natural title" /> after.`+"\n")
	good := strings.NewReplacer("Natural description", "自然说明", "Natural title", "自然标题").Replace(d.Units[0].Source)
	checkFormal(t, d, raw, good, true)
	for _, injected := range []string{`说明" onclick="bad`, `说明"><span>新增</span>`, `说明" extra="new`, "说明<unknown>新增</unknown>"} {
		checkFormal(t, d, raw, strings.Replace(good, "自然说明", injected, 1), false)
	}
	var firstOpen, firstClose, secondOpen, secondClose string
	for _, p := range d.Units[0].Protected {
		if strings.Contains(p.Raw, `src="x.png"`) {
			firstOpen = p.Token
			for _, bad := range []string{
				strings.Replace(p.Raw, "x.png", "y.png", 1),
				strings.Replace(p.Raw, "<img", "<input", 1),
				strings.TrimSuffix(p.Raw, `"`),
				strings.Replace(p.Raw, `src="x.png" `, "", 1),
				strings.Replace(p.Raw, "<img", `<img onclick="bad"`, 1),
			} {
				checkFormal(t, d, raw, strings.Replace(good, p.Token, bad, 1), false)
			}
		} else if p.Role == "open" {
			secondOpen = p.Token
		} else if p.Role == "close" && p.Raw == "" {
			firstClose = p.Token
		} else if p.Role == "close" {
			secondClose = p.Token
		}
	}
	if firstOpen == "" || firstClose == "" || secondOpen == "" || secondClose == "" {
		t.Fatal("expected two independently editable bound attributes", d.Units[0])
	}
	for _, bad := range []string{
		strings.Replace(good, firstOpen, "", 1),
		good + firstOpen,
		strings.Replace(good, firstClose+secondOpen, firstClose+"新增"+secondOpen, 1),
		strings.Replace(good, firstOpen+"自然说明"+firstClose+secondOpen+"自然标题"+secondClose,
			secondOpen+"自然标题"+secondClose+firstOpen+"自然说明"+firstClose, 1),
	} {
		checkFormal(t, d, raw, bad, false)
	}
	// A paired start tag cannot move after its element's visible body.
	d, raw = formalFixture(t, "x.md", `Text <span title="Natural title">visible text</span> after.`+"\n")
	s := d.Slots[0]
	bad := strings.Replace(s.Source, s.Protected[0].Token+s.Protected[1].Token+"Natural title"+s.Protected[2].Token+"visible text",
		s.Protected[0].Token+"visible text"+s.Protected[1].Token+"Natural title"+s.Protected[2].Token, 1)
	checkFormal(t, d, raw, formalTarget(d, []string{bad}), false)
}

func TestMarkdownInlineHTMLAttributeProtectionBoundaries(t *testing.T) {
	for _, bad := range []string{
		`Text <unknown alt="Natural description"> after.`,
		`Text <img src="x.png" alt=description> after.`,
		`Text <img src="x.png" alt="Natural description> after.`,
		`Text <span title="Natural title">visible text`,
		`Text <img`,
	} {
		if _, _, err := protectBlock([]byte(bad), "markdown/Paragraph"); err == nil {
			t.Fatalf("unsupported/malformed HTML accepted: %s", bad)
		}
	}
	for _, tag := range []string{"code", "pre", "script", "style"} {
		source := `<` + tag + ` title="Opaque description">opaque content</` + tag + `>`
		visible, locked, err := protectBlock([]byte("Text "+source+" after."), "markdown/Paragraph")
		if err != nil || len(locked) != 1 || locked[0].Raw != source || strings.Contains(visible, "Opaque description") {
			t.Fatalf("opaque %s changed: %s %+v %v", tag, visible, locked, err)
		}
	}
	// The current bounded HTML lexer rejects URL autolinks as unsupported
	// tags. This attribute change must preserve that fail-closed behavior.
	if _, _, err := protectBlock([]byte("Text <https://example.com> after."), "markdown/Paragraph"); err == nil {
		t.Fatal("existing unsupported autolink behavior changed")
	}
	d, raw := formalFixture(t, "x.md", "Text with `go mod tidy` and [the guide](/fixed).\n")
	checkFormal(t, d, raw, d.Units[0].Source, true)
	for _, machine := range []string{"`go mod tidy`", "](/fixed)"} {
		found := false
		for _, p := range d.Units[0].Protected {
			if p.Raw == machine {
				found = true
				checkFormal(t, d, raw, strings.Replace(d.Units[0].Source, p.Token, "changed", 1), false)
			}
		}
		if !found {
			t.Fatal("existing inline binding exposed", machine)
		}
	}
}
