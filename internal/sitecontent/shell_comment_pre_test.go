package sitecontent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const shellCommentTestMetadata = "---\ntitle: The Go Programming Language\nsummary: Build services with Go.\ntemplate: true\n---\n"

func TestShellCommentPreLanguageContract(t *testing.T) {
	raw := []byte(shellCommentTestMetadata + `<textarea>// You can edit this code!
// Click here and start typing.
package main
func main() {}
</textarea><pre class="some-output"><noscript>Hello, 世界</noscript></pre><pre><code class="language-go">// Teaching comment.
package main
</code></pre>`)
	s, err := ExtractShellLanguageSource("_content/index.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	a := ShellPageAdapter{*s, raw}
	source, _, err := a.CommentSourceBlocks()
	if err != nil {
		t.Fatal(err)
	}
	target, err := a.CommentTargetBlocks(raw)
	if err != nil || !reflect.DeepEqual(source, target) {
		t.Fatal("Shell source/target selection differs", err)
	}
	if len(source) != 2 {
		t.Fatal("expected explicit Go pre and textarea only", source)
	}
	counts := map[string]int{}
	for _, block := range source {
		if block.Language != "go" || block.DeclaredLanguage != "go" || block.Unsupported {
			t.Fatal("missing explicit language authority", block)
		}
		counts[block.EnclosingKind]++
		switch block.EnclosingKind {
		case "html-textarea":
			if block.Coordinate != "html-textarea/1" || len(block.Comments) != 2 {
				t.Fatal("textarea coordinates/comments changed", block)
			}
			for i, text := range []string{"You can edit this code!", "Click here and start typing."} {
				if block.Comments[i].Kind != "teaching-comment" || !strings.Contains(block.Comments[i].Text, text) {
					t.Fatal("teaching comment lost", block.Comments)
				}
			}
		case "html-pre":
			if block.Coordinate != "html-pre/2" || len(block.Comments) != 1 || !strings.Contains(block.Comments[0].Text, "Teaching comment.") {
				t.Fatal("explicit pre identity/comments changed", block)
			}
		default:
			t.Fatal("unexpected block", block)
		}
	}
	if counts["html-textarea"] != 1 || counts["html-pre"] != 1 {
		t.Fatal(counts)
	}
}

func TestShellCommentPreExplicitAuthorityRequired(t *testing.T) {
	// Content-based Go inference must not turn an unbound Shell display into
	// a code-comment surface. Unknown explicit declarations still fail closed.
	for _, item := range []struct {
		body        string
		count       int
		unsupported bool
	}{
		{"<pre>package main\n// Teaching comment.\n</pre>", 0, false},
		{"<pre class=\"language-ruby\"># Teaching comment.\n</pre>", 1, true},
	} {
		blocks, _, err := shellPageCommentBlocks("_content/index.md", []byte(shellCommentTestMetadata+item.body))
		if err != nil || len(blocks) != item.count {
			t.Fatal(blocks, err)
		}
		if item.count != 0 && blocks[0].Unsupported != item.unsupported {
			t.Fatal("unsupported declaration gate changed", blocks)
		}
	}
}

func TestGenericPageUnspecifiedPreUnchanged(t *testing.T) {
	raw := []byte("<pre><noscript>Hello, 世界</noscript></pre>")
	blocks, err := PageCommentBlocks("_content/doc/example.html", raw)
	if err != nil || len(blocks) != 1 {
		t.Fatal(blocks, err)
	}
	b := blocks[0]
	want := i18n.AnalyzeCommentCode("Hello, 世界", "", "html-pre/1", false)
	if !b.Unsupported || b.Diagnostic != "UNSUPPORTED_COMMENT_LANGUAGE=unspecified" || b.DeclaredLanguage != "" || b.Coordinate != want.Coordinate || b.SHA256 != want.SHA256 {
		t.Fatal("generic Page unspecified pre contract changed", b)
	}
}
