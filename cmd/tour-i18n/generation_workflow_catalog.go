package main

import (
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

func readonlyGenerationCatalog(args []string) bool {
	if len(args) < 2 {
		return false
	}
	return (args[0] == "generation-bundle" && (args[1] == "current-check" || args[1] == "import" || args[1] == "legacy-provenance")) ||
		(args[0] == "retranslation" && args[1] == "process")
}

func attachCommandTranslationPackages(root string, c *i18n.Catalog, args []string) error {
	if readonlyGenerationCatalog(args) {
		return sitecontent.AttachRegisteredTranslationPackagesReadonly(root, c)
	}
	return sitecontent.AttachRegisteredTranslationPackages(root, c)
}
