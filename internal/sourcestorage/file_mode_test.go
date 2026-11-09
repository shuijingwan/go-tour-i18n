package sourcestorage

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryFileMode(t *testing.T) {
	for _, tc := range []struct{ mode, want fs.FileMode }{
		{0444, 0644}, {0464, 0644}, {0644, 0644}, {0664, 0644},
		{0555, 0755}, {0575, 0755}, {0755, 0755}, {0775, 0755},
	} {
		got, err := RepositoryFileMode(tc.mode)
		if err != nil || got != uint32(tc.want) {
			t.Fatal(tc, got, err)
		}
	}
	for _, mode := range []fs.FileMode{0, 0600, 0700, 0744, 0666, 0777, fs.ModeDir | 0644, fs.ModeSymlink | 0644, fs.ModeNamedPipe | 0644, fs.ModeSetuid | 0644, fs.ModeSetgid | 0755, fs.ModeSticky | 0644} {
		if _, err := RepositoryFileMode(mode); err == nil {
			t.Fatal("unsafe/unsupported mode accepted", mode)
		}
	}
	for _, old := range []fs.FileMode{0644, 0664} {
		if !EquivalentRepositoryFileModes(old, 0644) || EquivalentRepositoryFileModes(old, 0755) {
			t.Fatal("write compatibility changed execution authority", old)
		}
	}
}

func TestFrozenSourceRepositoryModeCompatibility(t *testing.T) {
	root := t.TempDir()
	path := "_content/article.tmpl"
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	want := []byte("<article>Frozen source template</article>\n")
	if err := os.WriteFile(full, want, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(full, 0644); err != nil {
		t.Fatal(err)
	}
	f := FrozenSourceFile{Path: path, GitMode: "100644", LocalMode: 0664, Size: int64(len(want)), SHA256: digest(want)}
	before, _ := Encode(f)
	if got, err := ReadAdoptedSource(root, f); err != nil || !bytes.Equal(got, want) {
		t.Fatal("historical 0664 observation rejected clean 0644 bytes", err)
	}
	for _, mode := range []fs.FileMode{0664, 0644} {
		if err := os.Chmod(full, mode); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadAdoptedSource(root, f); err != nil {
			t.Fatal(err)
		}
	}
	for _, mutation := range []string{"executable", "bytes", "missing", "symlink", "directory"} {
		t.Run(mutation, func(t *testing.T) {
			defer func() {
				_ = os.Remove(full)
				if err := os.WriteFile(full, want, 0644); err != nil {
					t.Fatal(err)
				}
			}()
			switch mutation {
			case "executable":
				if err := os.Chmod(full, 0755); err != nil {
					t.Fatal(err)
				}
			case "bytes":
				changed := bytes.Clone(want)
				changed[1] ^= 1
				if err := os.WriteFile(full, changed, 0644); err != nil {
					t.Fatal(err)
				}
			case "missing", "symlink", "directory":
				if err := os.Remove(full); err != nil {
					t.Fatal(err)
				}
				if mutation == "symlink" {
					other := filepath.Join(root, "same-bytes")
					if err := os.WriteFile(other, want, 0644); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(other, full); err != nil {
						t.Fatal(err)
					}
				} else if mutation == "directory" {
					if err := os.Mkdir(full, 0755); err != nil {
						t.Fatal(err)
					}
				}
			}
			if _, err := ReadAdoptedSource(root, f); err == nil {
				t.Fatal("mode compatibility bypassed frozen source integrity", mutation)
			}
		})
	}
	after, _ := Encode(f)
	if !bytes.Equal(before, after) || f.LocalMode != 0664 {
		t.Fatal("historical observation rewritten")
	}
}
