package i18n

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/present"
)

type signature struct {
	Directives     []string
	LinkTargets    []string
	LinkInlineCode [][]string
	InlineCode     []string
	Preformatted   []string
}

var (
	directiveRE = regexp.MustCompile(`^\.([A-Za-z][A-Za-z0-9_-]*)(?:\s|$)`)
	linkRE      = regexp.MustCompile(`\[\[([^\]]+)\]\[([^\]]*)\]\]`)
)

func ValidateCandidate(root string, catalog *Catalog, pageID string, candidate []byte) error {
	page, err := catalog.Page(pageID)
	if err != nil {
		return err
	}
	if bytes.Contains(page.Source, []byte("#appengine:")) {
		return fmt.Errorf("%s: standalone source contains #appengine content", pageID)
	}
	candidate = normalizeLF(candidate)
	if len(bytes.TrimSpace(candidate)) == 0 {
		return fmt.Errorf("%s: candidate is empty", pageID)
	}
	if bytes.Contains(candidate, []byte("#appengine:")) {
		return fmt.Errorf("%s: standalone candidate contains #appengine content", pageID)
	}
	glossary, err := LoadGlossary(root, "zh-CN")
	if err != nil {
		return err
	}
	if err := validateGlossary(pageID, page.Source, candidate, glossary); err != nil {
		return err
	}
	if err := parseSinglePage(root, page.Article, candidate); err != nil {
		return fmt.Errorf("%s: candidate present structure: %w", pageID, err)
	}
	expected, err := structuralSignature(page.Source)
	if err != nil {
		return fmt.Errorf("%s: source policy: %w", pageID, err)
	}
	actual, err := structuralSignature(candidate)
	if err != nil {
		return fmt.Errorf("%s: candidate policy: %w", pageID, err)
	}
	if err := compareProtected("present directives", expected.Directives, actual.Directives); err != nil {
		return diagnostic(pageID, err)
	}
	if err := compareProtected("link targets", expected.LinkTargets, actual.LinkTargets); err != nil {
		return diagnostic(pageID, err)
	}
	if err := compareLinkInlineCode(expected.LinkInlineCode, actual.LinkInlineCode); err != nil {
		return diagnostic(pageID, err)
	}
	if err := compareUnorderedProtected("inline code", expected.InlineCode, actual.InlineCode); err != nil {
		return diagnostic(pageID, err)
	}
	if err := comparePreformatted(string(page.Source), string(candidate)); err != nil {
		return diagnostic(pageID, err)
	}
	expectedFonts, err := parsedFontSpans(root, page.Article, page.Source)
	if err != nil {
		return fmt.Errorf("%s: source font structure: %w", pageID, err)
	}
	actualFonts, err := parsedFontSpans(root, page.Article, candidate)
	if err != nil {
		return fmt.Errorf("%s: candidate font structure: %w", pageID, err)
	}
	if err := compareFontSpans(expectedFonts, actualFonts); err != nil {
		return diagnostic(pageID, err)
	}
	if err := compareSectionStructure(root, page.Article, page.Source, candidate); err != nil {
		return diagnostic(pageID, err)
	}
	return nil
}

func validateGlossary(pageID string, source, candidate []byte, glossary *Glossary) error {
	for _, forbidden := range glossary.Forbidden {
		if bytes.Contains(candidate, []byte(forbidden)) {
			return fmt.Errorf("%s: candidate contains forbidden zh-CN translation %q", pageID, forbidden)
		}
	}
	sourceLinks := linkRE.FindAllSubmatch(source, -1)
	candidateLinks := linkRE.FindAllSubmatch(candidate, -1)
	if len(sourceLinks) == len(candidateLinks) {
		for i := range sourceLinks {
			sourceLabel := string(sourceLinks[i][2])
			key, wrapper := glossaryKeyForLabel(sourceLabel, glossary.Mandatory)
			if key == "" {
				continue
			}
			target := string(sourceLinks[i][1])
			if string(candidateLinks[i][1]) != target {
				continue // The structural target validator reports this mismatch.
			}
			want := wrapper + glossary.Mandatory[key] + wrapper
			if got := string(candidateLinks[i][2]); got != want {
				return fmt.Errorf("%s: glossary link label for target %q = %q, want %q", pageID, target, got, want)
			}
		}
	}
	if regexp.MustCompile(`\bslides\b`).Match(source) {
		if !bytes.Contains(candidate, []byte(glossary.Mandatory["slides"])) {
			return fmt.Errorf("%s: glossary requires slides to use %q", pageID, glossary.Mandatory["slides"])
		}
	}
	return nil
}

