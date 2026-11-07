package i18n

import (
	"testing"
)

func TestOrdinaryImportedGenerationKeepsProcessScope(t *testing.T) {
	root, catalog, batch := generationBundleFixture(t, "zh-CN", RetranslationGeneratorChatGPT)
	bundle, gm, err := ExportTranslationUnitGenerationBundle(root, catalog, GenerationBundleOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil {
		t.Fatal(err)
	}
	outputs := t.TempDir()
	writeGenerationFixtureOutputs(t, root, batch, outputs, gm)
	if _, err := ImportGenerationOutputDirectory(root, catalog, bundle, "chatgpt", FormalChatGPTGenerationModel, outputs); err != nil {
		t.Fatal(err)
	}
	m := readRetranslationManifest(t, root, batch)
	scoped, err := reconstructImportedApprovedIntervalPackage(root, catalog, &m)
	if err != nil || scoped != catalog || len(scoped.TranslationScope) != 0 {
		t.Fatal("ordinary imported batch changed its caller's scope", err)
	}
	result, err := ProcessRetranslationBatch(root, catalog, RetranslationProcessOptions{Locale: "zh-CN", BatchID: batch})
	if err != nil || result.RestorePassed != len(gm.ExpectedOutputs) || result.ValidationPassed != len(gm.ExpectedOutputs) {
		t.Fatal("ordinary imported process changed behavior", result, err)
	}
}
