package sitecontent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Engineering assembly only. Never records/applies reconciliation or exports a
// revision. Frozen documents are copied verbatim, with explicit package binding.
func TestSemanticRoleQCReconciliationPreflight(t *testing.T) {
	dir := os.Getenv("GO_TOUR_SYS08_QC_PREFLIGHT")
	if dir == "" {
		t.Skip("explicit engineering output directory required")
	}
	root := repositoryRoot(t)
	c, e := i18n.BuildCatalog(root)
	if e != nil {
		t.Fatal(e)
	}
	if e = AttachRegisteredTranslationPackages(root, c); e != nil {
		t.Fatal(e)
	}
	if e = c.UseTranslationPackage("learn-docs-v1"); e != nil {
		t.Fatal(e)
	}
	f, e := os.Open("testdata/protected-page-before.json.gz")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	z, e := gzip.NewReader(f)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	var pages []struct{ Document *Document }
	if e = json.NewDecoder(z).Decode(&pages); e != nil {
		t.Fatal(e)
	}
	old := map[string]Document{}
	for _, p := range pages {
		if p.Document != nil {
			old[p.Document.Path] = *p.Document
		}
	}
	var frozen HistoricalPageRepresentations
	for _, descriptor := range c.TranslationPackages {
		if descriptor.ID != "learn-docs-v1" {
			continue
		}
		frozen.Package, frozen.PackageSHA256 = descriptor.ID, descriptor.AuthoritySHA256
		for _, u := range descriptor.Units {
			d, ok := old[u.Unit.SourcePath]
			if !ok {
				t.Fatal(u.Unit.SourcePath)
			}
			frozen.Documents = append(frozen.Documents, d)
		}
	}
	if len(frozen.Documents) != 49 {
		t.Fatal("frozen package scope", len(frozen.Documents))
	}
	write := func(name string, value any) i18n.GlossaryArchiveReference {
		b, e := json.MarshalIndent(value, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		b = append(b, '\n')
		p := filepath.Join(dir, name)
		if e = os.MkdirAll(filepath.Join(root, dir), 0755); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, p), b, 0644); e != nil {
			t.Fatal(e)
		}
		return i18n.GlossaryArchiveReference{Path: filepath.ToSlash(p), SHA256: digest(b)}
	}
	historical := write("qc014-historical-representations.json", frozen)
	resolution, e := resolvePageReplacements(root, c, historical)
	if e != nil {
		t.Fatal(e)
	}
	const oldID = "3d04565df345ae012a9e34fddf1eabc577320d02a2eb124dcf3004bfdfa9132c"
	const base = "data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-014/"
	m, e := readRegular(root, base+"manifest.json")
	if e != nil {
		t.Fatal(e)
	}
	r, e := readRegular(root, base+"quality-check-results.json")
	if e != nil {
		t.Fatal(e)
	}
	var snapshot i18n.QualityCheckSnapshotManifest
	var results struct {
		Results []struct {
			UnitID  string `json:"unit_id"`
			Rating  string `json:"rating"`
			Finding string `json:"finding"`
		} `json:"results"`
	}
	if e = json.Unmarshal(m, &snapshot); e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(r, &results); e != nil {
		t.Fatal(e)
	}
	finding := ""
	for _, result := range results.Results {
		if result.UnitID == oldID && result.Rating == "B" {
			finding = result.Finding
		}
	}
	if finding == "" {
		t.Fatal("exact original B finding missing")
	}
	p, ok := resolution.Units[oldID]
	if !ok || p.OldUnitID == p.CurrentUnitID {
		t.Fatal("successor representation missing")
	}
	var selected *i18n.QualityCheckSnapshotUnit
	for i := range snapshot.Units {
		if snapshot.Units[i].UnitID == oldID {
			selected = &snapshot.Units[i]
		}
	}
	if selected == nil {
		t.Fatal("exact Snapshot Unit missing")
	}
	vb, e := readRegular(root, selected.ValidationPath)
	if e != nil || digest(vb) != selected.ValidationSHA256 {
		t.Fatal("old validation drift", e)
	}
	var v struct {
		RawResponsePath string `json:"raw_response_path"`
	}
	if e = json.Unmarshal(vb, &v); e != nil {
		t.Fatal(e)
	}
	raw, e := readRegular(root, filepath.ToSlash(filepath.Join("data/retranslation-runs/zh-CN", selected.SelectedBatchID, v.RawResponsePath)))
	if e != nil {
		t.Fatal(e)
	}
	adapter := func(c *i18n.Catalog, id string) i18n.TranslationUnitAdapter {
		for _, u := range c.Adapted {
			if u.Unit.ID == id {
				return u.Adapter
			}
		}
		t.Fatal("missing exact adapter", id)
		return nil
	}
	before := adapter(resolution.HistoricalCatalog, oldID)
	after := adapter(c, p.CurrentUnitID)
	baseline, e := after.(i18n.ReplacementRevisionAdapter).RebaseRevisionPredecessor(before, raw)
	if e != nil {
		t.Fatal("fips140 exact successor baseline", e)
	}
	oldBytes, e := before.Restore(string(raw))
	if e != nil {
		t.Fatal(e)
	}
	newBytes, e := after.Restore(string(baseline))
	if e != nil || !bytes.Equal(oldBytes, newBytes) {
		t.Fatal("reviewed bytes changed", e)
	}
	proof := write("qc014-representation-necessity.json", i18n.RepresentationNecessityProof{Reason: "protected-natural-language-unrepresentable", Units: []i18n.RepresentationNecessityUnit{{Replacement: p, Rating: "B", FindingSHA256: digest([]byte(finding))}}})
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	args := []string{"go", "run", "-mod=readonly", "./cmd/tour-i18n", "quality-check", "surface-reopen", "--content-package", "learn-docs-v1", "--authority-source", "quality-check", "--locale", "zh-CN", "--reopen-id", "learn-docs-qc014-representation-successor", "--previous-snapshot-id", "learn-docs-v1-qc-014", "--previous-rating", "B", "--finding", finding, "--unit-id", p.CurrentUnitID, "--historical-documents", historical.Path, "--representation-proof", proof.Path}
	for i := range args {
		args[i] = q(args[i])
	}
	write("qc014-readonly-preflight.json", map[string]any{"report_type": "engineering-read-only-no-formal-apply", "result": "PASS", "authority_source": "quality-check", "snapshot_sha256": digest(m), "results_sha256": digest(r), "snapshot_unit": selected, "historical_documents": historical, "engineering_proof": proof, "replacement": p, "byte_exact_predecessor_rebase": true, "next_canonical_command": strings.Join(args, " "), "formal_authorization_gate": "NOT_EXECUTED: canonical CLI must recompute complete QC authority, currentness and unconsumed-output gates before any write.", "formal_mutation": "none"})
	t.Log("QC_B_RECONCILIATION_PREFLIGHT=PASS no formal apply/export")
}
