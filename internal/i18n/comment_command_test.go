package i18n

import "testing"

func TestCommentResidualBareCommand(t *testing.T) {
	for _, s := range []string{"cd ..\nmkdir hello\ncd hello\n", "cd %HOMEPATH%\n", "go install example.com/m@latest\n", "go mod init <company-name>/stringtools\n", "GOPRIVATE=*.corp.example.com,*.research.example.com\n", "export PATH=\"$PATH:$(go env GOPATH)/bin\"\n"} {
		if _, ok := BareDocumentationCommand(s); !ok {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"unknown some words", "go minimum-go-version", "go test fuzz v1\nstring(\"x\")", "echo '# literal'", "cd foo # Teaching comment.", "cd > output", "go get x; unknown"} {
		if _, ok := BareDocumentationCommand(s); ok {
			t.Fatal("unproved command", s)
		}
	}
}
