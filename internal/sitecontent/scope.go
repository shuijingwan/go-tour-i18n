// Package sitecontent owns repository content scope, independently of deployment identity.
package sitecontent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const GlobalPath = "data/site-content-scope.json"
const SnapshotPath = "data/site-content-sources.zip"
const GlobalSchema = "go-learning/site-content-scope/v1"
const LocaleSchema = contentidentity.LocaleSchema
const PublicName = "Go Learning & Documentation Translations"

type Version struct {
	ID       string   `json:"id"`
	Packages []string `json:"packages"`
}
type Surface struct {
	ID          string `json:"id"`
	Package     string `json:"package"`
	RouteFamily string `json:"route_family"`
	Parser      string `json:"parser"`
	Publication string `json:"publication"`
}
type Package struct {
	ID             string       `json:"id"`
	Activation     string       `json:"activation"`
	Routes         []string     `json:"canonical_routes"`
	Surfaces       []string     `json:"surfaces"`
	Dependencies   []Dependency `json:"contract_dependencies"`
	SourceIdentity string       `json:"source_identity_sha256"`
	Identity       string       `json:"identity_sha256"`
	ParserContract string       `json:"parser_contract,omitempty"`
}
type Dependency struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Source struct {
	Path             string       `json:"source_path"`
	Kind             string       `json:"source_kind"`
	Origin           string       `json:"origin"`
	Package          string       `json:"package"`
	Surface          string       `json:"surface"`
	Route            string       `json:"canonical_route,omitempty"`
	Alias            string       `json:"redirect_alias,omitempty"`
	Redirect         string       `json:"redirect_destination,omitempty"`
	ResolvedRedirect string       `json:"resolved_redirect_destination,omitempty"`
	SHA256           string       `json:"source_sha256"`
	Dependencies     []Dependency `json:"dependencies"`
}
type Global struct {
	Schema             string    `json:"schema"`
	PublicName         string    `json:"public_name"`
	UpstreamRepository string    `json:"upstream_repository"`
	UpstreamCommit     string    `json:"upstream_commit"`
	SnapshotSHA256     string    `json:"snapshot_sha256"`
	Versions           []Version `json:"site_versions"`
	Surfaces           []Surface `json:"surfaces"`
	Packages           []Package `json:"packages"`
	Sources            []Source  `json:"source_inventory"`
	Identity           string    `json:"identity_sha256"`
}
type Reference = contentidentity.Reference
type Completion = contentidentity.Completion
type Locale = contentidentity.Locale

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func identity(v any) string  { b, _ := json.Marshal(v); return digest(b) }
func validPath(p string) bool {
	return fs.ValidPath(p) && p != "." && !bytes.ContainsAny([]byte(p), "\\\x00")
}

func validSHA(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && s == strings.ToLower(s)
}
func validRoute(s string) bool {
	return strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "//") && !strings.ContainsAny(s, "?#\\") && (path.Clean(s) == s || path.Clean(s)+"/" == s)
}

// StrictJSON also rejects duplicate object members; encoding/json alone accepts them.
func StrictJSON(b []byte, v any) error { return contentidentity.StrictJSON(b, v) }

func readRegular(root, p string) ([]byte, error) {
	if !validPath(p) {
		return nil, fmt.Errorf("unsafe repository path %q", p)
	}
	current := root
	for _, part := range splitPath(p) {
		current = filepath.Join(current, part)
		st, err := os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink forbidden: %s", p)
		}
	}
	st, err := os.Stat(current)
	if err != nil || !st.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", p)
	}
	return os.ReadFile(current)
}
func splitPath(p string) []string {
	var result []string
	for p != "." {
		result = append([]string{path.Base(p)}, result...)
		p = path.Dir(p)
	}
	return result
}

