package i18n

import (
	"strings"
	"testing"
)

func TestPreformattedTeachingCommentClassification(t *testing.T) {
	translatable := []string{
		"alias for uint8",
		"represents a Unicode code point",
		"j is an int",
		"read i through the pointer p",
		"Compile error!",
		"here v has type T",
		"no match; here v has the same type as i",
		"Send v to channel ch.",
		"use i",
		"receiving from c would block",
	}
	for _, comment := range translatable {
		if !isTranslatableTeachingComment(comment) {
			t.Errorf("comment %q was not classified as translatable", comment)
		}
	}

	static := []string{"int", "float64", "complex128", "OK", "len(a)=5", "len(b)=0, cap(b)=5", ""}
	for _, comment := range static {
		if isTranslatableTeachingComment(comment) {
			t.Errorf("comment %q was classified as translatable", comment)
		}
	}
}

func TestPreformattedTourTeachingCommentLanguageIsExposedAndStaticAnnotationsAreProtected(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	exposed := map[string][]string{
		"basics/11":     {"alias for ", "represents a Unicode code point"},
		"basics/14":     {" is an "},
		"moretypes/1":   {"read ", " through the pointer "},
		"methods/6":     {"Compile error!"},
		"methods/16":    {"here ", " has type ", "no match; "},
		"concurrency/2": {"Send ", " to channel ", "assign value to "},
		"concurrency/6": {"use ", "receiving from ", " would block"},
	}
	for pageID, comments := range exposed {
		page, err := catalog.Page(pageID)
		if err != nil {
			t.Fatal(err)
		}
		protected := protectTranslation(page.Source, page.SourceSHA256, nil)
		for _, comment := range comments {
			if !strings.Contains(protected.Text, comment) {
				t.Errorf("%s teaching comment %q is protected:\n%s", pageID, comment, protected.Text)
			}
		}
	}

	static := map[string][]string{
		"basics/14":    {"// int", "// float64", "// complex128"},
		"moretypes/13": {"// len(a)=5", "// len(b)=0, cap(b)=5"},
		"methods/6":    {"// OK"},
	}
	for pageID, comments := range static {
		page, err := catalog.Page(pageID)
		if err != nil {
			t.Fatal(err)
		}
		protected := protectTranslation(page.Source, page.SourceSHA256, nil)
		for _, comment := range comments {
			if strings.Contains(protected.Text, comment) {
				t.Errorf("%s static comment %q remains exposed:\n%s", pageID, comment, protected.Text)
			}
		}
	}
}

