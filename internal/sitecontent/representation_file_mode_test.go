package sitecontent

import (
	"bytes"
	"os"
	"testing"
)

func TestRepresentationRepositoryModeDependencies(t *testing.T) {
	old := []ShellRepresentationInput{{Reference: Reference{"data/exact-source.json", digest([]byte("source"))}, Mode: 0664}}
	current := append([]ShellRepresentationInput(nil), old...)
	current[0].Mode = 0644
	for _, changes := range []func([]ShellRepresentationInput, []ShellRepresentationInput) ([]ShellRepresentationDependencyChange, error){shellDependencyChanges, sharedDependencyChanges} {
		got, err := changes(old, current)
		if err != nil || len(got) != 0 || !sameRepresentationInputs(old, current) {
			t.Fatal("write observation falsely became semantic/dependency drift", got, err)
		}
		for _, mode := range []uint32{0755, 0775, 0666, 0600, uint32(os.ModeSymlink) | 0644} {
			bad := append([]ShellRepresentationInput(nil), current...)
			bad[0].Mode = mode
			if _, err := changes(old, bad); err == nil {
				t.Fatal("unsafe or executable mode accepted", mode)
			}
		}
		bad := append([]ShellRepresentationInput(nil), current...)
		bad[0].Reference.SHA256 = digest([]byte("changed source"))
		if _, err := changes(old, bad); err == nil || sameRepresentationInputs(old, bad) {
			t.Fatal("non-parser SHA drift accepted")
		}
		bad[0] = ShellRepresentationInput{Reference: Reference{old[0].Reference.Path, digest(nil)}, Absent: true}
		if _, err := changes(old, bad); err == nil {
			t.Fatal("present/absent drift accepted")
		}
		if _, err := changes(bad, old); err == nil {
			t.Fatal("absent/present drift accepted")
		}
	}
}

func legacyWriteObservations(inputs []ShellRepresentationInput) []ShellRepresentationInput {
	out := append([]ShellRepresentationInput(nil), inputs...)
	for i := range out {
		if !out[i].Absent && out[i].Mode == 0644 {
			out[i].Mode = 0664
		}
	}
	return out
}

func TestShellRepresentationRepositoryModeHistoricalReceipts(t *testing.T) {
	root, g, base := shellReattestationFixture(t)
	// Install a valid immutable legacy-format fixture, never edit either saved
	// receipt in place. Its old raw mode observations remain identity-bound.
	legacy := *base
	legacy.Inputs = legacyWriteObservations(base.Inputs)
	for i, in := range legacy.Inputs {
		if in.Reference.Path == repositoryModeImplementation {
			legacy.Inputs = append(legacy.Inputs[:i], legacy.Inputs[i+1:]...)
			break
		}
	}
	legacy.Identity = ""
	legacy.Identity = identity(legacy)
	ref := shellRepresentationReceipt(&legacy)
	if err := saveImmutable(root, ref.Path, legacy); err != nil {
		t.Fatal(err)
	}
	pointer, _ := Encode(ref)
	writeFixture(t, root, ShellRepresentationCurrent, pointer)
	before, _ := readRegular(root, ref.Path)
	if _, err := checkShellRepresentation(root, g, ref, successorValidation(root, nil), map[Reference]bool{}, 0); err != nil {
		t.Fatal("0664 historical input rejected 0644 checkout", err)
	}
	noop, _ := reattestationFixturePlan(t, root, g)
	if !noop.BaseCurrent || noop.Eligible {
		t.Fatal("mode-only difference requested new authority")
	}
	driftShellContract(t, root)
	if _, _, err := currentShellRepresentation(root, g, successorValidation(root, nil), false); err == nil {
		t.Fatal("SHA evolution bypassed additive authority")
	}
	plan, _ := reattestationFixturePlan(t, root, g)
	a := plan.Authority
	a.CurrentInputs = legacyWriteObservations(a.CurrentInputs)
	a.ValidationInputs = legacyWriteObservations(a.ValidationInputs)
	var err error
	a.Changes, err = shellDependencyChanges(a.OldInputs, a.CurrentInputs)
	if err != nil {
		t.Fatal(err)
	}
	a.Identity = ""
	a.Identity = identity(a)
	ar := shellReattestationReceipt(&a)
	if err := saveImmutable(root, ar.Path, a); err != nil {
		t.Fatal(err)
	}
	b, _ := Encode(ar)
	writeFixture(t, root, plan.CurrentPath, b)
	aBefore, _ := readRegular(root, ar.Path)
	if _, _, err := currentShellRepresentation(root, g, successorValidation(root, nil), false); err != nil {
		t.Fatal("historical mode envelope lost currentness despite exact SHA", err)
	}
	for _, mutation := range []string{"executable", "symlink", "directory", "missing", "bytes", "validation-sha"} {
		t.Run(mutation, func(t *testing.T) {
			path := "docs/SHELL_REPRESENTATION_SUCCESSOR.md"
			if mutation == "validation-sha" {
				path = repositoryModeImplementation
			}
			before, _ := readRegular(root, path)
			defer func() {
				_ = os.Remove(pathJoin(root, path))
				writeFixture(t, root, path, before)
			}()
			switch mutation {
			case "executable":
				if err := os.Chmod(pathJoin(root, path), 0755); err != nil {
					t.Fatal(err)
				}
			case "symlink", "directory", "missing":
				if err := os.Remove(pathJoin(root, path)); err != nil {
					t.Fatal(err)
				}
				if mutation == "symlink" {
					writeFixture(t, root, "data/same-mode-bytes", before)
					if err := os.Symlink(pathJoin(root, "data/same-mode-bytes"), pathJoin(root, path)); err != nil {
						t.Fatal(err)
					}
				} else if mutation == "directory" {
					if err := os.Mkdir(pathJoin(root, path), 0755); err != nil {
						t.Fatal(err)
					}
				}
			case "bytes", "validation-sha":
				changed := bytes.Clone(before)
				changed[0] ^= 1
				writeFixture(t, root, path, changed)
			}
			if _, _, err := currentShellRepresentation(root, g, successorValidation(root, nil), false); err == nil {
				t.Fatal("repository-mode compatibility weakened currentness", mutation)
			}
		})
	}
	for path, want := range map[string][]byte{ref.Path: before, ar.Path: aBefore} {
		got, err := readRegular(root, path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("immutable receipt bytes/identity changed", path, err)
		}
	}
}
