package sitecontent

import (
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestWorkflowDocumentEnvironmentCrossProcess(t *testing.T) {
	if root := os.Getenv("GO_TOUR_WORKFLOW_GRAPH_ROOT"); root != "" {
		finish := contentidentity.BeginMemoization()
		defer finish()
		g, err := LoadCurrent(root)
		if err != nil {
			t.Fatal(err)
		}
		v := successorValidation(root, nil)
		v.historicalMetrics = map[string]int{}
		docs, raw, err := sharedWorkflowDocuments(root, g, "learn-docs-v1", v)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(map[string]any{"builds": v.historicalMetrics["workflow-package-graph-build"], "identity": identity(packageRepresentationData{docs, raw})})
		t.Logf("WORKFLOW_GRAPH=%s", b)
		return
	}
	root := siteFixture(t, "current-authorization-fresh-fixture")
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	want := identity(packageRepresentationData{docs, raw})
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	for phase := 0; phase < 2; phase++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestWorkflowDocumentEnvironmentCrossProcess$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), "GO_TOUR_WORKFLOW_GRAPH_ROOT="+root)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(string(b), err)
		}
		found := false
		for _, line := range strings.Split(string(b), "\n") {
			at := strings.Index(line, "WORKFLOW_GRAPH=")
			if at < 0 {
				continue
			}
			var row struct {
				Builds   int
				Identity string
			}
			if json.Unmarshal([]byte(line[at+len("WORKFLOW_GRAPH="):]), &row) != nil {
				t.Fatal(line)
			}
			expected := 1
			if phase == 1 {
				expected = 0
			}
			if row.Builds != expected || row.Identity != want {
				t.Fatal("workflow graph changed or rebuilt", phase, row)
			}
			t.Logf("WORKFLOW_GRAPH phase=%d builds=%d identity=%s", phase, row.Builds, row.Identity)
			found = true
		}
		if !found {
			t.Fatal(string(b))
		}
	}
}

func TestWorkflowDocumentEnvironmentFreshRegistry(t *testing.T) {
	root := siteFixture(t, "current-authorization-fresh-fixture")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	want, raw, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	finish := contentidentity.BeginMemoization()
	got, next, err := sharedWorkflowDocuments(root, g, "learn-docs-v1")
	finish()
	if err != nil || !reflect.DeepEqual(want, got) || !reflect.DeepEqual(raw, next) {
		t.Fatal("canonical workflow identity changed", err)
	}
	original, err := readRegular(root, PageRegistryPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, PageRegistryPath, []byte("invalid current registry"))
	finish = contentidentity.BeginMemoization()
	defer finish()
	if _, _, err := sharedWorkflowDocuments(root, g, "learn-docs-v1"); err == nil {
		t.Fatal("registry observation skipped")
	}
	writeFixture(t, root, PageRegistryPath, original)
}
