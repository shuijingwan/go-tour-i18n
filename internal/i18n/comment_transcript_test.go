package i18n

import "testing"

func TestCommentResidualTranscript(t *testing.T) {
	for _, code := range []string{
		"$ go test\nPASS\nok example.com/m 0.1s\n",
		"$ echo '# not a comment' https://example.test/#fragment\n# output\n// output\n-- output\n",
		"C:\\Users\\you\\data-access> set DBUSER=username\nC:\\> cd %HOMEPATH%\n",
		"mysql> select * from album;\n-- output, never SQL source\nmysql>\n",
		"$ curl https://example.test \\\n  --data '{\"id\": \"1\"}'\n{}\n",
		"~ go test -fuzz FuzzFoo\nPASS\n",
		"$ export PATH=\"$PATH:$(go env GOPATH)/bin\"\n",
	} {
		r, ok := TerminalTranscript(code)
		if !ok {
			t.Fatal("valid transcript rejected", code)
		}
		pos := 0
		for _, item := range r {
			if item.Start != pos || item.Text != code[item.Start:item.End] {
				t.Fatal("record exact-set", r)
			}
			pos = item.End
		}
		if pos != len(code) {
			t.Fatal("uncovered bytes", r)
		}
	}
	for _, bad := range []string{"unclear language\n", "C:\\path\\to\\file", "user> command\n", "$ echo x # Teaching comment.\n", "$ cat <<EOF\n# body\nEOF\n", "$ echo 'unclosed\n", "mysql> select '-- literal'; -- Teaching comment.\n", "$ echo \"$(echo x # Hidden comment.)\"\n", "$ echo \"`echo x # Hidden comment.`\"\n"} {
		if _, ok := TerminalTranscript(bad); ok {
			t.Fatal("ambiguous transcript accepted", bad)
		}
	}
}
