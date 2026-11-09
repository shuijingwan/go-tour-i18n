package sitecontent

import (
	"bytes"
	"fmt"
	"io/fs"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	yaml "github.com/shuijingwan/go-tour-i18n/internal/sitecontent/shellyaml"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
)

const repositoryModeImplementation = "internal/sourcestorage/manifest.go"

func representationFileMode(mode fs.FileMode) (uint32, error) {
	return sourcestorage.RepositoryFileMode(mode)
}

func sameRepresentationMode(a, b ShellRepresentationInput) bool {
	if a.Absent || b.Absent {
		return a.Absent == b.Absent && a.Mode == 0 && b.Mode == 0 && a.Reference.SHA256 == digest(nil) && b.Reference.SHA256 == digest(nil)
	}
	return sourcestorage.EquivalentRepositoryFileModes(fs.FileMode(a.Mode), fs.FileMode(b.Mode))
}

func sameRepresentationInputs(a, b []ShellRepresentationInput) bool {
	if len(a) != len(b) || (a == nil) != (b == nil) {
		return false
	}
	for i := range a {
		if a[i].Reference != b[i].Reference || !sameRepresentationMode(a[i], b[i]) {
			return false
		}
	}
	return true
}

func sameRepresentationChanges(a, b []ShellRepresentationDependencyChange) bool {
	if len(a) != len(b) || (a == nil) != (b == nil) {
		return false
	}
	for i := range a {
		if a[i].Old.Reference != b[i].Old.Reference || a[i].Current.Reference != b[i].Current.Reference || !sameRepresentationMode(a[i].Old, b[i].Old) || !sameRepresentationMode(a[i].Current, b[i].Current) {
			return false
		}
	}
	return true
}

// Compare only after the immutable actual receipt's original self-identity,
// physical SHA/path and lineage have been verified. These local copies do not
// normalize its stored bytes, identity, frozen base or old input inventory.
func sameShellReattestationCurrentness(want, actual ShellRepresentationReattestation) bool {
	if !sameRepresentationInputs(want.CurrentInputs, actual.CurrentInputs) || !sameRepresentationInputs(want.ValidationInputs, actual.ValidationInputs) || !sameRepresentationChanges(want.Changes, actual.Changes) {
		return false
	}
	want.CurrentInputs, want.ValidationInputs, want.Changes, want.Identity = actual.CurrentInputs, actual.ValidationInputs, actual.Changes, actual.Identity
	return reflect.DeepEqual(want, actual)
}

func sameSharedReattestationCurrentness(want, actual SharedRepresentationReattestation) bool {
	if !sameRepresentationInputs(want.CurrentInputs, actual.CurrentInputs) || !sameRepresentationInputs(want.ValidationInputs, actual.ValidationInputs) || !sameRepresentationChanges(want.Changes, actual.Changes) {
		return false
	}
	want.CurrentInputs, want.ValidationInputs, want.Changes, want.Identity = actual.CurrentInputs, actual.ValidationInputs, actual.Changes, actual.Identity
	return reflect.DeepEqual(want, actual)
}

// The existing extractor remains the sole slot/structure authority. Keeping
// this consumer separate preserves the frozen Learn/Docs adapter authority.
func shellDocument(s ShellLanguageSource) Document {
	if s.Page != nil {
		return *s.Page
	}
	d := Document{Contract: ShellExtractionContract, Path: s.Path, Kind: "data", SourceSHA: s.SourceSHA, StructureSHA: s.StructureSHA, Dependencies: []Dependency{}, Units: []Unit{}}
	for _, slot := range s.Slots {
		d.Units = append(d.Units, slot.Unit)
	}
	return d
}

func parseStructuredRepresentation(d Document, raw []byte) (*Document, error) {
	if d.Contract == UnitContract {
		return ParseDocument(documentSource(d), raw)
	}
	if d.Contract != ShellExtractionContract {
		return nil, fmt.Errorf("unknown structured parser %s", d.Contract)
	}
	s, err := ExtractShellLanguageSource(d.Path, raw)
	if err != nil {
		return nil, err
	}
	v := shellDocument(*s)
	return &v, nil
}

func reconstructStructuredRepresentation(d Document, raw []byte, targets map[string]string) ([]byte, error) {
	if d.Contract == UnitContract {
		return Reconstruct(documentSource(d), raw, targets)
	}
	if d.Contract != ShellExtractionContract {
		return nil, fmt.Errorf("unknown structured reconstruction contract")
	}
	s, err := ExtractShellLanguageSource(d.Path, raw)
	if err != nil {
		return nil, err
	}
	if s.Page != nil || !reflect.DeepEqual(shellDocument(*s), d) {
		return nil, fmt.Errorf("structured shell representation/source mismatch")
	}
	return reconstructShellSlots(*s, raw, targets)
}

