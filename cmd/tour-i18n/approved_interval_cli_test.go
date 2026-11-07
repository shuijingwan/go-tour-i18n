package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestApprovedIntervalCLIContract(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	f := approvedIntervalRunFixture(t)
	t.Chdir(f.f.root)
	const selector = "engineering/cli-selection.json"
	args := []string{"generation-bundle", "export", "--content-package", "learn-docs-v1", "--locale", f.f.locale, "--batch-id", f.batch.BatchID, "--approved-interval-selection", selector, "--output", f.output}
	before := authorizationDiagnosisInventory(t, f.f.root)
	for _, tc := range []struct {
		name string
		argv []string
		want string
	}{
		{"duplicate-selection", append(append([]string{}, args...), "--approved-interval-selection", selector), "selection flag must appear once"},
		{"empty-selection", append(append([]string{}, args[:8]...), append([]string{"--approved-interval-selection="}, args[10:]...)...), "requires a path and explicit --content-package"},
		{"without-package", append(append([]string{}, args[:2]...), args[4:]...), "explicit --content-package"},
		{"wrong-package", replaceExportCLIFlag(t, args, "--content-package", "tour-v1"), "approved interval selection identity/scope mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := authorizationDiagnosisOutput(t, func() error { return run(tc.argv) }); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal("real run accepted invalid argv or failed at wrong gate", err)
			}
			if _, err := os.Lstat(f.output); !os.IsNotExist(err) {
				t.Fatal("rejected argv published output")
			}
		})
	}
	out, err := authorizationDiagnosisOutput(t, func() error { return run(args) })
	if err != nil || !strings.Contains(string(out), "Generation bundle exported:") {
		t.Fatal("real run approved export", err, string(out))
	}
	files, err := i18n.ReadTransportBundle(courseContextRead(t, filepath.Dir(f.output), filepath.Base(f.output)), 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(files["formal/approved-interval-selection.json"], courseContextRead(t, f.f.root, selector)) || len(files["formal/approved-interval-transport/"+f.f.ids[0]+".json"]) == 0 {
		t.Fatal("run did not dispatch the approved selector to canonical export")
	}
	t.Log("REAL_EXPORT_DISPATCH=PASS; TOP_LEVEL_CONTENT_PACKAGE_PARSE=PASS; canonical learn-docs-v1 selection and frozen approved proof")
	checkArgs := []string{"generation-bundle", "current-check", "--content-package", f.f.pkg, "--bundle", f.output}
	bad := append(append([]string{}, checkArgs...), "--approved-interval-selection", selector)
	if _, err = authorizationDiagnosisOutput(t, func() error { return run(bad) }); err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatal("real current-check accepted external override", err)
	}
	t.Log("CURRENT_CHECK_EXTERNAL_OVERRIDE_REJECTED=PASS")
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
		t.Fatal("run export/negative cases changed synthetic authority")
	}
	if err = os.Remove(filepath.Join(f.f.root, selector)); err != nil {
		t.Fatal(err)
	}
	before = authorizationDiagnosisInventory(t, f.f.root)
	out, err = authorizationDiagnosisOutput(t, func() error { return run(checkArgs) })
	if err != nil || !strings.Contains(string(out), "Generation bundle: CURRENT") {
		t.Fatal("real frozen reconstruction requires removed external selector", err, string(out))
	}
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, f.f.root)) {
		t.Fatal("run current-check changed synthetic authority")
	}
	t.Log("CURRENT_CHECK_FROZEN_SELECTION_REBUILD=PASS; external selection absent")
}
