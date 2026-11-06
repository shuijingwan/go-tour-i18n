package main

import (
	"strings"
	"testing"
)

func TestRepresentationMachineRevisionCanonicalCommand(t *testing.T) {
	args := strings.Fields(`quality-check representation-revision
--content-package learn-docs-v1 --locale zh-CN
--reconciliation-id learn-docs-qc014-machine-successors
--previous-snapshot-id learn-docs-v1-qc-014
--previous-snapshot-sha256 234a37330a348590ab3b46e30d720ad42718dc798cb3002e212b86ce47acb619
--formal-impact docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json
--formal-impact-sha256 8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e
--historical-documents docs/protected-representation-evidence/qc014-historical-representations.json
--historical-documents-sha256 e119897a3822d9abc42360c46889ac08c850c870f00ad51f250e3b71e91162e4
--id a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f
--reopen-id learn-docs-qc014-vuln-database-representation-revision`)
	pkg, remaining, err := parseTranslationContentPackageArgs(args)
	if err != nil || pkg != "learn-docs-v1" {
		t.Fatal(pkg, err)
	}
	o, err := parseRepresentationRevisionOptions(remaining[2:])
	if err != nil {
		t.Fatal(err)
	}
	if o.Record || o.Reconciliation.Apply || o.ReopenID != "learn-docs-qc014-vuln-database-representation-revision" || o.UnitID != "a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f" {
		t.Fatal(o)
	}
	for _, flag := range []string{"--apply", "--surface-review-id", "--language-authority-review-id", "--previous-rating"} {
		if _, err := parseRepresentationRevisionOptions(append(remaining[2:], flag, "invalid")); err == nil {
			t.Fatal("mixed/incorrect machine authority flags accepted", flag)
		}
	}
	if _, err := parseRepresentationRevisionOptions([]string{"--locale", "zh-CN"}); err == nil {
		t.Fatal("unbound authority")
	}
	record, err := parseRepresentationRevisionOptions(append(remaining[2:], "--record"))
	if err != nil || !record.Record {
		t.Fatal("explicit record contract", record, err)
	}
}