func shellEncodedTarget(slot ShellLanguageSlot, raw []byte, target string) (string, error) {
	u := slot.Unit
	if target == u.Source {
		return string(raw[u.Start:u.End]), nil
	}
	if err := textBytes([]byte(target)); err != nil {
		return "", err
	}
	if err := validatePageSlotSyntax(u, target, raw); err != nil {
		return "", err
	}
	v, err := restoreBlock(u, target)
	if err != nil {
		return "", err
	}
	switch slot.Encoding {
	case "yaml-quoted":
		if raw[u.Start] == '\'' {
			return "'" + strings.ReplaceAll(v, "'", "''") + "'", nil
		}
		return strconv.Quote(v), nil
	case "yaml-literal":
		span := string(raw[u.Start:u.End])
		lines := strings.SplitAfter(span, "\n")
		if len(lines) < 2 {
			return "", fmt.Errorf("invalid literal scalar framing")
		}
		indent := lines[1][:len(lines[1])-len(strings.TrimLeft(lines[1], " "))]
		parts := strings.Split(v, "\n")
		var out strings.Builder
		out.WriteString(lines[0])
		for _, line := range parts {
			out.WriteString(indent)
			out.WriteString(line)
			out.WriteByte('\n')
		}
		return out.String(), nil
	case "yaml-plain":
		// Preserve the scalar's encoding and punctuation/flow boundaries.
		if strings.ContainsAny(v, "\n\r") || strings.Contains(v, ": ") || strings.Contains(v, " #") || strings.ContainsAny(v, "[]{},") {
			return "", fmt.Errorf("target changes plain YAML scalar framing")
		}
	case "go-string-interior":
		if strings.ContainsAny(v, "\"\\\n") {
			return "", fmt.Errorf("target changes Go literal framing")
		}
	case "html-text", "html-attribute", "go-comment-interior":
		if slot.Encoding == "html-attribute" && u.Start > 0 && strings.ContainsRune(v, rune(raw[u.Start-1])) {
			return "", fmt.Errorf("target changes HTML attribute quote")
		}
		if slot.Encoding == "go-comment-interior" && (strings.TrimSpace(v) == "" || strings.ContainsAny(v, "\n\r") || strings.Contains(v, "//") || strings.Contains(v, "/*") || strings.Contains(v, "*/") || strings.HasPrefix(strings.TrimSpace(v), "go:") || strings.HasPrefix(strings.TrimSpace(v), "line ")) {
			return "", fmt.Errorf("target changes Go comment framing")
		}
	default:
		return "", fmt.Errorf("unknown shell slot encoding %s", slot.Encoding)
	}
	return v, nil
}

