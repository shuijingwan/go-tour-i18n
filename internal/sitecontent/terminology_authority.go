package sitecontent

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"go/ast"
	"go/doc/comment"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const (
	TerminologyAuthoritySchema = "go-learning/terminology-discovery-sources/v1"
	TerminologyManifestPath    = "data/terminology-discovery-sources.json"
	TerminologySnapshotPath    = "data/terminology-discovery-sources.zip"
	TerminologyWebsiteCommit   = "db076098077c07d3cef1b85a2cf56ff52777f587"
	TerminologyGoVersion       = "go1.27.1"
	TerminologyGoArchiveSHA    = "4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1"
	terminologyPackage         = "terminology-discovery"
)

var terminologyWebsitePaths = []string{
	"_content/ref/mod.md",
	"_content/doc/toolchain.md",
	"_content/doc/go1compat.html",
	"_content/doc/diagnostics.html",
	"_content/doc/pgo.md",
	"_content/doc/build-cover.md",
	"_content/doc/articles/race_detector.html",
	"_content/doc/effective_go.html",
	"_content/doc/faq.md",
	"_content/blog/pprof.md",
	"_content/blog/execution-traces-2024.md",
	"_content/blog/supply-chain.md",
	"_content/blog/govulncheck.md",
	"_content/blog/get-familiar-with-workspaces.md",
}

var terminologyGoPaths = []string{
	"go/doc/go_spec.html",
	"go/doc/go_mem.html",
	"go/src/cmd/go/alldocs.go",
	"go/src/cmd/compile/doc.go",
	"go/src/cmd/link/doc.go",
	"go/src/cmd/pprof/doc.go",
	"go/src/cmd/trace/doc.go",
	"go/src/cmd/vet/doc.go",
	"go/src/testing/testing.go",
	"go/src/runtime/pprof/pprof.go",
	"go/src/runtime/trace/trace.go",
}

type TerminologySourceOrigin struct {
	Repository    string `json:"repository,omitempty"`
	Commit        string `json:"commit,omitempty"`
	Version       string `json:"version,omitempty"`
	ArchiveSHA256 string `json:"archive_sha256,omitempty"`
}

type TerminologySource struct {
	ID           string `json:"id"`
	Namespace    string `json:"namespace"`
	Path         string `json:"path"`
	SnapshotPath string `json:"snapshot_path"`
	Parser       string `json:"parser"`
	SHA256       string `json:"source_sha256"`
	Bytes        int64  `json:"source_bytes"`
}

type TerminologyDiscoveryAuthority struct {
	Schema         string                  `json:"schema"`
	Website        TerminologySourceOrigin `json:"website"`
	GoSource       TerminologySourceOrigin `json:"go_source"`
	Sources        []TerminologySource     `json:"sources"`
	SnapshotSHA256 string                  `json:"snapshot_sha256"`
	Identity       string                  `json:"identity_sha256"`
}

type TerminologyAuthoritySummary struct {
	State              string `json:"state"`
	Applied            bool   `json:"applied"`
	ManifestIdentity   string `json:"manifest_identity_sha256,omitempty"`
	ManifestFileSHA256 string `json:"manifest_file_sha256,omitempty"`
	SnapshotSHA256     string `json:"snapshot_sha256,omitempty"`
	CorpusIdentity     string `json:"corpus_identity_sha256"`
	CorpusFileSHA256   string `json:"corpus_file_sha256,omitempty"`
	Sources            int    `json:"source_count"`
	Contributors       int    `json:"discovery_contributor_count"`
	Contexts           int    `json:"discovery_context_count"`
	ProjectedUTF8Bytes int    `json:"projected_utf8_bytes"`
}

func terminologyAuthorityIdentity(a TerminologyDiscoveryAuthority) string {
	a.Identity = ""
	return identity(a)
}

func terminologySourceID(namespace, p string) string {
	return "discovery:" + namespace + ":" + p
}

func terminologySnapshotMember(namespace, p string) string {
	return namespace + "/" + p
}

func terminologyParser(namespace, p string) string {
	if strings.HasSuffix(p, ".go") {
		return "go-doc-visible/v1"
	}
	if strings.HasSuffix(p, ".md") {
		return "markdown-visible/v1"
	}
	return "html-visible/v1"
}

