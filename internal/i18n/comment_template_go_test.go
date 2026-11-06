package i18n

import (
	"strings"
	"testing"
)

func TestCommentResidualTemplateGo(t *testing.T) {
	code := "func Reverse(s string) string {\n// Return the original string.\nif len(s) {{raw \"<\"}} 1 { return s }; return \"http://example\"\n}\n"
	b, ok := AnalyzeTemplateGoComments(code, "1")
	if !ok || len(b.Comments) != 1 || b.Code != code || b.Comments[0].Text != "// Return the original string." || !strings.Contains(b.Skeleton, `{{raw "<"}}`) {
		t.Fatal(b, ok)
	}
	for _, bad := range []string{`{{if .}}var x int{{end}}`, `var x = "{{raw \"<\"}}"`, "// {{raw \"<\"}}\nvar x int", `var x = {{other "<"}}`, `func f(){ if x {{raw "<"}} }`} {
		if _, ok := AnalyzeTemplateGoComments(bad, "1"); ok {
			t.Fatal("unproved template Go", bad)
		}
	}
}
