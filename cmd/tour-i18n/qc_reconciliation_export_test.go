package main

import (
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Parse the complete published command through the same global and subcommand
// parsers as run. Export execution is tested only in isolated sitecontent fixtures.
func TestQCRepresentationSuccessorCanonicalCommand(t *testing.T) {
	const id = "8e2b3ec2b00595cb7e8ea8587ea6002107a0e616c18f49fc46562702af10f1c3"
	pkg, args, err := parseTranslationContentPackageArgs([]string{
		"retranslation", "export", "--locale", "zh-CN", "--content-package", "learn-docs-v1",
		"--generator", "chatgpt", "--batch-id", "chatgpt-zh-CN-047", "--unit-kind", "page",
		"--previous-snapshot-id", "learn-docs-v1-qc-014", "--surface-reopen-id", "learn-docs-qc014-representation-successor",
		"--allow-reexport", "--id", id,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pkg != "learn-docs-v1" || !reflect.DeepEqual(args[:2], []string{"retranslation", "export"}) {
		t.Fatal("wrong package or dispatch", pkg, args)
	}
	got, jsonOutput, err := parseRetranslationExportOptions(args[2:])
	if err != nil {
		t.Fatal(err)
	}
	want := i18n.RetranslationExportOptions{Locale: "zh-CN", BatchID: "chatgpt-zh-CN-047", Generator: i18n.RetranslationGeneratorChatGPT, UnitKind: i18n.UnitKindPage, Limit: 60, PreviousSnapshotID: "learn-docs-v1-qc-014", SurfaceReopenID: "learn-docs-qc014-representation-successor", AllowReexport: true, UnitIDs: []string{id}}
	if jsonOutput || !reflect.DeepEqual(got, want) {
		t.Fatalf("canonical export options: %+v", got)
	}
}
