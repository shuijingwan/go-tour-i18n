package sitecontent

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const shared008RequestPath = "tmp/shared-structured-representation/site-v2-shell-site-tmpl-surface008-request.json"
const shared008RequestSHA = "f792f1df3a59d1cda519dd42b932f7b96bf50b8749253b20fef9eade8eac6f9b"

func shared008Request(t *testing.T) (string, Reference, SharedRepresentationRequest) {
	t.Helper()
	root := repositoryRoot(t)
	b, err := readRegular(root, shared008RequestPath)
	if os.IsNotExist(err) {
		t.Skip("exact retained first-use evidence is not in this checkout")
	}
	if err != nil || digest(b) != shared008RequestSHA {
		t.Fatal("retained request bytes/SHA", err)
	}
	var q SharedRepresentationRequest
	if err := StrictJSON(b, &q); err != nil {
		t.Fatal(err)
	}
	return root, Reference{shared008RequestPath, shared008RequestSHA}, q
}

func TestSharedRepresentationHistoricalUnitsEvidence(t *testing.T) {
	root, _, q := shared008Request(t)
	r, _, _, err := readStructuredGeneration(root, q.Generation)
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{r.Bundle}
	if r.CarryForward != nil {
		refs = append(refs, r.CarryForward.CurrentBundle)
	}
	for _, ref := range refs {
		b, err := readRegular(root, ref.Path)
		if err != nil || digest(b) != ref.SHA256 {
			t.Fatal("frozen archive", err)
		}
		z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range z.File {
			if strings.HasSuffix(f.Name, "internal/sitecontent/units.go") {
				t.Fatal("classification assumption changed: units.go is now a historical Bundle member")
			}
		}
	}
	var head Reference
	b, err := readRegular(root, ShellRepresentationCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(b, &head); err != nil {
		t.Fatal(err)
	}
	var home ShellRepresentationSuccessor
	if err := readReference(root, head, &home); err != nil {
		t.Fatal(err)
	}
	var rehead Reference
	b, err = readRegular(root, shellReattestationPath(home.Identity))
	if err != nil {
		t.Fatal(err)
	}
	if err := StrictJSON(b, &rehead); err != nil {
		t.Fatal(err)
	}
	var reattestation ShellRepresentationReattestation
	if err := readReference(root, rehead, &reattestation); err != nil {
		t.Fatal(err)
	}
	// Do not whitelist mixed modules. The new family needs no edits to these
	// independent frozen Homepage bindings, including its existing currentness.
	for _, name := range []string{"internal/sitecontent/units.go", "internal/sitecontent/shell_reconstruction.go", "internal/sitecontent/shell_representation_successor.go"} {
		found := false
		for _, input := range reattestation.CurrentInputs {
			if input.Reference.Path != name {
				continue
			}
			found = true
			b, err := readRegular(root, name)
			if err != nil || digest(b) != input.Reference.SHA256 {
				t.Fatal("unrelated Homepage dependency changed", name, err)
			}
		}
		if !found {
			t.Fatal("missing exact Homepage dependency", name)
		}
	}
	t.Log("UNITS_DRIFT_CLASS=not-historical-bundle-member; error owner=Homepage shellDependencyChanges; frozen Home bindings unchanged")
}

// Mirror only the request's reachable reference/source graph into t.TempDir,
// preserving bytes, modes and absence. It never invokes a real planner and does
// not synthesize/reattest any authority or substitute the consumed Generation.
func mirrorShared008Authority(t *testing.T, real string, request Reference) (string, func(string)) {
	t.Helper()
	root := t.TempDir()
	queue := []string{request.Path, GlobalPath, SourceManifestPath, PackageMembershipPath, PageRegistryPath, ShellAdoptionCurrent, ShellRepresentationCurrent, "data/glossary-source-corpus.json", "go.mod", "go.sum", "data/tour-pages.tsv", "data/tour-conditional-pages.tsv", "data/tour-examples.tsv", "UPSTREAM_MANIFEST.tsv"}
	queue = append(queue, PackageAuthorities...)
	queue = append(queue, sharedRepresentationParserPaths...)
	for _, dir := range []string{StructuredSourceSuccessorRoot + "current", PagePackageAdoptionRoot + "current", ShellRepresentationReattestationRoot + "current"} {
		entries, err := os.ReadDir(pathJoin(real, dir))
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		for _, entry := range entries {
			queue = append(queue, dir+"/"+entry.Name())
		}
	}
	seen := map[string]bool{}
	var total int64
	var walk func(any, string)
	walk = func(value any, base string) {
		switch x := value.(type) {
		case map[string]any:
			for _, v := range x {
				walk(v, base)
			}
		case []any:
			for _, v := range x {
				walk(v, base)
			}
		case string:
			if len(x) > 2048 || strings.ContainsAny(x, "\n\r\t\x00") {
				return
			}
			rooted := x == "AGENTS.md" || x == "go.mod" || x == "go.sum" || x == "UPSTREAM_MANIFEST.tsv"
			for _, prefix := range []string{"data/", "locales/", "internal/", "docs/", "_content/", "tmp/"} {
				rooted = rooted || strings.HasPrefix(x, prefix)
			}
			if rooted && validPath(x) {
				queue = append(queue, x)
			}
			if !rooted && validPath(x) && base != "." && !strings.Contains(x, " ") && (strings.HasSuffix(x, ".json") || strings.HasPrefix(x, "candidates/") || strings.HasPrefix(x, "validation/") || strings.HasPrefix(x, "raw-responses/") || strings.HasPrefix(x, "inputs/")) {
				queue = append(queue, path.Join(base, x))
			}
		}
	}
	drain := func() {
		for len(queue) > 0 {
			name := queue[0]
			queue = queue[1:]
			if seen[name] || !validPath(name) {
				continue
			}
			longComponent := false
			for _, component := range strings.Split(name, "/") {
				longComponent = longComponent || len(component) > 255
			}
			if longComponent {
				continue
			}
			seen[name] = true
			info, err := os.Lstat(pathJoin(real, name))
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if !info.Mode().IsRegular() {
				continue
			}
			b, err := readRegular(real, name)
			if err != nil {
				t.Fatal(err)
			}
			total += int64(len(b))
			if len(seen) > 30000 || total > 512<<20 {
				t.Fatal("bounded request reference graph exceeded")
			}
			writeFixture(t, root, name, b)
			if err := os.Chmod(pathJoin(root, name), info.Mode().Perm()); err != nil {
				t.Fatal(err)
			}
			if name == SourceManifestPath {
				var m FrozenSourceManifest
				if err := StrictJSON(b, &m); err != nil {
					t.Fatal(err)
				}
				for _, p := range m.Legacy.Paths {
					queue = append(queue, adoptedSourcePath(p))
				}
				continue // Manifest asset inventory is not a request to copy a full site.
			}
			if strings.HasSuffix(name, ".json") {
				var value any
				if json.Unmarshal(b, &value) == nil {
					walk(value, path.Dir(name))
				}
			}
			if strings.HasSuffix(name, ".zip") {
				z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
				if err != nil {
					t.Fatal(err)
				}
				for _, f := range z.File {
					if f.Name == "inputs/"+SourceManifestPath {
						continue
					}
					if !strings.HasSuffix(f.Name, ".json") || f.UncompressedSize64 > 32<<20 {
						continue
					}
					reader, err := f.Open()
					if err != nil {
						t.Fatal(err)
					}
					data, err := io.ReadAll(io.LimitReader(reader, 32<<20))
					reader.Close()
					if err != nil {
						t.Fatal(err)
					}
					var value any
					if json.Unmarshal(data, &value) == nil {
						walk(value, ".")
					}
				}
			}
		}
	}
	drain()
	t.Logf("isolated exact request graph: candidate paths=%d bytes=%d", len(seen), total)
	extend := func(name string) {
		t.Helper()
		if !validPath(name) {
			t.Fatal("unsafe fixture reference", name)
		}
		info, err := os.Lstat(pathJoin(real, name))
		if err != nil {
			t.Fatal(err)
		}
		if info.IsDir() {
			if err := os.MkdirAll(pathJoin(root, name), info.Mode().Perm()); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(pathJoin(real, name))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				queue = append(queue, name+"/"+entry.Name())
			}
		} else {
			delete(seen, name)
			queue = append(queue, name)
		}
		drain()
	}
	return root, extend
}

