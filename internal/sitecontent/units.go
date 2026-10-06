package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const UnitContract = "go-learning/content-units/v2"

// Unit records a source-bound text slot. Offsets are reconstruction coordinates,
// not persistent identities. Protected bytes remain exclusively in the source.
type Unit struct {
	ID        string           `json:"id"`
	Locator   string           `json:"locator"`
	Source    string           `json:"source"`
	SourceSHA string           `json:"source_sha256"`
	Start     int              `json:"start"`
	End       int              `json:"end"`
	Protected []ProtectedToken `json:"protected"`
}
type Document struct {
	Contract     string       `json:"contract"`
	UnitScope    string       `json:"unit_scope,omitempty"`
	StableIndex  int          `json:"stable_index,omitempty"`
	Slots        []Unit       `json:"internal_slots,omitempty"`
	Path         string       `json:"path"`
	Kind         string       `json:"kind"`
	Route        string       `json:"route"`
	SourceSHA    string       `json:"source_sha256"`
	StructureSHA string       `json:"structure_sha256"`
	Dependencies []Dependency `json:"dependencies"`
	Units        []Unit       `json:"units"`
}
type span struct {
	start, end int
	context    string
}

func textBytes(b []byte) error {
	if len(b) == 0 || !utf8.Valid(b) || bytes.ContainsAny(b, "\x00\r") || bytes.HasPrefix(b, []byte{239, 187, 191}) {
		return fmt.Errorf("content requires UTF-8 and LF; source EOF is preserved")
	}
	return nil
}

// ParseDocument uses the Markdown AST and HTML tokenizer; unsupported metadata
// and data schemas fail closed. It never treats redirect/assets as language.
func ParseDocument(s Source, b []byte) (*Document, error) {
	if !validPath(s.Path) || (s.Kind != "page" && s.Kind != "data") || digest(b) != s.SHA256 {
		return nil, fmt.Errorf("invalid document identity/kind: %s", s.Path)
	}
	if err := textBytes(b); err != nil {
		return nil, err
	}
	if bytes.Contains(b, []byte("{{")) {
		_, err := template.New("frozen-source").Funcs(template.FuncMap{"breadcrumbs": func(any) string { return "" }, "first": func(int, any) any { return nil }, "data": func(string) any { return nil }, "raw": func(any) string { return "" }}).Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("unknown/invalid template directive: %w", err)
		}
	}
	spans := []span{}
	add := func(a, z int, c string) {
		if a < z {
			spans = append(spans, span{a, z, c})
		}
	}
	if s.Kind == "data" {
		if err := parseLearnData(b, add); err != nil {
			return nil, err
		}
	} else {
		offset, err := parseMetadata(b, add)
		if err != nil {
			return nil, err
		}
		body := b[offset:]
		if err := teachingSpans(body, offset, path.Ext(s.Path) == ".md", add); err != nil {
			return nil, err
		}
		switch path.Ext(s.Path) {
		case ".html":
			if err := htmlBlocks(body, offset, "html", add); err != nil {
				return nil, err
			}
		case ".md":
			if err := markdownBlocks(body, offset, add); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown page parser")
		}
	}
	if s.Kind == "data" {
		spans = mergeDataSpans(spans)
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	d := &Document{Contract: UnitContract, Path: s.Path, Kind: s.Kind, Route: s.Route, SourceSHA: s.SHA256, Units: []Unit{}, Dependencies: s.Dependencies}
	seen := map[string]int{}
	previous := 0
	var structure bytes.Buffer
	for _, sp := range spans {
		if sp.start < previous || sp.end > len(b) {
			return nil, fmt.Errorf("overlapping parser text spans")
		}
		a, z := sp.start, sp.end
		for a < z && unicode.IsSpace(rune(b[a])) {
			a++
		}
		for z > a && unicode.IsSpace(rune(b[z-1])) {
			z--
		}
		if a == z {
			continue
		}
		source, locked, err := protectBlock(b[a:z], sp.context)
		if err != nil {
			return nil, fmt.Errorf("%s at %d: %w", s.Path, a, err)
		}
		if !hasLanguage(source) {
			continue
		}
		structure.Write(b[previous:a])
		structure.WriteString("\x00" + sp.context + "\x00" + protectionIdentity(locked))
		sh := unitSourceIdentity(source, locked)
		base := s.Path + "#" + sp.context + ":" + sh
		seen[base]++
		locator := base + ":" + strconv.Itoa(seen[base])
		id := digest([]byte(UnitContract + "\n" + locator))
		d.Units = append(d.Units, Unit{ID: id, Locator: locator, Source: source, SourceSHA: sh, Start: a, End: z, Protected: locked})
		previous = z
	}
	structure.Write(b[previous:])
	d.StructureSHA = identity(struct {
		Structure    string
		Dependencies []Dependency
	}{digest(structure.Bytes()), s.Dependencies})
	return d, nil
}

