package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func bridgeGraphInputs(t *testing.T) (PageAdapter, *surfacePageTransport, []byte, []byte) {
	t.Helper()
	source := []byte("# Source mapping\n\nRead `sql.OpenDB function` now.\n\nKeep ordinary prose.\n")
	d, err := ParseWorkflowDocument(Source{Path: "_content/doc/context.md", Kind: "page", Route: "/doc/context", SHA256: digest(source)}, source, 1)
	if err != nil {
		t.Fatal(err)
	}
	a := PageAdapter{*d, source}
	text, _, err := a.ProtectedInput(nil)
	if err != nil {
		t.Fatal(err)
	}
	discovery, _, err := a.SurfaceTransportProjection(a, []byte(text), nil)
	if err != nil {
		t.Fatal(err)
	}
	var region string
	for _, r := range discovery.Available {
		if r.Kind == "inline-api-descriptor" {
			region = r.ID
		}
	}
	p, decoder, err := a.SurfaceTransportProjection(a, []byte(text), []string{region})
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(strings.Replace(p.Template, "function", "procedure", 1))
	reviewed, err := decoder.Restore(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	return a, decoder.(*surfacePageTransport), raw, reviewed
}

func TestBridgeRepresentationEnvironmentCrossProcess(t *testing.T) {
	if os.Getenv("GO_TOUR_BRIDGE_GRAPH_CHILD") != "" {
		finish := contentidentity.BeginMemoization()
		defer finish()
		a, p, raw, reviewed := bridgeGraphInputs(t)
		input := struct {
			Kind          string
			Page          PageAdapter
			Transport     surfacePageTransport
			Raw, Reviewed []byte
		}{"surface-to-ordinary", a, *p, raw, reviewed}
		r, built, err := deriveBridgeRepresentation(input, func() (*bridgeRepresentationData, error) {
			adapter, template, ancestor, err := a.BridgeSurfaceRevision(p, raw, reviewed)
			if err != nil {
				return nil, err
			}
			return &bridgeRepresentationData{adapter.(PageAdapter), template, ancestor}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		wrapped := observedBridgePage{PageAdapter: a, original: a}
		adapter, template, ancestor, err := wrapped.BridgeSurfaceRevision(p, raw, reviewed)
		if err != nil || !reflect.DeepEqual(r.Page, adapter) || !bytes.Equal(template, r.Template) || !bytes.Equal(ancestor, r.Ancestor) {
			t.Fatal("consumer representation differs", err)
		}
		b, _ := json.Marshal(map[string]any{"built": built, "identity": identity(r)})
		t.Logf("BRIDGE_GRAPH=%s", b)
		return
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var identity string
	for phase := 0; phase < 2; phase++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestBridgeRepresentationEnvironmentCrossProcess$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), "GO_TOUR_BRIDGE_GRAPH_CHILD=1")
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(string(b), err)
		}
		found := false
		for _, line := range strings.Split(string(b), "\n") {
			at := strings.Index(line, "BRIDGE_GRAPH=")
			if at < 0 {
				continue
			}
			var row struct {
				Built    bool
				Identity string
			}
			if json.Unmarshal([]byte(line[at+len("BRIDGE_GRAPH="):]), &row) != nil {
				t.Fatal(line)
			}
			if row.Built != (phase == 0) {
				t.Fatal("bridge pure graph not reused", phase, row)
			}
			if phase == 0 {
				identity = row.Identity
			} else if identity != row.Identity {
				t.Fatal("bridge graph identity changed")
			}
			t.Logf("BRIDGE_GRAPH phase=%d built=%v identity=%s", phase, row.Built, row.Identity)
			found = true
		}
		if !found {
			t.Fatal(string(b))
		}
	}
}

func TestBridgeRepresentationEnvironmentDrift(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	finish := contentidentity.BeginMemoization()
	defer finish()
	a, p, raw, reviewed := bridgeGraphInputs(t)
	wrapped := observedBridgePage{PageAdapter: a, original: a}
	want, template, ancestor, err := a.BridgeSurfaceRevision(p, raw, reviewed)
	if err != nil {
		t.Fatal(err)
	}
	got, next, prior, err := wrapped.BridgeSurfaceRevision(p, raw, reviewed)
	if err != nil || !reflect.DeepEqual(want, got) || !bytes.Equal(template, next) || !bytes.Equal(ancestor, prior) {
		t.Fatal("canonical bridge output changed", err)
	}
	for _, kind := range []string{"source", "mapping", "raw", "candidate"} {
		t.Run(kind, func(t *testing.T) {
			b, _ := json.Marshal(p)
			var decoder surfacePageTransport
			json.Unmarshal(b, &decoder)
			r, target := bytes.Clone(raw), bytes.Clone(reviewed)
			copy := wrapped
			switch kind {
			case "source":
				// The canonical factory binds formal source identity; fresh raw
				// source reads/hash checks remain in its caller's authority gate.
				copy.Document.SourceSHA = digest(append(bytes.Clone(a.Source), 'x'))
			case "mapping":
				decoder.UnitID = "wrong"
			case "raw":
				r = []byte("broken")
			case "candidate":
				target = append(target, 'x')
			}
			if _, _, _, err := copy.BridgeSurfaceRevision(&decoder, r, target); err == nil {
				t.Fatal("changed semantic input hit prior graph", kind)
			}
		})
	}
	if c := BridgeRepresentationCatalog(t.TempDir(), &i18n.Catalog{Adapted: []i18n.AdaptedTranslationUnit{{Adapter: a}}}); reflect.TypeOf(c.Adapted[0].Adapter) != reflect.TypeOf(wrapped) {
		t.Fatal("private construction Catalog not decorated")
	}
}

