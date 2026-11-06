package sitecontent

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormalPageInlineDirectLanguage(t *testing.T) {
	for _, inline := range []string{"<code>x</code>", "<var>x</var>", "<kbd>x</kbd>", "<samp>x</samp>", `<a href="/fixed">command</a>`, `<a href="/fixed"><code>go test</code> command</a>`} {
		for _, nested := range []string{"", "<p>Nested paragraph.</p>", "<ul><li>Nested item.</li></ul>"} {
			source := "<ol><li>Run\n  " + inline + " now with <code>y</code>." + nested + "</li></ol>"
			d, raw := formalFixture(t, "x.html", source)
			values := []string{}
			for _, s := range d.Slots {
				values = append(values, strings.NewReplacer("Run", "运行", "now with", "现在使用", "command", "命令", "Nested paragraph.", "嵌套段落。", "Nested item.", "嵌套条目。").Replace(s.Source))
			}
			if !strings.Contains(d.Units[0].Source, "Run") || (nested != "" && len(d.Slots) != 2) {
				t.Fatal("direct language missing", d.Slots)
			}
			for _, p := range d.Units[0].Protected {
				if p.Role == "boundary" && strings.Contains(p.Raw, "Run") {
					t.Fatal("prose in boundary", p)
				}
			}
			target := formalTarget(d, values)
			got := checkFormal(t, d, raw, target, true)
			for _, machine := range []string{`href="/fixed"`, "<code>go test</code>", "<code>x</code>", "<var>x</var>", "<kbd>x</kbd>", "<samp>x</samp>", "<code>y</code>"} {
				if strings.Contains(source, machine) && !bytes.Contains(got, []byte(machine)) {
					t.Fatal("machine bytes changed", machine)
				}
			}
			for _, bad := range []string{"<b>new</b>", "</li><li>new", `<a href="/changed">new</a>`} {
				checkFormal(t, d, raw, strings.Replace(target, "运行", "运行"+bad, 1), false)
			}
			for _, p := range d.Units[0].Protected {
				if p.Raw == "<code>y</code>" || p.Role == "open" {
					checkFormal(t, d, raw, strings.Replace(target, p.Token, "", 1), false)
					checkFormal(t, d, raw, strings.Replace(target, p.Token, strings.ReplaceAll(p.Raw, "y", "z"), 1), false)
					if nested != "" {
						checkFormal(t, d, raw, strings.Replace(strings.Replace(target, p.Token, "", 1), "嵌套", p.Token+"嵌套", 1), false)
					}
				}
			}
		}
	}
}

func TestFormalPageOpaqueHTMLLanguage(t *testing.T) {
	for _, tag := range []string{"pre", "script", "style", "textarea"} {
		for _, body := range []string{"Opaque machine content.", "<p>Opaque machine content.</p>"} {
			source := "<div>Visible language.</div><" + tag + ">" + body + "</" + tag + ">"
			d, raw := formalFixture(t, "x.html", source)
			if strings.Contains(d.Units[0].Source, "Opaque") {
				t.Fatal("opaque language exposed", tag)
			}
			got := checkFormal(t, d, raw, formalTarget(d, []string{"可见文字。"}), true)
			if !bytes.Contains(got, []byte("<"+tag+">"+body+"</"+tag+">")) {
				t.Fatal("opaque bytes changed")
			}
			for _, p := range d.Units[0].Protected {
				if strings.Contains(p.Raw, "Opaque") {
					checkFormal(t, d, raw, strings.Replace(d.Units[0].Source, p.Token, "changed", 1), false)
				}
			}
		}
	}
}

func TestFrozenPageInlineLanguage(t *testing.T) {
	root := repositoryRoot(t)
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"add-a-test.html": "At the command line", "call-module-code.html": "Create a", "compile-install.html": "Discover the Go install path", "create-module.html": "Open a command prompt", "getting-started.html": "In your Go code", "greetings-multiple-people.html": "At the command line", "handle-errors.html": "At the command line"}
	seen := 0
	for _, d := range docs {
		name := d.Path[strings.LastIndex(d.Path, "/")+1:]
		phrase, ok := want[name]
		if !ok {
			continue
		}
		seen++
		if !strings.Contains(d.Units[0].Source, phrase) {
			t.Fatal("finding not open", d.Path, phrase)
		}
		for _, p := range d.Units[0].Protected {
			if p.Role == "boundary" && strings.Contains(p.Raw, phrase) {
				t.Fatal("finding still boundary", d.Path, p)
			}
		}
		got := checkFormal(t, d, raw[d.Path], d.Units[0].Source, true)
		if !bytes.Equal(got, raw[d.Path]) {
			t.Fatal("frozen roundtrip", d.Path)
		}
	}
	if seen != 7 {
		t.Fatal("frozen exact set", seen)
	}
}
