package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestManifestExactPathObservation(t *testing.T) {
	for _, source := range []string{"_content/example.md", "internal/web/example.go"} {
		t.Run(source, func(t *testing.T) {
			root := t.TempDir()
			adopted := adoptedSourcePath(source)
			writeFixture(t, root, source, []byte("source bytes\n"))
			if adopted != source {
				writeFixture(t, root, adopted, []byte("frozen bytes\n"))
			}
			before, after := newClosureValidation(root), newClosureValidation(root)
			before.disabled, after.disabled = true, true // Count actual SHA-backed reads.
			for _, name := range []string{source, adopted} {
				if err := before.optional(name); err != nil {
					t.Fatal(err)
				}
			}
			calls := []string{}
			if err := observeManifestSourcePaths(source, func(name string) error {
				calls = append(calls, name)
				return after.optional(name)
			}); err != nil {
				t.Fatal(err)
			}
			want := []string{source}
			if adopted != source {
				want = append(want, adopted)
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("observed paths=%v want=%v", calls, want)
			}
			if !reflect.DeepEqual(before.deps, after.deps) || !reflect.DeepEqual(before.modes, after.modes) || !reflect.DeepEqual(before.absent, after.absent) {
				t.Fatal("dependency identity/mode/absence changed")
			}
			x, _ := Encode(before.deps)
			y, _ := Encode(after.deps)
			if !bytes.Equal(x, y) || identity(before.deps) != identity(after.deps) {
				t.Fatal("dependency bytes/identity changed")
			}
			if source == adopted && (before.counts["sha256:"+source] != 2 || after.counts["sha256:"+source] != 1) {
				t.Fatal("exact pair was not reduced from two validations to one")
			}
			if err := after.verify(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestManifestExactPathFailClosed(t *testing.T) {
	for _, change := range []string{"adopted-bytes", "adopted-removed", "adopted-mode", "adopted-appeared", "same-path-appeared"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			source := "internal/web/example.go"
			if change == "same-path-appeared" {
				source = "_content/missing.md"
			} else {
				writeFixture(t, root, source, []byte("original\n"))
			}
			adopted := adoptedSourcePath(source)
			if change != "adopted-appeared" && change != "same-path-appeared" {
				writeFixture(t, root, adopted, []byte("original\n"))
			}
			v := newClosureValidation(root)
			calls := 0
			if err := observeManifestSourcePaths(source, func(name string) error {
				calls++
				return v.optional(name)
			}); err != nil {
				t.Fatal(err)
			}
			if change == "same-path-appeared" && (calls != 1 || !v.absent[source]) {
				t.Fatal("same-path absence was not observed exactly once")
			}
			switch change {
			case "adopted-removed":
				if err := os.Remove(filepath.Join(root, adopted)); err != nil {
					t.Fatal(err)
				}
			case "adopted-mode":
				if err := os.Chmod(filepath.Join(root, adopted), 0600); err != nil {
					t.Fatal(err)
				}
			default:
				writeFixture(t, root, adopted, []byte("modified\n"))
			}
			if err := v.verify(); err == nil {
				t.Fatal("dependency drift accepted", change)
			}
		})
	}
}
