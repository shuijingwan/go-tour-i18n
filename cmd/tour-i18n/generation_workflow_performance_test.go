package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Opt-in command-level proof. All import/process writes occur in an independent
// TempDir copy, never the real completed batch. No Snapshot/QC step is executed.
func TestTranslationUnitGenerationWorkflowPerformance(t *testing.T) {
	out, locale, batch := os.Getenv("GO_TOUR_WORKFLOW_PROFILE_DIR"), os.Getenv("GO_TOUR_WORKFLOW_LOCALE"), os.Getenv("GO_TOUR_WORKFLOW_BATCH")
	if out == "" || locale == "" || batch == "" {
		t.Skip("explicit isolated workflow proof destination/locale/batch required")
	}
	if !strings.HasPrefix(filepath.Clean(out), "/tmp/") {
		t.Fatal("proof evidence must be outside the repository in /tmp")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	copyStart := time.Now()
	files, size := 0, int64(0)
	copyFile := func(name string) {
		t.Helper()
		st, err := os.Lstat(filepath.Join(root, name))
		if err != nil || !st.Mode().IsRegular() {
			t.Fatalf("fixture input must be regular: %s: %v", name, err)
		}
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(temp, name)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, b, st.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
		files++
		size += int64(len(b))
	}
	// Known source/authority roots only; no .git, environment secrets or cache.
	for _, dir := range []string{"data", "locales", "_content", "docs", "internal", "cmd"} {
		if err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				copyFile(rel)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"go.mod", "go.sum", "AGENTS.md", "README.md"} {
		copyFile(name)
	}
	batchPath := filepath.Join("data", "retranslation-runs", locale, batch)
	b, err := os.ReadFile(filepath.Join(temp, batchPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest i18n.RetranslationBatchManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(filepath.Join(temp, batchPath, "generation-imports/initial-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var provenance i18n.GenerationResultBundleManifest
	if err := json.Unmarshal(b, &provenance); err != nil {
		t.Fatal(err)
	}
	bundlePath := filepath.Join(temp, "data/generation-archives", locale, provenance.GenerationBundleSHA256+".zip")
	staging := t.TempDir()
	expected := map[string][]byte{}
	for _, dir := range []string{"candidates", "validation"} {
		entries, err := os.ReadDir(filepath.Join(temp, batchPath, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			name := filepath.Join(dir, entry.Name())
			b, err := os.ReadFile(filepath.Join(temp, batchPath, name))
			if err != nil {
				t.Fatal(err)
			}
			expected[name] = b
		}
	}
	for _, u := range manifest.Units {
		name := filepath.Base(u.InputPath)
		b, err := os.ReadFile(filepath.Join(temp, batchPath, "raw-responses", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(staging, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Only the disposable copy starts at the pre-import state.
	for _, dir := range []string{"raw-responses", "generation-imports", "candidates", "validation"} {
		if err := os.RemoveAll(filepath.Join(temp, batchPath, dir)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(temp, batchPath, "result.json")); err != nil {
		t.Fatal(err)
	}
	oldWD, _ := os.Getwd()
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	metrics := map[string]any{"copy_seconds": time.Since(copyStart).Seconds(), "copy_file_count": files, "copy_bytes": size, "locale": locale, "batch": batch, "unit_count": manifest.UnitCount}
	commands := []struct {
		name string
		args []string
	}{
		{"current-check", []string{"generation-bundle", "current-check", "--bundle", bundlePath}},
		{"import", []string{"generation-bundle", "import", "--bundle", bundlePath, "--input-dir", staging, "--provider", provenance.Provider, "--model", provenance.Model}},
		{"process", []string{"retranslation", "process", "--locale", locale, "--batch-id", batch}},
	}
	total := 0.0
	for _, command := range commands {
		profile, err := os.Create(filepath.Join(out, command.name+".cpu.pprof"))
		if err != nil {
			t.Fatal(err)
		}
		if err := pprof.StartCPUProfile(profile); err != nil {
			t.Fatal(err)
		}
		stdout, err := os.Create(filepath.Join(out, command.name+".stdout.log"))
		if err != nil {
			t.Fatal(err)
		}
		originalStdout := os.Stdout
		os.Stdout = stdout
		start, cpu := time.Now(), bundleProofCPU()
		err = run(command.args)
		wall, usedCPU := time.Since(start).Seconds(), bundleProofCPU()-cpu
		os.Stdout = originalStdout
		stdout.Close()
		pprof.StopCPUProfile()
		profile.Close()
		metrics[command.name] = map[string]any{"wall_seconds": wall, "cpu_seconds": usedCPU}
		total += wall
		metrics["total_seconds"] = total
		b, _ := json.MarshalIndent(metrics, "", "  ")
		if e := os.WriteFile(filepath.Join(out, "metrics.json"), append(b, '\n'), 0600); e != nil {
			t.Fatal(e)
		}
		if err != nil {
			t.Fatal(command.name, err)
		}
		t.Logf("%s wall=%.6fs cpu=%.6fs", command.name, wall, usedCPU)
	}
	for name, want := range expected {
		got, err := os.ReadFile(filepath.Join(temp, batchPath, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("candidate/validation differs from immutable successful fixture", name, err)
		}
	}
	b, _ = os.ReadFile(filepath.Join(temp, batchPath, "generation-imports/initial-manifest.json"))
	var installed i18n.GenerationResultBundleManifest
	if err := json.Unmarshal(b, &installed); err != nil || !reflectGenerationProvenanceEqual(provenance, installed) {
		t.Fatal("import provenance/output exact-set changed", err)
	}
	metrics["candidate_bytes_identical"], metrics["validation_bytes_identical"], metrics["import_provenance_identical"] = true, true, true
	metrics["bundle_sha256"], metrics["bundle_identity"] = provenance.GenerationBundleSHA256, provenance.InputIdentitySHA256
	metrics["formal_repository_mutation"] = "none"
	b, _ = json.MarshalIndent(metrics, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "metrics.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

func reflectGenerationProvenanceEqual(a, b i18n.GenerationResultBundleManifest) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y) && a.GenerationBundleSHA256 != "" && contentidentity.SHA256Hex(x) == contentidentity.SHA256Hex(y)
}