// HTML token boundaries preserve tags, every attribute, comments, directives,
// and entire code/pre/script/style regions (including multi-line snippets).
func htmlSpans(b []byte, offset int, context string, add func(int, int, string)) error {
	z := html.NewTokenizer(bytes.NewReader(b))
	pos := 0
	protected := 0
	for {
		typ := z.Next()
		raw := z.Raw()
		a := pos
		pos += len(raw)
		if typ == html.ErrorToken {
			if z.Err() != io.EOF {
				return z.Err()
			}
			break
		}
		switch typ {
		case html.StartTagToken, html.SelfClosingTagToken:
			tag, _ := z.TagName()
			if protected == 0 && !knownHTMLTag(string(tag)) {
				return fmt.Errorf("unknown HTML structure %s", tag)
			}
			if err := htmlAttributeSpans(raw, offset+a, context, add); err != nil {
				return err
			}
			switch string(tag) {
			case "pre", "code", "script", "style", "textarea", "var", "kbd", "samp", "tt":
				if typ == html.StartTagToken {
					protected++
				}
			}
		case html.EndTagToken:
			tag, _ := z.TagName()
			switch string(tag) {
			case "pre", "code", "script", "style", "textarea", "var", "kbd", "samp", "tt":
				if protected > 0 {
					protected--
				}
			}
		case html.TextToken:
			if protected == 0 {
				add(offset+a, offset+pos, context)
			}
		}
	}
	return nil
}

func knownHTMLTag(tag string) bool {
	for _, known := range strings.Fields("html head body title meta link main section article header footer aside nav div span p a ul ol li pre code h1 h2 h3 h4 h5 h6 hr br em strong b i table thead tbody tfoot tr th td img script style svg circle path rect g line polyline polygon text desc figure figcaption details summary sup sub blockquote dl dt dd iframe video source button input form label textarea select option center font var kbd samp tt abbr acronym address cite dfn del ins small big u s strike mark time wbr noscript embed object param picture audio canvas math col colgroup caption legend fieldset") {
		if tag == known {
			return true
		}
	}
	return false
}

func htmlAttributeSpans(raw []byte, offset int, context string, add func(int, int, string)) error {
	// Tokenizer already established this as a tag. Walk its raw attribute bytes
	// to retain quoting and all machine attributes without HTML reserialization.
	i := 1
	for i < len(raw) && raw[i] != ' ' && raw[i] != '\n' && raw[i] != '\t' && raw[i] != '>' {
		i++
	}
	for i < len(raw) {
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\n' || raw[i] == '\t' || raw[i] == '/') {
			i++
		}
		if i >= len(raw) || raw[i] == '>' {
			break
		}
		a := i
		for i < len(raw) && raw[i] != '=' && raw[i] != ' ' && raw[i] != '\n' && raw[i] != '\t' && raw[i] != '>' {
			i++
		}
		key := strings.ToLower(string(raw[a:i]))
		for i < len(raw) && unicode.IsSpace(rune(raw[i])) {
			i++
		}
		if i >= len(raw) || raw[i] != '=' {
			continue
		}
		i++
		for i < len(raw) && unicode.IsSpace(rune(raw[i])) {
			i++
		}
		if i >= len(raw) {
			return fmt.Errorf("incomplete HTML attribute")
		}
		quote := raw[i]
		if quote != '\'' && quote != '"' {
			for i < len(raw) && !unicode.IsSpace(rune(raw[i])) && raw[i] != '>' {
				i++
			}
			if key == "alt" || key == "title" || key == "aria-label" {
				return fmt.Errorf("visible HTML attributes require quotes")
			}
			continue
		}
		i++
		start := i
		for i < len(raw) && raw[i] != quote {
			i++
		}
		if i >= len(raw) {
			return fmt.Errorf("unterminated HTML attribute")
		}
		if key == "alt" || key == "title" || key == "aria-label" {
			add(offset+start, offset+i, context+"/attribute/"+key)
		}
		i++
	}
	return nil
}

