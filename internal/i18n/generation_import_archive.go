package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func importedGenerationArchive(locale, sha string) string {
	return "data/generation-archives/" + locale + "/" + sha + ".zip"
}

// Preserve the actual imported invocation, never rebuild it using today's
// mutable authority. Recovery archives an existing ZIP against its receipt.
func ArchiveImportedGenerationBundle(root string, bundle []byte) (string, error) {
	m, files, err := readGenerationBundle(bundle)
	if err != nil {
		return "", err
	}
	if err := ValidateLocaleName(m.Locale); err != nil {
		return "", err
	}
	if err := validateBatchID(m.BatchID); err != nil {
		return "", err
	}
	dir := filepath.Join(root, "data/retranslation-runs", m.Locale, m.BatchID)
	b, err := readSnapshotReferencedFile(root, filepath.ToSlash(filepath.Join("data/retranslation-runs", m.Locale, m.BatchID, "generation-imports", filepath.Base(generationImportReceiptPath(dir, m)))))
	if err != nil {
		return "", err
	}
	var receipt GenerationResultBundleManifest
	if err := contentidentity.StrictJSON(b, &receipt); err != nil {
		return "", err
	}
	if err := checkImportedGenerationIdentity(receipt, m, bundle, files); err != nil {
		return "", err
	}
	p := importedGenerationArchive(m.Locale, sum(bundle))
	return p, writeCompatibilityImmutable(root, p, bundle)
}

func checkImportedGenerationIdentity(r GenerationResultBundleManifest, m GenerationBundleManifest, bundle []byte, files map[string][]byte) error {
	if r.SchemaVersion != GenerationResultBundleSchemaVersion || r.Kind != "go-tour-i18n/generation-result-bundle" || r.Locale != m.Locale || r.BatchID != m.BatchID || r.TaskKind != m.TaskKind || r.Attempt != m.Attempt || r.GenerationBundleSHA256 != sum(bundle) || r.InputIdentitySHA256 != m.InputIdentitySHA256 || r.GenerationManifest.BundlePath != "generation-manifest.json" || r.GenerationManifest.SHA256 != sum(files["manifest.json"]) || r.GenerationManifest.Size != int64(len(files["manifest.json"])) || len(r.Outputs) != len(m.ExpectedOutputs) {
		return fmt.Errorf("imported Generation provenance/input identity mismatch")
	}
	return requireFormalGenerationIdentity(r.Provider, r.Model, r.BatchID)
}

func verifyImportedRevisionInvocation(root string, c *Catalog, locale string, choice selectedRetranslationUnit, v *validatedRetranslationCandidate) ([]GlossaryArchiveReference, error) {
	refs := []GlossaryArchiveReference{}
	m, err := readRetranslationProcessManifest(choice.batchDir, locale, choice.batchID)
	if err != nil {
		return nil, err
	}
	// Existing authorization algorithm checks exact source, predecessor
	// candidate, independent B/C/D or Surface finding and protected baseline.
	inputs, err := revisionGenerationInputs(root, c, locale, m.Units)
	if err != nil {
		return nil, err
	}
	for _, attempt := range []int{1, v.attempt} {
		if attempt == 1 && v.attempt == 1 && len(refs) > 0 {
			continue
		}
		gm := GenerationBundleManifest{TaskKind: "translation-unit-revision", Attempt: 1}
		if attempt > 1 {
			gm.TaskKind = "translation-unit-retry"
			gm.Attempt = attempt
			gm.ExpectedOutputs = []GenerationBundleExpectedOutput{{UnitID: choice.manifest.UnitID}}
		}
		p := filepath.ToSlash(filepath.Join("data/retranslation-runs", locale, choice.batchID, "generation-imports", filepath.Base(generationImportReceiptPath(choice.batchDir, gm))))
		b, err := readCompatibilityFile(root, p)
		if err != nil {
			return nil, err
		}
		receiptSHA := sum(b)
		var receipt GenerationResultBundleManifest
		if err := contentidentity.StrictJSON(b, &receipt); err != nil {
			return nil, err
		}
		if !validSHA256(receipt.GenerationBundleSHA256) || !validSHA256(receipt.InputIdentitySHA256) {
			return nil, fmt.Errorf("invalid revision provenance identity")
		}
		archive := importedGenerationArchive(locale, receipt.GenerationBundleSHA256)
		bundle, err := readCompatibilityFile(root, archive)
		if err != nil {
			return nil, fmt.Errorf("original imported Generation Bundle archive required: %s: %w", archive, err)
		}
		original, files, err := readGenerationBundle(bundle)
		if err != nil {
			return nil, err
		}
		if err := checkImportedGenerationIdentity(receipt, original, bundle, files); err != nil {
			return nil, err
		}
		if original.TaskKind != gm.TaskKind || original.Attempt != attempt || original.Locale != locale || original.BatchID != choice.batchID {
			return nil, fmt.Errorf("revision invocation/attempt mismatch")
		}
		glossary, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
		if err != nil || !bytes.Equal(glossary, files["formal/glossary.yaml"]) {
			return nil, fmt.Errorf("revision invocation current glossary mismatch")
		}
		for _, input := range inputs {
			if !bytes.Equal(input.data, files[input.path]) {
				return nil, fmt.Errorf("revision predecessor/finding input changed: %s", input.path)
			}
		}
		for _, f := range original.Files {
			if f.RepositoryPath == "" || strings.HasPrefix(f.BundlePath, "authority/") || strings.HasPrefix(f.BundlePath, "formal/retry/") {
				continue
			}
			b, err := readCompatibilityFile(root, f.RepositoryPath)
			if err != nil || !bytes.Equal(b, files[f.BundlePath]) {
				return nil, fmt.Errorf("revision Generation input changed: %s", f.BundlePath)
			}
		}
		if original.TaskKind == "translation-unit-retry" {
			if _, err := verifyRetryBatchResult(choice.batchDir, c, m, files["formal/retry/result.json"], true); err != nil {
				return nil, err
			}
		}
		outputs := map[string]TransportBundleFile{}
		for _, out := range receipt.Outputs {
			if _, ok := outputs[out.BundlePath]; ok {
				return nil, fmt.Errorf("duplicate revision output provenance")
			}
			outputs[out.BundlePath] = out
		}
		for _, out := range original.ExpectedOutputs {
			f, ok := outputs[out.BundlePath]
			p := filepath.ToSlash(filepath.Join("data/retranslation-runs", locale, choice.batchID, out.InstallPath))
			b, err := readCompatibilityFile(root, p)
			if !ok || err != nil || sum(b) != f.SHA256 || int64(len(b)) != f.Size {
				return nil, fmt.Errorf("revision output provenance mismatch")
			}
			if out.UnitID == choice.manifest.UnitID && attempt == v.attempt && out.InstallPath != v.validation.RawResponsePath {
				return nil, fmt.Errorf("revision output does not bind selected final attempt")
			}
			refs = append(refs, GlossaryArchiveReference{p, sum(b)})
		}
		refs = append(refs, GlossaryArchiveReference{p, receiptSHA}, GlossaryArchiveReference{archive, sum(bundle)})
		if v.attempt == 1 {
			break
		}
	}
	return refs, nil
}
