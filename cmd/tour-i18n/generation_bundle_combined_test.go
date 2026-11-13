package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestGenerationBundleCombinedExportContract(t *testing.T) {
	doc := string(courseContextRead(t, "../..", "docs/GENERATION_BUNDLE_EXPORT_CURRENT.md"))
	for _, rule := range []string{"--current-check", "repository mutation=none", "standalone current-check", "verification 失败不发布 ZIP", "PRIOR / CAS / IDEMPOTENCE", "Formal Command Contract v3"} {
		if !strings.Contains(doc, rule) {
			t.Fatal("missing canonical option contract", rule)
		}
	}
	for _, registered := range []bool{false, true} {
		name := "native"
		if registered {
			name = "registered"
		}
		t.Run(name, func(t *testing.T) {
			f := makeNativeGenerationFixture(t, !registered)
			before := authorizationDiagnosisInventory(t, f.f.root)
			original := courseContextRead(t, filepath.Dir(f.output), filepath.Base(f.output))
			output := filepath.Join(t.TempDir(), "combined.zip")
			args := append(generationExportArgs(f, output), "--current-check")
			out, err := invokeGenerationCLI(t, f.f.root, f.f.catalog, args)
			if err != nil || !strings.Contains(string(out), "Generation bundle exported:") || !strings.Contains(string(out), "Generation bundle: CURRENT") || !strings.Contains(string(out), "identity="+f.manifest.InputIdentitySHA256) {
				t.Fatal("combined output contract", err, string(out))
			}
			if !bytes.Equal(original, courseContextRead(t, filepath.Dir(output), filepath.Base(output))) || !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
				t.Fatal("combined export changed canonical bytes or repository")
			}
			if out, err := invokeGenerationCLI(t, f.f.root, f.f.catalog, args); err == nil || strings.Contains(string(out), "CURRENT") {
				t.Fatal("existing output not refused", err, string(out))
			}
			for _, extra := range [][]string{{"--current-check=invalid"}, {"--current-check", "true"}} {
				rejected := filepath.Join(t.TempDir(), "rejected.zip")
				if _, err := invokeGenerationCLI(t, f.f.root, f.f.catalog, append(generationExportArgs(f, rejected), extra...)); err == nil {
					t.Fatal("invalid flag contract accepted", extra)
				}
				if _, err := os.Lstat(rejected); !os.IsNotExist(err) {
					t.Fatal("parser failure published ZIP", err)
				}
			}
			plain := filepath.Join(t.TempDir(), "plain.zip")
			out, err = invokeGenerationCLI(t, f.f.root, f.f.catalog, append(generationExportArgs(f, plain), "--current-check=false"))
			if err != nil || strings.Contains(string(out), "CURRENT") || !bytes.Equal(original, courseContextRead(t, filepath.Dir(plain), filepath.Base(plain))) {
				t.Fatal("default export semantics changed", err, string(out))
			}
			// A stale dependency still stops before output publication.
			receipt := filepath.Join(f.f.root, f.f.receipt)
			if filepath.IsAbs(f.f.receipt) {
				receipt = f.f.receipt
			}
			old, err := os.ReadFile(receipt)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(receipt, append(old, ' '), 0644); err != nil {
				t.Fatal(err)
			}
			rejected := filepath.Join(t.TempDir(), "stale.zip")
			if _, err := invokeGenerationCLI(t, f.f.root, f.f.catalog, append(generationExportArgs(f, rejected), "--current-check")); err == nil {
				t.Fatal("stale dependency accepted")
			}
			if _, err := os.Lstat(rejected); !os.IsNotExist(err) {
				t.Fatal("stale export published ZIP", err)
			}
		})
	}
}

func TestGenerationBundleCombinedRetry(t *testing.T) {
	f := makeNativeGenerationFixture(t, true)
	var formal i18n.RetranslationBatchManifest
	if err := json.Unmarshal(courseContextRead(t, f.f.root, f.batch.BatchPath+"/manifest.json"), &formal); err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(formal.Units[0].InputPath)
	courseContextWrite(t, f.f.root, f.batch.BatchPath+"/raw-responses/"+name, []byte("broken response\n"))
	r, err := i18n.ProcessRetranslationBatch(f.f.root, f.f.catalog, i18n.RetranslationProcessOptions{Locale: f.f.locale, BatchID: f.batch.BatchID})
	if err != nil || r.RestoreFailed+r.ValidationFailed != 1 {
		t.Fatal("synthetic retry fixture", r, err)
	}
	before := authorizationDiagnosisInventory(t, f.f.root)
	baseline := filepath.Join(t.TempDir(), "retry.zip")
	args := append(generationExportArgs(f, baseline), "--unit-id", formal.Units[0].UnitID)
	if _, err := invokeGenerationCLI(t, f.f.root, f.f.catalog, args); err != nil {
		t.Fatal(err)
	}
	combined := filepath.Join(t.TempDir(), "combined-retry.zip")
	args = append(generationExportArgs(f, combined), "--unit-id", formal.Units[0].UnitID, "--current-check")
	out, err := invokeGenerationCLI(t, f.f.root, f.f.catalog, args)
	if err != nil || !strings.Contains(string(out), "task=translation-unit-retry") || !strings.Contains(string(out), "attempt=2") || !strings.Contains(string(out), "Generation bundle: CURRENT") {
		t.Fatal(err, string(out))
	}
	if !bytes.Equal(courseContextRead(t, filepath.Dir(baseline), filepath.Base(baseline)), courseContextRead(t, filepath.Dir(combined), filepath.Base(combined))) || !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
		t.Fatal("retry bytes/scope/state changed")
	}
}

func TestGenerationBundleCombinedExportRun(t *testing.T) {
	f := makeNativeGenerationRunFixture(t)
	t.Chdir(f.f.root)
	out, err := authorizationDiagnosisOutput(t, func() error { return run(append(generationExportArgs(f, f.output), "--current-check")) })
	if err != nil || !strings.Contains(string(out), "Generation bundle: CURRENT") {
		t.Fatal(err, string(out))
	}
	assertRunGenerationBundle(t, f, true)
}
