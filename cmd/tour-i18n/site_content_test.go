package main

import (
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
	"path/filepath"
	"strings"
	"testing"
)

func TestSiteContentCLIRejectsUnsafeInputs(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"generation-export", "--plan-batch", "0"}, {"generation-export", "--unit", "x", "--plan-batch", "1"}, {"generation-plan", "--plan-batch", "1"}, {"review-plan", "--unit", "x"}, {"review-export", "--plan-batch", "-1"}, {"unknown"}, {"inventory", "--unknown"}, {"generation-check", "--bundle", "../outside.zip"}, {"review-record", "--input", "../outside.json"}, {"activation-apply", "--input", "../outside.json"}} {
		if err := siteContentCommand(root, args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if err := siteContentCommand(root, []string{"inventory", "--package", "unknown-package"}); err == nil || !strings.Contains(err.Error(), "unknown package") {
		t.Fatalf("unknown package: %v", err)
	}
}

func TestSiteContentCLIFixedPagePlan(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	b := captureStdout(t, func() error {
		return siteContentCommand(root, []string{"page-plan", "--package", "learn-docs-v1"})
	})
	var p sitecontent.BatchPlan
	if err := sitecontent.StrictJSON(b, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Sets) != 2 || len(p.Sets[0].Selected) != 30 || len(p.Sets[1].Selected) != 19 || p.Contract != sitecontent.PageContract {
		t.Fatal("CLI exposed internal slot packing instead of fixed Page plan")
	}
}