func expectedTerminologySources() []TerminologySource {
	result := make([]TerminologySource, 0, len(terminologyWebsitePaths)+len(terminologyGoPaths))
	for _, namespace := range []string{"website", "go-source"} {
		paths := terminologyWebsitePaths
		if namespace == "go-source" {
			paths = terminologyGoPaths
		}
		for _, p := range paths {
			result = append(result, TerminologySource{
				ID: terminologySourceID(namespace, p), Namespace: namespace, Path: p,
				SnapshotPath: terminologySnapshotMember(namespace, p), Parser: terminologyParser(namespace, p),
			})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func ValidateTerminologyAuthority(a TerminologyDiscoveryAuthority) error {
	if a.Schema != TerminologyAuthoritySchema || a.Website.Repository != "https://github.com/golang/website.git" || a.Website.Commit != TerminologyWebsiteCommit || a.Website.Version != "" || a.Website.ArchiveSHA256 != "" || a.GoSource.Repository != "" || a.GoSource.Commit != "" || a.GoSource.Version != TerminologyGoVersion || a.GoSource.ArchiveSHA256 != TerminologyGoArchiveSHA || !validSHA(a.SnapshotSHA256) || a.Identity != terminologyAuthorityIdentity(a) {
		return fmt.Errorf("invalid terminology discovery authority identity")
	}
	expected := expectedTerminologySources()
	if len(a.Sources) != len(expected) {
		return fmt.Errorf("terminology discovery source exact-set mismatch")
	}
	for i, source := range a.Sources {
		want := expected[i]
		if source.ID != want.ID || source.Namespace != want.Namespace || source.Path != want.Path || source.SnapshotPath != want.SnapshotPath || source.Parser != want.Parser || !validPath(source.Path) || !validPath(source.SnapshotPath) || !validSHA(source.SHA256) || source.Bytes <= 0 || source.Bytes > 16<<20 {
			return fmt.Errorf("invalid terminology discovery source %q", source.ID)
		}
	}
	return nil
}

func terminologyAuthorityFilesExist(root string) (bool, bool, error) {
	exists := func(p string) (bool, error) {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p)))
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("terminology authority path is not a regular file: %s", p)
		}
		return true, nil
	}
	m, err := exists(TerminologyManifestPath)
	if err != nil {
		return false, false, err
	}
	s, err := exists(TerminologySnapshotPath)
	return m, s, err
}

func LoadTerminologyAuthority(root string) (*TerminologyDiscoveryAuthority, []byte, error) {
	manifest, err := readRegular(root, TerminologyManifestPath)
	if err != nil {
		return nil, nil, err
	}
	if !utf8.Valid(manifest) || bytes.HasPrefix(manifest, []byte{0xef, 0xbb, 0xbf}) {
		return nil, nil, fmt.Errorf("terminology manifest must be UTF-8 without BOM")
	}
	var authority TerminologyDiscoveryAuthority
	if err := StrictJSON(manifest, &authority); err != nil {
		return nil, nil, err
	}
	if err := ValidateTerminologyAuthority(authority); err != nil {
		return nil, nil, err
	}
	snapshot, err := readRegular(root, TerminologySnapshotPath)
	if err != nil {
		return nil, nil, err
	}
	if digest(snapshot) != authority.SnapshotSHA256 {
		return nil, nil, fmt.Errorf("terminology snapshot SHA-256 mismatch")
	}
	if _, err := readTerminologySnapshot(authority, snapshot); err != nil {
		return nil, nil, err
	}
	return &authority, snapshot, nil
}

func verifyWebsiteCheckout(root, expectedCommit string) error {
	git := func(args ...string) (string, error) {
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
		return strings.TrimSpace(string(out)), err
	}
	head, err := git("rev-parse", "HEAD")
	if err != nil || head != expectedCommit {
		return fmt.Errorf("terminology website HEAD mismatch: got %q, want %q", head, expectedCommit)
	}
	status, err := git("status", "--porcelain", "--untracked-files=all")
	if err != nil || status != "" {
		return fmt.Errorf("terminology website checkout must be clean")
	}
	return nil
}

