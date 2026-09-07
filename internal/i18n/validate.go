package i18n

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/present"
)

type signature struct {
	Directives     []string
	Links          []linkStructure
	LinkTargets    []string
	LinkInlineCode [][]string
	InlineCode     []string
	Preformatted   []string
}

// linkStructure is the protected identity of one complete present link. A
// target with program-font payload can move with its complete link, but cannot
// be rebound to another link's protected payload.
type linkStructure struct {
	Target     string
	InlineCode []string
}

var (
	directiveRE  = regexp.MustCompile(`^\.([A-Za-z][A-Za-z0-9_-]*)(?:\s|$)`)
	linkRE       = regexp.MustCompile(`\[\[([^\]]+)\]\[([^\]]*)\]\]`)
	machineURLRE = regexp.MustCompile("(?i)\\b(?:[a-z][a-z0-9+.-]*://|www\\.)[^\\s<>\\[\\]`]+|\\b[a-z0-9](?:[a-z0-9-]{0,62}\\.)+[a-z]{2,}(?:/[^\\s<>\\[\\]`]*)?")
)

func ValidateCandidate(root string, catalog *Catalog, pageID string, candidate []byte) error {
	return ValidateCandidateForLocale(root, catalog, pageID, "zh-CN", candidate)
}

// ValidateCandidateForLocale applies the same present and structure validation
// using the glossary associated with the requested locale.
func ValidateCandidateForLocale(root string, catalog *Catalog, pageID, locale string, candidate []byte) error {
	if err := ValidateLocaleName(locale); err != nil {
		return err
	}
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
	glossary, err := LoadGlossary(root, locale)
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
	if err := compareLinkStructures(expected.Links, actual.Links); err != nil {
		return diagnostic(pageID, err)
	}
	if err := compareUnorderedProtected("inline code", expected.InlineCode, actual.InlineCode); err != nil {
		return diagnostic(pageID, err)
	}
	if err := comparePreformattedForLocale(string(page.Source), string(candidate), locale); err != nil {
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
		if containsForbiddenVisibleText(string(source), string(candidate), forbidden) {
			return fmt.Errorf("%s: candidate contains forbidden locale translation %q", pageID, forbidden)
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

// containsForbiddenVisibleText searches only candidate bytes that are open to
// translation. URLs, link targets, program-font spans, directives, and static
// preformatted content are structural or machine identities; a glossary
// forbidden spelling inside them is not a target-language translation. Safe Go
// teaching-comment bodies remain visible because they are translatable prose.
func containsForbiddenVisibleText(source, candidate, forbidden string) bool {
	if forbidden == "" {
		return false
	}
	visible := visibleCandidateTextBytes(source, candidate)
	for offset := 0; offset < len(candidate); {
		relative := strings.Index(candidate[offset:], forbidden)
		if relative < 0 {
			return false
		}
		start := offset + relative
		end := start + len(forbidden)
		if allVisible(visible, start, end) && forbiddenTermBoundaries(candidate, start, end) {
			return true
		}
		offset = start + 1
	}
	return false
}

func visibleCandidateTextBytes(source, candidate string) []bool {
	visible := make([]bool, len(candidate))
	for i := range visible {
		visible[i] = true
	}
	mask := func(start, end int, value bool) {
		if start < 0 {
			start = 0
		}
		if end > len(visible) {
			end = len(visible)
		}
		for i := start; i < end; i++ {
			visible[i] = value
		}
	}
	for _, match := range linkRE.FindAllStringSubmatchIndex(candidate, -1) {
		mask(match[2], match[3], false)
	}
	for _, match := range machineURLRE.FindAllStringIndex(candidate, -1) {
		mask(match[0], match[1], false)
	}
	for _, match := range directiveLineRE.FindAllStringIndex(candidate, -1) {
		mask(match[0], match[1], false)
	}
	for _, code := range append(presentInlineCodes(candidate), linkLabelInlineCodes(candidate)...) {
		mask(code.Start, code.End, false)
	}

	sourceBlocks := preformattedBlocks(source)
	candidateBlocks := preformattedBlocks(candidate)
	for _, block := range candidateBlocks {
		mask(block.Start, block.End, false)
	}
	if len(sourceBlocks) != len(candidateBlocks) {
		return visible
	}
	for i := range sourceBlocks {
		sourceAnalysis := analyzePreformattedGo(sourceBlocks[i].Text)
		candidateAnalysis := analyzePreformattedGo(candidateBlocks[i].Text)
		if sourceAnalysis.Static || candidateAnalysis.Static || len(sourceAnalysis.Comments) != len(candidateAnalysis.Comments) {
			continue
		}
		for j, sourceComment := range sourceAnalysis.Comments {
			if !sourceComment.Translatable {
				continue
			}
			candidateComment := candidateAnalysis.Comments[j]
			mask(candidateBlocks[i].Start+candidateComment.BodyStart, candidateBlocks[i].Start+candidateComment.BodyEnd, true)
		}
	}
	return visible
}

func allVisible(visible []bool, start, end int) bool {
	if start < 0 || end > len(visible) || start >= end {
		return false
	}
	for _, value := range visible[start:end] {
		if !value {
			return false
		}
	}
	return true
}

func forbiddenTermBoundaries(text string, start, end int) bool {
	return forbiddenBoundaryBefore(text, start) && forbiddenBoundaryAfter(text, end)
}

func forbiddenBoundaryBefore(text string, start int) bool {
	if start == 0 {
		return true
	}
	r, width := utf8.DecodeLastRuneInString(text[:start])
	if forbiddenWordRune(r) {
		return false
	}
	if isWordApostrophe(r) && start > width {
		previous, _ := utf8.DecodeLastRuneInString(text[:start-width])
		return !forbiddenWordRune(previous)
	}
	return true
}

func forbiddenBoundaryAfter(text string, end int) bool {
	if end == len(text) {
		return true
	}
	r, width := utf8.DecodeRuneInString(text[end:])
	if forbiddenWordRune(r) {
		return false
	}
	if isWordApostrophe(r) && end+width < len(text) {
		next, _ := utf8.DecodeRuneInString(text[end+width:])
		return !forbiddenWordRune(next)
	}
	return true
}

func forbiddenWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func isWordApostrophe(r rune) bool {
	return r == '\'' || r == '’'
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
			var codes []string
			for _, code := range presentInlineCodes(match[2]) {
				codes = append(codes, code.Raw)
			}
			sig.Links = append(sig.Links, linkStructure{Target: match[1], InlineCode: codes})
			sig.LinkTargets = append(sig.LinkTargets, match[1])
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

// compareLinkStructures allows a complete link with protected inline payload
// to move for target-language word order, while keeping its target and payload
// attached. Plain labels have no stable cross-language payload identity, so
// their targets continue to be compared in occurrence order.
func compareLinkStructures(expected, actual []linkStructure) error {
	if len(expected) != len(actual) {
		return protectedCountError("link", len(expected), len(actual))
	}
	expectedTargets := make([]string, len(expected))
	actualTargets := make([]string, len(actual))
	expectedStructured := make([]string, len(expected))
	actualStructured := make([]string, len(actual))
	var expectedPlain, actualPlain []string
	for i, link := range expected {
		expectedTargets[i] = link.Target
		expectedStructured[i] = linkStructureIdentity(link)
		if len(link.InlineCode) == 0 {
			expectedPlain = append(expectedPlain, link.Target)
		}
	}
	for i, link := range actual {
		actualTargets[i] = link.Target
		actualStructured[i] = linkStructureIdentity(link)
		if len(link.InlineCode) == 0 {
			actualPlain = append(actualPlain, link.Target)
		}
	}
	if err := compareUnorderedProtected("link targets", expectedTargets, actualTargets); err != nil {
		return err
	}
	if err := compareUnorderedProtected("link inline code", expectedStructured, actualStructured); err != nil {
		return err
	}
	if err := compareProtected("plain link targets", expectedPlain, actualPlain); err != nil {
		return err
	}
	return nil
}

func linkStructureIdentity(link linkStructure) string {
	codes := append([]string(nil), link.InlineCode...)
	sort.Strings(codes)
	return link.Target + "\x00" + strings.Join(codes, "\x00")
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
