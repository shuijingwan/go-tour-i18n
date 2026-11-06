// Package sourcestorage verifies repo-local frozen bytes and storage-only proofs.
package sourcestorage

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const ManifestPath = "data/site-source-manifest.json"
const ManifestSchema = "go-learning/repo-local-frozen-source/v1"
const ContractRoot = "data/site-source-contracts/"
const MigrationLock = "data/.site-source-migration"

type FrozenSourceFile struct {
	Path           string `json:"path"`
	UpstreamPath   string `json:"upstream_path,omitempty"`
	GitMode        string `json:"git_mode"`
	LocalMode      uint32 `json:"local_mode"`
	Role           string `json:"role"`
	Classification string `json:"classification"`
	Size           int64  `json:"size"`
	SHA256         string `json:"sha256"`
	UpstreamSHA256 string `json:"upstream_sha256,omitempty"`
	UpstreamBlob   string `json:"upstream_blob,omitempty"`
	ProjectCommit  string `json:"project_commit,omitempty"`
	ProjectBlob    string `json:"project_blob,omitempty"`
	Provenance     string `json:"provenance"`
}

type LegacySourceSubset struct {
	Path     string   `json:"path"`
	SHA256   string   `json:"sha256"`
	Encoding string   `json:"encoding"`
	Paths    []string `json:"paths"`
}

type FrozenSourceManifest struct {
	Schema     string             `json:"schema"`
	Repository string             `json:"upstream_repository"`
	Commit     string             `json:"upstream_commit"`
	Files      []FrozenSourceFile `json:"files"`
	Legacy     LegacySourceSubset `json:"registered_legacy_subset"`
	Identity   string             `json:"identity_sha256"`
}