func reconstructShellSlots(s ShellLanguageSource, raw []byte, targets map[string]string) ([]byte, error) {
	if digest(raw) != s.SourceSHA || len(targets) != len(s.Slots) {
		return nil, fmt.Errorf("shell target exact-set/source mismatch")
	}
	var out bytes.Buffer
	pos := 0
	for _, slot := range s.Slots {
		v, ok := targets[slot.Unit.ID]
		if !ok {
			return nil, fmt.Errorf("missing shell target slot")
		}
		encoded, err := shellEncodedTarget(slot, raw, v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", slot.Unit.ID, err)
		}
		out.Write(raw[pos:slot.Unit.Start])
		out.WriteString(encoded)
		pos = slot.Unit.End
	}
	out.Write(raw[pos:])
	if err := validateShellStructure(s, raw, out.Bytes()); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func validateShellStructure(source ShellLanguageSource, raw, target []byte) error {
	_, err := shellTargetValues(source, raw, target)
	return err
}

// Match only the frozen source gaps, then validate each captured slot against
// its source encoding and protected bindings. Never rediscover target language
// eligibility or classify target punctuation as source terminology.
func shellTargetValues(source ShellLanguageSource, raw, target []byte) ([]string, error) {
	if digest(raw) != source.SourceSHA {
		return nil, fmt.Errorf("shell frozen source mismatch")
	}
	if err := textBytes(target); err != nil {
		return nil, err
	}
	var pattern strings.Builder
	pattern.WriteString("(?s)\\A")
	pos := 0
	for _, slot := range source.Slots {
		u := slot.Unit
		if u.Start < pos || u.End < u.Start || u.End > len(raw) || (pos == u.Start && pos != 0) {
			return nil, fmt.Errorf("invalid or ambiguous frozen shell coordinates")
		}
		pattern.WriteString(regexp.QuoteMeta(string(raw[pos:u.Start])))
		pattern.WriteString("(.*?)")
		pos = u.End
	}
	pattern.WriteString(regexp.QuoteMeta(string(raw[pos:])))
	pattern.WriteString("\\z")
	matcher, err := regexp.Compile(pattern.String())
	if err != nil {
		return nil, err
	}
	matches := matcher.FindSubmatch(target)
	if len(matches) != len(source.Slots)+1 {
		return nil, fmt.Errorf("shell protected structure/identity changed")
	}
	values := make([]string, len(source.Slots))
	for i, slot := range source.Slots {
		encoded := string(matches[i+1])
		if encoded == string(raw[slot.Unit.Start:slot.Unit.End]) {
			values[i] = slot.Unit.Source
			continue
		}
		value := encoded
		if strings.HasPrefix(slot.Encoding, "yaml-") {
			n, err := shellYAMLNode([]byte("value: " + value + "\n"))
			if err != nil || n.Kind != yaml.MappingNode || len(n.Content) != 2 || n.Content[1].Kind != yaml.ScalarNode || n.Content[1].Tag != "!!str" {
				return nil, fmt.Errorf("shell target YAML scalar framing changed")
			}
			value = n.Content[1].Value
		}
		var bound string
		var err error
		if slot.Role == "inline-prose" {
			bound, err = bindInlineShellTokens(slot.Unit, value)
		} else {
			bound, err = bindFrozenShellTokens(slot.Unit, value)
		}
		if err != nil {
			return nil, fmt.Errorf("shell slot %d: %w", i, err)
		}
		reencoded, err := shellEncodedTarget(slot, raw, bound)
		if err != nil || reencoded != encoded {
			return nil, fmt.Errorf("shell slot %d framing/binding changed: %v", i, err)
		}
		values[i] = bound
	}
	return values, nil
}

// Rebind exact frozen raw identities without lexical rediscovery. Longer raw
// bindings own their bytes before contained atoms; duplicate raws retain the
// source occurrence order. Empty/ambiguous bindings fail closed.
func bindFrozenShellTokens(u Unit, value string) (string, error) {
	return bindFrozenShellTokensMode(u, value, nil)
}

// prebound is produced exclusively by the structural parser/binder, never
// supplied by a candidate. Remaining identities use the same frozen raw binder.
func bindFrozenShellTokensMode(u Unit, value string, prebound map[string]bool) (string, error) {
	remaining := value
	ordered := []ProtectedToken{}
	preboundRanges := [][2]int{}
	for _, p := range u.Protected {
		if prebound[p.Token] {
			if strings.Count(value, p.Token) != 1 {
				return "", fmt.Errorf("structural token exact-set changed")
			}
			remaining = strings.ReplaceAll(remaining, p.Token, "")
			at := strings.Index(value, p.Token)
			preboundRanges = append(preboundRanges, [2]int{at, at + len(p.Token)})
		} else {
			ordered = append(ordered, p)
		}
	}
	if strings.ContainsAny(remaining, "⟦⟧") {
		return "", fmt.Errorf("unexpected protected token in candidate")
	}
	sort.SliceStable(ordered, func(i, j int) bool { return len(ordered[i].Raw) > len(ordered[j].Raw) })
	if len(ordered) == 0 {
		_, err := restoreBlock(u, value)
		return value, err
	}
	byRaw := map[string][]ProtectedToken{}
	patterns := []string{}
	for _, p := range ordered {
		if p.Raw == "" {
			return "", fmt.Errorf("ambiguous empty shell binding")
		}
		if len(byRaw[p.Raw]) == 0 {
			patterns = append(patterns, regexp.QuoteMeta(p.Raw))
		}
		byRaw[p.Raw] = append(byRaw[p.Raw], p)
	}
	matcher, err := regexp.Compile(strings.Join(patterns, "|"))
	if err != nil {
		return "", err
	}
	var out strings.Builder
	pos := 0
	for _, match := range matcher.FindAllStringIndex(value, -1) {
		a, z := match[0], match[1]
		inside := false
		for _, r := range preboundRanges {
			if a < r[1] && z > r[0] {
				inside = true
				break
			}
		}
		if inside {
			continue
		}
		raw := value[a:z]
		bindings := byRaw[raw]
		if len(bindings) == 0 {
			return "", fmt.Errorf("duplicate protected raw identity: %s", raw)
		}
		p := bindings[0]
		byRaw[raw] = bindings[1:]
		// Natural CJK punctuation/adjacency is allowed; ASCII extensions of a
		// frozen machine word are changes to that identity.
		word := func(c byte) bool {
			return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
		}
		if p.Role == "atom" && word(p.Raw[0]) && word(p.Raw[len(p.Raw)-1]) && (a > 0 && word(value[a-1]) || z < len(value) && word(value[z])) {
			return "", fmt.Errorf("protected machine word changed: %s", p.Raw)
		}
		out.WriteString(value[pos:a])
		out.WriteString(p.Token)
		pos = z
	}
	for raw, remaining := range byRaw {
		if len(remaining) != 0 {
			return "", fmt.Errorf("protected raw identity changed: %s", raw)
		}
	}
	out.WriteString(value[pos:])
	bound := out.String()
	if _, err := restoreBlock(u, bound); err != nil {
		return "", err
	}
	return bound, nil
}

// ShellPageAdapter implements the existing shared TU adapter. It owns no QC,
// attempts, receipts or language decisions; one physical source is one Unit.
type ShellPageAdapter struct {
	Source ShellLanguageSource
	Raw    []byte
}

func (a ShellPageAdapter) ProtectedInput(g *i18n.Glossary) (string, []string, error) {
	return (PageAdapter{*a.Source.Page, a.Raw}).ProtectedInput(g)
}
func (a ShellPageAdapter) Restore(output string) ([]byte, error) {
	if a.Source.Page == nil {
		return nil, fmt.Errorf("Homepage formal Page required")
	}
	u := a.Source.Page.Units[0]
	output = strings.TrimSuffix(output, "\n")
	if _, err := restoreBlock(u, output); err != nil {
		return nil, err
	}
	boundaries := []ProtectedToken{}
	for _, p := range u.Protected {
		if p.Role == "boundary" {
			boundaries = append(boundaries, p)
		}
	}
	if len(boundaries) != len(a.Source.Slots)+1 || !strings.HasPrefix(output, boundaries[0].Token) || !strings.HasSuffix(output, boundaries[len(boundaries)-1].Token) {
		return nil, fmt.Errorf("Homepage boundary exact-set required")
	}
	values := map[string]string{}
	for i, slot := range a.Source.Slots {
		start := strings.Index(output, boundaries[i].Token) + len(boundaries[i].Token)
		end := strings.Index(output, boundaries[i+1].Token)
		if start > end {
			return nil, fmt.Errorf("Homepage slot order changed")
		}
		v := output[start:end]
		offset := 0
		for j, p := range u.Protected {
			if p.Token == boundaries[i].Token {
				offset = j + 1
				break
			}
		}
		pairs := []string{}
		for j, p := range slot.Unit.Protected {
			q := u.Protected[offset+j]
			if q.Raw != p.Raw || strings.Count(v, q.Token) != 1 {
				return nil, fmt.Errorf("Homepage token crossed slot boundary")
			}
			pairs = append(pairs, q.Token, p.Token)
		}
		if len(pairs) > 0 {
			v = strings.NewReplacer(pairs...).Replace(v)
		}
		values[slot.Unit.ID] = v
	}
	return reconstructShellSlots(a.Source, a.Raw, values)
}
func (a ShellPageAdapter) Validate(candidate []byte, g *i18n.Glossary) error {
	if err := validateShellStructure(a.Source, a.Raw, candidate); err != nil {
		return err
	}
	s, t, err := a.GlossaryVisiblePair(candidate)
	if err != nil {
		return err
	}
	return i18n.ValidateParsedGlossary(s, t, g)
}
func (a ShellPageAdapter) GlossaryVisiblePair(candidate []byte) (string, string, error) {
	values, err := shellTargetValues(a.Source, a.Raw, candidate)
	if err != nil {
		return "", "", err
	}
	visible := func(targets []string) string {
		lines := []string{}
		for i, slot := range a.Source.Slots {
			v := targets[i]
			for _, p := range slot.Unit.Protected {
				v = strings.ReplaceAll(v, p.Token, "")
			}
			lines = append(lines, v)
		}
		return strings.Join(lines, "\n")
	}
	sources := make([]string, len(a.Source.Slots))
	for i, slot := range a.Source.Slots {
		sources[i] = slot.Unit.Source
	}
	return visible(sources), visible(values), nil
}
func (a ShellPageAdapter) ReattestationAuthority(root string) (string, error) {
	return i18n.TranslationAdapterAuthority(root, "internal/sitecontent/shell_reconstruction.go", "ShellPageAdapter", []string{"internal/sitecontent/language_audit_view.go", "internal/sitecontent/language_audit_formal.go", "internal/sitecontent/language_audit_tour.go", "internal/sitecontent/comment_inventory.go", "internal/sitecontent/render_input.go", "internal/sitecontent/units.go", "internal/sitecontent/shell_language_source.go", "internal/sitecontent/shell_inline.go", "internal/sitecontent/page_replacement.go", "internal/sitecontent/shell_page_replacement.go", "internal/sitecontent/shellyaml/parse.go", "internal/sitecontent/formal_page.go", "internal/sitecontent/linguistic.go", "internal/sitecontent/semantic_role.go", "internal/sitecontent/semantic_go_packages.go", "internal/sitecontent/page_structure.go", "internal/sitecontent/structural_binding.go", "internal/sitecontent/markdown_references.go"})
}
