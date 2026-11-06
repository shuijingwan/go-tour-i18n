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
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func historicalEnvironmentInput(t *testing.T, c *i18n.Catalog, f HistoricalPageRepresentations) historicalPageEnvironmentInput {
	t.Helper()
	d := c.TranslationPackages[0]
	in := historicalPageEnvironmentInput{Frozen: f, Package: d.ID, Authority: d.AuthoritySHA256}
	for _, u := range d.Units {
		a, err := replacementPageAdapter(u.Adapter)
		if err != nil {
			t.Fatal(err)
		}
		u.Adapter = nil
		in.Current = append(in.Current, historicalPageBinding{u, a})
	}
	return in
}

func historicalEnvironmentCatalog(t *testing.T, root string, in historicalPageEnvironmentInput) (*i18n.Catalog, []PageAdapter) {
	t.Helper()
	c := &i18n.Catalog{}
	d := i18n.TranslationPackageDescriptor{ID: in.Package, AuthoritySHA256: in.Authority}
	var next []PageAdapter
	for _, b := range in.Current {
		raw, err := readRegular(root, b.Inventory.Unit.SourcePath)
		if err != nil {
			t.Fatal(err)
		}
		b.Inventory.Unit.Source, b.Inventory.Unit.SourceSHA256 = raw, digest(raw)
		b.Page.Source = raw
		b.Inventory.Adapter = b.Page
		d.Units = append(d.Units, b.Inventory)
		next = append(next, b.Page)
	}
	if err := c.RegisterTranslationPackage(d); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(d.ID); err != nil {
		t.Fatal(err)
	}
	return c, next
}

func TestHistoricalPageEnvironmentCrossProcess(t *testing.T) {
	if root := os.Getenv("GO_TOUR_HISTORICAL_ENV_FIXTURE"); root != "" {
		finish := contentidentity.BeginMemoization()
		defer finish()
		b, err := readRegular(root, "engineering/historical-input.json")
		if err != nil {
			t.Fatal(err)
		}
		var in historicalPageEnvironmentInput
		if json.Unmarshal(b, &in) != nil {
			t.Fatal("fixture input")
		}
		c, next := historicalEnvironmentCatalog(t, root, in)
		start := time.Now()
		r, built, err := historicalPageProjectionWithCounter(&in.Frozen, &c.TranslationPackages[0], next)
		if err != nil {
			t.Fatal(err)
		}
		ref := freezeReplacementFixture(t, root, in.Frozen)
		resolved, err := resolvePageReplacements(root, c, ref)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(resolved.Units, r.Units) || !reflect.DeepEqual(resolved.HistoricalCatalog.TranslationPackages[0], r.Descriptor) {
			t.Fatal("full historical projection differs")
		}
		raw, _ := json.Marshal(map[string]any{"built": built, "identity": identity(r.Units), "seconds": time.Since(start).Seconds()})
		t.Logf("HISTORICAL_ENV=%s", raw)
		return
	}
	root, _, c, f := replacementPageFixture(t)
	in := historicalEnvironmentInput(t, c, f)
	b, _ := json.Marshal(in)
	writeFixture(t, root, "engineering/historical-input.json", b)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var identity string
	for phase := 0; phase < 2; phase++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestHistoricalPageEnvironmentCrossProcess$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), "GO_TOUR_HISTORICAL_ENV_FIXTURE="+root)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(string(b), err)
		}
		found := false
		for _, line := range strings.Split(string(b), "\n") {
			at := strings.Index(line, "HISTORICAL_ENV=")
			if at < 0 {
				continue
			}
			var row struct {
				Built    bool
				Identity string
				Seconds  float64
			}
			if err := json.Unmarshal([]byte(line[at+len("HISTORICAL_ENV="):]), &row); err != nil {
				t.Fatal(err)
			}
			if row.Built != (phase == 0) {
				t.Fatal("complete historical projection not reused", phase, row.Built)
			}
			if phase == 0 {
				identity = row.Identity
			} else if identity != row.Identity {
				t.Fatal("historical graph identity changed")
			}
			t.Logf("HISTORICAL_ENV phase=%d built=%v seconds=%.6f identity=%s", phase, row.Built, row.Seconds, row.Identity)
			found = true
		}
		if !found {
			t.Fatal(string(b))
		}
	}
}

