package main

import (
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestGlossaryStaleExplicitPackageCommand(t *testing.T) {
	const id = "8eafdd7a23781d41ad01de6a5b46d597b781d0ef9c085dcba81918b0f2550991"
	pkg, args, err := parseTranslationContentPackageArgs([]string{
		"retranslation", "export", "--locale", "zh-CN", "--content-package", "learn-docs-v1",
		"--generator", "chatgpt", "--unit-kind", "page", "--allow-reexport", "--glossary-stale", "--id", id,
	})
	if err != nil || pkg != "learn-docs-v1" {
		t.Fatalf("package=%q err=%v", pkg, err)
	}
	got, jsonOutput, err := parseRetranslationExportOptions(args[2:])
	if err != nil {
		t.Fatal(err)
	}
	want := i18n.RetranslationExportOptions{Locale: "zh-CN", Generator: i18n.RetranslationGeneratorChatGPT, UnitKind: i18n.UnitKindPage, Limit: 60, AllowReexport: true, GlossaryStale: true, UnitIDs: []string{id}}
	if jsonOutput || !reflect.DeepEqual(got, want) {
		t.Fatalf("package-aware glossary-stale options=%+v", got)
	}
}
