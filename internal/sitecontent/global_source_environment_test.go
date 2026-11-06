package sitecontent

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func globalEnvironmentFixture(t *testing.T) (string, []byte) {
	t.Helper()
	root, from := t.TempDir(), repositoryRoot(t)
	copy := func(name string) {
		b, err := readRegular(from, name)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, b)
	}
	if err := filepath.WalkDir(filepath.Join(from, "_content/tour"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(from, p)
			copy(filepath.ToSlash(rel))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	copy("UPSTREAM_MANIFEST.tsv")
	for _, s := range ShellSources() {
		copy(s.Path)
	}
	b, err := ReadSourceSnapshot(from)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, SnapshotPath, b)
	g, err := buildGlobalCanonical(root, b)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := Encode(g)
	writeFixture(t, root, GlobalPath, encoded)
	return root, b
}

func TestGlobalAuthorityEnvironmentCrossProcess(t *testing.T) {
	if root := os.Getenv("GO_TOUR_GLOBAL_ENV_FIXTURE"); root != "" {
		finish := contentidentity.BeginMemoization()
		defer finish()
		b, err := readRegular(root, SnapshotPath)
		if err != nil {
			t.Fatal(err)
		}
		v := successorValidation(root, nil)
		v.historicalMetrics = map[string]int{}
		start := time.Now()
		g, err := rebuildHistoricalGlobal(root, v)
		if err != nil {
			t.Fatal(err)
		}
		// This goes through the expected Global comparison too; the cached
		// projection is data, never an authorization to accept its declaration.
		other, err := BuildGlobal(root, b, v)
		if err != nil || !reflect.DeepEqual(g, other) {
			t.Fatal("projection differs", err)
		}
		raw, _ := json.Marshal(map[string]any{"builds": v.historicalMetrics["global-environment-build"], "identity": g.Identity, "seconds": time.Since(start).Seconds()})
		t.Logf("GLOBAL_ENV=%s", raw)
		return
	}
	root, _ := globalEnvironmentFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var identity string
	for phase := 0; phase < 2; phase++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestGlobalAuthorityEnvironmentCrossProcess$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), "GO_TOUR_GLOBAL_ENV_FIXTURE="+root)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(string(b), err)
		}
		found := false
		for _, line := range strings.Split(string(b), "\n") {
			at := strings.Index(line, "GLOBAL_ENV=")
			if at < 0 {
				continue
			}
			var r struct {
				Builds   int
				Identity string
				Seconds  float64
			}
			if err := json.Unmarshal([]byte(line[at+len("GLOBAL_ENV="):]), &r); err != nil {
				t.Fatal(err)
			}
			expected := 1
			if phase == 1 {
				expected = 0
			}
			if r.Builds != expected {
				t.Fatal("duplicate complete Global derivation", phase, r)
			}
			if phase == 0 {
				identity = r.Identity
			} else if r.Identity != identity {
				t.Fatal("identity changed")
			}
			t.Logf("GLOBAL_ENV phase=%d builds=%d seconds=%.6f identity=%s", phase, r.Builds, r.Seconds, r.Identity)
			found = true
		}
		if !found {
			t.Fatal(string(b))
		}
	}
}

func TestGlobalAuthorityEnvironmentFreshGuards(t *testing.T) {
	root, snapshot := globalEnvironmentFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	load := func() (*Global, error) {
		done := contentidentity.BeginMemoization()
		defer done()
		return rebuildHistoricalGlobal(root)
	}
	want, err := load()
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{ShellSources()[0].Path, GlobalPath, SnapshotPath, "_content/tour/basics.article", "UPSTREAM_MANIFEST.tsv"}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			original, err := readRegular(root, p)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, p, original)
			changed := append(bytes.Clone(original), ' ')
			if p == SnapshotPath {
				changed = bytes.Clone(original)
				changed[0] ^= 1
			}
			if p == GlobalPath {
				var g Global
				json.Unmarshal(original, &g)
				g.UpstreamCommit = "wrong"
				changed, _ = json.Marshal(g)
			}
			if p == "UPSTREAM_MANIFEST.tsv" {
				changed = []byte("invalid manifest")
			}
			writeFixture(t, root, p, changed)
			if _, err := load(); err == nil {
				t.Fatal("warm projection accepted changed authority", p)
			}
		})
	}
	if got, err := load(); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("restored exact inputs", err)
	}
	done := contentidentity.BeginMemoization()
	defer done()
	first, err := BuildGlobal(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	first.Sources[0].SHA256 = "caller mutation"
	if got, err := BuildGlobal(root, snapshot); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("shared data aliases caller", err)
	}
}

func TestGlobalAuthorityEnvironmentCodecFallback(t *testing.T) {
	root, snapshot := globalEnvironmentFixture(t)
	want, err := buildGlobalCanonical(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	o, err := observeGlobalSource(root)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := json.Marshal(struct {
		Snapshot string
		Inputs   globalSourceObservation
	}{digest(snapshot), o})
	finish := contentidentity.BeginMemoization()
	_, err = contentidentity.Transform("sitecontent/global-source-environment/v1", key, func() ([]byte, error) { return []byte("invalid derived Global codec"), nil })
	finish()
	if err != nil {
		t.Fatal(err)
	}
	finish = contentidentity.BeginMemoization()
	defer finish()
	got, err := BuildGlobal(root, snapshot)
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatal("derived cache codec became an availability gate", err)
	}
}
