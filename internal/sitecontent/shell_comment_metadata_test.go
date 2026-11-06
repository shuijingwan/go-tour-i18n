package sitecontent

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestShellCommentMetadataContract(t *testing.T) {
	raw := []byte("---\ntitle: The Go Programming Language\nsummary: Build services with Go.\ntemplate: true\n---\n<h1>Build with Go</h1><textarea>// You can edit this code!\n// Click here and start typing.\npackage main\nfunc main() {}\n</textarea><script>// Machine script\n</script><style>/* Machine style */</style><pre><code class=\"language-go\">// Return the result.\nvar x int\n</code></pre>")
	s, err := ExtractShellLanguageSource("_content/index.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	a := ShellPageAdapter{*s, bytes.Clone(raw)}
	blocks, title, err := a.CommentSourceBlocks()
	if err != nil || title != "The Go Programming Language" {
		t.Fatal(title, err)
	}
	target, err := a.CommentTargetBlocks(raw)
	if err != nil || !reflect.DeepEqual(blocks, target) {
		t.Fatal("source/target comment identity changed", err)
	}
	body, _, _ := ShellPageRenderInput(raw)
	previous, err := pageCommentBlocks("_content/index.md", raw, len(raw)-len(body))
	if err != nil || !reflect.DeepEqual(previous, blocks[:len(previous)]) {
		t.Fatal("existing Markdown/pre comment coordinates changed", err)
	}
	var comments []string
	for _, b := range blocks {
		for _, c := range b.Comments {
			comments = append(comments, c.Text)
		}
		if strings.Contains(b.Code, "Machine") || strings.Contains(b.Code, "Build services") {
			t.Fatal("metadata/script/style entered comment inventory", b)
		}
	}
	for _, want := range []string{"You can edit this code!", "Click here and start typing.", "Return the result."} {
		if !strings.Contains(strings.Join(comments, "\n"), want) {
			t.Fatal("teaching comment omitted", want, comments)
		}
	}
	if digest(raw) != s.SourceSHA || !bytes.Equal(raw, a.Raw) {
		t.Fatal("shell source bytes/identity changed")
	}
}

func TestShellCommentMetadataFailClosed(t *testing.T) {
	for label, raw := range map[string]string{
		"duplicate":        "---\ntitle: X\ntitle: Y\ntemplate: true\n---\n<h1>X</h1>",
		"unknown":          "---\ntitle: X\nunknown: Y\ntemplate: true\n---\n<h1>X</h1>",
		"malformed":        "---\ntitle: [\ntemplate: true\n---\n<h1>X</h1>",
		"invalid-template": "---\ntitle: X\ntemplate: false\n---\n<h1>X</h1>",
		"unterminated":     "---\ntitle: X\nsummary: Y\n",
		"short":            "---",
	} {
		t.Run(label, func(t *testing.T) {
			if _, _, err := shellPageCommentBlocks("_content/index.md", []byte(raw)); err == nil {
				t.Fatal("invalid Shell metadata accepted")
			}
		})
	}
}

func TestGenericPageCommentSummaryRejected(t *testing.T) {
	raw := []byte("---\ntitle: X\nsummary: Y\n---\n<p>Example</p>")
	if _, err := PageCommentBlocks("_content/doc/example.md", raw); err == nil || !strings.Contains(err.Error(), "unknown metadata field summary") {
		t.Fatal("generic Page metadata contract widened", err)
	}
}
