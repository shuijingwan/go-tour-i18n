package sitecontent

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestMembershipClosureEarlyFailureReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_MEMBERSHIP_CLOSURE_PROFILE") != "after" {
		t.Skip("opt-in readonly early-failure measurement")
	}
	start := time.Now()
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	v := newClosureValidation(root)
	if err := v.currentInputs("zh-CN"); err != nil {
		t.Fatal(err)
	}
	g, err := v.loadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	base, err := adoptionBase(root)
	if err != nil {
		t.Fatal(err)
	}
	c, err := v.inventoryCatalog()
	if err != nil {
		t.Fatal(err)
	}
	// Reproduce the original missing-registration condition only in this
	// private context, without changing any source/receipt/formal bytes.
	missing, _ := packageByID(base, "site-v2-shell")
	c.HistoricalTranslationPackages = nil
	for _, d := range v.catalog.HistoricalTranslationPackages {
		if d.ID != missing.ID || d.AuthoritySHA256 != missing.Identity {
			c.HistoricalTranslationPackages = append(c.HistoricalTranslationPackages, d)
		}
	}
	v.catalog = c
	refs := []Reference{}
	for _, name := range []string{"learn-docs-v1-qc-021", "docs-entry-v1-qc-002"} {
		ref, err := ReferenceFile(root, "data/quality-check-snapshots/zh-CN/"+name+"/finalization.json")
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, ref)
	}
	inputs, err := v.compositionInputs(root, "learn-docs-v1", refs)
	if err != nil || g.Membership == nil {
		t.Fatal(err)
	}
	_, _, err = v.buildPackageClosureInputs(root, "zh-CN", "learn-docs-v1", nil, nil, inputs)
	if err == nil || !strings.Contains(err.Error(), "unregistered historical package authority site-v2-shell") {
		t.Fatal("expected original structural failure", err)
	}
	for key, count := range v.counts {
		if strings.HasPrefix(key, "closure:") || strings.HasPrefix(key, "generation:") || key == "current-corpus" || key == "current-review" {
			if count != 0 {
				t.Fatal("expensive graph entered", key, count)
			}
		}
	}
	b, _ := json.MarshalIndent(map[string]interface{}{"wall_seconds": time.Since(start).Seconds(), "error": err.Error(), "counts": v.counts, "mutation": "none", "fault_injection": "private descriptor omitted, original disk bytes retained"}, "", "  ")
	if err := os.WriteFile("/tmp/package-closure-perf-20261023/after-early-failure.json", b, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestHistoricalPackageRegistrationProfile(t *testing.T) {
	label := os.Getenv("GO_TOUR_MEMBERSHIP_CLOSURE_PROFILE")
	if label == "" {
		t.Skip("opt-in readonly installed registration diagnostic")
	}
	start := time.Now()
	root := repositoryRoot(t)
	done := contentidentity.BeginMemoization()
	defer done()
	base, err := adoptionBase(root)
	if err != nil {
		t.Fatal(err)
	}
	p, err := packageByID(base, "site-v2-shell")
	if err != nil {
		t.Fatal(err)
	}
	c, err := checkedCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	_, resolveErr := i18n.HistoricalTranslationPackageCatalog(c, p.ID, p.Identity)
	report := fmt.Sprintf("wall=%s\nrequired_package=%s\nrequired_authority=%s\nresolve_error=%v\n", time.Since(start), p.ID, p.Identity, resolveErr)
	if err := os.WriteFile("/tmp/package-closure-perf-20261023/"+label+"-registration.txt", []byte(report), 0600); err != nil {
		t.Fatal(err)
	}
	if label == "after" && resolveErr != nil {
		t.Fatal(resolveErr)
	}
}

// Explicit readonly real-authority builder. Only diagnostics go to /tmp;
// the closure publisher and every formal language writer are excluded.
func TestMembershipClosureReadonlyProfile(t *testing.T) {
	label := os.Getenv("GO_TOUR_MEMBERSHIP_CLOSURE_PROFILE")
	if label == "" {
		t.Skip("opt-in readonly installed authority proof")
	}
	if label != "before" && label != "after" {
		t.Fatal("expected before/after")
	}
	const dir = "/tmp/package-closure-perf-20261023/"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	root := repositoryRoot(t)
	done := contentidentity.BeginMemoization()
	defer done()
	f, err := os.Create(dir + label + ".cpu")
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
	phases := map[string]time.Duration{}
	v.profile = func(name string) func() {
		start := time.Now()
		return func() { phases[name] += time.Since(start) }
	}
	start := time.Now()
	report := map[string]interface{}{"mutation": "none"}
	defer func() {
		pprof.StopCPUProfile()
		report["wall_seconds"] = time.Since(start).Seconds()
		report["phases"] = phases
		report["counts"] = v.counts
		report["calls"] = v.metrics
		b, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(dir+label+"-profile.json", b, 0600); err != nil {
			t.Error(err)
		}
	}()
	refs := []Reference{}
	for _, name := range []string{
		"data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-021/finalization.json",
		"data/quality-check-snapshots/zh-CN/docs-entry-v1-qc-002/finalization.json",
	} {
		ref, err := ReferenceFile(root, name)
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, ref)
	}
	err = v.currentInputs("zh-CN")
	var closure *PackageClosure
	var files map[string][]byte
	if err == nil {
		inputs, inputErr := v.compositionInputs(root, "learn-docs-v1", refs)
		err = inputErr
		if err == nil {
			done := v.phase("build")
			closure, files, err = v.buildPackageClosureInputs(root, "zh-CN", "learn-docs-v1", nil, nil, inputs)
			done()
		}
	}
	if err == nil {
		done := v.phase("verify")
		err = v.verify()
		done()
	}
	if err != nil {
		report["error"] = err.Error()
		if label == "before" {
			t.Log("observed baseline failure:", err)
			return
		}
		t.Fatal(err)
	}
	if len(closure.PageComposition) != 2 || len(files) != 58 || closure.Membership == nil {
		t.Fatal("expected exact unified 51 Page + 7 structured files")
	}
	report["closure_identity"] = closure.Identity
	report["file_count"] = len(files)
	report["historical_package_authority"] = "PASS"
	report["page_composition"] = "PASS"
	report["structured_baseline"] = "PASS"
	report["current_validation"] = "PASS"
	b, _ := Encode(closure)
	if err := os.WriteFile(dir+label+"-planned-closure.json", b, 0600); err != nil {
		t.Fatal(err)
	}
}
