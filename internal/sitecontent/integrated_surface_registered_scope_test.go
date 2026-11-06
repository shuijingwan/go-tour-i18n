package sitecontent

import (
	"os"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// Explicit real-fixture acceptance of the canonical read-only scope builder.
// Output is diagnostic JSON in /tmp, never a saved Reviewer/Surface receipt.
func TestIntegratedSurfaceRegisteredPackageReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_SURFACE_REGISTERED_SCOPE_READONLY") != "1" {
		t.Skip("explicit current two-closure Surface acceptance")
	}
	done := contentidentity.BeginMemoization()
	defer done()
	root := repositoryRoot(t)
	closures := []Reference{
		{"data/site-workflow/zh-CN/learn-docs-v1/closure/fe866b9aa3a9b9ce2b26757b2bd6da513cabe41fe1f414ef6430ab6fa5202db9.json", "47588d761b615abb7f435b488c1058a83cc06e2b833d9118ccac5282a692e88e"},
		{"data/site-workflow/zh-CN/site-v2-shell/closure/1f5e9f0fb28b2098c1fff90778f652f9541cfc6ff4d8c75e72ec2093f70dc926.json", "38746745af4821aec945899c730b62b61be53de3b450f4ad8c6609f21e98336f"},
	}
	s, err := BuildIntegratedSurfaceScope(root, "zh-CN", closures)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.Closures, closures) || len(s.Closures) != 2 {
		t.Fatal("unexpected prospective closures")
	}
	tourComplete, tourCarried, shellBound := false, false, false
	for _, p := range s.ExistingCoverage.Packages {
		if p.Package == "tour-v1" && p.State == "complete" {
			tourComplete = true
		}
	}
	for _, a := range s.Carried {
		if a.Package == "tour-v1" && a.State == "carry" {
			tourCarried = true
		}
	}
	for _, a := range append(append([]SurfaceAsset{}, s.Carried...), s.Reviewed...) {
		if a.Package != "site-v2-shell" || a.ID != "_content/index.md" {
			continue
		}
		for _, artifact := range a.CandidateArtifacts {
			if artifact.SnapshotID == "site-v2-shell-qc-002" && artifact.Unit.RepresentationReconciliation != nil {
				shellBound = true
			}
		}
	}
	if !tourComplete || !tourCarried || !shellBound {
		t.Fatal("missing retained Tour carry or reconciled shell binding", tourComplete, tourCarried, shellBound)
	}
	b, err := Encode(s)
	if err != nil {
		t.Fatal(err)
	}
	const output = "/tmp/zh-CN-integrated-surface-registered-scope-20261018.json"
	if err := os.WriteFile(output, b, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("SURFACE_SCOPE=PASS CLOSURES=2 TOUR_CARRY=PASS SHELL_RECONCILIATION=PASS EVIDENCE=%s SHA256=%s", output, digest(b))
}
