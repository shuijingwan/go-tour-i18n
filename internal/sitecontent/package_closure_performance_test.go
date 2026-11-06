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

// Explicit opt-in: builds the real successor read-only, writing profiles only
// to /tmp. Never finalizes a closure or exports a Generation Bundle.
func TestPackageClosurePerformanceReadonly(t *testing.T) {
	label := os.Getenv("GO_TOUR_CLOSURE_PROFILE")
	if label == "" {
		t.Skip("explicit exact-fixture read-only performance/correctness check")
	}
	if label != "before" && label != "after" {
		t.Fatal("profile label must be before/after")
	}
	root := repositoryRoot(t)
	page := Reference{"data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-015/finalization.json", "530374a7aa563c103f975940b480f998fbd315541c56438e21d2557e22275b1d"}
	delta := Reference{"data/locale-language/zh-CN/structured-generation/e8111c186bb604ee0cea0c994936ad169a4a44374b710b8e4f52bbad8cb2be7f.json", "f0c71de4623b39981ee33ee8798770bcc7fc03a7263948a30993756f9864e6b1"}
	accepted := Reference{"data/site-workflow/zh-CN/learn-docs-v1/closure/fe866b9aa3a9b9ce2b26757b2bd6da513cabe41fe1f414ef6430ab6fa5202db9.json", "47588d761b615abb7f435b488c1058a83cc06e2b833d9118ccac5282a692e88e"}
	var want PackageClosure
	if err := readReference(root, accepted, &want); err != nil {
		t.Fatal(err)
	}
	done := contentidentity.BeginMemoization() // same existing scope as CLI
	defer done()
	f, err := os.Create("/tmp/go-tour-package-closure-" + label + ".cpu")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	v := newClosureValidation(root)
	start := time.Now()
	if err := v.currentInputs("zh-CN"); err != nil {
		t.Fatal(err)
	}
	inputsTime := time.Since(start)
	buildStart := time.Now()
	got, files, err := v.buildPackageClosure(root, "zh-CN", "learn-docs-v1", &page, []Reference{delta})
	buildTime := time.Since(buildStart)
	if err != nil {
		t.Fatal(err)
	}
	verifyStart := time.Now()
	if err := v.verify(); err != nil {
		t.Fatal(err)
	}
	verifyTime := time.Since(verifyStart)
	elapsed := time.Since(start)
	pprof.StopCPUProfile()
	if !reflect.DeepEqual(*got, want) || len(files) != 56 {
		t.Fatal("real successor closure identity/inventory changed", got.Identity, len(files))
	}
	for _, ref := range got.Files {
		if digest(files[ref.Path]) != ref.SHA256 {
			t.Fatal("projected target changed", ref.Path)
		}
	}
	canonical, _ := Encode(got)
	if label == "after" {
		before, err := os.ReadFile("/tmp/go-tour-package-closure-before.closure.json")
		if err != nil || !bytes.Equal(before, canonical) {
			t.Fatal("optimization changed exact closure JSON", err)
		}
	}
	if err := os.WriteFile("/tmp/go-tour-package-closure-"+label+".closure.json", canonical, 0600); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for key, count := range v.counts {
		if !strings.HasPrefix(key, "sha256:") {
			counts[key] = count
		}
	}
	report := struct {
		Wall, CurrentInputs, Build, Verify string
		Counts                             map[string]int
	}{elapsed.String(), inputsTime.String(), buildTime.String(), verifyTime.String(), counts}
	b, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile("/tmp/go-tour-package-closure-"+label+".measurements.json", b, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
}

func TestPackageClosureCurrentReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_CLOSURE_PROFILE") != "after" {
		t.Skip("explicit exact-fixture canonical current checker")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	ref := Reference{"data/site-workflow/zh-CN/learn-docs-v1/closure/fe866b9aa3a9b9ce2b26757b2bd6da513cabe41fe1f414ef6430ab6fa5202db9.json", "47588d761b615abb7f435b488c1058a83cc06e2b833d9118ccac5282a692e88e"}
	c, files, err := CheckPackageClosure(repositoryRoot(t), ref)
	if err != nil || c.Identity != "fe866b9aa3a9b9ce2b26757b2bd6da513cabe41fe1f414ef6430ab6fa5202db9" || len(files) != 56 || c.StructuredBaseline == nil || len(c.Structured) != 1 {
		t.Fatal("existing formal successor current-check", c, len(files), err)
	}
}
