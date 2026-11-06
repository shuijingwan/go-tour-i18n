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
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

// An opt-in sparse TempDir copy of the exact real invocation. No real process,
// import, retry or revalidation is executed. Never install its output in root.
func TestCurrentValidationRealProcessTempProof(t *testing.T) {
	out := os.Getenv("GO_TOUR_REAL_PROCESS_PROOF")
	if out == "" {
		t.Skip("explicit isolated proof destination required")
	}
	root, temp, c, m, authority, observed := copiedCommentRevisionRealFixture(t, false)
	const batch = "data/retranslation-runs/zh-CN/chatgpt-zh-CN-051"
	const failed = "a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f"
	var err error
	if err = os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(out, "real-after.cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err = pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	done := contentidentity.BeginMemoization()
	start := time.Now()
	result, err := i18n.ProcessRetranslationBatch(temp, c, i18n.RetranslationProcessOptions{Locale: "zh-CN", BatchID: m.BatchID})
	wall := time.Since(start).Seconds()
	done()
	pprof.StopCPUProfile()
	f.Close()
	if err != nil {
		t.Fatal("TempDir process", err)
	}
	if result.RestorePassed != 9 || result.ValidationPassed != 9 {
		t.Fatal(result)
	}
	preserved := 0
	stableSlots := []string{}
	rows := []map[string]any{}
	for _, r := range result.Units {
		revised, e := os.ReadFile(filepath.Join(temp, batch, r.CandidatePath))
		if e != nil {
			t.Fatal(e)
		}
		if r.UnitID != failed {
			old, e := os.ReadFile(filepath.Join(root, batch, r.CandidatePath))
			if e != nil || !bytes.Equal(old, revised) {
				t.Fatal("passed formal candidate differs", r.UnitID, e)
			}
			preserved++
		}
		u, err := c.Unit(r.UnitID)
		if err != nil {
			t.Fatal(err)
		}
		comparisons, err := i18n.ValidateTranslationUnitCommentCoverage(temp, c, u, "zh-CN", revised)
		if err != nil {
			t.Fatal("active comment gate", err)
		}
		byID := map[string]i18n.CommentSlotComparison{}
		for _, row := range comparisons {
			byID[row.SlotID] = row
		}
		for _, finding := range authority.CurrentValidation.Findings {
			if finding.UnitID == r.UnitID {
				for _, old := range finding.CommentSlots {
					row, ok := byID[old.SlotID]
					if !ok || row.ExactSourceCarry || row.ReviewRequired {
						t.Fatal("authorized translated slot lost identity", old.SlotID)
					}
					stableSlots = append(stableSlots, row.SlotID)
				}
			}
		}
		rows = append(rows, map[string]any{"unit_id": r.UnitID, "status": r.Status, "candidate_sha256": contentidentity.SHA256Hex(revised)})
	}
	for path, want := range observed {
		b, e := os.ReadFile(filepath.Join(root, path))
		if e != nil || contentidentity.SHA256Hex(b) != want {
			t.Fatal("real input changed", path)
		}
	}
	if preserved != 8 {
		t.Fatal(preserved)
	}
	if len(stableSlots) != 41 {
		t.Fatal("authorized scope changed", len(stableSlots))
	}
	evidence := map[string]any{"scope": "real nine-Page imported raw responses; sparse TempDir process only", "wall_seconds": wall, "process_result": result, "units": rows, "stable_authorized_slot_ids": stableSlots, "failed_unit_restore": "PASS", "active_validation": "PASS", "passed_formal_units_preserved": preserved, "formal_repository_mutation": "none", "observed_input_sha256": observed}
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(out, "real-temp-proof.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("real 9-Page equivalent process wall=%.3fs; preserved=%d", wall, preserved)
}

// Materialize exact real authority without changing repository files. A pending
// fixture omits only the selected batch's output/install state, so the existing
// exporter can be profiled after the real batch has legitimately completed.
func copiedCommentRevisionRealFixture(t *testing.T, pending bool) (string, string, *i18n.Catalog, i18n.RetranslationBatchManifest, i18n.QualityCheckSurfaceReopen, map[string]string) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	const batch = "data/retranslation-runs/zh-CN/chatgpt-zh-CN-051"
	const reopen = "data/quality-check-reopens/zh-CN/learn-docs-qc017-current-validation-code-comments.json"
	var authority i18n.QualityCheckSurfaceReopen
	receipt, err := os.ReadFile(filepath.Join(root, reopen))
	if err != nil {
		t.Fatal(err)
	}
	if contentidentity.SHA256Hex(receipt) != "c0568b10d39e91217dbaf689b9002311282f1aaeb8a24e5bdef2ffdbbd4f74ff" {
		t.Fatal("wrong reopen")
	}
	if err = json.Unmarshal(receipt, &authority); err != nil || authority.CurrentValidation == nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	observed := map[string]string{}
	copied := map[string]bool{}
	var copyFile func(string)
	var visit func(any)
	visit = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for _, child := range x {
				visit(child)
			}
		case []any:
			for _, child := range x {
				visit(child)
			}
		case string:
			if strings.HasPrefix(x, "{") {
				var nested any
				if json.Unmarshal([]byte(x), &nested) == nil {
					visit(nested)
				}
				return
			}
			if strings.ContainsAny(x, "\n\r") || filepath.IsAbs(x) || strings.Contains(x, "..") {
				return
			}
			for _, prefix := range []string{"data/", "locales/", "_content/", "docs/", "internal/", "content/"} {
				if strings.HasPrefix(x, prefix) {
					if st, e := os.Stat(filepath.Join(root, x)); e == nil && st.Mode().IsRegular() {
						copyFile(x)
					}
					return
				}
			}
		}
	}
	copyFile = func(path string) {
		if copied[path] {
			return
		}
		copied[path] = true
		if pending && (strings.HasPrefix(path, batch+"/raw-responses/") || strings.HasPrefix(path, batch+"/generation-imports/")) {
			return
		}
		// The new isolated process starts pending, while real results stay immutable.
		if path == batch+"/result.json" || strings.HasPrefix(path, batch+"/candidates/") || strings.HasPrefix(path, batch+"/validation/") {
			return
		}
		b, e := os.ReadFile(filepath.Join(root, path))
		if e != nil {
			t.Fatal(path, e)
		}
		observed[path] = contentidentity.SHA256Hex(b)
		dst := filepath.Join(temp, path)
		if e = os.MkdirAll(filepath.Dir(dst), 0755); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(dst, b, 0600); e != nil {
			t.Fatal(e)
		}
		if strings.HasSuffix(path, ".json") {
			var value any
			if json.Unmarshal(b, &value) == nil {
				visit(value)
			}
		}
	}
	tree := func(dir string) {
		t.Helper()
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			copyFile(filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	c, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = sitecontent.AttachRegisteredTranslationPackages(root, c); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	if err = c.UseTranslationPackage("learn-docs-v1"); err != nil {
		t.Fatal(err)
	}
	if len(c.TranslationScope) != 49 {
		t.Fatal("wrong package")
	}
	copyFile("AGENTS.md")
	copyFile("go.mod")
	copyFile("go.sum")
	copyFile("locales/zh-CN/locale.json")
	copyFile(reopen)
	copyFile(batch + "/manifest.json")
	for _, dir := range []string{"docs", "internal/i18n", "internal/sitecontent", "data/unified-glossary-reviews/zh-CN", "data/unified-glossary-reviews/zh-CN-bundles", "data/translation-authorizations/zh-CN", "data/source-storage-compatibility", "data/site-source-contracts", "data/generation-archives/zh-CN"} {
		tree(dir)
	}
	copyFile("data/site-source-manifest.json")
	copyFile(i18n.GlossaryCorpusPath)
	copyFile("locales/zh-CN/glossary.yaml")
	tree(batch + "/inputs")
	tree(batch + "/raw-responses")
	tree(batch + "/generation-imports")
	// Historical readers derive manifest names from selected batch IDs; those
	// paths are not standalone Reference fields in the flattened proof.
	manifests, err := filepath.Glob(filepath.Join(root, "data/retranslation-runs/zh-CN/*/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range manifests {
		rel, _ := filepath.Rel(root, path)
		copyFile(filepath.ToSlash(rel))
		imports := filepath.Join(filepath.Dir(path), "generation-imports")
		if st, e := os.Stat(imports); e == nil && st.IsDir() {
			rel, _ := filepath.Rel(root, imports)
			tree(filepath.ToSlash(rel))
		}
		if filepath.Dir(filepath.ToSlash(rel)) != batch {
			// Retry/revalidation evidence names are derived from attempt numbers.
			// Copy the historical batch intact, including those immutable histories.
			dir, _ := filepath.Rel(root, filepath.Dir(path))
			tree(filepath.ToSlash(dir))
		}
	}
	b, err := os.ReadFile(filepath.Join(root, batch, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if contentidentity.SHA256Hex(b) != "f81fece6b28560ae99e03ff7b33b0192371dc012bcf9ff6adaccdcd442b0e2d8" {
		t.Fatal("wrong manifest")
	}
	var m i18n.RetranslationBatchManifest
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m.UnitCount != 9 {
		t.Fatal("wrong scope")
	}
	for _, r := range m.Units {
		copyFile(r.SourcePath)
	}
	return root, temp, c, m, authority, observed
}
