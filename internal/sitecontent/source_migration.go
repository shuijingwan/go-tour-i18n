package sitecontent

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
)

var frozenRendererContracts = []string{
	"cmd/golangorg/godev.go", "cmd/golangorg/server.go",
	"internal/redirect/redirect.go", "internal/tmplfunc/func.go", "internal/tmplfunc/tmpl.go",
	"internal/web/page.go", "internal/web/pkg.go", "internal/web/render.go", "internal/web/site.go", "internal/web/tmpl.go",
}

// Explicit, already committed project consumers. Never infer an override merely
// from a differing hash or a directory name. Import also proves exact HEAD bytes.
func intentionalSourceOverride(p string) string { return sourcestorage.OverrideProvenance(p) }
func projectSourceRole(p string) string {
	switch p {
	case "_content/tour/site-metadata.json":
		return "project-generated-default"
	case "_content/tour/static/js/support.js", "_content/tour/static/go-dev/course-ad.js", "_content/tour/static/go-dev/course-ad.css":
		return "project-runtime"
	default:
		return "project-source"
	}
}

func sourceRole(p string) string {
	if !strings.HasPrefix(p, "_content/") {
		return "renderer-contract"
	}
	switch path.Ext(p) {
	case ".md", ".html":
		return "website-source"
	case ".tmpl":
		return "template"
	case ".yaml", ".json":
		return "website-data"
	case ".js", ".ts", ".css":
		return "runtime-script"
	}
	return "asset"
}

type SourceMigrationResult struct {
	ManifestPath       string `json:"manifest_path"`
	ManifestIdentity   string `json:"manifest_identity"`
	UpstreamFilesAdded int    `json:"upstream_files_added"`
	UpstreamBytesAdded int64  `json:"upstream_bytes_added"`
	PathCollisions     int    `json:"path_collisions"`
	ExactCollisions    int    `json:"exact_same_collisions"`
	OverrideCollisions int    `json:"intentional_override_collisions"`
	UnknownCollisions  int    `json:"unknown_collisions"`
	AlreadyApplied     bool   `json:"already_applied"`
	Applied            bool   `json:"applied"`
	ZIPRetirement      string `json:"zip_retirement"`
}

type sourceMigrationPlan struct {
	manifest FrozenSourceManifest
	result   SourceMigrationResult
	missing  map[string][]byte
	baseline map[string][]byte
	global   Global
}

func sourceGit(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-c", "core.fsmonitor=false", "-C", root}, args...)...)
	return cmd.Output()
}

func frozenGitFiles(websiteRoot string) (map[string][]byte, map[string]string, map[string]string, error) {
	paths := append([]string{"_content"}, frozenRendererContracts...)
	args := append([]string{"ls-tree", "-r", "-z", UpstreamCommit, "--"}, paths...)
	tree, err := sourceGit(websiteRoot, args...)
	if err != nil {
		return nil, nil, nil, err
	}
	modes, blobs := map[string]string{}, map[string]string{}
	for _, line := range bytes.Split(tree, []byte{0}) {
		if len(line) == 0 {
			continue
		}
		pair := strings.SplitN(string(line), "\t", 2)
		if len(pair) != 2 || !validPath(pair[1]) {
			return nil, nil, nil, fmt.Errorf("unsafe Git tree path")
		}
		fields := strings.Fields(pair[0])
		if len(fields) != 3 || fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") || modes[pair[1]] != "" {
			return nil, nil, nil, fmt.Errorf("unsupported/duplicate frozen Git entry: %s", pair[1])
		}
		modes[pair[1]], blobs[pair[1]] = fields[0], fields[2]
	}
	for _, p := range frozenRendererContracts {
		if modes[p] == "" {
			return nil, nil, nil, fmt.Errorf("missing frozen renderer contract: %s", p)
		}
	}
	cmd := exec.Command("git", append([]string{"-c", "core.fsmonitor=false", "-C", websiteRoot, "archive", "--format=tar", UpstreamCommit, "--"}, paths...)...)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}
	defer pipe.Close()
	files := map[string][]byte{}
	reader := tar.NewReader(pipe)
	for {
		h, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return nil, nil, nil, err
		}
		if h.Typeflag == tar.TypeDir {
			continue
		}
		if h.Typeflag == tar.TypeXGlobalHeader {
			if err := verifyFrozenGitArchiveMetadata(h); err != nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				return nil, nil, nil, err
			}
			continue
		}
		if h.Typeflag != tar.TypeReg || modes[h.Name] == "" || files[h.Name] != nil || h.Size > 256<<20 || h.Size < 0 {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return nil, nil, nil, fmt.Errorf("unsafe archive member: %s", h.Name)
		}
		b, err := io.ReadAll(reader)
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return nil, nil, nil, err
		}
		blobHash := sha1.New()
		fmt.Fprintf(blobHash, "blob %d\x00", len(b))
		blobHash.Write(b)
		if hex.EncodeToString(blobHash.Sum(nil)) != blobs[h.Name] {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return nil, nil, nil, fmt.Errorf("archive bytes differ from exact Git object: %s", h.Name)
		}
		files[h.Name] = b
	}
	if err := cmd.Wait(); err != nil {
		return nil, nil, nil, err
	}
	if len(files) != len(modes) {
		return nil, nil, nil, fmt.Errorf("Git archive/tree membership differs")
	}
	return files, modes, blobs, nil
}

