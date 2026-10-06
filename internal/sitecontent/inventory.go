package sitecontent

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/tour"
)

var learnRoots = []string{"_content/learn", "_content/doc/tutorial", "_content/doc/database", "_content/doc/modules", "_content/doc/security"}
var contractPaths = []string{"internal/web/page.go", "internal/web/site.go", "internal/redirect/redirect.go"}

const UpstreamCommit = tour.FrozenUpstreamCommit

var dataRE = regexp.MustCompile(`\bdata\s+"([^"]+\.yaml)"`)
var assetRE = regexp.MustCompile(`(?:[A-Za-z0-9_./-]+\.(?:png|gif|svg|jpg|jpeg|webp|graffle))`)

func listFiles(fsys fs.FS, directory string) ([]string, error) {
	var files []string
	err := fs.WalkDir(fsys, directory, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("symlink in source %s", p)
		}
		if !d.IsDir() {
			if !d.Type().IsRegular() {
				return fmt.Errorf("unsupported source %s", p)
			}
			files = append(files, p)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

// FrozenSnapshot reads only the frozen, clean checkout. It imports a reviewable
// inventory snapshot outside runtime _content; it is not an upstream source sync.
func FrozenSnapshot(sourceRoot string) ([]byte, error) {
	git := func(args ...string) (string, error) {
		b, err := exec.Command("git", append([]string{"-C", sourceRoot}, args...)...).Output()
		return strings.TrimSpace(string(b)), err
	}
	head, err := git("rev-parse", "HEAD")
	if err != nil || head != tour.FrozenUpstreamCommit {
		return nil, fmt.Errorf("upstream HEAD mismatch: %s", head)
	}
	status, err := git("status", "--porcelain", "--untracked-files=all")
	if err != nil || status != "" {
		return nil, fmt.Errorf("upstream checkout must be clean")
	}
	f := os.DirFS(sourceRoot)
	paths := append([]string{}, contractPaths...)
	for _, root := range learnRoots {
		files, err := listFiles(f, root)
		if err != nil {
			return nil, err
		}
		paths = append(paths, files...)
	}
	// Direct media and fragment dependencies outside the 80-file package are
	// separately bound. They do not silently enlarge the translation campaign.
	for _, p := range append([]string{}, paths...) {
		b, err := fs.ReadFile(f, p)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(p, "_content/") && (path.Ext(p) == ".yaml" || path.Ext(p) == ".md" || path.Ext(p) == ".html") {
			for _, d := range dependencyPaths(p, b) {
				if _, err := fs.Stat(f, d.Path); err == nil {
					paths = append(paths, d.Path)
				} else {
					return nil, fmt.Errorf("unresolved dependency %s in %s: %w", d.Path, p, err)
				}
			}
		}
	}
	paths = append(paths, "_content/site.tmpl", "_content/article.tmpl", "_content/doc/default.tmpl")
	sort.Strings(paths)
	var buffer bytes.Buffer
	z := zip.NewWriter(&buffer)
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		b, err := fs.ReadFile(f, p)
		if err != nil {
			return nil, err
		}
		h := &zip.FileHeader{Name: p, Method: zip.Deflate}
		h.SetMode(0644)
		w, err := z.CreateHeader(h)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(b); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func snapshotFS(b []byte) (fs.FS, error) {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	total := int64(0)
	for _, f := range z.File {
		if !validPath(f.Name) || seen[f.Name] || !f.Mode().IsRegular() || f.UncompressedSize64 > 16<<20 {
			return nil, fmt.Errorf("invalid snapshot entry %s", f.Name)
		}
		seen[f.Name] = true
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		n, readErr := io.Copy(io.Discard, io.LimitReader(r, 16<<20+1))
		closeErr := r.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		total += n
		if n > 16<<20 || total > 64<<20 {
			return nil, fmt.Errorf("snapshot too large")
		}
	}
	return z, nil
}

func pageRoute(p string) string {
	r := strings.TrimPrefix(p, "_content")
	if strings.HasSuffix(r, "/index.md") {
		return strings.TrimSuffix(r, "index.md")
	}
	if strings.HasSuffix(r, "/index.html") {
		return strings.TrimSuffix(r, "index.html")
	}
	return strings.TrimSuffix(r, path.Ext(r))
}

// Match frozen web.parseMeta's two metadata envelopes and JSON key folding.
// Malformed/ambiguous metadata fails closed instead of becoming a page.
func redirectMetadata(b []byte) (string, error) {
	if bytes.HasPrefix(b, []byte("<!--{")) {
		end := bytes.Index(b, []byte("}-->"))
		if end < 0 {
			return "", fmt.Errorf("unterminated JSON metadata")
		}
		var raw map[string]json.RawMessage
		if err := StrictJSON(b[4:end+1], &raw); err != nil {
			return "", err
		}
		folded := map[string]json.RawMessage{}
		for k, v := range raw {
			key := strings.ToLower(k)
			if _, exists := folded[key]; exists {
				return "", fmt.Errorf("ambiguous metadata key %s", key)
			}
			folded[key] = v
		}
		if v, ok := folded["redirect"]; ok {
			var target string
			if err := json.Unmarshal(v, &target); err != nil {
				return "", err
			}
			return target, nil
		}
		return "", nil
	}
	if !bytes.HasPrefix(b, []byte("---\n")) {
		return "", nil
	}
	end := bytes.Index(b[4:], []byte("\n---"))
	if end < 0 {
		return "", fmt.Errorf("unterminated YAML metadata")
	}
	for _, line := range strings.Split(string(b[4:4+end]), "\n") {
		if strings.HasPrefix(line, "redirect:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "redirect:")), `"'`), nil
		}
	}
	return "", nil
}

func redirectsFromSource(b []byte) (map[string]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "redirect.go", b, 0)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		v, ok := n.(*ast.ValueSpec)
		if !ok || len(v.Names) != 1 || v.Names[0].Name != "redirects" || len(v.Values) != 1 {
			return true
		}
		c, ok := v.Values[0].(*ast.CompositeLit)
		if !ok {
			return false
		}
		for _, e := range c.Elts {
			kv, ok := e.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			k, kok := kv.Key.(*ast.BasicLit)
			val, vok := kv.Value.(*ast.BasicLit)
			if !kok || !vok {
				continue
			}
			key, e1 := strconv.Unquote(k.Value)
			value, e2 := strconv.Unquote(val.Value)
			if e1 == nil && e2 == nil {
				m[key] = value
			}
		}
		return false
	})
	if m["/security/"] != "/doc/security/" {
		return nil, fmt.Errorf("unknown security canonical contract")
	}
	return m, nil
}