func TestPreformattedCommentIdentifierExtraction(t *testing.T) {
	tests := []struct {
		name  string
		block string
		want  [][]string
	}{
		{"type switch", "\tswitch v := i.(type) {\n\tcase T:\n\t\t// here v has type T\n\tcase S:\n\t\t// here v has type S\n\tdefault:\n\t\t// no match; here v has the same type as i\n\t}\n\n", [][]string{{"v", "T"}, {"v", "S"}, {"v", "i"}}},
		{"inference", "\tvar i int\n\tj := i // j is an int\n\n", [][]string{{"j", "int"}}},
		{"basic type alias", "\tbyte // alias for uint8\n\tuint8\n\n", [][]string{{"uint8"}}},
		{"pointer", "\ti := 42\n\tp := &i\n\tfmt.Println(*p) // read i through the pointer p\n\n", [][]string{{"i", "p"}}},
		{"channel", "\tch <- v // Send v to channel ch.\n\n", [][]string{{"v", "ch"}}},
		{"select", "\tselect {\n\tcase i := <-c:\n\t\t// use i\n\tdefault:\n\t\t// receiving from c would block\n\t}\n\n", [][]string{{"i"}, {"c"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := analyzePreformattedGo(tt.block)
			if analysis.Static {
				t.Fatalf("safe Go block classified static: %+v", analysis)
			}
			if len(analysis.Comments) != len(tt.want) {
				t.Fatalf("comments = %d, want %d", len(analysis.Comments), len(tt.want))
			}
			for i, comment := range analysis.Comments {
				var got []string
				for _, identifier := range comment.Identifiers {
					got = append(got, identifier.Value)
				}
				if strings.Join(got, ",") != strings.Join(tt.want[i], ",") {
					t.Errorf("comment %d identifiers = %v, want %v", i+1, got, tt.want[i])
				}
			}
		})
	}
}

func TestPreformattedScannerExcludesCommentLikeLiterals(t *testing.T) {
	block := "\tfmt.Println(\"https://example.test/a//b\")\n\tfmt.Println(`// raw`)\n\tr := '/'\n\n"
	analysis := analyzePreformattedGo(block)
	if analysis.Static || len(analysis.Comments) != 0 {
		t.Fatalf("literal-like comments were scanned: %+v", analysis)
	}
	p := protectTranslation([]byte("* Literals\n\n"+block+"Text.\n"), "12345678", nil)
	if strings.Contains(p.Text, "https://example.test") || strings.Contains(p.Text, "// raw") {
		t.Fatalf("comment-free block was not protected as a whole:\n%s", p.Text)
	}
	restored, failures := p.restore(p.Text)
	if len(failures) != 0 || restored != "* Literals\n\n"+block+"Text.\n" {
		t.Fatalf("restore = %q, failures=%v", restored, failures)
	}
}

func TestUnrecognizedPreformattedBlockIsStatic(t *testing.T) {
	block := "\techo hello // explain output to the user\n\n"
	analysis := analyzePreformattedGo(block)
	if !analysis.Static {
		t.Fatalf("non-Go block was treated as safely translatable: %+v", analysis)
	}
	p := protectTranslation([]byte("* Shell\n\n"+block+"Text.\n"), "12345678", nil)
	if strings.Contains(p.Text, "explain output") {
		t.Fatalf("non-Go comment remains exposed:\n%s", p.Text)
	}
}

func TestPreformattedStaticBlocksAreExactlyProtected(t *testing.T) {
	tests := []string{
		"\ti := 42\n\n",
		"\ti := 42 // int\n\n",
		"\ti := 42 // OK\n\n",
		"\ta := make([]int, 5) // len(a)=5\n\n",
		"\tb := make([]int, 0, 5) // len(b)=0, cap(b)=5\n\n",
		"\ti := 42 /* explanatory block comment */\n\n",
	}
	for _, block := range tests {
		source := "* Static\n\n" + block + "Text.\n"
		p := protectTranslation([]byte(source), "12345678", nil)
		if strings.Contains(p.Text, strings.TrimSpace(block)) {
			t.Errorf("static block remains exposed:\n%s", p.Text)
		}
		restored, failures := p.restore(p.Text)
		if len(failures) != 0 || restored != source {
			t.Errorf("restore = %q, failures=%v; want %q", restored, failures, source)
		}
	}
}

func TestStaticPreformattedTokenLeavesFollowingSourceSeparatorVisible(t *testing.T) {
	source := "* Static\n\n\tvalue := 1\n\nProse.\n"
	p := protectTranslation([]byte(source), "12345678", nil)
	static := protectedTokensOfKind(p, protectedPreformattedStatic)
	if len(static) != 1 {
		t.Fatalf("static tokens = %v", static)
	}
	if want := "\n\n" + static[0] + "\n\nProse."; !strings.Contains(p.Text, want) {
		t.Fatalf("protected input missing source block boundaries %q:\n%s", want, p.Text)
	}
	if strings.Contains(p.Text, static[0]+"Prose.") {
		t.Fatalf("static token absorbed following separator:\n%s", p.Text)
	}
	restored, failures := p.restore(p.Text)
	if len(failures) != 0 || restored != source {
		t.Fatalf("restore = %q, failures=%v; want %q", restored, failures, source)
	}
}

func TestStaticPreformattedPayloadPreservesEOFAndDirectiveBoundaries(t *testing.T) {
	tests := []struct {
		name, source, suffix string
	}{
		{"EOF", "* EOF\n\n\tvalue := 1\n", ""},
		{"directive", "* Directive\n\n\tvalue := 1\n\n.play example/value.go\n", "\n\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := protectTranslation([]byte(tt.source), "12345678", nil)
			static := protectedTokensOfKind(p, protectedPreformattedStatic)
			if len(static) != 1 {
				t.Fatalf("static tokens = %v", static)
			}
			if tt.suffix == "" {
				if !strings.HasSuffix(p.Text, static[0]) {
					t.Fatalf("EOF static token has unexpected suffix:\n%s", p.Text)
				}
			} else if !strings.Contains(p.Text, static[0]+tt.suffix) {
				t.Fatalf("static token lost directive separator %q:\n%s", tt.suffix, p.Text)
			}
			restored, failures := p.restore(p.Text)
			if len(failures) != 0 || restored != tt.source {
				t.Fatalf("restore = %q, failures=%v; want %q", restored, failures, tt.source)
			}
		})
	}
}

func TestTourStaticPreformattedTokensRetainVisibleBoundaries(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		pageID string
		wants  []string
	}{
		{"flowcontrol/10", []string{"\n\n⟪GTI18N_e1479cae_000001⟫\n\ndoes not call"}},
		{"moretypes/1", []string{"⟪GTI18N_7706eaf0_000008⟫\n\nThe ", "⟪GTI18N_7706eaf0_000011⟫\n\nThe "}},
	} {
		t.Run(tt.pageID, func(t *testing.T) {
			page, err := catalog.Page(tt.pageID)
			if err != nil {
				t.Fatal(err)
			}
			p := protectTranslation(page.Source, page.SourceSHA256, &Glossary{Keep: glossary.Keep})
			for _, want := range tt.wants {
				if !strings.Contains(p.Text, want) {
					t.Errorf("protected input missing %q:\n%s", want, p.Text)
				}
			}
			restored, failures := p.restore(p.Text)
			if len(failures) != 0 || restored != string(page.Source) {
				t.Fatalf("restore failed: %v", failures)
			}
		})
	}
}

