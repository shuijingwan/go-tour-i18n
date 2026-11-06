package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestPackageRepresentationEnvironmentCrossProcess(t *testing.T) {
	if root := os.Getenv("GO_TOUR_PACKAGE_ENV_FIXTURE"); root != "" {
		finish := contentidentity.BeginMemoization()
		defer finish()
		v := successorValidation(root, nil)
		v.historicalMetrics = map[string]int{}
		g, err := rebuildHistoricalGlobal(root, v)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		docs, raw, err := CurrentPackageDocuments(root, g, "learn-docs-v1", v)
		if err != nil {
			t.Fatal(err)
		}
		data := packageRepresentationData{docs, raw}
		b, _ := json.Marshal(map[string]any{"builds": v.historicalMetrics["package-representation-build"], "identity": identity(data), "seconds": time.Since(start).Seconds()})
		t.Logf("PACKAGE_ENV=%s", b)
		return
	}
	root, _ := globalEnvironmentFixture(t)
	g, err := rebuildHistoricalGlobal(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := PackageDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	want := identity(packageRepresentationData{docs, raw})
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	for phase := 0; phase < 2; phase++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestPackageRepresentationEnvironmentCrossProcess$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), "GO_TOUR_PACKAGE_ENV_FIXTURE="+root)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(string(b), err)
		}
		found := false
		for _, line := range strings.Split(string(b), "\n") {
			at := strings.Index(line, "PACKAGE_ENV=")
			if at < 0 {
				continue
			}
			var row struct {
				Builds   int
				Identity string
				Seconds  float64
			}
			if json.Unmarshal([]byte(line[at+len("PACKAGE_ENV="):]), &row) != nil {
				t.Fatal(line)
			}
			expected := 1
			if phase == 1 {
				expected = 0
			}
			if row.Builds != expected || row.Identity != want {
				t.Fatal("complete package representation not reused exactly", phase, row)
			}
			t.Logf("PACKAGE_ENV phase=%d builds=%d seconds=%.6f identity=%s", phase, row.Builds, row.Seconds, row.Identity)
			found = true
		}
		if !found {
			t.Fatal(string(b))
		}
	}
}

func TestPackageRepresentationEnvironmentInputs(t *testing.T) {
	root, snapshot := globalEnvironmentFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	g, err := buildGlobalCanonical(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"learn-docs-v1", "site-v2-shell"} {
		t.Run(id, func(t *testing.T) {
			want, raw, err := PackageDocuments(root, g, id)
			if err != nil {
				t.Fatal(err)
			}
			for phase := 0; phase < 2; phase++ {
				finish := contentidentity.BeginMemoization()
				got, next, err := sharedPackageRepresentation(root, g, id)
				finish()
				if err != nil || !reflect.DeepEqual(want, got) || !reflect.DeepEqual(raw, next) {
					t.Fatal("package data changed", phase, err)
				}
				got[0].SourceSHA = "caller mutation"
				next[got[0].Path] = []byte("caller mutation")
			}
		})
	}
	path := ShellSources()[0].Path
	original, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, path, append(bytes.Clone(original), ' '))
	finish := contentidentity.BeginMemoization()
	defer finish()
	if _, _, err := sharedPackageRepresentation(root, g, "site-v2-shell"); err == nil {
		t.Fatal("warm source mismatch accepted")
	}
}

func TestPackageRepresentationEnvironmentCodecFallback(t *testing.T) {
	root, snapshot := globalEnvironmentFixture(t)
	g, err := buildGlobalCanonical(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	want, raw, err := PackageDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	p, _ := packageByID(g, "learn-docs-v1")
	input := packageRepresentationInput{Package: p, SnapshotSHA: digest(snapshot)}
	for _, s := range g.Sources {
		if s.Package == p.ID && (s.Kind == "page" || s.Kind == "data") {
			input.Sources = append(input.Sources, s)
		}
	}
	key, _ := json.Marshal(input)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	finish := contentidentity.BeginMemoization()
	_, err = contentidentity.Transform("sitecontent/package-representation-environment/v1", key, func() ([]byte, error) { return []byte("invalid package representation codec"), nil })
	finish()
	if err != nil {
		t.Fatal(err)
	}
	finish = contentidentity.BeginMemoization()
	defer finish()
	got, next, err := sharedPackageRepresentation(root, g, p.ID)
	if err != nil || !reflect.DeepEqual(want, got) || !reflect.DeepEqual(raw, next) {
		t.Fatal("package codec prevented recomputation", err)
	}
}
