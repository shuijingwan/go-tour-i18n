package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func TestQualityCheckSnapshotExplicitPackageCLI(t *testing.T) {
	args := []string{"quality-check", "snapshot", "--locale", "zh-CN", "--snapshot-id", "new-id", "--content-package", "site-v2-shell", "--preflight"}
	pkg, remaining, err := parseTranslationContentPackageArgs(args)
	if err != nil || pkg != "" || !reflect.DeepEqual(args, remaining) {
		t.Fatal("dispatcher consumed Snapshot's explicit flag", pkg, remaining, err)
	}
	o, preflight, err := parseQualityCheckSnapshotOptions(remaining[2:])
	if err != nil || !preflight || o.ContentPackage != "site-v2-shell" || o.SnapshotID != "new-id" {
		t.Fatal(o, preflight, err)
	}
	_, prefixed, err := parseTranslationContentPackageArgs([]string{"--content-package", "site-v2-shell", "quality-check", "snapshot", "--locale", "zh-CN", "--snapshot-id", "new-id"})
	if err != nil {
		t.Fatal(err)
	}
	prefixOptions, _, err := parseQualityCheckSnapshotOptions(prefixed[2:])
	if err != nil || prefixOptions.ContentPackage != "site-v2-shell" {
		t.Fatal("legacy prefix became hidden scope", prefixOptions, err)
	}
	o, preflight, err = parseQualityCheckSnapshotOptions([]string{"--locale", "zh-CN", "--snapshot-id", "site-v2-shell-qc-003"})
	if err != nil || preflight || o.ContentPackage != "" {
		t.Fatal("Snapshot ID inferred package", o, err)
	}
	base := []string{"--locale", "zh-CN", "--snapshot-id", "new-id"}
	for _, extra := range [][]string{
		{"--content-package", ""}, {"--content-package", "first", "--content-package", "second"},
		{"--unit-id", "arbitrary"}, {"--id", "arbitrary"}, {"--limit", "1"}, {"--start-index", "1"}, {"--unit-kind", "page"}, {"--apply"}, {"--record"}, {"unexpected"},
	} {
		if _, _, err := parseQualityCheckSnapshotOptions(append(append([]string{}, base...), extra...)); err == nil {
			t.Fatal("unsafe/partial/ambiguous CLI accepted", extra)
		}
	}
	c := &i18n.Catalog{TranslationScope: []string{"hidden"}}
	if err := qualityCheckSnapshotCommand(t.TempDir(), c, base); err == nil || !strings.Contains(err.Error(), "no hidden scope") {
		t.Fatal("omitted package inherited caller state", err)
	}
	doc, err := os.ReadFile("../../docs/RETRANSLATION_RUNBOOK.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"Candidate Snapshot command contract", "COMMAND", "OPERATION", "INPUT_KIND", "INPUT_PATH_CONTRACT", "MUTATION_EXPECTATION", "IDEMPOTENCE", "CURRENTNESS_CHECK", "PRIOR_OR_CAS", "SUCCESS_STOP", "FAILURE_STOP", "site-v2-shell-qc-003"} {
		if !strings.Contains(string(doc), field) {
			t.Error("canonical Snapshot contract missing", field)
		}
	}
}

// Opt-in readonly proof of the real current Homepage successor. No import,
// process, Snapshot publication or QC action; output goes only to /tmp.
func TestPackageSnapshotCurrentRepositoryPreflight(t *testing.T) {
	out := os.Getenv("GO_TOUR_PACKAGE_SNAPSHOT_PREFLIGHT")
	if out == "" {
		t.Skip("explicit /tmp readonly proof destination required")
	}
	if !strings.HasPrefix(filepath.Clean(out), "/tmp/") {
		t.Fatal("evidence must be outside the repository")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	done := contentidentity.BeginMemoization()
	defer done()
	current, err := i18n.BuildSourceCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	c, err := i18n.ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := i18n.HydrateCatalogSources(c, current); err != nil {
		t.Fatal(err)
	}
	if err := sitecontent.AttachRegisteredTranslationPackagesReadonly(root, c); err != nil {
		t.Fatal(err)
	}
	sitecontent.AttachIntegratedSurfaceAuthority(c)
	site.AttachPreviewFailureAuthority(c)
	f, err := os.Create(filepath.Join(out, "real-preflight.json"))
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = f
	err = qualityCheckSnapshotCommand(root, c, []string{"--locale", "zh-CN", "--snapshot-id", "site-v2-shell-qc-004", "--content-package", "site-v2-shell", "--preflight"})
	os.Stdout = stdout
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Manifest i18n.QualityCheckSnapshotManifest `json:"manifest"`
		Recorded bool                              `json:"recorded"`
		Mutation string                            `json:"mutation"`
	}
	b, err := os.ReadFile(filepath.Join(out, "real-preflight.json"))
	if err != nil || json.Unmarshal(b, &result) != nil {
		t.Fatal("invalid preflight result", err)
	}
	m := result.Manifest
	if result.Recorded || result.Mutation != "none" || m.ContentPackage == nil || m.ContentPackage.ID != "site-v2-shell" || m.UnitCount != 1 || m.PageCount != 1 || m.ExampleCount != 0 || m.Units[0].UnitID != "f4365d05fcff295676c0964f81058e6485bb0361f8440cbf243220a6acc1e97d" {
		t.Fatal("wrong current package selection", m)
	}
	if _, err := os.Stat(filepath.Join(root, "data/quality-check-snapshots/zh-CN/site-v2-shell-qc-004")); !os.IsNotExist(err) {
		t.Fatal("preflight created real 004")
	}
	for _, id := range []string{"site-v2-shell-qc-001", "site-v2-shell-qc-002", "site-v2-shell-qc-003"} {
		path := filepath.Join(root, "data/quality-check-snapshots/zh-CN", id, "manifest.json")
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var legacy i18n.QualityCheckSnapshotManifest
		if err := contentidentity.StrictJSON(b, &legacy); err != nil || legacy.SnapshotID != id || legacy.ContentPackage != nil {
			t.Fatal("legacy Snapshot schema became unreadable", id, err)
		}
	}
	t.Logf("readonly current shell selection PASS: Units=%d Unit=%s mutation=none", m.UnitCount, m.Units[0].UnitID)
}
