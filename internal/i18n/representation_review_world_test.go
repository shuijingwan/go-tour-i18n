package i18n

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRepresentationReviewPreparationKeepsExplicitWorld(t *testing.T) {
	root, c, _, plan, _, _ := authorizationMachineFixture(t)
	o := plan.Receipt.Options
	parent, err := prepareMachineReconciliationInWorld(root, c, o, plan.Receipt.GlossarySHA256)
	if err != nil {
		t.Fatal(err)
	}
	p := parent.compatibility
	p.tuOnly, p.disableSuccessor, p.terminalSHA = true, true, parent.currentGlossarySHA
	p.historicalEvidenceRefs = map[string]string{}
	checkpointGlossary(t, root, "\n# independent later current world\n", "review-world-next")
	child, err := prepareRepresentationReviewMachine(root, c, o, parent)
	if err != nil || child.compatibility != p || !child.historical {
		t.Fatal("semantic reconstruction created another/default resolver", err)
	}
	if !p.tuOnly || !p.disableSuccessor || p.terminalSHA != parent.currentGlossarySHA || p.historicalSHA != parent.currentGlossarySHA || len(p.historicalEvidenceRefs) != 0 {
		t.Fatal("explicit world bindings changed")
	}
	if !reflect.DeepEqual(parent.receipt, child.receipt) || !reflect.DeepEqual(parent.resolution.Units, child.resolution.Units) {
		t.Fatal("machine preparation result changed")
	}
	// A separate historical invocation owns a separate mutable resolver.
	other, err := prepareMachineReconciliationInWorld(root, c, o, plan.Receipt.GlossarySHA256)
	if err != nil || other.compatibility == p {
		t.Fatal("historical invocation leaked world", err)
	}
	ordinary, err := prepareRepresentationReviewMachine(root, c, o, nil)
	if err != nil || ordinary.historical || ordinary.compatibility == p || ordinary.compatibility.historicalSHA != "" || ordinary.currentGlossarySHA == parent.currentGlossarySHA {
		t.Fatal("default current preparation inherited historical world", err)
	}
	wrong := o
	wrong.Package = "another-package"
	if _, err := prepareRepresentationReviewMachine(root, c, wrong, parent); err == nil {
		t.Fatal("foreign review options accepted")
	}
	if _, err := prepareRepresentationReviewMachine(root, c, o, ordinary); err == nil {
		t.Fatal("current context impersonated historical world")
	}
	archive, _ := glossaryArchivePath(o.Locale, parent.currentGlossarySHA)
	b, err := readCompatibilityFile(root, archive)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, filepath.ToSlash(archive), append(bytes.Clone(b), ' '))
	if _, err := prepareRepresentationReviewMachine(root, c, o, parent); err == nil {
		t.Fatal("warm world skipped archived glossary byte verification")
	}
}