func readWebsiteTerminologySources(root, expectedCommit string) (map[string][]byte, error) {
	if err := verifyWebsiteCheckout(root, expectedCommit); err != nil {
		return nil, err
	}
	result := map[string][]byte{}
	for _, p := range terminologyWebsitePaths {
		name := filepath.Join(root, filepath.FromSlash(p))
		info, err := os.Lstat(name)
		if err != nil {
			return nil, fmt.Errorf("required terminology website source %s: %w", p, err)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 16<<20 {
			return nil, fmt.Errorf("unsafe terminology website source %s", p)
		}
		b, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		if int64(len(b)) != info.Size() || !utf8.Valid(b) || bytes.HasPrefix(b, []byte{0xef, 0xbb, 0xbf}) || bytes.ContainsRune(b, 0) {
			return nil, fmt.Errorf("invalid terminology website source %s", p)
		}
		result[terminologySnapshotMember("website", p)] = b
	}
	if err := verifyWebsiteCheckout(root, expectedCommit); err != nil {
		return nil, fmt.Errorf("terminology website checkout changed while reading: %w", err)
	}
	return result, nil
}

func readGoTerminologySources(archivePath, expectedSHA string) (map[string][]byte, error) {
	info, err := os.Lstat(archivePath)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 128<<20 {
		return nil, fmt.Errorf("Go source archive must be a bounded regular file")
	}
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, err
	}
	if digest(archive) != expectedSHA {
		return nil, fmt.Errorf("Go source archive SHA-256 mismatch: got %s, want %s", digest(archive), expectedSHA)
	}
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	wanted := map[string]bool{}
	for _, p := range terminologyGoPaths {
		wanted[p] = true
	}
	result := map[string][]byte{}
	seen := map[string]bool{}
	var total int64
	t := tar.NewReader(gz)
	for {
		h, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		memberName := strings.TrimSuffix(h.Name, "/")
		if !validPath(memberName) || strings.Contains(h.Name, "\\") || seen[h.Name] {
			return nil, fmt.Errorf("unsafe or duplicate Go archive member %q", h.Name)
		}
		seen[h.Name] = true
		switch h.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg, tar.TypeRegA:
		default:
			return nil, fmt.Errorf("unsupported Go archive member type %q", h.Name)
		}
		if h.Size < 0 || h.Size > 32<<20 {
			return nil, fmt.Errorf("oversize Go archive member %q", h.Name)
		}
		total += h.Size
		if total > 512<<20 {
			return nil, fmt.Errorf("Go source archive expands beyond safety limit")
		}
		if !wanted[h.Name] {
			if _, err := io.Copy(io.Discard, t); err != nil {
				return nil, err
			}
			continue
		}
		b, err := io.ReadAll(io.LimitReader(t, 16<<20+1))
		if err != nil {
			return nil, err
		}
		if len(b) == 0 || len(b) > 16<<20 || int64(len(b)) != h.Size || !utf8.Valid(b) || bytes.HasPrefix(b, []byte{0xef, 0xbb, 0xbf}) || bytes.ContainsRune(b, 0) {
			return nil, fmt.Errorf("invalid selected Go source %s", h.Name)
		}
		result[terminologySnapshotMember("go-source", h.Name)] = b
	}
	for p := range wanted {
		if result[terminologySnapshotMember("go-source", p)] == nil {
			return nil, fmt.Errorf("required Go source archive path absent: %s", p)
		}
	}
	return result, nil
}

