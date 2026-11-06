package languageaudit

import (
	"reflect"
	"strings"
	"testing"
)

func testPolicy(locale, scriptName string) Policy {
	return Policy{Locale: locale, Profile: "explicit-test-policy", ProseScripts: []string{scriptName}, Spacing: []SpacingRule{{scriptName, "Latin", []Kind{Code, Link, Identity}}}, SentenceEnd: map[string]string{".": "。"}, SentenceTerminators: ".!?。！？؟", ClosingQuotes: "\"'”’»」』)", ForbiddenClausePunctuation: []string{" -- "}, HeadingSymbols: "&", ResidualMinWords: 1}
}
func doc(pieces ...Piece) Document {
	return Document{Owner: "asset", Blocks: []Block{{Locator: "paragraph/1", Context: "paragraph", Start: -1, End: -1, Pieces: pieces}}}
}
func count(r Result, class string, d Decision) int {
	n := 0
	for _, x := range r.Diagnostics {
		if x.Class == class && x.Decision == d {
			n++
		}
	}
	return n
}
func TestAuditUnicodePolicyAdjacency(t *testing.T) {
	for _, tc := range []struct{ locale, script, text string }{{"zh-CN", "Han", "文字"}, {"ru-RU", "Cyrillic", "текст"}, {"ar", "Arabic", "كلمة"}, {"fa-IR", "Arabic", "واژه"}, {"ko-KR", "Hangul", "문자"}, {"hi-IN", "Devanagari", "शब्द"}} {
		t.Run(tc.locale, func(t *testing.T) {
			p := testPolicy(tc.locale, tc.script)
			for _, k := range []Kind{Code, Link, Identity} {
				r, err := Scan(p, Document{}, doc(Piece{tc.text, Prose, ""}, Piece{"pkg42", k, "fixed"}, Piece{tc.text, Prose, ""}))
				if err != nil || count(r, "script-adjacency", Defect) != 2 {
					t.Fatal(k, r, err)
				}
			}
			r, err := Scan(p, Document{}, doc(Piece{tc.text + "\u00a0", Prose, ""}, Piece{"pkg", Code, ""}, Piece{" " + tc.text, Prose, ""}))
			if err != nil || len(r.Diagnostics) != 0 {
				t.Fatal("Unicode spacing", r, err)
			}
		})
	}
	// Different scripts alone are not a defect, nor is a Java/CJK profile an
	// implicit spacing contract. Combining marks and bidi controls stay intact.
	p := testPolicy("ja-JP", "Han")
	p.Spacing = nil
	r, err := Scan(p, Document{}, doc(Piece{"日本", Prose, ""}, Piece{"Go", Link, ""}, Piece{"語", Prose, ""}))
	if err != nil || len(r.Diagnostics) != 0 {
		t.Fatal(r, err)
	}
	p = testPolicy("ar", "Arabic")
	r, err = Scan(p, Document{}, doc(Piece{"نص\u200f", Prose, ""}, Piece{"Go", Code, ""}))
	if err != nil || len(r.Diagnostics) != 0 {
		t.Fatal("do not reinterpret bidi controls as lexical adjacency", r, err)
	}
}
func TestAuditPunctuationIsLocaleAndRepresentationScoped(t *testing.T) {
	p := testPolicy("zh-CN", "Han")
	r, err := Scan(p, Document{}, doc(Piece{"示例 ", Prose, ""}, Piece{"tool.dev/x", Link, "url"}, Piece{".”", Prose, ""}))
	if err != nil || count(r, "sentence-punctuation", Defect) != 1 {
		t.Fatal(r, err)
	}
	for _, s := range []string{"version v1.2 正常。", "地址 tool.dev 正常。", "示例..."} {
		r, err = Scan(p, Document{}, doc(Piece{s, Prose, ""}))
		if err != nil || count(r, "sentence-punctuation", Defect) != 0 {
			t.Fatal(s, r, err)
		}
	}
	r, err = Scan(p, Document{}, doc(Piece{"文字 ", Prose, ""}, Piece{"cmd -- flag.", Code, ""}, Piece{"结束。", Prose, ""}))
	if err != nil || count(r, "clause-punctuation", Defect) != 0 {
		t.Fatal(r, err)
	}
	for _, locale := range []string{"fr-FR", "de-DE"} {
		p = testPolicy(locale, "Latin")
		p.SentenceEnd = nil
		p.ForbiddenClausePunctuation = nil
		p.Spacing = nil
		r, err = Scan(p, Document{}, doc(Piece{"Documentation.", Prose, ""}))
		if err != nil || len(r.Diagnostics) != 0 {
			t.Fatal(locale, r, err)
		}
	}
	p = testPolicy("fa-IR", "Arabic")
	p.SentenceEnd = map[string]string{"?": "؟"}
	r, err = Scan(p, Document{}, doc(Piece{"پرسش?", Prose, ""}))
	if err != nil || count(r, "sentence-punctuation", Defect) != 1 {
		t.Fatal(r, err)
	}
}
func TestAuditResidualsAreCandidatesNotJudgments(t *testing.T) {
	p := testPolicy("zh-CN", "Han")
	p.ProtectedIdentities = []string{"Go", "GOPATH", "Kubernetes"}
	source := doc(Piece{"Open the Command Prompt window. The issue is open. The Go Programming Language", Prose, ""})
	target := doc(Piece{"请打开 Command Prompt 窗口，查看 issue。", Prose, ""}, Piece{"GOPATH", Code, ""}, Piece{" https://tool.dev/path#12 ", Prose, ""}, Piece{"Kubernetes", Prose, ""})
	first, err := Scan(p, source, target)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range first.Diagnostics {
		if d.Decision == Defect {
			t.Fatal("semantic residual classified as deterministic", d)
		}
		if strings.Contains(d.Evidence, "GOPATH") || strings.Contains(d.Evidence, "tool.dev") || d.Evidence == "Kubernetes" {
			t.Fatal("protected identity residual", d)
		}
	}
	if count(first, "source-prose-overlap", Candidate) != 2 {
		t.Fatal(first)
	}
	second, _ := Scan(p, source, target)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("non-deterministic scan")
	}
	// Same-script locales use explicit source phrase overlap, not an assumption
	// that a Latin span is English. Shared cognates remain review-only too.
	p = testPolicy("fr-FR", "Latin")
	p.Spacing = nil
	p.ResidualMinWords = 2
	r, err := Scan(p, source, doc(Piece{"Command Prompt", ProtectedWord, "frozen-atom"}))
	if err != nil || count(r, "protected-language-candidate", Candidate) != 1 {
		t.Fatal(r, err)
	}
}
func TestAuditStructureAndProtectedSymbols(t *testing.T) {
	p := testPolicy("zh-CN", "Han")
	source := doc(Piece{"See ", Prose, ""}, Piece{"the guide", Link, "/guide"}, Piece{".", Prose, ""})
	target := Document{Owner: "asset", Blocks: []Block{{Locator: "p1", Context: "paragraph", Pieces: []Piece{{Text: "请参阅", Kind: Prose}}}, {Locator: "p2", Context: "paragraph", Pieces: []Piece{{Text: "指南", Kind: Link, Binding: "/guide"}, {Text: "。", Kind: Prose}}}, {Locator: "h1", Context: "heading", Pieces: []Piece{{Text: "支持 ", Kind: Prose}, {Text: "&", Kind: ProtectedSymbol}, {Text: " 成本", Kind: Prose}}}}}
	r, err := Scan(p, source, target)
	if err != nil || count(r, "paragraph-link-split", Candidate) != 1 || count(r, "protected-prose-symbol", Candidate) != 1 {
		t.Fatal(r, err)
	}
	r, err = Scan(p, Document{}, doc(Piece{".", Prose, ""}))
	if err != nil || count(r, "punctuation-only-paragraph", Candidate) != 1 {
		t.Fatal(r, err)
	}
	p.PunctuationOnlyParagraphIsDefect = true
	r, err = Scan(p, Document{}, doc(Piece{".", Prose, ""}))
	if err != nil || count(r, "punctuation-only-paragraph", Defect) != 1 {
		t.Fatal(r, err)
	}
	p.ProseScripts = []string{"NotAUnicodeScript"}
	if _, err := Scan(p, source, target); err == nil {
		t.Fatal("unknown script policy accepted")
	}
}

