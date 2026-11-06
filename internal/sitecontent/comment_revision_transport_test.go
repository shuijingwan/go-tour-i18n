package sitecontent

import (
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestCommentRevisionBodyProjection(t *testing.T) {
	for _, fixture := range []struct{ path, text string }{
		{"x.md", "# Example\n```go\npackage p\nvar name string\n// Return the name.\n//go:generate echo keep\n```\n"},
		{"x.html", "<pre><code>package p\nvar name string\n// Return the name.\n//go:generate echo keep\n</code></pre>"},
		{"x.md", "# Example\n```json\n{\n// The vulnerability ID.\n\"id\": string\n}\n```\n"},
	} {
		a := PageAdapter{Document: Document{Path: fixture.path}, Source: []byte(fixture.text)}
		bodies, err := a.CommentRevisionBodies(a.Source)
		if err != nil || len(bodies) != 1 {
			t.Fatal(fixture.path, len(bodies), err)
		}
		body := bodies[0]
		text, tokens, err := body.Adapter.ProtectedInput(&i18n.Glossary{})
		if err != nil {
			t.Fatal(err)
		}
		translated := "返回名称。"
		for _, token := range tokens {
			translated += token
		}
		got, err := body.Adapter.Restore(translated)
		if err != nil || !strings.Contains(string(got), "返回名称") {
			t.Fatal(text, string(got), err)
		}
		if _, err := body.Adapter.Restore(text); err != nil {
			t.Fatal("baseline not reversible", err)
		}
		for _, bad := range []string{"", translated + "\nnew line", translated + " // new comment", translated + " newName", translated + " unknown()"} {
			if _, err := body.Adapter.Restore(bad); err == nil {
				t.Fatal("unsafe body accepted", bad)
			}
		}
	}
}

func TestCommentRevisionBodyIndentedProjection(t *testing.T) {
	for _, text := range []string{
		"# Example\n\n    package p\n    var name string\n    // Return the name.\n    //go:generate echo keep\n",
		"# Example\n\n- Example:\n\n    ```go\n    package p\n    var name string\n    // Return the name.\n    //go:generate echo keep\n    ```\n",
	} {
		a := PageAdapter{Document: Document{Path: "x.md"}, Source: []byte(text)}
		bodies, err := a.CommentRevisionBodies(a.Source)
		if err != nil || len(bodies) != 1 {
			t.Fatal(len(bodies), err)
		}
		body := bodies[0]
		if string(a.Source[body.Start:body.End]) != "Return the name." {
			t.Fatal("raw coordinate lost indentation", body.Start, body.End)
		}
		_, tokens, err := body.Adapter.ProtectedInput(&i18n.Glossary{})
		if err != nil || len(tokens) != 1 {
			t.Fatal(tokens, err)
		}
		got, err := body.Adapter.Restore("返回" + tokens[0] + "。")
		if err != nil || string(got) != "返回name。" {
			t.Fatal(string(got), err)
		}
	}
}

func TestCommentRevisionBodyTechnicalAtomMorphology(t *testing.T) {
	const source = "A list of IDs of the same vulnerability in other databases."
	text, locked, err := protectTeachingBody([]byte(source), 0, len(source), i18n.TeachingCommentBody{})
	if err != nil {
		t.Fatal(err)
	}
	a := revisionCommentBody{unit: Unit{Source: text, Protected: locked}, original: source}
	for _, target := range []string{"其他数据库中同一漏洞的 ID 列表。", "使用 JSON 和 API 表示 ID。"} {
		if _, err := a.Restore(target); err != nil {
			t.Fatal("registered technical atom is not a new identity", target, err)
		}
	}
	for _, target := range []string{"引入 newName。", "引入 new_identifier。", "调用 unknown()。", "调用 fmt.Println", "参考 example.go", "插入 // comment", "说明\nnew code"} {
		if _, err := a.Restore(target); err == nil {
			t.Fatal("new code/structure identity accepted", target)
		}
	}
	const original = "The vulnerability ID."
	text, locked, err = protectTeachingBody([]byte(original), 0, len(original), i18n.TeachingCommentBody{})
	if err != nil || len(locked) != 1 {
		t.Fatal(locked, err)
	}
	bound := revisionCommentBody{unit: Unit{Source: text, Protected: locked}, original: original}
	if _, err := bound.Restore("漏洞 " + locked[0].Token + " 和 ID。"); err == nil {
		t.Fatal("source-bound atom duplication accepted")
	}
}
