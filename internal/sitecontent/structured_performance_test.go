package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Opt-in profiling of the read-only planner, never of an already accepted
// Generation workflow. Profiles and semantic differential evidence stay in /tmp.
func TestStructuredPlanPerformanceReadonly(t *testing.T) {
	label := os.Getenv("GO_TOUR_STRUCTURED_PROFILE")
	if label == "" {
		t.Skip("explicit read-only structured performance diagnosis")
	}
	if label != "before" && label != "after" {
		t.Fatal("profile label must be before/after")
	}
	root := repositoryRoot(t)
	done := contentidentity.BeginMemoization()
	defer done()
	f, err := os.Create("/tmp/go-tour-structured-" + label + ".cpu")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	p, err := PlanStructuredAssets(root, "zh-CN")
	elapsed := time.Since(start)
	pprof.StopCPUProfile()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	name := "/tmp/go-tour-structured-" + label + ".plan.json"
	if err := os.WriteFile(name, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	if label == "after" {
		before, err := os.ReadFile("/tmp/go-tour-structured-before.plan.json")
		if err != nil || !bytes.Equal(before, encoded) {
			t.Fatal("planner semantic result changed", err)
		}
	}
	alloc, err := os.Create("/tmp/go-tour-structured-" + label + ".allocs")
	if err != nil {
		t.Fatal(err)
	}
	defer alloc.Close()
	if err := pprof.Lookup("allocs").WriteTo(alloc, 0); err != nil {
		t.Fatal(err)
	}
	t.Logf("planner wall=%s identity=%s expected=%v profile=%s", elapsed, p.Identity, p.Expected, filepath.Base(name))
}
