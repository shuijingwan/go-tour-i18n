package i18n

import "testing"

func TestCommentResidualJSONDocumentation(t *testing.T) {
	for _, code := range []string{`{"url":"https://example.test/#fragment","s":"// literal"}`, "\"key\": true, // Enable this setting.\n\"options\": {\"x\": 1}\n", "{\"settings\":{...\n\"x\":true}}"} {
		b, ok := AnalyzeJSONDocumentation(code, "1")
		if !ok || b.Code != code {
			t.Fatal(b, ok)
		}
		for _, c := range b.Comments {
			if code[c.Start:c.End] != c.Text {
				t.Fatal("comment coordinates", c)
			}
		}
	}
	b, ok := AnalyzeJSONDocumentation(`{"url":"https://example.test/#fragment","s":"// literal"}`, "1")
	if !ok || len(b.Comments) != 0 {
		t.Fatal(b)
	}
	for _, code := range []string{"unknown text", "{arbitrary syntax}", "{\"x\":...}", "\"not an object member\"", "{\"x\":1} trailing"} {
		if _, ok := AnalyzeJSONDocumentation(code, "1"); ok {
			t.Fatal("ambiguous JSON accepted", code)
		}
	}
}