func writeTerminologySnapshot(files map[string][]byte) ([]byte, error) {
	var buffer bytes.Buffer
	z := zip.NewWriter(&buffer)
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if !validPath(p) {
			return nil, fmt.Errorf("unsafe terminology snapshot path %q", p)
		}
		h := &zip.FileHeader{Name: p, Method: zip.Deflate}
		h.SetMode(0644)
		w, err := z.CreateHeader(h)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(files[p]); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func readTerminologySnapshot(authority TerminologyDiscoveryAuthority, snapshot []byte) (map[string][]byte, error) {
	z, err := zip.NewReader(bytes.NewReader(snapshot), int64(len(snapshot)))
	if err != nil {
		return nil, err
	}
	wanted := map[string]TerminologySource{}
	for _, source := range authority.Sources {
		wanted[source.SnapshotPath] = source
	}
	if len(z.File) != len(wanted) {
		return nil, fmt.Errorf("terminology snapshot exact-set mismatch")
	}
	result := map[string][]byte{}
	var total int64
	for _, f := range z.File {
		source, ok := wanted[f.Name]
		if !ok || !validPath(f.Name) || result[f.Name] != nil || !f.Mode().IsRegular() || f.UncompressedSize64 > 16<<20 {
			return nil, fmt.Errorf("invalid terminology snapshot member %q", f.Name)
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, readErr := io.ReadAll(io.LimitReader(r, 16<<20+1))
		closeErr := r.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		total += int64(len(b))
		if len(b) == 0 || len(b) > 16<<20 || total > 128<<20 || int64(len(b)) != source.Bytes || digest(b) != source.SHA256 || !utf8.Valid(b) || bytes.ContainsRune(b, 0) {
			return nil, fmt.Errorf("terminology snapshot member identity mismatch: %s", f.Name)
		}
		result[f.Name] = b
	}
	canonical, err := writeTerminologySnapshot(result)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, snapshot) {
		return nil, fmt.Errorf("terminology snapshot is not canonical/reproducible")
	}
	return result, nil
}

func buildTerminologyAuthority(websiteRoot, archivePath string) (*TerminologyDiscoveryAuthority, []byte, error) {
	website, err := readWebsiteTerminologySources(websiteRoot, TerminologyWebsiteCommit)
	if err != nil {
		return nil, nil, err
	}
	goSources, err := readGoTerminologySources(archivePath, TerminologyGoArchiveSHA)
	if err != nil {
		return nil, nil, err
	}
	for p, b := range goSources {
		website[p] = b
	}
	snapshot, err := writeTerminologySnapshot(website)
	if err != nil {
		return nil, nil, err
	}
	authority := &TerminologyDiscoveryAuthority{
		Schema:   TerminologyAuthoritySchema,
		Website:  TerminologySourceOrigin{Repository: "https://github.com/golang/website.git", Commit: TerminologyWebsiteCommit},
		GoSource: TerminologySourceOrigin{Version: TerminologyGoVersion, ArchiveSHA256: TerminologyGoArchiveSHA},
		Sources:  []TerminologySource{}, SnapshotSHA256: digest(snapshot),
	}
	for _, source := range expectedTerminologySources() {
		b := website[source.SnapshotPath]
		source.SHA256 = digest(b)
		source.Bytes = int64(len(b))
		authority.Sources = append(authority.Sources, source)
	}
	authority.Identity = terminologyAuthorityIdentity(*authority)
	if err := ValidateTerminologyAuthority(*authority); err != nil {
		return nil, nil, err
	}
	return authority, snapshot, nil
}

func discoveryMetadata(b []byte) (string, int, error) {
	if bytes.HasPrefix(b, []byte("<!--{")) {
		end := bytes.Index(b, []byte("}-->"))
		if end < 0 {
			return "", 0, fmt.Errorf("unterminated JSON metadata")
		}
		var metadata map[string]any
		if err := StrictJSON(b[4:end+1], &metadata); err != nil {
			return "", 0, err
		}
		title := ""
		titleSeen := false
		for key, value := range metadata {
			if strings.EqualFold(key, "title") {
				if titleSeen {
					return "", 0, fmt.Errorf("duplicate metadata title")
				}
				titleSeen = true
				var ok bool
				title, ok = value.(string)
				if !ok {
					return "", 0, fmt.Errorf("metadata title must be text")
				}
			}
		}
		return title, end + 4, nil
	}
	if !bytes.HasPrefix(b, []byte("---\n")) {
		return "", 0, nil
	}
	end := bytes.Index(b[4:], []byte("\n---\n"))
	if end < 0 {
		return "", 0, fmt.Errorf("unterminated YAML metadata")
	}
	title := ""
	for _, line := range strings.Split(string(b[4:4+end]), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "title:") {
			if title != "" {
				return "", 0, fmt.Errorf("duplicate metadata title")
			}
			title = strings.TrimSpace(strings.TrimPrefix(trimmed, "title:"))
			if strings.HasPrefix(title, `"`) || strings.HasPrefix(title, `'`) {
				quote := title[0]
				if len(title) < 2 || title[len(title)-1] != quote || strings.Contains(title[1:len(title)-1], string(quote)) {
					return "", 0, fmt.Errorf("unsupported quoted metadata title")
				}
				title = title[1 : len(title)-1]
			}
		}
	}
	return title, 4 + end + 5, nil
}