func TestPreformattedStaticCommentsAndBlockCommentsCannotChange(t *testing.T) {
	tests := []struct {
		source, candidate string
	}{
		{"\ti := 42 // int\n\n", "\ti := 42 // integer\n\n"},
		{"\ti := 42 // OK\n\n", "\ti := 42 // 好\n\n"},
		{"\ta := make([]int, 5) // len(a)=5\n\n", "\ta := make([]int, 5) // 长度为 5\n\n"},
		{"\ti := 42 /* note */\n\n", "\ti := 42 /* 注释 */\n\n"},
	}
	for _, tt := range tests {
		if err := comparePreformattedBlock(tt.source, tt.candidate); err == nil {
			t.Errorf("static comment mutation accepted:\n%s", tt.candidate)
		}
	}
}

func TestPreformattedMethods16TranslatedTeachingCommentsPassValidator(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("methods/16")
	if err != nil {
		t.Fatal(err)
	}
	candidate := translatedMethods16(string(page.Source))
	if err := ValidateCandidate(root, catalog, "methods/16", []byte(candidate)); err != nil {
		t.Fatalf("translated teaching comments rejected: %v\n%s", err, candidate)
	}

	p := protectTranslation(page.Source, page.SourceSHA256, nil)
	for _, code := range []string{"switch v := i.(type)", "case T:", "case S:", "default:"} {
		if strings.Contains(p.Text, code) {
			t.Errorf("non-comment code %q remains exposed:\n%s", code, p.Text)
		}
	}
	for _, body := range []string{"here v has type T", "here v has type S", "no match; here v has the same type as i"} {
		if strings.Contains(p.Text, body) {
			t.Errorf("teaching comment identifier remains exposed in %q:\n%s", body, p.Text)
		}
	}
	wantIdentifierValues := map[string]int{"v": 3, "i": 1, "T": 1, "S": 1}
	for value, want := range wantIdentifierValues {
		got := 0
		for i, protectedValue := range p.Values {
			if protectedValue == value && p.Kinds[i] == protectedPreformattedIdentifier {
				got++
			}
		}
		if got != want {
			t.Errorf("protected comment identifier %q count = %d, want %d", value, got, want)
		}
	}
	model := strings.Replace(p.Text, "here ", "此时 ", 3)
	model = strings.Replace(model, " has type ", " 的类型为 ", 2)
	model = strings.Replace(model, "no match; ", "没有匹配项；", 1)
	model = strings.Replace(model, " has the same type as ", " 的类型与 ", 1)
	for i, value := range p.Values {
		if value == "i" && p.Kinds[i] == protectedPreformattedIdentifier {
			model = strings.Replace(model, p.Tokens[i], p.Tokens[i]+" 相同", 1)
		}
	}
	restored, failures := p.restore(model)
	if len(failures) != 0 || restored != candidate {
		t.Fatalf("restore failures=%v\ngot:\n%s\nwant:\n%s", failures, restored, candidate)
	}
}

func TestMethods16CommentIdentifierMutationsFailCandidateValidation(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("methods/16")
	if err != nil {
		t.Fatal(err)
	}
	valid := translatedMethods16(string(page.Source))
	if strings.Contains(valid, "has type") || strings.Contains(valid, "same type") {
		t.Fatal("ordinary English word type was not translated in valid fixture")
	}
	swapped := strings.Replace(valid, "// 此时 v 的类型为 T", "// 此时 v 的类型为 TEMP_S", 1)
	swapped = strings.Replace(swapped, "// 此时 v 的类型为 S", "// 此时 v 的类型为 T", 1)
	swapped = strings.Replace(swapped, "TEMP_S", "S", 1)
	tests := map[string]string{
		"v replaced":  strings.Replace(valid, "// 此时 v 的类型为 T", "// 此时 value 的类型为 T", 1),
		"T case":      strings.Replace(valid, "// 此时 v 的类型为 T", "// 此时 v 的类型为 t", 1),
		"i missing":   strings.Replace(valid, "// 没有匹配项；此时 v 的类型与 i 相同", "// 没有匹配项；此时 v 的类型相同", 1),
		"v repeated":  strings.Replace(valid, "// 此时 v 的类型为 T", "// 此时 v 和 v 的类型为 T", 1),
		"T/S swapped": swapped,
	}
	for name, candidate := range tests {
		t.Run(name, func(t *testing.T) {
			if candidate == valid {
				t.Fatal("test mutation did not change candidate")
			}
			if err := ValidateCandidate(root, catalog, "methods/16", []byte(candidate)); err == nil {
				t.Fatalf("candidate with changed comment identifiers accepted:\n%s", candidate)
			}
		})
	}
}

