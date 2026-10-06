package sitecontent

import (
	"bytes"
	"strings"
	"testing"
)

func TestSiteRepairLinguisticBlocks(t *testing.T) {
	for _, fixture := range []struct{ source, target string }{
		{"## Create a folder for your code {#create_folder}\n", "为代码创建文件夹 ⟦P0⟧"},
		{"For other tutorials, see [Tutorials](/doc/tutorial/index).\n", "⟦P0⟧教程⟦P1⟧中可阅读其他教程。"},
		{"Use `go mod tidy` to update the module.\n", "更新模块时请使用 ⟦P0⟧。"},
	} {
		b := []byte(fixture.source)
		s := Source{Path: "x.md", Kind: "page", SHA256: digest(b)}
		d, err := ParseDocument(s, b)
		if err != nil {
			t.Fatal(err)
		}
		if len(d.Units) != 1 {
			t.Fatalf("not one coherent unit: %+v", d.Units)
		}
		u := d.Units[0]
		targets := map[string]string{u.ID: fixture.target}
		first, err := Reconstruct(s, b, targets)
		if err != nil {
			t.Fatal(err)
		}
		second, err := Reconstruct(s, b, targets)
		if err != nil || !bytes.Equal(first, second) {
			t.Fatal("nondeterministic reconstruction", err)
		}
		for _, machine := range []string{"{#create_folder}", "/doc/tutorial/index", "`go mod tidy`"} {
			if strings.Contains(fixture.source, machine) && !bytes.Contains(first, []byte(machine)) {
				t.Fatal("protected identity changed", machine)
			}
		}
		for _, bad := range []string{strings.ReplaceAll(fixture.target, "⟦P0⟧", ""), fixture.target + "⟦P0⟧", fixture.target + "<script>bad</script>"} {
			targets[u.ID] = bad
			if _, err := Reconstruct(s, b, targets); err == nil {
				t.Fatalf("invalid target accepted: %s", bad)
			}
		}
	}
}

func TestSiteRepairCompleteScalarAndCode(t *testing.T) {
	body := "- title: A complete title\n  content: Read <a href=\"/doc/tutorial/index\">the tutorial</a> for other examples.\n    It covers several examples.\n  url: /machine-route\n  credits: 5\n"
	s := Source{Path: "x.yaml", Kind: "data", SHA256: digest([]byte(body))}
	d, err := ParseDocument(s, []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Units) != 2 {
		t.Fatalf("scalar fragmented: %+v", d.Units)
	}
	targets := map[string]string{}
	for _, u := range d.Units {
		targets[u.ID] = "译文 " + u.Source
	}
	b, err := Reconstruct(s, []byte(body), targets)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("url: /machine-route\n  credits: 5")) {
		t.Fatal("machine YAML fields changed")
	}
	md := "# Heading\n\nA sentence, with [one link](/x) and `go run ./cmd`.\n\n```go\nfmt.Println(\"unchanged\")\n```\n\n<pre>go mod tidy\noutput</pre>\n\n<p id=\"fixed\">Use <code>GoIdentifier</code> with <a href=\"/x#fixed\" class=\"machine\">a label</a>.</p>\n"
	s = Source{Path: "x.md", Kind: "page", SHA256: digest([]byte(md))}
	d, err = ParseDocument(s, []byte(md))
	if err != nil {
		t.Fatal(err)
	}
	targets = map[string]string{}
	for _, u := range d.Units {
		if !hasLanguage(u.Source) {
			t.Fatal("punctuation/locked fragment unit", u.Source)
		}
		targets[u.ID] = "译文 " + u.Source
	}
	b, err = Reconstruct(s, []byte(md), targets)
	if err != nil {
		t.Fatal(err)
	}
	for _, literal := range []string{"fmt.Println(\"unchanged\")", "<pre>go mod tidy\noutput</pre>", "id=\"fixed\"", "<code>GoIdentifier</code>", "href=\"/x#fixed\" class=\"machine\""} {
		if !bytes.Contains(b, []byte(literal)) {
			t.Fatal("machine bytes changed", literal)
		}
	}
}

func TestSiteRepairNoMachineFragmentUnits(t *testing.T) {
	for _, body := range []string{".\n", ",\n", "## {#create_folder}\n", "{{.title}}\n", "`go mod tidy`\n"} {
		s := Source{Path: "x.md", Kind: "page", SHA256: digest([]byte(body))}
		d, err := ParseDocument(s, []byte(body))
		if err != nil {
			t.Fatal(err)
		}
		if len(d.Units) != 0 {
			t.Fatalf("machine/punctuation fragment became unit: %+v", d.Units)
		}
	}
}
