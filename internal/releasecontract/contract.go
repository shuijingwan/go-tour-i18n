// Package releasecontract defines the sealed content contract consumed by the
// existing build, publish, runtime, deployment and verification adapters.
package releasecontract

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const Schema = "go-learning/release-content/v1"
const Path = "site/release-content.json"
const BrowserPlanPath = "site/browser-plan.json"
const AssetPolicy = "shared-cloudflare"

type Manifest struct {
	SchemaVersion        int    `json:"schema_version"`
	Locale               string `json:"locale"`
	PublishedAt          string `json:"published_at"`
	UpstreamCommit       string `json:"upstream_commit"`
	UpstreamCommitTime   string `json:"upstream_commit_time"`
	TranslationUnits     int    `json:"translation_units"`
	Pages                int    `json:"pages"`
	EligibleExamples     int    `json:"eligible_examples"`
	Articles             int    `json:"articles"`
	ExecutionTransport   string `json:"execution_transport"`
	ExecutionProvider    string `json:"execution_provider"`
	LocalSocketEnabled   bool   `json:"local_socket_enabled"`
	GOOS                 string `json:"goos"`
	GOARCH               string `json:"goarch"`
	SiteIdentity         string `json:"site_identity"`
	BuildIdentity        string `json:"build_identity"`
	InputBuildSHA256     string `json:"input_build_sha256"`
	SharedAssetsIdentity string `json:"shared_assets_identity"`
}

func VerifyBundle(root string) (string, *Artifact, error) {
	id, e := VerifyChecksums(root)
	if e != nil {
		return "", nil, e
	}
	entries, e := os.ReadDir(root)
	if e != nil {
		return "", nil, e
	}
	want := map[string]bool{"bin": true, "_content": true, "release.json": true, "SHA256SUMS": true}
	if len(entries) != len(want) {
		return "", nil, fmt.Errorf("non-exact production root")
	}
	for _, x := range entries {
		if !want[x.Name()] {
			return "", nil, fmt.Errorf("unexpected bundle root entry")
		}
	}
	bin, err := os.ReadDir(filepath.Join(root, "bin"))
	if err != nil || len(bin) != 1 || bin[0].Name() != "tour" {
		return "", nil, fmt.Errorf("non-exact production binary set")
	}
	i, e := os.Stat(filepath.Join(root, "bin", "tour"))
	if e != nil || !i.Mode().IsRegular() || i.Mode()&0111 == 0 {
		return "", nil, fmt.Errorf("production binary must be regular executable")
	}
	a, e := Read(os.DirFS(filepath.Join(root, "_content")))
	if e != nil {
		return "", nil, e
	}
	b, e := ReadRegular(root, "release.json")
	if e != nil {
		return "", nil, e
	}
	var m Manifest
	if e = contentidentity.StrictJSON(b, &m); e != nil {
		return "", nil, e
	}
	if m.SchemaVersion != 3 || m.Locale != a.Locale || m.SiteIdentity != a.Identity || m.BuildIdentity != a.Identity || !ValidSHA(m.InputBuildSHA256) || m.SharedAssetsIdentity != a.SharedAssetsIdentity || m.GOOS != "linux" || m.GOARCH != "amd64" || m.LocalSocketEnabled || m.ExecutionTransport != "http-playground-proxy" || m.Pages < 1 || m.Articles < 1 || m.EligibleExamples < 0 || m.TranslationUnits != m.Pages+m.EligibleExamples {
		return "", nil, fmt.Errorf("production manifest contract mismatch")
	}
	return id, a, nil
}

type Artifact struct {
	Schema               string            `json:"schema"`
	Identity             string            `json:"identity"`
	Locale               string            `json:"locale"`
	Origin               string            `json:"origin"`
	CoverageIdentity     string            `json:"coverage_identity"`
	Routes               []string          `json:"canonical_routes"`
	Families             []string          `json:"families"`
	Aliases              map[string]string `json:"aliases"`
	Pages                map[string]string `json:"pages"`
	Inputs               map[string]string `json:"inputs"`
	Files                map[string]string `json:"files"`
	AssetPolicy          string            `json:"asset_policy"`
	AssetOrigin          string            `json:"asset_origin"`
	SharedAssets         map[string]string `json:"shared_assets"`
	SharedAssetsIdentity string            `json:"shared_assets_identity"`
}

