package contentidentity

import (
	"bytes"
	"fmt"
	"testing"
)

func TestTransformBudgetIndependentOfRawCapacity(t *testing.T) {
	done := BeginMemoization()
	defer done()
	m := activeMemo()
	m.store = nil      // this test exhausts L1 only; persistent fallback is tested separately
	m.bytes = 64 << 20 // simulate a full source-observation raw pool
	key := bytes.Repeat([]byte("exact descriptor;"), 16)
	output := []byte("private deterministic representation")
	calls := 0
	compute := func() ([]byte, error) { calls++; return bytes.Clone(output), nil }
	first, err := Transform("capacity-test/v1", key, compute)
	if err != nil {
		t.Fatal(err)
	}
	first[0] ^= 1
	second, err := Transform("capacity-test/v1", bytes.Clone(key), compute)
	if err != nil || calls != 1 || !bytes.Equal(second, output) {
		t.Fatal("raw capacity starved pure transform or output aliased", err, calls)
	}
	if m.bytes != 64<<20 || m.transformBytes < len(key)+len(output) || m.transformBytes > 16<<20 {
		t.Fatal("existing memory budgets changed")
	}
	key[len(key)/2+20] ^= 1
	if _, err := Transform("capacity-test/v1", key, compute); err != nil || calls != 2 {
		t.Fatal("different full input reused an observation", err, calls)
	}
	for i := 0; i < 2; i++ {
		_, err := Transform("capacity-error/v1", key, func() ([]byte, error) { calls++; return nil, fmt.Errorf("failed") })
		if err == nil {
			t.Fatal("failed computation accepted")
		}
	}
	if calls != 4 {
		t.Fatal("failed computation retained")
	}
	m.transformBytes = 16 << 20
	for i := 0; i < 2; i++ {
		if _, err := Transform("capacity-full/v1", key, compute); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 6 || m.transformBytes != 16<<20 {
		t.Fatal("full transform budget was exceeded")
	}
}