func projectedContextsFromSpans(b []byte, spans []span, initial ...LanguageProjection) ([]i18n.LanguageContext, error) {
	projections := append([]LanguageProjection{}, initial...)
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	previous := 0
	for _, sp := range spans {
		if sp.start < previous || sp.end > len(b) {
			return nil, fmt.Errorf("overlapping terminology parser spans")
		}
		previous = sp.end
		raw := bytes.TrimSpace(b[sp.start:sp.end])
		if len(raw) == 0 {
			continue
		}
		text, locked, err := protectBlock(raw, sp.context)
		if err != nil {
			return nil, err
		}
		protected := []string{}
		for _, item := range locked {
			text = strings.ReplaceAll(text, item.Token, "")
			if !strings.Contains(item.Raw, "\n") && len(item.Raw) < 256 {
				protected = append(protected, item.Raw)
			}
		}
		text = strings.TrimSpace(text)
		if text != "" && hasLanguage(text) {
			projections = append(projections, LanguageProjection{Text: text, Protected: protected})
		}
	}
	return finalizeLanguageContexts(projections), nil
}

type LanguageProjection struct {
	Text      string
	Protected []string
}

func finalizeLanguageContexts(projections []LanguageProjection) []i18n.LanguageContext {
	counts := map[string]int{}
	result := []i18n.LanguageContext{}
	for _, projection := range projections {
		projection.Text = strings.TrimSpace(projection.Text)
		if projection.Text == "" || !utf8.ValidString(projection.Text) || strings.ContainsRune(projection.Text, 0) {
			continue
		}
		projection.Protected = uniqueStrings(projection.Protected)
		base := identity(projection)
		counts[base]++
		result = append(result, i18n.LanguageContext{ID: fmt.Sprintf("%s:%06d", base, counts[base]), Text: projection.Text, Protected: projection.Protected})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func projectMarkupTerminology(source TerminologySource, b []byte) ([]i18n.LanguageContext, error) {
	title, offset, err := discoveryMetadata(b)
	if err != nil {
		return nil, err
	}
	initial := []LanguageProjection{}
	if strings.TrimSpace(title) != "" {
		initial = append(initial, LanguageProjection{Text: title, Protected: []string{}})
	}
	body := b[offset:]
	spans := []span{}
	add := func(a, z int, context string) { spans = append(spans, span{a, z, context}) }
	if source.Parser == "markdown-visible/v1" {
		if err := markdownBlocks(body, offset, add); err != nil {
			return nil, err
		}
		templateRanges, err := terminologyTemplateRanges(body, offset)
		if err != nil {
			return nil, err
		}
		spans = subtractTerminologyRanges(spans, templateRanges)
	} else if source.Parser == "html-visible/v1" {
		if err := htmlSpans(body, offset, "html-visible", add); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown markup terminology parser")
	}
	return projectedContextsFromSpans(b, spans, initial...)
}

func terminologyTemplateRanges(b []byte, offset int) ([]span, error) {
	ranges := []span{}
	for start := 0; ; {
		relative := bytes.Index(b[start:], []byte("{{"))
		if relative < 0 {
			break
		}
		a := start + relative
		quote := byte(0)
		escaped := false
		end := -1
		for i := a + 2; i < len(b)-1; i++ {
			c := b[i]
			if quote != 0 {
				if quote == '"' && c == '\\' && !escaped {
					escaped = true
					continue
				}
				if c == quote && !escaped {
					quote = 0
				}
				escaped = false
				continue
			}
			if c == '"' || c == '\'' || c == '`' {
				quote = c
				continue
			}
			if c == '}' && b[i+1] == '}' {
				end = i + 2
				break
			}
		}
		if end < 0 || quote != 0 {
			return nil, fmt.Errorf("unterminated template action")
		}
		ranges = append(ranges, span{start: offset + a, end: offset + end, context: "template"})
		start = end
	}
	return ranges, nil
}

func subtractTerminologyRanges(spans, excluded []span) []span {
	result := []span{}
	for _, candidate := range spans {
		parts := []span{candidate}
		for _, exclusion := range excluded {
			next := []span{}
			for _, part := range parts {
				if exclusion.end <= part.start || exclusion.start >= part.end {
					next = append(next, part)
					continue
				}
				if part.start < exclusion.start {
					next = append(next, span{start: part.start, end: exclusion.start, context: part.context})
				}
				if exclusion.end < part.end {
					next = append(next, span{start: exclusion.end, end: part.end, context: part.context})
				}
			}
			parts = next
		}
		result = append(result, parts...)
	}
	return result
}

func technicalDocTokens(text string) []string {
	result := []string{}
	for _, field := range strings.Fields(text) {
		token := strings.TrimFunc(field, func(r rune) bool {
			return unicode.IsPunct(r) && r != '-' && r != '_' && r != '/' && r != '.' && r != '='
		})
		if token == "" || len(token) > 128 {
			continue
		}
		if strings.HasPrefix(token, "-") || strings.ContainsAny(token, "/=") || (strings.Contains(token, ".") && !strings.HasSuffix(token, ".")) {
			result = append(result, token)
		}
	}
	return uniqueStrings(result)
}

func renderDocText(values []comment.Text) (string, []string) {
	var text strings.Builder
	protected := []string{}
	var walk func([]comment.Text)
	walk = func(values []comment.Text) {
		for _, value := range values {
			switch value := value.(type) {
			case comment.Plain:
				text.WriteString(string(value))
			case comment.Italic:
				text.WriteString(string(value))
			case *comment.Link:
				walk(value.Text)
				protected = append(protected, value.URL)
			case *comment.DocLink:
				walk(value.Text)
				target := strings.Trim(strings.Join([]string{value.ImportPath, value.Recv, value.Name}, "."), ".")
				if target != "" {
					protected = append(protected, target)
				}
			}
		}
	}
	walk(values)
	protected = append(protected, technicalDocTokens(text.String())...)
	return strings.TrimSpace(text.String()), uniqueStrings(protected)
}

func docCommentProjections(group *ast.CommentGroup) []LanguageProjection {
	if group == nil {
		return nil
	}
	doc := new(comment.Parser).Parse(group.Text())
	result := []LanguageProjection{}
	var blocks func([]comment.Block)
	blocks = func(values []comment.Block) {
		for _, value := range values {
			switch value := value.(type) {
			case *comment.Paragraph:
				text, protected := renderDocText(value.Text)
				result = append(result, LanguageProjection{Text: text, Protected: protected})
			case *comment.Heading:
				text, protected := renderDocText(value.Text)
				result = append(result, LanguageProjection{Text: text, Protected: protected})
			case *comment.List:
				for _, item := range value.Items {
					blocks(item.Content)
				}
			case *comment.Code:
				// Code examples and command transcripts are deliberately excluded.
			}
		}
	}
	blocks(doc.Content)
	return result
}

func exportedField(field *ast.Field) bool {
	for _, name := range field.Names {
		if name.IsExported() {
			return true
		}
	}
	if len(field.Names) != 0 {
		return false
	}
	switch value := field.Type.(type) {
	case *ast.Ident:
		return value.IsExported()
	case *ast.SelectorExpr:
		return value.Sel.IsExported()
	case *ast.StarExpr:
		if name, ok := value.X.(*ast.Ident); ok {
			return name.IsExported()
		}
	}
	return false
}

func projectGoDocTerminology(name string, b []byte) ([]i18n.LanguageContext, error) {
	f, err := parser.ParseFile(token.NewFileSet(), name, b, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("malformed Go terminology source %s: %w", name, err)
	}
	groups := map[token.Pos]*ast.CommentGroup{}
	add := func(group *ast.CommentGroup) {
		if group != nil {
			groups[group.Pos()] = group
		}
	}
	add(f.Doc)
	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if decl.Name.IsExported() {
				add(decl.Doc)
			}
		case *ast.GenDecl:
			exported := false
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if spec.Name.IsExported() {
						exported = true
						add(spec.Doc)
						add(spec.Comment)
						ast.Inspect(spec.Type, func(node ast.Node) bool {
							field, ok := node.(*ast.Field)
							if ok && exportedField(field) {
								add(field.Doc)
								add(field.Comment)
							}
							return true
						})
					}
				case *ast.ValueSpec:
					for _, identifier := range spec.Names {
						if identifier.IsExported() {
							exported = true
							add(spec.Doc)
							add(spec.Comment)
							break
						}
					}
				}
			}
			if exported {
				add(decl.Doc)
			}
		}
	}
	positions := make([]int, 0, len(groups))
	for position := range groups {
		positions = append(positions, int(position))
	}
	sort.Ints(positions)
	projections := []LanguageProjection{}
	for _, position := range positions {
		projections = append(projections, docCommentProjections(groups[token.Pos(position)])...)
	}
	contexts := finalizeLanguageContexts(projections)
	if len(contexts) == 0 {
		return nil, fmt.Errorf("Go source has no package/exported documentation: %s", name)
	}
	return contexts, nil
}

