package sitecontent

import (
	"bytes"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestStructuredSuccessorInvocationReuse(t *testing.T) {
	root, parent, consumption, _, _ := consumedReplacementFixture(t)
	proposal, input := replacementProposalFixture(t, root, parent, consumption)
	plan, err := ApplyConsumedStructuredSourceReplacement(root, proposal, input.PredecessorIdentity, input.PriorPointer.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	g, err := loadSourceCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	old, err := readProjectionCorpus(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	baseline := successorValidation(root, nil)
	baseline.disabled, baseline.builds = true, map[Reference]int{}
	want, err := projectAdoptedCorpusAtSources(root, g, old, false, baseline)
	if err != nil {
		t.Fatal("uncached projection", err)
	}
	v := successorValidation(root, nil)
	v.builds = map[Reference]int{}
	got, err := projectAdoptedCorpusAtSources(root, g, old, false, v)
	if err != nil {
		t.Fatal("shared invocation projection", err)
	}
	wantBytes, _ := Encode(want)
	gotBytes, _ := Encode(got)
	if !bytes.Equal(wantBytes, gotBytes) || want.Identity != got.Identity {
		t.Fatal("proof reuse changed corpus bytes/identity")
	}
	if baseline.builds[parent] <= 1 || v.builds[parent] != 1 || v.builds[plan.Receipt] != 1 {
		t.Fatalf("missing real repetition/reuse proof: baseline=%v reused=%v", baseline.builds, v.builds)
	}
	t.Logf("FULL_REBUILDS parent_before=%d parent_after=%d replacement_after=%d", baseline.builds[parent], v.builds[parent], v.builds[plan.Receipt])
	for ref, count := range v.builds {
		if count != 1 {
			t.Fatal("nested exact proof rebuilt", ref, count)
		}
	}
	if _, err := v.check(plan.Receipt, 0); err != nil || v.builds[plan.Receipt] != 1 {
		t.Fatal("same invocation root reuse", err)
	}
	p, err := v.check(parent, 0)
	if err != nil || p.Identity == plan.Authority.Identity {
		t.Fatal("unrelated exact authority aliased", err)
	}
	bad := plan.Receipt
	bad.SHA256 = digest([]byte("wrong SHA"))
	if _, err := v.check(bad, 0); err == nil {
		t.Fatal("wrong SHA reused")
	}
	if _, err := v.check(plan.Receipt, 32); err == nil {
		t.Fatal("cached subtree bypassed depth limit")
	}
	if _, err := v.check(parent, 33); err == nil {
		t.Fatal("depth limit bypassed")
	}
	v.active[parent] = true
	if _, err := v.check(parent, 0); err == nil {
		t.Fatal("active ancestry cycle reused")
	}
	delete(v.active, parent)
	for _, name := range []string{input.Body.Path, consumption.Path, input.PriorPointer.Path, GlobalPath} {
		t.Run("dependency:"+name, func(t *testing.T) {
			b, err := readRegular(root, name)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, name, b)
			writeFixture(t, root, name, append(bytes.Clone(b), ' '))
			if _, err := v.check(plan.Receipt, 0); err == nil {
				t.Fatal("changed dependency reused", name)
			}
		})
	}
}