func Digest(b []byte) string  { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func Identity(v any) string   { b, _ := json.Marshal(v); return Digest(b) }
func (a Artifact) ID() string { a.Identity = ""; return Identity(a) }
func ValidSHA(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == sha256.Size && s == strings.ToLower(s)
}
func SafePath(p string) bool {
	return fs.ValidPath(p) && p != "." && !strings.ContainsAny(p, "\\\x00\n\r\t")
}
func ValidRoute(p string) bool {
	return strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "//") && !strings.ContainsAny(p, "?#\\\x00") && (path.Clean(p) == p || path.Clean(p)+"/" == p)
}

func (a Artifact) Validate() error {
	u, e := url.Parse(a.Origin)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("invalid release origin")
	}
	asset, e := url.Parse(a.AssetOrigin)
	if e != nil || asset.Scheme != "https" || asset.Host == "" || asset.User != nil || asset.Path != "" || asset.RawQuery != "" || asset.Fragment != "" || a.AssetPolicy != AssetPolicy {
		return fmt.Errorf("invalid release asset policy/origin")
	}
	if a.Schema != Schema || !SafePath(a.Locale) || strings.Contains(a.Locale, "/") || !ValidSHA(a.CoverageIdentity) || a.Identity != a.ID() || len(a.Routes) == 0 || len(a.Families) == 0 || a.Aliases == nil || a.Pages == nil || len(a.Inputs) == 0 || len(a.Files) == 0 || len(a.SharedAssets) == 0 || a.SharedAssetsIdentity != Identity(a.SharedAssets) {
		return fmt.Errorf("incomplete/unsealed release content contract")
	}
	last := ""
	for _, r := range a.Routes {
		if !ValidRoute(r) || r <= last {
			return fmt.Errorf("non-exact canonical route set")
		}
		last = r
		if !strings.HasPrefix(r, "/tour/") && a.Pages[r] == "" {
			return fmt.Errorf("covered Site route missing rendered bytes: %s", r)
		}
	}
	for r, p := range a.Pages {
		if !contains(a.Routes, r) || strings.HasPrefix(r, "/tour/") || !SafePath(p) || !strings.HasPrefix(p, "site/pages/") || a.Files[p] == "" {
			return fmt.Errorf("uncovered/unsafe rendered page %s", r)
		}
	}
	for from, to := range a.Aliases {
		if !ValidRoute(from) || !contains(a.Routes, to) {
			return fmt.Errorf("alias lacks covered local target")
		}
	}
	for _, m := range []map[string]string{a.Inputs, a.Files, a.SharedAssets} {
		for p, h := range m {
			if !SafePath(p) || !ValidSHA(h) || strings.Contains(p, "site-shell/home.html") {
				return fmt.Errorf("unsafe/retired artifact input: %s", p)
			}
		}
	}
	for p, h := range a.SharedAssets {
		if a.Files[p] != h {
			return fmt.Errorf("shared dependency differs from sealed payload: %s", p)
		}
	}
	if a.Files[BrowserPlanPath] == "" {
		return fmt.Errorf("sealed browser plan required")
	}
	return nil
}
func contains(ss []string, s string) bool {
	i := sort.SearchStrings(ss, s)
	return i < len(ss) && ss[i] == s
}