func dependencyPaths(p string, b []byte) []Dependency {
	m := map[string]Dependency{}
	for _, hit := range dataRE.FindAllSubmatch(b, -1) {
		dp := path.Join(path.Dir(p), string(hit[1]))
		m[dp] = Dependency{Kind: "data", Path: dp}
	}
	for _, hit := range assetRE.FindAllString(string(b), -1) {
		// URLs with external scheme/host are retained as external links, not local assets.
		if strings.HasPrefix(hit, "//") {
			continue
		}
		idx := strings.Index(string(b), hit)
		if idx >= 0 && idx >= 3 && string(b[idx-3:idx]) == "://" {
			continue
		}
		dp := ""
		if strings.HasPrefix(hit, "/") {
			if strings.HasPrefix(hit, "/security/") {
				hit = "/doc" + hit
			}
			dp = "_content" + hit
		} else {
			dp = path.Join(path.Dir(p), hit)
		}
		m[dp] = Dependency{Kind: "asset", Path: dp}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := []Dependency{}
	for _, k := range keys {
		result = append(result, m[k])
	}
	return result
}

func learnInventory(f fs.FS) ([]Source, error) {
	raw, err := fs.ReadFile(f, "internal/redirect/redirect.go")
	if err != nil {
		return nil, err
	}
	redirects, err := redirectsFromSource(raw)
	if err != nil {
		return nil, err
	}
	all := []Source{}
	for i, root := range learnRoots {
		files, err := listFiles(f, root)
		if err != nil {
			return nil, err
		}
		for _, p := range files {
			b, err := fs.ReadFile(f, p)
			if err != nil {
				return nil, err
			}
			s := Source{Path: p, Origin: "frozen-upstream", Package: "learn-docs-v1", Surface: []string{"learn", "tutorial", "database", "modules", "security"}[i], SHA256: digest(b), Dependencies: []Dependency{}}
			switch path.Ext(p) {
			case ".yaml":
				s.Kind = "data"
			case ".md", ".html":
				s.Kind = "page"
				s.Route = pageRoute(p)
				target, err := redirectMetadata(b)
				if err != nil {
					return nil, fmt.Errorf("metadata %s: %w", p, err)
				}
				if target != "" {
					s.Kind = "redirect"
					s.Alias = s.Route
					s.Route = ""
					s.Redirect = target
				}
			case ".png", ".gif", ".graffle":
				s.Kind = "asset"
			default:
				return nil, fmt.Errorf("ambiguous/unknown source role: %s", p)
			}
			if s.Kind == "redirect" {
				// Preserve original destination and bind its redirect implementation.
				s.Dependencies = append(s.Dependencies, Dependency{"contract", "internal/redirect/redirect.go", digest(raw)})
			} else if s.Kind == "page" || s.Kind == "data" {
				s.Dependencies = dependencyPaths(p, b)
			}
			for j := range s.Dependencies {
				data, err := fs.ReadFile(f, s.Dependencies[j].Path)
				if err != nil {
					return nil, fmt.Errorf("dependency %s: %w", s.Dependencies[j].Path, err)
				}
				s.Dependencies[j].SHA256 = digest(data)
			}
			all = append(all, s)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Path < all[j].Path })
	// Resolve the actual frozen ServeMux map, including subtree handlers whose
	// Handler redirects to a fixed destination (it does not append a suffix).
	aliases := map[string]string{}
	for _, s := range all {
		if s.Kind == "redirect" {
			aliases[s.Alias] = s.Redirect
		}
	}
	for i := range all {
		if all[i].Kind != "redirect" {
			continue
		}
		t := all[i].Redirect
		seen := map[string]bool{}
		for {
			if seen[t] {
				return nil, fmt.Errorf("redirect cycle: %s", t)
			}
			seen[t] = true
			if next := aliases[t]; next != "" {
				t = next
				continue
			}
			if next := redirects[t]; next != "" {
				t = next
				continue
			}
			if strings.HasPrefix(t, "/security/") || t == "/security" {
				best := ""
				for k := range redirects {
					if strings.HasSuffix(k, "/") && (strings.HasPrefix(t, k) || t+"/" == k) && len(k) > len(best) {
						best = k
					}
				}
				if best == "" {
					return nil, fmt.Errorf("unmapped legacy alias %s", t)
				}
				t = redirects[best]
				continue
			}
			if !strings.HasPrefix(t, "/") && !validateExternalDestination(t) {
				return nil, fmt.Errorf("unknown redirect destination %s", t)
			}
			break
		}
		all[i].ResolvedRedirect = t
	}
	return all, nil
}

func BuildGlobal(root string, snapshot []byte) (*Global, error) {
	f, err := snapshotFS(snapshot)
	if err != nil {
		return nil, err
	}
	sources, err := learnInventory(f)
	if err != nil {
		return nil, err
	}
	catalog, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		return nil, err
	}
	g := &Global{Schema: GlobalSchema, PublicName: PublicName, UpstreamRepository: "https://github.com/golang/website.git", UpstreamCommit: tour.FrozenUpstreamCommit, SnapshotSHA256: digest(snapshot), Sources: sources,
		Versions: []Version{{"site-v1", []string{"tour-v1"}}, {"site-v2", []string{"tour-v1", "site-v2-shell", "learn-docs-v1"}}},
		Surfaces: []Surface{{"tour", "tour-v1", "/tour/**", "present.Section", "legacy-formal-local"}, {"homepage", "site-v2-shell", "/", UnitContract, "requires-independent-completion"}, {"translation", "site-v2-shell", "/translation/", UnitContract, "requires-independent-completion"}},
		Packages: []Package{{ID: "tour-v1", Activation: "legacy-evidence", Routes: []string{"/tour/", "/tour/list"}, Surfaces: []string{"tour"}}, {ID: "site-v2-shell", Activation: "independent-shell-gates", Routes: []string{"/", "/translation/"}, Surfaces: []string{"homepage", "translation"}}, {ID: "learn-docs-v1", Activation: "atomic-locale-package", Routes: []string{}, Surfaces: []string{"learn", "tutorial", "database", "modules", "security"}}},
	}
	for i, id := range g.Packages[2].Surfaces {
		g.Surfaces = append(g.Surfaces, Surface{id, "learn-docs-v1", strings.TrimPrefix(learnRoots[i], "_content") + "/**", UnitContract, "requires-independent-completion"})
	}
	for _, s := range ShellSources() {
		b, err := readRegular(root, s.Path)
		if err != nil {
			return nil, err
		}
		s.Origin = "repository"
		s.SHA256 = digest(b)
		s.Dependencies = []Dependency{}
		g.Sources = append(g.Sources, s)
	}
	for _, p := range catalog.Pages {
		// Page.Route remains the Tour-internal identity; scope owns its public projection.
		g.Packages[0].Routes = append(g.Packages[0].Routes, "/tour"+p.Route)
	}
	for _, s := range sources {
		if s.Kind == "page" {
			g.Packages[2].Routes = append(g.Packages[2].Routes, s.Route)
		}
	}
	files, err := listFiles(os.DirFS(root), "_content/tour")
	if err != nil {
		return nil, err
	}
	for _, p := range files {
		b, err := readRegular(root, p)
		if err != nil {
			return nil, err
		}
		kind := "asset"
		if path.Ext(p) == ".article" {
			kind = "page"
		}
		g.Sources = append(g.Sources, Source{Path: p, Kind: kind, Origin: "repository", Package: "tour-v1", Surface: "tour", SHA256: digest(b), Dependencies: []Dependency{}})
	}
	for i := range g.Packages {
		p := &g.Packages[i]
		if p.ID == "site-v2-shell" || p.ID == "learn-docs-v1" {
			p.ParserContract = UnitContract
		}
		sort.Strings(p.Routes)
		part := []Source{}
		for _, s := range g.Sources {
			if s.Package == p.ID {
				part = append(part, s)
			}
		}
		// Contract dependencies are part of learn/doc source authority, outside its exact 80-file scope.
		deps := []Dependency{}
		if p.ID == "learn-docs-v1" {
			for _, dp := range append(append([]string{}, contractPaths...), "_content/site.tmpl", "_content/article.tmpl", "_content/doc/default.tmpl") {
				b, err := fs.ReadFile(f, dp)
				if err != nil {
					return nil, err
				}
				deps = append(deps, Dependency{"contract", dp, digest(b)})
			}
		}
		p.Dependencies = deps
		p.SourceIdentity = identity(struct {
			Sources   []Source
			Contracts []Dependency
		}{part, deps})
		p.Identity = identity(*p)
	}
	g.Identity = identity(*g)
	if err := ValidateGlobal(*g); err != nil {
		return nil, err
	}
	return g, nil
}