func ProjectTerminologyContributors(authority TerminologyDiscoveryAuthority, snapshot []byte) ([]i18n.LanguageContributor, error) {
	if err := ValidateTerminologyAuthority(authority); err != nil {
		return nil, err
	}
	if digest(snapshot) != authority.SnapshotSHA256 {
		return nil, fmt.Errorf("terminology snapshot SHA-256 mismatch")
	}
	files, err := readTerminologySnapshot(authority, snapshot)
	if err != nil {
		return nil, err
	}
	result := make([]i18n.LanguageContributor, 0, len(authority.Sources))
	for _, source := range authority.Sources {
		b := files[source.SnapshotPath]
		var contexts []i18n.LanguageContext
		if source.Parser == "go-doc-visible/v1" {
			contexts, err = projectGoDocTerminology(source.Path, b)
		} else {
			contexts, err = projectMarkupTerminology(source, b)
		}
		if err != nil {
			return nil, fmt.Errorf("project terminology source %s: %w", source.ID, err)
		}
		if len(contexts) == 0 {
			return nil, fmt.Errorf("empty terminology projection: %s", source.ID)
		}
		contributor := i18n.LanguageContributor{
			ID: source.ID, Parser: source.Parser, Package: terminologyPackage,
			Path: source.SnapshotPath, Route: "", Structured: false,
			Purpose: i18n.ContributorPurposeTerminologyDiscovery, Contexts: contexts,
		}
		contributor.SourceSHA = i18n.ContributorIdentity(contributor)
		result = append(result, contributor)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func terminologySummary(authority *TerminologyDiscoveryAuthority, snapshot []byte, corpus *i18n.GlossarySourceCorpus, applied bool) (*TerminologyAuthoritySummary, error) {
	summary := &TerminologyAuthoritySummary{Applied: applied, CorpusIdentity: corpus.Identity}
	if authority == nil {
		summary.State = "not-applied"
		return summary, nil
	}
	if err := ValidateTerminologyAuthority(*authority); err != nil {
		return nil, err
	}
	if digest(snapshot) != authority.SnapshotSHA256 {
		return nil, fmt.Errorf("terminology snapshot SHA-256 mismatch")
	}
	if _, err := readTerminologySnapshot(*authority, snapshot); err != nil {
		return nil, err
	}
	contributors := []i18n.LanguageContributor{}
	for _, contributor := range corpus.Contributors {
		if i18n.EffectiveContributorPurpose(contributor) == i18n.ContributorPurposeTerminologyDiscovery {
			contributors = append(contributors, contributor)
		}
	}
	manifest, err := Encode(authority)
	if err != nil {
		return nil, err
	}
	summary.State = "active"
	summary.ManifestIdentity = authority.Identity
	summary.ManifestFileSHA256 = digest(manifest)
	summary.SnapshotSHA256 = authority.SnapshotSHA256
	summary.Sources = len(authority.Sources)
	summary.Contributors = len(contributors)
	for _, contributor := range contributors {
		for _, context := range contributor.Contexts {
			summary.Contexts++
			summary.ProjectedUTF8Bytes += len(context.Text)
			for _, protected := range context.Protected {
				summary.ProjectedUTF8Bytes += len(protected)
			}
		}
	}
	return summary, nil
}

func CheckTerminologyAuthority(root string) (*TerminologyAuthoritySummary, error) {
	manifestExists, snapshotExists, err := terminologyAuthorityFilesExist(root)
	if err != nil {
		return nil, err
	}
	if manifestExists != snapshotExists {
		return nil, fmt.Errorf("terminology discovery authority is incomplete")
	}
	corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return nil, err
	}
	corpusBytes, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		return nil, err
	}
	if !manifestExists {
		if corpus.Schema != i18n.GlossaryCorpusSchema {
			return nil, fmt.Errorf("v2 corpus exists without terminology discovery authority")
		}
		want, err := currentCorpusProjection(root, nil, nil)
		if err != nil {
			return nil, err
		}
		if identity(want) != identity(corpus) {
			return nil, fmt.Errorf("inactive unified corpus is stale")
		}
		summary, err := terminologySummary(nil, nil, corpus, false)
		if summary != nil {
			summary.CorpusFileSHA256 = digest(corpusBytes)
		}
		return summary, err
	}
	authority, snapshot, err := LoadTerminologyAuthority(root)
	if err != nil {
		return nil, err
	}
	want, err := currentCorpusProjection(root, authority, snapshot)
	if err != nil {
		return nil, err
	}
	if identity(want) != identity(corpus) {
		return nil, fmt.Errorf("terminology discovery projection or unified corpus is stale")
	}
	summary, err := terminologySummary(authority, snapshot, corpus, true)
	if summary != nil {
		summary.CorpusFileSHA256 = digest(corpusBytes)
	}
	return summary, err
}