func TestAuditOtherScriptPairsAndInlineResiduals(t *testing.T) {
	p := testPolicy("ar", "Arabic")
	p.Spacing = []SpacingRule{{"Arabic", "Cyrillic", []Kind{Code}}}
	r, err := Scan(p, Document{}, doc(Piece{"كلمة", Prose, ""}, Piece{"пакет", Code, ""}))
	if err != nil || count(r, "script-adjacency", Defect) != 1 {
		t.Fatal(r, err)
	}
	p = testPolicy("zh-CN", "Han")
	r, err = Scan(p, doc(Piece{"the issue remains open", Prose, ""}), doc(Piece{"该issue仍开放。", Prose, ""}))
	if err != nil || count(r, "source-prose-overlap", Candidate) != 1 || count(r, "script-adjacency", Defect) != 0 {
		t.Fatal("plain foreign prose is not a technical atom", r, err)
	}
	r, err = Scan(p, Document{}, doc(Piece{"Use ", Prose, ""}, Piece{"汉字", Code, ""}, Piece{".", Prose, ""}))
	if err != nil || count(r, "sentence-punctuation", Defect) != 0 {
		t.Fatal("code script is not prose context", r, err)
	}
	p = testPolicy("ru-RU", "Cyrillic")
	p.SentenceEnd = nil
	r, err = Scan(p, doc(Piece{"пример.рф", Identity, "verified-domain"}), doc(Piece{"Адрес ", Prose, ""}, Piece{"пример.рф", Identity, "verified-domain"}))
	if err != nil || count(r, "source-prose-overlap", Candidate) != 0 {
		t.Fatal("Unicode domain body", r, err)
	}
	p = testPolicy("fr-FR", "Latin")
	p.SentenceEnd = nil
	p.Spacing = nil
	r, err = Scan(p, doc(Piece{"and/or e.g.", Prose, ""}), doc(Piece{"and/or e.g.", Prose, ""}))
	if err != nil || count(r, "source-prose-overlap", Candidate) == 0 {
		t.Fatal("ordinary abbreviation suppressed as technical identity", r, err)
	}
	p.Spacing = []SpacingRule{{"Latin", "Cyrillic", []Kind{Prose}}}
	if _, err := Scan(p, Document{}, Document{}); err == nil {
		t.Fatal("blanket cross-script rule accepted")
	}
}

