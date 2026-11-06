package sitecontent

import (
	"strings"
	"testing"
)

func TestCommentBlockDispositionSourceHTMLPreflight(t *testing.T) {
	raw := []byte(`<!--{"Title":"Future source","Sidebar":"bug","literal":"<pre>ignored</pre>"}-->
<pre class="language-go">// Return the module path.
var path string
</pre>`)
	if _, err := PageCommentBlocks("future.html", raw); err == nil {
		t.Fatal("formal Page metadata schema was relaxed")
	}
	b, err := SourceHTMLCommentBlocks(raw)
	if err != nil || len(b) != 1 || len(b[0].Comments) != 1 || b[0].Unsupported {
		t.Fatal(b, err)
	}
	if b[0].DeclaredLanguage != "go" || b[0].Start != strings.Index(string(raw), `<pre class=`) || b[0].End != len(raw) {
		t.Fatal("source declaration/absolute coordinates lost", b[0])
	}
	if _, err := SourceHTMLCommentBlocks([]byte("<pre>unclosed")); err == nil {
		t.Fatal("malformed pre accepted")
	}
}
