package i18n

import "testing"

func TestParsedGlossaryFoundation(t *testing.T) {
	old := []byte("locale: zh-CN\nmandatory:\n  channel: 通道\n")
	next := []byte("locale: zh-CN\nmandatory:\n  channel: 信道\n")
	delta, err := ParsedGlossaryDelta("zh-CN", old, next)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := ParsedGlossaryImpact("A channel carries values", "通道传递值", delta); ok {
		t.Fatal("affected channel carried")
	}
	if ok, _ := ParsedGlossaryImpact("SQL transactions", "SQL 事务", delta); !ok {
		t.Fatal("unaffected parser context cannot carry")
	}
	g := &Glossary{Mandatory: map[string]string{"Run": "运行"}, Keep: []string{"Go"}, Forbidden: []string{"错误译法"}}
	if err := ValidateParsedGlossary("Run", "Run", g); err == nil {
		t.Fatal("mandatory exact label bypass")
	}
	if err := ValidateParsedGlossary("Learn Go", "学习", g); err == nil {
		t.Fatal("keep bypass")
	}
	if err := ValidateParsedGlossary("Text", "错误译法", g); err == nil {
		t.Fatal("forbidden bypass")
	}
	if err := ValidateParsedGlossary("Run", "运行", g); err != nil {
		t.Fatal(err)
	}
}