func TestBridgeCurrentSourceObservationOwner(t *testing.T) {
	root, _, _, _, _ := historicalGlobalSuccessorFixture(t)
	corpus, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptCorpusProjection(root, digest(corpus), true); err != nil {
		t.Fatal(err)
	}
	a, _, _, _ := bridgeGraphInputs(t)
	original := adoptedPageAdapter{PageAdapter: a, root: root}
	c := &i18n.Catalog{Adapted: []i18n.AdaptedTranslationUnit{{Adapter: original}}}
	var prior *structuredSuccessorValidation
	for phase := 0; phase < 2; phase++ {
		_, err := withBridgeConstructionEnvironment(root, c, func(scoped *i18n.Catalog) (*i18n.SameUnitBridgeResolution, error) {
			page := scoped.Adapted[0].Adapter.(observedBridgePage)
			v := page.proof
			if v == nil || v == prior {
				t.Fatal("proof owner escaped invocation")
			}
			prior = v
			v.historicalMetrics = map[string]int{}
			text, tokens, err := page.ProtectedInput(nil)
			if err != nil {
				return nil, err
			}
			first := v.historicalMetrics["current-source-projection-build"]
			text2, tokens2, err := page.ProtectedInput(nil)
			if err != nil {
				return nil, err
			}
			if first != 1 || v.historicalMetrics["current-source-projection-build"] != first || text != text2 || !reflect.DeepEqual(tokens, tokens2) {
				t.Fatal("current source graph rebuilt or changed", v.historicalMetrics)
			}
			if want, err := original.ReattestationAuthority(root); err != nil {
				t.Fatal(err)
			} else if got, err := page.ReattestationAuthority(root); err != nil || got != want {
				t.Fatal("authority identity changed", err)
			}
			t.Logf("QC_CURRENT_GRAPH phase=%d cold_builds=%d warm_additional_builds=%d", phase, first, v.historicalMetrics["current-source-projection-build"]-first)
			return &i18n.SameUnitBridgeResolution{}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	eager := BridgeRepresentationCatalog(root, c).Adapted[0].Adapter.(observedBridgePage)
	eager.proof.historicalMetrics = map[string]int{}
	for phase := 0; phase < 2; phase++ {
		if _, _, err := eager.ProtectedInput(nil); err != nil {
			t.Fatal(err)
		}
		if eager.proof.readonlyOwner || eager.proof.sharedSourceObservation {
			t.Fatal("record verification deferred beyond its eager call")
		}
		if eager.proof.historicalMetrics["BuildGlobal"] != 1 {
			t.Fatal("record rebuilt deterministic historical source graph", eager.proof.historicalMetrics)
		}
	}
	t.Log("QC_CURRENT_RECORD proof_builds=1 additional_full_builds=0 eager_boundaries=yes")
	for _, path := range []string{i18n.GlossaryCorpusPath, GlobalPath, "_content/article.tmpl"} {
		t.Run("concurrent-"+path, func(t *testing.T) {
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, before)
			_, err = withBridgeConstructionEnvironment(root, c, func(scoped *i18n.Catalog) (*i18n.SameUnitBridgeResolution, error) {
				page := scoped.Adapted[0].Adapter.(observedBridgePage)
				if _, _, err := page.ProtectedInput(nil); err != nil {
					return nil, err
				}
				writeFixture(t, root, path, append(bytes.Clone(before), ' '))
				return &i18n.SameUnitBridgeResolution{}, nil
			})
			if err == nil {
				t.Fatal("outer source observation guard skipped", path)
			}
		})
	}
	if _, ok := c.Adapted[0].Adapter.(adoptedPageAdapter); !ok {
		t.Fatal("caller Catalog mutated")
	}
}

func TestBridgeRepresentationEnvironmentCodecFallback(t *testing.T) {
	a, p, raw, reviewed := bridgeGraphInputs(t)
	input := struct {
		Kind          string
		Page          PageAdapter
		Transport     surfacePageTransport
		Raw, Reviewed []byte
	}{"surface-to-ordinary", a, *p, raw, reviewed}
	key, _ := json.Marshal(input)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	finish := contentidentity.BeginMemoization()
	_, err := contentidentity.Transform("sitecontent/ordinary-bridge-representation/v1", key, func() ([]byte, error) { return []byte("{}"), nil })
	finish()
	if err != nil {
		t.Fatal(err)
	}
	finish = contentidentity.BeginMemoization()
	defer finish()
	calls := 0
	r, built, err := deriveBridgeRepresentation(input, func() (*bridgeRepresentationData, error) {
		calls++
		adapter, template, ancestor, err := a.BridgeSurfaceRevision(p, raw, reviewed)
		if err != nil {
			return nil, err
		}
		return &bridgeRepresentationData{adapter.(PageAdapter), template, ancestor}, nil
	})
	if err != nil || !built || calls != 1 {
		t.Fatal("invalid derived codec did not recompute", err, built, calls)
	}
	got, err := r.Page.Restore(string(r.Template))
	if err != nil || !bytes.Equal(bridgeEOF(got), bridgeEOF(reviewed)) {
		t.Fatal("fallback restore changed candidate", err)
	}
}
