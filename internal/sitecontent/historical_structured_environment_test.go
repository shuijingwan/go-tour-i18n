package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func historicalGraphFixture(t *testing.T) historicalStructuredInput {
	t.Helper()
	raw := []byte(strings.Repeat("Use the CLI with [this guide](/fixed).\n\n", 12))
	d, err := ParseDocument(Source{Path: "history.md", Kind: "page", SHA256: digest(raw), Dependencies: []Dependency{}}, raw)
	if err != nil {
		t.Fatal(err)
	}
	return historicalStructuredInput{*d, raw, bytes.Clone(raw)}
}

func TestQCSurfaceHistoryDerivationCrossProcess(t *testing.T) {
	if root := os.Getenv("GO_TOUR_HISTORY_GRAPH_ROOT"); root != "" {
		// Every child observes formal fixture bytes anew; only the parser data
		// graph survives the process boundary, never an acceptance verdict.
		b, err := os.ReadFile(filepath.Join(root, "document.json"))
		if err != nil {
			t.Fatal(err)
		}
		var d Document
		if err := json.Unmarshal(b, &d); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(root, "source.md"))
		if err != nil {
			t.Fatal(err)
		}
		target, err := os.ReadFile(filepath.Join(root, "candidate.md"))
		if err != nil {
			t.Fatal(err)
		}
		b, err = os.ReadFile(filepath.Join(root, "expected-graph.json"))
		if err != nil {
			t.Fatal(err)
		}
		var want historicalStructuredGraph
		if err := json.Unmarshal(b, &want); err != nil {
			t.Fatal(err)
		}
		finish := contentidentity.BeginMemoization()
		defer finish()
		got, built, err := sharedHistoricalStructuredGraph(d, raw, target)
		if err != nil || !reflect.DeepEqual(&want, got) {
			t.Fatal("historical graph identity changed", err)
		}
		values := map[string]string{}
		for _, u := range d.Units {
			values[u.ID] = u.Source
		}
		out, err := reconstructFrozenStructuredRepresentation(d, raw, values)
		if err != nil || !bytes.Equal(out, target) {
			t.Fatal("canonical frozen restoration changed", err)
		}
		// A consumer's private graph must not corrupt another consumer's data.
		got.Source.Units[0].Source = "caller mutation"
		again, repeated, err := sharedHistoricalStructuredGraph(d, raw, target)
		if err != nil || repeated || !reflect.DeepEqual(&want, again) {
			t.Fatal("graph ownership or invocation reuse changed", err)
		}
		count := 0
		if built {
			count = 1
		}
		t.Logf("QC_SURFACE_GRAPH builds=%d identity=%s", count, identity(again))
		return
	}
	in := historicalGraphFixture(t)
	root := t.TempDir()
	b, _ := json.Marshal(in.Document)
	writeFixture(t, root, "document.json", b)
	writeFixture(t, root, "source.md", in.Raw)
	writeFixture(t, root, "candidate.md", in.Target)
	want, err := deriveHistoricalStructuredGraph(in)
	if err != nil {
		t.Fatal(err)
	}
	b, _ = json.Marshal(want)
	writeFixture(t, root, "expected-graph.json", b)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var previousIdentity string
	for phase := 0; phase < 2; phase++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestQCSurfaceHistoryDerivationCrossProcess$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), "GO_TOUR_HISTORY_GRAPH_ROOT="+root)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(string(b), err)
		}
		found := false
		for _, line := range strings.Split(string(b), "\n") {
			at := strings.Index(line, "QC_SURFACE_GRAPH builds=")
			if at < 0 {
				continue
			}
			expected := "1"
			if phase == 1 {
				expected = "0"
			}
			if !strings.HasPrefix(line[at:], "QC_SURFACE_GRAPH builds="+expected+" identity=") {
				t.Fatal("complete historical graph rebuilt", line)
			}
			gotIdentity := strings.Split(line[at:], "identity=")[1]
			if phase > 0 && gotIdentity != previousIdentity {
				t.Fatal("warm historical graph identity changed")
			}
			previousIdentity = gotIdentity
			t.Logf("QC_SURFACE_HISTORY phase=%d %s", phase, line[at:])
			found = true
		}
		if !found {
			t.Fatal(string(b))
		}
	}
}

func TestQCSurfaceHistoryDerivationInputsAndGates(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	in := historicalGraphFixture(t)
	finish := contentidentity.BeginMemoization()
	defer finish()
	if _, built, err := sharedHistoricalStructuredGraph(in.Document, in.Raw, in.Target); err != nil || !built {
		t.Fatal("initial graph did not build", err)
	}
	values := map[string]string{}
	for _, u := range in.Document.Units {
		values[u.ID] = u.Source
	}
	for _, field := range []string{"source", "document", "protected-target", "target-structure"} {
		t.Run(field, func(t *testing.T) {
			b, _ := json.Marshal(in.Document)
			var d Document
			_ = json.Unmarshal(b, &d)
			raw := bytes.Clone(in.Raw)
			targets := map[string]string{}
			for k, v := range values {
				targets[k] = v
			}
			switch field {
			case "source":
				raw = append(raw, ' ')
			case "document":
				d.StructureSHA = strings.Repeat("0", 64)
			case "protected-target":
				for k := range targets {
					targets[k] = "missing protected identities"
					break
				}
			case "target-structure":
				for k := range targets {
					targets[k] = "## " + targets[k]
					break
				}
			}
			if _, err := reconstructFrozenStructuredRepresentation(d, raw, targets); err == nil {
				t.Fatal("warm data bypassed fresh frozen/protected comparison")
			}
		})
	}
	changed := bytes.ReplaceAll(in.Target, []byte("this guide"), []byte("another guide"))
	if _, built, err := sharedHistoricalStructuredGraph(in.Document, in.Raw, changed); err != nil || !built {
		t.Fatal("changed target did not derive its own graph", err)
	}
	bad := []byte{0xff}
	for i := 0; i < 2; i++ {
		if _, built, err := sharedHistoricalStructuredGraph(in.Document, bad, in.Target); err == nil || !built {
			t.Fatal("failed derivation cached", i, err)
		}
	}
}

func TestQCSurfaceHistoryDerivationCodecFallback(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	in := historicalGraphFixture(t)
	want, err := deriveHistoricalStructuredGraph(in)
	if err != nil {
		t.Fatal(err)
	}
	finish := contentidentity.BeginMemoization()
	defer finish()
	key, _ := json.Marshal(in)
	_, err = contentidentity.Transform("sitecontent/historical-structured-graph/v1", key, func() ([]byte, error) { return []byte("{}"), nil })
	if err != nil {
		t.Fatal(err)
	}
	got, built, err := sharedHistoricalStructuredGraph(in.Document, in.Raw, in.Target)
	if err != nil || !built || !reflect.DeepEqual(want, got) {
		t.Fatal("unusable derived graph did not fall back to canonical parser", err)
	}
}