func verifyFrozenGitArchiveMetadata(h *tar.Header) error {
	if h.Typeflag != tar.TypeXGlobalHeader || h.Name != "pax_global_header" || len(h.PAXRecords) != 1 || h.PAXRecords["comment"] != UpstreamCommit {
		return fmt.Errorf("Git archive metadata does not bind exact frozen commit")
	}
	return nil
}

func planSourceMigration(root, websiteRoot string) (*sourceMigrationPlan, error) {
	if _, err := os.Lstat(pathJoin(root, SourceManifestPath)); !os.IsNotExist(err) {
		return nil, fmt.Errorf("source manifest already exists or is unsafe")
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	files, modes, blobs, err := frozenGitFiles(websiteRoot)
	if err != nil {
		return nil, err
	}
	commit, err := sourceGit(root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	p := &sourceMigrationPlan{manifest: FrozenSourceManifest{Schema: sourceManifestSchema, Repository: g.UpstreamRepository, Commit: UpstreamCommit, Files: []FrozenSourceFile{}}, result: SourceMigrationResult{ManifestPath: SourceManifestPath, ZIPRetirement: "RETAINED_BLOCKED"}, missing: map[string][]byte{}, baseline: map[string][]byte{}, global: *g}
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, upstream := range keys {
		b := files[upstream]
		name := adoptedSourcePath(upstream)
		mode := uint32(0644)
		if modes[upstream] == "100755" {
			mode = 0755
		}
		f := FrozenSourceFile{Path: name, UpstreamPath: upstream, GitMode: modes[upstream], LocalMode: mode, Role: sourceRole(upstream), Classification: "upstream-exact", Size: int64(len(b)), SHA256: digest(b), UpstreamSHA256: digest(b), UpstreamBlob: blobs[upstream], Provenance: "exact frozen Git object"}
		current, err := readRegular(root, name)
		if os.IsNotExist(err) {
			p.missing[name] = b
			p.result.UpstreamFilesAdded++
			p.result.UpstreamBytesAdded += int64(len(b))
		} else if err != nil {
			return nil, err
		} else {
			p.baseline[name] = current
			p.result.PathCollisions++
			info, _ := os.Lstat(pathJoin(root, name))
			f.LocalMode, err = sourcestorage.RepositoryFileMode(info.Mode())
			if err != nil {
				return nil, err
			}
			if bytes.Equal(current, b) {
				p.result.ExactCollisions++
			} else {
				note := intentionalSourceOverride(name)
				project, gitErr := sourceGit(root, "show", "HEAD:"+name)
				if note == "" || gitErr != nil || !bytes.Equal(current, project) {
					return nil, fmt.Errorf("unknown collision: %s; mutation=unknown; no files written", name)
				}
				blob, err := sourceGit(root, "rev-parse", "HEAD:"+name)
				if err != nil {
					return nil, err
				}
				f.Classification, f.Role, f.Size, f.SHA256 = "project-override", "project-runtime", int64(len(current)), digest(current)
				f.ProjectCommit, f.ProjectBlob, f.Provenance = strings.TrimSpace(string(commit)), strings.TrimSpace(string(blob)), note
				p.result.OverrideCollisions++
			}
		}
		p.manifest.Files = append(p.manifest.Files, f)
	}
	currentFiles, err := listFiles(os.DirFS(root), "_content")
	if err != nil {
		return nil, err
	}
	for _, name := range currentFiles {
		if files[name] != nil {
			continue
		}
		b, err := readRegular(root, name)
		if err != nil {
			return nil, err
		}
		project, err := sourceGit(root, "show", "HEAD:"+name)
		if err != nil || !bytes.Equal(b, project) {
			return nil, fmt.Errorf("unproven project-owned source: %s; no files written", name)
		}
		blob, err := sourceGit(root, "rev-parse", "HEAD:"+name)
		if err != nil {
			return nil, err
		}
		info, _ := os.Lstat(pathJoin(root, name))
		tree, err := sourceGit(root, "ls-tree", "HEAD", "--", name)
		if err != nil || len(strings.Fields(string(tree))) < 3 {
			return nil, fmt.Errorf("project Git mode missing: %s", name)
		}
		mode := strings.Fields(string(tree))[0]
		localMode, err := sourcestorage.RepositoryFileMode(info.Mode())
		if err != nil {
			return nil, err
		}
		p.baseline[name] = b
		p.manifest.Files = append(p.manifest.Files, FrozenSourceFile{Path: name, GitMode: mode, LocalMode: localMode, Role: projectSourceRole(name), Classification: "project-owned", Size: int64(len(b)), SHA256: digest(b), ProjectCommit: strings.TrimSpace(string(commit)), ProjectBlob: strings.TrimSpace(string(blob)), Provenance: "existing committed project source; retained without mutation"})
	}
	legacy, err := readRegular(root, SnapshotPath)
	if err != nil || digest(legacy) != g.SnapshotSHA256 {
		return nil, fmt.Errorf("legacy snapshot binding changed")
	}
	z, err := zip.NewReader(bytes.NewReader(legacy), int64(len(legacy)))
	if err != nil {
		return nil, err
	}
	p.manifest.Legacy = LegacySourceSubset{Path: SnapshotPath, SHA256: digest(legacy), Encoding: "go-zip-deflate-0644-v1", Paths: []string{}}
	for _, entry := range z.File {
		r, err := entry.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil || !bytes.Equal(b, files[entry.Name]) {
			return nil, fmt.Errorf("legacy subset does not match adopted upstream: %s", entry.Name)
		}
		p.manifest.Legacy.Paths = append(p.manifest.Legacy.Paths, entry.Name)
	}
	sealSourceManifest(&p.manifest)
	if err := validateSourceManifest(p.manifest); err != nil {
		return nil, err
	}
	p.result.ManifestIdentity = p.manifest.Identity
	return p, nil
}

// MigrateFrozenSources is offline, no-overwrite, and preserves all existing
// content. Pending/interrupted mutation is explicit and cannot authorize source
// readers. It is not upstream sync, language generation or package activation.
func MigrateFrozenSources(root, websiteRoot string, apply bool) (*SourceMigrationResult, error) {
	if _, err := os.Lstat(pathJoin(root, sourceMigrationLock)); !os.IsNotExist(err) {
		return nil, fmt.Errorf("source migration pending; mutation=unknown")
	}
	if _, err := os.Lstat(pathJoin(root, SourceManifestPath)); err == nil {
		m, err := CheckFrozenSources(root)
		if err != nil {
			return nil, err
		}
		if _, err := LoadCurrent(root); err != nil {
			return nil, err
		}
		return &SourceMigrationResult{ManifestPath: SourceManifestPath, ManifestIdentity: m.Identity, AlreadyApplied: true, ZIPRetirement: "RETAINED_BLOCKED"}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	p, err := planSourceMigration(root, websiteRoot)
	if err != nil {
		return nil, err
	}
	if !apply {
		return &p.result, nil
	}
	lock, err := os.OpenFile(pathJoin(root, sourceMigrationLock), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	if err := lock.Close(); err != nil {
		return nil, err
	}
	created := []string{}
	rollback := func(cause error) (*SourceMigrationResult, error) {
		for i := len(created) - 1; i >= 0; i-- {
			if err := os.Remove(pathJoin(root, created[i])); err != nil {
				return nil, fmt.Errorf("%v; rollback failed: %v; mutation=unknown", cause, err)
			}
		}
		if err := os.Remove(lock.Name()); err != nil {
			return nil, fmt.Errorf("%v; lock cleanup failed: %v; mutation=unknown", cause, err)
		}
		return nil, fmt.Errorf("%v; mutation=none (rolled back new files)", cause)
	}
	for name, before := range p.baseline {
		current, err := readRegular(root, name)
		if err != nil || !bytes.Equal(current, before) {
			return rollback(fmt.Errorf("existing source changed during preflight: %s", name))
		}
	}
	keys := make([]string, 0, len(p.missing))
	for p := range p.missing {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	modeByPath := map[string]os.FileMode{}
	for _, f := range p.manifest.Files {
		modeByPath[f.Path] = os.FileMode(f.LocalMode)
	}
	for _, name := range keys {
		if err := installSourceFile(root, name, p.missing[name], modeByPath[name]); err != nil {
			return rollback(err)
		}
		created = append(created, name)
	}
	snapshot, err := reconstructSourceSubset(root, &p.manifest)
	if err != nil {
		return rollback(err)
	}
	current, err := BuildGlobal(root, snapshot)
	if err != nil || !reflect.DeepEqual(*current, p.global) {
		return rollback(fmt.Errorf("source storage migration would change registered global/package identity: %v", err))
	}
	b, err := Encode(p.manifest)
	if err != nil {
		return rollback(err)
	}
	if err := installSourceFile(root, SourceManifestPath, b, 0644); err != nil {
		return rollback(err)
	}
	created = append(created, SourceManifestPath)
	if err := os.Remove(lock.Name()); err != nil {
		return nil, fmt.Errorf("migration installed but lock cleanup failed: %v; mutation=unknown", err)
	}
	p.result.Applied = true
	return &p.result, nil
}

func installSourceFile(root, name string, b []byte, mode os.FileMode) error {
	if !validPath(name) {
		return fmt.Errorf("unsafe source destination")
	}
	parent := root
	for _, part := range splitPath(path.Dir(name)) {
		parent = pathJoin(parent, part)
		info, err := os.Lstat(parent)
		if os.IsNotExist(err) {
			if err := os.Mkdir(parent, 0755); err != nil {
				return err
			}
			info, err = os.Lstat(parent)
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe source parent: %s", parent)
		}
	}
	f, err := os.CreateTemp(parent, ".frozen-source-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Link(f.Name(), pathJoin(root, name))
}
