package sitecontent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/languageaudit"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

func auditTourUnit(kind i18n.UnitKind, source string) (*i18n.Catalog, *i18n.TranslationUnit) {
	u := &i18n.TranslationUnit{ID: "fixture/1", Kind: kind, SourcePath: "_content/tour/fixture.article", Source: []byte(source), SourceSHA256: digest([]byte(source))}
	c := &i18n.Catalog{}
	if kind == i18n.UnitKindPage {
		c.Pages = []i18n.Page{{ID: u.ID, Article: "fixture.article", Source: u.Source, SourceSHA256: u.SourceSHA256}}
	} else if kind == i18n.UnitKindExample {
		u.ID = "example:fixture.go"
		u.SourcePath = "_content/tour/fixture.go"
		c.Examples = []i18n.Example{{ID: u.ID, SourcePath: u.SourcePath, Source: u.Source, SourceSHA256: u.SourceSHA256, EligibleTranslation: true, EligibilityKnown: true}}
	}
	return c, u
}

func TestLanguageAuditTourPageUsesPresentAndTeachingCommentAuthority(t *testing.T) {
	source := "* Source title\n\nSee [[https://go.dev][Go]] guide.\n\nUse `func` `type` `var` `fmt.Println` `Record` `Value` and `nil`.\n\n\tvar cache int\n\t// cache stores the current result.\n"
	target := "* 标题\n\n请查看 [[https://go.dev][Go]]指南。\n\n使用 `func` `type` `var` `fmt.Println` `Record` `Value` 和 `nil`。\n\n\tvar cache int\n\t// cache 存储当前结果。\n"
	c, u := auditTourUnit(i18n.UnitKindPage, source)
	sv, tv, err := ProjectTourLanguageAudit(t.TempDir(), c, u, "target.article", []byte(target))
	if err != nil {
		t.Fatal(err)
	}
	r, err := languageaudit.Scan(auditFixturePolicy("zh-CN", "Han"), sv, tv)
	if err != nil || auditClassCount(r, "script-adjacency", languageaudit.Defect) != 1 {
		t.Fatal(r, err)
	}
	for _, d := range r.Diagnostics {
		if d.Decision == languageaudit.Candidate {
			t.Fatal("protected Go syntax or source-bound identifier residual", d)
		}
	}
	s2, t2, err := ProjectTourLanguageAudit(t.TempDir(), c, u, "target.article", []byte(target))
	if err != nil || !reflect.DeepEqual(sv, s2) || !reflect.DeepEqual(tv, t2) {
		t.Fatal("projection determinism", err)
	}
	// Unknown directives fail closed rather than claiming complete coverage.
	_, _, err = ProjectTourLanguageAudit(t.TempDir(), c, u, "target.article", []byte("* 标题\n\n.caption 未支持的附注\n"))
	if err == nil {
		t.Fatal("unmapped representation silently omitted")
	}
}

func TestLanguageAuditTourExampleExcludesCodeAndSourceSelectsComments(t *testing.T) {
	source := `package main
import "fmt"
type Record struct { Value string }
//go:noinline
func Exported() {
 value := "Command Prompt issue package func type var . -- "
 fmt.Println(value)
 // Exported prints the current result.
 // Output: fixed
}
`
	target := strings.Replace(source, "Exported prints the current result.", "Exported 打印当前结果。", 1)
	c, u := auditTourUnit(i18n.UnitKindExample, source)
	sv, tv, err := ProjectTourLanguageAudit(t.TempDir(), c, u, "target.go", []byte(target))
	if err != nil {
		t.Fatal(err)
	}
	r, err := languageaudit.Scan(auditFixturePolicy("zh-CN", "Han"), sv, tv)
	if err != nil || len(r.Diagnostics) != 0 {
		t.Fatal("Go syntax/literal/identifier false positives", r, err)
	}
	if len(tv.Blocks) != 1 || tv.Blocks[0].Context != "teaching-comment" {
		t.Fatal("comment coverage", tv)
	}
	for _, mutation := range []string{strings.Replace(target, "fixed", "changed", 1), strings.Replace(target, "Command Prompt issue", "modified", 1), "package broken {"} {
		if _, _, err := ProjectTourLanguageAudit(t.TempDir(), c, u, "target.go", []byte(mutation)); err == nil {
			t.Fatal("immutable Go source accepted")
		}
	}
	// A natural-language residue is still a candidate, never a code exclusion.
	target = strings.Replace(source, "Exported prints the current result.", "Exported 仍有 issue。", 1)
	source = strings.Replace(source, "prints the current result.", "prints the issue result.", 1)
	c, u = auditTourUnit(i18n.UnitKindExample, source)
	sv, tv, err = ProjectTourLanguageAudit(t.TempDir(), c, u, "target.go", []byte(target))
	if err != nil {
		t.Fatal(err)
	}
	r, err = languageaudit.Scan(auditFixturePolicy("zh-CN", "Han"), sv, tv)
	if err != nil || auditClassCount(r, "source-prose-overlap", languageaudit.Candidate) != 1 {
		t.Fatal(r, err)
	}
}

func TestLanguageAuditTourUIPlainRichAndOpaquePlaceholders(t *testing.T) {
	plain := ui.Message{Kind: "plain", Text: "普通 <code> 字面量"}
	d, err := ProjectTourUIMessageAudit("ui.json", "sample.text", plain)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Blocks) != 1 || d.Blocks[0].Pieces[0].Text != plain.Text {
		t.Fatal("plain message interpreted as markup", d)
	}
	rich := ui.Message{Kind: "rich", Text: `<p>文字<a href="https://go.dev">Go</a>边界。</p>`}
	d, err = ProjectTourUIMessageAudit("ui.json", "sample.rich", rich)
	if err != nil {
		t.Fatal(err)
	}
	r, err := languageaudit.Scan(auditFixturePolicy("zh-CN", "Han"), languageaudit.Document{}, d)
	if err != nil || auditClassCount(r, "script-adjacency", languageaudit.Defect) != 2 {
		t.Fatal(r, err)
	}
	if _, err := ProjectTourUIMessageAudit("ui.json", "sample.bad", ui.Message{Kind: "rich", Text: "<script>bad</script>"}); err == nil {
		t.Fatal("UI whitelist bypass")
	}
}

func TestLanguageAuditTourUIPlaceholderBinding(t *testing.T) {
	source := ui.Message{Kind: "plain", Text: "{count} tests failed."}
	target := ui.Message{Kind: "plain", Text: "{count} 个测试失败。"}
	sv, err := ProjectTourUIMessageAudit("ui.json", "sample.counter", source, source)
	if err != nil {
		t.Fatal(err)
	}
	tv, err := ProjectTourUIMessageAudit("ui.json", "sample.counter", target, source)
	if err != nil {
		t.Fatal(err)
	}
	r, err := languageaudit.Scan(auditFixturePolicy("zh-CN", "Han"), sv, tv)
	if err != nil || len(r.Diagnostics) != 0 {
		t.Fatal("placeholder residual", r, err)
	}
	if _, err := ProjectTourUIMessageAudit("ui.json", "sample.counter", ui.Message{Kind: "plain", Text: "{other} 失败。"}, source); err == nil {
		t.Fatal("unknown runtime placeholder")
	}
}