func TestSharedRepresentationExactRequestHistoricalBoundary(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	real, request, q := shared008Request(t)
	root, extend := mirrorShared008Authority(t, real, request)
	var plan *SharedRepresentationPlan
	// Only fill an explicitly reported missing fixture file. Never rewrite a
	// fact, relax a gate or pick a different Generation to obtain eligibility.
	for attempt := 0; attempt < 24; attempt++ {
		var err error
		plan, err = PlanSharedStructuredRepresentation(root, request)
		if err == nil {
			break
		}
		var missing *os.PathError
		if !errors.As(err, &missing) || !os.IsNotExist(missing.Err) {
			t.Fatal("isolated exact request", err)
		}
		rel, relErr := filepath.Rel(root, missing.Path)
		if relErr != nil || !validPath(filepath.ToSlash(rel)) {
			t.Fatal(err)
		}
		extend(filepath.ToSlash(rel))
	}
	if plan == nil {
		t.Fatal("exact request did not produce an isolated prospective proof")
	}
	if plan.Request != request || plan.Authority.Request.Generation != q.Generation || len(plan.Authority.Released) != 1 || plan.Authority.Released[0].SlotID != q.Slots[0] {
		t.Fatal("request/consumed evidence/topology changed")
	}
	before, err := readRegular(root, q.Generation.Path)
	if err != nil {
		t.Fatal(err)
	}
	parser := sharedRepresentationParserPaths[0]
	original, err := readRegular(root, parser)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, parser, append(bytes.Clone(original), []byte("\n// synthetic prospective parser epoch\n")...))
	if _, _, _, err := sharedRepresentationConsumedGeneration(root, q.Generation, newClosureValidation(root)); err != nil {
		t.Fatal("historical frozen Generation required today's parser bytes", err)
	}
	if err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		return checkSharedRepresentationCurrent(root, g, plan.Authority, v)
	}); err == nil || !strings.Contains(err.Error(), parser) {
		t.Fatal("old plan authority did not reject changed current parser", err)
	}
	current, err := PlanSharedStructuredRepresentation(root, request)
	if err != nil {
		t.Fatal("current parser binding", err)
	}
	if current.Identity == plan.Identity {
		t.Fatal("old plan remained current after parser dependency drift")
	}
	found := false
	for _, input := range current.Authority.Inputs {
		if input.Reference.Path == parser {
			found = true
			if input.Reference.SHA256 == digest(original) {
				t.Fatal("old parser epoch bound")
			}
		}
	}
	if !found {
		t.Fatal("current parser dependency omitted")
	}
	if after, _ := readRegular(root, q.Generation.Path); !bytes.Equal(before, after) {
		t.Fatal("historical receipt rewritten")
	}
	if after, _ := readRegular(root, request.Path); digest(after) != request.SHA256 {
		t.Fatal("request rewritten")
	}
	if _, err := os.Stat(pathJoin(root, current.CurrentPath)); !os.IsNotExist(err) {
		t.Fatal("readonly plan installed a head", err)
	}
	for _, ref := range []Reference{plan.Receipt, current.Receipt} {
		if _, err := os.Stat(pathJoin(root, ref.Path)); !os.IsNotExist(err) {
			t.Fatal("readonly proof published a receipt", err)
		}
	}
}

func TestSharedRepresentationIndependentAuthorityDrift(t *testing.T) {
	for _, name := range []string{"internal/sitecontent/units.go", "internal/sitecontent/unclassified-authority.go"} {
		old := []ShellRepresentationInput{{Reference: Reference{name, strings.Repeat("a", 64)}, Mode: 0644}}
		current := []ShellRepresentationInput{{Reference: Reference{name, strings.Repeat("b", 64)}, Mode: 0644}}
		if _, err := shellDependencyChanges(old, current); err == nil || !strings.Contains(err.Error(), "non-parser representation authority drift") {
			t.Fatal("another family's parser list granted drift permission", name, err)
		}
	}
}
