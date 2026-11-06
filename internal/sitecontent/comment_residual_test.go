package sitecontent

import (
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"strings"
	"testing"
)

func TestCommentResidualSourceContext(t *testing.T) {
	blocks, err := PageCommentBlocks("any.md", []byte("You should see this\noutput:\n\n```\n# literal output\n// literal output\n```\n"))
	if err != nil || len(blocks) != 1 {
		t.Fatal(blocks, err)
	}
	if strings.Join(strings.Fields(blocks[0].SourceContext), " ") != "You should see this output:" {
		t.Fatal(blocks[0].SourceContext)
	}
	b := i18n.SourceCommentBlockDisposition(blocks[0], "", nil)
	if b.Unsupported || b.Disposition != "non-code-output-or-literal" || len(b.Comments) != 0 {
		t.Fatal(b)
	}
	blocks, err = PageCommentBlocks("any.html", []byte("<pre>require <var>module-path</var> <var>module-version</var></pre>"))
	if err != nil || len(blocks) != 1 || !blocks[0].VariableMarkup {
		t.Fatal(blocks, err)
	}
	b = i18n.SourceCommentBlockDisposition(blocks[0], "go.mod file reference", nil)
	if b.Unsupported || b.Disposition != "machine-semantic-opaque" {
		t.Fatal(b)
	}
	blocks[0].VariableMarkup = false
	if !i18n.SourceCommentBlockDisposition(blocks[0], "go.mod file reference", nil).Unsupported {
		t.Fatal("metavariables guessed without source markup")
	}
}