// ValidateGlobal validates static schema and relational identity before current checks.
func ValidateGlobal(g Global) error {
	if g.Schema != GlobalSchema || g.PublicName != PublicName || g.UpstreamRepository != "https://github.com/golang/website.git" || g.UpstreamCommit != UpstreamCommit || !validSHA(g.SnapshotSHA256) || len(g.Packages) < 3 || len(g.Versions) != 2 {
		return fmt.Errorf("incompatible global content scope")
	}
	packages := map[string]Package{}
	if g.Versions[0].ID != "site-v1" || !reflect.DeepEqual(g.Versions[0].Packages, []string{"tour-v1"}) || g.Versions[1].ID != "site-v2" {
		return fmt.Errorf("unknown site version/package contract")
	}
	surfaces := map[string]Surface{}
	routes := map[string]bool{}
	for _, s := range g.Surfaces {
		if s.ID == "" || surfaces[s.ID].ID != "" {
			return fmt.Errorf("duplicate/empty surface")
		}
		surfaces[s.ID] = s
	}
	for _, p := range g.Packages {
		if p.ID == "" || packages[p.ID].ID != "" {
			return fmt.Errorf("duplicate/empty package")
		}
		packages[p.ID] = p
		if packageWorkflow(p.ID) && p.ParserContract != UnitContract {
			return fmt.Errorf("unknown package parser contract: %s", p.ID)
		}
		for _, r := range p.Routes {
			if !validRoute(r) || routes[r] {
				return fmt.Errorf("duplicate/invalid canonical route %q", r)
			}
			routes[r] = true
		}
		for _, id := range p.Surfaces {
			if surfaces[id].Package != p.ID {
				return fmt.Errorf("unknown or wrong package surface %s", id)
			}
		}
		for _, d := range p.Dependencies {
			if d.Kind != "contract" || !validPath(d.Path) || !validSHA(d.SHA256) {
				return fmt.Errorf("invalid package contract dependency")
			}
		}
		copy := p
		copy.Identity = ""
		if p.Identity != identity(copy) {
			return fmt.Errorf("package identity mismatch: %s", p.ID)
		}
	}
	for _, s := range g.Surfaces {
		if packages[s.Package].ID == "" {
			return fmt.Errorf("unknown surface package")
		}
	}
	for _, v := range g.Versions {
		members := map[string]bool{}
		for _, id := range v.Packages {
			if packages[id].ID == "" || members[id] {
				return fmt.Errorf("unknown version package")
			}
			members[id] = true
		}
		if v.ID == "site-v2" {
			for id := range packages {
				if !members[id] {
					return fmt.Errorf("package absent from site-v2: %s", id)
				}
			}
			for _, id := range []string{"tour-v1", "site-v2-shell", "learn-docs-v1"} {
				if !members[id] {
					return fmt.Errorf("missing required package %s", id)
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, s := range g.Sources {
		if !validPath(s.Path) || !validSHA(s.SHA256) || (s.Origin != "repository" && s.Origin != "frozen-upstream") || seen[s.Path] || surfaces[s.Surface].Package != s.Package {
			return fmt.Errorf("invalid source membership/path %s", s.Path)
		}
		seen[s.Path] = true
		switch s.Kind {
		case "page":
			if s.Package == "learn-docs-v1" && !routes[s.Route] {
				return fmt.Errorf("page without canonical route %s", s.Path)
			}
		case "data", "asset":
			if s.Route != "" {
				return fmt.Errorf("data/asset cannot be canonical page")
			}
		case "redirect":
			if s.Route != "" || !validRoute(s.Alias) || s.Redirect == "" || s.ResolvedRedirect == "" || routes[s.Alias] {
				return fmt.Errorf("redirect masquerades as page %s", s.Path)
			}
		default:
			return fmt.Errorf("unknown source kind")
		}
		for _, d := range s.Dependencies {
			if !validPath(d.Path) || !validSHA(d.SHA256) || (d.Kind != "data" && d.Kind != "asset" && d.Kind != "contract") {
				return fmt.Errorf("invalid dependency %s", d.Path)
			}
		}
	}
	copy := g
	copy.Identity = ""
	if g.Identity != identity(copy) {
		return fmt.Errorf("global scope identity mismatch")
	}
	return nil
}

func LoadCurrent(root string) (*Global, error) {
	b, err := readRegular(root, GlobalPath)
	if err != nil {
		return nil, err
	}
	var g Global
	if err := StrictJSON(b, &g); err != nil {
		return nil, err
	}
	if err := ValidateGlobal(g); err != nil {
		return nil, err
	}
	snapshot, err := readRegular(root, SnapshotPath)
	if err != nil {
		return nil, err
	}
	want, err := BuildGlobal(root, snapshot)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(g, *want) {
		return nil, fmt.Errorf("content scope STALE: source inventory, roles, routes, dependencies or package contract changed")
	}
	return &g, nil
}

func packageByID(g *Global, id string) (Package, error) {
	for _, p := range g.Packages {
		if p.ID == id {
			return p, nil
		}
	}
	return Package{}, fmt.Errorf("unknown package %s", id)
}
func LocalePath(locale string) string { return "locales/" + locale + "/content-scope.json" }

// Missing sparse authority is incomplete, never a completion claim. The locale
// itself must already have its existing canonical identity; no second init.
func readLocaleAuthority(root string, g *Global, locale string) (*Locale, error) {
	if err := i18n.ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	b, err := readRegular(root, LocalePath(locale))
	if os.IsNotExist(err) {
		identityBytes, err := readRegular(root, "locales/"+locale+"/locale.json")
		if err != nil {
			return nil, err
		}
		if err := i18n.ValidateLocaleIdentityBytes(locale, identityBytes); err != nil {
			return nil, err
		}
		return &Locale{Schema: LocaleSchema, Locale: locale, Packages: []Completion{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var l Locale
	if err := StrictJSON(b, &l); err != nil {
		return nil, err
	}
	if l.Locale != locale {
		return nil, fmt.Errorf("locale scope identity mismatch")
	}
	if err := ValidateLocale(g, l); err != nil {
		return nil, err
	}
	return &l, nil
}

func ValidateLocale(g *Global, l Locale) error {
	if l.Schema != LocaleSchema || !validPath(l.Locale) || path.Base(l.Locale) != l.Locale || l.Packages == nil {
		return fmt.Errorf("invalid locale content scope")
	}
	seen := map[string]bool{}
	for _, c := range l.Packages {
		p, err := packageByID(g, c.Package)
		if err != nil {
			return err
		}
		if seen[c.Package] {
			return fmt.Errorf("duplicate locale package")
		}
		seen[c.Package] = true
		if c.PackageIdentity != p.Identity || c.SourceIdentity != p.SourceIdentity {
			return fmt.Errorf("locale package scope STALE: %s", c.Package)
		}
		switch c.State {
		case "complete":
			if packageWorkflow(c.Package) {
				if c.EvidenceKind != PackageCompletionKind || c.LegacySurfaceState != "" || len(c.Evidence) != 2 || !validSHA(c.ContextIdentity) || c.EvidenceIdentity != identity(c.Evidence) || !reflect.DeepEqual(c.Routes, p.Routes) || !reflect.DeepEqual(c.Surfaces, p.Surfaces) || !reflect.DeepEqual(c.RouteFamilies, routeFamilies(c.Package)) {
					return fmt.Errorf("unsupported/unproven atomic package completion")
				}
				for _, r := range c.Evidence {
					if !validPath(r.Path) || !validSHA(r.SHA256) {
						return fmt.Errorf("invalid completion evidence reference")
					}
				}
				continue
			}
			// V2-A only admits the verified existing Tour closure; future activation requires new gates.
			if c.Package != "tour-v1" || c.EvidenceKind != "legacy-tour-closure/v1" || (c.LegacySurfaceState != "current" && c.LegacySurfaceState != "historical-verified") || len(c.Evidence) == 0 || !validSHA(c.ContextIdentity) || c.EvidenceIdentity != identity(c.Evidence) || !reflect.DeepEqual(c.Routes, p.Routes) || !reflect.DeepEqual(c.Surfaces, p.Surfaces) || !reflect.DeepEqual(c.RouteFamilies, []string{"/tour/**"}) {
				return fmt.Errorf("unsupported/unproven completion")
			}
			last := ""
			for _, r := range c.Evidence {
				if !validPath(r.Path) || !validSHA(r.SHA256) || r.Path <= last {
					return fmt.Errorf("invalid completion evidence reference")
				}
				last = r.Path
			}
		default:
			return fmt.Errorf("sparse authority stores only complete packages; omit incomplete packages")
		}
	}
	return nil
}

// PackageStates projects missing completion records as incomplete without
// binding unfinished work to global source/package identities.
func PackageStates(g *Global, l *Locale) []Completion {
	completed := map[string]Completion{}
	for _, c := range l.Packages {
		completed[c.Package] = c
	}
	states := []Completion{}
	for _, p := range g.Packages {
		c, ok := completed[p.ID]
		if !ok {
			c = Completion{Package: p.ID, State: "incomplete"}
		}
		states = append(states, c)
	}
	return states
}

func sortedReferences(root string, paths []string) ([]Reference, error) {
	sort.Strings(paths)
	refs := []Reference{}
	last := ""
	for _, p := range paths {
		if p == last {
			continue
		}
		last = p
		b, err := readRegular(root, p)
		if err != nil {
			return nil, err
		}
		refs = append(refs, Reference{p, digest(b)})
	}
	return refs, nil
}

// WriteNew never overwrites authority or historical evidence.
func WriteNew(root, p string, b []byte) error {
	if !validPath(p) {
		return fmt.Errorf("unsafe output")
	}
	current := root
	for _, part := range splitPath(path.Dir(p)) {
		current = filepath.Join(current, part)
		st, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe output parent %s", p)
		}
	}
	f, err := os.OpenFile(filepath.Join(root, filepath.FromSlash(p)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func Encode(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	return append(b, '\n'), err
}