func glossaryKeyForLabel(label string, mandatory map[string]string) (key, wrapper string) {
	if _, ok := mandatory[label]; ok {
		return label, ""
	}
	if len(label) >= 2 && label[0] == '"' && label[len(label)-1] == '"' {
		inner := label[1 : len(label)-1]
		if _, ok := mandatory[inner]; ok {
			return inner, "\""
		}
	}
	return "", ""
}

func structuralSignature(source []byte) (signature, error) {
	var sig signature
	s := bufio.NewScanner(bytes.NewReader(source))
	for s.Scan() {
		line := s.Text()
		if match := directiveRE.FindStringSubmatch(line); match != nil {
			switch match[1] {
			case "play", "image":
			default:
				return sig, fmt.Errorf("unsupported directive type .%s", match[1])
			}
			sig.Directives = append(sig.Directives, strings.TrimSpace(line))
		}
		for _, match := range linkRE.FindAllStringSubmatch(line, -1) {
			sig.LinkTargets = append(sig.LinkTargets, match[1])
			var codes []string
			for _, code := range presentInlineCodes(match[2]) {
				codes = append(codes, code.Raw)
			}
			sig.LinkInlineCode = append(sig.LinkInlineCode, codes)
		}
		for _, code := range presentInlineCodes(line) {
			sig.InlineCode = append(sig.InlineCode, code.Raw)
		}
		if isPreformattedLine(line) {
			sig.Preformatted = append(sig.Preformatted, line)
		}
	}
	return sig, s.Err()
}

// compareLinkInlineCode keeps each program payload attached to the link where
// it appeared, while treating multiple program spans in one translated label
// as an unordered set. This preserves natural target-language label order
// without allowing a span to be changed, removed, or moved to another link.
func compareLinkInlineCode(expected, actual [][]string) error {
	if len(expected) != len(actual) {
		return protectedCountError("link label", len(expected), len(actual))
	}
	for i := range expected {
		if err := compareUnorderedProtected("link inline code at link index "+fmt.Sprint(i+1), expected[i], actual[i]); err != nil {
			return err
		}
	}
	return nil
}

func compareProtected(kind string, expected, actual []string) error {
	limit := len(expected)
	if len(actual) < limit {
		limit = len(actual)
	}
	for i := 0; i < limit; i++ {
		if expected[i] != actual[i] {
			return fmt.Errorf("%s mismatch at index %d: expected %q, actual %q", kind, i+1, shorten(expected[i]), shorten(actual[i]))
		}
	}
	if len(expected) != len(actual) {
		return fmt.Errorf("%s count mismatch: expected %d, actual %d; first difference index %d", kind, len(expected), len(actual), limit+1)
	}
	return nil
}

func compareUnorderedProtected(kind string, expected, actual []string) error {
	if len(expected) != len(actual) {
		return protectedCountError(kind, len(expected), len(actual))
	}
	want := append([]string(nil), expected...)
	got := append([]string(nil), actual...)
	sort.Strings(want)
	sort.Strings(got)
	for i := range want {
		if want[i] != got[i] {
			return fmt.Errorf("%s payload mismatch: expected %q, actual %q", kind, shorten(want[i]), shorten(got[i]))
		}
	}
	return nil
}

type sectionStructure struct {
	path            string
	preformatted    int
	directives      int
	directiveLayout []string
}

