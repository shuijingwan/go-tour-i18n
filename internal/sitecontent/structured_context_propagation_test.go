package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// All mutations below belong to an isolated synthetic source/language fixture.
func plannerPropagationFixture(t *testing.T) (string, Reference, Reference) {
	t.Helper()
	root, _, _, parent, replacement := historicalGlobalSuccessorFixture(t, "_content/doc/index.html", "_content/doc/install.html")
	proposal := PagePackageProposal{Schema: PagePackageAdoptionSchema, Package: "future-entry-pages", Surface: "entry-pages"}
	for _, name := range []string{"_content/doc/index.html", "_content/doc/install.html"} {
		ref, err := ReferenceFile(root, name)
		if err != nil {
			t.Fatal(err)
		}
		proposal.Sources = append(proposal.Sources, ref)
	}
	raw, _ := Encode(proposal)
	writeFixture(t, root, "data/planner-page-proposal.json", raw)
	ref, _ := ReferenceFile(root, "data/planner-page-proposal.json")
	a, err := BuildPagePackageAdoption(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptPagePackage(root, ref, a.BaseIdentity); err != nil {
		t.Fatal(err)
	}
	old, _ := readRegular(root, i18n.GlossaryCorpusPath)
	if _, err := AdoptCorpusProjection(root, digest(old), true); err != nil {
		t.Fatal(err)
	}
	bundle, _, err := i18n.ExportUnifiedGlossaryBundle(root, "zh-CN", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := i18n.RecordUnifiedGlossaryReview(root, bundle, "synthetic-planner-propagation", "synthetic-independent-reviewer", "synthetic-generation", "passed", nil); err != nil {
		t.Fatal(err)
	}
	return root, parent, replacement
}

func TestStructuredPlannerContextPropagation(t *testing.T) {
	root, parent, replacement := plannerPropagationFixture(t)
	// The default/eager path is the pre-propagation semantic baseline. It
	// intentionally supplies no shared source graph to its downstream readers.
	baseline := newClosureValidation(root)
	if err := baseline.currentInputs("zh-CN"); err != nil {
		t.Fatal(err)
	}
	want, err := baseline.planStructuredAssets(root, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, _ := Encode(want)
	wantGlobal, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	wantGlobalBytes, _ := Encode(wantGlobal)
	wantCorpus, err := BuildUnifiedCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	wantCorpusBytes, _ := Encode(wantCorpus)
	var previous *structuredSuccessorValidation
	for invocation := 0; invocation < 2; invocation++ {
		validation := newClosureValidation(root)
		if err := validation.currentInputs("zh-CN"); err != nil {
			t.Fatal(err)
		}
		var owned *structuredSuccessorValidation
		err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
			owned = v
			if v == previous || !v.readonlyOwner || successorValidation(root, []*structuredSuccessorValidation{v}) != v {
				return fmt.Errorf("invocation ownership/isolation changed")
			}
			v.historicalMetrics, v.builds = map[string]int{}, map[Reference]int{}
			validation.readonlySuccessors = v
			got, err := validation.planStructuredAssets(root, "zh-CN")
			if err != nil {
				return err
			}
			gotBytes, _ := Encode(got)
			if !bytes.Equal(wantBytes, gotBytes) || want.Identity != got.Identity {
				return fmt.Errorf("StructuredPlan bytes/identity changed")
			}
			g, err := LoadCurrent(root, v)
			if err != nil {
				return err
			}
			globalBytes, _ := Encode(g)
			if !bytes.Equal(wantGlobalBytes, globalBytes) {
				return fmt.Errorf("Global bytes/identity changed")
			}
			corpus, err := BuildUnifiedCorpus(root, v)
			if err != nil {
				return err
			}
			corpusBytes, _ := Encode(corpus)
			if !bytes.Equal(wantCorpusBytes, corpusBytes) {
				return fmt.Errorf("corpus bytes/identity changed")
			}
			// Exercise the same discovery subtree used by currentTourCheckpoint,
			// without constructing synthetic Tour language judgments.
			catalog, err := checkedCatalog(root)
			if err != nil {
				return err
			}
			if err := AttachRegisteredTranslationPackages(root, catalog, v); err != nil {
				return err
			}
			if _, err := planLocaleLanguage(root, "zh-CN", true, validation); err != nil {
				return err
			}
			if v.historicalMetrics["aggregate-verifies"] != 0 {
				return fmt.Errorf("nested readonly aggregate verification")
			}
			return validation.verify()
		})
		if err != nil {
			t.Fatal(err)
		}
		if owned.readonlyOwner || owned.historicalMetrics["aggregate-verifies"] != 1 || owned.historicalMetrics["BuildGlobal"] != 1 || owned.historicalMetrics["snapshotFS"] != 1 || owned.builds[parent] != 1 || owned.builds[replacement] != 1 {
			t.Fatalf("fragmented source graph: metrics=%v builds=%v", owned.historicalMetrics, owned.builds)
		}
		previous = owned
		t.Logf("PLANNER_GRAPH invocation=%d owner_contexts=1 Global=%d snapshotFS=%d aggregate=%d predecessor=%d", invocation, owned.historicalMetrics["BuildGlobal"], owned.historicalMetrics["snapshotFS"], owned.historicalMetrics["aggregate-verifies"], owned.builds[parent])
	}
	// Default/publication checks neither inherit the escaped planner graph nor
	// retain deferred verification. Mutating a returned receipt cannot poison it.
	fresh := successorValidation(root, nil)
	fresh.historicalMetrics = map[string]int{}
	if fresh == previous || fresh.readonlyOwner {
		t.Fatal("default check inherited readonly graph")
	}
	a, err := fresh.check(parent, 0)
	if err != nil || fresh.historicalMetrics["aggregate-verifies"] == 0 {
		t.Fatal("default check is no longer eager", err)
	}
	a.Current.Path = "caller-mutated"
	b, err := CheckStructuredSourceSuccessor(root, parent)
	if err != nil || b.Current.Path == a.Current.Path {
		t.Fatal("default/result isolation failed", err)
	}
	// Mutation after a shared nested proof must still fail at the outer boundary.
	var authority StructuredSourceSuccessor
	if err := readReference(root, replacement, &authority); err != nil {
		t.Fatal(err)
	}
	body, err := readRegular(root, authority.Body.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer writeFixture(t, root, authority.Body.Path, body)
	err = withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		if _, err := LoadCurrent(root, v); err != nil {
			return err
		}
		info, err := os.Stat(filepath.Join(root, authority.Body.Path))
		if err != nil {
			return err
		}
		changed := bytes.Clone(body)
		changed[len(changed)/2] ^= 1
		writeFixture(t, root, authority.Body.Path, changed)
		return os.Chtimes(filepath.Join(root, authority.Body.Path), info.ModTime(), info.ModTime())
	})
	if err == nil {
		t.Fatal("mid-invocation source mutation escaped aggregate verification")
	}
}
