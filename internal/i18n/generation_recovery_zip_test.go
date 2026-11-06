package i18n

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerationRecoveryZIPConflictAndNoOverwrite(t *testing.T) {
	root, c, batch, bundle, m := generationRecoveryFixture(t, 2)
	name := filepath.Base(m.ExpectedOutputs[0].BundlePath)
	input := generationRecoveryInput(t, root, batch, m.ExpectedOutputs[0])
	archive := filepath.Join(t.TempDir(), "partial.zip")
	writeZIP := func(outputs ...[]byte) {
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		for _, data := range outputs {
			w, err := z.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(data); err != nil {
				t.Fatal(err)
			}
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(archive, b.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(t.TempDir(), "recovery")
	other := []byte(appendBeforeRetranslationArtifactEOF(string(input), "独立说明。"))
	writeZIP(input, other)
	if _, err := RecoverGenerationOutputs(root, c, bundle, []string{archive}, target); err == nil || !strings.Contains(err.Error(), "conflicting valid") {
		t.Fatal("ZIP conflict not rejected", err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatal("conflict wrote staging")
	}
	writeZIP(input, input)
	r, err := RecoverGenerationOutputs(root, c, bundle, []string{archive}, target)
	if err != nil || len(r.Recovered) != 1 || len(r.Missing) != 1 {
		t.Fatal(r, err)
	}
	if _, err := RecoverGenerationOutputs(root, c, bundle, []string{archive}, target); err == nil {
		t.Fatal("recovery overwritten")
	}
	if _, err := os.Lstat(filepath.Join(root, "data/retranslation-runs/zh-CN", batch, "generation-imports")); !os.IsNotExist(err) {
		t.Fatal("recovery created provenance")
	}
}
