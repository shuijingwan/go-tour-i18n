package ui

import "testing"

func TestVisibleMessageTextUsesRichMarkupAuthority(t *testing.T) {
	text, err := VisibleMessageText(Message{Kind: "rich", Text: `<p>Go &amp; Tour <a href="https://go.dev">label</a></p>`})
	if err != nil || text != " Go & Tour  label  " {
		t.Fatalf("visible=%q err=%v", text, err)
	}
	text, err = VisibleMessageText(Message{Kind: "plain", Text: `<a href="https://go.dev">literal</a>`})
	if err != nil || text != `<a href="https://go.dev">literal</a>` {
		t.Fatalf("plain message changed: %q %v", text, err)
	}
	if _, err := VisibleMessageText(Message{Kind: "rich", Text: `<a href="https://bad.example">label</a>`}); err == nil {
		t.Fatal("unknown markup accepted")
	}
}