func TestPreformattedMethods16NonCommentMutationsFail(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("methods/16")
	if err != nil {
		t.Fatal(err)
	}
	valid := translatedMethods16(string(page.Source))
	tests := map[string]string{
		"v":                strings.Replace(valid, "switch v :=", "switch value :=", 1),
		"i":                strings.Replace(valid, "i.(type)", "x.(type)", 1),
		"T":                strings.Replace(valid, "case T:", "case U:", 1),
		"S":                strings.Replace(valid, "case S:", "case R:", 1),
		"switch":           strings.Replace(valid, "switch v", "select v", 1),
		"case":             strings.Replace(valid, "case T:", "caseT:", 1),
		"default":          strings.Replace(valid, "default:", "fallback:", 1),
		"type":             strings.Replace(valid, "i.(type)", "i.(T)", 1),
		"assignment":       strings.Replace(valid, ":=", "=", 1),
		"parenthesis":      strings.Replace(valid, "i.(type)", "i.type", 1),
		"brace":            strings.Replace(valid, " {", "", 1),
		"colon":            strings.Replace(valid, "case T:", "case T", 1),
		"indentation":      strings.Replace(valid, "\t\t// 此时", "\t// 此时", 1),
		"newline":          strings.Replace(valid, "case T:\n", "case T: ", 1),
		"missing-comment":  strings.Replace(valid, "\t\t// 此时 v 的类型为 T\n", "", 1),
		"moved-comment":    strings.Replace(valid, "case T:\n\t\t// 此时 v 的类型为 T", "\t\t// 此时 v 的类型为 T\ncase T:", 1),
		"different-marker": strings.Replace(valid, "// 此时 v 的类型为 T", "/* 此时 v 的类型为 T */", 1),
		"empty-comment":    strings.Replace(valid, "此时 v 的类型为 T", "", 1),
	}
	for name, candidate := range tests {
		t.Run(name, func(t *testing.T) {
			if candidate == valid {
				t.Fatal("test mutation did not change candidate")
			}
			if err := ValidateCandidate(root, catalog, "methods/16", []byte(candidate)); err == nil {
				t.Fatalf("invalid mutation accepted:\n%s", candidate)
			}
		})
	}
}

func TestPreformattedConcurrency2MultilineTeachingCommentTranslation(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("concurrency/2")
	if err != nil {
		t.Fatal(err)
	}
	candidate := strings.NewReplacer(
		"Send v to channel ch.", "将 v 发送到信道 ch。",
		"Receive from ch, and", "从 ch 接收值，并",
		"assign value to v.", "将值赋给 v。",
	).Replace(string(page.Source))
	if err := ValidateCandidate(root, catalog, "concurrency/2", []byte(candidate)); err != nil {
		t.Fatalf("multiline teaching comment translation rejected: %v\n%s", err, candidate)
	}
	broken := strings.Replace(candidate, "\n\t           // 将值赋给 v。", " 将值赋给 v。", 1)
	if broken == candidate {
		t.Fatal("multiline mutation did not change candidate")
	}
	if err := ValidateCandidate(root, catalog, "concurrency/2", []byte(broken)); err == nil {
		t.Fatalf("collapsed teaching comment lines accepted:\n%s", broken)
	}
}

func TestFlowcontrol8PreformattedProtectionRoundTrip(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.Page("flowcontrol/8")
	if err != nil {
		t.Fatal(err)
	}
	p := protectTranslation(page.Source, page.SourceSHA256, nil)
	for _, block := range []string{"z -= (z*z - x) / (2*z)", "z := 1.0", "z := float64(1)"} {
		if strings.Contains(p.Text, block) {
			t.Errorf("comment-free block %q remains exposed:\n%s", block, p.Text)
		}
	}
	restored, failures := p.restore(p.Text)
	if len(failures) != 0 || restored != string(page.Source) {
		t.Fatalf("restore failures=%v\ngot:\n%s", failures, restored)
	}
	if err := ValidateCandidate(root, catalog, "flowcontrol/8", []byte(restored)); err != nil {
		t.Fatalf("restored source rejected: %v", err)
	}
}

func translatedMethods16(source string) string {
	return strings.NewReplacer(
		"// here v has type T", "// 此时 v 的类型为 T",
		"// here v has type S", "// 此时 v 的类型为 S",
		"// no match; here v has the same type as i", "// 没有匹配项；此时 v 的类型与 i 相同",
	).Replace(source)
}