// CompareInventory is a read-only, per-file diff primitive for the later V2-C
// preview. It neither maps ambiguous changes nor updates authority.
type Change struct {
	Path    string   `json:"path"`
	Changes []string `json:"changes"`
}

func CompareInventory(old, next []Source) []Change {
	a, b := map[string]Source{}, map[string]Source{}
	for _, s := range old {
		a[s.Path] = s
	}
	for _, s := range next {
		b[s.Path] = s
	}
	keys := map[string]bool{}
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	ordered := []string{}
	for k := range keys {
		ordered = append(ordered, k)
	}
	sort.Strings(ordered)
	result := []Change{}
	for _, k := range ordered {
		x, xok := a[k]
		y, yok := b[k]
		c := []string{}
		switch {
		case !xok:
			c = append(c, "added")
		case !yok:
			c = append(c, "removed")
		default:
			if x.SHA256 != y.SHA256 {
				c = append(c, "content_changed")
			}
			if x.Kind != y.Kind || x.Surface != y.Surface || x.Package != y.Package {
				c = append(c, "role_changed")
			}
			if x.Route != y.Route || x.Alias != y.Alias || x.Redirect != y.Redirect || x.ResolvedRedirect != y.ResolvedRedirect {
				c = append(c, "route_changed")
			}
			if identity(x.Dependencies) != identity(y.Dependencies) {
				c = append(c, "dependency_changed")
			}
			if len(c) == 0 {
				c = append(c, "unchanged")
			}
		}
		result = append(result, Change{k, c})
	}
	return result
}

func validateExternalDestination(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}
