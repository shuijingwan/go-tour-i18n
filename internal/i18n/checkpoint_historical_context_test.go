package i18n

import (
	"fmt"
	"testing"
)

func TestCheckpointFlatHistoricalContext(t *testing.T) {
	root, c, _, snapshot := checkpointFixture(t)
	p, err := VerifyHistoricalQualityCheckFinalization(root, "zh-CN", snapshot, c)
	if err != nil {
		t.Fatal(err)
	}
	path := "data/synthetic-representation/current.json"
	frozen := []byte("sealed pointer bytes\n")
	ref := GlossaryArchiveReference{Path: path, SHA256: sum(frozen)}
	writeSnapshotSource(t, root, path, frozen)
	p.References = append(p.References, ref)
	if err := checkFlatHistoricalIntegrity(root, c, p); err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, path, []byte("current successor pointer bytes\n"))
	if err := checkFlatHistoricalIntegrity(root, c, p); err == nil {
		t.Fatal("changed pointer accepted without historical proof")
	}
	c.HistoricalContextReader = func(_ string, locale string, want GlossaryArchiveReference) ([]byte, error) {
		if locale != "zh-CN" || want != ref {
			return nil, fmt.Errorf("not the sealed representation context")
		}
		return frozen, nil // Exact lifecycle proof is tested in sitecontent.
	}
	if err := checkFlatHistoricalIntegrity(root, c, p); err != nil {
		t.Fatal("flat checkpoint did not consume the historical context reader", err)
	}
	if got, err := ReadHistoricalTranslationReference(root, c, p, ref); err != nil || string(got) != string(frozen) {
		t.Fatal("verified historical reference lost its frozen bytes", err)
	}
	outside := ref
	outside.SHA256 = sum([]byte("current successor pointer bytes\n"))
	if _, err := ReadHistoricalTranslationReference(root, c, p, outside); err == nil {
		t.Fatal("current bytes accepted as an unsealed historical reference")
	}
	// This exception cannot hide corrupt original independent Reviewer proof.
	qc := p.Finalization.QCResults[0].Path
	original, err := readCompatibilityFile(root, qc)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotSource(t, root, qc, []byte("corrupt independent Reviewer evidence"))
	if err := checkFlatHistoricalIntegrity(root, c, p); err == nil {
		t.Fatal("historical context reader hid corrupt Reviewer evidence")
	}
	writeSnapshotSource(t, root, qc, original)
	c.HistoricalContextReader = func(string, string, GlossaryArchiveReference) ([]byte, error) {
		return []byte("current successor pointer bytes\n"), nil
	}
	if err := checkFlatHistoricalIntegrity(root, c, p); err == nil {
		t.Fatal("current bytes substituted for sealed pointer SHA")
	}
}
