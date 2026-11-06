package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"golang.org/x/net/html"
)

func copiedSurfaceTransportRepository(t *testing.T) string {
	t.Helper()
	src := repositoryRoot(t)
	root := t.TempDir()
	if info, err := os.Stat(src); err != nil {
		t.Fatal(err)
	} else if err := os.Chmod(root, info.Mode().Perm()); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"data", "locales", "_content", "internal", "docs", "tour", "cmd", "scripts"} {
		if err := filepath.WalkDir(filepath.Join(src, dir), func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			rel, e := filepath.Rel(src, p)
			if e != nil {
				return e
			}
			dst := filepath.Join(root, rel)
			if d.IsDir() {
				if err := os.MkdirAll(dst, 0700); err != nil {
					return err
				}
				info, err := d.Info()
				if err != nil {
					return err
				}
				return os.Chmod(dst, info.Mode().Perm())
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("disposable proof refuses symlink: %s", rel)
			}
			in, e := os.Open(p)
			if e != nil {
				return e
			}
			defer in.Close()
			f, e := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				return e
			}
			_, e = io.Copy(f, in)
			closeErr := f.Close()
			if e != nil {
				return e
			}
			if closeErr != nil {
				return closeErr
			}
			info, e := d.Info()
			if e != nil {
				return e
			}
			return os.Chmod(dst, info.Mode().Perm())
		}); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".zip") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, b)
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(root, name), info.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(filepath.Join(src, "production/identity.json"))
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "production/identity.json", b)
	if info, err := os.Stat(filepath.Join(src, "production/identity.json")); err != nil {
		t.Fatal(err)
	} else if err := os.Chmod(filepath.Join(root, "production/identity.json"), info.Mode().Perm()); err != nil {
		t.Fatal(err)
	}
	return root
}

