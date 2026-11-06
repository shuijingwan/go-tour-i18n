package sitecontent

import (
	"os"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestCommentInventoryLearnDocsFencesAndHTML(t *testing.T) {
	for _, fixture := range []struct{ path, raw string }{
		{"x.md", "# Example\n```go\npackage p\n// Return the module path.\nvar path string\n```\n"},
		{"x.md", "# Example\n    // Return the module path.\n    var path string\n"},
		{"x.html", "<pre><code>// Return the module path.\nvar path string\n</code></pre>"},
	} {
		b, err := PageCommentBlocks(fixture.path, []byte(fixture.raw))
		if err != nil || len(b) != 1 || len(b[0].Comments) != 1 || b[0].Comments[0].Kind != "teaching-comment" {
			t.Fatal(b, err)
		}
	}
	// Raw HTML inside a Markdown fence must not be parsed a second time.
	b, err := PageCommentBlocks("x.md", []byte("```html\n<pre>// False positive text</pre>\n```\n"))
	if err != nil || len(b) != 1 || !b[0].Unsupported {
		t.Fatal(b, err)
	}
}

func TestCommentInventoryKnownVulnDatabase(t *testing.T) {
	raw, err := os.ReadFile("../../_content/doc/security/vuln/database.md")
	if err != nil {
		t.Fatal(err)
	}
	b, err := PageCommentBlocks("_content/doc/security/vuln/database.md", raw)
	if err != nil || len(b) != 3 {
		t.Fatal(b, err)
	}
	found := false
	for _, v := range b {
		if v.Unsupported {
			t.Fatal(v)
		}
		for _, s := range v.Comments {
			if s.Kind != "teaching-comment" {
				t.Fatal(s)
			}
			if s.Policy != "protected" {
				t.Fatal("JSON inventory cannot activate formal Page slots", s)
			}
			if strings.Contains(s.Text, "The vulnerability ID.") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("known human sample omitted")
	}
	r, f := i18n.CompareCommentBlocks("learn-docs-v1/database", "zh", b, b)
	if i18n.ValidateCommentCoverage(r, f, true) == nil {
		t.Fatal("known untranslated sample passed")
	}
	for _, code := range []string{"{\"id\": \"https://host/path\", // The vulnerability ID.\n\"path\": string,}", "{\"s\":\"escaped \\\" // literal\"}"} {
		v := i18n.AnalyzeCommentCode(code, "json", "block/1", false)
		if v.Unsupported {
			t.Fatal(v)
		}
	}
}
