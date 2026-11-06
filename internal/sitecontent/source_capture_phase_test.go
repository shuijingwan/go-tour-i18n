package sitecontent

import (
	"bytes"
	"os"
	"testing"
)

func TestReadonlySourceCapturePhase(t *testing.T) {
	for _, phase := range []string{"source", "registration"} {
		t.Run(phase, func(t *testing.T) {
			root := t.TempDir()
			const path = "source.bin"
			before := []byte("exact initial source bytes")
			writeFixture(t, root, path, before)
			for invocation := 0; invocation < 2; invocation++ {
				err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
					v.currentSourcePhase = phase == "source"
					v.registrationReuse = phase == "registration"
					v.historicalMetrics = map[string]int{}
					for i := 0; i < 8; i++ {
						if err := v.observe(Reference{Path: path}); err != nil {
							return err
						}
					}
					if v.historicalMetrics["source-path-capture"] != 1 || v.historicalMetrics["source-path-capture-reuse"] != 7 {
						t.Fatal("repeated traversal recaptured bytes, or prior invocation supplied observation", v.historicalMetrics)
					}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			for _, mutation := range []string{"bytes", "mode", "symlink", "absent-to-present"} {
				t.Run(mutation, func(t *testing.T) {
					writeFixture(t, root, path, before)
					if mutation == "absent-to-present" {
						if err := os.Remove(pathJoin(root, path)); err != nil {
							t.Fatal(err)
						}
					}
					err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
						v.currentSourcePhase = phase == "source"
						v.registrationReuse = phase == "registration"
						if err := v.observe(Reference{Path: path}); err != nil {
							return err
						}
						switch mutation {
						case "bytes", "absent-to-present":
							writeFixture(t, root, path, append(bytes.Clone(before), '!'))
						case "mode":
							if err := os.Chmod(pathJoin(root, path), 0600); err != nil {
								return err
							}
						case "symlink":
							writeFixture(t, root, "other.bin", before)
							if err := os.Remove(pathJoin(root, path)); err != nil {
								return err
							}
							if err := os.Symlink("other.bin", pathJoin(root, path)); err != nil {
								return err
							}
						}
						return v.observe(Reference{Path: path})
					})
					if err == nil {
						t.Fatal("capture reuse bypassed final fresh guard", mutation)
					}
					if err := os.Remove(pathJoin(root, path)); err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}

func TestEagerSourceCaptureStillFresh(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "source.bin", []byte("before"))
	v := successorValidation(root, nil)
	v.currentSourcePhase = true // Phase flag alone cannot defer any observation.
	if err := v.observe(Reference{Path: "source.bin"}); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "source.bin", []byte("after"))
	if err := v.observe(Reference{Path: "source.bin"}); err == nil {
		t.Fatal("eager/default reader acquired readonly reuse")
	}
}
