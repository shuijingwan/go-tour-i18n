package shellyaml

import (
	"testing"
)

func TestFrozenSubsetRejectsUnknownSyntax(t *testing.T) {
	for _, raw := range []string{
		"name: &identity text\n", "name: *identity\n", "name: !!str text\n",
		"name: text\nname: duplicate\n", "name: text\n---\nname: next\n",
		"name: >\n  folded text\n", "name: |+\n  text\n",
		"name: \"multiline\n  text\"\n", "name: \"text\" trailing\n",
		"name: \"invalid \\x61 escape\"\n", "name: [a b, c\n", "name: {a: text, a: duplicate}\n",
		"name: text\n\tchild: illegal\n", "name: text\n child: wrong indentation\n",
	} {
		if _, err := Parse([]byte(raw)); err == nil {
			t.Fatalf("unsupported YAML accepted: %q", raw)
		}
	}
}

func TestFrozenSubsetNestedFlowLiteralAndUnicode(t *testing.T) {
	n, err := Parse([]byte("main:\n  - name: 'Jane''s menu'\n    links:\n      [{name: Go, url: 'https://go.dev/',}, {name: 中文}]\n    quote: |-\n      Use Go.\n      Keep structure.\n"))
	if err != nil {
		t.Fatal(err)
	}
	item := n.Content[1].Content[0]
	if item.Content[1].Value != "Jane's menu" || item.Content[3].Content[1].Content[1].Value != "中文" || item.Content[5].Value != "Use Go.\nKeep structure." {
		t.Fatal("frozen subset semantic decoding changed", item)
	}
}
