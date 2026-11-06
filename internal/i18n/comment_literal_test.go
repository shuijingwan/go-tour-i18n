package i18n

import "testing"

func TestCommentResidualLiteral(t *testing.T) {
	for _, f := range []struct{ code, context string }{
		{"root/\n  go.mod\n  cmd/\n    main.go", ""},
		{"<home>/\n |-- hello/\n |-- greetings/", ""},
		{"<prefix>/<description>", ""},
		{"v1.2.3-alpha\nvx.4.x", ""},
		{"HTTP/1.1 200 OK\nContent-Type: application/json\nContent-Length: 2\n\n{\"a\": 1}\n", ""},
		{"module example.com/m/v2 v2.0.0", ""},
		{"(value, type)", "An interface is a tuple of a value and a concrete type:"},
		{"go test fuzz v1\n[]byte(\"hello\")\nint64(572293)", ""},
		{"# literal output\n// literal output", "You should see this output:"},
		{"ValidString reports whether s is valid.", "First, consider the docs for utf8.ValidString."},
		{"0. hello [initial]\n1. world [changed]\n...", "For an input, the mutation strategy may look like the following:"},
	} {
		if _, ok := SourceLiteralProof(f.code, f.context); !ok {
			t.Fatal(f)
		}
	}
	for _, s := range []string{"unknown language", "root/\n  // Teaching comment.", "(value, type)", "go test fuzz v1\nunknown()", "v1.x; unknown"} {
		if _, ok := SourceLiteralProof(s, ""); ok {
			t.Fatal("unproved literal", s)
		}
	}
}
