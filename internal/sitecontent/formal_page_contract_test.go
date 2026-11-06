package sitecontent

import (
	"bytes"
	"strings"
	"testing"
)

func formalFixture(t *testing.T, name, source string) (Document, []byte) {
	t.Helper()
	raw := []byte(source)
	d, err := ParseWorkflowDocument(Source{Path: name, Kind: "page", SHA256: digest(raw)}, raw, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Units) != 1 || len(d.Slots) == 0 {
		t.Fatal("physical Page must be exactly one TU", d)
	}
	return *d, raw
}

// Construct test inputs using the actual slot-to-Page token mapping.
func formalTarget(d Document, values []string) string {
	var out strings.Builder
	i := 0
	for si, s := range d.Slots {
		out.WriteString(d.Units[0].Protected[i].Token)
		i++
		pairs := []string{}
		for _, p := range s.Protected {
			pairs = append(pairs, p.Token, d.Units[0].Protected[i].Token)
			i++
		}
		v := values[si]
		if len(pairs) != 0 {
			v = strings.NewReplacer(pairs...).Replace(v)
		}
		out.WriteString(v)
	}
	out.WriteString(d.Units[0].Protected[i].Token)
	return out.String()
}

func checkFormal(t *testing.T, d Document, raw []byte, target string, wantPass bool) []byte {
	t.Helper()
	b, err := ReconstructWorkflow(d, raw, map[string]string{d.Units[0].ID: target})
	if (err == nil) != wantPass {
		t.Fatalf("pass=%t: %v, target=%q", wantPass, err, target)
	}
	return b
}

func TestFormalPageSlotCoordinates(t *testing.T) {
	d, raw := formalFixture(t, "x.md", "```go\nfunc main() {\n // Create a Context with a timeout.\n Context()\n}\n```\n")
	if len(d.Slots) != 1 || !strings.Contains(d.Slots[0].Source, "with a timeout") || len(d.Slots[0].Protected) != 1 {
		t.Fatal("expected one source-bound teaching body", d.Slots)
	}
	for _, trailing := range []string{"。", ""} {
		target := formalTarget(d, []string{"创建带超时的" + d.Slots[0].Protected[0].Token + trailing})
		got := checkFormal(t, d, raw, target, true)
		if !bytes.Contains(got, []byte("Context")) || !bytes.Contains(got, []byte("Context()")) || !bytes.Contains(got, []byte("// ")) {
			t.Fatal("surrounding machine bytes changed", string(got))
		}
	}
}

func TestFormalPageTeachingComments(t *testing.T) {
	d, raw := formalFixture(t, "x.md", "```go\nfunc main() {\n // Create a helper function for preparing failure results.\n fail()\n}\n```\n")
	target := formalTarget(d, []string{"创建一个辅助函数，用于准备失败结果。"})
	got := checkFormal(t, d, raw, target, true)
	want := strings.Replace(string(raw), d.Slots[0].Source, "创建一个辅助函数，用于准备失败结果。", 1)
	if string(got) != want {
		t.Fatalf("code/comment marker changed: %q", got)
	}
	for _, bad := range []string{"译文\n bad()", "译文\n", "译文\n}\nfunc other() {"} {
		checkFormal(t, d, raw, formalTarget(d, []string{bad}), false)
	}
	// Code is in a boundary token, and cannot be rewritten or removed.
	checkFormal(t, d, raw, strings.Replace(target, d.Units[0].Protected[0].Token, "```go\nchanged()\n// ", 1), false)
}

func TestFormalPageBindingMovement(t *testing.T) {
	d, raw := formalFixture(t, "x.md", "Use `go mod tidy` with [the guide](/fixed)\n  for more details.\n")
	if len(d.Slots) != 1 || len(d.Slots[0].Protected) != 4 {
		t.Fatal(d.Slots)
	}
	good := "参阅 ⟦P1⟧指南⟦P2⟧，⟦P3⟧然后使用 ⟦P0⟧。"
	got := checkFormal(t, d, raw, formalTarget(d, []string{good}), true)
	if !bytes.Contains(got, []byte("[指南](/fixed)")) || !bytes.Contains(got, []byte("`go mod tidy`")) {
		t.Fatal(string(got))
	}
	for _, bad := range []string{
		good + "⟦P0⟧", good + "⟦P99⟧",
		strings.Replace(good, "⟦P0⟧", "`go mod download`", 1),
		strings.Replace(good, "⟦P2⟧", "](/changed)", 1),
		"⟦P2⟧指南⟦P1⟧ ⟦P3⟧⟦P0⟧",
		good + "<a href=\"/new\">新增</a>", good + "<code>new</code>", good + "[new](/new)", good + "{{.title}}",
		good + "\n# New", good + "\r", good + "\ufeff", good + string([]byte{0xff}),
	} {
		checkFormal(t, d, raw, formalTarget(d, []string{bad}), false)
	}
	// Moving an authorized wrap can still create a new block at its new location.
	checkFormal(t, d, raw, formalTarget(d, []string{"使用 ⟦P0⟧ ⟦P1⟧指南⟦P2⟧⟦P3⟧# 新标题"}), false)
	checkFormal(t, d, raw, formalTarget(d, []string{strings.Replace(good, "⟦P3⟧", "  ⟦P3⟧", 1)}), false)
}