// ReadSnapshot preserves the original registered projection and its exact
// digest, but reads adopted bytes from this repository. The ZIP is only a
// compatibility encoding in memory, never the current physical source.
// Historical fixtures without a migrated manifest retain their original reader.
func ReadSnapshot(root string) ([]byte, error) {
	if _, err := os.Lstat(pathJoin(root, MigrationLock)); !os.IsNotExist(err) {
		return nil, fmt.Errorf("source migration pending; mutation=unknown")
	}
	m, err := LoadManifest(root)
	if os.IsNotExist(err) {
		if _, err := os.Lstat(pathJoin(root, strings.TrimSuffix(ContractRoot, "/"))); err == nil {
			files, err := listFiles(os.DirFS(root), strings.TrimSuffix(ContractRoot, "/"))
			if err != nil || len(files) != 0 {
				return nil, fmt.Errorf("adopted source contracts exist without their manifest")
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		return readRegular(root, LegacyZIPPath)
	}
	if err != nil {
		return nil, err
	}
	return ReconstructSubset(root, m)
}

func LoadManifest(root string) (*FrozenSourceManifest, error) {
	b, err := readRegular(root, ManifestPath)
	if err != nil {
		return nil, err
	}
	var m FrozenSourceManifest
	if err := StrictJSON(b, &m); err != nil {
		return nil, err
	}
	if err := ValidateManifest(m); err != nil {
		return nil, err
	}
	canonical, err := Encode(m)
	if err != nil || !bytes.Equal(b, canonical) {
		return nil, fmt.Errorf("non-canonical frozen source manifest")
	}
	return &m, nil
}

func ValidateManifest(m FrozenSourceManifest) error {
	copy := m
	copy.Identity = ""
	if m.Schema != ManifestSchema || m.Repository != "https://github.com/golang/website.git" || m.Commit != UpstreamCommit || m.Identity != identity(copy) || len(m.Files) == 0 {
		return fmt.Errorf("frozen source manifest identity mismatch")
	}
	files := map[string]FrozenSourceFile{}
	previous := ""
	for _, f := range m.Files {
		if !validPath(f.Path) || !utf8.ValidString(f.Path) || !utf8.ValidString(f.UpstreamPath) || f.Path <= previous || f.Size < 0 || !validSHA(f.SHA256) || (f.GitMode != "100644" && f.GitMode != "100755") || f.LocalMode > 0777 || f.Provenance == "" {
			return fmt.Errorf("invalid/duplicate frozen source entry: %s", f.Path)
		}
		previous = f.Path
		if !strings.HasPrefix(f.Path, "_content/") && !strings.HasPrefix(f.Path, ContractRoot) {
			return fmt.Errorf("source outside adopted boundary: %s", f.Path)
		}
		switch f.Role {
		case "website-source", "website-data", "template", "asset", "runtime-script", "renderer-contract", "project-source", "project-runtime", "project-generated-default":
		default:
			return fmt.Errorf("unknown frozen source role")
		}
		switch f.Classification {
		case "upstream-exact", "project-override":
			if !validPath(f.UpstreamPath) || !validSHA(f.UpstreamSHA256) || !gitObjectID(f.UpstreamBlob) || (f.Path != f.UpstreamPath && f.Path != ContractRoot+f.UpstreamPath) {
				return fmt.Errorf("invalid upstream provenance: %s", f.Path)
			}
			if f.Classification == "upstream-exact" && (f.SHA256 != f.UpstreamSHA256 || f.ProjectCommit != "" || f.ProjectBlob != "") {
				return fmt.Errorf("exact source differs from upstream: %s", f.Path)
			}
			if f.Classification == "project-override" && (OverrideProvenance(f.Path) == "" || !gitObjectID(f.ProjectCommit) || !gitObjectID(f.ProjectBlob)) {
				return fmt.Errorf("unproven project override: %s", f.Path)
			}
		case "project-owned":
			if !strings.HasPrefix(f.Path, "_content/") || f.UpstreamPath != "" || f.UpstreamSHA256 != "" || f.UpstreamBlob != "" || !gitObjectID(f.ProjectCommit) || !gitObjectID(f.ProjectBlob) {
				return fmt.Errorf("invalid project source provenance: %s", f.Path)
			}
		default:
			return fmt.Errorf("unknown source classification")
		}
		files[f.Path] = f
	}
	if m.Legacy.Path != LegacyZIPPath || !validSHA(m.Legacy.SHA256) || m.Legacy.Encoding != "go-zip-deflate-0644-v1" || len(m.Legacy.Paths) == 0 {
		return fmt.Errorf("invalid legacy subset identity")
	}
	previous = ""
	for _, p := range m.Legacy.Paths {
		local := adoptedSourcePath(p)
		f, ok := files[local]
		if !validPath(p) || p <= previous || !ok || f.UpstreamPath != p || f.Classification != "upstream-exact" {
			return fmt.Errorf("invalid/duplicate/non-exact legacy subset member: %s", p)
		}
		previous = p
	}
	return nil
}

func gitObjectID(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func adoptedSourcePath(upstream string) string {
	if strings.HasPrefix(upstream, "_content/") {
		return upstream
	}
	return ContractRoot + upstream
}

func ReadAdoptedSource(root string, f FrozenSourceFile) ([]byte, error) {
	b, err := readRegular(root, f.Path)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(pathJoin(root, f.Path))
	if err != nil || uint32(info.Mode().Perm()) != f.LocalMode || int64(len(b)) != f.Size || digest(b) != f.SHA256 {
		return nil, fmt.Errorf("adopted frozen source changed: %s", f.Path)
	}
	return b, nil
}

func ReconstructSubset(root string, m *FrozenSourceManifest) ([]byte, error) {
	files := map[string]FrozenSourceFile{}
	for _, f := range m.Files {
		files[f.Path] = f
	}
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for _, p := range m.Legacy.Paths {
		b, err := ReadAdoptedSource(root, files[adoptedSourcePath(p)])
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
	if digest(out.Bytes()) != m.Legacy.SHA256 {
		return nil, fmt.Errorf("registered source subset cannot reconstruct exact legacy identity")
	}
	return out.Bytes(), nil
}

// CheckSources checks the complete adoption, while package currentness
// consumes only its registered subset. Unrelated mirror bytes are not a package
// freshness input and never become TranslationUnits or publication coverage.
func CheckSources(root string) (*FrozenSourceManifest, error) {
	m, err := LoadManifest(root)
	if err != nil {
		return nil, err
	}
	expected := map[string]bool{}
	for _, f := range m.Files {
		if _, err := ReadAdoptedSource(root, f); err != nil {
			return nil, err
		}
		expected[f.Path] = true
	}
	for _, dir := range []string{"_content", strings.TrimSuffix(ContractRoot, "/")} {
		err := filepath.WalkDir(pathJoin(root, dir), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink in adopted source: %s", p)
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil || !expected[filepath.ToSlash(rel)] {
				return fmt.Errorf("unregistered repo-local source: %s", p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if _, err := ReadSnapshot(root); err != nil {
		return nil, err
	}
	return m, nil
}

func SealManifest(m *FrozenSourceManifest) {
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	sort.Strings(m.Legacy.Paths)
	m.Identity = ""
	m.Identity = identity(*m)
}

const LegacyZIPPath = "data/site-content-sources.zip"
const UpstreamCommit = "db076098077c07d3cef1b85a2cf56ff52777f587"

func digest(b []byte) string { return contentidentity.SHA256Hex(b) }
func identity(v any) string  { b, _ := json.Marshal(v); return digest(b) }
func Encode(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	return append(b, '\n'), err
}
func StrictJSON(b []byte, v any) error { return contentidentity.StrictJSON(b, v) }
func validPath(p string) bool {
	return fs.ValidPath(p) && p != "." && !strings.ContainsAny(p, "\\\x00")
}
func validSHA(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && s == strings.ToLower(s)
}
func pathJoin(root, p string) string { return filepath.Join(root, filepath.FromSlash(p)) }
func readRegular(root, p string) ([]byte, error) {
	if !validPath(p) {
		return nil, fmt.Errorf("unsafe source storage path %q", p)
	}
	current := root
	parts := strings.Split(p, "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || (i < len(parts)-1 && !info.IsDir()) || (i == len(parts)-1 && !info.Mode().IsRegular()) {
			return nil, fmt.Errorf("unsafe source storage path %s", p)
		}
	}
	return os.ReadFile(current)
}
func listFiles(f fs.FS, dir string) ([]string, error) {
	files := []string{}
	err := fs.WalkDir(f, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("source symlink")
		}
		if !d.IsDir() {
			if !d.Type().IsRegular() {
				return fmt.Errorf("non-regular source")
			}
			files = append(files, p)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func OverrideProvenance(p string) string {
	switch p {
	case "_content/js/playground.js":
		return "已提交的 Tour HTTPTransport 与本地化 runtime message 接线"
	case "_content/tour/static/css/app.css":
		return "已提交的多语言、RTL、编辑器与移动端布局"
	case "_content/tour/static/js/app.js", "_content/tour/static/js/controllers.js", "_content/tour/static/js/directives.js", "_content/tour/static/js/page.js", "_content/tour/static/js/services.js", "_content/tour/static/js/values.js":
		return "已提交的 Tour i18n、SEO、publication 与导航 runtime 接线"
	case "_content/tour/static/partials/editor.html", "_content/tour/static/partials/list.html", "_content/tour/template/index.tmpl":
		return "已提交的本地化 Tour shell、编辑器与目录模板"
	}
	return ""
}