// Opt-in engineering proof: copy immutable real inputs into TempDir, then run
// readonly discovery/preflight only. It performs no real record/export/process.
func TestSurfaceTransportRealFiveCapabilityFixture(t *testing.T) {
	out := os.Getenv("GO_TOUR_SURFACE_CAPABILITY_PROOF")
	if out == "" {
		t.Skip("explicit disposable real-fixture proof requested")
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	root := copiedSurfaceTransportRepository(t)
	t.Log("REAL_FIXTURE_COPY_COMPLETE; original repository remains readonly")
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	AttachIntegratedSurfaceAuthority(c)
	if err := c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	discovery, err := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", "chatgpt-zh-CN-060", nil, false)
	if err != nil {
		t.Fatal("real fixture source-coordinate discovery", err)
	}
	data, _ := json.MarshalIndent(discovery, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "discovery.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	request := discovery.Authority.Request
	// These are explicit fixture scope proposals for the five known findings,
	// not production inference from reviewer prose. Every selected ID is a
	// source-adapter coordinate emitted above; the request is saved for review.
	ids := map[string][]string{
		"0895c1f2bf020716e82386c738f1b63ff49194250cf4cf254f9cf346a453fc10": {"randomFormat", "math/rand", "name"},
		"393fb67915a49876749590fd6480c9a3482597a9cebdd61e72c9a5f3d1449e2c": {"Hello", "go test", "TestHelloName", "Errorf", "TestHelloEmpty", "_test.go"},
		"66264496c6d846fefd6419e37c58df366152af07523cdd1f81088c8839f0e5ec": {"string", "errors.New", "log", "Fatal"},
		"b49a0f38f79787a87a714c55b10e2987434ebdcd78c2c1cc0480e46523893493": {"string", "Hello", "Hellos", "for", "range", "name/value"},
	}
	for _, binding := range discovery.Authority.Bindings {
		q := i18n.SurfaceTransportUnitRequest{UnitID: binding.UnitID, FindingSHA256: binding.FindingSHA256}
		for _, region := range binding.Projection.Available {
			selected := false
			if region.Kind == "inline-api-descriptor" {
				selected = (strings.Contains(region.Preview, "FormatDSN method") || strings.Contains(region.Preview, "sql.OpenDB function") || strings.Contains(region.Preview, "log package"))
			}
			if region.Kind == "ordinary-prose" {
				visible := surfaceFixtureVisible(region.RestoredPreview)
				for _, needle := range ids[binding.UnitID] {
					if needle == "name/value" {
						selected = selected || strings.Contains(visible, needle)
					} else {
						selected = selected || regexp.MustCompile(regexp.QuoteMeta(needle)+`\p{Han}`).MatchString(visible)
					}
				}
			}
			if selected {
				q.Regions = append(q.Regions, region.ID)
			}
		}
		if len(q.Regions) == 0 {
			t.Fatal("real finding has no source-owned capability", binding.UnitID)
		}
		request.Units = append(request.Units, q)
	}
	data, _ = json.MarshalIndent(request, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "request.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	p, err := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", "chatgpt-zh-CN-060", &request, false)
	if err != nil || !p.RevisionTransportCapable || p.Recorded {
		t.Fatal("real five-unit capability preflight", err)
	}
	if len(p.Authority.Bindings) != 5 {
		t.Fatal("real fixture exact scope changed")
	}
	report := []map[string]any{}
	for _, binding := range p.Authority.Bindings {
		if strings.HasPrefix(binding.UnitID, "872da6") && (!strings.Contains(binding.Projection.Template, "method") || !strings.Contains(binding.Projection.Template, "function")) {
			t.Fatal("real inline descriptor findings remain opaque")
		}
		if strings.HasPrefix(binding.UnitID, "662644") && !strings.Contains(binding.Projection.Template, "package") {
			t.Fatal("real log descriptor remains opaque")
		}
		if strings.HasPrefix(binding.UnitID, "b49a0f") && !strings.Contains(binding.Projection.Template, "name/value") {
			t.Fatal("real untranslated natural-language finding remains opaque")
		}
		u, err := c.Unit(binding.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		factory := u.Adapter.(i18n.SurfaceTransportAdapter)
		adapter, err := factory.ReadSurfaceTransportProjection(binding.Projection.Codec)
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := readRegular(root, binding.Predecessor.CandidatePath)
		if err != nil {
			t.Fatal(err)
		}
		got, err := adapter.Restore(binding.Projection.Template)
		if err != nil || !bytes.Equal(bridgeEOF(got), candidate) {
			t.Fatal("real exact restore", binding.UnitID, err)
		}
		var codec surfacePageTransport
		if err := json.Unmarshal(binding.Projection.Codec, &codec); err != nil {
			t.Fatal(err)
		}
		opened := map[string]bool{}
		for _, part := range codec.Parts {
			if part.Region == "" || opened[part.Region] {
				continue
			}
			opened[part.Region] = true
			value := " " + part.Value
			if part.Atom >= 0 {
				value = "词"
			} // structural capability probe only, never a language result
			test := strings.Replace(codec.Template, part.Token+part.Value+part.End, part.Token+value+part.End, 1)
			changed, err := adapter.Restore(test)
			if err != nil {
				t.Fatal("real authorized edit not executable", binding.UnitID, part.Region, err)
			}
			if err := i18n.ValidateTranslationUnitCandidate(root, c, u.ID, "zh-CN", changed); err != nil {
				t.Fatal("real active structural/identity validator", binding.UnitID, err)
			}
		}
		if len(opened) != len(binding.Projection.Editable) {
			t.Fatal("declared real regions are not editable", binding.UnitID)
		}
		if _, err := adapter.Restore(binding.Projection.Template + "outside"); err == nil {
			t.Fatal("real opaque boundary escaped")
		}
		raw, err := readRegular(root, binding.Raw.Path)
		if err != nil {
			t.Fatal(err)
		}
		old := "ordinary/inherited"
		if bytes.Contains(raw, []byte("⟪GTI18N_")) {
			old = "teaching-comment-restricted"
		}
		report = append(report, map[string]any{"unit_id": binding.UnitID, "old_representation_kind": old, "new_revision_transport_kind": binding.Projection.Kind, "editable_scope": binding.Projection.Editable, "opaque_scope_preserved": true, "restore_exact": true, "capability": "PASS", "synthetic_edit_probe_only": true})
	}
	data, _ = json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "five-unit-capabilities.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, p.Path)); !os.IsNotExist(err) {
		t.Fatal("real fixture readonly preflight recorded authority", err)
	}
	t.Log("REAL_5_UNIT_CAPABILITY=PASS; RECORD=no; GENERATION=no; EXPORT=no; original formal artifacts unchanged")
}

func surfaceFixtureVisible(s string) string {
	z := html.NewTokenizer(strings.NewReader(s))
	var b strings.Builder
	for {
		switch z.Next() {
		case html.ErrorToken:
			return b.String()
		case html.TextToken:
			b.Write(z.Text())
		}
	}
}
