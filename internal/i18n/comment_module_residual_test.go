package i18n

import "testing"

func TestCommentResidualModuleWork(t *testing.T) {
	for _, code := range []string{"module example/m\ngo 1.20\nrequire example.com/x v0.0.0-unpublished\n", "require example.com/x v1.0.0 // indirect\n", "module example/m\nreplace example.com/x => \"./quoted//path\"\n"} {
		if b := AnalyzeModuleCommentCode(code, "1"); b.Unsupported {
			t.Fatal(b)
		}
	}
	for _, code := range []string{"go 1.18\nuse ./hello\n", "go 1.18\nuse (\n ./hello\n ./example/hello // Read this module.\n)\n"} {
		if b := AnalyzeWorkCommentCode(code, "1"); b.Unsupported {
			t.Fatal(b)
		}
	}
	for _, code := range []string{"go minimum-go-version", "require module-path module-version", "unknown x", "go install example.com/x@latest", "use (\n\"unclosed\n"} {
		if !AnalyzeModuleCommentCode(code, "1").Unsupported || !AnalyzeWorkCommentCode(code, "1").Unsupported {
			t.Fatal("semantic laxness introduced", code)
		}
	}
}
