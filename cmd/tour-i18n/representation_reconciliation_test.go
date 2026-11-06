package main

import "testing"

func TestRepresentationReconciliationCanonicalCommand(t *testing.T) {
	pkg, args, err := parseTranslationContentPackageArgs([]string{"quality-check", "representation-reconcile", "--content-package", "learn-docs-v1", "--locale", "zh-CN", "--reconciliation-id", "learn-docs-qc014-machine-successors", "--previous-snapshot-id", "learn-docs-v1-qc-014", "--previous-snapshot-sha256", "234a37330a348590ab3b46e30d720ad42718dc798cb3002e212b86ce47acb619", "--formal-impact", "docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json", "--formal-impact-sha256", "8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e", "--historical-documents", "docs/protected-representation-evidence/qc014-historical-representations.json", "--historical-documents-sha256", "e119897a3822d9abc42360c46889ac08c850c870f00ad51f250e3b71e91162e4"})
	if err != nil || pkg != "learn-docs-v1" {
		t.Fatal(pkg, err)
	}
	o, err := parseRepresentationReconciliationOptions(args[2:])
	if err != nil {
		t.Fatal(err)
	}
	if o.Apply || o.Locale != "zh-CN" || o.ReconciliationID != "learn-docs-qc014-machine-successors" || o.PreviousSnapshotID != "learn-docs-v1-qc-014" || o.Impact.SHA256 != "8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e" {
		t.Fatal("wrong canonical preflight contract", o)
	}
	withAuthority, err := parseRepresentationReconciliationOptions(append(args[2:], "--language-authority-review-id", "independent-semantic-review"))
	if err != nil || len(withAuthority.LanguageReviewIDs) != 1 || withAuthority.LanguageReviewIDs[0] != "independent-semantic-review" || withAuthority.Apply {
		t.Fatal("canonical semantic authority preflight flags", withAuthority, err)
	}
	if _, err := parseRepresentationReconciliationOptions(append(args[2:], "--id", "arbitrary-unit")); err == nil {
		t.Fatal("ad hoc exact-set override accepted")
	}
	if _, err := parseRepresentationReconciliationOptions([]string{"--locale", "zh-CN"}); err == nil {
		t.Fatal("unbound authority accepted")
	}
}