func writeCASFile(root, relative string, data []byte, prior string, allowCreate bool) error {
	name := filepath.Join(root, filepath.FromSlash(relative))
	info, err := os.Lstat(name)
	if os.IsNotExist(err) {
		if !allowCreate || prior != "" {
			return fmt.Errorf("CAS target missing: %s", relative)
		}
	} else if err != nil {
		return err
	} else {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe CAS target: %s", relative)
		}
		current, err := os.ReadFile(name)
		if err != nil || digest(current) != prior {
			return fmt.Errorf("CAS target changed: %s", relative)
		}
	}
	temp, err := os.CreateTemp(filepath.Dir(name), ".terminology-authority-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0644); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), name)
}

// PrepareTerminologyAuthority performs a fully offline preflight. Apply is an
// explicit CAS transition of the two raw source artifacts and unified corpus;
// it never writes locale, glossary, Review, TranslationUnit, or package state.
func PrepareTerminologyAuthority(root, websiteRoot, archivePath, priorCorpusSHA, priorTerminologySHA string, apply bool) (*TerminologyAuthoritySummary, error) {
	authority, snapshot, err := buildTerminologyAuthority(websiteRoot, archivePath)
	if err != nil {
		return nil, err
	}
	if err := CheckUnifiedCorpus(root); err != nil {
		return nil, fmt.Errorf("current unified corpus preflight: %w", err)
	}
	corpus, err := buildUnifiedCorpus(root, authority, snapshot)
	if err != nil {
		return nil, err
	}
	manifest, err := Encode(authority)
	if err != nil {
		return nil, err
	}
	corpusBytes, err := Encode(corpus)
	if err != nil {
		return nil, err
	}
	summary, err := terminologySummary(authority, snapshot, corpus, apply)
	if err != nil {
		return nil, err
	}
	summary.CorpusFileSHA256 = digest(corpusBytes)
	if !apply {
		summary.State = "preflight"
		summary.Applied = false
		return summary, nil
	}
	if !validSHA(priorCorpusSHA) {
		return nil, fmt.Errorf("--prior-sha256 exact current corpus file SHA is required for --apply")
	}
	currentCorpus, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil || digest(currentCorpus) != priorCorpusSHA {
		return nil, fmt.Errorf("static corpus authority changed")
	}
	manifestExists, snapshotExists, err := terminologyAuthorityFilesExist(root)
	if err != nil || manifestExists != snapshotExists {
		return nil, fmt.Errorf("terminology discovery authority is incomplete")
	}
	oldManifestSHA, oldSnapshotSHA := "", ""
	if manifestExists {
		if _, _, err := LoadTerminologyAuthority(root); err != nil {
			return nil, fmt.Errorf("refusing to overwrite unrecognized terminology authority: %w", err)
		}
		oldManifest, err := readRegular(root, TerminologyManifestPath)
		if err != nil {
			return nil, err
		}
		oldSnapshot, err := readRegular(root, TerminologySnapshotPath)
		if err != nil {
			return nil, err
		}
		oldManifestSHA, oldSnapshotSHA = digest(oldManifest), digest(oldSnapshot)
		if priorTerminologySHA == "" || priorTerminologySHA != oldManifestSHA {
			return nil, fmt.Errorf("--prior-terminology-sha256 exact current manifest file SHA is required for refresh")
		}
	} else if priorTerminologySHA != "" {
		return nil, fmt.Errorf("--prior-terminology-sha256 supplied for initial creation")
	}
	lock, err := os.OpenFile(filepath.Join(root, "data", ".terminology-authority.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	lock.Close()
	defer os.Remove(lock.Name())
	if err := writeCASFile(root, TerminologySnapshotPath, snapshot, oldSnapshotSHA, !snapshotExists); err != nil {
		return nil, err
	}
	if err := writeCASFile(root, TerminologyManifestPath, manifest, oldManifestSHA, !manifestExists); err != nil {
		return nil, err
	}
	if err := writeCASFile(root, i18n.GlossaryCorpusPath, corpusBytes, priorCorpusSHA, false); err != nil {
		return nil, err
	}
	return summary, nil
}
