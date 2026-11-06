package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Explicit opt-in; reconstructs only the current scope. All diagnostic output
// stays in /tmp; no receipt, language artifact or activation is written.
func TestIntegratedSurfacePerformanceReadonly(t *testing.T) {
	label := os.Getenv("GO_TOUR_INTEGRATED_SURFACE_PROFILE")
	if label == "" {
		t.Skip("explicit current-fixture profiling")
	}
	if label != "before" && label != "after" {
		t.Fatal("expected before/after")
	}
	root := repositoryRoot(t)
	done := contentidentity.BeginMemoization()
	defer done()
	const receipt = "data/locale-language/zh-CN/surface/20261018-zh-CN-integrated-surface-004.json"
	b, err := readRegular(root, receipt)
	if err != nil {
		t.Fatal(err)
	}
	var accepted IntegratedSurfaceReceipt
	if err := StrictJSON(b, &accepted); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create("/tmp/go-tour-integrated-surface-" + label + ".cpu")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	v := newClosureValidation(root)
	v.metrics = map[string]int{}
	phases := map[string]string{}
	v.profile = func(name string) func() {
		start := time.Now()
		return func() { phases[name] = time.Since(start).String(); t.Logf("PHASE %s=%s", name, phases[name]) }
	}
	start := time.Now()
	if err := v.currentInputs("zh-CN"); err != nil {
		t.Fatal(err)
	}
	s, err := v.buildIntegratedSurfaceScope(root, "zh-CN", accepted.Scope.Closures)
	if err != nil {
		t.Fatal(err)
	}
	verifyStart := time.Now()
	if err := v.verify(); err != nil {
		t.Fatal(err)
	}
	phases["verify"] = time.Since(verifyStart).String()
	phases["wall"] = time.Since(start).String()
	pprof.StopCPUProfile()
	if !reflect.DeepEqual(*s, accepted.Scope) || s.Identity != "bc856812ae5bb9d5c9f220ca2db2800f2dfb53f2fc9676383f7c3fe846371e37" || len(s.Carried) != 530 || len(s.Reviewed) != 2 {
		t.Fatal("formal scope changed", s.Identity, len(s.Carried), len(s.Reviewed))
	}
	if s.Reviewed[0].ID != "_content/doc/security/fips140.md" || s.Reviewed[1].ID != "_content/learn/books.yaml" {
		t.Fatal("review-required set changed")
	}
	canonical, _ := Encode(s)
	if label == "after" {
		before, err := os.ReadFile("/tmp/go-tour-integrated-surface-before.scope.json")
		if err != nil || !bytes.Equal(before, canonical) {
			t.Fatal("scope JSON changed", err)
		}
	}
	if err := os.WriteFile("/tmp/go-tour-integrated-surface-"+label+".scope.json", canonical, 0600); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for k, n := range v.counts {
		if !strings.HasPrefix(k, "sha256:") {
			counts[k] = n
		}
	}
	report, _ := json.MarshalIndent(struct {
		Phases        map[string]string
		Counts, Calls map[string]int
	}{phases, counts, v.metrics}, "", "  ")
	if err := os.WriteFile("/tmp/go-tour-integrated-surface-"+label+".measurements.json", report, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log(string(report))
}
