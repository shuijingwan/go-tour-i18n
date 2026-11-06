package main

import (
	"os"
	"testing"
)

func TestParseQualityCheckReplacementOptions(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, path := range []string{"historical.json", "proof.json"} {
		if err := os.WriteFile(path, []byte("{}\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	args := []string{"--authority-source", "quality-check", "--locale", "zh-CN", "--reopen-id", "representation-successor", "--previous-snapshot-id", "qc-001", "--previous-rating", "B", "--finding", "Exact original finding", "--unit-id", "current-id", "--historical-documents", "historical.json", "--representation-proof", "proof.json"}
	o, err := parseQualityCheckSurfaceReopenOptions(args)
	if err != nil || o.AuthoritySource != "quality-check" || o.Finding != "Exact original finding" || o.HistoricalDocuments.SHA256 == "" || o.RepresentationProof.SHA256 == "" {
		t.Fatal(o, err)
	}
	for _, extra := range [][]string{{"--previous-rating", "A"}, {"--surface-review-id", "surface-001"}, {"--integrated-surface"}, {"--previous-reopen-id", "preview-old"}, {"--authority-source", "unknown"}, {"--representation-proof", "../proof.json"}, {"--preview-failure", "proof.json"}} {
		if _, err := parseQualityCheckSurfaceReopenOptions(append(append([]string{}, args...), extra...)); err == nil {
			t.Fatal("mixed/unbound authority accepted", extra)
		}
	}
}