func compareSectionStructure(root, article string, source, candidate []byte) error {
	expected, err := sectionStructures(root, article, source)
	if err != nil {
		return err
	}
	actual, err := sectionStructures(root, article, candidate)
	if err != nil {
		return err
	}
	if len(expected) != len(actual) {
		return fmt.Errorf("section topology count mismatch: expected %d, actual %d", len(expected), len(actual))
	}
	for i := range expected {
		want, got := expected[i], actual[i]
		if want.path != got.path {
			return fmt.Errorf("section topology mismatch at index %d: expected path %s, actual %s", i+1, want.path, got.path)
		}
		if want.preformatted != got.preformatted {
			return fmt.Errorf("preformatted block section mismatch at %s: expected %d, actual %d", want.path, want.preformatted, got.preformatted)
		}
		if want.directives != got.directives {
			return fmt.Errorf("directive section mismatch at %s: expected %d, actual %d", want.path, want.directives, got.directives)
		}
		if !sameStrings(want.directiveLayout, got.directiveLayout) {
			return fmt.Errorf("directive placement mismatch at %s: expected %s, actual %s", want.path, strings.Join(want.directiveLayout, ", "), strings.Join(got.directiveLayout, ", "))
		}
	}
	return nil
}

func sectionStructures(root, article string, source []byte) ([]sectionStructure, error) {
	doc, err := parsePresentPage(root, article, source)
	if err != nil {
		return nil, err
	}
	var result []sectionStructure
	for i, section := range doc.Sections {
		collectSectionStructures(section, fmt.Sprintf("%d", i+1), &result)
	}
	return result, nil
}

func collectSectionStructures(section present.Section, path string, result *[]sectionStructure) {
	structure := sectionStructure{path: path}
	for _, elem := range section.Elem {
		switch value := elem.(type) {
		case present.Text:
			if value.Pre {
				structure.preformatted++
				structure.directiveLayout = append(structure.directiveLayout, "preformatted")
			} else {
				structure.directiveLayout = appendCollapsedProse(structure.directiveLayout)
			}
		case present.List:
			structure.directiveLayout = append(structure.directiveLayout, "list")
		case present.Section:
			structure.directiveLayout = append(structure.directiveLayout, "section")
		case present.Code, present.Image:
			structure.directives++
			structure.directiveLayout = append(structure.directiveLayout, "directive")
		}
	}
	if structure.directives == 0 {
		structure.directiveLayout = nil
	}
	*result = append(*result, structure)
	child := 0
	for _, elem := range section.Elem {
		if nested, ok := elem.(present.Section); ok {
			child++
			collectSectionStructures(nested, fmt.Sprintf("%s.%d", path, child), result)
		}
	}
}

// appendCollapsedProse records an ordinary-text region without preserving how
// many present.Text elements it contains. Translators may legitimately split or
// merge paragraphs, but a directive must remain between the same surrounding
// top-level structural elements.
func appendCollapsedProse(layout []string) []string {
	if len(layout) == 0 || layout[len(layout)-1] != "prose" {
		return append(layout, "prose")
	}
	return layout
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func protectedCountError(kind string, expected, actual int) error {
	return fmt.Errorf("%s count mismatch: expected %d, actual %d", kind, expected, actual)
}

func preformattedBlockError(index int, err error) error {
	return fmt.Errorf("preformatted code block mismatch at index %d: %v", index, err)
}

func preformattedCommentError(index int, message string) error {
	return fmt.Errorf("line comment mismatch at index %d: %s", index, message)
}

func preformattedComparisonError(message string) error {
	return fmt.Errorf("%s", message)
}

func diagnostic(pageID string, err error) error {
	return fmt.Errorf("%s: protected structure validation failed: %v; check the named directive or protected content near the first difference", pageID, err)
}

func shorten(s string) string {
	if len(s) <= 120 {
		return s
	}
	return s[:117] + "..."
}
