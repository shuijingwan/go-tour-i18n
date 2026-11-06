package sitecontent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestClosureValidationVerifiedByteReuse(t *testing.T) {
	root := t.TempDir()
	name := "large-evidence.json"
	original := []byte(`{"body":"` + strings.Repeat("evidence", 1<<14) + `"}`)
	writeFixture(t, root, name, original)
	v := newClosureValidation(root)
	ref := Reference{name, digest(original)}
	proof := v.beginProof()
	if err := v.bind(ref); err != nil {
		t.Fatal(err)
	}
	v.endProof()
	v.proofs["node"] = proof
	for i := 0; i < 3; i++ {
		if err := v.reuse("node"); err != nil {
			t.Fatal(err)
		}
	}
	if err := v.verify(); err != nil {
		t.Fatal(err)
	}
	if v.counts["sha256:"+name] != 1 {
		t.Fatal("identical bytes rehashed", v.counts)
	}
	// Reader consumers cannot mutate the private verified observation.
	b, err := v.readRegular(root, name)
	if err != nil {
		t.Fatal(err)
	}
	b[0] = '!'
	if err := v.bind(ref); err != nil {
		t.Fatal("caller alias corrupted verified bytes", err)
	}
	if err := v.bind(Reference{name, strings.Repeat("0", 64)}); err == nil {
		t.Fatal("wrong expected SHA accepted")
	}
	// Equal size and timestamp are irrelevant: exact byte changes fail closed.
	changed := bytes.Clone(original)
	changed[len(changed)-3] = 'X'
	writeFixture(t, root, name, changed)
	if err := v.reuse("node"); err == nil {
		t.Fatal("stale evidence accepted")
	}
}

func TestWorkflowMemoClosureInputGuard(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	TestClosureValidationInputGuard(t)
}

func TestWorkflowMemoClosureGraph(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	t.Run("baseline-delta", TestClosureValidationGraphBaselineDelta)
	t.Run("initial-carry", TestClosureValidationGraphInitialCarry)
}

func TestClosureValidationVerifiedBytesBoundedFallback(t *testing.T) {
	root := t.TempDir()
	name := "evidence.json"
	original := []byte(`{"body":"evidence"}`)
	writeFixture(t, root, name, original)
	v := newClosureValidation(root)
	v.verifiedBytesSize = maxVerifiedBytes
	ref := Reference{name, digest(original)}
	for i := 0; i < 2; i++ {
		if err := v.bind(ref); err != nil {
			t.Fatal(err)
		}
	}
	if len(v.verifiedBytes) != 0 || v.counts["sha256:"+name] != 2 {
		t.Fatal("bounded fallback skipped SHA", v.counts)
	}
	if err := v.verify(); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, name, []byte(`{"body":"modified"}`))
	if err := v.bind(ref); err == nil {
		t.Fatal("SHA fallback accepted changed bytes")
	}
}

func TestClosureValidationVerifiedBytesLimit(t *testing.T) {
	if maxVerifiedBytes != 536870912 {
		t.Fatal("unexpected byte admission limit", maxVerifiedBytes)
	}
	root := t.TempDir()
	name := "input.json"
	original := []byte(`{"body":"evidence"}`)
	writeFixture(t, root, name, original)
	v := newClosureValidation(root)
	// The old 64 MiB boundary no longer forces SHA fallback. Reserve logical
	// capacity instead of allocating hundreds of MiB in this unit test.
	v.verifiedBytesSize = 67108864
	ref := Reference{name, digest(original)}
	for i := 0; i < 2; i++ {
		if err := v.bind(ref); err != nil {
			t.Fatal(err)
		}
	}
	if v.counts["sha256:"+name] != 1 || v.verifiedBytesSize != 67108864+len(original) {
		t.Fatal("old boundary still forces fallback", v.counts)
	}
	fresh := newClosureValidation(root)
	if len(fresh.verifiedBytes) != 0 || fresh.verifiedBytesSize != 0 {
		t.Fatal("verified bytes escaped invocation")
	}
	writeFixture(t, root, name, []byte(`{"body":"modified"}`))
	if err := v.bind(ref); err == nil {
		t.Fatal("larger byte admission masked changed bytes")
	}
}
