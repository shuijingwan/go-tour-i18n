package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func roundtripFixture(t *testing.T, name, kind, body string) (Source, *Document, []byte) {
	t.Helper()
	b := []byte(body)
	s := Source{Path: name, Kind: kind, SHA256: digest(b)}
	d, err := ParseDocument(s, b)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Units) != 1 {
		t.Fatalf("expected one unit: %+v", d.Units)
	}
	return s, d, b
}

func TestReconstructUTF8Trim(t *testing.T) {
	for _, padding := range []string{"", " \t", "\u0085\u00a0\u2003"} {
		t.Run(padding, func(t *testing.T) {
			s, d, source := roundtripFixture(t, "x.html", "page", "<p>"+padding+"Tour of Go"+padding+"</p>\n")
			u := d.Units[0]
			if u.Source != "Tour of Go" || string(source[u.Start:u.End]) != u.Source || len(u.Protected) != 0 {
				t.Fatalf("trim/byte coordinates changed: %+v", u)
			}
			b, err := Reconstruct(s, source, map[string]string{u.ID: "Go 语言之旅"})
			if err != nil {
				t.Fatal(err)
			}
			if !utf8.Valid(b) || string(b) != "<p>"+padding+"Go 语言之旅"+padding+"</p>\n" {
				t.Fatalf("partial UTF-8 or whitespace/EOF changed: %q", b)
			}
			s.SHA256 = digest(b)
			next, err := ParseDocument(s, b)
			if err != nil || len(next.Units) != 1 || next.StructureSHA != d.StructureSHA || next.Units[0].Source != "Go 语言之旅" {
				t.Fatalf("UTF-8 round-trip: %+v, %v", next, err)
			}
		})
	}
	// Exercise whitespace within the lexical scanner, as well as span edges.
	text := "自然旅\u2003自然旅"
	visible, _, err := protectBlock([]byte(text), "html/text")
	if err != nil || visible != text || !utf8.ValidString(visible) {
		t.Fatalf("lexical scanner split a rune: %q, %v", visible, err)
	}
}

func TestReconstructTargetLexicalAtoms(t *testing.T) {
	for _, fixture := range []struct{ name, kind, body string }{
		{"x.yaml", "data", "- title: CLIs\n  url: /fixed\n"},
		{"x.html", "page", "<p>CLIs</p>\n"},
		{"x.html", "page", "<div>CLIs</div>\n"},
		{"x.md", "page", "CLIs\n"},
		{"x.md", "page", "<div>CLIs</div>\n"},
	} {
		t.Run(fixture.name+fixture.body, func(t *testing.T) {
			s, d, b := roundtripFixture(t, fixture.name, fixture.kind, fixture.body)
			if d.Units[0].Source != "CLIs" || len(d.Units[0].Protected) != 0 {
				t.Fatal("source protection changed", d.Units)
			}
			for _, target := range []string{"CLI", "API", "HTTP", "CPU", "URL"} {
				got, err := Reconstruct(s, b, map[string]string{d.Units[0].ID: target})
				if err != nil || string(got) != strings.Replace(fixture.body, "CLIs", target, 1) {
					t.Fatalf("target lexical atom %s: %q, %v", target, got, err)
				}
			}
		})
	}
}

func TestReconstructSourceProtectionStrict(t *testing.T) {
	s, d, b := roundtripFixture(t, "x.md", "page", "This workshop will walk you through building a CLI app with [a guide](/fixed).\n")
	u := d.Units[0]
	if len(u.Protected) != 3 || u.Protected[0].Raw != "CLI" || u.Protected[0].Role != "atom" {
		t.Fatal("source CLI atom missing", u)
	}
	good := "使用 ⟦P0⟧ 构建应用，参阅 ⟦P1⟧指南⟦P2⟧。"
	got, err := Reconstruct(s, b, map[string]string{u.ID: good})
	if err != nil || !bytes.Contains(got, []byte("CLI")) || !bytes.Contains(got, []byte("[指南](/fixed)")) {
		t.Fatalf("source raw bytes not preserved: %q, %v", got, err)
	}
	for _, bad := range []string{
		strings.ReplaceAll(good, "⟦P0⟧", ""),
		strings.ReplaceAll(good, "⟦P0⟧", "CLI"),
		strings.ReplaceAll(good, "⟦P0⟧", "⟦P00⟧"),
		good + "⟦P0⟧", good + "⟦P99⟧",
		"⟦P2⟧指南⟦P1⟧ 使用 ⟦P0⟧。",
		strings.ReplaceAll(good, "⟦P2⟧", "](/changed)"),
		good + " <a href=\"/new\">链接</a>",
		good + " <code>CLI</code>", good + " [链接](/new)",
		good + "\n\n# New heading", good + " {{.title}}",
		good + "\r", good + string([]byte{0xff}),
	} {
		if _, err := Reconstruct(s, b, map[string]string{u.ID: bad}); err == nil {
			t.Fatalf("invalid target accepted: %q", bad)
		}
	}
	s, d, b = roundtripFixture(t, "x.md", "page", "# Heading {#fixed}\n")
	if _, err := Reconstruct(s, b, map[string]string{d.Units[0].ID: "⟦P0⟧ 标题"}); err == nil {
		t.Fatal("heading suffix moved")
	}
}

