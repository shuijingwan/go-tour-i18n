package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// These tests exercise the existing eager proof graph, independently of the
// caller optimization. They run unchanged on baseline and optimized readers.
func TestCorpusProofOwnerBoundaries(t *testing.T) {
	root, _, _, parent, child := historicalGlobalSuccessorFixture(t)
	old, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := AdoptCorpusProjection(root, digest(old), true)
	if err != nil {
		t.Fatal(err)
	}
	v := successorValidation(root, nil)
	v.builds = map[Reference]int{}
	// A partial source proof must still visit the other authority needed by the
	// complete corpus. Source proof reuse supplies no package/language PASS.
	if _, err := v.check(parent, 0); err != nil {
		t.Fatal(err)
	}
	first, err := checkedAppliedCorpusSuccessor(root, v)
	if err != nil {
		t.Fatal(err)
	}
	second, err := checkedAppliedCorpusSuccessor(root, v)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := Encode(first)
	b, _ := Encode(second)
	if !bytes.Equal(a, b) || first.Result.Identity != installed.Result.Identity || v.builds[parent] != 1 || v.builds[child] != 1 || v.readonlyOwner {
		t.Fatal("eager duplicate/partial proof boundary", v.builds)
	}
	t.Logf("SAME_OWNER parent_builds=%d child_builds=%d eager=%t", v.builds[parent], v.builds[child], !v.readonlyOwner)
	fresh := successorValidation(root, nil)
	fresh.builds = map[Reference]int{}
	if _, err := checkedAppliedCorpusSuccessor(root, fresh); err != nil || fresh.builds[parent] != 1 || fresh.builds[child] != 1 {
		t.Fatal("cross-invocation proof shared", err, fresh.builds)
	}
	if _, err := v.historicalGlobal(t.TempDir()); err == nil {
		t.Fatal("different root reused historical graph")
	}
	// A different expected source authority is a different projection key.
	if _, err := v.historicalGlobal(root, Reference{GlobalPath, digest([]byte("different authority"))}); err == nil {
		t.Fatal("wrong source scope/authority reused")
	}
	var successor StructuredSourceSuccessor
	if err := readReference(root, child, &successor); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{successor.Body.Path, i18n.GlossaryCorpusPath, CorpusSuccessorPath(installed)} {
		t.Run(filepath.Base(name), func(t *testing.T) {
			before, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, before)
			changed := bytes.Clone(before)
			changed[len(changed)/2] ^= 1
			writeFixture(t, root, name, changed)
			if err := os.Chtimes(filepath.Join(root, name), info.ModTime(), info.ModTime()); err != nil {
				t.Fatal(err)
			}
			if proof, err := checkedAppliedCorpusSuccessor(root, v); err == nil && proof != nil {
				t.Fatal("same owner accepted changed bytes")
			}
			if proof, err := checkedAppliedCorpusSuccessor(root); err == nil && proof != nil {
				t.Fatal("new owner reused old corpus PASS")
			}
			if err := CheckUnifiedCorpus(root); err == nil {
				t.Fatal("full consumer accepted changed authority")
			}
		})
	}
	// Failed nodes cannot enter the existing immutable graph. Mutation repair
	// requires a fresh observation scope rather than resurrecting a failed one.
	before, err := readRegular(root, child.Path)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, child.Path, append(bytes.Clone(before), 'x'))
	failed := successorValidation(root, nil)
	if _, err := failed.check(child, 0); err == nil {
		t.Fatal("invalid node passed")
	}
	if _, ok := failed.proofs[child]; ok {
		t.Fatal("failed proof cached")
	}
	writeFixture(t, root, child.Path, before)
	if _, err := checkedAppliedCorpusSuccessor(root); err != nil {
		t.Fatal("failed observation escaped invocation", err)
	}
	// Mutating a consumer result cannot mutate the stored historical graph.
	g, err := fresh.historicalGlobal(root)
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := Encode(g)
	g.Sources = nil
	again, err := fresh.historicalGlobal(root)
	if err != nil {
		t.Fatal(err)
	}
	actual, _ := Encode(again)
	if !bytes.Equal(expected, actual) {
		t.Fatal("consumer changed private graph")
	}
}
