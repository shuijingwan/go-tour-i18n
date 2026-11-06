package i18n

import (
	"strings"
	"testing"
)

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

func TestParsedGlossaryKeepTermIdentity(t *testing.T) {
	g := &Glossary{Keep: []string{"Go"}, Preferred: map[string]string{"gopher": "Gopher"}}
	for _, fixture := range []struct {
		name, source, target string
		wantFailure          bool
	}{
		{name: "Go Gopher natural translation", source: "Go gophers with wrench", target: "拿着扳手的 Go Gopher"},
		{name: "Go Gopher exact identity", source: "The Go Gopher", target: "Go Gopher"},
		{name: "Google is not Go", source: "Google", target: "Google"},
		{name: "Google does not require Go", source: "Google services", target: "谷歌服务"},
		{name: "Gopher does not require Go", source: "The Gopher", target: "吉祥物"},
		{name: "punctuation", source: "Go.", target: "Go。"},
		{name: "CJK adjacency", source: "Go gophers with wrench", target: "拿着扳手的Go Gopher"},
		{name: "Japanese adjacency", source: "The Go language", target: "Go言語"},
		{name: "Korean adjacency", source: "The Go language", target: "Go언어"},
		{name: "ordinary verb", source: "A lot can go right.", target: "很多事情都会顺利得多。"},
		{name: "ordinary inflection", source: "Everything goes well.", target: "一切顺利。"},
		{name: "existing Where to Go exception", source: "Where to Go", target: "下一步"},
		{name: "deleted", source: "Go gophers with wrench", target: "拿着扳手的 Gopher", wantFailure: true},
		{name: "renamed", source: "The Go language", target: "Golang 语言", wantFailure: true},
		{name: "Google cannot replace Go", source: "Go", target: "Google", wantFailure: true},
		{name: "Gopher cannot replace Go", source: "Go", target: "Gopher", wantFailure: true},
		{name: "lowercase identity drift", source: "Go", target: "go", wantFailure: true},
		{name: "uppercase identity drift", source: "Go", target: "GO", wantFailure: true},
		{name: "inflection cannot replace identity", source: "Go", target: "going", wantFailure: true},
		{name: "ASCII suffix", source: "Go", target: "Go2", wantFailure: true},
		{name: "ASCII prefix", source: "Go", target: "myGo", wantFailure: true},
		{name: "identifier boundary", source: "Go", target: "Go_language", wantFailure: true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			err := ValidateParsedGlossary(fixture.source, fixture.target, g)
			if (err != nil) != fixture.wantFailure {
				t.Fatalf("source=%q target=%q error=%v wantFailure=%t", fixture.source, fixture.target, err, fixture.wantFailure)
			}
			if err != nil && !strings.Contains(err.Error(), `keep term "Go" changed`) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParsedGlossaryKeepStructuredTestimonialOrdinaryGo(t *testing.T) {
	// Exact frozen slot e99255fdded8bac3a6a20aaceb62d4a60b55a75e84def3e8f6dd035bf2be8181
	// from the trusted nine-output structured invocation. This is glossary
	// validation only, not import or acceptance of the now-stale Bundle.
	source := `"...when a programming language is designed for exactly the environment most⟦P0⟧of us use right now—scalable, cloud-based servers that are optimized for⟦P1⟧performance—a lot can go right."`
	target := `“……当一门编程语言正是为我们大多数人⟦P0⟧如今使用的环境而设计——可扩展、基于云且针对⟦P1⟧性能优化的服务器——很多事情都会顺利得多。”`
	if err := ValidateParsedGlossary(source, target, &Glossary{Keep: []string{"Go"}}); err != nil {
		t.Fatal(err)
	}
}

func TestParsedGlossaryKeepCanonicalSpellingGenericTerm(t *testing.T) {
	g := &Glossary{Keep: []string{"Rust"}}
	if err := ValidateParsedGlossary("Use Rust.", "Rust言語を使う。", g); err != nil {
		t.Fatal(err)
	}
	if err := ValidateParsedGlossary("Remove rust.", "锈迹を落とす。", g); err != nil {
		t.Fatal(err)
	}
	if err := ValidateParsedGlossary("Use Rust.", "rust を使う。", g); err == nil {
		t.Fatal("canonical identity changed")
	}
}