func TestFormalPageHTMLStructure(t *testing.T) {
	d, raw := formalFixture(t, "x.html", "<p>Read <a href=\"/fixed\">the guide</a> with <code>CLI</code>.</p>\n")
	good := "使用 ⟦P2⟧，参阅 ⟦P0⟧指南⟦P1⟧。"
	checkFormal(t, d, raw, formalTarget(d, []string{good}), true)
	for _, bad := range []string{good + "<b>新增</b>", strings.Replace(good, "⟦P2⟧", "<code>API</code>", 1), strings.Replace(good, "⟦P1⟧", "", 1), good + "</p><p>新增"} {
		checkFormal(t, d, raw, formalTarget(d, []string{bad}), false)
	}
	d, raw = formalFixture(t, "x.html", "<img src=\"/fixed\" alt=\"A natural description\">\n")
	checkFormal(t, d, raw, formalTarget(d, []string{"自然说明"}), true)
	checkFormal(t, d, raw, formalTarget(d, []string{"说明\" onclick=\"bad"}), false)
}

func TestFormalPageCrossSlotAndSuffix(t *testing.T) {
	d, raw := formalFixture(t, "x.md", "First paragraph.\n\nUse `CLI` here.\n")
	target := formalTarget(d, []string{"第一段。", "使用 ⟦P0⟧。"})
	checkFormal(t, d, raw, target, true)
	var atom string
	for _, p := range d.Units[0].Protected {
		if p.Raw == "`CLI`" {
			atom = p.Token
		}
	}
	leaked := strings.Replace(target, atom, "", 1)
	leaked = strings.Replace(leaked, "第一段。", "第一段。"+atom, 1)
	checkFormal(t, d, raw, leaked, false)
	for _, source := range []string{"## Syntax {#tool-syntax}\n", "## Add a fuzz test {#fuzz_test}\n"} {
		d, raw := formalFixture(t, "x.md", source)
		checkFormal(t, d, raw, formalTarget(d, []string{"语法 ⟦P0⟧"}), true)
		checkFormal(t, d, raw, formalTarget(d, []string{"运行它。⟦P0⟧语法"}), false)
	}
}

func TestMarkdownEmphasisLanguageProtection(t *testing.T) {
	for _, source := range []string{"End the transaction with _one_ of the following:", "Do _not_ continue.", "Choose _either_ option.", "Use *one* option.", "Use **one** option.", "Use ***one*** option."} {
		d, raw := formalFixture(t, "x.md", source+"\n")
		s := d.Slots[0]
		if !strings.Contains(s.Source, "one") && !strings.Contains(s.Source, "not") && !strings.Contains(s.Source, "either") {
			t.Fatal("emphasized language became machine atom", s)
		}
		for _, p := range s.Protected {
			if strings.Contains(p.Raw, "one") || strings.Contains(p.Raw, "not") || strings.Contains(p.Raw, "either") {
				t.Fatal("natural language locked", p)
			}
		}
		translated := strings.NewReplacer("one", "一个", "not", "不要", "either", "任一").Replace(s.Source)
		got := checkFormal(t, d, raw, formalTarget(d, []string{translated}), true)
		if !bytes.Contains(got, []byte("_")) && !bytes.Contains(got, []byte("*")) {
			t.Fatal("emphasis lost")
		}
	}
	d, raw := formalFixture(t, "x.md", "Use _one_ and _either_ option.\n")
	checkFormal(t, d, raw, formalTarget(d, []string{"使用 ⟦P0⟧一个⟦P1⟧或 ⟦P2⟧任一⟦P3⟧选项。"}), true)
	checkFormal(t, d, raw, formalTarget(d, []string{"使用 ⟦P1⟧一个⟦P0⟧或 ⟦P2⟧任一⟦P3⟧选项。"}), false)
	checkFormal(t, d, raw, formalTarget(d, []string{"使用 ⟦P0⟧一个 ⟦P2⟧任一⟦P1⟧选项⟦P3⟧。"}), false)
	for _, tech := range []string{"SQL", "API", "go.mod", "foo_bar", "https://example.com", "--flag"} {
		visible, locked, err := protectBlock([]byte("Use _"+tech+"_ here."), "markdown/Paragraph")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, p := range locked {
			found = found || p.Raw == tech
		}
		if !found || strings.Contains(visible, tech) {
			t.Fatalf("technical identity exposed: %s: %s %+v", tech, visible, locked)
		}
	}
}
