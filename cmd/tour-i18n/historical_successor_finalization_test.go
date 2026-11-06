package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestHistoricalSuccessorFinalizeCommandContract(t *testing.T) {
	args := []string{"quality-check", "finalize", "--content-package", "site-v2-shell", "--locale", "zh-CN", "--snapshot-id", "qc004", "--previous-snapshot-id", "qc002"}
	pkg, remaining, err := parseTranslationContentPackageArgs(args)
	if err != nil || pkg != "site-v2-shell" || strings.Contains(strings.Join(remaining, " "), "--content-package") || !strings.Contains(strings.Join(remaining, " "), "--previous-snapshot-id qc002") {
		t.Fatal(pkg, remaining, err)
	}
	b, err := os.ReadFile("../../docs/TRANSLATION_QUALITY_REVIEW.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"INPUT_KIND", "INPUT_PATH_CONTRACT", "MUTATION_EXPECTATION", "IDEMPOTENCE", "CURRENTNESS_CHECK", "PRIOR_OR_CAS", "SUCCESS_STOP", "FAILURE_STOP", "frozen predecessor event worlds", "write-once/no-overwrite"} {
		if !strings.Contains(string(b), s) {
			t.Fatal("finalize contract missing", s)
		}
	}
}

// Explicit proof executes the actual CLI only in a private complete authority
// copy. It never changes the repository's Snapshot, QC or language artifacts.
func TestHistoricalSuccessorFinalizeRepositoryCopy(t *testing.T) {
	out := os.Getenv("GO_TOUR_FINALIZATION_LINEAGE_PROOF")
	if out == "" {
		t.Skip("explicit copied-repository proof only")
	}
	if !strings.HasPrefix(filepath.Clean(out), "/tmp/") {
		t.Fatal("external evidence destination required")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	destination := "data/quality-check-snapshots/zh-CN/site-v2-shell-qc-004/finalization.json"
	if _, err := os.Lstat(filepath.Join(root, destination)); !os.IsNotExist(err) {
		t.Fatal("real destination must remain absent", err)
	}
	temp := t.TempDir()
	for _, dir := range []string{"data", "locales", "_content", "docs", "internal", "cmd"} {
		err = filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			st, err := os.Lstat(path)
			if err != nil {
				return err
			}
			if !st.Mode().IsRegular() {
				t.Fatalf("nonregular fixture input: %s", path)
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			dst := filepath.Join(temp, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(dst, raw, st.Mode().Perm()); err != nil {
				return err
			}
			return os.Chmod(dst, st.Mode().Perm()) // WriteFile creation is subject to umask.
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"go.mod", "go.sum", "AGENTS.md", "README.md"} {
		st, err := os.Stat(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(temp, name), b, 0644); err != nil {
			t.Fatal(err)
		}
		if err = os.Chmod(filepath.Join(temp, name), st.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	if err = os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	log, err := os.Create(filepath.Join(out, "copied-cli.stdout.log"))
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = log
	start := time.Now()
	err = run([]string{"quality-check", "finalize", "--content-package", "site-v2-shell", "--locale", "zh-CN", "--snapshot-id", "site-v2-shell-qc-004", "--previous-snapshot-id", "site-v2-shell-qc-002"})
	elapsed := time.Since(start).Seconds()
	os.Stdout = stdout
	log.Close()
	result := map[string]any{"seconds": elapsed, "real_recorded": false, "real_mutation": "none", "error": ""}
	if err != nil {
		result["error"] = err.Error()
	} else {
		b, readErr := os.ReadFile(filepath.Join(temp, destination))
		if readErr != nil {
			t.Fatal(readErr)
		}
		var f i18n.QualityCheckFinalization
		if json.Unmarshal(b, &f) != nil || f.SnapshotID != "site-v2-shell-qc-004" || f.PreviousSnapshotID != "site-v2-shell-qc-002" || len(f.Units) != 1 || f.Units[0].UnitID != "f4365d05fcff295676c0964f81058e6485bb0361f8440cbf243220a6acc1e97d" {
			t.Fatal("wrong copied finalization", f)
		}
		if writeErr := os.WriteFile(filepath.Join(out, "copied-finalization.json"), b, 0600); writeErr != nil {
			t.Fatal(writeErr)
		}
		result["would_pass"] = true
		result["current_snapshot"] = f.SnapshotID
		result["predecessor"] = f.PreviousSnapshotID
		result["unit"] = f.Units[0].UnitID
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	if writeErr := os.WriteFile(filepath.Join(out, "real-proof.json"), b, 0600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if _, statErr := os.Lstat(filepath.Join(root, destination)); !os.IsNotExist(statErr) {
		t.Fatal("real finalization created", statErr)
	}
	if err != nil {
		t.Fatal("copied exact CLI finalization", err)
	}
	t.Logf("copied real lineage PASS, real mutation=none; seconds=%.6f", elapsed)
}