func TestReconstructStructureStrict(t *testing.T) {
	for _, fixture := range []struct {
		name, kind, body string
		bad              []string
	}{
		{"x.yaml", "data", "- title: CLIs\n  url: /fixed\n", []string{
			"CLI\n  cta: 新增", "CLI\n- title: New\n  url: /new", "CLI\n  unknown: value",
			"CLI\n  url: /changed", "|\n    CLI", "CLI <a href=\"/new\">链接</a>", "CLI <code>CLI</code>",
		}},
		{"x.html", "page", "<p>CLIs</p>\n", []string{"CLI <a href=\"/new\">链接</a>", "<code>CLI</code>", "CLI</p><p>New"}},
		{"x.md", "page", "CLIs\n", []string{"[CLI](/new)", "CLI\n\nNew", "# CLI", "{{.title}} CLI", "\ufeffCLI"}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			s, d, b := roundtripFixture(t, fixture.name, fixture.kind, fixture.body)
			for _, bad := range fixture.bad {
				if _, err := Reconstruct(s, b, map[string]string{d.Units[0].ID: bad}); err == nil {
					t.Fatalf("structure mutation accepted: %q", bad)
				}
			}
		})
	}
}

// Optional read-only regression against an existing, exact structured bundle
// and output. No import, artifact/evidence save, or target installation occurs.
func TestStructuredOutputReadOnlyRegression(t *testing.T) {
	dir := os.Getenv("GO_TOUR_STRUCTURED_REGRESSION_DIR")
	if dir == "" {
		t.Skip("set GO_TOUR_STRUCTURED_REGRESSION_DIR to existing staging")
	}
	read := func(name, sha string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || digest(b) != sha {
			t.Fatalf("input byte identity: %s: %v", name, err)
		}
		return b
	}
	bundle := read("zh-CN-site-v2-structured-assets-generation.zip", "39ee364d065581165e833613e4500af229e690364dbbe66c7c4f325b60df4845")
	output := read("zh-CN-site-v2-structured-assets-generation-result.with-final-newline.json", "6f944aa120bfb95a2d35e1cdd8ea29cbc59a1896598b3349d4530c9149903214")
	root := repositoryRoot(t)
	m, err := CheckStructuredAssetsBundle(root, bundle)
	if err != nil {
		t.Fatal(err)
	}
	var o StructuredOutput
	if err := StrictJSON(output, &o); err != nil {
		t.Fatal(err)
	}
	if m.Identity != "2550ef8ea0b0746e45d99c3abd963f29161bec7bfefd0a33d36308bb2879b19c" || o.BundleSHA != m.Identity || len(o.Outputs) != 9 {
		t.Fatal("bundle/output identity or scope changed")
	}
	files, err := validateStructuredOutput(root, m.Plan, o)
	if err != nil {
		t.Fatal(err)
	}
	slots := 0
	for _, out := range o.Outputs {
		if len(files[out.Path]) == 0 {
			t.Fatal("missing reconstructed output", out.Path)
		}
		slots += len(out.Slots)
		t.Log("PASS ValidateParsedGlossary + Reconstruct:", out.Path)
	}
	if len(files) != 9 || slots != 78 {
		t.Fatalf("scope: %d outputs, %d slots", len(files), slots)
	}
}
