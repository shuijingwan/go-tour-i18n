package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStructuredReadonlyAggregateBoundary(t *testing.T) {
	root, g, old, parent, receipt := historicalGlobalSuccessorFixture(t)
	baseline := successorValidation(root, nil)
	baseline.historicalMetrics = map[string]int{}
	want, err := projectAdoptedCorpusAtSources(root, g, old, false, baseline)
	if err != nil {
		t.Fatal(err)
	}
	var after *structuredSuccessorValidation
	var gotBytes []byte
	err = withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		after = v
		v.historicalMetrics, v.builds = map[string]int{}, map[Reference]int{}
		got, err := projectAdoptedCorpusAtSources(root, g, old, false, v)
		if err != nil {
			return err
		}
		gotBytes, err = Encode(got)
		if err != nil {
			return err
		}
		if v.historicalMetrics["aggregate-verifies"] != 0 {
			return fmt.Errorf("nested aggregate scan")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, _ := Encode(want)
	if !bytes.Equal(wantBytes, gotBytes) {
		t.Fatal("readonly result bytes/identity changed")
	}
	if baseline.historicalMetrics["aggregate-verifies"] != 9 || after.historicalMetrics["aggregate-verifies"] != 1 || after.builds[parent] != 1 || after.builds[receipt] != 1 || after.historicalMetrics["BuildGlobal"] != 1 || after.historicalMetrics["snapshotFS"] != 1 {
		t.Fatalf("incorrect boundary/reuse: baseline=%v after=%v builds=%v", baseline.historicalMetrics, after.historicalMetrics, after.builds)
	}
	t.Logf("AGGREGATE_COUNTS before=%d after=%d contexts=1 before_rereads=%d after_rereads=%d predecessor_builds=%d historical_builds=%d snapshot_builds=%d", baseline.historicalMetrics["aggregate-verifies"], after.historicalMetrics["aggregate-verifies"], baseline.historicalMetrics["aggregate-rereads"], after.historicalMetrics["aggregate-rereads"], after.builds[parent], after.historicalMetrics["BuildGlobal"], after.historicalMetrics["snapshotFS"])
	// An escaped context is eager again. Default/publication readers never opt
	// into the readonly owner and preserve their existing intermediate checks.
	if after.readonlyOwner || successorValidation(root, nil).readonlyOwner {
		t.Fatal("deferred mode leaked")
	}
	beforeChecks := after.historicalMetrics["aggregate-verifies"]
	if _, err := after.check(parent, 0); err != nil {
		t.Fatal(err)
	}
	if after.historicalMetrics["aggregate-verifies"] != beforeChecks+1 {
		t.Fatal("default eager cache check weakened")
	}
	var a StructuredSourceSuccessor
	if err := readReference(root, parent, &a); err != nil {
		t.Fatal(err)
	}
	body, err := readRegular(root, a.Body.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer writeFixture(t, root, a.Body.Path, body)
	err = withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		if _, err := v.check(parent, 0); err != nil {
			return err
		}
		if _, err := v.check(parent, 0); err != nil {
			return err
		}
		changed := bytes.Clone(body)
		changed[len(changed)/2] ^= 1
		info, err := os.Stat(filepath.Join(root, a.Body.Path))
		if err != nil {
			return err
		}
		writeFixture(t, root, a.Body.Path, changed)
		return os.Chtimes(filepath.Join(root, a.Body.Path), info.ModTime(), info.ModTime())
	})
	if err == nil {
		t.Fatal("mutation after nested cache hit passed outermost boundary")
	}
}

func TestStructuredReadonlyAggregateFailClosed(t *testing.T) {
	for _, kind := range []string{"same-size-mtime", "mode", "absent", "directory", "sha", "fallback", "ordinary-error"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			name := "input/value.json"
			original := []byte("{\"value\":1}\n")
			writeFixture(t, root, name, original)
			writeFixture(t, root, "members/one.txt", []byte("member\n"))
			var checked *structuredSuccessorValidation
			err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
				checked = v
				v.historicalMetrics = map[string]int{}
				if kind == "ordinary-error" {
					return fmt.Errorf("computation failed")
				}
				if kind == "fallback" {
					v.guard.verifiedBytesSize = maxVerifiedBytes
				}
				if err := v.guard.bind(Reference{name, digest(original)}); err != nil {
					return err
				}
				if kind == "sha" {
					return v.guard.bind(Reference{name, digest([]byte("wrong SHA"))})
				}
				if err := v.guard.optional("input/missing.json"); err != nil {
					return err
				}
				if err := v.guard.directory("members"); err != nil {
					return err
				}
				if kind == "fallback" {
					if err := v.guard.bind(Reference{name, digest(original)}); err != nil {
						return err
					}
					if v.guard.counts["sha256:"+name] != 2 || v.guard.verifiedBytesSize != maxVerifiedBytes {
						return fmt.Errorf("SHA fallback not exercised")
					}
				}
				if err := v.verifyObserved(); err != nil {
					return err
				}
				if v.historicalMetrics["aggregate-verifies"] != 0 {
					return fmt.Errorf("premature aggregate")
				}
				switch kind {
				case "mode":
					return os.Chmod(filepath.Join(root, name), 0600)
				case "absent":
					writeFixture(t, root, "input/missing.json", []byte("appeared\n"))
				case "directory":
					writeFixture(t, root, "members/two.txt", []byte("new member\n"))
				default:
					info, err := os.Stat(filepath.Join(root, name))
					if err != nil {
						return err
					}
					writeFixture(t, root, name, []byte("{\"value\":2}\n"))
					return os.Chtimes(filepath.Join(root, name), info.ModTime(), info.ModTime())
				}
				return nil
			})
			if err == nil {
				t.Fatal("observed mutation/invalid reference accepted")
			}
			if checked == nil || checked.readonlyOwner {
				t.Fatal("owner mode not reset on error")
			}
			want := 1
			if kind == "sha" || kind == "ordinary-error" {
				want = 0
			}
			if checked.historicalMetrics["aggregate-verifies"] != want {
				t.Fatalf("aggregate count=%d want=%d", checked.historicalMetrics["aggregate-verifies"], want)
			}
			if kind == "fallback" && checked.guard.counts["sha256:"+name] < 3 {
				t.Fatal("final fallback did not compute SHA")
			}
		})
	}
}