func Read(f fs.FS) (*Artifact, error) {
	b, e := fs.ReadFile(f, Path)
	if e != nil {
		return nil, e
	}
	var a Artifact
	if e = contentidentity.StrictJSON(b, &a); e != nil {
		return nil, e
	}
	if e = a.Validate(); e != nil {
		return nil, e
	}
	for p, h := range a.Files {
		b, e := fs.ReadFile(f, p)
		if e != nil || Digest(b) != h {
			return nil, fmt.Errorf("sealed content changed: %s", p)
		}
	}
	actual := map[string]string{}
	e = fs.WalkDir(f, ".", func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if p == "." || d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("unsupported content entry")
		}
		if p == Path || p == "tour/site-metadata.json" || strings.HasPrefix(p, "tour/prerender/") {
			return nil
		}
		b, e := fs.ReadFile(f, p)
		if e != nil {
			return e
		}
		actual[p] = Digest(b)
		return nil
	})
	if e != nil {
		return nil, e
	}
	if !reflect.DeepEqual(actual, a.Files) {
		return nil, fmt.Errorf("content exact-set mismatch")
	}
	planBytes, e := fs.ReadFile(f, BrowserPlanPath)
	if e != nil {
		return nil, e
	}
	var plan struct {
		Schema   string   `json:"schema"`
		Locale   string   `json:"locale"`
		Families []string `json:"families"`
		Routes   []string `json:"canonical_routes"`
		Pages    []struct {
			Route  string `json:"route"`
			Family string `json:"family"`
			Format string `json:"format"`
			Source string `json:"source"`
			HTML   string `json:"html"`
		} `json:"pages"`
	}
	if e = contentidentity.StrictJSON(planBytes, &plan); e != nil {
		return nil, e
	}
	if plan.Schema != "go-learning/preview-browser-plan/v1" || plan.Locale != a.Locale || !reflect.DeepEqual(plan.Routes, a.Routes) || !reflect.DeepEqual(plan.Families, a.Families) || len(plan.Pages) != len(a.Pages) {
		return nil, fmt.Errorf("sealed browser plan/coverage drift")
	}
	seen := map[string]bool{}
	for _, p := range plan.Pages {
		if a.Pages[p.Route] == "" || seen[p.Route] || !SafePath(p.Source) || strings.Contains(p.Source, "site-shell/home.html") || p.HTML == "" {
			return nil, fmt.Errorf("invalid sealed browser page")
		}
		seen[p.Route] = true
	}
	return &a, nil
}
func Write(root string, a *Artifact) error {
	a.Identity = a.ID()
	if e := a.Validate(); e != nil {
		return e
	}
	b, e := json.MarshalIndent(a, "", "  ")
	if e != nil {
		return e
	}
	p := filepath.Join(root, filepath.FromSlash(Path))
	if e = os.MkdirAll(filepath.Dir(p), 0755); e != nil {
		return e
	}
	return os.WriteFile(p, append(b, '\n'), 0644)
}
func Current(root string, a *Artifact) error {
	if e := a.Validate(); e != nil {
		return e
	}
	for p, h := range a.Inputs {
		b, e := ReadRegular(root, p)
		if e != nil || Digest(b) != h {
			return fmt.Errorf("release input STALE: %s", p)
		}
	}
	return nil
}
func ReadRegular(root, p string) ([]byte, error) {
	if !SafePath(p) {
		return nil, fmt.Errorf("unsafe path %s", p)
	}
	cur := root
	for _, part := range strings.Split(p, "/") {
		cur = filepath.Join(cur, part)
		i, e := os.Lstat(cur)
		if e != nil {
			return nil, e
		}
		if i.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink input %s", p)
		}
	}
	i, e := os.Stat(cur)
	if e != nil {
		return nil, e
	}
	if !i.Mode().IsRegular() {
		return nil, fmt.Errorf("non-regular input %s", p)
	}
	return os.ReadFile(cur)
}

// Checksums validates all paths before opening any file and checks the exact
// actual set; a self-consistent subset or traversal manifest is never accepted.
func Checksums(root string) ([]byte, error) {
	var paths []string
	i, e := os.Lstat(root)
	if e != nil {
		return nil, e
	}
	if !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("real artifact root required")
	}
	e = filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if p == root || d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("unsupported artifact entry %s", p)
		}
		r, e := filepath.Rel(root, p)
		if e != nil {
			return e
		}
		r = filepath.ToSlash(r)
		if !SafePath(r) || strings.Contains(r, "site-shell/home.html") {
			return fmt.Errorf("unsafe/retired artifact path %s", r)
		}
		if r != "SHA256SUMS" {
			paths = append(paths, r)
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	sort.Strings(paths)
	var out bytes.Buffer
	for _, p := range paths {
		b, e := ReadRegular(root, p)
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&out, "%s  %s\n", Digest(b), p)
	}
	return out.Bytes(), nil
}
func VerifyChecksums(root string) (string, error) {
	want, e := Checksums(root)
	if e != nil {
		return "", e
	}
	got, e := ReadRegular(root, "SHA256SUMS")
	if e != nil {
		return "", e
	}
	if !bytes.Equal(got, want) {
		return "", fmt.Errorf("artifact checksum exact-set or bytes mismatch")
	}
	return Digest(got), nil
}
func SameCoverage(a, b *Artifact) bool {
	return a.CoverageIdentity == b.CoverageIdentity && reflect.DeepEqual(a.Routes, b.Routes) && a.Origin == b.Origin && a.SharedAssetsIdentity == b.SharedAssetsIdentity
}
