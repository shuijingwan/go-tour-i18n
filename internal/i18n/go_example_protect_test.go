package i18n

import (
	"bytes"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestGoExampleProtectionRoundTripAndNaturalComments(t *testing.T) {
	glossary := &Glossary{Keep: []string{"Go", "gofmt", "goroutine", "goroutines"}, Preferred: map[string]string{"channel": "通道"}}
	source := []byte("package main\n\n// A goroutine sends values to a channel in Go.\nfunc main() {}\n")
	protected, err := prepareGoExampleTranslationInput(source, sum(source), glossary)
	if err != nil {
		t.Fatal(err)
	}
	for _, visible := range []string{"A ", " sends values to a channel in ", "."} {
		if !strings.Contains(protected.Text, visible) {
			t.Errorf("protected input does not retain natural text %q:\n%s", visible, protected.Text)
		}
	}
	for _, hidden := range []string{"package main", "func main", "goroutine", "Go"} {
		if strings.Contains(protected.Text, hidden) {
			t.Errorf("protected input exposes protected text %q:\n%s", hidden, protected.Text)
		}
	}
	if !strings.Contains(protected.Text, "channel") {
		t.Fatalf("preferred term channel was incorrectly protected:\n%s", protected.Text)
	}
	restored, failures := protected.restore(protected.Text)
	if len(failures) != 0 || !bytes.Equal([]byte(restored), source) {
		t.Fatalf("round trip=%q failures=%v", restored, failures)
	}
	translated := strings.Replace(protected.Text, "A ", "一个 ", 1)
	translated = strings.Replace(translated, " sends values to a channel in ", " 向 channel 发送值，并使用 ", 1)
	restored, failures = protected.restore(translated)
	if len(failures) != 0 {
		t.Fatal(failures)
	}
	if !strings.Contains(restored, "// 一个 goroutine 向 channel 发送值，并使用 Go.") || !strings.Contains(restored, "func main() {}") {
		t.Fatalf("translated restore=%q", restored)
	}
}

func TestGoExampleProtectionKeepsGlossaryTermsInsideNaturalComments(t *testing.T) {
	glossary := &Glossary{Keep: []string{"Go", "gofmt", "goroutine", "goroutines"}, Preferred: map[string]string{"channel": "通道"}}
	source := []byte("package main\n\n// A goroutine and other goroutines use gofmt in Go with a channel.\nfunc main() {}\n")
	protected, err := prepareGoExampleTranslationInput(source, sum(source), glossary)
	if err != nil {
		t.Fatal(err)
	}
	for _, keep := range []string{"goroutine", "goroutines", "gofmt", "Go"} {
		if strings.Contains(protected.Text, keep) {
			t.Errorf("glossary.keep %q remains exposed:\n%s", keep, protected.Text)
		}
	}
	if !strings.Contains(protected.Text, "channel") {
		t.Fatalf("preferred channel was protected:\n%s", protected.Text)
	}
	restored, failures := protected.restore(protected.Text)
	if len(failures) != 0 || restored != string(source) {
		t.Fatalf("restore=%q failures=%v", restored, failures)
	}
}

func TestGoExampleProtectionTreatsShiftVerbAsTranslatable(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := catalog.Unit("example:basics/numeric-constants.go")
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	protected, err := prepareTranslationUnitInput(unit, glossary)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(protected.Text, "Shift it right again 99 places, so we end up with") {
		t.Fatalf("Shift verb was protected instead of exposed:\n%s", protected.Text)
	}
	if containsString(protected.Values, "Shift") {
		t.Fatalf("Shift verb became an independent keep token: %q", protected.Values)
	}

	keyboard := []byte("package main\n\n// Press the Shift key to continue.\nfunc main() {}\n")
	keyboardProtected, err := prepareGoExampleTranslationInput(keyboard, sum(keyboard), glossary)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(keyboardProtected.Values, "Shift") || strings.Contains(keyboardProtected.Text, "Shift key") {
		t.Fatalf("keyboard Shift was not protected: text=%q values=%q", keyboardProtected.Text, keyboardProtected.Values)
	}
}

func TestGoExampleProtectionBlockAndMultipleCommentsStayOneInput(t *testing.T) {
	source := []byte("package main\n\n/* This is a longer explanation. */\nfunc a() {}\n\n// Second explanation related to the first one.\nfunc b() {}\n")
	unit := &TranslationUnit{ID: "example:demo.go", Kind: UnitKindExample, Source: source, SourceSHA256: sum(source)}
	protected, err := prepareTranslationUnitInput(unit, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"This is a longer explanation.", "Second explanation related to the first one."} {
		if !strings.Contains(protected.Text, text) {
			t.Fatalf("one protected input does not contain %q:\n%s", text, protected.Text)
		}
	}
	translated := strings.Replace(protected.Text, "This is a longer explanation.", "这是一段更长的说明。", 1)
	translated = strings.Replace(translated, "Second explanation related to the first one.", "第二段说明与第一段相关。", 1)
	restored, failures := protected.restore(translated)
	if len(failures) != 0 {
		t.Fatal(failures)
	}
	if !strings.Contains(restored, "/* 这是一段更长的说明。 */") || !strings.Contains(restored, "// 第二段说明与第一段相关。") {
		t.Fatalf("restored comments=%q", restored)
	}
	if !strings.Contains(restored, "func a() {}") || !strings.Contains(restored, "func b() {}") {
		t.Fatalf("code changed=%q", restored)
	}
}

func TestGoExampleProtectionIgnoresCommentLikeStrings(t *testing.T) {
	source := []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"// hello\")\n\tfmt.Println(\"/* hello */\")\n\tfmt.Println(`// raw /* string */`)\n}\n")
	comments, err := scanGoExampleComments(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 0 {
		t.Fatalf("string contents classified as comments: %+v", comments)
	}
	protected, err := prepareGoExampleTranslationInput(source, sum(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(protected.Text, "hello") || strings.Contains(protected.Text, "raw") {
		t.Fatalf("string content was exposed:\n%s", protected.Text)
	}
	restored, failures := protected.restore(protected.Text)
	if len(failures) != 0 || restored != string(source) {
		t.Fatalf("restore=%q failures=%v", restored, failures)
	}
}

func TestGoExampleSpecialCommentsAreProtected(t *testing.T) {
	tests := []struct {
		name, comment string
		kind          goExampleCommentKind
	}{
		{"go directive", "//go:noinline", goExampleCommentGoDirective},
		{"go build", "//go:build OMIT", goExampleCommentGoDirective},
		{"line directive", "//line generated.go:10", goExampleCommentLineDirective},
		{"legacy build", "// +build linux", goExampleCommentLegacyBuild},
		{"present start", "// START OMIT", goExampleCommentPresentMarker},
		{"present end", "// END OMIT", goExampleCommentPresentMarker},
		{"test output", "// Output: hello", goExampleCommentTestMarker},
		{"unordered output", "// Unordered output: hello", goExampleCommentTestMarker},
		{"generated", "// Code generated by tool. DO NOT EDIT.", goExampleCommentGeneratedMarker},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(test.comment + "\npackage main\n")
			comments, err := scanGoExampleComments(source)
			if err != nil {
				t.Fatal(err)
			}
			if len(comments) != 1 || comments[0].Kind != test.kind {
				t.Fatalf("comments=%+v, want kind %s", comments, test.kind)
			}
			protected, err := prepareGoExampleTranslationInput(source, sum(source), nil)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(protected.Text, test.comment) {
				t.Fatalf("special comment exposed:\n%s", protected.Text)
			}
			restored, failures := protected.restore(protected.Text)
			if len(failures) != 0 || restored != string(source) {
				t.Fatalf("restore=%q failures=%v", restored, failures)
			}
		})
	}
}

func TestGoExampleProtectionRejectsTokenDamageAndReordering(t *testing.T) {
	source := []byte("package main\n\n// First explanation is here.\nfunc a() {}\n\n// Second explanation is here.\nfunc b() {}\n")
	protected, err := prepareGoExampleTranslationInput(source, sum(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(protected.Tokens) < 3 {
		t.Fatalf("tokens=%v", protected.Tokens)
	}
	mutations := map[string]string{
		"deleted":    strings.Replace(protected.Text, protected.Tokens[0], "", 1),
		"duplicated": protected.Text + protected.Tokens[0],
		"modified":   strings.Replace(protected.Text, protected.Tokens[0], "⟪GTI18N_deadbeef_999999⟫", 1),
		"reordered":  strings.Replace(strings.Replace(protected.Text, protected.Tokens[0], "TEMP", 1), protected.Tokens[1], protected.Tokens[0], 1) + protected.Tokens[1],
	}
	for name, output := range mutations {
		t.Run(name, func(t *testing.T) {
			if restored, failures := protected.restore(output); restored != "" || len(failures) == 0 {
				t.Fatalf("damaged input restored=%q failures=%v", restored, failures)
			}
		})
	}
}

func TestPrepareTranslationUnitInputKeepsPageBehavior(t *testing.T) {
	glossary := &Glossary{Keep: []string{"Go", "goroutine"}}
	source := []byte("* Page\n\nA goroutine is a lightweight thread managed by the Go runtime.\n")
	unit := &TranslationUnit{ID: "page/1", Kind: UnitKindPage, Source: source, SourceSHA256: sum(source)}
	want := prepareDefaultTranslationInput(source, unit.SourceSHA256, glossary)
	got, err := prepareTranslationUnitInput(unit, glossary)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("page protection changed\nwant=%+v\ngot=%+v", want, got)
	}
}

func TestAllCatalogExamplesProtectAndRestore(t *testing.T) {
	root := repoRoot(t)
	catalog, err := BuildCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	withNatural, withoutNatural, specialOnly := 0, 0, 0
	specialKinds := map[goExampleCommentKind]bool{}
	keepFiles := map[string]bool{}
	keepItems := map[string]bool{}
	var conservativelyProtected []string
	for _, example := range catalog.Examples {
		comments, err := scanGoExampleComments(example.Source)
		if err != nil {
			t.Fatalf("%s: %v", example.ID, err)
		}
		natural := false
		onlySpecial := len(comments) > 0
		for _, comment := range comments {
			switch comment.Kind {
			case goExampleCommentNatural:
				natural = true
				onlySpecial = false
				payload := string(example.Source[comment.PayloadStart:comment.PayloadEnd])
				for _, span := range goExampleKeepProtectionSpans(payload, glossary, nil) {
					keepFiles[example.ID] = true
					keepItems[payload[span.start:span.end]] = true
				}
			case goExampleCommentNonNatural:
				onlySpecial = false
				conservativelyProtected = append(conservativelyProtected, example.ID+":"+string(example.Source[comment.PayloadStart:comment.PayloadEnd]))
			default:
				specialKinds[comment.Kind] = true
			}
		}
		if natural {
			withNatural++
		} else {
			withoutNatural++
		}
		hasNatural, err := hasTranslatableGoExampleComment(example.Source)
		if err != nil || hasNatural != natural {
			t.Fatalf("%s translatable helper=%t, want %t: %v", example.ID, hasNatural, natural, err)
		}
		if onlySpecial {
			specialOnly++
		}
		protected, err := prepareGoExampleTranslationInput(example.Source, example.SourceSHA256, glossary)
		if err != nil {
			t.Fatalf("%s protect: %v", example.ID, err)
		}
		restored, failures := protected.restore(protected.Text)
		if len(failures) != 0 || !bytes.Equal([]byte(restored), example.Source) {
			t.Fatalf("%s round trip failed: %v", example.ID, failures)
		}
	}
	if withNatural+withoutNatural != 93 {
		t.Fatalf("classified examples=%d, want 93", withNatural+withoutNatural)
	}
	if withNatural != 19 || withoutNatural != 74 || specialOnly != 71 {
		t.Fatalf("corpus classification natural=%d without-natural=%d special-only=%d, want 19/74/71", withNatural, withoutNatural, specialOnly)
	}
	kinds := make([]string, 0, len(specialKinds))
	for kind := range specialKinds {
		kinds = append(kinds, string(kind))
	}
	sort.Strings(kinds)
	items := make([]string, 0, len(keepItems))
	for item := range keepItems {
		items = append(items, item)
	}
	sort.Strings(items)
	if !reflect.DeepEqual(items, []string{"goroutine"}) || len(keepFiles) != 1 {
		t.Fatalf("corpus glossary.keep files=%d items=%v, want 1 and [goroutine]", len(keepFiles), items)
	}
	sort.Strings(conservativelyProtected)
	t.Logf("examples: natural=%d without-natural=%d special-only=%d glossary-keep-files=%d keep-items=%v special-kinds=%v conservatively-protected=%v", withNatural, withoutNatural, specialOnly, len(keepFiles), items, kinds, conservativelyProtected)
}
