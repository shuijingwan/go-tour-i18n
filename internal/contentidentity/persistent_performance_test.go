package contentidentity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// Bounded parser evidence only: one existing formal JSON input, not a workflow
// replay or a slow command baseline. Each pass reopens and hashes the source.
func TestPersistentExistingJSONPerformance(t *testing.T) {
	path, out := os.Getenv("GO_TOUR_PURE_JSON_INPUT"), os.Getenv("GO_TOUR_PURE_JSON_OUT")
	if path == "" || out == "" {
		t.Skip("explicit readonly JSON and /tmp evidence required")
	}
	if !filepath.IsAbs(out) || filepath.Dir(out) == "/" {
		t.Fatal("explicit evidence required")
	}
	// Use an untyped parsed graph to measure the shared strict token walk while
	// preserving all actual fields. Consumers retain their own exact typed keys.
	var expected map[string]any
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := strictJSONDecode(b, &expected); err != nil {
		t.Fatal(err)
	}
	sha := exactSHA256(b)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	metrics := map[string]any{"input_sha256": sha, "input_bytes": len(b)}
	for _, phase := range []string{"uncached-pure-parse", "cold", "warm-new-invocation"} {
		fresh, err := os.ReadFile(path)
		if err != nil || exactSHA256(fresh) != sha {
			t.Fatal("formal input changed", err)
		}
		finish := BeginMemoization()
		if phase == "uncached-pure-parse" {
			activeMemo().store = nil
		}
		var got map[string]any
		start := time.Now()
		err = StrictJSON(fresh, &got)
		elapsed := time.Since(start).Seconds()
		finish()
		if err != nil || !reflect.DeepEqual(got, expected) {
			t.Fatal("cold/warm value differs", phase, err)
		}
		metrics[phase] = elapsed
		t.Logf("%s seconds=%.6f", phase, elapsed)
	}
	output, _ := json.MarshalIndent(metrics, "", "  ")
	if err := os.WriteFile(out, append(output, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