func htmlProtectedRanges(b []byte) ([]span, error) {
	z := html.NewTokenizer(bytes.NewReader(b))
	pos := 0
	depth := 0
	start := 0
	result := []span{}
	for {
		typ := z.Next()
		a := pos
		pos += len(z.Raw())
		if typ == html.ErrorToken {
			if z.Err() != io.EOF {
				return nil, z.Err()
			}
			break
		}
		if typ != html.StartTagToken && typ != html.EndTagToken {
			continue
		}
		tag, _ := z.TagName()
		switch string(tag) {
		case "pre", "code", "script", "style", "textarea", "var", "kbd", "samp", "tt":
			if typ == html.StartTagToken {
				if depth == 0 {
					start = a
				}
				depth++
			} else if depth > 0 {
				depth--
				if depth == 0 {
					result = append(result, span{start: start, end: pos})
				}
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unclosed protected HTML")
	}
	return result, nil
}

func parseMetadata(b []byte, add func(int, int, string)) (int, error) {
	if bytes.HasPrefix(b, []byte("<!--{")) {
		end := bytes.Index(b, []byte("}-->"))
		if end < 0 {
			return 0, fmt.Errorf("unterminated metadata")
		}
		raw := b[4 : end+1]
		var m struct {
			Title      string
			Path       string
			Breadcrumb bool
			Template   bool
			HideTOC    bool
		}
		if err := StrictJSON(raw, &m); err != nil {
			return 0, err
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		_, _ = dec.Token()
		for dec.More() {
			key, err := dec.Token()
			if err != nil {
				return 0, err
			}
			a := int(dec.InputOffset())
			var value json.RawMessage
			if err := dec.Decode(&value); err != nil {
				return 0, err
			}
			z := int(dec.InputOffset())
			if strings.EqualFold(key.(string), "title") {
				for a < z && raw[a] != '"' {
					a++
				}
				if a == z || bytes.Contains(value, []byte(`\`)) {
					return 0, fmt.Errorf("unsupported metadata title encoding")
				}
				add(4+a+1, 4+z-1, "metadata/title")
			}
		}
		return end + 4, nil
	}
	if bytes.HasPrefix(b, []byte("---\n")) {
		end := bytes.Index(b[4:], []byte("\n---\n"))
		if end < 0 {
			return 0, fmt.Errorf("unterminated front matter")
		}
		pos := 4
		seen := map[string]bool{}
		for _, line := range bytes.Split(b[4:4+end], []byte("\n")) {
			k, v, ok := strings.Cut(string(line), ":")
			if !ok || seen[k] {
				return 0, fmt.Errorf("unknown/duplicate metadata")
			}
			seen[k] = true
			switch k {
			case "title", "breadcrumbTitle":
				a := pos + len(k) + 1
				for a < pos+len(line) && b[a] == ' ' {
					a++
				}
				z := pos + len(line)
				if a < z && b[a] == '"' {
					if b[z-1] != '"' {
						return 0, fmt.Errorf("invalid quoted metadata")
					}
					a++
					z--
				}
				add(a, z, "metadata/"+k)
			case "layout":
				if strings.TrimSpace(v) != "article" {
					return 0, fmt.Errorf("unknown layout")
				}
			case "breadcrumb", "template":
				if strings.TrimSpace(v) != "true" && strings.TrimSpace(v) != "false" {
					return 0, fmt.Errorf("invalid metadata bool")
				}
			default:
				return 0, fmt.Errorf("unknown metadata field %s", k)
			}
			pos += len(line) + 1
		}
		return 4 + end + 5, nil
	}
	return 0, nil
}

// Frozen Learn data is a deliberately narrow sequence-of-mappings schema.
// Continuation indentation is validated; keys, URLs and numeric fields never
// enter language slots. Unrecognized YAML syntax must be explicitly supported.
func parseLearnData(b []byte, add func(int, int, string)) error {
	visible := map[string]bool{"title": true, "content": true, "cta": true, "description": true, "eyebrow": true, "blurb": true}
	machine := map[string]bool{"url": true, "thumbnail": true, "thumbnailDark": true, "length": true, "credits": true, "rating": true}
	pos := 0
	item := ""
	key := ""
	seen := map[string]bool{}
	baseIndent := -1
	scalarOpen := false
	itemURLs := map[int]string{}
	itemStart := -1
	url := ""
	seenURLs := map[string]bool{}
	scanPos := 0
	finish := func() error {
		if itemStart < 0 {
			return nil
		}
		if url == "" || seenURLs[url] {
			return fmt.Errorf("data items require unique machine URL")
		}
		seenURLs[url] = true
		itemURLs[itemStart] = url
		return nil
	}
	for _, line := range bytes.Split(b, []byte("\n")) {
		trim := strings.TrimSpace(string(line))
		if strings.HasPrefix(trim, "- ") {
			if err := finish(); err != nil {
				return err
			}
			itemStart = scanPos
			url = ""
		}
		if strings.HasPrefix(trim, "url:") {
			url = strings.TrimSpace(strings.TrimPrefix(trim, "url:"))
		}
		scanPos += len(line) + 1
	}
	if err := finish(); err != nil {
		return err
	}
	for _, line := range bytes.Split(b, []byte("\n")) {
		raw := string(line)
		trim := strings.TrimSpace(raw)
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if trim == "" {
			pos += len(line) + 1
			continue
		}
		if strings.ContainsAny(raw, "\t") {
			return fmt.Errorf("unsupported YAML indentation")
		}
		if strings.HasPrefix(trim, "- ") {
			if !strings.HasPrefix(trim, "- title:") {
				return fmt.Errorf("data item must start with title")
			}
			seen = map[string]bool{}
			item = digest([]byte(itemURLs[pos]))
			baseIndent = indent + 2
			trim = strings.TrimPrefix(trim, "- ")
		} else if indent > baseIndent && key != "" {
			if !scalarOpen || strings.Contains(trim, ": ") || strings.HasSuffix(trim, ":") || strings.HasPrefix(trim, "- ") {
				return fmt.Errorf("unknown nested YAML structure")
			}
			if visible[key] {
				a := pos + indent
				if err := htmlScalar(b[a:pos+len(line)], a, "data/"+item+"/"+key, add); err != nil {
					return err
				}
			}
			pos += len(line) + 1
			continue
		} else if item == "" || indent != baseIndent {
			return fmt.Errorf("unknown YAML structure")
		}
		k, v, ok := strings.Cut(trim, ":")
		if !ok || seen[k] || (!visible[k] && !machine[k]) {
			return fmt.Errorf("unknown/duplicate data field %s", k)
		}
		seen[k] = true
		key = k
		scalarOpen = strings.TrimSpace(v) != ""
		if visible[k] {
			value := strings.TrimSpace(v)
			if strings.HasPrefix(value, "\"") || strings.HasPrefix(value, "'") || strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") || strings.Contains(value, ": ") || strings.Contains(value, " #") || value == "null" || value == "~" || value == "true" || value == "false" {
				return fmt.Errorf("unsupported/non-text YAML scalar")
			}
		}
		if strings.HasPrefix(strings.TrimSpace(v), "|") || strings.HasPrefix(strings.TrimSpace(v), ">") || strings.HasPrefix(strings.TrimSpace(v), "&") || strings.HasPrefix(strings.TrimSpace(v), "*") {
			return fmt.Errorf("unsupported YAML scalar")
		}
		if visible[k] && strings.TrimSpace(v) != "" {
			a := pos + strings.Index(raw, k+":") + len(k) + 1
			for a < pos+len(line) && b[a] == ' ' {
				a++
			}
			if err := htmlScalar(b[a:pos+len(line)], a, "data/"+item+"/"+k, add); err != nil {
				return err
			}
		}
		pos += len(line) + 1
	}
	return nil
}

// Reconstruct accepts an exact unit map, not arbitrary target directories.
// Parsing the result must reproduce exactly the protected structure.
func Reconstruct(s Source, source []byte, targets map[string]string) ([]byte, error) {
	d, err := ParseDocument(s, source)
	if err != nil {
		return nil, err
	}
	if len(targets) != len(d.Units) {
		return nil, fmt.Errorf("target exact-set mismatch")
	}
	var out bytes.Buffer
	pos := 0
	for _, u := range d.Units {
		v, ok := targets[u.ID]
		if !ok || v == "" || !utf8.ValidString(v) || strings.ContainsAny(v, "\x00\r") {
			return nil, fmt.Errorf("invalid/missing plain target %s", u.ID)
		}
		out.Write(source[pos:u.Start])
		restored, err := restoreBlock(u, v)
		if err != nil {
			return nil, err
		}
		out.WriteString(restored)
		pos = u.End
	}
	out.Write(source[pos:])
	b := out.Bytes()
	next := s
	next.SHA256 = digest(b)
	parsed, err := ParseDocument(next, b)
	if err != nil {
		return nil, err
	}
	if parsed.StructureSHA != d.StructureSHA || len(parsed.Units) != len(d.Units) {
		return nil, fmt.Errorf("protected structure/parser round-trip changed")
	}
	return b, nil
}

func PackageDocuments(root string, g *Global, id string) ([]Document, map[string][]byte, error) {
	if _, err := packageByID(g, id); err != nil {
		return nil, nil, err
	}
	raw := map[string][]byte{}
	docs := []Document{}
	if id == "site-v2-shell" {
		for _, s := range g.Sources {
			if s.Package != id {
				continue
			}
			b, err := readRegular(root, s.Path)
			if err != nil {
				return nil, nil, err
			}
			d, err := ParseDocument(s, b)
			if err != nil {
				return nil, nil, err
			}
			docs = append(docs, *d)
			raw[s.Path] = b
		}
		if len(docs) != 2 {
			return nil, nil, fmt.Errorf("shell requires exact homepage + translation sources")
		}
		return docs, raw, nil
	}
	if id != "learn-docs-v1" {
		return nil, nil, fmt.Errorf("Tour retains its existing TranslationUnit workflow")
	}
	snapshot, err := readRegular(root, SnapshotPath)
	if err != nil {
		return nil, nil, err
	}
	f, err := snapshotFS(snapshot)
	if err != nil {
		return nil, nil, err
	}
	for _, s := range g.Sources {
		if s.Package == id && (s.Kind == "page" || s.Kind == "data") {
			b, err := fs.ReadFile(f, s.Path)
			if err != nil {
				return nil, nil, err
			}
			d, err := ParseDocument(s, b)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", s.Path, err)
			}
			docs = append(docs, *d)
			raw[s.Path] = b
		}
	}
	if len(docs) != 56 {
		return nil, nil, fmt.Errorf("frozen localizable exact-set must be 49 page + 7 data")
	}
	return docs, raw, nil
}

func ShellSources() []Source {
	return []Source{{Path: "data/site-shell/home.html", Kind: "page", Package: "site-v2-shell", Surface: "homepage", Route: "/"}, {Path: "data/site-shell/translation.html", Kind: "page", Package: "site-v2-shell", Surface: "translation", Route: "/translation/"}}
}

func documentSource(d Document) Source {
	return Source{Path: d.Path, Kind: d.Kind, Route: d.Route, SHA256: d.SourceSHA, Dependencies: d.Dependencies}
}
