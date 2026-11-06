package i18n

import "testing"

func TestCommentBlockDispositionModuleGrammar(t *testing.T) {
	code := "module example.com/m\nrequire example.com/x v1.2.3 // indirect\nretract v1.1.0 // Published accidentally.\n// Read the module documentation.\nreplace example.com/y => \"./quoted//directory\"\n"
	b := AnalyzeModuleCommentCode(code, "block/1")
	if b.Unsupported || len(b.Comments) != 3 {
		t.Fatal(b)
	}
	if b.Comments[0].Kind != "machine-semantic-comment" || b.Comments[1].Kind != "machine-semantic-comment" || b.Comments[2].Kind != "teaching-comment" {
		t.Fatal(b.Comments)
	}
	for _, bad := range []string{"$ go mod init example.com/m\n", "go mod tidy\n", "require <module-path> <version>\n", "module m\nunknown syntax\n"} {
		if !AnalyzeModuleCommentCode(bad, "block/1").Unsupported {
			t.Fatal("non-module grammar accepted", bad)
		}
	}
	for _, good := range []string{"module example.com/m\n", "require example.com/x v1.2.3\n", "go 1.18\n", "retract [v1.0.0,v1.0.5] // Build broken on some platforms.\n"} {
		if AnalyzeModuleCommentCode(good, "block/1").Unsupported {
			t.Fatal(good)
		}
	}
	if _, ok := ModuleCommentAuthority("", "Managing dependencies"); ok {
		t.Fatal("title keyword guessed format")
	}
	if _, ok := ModuleCommentAuthority("", "go.mod file reference"); !ok {
		t.Fatal("explicit format metadata ignored")
	}
	if _, ok := ModuleCommentAuthority("shell", "go.mod file reference"); ok {
		t.Fatal("metadata overrode explicit language")
	}
}