func TestHistoricalPageEnvironmentFreshInputs(t *testing.T) {
	root, _, c, f := replacementPageFixture(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	ref := freezeReplacementFixture(t, root, f)
	run := func(c *i18n.Catalog) (*i18n.TranslationUnitReplacementResolution, error) {
		finish := contentidentity.BeginMemoization()
		defer finish()
		return resolvePageReplacements(root, c, ref)
	}
	want, err := run(c)
	if err != nil {
		t.Fatal(err)
	}
	equal := func(a, b *i18n.TranslationUnitReplacementResolution) bool {
		if a == nil || b == nil {
			return false
		}
		data := func(c *i18n.Catalog) *i18n.Catalog {
			x := *c
			x.ReplacementResolver, x.BridgeResolver = nil, nil
			x.SurfaceAuthorityReader, x.HistoricalContextReader, x.PreviewFailureReader = nil, nil, nil
			return &x
		}
		return reflect.DeepEqual(a.Units, b.Units) && reflect.DeepEqual(data(a.HistoricalCatalog), data(b.HistoricalCatalog))
	}
	got, err := run(c)
	if err != nil || !equal(want, got) {
		t.Fatal("warm graph differs", err)
	}
	got.Units["caller-mutation"] = i18n.TranslationUnitReplacement{}
	if again, err := run(c); err != nil || !equal(want, again) {
		t.Fatal("caller changed persistent graph", err)
	}
	original, err := readRegular(root, ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, ref.Path, append(bytes.Clone(original), ' '))
	if _, err := run(c); err == nil {
		t.Fatal("fresh frozen reference skipped")
	}
	writeFixture(t, root, ref.Path, original)
	input := historicalEnvironmentInput(t, c, f)
	path := input.Current[0].Inventory.Unit.SourcePath
	before, _ := readRegular(root, path)
	writeFixture(t, root, path, append(bytes.Clone(before), ' '))
	fresh, _ := historicalEnvironmentCatalog(t, root, input)
	if _, err := run(fresh); err == nil {
		t.Fatal("changed current representation reused old graph")
	}
	writeFixture(t, root, path, before)
	// Other package/catalog data is supplied fresh, never serialized inside
	// this affected-package representation component.
	c.Pages = []i18n.Page{{ID: "fresh-unrelated-source-view"}}
	if r, err := run(c); err != nil || !reflect.DeepEqual(c.Pages, r.HistoricalCatalog.Pages) {
		t.Fatal("unrelated fresh Catalog lost", err)
	}
}

func TestHistoricalPageEnvironmentCodecFallback(t *testing.T) {
	root, _, c, f := replacementPageFixture(t)
	ref := freezeReplacementFixture(t, root, f)
	want, err := resolvePageReplacements(root, c, ref)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	in := historicalEnvironmentInput(t, c, f)
	key, _ := json.Marshal(in)
	finish := contentidentity.BeginMemoization()
	_, err = contentidentity.Transform("sitecontent/historical-page-environment/v1", key, func() ([]byte, error) { return []byte("invalid historical component codec"), nil })
	finish()
	if err != nil {
		t.Fatal(err)
	}
	finish = contentidentity.BeginMemoization()
	defer finish()
	got, err := resolvePageReplacements(root, c, ref)
	if err != nil || !reflect.DeepEqual(want.Units, got.Units) || !reflect.DeepEqual(want.HistoricalCatalog.TranslationPackages, got.HistoricalCatalog.TranslationPackages) {
		t.Fatal("derived historical codec prevented recomputation", err)
	}
}