func TestAuditResidualOpaqueAndLiteralBarriers(t *testing.T) {
	p := testPolicy("fr-FR", "Latin")
	p.Spacing, p.SentenceEnd = nil, nil
	p.ResidualMinWords = 2
	for _, kind := range []Kind{Opaque, Code, Identity} {
		// Even a literal without letters must break phrase matching.
		barrier := doc(Piece{"Alpha", Prose, ""}, Piece{"123", kind, "proof"}, Piece{"Beta", Prose, ""})
		plain := doc(Piece{"Alpha Beta", Prose, ""})
		for _, pair := range [][2]Document{{barrier, plain}, {plain, barrier}} {
			r, err := Scan(p, pair[0], pair[1])
			if err != nil || count(r, "source-prose-overlap", Candidate) != 0 {
				t.Fatal("residual crossed opaque/literal barrier", kind, r, err)
			}
		}
	}
	// Dotted words and underscores alone are not identity authority.
	plain := doc(Piece{"a.b x_y", Prose, ""})
	r, err := Scan(p, plain, plain)
	if err != nil || count(r, "source-prose-overlap", Candidate) == 0 {
		t.Fatal("unproved spelling silently suppressed", r, err)
	}
}

func TestAuditLiteralParagraphAndWholeIdentity(t *testing.T) {
	p := testPolicy("zh-CN", "Han")
	p.PunctuationOnlyParagraphIsDefect = true
	r, err := Scan(p, Document{}, doc(Piece{".", Code, "literal"}))
	if err != nil || len(r.Diagnostics) != 0 {
		t.Fatal("literal punctuation treated as paragraph defect", r, err)
	}
	p.ProtectedIdentities = []string{"Go"}
	for _, text := range []string{"Go2", "Go_thing", "2Go", "Going"} {
		parts := normalized([]Piece{{Text: text, Kind: Prose}}, p)
		if len(parts) != 1 || parts[0].Kind != Prose || parts[0].Text != text {
			t.Fatal("identity prefix/suffix split", text, parts)
		}
	}
}

func TestAuditLongResidualThresholdIsNotSilentlyTruncated(t *testing.T) {
	p := testPolicy("de-DE", "Latin")
	p.Spacing, p.SentenceEnd = nil, nil
	p.ResidualMinWords = 130
	d := doc(Piece{Text: strings.Repeat("shared ", 130), Kind: Prose})
	r, err := Scan(p, d, d)
	if err != nil || count(r, "source-prose-overlap", Candidate) != 1 {
		t.Fatal(r, err)
	}
}

func TestAuditSourceBoundIdentityMetadataIsLocaleIndependent(t *testing.T) {
	for _, tc := range []struct{ script, prose string }{{"Han", "文字"}, {"Cyrillic", "слово"}, {"Arabic", "كلمة"}} {
		in := []Piece{{Text: "cache" + tc.prose, Kind: Prose}}
		pieces, err := BindIdentities(in, []string{"cache"})
		if err != nil || len(pieces) != 2 || pieces[0].Kind != Identity || in[0].Text != "cache"+tc.prose {
			t.Fatal(pieces, err)
		}
		p := testPolicy("test", tc.script)
		r, err := Scan(p, Document{}, doc(pieces...))
		if err != nil || count(r, "script-adjacency", Defect) != 1 {
			t.Fatal(r, err)
		}
	}
	if _, err := BindIdentities(nil, []string{""}); err == nil {
		t.Fatal("invalid identity authority")
	}
}
